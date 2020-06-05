package define

import "strings"

// SymbolInfo symbol infos
type SymbolInfo struct {
	ID          int64  `xorm:"pk autoincr null 'id'"`
	Exchange    string `xorm:"notnull unique(esr)  'exchange'"`
	Symbol      string `xorm:"notnull unique(esr) 'symbol'"`
	Resolutions string `xorm:"notnull unique(esr) 'resolutions'"`
	Pricescale  int    `xorm:"notnull 'pricescale'"`
}

func (si *SymbolInfo) GetResolutions() []string {
	return strings.Split(si.Resolutions, ",")
}
