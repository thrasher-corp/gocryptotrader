package deribit

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var errInvalidWithdrawalID = errors.New("invalid withdrawal ID")

// AddToAddressBookRequest adds an address, or a Coinbase wallet type account's Coinbase Travel Network counterparty, to
// the address book
type AddToAddressBookRequest struct {
	// Currency is required, except for a counterparty, which applies to every currency
	Currency currency.Code `json:"currency,omitzero"`
	// Type is transfer, withdrawal or deposit_source; Coinbase wallet type accounts add transfer and withdrawal entries
	// only
	Type string `json:"type"`
	// Address is required unless CounterpartyID is set
	Address string `json:"address,omitempty"`
	// Network, Tag and CounterpartyID apply to Coinbase wallet type accounts: Network is the resource name of one of
	// the currency's Coinbase networks, which an address needs, Tag a destination tag or memo, and CounterpartyID a
	// counterparty added in place of an address and network
	Network        string `json:"network,omitempty"`
	Tag            string `json:"tag,omitempty"`
	CounterpartyID string `json:"counterparty_id,omitempty"`
	Label          string `json:"label"`
	// The beneficiary fields, Agreed, Personal and ExtraCurrencies apply to legacy wallet type accounts;
	// BeneficiaryVASPWebsite is required for a VASP Deribit does not list
	BeneficiaryVASPName    string `json:"beneficiary_vasp_name,omitempty"`
	BeneficiaryVASPDID     string `json:"beneficiary_vasp_did,omitempty"`
	BeneficiaryVASPWebsite string `json:"beneficiary_vasp_website,omitempty"`
	BeneficiaryFirstName   string `json:"beneficiary_first_name,omitempty"`
	BeneficiaryLastName    string `json:"beneficiary_last_name,omitempty"`
	BeneficiaryCompanyName string `json:"beneficiary_company_name,omitempty"`
	BeneficiaryAddress     string `json:"beneficiary_address,omitempty"`
	// Agreed confirms the beneficiary information may be shared with third parties, and Personal that the address is
	// the user's own unhosted wallet; they are pointers so false can be sent
	Agreed   *bool `json:"agreed,omitempty"`
	Personal *bool `json:"personal,omitempty"`
	// ExtraCurrencies limits an ERC20 address to these ERC20 currencies, as it is otherwise added for all of them
	ExtraCurrencies []currency.Code `json:"extra_currencies,omitempty"`
	// Name, Country, FinancialInstitution, Account, AccountLocation, EntityType, WalletType and IsSelf are a Coinbase
	// wallet type account's travel rule data, which Deribit discards for a counterparty; Name and Country, an
	// ISO 3166-1 alpha-2 code, satisfy the travel rule
	Name                 string `json:"name,omitempty"`
	Country              string `json:"country,omitempty"`
	FinancialInstitution string `json:"financial_institution,omitempty"`
	Account              string `json:"account,omitempty"`
	AccountLocation      string `json:"account_location,omitempty"`
	// EntityType is organization or individual, and WalletType exchange or self_hosted
	EntityType string `json:"entity_type,omitempty"`
	WalletType string `json:"wallet_type,omitempty"`
	// IsSelf marks the beneficiary as the account owner; it is a pointer so false can be sent
	IsSelf *bool `json:"is_self,omitempty"`
}

