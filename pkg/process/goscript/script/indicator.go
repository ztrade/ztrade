// this file was generated by gomacro command: import "github.com/ztrade/indicator"
// DO NOT EDIT! Any change will be lost when the file is re-generated

package script

import (
	"github.com/cosmos72/gomacro/imports"
	indicator "github.com/ztrade/indicator"
	. "reflect"
)

// reflection: allow interpreted code to import "github.com/ztrade/indicator"
func init() {
	imports.Packages["github.com/ztrade/indicator"] = imports.Package{
		Name: "indicator",
		Binds: map[string]Value{
			"ExtraIndicators":    ValueOf(&indicator.ExtraIndicators).Elem(),
			"NewBoll":            ValueOf(indicator.NewBoll),
			"NewCommonIndicator": ValueOf(indicator.NewCommonIndicator),
			"NewCrossTool":       ValueOf(indicator.NewCrossTool),
			"NewEMA":             ValueOf(indicator.NewEMA),
			"NewJsonIndicator":   ValueOf(indicator.NewJsonIndicator),
			"NewMACD":            ValueOf(indicator.NewMACD),
			"NewMACDWithSMA":     ValueOf(indicator.NewMACDWithSMA),
			"NewMAGroup":         ValueOf(indicator.NewMAGroup),
			"NewMixed":           ValueOf(indicator.NewMixed),
			"NewRSI":             ValueOf(indicator.NewRSI),
			"NewSMA":             ValueOf(indicator.NewSMA),
			"NewSMMA":            ValueOf(indicator.NewSMMA),
			"NewStoch":           ValueOf(indicator.NewStoch),
			"NewStochRSI":        ValueOf(indicator.NewStochRSI),
			"RegisterIndicator":  ValueOf(indicator.RegisterIndicator),
		}, Types: map[string]Type{
			"Boll":                   TypeOf((*indicator.Boll)(nil)).Elem(),
			"CommonIndicator":        TypeOf((*indicator.CommonIndicator)(nil)).Elem(),
			"CrossTool":              TypeOf((*indicator.CrossTool)(nil)).Elem(),
			"Crosser":                TypeOf((*indicator.Crosser)(nil)).Elem(),
			"EMA":                    TypeOf((*indicator.EMA)(nil)).Elem(),
			"Indicator":              TypeOf((*indicator.Indicator)(nil)).Elem(),
			"JsonIndicator":          TypeOf((*indicator.JsonIndicator)(nil)).Elem(),
			"MABase":                 TypeOf((*indicator.MABase)(nil)).Elem(),
			"MACD":                   TypeOf((*indicator.MACD)(nil)).Elem(),
			"MAGroup":                TypeOf((*indicator.MAGroup)(nil)).Elem(),
			"Mixed":                  TypeOf((*indicator.Mixed)(nil)).Elem(),
			"NewCommonIndicatorFunc": TypeOf((*indicator.NewCommonIndicatorFunc)(nil)).Elem(),
			"RSI":                    TypeOf((*indicator.RSI)(nil)).Elem(),
			"SMA":                    TypeOf((*indicator.SMA)(nil)).Elem(),
			"SMMA":                   TypeOf((*indicator.SMMA)(nil)).Elem(),
			"Stoch":                  TypeOf((*indicator.Stoch)(nil)).Elem(),
			"StochRSI":               TypeOf((*indicator.StochRSI)(nil)).Elem(),
			"Updater":                TypeOf((*indicator.Updater)(nil)).Elem(),
		}, Proxies: map[string]Type{
			"CommonIndicator": TypeOf((*P_CommonIndicator)(nil)).Elem(),
			"Crosser":         TypeOf((*P_Crosser)(nil)).Elem(),
			"Indicator":       TypeOf((*P_Indicator)(nil)).Elem(),
			"Updater":         TypeOf((*P_Updater)(nil)).Elem(),
		}, Wrappers: map[string][]string{
			"EMA":  []string{"Result"},
			"SMA":  []string{"Result"},
			"SMMA": []string{"Result"},
		},
	}
}

// --------------- proxy for github.com/ztrade/indicator.CommonIndicator ---------------
type P_CommonIndicator struct {
	Object     interface{}
	Indicator_ func(interface{}) map[string]float64
	Result_    func(interface{}) float64
	Update_    func(_proxy_obj_ interface{}, price float64)
}

func (P *P_CommonIndicator) Indicator() map[string]float64 {
	return P.Indicator_(P.Object)
}
func (P *P_CommonIndicator) Result() float64 {
	return P.Result_(P.Object)
}
func (P *P_CommonIndicator) Update(price float64) {
	P.Update_(P.Object, price)
}

// --------------- proxy for github.com/ztrade/indicator.Crosser ---------------
type P_Crosser struct {
	Object      interface{}
	FastResult_ func(interface{}) float64
	SlowResult_ func(interface{}) float64
	Update_     func(_proxy_obj_ interface{}, price float64)
}

func (P *P_Crosser) FastResult() float64 {
	return P.FastResult_(P.Object)
}
func (P *P_Crosser) SlowResult() float64 {
	return P.SlowResult_(P.Object)
}
func (P *P_Crosser) Update(price float64) {
	P.Update_(P.Object, price)
}

// --------------- proxy for github.com/ztrade/indicator.Indicator ---------------
type P_Indicator struct {
	Object  interface{}
	Result_ func(interface{}) float64
	Update_ func(_proxy_obj_ interface{}, price float64)
}

func (P *P_Indicator) Result() float64 {
	return P.Result_(P.Object)
}
func (P *P_Indicator) Update(price float64) {
	P.Update_(P.Object, price)
}

// --------------- proxy for github.com/ztrade/indicator.Updater ---------------
type P_Updater struct {
	Object  interface{}
	Update_ func(_proxy_obj_ interface{}, price float64)
}

func (P *P_Updater) Update(price float64) {
	P.Update_(P.Object, price)
}