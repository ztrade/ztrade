package ctl

import (
	"path"

	"github.com/ztrade/ztrade/pkg/event"
	"github.com/ztrade/ztrade/pkg/process/goscript"
)

type Scripter interface {
	event.Processer
	AddScript(name, src, param string) (err error)
	RemoveScript(name string) error
	ScriptCount() int
}

func NewScript(file, param, symbol string) (s Scripter, err error) {
	var gEngine *goscript.GoEngine
	gEngine, err = goscript.NewGoEngine(symbol)
	if err != nil {
		return
	}
	s = gEngine
	err = s.AddScript(path.Base(file), file, param)
	return
}
