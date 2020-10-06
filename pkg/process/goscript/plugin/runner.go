package plugin

import (
	. "github.com/SuperGod/trademodel"
	"github.com/ztrade/ztrade/pkg/common"

	// 	. "github.com/ztrade/ztrade/pkg/event"
	"github.com/ztrade/ztrade/pkg/process/goscript/engine"
)

type Runner interface {
	Param() (paramInfo []common.Param)
	Init(engine *engine.Engine, params common.ParamData)
	OnCandle(candle Candle)
	OnPosition(pos, price float64)
	OnTrade(trade Trade)
	OnTradeHistory(trade Trade)
	OnDepth(depth Depth)
	// OnEvent(e Event)
}
