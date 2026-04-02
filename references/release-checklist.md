# Release checklist

Run a small representative validation set before treating the skill as publish-ready.

## Minimum command checks

- `go run ./scripts/aliyunctl.go`
- `go run ./scripts/aliyunctl.go init-account --account test-account`
- `go run ./scripts/aliyunctl.go env-check --account test-account`
- `go run ./scripts/aliyunctl.go bill-summary --account <account> --cycle <yyyy-mm> --format summary`
- `go run ./scripts/aliyunctl.go ecs-list --account <account> --region <region> --format summary`
- `go run ./scripts/aliyunctl.go prom-query --account <account> --query 'up' --format summary`
- `go run ./scripts/aliyunctl.go rds-list --account <account> --region <region> --format summary`
- `go run ./scripts/aliyunctl.go ssl-summary --account <account> --format summary`
- `go run ./scripts/aliyunctl.go resource-package-summary --account <account> --format summary`
- `go run ./scripts/aliyunctl.go oss-list --account <account> --format summary`
- `go run ./scripts/aliyunctl.go cdn-list --account <account> --format summary`
- `go run ./scripts/aliyunctl.go sls-list-projects --account <account>`

## Review points

- Command names are consistent.
- `--format` behavior is predictable.
- Error messages are actionable.
- No temporary `.verify*` files remain.
- No extra docs beyond `SKILL.md` and necessary `references/` files.
