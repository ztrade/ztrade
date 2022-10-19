package binance

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/ztrade/trademodel"
	. "github.com/ztrade/ztrade/pkg/core"
	"github.com/ztrade/ztrade/pkg/exchange/ws"
)

type OneLevel [2]string

type BinanceOrderbookSpot struct {
	LastUpdateID int        `json:"lastUpdateId"`
	Buys         []OneLevel `json:"bids"`
	Sells        []OneLevel `json:"asks"`
}

type BinanceSpotTrade struct {
	Name      string `json:"e"`
	EventTime int64  `json:"E"`
	TradeTime int64  `json:"T"`
	Symbol    string `json:"s"`
	TradeID   int64  `json:"t"`
	Price     string `json:"p"`
	Amount    string `json:"q"`
	BuyID     int64  `json:"b"`
	SellID    int64  `json:"a"`
	IsSell    bool   `json:"m"`
}

func (b *BinanceSpot) parseBinanceSpotDepth(symbol string) ws.MessageFn {
	return func(message []byte) error {
		var ob BinanceOrderbookSpot
		err := json.Unmarshal([]byte(message), &ob)
		if err != nil {
			logrus.Error("binance_spot depth json unmarshal error:", err.Error())
			return err
		}
		depth := transBinanceSpotOB(&ob)
		temp := NewExchangeData(b.Name, EventDepth, &depth)
		temp.Symbol = symbol
		b.datas <- temp
		return nil
	}
}

func (b *BinanceSpot) parseBinanceSpotMarketTrade(symbol string) ws.MessageFn {
	return func(message []byte) error {
		var t BinanceSpotTrade
		err := json.Unmarshal([]byte(message), &t)
		if err != nil {
			logrus.Error("binance_spot markettrade json unmarshal error:", err.Error())
			return err
		}
		td := transBinanceSpotTrade(&t)
		temp := NewExchangeData(b.Name, EventTradeMarket, &td)
		temp.Symbol = symbol
		b.datas <- temp
		return nil
	}
}

func transBinanceSpotOB(o *BinanceOrderbookSpot) (b trademodel.Depth) {
	b.UpdateTime = time.Now()
	b.Buys = make([]trademodel.DepthInfo, len(o.Buys))
	b.Sells = make([]trademodel.DepthInfo, len(o.Sells))
	for k, v := range o.Buys {
		b.Buys[k].Price, _ = strconv.ParseFloat(v[0], 64)
		b.Buys[k].Amount, _ = strconv.ParseFloat(v[1], 64)
	}
	for k, v := range o.Sells {
		b.Sells[k].Price, _ = strconv.ParseFloat(v[0], 64)
		b.Sells[k].Amount, _ = strconv.ParseFloat(v[1], 64)
	}
	return
}

func transBinanceSpotTrade(t *BinanceSpotTrade) (b trademodel.Trade) {
	b.ID = fmt.Sprintf("%d", t.TradeID)
	b.Remark = t.Symbol
	if t.IsSell {
		b.Side = "sell"
	} else {
		b.Side = "buy"
	}
	b.Price, _ = strconv.ParseFloat(t.Price, 64)
	b.Amount, _ = strconv.ParseFloat(t.Amount, 64)
	b.Time = time.UnixMilli(t.TradeTime)
	return
}
