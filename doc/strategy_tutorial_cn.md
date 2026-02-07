# ztrade Strategy 教程（中文）

本教程面向“想在 ztrade 里写、跑、回测策略”的开发者，基于仓库中的现有策略写法（例如 `ema_simple`、`boll` 一类），整理出一套**最小可运行**且更清晰的说明。

> 核心理念：ztrade 的行情/成交/仓位等都通过事件驱动回调进入策略；K 线基础周期固定是 `1m`，更大周期用 `engine.Merge()` 在策略内部合成。

---

## 1. 两种运行策略的方式

ztrade 的脚本引擎支持两条路线：

### A. 编译为 Go Plugin（推荐，默认就能用）

- 你的策略文件（例如 `demo.go`）会被 `ztrade build` 编译成 `.so`（Linux）、`.dylib`（macOS）、`.dll`（Windows）。
- 运行/回测时把 `--script` 指向生成的插件文件。
- 优点：性能最好、对 ztrade 的默认构建方式兼容。

### B. 直接运行 `.go` 源码（需要 `ixgo` 构建标签）

- 需要把 ztrade 编译为带 `-tags ixgo` 的版本，才能直接 `--script your.go`。
- 优点：改代码快；缺点：需要特殊构建。

下面的教程以“写策略代码”为主，最后会分别给出两种方式的运行命令。

---

## 2. 策略必须实现的接口（回调函数）

策略本质是一个 `struct`，并实现以下方法（建议全部实现，即便某些为空）：

- `Param() []Param`：声明策略参数（用于命令行 `--param` 解析）
- `Init(engine Engine, params ParamData) error`：初始化，拿到 `engine` 并注册合成周期
- `OnCandle(candle *Candle)`：每根 `1m` K 线回调
- `OnPosition(pos, price float64)`：仓位变化回调（回测/实盘都会触发）
- `OnTrade(trade *Trade)`：自己的订单成交回调
- `OnTradeMarket(trade *Trade)`：市场成交（交易所推送）回调
- `OnDepth(depth *Depth)`：深度（order book）推送回调

重要：
- **ztrade 内部固定订阅的是 `1m` K 线**。
- 你要用 `15m/1h` 等更大周期，必须在 `Init` 中调用 `engine.Merge("1m", "15m", fn)`。

---

## 3. 最小可运行策略示例（建议从这里抄）

创建一个文件 `demo.go`：

```go
package strategy

import (
    "fmt"

    "github.com/ztrade/indicator"
    . "github.com/ztrade/trademodel"
)

type Demo struct {
    engine   Engine
    position float64

    // 参数
    largeBin int
    amount   float64
    fast     int
    slow     int

    // 指标
    ema indicator.CommonIndicator
}

func NewDemo() *Demo {
    return new(Demo)
}

func (d *Demo) Param() (paramInfo []Param) {
    return []Param{
        IntParam("bin", "大级别(分钟)", "合成K线周期(分钟)", 15, &d.largeBin),
        FloatParam("amount", "仓位", "每次开仓数量", 1, &d.amount),
        IntParam("fast", "EMA快线", "EMA fast", 7, &d.fast),
        IntParam("slow", "EMA慢线", "EMA slow", 30, &d.slow),
    }
}

func (d *Demo) Init(engine Engine, params ParamData) (err error) {
    d.engine = engine

    // 交叉指标：engine.AddIndicator("ema", fast, slow)
    d.ema = engine.AddIndicator("ema", d.fast, d.slow)

    // 合成更大周期K线：1m -> X m
    large := fmt.Sprintf("%dm", d.largeBin)
    engine.Merge("1m", large, d.OnCandleLarge)

    return nil
}

// 1m 回调（如果你只用合成周期，也可以留空）
func (d *Demo) OnCandle(candle *Candle) {}

func (d *Demo) OnCandleLarge(candle *Candle) {
    d.ema.Update(candle.Close)
    inds := d.ema.Indicator()

    if inds["crossUp"] == 1 {
        d.engine.CancelAllOrder()
        if d.position < 0 {
            d.engine.CloseShort(candle.Close, -d.position)
        }
        d.engine.OpenLong(candle.Close, d.amount)
        return
    }

    if inds["crossDown"] == 1 {
        d.engine.CancelAllOrder()
        if d.position > 0 {
            d.engine.CloseLong(candle.Close, d.position)
        }
        d.engine.OpenShort(candle.Close, d.amount)
        return
    }
}

func (d *Demo) OnPosition(pos, price float64) {
    d.position = pos
}

func (d *Demo) OnTrade(trade *Trade)       {}
func (d *Demo) OnTradeMarket(trade *Trade) {}
func (d *Demo) OnDepth(depth *Depth)       {}
```

