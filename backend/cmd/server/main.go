package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
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

	// Serve frontend static files
	// Look for frontend dist in multiple locations
	frontendDist := os.Getenv("FRONTEND_DIST")
	if frontendDist == "" {
		// Try relative paths from the binary location
		candidates := []string{
			"./frontend/dist",          // when running from project root
			"../frontend/dist",         // when running from backend dir
			"../../frontend/dist",      // when running from backend/cmd/server
			"/app/frontend/dist",       // Docker path
		}
		for _, c := range candidates {
			if info, err := os.Stat(filepath.Join(c, "index.html")); err == nil && !info.IsDir() {
				frontendDist = c
				break
			}
		}
	}
	if frontendDist != "" {
		logger.Log.Infof("Serving frontend from: %s", frontendDist)
		r.Static("/assets", filepath.Join(frontendDist, "assets"))
		r.StaticFile("/favicon.ico", filepath.Join(frontendDist, "favicon.ico"))
		indexFile := filepath.Join(frontendDist, "index.html")
		r.GET("/", func(c *gin.Context) {
			c.File(indexFile)
		})
		r.NoRoute(func(c *gin.Context) {
			// Only serve index.html for non-API routes (SPA routing)
			if len(c.Request.URL.Path) > 4 && c.Request.URL.Path[:4] == "/api" {
				c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "not found"})
				return
			}
			c.File(indexFile)
		})
	} else {
		logger.Log.Warn("Frontend dist directory not found, serving API only")
		r.NoRoute(func(c *gin.Context) {
			c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "not found"})
		})
	}

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

			// Admin routes
			admin := auth.Group("")
			admin.Use(middleware.AdminMiddleware())
			{
				auth.GET("/users", h.ListUsers)
				auth.POST("/users", h.CreateUser)
				auth.PUT("/users/:id", h.UpdateUser)
				auth.DELETE("/users/:id", h.DeleteUser)
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
