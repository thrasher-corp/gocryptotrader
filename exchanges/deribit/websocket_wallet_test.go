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
	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		_, err := ex.WebsocketAPI().AddToAddressBook(t.Context(), &AddToAddressBookRequest{Currency: currency.BTC, Type: AddressBookTypeWithdrawal, Address: liveForeignAddress, Label: "gct live test"})
		requireRefusedOnMerits(t, err, "WebsocketAPI.AddToAddressBook")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().AddToAddressBook(t.Context(), testLegacyAddToAddressBookRequest())
		require.NoError(t, err, "WebsocketAPI.AddToAddressBook must not error")
		assert.Equal(t, testLegacyAddressBookEntry(), result, "WebsocketAPI.AddToAddressBook should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().AddToAddressBook(t.Context(), testLegacyAddToAddressBookRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.AddToAddressBook should error without an authenticated connection")
}

func TestWebsocketAPICancelTransferByID(t *testing.T) {
	t.Parallel()
	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		_, err := ex.WebsocketAPI().CancelTransferByID(t.Context(), &CancelTransferByIDRequest{Currency: currency.BTC, ID: liveForeignID})
		requireRefusedOnMerits(t, err, "WebsocketAPI.CancelTransferByID")
	})

	arg := &CancelTransferByIDRequest{Currency: currency.BTC, ID: mockTransferID}
	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().CancelTransferByID(t.Context(), arg)
		require.NoError(t, err, "WebsocketAPI.CancelTransferByID must not error")
		assert.Equal(t, testTransfer(), result, "WebsocketAPI.CancelTransferByID should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().CancelTransferByID(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.CancelTransferByID should error without an authenticated connection")
}

func TestWebsocketAPICancelWithdrawal(t *testing.T) {
	t.Parallel()
	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		_, err := ex.WebsocketAPI().CancelWithdrawal(t.Context(), &CancelWithdrawalRequest{Currency: currency.BTC, ID: liveForeignID})
		requireRefusedOnMerits(t, err, "WebsocketAPI.CancelWithdrawal")
	})

	arg := &CancelWithdrawalRequest{Currency: currency.BTC, ID: mockWithdrawalID}
	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().CancelWithdrawal(t.Context(), arg)
		require.NoError(t, err, "WebsocketAPI.CancelWithdrawal must not error")
		assert.Equal(t, testLegacyWithdrawal(), result, "WebsocketAPI.CancelWithdrawal should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().CancelWithdrawal(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.CancelWithdrawal should error without an authenticated connection")
}

func TestWebsocketAPICoinbaseEstimateWithdrawalFee(t *testing.T) {
	t.Parallel()
	arg := &CoinbaseEstimateWithdrawalFeeRequest{Currency: currency.BTC, NetworkName: mockBitcoinNetwork}
	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().CoinbaseEstimateWithdrawalFee(t.Context(), arg)
		if requireAnsweredOnMerits(t, err, "WebsocketAPI.CoinbaseEstimateWithdrawalFee") {
			assert.NotNil(t, result, "WebsocketAPI.CoinbaseEstimateWithdrawalFee should return the estimate")
		}
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().CoinbaseEstimateWithdrawalFee(t.Context(), arg)
		require.NoError(t, err, "WebsocketAPI.CoinbaseEstimateWithdrawalFee must not error")
		assert.Equal(t, testWithdrawalFeeEstimate(t), result, "WebsocketAPI.CoinbaseEstimateWithdrawalFee should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().CoinbaseEstimateWithdrawalFee(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.CoinbaseEstimateWithdrawalFee should error without an authenticated connection")
}

func TestWebsocketAPICreateDepositAddress(t *testing.T) {
	t.Parallel()
	arg := &CreateDepositAddressRequest{Currency: currency.BTC}

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().CreateDepositAddress(t.Context(), arg)
		if errors.Is(err, common.ErrNoResponse) {
			t.Skip("Deribit creates at most one BTC deposit address a day")
		}
		require.NoError(t, err, "WebsocketAPI.CreateDepositAddress must not error")
		assert.Equal(t, currency.BTC, result.Currency, "WebsocketAPI.CreateDepositAddress should return a BTC address")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().CreateDepositAddress(t.Context(), arg)
		require.NoError(t, err, "WebsocketAPI.CreateDepositAddress must not error")
		assert.Equal(t, testLegacyDepositAddress(), result, "WebsocketAPI.CreateDepositAddress should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().CreateDepositAddress(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.CreateDepositAddress should error without an authenticated connection")
}

