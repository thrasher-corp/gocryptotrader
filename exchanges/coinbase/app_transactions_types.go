package coinbase

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Coinbase App transaction types; Coinbase adds more over time, and AppTransactionTypeTx is the uncategorised default
const (
	AppTransactionTypeAdvancedTradeFill        = "advanced_trade_fill"
	AppTransactionTypeBuy                      = "buy"
	AppTransactionTypeClawback                 = "clawback"
	AppTransactionTypeDerivativesSettlement    = "derivatives_settlement"
	AppTransactionTypeEarnPayout               = "earn_payout"
	AppTransactionTypeFiatDeposit              = "fiat_deposit"
	AppTransactionTypeFiatWithdrawal           = "fiat_withdrawal"
	AppTransactionTypeIncentivesRewardsPayout  = "incentives_rewards_payout"
	AppTransactionTypeIncentivesSharedClawback = "incentives_shared_clawback"
	AppTransactionTypeINTXDeposit              = "intx_deposit"
	AppTransactionTypeINTXWithdrawal           = "intx_withdrawal"
	AppTransactionTypeReceive                  = "receive"
	AppTransactionTypeRequest                  = "request"
	AppTransactionTypeRetailSimpleDust         = "retail_simple_dust"
	AppTransactionTypeSell                     = "sell"
	AppTransactionTypeSend                     = "send"
	AppTransactionTypeStakingTransfer          = "staking_transfer"
	AppTransactionTypeSubscriptionRebate       = "subscription_rebate"
	AppTransactionTypeSubscription             = "subscription"
	AppTransactionTypeTrade                    = "trade"
	AppTransactionTypeTransfer                 = "transfer"
	AppTransactionTypeTx                       = "tx"
	AppTransactionTypeUnstakingTransfer        = "unstaking_transfer"
	AppTransactionTypeUnsupportedAssetRecovery = "unsupported_asset_recovery"
	AppTransactionTypeUnwrapAsset              = "unwrap_asset"
	AppTransactionTypeVaultWithdrawal          = "vault_withdrawal"
	AppTransactionTypeWrapAsset                = "wrap_asset"
	// AppTransactionTypeFCMFuturesUSDCSell converts USDC to USD for a futures trade's expected margin
	AppTransactionTypeFCMFuturesUSDCSell = "fcm_futures_usdc_sell"
	// AppTransactionTypeFCMFuturesUSDCSellAdditionalEncumbermentRollup converts USDC to USD for open futures positions'
	// further margin or losses
	AppTransactionTypeFCMFuturesUSDCSellAdditionalEncumbermentRollup = "fcm_futures_usdc_sell_additional_encumberment_rollup"
)

// Coinbase App transaction statuses, which vary by transaction type
const (
	AppTransactionStatusCancelled = "canceled"
	AppTransactionStatusCompleted = "completed"
	AppTransactionStatusExpired   = "expired"
	AppTransactionStatusFailed    = "failed"
	AppTransactionStatusPending   = "pending"
	// AppTransactionStatusWaitingForClearing and AppTransactionStatusWaitingForSignature are a vault withdrawal's
	// statuses while it awaits clearing or approval
	AppTransactionStatusWaitingForClearing  = "waiting_for_clearing"
	AppTransactionStatusWaitingForSignature = "waiting_for_signature"
)

// Coinbase App transaction network statuses
const (
	AppTransactionNetworkStatusOffBlockchain = "off_blockchain"
	AppTransactionNetworkStatusConfirmed     = "confirmed"
	AppTransactionNetworkStatusPending       = "pending"
	AppTransactionNetworkStatusUnconfirmed   = "unconfirmed"
)

// Travel rule beneficiary wallet types
const (
	AppTravelRuleWalletTypeSelfHosted = "WALLET_TYPE_SELF_HOSTED"
	AppTravelRuleWalletTypeExchange   = "WALLET_TYPE_EXCHANGE"
)

