package ok

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	. "github.com/ztrade/trademodel"
	. "github.com/ztrade/ztrade/pkg/core"
	"github.com/ztrade/ztrade/pkg/exchange/ok/api/market"
	"github.com/ztrade/ztrade/pkg/exchange/ok/api/trade"
)

var (
	background = context.Background()

	ApiAddr       = "https://www.okex.com/"
	WSOkexPUbilc  = "wss://wsaws.okex.com:8443/ws/v5/public"
	WSOkexPrivate = "wss://wsaws.okex.com:8443/ws/v5/private"
)

var _ Exchange = &OkexTrade{}

func init() {
	RegisterExchange("okex", NewOkexExchange)
}

type OrderInfo struct {
	Order
	Action TradeType
	Filled bool
}

type OkexTrade struct {
	Name      string
	tradeApi  *trade.ClientWithResponses
	marketApi *market.ClientWithResponses
	symbol    string

	datas   chan *ExchangeData
	closeCh chan bool

	apiKey    string
	apiSecret string
	apiPwd    string

	klineLimit int
	wsUser     *websocket.Conn
	wsPublic   *websocket.Conn

	ordersCache     sync.Map
	stopOrdersCache sync.Map
}

func NewOkexExchange(cfg *viper.Viper, cltName, symbol string) (e Exchange, err error) {
	b, err := NewOkexTradeWithSymbol(cfg, cltName, symbol)
	if err != nil {
		return
	}
	e = b
	return
}

func NewOkexTradeWithSymbol(cfg *viper.Viper, cltName, symbol string) (b *OkexTrade, err error) {
	b = new(OkexTrade)
	b.Name = "okex"
	if cltName == "" {
		cltName = "okex"
	}
	b.klineLimit = 100
	// isDebug := cfg.GetBool(fmt.Sprintf("exchanges.%s.debug", cltName))
	b.apiKey = cfg.GetString(fmt.Sprintf("exchanges.%s.key", cltName))
	b.apiSecret = cfg.GetString(fmt.Sprintf("exchanges.%s.secret", cltName))
	b.apiPwd = cfg.GetString(fmt.Sprintf("exchanges.%s.pwd", cltName))

	b.symbol = symbol
	b.datas = make(chan *ExchangeData, 1024)
	b.closeCh = make(chan bool)

	b.tradeApi, err = trade.NewClientWithResponses(ApiAddr)
	if err != nil {
		return
	}
	b.marketApi, err = market.NewClientWithResponses(ApiAddr)
	if err != nil {
		return
	}
	clientProxy := cfg.GetString("proxy")
	if clientProxy != "" {
		var proxyURL *url.URL
		proxyURL, err = url.Parse(clientProxy)
		if err != nil {
			return
		}
		clt := b.marketApi.ClientInterface.(*market.Client).Client.(*http.Client)
		*clt = http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
		clt = b.tradeApi.ClientInterface.(*trade.Client).Client.(*http.Client)
		*clt = http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
		websocket.DefaultDialer.Proxy = http.ProxyURL(proxyURL)
		websocket.DefaultDialer.HandshakeTimeout = time.Second * 60
	}
	b.Start(map[string]interface{}{})
	return
}

func NewOkexTrade(cfg *viper.Viper, cltName string) (b *OkexTrade, err error) {
	return NewOkexTradeWithSymbol(cfg, cltName, "BTCUSDT")
}

func (b *OkexTrade) auth(ctx context.Context, req *http.Request) (err error) {
	var temp []byte
	if req.Method != "GET" {
		temp, err = ioutil.ReadAll(req.Body)
		if err != nil {
			return
		}
		req.Body.Close()
		buf := bytes.NewBuffer(temp)
		req.Body = io.NopCloser(buf)
	} else {
		temp = []byte(fmt.Sprintf("?%s", req.URL.RawQuery))
	}
	var signStr string
	tmStr := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	signStr = fmt.Sprintf("%s%s%s%s", tmStr, req.Method, req.URL.Path, string(temp))

	h := hmac.New(sha256.New, []byte(b.apiSecret))
	h.Write([]byte(signStr))
	ret := h.Sum(nil)
	n := base64.StdEncoding.EncodedLen(len(ret))
	dst := make([]byte, n)
	base64.StdEncoding.Encode(dst, ret)
	sign := string(dst)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OK-ACCESS-KEY", b.apiKey)
	req.Header.Set("OK-ACCESS-SIGN", sign)

	req.Header.Set("OK-ACCESS-TIMESTAMP", tmStr)
	req.Header.Set("OK-ACCESS-PASSPHRASE", b.apiPwd)
	return
}

