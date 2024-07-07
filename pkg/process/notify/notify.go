package notify

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"text/template"

	log "github.com/sirupsen/logrus"
	"github.com/ztrade/exchange"
	. "github.com/ztrade/trademodel"
	. "github.com/ztrade/ztrade/pkg/core"
	. "github.com/ztrade/ztrade/pkg/event"
)

var (
	defaultNotifyConfig NotifyConfig
)

type NotifyConfig struct {
	Headers map[string]string
	Url     string
	Method  string
	Body    string
	Notify  struct {
		Trade  bool
		Order  bool
		Blance bool
	}
}

type Notify struct {
	BaseProcesser
	clt      http.Client
	cfg      *NotifyConfig
	bodyTmpl *template.Template
}

func NewNotify(cfg exchange.Config) (n *Notify, err error) {
	var nCfg = defaultNotifyConfig
	err = cfg.UnmarshalKey("notify", &nCfg)
	if err != nil {
		return
	}
	tpl, err := template.New("notify").Parse(nCfg.Body)
	if err != nil {
		return
	}
	fmt.Println(nCfg)
	n = new(Notify)
	n.cfg = &nCfg
	n.bodyTmpl = tpl
	return
}

func (n *Notify) Init(bus *Bus) (err error) {
	n.BaseProcesser.Init(bus)
	n.Subscribe(EventTrade, n.OnEventTrade)
	n.Subscribe(EventOrder, n.OnEventOrder)
	n.Subscribe(EventBalance, n.OnEventBalance)
	n.Subscribe(EventNotify, n.OnEventNotify)
	return
}

func (n *Notify) Start() (err error) {
	return
}

func (n *Notify) Stop() (err error) {
	return
}

func (n *Notify) OnEventTrade(e *Event) (err error) {
	if !n.cfg.Notify.Trade {
		return nil
	}
	t := e.GetData().(*Trade)
	if t == nil {
		err = fmt.Errorf("notify OnEventTrade type error:%#v", e.GetData())
		log.Error(err.Error())
		return
	}
	msg := fmt.Sprintf("%s price: %f, amount: %f", t.Action.String(), t.Price, t.Amount)
	return n.SendNotify(&NotifyEvent{Title: "Trade", Content: msg})
}

func (n *Notify) OnEventOrder(e *Event) (err error) {
	if !n.cfg.Notify.Order {
		return nil
	}
	act := e.GetData().(*TradeAction)
	if act == nil {
		log.Errorf("Notify decode tradeaction error: %##v", e.GetData())
		return
	}
	msg := fmt.Sprintf("%s %s price: %f, amount: %f", act.Symbol, act.Action.String(), act.Price, act.Amount)
	return n.SendNotify(&NotifyEvent{Title: "Order", Content: msg})
}

func (n *Notify) OnEventBalance(e *Event) (err error) {
	if !n.cfg.Notify.Blance {
		return nil
	}
	balance, ok := e.GetData().(*Balance)
	if !ok {
		log.Errorf("Notify OnEventBalance type error: %##v", e.GetData())
		return
	}
	msg := fmt.Sprintf("%s: %f", balance.Currency, balance.Balance)
	return n.SendNotify(&NotifyEvent{Title: "Blance", Content: msg})
}

func (n *Notify) OnEventNotify(e *Event) (err error) {
	nEvent, ok := e.GetData().(*NotifyEvent)
	if !ok {
		log.Errorf("Notify OnEventNotify type error: %##v", e.GetData())
		return
	}
	return n.SendNotify(nEvent)
}

func (n *Notify) SendNotify(evt *NotifyEvent) (err error) {
	var b bytes.Buffer
	err = n.bodyTmpl.Execute(&b, evt)
	if err != nil {
		return
	}
	req, err := http.NewRequest(n.cfg.Method, n.cfg.Url, &b)
	if err != nil {
		return
	}
	for k, v := range n.cfg.Headers {
		req.Header.Set(k, v)
	}
	resp, err := n.clt.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if resp.StatusCode != 200 {
		err = errors.New(string(body))
	}
	return
}
