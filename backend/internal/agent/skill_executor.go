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
	mu         sync.RWMutex
	tokens     map[uint]*platformToken // keyed by CloudPlatform.ID
	httpClient *http.Client
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
		tokens:     make(map[uint]*platformToken),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
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
		expiresAt: time.Now().Add(23 * time.Hour),
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
	url := fmt.Sprintf("%s/v3/auth/tokens", strings.TrimRight(p.AuthURL, "/"))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := se.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return resp.Header.Get("X-Subject-Token"), nil
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

	resp, err := se.httpClient.Do(req)
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

// ExecuteTool runs a tool call against the bound cloud platform.
// It determines the correct API endpoint based on the platform type and tool name.
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

	switch strings.ToLower(platform.Type) {
	case "easystack":
		return se.executeEasyStack(platform, token, toolName, params, getString, getInt)
	case "zstack":
		return se.executeZStack(platform, token, toolName, params, getString, getInt)
	default:
		return fmt.Sprintf(`{"error":"unsupported platform type: %s"}`, platform.Type), nil
	}
}

// executeEasyStack runs a tool against an EasyStack (OpenStack) cloud platform.
func (se *SkillExecutor) executeEasyStack(p model.CloudPlatform, token, toolName string, params map[string]interface{}, getString func(string) string, getInt func(string) int) (string, error) {
	baseURL := strings.TrimRight(p.AuthURL, "/")
	projectID := p.ProjectID

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
		req.Header.Set("X-Auth-Token", token)
		if projectID != "" {
			req.Header.Set("X-Project-Id", projectID)
		}
		resp, err := se.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
		}
		return respBody, nil
	}

	var result json.RawMessage
	var err error

	switch toolName {
	// Compute
	case "list_servers":
		result, err = doReq("GET", fmt.Sprintf("%s/v2.1/%s/servers/detail", baseURL, projectID), nil)
	case "get_server":
		result, err = doReq("GET", fmt.Sprintf("%s/v2.1/%s/servers/%s", baseURL, projectID, getString("server_id")), nil)
	case "create_server":
		result, err = doReq("POST", fmt.Sprintf("%s/v2.1/%s/servers", baseURL, projectID), map[string]interface{}{
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
		_, err = doReq("POST", fmt.Sprintf("%s/v2.1/%s/servers/%s/action", baseURL, projectID, getString("server_id")),
			map[string]interface{}{"os-start": nil})
		if err == nil {
			return `{"status":"success","message":"云主机启动命令已发送"}`, nil
		}
	case "stop_server":
		_, err = doReq("POST", fmt.Sprintf("%s/v2.1/%s/servers/%s/action", baseURL, projectID, getString("server_id")),
			map[string]interface{}{"os-stop": nil})
		if err == nil {
			return `{"status":"success","message":"云主机关闭命令已发送"}`, nil
		}
	case "reboot_server":
		rt := getString("type")
		if rt == "" {
			rt = "SOFT"
		}
		_, err = doReq("POST", fmt.Sprintf("%s/v2.1/%s/servers/%s/action", baseURL, projectID, getString("server_id")),
			map[string]interface{}{"reboot": map[string]string{"type": rt}})
		if err == nil {
			return `{"status":"success","message":"云主机重启命令已发送"}`, nil
		}
	case "delete_server":
		_, err = doReq("DELETE", fmt.Sprintf("%s/v2.1/%s/servers/%s", baseURL, projectID, getString("server_id")), nil)
		if err == nil {
			return `{"status":"success","message":"云主机删除命令已发送"}`, nil
		}
	case "list_flavors":
		result, err = doReq("GET", fmt.Sprintf("%s/v2.1/%s/flavors/detail", baseURL, projectID), nil)
	case "list_images":
		result, err = doReq("GET", fmt.Sprintf("%s/v2/images", baseURL), nil)
	// Storage
	case "list_volumes":
		result, err = doReq("GET", fmt.Sprintf("%s/v2/%s/volumes/detail", baseURL, projectID), nil)
	case "create_volume":
		volParams := map[string]interface{}{"name": getString("name"), "size": getInt("size")}
		result, err = doReq("POST", fmt.Sprintf("%s/v2/%s/volumes", baseURL, projectID), map[string]interface{}{"volume": volParams})
	case "delete_volume":
		_, err = doReq("DELETE", fmt.Sprintf("%s/v2/%s/volumes/%s", baseURL, projectID, getString("volume_id")), nil)
		if err == nil {
			return `{"status":"success","message":"云硬盘删除命令已发送"}`, nil
		}
	case "extend_volume":
		_, err = doReq("POST", fmt.Sprintf("%s/v3/%s/volumes/%s/action", baseURL, projectID, getString("volume_id")),
			map[string]interface{}{"os-extend": map[string]int{"new_size": getInt("new_size")}})
		if err == nil {
			return `{"status":"success","message":"云硬盘扩容命令已发送"}`, nil
		}
	case "list_volume_snapshots":
		result, err = doReq("GET", fmt.Sprintf("%s/v2/%s/snapshots/detail", baseURL, projectID), nil)
	// Network
	case "list_networks":
		result, err = doReq("GET", fmt.Sprintf("%s/v2.0/networks", baseURL), nil)
	case "list_subnets":
		result, err = doReq("GET", fmt.Sprintf("%s/v2.0/subnets", baseURL), nil)
	case "list_routers":
		result, err = doReq("GET", fmt.Sprintf("%s/v2.0/routers", baseURL), nil)
	case "list_floating_ips":
		result, err = doReq("GET", fmt.Sprintf("%s/v2.0/floatingips", baseURL), nil)
	case "list_security_groups":
		result, err = doReq("GET", fmt.Sprintf("%s/v2.0/security-groups", baseURL), nil)
	case "create_security_group":
		result, err = doReq("POST", fmt.Sprintf("%s/v2.0/security-groups", baseURL),
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
		result, err = doReq("POST", fmt.Sprintf("%s/v2.0/security-group-rules", baseURL),
			map[string]interface{}{"security_group_rule": ruleParams})
	// LB
	case "list_loadbalancers":
		result, err = doReq("GET", fmt.Sprintf("%s/v2.0/lbaas/loadbalancers", baseURL), nil)
	case "list_listeners":
		result, err = doReq("GET", fmt.Sprintf("%s/v2.0/lbaas/listeners", baseURL), nil)
	case "list_pools":
		result, err = doReq("GET", fmt.Sprintf("%s/v2.0/lbaas/pools", baseURL), nil)
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
		result, err = doReq("POST", fmt.Sprintf("%s/api/ecms/%s/metrics/query", baseURL, projectID),
			map[string]interface{}{"expr": getString("expr"), "start": start, "end": end, "step": step})
	case "list_alerts":
		alertURL := fmt.Sprintf("%s/v1/%s/alerts?all_tenants=true", baseURL, projectID)
		if s := getString("states"); s != "" {
			alertURL += "&states=" + s
		}
		if s := getString("severities"); s != "" {
			alertURL += "&severities=" + s
		}
		result, err = doReq("GET", alertURL, nil)

	// ==================== Monitoring Alarm Skill Tools (ECF 6.2.1 Observability) ====================

	// -- Active alarms (firing) --
	case "list_active_alerts":
		alertURL := fmt.Sprintf("%s/v1/%s/alerts?all_tenants=true&states=firing", baseURL, projectID)
		if s := getString("severities"); s != "" {
			alertURL += "&severities=" + s
		}
		if s := getString("categories"); s != "" {
			alertURL += "&categories=" + s
		}
		result, err = doReq("GET", alertURL, nil)

	// -- Recovered alarms (resolved) --
	case "list_recovered_alerts":
		alertURL := fmt.Sprintf("%s/v1/%s/alerts?all_tenants=true&states=resolved", baseURL, projectID)
		if s := getString("severities"); s != "" {
			alertURL += "&severities=" + s
		}
		if s := getString("categories"); s != "" {
			alertURL += "&categories=" + s
		}
		if s := getString("start"); s != "" {
			alertURL += "&start=" + s
		}
		if s := getString("end"); s != "" {
			alertURL += "&end=" + s
		}
		result, err = doReq("GET", alertURL, nil)

	// -- Alarm severity summary (critical/warning/info counts) --
	case "get_alarm_severity_summary":
		alertURL := fmt.Sprintf("%s/v1/%s/alerts?all_tenants=true", baseURL, projectID)
		result, err = doReq("GET", alertURL, nil)
		if err == nil && result != nil {
			// Parse and extract severity statistics
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
			}
			if json.Unmarshal(result, &alertResp) == nil {
				summary := map[string]interface{}{
					"total":    alertResp.Data.Statistics.Total,
					"critical": alertResp.Data.Statistics.Critical,
					"warning":  alertResp.Data.Statistics.Warning,
					"info":     alertResp.Data.Statistics.Info,
				}
				summaryJSON, _ := json.Marshal(summary)
				result = summaryJSON
			}
		}

	// -- Control plane service status (40+ service_*_state metrics) --
	case "get_control_plane_status":
		metricsFilter := []string{
			"service_control_api_state",
			"service_compute_api_state",
			"service_compute_conductor_state",
			"service_compute_scheduler_state",
			"service_network_api_state",
			"service_network_dhcp_state",
			"service_network_l3_state",
			"service_network_metadata_state",
			"service_network_lb_state",
			"service_storage_api_state",
			"service_storage_scheduler_state",
			"service_storage_volume_state",
			"service_image_api_state",
			"service_identity_api_state",
			"service_monitoring_api_state",
			"service_database_state",
			"service_mq_state",
			"service_orchestration_api_state",
			"service_baremetal_api_state",
			"service_container_api_state",
		}
		metricsReq := map[string]interface{}{
			"metrics_filter": metricsFilter,
			"time":           time.Now().Unix(),
		}
		result, err = doReq("POST", fmt.Sprintf("%s/api/ecms/control_plane/metrics/query", baseURL), metricsReq)

	// -- Storage cluster status --
	case "get_storage_cluster_status":
		storageMetrics := []string{
			"storage_health_status",
			"ceph_mon_quorum_status",
			"storage_osd_total",
			"storage_osd_up",
			"storage_osd_down",
			"storage_actual_capacity_total_bytes",
			"storage_actual_capacity_free_bytes",
			"storage_actual_capacity_used_bytes",
			"storage_user_data_pool_bytes",
			"storage_cluster_iops_read",
			"storage_cluster_iops_write",
			"storage_cluster_throughput_read",
			"storage_cluster_throughput_write",
		}
		metricsReq := map[string]interface{}{
			"metrics_filter": storageMetrics,
			"time":           time.Now().Unix(),
		}
		result, err = doReq("POST", fmt.Sprintf("%s/api/ecms/storage/metrics/query", baseURL), metricsReq)

	// -- Dashboard overview metrics (VM state, CPU, memory, storage, top5) --
	case "get_dashboard_overview":
		dashMetrics := []string{
			"dashboard_instances_state",
			"dashboard_instances_vcpu_usage",
			"dashboard_cpu_total",
			"dashboard_cpu_used",
			"dashboard_memory_total",
			"dashboard_memory_usage",
			"dashboard_storage_total",
			"dashboard_storage_used",
			"dashboard_cache_disk_total",
			"dashboard_cache_disk_used",
		}
		metricsReq := map[string]interface{}{
			"metrics_filter": dashMetrics,
			"time":           time.Now().Unix(),
		}
		result, err = doReq("POST", fmt.Sprintf("%s/api/ecms/%s/metrics/query", baseURL, projectID), metricsReq)

	// -- Query metrics range (PromQL with time range) --
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
		result, err = doReq("POST", fmt.Sprintf("%s/api/ecms/%s/metrics/query_range", baseURL, projectID),
			map[string]interface{}{"expr": getString("expr"), "start": start, "end": end, "step": step})

	// -- All cloud service health check (comprehensive) --
	case "check_all_services_health":
		// Query all known service state metrics from the control plane
		allServiceMetrics := []string{
			"service_control_api_state",
			"service_compute_api_state",
			"service_compute_conductor_state",
			"service_compute_scheduler_state",
			"service_network_api_state",
			"service_network_dhcp_state",
			"service_network_l3_state",
			"service_network_metadata_state",
			"service_network_lb_state",
			"service_storage_api_state",
			"service_storage_scheduler_state",
			"service_storage_volume_state",
			"service_image_api_state",
			"service_identity_api_state",
			"service_monitoring_api_state",
			"service_database_state",
			"service_mq_state",
			"service_orchestration_api_state",
			"service_baremetal_api_state",
			"service_container_api_state",
			"service_billing_api_state",
			"service_object_storage_api_state",
			"service_dns_api_state",
			"service_vpn_api_state",
			"service_firewall_api_state",
			"service_key_manager_api_state",
		}
		metricsReq := map[string]interface{}{
			"metrics_filter": allServiceMetrics,
			"time":           time.Now().Unix(),
		}
		result, err = doReq("POST", fmt.Sprintf("%s/api/ecms/control_plane/metrics/query", baseURL), metricsReq)

	// ==================== Metering Service Tools (ECF 6.2.1 Chapter 14) ====================

	// -- Top-5 resource usage (cpu.util / memory.util) --
	case "get_resource_top5":
		metric := getString("metric")
		if metric == "" {
			metric = "cpu.util"
		}
		top5URL := fmt.Sprintf("%s/v2/extension/resources/top5/%s", baseURL, metric)
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
			baseURL, metricName, startTime, stopTime, granularity, resourceID)
		result, err = doReq("GET", metricURL, nil)

	// -- Virtual resource alarms (Ceilometer-style) --
	case "list_resource_alarms":
		result, err = doReq("GET", fmt.Sprintf("%s/v2/alarms", baseURL), nil)

	// -- Get specific alarm details --
	case "get_resource_alarm":
		result, err = doReq("GET", fmt.Sprintf("%s/v2/alarms/%s", baseURL, getString("alarm_id")), nil)

	// -- Alarm history --
	case "get_alarm_history":
		result, err = doReq("GET", fmt.Sprintf("%s/v2/alarms/%s/history", baseURL, getString("alarm_id")), nil)

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
	if len(resultStr) > 8000 {
		logger.Log.Warnf("[SkillExecutor] Tool '%s' result truncated: %d bytes → 8000 bytes. Data loss may cause inaccurate AI responses.", toolName, len(resultStr))
		resultStr = resultStr[:8000] + "...(truncated)"
	}
	return resultStr, nil
}

// executeZStack runs a tool against a ZStack cloud platform.
func (se *SkillExecutor) executeZStack(p model.CloudPlatform, sessionID, toolName string, params map[string]interface{}, getString func(string) string, getInt func(string) int) (string, error) {
	endpoint := strings.TrimRight(p.Endpoint, "/")

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
		resp, err := se.httpClient.Do(req)
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
	if len(resultStr) > 8000 {
		resultStr = resultStr[:8000] + "...(truncated)"
	}
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
