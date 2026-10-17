package kraken

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	errFundingMethodIDEmpty        = errors.New("funding method ID is empty")
	errFundingAddressIDEmpty       = errors.New("funding address ID is empty")
	errFundingAddressEmpty         = errors.New("funding address is empty")
	errFundingAddressNameEmpty     = errors.New("funding address name is empty")
	errFundingAddressUpdateEmpty   = errors.New("funding address name or description is required")
	errFundingScopeEmpty           = errors.New("funding scope is empty")
	errFundingScopeAmbiguous       = errors.New("funding scope sets more than one ID")
	errFundingNetworkGroupScope    = errors.New("funding network group scope is not supported")
	errFundingAssetClassEmpty      = errors.New("funding asset class is empty")
	errInvalidFundingAssetClass    = errors.New("invalid funding asset class")
	errInvalidFundingDirection     = errors.New("invalid funding direction")
	errInvalidFundingLimit         = errors.New("invalid funding page limit")
	errFundingCursorWithFilters    = errors.New("funding cursor cannot be combined with filters")
	errFundingStatusFilterConflict = errors.New("funding status list and range cannot be combined")
	errFundingFeeConflict          = errors.New("funding fee token and maximum fee cannot be combined")
)

// FundingAsset identifies an asset by class and name
type FundingAsset struct {
	// Class is currency or tokenized_asset
	Class string        `json:"class"`
	Name  currency.Code `json:"name"`
}

// FundingAmount is an amount of an asset
type FundingAmount struct {
	Asset  FundingAsset `json:"asset"`
	Amount types.Number `json:"amount"`
}

// FundingScope selects a funding method, a network or a network group, and holds the scope a withdrawal address is
// saved against. At most one ID is set
type FundingScope struct {
	MethodID       string `json:"method_id,omitempty"`
	NetworkID      string `json:"network_id,omitempty"`
	NetworkGroupID string `json:"network_group_id,omitempty"`
}

// FundingFeesRequest holds the parameters of Calculate Funding Fees
type FundingFeesRequest struct {
	MethodID string
	Amount   float64
	// FeeIncluded makes Amount the total debited, fee included
	FeeIncluded bool
	// WithdrawalFeeToken recalculates with the fee rate a previous withdrawal quote pinned
	WithdrawalFeeToken string
	// RebaseMultiplier is rebased or base, choosing the units of a tokenised asset's amounts
	RebaseMultiplier string
	// AccountID selects a wallet account other than the default
	AccountID string
}

// FundingFeesResponse holds a funding fee quote
type FundingFeesResponse struct {
	Fee         FundingAmount     `json:"fee"`
	GrossAmount FundingAmount     `json:"gross_amount"`
	NetAmount   FundingAmount     `json:"net_amount"`
	FeeDetails  FundingFeeDetails `json:"fee_details"`
	// WithdrawalFeeToken is only quoted for withdrawals. It pins the quoted fee rate for any withdrawal made within 5
	// minutes of the quote
	WithdrawalFeeToken string `json:"withdrawal_fee_token"`
}

// FundingFeeDetails holds the components of a calculated funding fee
type FundingFeeDetails struct {
	BaseFee       FundingAmount `json:"base_fee"`
	FeePercentage types.Number  `json:"fee_percentage"`
}

// FundingDepositAddressRequest holds the parameters of Claim Funding Deposit Address
type FundingDepositAddressRequest struct {
	MethodID string
	// AccountID selects a wallet account other than the default
	AccountID string
}

// FundingDepositAddressResponse holds a claimed deposit address
type FundingDepositAddressResponse struct {
	AddressDetails FundingDepositAddressDetails `json:"address_details"`
}

// FundingDepositAddressDetails holds a deposit address. Crypto is set for a crypto address and Fiat for bank details
type FundingDepositAddressDetails struct {
	Crypto *FundingDepositCryptoAddress `json:"crypto"`
	Fiat   *FundingDepositFiatAddress   `json:"fiat"`
}

