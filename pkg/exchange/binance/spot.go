package binance

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	gobinance "github.com/adshao/go-binance/v2"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/ztrade/trademodel"
	. "github.com/ztrade/trademodel"
	. "github.com/ztrade/ztrade/pkg/core"
	"github.com/ztrade/ztrade/pkg/exchange/ws"
	// . "github.com/ztrade/ztrade/pkg/event"
)

var (
	BinanceSpotAddr = "wss://stream.binance.com:9443/ws/"
)

var _ Exchange = &BinanceSpot{}

func init() {
	RegisterExchange("binancespot", NewBinanceSpot)
}

type BinanceSpot struct {
	Name     string
	currency string
	api      *gobinance.Client

	datas   chan *ExchangeData
	closeCh chan bool
	symbols []gobinance.Symbol

	cancelService    *gobinance.CancelOpenOrdersService
	cancelOneService *gobinance.CancelOrderService
	klineLimit       int
	wsUserListenKey  string
	wsUser           *websocket.Conn

	wsDepth       *ws.WSConn
	wsMarketTrade *ws.WSConn
}

func NewBinanceSpot(cfg *viper.Viper, cltName string) (e Exchange, err error) {
	b, err := NewBinanceSpotEx(cfg, cltName)
	if err != nil {
		return
	}
	e = b
	return
}

func NewBinanceSpotEx(cfg *viper.Viper, cltName string) (b *BinanceSpot, err error) {
	b = new(BinanceSpot)
	b.Name = "binance_spot"
	if cltName == "" {
		cltName = "binance_spot"
	}
	b.klineLimit = 1500
	// isDebug := cfg.GetBool(fmt.Sprintf("exchanges.%s.debug", cltName))
	apiKey := cfg.GetString(fmt.Sprintf("exchanges.%s.key", cltName))
	apiSecret := cfg.GetString(fmt.Sprintf("exchanges.%s.secret", cltName))
	b.datas = make(chan *ExchangeData)
	b.closeCh = make(chan bool)
	b.currency = cfg.GetString(fmt.Sprintf("exchanges.%s.currency", cltName))
	if b.currency == "" {
		b.currency = "USDT"
	}

	b.api = gobinance.NewClient(apiKey, apiSecret)
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
	b.cancelService = b.api.NewCancelOpenOrdersService()
	b.cancelOneService = b.api.NewCancelOrderService()
	b.Start(map[string]interface{}{})
	return
}

func (b *BinanceSpot) Start(param map[string]interface{}) (err error) {
	// watch position and order changed
	err = b.startUserWS()
	return
}
func (b *BinanceSpot) Stop() (err error) {
	close(b.closeCh)
	return
}

// KlineChan get klines
func (b *BinanceSpot) GetKline(symbol, bSize string, start, end time.Time) (data chan *Candle, errCh chan error) {
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
				temp = transSpotCandle(v)
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
func (b *BinanceSpot) WatchKline(symbol SymbolInfo) (datas chan *CandleInfo, stopC chan struct{}, err error) {
	f := newSpotKlineFilter(b.Name, symbol.Resolutions)
	datas = f.GetData()
	doneC, stopC, err := gobinance.WsKlineServe(symbol.Symbol, symbol.Resolutions, f.ProcessEvent, b.handleError("watchKline"))
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

func (b *BinanceSpot) handleError(typ string) func(error) {
	return func(err error) {
		log.Errorf("binance %s error:%s", typ, err.Error())
	}
}

func (b *BinanceSpot) Watch(param WatchParam) (err error) {
	symbol := param.Extra.(string)
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
			var prev *trademodel.Candle
			log.Infof("emitCandles wathKline :%##v", symbolInfo)
			for v := range datas {
				candle := v.Data.(*Candle)
				if candle == nil {
					log.Error("emitCandles data type error:", reflect.TypeOf(v.Data))
					continue
				}
				if prev == nil {
					prev = candle
					continue
				}
				if candle.Start == prev.Start {
					prev = candle
					continue
				}
				d := NewExchangeData("candle", EventCandle, prev)
				d.Symbol = v.Symbol
				d.Data.Extra = cParam.BinSize
				b.datas <- d
				prev = candle
			}
			if err != nil {
				log.Error("exchange emitCandle error:", err.Error())
			}
		}()

	case EventDepth:
		if b.wsDepth == nil {
			addr := fmt.Sprintf("%s%s@depth20@100ms", BinanceSpotAddr, strings.ToLower(symbol))
			b.wsDepth, err = ws.NewWSConn(addr, nil, b.parseBinanceSpotDepth(symbol))
		}
	case EventTradeMarket:
		addr := fmt.Sprintf("%s%s@trade", BinanceSpotAddr, strings.ToLower(symbol))
		b.wsMarketTrade, err = ws.NewWSConn(addr, nil, b.parseBinanceSpotMarketTrade(symbol))
	default:
		err = fmt.Errorf("unknown wathc param: %s", param.Type)
	}
	if err != nil {
		return
	}
	go func() {
		<-b.closeCh
		if stopC != nil {
			close(stopC)
		}
	}()
	return
}

