# 记忆型问答系统

这是一个使用 Go 语言实现的后端和 Next.js 实现的前端的记忆型问答系统。系统能够记住用户的问题和答案，并在下次提问时提供相应的回答。

## 目录

- [功能简介](#功能简介)
- [技术栈](#技术栈)
- [前提条件](#前提条件)
- [安装与运行](#安装与运行)
  - [后端（Go）](#后端go)
  - [前端（Next.js）](#前端nextjs)
- [项目结构](#项目结构)
- [注意事项](#注意事项)
- [未来改进](#未来改进)

## 功能简介

- **提问功能**：用户可以输入问题，系统会返回已有的答案。如果没有答案，提示用户提供答案。
- **记忆功能**：系统能够记住用户提供的答案，供下次查询使用。
- **前后端分离**：后端使用 Go 提供 API，前端使用 Next.js 构建用户界面。
- **跨域支持**：后端已配置 CORS，允许前端进行跨域请求。

## 技术栈

- **后端**：
  - Go
  - Gorilla Mux（路由管理）
  - Gorilla Handlers（CORS 处理）
- **前端**：
  - Next.js
  - React

## 前提条件

确保你的机器上已安装以下软件：

- [Go](https://golang.org/dl/) （版本 1.16 或以上）
- [Node.js](https://nodejs.org/) 和 npm （版本 12 或以上）

## 安装与运行

### 后端（Go）

1. **克隆仓库并进入后端目录**（假设后端代码在 `backend` 文件夹）：

   ````bash
   git clone <仓库地址>
   cd backend    ```

   ````

2. **安装依赖**：

   ````bash
   go get -u github.com/gorilla/mux
   go get -u github.com/gorilla/handlers    ```

   ````

3. **保存 `main.go` 文件**：

   将以下内容保存为 `main.go`：

   ````go:main.go
   package main

   import (
       "encoding/json"
       "net/http"
       "sync"

       "github.com/gorilla/mux"
       "github.com/gorilla/handlers"
   )

   type QA struct {
       Question string `json:"question"`
       Answer   string `json:"answer"`
   }

   var (
       qaStore = make(map[string]string)
       mutex   = &sync.Mutex{}
   )

   func getAnswer(w http.ResponseWriter, r *http.Request) {
       var qa QA
       err := json.NewDecoder(r.Body).Decode(&qa)
       if err != nil {
           http.Error(w, err.Error(), http.StatusBadRequest)
           return
       }

       mutex.Lock()
       answer, exists := qaStore[qa.Question]
       if !exists {
           answer = "抱歉，我还没有回答过这个问题。你可以告诉我答案。"
       }
       mutex.Unlock()

       json.NewEncoder(w).Encode(QA{Question: qa.Question, Answer: answer})
   }

   func saveAnswer(w http.ResponseWriter, r *http.Request) {
       var qa QA
       err := json.NewDecoder(r.Body).Decode(&qa)
       if err != nil {
           http.Error(w, err.Error(), http.StatusBadRequest)
           return
       }

       mutex.Lock()
       qaStore[qa.Question] = qa.Answer
       mutex.Unlock()

       w.WriteHeader(http.StatusOK)
   }

   func main() {
       r := mux.NewRouter()
       r.HandleFunc("/api/get-answer", getAnswer).Methods("POST")
       r.HandleFunc("/api/save-answer", saveAnswer).Methods("POST")

       headersOk := handlers.AllowedHeaders([]string{"Content-Type"})
       originsOk := handlers.AllowedOrigins([]string{"http://localhost:3000"})
       methodsOk := handlers.AllowedMethods([]string{"POST"})

       http.ListenAndServe(":8080", handlers.CORS(originsOk, headersOk, methodsOk)(r))
   }    ```

   ````

4. **运行后端服务器**：

   ````bash
   go run main.go    ```

   后端服务器将运行在 `http://localhost:8080`。
   ````

### 前端（Next.js）

1. **创建并进入前端项目目录**（假设前端代码在 `frontend` 文件夹）：

   ````bash
   npx create-next-app@latest my-qa-app
   cd my-qa-app    ```

   ````

2. **安装依赖**（如果有其他依赖，可在此步骤安装）：

   ````bash
   npm install    ```

   ````

3. **替换 `pages/index.js` 文件内容**：

   将以下内容保存为 `pages/index.js`：

   ````javascript:pages/index.js
   import { useState } from 'react';

   export default function Home() {
       const [question, setQuestion] = useState('');
       const [answer, setAnswer] = useState('');
       const [newAnswer, setNewAnswer] = useState('');

       const handleAsk = async () => {
           const res = await fetch('http://localhost:8080/api/get-answer', {
               method: 'POST',
               headers: {
                   'Content-Type': 'application/json',
               },
               body: JSON.stringify({ question }),
           });
           const data = await res.json();
           setAnswer(data.answer);
       };

       const handleSave = async () => {
           await fetch('http://localhost:8080/api/save-answer', {
               method: 'POST',
               headers: {
                   'Content-Type': 'application/json',
               },
               body: JSON.stringify({ question, answer: newAnswer }),
           });
           setAnswer(newAnswer);
           setNewAnswer('');
       };

       return (
           <div style={{ padding: '20px' }}>
               <h1>问答系统</h1>
               <input
                   type="text"
                   value={question}
                   onChange={(e) => setQuestion(e.target.value)}
                   placeholder="请输入你的问题"
                   style={{ width: '300px', padding: '8px' }}
               />
               <button onClick={handleAsk} style={{ marginLeft: '10px', padding: '8px 16px' }}>
                   提问
               </button>
               <div style={{ marginTop: '20px' }}>
                   <h2>回答：</h2>
                   <p>{answer}</p>
                   {answer.startsWith('抱歉') && (
                       <div>
                           <input
                               type="text"
                               value={newAnswer}
                               onChange={(e) => setNewAnswer(e.target.value)}
                               placeholder="请输入答案"
                               style={{ width: '300px', padding: '8px' }}
                           />
                           <button onClick={handleSave} style={{ marginLeft: '10px', padding: '8px 16px' }}>
                               保存答案
                           </button>
                       </div>
                   )}
               </div>
           </div>
       );
   }    ```

   > **注意**：已移除 `!qaStore[question]` 判断，因为前端无法直接访问后端的 `qaStore`。改为仅根据后端返回的回答内容来决定是否显示保存答案的输入框。

   ````

4. **运行前端服务器**：

   ````bash
   npm run dev    ```

   前端服务器将运行在 `http://localhost:3000`。
   ````

## 项目结构

```
my-qa-app/
│
├── backend/
│   └── main.go
│
├── frontend/
│   ├── pages/
│   │   └── index.js
│   └── ...
│
└── README.md
```

## 注意事项

- **跨域问题**：后端已配置 CORS，允许来自 `http://localhost:3000` 的请求。如果前端部署到其他地址，需要在后端 `main.go` 中的 `originsOk` 中添加相应地址。
- **数据持久化**：当前实现使用内存存储问题和答案，服务器重启后数据会丢失。建议使用数据库（如 SQLite、PostgreSQL 等）来持久化数据。

- **安全性**：在生产环境中，应考虑添加身份验证、输入验证等安全措施，防止恶意攻击和数据泄露。

## 未来改进

- **数据库集成**：将内存存储替换为数据库，实现数据持久化。
- **用户认证**：添加用户登录功能，区分不同用户的问答数据。
- **高级功能**：支持问题分类、搜索功能、多语言支持等。
- **UI 优化**：提升前端界面的用户体验和美观度。
- **部署**：将应用部署到云平台，实现线上访问。

## 联系我们

如有任何问题或建议，请联系 [你的联系方式]。
