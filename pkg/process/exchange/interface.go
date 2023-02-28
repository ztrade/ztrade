package exchange

import (
	"github.com/spf13/viper"
	"github.com/ztrade/exchange"
)

func GetTradeExchange(name string, cfg *viper.Viper, cltName, symbol string) (t *TradeExchange, err error) {
	ex, err := exchange.NewExchange(name, exchange.WrapViper(cfg), cltName)
	if err != nil {
		return
	}
	t = NewTradeExchange(name, ex, symbol)
	return
}