func (b *OkexTrade) Start(param map[string]interface{}) (err error) {
	err = b.runPublic()
	if err != nil {
		return
	}
	err = b.runPrivate()
	if err != nil {
		return
	}
	return
}
func (b *OkexTrade) Stop() (err error) {
	close(b.closeCh)
	return
}

// KlineChan get klines
func (b *OkexTrade) GetKline(symbol, bSize string, start, end time.Time) (data chan *Candle, errCh chan error) {
	data = make(chan *Candle, 1024*10)
	errCh = make(chan error, 1)
	go func() {
		defer func() {
			close(data)
			close(errCh)
		}()
		nStart := start.Unix() * 1000
		nEnd := end.Unix() * 1000
		tempEnd := nEnd
		var nPrevStart int64
		var resp *market.GetApiV5MarketHistoryCandlesResponse
		var startStr, endStr string
		var err error
		for {
			ctx, cancel := context.WithTimeout(background, time.Second*3)
			startStr = strconv.FormatInt(nStart, 10)
			tempEnd = nStart + 100*60*1000
			endStr = strconv.FormatInt(tempEnd, 10)
			var params = market.GetApiV5MarketHistoryCandlesParams{InstId: b.symbol, Bar: &bSize, Before: &startStr, After: &endStr}
			resp, err = b.marketApi.GetApiV5MarketHistoryCandlesWithResponse(ctx, &params)
			cancel()
			if err != nil {
				errCh <- err
				return
			}
			klines, err := parseCandles(resp)
			if err != nil {
				if strings.Contains(err.Error(), "Requests too frequent.") {
					time.Sleep(time.Second)
					continue
				}
				errCh <- err
				return
			}
			sort.Slice(klines, func(i, j int) bool {
				return klines[i].Start < klines[j].Start
			})

			for _, v := range klines {
				if v.Start*1000 <= nPrevStart {
					continue
				}
				data <- v
				nStart = v.Start * 1000
			}
			if len(klines) == 0 {
				nStart = tempEnd
			}
			if nStart >= nEnd || nStart <= nPrevStart {
				fmt.Println(time.Unix(nStart/1000, 0), start, end)
				break
			}
			nPrevStart = nStart
		}
	}()

	return
}

