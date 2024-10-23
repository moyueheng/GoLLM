package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/prompts"
	"gopkg.in/yaml.v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Conversation 用于管理和组织整个对话流程
type Conversation struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Name      string    `json:"name"`
	Messages  []Message `json:"messages"`
}

// Message 用于记录对话中的具体交互内容
type Message struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ConversationID uint      `json:"conversation_id"`
	Sender         string    `json:"sender"` // "user" 或 "assistant"
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
}

var (
	db     *gorm.DB
	config struct {
		Ollama struct {
			BaseURL string `yaml:"base_url"`
			Model   string `yaml:"model"`
		} `yaml:"ollama"`
	}
	llm *ollama.LLM
)

func loadConfig() {
	// 优先从环境变量获取配置
	if baseURL := os.Getenv("OLLAMA_BASE_URL"); baseURL != "" {
		config.Ollama.BaseURL = baseURL
	}
	if model := os.Getenv("OLLAMA_MODEL"); model != "" {
		config.Ollama.Model = model
	}

	// 如果环境变量未设置，则从配置文件读取
	if config.Ollama.BaseURL == "" || config.Ollama.Model == "" {
		configFile, err := os.ReadFile("config.yaml")
		if err != nil {
			panic("无法读取配置文件: " + err.Error())
		}

		err = yaml.Unmarshal(configFile, &config)
		if err != nil {
			panic("无法解析配置文件: " + err.Error())
		}
	}

	// 确保必要的配置项已设置
	if config.Ollama.BaseURL == "" {
		panic("Ollama BaseURL 未设置")
	}
	if config.Ollama.Model == "" {
		panic("Ollama Model 未设置")
	}
}

// initDB 初始化数据库
func initDB() {
	var err error
	db, err = gorm.Open(sqlite.Open("chat_history.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// 自动迁移模型
	db.AutoMigrate(&Conversation{}, &Message{})
}

// initLLM 初始化 OpenAI 客户端
func initLLM(model_type string) error {
	if model_type == "ollama" {
		var err error
		llm, err = ollama.New(
			ollama.WithModel(config.Ollama.Model),
			ollama.WithServerURL(config.Ollama.BaseURL),
		)
		if err != nil {
			return fmt.Errorf("无法初始化 Ollama: %w", err)
		}
		return nil
	}
	return fmt.Errorf("不支持的模型类型: %s", model_type)
}

var system_prompt string

func init() {
	promptBytes, err := os.ReadFile("prompt.md")
	if err != nil {
		panic("无法读取 prompt 文件: " + err.Error())
	}
	system_prompt = string(promptBytes)
}

// chatMessage 处理获取答案的请求
func chatMessage(c *gin.Context) {
	var request struct {
		ConversationID uint   `json:"conversation_id"`
		Question       string `json:"question"`
	}

	// 解析请求体中的 JSON 数据
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无法解析请求数据: " + err.Error(),
		})
		return
	}

	var conversation Conversation
	if request.ConversationID != 0 {
		// 如果提供了 ConversationID，尝试查找现有对话
		if err := db.First(&conversation, request.ConversationID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "找不到指定的对话",
			})
			return
		}
	} else {
		// 如果没有提供 ConversationID，创建新对话并生成名称
		conversation = Conversation{}

		// 调用大模型生成会话名称
		ctx := context.Background()
		namePrompt := fmt.Sprintf("请为以下问题生成一个简短的会话名称（不超过20个字）：%s", request.Question)
		completion, err := llm.Call(ctx, namePrompt, llms.WithTemperature(0.5))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "无法生成会话名称: " + err.Error(),
			})
			return
		}
		conversation.Name = strings.TrimSpace(completion)

		db.Create(&conversation)
	}

	// 保存用户的问题到消息表
	userMessage := Message{
		ConversationID: conversation.ID,
		Sender:         "user",
		Content:        request.Question,
	}
	db.Create(&userMessage)

	// 调用 Ollama API 获取答案

	// 读取prompt文件

	// 将prompt内容转换为字符串
	// 读取会话历史消息
	var messages []Message
	if err := db.Where("conversation_id = ?", conversation.ID).Order("created_at asc").Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "无法获取会话历史: " + err.Error(),
		})
		return
	}

	// 构建prompt
	promptMessages := []prompts.MessageFormatter{
		prompts.NewSystemMessagePromptTemplate(
			system_prompt,
			nil,
		),
	}

	// 添加历史消息到prompt
	// 限制历史消息数量为最多10条
	messageCount := len(messages)
	startIndex := 0
	if messageCount > 10 {
		startIndex = messageCount - 10
	}
	messages = messages[startIndex:]
	for _, msg := range messages {
		if msg.Sender == "user" {
			promptMessages = append(promptMessages, prompts.NewHumanMessagePromptTemplate(
				msg.Content,
				nil,
			))
		} else if msg.Sender == "assistant" {
			promptMessages = append(promptMessages, prompts.NewAIMessagePromptTemplate(
				msg.Content,
				nil,
			))
		}
	}

	// 添加当前用户问题
	promptMessages = append(promptMessages, prompts.NewHumanMessagePromptTemplate(
		"{{.question}}",
		[]string{"question"},
	))

	prompt_template := prompts.NewChatPromptTemplate(promptMessages)
	prompt, err := prompt_template.Format(map[string]any{
		"question": request.Question,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "无法格式化prompt: " + err.Error(),
		})
		return
	}
	ctx := context.Background()

	completion, err := llm.Call(ctx, prompt,
		llms.WithTemperature(0.8),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "无法获取Ollama的回答: " + err.Error(),
		})
		return
	}

	assistantAnswer := completion

	// 保存助手的回答到消息表
	assistantMessage := Message{
		ConversationID: conversation.ID,
		Sender:         "assistant",
		Content:        assistantAnswer,
	}
	db.Create(&assistantMessage)

	// 返回问题、答案和会话信息
	c.JSON(http.StatusOK, gin.H{
		"conversation_id":   conversation.ID,
		"conversation_name": conversation.Name,
		"question":          request.Question,
		"answer":            assistantAnswer,
	})
}

