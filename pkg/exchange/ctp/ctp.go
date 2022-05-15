package ctp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/ztrade/base/common"
	"github.com/ztrade/ctp"
	. "github.com/ztrade/trademodel"
	"github.com/ztrade/ztrade/pkg/core"
)

var (
	reqID uint64
)

func getReqID() int {
	n := atomic.AddUint64(&reqID, 1)
	return int(n)
}

type Config struct {
	TdServer string
	MdServer string
	BrokerID string
	User     string
	Password string
	AppID    string
	AuthCode string
}

type CtpExchange struct {
	name          string
	mdApi         *ctp.CThostFtdcMdApi
	mdSpi         *MdSpi
	tdApi         *ctp.CThostFtdcTraderApi
	tdSpi         *TdSpi
	cfg           *Config
	datas         chan *core.ExchangeData
	prevVolume    float64
	orderID       uint64
	orders        map[string]*Order
	inited        uint32
	stopChan      chan bool
	strStart      string
	startOnce     sync.Once
	positionReqID int
	positionCache []Position
}

func NewCtp(cfg *viper.Viper, cltName string) (e core.Exchange, err error) {
	b, err := NewCtpExchange(cfg, cltName)
	if err != nil {
		return
	}
	e = b
	return
}

func init() {
	core.RegisterExchange("ctp", NewCtp)
}

func parseSymbol(str string) (exchange, symbol string, err error) {
	strs := strings.Split(str, ".")
	if len(strs) != 2 {
		err = errors.New("symbol format error {EXCHANGE}.{SYMBOL}")
		return
	}
	exchange, symbol = strs[0], strs[1]
	return
}

func formatSymbol(exchange, symbol string) string {
	return fmt.Sprintf("%s.%s", exchange, symbol)
}

func NewCtpExchange(cfg *viper.Viper, cltName string) (c *CtpExchange, err error) {
	var ctpConfig Config
	err = cfg.UnmarshalKey("exchanges.ctp", &ctpConfig)
	if err != nil {
		return
	}
	c = &CtpExchange{name: cltName, cfg: &ctpConfig}
	t := time.Now()
	c.strStart = t.Format("01021504")
	c.stopChan = make(chan bool)
	c.datas = make(chan *core.ExchangeData, 1024)
	c.orders = make(map[string]*Order)
	err = c.initConn()
	if err != nil {
		return
	}
	c.Start(map[string]interface{}{})
	return
}

func (c *CtpExchange) HasInit() bool {
	v := atomic.LoadUint32(&c.inited)
	return v == 1
}

func (c *CtpExchange) SetInit(bInit bool) {
	if bInit {
		atomic.StoreUint32(&c.inited, 1)
	} else {
		atomic.StoreUint32(&c.inited, 0)
	}
}

func (c *CtpExchange) reConnectTdApi() {
	var err error
Out:
	for {
		c.tdSpi.WaiDisconnect(c.stopChan)
		select {
		case <-c.stopChan:
			break Out
		default:
		}
		if !c.HasInit() {
			time.Sleep(time.Second * 10)
			continue
		}

		if c.tdApi != nil {
			// c.tdApi.Join()
			c.tdApi.Release()
			c.tdApi = nil
		}
		err = c.initTdApi()
		if err != nil {
			logrus.Errorf("initTdApi failed: %s", err.Error())
		}
	}
}

func (c *CtpExchange) reConnectMdApi() {
	var err error
Out:
	for {
		c.mdSpi.WaiDisconnect(c.stopChan)
		select {
		case <-c.stopChan:
			break Out
		default:
		}
		if !c.HasInit() {
			time.Sleep(time.Second * 10)
			continue
		}
		if c.mdApi != nil {
			// c.mdApi.Join()
			c.mdApi.Release()
			c.mdApi = nil
		}
		err = c.initMdApi()
		if err != nil {
			logrus.Errorf("initMdApi failed: %s", err.Error())
		}
	}
}

func (c *CtpExchange) initConn() (err error) {
	err = c.initTdApi()
	if err != nil {
		err = fmt.Errorf("initTdApi failed: %w", err)
		return
	}
	err = c.initMdApi()
	if err != nil {
		err = fmt.Errorf("initMdApi failed: %w", err)
		return
	}
	c.SetInit(true)
	return
}