func TestWebsocketAPIDeleteAddressBeneficiary(t *testing.T) {
	t.Parallel()
	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		err := ex.WebsocketAPI().DeleteAddressBeneficiary(t.Context(), &AddressBeneficiaryRequest{Currency: currency.BTC, Address: liveForeignAddress})
		requireRefusedOnMerits(t, err, "WebsocketAPI.DeleteAddressBeneficiary")
	})

	arg := &AddressBeneficiaryRequest{Currency: currency.BTC, Address: mockLegacyAddress}
	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		require.NoError(t, ex.WebsocketAPI().DeleteAddressBeneficiary(t.Context(), arg), "WebsocketAPI.DeleteAddressBeneficiary must not error")
	})

	err := newTestExchange(t).WebsocketAPI().DeleteAddressBeneficiary(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.DeleteAddressBeneficiary should error without an authenticated connection")
}

func TestWebsocketAPIGetAddressBeneficiary(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		arg := testAddressBeneficiaryRequest(t)
		result, err := ex.WebsocketAPI().GetAddressBeneficiary(t.Context(), arg)
		require.NoError(t, err, "WebsocketAPI.GetAddressBeneficiary must not error")
		assert.Equal(t, arg.Address, result.Address, "WebsocketAPI.GetAddressBeneficiary should return the address's beneficiary")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		arg := testAddressBeneficiaryRequest(t)
		result, err := ex.WebsocketAPI().GetAddressBeneficiary(t.Context(), arg)
		require.NoError(t, err, "WebsocketAPI.GetAddressBeneficiary must not error")
		assert.Equal(t, testAddressBeneficiary(), result, "WebsocketAPI.GetAddressBeneficiary should decode every field")
	})

	arg := &AddressBeneficiaryRequest{Currency: currency.BTC, Address: mockLegacyAddress}
	_, err := newTestExchange(t).WebsocketAPI().GetAddressBeneficiary(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetAddressBeneficiary should error without an authenticated connection")
}

func TestWebsocketAPIGetAddressBook(t *testing.T) {
	t.Parallel()
	arg := &GetAddressBookRequest{Currency: currency.BTC, Type: AddressBookTypeWithdrawal}

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetAddressBook(t.Context(), arg)
		require.NoError(t, err, "WebsocketAPI.GetAddressBook must not error")
		for i := range result {
			assert.Equal(t, AddressBookTypeWithdrawal, result[i].Type, "WebsocketAPI.GetAddressBook should return withdrawal entries")
		}
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetAddressBook(t.Context(), arg)
		require.NoError(t, err, "WebsocketAPI.GetAddressBook must not error")
		assert.Equal(t, testLegacyAddressBook(), result, "WebsocketAPI.GetAddressBook should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().GetAddressBook(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetAddressBook should error without an authenticated connection")
}

func TestWebsocketAPIGetCurrentDepositAddress(t *testing.T) {
	t.Parallel()
	arg := &GetCurrentDepositAddressRequest{Currency: currency.BTC}

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetCurrentDepositAddress(t.Context(), arg)
		if errors.Is(err, common.ErrNoResponse) {
			t.Skip("the account has no BTC deposit address")
		}
		require.NoError(t, err, "WebsocketAPI.GetCurrentDepositAddress must not error")
		assert.Equal(t, currency.BTC, result.Currency, "WebsocketAPI.GetCurrentDepositAddress should return a BTC address")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetCurrentDepositAddress(t.Context(), arg)
		require.NoError(t, err, "WebsocketAPI.GetCurrentDepositAddress must not error")
		assert.Equal(t, testLegacyDepositAddress(), result, "WebsocketAPI.GetCurrentDepositAddress should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().GetCurrentDepositAddress(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetCurrentDepositAddress should error without an authenticated connection")
}

