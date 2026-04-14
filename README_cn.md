# ztrade
希望ztrade能成为 "你的最后一个交易系统"！

[English](README.md)

# Features

1. 使用go语言来开发/运行策略，不需要其他脚本语言
2. 基于事件模型，方便扩展
3. 支持币安,okx,ctp
4. 使用[ixgo](https://github.com/goplus/ixgo)作为脚本引擎
5. 可以将策略编译为go plugin,执行效率高

# 编译

``` shell
make
```

## 运行
``` shell
cd dist
./ztrade --help
```

# 使用
## 在配置文件中填写你的secret，key

## 下载K线历史

``` shell
# 首次运行
./ztrade download --binSize 1m --start "2020-01-01 08:00:00" --end "2021-01-01 08:00:00" --exchange binance --symbol BTCUSDT
# 自动下载K线
./ztrade download --symbol BTCUSDT -a --exchange binance
```

## 编译策略（默认流程）

默认编译出的 `ztrade` 二进制不带 `ixgo` 引擎，所以需要先把 `.go` 策略编译为插件（如 `.so`），再进行回测/实盘执行。

``` shell
./ztrade build --script /path/to/debug.go --output debug.so
```

## 回测

默认模式（不带 `ixgo`）：
- 先 `build` 生成插件（`.so`）
- `backtest/trade` 使用插件文件

``` shell
./ztrade backtest --script debug.so --start "2020-01-01 08:00:00" --end "2021-01-01 08:00:00" --symbol BTCUSDT --exchange binance
```

## 实盘

`ixgo` 模式（编译 ztrade 时带 `-tags ixgo`）：
- 可以直接使用 `.go` 策略文件回测/执行（无需先编译 `.so`）
- 但 ixgo 引擎本身有一些限制，参考官方文档：<https://github.com/goplus/ixgo>

`ixgo` 模式示例：

``` shell
./ztrade backtest --script /path/to/debug.go --start "2020-01-01 08:00:00" --end "2021-01-01 08:00:00" --symbol BTCUSDT --exchange binance
./ztrade trade --symbol BTCUSDT --exchange binance --script /path/to/debug.go
```

默认模式（插件）实盘示例：

``` shell
./ztrade trade --symbol BTCUSDT --exchange binance --script debug.so
```


## 策略

策略文档:
- English: [doc/strategy.md](doc/strategy.md)
- 中文: [doc/strategy_cn.md](doc/strategy_cn.md)

参考例子：
[strategy](https://github.com/ztrade/strategy)

## 策略教程

- English: [doc/strategy_tutorial.md](doc/strategy_tutorial.md)
- 中文: [doc/strategy_tutorial_cn.md](doc/strategy_tutorial_cn.md)

## 编译策略插件（build）

`build` 命令用于把策略源码（`.go`）编译成插件文件（Linux 通常是 `.so`），便于在默认（不带 `ixgo`）的 ztrade 中回测和实盘。

基础示例：

``` shell
./ztrade build --script /path/to/demo.go --output demo.so
```

编译完成后，可直接用于回测/实盘：

``` shell
./ztrade backtest --script demo.so --start "2020-01-01 08:00:00" --end "2021-01-01 08:00:00" --symbol BTCUSDT --exchange binance
./ztrade trade --script demo.so --symbol BTCUSDT --exchange binance
```

带私有依赖示例：

``` shell
./ztrade build --script /path/to/demo.go --output demo.so --moduleRoot /path/to/deps-module
```

默认情况下，`ztrade build` 会从策略源码目录开始向上查找最近的 `go.mod`。如果你希望忽略源码目录及父目录中的 `go.mod`，可以添加 `--ignoreSourceModuleRoot`。

## 鸣谢

[ixgo](https://github.com/goplus/ixgo)

[vnpy](https://github.com/vnpy/vnpy)
