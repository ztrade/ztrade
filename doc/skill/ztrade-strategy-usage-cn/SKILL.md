---
name: ztrade-strategy-usage-cn
description: "适用场景：用中文编写或重构 ztrade 策略、生成 ztrade 插件模式的 build/backtest/trade 命令、说明已安装到 PATH 中的 ztrade 程序如何使用。"
---

# ZTrade 策略使用 Skill

## 目标

帮助 agent 以插件模式使用已经可运行的 `ztrade` 程序：

- 编写或重构 ztrade 策略
- 生成正确的 `ztrade build`、`ztrade backtest`、`ztrade trade` 命令
- 默认让回测输出 Markdown 报告

## 范围

- 假设 `ztrade` 已安装到系统 `PATH`。
- 命令直接使用 `ztrade`，不要写成 `./ztrade`。
- 默认使用插件模式。
- 除非用户明确要求，否则不要生成 ixgo 模式命令。

## 参考文档

需要示例和详细用法时，打开：

- `references/strategy_guide.md`

## 策略契约

生成的策略应实现：

- `Param() []Param`
- `Init(engine Engine, params ParamData) error`
- `OnCandle(candle *Candle)`
- `OnPosition(pos, price float64)`
- `OnTrade(trade *Trade)`
- `OnTradeMarket(trade *Trade)`
- `OnDepth(depth *Depth)`

构造函数使用 `NewXxx() *Xxx` 命名。

## 默认命令

编译插件：

```bash
ztrade build --script /path/to/demo.go --output demo.so
```

回测：

```bash
ztrade backtest --script demo.so --start "2020-01-01 08:00:00" --end "2021-01-01 08:00:00" --exchange binance --symbol BTCUSDT --markdown backtest.md
```

实盘：

```bash
ztrade trade --script demo.so --exchange binance --symbol BTCUSDT
```

## 输出规则

当需要生成策略时：

1. 给出完整 `.go` 策略文件。
2. 说明参数含义。
3. 给出一条 `ztrade build` 命令。
4. 至少给出一条带 `--markdown` 的 `ztrade backtest` 命令。
5. 给出一条 `ztrade trade` 命令模板。

当需要说明用法时：

1. 先说明当前使用插件模式。
2. 使用可直接执行的 `ztrade` 命令。
3. 回测示例默认带 `--markdown`。
4. 回答聚焦在程序用法，不展开仓库内部实现。
