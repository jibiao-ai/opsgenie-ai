package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jibiao-ai/cloud-agent/internal/config"
	"github.com/jibiao-ai/cloud-agent/internal/easystack"
	"github.com/jibiao-ai/cloud-agent/internal/model"
	"github.com/jibiao-ai/cloud-agent/internal/repository"
	"github.com/jibiao-ai/cloud-agent/pkg/logger"
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

// Agent orchestrates AI reasoning with EasyStack tool calls
type Agent struct {
	aiCfg    config.AIConfig
	esClient *easystack.Client
	tools    []ToolDefinition
}

// NewAgent creates a new Agent
func NewAgent(aiCfg config.AIConfig, esClient *easystack.Client) *Agent {
	a := &Agent{
		aiCfg:    aiCfg,
		esClient: esClient,
	}
	a.initTools()
	return a
}

func (a *Agent) initTools() {
	toolDefs := []string{
		// Compute
		`{"type":"function","function":{"name":"list_servers","description":"列举所有云主机及其详细信息，包括状态、规格、IP地址等","parameters":{"type":"object","properties":{},"required":[]}}}`,
		`{"type":"function","function":{"name":"get_server","description":"查询指定云主机的详细信息","parameters":{"type":"object","properties":{"server_id":{"type":"string","description":"云主机ID"}},"required":["server_id"]}}}`,
		`{"type":"function","function":{"name":"create_server","description":"创建一台新的云主机","parameters":{"type":"object","properties":{"name":{"type":"string","description":"云主机名称"},"flavor_id":{"type":"string","description":"规格ID"},"image_id":{"type":"string","description":"镜像ID"},"network_id":{"type":"string","description":"网络ID"}},"required":["name","flavor_id","image_id","network_id"]}}}`,
		`{"type":"function","function":{"name":"start_server","description":"启动一台已停止的云主机","parameters":{"type":"object","properties":{"server_id":{"type":"string","description":"云主机ID"}},"required":["server_id"]}}}`,
		`{"type":"function","function":{"name":"stop_server","description":"关闭一台运行中的云主机","parameters":{"type":"object","properties":{"server_id":{"type":"string","description":"云主机ID"}},"required":["server_id"]}}}`,
		`{"type":"function","function":{"name":"reboot_server","description":"重启云主机","parameters":{"type":"object","properties":{"server_id":{"type":"string","description":"云主机ID"},"type":{"type":"string","enum":["SOFT","HARD"],"description":"重启类型：SOFT软重启，HARD硬重启"}},"required":["server_id"]}}}`,
		`{"type":"function","function":{"name":"delete_server","description":"删除云主机（危险操作，需确认）","parameters":{"type":"object","properties":{"server_id":{"type":"string","description":"云主机ID"}},"required":["server_id"]}}}`,
		`{"type":"function","function":{"name":"resize_server","description":"调整云主机规格","parameters":{"type":"object","properties":{"server_id":{"type":"string","description":"云主机ID"},"flavor_id":{"type":"string","description":"目标规格ID"}},"required":["server_id","flavor_id"]}}}`,
		`{"type":"function","function":{"name":"create_server_snapshot","description":"为云主机创建快照","parameters":{"type":"object","properties":{"server_id":{"type":"string","description":"云主机ID"},"name":{"type":"string","description":"快照名称"}},"required":["server_id","name"]}}}`,
		`{"type":"function","function":{"name":"attach_volume","description":"将云硬盘挂载到云主机","parameters":{"type":"object","properties":{"server_id":{"type":"string","description":"云主机ID"},"volume_id":{"type":"string","description":"云硬盘ID"},"device":{"type":"string","description":"设备路径，如/dev/vdb"}},"required":["server_id","volume_id"]}}}`,
		`{"type":"function","function":{"name":"detach_volume","description":"从云主机卸载云硬盘","parameters":{"type":"object","properties":{"server_id":{"type":"string","description":"云主机ID"},"volume_id":{"type":"string","description":"云硬盘ID"}},"required":["server_id","volume_id"]}}}`,
		`{"type":"function","function":{"name":"list_flavors","description":"列举所有可用的云主机规格（CPU、内存、磁盘等配置）","parameters":{"type":"object","properties":{},"required":[]}}}`,
		`{"type":"function","function":{"name":"list_keypairs","description":"列举所有SSH密钥对","parameters":{"type":"object","properties":{},"required":[]}}}`,
		// Storage
		`{"type":"function","function":{"name":"list_volumes","description":"列举所有云硬盘及其详细信息","parameters":{"type":"object","properties":{},"required":[]}}}`,
		`{"type":"function","function":{"name":"create_volume","description":"创建一个新的云硬盘","parameters":{"type":"object","properties":{"name":{"type":"string","description":"云硬盘名称"},"size":{"type":"integer","description":"大小(GB)"},"volume_type":{"type":"string","description":"云硬盘类型"}},"required":["name","size"]}}}`,
		`{"type":"function","function":{"name":"delete_volume","description":"删除云硬盘（危险操作）","parameters":{"type":"object","properties":{"volume_id":{"type":"string","description":"云硬盘ID"}},"required":["volume_id"]}}}`,
		`{"type":"function","function":{"name":"extend_volume","description":"扩容云硬盘","parameters":{"type":"object","properties":{"volume_id":{"type":"string","description":"云硬盘ID"},"new_size":{"type":"integer","description":"新大小(GB)"}},"required":["volume_id","new_size"]}}}`,
		`{"type":"function","function":{"name":"list_volume_snapshots","description":"列举所有云硬盘快照","parameters":{"type":"object","properties":{},"required":[]}}}`,
		// Network
		`{"type":"function","function":{"name":"list_networks","description":"列举所有网络","parameters":{"type":"object","properties":{},"required":[]}}}`,
		`{"type":"function","function":{"name":"create_network","description":"创建新网络","parameters":{"type":"object","properties":{"name":{"type":"string","description":"网络名称"}},"required":["name"]}}}`,
		`{"type":"function","function":{"name":"list_subnets","description":"列举所有子网","parameters":{"type":"object","properties":{},"required":[]}}}`,
		`{"type":"function","function":{"name":"list_routers","description":"列举所有路由器","parameters":{"type":"object","properties":{},"required":[]}}}`,
		`{"type":"function","function":{"name":"list_floating_ips","description":"列举所有浮动IP/公网IP","parameters":{"type":"object","properties":{},"required":[]}}}`,
		`{"type":"function","function":{"name":"list_security_groups","description":"列举所有安全组","parameters":{"type":"object","properties":{},"required":[]}}}`,
		`{"type":"function","function":{"name":"create_security_group","description":"创建安全组","parameters":{"type":"object","properties":{"name":{"type":"string","description":"安全组名称"},"description":{"type":"string","description":"描述"}},"required":["name"]}}}`,
		`{"type":"function","function":{"name":"create_security_group_rule","description":"创建安全组规则","parameters":{"type":"object","properties":{"security_group_id":{"type":"string","description":"安全组ID"},"direction":{"type":"string","enum":["ingress","egress"],"description":"方向"},"protocol":{"type":"string","description":"协议(tcp/udp/icmp)"},"port_range_min":{"type":"integer","description":"起始端口"},"port_range_max":{"type":"integer","description":"结束端口"}},"required":["security_group_id","direction"]}}}`,
		// Load Balancer
		`{"type":"function","function":{"name":"list_loadbalancers","description":"列举所有负载均衡器","parameters":{"type":"object","properties":{},"required":[]}}}`,
		`{"type":"function","function":{"name":"list_listeners","description":"列举所有监听器","parameters":{"type":"object","properties":{},"required":[]}}}`,
		`{"type":"function","function":{"name":"list_pools","description":"列举所有后端池","parameters":{"type":"object","properties":{},"required":[]}}}`,
		// Monitoring
		`{"type":"function","function":{"name":"query_metrics","description":"查询监控指标数据，使用PromQL表达式","parameters":{"type":"object","properties":{"expr":{"type":"string","description":"PromQL查询表达式，如cpu_util, memory_util等"},"start":{"type":"integer","description":"开始时间(Unix时间戳)"},"end":{"type":"integer","description":"结束时间(Unix时间戳)"},"step":{"type":"integer","description":"采样步长(秒)"}},"required":["expr"]}}}`,
		`{"type":"function","function":{"name":"list_alerts","description":"查询告警信息","parameters":{"type":"object","properties":{"states":{"type":"string","description":"告警状态过滤"},"severities":{"type":"string","description":"严重等级过滤"}},"required":[]}}}`,
		// Images
		`{"type":"function","function":{"name":"list_images","description":"列举所有可用镜像","parameters":{"type":"object","properties":{},"required":[]}}}`,
	}

	a.tools = make([]ToolDefinition, 0, len(toolDefs))
	for _, def := range toolDefs {
		var tool ToolDefinition
		json.Unmarshal([]byte(def), &tool)
		a.tools = append(a.tools, tool)
	}
}