func (b *BinanceSpot) CancelOrder(old *Order) (order *Order, err error) {
	resp, err := b.cancelOneService.Symbol(old.Symbol).Do(context.Background())
	if err != nil {
		return
	}
	price, err := strconv.ParseFloat(resp.Price, 64)
	if err != nil {
		panic(fmt.Sprintf("CancelOrder parse price %s error: %s", resp.Price, err.Error()))
	}
	amount, err := strconv.ParseFloat(resp.OrigQuantity, 64)
	if err != nil {
		panic(fmt.Sprintf("CancelOrder parse damount %s error: %s", resp.OrigQuantity, err.Error()))
	}
	order = &Order{
		OrderID:  strconv.FormatInt(resp.OrderID, 10),
		Symbol:   resp.Symbol,
		Currency: resp.Symbol,
		Amount:   amount,
		Price:    price,
		Status:   strings.ToUpper(string(resp.Status)),
		Side:     strings.ToLower(string(resp.Side)),
		Time:     time.Unix(resp.TransactTime/1000, 0),
	}

	return
}

func (b *BinanceSpot) fixPrice(price float64, symbol string) (ret float64) {
	var temp interface{}
	var ok bool
	var str string
	var n, fvalue float64
	var err error
	ret = price
	for _, v := range b.symbols {
		if v.Symbol == symbol {
			n = math.Pow(10, float64(v.BaseAssetPrecision))
			for _, f := range v.Filters {
				temp, ok = f["filterType"]
				if !ok {
					continue
				}
				str, ok = temp.(string)
				if !ok || str != "PRICE_FILTER" {
					continue
				}
				temp, ok = f["tickSize"]
				if !ok {
					continue
				}
				str, ok = temp.(string)
				if !ok {
					continue
				}
				fvalue, err = strconv.ParseFloat(str, 64)
				if err != nil {
					log.Errorf("binance fixPrice failed:%s", str)
				}
				ret = float64(int(ret*n)/int(fvalue*n)) * fvalue
			}
			break
		}
	}
	return
}

func (b *BinanceSpot) ProcessOrder(act TradeAction) (ret *Order, err error) {
	ctx := context.Background()
	var side gobinance.SideType
	if act.Action.IsLong() {
		side = gobinance.SideTypeBuy
	} else {
		side = gobinance.SideTypeSell
	}
	orderType := gobinance.OrderTypeLimit
	var resp *gobinance.CreateOrderResponse
	if act.Action.IsStop() {
		orderType = gobinance.OrderTypeStopLossLimit
		price := act.Price * 0.9
		if act.Action.IsLong() {
			price = act.Price * 1.1
		}
		price = b.fixPrice(price, act.Symbol)
		fmt.Println(price)
		resp, err = b.api.NewCreateOrderService().Symbol(act.Symbol).
			StopPrice(fmt.Sprintf("%f", act.Price)).
			Price(fmt.Sprintf("%f", price)).
			Quantity(fmt.Sprintf("%f", act.Amount)).
			TimeInForce(gobinance.TimeInForceTypeGTC).
			Type(orderType).
			Side(side).
			Do(ctx)
		if err != nil {
			return
		}
		ret = transSpotStopCreateOrder(resp)
		if ret.Price == 0 {
			ret.Price = act.Price
		}
		if ret.Amount == 0 {
			ret.Amount = act.Amount
		}
		return
	}
	resp, err = b.api.NewCreateOrderService().Symbol(act.Symbol).
		Price(fmt.Sprintf("%f", act.Price)).
		Quantity(fmt.Sprintf("%f", act.Amount)).
		TimeInForce(gobinance.TimeInForceTypeGTC).
		Type(orderType).
		Side(side).
		Do(ctx)
	if err != nil {
		return
	}
	ret = transSpotCreateOrder(resp)
	return
}

