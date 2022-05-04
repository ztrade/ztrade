package binance

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	gobinance "github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	. "github.com/ztrade/trademodel"
	. "github.com/ztrade/ztrade/pkg/core"
	// . "github.com/ztrade/ztrade/pkg/event"
)

var (
	defaultBinSizes = map[string]bool{"1m": true, "5m": true, "1h": true, "1d": true}
	background      = context.Background()
)

var _ Exchange = &BinanceTrade{}

func init() {
	RegisterExchange("binance", NewBinanceExchange)
}

type OrderInfo struct {
	Order
	Action TradeType
	Filled bool
}

type BinanceTrade struct {
	Name   string
	api    *futures.Client
	symbol string

	datas   chan *ExchangeData
	closeCh chan bool

	cancelService   *futures.CancelAllOpenOrdersService
	klineLimit      int
	wsUserListenKey string
	wsUser          *websocket.Conn
}

func NewBinanceExchange(cfg *viper.Viper, cltName, symbol string) (e Exchange, err error) {
	b, err := NewBinanceTradeWithSymbol(cfg, cltName, symbol)
	if err != nil {
		return
	}
	e = b
	return
}

func NewBinanceTradeWithSymbol(cfg *viper.Viper, cltName, symbol string) (b *BinanceTrade, err error) {
	b = new(BinanceTrade)
	b.Name = "binance"
	if cltName == "" {
		cltName = "binance"
	}
	b.klineLimit = 1500
	// isDebug := cfg.GetBool(fmt.Sprintf("exchanges.%s.debug", cltName))
	apiKey := cfg.GetString(fmt.Sprintf("exchanges.%s.key", cltName))
	apiSecret := cfg.GetString(fmt.Sprintf("exchanges.%s.secret", cltName))

	b.symbol = symbol
	b.datas = make(chan *ExchangeData)
	b.closeCh = make(chan bool)

	// if isDebug{
	//     b.api = gobinance.NewFuturesClient(apiKey string, secretKey string)
	// }
	b.api = gobinance.NewFuturesClient(apiKey, apiSecret)
	clientProxy := cfg.GetString("proxy")
	if clientProxy != "" {
		var proxyURL *url.URL
		proxyURL, err = url.Parse(clientProxy)
		if err != nil {
			return
		}
		b.api.HTTPClient = &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
		websocket.DefaultDialer.Proxy = http.ProxyURL(proxyURL)
		websocket.DefaultDialer.HandshakeTimeout = time.Second * 60
	}
	b.cancelService = b.api.NewCancelAllOpenOrdersService().Symbol(b.symbol)
	return
}

func NewBinanceTrade(cfg *viper.Viper, cltName string) (b *BinanceTrade, err error) {
	return NewBinanceTradeWithSymbol(cfg, cltName, "BTCUSDT")
}

func (b *BinanceTrade) Start(param map[string]interface{}) (err error) {
	// watch position and order changed
	err = b.startUserWS()
	return
}
func (b *BinanceTrade) Stop() (err error) {
	close(b.closeCh)
	return
}

// KlineChan get klines
func (b *BinanceTrade) GetKline(symbol, bSize string, start, end time.Time) (data chan *Candle, errCh chan error) {
	data = make(chan *Candle, 1024*10)
	errCh = make(chan error, 1)
	go func() {
		defer func() {
			close(data)
			close(errCh)
		}()
		var temp *Candle
		ctx := context.Background()
		nStart := start.Unix() * 1000
		nEnd := end.Unix() * 1000
		var nPrevStart int64
		for {
			klines, err := b.api.NewKlinesService().Interval(bSize).Symbol(symbol).StartTime(nStart).EndTime(nEnd).Limit(b.klineLimit).Do(ctx)
			if err != nil {
				errCh <- err
				return
			}
			sort.Slice(klines, func(i, j int) bool {
				return klines[i].OpenTime < klines[j].OpenTime
			})

			for _, v := range klines {
				if v.OpenTime <= nPrevStart {
					continue
				}
				temp = transCandle(v)
				data <- temp
				nStart = temp.Start * 1000
			}
			if nStart >= nEnd || nStart <= nPrevStart || len(klines) == 0 {
				fmt.Println(time.Unix(nStart/1000, 0), start, end)
				break
			}
			nPrevStart = nStart
		}
	}()

	return
}

// WatchKline watch kline changes
func (b *BinanceTrade) WatchKline(symbol SymbolInfo) (datas chan *CandleInfo, stopC chan struct{}, err error) {
	f := newKlineFilter(b.Name, symbol.Resolutions)
	datas = f.GetData()
	doneC, stopC, err := futures.WsKlineServe(symbol.Symbol, symbol.Resolutions, f.ProcessEvent, b.handleError("watchKline"))
	go func() {
		select {
		case <-doneC:
		case <-b.closeCh:
			close(stopC)
		}
		f.Close()
	}()
	if err != nil {
		return
	}
	return
}

