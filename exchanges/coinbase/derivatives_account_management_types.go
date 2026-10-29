package coinbase

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	errDerivativesInvalidAccessLog            = errors.New("invalid gateway access log")
	errDerivativesMalformedLeverage           = errors.New("malformed gateway leverage")
	errDerivativesInvalidMatchingEngineLimits = errors.New("invalid gateway matching engine limits")
)

// DerivativesChangeMarginModelRequest changes the margin model of the account or one of its subaccounts
type DerivativesChangeMarginModelRequest struct {
	// UserID selects a subaccount; zero selects the authenticated account
	UserID uint64
	// MarginModel is cross_pm, cross_sm, segregated_pm or segregated_sm
	MarginModel string
	// DryRun returns the change's effect without making it
	DryRun bool
}

// DerivativesChangeMarginModelWireRequest holds private/change_margin_model's params
type DerivativesChangeMarginModelWireRequest struct {
	UserID      uint64 `json:"user_id,omitempty"`
	MarginModel string `json:"margin_model"`
	DryRun      bool   `json:"dry_run,omitempty"`
}

// DerivativesMarginModelChangeResponse is a margin model change's effect on one currency; the gateway sends the
// currency in lower case
type DerivativesMarginModelChangeResponse struct {
	Currency currency.Code               `json:"currency"`
	OldState DerivativesMarginModelState `json:"old_state"`
	NewState DerivativesMarginModelState `json:"new_state"`
}

// DerivativesMarginModelState is a currency's available balance and margin rates under a margin model
type DerivativesMarginModelState struct {
	AvailableBalance      float64 `json:"available_balance"`
	InitialMarginRate     float64 `json:"initial_margin_rate"`
	MaintenanceMarginRate float64 `json:"maintenance_margin_rate"`
}

// DerivativesGetAccessLogRequest selects a page of the account's access log, newest first
type DerivativesGetAccessLogRequest struct {
	Offset uint64
	// Count defaults to 10, at most 1000
	Count uint64
}

// DerivativesAccessLogWireRequest holds private/get_access_log's params
type DerivativesAccessLogWireRequest struct {
	Offset uint64 `json:"offset,omitempty"`
	Count  uint64 `json:"count,omitempty"`
}

// DerivativesAccessLogResponse is a page of the account's access log
type DerivativesAccessLogResponse struct {
	// RecordsTotal counts every entry of the log; the documented example sends it with the page, while the schema
	// documents a bare list of entries, which leaves it zero
	RecordsTotal uint64                      `json:"records_total"`
	Entries      []DerivativesAccessLogEntry `json:"data"`
}

// UnmarshalJSON decodes the page the documented example returns, or the bare list of entries the schema documents
func (d *DerivativesAccessLogResponse) UnmarshalJSON(data []byte) error {
	if trimmed := bytes.TrimSpace(data); len(trimmed) > 0 && trimmed[0] == '[' {
		var entries []DerivativesAccessLogEntry
		if err := json.Unmarshal(trimmed, &entries); err != nil {
			return fmt.Errorf("%w: %w", errDerivativesInvalidAccessLog, err)
		}
		*d = DerivativesAccessLogResponse{Entries: entries}
		return nil
	}
	type accessLogPage DerivativesAccessLogResponse
	if err := json.Unmarshal(data, (*accessLogPage)(d)); err != nil {
		return fmt.Errorf("%w: %w", errDerivativesInvalidAccessLog, err)
	}
	return nil
}

// DerivativesAccessLogEntry is an access log entry: a login, or a change to the account's security or API keys
type DerivativesAccessLogEntry struct {
	ID        uint64     `json:"id"`
	Timestamp types.Time `json:"timestamp"`
	IP        string     `json:"ip"`
	// Country and City are estimated from the IP address
	Country string `json:"country"`
	City    string `json:"city"`
	// Log is the action: success or failure for a login, changed_email, changed_password, enabled_tfa, disabled_tfa,
	// enabled_subaccount_login, disabled_subaccount_login, new_api_key, removed_api_key, changed_scope,
	// changed_whitelist, disabled_api_key, enabled_api_key or reset_api_key. Result holds it where the documented
	// example names it result
	Log    string `json:"log"`
	Result string `json:"result"`
	// Data holds the action's details as sent, an object or a string such as a subaccount's user ID or an API key's
	// client ID, depending on the action
	Data json.RawMessage `json:"data"`
}

// DerivativesGetAccountSummariesRequest requests the summaries of the account or a subaccount in every currency it
// holds
type DerivativesGetAccountSummariesRequest struct {
	SubaccountID uint64
	// Extended includes the account's details and fees
	Extended bool
	// IncludeIsolated includes the summaries of the active isolated margin subaccounts, which are left out otherwise
	IncludeIsolated bool
}

// DerivativesAccountSummariesWireRequest holds private/get_account_summaries' params
type DerivativesAccountSummariesWireRequest struct {
	SubaccountID    uint64 `json:"subaccount_id,omitempty"`
	Extended        bool   `json:"extended,omitempty"`
	IncludeIsolated bool   `json:"include_isolated,omitempty"`
}

// DerivativesAccountSummariesResponse is the account's summary in each currency it holds, the active isolated margin
// subaccounts' summaries when asked for and, when extended, the account's details
type DerivativesAccountSummariesResponse struct {
	DerivativesAccountDetails
	Summaries []DerivativesAccountSummary `json:"summaries"`
	// IsolatedAccountSummaries lists each active isolated margin subaccount's summaries; a recycled subaccount, or one
	// whose summaries the gateway could not aggregate, is left out
	IsolatedAccountSummaries []DerivativesIsolatedAccountSummaries `json:"isolated_account_summaries"`
	// AffiliatePromotionFee is sent when it is greater than zero
	AffiliatePromotionFee float64 `json:"affiliate_promotion_fee"`
	ReceiveNotifications  bool    `json:"receive_notifications"`
	// TradingProductsDetails holds which trading products are enabled or can be overwritten, as sent, as its form is
	// undocumented
	TradingProductsDetails json.RawMessage `json:"trading_products_details"`
}

