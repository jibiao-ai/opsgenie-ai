package repository

import (
	"fmt"
	"os"
	"time"

	"github.com/jibiao-ai/opsgenie-ai/internal/config"
	"github.com/jibiao-ai/opsgenie-ai/internal/model"
	"github.com/jibiao-ai/opsgenie-ai/pkg/logger"
	"golang.org/x/crypto/bcrypt"
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
			dbPath = "opsgenie_ai.db"
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
		&model.AgentSkill{},
		&model.Conversation{},
		&model.Message{},
		&model.TaskLog{},
		&model.Workflow{},
		&model.ScheduledTask{},
		&model.EasyStackEndpoint{},
		&model.AIProvider{},
		&model.CloudPlatform{},
		&model.OperationLog{},
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
	// Ensure default admin user exists with correct bcrypt password hash.
	// Default password: Admin@2024!  (meets strength policy: upper/lower/digit/special, >=9 chars)
	const adminPlainPassword = "Admin@2024!"

	// Dynamically generate bcrypt hash so it is always valid
	adminHashBytes, _ := bcrypt.GenerateFromPassword([]byte(adminPlainPassword), 10)
	adminPasswordHash := string(adminHashBytes)

	var admin model.User
	result := db.Where("username = ?", "admin").First(&admin)
	if result.Error != nil {
		// Admin not found — create it
		admin = model.User{
			Username: "admin",
			Password: adminPasswordHash,
			Email:    "admin@cloudagent.local",
			Role:     "admin",
		}
		db.Create(&admin)
		logger.Log.Info("Default admin user created")
	} else {
		// Ensure role is admin
		if admin.Role != "admin" {
			db.Model(&admin).Update("role", "admin")
			logger.Log.Info("Default admin user role fixed to admin")
		}
	}

	// Create default EasyStack ops agent
	var count int64
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

	// Seed default AI providers — insert if name not exists
	defaultProviders := []model.AIProvider{
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
		{
			Name:        "siliconflow",
			Label:       "硅基流动 SiliconFlow",
			BaseURL:     "https://api.siliconflow.cn/v1",
			Model:       "Qwen/Qwen2.5-7B-Instruct",
			IsDefault:   false,
			IsEnabled:   true,
			Description: "硅基流动，支持 Qwen、DeepSeek、GLM 等开源模型",
		},
		{
			Name:        "moonshot",
			Label:       "Moonshot AI (Kimi)",
			BaseURL:     "https://api.moonshot.cn/v1",
			Model:       "moonshot-v1-8k",
			IsDefault:   false,
			IsEnabled:   true,
			Description: "Moonshot AI Kimi，支持超长上下文，moonshot-v1-8k/32k/128k",
		},
		{
			Name:        "baidu",
			Label:       "百度文心一言",
			BaseURL:     "https://qianfan.baidubce.com/v2",
			Model:       "ernie-4.5-8k",
			IsDefault:   false,
			IsEnabled:   true,
			Description: "百度文心一言，支持 ERNIE 4.5、4.0、Speed 等系列",
		},
		{
			Name:        "zhipu",
			Label:       "智谱 ChatGLM",
			BaseURL:     "https://open.bigmodel.cn/api/paas/v4",
			Model:       "glm-4-flash",
			IsDefault:   false,
			IsEnabled:   true,
			Description: "智谱 AI GLM-4，支持 glm-4、glm-4-flash、glm-4-plus",
		},
		{
			Name:        "volcengine",
			Label:       "火山引擎（豆包）",
			BaseURL:     "https://ark.cn-beijing.volces.com/api/v3",
			Model:       "doubao-pro-4k",
			IsDefault:   false,
			IsEnabled:   true,
			Description: "字节跳动火山引擎豆包，支持 doubao-pro、doubao-lite 系列",
		},
		{
			Name:        "hunyuan",
			Label:       "腾讯混元",
			BaseURL:     "https://api.hunyuan.cloud.tencent.com/v1",
			Model:       "hunyuan-pro",
			IsDefault:   false,
			IsEnabled:   true,
			Description: "腾讯混元大模型，支持 hunyuan-pro、hunyuan-standard",
		},
		{
			Name:        "baichuan",
			Label:       "百川智能",
			BaseURL:     "https://api.baichuan-ai.com/v1",
			Model:       "Baichuan4",
			IsDefault:   false,
			IsEnabled:   true,
			Description: "百川智能，支持 Baichuan4、Baichuan3-Turbo 等",
		},
		{
			Name:        "anthropic",
			Label:       "Anthropic Claude",
			BaseURL:     "https://api.anthropic.com/v1",
			Model:       "claude-3-5-sonnet-20241022",
			IsDefault:   false,
			IsEnabled:   true,
			Description: "Anthropic Claude 系列，claude-3-5-sonnet/haiku/opus",
		},
		{
			Name:        "gemini",
			Label:       "Google Gemini",
			BaseURL:     "https://generativelanguage.googleapis.com/v1beta/openai",
			Model:       "gemini-2.0-flash",
			IsDefault:   false,
			IsEnabled:   true,
			Description: "Google Gemini，支持 gemini-2.0-flash、gemini-1.5-pro",
		},
	}
	for _, p := range defaultProviders {
		var existing model.AIProvider
		if err := db.Where("name = ?", p.Name).First(&existing).Error; err != nil {
			// Not found — insert
			db.Create(&p)
		}
	}
	logger.Log.Info("Default AI providers seeded")

	// Seed default skills with tool definitions
	db.Model(&model.Skill{}).Count(&count)
	if count == 0 {
		skills := []model.Skill{
			{
				Name:        "云主机管理",
				Description: "云主机(Nova)相关操作：查询、创建、启停、重启、调整规格、快照等",
				Type:        "cloud_api",
				Config:      `{"service":"compute","api_version":"v2.1"}`,
				ToolDefs: `[
{"type":"function","function":{"name":"list_servers","description":"列举所有云主机及其详细信息","parameters":{"type":"object","properties":{}}}},
{"type":"function","function":{"name":"get_server","description":"查询指定云主机的详细信息","parameters":{"type":"object","properties":{"server_id":{"type":"string","description":"云主机ID"}},"required":["server_id"]}}},
{"type":"function","function":{"name":"create_server","description":"创建一台新的云主机","parameters":{"type":"object","properties":{"name":{"type":"string","description":"云主机名称"},"flavor_id":{"type":"string","description":"规格ID"},"image_id":{"type":"string","description":"镜像ID"},"network_id":{"type":"string","description":"网络ID"}},"required":["name","flavor_id","image_id","network_id"]}}},
{"type":"function","function":{"name":"start_server","description":"启动一台已停止的云主机","parameters":{"type":"object","properties":{"server_id":{"type":"string","description":"云主机ID"}},"required":["server_id"]}}},
{"type":"function","function":{"name":"stop_server","description":"关闭一台运行中的云主机","parameters":{"type":"object","properties":{"server_id":{"type":"string","description":"云主机ID"}},"required":["server_id"]}}},
{"type":"function","function":{"name":"reboot_server","description":"重启云主机","parameters":{"type":"object","properties":{"server_id":{"type":"string","description":"云主机ID"},"type":{"type":"string","enum":["SOFT","HARD"],"description":"重启类型"}},"required":["server_id"]}}},
{"type":"function","function":{"name":"delete_server","description":"删除云主机（危险操作）","parameters":{"type":"object","properties":{"server_id":{"type":"string","description":"云主机ID"}},"required":["server_id"]}}},
{"type":"function","function":{"name":"list_flavors","description":"列举所有可用的云主机规格","parameters":{"type":"object","properties":{}}}},
{"type":"function","function":{"name":"list_images","description":"列举所有可用镜像","parameters":{"type":"object","properties":{}}}}
]`,
				IsActive: true,
			},
			{
				Name:        "云硬盘管理",
				Description: "云硬盘(Cinder)相关操作：查询、创建、删除、扩容、快照管理",
				Type:        "cloud_api",
				Config:      `{"service":"storage","api_version":"v2"}`,
				ToolDefs: `[
{"type":"function","function":{"name":"list_volumes","description":"列举所有云硬盘及其详细信息","parameters":{"type":"object","properties":{}}}},
{"type":"function","function":{"name":"create_volume","description":"创建一个新的云硬盘","parameters":{"type":"object","properties":{"name":{"type":"string","description":"云硬盘名称"},"size":{"type":"integer","description":"大小(GB)"}},"required":["name","size"]}}},
{"type":"function","function":{"name":"delete_volume","description":"删除云硬盘（危险操作）","parameters":{"type":"object","properties":{"volume_id":{"type":"string","description":"云硬盘ID"}},"required":["volume_id"]}}},
{"type":"function","function":{"name":"extend_volume","description":"扩容云硬盘","parameters":{"type":"object","properties":{"volume_id":{"type":"string","description":"云硬盘ID"},"new_size":{"type":"integer","description":"新大小(GB)"}},"required":["volume_id","new_size"]}}},
{"type":"function","function":{"name":"list_volume_snapshots","description":"列举所有云硬盘快照","parameters":{"type":"object","properties":{}}}}
]`,
				IsActive: true,
			},
			{
				Name:        "网络管理",
				Description: "SDN网络(Neutron)相关操作：网络、子网、路由器、浮动IP、安全组等",
				Type:        "cloud_api",
				Config:      `{"service":"network","api_version":"v2.0"}`,
				ToolDefs: `[
{"type":"function","function":{"name":"list_networks","description":"列举所有网络","parameters":{"type":"object","properties":{}}}},
{"type":"function","function":{"name":"list_subnets","description":"列举所有子网","parameters":{"type":"object","properties":{}}}},
{"type":"function","function":{"name":"list_routers","description":"列举所有路由器","parameters":{"type":"object","properties":{}}}},
{"type":"function","function":{"name":"list_floating_ips","description":"列举所有浮动IP","parameters":{"type":"object","properties":{}}}},
{"type":"function","function":{"name":"list_security_groups","description":"列举所有安全组","parameters":{"type":"object","properties":{}}}},
{"type":"function","function":{"name":"create_security_group","description":"创建安全组","parameters":{"type":"object","properties":{"name":{"type":"string","description":"安全组名称"},"description":{"type":"string","description":"描述"}},"required":["name"]}}},
{"type":"function","function":{"name":"create_security_group_rule","description":"创建安全组规则","parameters":{"type":"object","properties":{"security_group_id":{"type":"string","description":"安全组ID"},"direction":{"type":"string","enum":["ingress","egress"]},"protocol":{"type":"string","description":"协议"},"port_range_min":{"type":"integer"},"port_range_max":{"type":"integer"}},"required":["security_group_id","direction"]}}}
]`,
				IsActive: true,
			},
			{
				Name:        "监控告警",
				Description: "监控(ECMS/Prometheus)相关操作：指标查询、告警查看",
				Type:        "cloud_api",
				Config:      `{"service":"monitor","api_version":"v1"}`,
				ToolDefs: `[
{"type":"function","function":{"name":"query_metrics","description":"查询监控指标数据(PromQL)","parameters":{"type":"object","properties":{"expr":{"type":"string","description":"PromQL查询表达式"},"start":{"type":"integer","description":"开始时间(Unix时间戳)"},"end":{"type":"integer","description":"结束时间(Unix时间戳)"},"step":{"type":"integer","description":"采样步长(秒)"}},"required":["expr"]}}},
{"type":"function","function":{"name":"list_alerts","description":"查询告警信息","parameters":{"type":"object","properties":{"states":{"type":"string","description":"告警状态过滤"},"severities":{"type":"string","description":"严重等级过滤"}}}}}
]`,
				IsActive: true,
			},
			{
				Name:        "负载均衡管理",
				Description: "负载均衡(Octavia)相关操作：查询LB、监听器、后端池",
				Type:        "cloud_api",
				Config:      `{"service":"loadbalancer","api_version":"v2.0"}`,
				ToolDefs: `[
{"type":"function","function":{"name":"list_loadbalancers","description":"列举所有负载均衡器","parameters":{"type":"object","properties":{}}}},
{"type":"function","function":{"name":"list_listeners","description":"列举所有监听器","parameters":{"type":"object","properties":{}}}},
{"type":"function","function":{"name":"list_pools","description":"列举所有后端池","parameters":{"type":"object","properties":{}}}}
]`,
				IsActive: true,
			},
		}
		for _, s := range skills {
			db.Create(&s)
		}
		logger.Log.Info("Default skills created")

		// Auto-associate all skills with the first agent (EasyStack 运维助手)
		var firstAgent model.Agent
		if err := db.First(&firstAgent).Error; err == nil {
			var allSkills []model.Skill
			db.Find(&allSkills)
			for _, s := range allSkills {
				db.Create(&model.AgentSkill{AgentID: firstAgent.ID, SkillID: s.ID})
			}
			logger.Log.Infof("Associated %d skills with agent '%s'", len(allSkills), firstAgent.Name)
		}
	}

	// Seed the "监控告警" (Monitoring Alarm) skill — idempotent insert by name
	seedMonitoringAlarmSkill(db)

	// Seed the "可观测员工" (Observable Employee) agent — idempotent insert by name
	seedObservableEmployeeAgent(db)
}

