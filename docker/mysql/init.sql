-- Cloud Agent Platform Database Initialization
-- Auto-migrated by GORM, this file is for manual reference

CREATE DATABASE IF NOT EXISTS cloud_agent
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_unicode_ci;

USE cloud_agent;

-- Grant permissions
GRANT ALL PRIVILEGES ON cloud_agent.* TO 'cloudagent'@'%';
FLUSH PRIVILEGES;

-- AI Providers table (also created by GORM AutoMigrate)
CREATE TABLE IF NOT EXISTS `ai_providers` (
  `id`          bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at`  datetime(3) DEFAULT NULL,
  `updated_at`  datetime(3) DEFAULT NULL,
  `deleted_at`  datetime(3) DEFAULT NULL,
  `name`        varchar(64)  NOT NULL COMMENT 'provider key, e.g. openai',
  `label`       varchar(128) NOT NULL COMMENT 'display name, e.g. OpenAI',
  `api_key`     varchar(512) DEFAULT '' COMMENT 'plaintext api key, empty until configured',
  `base_url`    varchar(512) DEFAULT '',
  `model`       varchar(128) DEFAULT '',
  `is_default`  tinyint(1)   DEFAULT 0,
  `is_enabled`  tinyint(1)   DEFAULT 1,
  `description` varchar(256) DEFAULT '',
  `icon_url`    varchar(512) DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_ai_providers_name` (`name`),
  KEY `idx_ai_providers_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Seed default AI providers (only insert if table is empty)
INSERT INTO `ai_providers` (`name`, `label`, `api_key`, `base_url`, `model`, `is_default`, `is_enabled`, `description`, `created_at`, `updated_at`)
SELECT * FROM (
  SELECT 'openai'   AS name, 'OpenAI'       AS label, '' AS api_key, 'https://api.openai.com/v1'                          AS base_url, 'gpt-4o'          AS model, 1 AS is_default, 1 AS is_enabled, 'OpenAI GPT 系列模型，支持 GPT-4o、GPT-4、GPT-3.5 等'              AS description, NOW() AS created_at, NOW() AS updated_at UNION ALL
  SELECT 'deepseek',           'DeepSeek',    '',       'https://api.deepseek.com/v1',                                                'deepseek-chat',   0,          1,          '深度求索 DeepSeek 系列模型，高性价比国产大模型',                    NOW(), NOW() UNION ALL
  SELECT 'qwen',               '通义千问',    '',       'https://dashscope.aliyuncs.com/compatible-mode/v1',                          'qwen-plus',       0,          1,          '阿里云通义千问系列模型，支持 Qwen-Plus、Qwen-Max 等',               NOW(), NOW() UNION ALL
  SELECT 'glm',                '智谱 GLM',    '',       'https://open.bigmodel.cn/api/paas/v4',                                       'glm-4',           0,          1,          '智谱 AI GLM 系列模型，支持 GLM-4、GLM-4-Flash 等',                 NOW(), NOW() UNION ALL
  SELECT 'minimax',            'MiniMax',     '',       'https://api.minimax.chat/v1',                                                'abab6.5s-chat',   0,          1,          'MiniMax 大模型，支持 abab6.5s-chat 等系列',                         NOW(), NOW()
) AS tmp
WHERE NOT EXISTS (SELECT 1 FROM `ai_providers` LIMIT 1);