// FundingDepositCryptoAddress is a crypto deposit address. A deposit sent without the tag or memo the address carries
// may not be credited
type FundingDepositCryptoAddress struct {
	Address string `json:"address"`
	Tag     string `json:"tag"`
	Memo    string `json:"memo"`
}

// FundingDepositFiatAddress holds the bank details to deposit fiat to
type FundingDepositFiatAddress struct {
	AccountNumber string `json:"account"`
	AccountType   string `json:"account_type"`
	// Address is a bank or wallet address
	Address     string `json:"address"`
	BankName    string `json:"bank"`
	BankCode    string `json:"bank_code"`
	BIC         string `json:"bic"`
	Branch      string `json:"branch"`
	BankAddress string `json:"bank_address"`
	// BSB is an Australian Bank State Branch code
	BSB           string `json:"bsb"`
	IBAN          string `json:"iban"`
	Memo          string `json:"memo"`
	NameOnAccount string `json:"name_on_account"`
	RoutingNumber string `json:"routing"`
	SortCode      string `json:"sort"`
	SWIFTCode     string `json:"swift"`
	Tag           string `json:"tag"`
	TransitNumber string `json:"transit"`
}

// FundingAddressRequest holds the parameters of Create Funding Address, which saves crypto withdrawal addresses only
type FundingAddressRequest struct {
	// Scope sets which withdrawals may use the address: those of one funding method, of any method on a network, or of
	// any method on a group of networks
	Scope   FundingScope
	Address string
	// Tag and Memo are a destination tag or memo for chains taking one; Kraken stores either under the name the method
	// or network uses
	Tag  string
	Memo string
	// Name must be unique and contain a character other than whitespace
	Name        string
	Description string
	// AccountID selects a wallet account other than the default
	AccountID string
}

// FundingAddressResponse holds a saved withdrawal address's ID
type FundingAddressResponse struct {
	AddressID string `json:"address_id"`
	Verified  bool   `json:"verified"`
}

// FundingWithdrawalRequest holds the parameters of Create Funding Withdrawal
type FundingWithdrawalRequest struct {
	// Scope selects the withdrawal method by MethodID, or by NetworkID when the asset has a single withdrawal method on
	// that network
	Scope FundingScope
	// AddressID is a saved withdrawal address that the method may use, whatever scope the address is saved against
	AddressID string
	Asset     FundingAsset
	Amount    float64
	// FeeIncluded makes Amount the total debited, fee included. A withdrawal using a fee token should match the quote's
	// setting
	FeeIncluded bool
	// FeeToken is a withdrawal fee token from CalculateFundingFees, pinning its quoted fee rate. Without one the current
	// fee is charged
	FeeToken string
	// MaximumFee caps the current fee, in the withdrawal's asset, failing the withdrawal if the fee exceeds it. It cannot
	// be set with FeeToken
	MaximumFee float64
	// RebaseMultiplier is rebased or base, choosing the units of a tokenised asset's amount and fee
	RebaseMultiplier string
	// ExpectedAddress fails the withdrawal unless it matches the crypto address saved for AddressID
	ExpectedAddress string
	// AccountID selects a wallet account other than the default
	AccountID string
}

// FundingWithdrawalResponse holds a created withdrawal
type FundingWithdrawalResponse struct {
	WithdrawalID string                  `json:"withdrawal_id"`
	NetAmount    FundingWithdrawalAmount `json:"net_amount"`
	GrossAmount  FundingWithdrawalAmount `json:"gross_amount"`
	Fee          FundingWithdrawalAmount `json:"fee"`
	// ApprovalRequestID is set when the withdrawal waits for an approval, and is cancelled if the approval is refused
	ApprovalRequestID string `json:"approval_request_id"`
}

// FundingWithdrawalAmount is a withdrawal amount and the units a tokenised asset's amount is in
type FundingWithdrawalAmount struct {
	AssetAmount FundingAmount `json:"asset_amount"`
	// RebaseMultiplier is rebased or base
	RebaseMultiplier string `json:"rebase_multiplier"`
}

