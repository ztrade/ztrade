package ok

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/bitly/go-simplejson"
	"github.com/gorilla/websocket"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	. "github.com/ztrade/trademodel"
	. "github.com/ztrade/ztrade/pkg/core"
)

var (
	pongMsg = []byte("pong")
)

func (b *OkexTrade) runPrivate() (err error) {
	u, err := url.Parse(WSOkexPrivate)
	if err != nil {
		return
	}

	b.wsUser, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
		return
	}
	login := OPParam{
		OP:   "login",
		Args: []interface{}{NewLoginArg(b.apiKey, b.apiPwd, b.apiSecret)},
	}
	err = b.wsUser.WriteJSON(login)
	if err != nil {
		return
	}
	go b.runPrivateLoop(b.wsUser, b.closeCh)
	return
}

type LoginArg struct {
	ApiKey     string `json:"apiKey"`
	Passphrase string `json:"passphrase"`
	Timestamp  int64  `json:"timestamp"`
	Sign       string `json:"sign"`
}

func NewLoginArg(apiKey, pass, secret string) *LoginArg {
	a := new(LoginArg)
	a.ApiKey = apiKey
	a.Passphrase = pass
	t := time.Now()
	a.Timestamp = t.Unix()
	src := fmt.Sprintf("%dGET/users/self/verify", a.Timestamp)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(src))
	ret := h.Sum(nil)
	n := base64.StdEncoding.EncodedLen(len(ret))
	dst := make([]byte, n)
	base64.StdEncoding.Encode(dst, ret)
	a.Sign = string(dst)
	return a
}

type OPArg struct {
	Channel  string `json:"channel"`
	InstType string `json:"instType"`
	Uly      string `json:"uly"`
	InstID   string `json:"instId"`
}

type OPParam struct {
	OP   string        `json:"op"`
	Args []interface{} `json:"args"`
}

func (b *OkexTrade) getSymbolSub() (p OPParam) {
	p.OP = "subscribe"
	p.Args = append(p.Args, OPArg{Channel: "orders", InstType: "SWAP", InstID: b.symbol})
	p.Args = append(p.Args, OPArg{Channel: "orders-algo", InstType: "SWAP", InstID: b.symbol})
	p.Args = append(p.Args, OPArg{Channel: "algo-advance", InstType: "SWAP", InstID: b.symbol})
	p.Args = append(p.Args, OPArg{Channel: "positions", InstType: "SWAP", InstID: b.symbol})
	return
}

// {"arg":{"channel":"trades","instId":"BSV-USD-210924"},"data":[{"instId":"BSV-USD-210924","tradeId":"1957771","px":"166.5","sz":"22","side":"sell","ts":"1621862533713"}]}
// {"arg":{"channel":"books","instId":"BSV-USD-210924"},"action":"update","data":[{"asks":[["167.4","54","0","1"]],"bids":[["164.81","0","0","0"],["164.69","79","0","1"]],"ts":"1621862579059","checksum":71082836}]}
// books5 {"arg":{"channel":"books5","instId":"BSV-USD-210924"},"data":[{"asks":[["166.35","20","0","1"],["166.4","135","0","2"],["166.42","86","0","1"],["166.45","310","0","1"],["166.46","61","0","2"]],"bids":[["166.14","33","0","1"],["166.07","106","0","1"],["166.05","2","0","1"],["166.04","97","0","1"],["165.98","20","0","1"]],"instId":"BSV-USD-210924","ts":"1621862688397"}]}

func (b *OkexTrade) runPrivateLoop(c *websocket.Conn, closeCh chan bool) {
	var err error
	// var t Trade
	// 	defer c.Close()
	var sj *simplejson.Json
	var evt string
	var message []byte
	var channel string
	var orders []OrderNormal
	var algoOrders []AlgoOrder
	// var algoOrders []AlgoOrder
	// 	var trades []model.Trade
	// 	var depths5 []model.OB
	var pos []OKEXPos
	var lastMsgTime time.Time
	go func() {
		ticker := time.NewTicker(time.Second * 5)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				dur := time.Since(lastMsgTime)
				if dur > time.Second*5 {
					c.WriteMessage(websocket.TextMessage, []byte("ping"))
				}
			case <-closeCh:
				return
			}
		}
	}()
	log.Info("okex runPrivate")
