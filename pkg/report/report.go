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
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/montanaflynn/stats"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/ztrade/base/common"
	. "github.com/ztrade/trademodel"
	"xorm.io/xorm"
)

//go:embed report.tmpl
var reportTmpl string

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type ReportResult struct {
	TotalAction      int     // 总开单次数
	WinRate          float64 // 胜率
	TotalProfit      float64 // 总收益
	MaxLose          float64 // 最大单次亏损百分比
	MaxDrawdown      float64 // 最大回撤百分比
	MaxDrawdownValue float64 // 最大回撤值
	TotalFee         float64 // 总手续费
	StartBalance     float64 // 起始余额
	EndBalance       float64 // 结束余额
	ProfitPercent    float64 // 收益率
	ProfitVariance   float64 // 盈利方差
	LoseVariance     float64 // 亏损方差

	TotalReturn      float64 // 总收益率
	AnnualReturn     float64 // 年化收益率
	SharpeRatio      float64 // 夏普比率
	SortinoRatio     float64 // 索提诺比率
	Volatility       float64 // 年化波动率
	ProfitFactor     float64 // 盈亏比
	CalmarRatio      float64 // 卡玛比率
	ConsistencyScore float64 // 连续性得分
	SmoothnessScore  float64 // 平滑性得分
	OverallScore     float64 // 综合得分
	LongTrades       int     // 做多次数
	ShortTrades      int     // 做空次数

	Actions []*RptAct `json:"-"` // 所有的操作记录
}

type Report struct {
	actions       []TradeAction
	trades        []Trade
	balanceInit   float64
	balanceEnd    float64
	maxLose       float64
	profitHistory []float64
	tmplDatas     []*RptAct
	fee           float64

	lever        float64
	riskFreeRate float64 // 无风险利率
	startTime    time.Time
	endTime      time.Time

	result ReportResult
}

type RptAct struct {
	Trade       `xorm:"extends"`
	Total       float64
	TotalProfit float64 // total profit,sum of all history profits,if action is open, total profit is zero
	Profit      float64 // profit, if action is open, profit is zero
	ProfitRate  float64
	Fee         float64
	IsFinish    bool
}

func NewReportSimple() *Report {
	rep := new(Report)
	rep.riskFreeRate = 0.02
	return rep
}

func NewReport(trades []Trade, balanceInit float64) *Report {
	rep := new(Report)
	rep.trades = trades
	rep.balanceInit = balanceInit
	rep.riskFreeRate = 0.02
	return rep
}

// SetTimeRange set time range for report
func (r *Report) SetTimeRange(start, end time.Time) {
	r.startTime = start
	r.endTime = end
}

func (r *Report) SetFee(fee float64) {
	r.fee = fee
}

func (r *Report) SetLever(lever float64) {
	r.lever = lever
}

func (r *Report) Analyzer() (err error) {
	r.tmplDatas = nil
	r.profitHistory = nil
	r.result = ReportResult{}
	r.maxLose = 0
	r.balanceEnd = r.balanceInit
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
	var profit, profitRate, fee float64
	var profitArray, loseArray []float64
	bal := common.NewLeverBalance()
	bal.Set(r.balanceInit)
	bal.SetFee(r.fee)
	bal.SetLever(r.lever)
	// startBalance := bal.Get()

	for _, v := range r.trades {
		profit, profitRate, fee, err = bal.AddTrade(v)
		if err != nil {
			log.Error("Report add trade error:", err.Error())
			return
		}
		r.result.TotalFee = common.FloatAdd(r.result.TotalFee, fee)
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

		tmplData = &RptAct{Trade: v,
			Total:    common.FormatFloat(bal.Get(), 4),
			Profit:   common.FormatFloat(profit, 4),
			Fee:      fee,
			IsFinish: false,
		}
		if lastTmplData != nil {
			tmplData.TotalProfit = lastTmplData.TotalProfit
		}
		r.tmplDatas = append(r.tmplDatas, tmplData)
		// one round finish
		if longAmount == shortAmount {
			tmplData.IsFinish = true
			tmplData.Profit = common.FormatFloat(profit, 4)
			tmplData.ProfitRate = common.FormatFloat(profitRate, 4)
			if profit > 0 {
				profitArray = append(profitArray, profit)
				profitTotal = profitTotal.Add(decimal.NewFromFloat(profit))
			} else {
				loseArray = append(loseArray, profit)
				loseTotal = loseTotal.Add(decimal.NewFromFloat(profit))
			}
			tmplData.TotalProfit = common.FormatFloat(common.FloatAdd(tmplData.TotalProfit, tmplData.Profit), 4)
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
			r.balanceEnd = bal.Get()
		}
		lastTmplData = tmplData
	}
	r.result.TotalAction = len(r.tmplDatas)
	// endBalance - startBalance
	if total > 0 {
		r.result.WinRate = common.FormatFloat(common.FloatDiv(float64(success), float64(total)), 4)
	}
	if !loseTotal.IsZero() {
		r.result.ProfitFactor, _ = profitTotal.Div(loseTotal.Abs()).Float64()
		r.result.ProfitFactor = common.FormatFloat(r.result.ProfitFactor, 4)
	} else {
		r.result.ProfitFactor, _ = profitTotal.Float64()
		r.result.ProfitFactor = common.FormatFloat(r.result.ProfitFactor, 4)
	}
	if len(profitArray) >= 2 {
		r.result.ProfitVariance, err = stats.Variance(profitArray)
		if err != nil {
			return err
		}
	}
	r.result.ProfitVariance = common.FormatFloat(r.result.ProfitVariance, 4)
	if len(loseArray) >= 2 {
		r.result.LoseVariance, err = stats.Variance(loseArray)
		if err != nil {
			return err
		}
	}
	r.result.LoseVariance = common.FormatFloat(r.result.LoseVariance, 4)
	r.result.Actions = r.tmplDatas
	r.result.StartBalance = r.balanceInit
	r.result.EndBalance = common.FormatFloat(r.balanceEnd, 4)

	err = r.CalculateMetrics(&r.result)
	return err
}