// FundingAddressDeletionRequest holds the parameters of Delete Funding Address
type FundingAddressDeletionRequest struct {
	AddressID string
	// AccountID selects a wallet account other than the default
	AccountID string
}

// FundingAddressDeletionResponse holds whether a saved withdrawal address was deleted
type FundingAddressDeletionResponse struct {
	Deleted bool `json:"result"`
}

// FundingAddressesRequest holds the parameters of List Funding Addresses
type FundingAddressesRequest struct {
	// Scope filters the addresses to those a funding method, network or network group may use, including those saved
	// against a broader scope: a method lists its network's and network group's addresses, a network its group's
	Scope FundingScope
	// Cursor requests the page after the one that returned it, keeping that request's scope, so Scope must be empty
	Cursor string
	// Limit is the page size, up to 500
	Limit uint64
	// AccountID selects a wallet account other than the default
	AccountID string
}

// FundingAddressesResponse holds a page of saved withdrawal addresses
type FundingAddressesResponse struct {
	Addresses []FundingAddress `json:"addresses"`
	// NextCursor requests the next page, and is empty on the last
	NextCursor string `json:"next_cursor"`
}

// FundingAddress is a saved withdrawal address
type FundingAddress struct {
	AddressID      string                          `json:"address_id"`
	Scope          FundingScope                    `json:"scope"`
	AddressDetails FundingWithdrawalAddressDetails `json:"address_details"`
	Name           string                          `json:"name"`
	Description    string                          `json:"description"`
	// Modifier is third_party for an address belonging to a third party
	Modifier string `json:"modifier"`
	Verified bool   `json:"verified"`
}

// FundingWithdrawalAddressDetails holds a withdrawal address. Crypto is set for a crypto address and Fiat for bank or
// card details
type FundingWithdrawalAddressDetails struct {
	Crypto *FundingWithdrawalCryptoAddress `json:"crypto"`
	Fiat   *FundingWithdrawalFiatAddress   `json:"fiat"`
}

// FundingWithdrawalCryptoAddress is a crypto withdrawal address
type FundingWithdrawalCryptoAddress struct {
	Address     string             `json:"address"`
	Tag         string             `json:"tag"`
	Beneficiary FundingBeneficiary `json:"beneficiary"`
	Memo        string             `json:"memo"`
}

// FundingWithdrawalFiatAddress holds the bank or card details to withdraw fiat to
type FundingWithdrawalFiatAddress struct {
	AccountNumber string `json:"account"`
	AccountType   string `json:"account_type"`
	// Address is a bank or wallet address
	Address     string `json:"address"`
	BankName    string `json:"bank"`
	BankCode    string `json:"bank_code"`
	BIC         string `json:"bic"`
	Branch      string `json:"branch"`
	BranchCode  string `json:"branch_code"`
	BankAddress string `json:"bank_address"`
	// BSB is an Australian Bank State Branch code
	BSB                       string             `json:"bsb"`
	Beneficiary               FundingBeneficiary `json:"beneficiary"`
	CardNumber                string             `json:"card_number"`
	CardNumberLastFour        string             `json:"card_number_last"`
	CardType                  string             `json:"card_type"`
	CardVerificationCode      string             `json:"cvc"`
	ExpiryMonth               string             `json:"expire_month"`
	ExpiryYear                string             `json:"expire_year"`
	IBAN                      string             `json:"iban"`
	MaskedIBAN                string             `json:"masked_iban"`
	IntermediaryBankAddress   string             `json:"intermediary_address"`
	IntermediaryBankName      string             `json:"intermediary_bank"`
	IntermediaryBranch        string             `json:"intermediary_branch"`
	IntermediaryRoutingNumber string             `json:"intermediary_routing"`
	// IntermediarySWIFTCode is the intermediary bank's SWIFT code or BIC
	IntermediarySWIFTCode string `json:"intermediary_swift"`
	Memo                  string `json:"memo"`
	MerchantReference     string `json:"merchant_reference"`
	NameOnAccount         string `json:"name_on_account"`
	Notes                 string `json:"notes"`
	Password              string `json:"password"`
	RoutingNumber         string `json:"routing"`
	SortCode              string `json:"sort"`
	SWIFTCode             string `json:"swift"`
	Tag                   string `json:"tag"`
	ThirdPartyEmail       string `json:"third_party_email"`
	// TransactionID is the payment provider's transaction ID
	TransactionID string `json:"transaction_id"`
	TransitNumber string `json:"transit"`
	Username      string `json:"username"`
	// Signature proves ownership of the address
	Signature string `json:"signature"`
}

