package deribit

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	errInvalidAPIKeyScopes         = errors.New("invalid API key scopes")
	errInvalidAccessLog            = errors.New("invalid access log")
	errInvalidMatchingEngineLimits = errors.New("invalid matching engine limits")
)

// APIKeyScopes lists an API key's maximum access scopes, such as account:read or trade:read_write. Deribit's schema
// documents a list, while its examples send and return a space separated string, so requests send the string and
// responses are accepted in either form
type APIKeyScopes []string

// MarshalJSON encodes the scopes as the space separated string Deribit's examples send
func (a APIKeyScopes) MarshalJSON() ([]byte, error) {
	return json.Marshal(strings.Join(a, " "))
}

// UnmarshalJSON decodes scopes sent as a space separated string or as a list
func (a *APIKeyScopes) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		return nil
	}
	if len(data) > 0 && data[0] == '"' {
		var scopes string
		if err := json.Unmarshal(data, &scopes); err != nil {
			return fmt.Errorf("%w: %w", errInvalidAPIKeyScopes, err)
		}
		*a = strings.Fields(scopes)
		return nil
	}
	var scopes []string
	if err := json.Unmarshal(data, &scopes); err != nil {
		return fmt.Errorf("%w: %w", errInvalidAPIKeyScopes, err)
	}
	*a = scopes
	return nil
}

// APIKey contains an API key and its settings
type APIKey struct {
	ID        uint64     `json:"id"`
	Timestamp types.Time `json:"timestamp"`
	ClientID  string     `json:"client_id"`
	// ClientSecret is the key's secret, or the MD5 fingerprint of an asymmetric key's public key
	ClientSecret string `json:"client_secret"`
	// PublicKey is the PEM encoded Ed25519 or RSA public key of an asymmetric key
	PublicKey string       `json:"public_key"`
	MaxScope  APIKeyScopes `json:"max_scope"`
	Enabled   bool         `json:"enabled"`
	// Default is deprecated
	Default bool   `json:"default"`
	Name    string `json:"name"`
	// EnabledFeatures lists the key's advanced features: restricted_block_trades and block_trade_approval
	EnabledFeatures []string `json:"enabled_features"`
	IPWhitelist     []string `json:"ip_whitelist"`
}

// ChangeAPIKeyNameRequest renames an API key
type ChangeAPIKeyNameRequest struct {
	ID uint64 `json:"id"`
	// Name may hold up to 16 letters, numbers and underscores
	Name string `json:"name"`
}

// ChangeMarginModelRequest changes the margin model of the account or one of its subaccounts
type ChangeMarginModelRequest struct {
	// UserID selects a subaccount; zero selects the authenticated account
	UserID uint64 `json:"user_id,omitempty"`
	// MarginModel is cross_pm, cross_sm, segregated_pm or segregated_sm
	MarginModel string `json:"margin_model"`
	// DryRun returns the change's effect without making it
	DryRun bool `json:"dry_run,omitempty"`
}

// MarginModelChange contains a margin model change's effect on one currency
type MarginModelChange struct {
	Currency currency.Code    `json:"currency"`
	OldState MarginModelState `json:"old_state"`
	NewState MarginModelState `json:"new_state"`
}

// MarginModelState contains a currency's margin rates and available balance under a margin model
type MarginModelState struct {
	MaintenanceMarginRate float64 `json:"maintenance_margin_rate"`
	InitialMarginRate     float64 `json:"initial_margin_rate"`
	AvailableBalance      float64 `json:"available_balance"`
}

// ChangeScopeInAPIKeyRequest changes an API key's maximum scope
type ChangeScopeInAPIKeyRequest struct {
	ID       uint64       `json:"id"`
	MaxScope APIKeyScopes `json:"max_scope"`
}

// ChangeSubaccountNameRequest renames a subaccount
type ChangeSubaccountNameRequest struct {
	SubaccountID uint64 `json:"sid"`
	Name         string `json:"name"`
}

// CreateAPIKeyRequest creates an API key, whose scope cannot exceed the scope of the key creating it
type CreateAPIKeyRequest struct {
	MaxScope APIKeyScopes `json:"max_scope"`
	// Name may hold up to 16 letters, numbers and underscores
	Name string `json:"name,omitempty"`
	// PublicKey is a PEM encoded Ed25519 or RSA public key, which makes the key asymmetric, signing requests with its
	// private key
	PublicKey string `json:"public_key,omitempty"`
	// EnabledFeatures lists the advanced features to enable: restricted_block_trades and block_trade_approval
	EnabledFeatures []string `json:"enabled_features,omitempty"`
}