// executeTool runs a tool call against EasyStack
func (a *Agent) executeTool(name string, args json.RawMessage) (string, error) {
	var params map[string]interface{}
	if err := json.Unmarshal(args, &params); err != nil {
		params = make(map[string]interface{})
	}

	getString := func(key string) string {
		if v, ok := params[key]; ok {
			return fmt.Sprintf("%v", v)
		}
		return ""
	}
	getInt := func(key string) int {
		if v, ok := params[key]; ok {
			switch val := v.(type) {
			case float64:
				return int(val)
			}
		}
		return 0
	}

	var result json.RawMessage
	var err error

	switch name {
	// Compute
	case "list_servers":
		result, err = a.esClient.ListServers()
	case "get_server":
		result, err = a.esClient.GetServer(getString("server_id"))
	case "create_server":
		result, err = a.esClient.CreateServer(map[string]interface{}{
			"name":      getString("name"),
			"flavorRef": getString("flavor_id"),
			"networks":  []map[string]string{{"uuid": getString("network_id")}},
			"block_device_mapping_v2": []map[string]interface{}{
				{
					"boot_index":            0,
					"uuid":                  getString("image_id"),
					"source_type":           "image",
					"destination_type":       "volume",
					"volume_size":           20,
					"delete_on_termination": true,
				},
			},
		})
	case "start_server":
		err = a.esClient.StartServer(getString("server_id"))
		if err == nil {
			return `{"status":"success","message":"云主机启动命令已发送"}`, nil
		}
	case "stop_server":
		err = a.esClient.StopServer(getString("server_id"))
		if err == nil {
			return `{"status":"success","message":"云主机关闭命令已发送"}`, nil
		}
	case "reboot_server":
		rebootType := getString("type")
		if rebootType == "" {
			rebootType = "SOFT"
		}
		err = a.esClient.RebootServer(getString("server_id"), rebootType)
		if err == nil {
			return `{"status":"success","message":"云主机重启命令已发送"}`, nil
		}
	case "delete_server":
		err = a.esClient.DeleteServer(getString("server_id"))
		if err == nil {
			return `{"status":"success","message":"云主机删除命令已发送"}`, nil
		}
	case "resize_server":
		err = a.esClient.ResizeServer(getString("server_id"), getString("flavor_id"))
		if err == nil {
			return `{"status":"success","message":"云主机调整规格命令已发送"}`, nil
		}
	case "create_server_snapshot":
		result, err = a.esClient.CreateServerSnapshot(getString("server_id"), getString("name"))
	case "attach_volume":
		device := getString("device")
		if device == "" {
			device = "/dev/vdb"
		}
		result, err = a.esClient.AttachVolume(getString("server_id"), getString("volume_id"), device)
	case "detach_volume":
		err = a.esClient.DetachVolume(getString("server_id"), getString("volume_id"))
		if err == nil {
			return `{"status":"success","message":"云硬盘卸载命令已发送"}`, nil
		}
	case "list_flavors":
		result, err = a.esClient.ListFlavors()
	case "list_keypairs":
		result, err = a.esClient.ListKeypairs()

	// Storage
	case "list_volumes":
		result, err = a.esClient.ListVolumes()
	case "create_volume":
		volParams := map[string]interface{}{
			"name": getString("name"),
			"size": getInt("size"),
		}
		if vt := getString("volume_type"); vt != "" {
			volParams["volume_type"] = vt
		}
		result, err = a.esClient.CreateVolume(volParams)
	case "delete_volume":
		err = a.esClient.DeleteVolume(getString("volume_id"))
		if err == nil {
			return `{"status":"success","message":"云硬盘删除命令已发送"}`, nil
		}
	case "extend_volume":
		err = a.esClient.ExtendVolume(getString("volume_id"), getInt("new_size"))
		if err == nil {
			return `{"status":"success","message":"云硬盘扩容命令已发送"}`, nil
		}
	case "list_volume_snapshots":
		result, err = a.esClient.ListVolumeSnapshots()

	// Network
	case "list_networks":
		result, err = a.esClient.ListNetworks()
	case "create_network":
		result, err = a.esClient.CreateNetwork(getString("name"))
	case "list_subnets":
		result, err = a.esClient.ListSubnets()
	case "list_routers":
		result, err = a.esClient.ListRouters()
	case "list_floating_ips":
		result, err = a.esClient.ListFloatingIPs()
	case "list_security_groups":
		result, err = a.esClient.ListSecurityGroups()
	case "create_security_group":
		result, err = a.esClient.CreateSecurityGroup(getString("name"), getString("description"))
	case "create_security_group_rule":
		ruleParams := map[string]interface{}{
			"security_group_id": getString("security_group_id"),
			"direction":         getString("direction"),
		}
		if p := getString("protocol"); p != "" {
			ruleParams["protocol"] = p
		}
		if min := getInt("port_range_min"); min > 0 {
			ruleParams["port_range_min"] = min
		}
		if max := getInt("port_range_max"); max > 0 {
			ruleParams["port_range_max"] = max
		}
		result, err = a.esClient.CreateSecurityGroupRule(ruleParams)

	// Load Balancer
	case "list_loadbalancers":
		result, err = a.esClient.ListLoadBalancers()
	case "list_listeners":
		result, err = a.esClient.ListListeners()
	case "list_pools":
		result, err = a.esClient.ListPools()

	// Monitoring
	case "query_metrics":
		start := int64(getInt("start"))
		end := int64(getInt("end"))
		step := int64(getInt("step"))
		if start == 0 {
			start = time.Now().Add(-1 * time.Hour).Unix()
		}
		if end == 0 {
			end = time.Now().Unix()
		}
		if step == 0 {
			step = 60
		}
		result, err = a.esClient.QueryMetrics(getString("expr"), start, end, step)
	case "list_alerts":
		result, err = a.esClient.ListAlerts(getString("states"), "", getString("severities"))

	// Images
	case "list_images":
		result, err = a.esClient.ListImages()

	default:
		return fmt.Sprintf(`{"error":"unknown tool: %s"}`, name), nil
	}

	if err != nil {
		return fmt.Sprintf(`{"error":"%s"}`, err.Error()), nil
	}

	if result == nil {
		return `{"status":"success"}`, nil
	}

	// Truncate very large responses
	resultStr := string(result)
	if len(resultStr) > 8000 {
		resultStr = resultStr[:8000] + "...(truncated)"
	}
	return resultStr, nil
}

