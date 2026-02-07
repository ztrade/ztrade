# 架构优化计划

> ✅ 已完成 | 🔲 待实施

## ✅ 1. 回测固定使用最小粒度 K 线，策略通过 Merge 合成大周期

**优先级**：🟡 中 | **状态：已完成**

**问题原因**：
回测入口 `back.go` 的 `Run()` 方法中硬编码了 `bSize := "1m"`，局部变量命名不清晰，且缺少对设计意图的说明。

**解决方案（已实施）**：
- 参考 vnpy 的设计思路：回测固定使用最小粒度（1m）K 线数据
- 策略通过 `engine.Merge()` 合成更大周期（5m、1h 等），无需从数据库预加载多周期数据
- `Run()` 方法中定义 `const binSize = "1m"` 常量，语义清晰
- 移除了不必要的 `binSize` 字段和 `SetBinSize()` 方法

---

## ✅ 2. 回测与实盘的初始化流程统一

**优先级**：🔴 高 | **状态：已完成**

**问题原因**：
回测使用 `NewScript()`（`script.go`）内部创建一个全新的 `GoEngine`，而实盘使用外部创建的 `GoEngine` + `AddScript()`。两条路径的策略加载逻辑完全不同，可能导致行为差异。

**解决方案（已实施）**：
- `back.go` 的 `Run()` 改为直接使用 `goscript.NewGoEngine()` + `AddScript()`，与 `trade.go` 一致
- `NewScript()` 添加 `Deprecated` 注释，建议直接使用 `GoEngine`
- 两条路径现在使用完全相同的策略加载方式

---

## ✅ 3. 统一的配置注入机制

**优先级**：🟡 中 | **状态：已完成**

**问题原因**：
`Trade` 通过包级全局变量 `cfg` 获取配置，各组件获取配置的方式不一致，组件难以独立测试。

**解决方案（已实施）**：
- `Trade` 结构体新增 `cfg` 字段，内部使用 `b.cfg` 而非全局变量
- 新增 `NewTradeWithConfig(cfg, exchange, symbol)` 方法，支持显式注入配置
- 保留 `NewTrade(exchange, symbol)` 和 `SetConfig()` 作为向后兼容 API
- 所有配置引用从全局变量改为实例字段

---

## ✅ 4. Processer 的生命周期管理

**优先级**：🔴 高 | **状态：已完成**

**问题原因**：
`Processers.Stop()` 在遇到第一个错误时立即返回，导致后续 Processer 无法被清理。Stop 按添加顺序执行，可能导致数据丢失。`WaitClose` 用固定的 `time.Sleep` 等待，不够可靠。

**解决方案（已实施）**：
- `Stop()` 改为**逆序**停止（先停下游消费者，再停上游生产者）
- `Stop()` 收集所有错误，使用 `errors.Join` 聚合，不再提前返回
- `WaitClose()` 改为先调用 `bus.WaitEmpty()` 等待事件队列清空，再调用 `bus.Close()`
- `WaitClose()` 添加 timeout 安全机制，超时后强制关闭并警告

---

## ✅ 5. 标准化的回测结果输出接口

**优先级**：🟢 低 | **状态：已完成**

**问题原因**：
`Backtest.Result()` 是空方法。回测结果完全依赖外部传入的 `Reporter`，而 `Reporter` 接口只有写入方法没有读取方法。获取结果需要知道具体实现类型。

**解决方案（已实施）**：
- `rpt` 包新增 `ResultProvider` 接口，定义 `ProvideResult() (any, error)` 方法
- `report.Report` 实现 `ResultProvider` 接口（桥接方法委托到 `GetResult()`）
- `Backtest.Result()` 返回 `(any, error)`，内部检查 Reporter 是否实现 `ResultProvider`
- 向后兼容：现有 Reporter 不需要实现新接口

---

## ✅ 6. 风控中间层

**优先级**：🟡 中 | **状态：已完成**

**问题原因**：
缺少独立的风控层来处理仓位限制、每日亏损熔断、下单频率限制、异常价格检测。策略 bug 可能导致无限下单或巨额亏损。

**解决方案（已实施）**：
- 新增 `pkg/process/risk/risk.go`，实现 `RiskManager` Processer
- 采用**守护者/观察者**模式：订阅交易事件，监控仓位和盈亏
- 支持配置参数：
  - `MaxPosition`: 最大持仓量
  - `MaxDailyLoss`: 每日最大亏损比例
  - `MaxOrderRate`: 最大下单频率（次/分钟）
  - `PriceDeviation`: 价格偏离阈值
- 当限制被触发时：发送 `CancelAll` 取消所有挂单 + 强制平仓 + 通知告警
- 通过 `SetRiskConfig()` 在 `Backtest` 和 `Trade` 中可选启用
- 采用 **事后守护** 模式（非前置拦截）：在 pub-sub 事件总线中，所有 EventOrder 订阅者按注册顺序接收事件，交易所先执行订单，风控再监控结果并在异常时发出 CancelAll/强制平仓
- 注意：当前架构下风控不能阻止订单发出，只能在检测到违规后进行补救操作

---

## ✅ 7. 多品种支持（预留）

**优先级**：🟢 低 | **状态：预留方向**

**问题原因**：
`Trade` 和 `Backtest` 都绑定单个 `symbol`，多品种策略（套利、配对交易）需要启动多个实例，实例之间无法共享策略状态。

**后续迭代方向**：
- Runner 接口扩展多品种声明（如 `MultiSymbolRunner`）
- Engine 支持跨品种下单（如 `DoOrderForSymbol()`）
- VExchange 支持多品种订单匹配和持仓跟踪
- TradeExchange 支持多品种订阅
- 按品种路由事件（EventCandle 携带 symbol 标签）
