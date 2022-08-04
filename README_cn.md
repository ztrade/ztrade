# ztrade
希望ztrade能成为 "你的最后一个交易系统"！

# Features

1. 使用go语言来开发/运行策略，不需要其他脚本语言
2. 基于事件模型，方便扩展
3. 支持币安,okx,ctp
4. 使用[gomacro](https://github.com/cosmos72/gomacro)作为脚本引擎
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

## 回测

``` shell
./ztrade backtest --script debug.go --start "2020-01-01 08:00:00" --end "2021-01-01 08:00:00" --symbol BTCUSDT --exchange binance
```

## 实盘

``` shell
./ztrade trade --symbol BTCUSDT --exchange binance --script debug.go
```


## 策略

策略文档:
[策略](./doc/strategy.md)

参考例子：
[strategy](https://github.com/ztrade/strategy)

## 鸣谢

[gomacro](https://github.com/cosmos72/gomacro)

[vnpy](https://github.com/vnpy/vnpy)