// Travel rule IsSelf values, which say whether the user owns the wallet sent to
const (
	AppTravelRuleIsSelfFalse = "IS_SELF_FALSE"
	AppTravelRuleIsSelfTrue  = "IS_SELF_TRUE"
)

// AppSendMoneyRequest sends funds from an account to a blockchain address or an email address
type AppSendMoneyRequest struct {
	AccountID string
	// To is the blockchain address or email address of the recipient; sends to phone numbers are disabled
	To       string
	Amount   float64
	Currency currency.Code
	// Description is included in the email to the recipient
	Description string
	// SkipNotifications sends no notification emails, for small amounts such as tips
	SkipNotifications bool
	// IdempotencyKey is a lower case UUIDv4 of at most 100 characters that makes the send idempotent: a send repeating
	// a previous send's key returns the previous send instead of sending again. Coinbase recommends it
	IdempotencyKey string
	// DestinationTag is the destination tag or memo the currencies that take one name the beneficiary by
	DestinationTag string
	// Network is the blockchain a send to an address goes on, such as ethereum or polygon; the currency's default
	// network when unset
	Network string
	// TravelRuleData holds the beneficiary details the user's region requires for a send
	TravelRuleData *AppTravelRuleData
}

// AppTravelRuleData is the beneficiary information the Travel Rule requires for a send. Which fields are required
// depends on the Coinbase entity the user signed up with: none for the US; BeneficiaryWalletType for Canada, with the
// beneficiary's name and address for exchange sends of 999 CAD or more; IsSelf, BeneficiaryName and
// BeneficiaryWalletType for Bermuda; and those with the beneficiary's country, BeneficiaryAddress's Country, for Great
// Britain, Singapore, Hong Kong, the Philippines and the EU, which also require BeneficiaryFinancialInstitution for
// exchange sends. Singapore, Hong Kong and the Philippines require a send to a self hosted wallet, and the EU one of
// more than 1000 EUR, to pass a Satoshi test in the Coinbase app first
type AppTravelRuleData struct {
	BeneficiaryWalletType string                `json:"beneficiary_wallet_type,omitempty"`
	IsSelf                string                `json:"is_self,omitempty"`
	BeneficiaryName       string                `json:"beneficiary_name,omitempty"`
	BeneficiaryAddress    *AppTravelRuleAddress `json:"beneficiary_address,omitempty"`
	// BeneficiaryFinancialInstitution is the ID of the financial institution or virtual asset service provider the
	// beneficiary's wallet is with, from Coinbase's travel rule VASP list
	BeneficiaryFinancialInstitution string `json:"beneficiary_financial_institution,omitempty"`
	// TransferPurpose is what the send is for, such as rent
	TransferPurpose string `json:"transfer_purpose,omitempty"`
}

// AppTravelRuleAddress is a travel rule beneficiary's postal address
type AppTravelRuleAddress struct {
	Address1 string `json:"address1,omitempty"`
	Address2 string `json:"address2,omitempty"`
	Address3 string `json:"address3,omitempty"`
	City     string `json:"city,omitempty"`
	State    string `json:"state,omitempty"`
	// Country is an ISO 3166-1 alpha-2 country code
	Country    string `json:"country,omitempty"`
	PostalCode string `json:"postal_code,omitempty"`
}

// AppSendMoneyWireRequest is Send Money's request body
type AppSendMoneyWireRequest struct {
	Type   string `json:"type"`
	To     string `json:"to"`
	Amount string `json:"amount"`
	// Currency stays a string, the code as the App API takes it
	Currency          string             `json:"currency"`
	Description       string             `json:"description,omitempty"`
	SkipNotifications bool               `json:"skip_notifications,omitempty"`
	IdempotencyKey    string             `json:"idem,omitempty"`
	DestinationTag    string             `json:"destination_tag,omitempty"`
	Network           string             `json:"network,omitempty"`
	TravelRuleData    *AppTravelRuleData `json:"travel_rule_data,omitempty"`
}