// DerivativesIsolatedAccountSummaries is an isolated margin subaccount's summary in each currency; its risk fields
// are per currency, so the settlement currency's summary is the one to read
type DerivativesIsolatedAccountSummaries struct {
	// ID is the subaccount's user ID, which subaccount ID parameters take
	ID        uint64                      `json:"id"`
	Summaries []DerivativesAccountSummary `json:"summaries"`
}

// DerivativesAccountDetails is an account's identity and settings, which extended summaries include
type DerivativesAccountDetails struct {
	ID uint64 `json:"id"`
	// Type is main or subaccount
	Type       string `json:"type"`
	Username   string `json:"username"`
	SystemName string `json:"system_name"`
	Email      string `json:"email"`
	// LoginEnabled is set when a subaccount can log in with its email and password
	LoginEnabled              bool `json:"login_enabled"`
	SecurityKeysEnabled       bool `json:"security_keys_enabled"`
	MMPEnabled                bool `json:"mmp_enabled"`
	InterUserTransfersEnabled bool `json:"interuser_transfers_enabled"`
	// ReferrerID identifies the affiliate link the account registered with, as the suffix of its path after /reg-
	ReferrerID        string     `json:"referrer_id"`
	CreationTimestamp types.Time `json:"creation_timestamp"`
	// SelfTradingRejectMode is reject_taker or cancel_maker
	SelfTradingRejectMode string `json:"self_trading_reject_mode"`
	// SelfTradingExtendedToSubaccounts is documented as a string and sent as a boolean in the example, so either is
	// accepted
	SelfTradingExtendedToSubaccounts types.Boolean `json:"self_trading_extended_to_subaccounts"`
	// BlockRFQSelfMatchPrevention stops Block RFQ trades between accounts of the same legal entity, apart from self
	// trading prevention
	BlockRFQSelfMatchPrevention bool `json:"block_rfq_self_match_prevention"`
}

// DerivativesGetAccountSummaryRequest requests the summary of the account or a subaccount in one currency
type DerivativesGetAccountSummaryRequest struct {
	// Currency is BTC, ETH, STETH, ETHW, USDC, USDT, EURR, SOL, XRP, USYC, PAXG, BNB or USDE
	Currency     currency.Code
	SubaccountID uint64
	// Extended includes the account's details and fees
	Extended bool
}

// DerivativesAccountSummaryWireRequest holds private/get_account_summary's params
type DerivativesAccountSummaryWireRequest struct {
	// Currency stays a string, the code in the case the gateway takes
	Currency     string `json:"currency"`
	SubaccountID uint64 `json:"subaccount_id,omitempty"`
	Extended     bool   `json:"extended,omitempty"`
}

// DerivativesAccountSummaryResponse is the account's summary in one currency and, when extended, its details
type DerivativesAccountSummaryResponse struct {
	DerivativesAccountSummary
	DerivativesAccountDetails
	IsDirectAccessAllowed bool `json:"is_direct_access_allowed"`
}

// DerivativesPortfolioSummary is a portfolio's balances, margins, profit and loss and greeks in one currency, as
// account summaries and portfolio simulations report them. With cross collateral enabled, the margins and funds
// aggregate every cross collateral currency, converted at each one's index
type DerivativesPortfolioSummary struct {
	Currency currency.Code `json:"currency"`
	// Balance is the cash balance, which leaves out open futures profit and loss and the options' mark value
	Balance float64 `json:"balance"`
	// Equity is the balance plus the futures' session profit and loss and the options' mark value
	Equity float64 `json:"equity"`
	// MarginBalance is the collateral margin is held against: equity less the options' value under standard margin,
	// equity under segregated portfolio margin and equity less any loan under cross portfolio margin
	MarginBalance float64 `json:"margin_balance"`
	// AvailableFunds is the margin balance less the initial margin, floored at zero, where only reducing orders can be
	// placed
	AvailableFunds float64 `json:"available_funds"`
	// AvailableWithdrawalFunds is never negative, and usually less than AvailableFunds, as withdrawals also leave out
	// session profit and locked and reserved balances
	AvailableWithdrawalFunds float64 `json:"available_withdrawal_funds"`
	// SpotReserve is reserved by open spot orders, and AdditionalReserve, on the cross_sm margin model only, for the
	// premium of open buy option and option combo orders
	SpotReserve       float64 `json:"spot_reserve"`
	AdditionalReserve float64 `json:"additional_reserve"`
	// LockedBalance is left out of the funds available to withdraw
	LockedBalance float64 `json:"locked_balance"`
	// FeeBalance can pay fees
	FeeBalance        float64 `json:"fee_balance"`
	InitialMargin     float64 `json:"initial_margin"`
	MaintenanceMargin float64 `json:"maintenance_margin"`
	// ProjectedInitialMargin and ProjectedMaintenanceMargin leave out the instruments expiring at the nearest expiry
	ProjectedInitialMargin     float64 `json:"projected_initial_margin"`
	ProjectedMaintenanceMargin float64 `json:"projected_maintenance_margin"`
	// TotalProfitLoss covers the open positions since they opened, while the session fields reset at each daily
	// settlement
	TotalProfitLoss                    float64 `json:"total_pl"`
	SessionRealisedProfitLoss          float64 `json:"session_rpl"`
	SessionUnrealisedProfitLoss        float64 `json:"session_upl"`
	FuturesProfitLoss                  float64 `json:"futures_pl"`
	FuturesSessionRealisedProfitLoss   float64 `json:"futures_session_rpl"`
	FuturesSessionUnrealisedProfitLoss float64 `json:"futures_session_upl"`
	OptionsProfitLoss                  float64 `json:"options_pl"`
	OptionsSessionRealisedProfitLoss   float64 `json:"options_session_rpl"`
	OptionsSessionUnrealisedProfitLoss float64 `json:"options_session_upl"`
	// OptionsValue is the open options' mark value
	OptionsValue float64 `json:"options_value"`
	OptionsDelta float64 `json:"options_delta"`
	OptionsGamma float64 `json:"options_gamma"`
	// OptionsTheta is a day's theta, scaled down for options expiring within a day
	OptionsTheta float64 `json:"options_theta"`
	OptionsVega  float64 `json:"options_vega"`
	// OptionsGammaByIndex, OptionsThetaByIndex, OptionsVegaByIndex and DeltaTotalByIndex are keyed by index name, such
	// as btc_usd
	OptionsGammaByIndex map[string]float64 `json:"options_gamma_map"`
	OptionsThetaByIndex map[string]float64 `json:"options_theta_map"`
	OptionsVegaByIndex  map[string]float64 `json:"options_vega_map"`
	// DeltaTotal sums the options' net transaction deltas and the futures' positions, and decays during settlement
	DeltaTotal float64 `json:"delta_total"`
	// ProjectedDeltaTotal leaves out the positions expiring at the nearest expiry
	ProjectedDeltaTotal float64 `json:"projected_delta_total"`
	// DeltaTotalByIndex holds raw position deltas, which do not add up to DeltaTotal
	DeltaTotalByIndex map[string]float64 `json:"delta_total_map"`
	// MarginModel is cross_pm, cross_sm, segregated_pm or segregated_sm
	MarginModel               string `json:"margin_model"`
	PortfolioMarginingEnabled bool   `json:"portfolio_margining_enabled"`
	CrossCollateralEnabled    bool   `json:"cross_collateral_enabled"`
	// The USD totals cover every cross collateral currency, and are only sent to cross margin accounts
	TotalEquityUSD            float64 `json:"total_equity_usd"`
	TotalInitialMarginUSD     float64 `json:"total_initial_margin_usd"`
	TotalMaintenanceMarginUSD float64 `json:"total_maintenance_margin_usd"`
	TotalMarginBalanceUSD     float64 `json:"total_margin_balance_usd"`
	TotalDeltaTotalUSD        float64 `json:"total_delta_total_usd"`
}

