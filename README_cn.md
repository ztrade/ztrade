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
复制 pkg/helper/helper.go 到你自己的策略目录,然后就可以使用go语言方便的开发策略了


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

## 鸣谢

[gomacro](https://github.com/cosmos72/gomacro)

[vnpy](https://github.com/vnpy/vnpy)