// AppListTransactionsRequest selects a page of an account's transactions
type AppListTransactionsRequest struct {
	AccountID string
	// Page selects the page; nil selects the first
	Page *AppPageRequest
}

// AppGetTransactionRequest selects one of an account's transactions
type AppGetTransactionRequest struct {
	AccountID     string
	TransactionID string
}

// AppTransactionResponse is an event on an account. Amount and NativeAmount are negative for a debit, such as a send or
// an advanced trade sell's fill; a transaction with a counterparty sets To or From. Coinbase recommends describing a
// transaction from Details, as types and statuses change over time
type AppTransactionResponse struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	Status       string    `json:"status"`
	Amount       AppMoney  `json:"amount"`
	NativeAmount AppMoney  `json:"native_amount"`
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"created_at"`
	// UpdatedAt and Details are documented as no longer sent by List Transactions and Show Transaction since 7
	// February 2024, although their examples hold them; Send Money's response holds UpdatedAt
	UpdatedAt    time.Time `json:"updated_at"`
	Resource     string    `json:"resource"`
	ResourcePath string    `json:"resource_path"`
	// AdvancedTradeFill is set for an advanced_trade_fill transaction
	AdvancedTradeFill *AppAdvancedTradeFill  `json:"advanced_trade_fill"`
	Details           *AppTransactionDetails `json:"details"`
	// Network describes the networks a crypto transaction moved on, with its on-chain hash, for certain types
	Network *AppTransactionNetwork `json:"network"`
	// To is the receiving party of a debit, and From the originating party of a credit, for certain types
	To   *AppTransactionParty `json:"to"`
	From *AppTransactionParty `json:"from"`
	// Cancellable and IdempotencyKey are sent for a send only
	Cancellable    bool   `json:"cancelable"`
	IdempotencyKey string `json:"idem"`
	// Buy, Sell and Trade describe a buy, sell or trade transaction; Coinbase documents no fields for them, so they
	// are kept as sent
	Buy   json.RawMessage `json:"buy"`
	Sell  json.RawMessage `json:"sell"`
	Trade json.RawMessage `json:"trade"`
}

// AppAdvancedTradeFill is the advanced trade fill an advanced_trade_fill transaction posts
type AppAdvancedTradeFill struct {
	FillPrice types.Number `json:"fill_price"`
	ProductID string       `json:"product_id"`
	OrderID   string       `json:"order_id"`
	// Commission is the fill's commission, in the quote currency
	Commission types.Number `json:"commission"`
	OrderSide  string       `json:"order_side"`
}

// AppTransactionDetails describe a transaction for display, such as "Received Bitcoin"
type AppTransactionDetails struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	// Header is the amount received in the transaction's and the user's native currency, such as "Received 0.005378
	// BTC ($49.92)"
	Header string `json:"header"`
	// Health is the transaction's health, such as Positive
	Health          string `json:"health"`
	SubsidebarLabel string `json:"subsidebar_label"`
}

// AppTransactionNetwork describes the network a crypto transaction moved on; Hash, TransactionFee and NetworkName are
// sent for a send only
type AppTransactionNetwork struct {
	Status            string    `json:"status"`
	Hash              string    `json:"hash"`
	TransactionFee    *AppMoney `json:"transaction_fee"`
	TransactionAmount *AppMoney `json:"transaction_amount"`
	NetworkName       string    `json:"network_name"`
	// Name is the network's name, which the docs' examples send in place of NetworkName
	Name string `json:"name"`
}

// AppTransactionParty is a transaction's counterparty: a Coinbase account or user, named by its ID and resource path,
// a blockchain address, an email address, or another resource such as a blockchain network
type AppTransactionParty struct {
	ID string `json:"id"`
	// Resource names the party's kind, such as account, user, email or bitcoin_address
	Resource     string `json:"resource"`
	ResourcePath string `json:"resource_path"`
	Address      string `json:"address"`
	Email        string `json:"email"`
}