func (b *OkexTrade) Watch(param WatchParam) (err error) {
	log.Info("okex watch:", param)
	switch param.Type {
	case EventWatchCandle:
		var p = OPParam{
			OP: "subscribe",
			Args: []interface{}{
				OPArg{Channel: "candle1m", InstType: "SWAP", InstID: b.symbol},
			},
		}
		err = b.wsPublic.WriteJSON(p)
	case EventDepth:
		var p = OPParam{
			OP: "subscribe",
			Args: []interface{}{
				OPArg{Channel: "books5", InstType: "SWAP", InstID: b.symbol},
			},
		}
		err = b.wsPublic.WriteJSON(p)
	case EventTradeMarket:
		var p = OPParam{
			OP: "subscribe",
			Args: []interface{}{
				OPArg{Channel: "trades", InstType: "SWAP", InstID: b.symbol},
			},
		}
		err = b.wsPublic.WriteJSON(p)
	default:
		err = fmt.Errorf("unknown wath param: %s", param.Type)
	}
	return
}
func (b *OkexTrade) processStopOrder(act TradeAction) (ret *Order, err error) {
	ctx, cancel := context.WithTimeout(background, time.Second*2)
	defer cancel()
	var side, posSide string
	if act.Action.IsLong() {
		side = "buy"
		posSide = "long"
	} else {
		side = "sell"
		posSide = "short"
	}
	reduceOnly := true
	var orderPx = "-1"
	triggerPx := fmt.Sprintf("%f", act.Price)
	// PostApiV5TradeOrderAlgoJSONBody defines parameters for PostApiV5TradeOrderAlgo.
	params := trade.PostApiV5TradeOrderAlgoJSONBody{
		// 非必填<br>保证金币种，如：USDT<br>仅适用于单币种保证金模式下的全仓杠杆订单
		//	Ccy *string `json:"ccy,omitempty"`

		// 必填<br>产品ID，如：`BTC-USDT`
		InstId: b.symbol,

		// 必填<br>订单类型。<br>`conditional`：单向止盈止损<br>`oco`：双向止盈止损<br>`trigger`：计划委托<br>`iceberg`：冰山委托<br>`twap`：时间加权委托
		OrdType: "conditional",

		// 非必填<br>委托价格<br>委托价格为-1时，执行市价委托<br>适用于`计划委托`
		OrderPx: &orderPx,

		// 可选<br>持仓方向<br>在双向持仓模式下必填，且仅可选择 `long` 或 `short`
		PosSide: &posSide,

		// 非必填<br>挂单限制价<br>适用于`冰山委托`和`时间加权委托`
		//	PxLimit *string `json:"pxLimit,omitempty"`

		// 非必填<br>距离盘口的比例价距<br>适用于`冰山委托`和`时间加权委托`
		//	PxSpread *string `json:"pxSpread,omitempty"`

		// 非必填<br>距离盘口的比例<br>pxVar和pxSpread只能传入一个<br>适用于`冰山委托`和`时间加权委托`
		//	PxVar *string `json:"pxVar,omitempty"`

		// 非必填<br>是否只减仓，`true` 或 `false`，默认`false`<br>仅适用于币币杠杆订单
		ReduceOnly: &reduceOnly,

		// 必填<br>订单方向。买：`buy` 卖：`sell`
		Side: side,

		// 非必填<br>止损委托价，如果填写此参数，必须填写止损触发价<br>委托价格为-1时，执行市价止损<br>适用于`止盈止损委托`
		SlOrdPx: &orderPx,

		// 非必填<br>止损触发价，如果填写此参数，必须填写止损委托价<br>适用于`止盈止损委托`
		SlTriggerPx: &triggerPx,

		// 必填<br>委托数量
		Sz: fmt.Sprintf("%d", int(act.Amount)),

		// 非必填<br>单笔数量<br>适用于`冰山委托`和`时间加权委托`
		//	SzLimit *string `json:"szLimit,omitempty"`

		// 必填<br>交易模式<br>保证金模式：`isolated`：逐仓 ；`cross`<br>全仓非保证金模式：`cash`：非保证金
		TdMode: "isolated",

		// 非必填<br>市价单委托数量的类型<br>交易货币：`base_ccy`<br>计价货币：`quote_ccy`<br>仅适用于币币订单
		//	TgtCcy *string `json:"tgtCcy,omitempty"`

		// 非必填<br>挂单限制价<br>适用于`时间加权委托`
		//	TimeInterval *string `json:"timeInterval,omitempty"`

		// 非必填<br>止盈委托价，如果填写此参数，必须填写止盈触发价<br>委托价格为-1时，执行市价止盈<br>适用于`止盈止损委托`
		//        TpOrdPx ,

		// 非必填<br>止盈触发价，如果填写此参数，必须填写止盈委托价<br>适用于`止盈止损委托`
		//	TpTriggerPx *string `json:"tpTriggerPx,omitempty"`

		// 非必填<br>计划委托触发价格<br>适用于`计划委托`
		//	TriggerPx *string `json:"triggerPx,omitempty"`
	}
	resp, err := b.tradeApi.PostApiV5TradeOrderAlgoWithResponse(ctx, params, b.auth)
	if err != nil {
		return
	}

	fmt.Println(string(resp.Body))
	orders, err := parsePostAlgoOrders(b.symbol, "open", side, act.Price, act.Amount, resp.Body)
	if err != nil {
		return
	}
	if len(orders) != 1 {
		err = fmt.Errorf("orders len not match: %#v", orders)
		log.Warnf(err.Error())
		return
	}
	ret = orders[0]
	return
}

