package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"math"
	"os"
	"sort"

	. "github.com/SuperGod/trademodel"
	"github.com/ztrade/ztrade/pkg/common"
)

type Report struct {
	actions          []TradeAction
	trades           []Trade
	balanceInit      float64
	profit           float64
	maxLose          float64
	winRate          float64
	profitHistory    []float64
	tmplDatas        []*tmplAct
	totalAction      int
	maxDrawdown      float64
	maxDrawdownValue float64
}

type tmplAct struct {
	Trade
	Total       float64
	TotalProfit float64 // total profit,sum of all history profits,if action is open, total profit is zero
	Profit      float64 // profit, if action is open, profit is zero
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

func (r *Report) Analyzer() (err error) {
	var longTotal, longAmount, longOnce float64
	var shortTotal, shortAmount, shortOnce float64
	var actTotal, lose float64
	var success, total int
	var tmplData, lastTmplData *tmplAct
	var lastMaxTotal, lastMinTotal, drawdown, drawdownValue float64
	bal := common.NewVBalance()
	bal.Set(r.balanceInit)
	fmt.Println("balance init:", r.balanceInit)
	startBalance := bal.Get()
	for k, v := range r.trades {
		if k == len(r.trades)-1 {
			if v.Action.IsOpen() {
				break
			}
		}
		bal.AddTrade(v)
		actTotal = common.FloatMul(v.Price, v.Amount)
		if v.Action.IsLong() {
			longTotal = common.FloatAdd(longTotal, actTotal)
			longAmount = common.FloatAdd(longAmount, v.Amount)
			longOnce = common.FloatAdd(longOnce, actTotal)
			// log.Println("buy action", v.Time, v.Action, v.Price, v.Amount)
			// longPrice = longTotal / longAmount
		} else {
			// log.Println("sell action", v.Time, v.Action, v.Price, v.Amount)
			shortTotal = common.FloatAdd(shortTotal, actTotal)
			shortAmount = common.FloatAdd(shortAmount, v.Amount)
			shortOnce = actTotal
			// shortPrice = shortTotal / shortAmount
		}
		r.totalAction++
		tmplData = &tmplAct{Trade: v,
			Total:  common.FloatSub(shortTotal, longTotal),
			Profit: 0}
		r.tmplDatas = append(r.tmplDatas, tmplData)
		// log.Println("amount:", longAmount, shortAmount)
		// one round finish
		if longAmount == shortAmount {
			if lastTmplData != nil {
				tmplData.TotalProfit = lastTmplData.TotalProfit
			}
			tmplData.Profit = common.FloatSub(shortOnce, longOnce)
			tmplData.TotalProfit = common.FloatAdd(tmplData.TotalProfit, tmplData.Profit)
			r.profitHistory = append(r.profitHistory, shortTotal-longTotal)
			total++
			if longOnce <= shortOnce {
				success++
			} else {
				lose = common.FloatDiv((common.FloatMul(common.FloatSub(longOnce, shortOnce), 100)), longOnce)
				if math.Abs(lose) > math.Abs(r.maxLose) {
					r.maxLose = lose
				}
			}
			shortOnce = 0
			longOnce = 0
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
	if lastTmplData != nil {
		r.profit = lastTmplData.TotalProfit
	}
	endBalance := bal.Get()
	r.profit = endBalance - startBalance
	if total > 0 {
		r.winRate = common.FloatDiv(float64(success), float64(total))
	}
	fmt.Println("fee total:", bal.GetFeeTotal())
	return
}

func (r *Report) WinRate() (rate float64) {
	rate = r.winRate
	return
}

func (r *Report) Profit() (profit float64) {
	profit = r.profit
	return
}

// MaxLose max total lose
func (r *Report) MaxLose() (lose float64) {
	lose = r.maxLose
	return
}

// MaxDrawdown get max drawdown percent
func (r *Report) MaxDrawdown() float64 {
	return r.maxDrawdown
}

// MaxDrawdown get max drawdown value
func (r *Report) MaxDrawdownValue() float64 {
	return r.maxDrawdownValue
}

func (r *Report) GetReport() (report string) {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("Total action:%d\n", len(r.actions)))
	buf.WriteString(fmt.Sprintf("Win rate:%f\n", r.WinRate()))
	buf.WriteString(fmt.Sprintf("Profit:%f\n", r.Profit()))
	buf.WriteString(fmt.Sprintf("Max lose percent:%f\n", r.MaxLose()))
	buf.WriteString(fmt.Sprintf("Max drawdown percent:%f%%\n", r.MaxDrawdown()))
	buf.WriteString(fmt.Sprintf("Max drawdown value :%f\n", r.MaxDrawdown()))
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
	tmpl, err := template.ParseFiles(common.GetExecDir() + "/report/report.tmpl")
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
	err = tmpl.Execute(w, data)
	return
}

func (r *Report) OnBalanceInit(balance float64) (err error) {
	r.balanceInit = balance
	return
}

func (r *Report) OnTrade(t Trade) {
	r.trades = append(r.trades, t)
	return
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
