package define

const (
	Long  = 1
	Short = 2
)

type Position struct {
	Symbol      string
	Type        int     // 合约类型，Long: 多头，Short: 空头
	Hold        float64 // 持有仓位
	Price       float64 //开仓价格
	ProfitRatio float64 // 盈利比例,正数表示盈利，负数表示亏岁
}
