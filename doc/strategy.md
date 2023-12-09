# 策略说明

## 创建策略项目
创建策略项目可以直接使用模板，也可以全部手动创建
### 直接使用模板
```
git clone https://github.com/ztrade/strategy
cd strategy
# 添加自定义策略
```

### 手动创建项目

```
mkdir strategy
cd strategy
go mod init strategy
# 复制 helper.go，并且修改package
cp $SRC/pkg/helper/helper.go ./define.go
# 创建新的策略
touch demo.go
```


## 策略
没一个策略，都是一个go的struct，这个struct一般长这个样子:

```
package strategy

import (
	. "github.com/ztrade/trademodel"
)

type Demo struct {
	engine Engine // 这个是和引擎交互用的

	position float64 //仓位

	strParam   string
	intParam   int
	floatParam float64
}

// 这个是策略的创建函数，必须是下面这种格式： func  New{struct}() *{struct}
func NewDemo() *Demo {
	return new(Demo)
}

// 这个函数提供策略需要的参数列表，供ztrade引擎调用
// ztrade 可以通过在命令行直接输入参数的方式，来传递参数
func (d *Demo) Param() []Param {
	return []Param{
		// 这个函数有 5个参数:
		// 1. 这个Param的key
		// 2. 这个Param的中文简称
		// 3. 这个Param的具体解释
		// 4. 这个Param的默认值
		// 5. 这个Param对应的变量的的指针
		StringParam("str", "字符串参数", "只是一个简单的参数", "15m", &d.strParam),
		IntParam("intparam", "数字参数", "一个简单的数字参数 ", 12, &d.intParam),
		FloatParam("floatparam", "浮点参数", "简单的浮点参数", 1, &d.floatParam),
	}
}

// 这个函数是在策略初始化的时候调用的
// engine就是ztrade引擎的接口
// params是传入的参数信息
func (d *Demo) Init(engine Engine, params ParamData) {
	// 这里 d.strParam,d.intParam,d.floatParam已经自动解析了，无需再次解析
	d.engine = engine
	// 合并K线，第一个参数是原始K线级别，第二个参数是目标级别，第三个参数是K线合并完成后的回调函数
	engine.Merge("1m", "30m", d.OnCandle30m)
	// 合并K线，第一个参数是原始K线级别，第二个参数是目标级别，第三个参数是K线合并完成后的回调函数
	engine.Merge("1m", "1h", d.OnCandle1h)
}

// 1m K线回调函数
// 在回测中,candle.ID是数据库中的ID
// 在实盘中 candle.ID=-1表示是历史数据,非-1表示正常的实时K线
func (d *Demo) OnCandle(candle *Candle) {
}

// 仓位同步函数， pos 表示仓位， 正数表示多仓，负数表示空仓，price是开仓的价格
func (d *Demo) OnPosition(pos, price float64) {
	d.position = pos
}

// 自己的订单成交时候的回调函数
func (d *Demo) OnTrade(trade *Trade) {

}

// 交易所中实时的成交信息
func (d *Demo) OnTradeMarket(trade *Trade) {

}

// 交易所中实时推送的深度信息，根据交易所限制不同，深度信息也不同
func (d *Demo) OnDepth(depth *Depth) {
}

// Init函数中定义的 30m K线回调函数
// 在回测中,candle.ID是数据库中的ID
// 在实盘中 candle.ID=-1表示是历史数据,非-1表示正常的实时K线
func (d *Demo) OnCandle30m(candle *Candle) {
}

// Init函数中定义的 1h K线回调函数
// 在回测中,candle.ID是数据库中的ID
// 在实盘中 candle.ID=-1表示是历史数据,非-1表示正常的实时K线
func (d *Demo) OnCandle1h(candle *Candle) {
	// 自定义判断逻辑
	//...

	// 可以在策略启动后的任何地方调用交易函数
	// OpenLong,CloseLong,OpenShort,CloseShort,StopLong,StopShort...
	d.engine.OpenLong(candle.Close, 1)
}

```

## Engine说明
Engine的定义如下:

