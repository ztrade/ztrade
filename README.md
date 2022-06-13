# ztrade
I hope ztrade is "The last trade app you need" !

[中文](README_cn.md)

# Features

1. Develop/write strategy with only go language,no other script need
2. Event base framework,easy to extend
3. Support binance,okx,ctp
4. use [gomacro](https://github.com/cosmos72/gomacro) as script engine
5. can build strategy to go golang plugin,best performance

# build

``` shell
make
```

## simple run
``` shell
cd dist
./ztrade --help
```

# Use
## replace your key and secret
replace your key and secret in dist/configs/ztrade.yaml

## download history Kline

``` shell
# run first
./ztrade download --binSize 1m --start "2020-01-01 08:00:00" --end "2021-01-01 08:00:00" --exchange binance --symbol BTCUSDT
# auto download kline
./ztrade download --symbol BTCUSDT -a --exchange binance
```

## backtest

``` shell
./ztrade backtest --script debug.go --start "2020-01-01 08:00:00" --end "2021-01-01 08:00:00" --symbol BTCUSDT --exchange binance
```

## real trade

``` shell
./ztrade trade --symbol BTCUSDT --exchange binance --script debug.go
```


## strategy
show examples:

[strategy](https://github.com/ztrade/strategy)


## Thanks

[gomacro](https://github.com/cosmos72/gomacro)

[vnpy](https://github.com/vnpy/vnpy)
