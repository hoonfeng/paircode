#!/usr/bin/env bash
# 项目验证脚本（Linux/macOS CI 用）
set -euo pipefail

echo "=== 1. go vet ==="
go vet ./...

echo "=== 2. go build ==="
go build ./cmd/companion

echo "=== 3. 循环依赖检测 ==="
go run ./scripts/check_cycles.go

echo "=== 4. go test (short) ==="
go test -short -count=1 ./cmd/companion/agent/... ./pkg/... ./internal/...

echo "✅ 全部通过"
