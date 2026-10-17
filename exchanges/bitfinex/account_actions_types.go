package bitfinex

import (
	"bytes"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Token capabilities, which GenerateTokenRequest grants
const (
	TokenCapabilityAccount  = "a"
	TokenCapabilityOrders   = "o"
	TokenCapabilityFunding  = "f"
	TokenCapabilitySettings = "s"
	TokenCapabilityWallets  = "w"
	// TokenCapabilityWithdraw needs the token's write permission
	TokenCapabilityWithdraw    = "wd"
	TokenCapabilityBitfinexPay = "bp"
)

// Two factor authentication methods, which a TwoFactorToken validates a request with
const (
	TwoFactorMethodOneTimePassword = "otp"
	TwoFactorMethodU2F             = "u2f"
)

// TransactionMethodLightning is the deposit and withdrawal method of BTC over the Lightning Network, LNX, whose
// withdrawals pay a WithdrawalRequest's Invoice rather than an address
const TransactionMethodLightning = "lnx"

// Order and offer types GetBalanceAvailable calculates for
const (
	BalanceAvailableTypeExchange   = "EXCHANGE"
	BalanceAvailableTypeMargin     = "MARGIN"
	BalanceAvailableTypeDerivative = "DERIV"
	BalanceAvailableTypeFunding    = "FUNDING"
)

// UserInfoResponse is the account's details
type UserInfoResponse struct {
	ID        uint64     `tuple:"0"`
	Email     string     `tuple:"1"`
	Username  string     `tuple:"2"`
	CreatedAt types.Time `tuple:"3"`
	// Verified is whether the account has passed identity verification
	Verified          types.Boolean `tuple:"4"`
	VerificationLevel uint64        `tuple:"5"`
	Timezone          string        `tuple:"7"`
	Locale            string        `tuple:"8"`
	// Company is where the account is registered, bitfinex or eosfinex
	Company       string        `tuple:"9"`
	EmailVerified types.Boolean `tuple:"10"`
	// SubAccountType is kept as sent, as the docs leave its type to be decided
	SubAccountType         json.RawMessage `tuple:"12"`
	MasterAccountCreatedAt types.Time      `tuple:"14"`
	GroupID                uint64          `tuple:"15"`
	// MasterAccountID is the master account's ID when the account is a sub-account
	MasterAccountID                   uint64        `tuple:"16"`
	InheritsMasterAccountVerification types.Boolean `tuple:"17"`
	IsGroupMaster                     types.Boolean `tuple:"18"`
	GroupWithdrawEnabled              types.Boolean `tuple:"19"`
	PaperTradingEnabled               types.Boolean `tuple:"21"`
	MerchantEnabled                   types.Boolean `tuple:"22"`
	CompetitionEnabled                types.Boolean `tuple:"23"`
	// TwoFactorModes are the two factor authentication methods the account has enabled, u2f and otp
	TwoFactorModes []string `tuple:"26"`
	// IsSecuritiesMaster is deprecated
	IsSecuritiesMaster             types.Boolean `tuple:"28"`
	SecuritiesEnabled              types.Boolean `tuple:"29"`
	IsSecuritiesInvestorAccredited types.Boolean `tuple:"30"`
	// IsSecuritiesElSalvador is whether the account is verified for El Salvador securities
	IsSecuritiesElSalvador types.Boolean `tuple:"31"`
	// IsSecuritiesAIFC is whether the account is verified for Astana International Financial Centre securities
	IsSecuritiesAIFC types.Boolean `tuple:"32"`
	// AllowDisableContextSwitch is whether the account can stop its master account switching into it
	AllowDisableContextSwitch types.Boolean `tuple:"38"`
	// ContextSwitchDisabled is whether the master account is stopped from switching into the account
	ContextSwitchDisabled types.Boolean `tuple:"39"`
	HighNetWorth          types.Boolean `tuple:"42"`
	LastLoginAt           time.Time     `tuple:"44"`
	// VerificationLevelSubmitted is the highest verification level the account has applied for
	VerificationLevelSubmitted uint64 `tuple:"47"`
	// ComplianceCountries are the codes of the countries of the account's residence and nationality, from its
	// verification
	ComplianceCountries []string `tuple:"49"`
	// ComplianceResidenceCountries are the codes of the countries of the account's residence, from its verification
	ComplianceResidenceCountries []string `tuple:"50"`
	// ComplianceAccountType is individual or corporate
	ComplianceAccountType string        `tuple:"51"`
	PermittedBeneficiary  types.Boolean `tuple:"53"`
	IsMerchantEnterprise  types.Boolean `tuple:"54"`
}

// UnmarshalJSON decodes a user info array
func (u *UserInfoResponse) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, u) }

