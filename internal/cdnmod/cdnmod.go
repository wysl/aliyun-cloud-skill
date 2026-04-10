package cdnmod

import (
	"aliyun-cloud-skill/internal/aliyuncli"
	"aliyun-cloud-skill/internal/ossmod"
	"encoding/json"
	"fmt"
	"strings"
)

type Domain struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Cname  string `json:"cname"`
}

// UsageData represents CDN usage statistics for a domain
type UsageData struct {
	DomainName    string  `json:"domainName"`
	Status        string  `json:"status"`
	Traffic       float64 `json:"traffic"`       // 下行流量 (bytes)
	SrcTraffic    float64 `json:"srcTraffic"`    // 回源流量 (bytes)
	HitRate       float64 `json:"hitRate"`       // 缓存命中率 (%)
	TrafficGB     float64 `json:"trafficGB"`     // 下行流量 (GB)
	SrcTrafficGB  float64 `json:"srcTrafficGB"`  // 回源流量 (GB)
}

func List(env map[string]string) ([]Domain, error) {
	args := []string{"cdn", "DescribeUserDomains"}
	var data struct {
		Domains struct {
			PageData []struct {
				DomainName   string `json:"DomainName"`
				DomainStatus string `json:"DomainStatus"`
				Cname        string `json:"Cname"`
			} `json:"PageData"`
		} `json:"Domains"`
	}
	if err := aliyuncli.RunJSON(args, env, &data); err != nil {
		return nil, err
	}
	res := make([]Domain, 0, len(data.Domains.PageData))
	for _, it := range data.Domains.PageData {
		res = append(res, Domain{Name: it.DomainName, Status: it.DomainStatus, Cname: it.Cname})
	}
	return res, nil
}

