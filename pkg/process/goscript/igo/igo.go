package igo

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"

	"github.com/goplus/igop"
	_ "github.com/goplus/igop/pkg/encoding/json"
	_ "github.com/goplus/igop/pkg/fmt"
	_ "github.com/goplus/igop/pkg/math"
	"github.com/ztrade/base/common"
	"github.com/ztrade/ztrade/pkg/process/goscript/engine"
	"golang.org/x/tools/go/ssa"
)

func init() {
	igop.RegisterCustomBuiltin("min", min)
	igop.RegisterCustomBuiltin("max", max)
	igop.RegisterCustomBuiltin("formatFloat", common.FormatFloat)
	igop.RegisterCustomBuiltin("FloatAdd", common.FloatAdd)
	igop.RegisterCustomBuiltin("FloatSub", common.FloatSub)
	igop.RegisterCustomBuiltin("FloatMul", common.FloatMul)
	igop.RegisterCustomBuiltin("FloatDiv", common.FloatDiv)
	igop.RegisterCustomBuiltin("StringParam", common.StringParam)
	igop.RegisterCustomBuiltin("FloatParam", common.FloatParam)
	igop.RegisterCustomBuiltin("IntParam", common.IntParam)
	igop.RegisterCustomBuiltin("BoolParam", common.BoolParam)
}
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

var (
	fixImport = `
import (
    "github.com/ztrade/base/engine"
    "github.com/ztrade/base/common"
)
`

	fixType = `
type Engine = engine.Engine
type Param = common.Param
type ParamData = common.ParamData
`
)

func fixSource(file string) (ret string, err error) {
	f, err := os.Open(file)
	if err != nil {
		return
	}
	defer f.Close()
	r := bufio.NewReader(f)
	var temp bytes.Buffer
	var l []byte
	var bFixType bool
	for {
		l, err = r.ReadBytes('\n')
		if err != nil {
			break
		}

		if bytes.HasPrefix(l, []byte("package")) {
			temp.Write(l)
			temp.WriteString(fixImport)
		} else if !bFixType && bytes.HasPrefix(l, []byte("type")) {
			temp.WriteString(fixType)
			temp.Write(l)
			bFixType = true
		} else {
			temp.Write(l)
		}
	}
	if err == io.EOF {
		err = nil
	}
	ret = temp.String()
	return
}

func NewRunner(file string) (r engine.Runner, err error) {
	source, err := fixSource(file)
	if err != nil {
		return
	}
	ctx := igop.NewContext(0)
	pkg, err := ctx.LoadFile(filepath.Base(file), source)
	if err != nil {
		err = fmt.Errorf("igop parse file failed: %s", err.Error())
		return
	}
	// pkg.Members["Engine"] = s
	var typs []string
	for k, v := range pkg.Members {
		_, ok := v.(*ssa.Type)
		if !ok {
			continue
		}
		typs = append(typs, k)
	}

	interp, err := ctx.NewInterp(pkg)
	if err != nil {
		err = fmt.Errorf("igop NewInterp failed: %s", err.Error())
		return
	}

	var ok bool
	var temp igoImpl
	var fn interface{}
	for _, v := range typs {
		fn, ok = interp.GetFunc(fmt.Sprintf("New%s", v))
		fmt.Println(v, fn, ok)
		if !ok {
			continue
		}

		rets := reflect.ValueOf(fn).Call([]reflect.Value{})
		if len(rets) != 1 {
			continue
		}
		temp, ok = rets[0].Interface().(igoImpl)
		if ok {
			break
		}
	}
	if temp == nil {
		return
	}
	r = &igoRunner{impl: temp}
	return
}
