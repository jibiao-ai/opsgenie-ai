package handler

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
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
	"github.com/jibiao-ai/opsgenie-ai/internal/agent"
	"github.com/jibiao-ai/opsgenie-ai/internal/model"
	"github.com/jibiao-ai/opsgenie-ai/internal/repository"
	"github.com/jibiao-ai/opsgenie-ai/internal/service"
	"github.com/jibiao-ai/opsgenie-ai/pkg/logger"
	"github.com/jibiao-ai/opsgenie-ai/pkg/response"
)

type Handler struct {
	chatService *service.ChatService
}

func NewHandler(chatService *service.ChatService) *Handler {
	return &Handler{chatService: chatService}
}

// ==================== Operation Log Helper ====================

// recordOperationLog persists an operation log entry. It is fire-and-forget;
// errors are logged but do not affect the calling handler's response.
func recordOperationLog(c *gin.Context, module, action string, targetID uint, targetName, detail string) {
	userID := c.GetUint("user_id")
	username := ""
	var user model.User
	if err := repository.DB.Select("username").First(&user, userID).Error; err == nil {
		username = user.Username
	}
	ip := c.ClientIP()
	log := model.OperationLog{
		UserID:     userID,
		Username:   username,
		Module:     module,
		Action:     action,
		TargetID:   targetID,
		TargetName: targetName,
		Detail:     detail,
		IP:         ip,
	}
	if err := repository.DB.Create(&log).Error; err != nil {
		logger.Log.Warnf("Failed to record operation log: %v", err)
	}
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
	var req struct {
		Name            string   `json:"name"`
		Description     string   `json:"description"`
		SystemPrompt    string   `json:"system_prompt"`
		Model           string   `json:"model"`
		Temperature     float64  `json:"temperature"`
		MaxTokens       int      `json:"max_tokens"`
		IsActive        bool     `json:"is_active"`
		SkillIDs        []uint   `json:"skill_ids"`
		CloudPlatformID *uint    `json:"cloud_platform_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}
	agent := model.Agent{
		Name:            req.Name,
		Description:     req.Description,
		SystemPrompt:    req.SystemPrompt,
		Model:           req.Model,
		Temperature:     req.Temperature,
		MaxTokens:       req.MaxTokens,
		IsActive:        req.IsActive,
		CloudPlatformID: req.CloudPlatformID,
		CreatedBy:       c.GetUint("user_id"),
	}
	if err := h.chatService.CreateAgent(&agent); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	// Update skill associations
	if len(req.SkillIDs) > 0 {
		if err := h.chatService.UpdateAgentSkills(agent.ID, req.SkillIDs); err != nil {
			logger.Log.Warnf("Failed to associate skills with agent %d: %v", agent.ID, err)
		}
	}
	// Reload with associations
	agentFull, _ := h.chatService.GetAgent(agent.ID)
	// Record operation log
	recordOperationLog(c, "agent", "create", agent.ID, agent.Name,
		fmt.Sprintf("新建智能体: %s, 模型: %s, 技能: %v", agent.Name, agent.Model, req.SkillIDs))
	if agentFull != nil {
		response.Success(c, agentFull)
	} else {
		response.Success(c, agent)
	}
}

func (h *Handler) UpdateAgent(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	// Parse update request as a map to support partial updates
	var req struct {
		Name            string   `json:"name"`
		Description     string   `json:"description"`
		SystemPrompt    string   `json:"system_prompt"`
		Model           string   `json:"model"`
		Temperature     *float64 `json:"temperature"`       // pointer to distinguish 0 from absent
		MaxTokens       *int     `json:"max_tokens"`        // pointer to distinguish 0 from absent
		IsActive        *bool    `json:"is_active"`         // pointer to distinguish false from absent
		SkillIDs        []uint   `json:"skill_ids"`         // associated skill IDs
		CloudPlatformID *uint    `json:"cloud_platform_id"` // bound cloud platform
		ClearPlatform   bool     `json:"clear_platform"`    // explicitly unbind cloud platform
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}

	// Fetch existing agent first
	agent, err := h.chatService.GetAgent(uint(id))
	if err != nil {
		response.BadRequest(c, "agent not found")
		return
	}

	// Apply partial updates — only overwrite fields that are provided
	if req.Name != "" {
		agent.Name = req.Name
	}
	if req.Description != "" {
		agent.Description = req.Description
	}
	if req.SystemPrompt != "" {
		agent.SystemPrompt = req.SystemPrompt
	}
	if req.Model != "" {
		agent.Model = req.Model
	}
	if req.Temperature != nil {
		agent.Temperature = *req.Temperature
	}
	if req.MaxTokens != nil {
		agent.MaxTokens = *req.MaxTokens
	}
	if req.IsActive != nil {
		agent.IsActive = *req.IsActive
	}
	if req.CloudPlatformID != nil {
		agent.CloudPlatformID = req.CloudPlatformID
	} else if req.ClearPlatform {
		agent.CloudPlatformID = nil
	}

	if err := h.chatService.UpdateAgent(agent); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	// Update skill associations if provided
	if req.SkillIDs != nil {
		if err := h.chatService.UpdateAgentSkills(agent.ID, req.SkillIDs); err != nil {
			logger.Log.Warnf("Failed to update skills for agent %d: %v", agent.ID, err)
		}
	}

	// Reload with full associations
	agentFull, _ := h.chatService.GetAgent(agent.ID)

	// Record operation log
	recordOperationLog(c, "agent", "update", agent.ID, agent.Name,
		fmt.Sprintf("更新智能体: %s", agent.Name))
	if agentFull != nil {
		response.Success(c, agentFull)
	} else {
		response.Success(c, agent)
	}
}

func (h *Handler) DeleteAgent(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	// Fetch agent info before deletion for logging
	agentInfo, _ := h.chatService.GetAgent(uint(id))
	if err := h.chatService.DeleteAgent(uint(id)); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	// Record operation log
	agentName := ""
	if agentInfo != nil {
		agentName = agentInfo.Name
	}
	recordOperationLog(c, "agent", "delete", uint(id), agentName,
		fmt.Sprintf("删除智能体: %s", agentName))
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

// GetAgentSkills returns skills associated with a specific agent
func (h *Handler) GetAgentSkills(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	skills, err := h.chatService.GetSkillsByAgent(uint(id))
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

func (h *Handler) UpdateScheduledTask(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req struct {
		Name     string `json:"name"`
		CronExpr string `json:"cron_expr"`
		TaskType string `json:"task_type"`
		Config   string `json:"config"`
		IsActive *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}
	task, err := h.chatService.GetScheduledTask(uint(id))
	if err != nil {
		response.BadRequest(c, "task not found")
		return
	}
	if req.Name != "" {
		task.Name = req.Name
	}
	if req.CronExpr != "" {
		task.CronExpr = req.CronExpr
	}
	if req.TaskType != "" {
		task.TaskType = req.TaskType
	}
	if req.Config != "" {
		task.Config = req.Config
	}
	if req.IsActive != nil {
		task.IsActive = *req.IsActive
	}
	if err := h.chatService.UpdateScheduledTask(task); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, task)
}

func (h *Handler) DeleteScheduledTask(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := h.chatService.DeleteScheduledTask(uint(id)); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, nil)
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
	// Record operation log
	recordOperationLog(c, "user", "create", user.ID, user.Username,
		fmt.Sprintf("新建用户: %s, 角色: %s", user.Username, user.Role))
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
	// Record operation log
	recordOperationLog(c, "user", "update", user.ID, user.Username,
		fmt.Sprintf("更新用户: %s", user.Username))
	// Clear password hash from response
	user.Password = ""
	response.Success(c, user)
}

func (h *Handler) DeleteUser(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	// Fetch user info before deletion for logging
	var delUser model.User
	repository.DB.Unscoped().Select("username").First(&delUser, id)
	if err := service.DeleteUser(uint(id)); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	// Record operation log
	recordOperationLog(c, "user", "delete", uint(id), delUser.Username,
		fmt.Sprintf("删除用户: %s", delUser.Username))
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

// ==================== Operation Logs ====================

// ListOperationLogs returns operation log entries with optional filtering.
// Supports query params: module, action, page, page_size
func (h *Handler) ListOperationLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	moduleFilter := c.Query("module")
	actionFilter := c.Query("action")

	query := repository.DB.Model(&model.OperationLog{})
	if moduleFilter != "" {
		query = query.Where("module = ?", moduleFilter)
	}
	if actionFilter != "" {
		query = query.Where("action = ?", actionFilter)
	}

	var total int64
	query.Count(&total)

	var logs []model.OperationLog
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"items":     logs,
	})
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

	// Parse error response – support multiple formats:
	//  1. OpenAI style:      {"error": {"message": "...", "type": "...", "code": "..."}}
	//  2. SiliconFlow style: {"code": 30001, "message": "Sorry, your account balance is insufficient"}
	//  3. Plain string:      {"error": "some error message"}
	//  4. Raw body fallback
	var errResp map[string]interface{}
	json.Unmarshal(bodyBytes, &errResp)
	errMsg := ""
	if errResp != nil {
		// Format 1: OpenAI {"error": {"message": "..."}}
		if e, ok := errResp["error"]; ok {
			switch ev := e.(type) {
			case map[string]interface{}:
				if msg, ok := ev["message"].(string); ok && msg != "" {
					errMsg = msg
				}
			case string:
				// Format 3: {"error": "some error message"}
				if ev != "" {
					errMsg = ev
				}
			}
		}
		// Format 2: SiliconFlow / generic {"message": "..."}
		if errMsg == "" {
			if msg, ok := errResp["message"].(string); ok && msg != "" {
				errMsg = msg
			}
		}
	}
	// Fallback: include HTTP status and truncated body
	if errMsg == "" {
		bodyStr := string(bodyBytes)
		if len(bodyStr) > 200 {
			bodyStr = bodyStr[:200] + "..."
		}
		if bodyStr != "" {
			errMsg = fmt.Sprintf("API 返回错误 (HTTP %d): %s", resp.StatusCode, bodyStr)
		} else {
			errMsg = fmt.Sprintf("API 返回错误 (HTTP %d)", resp.StatusCode)
		}
	}

	logger.Log.Warnf("AI provider test failed for %s: HTTP %d - %s", provider.Name, resp.StatusCode, errMsg)
	response.BadRequest(c, errMsg)
}

// ==================== Resource Monitor ====================

// authenticateEasyStack obtains a Keystone token for an EasyStack platform.
// Returns (token, error). Includes project scope for proper API access.
// Uses resolveEasyStackServiceURL to construct the correct Keystone URL
// instead of relying on p.AuthURL which may be stale or empty.
func authenticateEasyStack(client *http.Client, p model.CloudPlatform) (string, error) {
	domain := p.DomainName
	if domain == "" {
		domain = "Default"
	}
	authPayload := map[string]interface{}{
		"auth": map[string]interface{}{
			"identity": map[string]interface{}{
				"methods": []string{"password"},
				"password": map[string]interface{}{
					"user": map[string]interface{}{
						"name":     p.Username,
						"password": p.Password,
						"domain":   map[string]string{"name": domain},
					},
				},
			},
			"scope": map[string]interface{}{
				"project": map[string]interface{}{
					"name":   p.ProjectName,
					"domain": map[string]string{"name": domain},
				},
			},
		},
	}
	authBody, _ := json.Marshal(authPayload)
	keystoneURL := resolveEasyStackServiceURL(p, "keystone") + "/v3/auth/tokens"
	authReq, err := http.NewRequest("POST", keystoneURL, bytes.NewReader(authBody))
	if err != nil {
		return "", fmt.Errorf("create auth request: %w", err)
	}
	authReq.Header.Set("Content-Type", "application/json")
	authResp, err := client.Do(authReq)
	if err != nil {
		return "", fmt.Errorf("auth request failed: %w", err)
	}
	defer authResp.Body.Close()
	if authResp.StatusCode != 200 && authResp.StatusCode != 201 {
		body, _ := io.ReadAll(authResp.Body)
		return "", fmt.Errorf("auth failed (HTTP %d): %s", authResp.StatusCode, string(body))
	}
	token := authResp.Header.Get("X-Subject-Token")
	if token == "" {
		return "", fmt.Errorf("empty X-Subject-Token in response")
	}
	return token, nil
}

// authenticateEasyStackFull is like authenticateEasyStack but also extracts the ProjectID
// from the Keystone token response body (token.project.id).
// This enables GetResourceMonitor to work even when ProjectID is not stored in the DB.
func authenticateEasyStackFull(client *http.Client, p model.CloudPlatform) (token string, projectID string, err error) {
	domain := p.DomainName
	if domain == "" {
		domain = "Default"
	}
	authPayload := map[string]interface{}{
		"auth": map[string]interface{}{
			"identity": map[string]interface{}{
				"methods": []string{"password"},
				"password": map[string]interface{}{
					"user": map[string]interface{}{
						"name":     p.Username,
						"password": p.Password,
						"domain":   map[string]string{"name": domain},
					},
				},
			},
			"scope": map[string]interface{}{
				"project": map[string]interface{}{
					"name":   p.ProjectName,
					"domain": map[string]string{"name": domain},
				},
			},
		},
	}
	authBody, _ := json.Marshal(authPayload)
	keystoneURL := resolveEasyStackServiceURL(p, "keystone") + "/v3/auth/tokens"
	logger.Log.Infof("[authenticateEasyStackFull] POST %s (user=%s, project=%s)", keystoneURL, p.Username, p.ProjectName)
	authReq, err := http.NewRequest("POST", keystoneURL, bytes.NewReader(authBody))
	if err != nil {
		return "", "", fmt.Errorf("create auth request: %w", err)
	}
	authReq.Header.Set("Content-Type", "application/json")
	authResp, err := client.Do(authReq)
	if err != nil {
		return "", "", fmt.Errorf("auth request failed: %w", err)
	}
	defer authResp.Body.Close()
	respBody, _ := io.ReadAll(authResp.Body)
	if authResp.StatusCode != 200 && authResp.StatusCode != 201 {
		return "", "", fmt.Errorf("auth failed (HTTP %d): %s", authResp.StatusCode, string(respBody[:min(len(respBody), 500)]))
	}
	token = authResp.Header.Get("X-Subject-Token")
	if token == "" {
		return "", "", fmt.Errorf("empty X-Subject-Token in response")
	}
	// Extract project ID from the token response body
	var tokenResp struct {
		Token struct {
			Project struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"project"`
		} `json:"token"`
	}
	if err := json.Unmarshal(respBody, &tokenResp); err == nil && tokenResp.Token.Project.ID != "" {
		projectID = tokenResp.Token.Project.ID
		logger.Log.Infof("[authenticateEasyStackFull] Extracted ProjectID=%s (name=%s) from token response", projectID, tokenResp.Token.Project.Name)
	}
	return token, projectID, nil
}

