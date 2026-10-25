package cryptodotcom

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Balance history timeframes, the interval between the points of a balance history
const (
	BalanceHistoryTimeframeHourly = "H1"
	BalanceHistoryTimeframeDaily  = "D1"
)

// Account hierarchies, whether an account is the master account or one of its sub-accounts
const (
	AccountHierarchyMaster     = "MASTER"
	AccountHierarchySubAccount = "SUB_ACCOUNT"
)

// Access levels of an account's margin and derivatives trading
const (
	AccessLevelDefault  = "DEFAULT"
	AccessLevelDisabled = "DISABLED"
)

// BalanceSummary holds the margin and risk totals Crypto.com reports for an account or an isolated position, which
// the balance methods and the user.balance and user.account_risk channels share
type BalanceSummary struct {
	InstrumentName             string              `json:"instrument_name"`
	TotalAvailableBalance      types.PreciseNumber `json:"total_available_balance"`
	TotalMarginBalance         types.PreciseNumber `json:"total_margin_balance"`
	TotalInitialMargin         types.PreciseNumber `json:"total_initial_margin"`
	TotalMaintenanceMargin     types.PreciseNumber `json:"total_maintenance_margin"`
	TotalPositionCost          types.PreciseNumber `json:"total_position_cost"`
	TotalCashBalance           types.PreciseNumber `json:"total_cash_balance"`
	TotalCollateralValue       types.PreciseNumber `json:"total_collateral_value"`
	TotalSessionUnrealisedPNL  types.PreciseNumber `json:"total_session_unrealized_pnl"`
	TotalSessionRealisedPNL    types.PreciseNumber `json:"total_session_realized_pnl"`
	IsLiquidating              bool                `json:"is_liquidating"`
	TotalEffectiveLeverage     types.Number        `json:"total_effective_leverage"`
	PositionLimit              types.PreciseNumber `json:"position_limit"`
	UsedPositionLimit          types.PreciseNumber `json:"used_position_limit"`
	TotalIsolatedCashBalance   types.PreciseNumber `json:"total_isolated_cash_balance"`
	TotalPositionInitialMargin types.PreciseNumber `json:"total_position_im"`
	TotalHaircut               types.PreciseNumber `json:"total_haircut"`
	TotalRiskExposure          types.PreciseNumber `json:"total_risk_exposure"`
	TotalBorrow                types.PreciseNumber `json:"total_borrow"`
	MarginScore                types.Number        `json:"margin_score"`
	HasRisk                    bool                `json:"has_risk"`
	Terminatable               bool                `json:"terminatable"`
	// CreditLimits are left undecoded, as the docs leave their shape to be decided
	CreditLimits []json.RawMessage `json:"credit_limits"`
}

// CollateralBalance is an account's balance of one collateral; the user.account_risk channel reports it under balances,
// and the user.position_balance channel reports only its instrument name, quantity and update time
type CollateralBalance struct {
	// InstrumentName stays a string, as collateral tokens such as SOL.staked-COLLATERAL are not plain currencies
	InstrumentName string              `json:"instrument_name"`
	Quantity       types.PreciseNumber `json:"quantity"`
	MarketValue    types.PreciseNumber `json:"market_value"`
	// CollateralEligible is a types.Boolean, as the docs warn some methods send it as a string
	CollateralEligible   types.Boolean       `json:"collateral_eligible"`
	Haircut              types.Number        `json:"haircut"`
	CollateralAmount     types.PreciseNumber `json:"collateral_amount"`
	MaxWithdrawalBalance types.PreciseNumber `json:"max_withdrawal_balance"`
	ReservedQuantity     types.PreciseNumber `json:"reserved_qty"`
	HourlyInterestRate   types.Number        `json:"hourly_interest_rate"`
	UpdateTime           types.Time          `json:"update_timestamp_ms"`
}

// UserBalanceResponse is an account's balance, as user-balance, get-subaccount-balances and the user.balance channel
// report it
type UserBalanceResponse struct {
	BalanceSummary
	PositionBalances  []CollateralBalance       `json:"position_balances"`
	IsolatedPositions []IsolatedPositionBalance `json:"isolated_positions"`
	// SubAccountID is only set by get-subaccount-balances
	SubAccountID string `json:"account"`
}

// IsolatedPositionBalance is the balance of an isolated position, which is margined apart from the account
type IsolatedPositionBalance struct {
	BalanceSummary
	PositionBalances []CollateralBalance `json:"position_balances"`
	IsolationID      string              `json:"isolation_id"`
	Leverage         types.Number        `json:"leverage"`
	IsolationType    string              `json:"isolation_type"`
}

// UserBalanceHistoryRequest filters a balance history; every field is optional
type UserBalanceHistoryRequest struct {
	// Timeframe is a BalanceHistoryTimeframe constant, daily when empty
	Timeframe string
	// EndTime is exclusive, and the current time when zero
	EndTime time.Time
	// Limit is at most 30 points daily or 120 hourly
	Limit uint64
}

// userBalanceHistoryParams is the params of a user-balance-history request; the end time is sent in nanoseconds,
// which Crypto.com accepts besides milliseconds
type userBalanceHistoryParams struct {
	Timeframe string `json:"timeframe,omitempty"`
	EndTime   uint64 `json:"end_time,omitempty,string"`
	Limit     uint64 `json:"limit,omitempty,string"`
}

// UserBalanceHistoryResponse is an account's balance history
type UserBalanceHistoryResponse struct {
	InstrumentName string                `json:"instrument_name"`
	Balances       []BalanceHistoryPoint `json:"data"`
}

// BalanceHistoryPoint is an account's total cash balance at a point in time
type BalanceHistoryPoint struct {
	Time             types.Time          `json:"t"`
	TotalCashBalance types.PreciseNumber `json:"c"`
}

