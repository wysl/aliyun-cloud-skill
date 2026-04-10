package billing

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"aliyun-cloud-skill/internal/aliyuncli"
)

type CategorySummary struct {
	Category string  `json:"category"`
	Total    float64 `json:"total"`
	Count    int     `json:"count"`
}

type Report struct {
	BillingCycle string            `json:"billingCycle"`
	Account      string            `json:"account"`
	Categories   []CategorySummary `json:"categories"`
	GrandTotal   float64           `json:"grandTotal"`
	RawCount     int               `json:"rawCount"`
}

// AccountBalance represents account balance information
type AccountBalance struct {
	Account           string  `json:"account"`
	AvailableAmount   float64 `json:"availableAmount"`
	AvailableCashAmount float64 `json:"availableCashAmount"`
	CreditAmount      float64 `json:"creditAmount"`
	MybankCreditAmount float64 `json:"mybankCreditAmount"`
	QuotaLimit        float64 `json:"quotaLimit"`
	Currency          string  `json:"currency"`
}

func QueryBill(account string, cycle string, env map[string]string) (Report, error) {
	args := []string{"bssopenapi", "QueryBillOverview", "--BillingCycle", cycle}
	out, err := aliyuncli.RunRaw(args, env)
	if err != nil {
		return Report{}, err
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(out), &data); err != nil {
		return Report{}, err
	}
	items := digItems(data)
	agg := map[string]*CategorySummary{}
	var total float64
	for _, item := range items {
		name := str(item["ProductName"])
		detail := str(item["ProductDetail"])
		code := str(item["CommodityCode"])
		cat := categorize(name, detail, code)
		cash := num(item["CashAmount"])
		total += cash
		if _, ok := agg[cat]; !ok {
			agg[cat] = &CategorySummary{Category: cat}
		}
		agg[cat].Total += cash
		agg[cat].Count++
	}
	cats := make([]CategorySummary, 0, len(agg))
	for _, v := range agg {
		cats = append(cats, *v)
	}
	sort.Slice(cats, func(i, j int) bool { return cats[i].Total > cats[j].Total })
	return Report{BillingCycle: cycle, Account: account, Categories: cats, GrandTotal: total, RawCount: len(items)}, nil
}

func Format(r Report, output string) string {
	if output == "json" {
		b, _ := json.MarshalIndent(r, "", "  ")
		return string(b)
	}
	lines := []string{
		fmt.Sprintf("账单账号: %s", r.Account),
		fmt.Sprintf("账期: %s", r.BillingCycle),
		fmt.Sprintf("条目数: %d", r.RawCount),
		fmt.Sprintf("总金额: %.2f", r.GrandTotal),
		"",
		"分类汇总:",
	}
	for _, c := range r.Categories {
		lines = append(lines, fmt.Sprintf("- %s: %.2f (%d项)", c.Category, c.Total, c.Count))
	}
	return strings.Join(lines, "\n")
}

func digItems(data map[string]any) []map[string]any {
	D, _ := data["Data"].(map[string]any)
	I1, _ := D["Items"].(map[string]any)
	raw := I1["Item"]
	arr, _ := raw.([]any)
	out := []map[string]any{}
	for _, v := range arr {
		if m, ok := v.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func categorize(productName, productDetail, commodityCode string) string {
	p := strings.ToLower(productName)
	d := strings.ToLower(productDetail)
	c := strings.ToLower(commodityCode)
	contains := func(s string, kws ...string) bool {
		for _, kw := range kws {
			if strings.Contains(s, kw) {
				return true
			}
		}
		return false
	}
	switch {
	case contains(p, "证书", "ssl", "cas", "安全", "waf"):
		return "安全"
	case contains(p, "ecs", "云服务器", "服务器", "容器", "k8s", "ack") || contains(c, "vm", "ecs", "snapshot", "disk") || contains(d, "快照", "云服务器", "ecs"):
		return "计算"
	case contains(p, "rds", "数据库", "polardb", "mysql", "redis", "mongodb") || contains(c, "rds", "polardb", "redis", "mongodb"):
		return "数据库"
	case contains(p, "负载均衡", "slb", "alb", "nlb", "cdn", "eip", "带宽", "网络", "流量", "dcdn") || contains(c, "slb", "alb", "nlb", "cdn", "eip", "cbwp", "flowbag", "dcdn"):
		return "网络"
	case contains(p, "oss", "对象存储", "存储", "nas", "文件存储") || contains(c, "oss", "nas"):
		return "存储"
	case contains(p, "监控", "grafana", "arms", "日志", "sls", "log") || contains(c, "cms", "grafana", "arms", "sls"):
		return "监控"
	case contains(p, "短信", "sms", "消息", "mns") || contains(c, "dysms", "sms", "mns"):
		return "通信"
	case contains(p, "大模型", "百炼", "ai", "模型", "sfm"):
		return "AI/大模型"
	default:
		return "其他"
	}
}

func str(v any) string { if s, ok := v.(string); ok { return s }; return "" }
func num(v any) float64 {
	s := fmt.Sprintf("%v", v)
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// parseMoney parses money string with comma (e.g., "4,219.67")
func parseMoney(s string) float64 {
	s = strings.ReplaceAll(s, ",", "")
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// QueryAccountBalance queries account balance via BSS API
func QueryAccountBalance(account string, env map[string]string) (AccountBalance, error) {
	args := []string{"bssopenapi", "QueryAccountBalance"}
	out, err := aliyuncli.RunRaw(args, env)
	if err != nil {
		return AccountBalance{}, err
	}
	var data struct {
		Success bool   `json:"Success"`
		Code    string `json:"Code"`
		Message string `json:"Message"`
		Data    struct {
			AvailableAmount     string `json:"AvailableAmount"`
			AvailableCashAmount string `json:"AvailableCashAmount"`
			CreditAmount        string `json:"CreditAmount"`
			MybankCreditAmount  string `json:"MybankCreditAmount"`
			QuotaLimit          string `json:"QuotaLimit"`
			Currency            string `json:"Currency"`
		} `json:"Data"`
	}
	if err := json.Unmarshal([]byte(out), &data); err != nil {
		return AccountBalance{}, err
	}
	if !data.Success {
		return AccountBalance{}, fmt.Errorf("API error: %s", data.Message)
	}
	return AccountBalance{
		Account:             account,
		AvailableAmount:     parseMoney(data.Data.AvailableAmount),
		AvailableCashAmount: parseMoney(data.Data.AvailableCashAmount),
		CreditAmount:        parseMoney(data.Data.CreditAmount),
		MybankCreditAmount:  parseMoney(data.Data.MybankCreditAmount),
		QuotaLimit:          parseMoney(data.Data.QuotaLimit),
		Currency:            data.Data.Currency,
	}, nil
}

// FormatBalance formats account balance for display
func FormatBalance(b AccountBalance, output string) string {
	if output == "json" {
		bb, _ := json.MarshalIndent(b, "", "  ")
		return string(bb)
	}
	lines := []string{
		fmt.Sprintf("账户: %s", b.Account),
		fmt.Sprintf("可用余额: %.2f %s", b.AvailableAmount, b.Currency),
		fmt.Sprintf("可用现金: %.2f %s", b.AvailableCashAmount, b.Currency),
		fmt.Sprintf("信用额度: %.2f %s", b.CreditAmount, b.Currency),
	}
	return strings.Join(lines, "\n")
}
