# AIOPS 智能运维平台

> 基于 AI 大模型的智能云平台运维系统，支持多云接入、自然语言运维操作、自动化工作流。

## 功能特性

- 🤖 **多模型 AI 对话** — 支持 13 家主流 AI 厂商，可随时切换，一键测试连通性
- ☁️ **多云平台接入** — 支持 EasyStack（Keystone 认证）和 ZStack（AccessKey 认证）
- 🛡️ **多用户权限管理** — admin / user 双角色，bcrypt 密码加密，密码强度校验
- ⚡ **定时任务与工作流** — 支持 Cron 表达式调度、工作流编排
- 🎨 **三套主题切换** — 白色 / 黑色 / 蓝色主题，实时切换

## 支持的 AI 模型

| 厂商 | 标识 | 推荐模型 | 说明 |
|------|------|----------|------|
| OpenAI | 🤖 | gpt-4o | GPT-4o / GPT-4 / GPT-3.5 系列 |
| DeepSeek | 🔍 | deepseek-chat | 深度求索，高性价比国产大模型 |
| 通义千问 | ☁️ | qwen-plus | 阿里云 Qwen-Plus / Qwen-Max 系列 |
| 智谱 GLM | 🧠 | glm-4 | 智谱 AI GLM-4 / GLM-4-Flash 系列 |
| MiniMax | ⚡ | abab6.5s-chat | MiniMax abab 系列 |
| 硅基流动 | 💎 | Qwen/Qwen2.5-7B-Instruct | 支持 Qwen / DeepSeek / GLM 开源模型推理，性价比极高 |
| Moonshot (Kimi) | 🌙 | moonshot-v1-8k | 超长上下文，8k / 32k / 128k |
| 百度文心一言 | 🔵 | ernie-4.5-8k | ERNIE 4.5 / 4.0 / Speed 系列 |
| 火山引擎（豆包） | 🔥 | doubao-pro-4k | 字节豆包 doubao-pro / lite 系列 |
| 腾讯混元 | 🌀 | hunyuan-pro | 混元 pro / standard 系列 |
| 百川智能 | 🐋 | Baichuan4 | Baichuan4 / Baichuan3-Turbo 系列 |
| Anthropic Claude | 🎭 | claude-3-5-sonnet-20241022 | claude-3-5-sonnet / haiku / opus |
| Google Gemini | ✨ | gemini-2.0-flash | gemini-2.0-flash / 1.5-pro 系列 |

## 支持的云平台

| 类型 | 认证方式 | 说明 |
|------|----------|------|
| EasyStack | Keystone Token | 填写 AuthURL / 用户名 / 密码 / 域名 / 项目名称 |
| ZStack | AccessKey | 填写 Endpoint / AccessKeyID / AccessKeySecret |

## 快速部署

### 使用 Docker Compose（推荐）

```bash
# 克隆项目
git clone <repo-url>
cd cloud-agent

# 启动所有服务（MySQL + RabbitMQ + Backend + Frontend）
docker compose up -d

# 查看服务状态
docker compose ps

# 查看后端日志
docker compose logs -f backend
```

访问地址：http://localhost （或服务器 IP）

### 本地开发模式

```bash
# 启动基础服务
docker compose up -d mysql rabbitmq

# 后端（需要 Go 1.21+）
cd backend
go mod download
go run cmd/server/main.go

# 前端（需要 Node.js 18+）
cd frontend
npm install
npm run dev
```

