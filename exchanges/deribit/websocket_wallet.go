package deribit

import (
	"context"
)

// AddToAddressBook adds an address book entry as Exchange.AddToAddressBook does, over the websocket
func (w WebsocketAPI) AddToAddressBook(ctx context.Context, arg *AddToAddressBookRequest) (*AddressBookEntryResponse, error) {
	return w.e.addToAddressBook(ctx, arg, true)
}

// CancelTransferByID cancels a transfer as Exchange.CancelTransferByID does, over the websocket
func (w WebsocketAPI) CancelTransferByID(ctx context.Context, arg *CancelTransferByIDRequest) (*TransferResponse, error) {
	return w.e.cancelTransferByID(ctx, arg, true)
}

// CancelWithdrawal cancels a withdrawal as Exchange.CancelWithdrawal does, over the websocket
func (w WebsocketAPI) CancelWithdrawal(ctx context.Context, arg *CancelWithdrawalRequest) (*WithdrawalResponse, error) {
	return w.e.cancelWithdrawal(ctx, arg, true)
}

// CoinbaseEstimateWithdrawalFee estimates a withdrawal's network fee as Exchange.CoinbaseEstimateWithdrawalFee does,
// over the websocket
func (w WebsocketAPI) CoinbaseEstimateWithdrawalFee(ctx context.Context, arg *CoinbaseEstimateWithdrawalFeeRequest) (*WithdrawalFeeEstimateResponse, error) {
	return w.e.coinbaseEstimateWithdrawalFee(ctx, arg, true)
}

// CreateDepositAddress creates a deposit address as Exchange.CreateDepositAddress does, over the websocket
func (w WebsocketAPI) CreateDepositAddress(ctx context.Context, arg *CreateDepositAddressRequest) (*AddressBookEntryResponse, error) {
	return w.e.createDepositAddress(ctx, arg, true)
}

// DeleteAddressBeneficiary deletes an address's beneficiary information as Exchange.DeleteAddressBeneficiary does, over
// the websocket
func (w WebsocketAPI) DeleteAddressBeneficiary(ctx context.Context, arg *AddressBeneficiaryRequest) error {
	return w.e.deleteAddressBeneficiary(ctx, arg, true)
}

// GetAddressBeneficiary returns an address's beneficiary information as Exchange.GetAddressBeneficiary does, over the
// websocket
func (w WebsocketAPI) GetAddressBeneficiary(ctx context.Context, arg *AddressBeneficiaryRequest) (*AddressBeneficiaryResponse, error) {
	return w.e.getAddressBeneficiary(ctx, arg, true)
}

// GetAddressBook returns address book entries as Exchange.GetAddressBook does, over the websocket
func (w WebsocketAPI) GetAddressBook(ctx context.Context, arg *GetAddressBookRequest) ([]AddressBookEntryResponse, error) {
	return w.e.getAddressBook(ctx, arg, true)
}

// GetCurrentDepositAddress returns a currency's deposit address as Exchange.GetCurrentDepositAddress does, over the
// websocket
func (w WebsocketAPI) GetCurrentDepositAddress(ctx context.Context, arg *GetCurrentDepositAddressRequest) (*AddressBookEntryResponse, error) {
	return w.e.getCurrentDepositAddress(ctx, arg, true)
}

// GetDeposits returns a page of deposits as Exchange.GetDeposits does, over the websocket
func (w WebsocketAPI) GetDeposits(ctx context.Context, arg *WalletHistoryRequest) (*DepositsResponse, error) {
	return w.e.getDeposits(ctx, arg, true)
}

// GetRewardEligibility returns the account's reward eligibility as Exchange.GetRewardEligibility does, over the
// websocket
func (w WebsocketAPI) GetRewardEligibility(ctx context.Context) (RewardEligibilityResponse, error) {
	return w.e.getRewardEligibility(ctx, true)
}

// GetTransfers returns a page of transfers as Exchange.GetTransfers does, over the websocket
func (w WebsocketAPI) GetTransfers(ctx context.Context, arg *WalletHistoryRequest) (*TransfersResponse, error) {
	return w.e.getTransfers(ctx, arg, true)
}

// GetWithdrawals returns a page of withdrawals as Exchange.GetWithdrawals does, over the websocket
func (w WebsocketAPI) GetWithdrawals(ctx context.Context, arg *WalletHistoryRequest) (*WithdrawalsResponse, error) {
	return w.e.getWithdrawals(ctx, arg, true)
}

