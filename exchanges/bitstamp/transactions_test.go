package bitstamp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetUserTransactions(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *UserTransactionsRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &UserTransactionsRequest{Limit: 1001}, err: errInvalidLimit},
		{req: &UserTransactionsRequest{Since: time.Unix(2, 0), Until: time.Unix(1, 0)}, err: common.ErrStartAfterEnd},
	} {
		_, err := e.GetUserTransactions(t.Context(), tc.req)
		assert.ErrorIs(t, err, tc.err, "GetUserTransactions should return the correct error")
	}

	skipIfLiveWithoutCredentials(t)
	transactions, err := e.GetUserTransactions(t.Context(), &UserTransactionsRequest{Pair: spotPair, Limit: 10, Sort: SortDescending})
	require.NoError(t, err, "GetUserTransactions must not error")
	if mockTests {
		require.Len(t, transactions, 2, "GetUserTransactions must return the transactions")
		exp := UserTransactionResponse{
			ID:            50197021,
			DateTime:      time.Date(2022, 3, 1, 10, 55, 53, 0, time.UTC),
			Type:          TransactionTypeMarketTrade,
			Fee:           8.04,
			OrderID:       1458532827766784,
			Amounts:       map[currency.Code]float64{currency.USD: -2010, currency.BTC: 0.1, currency.EUR: 0},
			ExchangeRates: map[currency.Pair]float64{spotPair: 20100},
		}
		assert.Equal(t, exp, transactions[1], "GetUserTransactions should return the transaction")
	}
}

func TestGetCryptoTransactions(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *CryptoTransactionsRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &CryptoTransactionsRequest{Limit: 1001}, err: errInvalidLimit},
		{req: &CryptoTransactionsRequest{Since: time.Unix(2, 0), Until: time.Unix(1, 0)}, err: common.ErrStartAfterEnd},
	} {
		_, err := e.GetCryptoTransactions(t.Context(), tc.req)
		assert.ErrorIs(t, err, tc.err, "GetCryptoTransactions should return the correct error")
	}

	skipIfLiveWithoutCredentials(t)
	txs, err := e.GetCryptoTransactions(t.Context(), &CryptoTransactionsRequest{Limit: 10, IncludeIOUs: true})
	require.NoError(t, err, "GetCryptoTransactions must not error")
	if mockTests {
		exp := &CryptoTransactionsResponse{
			RippleIOUTransactions: []CryptoTransaction{},
			Deposits: []CryptoDeposit{{
				ID:                 1,
				Network:            "bitcoin",
				Currency:           currency.BTC,
				OriginatorAddress:  "1htZQJS6YUUPIGNNPC9425xENXQS2JwM7C",
				TxID:               "e4123d1d57df4106aaae5ec4d77eb6cd42e226d020a0a2c1c7919d14b93",
				Amount:             1.23,
				DateTime:           types.Time(time.Unix(1759995000, 0)),
				Status:             DepositStatusPending,
				PendingReason:      DepositPendingReasonAddressVerificationNeeded,
				DestinationAddress: "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
			}},
			Withdrawals: []CryptoTransaction{{
				Currency:           currency.BTC,
				Network:            "bitcoin",
				DestinationAddress: "3FiKkjgZ6Sj4RWp3ZsCjYh5Pt7ZCBsL7uF",
				Amount:             0.00012,
				DateTime:           types.Time(time.Unix(1642665114, 0)),
			}},
		}
		assert.Equal(t, exp, txs, "GetCryptoTransactions should decode the transactions")
	}
}

func TestGetCryptoDeposits(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *CryptoDepositsRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &CryptoDepositsRequest{Limit: 1001}, err: errInvalidLimit},
		{req: &CryptoDepositsRequest{Since: time.Unix(2, 0), Until: time.Unix(1, 0)}, err: common.ErrStartAfterEnd},
	} {
		_, err := e.GetCryptoDeposits(t.Context(), tc.req)
		assert.ErrorIs(t, err, tc.err, "GetCryptoDeposits should return the correct error")
	}

	skipIfLiveWithoutCredentials(t)
	deposits, err := e.GetCryptoDeposits(t.Context(), &CryptoDepositsRequest{Limit: 10, Status: "PENDING"})
	require.NoError(t, err, "GetCryptoDeposits must not error")
	if mockTests {
		require.Len(t, deposits, 1, "GetCryptoDeposits must return the deposits")
		exp := CryptoDepositResponse{
			ID:                 1,
			Network:            "bitcoin",
			Currency:           currency.BTC,
			DestinationAddress: "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
			OriginatorAddress:  "1htZQJS6YUUPIGNNPC9425xENXQS2JwM7C",
			TxID:               "e4123d1d57df4106aaae5ec4d77eb6cd42e226d020a0a2c1c7919d14b93",
			Amount:             1.23,
			DateTime:           types.Time(time.Unix(1759995000, 0)),
			Status:             "PENDING",
			PendingReason:      "ADDRESS_VERIFICATION_NEEDED",
		}
		assert.Equal(t, exp, deposits[0], "GetCryptoDeposits should return the deposit")
	}
}

func TestUpdateCryptoDepositOriginator(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, e.UpdateCryptoDepositOriginator(t.Context(), 0, &DepositOriginatorRequest{}), errDepositIDRequired, "UpdateCryptoDepositOriginator should error without a deposit ID")
	assert.ErrorIs(t, e.UpdateCryptoDepositOriginator(t.Context(), 1, nil), common.ErrNilPointer, "UpdateCryptoDepositOriginator should error on a nil request")
	assert.ErrorIs(t, e.UpdateCryptoDepositOriginator(t.Context(), 1, &DepositOriginatorRequest{OriginatorThirdParty: new(true)}), errOriginatorRequired, "UpdateCryptoDepositOriginator should error on a third party deposit without an originator")
	assert.ErrorIs(t, e.UpdateCryptoDepositOriginator(t.Context(), 1, &DepositOriginatorRequest{OriginatorInfo: &CustomerInfo{CorporateInfo: &CorporateInfo{}}}), errCompanyNameRequired, "UpdateCryptoDepositOriginator should error on an unnamed originator")
	assert.ErrorIs(t, e.UpdateCryptoDepositOriginator(t.Context(), 1, &DepositOriginatorRequest{BeneficiaryInfo: &CustomerInfo{RetailInfo: &RetailInfo{FirstName: "John", LastName: "Doe"}}}), errIdentificationRequired, "UpdateCryptoDepositOriginator should error on an unidentified beneficiary")

	if !mockTests {
		t.Skip("UpdateCryptoDepositOriginator is not tested live as it requires a pending deposit")
	}
	err := e.UpdateCryptoDepositOriginator(t.Context(), 1, &DepositOriginatorRequest{
		OriginatorThirdParty: new(true),
		OriginatorInfo:       &CustomerInfo{RetailInfo: &RetailInfo{FirstName: "John", LastName: "Doe"}},
	})
	assert.NoError(t, err, "UpdateCryptoDepositOriginator should not error")
}

func TestRejectCryptoDeposit(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, e.RejectCryptoDeposit(t.Context(), 0), errDepositIDRequired, "RejectCryptoDeposit should error without a deposit ID")

	if !mockTests {
		t.Skip("RejectCryptoDeposit is not tested live as it requires a pending deposit")
	}
	assert.NoError(t, e.RejectCryptoDeposit(t.Context(), 1), "RejectCryptoDeposit should not error")
}