## 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `DB_DRIVER` | `mysql` | 数据库驱动，可选 `mysql` / `sqlite` |
| `DB_HOST` | `mysql` | MySQL 主机名 |
| `DB_PORT` | `3306` | MySQL 端口 |
| `DB_NAME` | `cloud_agent` | 数据库名 |
| `DB_USER` | `root` | 数据库用户名 |
| `DB_PASSWORD` | `password` | 数据库密码 |
| `DB_PATH` | `cloud_agent.db` | SQLite 文件路径（仅 sqlite 模式） |
| `JWT_SECRET` | `change-me-in-production` | JWT 签名密钥，**生产环境必须修改** |
| `SERVER_PORT` | `8080` | 后端监听端口 |
| `RABBITMQ_URL` | `amqp://guest:guest@rabbitmq:5672/` | RabbitMQ 连接地址（可选） |
| `EASYSTACK_AUTH_URL` | — | EasyStack Keystone 地址 |
| `EASYSTACK_USERNAME` | — | EasyStack 用户名 |
| `EASYSTACK_PASSWORD` | — | EasyStack 密码 |
| `EASYSTACK_DOMAIN` | `Default` | EasyStack 域名 |
| `EASYSTACK_PROJECT` | — | EasyStack 项目名 |

## 默认账号

| 字段 | 值 |
|------|----|
| 用户名 | `admin` |
| 密码 | `Admin@2024!` |

> ⚠️ **首次登录后请立即修改密码！**

## 密码安全要求

系统强制密码策略（创建/修改用户时生效）：

- ✅ 长度至少 **9 位**
- ✅ 包含至少一个**大写字母**（A-Z）
- ✅ 包含至少一个**小写字母**（a-z）
- ✅ 包含至少一个**数字**（0-9）
- ✅ 包含至少一个**特殊字符**（`!@#$%^&*` 等）

示例合法密码：`Admin@2024!`、`MyP@ssw0rd!`

## 整体架构

```
╔══════════════════════════════════════════════════════════════════════╗
║                         【 展示层 / 接入层 】                          ║
║                                                                      ║
║   ┌─────────────────────────────────────────────────────────────┐   ║
║   │                    Web 前端（浏览器）                          │   ║
║   │   React 18 + Vite + Tailwind CSS + Lucide Icons             │   ║
║   │   仪表盘 │ 即时对话 │ 模型配置 │ 接入云平台 │ 用户管理          │   ║
║   │                   三套主题（白/黑/蓝）                         │   ║
║   └──────────────────────────┬──────────────────────────────────┘   ║
║                              │ HTTP / WebSocket                      ║
║   ┌──────────────────────────▼──────────────────────────────────┐   ║
║   │                   Nginx 反向代理                              │   ║
║   │         /* → 静态前端文件       /api/* → 后端 :8080           │   ║
║   └──────────────────────────────────────────────────────────────┘   ║
╚══════════════════════════════════════════════════════════════════════╝
                              │
╔══════════════════════════════════════════════════════════════════════╗
║                         【 应用服务层 】                               ║
║                                                                      ║
║   ┌──────────────┐  ┌──────────────┐  ┌───────────────────────┐    ║
║   │  认证鉴权     │  │  REST API    │  │  WebSocket 实时通信    │    ║
║   │  JWT 中间件  │  │  Gin Router  │  │  流式 AI 对话输出      │    ║
║   │  Admin 权限  │  │  CRUD 接口   │  │  心跳保活              │    ║
║   └──────────────┘  └──────────────┘  └───────────────────────┘    ║
║                                                                      ║
║   ┌──────────────────────────────────────────────────────────────┐  ║
║   │                     AI Agent 推理引擎                          │  ║
║   │   自然语言理解 → Function Calling → 运维意图识别 → 工具执行    │  ║
║   │   支持多轮对话 │ 流式输出 │ Tool Call 链式调用                 │  ║
║   └──────────────────────────────────────────────────────────────┘  ║
║                                                                      ║
║   ┌──────────────────────────────────────────────────────────────┐  ║
║   │                    异步任务调度层                               │  ║
║   │          RabbitMQ 消息队列 │ 定时任务(Cron) │ 工作流引擎        │  ║
║   └──────────────────────────────────────────────────────────────┘  ║
╚══════════════════════════════════════════════════════════════════════╝
                              │
╔══════════════════════════════════════════════════════════════════════╗
║                         【 能力集成层 】                               ║
║                                                                      ║
║   ┌─────────────────────────┐   ┌──────────────────────────────┐   ║
║   │      AI 模型接入          │   │       云平台接入              │   ║
║   │  OpenAI / DeepSeek       │   │  ┌────────────────────────┐  │   ║
║   │  Qwen / GLM / MiniMax    │   │  │  EasyStack             │  │   ║
║   │  Kimi / 硅基流动          │   │  │  Keystone Token 认证   │  │   ║
║   │  文心 / 豆包 / 混元        │   │  │  云主机/网络/存储/监控   │  │   ║
║   │  百川 / Claude / Gemini  │   │  ├────────────────────────┤  │   ║
║   │  统一 OpenAI 兼容协议      │   │  │  ZStack                │  │   ║
║   │  支持 API Key 配置和测试   │   │  │  AccessKey 认证        │  │   ║
║   └─────────────────────────┘   │  │  虚机/存储/网络/告警     │  │   ║
║                                  │  └────────────────────────┘  │   ║
║                                  │  支持同时接入多套平台实例       │   ║
║                                  └──────────────────────────────┘   ║
╚══════════════════════════════════════════════════════════════════════╝
                              │
╔══════════════════════════════════════════════════════════════════════╗
║                         【 数据存储层 】                               ║
║                                                                      ║
║   ┌───────────────────────────┐   ┌──────────────────────────────┐  ║
║   │       MySQL 8.0           │   │       RabbitMQ 3             │  ║
║   │  GORM ORM / 自动迁移       │   │   异步任务队列 / 消息持久化    │  ║
║   │  用户 / Agent / 会话 / 消息 │   └──────────────────────────────┘  ║
║   │  技能 / 工作流 / 定时任务   │                                      ║
║   │  云平台 / AI提供商 / 日志   │   开发模式可替换为 SQLite            ║
║   └───────────────────────────┘                                      ║
╚══════════════════════════════════════════════════════════════════════╝
                              │
╔══════════════════════════════════════════════════════════════════════╗
║                         【 基础设施层 】                               ║
║                                                                      ║
║        Docker + Docker Compose   │   Linux 虚拟机 / 云服务器          ║
║        Nginx Alpine 容器          │   公网 IP + 防火墙策略              ║
╚══════════════════════════════════════════════════════════════════════╝
```

