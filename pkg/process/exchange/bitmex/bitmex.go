package bitmex

import (
	"fmt"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/ztrade/base/common"
	. "github.com/ztrade/ztrade/pkg/define"
	. "github.com/ztrade/ztrade/pkg/event"

	"github.com/SuperGod/coinex"
	"github.com/SuperGod/coinex/bitmex"
	. "github.com/SuperGod/trademodel"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	defaultBinSizes = map[string]bool{"1m": true, "5m": true, "1h": true, "1d": true}
)

type OrderInfo struct {
	Order
	Action TradeType
	Filled bool
}

type BitmexTrade struct {
	BaseProcesser
	bm             *bitmex.Bitmex
	symbol         string
	posChan        chan []coinex.Position
	orderChan      chan []Order
	orders         map[string]*OrderInfo
	actChan        chan TradeAction
	position       float64
	positionUpdate int64
	watch          bool
	closeCh        chan bool
	lastKlines     map[string]time.Time
	depthChan      chan Depth
	tradeChan      chan Trade
	balanceChan    chan Balance
}

func NewBitmexTradeWithSymbol(cfg *viper.Viper, cltName, symbol string) (b *BitmexTrade, err error) {
	b = new(BitmexTrade)
	b.Name = "bitmex"
	if cltName == "" {
		err = fmt.Errorf("must input bitmex client name")
		return
	}
	isDebug := cfg.GetBool(fmt.Sprintf("exchanges.%s.debug", cltName))
	b.bm = bitmex.GetClientByViperName(cfg, cltName, isDebug)
	b.symbol = symbol
	b.bm.SetSymbol(symbol)

	b.orders = make(map[string]*OrderInfo)
	b.posChan = make(chan []coinex.Position, 10)
	b.orderChan = make(chan []Order, 10)
	b.actChan = make(chan TradeAction, 10)
	b.depthChan = make(chan Depth, 10)
	b.tradeChan = make(chan Trade, 10)
	b.balanceChan = make(chan Balance, 10)
	b.bm.WS().SetPositionChan(b.posChan)
	b.bm.WS().SetOrderChan(b.orderChan)
	b.bm.WS().SetDepthChan(b.depthChan)
	b.bm.WS().SetTradeChan(b.tradeChan)
	b.bm.WS().SetBalanceChan(b.balanceChan)
	b.lastKlines = make(map[string]time.Time)
	return
}

func NewBitmexTrade(cfg *viper.Viper, cltName string) (b *BitmexTrade, err error) {
	return NewBitmexTradeWithSymbol(cfg, cltName, "XBTUSD")
}

func (b *BitmexTrade) Init(bus *Bus) (err error) {
	b.BaseProcesser.Init(bus)
	bus.Subscribe(EventCandleParam, b.onEventCandleParam)
	bus.Subscribe(EventOrder, b.onEventOrder)
	bus.Subscribe(EventOrderCancelAll, b.onEventOrderCancelAll)
	bus.Subscribe(EventWatch, b.onEventWatch)
	return
}

func (b *BitmexTrade) startWS() (err error) {
	contracts, err := b.bm.Contracts()
	if err != nil {
		log.Error("get contracts error:", err.Error())
		return
	}
	var subs []bitmex.SubscribeInfo
	subCurrencys := make(map[string]bool)
	var ok bool
	// subscribe all position and orders
	for _, v := range contracts {
		_, ok = subCurrencys[v.Symbol]
		if ok {
			continue
		}
		subCurrencys[v.Symbol] = true
		subs = append(subs, bitmex.SubscribeInfo{Op: bitmex.BitmexWSPosition, Param: v.Symbol})
		subs = append(subs, bitmex.SubscribeInfo{Op: bitmex.BitmexWSOrder, Param: v.Symbol})
	}
	subs = append(subs, bitmex.SubscribeInfo{Op: bitmex.BitmexWSMargin})
	b.bm.WS().SetSubscribe(subs)
	err = b.bm.StartWS()
	if err != nil {
		return
	}
	go b.handleData()
	go b.orderRoutine()
	go b.checkPos()
	return
}

func (b *BitmexTrade) checkPos() {
	t := time.NewTicker(time.Second * 30)
	var posTime, nDur int64
	for {
		select {
		case _ = <-t.C:
			posTime = atomic.LoadInt64(&b.positionUpdate)
			nDur = time.Now().Unix() - posTime
			if nDur > 30 {
				pos, err := b.bm.Positions()
				if err != nil {
					log.Error("bitmextrade checkPos get position failed:", err.Error())
					continue
				}
				for _, v := range pos {
					if v.Info.Symbol == b.symbol {
						posTime = time.Now().Unix()
						atomic.StoreInt64(&b.positionUpdate, posTime)
						b.Send(v.Info.Symbol, EventPosition, Position{
							Symbol:      b.symbol,
							Hold:        v.Hold,
							Price:       v.Price,
							ProfitRatio: v.ProfitRatio,
						})
						return
					}
				}
				// send zero position to clear position
				posEmpty := Position{
					Symbol:      b.symbol,
					Hold:        0,
					Price:       0,
					ProfitRatio: 0,
				}
				b.Send(b.symbol, EventPosition, posEmpty)
			}
		}
	}
}

