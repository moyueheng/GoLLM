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

