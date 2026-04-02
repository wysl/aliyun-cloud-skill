package prom

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func Query(baseURL, user, pass, promql string, datasource int, timeoutSec int) (map[string]any, error) {
	u := fmt.Sprintf("%s/api/datasources/proxy/%d/api/v1/query?%s", strings.TrimRight(baseURL, "/"), datasource, url.Values{"query": []string{promql}}.Encode())
	return doRequest(u, user, pass, timeoutSec)
}

func QueryRange(baseURL, user, pass, promql, start, end, step string, datasource int, timeoutSec int) (map[string]any, error) {
	vals := url.Values{"query": []string{promql}, "start": []string{start}, "end": []string{end}, "step": []string{step}}
	u := fmt.Sprintf("%s/api/datasources/proxy/%d/api/v1/query_range?%s", strings.TrimRight(baseURL, "/"), datasource, vals.Encode())
	return doRequest(u, user, pass, timeoutSec)
}

func Labels(baseURL, user, pass string, datasource int, timeoutSec int) (map[string]any, error) {
	u := fmt.Sprintf("%s/api/datasources/proxy/%d/api/v1/labels", strings.TrimRight(baseURL, "/"), datasource)
	return doRequest(u, user, pass, timeoutSec)
}

func LabelValues(baseURL, user, pass, label string, datasource int, timeoutSec int) (map[string]any, error) {
	u := fmt.Sprintf("%s/api/datasources/proxy/%d/api/v1/label/%s/values", strings.TrimRight(baseURL, "/"), datasource, url.PathEscape(label))
	return doRequest(u, user, pass, timeoutSec)
}

func Series(baseURL, user, pass, match string, datasource int, timeoutSec int) (map[string]any, error) {
	vals := url.Values{}
	if strings.TrimSpace(match) != "" { vals.Add("match[]", match) }
	u := fmt.Sprintf("%s/api/datasources/proxy/%d/api/v1/series", strings.TrimRight(baseURL, "/"), datasource)
	if enc := vals.Encode(); enc != "" { u += "?" + enc }
	return doRequest(u, user, pass, timeoutSec)
}

func doRequest(u, user, pass string, timeoutSec int) (map[string]any, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil { return nil, err }
	token := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
	req.Header.Set("Authorization", "Basic "+token)
	client := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}
	resp, err := client.Do(req)
	if err != nil { return nil, err }
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 { return nil, fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(body))) }
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil { return nil, err }
	return data, nil
}

func FormatResult(data map[string]any, output string, query string) string {
	if output == "json" { b, _ := json.MarshalIndent(data, "", "  "); return string(b) }
	if data["status"] != "success" { b, _ := json.MarshalIndent(data, "", "  "); return string(b) }
	d, _ := data["data"].(map[string]any)
	rt, _ := d["resultType"].(string)
	result, _ := d["result"].([]any)
	lines := []string{fmt.Sprintf("Query: %s", query), fmt.Sprintf("Result type: %s", rt), fmt.Sprintf("Series count: %d", len(result)), ""}
	for i, raw := range result {
		if i >= 20 { lines = append(lines, fmt.Sprintf("... and %d more", len(result)-20)); break }
		item, _ := raw.(map[string]any)
		metric, _ := item["metric"].(map[string]any)
		labels := []string{}
		for k, v := range metric { if k == "__name__" { continue }; labels = append(labels, fmt.Sprintf("%s=%q", k, fmt.Sprintf("%v", v))) }
		name := fmt.Sprintf("%v", metric["__name__"]); if name == "<nil>" || name == "" { name = "metric" }
		if val, ok := item["value"].([]any); ok && len(val) > 1 {
			lines = append(lines, fmt.Sprintf("- %s{%s}: %v", name, strings.Join(labels, ", "), val[1]))
		} else if vals, ok := item["values"].([]any); ok {
			lines = append(lines, fmt.Sprintf("- %s{%s}: %d points", name, strings.Join(labels, ", "), len(vals)))
		}
	}
	return strings.Join(lines, "\n")
}

func FormatLabels(data map[string]any, output string) string {
	if output == "json" { b, _ := json.MarshalIndent(data, "", "  "); return string(b) }
	arr, _ := data["data"].([]any)
	items := []string{}
	for i, v := range arr { if i >= 100 { break }; items = append(items, fmt.Sprintf("%v", v)) }
	return fmt.Sprintf("Labels: %d found\n%s", len(arr), strings.Join(items, ", "))
}

func FormatSeries(data map[string]any, output string) string {
	if output == "json" { b, _ := json.MarshalIndent(data, "", "  "); return string(b) }
	arr, _ := data["data"].([]any)
	lines := []string{fmt.Sprintf("Series: %d", len(arr)), ""}
	for i, raw := range arr { if i >= 30 { lines = append(lines, fmt.Sprintf("... and %d more", len(arr)-30)); break }; b, _ := json.Marshal(raw); lines = append(lines, string(b)) }
	return strings.Join(lines, "\n")
}
