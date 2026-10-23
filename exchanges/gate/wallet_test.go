package gate

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestListCurrencyChain(t *testing.T) {
	t.Parallel()
	_, err := e.ListCurrencyChain(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "ListCurrencyChain must reject an empty currency")

	result, err := e.ListCurrencyChain(t.Context(), currency.BTC)
	require.NoError(t, err, "ListCurrencyChain must not error")
	if mockTests {
		exp := []*CurrencyChainResponse{
			{Chain: "BTC", ChineseChainName: "Bitcoin", ChainName: "Bitcoin", WithdrawalPrecision: 8},
			{Chain: "BSC", ChineseChainName: "BNB Smart Chain（BEP20）", ChainName: "BNB Smart Chain(BEP20)", ContractAddress: "0x7130d2a12b9bcbfae4f2634d864a1ee1ce3ead9c", WithdrawalPrecision: 8},
		}
		assert.Equal(t, exp, result, "ListCurrencyChain should decode every field")
		return
	}
	assert.NotEmpty(t, result, "ListCurrencyChain should return chains")
}

func TestGenerateCurrencyDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GenerateCurrencyDepositAddress(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GenerateCurrencyDepositAddress must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GenerateCurrencyDepositAddress(t.Context(), currency.BTC)
	require.NoError(t, err, "GenerateCurrencyDepositAddress must not error")
	if mockTests {
		exp := &CurrencyDepositAddressInfoResponse{
			Currency:         currency.BTC,
			Address:          "1HkxtBAMrA3tP5ENnYY2CZortjZvFDH5Cs",
			MinDepositAmount: 0.000006,
			MultichainAddresses: []*MultiChainAddressItem{
				{Chain: "BTC", Address: "1HkxtBAMrA3tP5ENnYY2CZortjZvFDH5Cs", MinConfirms: 1},
				{Chain: "BSC", Address: "0x1234567890abcdef1234567890abcdef12345678", MinConfirms: 15},
				{Chain: "XLM", Address: "GD4QXLLNHVFG5KZWO3NZKZ4MTK5ZRP7L5FDDOWKB35HEF2DTJQXEYPYV", PaymentID: "1067234", PaymentName: "Memo", ObtainFailed: 1, MinConfirms: 3},
			},
		}
		assert.Equal(t, exp, result, "GenerateCurrencyDepositAddress should decode every field")
		return
	}
	assert.NotNil(t, result, "GenerateCurrencyDepositAddress should return an address")
}