func (b *BitmexTrade) handleData() {
	var o *OrderInfo
	var ok bool
	var posTime int64
	for {
		select {
		case ba := <-b.balanceChan:
			b.Send("balance", EventBalance, ba)
		case pos := <-b.posChan:
			for _, v := range pos {
				if v.Info.Symbol == b.symbol {
					posTime = time.Now().Unix()
					atomic.StoreInt64(&b.positionUpdate, posTime)

					b.Send(v.Info.Symbol, EventPosition, v)
				}
			}
		case order := <-b.orderChan:
			for _, v := range order {
				o, ok = b.orders[v.OrderID]
				if !ok || o.Filled {
					return
				}
				o.Order = v
				if !bitmex.IsOrderFilled(&v) {
					continue
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

			}
		case depth := <-b.depthChan:
			b.Send("depth", EventDepth, depth)
		case trade := <-b.tradeChan:
			b.Send("trade_history", EventTradeHistory, trade)
		}
	}
}

func (b *BitmexTrade) SetCloseChan(closeCh chan bool) {
	b.closeCh = closeCh
	return
}

func (b *BitmexTrade) Start() (err error) {
	err = b.startWS()
	return
}

func (b *BitmexTrade) onEventCandleParam(e Event) (err error) {
	var cParam CandleParam
	err = mapstructure.Decode(e.GetData(), &cParam)
	if err != nil {
	}
	go b.emitCandles(cParam)
	return
}

func (b *BitmexTrade) onEventOrder(e Event) (err error) {
	var act TradeAction
	err = mapstructure.Decode(e.GetData(), &act)
	if err != nil {
		return
	}
	b.actChan <- act
	return
}
func (b *BitmexTrade) onEventOrderCancelAll(e Event) (err error) {
	b.cancelAllOrder()
	return
}

func (b *BitmexTrade) onEventWatch(e Event) (err error) {
	var param WatchParam
	err = mapstructure.Decode(e.GetData(), &param)
	if err != nil {
		return
	}
	switch param.Type {
	case EventDepth:
		sub := bitmex.SubscribeInfo{Op: bitmex.BitmexWSOrderbookL2_25, Param: b.symbol}
		b.bm.WS().AddSubscribe(sub)
	case EventTradeHistory:
		sub := bitmex.SubscribeInfo{Op: bitmex.BitmexWSTrade, Param: b.symbol}
		b.bm.WS().AddSubscribe(sub)
	}
	return
}

func (b *BitmexTrade) cancelAllOrder() {
	ret, err := doOrderWithRetry(10, func() (interface{}, error) {
		orders, err := b.bm.CancelAllOrders()
		return orders, err
	})
	if err != nil {
		log.Errorf("cancel allorder error %s", err.Error())
		return
	}
	log.Info("cancel order:", ret)
}

// orderRoutine process order routine
func (b *BitmexTrade) orderRoutine() {
	var err error
	var ret interface{}
	for v := range b.actChan {
		ret, err = doOrderWithRetry(10, func() (interface{}, error) {
			order, e := b.processOrder(v)
			return order, e
		})
		if err == nil {
			od := ret.(*Order)
			b.orders[od.OrderID] = &OrderInfo{Order: *od, Action: v.Action}
		}

	}
}

func (b *BitmexTrade) processOrder(act TradeAction) (ret *Order, err error) {
	switch act.Action {
	case OpenLong:
		ret, err = b.bm.OpenLong(act.Price, act.Amount)
	case OpenShort:
		ret, err = b.bm.OpenShort(act.Price, act.Amount)
	case CloseLong:
		ret, err = b.bm.CloseLong(act.Price, act.Amount)

	case CloseShort:
		ret, err = b.bm.CloseShort(act.Price, act.Amount)

	case StopLong:
		ret, err = b.bm.StopLoseSellMarket(act.Price, act.Amount)
	case StopShort:
		ret, err = b.bm.StopLoseBuyMarket(act.Price, act.Amount)
	default:
		err = fmt.Errorf("unsupport order action:%s", act.Action)
	}
	if err != nil {
		log.Errorf("order action %##v error:%s", act, err.Error())
		return
	}
	return
}

func (b *BitmexTrade) emitRecentCandles(param CandleParam, recent int) (tLast int64, err error) {
	dur, err := common.GetBinSizeDuration(param.BinSize)
	if err != nil {
		log.Errorf("downCandles GetBinSizeDuration failed:", param.BinSize, err.Error())
		return
	}
	tEnd := time.Now()
	tStart := tEnd.Add(0 - (time.Duration(recent) * dur))
	klines, errChan := b.KlineChan(tStart, tEnd, param.Symbol, param.BinSize)
	for datas := range klines {
		for k := 0; k != len(datas); k++ {
			v, ok := datas[k].(*Candle)
			if !ok {
				log.Errorf("candles type error:%s", reflect.TypeOf(datas[k]))
				continue
			}
			tLast = v.Start
			b.Send(NewCandleName("recent", param.BinSize).String(), EventCandle, v)
		}
	}
	err = <-errChan
	return
}

func (b *BitmexTrade) emitCandles(param CandleParam) {
	if param.BinSize != "1m" {
		log.Info("BitmexTrade emit candle binsize not 1m:", param)
		return
	}
	nRecent := 60 * 24 * 3
	tLast, err := b.emitRecentCandles(param, nRecent)
	if err != nil {
		log.Errorf("downCandles emitRecentCandles failed:", err.Error())
		return
	}
	symbols := []SymbolInfo{SymbolInfo{Exchange: "bitmex", Symbol: param.Symbol, Resolutions: param.BinSize}}
	datas := make(chan *CandleInfo, 10)
	err = b.WatchKline(symbols, datas)
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
		log.Error("bitmex emitCandle error:", err.Error())
	}
}