// authenticateZStack obtains a session token for a ZStack platform.
// Returns (sessionId, error).
// Supports both AccessKey HMAC signing (preferred) and plain password login.
func authenticateZStack(client *http.Client, p model.CloudPlatform) (string, error) {
	var loginPayload map[string]interface{}
	var loginURL string
	var method string

	if p.AccessKeyID != "" && p.AccessKeySecret != "" {
		// Use AccessKey HMAC authentication (consistent with SkillExecutor)
		endpoint := strings.TrimRight(p.Endpoint, "/")
		loginURL = endpoint + "/v1/accounts/login"
		method = "POST"

		ts := time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
		stringToSign := fmt.Sprintf("POST\n\napplication/json\n%s\n/v1/accounts/login", ts)
		mac := hmacSHA256([]byte(p.AccessKeySecret), []byte(stringToSign))
		sig := hex.EncodeToString(mac)

		loginPayload = map[string]interface{}{
			"loginByAccessKey": map[string]string{
				"accessKeyId":     p.AccessKeyID,
				"accessKeySecret": sig,
			},
		}
		loginBody, _ := json.Marshal(loginPayload)
		req, err := http.NewRequest(method, loginURL, bytes.NewReader(loginBody))
		if err != nil {
			return "", fmt.Errorf("create login request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Date", ts)
		req.Header.Set("Authorization", fmt.Sprintf("ZStack %s:%s", p.AccessKeyID, sig))
		resp, err := client.Do(req)
		if err != nil {
			return "", fmt.Errorf("login request failed: %w", err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 400 {
			return "", fmt.Errorf("ZStack HMAC login failed (HTTP %d): %s", resp.StatusCode, string(body))
		}

		var loginResp struct {
			Inventory struct {
				UUID string `json:"uuid"`
			} `json:"inventory"`
		}
		if err := json.Unmarshal(body, &loginResp); err != nil {
			return "", fmt.Errorf("parse ZStack login response: %w", err)
		}
		if loginResp.Inventory.UUID == "" {
			return "", fmt.Errorf("empty session UUID from ZStack login")
		}
		return loginResp.Inventory.UUID, nil
	}

	// Fallback: plain password login
	if p.Username == "" || p.Password == "" {
		return "", fmt.Errorf("ZStack platform missing credentials")
	}

	loginURL = strings.TrimRight(p.Endpoint, "/") + "/zstack/v1/accounts/login"
	loginPayload = map[string]interface{}{
		"logInByAccount": map[string]string{
			"accountName": p.Username,
			"password":    p.Password,
		},
	}
	loginBody, _ := json.Marshal(loginPayload)
	req, err := http.NewRequest("PUT", loginURL, bytes.NewReader(loginBody))
	if err != nil {
		return "", fmt.Errorf("create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("ZStack login failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Parse session UUID from response: {"inventory":{"uuid":"..."}}
	var loginResp struct {
		Inventory struct {
			UUID string `json:"uuid"`
		} `json:"inventory"`
	}
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return "", fmt.Errorf("parse ZStack login response: %w", err)
	}
	if loginResp.Inventory.UUID == "" {
		return "", fmt.Errorf("empty session UUID from ZStack login")
	}
	return loginResp.Inventory.UUID, nil
}

// hmacSHA256 computes HMAC-SHA256 of the message using the given key.
func hmacSHA256(key, message []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(message)
	return h.Sum(nil)
}

// resolveEasyStackServiceURL builds the correct service URL for a given OpenStack component.
// Multi-domain mode: http://<component>.<BaseDomain> (e.g., http://nova.opsl2.svc.cluster.local)
// Legacy mode: all services share the AuthURL.
func resolveEasyStackServiceURL(p model.CloudPlatform, component string) string {
	if p.HostIP != "" && p.BaseDomain != "" {
		bd := strings.TrimLeft(p.BaseDomain, ".")
		return fmt.Sprintf("http://%s.%s", component, bd)
	}
	return strings.TrimRight(p.AuthURL, "/")
}

// fetchEasyStackServers fetches the total VM count from a connected EasyStack platform.
// Uses Nova API: GET /v2.1/{project_id}/servers/detail?all_tenants=1
func fetchEasyStackServers(client *http.Client, p model.CloudPlatform, token string) int {
	if p.ProjectID == "" {
		logger.Log.Warnf("EasyStack: fetchEasyStackServers skipped for %s: ProjectID is empty", p.Name)
		return 0
	}
	novaURL := resolveEasyStackServiceURL(p, "nova")
	serversURL := fmt.Sprintf("%s/v2.1/%s/servers/detail?all_tenants=1", novaURL, p.ProjectID)
	req, err := http.NewRequest("GET", serversURL, nil)
	if err != nil {
		logger.Log.Warnf("EasyStack: create servers request failed: %v", err)
		return 0
	}
	req.Header.Set("X-Auth-Token", token)
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Warnf("EasyStack: fetch servers failed: %v", err)
		return 0
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logger.Log.Warnf("EasyStack: fetch servers HTTP %d: %s", resp.StatusCode, string(body[:min(len(body), 200)]))
		return 0
	}
	var serversResp struct {
		Servers []json.RawMessage `json:"servers"`
	}
	if err := json.Unmarshal(body, &serversResp); err != nil {
		logger.Log.Warnf("EasyStack: parse servers response failed: %v", err)
		return 0
	}
	logger.Log.Infof("EasyStack: fetched %d servers from %s", len(serversResp.Servers), novaURL)
	return len(serversResp.Servers)
}

// fetchEasyStackVolumes fetches the total volume count from a connected EasyStack platform.
// Uses Cinder API: GET /v3/{project_id}/volumes/detail?all_tenants=1
func fetchEasyStackVolumes(client *http.Client, p model.CloudPlatform, token string) int {
	if p.ProjectID == "" {
		logger.Log.Warnf("EasyStack: fetchEasyStackVolumes skipped for %s: ProjectID is empty", p.Name)
		return 0
	}
	cinderURL := resolveEasyStackServiceURL(p, "cinder")
	volumesURL := fmt.Sprintf("%s/v3/%s/volumes/detail?all_tenants=1", cinderURL, p.ProjectID)
	req, err := http.NewRequest("GET", volumesURL, nil)
	if err != nil {
		logger.Log.Warnf("EasyStack: create volumes request failed: %v", err)
		return 0
	}
	req.Header.Set("X-Auth-Token", token)
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Warnf("EasyStack: fetch volumes failed: %v", err)
		return 0
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logger.Log.Warnf("EasyStack: fetch volumes HTTP %d: %s", resp.StatusCode, string(body[:min(len(body), 200)]))
		return 0
	}
	var volumesResp struct {
		Volumes []json.RawMessage `json:"volumes"`
	}
	if err := json.Unmarshal(body, &volumesResp); err != nil {
		logger.Log.Warnf("EasyStack: parse volumes response failed: %v", err)
		return 0
	}
	logger.Log.Infof("EasyStack: fetched %d volumes from %s", len(volumesResp.Volumes), cinderURL)
	return len(volumesResp.Volumes)
}

// AlertItem is the unified alert structure returned by the resource monitor API.
type AlertItem struct {
	Name      string `json:"name"`
	Severity  string `json:"severity"`
	State     string `json:"state"`
	Platform  string `json:"platform"`
	Timestamp string `json:"timestamp"`
	Target    string `json:"target"`
}

// fetchEasyStackAlerts fetches alerts from the EasyStack ECMS alert API.
// Uses the legacy ECMS API (section 15.1.4.3) which is confirmed working:
//   - GET /apis/monitoring/v1/ecms/alerts                    → returns unresolved (firing) alerts
//   - GET /apis/monitoring/v1/ecms/alerts?status=resolved    → returns resolved alerts
//   - GET /apis/monitoring/v1/ecms/alerts?status=unresolved  → returns unresolved alerts (explicit)
//
// Response format (legacy):
//
//	{
//	  "alerts_status": "unresolved",
//	  "total": 10,
//	  "level_info": { "critical": 7, "warning": 3, "info": 0 },
//	  "type_info":  { "service": 0, "storage": 0, "log": 0, "host": 0, "others": 10 },
//	  "alerts_meta": {
//	    "results": [
//	      {
//	        "startsAt": "...", "endsAt": "...", "status": "firing",
//	        "labels": { "alertname": "...", "severity": "critical", "host_ip": "...", "instance": "...", "node_name": "...", "category": "..." },
//	        "annotations": { "summaryCN": "...", "descriptionCN": "..." }
//	      }
//	    ]
//	  }
//	}
func fetchEasyStackAlerts(client *http.Client, p model.CloudPlatform, token, platformName string) (firing, resolved int, alerts []AlertItem) {
	emlaURL := resolveEasyStackServiceURL(p, "emla")
	baseAlertsURL := fmt.Sprintf("%s/apis/monitoring/v1/ecms/alerts", emlaURL)
	logger.Log.Infof("[ResourceMonitor] fetchEasyStackAlerts: emlaURL=%s, ProjectID=%s", emlaURL, p.ProjectID)

	// Helper: fetch alerts from a URL and parse the legacy ECMS response
	fetchAlerts := func(url, queryState string) (int, []AlertItem) {
		logger.Log.Infof("[ResourceMonitor] Fetching %s alerts from: %s", queryState, url)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			logger.Log.Warnf("EasyStack: create alerts request failed (%s): %v", queryState, err)
			return 0, nil
		}
		req.Header.Set("X-Auth-Token", token)
		req.Header.Set("Accept", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			logger.Log.Warnf("EasyStack: fetch %s alerts from %s failed: %v", queryState, url, err)
			return 0, nil
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 400 {
			logger.Log.Warnf("EasyStack: fetch %s alerts HTTP %d: %s", queryState, resp.StatusCode, string(body[:min(len(body), 500)]))
			return 0, nil
		}
		logger.Log.Infof("EasyStack: fetch %s alerts → HTTP %d (%d bytes), body preview: %s", queryState, resp.StatusCode, len(body), string(body[:min(len(body), 300)]))

		// Parse legacy ECMS response:
		// { "alerts_status": "...", "total": N, "level_info": {...}, "alerts_meta": { "results": [...] } }
		var alertsResp struct {
			AlertsStatus string `json:"alerts_status"`
			Total        int    `json:"total"`
			LevelInfo    struct {
				Critical int `json:"critical"`
				Warning  int `json:"warning"`
				Info     int `json:"info"`
			} `json:"level_info"`
			AlertsMeta struct {
				Results []struct {
					StartsAt    string            `json:"startsAt"`
					EndsAt      string            `json:"endsAt"`
					Status      string            `json:"status"`
					Labels      map[string]string `json:"labels"`
					Annotations struct {
						SummaryCN     string `json:"summaryCN"`
						SummaryEN     string `json:"summaryEN"`
						DescriptionCN string `json:"descriptionCN"`
						DescriptionEN string `json:"descriptionEN"`
					} `json:"annotations"`
				} `json:"results"`
			} `json:"alerts_meta"`
		}
		if err := json.Unmarshal(body, &alertsResp); err != nil {
			logger.Log.Warnf("EasyStack: parse %s alerts response failed: %v, raw body: %s", queryState, err, string(body[:min(len(body), 500)]))
			return 0, nil
		}
		logger.Log.Infof("EasyStack: %s alerts parsed OK — alerts_status=%s, total=%d, results count=%d",
			queryState, alertsResp.AlertsStatus, alertsResp.Total, len(alertsResp.AlertsMeta.Results))

		var items []AlertItem
		for _, r := range alertsResp.AlertsMeta.Results {
			// Alert name from labels.alertname
			name := r.Labels["alertname"]
			if name == "" {
				// Fallback to annotation summary
				name = r.Annotations.SummaryCN
				if name == "" {
					name = r.Annotations.DescriptionCN
				}
			}

			// Severity from labels.severity
			severity := r.Labels["severity"]

			// Determine state from the item's own status field
			state := r.Status
			if state == "" {
				state = queryState
			}

			// Timestamp
			ts := r.StartsAt
			if ts == "" {
				ts = r.EndsAt
			}

			// Build alert target: prefer node_name > host_ip > instance > labels.node
			target := ""
			if v := r.Labels["node_name"]; v != "" {
				target = v
			} else if v := r.Labels["host_ip"]; v != "" {
				target = v
			} else if v := r.Labels["instance"]; v != "" {
				target = v
			} else if v := r.Labels["node"]; v != "" {
				target = v
			}

			items = append(items, AlertItem{
				Name:      name,
				Severity:  severity,
				State:     state,
				Platform:  platformName,
				Timestamp: ts,
				Target:    target,
			})
		}
		count := alertsResp.Total
		if count == 0 {
			count = len(items)
		}
		return count, items
	}

	// 1. Fetch firing (unresolved) alerts — default status is "unresolved"
	firing, firingAlerts := fetchAlerts(baseAlertsURL, "firing")
	logger.Log.Infof("[ResourceMonitor] Platform %s: firing alerts=%d", platformName, firing)

	// 2. Fetch resolved alerts
	resolved, resolvedAlerts := fetchAlerts(baseAlertsURL+"?status=resolved", "resolved")
	logger.Log.Infof("[ResourceMonitor] Platform %s: resolved alerts=%d", platformName, resolved)

	alerts = append(firingAlerts, resolvedAlerts...)
	return firing, resolved, alerts
}

// fetchZStackVMs fetches the total VM count from a connected ZStack platform.
// Uses GET /zstack/v1/vm-instances (QueryVmInstance API).
func fetchZStackVMs(client *http.Client, endpoint, sessionID string) int {
	apiURL := strings.TrimRight(endpoint, "/") + "/zstack/v1/vm-instances"
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		logger.Log.Warnf("ZStack: create VM request failed: %v", err)
		return 0
	}
	req.Header.Set("Authorization", "OAuth "+sessionID)
	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Warnf("ZStack: fetch VMs failed: %v", err)
		return 0
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	// ZStack response: { "inventories": [...] } or { "total": N }
	var vmResp struct {
		Inventories []json.RawMessage `json:"inventories"`
		Total       int               `json:"total"`
	}
	if err := json.Unmarshal(body, &vmResp); err != nil {
		logger.Log.Warnf("ZStack: parse VM response failed: %v", err)
		return 0
	}
	if vmResp.Total > 0 {
		return vmResp.Total
	}
	return len(vmResp.Inventories)
}

// fetchZStackVolumes fetches the total volume count from a connected ZStack platform.
// Uses GET /zstack/v1/volumes (QueryVolume API).
func fetchZStackVolumes(client *http.Client, endpoint, sessionID string) int {
	apiURL := strings.TrimRight(endpoint, "/") + "/zstack/v1/volumes"
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		logger.Log.Warnf("ZStack: create volume request failed: %v", err)
		return 0
	}
	req.Header.Set("Authorization", "OAuth "+sessionID)
	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Warnf("ZStack: fetch volumes failed: %v", err)
		return 0
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var volResp struct {
		Inventories []json.RawMessage `json:"inventories"`
		Total       int               `json:"total"`
	}
	if err := json.Unmarshal(body, &volResp); err != nil {
		logger.Log.Warnf("ZStack: parse volume response failed: %v", err)
		return 0
	}
	if volResp.Total > 0 {
		return volResp.Total
	}
	return len(volResp.Inventories)
}