// seedMonitoringAlarmSkill creates the comprehensive monitoring alarm skill
// based on ECF 6.2.1 Observability (Ch.15) and Metering (Ch.14) APIs.
// It wraps: active alarms, recovered alarms, alarm severity, control plane status,
// storage cluster status, dashboard overview, metrics range query, service health check,
// resource top5, resource metric data, resource alarms, and alarm history.
func seedMonitoringAlarmSkill(db *gorm.DB) {
	const skillName = "监控告警(可观测)"

	var existing model.Skill
	if err := db.Where("name = ?", skillName).First(&existing).Error; err == nil {
		// Already exists — update ToolDefs to latest version
		existing.ToolDefs = monitoringAlarmToolDefs
		existing.Description = "EasyStack ECF 6.2.1 可观测与计量服务完整技能包：活跃告警、已恢复告警、告警等级统计、" +
			"控制面服务状态、存储集群状态、监控大盘概览、PromQL时序查询、全平台服务健康检查、" +
			"资源TOP5用量、资源监控数据、虚拟资源告警、告警历史"
		existing.Config = `{"service":"observability+metering","api_version":"v1+v2","domains":["emla.opsl2.svc.cluster.local","keystone.opsl2.svc.cluster.local"]}`
		db.Save(&existing)
		logger.Log.Infof("Updated monitoring alarm skill '%s'", skillName)
		return
	}

	skill := model.Skill{
		Name: skillName,
		Description: "EasyStack ECF 6.2.1 可观测与计量服务完整技能包：活跃告警、已恢复告警、告警等级统计、" +
			"控制面服务状态、存储集群状态、监控大盘概览、PromQL时序查询、全平台服务健康检查、" +
			"资源TOP5用量、资源监控数据、虚拟资源告警、告警历史",
		Type:     "cloud_api",
		Config:   `{"service":"observability+metering","api_version":"v1+v2","domains":["emla.opsl2.svc.cluster.local","keystone.opsl2.svc.cluster.local"]}`,
		ToolDefs: monitoringAlarmToolDefs,
		IsActive: true,
	}
	db.Create(&skill)
	logger.Log.Infof("Created monitoring alarm skill '%s' (ID=%d)", skillName, skill.ID)
}