func (b *BinanceSpot) CancelAllOrders() (orders []*Order, err error) {
	ctx := context.Background()
	_, err = b.cancelService.Symbol("BTCUSDT").Do(ctx)
	if err != nil {
		log.Warn("BinanaceSpot cancel order:", err.Error())
		err = nil
	}
	return
}

func (b *BinanceSpot) GetSymbols() (symbols []SymbolInfo, err error) {
	ctx, cancel := context.WithTimeout(background, time.Second*5)
	defer cancel()
	resp, err := b.api.NewExchangeInfoService().Do(ctx)
	if err != nil {
		return
	}
	b.symbols = resp.Symbols
	symbols = make([]SymbolInfo, len(resp.Symbols))
	for i, v := range resp.Symbols {
		symbols[i] = SymbolInfo{
			Exchange:    "binance",
			Symbol:      v.Symbol,
			Resolutions: "1m,5m,15m,30m,1h,4h,1d,1w",
			Pricescale:  v.QuotePrecision,
		}
	}

	return
}

func (b *BinanceSpot) GetDataChan() chan *ExchangeData {
	return b.datas
}

func transSpotOrder(fo *gobinance.Order) (o *Order) {
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

func transSpotStopCreateOrder(fo *gobinance.CreateOrderResponse) (o *Order) {
	price, _ := strconv.ParseFloat(fo.Price, 64)
	amount, _ := strconv.ParseFloat(fo.OrigQuantity, 64)
	o = &Order{
		OrderID:  strconv.FormatInt(fo.OrderID, 10),
		Symbol:   fo.Symbol,
		Currency: fo.Symbol,
		Amount:   amount,
		Price:    price,
		Status:   strings.ToUpper(string(fo.Status)),
		Side:     strings.ToLower(string(fo.Side)),
		Time:     time.Unix(fo.TransactTime/1000, 0),
	}
	return
}

func transSpotCreateOrder(fo *gobinance.CreateOrderResponse) (o *Order) {
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
		Time:     time.Unix(fo.TransactTime/1000, 0),
	}
	return
}

func transSpotCandle(candle *gobinance.Kline) (ret *Candle) {
	ret = &Candle{
		ID:       0,
		Start:    candle.OpenTime / 1000,
		Open:     parseFloat(candle.Open),
		High:     parseFloat(candle.High),
		Low:      parseFloat(candle.Low),
		Close:    parseFloat(candle.Close),
		Turnover: parseFloat(candle.QuoteAssetVolume),
		Volume:   parseFloat(candle.Volume),
		Trades:   candle.TradeNum,
	}
	return
}

func transSpotWSCandle(candle *gobinance.WsKline) (ret *CandleInfo) {
	ret = &CandleInfo{
		Symbol: candle.Symbol,
		Data: &Candle{
			ID:       0,
			Start:    candle.StartTime / 1000,
			Open:     parseFloat(candle.Open),
			High:     parseFloat(candle.High),
			Low:      parseFloat(candle.Low),
			Close:    parseFloat(candle.Close),
			Turnover: parseFloat(candle.QuoteVolume),
			Volume:   parseFloat(candle.Volume),
			Trades:   candle.TradeNum,
		}}
	return
}