Out:
	for {
		select {
		case <-closeCh:
			break Out
		default:
		}
		_, message, err = c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}
		if bytes.Equal(message, pongMsg) {
			continue
		}
		lastMsgTime = time.Now()
		// fmt.Println(string(message))
		sj, err = simplejson.NewJson(message)
		if err != nil {
			log.Warnf("parse json error:%s", string(message))
			continue
		}
		evtValue, ok := sj.CheckGet("event")
		if ok {
			evt, err = evtValue.String()
			if err != nil {
				log.Warnf("login error:%s %s", string(message), err.Error())
				return
			}
			switch evt {
			case "error":
				log.Errorf("recv error: %s", string(message))
				return
			case "login":
				var code string
				code, err = sj.Get("code").String()
				if code != "0" {
					log.Warnf("login error:%s %s", string(message), err.Error())
					return
				}
				param := b.getSymbolSub()
				err = c.WriteJSON(param)
				if err != nil {
					return
				}
			default:
			}
			continue
		}
		arg, ok := sj.CheckGet("arg")
		if !ok {
			continue
		}
		channelValue, ok := arg.CheckGet("channel")
		if !ok {
			continue
		}
		channel, err = channelValue.String()
		if err != nil {
			log.Warnf("channelValue %v is not string error:%s", channelValue, err.Error())
			continue
		}
		switch channel {
		case "orders":
			fmt.Println("orders:", string(message))
			orders, err = parseOkexOrder(sj.Get("data"))
			if err != nil {
				log.Warnf("parseOkexOrder error:%s, %s", orders, err.Error())
				continue
			}
			for _, v := range orders {
				o := v.GetOrder()
				if o == nil {
					continue
				}
				o.Status = OrderStatusFilled
				b.ordersCache.Delete(v.OrdID)
				b.stopOrdersCache.Delete(v.OrdID)
				b.datas <- NewExchangeData(b.Name, EventOrder, o)
			}

		case "orders-algo":
			fmt.Println("orders-algo:", string(message))
			// 算法单最终还是会生成一个普通单子
			algoOrders, err = parseOkexAlgoOrder(sj.Get("data"))
			if err != nil {
				log.Warnf("parseOkexOrder error:%s, %s", orders, err.Error())
				continue
			}
			for _, v := range algoOrders {
				if v.State == "filled" {
					b.stopOrdersCache.Delete(v.AlgoID)
				}
			}
		case "algo-advance":
		case "positions":
			pos, err = parseOkexPos(sj.Get("data"))
			if err != nil {
				log.Warnf("parseOkexPos error:%s, %s", pos, err.Error())
				continue
			}
			for _, v := range pos {
				t := v.GetPos()
				if t == nil {
					continue
				}
				b.datas <- NewExchangeData(b.Name, EventPosition, t)
			}
		default:
		}
	}
}