func (b *BinanceTrade) handleError(typ string) func(error) {
	return func(err error) {
		log.Errorf("binance %s error:%s", typ, err.Error())
	}
}
func (b *BinanceTrade) handleAggTradeEvent(evt *futures.WsAggTradeEvent) {
	var err error
	var trade Trade
	trade.ID = fmt.Sprintf("%d", evt.AggregateTradeID)
	trade.Amount, err = strconv.ParseFloat(evt.Quantity, 64)
	if err != nil {
		log.Errorf("AggTradeEvent parse amount failed: %s", evt.Quantity)
	}
	trade.Price, err = strconv.ParseFloat(evt.Price, 64)
	if err != nil {
		log.Errorf("AggTradeEvent parse amount failed: %s", evt.Quantity)
	}
	trade.Time = time.Unix(evt.Time/1000, (evt.Time%1000)*int64(time.Millisecond))
	b.datas <- NewExchangeData(b.Name, EventTradeMarket, &trade)
}

func (b *BinanceTrade) handleDepth(evt *futures.WsDepthEvent) {
	var depth Depth
	var err error
	var price, amount float64
	depth.UpdateTime = time.Unix(evt.TransactionTime/1000, (evt.TransactionTime%1000)*int64(time.Millisecond))
	for _, v := range evt.Asks {
		// depth.Sells
		price, err = strconv.ParseFloat(v.Price, 64)
		if err != nil {
			log.Errorf("handleDepth parse price failed: %s", v.Price)
		}
		amount, err = strconv.ParseFloat(v.Quantity, 64)
		if err != nil {
			log.Errorf("handleDepth parse amount failed: %s", v.Quantity)
		}
		depth.Sells = append(depth.Sells, DepthInfo{Price: price, Amount: amount})
	}
	for _, v := range evt.Bids {
		// depth.Sells
		price, err = strconv.ParseFloat(v.Price, 64)
		if err != nil {
			log.Errorf("handleDepth parse price failed: %s", v.Price)
		}
		amount, err = strconv.ParseFloat(v.Quantity, 64)
		if err != nil {
			log.Errorf("handleDepth parse amount failed: %s", v.Quantity)
		}
		depth.Buys = append(depth.Buys, DepthInfo{Price: price, Amount: amount})
	}
	b.datas <- NewExchangeData(b.Name, EventDepth, &depth)
}

func (b *BinanceTrade) Watch(param WatchParam) (err error) {
	var stopC chan struct{}
	switch param.Type {
	case EventWatchCandle:
		cParam, ok := param.Data.(*CandleParam)
		if !ok {
			err = fmt.Errorf("event not CandleParam %s %#v", param.Type, param.Data)
			return
		}
		symbolInfo := SymbolInfo{Exchange: cParam.Exchange, Symbol: cParam.Symbol, Resolutions: cParam.BinSize}
		var datas chan *CandleInfo
		datas, _, err = b.WatchKline(symbolInfo)
		if err != nil {
			log.Errorf("emitCandles wathKline failed: %s", err.Error())
			return
		}
		go func() {
			var tLast int64
			log.Infof("emitCandles wathKline :%##v", symbolInfo)
			for v := range datas {
				candle := v.Data.(*Candle)
				if candle == nil {
					log.Error("emitCandles data type error:", reflect.TypeOf(v.Data))
					continue
				}
				if candle.Start == tLast {
					continue
				}
				d := NewExchangeData("candle", EventCandle, candle)
				d.Data.Extra = cParam.BinSize
				b.datas <- d
				tLast = candle.Start
			}
			if b.closeCh != nil {
				b.closeCh <- true
			}

			if err != nil {
				log.Error("exchange emitCandle error:", err.Error())
			}
		}()

	case EventDepth:
		_, stopC, err = futures.WsPartialDepthServe(b.symbol, 10, b.handleDepth, b.handleError("depth"))
	case EventTradeMarket:
		_, stopC, err = futures.WsAggTradeServe(b.symbol, b.handleAggTradeEvent, b.handleError("aggTrade"))
	default:
		err = fmt.Errorf("unknown wathc param: %s", param.Type)
	}
	if err != nil {
		return
	}
	go func() {
		<-b.closeCh
		close(stopC)
	}()
	return
}