// FundingBeneficiary holds the beneficiary of a withdrawal address
type FundingBeneficiary struct {
	// Recipient is sender when the beneficiary is the account holder, or other
	Recipient string `json:"recipient"`
	// Type is individual or business
	Type                 string `json:"typ"`
	FirstName            string `json:"name"`
	LastName             string `json:"last_name"`
	Country              string `json:"country"`
	StateOrProvince      string `json:"state_province"`
	AddressLine1         string `json:"address_1"`
	AddressLine2         string `json:"address_2"`
	City                 string `json:"city"`
	PostalCode           string `json:"postal_code"`
	CounterpartyVASPName string `json:"counterparty_vasp_name"`
}

// FundingAssetsRequest holds the parameters of List Funding Assets
type FundingAssetsRequest struct {
	// Direction is deposit or withdraw
	Direction string
	// AssetClass filters the assets to a class: currency or tokenized_asset
	AssetClass string
	// AccountID selects a wallet account other than the default
	AccountID string
}

// FundingAssetsResponse holds the assets that can be funded in a direction, by asset class
type FundingAssetsResponse struct {
	Currencies      []FundingAvailableAsset `json:"currency"`
	TokenisedAssets []FundingAvailableAsset `json:"tokenized_asset"`
}

// FundingAvailableAsset is an asset that can be funded
type FundingAvailableAsset struct {
	Name currency.Code `json:"name"`
}

// FundingClaimedAddressesRequest holds the parameters of List Funding Claimed Addresses (v2)
type FundingClaimedAddressesRequest struct {
	// Scope filters the addresses to those of a funding method or of a network's methods; a network group cannot be set.
	// A method's filter lists the addresses it shares with another method under the method filtered by
	Scope FundingScope
	// Asset filters the addresses to an asset class, or to an asset when Name is set too
	Asset FundingAsset
	// Cursor requests the page after the one that returned it, keeping that request's filters, so Scope and Asset must
	// be empty
	Cursor string
	// Limit is the page size, up to 500
	Limit uint64
	// AccountID selects a wallet account other than the default
	AccountID string
}

// FundingClaimedAddressesResponse holds a page of claimed deposit addresses
type FundingClaimedAddressesResponse struct {
	Addresses []FundingClaimedAddress `json:"addresses"`
	// NextCursor requests the next page, and is empty on the last
	NextCursor string `json:"next_cursor"`
}

// FundingClaimedAddress is a claimed deposit address
type FundingClaimedAddress struct {
	// MethodID is the method the request's scope filtered by, or, without a method filter, the method the address is
	// stored against
	MethodID string `json:"method_id"`
	// SharesAddressesWithMethodID is the method a shared address is stored against. Kraken's example sends it, though its
	// schema does not list it
	SharesAddressesWithMethodID string                       `json:"shares_addresses_with_method_id"`
	ID                          string                       `json:"id"`
	LastUsed                    time.Time                    `json:"last_used"`
	ExpiryTime                  time.Time                    `json:"expire_time"`
	AddressDetails              FundingDepositAddressDetails `json:"address_details"`
}

// FundingLimitsRequest holds the parameters of List Funding Deposit Limits and List Funding Withdrawal Limits
type FundingLimitsRequest struct {
	Asset FundingAsset
	// PreferredAsset expresses the limits' requested asset amounts in another asset, rather than each funding method's
	// asset. Its Name must be set when its Class is
	PreferredAsset FundingAsset
	// AccountID selects a wallet account other than the default
	AccountID string
}