type OrderNormal struct {
	AccFillSz       string `json:"accFillSz"`
	AmendResult     string `json:"amendResult"`
	AvgPx           string `json:"avgPx"`
	CTime           string `json:"cTime"`
	Category        string `json:"category"`
	Ccy             string `json:"ccy"`
	ClOrdID         string `json:"clOrdId"`
	Code            string `json:"code"`
	ExecType        string `json:"execType"`
	Fee             string `json:"fee"`
	FeeCcy          string `json:"feeCcy"`
	FillFee         string `json:"fillFee"`
	FillFeeCcy      string `json:"fillFeeCcy"`
	FillNotionalUsd string `json:"fillNotionalUsd"`
	FillPx          string `json:"fillPx"`
	FillSz          string `json:"fillSz"`
	FillTime        string `json:"fillTime"`
	InstID          string `json:"instId"`
	InstType        string `json:"instType"`
	Lever           string `json:"lever"`
	Msg             string `json:"msg"`
	NotionalUsd     string `json:"notionalUsd"`
	OrdID           string `json:"ordId"`
	OrdType         string `json:"ordType"`
	Pnl             string `json:"pnl"`
	PosSide         string `json:"posSide"`
	Px              string `json:"px"`
	Rebate          string `json:"rebate"`
	RebateCcy       string `json:"rebateCcy"`
	ReduceOnly      string `json:"reduceOnly"`
	ReqID           string `json:"reqId"`
	Side            string `json:"side"`
	SlOrdPx         string `json:"slOrdPx"`
	SlTriggerPx     string `json:"slTriggerPx"`
	SlTriggerPxType string `json:"slTriggerPxType"`
	Source          string `json:"source"`
	State           string `json:"state"`
	Sz              string `json:"sz"`
	Tag             string `json:"tag"`
	TdMode          string `json:"tdMode"`
	TgtCcy          string `json:"tgtCcy"`
	TpOrdPx         string `json:"tpOrdPx"`
	TpTriggerPx     string `json:"tpTriggerPx"`
	TpTriggerPxType string `json:"tpTriggerPxType"`
	TradeID         string `json:"tradeId"`
	UTime           string `json:"uTime"`
}

func (o *OrderNormal) GetOrder() (ret *Order) {
	if o.State != "filled" {
		return
	}
	var side = transSide(o.Side)
	ret = &Order{
		OrderID: o.OrdID,
		Symbol:  o.InstID,
		// Currency:o.
		Amount: parseFloat(o.Sz),
		Price:  parseFloat(o.AvgPx),
		Status: o.State,
		Side:   side,
		Time:   parseTime(o.FillTime),
	}
	return
}

type AlgoOrder struct {
	ActualPx    string `json:"actualPx"`
	ActualSide  string `json:"actualSide"`
	ActualSz    string `json:"actualSz"`
	AlgoID      string `json:"algoId"`
	CTime       string `json:"cTime"`
	Ccy         string `json:"ccy"`
	InstID      string `json:"instId"`
	InstType    string `json:"instType"`
	Lever       string `json:"lever"`
	NotionalUsd string `json:"notionalUsd"`
	OrdID       string `json:"ordId"`
	OrdPx       string `json:"ordPx"`
	OrdType     string `json:"ordType"`
	PosSide     string `json:"posSide"`
	ReduceOnly  string `json:"reduceOnly"`
	Side        string `json:"side"`
	SlOrdPx     string `json:"slOrdPx"`
	SlTriggerPx string `json:"slTriggerPx"`
	State       string `json:"state"`
	Sz          string `json:"sz"`
	TdMode      string `json:"tdMode"`
	TgtCcy      string `json:"tgtCcy"`
	TpOrdPx     string `json:"tpOrdPx"`
	TpTriggerPx string `json:"tpTriggerPx"`
	TriggerPx   string `json:"triggerPx"`
	TriggerTime string `json:"triggerTime"`
}

func transSide(oSide string) (side string) {
	switch oSide {
	case "buy":
		side = "long"
	case "sell":
		side = "short"
	default:
		side = oSide
	}
	return
}

func parseTime(v string) time.Time {
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		logrus.Errorf("parseTime failed: %s", err.Error())
		return time.Now()
	}
	return time.Unix(n/1000, (n%1000)*int64(time.Millisecond))
}

