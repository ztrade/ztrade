package gateio

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/antihax/optional"
	"github.com/gateio/gateapi-go/v6"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/ztrade/trademodel"
	. "github.com/ztrade/ztrade/pkg/core"
	"github.com/ztrade/ztrade/pkg/exchange/ws"
)

var (
	GateIOUsdtFuturesWS = "wss://fx-ws.gateio.ws/v4/ws/"
)
var _ Exchange = &GateIO{}

func init() {
	RegisterExchange("gateio", NewGateIOExchange)
}

type GateIO struct {
	Name    string
	api     *gateapi.APIClient
	key     string
	secret  string
	settle  string
	userID  string
	datas   chan *ExchangeData
	closeCh chan bool
	symbols []SymbolInfo

	wsDepth       *ws.WSConn
	wsMarketTrade *ws.WSConn
	wsKline       *ws.WSConn
	wsUser        *ws.WSConn
}

func NewGateIOExchange(cfg *viper.Viper, cltName string) (e Exchange, err error) {
	g, err := NewGateIO(cfg, cltName)
	if err != nil {
		return nil, err
	}
	e = g
	return
}
func NewGateIO(cfg *viper.Viper, cltName string) (e *GateIO, err error) {
	g := new(GateIO)
	g.datas = make(chan *ExchangeData)
	g.closeCh = make(chan bool)
	apiCfg := gateapi.NewConfiguration()
	// apiCfg.Debug = true
	g.key = cfg.GetString(fmt.Sprintf("exchanges.%s.key", cltName))
	g.secret = cfg.GetString(fmt.Sprintf("exchanges.%s.secret", cltName))
	g.settle = cfg.GetString(fmt.Sprintf("exchanges.%s.settle", cltName))
	g.userID = cfg.GetString(fmt.Sprintf("exchanges.%s.user", cltName))
	if g.settle == "" {
		g.settle = "usdt"
	}
	apiCfg.Key = g.key
	apiCfg.Secret = g.secret
	clientProxy := cfg.GetString("proxy")
	if clientProxy != "" {
		var proxyURL *url.URL
		proxyURL, err = url.Parse(clientProxy)
		if err != nil {
			return
		}
		apiCfg.HTTPClient = &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
		websocket.DefaultDialer.Proxy = http.ProxyURL(proxyURL)
		websocket.DefaultDialer.HandshakeTimeout = time.Second * 60
	}
	g.api = gateapi.NewAPIClient(apiCfg)
	e = g
	_, err = g.GetSymbols()
	return
}

func (g *GateIO) Start(map[string]interface{}) (err error) {
	return g.startUserWS()
}

func (g *GateIO) Stop() (err error) {
	close(g.closeCh)
	return
}

func (g *GateIO) subPrivate(channel string, payload []interface{}) map[string]interface{} {
	ts := time.Now().Unix()
	hash := hmac.New(sha512.New, []byte(g.secret))
	hash.Write([]byte(fmt.Sprintf("channel=%s&event=%s&time=%d", channel, "subscribe", ts)))
	req := map[string]interface{}{
		"time":    ts,
		"channel": channel,
		"event":   "subscribe",
		"payload": payload,
		"auth": map[string]interface{}{
			"method": "api_key",
			"KEY":    g.key,
			"SIGN":   hex.EncodeToString(hash.Sum(nil)),
		},
	}
	return req
}

func (g *GateIO) startUserWS() (err error) {
	g.wsUser, err = ws.NewWSConnWithoutPing(fmt.Sprintf("%s%s", GateIOUsdtFuturesWS, g.settle), func(ws *ws.WSConn) error {
		req := g.subPrivate("futures.positions", []interface{}{g.userID, "!all"})
		ws.WriteMsg(req)
		fmt.Printf("|%s|\n", g.userID)
		req = g.subPrivate("futures.usertrades", []interface{}{g.userID, "!all"})
		ws.WriteMsg(req)
		return nil
	}, g.parseUserData)
	return
}

// Kline get klines
func (g *GateIO) GetKline(symbol, bSize string, start, end time.Time) (data chan *trademodel.Candle, errCh chan error) {
	data = make(chan *trademodel.Candle, 1024*10)
	errCh = make(chan error, 1)
	dur, err := time.ParseDuration(bSize)
	if err != nil {
		errCh <- err
		close(data)
		close(errCh)
		return
	}
	go func() {
		defer func() {
			close(data)
			close(errCh)
		}()
		var temp *trademodel.Candle
		ctx := context.Background()
		nStart := start.Unix()
		nEnd := end.Unix()
		var nPrevStart int64
		nDur := int64(dur / time.Second)
		var opt = gateapi.ListFuturesCandlesticksOpts{}
		opt.Interval = optional.NewString(bSize)
		for {
			opt.From = optional.NewInt64(nStart)
			opt.To = optional.NewInt64(nEnd)

			tMax := time.Now().Unix() - nDur
			klines, resp, err := g.api.FuturesApi.ListFuturesCandlesticks(ctx, g.settle, symbol, &opt)
			resp.Body.Close()
			if err != nil {
				errCh <- err
				return
			}
			sort.Slice(klines, func(i, j int) bool {
				return klines[i].T < klines[j].T
			})
			for k, v := range klines {
				if int64(v.T) <= nPrevStart {
					continue
				}
				temp = transCandle(&v)
				data <- temp
				nStart = temp.Start
				if k == len(klines)-1 {
					// check if candle is unfinished
					if int64(v.T) > tMax {
						logrus.Infof("skip unfinished candle: %##v\n", v)
						break
					}
				}
			}
			if nStart >= nEnd || nStart <= nPrevStart || len(klines) == 0 {
				fmt.Println(time.Unix(nStart, 0), start, end)
				break
			}
			nPrevStart = nStart
		}
	}()
	return
}

