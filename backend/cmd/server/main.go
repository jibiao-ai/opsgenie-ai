package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jibiao-ai/cloud-agent/internal/agent"
	"github.com/jibiao-ai/cloud-agent/internal/config"
	"github.com/jibiao-ai/cloud-agent/internal/easystack"
	"github.com/jibiao-ai/cloud-agent/internal/handler"
	"github.com/jibiao-ai/cloud-agent/internal/middleware"
	"github.com/jibiao-ai/cloud-agent/internal/mq"
	"github.com/jibiao-ai/cloud-agent/internal/repository"
	"github.com/jibiao-ai/cloud-agent/internal/service"
	"github.com/jibiao-ai/cloud-agent/pkg/logger"
)

func main() {
	// Initialize logger
	logger.Init()
	logger.Log.Info("Starting Cloud Agent Platform...")

	// Load configuration
	cfg := config.Load()

	// Set Gin mode
	gin.SetMode(cfg.Server.Mode)

	// Initialize database
	if err := repository.InitDB(cfg.Database); err != nil {
		logger.Log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize RabbitMQ
	rabbitMQ := mq.NewRabbitMQ(cfg.RabbitMQ)
	if err := rabbitMQ.Connect(); err != nil {
		logger.Log.Warnf("Failed to connect to RabbitMQ (will continue without MQ): %v", err)
	} else {
		defer rabbitMQ.Close()

		// Start consuming task queue
		rabbitMQ.Consume(mq.QueueAgentTask, func(msg mq.TaskMessage) error {
			logger.Log.Infof("Processing task: %s (type: %s)", msg.ID, msg.Type)
			return nil
		})
	}

	// Initialize EasyStack client
	esClient := easystack.NewClient(cfg.EasyStack)

	// Initialize AI Agent
	aiAgent := agent.NewAgent(cfg.AI, esClient)

	// Initialize services
	chatService := service.NewChatService(aiAgent)

	// Initialize handlers
	h := handler.NewHandler(chatService)

	// Setup Gin router
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// CORS
	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool { return true },
		AllowMethods:    []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:    []string{"Origin", "Content-Type", "Authorization", "Accept"},
		ExposeHeaders:   []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Backend serves API only. Frontend static files are served by the nginx container.
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "not found"})
	})

	// API routes
	api := r.Group("/api")
	{
		// Public routes
		api.POST("/login", h.Login)

		// Protected routes
		auth := api.Group("")
		auth.Use(middleware.AuthMiddleware())
		{
			// Profile
			auth.GET("/profile", h.GetProfile)

			// Dashboard
			auth.GET("/dashboard", h.GetDashboard)

			// Agents
			auth.GET("/agents", h.ListAgents)
			auth.GET("/agents/:id", h.GetAgent)
			auth.POST("/agents", h.CreateAgent)
			auth.PUT("/agents/:id", h.UpdateAgent)
			auth.DELETE("/agents/:id", h.DeleteAgent)

			// Conversations
			auth.GET("/conversations", h.ListConversations)
			auth.POST("/conversations", h.CreateConversation)
			auth.DELETE("/conversations/:id", h.DeleteConversation)

			// Messages
			auth.GET("/conversations/:id/messages", h.GetMessages)
			auth.POST("/conversations/:id/messages", h.SendMessage)

			// WebSocket
			auth.GET("/ws", h.WebSocketChat)

			// Skills
			auth.GET("/skills", h.ListSkills)

			// Workflows
			auth.GET("/workflows", h.ListWorkflows)
			auth.POST("/workflows", h.CreateWorkflow)

			// Scheduled Tasks
			auth.GET("/scheduled-tasks", h.ListScheduledTasks)
			auth.POST("/scheduled-tasks", h.CreateScheduledTask)

			// Task Logs
			auth.GET("/task-logs", h.ListTaskLogs)

			// AI Providers
			auth.GET("/ai-providers", h.GetAIProviders)
			auth.PUT("/ai-providers/:id", h.UpdateAIProvider)
			auth.POST("/ai-providers/:id/test", h.TestAIProvider)

			// Cloud Platforms
			auth.GET("/cloud-platforms", h.ListCloudPlatforms)
			auth.POST("/cloud-platforms", h.CreateCloudPlatform)
			auth.PUT("/cloud-platforms/:id", h.UpdateCloudPlatform)
			auth.DELETE("/cloud-platforms/:id", h.DeleteCloudPlatform)
			auth.POST("/cloud-platforms/:id/test", h.TestCloudPlatform)

			// Admin routes
			admin := auth.Group("")
			admin.Use(middleware.AdminMiddleware())
			{
				admin.GET("/users", h.ListUsers)
				admin.POST("/users", h.CreateUser)
				admin.PUT("/users/:id", h.UpdateUser)
				admin.DELETE("/users/:id", h.DeleteUser)
			}
		}
	}

	// Start server
	port := cfg.Server.Port
	logger.Log.Infof("Server starting on port %s", port)

	// Graceful shutdown
	go func() {
		if err := r.Run(fmt.Sprintf(":%s", port)); err != nil {
			logger.Log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Log.Info("Shutting down server...")
}
