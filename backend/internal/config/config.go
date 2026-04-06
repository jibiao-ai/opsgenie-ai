package config

import (
	"os"
	"strconv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	RabbitMQ RabbitMQConfig
	EasyStack EasyStackConfig
	AI       AIConfig
}

type ServerConfig struct {
	Port string
	Mode string
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

type AIConfig struct {
	Provider   string // openai, azure, local
	APIKey     string
	BaseURL    string
	Model      string
}

type RabbitMQConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	VHost    string
}

type EasyStackConfig struct {
	AuthURL    string
	Username   string
	Password   string
	DomainName string
	ProjectName string
	ProjectID  string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
			Mode: getEnv("GIN_MODE", "debug"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "mysql"),
			Port:     getEnvInt("DB_PORT", 3306),
			User:     getEnv("DB_USER", "cloudagent"),
			Password: getEnv("DB_PASSWORD", "cloudagent123"),
			DBName:   getEnv("DB_NAME", "opsgenie_ai"),
		},
		RabbitMQ: RabbitMQConfig{
			Host:     getEnv("RABBITMQ_HOST", "rabbitmq"),
			Port:     getEnvInt("RABBITMQ_PORT", 5672),
			User:     getEnv("RABBITMQ_USER", "guest"),
			Password: getEnv("RABBITMQ_PASSWORD", "guest"),
			VHost:    getEnv("RABBITMQ_VHOST", "/"),
		},
		EasyStack: EasyStackConfig{
			AuthURL:    getEnv("EASYSTACK_AUTH_URL", "http://keystone.example.com"),
			Username:   getEnv("EASYSTACK_USERNAME", "admin"),
			Password:   getEnv("EASYSTACK_PASSWORD", ""),
			DomainName: getEnv("EASYSTACK_DOMAIN", "default"),
			ProjectName: getEnv("EASYSTACK_PROJECT", "admin"),
			ProjectID:  getEnv("EASYSTACK_PROJECT_ID", ""),
		},
		AI: AIConfig{
			Provider: getEnv("AI_PROVIDER", "openai"),
			APIKey:   getEnv("AI_API_KEY", ""),
			BaseURL:  getEnv("AI_BASE_URL", "https://api.openai.com/v1"),
			Model:    getEnv("AI_MODEL", "gpt-4"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
