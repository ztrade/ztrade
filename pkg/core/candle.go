package core

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	. "github.com/ztrade/trademodel"
)

func FormatCandleName(name, binSize string) string {
	return fmt.Sprintf("%s:%s", name, binSize)
}

// ParseCandleName parse string to CandleName
func ParseCandleName(str string) (name, binSize string) {
	n := strings.Index(str, ":")
	if n == -1 {
		name = str
		return
	}
	return str[0:n], str[n+1:]
}

// Map2Candle convert candle to map
func Map2Candle(data interface{}) (candle *Candle) {
	candle, ok := data.(*Candle)
	if ok {
		return
	}
	candle = new(Candle)
	err := mapstructure.Decode(data, &candle)
	if err != nil {
		log.Error("Map2Candle failed:", data, err.Error())
	}
	return
}