func Detail(env map[string]string, domain string) (map[string]any, error) {
	args := []string{"cdn", "DescribeCdnDomainDetail", "--DomainName", domain}
	var data map[string]any
	if err := aliyuncli.RunJSON(args, env, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func Traffic(env map[string]string, domain, start, end string) (map[string]any, error) {
	args := []string{"cdn", "DescribeDomainTrafficData", "--DomainName", domain}
	if strings.TrimSpace(start) != "" {
		args = append(args, "--StartTime", start)
	}
	if strings.TrimSpace(end) != "" {
		args = append(args, "--EndTime", end)
	}
	var data map[string]any
	if err := aliyuncli.RunJSON(args, env, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func Bandwidth(env map[string]string, domain, start, end string) (map[string]any, error) {
	args := []string{"cdn", "DescribeDomainBpsData", "--DomainName", domain}
	if strings.TrimSpace(start) != "" {
		args = append(args, "--StartTime", start)
	}
	if strings.TrimSpace(end) != "" {
		args = append(args, "--EndTime", end)
	}
	var data map[string]any
	if err := aliyuncli.RunJSON(args, env, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func Refresh(env map[string]string, paths []string) (string, error) {
	args := []string{"cdn", "RefreshObjectCaches", "--ObjectPath", strings.Join(paths, "\n")}
	return runAliyunRaw(args, env)
}

func Push(env map[string]string, urls []string) (string, error) {
	args := []string{"cdn", "PushObjectCache", "--ObjectPath", strings.Join(urls, "\n")}
	return runAliyunRaw(args, env)
}

// SourceConfig represents CDN source (origin) configuration
type SourceConfig struct {
	DomainName string   `json:"domainName"` // CDN domain name
	Status     string   `json:"status"`     // CDN domain status
	Sources    []Source `json:"sources"`    // Source (origin) configurations
}

// Source represents a single CDN source (origin)
type Source struct {
	Content string `json:"content"` // Source content (e.g., bucket OSS domain)
	Type    string `json:"type"`    // Source type (oss, ip, domain)
	Port    int    `json:"port"`    // Port number
}

// AutoWarmupResult represents the result of auto warmup operation
type AutoWarmupResult struct {
	CDNDomain    string   `json:"cdnDomain"`    // CDN domain name
	BucketName   string   `json:"bucketName"`   // OSS bucket name
	ObjectsFound int      `json:"objectsFound"` // Number of recent objects found
	WarmupUrls   []string `json:"warmupUrls"`   // URLs warmed up
	PushResult   string   `json:"pushResult"`   // CDN push API result
}

// GetSourceConfigs gets CDN source configurations for all domains
func GetSourceConfigs(env map[string]string) ([]SourceConfig, error) {
	domains, err := List(env)
	if err != nil {
		return nil, err
	}

	result := make([]SourceConfig, 0, len(domains))
	for _, d := range domains {
		detail, err := Detail(env, d.Name)
		if err != nil {
			continue
		}

		// Extract source configurations from detail
		var sources []Source
		if model, ok := detail["GetDomainDetailModel"].(map[string]any); ok {
			if sourceModels, ok := model["SourceModels"].(map[string]any); ok {
				if sourceList, ok := sourceModels["SourceModel"].([]any); ok {
					for _, s := range sourceList {
						if sm, ok := s.(map[string]any); ok {
							src := Source{
								Content: getString(sm, "Content"),
								Type:    getString(sm, "Type"),
							}
							if port, ok := sm["Port"].(float64); ok {
								src.Port = int(port)
							}
							sources = append(sources, src)
						}
					}
				}
			}
		}

		result = append(result, SourceConfig{
			DomainName: d.Name,
			Status:     d.Status,
			Sources:    sources,
		})
	}

	return result, nil
}

// MatchBucketWithSource checks if an OSS bucket matches a CDN source
// Returns the CDN domain name if matched, empty string otherwise
func MatchBucketWithSource(bucketName string, sourceConfigs []SourceConfig) string {
	for _, cfg := range sourceConfigs {
		for _, src := range cfg.Sources {
			// Check if source type is OSS and content contains bucket name
			if src.Type == "oss" && strings.Contains(src.Content, bucketName) {
				return cfg.DomainName
			}
		}
	}
	return ""
}

// GenerateWarmupUrls generates CDN warmup URLs from OSS object keys
// AutoWarmup performs automatic CDN warmup for recently uploaded OSS objects
// Workflow:
// 1. Get CDN domain source configurations
// 2. Query recent OSS objects (within specified hours) using ossutil
// 3. Match OSS bucket with CDN source
// 4. Generate warmup URLs and execute push
func AutoWarmup(env map[string]string, bucketName string, hours int) ([]AutoWarmupResult, error) {
	// Step 1: Get CDN source configurations
	sourceConfigs, err := GetSourceConfigs(env)
	if err != nil {
		return nil, fmt.Errorf("failed to get CDN source configs: %w", err)
	}

	// Step 2: Query recent OSS objects using ossmod.RecentObjects (uses ossutil)
	objects, err := ossmod.RecentObjects(env, bucketName, "", 100, hours, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent objects: %w", err)
	}

	if len(objects) == 0 {
		return []AutoWarmupResult{}, nil
	}

	// Step 3: Match bucket with CDN source
	cdnDomain := MatchBucketWithSource(bucketName, sourceConfigs)
	if cdnDomain == "" {
		return nil, fmt.Errorf("OSS bucket %s is not configured as CDN source", bucketName)
	}

	// Step 4: Generate warmup URLs from OSS objects
	warmupUrls := make([]string, 0, len(objects))
	for _, obj := range objects {
		// Generate URL: https://cdn-domain/object-key
		url := fmt.Sprintf("https://%s/%s", cdnDomain, obj.Key)
		warmupUrls = append(warmupUrls, url)
	}

	// Step 5: Execute CDN push (warmup)
	pushResult, err := Push(env, warmupUrls)
	if err != nil {
		return nil, fmt.Errorf("failed to push URLs: %w", err)
	}

	return []AutoWarmupResult{
		{
			CDNDomain:    cdnDomain,
			BucketName:   bucketName,
			ObjectsFound: len(objects),
			WarmupUrls:   warmupUrls,
			PushResult:   pushResult,
		},
	}, nil
}

// queryRecentObjects is no longer needed - using ossmod.RecentObjects instead

// getString helper function for map[string]any
func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// FormatAutoWarmup formats auto warmup result for output
func FormatAutoWarmup(results []AutoWarmupResult, format string) string {
	if format == "json" {
		b, _ := json.MarshalIndent(results, "", "  ")
		return string(b)
	}

	if len(results) == 0 {
		return "未找到需要预热的文件"
	}

	lines := []string{}
	for _, r := range results {
		lines = append(lines, fmt.Sprintf("CDN 域名: %s", r.CDNDomain))
		lines = append(lines, fmt.Sprintf("OSS Bucket: %s", r.BucketName))
		lines = append(lines, fmt.Sprintf("发现文件数: %d", r.ObjectsFound))
		lines = append(lines, "预热 URL 列表:")
		for _, url := range r.WarmupUrls {
			lines = append(lines, fmt.Sprintf("  - %s", url))
		}
		lines = append(lines, fmt.Sprintf("预热结果: %s", r.PushResult))
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// QueryTrafficData queries CDN traffic data for a domain
func QueryTrafficData(env map[string]string, domain string, startTime, endTime string) (float64, error) {
	args := []string{"cdn", "DescribeDomainTrafficData", "--DomainName", domain}
	if strings.TrimSpace(startTime) != "" {
		args = append(args, "--StartTime", startTime)
	}
	if strings.TrimSpace(endTime) != "" {
		args = append(args, "--EndTime", endTime)
	}

	var data struct {
		TrafficData struct {
			DataInterval int `json:"DataInterval"`
			TrafficValue []struct {
				Traffic float64 `json:"Traffic"`
			} `json:"TrafficValue"`
		} `json:"TrafficData"`
	}

	if err := aliyuncli.RunJSON(args, env, &data); err != nil {
		return 0, err
	}

	// Sum up all traffic values
	var totalTraffic float64
	for _, v := range data.TrafficData.TrafficValue {
		totalTraffic += v.Traffic
	}

	return totalTraffic, nil
}

// QuerySrcTrafficData queries CDN source traffic data for a domain
func QuerySrcTrafficData(env map[string]string, domain string, startTime, endTime string) (float64, error) {
	args := []string{"cdn", "DescribeDomainSrcTrafficData", "--DomainName", domain}
	if strings.TrimSpace(startTime) != "" {
		args = append(args, "--StartTime", startTime)
	}
	if strings.TrimSpace(endTime) != "" {
		args = append(args, "--EndTime", endTime)
	}

	var data struct {
		SrcTrafficData struct {
			DataInterval int `json:"DataInterval"`
			SrcTrafficValue []struct {
				Traffic float64 `json:"Traffic"`
			} `json:"SrcTrafficValue"`
		} `json:"SrcTrafficData"`
	}

	if err := aliyuncli.RunJSON(args, env, &data); err != nil {
		return 0, err
	}

	// Sum up all source traffic values
	var totalTraffic float64
	for _, v := range data.SrcTrafficData.SrcTrafficValue {
		totalTraffic += v.Traffic
	}

	return totalTraffic, nil
}

// QueryHitRateData queries CDN hit rate data for a domain
func QueryHitRateData(env map[string]string, domain string, startTime, endTime string) (float64, error) {
	args := []string{"cdn", "DescribeDomainHitRateData", "--DomainName", domain}
	if strings.TrimSpace(startTime) != "" {
		args = append(args, "--StartTime", startTime)
	}
	if strings.TrimSpace(endTime) != "" {
		args = append(args, "--EndTime", endTime)
	}

	var data struct {
		HitRateData struct {
			DataInterval int `json:"DataInterval"`
			HitRateValue []struct {
				HitRate float64 `json:"HitRate"`
			} `json:"HitRateValue"`
		} `json:"HitRateData"`
	}

	if err := aliyuncli.RunJSON(args, env, &data); err != nil {
		return 0, err
	}

	// Calculate average hit rate
	if len(data.HitRateData.HitRateValue) == 0 {
		return 0, nil
	}

	var totalHitRate float64
	for _, v := range data.HitRateData.HitRateValue {
		totalHitRate += v.HitRate
	}

	avgHitRate := totalHitRate / float64(len(data.HitRateData.HitRateValue))
	return avgHitRate, nil
}

// Usage queries CDN usage statistics for all domains
func Usage(env map[string]string, startTime, endTime string) ([]UsageData, error) {
	// Get domain list
	domains, err := List(env)
	if err != nil {
		return nil, err
	}

	res := make([]UsageData, 0, len(domains))
	for _, d := range domains {
		item := UsageData{
			DomainName: d.Name,
			Status:     d.Status,
		}

		// Query traffic data
		traffic, err := QueryTrafficData(env, d.Name, startTime, endTime)
		if err == nil {
			item.Traffic = traffic
			item.TrafficGB = traffic / (1024 * 1024 * 1024) // Convert to GB
		}

		// Query source traffic data
		srcTraffic, err := QuerySrcTrafficData(env, d.Name, startTime, endTime)
		if err == nil {
			item.SrcTraffic = srcTraffic
			item.SrcTrafficGB = srcTraffic / (1024 * 1024 * 1024) // Convert to GB
		}

		// Query hit rate data
		hitRate, err := QueryHitRateData(env, d.Name, startTime, endTime)
		if err == nil {
			item.HitRate = hitRate
		}

		res = append(res, item)
	}

	return res, nil
}

// FormatUsage formats CDN usage data for output
func FormatUsage(items []UsageData, output string) string {
	if output == "json" {
		b, _ := json.MarshalIndent(items, "", "  ")
		return string(b)
	}
	if len(items) == 0 {
		return "未找到 CDN 域名"
	}

	lines := []string{fmt.Sprintf("域名数量: %d", len(items)), ""}
	for _, it := range items {
		lines = append(lines, fmt.Sprintf("- %s", it.DomainName))
		lines = append(lines, fmt.Sprintf("  状态: %s", it.Status))
		
		// Format traffic (下行流量)
		if it.TrafficGB > 0 {
			lines = append(lines, fmt.Sprintf("  下行流量: %.2f GB", it.TrafficGB))
		} else {
			lines = append(lines, "  下行流量: 0")
		}
		
		// Format source traffic (回源流量)
		if it.SrcTrafficGB > 0 {
			lines = append(lines, fmt.Sprintf("  回源流量: %.2f GB", it.SrcTrafficGB))
		} else {
			lines = append(lines, "  回源流量: 0")
		}
		
		// Format hit rate (缓存命中率)
		if it.HitRate > 0 {
			lines = append(lines, fmt.Sprintf("  缓存命中率: %.2f%%", it.HitRate))
		} else {
			lines = append(lines, "  缓存命中率: 无访问")
		}
	}

	return strings.Join(lines, "\n")
}

func FormatDomains(items []Domain, output string) string {
	if output == "json" {
		b, _ := json.MarshalIndent(items, "", "  ")
		return string(b)
	}
	if len(items) == 0 {
		return "未找到 CDN 域名"
	}
	lines := []string{fmt.Sprintf("域名数量: %d", len(items)), ""}
	for _, it := range items {
		lines = append(lines, fmt.Sprintf("- %s", it.Name))
		lines = append(lines, fmt.Sprintf("  状态: %s | CNAME: %s", it.Status, it.Cname))
	}
	return strings.Join(lines, "\n")
}

func FormatAny(data map[string]any, output string) string {
	b, _ := json.MarshalIndent(data, "", "  ")
	return string(b)
}

func runAliyunRaw(args []string, env map[string]string) (string, error) {
	return aliyuncli.RunRaw(args, env)
}