// Subaccount contains the details of the main account or a subaccount; a created subaccount also reports its trading
// products and referrals, and listed accounts report their portfolios when asked
type Subaccount struct {
	ID uint64 `json:"id"`
	// Type is main or subaccount
	Type       string `json:"type"`
	Username   string `json:"username"`
	SystemName string `json:"system_name"`
	Email      string `json:"email"`
	// NotConfirmedEmail is a new email address awaiting confirmation, reported only with the portfolio
	NotConfirmedEmail string `json:"not_confirmed_email"`
	LoginEnabled      bool   `json:"login_enabled"`
	// IsPassword is set when the subaccount has a password
	IsPassword bool `json:"is_password"`
	// ReceiveNotifications is set when the subaccount's notification emails are sent to the main account's email
	ReceiveNotifications    bool     `json:"receive_notifications"`
	SecurityKeysEnabled     bool     `json:"security_keys_enabled"`
	SecurityKeysAssignments []string `json:"security_keys_assignments"`
	MarginModel             string   `json:"margin_model"`
	// ProofID and ProofIDSignature identify the account's entries in Deribit's proof of liabilities, so should be kept
	// secret
	ProofID          string `json:"proof_id"`
	ProofIDSignature string `json:"proof_id_signature"`
	// Portfolio holds the account's balances and margins keyed by lower case currency
	Portfolio               map[string]SubaccountPortfolio `json:"portfolio"`
	DisabledTradingProducts []string                       `json:"disabled_trading_products"`
	TradingProductsDetails  []TradingProductDetail         `json:"trading_products_details"`
	ReferralsCount          uint64                         `json:"referrals_count"`
}

// SubaccountPortfolio contains an account's balance and margins in one currency
type SubaccountPortfolio struct {
	Currency                 currency.Code `json:"currency"`
	Balance                  float64       `json:"balance"`
	Equity                   float64       `json:"equity"`
	MarginBalance            float64       `json:"margin_balance"`
	InitialMargin            float64       `json:"initial_margin"`
	MaintenanceMargin        float64       `json:"maintenance_margin"`
	AvailableFunds           float64       `json:"available_funds"`
	AvailableWithdrawalFunds float64       `json:"available_withdrawal_funds"`
	// AdditionalReserve is reserved for the premium of open buy option and option combo orders, on the cross_sm margin
	// model only
	AdditionalReserve float64 `json:"additional_reserve"`
	// SpotReserve is reserved by open spot orders
	SpotReserve float64 `json:"spot_reserve"`
}

// TradingProductDetail contains whether a trading product is enabled for an account and can be overwritten
type TradingProductDetail struct {
	Product         string `json:"product"`
	Enabled         bool   `json:"enabled"`
	Overwritable    bool   `json:"overwriteable"`
	RequiresConsent bool   `json:"requires_consent"`
}

// memberIDParams contains the member of a request
type memberIDParams struct {
	MemberID uint64 `json:"member_id"`
}

// MembersResponse contains the account's Direct Access members, the Starbase identities that may trade on the account
// or its subaccounts
type MembersResponse struct {
	Members               []Member `json:"members"`
	IsDirectAccessAllowed bool     `json:"is_direct_access_allowed"`
}

// Member contains a Direct Access member; a subaccount is only sent its members' names and states
type Member struct {
	ID         uint64 `json:"member_id"`
	MainUserID uint64 `json:"main_uid"`
	Name       string `json:"name"`
	// IsActive is set when the member has Direct Access trading rights
	IsActive bool `json:"is_active"`
	// Accounts lists the user IDs of the accounts the member may trade on
	Accounts                  []uint64   `json:"accounts"`
	LastModificationTimestamp types.Time `json:"m_tstamp"`
}

// apiKeyIDParams contains the API key of a request
type apiKeyIDParams struct {
	ID uint64 `json:"id"`
}

// EditAPIKeyRequest edits an API key's settings; settings left out are unchanged
type EditAPIKeyRequest struct {
	ID       uint64       `json:"id"`
	MaxScope APIKeyScopes `json:"max_scope"`
	// Name may hold up to 16 letters, numbers and underscores
	Name    string `json:"name,omitempty"`
	Enabled *bool  `json:"enabled,omitempty"`
	// EnabledFeatures and IPWhitelist replace the key's lists; an empty, non-nil list clears one
	EnabledFeatures []string `json:"enabled_features,omitzero"`
	IPWhitelist     []string `json:"ip_whitelist,omitzero"`
}

// GetAccessLogRequest requests a page of the account's access log
type GetAccessLogRequest struct {
	Offset uint64 `json:"offset,omitempty"`
	// Count defaults to 10, at most 1000
	Count uint64 `json:"count,omitempty"`
}

// AccessLogResponse contains a page of the account's access log
type AccessLogResponse struct {
	// RecordsTotal counts every entry of the log, which Deribit's documented example returns along with the page; its
	// schema documents a bare list of entries, which leaves it zero
	RecordsTotal uint64           `json:"records_total"`
	Entries      []AccessLogEntry `json:"data"`
}

