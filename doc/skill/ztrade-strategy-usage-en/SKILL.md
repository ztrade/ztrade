---
name: ztrade-strategy-usage-en
description: "Use when: write or refactor ztrade strategies in English, generate ztrade build/backtest/trade commands in plugin mode, or explain how to use a ztrade binary installed in PATH."
---

# ZTrade Strategy Usage Skill

## Goal

Help an agent use a ready-to-run `ztrade` program in plugin mode:

- write or refactor ztrade strategies
- generate correct `ztrade build`, `ztrade backtest`, and `ztrade trade` commands
- default backtest output to Markdown reports

## Scope

- Assume `ztrade` is installed in the system `PATH`.
- Use `ztrade`, not `./ztrade`.
- Use plugin mode by default.
- Do not generate ixgo-mode commands unless the user explicitly asks for them.

## Reference

Open this file for examples and detailed usage:

- `references/strategy_guide.md`

## Required Strategy Contract

Generated strategies should implement:

- `Param() []Param`
- `Init(engine Engine, params ParamData) error`
- `OnCandle(candle *Candle)`
- `OnPosition(pos, price float64)`
- `OnTrade(trade *Trade)`
- `OnTradeMarket(trade *Trade)`
- `OnDepth(depth *Depth)`

Use `NewXxx() *Xxx` constructor naming.

## Default Commands

Build plugin:

```bash
ztrade build --script /path/to/demo.go --output demo.so
```

Backtest:

```bash
ztrade backtest --script demo.so --start "2020-01-01 08:00:00" --end "2021-01-01 08:00:00" --exchange binance --symbol BTCUSDT --markdown backtest.md
```

Trade:

```bash
ztrade trade --script demo.so --exchange binance --symbol BTCUSDT
```

## Response Rules

When generating a strategy:

1. Provide the full `.go` strategy file.
2. Explain the parameters.
3. Provide one `ztrade build` command.
4. Provide at least one `ztrade backtest` command using `--markdown`.
5. Provide one `ztrade trade` command template.

When answering usage questions:

1. State that plugin mode is being used.
2. Use executable commands with `ztrade`.
3. Backtest examples should default to Markdown output via `--markdown`.
4. Keep the answer focused on using the program, not repository internals.
