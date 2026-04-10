package ecs

import (
	"aliyun-cloud-skill/internal/aliyuncli"
	"aliyun-cloud-skill/internal/prom"
	"encoding/json"
	"fmt"
	"strings"
)

type Instance struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Status    string   `json:"status"`
	Type      string   `json:"type"`
	IP        string   `json:"ip"`
	PrivateIP []string `json:"privateIp,omitempty"`
	CPU       int      `json:"cpu,omitempty"`
	MemoryMB  int      `json:"memoryMB,omitempty"`
	Region    string   `json:"region,omitempty"`
	Zone      string   `json:"zone,omitempty"`
	VSwitchId string   `json:"vswitchId,omitempty"`
}

// UsageData represents resource usage for an ECS instance
type UsageData struct {
	InstanceID   string  `json:"instanceId"`
	InstanceName string  `json:"instanceName"`
	CPU          float64 `json:"cpu"`
	Memory       float64 `json:"memory"`
	Disk         float64 `json:"disk"`
}

type actionResponse struct {
	Action     string `json:"action"`
	InstanceID string `json:"instanceId"`
	Region     string `json:"region"`
	Message    string `json:"message"`
}

func List(region string, env map[string]string, status string) ([]Instance, error) {
	args := []string{"ecs", "DescribeInstances", "--RegionId", region}
	if strings.TrimSpace(status) != "" { args = append(args, "--Status", status) }
	var data struct {
		Instances struct {
			Instance []struct {
				InstanceId   string `json:"InstanceId"`
				InstanceName string `json:"InstanceName"`
				Status       string `json:"Status"`
				InstanceType string `json:"InstanceType"`
				Cpu          int    `json:"Cpu"`
				Memory       int    `json:"Memory"`
				RegionId     string `json:"RegionId"`
				ZoneId       string `json:"ZoneId"`
				PublicIpAddress struct { IpAddress []string `json:"IpAddress"` } `json:"PublicIpAddress"`
				VpcAttributes struct {
					PrivateIpAddress struct { IpAddress []string `json:"IpAddress"` } `json:"PrivateIpAddress"`
					VSwitchId         string `json:"VSwitchId"`
				} `json:"VpcAttributes"`
			} `json:"Instance"`
		} `json:"Instances"`
	}
	if err := aliyuncli.RunJSON(args, env, &data); err != nil { return nil, err }
	res := make([]Instance, 0, len(data.Instances.Instance))
	for _, it := range data.Instances.Instance {
		ip := ""
		if len(it.PublicIpAddress.IpAddress) > 0 { ip = it.PublicIpAddress.IpAddress[0] }
		res = append(res, Instance{
			ID: it.InstanceId, Name: it.InstanceName, Status: it.Status, Type: it.InstanceType, IP: ip,
			PrivateIP: it.VpcAttributes.PrivateIpAddress.IpAddress, CPU: it.Cpu, MemoryMB: it.Memory,
			Region: it.RegionId, Zone: it.ZoneId, VSwitchId: it.VpcAttributes.VSwitchId,
		})
	}
	return res, nil
}

func Detail(region string, env map[string]string, instanceID string) (*Instance, error) {
	items, err := List(region, env, "")
	if err != nil { return nil, err }
	for _, it := range items {
		if it.ID == instanceID { return &it, nil }
	}
	return nil, fmt.Errorf("instance not found: %s", instanceID)
}

func Start(region string, env map[string]string, instanceID string) (string, error) {
	return doAction("StartInstance", region, env, instanceID, false)
}
func Stop(region string, env map[string]string, instanceID string, force bool) (string, error) {
	return doAction("StopInstance", region, env, instanceID, force)
}
func Reboot(region string, env map[string]string, instanceID string, force bool) (string, error) {
	return doAction("RebootInstance", region, env, instanceID, force)
}

func doAction(action, region string, env map[string]string, instanceID string, force bool) (string, error) {
	args := []string{"ecs", action, "--RegionId", region, "--InstanceId", instanceID}
	if force {
		switch action {
		case "StopInstance":
			args = append(args, "--ForceStop", "true")
		case "RebootInstance":
			args = append(args, "--ForceReboot", "true")
		}
	}
	_, err := aliyuncli.RunRaw(args, env)
	if err != nil {
		return "", fmt.Errorf("%s failed: %w", action, err)
	}
	resp := actionResponse{Action: action, InstanceID: instanceID, Region: region, Message: fmt.Sprintf("%s request submitted", action)}
	b, _ := json.MarshalIndent(resp, "", "  ")
	return string(b), nil
}

func FormatList(items []Instance, output string) string {
	if output == "json" { b, _ := json.MarshalIndent(items, "", "  "); return string(b) }
	if len(items) == 0 { return "未找到实例" }
	lines := []string{fmt.Sprintf("实例数量: %d", len(items)), ""}
	for _, it := range items {
		name := it.Name; if name == "" { name = it.ID }
		ip := it.IP; if ip == "" { ip = "N/A" }
		lines = append(lines, fmt.Sprintf("- %s (%s)", name, it.ID))
		lines = append(lines, fmt.Sprintf("  状态: %s | 公网IP: %s", it.Status, ip))
	}
	return strings.Join(lines, "\n")
}

