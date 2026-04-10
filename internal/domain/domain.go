package domain

import (
	"aliyun-cloud-monitor/internal/aliyuncli"
	"aliyun-cloud-monitor/internal/sslcert"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Domain represents a domain name information
type Domain struct {
	Name       string `json:"name"`       // 域名名称
	Status     string `json:"status"`     // 域名状态
	ExpireDate string `json:"expireDate"` // 到期日期
	DaysLeft   int    `json:"daysLeft"`   // 剩余天数
	RegType    string `json:"regType"`    // 注册类型
	DomainType string `json:"domainType"` // 域名类型
}

// DomainWithCert represents a domain with its SSL certificate info
type DomainWithCert struct {
	Domain
	HasCert     bool   `json:"hasCert"`     // 是否有证书
	CertName    string `json:"certName"`    // 证书名称
	CertExpire  string `json:"certExpire"`  // 证书到期日期
	CertDaysLeft int   `json:"certDaysLeft"` // 证书剩余天数
	CertStatus  string `json:"certStatus"`  // 证书状态
}

// List queries domain list from Aliyun Domain API
func List(env map[string]string) ([]Domain, error) {
	args := []string{
		"domain", "query-domain-list",
		"--page-num", "1",
		"--page-size", "100",
	}

	var data struct {
		TotalItemNum int `json:"TotalItemNum"`
		Data struct {
			Domain []struct {
				DomainName       string `json:"DomainName"`
				DomainStatus     string `json:"DomainStatus"`
				DeadDate         string `json:"DeadDate"`
				DeadDateLong     int64  `json:"DeadDateLong"`
				DomainRegType    string `json:"DomainRegType"`
				DomainType       string `json:"DomainType"`
				DomainAuditStatus string `json:"DomainAuditStatus"`
			} `json:"Domain"`
		} `json:"Data"`
	}

	if err := aliyuncli.RunJSON(args, env, &data); err != nil {
		return nil, fmt.Errorf("query domain list failed: %w", err)
	}

	res := make([]Domain, 0, len(data.Data.Domain))
	now := time.Now()

	for _, it := range data.Data.Domain {
		expireDate := ""
		daysLeft := 0

		// Parse expire date from timestamp
		if it.DeadDateLong > 0 {
			t := time.Unix(it.DeadDateLong/1000, 0)
			expireDate = t.Format("2006-01-02")
			daysLeft = int(t.Sub(now).Hours() / 24)
		}

		// Map domain status, using DomainAuditStatus to correct false "审核中"
		status := mapDomainStatus(it.DomainStatus, it.DomainAuditStatus)

		res = append(res, Domain{
			Name:       it.DomainName,
			Status:     status,
			ExpireDate: expireDate,
			DaysLeft:   daysLeft,
			RegType:    it.DomainRegType,
			DomainType: it.DomainType,
		})
	}

	// Sort by days left
	sort.Slice(res, func(i, j int) bool {
		return res[i].DaysLeft < res[j].DaysLeft
	})

	return res, nil
}

// ListWithCerts queries domain list with matching SSL certificates (wildcard only)
func ListWithCerts(env map[string]string, showSize int) ([]DomainWithCert, error) {
	// Get domain list
	domains, err := List(env)
	if err != nil {
		return nil, err
	}

	// Get SSL certificate list
	certs, err := sslcert.List(env, showSize)
	if err != nil {
		// If SSL query fails, return domains without cert info
		res := make([]DomainWithCert, 0, len(domains))
		for _, d := range domains {
			res = append(res, DomainWithCert{
				Domain:     d,
				HasCert:    false,
			})
		}
		return res, nil
	}

	// Create a map for wildcard certificates only
	// Key is the base domain (without *. prefix)
	certMap := make(map[string]sslcert.Certificate)
	for _, cert := range certs {
		// Only process wildcard certificates (domain starts with "*.")
		if cert.Domain != "" && strings.HasPrefix(cert.Domain, "*.") {
			// Extract base domain (remove "*. " prefix)
			baseDomain := strings.TrimPrefix(cert.Domain, "*.")
			certMap[baseDomain] = cert
		}
	}

	// Match domains with wildcard certificates
	res := make([]DomainWithCert, 0, len(domains))
	for _, d := range domains {
		item := DomainWithCert{
			Domain:  d,
			HasCert: false,
		}

		// Check if domain has a matching wildcard certificate
		if cert, ok := certMap[d.Name]; ok {
			item.HasCert = true
			item.CertName = cert.Name
			item.CertExpire = cert.ExpireDate
			item.CertDaysLeft = cert.DaysLeft
			item.CertStatus = cert.Status
		}

		res = append(res, item)
	}

	return res, nil
}

func mapDomainStatus(status string, auditStatus string) string {
	// DomainAuditStatus takes precedence: if audit succeeded, domain is usable regardless of DomainStatus
	if strings.EqualFold(auditStatus, "SUCCEED") {
		return "正常"
	}

	switch status {
	case "1":
		return "正常"
	case "2":
		return "未实名"
	case "3":
		return "审核中"
	case "4":
		return "已过期"
	case "5":
		return "赎回期"
	default:
		return status
	}
}

// FormatList formats domain list for output
func FormatList(items []Domain, output string) string {
	if output == "json" {
		b, _ := json.MarshalIndent(items, "", "  ")
		return string(b)
	}
	if len(items) == 0 {
		return "未找到域名"
	}

	lines := []string{fmt.Sprintf("域名总数: %d", len(items)), ""}
	for _, it := range items {
		lines = append(lines, fmt.Sprintf("- %s", it.Name))
		lines = append(lines, fmt.Sprintf("  状态: %s | 类型: %s | 到期: %s | 剩余: %d 天", it.Status, it.DomainType, it.ExpireDate, it.DaysLeft))
	}

	return strings.Join(lines, "\n")
}

// FormatListWithCerts formats domain list with SSL certificates for output
func FormatListWithCerts(items []DomainWithCert, output string) string {
	if output == "json" {
		b, _ := json.MarshalIndent(items, "", "  ")
		return string(b)
	}
	if len(items) == 0 {
		return "未找到域名"
	}

	lines := []string{fmt.Sprintf("域名总数: %d", len(items)), ""}
	for _, it := range items {
		lines = append(lines, fmt.Sprintf("- %s", it.Name))
		lines = append(lines, fmt.Sprintf("  域名状态: %s | 类型: %s | 到期: %s | 剩余: %d 天", it.Status, it.DomainType, it.ExpireDate, it.DaysLeft))
		if it.HasCert {
			lines = append(lines, fmt.Sprintf("  SSL证书: %s | 状态: %s | 到期: %s | 剩余: %d 天", it.CertName, it.CertStatus, it.CertExpire, it.CertDaysLeft))
		} else {
			lines = append(lines, "  SSL证书: 无")
		}
	}

	return strings.Join(lines, "\n")
}