// AccountSummaryResponse is the account's fee rates, its trading volume, fees and funding earnings over the last 30
// days, and its LEO discount level
type AccountSummaryResponse struct {
	FeeRates             SummaryFeeRates             `tuple:"4"`
	TradingVolumeAndFees SummaryTradingVolumeAndFees `tuple:"5"`
	FundingEarnings      SummaryFundingEarnings      `tuple:"6"`
	LEOInfo              SummaryLEOInfo              `tuple:"9"`
}

// UnmarshalJSON decodes a summary array
func (a *AccountSummaryResponse) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, a) }

// SummaryFeeRates is the account's maker and taker fee rates, each a fraction of the amount traded; 0.001 is 0.1
// percent
type SummaryFeeRates struct {
	Maker SummaryMakerFeeRates `tuple:"0"`
	Taker SummaryTakerFeeRates `tuple:"1"`
}

// UnmarshalJSON decodes a fee info array
func (s *SummaryFeeRates) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, s) }

// SummaryMakerFeeRates is the account's maker fee rates. The docs name the first three alike, MAKER_FEE, so their
// counter currencies are taken from the taker fee rates' order: crypto, stablecoin and fiat
type SummaryMakerFeeRates struct {
	ToCrypto float64 `tuple:"0"`
	ToStable float64 `tuple:"1"`
	ToFiat   float64 `tuple:"2"`
	// DerivativeRebate is the maker rebate on derivative trades
	DerivativeRebate float64 `tuple:"5"`
}

// UnmarshalJSON decodes a maker fee info array
func (s *SummaryMakerFeeRates) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, s) }

// SummaryTakerFeeRates is the account's taker fee rates
type SummaryTakerFeeRates struct {
	// ToCrypto is the rate of trades of crypto for crypto
	ToCrypto float64 `tuple:"0"`
	// ToStable is the rate of trades of crypto for a stablecoin
	ToStable float64 `tuple:"1"`
	// ToFiat is the rate of trades of crypto for fiat
	ToFiat     float64 `tuple:"2"`
	Derivative float64 `tuple:"5"`
}

// UnmarshalJSON decodes a taker fee info array
func (s *SummaryTakerFeeRates) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, s) }

// SummaryTradingVolumeAndFees is the account's trading volume and the trading fees it paid over the last 30 days
type SummaryTradingVolumeAndFees struct {
	// Volumes are the volume traded in each currency, and in total in USD
	Volumes []SummaryTradeVolume `tuple:"0"`
	// Fees are the trading fees paid in each currency
	Fees map[currency.Code]float64 `tuple:"1"`
	// TotalFeesUSD is the USD value of all the trading fees paid
	TotalFeesUSD float64 `tuple:"2"`
}

// UnmarshalJSON decodes a trading volume and fees array
func (s *SummaryTradingVolumeAndFees) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, s)
}

// SummaryTradeVolume is the volume the account traded in a currency over the last 30 days, or in total in USD
type SummaryTradeVolume struct {
	// Currency is the currency traded, such as BTC, or "Total (USD)" for the total
	Currency string  `json:"curr"`
	Volume   float64 `json:"vol"`
	// The total alone has the volumes below, which the docs leave undescribed
	VolumeSafe     float64 `json:"vol_safe"`
	VolumeMaker    float64 `json:"vol_maker"`
	VolumeBFX      float64 `json:"vol_BFX"`
	VolumeBFXSafe  float64 `json:"vol_BFX_safe"`
	VolumeBFXMaker float64 `json:"vol_BFX_maker"`
}

// SummaryFundingEarnings is what the funding the account provided earned over the last 30 days
type SummaryFundingEarnings struct {
	// Earnings are what was earned in each currency
	Earnings map[currency.Code]float64 `tuple:"1"`
	// TotalEarningsUSD is the USD value of all that was earned
	TotalEarningsUSD float64 `tuple:"2"`
}

// UnmarshalJSON decodes a funding earnings array
func (s *SummaryFundingEarnings) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, s) }

