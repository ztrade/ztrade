package igo

import (
	"github.com/ztrade/ztrade/pkg/process/goscript/engine"
)

func init() {
	engine.Register(".go", NewRunner)
	engine.Register(".gop", NewRunner)
}