// UnmarshalJSON decodes the page Deribit's documented example returns, or the bare list of entries its schema documents
func (a *AccessLogResponse) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '[' {
		var entries []AccessLogEntry
		if err := json.Unmarshal(data, &entries); err != nil {
			return fmt.Errorf("%w: %w", errInvalidAccessLog, err)
		}
		*a = AccessLogResponse{Entries: entries}
		return nil
	}
	type accessLogPage AccessLogResponse
	if err := json.Unmarshal(data, (*accessLogPage)(a)); err != nil {
		return fmt.Errorf("%w: %w", errInvalidAccessLog, err)
	}
	return nil
}

// AccessLogEntry contains an access log entry: a login, or a change to the account's security or API keys
type AccessLogEntry struct {
	ID        uint64     `json:"id"`
	Timestamp types.Time `json:"timestamp"`
	IP        string     `json:"ip"`
	// Country and City are estimated from the IP address
	Country string `json:"country"`
	City    string `json:"city"`
	// Log is the action, such as success, failure, new_api_key or changed_scope; Result holds the action where Deribit's
	// documented example names it result
	Log    string `json:"log"`
	Result string `json:"result"`
	// Data holds the action's details, such as a subaccount's user ID or an API key's client ID, as the JSON value
	// Deribit sends, whose form depends on the action
	Data any `json:"data"`
}

// GetAccountSummariesRequest requests the summaries of the account or a subaccount in every currency it holds
type GetAccountSummariesRequest struct {
	SubaccountID uint64 `json:"subaccount_id,omitempty"`
	// Extended includes the account's details and fees
	Extended bool `json:"extended,omitempty"`
}

// AccountSummariesResponse contains the account's summary in each currency it holds and, when extended, its details
type AccountSummariesResponse struct {
	AccountDetails
	Summaries []AccountSummary `json:"summaries"`
	// AffiliatePromotionFee, TradingProductsDetails and ReceiveNotifications are documented for the account and for
	// each summary
	AffiliatePromotionFee float64 `json:"affiliate_promotion_fee"`
	// TradingProductsDetails holds which trading products are enabled or can be overwritten, as the JSON value Deribit
	// sends, whose form is undocumented
	TradingProductsDetails any  `json:"trading_products_details"`
	ReceiveNotifications   bool `json:"receive_notifications"`
}

// AccountDetails contains an account's identity and settings, which extended summaries include
type AccountDetails struct {
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
	// SelfTradingExtendedToSubaccounts is documented as a string, so either form is accepted
	SelfTradingExtendedToSubaccounts types.Boolean `json:"self_trading_extended_to_subaccounts"`
	BlockRFQSelfMatchPrevention      bool          `json:"block_rfq_self_match_prevention"`
}

// GetAccountSummaryRequest requests the summary of the account or a subaccount in one currency
type GetAccountSummaryRequest struct {
	Currency     currency.Code `json:"currency"`
	SubaccountID uint64        `json:"subaccount_id,omitempty"`
	// Extended includes the account's details and fees
	Extended bool `json:"extended,omitempty"`
}

// AccountSummaryResponse contains the account's summary in one currency and, when extended, its details
type AccountSummaryResponse struct {
	AccountSummary
	AccountDetails
	IsDirectAccessAllowed bool `json:"is_direct_access_allowed"`
}