// monitoringAlarmToolDefs is the JSON array of OpenAI-compatible tool definitions
// for the monitoring alarm skill, covering all ECF 6.2.1 observability + metering APIs.
const monitoringAlarmToolDefs = `[
{"type":"function","function":{"name":"list_active_alerts","description":"查询当前所有活跃(firing)告警，可按严重等级和分类过滤","parameters":{"type":"object","properties":{"severities":{"type":"string","description":"告警严重等级过滤，可选: critical, warning, info，多个用逗号分隔"},"categories":{"type":"string","description":"告警分类过滤，可选: service, storage, host, logging，多个用逗号分隔"}}}}},
{"type":"function","function":{"name":"list_recovered_alerts","description":"查询已恢复(resolved)的告警，支持时间范围和等级过滤","parameters":{"type":"object","properties":{"severities":{"type":"string","description":"告警严重等级过滤"},"categories":{"type":"string","description":"告警分类过滤"},"start":{"type":"string","description":"开始时间(Unix时间戳或ISO格式)"},"end":{"type":"string","description":"结束时间(Unix时间戳或ISO格式)"}}}}},
{"type":"function","function":{"name":"get_alarm_severity_summary","description":"获取告警等级统计摘要：critical/warning/info各有多少条告警","parameters":{"type":"object","properties":{}}}},
{"type":"function","function":{"name":"get_control_plane_status","description":"查询控制面所有云服务状态（计算、网络、存储、镜像、认证、监控、数据库、消息队列等20+服务的运行状态）","parameters":{"type":"object","properties":{}}}},
{"type":"function","function":{"name":"get_storage_cluster_status","description":"查询存储集群健康状态：Ceph健康、OSD数量与状态、容量、IOPS、吞吐量等","parameters":{"type":"object","properties":{}}}},
{"type":"function","function":{"name":"get_dashboard_overview","description":"获取监控大盘概览数据：云主机状态、CPU/内存/存储总量与使用率","parameters":{"type":"object","properties":{}}}},
{"type":"function","function":{"name":"query_metrics","description":"使用PromQL查询监控指标数据（即时查询）","parameters":{"type":"object","properties":{"expr":{"type":"string","description":"PromQL查询表达式，例如: up, node_cpu_seconds_total"},"start":{"type":"integer","description":"开始时间(Unix时间戳)"},"end":{"type":"integer","description":"结束时间(Unix时间戳)"},"step":{"type":"integer","description":"采样步长(秒)"}},"required":["expr"]}}},
{"type":"function","function":{"name":"query_metrics_range","description":"使用PromQL查询监控指标时间范围数据（范围查询，返回时序数据点）","parameters":{"type":"object","properties":{"expr":{"type":"string","description":"PromQL查询表达式"},"start":{"type":"integer","description":"开始时间(Unix时间戳)"},"end":{"type":"integer","description":"结束时间(Unix时间戳)"},"step":{"type":"integer","description":"采样步长(秒)"}},"required":["expr"]}}},
{"type":"function","function":{"name":"check_all_services_health","description":"全面检查所有云平台服务的健康状态（26+服务），返回每个服务的运行/告警/停止状态","parameters":{"type":"object","properties":{}}}},
{"type":"function","function":{"name":"get_resource_top5","description":"获取资源使用率TOP5（CPU或内存），返回使用率最高的5个资源","parameters":{"type":"object","properties":{"metric":{"type":"string","description":"指标名称: cpu.util 或 memory.util","enum":["cpu.util","memory.util"]},"start":{"type":"string","description":"开始时间"},"end":{"type":"string","description":"结束时间"}}}}},
{"type":"function","function":{"name":"get_resource_metric_data","description":"查询指定资源的监控时序数据（CPU利用率、内存、网络流量、磁盘IO等）","parameters":{"type":"object","properties":{"resource_id":{"type":"string","description":"资源ID（云主机ID等）"},"metric_name":{"type":"string","description":"指标名称，例如: cpu.util, memory.util, network.incoming.bytes.rate, disk.read.bytes.rate"},"start_time":{"type":"string","description":"开始时间(UTC格式: 2006-01-02T15:04:05)"},"stop_time":{"type":"string","description":"结束时间(UTC格式)"},"granularity":{"type":"string","description":"采样粒度(秒)，默认300"}},"required":["resource_id","metric_name"]}}},
{"type":"function","function":{"name":"list_resource_alarms","description":"列举所有虚拟资源告警规则（Ceilometer告警）","parameters":{"type":"object","properties":{}}}},
{"type":"function","function":{"name":"get_resource_alarm","description":"查询指定虚拟资源告警规则的详细信息","parameters":{"type":"object","properties":{"alarm_id":{"type":"string","description":"告警规则ID"}},"required":["alarm_id"]}}},
{"type":"function","function":{"name":"get_alarm_history","description":"查询指定告警规则的历史变更记录","parameters":{"type":"object","properties":{"alarm_id":{"type":"string","description":"告警规则ID"}},"required":["alarm_id"]}}},
{"type":"function","function":{"name":"list_alerts","description":"查询告警信息（通用），支持状态和等级过滤","parameters":{"type":"object","properties":{"states":{"type":"string","description":"告警状态过滤(firing/resolved)"},"severities":{"type":"string","description":"严重等级过滤(critical/warning/info)"}}}}}
]`

