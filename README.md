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
Just copy pkg/helper/helper.go to your own strategy dir,and then you can develop it as you would normally write go code

[strategy.go](pkg/helper/strategy.go)

```
type DemoStrategy struct {
}

func NewDemoStrategy() *DemoStrategy {
	return new(DemoStrategy)
}

// Param define you script params here
func (s *DemoStrategy) Param() (paramInfo []Param) {
	paramInfo = []Param{
		Param{Name: "symbol", Type: "string", Info: "symbol code"},
	}
	return
}

// Init strategy
func (s *DemoStrategy) Init(engine *Engine, params ParamData) {
	return
}

// OnCandle call when 1m candle reached
func (s *DemoStrategy) OnCandle(candle *Candle) {
	var param Param
	param.Name = "hello"
	fmt.Println("candle:", candle, param)
	return
}

// OnPosition call when position is updated
func (s *DemoStrategy) OnPosition(pos float64) {
	fmt.Println("position:", pos)
	return
}

// OnTradeMarket call every trade occurs
func (s *DemoStrategy) OnTradeMarket(trade Trade) {
	fmt.Println("trade:", trade)
	return
}

// OnTrade call when you own trade occures
func (s *DemoStrategy) OnTrade(trade *Trade) {
	fmt.Println("tradeHistory:", trade)
	return
}

// OnDepth call when orderbook updated
func (s *DemoStrategy) OnDepth(depth *Depth) {
	fmt.Println("depth:", depth)
	return
}

```

## Thanks

[gomacro](https://github.com/cosmos72/gomacro)

[vnpy](https://github.com/vnpy/vnpy)