func (c *CtpExchange) initMdApi() (err error) {
	WaitTradeTime()
	c.mdApi = ctp.MdCreateFtdcMdApi("./ctp/md", false, false)
	c.mdApi.RegisterFront(fmt.Sprintf("tcp://%s", c.cfg.MdServer))
	c.mdSpi, err = NewMdSpi(c, c.cfg, c.mdApi)
	if err != nil {
		return
	}
	c.mdApi.RegisterSpi(c.mdSpi)
	c.mdApi.Init()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	err = c.mdSpi.WaitLogin(ctx)
	cancel()
	if err != nil {
		return
	}
	// TODO: default watch all
	// c.mdApi.SubscribeMarketData([]string{c.symbol})
	return
}

func (c *CtpExchange) initTdApi() (err error) {
	WaitTradeTime()
	err = os.MkdirAll("./ctp", os.ModePerm)
	if err != nil {
		return
	}
	tdApi := ctp.TdCreateFtdcTraderApi("./ctp/td")
	tdApi.SubscribePrivateTopic(ctp.THOST_TERT_QUICK)
	tdApi.SubscribePublicTopic(ctp.THOST_TERT_QUICK)
	tdSpi := NewTdSpi(c, c.cfg, tdApi)
	c.tdSpi = tdSpi

	c.tdApi = tdApi
	tdApi.RegisterSpi(tdSpi)
	tdApi.RegisterFront(fmt.Sprintf("tcp://%s", c.cfg.TdServer))
	tdApi.Init()
	time.Sleep(time.Second * 10)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	logrus.Println("wait TdApi login")
	err = tdSpi.WaitLogin(ctx)
	if err != nil {
		logrus.Errorf("login error: %s", err.Error())
		return
	}
	logrus.Println("TdApi login success")

	return
}

func (c *CtpExchange) Start(map[string]interface{}) error {
	c.startOnce.Do(func() {
		go c.syncPosition()
		go c.reConnectMdApi()
		go c.reConnectTdApi()
	})
	return nil
}

func (c *CtpExchange) Stop() error {
	close(c.stopChan)
	return nil
}

func (c *CtpExchange) syncPosition() (err error) {
	tick := time.NewTicker(time.Second * 5)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			if !c.HasInit() {
				continue
			}
			reqID := getReqID()
			pQryInvestorPosition := &ctp.CThostFtdcQryInvestorPositionDetailField{}
			nRet := c.tdApi.ReqQryInvestorPositionDetail(pQryInvestorPosition, reqID)
			if nRet != 0 {
				logrus.Errorf("ReqQryInvestorPositionDetail failed: %d", nRet)
			} else {
				c.positionReqID = reqID
			}
		case <-c.stopChan:
			return
		}
	}
	return
}

// Kline get klines
func (c *CtpExchange) GetKline(symbol, bSize string, start, end time.Time) (data chan *Candle, err chan error) {
	return
}

func (c *CtpExchange) Watch(core.WatchParam) error {
	return nil
}

func (c *CtpExchange) CancelOrder(old *Order) (order *Order, err error) {
	return
}

// for trade
// ProcessOrder process order
func (c *CtpExchange) ProcessOrder(act TradeAction) (ret *Order, err error) {
	exchangeID, symbol, err := parseSymbol(act.Symbol)
	if err != nil {
		return
	}
	orderID := atomic.AddUint64(&c.orderID, 1)
	strOrderID := fmt.Sprintf("%s%d", c.strStart, orderID)
	var action ctp.CThostFtdcInputOrderField
	// ----------必填字段----------------
	action.BrokerID = c.cfg.BrokerID
	///投资者代码
	action.InvestorID = c.cfg.User
	///交易所代码
	action.ExchangeID = exchangeID
	///合约代码
	action.InstrumentID = symbol
	///报单价格条件 THOST_FTDC_OPT_LimitPrice
	action.OrderPriceType = '2'
	if act.Action.IsLong() {
		///买卖方向
		action.Direction = '0'
	} else {
		action.Direction = '1'
	}
	///价格
	action.LimitPrice = act.Price
	///数量
	action.VolumeTotalOriginal = int(act.Amount)
	if act.Action.IsOpen() {
		///组合开平标志
		action.CombOffsetFlag = "0"
	} else {
		action.CombOffsetFlag = "1"
	}
	///组合投机套保标志 THOST_FTDC_ECIDT_Speculation
	action.CombHedgeFlag = "1"
	///触发条件 THOST_FTDC_CC_Immediately
	action.ContingentCondition = '1'
	///有效期类型THOST_FTDC_TC_GFD
	action.TimeCondition = '3'
	///成交量类型 THOST_FTDC_VC_AV
	action.VolumeCondition = '1'
	///最小成交量
	action.MinVolume = 0
	// -----------------选填字段------------------------
	///报单引用
	action.OrderRef = strOrderID
	///用户代码
	action.UserID = c.cfg.User
	///GTD日期
	// action.GTDDate;
	///止损价
	// action.StopPrice;
	///强平原因
	action.ForceCloseReason = '0'
	///自动挂起标志
	// action.IsAutoSuspend;
	///业务单元
	// action.BusinessUnit;
	///请求编号
	action.RequestID = int(orderID)
	///用户强评标志
	// action.UserForceClose;
	///互换单标志
	// action.IsSwapOrder;

	///投资单元代码
	// action.InvestUnitID
	///资金账号
	// TThostFtdcAccountIDType	AccountID;
	///币种代码
	// action.CurrencyID
	///交易编码
	// action.ClientID;
	///IP地址
	// action.IPAddress;
	///Mac地址
	// action.MacAddress;
	logrus.Info("processOrder ReqOrderInsert", action)
	nRet := c.tdApi.ReqOrderInsert(&action, getReqID())
	if nRet != 0 {
		err = fmt.Errorf("ReqOrderInsert error: %d", nRet)
		return
	}
	ret = &Order{
		OrderID:  strconv.FormatUint(orderID, 10),
		Symbol:   act.Symbol,
		Currency: "cny",
		Amount:   act.Amount,
		Price:    act.Price,
		Status:   "send",
		Time:     time.Now(),
	}
	if act.Action.IsLong() {
		ret.Side = "buy"
	} else {
		ret.Side = "sell"
	}
	c.orders[strOrderID] = ret
	return
}