func (b *OkexTrade) ProcessOrder(act TradeAction) (ret *Order, err error) {
	if act.Action.IsStop() {
		ret, err = b.processStopOrder(act)
		fmt.Println("store algo id:", ret.OrderID)
		b.stopOrdersCache.Store(ret.OrderID, ret)
		return
	}
	ctx, cancel := context.WithTimeout(background, time.Second*2)
	defer cancel()
	var side, posSide, px string
	if act.Action.IsLong() {
		side = "buy"
		posSide = "long"
	} else {
		side = "sell"
		posSide = "short"
	}
	ordType := "limit"
	tag := "ztrade"
	px = fmt.Sprintf("%f", act.Price)
	params := trade.PostApiV5TradeOrderJSONRequestBody{
		//ClOrdId *string `json:"clOrdId,omitempty"`
		// 必填<br>产品ID，如：`BTC-USDT`
		InstId: b.symbol,
		// 必填<br>订单类型。<br>市价单：`market`<br>限价单：`limit`<br>只做maker单：`post_only`<br>全部成交或立即取消：`fok`<br>立即成交并取消剩余：`ioc`<br>市价委托立即成交并取消剩余：`optimal_limit_ioc`（仅适用交割、永续）
		OrdType: ordType,

		// 可选<br>持仓方向<br>在双向持仓模式下必填，且仅可选择 `long` 或 `short`
		PosSide: &posSide,

		// 可选<br>委托价格<br>仅适用于`limit`、`post_only`、`fok`、`ioc`类型的订单
		Px: &px,

		// 非必填<br>是否只减仓，`true` 或 `false`，默认`false`<br>仅适用于币币杠杆订单
		//	ReduceOnly *bool `json:"reduceOnly,omitempty"`
		// 必填<br>订单方向。买：`buy` 卖：`sell`
		Side: side,
		// 必填<br>委托数量
		Sz: fmt.Sprintf("%d", int(act.Amount)),
		// 非必填<br>订单标签<br>字母（区分大小写）与数字的组合，可以是纯字母、纯数字，且长度在1-8位之间。
		Tag: &tag,
		// 必填<br>交易模式<br>保证金模式：`isolated`：逐仓 ；`cross`<br>全仓非保证金模式：`cash`：非保证金
		TdMode: "isolated",
		// 非必填<br>市价单委托数量的类型<br>交易货币：`base_ccy`<br>计价货币：`quote_ccy`<br>仅适用于币币订单
		//	TgtCcy *string `json:"tgtCcy,omitempty"`
	}
	resp, err := b.tradeApi.PostApiV5TradeOrderWithResponse(ctx, params, b.auth)
	if err != nil {
		return
	}

	orders, err := parsePostOrders(b.symbol, "open", side, act.Price, act.Amount, resp.Body)
	if err != nil {
		return
	}
	if len(orders) != 1 {
		err = fmt.Errorf("orders len not match: %#v", orders)
		log.Warnf(err.Error())
		return
	}
	ret = orders[0]
	b.ordersCache.Store(ret.OrderID, ret)
	return
}

type CancelNormalResp struct {
	Code string        `json:"code"`
	Msg  string        `json:"msg"`
	Data []OrderNormal `json:"data"`
}

type CancelAlgoResp struct {
	Code string      `json:"code"`
	Msg  string      `json:"msg"`
	Data []AlgoOrder `json:"data"`
}