### 分层说明

| 层级 | 职责 |
|------|------|
| **展示层** | React SPA，用户交互界面，Nginx 承载静态文件并反向代理 API |
| **应用服务层** | Go+Gin 核心业务逻辑，JWT 鉴权、AI 推理引擎、异步任务调度 |
| **能力集成层** | 对接 13 家 AI 模型厂商（统一 OpenAI 协议）+ EasyStack/ZStack 多云平台 |
| **数据存储层** | MySQL 持久化所有业务数据，RabbitMQ 异步解耦耗时任务 |
| **基础设施层** | Docker 容器化部署，一键 `docker compose up` 拉起全栈环境 |

## API 接口概览

所有 API 以 `/api` 为前缀，受保护接口需携带 `Authorization: Bearer <token>` 请求头。

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/login` | 用户登录，返回 JWT Token |
| GET | `/api/profile` | 获取当前用户信息 |
| GET | `/api/dashboard` | 获取仪表盘统计数据 |
| GET/POST/PUT/DELETE | `/api/agents` | 智能体 CRUD |
| GET/POST/DELETE | `/api/conversations` | 会话 CRUD |
| GET/POST | `/api/conversations/:id/messages` | 消息列表和发送 |
| GET | `/api/ws` | WebSocket 实时对话 |
| GET | `/api/skills` | 技能中心列表 |
| GET/POST | `/api/workflows` | 工作流 CRUD |
| GET/POST | `/api/scheduled-tasks` | 定时任务 CRUD |
| GET/PUT/POST | `/api/ai-providers` | AI 模型提供商配置和测试 |
| GET/POST/PUT/DELETE/POST | `/api/cloud-platforms` | 云平台接入 CRUD 和连接测试 |
| GET/POST/PUT/DELETE | `/api/users` | 用户管理（Admin 权限） |

## License

MIT