func TestGetWithdrawalRecords(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetWithdrawalRecords(t.Context(), &WithdrawalRecordsRequest{Currency: currency.BTC, From: endTime, To: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetWithdrawalRecords must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWithdrawalRecords(t.Context(), &WithdrawalRecordsRequest{Currency: currency.BTC, From: startTime, To: endTime, Offset: 10, Limit: 100})
	require.NoError(t, err, "GetWithdrawalRecords must not error")
	if mockTests {
		exp := []*WithdrawalResponse{{
			ID:                "w1879219868",
			WithdrawalOrderID: "202504211521368538928",
			Timestamp:         types.Time(time.Unix(1744150000, 0)),
			FinalisedTime:     types.Time(time.Unix(1744150100, 0)),
			Currency:          currency.BTC,
			WithdrawalAddress: "1HkxtBAMrA3tP5ENnYY2CZortjZvFDH5Cs",
			TransactionID:     "Internal transaction 260594131",
			BlockNumber:       1000,
			Amount:            0.5,
			Fee:               0.0001,
			Status:            "DONE",
			BusinessType:      "appbankgp",
			Chain:             "BTC",
		}}
		assert.Equal(t, exp, result, "GetWithdrawalRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetWithdrawalRecords should return records")
}

func TestGetDepositRecords(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetDepositRecords(t.Context(), currency.BTC, endTime, startTime, 0, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetDepositRecords must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDepositRecords(t.Context(), currency.BTC, startTime, endTime, 10, 100)
	require.NoError(t, err, "GetDepositRecords must not error")
	if mockTests {
		exp := []*DepositRecordResponse{
			{
				ID:            "d123456",
				Timestamp:     types.Time(time.Unix(1744150000, 0)),
				Currency:      currency.BTC,
				Address:       "bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc",
				TransactionID: "0xabcdef",
				Amount:        10,
				Status:        "BLOCKED",
				RefundStatus:  "REFUNDING",
				Chain:         "BTC",
			},
			{
				ID:            "210496",
				Timestamp:     types.Time(time.Unix(1744160000, 0)),
				Currency:      currency.BTC,
				Address:       "1HkxtBAMrA3tP5ENnYY2CZortjZvFDH5Cs",
				TransactionID: "128988928203223323290",
				Amount:        222.61,
				Status:        "DONE",
				Chain:         "BTC",
			},
		}
		assert.Equal(t, exp, result, "GetDepositRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDepositRecords should return records")
}

func TestTransferCurrencyParamMarshalJSON(t *testing.T) {
	t.Parallel()
	payload, err := json.Marshal(&TransferCurrencyRequest{
		Currency: currency.USDT,
		From:     asset.CoinMarginedFutures,
		To:       asset.DeliveryFutures,
		Amount:   12.5,
		Settle:   currency.USDT,
	})
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"currency":"USDT","from":"futures","to":"delivery","amount":"12.5","currency_pair":"","settle":"usdt"}`, string(payload), "TransferCurrencyRequest should send Gate account names")

	_, err = json.Marshal(&TransferCurrencyRequest{From: asset.Spot, To: asset.CrossMargin})
	assert.ErrorIs(t, err, asset.ErrNotSupported, "TransferCurrencyRequest should reject an account Gate does not support")
}

func TestTransferCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.TransferCurrency(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "TransferCurrency must reject a nil request")

	arg := &TransferCurrencyRequest{}
	_, err = e.TransferCurrency(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "TransferCurrency must reject an empty currency")

	arg.Currency = currency.BTC
	_, err = e.TransferCurrency(t.Context(), arg)
	require.ErrorIs(t, err, asset.ErrInvalidAsset, "TransferCurrency must reject an empty source account")

	arg.From = asset.Margin
	_, err = e.TransferCurrency(t.Context(), arg)
	require.ErrorIs(t, err, asset.ErrInvalidAsset, "TransferCurrency must reject an empty destination account")

	arg.To = asset.CrossMargin
	_, err = e.TransferCurrency(t.Context(), arg)
	require.ErrorIs(t, err, asset.ErrNotSupported, "TransferCurrency must reject an account Gate does not support")

	arg.To = asset.Futures
	_, err = e.TransferCurrency(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "TransferCurrency must require a pair for margin transfers")

	arg.CurrencyPair = getPair(t, asset.Margin)
	_, err = e.TransferCurrency(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "TransferCurrency must require a settlement currency for futures transfers")

	arg.To = asset.DeliveryFutures
	_, err = e.TransferCurrency(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "TransferCurrency must require a settlement currency for delivery transfers")

	arg.Settle = currency.USDT
	_, err = e.TransferCurrency(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "TransferCurrency must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.TransferCurrency(t.Context(), &TransferCurrencyRequest{
		Currency:     currency.BTC,
		From:         asset.Spot,
		To:           asset.Margin,
		Amount:       1202.000,
		CurrencyPair: getPair(t, asset.Spot),
	})
	require.NoError(t, err, "TransferCurrency must not error for a margin transfer")
	if mockTests {
		assert.Equal(t, &TransactionIDResponse{TransactionID: 59636381286}, result, "TransferCurrency should decode the transaction ID")
	}

	// Perpetual futures assets must be sent as Gate's futures account rather than their asset names
	result, err = e.TransferCurrency(t.Context(), &TransferCurrencyRequest{
		Currency: currency.USDT,
		Settle:   currency.USDT,
		From:     asset.Spot,
		To:       asset.USDTMarginedFutures,
		Amount:   1202.000,
	})
	require.NoError(t, err, "TransferCurrency must not error for a futures transfer")
	if mockTests {
		assert.Equal(t, &TransactionIDResponse{TransactionID: 59636381287}, result, "TransferCurrency should decode the transaction ID")
	}
}

func TestGetTradingAccountTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradingAccountTransfer(t.Context(), "")
	require.ErrorIs(t, err, errTransactionIDRequired, "GetTradingAccountTransfer must reject an empty transaction ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTradingAccountTransfer(t.Context(), "59636381286")
	require.NoError(t, err, "GetTradingAccountTransfer must not error")
	if mockTests {
		exp := &TradingAccountTransferResponse{
			TransactionID: "59636381286",
			Status:        "success",
			Currency:      currency.USDT,
			Amount:        10,
			FromAccount:   "spot",
			ToAccount:     "futures",
			Settle:        currency.NewCode("usdt"),
		}
		assert.Equal(t, exp, result, "GetTradingAccountTransfer should decode every field")
		return
	}
	assert.NotNil(t, result, "GetTradingAccountTransfer should return a transfer")
}

func TestSubAccountTransfer(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	_, err := e.SubAccountTransfer(ctx, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SubAccountTransfer must reject a nil request")
	req := &SubAccountTransferRequest{SubAccountType: "index"}
	_, err = e.SubAccountTransfer(ctx, req)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SubAccountTransfer must reject an empty currency")
	req.Currency = currency.BTC
	_, err = e.SubAccountTransfer(ctx, req)
	require.ErrorIs(t, err, errInvalidSubAccount, "SubAccountTransfer must reject an empty sub-account")
	req.SubAccount = "1337"
	_, err = e.SubAccountTransfer(ctx, req)
	require.ErrorIs(t, err, errInvalidTransferDirection, "SubAccountTransfer must reject an empty direction")
	req.Direction = "to"
	_, err = e.SubAccountTransfer(ctx, req)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "SubAccountTransfer must reject a zero amount")
	req.Amount = 1.337
	_, err = e.SubAccountTransfer(ctx, req)
	require.ErrorIs(t, err, asset.ErrNotSupported, "SubAccountTransfer must reject an unsupported sub-account type")
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	req.SubAccountType = "spot"
	result, err := e.SubAccountTransfer(ctx, req)
	require.NoError(t, err, "SubAccountTransfer must not error")
	if mockTests {
		assert.Equal(t, &TransactionIDResponse{TransactionID: 59636381288}, result, "SubAccountTransfer should decode the transaction ID")
	}
}

func TestGetSubAccountTransferHistory(t *testing.T) {
	t.Parallel()

	from := time.Date(2024, time.March, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	earliest := time.Date(2020, time.April, 10, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name          string
		from          time.Time
		to            time.Time
		offset        uint64
		limit         uint64
		allAccounts   bool
		expectedQuery url.Values
		expectedErr   error
	}{
		{
			name:          "unset range keeps the exchange default window",
			expectedQuery: url.Values{"sub_uid": {"1337"}},
		},
		{
			name:          "unset sub-account includes all accounts",
			allAccounts:   true,
			expectedQuery: url.Values{},
		},
		{
			name:          "full range and pagination are forwarded",
			from:          from,
			to:            to,
			offset:        5,
			limit:         10,
			expectedQuery: url.Values{"sub_uid": {"1337"}, "from": {strconv.FormatInt(from.Unix(), 10)}, "to": {strconv.FormatInt(to.Unix(), 10)}, "offset": {"5"}, "limit": {"10"}},
		},
		{
			name:          "start without end is forwarded",
			from:          from,
			expectedQuery: url.Values{"sub_uid": {"1337"}, "from": {strconv.FormatInt(from.Unix(), 10)}},
		},
		{
			name:          "end without start is forwarded",
			to:            to,
			expectedQuery: url.Values{"sub_uid": {"1337"}, "to": {strconv.FormatInt(to.Unix(), 10)}},
		},
		{
			name:          "earliest time is accepted",
			from:          earliest,
			to:            earliest.Add(time.Hour),
			expectedQuery: url.Values{"sub_uid": {"1337"}, "from": {strconv.FormatInt(earliest.Unix(), 10)}, "to": {strconv.FormatInt(earliest.Add(time.Hour).Unix(), 10)}},
		},
		{
			name:          "30 day range is accepted",
			from:          from,
			to:            from.Add(30 * 24 * time.Hour),
			expectedQuery: url.Values{"sub_uid": {"1337"}, "from": {strconv.FormatInt(from.Unix(), 10)}, "to": {strconv.FormatInt(from.Add(30*24*time.Hour).Unix(), 10)}},
		},
		{
			name:          "30 day range with fractional seconds is accepted",
			from:          from.Add(100 * time.Millisecond),
			to:            from.Add(30*24*time.Hour + 500*time.Millisecond),
			expectedQuery: url.Values{"sub_uid": {"1337"}, "from": {strconv.FormatInt(from.Unix(), 10)}, "to": {strconv.FormatInt(from.Add(30*24*time.Hour).Unix(), 10)}},
		},
		{
			name:        "start after end is rejected",
			from:        to,
			to:          from,
			expectedErr: common.ErrStartAfterEnd,
		},
		{
			name:        "range over 30 days is rejected",
			from:        from,
			to:          from.Add(30*24*time.Hour + time.Second),
			expectedErr: errSubAccountTransferHistoryRange,
		},
		{
			name:        "range over 30 days with fractional seconds is rejected",
			from:        from.Add(900 * time.Millisecond),
			to:          from.Add(30*24*time.Hour + 1100*time.Millisecond),
			expectedErr: errSubAccountTransferHistoryRange,
		},
		{
			name:        "start before the earliest available record is rejected",
			from:        earliest.Add(-time.Second),
			to:          to,
			expectedErr: errSubAccountTransferHistoryStart,
		},
		{
			name:        "start before the earliest available record without an end is rejected",
			from:        earliest.Add(-time.Second),
			expectedErr: errSubAccountTransferHistoryStart,
		},
		{
			name:        "start before the earliest available record is reported before start after end",
			from:        earliest.Add(-time.Hour),
			to:          earliest.Add(-2 * time.Hour),
			expectedErr: errSubAccountTransferHistoryStart,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var requests atomic.Int64
			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				assert.Equal(t, http.MethodGet, r.Method, "transfer history request method should be GET")
				assert.Equal(t, "/api/v4/wallet/sub_account_transfers", r.URL.Path, "transfer history request path should match the endpoint")
				assert.Equal(t, tc.expectedQuery, r.URL.Query(), "query parameters should match the requested transfer history")
				_, err := w.Write([]byte(`[{"timest":"1709251200","uid":"10001","sub_account":"1337","sub_account_type":"spot","currency":"BTC","amount":"1.5","direction":"to","source":"web"}]`))
				assert.NoError(t, err, "Mocked transfer history response should be written")
			}))

			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")
			ex.API.AuthenticatedSupport = true
			ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})

			subAccountUserID := "1337"
			if tc.allAccounts {
				subAccountUserID = ""
			}
			got, err := ex.GetSubAccountTransferHistory(t.Context(), subAccountUserID, tc.from, tc.to, tc.offset, tc.limit)
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr, "an unusable time range must be reported to the caller")
				assert.Zero(t, requests.Load(), "a request with an unusable time range should not be sent")
				return
			}
			require.NoError(t, err, "GetSubAccountTransferHistory must not error")
			assert.Equal(t, int64(1), requests.Load(), "a valid request should reach the exchange once")
			assert.Equal(t, []*SubAccountTransferResponse{{
				MainAccountUserID: "10001",
				Timestamp:         types.Time(time.Unix(1709251200, 0)),
				Source:            "web",
				Currency:          currency.BTC,
				SubAccount:        "1337",
				TransferDirection: "to",
				Amount:            1.5,
				SubAccountType:    "spot",
			}}, got, "transfer history should decode the mocked record")
		})
	}
	t.Run("exchange", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		}
		result, err := e.GetSubAccountTransferHistory(t.Context(), "", time.Time{}, time.Time{}, 0, 0)
		require.NoError(t, err, "GetSubAccountTransferHistory must not error")
		if mockTests {
			exp := []*SubAccountTransferResponse{{
				MainAccountUserID: "10001",
				Timestamp:         types.Time(time.Unix(1592809000, 0)),
				Source:            "web",
				Currency:          currency.BTC,
				SubAccount:        "10002",
				TransferDirection: "to",
				Amount:            1,
				SubAccountType:    "spot",
				ClientOrderID:     "da3ce7a088c8b0372b741419c7829033",
				Status:            "success",
			}}
			assert.Equal(t, exp, result, "GetSubAccountTransferHistory should decode every field")
			return
		}
		assert.NotNil(t, result, "GetSubAccountTransferHistory should return records")
	})
}

func TestInterSubAccountTransferParamsMarshalJSON(t *testing.T) {
	t.Parallel()
	payload, err := json.Marshal(&InterSubAccountTransferRequest{
		Currency:                currency.USDT,
		SubAccountFromUserID:    "1234",
		SubAccountFromAssetType: asset.USDTMarginedFutures,
		SubAccountToUserID:      "4567",
		SubAccountToAssetType:   asset.DeliveryFutures,
		Amount:                  3,
	})
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"currency":"USDT","sub_account_from":"1234","sub_account_from_type":"futures","sub_account_to":"4567","sub_account_to_type":"delivery","amount":"3"}`, string(payload), "InterSubAccountTransferRequest should send Gate account names")

	_, err = json.Marshal(&InterSubAccountTransferRequest{SubAccountFromAssetType: asset.Spot})
	assert.ErrorIs(t, err, asset.ErrInvalidAsset, "InterSubAccountTransferRequest should reject an empty account type")
}

func TestCheckSubAccountTransferAccountType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		asset       asset.Item
		expectedErr error
	}{
		{asset: asset.Spot},
		{asset: asset.USDTMarginedFutures},
		{asset: asset.CoinMarginedFutures},
		{asset: asset.DeliveryFutures},
		{asset: asset.Margin, expectedErr: asset.ErrNotSupported},
		{asset: asset.Options, expectedErr: asset.ErrNotSupported},
		{asset: asset.CrossMargin, expectedErr: asset.ErrNotSupported},
		{asset: asset.Empty, expectedErr: asset.ErrInvalidAsset},
	} {
		assert.ErrorIsf(t, checkSubAccountTransferAccountType(tc.asset), tc.expectedErr, "checkSubAccountTransferAccountType should return the expected error for %q", tc.asset)
	}
}

func TestSubAccountTransferToSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.SubAccountTransferToSubAccount(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SubAccountTransferToSubAccount must reject a nil request")

	arg := &InterSubAccountTransferRequest{}
	_, err = e.SubAccountTransferToSubAccount(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SubAccountTransferToSubAccount must reject an empty currency")

	arg.Currency = currency.BTC
	_, err = e.SubAccountTransferToSubAccount(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidSubAccountUserID, "SubAccountTransferToSubAccount must reject an empty source sub-account")

	arg.SubAccountFromUserID = "123456"
	_, err = e.SubAccountTransferToSubAccount(t.Context(), arg)
	require.ErrorIs(t, err, asset.ErrInvalidAsset, "SubAccountTransferToSubAccount must reject an empty source account type")

	arg.SubAccountFromAssetType = asset.Spot
	_, err = e.SubAccountTransferToSubAccount(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidSubAccountUserID, "SubAccountTransferToSubAccount must reject an empty destination sub-account")

	arg.SubAccountToUserID = "993883"
	_, err = e.SubAccountTransferToSubAccount(t.Context(), arg)
	require.ErrorIs(t, err, asset.ErrInvalidAsset, "SubAccountTransferToSubAccount must reject an empty destination account type")

	arg.SubAccountToAssetType = asset.Margin
	_, err = e.SubAccountTransferToSubAccount(t.Context(), arg)
	require.ErrorIs(t, err, asset.ErrNotSupported, "SubAccountTransferToSubAccount must reject an account type Gate does not support")

	arg.SubAccountToAssetType = asset.USDTMarginedFutures
	_, err = e.SubAccountTransferToSubAccount(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "SubAccountTransferToSubAccount must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SubAccountTransferToSubAccount(t.Context(), &InterSubAccountTransferRequest{
		Currency:                currency.BTC,
		SubAccountFromUserID:    "1234",
		SubAccountFromAssetType: asset.Spot,
		SubAccountToUserID:      "4567",
		SubAccountToAssetType:   asset.Spot,
		Amount:                  1234,
	})
	require.NoError(t, err, "SubAccountTransferToSubAccount must not error")
	if mockTests {
		assert.Equal(t, &TransactionIDResponse{TransactionID: 59636381289}, result, "SubAccountTransferToSubAccount should decode the transaction ID")
	}
}

func TestGetTransferOrderStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetTransferOrderStatus(t.Context(), "", "")
	require.ErrorIs(t, err, errTransferStatusIDRequired, "GetTransferOrderStatus must require a client order ID or transaction ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTransferOrderStatus(t.Context(), "12345678", "abcdefg")
	require.NoError(t, err, "GetTransferOrderStatus must not error")
	if mockTests {
		assert.Equal(t, &TransferStatusResponse{TransactionID: "abcdefg", Status: "SUCCESS"}, result, "GetTransferOrderStatus should decode every field")
		return
	}
	assert.NotNil(t, result, "GetTransferOrderStatus should return a status")
}

func TestGetWithdrawalStatus(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWithdrawalStatus(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err, "GetWithdrawalStatus must not error")
	if mockTests {
		exp := []*WithdrawalStatusResponse{{
			Currency:                   currency.GT,
			CurrencyName:               "GateToken",
			CurrencyNameChinese:        "GateToken",
			DepositFee:                 0,
			WithdrawPercentage:         DisplayNumber{Text: "0%"},
			FixedWithdrawalFee:         0.01,
			WithdrawDayLimit:           20000,
			WithdrawDayLimitRemain:     20000,
			WithdrawAmountMini:         0.11,
			WithdrawEachTimeLimit:      20000,
			WithdrawFixOnChains:        map[string]types.Number{"BTC": 20, "ETH": 15, "TRX": 0, "EOS": 2.5},
			WithdrawPercentageOnChains: map[string]DisplayNumber{"ETH": {Text: "0%"}, "GTEVM": {Text: "0.1%", Value: 0.1}},
		}}
		assert.Equal(t, exp, result, "GetWithdrawalStatus should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetWithdrawalStatus should return currencies")
}

func TestGetSubAccountBalances(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccountBalances(t.Context(), "", 0, 0)
	require.NoError(t, err, "GetSubAccountBalances must not error")
	if mockTests {
		exp := []*SubAccountBalancesResponse{{
			UserID:    "10003",
			Available: map[currency.Code]types.Number{currency.BTC: 0.1, currency.GT: 2000, currency.USDT: 10},
			Locking:   map[currency.Code]types.Number{currency.BTC: 0, currency.GT: 0.5, currency.USDT: 0},
		}}
		assert.Equal(t, exp, result, "GetSubAccountBalances should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSubAccountBalances should return balances")
}

func TestGetSubAccountMarginBalances(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccountMarginBalances(t.Context(), "")
	require.NoError(t, err, "GetSubAccountMarginBalances must not error")
	if mockTests {
		exp := []*SubAccountMarginBalanceResponse{{
			UserID: "10000",
			Available: []*SubAccountMarginPairBalance{{
				CurrencyPair:          currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter),
				AccountType:           "mmr",
				Leverage:              3,
				Risk:                  9999.99,
				MaintenanceMarginRate: 0.1,
				Base:                  MarginCurrencyBalance{Currency: currency.BTC, Available: 0.1},
				Quote:                 MarginCurrencyBalance{Currency: currency.USDT, Available: 5, Locked: 1, BorrowedAmount: 2, UnpaidInterest: 0.01},
			}},
		}}
		assert.Equal(t, exp, result, "GetSubAccountMarginBalances should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSubAccountMarginBalances should return balances")
}

func TestGetSubAccountFuturesBalances(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccountFuturesBalances(t.Context(), "", currency.EMPTYCODE)
	require.NoError(t, err, "GetSubAccountFuturesBalances must not error")
	if mockTests {
		exp := []*FuturesSubAccountBalanceResponse{{
			UserID: "10003",
			Available: map[string]*AccountMargin{
				"usd": {},
				"usdt": {
					User:                          10003,
					UpdateTime:                    types.Time(time.Unix(1741766400, 0)),
					UpdateID:                      37,
					Total:                         0.000000005,
					UnrealisedPNL:                 0.5,
					PositionMargin:                1,
					OrderMargin:                   2,
					Available:                     500.7,
					Point:                         3,
					Currency:                      currency.USDT,
					InDualMode:                    true,
					EnableCredit:                  true,
					PositionInitialMargin:         4,
					MaintenanceMargin:             5,
					Bonus:                         6,
					EnableEvolvedClassic:          true,
					CrossOrderMargin:              7,
					CrossInitialMargin:            8,
					CrossMaintenanceMargin:        9,
					CrossUnrealisedPNL:            10,
					CrossAvailable:                500.7,
					CrossMarginBalance:            11,
					CrossMaintenanceMarginRate:    0.01,
					CrossInitialMarginRate:        0.02,
					IsolatedPositionMargin:        12,
					EnableNewDualMode:             true,
					MarginMode:                    1,
					MarginModeName:                "multi_currency",
					EnableTieredMaintenanceMargin: true,
					EnableDualPlus:                true,
					PositionMode:                  "dual",
					History: MarginHistoryTotals{
						DepositAndWithdrawal:      1.89047097,
						PNL:                       0.10519,
						Fee:                       -0.141010882,
						ReferrerRebates:           0.2,
						FundingCost:               0.3,
						PointDepositAndWithdrawal: 0.4,
						PointFee:                  0.5,
						PointReferrerRebates:      0.6,
						BonusDepositAndWithdrawal: 0.7,
						BonusOffset:               0.8,
						CrossSettle:               -1.854650083,
					},
				},
			},
		}}
		assert.Equal(t, exp, result, "GetSubAccountFuturesBalances should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSubAccountFuturesBalances should return balances")
}

func TestGetSubAccountCrossMarginBalances(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccountCrossMarginBalances(t.Context(), "")
	require.NoError(t, err, "GetSubAccountCrossMarginBalances must not error")
	if mockTests {
		exp := []*SubAccountCrossMarginInfoResponse{{
			UserID: "100000",
			Available: &SubAccountCrossMarginDetail{
				UserID:                     100003,
				Total:                      20,
				Borrowed:                   1,
				UnpaidInterestUSDT:         0.5,
				BorrowedNet:                3,
				TotalNetAssets:             20,
				Leverage:                   3,
				Risk:                       9999.99,
				TotalInitialMargin:         0.1,
				TotalMarginBalance:         20,
				TotalMaintenanceMargin:     0.2,
				TotalInitialMarginRate:     9999.99,
				TotalMaintenanceMarginRate: 9999.98,
				TotalAvailableMargin:       19.9,
				CurrencyBalances:           map[string]CrossMarginBalance{"USDT": {Available: 20, Freeze: 0.5, Borrowed: 1, Interest: 0.25}},
			},
		}}
		assert.Equal(t, exp, result, "GetSubAccountCrossMarginBalances should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSubAccountCrossMarginBalances should return balances")
}

func TestGetSavedAddresses(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSavedAddresses(t.Context(), &SavedAddressesRequest{Currency: currency.BTC, Verified: "1", Page: 10})
	require.NoError(t, err, "GetSavedAddresses must not error")
	if mockTests {
		exp := []*WalletSavedAddressResponse{{Currency: currency.BTC, Chain: "BTC", Address: "1HkxtBAMrA3tP5ENnYY2CZortjZvFDH5Cs", Name: "gate", Verified: true}}
		assert.Equal(t, exp, result, "GetSavedAddresses should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSavedAddresses should return addresses")
}

func TestGetPersonalTradingFee(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPersonalTradingFee(t.Context(), getPair(t, asset.Spot), currency.EMPTYCODE)
	require.NoError(t, err, "GetPersonalTradingFee must not error")
	if mockTests {
		exp := &PersonalTradingFeeResponse{
			UserID:                         10001,
			TakerFee:                       0.002,
			MakerFee:                       0.002,
			RetailPriceImprovementMakerFee: -0.00175,
			GTDiscount:                     true,
			GTTakerFee:                     0.0015,
			GTMakerFee:                     0.0015,
			LoanFee:                        0.18,
			PointType:                      "1",
			FuturesTakerFee:                -0.00025,
			FuturesMakerFee:                0.00075,
			FuturesRPIMakerFee:             -0.00175,
			DeliveryTakerFee:               0.00016,
			DeliveryMakerFee:               -0.00015,
			FeeDeductionType:               3,
			RPIMarketMakerLevel:            2,
		}
		assert.Equal(t, exp, result, "GetPersonalTradingFee should decode every field")
	} else {
		assert.NotNil(t, result, "GetPersonalTradingFee should return fees")
	}

	// Gate documents the settlement currencies in upper case
	_, err = e.GetPersonalTradingFee(t.Context(), currency.EMPTYPAIR, currency.USDT)
	assert.NoError(t, err, "GetPersonalTradingFee should not error for a settlement currency")
}

func TestGetUsersTotalBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUsersTotalBalance(t.Context(), currency.BTC)
	require.NoError(t, err, "GetUsersTotalBalance must not error")
	if mockTests {
		exp := &UsersAllAccountBalanceResponse{
			Details: map[string]CurrencyBalanceAmount{
				"spot":         {Currency: currency.BTC, Amount: 1.5},
				"margin":       {Currency: currency.BTC, Amount: 0.25, Borrowed: 0.1},
				"cross_margin": {Currency: currency.BTC, Amount: 0, Borrowed: 0},
				"futures":      {Currency: currency.BTC, Amount: 0.75, UnrealisedPNL: -0.05},
			},
			Total: CurrencyBalanceAmount{Currency: currency.BTC, Amount: 2.5, UnrealisedPNL: -0.05, Borrowed: 0.1},
		}
		assert.Equal(t, exp, result, "GetUsersTotalBalance should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUsersTotalBalance should return balances")
}

func TestGetConvertibleSmallBalanceCurrencies(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetConvertibleSmallBalanceCurrencies(t.Context())
	require.NoError(t, err, "GetConvertibleSmallBalanceCurrencies must not error")
	if mockTests {
		exp := []*ConvertibleSmallBalanceResponse{
			{Currency: currency.FLOKI, AvailableBalance: 182.294, EstimatedAsBTC: 0.00000012, ConvertibleToGT: 0.00108},
			{Currency: currency.NewCode("MBLK"), AvailableBalance: 0.91723337, EstimatedAsBTC: 0.00000102, ConvertibleToGT: 0.009188},
		}
		assert.Equal(t, exp, result, "GetConvertibleSmallBalanceCurrencies should decode every field")
		return
	}
	assert.NotNil(t, result, "GetConvertibleSmallBalanceCurrencies should return balances")
}

func TestConvertSmallBalances(t *testing.T) {
	t.Parallel()
	err := e.ConvertSmallBalances(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "ConvertSmallBalances must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.ConvertSmallBalances(t.Context(), currency.F16)
	require.NoError(t, err, "ConvertSmallBalances must not error")
}

func TestGetConvertibleSmallBalanceCurrencyHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetConvertibleSmallBalanceCurrencyHistory(t.Context(), currency.ETH, 1, 10)
	require.NoError(t, err, "GetConvertibleSmallBalanceCurrencyHistory must not error")
	if mockTests {
		exp := []*SmallCurrencyBalanceResponse{{ID: "28583810", CreateTime: types.Time(time.Unix(1706670777, 0)), Currency: currency.ETH, Amount: 0.00182294, GTAmount: 0.001079}}
		assert.Equal(t, exp, result, "GetConvertibleSmallBalanceCurrencyHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetConvertibleSmallBalanceCurrencyHistory should return records")
}

func TestGetUIDTransferHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetUIDTransferHistory(t.Context(), &UIDTransferHistoryRequest{From: endTime, To: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUIDTransferHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUIDTransferHistory(t.Context(), &UIDTransferHistoryRequest{ID: 111, From: startTime, To: endTime, Limit: 10, TransactionType: "withdraw"})
	require.NoError(t, err, "GetUIDTransferHistory must not error")
	if mockTests {
		exp := []*UIDTransferRecordResponse{{
			ID:              111,
			InitiatorUserID: 1132,
			RecipientUserID: 12324,
			Currency:        currency.BTC,
			Amount:          1.2,
			CreateTime:      types.Time(time.Unix(1744150000, 0)),
			Status:          "PENDING",
			Message:         "The other party has not completed KYC,There is a security risk in your account, please contact customer service",
			TransactionType: "withdraw",
		}}
		assert.Equal(t, exp, result, "GetUIDTransferHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUIDTransferHistory should return records")
}

func TestGetLowCapExchangeList(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLowCapExchangeList(t.Context())
	require.NoError(t, err, "GetLowCapExchangeList must not error")
	if mockTests {
		exp := []currency.Code{currency.NewCode("0DOG"), currency.NewCode("100X"), currency.NewCode("3AC_OLD")}
		assert.Equal(t, exp, result, "GetLowCapExchangeList should decode every currency")
		return
	}
	assert.NotEmpty(t, result, "GetLowCapExchangeList should return currencies")
}
