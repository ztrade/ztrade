package common

import (
	"errors"

	. "github.com/SuperGod/trademodel"
	"github.com/shopspring/decimal"
)

var (
	ErrNoBalance = errors.New("no balance")
)

type VBalance struct {
	total    decimal.Decimal
	position decimal.Decimal
	feeTotal decimal.Decimal
	//  开仓的总价值
	longCost  decimal.Decimal
	shortCost decimal.Decimal
	fee       decimal.Decimal
}

func NewVBalance() *VBalance {
	b := new(VBalance)
	b.total = decimal.NewFromFloat(1000)
	b.fee = decimal.NewFromFloat(0.00075)
	return b
}

func (b *VBalance) Set(total float64) {
	b.total = decimal.NewFromFloat(total)
}

func (b *VBalance) SetFee(fee float64) {
	b.fee = decimal.NewFromFloat(fee)
}

func (b *VBalance) Pos() (pos float64) {
	pos, _ = b.position.Float64()
	return
}

func (b *VBalance) Get() (total float64) {
	// return b.total + b.costTotal
	total, _ = b.total.Float64()
	return
}

func (b *VBalance) GetFeeTotal() (fee float64) {
	fee, _ = b.feeTotal.Float64()
	return
}

func (b *VBalance) AddTrade(tr Trade) (profit float64, err error) {
	amount := decimal.NewFromFloat(tr.Amount).Abs()
	// 仓位价值
	cost := amount.Mul(decimal.NewFromFloat(tr.Price)).Abs()
	fee := cost.Mul(b.fee)
	costAll, _ := cost.Add(fee).Float64()
	if tr.Action.IsOpen() && costAll >= b.Get() {
		err = ErrNoBalance
		return
	}
	if tr.Action.IsLong() {
		b.position = b.position.Add(amount)
		b.longCost = b.longCost.Add(cost)
	} else {
		b.position = b.position.Sub(amount)
		b.shortCost = b.shortCost.Add(cost)
	}
	isPositionZero := b.position.Equal(decimal.NewFromInt(0))
	// !TODO: 开仓后要修改total
	if tr.Action.IsOpen() && !isPositionZero {
		b.total = b.total.Sub(cost)
	}
	// 计算盈利
	if isPositionZero {
		fee := b.shortCost.Add(b.longCost).Mul(b.fee)
		b.feeTotal = b.feeTotal.Add(fee)
		var prof decimal.Decimal
		prof = b.shortCost.Sub(b.longCost).Sub(fee)
		if tr.Action.IsLong() {
			b.total = b.total.Add(b.shortCost).Add(prof)
		} else {
			b.total = b.total.Add(b.longCost).Add(prof)
		}
		profit, _ = prof.Float64()
		b.longCost = decimal.NewFromInt(0)
		b.shortCost = decimal.NewFromInt(0)
	}
	return
}