func (b *BinanceTrade) ProcessOrder(act TradeAction) (ret *Order, err error) {
	ctx := context.Background()
	orderType := futures.OrderTypeLimit
	if act.Action.IsStop() {
		orderType = futures.OrderTypeStopMarket
	}
	var side futures.SideType
	if act.Action.IsLong() {
		side = futures.SideTypeBuy
	} else {
		side = futures.SideTypeSell
	}
	resp, err := b.api.NewCreateOrderService().Symbol(b.symbol).
		Price(fmt.Sprintf("%f", act.Price)).
		Quantity(fmt.Sprintf("%f", act.Amount)).
		TimeInForce(futures.TimeInForceTypeGTC).
		Type(orderType).
		Side(side).
		Do(ctx)
	if err != nil {
		return
	}
	ret = transCreateOrder(resp)
	return
}

func (b *BinanceTrade) CancelAllOrders() (orders []*Order, err error) {
	ctx := context.Background()
	ret, err := b.api.NewListOrdersService().Symbol(b.symbol).Do(ctx)
	var st string
	for _, v := range ret {
		st = string(v.Status)
		if st == OrderStatusFilled || st == OrderStatusCanceled {
			continue
		}
		orders = append(orders, transOrder(v))
	}
	err = b.cancelService.Symbol(b.symbol).Do(context.Background())
	return
}

func (b *BinanceTrade) GetSymbols() (symbols []SymbolInfo, err error) {
	ctx, cancel := context.WithTimeout(background, time.Second*5)
	defer cancel()
	resp, err := b.api.NewExchangeInfoService().Do(ctx)
	if err != nil {
		return
	}
	symbols = make([]SymbolInfo, len(resp.Symbols))
	for i, v := range resp.Symbols {
		symbols[i] = SymbolInfo{
			Exchange:    "binance",
			Symbol:      v.Symbol,
			Resolutions: "1m,5m,15m,30m,1h,4h,1d,1w",
			Pricescale:  v.QuantityPrecision,
		}
	}

	return
}

func (b *BinanceTrade) GetDataChan() chan *ExchangeData {
	return b.datas
}

func transOrder(fo *futures.Order) (o *Order) {
	price, err := strconv.ParseFloat(fo.Price, 64)
	if err != nil {
		panic(fmt.Sprintf("parse price %s error: %s", fo.Price, err.Error()))
	}
	amount, err := strconv.ParseFloat(fo.OrigQuantity, 64)
	if err != nil {
		panic(fmt.Sprintf("parse damount %s error: %s", fo.OrigQuantity, err.Error()))
	}
	o = &Order{
		OrderID:  strconv.FormatInt(fo.OrderID, 10),
		Symbol:   fo.Symbol,
		Currency: fo.Symbol,
		Amount:   amount,
		Price:    price,
		Status:   strings.ToUpper(string(fo.Status)),
		Side:     strings.ToLower(string(fo.Side)),
		Time:     time.Unix(fo.Time/1000, 0),
	}
	return
}

func transCreateOrder(fo *futures.CreateOrderResponse) (o *Order) {
	price, err := strconv.ParseFloat(fo.Price, 64)
	if err != nil {
		panic(fmt.Sprintf("parse price %s error: %s", fo.Price, err.Error()))
	}
	amount, err := strconv.ParseFloat(fo.OrigQuantity, 64)
	if err != nil {
		panic(fmt.Sprintf("parse damount %s error: %s", fo.OrigQuantity, err.Error()))
	}
	o = &Order{
		OrderID:  strconv.FormatInt(fo.OrderID, 10),
		Symbol:   fo.Symbol,
		Currency: fo.Symbol,
		Amount:   amount,
		Price:    price,
		Status:   strings.ToUpper(string(fo.Status)),
		Side:     strings.ToLower(string(fo.Side)),
		Time:     time.Unix(fo.UpdateTime/1000, 0),
	}
	return
}

func transCandle(candle *futures.Kline) (ret *Candle) {
	ret = &Candle{
		ID:     0,
		Start:  candle.OpenTime / 1000,
		Open:   parseFloat(candle.Open),
		High:   parseFloat(candle.High),
		Low:    parseFloat(candle.Low),
		Close:  parseFloat(candle.Close),
		VWP:    parseFloat(candle.QuoteAssetVolume),
		Volume: parseFloat(candle.Volume),
		Trades: candle.TradeNum,
	}
	return
}

func parseFloat(str string) float64 {
	f, err := strconv.ParseFloat(str, 64)
	if err != nil {
		panic("binance parseFloat error:" + err.Error())
	}
	return f
}

func transWSCandle(candle *futures.WsKline) (ret *CandleInfo) {
	ret = &CandleInfo{
		Symbol: candle.Symbol,
		Data: &Candle{
			ID:     0,
			Start:  candle.StartTime / 1000,
			Open:   parseFloat(candle.Open),
			High:   parseFloat(candle.High),
			Low:    parseFloat(candle.Low),
			Close:  parseFloat(candle.Close),
			VWP:    parseFloat(candle.QuoteVolume),
			Volume: parseFloat(candle.Volume),
			Trades: candle.TradeNum,
		}}
	return
}
