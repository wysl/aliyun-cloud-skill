# Command reference

Use the unified Go entrypoint:

```bash
go run ./scripts/aliyunctl.go <command> ...
```

## Account and setup

- `init-account` — create isolated account skeleton
- `fix-permissions` — fix secret directory/file permissions
- `hash-secrets` — generate secret fingerprints into `cache.json`
- `env-check` — validate required env keys
- `refresh` — refresh resource inventory through ResourceCenter

## Billing

- `bill-summary` — query billing overview summary
- `account-balance` — query account available balance

## ECS

- `ecs-list` — list ECS instances
- `ecs-detail` — show ECS instance detail
- `ecs-usage` — show ECS resource usage (CPU, memory, disk)
- `ecs-start` — start ECS instance
- `ecs-stop` — stop ECS instance
- `ecs-reboot` — reboot ECS instance

## Prometheus / Grafana

- `prom-query` — run instant query through Grafana datasource proxy
- `prom-range` — run range query through Grafana datasource proxy
- `prom-labels` — list Prometheus labels
- `prom-label-values` — list values for one label
- `prom-series` — list matching series

## RDS

- `rds-list` — list RDS instances
- `rds-detail` — show RDS instance detail
- `rds-usage` — show RDS resource usage (CPU, memory, IOPS)
- `rds-performance` — show RDS resource usage
- `rds-list-backups` — list RDS backups

## PolarDB

- `polardb-list` — list PolarDB clusters
- `polardb-usage` — show PolarDB resource usage (CPU, memory, IOPS)

## ALB

- `alb-list` — list ALB instances
- `alb-usage` — show ALB resource usage (bandwidth, active connections, QPS, error rate)
- `alb-acl-list` — list ALB ACLs
- `alb-acl-entries` — list entries in an ACL
- `alb-listener-acl` — show listener ACL configuration (white/black list mode)

## VPC

- `vpc-list` — list VPCs in a region
- `vpc-detail` — show VPC detail (vswitches, route tables, route entries)
- `vswitch-list` — list VSwitches in a region or VPC
- `routetable-list` — list route tables in a region or VPC
- `vswitch-resources` — list cloud resources in a VSwitch (ECS instances, ALB zone mappings)

## SMS

- `sms-stats` — show SMS sending statistics (24h volume, success rate)

## Security Group

- `sg-list` — list security groups with bound ECS instances by region
- `sg-rules` — show security group rules (ingress/egress)
- `sg-add-ingress` — add ingress rule to security group
- `sg-add-egress` — add egress rule to security group
- `sg-join` — add ECS instance to security group
- `sg-leave` — remove ECS instance from security group

## Domain

- `domain-list` — list domains with expiration dates and SSL certificates

## SSL certificates

- `ssl-list` — list SSL certificates
- `ssl-expiring` — list certificates expiring soon
- `ssl-summary` — summarize SSL certificate health

## Resource packages

- `resource-package-list` — list resource packages
- `resource-package-expiring` — list expiring resource packages
- `resource-package-summary` — summarize resource package health

## OSS

- `oss-list` — list OSS buckets
- `oss-info` — show OSS bucket info
- `oss-ls` — list OSS objects
- `oss-recent` — list recent OSS objects
- `oss-usage` — show OSS bucket usage (storage, object count)

## CDN

- `cdn-list` — list CDN domains
- `cdn-detail` — show CDN domain detail
- `cdn-traffic` — show CDN traffic data
- `cdn-usage` — show CDN usage statistics (traffic, source traffic, cache hit rate)
- `cdn-bandwidth` — show CDN bandwidth data
- `cdn-refresh` — refresh CDN paths
- `cdn-push` — push CDN URLs for preload/warmup
- `cdn-auto-warmup` — auto warmup CDN from recent OSS uploads

## SLS

- `sls-list-projects` — list SLS projects
- `sls-list-logstores` — list SLS logstores
- `sls-get-index` — get SLS index config
- `sls-update-index` — update SLS index with `content` field
- `sls-get-logs` — get raw SLS logs
- `sls-query-ips` — extract/query IP information from SLS logs

## Notes

- Prefer `--format summary` for operator-facing output.
- Use `--format json` when piping to tools or inspecting raw structures.
- Some commands require additional env values beyond AK/SK, especially Prometheus/Grafana queries.