// AddressBookEntryResponse contains an address book entry or deposit address. Legacy wallet type accounts receive the
// beneficiary fields, Agreed and Personal, and deposit addresses with only their address, currency, type and creation
// time; Coinbase wallet type accounts receive the ID, kind, network and travel rule fields, leaving out unset ones
type AddressBookEntryResponse struct {
	// ID identifies a Coinbase wallet type entry, in place of its address
	ID string `json:"id"`
	// Currency is "all" for a Coinbase Travel Network counterparty, which applies to every currency
	Currency currency.Code `json:"currency"`
	// Address is a Coinbase Travel Network counterparty's ID when Kind is ctn
	Address string `json:"address"`
	// Type is transfer, withdrawal, deposit_source or, for a deposit address, deposit
	Type string `json:"type"`
	// Kind is onchain for a blockchain address and ctn for a Coinbase Travel Network counterparty
	Kind              string     `json:"kind"`
	Label             string     `json:"label"`
	CreationTimestamp types.Time `json:"creation_timestamp"`
	UpdatedTimestamp  types.Time `json:"updated_timestamp"`
	// Status is admin_locked, waiting, confirmed or ready for a legacy wallet type entry, and active, for a deposit
	// address, waiting or ready for a Coinbase wallet type entry; an entry waits out the account's address delay
	Status           string             `json:"status"`
	WaitingTimestamp AddressWaitingTime `json:"waiting_timestamp"`
	// RequiresConfirmation is set when withdrawals to the address need email confirmation, and
	// RequiresConfirmationChange while a change to that setting awaits confirmation
	RequiresConfirmation       bool `json:"requires_confirmation"`
	RequiresConfirmationChange bool `json:"requires_confirmation_change"`
	RequiresSecurityKey        bool `json:"requires_security_key"`
	// InfoRequired is set when the entry needs more beneficiary information; Coinbase wallet type deposit addresses and
	// counterparties always set it
	InfoRequired       bool   `json:"info_required"`
	Network            string `json:"network"`
	NetworkDisplayName string `json:"network_display_name"`
	Tag                string `json:"tag"`
	// The beneficiary fields, Agreed and Personal are a legacy wallet type entry's travel rule information
	BeneficiaryVASPName    string `json:"beneficiary_vasp_name"`
	BeneficiaryVASPDID     string `json:"beneficiary_vasp_did"`
	BeneficiaryVASPWebsite string `json:"beneficiary_vasp_website"`
	BeneficiaryFirstName   string `json:"beneficiary_first_name"`
	BeneficiaryLastName    string `json:"beneficiary_last_name"`
	BeneficiaryCompanyName string `json:"beneficiary_company_name"`
	BeneficiaryAddress     string `json:"beneficiary_address"`
	Agreed                 bool   `json:"agreed"`
	Personal               bool   `json:"personal"`
	// Name to IsSelf are a Coinbase wallet type entry's travel rule data, and the beneficiary timestamps record when it
	// was first stored and last changed
	Name                        string     `json:"name"`
	Country                     string     `json:"country"`
	FinancialInstitution        string     `json:"financial_institution"`
	Account                     string     `json:"account"`
	AccountLocation             string     `json:"account_location"`
	EntityType                  string     `json:"entity_type"`
	WalletType                  string     `json:"wallet_type"`
	IsSelf                      bool       `json:"is_self"`
	BeneficiaryCreatedTimestamp types.Time `json:"beneficiary_created"`
	BeneficiaryUpdatedTimestamp types.Time `json:"beneficiary_updated"`
}

// AddressWaitingTime is when a waiting address book entry becomes usable; Deribit documents it as a boolean while its
// examples send a millisecond timestamp, or null for an entry not waiting, so a boolean decodes as the zero time
type AddressWaitingTime types.Time

// UnmarshalJSON decodes a timestamp, or a boolean as the zero time
func (a *AddressWaitingTime) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("true")) || bytes.Equal(data, []byte("false")) {
		*a = AddressWaitingTime{}
		return nil
	}
	return (*types.Time)(a).UnmarshalJSON(data)
}

// Time returns when the entry becomes usable
func (a AddressWaitingTime) Time() time.Time {
	return time.Time(a)
}

// CancelTransferByIDRequest cancels a transfer that has not been processed
type CancelTransferByIDRequest struct {
	Currency currency.Code `json:"currency"`
	ID       uint64        `json:"id"`
}

