package kraken

import (
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// FuturesAccountsResponse holds every futures account
type FuturesAccountsResponse struct {
	Accounts   FuturesAccounts `json:"accounts"`
	ServerTime time.Time       `json:"serverTime"`
}

// FuturesAccounts holds the cash account, the multi-collateral margin account and the single-collateral margin
// accounts, which are keyed by name, such as fi_xbtusd
type FuturesAccounts struct {
	Cash           FuturesCashAccount
	Flex           FuturesFlexAccount
	MarginAccounts map[string]FuturesMarginAccount
}

// UnmarshalJSON decodes the cash and flex accounts and the margin accounts keyed beside them
func (a *FuturesAccounts) UnmarshalJSON(data []byte) error {
	var accounts map[string]json.RawMessage
	if err := json.Unmarshal(data, &accounts); err != nil {
		return err
	}
	a.MarginAccounts = make(map[string]FuturesMarginAccount, len(accounts))
	for name, account := range accounts {
		var err error
		switch name {
		case "cash":
			err = json.Unmarshal(account, &a.Cash)
		case "flex":
			err = json.Unmarshal(account, &a.Flex)
		default:
			var marginAccount FuturesMarginAccount
			if err = json.Unmarshal(account, &marginAccount); err == nil {
				a.MarginAccounts[name] = marginAccount
			}
		}
		if err != nil {
			return fmt.Errorf("error decoding %s account: %w", name, err)
		}
	}
	return nil
}

// FuturesCashAccount is the cash account
type FuturesCashAccount struct {
	// Type is cashAccount
	Type string `json:"type"`
	// Balances is keyed by lowercase currency code, such as xbt
	Balances map[string]types.Number `json:"balances"`
}

// FuturesFlexAccount is the multi-collateral margin account, whose values are in USD unless stated otherwise
type FuturesFlexAccount struct {
	// Type is multiCollateralMarginAccount
	Type string `json:"type"`
	// Currencies holds each collateral currency, keyed by currency code, such as XBT
	Currencies map[string]FuturesFlexCurrency `json:"currencies"`
	// InitialMargin is held for open positions, and InitialMarginWithOrders for open positions and orders
	InitialMargin           float64 `json:"initialMargin"`
	InitialMarginWithOrders float64 `json:"initialMarginWithOrders"`
	MaintenanceMargin       float64 `json:"maintenanceMargin"`
	BalanceValue            float64 `json:"balanceValue"`
	// PortfolioValue is BalanceValue plus unrealised PnL
	PortfolioValue float64 `json:"portfolioValue"`
	// CollateralValue is the value of the balances usable as margin, after haircuts
	CollateralValue   float64 `json:"collateralValue"`
	UnrealisedPNL     float64 `json:"pnl"`
	UnrealisedFunding float64 `json:"unrealizedFunding"`
	// TotalUnrealised is the unrealised PnL plus unrealised funding
	TotalUnrealised float64 `json:"totalUnrealized"`
	// TotalUnrealisedAsMargin is the part of TotalUnrealised usable as margin, after the haircut and conversion fee
	TotalUnrealisedAsMargin float64 `json:"totalUnrealizedAsMargin"`
	// AvailableMargin is MarginEquity less the initial margin
	AvailableMargin float64 `json:"availableMargin"`
	// MarginEquity is the balance value after haircuts plus TotalUnrealisedAsMargin
	MarginEquity             float64                          `json:"marginEquity"`
	PortfolioMarginBreakdown *FuturesPortfolioMarginBreakdown `json:"portfolioMarginBreakdown"`
	// UnrealisedPNLInterestRate is the interest rate applied to unrealised PnL
	UnrealisedPNLInterestRate float64 `json:"upnlInterestRate"`
}

// FuturesFlexCurrency is a collateral currency of the multi-collateral margin account
type FuturesFlexCurrency struct {
	Quantity float64 `json:"quantity"`
	// Value is the quantity's USD value
	Value float64 `json:"value"`
	// Collateral is the USD value usable as margin, after the haircut
	Collateral float64 `json:"collateral"`
	// Available is the margin available for trading, in the currency
	Available float64 `json:"available"`
}

// FuturesPortfolioMarginBreakdown breaks a portfolio margin calculation down into its components
type FuturesPortfolioMarginBreakdown struct {
	TotalCrossAssetNettedMarketRisk          float64   `json:"totalCrossAssetNettedMarketRisk"`
	TotalMarketRisk                          float64   `json:"totalMarketRisk"`
	TotalScenarioPNLs                        []float64 `json:"totalScenarioPnls"`
	TotalAbsoluteOptionPositionDeltaNotional float64   `json:"totalAbsoluteOptionPositionDeltaNotional"`
	NetPortfolioDelta                        float64   `json:"netPortfolioDelta"`
	TotalPremium                             float64   `json:"totalPremium"`
	IsBuyOnly                                bool      `json:"isBuyOnly"`
	FuturesMaintenanceMargin                 float64   `json:"futuresMaintenanceMargin"`
}

// FuturesMarginAccount is a single-collateral margin account, whose auxiliary values, margin requirements and trigger
// estimates are in its currency
type FuturesMarginAccount struct {
	// Type is marginAccount
	Type     string        `json:"type"`
	Currency currency.Code `json:"currency"`
	// Balances is keyed by lowercase currency code, such as xbt, or by contract symbol, such as FI_XBTUSD_171215
	Balances           map[string]types.Number   `json:"balances"`
	Auxiliary          FuturesMarginAuxiliary    `json:"auxiliary"`
	MarginRequirements FuturesMarginRequirements `json:"marginRequirements"`
	TriggerEstimates   FuturesMarginRequirements `json:"triggerEstimates"`
}

// FuturesMarginAuxiliary holds a margin account's auxiliary values
type FuturesMarginAuxiliary struct {
	USD            float64 `json:"usd"`
	PortfolioValue float64 `json:"pv"`
	UnrealisedPNL  float64 `json:"pnl"`
	AvailableFunds float64 `json:"af"`
	Funding        float64 `json:"funding"`
}

// FuturesMarginRequirements holds a margin account's margin requirements, or the estimates that trigger them
type FuturesMarginRequirements struct {
	InitialMargin        float64 `json:"im"`
	MaintenanceMargin    float64 `json:"mm"`
	LiquidationThreshold float64 `json:"lt"`
	TerminationThreshold float64 `json:"tt"`
}

// FuturesOpenPositionsResponse holds the open positions, latest fill first
type FuturesOpenPositionsResponse struct {
	OpenPositions []FuturesOpenPosition `json:"openPositions"`
	ServerTime    time.Time             `json:"serverTime"`
}

// FuturesOpenPosition is an open position
type FuturesOpenPosition struct {
	Symbol string `json:"symbol"`
	// Side is long or short
	Side              string  `json:"side"`
	Size              float64 `json:"size"`
	AverageEntryPrice float64 `json:"price"`
	UnrealisedPNL     float64 `json:"unrealizedPnl"`
	UnrealisedFunding float64 `json:"unrealizedFunding"`
	// PNLCurrency is the currency the position pays realised profit in, USD by default
	PNLCurrency currency.Code `json:"pnlCurrency"`
	// MaximumFixedLeverage is set for an isolated position
	MaximumFixedLeverage float64 `json:"maxFixedLeverage"`
	// Greeks is set for an option position
	Greeks *FuturesPositionGreeks `json:"greeks"`
}

// FuturesPositionGreeks holds an option position's greeks
type FuturesPositionGreeks struct {
	// ImpliedVolatility is -1 when it cannot be calculated or is out of bounds
	ImpliedVolatility float64 `json:"iv"`
	Delta             float64 `json:"delta"`
	Gamma             float64 `json:"gamma"`
	Vega              float64 `json:"vega"`
	Theta             float64 `json:"theta"`
	Rho               float64 `json:"rho"`
}

// FuturesUnwindQueueResponse holds where each open position ranks in the unwind queue
type FuturesUnwindQueueResponse struct {
	Queue      []FuturesUnwindQueuePosition `json:"queue"`
	ServerTime time.Time                    `json:"serverTime"`
}

// FuturesUnwindQueuePosition is where an open position ranks in the unwind queue
type FuturesUnwindQueuePosition struct {
	Symbol string `json:"symbol"`
	// Percentile is the position's percentile rank in the queue: 20, 40, 80 or 100
	Percentile uint64 `json:"percentile"`
}

// FuturesPortfolioMarginParametersResponse holds the parameters of the portfolio margin calculation and the
// account's options trading limits
type FuturesPortfolioMarginParametersResponse struct {
	CrossAssetNettingFactor             float64 `json:"crossAssetNettingFactor"`
	ExtremePriceShockMultiplier         float64 `json:"extremePriceShockMultiplier"`
	VolatilityShockMultiplicationFactor float64 `json:"volShockMultiplicationFactor"`
	VolatilityShockExponentFactor       float64 `json:"volShockExponentFactor"`
	OptionExpiryTimeShockHours          uint64  `json:"optionExpiryTimeShockHours"`
	OptionsInitialMarginFactor          float64 `json:"optionsInitialMarginFactor"`
	// TotalOptionOrdersInInitialMargin is how many option orders the initial margin calculation considers
	TotalOptionOrdersInInitialMargin uint64                   `json:"totalOptionOrdersConsideredInInitialMarginCalc"`
	PriceShockLevels                 []float64                `json:"priceShockLevels"`
	OptionsUserLimits                FuturesOptionsUserLimits `json:"optionsUserLimits"`
	ServerTime                       time.Time                `json:"serverTime"`
}

// FuturesOptionsUserLimits holds the account's options trading limits
type FuturesOptionsUserLimits struct {
	MaximumNetPositionDelta float64 `json:"maxNetPositionDelta"`
	// LimitsPerBaseCurrency is keyed by option contract base currency, such as BTC
	LimitsPerBaseCurrency map[string]FuturesOptionsBaseCurrencyLimits `json:"limitsPerBaseCurrency"`
}

// FuturesOptionsBaseCurrencyLimits holds the account's limits on the options of a base currency
type FuturesOptionsBaseCurrencyLimits struct {
	MaximumTotalPositionSize   float64 `json:"maxTotalPositionSize"`
	MaximumTotalOpenOrdersSize float64 `json:"maxTotalOpenOrdersSize"`
}

// FuturesSimulatedPosition is a position of a portfolio to simulate
type FuturesSimulatedPosition struct {
	Instrument currency.Pair
	Size       float64
	EntryPrice float64
}

// futuresPortfolioSimulationJSON is the portfolio Calculate portfolio margin, pnl and greeks takes in its json
// parameter
type futuresPortfolioSimulationJSON struct {
	Positions []futuresSimulatedPositionJSON `json:"positions"`
}

// futuresSimulatedPositionJSON is a position of a portfolio to simulate, as sent
type futuresSimulatedPositionJSON struct {
	Instrument string  `json:"instrument"`
	Size       float64 `json:"size"`
	EntryPrice float64 `json:"entryPrice"`
}

// FuturesSimulatePortfolioResponse holds the margin requirements, PnL and option greeks of a simulated portfolio
type FuturesSimulatePortfolioResponse struct {
	MaintenanceMargin        float64                          `json:"maintenanceMargin"`
	InitialMargin            float64                          `json:"initialMargin"`
	PNL                      float64                          `json:"pnl"`
	PortfolioMarginBreakdown FuturesPortfolioMarginBreakdown  `json:"portfolioMarginBreakdown"`
	Greeks                   map[string]FuturesPositionGreeks `json:"greeks"`
	ServerTime               time.Time                        `json:"serverTime"`
}
