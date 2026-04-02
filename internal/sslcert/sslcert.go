package sslcert

import (
	"aliyun-cloud-monitor/internal/aliyuncli"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type Certificate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Domain      string `json:"domain"`
	Brand       string `json:"brand"`
	Status      string `json:"status"`
	InstanceType string `json:"instanceType"`
	AutoReissue string `json:"autoReissue"`
	ExpireDate  string `json:"expireDate"`
	DaysLeft    int    `json:"daysLeft"`
}

func List(env map[string]string, showSize int) ([]Certificate, error) {
	args := []string{"cas", "list-instances", "--endpoint", "cas.aliyuncs.com", "--region", "cn-hangzhou", "--show-size", fmt.Sprintf("%d", showSize)}
	var data struct {
		TotalCount int `json:"TotalCount"`
		InstanceList []struct {
			CertificateId any `json:"CertificateId"`
			CertificateName string `json:"CertificateName"`
			Domain string `json:"Domain"`
			Brand string `json:"Brand"`
			CertificateStatus string `json:"CertificateStatus"`
			InstanceType string `json:"InstanceType"`
			AutoReissue string `json:"AutoReissue"`
			CertificateNotAfter float64 `json:"CertificateNotAfter"`
		} `json:"InstanceList"`
	}
	if err := aliyuncli.RunJSON(args, env, &data); err != nil { return nil, err }
	res := make([]Certificate, 0, len(data.InstanceList))
	now := time.Now().UTC()
	for _, it := range data.InstanceList {
		expire := ""
		days := 0
		if it.CertificateNotAfter > 0 {
			t := time.Unix(int64(it.CertificateNotAfter/1000), 0).UTC()
			expire = t.Format("2006-01-02")
			days = int(t.Sub(now).Hours() / 24)
		}
		res = append(res, Certificate{ID: fmt.Sprintf("%v", it.CertificateId), Name: it.CertificateName, Domain: it.Domain, Brand: it.Brand, Status: it.CertificateStatus, InstanceType: it.InstanceType, AutoReissue: it.AutoReissue, ExpireDate: expire, DaysLeft: days})
	}
	sort.Slice(res, func(i, j int) bool { return res[i].DaysLeft < res[j].DaysLeft })
	return res, nil
}

func Expiring(items []Certificate, days int) []Certificate {
	out := []Certificate{}
	for _, it := range items { if it.DaysLeft <= days { out = append(out, it) } }
	return out
}

func Summary(items []Certificate) map[string]any {
	expired := 0
	expiring30 := 0
	expiring60 := 0
	buy := 0
	test := 0
	for _, it := range items {
		if it.InstanceType == "BUY" { buy++ }
		if it.InstanceType == "TEST" { test++ }
		switch {
		case it.DaysLeft <= 0:
			expired++
		case it.DaysLeft <= 30:
			expiring30++
		case it.DaysLeft <= 60:
			expiring60++
		}
	}
	return map[string]any{"total": len(items), "buy": buy, "test": test, "expired": expired, "expiring30": expiring30, "expiring60": expiring60}
}

func FormatList(items []Certificate, output string) string {
	if output == "json" { b, _ := json.MarshalIndent(items, "", "  "); return string(b) }
	if len(items) == 0 { return "未找到 SSL 证书实例" }
	lines := []string{fmt.Sprintf("SSL 证书总数: %d", len(items)), ""}
	for _, it := range items {
		lines = append(lines, fmt.Sprintf("- %s (ID: %s)", it.Name, it.ID))
		lines = append(lines, fmt.Sprintf("  域名: %s | 品牌: %s | 状态: %s", it.Domain, it.Brand, it.Status))
		lines = append(lines, fmt.Sprintf("  类型: %s | 自动续期: %s | 过期: %s | 剩余: %d天", it.InstanceType, it.AutoReissue, it.ExpireDate, it.DaysLeft))
	}
	return strings.Join(lines, "\n")
}

func FormatSummary(summary map[string]any, output string) string {
	b, _ := json.MarshalIndent(summary, "", "  ")
	if output == "json" { return string(b) }
	return fmt.Sprintf("证书总数: %v\n正式购买: %v\n测试证书: %v\n已过期: %v\n30天内到期: %v\n60天内到期: %v", summary["total"], summary["buy"], summary["test"], summary["expired"], summary["expiring30"], summary["expiring60"])
}