// SummaryLEOInfo is the account's LEO discount level and the LEO it held on average over the last 30 days
type SummaryLEOInfo struct {
	Level         uint64  `json:"leo_lev"`
	AverageAmount float64 `json:"leo_amount_avg"`
}

// LoginHistoryRequest holds the window of the logins GetLoginHistory returns
type LoginHistoryRequest struct {
	Start time.Time
	End   time.Time
	// Limit is the most logins returned, at most 250
	Limit uint64
}

// Login is a login to the account
type Login struct {
	ID        uint64         `tuple:"0"`
	Time      types.Time     `tuple:"2"`
	IP        string         `tuple:"4"`
	ExtraInfo LoginExtraInfo `tuple:"7"`
}

// UnmarshalJSON decodes a login info array
func (l *Login) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, l) }

// LoginExtraInfo is a JSON object of more about a login, such as its network, location and user agent
type LoginExtraInfo json.RawMessage

// UnmarshalJSON keeps the JSON as sent, except that a string encoding a JSON object, as Bitfinex sends the info,
// gives the object
func (l *LoginExtraInfo) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '"' {
		var encoded string
		if err := json.Unmarshal(data, &encoded); err != nil {
			return err
		}
		if object := bytes.TrimSpace([]byte(encoded)); len(object) > 0 && object[0] == '{' && json.Valid(object) {
			*l = object
			return nil
		}
	}
	*l = bytes.Clone(data)
	return nil
}

// MarshalJSON encodes the info as the JSON it holds
func (l LoginExtraInfo) MarshalJSON() ([]byte, error) {
	if l == nil {
		return []byte("null"), nil
	}
	return l, nil
}

// KeyPermission is whether the API key or token reads and writes in a scope
type KeyPermission struct {
	// Scope is account, history, orders, positions, funding, settings, wallets, withdraw, ui_withdraw or bfxpay
	Scope string        `tuple:"0"`
	Read  types.Boolean `tuple:"1"`
	Write types.Boolean `tuple:"2"`
}

// UnmarshalJSON decodes a key permission array
func (k *KeyPermission) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, k) }

// GenerateTokenRequest holds the lifetime and capabilities of an authentication token
type GenerateTokenRequest struct {
	// TimeToLive is how long the token lasts, in whole seconds; zero leaves Bitfinex's default of 120 seconds
	TimeToLive time.Duration
	// Capabilities are what the token may do, such as TokenCapabilityOrders
	Capabilities []string
	// WritePermission lets the token write in its capabilities, which TokenCapabilityWithdraw needs
	WritePermission bool
	// UserIP is the IP address of the user the token is for
	UserIP string
}

// generateTokenBody is the body of a Generate Token request
type generateTokenBody struct {
	Scope           string   `json:"scope"`
	TimeToLive      int64    `json:"ttl,omitempty"`
	Capabilities    []string `json:"caps,omitempty"`
	WritePermission bool     `json:"writePermission,omitempty"`
	UserIP          string   `json:"_cust_ip,omitempty"`
}

// GenerateTokenResponse holds a generated authentication token
type GenerateTokenResponse struct {
	Token string `tuple:"0"`
}

// UnmarshalJSON decodes a token array
func (g *GenerateTokenResponse) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, g) }

// ChangelogRequest holds the window of the changes GetChangelog returns
type ChangelogRequest struct {
	Start time.Time
	End   time.Time
	// Limit is the most changes returned, at most 500
	Limit uint64
}

// ChangelogEntry is a change to the account
type ChangelogEntry struct {
	CreatedAt types.Time `tuple:"0"`
	// Log describes the change, such as "settings: api key created"
	Log string `tuple:"2"`
	// IP is the IP address the change was made from
	IP string `tuple:"5"`
	// UserAgent is the user agent of the browser the change was made with
	UserAgent string `tuple:"6"`
}

// UnmarshalJSON decodes a changelog array
func (c *ChangelogEntry) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, c) }

// TwoFactorToken validates a request with two factor authentication, by a one time password or a U2F (WebAuthn)
// assertion
type TwoFactorToken struct {
	// Method is TwoFactorMethodOneTimePassword or TwoFactorMethodU2F
	Method string `json:"method"`
	// Token is the 6 digit one time password, for TwoFactorMethodOneTimePassword
	Token string `json:"token,omitempty"`
	// Response is the assertion, for TwoFactorMethodU2F
	Response *WebAuthnAssertion `json:"response,omitempty"`
}

