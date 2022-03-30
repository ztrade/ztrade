package exchange

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/ztrade/trademodel"
	. "github.com/ztrade/trademodel"
	. "github.com/ztrade/ztrade/pkg/core"
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

	datas   chan *ExchangeData
	actChan chan TradeAction

	orders map[string]*OrderInfo

	closeCh chan bool

	positionUpdate int64
	exchangeName   string
	symbol         string

	candleParam CandleParam
}

func NewTradeExchange(exName string, impl Exchange, symbol string) *TradeExchange {
	te := new(TradeExchange)
	te.exchangeName = exName
	te.impl = impl
	te.datas = impl.GetDataChan()
	te.actChan = make(chan TradeAction, 10)
	te.orders = make(map[string]*OrderInfo)
	te.closeCh = make(chan bool)
	te.symbol = symbol
	return te
}

func (b *TradeExchange) Init(bus *Bus) (err error) {
	b.BaseProcesser.Init(bus)
	b.Subscribe(EventOrder, b.onEventOrder)
	b.Subscribe(EventWatch, b.onEventWatch)
	return
}

func (b *TradeExchange) Start() (err error) {
	go b.recvDatas()
	go b.orderRoutine()
	return
}

func (b *TradeExchange) recvDatas() {
	var ok bool
	var balance *Balance
	var depth *Depth
	var order *Order
	var pos *Position
	var trade *Trade
	var posTime int64
	var o *OrderInfo
	var candle *Candle
	bFirst := true
	var err error
	var tFirstLastStart int64
Out:
	for data := range b.datas {
		switch data.GetType() {
		case EventCandle:
			candle = data.GetData().(*Candle)
			if bFirst {
				bFirst = false
				param := b.candleParam
				param.End = candle.Time().Add(-1 * time.Second)
				tFirstLastStart, err = b.emitRecentCandles(param)
				if err != nil {
					log.Errorf("TradeExchange recv data:", err.Error())
					panic(err.Error())
				}
				if candle.Start <= tFirstLastStart {
					continue
				}
			}
			b.Send(data.Name, data.GetType(), candle)
		case EventBalance:
			balance = data.GetData().(*Balance)
			b.Send(b.exchangeName, EventBalance, balance)
		case EventDepth:
			depth = data.GetData().(*Depth)
			b.Send(b.exchangeName, EventDepth, depth)
		case EventOrder:
			order = data.GetData().(*Order)
			o, ok = b.orders[order.OrderID]
			if !ok || o.Filled {
				continue Out
			}
			o.Order = *order
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
			b.Send(o.OrderID, EventTrade, &tr)
		case EventPosition:
			pos = data.GetData().(*Position)
			posTime = time.Now().Unix()
			atomic.StoreInt64(&b.positionUpdate, posTime)
			b.Send(pos.Symbol, EventPosition, pos)
		case EventTradeMarket:
			trade = data.GetData().(*Trade)
			b.Send(b.exchangeName, EventTradeMarket, trade)
		}
	}
}

func (b *TradeExchange) onEventCandleParam(e Event) (err error) {
	wParam, ok := e.GetData().(*WatchParam)
	if !ok {
		err = fmt.Errorf("event not watch %s %#v", e.Name, e.Data)
		return
	}
	cParam, _ := wParam.Data.(*CandleParam)
	if cParam == nil {
		err = fmt.Errorf("event not CandleParam %s %#v", e.Name, e.Data)
		return
	}
	go b.emitCandles(*cParam)
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

func (b *TradeExchange) onEventWatch(e Event) (err error) {
	if e.Name == "candle" {
		return b.onEventCandleParam(e)
	}
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
		if v.Action == trademodel.CancelAll {
			b.cancelAllOrder()
			continue
		}
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
		log.Info("TradeExchange emit candle binsize not 1m:", param)
		return
	}
	watchParam := WatchParam{Type: EventWatchCandle, Data: &param}
	b.candleParam = param
	err := b.impl.Watch(watchParam)
	if err != nil {
		log.Errorf("emitCandles wathKline failed:", err.Error())
		return
	}
}

func (b *TradeExchange) emitRecentCandles(param CandleParam) (tLast int64, err error) {
	klines, errChan := b.impl.GetKline(param.Symbol, param.BinSize, param.Start, param.End)
	for v := range klines {
		tLast = v.Start
		b.Send(FormatCandleName("recent", param.BinSize), EventCandle, v)
	}
	err = <-errChan
	return
}
