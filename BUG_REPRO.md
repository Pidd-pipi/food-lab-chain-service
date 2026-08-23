# BUG_REPRO: flc-004 coldchain nil 路径（typed-nil 与 nil map）

## Bug 是什么
- `coldchain.Evaluator` 把 typed-nil 的默认策略装进接口返回，`Evaluate` 的 nil 判断被绕过，未配置默认策略时设备上报即 nil 指针 panic；
- `Registry` 的 labels map 未初始化，给新设备加标签时 `assignment to entry in nil map`；
- 首次 `SetRule` 写入未初始化的 rules map 同样 panic；
- `Summarize` 空输入返回 nil map，调用方写入即 panic。

## 如何触发
1. `coldchain.NewEvaluator(nil)` 后对任意设备调用 `Evaluate`；
2. `NewRegistry()` 注册设备后调用 `SetLabel`；
3. 空报告列表调用 `Summarize` 后写入返回的 map。

## 真实错误信息
```
panic: value method food-lab-chain-service/coldchain.Policy.Level called using nil *Policy pointer [recovered]
	panic: value method food-lab-chain-service/coldchain.Policy.Level called using nil *Policy pointer
goroutine 4 [running]:
food-lab-chain-service/coldchain.(*Policy).Level(...)
	.../coldchain/policy.go:43 +0xc0
```
加标签/首次设规则/空汇总分别报 `assignment to entry in nil map`。
