package resourcecenter

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

type Summary struct {
	Total     int                 `json:"total"`
	ByType    map[string]int      `json:"byType"`
	ByRegion  map[string]int      `json:"byRegion"`
	Resources []map[string]string `json:"resources,omitempty"`
}

func Search(env map[string]string, maxPages int) (Summary, error) {
	byType := map[string]int{}
	byRegion := map[string]int{}
	resources := []map[string]string{}
	nextToken := ""

	for page := 0; page < maxPages; page++ {
		args := []string{"resourcecenter", "SearchResources", "--MaxResults", "100"}
		if nextToken != "" {
			args = append(args, "--NextToken", nextToken)
		}
		cmd := exec.Command("aliyun", args...)
		cmd.Env = mergeEnv(os.Environ(), env)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return Summary{}, fmt.Errorf("aliyun search failed: %v: %s", err, strings.TrimSpace(string(out)))
		}

		var data struct {
			NextToken string `json:"NextToken"`
			Resources []struct {
				ResourceType string `json:"ResourceType"`
				RegionId     string `json:"RegionId"`
				ResourceId   string `json:"ResourceId"`
				ResourceName string `json:"ResourceName"`
			} `json:"Resources"`
		}
		if err := json.Unmarshal(out, &data); err != nil {
			return Summary{}, fmt.Errorf("invalid JSON from aliyun: %w", err)
		}

		for _, r := range data.Resources {
			rt := r.ResourceType
			if rt == "" {
				rt = "Unknown"
			}
			rg := r.RegionId
			if rg == "" {
				rg = "global"
			}
			byType[rt]++
			byRegion[rg]++
			resources = append(resources, map[string]string{
				"type":   rt,
				"region": rg,
				"id":     r.ResourceId,
				"name":   r.ResourceName,
			})
		}

		if data.NextToken == "" {
			break
		}
		nextToken = data.NextToken
	}

	sort.Slice(resources, func(i, j int) bool {
		a := resources[i]["type"] + "/" + resources[i]["region"] + "/" + resources[i]["id"]
		b := resources[j]["type"] + "/" + resources[j]["region"] + "/" + resources[j]["id"]
		return a < b
	})

	total := 0
	for _, c := range byType {
		total += c
	}
	return Summary{Total: total, ByType: byType, ByRegion: byRegion, Resources: resources}, nil
}

func mergeEnv(base []string, kv map[string]string) []string {
	out := append([]string{}, base...)
	for k, v := range kv {
		out = append(out, k+"="+v)
	}
	return out
}