// FundingDepositLimitsResponse holds an asset's deposit limits for each funding method
type FundingDepositLimitsResponse struct {
	DepositLimits []FundingDepositLimit `json:"deposit_limits"`
}

// FundingDepositLimit holds a funding method's deposit limits
type FundingDepositLimit struct {
	MethodID string `json:"method_id"`
	// MaximumAmount is the most the amount based limits allow depositing, and is nil when no amount based limit applies.
	// Attempt and success limits are not counted
	MaximumAmount *FundingAmount `json:"maximum_amount"`
	Limits        []FundingLimit `json:"limits"`
}

// FundingLimit is a funding limit over a time window
type FundingLimit struct {
	// TimeWindow is the window's length, which Kraken documents without a unit; its example, 86400, reads as a day in
	// seconds
	TimeWindow types.Number       `json:"time_window"`
	Limit      FundingLimitValues `json:"limit"`
}

// FundingLimitValues holds a limit's maximum and what is used and remains of it in the time window
type FundingLimitValues struct {
	// LimitType is attempt or success for a limit counting transactions, or amount or an equiv_amount_ currency, such as
	// equiv_amount_usd, for a limit on amounts
	LimitType string            `json:"limit_type"`
	Remaining FundingLimitValue `json:"remaining"`
	Maximum   FundingLimitValue `json:"maximum"`
	Used      FundingLimitValue `json:"used"`
}

// FundingLimitValue is a part of a funding limit: a count for a limit counting transactions, or an amount in several
// assets for a limit on amounts
type FundingLimitValue struct {
	Count types.Number `json:"-"`
	// PolicyAssetAmount is in the asset the limit's policy is set in
	PolicyAssetAmount FundingAmount `json:"policy_asset_amount"`
	USDAmount         FundingAmount `json:"usd_amount"`
	// RequestedAssetAmount is in the request's preferred asset, or the funding method's asset without one
	RequestedAssetAmount FundingAmount `json:"requested_asset_amount"`
}

// UnmarshalJSON decodes a count, which Kraken sends as a decimal string, or an object of amounts
func (f *FundingLimitValue) UnmarshalJSON(data []byte) error {
	if len(data) != 0 && data[0] == '{' {
		type amounts FundingLimitValue
		return json.Unmarshal(data, (*amounts)(f))
	}
	return json.Unmarshal(data, &f.Count)
}

// FundingDepositsRequest holds the parameters of List Funding Deposits
type FundingDepositsRequest struct {
	// Asset filters the deposits to an asset class, or to an asset when Name is set too
	Asset FundingAsset
	Scope FundingScope
	// Statuses filters the deposits to any of these statuses: initial, pending, settled, success or failure
	Statuses []string
	// StatusFrom and StatusTo filter the deposits to the statuses between them inclusively, in the order Statuses lists
	// them; either may be empty to leave the range open. They cannot be set with Statuses
	StatusFrom string
	StatusTo   string
	StartTime  time.Time
	EndTime    time.Time
	// RebaseMultiplier is rebased or base, choosing the units of a tokenised asset's amounts
	RebaseMultiplier string
	// Cursor requests the page after the one that returned it, keeping that request's filters, so no filter may be set
	Cursor string
	// Limit is the page size, up to 500
	Limit uint64
	// AccountID selects a wallet account other than the default
	AccountID string
}

// FundingDepositsResponse holds a page of deposits, newest first
type FundingDepositsResponse struct {
	Deposits []FundingDeposit `json:"deposits"`
	// NextCursor requests the next page, and is empty on the last
	NextCursor string `json:"next_cursor"`
}

