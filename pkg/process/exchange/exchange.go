package exchange

import (
	"reflect"
	"sync/atomic"
	"time"

	. "github.com/SuperGod/trademodel"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/ztrade/base/common"
	. "github.com/ztrade/ztrade/pkg/define"
	. "github.com/ztrade/ztrade/pkg/event"
)

type OrderInfo struct {
	Order
	Action TradeType
	Filled bool
}

type TradeExchange struct {
	BaseProcesser
	impl Exchange

	datas   *ExchangeChan
	actChan chan TradeAction

	orders map[string]*OrderInfo

	closeCh chan bool

	positionUpdate int64
}

func NewTradeExchange(impl Exchange) *TradeExchange {
	te := new(TradeExchange)
	te.impl = impl
	te.datas = impl.GetDataChan()
	te.actChan = make(chan TradeAction, 10)
	te.orders = make(map[string]*OrderInfo)
	te.closeCh = make(chan bool)
	return te
}

func (b *TradeExchange) Init(bus *Bus) (err error) {
	b.BaseProcesser.Init(bus)
	bus.Subscribe(EventCandleParam, b.onEventCandleParam)
	bus.Subscribe(EventOrder, b.onEventOrder)
	bus.Subscribe(EventOrderCancelAll, b.onEventOrderCancelAll)
	bus.Subscribe(EventWatch, b.onEventWatch)
	return
}

func (b *TradeExchange) Start() (err error) {
	go b.recvDatas()
	go b.orderRoutine()
	return
}

func (b *TradeExchange) recvDatas() {
	var ok bool
	var balance Balance
	var depth Depth
	var order Order
	var pos Position
	var trade Trade
	var posTime int64
	var o *OrderInfo
Out:
	for {
		select {
		case balance, ok = <-b.datas.BalanceChan:
			if !ok {
				return
			}
			b.Send("balance", EventBalance, balance)
		case depth, ok = <-b.datas.DepthChan:
			if !ok {
				return
			}
			b.Send("depth", EventDepth, depth)
		case order, ok = <-b.datas.OrderChan:
			if !ok {
				return
			}
			o, ok = b.orders[order.OrderID]
			if !ok || o.Filled {
				continue Out
			}
			o.Order = order
			if order.Status != OrderStatusFilled {
				continue Out
			}
			o.Filled = true
			tr := Trade{ID: o.OrderID,
				Action: o.Action,
				Time:   o.Time,
				Price:  o.Price,
				Amount: o.Amount,
				Side:   o.Side,
				Remark: ""}
			b.Send(o.OrderID, EventTrade, tr)
		case pos, ok = <-b.datas.PosChan:
			if !ok {
				return
			}
			posTime = time.Now().Unix()
			atomic.StoreInt64(&b.positionUpdate, posTime)
			b.Send(pos.Symbol, EventPosition, pos)
		case trade, ok = <-b.datas.TradeChan:
			if !ok {
				return
			}
			b.Send("trade_history", EventTradeHistory, trade)
		}
	}
}

func (b *TradeExchange) onEventCandleParam(e Event) (err error) {
	var cParam CandleParam
	err = mapstructure.Decode(e.GetData(), &cParam)
	if err != nil {
	}
	go b.emitCandles(cParam)
	return
}

func (b *TradeExchange) onEventOrder(e Event) (err error) {
	var act TradeAction
	err = mapstructure.Decode(e.GetData(), &act)
	if err != nil {
		return
	}

	b.actChan <- act
	return
}
func (b *TradeExchange) onEventOrderCancelAll(e Event) (err error) {
	b.cancelAllOrder()
	return
}

func (b *TradeExchange) onEventWatch(e Event) (err error) {
	var param WatchParam
	err = mapstructure.Decode(e.GetData(), &param)
	if err != nil {
		return
	}
	err = b.impl.Watch(param)
	return
}

// orderRoutine process order routine
func (b *TradeExchange) orderRoutine() {
	var err error
	var ret interface{}
	for v := range b.actChan {
		ret, err = doOrderWithRetry(10, func() (interface{}, error) {
			order, e := b.impl.ProcessOrder(v)
			return order, e
		})
		if err == nil {
			od := ret.(*Order)
			b.orders[od.OrderID] = &OrderInfo{Order: *od, Action: v.Action}
		}

	}
}

func (b *TradeExchange) cancelAllOrder() {
	ret, err := doOrderWithRetry(10, func() (interface{}, error) {
		orders, err := b.impl.CancelAllOrders()
		return orders, err
	})
	if err != nil {
		log.Errorf("cancel allorder error %s", err.Error())
		return
	}
	log.Info("cancel order:", ret)
}

func (b *TradeExchange) emitCandles(param CandleParam) {
	if param.BinSize != "1m" {
		log.Info("BitmexTrade emit candle binsize not 1m:", param)
		return
	}
	// emit recent cancles
	nRecent := 60 * 24 * 3
	tLast, err := b.emitRecentCandles(param, nRecent)
	if err != nil {
		log.Errorf("downCandles emitRecentCandles failed:", err.Error())
		return
	}

	symbolInfo := SymbolInfo{Exchange: "bitmex", Symbol: param.Symbol, Resolutions: param.BinSize}
	datas, _, err := b.impl.WatchKline(symbolInfo)
	if err != nil {
		log.Errorf("emitCandles wathKline failed:", err.Error())
		return
	}
	// log.Infof("emitCandles wathKline :%##v", symbols)
	for v := range datas {
		candle := v.Data.(*Candle)
		if candle == nil {
			log.Error("emitCandles data type error:", reflect.TypeOf(v.Data))
			continue
		}
		if candle.Start == tLast {
			continue
		}
		b.Send(NewCandleName("candle", param.BinSize).String(), EventCandle, candle)
		tLast = candle.Start
	}
	if b.closeCh != nil {
		b.closeCh <- true
	}

	if err != nil {
		log.Error("exchange emitCandle error:", err.Error())
	}
}

func (b *TradeExchange) emitRecentCandles(param CandleParam, recent int) (tLast int64, err error) {
	dur, err := common.GetBinSizeDuration(param.BinSize)
	if err != nil {
		log.Errorf("downCandles GetBinSizeDuration failed:", param.BinSize, err.Error())
		return
	}
	tEnd := time.Now()
	tStart := tEnd.Add(0 - (time.Duration(recent) * dur))
	klines, errChan := b.impl.KlineChan(tStart, tEnd, param.Symbol, param.BinSize)
	for v := range klines {
		tLast = v.Start
		b.Send(NewCandleName("recent", param.BinSize).String(), EventCandle, v)
	}
	err = <-errChan
	return
}
