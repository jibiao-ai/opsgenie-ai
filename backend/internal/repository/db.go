package repository

import (
	"fmt"
	"os"
	"time"

	"github.com/jibiao-ai/cloud-agent/internal/config"
	"github.com/jibiao-ai/cloud-agent/internal/model"
	"github.com/jibiao-ai/cloud-agent/pkg/logger"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB(cfg config.DatabaseConfig) error {
	var db *gorm.DB
	var err error

	dbDriver := os.Getenv("DB_DRIVER")
	if dbDriver == "" {
		dbDriver = "mysql"
	}

	switch dbDriver {
	case "sqlite":
		dbPath := os.Getenv("DB_PATH")
		if dbPath == "" {
			dbPath = "cloud_agent.db"
		}
		db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
		if err != nil {
			return fmt.Errorf("failed to open sqlite: %w", err)
		}
		logger.Log.Infof("Using SQLite database: %s", dbPath)

	default: // mysql
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)

		for i := 0; i < 30; i++ {
			db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
			if err == nil {
				break
			}
			logger.Log.Warnf("Failed to connect to database (attempt %d/30): %v", i+1, err)
			time.Sleep(2 * time.Second)
		}
		if err != nil {
			return fmt.Errorf("failed to connect to database after retries: %w", err)
		}

		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetConnMaxLifetime(time.Hour)
		logger.Log.Info("Using MySQL database")
	}

	// Auto migrate
	err = db.AutoMigrate(
		&model.User{},
		&model.Agent{},
		&model.Skill{},
		&model.Conversation{},
		&model.Message{},
		&model.TaskLog{},
		&model.Workflow{},
		&model.ScheduledTask{},
		&model.EasyStackEndpoint{},
		&model.AIProvider{},
	)
	if err != nil {
		return fmt.Errorf("auto migration failed: %w", err)
	}

	DB = db
	logger.Log.Info("Database connection established and migrated")

	// Seed default data
	seedDefaultData(db)

	return nil
}