func (g *GateIO) doStopOrder(act trademodel.TradeAction) (ret *trademodel.Order, err error) {
	ctx := context.Background()
	order := gateapi.FuturesPriceTriggeredOrder{
		Initial: gateapi.FuturesInitialOrder{
			Contract:   act.Symbol,
			Size:       0,
			Price:      fmt.Sprintf("%f", act.Price),
			ReduceOnly: true,
			IsClose:    true,
			Close:      true,
			Tif:        "gtc",
		},
		Trigger: gateapi.FuturesPriceTrigger{
			StrategyType: 0,
			PriceType:    1,
			Price:        fmt.Sprintf("%f", act.Price),
		},
	}
	if act.Action.IsLong() {
		// order.OrderType = "close-short-order"
		order.Trigger.Rule = 1
	} else {
		// order.OrderType = "close-long-order"
		order.Trigger.Rule = 2
	}
	retOrder, resp, err := g.api.FuturesApi.CreatePriceTriggeredOrder(ctx, g.settle, order)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}
	ret = &trademodel.Order{
		OrderID:  fmt.Sprintf("%d", retOrder.Id),
		Symbol:   act.Symbol,
		Currency: act.Symbol,
		Amount:   act.Amount,
		Price:    act.Price,
		Status:   "",
		Time:     time.Now(),
		// Remark   string
	}
	return
}

// for trade
// ProcessOrder process order
func (g *GateIO) ProcessOrder(act trademodel.TradeAction) (ret *trademodel.Order, err error) {
	if act.Action.IsStop() {
		return g.doStopOrder(act)
	}
	ctx := context.Background()
	order := gateapi.FuturesOrder{Contract: act.Symbol,
		Price: fmt.Sprintf("%f", act.Price),
		Size:  int64(act.Amount),
		Tif:   "gtc",
	}
	if !act.Action.IsLong() {
		order.Size = -int64(act.Amount)
	}

	if !act.Action.IsOpen() {
		order.ReduceOnly = true
		order.Close = true
		order.Size = 0
	}
	retOrder, resp, err := g.api.FuturesApi.CreateFuturesOrder(ctx, g.settle, order)
	defer resp.Body.Close()
	if err != nil {
		return
	}
	ret, err = transOrder(&retOrder)
	return
}
func (g *GateIO) CancelAllOrders() (orders []*trademodel.Order, err error) {
	var retOrders []gateapi.FuturesOrder
	var triggerOrders []gateapi.FuturesPriceTriggeredOrder
	var resp *http.Response

	for _, v := range g.symbols {
		retOrders, resp, err = g.api.FuturesApi.CancelFuturesOrders(context.Background(), g.settle, v.Symbol, &gateapi.CancelFuturesOrdersOpts{})
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		var temp *trademodel.Order
		for _, v := range retOrders {
			temp, err = transOrder(&v)
			if err != nil {
				return
			}
			orders = append(orders, temp)
		}

		triggerOrders, resp, err = g.api.FuturesApi.CancelPriceTriggeredOrderList(context.Background(), g.settle, v.Symbol)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		for _, v := range triggerOrders {
			price, _ := strconv.ParseFloat(v.Trigger.Price, 64)
			ret := &trademodel.Order{
				OrderID:  fmt.Sprintf("%d", v.Id),
				Symbol:   v.Initial.Contract,
				Currency: g.settle,
				Amount:   math.Abs(float64(v.Initial.Size)),
				Price:    price,
				Status:   "canceled",
				Time:     time.Now(),
				// Remark   string
			}
			if v.Trigger.Rule == 1 {
				ret.Side = "buy"
			} else {
				ret.Side = "sell"
			}
			orders = append(orders, ret)
		}
	}

	return
}

