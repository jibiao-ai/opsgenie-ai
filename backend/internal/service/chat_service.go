package service

import (
	"encoding/json"

	agentpkg "github.com/jibiao-ai/cloud-agent/internal/agent"
	"github.com/jibiao-ai/cloud-agent/internal/model"
	"github.com/jibiao-ai/cloud-agent/internal/repository"
	"github.com/jibiao-ai/cloud-agent/pkg/logger"
)

type ChatService struct {
	agent *agentpkg.Agent
}

func NewChatService(agent *agentpkg.Agent) *ChatService {
	return &ChatService{agent: agent}
}

// CreateConversation creates a new conversation
func (s *ChatService) CreateConversation(userID, agentID uint, title string) (*model.Conversation, error) {
	conv := &model.Conversation{
		UserID:  userID,
		AgentID: agentID,
		Title:   title,
	}
	if err := repository.DB.Create(conv).Error; err != nil {
		return nil, err
	}
	repository.DB.Preload("Agent").First(conv, conv.ID)
	return conv, nil
}

// GetConversations returns all conversations for a user
func (s *ChatService) GetConversations(userID uint) ([]model.Conversation, error) {
	var convs []model.Conversation
	err := repository.DB.Where("user_id = ?", userID).
		Preload("Agent").
		Order("updated_at DESC").
		Find(&convs).Error
	return convs, err
}

// GetConversation returns a single conversation
func (s *ChatService) GetConversation(id, userID uint) (*model.Conversation, error) {
	var conv model.Conversation
	err := repository.DB.Where("id = ? AND user_id = ?", id, userID).
		Preload("Agent").
		First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

// DeleteConversation deletes a conversation
func (s *ChatService) DeleteConversation(id, userID uint) error {
	// Delete messages first
	repository.DB.Where("conversation_id = ?", id).Delete(&model.Message{})
	return repository.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Conversation{}).Error
}

// GetMessages returns all messages in a conversation
func (s *ChatService) GetMessages(conversationID, userID uint) ([]model.Message, error) {
	// Verify ownership
	var conv model.Conversation
	if err := repository.DB.Where("id = ? AND user_id = ?", conversationID, userID).First(&conv).Error; err != nil {
		return nil, err
	}

	var msgs []model.Message
	err := repository.DB.Where("conversation_id = ?", conversationID).
		Order("created_at ASC").
		Find(&msgs).Error
	return msgs, err
}

