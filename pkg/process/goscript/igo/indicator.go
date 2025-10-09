// export by github.com/goplus/ixgo/cmd/qexp

package igo

import (
	q "github.com/ztrade/indicator"

	"reflect"

	"github.com/goplus/ixgo"
)

func init() {
	ixgo.RegisterPackage(&ixgo.Package{
		Name: "indicator",
		Path: "github.com/ztrade/indicator",
		Deps: map[string]string{
			"encoding/json":                 "json",
			"fmt":                           "fmt",
			"github.com/shopspring/decimal": "decimal",
			"math":                          "math",
			"reflect":                       "reflect",
			"strings":                       "strings",
		},
		Interfaces: map[string]reflect.Type{
			"CommonIndicator": reflect.TypeOf((*q.CommonIndicator)(nil)).Elem(),
			"Crosser":         reflect.TypeOf((*q.Crosser)(nil)).Elem(),
			"Indicator":       reflect.TypeOf((*q.Indicator)(nil)).Elem(),
			"Updater":         reflect.TypeOf((*q.Updater)(nil)).Elem(),
		},
		NamedTypes: map[string]reflect.Type{
			"Boll":                   reflect.TypeOf((*q.Boll)(nil)).Elem(),
			"CrossTool":              reflect.TypeOf((*q.CrossTool)(nil)).Elem(),
			"EMA":                    reflect.TypeOf((*q.EMA)(nil)).Elem(),
			"JsonIndicator":          reflect.TypeOf((*q.JsonIndicator)(nil)).Elem(),
			"MABase":                 reflect.TypeOf((*q.MABase)(nil)).Elem(),
			"MACD":                   reflect.TypeOf((*q.MACD)(nil)).Elem(),
			"MAGroup":                reflect.TypeOf((*q.MAGroup)(nil)).Elem(),
			"Mixed":                  reflect.TypeOf((*q.Mixed)(nil)).Elem(),
			"NewCommonIndicatorFunc": reflect.TypeOf((*q.NewCommonIndicatorFunc)(nil)).Elem(),
			"RSI":                    reflect.TypeOf((*q.RSI)(nil)).Elem(),
			"SMA":                    reflect.TypeOf((*q.SMA)(nil)).Elem(),
			"SMMA":                   reflect.TypeOf((*q.SMMA)(nil)).Elem(),
			"Stoch":                  reflect.TypeOf((*q.Stoch)(nil)).Elem(),
			"StochRSI":               reflect.TypeOf((*q.StochRSI)(nil)).Elem(),
		},
		AliasTypes: map[string]reflect.Type{},
		Vars: map[string]reflect.Value{
			"ExtraIndicators": reflect.ValueOf(&q.ExtraIndicators),
		},
		Funcs: map[string]reflect.Value{
			"NewBoll":            reflect.ValueOf(q.NewBoll),
			"NewCommonIndicator": reflect.ValueOf(q.NewCommonIndicator),
			"NewCrossTool":       reflect.ValueOf(q.NewCrossTool),
			"NewEMA":             reflect.ValueOf(q.NewEMA),
			"NewJsonIndicator":   reflect.ValueOf(q.NewJsonIndicator),
			"NewMACD":            reflect.ValueOf(q.NewMACD),
			"NewMACDWithSMA":     reflect.ValueOf(q.NewMACDWithSMA),
			"NewMAGroup":         reflect.ValueOf(q.NewMAGroup),
			"NewMixed":           reflect.ValueOf(q.NewMixed),
			"NewRSI":             reflect.ValueOf(q.NewRSI),
			"NewSMA":             reflect.ValueOf(q.NewSMA),
			"NewSMMA":            reflect.ValueOf(q.NewSMMA),
			"NewStoch":           reflect.ValueOf(q.NewStoch),
			"NewStochRSI":        reflect.ValueOf(q.NewStochRSI),
			"RegisterIndicator":  reflect.ValueOf(q.RegisterIndicator),
		},
		TypedConsts:   map[string]ixgo.TypedConst{},
		UntypedConsts: map[string]ixgo.UntypedConst{},
	})
}
