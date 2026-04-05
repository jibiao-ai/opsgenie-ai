package handler

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jibiao-ai/cloud-agent/internal/model"
	"github.com/jibiao-ai/cloud-agent/internal/repository"
	"github.com/jibiao-ai/cloud-agent/internal/service"
	"github.com/jibiao-ai/cloud-agent/pkg/logger"
	"github.com/jibiao-ai/cloud-agent/pkg/response"
)

type Handler struct {
	chatService *service.ChatService
}

func NewHandler(chatService *service.ChatService) *Handler {
	return &Handler{chatService: chatService}
}

// ==================== Auth ====================

func (h *Handler) Login(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}

	resp, err := service.Login(req)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	response.Success(c, resp)
}

func (h *Handler) GetProfile(c *gin.Context) {
	userID := c.GetUint("user_id")
	var user model.User
	if err := service.GetUserByID(userID, &user); err != nil {
		response.InternalError(c, "user not found")
		return
	}
	response.Success(c, user)
}

// ==================== Dashboard ====================

func (h *Handler) GetDashboard(c *gin.Context) {
	userID := c.GetUint("user_id")
	stats, err := h.chatService.GetDashboardStats(userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, stats)
}

// ==================== Agents ====================

func (h *Handler) ListAgents(c *gin.Context) {
	agents, err := h.chatService.GetAgents()
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, agents)
}

func (h *Handler) GetAgent(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	agent, err := h.chatService.GetAgent(uint(id))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, agent)
}

func (h *Handler) CreateAgent(c *gin.Context) {
	var agent model.Agent
	if err := c.ShouldBindJSON(&agent); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}
	agent.CreatedBy = c.GetUint("user_id")
	if err := h.chatService.CreateAgent(&agent); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, agent)
}

func (h *Handler) UpdateAgent(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var agent model.Agent
	if err := c.ShouldBindJSON(&agent); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}
	agent.ID = uint(id)
	if err := h.chatService.UpdateAgent(&agent); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, agent)
}

func (h *Handler) DeleteAgent(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := h.chatService.DeleteAgent(uint(id)); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, nil)
}

// ==================== Conversations ====================

func (h *Handler) ListConversations(c *gin.Context) {
	userID := c.GetUint("user_id")
	convs, err := h.chatService.GetConversations(userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, convs)
}

