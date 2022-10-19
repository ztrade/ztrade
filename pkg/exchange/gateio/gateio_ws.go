package gateio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/ztrade/trademodel"
	. "github.com/ztrade/ztrade/pkg/core"
)

func (g *GateIO) parseDepth(message []byte) (err error) {
	var ob GateFuturesOB
	err = json.Unmarshal(message, &ob)
	if err != nil {
		if bytes.Contains(message, []byte("subscribe")) {
			fmt.Println("got subscribe return:", string(message))
			err = nil
		}
		return
	}
	depth := transDepth(&ob)
	temp := NewExchangeData(g.Name, EventDepth, &depth)
	temp.Symbol = ob.Contract
	g.datas <- temp
	return nil
}

func (g *GateIO) parseMarketTrade(message []byte) (err error) {
	var t GateFuturesTrade
	err = json.Unmarshal(message, &t)
	if err != nil {
		if bytes.Contains(message, []byte("subscribe")) {
			fmt.Println("got subscribe return:", string(message))
			err = nil
		}
		return
	}
	trades := transTrade(&t)
	for _, v := range trades {
		tempTrade := v
		temp := NewExchangeData(g.Name, EventTradeMarket, &tempTrade)
		temp.Symbol = tempTrade.Remark
		g.datas <- temp
	}
	return nil
}

type OneLevel struct {
	Price  string  `json:"p"`
	Amount float64 `json:"s"`
}

type GateFuturesOB struct {
	T        int64      `json:"t"`
	ID       int        `json:"id"`
	Contract string     `json:"contract"`
	Asks     []OneLevel `json:"asks"`
	Bids     []OneLevel `json:"bids"`
}

type GateFuturesTrade struct {
	Time    int    `json:"time"`
	Channel string `json:"channel"`
	Event   string `json:"event"`
	Result  []struct {
		ID           int    `json:"id"`
		Size         int    `json:"size"`
		CreateTime   int    `json:"create_time"`
		CreateTimeMs int64  `json:"create_time_ms"`
		Price        string `json:"price"`
		Contract     string `json:"contract"`
	} `json:"result"`
}

func transDepth(ob *GateFuturesOB) (dep trademodel.Depth) {
	dep.UpdateTime = time.UnixMilli(ob.T)

	dep.Buys = make([]trademodel.DepthInfo, len(ob.Bids))
	dep.Sells = make([]trademodel.DepthInfo, len(ob.Asks))
	var price float64
	var err error
	for k, v := range ob.Asks {
		price, err = strconv.ParseFloat(v.Price, 64)
		if err != nil {
			logrus.Errorf("GateIO transDepth Asks failed: %s", v.Price)
			continue
		}
		dep.Sells[k] = trademodel.DepthInfo{
			Price:  price,
			Amount: v.Amount,
		}
	}
	for k, v := range ob.Bids {
		price, err = strconv.ParseFloat(v.Price, 64)
		if err != nil {
			logrus.Errorf("GateIO transDepth bids failed: %s", v.Price)
			continue
		}
		dep.Buys[k] = trademodel.DepthInfo{
			Price:  price,
			Amount: v.Amount,
		}
	}
	return
}

func transTrade(t *GateFuturesTrade) (trades []trademodel.Trade) {
	trades = make([]trademodel.Trade, len(t.Result))
	var err error
	for k, v := range t.Result {
		trades[k].Amount = math.Abs(float64(v.Size))
		trades[k].Price, err = strconv.ParseFloat(v.Price, 64)
		if err != nil {
			logrus.Errorf("GateIO transTrade failed: %s", err.Error())
		}
		trades[k].ID = fmt.Sprintf("%d", v.ID)
		if v.Size > 0 {
			trades[k].Side = "buy"
		} else {
			trades[k].Side = "sell"
		}
		trades[k].Remark = v.Contract
		trades[k].Time = time.UnixMilli(v.CreateTimeMs)
	}
	return
}
