package binance

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	. "github.com/SuperGod/trademodel"
	gobinance "github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	. "github.com/ztrade/ztrade/pkg/define"
	. "github.com/ztrade/ztrade/pkg/event"
	"github.com/ztrade/ztrade/pkg/process/exchange"
)

var (
	defaultBinSizes = map[string]bool{"1m": true, "5m": true, "1h": true, "1d": true}
	background      = context.Background()
)

var _ exchange.Exchange = &BinanceTrade{}

type OrderInfo struct {
	Order
	Action TradeType
	Filled bool
}

type BinanceTrade struct {
	BaseProcesser
	api    *futures.Client
	symbol string

	datas   *exchange.ExchangeChan
	closeCh chan bool

	cancelService   *futures.CancelAllOpenOrdersService
	klineLimit      int
	wsUserListenKey string
	wsUser          *websocket.Conn
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
	b.datas = exchange.NewExchangeChan()
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
	}
	b.cancelService = b.api.NewCancelAllOpenOrdersService().Symbol(b.symbol)
	return
}

func NewBinanceTrade(cfg *viper.Viper, cltName string) (b *BinanceTrade, err error) {
	return NewBinanceTradeWithSymbol(cfg, cltName, "BTCUSDT")
}

func (b *BinanceTrade) Start() (err error) {
	// watch position and order changed
	ctx := context.Background()
	listenKey, err := b.api.NewStartUserStreamService().Do(ctx)
	if err != nil {
		return
	}
	go func() {
		var err error
		ticker := time.NewTicker(time.Minute * 50)
		for range ticker.C {
			err = b.api.NewKeepaliveUserStreamService().ListenKey(listenKey).Do(ctx)
			if err != nil {
				log.Errorf("set listenkey error:%s", err.Error())
				break
			}
		}
	}()
	return
}
func (b *BinanceTrade) Stop() (err error) {
	close(b.closeCh)
	return
}

func (b *BinanceTrade) handleData() (err error) {
	return
}

// KlineChan get klines
func (b *BinanceTrade) KlineChan(start, end time.Time, symbol, bSize string) (data chan *Candle, errCh chan error) {
	data = make(chan *Candle)
	errCh = make(chan error)
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
			// fmt.Println("begin", bSize, symbol, nStart, nEnd)
			klines, err := b.api.NewKlinesService().Interval(bSize).Symbol(symbol).StartTime(nStart).EndTime(nEnd).Limit(b.klineLimit).Do(ctx)
			if err != nil {
				errCh <- err
				return
			}
			sort.Slice(klines, func(i, j int) bool {
				return klines[i].OpenTime < klines[j].OpenTime
			})
			nPrevStart = nStart
			for _, v := range klines {
				if v.OpenTime <= nPrevStart {
					continue
				}
				temp = transCandle(v)
				data <- temp
				nStart = temp.Start * 1000
			}
			if nStart >= nEnd || nStart >= nPrevStart || len(klines) == 0 {
				fmt.Println(time.Unix(nStart/1000, 0), start, end)
				break
			}

		}
	}()

	return
}

// WatchKline watch kline changes
func (b *BinanceTrade) WatchKline(symbol SymbolInfo) (datas chan *CandleInfo, stopC chan struct{}, err error) {
	f := newKlineFilter(b.Name, symbol.Resolutions)
	datas = f.GetData()
	doneC, stopC, err := futures.WsKlineServe(symbol.Symbol, symbol.Resolutions, f.ProcessEvent, b.handleError)
	go func() {
		select {
		case <-doneC:
		}
		f.Close()
	}()
	if err != nil {
		return
	}
	return
}

func (b *BinanceTrade) handleError(err error) {
	log.Errorf("binance watchkline error:%s", err.Error())
}

func (b *BinanceTrade) Watch(param WatchParam) (err error) {
	switch param.Type {
	case EventDepth:
	case EventTradeHistory:
	}
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

func (b *BinanceTrade) GetDataChan() *exchange.ExchangeChan {
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