// AccountsRequest pages through an account's sub-accounts
type AccountsRequest struct {
	// PageSize is 20 when zero
	PageSize uint64 `json:"page_size,omitempty,string"`
	// Page counts from 0
	Page uint64 `json:"page,omitempty,string"`
}

// AccountsResponse is the API key's account, its master account and the master account's sub-accounts
type AccountsResponse struct {
	ThisAccount   CurrentAccount `json:"this_account"`
	MasterAccount Account        `json:"master_account"`
	SubAccounts   []Account      `json:"sub_account_list"`
}

// CurrentAccount identifies the account of the API key making the request
type CurrentAccount struct {
	UUID      string `json:"uuid"`
	Hierarchy string `json:"hierarchy"`
}

// Account is a master account or a sub-account
type Account struct {
	UUID                           string     `json:"uuid"`
	Hierarchy                      string     `json:"hierarchy"`
	MasterAccountUUID              string     `json:"master_account_uuid"`
	MarginAccountUUID              string     `json:"margin_account_uuid"`
	Label                          string     `json:"label"`
	Enabled                        bool       `json:"enabled"`
	Tradable                       bool       `json:"tradable"`
	Name                           string     `json:"name"`
	Email                          string     `json:"email"`
	MobileNumber                   string     `json:"mobile_number"`
	CountryCode                    string     `json:"country_code"`
	Address                        string     `json:"address"`
	MarginAccess                   string     `json:"margin_access"`
	DerivativesAccess              string     `json:"derivatives_access"`
	CreateTime                     types.Time `json:"create_time"`
	UpdateTime                     types.Time `json:"update_time"`
	TwoFactorAuthenticationEnabled bool       `json:"two_fa_enabled"`
	KYCLevel                       string     `json:"kyc_level"`
	Suspended                      bool       `json:"suspended"`
	Terminated                     bool       `json:"terminated"`
}

// SubAccountTransferRequest moves funds between the master account and its sub-accounts; every field is required
type SubAccountTransferRequest struct {
	FromAccountUUID string        `json:"from"`
	ToAccountUUID   string        `json:"to"`
	Currency        currency.Code `json:"currency"`
	Amount          types.Number  `json:"amount"`
}

// PositionsRequest filters the open positions
type PositionsRequest struct {
	// InstrumentName is a derivative such as BTCUSD-PERP, every instrument's positions when empty
	InstrumentName string `json:"instrument_name,omitempty"`
}

// PositionResponse is an open derivatives position, as get-positions and the user.positions, user.account_risk and
// user.position_balance channels report it. Only the user.positions and user.account_risk channels set the margins,
// prices, leverages and market value, which get-positions leaves out
type PositionResponse struct {
	AccountID      string `json:"account_id"`
	InstrumentName string `json:"instrument_name"`
	// InstrumentType is an InstrumentType constant
	InstrumentType string `json:"type"`
	// Quantity is negative for a short position
	Quantity                  types.PreciseNumber `json:"quantity"`
	Cost                      types.PreciseNumber `json:"cost"`
	OpenPositionPNL           types.PreciseNumber `json:"open_position_pnl"`
	OpenPositionCost          types.PreciseNumber `json:"open_pos_cost"`
	SessionPNL                types.PreciseNumber `json:"session_pnl"`
	UpdateTime                types.Time          `json:"update_timestamp_ms"`
	IsolationID               string              `json:"isolation_id"`
	IsolationType             string              `json:"isolation_type"`
	SessionUnrealisedPNL      types.PreciseNumber `json:"session_unrealized_pnl"`
	PositionInitialMargin     types.PreciseNumber `json:"pos_initial_margin"`
	PositionMaintenanceMargin types.PreciseNumber `json:"pos_maintenance_margin"`
	MarketValue               types.PreciseNumber `json:"market_value"`
	MarkPrice                 types.Number        `json:"mark_price"`
	TargetLeverage            types.Number        `json:"target_leverage"`
	Leverage                  types.Number        `json:"leverage"`
	LiquidationPrice          types.Number        `json:"liquidation_price"`
	IsolatedMarginBalance     types.PreciseNumber `json:"isolated_margin_balance"`
	EffectiveLeverage         types.Number        `json:"effective_leverage"`
}

// FeeCreditBalancesRequest pages through an account's fee credits
type FeeCreditBalancesRequest struct {
	// Page counts from 0
	Page uint64 `json:"page,omitempty"`
	// PageSize is at most and by default 200
	PageSize uint64 `json:"page_size,omitempty"`
}

// feeCreditBalancesResponse is the result of a get-fee-credit-balances request
type feeCreditBalancesResponse struct {
	FeeCredits []FeeCreditResponse `json:"fee_credits"`
}

// FeeCreditResponse is the fee credits expiring on one day; the campaign names, distributed amount and distributions
// are only reported to master accounts. Crypto.com updates fee credits daily at 00:00 UTC
type FeeCreditResponse struct {
	ExpiryDate types.Time `json:"expiry_date"`
	// CampaignNames lists the campaigns the credits came from, separated by commas
	CampaignNames         string                  `json:"type"`
	DistributedAmount     types.PreciseNumber     `json:"distributed_amount"`
	AvailableBalance      types.PreciseNumber     `json:"available_balance"`
	CampaignDistributions []FeeCreditDistribution `json:"details"`
}

// FeeCreditDistribution is the fee credit one campaign distributed
type FeeCreditDistribution struct {
	CampaignName      string              `json:"type"`
	DistributionDate  types.Time          `json:"distribution_date"`
	DistributedAmount types.PreciseNumber `json:"distributed_amount"`
}
