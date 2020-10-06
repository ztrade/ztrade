package common

import "github.com/SuperGod/trademodel"

type CandleFn func(candle trademodel.Candle)

type Param struct {
	Name string
	Type string
	Info string
}

type ParamData map[string]interface{}

func (d ParamData) GetString(key, defaultValue string) string {
	v, ok := d[key]
	if !ok {
		return defaultValue
	}
	ret := v.(string)
	if ret == "" {
		return defaultValue
	}
	return ret
}
func (d ParamData) GetFloat(key string, defaultValue float64) float64 {
	v, ok := d[key]
	if !ok {
		return defaultValue
	}
	ret := v.(float64)
	if ret == 0 {
		return defaultValue
	}
	return ret
}
