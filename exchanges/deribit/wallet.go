package deribit

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"unicode/utf8"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

// Address book types, which select address book entries in requests and identify them in responses
const (
	AddressBookTypeTransfer   = "transfer"
	AddressBookTypeWithdrawal = "withdrawal"
	// AddressBookTypeDepositSource registers a legacy wallet type account's deposit source, giving the originator
	// information of deposits from it
	AddressBookTypeDepositSource = "deposit_source"
	// AddressBookTypeDeposit selects a Coinbase wallet type account's deposit addresses, which only GetAddressBook
	// reads
	AddressBookTypeDeposit = "deposit"
)

// Withdrawal priorities, which set a legacy wallet type BTC withdrawal's fee
const (
	WithdrawalPriorityInsane      = "insane"
	WithdrawalPriorityExtremeHigh = "extreme_high"
	WithdrawalPriorityVeryHigh    = "very_high"
	WithdrawalPriorityHigh        = "high"
	WithdrawalPriorityMid         = "mid"
	WithdrawalPriorityLow         = "low"
	WithdrawalPriorityVeryLow     = "very_low"
)

// Travel rule entity and wallet types of a Coinbase wallet type account's beneficiary
const (
	BeneficiaryEntityTypeOrganisation = "organization"
	BeneficiaryEntityTypeIndividual   = "individual"
	BeneficiaryWalletTypeExchange     = "exchange"
	BeneficiaryWalletTypeSelfHosted   = "self_hosted"
)

const (
	// maximumWalletHistoryCount is the most deposits, transfers or withdrawals a page holds
	maximumWalletHistoryCount = 1000
	// maximumAddressBeneficiariesLimit is the most address beneficiaries a page holds
	maximumAddressBeneficiariesLimit = 1000
	// minimumIdempotencyNonceLength and maximumIdempotencyNonceLength bound an idempotency nonce's characters
	minimumIdempotencyNonceLength = 8
	maximumIdempotencyNonceLength = 128
)

var (
	errInvalidAddressBookType           = errors.New("invalid address book type")
	errAddressRequired                  = errors.New("address is required")
	errAddressOrCounterpartyRequired    = errors.New("address or counterparty ID is required")
	errAddressOrEntryIDRequired         = errors.New("address or address book entry ID is required")
	errInvalidBeneficiaryEntityType     = errors.New("invalid beneficiary entity type")
	errInvalidBeneficiaryWalletType     = errors.New("invalid beneficiary wallet type")
	errBeneficiaryVASPNameRequired      = errors.New("beneficiary VASP name is required")
	errBeneficiaryVASPDIDRequired       = errors.New("beneficiary VASP DID is required")
	errBeneficiaryAddressRequired       = errors.New("beneficiary address is required")
	errTransferIDRequired               = errors.New("transfer ID is required")
	errWithdrawalIDRequired             = errors.New("withdrawal ID is required")
	errNetworkNameRequired              = errors.New("network name is required")
	errTransferDestinationRequired      = errors.New("transfer destination is required")
	errInvalidWithdrawalPriority        = errors.New("invalid withdrawal priority")
	errInvalidIdempotencyNonce          = errors.New("idempotency nonce must be 8 to 128 characters")
	errInvalidAddressBeneficiariesLimit = errors.New("address beneficiaries limit must be at most 1000")
	errDepositUserIDRequired            = errors.New("deposit user ID is required")
	errTransactionHashRequired          = errors.New("transaction hash is required")
)

// AddToAddressBook adds an address, or a Coinbase wallet type account's Coinbase Travel Network counterparty, to the
// address book; a new withdrawal address is usable once the account's address delay, three days by default, has passed
func (e *Exchange) AddToAddressBook(ctx context.Context, arg *AddToAddressBookRequest) (*AddressBookEntry, error) {
	return e.addToAddressBook(ctx, arg, false)
}

