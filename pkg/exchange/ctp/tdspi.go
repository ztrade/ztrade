package ctp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/ztrade/ctp"
)

type TdSpi struct {
	ctp.CThostFtdcTraderSpiBase
	hasLogin  SafeWait
	symbols   map[string]*ctp.CThostFtdcInstrumentField
	ex        *CtpExchange
	api       *ctp.CThostFtdcTraderApi
	cfg       *Config
	frontID   int
	sessionID int
}

func NewTdSpi(ex *CtpExchange, cfg *Config, api *ctp.CThostFtdcTraderApi) *TdSpi {
	td := new(TdSpi)
	td.cfg = cfg
	td.ex = ex
	td.api = api
	td.symbols = make(map[string]*ctp.CThostFtdcInstrumentField)
	return td
}
func (s *TdSpi) GetSymbols() (symbols map[string]*ctp.CThostFtdcInstrumentField) {
	return s.symbols
}

func (s *TdSpi) OnFrontConnected() {
	nRet := s.api.ReqAuthenticate(&ctp.CThostFtdcReqAuthenticateField{BrokerID: s.cfg.BrokerID, UserID: s.cfg.User, UserProductInfo: "", AuthCode: s.cfg.AuthCode, AppID: s.cfg.AppID}, getReqID())
	if nRet != 0 {
		s.hasLogin.Done(fmt.Errorf("ReqAuthenticate failed: %d", nRet))
	}
}

func (s *TdSpi) OnRspAuthenticate(pRspAuthenticateField *ctp.CThostFtdcRspAuthenticateField, pRspInfo *ctp.CThostFtdcRspInfoField, nRequestID int, bIsLast bool) {
	if pRspInfo != nil && pRspInfo.ErrorID != 0 {
		err := fmt.Errorf("OnRspAuthenticate error %d,%s", pRspInfo.ErrorID, pRspInfo.ErrorMsg)
		s.hasLogin.Done(err)
		return
	}
	nRet := s.api.ReqUserLogin(&ctp.CThostFtdcReqUserLoginField{UserID: s.cfg.User, BrokerID: s.cfg.BrokerID, Password: s.cfg.Password}, getReqID())
	if nRet != 0 {
		s.hasLogin.Done(fmt.Errorf("ReqUserLogin failed: %d", nRet))
	}
}
func (s *TdSpi) OnRspUserLogin(pRspUserLogin *ctp.CThostFtdcRspUserLoginField, pRspInfo *ctp.CThostFtdcRspInfoField, nRequestID int, bIsLast bool) {
	if pRspInfo != nil && pRspInfo.ErrorID != 0 {
		err := fmt.Errorf("OnRspUserLogin error %d,%s", pRspInfo.ErrorID, pRspInfo.ErrorMsg)
		s.hasLogin.Done(err)
		return
	}
	pSettlementInfoConfirm := &ctp.CThostFtdcSettlementInfoConfirmField{
		BrokerID:   pRspUserLogin.BrokerID,
		InvestorID: pRspUserLogin.UserID,
	}
	s.frontID = pRspUserLogin.FrontID
	s.sessionID = pRspUserLogin.SessionID
	nRet := s.api.ReqSettlementInfoConfirm(pSettlementInfoConfirm, getReqID())
	if nRet != 0 {
		s.hasLogin.Done(fmt.Errorf("SettlementInfoConfirm failed: %d", nRet))
	}
}

func (s *TdSpi) OnRspSettlementInfoConfirm(pSettlementInfoConfirm *ctp.CThostFtdcSettlementInfoConfirmField, pRspInfo *ctp.CThostFtdcRspInfoField, nRequestID int, bIsLast bool) {
	if pRspInfo != nil && pRspInfo.ErrorID != 0 {
		err := fmt.Errorf("OnRspSettlementInfoConfirm error: %s", pRspInfo.ErrorMsg)
		logrus.Error(err.Error())
		s.hasLogin.Done(err)
		return
	}
	buf, _ := json.Marshal(pSettlementInfoConfirm)
	logrus.Info("OnRspSettlementInfoConfirm:", string(buf))
	s.hasLogin.Done(nil)
}

func (s *TdSpi) OnRtnInstrumentStatus(pInstrumentStatus *ctp.CThostFtdcInstrumentStatusField) {
	// buf, _ := json.Marshal(pInstrumentStatus)
	// fmt.Println("OnRtnInstrumentStatus:", string(buf))
}

func (s *TdSpi) OnRspQryInstrument(pInstrument *ctp.CThostFtdcInstrumentField, pRspInfo *ctp.CThostFtdcRspInfoField, nRequestID int, bIsLast bool) {
	// defer func() {
	// if bIsLast {
	//
	// }
	// }()
	if pRspInfo != nil && pRspInfo.ErrorID != 0 {
		logrus.Error("OnRspQryInstrument error", pRspInfo.ErrorID, pRspInfo.ErrorMsg)
	}
	if pInstrument == nil {
		logrus.Warn("pInstrument is null")
		return
	}
	if pInstrument.ProductClass != '1' {
		return
	}
	s.symbols[pInstrument.InstrumentID] = pInstrument

}
func (s *TdSpi) WaitLogin(ctx context.Context) (err error) {
	return s.hasLogin.Wait(ctx)
}

