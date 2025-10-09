// export by github.com/goplus/ixgo/cmd/qexp

package igo

import (
	q "github.com/ztrade/base/engine"

	"go/constant"
	"reflect"

	"github.com/goplus/ixgo"
)

func init() {
	ixgo.RegisterPackage(&ixgo.Package{
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
		TypedConsts: map[string]ixgo.TypedConst{},
		UntypedConsts: map[string]ixgo.UntypedConst{
			"StatusFail":    {"untyped int", constant.MakeInt64(int64(q.StatusFail))},
			"StatusRunning": {"untyped int", constant.MakeInt64(int64(q.StatusRunning))},
			"StatusSuccess": {"untyped int", constant.MakeInt64(int64(q.StatusSuccess))},
		},
	})
}
