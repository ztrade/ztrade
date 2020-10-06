package plugin

import (
	. "github.com/SuperGod/trademodel"
	"github.com/ztrade/base/common"
	"github.com/ztrade/base/engine"
	// 	. "github.com/ztrade/ztrade/pkg/event"
)

type Runner interface {
	Param() (paramInfo []common.Param)
	Init(engine engine.Engine, params common.ParamData)
	OnCandle(candle Candle)
	OnPosition(pos, price float64)
	OnTrade(trade Trade)
	OnTradeHistory(trade Trade)
	OnDepth(depth Depth)
	// OnEvent(e Event)
}
