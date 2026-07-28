# ZTrade 策略使用指南

本指南默认约定：

- `ztrade` 已安装在系统 `PATH`
- 使用插件模式
- 回测示例默认输出 Markdown 报告

## 1. 使用流程

默认按这个顺序处理：

1. 定义策略参数。
2. 创建包含 `engine`、`position`、参数和指标的策略结构体。
3. 实现必需的生命周期方法。
4. 如需多周期，在 `Init` 中注册合成周期。
5. 在 K 线回调里生成交易信号。
6. 通过 `OnPosition` 同步仓位。
7. 编译策略插件。
8. 执行回测和实盘命令。

## 2. 必需契约

一个 ztrade 策略应实现：

- `Param() []Param`
- `Init(engine Engine, params ParamData) error`
- `OnCandle(candle *Candle)`
- `OnPosition(pos, price float64)`
- `OnTrade(trade *Trade)`
- `OnTradeMarket(trade *Trade)`
- `OnDepth(depth *Depth)`

推荐字段：

- `engine Engine`
- `position float64`
- 参数字段
- 指标字段
- 可选状态字段

构造函数命名：

- `NewXxx() *Xxx`

## 3. 最小策略骨架

```go
package strategy

import (
    . "github.com/ztrade/trademodel"
)

type Demo struct {
    engine   Engine
    position float64

    amount float64
    fast   int
    slow   int
}

func NewDemo() *Demo {
    return new(Demo)
}

func (d *Demo) Param() (paramInfo []Param) {
    paramInfo = []Param{
        FloatParam("amount", "仓位", "下单数量", 1, &d.amount),
        IntParam("fast", "EMA快线", "快线长度", 7, &d.fast),
        IntParam("slow", "EMA慢线", "慢线长度", 30, &d.slow),
    }
    return
}

func (d *Demo) Init(engine Engine, params ParamData) (err error) {
    d.engine = engine
    return nil
}

func (d *Demo) OnCandle(candle *Candle) {}
func (d *Demo) OnPosition(pos, price float64) { d.position = pos }
func (d *Demo) OnTrade(trade *Trade)       {}
func (d *Demo) OnTradeMarket(trade *Trade) {}
func (d *Demo) OnDepth(depth *Depth)       {}
```

策略代码应使用当前环境中 `ztrade` 程序所要求的符号命名。
如果本地版本暴露的辅助类型或 import 约定不同，应按该本地约定调整。

## 4. 参数与 `--param`

参数在 `Param()` 中声明。
通过 `--param` 传入 JSON 字符串。

示例：

```bash
ztrade backtest --script demo.so --param '{"amount":1,"fast":7,"slow":30}' --start "2020-01-01 08:00:00" --end "2021-01-01 08:00:00" --exchange binance --symbol BTCUSDT --markdown backtest.md
```

## 5. 多周期用法

ztrade 以 `1m` 为基础周期。
更大周期应在 `Init` 中通过 `Merge` 合成。

示例：

```go
func (d *Demo) Init(engine Engine, params ParamData) (err error) {
    d.engine = engine
    engine.Merge("1m", "15m", d.OnCandle15m)
    return nil
}

func (d *Demo) OnCandle15m(candle *Candle) {
    // 信号逻辑
}
```

推荐模式：

- 主要入场和出场信号放在合成周期回调中。
- 除非必须处理 1m 逻辑，否则 `OnCandle` 保持轻量。

## 6. 交易安全规则

始终遵循这些规则：

- 开多前先处理已有空仓。
- 开空前先处理已有多仓。
- 需要切换方向时优先 `CancelAllOrder()`。
- 不要把本地状态当成最终仓位，仓位以 `OnPosition` 为准。
- 如果策略有滑点，优先使用价格辅助函数。

## 7. 插件流程

编译插件：

```bash
ztrade build --script /path/to/demo.go --output demo.so
```

使用本地或私有依赖：

```bash
ztrade build --script /path/to/demo.go --output demo.so --moduleRoot /path/to/deps-module
```

忽略源码目录模块自动发现：

```bash
ztrade build --script /path/to/demo.go --output demo.so --ignoreSourceModuleRoot
```

回测 / 实盘：

```bash
ztrade backtest --script demo.so --start "2020-01-01 08:00:00" --end "2021-01-01 08:00:00" --exchange binance --symbol BTCUSDT --markdown backtest.md
ztrade trade --script demo.so --exchange binance --symbol BTCUSDT
```

回测报告输出参数：

- `--report` / `-o`：HTML 报告输出路径，默认 `report.html`
- `--markdown`：输出 Markdown 报告，而不是 HTML
- `--lang`：报告语言，可选 `en` 或 `zh`
- `--console`：直接把 JSON 结果打印到终端

示例：

```bash
ztrade backtest --script demo.so --start "2020-01-01 08:00:00" --end "2021-01-01 08:00:00" --exchange binance --symbol BTCUSDT --markdown backtest.md --lang zh
```