// TransferResponse contains a transfer between the account and a subaccount or another user
type TransferResponse struct {
	ID uint64 `json:"id"`
	// Type is user for a transfer to or from another user, and subaccount for one to or from a subaccount
	Type     string        `json:"type"`
	Currency currency.Code `json:"currency"`
	Amount   float64       `json:"amount"`
	// OtherSide is the subaccount's name for a subaccount transfer, the destination address for a transfer to another
	// user and the sender's username for one from another user
	OtherSide string `json:"other_side"`
	// State is prepared, confirmed, cancelled, waiting_for_admin, insufficient_funds, withdrawal_limit or otherwise the
	// reason the transfer was rejected
	State string `json:"state"`
	// Direction is payment or income
	Direction        string     `json:"direction"`
	CreatedTimestamp types.Time `json:"created_timestamp"`
	UpdatedTimestamp types.Time `json:"updated_timestamp"`
	// Nonce is the idempotency nonce the transfer was requested with
	Nonce string `json:"nonce"`
}

// CancelWithdrawalRequest cancels a withdrawal that has not been processed
type CancelWithdrawalRequest struct {
	Currency currency.Code `json:"currency"`
	ID       uint64        `json:"id"`
}

// WithdrawalResponse contains a withdrawal. Legacy wallet type accounts receive numeric IDs, fees, priorities and
// nonces; Coinbase wallet type accounts receive UUID IDs and the custodian, network and clearance fields, leaving out
// unset ones
type WithdrawalResponse struct {
	ID       WithdrawalID  `json:"id"`
	Currency currency.Code `json:"currency"`
	// State is unconfirmed, confirmed, cancelled, completed, interrupted or rejected for a legacy wallet type
	// withdrawal, and prepared, awaiting_second_email, unconfirmed, completed or cancelled for a Coinbase wallet type
	// withdrawal, which starts as prepared until confirmed by email unless the address is trusted
	State  string  `json:"state"`
	Amount float64 `json:"amount"`
	// Address is a Coinbase Travel Network counterparty's ID for a withdrawal to one
	Address string `json:"address"`
	// TransactionID is empty until the transaction is broadcast
	TransactionID string `json:"transaction_id"`
	// Fee is in the withdrawn currency
	Fee float64 `json:"fee"`
	// Priority is the ID of the withdrawal's priority level
	Priority float64 `json:"priority"`
	// Nonce is the idempotency nonce the withdrawal was requested with
	Nonce string `json:"nonce"`
	// TransferIdempotencyKey is the custodian's idempotency key, set once the withdrawal is submitted to it
	TransferIdempotencyKey string `json:"transfer_idem"`
	Network                string `json:"network"`
	NetworkDisplayName     string `json:"network_display_name"`
	DestinationTag         string `json:"destination_tag"`
	// PaymentID is assigned by the custodian
	PaymentID string `json:"payment_id"`
	// SourceCTNID and DestinationCTNID identify the Coinbase Travel Network counterparties of a withdrawal to one
	SourceCTNID      string     `json:"source_ctn_id"`
	DestinationCTNID string     `json:"destination_ctn_id"`
	CreatedTimestamp types.Time `json:"created_timestamp"`
	UpdatedTimestamp types.Time `json:"updated_timestamp"`
	// ConfirmedTimestamp is zero until the withdrawal is confirmed
	ConfirmedTimestamp types.Time `json:"confirmed_timestamp"`
	// ClearanceState is the compliance clearance state, left out until a clearance decision is recorded
	ClearanceState string `json:"clearance_state"`
	Note           string `json:"note"`
}

// WithdrawalID identifies a withdrawal; a legacy wallet type withdrawal's numeric ID decodes as its decimal text, and a
// Coinbase wallet type withdrawal's UUID as sent
type WithdrawalID string