// {"accFillSz":"0","amendResult":"","avgPx":"0","cTime":"1639669567849","category":"normal","ccy":"","clOrdId":"","code":"0","execType":"","fee":"0","feeCcy":"USDT","fillFee":"0","fillFeeCcy":"","fillNotionalUsd":"","fillPx":"","fillSz":"0","fillTime":"","instId":"SOL-USDT-SWAP","instType":"SWAP","lever":"10","msg":"","notionalUsd":"185.666","ordId":"391737792421838866","ordType":"limit","pnl":"0","posSide":"short","px":"200","rebate":"0","rebateCcy":"USDT","reduceOnly":"false","reqId":"","side":"sell","slOrdPx":"","slTriggerPx":"","slTriggerPxType":"last","source":"","state":"live","sz":"1","tag":"","tdMode":"isolated","tgtCcy":"","tpOrdPx":"","tpTriggerPx":"","tpTriggerPxType":"last","tradeId":"","uTime":"1639669567849"}
func parseOkexOrder(sj *simplejson.Json) (orders []OrderNormal, err error) {
	arr, err := sj.Array()
	if err != nil {
		return
	}
	var ret map[string]interface{}
	var ok bool
	for _, v := range arr {
		ret, ok = v.(map[string]interface{})
		if !ok {
			err = errors.New("parseOrderbook error")
			return
		}
		var o OrderNormal
		err = mapstructure.Decode(ret, &o)
		if err != nil {
			return
		}
		orders = append(orders, o)
	}
	return
}

func parseOkexPos(sj *simplejson.Json) (trades []OKEXPos, err error) {
	arr, err := sj.Array()
	if err != nil {
		return
	}
	var ret map[string]interface{}
	var ok bool
	for _, v := range arr {
		ret, ok = v.(map[string]interface{})
		if !ok {
			err = errors.New("parseOkexTrades error")
			return
		}
		var t OKEXPos
		err = mapstructure.Decode(ret, &t)
		if err != nil {
			return
		}
		trades = append(trades, t)
	}
	return
}

type OKEXPos struct {
	Adl         string `json:"adl"`
	AvailPos    string `json:"availPos"`
	AvgPx       string `json:"avgPx"`
	CTime       string `json:"cTime"`
	Ccy         string `json:"ccy"`
	DeltaBS     string `json:"deltaBS"`
	DeltaPA     string `json:"deltaPA"`
	GammaBS     string `json:"gammaBS"`
	GammaPA     string `json:"gammaPA"`
	Imr         string `json:"imr"`
	InstID      string `json:"instId"`
	InstType    string `json:"instType"`
	Interest    string `json:"interest"`
	Last        string `json:"last"`
	Lever       string `json:"lever"`
	Liab        string `json:"liab"`
	LiabCcy     string `json:"liabCcy"`
	LiqPx       string `json:"liqPx"`
	MarkPx      string `json:"markPx"`
	Margin      string `json:"margin"`
	MgnMode     string `json:"mgnMode"`
	MgnRatio    string `json:"mgnRatio"`
	Mmr         string `json:"mmr"`
	NotionalUsd string `json:"notionalUsd"`
	OptVal      string `json:"optVal"`
	PTime       string `json:"pTime"`
	Pos         string `json:"pos"`
	PosCcy      string `json:"posCcy"`
	PosID       string `json:"posId"`
	PosSide     string `json:"posSide"`
	ThetaBS     string `json:"thetaBS"`
	ThetaPA     string `json:"thetaPA"`
	TradeID     string `json:"tradeId"`
	UTime       string `json:"uTime"`
	Upl         string `json:"upl"`
	UplRatio    string `json:"uplRatio"`
	VegaBS      string `json:"vegaBS"`
	VegaPA      string `json:"vegaPA"`
}

func (ot *OKEXPos) GetPos() (pos *Position) {
	var typ int
	if ot.PosSide == "long" {
		typ = Long
	} else {
		typ = Short
	}
	pos = &Position{
		Symbol:      ot.InstID,
		Type:        typ,
		Hold:        parseFloat(ot.Pos),
		ProfitRatio: parseFloat(ot.Upl),
	}
	return
}

func parseOkexAlgoOrder(sj *simplejson.Json) (orders []AlgoOrder, err error) {
	arr, err := sj.Array()
	if err != nil {
		return
	}
	var ret map[string]interface{}
	var ok bool
	for _, v := range arr {
		ret, ok = v.(map[string]interface{})
		if !ok {
			err = errors.New("parseOrderbook error")
			return
		}
		var o AlgoOrder
		err = mapstructure.Decode(ret, &o)
		if err != nil {
			return
		}
		orders = append(orders, o)
	}
	return
}
