package script

import (
	"fmt"

	"github.com/ztrade/base/common"
	bengine "github.com/ztrade/base/engine"
	. "github.com/ztrade/ztrade/pkg/event"
	"github.com/ztrade/ztrade/pkg/process/goscript/engine"

	. "github.com/SuperGod/trademodel"
	"github.com/cosmos72/gomacro/fast"
)

type Runner struct {
	p    *fast.Interp
	info *CallInfo
}

func NewRunnerExport(file string) (r engine.Runner, err error) {
	temp, err := NewRunner(file)
	if err != nil {
		return
	}
	r = temp
	return
}

func NewRunner(file string) (r *Runner, err error) {
	r = new(Runner)
	r.p = fast.New()
	// importInfo := r.p.ImportPackage("", "github.com/SuperGod/trademodel")
	// r.p.Comp.CompGlobals.KnownImports["github.com/SuperGod/trademodel"] = importInfo
	// fmt.Println("import:", importInfo)
	_, err = r.p.EvalFile(file)
	if err != nil {
		return
	}
	err = r.extraScript()
	return
}

func (r *Runner) extraScript() (err error) {
	var info *CallInfo
	for k, v := range r.p.Comp.Types {
		info, err = NewCallInfo(r.p, k, v)
		if err != nil {
			err = nil
			continue
		}
		break
	}
	if info == nil {
		err = fmt.Errorf("extra script error")
		return
	}
	r.info = info
	return
}

func (r *Runner) Param() (paramInfo []common.Param, err error) {
	return r.info.Param()
}
func (r *Runner) Init(engine bengine.Engine, params common.ParamData) (err error) {
	return r.info.Init(engine, params)
}
func (r *Runner) OnCandle(candle Candle) (err error) {
	return r.info.OnCandle(candle)
}

func (r *Runner) OnPosition(pos, price float64) (err error) {
	return r.info.OnPosition(pos, price)
}

func (r *Runner) OnTrade(trade Trade) (err error) {
	return r.info.OnTrade(trade)
}
func (r *Runner) OnTradeHistory(trade Trade) (err error) {
	return r.info.OnTradeHistory(trade)
}
func (r *Runner) OnDepth(depth Depth) (err error) {
	return r.info.OnDepth(depth)
}

func (r *Runner) OnEvent(e Event) (err error) {
	return r.info.OnEvent(e)
}

func (r *Runner) GetName() string {
	return r.info.name
}
