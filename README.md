# Aliyun Cloud Monitor Skill - 项目结构说明

本文档说明 skill 的文件结构及其职责，供新增功能时参考。

## 文件职责

| 文件 | 职责 | 内容 |
|------|------|------|
| `SKILL.md` | 触发、概述、导航 | Skill 元数据、功能概述、核心示例、入口导航 |
| `references/commands.md` | 完整命令清单 | 所有可用命令的分类列表和简要说明 |
| `references/release-checklist.md` | 发布前验证 | 发布前需要验证的命令和检查项 |
| `references/onboarding-flow.md` | Onboarding 细节 | 初始化账号的详细步骤和参数收集顺序 |
| `references/audit-notes.md` | 设计说明 | 为什么这样设计、架构决策记录 |

## 导航指引

| 问题 | 答案位置 |
|------|----------|
| 是什么 / 何时用 | `SKILL.md` |
| 怎么初始化账号 | `references/onboarding-flow.md` |
| 有哪些命令 | `references/commands.md` |
| 怎么做发布前检查 | `references/release-checklist.md` |
| 为什么这样设计 | `references/audit-notes.md` |

## 新增功能时的修改清单

当新增功能（如新命令、新模块）时，需要按以下清单更新对应文件：

### 1. 代码层面

| 步骤 | 文件 | 操作 |
|------|------|------|
| 模块实现 | `internal/<module>/<module>.go` | 新建或修改模块文件，添加数据结构和函数 |
| 命令路由 | `internal/app/app.go` | 在 switch 语句中添加新命令 case |
| 命令说明 | `internal/app/app.go` | 在 Usage() 函数中添加命令说明文本 |
| 函数实现 | `internal/app/app.go` | 实现新命令的处理函数 |

### 2. 文档层面

| 步骤 | 文件 | 操作 |
|------|------|------|
| 更新 description | `SKILL.md` | 在 YAML front matter 的 description 中添加新功能关键词 |
| 更新 Core examples | `SKILL.md` | 添加新命令的使用示例 |
| 更新 Supported workflows | `SKILL.md` | 添加新功能的工作流程说明 |
| 更新命令清单 | `references/commands.md` | 在对应分类下添加新命令说明 |
| 更新发布验证 | `references/release-checklist.md` | 如有必要，添加新命令的验证步骤 |

### 3. 示例：新增 `account-balance` 命令

```markdown
#### 代码层面修改：
1. `internal/billing/billing.go`：
   - 添加 AccountBalance 结构体
   - 添加 QueryAccountBalance 函数
   - 添加 FormatBalance 函数

2. `internal/app/app.go`：
   - switch 语句：添加 `case "account-balance":`
   - Usage()：添加 `account-balance  Query account available balance`
   - 添加 AccountBalance 函数实现

#### 文档层面修改：
1. `SKILL.md`：
   - description：添加 "account balance queries"
   - Core examples：添加示例命令
   - Supported workflows：添加 "Query account balance and billing summaries"

2. `references/commands.md`：
   - Billing 分类下添加 `account-balance — query account available balance`
```

## 项目目录结构

```
aliyun-cloud-monitor2/
├── SKILL.md                    # Skill 入口（触发、概述、导航）
├── README.md                   # 项目结构说明（本文件）
├── scripts/
│   └── aliyunctl.go           # 程序入口（main 函数）
├── internal/
│   ├── app/
│   │   └── app.go             # 命令路由和处理
│   ├── account/               # 账号管理
│   ├── billing/               # 账单查询
│   ├── ecs/                   # ECS 模块
│   ├── rds/                   # RDS 模块
│   ├── polardb/               # PolarDB 模块
│   ├── prom/                  # Prometheus 查询
│   ├── sslcert/               # SSL 证书
│   ├── resourcepkg/           # 资源包
│   ├── ossmod/                # OSS 模块
│   ├── cdnmod/                # CDN 模块
│   ├── slsmod/                # SLS 模块
│   ├── aliyuncli/             # 阿里云 CLI 封装
│   ├── envfile/               # 环境变量解析
│   └── resourcecenter/        # 资源中心
├── references/
│   ├── commands.md            # 完整命令清单
│   ├── release-checklist.md   # 发布前验证
│   ├── onboarding-flow.md     # Onboarding 细节
│   └ audit-notes.md           # 设计说明
└── accounts/
    └ <account>/               # 按账号隔离
        ├── secrets/
        │   └ runtime.env      # AK/SK 等敏感信息
        │   └ cache.json       # 指纹缓存
        ├── list/              # 资源清单
        └ reports/             # 报告输出
```

## 设计原则

1. **账号隔离**：每个账号独立存储在 `accounts/<account>/` 下
2. **最小化**：只保留直接支持 skill 的文件，不添加冗余文档
3. **统一入口**：所有命令通过 `aliyunctl.go` 入口执行
4. **职责分离**：代码在 `internal/`，文档在 `references/`
5. **文档导航**：通过 `SKILL.md` 导航到具体文档文件