func transOrder(retOrder *gateapi.FuturesOrder) (ret *trademodel.Order, err error) {
	p, err := strconv.ParseFloat(retOrder.Price, 64)
	if err != nil {
		return
	}

	ret = &trademodel.Order{
		OrderID:  fmt.Sprintf("%d", retOrder.Id),
		Symbol:   retOrder.Contract,
		Currency: retOrder.Contract,
		Amount:   math.Abs(float64(retOrder.Size)),
		Price:    p,
		Status:   retOrder.Status,
		Time:     time.Unix(int64(retOrder.CreateTime), 0),
		// Remark   string
	}
	if retOrder.Size > 0 {
		ret.Side = "buy"
	} else {
		ret.Side = "sell"
	}
	if retOrder.Status == "finished" {
		ret.Filled = float64(retOrder.Size)
	}
	return
}
func (g *GateIO) CancelOrder(old *trademodel.Order) (order *trademodel.Order, err error) {
	ctx := context.Background()
	retOrder, resp, err := g.api.FuturesApi.CancelFuturesOrder(ctx, g.settle, old.OrderID)
	defer resp.Body.Close()
	if err != nil {
		logrus.Errorf("cancel order:%s failed, try cacel stop order", old.OrderID)
		return g.cancelStopOrder(old)
	}
	order, err = transOrder(&retOrder)
	return
}

func (g *GateIO) cancelStopOrder(old *trademodel.Order) (order *trademodel.Order, err error) {
	ctx := context.Background()
	retOrder, resp, err := g.api.FuturesApi.CancelPriceTriggeredOrder(ctx, g.settle, old.OrderID)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}
	order = &trademodel.Order{
		OrderID: fmt.Sprintf("%d", retOrder.Id),
		Price:   old.Price,
		Amount:  old.Amount,
		Time:    time.Now(),
		Side:    old.Side,
		Status:  "canceled",
	}
	return
}

// GetBalanceChan
func (g *GateIO) GetDataChan() chan *ExchangeData {
	return g.datas
}

func (g *GateIO) GetSymbols() (symbols []SymbolInfo, err error) {
	contracts, resp, err := g.api.FuturesApi.ListFuturesContracts(context.Background(), "usdt")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	for _, v := range contracts {
		temp := SymbolInfo{
			Exchange:    "gateio",
			Symbol:      v.Name,
			Resolutions: "",
			Pricescale:  0,
		}
		symbols = append(symbols, temp)
	}
	g.symbols = symbols
	return
}

func (g *GateIO) Watch(param WatchParam) (err error) {
	symbol := param.Extra.(string)
	var stopC chan struct{}
	switch param.Type {
	case EventWatchCandle:
		cParam, ok := param.Data.(*CandleParam)
		if !ok {
			err = fmt.Errorf("event not CandleParam %s %#v", param.Type, param.Data)
			return
		}
		if g.wsKline == nil {
			g.wsKline, err = ws.NewWSConnWithoutPing(fmt.Sprintf("%s%s", GateIOUsdtFuturesWS, g.settle), func(ws *ws.WSConn) error {
				data := map[string]interface{}{"time": time.Now().Unix(),
					"channel": "futures.candlesticks",
					"event":   "subscribe",
					"payload": []string{cParam.BinSize, cParam.Symbol}}
				ws.WriteMsg(data)
				return nil
			}, g.parseKline(cParam.Symbol))
		}

	case EventDepth:
		if g.wsDepth == nil {
			g.wsDepth, err = ws.NewWSConnWithoutPing(fmt.Sprintf("%s%s", GateIOUsdtFuturesWS, g.settle), func(ws *ws.WSConn) error {
				data := map[string]interface{}{"time": time.Now().Unix(),
					"channel": "futures.order_book",
					"event":   "subscribe",
					"payload": []string{symbol, "20", "0"}}
				ws.WriteMsg(data)
				return nil
			}, g.parseDepth)
		}
	case EventTradeMarket:
		if g.wsMarketTrade == nil {
			g.wsMarketTrade, err = ws.NewWSConnWithoutPing(fmt.Sprintf("%s%s", GateIOUsdtFuturesWS, g.settle), func(ws *ws.WSConn) error {
				data := map[string]interface{}{"time": time.Now().Unix(),
					"channel": "futures.trades",
					"event":   "subscribe",
					"payload": []string{symbol}}
				ws.WriteMsg(data)
				return nil
			}, g.parseMarketTrade)
		}
	default:
		err = fmt.Errorf("unknown wathc param: %s", param.Type)
	}
	if err != nil {
		return
	}
	go func() {
		<-g.closeCh
		if stopC != nil {
			close(stopC)
		}
	}()
	return
}

func transCandle(candle *gateapi.FuturesCandlestick) (ret *trademodel.Candle) {
	ret = &trademodel.Candle{
		Start:  int64(candle.T),
		Volume: float64(candle.V),
	}
	ret.Close, _ = strconv.ParseFloat(candle.C, 64)
	ret.Open, _ = strconv.ParseFloat(candle.O, 64)
	ret.High, _ = strconv.ParseFloat(candle.H, 64)
	ret.Low, _ = strconv.ParseFloat(candle.L, 64)
	return
}
