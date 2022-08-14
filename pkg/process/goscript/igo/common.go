// export by github.com/goplus/igop/cmd/qexp

package igo

import (
	q "github.com/ztrade/base/common"

	"go/constant"
	"reflect"

	"github.com/goplus/igop"
)

func init() {
	igop.RegisterPackage(&igop.Package{
		Name: "common",
		Path: "github.com/ztrade/base/common",
		Deps: map[string]string{
			"bufio":                          "bufio",
			"errors":                         "errors",
			"fmt":                            "fmt",
			"github.com/bitly/go-simplejson": "simplejson",
			"github.com/shopspring/decimal":  "decimal",
			"github.com/sirupsen/logrus":     "logrus",
			"github.com/ztrade/trademodel":   "trademodel",
			"io":                             "io",
			"os":                             "os",
			"os/exec":                        "exec",
			"path/filepath":                  "filepath",
			"regexp":                         "regexp",
			"runtime":                        "runtime",
			"strconv":                        "strconv",
			"strings":                        "strings",
			"time":                           "time",
		},
		Interfaces: map[string]reflect.Type{},
		NamedTypes: map[string]reflect.Type{
			"CandleFn":   reflect.TypeOf((*q.CandleFn)(nil)).Elem(),
			"Entry":      reflect.TypeOf((*q.Entry)(nil)).Elem(),
			"KlineMerge": reflect.TypeOf((*q.KlineMerge)(nil)).Elem(),
			"Param":      reflect.TypeOf((*q.Param)(nil)).Elem(),
			"ParamData":  reflect.TypeOf((*q.ParamData)(nil)).Elem(),
			"VBalance":   reflect.TypeOf((*q.VBalance)(nil)).Elem(),
		},
		AliasTypes: map[string]reflect.Type{},
		Vars: map[string]reflect.Value{
			"Day":          reflect.ValueOf(&q.Day),
			"ErrNoBalance": reflect.ValueOf(&q.ErrNoBalance),
			"Week":         reflect.ValueOf(&q.Week),
		},
		Funcs: map[string]reflect.Value{
			"BoolParam":          reflect.ValueOf(q.BoolParam),
			"Copy":               reflect.ValueOf(q.Copy),
			"CopyWithMainPkg":    reflect.ValueOf(q.CopyWithMainPkg),
			"FloatAdd":           reflect.ValueOf(q.FloatAdd),
			"FloatDiv":           reflect.ValueOf(q.FloatDiv),
			"FloatMul":           reflect.ValueOf(q.FloatMul),
			"FloatParam":         reflect.ValueOf(q.FloatParam),
			"FloatSub":           reflect.ValueOf(q.FloatSub),
			"FormatFloat":        reflect.ValueOf(q.FormatFloat),
			"GetBinSizeDuration": reflect.ValueOf(q.GetBinSizeDuration),
			"GetExecDir":         reflect.ValueOf(q.GetExecDir),
			"IntParam":           reflect.ValueOf(q.IntParam),
			"MergeKlineChan":     reflect.ValueOf(q.MergeKlineChan),
			"NewKlineMerge":      reflect.ValueOf(q.NewKlineMerge),
			"NewKlineMergeStr":   reflect.ValueOf(q.NewKlineMergeStr),
			"NewVBalance":        reflect.ValueOf(q.NewVBalance),
			"OpenURL":            reflect.ValueOf(q.OpenURL),
			"ParseBinSizes":      reflect.ValueOf(q.ParseBinSizes),
			"ParseBinStrs":       reflect.ValueOf(q.ParseBinStrs),
			"ParseParams":        reflect.ValueOf(q.ParseParams),
			"StringParam":        reflect.ValueOf(q.StringParam),
		},
		TypedConsts: map[string]igop.TypedConst{},
		UntypedConsts: map[string]igop.UntypedConst{
			"DefaultBinSizes": {"untyped string", constant.MakeString(string(q.DefaultBinSizes))},
		},
	})
}