// AccountSummary contains an account's balances, margins, profit and loss, greeks and rate limits in one currency
type AccountSummary struct {
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
	LockedBalance     float64 `json:"locked_balance"`
	// FeeBalance can pay fees
	FeeBalance float64 `json:"fee_balance"`
	// FeeGroup is the account's volume fee discount level, sent with extended summaries when the account has one
	FeeGroup          string  `json:"fee_group"`
	InitialMargin     float64 `json:"initial_margin"`
	MaintenanceMargin float64 `json:"maintenance_margin"`
	// ProjectedInitialMargin and ProjectedMaintenanceMargin leave out the instruments expiring at the nearest expiry
	ProjectedInitialMargin     float64 `json:"projected_initial_margin"`
	ProjectedMaintenanceMargin float64 `json:"projected_maintenance_margin"`
	// CloseOutMargin is half the positions' maintenance margin, below which close-out liquidation takes over; Deribit
	// only sends it where close-out margin is enabled
	CloseOutMargin          float64 `json:"close_out_margin"`
	ProjectedCloseOutMargin float64 `json:"projected_close_out_margin"`
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
	DeltaTotalByIndex         map[string]float64 `json:"delta_total_map"`
	DepositAddress            string             `json:"deposit_address"`
	MarginModel               string             `json:"margin_model"`
	PortfolioMarginingEnabled bool               `json:"portfolio_margining_enabled"`
	CrossCollateralEnabled    bool               `json:"cross_collateral_enabled"`
	// The USD totals cover every cross collateral currency, and are only sent to cross margin accounts
	TotalEquityUSD            float64       `json:"total_equity_usd"`
	TotalInitialMarginUSD     float64       `json:"total_initial_margin_usd"`
	TotalMaintenanceMarginUSD float64       `json:"total_maintenance_margin_usd"`
	TotalMarginBalanceUSD     float64       `json:"total_margin_balance_usd"`
	TotalDeltaTotalUSD        float64       `json:"total_delta_total_usd"`
	Limits                    AccountLimits `json:"limits"`
	// HasNonBlockchainEquity is set when the account holds equity left out of proof of reserves
	HasNonBlockchainEquity bool `json:"has_non_block_chain_equity"`
	// Fees holds the account's discounted fees, sent with extended summaries; it is keyed by index name, such as
	// btc_usd, and then by instrument type: future, perpetual or option
	Fees                  map[string]map[string]AccountFee `json:"fees"`
	AffiliatePromotionFee float64                          `json:"affiliate_promotion_fee"`
	// TradingProductsDetails holds which trading products are enabled or can be overwritten, as the JSON value Deribit
	// sends, whose form is undocumented
	TradingProductsDetails any  `json:"trading_products_details"`
	ReceiveNotifications   bool `json:"receive_notifications"`
}

// AccountFee contains the account's fee rates for an instrument type
type AccountFee struct {
	Default    AccountFeeRates `json:"default"`
	BlockTrade float64         `json:"block_trade"`
}

// AccountFeeRates contains maker and taker fee rates
type AccountFeeRates struct {
	// Type is fixed for a fee rate, or relative for a fraction of the instrument's fee
	Type  string  `json:"type"`
	Maker float64 `json:"maker"`
	Taker float64 `json:"taker"`
}

// AccountLimits contains the account's rate limits
type AccountLimits struct {
	// LimitsPerCurrency is set when the matching engine's trading and quote limits are set per settlement currency
	LimitsPerCurrency bool                 `json:"limits_per_currency"`
	NonMatchingEngine AccountRateLimit     `json:"non_matching_engine"`
	MatchingEngine    MatchingEngineLimits `json:"matching_engine"`
}

// AccountRateLimit contains a sustained rate of requests a second and the burst of requests allowed
type AccountRateLimit struct {
	Rate  uint64 `json:"rate"`
	Burst uint64 `json:"burst"`
}

// MatchingEngineLimits contains the matching engine's rate limits
type MatchingEngineLimits struct {
	// Spot applies to spot trading, and CancelAll to cancelling every order, or every order with a label, without a
	// currency
	Spot      AccountRateLimit `json:"spot"`
	CancelAll AccountRateLimit `json:"cancel_all"`
	// MatchingEngineTradingLimits holds the trading and quote limits when they apply to every currency
	MatchingEngineTradingLimits
	// Currencies holds the trading and quote limits keyed by lower case settlement currency, when they are set per
	// settlement currency, which Deribit sends as keys of the matching engine object
	Currencies map[string]MatchingEngineTradingLimits `json:"-"`
}

// UnmarshalJSON decodes the matching engine limits that apply to every currency and those set per settlement currency;
// quotes, max_quotes and guaranteed_quotes, the names Deribit's documented account summary example gives them, decode
// as maximum_quotes, maximum_mass_quotes and guaranteed_mass_quotes
func (m *MatchingEngineLimits) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return fmt.Errorf("%w: %w", errInvalidMatchingEngineLimits, err)
	}
	limits := MatchingEngineLimits{}
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
			var currencyLimits MatchingEngineTradingLimits
			if err := json.Unmarshal(value, &currencyLimits); err != nil {
				return fmt.Errorf("%w: %s: %w", errInvalidMatchingEngineLimits, key, err)
			}
			if limits.Currencies == nil {
				limits.Currencies = make(map[string]MatchingEngineTradingLimits)
			}
			limits.Currencies[key] = currencyLimits
			continue
		}
		if err := json.Unmarshal(value, target); err != nil {
			return fmt.Errorf("%w: %s: %w", errInvalidMatchingEngineLimits, key, err)
		}
	}
	*m = limits
	return nil
}

// MatchingEngineTradingLimits contains the matching engine's trading and quote rate limits
type MatchingEngineTradingLimits struct {
	// Trading is keyed by product: total covers every product, and others, such as perpetuals, one product
	Trading              map[string]AccountRateLimit `json:"trading"`
	MaximumQuotes        AccountRateLimit            `json:"maximum_quotes"`
	MaximumMassQuotes    AccountRateLimit            `json:"maximum_mass_quotes"`
	GuaranteedMassQuotes AccountRateLimit            `json:"guaranteed_mass_quotes"`
}

