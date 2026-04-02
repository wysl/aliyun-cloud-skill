package polardb

import (
	"aliyun-cloud-monitor/internal/aliyuncli"
	"aliyun-cloud-monitor/internal/prom"
	"encoding/json"
	"fmt"
	"strings"
)

type Instance struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Status        string `json:"status"`
	Engine        string `json:"engine"`
	EngineVersion string `json:"engineVersion"`
	Class         string `json:"class"`
}

// UsageData represents resource usage for a PolarDB instance
type UsageData struct {
	InstanceID   string  `json:"instanceId"`
	InstanceName string  `json:"instanceName"`
	Engine       string  `json:"engine"`
	CPU          float64 `json:"cpu"`
	Memory       float64 `json:"memory"`
	IOPS         float64 `json:"iops"`
}

func List(region string, env map[string]string) ([]Instance, error) {
	args := []string{"polardb", "DescribeDBClusters", "--RegionId", region}
	var data struct {
		Items struct {
			DBCluster []struct {
				DBClusterId          string `json:"DBClusterId"`
				DBClusterDescription string `json:"DBClusterDescription"`
				DBClusterStatus      string `json:"DBClusterStatus"`
				Engine               string `json:"Engine"`
				DBVersion            string `json:"DBVersion"`
				DBNodeClass          string `json:"DBNodeClass"`
			} `json:"DBCluster"`
		} `json:"Items"`
	}
	if err := aliyuncli.RunJSON(args, env, &data); err != nil { return nil, err }
	res := make([]Instance, 0, len(data.Items.DBCluster))
	for _, it := range data.Items.DBCluster {
		res = append(res, Instance{
			ID:            it.DBClusterId,
			Name:          it.DBClusterDescription,
			Status:        it.DBClusterStatus,
			Engine:        it.Engine,
			EngineVersion: it.DBVersion,
			Class:         it.DBNodeClass,
		})
	}
	return res, nil
}

// Usage queries Prometheus for PolarDB resource usage (CPU, memory, IOPS)
// Metric names: AliyunPolardb_cluster_*
func Usage(baseURL, user, pass string, datasource, timeoutSec int) ([]UsageData, error) {
	// Query CPU usage
	cpuData, err := prom.Query(baseURL, user, pass, "AliyunPolardb_cluster_cpu_utilization", datasource, timeoutSec)
	if err != nil { return nil, fmt.Errorf("query CPU failed: %w", err) }
	
	// Query memory usage
	memData, err := prom.Query(baseURL, user, pass, "AliyunPolardb_cluster_memory_utilization", datasource, timeoutSec)
	if err != nil { return nil, fmt.Errorf("query memory failed: %w", err) }
	
	// Query IOPS usage
	iopsData, err := prom.Query(baseURL, user, pass, "AliyunPolardb_cluster_iops_usage", datasource, timeoutSec)
	if err != nil { return nil, fmt.Errorf("query IOPS failed: %w", err) }
	
	// Aggregate by instance ID
	usageMap := make(map[string]*UsageData)
	
	// Process CPU data
	cpuResults := getResults(cpuData)
	for _, item := range cpuResults {
		instanceID := getStringLabel(item, "instanceId")
		instanceName := getStringLabel(item, "instanceName")
		engine := getStringLabel(item, "engine")
		if instanceID == "" { continue }
		if _, ok := usageMap[instanceID]; !ok {
			usageMap[instanceID] = &UsageData{InstanceID: instanceID, InstanceName: instanceName, Engine: engine}
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
	
	// Process IOPS data
	iopsResults := getResults(iopsData)
	for _, item := range iopsResults {
		instanceID := getStringLabel(item, "instanceId")
		if instanceID == "" { continue }
		if _, ok := usageMap[instanceID]; ok {
			usageMap[instanceID].IOPS = getFloatValue(item)
		}
	}
	
	// Convert map to slice
	result := make([]UsageData, 0, len(usageMap))
	for _, v := range usageMap {
		result = append(result, *v)
	}
	
	return result, nil
}

func getSeriesCount(data map[string]any) int {
	if d, ok := data["data"].(map[string]any); ok {
		if r, ok := d["result"].([]any); ok {
			return len(r)
		}
	}
	return 0
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

// getStringLabel tries multiple label names to get the value
func getStringLabel(item map[string]any, key string) string {
	if labels, ok := item["metric"].(map[string]any); ok {
		// Try exact key first
		if v, ok := labels[key].(string); ok && v != "" {
			return v
		}
		// Try alternative keys
		altKeys := map[string][]string{
			"instanceId":   {"clusterId", "id"},
			"instanceName": {"desc", "clusterDescription"},
		}
		if alts, ok := altKeys[key]; ok {
			for _, alt := range alts {
				if v, ok := labels[alt].(string); ok && v != "" {
					return v
				}
			}
		}
		// Fallback: try common keys
		commonKeys := []string{"clusterId", "id", "instanceId"}
		nameKeys := []string{"desc", "clusterDescription", "instanceName"}
		if key == "instanceId" {
			for _, k := range commonKeys {
				if v, ok := labels[k].(string); ok && v != "" {
					return v
				}
			}
		}
		if key == "instanceName" {
			for _, k := range nameKeys {
				if v, ok := labels[k].(string); ok && v != "" {
					return v
				}
			}
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
	if len(items) == 0 { return "未找到 PolarDB 实例" }
	lines := []string{fmt.Sprintf("实例数量: %d", len(items)), ""}
	for _, it := range items {
		name := it.Name; if name == "" { name = it.ID }
		lines = append(lines, fmt.Sprintf("- %s", name))
		lines = append(lines, fmt.Sprintf("  ID: %s | 引擎: %s/%s", it.ID, it.Engine, it.EngineVersion))
		lines = append(lines, fmt.Sprintf("  规格: %s | 状态: %s", it.Class, it.Status))
	}
	return strings.Join(lines, "\n")
}

func FormatUsage(items []UsageData, output string) string {
	if output == "json" { b, _ := json.MarshalIndent(items, "", "  "); return string(b) }
	if len(items) == 0 { return "未找到 PolarDB 实例资源消耗数据" }
	lines := []string{fmt.Sprintf("实例数量: %d", len(items)), ""}
	for _, it := range items {
		name := it.InstanceName
		if name == "" { name = it.InstanceID }
		engine := it.Engine
		if engine == "" { engine = "PolarDB" }
		lines = append(lines, fmt.Sprintf("- %s (%s)", name, it.InstanceID))
		lines = append(lines, fmt.Sprintf("  引擎: %s | CPU: %.2f%% | 内存: %.2f%% | IOPS: %.2f%%", engine, it.CPU, it.Memory, it.IOPS))
	}
	return strings.Join(lines, "\n")
}