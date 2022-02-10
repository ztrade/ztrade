package ok

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/bitly/go-simplejson"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	. "github.com/ztrade/trademodel"
	. "github.com/ztrade/ztrade/pkg/core"
)

func (b *OkexTrade) runPublic() (err error) {
	u, err := url.Parse(WSOkexPUbilc)
	if err != nil {
		return
	}

	b.wsPublic, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial public:", err)
		return
	}
	// var p = OPParam{
	// 	OP: "subscribe",
	// 	Args: []interface{}{
	// 		OPArg{Channel: "candle1m", InstType: "SWAP", InstID: b.symbol},
	// 		// OPArg{Channel: "trades", InstType: "SWAP", InstID: b.symbol},
	// 		// OPArg{Channel: "books5", InstType: "SWAP", InstID: b.symbol},
	// 	},
	// }
	// err = b.wsPublic.WriteJSON(p)
	// if err != nil {
	// 	return
	// }
	go b.runPublicLoop(b.wsPublic, b.closeCh)
	return
}

func (b *OkexTrade) runPublicLoop(c *websocket.Conn, closeCh chan bool) {
	var err error
	var sj *simplejson.Json
	var message []byte
	var channel string
	var depths5 []*Depth

	var trades []*Trade
	var candles []*Candle
	var prev *Candle
	var lastMsgTime time.Time
	loopEnd := make(chan bool)
	defer func() {
		close(loopEnd)
		err = b.runPublic()
		if err != nil {
			log.Error("okex reconnect public failed:", err.Error())
		}
	}()
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
			case <-loopEnd:
				return
			case <-closeCh:
				return
			}
		}
	}()
	log.Info("okex runPublic")
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
		case "books5":
			value := sj.Get("data")
			if value == nil {
				continue
			}
			depths5, err = parseOkexBooks5(value)
			if err != nil {
				log.Warnf("parseOkexBooks5 failed:%s, %s", string(message), err.Error())
				continue
			}
			for _, v := range depths5 {
				b.datas <- NewExchangeData(b.Name, EventDepth, v)
			}

		case "trades":
			value := sj.Get("data")
			if value == nil {
				continue
			}
			trades, err = parseOkexTrade(value)
			if err != nil {
				log.Warnf("parseOkexTrade failed:%s, %s", string(message), err.Error())
				continue
			}
			for _, v := range trades {
				b.datas <- NewExchangeData(b.Name, EventTradeMarket, v)
			}
		case "candle1m":
			value := sj.Get("data")
			if value == nil {
				continue
			}

			candles, err = parseWsCandle(value)
			if err != nil {
				log.Warnf("parseWsCandle failed:%s, %s", string(message), err.Error())
				continue
			}
			if len(candles) != 1 {
				log.Warnf("parseWsCandle candles len warn :%d, %s", len(candles), err.Error())
			}
			if prev == nil || prev.Start == candles[0].Start {
				prev = candles[0]
				continue
			}
			b.datas <- NewExchangeData(NewCandleName("candle", "1m").String(), EventCandle, prev)
			prev = candles[0]
		default:
		}
	}
}

// books5 {"arg":{"channel":"books5","instId":"BSV-USD-210924"},"data":[{"asks":[["166.35","20","0","1"],["166.4","135","0","2"],["166.42","86","0","1"],["166.45","310","0","1"],["166.46","61","0","2"]],"bids":[["166.14","33","0","1"],["166.07","106","0","1"],["166.05","2","0","1"],["166.04","97","0","1"],["165.98","20","0","1"]],"instId":"BSV-USD-210924","ts":"1621862688397"}]}
func parseOkexBooks5(sj *simplejson.Json) (depths []*Depth, err error) {
	arr, err := sj.Array()
	if err != nil {
		err = fmt.Errorf("parseOkexBooks5 not array: %w", err)
		return
	}

	var obj, asks, bids *simplejson.Json
	var askInfo, bidInfo []DepthInfo
	var tsStr string
	var nTs int64
	for k := range arr {
		obj = sj.GetIndex(k)
		asks = obj.Get("asks")
		bids = obj.Get("bids")
		if asks == nil || bids == nil {
			err = errors.New("data error")
			return
		}
		askInfo, err = parseOrderbook(asks)
		if err != nil {
			return
		}
		bidInfo, err = parseOrderbook(bids)
		if err != nil {
			return
		}
		tsStr, err = obj.Get("ts").String()
		if err != nil {
			return
		}
		dp := Depth{Buys: bidInfo, Sells: askInfo}
		nTs, err = strconv.ParseInt(tsStr, 10, 64)
		if err != nil {
			return
		}
		dp.UpdateTime = time.Unix(nTs/1000, (nTs%1000)*int64(time.Millisecond))
		depths = append(depths, &dp)
	}
	return
}

