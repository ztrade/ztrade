package report

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"math"
	"os"
	"sort"

	jsoniter "github.com/json-iterator/go"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/ztrade/base/common"
	. "github.com/ztrade/trademodel"
	"xorm.io/xorm"
)

//go:embed report.tmpl
var reportTmpl string

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type Report struct {
	actions          []TradeAction
	trades           []Trade
	balanceInit      float64
	profit           float64
	maxLose          float64
	winRate          float64
	profitHistory    []float64
	tmplDatas        []*RptAct
	totalAction      int
	maxDrawdown      float64
	maxDrawdownValue float64
	fee              float64
	profitLoseRatio  float64
}

type RptAct struct {
	Trade       `xorm:"extends"`
	Total       float64
	TotalProfit float64 // total profit,sum of all history profits,if action is open, total profit is zero
	Profit      float64 // profit, if action is open, profit is zero
	Fee         float64
}

func NewReportSimple() *Report {
	rep := new(Report)
	return rep
}

func NewReport(trades []Trade, balanceInit float64) *Report {
	rep := new(Report)
	rep.trades = trades
	rep.balanceInit = balanceInit
	return rep
}

func (r *Report) SetFee(fee float64) {
	r.fee = fee
}

func (r *Report) Analyzer() (err error) {
	nLen := len(r.trades)
	if nLen == 0 {
		return
	}
	i := nLen
	for ; i > 0; i-- {
		if !r.trades[i-1].Action.IsOpen() {
			break
		}
	}
	r.trades = r.trades[0:i]
	profitTotal := decimal.New(0, 0)
	loseTotal := decimal.New(0, 0)
	var longAmount, costOnce float64
	var shortAmount float64
	var actTotal, lose float64
	var success, total int
	var tmplData, lastTmplData *RptAct
	var lastMaxTotal, lastMinTotal, drawdown, drawdownValue float64
	var profit, fee float64
	bal := common.NewVBalance()
	bal.Set(r.balanceInit)
	bal.SetFee(r.fee)
	//	startBalance := bal.Get()

	for _, v := range r.trades {
		profit, fee, err = bal.AddTrade(v)
		if err != nil {
			log.Error("Report add trade error:", err.Error())
			return
		}
		actTotal = common.FloatMul(v.Price, v.Amount)
		if v.Action.IsLong() {
			longAmount = common.FloatAdd(longAmount, v.Amount)
			// log.Println("buy action", v.Time, v.Action, v.Price, v.Amount)
		} else {
			// log.Println("sell action", v.Time, v.Action, v.Price, v.Amount)
			shortAmount = common.FloatAdd(shortAmount, v.Amount)
		}
		if v.Action.IsOpen() {
			costOnce = common.FloatAdd(costOnce, actTotal)
		}

		r.totalAction++
		tmplData = &RptAct{Trade: v,
			Total:  bal.Get(),
			Profit: profit,
			Fee:    fee,
		}
		r.tmplDatas = append(r.tmplDatas, tmplData)
		// one round finish
		if longAmount == shortAmount {
			if lastTmplData != nil {
				tmplData.TotalProfit = lastTmplData.TotalProfit
			}
			tmplData.Profit = profit
			if profit > 0 {
				profitTotal = profitTotal.Add(decimal.NewFromFloat(profit))
			} else {
				loseTotal = loseTotal.Add(decimal.NewFromFloat(profit))
			}
			tmplData.TotalProfit = common.FloatAdd(tmplData.TotalProfit, tmplData.Profit)
			r.profitHistory = append(r.profitHistory, profit)
			total++
			if profit > 0 {
				success++
			} else {
				if costOnce != 0 {
					// profit / cost
					lose = common.FloatDiv(common.FloatMul(profit, 100), costOnce)
				}
				if math.Abs(lose) > math.Abs(r.maxLose) {
					r.maxLose = lose
				}
			}
			costOnce = 0
		}
		if tmplData.TotalProfit != 0 {
			lastTmplData = tmplData
			if tmplData.TotalProfit > lastMaxTotal {
				lastMaxTotal = tmplData.TotalProfit
				// update lastMinTotal
				lastMinTotal = lastMaxTotal
			}
			if tmplData.TotalProfit < lastMinTotal {
				lastMinTotal = tmplData.TotalProfit
			}
			if lastMaxTotal != 0 {
				drawdownValue = common.FloatSub(lastMaxTotal, lastMinTotal)
				if drawdownValue > r.maxDrawdownValue {
					r.maxDrawdownValue = drawdownValue
				}

				drawdown = common.FloatDiv(common.FloatMul(drawdownValue, 100), lastMaxTotal)
				if drawdown > r.maxDrawdown {
					r.maxDrawdown = drawdown
				}
			}
		}
	}
	//	endBalance := bal.Get()
	if lastTmplData != nil {
		r.profit = lastTmplData.TotalProfit
	}
	// endBalance - startBalance
	if total > 0 {
		r.winRate = common.FloatDiv(float64(success), float64(total))
	}
	if !loseTotal.IsZero() {
		r.profitLoseRatio, _ = profitTotal.Div(loseTotal.Abs()).Float64()
	} else {
		r.profitLoseRatio, _ = profitTotal.Float64()
	}
	return
}

