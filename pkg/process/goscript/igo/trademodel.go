// export by github.com/goplus/igop/cmd/qexp

package igo

import (
	q "github.com/ztrade/trademodel"

	"go/constant"
	"reflect"

	"github.com/goplus/igop"
)

func init() {
	igop.RegisterPackage(&igop.Package{
		Name: "trademodel",
		Path: "github.com/ztrade/trademodel",
		Deps: map[string]string{
			"fmt":  "fmt",
			"time": "time",
		},
		Interfaces: map[string]reflect.Type{},
		NamedTypes: map[string]reflect.Type{
			"Balance":     reflect.TypeOf((*q.Balance)(nil)).Elem(),
			"Candle":      reflect.TypeOf((*q.Candle)(nil)).Elem(),
			"CandleList":  reflect.TypeOf((*q.CandleList)(nil)).Elem(),
			"Currency":    reflect.TypeOf((*q.Currency)(nil)).Elem(),
			"Depth":       reflect.TypeOf((*q.Depth)(nil)).Elem(),
			"DepthInfo":   reflect.TypeOf((*q.DepthInfo)(nil)).Elem(),
			"Order":       reflect.TypeOf((*q.Order)(nil)).Elem(),
			"Orderbook":   reflect.TypeOf((*q.Orderbook)(nil)).Elem(),
			"Position":    reflect.TypeOf((*q.Position)(nil)).Elem(),
			"Symbol":      reflect.TypeOf((*q.Symbol)(nil)).Elem(),
			"Ticker":      reflect.TypeOf((*q.Ticker)(nil)).Elem(),
			"Trade":       reflect.TypeOf((*q.Trade)(nil)).Elem(),
			"TradeAction": reflect.TypeOf((*q.TradeAction)(nil)).Elem(),
			"TradeType":   reflect.TypeOf((*q.TradeType)(nil)).Elem(),
		},
		AliasTypes: map[string]reflect.Type{},
		Vars: map[string]reflect.Value{
			"OrderStatusCanceled": reflect.ValueOf(&q.OrderStatusCanceled),
			"OrderStatusFilled":   reflect.ValueOf(&q.OrderStatusFilled),
		},
		Funcs: map[string]reflect.Value{
			"NewTradeType": reflect.ValueOf(q.NewTradeType),
		},
		TypedConsts: map[string]igop.TypedConst{
			"CancelAll":   {reflect.TypeOf(q.CancelAll), constant.MakeInt64(int64(q.CancelAll))},
			"CancelOne":   {reflect.TypeOf(q.CancelOne), constant.MakeInt64(int64(q.CancelOne))},
			"Close":       {reflect.TypeOf(q.Close), constant.MakeInt64(int64(q.Close))},
			"CloseLong":   {reflect.TypeOf(q.CloseLong), constant.MakeInt64(int64(q.CloseLong))},
			"CloseShort":  {reflect.TypeOf(q.CloseShort), constant.MakeInt64(int64(q.CloseShort))},
			"DirectLong":  {reflect.TypeOf(q.DirectLong), constant.MakeInt64(int64(q.DirectLong))},
			"DirectShort": {reflect.TypeOf(q.DirectShort), constant.MakeInt64(int64(q.DirectShort))},
			"Limit":       {reflect.TypeOf(q.Limit), constant.MakeInt64(int64(q.Limit))},
			"Market":      {reflect.TypeOf(q.Market), constant.MakeInt64(int64(q.Market))},
			"Open":        {reflect.TypeOf(q.Open), constant.MakeInt64(int64(q.Open))},
			"OpenLong":    {reflect.TypeOf(q.OpenLong), constant.MakeInt64(int64(q.OpenLong))},
			"OpenShort":   {reflect.TypeOf(q.OpenShort), constant.MakeInt64(int64(q.OpenShort))},
			"Stop":        {reflect.TypeOf(q.Stop), constant.MakeInt64(int64(q.Stop))},
			"StopLong":    {reflect.TypeOf(q.StopLong), constant.MakeInt64(int64(q.StopLong))},
			"StopShort":   {reflect.TypeOf(q.StopShort), constant.MakeInt64(int64(q.StopShort))},
		},
		UntypedConsts: map[string]igop.UntypedConst{
			"Long":  {"untyped int", constant.MakeInt64(int64(q.Long))},
			"Short": {"untyped int", constant.MakeInt64(int64(q.Short))},
		},
	})
}
