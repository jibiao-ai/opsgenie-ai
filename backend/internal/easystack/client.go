package easystack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jibiao-ai/opsgenie-ai/internal/config"
	"github.com/jibiao-ai/opsgenie-ai/pkg/logger"
)

// Deprecated: Client is the legacy EasyStack API client.
// New code should use agent.SkillExecutor which supports multi-domain endpoint
// resolution (HostIP + BaseDomain) and per-platform HTTP clients with custom DNS.
// This client is retained only for backward compatibility and will be removed in a future version.
type Client struct {
	cfg        config.EasyStackConfig
	httpClient *http.Client
	token      string
	tokenExp   time.Time
	mu         sync.RWMutex
}

// NewClient creates a new EasyStack client
func NewClient(cfg config.EasyStackConfig) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Authenticate gets a project-scoped token from Keystone
func (c *Client) Authenticate() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	authReq := map[string]interface{}{
		"auth": map[string]interface{}{
			"identity": map[string]interface{}{
				"methods": []string{"password"},
				"password": map[string]interface{}{
					"user": map[string]interface{}{
						"name":     c.cfg.Username,
						"password": c.cfg.Password,
						"domain":   map[string]string{"name": c.cfg.DomainName},
					},
				},
			},
			"scope": map[string]interface{}{
				"project": map[string]interface{}{
					"name":   c.cfg.ProjectName,
					"domain": map[string]string{"name": c.cfg.DomainName},
				},
			},
		},
	}

	body, _ := json.Marshal(authReq)
	url := fmt.Sprintf("%s/v3/auth/tokens", c.cfg.AuthURL)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create auth request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	c.token = resp.Header.Get("X-Subject-Token")
	c.tokenExp = time.Now().Add(23 * time.Hour) // tokens typically valid for 24h

	logger.Log.Info("EasyStack authentication successful")
	return nil
}

// getToken returns a valid token, refreshing if needed
func (c *Client) getToken() (string, error) {
	c.mu.RLock()
	if c.token != "" && time.Now().Before(c.tokenExp) {
		defer c.mu.RUnlock()
		return c.token, nil
	}
	c.mu.RUnlock()

	if err := c.Authenticate(); err != nil {
		return "", err
	}
	return c.token, nil
}

