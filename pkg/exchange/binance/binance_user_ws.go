package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	. "github.com/ztrade/trademodel"
	. "github.com/ztrade/ztrade/pkg/core"
)

var (
	wsURL = "wss://fstream.binance.com/ws"
)

type accountInfo struct {
	Reason  string `json:"m,omitempty"` // 事件推出原因
	Balance []struct {
		Name    string `json:"a"`  // 资产名称
		Balance string `json:"wb"` // 钱包余额
		Availa  string `json:"cw"` // 除去逐仓仓位保证金的钱包余额
	} `json:"B,omitempty"` // B 余额信息
	Positions []struct {
		Symbol     string `json:"s"`  // 交易对
		Amount     string `json:"pa"` // 仓位
		OpenPrice  string `json:"ep"` // 入仓价格
		ProfitDone string `json:"cr"` // (费前)累计实现损益
		Profit     string `json:"up"` // 持仓未实现盈亏
		Mode       string `json:"mt"` // 保证金模式
		Margin     string `json:"iw"` // 若为逐仓，仓位保证金
		Side       string `json:"ps"` // 持仓方向
	} `json:"p"`
}

//   // 特殊的自定义订单ID:
//   // "autoclose-"开头的字符串: 系统强平订单
//   // "adl_autoclose": ADL自动减仓订单

type orderInfo struct {
	Symbol         string `json:"s"`  //  "s":"BTCUSDT",                  // 交易对
	Client         string `json:"c"`  // "c":"TEST",                     // 客户端自定订单ID
	Side           string `json:"S"`  // "S":"SELL",                     // 订单方向
	OrderType      string `json:"o"`  // "o":"LIMIT",                    // 订单类型
	OrderOption    string `json:"f"`  // "f":"GTC",                      // 有效方式
	Amount         string `json:"q"`  // "q":"0.001",                    // 订单原始数量
	Price          string `json:"p"`  // "p":"9910",                     // 订单原始价格
	AvgPrice       string `json:"ap"` // "ap":"0",                       // 订单平均价格
	StopPrice      string `json:"sp"` // "sp":"0",                       // 订单停止价格
	ExecType       string `json:"x"`  // "x":"NEW",                      // 本次事件的具体执行类型
	Status         string `json:"X"`  // "X":"NEW",                      // 订单的当前状态
	ID             int64  `json:"i"`  // "i":8886774,                    // 订单ID
	LastFill       string `json:"l"`  // "l":"0",                        // 订单末次成交数量
	TotalFill      string `json:"z"`  // "z":"0",                        // 订单累计已成交数量
	LastFillPrice  string `json:"L"`  // "L":"0",                        // 订单末次成交价格
	FeeSymbol      string `json:"N"`  // "N": "USDT",                    // 手续费资产类型
	FeeAmount      string `json:"n"`  // "n": "0",                       // 手续费数量
	TradeTime      int64  `json:"T"`  // "T":1568879465651,              // 成交时间
	TradeID        int64  `json:"t"`  // "t":0,                          // 成交ID
	Buy            string `json:"b"`  // "b":"0",                        // 买单净值
	Sell           string `json:"a"`  // "a":"9.91"                      // 卖单净值
	IsMaker        bool   `json:"m" ` // "m": false,                     // 该成交是作为挂单成交吗？
	IsReduce       bool   `json:"R"`  // "R":false,                      // 是否是只减仓单
	TriggerType    string `json:"wt"` // "wt": "CONTRACT_PRICE",         // 触发价类型
	IsTriggerClose bool   `json:"cp"` // "cp":false,                     // 是否为触发平仓单; 仅在条件订单情况下会推送此字段
	TraceStopPrice string `json:"AP"` // "AP":"7476.89",                 // 追踪止损激活价格, 仅在追踪止损单时会推送此字段
	TraceStopRate  string `json:"cr"` // "cr":"5.0"                      // 追踪止损回调比例, 仅在追踪止损单时会推送此字段
}

type oneLevel [2]string
type wsUserResp struct {
	// orderbook
	Name      string      `json:"e, omitempty"`
	EventTime int64       `json:"E, omitempty"`
	TradeTime int64       `json:"T, omitempty"`
	O         orderInfo   `json:"o, omitempty"`
	A         accountInfo `json:"a, omitempty"`
}

func (b *BinanceTrade) updateUserListenKey() {
	ctx := context.Background()
	var listenKey string
	var err error
	ticker := time.NewTicker(time.Minute * 30)
	for {
		select {
		case <-b.closeCh:
			break
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
				break
			}
		default:
		}
	}
}

func (b *BinanceTrade) startUserWS() (err error) {
	ctx := context.Background()
	listenKey, err := b.api.NewStartUserStreamService().Do(ctx)
	if err != nil {
		return
	}
	b.wsUserListenKey = listenKey
	userInfoURL := fmt.Sprintf("%s/%s", wsURL, b.wsUserListenKey)
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
	go b.updateUserListenKey()
	go b.wsUserLoop()
	return
}

func (b *BinanceTrade) wsUserLoop() {
	var resp wsUserResp
	var order Order
	var pos Position
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
			log.Println("read:", err)
			return
		}
		fmt.Println(string(message))
		err = json.Unmarshal(message, &resp)
		if err != nil {
			log.Errorf("unmarshal error:%s message:%s", err.Error(), string(message))
			continue
		}
		switch resp.Name {
		case "ORDER_TRADE_UPDATE":
			order.OrderID = strconv.FormatInt(resp.O.ID, 10)
			order.Symbol = resp.O.Symbol
			order.Currency = resp.O.Symbol
			order.Amount, _ = strconv.ParseFloat(resp.O.Amount, 64)
			order.Price, _ = strconv.ParseFloat(resp.O.Price, 64)
			order.Status = resp.O.Status
			order.Side = resp.O.Side
			order.Time = time.Unix(0, resp.TradeTime*int64(time.Millisecond))
			b.datas <- &ExchangeData{Name: EventOrder, Data: &order}
		case "ACCOUNT_UPDATE":
			var profit, total float64
			var balance Balance
			for _, v := range resp.A.Balance {
				if v.Name == "USDT" {
					balance.Balance, _ = strconv.ParseFloat(v.Balance, 64)
					balance.Available, _ = strconv.ParseFloat(v.Availa, 64)
					b.datas <- &ExchangeData{Name: EventBalance, Data: &balance}
					total = balance.Balance
				}
			}
			var side string
			for _, v := range resp.A.Positions {
				pos.Symbol = v.Symbol
				profit, _ = strconv.ParseFloat(v.Profit, 64)
				if v.Margin != "" {
					total, _ = strconv.ParseFloat(v.Margin, 64)
				}
				if total > 0 {
					pos.ProfitRatio = profit / total
				}
				pos.Price, _ = strconv.ParseFloat(v.OpenPrice, 64)
				pos.Hold, _ = strconv.ParseFloat(v.Amount, 64)
				side = strings.ToLower(v.Side)
				switch side {
				case "long":
					pos.Type = Long
				case "short":
					pos.Type = Short
				}
				b.datas <- &ExchangeData{Name: EventPosition, Data: &pos}
			}
		default:
			continue
		}

		// log.Printf("%##v", order)
	}
}
