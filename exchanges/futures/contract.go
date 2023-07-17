package futures

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

type Contract struct {
	Name       currency.Pair
	Underlying currency.Pair
	Asset      asset.Item
	StartDate  time.Time
	EndDate    time.Time
	IsActive   bool
	Type       ContractType
	// Optional values if the exchange offers them
	SettlementCurrencies currency.Currencies
	MarginCurrency       currency.Code
	Multiplier           float64
	MaxLeverage          float64
}

type ContractType uint8

const (
	Unset ContractType = iota
	Perpetual
	LongDated
	Weekly
	Fortnightly
	Monthly
	Quarterly
	SemiAnnually
	HalfYearly
	NineMonthly
	Yearly
	Unknown
)