// FundingDeposit is a deposit
type FundingDeposit struct {
	DepositID string `json:"deposit_id"`
	MethodID  string `json:"method_id"`
	NetworkID string `json:"network_id"`
	// Status is initial, pending, settled, success or failure
	Status string `json:"status"`
	// Amount is nil while the amount is unknown
	Amount *FundingAmount `json:"amount"`
	// Fee is nil until the deposit completes
	Fee          *FundingAmount `json:"fee"`
	CreationTime time.Time      `json:"create_time"`
}

// FundingMethodsRequest holds the parameters of List Funding Methods
type FundingMethodsRequest struct {
	// Direction is deposit or withdraw
	Direction string
	// Asset filters the methods to an asset class, or to an asset when Name is set too
	Asset FundingAsset
	// RebaseMultiplier is rebased or base, choosing the units of a tokenised asset's amounts
	RebaseMultiplier string
	// Cursor requests the page after the one that returned it, keeping that request's filters, so Asset and
	// RebaseMultiplier must be empty
	Cursor string
	// Limit is the page size, up to 10000
	Limit uint64
	// AccountID selects a wallet account other than the default
	AccountID string
}

// FundingMethodsResponse holds a page of funding methods. Kraken sends withdrawal methods an empty withdrawal object,
// which documents no fields, so it is left out
type FundingMethodsResponse struct {
	Methods []FundingMethod `json:"methods"`
	// NextCursor requests the next page, and is empty on the last
	NextCursor string `json:"next_cursor"`
}

// FundingMethod is a way to deposit or withdraw an asset on a network
type FundingMethod struct {
	// Deposit is only set for deposit methods
	Deposit    *FundingMethodDeposit `json:"deposit"`
	Asset      FundingAsset          `json:"asset"`
	MethodID   string                `json:"method_id"`
	MethodName string                `json:"method_name"`
	// MinimumAmount and MaximumAmount bound each transaction, and are zero when Kraken sends no bound
	MinimumAmount types.Number         `json:"minimum_amount"`
	MaximumAmount types.Number         `json:"maximum_amount"`
	Fees          FundingMethodFees    `json:"fees"`
	Network       FundingMethodNetwork `json:"network"`
}

// FundingMethodDeposit holds the details only deposit methods have
type FundingMethodDeposit struct {
	// SharesAddressesWithMethodID is the method whose deposit addresses this method reuses, such as ETH on Ethereum for
	// an ERC-20 token; it is empty when the method has its own addresses
	SharesAddressesWithMethodID string `json:"shares_addresses_with_method_id"`
	// AddressSetupFee is charged when an address is first claimed for the method
	AddressSetupFee   FundingAmount            `json:"address_setup_fee"`
	AddressGeneration FundingAddressGeneration `json:"address_generation"`
	// AllowCreditCards is only set for card methods, false allowing debit cards alone in the account's country
	AllowCreditCards *bool `json:"allow_credit_cards"`
	// ExemptedWithdrawalHold is only set for methods imposing a withdrawal hold, true exempting their deposits from it
	ExemptedWithdrawalHold *bool `json:"exempted_withdrawal_hold"`
}

// FundingAddressGeneration holds whether more deposit addresses can be claimed for a method
type FundingAddressGeneration struct {
	// Status is unsupported, unlimited or limited
	Status string `json:"status"`
	// Limit is the number of unused addresses a limited method may hold
	Limit uint64 `json:"limit"`
}

// FundingMethodFees holds the fees a funding method charges on each transaction
type FundingMethodFees struct {
	Base       FundingAmount `json:"base"`
	Percentage types.Number  `json:"percentage"`
	// Included is whether the fee comes out of the transaction's amount
	Included bool          `json:"included"`
	Minimum  FundingAmount `json:"min"`
	Maximum  FundingAmount `json:"max"`
}

// FundingMethodNetwork is the network a funding method uses
type FundingMethodNetwork struct {
	NetworkID          string `json:"network_id"`
	NetworkName        string `json:"network_name"`
	ContractAddress    string `json:"contract_address"`
	OnChainAssetSymbol string `json:"on_chain_asset_symbol"`
}

