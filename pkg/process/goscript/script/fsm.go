// this file was generated by gomacro command: import "github.com/ztrade/base/fsm"
// DO NOT EDIT! Any change will be lost when the file is re-generated

package script

import (
	. "reflect"
	fsm "github.com/ztrade/base/fsm"
	"github.com/cosmos72/gomacro/imports"
)

// reflection: allow interpreted code to import "github.com/ztrade/base/fsm"
func init() {
	imports.Packages["github.com/ztrade/base/fsm"] = imports.Package{
		Name: "fsm",
		Binds: map[string]Value{
			"NewFSM":	ValueOf(fsm.NewFSM),
		}, Types: map[string]Type{
			"Callback":	TypeOf((*fsm.Callback)(nil)).Elem(),
			"EventDesc":	TypeOf((*fsm.EventDesc)(nil)).Elem(),
			"FSM":	TypeOf((*fsm.FSM)(nil)).Elem(),
			"Rule":	TypeOf((*fsm.Rule)(nil)).Elem(),
		},
	}
}