func FormatDetail(item *Instance, output string) string {
	if output == "json" { b, _ := json.MarshalIndent(item, "", "  "); return string(b) }
	if item == nil { return "未找到实例" }
	lines := []string{"=== 实例详情 ===", "", fmt.Sprintf("实例ID: %s", item.ID), fmt.Sprintf("名称: %s", item.Name), fmt.Sprintf("状态: %s", item.Status), fmt.Sprintf("类型: %s", item.Type), fmt.Sprintf("CPU: %d 核", item.CPU), fmt.Sprintf("内存: %d MB", item.MemoryMB), fmt.Sprintf("区域: %s", item.Region), fmt.Sprintf("可用区: %s", item.Zone)}
	if item.IP != "" { lines = append(lines, fmt.Sprintf("公网IP: %s", item.IP)) }
	if len(item.PrivateIP) > 0 { lines = append(lines, fmt.Sprintf("内网IP: %s", strings.Join(item.PrivateIP, ", "))) }
	return strings.Join(lines, "\n")
}

// Usage queries Prometheus for ECS resource usage (CPU, memory, disk)
func Usage(baseURL, user, pass string, datasource, timeoutSec int) ([]UsageData, error) {
	// Query CPU usage
	cpuData, err := prom.Query(baseURL, user, pass, "AliyunEcs_cpu_total", datasource, timeoutSec)
	if err != nil { return nil, fmt.Errorf("query CPU failed: %w", err) }
	
	// Query memory usage
	memData, err := prom.Query(baseURL, user, pass, "AliyunEcs_memory_usedutilization", datasource, timeoutSec)
	if err != nil { return nil, fmt.Errorf("query memory failed: %w", err) }
	
	// Query disk usage (root partition only)
	diskData, err := prom.Query(baseURL, user, pass, `AliyunEcs_diskusage_utilization{diskname="/"}`, datasource, timeoutSec)
	if err != nil { return nil, fmt.Errorf("query disk failed: %w", err) }
	
	// Aggregate by instance ID
	usageMap := make(map[string]*UsageData)
	
	// Process CPU data
	cpuResults := getResults(cpuData)
	for _, item := range cpuResults {
		instanceID := getStringLabel(item, "instanceId")
		instanceName := getStringLabel(item, "instanceName")
		if instanceID == "" { continue }
		if _, ok := usageMap[instanceID]; !ok {
			usageMap[instanceID] = &UsageData{InstanceID: instanceID, InstanceName: instanceName}
		}
		usageMap[instanceID].CPU = getFloatValue(item)
	}
	
	// Process memory data
	memResults := getResults(memData)
	for _, item := range memResults {
		instanceID := getStringLabel(item, "instanceId")
		if instanceID == "" { continue }
		if _, ok := usageMap[instanceID]; ok {
			usageMap[instanceID].Memory = getFloatValue(item)
		}
	}
	
	// Process disk data
	diskResults := getResults(diskData)
	for _, item := range diskResults {
		instanceID := getStringLabel(item, "instanceId")
		if instanceID == "" { continue }
		if _, ok := usageMap[instanceID]; ok {
			usageMap[instanceID].Disk = getFloatValue(item)
		}
	}
	
	// Convert map to slice, filter out host-* instances
	result := make([]UsageData, 0, len(usageMap))
	for _, v := range usageMap {
		// Skip instances with ID starting with "host-"
		if strings.HasPrefix(v.InstanceID, "host-") {
			continue
		}
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

func FormatUsage(items []UsageData, output string) string {
	if output == "json" { b, _ := json.MarshalIndent(items, "", "  "); return string(b) }
	if len(items) == 0 { return "未找到实例资源消耗数据" }
	lines := []string{fmt.Sprintf("实例数量: %d", len(items)), ""}
	for _, it := range items {
		name := it.InstanceName
		if name == "" { name = it.InstanceID }
		lines = append(lines, fmt.Sprintf("- %s (%s)", name, it.InstanceID))
		lines = append(lines, fmt.Sprintf("  CPU: %.2f%% | 内存: %.2f%% | 磁盘: %.2f%%", it.CPU, it.Memory, it.Disk))
	}
	return strings.Join(lines, "\n")
}

// ListByVSwitch lists ECS instances in a specific VSwitch
func ListByVSwitch(region, vswitchId string, env map[string]string) ([]Instance, error) {
	instances, err := List(region, env, "")
	if err != nil {
		return nil, err
	}
	res := make([]Instance, 0)
	for _, it := range instances {
		if it.VSwitchId == vswitchId {
			res = append(res, it)
		}
	}
	return res, nil
}