// DerivativesAccountSummary is an account's balances, margins, profit and loss, greeks, fees and rate limits in one
// currency
type DerivativesAccountSummary struct {
	DerivativesPortfolioSummary
	// CloseOutMargin is half the positions' maintenance margin, below which close-out liquidation takes over;
	// ProjectedCloseOutMargin leaves out the instruments expiring at the nearest expiry. The gateway only sends them
	// where close-out margin is enabled
	CloseOutMargin          float64 `json:"close_out_margin"`
	ProjectedCloseOutMargin float64 `json:"projected_close_out_margin"`
	DepositAddress          string  `json:"deposit_address"`
	// FeeGroup is the account's volume fee discount level, sent with extended summaries when the account has one
	FeeGroup string `json:"fee_group"`
	// Fees holds the account's discounted fees, sent with extended summaries; it is keyed by index name, such as
	// btc_usd, and then by instrument type: future, perpetual or option
	Fees   map[string]map[string]DerivativesAccountFee `json:"fees"`
	Limits DerivativesAccountLimits                    `json:"limits"`
	// HasNonBlockchainEquity is set when the account holds equity left out of proof of reserves
	HasNonBlockchainEquity bool `json:"has_non_block_chain_equity"`
	// AffiliatePromotionFee is sent when it is greater than zero
	AffiliatePromotionFee float64 `json:"affiliate_promotion_fee"`
	ReceiveNotifications  bool    `json:"receive_notifications"`
	// TradingProductsDetails holds which trading products are enabled or can be overwritten, as sent, as its form is
	// undocumented
	TradingProductsDetails json.RawMessage `json:"trading_products_details"`
}

// DerivativesAccountFee is the account's fee rates for an instrument type
type DerivativesAccountFee struct {
	Default    DerivativesAccountFeeRates `json:"default"`
	BlockTrade float64                    `json:"block_trade"`
}

// DerivativesAccountFeeRates are maker and taker fee rates
type DerivativesAccountFeeRates struct {
	// Type is fixed for a fee rate, or relative for a fraction of the instrument's fee
	Type  string  `json:"type"`
	Maker float64 `json:"maker"`
	Taker float64 `json:"taker"`
}

// DerivativesAccountLimits are the account's rate limits, which a support article rather than the schema documents
type DerivativesAccountLimits struct {
	// LimitsPerCurrency is set when the matching engine's trading and quote limits are set per settlement currency
	LimitsPerCurrency bool                            `json:"limits_per_currency"`
	NonMatchingEngine DerivativesAccountRateLimit     `json:"non_matching_engine"`
	MatchingEngine    DerivativesMatchingEngineLimits `json:"matching_engine"`
}

// DerivativesAccountRateLimit is a sustained rate of requests a second and the burst of requests allowed
type DerivativesAccountRateLimit struct {
	Rate  uint64 `json:"rate"`
	Burst uint64 `json:"burst"`
}

// DerivativesMatchingEngineLimits are the matching engine's rate limits
type DerivativesMatchingEngineLimits struct {
	// Spot applies to spot trading, and CancelAll to cancelling every order, or every order with a label, without a
	// currency
	Spot      DerivativesAccountRateLimit `json:"spot"`
	CancelAll DerivativesAccountRateLimit `json:"cancel_all"`
	// DerivativesMatchingEngineTradingLimits holds the trading and quote limits when they apply to every currency
	DerivativesMatchingEngineTradingLimits
	// Currencies holds the trading and quote limits keyed by lower case settlement currency, when they are set per
	// settlement currency, which the gateway sends as keys of the matching engine object
	Currencies map[string]DerivativesMatchingEngineTradingLimits `json:"-"`
}

