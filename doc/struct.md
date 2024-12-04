# trademodel
trademodel定义了常用的一些交易相关的model

K线定义
```
// 每一根K线数据
type Candle struct {
	ID       int64   `xorm:"pk autoincr null 'id'"`     // 数据库中ID, ID=-1表示是历史数据,非-1表示正常的实时K线
	Start    int64   `xorm:"unique index 'start'"`      // 开始时间戳，单位是毫秒
	Open     float64 `xorm:"notnull 'open'"`            //    开盘价
	High     float64 `xorm:"notnull 'high'"`            //    最高价
	Low      float64 `xorm:"notnull 'low'"`             //    最低价
	Close    float64 `xorm:"notnull 'close'"`           //    收盘价
	Volume   float64 `xorm:"notnull 'volume'"`          //    成交量
	Turnover float64 `xorm:"turnover 'turnover'"`       //    成交额
	Trades   int64   `xorm:"notnull 'trades'"`          //    成交笔数
	Table    string  `xorm:"-"`                         // 数据库中表名
}
```

订单定义
```
// 订单类型
type TradeType int

const (
	CancelOne   TradeType = -2          //  取消一个订单
	CancelAll   TradeType = -1          //  取消所有订单
	DirectLong  TradeType = 1           //  做多
	DirectShort TradeType = 1 << 1      //  做空

	Limit  TradeType = 1 << 3            // 限价单
	Market TradeType = 1 << 4            // 市价单
	Stop   TradeType = 1 << 5            // 止损单

	Open  TradeType = 1 << 6             // 开仓
	Close TradeType = 1 << 7             // 平仓

	OpenLong   = Open | DirectLong       // 开多
	OpenShort  = Open | DirectShort      // 开空
	CloseLong  = Close | DirectLong      // 平多
	CloseShort = Close | DirectShort     // 平空
	StopLong   = Stop | DirectLong       // 多单止损单
	StopShort  = Stop | DirectShort      // 空单止损单
)

// 每一个成交单数据
type Trade struct {
	ID     string       //   交易所成交单的ID
	Action TradeType    //   订单类型
	Time   time.Time    //   成交时间
	Price  float64      //   成交价格
	Amount float64      //   成交数量
	Side   string       //   成交方向, long: 多仓, short: 空仓
	Remark string       //   备注,如果订单失败,remark中包含error字样
}
```

订单薄定义
```
// 订单薄中的每一个挂单
type DepthInfo struct {
	Price  float64
	Amount float64
}

// 订单薄数据
type Depth Orderbook

// 订单薄数据
type Orderbook struct {
	Sells      []DepthInfo   // 卖单
	Buys       []DepthInfo   // 买单
	UpdateTime time.Time     // 最新更新时间
}

```