// AffiliateProgramInfoResponse contains the account's affiliate program status, link and payouts
type AffiliateProgramInfoResponse struct {
	IsEnabled          bool                    `json:"is_enabled"`
	NumberOfAffiliates float64                 `json:"number_of_affiliates"`
	Link               string                  `json:"link"`
	Received           AffiliateProgramPayouts `json:"received"`
}

// AffiliateProgramPayouts contains the affiliate payouts received in each currency
type AffiliateProgramPayouts struct {
	BTC float64 `json:"btc"`
	ETH float64 `json:"eth"`
}

// AccountCurrency contains a currency available to the account
type AccountCurrency struct {
	Currency currency.Code `json:"currency"`
}

// Announcement contains a platform announcement
type Announcement struct {
	ID    uint64 `json:"id"`
	Title string `json:"title"`
	// Body is HTML
	Body      string `json:"body"`
	Important bool   `json:"important"`
	// Confirmation is set when the user must confirm the announcement
	Confirmation         bool       `json:"confirmation"`
	PublicationTimestamp types.Time `json:"publication_timestamp"`
}

// Position contains a position in an instrument; a closed position has a zero size and the direction zero
type Position struct {
	InstrumentName string `json:"instrument_name"`
	// Kind is future, option, spot, future_combo or option_combo
	Kind   string `json:"kind"`
	UserID uint64 `json:"user_id"`
	// Direction is buy, sell or zero
	Direction string `json:"direction"`
	// Size is in USD for futures and in the base currency for options, and SizeCurrency, for futures only, in the base
	// currency
	Size         float64 `json:"size"`
	SizeCurrency float64 `json:"size_currency"`
	AveragePrice float64 `json:"average_price"`
	// AveragePriceUSD and FloatingProfitLossUSD are only sent for options
	AveragePriceUSD float64 `json:"average_price_usd"`
	MarkPrice       float64 `json:"mark_price"`
	IndexPrice      float64 `json:"index_price"`
	// SettlementPrice is the instrument's last settlement price, zero before its first settlement, and not sent for spot
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
	// Leverage is the leverage available to a future position, which the initial margin curve tapers to a fraction for
	// large positions
	Leverage              float64 `json:"leverage"`
	TotalProfitLoss       float64 `json:"total_profit_loss"`
	FloatingProfitLoss    float64 `json:"floating_profit_loss"`
	FloatingProfitLossUSD float64 `json:"floating_profit_loss_usd"`
	RealisedProfitLoss    float64 `json:"realized_profit_loss"`
	// RealisedFunding is a perpetual position's funding this session, which its session realised profit and loss
	// includes, and InterestValue the value it is calculated from
	RealisedFunding float64 `json:"realized_funding"`
	InterestValue   float64 `json:"interest_value"`
	// EstimatedLiquidationPrice is deprecated: Deribit no longer calculates it and sends null
	EstimatedLiquidationPrice float64 `json:"estimated_liquidation_price"`
}

// GetPositionsRequest requests the account's or a subaccount's open positions
type GetPositionsRequest struct {
	// Currency is a settlement currency, or any for every currency, which Deribit prefers to requests per currency; it
	// is left out when empty
	Currency currency.Code `json:"currency,omitzero"`
	// Kind is future, option, future_combo or option_combo; spot holdings are balances, not positions
	Kind         string `json:"kind,omitempty"`
	SubaccountID uint64 `json:"subaccount_id,omitempty"`
}

// withPortfolioParams contains whether a subaccounts request includes their portfolios
type withPortfolioParams struct {
	WithPortfolio bool `json:"with_portfolio,omitempty"`
}

// GetSubaccountsDetailsRequest requests the positions, and optionally open orders, of the account and its subaccounts
// in a currency
type GetSubaccountsDetailsRequest struct {
	Currency       currency.Code `json:"currency"`
	WithOpenOrders bool          `json:"with_open_orders,omitempty"`
}

// SubaccountDetails contains the positions of the main account or a subaccount in a currency and, when asked, its open
// orders
type SubaccountDetails struct {
	UserID     uint64     `json:"uid"`
	Positions  []Position `json:"positions"`
	OpenOrders []Order    `json:"open_orders"`
}

// GetTransactionLogRequest requests a page of the account's transaction log in a currency, newest first
type GetTransactionLogRequest struct {
	Currency currency.Code `json:"currency"`
	// StartTime and EndTime bound the entries inclusively; Deribit treats a time after now as now
	StartTime time.Time `json:"-"`
	EndTime   time.Time `json:"-"`
	// Query holds case sensitive keywords, separated by spaces, which an entry must all match, such as trade, deposit,
	// transfer, settlement, maker or buy
	Query string `json:"query,omitempty"`
	// Count defaults to 100, at most 1000
	Count uint64 `json:"count,omitempty"`
	// SubaccountID selects a subaccount, including a removed one, for the main account
	SubaccountID uint64 `json:"subaccount_id,omitempty"`
	// Continuation is the previous page's continuation, which selects the next older page
	Continuation uint64 `json:"continuation,omitempty"`
}