// WebAuthnAssertion is the result of a WebAuthn assertion, with its buffers base64url encoded
type WebAuthnAssertion struct {
	ID string `json:"id,omitempty"`
	// RawID is used over ID when set, by the Thalex transfers
	RawID    string                     `json:"rawId,omitempty"`
	Response *WebAuthnAssertionResponse `json:"response,omitempty"`
}

// WebAuthnAssertionResponse is the client's response in a WebAuthn assertion, with its buffers base64url encoded
type WebAuthnAssertionResponse struct {
	// AuthenticatorData is the authenticator data, for TransferBetweenWallets
	AuthenticatorData string `json:"authenticatorData,omitempty"`
	// AttestationObject is the authenticator data, for the Thalex transfers, which name it so
	AttestationObject string `json:"attestationObject,omitempty"`
	ClientDataJSON    string `json:"clientDataJSON,omitempty"`
	Signature         string `json:"signature,omitempty"`
	// UserHandle is used by the Thalex transfers alone, and may be empty
	UserHandle string `json:"userHandle,omitempty"`
}

// TransferBetweenWalletsRequest holds a transfer of funds between the account's wallets, or to a wallet of its
// sub-account or master account
type TransferBetweenWalletsRequest struct {
	// From and To are WalletTypeExchange, WalletTypeMargin or WalletTypeFunding
	From     string
	To       string
	Currency currency.Code
	// CurrencyTo converts the funds to another currency, such as UST to USTF0, the collateral of derivatives, which a
	// transfer to the margin wallet delivers to the derivatives wallet
	CurrencyTo currency.Code
	Amount     float64
	// DestinationEmail transfers to the sub-account or master account with this email address
	DestinationEmail string
	// DestinationUserID transfers to the sub-account or master account with this user ID
	DestinationUserID uint64
	// TwoFactorToken validates a transfer to another account authenticated by token rather than API key; it is not
	// needed with the API keys requests are signed with
	TwoFactorToken *TwoFactorToken
}

// transferBody is the body of a Transfer Between Wallets request
type transferBody struct {
	From              string          `json:"from"`
	To                string          `json:"to"`
	Currency          string          `json:"currency"`
	CurrencyTo        string          `json:"currency_to,omitempty"`
	Amount            string          `json:"amount"`
	DestinationEmail  string          `json:"email_dst,omitempty"`
	DestinationUserID uint64          `json:"user_id_dst,omitempty"`
	TwoFactorToken    *TwoFactorToken `json:"tfaToken,omitempty"`
}

// WalletTransfer is a transfer of funds between wallets
type WalletTransfer struct {
	UpdatedAt  types.Time    `tuple:"0"`
	WalletFrom string        `tuple:"1"`
	WalletTo   string        `tuple:"2"`
	Currency   currency.Code `tuple:"4"`
	// CurrencyTo is the currency the funds were converted to, if any
	CurrencyTo currency.Code `tuple:"5"`
	Amount     float64       `tuple:"7"`
}

// UnmarshalJSON decodes a transfer array
func (w *WalletTransfer) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, w) }

// DepositAddressRequest holds the method and wallet of a deposit address
type DepositAddressRequest struct {
	// Wallet is the wallet deposits to the address credit: WalletTypeExchange, WalletTypeMargin or WalletTypeFunding
	Wallet string
	// Method is the deposit method, such as bitcoin or tetheruse, which the pub:map:tx:method configuration maps to its
	// currencies
	Method string
	// Renew generates a new address, keeping the old ones valid; Bitfinex allows 15 addresses per currency a month
	Renew bool
}

// depositAddressBody is the body of a Deposit Address request
type depositAddressBody struct {
	Wallet string `json:"wallet"`
	Method string `json:"method"`
	Renew  uint64 `json:"op_renew,omitempty"`
}

// DepositAddress is a deposit address
type DepositAddress struct {
	// Method is the deposit method, such as BITCOIN
	Method   string        `tuple:"1"`
	Currency currency.Code `tuple:"2"`
	// Address is the address, or for a currency needing a tag, memo or payment ID, that tag
	Address string `tuple:"4"`
	// PoolAddress is the address shared by every account, for a currency needing a tag, memo or payment ID
	PoolAddress string `tuple:"5"`
}

