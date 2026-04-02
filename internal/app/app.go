package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aliyun-cloud-monitor/internal/account"
	"aliyun-cloud-monitor/internal/envfile"
	"aliyun-cloud-monitor/internal/resourcecenter"
	"aliyun-cloud-monitor/internal/billing"
	"aliyun-cloud-monitor/internal/ecs"
	"aliyun-cloud-monitor/internal/securitygroup"
	"aliyun-cloud-monitor/internal/vpc"
	"aliyun-cloud-monitor/internal/prom"
	"aliyun-cloud-monitor/internal/rds"
	"aliyun-cloud-monitor/internal/polardb"
	"aliyun-cloud-monitor/internal/alb"
	"aliyun-cloud-monitor/internal/sms"
	"aliyun-cloud-monitor/internal/domain"
	"aliyun-cloud-monitor/internal/sslcert"
	"aliyun-cloud-monitor/internal/resourcepkg"
	"aliyun-cloud-monitor/internal/ossmod"
	"aliyun-cloud-monitor/internal/cdnmod"
	"aliyun-cloud-monitor/internal/slsmod"
)

type App struct {
	BaseDir string
	Paths   account.Paths
}

type Cache struct {
	AccountName  string            `json:"accountName"`
	RegionIDs    []string          `json:"regionIds"`
	Grafana      map[string]string `json:"grafana"`
	Feishu       map[string]bool   `json:"feishu"`
	Fingerprints map[string]string `json:"fingerprints"`
	Notes        []string          `json:"notes"`
}

func New(baseDir string) App {
	return App{BaseDir: baseDir, Paths: account.New(baseDir)}
}

func (a App) Run(args []string) int {
	if len(args) < 1 {
		Usage()
		return 2
	}
	var err error
	switch args[0] {
	case "init-account":
		err = a.InitAccount(args[1:])
	case "fix-permissions":
		err = a.FixPermissions(args[1:])
	case "hash-secrets":
		err = a.HashSecrets(args[1:])
	case "refresh":
		err = a.Refresh(args[1:])
	case "env-check":
		err = a.EnvCheck(args[1:])
	case "bill-summary":
		err = a.BillSummary(args[1:])
	case "account-balance":
		err = a.AccountBalance(args[1:])
	case "ecs-list":
		err = a.ECSList(args[1:])
	case "prom-query":
		err = a.PromQuery(args[1:])
	case "ecs-detail":
		err = a.ECSDetail(args[1:])
	case "ecs-start":
		err = a.ECSStart(args[1:])
	case "ecs-stop":
		err = a.ECSStop(args[1:])
	case "ecs-reboot":
		err = a.ECSReboot(args[1:])
	case "ecs-usage":
		err = a.ECSUsage(args[1:])
	case "prom-range":
		err = a.PromRange(args[1:])
	case "prom-labels":
		err = a.PromLabels(args[1:])
	case "prom-label-values":
		err = a.PromLabelValues(args[1:])
	case "prom-series":
		err = a.PromSeries(args[1:])
	case "rds-list":
		err = a.RDSList(args[1:])
	case "rds-usage":
		err = a.RDSUsage(args[1:])
	case "rds-detail":
		err = a.RDSDetail(args[1:])
	case "rds-performance":
		err = a.RDSPerformance(args[1:])
	case "rds-list-backups":
		err = a.RDSListBackups(args[1:])
	case "polardb-list":
		err = a.PolarDBList(args[1:])
	case "polardb-usage":
		err = a.PolarDBUsage(args[1:])
	case "alb-list":
		err = a.ALBList(args[1:])
	case "alb-usage":
		err = a.ALBUsage(args[1:])
	case "alb-acl-list":
		err = a.ALBAclList(args[1:])
	case "alb-acl-entries":
		err = a.ALBAclEntries(args[1:])
	case "alb-listener-acl":
		err = a.ALBListenerAcl(args[1:])
	case "vpc-list":
		err = a.VPCList(args[1:])
	case "vpc-detail":
		err = a.VPCDetail(args[1:])
	case "vswitch-list":
		err = a.VSwitchList(args[1:])
	case "routetable-list":
		err = a.RouteTableList(args[1:])
	case "vswitch-resources":
		err = a.VSwitchResources(args[1:])
	case "sms-stats":
		err = a.SMSStats(args[1:])
	case "sg-list":
		err = a.SGList(args[1:])
	case "sg-rules":
		err = a.SGRules(args[1:])
	case "sg-add-ingress":
		err = a.SGAddIngress(args[1:])
	case "sg-add-egress":
		err = a.SGAddEgress(args[1:])
	case "sg-join":
		err = a.SGJoin(args[1:])
	case "sg-leave":
		err = a.SGLeave(args[1:])
	case "domain-list":
		err = a.DomainList(args[1:])
	case "ssl-list":
		err = a.SSLList(args[1:])
	case "ssl-expiring":
		err = a.SSLExpiring(args[1:])
	case "ssl-summary":
		err = a.SSLSummary(args[1:])
	case "resource-package-list":
		err = a.ResourcePackageList(args[1:])
	case "resource-package-expiring":
		err = a.ResourcePackageExpiring(args[1:])
	case "resource-package-summary":
		err = a.ResourcePackageSummary(args[1:])
	case "oss-list":
		err = a.OSSList(args[1:])
	case "oss-info":
		err = a.OSSInfo(args[1:])
	case "oss-ls":
		err = a.OSSLS(args[1:])
	case "oss-recent":
		err = a.OSSRecent(args[1:])
	case "oss-usage":
		err = a.OSSUsage(args[1:])
	case "cdn-list":
		err = a.CDNList(args[1:])
	case "cdn-detail":
		err = a.CDNDetail(args[1:])
	case "cdn-traffic":
		err = a.CDNTraffic(args[1:])
	case "cdn-usage":
		err = a.CDNUsage(args[1:])
	case "cdn-bandwidth":
		err = a.CDNBandwidth(args[1:])
	case "cdn-refresh":
		err = a.CDNRefresh(args[1:])
	case "cdn-push":
		err = a.CDNPush(args[1:])
	case "cdn-auto-warmup":
		err = a.CDNAutoWarmup(args[1:])
	case "sls-list-projects":
		err = a.SLSListProjects(args[1:])
	case "sls-list-logstores":
		err = a.SLSListLogstores(args[1:])
	case "sls-get-index":
		err = a.SLSGetIndex(args[1:])
	case "sls-update-index":
		err = a.SLSUpdateIndex(args[1:])
	case "sls-get-logs":
		err = a.SLSGetLogs(args[1:])
	case "sls-query-ips":
		err = a.SLSQueryIPs(args[1:])
	default:
		Usage()
		return 2
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return 0
}

func Usage() {
	fmt.Fprintf(os.Stderr, `aliyunctl - unified Aliyun skill helper

Commands:
  init-account     Create isolated account skeleton
  fix-permissions  Set secrets dir/file permissions
  hash-secrets     Write SHA-256 fingerprints to cache.json
  refresh          Discover resources via aliyun resourcecenter SearchResources
  env-check        Validate required env keys
  bill-summary     Query billing overview summary
  account-balance  Query account available balance
  ecs-list         List ECS instances
  ecs-detail       Show ECS instance detail
  ecs-start        Start ECS instance
  ecs-stop         Stop ECS instance
  ecs-reboot       Reboot ECS instance
  ecs-usage        Show ECS resource usage (CPU, memory, disk)
  prom-query       Run Grafana-proxied Prometheus instant query
  prom-range       Run Grafana-proxied Prometheus range query
  prom-labels      List Prometheus labels
  prom-label-values Show values for one Prometheus label
  prom-series      List Prometheus series
  rds-list         List RDS instances
  rds-usage        Show RDS resource usage (CPU, memory, IOPS)
  rds-detail       Show RDS instance detail
  rds-performance  Show RDS resource usage
  rds-list-backups List RDS backups
  polardb-list     List PolarDB clusters
  polardb-usage    Show PolarDB resource usage (CPU, memory, IOPS)
  alb-list         List ALB instances
  alb-usage        Show ALB resource usage (bandwidth, connections, error rate)
  alb-acl-list     List ALB ACLs
  alb-acl-entries  List entries in an ACL
  alb-listener-acl Show listener ACL configuration
  vpc-list         List VPCs in a region
  vpc-detail       Show VPC detail (vswitches, route tables, route entries)
  vswitch-list     List VSwitches in a region or VPC
  routetable-list  List route tables in a region or VPC
  vswitch-resources List cloud resources in a VSwitch (ECS instances, ALB zone mappings)
  sms-stats        Show SMS sending statistics (24h volume, success rate)
  sg-list          List security groups with bound ECS instances
  sg-rules         Show security group rules
  sg-add-ingress   Add ingress rule to security group
  sg-add-egress    Add egress rule to security group
  sg-join          Add ECS instance to security group
  sg-leave         Remove ECS instance from security group
  domain-list      List domains with expiration dates and SSL certificates
  ssl-list         List SSL certificates
  ssl-expiring     List certificates expiring soon
  ssl-summary      Summarize SSL certificate health
  resource-package-list      List resource packages
  resource-package-expiring  List expiring resource packages
  resource-package-summary   Summarize resource package health
  oss-list         List OSS buckets
  oss-info         Show OSS bucket info
  oss-ls           List OSS objects
  oss-recent       List recent OSS objects
  oss-usage        Show OSS bucket usage (storage, traffic, request count)
  cdn-list         List CDN domains
  cdn-detail       Show CDN domain detail
  cdn-traffic      Show CDN traffic data
  cdn-usage        Show CDN usage statistics (traffic, src traffic, hit rate)
  cdn-bandwidth    Show CDN bandwidth data
  cdn-refresh      Refresh CDN paths
  cdn-push         Push CDN URLs
  cdn-auto-warmup  Auto warmup CDN from recent OSS uploads
  sls-list-projects    List SLS projects
  sls-list-logstores   List SLS logstores
  sls-get-index        Get SLS index config
  sls-update-index     Update SLS index with content field
  sls-get-logs         Get raw SLS logs
  sls-query-ips        Query IPs from SLS logs
`)
}

func (a App) InitAccount(args []string) error {
	fs := flag.NewFlagSet("init-account", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*accountName) == "" {
		return errors.New("--account is required")
	}
	for _, d := range []string{a.Paths.SecretsDir(*accountName), a.Paths.ListDir(*accountName), a.Paths.ReportsDir(*accountName)} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	p := a.Paths.EnvPath(*accountName)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		content := fmt.Sprintf(`# Fill real values for this Aliyun account
ALIYUN_ACCOUNT_NAME=%s
ALIBABA_CLOUD_ACCESS_KEY_ID=
ALIBABA_CLOUD_ACCESS_KEY_SECRET=
ALIYUN_REGION_IDS=cn-shanghai
GRAFANA_URL=
GRAFANA_ADMIN_USER=
GRAFANA_ADMIN_PASSWORD=
FEISHU_BOT_WEBHOOK=
`, *accountName)
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			return err
		}
	}
	_ = os.Chmod(a.Paths.SecretsDir(*accountName), 0o700)
	_ = os.Chmod(p, 0o600)
	fmt.Println(a.Paths.AccountDir(*accountName))
	return nil
}