// getTransactionLogParams contains a transaction log request's params, with its times in Unix milliseconds
type getTransactionLogParams struct {
	*GetTransactionLogRequest
	StartTimestamp int64 `json:"start_timestamp"`
	EndTimestamp   int64 `json:"end_timestamp"`
}

// TransactionLogResponse contains a page of the transaction log and the continuation of the next older page, zero when
// there is none
type TransactionLogResponse struct {
	Logs         []TransactionLogEntry `json:"logs"`
	Continuation uint64                `json:"continuation"`
}

// TransactionLogEntry contains a transaction log entry, whose fields depend on its type
type TransactionLogEntry struct {
	ID uint64 `json:"id"`
	// UserSequence numbers the user's entries in the currency; Deribit sends null for an entry without one
	UserSequence uint64        `json:"user_seq"`
	UserID       uint64        `json:"user_id"`
	Username     string        `json:"username"`
	Currency     currency.Code `json:"currency"`
	Timestamp    types.Time    `json:"timestamp"`
	// Type is the entry's category, such as trade, deposit, withdrawal, settlement, delivery, transfer, expiry, LSP
	// transfer or trade allocation; Deribit adds types over time
	Type string `json:"type"`
	// Role is source or destination for a position move, which both sides log with the same type
	Role string `json:"role"`
	// Info holds the entry's additional information as the JSON value Deribit sends: a string such as "Source: api"
	// for a trade, an object for a transfer and other types, or null
	Info any `json:"info"`
	// Related holds the entries a correction corrects
	Related []TransactionLogEntry `json:"related"`
	// Cashflow is signed and its meaning depends on the type, while Change is the cash balance's change
	Cashflow       float64 `json:"cashflow"`
	Change         float64 `json:"change"`
	Balance        float64 `json:"balance"`
	Commission     float64 `json:"commission"`
	FeeBalance     float64 `json:"fee_balance"`
	Equity         float64 `json:"equity"`
	InstrumentName string  `json:"instrument_name"`
	// Side is open buy, open sell, close buy or close sell for a trade, long or short for a settlement, and - for an
	// entry without one
	Side string `json:"side"`
	// Amount is in USD for perpetuals and inverse futures, in the base currency for options and linear futures, and in
	// cash for spot
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
	// ProfitAsCashflow is set for a future trade whose session profit and loss is in Cashflow
	ProfitAsCashflow            bool    `json:"profit_as_cashflow"`
	SessionRealisedProfitLoss   float64 `json:"session_rpl"`
	SessionUnrealisedProfitLoss float64 `json:"session_upl"`
	// InterestProfitLoss is a perpetual's funding since the previous trade or position change, and
	// TotalInterestProfitLoss the session's funding since the daily settlement at 08:00 UTC
	InterestProfitLoss      float64 `json:"interest_pl"`
	TotalInterestProfitLoss float64 `json:"total_interest_pl"`
	BlockRFQID              uint64  `json:"block_rfq_id"`
	IP                      string  `json:"ip"`
	// The Starbase fields are only sent for trades matched in Starbase; StarbaseTimestamp is in nanoseconds
	StarbaseMatchID   uint64     `json:"starbase_match_id"`
	StarbaseOrderID   uint64     `json:"starbase_order_id"`
	StarbaseTimestamp types.Time `json:"starbase_timestamp"`
}

// UserLock contains whether the account is locked in a currency
type UserLock struct {
	Currency currency.Code `json:"currency"`
	// Locked is set when the account is locked in the currency, as Deribit's documented example names it; Enabled holds
	// it where Deribit sends the name its schema documents
	Locked  bool `json:"locked"`
	Enabled bool `json:"enabled"`
	// Message explains a lock
	Message string `json:"message"`
}