func (b *OkexTrade) cancelAllNormal() (orders []*Order, err error) {
	ctx, cancel := context.WithTimeout(background, time.Second*3)
	defer cancel()
	instType := "SWAP"
	var params = trade.GetApiV5TradeOrdersPendingParams{
		InstId:   &b.symbol,
		InstType: &instType,
	}
	resp, err := b.tradeApi.GetApiV5TradeOrdersPendingWithResponse(ctx, &params, b.auth)
	if err != nil {
		return
	}
	var orderResp CancelNormalResp
	err = json.Unmarshal(resp.Body, &orderResp)
	if err != nil {
		return
	}
	if orderResp.Code != "0" {
		err = errors.New(string(resp.Body))
		return
	}

	var body trade.PostApiV5TradeCancelBatchOrdersJSONRequestBody
	for _, v := range orderResp.Data {
		temp := v.OrdID
		fmt.Println("order:", v.OrdID)
		body = append(body, trade.CancelBatchOrder{
			InstId: b.symbol,
			OrdId:  &temp,
		})
	}

	cancelResp, err := b.tradeApi.PostApiV5TradeCancelBatchOrdersWithResponse(ctx, body, b.auth)
	if err != nil {
		return
	}
	temp := OKEXOrder{}
	err = json.Unmarshal(cancelResp.Body, &temp)
	if err != nil {
		return
	}
	if temp.Code != "0" {
		err = errors.New(string(cancelResp.Body))
	}
	return
}

func (b *OkexTrade) cancelAllAlgo() (orders []*Order, err error) {
	ctx, cancel := context.WithTimeout(background, time.Second*3)
	defer cancel()
	instType := "SWAP"
	var params = trade.GetApiV5TradeOrdersAlgoPendingParams{
		OrdType:  "conditional",
		InstId:   &b.symbol,
		InstType: &instType,
	}
	resp, err := b.tradeApi.GetApiV5TradeOrdersAlgoPendingWithResponse(ctx, &params, b.auth)
	if err != nil {
		return
	}
	var orderResp CancelAlgoResp
	err = json.Unmarshal(resp.Body, &orderResp)
	if err != nil {
		return
	}
	if orderResp.Code != "0" {
		err = errors.New(string(resp.Body))
		return
	}

	var body trade.PostApiV5TradeCancelAlgosJSONRequestBody
	for _, v := range orderResp.Data {
		body = append(body, trade.CancelAlgoOrder{
			InstId: b.symbol,
			AlgoId: v.AlgoID,
		})
	}

	cancelResp, err := b.tradeApi.PostApiV5TradeCancelAlgosWithResponse(ctx, body, b.auth)
	if err != nil {
		return
	}
	temp := OKEXAlgoOrder{}
	err = json.Unmarshal(cancelResp.Body, &temp)
	if err != nil {
		return
	}
	if temp.Code != "0" {
		err = errors.New(string(cancelResp.Body))
	}
	return
}
func (b *OkexTrade) CancelAllOrders() (orders []*Order, err error) {
	temp, err := b.cancelAllNormal()
	if err != nil {
		return
	}
	orders, err = b.cancelAllAlgo()
	if err != nil {
		return
	}
	orders = append(temp, orders...)
	// b.ordersCache.Range(func(key, value interface{}) bool {
	// 	orderId := key.(string)
	// 	ctx, cancel := context.WithTimeout(background, time.Second*3)
	// 	var body = trade.PostApiV5TradeCancelOrderJSONRequestBody{
	// 		InstId: b.symbol,
	// 		OrdId:  &orderId,
	// 	}
	// 	resp, err := b.tradeApi.PostApiV5TradeCancelOrderWithResponse(ctx, body, b.auth)
	// 	cancel()
	// 	if err != nil {
	// 		return false
	// 	}
	// 	fmt.Println(string(resp.Body))
	// 	return true
	// })
	// b.stopOrdersCache.Range(func(key, value interface{}) bool {
	// 	orderId := key.(string)
	// 	ctx, cancel := context.WithTimeout(background, time.Second*3)
	// 	var body = trade.PostApiV5TradeCancelAlgosJSONRequestBody{
	// 		AlgoId: orderId,
	// 		InstId: b.symbol,
	// 	}
	// 	fmt.Println("body:", body)
	// 	resp, err := b.tradeApi.PostApiV5TradeCancelAlgosWithResponse(ctx, body, b.auth)
	// 	cancel()
	// 	if err != nil {
	// 		return false
	// 	}
	// 	fmt.Println("algo:", string(resp.Body))
	// 	return true
	// })

	//	orders, err = parsePostOrders(b.symbol, "cancel", "", 0, 0, resp.Body)
	return
}