// UnmarshalJSON decodes the matching engine limits that apply to every currency and those set per settlement currency;
// quotes, max_quotes and guaranteed_quotes, the names the documented example gives them, decode as maximum_quotes,
// maximum_mass_quotes and guaranteed_mass_quotes, the names Deribit sends
func (d *DerivativesMatchingEngineLimits) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return fmt.Errorf("%w: %w", errDerivativesInvalidMatchingEngineLimits, err)
	}
	var limits DerivativesMatchingEngineLimits
	for key, value := range fields {
		var target any
		switch key {
		case "spot":
			target = &limits.Spot
		case "cancel_all":
			target = &limits.CancelAll
		case "trading":
			target = &limits.Trading
		case "maximum_quotes", "quotes":
			target = &limits.MaximumQuotes
		case "maximum_mass_quotes", "max_quotes":
			target = &limits.MaximumMassQuotes
		case "guaranteed_mass_quotes", "guaranteed_quotes":
			target = &limits.GuaranteedMassQuotes
		default:
			var currencyLimits DerivativesMatchingEngineTradingLimits
			if err := json.Unmarshal(value, &currencyLimits); err != nil {
				return fmt.Errorf("%w: %s: %w", errDerivativesInvalidMatchingEngineLimits, key, err)
			}
			if limits.Currencies == nil {
				limits.Currencies = make(map[string]DerivativesMatchingEngineTradingLimits)
			}
			limits.Currencies[key] = currencyLimits
			continue
		}
		if err := json.Unmarshal(value, target); err != nil {
			return fmt.Errorf("%w: %s: %w", errDerivativesInvalidMatchingEngineLimits, key, err)
		}
	}
	*d = limits
	return nil
}

// DerivativesMatchingEngineTradingLimits are the matching engine's trading and quote rate limits
type DerivativesMatchingEngineTradingLimits struct {
	// Trading is keyed by product: total covers every product, and others, such as perpetuals, one product
	Trading              map[string]DerivativesAccountRateLimit `json:"trading"`
	MaximumQuotes        DerivativesAccountRateLimit            `json:"maximum_quotes"`
	MaximumMassQuotes    DerivativesAccountRateLimit            `json:"maximum_mass_quotes"`
	GuaranteedMassQuotes DerivativesAccountRateLimit            `json:"guaranteed_mass_quotes"`
}

// DerivativesGetLeverageRequest requests the custom leverage of one future, or of every future that has one
type DerivativesGetLeverageRequest struct {
	// InstrumentName selects one active future; empty selects every future with a custom leverage
	InstrumentName string
	// SubaccountID selects a subaccount for retail broker callers; zero selects the authenticated account
	SubaccountID uint64
	// Isolated reads the isolated margin leverage, which only the main account can; the gateway reads the cross
	// margin leverage without it, rather than refusing
	Isolated bool
}

// DerivativesLeverageWireRequest holds private/get_leverage's params
type DerivativesLeverageWireRequest struct {
	InstrumentName string `json:"instrument_name,omitempty"`
	SubaccountID   uint64 `json:"subaccount_id,omitempty"`
	Isolated       bool   `json:"isolated,omitempty"`
}

// DerivativesLeverageResponse is a future's custom and maximum leverage
type DerivativesLeverageResponse struct {
	InstrumentName string `json:"instrument_name"`
	// Leverage is the custom leverage, zero when the future uses MaximumLeverage, which the gateway sends as null
	Leverage float64 `json:"leverage"`
	// MaximumLeverage is the effective maximum leverage
	MaximumLeverage float64 `json:"max_leverage"`
}

// DerivativesLeverages decodes private/get_leverage's result, which is one future's leverage, a list of them or an
// empty object
type DerivativesLeverages []DerivativesLeverageResponse

// UnmarshalJSON decodes the leverage the gateway returns for one future, the list it returns for every future with a
// custom leverage, or the empty object it returns a regular subaccount, from which isolated leverage is hidden
func (d *DerivativesLeverages) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("null")) {
		return nil
	}
	if len(trimmed) > 0 && trimmed[0] == '[' {
		var leverages []DerivativesLeverageResponse
		if err := json.Unmarshal(trimmed, &leverages); err != nil {
			return fmt.Errorf("%w: %w", errDerivativesMalformedLeverage, err)
		}
		*d = leverages
		return nil
	}
	var leverage DerivativesLeverageResponse
	if err := json.Unmarshal(trimmed, &leverage); err != nil {
		return fmt.Errorf("%w: %w", errDerivativesMalformedLeverage, err)
	}
	if leverage == (DerivativesLeverageResponse{}) {
		*d = DerivativesLeverages{}
		return nil
	}
	*d = DerivativesLeverages{leverage}
	return nil
}

// DerivativesPMEParamsResponse is the account's portfolio margin open order restrictions and risk limits in a
// currency; only Enabled is sent when portfolio margin is disabled
type DerivativesPMEParamsResponse struct {
	Enabled                     bool    `json:"enabled"`
	MaximumFutureOpenOrders     uint64  `json:"max_future_open_orders"`
	MaximumFutureOpenOrdersSize float64 `json:"max_future_open_orders_size"`
	MaximumOptionOpenOrders     uint64  `json:"max_option_open_orders"`
	// MaximumOptionOpenOrdersSize is in contracts
	MaximumOptionOpenOrdersSize float64 `json:"max_option_open_orders_size"`
	MaximumRiskReducingOrders   uint64  `json:"max_risk_reducing_orders"`
	MaximumTotalOpenOrders      uint64  `json:"max_total_open_orders"`
}

// DerivativesGetPositionRequest requests the account's position in an instrument
type DerivativesGetPositionRequest struct {
	InstrumentName string
	// IncludeIsolated reads the position of the isolated margin subaccount bound to the instrument, in place of the
	// cross margin position, when one is bound
	IncludeIsolated bool
}

// DerivativesPositionWireRequest holds private/get_position's params
type DerivativesPositionWireRequest struct {
	InstrumentName  string `json:"instrument_name"`
	IncludeIsolated bool   `json:"include_isolated,omitempty"`
}

