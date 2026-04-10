package main

import (
	"os"
	"path/filepath"

	"aliyun-cloud-skill/internal/app"
)

func main() {
	baseDir := mustBaseDir()
	a := app.New(baseDir)
	os.Exit(a.Run(os.Args[1:]))
}

func mustBaseDir() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	if _, err := os.Stat(filepath.Join(wd, "SKILL.md")); err == nil {
		return wd
	}
	if _, err := os.Stat(filepath.Join(wd, "scripts", "aliyunctl.go")); err == nil {
		return wd
	}
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		root := filepath.Dir(dir)
		if _, err := os.Stat(filepath.Join(root, "SKILL.md")); err == nil {
			return root
		}
	}
	return wd
}
