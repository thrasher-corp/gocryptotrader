package futures

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
)

// Contract holds details on futures contracts
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
	LatestRate           fundingrate.Rate
}

// ContractType holds the various style of contracts offered by futures exchanges
type ContractType uint8

// Contract type definitions
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

// String returns the string representation of the contract type
func (c ContractType) String() string {
	switch c {
	case Perpetual:
		return "perpetual"
	case LongDated:
		return "long_dated"
	case Weekly:
		return "weekly"
	case Fortnightly:
		return "fortnightly"
	case Monthly:
		return "monthly"
	case Quarterly:
		return "quarterly"
	case SemiAnnually:
		return "semi-annually"
	case HalfYearly:
		return "half-yearly"
	case NineMonthly:
		return "nine-monthly"
	case Yearly:
		return "yearly"
	case Unknown:
		return "unknown"
	default:
		return "unset"
	}
}
