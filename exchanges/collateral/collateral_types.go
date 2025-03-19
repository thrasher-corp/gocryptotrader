package collateral

import (
	"errors"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

// Mode defines the different collateral types supported by exchanges
// For example, FTX had a global collateral pool
// Binance has either singular position collateral calculation
// or cross aka asset level collateral calculation
type Mode uint8

const (
	// UnsetMode is the default value
	UnsetMode Mode = 0
	// SingleMode has allocated collateral per position
	SingleMode Mode = 1 << (iota - 1)
	// MultiMode has collateral allocated across the whole asset
	MultiMode
	// PortfolioMode has collateral allocated across account
	PortfolioMode
	// UnknownMode has collateral allocated in an unknown manner at present, but is not unset
	UnknownMode
	// SpotFuturesMode has collateral allocated across spot and futures accounts
	SpotFuturesMode
)

const (
	unsetCollateralStr       = "unset"
	singleCollateralStr      = "single"
	multiCollateralStr       = "multi"
	portfolioCollateralStr   = "portfolio"
	spotFuturesCollateralStr = "spot_futures"
	unknownCollateralStr     = "unknown"
)

// ErrInvalidCollateralMode is returned when converting invalid string to collateral mode
var ErrInvalidCollateralMode = errors.New("invalid collateral mode")

var supportedCollateralModes = SingleMode | MultiMode | PortfolioMode | SpotFuturesMode

// ByPosition shows how much collateral is used
// from positions
type ByPosition struct {
	PositionCurrency currency.Pair
	Size             decimal.Decimal
	OpenOrderSize    decimal.Decimal
	PositionSize     decimal.Decimal
	MarkPrice        decimal.Decimal
	RequiredMargin   decimal.Decimal
	CollateralUsed   decimal.Decimal
}

// ByCurrency individual collateral contribution
// along with what the potentially scaled collateral
// currency it is represented as
// eg in Bybit ScaledCurrency is USDC
type ByCurrency struct {
	Currency                    currency.Code
	SkipContribution            bool
	TotalFunds                  decimal.Decimal
	AvailableForUseAsCollateral decimal.Decimal
	CollateralContribution      decimal.Decimal
	AdditionalCollateralUsed    decimal.Decimal
	FairMarketValue             decimal.Decimal
	Weighting                   decimal.Decimal
	ScaledCurrency              currency.Code
	UnrealisedPNL               decimal.Decimal
	ScaledUsed                  decimal.Decimal
	ScaledUsedBreakdown         *UsedBreakdown
	Error                       error
}

// UsedBreakdown provides a detailed
// breakdown of where collateral is currently being allocated
type UsedBreakdown struct {
	LockedInStakes                  decimal.Decimal
	LockedInNFTBids                 decimal.Decimal
	LockedInFeeVoucher              decimal.Decimal
	LockedInSpotMarginFundingOffers decimal.Decimal
	LockedInSpotOrders              decimal.Decimal
	LockedAsCollateral              decimal.Decimal
	UsedInPositions                 decimal.Decimal
	UsedInSpotMarginBorrows         decimal.Decimal
}