// doRequest performs an authenticated HTTP request to EasyStack API
func (c *Client) doRequest(method, url string, body interface{}) ([]byte, int, error) {
	token, err := c.getToken()
	if err != nil {
		return nil, 0, fmt.Errorf("get token failed: %w", err)
	}

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal request body failed: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token)
	if c.cfg.ProjectID != "" {
		req.Header.Set("X-Project-Id", c.cfg.ProjectID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response failed: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

// ==================== Compute (Nova) APIs ====================

// ListServers lists all servers - GET /v2.1/{project_id}/servers/detail
func (c *Client) ListServers() (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2.1/%s/servers/detail", c.cfg.AuthURL, c.cfg.ProjectID)
	body, status, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("list servers failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// GetServer gets a specific server - GET /v2.1/{project_id}/servers/{server_id}
func (c *Client) GetServer(serverID string) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2.1/%s/servers/%s", c.cfg.AuthURL, c.cfg.ProjectID, serverID)
	body, status, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("get server failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// CreateServer creates a new server - POST /v2.1/{project_id}/servers
func (c *Client) CreateServer(params map[string]interface{}) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2.1/%s/servers", c.cfg.AuthURL, c.cfg.ProjectID)
	reqBody := map[string]interface{}{"server": params}
	body, status, err := c.doRequest("POST", url, reqBody)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("create server failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// ServerAction performs an action on a server - POST /v2.1/{project_id}/servers/{server_id}/action
func (c *Client) ServerAction(serverID string, action map[string]interface{}) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2.1/%s/servers/%s/action", c.cfg.AuthURL, c.cfg.ProjectID, serverID)
	body, status, err := c.doRequest("POST", url, action)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("server action failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// StartServer starts a server
func (c *Client) StartServer(serverID string) error {
	_, err := c.ServerAction(serverID, map[string]interface{}{"os-start": nil})
	return err
}

// StopServer stops a server
func (c *Client) StopServer(serverID string) error {
	_, err := c.ServerAction(serverID, map[string]interface{}{"os-stop": nil})
	return err
}

// RebootServer reboots a server
func (c *Client) RebootServer(serverID string, rebootType string) error {
	if rebootType == "" {
		rebootType = "SOFT"
	}
	_, err := c.ServerAction(serverID, map[string]interface{}{
		"reboot": map[string]string{"type": rebootType},
	})
	return err
}

// ResizeServer resizes a server
func (c *Client) ResizeServer(serverID, flavorRef string) error {
	_, err := c.ServerAction(serverID, map[string]interface{}{
		"resize": map[string]string{"flavorRef": flavorRef},
	})
	return err
}

// DeleteServer deletes a server - DELETE /v2.1/{project_id}/servers/{server_id}
func (c *Client) DeleteServer(serverID string) error {
	url := fmt.Sprintf("%s/v2.1/%s/servers/%s", c.cfg.AuthURL, c.cfg.ProjectID, serverID)
	_, status, err := c.doRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("delete server failed (HTTP %d)", status)
	}
	return nil
}

// CreateServerSnapshot creates a snapshot of a server
func (c *Client) CreateServerSnapshot(serverID, name string) (json.RawMessage, error) {
	return c.ServerAction(serverID, map[string]interface{}{
		"createImage": map[string]string{"name": name},
	})
}

// AttachVolume attaches a volume to a server - POST /v2.1/{project_id}/servers/{server_id}/os-volume_attachments
func (c *Client) AttachVolume(serverID, volumeID, device string) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2.1/%s/servers/%s/os-volume_attachments", c.cfg.AuthURL, c.cfg.ProjectID, serverID)
	reqBody := map[string]interface{}{
		"volumeAttachment": map[string]string{
			"volumeId": volumeID,
			"device":   device,
		},
	}
	body, status, err := c.doRequest("POST", url, reqBody)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("attach volume failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// DetachVolume detaches a volume from a server
func (c *Client) DetachVolume(serverID, volumeID string) error {
	url := fmt.Sprintf("%s/v2.1/%s/servers/%s/os-volume_attachments/%s",
		c.cfg.AuthURL, c.cfg.ProjectID, serverID, volumeID)
	_, status, err := c.doRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("detach volume failed (HTTP %d)", status)
	}
	return nil
}

// ListFlavors lists all flavors - GET /v2.1/{project_id}/flavors/detail
func (c *Client) ListFlavors() (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2.1/%s/flavors/detail", c.cfg.AuthURL, c.cfg.ProjectID)
	body, status, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("list flavors failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// ListKeypairs lists all keypairs - GET /v2.1/{project_id}/os-keypairs
func (c *Client) ListKeypairs() (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2.1/%s/os-keypairs", c.cfg.AuthURL, c.cfg.ProjectID)
	body, status, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("list keypairs failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// ==================== Block Storage (Cinder) APIs ====================

// ListVolumes lists all volumes - GET /v2/{project_id}/volumes
func (c *Client) ListVolumes() (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2/%s/volumes", c.cfg.AuthURL, c.cfg.ProjectID)
	body, status, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("list volumes failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// CreateVolume creates a volume - POST /v2/{project_id}/volumes
func (c *Client) CreateVolume(params map[string]interface{}) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2/%s/volumes", c.cfg.AuthURL, c.cfg.ProjectID)
	reqBody := map[string]interface{}{"volume": params}
	body, status, err := c.doRequest("POST", url, reqBody)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("create volume failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// DeleteVolume deletes a volume - DELETE /v2/{project_id}/volumes/{volume_id}
func (c *Client) DeleteVolume(volumeID string) error {
	url := fmt.Sprintf("%s/v2/%s/volumes/%s", c.cfg.AuthURL, c.cfg.ProjectID, volumeID)
	_, status, err := c.doRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("delete volume failed (HTTP %d)", status)
	}
	return nil
}

// ExtendVolume extends a volume - POST /v2/{project_id}/volumes/{volume_id}/action
func (c *Client) ExtendVolume(volumeID string, newSize int) error {
	url := fmt.Sprintf("%s/v2/%s/volumes/%s/action", c.cfg.AuthURL, c.cfg.ProjectID, volumeID)
	reqBody := map[string]interface{}{
		"os-extend": map[string]int{"new_size": newSize},
	}
	_, status, err := c.doRequest("POST", url, reqBody)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("extend volume failed (HTTP %d)", status)
	}
	return nil
}

// ListVolumeSnapshots lists volume snapshots - GET /v2/{project_id}/snapshots/detail
func (c *Client) ListVolumeSnapshots() (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2/%s/snapshots/detail", c.cfg.AuthURL, c.cfg.ProjectID)
	body, status, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("list snapshots failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// CreateVolumeSnapshot creates a volume snapshot - POST /v2/{project_id}/snapshots
func (c *Client) CreateVolumeSnapshot(volumeID, name string, force bool) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2/%s/snapshots", c.cfg.AuthURL, c.cfg.ProjectID)
	reqBody := map[string]interface{}{
		"snapshot": map[string]interface{}{
			"volume_id": volumeID,
			"name":      name,
			"force":     force,
		},
	}
	body, status, err := c.doRequest("POST", url, reqBody)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("create snapshot failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// ==================== Network (Neutron SDN) APIs ====================

// ListNetworks lists networks - GET /v2.0/networks
func (c *Client) ListNetworks() (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2.0/networks", c.cfg.AuthURL)
	body, status, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("list networks failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// CreateNetwork creates a network - POST /v2.0/networks
func (c *Client) CreateNetwork(name string) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2.0/networks", c.cfg.AuthURL)
	reqBody := map[string]interface{}{
		"network": map[string]string{"name": name},
	}
	body, status, err := c.doRequest("POST", url, reqBody)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("create network failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// ListSubnets lists subnets - GET /v2.0/subnets
func (c *Client) ListSubnets() (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2.0/subnets", c.cfg.AuthURL)
	body, status, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("list subnets failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// CreateSubnet creates a subnet - POST /v2.0/subnets
func (c *Client) CreateSubnet(params map[string]interface{}) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2.0/subnets", c.cfg.AuthURL)
	reqBody := map[string]interface{}{"subnet": params}
	body, status, err := c.doRequest("POST", url, reqBody)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("create subnet failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// ListRouters lists routers - GET /v2.0/routers
func (c *Client) ListRouters() (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2.0/routers", c.cfg.AuthURL)
	body, status, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("list routers failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// CreateRouter creates a router - POST /v2.0/routers
func (c *Client) CreateRouter(name string) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2.0/routers", c.cfg.AuthURL)
	reqBody := map[string]interface{}{
		"router": map[string]interface{}{
			"name":           name,
			"admin_state_up": true,
		},
	}
	body, status, err := c.doRequest("POST", url, reqBody)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("create router failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// AddRouterInterface adds an interface to a router - PUT /v2.0/routers/{router_id}/add_router_interface
func (c *Client) AddRouterInterface(routerID, subnetID string) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2.0/routers/%s/add_router_interface", c.cfg.AuthURL, routerID)
	reqBody := map[string]string{"subnet_id": subnetID}
	body, status, err := c.doRequest("PUT", url, reqBody)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("add router interface failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// ListFloatingIPs lists floating IPs - GET /v2.0/floatingips
func (c *Client) ListFloatingIPs() (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2.0/floatingips", c.cfg.AuthURL)
	body, status, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("list floating IPs failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// CreateFloatingIP creates a floating IP - POST /v2.0/floatingips
func (c *Client) CreateFloatingIP(networkID string) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2.0/floatingips", c.cfg.AuthURL)
	reqBody := map[string]interface{}{
		"floatingip": map[string]string{"floating_network_id": networkID},
	}
	body, status, err := c.doRequest("POST", url, reqBody)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("create floating IP failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// BindFloatingIP binds or unbinds a floating IP - PUT /v2.0/floatingips/{floatingip_id}
func (c *Client) BindFloatingIP(floatingIPID string, portID *string) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2.0/floatingips/%s", c.cfg.AuthURL, floatingIPID)
	reqBody := map[string]interface{}{
		"floatingip": map[string]interface{}{"port_id": portID},
	}
	body, status, err := c.doRequest("PUT", url, reqBody)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("bind floating IP failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// ListSecurityGroups lists security groups - GET /v2.0/security-groups
func (c *Client) ListSecurityGroups() (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2.0/security-groups", c.cfg.AuthURL)
	body, status, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("list security groups failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// CreateSecurityGroup creates a security group - POST /v2.0/security-groups
func (c *Client) CreateSecurityGroup(name, description string) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2.0/security-groups", c.cfg.AuthURL)
	reqBody := map[string]interface{}{
		"security_group": map[string]string{
			"name":        name,
			"description": description,
		},
	}
	body, status, err := c.doRequest("POST", url, reqBody)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("create security group failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// CreateSecurityGroupRule creates a security group rule - POST /v2.0/security-group-rules
func (c *Client) CreateSecurityGroupRule(params map[string]interface{}) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2.0/security-group-rules", c.cfg.AuthURL)
	reqBody := map[string]interface{}{"security_group_rule": params}
	body, status, err := c.doRequest("POST", url, reqBody)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("create security group rule failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// ListPorts lists ports - GET /v2.0/ports
func (c *Client) ListPorts() (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2.0/ports", c.cfg.AuthURL)
	body, status, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("list ports failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// ==================== Load Balancer (Octavia) APIs ====================

// ListLoadBalancers lists load balancers - GET /v2.0/lbaas/loadbalancers
func (c *Client) ListLoadBalancers() (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2.0/lbaas/loadbalancers", c.cfg.AuthURL)
	body, status, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("list load balancers failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// CreateLoadBalancer creates a load balancer - POST /v2.0/lbaas/loadbalancers
func (c *Client) CreateLoadBalancer(params map[string]interface{}) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2.0/lbaas/loadbalancers", c.cfg.AuthURL)
	reqBody := map[string]interface{}{"loadbalancer": params}
	body, status, err := c.doRequest("POST", url, reqBody)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("create LB failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// ListListeners lists listeners - GET /v2.0/lbaas/listeners
func (c *Client) ListListeners() (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2.0/lbaas/listeners", c.cfg.AuthURL)
	body, status, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("list listeners failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// ListPools lists pools - GET /v2.0/lbaas/pools
func (c *Client) ListPools() (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2.0/lbaas/pools", c.cfg.AuthURL)
	body, status, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("list pools failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// ==================== Monitoring (ECMS) APIs ====================

// QueryMetrics queries metrics - POST /api/ecms/{project_id}/metrics/query
func (c *Client) QueryMetrics(expr string, start, end, step int64) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/api/ecms/%s/metrics/query", c.cfg.AuthURL, c.cfg.ProjectID)
	reqBody := map[string]interface{}{
		"expr":  expr,
		"start": start,
		"end":   end,
		"step":  step,
	}
	body, status, err := c.doRequest("POST", url, reqBody)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("query metrics failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// ListAlerts lists alerts from the EasyStack ECMS alert API.
// Uses GET /apis/monitoring/v1/ecms/alerts (Prometheus-compatible ECMS endpoint).
// Optional query parameters: alerts_status (unresolved/resolved), severity (critical/warning/info).
// Response format: { "code": 0, "data": { "statistics": {...}, "items": [...] } }
func (c *Client) ListAlerts(alertsStatus, category, severity string) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/apis/monitoring/v1/ecms/alerts", c.cfg.AuthURL)
	var params []string
	if alertsStatus != "" {
		params = append(params, "alerts_status="+alertsStatus)
	}
	if severity != "" {
		params = append(params, "severity="+severity)
	}
	if len(params) > 0 {
		url += "?" + strings.Join(params, "&")
	}
	body, status, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("list alerts failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// ==================== Image (Glance) APIs ====================

// ListImages lists images - GET /v2/images
func (c *Client) ListImages() (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v2/images", c.cfg.AuthURL)
	body, status, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("list images failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// ==================== Quota APIs ====================

// GetQuotas gets domain quotas - GET /v1/quotas/domains/{domain_id}
func (c *Client) GetQuotas(domainID string) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/v1/quotas/domains/%s", c.cfg.AuthURL, domainID)
	body, status, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("get quotas failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}

// ExecuteAPICall is a generic method for the agent to call any EasyStack API
func (c *Client) ExecuteAPICall(method, path string, reqBody interface{}) (json.RawMessage, error) {
	url := fmt.Sprintf("%s%s", c.cfg.AuthURL, path)
	body, status, err := c.doRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("API call failed (HTTP %d): %s", status, string(body))
	}
	return body, nil
}
