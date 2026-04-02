package alb

import (
	"aliyun-cloud-monitor/internal/aliyuncli"
	"aliyun-cloud-monitor/internal/prom"
	"encoding/json"
	"fmt"
	"strings"
)

type Instance struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	DNSName string `json:"dnsName"`
	AddressType string `json:"addressType"`
}

// UsageData represents resource usage for an ALB instance
type UsageData struct {
	InstanceID        string  `json:"instanceId"`
	InstanceName      string  `json:"instanceName"`
	OutBits           float64 `json:"outBits"`           // 出带宽
	ActiveConnection  float64 `json:"activeConnection"`  // 活跃连接数
	QPS               float64 `json:"qps"`               // QPS
	HTTP4XX           float64 `json:"http4xx"`           // 4XX数量
	HTTP5XX           float64 `json:"http5xx"`           // 5XX数量
	ErrorRate         float64 `json:"errorRate"`         // 连接失败率 = (4xx + 5xx) / QPS
}

func List(region string, env map[string]string) ([]Instance, error) {
	args := []string{"alb", "ListLoadBalancers", "--RegionId", region}
	var data struct {
		LoadBalancers []struct {
			LoadBalancerId     string `json:"LoadBalancerId"`
			LoadBalancerName   string `json:"LoadBalancerName"`
			LoadBalancerStatus string `json:"LoadBalancerStatus"`
			DNSName            string `json:"DNSName"`
			AddressType        string `json:"AddressType"`
		} `json:"LoadBalancers"`
	}
	if err := aliyuncli.RunJSON(args, env, &data); err != nil {
		return nil, err
	}
	res := make([]Instance, 0, len(data.LoadBalancers))
	for _, it := range data.LoadBalancers {
		res = append(res, Instance{
			ID:          it.LoadBalancerId,
			Name:        it.LoadBalancerName,
			Status:      it.LoadBalancerStatus,
			DNSName:     it.DNSName,
			AddressType: it.AddressType,
		})
	}
	return res, nil
}

