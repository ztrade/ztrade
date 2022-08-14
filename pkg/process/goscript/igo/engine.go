// export by github.com/goplus/igop/cmd/qexp

package igo

import (
	q "github.com/ztrade/base/engine"

	"go/constant"
	"reflect"

	"github.com/goplus/igop"
)

func init() {
	igop.RegisterPackage(&igop.Package{
		Name: "engine",
		Path: "github.com/ztrade/base/engine",
		Deps: map[string]string{
			"github.com/ztrade/base/common": "common",
			"github.com/ztrade/indicator":   "indicator",
			"github.com/ztrade/trademodel":  "trademodel",
		},
		Interfaces: map[string]reflect.Type{
			"Engine": reflect.TypeOf((*q.Engine)(nil)).Elem(),
		},
		NamedTypes:  map[string]reflect.Type{},
		AliasTypes:  map[string]reflect.Type{},
		Vars:        map[string]reflect.Value{},
		Funcs:       map[string]reflect.Value{},
		TypedConsts: map[string]igop.TypedConst{},
		UntypedConsts: map[string]igop.UntypedConst{
			"StatusFail":    {"untyped int", constant.MakeInt64(int64(q.StatusFail))},
			"StatusRunning": {"untyped int", constant.MakeInt64(int64(q.StatusRunning))},
			"StatusSuccess": {"untyped int", constant.MakeInt64(int64(q.StatusSuccess))},
		},
	})
}
