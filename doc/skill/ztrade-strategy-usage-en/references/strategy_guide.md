# ZTrade Strategy Usage Guide

This guide assumes:

- `ztrade` is installed in the system `PATH`
- plugin mode is the default workflow
- backtest examples should write Markdown reports

## 1. Workflow

Use this order by default:

1. Define strategy parameters.
2. Create a strategy struct with `engine`, `position`, params, and indicators.
3. Implement the required lifecycle methods.
4. Register merged timeframes in `Init` if needed.
5. Put signal generation in candle callbacks.
6. Keep position synchronized through `OnPosition`.
7. Build the strategy plugin.
8. Run backtest and trade commands.

## 2. Required contract

A ztrade strategy should implement:

- `Param() []Param`
- `Init(engine Engine, params ParamData) error`
- `OnCandle(candle *Candle)`
- `OnPosition(pos, price float64)`
- `OnTrade(trade *Trade)`
- `OnTradeMarket(trade *Trade)`
- `OnDepth(depth *Depth)`

Recommended fields:

- `engine Engine`
- `position float64`
- parameter fields
- indicator fields
- optional state fields

Constructor naming:

- `NewXxx() *Xxx`

## 3. Minimal strategy skeleton

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
        FloatParam("amount", "Amount", "Order amount", 1, &d.amount),
        IntParam("fast", "EMA fast", "Fast EMA length", 7, &d.fast),
        IntParam("slow", "EMA slow", "Slow EMA length", 30, &d.slow),
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

Use the symbol names expected by the `ztrade` program in your environment.
If the local build exposes helper aliases differently, adjust imports and types to that local contract.

## 4. Parameters and `--param`

Parameters are declared in `Param()`.
Pass values through a JSON string in `--param`.

Example:

```bash
ztrade backtest --script demo.so --param '{"amount":1,"fast":7,"slow":30}' --start "2020-01-01 08:00:00" --end "2021-01-01 08:00:00" --exchange binance --symbol BTCUSDT --markdown backtest.md
```

## 5. Multi-timeframe usage

ztrade uses `1m` as the base stream.
Higher timeframes should be merged in `Init`.

Example:

```go
func (d *Demo) Init(engine Engine, params ParamData) (err error) {
    d.engine = engine
    engine.Merge("1m", "15m", d.OnCandle15m)
    return nil
}

func (d *Demo) OnCandle15m(candle *Candle) {
    // signal logic
}
```

Recommended pattern:

- Use merged timeframe callbacks for main entry and exit signals.
- Keep `OnCandle` lightweight unless 1m logic is required.

## 6. Trading safety

Always follow these rules:

- Before opening long, handle existing short position first.
- Before opening short, handle existing long position first.
- Use `CancelAllOrder()` before switching direction when appropriate.
- Do not manually treat local state as final position; trust `OnPosition`.
- If strategy has slippage, use price helper functions.

## 7. Plugin workflow

Build plugin:

```bash
ztrade build --script /path/to/demo.go --output demo.so
```

With local/private dependencies:

```bash
ztrade build --script /path/to/demo.go --output demo.so --moduleRoot /path/to/deps-module
```

Ignore source module auto-discovery:

```bash
ztrade build --script /path/to/demo.go --output demo.so --ignoreSourceModuleRoot
```

Backtest / trade:

```bash
ztrade backtest --script demo.so --start "2020-01-01 08:00:00" --end "2021-01-01 08:00:00" --exchange binance --symbol BTCUSDT --markdown backtest.md
ztrade trade --script demo.so --exchange binance --symbol BTCUSDT
```

Backtest report options:

- `--report` / `-o`: output HTML report path, default `report.html`
- `--markdown`: output Markdown report path instead of HTML
- `--lang`: report language, `en` or `zh`
- `--console`: print the JSON result to stdout

Example:

```bash
ztrade backtest --script demo.so --start "2020-01-01 08:00:00" --end "2021-01-01 08:00:00" --exchange binance --symbol BTCUSDT --markdown backtest.md --lang zh
```