// DerivativesPositionResponse is a position in an instrument; a closed position has a zero size and the direction zero.
// An isolated margin position is owned by its subaccount, whose user ID UserID holds
type DerivativesPositionResponse struct {
	InstrumentName string `json:"instrument_name"`
	// Kind is future, option, spot, future_combo or option_combo
	Kind   string `json:"kind"`
	UserID uint64 `json:"user_id"`
	// Isolated and MainUserID, the owning main account's user ID, are only sent for an isolated margin position
	Isolated   bool   `json:"isolated"`
	MainUserID uint64 `json:"main_uid"`
	// Direction is buy, sell or zero
	Direction string `json:"direction"`
	// Size is in the quote currency, such as USD, for futures and in the base currency for options, and
	// SizeCurrency, for futures only, in the base currency
	Size         float64 `json:"size"`
	SizeCurrency float64 `json:"size_currency"`
	AveragePrice float64 `json:"average_price"`
	// AveragePriceUSD and FloatingProfitLossUSD are only sent for options
	AveragePriceUSD float64 `json:"average_price_usd"`
	MarkPrice       float64 `json:"mark_price"`
	IndexPrice      float64 `json:"index_price"`
	// SettlementPrice is the instrument's last settlement price, zero before its first settlement, and not sent for
	// spot
	SettlementPrice float64 `json:"settlement_price"`
	Delta           float64 `json:"delta"`
	// Gamma, Vega and Theta are only sent for options
	Gamma             float64 `json:"gamma"`
	Vega              float64 `json:"vega"`
	Theta             float64 `json:"theta"`
	InitialMargin     float64 `json:"initial_margin"`
	MaintenanceMargin float64 `json:"maintenance_margin"`
	// OpenOrdersMargin is only sent for futures
	OpenOrdersMargin float64 `json:"open_orders_margin"`
	// Leverage is the leverage available to a future position; the schema documents an integer, while custom leverage
	// is any number, so a fraction is accepted
	Leverage              float64 `json:"leverage"`
	TotalProfitLoss       float64 `json:"total_profit_loss"`
	FloatingProfitLoss    float64 `json:"floating_profit_loss"`
	FloatingProfitLossUSD float64 `json:"floating_profit_loss_usd"`
	RealisedProfitLoss    float64 `json:"realized_profit_loss"`
	// RealisedFunding is a perpetual position's funding this session, which its session realised profit and loss
	// includes, and InterestValue the value it is calculated from
	RealisedFunding float64 `json:"realized_funding"`
	InterestValue   float64 `json:"interest_value"`
	// EstimatedLiquidationPrice is deprecated: the gateway no longer calculates it and sends null
	EstimatedLiquidationPrice float64 `json:"estimated_liquidation_price"`
}

// DerivativesGetPositionsRequest requests the open positions of the account or a subaccount
type DerivativesGetPositionsRequest struct {
	// Currency is BTC, ETH, USDC, USDT, EURR or any, as currency.NewCode(DerivativesCurrencyAny); it is left out when
	// empty
	Currency currency.Code
	// Kind is future, option, future_combo or option_combo; spot holdings are balances, not positions
	Kind         string
	SubaccountID uint64
	// IncludeIsolated includes the positions of the isolated margin subaccounts, which are left out otherwise
	IncludeIsolated bool
}

// DerivativesPositionsWireRequest holds private/get_positions' params
type DerivativesPositionsWireRequest struct {
	// Currency stays a string, the code in the case the gateway takes
	Currency        string `json:"currency,omitempty"`
	Kind            string `json:"kind,omitempty"`
	SubaccountID    uint64 `json:"subaccount_id,omitempty"`
	IncludeIsolated bool   `json:"include_isolated,omitempty"`
}

// DerivativesRiskProfileResponse is the account's risk profile permissions
type DerivativesRiskProfileResponse struct {
	// MaximumLeverage is the leverage ceiling, zero when unconstrained, which the gateway sends as null
	MaximumLeverage           float64 `json:"max_leverage"`
	PortfolioMarginingAllowed bool    `json:"portfolio_margining_allowed"`
	ShortOptionAllowed        bool    `json:"short_option_allowed"`
}

// DerivativesGetSubaccountsRequest requests the main account's and its subaccounts' details
type DerivativesGetSubaccountsRequest struct {
	// WithPortfolio includes each account's balances and margins
	WithPortfolio bool
	// IncludeIsolated includes the isolated margin subaccounts the gateway manages, which are hidden otherwise
	IncludeIsolated bool
}

// DerivativesSubaccountsWireRequest holds private/get_subaccounts' params
type DerivativesSubaccountsWireRequest struct {
	WithPortfolio   bool `json:"with_portfolio,omitempty"`
	IncludeIsolated bool `json:"include_isolated,omitempty"`
}

// DerivativesSubaccountResponse is the main account's or a subaccount's details; a subaccount is sent limited details
// of the main account
type DerivativesSubaccountResponse struct {
	// ID is the account's user ID, which subaccount ID parameters take
	ID uint64 `json:"id"`
	// Type is main or subaccount
	Type       string `json:"type"`
	Username   string `json:"username"`
	SystemName string `json:"system_name"`
	Email      string `json:"email"`
	// NotConfirmedEmail is a new email address awaiting confirmation, sent only with the portfolio
	NotConfirmedEmail string `json:"not_confirmed_email"`
	LoginEnabled      bool   `json:"login_enabled"`
	// IsPassword is set when the subaccount has a password
	IsPassword bool `json:"is_password"`
	// ReceiveNotifications is set when the subaccount's notification emails go to the main account's email
	ReceiveNotifications bool `json:"receive_notifications"`
	SecurityKeysEnabled  bool `json:"security_keys_enabled"`
	// SecurityKeysAssignments names the assignments with security keys
	SecurityKeysAssignments []string `json:"security_keys_assignments"`
	MarginModel             string   `json:"margin_model"`
	// Isolated is set for an isolated margin subaccount the gateway manages, which IsolatedMarginInstrument is bound
	// to; the gateway sends null for other accounts, and for an unbound subaccount's instrument, which may later be
	// bound to another instrument
	Isolated                 bool   `json:"isolated"`
	IsolatedMarginInstrument string `json:"isolated_margin_instrument"`
	// ProofID and ProofIDSignature identify the account's entries in the proof of liabilities, so should be kept
	// secret
	ProofID          string `json:"proof_id"`
	ProofIDSignature string `json:"proof_id_signature"`
	// Portfolio holds the account's balances and margins keyed by lower case currency
	Portfolio map[string]DerivativesSubaccountPortfolio `json:"portfolio"`
}