func TestWebsocketAPIGetDeposits(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetDeposits(t.Context(), testWalletHistoryRequest())
		require.NoError(t, err, "WebsocketAPI.GetDeposits must not error")
		assert.LessOrEqual(t, len(result.Deposits), 10, "WebsocketAPI.GetDeposits should return at most the requested count")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetDeposits(t.Context(), testWalletHistoryRequest())
		require.NoError(t, err, "WebsocketAPI.GetDeposits must not error")
		assert.Equal(t, testDeposits(), result, "WebsocketAPI.GetDeposits should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().GetDeposits(t.Context(), testWalletHistoryRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetDeposits should error without an authenticated connection")
}

func TestWebsocketAPIGetRewardEligibility(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetRewardEligibility(t.Context())
		require.NoError(t, err, "WebsocketAPI.GetRewardEligibility must not error")
		assert.NotEmpty(t, result, "WebsocketAPI.GetRewardEligibility should return the reward bearing currencies")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetRewardEligibility(t.Context())
		require.NoError(t, err, "WebsocketAPI.GetRewardEligibility must not error")
		assert.Equal(t, testRewardEligibility(), result, "WebsocketAPI.GetRewardEligibility should decode every currency's eligibility")
	})

	_, err := newTestExchange(t).WebsocketAPI().GetRewardEligibility(t.Context())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetRewardEligibility should error without an authenticated connection")
}

func TestWebsocketAPIGetTransfers(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetTransfers(t.Context(), testWalletHistoryRequest())
		require.NoError(t, err, "WebsocketAPI.GetTransfers must not error")
		assert.LessOrEqual(t, len(result.Transfers), 10, "WebsocketAPI.GetTransfers should return at most the requested count")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetTransfers(t.Context(), testWalletHistoryRequest())
		require.NoError(t, err, "WebsocketAPI.GetTransfers must not error")
		assert.Equal(t, testTransfers(), result, "WebsocketAPI.GetTransfers should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().GetTransfers(t.Context(), testWalletHistoryRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetTransfers should error without an authenticated connection")
}

func TestWebsocketAPIGetWithdrawals(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetWithdrawals(t.Context(), testWalletHistoryRequest())
		require.NoError(t, err, "WebsocketAPI.GetWithdrawals must not error")
		assert.LessOrEqual(t, len(result.Withdrawals), 10, "WebsocketAPI.GetWithdrawals should return at most the requested count")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetWithdrawals(t.Context(), testWalletHistoryRequest())
		require.NoError(t, err, "WebsocketAPI.GetWithdrawals must not error")
		assert.Equal(t, testWithdrawals(), result, "WebsocketAPI.GetWithdrawals should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().GetWithdrawals(t.Context(), testWalletHistoryRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetWithdrawals should error without an authenticated connection")
}

func TestWebsocketAPIListAddressBeneficiaries(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().ListAddressBeneficiaries(t.Context(), testListAddressBeneficiariesRequest())
		require.NoError(t, err, "WebsocketAPI.ListAddressBeneficiaries must not error")
		assert.LessOrEqual(t, len(result.Beneficiaries), 10, "WebsocketAPI.ListAddressBeneficiaries should return at most the requested limit")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().ListAddressBeneficiaries(t.Context(), testListAddressBeneficiariesRequest())
		require.NoError(t, err, "WebsocketAPI.ListAddressBeneficiaries must not error")
		assert.Equal(t, testAddressBeneficiaries(), result, "WebsocketAPI.ListAddressBeneficiaries should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().ListAddressBeneficiaries(t.Context(), testListAddressBeneficiariesRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.ListAddressBeneficiaries should error without an authenticated connection")
}

func TestWebsocketAPIRemoveFromAddressBook(t *testing.T) {
	t.Parallel()
	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		err := ex.WebsocketAPI().RemoveFromAddressBook(t.Context(), &RemoveFromAddressBookRequest{Currency: currency.BTC, Type: AddressBookTypeWithdrawal, Address: liveForeignAddress})
		requireRefusedOnMerits(t, err, "WebsocketAPI.RemoveFromAddressBook")
	})

	arg := &RemoveFromAddressBookRequest{Currency: currency.BTC, Type: AddressBookTypeTransfer, Address: mockLegacyAddress}
	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		require.NoError(t, ex.WebsocketAPI().RemoveFromAddressBook(t.Context(), arg), "WebsocketAPI.RemoveFromAddressBook must not error")
	})

	err := newTestExchange(t).WebsocketAPI().RemoveFromAddressBook(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.RemoveFromAddressBook should error without an authenticated connection")
}