// UnmarshalJSON decodes a deposit address array
func (d *DepositAddress) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, d) }

// DepositAddressListRequest holds the method and page of the deposit addresses GetDepositAddressList returns
type DepositAddressListRequest struct {
	// Method is the deposit method, such as bitcoin or tetheruse
	Method string
	// Page is the page returned, from 1, the default
	Page uint64
	// PageSize is how many addresses a page holds, from 20, the default, to 100
	PageSize uint64
}

// depositAddressListBody is the body of a Deposit Address List request
type depositAddressListBody struct {
	Method   string `json:"method"`
	Page     uint64 `json:"page,omitempty"`
	PageSize uint64 `json:"pageSize,omitempty"`
}

// WalletDepositAddress is a deposit address and the wallet deposits to it credit
type WalletDepositAddress struct {
	// Address is the address, or for a currency needing a tag, memo or payment ID, that tag
	Address    string `tuple:"0"`
	WalletType string `tuple:"1"`
}

// UnmarshalJSON decodes a deposit address list array
func (w *WalletDepositAddress) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, w) }

// generateInvoiceBody is the body of a Generate Invoice request
type generateInvoiceBody struct {
	Wallet   string `json:"wallet"`
	Currency string `json:"currency"`
	Amount   string `json:"amount"`
}

// GenerateInvoiceResponse is a Lightning Network invoice for a deposit
type GenerateInvoiceResponse struct {
	InvoiceHash string `tuple:"0"`
	Invoice     string `tuple:"1"`
	// Amount is in BTC
	Amount types.Number `tuple:"4"`
}

// UnmarshalJSON decodes an invoice array
func (g *GenerateInvoiceResponse) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, g) }

// lnxInvoicePaymentsBody is the body of an LNX Invoice Payments request, whose response depends on its action
type lnxInvoicePaymentsBody struct {
	Action string                  `json:"action"`
	Query  lnxInvoicePaymentsQuery `json:"query"`
}

// lnxInvoicePaymentsQuery is the query of an LNX Invoice Payments request; the list actions take the offset and the
// single record actions the payment hash
type lnxInvoicePaymentsQuery struct {
	Offset      uint64 `json:"offset,omitempty"`
	PaymentHash string `json:"txid,omitempty"`
}

// lnxInvoiceListResponse is the invoices or payments the LNX Invoice Payments list actions return
type lnxInvoiceListResponse struct {
	Invoices []LNXInvoice `json:"invoices"`
	Payments []LNXInvoice `json:"payments"`
}

// LNXInvoice is a Lightning Network invoice of the account, or a payment of one, which Bitfinex describes alike
type LNXInvoice struct {
	PaymentHash string     `json:"payment_hash"`
	CreatedAt   types.Time `json:"created_at"`
	// ExpiresAt is unset for an invoice without an expiry
	ExpiresAt          types.Time   `json:"invoice_expiry"`
	Invoice            string       `json:"invoice"`
	RecipientPublicKey string       `json:"recipient_pub_key"`
	Memo               string       `json:"memo"`
	AmountSatoshis     float64      `json:"amount"`
	AmountBTC          types.Number `json:"bfx_amount"`
	Status             string       `json:"state"`
}

// WithdrawalRequest holds a withdrawal from one of the account's wallets
type WithdrawalRequest struct {
	// Wallet is WalletTypeExchange, WalletTypeMargin or WalletTypeFunding
	Wallet string
	// Method is the withdrawal method, such as bitcoin or tetheruse, which the pub:map:tx:method configuration maps to
	// its currencies; TransactionMethodLightning pays Invoice rather than withdrawing Amount to Address
	Method  string
	Amount  float64
	Address string
	// Invoice is the Lightning Network invoice a TransactionMethodLightning withdrawal pays
	Invoice string
	// PaymentID is the destination's tag, memo or payment ID
	PaymentID string
	// DeductFee takes the withdrawal fee from Amount rather than from the wallet
	DeductFee bool
	// Note is a personal note on the withdrawal
	Note string
	// TravelRule sends the travel rule details Bitfinex requires of withdrawals worth $1,000 or more
	TravelRule *TravelRuleDetails
}