// UnmarshalJSON decodes a numeric or string withdrawal ID
func (w *WithdrawalID) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		return nil
	}
	if len(data) > 0 && data[0] == '"' {
		var id string
		if err := json.Unmarshal(data, &id); err != nil {
			return fmt.Errorf("%w: %w", errInvalidWithdrawalID, err)
		}
		*w = WithdrawalID(id)
		return nil
	}
	if _, err := strconv.ParseUint(string(data), 10, 64); err != nil {
		return fmt.Errorf("%w: %s", errInvalidWithdrawalID, data)
	}
	*w = WithdrawalID(data)
	return nil
}

// CoinbaseEstimateWithdrawalFeeRequest estimates a Coinbase wallet type withdrawal's network fee
type CoinbaseEstimateWithdrawalFeeRequest struct {
	Currency currency.Code `json:"currency"`
	// NetworkName is the resource name of one of the currency's Coinbase networks, which Deribit passes to the
	// custodian unchecked
	NetworkName string `json:"network_name"`
}

// WithdrawalFeeEstimateResponse contains a Coinbase wallet type withdrawal's estimated network fee, in the withdrawn
// currency; the fee charged at settlement may differ
type WithdrawalFeeEstimateResponse struct {
	EstimatedFee types.Number `json:"estimated_fee"`
	// EstimatedFeeBeforeSubsidy falls back to EstimatedFee when the custodian does not report it, so equal fees do not
	// show there was no subsidy
	EstimatedFeeBeforeSubsidy types.Number `json:"estimated_fee_before_subsidy"`
	// ExpiresAt is zero when the custodian sets no expiry
	ExpiresAt time.Time `json:"expires_at"`
}

// CreateDepositAddressRequest creates a deposit address
type CreateDepositAddressRequest struct {
	Currency currency.Code `json:"currency"`
	// Network and Label apply to Coinbase wallet type accounts, which need Network, the resource name of one of the
	// currency's Coinbase networks; Label defaults to "deposit address"
	Network string `json:"network,omitempty"`
	Label   string `json:"label,omitempty"`
}

// AddressBeneficiaryRequest selects the beneficiary information of an address
type AddressBeneficiaryRequest struct {
	Currency currency.Code `json:"currency"`
	Address  string        `json:"address"`
	// Tag is an XRP address's destination tag
	Tag string `json:"tag,omitempty"`
}

// AddressBeneficiaryResponse contains an address's beneficiary information, which the travel rule needs for withdrawals
// to it
type AddressBeneficiaryResponse struct {
	Currency currency.Code `json:"currency"`
	Address  string        `json:"address"`
	// Tag is an XRP address's destination tag
	Tag    string `json:"tag"`
	UserID uint64 `json:"user_id"`
	// Agreed confirms the information may be shared with third parties, Personal that the address is the user's own
	// unhosted wallet, and Unhosted that it belongs to an unhosted wallet
	Agreed                 bool       `json:"agreed"`
	Personal               bool       `json:"personal"`
	Unhosted               bool       `json:"unhosted"`
	BeneficiaryVASPName    string     `json:"beneficiary_vasp_name"`
	BeneficiaryVASPDID     string     `json:"beneficiary_vasp_did"`
	BeneficiaryVASPWebsite string     `json:"beneficiary_vasp_website"`
	BeneficiaryFirstName   string     `json:"beneficiary_first_name"`
	BeneficiaryLastName    string     `json:"beneficiary_last_name"`
	BeneficiaryCompanyName string     `json:"beneficiary_company_name"`
	BeneficiaryAddress     string     `json:"beneficiary_address"`
	CreatedTimestamp       types.Time `json:"created"`
	UpdatedTimestamp       types.Time `json:"updated"`
}

// GetAddressBookRequest selects address book entries
type GetAddressBookRequest struct {
	// Currency may be "all" for a Coinbase wallet type account, whose counterparties are listed under it alone
	Currency currency.Code `json:"currency"`
	// Type is transfer, withdrawal, deposit_source, for legacy wallet type accounts, or deposit, for Coinbase wallet
	// type accounts' deposit addresses
	Type string `json:"type"`
}