// Usage queries Prometheus for ALB resource usage (out bandwidth, active connections, QPS, error rate)
// Metric names:
// - Out bandwidth: AliyunAlb_LoadBalancerOutBits
// - Active connections: AliyunAlb_LoadBalancerActiveConnection
// - QPS: AliyunAlb_LoadBalancerQPS
// - 4XX: AliyunAlb_LoadBalancerHTTPCodeUpstream4XX
// - 5XX: AliyunAlb_LoadBalancerHTTPCodeUpstream5XX
// Error rate = (4xx + 5xx) / QPS, initial value is 0
func Usage(baseURL, user, pass string, datasource, timeoutSec int) ([]UsageData, error) {
	// Query 4XX first to get instance list (this metric usually has data)
	http4xxData, err := prom.Query(baseURL, user, pass, "AliyunAlb_LoadBalancerHTTPCodeUpstream4XX", datasource, timeoutSec)
	if err != nil { return nil, fmt.Errorf("query 4XX failed: %w", err) }
	
	// Query 5XX
	http5xxData, err := prom.Query(baseURL, user, pass, "AliyunAlb_LoadBalancerHTTPCodeUpstream5XX", datasource, timeoutSec)
	if err != nil { return nil, fmt.Errorf("query 5XX failed: %w", err) }
	
	// Query out bandwidth
	outBitsData, err := prom.Query(baseURL, user, pass, "AliyunAlb_LoadBalancerOutBits", datasource, timeoutSec)
	if err != nil { return nil, fmt.Errorf("query out bandwidth failed: %w", err) }
	
	// Query active connections
	activeConnData, err := prom.Query(baseURL, user, pass, "AliyunAlb_LoadBalancerActiveConnection", datasource, timeoutSec)
	if err != nil { return nil, fmt.Errorf("query active connections failed: %w", err) }
	
	// Query QPS
	qpsData, err := prom.Query(baseURL, user, pass, "AliyunAlb_LoadBalancerQPS", datasource, timeoutSec)
	if err != nil { return nil, fmt.Errorf("query QPS failed: %w", err) }
	
	// Aggregate by instance ID (use 4XX data to get instance list)
	usageMap := make(map[string]*UsageData)
	
	// Process 4XX data first to build instance list
	http4xxResults := getResults(http4xxData)
	for _, item := range http4xxResults {
		instanceID := getStringLabel(item, "loadBalancerId")
		if instanceID == "" { instanceID = getStringLabel(item, "id") }
		instanceName := getStringLabel(item, "loadBalancerName")
		if instanceID == "" { continue }
		if _, ok := usageMap[instanceID]; !ok {
			usageMap[instanceID] = &UsageData{InstanceID: instanceID, InstanceName: instanceName}
		}
		usageMap[instanceID].HTTP4XX += getFloatValue(item)
	}
	
	// If no instances from 4XX, try to get from outBits
	if len(usageMap) == 0 {
		outBitsResults := getResults(outBitsData)
		for _, item := range outBitsResults {
			instanceID := getStringLabel(item, "loadBalancerId")
			if instanceID == "" { instanceID = getStringLabel(item, "id") }
			instanceName := getStringLabel(item, "loadBalancerName")
			if instanceID == "" { continue }
			if _, ok := usageMap[instanceID]; !ok {
				usageMap[instanceID] = &UsageData{InstanceID: instanceID, InstanceName: instanceName}
			}
		}
	}
	
	// Process 5XX data
	http5xxResults := getResults(http5xxData)
	for _, item := range http5xxResults {
		instanceID := getStringLabel(item, "loadBalancerId")
		if instanceID == "" { instanceID = getStringLabel(item, "id") }
		if instanceID == "" { continue }
		if _, ok := usageMap[instanceID]; ok {
			usageMap[instanceID].HTTP5XX += getFloatValue(item)
		}
	}
	
	// Process out bandwidth data
	outBitsResults := getResults(outBitsData)
	for _, item := range outBitsResults {
		instanceID := getStringLabel(item, "loadBalancerId")
		if instanceID == "" { instanceID = getStringLabel(item, "id") }
		if instanceID == "" { continue }
		if _, ok := usageMap[instanceID]; ok {
			usageMap[instanceID].OutBits = getFloatValue(item)
		}
	}
	
	// Process active connections data
	activeConnResults := getResults(activeConnData)
	for _, item := range activeConnResults {
		instanceID := getStringLabel(item, "loadBalancerId")
		if instanceID == "" { instanceID = getStringLabel(item, "id") }
		if instanceID == "" { continue }
		if _, ok := usageMap[instanceID]; ok {
			usageMap[instanceID].ActiveConnection = getFloatValue(item)
		}
	}
	
	// Process QPS data
	qpsResults := getResults(qpsData)
	for _, item := range qpsResults {
		instanceID := getStringLabel(item, "loadBalancerId")
		if instanceID == "" { instanceID = getStringLabel(item, "id") }
		if instanceID == "" { continue }
		if _, ok := usageMap[instanceID]; ok {
			usageMap[instanceID].QPS += getFloatValue(item)
		}
	}
	
	// Calculate error rate: (4XX + 5XX) / QPS
	// If QPS is 0, error rate is 0 (initial value as per user request)
	for _, v := range usageMap {
		if v.QPS > 0 {
			v.ErrorRate = (v.HTTP4XX + v.HTTP5XX) / v.QPS * 100
		} else {
			v.ErrorRate = 0 // 初始值为0
		}
	}
	
	// Convert map to slice
	result := make([]UsageData, 0, len(usageMap))
	for _, v := range usageMap {
		result = append(result, *v)
	}
	
	return result, nil
}

func getResults(data map[string]any) []map[string]any {
	result := []map[string]any{}
	if d, ok := data["data"].(map[string]any); ok {
		if r, ok := d["result"].([]any); ok {
			for _, item := range r {
				if m, ok := item.(map[string]any); ok {
					result = append(result, m)
				}
			}
		}
	}
	return result
}

func getStringLabel(item map[string]any, key string) string {
	if labels, ok := item["metric"].(map[string]any); ok {
		if v, ok := labels[key].(string); ok {
			return v
		}
	}
	return ""
}

func getFloatValue(item map[string]any) float64 {
	if v, ok := item["value"].([]any); ok && len(v) == 2 {
		if f, ok := v[1].(float64); ok {
			return f
		}
		if s, ok := v[1].(string); ok {
			var f float64
			fmt.Sscanf(s, "%f", &f)
			return f
		}
	}
	return 0
}