这份示例基本对应 `strategy/` 目录中 EMA 交叉策略的主干模式：
- `Init` 里注册 `Merge`
- 合成周期回调里更新指标、产生交易信号
- `OnPosition` 里同步仓位

---

## 4. 参数系统：`Param()` 与命令行 `--param`

`Param()` 的返回值是一组 `Param` 描述，框架会用它来解析 `--param`。

- 参数的 key 是 `IntParam/StringParam/FloatParam/BoolParam` 的第一个入参
- `--param` 是 **JSON 字符串**，key 对应上面的参数 key

示例：

```bash
./ztrade backtest \
  --script demo.so \
  --param '{"bin":15,"amount":1,"fast":7,"slow":30}' \
  --start "2020-01-01 08:00:00" --end "2021-01-01 08:00:00" \
  --exchange binance --symbol BTCUSDT
```

---

## 5. Engine 常用 API（策略里最常用的那一小撮）

下列方法最常用：

- 下单/平仓：`OpenLong` / `CloseLong` / `OpenShort` / `CloseShort`
- 止损：`StopLong` / `StopShort`
- 撤单：`CancelOrder` / `CancelAllOrder`
- 仓位与余额：`Position()` / `Balance()`
- 指标：`AddIndicator(name string, params ...int)`
- 多周期K线：`Merge(src, dst string, fn CandleFn)`
- 日志与通知：`Log(...)` / `SendNotify(...)`

### 指标使用小贴士

`AddIndicator` 返回 `indicator.CommonIndicator`：

- `Update(price)`：喂入价格（通常用收盘价）
- `Result()`：主线结果
- `Indicator()`：返回一个 `map[string]float64`

例如 EMA 交叉指标一般会用到：

- `fast` / `slow`
- `crossUp`（金叉=1）/ `crossDown`（死叉=1）

---

## 6. K 线与 `engine.Merge()`（多周期）

ztrade 在回测/实盘都会先订阅 `1m` K 线。

- 你在 `OnCandle` 拿到的是 `1m`
- `Merge("1m", "15m", fn)` 会在内部聚合后，触发你的 `fn(*Candle)`

常见写法（参考 `ema_simple`）：

- 在 `Init` 里 `Merge` 一次
- 交易逻辑放到合成周期回调里（例如 `OnCandleLarge`），避免 1m 过于噪声

---

## 7. 编译与运行

### 7.1 编译策略为插件（推荐）

在 ztrade 可执行文件所在目录（或 `dist/`）执行：

```bash
./ztrade build --script /path/to/demo.go --output demo.so
```

然后用 `.so` 回测：

```bash
./ztrade backtest \
  --script demo.so \
  --param '{"bin":15,"amount":1,"fast":7,"slow":30}' \
  --start "2020-01-01 08:00:00" --end "2021-01-01 08:00:00" \
  --exchange binance --symbol BTCUSDT
```

实盘：

```bash
./ztrade trade \
  --script demo.so \
  --param '{"bin":15,"amount":1,"fast":7,"slow":30}' \
  --exchange binance --symbol BTCUSDT
```

### 7.2 直接运行 `.go`（需要 `ixgo` 版本的 ztrade）

编译 ztrade（带 `-tags ixgo`）：

```bash
CGO_ENABLED=0 go build -ldflags="-checklinkname=0" -tags ixgo -o dist/ztrade ./
```

然后你就可以直接：

```bash
./ztrade backtest --script /path/to/demo.go ...
```

---

## 8. 常见坑（很重要）

1. **`Init` 建议返回 `error`**：框架 Runner 接口是 `Init(...) error`，不返回会导致插件/解释器模式不兼容。
2. **K线基础永远是 `1m`**：你想用 `1h/4h` 必须 `Merge`。
3. **仓位来源以 `OnPosition` 为准**：不要只靠自己累加推断仓位。
4. **确保有构造函数 `NewXxx()`**：无论是插件模式还是 ixgo 模式，都依赖 `New{Struct}` 来实例化策略。

---

## 9. 下一步建议

- 先从 `ema_simple` 这类“单指标 + 合成周期”的策略开始。
- 再逐步引入：止损单（`StopLong/StopShort`）、多指标组合、风控模块（risk processer）。
