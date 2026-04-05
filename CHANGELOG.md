# CHANGELOG

## [v1.2.0] - 2026-04-04

### 新增
- 多云平台接入（EasyStack Keystone 认证 + ZStack AccessKey 认证），支持同时接入多套实例
- AI 模型厂商扩充至 13 家：新增硅基流动、Moonshot(Kimi)、百度文心、火山引擎(豆包)、腾讯混元、百川智能、Anthropic Claude、Google Gemini
- 三套 UI 主题切换（白色 / 黑色 / 蓝色），localStorage 持久化
- 密码强度校验（≥9位 + 大写 + 小写 + 数字 + 特殊字符）
- 前端密码强度条和实时需求 CheckList
- 功能架构图 + 组件架构图（PNG，五层分层框图）
- 一键部署脚本 `scripts/deploy.sh`

### 变更
- 品牌全局替换为「AIOPS 智能运维平台」
- 菜单重命名：AI模型→模型配置，技能→技能中心，对话→即时对话
- 主色调统一为 #513CC8（深紫蓝），全站 primary 色系
- 顶部 Header 统一主色，高度与 Sidebar 对齐
- admin 初始密码升级为 `Admin@2024!`，动态 bcrypt hash（不再硬编码）
- README 完整重写，含架构图、部署指南、API 概览

### 修复
- Admin 路由注册错误导致 AdminMiddleware 失效
- CreateUser/UpdateUser 密码明文存储导致登录 401
- seedDefaultData 改为 upsert 逻辑，修复旧数据库明文密码问题
- docker-compose backend 挂载不存在的 frontend/dist 导致部署失败
- backend 去掉静态文件服务逻辑，专注 API 服务

---

## [v1.0.0] - 初始版本

- Go + Gin 后端 REST API
- React 18 + Vite + Tailwind CSS 前端
- MySQL + RabbitMQ 基础架构
- EasyStack 云平台运维 Agent（初版）
- JWT 认证，多用户管理