// DerivativesSubaccountPortfolio is an account's balance and margins in one currency, which the gateway sends in lower
// case
type DerivativesSubaccountPortfolio struct {
	Currency          currency.Code `json:"currency"`
	Balance           float64       `json:"balance"`
	Equity            float64       `json:"equity"`
	MarginBalance     float64       `json:"margin_balance"`
	InitialMargin     float64       `json:"initial_margin"`
	MaintenanceMargin float64       `json:"maintenance_margin"`
	// AvailableFunds aggregates every cross collateral currency, converted at each one's index, with cross collateral
	// enabled
	AvailableFunds           float64 `json:"available_funds"`
	AvailableWithdrawalFunds float64 `json:"available_withdrawal_funds"`
	// AdditionalReserve is reserved for the premium of open buy option and option combo orders, on the cross_sm margin
	// model only, and SpotReserve by open spot orders
	AdditionalReserve float64 `json:"additional_reserve"`
	SpotReserve       float64 `json:"spot_reserve"`
}

// DerivativesGetSubaccountsDetailsRequest requests the positions, and optionally the open orders, of the main account
// and its subaccounts in a currency
type DerivativesGetSubaccountsDetailsRequest struct {
	// Currency is BTC, ETH, USDC, USDT or EURR
	Currency       currency.Code
	WithOpenOrders bool
	// IncludeIsolated includes the isolated margin subaccounts' positions and open orders, which are left out
	// otherwise
	IncludeIsolated bool
}

// DerivativesSubaccountsDetailsWireRequest holds private/get_subaccounts_details' params
type DerivativesSubaccountsDetailsWireRequest struct {
	// Currency stays a string, the code in the case the gateway takes
	Currency        string `json:"currency"`
	WithOpenOrders  bool   `json:"with_open_orders,omitempty"`
	IncludeIsolated bool   `json:"include_isolated,omitempty"`
}

// DerivativesSubaccountDetailsResponse is the main account's or a subaccount's positions in a currency and, when asked
// for, its open orders
type DerivativesSubaccountDetailsResponse struct {
	UserID     uint64                        `json:"uid"`
	Positions  []DerivativesPositionResponse `json:"positions"`
	OpenOrders []DerivativesOrderResponse    `json:"open_orders"`
}

// DerivativesTradingLimitsResponse is the account's open order and position limits in a currency and margin model;
// the gateway leaves out limits that do not apply
type DerivativesTradingLimitsResponse struct {
	Currency                          currency.Code `json:"currency"`
	MaximumFutureOpenOrders           uint64        `json:"max_future_open_orders"`
	MaximumFutureOpenOrdersSize       float64       `json:"max_future_open_orders_size"`
	MaximumFuturePositionSize         float64       `json:"max_future_position_size"`
	MaximumOpenOrdersInInstrument     uint64        `json:"max_open_orders_in_instrument"`
	MaximumOptionOpenOrders           uint64        `json:"max_option_open_orders"`
	MaximumOptionOpenOrdersSize       float64       `json:"max_option_open_orders_size"`
	MaximumRiskReducingOrders         uint64        `json:"max_risk_reducing_orders"`
	MaximumTotalOpenOrders            uint64        `json:"max_total_open_orders"`
	MaximumTotalShortOptionsPositions float64       `json:"max_total_short_options_positions"`
	PortfolioMarginingEnabled         bool          `json:"portfolio_margining_enabled"`
}

// DerivativesGetTransactionLogRequest requests a page of the account's transaction log in a currency, newest first
type DerivativesGetTransactionLogRequest struct {
	// Currency is BTC, ETH, STETH, ETHW, USDC, USDT, EURR, SOL, XRP, USYC, PAXG, BNB or USDE
	Currency currency.Code
	// StartTime and EndTime are required; the log goes back to the account's creation
	StartTime time.Time
	EndTime   time.Time
	// Query holds keywords an entry must match, such as trade, maker, taker, open, close, liquidation, buy, sell,
	// withdrawal, delivery, settlement, deposit, transfer, option, future, correction, block_trade, swap or expiry, or a
	// withdrawal or transfer address
	Query string
	// Count defaults to 100, at most 250
	Count uint64
	// SubaccountID selects a subaccount; an isolated margin subaccount's history is only read this way
	SubaccountID uint64
	// Continuation is the previous page's continuation, which selects the next older page
	Continuation uint64
}

// DerivativesTransactionLogWireRequest holds private/get_transaction_log's params, with the window in Unix milliseconds
type DerivativesTransactionLogWireRequest struct {
	// Currency stays a string, the code in the case the gateway takes
	Currency       string `json:"currency"`
	StartTimestamp uint64 `json:"start_timestamp"`
	EndTimestamp   uint64 `json:"end_timestamp"`
	Query          string `json:"query,omitempty"`
	Count          uint64 `json:"count,omitempty"`
	SubaccountID   uint64 `json:"subaccount_id,omitempty"`
	Continuation   uint64 `json:"continuation,omitempty"`
}

// DerivativesTransactionLogResponse is a page of the transaction log and the continuation of the next older page,
// zero when there is none
type DerivativesTransactionLogResponse struct {
	Logs         []DerivativesTransactionLogEntry `json:"logs"`
	Continuation uint64                           `json:"continuation"`
}

