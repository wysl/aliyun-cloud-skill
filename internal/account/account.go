package account

import "path/filepath"

type Paths struct {
	BaseDir string
}

func New(baseDir string) Paths { return Paths{BaseDir: baseDir} }
func (p Paths) AccountsDir() string { return filepath.Join(p.BaseDir, "accounts") }
func (p Paths) AccountDir(account string) string { return filepath.Join(p.AccountsDir(), account) }
func (p Paths) SecretsDir(account string) string { return filepath.Join(p.AccountDir(account), "secrets") }
func (p Paths) ListDir(account string) string { return filepath.Join(p.AccountDir(account), "list") }
func (p Paths) ReportsDir(account string) string { return filepath.Join(p.AccountDir(account), "reports") }
func (p Paths) EnvPath(account string) string { return filepath.Join(p.SecretsDir(account), "runtime.env") }
func (p Paths) CachePath(account string) string { return filepath.Join(p.SecretsDir(account), "cache.json") }
