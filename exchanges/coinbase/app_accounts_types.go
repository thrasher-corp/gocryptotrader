package coinbase

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Coinbase App account types; Coinbase may add more
const (
	AppAccountTypeWallet = "wallet"
	AppAccountTypeFiat   = "fiat"
	AppAccountTypeVault  = "vault"
	// AppAccountTypeCollateral is a loan collateral account, listed even when it holds nothing
	AppAccountTypeCollateral = "collateral"
)

// AppMoney is an amount of a currency as the Coinbase App API sends it
type AppMoney struct {
	Amount   types.Number  `json:"amount"`
	Currency currency.Code `json:"currency"`
}

// AppCountry is a country, by its ISO 3166-1 alpha-2 code
type AppCountry struct {
	Code string `json:"code"`
	Name string `json:"name"`
	// IsInEurope is undocumented
	IsInEurope bool `json:"is_in_europe"`
}

// AppUser is the user the credentials act for. Email is sent to OAuth2 tokens granted identity:user:email; the
// fields from Username on are undocumented
type AppUser struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	AvatarURL      string        `json:"avatar_url"`
	Resource       string        `json:"resource"`
	ResourcePath   string        `json:"resource_path"`
	TimeZone       string        `json:"time_zone"`
	NativeCurrency currency.Code `json:"native_currency"`
	// BitcoinUnit is the unit bitcoin amounts are shown in
	BitcoinUnit                           string            `json:"bitcoin_unit"`
	Country                               AppCountry        `json:"country"`
	CreatedAt                             time.Time         `json:"created_at"`
	Email                                 string            `json:"email"`
	Username                              string            `json:"username"`
	ProfileLocation                       string            `json:"profile_location"`
	ProfileBio                            string            `json:"profile_bio"`
	ProfileURL                            string            `json:"profile_url"`
	LegacyID                              string            `json:"legacy_id"`
	State                                 string            `json:"state"`
	Nationality                           *AppCountry       `json:"nationality"`
	RegionSupportsFiatTransfers           bool              `json:"region_supports_fiat_transfers"`
	RegionSupportsCryptoToCryptoTransfers bool              `json:"region_supports_crypto_to_crypto_transfers"`
	SupportsRewards                       bool              `json:"supports_rewards"`
	Tiers                                 *AppUserTiers     `json:"tiers"`
	ReferralMoney                         *AppReferralMoney `json:"referral_money"`
	HasBlockingBuyRestrictions            bool              `json:"has_blocking_buy_restrictions"`
	HasMadeAPurchase                      bool              `json:"has_made_a_purchase"`
	HasBuyDepositPaymentMethods           bool              `json:"has_buy_deposit_payment_methods"`
	HasUnverifiedBuyDepositPaymentMethods bool              `json:"has_unverified_buy_deposit_payment_methods"`
	NeedsKYCRemediation                   bool              `json:"needs_kyc_remediation"`
	ShowInstantACHUX                      bool              `json:"show_instant_ach_ux"`
	UserType                              string            `json:"user_type"`
	SendsDisabled                         bool              `json:"sends_disabled"`
}

// AppUserTiers describes a user's verification tier, as the Coinbase App shows it
type AppUserTiers struct {
	CompletedDescription string `json:"completed_description"`
	UpgradeButtonText    string `json:"upgrade_button_text"`
	Header               string `json:"header"`
	Body                 string `json:"body"`
}

// AppReferralMoney is what a user earns for a referral, and the amount the referred user must buy to earn it
type AppReferralMoney struct {
	Amount            types.Number  `json:"amount"`
	Currency          currency.Code `json:"currency"`
	CurrencySymbol    string        `json:"currency_symbol"`
	ReferralThreshold types.Number  `json:"referral_threshold"`
}

// AppPersonalDetails is a user's legal name and KYC verified primary address. OAuth2 tokens get the name with
// identity:user:name:read and the address with identity:user:address:read
type AppPersonalDetails struct {
	FirstName      string              `json:"firstName"`
	LastName       string              `json:"lastName"`
	PrimaryAddress *AppPersonalAddress `json:"primaryAddress"`
}

// AppPersonalAddress is a user's primary postal address
type AppPersonalAddress struct {
	Line1      string     `json:"line1"`
	City       string     `json:"city"`
	State      string     `json:"state"`
	PostalCode string     `json:"postalCode"`
	Country    AppCountry `json:"country"`
}

// AppListAccountsRequest selects a page of the accounts the credentials can access
type AppListAccountsRequest struct {
	// IncludeAvailableToSend sets AvailableToSend on each eligible account; only USDC accounts are eligible for now
	IncludeAvailableToSend bool
	// Page selects the page; nil selects the first
	Page *AppPageRequest
}

// AppGetAccountRequest selects an account by its ID, or a currency's primary account by the currency; exactly one of
// the two must be set
type AppGetAccountRequest struct {
	AccountID string
	Currency  currency.Code
	// IncludeAvailableToSend sets AvailableToSend when the account is eligible; only USDC accounts are for now
	IncludeAvailableToSend bool
}

// AppAccount is a wallet, fiat account, vault or loan collateral account. AllowDeposits, AllowWithdrawals and
// PortfolioID are undocumented
type AppAccount struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Primary is set for a currency's primary account, which is always a wallet
	Primary  bool        `json:"primary"`
	Type     string      `json:"type"`
	Currency AppCurrency `json:"currency"`
	Balance  AppMoney    `json:"balance"`
	// AvailableToSend is what can be sent from the account now, after holds and the user's send limits, so it may be
	// less than Balance; it is nil unless asked for, and for accounts it is unavailable for, and an actual zero is sent
	// as zero. Send limits apply across a user's accounts, so their AvailableToSend amounts cannot be added up
	AvailableToSend  *AppMoney `json:"available_to_send"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Resource         string    `json:"resource"`
	ResourcePath     string    `json:"resource_path"`
	Ready            bool      `json:"ready"`
	AllowDeposits    bool      `json:"allow_deposits"`
	AllowWithdrawals bool      `json:"allow_withdrawals"`
	PortfolioID      string    `json:"portfolio_id"`
}