func (r *Report) WinRate() (rate float64) {
	rate = common.FormatFloat(r.winRate, 2)
	return
}

func (r *Report) Profit() (profit float64) {
	profit = common.FormatFloat(r.profit, 4)
	return
}

func (r *Report) ProfitLoseRatio() float64 {
	return r.profitLoseRatio
}

// MaxLose max total lose
func (r *Report) MaxLose() (lose float64) {
	lose = common.FormatFloat(r.maxLose, 2)
	return
}

// MaxDrawdown get max drawdown percent
func (r *Report) MaxDrawdown() float64 {
	return common.FormatFloat(r.maxDrawdown, 2)
}

// MaxDrawdown get max drawdown value
func (r *Report) MaxDrawdownValue() float64 {
	return common.FormatFloat(r.maxDrawdownValue, 4)
}

func (r *Report) GetReport() (report string) {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("Total action:%d\n", len(r.actions)))
	buf.WriteString(fmt.Sprintf("Win rate:%f\n", r.WinRate()))
	buf.WriteString(fmt.Sprintf("Profit:%f\n", r.Profit()))
	buf.WriteString(fmt.Sprintf("Max lose percent:%f\n", r.MaxLose()))
	buf.WriteString(fmt.Sprintf("Max drawdown percent:%f%%\n", r.MaxDrawdown()))
	buf.WriteString(fmt.Sprintf("Max drawdown value :%f\n", r.MaxDrawdown()))
	buf.WriteString(fmt.Sprintf("Profit lose ratio: %f\n", r.ProfitLoseRatio()))
	data, _ := json.Marshal(r.profitHistory)
	buf.WriteString(string(data))
	report = buf.String()
	return
}

func (r *Report) GenHTMLReport(fPath string) (err error) {
	f, err := os.OpenFile(fPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return
	}
	defer f.Close()
	err = r.GenHTML(f)
	return
}

func (r *Report) GenHTML(w io.Writer) (err error) {
	tmpl, err := template.New("report").Parse(reportTmpl)
	if err != nil {
		log.Println("tmpl parse failed:", err.Error())
		return
	}
	data := make(map[string]interface{})
	data["totalAction"] = r.totalAction
	data["winRate"] = r.WinRate()
	data["profit"] = r.Profit()
	data["maxLose"] = r.MaxLose()
	data["actions"] = r.tmplDatas
	data["maxDrawdown"] = r.MaxDrawdown()
	data["maxDrawdownValue"] = r.MaxDrawdownValue()
	data["profitLoseRatio"] = r.ProfitLoseRatio()
	err = tmpl.Execute(w, data)
	return
}

func (r *Report) OnBalanceInit(balance, fee float64) (err error) {
	r.balanceInit = balance
	r.fee = fee
	return
}

func (r *Report) OnTrade(t Trade) {
	r.trades = append(r.trades, t)
}

func (r *Report) GenRPT(fPath string) (err error) {
	sort.Slice(r.trades, func(i int, j int) bool {
		return r.trades[i].Time.Unix() < r.trades[j].Time.Unix()
	})
	err = r.Analyzer()
	if err != nil {
		return
	}
	err = r.GenHTMLReport(fPath)
	if err != nil {
		return
	}
	return
}

func (r *Report) GetResult() (ret ReportResult, err error) {
	sort.Slice(r.trades, func(i int, j int) bool {
		return r.trades[i].Time.Unix() < r.trades[j].Time.Unix()
	})
	err = r.Analyzer()
	if err != nil {
		return
	}
	ret.TotalAction = r.totalAction
	ret.Actions = r.tmplDatas
	ret.WinRate = r.WinRate()
	ret.Profit = r.Profit()
	ret.MaxLose = r.MaxLose()
	ret.MaxDrawdown = r.MaxDrawdown()
	ret.MaxDrawDownValue = r.MaxDrawdownValue()
	return
}

func (r *Report) ExportToDB(dbPath string) (err error) {
	eng, err := xorm.NewEngine("sqlite", dbPath)
	if err != nil {
		return
	}
	var data RptAct
	err = eng.Sync2(&data)
	if err != nil {
		return
	}
	defer eng.Close()
	fmt.Println("tmpl len:", len(r.tmplDatas))
	for _, v := range r.tmplDatas {
		_, err = eng.Insert(v)
		if err != nil {
			return
		}
	}
	return
}

type ReportResult struct {
	TotalAction      int
	WinRate          float64
	Profit           float64
	MaxLose          float64
	MaxDrawdown      float64
	MaxDrawDownValue float64
	Actions          []*RptAct `json:"-"`
	TotalFee         float64
}