func (a App) FixPermissions(args []string) error {
	fs := flag.NewFlagSet("fix-permissions", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*accountName) == "" {
		return errors.New("--account is required")
	}
	sec := a.Paths.SecretsDir(*accountName)
	if err := os.Chmod(sec, 0o700); err != nil {
		return err
	}
	for _, f := range []string{"runtime.env", "cache.json"} {
		p := filepath.Join(sec, f)
		if _, err := os.Stat(p); err == nil {
			if err := os.Chmod(p, 0o600); err != nil {
				return err
			}
			fmt.Printf("600 %s\n", p)
		}
	}
	fmt.Printf("700 %s\n", sec)
	return nil
}

func (a App) HashSecrets(args []string) error {
	fs := flag.NewFlagSet("hash-secrets", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*accountName) == "" {
		return errors.New("--account is required")
	}
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil {
		return err
	}
	cache := Cache{
		AccountName: env["ALIYUN_ACCOUNT_NAME"],
		RegionIDs:   envfile.SplitCSV(env["ALIYUN_REGION_IDS"]),
		Grafana:     map[string]string{"url": env["GRAFANA_URL"]},
		Feishu:      map[string]bool{"webhookConfigured": strings.TrimSpace(env["FEISHU_BOT_WEBHOOK"]) != ""},
		Fingerprints: map[string]string{
			"accessKeyIdSha256":         sha(env["ALIBABA_CLOUD_ACCESS_KEY_ID"]),
			"accessKeySecretSha256":     sha(env["ALIBABA_CLOUD_ACCESS_KEY_SECRET"]),
			"grafanaUrlSha256":          sha(env["GRAFANA_URL"]),
			"grafanaAdminUserSha256":    sha(env["GRAFANA_ADMIN_USER"]),
			"grafanaAdminPasswordSha256": sha(env["GRAFANA_ADMIN_PASSWORD"]),
			"feishuWebhookSha256":       sha(env["FEISHU_BOT_WEBHOOK"]),
		},
		Notes: []string{"SHA-256 fingerprints are for change detection only, not encryption."},
	}
	b, _ := json.MarshalIndent(cache, "", "  ")
	if err := os.WriteFile(a.Paths.CachePath(*accountName), b, 0o600); err != nil {
		return err
	}
	fmt.Println(a.Paths.CachePath(*accountName))
	return nil
}