// ListAddressBeneficiaries returns a page of address beneficiaries as Exchange.ListAddressBeneficiaries does, over the
// websocket
func (w WebsocketAPI) ListAddressBeneficiaries(ctx context.Context, arg *ListAddressBeneficiariesRequest) (*AddressBeneficiariesResponse, error) {
	return w.e.listAddressBeneficiaries(ctx, arg, true)
}

// RemoveFromAddressBook removes an address book entry as Exchange.RemoveFromAddressBook does, over the websocket
func (w WebsocketAPI) RemoveFromAddressBook(ctx context.Context, arg *RemoveFromAddressBookRequest) error {
	return w.e.removeFromAddressBook(ctx, arg, true)
}

// SaveAddressBeneficiary saves an address's beneficiary information as Exchange.SaveAddressBeneficiary does, over the
// websocket
func (w WebsocketAPI) SaveAddressBeneficiary(ctx context.Context, arg *SaveAddressBeneficiaryRequest) (*AddressBeneficiaryResponse, error) {
	return w.e.saveAddressBeneficiary(ctx, arg, true)
}

// SetAddressRequiresSecurityKey sets whether withdrawals to an address book entry need a security key as
// Exchange.SetAddressRequiresSecurityKey does, over the websocket
func (w WebsocketAPI) SetAddressRequiresSecurityKey(ctx context.Context, arg *SetAddressRequiresSecurityKeyRequest) (*AddressBookEntryResponse, error) {
	return w.e.setAddressRequiresSecurityKey(ctx, arg, true)
}

// SetClearanceOriginator sets a deposit's originator as Exchange.SetClearanceOriginator does, over the websocket
func (w WebsocketAPI) SetClearanceOriginator(ctx context.Context, arg *SetClearanceOriginatorRequest) (*DepositResponse, error) {
	return w.e.setClearanceOriginator(ctx, arg, true)
}

// SetCoinbaseSubaccountWithdrawalsAllowed allows or forbids withdrawals from subaccounts as
// Exchange.SetCoinbaseSubaccountWithdrawalsAllowed does, over the websocket
func (w WebsocketAPI) SetCoinbaseSubaccountWithdrawalsAllowed(ctx context.Context, enabled bool) (*CoinbaseSubaccountWithdrawalsResponse, error) {
	return w.e.setCoinbaseSubaccountWithdrawalsAllowed(ctx, enabled, true)
}

// SubmitTransferBetweenSubaccounts transfers funds between subaccounts as Exchange.SubmitTransferBetweenSubaccounts
// does, over the websocket
func (w WebsocketAPI) SubmitTransferBetweenSubaccounts(ctx context.Context, arg *SubmitTransferBetweenSubaccountsRequest) (*TransferResponse, error) {
	return w.e.submitTransferBetweenSubaccounts(ctx, arg, true)
}

// SubmitTransferToSubaccount transfers funds to a subaccount as Exchange.SubmitTransferToSubaccount does, over the
// websocket
func (w WebsocketAPI) SubmitTransferToSubaccount(ctx context.Context, arg *SubmitTransferToSubaccountRequest) (*TransferResponse, error) {
	return w.e.submitTransferToSubaccount(ctx, arg, true)
}

// SubmitTransferToUser transfers funds to another user as Exchange.SubmitTransferToUser does, over the websocket
func (w WebsocketAPI) SubmitTransferToUser(ctx context.Context, arg *SubmitTransferToUserRequest) (*TransferResponse, error) {
	return w.e.submitTransferToUser(ctx, arg, true)
}

// UpdateInAddressBook updates an address book entry as Exchange.UpdateInAddressBook does, over the websocket
func (w WebsocketAPI) UpdateInAddressBook(ctx context.Context, arg *UpdateInAddressBookRequest) (*UpdateInAddressBookResponse, error) {
	return w.e.updateInAddressBook(ctx, arg, true)
}

// Withdraw withdraws funds as Exchange.Withdraw does, over the websocket
func (w WebsocketAPI) Withdraw(ctx context.Context, arg *WithdrawRequest) (*WithdrawalResponse, error) {
	return w.e.withdraw(ctx, arg, true)
}
