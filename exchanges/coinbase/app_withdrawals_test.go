package coinbase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// testAppWithdrawalTransfer returns the docs' withdrawal transfer, whose commit holds a user reference
func testAppWithdrawalTransfer(userReference string) *AppTransfer {
	return &AppTransfer{
		UserEnteredAmount: Amount{Value: 9, Currency: currency.USD},
		Amount:            Amount{Value: 9, Currency: currency.USD},
		Total:             Amount{Value: 9, Currency: currency.USD},
		Subtotal:          Amount{Value: 9, Currency: currency.USD},
		IdempotencyKey:    "97c0dcf2-daf7-446c-8dcc-66a9b72b6391",
		ID:                "97c0dcf2-daf7-446c-8dcc-66a9b72b6391",
		Source: AppTransferParty{
			Type:    "LEDGER_ACCOUNT",
			Network: "internal_retail",
			LedgerAccount: &AppLedgerAccount{
				AccountID: "6c770048-a3aa-580b-b153-2a6791649ee4",
				Currency:  currency.USD,
				Owner: AppLedgerAccountOwner{
					ID:   "5a48fda3bbf66c03a6509af2",
					Type: "RETAIL",
				},
			},
		},
		Target: AppTransferParty{
			Type:    "EXTERNAL_PAYMENT_METHOD",
			Network: "ach",
			ExternalPaymentMethod: &AppExternalPaymentMethod{
				PaymentMethodID: "5a48fe239b15170130598e9c",
			},
		},
		PayoutAt:      time.Date(2025, 4, 12, 19, 29, 48, 611016197, time.UTC),
		UserReference: userReference,
		Type:          AppTransferTypeWithdrawal,
		UserWarnings:  []json.RawMessage{},
		Fees:          []AppTransferFee{},
		TotalFee: AppTransferFee{
			Title:       "Fee Total",
			Description: "Total fee associated with this transaction",
			Amount:      Amount{Currency: currency.USD},
			Type:        "COINBASE",
		},
		CancellationReason: json.RawMessage("null"),
		NextStep:           json.RawMessage("null"),
	}
}

func TestAppWithdrawFunds(t *testing.T) {
	t.Parallel()
	testAppFiatTransferRequestErrors(t, "WithdrawFunds", e.App().WithdrawFunds)

	if !mockTests {
		t.Skip("a withdrawal needs a payment method's ID, which Advanced Trade's List Payment Methods returns, so WithdrawFunds is tested against its documented response only")
	}
	resp, err := e.App().WithdrawFunds(t.Context(), &AppFiatTransferRequest{AccountID: testAppAccountID, Amount: 10, Currency: currency.USD, PaymentMethodID: testAppPaymentMethodID})
	require.NoError(t, err, "WithdrawFunds must not error")
	assert.Equal(t, testAppWithdrawalTransfer(""), resp, "WithdrawFunds should decode every field")

	req := &AppFiatTransferRequest{AccountID: testAppAccountID, Amount: 10, Currency: currency.USD, PaymentMethodID: testAppPaymentMethodID}
	_, err = newResponseServerExchange(t, `null`).App().WithdrawFunds(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "WithdrawFunds should error for a null response")
	_, err = newUnavailableServerExchange(t).App().WithdrawFunds(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "WithdrawFunds should return the request failure")
}

func TestAppCommitWithdrawal(t *testing.T) {
	t.Parallel()
	testAppFiatTransferIDRequestErrors(t, "CommitWithdrawal", e.App().CommitWithdrawal)

	if !mockTests {
		t.Skip("a committed withdrawal cannot be undone, so CommitWithdrawal is tested against its documented response only")
	}
	resp, err := e.App().CommitWithdrawal(t.Context(), &AppFiatTransferIDRequest{AccountID: testAppAccountID, TransferID: testAppUncommittedTransferID})
	require.NoError(t, err, "CommitWithdrawal must not error")
	assert.Equal(t, testAppWithdrawalTransfer("CODE"), resp, "CommitWithdrawal should decode every field")

	req := &AppFiatTransferIDRequest{AccountID: testAppAccountID, TransferID: testAppUncommittedTransferID}
	_, err = newResponseServerExchange(t, `null`).App().CommitWithdrawal(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CommitWithdrawal should error for a null response")
	_, err = newUnavailableServerExchange(t).App().CommitWithdrawal(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "CommitWithdrawal should return the request failure")
}

func TestAppListWithdrawals(t *testing.T) {
	t.Parallel()
	testAppListFiatTransfersRequestErrors(t, "ListWithdrawals", e.App().ListWithdrawals)

	skipLiveWithoutCredentials(t)
	resp, err := e.App().ListWithdrawals(t.Context(), &AppListFiatTransfersRequest{AccountID: testAppAccount(t)})
	require.NoError(t, err, "ListWithdrawals must not error")
	if mockTests {
		exp := &AppListResponse[AppFiatTransferResponse]{
			Data:       []AppFiatTransferResponse{testAppFiatTransfer(t, "withdrawal")},
			Pagination: AppPagination{Limit: 25, Order: AppOrderDescending},
		}
		assert.Equal(t, exp, resp, "ListWithdrawals should decode every field")
	} else {
		for i := range resp.Data {
			assert.Equalf(t, "withdrawal", resp.Data[i].Resource, "ListWithdrawals should return withdrawal %d as a withdrawal", i)
		}
	}

	_, err = newUnavailableServerExchange(t).App().ListWithdrawals(t.Context(), &AppListFiatTransfersRequest{AccountID: testAppAccountID})
	assert.ErrorIs(t, err, request.ErrBadStatus, "ListWithdrawals should return the request failure")
}

func TestAppGetWithdrawal(t *testing.T) {
	t.Parallel()
	testAppFiatTransferIDRequestErrors(t, "GetWithdrawal", e.App().GetWithdrawal)

	skipLiveWithoutCredentials(t)
	req := &AppFiatTransferIDRequest{AccountID: testAppAccountID, TransferID: testAppFiatTransferID}
	if !mockTests {
		req.AccountID = testAppAccount(t)
		withdrawals, err := e.App().ListWithdrawals(t.Context(), &AppListFiatTransfersRequest{AccountID: req.AccountID, Page: &AppPageRequest{Limit: 1}})
		require.NoError(t, err, "ListWithdrawals must not error")
		if len(withdrawals.Data) == 0 {
			t.Skip("the account has no withdrawals")
		}
		req.TransferID = withdrawals.Data[0].ID
	}
	resp, err := e.App().GetWithdrawal(t.Context(), req)
	require.NoError(t, err, "GetWithdrawal must not error")
	if mockTests {
		exp := testAppFiatTransfer(t, "withdrawal")
		assert.Equal(t, &exp, resp, "GetWithdrawal should decode every field")
	} else {
		assert.Equal(t, req.TransferID, resp.ID, "GetWithdrawal should return the withdrawal")
	}

	_, err = newResponseServerExchange(t, `null`).App().GetWithdrawal(t.Context(), &AppFiatTransferIDRequest{AccountID: testAppAccountID, TransferID: testAppFiatTransferID})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetWithdrawal should error for a null response")
	_, err = newUnavailableServerExchange(t).App().GetWithdrawal(t.Context(), &AppFiatTransferIDRequest{AccountID: testAppAccountID, TransferID: testAppFiatTransferID})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetWithdrawal should return the request failure")
}