// CalculateMetrics 计算所有指标
func (r *Report) CalculateMetrics(metrics *ReportResult) (err error) {
	if len(r.trades) == 0 || len(r.tmplDatas) == 0 {
		return
	}

	equity := r.CalculateEquityCurve()
	returns := r.calculateReturns()

	metrics.TotalProfit = common.FormatFloat(r.tmplDatas[len(r.tmplDatas)-1].TotalProfit, 4)
	metrics.TotalReturn = common.FormatFloat(metrics.TotalProfit/r.balanceInit, 4)
	metrics.AnnualReturn = common.FormatFloat(r.calculateAnnualReturn(equity), 4)
	metrics.MaxDrawdown = common.FormatFloat(r.calculateMaxDrawdown(equity), 4)
	metrics.ProfitFactor = common.FormatFloat(r.calculateProfitFactor(), 4)

	metrics.Volatility = common.FormatFloat(r.calculateVolatility(returns), 4)
	metrics.SharpeRatio = common.FormatFloat(r.calculateSharpeRatio(metrics.AnnualReturn, metrics.Volatility), 4)
	metrics.SortinoRatio = common.FormatFloat(r.calculateSortinoRatio(returns, metrics.AnnualReturn), 4)
	metrics.CalmarRatio = common.FormatFloat(r.calculateCalmarRatio(metrics.AnnualReturn, metrics.MaxDrawdown), 4)
	metrics.ConsistencyScore = common.FormatFloat(r.calculateConsistencyScore(returns), 4)
	metrics.SmoothnessScore = common.FormatFloat(r.calculateSmoothnessScore(equity), 4)

	// 统计交易数量
	metrics.LongTrades, metrics.ShortTrades = r.countTradeTypes()

	metrics.OverallScore = common.FormatFloat(r.calculateOverallScore(metrics), 4)
	return
}

// CalculateEquityCurve 计算净值曲线
func (r *Report) CalculateEquityCurve() []float64 {
	processedTrades := r.tmplDatas
	equity := make([]float64, len(processedTrades)+1)
	equity[0] = r.balanceInit

	for i, pt := range processedTrades {
		equity[i+1] = pt.Total
	}

	return equity
}

// calculateAnnualReturn 计算年化收益率
func (r *Report) calculateAnnualReturn(equity []float64) float64 {
	if len(equity) < 2 {
		return 0
	}
	// fmt.Println("startTime:", r.startTime)
	// fmt.Println("endTime:", r.endTime)
	// fmt.Println("equity[len(equity)-1]:", equity[len(equity)-1])
	// fmt.Println("equity[0]:", equity[0])
	totalReturn := equity[len(equity)-1]/equity[0] - 1
	// 计算回测期间的年数
	years := r.endTime.Sub(r.startTime).Hours() / (24 * 365.25)
	if years == 0 {
		years = 1
	}
	// fmt.Println("totalReturn:", totalReturn)
	// fmt.Println("years:", years)
	return math.Pow(1+totalReturn, 1/years) - 1
}
func (r *Report) calculateReturns() []float64 {
	returns := make([]float64, 0)
	for _, pt := range r.tmplDatas {
		if pt.IsFinish {
			returns = append(returns, pt.ProfitRate)
		}
	}
	return returns
}

