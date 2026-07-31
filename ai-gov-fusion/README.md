# AI-GOV Fusion

TokenHub 融合基线 —— 企业级 AI 智能网关治理平台（Token 治理底座）

基线版本: v3.2.0
上游: TokenHub v0.4.0 (Apache 2.0)

## 目录结构
- backend/ - Go 后端（GORM + net/http）
  - internal/server/ - 存量代码 + 11 个新增治理包
- frontend/ - Next.js 管理控制台
- schema/ - DDL（40 表）
- deploy/ - Docker Compose + systemd

## 新增包
fund | pricing | idempotency | party | authz | routing | modelgrant | security | abac | ui_permission | audit

## 快速开始
./start.sh