func (a App) EnvCheck(args []string) error {
	fs := flag.NewFlagSet("env-check", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*accountName) == "" {
		return errors.New("--account is required")
	}
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil {
		return err
	}
	required := []string{"ALIYUN_ACCOUNT_NAME", "ALIBABA_CLOUD_ACCESS_KEY_ID", "ALIBABA_CLOUD_ACCESS_KEY_SECRET", "ALIYUN_REGION_IDS"}
	missing := []string{}
	for _, k := range required {
		if strings.TrimSpace(env[k]) == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required keys:\n- %s", strings.Join(missing, "\n- "))
	}
	fmt.Println("ok")
	return nil
}

func (a App) Refresh(args []string) error {
	fs := flag.NewFlagSet("refresh", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	maxPages := fs.Int("max-pages", 20, "max pagination count")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*accountName) == "" {
		return errors.New("--account is required")
	}
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil {
		return err
	}
	if strings.TrimSpace(env["ALIBABA_CLOUD_ACCESS_KEY_ID"]) == "" || strings.TrimSpace(env["ALIBABA_CLOUD_ACCESS_KEY_SECRET"]) == "" {
		return errors.New("missing AK/SK in runtime.env")
	}
	summary, err := resourcecenter.Search(env, *maxPages)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(a.Paths.ListDir(*accountName), 0o755); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(summary, "", "  ")
	out := filepath.Join(a.Paths.ListDir(*accountName), "resource-summary.json")
	if err := os.WriteFile(out, b, 0o644); err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}

func sha(v string) string {
	if strings.TrimSpace(v) == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])
}