// fetchZStackAlerts fetches alerts from a connected ZStack platform.
// Uses GET /zstack/v1/alarms (QueryAlarm API).
func fetchZStackAlerts(client *http.Client, endpoint, sessionID, platformName string) (firing, resolved int, alerts []AlertItem) {
	apiURL := strings.TrimRight(endpoint, "/") + "/zstack/v1/alarms"
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		logger.Log.Warnf("ZStack: create alarms request failed: %v", err)
		return 0, 0, nil
	}
	req.Header.Set("Authorization", "OAuth "+sessionID)
	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Warnf("ZStack: fetch alarms failed: %v", err)
		return 0, 0, nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var alarmsResp struct {
		Inventories []struct {
			UUID        string `json:"uuid"`
			Name        string `json:"name"`
			Description string `json:"description"`
			State       string `json:"state"`  // Enabled/Disabled
			Status      string `json:"status"` // Alarm/OK
			Severity    string `json:"severity"`
			CreateDate  string `json:"createDate"`
			LastOpDate  string `json:"lastOpDate"`
		} `json:"inventories"`
	}
	if err := json.Unmarshal(body, &alarmsResp); err != nil {
		logger.Log.Warnf("ZStack: parse alarms response failed: %v", err)
		return 0, 0, nil
	}

	for _, alarm := range alarmsResp.Inventories {
		state := "resolved"
		if alarm.Status == "Alarm" || (alarm.State == "Enabled" && alarm.Status != "OK") {
			state = "firing"
			firing++
		} else {
			resolved++
		}
		severity := alarm.Severity
		if severity == "" {
			severity = "warning"
		}
		ts := alarm.LastOpDate
		if ts == "" {
			ts = alarm.CreateDate
		}
		name := alarm.Name
		if name == "" {
			name = alarm.Description
		}
		alerts = append(alerts, AlertItem{
			Name:      name,
			Severity:  strings.ToLower(severity),
			State:     state,
			Platform:  platformName,
			Timestamp: ts,
		})
	}
	return firing, resolved, alerts
}