// calculateProfitFactor 计算盈亏比
func (r *Report) calculateProfitFactor() float64 {
	var grossProfit, grossLoss float64

	for _, pt := range r.tmplDatas {
		if pt.IsFinish {
			if pt.Profit > 0 {
				grossProfit += pt.ProfitRate
			} else {
				grossLoss += math.Abs(pt.ProfitRate)
			}
		}
	}

	if grossLoss == 0 {
		if grossProfit > 0 {
			return math.Inf(1)
		}
		return 0
	}

	return grossProfit / grossLoss
}

// calculateVolatility 计算年化波动率
func (r *Report) calculateVolatility(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}

	var sum float64
	for _, r := range returns {
		sum += r
	}
	mean := sum / float64(len(returns))

	var variance float64
	for _, r := range returns {
		variance += (r - mean) * (r - mean)
	}
	variance /= float64(len(returns) - 1)

	stdDev := math.Sqrt(variance)

	// 年化波动率
	// 计算回测期间的年数
	years := r.endTime.Sub(r.startTime).Hours() / (24 * 365.25)
	if years == 0 {
		years = 1
	}
	tradesPerYear := float64(len(returns)) / years

	return stdDev * math.Sqrt(tradesPerYear)
}

// calculateSharpeRatio 计算夏普比率
func (r *Report) calculateSharpeRatio(annualReturn, volatility float64) float64 {
	if volatility == 0 {
		return 0
	}
	return (annualReturn - r.riskFreeRate) / volatility
}

// calculateSortinoRatio 计算索提诺比率
func (r *Report) calculateSortinoRatio(returns []float64, annualReturn float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	var downsideSum float64
	for _, r := range returns {
		if r < 0 {
			downsideSum += r * r
		}
	}

	downsideVariance := downsideSum / float64(len(returns))
	downsideStdDev := math.Sqrt(downsideVariance)

	if downsideStdDev == 0 {
		return 0
	}

	// 年化下行风险
	years := r.endTime.Sub(r.startTime).Hours() / (24 * 365.25)
	if years == 0 {
		years = 1
	}
	tradesPerYear := float64(len(returns)) / years
	annualDownsideRisk := downsideStdDev * math.Sqrt(tradesPerYear)

	return (annualReturn - r.riskFreeRate) / annualDownsideRisk
}

// calculateOverallScore 计算综合评分
func (r *Report) calculateOverallScore(metrics *ReportResult) float64 {
	scores := []float64{
		math.Max(0, metrics.SharpeRatio) / 2.0,
		math.Max(0, metrics.SortinoRatio) / 2.5,
		1.0 - metrics.MaxDrawdown,
		metrics.WinRate,
		math.Min(metrics.ProfitFactor/3.0, 1.0),
		math.Max(0, metrics.CalmarRatio) / 3.0,
		metrics.ConsistencyScore,
		metrics.SmoothnessScore,
	}

	var totalScore float64
	for _, score := range scores {
		totalScore += math.Min(score, 1.0)
	}

	return totalScore / float64(len(scores))
}

// calculateCalmarRatio 计算Calmar比率
func (r *Report) calculateCalmarRatio(annualReturn, maxDrawdown float64) float64 {
	if maxDrawdown == 0 {
		if annualReturn > 0 {
			return math.Inf(1)
		}
		return 0
	}
	return annualReturn / maxDrawdown
}

// calculateConsistencyScore 计算一致性评分
func (r *Report) calculateConsistencyScore(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}

	// 计算正收益的连续性
	positiveStreaks := 0
	maxPositiveStreak := 0
	currentStreak := 0

	for _, r := range returns {
		if r > 0 {
			currentStreak++
			if currentStreak > maxPositiveStreak {
				maxPositiveStreak = currentStreak
			}
		} else {
			positiveStreaks += currentStreak
			currentStreak = 0
		}
	}
	positiveStreaks += currentStreak

	// 计算月度胜率的一致性（简化版）
	monthlyWinRate := r.calculateWinRate() // 使用整体胜率作为近似

	streakScore := float64(maxPositiveStreak) / float64(len(returns))

	return (monthlyWinRate + streakScore) / 2
}

