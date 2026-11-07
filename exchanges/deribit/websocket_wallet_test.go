package deribit

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

func TestWebsocketAPIAddToAddressBook(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("Deribit asks for a security key to add a withdrawal address, which live tests cannot give")
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().AddToAddressBook(t.Context(), testLegacyAddToAddressBookRequest())
	require.NoError(t, err, "WebsocketAPI.AddToAddressBook must not error")
	assert.Equal(t, testLegacyAddressBookEntry(), result, "WebsocketAPI.AddToAddressBook should decode every field")

	_, err = newTestExchange(t).WebsocketAPI().AddToAddressBook(t.Context(), testLegacyAddToAddressBookRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.AddToAddressBook should error without an authenticated connection")
}

func TestWebsocketAPICancelTransferByID(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("WebsocketAPI.CancelTransferByID needs a pending transfer, which only a transfer to another user, which live tests cannot undo, leaves")
	}
	ex, _ := newWebsocketTestExchange(t)
	arg := &CancelTransferByIDRequest{Currency: currency.BTC, ID: mockTransferID}
	result, err := ex.WebsocketAPI().CancelTransferByID(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.CancelTransferByID must not error")
	assert.Equal(t, testTransfer(), result, "WebsocketAPI.CancelTransferByID should decode every field")

	_, err = newTestExchange(t).WebsocketAPI().CancelTransferByID(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.CancelTransferByID should error without an authenticated connection")
}

func TestWebsocketAPICancelWithdrawal(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("WebsocketAPI.CancelWithdrawal needs a pending withdrawal, which only a withdrawal, which live tests cannot undo, leaves")
	}
	ex, _ := newWebsocketTestExchange(t)
	arg := &CancelWithdrawalRequest{Currency: currency.BTC, ID: mockWithdrawalID}
	result, err := ex.WebsocketAPI().CancelWithdrawal(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.CancelWithdrawal must not error")
	assert.Equal(t, testLegacyWithdrawal(), result, "WebsocketAPI.CancelWithdrawal should decode every field")

	_, err = newTestExchange(t).WebsocketAPI().CancelWithdrawal(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.CancelWithdrawal should error without an authenticated connection")
}

func TestWebsocketAPICoinbaseEstimateWithdrawalFee(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	if !mockTests {
		t.Skip("WebsocketAPI.CoinbaseEstimateWithdrawalFee needs an institutional Coinbase wallet type account")
	}
	ex, _ := newWebsocketTestExchange(t)
	arg := &CoinbaseEstimateWithdrawalFeeRequest{Currency: currency.BTC, NetworkName: mockBitcoinNetwork}
	result, err := ex.WebsocketAPI().CoinbaseEstimateWithdrawalFee(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.CoinbaseEstimateWithdrawalFee must not error")
	assert.Equal(t, testWithdrawalFeeEstimate(t), result, "WebsocketAPI.CoinbaseEstimateWithdrawalFee should decode every field")

	_, err = newTestExchange(t).WebsocketAPI().CoinbaseEstimateWithdrawalFee(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.CoinbaseEstimateWithdrawalFee should error without an authenticated connection")
}

func TestWebsocketAPICreateDepositAddress(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	ex, _ := newWebsocketTestExchange(t)
	arg := &CreateDepositAddressRequest{Currency: currency.BTC}
	result, err := ex.WebsocketAPI().CreateDepositAddress(t.Context(), arg)
	if !mockTests && errors.Is(err, common.ErrNoResponse) {
		t.Skip("Deribit creates at most one BTC deposit address a day")
	}
	require.NoError(t, err, "WebsocketAPI.CreateDepositAddress must not error")
	if mockTests {
		assert.Equal(t, testLegacyDepositAddress(), result, "WebsocketAPI.CreateDepositAddress should decode every field")
	} else {
		assert.Equal(t, currency.BTC, result.Currency, "WebsocketAPI.CreateDepositAddress should return a BTC address")
	}

	_, err = newTestExchange(t).WebsocketAPI().CreateDepositAddress(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.CreateDepositAddress should error without an authenticated connection")
}

func TestWebsocketAPIDeleteAddressBeneficiary(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("WebsocketAPI.DeleteAddressBeneficiary would delete an address's travel rule information, which a live test cannot restore")
	}
	ex, _ := newWebsocketTestExchange(t)
	arg := &AddressBeneficiaryRequest{Currency: currency.BTC, Address: mockLegacyAddress}
	require.NoError(t, ex.WebsocketAPI().DeleteAddressBeneficiary(t.Context(), arg), "WebsocketAPI.DeleteAddressBeneficiary must not error")

	err := newTestExchange(t).WebsocketAPI().DeleteAddressBeneficiary(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.DeleteAddressBeneficiary should error without an authenticated connection")
}

func TestWebsocketAPIGetAddressBeneficiary(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	arg := testAddressBeneficiaryRequest(t)
	result, err := ex.WebsocketAPI().GetAddressBeneficiary(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetAddressBeneficiary must not error")
	if mockTests {
		assert.Equal(t, testAddressBeneficiary(), result, "WebsocketAPI.GetAddressBeneficiary should decode every field")
	} else {
		assert.Equal(t, arg.Address, result.Address, "WebsocketAPI.GetAddressBeneficiary should return the address's beneficiary")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetAddressBeneficiary(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetAddressBeneficiary should error without an authenticated connection")
}

func TestWebsocketAPIGetAddressBook(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	arg := &GetAddressBookRequest{Currency: currency.BTC, Type: AddressBookTypeWithdrawal}
	result, err := ex.WebsocketAPI().GetAddressBook(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetAddressBook must not error")
	if mockTests {
		assert.Equal(t, testLegacyAddressBook(), result, "WebsocketAPI.GetAddressBook should decode every field")
	} else {
		for i := range result {
			assert.Equal(t, AddressBookTypeWithdrawal, result[i].Type, "WebsocketAPI.GetAddressBook should return withdrawal entries")
		}
	}

	_, err = newTestExchange(t).WebsocketAPI().GetAddressBook(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetAddressBook should error without an authenticated connection")
}

func TestWebsocketAPIGetCurrentDepositAddress(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	arg := &GetCurrentDepositAddressRequest{Currency: currency.BTC}
	result, err := ex.WebsocketAPI().GetCurrentDepositAddress(t.Context(), arg)
	if !mockTests && errors.Is(err, common.ErrNoResponse) {
		t.Skip("the account has no BTC deposit address")
	}
	require.NoError(t, err, "WebsocketAPI.GetCurrentDepositAddress must not error")
	if mockTests {
		assert.Equal(t, testLegacyDepositAddress(), result, "WebsocketAPI.GetCurrentDepositAddress should decode every field")
	} else {
		assert.Equal(t, currency.BTC, result.Currency, "WebsocketAPI.GetCurrentDepositAddress should return a BTC address")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetCurrentDepositAddress(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetCurrentDepositAddress should error without an authenticated connection")
}

func TestWebsocketAPIGetDeposits(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetDeposits(t.Context(), testWalletHistoryRequest())
	require.NoError(t, err, "WebsocketAPI.GetDeposits must not error")
	if mockTests {
		assert.Equal(t, testDeposits(), result, "WebsocketAPI.GetDeposits should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Deposits), 10, "WebsocketAPI.GetDeposits should return at most the requested count")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetDeposits(t.Context(), testWalletHistoryRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetDeposits should error without an authenticated connection")
}

func TestWebsocketAPIGetRewardEligibility(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetRewardEligibility(t.Context())
	require.NoError(t, err, "WebsocketAPI.GetRewardEligibility must not error")
	if mockTests {
		assert.Equal(t, testRewardEligibility(), result, "WebsocketAPI.GetRewardEligibility should decode every currency's eligibility")
	} else {
		assert.NotEmpty(t, result, "WebsocketAPI.GetRewardEligibility should return the reward bearing currencies")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetRewardEligibility(t.Context())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetRewardEligibility should error without an authenticated connection")
}

func TestWebsocketAPIGetTransfers(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetTransfers(t.Context(), testWalletHistoryRequest())
	require.NoError(t, err, "WebsocketAPI.GetTransfers must not error")
	if mockTests {
		assert.Equal(t, testTransfers(), result, "WebsocketAPI.GetTransfers should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Transfers), 10, "WebsocketAPI.GetTransfers should return at most the requested count")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetTransfers(t.Context(), testWalletHistoryRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetTransfers should error without an authenticated connection")
}

func TestWebsocketAPIGetWithdrawals(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetWithdrawals(t.Context(), testWalletHistoryRequest())
	require.NoError(t, err, "WebsocketAPI.GetWithdrawals must not error")
	if mockTests {
		assert.Equal(t, testWithdrawals(), result, "WebsocketAPI.GetWithdrawals should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Withdrawals), 10, "WebsocketAPI.GetWithdrawals should return at most the requested count")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetWithdrawals(t.Context(), testWalletHistoryRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetWithdrawals should error without an authenticated connection")
}

func TestWebsocketAPIListAddressBeneficiaries(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().ListAddressBeneficiaries(t.Context(), testListAddressBeneficiariesRequest())
	require.NoError(t, err, "WebsocketAPI.ListAddressBeneficiaries must not error")
	if mockTests {
		assert.Equal(t, testAddressBeneficiaries(), result, "WebsocketAPI.ListAddressBeneficiaries should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Beneficiaries), 10, "WebsocketAPI.ListAddressBeneficiaries should return at most the requested limit")
	}

	_, err = newTestExchange(t).WebsocketAPI().ListAddressBeneficiaries(t.Context(), testListAddressBeneficiariesRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.ListAddressBeneficiaries should error without an authenticated connection")
}

func TestWebsocketAPIRemoveFromAddressBook(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("WebsocketAPI.RemoveFromAddressBook would remove an entry, which adding again restarts the address delay of")
	}
	ex, _ := newWebsocketTestExchange(t)
	arg := &RemoveFromAddressBookRequest{Currency: currency.BTC, Type: AddressBookTypeTransfer, Address: mockLegacyAddress}
	require.NoError(t, ex.WebsocketAPI().RemoveFromAddressBook(t.Context(), arg), "WebsocketAPI.RemoveFromAddressBook must not error")

	err := newTestExchange(t).WebsocketAPI().RemoveFromAddressBook(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.RemoveFromAddressBook should error without an authenticated connection")
}

func TestWebsocketAPISaveAddressBeneficiary(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("WebsocketAPI.SaveAddressBeneficiary would overwrite an address's travel rule information, which a live test cannot restore")
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().SaveAddressBeneficiary(t.Context(), testSaveAddressBeneficiaryRequest())
	require.NoError(t, err, "WebsocketAPI.SaveAddressBeneficiary must not error")
	assert.Equal(t, testAddressBeneficiary(), result, "WebsocketAPI.SaveAddressBeneficiary should decode every field")

	_, err = newTestExchange(t).WebsocketAPI().SaveAddressBeneficiary(t.Context(), testSaveAddressBeneficiaryRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.SaveAddressBeneficiary should error without an authenticated connection")
}

func TestWebsocketAPISetAddressRequiresSecurityKey(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("Deribit asks for a security key to change a withdrawal address, which live tests cannot give")
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().SetAddressRequiresSecurityKey(t.Context(), testLegacySecurityKeyRequest())
	require.NoError(t, err, "WebsocketAPI.SetAddressRequiresSecurityKey must not error")
	assert.Equal(t, testLegacySecurityKeyEntry(), result, "WebsocketAPI.SetAddressRequiresSecurityKey should decode every field")

	_, err = newTestExchange(t).WebsocketAPI().SetAddressRequiresSecurityKey(t.Context(), testLegacySecurityKeyRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.SetAddressRequiresSecurityKey should error without an authenticated connection")
}

func TestWebsocketAPISetClearanceOriginator(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("WebsocketAPI.SetClearanceOriginator needs a deposit awaiting its originator, which live tests do not have")
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().SetClearanceOriginator(t.Context(), testSetClearanceOriginatorRequest())
	require.NoError(t, err, "WebsocketAPI.SetClearanceOriginator must not error")
	assert.Equal(t, testClearedDeposit(), result, "WebsocketAPI.SetClearanceOriginator should decode every field")

	_, err = newTestExchange(t).WebsocketAPI().SetClearanceOriginator(t.Context(), testSetClearanceOriginatorRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.SetClearanceOriginator should error without an authenticated connection")
}

func TestWebsocketAPISetCoinbaseSubaccountWithdrawalsAllowed(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("WebsocketAPI.SetCoinbaseSubaccountWithdrawalsAllowed needs a Coinbase wallet type main account")
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().SetCoinbaseSubaccountWithdrawalsAllowed(t.Context(), true)
	require.NoError(t, err, "WebsocketAPI.SetCoinbaseSubaccountWithdrawalsAllowed must not error")
	assert.Equal(t, &CoinbaseSubaccountWithdrawalsResponse{Enabled: true}, result, "WebsocketAPI.SetCoinbaseSubaccountWithdrawalsAllowed should decode the setting")

	_, err = newTestExchange(t).WebsocketAPI().SetCoinbaseSubaccountWithdrawalsAllowed(t.Context(), true)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.SetCoinbaseSubaccountWithdrawalsAllowed should error without an authenticated connection")
}

func TestWebsocketAPISubmitTransferBetweenSubaccounts(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("WebsocketAPI.SubmitTransferBetweenSubaccounts needs subaccounts, which live tests do not have")
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().SubmitTransferBetweenSubaccounts(t.Context(), testSubmitTransferBetweenSubaccountsRequest())
	require.NoError(t, err, "WebsocketAPI.SubmitTransferBetweenSubaccounts must not error")
	assert.Equal(t, testTransferBetweenSubaccounts(), result, "WebsocketAPI.SubmitTransferBetweenSubaccounts should decode every field")

	_, err = newTestExchange(t).WebsocketAPI().SubmitTransferBetweenSubaccounts(t.Context(), testSubmitTransferBetweenSubaccountsRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.SubmitTransferBetweenSubaccounts should error without an authenticated connection")
}

func TestWebsocketAPISubmitTransferToSubaccount(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("WebsocketAPI.SubmitTransferToSubaccount needs a subaccount, which live tests do not have")
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().SubmitTransferToSubaccount(t.Context(), testSubmitTransferToSubaccountRequest())
	require.NoError(t, err, "WebsocketAPI.SubmitTransferToSubaccount must not error")
	assert.Equal(t, testTransfer(), result, "WebsocketAPI.SubmitTransferToSubaccount should decode every field")

	_, err = newTestExchange(t).WebsocketAPI().SubmitTransferToSubaccount(t.Context(), testSubmitTransferToSubaccountRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.SubmitTransferToSubaccount should error without an authenticated connection")
}

func TestWebsocketAPISubmitTransferToUser(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("WebsocketAPI.SubmitTransferToUser moves funds to another user, which a live test cannot undo")
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().SubmitTransferToUser(t.Context(), testSubmitTransferToUserRequest())
	require.NoError(t, err, "WebsocketAPI.SubmitTransferToUser must not error")
	assert.Equal(t, testTransfer(), result, "WebsocketAPI.SubmitTransferToUser should decode every field")

	_, err = newTestExchange(t).WebsocketAPI().SubmitTransferToUser(t.Context(), testSubmitTransferToUserRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.SubmitTransferToUser should error without an authenticated connection")
}

func TestWebsocketAPIUpdateInAddressBook(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("Deribit asks for a security key to update a withdrawal address, which live tests cannot give")
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().UpdateInAddressBook(t.Context(), testLegacyUpdateInAddressBookRequest())
	require.NoError(t, err, "WebsocketAPI.UpdateInAddressBook must not error")
	assert.Equal(t, &UpdateInAddressBookResponse{}, result, "WebsocketAPI.UpdateInAddressBook should accept a legacy wallet type acknowledgement")

	_, err = newTestExchange(t).WebsocketAPI().UpdateInAddressBook(t.Context(), testLegacyUpdateInAddressBookRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.UpdateInAddressBook should error without an authenticated connection")
}

func TestWebsocketAPIWithdraw(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("WebsocketAPI.Withdraw moves funds off Deribit, which a live test cannot undo")
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().Withdraw(t.Context(), testLegacyWithdrawRequest())
	require.NoError(t, err, "WebsocketAPI.Withdraw must not error")
	assert.Equal(t, testRequestedLegacyWithdrawal(), result, "WebsocketAPI.Withdraw should decode every field")

	_, err = newTestExchange(t).WebsocketAPI().Withdraw(t.Context(), testLegacyWithdrawRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.Withdraw should error without an authenticated connection")
}