// GetCurrentDepositAddressRequest selects a currency's current deposit address
type GetCurrentDepositAddressRequest struct {
	Currency currency.Code `json:"currency"`
	// Network is required by Coinbase wallet type accounts, whose deposit addresses are per network
	Network string `json:"network,omitempty"`
}

// WalletHistoryRequest requests a page of a currency's latest deposits, transfers or withdrawals
type WalletHistoryRequest struct {
	Currency currency.Code `json:"currency"`
	// Count is up to 1000, and 10 when zero
	Count  uint64 `json:"count,omitempty"`
	Offset uint64 `json:"offset,omitempty"`
}

// DepositsResponse contains a page of deposits
type DepositsResponse struct {
	Deposits []DepositResponse `json:"data"`
	// Count is the number of deposits available, not the number returned
	Count uint64 `json:"count"`
}

// DepositResponse contains a deposit
type DepositResponse struct {
	Currency currency.Code `json:"currency"`
	// UserID is the receiving user, which Deribit documents for SetClearanceOriginator's result only
	UserID  uint64  `json:"user_id"`
	Address string  `json:"address"`
	Amount  float64 `json:"amount"`
	// State is pending until compliance checks finish, then completed or rejected, or replaced when the transaction was
	// replaced on chain under a new hash
	State             string     `json:"state"`
	TransactionID     string     `json:"transaction_id"`
	SourceAddress     string     `json:"source_address"`
	ReceivedTimestamp types.Time `json:"received_timestamp"`
	UpdatedTimestamp  types.Time `json:"updated_timestamp"`
	Note              string     `json:"note"`
	// ClearanceState is in_progress, pending_admin_decision, pending_user_input, success, failed, cancelled,
	// refund_initiated or refunded
	ClearanceState      string `json:"clearance_state"`
	RefundTransactionID string `json:"refund_transaction_id"`
}

// RewardEligibilityResponse holds each reward bearing currency's reward eligibility, keyed by its upper case code
type RewardEligibilityResponse map[currency.Code]RewardEligibility

// UnmarshalJSON decodes each currency's reward eligibility, keying it by the upper case code, as Deribit's lower case
// keys would decode as codes that do not match the package's currency codes
func (r *RewardEligibilityResponse) UnmarshalJSON(data []byte) error {
	var eligibility map[string]RewardEligibility
	if err := json.Unmarshal(data, &eligibility); err != nil {
		return err
	}
	if eligibility == nil {
		return nil
	}
	*r = make(RewardEligibilityResponse, len(eligibility))
	for code, currencyEligibility := range eligibility {
		(*r)[currency.NewCode(strings.ToUpper(code))] = currencyEligibility
	}
	return nil
}

// RewardEligibility contains a currency's reward eligibility and recent reward rate
type RewardEligibility struct {
	// EligibilityStatus is eligible, partially_eligible when custody balances are excluded, or non_eligible
	EligibilityStatus string `json:"eligibility_status"`
	// APRSimpleMovingAverage7Day is the simple moving average of the last seven days' annual percentage rate of reward
	APRSimpleMovingAverage7Day float64 `json:"apr_sma7"`
}

// TransfersResponse contains a page of transfers
type TransfersResponse struct {
	Transfers []TransferResponse `json:"data"`
	// Count is the number of transfers available, not the number returned
	Count uint64 `json:"count"`
}

// WithdrawalsResponse contains a page of withdrawals
type WithdrawalsResponse struct {
	Withdrawals []WithdrawalResponse `json:"data"`
	// Count is the number of withdrawals available, not the number returned
	Count uint64 `json:"count"`
}

