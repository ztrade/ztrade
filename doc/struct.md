# Trademodel

- English: [struct.md](struct.md)
- 中文: [struct_cn.md](struct_cn.md)

`trademodel` defines commonly used trading data models.

## Candle definition
```
// One candle record
type Candle struct {
	ID       int64   `xorm:"pk autoincr null 'id'"`     // DB row id. ID=-1 means history replay; otherwise realtime candle.
	Start    int64   `xorm:"unique index 'start'"`      // Start timestamp in milliseconds
	Open     float64 `xorm:"notnull 'open'"`            // Open price
	High     float64 `xorm:"notnull 'high'"`            // High price
	Low      float64 `xorm:"notnull 'low'"`             // Low price
	Close    float64 `xorm:"notnull 'close'"`           // Close price
	Volume   float64 `xorm:"notnull 'volume'"`          // Volume
	Turnover float64 `xorm:"turnover 'turnover'"`       // Turnover
	Trades   int64   `xorm:"notnull 'trades'"`          // Number of trades
	Table    string  `xorm:"-"`                         // Table name in DB
}
```

## Trade definition
```
// Order action type
type TradeType int

const (
	CancelOne   TradeType = -2          // Cancel one order
	CancelAll   TradeType = -1          // Cancel all orders
	DirectLong  TradeType = 1           // Long direction
	DirectShort TradeType = 1 << 1      // Short direction

	Limit  TradeType = 1 << 3            // Limit order
	Market TradeType = 1 << 4            // Market order
	Stop   TradeType = 1 << 5            // Stop order

	Open  TradeType = 1 << 6             // Open position
	Close TradeType = 1 << 7             // Close position

	OpenLong   = Open | DirectLong       // Open long
	OpenShort  = Open | DirectShort      // Open short
	CloseLong  = Close | DirectLong      // Close long
	CloseShort = Close | DirectShort     // Close short
	StopLong   = Stop | DirectLong       // Long stop order
	StopShort  = Stop | DirectShort      // Short stop order
)

// Filled trade data
type Trade struct {
	ID     string       // Exchange trade id
	Action TradeType    // Order action type
	Time   time.Time    // Fill time
	Price  float64      // Fill price
	Amount float64      // Fill amount
	Side   string       // Side: long or short
	Remark string       // Remark; contains error text when order fails
}
```

## Order book definition
```
// One level in order book
type DepthInfo struct {
	Price  float64
	Amount float64
}

// Depth data alias
type Depth Orderbook

// Order book snapshot
type Orderbook struct {
	Sells      []DepthInfo   // Asks
	Buys       []DepthInfo   // Bids
	UpdateTime time.Time     // Last update time
}

```