// DerivativesTransactionLogEntry is a transaction log entry, whose fields depend on its type
type DerivativesTransactionLogEntry struct {
	ID uint64 `json:"id"`
	// UserSequence numbers the user's entries in the currency
	UserSequence uint64        `json:"user_seq"`
	UserID       uint64        `json:"user_id"`
	Username     string        `json:"username"`
	Currency     currency.Code `json:"currency"`
	Timestamp    types.Time    `json:"timestamp"`
	// Type is the entry's category, such as trade, deposit, withdrawal, settlement, delivery, transfer, swap,
	// correction, expiry, LSP transfer, LSP commission, ADL transfer or ADL commission; the gateway adds types over
	// time
	Type string `json:"type"`
	// Role is source or destination for a position move, which both sides log with the same type
	Role string `json:"role"`
	// Info holds the entry's additional information as sent: a string such as "Source: api" for a trade, an object for
	// a transfer and other types, or null
	Info json.RawMessage `json:"info"`
	// Cashflow is a future's realised session profit and loss, or an option's premium paid or received, while Change
	// is the cash balance's change
	Cashflow       float64 `json:"cashflow"`
	Change         float64 `json:"change"`
	Balance        float64 `json:"balance"`
	Equity         float64 `json:"equity"`
	Commission     float64 `json:"commission"`
	InstrumentName string  `json:"instrument_name"`
	// Side is open buy, open sell, close buy or close sell for a trade or delivery, long or short for a settlement, and
	// - for an entry without one
	Side string `json:"side"`
	// Amount is in USD for perpetuals and inverse futures, and in the base currency for options and linear futures
	Amount          float64       `json:"amount"`
	Contracts       float64       `json:"contracts"`
	Price           float64       `json:"price"`
	PriceCurrency   currency.Code `json:"price_currency"`
	Position        float64       `json:"position"`
	MarkPrice       float64       `json:"mark_price"`
	IndexPrice      float64       `json:"index_price"`
	SettlementPrice float64       `json:"settlement_price"`
	TradeID         string        `json:"trade_id"`
	OrderID         string        `json:"order_id"`
	// UserRole is maker or taker, and FeeRole the role the fee was charged at, which an iceberg order can make differ
	UserRole string `json:"user_role"`
	FeeRole  string `json:"fee_role"`
	// ProfitAsCashflow is set when the cashflow is waiting for settlement
	ProfitAsCashflow            bool    `json:"profit_as_cashflow"`
	SessionRealisedProfitLoss   float64 `json:"session_rpl"`
	SessionUnrealisedProfitLoss float64 `json:"session_upl"`
	// InterestProfitLoss is a perpetual's funding since the previous trade or position change, and
	// TotalInterestProfitLoss the session's funding since the daily settlement at 08:00 UTC
	InterestProfitLoss      float64 `json:"interest_pl"`
	TotalInterestProfitLoss float64 `json:"total_interest_pl"`
	BlockRFQID              uint64  `json:"block_rfq_id"`
	// IP is the address the trade was placed from
	IP string `json:"ip"`
	// The Starbase fields are only sent for trades matched in Starbase; StarbaseTimestamp is in nanoseconds
	StarbaseMatchID   uint64     `json:"starbase_match_id"`
	StarbaseOrderID   uint64     `json:"starbase_order_id"`
	StarbaseTimestamp types.Time `json:"starbase_timestamp"`
}

// DerivativesUserLockResponse is whether the account is locked in a currency
type DerivativesUserLockResponse struct {
	Currency currency.Code `json:"currency"`
	// Locked is set when the account is locked in the currency, as the documented example names it; Enabled holds it
	// where the gateway sends the name the schema documents
	Locked  bool `json:"locked"`
	Enabled bool `json:"enabled"`
	// Message explains a lock
	Message string `json:"message"`
}

// DerivativesSimulatePortfolioRequest simulates the margins of the account's portfolio, optionally with simulated
// positions, for SimulatePortfolio and PMESimulate
type DerivativesSimulatePortfolioRequest struct {
	// Currency is BTC, ETH, USDC, USDT or EURR for SimulatePortfolio, and BTC, ETH, USDC, USDT or CROSS, the cross
	// collateral portfolio, for PMESimulate
	Currency currency.Code
	// AddPositions adds the simulated positions to the account's positions when nil or true, as the gateway defaults
	// to, and simulates them alone when false
	AddPositions *bool
	// SimulatedPositions holds each instrument's simulated position size, in the units the method documents
	SimulatedPositions map[string]float64
}

// DerivativesSimulatePortfolioWireRequest holds private/simulate_portfolio's and private/pme/simulate's params
type DerivativesSimulatePortfolioWireRequest struct {
	// Currency stays a string, the code in the case the gateway takes
	Currency           string             `json:"currency"`
	AddPositions       *bool              `json:"add_positions,omitempty"`
	SimulatedPositions map[string]float64 `json:"simulated_positions,omitempty"`
}

// DerivativesPMESimulateResponse is a portfolio margin simulation: the model's parameters, the risk vectors of each
// instrument and currency, the resulting margins, and the portfolio, index prices and tickers simulated; the schema
// documents an object, so its fields follow the documented example
type DerivativesPMESimulateResponse struct {
	ModelParameters DerivativesPMEModelParameters `json:"model_params"`
	// AggregatedRiskVectors is keyed by base and settlement currency, such as btc_btc
	AggregatedRiskVectors map[string]DerivativesPMERiskVectors `json:"aggregated_risk_vectors"`
	// InitialRiskVectors is keyed by instrument
	InitialRiskVectors map[string]DerivativesPMERiskVectors `json:"initial_risk_vectors"`
	// Margins is keyed by lower case settlement currency
	Margins   map[string]DerivativesPMEMargins `json:"margins"`
	Portfolio DerivativesPMEPortfolio          `json:"portfolio"`
	// IndexPrices is keyed by index name, such as btc_usd
	IndexPrices map[string]float64 `json:"index_price"`
	// Tickers is keyed by instrument
	Tickers map[string]DerivativesPMETicker `json:"ticker"`
}

// DerivativesPMEModelParameters are the portfolio margin model's parameters
type DerivativesPMEModelParameters struct {
	// CurrencyPairs is keyed by index name, such as btc_usd
	CurrencyPairs map[string]DerivativesPMECurrencyPairParameters `json:"currency_pair"`
	// Currencies is keyed by lower case currency
	Currencies map[string]DerivativesPMECurrencyParameters `json:"currency"`
	General    DerivativesPMEGeneralParameters             `json:"general"`
}

