package aliyuncli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func MergeEnv(base []string, kv map[string]string) []string {
	out := append([]string{}, base...)
	for k, v := range kv {
		out = append(out, k+"="+v)
	}
	return out
}

func RunRaw(args []string, env map[string]string) (string, error) {
	cmd := exec.Command("aliyun", args...)
	cmd.Env = MergeEnv(os.Environ(), env)
	body, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("aliyun %s failed: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(string(body)))
	}
	return string(body), nil
}

func RunJSON(args []string, env map[string]string, out any) error {
	body, err := RunRaw(args, env)
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(body), out); err != nil {
		return fmt.Errorf("invalid aliyun JSON for %s: %w", strings.Join(args, " "), err)
	}
	return nil
}
