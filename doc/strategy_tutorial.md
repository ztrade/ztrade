# ztrade Strategy Tutorial (English)

This guide shows how to write and run strategies in **ztrade**, based on the patterns used in the existing strategies (e.g. EMA cross, Bollinger-based strategies).

> Key idea: ztrade is event-driven. The framework always provides **1m** candles as the base stream; you build higher timeframes (5m/15m/1h/…) inside your strategy via `engine.Merge()`.

---

## 1. Two ways to run a strategy

### A. Compile as a Go plugin (recommended; works with default builds)

- Use `ztrade build` to compile your strategy source file into a plugin: `.so` (Linux), `.dylib` (macOS), `.dll` (Windows).
- Use `--script` to point to that plugin in backtest/live trade.
- Best performance and works with the default ztrade build.

### B. Run `.go` source directly (requires `ixgo` build tag)

- You must build ztrade with `-tags ixgo` to run `--script your.go` directly.
- Faster iteration, but requires a special build.

---

## 2. Strategy lifecycle & required callbacks

A strategy is a Go `struct` that implements these methods (implement all of them; empty bodies are fine):

- `Param() []Param` – declare CLI parameters
- `Init(engine Engine, params ParamData) error` – initialize, capture `engine`, register merges
- `OnCandle(candle *Candle)` – called for every **1m** candle
- `OnPosition(pos, price float64)` – called when position changes
- `OnTrade(trade *Trade)` – called when *your* order fills
- `OnTradeMarket(trade *Trade)` – called for market trades (exchange stream)
- `OnDepth(depth *Depth)` – called for order book updates

Important:
- ztrade subscribes to **1m** candles.
- Use `engine.Merge("1m", "15m", fn)` to synthesize larger timeframes.

---

## 3. Minimal working strategy example

Create `demo.go`:

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

    // params
    largeBin int
    amount   float64
    fast     int
    slow     int

    // indicator
    ema indicator.CommonIndicator
}

func NewDemo() *Demo { return new(Demo) }

func (d *Demo) Param() (paramInfo []Param) {
    return []Param{
        IntParam("bin", "Timeframe (min)", "Merged timeframe in minutes", 15, &d.largeBin),
        FloatParam("amount", "Amount", "Order size", 1, &d.amount),
        IntParam("fast", "EMA fast", "Fast EMA length", 7, &d.fast),
        IntParam("slow", "EMA slow", "Slow EMA length", 30, &d.slow),
    }
}

func (d *Demo) Init(engine Engine, params ParamData) (err error) {
    d.engine = engine
    d.ema = engine.AddIndicator("ema", d.fast, d.slow)

    large := fmt.Sprintf("%dm", d.largeBin)
    engine.Merge("1m", large, d.OnCandleLarge)

    return nil
}

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

func (d *Demo) OnPosition(pos, price float64) { d.position = pos }
func (d *Demo) OnTrade(trade *Trade)       {}
func (d *Demo) OnTradeMarket(trade *Trade) {}
func (d *Demo) OnDepth(depth *Depth)       {}
```

This mirrors the common style in the existing strategies:
- register `Merge` in `Init`
- update indicators + trading logic in the merged-candle callback
- keep position in sync via `OnPosition`

---

## 4. Parameters: `Param()` + `--param`

- `Param()` returns a list of parameter definitions.
- `--param` is a **JSON string** where keys match parameter names.

Example:

```bash
./ztrade backtest \
  --script demo.so \
  --param '{"bin":15,"amount":1,"fast":7,"slow":30}' \
  --start "2020-01-01 08:00:00" --end "2021-01-01 08:00:00" \
  --exchange binance --symbol BTCUSDT
```

---

## 5. Engine API quick reference

Commonly used methods:

- Orders: `OpenLong`, `CloseLong`, `OpenShort`, `CloseShort`
- Stops: `StopLong`, `StopShort`
- Cancel: `CancelOrder`, `CancelAllOrder`
- State: `Position()`, `Balance()`
- Indicators: `AddIndicator(name, ...params)`
- Timeframes: `Merge(src, dst, fn)`
- Logging/notify: `Log(...)`, `SendNotify(...)`

Indicator tips:
- call `Update(price)` (usually `candle.Close`)
- use `Indicator()` map fields like `crossUp`/`crossDown` for cross indicators

---

## 6. Build & run

### 6.1 Build as plugin (recommended)

```bash
./ztrade build --script /path/to/demo.go --output demo.so
```

Backtest with plugin:

```bash
./ztrade backtest --script demo.so ...
```

Live trading:

```bash
./ztrade trade --script demo.so ...
```

### 6.2 Run `.go` directly (requires `ixgo` build)

Build ztrade with `ixgo`:

```bash
CGO_ENABLED=0 go build -ldflags="-checklinkname=0" -tags ixgo -o dist/ztrade ./
```

Then:

```bash
./ztrade backtest --script /path/to/demo.go ...
```

---

## 7. Common pitfalls

1. **Make `Init` return `error`**: the runner interface is `Init(...) error`.
2. **Base candles are always `1m`**: larger timeframes require `Merge`.
3. **Trust `OnPosition`**: don’t infer position only from your own orders.
4. **Provide `NewXxx()`**: both plugin and ixgo modes rely on `New{Struct}` constructors.
