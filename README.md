# 设计稿协作批注与版本评审平台

> 编号 GO-CG-043 / 代码生成基线 / 离线可运行

从 0 到 1 构建的设计稿评审系统：管理画布标注、讨论、状态流转、版本对比与评审纪要。所有依赖（包括附件存储、通知、审计与回调）都提供**本地可验证适配器**，不调用任何第三方接口、CDN、云存储、在线模型或真实消息服务。

## 模块职责

| 目录 | 职责 |
| --- | --- |
| `cmd/server` | 进程入口，装配 wire 与优雅停机 |
| `internal/config` | 环境变量 + 默认值加载，敏感字段脱敏 |
| `internal/domain` | 领域实体、值对象、不变式与状态机（`project`、`canvas`、`annotation`、`review`、`audit`） |
| `internal/application` | 用例接口、DTO、TxRunner 与 Clock/ID 抽象 |
| `internal/service` | 用例实现，组合仓储 + 审计 + 通知 + 存储；事务边界在此声明 |
| `internal/repository/memory` | 进程内仓储实现（含乐观锁、唯一约束、并发安全），用于离线默认运行与测试 |
| `internal/transport/http` | Gin 路由、Handler、错误码与 OpenAPI 描述 |
| `internal/middleware` | RequestID、ActorID、Recovery、SecurityHeaders、CORS、AuditLog |
| `internal/platform/storage` | 本地文件附件存储，类型 + 大小 + 路径穿越校验 |
| `internal/platform/audit` | 内存审计日志适配器 |
| `internal/platform/notify` | 内存消息收件箱（未读提醒） |
| `internal/platform/events` | 进程内事件总线（用于离线回调扩展） |
| `internal/seed` | 启动时加载的演示数据（幂等） |
| `migrations` | 可重复执行的 SQL 迁移与种子数据 |
| `api/openapi` | OpenAPI 3.0 规范 |
| `web` | Vue 3 + TypeScript + Vite + Pinia 中文前端 |
| `scripts` | 运维脚本占位 |
| `tests/integration` | 可选的 PostgreSQL 集成测试（构建标签 `integration`） |

## 本地启动

### 后端

```bash
cp .env.example .env   # 可选：调整端口与存储目录
go run ./cmd/server
```

默认监听 `:8080`，使用内存运行时与内置演示数据。`/healthz` 与 `/readyz` 可用作探针。

### 前端

```bash
cd web
npm install
npm run dev    # 开发服务器，默认 http://localhost:5173
npm run build  # 生产构建到 web/dist
npm run test   # Vitest 单元测试
```

Vite 开发服务器已配置代理：`/api`、`/healthz`、`/readyz` 转发到 `http://localhost:8080`。

## 配置

所有配置项见 `.env.example`。关键字段：

| 环境变量 | 默认 | 说明 |
| --- | --- | --- |
| `RUNTIME_MODE` | `memory` | `memory`（离线默认）或 `postgres` |
| `SEED_ON_STARTUP` | `true` | 启动时加载演示数据（幂等，不覆盖已有） |
| `HTTP_ADDR` | `:8080` | HTTP 监听地址 |
| `STORAGE_BASE_DIR` | `./var/attachments` | 附件本地存储根目录 |
| `STORAGE_MAX_BYTES` | `20971520` | 单个附件上限 20 MiB |
| `STORAGE_ALLOWED_TYPES` | `image/png,image/jpeg,image/webp,image/gif,application/pdf,text/plain` | 允许的 MIME 类型白名单 |
| `POSTGRES_DSN` | `postgres://postgres:postgres@localhost:5432/design_review?sslmode=disable` | 仅 `postgres` 模式使用 |
| `DEFAULT_USER_ID` | `user-local` | 默认离线用户 ID（来自 `X-User-ID` 头） |

## 迁移与演示数据

```bash
# 仅 RUNTIME_MODE=postgres 时需要
psql -d design_review -f migrations/0001_init.up.sql
psql -d design_review -f migrations/seed.sql

# 回滚
psql -d design_review -f migrations/0001_init.down.sql
```

迁移脚本幂等（使用 `CREATE TABLE IF NOT EXISTS`、`ON CONFLICT DO NOTHING`），可重复执行；演示数据不会覆盖已有行。

## 接口示例

完整 OpenAPI 规范见 `api/openapi/openapi.yaml`。所有列表接口统一支持 `page`、`page_size`、`sort_by`、`order` 与白名单筛选，错误响应包含 `code`、`message`、`field_errors`、`request_id`。

