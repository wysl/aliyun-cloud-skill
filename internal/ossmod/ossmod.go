package ossmod

import (
	"aliyun-cloud-skill/internal/aliyuncli"
	"aliyun-cloud-skill/internal/prom"
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Bucket struct {
	Name         string `json:"name"`
	Region       string `json:"region"`
	CreationTime string `json:"creationTime"`
}

type Object struct {
	Key          string `json:"key"`
	Size         int64  `json:"size"`
	LastModified string `json:"lastModified"`
}

// UsageData represents OSS bucket usage statistics
type UsageData struct {
	BucketName   string  `json:"bucketName"`
	Region       string  `json:"region"`
	Storage      float64 `json:"storage"`       // 存储使用量 (bytes)
	StorageGB    float64 `json:"storageGB"`    // 存储使用量 (GB)
	ObjectCount  int64   `json:"objectCount"`  // 对象数量
	Traffic      float64 `json:"traffic"`       // 当月流量 (bytes)
	TrafficGB    float64 `json:"trafficGB"`    // 当月流量 (GB)
	RequestCount float64 `json:"requestCount"` // 当月请求数
}

func ListBuckets(env map[string]string) ([]Bucket, error) {
	out, err := aliyuncli.RunRaw([]string{"oss", "ls"}, env)
	if err != nil {
		return nil, err
	}
	s := bufio.NewScanner(strings.NewReader(string(out)))
	res := []Bucket{}
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		// Skip header line, empty lines, and summary line
		if line == "" || strings.Contains(line, "CreationTime") || strings.Contains(line, "Bucket Number") {
			continue
		}
		parts := strings.Fields(line)
		// Expected format: "2026-03-09 16:43:38 +0800 CST oss-cn-shanghai Standard oss://bucket-name"
		// parts[0-3]: creation time (date, time, timezone offset, timezone name)
		// parts[4]: region (e.g., "oss-cn-shanghai")
		// parts[5]: storage class
		// parts[6]: bucket name (oss://bucket-name)
		if len(parts) < 7 {
			continue
		}
		name := strings.TrimSuffix(strings.TrimPrefix(parts[6], "oss://"), "/")
		region := parts[4]
		// Remove "oss-" prefix from region if present (e.g., "oss-cn-shanghai" -> "cn-shanghai")
		region = strings.TrimPrefix(region, "oss-")
		ctime := parts[0] + " " + parts[1] + " " + parts[2] + " " + parts[3]
		res = append(res, Bucket{Name: name, Region: region, CreationTime: ctime})
	}
	return res, nil
}

func BucketInfo(env map[string]string, bucket string) (string, error) {
	return aliyuncli.RunRaw([]string{"oss", "stat", "oss://"+bucket}, env)
}

func ListObjects(env map[string]string, bucket, prefix string, maxKeys int) ([]Object, error) {
	out, err := aliyuncli.RunRaw([]string{"oss", "ls", "oss://" + bucket + "/" + prefix}, env)
	if err != nil {
		return nil, err
	}
	s := bufio.NewScanner(strings.NewReader(string(out)))
	res := []Object{}
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		// Skip empty lines, header line, and summary line
		if line == "" || strings.Contains(line, "LastModifiedTime") || strings.Contains(line, "Object Number is") {
			continue
		}
		parts := strings.Fields(line)
		// Expected format: "2026-03-27 13:58:48 +0800 CST 0 Standard ETAG oss://bucket/key"
		// parts[0-3]: timestamp (2026-03-27 13:58:48 +0800 CST)
		// parts[4]: size
		// parts[5]: storage class
		// parts[6]: ETAG
		// parts[7]: object key (oss://bucket/key)
		if len(parts) < 8 {
			continue
		}
		size, _ := strconv.ParseInt(parts[4], 10, 64)
		// Extract object key (remove oss://bucket/ prefix)
		key := parts[7]
		if strings.HasPrefix(key, "oss://") {
			// Remove oss://bucket/ prefix to get just the object key
			key = strings.TrimPrefix(key, "oss://"+bucket+"/")
		}
		// LastModified: use only date and time (parts[0] + " " + parts[1])
		lastModified := parts[0] + " " + parts[1]
		res = append(res, Object{
			LastModified: lastModified,
			Size:         size,
			Key:          key,
		})
		if len(res) >= maxKeys {
			break
		}
	}
	return res, nil
}