// ListAddressBeneficiariesRequest filters and pages address beneficiaries; zero fields do not filter
type ListAddressBeneficiariesRequest struct {
	Currency currency.Code `json:"currency,omitzero"`
	Address  string        `json:"address,omitempty"`
	// Tag is an XRP address's destination tag
	Tag           string    `json:"tag,omitempty"`
	CreatedBefore time.Time `json:"-"`
	CreatedAfter  time.Time `json:"-"`
	UpdatedBefore time.Time `json:"-"`
	UpdatedAfter  time.Time `json:"-"`
	// Personal and Unhosted filter by those flags when set
	Personal               *bool  `json:"personal,omitempty"`
	Unhosted               *bool  `json:"unhosted,omitempty"`
	BeneficiaryVASPName    string `json:"beneficiary_vasp_name,omitempty"`
	BeneficiaryVASPDID     string `json:"beneficiary_vasp_did,omitempty"`
	BeneficiaryVASPWebsite string `json:"beneficiary_vasp_website,omitempty"`
	// Limit is up to 1000, and 100 when zero
	Limit uint64 `json:"limit,omitempty"`
	// Continuation is the previous page's continuation token
	Continuation string `json:"continuation,omitempty"`
}

// ListAddressBeneficiariesParams contains a list address beneficiaries request's params, with its times in Unix
// milliseconds
type ListAddressBeneficiariesParams struct {
	*ListAddressBeneficiariesRequest
	CreatedBefore uint64 `json:"created_before,omitempty"`
	CreatedAfter  uint64 `json:"created_after,omitempty"`
	UpdatedBefore uint64 `json:"updated_before,omitempty"`
	UpdatedAfter  uint64 `json:"updated_after,omitempty"`
}

// AddressBeneficiariesResponse contains a page of address beneficiaries and the continuation token of the next page
type AddressBeneficiariesResponse struct {
	Beneficiaries []AddressBeneficiaryResponse `json:"data"`
	Continuation  string                       `json:"continuation"`
	// Count is the number of beneficiaries available, not the number returned
	Count uint64 `json:"count"`
}

// RemoveFromAddressBookRequest removes an address book entry
type RemoveFromAddressBookRequest struct {
	// Currency may be "all" for a Coinbase wallet type account's counterparty
	Currency currency.Code `json:"currency"`
	// Type is transfer or withdrawal
	Type string `json:"type"`
	// Address identifies the entry, which Coinbase wallet type accounts may identify by EntryID instead, or tell apart
	// from entries sharing its address by Network and Tag
	Address string `json:"address,omitempty"`
	Network string `json:"network,omitempty"`
	Tag     string `json:"tag,omitempty"`
	EntryID string `json:"id,omitempty"`
}

// SaveAddressBeneficiaryRequest saves an address's beneficiary information
type SaveAddressBeneficiaryRequest struct {
	Currency currency.Code `json:"currency"`
	Address  string        `json:"address"`
	// Tag is an XRP address's destination tag
	Tag string `json:"tag,omitempty"`
	// Agreed confirms the information may be shared with third parties, Personal that the address is the user's own
	// unhosted wallet, and Unhosted that it belongs to an unhosted wallet
	Agreed              bool   `json:"agreed"`
	Personal            bool   `json:"personal"`
	Unhosted            bool   `json:"unhosted"`
	BeneficiaryVASPName string `json:"beneficiary_vasp_name"`
	BeneficiaryVASPDID  string `json:"beneficiary_vasp_did"`
	// BeneficiaryVASPWebsite is required for a VASP Deribit does not list
	BeneficiaryVASPWebsite string `json:"beneficiary_vasp_website,omitempty"`
	BeneficiaryFirstName   string `json:"beneficiary_first_name,omitempty"`
	BeneficiaryLastName    string `json:"beneficiary_last_name,omitempty"`
	BeneficiaryCompanyName string `json:"beneficiary_company_name,omitempty"`
	BeneficiaryAddress     string `json:"beneficiary_address"`
}

