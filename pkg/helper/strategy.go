package helper

import (
	"fmt"

	. "github.com/ztrade/trademodel"
)

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
func (s *DemoStrategy) Init(engine Engine, params ParamData) {
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
func (s *DemoStrategy) OnPosition(pos, price float64) {
	fmt.Println("position:", pos, price)
	return
}

// OnTrade call call you own trade occures
func (s *DemoStrategy) OnTrade(trade *Trade) {
	fmt.Println("trade:", trade)
	return
}

// OnTradeMarket call when trade occures
func (s *DemoStrategy) OnTradeMarket(trade *Trade) {
	fmt.Println("tradeHistory:", trade)
	return
}

// OnDepth call when orderbook updated
func (s *DemoStrategy) OnDepth(depth *Depth) {
	fmt.Println("depth:", depth)
	return
}