func TestWebsocketAPISaveAddressBeneficiary(t *testing.T) {
	t.Parallel()
	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		arg := testSaveAddressBeneficiaryRequest()
		arg.Address = liveForeignAddress
		_, err := ex.WebsocketAPI().SaveAddressBeneficiary(t.Context(), arg)
		requireRefusedOnMerits(t, err, "WebsocketAPI.SaveAddressBeneficiary")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().SaveAddressBeneficiary(t.Context(), testSaveAddressBeneficiaryRequest())
		require.NoError(t, err, "WebsocketAPI.SaveAddressBeneficiary must not error")
		assert.Equal(t, testAddressBeneficiary(), result, "WebsocketAPI.SaveAddressBeneficiary should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().SaveAddressBeneficiary(t.Context(), testSaveAddressBeneficiaryRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.SaveAddressBeneficiary should error without an authenticated connection")
}

func TestWebsocketAPISetAddressRequiresSecurityKey(t *testing.T) {
	t.Parallel()
	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		_, err := ex.WebsocketAPI().SetAddressRequiresSecurityKey(t.Context(), &SetAddressRequiresSecurityKeyRequest{Currency: currency.BTC, Type: AddressBookTypeWithdrawal, Address: liveForeignAddress, RequiresSecurityKey: true})
		requireRefusedOnMerits(t, err, "WebsocketAPI.SetAddressRequiresSecurityKey")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().SetAddressRequiresSecurityKey(t.Context(), testLegacySecurityKeyRequest())
		require.NoError(t, err, "WebsocketAPI.SetAddressRequiresSecurityKey must not error")
		assert.Equal(t, testLegacySecurityKeyEntry(), result, "WebsocketAPI.SetAddressRequiresSecurityKey should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().SetAddressRequiresSecurityKey(t.Context(), testLegacySecurityKeyRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.SetAddressRequiresSecurityKey should error without an authenticated connection")
}

func TestWebsocketAPISetClearanceOriginator(t *testing.T) {
	t.Parallel()
	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		arg := testSetClearanceOriginatorRequest()
		arg.Deposit.UserID = liveForeignID
		_, err := ex.WebsocketAPI().SetClearanceOriginator(t.Context(), arg)
		requireRefusedOnMerits(t, err, "WebsocketAPI.SetClearanceOriginator")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().SetClearanceOriginator(t.Context(), testSetClearanceOriginatorRequest())
		require.NoError(t, err, "WebsocketAPI.SetClearanceOriginator must not error")
		assert.Equal(t, testClearedDeposit(), result, "WebsocketAPI.SetClearanceOriginator should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().SetClearanceOriginator(t.Context(), testSetClearanceOriginatorRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.SetClearanceOriginator should error without an authenticated connection")
}

func TestWebsocketAPISetCoinbaseSubaccountWithdrawalsAllowed(t *testing.T) {
	t.Parallel()
	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		// false is Deribit's default, as nothing reads the setting back to restore
		result, err := ex.WebsocketAPI().SetCoinbaseSubaccountWithdrawalsAllowed(t.Context(), false)
		if requireAnsweredOnMerits(t, err, "WebsocketAPI.SetCoinbaseSubaccountWithdrawalsAllowed") {
			assert.False(t, result.Enabled, "WebsocketAPI.SetCoinbaseSubaccountWithdrawalsAllowed should forbid subaccount withdrawals")
		}
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().SetCoinbaseSubaccountWithdrawalsAllowed(t.Context(), true)
		require.NoError(t, err, "WebsocketAPI.SetCoinbaseSubaccountWithdrawalsAllowed must not error")
		assert.Equal(t, &CoinbaseSubaccountWithdrawalsResponse{Enabled: true}, result, "WebsocketAPI.SetCoinbaseSubaccountWithdrawalsAllowed should decode the setting")
	})

	_, err := newTestExchange(t).WebsocketAPI().SetCoinbaseSubaccountWithdrawalsAllowed(t.Context(), true)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.SetCoinbaseSubaccountWithdrawalsAllowed should error without an authenticated connection")
}

