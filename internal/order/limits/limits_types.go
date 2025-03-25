package limits

import (
	"errors"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/key"
)

// Public errors for order limits
var (
	ErrCannotLoadLimit         = errors.New("cannot load limit, levels not supplied")
	ErrOrderLimitNotFound      = errors.New("order limit not found")
	ErrExchangeLimitNotLoaded  = errors.New("exchange limits not loaded")
	ErrPriceBelowMin           = errors.New("price below minimum limit")
	ErrPriceExceedsMax         = errors.New("price exceeds maximum limit")
	ErrPriceExceedsStep        = errors.New("price exceeds step limit") // price is not divisible by its step
	ErrAmountBelowMin          = errors.New("amount below minimum limit")
	ErrAmountExceedsMax        = errors.New("amount exceeds maximum limit")
	ErrAmountExceedsStep       = errors.New("amount exceeds step limit") // amount is not divisible by its step
	ErrNotionalValue           = errors.New("total notional value is under minimum limit")
	ErrMarketAmountBelowMin    = errors.New("market order amount below minimum limit")
	ErrMarketAmountExceedsMax  = errors.New("market order amount exceeds maximum limit")
	ErrMarketAmountExceedsStep = errors.New("market order amount exceeds step limit") // amount is not divisible by its step for a market order
)

var (
	errExchangeNameEmpty   = errors.New("exchange name is empty")
	errAssetInvalid        = errors.New("asset is invalid")
	errPairNotSet          = errors.New("currency pair is not set")
	errInvalidPriceLevels  = errors.New("invalid price levels, cannot load limits")
	errInvalidAmountLevels = errors.New("invalid amount levels, cannot load limits")
	errInvalidQuoteLevels  = errors.New("invalid quote levels, cannot load limits")
)

// executionLimits defines minimum and maximum values in relation to
// order size, order pricing, total notional values, total maximum orders etc
// for execution on an exchange.
type executionLimits struct {
	epaLimits map[key.ExchangePairAsset]*MinMaxLevel
	mtx       sync.RWMutex
}

var executionLimitsManager = executionLimits{
	epaLimits: make(map[key.ExchangePairAsset]*MinMaxLevel),
}

// MinMaxLevel defines the minimum and maximum parameters for a currency pair
// for outbound exchange execution
type MinMaxLevel struct {
	UpdatedAt               time.Time
	Key                     key.ExchangePairAsset
	MinPrice                float64
	MaxPrice                float64
	PriceStepIncrementSize  float64
	MultiplierUp            float64
	MultiplierDown          float64
	MultiplierDecimal       float64
	AveragePriceMinutes     int64
	MinimumBaseAmount       float64
	MaximumBaseAmount       float64
	MinimumQuoteAmount      float64
	MaximumQuoteAmount      float64
	AmountStepIncrementSize float64
	QuoteStepIncrementSize  float64
	MinNotional             float64
	MaxIcebergParts         int64
	MarketMinQty            float64
	MarketMaxQty            float64
	MarketStepIncrementSize float64
	MaxTotalOrders          int64
	MaxAlgoOrders           int64
}
