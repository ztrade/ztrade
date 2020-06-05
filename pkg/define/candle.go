package define

import (
	"fmt"
	"strings"

	. "github.com/SuperGod/trademodel"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
)

// CandleName candle info
type CandleName struct {
	Name    string
	BinSize string
}

// String return string
func (c *CandleName) String() string {
	return fmt.Sprintf("%s:%s", c.Name, c.BinSize)
}

// NewCandleName create CandleName with name and binSize
func NewCandleName(name, binSize string) *CandleName {
	return &CandleName{Name: name, BinSize: binSize}
}

// ParseCandleName parse string to CandleName
func ParseCandleName(name string) *CandleName {
	n := strings.Index(name, ":")
	if n == -1 {
		return &CandleName{Name: name}
	}
	return &CandleName{Name: name[0:n], BinSize: name[n+1:]}
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
