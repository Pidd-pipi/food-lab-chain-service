# Food Lab Chain Service

This standard-library service records food-safety specimen custody and handoffs between collection and laboratory teams. Besides the specimen chain API it ships three reusable business modules for the lab workflow:

- `intake`: specimen batch intake — create a batch, append specimen lines, validate against intake policy, filter/summarize, and process or reconcile batches under a concurrency limiter.
- `coldchain`: cold-storage temperature monitoring — ring-buffer reading history, excursion counting, per-device alert policies, a worker-pool dispatcher, a device registry, and daily excursion reports.
- `custody`: custody traceability — assemble specimen chains from chronological events, filter/page chains, group by food type, and export them as CSV or TSV.

The `ops_*` files in the root package form an operations layer (records, optimistic revision, state machine, audit log, policy checks, 112 control rules) exposed over HTTP under `/api/ops`, together with the enterprise server middleware.

## Layout

`config`, `domain`, `store`, `validation`, `health`, `api`, `intake`, `coldchain`, and `trace` hold the service layers. The small browser client is embedded from `web`.

## Run and test

The optional `PORT` environment variable defaults to `8080`.

```text
cd backend && go build ./...
cd backend && go test ./...
cd backend && PORT=8080 go run .
```

Use `GET /healthz`, `GET /api/specimens`, and `POST /api/specimens/spec-01/handoff` with `{"to":"Microbiology Bench"}`. Empty recipients return `400`; sealed specimens return `409`.

The operations API lives under `/api/ops`:

```text
GET  /api/ops/records?status=active&page=1&pageSize=25
POST /api/ops/records                          {"id":"op-1","subject":"...","owner":"...","priority":"high","labels":{"site":"west"}}
GET  /api/ops/records/{id}
POST /api/ops/records/{id}/transition          {"target":"active","expected":0}
GET  /api/ops/snapshot
```

## Verification

Ran `gofmt -w .`, `go build ./...`, and `go test ./...` successfully. A live service on `PORT=18184` returned `200` for health, specimen collection, `/`, and `/app.js`; a valid handoff returned `200`, and an empty recipient returned `400`. The process was stopped in the same smoke operation after verification.

## Engineering Notes

食品样本链路流程代码按领域模型、校验、状态转换、并发安全存储、审计事件和 HTTP 生命周期分层。请求会保留请求标识并经过恢复与超时保护；状态写入使用版本校验，错误通过可识别的领域错误返回。批处理与监控模块都围绕上下文取消、并发限额和资源释放设计，避免句柄与 goroutine 泄漏。

## Enterprise Layout

```text
.
├── backend/       # Go module, source, tests, and embedded web assets
│   ├── api/       # specimen chain HTTP handlers
│   ├── intake/    # batch intake workflow
│   ├── coldchain/ # temperature monitoring and alerting
│   ├── custody/  # custody traceability and export
│   └── web/       # embedded browser client
├── database/      # persistence documentation and future schemas
├── output/        # verification records
└── runtime_smoke.json
```

Run `cd backend && go test ./...`, `cd backend && go build ./...`, or `cd backend && PORT=8080 go run .`. The health check is `GET /healthz`; the main API endpoints are `GET /api/specimens` and `POST /api/specimens/{id}/handoff`, plus the operations API under `/api/ops`.