func (a App) BillSummary(args []string) error {
	fs := flag.NewFlagSet("bill-summary", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	cycle := fs.String("cycle", "", "billing cycle YYYY-MM")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	if strings.TrimSpace(*cycle) == "" { return errors.New("--cycle is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	report, err := billing.QueryBill(*accountName, *cycle, env)
	if err != nil { return err }
	fmt.Println(billing.Format(report, *format))
	return nil
}

func (a App) AccountBalance(args []string) error {
	fs := flag.NewFlagSet("account-balance", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	balance, err := billing.QueryAccountBalance(*accountName, env)
	if err != nil { return err }
	fmt.Println(billing.FormatBalance(balance, *format))
	return nil
}

func (a App) ECSList(args []string) error {
	fs := flag.NewFlagSet("ecs-list", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	region := fs.String("region", "", "region (optional, defaults to account's ALIYUN_REGION_IDS)")
	status := fs.String("status", "", "status")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*accountName) == "" {
		return errors.New("--account is required")
	}
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil {
		return err
	}

	// Get region list: use command line argument if specified, otherwise use account's regions
	regionIDs := envfile.SplitCSV(env["ALIYUN_REGION_IDS"])
	if len(regionIDs) == 0 {
		regionIDs = []string{"cn-shanghai"} // fallback default
	}
	if strings.TrimSpace(*region) != "" {
		regionIDs = []string{*region} // use command line region if specified
	}

	// Query ECS instances in each region
	allItems := []ecs.Instance{}
	for _, r := range regionIDs {
		items, err := ecs.List(r, env, *status)
		if err != nil {
			return err
		}
		allItems = append(allItems, items...)
	}

	fmt.Println(ecs.FormatList(allItems, *format))
	return nil
}

func (a App) PromQuery(args []string) error {
	fs := flag.NewFlagSet("prom-query", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	query := fs.String("query", "", "promql")
	datasource := fs.Int("datasource", 1, "grafana datasource id")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	if strings.TrimSpace(*query) == "" { return errors.New("--query is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	baseURL := env["GRAFANA_URL"]
	user := env["GRAFANA_ADMIN_USER"]
	pass := env["GRAFANA_ADMIN_PASSWORD"]
	if strings.TrimSpace(baseURL) == "" || strings.TrimSpace(user) == "" || strings.TrimSpace(pass) == "" {
		return errors.New("missing Grafana config in runtime.env")
	}
	data, err := prom.Query(baseURL, user, pass, *query, *datasource, 30)
	if err != nil { return err }
	fmt.Println(prom.FormatResult(data, *format, *query))
	return nil
}


func (a App) ECSDetail(args []string) error {
	fs := flag.NewFlagSet("ecs-detail", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	region := fs.String("region", "cn-shanghai", "region")
	instanceID := fs.String("instance-id", "", "instance id")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" || strings.TrimSpace(*instanceID) == "" { return errors.New("--account and --instance-id are required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	item, err := ecs.Detail(*region, env, *instanceID)
	if err != nil { return err }
	fmt.Println(ecs.FormatDetail(item, *format))
	return nil
}

func (a App) ECSStart(args []string) error {
	fs := flag.NewFlagSet("ecs-start", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	region := fs.String("region", "cn-shanghai", "region")
	instanceID := fs.String("instance-id", "", "instance id")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" || strings.TrimSpace(*instanceID) == "" { return errors.New("--account and --instance-id are required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	out, err := ecs.Start(*region, env, *instanceID)
	if err != nil { return err }
	fmt.Println(out)
	return nil
}

func (a App) ECSStop(args []string) error {
	fs := flag.NewFlagSet("ecs-stop", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	region := fs.String("region", "cn-shanghai", "region")
	instanceID := fs.String("instance-id", "", "instance id")
	force := fs.Bool("force", false, "force stop")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" || strings.TrimSpace(*instanceID) == "" { return errors.New("--account and --instance-id are required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	out, err := ecs.Stop(*region, env, *instanceID, *force)
	if err != nil { return err }
	fmt.Println(out)
	return nil
}

func (a App) ECSReboot(args []string) error {
	fs := flag.NewFlagSet("ecs-reboot", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	region := fs.String("region", "cn-shanghai", "region")
	instanceID := fs.String("instance-id", "", "instance id")
	force := fs.Bool("force", false, "force reboot")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" || strings.TrimSpace(*instanceID) == "" { return errors.New("--account and --instance-id are required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	out, err := ecs.Reboot(*region, env, *instanceID, *force)
	if err != nil { return err }
	fmt.Println(out)
	return nil
}

func (a App) ECSUsage(args []string) error {
	fs := flag.NewFlagSet("ecs-usage", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	region := fs.String("region", "", "region (optional, filter by region)")
	format := fs.String("format", "summary", "output format: summary|json")
	datasource := fs.Int("datasource", 1, "grafana datasource id")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	baseURL, user, pass, err := promConfig(env)
	if err != nil { return err }
	items, err := ecs.Usage(baseURL, user, pass, *datasource, 30)
	if err != nil { return err }
	
	// Filter by region if specified
	if strings.TrimSpace(*region) != "" {
		// Get instance IDs for the specified region
		regionInstances, err := ecs.List(*region, env, "")
		if err != nil { return err }
		regionIDs := make(map[string]bool)
		for _, inst := range regionInstances {
			regionIDs[inst.ID] = true
		}
		// Filter usage data
		filtered := []ecs.UsageData{}
		for _, item := range items {
			if regionIDs[item.InstanceID] {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	
	fmt.Println(ecs.FormatUsage(items, *format))
	return nil
}

func promConfig(env map[string]string) (string, string, string, error) {
	baseURL := env["GRAFANA_URL"]
	user := env["GRAFANA_ADMIN_USER"]
	pass := env["GRAFANA_ADMIN_PASSWORD"]
	if strings.TrimSpace(baseURL) == "" || strings.TrimSpace(user) == "" || strings.TrimSpace(pass) == "" {
		return "", "", "", errors.New("missing Grafana config in runtime.env")
	}
	return baseURL, user, pass, nil
}

func (a App) PromRange(args []string) error {
	fs := flag.NewFlagSet("prom-range", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	query := fs.String("query", "", "promql")
	start := fs.String("start", "", "start")
	end := fs.String("end", "", "end")
	step := fs.String("step", "1m", "step")
	datasource := fs.Int("datasource", 1, "grafana datasource id")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" || strings.TrimSpace(*query) == "" || strings.TrimSpace(*start) == "" || strings.TrimSpace(*end) == "" { return errors.New("missing required args: --account, --query, --start, --end") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	baseURL, user, pass, err := promConfig(env)
	if err != nil { return err }
	data, err := prom.QueryRange(baseURL, user, pass, *query, *start, *end, *step, *datasource, 30)
	if err != nil { return err }
	fmt.Println(prom.FormatResult(data, *format, *query))
	return nil
}

func (a App) PromLabels(args []string) error {
	fs := flag.NewFlagSet("prom-labels", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	datasource := fs.Int("datasource", 1, "grafana datasource id")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	baseURL, user, pass, err := promConfig(env)
	if err != nil { return err }
	data, err := prom.Labels(baseURL, user, pass, *datasource, 30)
	if err != nil { return err }
	fmt.Println(prom.FormatLabels(data, *format))
	return nil
}

func (a App) PromLabelValues(args []string) error {
	fs := flag.NewFlagSet("prom-label-values", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	label := fs.String("label", "", "label")
	datasource := fs.Int("datasource", 1, "grafana datasource id")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" || strings.TrimSpace(*label) == "" { return errors.New("--account and --label are required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	baseURL, user, pass, err := promConfig(env)
	if err != nil { return err }
	data, err := prom.LabelValues(baseURL, user, pass, *label, *datasource, 30)
	if err != nil { return err }
	fmt.Println(prom.FormatLabels(data, *format))
	return nil
}

func (a App) PromSeries(args []string) error {
	fs := flag.NewFlagSet("prom-series", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	match := fs.String("match", "", "match[] selector")
	datasource := fs.Int("datasource", 1, "grafana datasource id")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	baseURL, user, pass, err := promConfig(env)
	if err != nil { return err }
	data, err := prom.Series(baseURL, user, pass, *match, *datasource, 30)
	if err != nil { return err }
	fmt.Println(prom.FormatSeries(data, *format))
	return nil
}


func (a App) RDSList(args []string) error {
	fs := flag.NewFlagSet("rds-list", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	region := fs.String("region", "", "region (optional, defaults to account's ALIYUN_REGION_IDS)")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*accountName) == "" {
		return errors.New("--account is required")
	}
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil {
		return err
	}

	// Get region list: use command line argument if specified, otherwise use account's regions
	regionIDs := envfile.SplitCSV(env["ALIYUN_REGION_IDS"])
	if len(regionIDs) == 0 {
		regionIDs = []string{"cn-shanghai"} // fallback default
	}
	if strings.TrimSpace(*region) != "" {
		regionIDs = []string{*region} // use command line region if specified
	}

	// Query RDS instances in each region
	allItems := []rds.Instance{}
	for _, r := range regionIDs {
		items, err := rds.List(r, env)
		if err != nil {
			return err
		}
		allItems = append(allItems, items...)
	}

	fmt.Println(rds.FormatInstances(allItems, *format))
	return nil
}

func (a App) RDSUsage(args []string) error {
	fs := flag.NewFlagSet("rds-usage", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	format := fs.String("format", "summary", "output format: summary|json")
	region := fs.String("region", "", "region (optional, filter by region)")
	datasource := fs.Int("datasource", 1, "grafana datasource id")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	baseURL, user, pass, err := promConfig(env)
	if err != nil { return err }
	items, err := rds.Usage(baseURL, user, pass, *datasource, 30)
	if err != nil { return err }
	
	// Filter by region if specified
	if strings.TrimSpace(*region) != "" {
		regionInstances, err := rds.List(*region, env)
		if err != nil { return err }
		regionIDs := make(map[string]bool)
		for _, inst := range regionInstances {
			regionIDs[inst.ID] = true
		}
		filtered := []rds.UsageData{}
		for _, item := range items {
			if regionIDs[item.InstanceID] {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	
	fmt.Println(rds.FormatUsage(items, *format))
	return nil
}

func (a App) RDSDetail(args []string) error {
	fs := flag.NewFlagSet("rds-detail", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	instanceID := fs.String("instance-id", "", "instance id")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" || strings.TrimSpace(*instanceID) == "" { return errors.New("--account and --instance-id are required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	data, err := rds.Detail(env, *instanceID)
	if err != nil { return err }
	fmt.Println(rds.FormatAny(data, *format))
	return nil
}

func (a App) RDSPerformance(args []string) error {
	fs := flag.NewFlagSet("rds-performance", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	instanceID := fs.String("instance-id", "", "instance id")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" || strings.TrimSpace(*instanceID) == "" { return errors.New("--account and --instance-id are required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	data, err := rds.Performance(env, *instanceID)
	if err != nil { return err }
	fmt.Println(rds.FormatAny(data, *format))
	return nil
}

func (a App) RDSListBackups(args []string) error {
	fs := flag.NewFlagSet("rds-list-backups", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	instanceID := fs.String("instance-id", "", "instance id")
	start := fs.String("start", "", "start")
	end := fs.String("end", "", "end")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" || strings.TrimSpace(*instanceID) == "" { return errors.New("--account and --instance-id are required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	items, err := rds.ListBackups(env, *instanceID, *start, *end)
	if err != nil { return err }
	fmt.Println(rds.FormatBackups(items, *format))
	return nil
}

func (a App) PolarDBList(args []string) error {
	fs := flag.NewFlagSet("polardb-list", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	region := fs.String("region", "", "region (optional, defaults to account's ALIYUN_REGION_IDS)")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*accountName) == "" {
		return errors.New("--account is required")
	}
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil {
		return err
	}

	// Get region list: use command line argument if specified, otherwise use account's regions
	regionIDs := envfile.SplitCSV(env["ALIYUN_REGION_IDS"])
	if len(regionIDs) == 0 {
		regionIDs = []string{"cn-shanghai"} // fallback default
	}
	if strings.TrimSpace(*region) != "" {
		regionIDs = []string{*region} // use command line region if specified
	}

	// Query PolarDB instances in each region
	allItems := []polardb.Instance{}
	for _, r := range regionIDs {
		items, err := polardb.List(r, env)
		if err != nil {
			return err
		}
		allItems = append(allItems, items...)
	}

	fmt.Println(polardb.FormatInstances(allItems, *format))
	return nil
}

func (a App) PolarDBUsage(args []string) error {
	fs := flag.NewFlagSet("polardb-usage", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	region := fs.String("region", "", "region (optional, filter by region)")
	format := fs.String("format", "summary", "output format: summary|json")
	datasource := fs.Int("datasource", 1, "grafana datasource id")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	baseURL, user, pass, err := promConfig(env)
	if err != nil { return err }
	items, err := polardb.Usage(baseURL, user, pass, *datasource, 30)
	if err != nil { return err }
	
	// Filter by region if specified
	if strings.TrimSpace(*region) != "" {
		regionInstances, err := polardb.List(*region, env)
		if err != nil { return err }
		regionIDs := make(map[string]bool)
		for _, inst := range regionInstances {
			regionIDs[inst.ID] = true
		}
		filtered := []polardb.UsageData{}
		for _, item := range items {
			if regionIDs[item.InstanceID] {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	
	fmt.Println(polardb.FormatUsage(items, *format))
	return nil
}

func (a App) ALBList(args []string) error {
	fs := flag.NewFlagSet("alb-list", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	region := fs.String("region", "", "region (optional, defaults to account's ALIYUN_REGION_IDS)")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*accountName) == "" {
		return errors.New("--account is required")
	}
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil {
		return err
	}

	// Get region list: use command line argument if specified, otherwise use account's regions
	regionIDs := envfile.SplitCSV(env["ALIYUN_REGION_IDS"])
	if len(regionIDs) == 0 {
		regionIDs = []string{"cn-shanghai"} // fallback default
	}
	if strings.TrimSpace(*region) != "" {
		regionIDs = []string{*region} // use command line region if specified
	}

	// Query ALB instances in each region
	allItems := []alb.Instance{}
	for _, r := range regionIDs {
		items, err := alb.List(r, env)
		if err != nil {
			return err
		}
		allItems = append(allItems, items...)
	}

	fmt.Println(alb.FormatInstances(allItems, *format))
	return nil
}

func (a App) ALBUsage(args []string) error {
	fs := flag.NewFlagSet("alb-usage", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	region := fs.String("region", "", "region (optional, filter by region)")
	format := fs.String("format", "summary", "output format: summary|json")
	datasource := fs.Int("datasource", 1, "grafana datasource id")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	baseURL, user, pass, err := promConfig(env)
	if err != nil { return err }
	// Retry up to 3 times if data is empty, timeout 30s per query, 2s interval between retries
	var items []alb.UsageData
	for i := 0; i < 3; i++ {
		items, err = alb.Usage(baseURL, user, pass, *datasource, 30)
		if err != nil { return err }
		if len(items) > 0 { break }
		if i < 2 { time.Sleep(2 * time.Second) }
	}
	
	// Filter by region if specified
	if strings.TrimSpace(*region) != "" {
		regionInstances, err := alb.List(*region, env)
		if err != nil { return err }
		regionIDs := make(map[string]bool)
		for _, inst := range regionInstances {
			regionIDs[inst.ID] = true
		}
		filtered := []alb.UsageData{}
		for _, item := range items {
			if regionIDs[item.InstanceID] {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	
	fmt.Println(alb.FormatUsage(items, *format))
	return nil
}

func (a App) ALBAclList(args []string) error {
	fs := flag.NewFlagSet("alb-acl-list", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	region := fs.String("region", "", "region (required)")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	if strings.TrimSpace(*region) == "" { return errors.New("--region is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	items, err := alb.ListAcls(*region, env)
	if err != nil { return err }
	fmt.Println(alb.FormatAcls(items, *format))
	return nil
}

func (a App) ALBAclEntries(args []string) error {
	fs := flag.NewFlagSet("alb-acl-entries", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	region := fs.String("region", "", "region (required)")
	aclID := fs.String("acl-id", "", "ACL ID (required)")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	if strings.TrimSpace(*region) == "" { return errors.New("--region is required") }
	if strings.TrimSpace(*aclID) == "" { return errors.New("--acl-id is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	items, err := alb.ListAclEntries(*region, *aclID, env)
	if err != nil { return err }
	fmt.Println(alb.FormatAclEntries(*aclID, items, *format))
	return nil
}

func (a App) ALBListenerAcl(args []string) error {
	fs := flag.NewFlagSet("alb-listener-acl", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	region := fs.String("region", "", "region (required)")
	listenerID := fs.String("listener-id", "", "listener ID (optional, show all if not specified)")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	if strings.TrimSpace(*region) == "" { return errors.New("--region is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	
	if strings.TrimSpace(*listenerID) != "" {
		// Get single listener ACL config
		item, err := alb.GetListenerAcl(*region, *listenerID, env)
		if err != nil { return err }
		items := []alb.ListenerAclConfig{item}
		fmt.Println(alb.FormatListenersAcl(items, *format))
	} else {
		// Get all listeners ACL config
		items, err := alb.ListListenersWithAcl(*region, env)
		if err != nil { return err }
		fmt.Println(alb.FormatListenersAcl(items, *format))
	}
	return nil
}

func (a App) VPCList(args []string) error {
	fs := flag.NewFlagSet("vpc-list", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	region := fs.String("region", "", "region (required)")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	if strings.TrimSpace(*region) == "" { return errors.New("--region is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	items, err := vpc.ListVPCs(*region, env)
	if err != nil { return err }
	fmt.Println(vpc.FormatVPCs(items, *format))
	return nil
}

func (a App) VPCDetail(args []string) error {
	fs := flag.NewFlagSet("vpc-detail", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	region := fs.String("region", "", "region (required)")
	vpcID := fs.String("vpc-id", "", "VPC ID (required)")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	if strings.TrimSpace(*region) == "" { return errors.New("--region is required") }
	if strings.TrimSpace(*vpcID) == "" { return errors.New("--vpc-id is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	detail, err := vpc.GetVPCDetail(*region, *vpcID, env)
	if err != nil { return err }
	fmt.Println(vpc.FormatVPCDetail(detail, *format))
	return nil
}

func (a App) VSwitchList(args []string) error {
	fs := flag.NewFlagSet("vswitch-list", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	region := fs.String("region", "", "region (required)")
	vpcID := fs.String("vpc-id", "", "VPC ID (optional, filter by VPC)")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	if strings.TrimSpace(*region) == "" { return errors.New("--region is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	items, err := vpc.ListVSwitches(*region, *vpcID, env)
	if err != nil { return err }
	fmt.Println(vpc.FormatVSwitches(items, *format))
	return nil
}

func (a App) RouteTableList(args []string) error {
	fs := flag.NewFlagSet("routetable-list", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	region := fs.String("region", "", "region (required)")
	vpcID := fs.String("vpc-id", "", "VPC ID (optional, filter by VPC)")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	if strings.TrimSpace(*region) == "" { return errors.New("--region is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	items, err := vpc.ListRouteTables(*region, *vpcID, env)
	if err != nil { return err }
	fmt.Println(vpc.FormatRouteTables(items, *format))
	return nil
}

func (a App) VSwitchResources(args []string) error {
	fs := flag.NewFlagSet("vswitch-resources", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	region := fs.String("region", "", "region (required)")
	vswitchID := fs.String("vswitch-id", "", "VSwitch ID (required)")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	if strings.TrimSpace(*region) == "" { return errors.New("--region is required") }
	if strings.TrimSpace(*vswitchID) == "" { return errors.New("--vswitch-id is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	
	// Query ECS instances in the VSwitch
	ecsInstances, err := ecs.ListByVSwitch(*region, *vswitchID, env)
	if err != nil { return err }
	
	// Query ALB zone mappings in the VSwitch
	albMappings, err := alb.ListAllZoneMappings(*region, env)
	if err != nil { return err }
	albInVSwitch := []alb.ZoneMapping{}
	for _, m := range albMappings {
		if m.VSwitchId == *vswitchID {
			albInVSwitch = append(albInVSwitch, m)
		}
	}
	
	// Format output
	if *format == "json" {
		data := map[string]any{
			"vswitchId": *vswitchID,
			"ecs": ecsInstances,
			"alb": albInVSwitch,
		}
		b, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(b))
	} else {
		fmt.Printf("交换机: %s\n\n", *vswitchID)
		fmt.Printf("=== ECS 实例 (%d) ===\n", len(ecsInstances))
		if len(ecsInstances) == 0 {
			fmt.Println("无 ECS 实例")
		} else {
			for _, it := range ecsInstances {
				fmt.Printf("- %s (%s)\n", it.Name, it.ID)
				fmt.Printf("  可用区: %s | 状态: %s\n", it.Zone, it.Status)
			}
		}
		fmt.Printf("\n=== ALB Zone 映射 (%d) ===\n", len(albInVSwitch))
		if len(albInVSwitch) == 0 {
			fmt.Println("无 ALB Zone 映射")
		} else {
			for _, m := range albInVSwitch {
				fmt.Printf("- %s (%s)\n", m.LoadBalancerName, m.LoadBalancerID)
				fmt.Printf("  内网IP: %s | EIP: %s | 可用区: %s\n", m.IntranetAddress, m.EipAddress, m.ZoneId)
			}
		}
	}
	return nil
}

func (a App) SMSStats(args []string) error {
	fs := flag.NewFlagSet("sms-stats", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	stats, err := sms.QuerySendStatistics(env)
	if err != nil { return err }
	fmt.Println(sms.FormatStats(stats, *format))
	return nil
}

func (a App) SGList(args []string) error {
	fs := flag.NewFlagSet("sg-list", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	region := fs.String("region", "", "region (required)")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	if strings.TrimSpace(*region) == "" { return errors.New("--region is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	items, err := securitygroup.ListWithInstances(*region, env)
	if err != nil { return err }
	fmt.Println(securitygroup.FormatList(items, *format))
	return nil
}

func (a App) SGRules(args []string) error {
	fs := flag.NewFlagSet("sg-rules", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	region := fs.String("region", "", "region (required)")
	securityGroupID := fs.String("sg-id", "", "security group ID (required)")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	if strings.TrimSpace(*region) == "" { return errors.New("--region is required") }
	if strings.TrimSpace(*securityGroupID) == "" { return errors.New("--sg-id is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	rules, err := securitygroup.GetRules(*region, *securityGroupID, env)
	if err != nil { return err }
	fmt.Println(securitygroup.FormatRules(*securityGroupID, rules, *format))
	return nil
}

func (a App) SGAddIngress(args []string) error {
	fs := flag.NewFlagSet("sg-add-ingress", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	region := fs.String("region", "", "region (required)")
	securityGroupID := fs.String("sg-id", "", "security group ID (required)")
	protocol := fs.String("protocol", "TCP", "IP protocol (TCP/UDP/ICMP/ALL)")
	portRange := fs.String("port", "", "port range (e.g., 80/80, 22/22)")
	sourceCidr := fs.String("source", "0.0.0.0/0", "source CIDR (default: 0.0.0.0/0)")
	policy := fs.String("policy", "Accept", "policy (Accept/Drop)")
	priority := fs.Int("priority", 1, "priority (1-100, lower is higher)")
	description := fs.String("desc", "", "rule description")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	if strings.TrimSpace(*region) == "" { return errors.New("--region is required") }
	if strings.TrimSpace(*securityGroupID) == "" { return errors.New("--sg-id is required") }
	if strings.TrimSpace(*portRange) == "" { return errors.New("--port is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	err = securitygroup.AddIngressRule(*region, *securityGroupID, *protocol, *portRange, *sourceCidr, *policy, *description, *priority, env)
	if err != nil { return err }
	fmt.Printf("已添加入方向规则: %s %s -> %s (%s)\n", *protocol, *portRange, *sourceCidr, *policy)
	return nil
}

func (a App) SGAddEgress(args []string) error {
	fs := flag.NewFlagSet("sg-add-egress", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	region := fs.String("region", "", "region (required)")
	securityGroupID := fs.String("sg-id", "", "security group ID (required)")
	protocol := fs.String("protocol", "TCP", "IP protocol (TCP/UDP/ICMP/ALL)")
	portRange := fs.String("port", "", "port range (e.g., 80/80, -1/-1 for all)")
	destCidr := fs.String("dest", "0.0.0.0/0", "destination CIDR (default: 0.0.0.0/0)")
	policy := fs.String("policy", "Accept", "policy (Accept/Drop)")
	priority := fs.Int("priority", 1, "priority (1-100, lower is higher)")
	description := fs.String("desc", "", "rule description")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	if strings.TrimSpace(*region) == "" { return errors.New("--region is required") }
	if strings.TrimSpace(*securityGroupID) == "" { return errors.New("--sg-id is required") }
	if strings.TrimSpace(*portRange) == "" { return errors.New("--port is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	err = securitygroup.AddEgressRule(*region, *securityGroupID, *protocol, *portRange, *destCidr, *policy, *description, *priority, env)
	if err != nil { return err }
	fmt.Printf("已添加出方向规则: %s %s -> %s (%s)\n", *protocol, *portRange, *destCidr, *policy)
	return nil
}

func (a App) SGJoin(args []string) error {
	fs := flag.NewFlagSet("sg-join", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	region := fs.String("region", "", "region (required)")
	securityGroupID := fs.String("sg-id", "", "security group ID (required)")
	instanceID := fs.String("instance-id", "", "ECS instance ID (required)")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	if strings.TrimSpace(*region) == "" { return errors.New("--region is required") }
	if strings.TrimSpace(*securityGroupID) == "" { return errors.New("--sg-id is required") }
	if strings.TrimSpace(*instanceID) == "" { return errors.New("--instance-id is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	err = securitygroup.JoinSecurityGroup(*region, *securityGroupID, *instanceID, env)
	if err != nil { return err }
	fmt.Printf("实例 %s 已加入安全组 %s\n", *instanceID, *securityGroupID)
	return nil
}

func (a App) SGLeave(args []string) error {
	fs := flag.NewFlagSet("sg-leave", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	region := fs.String("region", "", "region (required)")
	securityGroupID := fs.String("sg-id", "", "security group ID (required)")
	instanceID := fs.String("instance-id", "", "ECS instance ID (required)")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	if strings.TrimSpace(*region) == "" { return errors.New("--region is required") }
	if strings.TrimSpace(*securityGroupID) == "" { return errors.New("--sg-id is required") }
	if strings.TrimSpace(*instanceID) == "" { return errors.New("--instance-id is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	err = securitygroup.LeaveSecurityGroup(*region, *securityGroupID, *instanceID, env)
	if err != nil { return err }
	fmt.Printf("实例 %s 已移出安全组 %s\n", *instanceID, *securityGroupID)
	return nil
}

func (a App) DomainList(args []string) error {
	fs := flag.NewFlagSet("domain-list", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	format := fs.String("format", "summary", "output format: summary|json")
	showSize := fs.Int("show-size", 50, "SSL certificate list size")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	items, err := domain.ListWithCerts(env, *showSize)
	if err != nil { return err }
	fmt.Println(domain.FormatListWithCerts(items, *format))
	return nil
}

func (a App) SSLList(args []string) error {
	fs := flag.NewFlagSet("ssl-list", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	showSize := fs.Int("show-size", 50, "show size")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	items, err := sslcert.List(env, *showSize)
	if err != nil { return err }
	fmt.Println(sslcert.FormatList(items, *format))
	return nil
}

func (a App) SSLExpiring(args []string) error {
	fs := flag.NewFlagSet("ssl-expiring", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	days := fs.Int("days", 30, "days")
	showSize := fs.Int("show-size", 100, "show size")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	items, err := sslcert.List(env, *showSize)
	if err != nil { return err }
	fmt.Println(sslcert.FormatList(sslcert.Expiring(items, *days), *format))
	return nil
}

func (a App) SSLSummary(args []string) error {
	fs := flag.NewFlagSet("ssl-summary", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	showSize := fs.Int("show-size", 100, "show size")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	items, err := sslcert.List(env, *showSize)
	if err != nil { return err }
	fmt.Println(sslcert.FormatSummary(sslcert.Summary(items), *format))
	return nil
}


func (a App) ResourcePackageList(args []string) error {
	fs := flag.NewFlagSet("resource-package-list", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	items, err := resourcepkg.List(env)
	if err != nil { return err }
	fmt.Println(resourcepkg.FormatList(items, *format))
	return nil
}

func (a App) ResourcePackageExpiring(args []string) error {
	fs := flag.NewFlagSet("resource-package-expiring", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	days := fs.Int("days", 30, "days")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	items, err := resourcepkg.List(env)
	if err != nil { return err }
	fmt.Println(resourcepkg.FormatList(resourcepkg.Expiring(items, *days), *format))
	return nil
}

func (a App) ResourcePackageSummary(args []string) error {
	fs := flag.NewFlagSet("resource-package-summary", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	items, err := resourcepkg.List(env)
	if err != nil { return err }
	fmt.Println(resourcepkg.FormatSummary(resourcepkg.Summary(items), *format))
	return nil
}

func (a App) OSSList(args []string) error {
	fs := flag.NewFlagSet("oss-list", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	items, err := ossmod.ListBuckets(env)
	if err != nil { return err }
	fmt.Println(ossmod.FormatBuckets(items, *format))
	return nil
}

func (a App) OSSInfo(args []string) error {
	fs := flag.NewFlagSet("oss-info", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	bucket := fs.String("bucket", "", "bucket")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" || strings.TrimSpace(*bucket) == "" { return errors.New("--account and --bucket are required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	out, err := ossmod.BucketInfo(env, *bucket)
	if err != nil { return err }
	fmt.Println(out)
	return nil
}

func (a App) OSSLS(args []string) error {
	fs := flag.NewFlagSet("oss-ls", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	bucket := fs.String("bucket", "", "bucket")
	prefix := fs.String("prefix", "", "prefix")
	maxKeys := fs.Int("max-keys", 100, "max keys")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" || strings.TrimSpace(*bucket) == "" { return errors.New("--account and --bucket are required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	items, err := ossmod.ListObjects(env, *bucket, *prefix, *maxKeys)
	if err != nil { return err }
	fmt.Println(ossmod.FormatObjects(items, *format, *bucket))
	return nil
}

func (a App) OSSRecent(args []string) error {
	fs := flag.NewFlagSet("oss-recent", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	bucket := fs.String("bucket", "", "bucket")
	prefix := fs.String("prefix", "", "prefix")
	maxKeys := fs.Int("max-keys", 100, "max keys")
	hours := fs.Int("hours", 1, "hours")
	format := fs.String("format", "summary", "output format: summary|json")
	fileTypes := fs.String("file-types", "", "comma-separated file types")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" || strings.TrimSpace(*bucket) == "" { return errors.New("--account and --bucket are required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	fts := []string{}
	if strings.TrimSpace(*fileTypes) != "" { for _, x := range strings.Split(*fileTypes, ",") { x = strings.TrimSpace(x); if x != "" { fts = append(fts, x) } } }
	items, err := ossmod.RecentObjects(env, *bucket, *prefix, *maxKeys, *hours, fts)
	if err != nil { return err }
	fmt.Println(ossmod.FormatObjects(items, *format, *bucket))
	return nil
}

func (a App) OSSUsage(args []string) error {
	fs := flag.NewFlagSet("oss-usage", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	format := fs.String("format", "summary", "output format: summary|json")
	datasource := fs.Int("datasource", 1, "grafana datasource id")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	baseURL, user, pass, err := promConfig(env)
	if err != nil { return err }
	items, err := ossmod.Usage(env, baseURL, user, pass, *datasource, 30)
	if err != nil { return err }
	fmt.Println(ossmod.FormatUsage(items, *format))
	return nil
}

func (a App) CDNList(args []string) error {
	fs := flag.NewFlagSet("cdn-list", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	items, err := cdnmod.List(env)
	if err != nil { return err }
	fmt.Println(cdnmod.FormatDomains(items, *format))
	return nil
}

func (a App) CDNDetail(args []string) error {
	fs := flag.NewFlagSet("cdn-detail", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	domain := fs.String("domain", "", "domain")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" || strings.TrimSpace(*domain) == "" { return errors.New("--account and --domain are required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	data, err := cdnmod.Detail(env, *domain)
	if err != nil { return err }
	fmt.Println(cdnmod.FormatAny(data, *format))
	return nil
}

func (a App) CDNTraffic(args []string) error {
	fs := flag.NewFlagSet("cdn-traffic", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	domain := fs.String("domain", "", "domain")
	start := fs.String("start", "", "start")
	end := fs.String("end", "", "end")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" || strings.TrimSpace(*domain) == "" { return errors.New("--account and --domain are required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	data, err := cdnmod.Traffic(env, *domain, *start, *end)
	if err != nil { return err }
	fmt.Println(cdnmod.FormatAny(data, *format))
	return nil
}

func (a App) CDNUsage(args []string) error {
	fs := flag.NewFlagSet("cdn-usage", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	start := fs.String("start", "", "start time (yyyy-mm-dd)")
	end := fs.String("end", "", "end time (yyyy-mm-dd)")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	
	// Default to last 24 hours if not specified
	startTime := *start
	endTime := *end
	if startTime == "" || endTime == "" {
		now := time.Now()
		startTime = now.Add(-24 * time.Hour).Format("2006-01-02T15:04:05Z")
		endTime = now.Format("2006-01-02T15:04:05Z")
	}
	
	items, err := cdnmod.Usage(env, startTime, endTime)
	if err != nil { return err }
	fmt.Println(cdnmod.FormatUsage(items, *format))
	return nil
}

func (a App) CDNBandwidth(args []string) error {
	fs := flag.NewFlagSet("cdn-bandwidth", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	domain := fs.String("domain", "", "domain")
	start := fs.String("start", "", "start")
	end := fs.String("end", "", "end")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" || strings.TrimSpace(*domain) == "" { return errors.New("--account and --domain are required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	data, err := cdnmod.Bandwidth(env, *domain, *start, *end)
	if err != nil { return err }
	fmt.Println(cdnmod.FormatAny(data, *format))
	return nil
}

func (a App) CDNRefresh(args []string) error {
	fs := flag.NewFlagSet("cdn-refresh", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	paths := fs.String("paths", "", "comma-separated paths")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" || strings.TrimSpace(*paths) == "" { return errors.New("--account and --paths are required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	arr := []string{}
	for _, x := range strings.Split(*paths, ",") { x = strings.TrimSpace(x); if x != "" { arr = append(arr, x) } }
	out, err := cdnmod.Refresh(env, arr)
	if err != nil { return err }
	fmt.Println(out)
	return nil
}

func (a App) CDNPush(args []string) error {
	fs := flag.NewFlagSet("cdn-push", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	urls := fs.String("urls", "", "comma-separated urls")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" || strings.TrimSpace(*urls) == "" { return errors.New("--account and --urls are required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	arr := []string{}
	for _, x := range strings.Split(*urls, ",") { x = strings.TrimSpace(x); if x != "" { arr = append(arr, x) } }
	out, err := cdnmod.Push(env, arr)
	if err != nil { return err }
	fmt.Println(out)
	return nil
}

func (a App) CDNAutoWarmup(args []string) error {
	fs := flag.NewFlagSet("cdn-auto-warmup", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	bucket := fs.String("bucket", "", "OSS bucket name")
	hours := fs.Int("hours", 1, "hours to look back for recent uploads")
	format := fs.String("format", "summary", "output format: summary|json")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	results, err := cdnmod.AutoWarmup(env, *bucket, *hours)
	if err != nil { return err }
	fmt.Println(cdnmod.FormatAutoWarmup(results, *format))
	return nil
}

func (a App) SLSListProjects(args []string) error {
	fs := flag.NewFlagSet("sls-list-projects", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" { return errors.New("--account is required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	out, err := slsmod.ListProjects(env)
	if err != nil { return err }
	fmt.Println(out)
	return nil
}

func (a App) SLSListLogstores(args []string) error {
	fs := flag.NewFlagSet("sls-list-logstores", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	project := fs.String("project", "", "project")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" || strings.TrimSpace(*project) == "" { return errors.New("--account and --project are required") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	out, err := slsmod.ListLogstores(env, *project)
	if err != nil { return err }
	fmt.Println(out)
	return nil
}

func (a App) SLSGetIndex(args []string) error {
	fs := flag.NewFlagSet("sls-get-index", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	project := fs.String("project", "", "project")
	logstore := fs.String("logstore", "", "logstore")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" || strings.TrimSpace(*project) == "" || strings.TrimSpace(*logstore) == "" { return errors.New("missing required args: --account, --project, --logstore") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	out, err := slsmod.GetIndex(env, *project, *logstore)
	if err != nil { return err }
	fmt.Println(out)
	return nil
}

func (a App) SLSUpdateIndex(args []string) error {
	fs := flag.NewFlagSet("sls-update-index", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	project := fs.String("project", "", "project")
	logstore := fs.String("logstore", "", "logstore")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" || strings.TrimSpace(*project) == "" || strings.TrimSpace(*logstore) == "" { return errors.New("missing required args: --account, --project, --logstore") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	out, err := slsmod.UpdateIndexAddContent(env, *project, *logstore)
	if err != nil { return err }
	fmt.Println(out)
	return nil
}

func (a App) SLSGetLogs(args []string) error {
	fs := flag.NewFlagSet("sls-get-logs", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	project := fs.String("project", "", "project")
	logstore := fs.String("logstore", "", "logstore")
	from := fs.String("from", "", "from")
	to := fs.String("to", "", "to")
	query := fs.String("query", "*", "query")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" || strings.TrimSpace(*project) == "" || strings.TrimSpace(*logstore) == "" || strings.TrimSpace(*from) == "" || strings.TrimSpace(*to) == "" { return errors.New("missing required args: --account, --project, --logstore, --from, --to") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	out, err := slsmod.GetLogs(env, *project, *logstore, *from, *to, *query)
	if err != nil { return err }
	fmt.Println(out)
	return nil
}

func (a App) SLSQueryIPs(args []string) error {
	fs := flag.NewFlagSet("sls-query-ips", flag.ContinueOnError)
	accountName := fs.String("account", "", "account name")
	project := fs.String("project", "", "project")
	logstore := fs.String("logstore", "", "logstore")
	from := fs.String("from", "", "from")
	to := fs.String("to", "", "to")
	query := fs.String("query", "[HTTP]", "query")
	if err := fs.Parse(args); err != nil { return err }
	if strings.TrimSpace(*accountName) == "" || strings.TrimSpace(*project) == "" || strings.TrimSpace(*logstore) == "" || strings.TrimSpace(*from) == "" || strings.TrimSpace(*to) == "" { return errors.New("missing required args: --account, --project, --logstore, --from, --to") }
	env, err := envfile.Parse(a.Paths.EnvPath(*accountName))
	if err != nil { return err }
	out, err := slsmod.QueryIPs(env, *project, *logstore, *from, *to, *query)
	if err != nil { return err }
	fmt.Println(out)
	return nil
}