// getAllConversations 处理获取所有会话的请求
func getAllConversations(c *gin.Context) {
	var conversations []Conversation
	if err := db.Order("created_at desc").Find(&conversations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "无法获取会话列表: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, conversations)
}

// deleteConversation 处理删除指定会话的请求
func deleteConversation(c *gin.Context) {
	// 从URL参数中获取会话ID
	conversationID := c.Param("id")

	// 检查会话ID是否有效
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的会话ID",
		})
		return
	}

	// 开始数据库事务
	tx := db.Begin()

	// 删除与会话相关的所有消息
	if err := tx.Where("conversation_id = ?", conversationID).Delete(&Message{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "删除会话消息失败: " + err.Error(),
		})
		return
	}

	// 删除会话
	if err := tx.Where("id = ?", conversationID).Delete(&Conversation{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "删除会话失败: " + err.Error(),
		})
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "提交事务失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "会话删除成功",
	})
}

// getConversationMessages 处理获取指定会话的所有消息的请求
func getConversationMessages(c *gin.Context) {
	// 从URL参数中获取会话ID
	conversationID := c.Param("id")

	// 检查会话ID是否有效
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的会话ID",
		})
		return
	}

	var messages []Message
	if err := db.Where("conversation_id = ?", conversationID).Order("created_at asc").Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取会话消息失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, messages)
}

func clearConversationMessages(c *gin.Context) {
	conversationID := c.Param("id")

	if err := db.Where("conversation_id = ?", conversationID).Delete(&Message{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "清空会话消息失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "会话消息已清空"})
}

func getSystemInfo(c *gin.Context) {
	info := gin.H{
		"model":   config.Ollama.Model,
		"version": "1.0.0", // 假设的版本号
		// 可以添加其他系统信息
	}
	c.JSON(http.StatusOK, info)
}

func getConversation(c *gin.Context) {
	conversationID := c.Param("id")

	var conversation Conversation
	if err := db.First(&conversation, conversationID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "找不到指定的会话"})
		return
	}

	c.JSON(http.StatusOK, conversation)
}

func updateConversationName(c *gin.Context) {
	conversationID := c.Param("id")
	var request struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	var conversation Conversation
	if err := db.First(&conversation, conversationID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "找不到指定的会话"})
		return
	}

	conversation.Name = request.Name
	if err := db.Save(&conversation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新会话名称失败"})
		return
	}

	c.JSON(http.StatusOK, conversation)
}

func main() {
	// 加载配置
	loadConfig()

	// 初始化数据库
	initDB()

	// 初始化 OpenAI 客户端
	err := initLLM("ollama")
	if err != nil {
		panic(err)
	}

	// 创建一个默认的 Gin 路由器
	r := gin.Default()

	// 设置路由
	r.POST("/api/chat_message", chatMessage)                               // 处理聊天消息
	r.GET("/api/conversations", getAllConversations)                       // 获取所有会话
	r.DELETE("/api/conversations/:id", deleteConversation)                 // 删除指定会话
	r.GET("/api/conversations/:id/messages", getConversationMessages)      // 获取指定会话的所有消息
	r.DELETE("/api/conversations/:id/messages", clearConversationMessages) // 清空指定会话的所有消息
	r.GET("/api/system_info", getSystemInfo)                               // 获取系统信息
	r.GET("/api/conversations/:id", getConversation)                       // 获取指定会话的详细信息
	r.PUT("/api/conversations/:id/name", updateConversationName)           // 更新指定会话的名称

	// 启动服务器，监听 8081 端口
	r.Run(":8081")
}