```bash
# 创建项目
curl -X POST http://localhost:8080/api/v1/projects \
  -H 'Content-Type: application/json' \
  -d '{"id":"p1","name":"首页改版","description":"Q4 评审"}'

# 创建画板
curl -X POST http://localhost:8080/api/v1/boards \
  -H 'Content-Type: application/json' \
  -d '{"id":"b1","project_id":"p1","name":"桌面端","width":1440,"height":1024}'

# 创建版本并发布
curl -X POST http://localhost:8080/api/v1/versions \
  -H 'Content-Type: application/json' \
  -d '{"id":"v1","board_id":"b1","number":1,"preview_key":"preview-v1.png"}'
curl -X POST http://localhost:8080/api/v1/versions/v1/publish

# 创建批注（点锚）
curl -X POST http://localhost:8080/api/v1/annotations \
  -H 'Content-Type: application/json' \
  -d '{"id":"a1","project_id":"p1","board_id":"b1","version_id":"v1","title":"对比度不足","point":{"x":120,"y":40},"priority":"high","assignee_id":"designer-1"}'

# 添加回复（Open → Replied）
curl -X POST http://localhost:8080/api/v1/annotations/a1/replies \
  -H 'Content-Type: application/json' \
  -d '{"body":"已修复，请复核"}'

# 提交复核 → 解决
curl -X POST http://localhost:8080/api/v1/annotations/a1/request-review
curl -X POST http://localhost:8080/api/v1/annotations/a1/resolve

# 复核重新打开（唯一返回路径）
curl -X POST http://localhost:8080/api/v1/annotations/a1/reopen \
  -H 'Content-Type: application/json' \
  -d '{"reason":"对比度仍然不达标"}'

# 批量迁移锚点（新版本发布后）
curl -X POST http://localhost:8080/api/v1/annotations/migrate-anchors \
  -H 'Content-Type: application/json' \
  -d '{"from_version_id":"v1","to_version_id":"v2","reason":"版本切换"}'

# 本地导出 CSV
curl http://localhost:8080/api/v1/export/annotations.csv?project_id=p1 -o annotations.csv
```

## 状态规则

批注状态机（`internal/domain/annotation`）：

```
open     -> replied | review | closed
replied  -> review | open | closed
review   -> resolved | open | closed
resolved -> open   (仅通过复核重新打开，唯一返回路径)
closed   -> (终态)
```

强制不变式：

1. **解决批注必须至少有一条回复**：`Resolve()` 调用前若 `len(Replies)==0` 返回 `ErrCannotResolveWithoutReply`。
2. **已解决的批注只能通过复核重新打开**：`Reopen()` 仅在 `Status==Resolved` 时允许，其它状态返回 `ErrCannotReopenUnresolved`。
3. **版本替换不得静默丢失原批注**：发布新版本时不会自动删除旧批注；调用 `MigrateAnchors` 时使用事务批量迁移，并写入审计。`MarkStale` 给旧版本批注打上 `[已失效]` 前缀，便于人工重定位。
4. **附件类型 + 大小 + 路径穿越校验**：`Store.ValidateUpload` 拒绝白名单外的 MIME 类型、超过上限的字节数，以及包含 `/\` 的文件名。
5. **乐观并发控制**：仓储层在 `Update` 时校验 `expectedVersion`；并发写入会观察到 `STALE_VERSION` 错误（409）。
6. **唯一约束**：版本号在画板内唯一，成员在项目内唯一，迁移脚本可重复执行。

## 测试命令

```bash
# 后端
go fmt ./...
go vet ./...
go build ./...
go test ./...
go test -race ./...

# 前端
cd web
npm install
npm run test
npm run build

# 集成测试（可选，需 PostgreSQL）
go test -tags=integration ./tests/integration/...
```

后端测试自包含：`go test ./...` 不依赖运行中的 PostgreSQL 实例；进程内仓储实现提供乐观锁、唯一约束与并发安全的等价语义。

## 实际验证结果

下述命令均在交付前实际执行：

- `gofmt -l .` —— 无输出（全部符合格式）。
- `go build ./...` —— 通过。
- `go vet ./...` —— 无告警。
- `go test ./...` —— 全部通过（领域、仓储、HTTP、并发）。
- `go test -race ./...` —— 通过。
- 前端 `npm install`、`npm run test`、`npm run build` —— 通过。

## 安全与运维

- 不提交密钥、依赖缓存、构建产物或运行期数据（见 `.gitignore`）。
- `Config.String()` 在打印 DSN 时对密码进行脱敏。
- 安全响应头：`X-Content-Type-Options`、`X-Frame-Options`、`X-XSS-Protection`、`Referrer-Policy`、`Content-Security-Policy`。
- 优雅停机：捕获 `SIGINT` / `SIGTERM`，等待在途请求最多 `HTTP_SHUTDOWN_TIMEOUT`。
- Panic 恢复中间件捕获 handler 中的 panic，返回稳定的 500 错误并记录堆栈与 `request_id`。
- 审计日志：每次 HTTP 请求与领域动作都写入审计表，可按 `actor_id`、`entity_type`、`entity_id`、时间区间查询。
