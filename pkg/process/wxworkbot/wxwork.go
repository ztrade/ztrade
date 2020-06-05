package wxworkbot

import (
	"bytes"
	"text/template"
	. "ztrade/pkg/define"
	. "ztrade/pkg/event"

	"github.com/SuperGod/coinex"
	. "github.com/SuperGod/trademodel"
	"github.com/SuperGod/wxwork"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
)

var (
	tradeNotifyTpl = `### {{.symbol}} {{.trade.Action}}
Price:
{{- if .isLong -}}
<font color="info">    {{.trade.Price}}</font>
{{- else -}}
<font color="warning">    {{.trade.Price}}</font>
{{end}}

{{- if .isStop -}}
<font color="warning">Stop lose:    </font>
{{- else}}
<font color="info">Trade:    </font>
{{- end -}}

{{- if .isLong -}}
<font color="info">{{.trade.Amount}}</font>
{{- else -}}
<font color="warning">-{{.trade.Amount}}</font>
{{- end}}

ID: {{.trade.ID}}
Time: {{.trade.Time.Format "2006-01-02 15:04:05"}}
`

	positionTpl = `{{.symbol}} Hold
{{- if .isLong -}}
<font color="info">    {{.pos}}</font>
{{- else -}}
<font color="warning">    {{.pos}}</font>
{{end}}
`
	tradeTpl *template.Template
	posTpl   *template.Template
)

func init() {
	var err error
	tradeTpl, err = template.New("tradeNotify").Parse(tradeNotifyTpl)
	if err != nil {
		panic("trade templte error:" + err.Error())
	}
	posTpl, err = template.New("positionNotify").Parse(positionTpl)
	if err != nil {
		panic("position templte error:" + err.Error())
	}
}

// WXWork send notify to wxwork
type WXWork struct {
	BaseProcesser
	wxAPI       *wxwork.API
	tradeNotify bool
	symbol      string
	pos         float64
}

// NewWXWork WXWork constructor
// tradeNotify: send trade notify or not
func NewWXWork(tradeNotify bool) (w *WXWork, err error) {
	w = new(WXWork)
	w.tradeNotify = tradeNotify
	w.Name = "wxwork"
	w.wxAPI, err = wxwork.GetAPI()
	return
}

func (w *WXWork) Init(bus *Bus) (err error) {
	w.BaseProcesser.Init(bus)
	bus.Subscribe(EventCandleParam, w.onEventCandleParam)
	bus.Subscribe(EventNotify, w.onEventNotify)
	bus.Subscribe(EventTrade, w.onEventTrade)
	bus.Subscribe(EventPosition, w.onEventPosition)
	return
}

func (w *WXWork) onEventCandleParam(e Event) (err error) {
	var cParam CandleParam
	err = mapstructure.Decode(e.GetData(), &cParam)
	if err != nil {
		return
	}
	w.symbol = cParam.Symbol
	return
}

func (w *WXWork) onEventNotify(e Event) (err error) {
	var notify NotifyEvent
	err = mapstructure.Decode(e.GetData(), &notify)
	if err != nil {
		return
	}
	switch notify.Type {
	case "text":
		err = w.wxAPI.SendTextToUser(notify.Content)
	case "markdown":
		err = w.wxAPI.SendMarkdownToUser(notify.Content)
	default:
		log.Error("unsupport notify type:", notify.Type)
	}
	return
}

func (w *WXWork) onEventTrade(e Event) (err error) {
	if !w.tradeNotify {
		return
	}
	var trade Trade
	err = mapstructure.Decode(e.GetData(), &trade)
	if err != nil {
		return
	}
	err = w.sendTrade(trade)
	return
}
func (w *WXWork) onEventPosition(e Event) (err error) {
	pos := new(coinex.Position)
	err = mapstructure.Decode(e.GetData(), pos)
	if err != nil {
		return
	}
	if w.pos != pos.Hold {
		w.pos = pos.Hold
		err = w.sendPos(w.pos)
	}
	return
}

func (w *WXWork) sendPos(pos float64) (err error) {
	var buf bytes.Buffer
	data := map[string]interface{}{
		"pos":    pos,
		"symbol": w.symbol,
		"isLong": pos > 0,
	}
	err = posTpl.Execute(&buf, data)
	if err != nil {
		return
	}
	err = w.wxAPI.SendMarkdownToUser(buf.String())
	return
}

func (w *WXWork) sendTrade(trade Trade) (err error) {
	var buf bytes.Buffer
	data := map[string]interface{}{
		"trade":  trade,
		"isLong": trade.Action.IsLong(),
		"isOpen": trade.Action.IsOpen(),
		"isStop": trade.Action.IsStop(),
		"symbol": w.symbol,
	}
	err = tradeTpl.Execute(&buf, data)
	if err != nil {
		return
	}
	err = w.wxAPI.SendMarkdownToUser(buf.String())
	return
}