func TestWebsocketAPISubmitTransferBetweenSubaccounts(t *testing.T) {
	t.Parallel()
	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		_, err := ex.WebsocketAPI().SubmitTransferBetweenSubaccounts(t.Context(), &SubmitTransferBetweenSubaccountsRequest{Currency: currency.BTC, Amount: 0.0001, Destination: liveForeignID})
		requireRefusedOnMerits(t, err, "WebsocketAPI.SubmitTransferBetweenSubaccounts")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().SubmitTransferBetweenSubaccounts(t.Context(), testSubmitTransferBetweenSubaccountsRequest())
		require.NoError(t, err, "WebsocketAPI.SubmitTransferBetweenSubaccounts must not error")
		assert.Equal(t, testTransferBetweenSubaccounts(), result, "WebsocketAPI.SubmitTransferBetweenSubaccounts should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().SubmitTransferBetweenSubaccounts(t.Context(), testSubmitTransferBetweenSubaccountsRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.SubmitTransferBetweenSubaccounts should error without an authenticated connection")
}

func TestWebsocketAPISubmitTransferToSubaccount(t *testing.T) {
	t.Parallel()
	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		_, err := ex.WebsocketAPI().SubmitTransferToSubaccount(t.Context(), &SubmitTransferToSubaccountRequest{Currency: currency.BTC, Amount: 0.0001, Destination: liveForeignID})
		requireRefusedOnMerits(t, err, "WebsocketAPI.SubmitTransferToSubaccount")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().SubmitTransferToSubaccount(t.Context(), testSubmitTransferToSubaccountRequest())
		require.NoError(t, err, "WebsocketAPI.SubmitTransferToSubaccount must not error")
		assert.Equal(t, testTransfer(), result, "WebsocketAPI.SubmitTransferToSubaccount should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().SubmitTransferToSubaccount(t.Context(), testSubmitTransferToSubaccountRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.SubmitTransferToSubaccount should error without an authenticated connection")
}

func TestWebsocketAPISubmitTransferToUser(t *testing.T) {
	t.Parallel()
	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		_, err := ex.WebsocketAPI().SubmitTransferToUser(t.Context(), &SubmitTransferToUserRequest{Currency: currency.BTC, Amount: 0.0001, Destination: liveForeignAddress})
		requireRefusedOnMerits(t, err, "WebsocketAPI.SubmitTransferToUser")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().SubmitTransferToUser(t.Context(), testSubmitTransferToUserRequest())
		require.NoError(t, err, "WebsocketAPI.SubmitTransferToUser must not error")
		assert.Equal(t, testTransfer(), result, "WebsocketAPI.SubmitTransferToUser should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().SubmitTransferToUser(t.Context(), testSubmitTransferToUserRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.SubmitTransferToUser should error without an authenticated connection")
}

func TestWebsocketAPIUpdateInAddressBook(t *testing.T) {
	t.Parallel()
	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		arg := testLegacyUpdateInAddressBookRequest()
		arg.Address = liveForeignAddress
		_, err := ex.WebsocketAPI().UpdateInAddressBook(t.Context(), arg)
		requireRefusedOnMerits(t, err, "WebsocketAPI.UpdateInAddressBook")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().UpdateInAddressBook(t.Context(), testLegacyUpdateInAddressBookRequest())
		require.NoError(t, err, "WebsocketAPI.UpdateInAddressBook must not error")
		assert.Equal(t, &UpdateInAddressBookResponse{}, result, "WebsocketAPI.UpdateInAddressBook should accept a legacy wallet type acknowledgement")
	})

	_, err := newTestExchange(t).WebsocketAPI().UpdateInAddressBook(t.Context(), testLegacyUpdateInAddressBookRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.UpdateInAddressBook should error without an authenticated connection")
}

func TestWebsocketAPIWithdraw(t *testing.T) {
	t.Parallel()
	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		arg := testLegacyWithdrawRequest()
		arg.Address = liveForeignAddress
		arg.Amount = 0.0001
		arg.Nonce = ""
		_, err := ex.WebsocketAPI().Withdraw(t.Context(), arg)
		requireRefusedOnMerits(t, err, "WebsocketAPI.Withdraw")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().Withdraw(t.Context(), testLegacyWithdrawRequest())
		require.NoError(t, err, "WebsocketAPI.Withdraw must not error")
		assert.Equal(t, testRequestedLegacyWithdrawal(), result, "WebsocketAPI.Withdraw should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().Withdraw(t.Context(), testLegacyWithdrawRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.Withdraw should error without an authenticated connection")
}