// seedObservableEmployeeAgent creates the "可观测员工" (Observable Employee) agent
// and associates it with the monitoring alarm skill and the first available EasyStack platform.
// This agent follows the Eino ReAct pattern: observe → think → act using tool calls.
func seedObservableEmployeeAgent(db *gorm.DB) {
	const agentName = "可观测员工"

	var existing model.Agent
	if err := db.Where("name = ?", agentName).First(&existing).Error; err == nil {
		// Already exists — update system prompt to latest version
		existing.SystemPrompt = observableEmployeeSystemPrompt()
		existing.Description = "基于 Eino ReAct 模式的可观测智能体，专注于 EasyStack 云平台的监控告警、服务健康检查、" +
			"存储集群状态、资源使用率分析。自动关联监控告警(可观测)技能并绑定 EasyStack 平台。"
		db.Save(&existing)
		logger.Log.Infof("Updated observable employee agent '%s'", agentName)

		// Ensure monitoring alarm skill association
		ensureObservableAgentSkillAssociation(db, existing.ID)
		return
	}

	// Find the first active EasyStack platform to bind
	var platform model.CloudPlatform
	var platformID *uint
	if err := db.Where("type = ? AND is_active = ?", "easystack", true).First(&platform).Error; err == nil {
		platformID = &platform.ID
	}

	agent := model.Agent{
		Name: agentName,
		Description: "基于 Eino ReAct 模式的可观测智能体，专注于 EasyStack 云平台的监控告警、服务健康检查、" +
			"存储集群状态、资源使用率分析。自动关联监控告警(可观测)技能并绑定 EasyStack 平台。",
		SystemPrompt:    observableEmployeeSystemPrompt(),
		Model:           "gpt-4",
		Temperature:     0.2,
		MaxTokens:       8192,
		CloudPlatformID: platformID,
		IsActive:        true,
		CreatedBy:       1,
	}
	db.Create(&agent)
	logger.Log.Infof("Created observable employee agent '%s' (ID=%d)", agentName, agent.ID)

	// Associate monitoring alarm skill
	ensureObservableAgentSkillAssociation(db, agent.ID)
}