// calculateMaxDrawdown 计算最大回撤
func (r *Report) calculateMaxDrawdown(equity []float64) float64 {
	maxDrawdown := 0.0
	peak := equity[0]

	for i := 1; i < len(equity); i++ {
		if equity[i] > peak {
			peak = equity[i]
		}
		drawdown := (peak - equity[i]) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}

// calculateWinRate 计算胜率
func (r *Report) calculateWinRate() float64 {
	if len(r.tmplDatas) == 0 {
		return 0
	}

	wins := 0
	totalCloseTrades := 0
	for _, pt := range r.tmplDatas {
		if pt.IsFinish {
			totalCloseTrades++
			if pt.Profit > 0 {
				wins++
			}
		}
	}

	if totalCloseTrades == 0 {
		return 0
	}
	return float64(wins) / float64(totalCloseTrades)
}

// calculateSmoothnessScore 计算平滑度评分
func (r *Report) calculateSmoothnessScore(equity []float64) float64 {
	if len(equity) < 3 {
		return 0
	}

	// 计算收益曲线的二阶导数（加速度）来衡量平滑度
	var smoothnessSum float64
	for i := 1; i < len(equity)-1; i++ {
		acceleration := equity[i+1] - 2*equity[i] + equity[i-1]
		smoothnessSum += math.Abs(acceleration)
	}

	avgAcceleration := smoothnessSum / float64(len(equity)-2)

	// 计算回撤的严重程度
	drawdownSeverity := 1.0 - r.calculateMaxDrawdown(equity)

	// 结合加速度和回撤来计算平滑度
	maxPossibleAcceleration := r.balanceInit * 0.1 // 基于初始资本的10%
	smoothnessFromAcceleration := 1.0 - math.Min(avgAcceleration/maxPossibleAcceleration, 1.0)

	return (smoothnessFromAcceleration + drawdownSeverity) / 2
}

// countTradeTypes 统计交易类型数量
func (r *Report) countTradeTypes() (int, int) {
	longTrades := 0
	shortTrades := 0

	for _, pt := range r.tmplDatas {
		if pt.IsFinish {
			switch pt.Trade.Action {
			case CloseLong:
				longTrades++
			case CloseShort:
				shortTrades++
			}
		}
	}

	return longTrades, shortTrades
}

func (r *Report) WinRate() (rate float64) {
	rate = common.FormatFloat(r.result.WinRate, 2)
	return
}

func (r *Report) Profit() (profit float64) {
	profit = common.FormatFloat(r.result.TotalProfit, 4)
	return
}

func (r *Report) ProfitPercent() float64 {
	return common.FormatFloat((r.EndBalance()*100)/r.balanceInit, 4)
}

func (r *Report) ProfitVariance() float64 {
	return common.FormatFloat(r.result.ProfitVariance, 4)
}

func (r *Report) LoseVariance() float64 {
	return common.FormatFloat(r.result.LoseVariance, 4)
}

func (r *Report) EndBalance() float64 {
	return common.FormatFloat(r.balanceEnd, 4)
}

func (r *Report) ProfitLoseRatio() float64 {
	return common.FormatFloat(r.result.ProfitFactor, 4)
}

// MaxLose max total lose
func (r *Report) MaxLose() (lose float64) {
	lose = common.FormatFloat(r.maxLose, 2)
	return
}

// MaxDrawdown get max drawdown percent
func (r *Report) MaxDrawdown() float64 {
	return common.FormatFloat(r.result.MaxDrawdown, 2)
}

// MaxDrawdown get max drawdown value
func (r *Report) MaxDrawdownValue() float64 {
	return common.FormatFloat(r.result.MaxDrawdownValue, 4)
}

func (r *Report) GetReport() (report string) {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("Total action:%d\n", len(r.result.Actions)))
	buf.WriteString(fmt.Sprintf("Win rate:%f\n", r.result.WinRate))
	buf.WriteString(fmt.Sprintf("Profit:%f\n", r.result.TotalProfit))
	buf.WriteString(fmt.Sprintf("Max lose percent:%f\n", r.result.MaxLose))
	buf.WriteString(fmt.Sprintf("Max drawdown percent:%f%%\n", r.result.MaxDrawdown))
	buf.WriteString(fmt.Sprintf("Profit lose ratio: %f\n", r.result.ProfitFactor))
	buf.WriteString(fmt.Sprintf("StartBalance: %f\n", r.result.StartBalance))
	buf.WriteString(fmt.Sprintf("EndBalance: %f\n", r.result.EndBalance))
	buf.WriteString(fmt.Sprintf("ProfitPercent:%f\n", r.result.TotalReturn))
	buf.WriteString(fmt.Sprintf("ProfitVariance:%f\n", r.result.ProfitVariance))
	buf.WriteString(fmt.Sprintf("LoseVariance:%f\n", r.result.LoseVariance))
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
	err = tmpl.Execute(w, r.result)
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
	ret = r.result
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