// CustodyAccount contains a custody account and its balances
type CustodyAccount struct {
	Name     string        `json:"name"`
	Currency currency.Code `json:"currency"`
	// ClientID is the client ID of the API key that reserves and releases funds on the custody platform
	ClientID   string  `json:"client_id"`
	ExternalID string  `json:"external_id"`
	Balance    float64 `json:"balance"`
	// PendingWithdrawalBalance is being moved from the trading account to the custody account
	PendingWithdrawalBalance float64 `json:"pending_withdrawal_balance"`
	// AutoDeposit is set when funds added to the custody balance move to the trading balance automatically
	AutoDeposit                   bool   `json:"auto_deposit"`
	WithdrawalsRequireSecurityKey bool   `json:"withdrawals_require_security_key"`
	DepositAddress                string `json:"deposit_address"`
	WithdrawalAddress             string `json:"withdrawal_address"`
	// PendingWithdrawalAddress replaces WithdrawalAddress from WithdrawalAddressChange; its JSON name is Deribit's
	PendingWithdrawalAddress string     `json:"pending_withdrawal_addres"` //nolint:misspell // Deribit's schema spells the field so
	WithdrawalAddressChange  types.Time `json:"withdrawal_address_change"`
}

// SimulatePortfolioRequest simulates the margins of the account's portfolio, optionally with simulated positions, for
// SimulatePortfolio and PMESimulate
type SimulatePortfolioRequest struct {
	Currency currency.Code `json:"currency"`
	// AddPositions adds the simulated positions to the account's positions when nil or true, as Deribit defaults to, and
	// simulates them alone when false
	AddPositions *bool `json:"add_positions,omitempty"`
	// SimulatedPositions holds each instrument's simulated position size, in the units the method documents
	SimulatedPositions map[string]float64 `json:"simulated_positions,omitempty"`
}

// PMESimulateResponse contains a portfolio margin simulation: the model's parameters, the risk vectors of each
// instrument and currency, the resulting margins, and the portfolio, index prices and tickers simulated
type PMESimulateResponse struct {
	ModelParameters PMEModelParameters `json:"model_params"`
	// AggregatedRiskVectors is keyed by base and settlement currency, such as btc_btc
	AggregatedRiskVectors map[string]PMERiskVectors `json:"aggregated_risk_vectors"`
	// InitialRiskVectors is keyed by instrument
	InitialRiskVectors map[string]PMERiskVectors `json:"initial_risk_vectors"`
	// Margins is keyed by lower case settlement currency
	Margins   map[string]PMEMargins `json:"margins"`
	Portfolio PMEPortfolio          `json:"portfolio"`
	// IndexPrices is keyed by index name, such as btc_usd
	IndexPrices map[string]float64 `json:"index_price"`
	// Tickers is keyed by instrument
	Tickers map[string]PMETicker `json:"ticker"`
}

// PMEModelParameters contains the portfolio margin model's parameters
type PMEModelParameters struct {
	// CurrencyPairs is keyed by index name, such as btc_usd
	CurrencyPairs map[string]PMECurrencyPairParameters `json:"currency_pair"`
	// Currencies is keyed by lower case currency
	Currencies map[string]PMECurrencyParameters `json:"currency"`
	General    PMEGeneralParameters             `json:"general"`
}

