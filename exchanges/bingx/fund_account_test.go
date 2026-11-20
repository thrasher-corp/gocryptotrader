package bingx

import (
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// accountWalletJSONBodyExchange returns an exchange whose test server checks that a request is a JSON body POST to path
// whose business members are exp, and replies with reply
func accountWalletJSONBodyExchange(tb testing.TB, path, exp, reply string) *Exchange {
	tb.Helper()
	return newTestServerExchange(tb, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(tb, http.MethodPost, r.Method, "method should be POST")
		assert.Equal(tb, path, r.URL.Path, "path should be the endpoint's")
		assert.Equal(tb, "application/json", r.Header.Get("Content-Type"), "content type should be JSON")
		assert.Empty(tb, r.URL.RawQuery, "query should be empty")
		raw, err := io.ReadAll(r.Body)
		assert.NoError(tb, err, "reading the body should not error")
		var body map[string]json.RawMessage
		assert.NoError(tb, json.Unmarshal(raw, &body), "body should be a JSON object")
		for _, k := range []string{"recvWindow", "timestamp", "signature"} {
			assert.Containsf(tb, body, k, "body should carry %s", k)
			delete(body, k)
		}
		members, err := json.Marshal(body)
		assert.NoError(tb, err, "marshalling the body's members should not error")
		assert.JSONEq(tb, exp, string(members), "body should carry the request's members")
		_, _ = io.WriteString(w, reply)
	})
}

func TestGetSpotAccountAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSpotAccountAssets(t.Context())
	require.NoError(t, err, "GetSpotAccountAssets must not error")
	require.NotNil(t, result, "GetSpotAccountAssets must return a result")
	if mockTests {
		exp := &AssetBalancesResponse{
			Balances: []AssetBalance{
				{Asset: currency.USDT, Free: 566773.193402631, Locked: 244.18616265388994},
				{Asset: currency.NewCode("CHEEMS"), Free: 294854132046232, Locked: 18350553840},
				{Asset: currency.NewCode("VST")},
			},
		}
		assert.Equal(t, exp, result, "GetSpotAccountAssets should decode every field")
	}
}

func TestGetAssetTransferRecords(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *AssetTransferRecordsRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no type or transaction ID", req: &AssetTransferRecordsRequest{Current: 1}, err: errTransferTypeOrIDRequired},
		{name: "reversed time range", req: &AssetTransferRecordsRequest{Type: AssetTransferTypeFundToPerpetualFutures, StartTime: time.UnixMilli(2000), EndTime: time.UnixMilli(1000)}, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetAssetTransferRecords(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "GetAssetTransferRecords should return the validation error")
		})
	}

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetAssetTransferRecords(t.Context(), &AssetTransferRecordsRequest{
			Type:          AssetTransferTypeFundToPerpetualFutures,
			TransactionID: 37600111,
			StartTime:     time.UnixMilli(1702252271000),
			EndTime:       time.UnixMilli(1702351131000),
			Current:       1,
			Size:          100,
		})
		require.NoError(t, err, "GetAssetTransferRecords must not error")
		require.NotNil(t, result, "GetAssetTransferRecords must return a result")
		if mockTests {
			exp := &AssetTransferRecordsResponse{
				Total: 2,
				Rows: []AssetTransferRecord{
					{
						Asset:         currency.NewCode("VST"),
						Amount:        100000,
						Type:          AssetTransferTypeFundToPerpetualFutures,
						Status:        "CONFIRMED",
						TransactionID: 37600111,
						Timestamp:     types.Time(time.UnixMilli(1702252271000)),
					},
					{
						Asset:         currency.USDT,
						Amount:        2218.72352626,
						Type:          AssetTransferTypeFundToPerpetualFutures,
						Status:        "CONFIRMED",
						TransactionID: 37600222,
						Timestamp:     types.Time(time.UnixMilli(1702351131000)),
					},
				},
			}
			assert.Equal(t, exp, result, "GetAssetTransferRecords should decode every field")
		}
	})
}

func TestMainAccountInternalTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.MainAccountInternalTransfer(t.Context(), &InternalTransferRequest{Coin: currency.USDT})
	require.ErrorIs(t, err, errUserAccountTypeRequired, "MainAccountInternalTransfer must return the request's validation error")

	t.Run("JSON body", func(t *testing.T) {
		t.Parallel()
		ex := accountWalletJSONBodyExchange(t, "/openApi/wallets/v1/capital/innerTransfer/apply",
			`{"coin":"USDT","userAccountType":2,"userAccount":"13800000000","amount":0.5,"callingCode":"86","walletType":15,"transferClientId":"abc123"}`,
			`{"code":0,"timestamp":1702558152381,"data":{"id":"1251111922229444400","transferClientId":"abc123"}}`)
		result, err := ex.MainAccountInternalTransfer(t.Context(), &InternalTransferRequest{
			Coin:             currency.NewCode("usdt"),
			UserAccountType:  UserAccountTypePhoneNumber,
			UserAccount:      "13800000000",
			Amount:           0.5,
			CallingCode:      "86",
			WalletType:       WalletTypeSpot,
			TransferClientID: "abc123",
		})
		require.NoError(t, err, "MainAccountInternalTransfer must not error")
		assert.Equal(t, &InternalTransferResponse{ID: "1251111922229444400", TransferClientID: "abc123"}, result, "MainAccountInternalTransfer should decode every field")
	})

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		if !mockTests {
			sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
		}
		result, err := e.MainAccountInternalTransfer(t.Context(), &InternalTransferRequest{
			Coin:             currency.USDT,
			UserAccountType:  UserAccountTypeUID,
			UserAccount:      "16779999",
			Amount:           10,
			WalletType:       WalletTypeFund,
			TransferClientID: "abc123",
		})
		require.NoError(t, err, "MainAccountInternalTransfer must not error")
		require.NotNil(t, result, "MainAccountInternalTransfer must return a result")
		if mockTests {
			assert.Equal(t, &InternalTransferResponse{ID: "12******1"}, result, "MainAccountInternalTransfer should decode the documented example")
		}
	})
}

func TestFundAccountInternalTransferBody(t *testing.T) {
	t.Parallel()
	valid := InternalTransferRequest{
		Coin:            currency.NewCode("usdt"),
		UserAccountType: UserAccountTypeUID,
		UserAccount:     "16779999",
		Amount:          10,
		WalletType:      WalletTypeFund,
	}
	for _, tc := range []struct {
		name   string
		modify func(*InternalTransferRequest)
		err    error
	}{
		{name: "empty coin", modify: func(r *InternalTransferRequest) { r.Coin = currency.EMPTYCODE }, err: currency.ErrCurrencyCodeEmpty},
		{name: "no user account type", modify: func(r *InternalTransferRequest) { r.UserAccountType = 0 }, err: errUserAccountTypeRequired},
		{name: "no user account", modify: func(r *InternalTransferRequest) { r.UserAccount = "" }, err: errUserAccountRequired},
		{name: "phone number without calling code", modify: func(r *InternalTransferRequest) { r.UserAccountType = UserAccountTypePhoneNumber }, err: errCallingCodeRequired},
		{name: "zero amount", modify: func(r *InternalTransferRequest) { r.Amount = 0 }, err: limits.ErrAmountBelowMin},
		{name: "negative amount", modify: func(r *InternalTransferRequest) { r.Amount = -1 }, err: limits.ErrAmountBelowMin},
		{name: "no wallet type", modify: func(r *InternalTransferRequest) { r.WalletType = 0 }, err: errWalletTypeRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := valid
			tc.modify(&req)
			_, err := fundAccountInternalTransferBody(&req)
			assert.ErrorIs(t, err, tc.err, "fundAccountInternalTransferBody should return the validation error")
		})
	}

	t.Run("nil request", func(t *testing.T) {
		t.Parallel()
		_, err := fundAccountInternalTransferBody(nil)
		assert.ErrorIs(t, err, common.ErrNilPointer, "fundAccountInternalTransferBody should reject a nil request")
	})

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		req := valid
		req.UserAccountType = UserAccountTypePhoneNumber
		req.CallingCode = "86"
		body, err := fundAccountInternalTransferBody(&req)
		require.NoError(t, err, "fundAccountInternalTransferBody must not error")
		require.NotNil(t, body, "fundAccountInternalTransferBody must return a body")
		assert.Equal(t, "USDT", body.Coin.String(), "the body's coin should be upper case")
		assert.Equal(t, "usdt", req.Coin.String(), "the request's coin should be left as it was")
		body.Coin = req.Coin
		assert.Equal(t, req, *body, "the body should carry every other field unchanged")
	})
}

func TestAssetTransfer(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *AssetTransferRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no from account", req: &AssetTransferRequest{ToAccount: TransferAccountSpot, Asset: currency.USDT, Amount: 1}, err: errTransferAccountRequired},
		{name: "no to account", req: &AssetTransferRequest{FromAccount: TransferAccountFund, Asset: currency.USDT, Amount: 1}, err: errTransferAccountRequired},
		{name: "empty asset", req: &AssetTransferRequest{FromAccount: TransferAccountFund, ToAccount: TransferAccountSpot, Amount: 1}, err: currency.ErrCurrencyCodeEmpty},
		{name: "zero amount", req: &AssetTransferRequest{FromAccount: TransferAccountFund, ToAccount: TransferAccountSpot, Asset: currency.USDT}, err: limits.ErrAmountBelowMin},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.AssetTransfer(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "AssetTransfer should return the validation error")
		})
	}

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		if !mockTests {
			sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
		}
		result, err := e.AssetTransfer(t.Context(), &AssetTransferRequest{FromAccount: TransferAccountFund, ToAccount: TransferAccountSpot, Asset: currency.USDT, Amount: 1095})
		require.NoError(t, err, "AssetTransfer must not error")
		require.NotNil(t, result, "AssetTransfer must return a result")
		if mockTests {
			assert.Equal(t, &AssetTransferResponse{TransferID: "17********28"}, result, "AssetTransfer should decode every field")
		}
	})
}