func RecentObjects(env map[string]string, bucket, prefix string, maxKeys int, hours int, fileTypes []string) ([]Object, error) {
	// Use ossutil-v2 with --max-age parameter for server-side filtering
	accessKeyID := env["ALIBABA_CLOUD_ACCESS_KEY_ID"]
	accessKeySecret := env["ALIBABA_CLOUD_ACCESS_KEY_SECRET"]
	region := env["ALIYUN_REGION_IDS"]
	if region == "" {
		region = "cn-shanghai"
	}
	endpoint := "oss-" + region + ".aliyuncs.com"

	// Build ossutil command with --max-age parameter
	// --max-age format: 1h (1 hour), 24h (24 hours), 1d (1 day), etc.
	maxAgeStr := fmt.Sprintf("%dh", hours)
	cmd := exec.Command("ossutil", "ls", "oss://"+bucket+"/"+prefix,
		"--max-age", maxAgeStr,
		"-i", accessKeyID,
		"-k", accessKeySecret,
		"-e", endpoint,
		"--region", region)

	out, err := cmd.Output()
	if err != nil {
		// Fallback to original method if ossutil fails
		return recentObjectsFallback(env, bucket, prefix, maxKeys, hours, fileTypes)
	}

	// Parse output
	result := []Object{}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines, header line, and summary line
		if line == "" || strings.Contains(line, "LastModifiedTime") || strings.Contains(line, "Object Number is") {
			continue
		}
		parts := strings.Fields(line)
		// Expected format: "2026-03-27 13:58:48 +0800 CST 0 Standard ETAG oss://bucket/key"
		if len(parts) < 8 {
			continue
		}

		// Extract object key (remove oss://bucket/ prefix)
		key := parts[7]
		if strings.HasPrefix(key, "oss://") {
			key = strings.TrimPrefix(key, "oss://"+bucket+"/")
		}

		// Filter by file types if specified
		if len(fileTypes) > 0 {
			ok := false
			for _, ft := range fileTypes {
				if strings.HasSuffix(strings.ToLower(key), strings.ToLower(ft)) {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}

		size, _ := strconv.ParseInt(parts[4], 10, 64)
		lastModified := parts[0] + " " + parts[1]

		result = append(result, Object{
			LastModified: lastModified,
			Size:         size,
			Key:          key,
		})

		if len(result) >= maxKeys {
			break
		}
	}

	return result, nil
}

// recentObjectsFallback is the fallback method using original ossutil (v1)
func recentObjectsFallback(env map[string]string, bucket, prefix string, maxKeys int, hours int, fileTypes []string) ([]Object, error) {
	items, err := ListObjects(env, bucket, prefix, maxKeys*5)
	if err != nil {
		return nil, err
	}
	threshold := time.Now().Add(-time.Duration(hours) * time.Hour)
	out := []Object{}
	for _, it := range items {
		if len(fileTypes) > 0 {
			ok := false
			for _, ft := range fileTypes {
				if strings.HasSuffix(strings.ToLower(it.Key), strings.ToLower(ft)) {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		if t, err := time.Parse("2006-01-02 15:04:05", it.LastModified); err == nil {
			if t.Before(threshold) {
				continue
			}
		}
		out = append(out, it)
		if len(out) >= maxKeys {
			break
		}
	}
	return out, nil
}

// Usage queries OSS bucket usage statistics using ossutil and Prometheus
// Returns storage usage (ossutil), traffic (Prometheus), request count (Prometheus)
func Usage(env map[string]string, baseURL, user, pass string, datasource, timeoutSec int) ([]UsageData, error) {
	// Get bucket list first
	buckets, err := ListBuckets(env)
	if err != nil {
		return nil, err
	}

	// Get AccessKey from env
	accessKeyID := env["ALIBABA_CLOUD_ACCESS_KEY_ID"]
	accessKeySecret := env["ALIBABA_CLOUD_ACCESS_KEY_SECRET"]
	defaultRegion := env["ALIYUN_REGION_IDS"]
	if defaultRegion == "" {
		defaultRegion = "cn-shanghai"
	}

	// Create usage map for aggregation, using each bucket's region
	usageMap := make(map[string]*UsageData)
	for _, b := range buckets {
		bucketRegion := b.Region
		if bucketRegion == "" {
			bucketRegion = defaultRegion // Fallback to default region
		}
		usageMap[b.Name] = &UsageData{
			BucketName: b.Name,
			Region:     bucketRegion,
		}
	}

	// Use ossutil du to get storage usage for each bucket
	for name := range usageMap {
		bucketRegion := usageMap[name].Region
		bucketEndpoint := "oss-" + bucketRegion + ".aliyuncs.com"
		cmd := exec.Command("ossutil", "du", "oss://"+name,
			"-i", accessKeyID,
			"-k", accessKeySecret,
			"-e", bucketEndpoint,
			"--region", bucketRegion)
		out, err := cmd.Output()
		if err == nil {
			output := string(out)
			output = strings.ReplaceAll(output, "\r", "")
			lines := strings.Split(output, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				// ossutil-v2 format: "total du size:592528613" (no "(byte)" suffix)
				if strings.HasPrefix(line, "total du size:") {
					idx := strings.Index(line, ":")
					if idx > 0 {
						sizeStr := strings.TrimSpace(line[idx+1:])
						var size float64
						fmt.Sscanf(sizeStr, "%f", &size)
						usageMap[name].Storage = size
						usageMap[name].StorageGB = size / (1024 * 1024 * 1024)
					}
				}
				if strings.HasPrefix(line, "total object count:") {
					idx := strings.Index(line, ":")
					if idx > 0 {
						rest := line[idx+1:]
						rest = strings.TrimSpace(rest)
						var count int64
						fmt.Sscanf(rest, "%d", &count)
						usageMap[name].ObjectCount = count
					}
				}
			}
		}
	}

	// Query Prometheus for traffic (InternetSend)
	trafficData, err := prom.Query(baseURL, user, pass, "sum_over_time(AliyunOss_InternetSend{product=\"oss\"}[30d])", datasource, timeoutSec)
	if err == nil {
		trafficResults := getResults(trafficData)
		for _, item := range trafficResults {
			bucketName := getStringLabel(item, "BucketName")
			if bucketName == "" {
				bucketName = getStringLabel(item, "id")
			}
			if bucketName == "" {
				continue
			}
			if usageMap[bucketName] != nil {
				usageMap[bucketName].Traffic = getFloatValue(item)
				usageMap[bucketName].TrafficGB = usageMap[bucketName].Traffic / (1024 * 1024 * 1024)
			}
		}
	}

	// Query Prometheus for request count (ValidRequestCount)
	requestData, err := prom.Query(baseURL, user, pass, "sum_over_time(AliyunOss_ValidRequestCount{product=\"oss\"}[30d])", datasource, timeoutSec)
	if err == nil {
		requestResults := getResults(requestData)
		for _, item := range requestResults {
			bucketName := getStringLabel(item, "BucketName")
			if bucketName == "" {
				bucketName = getStringLabel(item, "id")
			}
			if bucketName == "" {
				continue
			}
			if usageMap[bucketName] != nil {
				usageMap[bucketName].RequestCount = getFloatValue(item)
			}
		}
	}

	// Convert map to slice
	result := make([]UsageData, 0, len(usageMap))
	for _, v := range usageMap {
		result = append(result, *v)
	}

	return result, nil
}

// Helper functions for Prometheus data parsing
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

func FormatBuckets(items []Bucket, output string) string {
	if output == "json" { b, _ := json.MarshalIndent(items, "", "  "); return string(b) }
	if len(items) == 0 { return "未找到存储桶" }
	lines := []string{fmt.Sprintf("存储桶数量: %d", len(items)), ""}
	for _, it := range items { lines = append(lines, fmt.Sprintf("- %s", it.Name)) }
	return strings.Join(lines, "\n")
}

func FormatObjects(items []Object, output string, bucket string) string {
	if output == "json" { b, _ := json.MarshalIndent(items, "", "  "); return string(b) }
	if len(items) == 0 { return fmt.Sprintf("存储桶 %s 中没有文件", bucket) }
	lines := []string{fmt.Sprintf("对象数量: %d", len(items)), ""}
	for _, it := range items { lines = append(lines, fmt.Sprintf("- %s | %d bytes | %s", it.Key, it.Size, it.LastModified)) }
	return strings.Join(lines, "\n")
}

// FormatUsage formats OSS usage data for output
func FormatUsage(items []UsageData, output string) string {
	if output == "json" {
		b, _ := json.MarshalIndent(items, "", "  ")
		return string(b)
	}
	if len(items) == 0 {
		return "未找到 OSS 存储桶使用数据"
	}

	lines := []string{fmt.Sprintf("存储桶数量: %d", len(items)), ""}
	for _, it := range items {
		lines = append(lines, fmt.Sprintf("- %s", it.BucketName))
		lines = append(lines, fmt.Sprintf("  区域: %s", it.Region))
		
		// Format storage usage
		if it.StorageGB > 0 {
			lines = append(lines, fmt.Sprintf("  存储使用量: %.2f GB", it.StorageGB))
		} else {
			lines = append(lines, "  存储使用量: 无数据")
		}
		
		// Format object count
		if it.ObjectCount > 0 {
			lines = append(lines, fmt.Sprintf("  对象数量: %d", it.ObjectCount))
		} else {
			lines = append(lines, "  对象数量: 0")
		}
		
		// Format traffic
		if it.TrafficGB > 0 {
			lines = append(lines, fmt.Sprintf("  当月流量: %.2f GB", it.TrafficGB))
		} else {
			lines = append(lines, "  当月流量: 无数据")
		}
		
		// Format request count
		if it.RequestCount > 0 {
			lines = append(lines, fmt.Sprintf("  当月请求数: %.0f", it.RequestCount))
		} else {
			lines = append(lines, "  当月请求数: 无数据")
		}
	}

	return strings.Join(lines, "\n")
}

