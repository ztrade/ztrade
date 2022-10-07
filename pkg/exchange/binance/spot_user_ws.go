package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	. "github.com/ztrade/trademodel"
	. "github.com/ztrade/ztrade/pkg/core"
)

var (
	wsSpotURL = "wss://stream.binance.com:9443"
)

func (b *BinanceSpot) updateUserListenKey() {
	ctx := context.Background()
	var listenKey string
	var err error
	ticker := time.NewTicker(time.Minute * 30)
Out:
	for {
		select {
		case <-b.closeCh:
			break Out
		case <-ticker.C:
			for i := 0; i < 10; i++ {
				listenKey, err = b.api.NewStartUserStreamService().Do(ctx)
				if err != nil {
					log.Error("update listen key failed:", err.Error())
					continue
				}
				if listenKey != b.wsUserListenKey {
					log.Info("listen key reset:", listenKey)
					b.wsUserListenKey = listenKey
				}
				break
			}
			if err != nil {
				break Out
			}
		}
	}
}

func (b *BinanceSpot) startUserWS() (err error) {
	ctx := context.Background()
	listenKey, err := b.api.NewStartUserStreamService().Do(ctx)
	if err != nil {
		return
	}
	b.wsUserListenKey = listenKey
	userInfoURL := fmt.Sprintf("%s/ws/%s", wsSpotURL, b.wsUserListenKey)
	u, err := url.Parse(userInfoURL)
	if err != nil {
		log.Error("parse userInfoURL error:", err.Error())
		return
	}
	log.Printf("connecting user info to %s", u.String())

	b.wsUser, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		err = fmt.Errorf("connect to user info error: %w", err)
		return
	}
	log.Printf("connect success %s", u.String())
	go b.updateUserListenKey()
	go b.wsUserLoop()
	return
}

type spotBalance struct {
	Symbol    string `json:"a,omitempty"`
	Available string `json:"f,omitempty"`
	Freeze    string `json:"l,omitempty"`
}

type wsSpotUserResp struct {
	// orderbook
	Name           string `json:"e,omitempty"`
	EventTime      int64  `json:"E,omitempty"`
	LastUpdateTime int64  `json:"u,omitempty"`

	Balances []spotBalance `json:"B,omitempty"`
}

func (b *BinanceSpot) wsUserLoop() {
	var resp wsSpotUserResp
	// var order Order
	// var pos Position
	// type Position struct {
	// 	Symbol      string
	// 	Type        int     // 合约类型，Long: 多头，Short: 空头
	// 	Hold        float64 // 持有仓位
	// 	Price       float64 //开仓价格
	// 	ProfitRatio float64 // 盈利比例,正数表示盈利，负数表示亏岁
	// }

	for {
		_, message, err := b.wsUser.ReadMessage()
		if err != nil {
			log.Println("read error:", err)
			return
		}
		err = json.Unmarshal(message, &resp)
		if err != nil {
			log.Errorf("unmarshal error:%s message:%s", err.Error(), string(message))
			continue
		}
		if len(resp.Balances) > 0 {
			var balance Balance
			for _, v := range resp.Balances {
				if v.Symbol == "USDT" {
					balance.Balance, _ = strconv.ParseFloat(v.Available, 64)
					balance.Available, _ = strconv.ParseFloat(v.Available, 64)
					d := NewExchangeData(b.Name, EventBalance, &balance)
					d.Symbol = "BTCUSDT"
					b.datas <- d
				}
				if v.Symbol == "BTC" {
					var pos Position
					pos.Hold, _ = strconv.ParseFloat(v.Available, 64)
					d := NewExchangeData(b.Name, EventPosition, &pos)
					d.Symbol = "BTCUSDT"
					b.datas <- d
				}
			}
		}

	}
}
