package exchange

import (
	"github.com/spf13/viper"
	. "github.com/ztrade/ztrade/pkg/core"
)

func GetTradeExchange(name string, cfg *viper.Viper, cltName, symbol string) (t *TradeExchange, err error) {
	ex, err := NewExchange(name, cfg, cltName)
	if err != nil {
		return
	}
	t = NewTradeExchange(name, ex, symbol)
	return
}
