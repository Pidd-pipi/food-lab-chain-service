# BUG_REPRO: flc-010 服务退出泄漏与响应头失效

## Bug 是什么
- `runtime.go serveHTTP` 用无缓冲 errCh 且退出分支不回收 `ListenAndServe` goroutine，优雅退出后 goroutine 挂住、内存不降；
- `Shutdown` 用 `context.Background()` 无超时兜底，慢连接时进程退不出；
- `requestIDMiddleware` 用非原子计数且不回写 `X-Request-ID` 响应头；
- `opsEnterpriseMiddleware` 在响应写完后才设置 `X-Operations-Latency-Ms`，头被 net/http 丢弃，耗时字段永不出现。

## 如何触发
1. 启动服务后向进程发 SIGTERM 优雅退出，连续两轮后观察 goroutine 数增长；
2. 任意请求后检查响应头。

## 真实错误信息
- 连续两次优雅退出后 goroutine 数不回落（`TestServeHTTPLeaksNoGoroutine`）；
- 响应头缺 `X-Operations-Latency-Ms` 与 `X-Request-ID`；
- 并发请求下非原子计数触发 data race（`TestParallelRequestIDs`）。
