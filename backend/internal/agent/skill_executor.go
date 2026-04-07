package agent

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jibiao-ai/opsgenie-ai/internal/model"
	"github.com/jibiao-ai/opsgenie-ai/pkg/logger"
)

// SkillExecutor authenticates to a cloud platform and executes tool calls.
// It caches tokens per platform ID to avoid re-authenticating on every call.
type SkillExecutor struct {
	mu          sync.RWMutex
	tokens      map[uint]*platformToken // keyed by CloudPlatform.ID
	httpClients map[uint]*http.Client   // per-platform HTTP client (with custom DNS)
}

type platformToken struct {
	tokenType string // "keystone" or "zstack"
	token     string // X-Auth-Token (EasyStack) or session ID (ZStack)
	expiresAt time.Time
	platform  model.CloudPlatform
}

// NewSkillExecutor creates a new executor.
func NewSkillExecutor() *SkillExecutor {
	return &SkillExecutor{
		tokens:      make(map[uint]*platformToken),
		httpClients: make(map[uint]*http.Client),
	}
}

// getHTTPClient returns a cached HTTP client for the platform.
// If the platform has HostIP+BaseDomain, the client uses custom DNS resolution.
func (se *SkillExecutor) getHTTPClient(p model.CloudPlatform) *http.Client {
	se.mu.RLock()
	if c, ok := se.httpClients[p.ID]; ok {
		se.mu.RUnlock()
		return c
	}
	se.mu.RUnlock()

	client := NewHTTPClientWithCustomDNS(p.HostIP, p.BaseDomain, 30*time.Second)

	se.mu.Lock()
	se.httpClients[p.ID] = client
	se.mu.Unlock()

	return client
}

// Authenticate obtains a valid token for the given cloud platform.
// The result is cached so subsequent calls within the validity period are fast.
func (se *SkillExecutor) Authenticate(p model.CloudPlatform) (string, error) {
	se.mu.RLock()
	if cached, ok := se.tokens[p.ID]; ok && time.Now().Before(cached.expiresAt) {
		se.mu.RUnlock()
		return cached.token, nil
	}
	se.mu.RUnlock()

	var token string
	var err error
	var tokenType string

	switch strings.ToLower(p.Type) {
	case "easystack":
		token, err = se.authenticateEasyStack(p)
		tokenType = "keystone"
	case "zstack":
		token, err = se.authenticateZStack(p)
		tokenType = "zstack"
	default:
		return "", fmt.Errorf("unsupported platform type: %s", p.Type)
	}
	if err != nil {
		return "", fmt.Errorf("cloud platform '%s' authentication failed: %w", p.Name, err)
	}

	se.mu.Lock()
	se.tokens[p.ID] = &platformToken{
		tokenType: tokenType,
		token:     token,
		expiresAt: time.Now().Add(5 * time.Hour), // EasyStack token expires in ~6h, refresh at 5h
		platform:  p,
	}
	se.mu.Unlock()

	logger.Log.Infof("SkillExecutor: authenticated to %s platform '%s'", p.Type, p.Name)
	return token, nil
}