// SetAddressRequiresSecurityKeyRequest sets whether withdrawals to an address book entry need a security key
type SetAddressRequiresSecurityKeyRequest struct {
	// Currency may be "all" for a Coinbase wallet type account's counterparty
	Currency currency.Code `json:"currency"`
	// Type is withdrawal, or transfer for a Coinbase wallet type account
	Type string `json:"type"`
	// Address identifies the entry, which Coinbase wallet type accounts may identify by EntryID instead, or tell apart
	// from entries sharing its address by Network and Tag
	Address             string `json:"address,omitempty"`
	Network             string `json:"network,omitempty"`
	Tag                 string `json:"tag,omitempty"`
	EntryID             string `json:"id,omitempty"`
	RequiresSecurityKey bool   `json:"value"`
}

// SetClearanceOriginatorRequest sets the originator of a deposit, which the travel rule needs to clear it
type SetClearanceOriginatorRequest struct {
	Deposit    ClearanceDepositID  `json:"deposit_id"`
	Originator ClearanceOriginator `json:"originator"`
}

// ClearanceDepositID identifies a deposit by its currency, receiving user, address and transaction hash
type ClearanceDepositID struct {
	Currency        currency.Code `json:"currency"`
	UserID          uint64        `json:"user_id"`
	Address         string        `json:"address"`
	TransactionHash string        `json:"tx_hash"`
}

// ClearanceOriginator contains the travel rule information of a deposit's originator
type ClearanceOriginator struct {
	IsPersonal  bool   `json:"is_personal"`
	FirstName   string `json:"first_name,omitempty"`
	LastName    string `json:"last_name,omitempty"`
	CompanyName string `json:"company_name,omitempty"`
	Address     string `json:"address,omitempty"`
}

// SetCoinbaseSubaccountWithdrawalsAllowedRequest contains whether subaccounts may withdraw, always sent so false can be
type SetCoinbaseSubaccountWithdrawalsAllowedRequest struct {
	Enabled bool `json:"enabled"`
}

// CoinbaseSubaccountWithdrawalsResponse contains whether a Coinbase wallet type account's subaccounts may withdraw
type CoinbaseSubaccountWithdrawalsResponse struct {
	Enabled bool `json:"enabled"`
}

// SubmitTransferBetweenSubaccountsRequest transfers funds between subaccounts, or a subaccount and the main account
type SubmitTransferBetweenSubaccountsRequest struct {
	Currency currency.Code `json:"currency"`
	Amount   float64       `json:"amount"`
	// Destination and Source are account IDs; Source defaults to the requesting account, and only a request with the
	// mainaccount scope may name another subaccount
	Destination uint64 `json:"destination"`
	Source      uint64 `json:"source,omitempty"`
	// Nonce, of 8 to 128 characters, makes a repeated request return the transfer it created instead of another
	Nonce string `json:"nonce,omitempty"`
}

// SubmitTransferToSubaccountRequest transfers funds from the main account to a subaccount
type SubmitTransferToSubaccountRequest struct {
	Currency    currency.Code `json:"currency"`
	Amount      float64       `json:"amount"`
	Destination uint64        `json:"destination"`
}

// SubmitTransferToUserRequest transfers funds from the main account to another Deribit user
type SubmitTransferToUserRequest struct {
	Currency currency.Code `json:"currency"`
	Amount   float64       `json:"amount"`
	// Destination is the user's address from the address book
	Destination string `json:"destination"`
}

