package ctp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/sirupsen/logrus"
	"github.com/ztrade/ctp"
)

type MdSpi struct {
	hasLogin  SafeWait
	ex        *CtpExchange
	cfg       *Config
	api       *ctp.CThostFtdcMdApi
	connected uint32
}

func NewMdSpi(ex *CtpExchange, cfg *Config, api *ctp.CThostFtdcMdApi) (spi *MdSpi, err error) {
	spi = new(MdSpi)
	spi.api = api
	spi.cfg = cfg
	spi.ex = ex
	return
}

func (s *MdSpi) WaitDisconnect(closeChan chan bool) {
	for {
		select {
		case <-closeChan:
			return
		default:
		}
		isConnected := atomic.LoadUint32(&s.connected)
		if isConnected == 0 {
			return
		}
	}
}

func (s *MdSpi) WaitLogin(ctx context.Context) error {
	return s.hasLogin.Wait(ctx)
}

func (s *MdSpi) OnFrontConnected() {
	logrus.Println("mdSpi OnFrontConnected")
	nRet := s.api.ReqUserLogin(&ctp.CThostFtdcReqUserLoginField{UserID: s.cfg.User, BrokerID: s.cfg.BrokerID, Password: s.cfg.Password}, 0)
	if nRet != 0 {
		s.hasLogin.Done(fmt.Errorf("ReqUserLogin failed: %d", nRet))
		return
	}
	atomic.StoreUint32(&s.connected, 1)
}

func (s *MdSpi) OnFrontDisconnected(nReason int) {
	logrus.Println("OnFrontDisconnected:", nReason)
	atomic.StoreUint32(&s.connected, 0)
}

func (s *MdSpi) OnHeartBeatWarning(nTimeLapse int) {
	logrus.Println("OnHeartBeatWarning:", nTimeLapse)
}

func (s *MdSpi) OnRspUserLogin(pRspUserLogin *ctp.CThostFtdcRspUserLoginField, pRspInfo *ctp.CThostFtdcRspInfoField, nRequestID int, bIsLast bool) {
	if pRspInfo != nil && pRspInfo.ErrorID != 0 {
		err := fmt.Errorf("OnRspUserLogin error %d,%s", pRspInfo.ErrorID, pRspInfo.ErrorMsg)
		s.hasLogin.Done(err)
	}
	s.hasLogin.Done(nil)
}

func (s *MdSpi) OnRspUserLogout(pUserLogout *ctp.CThostFtdcUserLogoutField, pRspInfo *ctp.CThostFtdcRspInfoField, nRequestID int, bIsLast bool) {
	buf, _ := json.Marshal(pUserLogout)
	logrus.Infof("%d isLast: %t login success: %s", nRequestID, bIsLast, string(buf))
	logrus.Infof("login error: %d %s", pRspInfo.ErrorID, pRspInfo.ErrorMsg)
}

func (s *MdSpi) OnRspQryMulticastInstrument(pMulticastInstrument *ctp.CThostFtdcMulticastInstrumentField, pRspInfo *ctp.CThostFtdcRspInfoField, nRequestID int, bIsLast bool) {
	buf, _ := json.Marshal(pMulticastInstrument)
	logrus.Infof("%d isLast: %t logout success: %s", nRequestID, bIsLast, string(buf))
	logrus.Infof("logout error: %d %s", pRspInfo.ErrorID, pRspInfo.ErrorMsg)
}
func (s *MdSpi) OnRspError(pRspInfo *ctp.CThostFtdcRspInfoField, nRequestID int, bIsLast bool) {
	logrus.Warnf("%d resp error: %t:ErrorID: %d ErrorMsg:%s", nRequestID, bIsLast, pRspInfo.ErrorID, pRspInfo.ErrorMsg)
}

func (s *MdSpi) OnRspSubMarketData(pSpecificInstrument *ctp.CThostFtdcSpecificInstrumentField, pRspInfo *ctp.CThostFtdcRspInfoField, nRequestID int, bIsLast bool) {
	logrus.Info("onSubMarketData:", pSpecificInstrument.InstrumentID)
	if pRspInfo != nil && pRspInfo.ErrorID != 0 {
		logrus.Warnf("%d onSubMarketData: %t: ErrorID: %d ErrorMsg:%s", nRequestID, bIsLast, pRspInfo.ErrorID, pRspInfo.ErrorMsg)
	}
}

func (s *MdSpi) OnRspUnSubMarketData(pSpecificInstrument *ctp.CThostFtdcSpecificInstrumentField, pRspInfo *ctp.CThostFtdcRspInfoField, nRequestID int, bIsLast bool) {
	logrus.Info("onUnSubMarketData:", pSpecificInstrument.InstrumentID)
	logrus.Warnf("%d onUnSubMarketData: %t: ErrorID: %d ErrorMsg:%s", nRequestID, bIsLast, pRspInfo.ErrorID, pRspInfo.ErrorMsg)
}

func (s *MdSpi) OnRspSubForQuoteRsp(pSpecificInstrument *ctp.CThostFtdcSpecificInstrumentField, pRspInfo *ctp.CThostFtdcRspInfoField, nRequestID int, bIsLast bool) {
	logrus.Info("onSubForQuoteRsp:", pSpecificInstrument.InstrumentID)
	logrus.Warnf("%d onSubForQuoteRsp: %t: ErrorID: %d ErrorMsg:%s", nRequestID, bIsLast, pRspInfo.ErrorID, pRspInfo.ErrorMsg)
}

func (s *MdSpi) OnRspUnSubForQuoteRsp(pSpecificInstrument *ctp.CThostFtdcSpecificInstrumentField, pRspInfo *ctp.CThostFtdcRspInfoField, nRequestID int, bIsLast bool) {
	logrus.Info("onUnSubForQuoteRsp:", pSpecificInstrument.InstrumentID)
	logrus.Warnf("%d onUnSubForQuoteRsp: %t: ErrorID: %d ErrorMsg:%s", nRequestID, bIsLast, pRspInfo.ErrorID, pRspInfo.ErrorMsg)
}

func (s *MdSpi) OnRtnDepthMarketData(pDepthMarketData *ctp.CThostFtdcDepthMarketDataField) {
	if pDepthMarketData == nil {
		logrus.Errorf("marketdata is nil")
		return
	}
	s.ex.onDepthData(pDepthMarketData)

}
func (s *MdSpi) OnRtnForQuoteRsp(pForQuoteRsp *ctp.CThostFtdcForQuoteRspField) {
	buf, _ := json.Marshal(pForQuoteRsp)
	logrus.Info("onForQuoteRsp:", string(buf))
}
