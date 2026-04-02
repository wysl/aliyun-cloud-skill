# Interactive onboarding flow

Collect in this order:
1. 账户名称
2. AK
3. SK
4. 区域列表
5. Grafana URL（可空）
6. Grafana 管理员账号（可空）
7. Grafana 管理员密码（可空）
8. 飞书 webhook（可空）

Then:
1. `go run ./scripts/aliyunctl.go init-account --account <name>`
2. 填写 `accounts/<name>/secrets/runtime.env`
3. `go run ./scripts/aliyunctl.go fix-permissions --account <name>`
4. `go run ./scripts/aliyunctl.go hash-secrets --account <name>`