func seedDefaultData(db *gorm.DB) {
	// Create default admin user
	var count int64
	db.Model(&model.User{}).Count(&count)
	if count == 0 {
		admin := model.User{
			Username: "admin",
			Password: "$2a$10$5HCtytk2H8rwfdEB9ysMcepF3tLhnpiPE5XoktVUMwMOgyF2quBlO", // admin123
			Email:    "admin@cloudagent.local",
			Role:     "admin",
		}
		db.Create(&admin)
		logger.Log.Info("Default admin user created")
	}

	// Create default EasyStack ops agent
	db.Model(&model.Agent{}).Count(&count)
	if count == 0 {
		agents := []model.Agent{
			{
				Name:        "EasyStack 运维助手",
				Description: "基于 EasyStack ECF 6.2.1 API 的智能运维Agent，支持云主机、云硬盘、网络、监控告警等运维操作",
				SystemPrompt: `你是一个专业的 EasyStack 云平台运维智能助手。你可以帮助用户完成以下运维任务：

1. **云主机管理**: 查询、创建、启动、关闭、重启、暂停、恢复云主机，调整规格、创建快照、挂载/卸载云硬盘
2. **云硬盘管理**: 查询、创建、删除、扩容云硬盘，管理快照
3. **网络管理**: 查询、创建网络和子网，管理路由器、浮动IP、安全组和安全组规则
4. **负载均衡**: 查询、创建、管理负载均衡器、监听器、后端池和成员
5. **监控告警**: 查询系统指标、性能数据，查看告警信息
6. **配额管理**: 查询和修改域配额

你应该：
- 用清晰的中文回答用户问题
- 当需要执行操作时，调用对应的 EasyStack API
- 在执行危险操作前（如删除、重启），先确认用户意图
- 提供操作结果的清晰摘要
- 如果遇到错误，解释可能的原因和解决方案`,
				Model:       "gpt-4",
				Temperature: 0.3,
				MaxTokens:   4096,
				Skills:      `["easystack_compute","easystack_storage","easystack_network","easystack_monitor","easystack_lb"]`,
				IsActive:    true,
				CreatedBy:   1,
			},
			{
				Name:        "故障诊断专家",
				Description: "专门进行云平台故障诊断和问题排查的智能Agent",
				SystemPrompt: `你是一个专业的 EasyStack 云平台故障诊断专家。你的职责是：

1. 分析用户描述的问题症状
2. 通过查询监控指标和资源状态来诊断问题根因
3. 提供详细的诊断报告和修复建议
4. 如有需要，执行修复操作

诊断流程：
1. 收集信息：询问问题详情，查询相关资源状态
2. 分析问题：根据指标数据和状态信息判断问题类型
3. 提供方案：给出明确的修复步骤
4. 执行修复：在用户确认后执行修复操作`,
				Model:       "gpt-4",
				Temperature: 0.2,
				MaxTokens:   4096,
				Skills:      `["easystack_compute","easystack_storage","easystack_network","easystack_monitor"]`,
				IsActive:    true,
				CreatedBy:   1,
			},
			{
				Name:        "资源优化顾问",
				Description: "分析资源使用情况并提供优化建议的智能Agent",
				SystemPrompt: `你是一个 EasyStack 云平台资源优化顾问。你的职责是：

1. 分析云资源使用情况（CPU、内存、磁盘、网络）
2. 识别资源浪费和瓶颈
3. 提供资源优化建议
4. 帮助用户调整资源配置

你应该关注：
- 闲置或低利用率的云主机
- 未挂载的云硬盘
- 过大或过小的规格配置
- 网络带宽使用情况
- 配额使用和分配优化`,
				Model:       "gpt-4",
				Temperature: 0.3,
				MaxTokens:   4096,
				Skills:      `["easystack_compute","easystack_storage","easystack_network","easystack_monitor"]`,
				IsActive:    true,
				CreatedBy:   1,
			},
		}
		for _, a := range agents {
			db.Create(&a)
		}
		logger.Log.Info("Default agents created")
	}

	// Seed default AI providers
	db.Model(&model.AIProvider{}).Count(&count)
	if count == 0 {
		providers := []model.AIProvider{
			{
				Name:        "openai",
				Label:       "OpenAI",
				BaseURL:     "https://api.openai.com/v1",
				Model:       "gpt-4o",
				IsDefault:   true,
				IsEnabled:   true,
				Description: "OpenAI GPT 系列模型，支持 GPT-4o、GPT-4、GPT-3.5 等",
			},
			{
				Name:        "deepseek",
				Label:       "DeepSeek",
				BaseURL:     "https://api.deepseek.com/v1",
				Model:       "deepseek-chat",
				IsDefault:   false,
				IsEnabled:   true,
				Description: "深度求索 DeepSeek 系列模型，高性价比国产大模型",
			},
			{
				Name:        "qwen",
				Label:       "通义千问",
				BaseURL:     "https://dashscope.aliyuncs.com/compatible-mode/v1",
				Model:       "qwen-plus",
				IsDefault:   false,
				IsEnabled:   true,
				Description: "阿里云通义千问系列模型，支持 Qwen-Plus、Qwen-Max 等",
			},
			{
				Name:        "glm",
				Label:       "智谱 GLM",
				BaseURL:     "https://open.bigmodel.cn/api/paas/v4",
				Model:       "glm-4",
				IsDefault:   false,
				IsEnabled:   true,
				Description: "智谱 AI GLM 系列模型，支持 GLM-4、GLM-4-Flash 等",
			},
			{
				Name:        "minimax",
				Label:       "MiniMax",
				BaseURL:     "https://api.minimax.chat/v1",
				Model:       "abab6.5s-chat",
				IsDefault:   false,
				IsEnabled:   true,
				Description: "MiniMax 大模型，支持 abab6.5s-chat 等系列",
			},
		}
		for _, p := range providers {
			db.Create(&p)
		}
		logger.Log.Info("Default AI providers created")
	}

	// Create default skills
	db.Model(&model.Skill{}).Count(&count)
	if count == 0 {
		skills := []model.Skill{
			{
				Name:        "云主机管理",
				Description: "EasyStack 云主机(Nova)相关操作：查询、创建、启停、重启、调整规格、快照等",
				Type:        "easystack_api",
				Config:      `{"service":"compute","api_version":"v2.1","capabilities":["list_servers","create_server","server_actions","attach_volume","detach_volume","list_flavors","list_keypairs"]}`,
				IsActive:    true,
			},
			{
				Name:        "云硬盘管理",
				Description: "EasyStack 云硬盘(Cinder)相关操作：查询、创建、删除、扩容、快照管理",
				Type:        "easystack_api",
				Config:      `{"service":"storage","api_version":"v2","capabilities":["list_volumes","create_volume","delete_volume","extend_volume","list_snapshots","create_snapshot"]}`,
				IsActive:    true,
			},
			{
				Name:        "网络管理",
				Description: "EasyStack SDN网络(Neutron)相关操作：网络、子网、路由器、浮动IP、安全组等",
				Type:        "easystack_api",
				Config:      `{"service":"network","api_version":"v2.0","capabilities":["list_networks","create_network","list_subnets","create_subnet","list_routers","list_floatingips","list_security_groups","create_security_group_rule"]}`,
				IsActive:    true,
			},
			{
				Name:        "监控告警",
				Description: "EasyStack 监控(ECMS)相关操作：指标查询、告警查看、性能分析",
				Type:        "easystack_api",
				Config:      `{"service":"monitor","api_version":"v1","capabilities":["query_metrics","list_alerts","resource_top5"]}`,
				IsActive:    true,
			},
			{
				Name:        "负载均衡管理",
				Description: "EasyStack 负载均衡(Octavia)相关操作：查询、创建、管理LB、监听器、后端池",
				Type:        "easystack_api",
				Config:      `{"service":"loadbalancer","api_version":"v2.0","capabilities":["list_loadbalancers","create_loadbalancer","list_listeners","list_pools","list_members"]}`,
				IsActive:    true,
			},
		}
		for _, s := range skills {
			db.Create(&s)
		}
		logger.Log.Info("Default skills created")
	}
}
