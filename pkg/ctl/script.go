package ctl

import (
	"path"
	"ztrade/pkg/event"
	"ztrade/pkg/process/goscript"
)

type Scripter interface {
	event.Processer
	AddScript(name, src string, param map[string]interface{}) (err error)
	RemoveScript(name string) error
}

func NewScript(file string, param map[string]interface{}) (s Scripter, err error) {
	var gEngine *goscript.GoEngine
	gEngine, err = goscript.NewDefaultGoEngine()
	if err != nil {
		return
	}
	s = gEngine
	err = s.AddScript(path.Base(file), file, param)
	return
}