// UpdateInAddressBookRequest updates an address book entry's label and beneficiary information
type UpdateInAddressBookRequest struct {
	// Currency may be "all" for a Coinbase wallet type account's counterparty
	Currency currency.Code `json:"currency"`
	// Type is transfer, withdrawal or deposit_source; Coinbase wallet type accounts update transfer and withdrawal
	// entries only
	Type string `json:"type"`
	// Address identifies the entry, which Coinbase wallet type accounts may identify by EntryID instead, or tell apart
	// from entries sharing its address by Network and Tag
	Address string `json:"address,omitempty"`
	Network string `json:"network,omitempty"`
	Tag     string `json:"tag,omitempty"`
	EntryID string `json:"id,omitempty"`
	// Label is required for legacy wallet type accounts
	Label string `json:"label,omitempty"`
	// The beneficiary fields, Agreed and Personal apply to legacy wallet type accounts; BeneficiaryVASPWebsite is
	// required for a VASP Deribit does not list
	BeneficiaryVASPName    string `json:"beneficiary_vasp_name,omitempty"`
	BeneficiaryVASPDID     string `json:"beneficiary_vasp_did,omitempty"`
	BeneficiaryVASPWebsite string `json:"beneficiary_vasp_website,omitempty"`
	BeneficiaryFirstName   string `json:"beneficiary_first_name,omitempty"`
	BeneficiaryLastName    string `json:"beneficiary_last_name,omitempty"`
	BeneficiaryCompanyName string `json:"beneficiary_company_name,omitempty"`
	BeneficiaryAddress     string `json:"beneficiary_address,omitempty"`
	// Agreed confirms the beneficiary information may be shared with third parties, and Personal that the address is
	// the user's own unhosted wallet; they are pointers so false can be sent
	Agreed   *bool `json:"agreed,omitempty"`
	Personal *bool `json:"personal,omitempty"`
	// Name, Country, FinancialInstitution, Account, AccountLocation, EntityType, WalletType and IsSelf are a Coinbase
	// wallet type account's travel rule data; Name and Country, an ISO 3166-1 alpha-2 code, satisfy the travel rule
	Name                 string `json:"name,omitempty"`
	Country              string `json:"country,omitempty"`
	FinancialInstitution string `json:"financial_institution,omitempty"`
	Account              string `json:"account,omitempty"`
	AccountLocation      string `json:"account_location,omitempty"`
	// EntityType is organization or individual, and WalletType exchange or self_hosted
	EntityType string `json:"entity_type,omitempty"`
	WalletType string `json:"wallet_type,omitempty"`
	// IsSelf marks the beneficiary as the account owner; it is a pointer so false can be sent
	IsSelf *bool `json:"is_self,omitempty"`
}

// UpdateInAddressBookResponse contains a Coinbase wallet type account's updated address book entry; legacy wallet type
// accounts are only sent an acknowledgement, which leaves Entry nil
type UpdateInAddressBookResponse struct {
	Entry *AddressBookEntryResponse
}

// UnmarshalJSON decodes a legacy wallet type account's "ok" acknowledgement, leaving Entry nil, or a Coinbase wallet
// type account's updated entry
func (u *UpdateInAddressBookResponse) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || data[0] != '"' {
		return json.Unmarshal(data, &u.Entry)
	}
	var acknowledgement string
	if err := json.Unmarshal(data, &acknowledgement); err != nil {
		return err
	}
	if acknowledgement != resultOK {
		return fmt.Errorf("%w: update_in_address_book returned %q", errUnexpectedResult, acknowledgement)
	}
	u.Entry = nil
	return nil
}

// WithdrawRequest withdraws funds to an address book address
type WithdrawRequest struct {
	Currency currency.Code `json:"currency"`
	Amount   float64       `json:"amount"`
	// Address is the destination, which must be in the address book; Coinbase wallet type accounts may identify its
	// entry by EntryID instead, tell it apart from entries sharing its address by Network and Tag, or give a Coinbase
	// Travel Network counterparty's ID as the address
	Address string `json:"address,omitempty"`
	Network string `json:"network,omitempty"`
	Tag     string `json:"tag,omitempty"`
	EntryID string `json:"id,omitempty"`
	// Priority sets a legacy wallet type BTC withdrawal's fee, high by default
	Priority string `json:"priority,omitempty"`
	// Nonce, of 8 to 128 characters, makes a repeated legacy wallet type request return the withdrawal it created
	// instead of another; Coinbase wallet type accounts ignore it, so a retry withdraws again
	Nonce string `json:"nonce,omitempty"`
}
