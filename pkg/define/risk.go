package define

// RiskLimits risk limits
type RiskLimits map[string]RiskLimit

func NewRiskLimits() (rl RiskLimits) {
	rl = make(RiskLimits)
	return rl
}

func (rl RiskLimits) Update(limit RiskLimit) {
	rl[limit.Key()] = limit
}

func (rl RiskLimits) GetLimitRatio(limit RiskLimit) (ret float64) {
	l, ok := rl[limit.Key()]
	if ok {
		ret = l.MaxLostRatio
		return
	}
	limit.Lever = 0
	l, ok = rl[limit.Key()]
	if ok {
		ret = l.MaxLostRatio
		return
	}
	limit.Code = ""
	l, ok = rl[limit.Key()]
	if ok {
		ret = l.MaxLostRatio
		return
	}
	return
}