```
const (
    // 策略运行中
	StatusRunning = 0
    // 策略成功
	StatusSuccess = 1
    // 策略失败
	StatusFail    = -1
)

type Engine interface {
    // 开多，返回order id
	OpenLong(price, amount float64) string
    // 平多，返回order id
	CloseLong(price, amount float64) string
    // 开空，返回order id
	OpenShort(price, amount float64) string
    // 平空，返回order id
	CloseShort(price, amount float64) string
    // 发送 多单止损 订单，返回order id
	StopLong(price, amount float64) string
    // 发送 空单止损 订单，返回order id
	StopShort(price, amount float64) string
    // 取消某个订单，参数是order id
	CancelOrder(string)
    // 取消所有订单
	CancelAllOrder()
    // 执行订单，一般调用上面的就可以了，不用调用这个
	DoOrder(typ trademodel.TradeType, price, amount float64) string
    // 添加指标，指标具体文档在下面
	AddIndicator(name string, params ...int) (ind indicator.CommonIndicator)
    // 获取当前的仓位
	Position() (pos, price float64)
    // 获取当前的余额
	Balance() float64
    // 日志
	Log(v ...interface{})
    // 添加新的订阅事件，当前无需调用
	Watch(watchType string)
    // 发送消息通知，需要添加消息类型的processer才会生效
	SendNotify(title, content, contentType string)
    // 合并K线，src是原始级别，这里固定是1m,dst是目标级别，fn是回调函数
	Merge(src, dst string, fn common.CandleFn)
    // 设置余额，仅在回测时有用
	SetBalance(balance float64)
    // 更新状态，一般无需调用，状态说明见上面的定义
	UpdateStatus(status int, msg string)
}

```

## 指标说明
ztrade内置了一些常见的指标，代码详见 [indicator](https://github.com/ztrade/indicator)

| 名称     | 说明                      | 参数                                              | 例子                                                      |
|----------|---------------------------|---------------------------------------------------|-----------------------------------------------------------|
| EMA      | 只有一个参数表示是一根EMA | 数字                                              | AddIndicator("EMA", 9) 长度为9的EMA                       |
| EMA      | 两个参数表示EMA交叉指标   | 两个数字：快线、慢线                              | AddIndicator("EMA", 9, 26) 长度为9的EMA和长度为26的EMA    |
| MACD     | 标准的MACD                | 三个数字:快线、慢线、dea                          | AddIndicator("macd",12,26,9)                              |
| SMAMACD  | 使用SMA代替EMA计算的MACD  | 三个数字:快线、慢线、dea                          | AddIndicator("macd",12,26,9)                              |
| SMA      | 只有一个参数表示一根SMA   | 数字                                              | AddIndicator("SMA", 9) 长度为9的SMA                       |
| SMA      | 两个参数表示SMA交叉指标   | 两个数字：快线、慢线                              | AddIndicator("SMA", 9, 26) 长度为9的SMA和长度为26的SMA    |
| SSMA     | 只有一个参数表示一根SSMA  | 数字                                              | AddIndicator("SSMA", 9) 长度为9的SSMA                     |
| SSMA     | 两个参数表示SSMA交叉指标  | 两个数字：快线、慢线                              | AddIndicator("SSMA", 9, 26) 长度为9的SSMA和长度为26的SSMA |
| STOCHRSI | 随机相对强弱指数          | 4个参数：STOCH窗口长度、RSI窗口长度、平滑k、平滑d | AddIndicator("STOCHRSI", 14,14,3,3)                       |
| RSI      | 只有一个参数表示一根RSI   | 数字                                              | AddIndicator("RSI", 9)表示一根长度是9的RSI                |
| RSI      | 两个参数表示RSI交叉指标   | 两个数字:快线、慢线                               | AddIndicator("RSI", 9, 26)长度为9的RSI和长度为26的RSI     |
| BOLL     | BOLL指标                  | 两个参数：长度、多元                              | AddIndicator("BOLL", 20,2）                               |

### 返回值 CommonIndicator 说明

1. 如果是单根指标线，Result()返回当前的值，如果是两个指标线，则返回的是快线的当前值
2. Indicator() 返回一个map
针对有两根线的 EMA/SMA/SSMA/RSI
```
result: 同Result()的值
fast: 快线的值
slow: 慢线的值
crossUp: 1 表示金叉， 0表示没有金叉
crossDown: 1 表示死叉， 0 表示没有死叉
```

boll指标

```
result: 同Result()的值,表示中间的均线的值
top: 上边线的值
bottom: 下边线的值
```


MACD/SMAMACD/STOCHRSI用这种方法只能获取到Result的值，建议直接使用 NewMACD/NewMACDWithSMA/NewSTOCHRSI方法