func parseOrderbook(sj *simplejson.Json) (datas []DepthInfo, err error) {
	arr, err := sj.Array()
	if err != nil {
		return
	}
	var ret []interface{}
	var ok bool
	for _, v := range arr {
		ret, ok = v.([]interface{})
		if !ok {
			err = errors.New("parseOrderbook error")
			return
		}
		if len(ret) != 4 {
			err = errors.New("parseOrderbook len error")
			return
		}
		di := DepthInfo{Price: getFloat(ret[0]), Amount: getFloat(ret[1])}
		datas = append(datas, di)
	}
	return
}

func parseOkexTrade(sj *simplejson.Json) (trades []*Trade, err error) {
	arr, err := sj.Array()
	if err != nil {
		return
	}
	// {"instId":"BSV-USD-210924","tradeId":"1957771","px":"166.5","sz":"22","side":"sell","ts":"1621862533713"}
	var temp map[string]interface{}
	var ok bool
	for _, v := range arr {
		temp, ok = v.(map[string]interface{})
		if !ok {
			log.Warnf("data error:%#v", temp)
			continue
		}
		var t Trade
		t.Remark = getStr(temp, "instId")
		t.ID = getStr(temp, "tradeId")
		t.Side = getStr(temp, "side")
		t.Price = getStrFloat(temp, "px")
		t.Amount = getStrFloat(temp, "sz")
		nTs := getStrTs(temp, "ts")
		t.Time = time.Unix(nTs/1000, int64(time.Millisecond)*(nTs%1000))
		trades = append(trades, &t)
	}
	return
}

func getStrTs(dict map[string]interface{}, key string) (nTs int64) {
	str := getStr(dict, key)
	if str == "" {
		log.Warnf("getStrTs of %s empty", key)
		return
	}
	nTs, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		log.Warnf("getStrTs of %s parse int err: %s", key, err.Error())
		return
	}
	return nTs
	// t = time.Unix(nTs/1000, (nTs%1000)*int64(time.Millisecond))
	return
}

func getStrFloat(dict map[string]interface{}, key string) float64 {
	str := getStr(dict, key)
	if str == "" {
		log.Warnf("getStrFloat of %s empty", key)
		return 0
	}
	f, err := strconv.ParseFloat(str, 64)
	if err != nil {
		log.Warnf("getStrFloat of %s failed: %s", key, err.Error())
		return 0
	}
	return f
}

func getStr(dict map[string]interface{}, key string) string {
	v, ok := dict[key]
	if !ok {
		log.Warnf("getStr of %s empty", key)
		return ""
	}
	str, ok := v.(string)
	if !ok {
		log.Warnf("getStr of %s type error: %#v", key, v)
		return ""
	}
	return str
}

func getFloat(v interface{}) float64 {
	str, ok := v.(string)
	if !ok {
		return 0
	}
	f, err := strconv.ParseFloat(str, 64)
	if err != nil {
		log.Warnf("getFloat %#v failed: %s", v, err.Error())
		return 0
	}
	return f
}

func parseWsCandle(sj *simplejson.Json) (ret []*Candle, err error) {
	arr, err := sj.Array()
	if err != nil {
		return
	}
	// {"instId":"BSV-USD-210924","tradeId":"1957771","px":"166.5","sz":"22","side":"sell","ts":"1621862533713"}
	var values []interface{}
	var ok bool
	var nTs int64
	for _, v := range arr {
		values, ok = v.([]interface{})
		if !ok {
			log.Warnf("transWsCandle data error:%#v", values)
			continue
		}
		if len(values) != 7 {
			log.Warnf("transWsCandle data len error:%#v", values)
			continue
		}
		nTs, err = strconv.ParseInt(values[0].(string), 10, 64)
		if err != nil {
			panic(fmt.Sprintf("trans candle error: %#v", values))
			return
		}
		temp := Candle{
			ID:     0,
			Start:  nTs / 1000,
			Open:   parseFloat(values[1].(string)),
			High:   parseFloat(values[2].(string)),
			Low:    parseFloat(values[3].(string)),
			Close:  parseFloat(values[4].(string)),
			Volume: parseFloat(values[5].(string)),
			VWP:    parseFloat(values[6].(string)),
		}
		ret = append(ret, &temp)
	}
	return
}
