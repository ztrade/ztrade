# ztrade
I hope ztrade is "The last trade app you need" !

# Features

1. Develop/write strategy with only go language,no other script need
2. Event base framework,easy to extend
3. Support bitmex exchange,more exchanges comming soon
4. Because the high performance of go([gomacro](https://github.com/cosmos72/gomacro) is also very fast), maybe ztrade can be used in high frequency trade

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
./ztrade download --binSize 1m --start "2020-01-01 08:00:00" --end "2020-06-01 08:00:00"
```

## backtest

``` shell
./ztrade backtest --script debug.go --start "2020-01-01 08:00:00" --end "2020-06-01 08:00:00"
```

## real trade

``` shell
./ztrade trade --exchange bitmextest --script debug.go
```


## strategy
Just copy pkg/helper/helper.go to your own strategy dir,and then you can develop it as you would normally write go code

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
func (s *DemoStrategy) OnCandle(candle Candle) {
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

// OnTrade call call every trade occurs
func (s *DemoStrategy) OnTrade(trade Trade) {
	fmt.Println("trade:", trade)
	return
}

// OnTradeHistory call when you own trade occures
func (s *DemoStrategy) OnTradeHistory(trade Trade) {
	fmt.Println("tradeHistory:", trade)
	return
}

// OnDepth call when orderbook updated
func (s *DemoStrategy) OnDepth(depth Depth) {
	fmt.Println("depth:", depth)
	return
}

```

## Thanks

[gomacro](https://github.com/cosmos72/gomacro)

[vnpy](https://github.com/vnpy/vnpy)