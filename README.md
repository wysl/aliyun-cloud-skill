# Aliyun Cloud Skill

## 更新记录

- 更新时间：2026-04-11
- 本次变更：同步当前项目目录结构与实现说明；核对 README / `SKILL.md` / `references/commands.md` 的文档一致性；补充 Grafana 连接信息属于账号级配置，只能存放在 `accounts/<account>/secrets/runtime.env`；新增 `prompt_template/` 目录与管理层 / 运维 / 通用版 HTML 报告预制提示词模板

这是一个基于 Go 的阿里云运维/巡检工具仓库，核心目标是：

- 以 `accounts/<account>/` 进行账号隔离
- 通过统一入口 `aliyunctl` 执行所有子命令
- 覆盖账号初始化、资源盘点、账单查询、资源使用率巡检，以及常见云产品运维操作

## 运行入口

主要入口是 Go 程序：

```bash
go run ./scripts/aliyunctl.go <command> ...
```

仓库中也包含已编译的 `aliyunctl` 二进制，但当前实现仍以 `scripts/aliyunctl.go` 为主入口。

## 当前目录结构

```text
aliyun-cloud-skill/
├── SKILL.md                     # Skill 定义、触发说明、核心用法
├── README.md                    # 项目结构与实现说明（本文件）
├── go.mod                       # Go 模块定义
├── scripts/
│   └── aliyunctl.go             # 程序入口：定位仓库根目录并调用 app.Run
├── internal/
│   ├── account/                 # 账号目录路径模型
│   ├── alb/                     # ALB 查询与操作
│   ├── aliyuncli/               # aliyun CLI 调用封装
│   ├── app/                     # 命令路由、参数解析、输出控制
│   ├── billing/                 # 账单与余额查询
│   ├── cdnmod/                  # CDN 查询、刷新、预热
│   ├── domain/                  # 域名查询
│   ├── ecs/                     # ECS 查询与启停重启、监控
│   ├── envfile/                 # runtime.env 解析
│   ├── ossmod/                  # OSS 查询与对象列表
│   ├── polardb/                 # PolarDB 查询与监控
│   ├── prom/                    # 通过 Grafana 代理执行 Prometheus 查询
│   ├── rds/                     # RDS 查询、备份、性能监控
│   ├── resourcecenter/          # 资源中心盘点
│   ├── resourcepkg/             # 资源包查询与到期检查
│   ├── securitygroup/           # 安全组规则与实例绑定管理
│   ├── slsmod/                  # SLS 项目、索引、日志、机器组、采集配置
│   ├── sms/                     # 短信发送统计
│   ├── sslcert/                 # SSL 证书查询与到期检查
│   └── vpc/                     # VPC / VSwitch / 路由表查询
├── references/
│   ├── commands.md              # 命令清单
│   ├── onboarding-flow.md       # 账号初始化流程
│   ├── release-checklist.md     # 发布前检查项
│   └── audit-notes.md           # 设计约束与演进说明
├── prompt_template/
│   ├── management_html_report_prompt.txt   # 管理层简版 HTML 报告提示词模板
│   ├── ops_html_report_prompt.txt          # 运维巡检详细版 HTML 报告提示词模板
│   └── full_html_report_prompt.txt         # 通用完整版 HTML 报告提示词模板
├── accounts/
│   ├── example/                 # 示例账号目录结构
│   └── <account>/               # 实际账号隔离目录
│       ├── secrets/
│       │   ├── runtime.env      # AK/SK、区域、Grafana、Webhook 等配置
│       │   └── cache.json       # 指纹缓存
│       ├── list/                # 资源盘点结果
│       └── reports/             # 报告与导出产物
├── index-config.json            # SLS 索引配置样例
├── index-config-v2.json         # SLS 索引配置样例（中文分词）
├── index-config-simple.json     # 精简版索引配置样例
└── .gitignore                   # 忽略账号敏感数据、报告产物、二进制等
```

## 核心实现

### 1. 程序入口

`scripts/aliyunctl.go` 负责：

- 自动识别仓库根目录
- 初始化 `app.App`
- 将命令行参数统一交给 `internal/app/app.go`

也就是说，所有功能都从一个入口进入，而不是分散在多个脚本里。

### 2. 统一命令路由

`internal/app/app.go` 是整个仓库的核心控制层，主要负责：

- 子命令分发
- 参数解析
- 加载账号环境变量
- 调用各业务模块
- 输出 `summary` 或 `json`
- 错误统一返回

目前已经实现的命令覆盖以下类别：

