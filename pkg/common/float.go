package common

import (
	"fmt"
	"strconv"

	"github.com/shopspring/decimal"
)

// FloatMul return a*b
func FloatMul(a, b float64) float64 {
	aDec := decimal.NewFromFloat(a)
	bDec := decimal.NewFromFloat(b)
	ret, _ := aDec.Mul(bDec).Float64()
	return ret
}

// FloatAdd return a*b
func FloatAdd(a, b float64) float64 {
	aDec := decimal.NewFromFloat(a)
	bDec := decimal.NewFromFloat(b)
	ret, _ := aDec.Add(bDec).Float64()
	return ret
}

// FloatSub return a-b
func FloatSub(a, b float64) float64 {
	aDec := decimal.NewFromFloat(a)
	bDec := decimal.NewFromFloat(b)
	ret, _ := aDec.Sub(bDec).Float64()
	return ret
}

// FloatDiv return a/b
func FloatDiv(a, b float64) float64 {
	aDec := decimal.NewFromFloat(a)
	bDec := decimal.NewFromFloat(b)
	ret, _ := aDec.Div(bDec).Float64()
	return ret
}

// FormatFloat format float with precision
func FormatFloat(n float64, precision int) float64 {
	str := fmt.Sprintf("%df", precision)
	n2, _ := strconv.ParseFloat(fmt.Sprintf("%."+str, n), 64)
	return n2
}