// FundingNetworksRequest holds the parameters of List Funding Networks
type FundingNetworksRequest struct {
	// AccountID selects a wallet account other than the default
	AccountID string
}

// FundingNetworksResponse holds the networks available for funding and the groups of networks sharing an address
// format
type FundingNetworksResponse struct {
	NetworkGroups []FundingNetworkGroup `json:"network_groups"`
	Networks      []FundingNetwork      `json:"networks"`
}

// FundingNetworkGroup is a group of networks sharing an address format, such as EVM networks
type FundingNetworkGroup struct {
	NetworkGroupID string   `json:"network_group_id"`
	Name           string   `json:"name"`
	NetworkIDs     []string `json:"network_ids"`
}

// FundingNetwork is a network available for funding
type FundingNetwork struct {
	NetworkID string `json:"network_id"`
	Name      string `json:"name"`
}

// FundingWithdrawalLimitsResponse holds the balance available to withdraw and an asset's withdrawal limits for each
// funding method
type FundingWithdrawalLimitsResponse struct {
	// AvailableBalance is the balance left after holds, open positions, margin requirements and other restrictions; a
	// method's limits may lower it further
	AvailableBalance FundingAmount            `json:"available_balance"`
	WithdrawalLimits []FundingWithdrawalLimit `json:"withdrawal_limits"`
}

// FundingWithdrawalLimit holds a funding method's withdrawal limits
type FundingWithdrawalLimit struct {
	MethodID string `json:"method_id"`
	// MaximumAmount is the most the available balance and the amount based limits allow withdrawing. Attempt and success
	// limits are not counted
	MaximumAmount FundingAmount `json:"maximum_amount"`
	// MaximumReason is balance or limits, whichever sets MaximumAmount
	MaximumReason string         `json:"maximum_reason"`
	Limits        []FundingLimit `json:"limits"`
}

// FundingWithdrawalsRequest holds the parameters of List Funding Withdrawals
type FundingWithdrawalsRequest struct {
	// Asset filters the withdrawals to an asset class, or to an asset when Name is set too
	Asset FundingAsset
	Scope FundingScope
	// Status filters the withdrawals to a status: pending, success or failed
	Status    string
	StartTime time.Time
	EndTime   time.Time
	// RebaseMultiplier is rebased or base, choosing the units of a tokenised asset's amounts
	RebaseMultiplier string
	// Cursor requests the page after the one that returned it, keeping that request's filters, so no filter may be set
	Cursor string
	// Limit is the page size, up to 500
	Limit uint64
	// AccountID selects a wallet account other than the default
	AccountID string
}

// FundingWithdrawalsResponse holds a page of withdrawals, newest first
type FundingWithdrawalsResponse struct {
	Withdrawals []FundingWithdrawal `json:"withdrawals"`
	// NextCursor requests the next page, and is empty on the last
	NextCursor string `json:"next_cursor"`
}

// FundingWithdrawal is a withdrawal
type FundingWithdrawal struct {
	WithdrawalID string        `json:"withdrawal_id"`
	Amount       FundingAmount `json:"amount"`
	Fee          FundingAmount `json:"fee"`
	MethodID     string        `json:"method_id"`
	// Status is pending, success or failed
	Status       string    `json:"status"`
	CreationTime time.Time `json:"create_time"`
	// AddressID is empty for a withdrawal made without a saved address
	AddressID          string `json:"address_id"`
	OnChainTransaction string `json:"onchain_transaction"`
	// OutputIndex is the index of a Bitcoin transaction's output
	OutputIndex types.Number `json:"utxo_vout"`
}

// FundingAddressUpdateRequest holds the parameters of Update Funding Address. Name, Description or both must be set
type FundingAddressUpdateRequest struct {
	AddressID string
	// Name must be unique and contain a character other than whitespace
	Name        string
	Description string
	// AccountID selects a wallet account other than the default
	AccountID string
}

// FundingAddressUpdateResponse holds an updated withdrawal address's verification
type FundingAddressUpdateResponse struct {
	Verified bool `json:"verified"`
}
