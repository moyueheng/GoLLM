package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"gopkg.in/yaml.v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Conversation 用于管理和组织整个对话流程
type Conversation struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
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
	client *openai.Client
	config struct {
		Ollama struct {
			BaseURL string `yaml:"base_url"`
			Model   string `yaml:"model"`
		} `yaml:"ollama"`
	}
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

// initOpenAI 初始化 OpenAI 客户端
func initOpenAI() {
	config := openai.DefaultConfig("")
	config.BaseURL = "http://localhost:11434/v1" // Ollama 的 OpenAI 兼容接口地址
	client = openai.NewClientWithConfig(config)
}

// getAnswer 处理获取答案的请求
func getAnswer(c *gin.Context) {
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
		// 如果没有提供 ConversationID，创建新对话
		conversation = Conversation{}
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
	response, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: "llama2", // 使用 Ollama 支持的模型名称
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: request.Question,
				},
			},
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "无法获取Ollama的回答: " + err.Error(),
		})
		return
	}

	assistantAnswer := response.Choices[0].Message.Content

	// 保存助手的回答到消息表
	assistantMessage := Message{
		ConversationID: conversation.ID,
		Sender:         "assistant",
		Content:        assistantAnswer,
	}
	db.Create(&assistantMessage)

	// 返回问题和答案
	c.JSON(http.StatusOK, gin.H{
		"question": request.Question,
		"answer":   assistantAnswer,
	})
}

func main() {
	// 加载配置
	loadConfig()

	// 初始化数据库
	initDB()

	// 初始化 OpenAI 客户端
	initOpenAI()

	// 创建一个默认的 Gin 路由器
	r := gin.Default()

	// 设置路由
	r.POST("/api/get-answer", getAnswer)

	// 启动服务器，监听 8081 端口
	r.Run(":8081")
}