func (c *CtpExchange) CancelAllOrders() (orders []*Order, err error) {
	for k, v := range c.orders {
		if v.Status == OrderStatusFilled || v.Status == OrderStatusCanceled {
			continue
		}
		err = c.cancelOrder(k, v)
		if err != nil {
			logrus.Errorf("cancel order %s failed: %s", k, err.Error())
		}
	}

	return
}

func (c *CtpExchange) cancelOrder(ref string, o *Order) (err error) {
	exchangeID, _, err := parseSymbol(o.Symbol)
	if err != nil {
		return
	}
	rID := getReqID()
	pInputOrderAction := &ctp.CThostFtdcInputOrderActionField{
		BrokerID:   c.cfg.BrokerID,
		InvestorID: c.cfg.User,
		// OrderActionRef
		OrderRef:   ref,
		RequestID:  rID,
		FrontID:    c.tdSpi.frontID,
		SessionID:  c.tdSpi.sessionID,
		ExchangeID: exchangeID,
		OrderSysID: o.OrderID,
		ActionFlag: byte(0),
		// LimitPrice     float64
		// VolumeChange   int
		// UserID         string
		// InstrumentID   string
		// InvestUnitID   string
		// IPAddress      string
		// MacAddress     string
	}
	nRet := c.tdApi.ReqOrderAction(pInputOrderAction, getReqID())
	if nRet != 0 {
		err = fmt.Errorf("cancel order failed: %d", nRet)
	}
	return
}

func (b *CtpExchange) GetSymbols() (symbols []core.SymbolInfo, err error) {
	return
}

// GetBalanceChan
func (c *CtpExchange) GetDataChan() chan *core.ExchangeData {
	return c.datas
}