// TravelRuleDetails identify a withdrawal's beneficiary, by the account's own verification data, a person's names or
// a company's name, and the virtual asset service provider it goes to
type TravelRuleDetails struct {
	// VASPDID is the provider's decentralised identifier, from the virtual asset service providers endpoint; leave it
	// and VASPName unset for self custody
	VASPDID string
	// VASPName names a provider that has no decentralised identifier
	VASPName string
	// BeneficiarySelf names the account itself as the beneficiary, from its verification data
	BeneficiarySelf bool
	// FirstName and LastName name a person as the beneficiary, and need each other
	FirstName string
	LastName  string
	// CorporateName names a company as the beneficiary
	CorporateName string
}

// withdrawalBody is the body of a Withdrawal request
type withdrawalBody struct {
	Wallet                   string `json:"wallet"`
	Method                   string `json:"method"`
	Amount                   string `json:"amount,omitempty"`
	Address                  string `json:"address,omitempty"`
	Invoice                  string `json:"invoice,omitempty"`
	PaymentID                string `json:"payment_id,omitempty"`
	DeductFee                uint64 `json:"fee_deduct,omitempty"`
	Note                     string `json:"note,omitempty"`
	TravelRuleTermsOfService bool   `json:"travel_rule_tos,omitempty"`
	VASPDID                  string `json:"vasp_did,omitempty"`
	VASPName                 string `json:"vasp_name,omitempty"`
	BeneficiarySelf          bool   `json:"beneficiary_self,omitempty"`
	DestinationFirstName     string `json:"dest_firstname,omitempty"`
	DestinationLastName      string `json:"dest_lastname,omitempty"`
	DestinationCorporateName string `json:"dest_corp_name,omitempty"`
}

// Withdrawal is a requested withdrawal
type Withdrawal struct {
	// ID is zero when the withdrawal failed
	ID        uint64  `tuple:"0"`
	Method    string  `tuple:"2"`
	PaymentID string  `tuple:"3"`
	Wallet    string  `tuple:"4"`
	Amount    float64 `tuple:"5"`
	Fee       float64 `tuple:"8"`
}

// UnmarshalJSON decodes a withdrawal array
func (w *Withdrawal) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, w) }

// MovementsRequest holds the currency and filters of the deposits and withdrawals GetMovements returns
type MovementsRequest struct {
	// Currency limits the movements to a currency, such as BTC; unset returns recent movements in every currency
	Currency currency.Code
	// Start and End bound when the movements were last updated
	Start time.Time
	End   time.Time
	// Limit is the most movements returned, at most 2500
	Limit uint64
	// IDs limits the movements to these deposit and withdrawal IDs
	IDs []uint64
	// Address limits the movements to deposits to this address
	Address string
}

// movementsBody is the body of a Movements request
type movementsBody struct {
	historyBody
	IDs     []uint64 `json:"id,omitempty"`
	Address string   `json:"address,omitempty"`
}

// Movement is a deposit or withdrawal
type Movement struct {
	ID       uint64        `tuple:"0"`
	Currency currency.Code `tuple:"1"`
	// Method is the method the movement went over, such as BITCOIN or TETHERUSE, or WIRE for fiat; the docs call it
	// the currency's extended name here, and its method where they detail a movement
	Method    string     `tuple:"2"`
	StartedAt types.Time `tuple:"5"`
	UpdatedAt types.Time `tuple:"6"`
	// Status is such as COMPLETED or CANCELED
	Status string `tuple:"9"`
	// Amount is positive for a deposit and negative for a withdrawal
	Amount types.Number `tuple:"12"`
	// Fees are negative, as charged
	Fees               types.Number `tuple:"13"`
	DestinationAddress string       `tuple:"16"`
	// PaymentID is the tag, memo or payment ID, if any
	PaymentID     string `tuple:"17"`
	TransactionID string `tuple:"20"`
	// Note is a withdrawal's personal note
	Note string `tuple:"21"`
}

// UnmarshalJSON decodes a movement array
func (m *Movement) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, m) }

// movementInfoBody is the body of a Movement Info request
type movementInfoBody struct {
	ID uint64 `json:"id"`
}