func FormatInstances(items []Instance, output string) string {
	if output == "json" { b, _ := json.MarshalIndent(items, "", "  "); return string(b) }
	if len(items) == 0 { return "未找到 ALB 实例" }
	lines := []string{fmt.Sprintf("实例数量: %d", len(items)), ""}
	for _, it := range items {
		name := it.Name; if name == "" { name = it.ID }
		lines = append(lines, fmt.Sprintf("- %s", name))
		lines = append(lines, fmt.Sprintf("  ID: %s | DNS: %s", it.ID, it.DNSName))
		lines = append(lines, fmt.Sprintf("  类型: %s | 状态: %s", it.AddressType, it.Status))
	}
	return strings.Join(lines, "\n")
}

func FormatUsage(items []UsageData, output string) string {
	if output == "json" { b, _ := json.MarshalIndent(items, "", "  "); return string(b) }
	if len(items) == 0 { return "未找到 ALB 实例资源消耗数据" }
	lines := []string{fmt.Sprintf("实例数量: %d", len(items)), ""}
	for _, it := range items {
		name := it.InstanceName
		if name == "" { name = it.InstanceID }
		// 转换带宽单位：bits/s -> Mbps
		outMbps := it.OutBits / 1000000
		lines = append(lines, fmt.Sprintf("- %s (%s)", name, it.InstanceID))
		lines = append(lines, fmt.Sprintf("  出带宽: %.2f Mbps | 活跃连接: %.0f | QPS: %.2f | 连接失败率: %.2f%%", outMbps, it.ActiveConnection, it.QPS, it.ErrorRate))
	}
	return strings.Join(lines, "\n")
}

// ACL related structures and functions

type Acl struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	IPVersion   string `json:"ipVersion"`
}