func (e *Exchange) addToAddressBook(ctx context.Context, arg *AddToAddressBookRequest, overWebsocket bool) (*AddressBookEntry, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if err := validateAddressBookType(arg.Type, AddressBookTypeTransfer, AddressBookTypeWithdrawal, AddressBookTypeDepositSource); err != nil {
		return nil, err
	}
	// A counterparty applies to every currency, so only an address needs one
	if arg.CounterpartyID == "" {
		if arg.Currency.IsEmpty() {
			return nil, currency.ErrCurrencyCodeEmpty
		}
		if arg.Address == "" {
			return nil, errAddressOrCounterpartyRequired
		}
	}
	if arg.Label == "" {
		return nil, errLabelRequired
	}
	if err := validateBeneficiaryTypes(arg.EntityType, arg.WalletType); err != nil {
		return nil, err
	}
	var resp *AddressBookEntry
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/add_to_address_book", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelTransferByID cancels a transfer that has not been processed
func (e *Exchange) CancelTransferByID(ctx context.Context, arg *CancelTransferByIDRequest) (*Transfer, error) {
	return e.cancelTransferByID(ctx, arg, false)
}

func (e *Exchange) cancelTransferByID(ctx context.Context, arg *CancelTransferByIDRequest, overWebsocket bool) (*Transfer, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.ID == 0 {
		return nil, errTransferIDRequired
	}
	var resp *Transfer
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/cancel_transfer_by_id", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelWithdrawal cancels a withdrawal that has not been processed; Deribit may already be processing one it still
// reports as pending, which cannot then be cancelled
func (e *Exchange) CancelWithdrawal(ctx context.Context, arg *CancelWithdrawalRequest) (*Withdrawal, error) {
	return e.cancelWithdrawal(ctx, arg, false)
}

func (e *Exchange) cancelWithdrawal(ctx context.Context, arg *CancelWithdrawalRequest, overWebsocket bool) (*Withdrawal, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.ID == 0 {
		return nil, errWithdrawalIDRequired
	}
	var resp *Withdrawal
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/cancel_withdrawal", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CoinbaseEstimateWithdrawalFee estimates the network fee of a withdrawal on one of a currency's Coinbase networks;
// only institutional Coinbase wallet type accounts may use it
func (e *Exchange) CoinbaseEstimateWithdrawalFee(ctx context.Context, arg *CoinbaseEstimateWithdrawalFeeRequest) (*WithdrawalFeeEstimateResponse, error) {
	return e.coinbaseEstimateWithdrawalFee(ctx, arg, false)
}

func (e *Exchange) coinbaseEstimateWithdrawalFee(ctx context.Context, arg *CoinbaseEstimateWithdrawalFeeRequest, overWebsocket bool) (*WithdrawalFeeEstimateResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.NetworkName == "" {
		return nil, errNetworkNameRequired
	}
	var resp *WithdrawalFeeEstimateResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/coinbase/estimate_withdrawal_fee", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CreateDepositAddress creates a deposit address, or returns common.ErrNoResponse when Deribit creates none: it creates
// a BTC address at most once a day and a single ERC20, Solana or XRP address, an ERC20 address serving every ERC20
// currency. Coinbase wallet type accounts name a network, whose existing address is returned unchanged
func (e *Exchange) CreateDepositAddress(ctx context.Context, arg *CreateDepositAddressRequest) (*AddressBookEntry, error) {
	return e.createDepositAddress(ctx, arg, false)
}

func (e *Exchange) createDepositAddress(ctx context.Context, arg *CreateDepositAddressRequest, overWebsocket bool) (*AddressBookEntry, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *AddressBookEntry
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/create_deposit_address", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("%w: no %s deposit address was created", common.ErrNoResponse, arg.Currency)
	}
	return resp, nil
}

// DeleteAddressBeneficiary deletes an address's beneficiary information
func (e *Exchange) DeleteAddressBeneficiary(ctx context.Context, arg *AddressBeneficiaryRequest) error {
	return e.deleteAddressBeneficiary(ctx, arg, false)
}

func (e *Exchange) deleteAddressBeneficiary(ctx context.Context, arg *AddressBeneficiaryRequest, overWebsocket bool) error {
	if err := arg.validate(); err != nil {
		return err
	}
	return e.sendOKRequest(ctx, overWebsocket, nonMatchingEPL, "private/delete_address_beneficiary", arg)
}

// GetAddressBeneficiary returns an address's beneficiary information
func (e *Exchange) GetAddressBeneficiary(ctx context.Context, arg *AddressBeneficiaryRequest) (*AddressBeneficiary, error) {
	return e.getAddressBeneficiary(ctx, arg, false)
}

func (e *Exchange) getAddressBeneficiary(ctx context.Context, arg *AddressBeneficiaryRequest, overWebsocket bool) (*AddressBeneficiary, error) {
	if err := arg.validate(); err != nil {
		return nil, err
	}
	var resp *AddressBeneficiary
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_address_beneficiary", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// validate checks an address beneficiary request names the address's currency and the address
func (a *AddressBeneficiaryRequest) validate() error {
	if a == nil {
		return common.ErrNilPointer
	}
	if a.Currency.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	if a.Address == "" {
		return errAddressRequired
	}
	return nil
}

// GetAddressBook returns the address book entries of a currency and type; a Coinbase wallet type account's Coinbase
// Travel Network counterparties are only listed under the "all" currency
func (e *Exchange) GetAddressBook(ctx context.Context, arg *GetAddressBookRequest) ([]AddressBookEntry, error) {
	return e.getAddressBook(ctx, arg, false)
}

func (e *Exchange) getAddressBook(ctx context.Context, arg *GetAddressBookRequest, overWebsocket bool) ([]AddressBookEntry, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := validateAddressBookType(arg.Type, AddressBookTypeTransfer, AddressBookTypeWithdrawal, AddressBookTypeDepositSource, AddressBookTypeDeposit); err != nil {
		return nil, err
	}
	var resp []AddressBookEntry
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_address_book", arg, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetCurrentDepositAddress returns a currency's most recently created or used deposit address, or common.ErrNoResponse
// when it has none; Coinbase wallet type accounts name a network, whose address it returns
func (e *Exchange) GetCurrentDepositAddress(ctx context.Context, arg *GetCurrentDepositAddressRequest) (*AddressBookEntry, error) {
	return e.getCurrentDepositAddress(ctx, arg, false)
}

func (e *Exchange) getCurrentDepositAddress(ctx context.Context, arg *GetCurrentDepositAddressRequest, overWebsocket bool) (*AddressBookEntry, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *AddressBookEntry
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_current_deposit_address", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("%w: no %s deposit address exists", common.ErrNoResponse, arg.Currency)
	}
	return resp, nil
}

// GetDeposits returns a page of a currency's latest deposits
func (e *Exchange) GetDeposits(ctx context.Context, arg *WalletHistoryRequest) (*DepositsResponse, error) {
	return e.getDeposits(ctx, arg, false)
}

func (e *Exchange) getDeposits(ctx context.Context, arg *WalletHistoryRequest, overWebsocket bool) (*DepositsResponse, error) {
	if err := arg.validate(); err != nil {
		return nil, err
	}
	var resp *DepositsResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_deposits", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// validate checks a wallet history request names a currency and a page size Deribit accepts
func (w *WalletHistoryRequest) validate() error {
	if w == nil {
		return common.ErrNilPointer
	}
	if w.Currency.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	if w.Count > maximumWalletHistoryCount {
		return fmt.Errorf("%w: %d exceeds %d", errInvalidCount, w.Count, maximumWalletHistoryCount)
	}
	return nil
}

// GetRewardEligibility returns whether the account earns rewards on each reward bearing currency, and the currency's
// recent reward rate
func (e *Exchange) GetRewardEligibility(ctx context.Context) (RewardEligibilityResponse, error) {
	return e.getRewardEligibility(ctx, false)
}

func (e *Exchange) getRewardEligibility(ctx context.Context, overWebsocket bool) (RewardEligibilityResponse, error) {
	var resp RewardEligibilityResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_reward_eligibility", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetTransfers returns a page of a currency's latest transfers between the account and its subaccounts or other users
func (e *Exchange) GetTransfers(ctx context.Context, arg *WalletHistoryRequest) (*TransfersResponse, error) {
	return e.getTransfers(ctx, arg, false)
}

func (e *Exchange) getTransfers(ctx context.Context, arg *WalletHistoryRequest, overWebsocket bool) (*TransfersResponse, error) {
	if err := arg.validate(); err != nil {
		return nil, err
	}
	var resp *TransfersResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_transfers", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetWithdrawals returns a page of a currency's latest withdrawals
func (e *Exchange) GetWithdrawals(ctx context.Context, arg *WalletHistoryRequest) (*WithdrawalsResponse, error) {
	return e.getWithdrawals(ctx, arg, false)
}

func (e *Exchange) getWithdrawals(ctx context.Context, arg *WalletHistoryRequest, overWebsocket bool) (*WithdrawalsResponse, error) {
	if err := arg.validate(); err != nil {
		return nil, err
	}
	var resp *WithdrawalsResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_withdrawals", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// ListAddressBeneficiaries returns a page of address beneficiaries matching every filter set
func (e *Exchange) ListAddressBeneficiaries(ctx context.Context, arg *ListAddressBeneficiariesRequest) (*AddressBeneficiariesResponse, error) {
	return e.listAddressBeneficiaries(ctx, arg, false)
}

func (e *Exchange) listAddressBeneficiaries(ctx context.Context, arg *ListAddressBeneficiariesRequest, overWebsocket bool) (*AddressBeneficiariesResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Limit > maximumAddressBeneficiariesLimit {
		return nil, fmt.Errorf("%w: %d", errInvalidAddressBeneficiariesLimit, arg.Limit)
	}
	if !arg.CreatedAfter.IsZero() && !arg.CreatedBefore.IsZero() && arg.CreatedAfter.After(arg.CreatedBefore) {
		return nil, fmt.Errorf("%w: created after %s is after created before %s", common.ErrStartAfterEnd, arg.CreatedAfter, arg.CreatedBefore)
	}
	if !arg.UpdatedAfter.IsZero() && !arg.UpdatedBefore.IsZero() && arg.UpdatedAfter.After(arg.UpdatedBefore) {
		return nil, fmt.Errorf("%w: updated after %s is after updated before %s", common.ErrStartAfterEnd, arg.UpdatedAfter, arg.UpdatedBefore)
	}
	params := &listAddressBeneficiariesParams{ListAddressBeneficiariesRequest: arg}
	if !arg.CreatedBefore.IsZero() {
		params.CreatedBefore = arg.CreatedBefore.UnixMilli()
	}
	if !arg.CreatedAfter.IsZero() {
		params.CreatedAfter = arg.CreatedAfter.UnixMilli()
	}
	if !arg.UpdatedBefore.IsZero() {
		params.UpdatedBefore = arg.UpdatedBefore.UnixMilli()
	}
	if !arg.UpdatedAfter.IsZero() {
		params.UpdatedAfter = arg.UpdatedAfter.UnixMilli()
	}
	var resp *AddressBeneficiariesResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/list_address_beneficiaries", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// RemoveFromAddressBook removes an address book entry
func (e *Exchange) RemoveFromAddressBook(ctx context.Context, arg *RemoveFromAddressBookRequest) error {
	return e.removeFromAddressBook(ctx, arg, false)
}

func (e *Exchange) removeFromAddressBook(ctx context.Context, arg *RemoveFromAddressBookRequest, overWebsocket bool) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	if err := validateAddressBookEntry(arg.Currency, arg.Type, arg.Address, arg.EntryID, AddressBookTypeTransfer, AddressBookTypeWithdrawal); err != nil {
		return err
	}
	return e.sendOKRequest(ctx, overWebsocket, nonMatchingEPL, "private/remove_from_address_book", arg)
}

// SaveAddressBeneficiary saves an address's beneficiary information, which the travel rule needs for withdrawals to it,
// and returns it as stored
func (e *Exchange) SaveAddressBeneficiary(ctx context.Context, arg *SaveAddressBeneficiaryRequest) (*AddressBeneficiary, error) {
	return e.saveAddressBeneficiary(ctx, arg, false)
}

func (e *Exchange) saveAddressBeneficiary(ctx context.Context, arg *SaveAddressBeneficiaryRequest, overWebsocket bool) (*AddressBeneficiary, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Address == "" {
		return nil, errAddressRequired
	}
	if arg.BeneficiaryVASPName == "" {
		return nil, errBeneficiaryVASPNameRequired
	}
	if arg.BeneficiaryVASPDID == "" {
		return nil, errBeneficiaryVASPDIDRequired
	}
	if arg.BeneficiaryAddress == "" {
		return nil, errBeneficiaryAddressRequired
	}
	var resp *AddressBeneficiary
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/save_address_beneficiary", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// SetAddressRequiresSecurityKey sets whether withdrawals to an address book entry need a security key, and returns the
// updated entry; legacy wallet type accounts need the mainaccount scope and set it for withdrawal entries only
func (e *Exchange) SetAddressRequiresSecurityKey(ctx context.Context, arg *SetAddressRequiresSecurityKeyRequest) (*AddressBookEntry, error) {
	return e.setAddressRequiresSecurityKey(ctx, arg, false)
}

func (e *Exchange) setAddressRequiresSecurityKey(ctx context.Context, arg *SetAddressRequiresSecurityKeyRequest, overWebsocket bool) (*AddressBookEntry, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if err := validateAddressBookEntry(arg.Currency, arg.Type, arg.Address, arg.EntryID, AddressBookTypeTransfer, AddressBookTypeWithdrawal); err != nil {
		return nil, err
	}
	var resp *AddressBookEntry
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/set_address_requires_security_key", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// SetClearanceOriginator sets the originator of a deposit, which the travel rule needs before the deposit clears, and
// returns the deposit
func (e *Exchange) SetClearanceOriginator(ctx context.Context, arg *SetClearanceOriginatorRequest) (*Deposit, error) {
	return e.setClearanceOriginator(ctx, arg, false)
}

func (e *Exchange) setClearanceOriginator(ctx context.Context, arg *SetClearanceOriginatorRequest, overWebsocket bool) (*Deposit, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Deposit.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Deposit.UserID == 0 {
		return nil, errDepositUserIDRequired
	}
	if arg.Deposit.Address == "" {
		return nil, errAddressRequired
	}
	if arg.Deposit.TransactionHash == "" {
		return nil, errTransactionHashRequired
	}
	var resp *Deposit
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/set_clearance_originator", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// SetCoinbaseSubaccountWithdrawalsAllowed allows or forbids withdrawals from a Coinbase wallet type account's
// subaccounts, which are forbidden by default; only the main account may set it
func (e *Exchange) SetCoinbaseSubaccountWithdrawalsAllowed(ctx context.Context, enabled bool) (*CoinbaseSubaccountWithdrawalsResponse, error) {
	return e.setCoinbaseSubaccountWithdrawalsAllowed(ctx, enabled, false)
}

func (e *Exchange) setCoinbaseSubaccountWithdrawalsAllowed(ctx context.Context, enabled, overWebsocket bool) (*CoinbaseSubaccountWithdrawalsResponse, error) {
	var resp *CoinbaseSubaccountWithdrawalsResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/set_coinbase_subaccount_withdrawals_allowed", &setCoinbaseSubaccountWithdrawalsAllowedParams{Enabled: enabled}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// SubmitTransferBetweenSubaccounts transfers funds between subaccounts, or a subaccount and the main account; Deribit
// needs the main account's authorisation even when a subaccount sends it
func (e *Exchange) SubmitTransferBetweenSubaccounts(ctx context.Context, arg *SubmitTransferBetweenSubaccountsRequest) (*Transfer, error) {
	return e.submitTransferBetweenSubaccounts(ctx, arg, false)
}

func (e *Exchange) submitTransferBetweenSubaccounts(ctx context.Context, arg *SubmitTransferBetweenSubaccountsRequest, overWebsocket bool) (*Transfer, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidAmount, arg.Amount)
	}
	if arg.Destination == 0 {
		return nil, errTransferDestinationRequired
	}
	if err := validateIdempotencyNonce(arg.Nonce); err != nil {
		return nil, err
	}
	var resp *Transfer
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/submit_transfer_between_subaccounts", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// SubmitTransferToSubaccount transfers funds from the main account to a subaccount
func (e *Exchange) SubmitTransferToSubaccount(ctx context.Context, arg *SubmitTransferToSubaccountRequest) (*Transfer, error) {
	return e.submitTransferToSubaccount(ctx, arg, false)
}

func (e *Exchange) submitTransferToSubaccount(ctx context.Context, arg *SubmitTransferToSubaccountRequest, overWebsocket bool) (*Transfer, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidAmount, arg.Amount)
	}
	if arg.Destination == 0 {
		return nil, errTransferDestinationRequired
	}
	var resp *Transfer
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/submit_transfer_to_subaccount", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// SubmitTransferToUser transfers funds from the main account to another Deribit user's address book address, without a
// blockchain transaction; it needs the mainaccount scope
func (e *Exchange) SubmitTransferToUser(ctx context.Context, arg *SubmitTransferToUserRequest) (*Transfer, error) {
	return e.submitTransferToUser(ctx, arg, false)
}

func (e *Exchange) submitTransferToUser(ctx context.Context, arg *SubmitTransferToUserRequest, overWebsocket bool) (*Transfer, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidAmount, arg.Amount)
	}
	if arg.Destination == "" {
		return nil, errTransferDestinationRequired
	}
	var resp *Transfer
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/submit_transfer_to_user", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// UpdateInAddressBook updates an address book entry's label and beneficiary information, which addresses added before
// December 2024 may lack; only Coinbase wallet type accounts are returned the updated entry
func (e *Exchange) UpdateInAddressBook(ctx context.Context, arg *UpdateInAddressBookRequest) (*UpdateInAddressBookResponse, error) {
	return e.updateInAddressBook(ctx, arg, false)
}

func (e *Exchange) updateInAddressBook(ctx context.Context, arg *UpdateInAddressBookRequest, overWebsocket bool) (*UpdateInAddressBookResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if err := validateAddressBookEntry(arg.Currency, arg.Type, arg.Address, arg.EntryID, AddressBookTypeTransfer, AddressBookTypeWithdrawal, AddressBookTypeDepositSource); err != nil {
		return nil, err
	}
	if err := validateBeneficiaryTypes(arg.EntityType, arg.WalletType); err != nil {
		return nil, err
	}
	var resp *UpdateInAddressBookResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/update_in_address_book", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// Withdraw withdraws funds to an address book address, which needs the mainaccount scope, although a Coinbase wallet
// type subaccount may withdraw once its main account allows it. Deribit checks the balance on request and again on
// email confirmation, and may reject the withdrawal if the balance falls or the network fee rises in between. On an
// account with two-factor authentication, it returns a SecurityKeyChallengeError until the request is confirmed with
// WithSecurityKeyAuthorisation
func (e *Exchange) Withdraw(ctx context.Context, arg *WithdrawRequest) (*Withdrawal, error) {
	return e.withdraw(ctx, arg, false)
}

func (e *Exchange) withdraw(ctx context.Context, arg *WithdrawRequest, overWebsocket bool) (*Withdrawal, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidAmount, arg.Amount)
	}
	if arg.Address == "" && arg.EntryID == "" {
		return nil, errAddressOrEntryIDRequired
	}
	switch arg.Priority {
	case "", WithdrawalPriorityInsane, WithdrawalPriorityExtremeHigh, WithdrawalPriorityVeryHigh, WithdrawalPriorityHigh,
		WithdrawalPriorityMid, WithdrawalPriorityLow, WithdrawalPriorityVeryLow:
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidWithdrawalPriority, arg.Priority)
	}
	if err := validateIdempotencyNonce(arg.Nonce); err != nil {
		return nil, err
	}
	var resp *Withdrawal
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/withdraw", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// validateAddressBookType checks an address book type is one a method accepts
func validateAddressBookType(addressBookType string, accepted ...string) error {
	if !slices.Contains(accepted, addressBookType) {
		return fmt.Errorf("%w: %q", errInvalidAddressBookType, addressBookType)
	}
	return nil
}

// validateAddressBookEntry checks a request names an entry's currency, an address book type the method accepts, and the
// entry by its address or, for a Coinbase wallet type account, its ID
func validateAddressBookEntry(ccy currency.Code, addressBookType, address, entryID string, accepted ...string) error {
	if ccy.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	if err := validateAddressBookType(addressBookType, accepted...); err != nil {
		return err
	}
	if address == "" && entryID == "" {
		return errAddressOrEntryIDRequired
	}
	return nil
}

// validateBeneficiaryTypes checks a Coinbase wallet type beneficiary's entity and wallet types are documented ones, or
// left out
func validateBeneficiaryTypes(entityType, walletType string) error {
	switch entityType {
	case "", BeneficiaryEntityTypeOrganisation, BeneficiaryEntityTypeIndividual:
	default:
		return fmt.Errorf("%w: %q", errInvalidBeneficiaryEntityType, entityType)
	}
	switch walletType {
	case "", BeneficiaryWalletTypeExchange, BeneficiaryWalletTypeSelfHosted:
	default:
		return fmt.Errorf("%w: %q", errInvalidBeneficiaryWalletType, walletType)
	}
	return nil
}

// validateIdempotencyNonce checks an idempotency nonce is left out or 8 to 128 characters long
func validateIdempotencyNonce(nonce string) error {
	if nonce == "" {
		return nil
	}
	if length := utf8.RuneCountInString(nonce); length < minimumIdempotencyNonceLength || length > maximumIdempotencyNonceLength {
		return fmt.Errorf("%w: %d characters", errInvalidIdempotencyNonce, length)
	}
	return nil
}
