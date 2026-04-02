package slsmod

import (
	"aliyun-cloud-monitor/internal/aliyuncli"
	"encoding/json"
	"fmt"
	"strings"
)

func ListProjects(env map[string]string) (string, error) {
	return runAliyunRaw([]string{"sls", "ListProject"}, env)
}

func ListLogstores(env map[string]string, project string) (string, error) {
	return runAliyunRaw([]string{"sls", "ListLogStores", "--project", project}, env)
}

func GetIndex(env map[string]string, project, logstore string) (string, error) {
	return runAliyunRaw([]string{"sls", "GetIndex", "--project", project, "--logstore", logstore}, env)
}

func UpdateIndexAddContent(env map[string]string, project, logstore string) (string, error) {
	body := `{"line":{"token":[","," ","\n","\t",":","=","\"","'",";","[","]","{","}","(",")","&","^","*","#","@","~",">","<","/","\\","?"],"caseSensitive":false,"chn":false},"keys":{"content":{"type":"text","token":[","," ","\n","\t",":","=","\"","'",";","[","]","{","}","(",")","&","^","*","#","@","~",">","<","/","\\","?"],"caseSensitive":false,"chn":false,"doc_value":true}}}`
	return runAliyunRaw([]string{"sls", "UpdateIndex", "--project", project, "--logstore", logstore, "--body", body}, env)
}

func GetLogs(env map[string]string, project, logstore, from, to, query string) (string, error) {
	args := []string{"sls", "GetLogs", "--project", project, "--logstore", logstore, "--from", from, "--to", to, "--query", query, "--line", "100"}
	return runAliyunRaw(args, env)
}

func QueryIPs(env map[string]string, project, logstore, from, to, query string) (string, error) {
	body, err := GetLogs(env, project, logstore, from, to, query)
	if err != nil { return "", err }
	var data []map[string]any
	if err := json.Unmarshal([]byte(body), &data); err != nil { return body, nil }
	ips := map[string]int{}
	for _, item := range data {
		content := fmt.Sprintf("%v", item["content"])
		for _, field := range strings.Fields(content) {
			if strings.Count(field, ".") == 3 { ips[field]++ }
		}
	}
	b, _ := json.MarshalIndent(map[string]any{"totalLogs": len(data), "ips": ips}, "", "  ")
	return string(b), nil
}

func runAliyunRaw(args []string, env map[string]string) (string, error) {
	return aliyuncli.RunRaw(args, env)
}
