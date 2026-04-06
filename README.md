<div align="center">

<img src="docs/logo-full.png" alt="OpsGenie AI" width="480" />

<br/>

# OpsGenie AI - 智能运维平台

![version](https://img.shields.io/badge/version-v3.1.0-513CC8?style=flat-square)
![license](https://img.shields.io/badge/license-MIT-blue?style=flat-square)
![go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)
![react](https://img.shields.io/badge/React-18-61DAFB?style=flat-square&logo=react)

**AI 大模型驱动的智能云平台运维系统 | 多云接入 · 自然语言运维 · Function Calling · 自动化工作流**

</div>

---

## 功能特性

- 🤖 **多模型 AI 对话** — 支持 13 家主流 AI 厂商，可随时切换，一键测试连通性
- ☁️ **多云平台接入** — 支持 EasyStack（Keystone 认证）和 ZStack（AccessKey 认证）
- ⚡ **技能中心 & Function Calling** — 智能体绑定技能（30+ 云 API），AI 自动调用工具完成运维操作
- 🔗 **三方联动** — 智能体 × 技能中心 × 云平台 闭环对接，对话即运维
- 🛡️ **多用户权限管理** — admin / user 双角色，bcrypt 密码加密，密码强度校验
- 📋 **操作日志审计** — 全面记录关键操作，支持多维度筛选查询
- ⏱️ **定时任务与工作流** — 支持 Cron 表达式调度、工作流编排
- 🎨 **三套主题切换** — 白色 / 黑色 / 蓝色主题，实时切换

## 支持的 AI 模型

| 厂商 | 标识 | 推荐模型 | 说明 |
|------|------|----------|------|
| OpenAI | 🤖 | gpt-4o | GPT-4o / GPT-4 / GPT-3.5 系列 |
| DeepSeek | 🔍 | deepseek-chat | 深度求索，高性价比国产大模型 |
| 通义千问 | ☁️ | qwen-plus | 阿里云 Qwen-Plus / Qwen-Max 系列 |
| 智谱 GLM | 🧠 | glm-4 | 智谱 AI GLM-4 / GLM-4-Flash 系列 |
| MiniMax | ⚡ | abab6.5s-chat | MiniMax abab 系列 |
| 硅基流动 | 💎 | Qwen/Qwen2.5-7B-Instruct | 支持 Qwen / DeepSeek / GLM 开源模型推理 |
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

## 技能中心 (Function Calling)

智能体可绑定以下预置技能，在对话中由 AI 自动调用：

| 技能 | 类型 | 包含工具 |
|------|------|----------|
| 计算资源管理 | cloud_api | 列出/创建/启动/停止/删除云主机 |
| 存储资源管理 | cloud_api | 列出/创建/删除云硬盘 |
| 网络资源管理 | cloud_api | 列出网络/子网/安全组/浮动IP |
| 监控服务 | cloud_api | 查询 CPU/内存/磁盘/网络指标 |
| 负载均衡 | cloud_api | 列出负载均衡器 |

## 快速部署

### 使用 Docker Compose（推荐）

```bash
# 克隆项目
git clone https://github.com/jibiao-ai/cloud-agent.git
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

### 功能架构图

![功能架构图](docs/aiops_feature_diagram.png)

### 组件架构图

![组件架构图](docs/aiops_component_diagram.png)

### 数据流（Function Calling）

```
用户对话 → AI 模型(Function Calling) → SkillExecutor → 云平台 Token 认证 → API 调用 → 结果 → AI 总结回复
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
| GET | `/api/agents/:id/skills` | 获取智能体关联技能 |
| GET/POST/DELETE | `/api/conversations` | 会话 CRUD |
| GET/POST | `/api/conversations/:id/messages` | 消息列表和发送 |
| GET | `/api/ws` | WebSocket 实时对话 |
| GET | `/api/skills` | 技能中心列表 |
| GET/POST | `/api/workflows` | 工作流 CRUD |
| GET/POST | `/api/scheduled-tasks` | 定时任务 CRUD |
| GET/PUT/POST | `/api/ai-providers` | AI 模型提供商配置和测试 |
| GET/POST/PUT/DELETE/POST | `/api/cloud-platforms` | 云平台接入 CRUD 和连接测试 |
| GET/POST/PUT/DELETE | `/api/users` | 用户管理（Admin 权限） |
| GET | `/api/operation-logs` | 操作日志查询 |

## License

MIT

---

<div align="center">

**OpsGenie AI** — 让运维更智能、更高效

<img src="docs/logo-icon.png" alt="OpsGenie AI" width="48" />

</div>
