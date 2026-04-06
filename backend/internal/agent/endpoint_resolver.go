package agent

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/jibiao-ai/opsgenie-ai/internal/model"
	"github.com/jibiao-ai/opsgenie-ai/pkg/logger"
)

// EasyStackServiceEndpoints holds resolved URLs for each EasyStack (OpenStack) service.
// Services are derived from HostIP + BaseDomain, e.g. for BaseDomain "opsl2.svc.cluster.local":
//
//	keystone → https://keystone.opsl2.svc.cluster.local
//	nova     → https://nova.opsl2.svc.cluster.local
//	cinder   → https://cinder.opsl2.svc.cluster.local
//	neutron  → https://neutron.opsl2.svc.cluster.local
//	glance   → https://glance.opsl2.svc.cluster.local
//	emla     → https://emla.opsl2.svc.cluster.local   (monitoring/observability)
//	heat     → https://heat.opsl2.svc.cluster.local
//	octavia  → https://octavia.opsl2.svc.cluster.local
type EasyStackServiceEndpoints struct {
	Keystone string // Identity (auth)
	Nova     string // Compute
	Cinder   string // Block Storage
	Neutron  string // Networking (SDN)
	Glance   string // Image
	Emla     string // Monitoring / Observability (ECMS/EMLA)
	Heat     string // Orchestration
	Octavia  string // Load Balancer
	Ceilometer string // Metering / Telemetry
}

// ResolveEasyStackEndpoints builds per-service URLs from a CloudPlatform.
// Priority:
//  1. HostIP + BaseDomain → construct "https://<service>.<base_domain>"
//  2. Fall back to AuthURL for all services (legacy single-URL mode)
func ResolveEasyStackEndpoints(p model.CloudPlatform) EasyStackServiceEndpoints {
	// If HostIP + BaseDomain are configured, build multi-domain endpoints
	if p.HostIP != "" && p.BaseDomain != "" {
		bd := strings.TrimLeft(p.BaseDomain, ".")
		ep := EasyStackServiceEndpoints{
			Keystone:   fmt.Sprintf("https://keystone.%s", bd),
			Nova:       fmt.Sprintf("https://nova.%s", bd),
			Cinder:     fmt.Sprintf("https://cinder.%s", bd),
			Neutron:    fmt.Sprintf("https://neutron.%s", bd),
			Glance:     fmt.Sprintf("https://glance.%s", bd),
			Emla:       fmt.Sprintf("https://emla.%s", bd),
			Heat:       fmt.Sprintf("https://heat.%s", bd),
			Octavia:    fmt.Sprintf("https://octavia.%s", bd),
			Ceilometer: fmt.Sprintf("https://ceilometer.%s", bd),
		}
		logger.Log.Infof("[EndpointResolver] EasyStack multi-domain mode: HostIP=%s, BaseDomain=%s → keystone=%s, emla=%s, nova=%s",
			p.HostIP, p.BaseDomain, ep.Keystone, ep.Emla, ep.Nova)
		return ep
	}

	// Legacy single-URL mode: all services via AuthURL
	base := strings.TrimRight(p.AuthURL, "/")
	logger.Log.Warnf("[EndpointResolver] EasyStack single-URL mode (legacy): all services via %s. Set HostIP + BaseDomain for multi-domain resolution.", base)
	return EasyStackServiceEndpoints{
		Keystone:   base,
		Nova:       base,
		Cinder:     base,
		Neutron:    base,
		Glance:     base,
		Emla:       base,
		Heat:       base,
		Octavia:    base,
		Ceilometer: base,
	}
}

// ServiceURLFor returns the appropriate base URL for a given tool name.
// This maps tool names to their corresponding OpenStack service endpoints.
func (ep *EasyStackServiceEndpoints) ServiceURLFor(toolName string) string {
	switch {
	// Compute (Nova)
	case strings.HasPrefix(toolName, "list_servers"),
		strings.HasPrefix(toolName, "get_server"),
		strings.HasPrefix(toolName, "create_server"),
		strings.HasPrefix(toolName, "start_server"),
		strings.HasPrefix(toolName, "stop_server"),
		strings.HasPrefix(toolName, "reboot_server"),
		strings.HasPrefix(toolName, "delete_server"),
		strings.HasPrefix(toolName, "list_flavors"):
		return ep.Nova

	// Image (Glance)
	case toolName == "list_images":
		return ep.Glance

	// Block Storage (Cinder)
	case strings.HasPrefix(toolName, "list_volume"),
		strings.HasPrefix(toolName, "create_volume"),
		strings.HasPrefix(toolName, "delete_volume"),
		strings.HasPrefix(toolName, "extend_volume"):
		return ep.Cinder

	// Network (Neutron)
	case strings.HasPrefix(toolName, "list_network"),
		strings.HasPrefix(toolName, "list_subnet"),
		strings.HasPrefix(toolName, "list_router"),
		strings.HasPrefix(toolName, "list_floating"),
		strings.HasPrefix(toolName, "list_security"),
		strings.HasPrefix(toolName, "create_security"),
		strings.HasPrefix(toolName, "list_port"):
		return ep.Neutron

	// Load Balancer (Octavia)
	case strings.HasPrefix(toolName, "list_loadbalancer"),
		strings.HasPrefix(toolName, "list_listener"),
		strings.HasPrefix(toolName, "list_pool"):
		return ep.Octavia

	// Monitoring / Observability (EMLA/ECMS)
	case toolName == "query_metrics",
		toolName == "query_metrics_range",
		toolName == "list_alerts",
		toolName == "list_active_alerts",
		toolName == "list_recovered_alerts",
		toolName == "get_alarm_severity_summary",
		toolName == "get_control_plane_status",
		toolName == "get_storage_cluster_status",
		toolName == "get_dashboard_overview",
		toolName == "check_all_services_health":
		return ep.Emla

	// Metering / Telemetry (Ceilometer)
	case toolName == "get_resource_top5",
		toolName == "get_resource_metric_data",
		toolName == "list_resource_alarms",
		toolName == "get_resource_alarm",
		toolName == "get_alarm_history":
		return ep.Ceilometer

	// Default: keystone (identity / quota / generic)
	default:
		return ep.Keystone
	}
}

// NewHTTPClientWithCustomDNS creates an HTTP client that resolves the BaseDomain
// services to the specified HostIP. This allows the Go program to reach
// internal K8s services without modifying /etc/hosts.
//
// For example, with HostIP=192.168.3.204 and BaseDomain=opsl2.svc.cluster.local:
//   - keystone.opsl2.svc.cluster.local → 192.168.3.204
//   - emla.opsl2.svc.cluster.local     → 192.168.3.204
//   - nova.opsl2.svc.cluster.local     → 192.168.3.204
func NewHTTPClientWithCustomDNS(hostIP, baseDomain string, timeout time.Duration) *http.Client {
	if hostIP == "" || baseDomain == "" {
		// No custom DNS needed
		return &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
	}

	bd := strings.TrimLeft(baseDomain, ".")

	dialer := &net.Dialer{Timeout: 10 * time.Second}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// addr is "host:port"
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				// addr might not have a port
				host = addr
				port = "443"
			}
			// If the host ends with our base domain, redirect to HostIP
			if strings.HasSuffix(host, bd) {
				logger.Log.Debugf("[CustomDNS] Resolving %s → %s:%s", addr, hostIP, port)
				addr = net.JoinHostPort(hostIP, port)
			}
			return dialer.DialContext(ctx, network, addr)
		},
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}