// SendMessage processes a user message through the agent
func (s *ChatService) SendMessage(conversationID, userID uint, content string, callback agentpkg.StreamCallback) (*model.Message, *model.Message, error) {
	// Verify ownership and get conversation with agent
	var conv model.Conversation
	if err := repository.DB.Where("id = ? AND user_id = ?", conversationID, userID).
		Preload("Agent").
		First(&conv).Error; err != nil {
		return nil, nil, err
	}

	// Save user message
	userMsg := &model.Message{
		ConversationID: conversationID,
		Role:           "user",
		Content:        content,
	}
	if err := repository.DB.Create(userMsg).Error; err != nil {
		return nil, nil, err
	}

	// Get conversation history
	var historyMsgs []model.Message
	repository.DB.Where("conversation_id = ? AND id < ?", conversationID, userMsg.ID).
		Order("created_at ASC").
		Limit(20). // Keep last 20 messages for context
		Find(&historyMsgs)

	// Convert to agent chat messages
	var history []agentpkg.ChatMessage
	for _, m := range historyMsgs {
		history = append(history, agentpkg.ChatMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	// Call agent
	response, err := s.agent.Chat(conv.Agent, history, content, callback)
	if err != nil {
		logger.Log.Errorf("Agent chat failed: %v", err)
		response = "抱歉，处理您的请求时出现了错误。请稍后再试。错误信息：" + err.Error()
	}

	// Save assistant message
	assistantMsg := &model.Message{
		ConversationID: conversationID,
		Role:           "assistant",
		Content:        response,
	}
	if err := repository.DB.Create(assistantMsg).Error; err != nil {
		return nil, nil, err
	}

	// Update conversation title if first message
	if conv.Title == "" || conv.Title == "新会话" {
		truncated := content
		if len(truncated) > 50 {
			truncated = truncated[:50] + "..."
		}
		repository.DB.Model(&conv).Update("title", truncated)
	}

	return userMsg, assistantMsg, nil
}

// GetAgents returns all available agents
func (s *ChatService) GetAgents() ([]model.Agent, error) {
	var agents []model.Agent
	err := repository.DB.Where("is_active = ?", true).Find(&agents).Error
	return agents, err
}

// GetAgent returns a single agent
func (s *ChatService) GetAgent(id uint) (*model.Agent, error) {
	var agent model.Agent
	err := repository.DB.First(&agent, id).Error
	return &agent, err
}

// CreateAgent creates a new agent
func (s *ChatService) CreateAgent(agent *model.Agent) error {
	return repository.DB.Create(agent).Error
}

// UpdateAgent updates an agent
func (s *ChatService) UpdateAgent(agent *model.Agent) error {
	return repository.DB.Save(agent).Error
}

// DeleteAgent soft-deletes an agent
func (s *ChatService) DeleteAgent(id uint) error {
	return repository.DB.Delete(&model.Agent{}, id).Error
}

// GetSkills returns all skills
func (s *ChatService) GetSkills() ([]model.Skill, error) {
	var skills []model.Skill
	err := repository.DB.Find(&skills).Error
	return skills, err
}

// GetWorkflows returns all workflows
func (s *ChatService) GetWorkflows() ([]model.Workflow, error) {
	var workflows []model.Workflow
	err := repository.DB.Find(&workflows).Error
	return workflows, err
}

// CreateWorkflow creates a new workflow
func (s *ChatService) CreateWorkflow(wf *model.Workflow) error {
	return repository.DB.Create(wf).Error
}

// GetScheduledTasks returns all scheduled tasks
func (s *ChatService) GetScheduledTasks() ([]model.ScheduledTask, error) {
	var tasks []model.ScheduledTask
	err := repository.DB.Find(&tasks).Error
	return tasks, err
}

// CreateScheduledTask creates a new scheduled task
func (s *ChatService) CreateScheduledTask(task *model.ScheduledTask) error {
	return repository.DB.Create(task).Error
}

// GetDashboardStats returns dashboard statistics
func (s *ChatService) GetDashboardStats(userID uint) (map[string]interface{}, error) {
	var conversationCount int64
	repository.DB.Model(&model.Conversation{}).Where("user_id = ?", userID).Count(&conversationCount)

	var messageCount int64
	repository.DB.Model(&model.Message{}).
		Joins("JOIN conversations ON messages.conversation_id = conversations.id").
		Where("conversations.user_id = ?", userID).
		Count(&messageCount)

	var agentCount int64
	repository.DB.Model(&model.Agent{}).Where("is_active = ?", true).Count(&agentCount)

	var skillCount int64
	repository.DB.Model(&model.Skill{}).Where("is_active = ?", true).Count(&skillCount)

	var taskCount int64
	repository.DB.Model(&model.TaskLog{}).Where("user_id = ?", userID).Count(&taskCount)

	// Cloud platform count
	var cloudPlatformCount int64
	repository.DB.Model(&model.CloudPlatform{}).Count(&cloudPlatformCount)

	// AI model (provider) count — only those with a configured API key
	var aiModelCount int64
	repository.DB.Model(&model.AIProvider{}).Where("api_key != '' AND is_enabled = ?", true).Count(&aiModelCount)

	var recentTasks []model.TaskLog
	repository.DB.Where("user_id = ?", userID).Order("created_at DESC").Limit(5).Find(&recentTasks)

	// Recent messages
	var recentMsgs []model.Message
	repository.DB.Joins("JOIN conversations ON messages.conversation_id = conversations.id").
		Where("conversations.user_id = ? AND messages.role = ?", userID, "user").
		Order("messages.created_at DESC").
		Limit(5).
		Find(&recentMsgs)

	// Recent conversations (for dashboard display)
	var recentConversations []model.Conversation
	repository.DB.Where("user_id = ?", userID).
		Preload("Agent").
		Order("updated_at DESC").
		Limit(6).
		Find(&recentConversations)

	return map[string]interface{}{
		"conversations":        conversationCount,
		"messages":             messageCount,
		"agents":               agentCount,
		"skills":               skillCount,
		"tasks":                taskCount,
		"cloud_platforms":      cloudPlatformCount,
		"ai_models":           aiModelCount,
		"recent_tasks":         recentTasks,
		"recent_msgs":          recentMsgs,
		"recent_conversations": recentConversations,
	}, nil
}

// GetUsers returns all users (admin only)
func (s *ChatService) GetUsers() ([]model.User, error) {
	var users []model.User
	err := repository.DB.Find(&users).Error
	return users, err
}

// CreateUser creates a new user
func (s *ChatService) CreateUser(user *model.User) error {
	// Hash password only if non-empty and not already a bcrypt hash
	if user.Password != "" && !isBcryptHash(user.Password) {
		hashed, err := HashPassword(user.Password)
		if err != nil {
			return err
		}
		user.Password = hashed
	}
	return repository.DB.Create(user).Error
}

// UpdateUser updates a user
func (s *ChatService) UpdateUser(user *model.User) error {
	updates := map[string]interface{}{
		"email": user.Email,
		"role":  user.Role,
	}
	// Hash password only if non-empty and not already a bcrypt hash
	if user.Password != "" && !isBcryptHash(user.Password) {
		hashed, err := HashPassword(user.Password)
		if err != nil {
			return err
		}
		updates["password"] = hashed
	}
	return repository.DB.Model(user).Updates(updates).Error
}

// DeleteUser deletes a user
func (s *ChatService) DeleteUser(id uint) error {
	return repository.DB.Delete(&model.User{}, id).Error
}

// TaskLog helpers
func (s *ChatService) GetTaskLogs(userID uint) ([]model.TaskLog, error) {
	var logs []model.TaskLog
	err := repository.DB.Where("user_id = ?", userID).Order("created_at DESC").Limit(50).Find(&logs).Error
	return logs, err
}

func (s *ChatService) CreateTaskLog(log *model.TaskLog) error {
	return repository.DB.Create(log).Error
}

func (s *ChatService) UpdateTaskLog(log *model.TaskLog) error {
	return repository.DB.Save(log).Error
}

// SerializeJSON helper
func SerializeJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
