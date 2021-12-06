package order

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var errExchangeNameEmpty = errors.New("exchange name empty")
var errNotFutureAsset = errors.New("asset type is not futures")

type ExchangeFuturesTracker struct {
	Exchange              string
	Positions             []FuturesTracker
	PNL                   decimal.Decimal
	PNLCalculation        exchange.PNLManagement
	OfflinePNLCalculation bool
}

type FuturesTrackerSetup struct {
	Exchange              string
	Asset                 asset.Item
	Pair                  currency.Pair
	PNLCalculation        exchange.PNLManagement
	OfflinePNLCalculation bool
}

// FuturesTracker order is a concept which holds both the opening and closing orders
// for a futures contract. This allows for PNL calculations
type FuturesTracker struct {
	Exchange              string
	Asset                 asset.Item
	ContractPair          currency.Pair
	UnderlyingAsset       currency.Code
	Exposure              decimal.Decimal
	CurrentDirection      Side
	Status                Status
	AverageLeverage       decimal.Decimal
	UnrealisedPNL         decimal.Decimal
	RealisedPNL           decimal.Decimal
	ShortPositions        []Detail
	LongPositions         []Detail
	PNLHistory            []PNLHistory
	EntryPrice            decimal.Decimal
	ClosingPrice          decimal.Decimal
	OfflinePNLCalculation bool
	PNLCalculation        exchange.PNLManagement
}

// PNLHistory tracks how a futures contract
// pnl is going over the history of exposure
type PNLHistory struct {
	Time          time.Time
	UnrealisedPNL decimal.Decimal
	RealisedPNL   decimal.Decimal
}