type AclEntry struct {
	Entry       string `json:"entry"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type ListenerAclConfig struct {
	ListenerID     string `json:"listenerId"`
	ListenerPort   int    `json:"listenerPort"`
	ListenerProtocol string `json:"listenerProtocol"`
	LoadBalancerID string `json:"loadBalancerId"`
	AclID          string `json:"aclId"`
	AclType        string `json:"aclType"`
	AclStatus      string `json:"aclStatus"`
}

// ListAcls lists all ACLs in a region
func ListAcls(region string, env map[string]string) ([]Acl, error) {
	args := []string{"alb", "ListAcls", "--RegionId", region}
	var data struct {
		Acls []struct {
			AclId            string `json:"AclId"`
			AclName          string `json:"AclName"`
			AclStatus        string `json:"AclStatus"`
			AddressIPVersion string `json:"AddressIPVersion"`
		} `json:"Acls"`
	}
	if err := aliyuncli.RunJSON(args, env, &data); err != nil {
		return nil, err
	}
	res := make([]Acl, 0, len(data.Acls))
	for _, it := range data.Acls {
		res = append(res, Acl{
			ID:        it.AclId,
			Name:      it.AclName,
			Status:    it.AclStatus,
			IPVersion: it.AddressIPVersion,
		})
	}
	return res, nil
}

// ListAclEntries lists entries in an ACL
func ListAclEntries(region, aclID string, env map[string]string) ([]AclEntry, error) {
	args := []string{"alb", "ListAclEntries", "--RegionId", region, "--AclId", aclID}
	var data struct {
		AclEntries []struct {
			Entry       string `json:"Entry"`
			Description string `json:"Description"`
			Status      string `json:"Status"`
		} `json:"AclEntries"`
	}
	if err := aliyuncli.RunJSON(args, env, &data); err != nil {
		return nil, err
	}
	res := make([]AclEntry, 0, len(data.AclEntries))
	for _, it := range data.AclEntries {
		res = append(res, AclEntry{
			Entry:       it.Entry,
			Description: it.Description,
			Status:      it.Status,
		})
	}
	return res, nil
}

// GetListenerAcl gets ACL configuration for a listener
func GetListenerAcl(region, listenerID string, env map[string]string) (ListenerAclConfig, error) {
	args := []string{"alb", "GetListenerAttribute", "--RegionId", region, "--ListenerId", listenerID}
	var data struct {
		ListenerId       string `json:"ListenerId"`
		ListenerPort     int    `json:"ListenerPort"`
		ListenerProtocol string `json:"ListenerProtocol"`
		LoadBalancerId   string `json:"LoadBalancerId"`
		AclConfig        struct {
			AclType    string `json:"AclType"`
			AclRelations []struct {
				AclId  string `json:"AclId"`
				Status string `json:"Status"`
			} `json:"AclRelations"`
		} `json:"AclConfig"`
	}
	if err := aliyuncli.RunJSON(args, env, &data); err != nil {
		return ListenerAclConfig{}, err
	}
	
	result := ListenerAclConfig{
		ListenerID:       data.ListenerId,
		ListenerPort:     data.ListenerPort,
		ListenerProtocol: data.ListenerProtocol,
		LoadBalancerID:   data.LoadBalancerId,
	}
	
	if len(data.AclConfig.AclRelations) > 0 {
		result.AclID = data.AclConfig.AclRelations[0].AclId
		result.AclStatus = data.AclConfig.AclRelations[0].Status
		result.AclType = data.AclConfig.AclType
	}
	
	return result, nil
}

// ListListenersWithAcl lists all listeners with their ACL configuration
func ListListenersWithAcl(region string, env map[string]string) ([]ListenerAclConfig, error) {
	args := []string{"alb", "ListListeners", "--RegionId", region}
	var data struct {
		Listeners []struct {
			ListenerId       string `json:"ListenerId"`
			ListenerPort     int    `json:"ListenerPort"`
			ListenerProtocol string `json:"ListenerProtocol"`
			LoadBalancerId   string `json:"LoadBalancerId"`
		} `json:"Listeners"`
	}
	if err := aliyuncli.RunJSON(args, env, &data); err != nil {
		return nil, err
	}
	
	res := make([]ListenerAclConfig, 0, len(data.Listeners))
	for _, it := range data.Listeners {
		// Get ACL config for each listener
		aclConfig, err := GetListenerAcl(region, it.ListenerId, env)
		if err != nil {
			// If error, add listener without ACL info
			res = append(res, ListenerAclConfig{
				ListenerID:       it.ListenerId,
				ListenerPort:     it.ListenerPort,
				ListenerProtocol: it.ListenerProtocol,
				LoadBalancerID:   it.LoadBalancerId,
			})
			continue
		}
		res = append(res, aclConfig)
	}
	return res, nil
}

func FormatAcls(items []Acl, output string) string {
	if output == "json" { b, _ := json.MarshalIndent(items, "", "  "); return string(b) }
	if len(items) == 0 { return "未找到 ACL" }
	lines := []string{fmt.Sprintf("ACL 数量: %d", len(items)), ""}
	for _, it := range items {
		lines = append(lines, fmt.Sprintf("- %s (%s)", it.Name, it.ID))
		lines = append(lines, fmt.Sprintf("  状态: %s | IP版本: %s", it.Status, it.IPVersion))
	}
	return strings.Join(lines, "\n")
}

func FormatAclEntries(aclID string, items []AclEntry, output string) string {
	if output == "json" {
		data := map[string]any{"aclId": aclID, "entries": items}
		b, _ := json.MarshalIndent(data, "", "  ")
		return string(b)
	}
	if len(items) == 0 { return fmt.Sprintf("ACL %s 无条目", aclID) }
	lines := []string{fmt.Sprintf("ACL: %s", aclID), "", "条目列表:", ""}
	for _, it := range items {
		desc := it.Description
		if desc == "" { desc = "-" }
		lines = append(lines, fmt.Sprintf("- %s (%s) [%s]", it.Entry, desc, it.Status))
	}
	return strings.Join(lines, "\n")
}

func FormatListenersAcl(items []ListenerAclConfig, output string) string {
	if output == "json" { b, _ := json.MarshalIndent(items, "", "  "); return string(b) }
	if len(items) == 0 { return "未找到监听器" }
	lines := []string{fmt.Sprintf("监听器数量: %d", len(items)), ""}
	for _, it := range items {
		aclStatus := "未启用"
		if it.AclID != "" {
			aclStatus = fmt.Sprintf("%s (%s模式)", it.AclID, it.AclType)
		}
		lines = append(lines, fmt.Sprintf("- %s:%d (%s)", it.ListenerProtocol, it.ListenerPort, it.ListenerID))
		lines = append(lines, fmt.Sprintf("  ALB: %s | ACL: %s", it.LoadBalancerID, aclStatus))
	}
	return strings.Join(lines, "\n")
}

// ZoneMapping represents ALB zone mapping info
type ZoneMapping struct {
	LoadBalancerID   string `json:"loadBalancerId"`
	LoadBalancerName string `json:"loadBalancerName"`
	ZoneId           string `json:"zoneId"`
	VSwitchId        string `json:"vswitchId"`
	IntranetAddress  string `json:"intranetAddress"`
	EipAddress       string `json:"eipAddress"`
	Status           string `json:"status"`
}

// GetLoadBalancerZoneMappings gets zone mappings for a specific ALB
func GetLoadBalancerZoneMappings(region, loadBalancerId string, env map[string]string) ([]ZoneMapping, error) {
	args := []string{"alb", "GetLoadBalancerAttribute", "--RegionId", region, "--LoadBalancerId", loadBalancerId}
	var data struct {
		LoadBalancerId   string `json:"LoadBalancerId"`
		LoadBalancerName string `json:"LoadBalancerName"`
		ZoneMappings     []struct {
			ZoneId    string `json:"ZoneId"`
			VSwitchId string `json:"VSwitchId"`
			Status    string `json:"Status"`
			LoadBalancerAddresses []struct {
				Address         string `json:"Address"`
				AllocationId    string `json:"AllocationId"`
				IntranetAddress string `json:"IntranetAddress"`
			} `json:"LoadBalancerAddresses"`
		} `json:"ZoneMappings"`
	}
	if err := aliyuncli.RunJSON(args, env, &data); err != nil {
		return nil, err
	}
	res := make([]ZoneMapping, 0, len(data.ZoneMappings))
	for _, zm := range data.ZoneMappings {
		eipAddr := ""
		intranetAddr := ""
		if len(zm.LoadBalancerAddresses) > 0 {
			eipAddr = zm.LoadBalancerAddresses[0].Address
			intranetAddr = zm.LoadBalancerAddresses[0].IntranetAddress
		}
		res = append(res, ZoneMapping{
			LoadBalancerID:   data.LoadBalancerId,
			LoadBalancerName: data.LoadBalancerName,
			ZoneId:           zm.ZoneId,
			VSwitchId:        zm.VSwitchId,
			IntranetAddress:  intranetAddr,
			EipAddress:       eipAddr,
			Status:           zm.Status,
		})
	}
	return res, nil
}

// ListAllZoneMappings lists zone mappings for all ALBs in a region
func ListAllZoneMappings(region string, env map[string]string) ([]ZoneMapping, error) {
	instances, err := List(region, env)
	if err != nil {
		return nil, err
	}
	allMappings := []ZoneMapping{}
	for _, inst := range instances {
		mappings, err := GetLoadBalancerZoneMappings(region, inst.ID, env)
		if err != nil {
			continue
		}
		allMappings = append(allMappings, mappings...)
	}
	return allMappings, nil
}

// FormatZoneMappings formats zone mappings for output
func FormatZoneMappings(items []ZoneMapping, output string) string {
	if output == "json" { b, _ := json.MarshalIndent(items, "", "  "); return string(b) }
	if len(items) == 0 { return "未找到 ALB Zone 映射" }
	lines := []string{fmt.Sprintf("Zone 映射数量: %d", len(items)), ""}
	for _, it := range items {
		lines = append(lines, fmt.Sprintf("- %s (%s)", it.LoadBalancerName, it.LoadBalancerID))
		lines = append(lines, fmt.Sprintf("  可用区: %s | 交换机: %s | 内网IP: %s", it.ZoneId, it.VSwitchId, it.IntranetAddress))
		if it.EipAddress != "" {
			lines = append(lines, fmt.Sprintf("  EIP: %s | 状态: %s", it.EipAddress, it.Status))
		}
	}
	return strings.Join(lines, "\n")
}