// PMECurrencyPairParameters contains a currency pair's portfolio margin parameters
type PMECurrencyPairParameters struct {
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

// PMECurrencyParameters contains a currency's portfolio margin parameters
type PMECurrencyParameters struct {
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

// PMEGeneralParameters contains the portfolio margin model's general parameters
type PMEGeneralParameters struct {
	// MaintenanceMarginFactor is the share of the initial margin held as maintenance margin
	MaintenanceMarginFactor  float64    `json:"mm_factor"`
	BucketsCount             uint64     `json:"buckets_count"`
	VolatilityScenariosCount uint64     `json:"vol_scenarios_count"`
	Timestamp                types.Time `json:"timestamp"`
}

// PMERiskVectors contains the simulated profit and loss of the main risk matrix, each price move's volatility scenarios
// in turn, and of the extended table's larger price moves
type PMERiskVectors struct {
	Standard []float64 `json:"standard"`
	Extended []float64 `json:"extended"`
}

// PMEMargins contains a settlement currency's simulated margins
type PMEMargins struct {
	InitialMargin        float64                 `json:"initial_margin"`
	MaintenanceMargin    float64                 `json:"maintenance_margin"`
	InitialMarginDetails PMEInitialMarginDetails `json:"initial_margin_details"`
}

// PMEInitialMarginDetails contains the components of a simulated initial margin
type PMEInitialMarginDetails struct {
	RiskMatrixMargin        float64                    `json:"risk_matrix_margin"`
	RiskMatrixMarginDetails PMERiskMatrixMarginDetails `json:"risk_matrix_margin_details"`
	SpotMargin              float64                    `json:"spot_margin"`
	MMPMargin               float64                    `json:"mmp_margin"`
	OpenOrdersMargin        float64                    `json:"open_orders_margin"`
}

// PMERiskMatrixMarginDetails contains the components of a simulated risk matrix margin
type PMERiskMatrixMarginDetails struct {
	// WorstCaseLoss is the loss of the worst case scenario, which WorstCaseBucket locates
	WorstCaseLoss          float64            `json:"worst_case"`
	WorstCaseBucket        PMEWorstCaseBucket `json:"worst_case_bucket"`
	DeltaShock             float64            `json:"delta_shock"`
	RollShock              float64            `json:"roll_shock"`
	CorrelationContingency float64            `json:"correlation_contingency"`
}

// PMEWorstCaseBucket locates the worst case scenario of a risk matrix
type PMEWorstCaseBucket struct {
	// Source is the risk vector, standard or extended, and Index the scenario's position in it
	Source string `json:"source"`
	Index  uint64 `json:"index"`
	Bucket int64  `json:"bucket"`
	// Side is the side of the price move, such as left
	Side string `json:"side"`
}

// PMEPortfolio contains the simulated portfolio
type PMEPortfolio struct {
	// Currencies holds each currency's simulated amount, keyed by currency
	Currencies map[string]float64 `json:"currency"`
	// Positions holds each instrument's position size
	Positions map[string]float64 `json:"position"`
}

// PMETicker contains an instrument's prices in a simulation
type PMETicker struct {
	MarkPrice  float64 `json:"mark_price"`
	IndexPrice float64 `json:"index_price"`
}

// subaccountIDParams contains the subaccount of a request
type subaccountIDParams struct {
	SubaccountID uint64 `json:"subaccount_id"`
}

// announcementIDParams contains the announcement of a request
type announcementIDParams struct {
	AnnouncementID uint64 `json:"announcement_id"`
}

// SetDisabledTradingProductsRequest sets the trading products disabled for a subaccount
type SetDisabledTradingProductsRequest struct {
	UserID uint64 `json:"user_id"`
	// TradingProducts lists perpetual, futures, options, future_combos, option_combos and spots; an empty list is sent
	// as one, which disables none
	TradingProducts []string `json:"trading_products"`
}

// SetEmailForSubaccountRequest sets the email address of a subaccount, which must be confirmed before it can log in
type SetEmailForSubaccountRequest struct {
	SubaccountID uint64 `json:"sid"`
	Email        string `json:"email"`
}

// emailLanguageParams contains the language of a set email language request
type emailLanguageParams struct {
	Language string `json:"language"`
}

// SetMemberRequest creates a Direct Access member, or edits an existing one
type SetMemberRequest struct {
	// ID selects the member to edit; zero creates a member
	ID uint64 `json:"member_id,omitempty"`
	// Name is required, also when only toggling IsActive
	Name string `json:"name"`
	// IsActive defaults to true when nil; a member cannot be created inactive, nor toggled in a request changing its
	// name or accounts
	IsActive *bool `json:"is_active,omitempty"`
	// Accounts lists the user IDs of the accounts the member may trade on, which creating a member requires; editing
	// replaces the member's list, which leaving it out clears
	Accounts []uint64 `json:"accounts,omitempty"`
}

// SetSelfTradingConfigRequest configures self trading prevention
type SetSelfTradingConfigRequest struct {
	// Mode is reject_taker, which rejects the incoming order, or cancel_maker, which cancels the resting order
	Mode string `json:"mode"`
	// ExtendedToSubaccounts prevents trades between the account's subaccounts too
	ExtendedToSubaccounts bool `json:"extended_to_subaccounts"`
	// BlockRFQSelfMatchPrevention prevents Block RFQ trades between accounts of the same legal entity, separately from
	// self trading prevention; it is unchanged when nil
	BlockRFQSelfMatchPrevention *bool `json:"block_rfq_self_match_prevention,omitempty"`
}

// SimulatePortfolioResponse contains the simulated portfolio's balances, margins, profit and loss and greeks
type SimulatePortfolioResponse struct {
	AccountSummary
	AvailableSubaccountTransferFunds float64 `json:"available_subaccount_transfer_funds"`
}

// ToggleNotificationsFromSubaccountRequest enables or disables the notifications of a subaccount
type ToggleNotificationsFromSubaccountRequest struct {
	SubaccountID uint64 `json:"sid"`
	State        bool   `json:"state"`
}

// ToggleSubaccountLoginRequest enables or disables a subaccount's login
type ToggleSubaccountLoginRequest struct {
	SubaccountID uint64 `json:"sid"`
	// State is enable or disable
	State string `json:"state"`
}

// GetAnnouncementsRequest requests the platform's announcements, newest first
type GetAnnouncementsRequest struct {
	// StartTime is the latest publication time returned, now when zero
	StartTime time.Time `json:"-"`
	// Count defaults to 5, and Deribit returns at most 50
	Count uint64 `json:"count,omitempty"`
}

// getAnnouncementsParams contains an announcements request's params, with its start time in Unix milliseconds
type getAnnouncementsParams struct {
	*GetAnnouncementsRequest
	StartTimestamp int64 `json:"start_timestamp,omitempty"`
}
