package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jibiao-ai/opsgenie-ai/internal/config"
	"github.com/jibiao-ai/opsgenie-ai/internal/model"
	"github.com/jibiao-ai/opsgenie-ai/internal/repository"
	"github.com/jibiao-ai/opsgenie-ai/pkg/logger"
)

// ToolDefinition defines a function/tool for the AI
type ToolDefinition struct {
	Type     string          `json:"type"`
	Function json.RawMessage `json:"function"`
}

// ChatMessage is an OpenAI-compatible message
type ChatMessage struct {
	Role       string          `json:"role"`
	Content    string          `json:"content,omitempty"`
	ToolCalls  json.RawMessage `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	Name       string          `json:"name,omitempty"`
}

// StreamCallback is called for each streaming chunk
type StreamCallback func(content string, done bool)

// Agent orchestrates AI reasoning with cloud platform tool calls.
// It now uses SkillExecutor for dynamic cloud platform authentication
// and tool execution based on the agent's associated skills.
type Agent struct {
	aiCfg         config.AIConfig
	esClient      interface{} // Legacy EasyStack client (kept for backward compat)
	SkillExecutor *SkillExecutor
}

// NewAgent creates a new Agent with a SkillExecutor.
func NewAgent(aiCfg config.AIConfig, esClient interface{}) *Agent {
	return &Agent{
		aiCfg:         aiCfg,
		esClient:      esClient,
		SkillExecutor: NewSkillExecutor(),
	}
}

// loadAgentSkillsAndPlatform loads the agent's associated skills and cloud platform from DB.
func loadAgentSkillsAndPlatform(agentID uint) ([]model.Skill, *model.CloudPlatform) {
	if repository.DB == nil {
		return nil, nil
	}

	// Load associated skills via the join table
	var agentSkills []model.AgentSkill
	repository.DB.Where("agent_id = ?", agentID).Preload("Skill").Find(&agentSkills)

	var skills []model.Skill
	for _, as := range agentSkills {
		if as.Skill.IsActive {
			skills = append(skills, as.Skill)
		}
	}

	// Load the agent's bound cloud platform
	var agent model.Agent
	if err := repository.DB.First(&agent, agentID).Error; err != nil {
		return skills, nil
	}
	if agent.CloudPlatformID == nil {
		// If no bound platform, try to find any active connected platform
		var platform model.CloudPlatform
		if err := repository.DB.Where("is_active = ? AND status = ?", true, "connected").First(&platform).Error; err == nil {
			return skills, &platform
		}
		// Fall back to any active platform
		if err := repository.DB.Where("is_active = ?", true).First(&platform).Error; err == nil {
			return skills, &platform
		}
		return skills, nil
	}

	var platform model.CloudPlatform
	if err := repository.DB.First(&platform, *agent.CloudPlatformID).Error; err != nil {
		return skills, nil
	}
	return skills, &platform
}

// Chat sends a user message through the AI with dynamic tool calling support.
// Tools are derived from the agent's associated skills, and tool execution
// authenticates to the agent's bound cloud platform for real API calls.
func (a *Agent) Chat(agentModel model.Agent, history []ChatMessage, userMsg string, callback StreamCallback) (string, error) {
	// Load skills and cloud platform for this agent
	skills, platform := loadAgentSkillsAndPlatform(agentModel.ID)

	// Build tool definitions from associated skills
	tools := BuildToolsForSkills(skills)

	// Build system prompt with cloud platform context
	systemPrompt := agentModel.SystemPrompt
	if platform != nil {
		systemPrompt += fmt.Sprintf("\n\n[系统信息] 你当前连接的云平台是 \"%s\" (类型: %s)。"+
			"当用户询问云资源相关问题时，你**必须**调用工具函数获取真实数据来回答。"+
			"\n\n[严格数据规范] "+
			"1. 你**禁止**编造、推测或凭记忆生成任何云平台数据（告警数量、服务状态、资源指标等）。"+
			"2. 所有数据**必须且只能**来源于工具调用的返回结果。"+
			"3. 如果工具返回了N条数据，你的回答中必须严格展示N条，不能多也不能少。"+
			"4. 如果工具调用失败或返回为空，你必须明确告知用户『数据获取失败』，而不是编造数据。"+
			"5. 在输出数据前，请先核对工具返回的原始数据条目数量，确保你的汇总与原始数据一致。",
			platform.Name, platform.Type)
	}
	if len(skills) > 0 {
		skillNames := make([]string, len(skills))
		for i, s := range skills {
			skillNames[i] = s.Name
		}
		systemPrompt += fmt.Sprintf("\n你已关联以下技能: %s", strings.Join(skillNames, "、"))
	}

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
	}
	messages = append(messages, history...)
	messages = append(messages, ChatMessage{Role: "user", Content: userMsg})

	// Resolve AI config: agent-specific > database provider > static config
	baseURL, _, dbModelName := a.getActiveAIConfig()
	modelName := dbModelName
	if modelName == "" {
		modelName = a.aiCfg.Model
	}
	if agentModel.Model != "" {
		modelName = agentModel.Model
	}

	supportsTools := a.providerSupportsTools(baseURL)
	supportsStream := a.providerSupportsStream(baseURL)
	isSiliconFlow := a.isSiliconFlowProvider(baseURL)

	logger.Log.Infof("Chat: agent='%s', model=%s, skills=%d, tools=%d, platform=%v, supportsTools=%v, baseURL=%s",
		agentModel.Name, modelName, len(skills), len(tools), platform != nil, supportsTools, baseURL)

	// IMPORTANT: If the provider does NOT support tool calling but we have tools,
	// warn loudly because the agent will not be able to call real cloud APIs.
	if !supportsTools && len(tools) > 0 {
		logger.Log.Warnf("WARNING: AI provider at '%s' does NOT support tool calling! Agent '%s' has %d tools but they will NOT be sent. LLM may hallucinate data instead of calling real APIs.",
			baseURL, agentModel.Name, len(tools))
	}

	// Loop for tool calling
	for iterations := 0; iterations < 10; iterations++ {
		reqBody := map[string]interface{}{
			"model":    modelName,
			"messages": messages,
		}
		if agentModel.Temperature > 0 {
			reqBody["temperature"] = agentModel.Temperature
		}
		if agentModel.MaxTokens > 0 {
			if isSiliconFlow {
				// SiliconFlow uses "max_new_tokens" instead of "max_tokens"
				reqBody["max_new_tokens"] = agentModel.MaxTokens
			} else {
				reqBody["max_tokens"] = agentModel.MaxTokens
			}
		}
		if supportsStream {
			reqBody["stream"] = callback != nil
		}
		// Only send tools when the provider supports them AND we have tools
		if supportsTools && len(tools) > 0 {
			reqBody["tools"] = tools
		}

		respBody, err := a.callAI(reqBody)
		if err != nil {
			return "", fmt.Errorf("AI call failed: %w", err)
		}

		var resp struct {
			Choices []struct {
				Message struct {
					Role      string `json:"role"`
					Content   string `json:"content"`
					ToolCalls []struct {
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
		}
		if err := json.Unmarshal(respBody, &resp); err != nil {
			return "", fmt.Errorf("parse AI response failed: %w", err)
		}

		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("empty AI response")
		}

		choice := resp.Choices[0]

		// If there are tool calls, execute them via SkillExecutor
		if len(choice.Message.ToolCalls) > 0 {
			toolCallsJSON, _ := json.Marshal(choice.Message.ToolCalls)
			messages = append(messages, ChatMessage{
				Role:      "assistant",
				Content:   choice.Message.Content,
				ToolCalls: toolCallsJSON,
			})

			for _, tc := range choice.Message.ToolCalls {
				logger.Log.Infof("[ToolCall] Executing tool: %s, id: %s, args: %s", tc.Function.Name, tc.ID, tc.Function.Arguments)

				var toolResult string
				if platform != nil {
					// Use SkillExecutor to execute against the real cloud platform
					toolResult, err = a.SkillExecutor.ExecuteTool(*platform, tc.Function.Name, json.RawMessage(tc.Function.Arguments))
					if err != nil {
						toolResult = fmt.Sprintf(`{"error":"tool execution failed: %s"}`, err.Error())
						logger.Log.Errorf("[ToolCall] Tool '%s' execution FAILED: %v", tc.Function.Name, err)
					} else {
						// Log result size and preview for debugging data accuracy
						preview := toolResult
						if len(preview) > 500 {
							preview = preview[:500] + "..."
						}
						logger.Log.Infof("[ToolCall] Tool '%s' SUCCESS, result_size=%d bytes, preview: %s", tc.Function.Name, len(toolResult), preview)
					}
				} else {
					toolResult = `{"error":"当前智能体未绑定云平台，无法执行云资源操作。请在智能体管理中关联一个云平台。"}`
				}

				messages = append(messages, ChatMessage{
					Role:       "tool",
					Content:    toolResult,
					ToolCallID: tc.ID,
					Name:       tc.Function.Name,
				})
			}

			// After all tool calls, inject a strict data-binding reminder
			messages = append(messages, ChatMessage{
				Role: "system",
				Content: "[数据校验指令] 以上工具已返回真实数据。你必须严格遵守：" +
					"1) 数量必须与返回数据完全一致，返回N条就展示N条，不可多不可少；" +
					"2) 所有数值、标签、指标名必须原样引用工具返回的数据，禁止编造；" +
					"3) 如果工具返回了error，如实告知用户该数据获取失败及错误原因；" +
					"4) 禁止使用「根据经验」「通常来说」「一般情况下」等推测性表述；" +
					"5) 回答末尾必须附加可信度评分：「📊 可信度：X/10」。",
			})
			continue
		}

		// No tool calls, return the final response
		if iterations == 0 && len(tools) > 0 {
			logger.Log.Warnf("[Chat] LLM returned final answer on FIRST iteration without calling ANY tools. Agent '%s' has %d tools available. The response may contain hallucinated data.",
				agentModel.Name, len(tools))
		}

		content := choice.Message.Content

		// Handle truncated response: if finish_reason is "length", the AI
		// output was cut short by max_tokens. We auto-continue by asking
		// the LLM to finish its response.
		if choice.FinishReason == "length" && len(content) > 0 {
			logger.Log.Warnf("[Chat] Response truncated (finish_reason=length) at iteration %d, content_length=%d. Auto-continuing.", iterations, len(content))

			// Append what we got so far and ask to continue
			messages = append(messages, ChatMessage{Role: "assistant", Content: content})
			messages = append(messages, ChatMessage{
				Role:    "user",
				Content: "你的回答被截断了，请从截断处继续完成回答。不要重复已经输出的内容，直接从断点处继续。如果剩余内容不多，请尽快给出结论和可信度评分。",
			})

			// Try up to 5 continuations
			var fullContent strings.Builder
			fullContent.WriteString(content)
			for contIdx := 0; contIdx < 5; contIdx++ {
				contReqBody := map[string]interface{}{
					"model":    modelName,
					"messages": messages,
				}
				if agentModel.Temperature > 0 {
					contReqBody["temperature"] = agentModel.Temperature
				}
				if agentModel.MaxTokens > 0 {
					contReqBody["max_tokens"] = agentModel.MaxTokens
				}
				contRespBody, contErr := a.callAI(contReqBody)
				if contErr != nil {
					logger.Log.Warnf("[Chat] Continuation request failed: %v", contErr)
					break
				}
				var contResp struct {
					Choices []struct {
						Message struct {
							Content string `json:"content"`
						} `json:"message"`
						FinishReason string `json:"finish_reason"`
					} `json:"choices"`
				}
				if err := json.Unmarshal(contRespBody, &contResp); err != nil || len(contResp.Choices) == 0 {
					break
				}
				contContent := contResp.Choices[0].Message.Content
				fullContent.WriteString(contContent)
				logger.Log.Infof("[Chat] Continuation %d/%d added %d chars, total=%d", contIdx+1, 5, len(contContent), fullContent.Len())

				if contResp.Choices[0].FinishReason != "length" {
					break // Response is complete
				}
				// Still truncated, append and ask to continue again
				messages = append(messages, ChatMessage{Role: "assistant", Content: contContent})
				if contIdx < 3 {
					messages = append(messages, ChatMessage{
						Role:    "user",
						Content: "继续完成回答，不要重复已输出内容。",
					})
				} else {
					// Last attempts: ask for a quick conclusion
					messages = append(messages, ChatMessage{
						Role:    "user",
						Content: "回答即将达到长度上限。请立即总结剩余要点，给出结论和可信度评分（📊 可信度：X/10），不再展开详细数据。",
					})
				}
			}
			content = fullContent.String()
		}

		logger.Log.Infof("[Chat] Final response at iteration %d, content_length=%d", iterations, len(content))
		if callback != nil {
			callback(content, true)
		}
		return content, nil
	}

	return "", fmt.Errorf("too many iterations in tool calling loop")
}

// getActiveAIConfig returns the AI config by checking the database for the default/enabled provider first.
func (a *Agent) getActiveAIConfig() (baseURL string, apiKey string, modelName string) {
	if repository.DB != nil {
		var provider model.AIProvider
		err := repository.DB.Where("is_default = ? AND is_enabled = ? AND api_key != ''", true, true).First(&provider).Error
		if err != nil {
			err = repository.DB.Where("is_enabled = ? AND api_key != ''", true).First(&provider).Error
		}
		if err == nil && provider.APIKey != "" && provider.BaseURL != "" {
			logger.Log.Infof("Using AI provider: %s (url=%s, model=%s)", provider.Label, provider.BaseURL, provider.Model)
			return provider.BaseURL, provider.APIKey, provider.Model
		}
	}
	return a.aiCfg.BaseURL, a.aiCfg.APIKey, a.aiCfg.Model
}

func (a *Agent) isSiliconFlowProvider(baseURL string) bool {
	return strings.Contains(strings.ToLower(baseURL), "siliconflow")
}

func (a *Agent) providerSupportsTools(baseURL string) bool {
	lower := strings.ToLower(baseURL)
	// Most major providers support OpenAI-compatible function/tool calling.
	// Only block providers known to NOT support it.
	noToolSupport := []string{"siliconflow", "api.minimax.chat", "hunyuan"}
	for _, p := range noToolSupport {
		if strings.Contains(lower, p) {
			return false
		}
	}
	return true
}

func (a *Agent) providerSupportsStream(baseURL string) bool {
	lower := strings.ToLower(baseURL)
	noStream := []string{"siliconflow", "api.minimax.chat", "aip.baidubce.com", "hunyuan", "baichuan"}
	for _, p := range noStream {
		if strings.Contains(lower, p) {
			return false
		}
	}
	return true
}

func (a *Agent) callAI(reqBody interface{}) ([]byte, error) {
	baseURL, apiKey, _ := a.getActiveAIConfig()
	body, _ := json.Marshal(reqBody)
	url := fmt.Sprintf("%s/chat/completions", strings.TrimRight(baseURL, "/"))

	logger.Log.Infof("callAI: url=%s, body_size=%d", url, len(body))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Errorf("callAI network error: %v", err)
		return nil, fmt.Errorf("AI API 网络请求失败: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		logger.Log.Errorf("callAI error: HTTP %d, url=%s, response=%s", resp.StatusCode, url, string(respBody))
		switch resp.StatusCode {
		case 401:
			return nil, fmt.Errorf("AI 服务认证失败(HTTP 401)，请检查 API Key")
		case 403:
			return nil, fmt.Errorf("AI 服务拒绝访问(HTTP 403): %s", string(respBody))
		case 404:
			return nil, fmt.Errorf("AI 服务接口未找到(HTTP 404)，请检查 Base URL")
		case 429:
			return nil, fmt.Errorf("AI 服务请求频率超限(HTTP 429)")
		default:
			return nil, fmt.Errorf("AI API 错误 (HTTP %d): %s", resp.StatusCode, string(respBody))
		}
	}

	return respBody, nil
}
