package rds

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

type Backup struct {
	ID     string `json:"id"`
	Time   string `json:"time"`
	SizeMB float64 `json:"sizeMB"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

// UsageData represents resource usage for an RDS instance
type UsageData struct {
	InstanceID   string  `json:"instanceId"`
	InstanceName string  `json:"instanceName"`
	Engine       string  `json:"engine"`
	CPU          float64 `json:"cpu"`
	Memory       float64 `json:"memory"`
	IOPS         float64 `json:"iops"`
}

func List(region string, env map[string]string) ([]Instance, error) {
	args := []string{"rds", "DescribeDBInstances", "--RegionId", region}
	var data struct {
		Items struct {
			DBInstance []struct {
				DBInstanceId string `json:"DBInstanceId"`
				DBInstanceDescription string `json:"DBInstanceDescription"`
				DBInstanceStatus string `json:"DBInstanceStatus"`
				Engine string `json:"Engine"`
				EngineVersion string `json:"EngineVersion"`
				DBInstanceClass string `json:"DBInstanceClass"`
			} `json:"DBInstance"`
		} `json:"Items"`
	}
	if err := aliyuncli.RunJSON(args, env, &data); err != nil { return nil, err }
	res := make([]Instance, 0, len(data.Items.DBInstance))
	for _, it := range data.Items.DBInstance {
		res = append(res, Instance{ID: it.DBInstanceId, Name: it.DBInstanceDescription, Status: it.DBInstanceStatus, Engine: it.Engine, EngineVersion: it.EngineVersion, Class: it.DBInstanceClass})
	}
	return res, nil
}

func Detail(env map[string]string, instanceID string) (map[string]any, error) {
	args := []string{"rds", "DescribeDBInstanceAttribute", "--DBInstanceId", instanceID}
	var data map[string]any
	if err := aliyuncli.RunJSON(args, env, &data); err != nil { return nil, err }
	return data, nil
}

func Performance(env map[string]string, instanceID string) (map[string]any, error) {
	args := []string{"rds", "DescribeResourceUsage", "--DBInstanceId", instanceID}
	var data map[string]any
	if err := aliyuncli.RunJSON(args, env, &data); err != nil { return nil, err }
	return data, nil
}

func ListBackups(env map[string]string, instanceID, start, end string) ([]Backup, error) {
	args := []string{"rds", "DescribeBackups", "--DBInstanceId", instanceID}
	if strings.TrimSpace(start) != "" { args = append(args, "--StartTime", start) }
	if strings.TrimSpace(end) != "" { args = append(args, "--EndTime", end) }
	var data struct {
		Items struct {
			Backup []struct {
				BackupId string `json:"BackupId"`
				BackupStartTime string `json:"BackupStartTime"`
				BackupSize float64 `json:"BackupSize"`
				BackupType string `json:"BackupType"`
				BackupStatus string `json:"BackupStatus"`
			} `json:"Backup"`
		} `json:"Items"`
	}
	if err := aliyuncli.RunJSON(args, env, &data); err != nil { return nil, err }
	res := make([]Backup, 0, len(data.Items.Backup))
	for _, it := range data.Items.Backup {
		res = append(res, Backup{ID: it.BackupId, Time: it.BackupStartTime, SizeMB: it.BackupSize / 1024 / 1024, Type: it.BackupType, Status: it.BackupStatus})
	}
	return res, nil
}

func FormatInstances(items []Instance, output string) string {
	if output == "json" { b, _ := json.MarshalIndent(items, "", "  "); return string(b) }
	if len(items) == 0 { return "未找到 RDS 实例" }
	lines := []string{fmt.Sprintf("实例数量: %d", len(items)), ""}
	for _, it := range items {
		name := it.Name; if name == "" { name = it.ID }
		lines = append(lines, fmt.Sprintf("- %s", name))
		lines = append(lines, fmt.Sprintf("  ID: %s | 引擎: %s/%s", it.ID, it.Engine, it.EngineVersion))
		lines = append(lines, fmt.Sprintf("  规格: %s | 状态: %s", it.Class, it.Status))
	}
	return strings.Join(lines, "\n")
}

func FormatAny(data map[string]any, output string) string {
	b, _ := json.MarshalIndent(data, "", "  ")
	return string(b)
}

func FormatBackups(items []Backup, output string) string {
	if output == "json" { b, _ := json.MarshalIndent(items, "", "  "); return string(b) }
	if len(items) == 0 { return "未找到备份" }
	lines := []string{fmt.Sprintf("备份数量: %d", len(items)), ""}
	for _, it := range items {
		lines = append(lines, fmt.Sprintf("- 备份ID: %s", it.ID))
		lines = append(lines, fmt.Sprintf("  时间: %s | 大小: %.2f MB | 类型: %s | 状态: %s", it.Time, it.SizeMB, it.Type, it.Status))
	}
	return strings.Join(lines, "\n")
}

// Usage queries Prometheus for RDS resource usage (CPU, memory, IOPS)
// Note: Metric names depend on the engine type (PostgreSQL uses AliyunRds_pg_*)
func Usage(baseURL, user, pass string, datasource, timeoutSec int) ([]UsageData, error) {
	// Query PostgreSQL metrics
	cpuData, err := prom.Query(baseURL, user, pass, "AliyunRds_pg_cpu_usage", datasource, timeoutSec)
	if err != nil { return nil, fmt.Errorf("query CPU failed: %w", err) }
	
	memData, err := prom.Query(baseURL, user, pass, "AliyunRds_pg_mem_usage", datasource, timeoutSec)
	if err != nil { return nil, fmt.Errorf("query memory failed: %w", err) }
	
	iopsData, err := prom.Query(baseURL, user, pass, "AliyunRds_pg_iops_usage", datasource, timeoutSec)
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
	if len(items) == 0 { return "未找到 RDS 实例资源消耗数据" }
	lines := []string{fmt.Sprintf("实例数量: %d", len(items)), ""}
	for _, it := range items {
		name := it.InstanceName
		if name == "" { name = it.InstanceID }
		engine := it.Engine
		if engine == "" { engine = "PostgreSQL" }
		lines = append(lines, fmt.Sprintf("- %s (%s)", name, it.InstanceID))
		lines = append(lines, fmt.Sprintf("  引擎: %s | CPU: %.2f%% | 内存: %.2f%% | IOPS: %.2f%%", engine, it.CPU, it.Memory, it.IOPS))
	}
	return strings.Join(lines, "\n")
}