// {"TradingDay":"20211119","InstrumentID":"al2201","ExchangeID":"","ExchangeInstID":"","LastPrice":18350,"PreSettlementPrice":18580,"PreClosePrice":18475,"PreOpenInterest":236229,"OpenPrice":18475,"HighestPrice":18490,"LowestPrice":18230,"Volume":165674,"Turnover":15192326925,"OpenInterest":250293,"ClosePrice":0,"SettlementPrice":0,"UpperLimitPrice":20065,"LowerLimitPrice":17090,"PreDelta":0,"CurrDelta":0,"UpdateTime":"23:40:51","UpdateMillisec":500,"BidPrice1":18350,"BidVolume1":25,"AskPrice1":18355,"AskVolume1":71,"BidPrice2":18345,"BidVolume2":43,"AskPrice2":18360,"AskVolume2":83,"BidPrice3":18340,"BidVolume3":59,"AskPrice3":18365,"AskVolume3":48,"BidPrice4":18335,"BidVolume4":61,"AskPrice4":18370,"AskVolume4":75,"BidPrice5":18330,"BidVolume5":69,"AskPrice5":18375,"AskVolume5":155,"AveragePrice":91700.12750944626,"ActionDay":"20211118"}
func (c *CtpExchange) onDepthData(pDepthMarketData *ctp.CThostFtdcDepthMarketDataField) {
	if !c.HasInit() {
		return
	}
	if c.prevVolume == 0 {
		c.prevVolume = float64(pDepthMarketData.Volume)
		return
	}
	now := time.Now()
	loc, _ := time.LoadLocation("Asia/Shanghai")
	date := now.Format("20060102")
	timeStr := fmt.Sprintf("%s %s.%03d", date, pDepthMarketData.UpdateTime, pDepthMarketData.UpdateMillisec)
	tm, err := time.ParseInLocation("20060102 15:04:05.000", timeStr, loc)
	if err != nil {
		logrus.Errorf("CtpExchange Parse MarketData timestamp %s failed %s", timeStr, err.Error())
	}
	var depth Depth
	depth.UpdateTime = tm
	depth.Buys = append(depth.Buys, DepthInfo{
		Price:  pDepthMarketData.BidPrice1,
		Amount: float64(pDepthMarketData.BidVolume1)})
	depth.Sells = append(depth.Sells, DepthInfo{
		Price:  pDepthMarketData.AskPrice1,
		Amount: float64(pDepthMarketData.AskVolume1)})
	temp := core.NewExchangeData(c.name, core.EventDepth, &depth)
	temp.Symbol = pDepthMarketData.InstrumentID
	c.datas <- temp
	var trade Trade
	trade.Time = tm
	// pDepthMarketData.UpdateTime
	trade.Amount = float64(pDepthMarketData.Volume) - c.prevVolume
	c.prevVolume = float64(pDepthMarketData.Volume)
	trade.Price = pDepthMarketData.LastPrice
	tempMarket := core.NewExchangeData(c.name, core.EventTradeMarket, &trade)
	tempMarket.Symbol = pDepthMarketData.InstrumentID
	c.datas <- temp
}

func (c *CtpExchange) onTrade(pTrade *ctp.CThostFtdcTradeField) {
	v, ok := c.orders[pTrade.OrderRef]
	if ok {
		if pTrade.Volume == int(v.Amount) {
			v.Status = OrderStatusFilled
		} else {
			v.Amount -= float64(pTrade.Volume)
		}
	}
	var trade Trade
	trade.ID = pTrade.TradeID
	// trade.Action
	// trade.Time   = pTrade.TradeDate + pTrade.TradeTime
	trade.Price = pTrade.Price
	trade.Amount = float64(pTrade.Volume)
	// trade.Side = pTrade.TradeType
	trade.Remark = pTrade.OrderRef
	buf, err := json.Marshal(pTrade)
	logrus.Println("trade:", string(buf), err)
	temp := core.NewExchangeData(c.name, core.EventTrade, &trade)
	temp.Symbol = pTrade.InstrumentID
	c.datas <- temp

}

func (c *CtpExchange) updateOrderStatus(id, orderSysID, status, err string) {
	o, ok := c.orders[id]
	if !ok {
		logrus.Warnf("updateOrderStatus %s not exist, %s, %s", id, status, err)
		return
	}
	o.OrderID = orderSysID
	o.Status = status
}

func (c *CtpExchange) updatePosition(pInvestorPosition *ctp.CThostFtdcInvestorPositionDetailField, reqID int, isLast bool) {
	if reqID != c.positionReqID {
		logrus.Errorf("updatePosition error, reqID notmatch: %d %d", c.positionReqID, reqID)
		return
	}
	symbol := formatSymbol(pInvestorPosition.ExchangeID, pInvestorPosition.InstrumentID)
	buf, _ := json.Marshal(pInvestorPosition)
	logrus.Warn("updatePosition:", string(buf))
	var pos Position
	pos.Hold = float64(pInvestorPosition.Volume)
	if pInvestorPosition.Direction == '1' {
		pos.Hold = 0 - pos.Hold
	}
	pos.Symbol = symbol
	pos.Price = pInvestorPosition.OpenPrice
	c.positionCache = append(c.positionCache, pos)
	if isLast {
		var posMerge Position
		var totalPrice float64
		for _, v := range c.positionCache {
			if v.Hold != 0 {
				totalPrice += v.Price
			}
			posMerge.Hold += v.Hold
		}
		posMerge.Price = common.FloatMul(totalPrice, posMerge.Hold)
		posMerge.Symbol = symbol
		temp := core.NewExchangeData(pInvestorPosition.InstrumentID, core.EventPosition, &posMerge)
		temp.Symbol = symbol
		c.datas <- temp
		c.positionCache = []Position{}
	}
}