// Chat sends a user message through the AI with tool calling support
func (a *Agent) Chat(agentModel model.Agent, history []ChatMessage, userMsg string, callback StreamCallback) (string, error) {
	messages := []ChatMessage{
		{Role: "system", Content: agentModel.SystemPrompt},
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

	logger.Log.Infof("Chat request: provider_url=%s, model=%s", baseURL, modelName)

	// Decide whether to include tools and stream in the request.
	// Providers like SiliconFlow return 403/400 when unsupported parameters
	// (e.g. "tools", "stream") are sent.
	supportsTools := a.providerSupportsTools(baseURL)
	supportsStream := a.providerSupportsStream(baseURL)
	isSiliconFlow := a.isSiliconFlowProvider(baseURL)

	// Loop for tool calling
	for iterations := 0; iterations < 10; iterations++ {
		reqBody := map[string]interface{}{
			"model":       modelName,
			"messages":    messages,
		}
		// Only include temperature/max_tokens if they have meaningful values
		if agentModel.Temperature > 0 {
			reqBody["temperature"] = agentModel.Temperature
		}
		if agentModel.MaxTokens > 0 {
			// SiliconFlow uses "max_tokens" but is strict about values;
			// cap it to avoid API errors
			if isSiliconFlow {
				reqBody["max_tokens"] = agentModel.MaxTokens
			} else {
				reqBody["max_tokens"] = agentModel.MaxTokens
			}
		}
		// Only send stream parameter for providers that support it
		if supportsStream {
			reqBody["stream"] = callback != nil
		}
		// Only attach tools when the provider is known to support them
		if supportsTools && len(a.tools) > 0 {
			reqBody["tools"] = a.tools
		}

		respBody, err := a.callAI(reqBody)
		if err != nil {
			return "", fmt.Errorf("AI call failed: %w", err)
		}

		// Parse response
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

		// If there are tool calls, execute them
		if len(choice.Message.ToolCalls) > 0 {
			toolCallsJSON, _ := json.Marshal(choice.Message.ToolCalls)
			messages = append(messages, ChatMessage{
				Role:      "assistant",
				Content:   choice.Message.Content,
				ToolCalls: toolCallsJSON,
			})

			for _, tc := range choice.Message.ToolCalls {
				logger.Log.Infof("Executing tool: %s with args: %s", tc.Function.Name, tc.Function.Arguments)
				toolResult, err := a.executeTool(tc.Function.Name, json.RawMessage(tc.Function.Arguments))
				if err != nil {
					toolResult = fmt.Sprintf(`{"error":"%s"}`, err.Error())
				}
				messages = append(messages, ChatMessage{
					Role:       "tool",
					Content:    toolResult,
					ToolCallID: tc.ID,
					Name:       tc.Function.Name,
				})
			}
			continue
		}

		// No tool calls, return the final response
		content := choice.Message.Content
		if callback != nil {
			callback(content, true)
		}
		return content, nil
	}

	return "", fmt.Errorf("too many iterations in tool calling loop")
}

// getActiveAIConfig returns the AI config by checking the database for the default/enabled provider first,
// falling back to the static config if no database provider is configured.
func (a *Agent) getActiveAIConfig() (baseURL string, apiKey string, modelName string) {
	// Try to find the default enabled provider from the database
	if repository.DB != nil {
		var provider model.AIProvider
		// First try: default + enabled + has API key
		err := repository.DB.Where("is_default = ? AND is_enabled = ? AND api_key != ''", true, true).First(&provider).Error
		if err != nil {
			// Fallback: any enabled provider with an API key
			err = repository.DB.Where("is_enabled = ? AND api_key != ''", true).First(&provider).Error
		}
		if err == nil && provider.APIKey != "" && provider.BaseURL != "" {
			logger.Log.Infof("Using database AI provider: %s (base_url=%s, model=%s)", provider.Label, provider.BaseURL, provider.Model)
			return provider.BaseURL, provider.APIKey, provider.Model
		}
	}
	// Fallback to static config
	return a.aiCfg.BaseURL, a.aiCfg.APIKey, a.aiCfg.Model
}

// isSiliconFlowProvider checks if the provider URL belongs to SiliconFlow.
func (a *Agent) isSiliconFlowProvider(baseURL string) bool {
	lower := strings.ToLower(baseURL)
	return strings.Contains(lower, "siliconflow")
}

// providerSupportsTools returns true if the AI provider is known to support
// OpenAI-compatible function calling (tools parameter).
func (a *Agent) providerSupportsTools(baseURL string) bool {
	lower := strings.ToLower(baseURL)
	// Providers known to SUPPORT tools — whitelist approach
	supportedPatterns := []string{
		"api.openai.com",
		"api.deepseek.com",
		"api.anthropic.com",
		"generativelanguage.googleapis.com", // Gemini
	}
	for _, pattern := range supportedPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	// All other providers: do NOT send tools to avoid 403/400
	return false
}

// providerSupportsStream returns true if the provider supports the "stream" parameter.
// SiliconFlow and some other providers reject the stream field entirely.
func (a *Agent) providerSupportsStream(baseURL string) bool {
	lower := strings.ToLower(baseURL)
	// Providers known NOT to handle stream well
	noStreamPatterns := []string{
		"siliconflow",
		"api.minimax.chat",
		"aip.baidubce.com",
		"hunyuan",
		"baichuan",
	}
	for _, pattern := range noStreamPatterns {
		if strings.Contains(lower, pattern) {
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
		// Provide user-friendly error messages for common HTTP status codes
		switch resp.StatusCode {
		case 401:
			return nil, fmt.Errorf("AI 服务认证失败(HTTP 401)，请检查 API Key 是否正确")
		case 403:
			return nil, fmt.Errorf("AI 服务拒绝访问(HTTP 403)，请检查 API Key 权限或模型名称是否正确: %s", string(respBody))
		case 404:
			return nil, fmt.Errorf("AI 服务接口未找到(HTTP 404)，请检查 Base URL 和模型名称是否正确")
		case 429:
			return nil, fmt.Errorf("AI 服务请求频率超限(HTTP 429)，请稍后重试")
		default:
			return nil, fmt.Errorf("AI API 错误 (HTTP %d): %s", resp.StatusCode, string(respBody))
		}
	}

	return respBody, nil
}