func (b *OkexTrade) GetDataChan() chan *ExchangeData {
	return b.datas
}

func transCandle(values [7]string) (ret *Candle) {
	nTs, err := strconv.ParseInt(values[0], 10, 64)
	if err != nil {
		panic(fmt.Sprintf("trans candle error: %#v", values))
		return nil
	}
	ret = &Candle{
		ID:     0,
		Start:  nTs / 1000,
		Open:   parseFloat(values[1]),
		High:   parseFloat(values[2]),
		Low:    parseFloat(values[3]),
		Close:  parseFloat(values[4]),
		Volume: parseFloat(values[5]),
		VWP:    parseFloat(values[6]),
	}
	return
}

func parseFloat(str string) float64 {
	f, err := strconv.ParseFloat(str, 64)
	if err != nil {
		panic("okex parseFloat error:" + err.Error())
	}
	return f
}

func parseCandles(resp *market.GetApiV5MarketHistoryCandlesResponse) (candles []*Candle, err error) {
	var candleResp CandleResp
	err = json.Unmarshal(resp.Body, &candleResp)
	if err != nil {
		return
	}
	if candleResp.Code != "0" {
		err = errors.New(string(resp.Body))
		return
	}
	for _, v := range candleResp.Data {
		temp := transCandle(v)
		candles = append(candles, temp)
	}
	return
}

func parsePostOrders(symbol, status, side string, amount, price float64, body []byte) (ret []*Order, err error) {
	temp := OKEXOrder{}
	err = json.Unmarshal(body, &temp)
	if err != nil {
		return
	}
	if temp.Code != "0" {
		err = fmt.Errorf("error resp: %s", string(body))
		return
	}
	fmt.Println("data:", temp.Data)
	for _, v := range temp.Data {
		if v.SCode != "0" {
			err = fmt.Errorf("%s %s", v.SCode, v.SMsg)
			return
		}

		temp := &Order{
			OrderID: v.OrdID,
			Symbol:  symbol,
			// Currency
			Side:   side,
			Status: status,
			Price:  price,
			Amount: amount,
			Time:   time.Now(),
		}
		ret = append(ret, temp)
	}
	return
}

func parsePostAlgoOrders(symbol, status, side string, amount, price float64, body []byte) (ret []*Order, err error) {
	temp := OKEXAlgoOrder{}
	err = json.Unmarshal(body, &temp)
	if err != nil {
		return
	}
	if temp.Code != "0" {
		err = fmt.Errorf("error resp: %s", string(body))
		return
	}
	for _, v := range temp.Data {
		if v.SCode != "0" {
			err = fmt.Errorf("%s %s", v.SCode, v.SMsg)
			return
		}

		temp := &Order{
			OrderID: v.AlgoID,
			Symbol:  symbol,
			// Currency
			Side:   side,
			Status: status,
			Price:  price,
			Amount: amount,
			Time:   time.Now(),
		}
		ret = append(ret, temp)
	}
	return
}

type CandleResp struct {
	Code string      `json:"code"`
	Msg  string      `json:"msg"`
	Data [][7]string `json:"data"`
}

type OKEXOrder struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		ClOrdID string `json:"clOrdId"`
		OrdID   string `json:"ordId"`
		Tag     string `json:"tag"`
		SCode   string `json:"sCode"`
		SMsg    string `json:"sMsg"`
	} `json:"data"`
}

type OKEXAlgoOrder struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		AlgoID string `json:"algoId"`
		SCode  string `json:"sCode"`
		SMsg   string `json:"sMsg"`
	} `json:"data"`
}
