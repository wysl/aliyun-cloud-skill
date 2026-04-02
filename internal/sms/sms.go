package sms

import (
	"aliyun-cloud-monitor/internal/aliyuncli"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// SendStats represents SMS sending statistics
type SendStats struct {
	TotalCount            int64   `json:"totalCount"`            // 短信发送量(24h内) - 发送成功的短信条数
	RespondedSuccessCount int64   `json:"respondedSuccessCount"` // 回执成功条数
	RespondedFailCount    int64   `json:"respondedFailCount"`    // 回执失败条数
	NoRespondedCount      int64   `json:"noRespondedCount"`      // 未收到回执条数
	SuccessRate           float64 `json:"successRate"`           // 短信成功率(24h内)
}

// QuerySendStatistics queries SMS send statistics from Aliyun DysmsAPI
// Returns SMS sending statistics for the last 24 hours
func QuerySendStatistics(env map[string]string) (*SendStats, error) {
	// Calculate date range: last 24 hours (today and yesterday)
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	startDate := yesterday.Format("20060102")
	endDate := now.Format("20060102")

	args := []string{
		"dysmsapi", "query-send-statistics",
		"--endpoint", "dysmsapi.aliyuncs.com",
		"--api-version", "2017-05-25",
		"--is-globe", "1", // 国内短信
		"--start-date", startDate,
		"--end-date", endDate,
		"--page-index", "1",
		"--page-size", "50",
	}

	var data struct {
		Code      string `json:"Code"`
		Message   string `json:"Message"`
		RequestId string `json:"RequestId"`
		Data      struct {
			TotalSize int `json:"TotalSize"`
			TargetList []struct {
				TotalCount            int    `json:"TotalCount"`
				RespondedSuccessCount int    `json:"RespondedSuccessCount"`
				RespondedFailCount    int    `json:"RespondedFailCount"`
				NoRespondedCount      int    `json:"NoRespondedCount"`
				SendDate              string `json:"SendDate"`
			} `json:"TargetList"`
		} `json:"Data"`
	}

	if err := aliyuncli.RunJSON(args, env, &data); err != nil {
		return nil, fmt.Errorf("query SMS statistics failed: %w", err)
	}

	if data.Code != "OK" {
		return nil, fmt.Errorf("API error: %s - %s", data.Code, data.Message)
	}

	// Aggregate statistics from all days
	stats := &SendStats{}
	for _, item := range data.Data.TargetList {
		stats.TotalCount += int64(item.TotalCount)
		stats.RespondedSuccessCount += int64(item.RespondedSuccessCount)
		stats.RespondedFailCount += int64(item.RespondedFailCount)
		stats.NoRespondedCount += int64(item.NoRespondedCount)
	}

	// Calculate success rate
	// 成功率 = 回执成功条数 / (回执成功条数 + 回执失败条数) * 100%
	// 未回执的短信不计入失败
	respondedTotal := stats.RespondedSuccessCount + stats.RespondedFailCount
	if respondedTotal > 0 {
		stats.SuccessRate = float64(stats.RespondedSuccessCount) / float64(respondedTotal) * 100
	} else {
		stats.SuccessRate = 0
	}

	return stats, nil
}

// FormatStats formats SMS statistics for output
func FormatStats(stats *SendStats, output string) string {
	if output == "json" {
		b, _ := json.MarshalIndent(stats, "", "  ")
		return string(b)
	}
	if stats == nil {
		return "未获取到短信发送统计数据"
	}

	lines := []string{
		fmt.Sprintf("短信发送量(24h内): %d", stats.TotalCount),
		fmt.Sprintf("短信成功率(24h内): %.2f%%", stats.SuccessRate),
	}

	return strings.Join(lines, "\n")
}