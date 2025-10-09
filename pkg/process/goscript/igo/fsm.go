// export by github.com/goplus/ixgo/cmd/qexp

package igo

import (
	q "github.com/ztrade/base/fsm"

	"reflect"

	"github.com/goplus/ixgo"
)

func init() {
	ixgo.RegisterPackage(&ixgo.Package{
		Name: "fsm",
		Path: "github.com/ztrade/base/fsm",
		Deps: map[string]string{
			"fmt": "fmt",
		},
		Interfaces: map[string]reflect.Type{},
		NamedTypes: map[string]reflect.Type{
			"Callback":  reflect.TypeOf((*q.Callback)(nil)).Elem(),
			"EventDesc": reflect.TypeOf((*q.EventDesc)(nil)).Elem(),
			"FSM":       reflect.TypeOf((*q.FSM)(nil)).Elem(),
			"Rule":      reflect.TypeOf((*q.Rule)(nil)).Elem(),
		},
		AliasTypes: map[string]reflect.Type{},
		Vars:       map[string]reflect.Value{},
		Funcs: map[string]reflect.Value{
			"NewFSM": reflect.ValueOf(q.NewFSM),
		},
		TypedConsts:   map[string]ixgo.TypedConst{},
		UntypedConsts: map[string]ixgo.UntypedConst{},
	})
}
