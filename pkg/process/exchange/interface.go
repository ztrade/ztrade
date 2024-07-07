package exchange

import (
	"fmt"

	"github.com/ztrade/exchange"
)

func GetTradeExchange(name string, cfg exchange.Config, cltName, symbol string) (t *TradeExchange, err error) {
	ex, err := exchange.NewExchange(name, cfg, cltName)
	if err != nil {
		return
	}
	t = NewTradeExchange(name, ex, symbol)
	localStop := cfg.GetBool(fmt.Sprintf("exchanges.%s.localstop", cltName))
	t.UseLocalStopOrder(localStop)
	return
}