func (h *Handler) CreateConversation(c *gin.Context) {
	var req struct {
		AgentID uint   `json:"agent_id" binding:"required"`
		Title   string `json:"title"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: agent_id is required")
		return
	}
	if req.Title == "" {
		req.Title = "新会话"
	}
	userID := c.GetUint("user_id")
	conv, err := h.chatService.CreateConversation(userID, req.AgentID, req.Title)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, conv)
}

func (h *Handler) DeleteConversation(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	userID := c.GetUint("user_id")
	if err := h.chatService.DeleteConversation(uint(id), userID); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, nil)
}

// ==================== Messages ====================

func (h *Handler) GetMessages(c *gin.Context) {
	convID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	userID := c.GetUint("user_id")
	msgs, err := h.chatService.GetMessages(uint(convID), userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, msgs)
}

func (h *Handler) SendMessage(c *gin.Context) {
	convID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	userID := c.GetUint("user_id")

	var req struct {
		Content     string   `json:"content" binding:"required"`
		Attachments []string `json:"attachments"` // file paths from upload
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "content is required")
		return
	}

	// Build the full message content including attachment info
	fullContent := req.Content
	if len(req.Attachments) > 0 {
		fullContent += "\n\n[附件信息]\n"
		for _, att := range req.Attachments {
			// Read file content for text-based files
			content, err := readAttachmentContent(att)
			if err == nil && content != "" {
				fullContent += fmt.Sprintf("文件: %s\n内容:\n%s\n---\n", filepath.Base(att), content)
			} else {
				fullContent += fmt.Sprintf("文件: %s (二进制文件，无法读取内容)\n", filepath.Base(att))
			}
		}
	}

	userMsg, assistantMsg, err := h.chatService.SendMessage(uint(convID), userID, fullContent, nil)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"user_message":      userMsg,
		"assistant_message": assistantMsg,
	})
}

// ==================== File Upload ====================

// UploadFile handles file uploads for chat attachments
func (h *Handler) UploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "请选择要上传的文件")
		return
	}

	// Validate file size (max 10MB)
	if file.Size > 10*1024*1024 {
		response.BadRequest(c, "文件大小不能超过 10MB")
		return
	}

	// Create uploads directory if not exists
	uploadDir := "uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		response.InternalError(c, "创建上传目录失败")
		return
	}

	// Generate unique filename
	ext := filepath.Ext(file.Filename)
	randBytes := make([]byte, 16)
	rand.Read(randBytes)
	newFilename := hex.EncodeToString(randBytes) + ext
	filePath := filepath.Join(uploadDir, newFilename)

	// Save file
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		response.InternalError(c, "保存文件失败")
		return
	}

	logger.Log.Infof("File uploaded: %s -> %s (%d bytes)", file.Filename, filePath, file.Size)

	response.Success(c, gin.H{
		"filename":      file.Filename,
		"filepath":      filePath,
		"size":          file.Size,
		"content_type":  file.Header.Get("Content-Type"),
	})
}

// readAttachmentContent reads the content of a text-based attachment file
func readAttachmentContent(filePath string) (string, error) {
	// Only read text-based files
	ext := strings.ToLower(filepath.Ext(filePath))
	textExts := map[string]bool{
		".txt": true, ".md": true, ".csv": true, ".json": true,
		".yaml": true, ".yml": true, ".xml": true, ".log": true,
		".conf": true, ".cfg": true, ".ini": true, ".sh": true,
		".py": true, ".go": true, ".js": true, ".ts": true,
		".html": true, ".css": true, ".sql": true, ".env": true,
	}
	if !textExts[ext] {
		return "", fmt.Errorf("not a text file")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	content := string(data)
	// Truncate very large files
	if len(content) > 8000 {
		content = content[:8000] + "\n... (文件内容过长，已截断)"
	}
	return content, nil
}

// ==================== WebSocket Chat ====================

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type WSMessage struct {
	Type           string `json:"type"` // message, heartbeat
	Content        string `json:"content,omitempty"`
	ConversationID uint   `json:"conversation_id,omitempty"`
}

func (h *Handler) WebSocketChat(c *gin.Context) {
	userID := c.GetUint("user_id")

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Log.Errorf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	var mu sync.Mutex

	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				logger.Log.Errorf("WebSocket read error: %v", err)
			}
			break
		}

		var wsMsg WSMessage
		if err := json.Unmarshal(msgBytes, &wsMsg); err != nil {
			continue
		}

		if wsMsg.Type == "heartbeat" {
			mu.Lock()
			conn.WriteJSON(WSMessage{Type: "heartbeat"})
			mu.Unlock()
			continue
		}

		if wsMsg.Type == "message" && wsMsg.Content != "" {
			// Send typing indicator
			mu.Lock()
			conn.WriteJSON(gin.H{"type": "typing", "content": ""})
			mu.Unlock()

			// Process message
			go func() {
				userMsg, assistantMsg, err := h.chatService.SendMessage(wsMsg.ConversationID, userID, wsMsg.Content, nil)
				mu.Lock()
				defer mu.Unlock()

				if err != nil {
					conn.WriteJSON(gin.H{
						"type":    "error",
						"content": err.Error(),
					})
					return
				}

				conn.WriteJSON(gin.H{
					"type":              "user_message",
					"message":           userMsg,
					"conversation_id":   wsMsg.ConversationID,
				})
				conn.WriteJSON(gin.H{
					"type":              "assistant_message",
					"message":           assistantMsg,
					"conversation_id":   wsMsg.ConversationID,
				})
			}()
		}
	}
}

// ==================== Skills ====================

func (h *Handler) ListSkills(c *gin.Context) {
	skills, err := h.chatService.GetSkills()
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, skills)
}

// ==================== Workflows ====================

func (h *Handler) ListWorkflows(c *gin.Context) {
	workflows, err := h.chatService.GetWorkflows()
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, workflows)
}

func (h *Handler) CreateWorkflow(c *gin.Context) {
	var wf model.Workflow
	if err := c.ShouldBindJSON(&wf); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}
	wf.CreatedBy = c.GetUint("user_id")
	if err := h.chatService.CreateWorkflow(&wf); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, wf)
}

// ==================== Scheduled Tasks ====================

func (h *Handler) ListScheduledTasks(c *gin.Context) {
	tasks, err := h.chatService.GetScheduledTasks()
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, tasks)
}

func (h *Handler) CreateScheduledTask(c *gin.Context) {
	var task model.ScheduledTask
	if err := c.ShouldBindJSON(&task); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}
	task.CreatedBy = c.GetUint("user_id")
	if err := h.chatService.CreateScheduledTask(&task); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, task)
}

// ==================== Users (Admin) ====================

func (h *Handler) ListUsers(c *gin.Context) {
	users, err := service.GetUsers()
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, users)
}

func (h *Handler) CreateUser(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}
	user := model.User{
		Username: req.Username,
		Password: req.Password,
		Email:    req.Email,
		Role:     req.Role,
	}
	if err := service.CreateUser(&user); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	// Clear password hash from response
	user.Password = ""
	response.Success(c, user)
}

func (h *Handler) UpdateUser(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}
	user := model.User{
		Username: req.Username,
		Password: req.Password,
		Email:    req.Email,
		Role:     req.Role,
	}
	user.ID = uint(id)
	if err := service.UpdateUser(&user); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	// Clear password hash from response
	user.Password = ""
	response.Success(c, user)
}

func (h *Handler) DeleteUser(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := service.DeleteUser(uint(id)); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, nil)
}

// ==================== Task Logs ====================

func (h *Handler) ListTaskLogs(c *gin.Context) {
	userID := c.GetUint("user_id")
	logs, err := h.chatService.GetTaskLogs(userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, logs)
}

// ==================== AI Providers ====================

// maskAPIKey masks all but the first 4 characters of an API key
func maskAPIKey(key string) string {
	if len(key) <= 4 {
		return key
	}
	return key[:4] + "****"
}

// GetAIProviders returns all AI providers with masked API keys
func (h *Handler) GetAIProviders(c *gin.Context) {
	var providers []model.AIProvider
	if err := repository.DB.Find(&providers).Error; err != nil {
		response.InternalError(c, err.Error())
		return
	}

	// Mask API keys before returning
	type AIProviderView struct {
		ID          uint      `json:"id"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		Name        string    `json:"name"`
		Label       string    `json:"label"`
		APIKey      string    `json:"api_key"` // masked
		BaseURL     string    `json:"base_url"`
		Model       string    `json:"model"`
		IsDefault   bool      `json:"is_default"`
		IsEnabled   bool      `json:"is_enabled"`
		Description string    `json:"description"`
		IconURL     string    `json:"icon_url"`
		Configured  bool      `json:"configured"` // true if api_key is set
	}

	views := make([]AIProviderView, len(providers))
	for i, p := range providers {
		views[i] = AIProviderView{
			ID:          p.ID,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
			Name:        p.Name,
			Label:       p.Label,
			APIKey:      maskAPIKey(p.APIKey),
			BaseURL:     p.BaseURL,
			Model:       p.Model,
			IsDefault:   p.IsDefault,
			IsEnabled:   p.IsEnabled,
			Description: p.Description,
			IconURL:     p.IconURL,
			Configured:  p.APIKey != "",
		}
	}
	response.Success(c, views)
}

