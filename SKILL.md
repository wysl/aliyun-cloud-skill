---
name: aliyun-cloud-monitor
description: Use for Alibaba Cloud monitoring and account-isolated operations. Supports account initialization, permission fixing, secret fingerprinting, resource inventory refresh, billing summaries, account balance queries, ECS/RDS/PolarDB/ALB operations, resource usage monitoring (CPU/memory/disk/IOPS/bandwidth/connections/QPS/error rate), security group management (list security groups with bound instances, query security group rules, add ingress/egress rules, join/leave security group), ALB ACL management (list ACLs, list ACL entries, query listener ACL config), VPC management (list VPCs, query VPC detail with vswitches/route tables/route entries), SMS sending statistics (24h volume, success rate), domain management with SSL certificates, CDN usage statistics (traffic, source traffic, hit rate), CDN auto warmup from recent OSS uploads, OSS bucket usage statistics (storage, object count, traffic, request count), Prometheus/Grafana queries, SSL checks, resource package checks, OSS inspection, CDN operations, and SLS queries. Use when auditing or operating Aliyun accounts isolated under accounts/<account>/.
---

# Aliyun Cloud Monitor

Use this skill for Alibaba Cloud monitoring with isolated per-account configuration.

## Directory model

Keep each account isolated under:
- `accounts/<account>/secrets`
- `accounts/<account>/list`
- `accounts/<account>/reports`

Keep the skill repository minimal. Only keep files that directly support the skill.
Do not add extra README-style documents unless they contain reusable operator knowledge that belongs in `references/`.

## Primary entrypoint

Use the Go entrypoint:

```bash
go run ./scripts/aliyunctl.go <command> ...
```

Core examples:

```bash
go run ./scripts/aliyunctl.go init-account --account prod-main
go run ./scripts/aliyunctl.go fix-permissions --account prod-main
go run ./scripts/aliyunctl.go hash-secrets --account prod-main
go run ./scripts/aliyunctl.go refresh --account prod-main
go run ./scripts/aliyunctl.go account-balance --account prod-main --format summary
go run ./scripts/aliyunctl.go bill-summary --account prod-main --cycle 2026-03 --format summary
go run ./scripts/aliyunctl.go ecs-list --account prod-main --region cn-shanghai --format summary
go run ./scripts/aliyunctl.go ecs-usage --account prod-main --format summary
go run ./scripts/aliyunctl.go rds-usage --account prod-main --format summary
go run ./scripts/aliyunctl.go polardb-list --account prod-main --format summary
go run ./scripts/aliyunctl.go polardb-usage --account prod-main --format summary
go run ./scripts/aliyunctl.go alb-list --account prod-main --format summary
go run ./scripts/aliyunctl.go alb-usage --account prod-main --format summary
go run ./scripts/aliyunctl.go sms-stats --account prod-main --format summary
go run ./scripts/aliyunctl.go sg-list --account prod-main --region cn-shanghai --format summary
go run ./scripts/aliyunctl.go sg-rules --account prod-main --region cn-shanghai --sg-id sg-xxx --format summary
go run ./scripts/aliyunctl.go sg-add-ingress --account prod-main --region cn-shanghai --sg-id sg-xxx --port 80/80 --source 0.0.0.0/0 --protocol TCP --policy Accept
go run ./scripts/aliyunctl.go sg-add-egress --account prod-main --region cn-shanghai --sg-id sg-xxx --port -1/-1 --dest 0.0.0.0/0 --protocol ALL --policy Accept
go run ./scripts/aliyunctl.go sg-join --account prod-main --region cn-shanghai --sg-id sg-xxx --instance-id i-xxx
go run ./scripts/aliyunctl.go sg-leave --account prod-main --region cn-shanghai --sg-id sg-xxx --instance-id i-xxx
go run ./scripts/aliyunctl.go alb-acl-list --account prod-main --region cn-shanghai --format summary
go run ./scripts/aliyunctl.go alb-acl-entries --account prod-main --region cn-shanghai --acl-id acl-xxx --format summary
go run ./scripts/aliyunctl.go alb-listener-acl --account prod-main --region cn-shanghai --format summary
go run ./scripts/aliyunctl.go vpc-list --account prod-main --region cn-shanghai --format summary
go run ./scripts/aliyunctl.go vpc-detail --account prod-main --region cn-shanghai --vpc-id vpc-xxx --format summary
go run ./scripts/aliyunctl.go vswitch-list --account prod-main --region cn-shanghai --format summary
go run ./scripts/aliyunctl.go routetable-list --account prod-main --region cn-shanghai --format summary
go run ./scripts/aliyunctl.go vswitch-resources --account prod-main --region cn-shanghai --vswitch-id vsw-xxx --format summary
go run ./scripts/aliyunctl.go domain-list --account prod-main --format summary
go run ./scripts/aliyunctl.go cdn-usage --account prod-main --format summary
go run ./scripts/aliyunctl.go cdn-auto-warmup --account prod-main --bucket my-bucket --hours 1 --format summary
go run ./scripts/aliyunctl.go oss-usage --account prod-main --format summary
go run ./scripts/aliyunctl.go prom-query --account prod-main --query 'up{job="ecs"}' --format summary
```

## Supported workflows

- Initialize an isolated account skeleton
- Fix secret file permissions
- Generate secret fingerprints for change detection
- Refresh Alibaba Cloud resources through ResourceCenter
- Query account balance and billing summaries
- Monitor ECS, RDS, PolarDB, ALB resource usage (CPU, memory, disk, IOPS, bandwidth, connections, QPS, error rate)
- List security groups with bound ECS instances by region
- Query security group rules (ingress/egress)
- Add ingress/egress rules to security groups
- Add/remove ECS instances to/from security groups
- List ALB ACLs and entries by region
- Query listener ACL configuration (white/black list mode)
- List VPCs with CIDR blocks and status
- Query VPC detail (vswitches, route tables, route entries)
- Query cloud resources in a VSwitch (ECS instances, ALB zone mappings)
- Query SMS sending statistics (24h volume, success rate)
- List domains with expiration dates and SSL certificates
- Query CDN usage statistics (traffic, source traffic, cache hit rate)
- Auto warmup CDN from recent OSS uploads
- Query OSS bucket usage statistics (storage, object count)
- Query billing, ECS, RDS, Prometheus, SSL, resource package, OSS, CDN, and SLS data
- Return either concise summaries or JSON depending on the command
- Read `references/commands.md` for the complete command list

## Onboarding

Collect in this order:
1. account name
2. AK
3. SK
4. region list
5. optional Grafana URL
6. optional Grafana admin user/password
7. optional Feishu webhook

Then:
1. run `init-account`
2. write values into `accounts/<account>/secrets/runtime.env`
3. run `fix-permissions`
4. run `hash-secrets`

Read `references/onboarding-flow.md` for the exact collection order.

## Notes

- Prefer replying in the current conversation by default.
- Treat secret fingerprints as inventory/change-detection only, not encryption.
- Keep operational guidance concise; detailed design notes belong in `references/`.
- Extend the Go entrypoint instead of adding many small one-off scripts.
- Validate key commands before publishing changes. See `references/release-checklist.md`.