func (s *TdSpi) OnRspOrderInsert(pInputOrder *ctp.CThostFtdcInputOrderField, pRspInfo *ctp.CThostFtdcRspInfoField, nRequestID int, bIsLast bool) {
	if pRspInfo != nil && pRspInfo.ErrorID != 0 {
		logrus.Error("OnRspOrderInsert error:", pRspInfo.ErrorMsg)
		return
	}
	buf, _ := json.Marshal(pInputOrder)
	logrus.Info("OnRspOrderInsert:", string(buf))
}

func (s *TdSpi) OnErrRtnOrderInsert(pInputOrder *ctp.CThostFtdcInputOrderField, pRspInfo *ctp.CThostFtdcRspInfoField) {
	if pRspInfo != nil && pRspInfo.ErrorID != 0 {
		logrus.Error("OnErrRtnOrderInsert error:", pRspInfo.ErrorMsg)
		return
	}
	buf, _ := json.Marshal(pInputOrder)
	logrus.Info("OnErrRtnOrderInsert:", string(buf))
}

// {"BrokerID":"9999","InvestorID":"164347","InstrumentID":"al2201","OrderRef":"1","UserID":"164347","OrderPriceType":50,"Direction":48,"CombOffsetFlag":"0","CombHedgeFlag":"1","LimitPrice":18640,"VolumeTotalOriginal":1,"TimeCondition":51,"GTDDate":"","VolumeCondition":49,"MinVolume":1,"ContingentCondition":49,"StopPrice":0,"ForceCloseReason":48,"IsAutoSuspend":0,"BusinessUnit":"9999cac","RequestID":0,"OrderLocalID":"       12405","ExchangeID":"SHFE","ParticipantID":"9999","ClientID":"9999164327","ExchangeInstID":"al2201","TraderID":"9999cac","InstallID":1,"OrderSubmitStatus":48,"NotifySequence":0,"TradingDay":"20211117","SettlementID":1,"OrderSysID":"       29722","OrderSource":48,"OrderStatus":48,"OrderType":48,"VolumeTraded":1,"VolumeTotal":0,"InsertDate":"20211117","InsertTime":"00:08:49","ActiveTime":"","SuspendTime":"","UpdateTime":"","CancelTime":"","ActiveTraderID":"9999cac","ClearingPartID":"","SequenceNo":21573,"FrontID":1,"SessionID":2040216403,"UserProductInfo":"","StatusMsg":"全部成交报单已提交","UserForceClose":0,"ActiveUserID":"","BrokerOrderSeq":32344,"RelativeOrderSysID":"","ZCETotalTradedVolume":0,"IsSwapOrder":0,"BranchID":"","InvestUnitID":"","AccountID":"","CurrencyID":"","IPAddress":"","MacAddress":""}
func (s *TdSpi) OnRtnOrder(pOrder *ctp.CThostFtdcOrderField) {
	s.ex.updateOrderStatus(pOrder.OrderRef, pOrder.OrderSysID, pOrder.StatusMsg, "")
}
func (s *TdSpi) OnRtnTrade(pTrade *ctp.CThostFtdcTradeField) {
	s.ex.onTrade(pTrade)
}

func (s *TdSpi) OnRspQrySettlementInfo(pSettlementInfo *ctp.CThostFtdcSettlementInfoField, pRspInfo *ctp.CThostFtdcRspInfoField, nRequestID int, bIsLast bool) {
	if pRspInfo != nil && pRspInfo.ErrorID != 0 {
		logrus.Error("OnRspQrySettlementInfo error:", pRspInfo.ErrorMsg)
		return
	}
	buf, _ := json.Marshal(pSettlementInfo)
	logrus.Info("OnRspQrySettlementInfo:", string(buf))
}

func (s *TdSpi) OnRspQryInvestorPosition(pInvestorPosition *ctp.CThostFtdcInvestorPositionField, pRspInfo *ctp.CThostFtdcRspInfoField, nRequestID int, bIsLast bool) {
	if pRspInfo != nil && pRspInfo.ErrorID != 0 {
		logrus.Errorf("OnRspQryInvestorPosition error:", pRspInfo.ErrorMsg)
		return
	}
	s.ex.updatePosition(pInvestorPosition)
}

func (s *TdSpi) OnRspOrderAction(pInputOrderAction *ctp.CThostFtdcInputOrderActionField, pRspInfo *ctp.CThostFtdcRspInfoField, nRequestID int, bIsLast bool) {
	if pRspInfo != nil && pRspInfo.ErrorID != 0 {
		logrus.Errorf("OnRspOrderAction error:", pRspInfo.ErrorMsg)
		return
	}
	buf, _ := json.Marshal(pInputOrderAction)
	logrus.Info("OnRspOrderAction:", string(buf))
}

func (s *TdSpi) OnErrRtnOrderAction(pOrderAction *ctp.CThostFtdcOrderActionField, pRspInfo *ctp.CThostFtdcRspInfoField) {
	buf, _ := json.Marshal(pOrderAction)
	logrus.Info("OnErrRtnOrderActionOnRspOrderAction:", string(buf))
}