// ensureObservableAgentSkillAssociation links the "监控告警(可观测)" skill to the agent.
func ensureObservableAgentSkillAssociation(db *gorm.DB, agentID uint) {
	var skill model.Skill
	if err := db.Where("name = ?", "监控告警(可观测)").First(&skill).Error; err != nil {
		logger.Log.Warnf("Monitoring alarm skill not found, cannot associate with agent %d", agentID)
		return
	}

	// Check if association already exists
	var existing model.AgentSkill
	if err := db.Where("agent_id = ? AND skill_id = ?", agentID, skill.ID).First(&existing).Error; err != nil {
		// Create association
		db.Create(&model.AgentSkill{AgentID: agentID, SkillID: skill.ID})
		logger.Log.Infof("Associated monitoring alarm skill (ID=%d) with agent (ID=%d)", skill.ID, agentID)
	}
}

// observableEmployeeSystemPrompt returns the comprehensive system prompt for the observable employee agent.
func observableEmployeeSystemPrompt() string {
	return "你是「可观测员工」——一个专业的 EasyStack 云平台可观测性与监控告警智能体。\n\n" +
		"## 角色定义\n" +
		"你是团队中负责云平台可观测性(Observability)的专职员工，基于 Eino ReAct（Observe-Think-Act）智能体模式运行：\n" +
		"1. **Observe（观察）**：通过工具调用采集实时告警、服务状态、监控指标等数据\n" +
		"2. **Think（思考）**：分析数据，判断问题根因、影响范围和紧急程度\n" +
		"3. **Act（行动）**：输出结构化的分析报告，提供处置建议\n\n" +
		"## 核心能力\n" +
		"### 1. 告警监控\n" +
		"- 查询活跃告警（firing）：按严重等级(critical/warning/info)和分类(service/storage/host/logging)过滤\n" +
		"- 查询已恢复告警（resolved）：支持时间范围查询\n" +
		"- 告警等级统计：快速了解各等级告警数量分布\n\n" +
		"### 2. 服务健康检查\n" +
		"- 控制面服务状态：检查计算(Nova)、网络(Neutron)、存储(Cinder)、镜像(Glance)、认证(Keystone)、监控(ECMS)等 20+ 服务的运行状态\n" +
		"- 全平台服务健康检查：26+ 服务的综合健康巡检\n" +
		"- 存储集群状态：Ceph 健康状态、OSD 数量、容量使用、IOPS、吞吐量\n\n" +
		"### 3. 指标查询与分析\n" +
		"- 监控大盘概览：云主机状态、CPU/内存/存储使用率\n" +
		"- PromQL 即时查询和范围查询\n" +
		"- 资源 TOP5 使用率排行（CPU/内存）\n" +
		"- 单资源时序监控数据查询\n\n" +
		"### 4. 虚拟资源告警\n" +
		"- Ceilometer 告警规则列表与详情\n" +
		"- 告警历史变更记录查询\n\n" +
		"## 连接的云平台\n" +
		"- **认证服务(Keystone)域名**: keystone.opsl2.svc.cluster.local\n" +
		"- **监控服务(EMLA)域名**: emla.opsl2.svc.cluster.local\n" +
		"- **平台类型**: EasyStack ECF 6.2.1\n\n" +
		"## 工作流程（ReAct 模式）\n" +
		"当用户提出请求时，按以下步骤执行：\n\n" +
		"**Step 1 - 理解需求**：分析用户问题，确定需要查询哪些数据\n" +
		"**Step 2 - 数据采集**：调用对应的工具函数获取实时数据\n" +
		"**Step 3 - 分析诊断**：对采集到的数据进行分析，识别异常、趋势和关联\n" +
		"**Step 4 - 输出报告**：用清晰的中文给出分析结果，包括：\n" +
		"  - 当前状态摘要\n" +
		"  - 发现的问题及严重程度\n" +
		"  - 可能的根因分析\n" +
		"  - 建议的处置措施\n\n" +
		"## 输出格式规范\n" +
		"- 使用 Markdown 格式组织输出\n" +
		"- 告警信息按严重等级排序：Critical > Warning > Info\n" +
		"- 服务状态使用状态标记：正常 | 异常 | 停止\n" +
		"- 数值类指标带单位（%, GB, IOPS, MB/s）\n" +
		"- 时间统一使用本地时间格式\n\n" +
		"## 注意事项\n" +
		"- 优先关注 critical 级别告警\n" +
		"- 发现服务不可用时立即提醒用户\n" +
		"- 对于存储集群容量不足(<20%)需特别告警\n" +
		"- 分析时结合多维度数据进行交叉验证\n" +
		"- 如果数据查询失败，说明可能原因并建议手动检查"
}