// MovementInfoResponse details a deposit or withdrawal
type MovementInfoResponse struct {
	// Movement's PaymentID is the memo or tag
	Movement Movement `tuple:"0,inline"`
	// Remark is a remark on the movement, such as on bank details
	Remark string `tuple:"4"`
	// BankFees are a wire's bank fees
	BankFees     types.Number `tuple:"24"`
	BankRouterID uint64       `tuple:"25"`
	// The external bank movement fields are the external provider's
	ExternalBankMovementID          string `tuple:"28"`
	ExternalBankMovementStatus      string `tuple:"29"`
	ExternalBankMovementDescription string `tuple:"30"`
	// ExternalBankAccountInfo is the external provider's information on the account's bank account
	ExternalBankAccountInfo json.RawMessage `tuple:"31"`
}

// UnmarshalJSON decodes a movement info array
func (m *MovementInfoResponse) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, m) }

// alertTypePrice is the only alert type, of an alert triggered by a trading pair's price
const alertTypePrice = "price"

// alertListBody is the body of an Alert List request
type alertListBody struct {
	Type string `json:"type"`
}

// Alert is a price alert
type Alert struct {
	// Key identifies the alert as type:symbol:price, such as price:tBTCUSD:600
	Key string `tuple:"0"`
	// Type is price
	Type   string  `tuple:"1"`
	Symbol string  `tuple:"2"`
	Price  float64 `tuple:"3"`
	// Countdown starts at the alert's count and falls each time the alert triggers; Bitfinex removes the alert at 0
	Countdown uint64 `tuple:"4"`
}

// UnmarshalJSON decodes an alert array
func (a *Alert) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, a) }

// SetAlertRequest holds a price alert on a trading pair
type SetAlertRequest struct {
	// Symbol is a trading pair, such as tBTCUSD
	Symbol string
	Price  float64
	// Count is how many times the alert triggers before Bitfinex removes it; zero leaves the default of 100
	Count uint64
}

// setAlertBody is the body of an Alert Set request
type setAlertBody struct {
	Type   string `json:"type"`
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
	Count  uint64 `json:"count,omitempty"`
}

// DeleteAlertRequest holds the trading pair and price of the alert DeleteAlert deletes
type DeleteAlertRequest struct {
	// Symbol is a trading pair, such as tBTCUSD
	Symbol string
	Price  float64
}

// DeleteAlertResponse holds whether an alert was deleted
type DeleteAlertResponse struct {
	Deleted bool `tuple:"0"`
}

// UnmarshalJSON decodes an alert delete array
func (d *DeleteAlertResponse) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, d) }

// BalanceAvailableRequest describes the order or funding offer GetBalanceAvailable finds the largest amount for
type BalanceAvailableRequest struct {
	// Symbol is a trading pair, such as tBTCUSD, or a funding currency, such as fUSD
	Symbol string
	// Type is BalanceAvailableTypeExchange, BalanceAvailableTypeMargin, BalanceAvailableTypeDerivative or
	// BalanceAvailableTypeFunding
	Type string
	// Direction is 1 to buy or -1 to sell, which the exchange and margin types need
	Direction int64
	// Rate is the order's price, which the exchange and margin types need
	Rate float64
	// Leverage is the leverage of a derivative order
	Leverage float64
}

// balanceAvailableBody is the body of a Balance Available for Orders/Offers request
type balanceAvailableBody struct {
	Symbol    string `json:"symbol"`
	Direction int64  `json:"dir,omitempty"`
	Rate      string `json:"rate,omitempty"`
	Type      string `json:"type"`
	Leverage  string `json:"lev,omitempty"`
}

// BalanceAvailableResponse holds the largest amount an order or funding offer can be placed with
type BalanceAvailableResponse struct {
	Amount float64 `tuple:"0"`
}

// UnmarshalJSON decodes a balance available array
func (b *BalanceAvailableResponse) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, b) }

// userSettingsWriteBody is the body of a User Settings Write request
type userSettingsWriteBody struct {
	Settings map[string]json.RawMessage `json:"settings"`
}

// userSettingKeysBody is the body of a User Settings Read or Delete request
type userSettingKeysBody struct {
	Keys []string `json:"keys"`
}

// UserSettingsCount holds how many settings a write created or changed
type UserSettingsCount struct {
	Count uint64 `tuple:"0"`
}

// UnmarshalJSON decodes a settings count array
func (u *UserSettingsCount) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, u) }

// userSetting is a setting's key and value, as User Settings Read returns them
type userSetting struct {
	Key   string          `tuple:"0"`
	Value json.RawMessage `tuple:"1"`
}

// UnmarshalJSON decodes a setting array
func (u *userSetting) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, u) }
