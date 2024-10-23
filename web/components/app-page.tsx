'use client'

import { useState, useEffect } from 'react'
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { ScrollArea } from "@/components/ui/scroll-area"
import { PlusCircle, Trash2, Edit2 } from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'

interface Message {
  id: number
  conversation_id: number
  sender: string
  content: string
  created_at: string
}

interface Conversation {
  id: number
  name: string
  created_at: string
  messages: Message[] | null
}

export function AppPage() {
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [currentConversation, setCurrentConversation] = useState<number | null>(null)
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [newConversationName, setNewConversationName] = useState('')
  const [editingConversation, setEditingConversation] = useState<number | null>(null)
  const [editName, setEditName] = useState('')

  useEffect(() => {
    fetchConversations()
  }, [])

  const fetchConversations = async () => {
    const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/conversations`)
    const data = await response.json()
    setConversations(data)
  }

  const fetchMessages = async (conversationId: number) => {
    const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/conversations/${conversationId}/messages`)
    const data = await response.json()
    setMessages(data)
  }

  const handleSendMessage = async () => {
    if (!input.trim()) return

    const newUserMessage: Message = {
      id: Date.now(),
      conversation_id: currentConversation || 0, // 如果没有当前会话，暂时使用0
      sender: 'user',
      content: input,
      created_at: new Date().toISOString()
    }

    // 立即添加用户消息到消息列表
    setMessages(prevMessages => [...prevMessages, newUserMessage])
    setInput('')

    try {
      const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/chat_message`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ conversation_id: currentConversation, question: input }),
      })
      const data = await response.json()

      const newAssistantMessage: Message = {
        id: Date.now() + 1,
        conversation_id: currentConversation || data.conversation_id,
        sender: 'assistant',
        content: data.answer,
        created_at: new Date().toISOString()
      }

      // 添加 AI 的回复到消息列表
      setMessages(prevMessages => [...prevMessages, newAssistantMessage])

      if (!currentConversation) {
        setCurrentConversation(data.conversation_id)
        await fetchConversations()
      }
    } catch (error) {
      console.error('发送消息失败:', error)
      // 这里可以添加错误处理，比如显示一个错误提示给用户
    }
  }

  // 修改handleNewConversation函数
  const handleNewConversation = async () => {
    if (!newConversationName.trim()) return

    try {
      const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/chat_message`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ question: `${newConversationName}` }),
      })
      const data = await response.json()

      const newConversation: Conversation = {
        id: data.conversation_id,
        name: newConversationName,
        created_at: new Date().toISOString(),
        messages: null
      }

      // 更新会话列表
      setConversations(prevConversations => [...prevConversations, newConversation])
      
      // 立即切换到新创建的会话
      setCurrentConversation(data.conversation_id)
      
      // 清空消息列表，因为这是一个新会话
      setMessages([])
      
      // 清空新会话名称输入
      setNewConversationName('')
      
      // 获取新创建会话的消息（如果有的话）
      await fetchMessages(data.conversation_id)
      
      // 重新获取所有会话列表
      await fetchConversations()
    } catch (error) {
      console.error('创建新会话失败:', error)
      // 这里可以添加错误处理，比如显示一个错误提示给用户
    }
  }

  // 修改 handleClearConversation 函数
  const handleClearConversation = async () => {
    if (!currentConversation) return

    const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/conversations/${currentConversation}/messages`, {
      method: 'DELETE',
    })

    if (response.ok) {
      setMessages([])
      setCurrentConversation(null)
      await fetchConversations()
    } else {
      // 处理错误
      console.error('清空会话失败')
    }
  }

  // 新增处理函数
  const handleDeleteConversation = async (conversationId: number) => {
    const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/conversations/${conversationId}`, {
      method: 'DELETE',
    })

    if (response.ok) {
      // 如果删除的是当前会话，清空消息和当前会话状态
      if (currentConversation === conversationId) {
        setMessages([])
        setCurrentConversation(null)
      }
      // 重新获取会话列表
      await fetchConversations()
    } else {
      // 处理错误
      console.error('删除会话失败')
    }
  }

  // 新增修改会话名称的函数
  const handleEditConversation = async (conversationId: number) => {
    if (!editName.trim()) return

    const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/conversations/${conversationId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: editName }),
    })

    if (response.ok) {
      await fetchConversations()
      setEditingConversation(null)
      setEditName('')
    } else {
      console.error('修改会话名称失败')
    }
  }

  return (
    <div className="flex h-screen bg-gray-100">
      {/* 左侧会话列表 */}
      <motion.div 
        initial={{ x: -300 }}
        animate={{ x: 0 }}
        transition={{ type: "spring", stiffness: 100 }}
        className="w-64 bg-white p-4 border-r"
      >
        <h2 className="text-xl font-bold mb-4">视⻅睿来AI助手</h2>
        {/* 新建会话输入框和按钮 */}
        <div className="mb-4">
          <Input
            value={newConversationName}
            onChange={(e) => setNewConversationName(e.target.value)}
            placeholder="询问用于新建会话"
            className="mb-2"
          />
          <Button onClick={handleNewConversation} className="w-full">
            <PlusCircle className="mr-2 h-4 w-4" />
            新增会话
          </Button>
        </div>
        <ScrollArea className="h-[calc(100vh-12rem)]">
          {conversations.map((conv) => (
            <div key={conv.id} className="flex items-center mb-2">
              {editingConversation === conv.id ? (
                <div className="flex-1 flex items-center">
                  <Input
                    value={editName}
                    onChange={(e) => setEditName(e.target.value)}
                    className="mr-2"
                  />
                  <Button
                    size="sm"
                    onClick={() => handleEditConversation(conv.id)}
                  >
                    保存
                  </Button>
                </div>
              ) : (
                <>
                  <Button
                    variant={currentConversation === conv.id ? "secondary" : "ghost"}
                    className="flex-1 justify-start"
                    onClick={() => {
                      setCurrentConversation(conv.id)
                      fetchMessages(conv.id)
                    }}
                  >
                    {conv.name}
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 p-0"
                    onClick={() => {
                      setEditingConversation(conv.id)
                      setEditName(conv.name)
                    }}
                  >
                    <Edit2 className="h-4 w-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 p-0"
                    onClick={() => handleDeleteConversation(conv.id)}
                  >
                    <Trash2 className="h-4 w-4 text-red-500" />
                  </Button>
                </>
              )}
            </div>
          ))}
        </ScrollArea>
      </motion.div>

      {/* 右侧消息区域 */}
      <div className="flex-1 flex flex-col">
        <ScrollArea className="flex-1 p-4">
          <AnimatePresence>
            {messages.map((msg, index) => (
              <motion.div
                key={index}
                initial={{ opacity: 0, y: 50 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -50 }}
                transition={{ duration: 0.3 }}
                className={`mb-4 ${msg.sender === 'user' ? 'text-right' : 'text-left'}`}
              >
                <div className={`inline-block p-2 rounded-lg ${msg.sender === 'user' ? 'bg-blue-500 text-white' : 'bg-gray-200'}`}>
                  {msg.content}
                </div>
              </motion.div>
            ))}
          </AnimatePresence>
        </ScrollArea>
        
        {/* 输入区域 */}
        <motion.div 
          initial={{ y: 100 }}
          animate={{ y: 0 }}
          transition={{ type: "spring", stiffness: 100 }}
          className="p-4 border-t flex"
        >
          <Input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyPress={(e) => e.key === 'Enter' && handleSendMessage()}
            placeholder="Type your message..."
            className="flex-1 mr-2"
          />
          <Button onClick={handleSendMessage}>Send</Button>
          <Button onClick={handleClearConversation} className="ml-2">
            清空会话
          </Button>
        </motion.div>
      </div>
    </div>
  )
}