// GetResourceMonitor returns resource monitoring data for the big-screen dashboard.
// It aggregates: cloud platform count, VM/volume counts per platform (via real API calls),
// alerting/resolved alert counts (via real API calls), and component health status.
//
// For EasyStack platforms: uses Keystone auth → Nova/Cinder/Observable APIs
// For ZStack platforms:    uses ZStack login  → VM/Volume/Alarm query APIs
func (h *Handler) GetResourceMonitor(c *gin.Context) {
	// 1. Cloud platform list + count
	var platforms []model.CloudPlatform
	if err := repository.DB.Find(&platforms).Error; err != nil {
		response.InternalError(c, err.Error())
		return
	}

	platformCount := len(platforms)

	// 2. For every *connected* platform, fetch real data via EasyStack/ZStack APIs.
	type PlatformResource struct {
		ID          uint   `json:"id"`
		Name        string `json:"name"`
		Type        string `json:"type"`
		Status      string `json:"status"`
		VMCount     int    `json:"vm_count"`
		VolumeCount int    `json:"volume_count"`
	}
	platformResources := make([]PlatformResource, 0, platformCount)

	totalVMs := 0
	totalVolumes := 0
	firingAlerts := 0
	resolvedAlerts := 0
	var alertList []AlertItem

	// Shared HTTP client with reasonable timeout for all platform API calls
	// Note: For multi-domain EasyStack platforms, a custom DNS client is created per platform.
	defaultClient := &http.Client{Timeout: 15 * time.Second}

	// Process platforms concurrently for better performance
	type platformResult struct {
		Resource PlatformResource
		Firing   int
		Resolved int
		Alerts   []AlertItem
	}
	results := make([]platformResult, len(platforms))
	var wg sync.WaitGroup

	for i, p := range platforms {
		wg.Add(1)
		go func(idx int, plat model.CloudPlatform) {
			defer wg.Done()

			pr := PlatformResource{
				ID:     plat.ID,
				Name:   plat.Name,
				Type:   plat.Type,
				Status: plat.Status,
			}
			var firing, resolved int
			var alerts []AlertItem

			switch {
			// ── EasyStack platform ──
			// Accept both multi-domain (HostIP+BaseDomain) and legacy (AuthURL) modes
			case plat.Status == "connected" && plat.Type == "easystack" &&
				(plat.AuthURL != "" || (plat.HostIP != "" && plat.BaseDomain != "")) &&
				plat.Username != "" && plat.Password != "":

				logger.Log.Infof("[ResourceMonitor] Processing EasyStack platform %q (ID=%d, HostIP=%s, BaseDomain=%s, AuthURL=%s, ProjectID=%s)",
					plat.Name, plat.ID, plat.HostIP, plat.BaseDomain, plat.AuthURL, plat.ProjectID)

				// Use custom DNS client for multi-domain mode (resolves *.BaseDomain → HostIP)
				apiClient := agent.NewHTTPClientWithCustomDNS(plat.HostIP, plat.BaseDomain, 15*time.Second)

				token, projectID, err := authenticateEasyStackFull(apiClient, plat)
				if err != nil {
					logger.Log.Warnf("[ResourceMonitor] EasyStack auth failed for %s: %v", plat.Name, err)
					break
				}
				logger.Log.Infof("[ResourceMonitor] EasyStack auth OK for %s, token=%s..., projectID=%s", plat.Name, token[:min(len(token), 12)], projectID)

				// If ProjectID was not stored in DB, use the one from the auth response
				if plat.ProjectID == "" && projectID != "" {
					logger.Log.Infof("[ResourceMonitor] Platform %s has empty ProjectID in DB, using auth-derived: %s", plat.Name, projectID)
					plat.ProjectID = projectID
				}

				// Fetch VMs (Nova API: GET /v2.1/{project_id}/servers/detail?all_tenants=1)
				pr.VMCount = fetchEasyStackServers(apiClient, plat, token)
				logger.Log.Infof("[ResourceMonitor] Platform %s: VMs=%d", plat.Name, pr.VMCount)

				// Fetch Volumes (Cinder API: GET /v3/{project_id}/volumes/detail?all_tenants=1)
				pr.VolumeCount = fetchEasyStackVolumes(apiClient, plat, token)
				logger.Log.Infof("[ResourceMonitor] Platform %s: Volumes=%d", plat.Name, pr.VolumeCount)

				// Fetch Alerts (ECMS API: GET /apis/monitoring/v1/ecms/alerts)
				firing, resolved, alerts = fetchEasyStackAlerts(apiClient, plat, token, plat.Name)

			// ── ZStack platform ──
			case plat.Status == "connected" && plat.Type == "zstack" && plat.Endpoint != "":

				sessionID, err := authenticateZStack(defaultClient, plat)
				if err != nil {
					logger.Log.Warnf("ZStack auth failed for %s: %v", plat.Name, err)
					break
				}

				endpoint := strings.TrimRight(plat.Endpoint, "/")

				// Fetch VMs (QueryVmInstance API)
				pr.VMCount = fetchZStackVMs(defaultClient, endpoint, sessionID)

				// Fetch Volumes (QueryVolume API)
				pr.VolumeCount = fetchZStackVolumes(defaultClient, endpoint, sessionID)

				// Fetch Alerts (QueryAlarm API)
				firing, resolved, alerts = fetchZStackAlerts(defaultClient, endpoint, sessionID, plat.Name)

			default:
				// Platform skipped – log why for debugging
				logger.Log.Warnf("[ResourceMonitor] Skipping platform %q (ID=%d): type=%s, status=%s, AuthURL=%q, HostIP=%q, BaseDomain=%q, Username=%q, HasPassword=%v, Endpoint=%q",
					plat.Name, plat.ID, plat.Type, plat.Status, plat.AuthURL, plat.HostIP, plat.BaseDomain, plat.Username, plat.Password != "", plat.Endpoint)
			}

			results[idx] = platformResult{
				Resource: pr,
				Firing:   firing,
				Resolved: resolved,
				Alerts:   alerts,
			}
		}(i, p)
	}

	wg.Wait()

	// Aggregate results from all platforms
	for _, res := range results {
		totalVMs += res.Resource.VMCount
		totalVolumes += res.Resource.VolumeCount
		firingAlerts += res.Firing
		resolvedAlerts += res.Resolved
		alertList = append(alertList, res.Alerts...)
		platformResources = append(platformResources, res.Resource)
	}

	// 3. Component health – derive from platform connectivity + internal services
	type ComponentHealth struct {
		Name   string `json:"name"`
		Status string `json:"status"` // healthy / degraded / down
		Detail string `json:"detail"`
	}
	components := []ComponentHealth{
		{Name: "认证服务 (Keystone)", Status: "healthy", Detail: "身份认证服务正常"},
		{Name: "计算服务 (Nova)", Status: "healthy", Detail: "虚拟机管理正常"},
		{Name: "存储服务 (Cinder)", Status: "healthy", Detail: "块存储服务正常"},
		{Name: "网络服务 (Neutron)", Status: "healthy", Detail: "网络管理正常"},
		{Name: "负载均衡 (Octavia)", Status: "healthy", Detail: "LB 服务正常"},
		{Name: "监控服务 (ECMS)", Status: "healthy", Detail: "指标采集正常"},
	}

	// Check if any platform is in failed state → degrade components
	hasFailed := false
	for _, p := range platforms {
		if p.Status == "failed" {
			hasFailed = true
			break
		}
	}
	if hasFailed {
		for i := range components {
			components[i].Status = "degraded"
			components[i].Detail += " (部分平台连接失败)"
		}
	}
	if platformCount == 0 {
		for i := range components {
			components[i].Status = "unknown"
			components[i].Detail = "暂无接入云平台"
		}
	}

	// 4. AI service and agent statistics (cross-module)
	var aiProviderCount int64
	repository.DB.Model(&model.AIProvider{}).Where("api_key != '' AND is_enabled = ?", true).Count(&aiProviderCount)

	var agentCount int64
	repository.DB.Model(&model.Agent{}).Where("is_active = ?", true).Count(&agentCount)

	response.Success(c, gin.H{
		"cloud_platforms":    platformCount,
		"total_vms":          totalVMs,
		"total_volumes":      totalVolumes,
		"firing_alerts":      firingAlerts,
		"resolved_alerts":    resolvedAlerts,
		"alerts":             alertList,
		"platform_resources": platformResources,
		"components":         components,
		"ai_providers":       aiProviderCount,
		"agents":             agentCount,
	})
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
		HostIP          string `json:"host_ip"`
		BaseDomain      string `json:"base_domain"`
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
	// For EasyStack: auto-generate AuthURL from HostIP + BaseDomain if provided
	authURL := req.AuthURL
	if req.Type == "easystack" && req.HostIP != "" && req.BaseDomain != "" {
		authURL = fmt.Sprintf("http://keystone.%s", strings.TrimLeft(req.BaseDomain, "."))
	}
	platform := model.CloudPlatform{
		Name:            req.Name,
		Type:            req.Type,
		AuthURL:         authURL,
		Username:        req.Username,
		Password:        req.Password,
		DomainName:      req.DomainName,
		ProjectName:     req.ProjectName,
		ProjectID:       req.ProjectID,
		HostIP:          req.HostIP,
		BaseDomain:      req.BaseDomain,
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
	// Record operation log
	recordOperationLog(c, "cloud_platform", "create", platform.ID, platform.Name,
		fmt.Sprintf("接入云平台: %s, 类型: %s", platform.Name, platform.Type))
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
		HostIP          string `json:"host_ip"`
		BaseDomain      string `json:"base_domain"`
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
	if req.HostIP != "" {
		platform.HostIP = req.HostIP
	}
	if req.BaseDomain != "" {
		platform.BaseDomain = req.BaseDomain
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
	// Re-generate AuthURL from HostIP + BaseDomain if both are provided
	if platform.Type == "easystack" && platform.HostIP != "" && platform.BaseDomain != "" {
		platform.AuthURL = fmt.Sprintf("http://keystone.%s", strings.TrimLeft(platform.BaseDomain, "."))
	}
	if err := repository.DB.Save(&platform).Error; err != nil {
		response.InternalError(c, err.Error())
		return
	}
	// Record operation log
	recordOperationLog(c, "cloud_platform", "update", platform.ID, platform.Name,
		fmt.Sprintf("更新云平台: %s", platform.Name))
	response.Success(c, platform)
}

// DeleteCloudPlatform soft-deletes a cloud platform
func (h *Handler) DeleteCloudPlatform(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	// Fetch platform info before deletion for logging
	var delPlatform model.CloudPlatform
	repository.DB.Unscoped().Select("name", "type").First(&delPlatform, id)
	if err := repository.DB.Delete(&model.CloudPlatform{}, id).Error; err != nil {
		response.InternalError(c, err.Error())
		return
	}
	// Record operation log
	recordOperationLog(c, "cloud_platform", "delete", uint(id), delPlatform.Name,
		fmt.Sprintf("删除云平台: %s (%s)", delPlatform.Name, delPlatform.Type))
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

	// Use custom DNS client for multi-domain mode; falls back to plain client otherwise
	client := agent.NewHTTPClientWithCustomDNS(platform.HostIP, platform.BaseDomain, 15*time.Second)
	var testErr error

	switch platform.Type {
	case "easystack":
		// EasyStack: call Keystone v3 token endpoint with project scope
		// Use resolveEasyStackServiceURL for correct multi-domain / legacy URL resolution
		keystoneBase := resolveEasyStackServiceURL(platform, "keystone")
		if keystoneBase == "" || (platform.Username == "" || platform.Password == "") {
			response.BadRequest(c, "EasyStack 平台缺少认证地址(IP+根域名 或 AuthURL)、Username 或 Password")
			return
		}
		keystoneURL := keystoneBase + "/v3/auth/tokens"
		domain := platform.DomainName
		if domain == "" {
			domain = "Default"
		}
		projectName := platform.ProjectName
		if projectName == "" {
			projectName = "admin"
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
				"scope": map[string]interface{}{
					"project": map[string]interface{}{
						"name":   projectName,
						"domain": map[string]string{"name": domain},
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
		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == 201 || resp.StatusCode == 200 {
			// Extract ProjectID from token response and auto-fill into DB
			updates := map[string]interface{}{"status": "connected"}
			var tokenResp struct {
				Token struct {
					Project struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"project"`
				} `json:"token"`
			}
			if err := json.Unmarshal(respBody, &tokenResp); err == nil && tokenResp.Token.Project.ID != "" {
				updates["project_id"] = tokenResp.Token.Project.ID
				logger.Log.Infof("[TestCloudPlatform] EasyStack %s: auto-filled ProjectID=%s (project=%s)",
					platform.Name, tokenResp.Token.Project.ID, tokenResp.Token.Project.Name)
			}
			// Also auto-generate AuthURL if missing but HostIP+BaseDomain are set
			if platform.AuthURL == "" && platform.HostIP != "" && platform.BaseDomain != "" {
				updates["auth_url"] = fmt.Sprintf("http://keystone.%s", strings.TrimLeft(platform.BaseDomain, "."))
			}
			repository.DB.Model(&platform).Updates(updates)
			msg := "EasyStack Keystone 认证成功"
			if pid, ok := updates["project_id"]; ok {
				msg += fmt.Sprintf("，已自动获取 Project ID: %s", pid)
			}
			response.Success(c, gin.H{"status": "connected", "message": msg})
			return
		}
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
