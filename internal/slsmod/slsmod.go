package slsmod

import (
	"aliyun-cloud-skill/internal/aliyuncli"
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

// CreateProject creates a new SLS project
func CreateProject(env map[string]string, projectName, description string) (string, error) {
	body := map[string]string{
		"projectName": projectName,
		"description": description,
	}
	bodyJson, _ := json.Marshal(body)
	return runAliyunRaw([]string{"sls", "CreateProject", "--body", string(bodyJson)}, env)
}

// CreateLogStore creates a new logstore in a project
func CreateLogStore(env map[string]string, project, logstoreName string, ttl, shardCount int) (string, error) {
	body := map[string]any{
		"logstoreName": logstoreName,
		"ttl":          ttl,
		"shardCount":   shardCount,
		"autoSplit":    true,
		"maxSplitShard": shardCount * 2,
		"telemetryType": "None",
	}
	bodyJson, _ := json.Marshal(body)
	return runAliyunRaw([]string{"sls", "CreateLogStore", "--project", project, "--body", string(bodyJson)}, env)
}

// ListMachineGroup lists machine groups in a project
func ListMachineGroup(env map[string]string, project string) (string, error) {
	return runAliyunRaw([]string{"sls", "ListMachineGroup", "--project", project}, env)
}

// GetMachineGroup gets machine group details
func GetMachineGroup(env map[string]string, project, machineGroupName string) (string, error) {
	return runAliyunRaw([]string{"sls", "GetMachineGroup", "--project", project, "--machineGroup", machineGroupName}, env)
}

// CreateMachineGroup creates a machine group
func CreateMachineGroup(env map[string]string, project, machineGroupName string, machineList []string) (string, error) {
	body := map[string]any{
		"groupName":            machineGroupName,
		"machineIdentifyType":  "ip",
		"machineList":          machineList,
		"groupType":            "",
		"groupAttribute":       map[string]string{},
	}
	bodyJson, _ := json.Marshal(body)
	return runAliyunRaw([]string{"sls", "CreateMachineGroup", "--project", project, "--body", string(bodyJson)}, env)
}

// CreateConfig creates a logtail config for file collection
func CreateConfig(env map[string]string, project, configName, logPath, logPattern, logstore string) (string, error) {
	body := map[string]any{
		"configName": configName,
		"inputType":  "file",
		"inputDetail": map[string]any{
			"logType":         "json_log",
			"logPath":         logPath,
			"filePattern":     logPattern,
			"localStorage":    true,
			"logBeginRegex":   ".*",
			"fileEncoding":    "utf8",
			"discardUnmatch":  false,
			"maxDepth":        10,
			"topicFormat":     "none",
			"preserve":        true,
			"preserveDepth":   0,
			"tailExisted":     false,
			"dockerFile":      false,
			"enableRawLog":    false,
			"adjustTimezone":  false,
			"discardNonUtf8":  false,
			"filterKey":       []string{},
			"filterRegex":     []string{},
			"sensitive_keys":  []string{},
			"timeKey":         "@timestamp",
			"timeFormat":      "%Y-%m-%dT%H:%M:%S.%f%z",
		},
		"outputType": "LogService",
		"outputDetail": map[string]string{
			"logstoreName": logstore,
		},
	}
	bodyJson, _ := json.Marshal(body)
	return runAliyunRaw([]string{"sls", "CreateConfig", "--project", project, "--body", string(bodyJson)}, env)
}

// ApplyConfigToMachineGroup applies a config to a machine group
func ApplyConfigToMachineGroup(env map[string]string, project, machineGroupName, configName string) (string, error) {
	return runAliyunRaw([]string{"sls", "ApplyConfigToMachineGroup", "--project", project, "--machineGroup", machineGroupName, "--configName", configName}, env)
}

func runAliyunRaw(args []string, env map[string]string) (string, error) {
	return aliyuncli.RunRaw(args, env)
}
