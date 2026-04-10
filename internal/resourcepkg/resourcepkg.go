package resourcepkg

import (
	"aliyun-cloud-skill/internal/aliyuncli"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Package struct {
	ID         string  `json:"id"`
	Code       string  `json:"code"`
	Remark     string  `json:"remark"`
	Status     string  `json:"status"`
	Total      float64 `json:"total"`
	Remaining  float64 `json:"remaining"`
	Unit       string  `json:"unit"`
	UsageRate  float64 `json:"usageRate"`
	ExpiryTime string  `json:"expiryTime"`
	DaysLeft   int     `json:"daysLeft"`
}

func List(env map[string]string) ([]Package, error) {
	args := []string{"bssopenapi", "query-resource-package-instances"}
	var data struct {
		Data struct {
			TotalCount int `json:"TotalCount"`
			Instances struct {
				Instance []struct {
					InstanceId string `json:"InstanceId"`
					CommodityCode string `json:"CommodityCode"`
					Remark string `json:"Remark"`
					Status string `json:"Status"`
					TotalAmount any `json:"TotalAmount"`
					RemainingAmount any `json:"RemainingAmount"`
					TotalAmountUnit string `json:"TotalAmountUnit"`
					ExpiryTime string `json:"ExpiryTime"`
				} `json:"Instance"`
			} `json:"Instances"`
		} `json:"Data"`
	}
	if err := aliyuncli.RunJSON(args, env, &data); err != nil { return nil, err }
	now := time.Now().UTC()
	res := make([]Package, 0, len(data.Data.Instances.Instance))
	for _, it := range data.Data.Instances.Instance {
		total := toFloat(it.TotalAmount)
		remaining := toFloat(it.RemainingAmount)
		usage := 0.0
		if total > 0 { usage = ((total - remaining) / total) * 100 }
		days := 0
		if strings.TrimSpace(it.ExpiryTime) != "" {
			if t, err := time.Parse(time.RFC3339, strings.ReplaceAll(it.ExpiryTime, "Z", "+00:00")); err == nil {
				days = int(t.Sub(now).Hours() / 24)
			}
		}
		res = append(res, Package{ID: it.InstanceId, Code: it.CommodityCode, Remark: it.Remark, Status: it.Status, Total: total, Remaining: remaining, Unit: it.TotalAmountUnit, UsageRate: usage, ExpiryTime: it.ExpiryTime, DaysLeft: days})
	}
	sort.Slice(res, func(i, j int) bool { return res[i].DaysLeft < res[j].DaysLeft })
	return res, nil
}

func Expiring(items []Package, days int) []Package {
	out := []Package{}
	for _, it := range items { if it.DaysLeft <= days { out = append(out, it) } }
	return out
}

func Summary(items []Package) map[string]any {
	expired := 0
	expiring30 := 0
	exhausted := 0
	for _, it := range items {
		if it.DaysLeft <= 0 { expired++ } else if it.DaysLeft <= 30 { expiring30++ }
		if it.Remaining <= 0 { exhausted++ }
	}
	return map[string]any{"total": len(items), "expired": expired, "expiring30": expiring30, "exhausted": exhausted}
}

// GroupedPackage represents merged resource packages by name
type GroupedPackage struct {
	Name       string
	Count      int
	IDs        []string
	Total      float64
	Remaining  float64
	Unit       string
	UsageRate  float64
	MaxDays    int
	Status     string
}

// GroupByName merges packages with the same name (Remark or Code)
func GroupByName(items []Package) []GroupedPackage {
	groups := make(map[string]*GroupedPackage)
	for _, it := range items {
		name := it.Remark
		if strings.TrimSpace(name) == "" { name = it.Code }
		if g, ok := groups[name]; ok {
			g.Count++
			g.IDs = append(g.IDs, it.ID)
			g.Total += it.Total
			g.Remaining += it.Remaining
			if it.DaysLeft > g.MaxDays { g.MaxDays = it.DaysLeft }
		} else {
			groups[name] = &GroupedPackage{
				Name:      name,
				Count:     1,
				IDs:       []string{it.ID},
				Total:     it.Total,
				Remaining: it.Remaining,
				Unit:      it.Unit,
				MaxDays:   it.DaysLeft,
				Status:    it.Status,
			}
		}
	}
	// Calculate usage rate for grouped packages
	result := make([]GroupedPackage, 0, len(groups))
	for _, g := range groups {
		if g.Total > 0 { g.UsageRate = ((g.Total - g.Remaining) / g.Total) * 100 }
		result = append(result, *g)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].MaxDays < result[j].MaxDays })
	return result
}

func FormatList(items []Package, output string) string {
	if output == "json" { b, _ := json.MarshalIndent(items, "", "  "); return string(b) }
	if len(items) == 0 { return "未找到资源包实例" }
	
	// Group packages by name for summary output
	grouped := GroupByName(items)
	
	lines := []string{fmt.Sprintf("资源包总数: %d (%d种)", len(items), len(grouped)), ""}
	for _, g := range grouped {
		countStr := ""
		if g.Count > 1 { countStr = fmt.Sprintf(" (%d个实例)", g.Count) }
		lines = append(lines, fmt.Sprintf("- %s%s", g.Name, countStr))
		if g.Count == 1 {
			lines = append(lines, fmt.Sprintf("  ID: %s", g.IDs[0]))
		} else {
			lines = append(lines, fmt.Sprintf("  IDs: %s", strings.Join(g.IDs, ", ")))
		}
		// Unified unit display (TB -> GB)
		remaining, total, unit := normalizeUnit(g.Remaining, g.Total, g.Unit)
		lines = append(lines, fmt.Sprintf("  余量/总量: %.2f/%.2f %s | 剩余天数: %d", remaining, total, unit, g.MaxDays))
	}
	return strings.Join(lines, "\n")
}

// normalizeUnit converts TB to GB for unified display
// Note: For TB units, API returns Total in TB but Remaining in GB
func normalizeUnit(remaining, total float64, unit string) (float64, float64, string) {
	lowerUnit := strings.ToLower(unit)
	if lowerUnit == "tb" || lowerUnit == "1tb" {
		// Total is in TB, convert to GB; Remaining is already in GB
		return remaining, total * 1024, "GB"
	}
	return remaining, total, unit
}

func FormatSummary(summary map[string]any, output string) string {
	b, _ := json.MarshalIndent(summary, "", "  ")
	if output == "json" { return string(b) }
	return fmt.Sprintf("资源包总数: %v\n已过期: %v\n30天内过期: %v\n已耗尽: %v", summary["total"], summary["expired"], summary["expiring30"], summary["exhausted"])
}

func toFloat(v any) float64 { f, _ := strconv.ParseFloat(fmt.Sprintf("%v", v), 64); return f }