// authenticateEasyStack obtains a Keystone token.
func (se *SkillExecutor) authenticateEasyStack(p model.CloudPlatform) (string, error) {
	authReq := map[string]interface{}{
		"auth": map[string]interface{}{
			"identity": map[string]interface{}{
				"methods": []string{"password"},
				"password": map[string]interface{}{
					"user": map[string]interface{}{
						"name":     p.Username,
						"password": p.Password,
						"domain":   map[string]string{"name": p.DomainName},
					},
				},
			},
			"scope": map[string]interface{}{
				"project": map[string]interface{}{
					"name":   p.ProjectName,
					"domain": map[string]string{"name": p.DomainName},
				},
			},
		},
	}
	body, _ := json.Marshal(authReq)
	// Resolve Keystone URL: prefer HostIP+BaseDomain, fall back to AuthURL
	endpoints := ResolveEasyStackEndpoints(p)
	url := fmt.Sprintf("%s/v3/auth/tokens", strings.TrimRight(endpoints.Keystone, "/"))

	logger.Log.Infof("[Keystone] Authenticating: POST %s (user=%s, project=%s, domain=%s)", url, p.Username, p.ProjectName, p.DomainName)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	httpClient := se.getHTTPClient(p)
	resp, err := httpClient.Do(req)
	if err != nil {
		logger.Log.Errorf("[Keystone] Network error: POST %s → %v", url, err)
		return "", fmt.Errorf("network error calling %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		logger.Log.Errorf("[Keystone] Auth failed: POST %s → HTTP %d: %s", url, resp.StatusCode, string(respBody[:min(len(respBody), 500)]))
		return "", fmt.Errorf("Keystone auth HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	token := resp.Header.Get("X-Subject-Token")
	logger.Log.Infof("[Keystone] Auth success: POST %s → HTTP %d, token=%s...", url, resp.StatusCode, token[:min(len(token), 16)])
	return token, nil
}

// authenticateZStack obtains a ZStack session ID via AccessKey HMAC signing.
func (se *SkillExecutor) authenticateZStack(p model.CloudPlatform) (string, error) {
	endpoint := strings.TrimRight(p.Endpoint, "/")
	ts := time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")

	stringToSign := fmt.Sprintf("POST\n\napplication/json\n%s\n/v1/accounts/login", ts)
	mac := hmac.New(sha256.New, []byte(p.AccessKeySecret))
	mac.Write([]byte(stringToSign))
	sig := hex.EncodeToString(mac.Sum(nil))

	loginBody := map[string]interface{}{
		"loginByAccessKey": map[string]string{
			"accessKeyId":     p.AccessKeyID,
			"accessKeySecret": sig,
		},
	}
	body, _ := json.Marshal(loginBody)
	url := fmt.Sprintf("%s/v1/accounts/login", endpoint)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Date", ts)
	req.Header.Set("Authorization", fmt.Sprintf("ZStack %s:%s", p.AccessKeyID, sig))

	httpClient := se.getHTTPClient(p)
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Inventory struct {
			UUID string `json:"uuid"`
		} `json:"inventory"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	return result.Inventory.UUID, nil
}

// clearTokenCache removes the cached token for a platform, forcing re-authentication on next call.
func (se *SkillExecutor) clearTokenCache(platformID uint) {
	se.mu.Lock()
	delete(se.tokens, platformID)
	se.mu.Unlock()
}

// ExecuteTool runs a tool call against the bound cloud platform.
// It determines the correct API endpoint based on the platform type and tool name.
// On 401/403 errors, it automatically clears the cached token and retries once.
func (se *SkillExecutor) ExecuteTool(platform model.CloudPlatform, toolName string, args json.RawMessage) (string, error) {
	token, err := se.Authenticate(platform)
	if err != nil {
		return "", err
	}

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
			if val, ok := v.(float64); ok {
				return int(val)
			}
		}
		return 0
	}

	execute := func(t string) (string, error) {
		switch strings.ToLower(platform.Type) {
		case "easystack":
			return se.executeEasyStack(platform, t, toolName, params, getString, getInt)
		case "zstack":
			return se.executeZStack(platform, t, toolName, params, getString, getInt)
		default:
			return fmt.Sprintf(`{"error":"unsupported platform type: %s"}`, platform.Type), nil
		}
	}

	result, err := execute(token)
	if err != nil && (strings.Contains(err.Error(), "HTTP 401") || strings.Contains(err.Error(), "HTTP 403")) {
		logger.Log.Warnf("[ExecuteTool] Tool '%s' got auth error: %v — clearing token cache and retrying", toolName, err)
		se.clearTokenCache(platform.ID)
		newToken, authErr := se.Authenticate(platform)
		if authErr != nil {
			return "", fmt.Errorf("re-authentication failed after 401/403: %w", authErr)
		}
		logger.Log.Infof("[ExecuteTool] Re-authenticated successfully, retrying tool '%s'", toolName)
		return execute(newToken)
	}
	return result, err
}

// executeEasyStack runs a tool against an EasyStack (OpenStack) cloud platform.
// It resolves service endpoints per tool: compute→Nova, monitoring→EMLA, storage→Cinder, etc.
func (se *SkillExecutor) executeEasyStack(p model.CloudPlatform, token, toolName string, params map[string]interface{}, getString func(string) string, getInt func(string) int) (string, error) {
	// Resolve per-service endpoints from HostIP + BaseDomain (or fallback to AuthURL)
	endpoints := ResolveEasyStackEndpoints(p)
	serviceURL := endpoints.ServiceURLFor(toolName)
	projectID := p.ProjectID
	httpClient := se.getHTTPClient(p)

	logger.Log.Infof("[EasyStack] Tool '%s' → service URL: %s", toolName, serviceURL)

	doReq := func(method, url string, body interface{}) (json.RawMessage, error) {
		var reqBody io.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			reqBody = bytes.NewBuffer(b)
		}
		req, err := http.NewRequest(method, url, reqBody)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-Auth-Token", token)
		logger.Log.Debugf("[EasyStack] %s %s (token=%s...)", method, url, token[:min(len(token), 16)])
		resp, err := httpClient.Do(req)
		if err != nil {
			logger.Log.Errorf("[EasyStack] Network error calling %s %s: %v", method, url, err)
			return nil, err
		}
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 400 {
			logger.Log.Errorf("[EasyStack] %s %s → HTTP %d: %s", method, url, resp.StatusCode, string(respBody[:min(len(respBody), 500)]))
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
		}
		logger.Log.Infof("[EasyStack] %s %s → HTTP %d (%d bytes)", method, url, resp.StatusCode, len(respBody))
		return respBody, nil
	}

	var result json.RawMessage
	var err error

	switch toolName {
	// Compute
	case "list_servers":
		result, err = doReq("GET", fmt.Sprintf("%s/v2.1/servers/detail", serviceURL), nil)
	case "get_server":
		result, err = doReq("GET", fmt.Sprintf("%s/v2.1/servers/%s", serviceURL, getString("server_id")), nil)
	case "create_server":
		result, err = doReq("POST", fmt.Sprintf("%s/v2.1/servers", serviceURL), map[string]interface{}{
			"server": map[string]interface{}{
				"name": getString("name"), "flavorRef": getString("flavor_id"),
				"networks": []map[string]string{{"uuid": getString("network_id")}},
				"block_device_mapping_v2": []map[string]interface{}{{
					"boot_index": 0, "uuid": getString("image_id"), "source_type": "image",
					"destination_type": "volume", "volume_size": 20, "delete_on_termination": true,
				}},
			},
		})
	case "start_server":
		_, err = doReq("POST", fmt.Sprintf("%s/v2.1/servers/%s/action", serviceURL, getString("server_id")),
			map[string]interface{}{"os-start": nil})
		if err == nil {
			return `{"status":"success","message":"云主机启动命令已发送"}`, nil
		}
	case "stop_server":
		_, err = doReq("POST", fmt.Sprintf("%s/v2.1/servers/%s/action", serviceURL, getString("server_id")),
			map[string]interface{}{"os-stop": nil})
		if err == nil {
			return `{"status":"success","message":"云主机关闭命令已发送"}`, nil
		}
	case "reboot_server":
		rt := getString("type")
		if rt == "" {
			rt = "SOFT"
		}
		_, err = doReq("POST", fmt.Sprintf("%s/v2.1/servers/%s/action", serviceURL, getString("server_id")),
			map[string]interface{}{"reboot": map[string]string{"type": rt}})
		if err == nil {
			return `{"status":"success","message":"云主机重启命令已发送"}`, nil
		}
	case "delete_server":
		_, err = doReq("DELETE", fmt.Sprintf("%s/v2.1/servers/%s", serviceURL, getString("server_id")), nil)
		if err == nil {
			return `{"status":"success","message":"云主机删除命令已发送"}`, nil
		}
	case "list_flavors":
		result, err = doReq("GET", fmt.Sprintf("%s/v2.1/flavors/detail", serviceURL), nil)
	case "list_images":
		result, err = doReq("GET", fmt.Sprintf("%s/v2/images", serviceURL), nil)
	// Storage
	case "list_volumes":
		result, err = doReq("GET", fmt.Sprintf("%s/v2/%s/volumes/detail", serviceURL, projectID), nil)
	case "create_volume":
		volParams := map[string]interface{}{"name": getString("name"), "size": getInt("size")}
		if vt := getString("volume_type"); vt != "" {
			volParams["volume_type"] = vt
		}
		if desc := getString("description"); desc != "" {
			volParams["description"] = desc
		}
		if imgRef := getString("imageRef"); imgRef != "" {
			volParams["imageRef"] = imgRef
		}
		result, err = doReq("POST", fmt.Sprintf("%s/v2/%s/volumes", serviceURL, projectID), map[string]interface{}{"volume": volParams})
	case "delete_volume":
		_, err = doReq("DELETE", fmt.Sprintf("%s/v2/%s/volumes/%s", serviceURL, projectID, getString("volume_id")), nil)
		if err == nil {
			return `{"status":"success","message":"云硬盘删除命令已发送"}`, nil
		}
	case "extend_volume":
		_, err = doReq("POST", fmt.Sprintf("%s/v2/%s/volumes/%s/action", serviceURL, projectID, getString("volume_id")),
			map[string]interface{}{"os-extend": map[string]int{"new_size": getInt("new_size")}})
		if err == nil {
			return `{"status":"success","message":"云硬盘扩容命令已发送"}`, nil
		}
	case "list_volume_snapshots":
		result, err = doReq("GET", fmt.Sprintf("%s/v2/%s/snapshots/detail", serviceURL, projectID), nil)
	// -- Volume types (per EasyStack API doc Section 3.1) --
	case "list_volume_types":
		result, err = doReq("GET", fmt.Sprintf("%s/v2/%s/types", serviceURL, projectID), nil)
	// -- Volume detail (per EasyStack API doc Section 4.5) --
	case "get_volume_detail":
		volID := getString("volume_id")
		if volID != "" {
			result, err = doReq("GET", fmt.Sprintf("%s/v2/%s/volumes/detail?id=%s", serviceURL, projectID, volID), nil)
		} else {
			result, err = doReq("GET", fmt.Sprintf("%s/v2/%s/volumes/detail", serviceURL, projectID), nil)
		}
	// -- Storage pools (per EasyStack API doc Section 4.5 interface 2) --
	case "get_storage_pools":
		result, err = doReq("GET", fmt.Sprintf("%s/v3/%s/scheduler-stats/get_pools?detail=true", serviceURL, projectID), nil)
	// -- Attach volume to server (per EasyStack API doc Section 4.3) --
	case "attach_volume":
		serverID := getString("server_id")
		volumeID := getString("volume_id")
		attachBody := map[string]interface{}{
			"volumeAttachment": map[string]interface{}{
				"volumeId": volumeID,
			},
		}
		if dev := getString("device"); dev != "" {
			attachBody["volumeAttachment"].(map[string]interface{})["device"] = dev
		}
		// attach_volume uses Nova endpoint (serviceURL already resolved to Nova)
		result, err = doReq("POST", fmt.Sprintf("%s/v2.1/servers/%s/os-volume_attachments", serviceURL, serverID), attachBody)
		if err == nil && result == nil {
			return `{"status":"success","message":"云硬盘挂载命令已发送"}`, nil
		}
	// -- Detach volume from server --
	case "detach_volume":
		serverID := getString("server_id")
		attachmentID := getString("attachment_id")
		// detach_volume uses Nova endpoint (serviceURL already resolved to Nova)
		_, err = doReq("DELETE", fmt.Sprintf("%s/v2.1/servers/%s/os-volume_attachments/%s", serviceURL, serverID, attachmentID), nil)
		if err == nil {
			return `{"status":"success","message":"云硬盘卸载命令已发送"}`, nil
		}
	// Network
	case "list_networks":
		result, err = doReq("GET", fmt.Sprintf("%s/v2.0/networks", serviceURL), nil)
	case "list_subnets":
		result, err = doReq("GET", fmt.Sprintf("%s/v2.0/subnets", serviceURL), nil)
	case "list_routers":
		result, err = doReq("GET", fmt.Sprintf("%s/v2.0/routers", serviceURL), nil)
	case "list_floating_ips":
		result, err = doReq("GET", fmt.Sprintf("%s/v2.0/floatingips", serviceURL), nil)
	case "list_security_groups":
		result, err = doReq("GET", fmt.Sprintf("%s/v2.0/security-groups", serviceURL), nil)
	case "create_security_group":
		result, err = doReq("POST", fmt.Sprintf("%s/v2.0/security-groups", serviceURL),
			map[string]interface{}{"security_group": map[string]string{"name": getString("name"), "description": getString("description")}})
	case "create_security_group_rule":
		ruleParams := map[string]interface{}{"security_group_id": getString("security_group_id"), "direction": getString("direction")}
		if p := getString("protocol"); p != "" {
			ruleParams["protocol"] = p
		}
		if min := getInt("port_range_min"); min > 0 {
			ruleParams["port_range_min"] = min
		}
		if max := getInt("port_range_max"); max > 0 {
			ruleParams["port_range_max"] = max
		}
		result, err = doReq("POST", fmt.Sprintf("%s/v2.0/security-group-rules", serviceURL),
			map[string]interface{}{"security_group_rule": ruleParams})
	// LB
	case "list_loadbalancers":
		result, err = doReq("GET", fmt.Sprintf("%s/v2.0/lbaas/loadbalancers", serviceURL), nil)
	case "list_listeners":
		result, err = doReq("GET", fmt.Sprintf("%s/v2.0/lbaas/listeners", serviceURL), nil)
	case "list_pools":
		result, err = doReq("GET", fmt.Sprintf("%s/v2.0/lbaas/pools", serviceURL), nil)
	// ==================== Monitoring / Observability (ECF 6.2.1) ====================
	// Metrics API prefix:  /emla/openapi/v1/{project_id}/... (Section 15.1)
	// Alert API:           /apis/monitoring/v1/ecms/alerts  (Prometheus-compat ECMS)

	// -- Instant PromQL query --
	case "query_metrics":
		expr := getString("expr")
		queryParams := map[string]interface{}{"expr": expr}
		if t := int64(getInt("time")); t > 0 {
			queryParams["time"] = t
		} else {
			queryParams["time"] = time.Now().Unix()
		}
		result, err = doReq("POST", fmt.Sprintf("%s/emla/openapi/v1/%s/metrics/query", serviceURL, projectID), queryParams)

	// -- Range PromQL query --
	case "query_metrics_range":
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
		result, err = doReq("POST", fmt.Sprintf("%s/emla/openapi/v1/%s/metrics/query/range", serviceURL, projectID),
			map[string]interface{}{"expr": getString("expr"), "start": start, "end": end, "step": step})

	// -- List alerts (generic, supports state and severity filters) --
	case "list_alerts":
		// Prometheus-compat ECMS alert API: /apis/monitoring/v1/ecms/alerts
		alertURL := fmt.Sprintf("%s/apis/monitoring/v1/ecms/alerts", serviceURL)
		qParams := []string{}
		if s := getString("alerts_status"); s != "" {
			qParams = append(qParams, "alerts_status="+s)
		}
		if s := getString("severity"); s != "" {
			qParams = append(qParams, "severity="+s)
		}
		if len(qParams) > 0 {
			alertURL += "?" + strings.Join(qParams, "&")
		}
		result, err = doReq("GET", alertURL, nil)
		// Pre-filter: extract only essential alert fields to reduce token usage
		result = preFilterAlerts(result, err)

	// -- Active alerts (unresolved / firing) --
	case "list_active_alerts":
		alertURL := fmt.Sprintf("%s/apis/monitoring/v1/ecms/alerts?alerts_status=unresolved", serviceURL)
		if s := getString("severity"); s != "" {
			alertURL += "&severity=" + s
		}
		result, err = doReq("GET", alertURL, nil)
		result = preFilterAlerts(result, err)

	// -- Recovered alerts (resolved) --
	case "list_recovered_alerts":
		alertURL := fmt.Sprintf("%s/apis/monitoring/v1/ecms/alerts?alerts_status=resolved", serviceURL)
		if s := getString("severity"); s != "" {
			alertURL += "&severity=" + s
		}
		result, err = doReq("GET", alertURL, nil)
		result = preFilterAlerts(result, err)

	// -- Alarm severity summary --
	case "get_alarm_severity_summary":
		alertURL := fmt.Sprintf("%s/apis/monitoring/v1/ecms/alerts", serviceURL)
		result, err = doReq("GET", alertURL, nil)
		if err == nil && result != nil {
			// Parse EMLA response and extract statistics only
			var alertResp struct {
				Code int `json:"code"`
				Data struct {
					Statistics struct {
						Total    int `json:"total"`
						Critical int `json:"critical"`
						Warning  int `json:"warning"`
						Info     int `json:"info"`
					} `json:"statistics"`
				} `json:"data"`
				// Also try flat format
				Total     int `json:"total"`
				LevelInfo struct {
					Critical int `json:"critical"`
					Warning  int `json:"warning"`
					Info     int `json:"info"`
				} `json:"level_info"`
			}
			if json.Unmarshal(result, &alertResp) == nil {
				summary := map[string]interface{}{}
				if alertResp.Data.Statistics.Total > 0 || alertResp.Data.Statistics.Critical > 0 {
					summary["total"] = alertResp.Data.Statistics.Total
					summary["critical"] = alertResp.Data.Statistics.Critical
					summary["warning"] = alertResp.Data.Statistics.Warning
					summary["info"] = alertResp.Data.Statistics.Info
				} else {
					summary["total"] = alertResp.Total
					summary["critical"] = alertResp.LevelInfo.Critical
					summary["warning"] = alertResp.LevelInfo.Warning
					summary["info"] = alertResp.LevelInfo.Info
				}
				summary["_source"] = "emla_alerts_messages_api"
				summaryJSON, _ := json.Marshal(summary)
				result = summaryJSON
			}
		}

	// -- Control plane service status (official ECF 6.2.1 metric names from Section 15.1.4.4) --
	case "get_control_plane_status":
		// Use EMLA metrics query with official service_*_state metric names
		serviceMetrics := strings.Join([]string{
			"service_control_api_state",
			"service_control_scheduler_state",
			"service_control_management_state",
			"service_compute_api_state",
			"service_compute_management_state",
			"service_compute_state",
			"service_compute_scheduler_state",
			"service_network_api_state",
			"service_network_dhcp_state",
			"service_network_l3_state",
			"service_network_lb_state",
			"service_network_metadata_state",
			"service_network_virtual_switch_state",
			"service_authentication_api_state",
			"service_image_management_state",
			"service_block_storage_api_state",
			"service_block_storage_scheduler_state",
			"service_block_storage_state",
			"service_monitoring_api_state",
			"service_monitoring_alert_api_state",
			"service_database_state",
			"service_rabbitmq_state",
			"service_orchestration_api_state",
			"service_hostha_state",
			"service_time_synchronization_state",
			"service_cloud_console_state",
		}, "|")
		// Use PromQL union query: {__name__=~"service_...|service_..."}
		expr := fmt.Sprintf(`{__name__=~"%s"}`, serviceMetrics)
		result, err = doReq("POST", fmt.Sprintf("%s/emla/openapi/v1/%s/metrics/query", serviceURL, projectID),
			map[string]interface{}{"expr": expr, "time": time.Now().Unix()})

	// -- Storage cluster status (official ECF 6.2.1 metric names from Section 15.1.4.5) --
	case "get_storage_cluster_status":
		storageMetrics := strings.Join([]string{
			"storage_health_status",
			"storage_osd_total",
			"storage_osd_up_total",
			"storage_osd_down_total",
			"storage_actual_capacity_total_bytes",
			"storage_actual_capacity_free_bytes",
			"storage_actual_capacity_usage_bytes",
			"storage_cluster_iops_read",
			"storage_cluster_iops_write",
			"storage_cluster_throughput_read",
			"storage_cluster_throughput_write",
			"storage_volume_pool_bytes",
			"storage_image_pool_bytes",
		}, "|")
		expr := fmt.Sprintf(`{__name__=~"%s"}`, storageMetrics)
		result, err = doReq("POST", fmt.Sprintf("%s/emla/openapi/v1/%s/metrics/query", serviceURL, projectID),
			map[string]interface{}{"expr": expr, "time": time.Now().Unix()})

	// -- Dashboard overview (official ECF 6.2.1 metric names from Section 15.1.4.2) --
	case "get_dashboard_overview":
		dashMetrics := strings.Join([]string{
			"dashboard_instances_state",
			"dashboard_instances_vcpu_usage",
			"dashboard_instances_memory_usage",
			"dashboard_instances_volumes_usage",
			"dashboard_control_plane_service_health",
			"dashboard_storage_service_health",
			"dashboard_node_state_total",
			"dashboard_node_state_online",
			"dashboard_cpu_total",
			"dashboard_cpu_usage",
			"dashboard_cpu_free",
			"dashboard_memory_total",
			"dashboard_memory_usage",
			"dashboard_memory_free",
			"dashboard_storage_total",
			"dashboard_storage_usage",
			"dashboard_storage_free",
		}, "|")
		expr := fmt.Sprintf(`{__name__=~"%s"}`, dashMetrics)
		result, err = doReq("POST", fmt.Sprintf("%s/emla/openapi/v1/%s/metrics/query", serviceURL, projectID),
			map[string]interface{}{"expr": expr, "time": time.Now().Unix()})

	// -- All services health check (comprehensive, all 42 service metrics) --
	case "check_all_services_health":
		allMetrics := strings.Join([]string{
			"service_control_api_state",
			"service_control_scheduler_state",
			"service_control_management_state",
			"service_compute_api_state",
			"service_compute_management_state",
			"service_compute_state",
			"service_compute_scheduler_state",
			"service_network_vnc_state",
			"service_network_api_state",
			"service_network_metadata_state",
			"service_network_virtual_switch_state",
			"service_network_dhcp_state",
			"service_network_l3_state",
			"service_network_lb_state",
			"service_authentication_api_state",
			"service_image_management_state",
			"service_virtualization_management_state",
			"service_hostha_state",
			"service_rabbitmq_state",
			"service_database_state",
			"service_automation_center_state",
			"service_time_synchronization_state",
			"service_cloud_console_state",
			"service_cloud_automation_state",
			"service_high_performance_cache_state",
			"service_block_storage_api_state",
			"service_block_storage_scheduler_state",
			"service_block_storage_state",
			"service_block_storage_backup_state",
			"service_monitoring_api_state",
			"service_monitoring_alert_api_state",
			"service_monitoring_storage_api_state",
			"service_log_collection_state",
			"service_orchestration_api_state",
			"service_data_protection_state",
			"service_billing_api_state",
			"service_object_storage_api_state",
			"service_container_cluster_management_api_state",
		}, "|")
		expr := fmt.Sprintf(`{__name__=~"%s"}`, allMetrics)
		result, err = doReq("POST", fmt.Sprintf("%s/emla/openapi/v1/%s/metrics/query", serviceURL, projectID),
			map[string]interface{}{"expr": expr, "time": time.Now().Unix()})

	// ==================== Metering Service Tools (ECF 6.2.1 Chapter 14) ====================

	// -- Top-5 resource usage (cpu.util / memory.util) --
	case "get_resource_top5":
		metric := getString("metric")
		if metric == "" {
			metric = "cpu.util"
		}
		top5URL := fmt.Sprintf("%s/v2/extension/resources/top5/%s", serviceURL, metric)
		if s := getString("start"); s != "" {
			top5URL += "?start=" + s
		}
		if s := getString("end"); s != "" {
			if strings.Contains(top5URL, "?") {
				top5URL += "&end=" + s
			} else {
				top5URL += "?end=" + s
			}
		}
		result, err = doReq("GET", top5URL, nil)

	// -- Resource monitoring data (time-series for a specific resource + metric) --
	case "get_resource_metric_data":
		resourceID := getString("resource_id")
		metricName := getString("metric_name")
		startTime := getString("start_time")
		stopTime := getString("stop_time")
		granularity := getString("granularity")
		if granularity == "" {
			granularity = "300"
		}
		if startTime == "" {
			startTime = time.Now().Add(-1 * time.Hour).UTC().Format("2006-01-02T15:04:05")
		}
		if stopTime == "" {
			stopTime = time.Now().UTC().Format("2006-01-02T15:04:05")
		}
		metricURL := fmt.Sprintf("%s/v2/extension/metric_data/%s/start/%s/stop/%s/granularity/%s/resource/%s",
			serviceURL, metricName, startTime, stopTime, granularity, resourceID)
		result, err = doReq("GET", metricURL, nil)

	// -- Virtual resource alarms (Ceilometer-style) --
	case "list_resource_alarms":
		result, err = doReq("GET", fmt.Sprintf("%s/v2/alarms", serviceURL), nil)

	// -- Get specific alarm details --
	case "get_resource_alarm":
		result, err = doReq("GET", fmt.Sprintf("%s/v2/alarms/%s", serviceURL, getString("alarm_id")), nil)

	// -- Alarm history --
	case "get_alarm_history":
		result, err = doReq("GET", fmt.Sprintf("%s/v2/alarms/%s/history", serviceURL, getString("alarm_id")), nil)

	default:
		return fmt.Sprintf(`{"error":"unknown tool: %s"}`, toolName), nil
	}

	if err != nil {
		return fmt.Sprintf(`{"error":"%s"}`, err.Error()), nil
	}
	if result == nil {
		return `{"status":"success"}`, nil
	}
	resultStr := string(result)
	resultStr = smartTruncateResult(toolName, resultStr, 30000)
	return resultStr, nil
}

// preFilterAlerts extracts only essential fields from EMLA alert responses
// to drastically reduce token usage (original alert JSON can be >65KB).
// Keeps: id, alertNameCN, status, severity, startsAt, endsAt, group, summary
func preFilterAlerts(result json.RawMessage, err error) json.RawMessage {
	if err != nil || result == nil {
		return result
	}

	// Try to parse the EMLA response format
	var resp map[string]interface{}
	if json.Unmarshal(result, &resp) != nil {
		return result
	}

	// Look for items array in data.items or directly
	var items []interface{}
	if data, ok := resp["data"].(map[string]interface{}); ok {
		if its, ok := data["items"].([]interface{}); ok {
			items = its
		}
	}
	if items == nil {
		if its, ok := resp["items"].([]interface{}); ok {
			items = its
		}
	}
	if items == nil {
		// Try as direct array
		var arr []interface{}
		if json.Unmarshal(result, &arr) == nil {
			items = arr
		}
	}

	if items == nil {
		return result // Cannot parse, return as-is
	}

	// Extract only essential fields per alert
	keepFields := map[string]bool{
		"id": true, "alertNameCN": true, "alertname": true, "alertNameEN": true,
		"status": true, "severity": true, "state": true,
		"startsAt": true, "endsAt": true, "starts_at": true, "ends_at": true,
		"group": true, "category": true, "summary": true, "description": true,
		"resource": true, "rule": true, "namespace": true, "service": true, "instance": true,
	}

	compacted := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]interface{}); ok {
			slim := make(map[string]interface{})
			for k, v := range m {
				if keepFields[k] {
					slim[k] = v
				}
				// Also extract from nested labels/annotations
				if k == "labels" || k == "annotations" {
					if nested, ok := v.(map[string]interface{}); ok {
						for nk, nv := range nested {
							if keepFields[nk] {
								slim[nk] = nv
							}
						}
					}
				}
			}
			compacted = append(compacted, slim)
		}
	}

	// Rebuild response with compacted items + statistics
	output := map[string]interface{}{
		"_total_alerts": len(compacted),
		"alerts":        compacted,
	}
	// Copy statistics if present
	if data, ok := resp["data"].(map[string]interface{}); ok {
		if stats, ok := data["statistics"]; ok {
			output["statistics"] = stats
		}
	}
	if stats, ok := resp["statistics"]; ok {
		output["statistics"] = stats
	}

	result, _ = json.Marshal(output)
	logger.Log.Infof("[PreFilter] Alerts compacted: %d items, output size: %d bytes", len(compacted), len(result))
	return result
}

// smartTruncateResult intelligently truncates large API results.
// For JSON arrays, it keeps all items but summarizes each to key fields.
// For other large results, it truncates with a summary header.
func smartTruncateResult(toolName string, resultStr string, maxBytes int) string {
	if len(resultStr) <= maxBytes {
		return resultStr
	}

	logger.Log.Warnf("[SkillExecutor] Tool '%s' result too large: %d bytes (limit %d). Applying smart truncation.", toolName, len(resultStr), maxBytes)

	// Try to parse as JSON and intelligently reduce
	var raw interface{}
	if err := json.Unmarshal([]byte(resultStr), &raw); err == nil {
		compacted := smartCompactJSON(raw, toolName)
		compactedBytes, _ := json.Marshal(compacted)
		compactedStr := string(compactedBytes)
		if len(compactedStr) <= maxBytes {
			logger.Log.Infof("[SkillExecutor] Smart compaction reduced '%s' result: %d → %d bytes", toolName, len(resultStr), len(compactedStr))
			return compactedStr
		}
		// Still too large after compaction, truncate the compacted version
		resultStr = compactedStr
	}

	// Final fallback: hard truncate with metadata
	truncated := resultStr[:maxBytes]
	summary := fmt.Sprintf("\n...[数据已截断: 原始 %d 字节, 显示前 %d 字节。请使用更精确的查询条件缩小范围]", len(resultStr), maxBytes)
	return truncated + summary
}

// smartCompactJSON reduces JSON data size by keeping essential fields
// for known resource types (servers, volumes, alerts, etc.)
func smartCompactJSON(data interface{}, toolName string) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		// Look for common OpenStack list patterns: {"servers": [...], "volumes": [...], etc.}
		for key, val := range v {
			if arr, ok := val.([]interface{}); ok && len(arr) > 0 {
				compactedArr := compactResourceArray(arr, key, toolName)
				result := make(map[string]interface{})
				result[key] = compactedArr
				result["_total_count"] = len(arr)
				// Copy non-array metadata fields (pagination links, etc.)
				for k, kv := range v {
					if k != key {
						result[k] = kv
					}
				}
				return result
			}
		}
		return v
	case []interface{}:
		return compactResourceArray(v, "", toolName)
	default:
		return v
	}
}

// compactResourceArray removes verbose nested fields from resource arrays
func compactResourceArray(arr []interface{}, resourceKey, toolName string) []interface{} {
	// Fields to keep for common resource types
	keepFields := map[string][]string{
		"servers":       {"id", "name", "status", "OS-EXT-STS:vm_state", "OS-EXT-STS:power_state", "created", "updated", "tenant_id", "hostId", "flavor", "addresses", "metadata"},
		"volumes":       {"id", "name", "status", "size", "volume_type", "created_at", "availability_zone", "attachments", "bootable"},
		"networks":      {"id", "name", "status", "subnets", "provider:network_type", "shared", "router:external"},
		"subnets":       {"id", "name", "cidr", "gateway_ip", "network_id", "enable_dhcp", "ip_version"},
		"routers":       {"id", "name", "status", "external_gateway_info"},
		"floatingips":   {"id", "floating_ip_address", "fixed_ip_address", "status", "port_id", "router_id"},
		"security_groups": {"id", "name", "description", "security_group_rules"},
		"loadbalancers": {"id", "name", "vip_address", "operating_status", "provisioning_status", "provider"},
		"images":        {"id", "name", "status", "size", "min_disk", "min_ram", "created_at"},
		"flavors":       {"id", "name", "vcpus", "ram", "disk"},
	}

	// Determine which fields to keep
	fields, hasFilter := keepFields[resourceKey]
	if !hasFilter {
		// Try to infer from tool name
		switch {
		case strings.Contains(toolName, "server") || strings.Contains(toolName, "compute"):
			fields = keepFields["servers"]
			hasFilter = true
		case strings.Contains(toolName, "volume") || strings.Contains(toolName, "storage"):
			fields = keepFields["volumes"]
			hasFilter = true
		case strings.Contains(toolName, "network"):
			fields = keepFields["networks"]
			hasFilter = true
		case strings.Contains(toolName, "alert") || strings.Contains(toolName, "alarm"):
			// Keep all fields for alerts - they're usually small and important
			return arr
		}
	}

	if !hasFilter {
		return arr
	}

	fieldSet := make(map[string]bool)
	for _, f := range fields {
		fieldSet[f] = true
	}

	compacted := make([]interface{}, len(arr))
	for i, item := range arr {
		if m, ok := item.(map[string]interface{}); ok {
			slim := make(map[string]interface{})
			for k, v := range m {
				if fieldSet[k] {
					slim[k] = v
				}
			}
			compacted[i] = slim
		} else {
			compacted[i] = item
		}
	}
	return compacted
}

// executeZStack runs a tool against a ZStack cloud platform.
func (se *SkillExecutor) executeZStack(p model.CloudPlatform, sessionID, toolName string, params map[string]interface{}, getString func(string) string, getInt func(string) int) (string, error) {
	endpoint := strings.TrimRight(p.Endpoint, "/")
	httpClient := se.getHTTPClient(p)

	doZStackQuery := func(apiPath string, conditions []map[string]interface{}) (json.RawMessage, error) {
		url := fmt.Sprintf("%s%s", endpoint, apiPath)
		reqBody := map[string]interface{}{"sessionId": sessionID}
		if conditions != nil {
			reqBody["conditions"] = conditions
		}
		b, _ := json.Marshal(reqBody)
		req, err := http.NewRequest("GET", url, bytes.NewBuffer(b))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("OAuth %s", sessionID))
		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
		}
		return body, nil
	}

	var result json.RawMessage
	var err error

	switch toolName {
	case "list_servers":
		result, err = doZStackQuery("/v1/vm-instances", nil)
	case "get_server":
		result, err = doZStackQuery(fmt.Sprintf("/v1/vm-instances/%s", getString("server_id")), nil)
	case "list_volumes":
		result, err = doZStackQuery("/v1/volumes", nil)
	case "list_networks":
		result, err = doZStackQuery("/v1/l3-networks", nil)
	case "list_images":
		result, err = doZStackQuery("/v1/images", nil)
	case "list_security_groups":
		result, err = doZStackQuery("/v1/security-groups", nil)
	case "list_alerts":
		result, err = doZStackQuery("/v1/alarms", nil)
	default:
		// ZStack may not support all tools; return helpful message
		return fmt.Sprintf(`{"info":"Tool '%s' is not supported on ZStack platform. Supported: list_servers, list_volumes, list_networks, list_images, list_security_groups, list_alerts"}`, toolName), nil
	}

	if err != nil {
		return fmt.Sprintf(`{"error":"%s"}`, err.Error()), nil
	}
	if result == nil {
		return `{"status":"success"}`, nil
	}
	resultStr := string(result)
	resultStr = smartTruncateResult(toolName, resultStr, 30000)
	return resultStr, nil
}

// BuildToolsForSkills collects all OpenAI-compatible tool definitions from the given skills.
// It parses the ToolDefs JSON from each skill and returns a de-duplicated list.
func BuildToolsForSkills(skills []model.Skill) []ToolDefinition {
	seen := make(map[string]bool)
	var tools []ToolDefinition
	for _, skill := range skills {
		if skill.ToolDefs == "" {
			continue
		}
		var defs []ToolDefinition
		if err := json.Unmarshal([]byte(skill.ToolDefs), &defs); err != nil {
			logger.Log.Warnf("Failed to parse ToolDefs for skill '%s': %v", skill.Name, err)
			continue
		}
		for _, d := range defs {
			// Extract function name for dedup
			var fn struct {
				Function struct {
					Name string `json:"name"`
				} `json:"function"`
			}
			raw, _ := json.Marshal(d)
			json.Unmarshal(raw, &fn)
			if fn.Function.Name != "" && !seen[fn.Function.Name] {
				seen[fn.Function.Name] = true
				tools = append(tools, d)
			}
		}
	}
	// Sort for deterministic order
	sort.Slice(tools, func(i, j int) bool {
		ni, _ := json.Marshal(tools[i])
		nj, _ := json.Marshal(tools[j])
		return string(ni) < string(nj)
	})
	return tools
}