- 账号初始化：`init-account`、`fix-permissions`、`hash-secrets`、`env-check`
- 资源盘点：`refresh`
- 账单：`bill-summary`、`account-balance`
- ECS：列表、详情、启停重启、使用率
- Prometheus / Grafana：即时查询、区间查询、标签、序列
- RDS：列表、详情、性能、备份、使用率
- PolarDB：列表、使用率
- ALB：列表、使用率、ACL 查询
- VPC：VPC / VSwitch / 路由表 / 交换机资源查询
- SMS：发送统计
- Security Group：规则查询、入/出方向放行、实例加组退组
- Domain / SSL：域名、证书、到期摘要
- Resource Package：列表、到期、摘要
- OSS：bucket、对象列表、近期对象、容量统计、流量与请求量
- CDN：流量、回源流量、命中率、带宽、域名详情、刷新、预热、自动 warmup
- SLS：项目、Logstore、索引、日志、机器组、采集配置

### 3. 账号隔离模型

`internal/account/account.go` 把账号目录结构统一抽象成路径模型，约定：

- `accounts/<account>/secrets`
- `accounts/<account>/list`
- `accounts/<account>/reports`

这样所有命令都通过同一套路径规则读写数据，避免散乱拼路径。

### 4. 环境配置与密钥处理

- `internal/envfile/envfile.go`：解析 `runtime.env` 并支持 CSV 区域列表拆分
- `hash-secrets`：将敏感配置生成 SHA-256 指纹写入 `cache.json`
- `fix-permissions`：统一修正 `secrets/`、`runtime.env`、`cache.json` 的权限

这里的指纹用于变更检测，不是加密存储。

### 5. 阿里云调用方式

仓库不是直接集成阿里云 SDK，而是以 `aliyun` CLI 为底层执行器：

- `internal/aliyuncli/aliyuncli.go` 提供统一的命令执行与 JSON 解析封装
- `internal/resourcecenter/resourcecenter.go` 使用 `aliyun resourcecenter SearchResources` 做资源盘点
- 其他产品模块也沿用相同思路封装各自命令

这让各模块的调用风格保持一致，也降低了接入新产品时的复杂度。

### 6. Prometheus / Grafana 查询

`internal/prom/prom.go` 不是直连 Prometheus，而是通过 Grafana datasource proxy 发请求，已支持：

- 即时查询 `query`
- 区间查询 `query_range`
- 标签列表
- 标签值列表
- Series 查询
- `summary/json` 两种输出格式

适合把账号级云资源巡检与 Grafana 监控查询放在同一个工具里处理。

### 7. SLS 扩展能力

`internal/slsmod/slsmod.go` 除了基础查询，还实现了：

- 项目 / Logstore 创建
- 索引读取与更新
- 原始日志拉取
- 从日志中提取 IP
- 机器组查询与创建
- Logtail 采集配置创建
- 将配置应用到机器组

对应的 `index-config*.json` 可以视作索引配置样例文件，方便人工调整或复用。

## 数据与产物约定

### `accounts/<account>/secrets/`

存放账号敏感配置：

- `runtime.env`
- `cache.json`

### `accounts/<account>/list/`

存放资源盘点类结果，例如 `refresh` 生成的资源汇总数据。

### `accounts/<account>/reports/`

存放报告、HTML 导出、脚本输出等产物，不应作为核心源码逻辑的一部分。

## 配套文档分工

- `SKILL.md`：给 Claude/skill 使用的入口说明与典型命令
- `references/commands.md`：完整命令索引
- `references/onboarding-flow.md`：初始化新账号时的信息收集顺序
- `references/release-checklist.md`：发布前应跑的最小验证集
- `references/audit-notes.md`：当前仓库设计约束，例如“单入口、少脚本、少重复逻辑”
- `prompt_template/*.txt`：生成管理层 / 运维 / 通用版 HTML 资源报告时可复用的预制提示词模板

## 当前实现特征总结

这个仓库当前已经不是“零散脚本集合”，而是一个：

- **单入口**：所有命令统一走 `aliyunctl`
- **账号隔离**：所有配置与产物按账号归档
- **CLI 驱动**：底层以 `aliyun` CLI 为主，而非 SDK
- **模块化实现**：每个云产品一个独立 package
- **输出统一**：面向人工查看时优先 `summary`，需要对接工具时可用 `json`
- **文档拆分**：Skill 入口、命令清单、onboarding、发布检查分别维护

## 维护 README 时应同步关注

当新增命令或模块时，通常需要同步更新：

- `internal/app/app.go` 中的命令路由与 `Usage()`
- `SKILL.md` 中的 description / examples / workflows
- `references/commands.md` 中的命令清单
- 本 README 中的目录结构或实现说明（如果架构层面发生变化）