// DerivativesPMECurrencyPairParameters are a currency pair's portfolio margin parameters
type DerivativesPMECurrencyPairParameters struct {
	// PriceRange is the largest price move the main risk matrix simulates
	PriceRange          float64 `json:"price_range"`
	ExtendedTableFactor float64 `json:"extended_table_factor"`
	// The volatility parameters set the volatility shocks
	VolatilityRangeUp           float64 `json:"volatility_range_up"`
	VolatilityRangeDown         float64 `json:"volatility_range_down"`
	MinimumVolatilityForShockUp float64 `json:"min_volatility_for_shock_up"`
	ShortTermVegaPower          float64 `json:"short_term_vega_power"`
	LongTermVegaPower           float64 `json:"long_term_vega_power"`
	// The delta shock applies MarginIncrement, the standard margin increment, to the delta beyond
	// DeltaTotalLiquidityShockThreshold, up to MaximumDeltaShock
	MarginIncrement                   float64 `json:"m_inc"`
	DeltaTotalLiquidityShockThreshold float64 `json:"delta_total_liq_shock_threshold"`
	MaximumDeltaShock                 float64 `json:"max_delta_shock"`
}

// DerivativesPMECurrencyParameters are a currency's portfolio margin parameters
type DerivativesPMECurrencyParameters struct {
	// AnnualisedMoveRisk and MinimumAnnualisedMove, the minimum expiry delta shock, set the roll shock
	AnnualisedMoveRisk    float64 `json:"annualised_move_risk"`
	MinimumAnnualisedMove float64 `json:"min_annualised_move"`
	// ExtendedDampener dampens the extended table's simulated profit and loss
	ExtendedDampener float64 `json:"extended_dampener"`
	Haircut          float64 `json:"haircut"`
	// EquitySideImpact applies the currency's equity to the risk matrix of a cross portfolio margin account, on none or
	// both sides
	EquitySideImpact            string  `json:"equity_side_impact"`
	MaximumOffsetableProfitLoss float64 `json:"max_offsetable_pnl"`
	ProfitLossOffset            float64 `json:"pnl_offset"`
	CorrelationSet              bool    `json:"correlation_set"`
}

// DerivativesPMEGeneralParameters are the portfolio margin model's general parameters
type DerivativesPMEGeneralParameters struct {
	// MaintenanceMarginFactor is the share of the initial margin held as maintenance margin
	MaintenanceMarginFactor  float64    `json:"mm_factor"`
	BucketsCount             uint64     `json:"buckets_count"`
	VolatilityScenariosCount uint64     `json:"vol_scenarios_count"`
	Timestamp                types.Time `json:"timestamp"`
}

// DerivativesPMERiskVectors are the simulated profit and loss of the main risk matrix, each price move's volatility
// scenarios in turn, and of the extended table's larger price moves
type DerivativesPMERiskVectors struct {
	Standard []float64 `json:"standard"`
	Extended []float64 `json:"extended"`
}

// DerivativesPMEMargins are a settlement currency's simulated margins
type DerivativesPMEMargins struct {
	InitialMargin        float64                            `json:"initial_margin"`
	MaintenanceMargin    float64                            `json:"maintenance_margin"`
	InitialMarginDetails DerivativesPMEInitialMarginDetails `json:"initial_margin_details"`
}

// DerivativesPMEInitialMarginDetails are the components of a simulated initial margin
type DerivativesPMEInitialMarginDetails struct {
	RiskMatrixMargin        float64                               `json:"risk_matrix_margin"`
	RiskMatrixMarginDetails DerivativesPMERiskMatrixMarginDetails `json:"risk_matrix_margin_details"`
	SpotMargin              float64                               `json:"spot_margin"`
	MMPMargin               float64                               `json:"mmp_margin"`
	OpenOrdersMargin        float64                               `json:"open_orders_margin"`
}

// DerivativesPMERiskMatrixMarginDetails are the components of a simulated risk matrix margin
type DerivativesPMERiskMatrixMarginDetails struct {
	// WorstCaseLoss is the loss of the worst case scenario, which WorstCaseBucket locates
	WorstCaseLoss          float64                       `json:"worst_case"`
	WorstCaseBucket        DerivativesPMEWorstCaseBucket `json:"worst_case_bucket"`
	DeltaShock             float64                       `json:"delta_shock"`
	RollShock              float64                       `json:"roll_shock"`
	CorrelationContingency float64                       `json:"correlation_contingency"`
}

// DerivativesPMEWorstCaseBucket locates the worst case scenario of a risk matrix
type DerivativesPMEWorstCaseBucket struct {
	// Source is the risk vector, standard or extended, and Index the scenario's position in it
	Source string `json:"source"`
	Index  uint64 `json:"index"`
	Bucket uint64 `json:"bucket"`
	// Side is the side of the price move, such as left
	Side string `json:"side"`
}

// DerivativesPMEPortfolio is the simulated portfolio
type DerivativesPMEPortfolio struct {
	// Currencies holds each currency's simulated amount, keyed by currency
	Currencies map[string]float64 `json:"currency"`
	// Positions holds each instrument's position size
	Positions map[string]float64 `json:"position"`
}

// DerivativesPMETicker is an instrument's prices in a simulation
type DerivativesPMETicker struct {
	MarkPrice  float64 `json:"mark_price"`
	IndexPrice float64 `json:"index_price"`
}

// DerivativesSimulatePortfolioResponse is the simulated portfolio's balances, margins, profit and loss and greeks
type DerivativesSimulatePortfolioResponse struct {
	DerivativesPortfolioSummary
	AvailableSubaccountTransferFunds float64 `json:"available_subaccount_transfer_funds"`
}

// DerivativesGetAnnouncementsRequest requests the platform's announcements, newest first
type DerivativesGetAnnouncementsRequest struct {
	// StartTime is the latest publication time returned, now when zero
	StartTime time.Time
	// Count defaults to 5; the gateway returns at most 50
	Count uint64
}

// DerivativesAnnouncementsWireRequest holds public/get_announcements' params, with the start time in Unix milliseconds
type DerivativesAnnouncementsWireRequest struct {
	StartTimestamp uint64 `json:"start_timestamp,omitempty"`
	Count          uint64 `json:"count,omitempty"`
}

// DerivativesAnnouncementResponse is a platform announcement
type DerivativesAnnouncementResponse struct {
	ID    uint64 `json:"id"`
	Title string `json:"title"`
	// Body is HTML
	Body      string `json:"body"`
	Important bool   `json:"important"`
	// Confirmation is set when the user must confirm the announcement
	Confirmation         bool       `json:"confirmation"`
	PublicationTimestamp types.Time `json:"publication_timestamp"`
}
