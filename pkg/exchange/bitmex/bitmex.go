package bitmex

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/SuperGod/coinex"
	"github.com/SuperGod/coinex/bitmex"
	. "github.com/SuperGod/trademodel"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	. "github.com/ztrade/ztrade/pkg/define"
	"github.com/ztrade/ztrade/pkg/process/exchange"
)

var _ exchange.Exchange = &BitmexTrade{}

func init() {
	exchange.RegisterExchange("bitmex", NewBitmexExchange)
}

type BitmexTrade struct {
	Name   string
	bm     *bitmex.Bitmex
	symbol string
	datas  *exchange.ExchangeChan

	posChan        chan []coinex.Position
	orderChan      chan []Order
	positionUpdate int64
	closeCh        chan bool
}

func NewBitmexExchange(cfg *viper.Viper, cltName, symbol string) (e exchange.Exchange, err error) {
	b, err := NewBitmexTradeWithSymbol(cfg, cltName, symbol)
	if err != nil {
		return
	}
	e = b
	return
}

func NewBitmexTradeWithSymbol(cfg *viper.Viper, cltName, symbol string) (b *BitmexTrade, err error) {
	b = new(BitmexTrade)
	b.Name = "bitmex"
	if cltName == "" {
		err = fmt.Errorf("must input bitmex client name")
		return
	}
	b.datas = exchange.NewExchangeChan()
	b.closeCh = make(chan bool)

	isDebug := cfg.GetBool(fmt.Sprintf("exchanges.%s.debug", cltName))
	b.bm = bitmex.GetClientByViperName(cfg, cltName, isDebug)
	b.symbol = symbol
	b.bm.SetSymbol(symbol)

	b.posChan = make(chan []coinex.Position, 10)
	b.orderChan = make(chan []Order, 10)
	b.bm.WS().SetPositionChan(b.posChan)
	b.bm.WS().SetOrderChan(b.orderChan)
	b.bm.WS().SetDepthChan(b.datas.DepthChan)
	b.bm.WS().SetTradeChan(b.datas.TradeChan)
	b.bm.WS().SetBalanceChan(b.datas.BalanceChan)
	return
}

func NewBitmexTrade(cfg *viper.Viper, cltName string) (b *BitmexTrade, err error) {
	return NewBitmexTradeWithSymbol(cfg, cltName, "XBTUSD")
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
						b.datas.PosChan <- Position{
							Symbol:      b.symbol,
							Hold:        v.Hold,
							Price:       v.Price,
							ProfitRatio: v.ProfitRatio,
						}
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
				b.datas.PosChan <- posEmpty
			}
		}
	}
}

func (b *BitmexTrade) handleData() {
	var posTime int64
	for {
		select {
		case pos := <-b.posChan:
			for _, v := range pos {
				if v.Info.Symbol == b.symbol {
					posTime = time.Now().Unix()
					atomic.StoreInt64(&b.positionUpdate, posTime)
					b.datas.PosChan <- Position{Symbol: v.Info.Symbol,
						Type:        v.Type,
						Hold:        v.Hold,
						Price:       v.Price,
						ProfitRatio: v.ProfitRatio,
					}
				}
			}
		case order := <-b.orderChan:
			for _, v := range order {
				b.datas.OrderChan <- v
			}
		}
	}
}

func (b *BitmexTrade) Start() (err error) {
	err = b.startWS()
	return
}

func (b *BitmexTrade) Stop() (err error) {
	close(b.closeCh)
	return
}

func (b *BitmexTrade) Watch(param WatchParam) (err error) {
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

func (b *BitmexTrade) CancelAllOrders() (orders []*Order, err error) {
	orders, err = b.bm.CancelAllOrders()
	return
}

func (b *BitmexTrade) ProcessOrder(act TradeAction) (ret *Order, err error) {
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
func (b *BitmexTrade) GetDataChan() *exchange.ExchangeChan {
	return b.datas
}