func TestGetTransferableCurrencies(t *testing.T) {
	t.Parallel()
	_, err := e.GetTransferableCurrencies(t.Context(), "", TransferAccountFund)
	require.ErrorIs(t, err, errTransferAccountRequired, "GetTransferableCurrencies must reject an empty from account")
	_, err = e.GetTransferableCurrencies(t.Context(), TransferAccountSpot, "")
	require.ErrorIs(t, err, errTransferAccountRequired, "GetTransferableCurrencies must reject an empty to account")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetTransferableCurrencies(t.Context(), TransferAccountSpot, TransferAccountFund)
	require.NoError(t, err, "GetTransferableCurrencies must not error")
	require.NotNil(t, result, "GetTransferableCurrencies must return a result")
	if mockTests {
		exp := &TransferableCurrenciesResponse{
			Coins: []TransferableCoin{
				{Asset: currency.USDT, Amount: 566773.193402631},
				{Asset: currency.NewCode("CHEEMS"), Amount: 24324.21},
			},
		}
		assert.Equal(t, exp, result, "GetTransferableCurrencies should decode every field")
	}
}

func TestGetAssetTransferRecordsNew(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *AssetTransferRecordsNewRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no accounts or transaction ID", req: &AssetTransferRecordsNewRequest{PageIndex: 1}, err: errTransferAccountRequired},
		{name: "one account without a transaction ID", req: &AssetTransferRecordsNewRequest{FromAccount: TransferAccountFund}, err: errTransferAccountRequired},
		{name: "reversed time range", req: &AssetTransferRecordsNewRequest{TransactionID: "37600111", StartTime: time.UnixMilli(2000), EndTime: time.UnixMilli(1000)}, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetAssetTransferRecordsNew(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "GetAssetTransferRecordsNew should return the validation error")
		})
	}

	t.Run("transaction ID without accounts", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()
			assert.Equal(t, "37600111", q.Get("tranId"), "the transaction ID should be sent")
			assert.False(t, q.Has("fromAccount"), "an unset from account should not be sent")
			assert.False(t, q.Has("toAccount"), "an unset to account should not be sent")
			_, _ = io.WriteString(w, `{"total":0,"rows":[]}`)
		})
		result, err := ex.GetAssetTransferRecordsNew(t.Context(), &AssetTransferRecordsNewRequest{TransactionID: "37600111"})
		require.NoError(t, err, "GetAssetTransferRecordsNew must not error")
		assert.Equal(t, &AssetTransferRecordsNewResponse{Rows: []AssetTransferRecordNew{}}, result, "GetAssetTransferRecordsNew should decode an empty page")
	})

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetAssetTransferRecordsNew(t.Context(), &AssetTransferRecordsNewRequest{
			FromAccount:   TransferAccountFund,
			ToAccount:     TransferAccountSpot,
			TransactionID: "37600111",
			StartTime:     time.UnixMilli(1702252271000),
			EndTime:       time.UnixMilli(1702351131000),
			PageIndex:     1,
			PageSize:      10,
		})
		require.NoError(t, err, "GetAssetTransferRecordsNew must not error")
		require.NotNil(t, result, "GetAssetTransferRecordsNew must return a result")
		if mockTests {
			exp := &AssetTransferRecordsNewResponse{
				Total: 2,
				Rows: []AssetTransferRecordNew{
					{
						Asset:       currency.NewCode("VST"),
						Amount:      100000,
						Status:      "CONFIRMED",
						TransferID:  "37600111",
						Timestamp:   types.Time(time.UnixMilli(1702252271000)),
						FromAccount: TransferAccountFund,
						ToAccount:   TransferAccountSpot,
					},
				},
			}
			assert.Equal(t, exp, result, "GetAssetTransferRecordsNew should decode every field")
		}
	})
}

func TestGetFundAccountAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFundAccountAssets(t.Context(), currency.NewCode("usdt"))
	require.NoError(t, err, "GetFundAccountAssets must not error")
	require.NotNil(t, result, "GetFundAccountAssets must return a result")
	if mockTests {
		exp := &AssetBalancesResponse{
			Balances: []AssetBalance{
				{Asset: currency.USDT, Free: 566773.193402631, Locked: 244.18616265388994},
				{Asset: currency.NewCode("CHEEMS"), Free: 294854132046232, Locked: 18350553840},
				{Asset: currency.NewCode("VST")},
			},
		}
		assert.Equal(t, exp, result, "GetFundAccountAssets should decode every field")
	}
}

func TestGetMainAccountInternalTransferRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetMainAccountInternalTransferRecords(t.Context(), &InternalTransferRecordsRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMainAccountInternalTransferRecords must return the request's validation error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMainAccountInternalTransferRecords(t.Context(), &InternalTransferRecordsRequest{
		Coin:             currency.BTC,
		ID:               "1251111922229444400",
		TransferClientID: "abc123",
		StartTime:        time.UnixMilli(1701519898118),
		EndTime:          time.UnixMilli(1702383898118),
		Offset:           1,
		Limit:            1000,
	})
	require.NoError(t, err, "GetMainAccountInternalTransferRecords must not error")
	require.NotNil(t, result, "GetMainAccountInternalTransferRecords must return a result")
	if mockTests {
		exp := &InternalTransferRecordsResponse{
			Data: []InternalTransferRecord{
				{
					ID:         1251111922229444400,
					Coin:       currency.BTC,
					Receiver:   1128763679,
					Amount:     0.0072366,
					Status:     WithdrawStatusCompleted,
					FromUID:    1128763678,
					RecordType: InternalTransferRecordTypeOut,
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetMainAccountInternalTransferRecords should decode every field")
	}
}

func TestFundAccountInternalTransferRecordsParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *InternalTransferRecordsRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "empty coin", req: &InternalTransferRecordsRequest{Limit: 10}, err: currency.ErrCurrencyCodeEmpty},
		{name: "reversed time range", req: &InternalTransferRecordsRequest{Coin: currency.BTC, StartTime: time.UnixMilli(2000), EndTime: time.UnixMilli(1000)}, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := fundAccountInternalTransferRecordsParams(tc.req)
			assert.ErrorIs(t, err, tc.err, "fundAccountInternalTransferRecordsParams should return the validation error")
		})
	}

	t.Run("every parameter", func(t *testing.T) {
		t.Parallel()
		params, err := fundAccountInternalTransferRecordsParams(&InternalTransferRecordsRequest{
			Coin:             currency.NewCode("btc"),
			ID:               "7",
			TransferClientID: "abc123",
			StartTime:        time.UnixMilli(1000),
			EndTime:          time.UnixMilli(2000),
			Offset:           3,
			Limit:            100,
		})
		require.NoError(t, err, "fundAccountInternalTransferRecordsParams must not error")
		exp := url.Values{
			"coin":             {"BTC"},
			"id":               {"7"},
			"transferClientId": {"abc123"},
			"startTime":        {"1000"},
			"endTime":          {"2000"},
			"offset":           {"3"},
			"limit":            {"100"},
		}
		assert.Equal(t, exp, params, "fundAccountInternalTransferRecordsParams should set every parameter")
	})

	t.Run("coin only", func(t *testing.T) {
		t.Parallel()
		params, err := fundAccountInternalTransferRecordsParams(&InternalTransferRecordsRequest{Coin: currency.BTC})
		require.NoError(t, err, "fundAccountInternalTransferRecordsParams must not error")
		assert.Equal(t, url.Values{"coin": {"BTC"}}, params, "fundAccountInternalTransferRecordsParams should leave unset parameters out")
	})
}

func TestGetAssetOverview(t *testing.T) {
	t.Parallel()
	t.Run("all accounts", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.False(t, r.URL.Query().Has("accountType"), "an unset account type should not be sent")
			_, _ = io.WriteString(w, `{"code":0,"timestamp":1719494258281,"data":{"result":[],"pageId":1,"total":0}}`)
		})
		result, err := ex.GetAssetOverview(t.Context(), "")
		require.NoError(t, err, "GetAssetOverview must not error")
		assert.Equal(t, &AssetOverviewResponse{Result: []AccountBalanceOverview{}, PageID: 1}, result, "GetAssetOverview should decode an empty page")
	})

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetAssetOverview(t.Context(), AccountTypeSpot)
		require.NoError(t, err, "GetAssetOverview must not error")
		require.NotNil(t, result, "GetAssetOverview must return a result")
		if mockTests {
			exp := &AssetOverviewResponse{
				Result: []AccountBalanceOverview{{AccountType: AccountTypeSpot, USDTBalance: 100}},
				PageID: 1,
				Total:  1,
			}
			assert.Equal(t, exp, result, "GetAssetOverview should decode every field")
		}
	})
}
