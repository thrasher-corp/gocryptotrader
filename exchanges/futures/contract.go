package futures

import (
	"errors"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
)

// var error definitions
var (
	ErrInvalidContractSettlementType = errors.New("invalid contract settlement type")
	ErrContractNotSupported          = errors.New("unsupported contract")
)

// Contract holds details on futures contracts
type Contract struct {
	Exchange       string
	Name           currency.Pair
	Underlying     currency.Pair
	Asset          asset.Item
	StartDate      time.Time
	EndDate        time.Time
	IsActive       bool
	Status         string
	Type           ContractType
	SettlementType ContractSettlementType
	// Optional values if the exchange offers them
	SettlementCurrency             currency.Code
	AdditionalSettlementCurrencies currency.Currencies
	MarginCurrency                 currency.Code
	Multiplier                     float64
	MaxLeverage                    float64
	LatestRate                     fundingrate.Rate
	FundingRateFloor               decimal.Decimal
	FundingRateCeiling             decimal.Decimal
}

// ContractSettlementType holds the various style of contracts offered by futures exchanges
type ContractSettlementType uint8

// ContractSettlementType definitions
const (
	UnsetSettlementType ContractSettlementType = iota
	Linear
	Inverse
	Quanto
	LinearOrInverse
	Hybrid
)

// String returns the string representation of a contract settlement type
func (d ContractSettlementType) String() string {
	switch d {
	case UnsetSettlementType:
		return "unset"
	case Linear:
		return "linear"
	case Inverse:
		return "inverse"
	case Quanto:
		return "quanto"
	case LinearOrInverse:
		return "linearOrInverse"
	case Hybrid:
		return "hybrid"
	default:
		return "unknown"
	}
}

// StringToContractSettlementType for converting case insensitive contract settlement type
func StringToContractSettlementType(cstype string) (ContractSettlementType, error) {
	cstype = strings.ToLower(cstype)
	switch cstype {
	case UnsetSettlementType.String(), "":
		return UnsetSettlementType, nil
	case Linear.String():
		return Linear, nil
	case Inverse.String():
		return Inverse, nil
	case Quanto.String():
		return Quanto, nil
	case "linearorinverse":
		return LinearOrInverse, nil
	case Hybrid.String():
		return Hybrid, nil
	default:
		return UnsetSettlementType, ErrInvalidContractSettlementType
	}
}

// ContractType holds the various style of contracts offered by futures exchanges
type ContractType uint8

// ContractType definitions
const (
	UnsetContractType ContractType = iota
	Perpetual
	LongDated
	Weekly
	Fortnightly
	ThreeWeekly
	Monthly
	Quarterly
	SemiAnnually
	HalfYearly
	NineMonthly
	Yearly
	Unknown
	Daily
)

// String returns the string representation of the contract type
func (c ContractType) String() string {
	switch c {
	case Daily:
		return "day"
	case Perpetual:
		return "perpetual"
	case LongDated:
		return "long_dated"
	case Weekly:
		return "weekly"
	case Fortnightly:
		return "fortnightly"
	case ThreeWeekly:
		return "three-weekly"
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