// UpdateAIProvider updates provider config and optionally sets it as default
func (h *Handler) UpdateAIProvider(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req struct {
		APIKey    string `json:"api_key"`
		BaseURL   string `json:"base_url"`
		Model     string `json:"model"`
		IsDefault bool   `json:"is_default"`
		IsEnabled bool   `json:"is_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}

	var provider model.AIProvider
	if err := repository.DB.First(&provider, id).Error; err != nil {
		response.BadRequest(c, "provider not found")
		return
	}

	// If api_key is provided (non-empty and not a masked value), update it
	if req.APIKey != "" && req.APIKey != maskAPIKey(provider.APIKey) {
		provider.APIKey = req.APIKey
	}
	if req.BaseURL != "" {
		provider.BaseURL = req.BaseURL
	}
	if req.Model != "" {
		provider.Model = req.Model
	}
	provider.IsEnabled = req.IsEnabled

	// Handle default: unset all others first if setting as default
	if req.IsDefault {
		if err := repository.DB.Model(&model.AIProvider{}).Where("id != ?", id).Update("is_default", false).Error; err != nil {
			response.InternalError(c, err.Error())
			return
		}
		provider.IsDefault = true
	} else {
		provider.IsDefault = false
	}

	if err := repository.DB.Save(&provider).Error; err != nil {
		response.InternalError(c, err.Error())
		return
	}

	// Return masked view
	provider.APIKey = maskAPIKey(provider.APIKey)
	response.Success(c, provider)
}

// TestAIProvider sends a test request to the AI provider's API
func (h *Handler) TestAIProvider(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var provider model.AIProvider
	if err := repository.DB.First(&provider, id).Error; err != nil {
		response.BadRequest(c, "provider not found")
		return
	}

	if provider.APIKey == "" {
		response.BadRequest(c, "API Key 未配置，请先保存 API Key")
		return
	}

	baseURL := provider.BaseURL
	if baseURL == "" {
		response.BadRequest(c, "Base URL 未配置")
		return
	}

	modelName := provider.Model
	if modelName == "" {
		modelName = "gpt-3.5-turbo"
	}

	// Build the test request payload (OpenAI-compatible chat completion)
	// NOTE: Use a minimal, widely-compatible payload.
	// - Use max_tokens = 10 (some providers reject very small values).
	// - Do NOT include "tools", "functions", or "stream" — many providers
	//   (e.g. SiliconFlow) return 403/400 for unsupported parameters.
	payload := map[string]interface{}{
		"model": modelName,
		"messages": []map[string]string{
			{"role": "user", "content": "Hi"},
		},
		"max_tokens": 10,
	}
	payloadBytes, _ := json.Marshal(payload)

	// Determine the chat completions endpoint
	endpoint := fmt.Sprintf("%s/chat/completions", strings.TrimRight(baseURL, "/"))

	// Create HTTP request
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		response.InternalError(c, fmt.Sprintf("创建请求失败: %v", err))
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+provider.APIKey)

	// Execute with a 15-second timeout
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Warnf("AI provider test failed for %s: %v", provider.Name, err)
		response.BadRequest(c, fmt.Sprintf("连接失败: %v", err))
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		response.Success(c, gin.H{
			"status":  "ok",
			"message": "连接成功，API Key 有效",
		})
		return
	}

	// Parse error response
	var errResp map[string]interface{}
	json.Unmarshal(bodyBytes, &errResp)
	errMsg := fmt.Sprintf("API 返回错误 (HTTP %d)", resp.StatusCode)
	if errResp != nil {
		if e, ok := errResp["error"]; ok {
			if eMap, ok := e.(map[string]interface{}); ok {
				if msg, ok := eMap["message"].(string); ok {
					errMsg = msg
				}
			}
		}
	}
	response.BadRequest(c, errMsg)
}

// ==================== Cloud Platforms ====================

// ListCloudPlatforms returns all cloud platforms
func (h *Handler) ListCloudPlatforms(c *gin.Context) {
	var platforms []model.CloudPlatform
	if err := repository.DB.Find(&platforms).Error; err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, platforms)
}

// CreateCloudPlatform creates a new cloud platform entry
func (h *Handler) CreateCloudPlatform(c *gin.Context) {
	var req struct {
		Name            string `json:"name" binding:"required"`
		Type            string `json:"type" binding:"required"`
		AuthURL         string `json:"auth_url"`
		Username        string `json:"username"`
		Password        string `json:"password"`
		DomainName      string `json:"domain_name"`
		ProjectName     string `json:"project_name"`
		ProjectID       string `json:"project_id"`
		AccessKeyID     string `json:"access_key_id"`
		AccessKeySecret string `json:"access_key_secret"`
		Endpoint        string `json:"endpoint"`
		Description     string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	if req.Type != "easystack" && req.Type != "zstack" {
		response.BadRequest(c, "type must be easystack or zstack")
		return
	}
	platform := model.CloudPlatform{
		Name:            req.Name,
		Type:            req.Type,
		AuthURL:         req.AuthURL,
		Username:        req.Username,
		Password:        req.Password,
		DomainName:      req.DomainName,
		ProjectName:     req.ProjectName,
		ProjectID:       req.ProjectID,
		AccessKeyID:     req.AccessKeyID,
		AccessKeySecret: req.AccessKeySecret,
		Endpoint:        req.Endpoint,
		Description:     req.Description,
		IsActive:        true,
		Status:          "unknown",
		CreatedBy:       c.GetUint("user_id"),
	}
	if err := repository.DB.Create(&platform).Error; err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, platform)
}

// UpdateCloudPlatform updates a cloud platform entry
func (h *Handler) UpdateCloudPlatform(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var platform model.CloudPlatform
	if err := repository.DB.First(&platform, id).Error; err != nil {
		response.BadRequest(c, "platform not found")
		return
	}
	var req struct {
		Name            string `json:"name"`
		Type            string `json:"type"`
		AuthURL         string `json:"auth_url"`
		Username        string `json:"username"`
		Password        string `json:"password"`
		DomainName      string `json:"domain_name"`
		ProjectName     string `json:"project_name"`
		ProjectID       string `json:"project_id"`
		AccessKeyID     string `json:"access_key_id"`
		AccessKeySecret string `json:"access_key_secret"`
		Endpoint        string `json:"endpoint"`
		Description     string `json:"description"`
		IsActive        *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}
	if req.Name != "" {
		platform.Name = req.Name
	}
	if req.Type != "" {
		platform.Type = req.Type
	}
	if req.AuthURL != "" {
		platform.AuthURL = req.AuthURL
	}
	if req.Username != "" {
		platform.Username = req.Username
	}
	if req.Password != "" {
		platform.Password = req.Password
	}
	if req.DomainName != "" {
		platform.DomainName = req.DomainName
	}
	if req.ProjectName != "" {
		platform.ProjectName = req.ProjectName
	}
	if req.ProjectID != "" {
		platform.ProjectID = req.ProjectID
	}
	if req.AccessKeyID != "" {
		platform.AccessKeyID = req.AccessKeyID
	}
	if req.AccessKeySecret != "" {
		platform.AccessKeySecret = req.AccessKeySecret
	}
	if req.Endpoint != "" {
		platform.Endpoint = req.Endpoint
	}
	if req.Description != "" {
		platform.Description = req.Description
	}
	if req.IsActive != nil {
		platform.IsActive = *req.IsActive
	}
	if err := repository.DB.Save(&platform).Error; err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, platform)
}

// DeleteCloudPlatform soft-deletes a cloud platform
func (h *Handler) DeleteCloudPlatform(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := repository.DB.Delete(&model.CloudPlatform{}, id).Error; err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, nil)
}

// TestCloudPlatform tests connectivity to a cloud platform
func (h *Handler) TestCloudPlatform(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var platform model.CloudPlatform
	if err := repository.DB.First(&platform, id).Error; err != nil {
		response.BadRequest(c, "platform not found")
		return
	}

	client := &http.Client{Timeout: 15 * time.Second}
	var testErr error

	switch platform.Type {
	case "easystack":
		// EasyStack: call Keystone v3 token endpoint
		if platform.AuthURL == "" || platform.Username == "" || platform.Password == "" {
			response.BadRequest(c, "EasyStack 平台缺少 AuthURL、Username 或 Password")
			return
		}
		keystoneURL := strings.TrimRight(platform.AuthURL, "/") + "/v3/auth/tokens"
		domain := platform.DomainName
		if domain == "" {
			domain = "Default"
		}
		authPayload := map[string]interface{}{
			"auth": map[string]interface{}{
				"identity": map[string]interface{}{
					"methods": []string{"password"},
					"password": map[string]interface{}{
						"user": map[string]interface{}{
							"name":     platform.Username,
							"password": platform.Password,
							"domain":   map[string]string{"name": domain},
						},
					},
				},
			},
		}
		body, _ := json.Marshal(authPayload)
		req, err := http.NewRequest("POST", keystoneURL, bytes.NewReader(body))
		if err != nil {
			testErr = fmt.Errorf("创建请求失败: %v", err)
			break
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			testErr = fmt.Errorf("连接失败: %v", err)
			break
		}
		defer resp.Body.Close()
		if resp.StatusCode == 201 || resp.StatusCode == 200 {
			repository.DB.Model(&platform).Update("status", "connected")
			response.Success(c, gin.H{"status": "connected", "message": "EasyStack Keystone 认证成功"})
			return
		}
		respBody, _ := io.ReadAll(resp.Body)
		testErr = fmt.Errorf("认证失败 (HTTP %d): %s", resp.StatusCode, string(respBody))

	case "zstack":
		// ZStack: call /zstack/v1/accounts/login
		if platform.Endpoint == "" {
			response.BadRequest(c, "ZStack 平台缺少 Endpoint")
			return
		}
		var loginURL string
		var loginBody []byte
		if platform.AccessKeyID != "" && platform.AccessKeySecret != "" {
			loginURL = strings.TrimRight(platform.Endpoint, "/") + "/zstack/v1/accounts/login"
			loginPayload := map[string]interface{}{
				"logInByExactAccount": map[string]string{
					"accountName": platform.AccessKeyID,
					"password":    platform.AccessKeySecret,
				},
			}
			loginBody, _ = json.Marshal(loginPayload)
		} else if platform.Username != "" && platform.Password != "" {
			loginURL = strings.TrimRight(platform.Endpoint, "/") + "/zstack/v1/accounts/login"
			loginPayload := map[string]interface{}{
				"logInByExactAccount": map[string]string{
					"accountName": platform.Username,
					"password":    platform.Password,
				},
			}
			loginBody, _ = json.Marshal(loginPayload)
		} else {
			response.BadRequest(c, "ZStack 平台缺少认证信息(AccessKey 或 Username/Password)")
			return
		}
		req, err := http.NewRequest("PUT", loginURL, bytes.NewReader(loginBody))
		if err != nil {
			testErr = fmt.Errorf("创建请求失败: %v", err)
			break
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			testErr = fmt.Errorf("连接失败: %v", err)
			break
		}
		defer resp.Body.Close()
		if resp.StatusCode == 200 || resp.StatusCode == 201 {
			repository.DB.Model(&platform).Update("status", "connected")
			response.Success(c, gin.H{"status": "connected", "message": "ZStack 登录认证成功"})
			return
		}
		respBody, _ := io.ReadAll(resp.Body)
		testErr = fmt.Errorf("认证失败 (HTTP %d): %s", resp.StatusCode, string(respBody))

	default:
		response.BadRequest(c, "不支持的平台类型: "+platform.Type)
		return
	}

	if testErr != nil {
		repository.DB.Model(&platform).Update("status", "failed")
		logger.Log.Warnf("Cloud platform test failed for %s(%s): %v", platform.Name, platform.Type, testErr)
		response.BadRequest(c, testErr.Error())
	}
}
