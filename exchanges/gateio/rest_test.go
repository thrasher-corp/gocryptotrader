package gateio

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

const canManipulateRealOrders = false

// apiCredentials holds the credentials used for due diligence testing; please supply your own
var apiCredentials = &accounts.Credentials{
	Key:    "",
	Secret: "",
}

var (
	e *Exchange
	// enabledAssetPair holds each asset's test pair, set by TestMain for mock tests and by ensureTestPairs for live tests
	enabledAssetPair map[asset.Item]currency.Pair
)

// mockTestPairs returns the pair each asset's recorded fixtures in testdata/http.json are keyed on
func mockTestPairs() map[asset.Item]currency.Pair {
	btcUSDT := currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter)
	return map[asset.Item]currency.Pair{
		asset.Spot:                btcUSDT,
		asset.Margin:              btcUSDT,
		asset.CrossMargin:         btcUSDT,
		asset.USDTMarginedFutures: btcUSDT,
		asset.CoinMarginedFutures: currency.NewPairWithDelimiter("BTC", "USD", currency.UnderscoreDelimiter),
		asset.DeliveryFutures:     currency.NewPairWithDelimiter("BTC", "USDT_20260626", currency.UnderscoreDelimiter),
		asset.Options:             currency.NewPairWithDelimiter("BTC", "USDT-20260925-75000-P", currency.UnderscoreDelimiter),
	}
}

// storeTestPairs makes the mock test pairs the only available and enabled pairs of each asset
func storeTestPairs(e *Exchange) error {
	for a, p := range mockTestPairs() {
		if err := e.CurrencyPairs.StorePairs(a, []currency.Pair{p}, false); err != nil {
			return err
		}
		if err := e.CurrencyPairs.StorePairs(a, []currency.Pair{p}, true); err != nil {
			return err
		}
	}
	return nil
}

func getTime() (startTime, endTime time.Time) {
	if mockTests {
		return time.UnixMilli(1744103854944), time.UnixMilli(1744190254944)
	}
	return time.Now().Add(-time.Hour * 48), time.Now()
}

func getTimeWithInterval(interval kline.Interval) (startTime, endTime time.Time) {
	if mockTests {
		startTime = time.UnixMilli(1744103854944)
		return startTime, startTime.Add(time.Duration(interval))
	}
	startTime = time.Now()
	return startTime.Add(-time.Hour * 48), startTime
}

func TestSetUnixTimeRangeParams(t *testing.T) {
	t.Parallel()
	from := time.Unix(1710000000, 0)
	to := from.Add(time.Hour)
	for _, tc := range []struct {
		name           string
		from           time.Time
		to             time.Time
		expectedParams url.Values
		expectedErr    error
	}{
		{
			name:           "both set",
			from:           from,
			to:             to,
			expectedParams: url.Values{"from": {strconv.FormatInt(from.Unix(), 10)}, "to": {strconv.FormatInt(to.Unix(), 10)}},
		},
		{
			name:           "from only",
			from:           from,
			expectedParams: url.Values{"from": {strconv.FormatInt(from.Unix(), 10)}},
		},
		{
			name:           "to only",
			to:             to,
			expectedParams: url.Values{"to": {strconv.FormatInt(to.Unix(), 10)}},
		},
		{
			name:           "both zero",
			expectedParams: url.Values{},
		},
		{
			name:           "start after end",
			from:           to,
			to:             from,
			expectedParams: url.Values{},
			expectedErr:    common.ErrStartAfterEnd,
		},
		{
			name:           "start equals end",
			from:           from,
			to:             from,
			expectedParams: url.Values{"from": {strconv.FormatInt(from.Unix(), 10)}, "to": {strconv.FormatInt(from.Unix(), 10)}},
		},
		{
			name:           "start after current time",
			from:           time.Date(2222, 1, 1, 0, 0, 0, 0, time.UTC),
			to:             time.Date(2222, 1, 2, 0, 0, 0, 0, time.UTC),
			expectedParams: url.Values{},
			expectedErr:    common.ErrStartAfterTimeNow,
		},
		{
			name:           "equal times after current time",
			from:           time.Date(2222, 1, 1, 0, 0, 0, 0, time.UTC),
			to:             time.Date(2222, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedParams: url.Values{},
			expectedErr:    common.ErrStartAfterTimeNow,
		},
		{
			name:           "unix epoch",
			from:           time.Unix(0, 0),
			to:             to,
			expectedParams: url.Values{},
			expectedErr:    common.ErrDateUnset,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params := url.Values{}
			err := setUnixTimeRangeParams(&params, tc.from, tc.to)
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr, "setUnixTimeRangeParams must return the expected error")
			} else {
				require.NoError(t, err, "setUnixTimeRangeParams must not error")
			}
			assert.Equal(t, tc.expectedParams, params, "params should match expected values")
		})
	}
}

func TestUnixSeconds(t *testing.T) {
	t.Parallel()
	assert.Equal(t, uint64(1767153378), unixSeconds(time.Unix(1767153378, 999999999)), "unixSeconds should return whole Unix seconds")
	assert.Zero(t, unixSeconds(time.Unix(0, 0)), "unixSeconds should return 0 at the Unix epoch")
	assert.Zero(t, unixSeconds(time.Unix(-1, 0)), "unixSeconds should return 0 for a time before 1970")
	assert.Zero(t, unixSeconds(time.Time{}), "unixSeconds should return 0 for a zero time")
}

func TestSendUnversionedHTTPRequest(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body string
		switch r.URL.Path {
		case "/apiw/v2/ok":
			body = `{"value":"pool"}`
		case "/apiw/v2/coded":
			body = `{"label":"INVALID_KEY","code":"10001","message":"invalid key"}`
		case "/apiw/v2/status":
			body = `{"code":4,"msg":"boom","trace_id":"trace-1","data":null}`
		default:
			assert.Failf(t, "unexpected request path", "request path %q should be one the test sends", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "Write should not error")
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.EdgeCase1.String(), server.URL), "SetRunningURL must not error")

	var result struct {
		Value string `json:"value"`
	}
	require.NoError(t, ex.sendUnversionedHTTPRequest(t.Context(), exchange.EdgeCase1, publicIsolatedMarginPoolLoansEPL, "/apiw/v2/ok", &result), "sendUnversionedHTTPRequest must not error")
	assert.Equal(t, "pool", result.Value, "sendUnversionedHTTPRequest should decode the response at the unprefixed path")

	assert.NoError(t, ex.sendUnversionedHTTPRequest(t.Context(), exchange.EdgeCase1, publicIsolatedMarginPoolLoansEPL, "/apiw/v2/ok", nil), "sendUnversionedHTTPRequest should not error without a result")

	for _, res := range []any{&result, nil} {
		err := ex.sendUnversionedHTTPRequest(t.Context(), exchange.EdgeCase1, publicIsolatedMarginPoolLoansEPL, "/apiw/v2/coded", res)
		require.Error(t, err, "sendUnversionedHTTPRequest must error on a coded error body")
		assert.Contains(t, err.Error(), "INVALID_KEY", "error should carry the error label")
	}

	var status *BotCreateResponse
	err := ex.sendUnversionedHTTPRequest(t.Context(), exchange.EdgeCase1, publicIsolatedMarginPoolLoansEPL, "/apiw/v2/status", &status)
	require.Error(t, err, "sendUnversionedHTTPRequest must surface an error carried in the response body")
	assert.Contains(t, err.Error(), "trace-1", "error should carry the trace ID")
}

func TestSendPublicPostRequest(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "request method should be POST")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "request should carry a JSON body")
		assert.Empty(t, r.Header.Get("KEY"), "request should not be signed")
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "ReadAll should not error")
		assert.JSONEq(t, `{"contracts":["BTC_USDT"]}`, string(body), "request body should be the marshalled data")
		var resp string
		switch r.URL.Path {
		case "/api/v4/ok":
			resp = `[{"contract":"BTC_USDT"}]`
		case "/api/v4/label":
			resp = `{"label":"INVALID_PARAM_VALUE","message":"invalid contract"}`
		case "/api/v4/coded":
			resp = `{"code":4,"msg":"boom","trace_id":"trace-1"}`
		default:
			resp = `{`
		}
		_, err = w.Write([]byte(resp))
		assert.NoError(t, err, "Write should not error")
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")
	body := &struct {
		Contracts []string `json:"contracts"`
	}{Contracts: []string{"BTC_USDT"}}

	var result []struct {
		Contract string `json:"contract"`
	}
	require.NoError(t, ex.sendPublicPostRequest(t.Context(), publicBatchFundingRatesEPL, "ok", body, &result), "sendPublicPostRequest must not error")
	assert.Equal(t, []struct {
		Contract string `json:"contract"`
	}{{Contract: "BTC_USDT"}}, result, "sendPublicPostRequest should decode the response")

	err := ex.sendPublicPostRequest(t.Context(), publicBatchFundingRatesEPL, "label", body, &result)
	require.Error(t, err, "sendPublicPostRequest must error on a labelled error body")
	assert.Contains(t, err.Error(), "INVALID_PARAM_VALUE", "error should carry the error label")

	var coded *BotCreateResponse
	err = ex.sendPublicPostRequest(t.Context(), publicBatchFundingRatesEPL, "coded", body, &coded)
	require.Error(t, err, "sendPublicPostRequest must surface an error carried in the response body")
	assert.Contains(t, err.Error(), "trace-1", "error should carry the trace ID")

	require.Error(t, ex.sendPublicPostRequest(t.Context(), publicBatchFundingRatesEPL, "malformed", body, &result), "sendPublicPostRequest must error on a malformed response")
}

func TestGateioAPIResponseAsError(t *testing.T) {
	t.Parallel()
	assert.NoError(t, (&gateioAPIResponse[*OTCQuoteResponse]{Message: "success"}).AsError(), "a zero code should not error")
	err := (&gateioAPIResponse[*OTCQuoteResponse]{Code: 1001, Message: "quote expired"}).AsError()
	require.Error(t, err, "a non-zero code must error")
	assert.Equal(t, "error code: 1001 message: quote expired", err.Error(), "error should carry the code and message")
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	require.NoError(t, testexch.MockHTTPInstance(e, ""), "MockHTTPInstance must not error")
	require.NoError(t, storeTestPairs(e), "storeTestPairs must not error")
	testexch.UpdatePairsOnce(t, e)
}

func TestGetAccountBalances(t *testing.T) {
	t.Parallel()
	ex := newIsolatedExchange(t)
	_, err := ex.UpdateAccountBalances(t.Context(), asset.Binary)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateAccountBalances must reject an unsupported asset")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
	}
	futuresBalance := accounts.CurrencyBalances{
		// Hold is what is not available, the float64 difference of the total and the available balance
		currency.USDT: {Currency: currency.USDT, Total: 9707.803567115145, Hold: 38.71218918100021, Free: 9669.091377934145},
	}
	for _, tc := range []struct {
		a   asset.Item
		exp accounts.CurrencyBalances
	}{
		{
			a: asset.Spot,
			exp: accounts.CurrencyBalances{
				currency.BTC:  {Currency: currency.BTC, Total: 969.3, Hold: 0.5, Free: 968.8},
				currency.USDT: {Currency: currency.USDT, Total: 2621, Hold: 120.75, Free: 2500.25},
			},
		},
		{
			a: asset.Margin,
			exp: accounts.CurrencyBalances{
				currency.BTC:  {Currency: currency.BTC, Total: 0.05, Free: 0.05, AvailableWithoutBorrow: 0.05},
				currency.USDT: {Currency: currency.USDT, Total: 1000, Free: 1000, AvailableWithoutBorrow: 1000},
			},
		},
		{
			// The interest owed counts as borrowed
			a: asset.CrossMargin,
			exp: accounts.CurrencyBalances{
				currency.BTC: {Currency: currency.BTC, Total: 0.6, Hold: 0.1, Free: 0.5, AvailableWithoutBorrow: 0.4799, Borrowed: 0.0201},
			},
		},
		{a: asset.USDTMarginedFutures, exp: futuresBalance},
		{
			a: asset.CoinMarginedFutures,
			exp: accounts.CurrencyBalances{
				currency.BTC: {Currency: currency.BTC, Total: 0.5, Hold: 0.04999999999999999, Free: 0.45},
			},
		},
		{a: asset.DeliveryFutures, exp: maps.Clone(futuresBalance)},
		{
			a: asset.Options,
			exp: accounts.CurrencyBalances{
				currency.USDT: {Currency: currency.USDT, Total: 1650.443022, Hold: 135.54148499999997, Free: 1514.901537},
			},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			started := time.Now()
			result, err := ex.UpdateAccountBalances(t.Context(), tc.a)
			require.NoError(t, err, "UpdateAccountBalances must not error")
			if !mockTests {
				assert.Len(t, result, 1, "UpdateAccountBalances should return the main account")
				return
			}
			require.Len(t, result, 1, "UpdateAccountBalances must return the main account")
			for c, b := range tc.exp {
				// Balances are stamped when they are saved
				assert.WithinRangef(t, result[0].Balances[c].UpdatedAt, started, time.Now(), "UpdateAccountBalances should stamp the %s balance", c)
				b.UpdatedAt = result[0].Balances[c].UpdatedAt
				tc.exp[c] = b
			}
			assert.Equal(t, accounts.SubAccounts{{AssetType: tc.a, Balances: tc.exp}}, result, "UpdateAccountBalances should return every balance")
		})
	}
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()
	ex := newIsolatedExchange(t)
	require.ErrorIs(t, ex.ValidateAPICredentials(t.Context(), asset.Binary), asset.ErrNotSupported, "ValidateAPICredentials must reject an unsupported asset")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
	}
	assert.NoError(t, ex.ValidateAPICredentials(t.Context(), asset.Spot), "ValidateAPICredentials should not error")
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawCryptocurrencyFunds(t.Context(), nil)
	require.ErrorIs(t, err, withdraw.ErrRequestCannotBeNil)

	arg := &withdraw.Request{
		Description: "WITHDRAW IT ALL",
	}
	_, err = e.WithdrawCryptocurrencyFunds(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrExchangeNameNotSet)

	arg.Exchange = e.Name
	_, err = e.WithdrawCryptocurrencyFunds(t.Context(), arg)
	require.ErrorContains(t, err, withdraw.ErrStrAmountMustBeGreaterThanZero)

	arg.Amount = 1
	_, err = e.WithdrawCryptocurrencyFunds(t.Context(), arg)
	require.ErrorContains(t, err, withdraw.ErrStrNoCurrencySet)

	arg.Currency = currency.USD
	arg.Crypto = withdraw.CryptoRequest{
		Address: "bc1qk0jareu4jytc0cfrhr5wgshsq8282awp",
		Chain:   "",
	}
	_, err = e.WithdrawCryptocurrencyFunds(t.Context(), arg)
	require.ErrorContains(t, err, withdraw.ErrStrCurrencyNotCrypto)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		cryptocurrencyChains, err := e.GetAvailableTransferChains(t.Context(), currency.BTC)
		require.NoError(t, err, "GetAvailableTransferChains must not error")
		require.NotEmpty(t, cryptocurrencyChains, "GetAvailableTransferChains must return some chains")
		result, err := e.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
			Exchange:    e.Name,
			Amount:      0.0001,
			Currency:    currency.BTC,
			Description: "WITHDRAW IT ALL",
			Type:        withdraw.Crypto,
			Crypto: withdraw.CryptoRequest{
				Address: core.BitcoinDonationAddress,
				Chain:   cryptocurrencyChains[0],
			},
		})
		require.NoError(t, err, "WithdrawCryptocurrencyFunds must not error")
		assert.NotEmpty(t, result.ID, "WithdrawCryptocurrencyFunds should return the withdrawal ID")
		return
	}
	// The fixture only matches a request carrying the tag as Gate's memo and the client order ID
	result, err := e.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
		Exchange:      e.Name,
		Amount:        25,
		Currency:      currency.XRP,
		Description:   "WITHDRAW IT ALL",
		Type:          withdraw.Crypto,
		ClientOrderID: "gct-withdrawal-1",
		Crypto: withdraw.CryptoRequest{
			Address:    "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
			AddressTag: "100156837",
			Chain:      "XRP",
		},
	})
	require.NoError(t, err, "WithdrawCryptocurrencyFunds must not error")
	// The record ID identifies the withdrawal, while the transaction hash is only known once it reaches the chain
	assert.Equal(t, &withdraw.ExchangeResponse{Name: e.Name, ID: "210496", Status: "DONE"}, result, "WithdrawCryptocurrencyFunds should return the withdrawal record")
}

func TestWithdrawFiatFunds(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFunds(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "WithdrawFiatFunds should not be supported")
}

func TestWithdrawFiatFundsToInternationalBank(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFundsToInternationalBank(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "WithdrawFiatFundsToInternationalBank should not be supported")
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	_, err := e.GetFeeByType(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFeeByType must reject a nil fee builder")

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.SetCredentials(&accounts.Credentials{})
	feeBuilder := &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: currency.NewBTCUSDT(), PurchasePrice: 1000, Amount: 2}
	result, err := ex.GetFeeByType(t.Context(), feeBuilder)
	require.NoError(t, err, "GetFeeByType must not error without credentials")
	assert.Equal(t, exchange.OfflineTradeFee, feeBuilder.FeeType, "GetFeeByType should estimate the trade fee offline without credentials")
	assert.Equal(t, 4.0, result, "GetFeeByType should return the worst case trade fee")

	result, err = ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyWithdrawalFee, Pair: currency.NewBTCUSDT()})
	require.NoError(t, err, "GetFeeByType must not error for a withdrawal fee")
	assert.Equal(t, 0.001, result, "GetFeeByType should return the BTC withdrawal fee")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	// The pair is given without Gate's delimiter, which the fixture only matches once it is formatted
	feeBuilder = &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: currency.NewBTCUSDT(), PurchasePrice: 1000, Amount: 2}
	result, err = e.GetFeeByType(t.Context(), feeBuilder)
	require.NoError(t, err, "GetFeeByType must not error")
	assert.Equal(t, exchange.CryptocurrencyTradeFee, feeBuilder.FeeType, "GetFeeByType should fetch the trade fee with credentials")
	if mockTests {
		assert.Equal(t, 4.0, result, "GetFeeByType should return the account's taker fee")
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	_, err := e.GetFee(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFee must reject a nil fee builder")

	result, err := e.GetFee(t.Context(), &exchange.FeeBuilder{FeeType: exchange.OfflineTradeFee, PurchasePrice: 1000, Amount: 2})
	require.NoError(t, err, "GetFee must not error for the offline trade fee")
	assert.Equal(t, 4.0, result, "GetFee should return the worst case trade fee")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err = e.GetFee(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: currency.NewBTCUSDT(), PurchasePrice: 1000, Amount: 2, IsMaker: true})
	require.NoError(t, err, "GetFee must not error")
	if mockTests {
		assert.Equal(t, 4.0, result, "GetFee should return the account's maker fee")
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderInfo(t.Context(), "1234", currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOrderInfo must reject an empty pair")
	_, err = e.GetOrderInfo(t.Context(), "", currency.NewBTCUSDT(), asset.Spot)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetOrderInfo must reject an empty order ID")
	_, err = e.GetOrderInfo(t.Context(), "1234", currency.NewBTCUSDT(), asset.Binary)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOrderInfo must reject an unsupported asset")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		a       asset.Item
		orderID string
		exp     *order.Detail
	}{
		{
			a:       asset.Spot,
			orderID: "1234",
			exp: &order.Detail{
				Exchange:             e.Name,
				OrderID:              "1234",
				ClientOrderID:        "t-abc123",
				Side:                 order.Buy,
				Type:                 order.Limit,
				Status:               order.Closed,
				Pair:                 getPair(t, asset.Spot),
				AssetType:            asset.Spot,
				Price:                65000,
				Amount:               0.001,
				ExecutedAmount:       0.001,
				AverageExecutedPrice: 63469.3,
				Cost:                 63.4693,
				CostAsset:            currency.USDT,
				Fee:                  0.00000022,
				FeeAsset:             currency.BTC,
				Date:                 time.UnixMilli(1744150000073),
				LastUpdated:          time.UnixMilli(1744150000074),
			},
		},
		{
			a:       asset.Margin,
			orderID: "1234",
			exp: &order.Detail{
				Exchange:             e.Name,
				OrderID:              "1234",
				ClientOrderID:        "t-margin-sell",
				Side:                 order.Sell,
				Type:                 order.Limit,
				Status:               order.Open,
				Pair:                 getPair(t, asset.Margin),
				AssetType:            asset.Margin,
				Price:                70000,
				Amount:               0.002,
				ExecutedAmount:       0.001,
				RemainingAmount:      0.001,
				AverageExecutedPrice: 70000,
				Cost:                 70,
				CostAsset:            currency.USDT,
				Fee:                  0.14,
				FeeAsset:             currency.USDT,
				Date:                 time.UnixMilli(1744150000173),
				LastUpdated:          time.UnixMilli(1744150100174),
			},
		},
		{
			a:       asset.CrossMargin,
			orderID: "1234",
			exp: &order.Detail{
				Exchange:        e.Name,
				OrderID:         "1234",
				Side:            order.Buy,
				Type:            order.Limit,
				Status:          order.Cancelled,
				Pair:            getPair(t, asset.CrossMargin),
				AssetType:       asset.CrossMargin,
				Price:           60000,
				Amount:          0.001,
				RemainingAmount: 0.001,
				CostAsset:       currency.USDT,
				FeeAsset:        currency.BTC,
				Date:            time.UnixMilli(1744150000273),
				LastUpdated:     time.UnixMilli(1744150300274),
			},
		},
		{
			// Gate finishes a partly filled IOC order as ioc, a status the order package has no name for
			a:       asset.USDTMarginedFutures,
			orderID: "15675402",
			exp: &order.Detail{
				Exchange:             e.Name,
				OrderID:              "15675402",
				ClientOrderID:        "t-usdt-market",
				Side:                 order.Short,
				Type:                 order.Market,
				Status:               order.Cancelled,
				TimeInForce:          order.ImmediateOrCancel,
				ReduceOnly:           true,
				Pair:                 getPair(t, asset.USDTMarginedFutures),
				AssetType:            asset.USDTMarginedFutures,
				SettlementCurrency:   currency.USDT,
				Amount:               10,
				ContractAmount:       10,
				ExecutedAmount:       6,
				RemainingAmount:      4,
				AverageExecutedPrice: 31500,
				Date:                 time.UnixMilli(1744150000123),
				LastUpdated:          time.UnixMilli(1744150000125),
			},
		},
		{
			a:       asset.CoinMarginedFutures,
			orderID: "15675394",
			exp: &order.Detail{
				Exchange:           e.Name,
				OrderID:            "15675394",
				ClientOrderID:      "t-my-custom-id",
				Side:               order.Long,
				Type:               order.Limit,
				Status:             order.Cancelled,
				TimeInForce:        order.GoodTillCancel,
				Pair:               getPair(t, asset.CoinMarginedFutures),
				AssetType:          asset.CoinMarginedFutures,
				SettlementCurrency: currency.BTC,
				Price:              3765,
				Amount:             6024,
				ContractAmount:     6024,
				RemainingAmount:    6024,
				Date:               time.Unix(1546569968, 0),
				LastUpdated:        time.UnixMilli(1546569970123),
			},
		},
		{
			a:       asset.DeliveryFutures,
			orderID: "123456",
			exp: &order.Detail{
				Exchange:           e.Name,
				OrderID:            "123456",
				ClientOrderID:      "t-my-custom-id",
				Side:               order.Long,
				Type:               order.Limit,
				Status:             order.Open,
				TimeInForce:        order.GoodTillCancel,
				Pair:               getPair(t, asset.DeliveryFutures),
				AssetType:          asset.DeliveryFutures,
				SettlementCurrency: currency.USDT,
				Price:              3765,
				Amount:             6024,
				ContractAmount:     6024,
				RemainingAmount:    6024,
				Date:               time.Unix(1546569968, 0),
				LastUpdated:        time.Unix(1546569968, 0),
			},
		},
		{
			// Gate sizes a sell negative
			a:       asset.Options,
			orderID: "1234",
			exp: &order.Detail{
				Exchange:        e.Name,
				OrderID:         "1234",
				Side:            order.Short,
				Type:            order.Limit,
				Status:          order.Open,
				TimeInForce:     order.GoodTillCancel,
				Pair:            getPair(t, asset.Options),
				AssetType:       asset.Options,
				Price:           100,
				Amount:          1,
				RemainingAmount: 1,
				Date:            time.Unix(1631763361, 0),
				LastUpdated:     time.Unix(1631763361, 0),
			},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			got, err := e.GetOrderInfo(t.Context(), tc.orderID, getPair(t, tc.a), tc.a)
			require.NoError(t, err, "GetOrderInfo must not error")
			if mockTests {
				assert.Equal(t, tc.exp, got, "GetOrderInfo should return the order")
				return
			}
			assert.Equal(t, tc.orderID, got.OrderID, "GetOrderInfo should return the requested order")
		})
	}
}

var mockedTickerTestCases = []struct {
	name         string
	asset        asset.Item
	pair         currency.Pair
	expectedPath string
}{
	{
		name:         "mocked USDT margined futures mark and index prices",
		asset:        asset.USDTMarginedFutures,
		pair:         currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter),
		expectedPath: "/api/v4/futures/usdt/tickers",
	},
	{
		name:         "mocked coin margined futures mark and index prices",
		asset:        asset.CoinMarginedFutures,
		pair:         currency.NewPairWithDelimiter("BTC", "USD", currency.UnderscoreDelimiter),
		expectedPath: "/api/v4/futures/btc/tickers",
	},
	{
		name:         "mocked delivery futures mark and index prices",
		asset:        asset.DeliveryFutures,
		pair:         currency.NewPairWithDelimiter("BTC", "USDT_20261225", currency.UnderscoreDelimiter),
		expectedPath: "/api/v4/delivery/usdt/tickers",
	},
	{
		name:         "mocked options mark and index prices",
		asset:        asset.Options,
		pair:         currency.NewPairWithDelimiter("BTC", "USDT-20261225-100000-C", currency.UnderscoreDelimiter),
		expectedPath: "/api/v4/options/tickers",
	},
}

// newFixedResponseExchange returns an exchange whose REST requests reach a server answering path with body, signing
// with placeholder credentials
func newFixedResponseExchange(t *testing.T, path, body string) *Exchange {
	t.Helper()
	return newRoutedExchange(t, map[string]string{path: body})
}

// newRoutedExchange returns an exchange whose REST requests reach a server answering each path with its body and every
// other path with a server error, signing with placeholder credentials
func newRoutedExchange(t *testing.T, routes map[string]string) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	ex.SkipAuthCheck = true
	require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := routes[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			body = `{"label":"SERVER_ERROR","message":"unavailable"}`
		}
		_, err := fmt.Fprint(w, body)
		assert.NoError(t, err, "writing the response should not error")
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	for _, u := range []exchange.URL{exchange.RestSpot, exchange.RestFutures} {
		require.NoError(t, ex.API.Endpoints.SetRunningURL(u.String(), server.URL), "SetRunningURL must not error")
	}
	return ex
}

// newIsolatedExchange returns an exchange set up as the shared one is but named after the test, so the ticker,
// orderbook, account and limits stores its wrapper calls write cannot collide with another test's
func newIsolatedExchange(t *testing.T) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	if mockTests {
		ex.API.AuthenticatedSupport = true
		ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
		require.NoError(t, testexch.MockHTTPInstance(ex), "MockHTTPInstance must not error")
		require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
		require.NoError(t, storeTestPairs(ex), "storeTestPairs must not error")
	} else {
		ensureTestPairs(t)
		ex.CurrencyPairs.Load(&e.CurrencyPairs)
		if apiCredentials.Key != "" && apiCredentials.Secret != "" {
			ex.API.AuthenticatedSupport = true
			ex.SetCredentials(apiCredentials)
		}
	}
	ex.Name = t.Name()
	return ex
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTicker(t.Context(), getPair(t, asset.Spot), asset.Binary)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateTicker must reject an unsupported asset")
	_, err = e.UpdateTicker(t.Context(), currency.NewPairWithDelimiter("BTC", "USDT-20260925-1-P", currency.UnderscoreDelimiter), asset.Options)
	require.ErrorIs(t, err, ticker.ErrTickerNotFound, "UpdateTicker must report an options contract its underlying's tickers do not list")
	_, err = newFixedResponseExchange(t, "/api/v4/futures/usdt/tickers", `[]`).UpdateTicker(t.Context(), currency.NewBTCUSDT(), asset.USDTMarginedFutures)
	require.ErrorIs(t, err, errNoTickerData, "UpdateTicker must reject an empty futures ticker list")

	t.Run("all supported assets", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			ex := newIsolatedExchange(t)
			btcusdtSpot := func() *ticker.Price {
				return &ticker.Price{Last: 67502.7, High: 67842.2, Low: 66610, Bid: 67502.6, Ask: 67502.7, BaseVolume: 4322.4404579762, QuoteVolume: 290264968.57010812722}
			}
			for _, tc := range []struct {
				a   asset.Item
				exp *ticker.Price
			}{
				{a: asset.Spot, exp: btcusdtSpot()},
				{a: asset.Margin, exp: btcusdtSpot()},
				{a: asset.CrossMargin, exp: btcusdtSpot()},
				{
					// The base volume is the contract count of volume_24h times the contract size, which beats the
					// truncated volume_24h_base
					a:   asset.USDTMarginedFutures,
					exp: &ticker.Price{Last: 67477.9, High: 67815.7, Low: 66580.1, BaseVolume: 32723.2988, QuoteVolume: 2196541426, MarkPrice: 67478, IndexPrice: 67503.92},
				},
				{
					a:   asset.CoinMarginedFutures,
					exp: &ticker.Price{Last: 67295.7, High: 67691.7, Low: 66490.4, BaseVolume: 14, QuoteVolume: 964612, MarkPrice: 67501.33, IndexPrice: 67499.05},
				},
				{
					a:   asset.DeliveryFutures,
					exp: &ticker.Price{Last: 67271.7, High: 68814.8, Low: 66355.1, BaseVolume: 7, QuoteVolume: 520353, MarkPrice: 67148.37, IndexPrice: 66878.24},
				},
				{
					a:   asset.Options,
					exp: &ticker.Price{Bid: 3408, BidSize: 455, Ask: 3752, AskSize: 2000, MarkPrice: 3597.4, IndexPrice: 67503.71},
				},
			} {
				t.Run(tc.a.String(), func(t *testing.T) {
					t.Parallel()
					started := time.Now()
					result, err := ex.UpdateTicker(t.Context(), getPair(t, tc.a), tc.a)
					require.NoError(t, err, "UpdateTicker must not error")
					// Gate's tickers carry no time, so they are stamped when stored
					assert.WithinRange(t, result.LastUpdated, started, time.Now(), "UpdateTicker should stamp the ticker when storing it")
					tc.exp.LastUpdated = result.LastUpdated
					tc.exp.Pair = getPair(t, tc.a)
					tc.exp.ExchangeName = ex.Name
					tc.exp.AssetType = tc.a
					assert.Equal(t, tc.exp, result, "UpdateTicker should return every field of the ticker")
				})
			}
			return
		}
		for _, a := range e.GetAssetTypes(false) {
			t.Run(a.String(), func(t *testing.T) {
				t.Parallel()
				pair := getPair(t, a)
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Setup must not error")
				testexch.UpdatePairsOnce(t, ex)
				// Isolate the cache so another test cannot supply the ticker being checked.
				ex.Name = t.Name()
				got, err := ex.UpdateTicker(t.Context(), pair, a)
				require.NoError(t, err, "UpdateTicker must not error")
				require.NotNil(t, got, "live ticker must not be nil")
				switch a {
				case asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.DeliveryFutures, asset.Options:
					if a == asset.Options {
						// Out-of-the-money options can have a zero mark price near expiry.
						assert.GreaterOrEqual(t, got.MarkPrice, 0.0, "live options mark price should be non-negative")
					} else {
						assert.Positive(t, got.MarkPrice, "live ticker mark price should be positive")
					}
					assert.Positive(t, got.IndexPrice, "live ticker index price should be positive")
				}
			})
		}
	})

	for _, tc := range mockedTickerTestCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			ex.Name = t.Name()
			if tc.asset == asset.Options {
				require.NoError(t, ex.UpdatePairs(currency.Pairs{tc.pair}, tc.asset, true), "mocked options pair must be enabled")
			}

			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method, "ticker request method should be GET")
				assert.Equal(t, tc.expectedPath, r.URL.Path, "ticker request path should match the asset endpoint")
				if tc.asset == asset.Options {
					assert.Equal(t, "BTC_USDT", r.URL.Query().Get("underlying"), "options underlying should match")
					_, err := fmt.Fprintf(w, `[{"name":%q,"last_price":"118.4","mark_price":"118.35","index_price":"100000.25"}]`, tc.pair.String())
					assert.NoError(t, err, "mocked response should be written")
					return
				}
				assert.Equal(t, tc.pair.String(), r.URL.Query().Get("contract"), "ticker request contract should match the pair")
				_, err := fmt.Fprintf(w, `[{"contract":%q,"last":"118.4","low_24h":"99.2","high_24h":"132.5","volume_24h_base":"5526","volume_24h_quote":"1665006","mark_price":"118.35","index_price":"118.36"}]`, tc.pair.String())
				assert.NoError(t, err, "mocked ticker response should be written")
			}))

			require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")

			got, err := ex.UpdateTicker(t.Context(), tc.pair, tc.asset)
			require.NoError(t, err, "mocked ticker retrieval must succeed")
			assert.Equal(t, 118.35, got.MarkPrice, "ticker mark price should match the mocked response")
			if tc.asset == asset.Options {
				assert.Equal(t, 100000.25, got.IndexPrice, "options index price should match")
			} else {
				assert.Equal(t, 118.36, got.IndexPrice, "ticker index price should match the mocked response")
			}
		})
	}

	if mockTests {
		return // The volume checks below pick the busiest live pairs, which the recorded tickers do not cover
	}
	// presence alone cannot tell base from quote, so compare against the source fields. Both are
	// picked by volume rather than positionally, since InEpsilon errors outright on a zero
	// expected value and an untraded pair would decide the test on liquidity, not on the mapping
	spotTicks, err := e.GetTickers(t.Context(), currency.EMPTYPAIR, "")
	require.NoError(t, err, "GetTickers must not error")
	require.NotEmpty(t, spotTicks, "GetTickers must return tickers")
	// a pair quoting near parity reports both volumes alike and cannot tell a correct mapping from
	// a swapped one, so those are filtered out before picking rather than asserted against
	spotTicks = slices.DeleteFunc(spotTicks, func(tk *TickerResponse) bool {
		return tk.BaseVolume.Float64() == 0 || tk.BaseVolume.Float64() == tk.QuoteVolume.Float64()
	})
	require.NotEmpty(t, spotTicks, "at least one spot pair must report base and quote volumes that differ")
	busiestSpot := slices.MaxFunc(spotTicks, func(a, b *TickerResponse) int {
		return cmp.Compare(a.QuoteVolume.Float64(), b.QuoteVolume.Float64())
	})
	p := busiestSpot.CurrencyPair
	tick, err := e.UpdateTicker(t.Context(), p, asset.Spot)
	require.NoError(t, err, "UpdateTicker must not error")
	assert.InEpsilonf(t, busiestSpot.BaseVolume.Float64(), tick.BaseVolume, 0.05, "UpdateTicker should take the spot base volume for %s from base_volume", p)
	assert.InEpsilonf(t, busiestSpot.QuoteVolume.Float64(), tick.QuoteVolume, 0.05, "UpdateTicker should take the spot quote volume for %s from quote_volume", p)

	futuresTicks, err := e.GetFuturesTickers(t.Context(), currency.USDT, currency.EMPTYPAIR)
	require.NoError(t, err, "GetFuturesTickers must not error")
	require.NotEmpty(t, futuresTicks, "GetFuturesTickers must return tickers")
	// a multiplier of 1 reports the contract count and the base volume identically, so those
	// contracts are filtered out before picking the busiest rather than asserted against
	futuresTicks = slices.DeleteFunc(futuresTicks, func(tk *FuturesTickerResponse) bool {
		return tk.Volume24HourBase.Float64() == 0 || tk.Volume24HourBase.Float64() == tk.Volume24Hour.Float64()
	})
	require.NotEmpty(t, futuresTicks, "at least one futures contract must report a base volume that differs from its contract count")
	busiestFutures := slices.MaxFunc(futuresTicks, func(a, b *FuturesTickerResponse) int {
		return cmp.Compare(a.Volume24HourQuote.Float64(), b.Volume24HourQuote.Float64())
	})
	fp := busiestFutures.Contract
	futuresTick, err := e.UpdateTicker(t.Context(), fp, asset.USDTMarginedFutures)
	require.NoError(t, err, "UpdateTicker must not error")
	assert.InEpsilonf(t, busiestFutures.Volume24HourBase.Float64(), futuresTick.BaseVolume, 0.05,
		"UpdateTicker should report a base volume for %s of the right order, not the contract count in volume_24h", fp)
	assert.InEpsilonf(t, busiestFutures.Volume24HourQuote.Float64(), futuresTick.QuoteVolume, 0.05,
		"UpdateTicker should take the quote volume for %s from volume_24h_quote", fp)
}

func TestFuturesBaseVolume(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		tick FuturesTickerResponse
		exp  float64
	}{
		{"base truncated a whole unit off the product", FuturesTickerResponse{Volume24HourBase: 52438, Volume24Hour: 524387795, QuantoMultiplier: 0.0001}, 52438.7795},
		{"base truncated the contract away entirely", FuturesTickerResponse{Volume24Hour: 5128, QuantoMultiplier: 0.0001}, 0.5128},
		{"decimal lots: volume_24h dropped a part contract, so base is the better bound", FuturesTickerResponse{Volume24HourBase: 8819700, Volume24Hour: 8819, QuantoMultiplier: 1000}, 8819700},
		{"no contract size, as coin margined and delivery report it", FuturesTickerResponse{Volume24HourBase: 30, Volume24Hour: 2352026}, 30},
		{"nothing traded", FuturesTickerResponse{}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.InDelta(t, tc.exp, futuresBaseVolume(&tc.tick), 1e-9, "futuresBaseVolume should return the 24h volume in the base currency")
		})
	}
}

// TestFuturesTickerUnmarshal pins the tags futuresBaseVolume reads: constructing the fields
// directly cannot catch one of them being renamed on the wire
func TestFuturesTickerUnmarshal(t *testing.T) {
	t.Parallel()
	var tk FuturesTickerResponse
	require.NoError(t, json.Unmarshal([]byte(`{"contract":"UB_USDT","volume_24h":"8819","volume_24h_base":"8819700","volume_24h_quote":"1234","quanto_multiplier":"1000"}`), &tk), "Unmarshal must not error")
	assert.Equal(t, "UB_USDT", tk.Contract.String(), "Contract should unmarshal")
	assert.Equal(t, 8819.0, tk.Volume24Hour.Float64(), "volume_24h should unmarshal")
	assert.Equal(t, 8819700.0, tk.Volume24HourBase.Float64(), "volume_24h_base should unmarshal")
	assert.Equal(t, 1000.0, tk.QuantoMultiplier.Float64(), "quanto_multiplier should unmarshal")
	assert.Equal(t, 8819700.0, futuresBaseVolume(&tk), "futuresBaseVolume should keep the explicit base volume when volume_24h dropped a part contract")
}

func TestListSpotCurrencies(t *testing.T) {
	t.Parallel()
	result, err := e.ListSpotCurrencies(t.Context())
	require.NoError(t, err, "ListSpotCurrencies must not error")
	if mockTests {
		require.Len(t, result, 5, "ListSpotCurrencies must return every recorded currency")
		exp := &CurrencyInfoResponse{
			Currency:        currency.NewCode("STPT"),
			Name:            "StandardTokenizationProtocol",
			Delisted:        true,
			DepositDisabled: true,
			TradeDisabled:   true,
			Chain:           "ETH",
			Chains:          []*SpotCurrencyChain{{Name: "ETH", Address: "0xDe7D85157d9714EADf595045CC12Ca4A5f3E2aDb", DepositDisabled: true}},
			Category:        []string{},
		}
		assert.Equal(t, exp, result[0], "ListSpotCurrencies should decode every field")
		return
	}
	assert.NotEmpty(t, result, "ListSpotCurrencies should return currencies")
}

func TestGetCurrencyDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrencyDetail(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetCurrencyDetail must reject an empty currency")

	result, err := e.GetCurrencyDetail(t.Context(), currency.BTC)
	require.NoError(t, err, "GetCurrencyDetail must not error")
	if mockTests {
		exp := &CurrencyInfoResponse{
			Currency:    currency.BTC,
			Name:        "Bitcoin",
			Chain:       "BTC",
			Chains:      []*SpotCurrencyChain{{Name: "BTC"}, {Name: "BSC", Address: "0x7130d2a12b9bcbfae4f2634d864a1ee1ce3ead9c"}},
			TotalSupply: "21000000",
			MarketCap:   1353080333764.8,
			Category:    []string{},
		}
		assert.Equal(t, exp, result, "GetCurrencyDetail should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCurrencyDetail should return a result")
}

func TestListSpotCurrencyPairs(t *testing.T) {
	t.Parallel()
	result, err := e.ListSpotCurrencyPairs(t.Context())
	require.NoError(t, err, "ListSpotCurrencyPairs must not error")
	if mockTests {
		require.Len(t, result, 6, "ListSpotCurrencyPairs must return every recorded currency pair")
		exp := &CurrencyPairDetailResponse{
			ID:                            getPair(t, asset.Spot),
			Base:                          currency.BTC,
			BaseName:                      "Aleph.im",
			Quote:                         currency.USDT,
			QuoteName:                     "Tether",
			Fee:                           0.2,
			MinBaseAmount:                 0.01,
			MinQuoteAmount:                3,
			MaxQuoteAmount:                5000000,
			AmountPrecision:               2,
			PricePrecision:                5,
			TradeStatus:                   "tradable",
			SellStart:                     types.Time(time.Unix(1608264000, 0)),
			BuyStart:                      types.Time(time.Unix(1608350400, 0)),
			Type:                          "normal",
			TradeURL:                      "https://www.gate.com/trade/ALEPH_USDT",
			MaximumQuoteRisePercentage:    0.4,
			MaximumQuoteDeclinePercentage: 0.4,
			Slippage:                      0.1,
			MarketOrderMaxStock:           7228044.81,
			MarketOrderMaxMoney:           200000,
		}
		assert.Equal(t, exp, result[0], "ListSpotCurrencyPairs should decode every field")
		return
	}
	assert.NotEmpty(t, result, "ListSpotCurrencyPairs should return currency pairs")
}

func TestGetCurrencyPairDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrencyPairDetail(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCurrencyPairDetail must reject an empty currency pair")

	unifiedMarket := currency.NewBTCUSD().Format(currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter})
	for _, tc := range []struct {
		pair currency.Pair
		exp  *CurrencyPairDetailResponse
	}{
		{
			pair: getPair(t, asset.Spot),
			exp: &CurrencyPairDetailResponse{
				ID:                            getPair(t, asset.Spot),
				Base:                          currency.BTC,
				BaseName:                      "Bitcoin",
				Quote:                         currency.USDT,
				QuoteName:                     "Tether",
				Fee:                           0.2,
				MinBaseAmount:                 0.000001,
				MinQuoteAmount:                3,
				MaxBaseAmount:                 100,
				MaxQuoteAmount:                5000000,
				AmountPrecision:               6,
				PricePrecision:                1,
				TradeStatus:                   "tradable",
				Type:                          "normal",
				TradeURL:                      "https://www.gate.com/trade/BTC_USDT",
				MaximumQuoteRisePercentage:    0.08,
				MaximumQuoteDeclinePercentage: 0.08,
				Slippage:                      0.03,
				MarketOrderMaxStock:           57.94,
				MarketOrderMaxMoney:           5000000,
			},
		},
		{
			// Unified markets list the quote currencies they accept in trade_quotes
			pair: unifiedMarket,
			exp: &CurrencyPairDetailResponse{
				ID:                            unifiedMarket,
				Base:                          currency.BTC,
				BaseName:                      "Bitcoin",
				Quote:                         currency.USD,
				QuoteName:                     "US Dollar",
				TradeQuotes:                   []currency.Code{currency.NewCode("RLUSD"), currency.USD, currency.NewCode("USD1"), currency.USDC},
				Fee:                           0.2,
				MinBaseAmount:                 0.000001,
				MinQuoteAmount:                1,
				MaxBaseAmount:                 100,
				MaxQuoteAmount:                5000000,
				AmountPrecision:               6,
				PricePrecision:                1,
				TradeStatus:                   "tradable",
				Type:                          "normal",
				TradeURL:                      "https://www.gate.com/trade/BTC_USD",
				MaximumQuoteRisePercentage:    0.25,
				MaximumQuoteDeclinePercentage: 0.25,
				Slippage:                      0.06,
				MarketOrderMaxStock:           1,
				MarketOrderMaxMoney:           100000,
			},
		},
	} {
		t.Run(tc.pair.String(), func(t *testing.T) {
			t.Parallel()
			result, err := e.GetCurrencyPairDetail(t.Context(), tc.pair)
			require.NoError(t, err, "GetCurrencyPairDetail must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetCurrencyPairDetail should decode every field")
				return
			}
			assert.Equal(t, tc.pair, result.ID, "GetCurrencyPairDetail should return the requested currency pair")
		})
	}
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, err := e.GetTickers(t.Context(), getPair(t, asset.Spot), "utc2")
	require.ErrorIs(t, err, errInvalidTimezone, "GetTickers must reject an unsupported timezone")

	for _, tc := range []struct {
		timezone string
		exp      *TickerResponse
	}{
		{
			exp: &TickerResponse{
				CurrencyPair:     getPair(t, asset.Spot),
				Last:             67502.7,
				LowestAsk:        67502.7,
				LowestSize:       9.03203,
				HighestBid:       67502.6,
				HighestSize:      12.1675,
				ChangePercentage: 0.37,
				BaseVolume:       4322.4404579762,
				QuoteVolume:      290264968.57010812722,
				High24Hour:       67842.2,
				Low24Hour:        66610,
			},
		},
		{
			// Only the all timezone returns both the UTC+0 and UTC+8 changes
			timezone: allTimeZones,
			exp: &TickerResponse{
				CurrencyPair:     getPair(t, asset.Spot),
				Last:             82177.8,
				LowestAsk:        82170.4,
				LowestSize:       2.799905,
				HighestBid:       82170.3,
				HighestSize:      4.246692,
				ChangePercentage: -1.89,
				ChangeUTC0:       0.4,
				ChangeUTC8:       -0.66,
				BaseVolume:       8949.156124,
				QuoteVolume:      742123076.3466114,
				High24Hour:       84352.2,
				Low24Hour:        81040.5,
			},
		},
	} {
		t.Run("timezone "+tc.timezone, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetTickers(t.Context(), getPair(t, asset.Spot), tc.timezone)
			require.NoError(t, err, "GetTickers must not error")
			require.Len(t, result, 1, "GetTickers must return one ticker for a currency pair")
			if mockTests {
				assert.Equal(t, tc.exp, result[0], "GetTickers should decode every field")
				return
			}
			assert.Equal(t, getPair(t, asset.Spot), result[0].CurrencyPair, "GetTickers should return the requested currency pair")
		})
	}

	result, err := e.GetTickers(t.Context(), currency.EMPTYPAIR, "")
	require.NoError(t, err, "GetTickers must not error for every currency pair")
	if mockTests {
		require.Len(t, result, 5, "GetTickers must return every recorded ticker")
		exp := &TickerResponse{
			CurrencyPair:     currency.NewPairWithDelimiter("SUSHI3L", "USDT", currency.UnderscoreDelimiter),
			Last:             0.00113,
			LowestAsk:        0.001127,
			HighestBid:       0.001124,
			ChangePercentage: -8.27,
			BaseVolume:       10218896.34,
			QuoteVolume:      11910.21673754,
			High24Hour:       0.001256,
			Low24Hour:        0.001104,
			ETFNetValue:      0.00113387,
			ETFPreNetValue:   0.00115756,
			ETFPreTimestamp:  types.Time(time.Unix(1775404800, 0)),
			ETFLeverage:      3.041778253883235,
		}
		assert.Equal(t, exp, result[2], "GetTickers should decode leveraged token fields")
		return
	}
	assert.NotEmpty(t, result, "GetTickers should return tickers")
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetTicker(t.Context(), currency.EMPTYPAIR, utc8TimeZone)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetTicker must reject an empty currency pair")

	_, err = newFixedResponseExchange(t, "/api/v4/spot/tickers", `[]`).GetTicker(t.Context(), currency.NewBTCUSDT(), "")
	require.ErrorIs(t, err, errNoTickerData, "GetTicker must reject an empty ticker list")

	result, err := e.GetTicker(t.Context(), getPair(t, asset.Spot), utc8TimeZone)
	require.NoError(t, err, "GetTicker must not error")
	if mockTests {
		exp := &TickerResponse{
			CurrencyPair:     getPair(t, asset.Spot),
			Last:             67197,
			LowestAsk:        67198.1,
			LowestSize:       0.316979,
			HighestBid:       67198,
			HighestSize:      12.696796,
			ChangePercentage: -0.64,
			BaseVolume:       10980.383815,
			QuoteVolume:      737010383.3594575,
			High24Hour:       68405.5,
			Low24Hour:        65996.1,
		}
		assert.Equal(t, exp, result, "GetTicker should decode every field")
		return
	}
	assert.Equal(t, getPair(t, asset.Spot), result.CurrencyPair, "GetTicker should return the requested currency pair")
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderbook(t.Context(), currency.EMPTYPAIR, "0.1", 10, false)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOrderbook must reject an empty currency pair")

	result, err := e.GetOrderbook(t.Context(), getPair(t, asset.Spot), "0.1", 10, false)
	require.NoError(t, err, "GetOrderbook must not error")
	if mockTests {
		exp := &OrderbookResponse{
			Current: types.Time(time.UnixMilli(1774968317360)),
			Update:  types.Time(time.UnixMilli(1774968317359)),
			Asks: OrderbookLevels{
				{Price: 67197.1, Amount: 0.485259},
				{Price: 67197.9, Amount: 0.016632},
				{Price: 67198, Amount: 0.0148},
				{Price: 67199, Amount: 0.0155},
				{Price: 67199.1, Amount: 0.024881},
			},
			Bids: OrderbookLevels{
				{Price: 67197, Amount: 0.442669},
				{Price: 67196.9, Amount: 0.01},
				{Price: 67196.7, Amount: 0.001547},
				{Price: 67196.1, Amount: 0.012274},
				{Price: 67196, Amount: 0.025678},
			},
		}
		assert.Equal(t, exp, result, "GetOrderbook should decode every field")
		return
	}
	assert.NotEmpty(t, result.Asks, "GetOrderbook should return asks")
}

func TestGetMarketTrades(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetMarketTrades(t.Context(), getPair(t, asset.Spot), "", true, endTime, startTime, 1, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMarketTrades must reject a start time after the end time")

	_, err = e.GetMarketTrades(t.Context(), currency.EMPTYPAIR, "", true, startTime, endTime, 1, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetMarketTrades must reject an empty currency pair")

	for _, tc := range []struct {
		name     string
		from, to time.Time
		limit    uint64
	}{
		{name: "latest"},
		{name: "time range", from: startTime, to: endTime, limit: 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetMarketTrades(t.Context(), getPair(t, asset.Spot), "", true, tc.from, tc.to, 1, tc.limit)
			require.NoError(t, err, "GetMarketTrades must not error")
			if mockTests {
				require.Len(t, result, 5, "GetMarketTrades must return every recorded trade")
				exp := &TradeResponse{
					ID:                "192742010",
					CreateTimeSeconds: types.Time(time.Unix(1775423972, 0)),
					CreateTime:        types.Time(time.Unix(0, 1775423972061432000)),
					CurrencyPair:      "BTC_USDT",
					Side:              "buy",
					Amount:            0.029077,
					Price:             67502.7,
					SequenceID:        "192742010",
				}
				assert.Equal(t, exp, result[0], "GetMarketTrades should decode every field")
				return
			}
			assert.NotNil(t, result, "GetMarketTrades should return a result")
		})
	}
}

func TestCandlestickUnmarshalJSON(t *testing.T) {
	t.Parallel()
	data := []byte(`[["1738108800","229534412.73508700","103734.3","104779.9","101336.6","101343.8","2232.94510000","true"],["1738195200","178316032.62306100","104718.6","106467.1","103286.4","103734.4","1695.00787000","true"],["1738281600","231315376.16747100","102431","106042.7","101555.9","104718.6","2228.03609000","true"]]`)
	var targets []CandlestickResponse
	err := json.Unmarshal(data, &targets)
	require.NoError(t, err)
	require.Len(t, targets, 3)
	assert.Equal(t, CandlestickResponse{
		Timestamp:           types.Time(time.Unix(1738108800, 0)),
		QuoteCurrencyVolume: 229534412.73508700,
		ClosePrice:          103734.3,
		HighestPrice:        104779.9,
		LowestPrice:         101336.6,
		OpenPrice:           101343.8,
		BaseCurrencyAmount:  2232.94510000,
		WindowClosed:        true,
	}, targets[0])
}

func TestGetCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := e.GetCandlesticks(t.Context(), currency.EMPTYPAIR, 0, time.Time{}, time.Time{}, kline.OneDay)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCandlesticks must reject an empty currency pair")

	_, err = e.GetCandlesticks(t.Context(), getPair(t, asset.Spot), 0, time.Time{}, time.Time{}, kline.FiveDay)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "GetCandlesticks must reject an unsupported interval")

	result, err := e.GetCandlesticks(t.Context(), getPair(t, asset.Spot), 0, time.Time{}, time.Time{}, kline.OneDay)
	require.NoError(t, err, "GetCandlesticks must not error")
	if mockTests {
		require.Len(t, result, 5, "GetCandlesticks must return every recorded candle")
		exp := &CandlestickResponse{
			Timestamp:           types.Time(time.Unix(1766361600, 0)),
			QuoteCurrencyVolume: 901846017.17068550,
			ClosePrice:          88622.1,
			HighestPrice:        90586.3,
			LowestPrice:         87899.6,
			OpenPrice:           88655.4,
			BaseCurrencyAmount:  10108.46374900,
			WindowClosed:        true,
		}
		assert.Equal(t, exp, result[0], "GetCandlesticks should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCandlesticks should return candles")
}

func TestGetAccountBatchFeeRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountBatchFeeRates(t.Context(), nil)
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "GetAccountBatchFeeRates must reject empty currency pairs")

	_, err = e.GetAccountBatchFeeRates(t.Context(), make([]string, 51))
	require.ErrorIs(t, err, errTooManyCurrencyPairs, "GetAccountBatchFeeRates must reject more than 50 currency pairs")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountBatchFeeRates(t.Context(), []string{"BTC_USDT", "ETH_USDT"})
	require.NoError(t, err, "GetAccountBatchFeeRates must not error")
	if mockTests {
		exp := map[string]*SpotTradingFeeRateResponse{
			"BTC_USDT": {
				CurrencyPair: getPair(t, asset.Spot),
				UserID:       10001,
				TakerFee:     0.002,
				MakerFee:     0.002,
				RPIMakerFee:  -0.00175,
				LoanFee:      0.18,
				PointType:    "1",
				DebitFee:     3,
				RPIMMLevel:   2,
			},
			"ETH_USDT": {
				CurrencyPair: currency.NewPairWithDelimiter("ETH", "USDT", currency.UnderscoreDelimiter),
				UserID:       10001,
				TakerFee:     0.001,
				MakerFee:     0.0009,
				RPIMakerFee:  -0.00175,
				GTDiscount:   true,
				GTTakerFee:   0.0008,
				GTMakerFee:   0.0007,
				LoanFee:      0.18,
				PointType:    "1",
				DebitFee:     1,
			},
		}
		assert.Equal(t, exp, result, "GetAccountBatchFeeRates should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetAccountBatchFeeRates should return fee rates")
}

func TestGetSpotAccounts(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpotAccounts(t.Context(), currency.BTC)
	require.NoError(t, err, "GetSpotAccounts must not error")
	if mockTests {
		exp := []*SpotAccountResponse{{Currency: currency.BTC, Available: 968.8, Locked: 0.5, UpdateID: 98}}
		assert.Equal(t, exp, result, "GetSpotAccounts should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSpotAccounts should return a result")
}

func TestGetSpotAccountBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotAccountBook(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpotAccountBook must reject a nil request")

	startTime, endTime := getTime()
	_, err = e.GetSpotAccountBook(t.Context(), &GetSpotAccountBookRequest{From: endTime.Add(-time.Hour * 24 * 31), To: endTime})
	require.ErrorIs(t, err, errTimeRangeTooLong, "GetSpotAccountBook must reject a range longer than 30 days")

	_, err = e.GetSpotAccountBook(t.Context(), &GetSpotAccountBookRequest{From: endTime, To: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSpotAccountBook must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpotAccountBook(t.Context(), &GetSpotAccountBookRequest{Currency: currency.BTC, From: startTime, To: endTime, Page: 1, Limit: 10})
	require.NoError(t, err, "GetSpotAccountBook must not error")
	if mockTests {
		exp := []*SpotAccountBookResponse{{
			ID:       "123456",
			Time:     types.Time(time.UnixMilli(1744150000123)),
			Currency: currency.BTC,
			Change:   1.03,
			Balance:  4.59316525194,
			Type:     "margin_in",
			Code:     "101",
			Text:     "3815099",
		}}
		assert.Equal(t, exp, result, "GetSpotAccountBook should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSpotAccountBook should return a result")
}

func TestCreateBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CreateBatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "CreateBatchOrders must reject an empty batch")

	arg := CreateOrderRequest{
		Side:    order.Sell.Lower(),
		Amount:  0.001,
		Price:   12349,
		Account: asset.Spot,
		Type:    "limit",
	}
	_, err = e.CreateBatchOrders(t.Context(), []CreateOrderRequest{arg})
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "CreateBatchOrders must reject an order without text")

	arg.Text = "1234"
	_, err = e.CreateBatchOrders(t.Context(), []CreateOrderRequest{arg})
	require.ErrorIs(t, err, errInvalidTextPrefix, "CreateBatchOrders must reject text without the t- prefix")

	arg.Text = "t-1234"
	_, err = e.CreateBatchOrders(t.Context(), []CreateOrderRequest{arg})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CreateBatchOrders must reject an empty currency pair")

	arg.CurrencyPair = getPair(t, asset.Spot)
	_, err = e.CreateBatchOrders(t.Context(), []CreateOrderRequest{arg, {Text: "t-1235", CurrencyPair: arg.CurrencyPair, Side: arg.Side, Amount: 1, Price: 1, Account: asset.Margin}})
	require.ErrorIs(t, err, errDifferentAccount, "CreateBatchOrders must reject orders from different accounts")

	invalidSide := arg
	invalidSide.Side = "short"
	_, err = e.CreateBatchOrders(t.Context(), []CreateOrderRequest{invalidSide})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "CreateBatchOrders must reject a side other than buy or sell")

	invalidAccount := arg
	invalidAccount.Account = asset.Options
	_, err = e.CreateBatchOrders(t.Context(), []CreateOrderRequest{invalidAccount})
	require.ErrorIs(t, err, asset.ErrInvalidAsset, "CreateBatchOrders must reject an account other than spot, margin or cross margin")

	zeroAmount := arg
	zeroAmount.Amount = 0
	_, err = e.CreateBatchOrders(t.Context(), []CreateOrderRequest{zeroAmount})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "CreateBatchOrders must reject an order without an amount")

	limitWithoutPrice := arg
	limitWithoutPrice.Price = 0
	_, err = e.CreateBatchOrders(t.Context(), []CreateOrderRequest{limitWithoutPrice})
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "CreateBatchOrders must reject a limit order without a price")

	_, err = e.CreateBatchOrders(t.Context(), slices.Repeat([]CreateOrderRequest{arg}, 11))
	require.ErrorIs(t, err, errTooManyBatchOrders, "CreateBatchOrders must reject more than 10 orders on a currency pair")

	fivePairs := make([]CreateOrderRequest, 5)
	for i, base := range []currency.Code{currency.BTC, currency.ETH, currency.LTC, currency.XRP, currency.SOL} {
		fivePairs[i] = arg
		fivePairs[i].CurrencyPair = currency.NewPairWithDelimiter(base.String(), "USDT", currency.UnderscoreDelimiter)
	}
	_, err = e.CreateBatchOrders(t.Context(), fivePairs)
	require.ErrorIs(t, err, errTooManyBatchOrders, "CreateBatchOrders must reject more than 4 currency pairs")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateBatchOrders(t.Context(), []CreateOrderRequest{
		{
			Text:         "t-123123123",
			CurrencyPair: getPair(t, asset.Spot),
			Side:         order.Sell.Lower(),
			Amount:       0.001,
			Price:        12349,
			Account:      asset.Spot,
			Type:         "limit",
		},
		{
			Text:         "t-123123124",
			CurrencyPair: getPair(t, asset.Spot),
			Side:         order.Buy.Lower(),
			Amount:       1,
			Price:        1234567789,
			Account:      asset.Spot,
			Type:         "limit",
		},
	})
	require.NoError(t, err, "CreateBatchOrders must not error")
	if mockTests {
		exp := []*SpotOrderResponse{
			{
				OrderID:            "12332324",
				BatchOrderID:       "12332324",
				Text:               "t-123123123",
				AmendText:          "-",
				Succeeded:          true,
				CreateTimeSeconds:  types.Time(time.Unix(1744150000, 0)),
				UpdateTimeSeconds:  types.Time(time.Unix(1744150001, 0)),
				CreateTime:         types.Time(time.UnixMilli(1744150000123)),
				UpdateTime:         types.Time(time.UnixMilli(1744150001123)),
				CurrencyPair:       "BTC_USDT",
				Status:             statusOpen,
				Type:               "limit",
				Account:            "spot",
				Side:               "sell",
				Amount:             0.001,
				Price:              12349,
				TimeInForce:        gtcTIF,
				RemainingAmount:    0.001,
				FeeCurrency:        currency.USDT,
				RebatedFeeCurrency: currency.BTC,
				STPID:              10240,
				STPAction:          "cn",
				FinishAs:           "open",
			},
			{
				Text:       "t-123123124",
				ErrorLabel: "BALANCE_NOT_ENOUGH",
				Message:    "Not enough balance",
			},
		}
		assert.Equal(t, exp, result, "CreateBatchOrders should decode every field")
		return
	}
	assert.NotEmpty(t, result, "CreateBatchOrders should return order results")
}

func TestValidateSpotOrderPrice(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderType string
		price     float64
		err       error
	}{
		{orderType: "limit", price: 1},
		{orderType: "limit", err: limits.ErrPriceBelowMin},
		{err: limits.ErrPriceBelowMin}, // Gate defaults to a limit order
		{orderType: "market"},
		{orderType: "market", price: -1, err: limits.ErrPriceBelowMin},
	} {
		err := validateSpotOrderPrice(tc.orderType, tc.price)
		if tc.err != nil {
			assert.ErrorIsf(t, err, tc.err, "validateSpotOrderPrice should reject %q orders priced %v", tc.orderType, tc.price)
			continue
		}
		assert.NoErrorf(t, err, "validateSpotOrderPrice should accept %q orders priced %v", tc.orderType, tc.price)
	}
}

func TestGetSpotOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpotOpenOrders(t.Context(), 0, 0, "")
	require.NoError(t, err, "GetSpotOpenOrders must not error")
	if mockTests {
		exp := []*SpotOrdersDetailResponse{{
			CurrencyPair: getPair(t, asset.Spot),
			Total:        1,
			Orders: []*SpotOrderResponse{{
				OrderID:            "12332324",
				Text:               "t-123456",
				AmendText:          "-",
				CreateTimeSeconds:  types.Time(time.Unix(1744150000, 0)),
				UpdateTimeSeconds:  types.Time(time.Unix(1744150100, 0)),
				CreateTime:         types.Time(time.UnixMilli(1744150000123)),
				UpdateTime:         types.Time(time.UnixMilli(1744150100123)),
				CurrencyPair:       "BTC_USDT",
				Status:             statusOpen,
				Type:               "limit",
				Account:            "spot",
				Side:               "buy",
				Amount:             1,
				Price:              5.00032,
				TimeInForce:        gtcTIF,
				RemainingAmount:    0.5,
				FilledAmount:       0.5,
				AverageFillPrice:   5.00032,
				FeeDeducted:        0.005,
				FeeCurrency:        currency.BTC,
				FilledTotal:        2.50016,
				RebatedFeeCurrency: currency.USDT,
				STPAction:          "-",
				FinishAs:           "open",
			}},
		}}
		assert.Equal(t, exp, result, "GetSpotOpenOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSpotOpenOrders should return a result")
}

func TestSpotClosePositionWhenCrossCurrencyDisabled(t *testing.T) {
	t.Parallel()
	_, err := e.SpotClosePositionWhenCrossCurrencyDisabled(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SpotClosePositionWhenCrossCurrencyDisabled must reject a nil request")

	_, err = e.SpotClosePositionWhenCrossCurrencyDisabled(t.Context(), &ClosePositionRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "SpotClosePositionWhenCrossCurrencyDisabled must reject an empty currency pair")

	_, err = e.SpotClosePositionWhenCrossCurrencyDisabled(t.Context(), &ClosePositionRequest{CurrencyPair: getPair(t, asset.Spot)})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "SpotClosePositionWhenCrossCurrencyDisabled must reject an empty amount")

	_, err = e.SpotClosePositionWhenCrossCurrencyDisabled(t.Context(), &ClosePositionRequest{Amount: 0.1, CurrencyPair: getPair(t, asset.Spot)})
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "SpotClosePositionWhenCrossCurrencyDisabled must reject an empty price")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SpotClosePositionWhenCrossCurrencyDisabled(t.Context(), &ClosePositionRequest{
		Amount:       0.1,
		Price:        1234567384,
		CurrencyPair: getPair(t, asset.Spot),
	})
	require.NoError(t, err, "SpotClosePositionWhenCrossCurrencyDisabled must not error")
	if mockTests {
		exp := &SpotOrderResponse{
			OrderID:            "1852454420",
			Text:               "apiv4",
			AmendText:          "-",
			CreateTimeSeconds:  types.Time(time.Unix(1744150000, 0)),
			UpdateTimeSeconds:  types.Time(time.Unix(1744150000, 0)),
			CreateTime:         types.Time(time.UnixMilli(1744150000073)),
			UpdateTime:         types.Time(time.UnixMilli(1744150000074)),
			CurrencyPair:       "BTC_USDT",
			Status:             statusOpen,
			Type:               "limit",
			Account:            crossMarginAccount,
			Side:               "buy",
			Amount:             0.1,
			Price:              1234567384,
			TimeInForce:        gtcTIF,
			AutoRepay:          true,
			RemainingAmount:    0.1,
			FeeCurrency:        currency.BTC,
			RebatedFeeCurrency: currency.USDT,
			STPAction:          "-",
			FinishAs:           "open",
		}
		assert.Equal(t, exp, result, "SpotClosePositionWhenCrossCurrencyDisabled should decode every field")
		return
	}
	assert.NotNil(t, result, "SpotClosePositionWhenCrossCurrencyDisabled should return a result")
}

func TestPlaceSpotOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceSpotOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceSpotOrder must reject a nil request")

	arg := &CreateOrderRequest{Type: "limit"}
	_, err = e.PlaceSpotOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "PlaceSpotOrder must reject an empty currency pair")

	arg.CurrencyPair = getPair(t, asset.Spot)
	_, err = e.PlaceSpotOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "PlaceSpotOrder must reject an invalid side")

	arg.Side = order.Buy.Lower()
	_, err = e.PlaceSpotOrder(t.Context(), arg)
	require.ErrorIs(t, err, asset.ErrInvalidAsset, "PlaceSpotOrder must reject an unsupported account")

	arg.Account = asset.Spot
	_, err = e.PlaceSpotOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "PlaceSpotOrder must reject an empty amount")

	arg.Amount = 1
	arg.Price = -1
	_, err = e.PlaceSpotOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "PlaceSpotOrder must reject a negative price")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PlaceSpotOrder(t.Context(), &CreateOrderRequest{
		CurrencyPair: getPair(t, asset.Spot),
		Side:         order.Buy.Lower(),
		Amount:       1,
		Price:        900000,
		Account:      asset.Spot,
		Type:         "limit",
	})
	require.NoError(t, err, "PlaceSpotOrder must not error")
	if mockTests {
		exp := &SpotOrderResponse{
			OrderID:            "1852454420",
			Text:               "apiv4",
			AmendText:          "-",
			CreateTimeSeconds:  types.Time(time.Unix(1744150000, 0)),
			UpdateTimeSeconds:  types.Time(time.Unix(1744150000, 0)),
			CreateTime:         types.Time(time.UnixMilli(1744150000073)),
			UpdateTime:         types.Time(time.UnixMilli(1744150000074)),
			CurrencyPair:       "BTC_USDT",
			Status:             statusOpen,
			Type:               "limit",
			Account:            "spot",
			Side:               "buy",
			Amount:             1,
			Price:              900000,
			TimeInForce:        gtcTIF,
			RemainingAmount:    1,
			FeeCurrency:        currency.BTC,
			RebatedFeeCurrency: currency.USDT,
			STPAction:          "-",
			FinishAs:           "open",
			Slippage:           0.05,
		}
		assert.Equal(t, exp, result, "PlaceSpotOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "PlaceSpotOrder should return a result")
}

func TestGetSpotOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpotOrders must reject a nil request")

	_, err = e.GetSpotOrders(t.Context(), &GetSpotOrdersRequest{CurrencyPair: getPair(t, asset.Spot)})
	require.ErrorIs(t, err, errInvalidOrderStatus, "GetSpotOrders must reject an empty status")

	_, err = e.GetSpotOrders(t.Context(), &GetSpotOrdersRequest{Status: statusOpen})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSpotOrders must reject open orders without a currency pair")

	_, err = e.GetSpotOrders(t.Context(), &GetSpotOrdersRequest{Status: statusFinished, Side: order.Long})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "GetSpotOrders must reject an invalid side")

	startTime, endTime := getTime()
	_, err = e.GetSpotOrders(t.Context(), &GetSpotOrdersRequest{Status: statusFinished, From: endTime, To: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSpotOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpotOrders(t.Context(), &GetSpotOrdersRequest{CurrencyPair: getPair(t, asset.Spot), Status: statusOpen, Page: 100, Limit: 100})
	require.NoError(t, err, "GetSpotOrders must not error")
	if mockTests {
		exp := []*SpotOrderResponse{{
			OrderID:            "1852454420",
			Text:               "t-abc123",
			AmendText:          "-",
			CreateTimeSeconds:  types.Time(time.Unix(1744150000, 0)),
			UpdateTimeSeconds:  types.Time(time.Unix(1744150000, 0)),
			CreateTime:         types.Time(time.UnixMilli(1744150000073)),
			UpdateTime:         types.Time(time.UnixMilli(1744150000074)),
			CurrencyPair:       "BTC_USDT",
			TradeQuote:         currency.USDT,
			Status:             statusOpen,
			Type:               "limit",
			Account:            "spot",
			Side:               "buy",
			Amount:             0.001,
			Price:              65000,
			TimeInForce:        gtcTIF,
			RemainingAmount:    0.001,
			FeeCurrency:        currency.BTC,
			RebatedFeeCurrency: currency.USDT,
			FinishAs:           "open",
			StopProfit:         &SpotOrderTPSL{TriggerPrice: 67000, OrderPrice: 67000},
			StopLoss:           &SpotOrderTPSL{TriggerPrice: 63000, OrderPrice: 63000},
		}}
		assert.Equal(t, exp, result, "GetSpotOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSpotOrders should return a result")
}

func TestCancelAllOpenOrdersSpecifiedCurrencyPair(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOpenOrdersSpecifiedCurrencyPair(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelAllOpenOrdersSpecifiedCurrencyPair must reject a nil request")

	_, err = e.CancelAllOpenOrdersSpecifiedCurrencyPair(t.Context(), &CancelAllSpotOrdersRequest{CurrencyPair: getPair(t, asset.Spot), Side: order.Long, Account: "spot"})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "CancelAllOpenOrdersSpecifiedCurrencyPair must reject an invalid side")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	cancelledBuy := []*SpotOrderResponse{{
		OrderID:            "1852454420",
		Text:               "t-abc123",
		AmendText:          "-",
		Succeeded:          true,
		CreateTimeSeconds:  types.Time(time.Unix(1744150000, 0)),
		UpdateTimeSeconds:  types.Time(time.Unix(1744150300, 0)),
		CreateTime:         types.Time(time.UnixMilli(1744150000073)),
		UpdateTime:         types.Time(time.UnixMilli(1744150300074)),
		CurrencyPair:       "BTC_USDT",
		Status:             "cancelled",
		Type:               "limit",
		Account:            "spot",
		Side:               "buy",
		Amount:             0.001,
		Price:              65000,
		TimeInForce:        gtcTIF,
		RemainingAmount:    0.001,
		FeeCurrency:        currency.BTC,
		RebatedFeeCurrency: currency.USDT,
		FinishAs:           "cancelled",
	}}
	unifiedMarket := currency.NewBTCUSD().Format(currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter})
	for _, tc := range []struct {
		name string
		arg  *CancelAllSpotOrdersRequest
		exp  []*SpotOrderResponse
	}{
		{name: "buy", arg: &CancelAllSpotOrdersRequest{CurrencyPair: getPair(t, asset.Spot), Side: order.Buy, Account: "spot"}, exp: cancelledBuy},
		{name: "sell", arg: &CancelAllSpotOrdersRequest{CurrencyPair: getPair(t, asset.Spot), Side: order.Sell, Account: "spot"}, exp: []*SpotOrderResponse{}},
		// UnknownSide and AnySide omit the optional side filter so Gate cancels both bids and asks
		{name: "unknown side", arg: &CancelAllSpotOrdersRequest{CurrencyPair: getPair(t, asset.Spot), Account: "spot"}, exp: []*SpotOrderResponse{}},
		{name: "any side", arg: &CancelAllSpotOrdersRequest{CurrencyPair: getPair(t, asset.Spot), Side: order.AnySide, Account: "spot"}, exp: []*SpotOrderResponse{}},
		{
			name: "unified market quote",
			arg:  &CancelAllSpotOrdersRequest{CurrencyPair: unifiedMarket, Account: "unified", TradeQuote: currency.USDC, ActionMode: "ACK"},
			exp:  []*SpotOrderResponse{{OrderID: "1852454421", Text: "t-abc124", AmendText: "-", Succeeded: true}},
		},
	} {
		result, err := e.CancelAllOpenOrdersSpecifiedCurrencyPair(t.Context(), tc.arg)
		require.NoErrorf(t, err, "CancelAllOpenOrdersSpecifiedCurrencyPair must not error for %s", tc.name)
		if mockTests {
			assert.Equalf(t, tc.exp, result, "CancelAllOpenOrdersSpecifiedCurrencyPair should decode every field for %s", tc.name)
		}
	}
}

func TestCancelBatchOrdersWithIDList(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBatchOrdersWithIDList(t.Context(), []CancelOrderByIDRequest{})
	require.ErrorIs(t, err, errNoValidParameterPassed, "CancelBatchOrdersWithIDList must reject an empty batch")

	_, err = e.CancelBatchOrdersWithIDList(t.Context(), make([]CancelOrderByIDRequest, 21))
	require.ErrorIs(t, err, errInvalidOrderSize, "CancelBatchOrdersWithIDList must reject more than 20 orders")

	_, err = e.CancelBatchOrdersWithIDList(t.Context(), []CancelOrderByIDRequest{{ID: "1234567"}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelBatchOrdersWithIDList must reject an empty currency pair")

	_, err = e.CancelBatchOrdersWithIDList(t.Context(), []CancelOrderByIDRequest{{CurrencyPair: getPair(t, asset.Spot)}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelBatchOrdersWithIDList must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelBatchOrdersWithIDList(t.Context(), []CancelOrderByIDRequest{
		{
			CurrencyPair: getPair(t, asset.Spot),
			ID:           "1234567",
		},
		{
			CurrencyPair: getPair(t, asset.Spot),
			ID:           "123453",
		},
	})
	require.NoError(t, err, "CancelBatchOrdersWithIDList must not error")
	if mockTests {
		exp := []*CancelOrderByIDResponse{
			{CurrencyPair: getPair(t, asset.Spot), OrderID: "1234567", Text: "t-1234567", Succeeded: true},
			{CurrencyPair: getPair(t, asset.Spot), OrderID: "123453", Label: "ORDER_NOT_FOUND", Message: "Order not found"},
		}
		assert.Equal(t, exp, result, "CancelBatchOrdersWithIDList should decode every field")
		return
	}
	assert.NotEmpty(t, result, "CancelBatchOrdersWithIDList should return order results")
}

func TestGetSpotOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotOrder(t.Context(), "", getPair(t, asset.Spot), asset.Spot)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetSpotOrder must reject an empty order ID")

	_, err = e.GetSpotOrder(t.Context(), "1234", currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSpotOrder must reject an empty currency pair")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpotOrder(t.Context(), "1234", getPair(t, asset.Spot), asset.Spot)
	require.NoError(t, err, "GetSpotOrder must not error")
	if mockTests {
		exp := &SpotOrderResponse{
			OrderID:            "1234",
			Text:               "t-abc123",
			AmendText:          "-",
			CreateTimeSeconds:  types.Time(time.Unix(1744150000, 0)),
			UpdateTimeSeconds:  types.Time(time.Unix(1744150000, 0)),
			CreateTime:         types.Time(time.UnixMilli(1744150000073)),
			UpdateTime:         types.Time(time.UnixMilli(1744150000074)),
			CurrencyPair:       "BTC_USDT",
			TradeQuote:         currency.USDT,
			Status:             "closed",
			Type:               "limit",
			Account:            "spot",
			Side:               "buy",
			Amount:             0.001,
			Price:              65000,
			TimeInForce:        gtcTIF,
			FilledAmount:       0.001,
			AverageFillPrice:   63469.3,
			FeeDeducted:        0.00000022,
			FeeCurrency:        currency.BTC,
			FillPrice:          63.4693,
			FilledTotal:        63.4693,
			RebatedFeeCurrency: currency.USDT,
			FinishAs:           "filled",
			StopProfit:         &SpotOrderTPSL{TriggerPrice: 67000, OrderPrice: 67000},
			StopLoss:           &SpotOrderTPSL{TriggerPrice: 63000, OrderPrice: 63000},
		}
		assert.Equal(t, exp, result, "GetSpotOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSpotOrder should return a result")
}

func TestAmendSpotOrder(t *testing.T) {
	t.Parallel()
	_, err := e.AmendSpotOrder(t.Context(), "", &AmendSpotOrderRequest{CurrencyPair: getPair(t, asset.Spot), Price: 1000})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "AmendSpotOrder must reject an empty order ID")

	_, err = e.AmendSpotOrder(t.Context(), "123", nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AmendSpotOrder must reject a nil request")

	_, err = e.AmendSpotOrder(t.Context(), "123", &AmendSpotOrderRequest{Price: 1000})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "AmendSpotOrder must reject an empty currency pair")

	_, err = e.AmendSpotOrder(t.Context(), "123", &AmendSpotOrderRequest{CurrencyPair: getPair(t, asset.Spot)})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "AmendSpotOrder must reject a request without amount or price")
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "AmendSpotOrder must reject a request without amount or price")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	// An empty stop loss cancels the existing one while a nil take profit or stop loss would leave it unchanged
	result, err := e.AmendSpotOrder(t.Context(), "123", &AmendSpotOrderRequest{
		CurrencyPair: getPair(t, asset.Spot),
		Price:        1000,
		StopProfit:   &SpotOrderTPSL{TriggerPrice: 67000, OrderPrice: 67000},
		StopLoss:     &SpotOrderTPSL{},
	})
	require.NoError(t, err, "AmendSpotOrder must not error")
	if mockTests {
		exp := &SpotOrderResponse{
			OrderID:            "123",
			Text:               "t-abc123",
			AmendText:          "-",
			CreateTimeSeconds:  types.Time(time.Unix(1744150000, 0)),
			UpdateTimeSeconds:  types.Time(time.Unix(1744150200, 0)),
			CreateTime:         types.Time(time.UnixMilli(1744150000073)),
			UpdateTime:         types.Time(time.UnixMilli(1744150200074)),
			CurrencyPair:       "BTC_USDT",
			Status:             statusOpen,
			Type:               "limit",
			Account:            "spot",
			Side:               "buy",
			Amount:             0.001,
			Price:              1000,
			TimeInForce:        gtcTIF,
			RemainingAmount:    0.001,
			FeeCurrency:        currency.BTC,
			RebatedFeeCurrency: currency.USDT,
			FinishAs:           "open",
			StopProfit:         &SpotOrderTPSL{TriggerPrice: 67000, OrderPrice: 67000},
		}
		assert.Equal(t, exp, result, "AmendSpotOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "AmendSpotOrder should return a result")
}

func TestAmendSpotOrderRequestValidate(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, (*AmendSpotOrderRequest)(nil).validate(), common.ErrNilPointer, "validate should reject a nil request")
	assert.ErrorIs(t, (&AmendSpotOrderRequest{Price: 1000}).validate(), currency.ErrCurrencyPairEmpty, "validate should reject an empty currency pair")

	err := (&AmendSpotOrderRequest{CurrencyPair: currency.NewBTCUSDT()}).validate()
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid, "validate should reject a request without an amount or price")
	assert.ErrorIs(t, err, limits.ErrPriceBelowMin, "validate should reject a request without an amount or price")

	assert.NoError(t, (&AmendSpotOrderRequest{CurrencyPair: currency.NewBTCUSDT(), Amount: 1}).validate(), "validate should accept an amount alone")
	assert.NoError(t, (&AmendSpotOrderRequest{CurrencyPair: currency.NewBTCUSDT(), Price: 1000}).validate(), "validate should accept a price alone")
}

func TestCancelSingleSpotOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelSingleSpotOrder(t.Context(), "", getPair(t, asset.Spot), false)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelSingleSpotOrder must reject an empty order ID")

	_, err = e.CancelSingleSpotOrder(t.Context(), "1234", currency.EMPTYPAIR, false)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelSingleSpotOrder must reject an empty currency pair")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelSingleSpotOrder(t.Context(), "1234", getPair(t, asset.Spot), false)
	require.NoError(t, err, "CancelSingleSpotOrder must not error")
	if mockTests {
		exp := &SpotOrderResponse{
			OrderID:            "1234",
			Text:               "t-abc123",
			AmendText:          "-",
			CreateTimeSeconds:  types.Time(time.Unix(1744150000, 0)),
			UpdateTimeSeconds:  types.Time(time.Unix(1744150300, 0)),
			CreateTime:         types.Time(time.UnixMilli(1744150000073)),
			UpdateTime:         types.Time(time.UnixMilli(1744150300074)),
			CurrencyPair:       "BTC_USDT",
			Status:             "cancelled",
			Type:               "limit",
			Account:            "spot",
			Side:               "buy",
			Amount:             0.001,
			Price:              65000,
			TimeInForce:        gtcTIF,
			RemainingAmount:    0.001,
			FeeCurrency:        currency.BTC,
			RebatedFeeCurrency: currency.USDT,
			FinishAs:           "cancelled",
		}
		assert.Equal(t, exp, result, "CancelSingleSpotOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelSingleSpotOrder should return a result")
}

func TestGetMySpotTradingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetMySpotTradingHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMySpotTradingHistory must reject a nil request")

	_, err = e.GetMySpotTradingHistory(t.Context(), &GetMySpotTradingHistoryRequest{OrderID: "987345678234"})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetMySpotTradingHistory must reject an order ID without a currency pair")

	startTime, endTime := getTime()
	_, err = e.GetMySpotTradingHistory(t.Context(), &GetMySpotTradingHistoryRequest{From: endTime, To: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMySpotTradingHistory must reject a start time after the end time")

	_, err = e.GetMySpotTradingHistory(t.Context(), &GetMySpotTradingHistoryRequest{From: endTime.Add(-time.Hour * 24 * 31), To: endTime})
	require.ErrorIs(t, err, errTimeRangeTooLong, "GetMySpotTradingHistory must reject a range longer than 30 days")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMySpotTradingHistory(t.Context(), &GetMySpotTradingHistoryRequest{
		CurrencyPair: getPair(t, asset.Spot),
		OrderID:      "987345678234",
		Page:         1,
		Limit:        100,
		From:         startTime,
		To:           endTime,
	})
	require.NoError(t, err, "GetMySpotTradingHistory must not error")
	if mockTests {
		exp := []*SpotPersonalTradeHistoryResponse{{
			TradeID:           "1232893232",
			CreateTimeSeconds: types.Time(time.Unix(1744150000, 0)),
			CreateTime:        types.Time(time.UnixMicro(1744150000123456)),
			CurrencyPair:      getPair(t, asset.Spot),
			OrderID:           "987345678234",
			Side:              "buy",
			Role:              "maker",
			Amount:            0.15,
			Price:             65000,
			Fee:               0.0005,
			FeeCurrency:       currency.BTC,
			AmendText:         "-",
			SequenceID:        "588018",
			Text:              "t-test",
			Deal:              9750,
			TradeQuote:        currency.USDT,
		}}
		assert.Equal(t, exp, result, "GetMySpotTradingHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMySpotTradingHistory should return a result")
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	result, err := e.GetServerTime(t.Context(), asset.Spot)
	require.NoError(t, err, "GetServerTime must not error")
	if mockTests {
		assert.Equal(t, time.UnixMilli(1774386128160), result, "GetServerTime should decode the server time")
		return
	}
	assert.WithinDuration(t, time.Now(), result, time.Minute, "GetServerTime should return the current time")
}

func TestCountdownCancelSpotOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CountdownCancelSpotOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CountdownCancelSpotOrders must reject a nil request")

	_, err = e.CountdownCancelSpotOrders(t.Context(), &CountdownCancelOrderRequest{Timeout: 4, CurrencyPair: getPair(t, asset.Spot)})
	require.ErrorIs(t, err, errInvalidCountdown, "CountdownCancelSpotOrders must reject a countdown shorter than 5 seconds")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		arg  *CountdownCancelOrderRequest
		exp  *TriggerTimeResponse
	}{
		{
			name: "start countdown",
			arg:  &CountdownCancelOrderRequest{Timeout: 10, CurrencyPair: getPair(t, asset.Spot)},
			exp:  &TriggerTimeResponse{TriggerTime: types.Time(time.UnixMilli(1744190264000))},
		},
		{
			// A timeout of 0 stops the countdown, which Gate documents as valid
			name: "stop countdown",
			arg:  &CountdownCancelOrderRequest{},
			exp:  &TriggerTimeResponse{},
		},
	} {
		result, err := e.CountdownCancelSpotOrders(t.Context(), tc.arg)
		require.NoErrorf(t, err, "CountdownCancelSpotOrders must not error for %s", tc.name)
		if mockTests {
			assert.Equalf(t, tc.exp, result, "CountdownCancelSpotOrders should decode the trigger time for %s", tc.name)
		}
	}
}

func TestAmendBatchSpotOrders(t *testing.T) {
	t.Parallel()
	_, err := e.AmendBatchSpotOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "AmendBatchSpotOrders must reject an empty batch")

	valid := &BatchAmendSpotOrderRequest{OrderID: "121212", CurrencyPair: getPair(t, asset.Spot), Amount: 1, AmendText: "test"}
	_, err = e.AmendBatchSpotOrders(t.Context(), slices.Repeat([]*BatchAmendSpotOrderRequest{valid}, 6))
	require.ErrorIs(t, err, errTooManyBatchOrders, "AmendBatchSpotOrders must reject more than 5 orders")

	_, err = e.AmendBatchSpotOrders(t.Context(), []*BatchAmendSpotOrderRequest{nil})
	require.ErrorIs(t, err, common.ErrNilPointer, "AmendBatchSpotOrders must reject a nil order")

	_, err = e.AmendBatchSpotOrders(t.Context(), []*BatchAmendSpotOrderRequest{{AmendSpotOrderRequest: valid.AmendSpotOrderRequest}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "AmendBatchSpotOrders must reject an empty order ID")

	_, err = e.AmendBatchSpotOrders(t.Context(), []*BatchAmendSpotOrderRequest{{OrderID: "121212", Amount: 1}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "AmendBatchSpotOrders must reject an empty currency pair")

	_, err = e.AmendBatchSpotOrders(t.Context(), []*BatchAmendSpotOrderRequest{{OrderID: "121212", CurrencyPair: getPair(t, asset.Spot)}})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "AmendBatchSpotOrders must reject an order without amount or price")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AmendBatchSpotOrders(t.Context(), []*BatchAmendSpotOrderRequest{valid})
	require.NoError(t, err, "AmendBatchSpotOrders must not error")
	if mockTests {
		exp := []*SpotOrderResponse{{
			OrderID:            "121212",
			BatchOrderID:       "121212",
			Text:               "t-123456",
			AmendText:          "test",
			Succeeded:          true,
			CreateTimeSeconds:  types.Time(time.Unix(1744150000, 0)),
			UpdateTimeSeconds:  types.Time(time.Unix(1744150100, 0)),
			CreateTime:         types.Time(time.UnixMilli(1744150000123)),
			UpdateTime:         types.Time(time.UnixMilli(1744150100123)),
			CurrencyPair:       "BTC_USDT",
			Status:             statusOpen,
			Type:               "limit",
			Account:            "spot",
			Side:               "buy",
			Amount:             1,
			Price:              5.00032,
			TimeInForce:        gtcTIF,
			RemainingAmount:    1,
			FeeCurrency:        currency.BTC,
			RebatedFeeCurrency: currency.USDT,
			STPID:              10240,
			STPAction:          "cn",
			FinishAs:           "open",
		}}
		assert.Equal(t, exp, result, "AmendBatchSpotOrders should decode every field")
		return
	}
	assert.NotEmpty(t, result, "AmendBatchSpotOrders should return order results")
}

func TestGetSpotInsuranceHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotInsuranceHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpotInsuranceHistory must reject a nil request")

	startTime, endTime := getTime()
	_, err = e.GetSpotInsuranceHistory(t.Context(), &GetSpotInsuranceHistoryRequest{Currency: currency.USDT, From: startTime, To: endTime})
	require.ErrorIs(t, err, errBusinessTypeRequired, "GetSpotInsuranceHistory must reject an empty business type")

	_, err = e.GetSpotInsuranceHistory(t.Context(), &GetSpotInsuranceHistoryRequest{Business: "unified", From: startTime, To: endTime})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetSpotInsuranceHistory must reject an empty currency")

	_, err = e.GetSpotInsuranceHistory(t.Context(), &GetSpotInsuranceHistoryRequest{Business: "unified", Currency: currency.USDT, To: endTime})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetSpotInsuranceHistory must reject an unset start time")

	_, err = e.GetSpotInsuranceHistory(t.Context(), &GetSpotInsuranceHistoryRequest{Business: "unified", Currency: currency.USDT, From: endTime, To: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSpotInsuranceHistory must reject a start time after the end time")

	result, err := e.GetSpotInsuranceHistory(t.Context(), &GetSpotInsuranceHistoryRequest{
		Business: "unified",
		Currency: currency.USDT,
		From:     startTime,
		To:       endTime,
		Page:     1,
		Limit:    2,
	})
	require.NoError(t, err, "GetSpotInsuranceHistory must not error")
	if mockTests {
		exp := []*SpotInsuranceHistoryResponse{{Currency: currency.USDT, Balance: 3678340.316884272366, Time: types.Time(time.UnixMilli(1744156800000))}}
		assert.Equal(t, exp, result, "GetSpotInsuranceHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSpotInsuranceHistory should return a result")
}

func TestCreatePriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CreatePriceTriggeredOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreatePriceTriggeredOrder must reject a nil request")

	_, err = e.CreatePriceTriggeredOrder(t.Context(), &PriceTriggeredOrderRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CreatePriceTriggeredOrder must reject an empty market")

	arg := &PriceTriggeredOrderRequest{Market: getPair(t, asset.Spot), Trigger: TriggerPriceInfo{Price: -1}}
	_, err = e.CreatePriceTriggeredOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "CreatePriceTriggeredOrder must reject a non-positive trigger price")

	arg.Trigger.Price = 1
	_, err = e.CreatePriceTriggeredOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidTriggerRule, "CreatePriceTriggeredOrder must reject an invalid trigger rule")

	arg.Trigger.Rule = ">="
	_, err = e.CreatePriceTriggeredOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "CreatePriceTriggeredOrder must reject an invalid side")

	arg.Put.Side = "SELL"
	arg.Put.Price = -1
	_, err = e.CreatePriceTriggeredOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "CreatePriceTriggeredOrder must reject a negative put price")

	arg.Put.Price = 1
	_, err = e.CreatePriceTriggeredOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "CreatePriceTriggeredOrder must reject an empty amount")

	arg.Put.Amount = 1
	arg.Put.TimeInForce = pocTIF
	_, err = e.CreatePriceTriggeredOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrInvalidTimeInForce, "CreatePriceTriggeredOrder must reject time in force other than gtc or ioc")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreatePriceTriggeredOrder(t.Context(), &PriceTriggeredOrderRequest{
		Trigger: TriggerPriceInfo{
			Price:      123,
			Rule:       ">=",
			Expiration: 3600,
		},
		Put: PutOrder{
			Type:        "limit",
			Side:        "sell",
			Price:       2312312,
			Amount:      30,
			TimeInForce: "GTC",
		},
		Market: getPair(t, asset.Spot),
	})
	require.NoError(t, err, "CreatePriceTriggeredOrder must not error")
	if mockTests {
		assert.Equal(t, &TriggerOrderIDResponse{ID: 1432329, IDString: "1432329"}, result, "CreatePriceTriggeredOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CreatePriceTriggeredOrder should return a result")
}

func TestGetPriceTriggeredOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.GetPriceTriggeredOrderList(t.Context(), "", currency.EMPTYPAIR, "", 0, 0)
	require.ErrorIs(t, err, errInvalidOrderStatus, "GetPriceTriggeredOrderList must reject an empty status")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPriceTriggeredOrderList(t.Context(), statusOpen, currency.EMPTYPAIR, "", 0, 0)
	require.NoError(t, err, "GetPriceTriggeredOrderList must not error")
	if mockTests {
		exp := []*SpotPriceTriggeredOrderResponse{{
			Trigger:      TriggerPriceInfo{Price: 100, Rule: ">=", Expiration: 3600},
			Put:          PutOrder{Type: "limit", Side: "buy", Price: 2.15, Amount: 2, Account: "normal", TimeInForce: gtcTIF, Text: "api"},
			AutoOrderID:  1283293,
			UserID:       1234,
			CreationTime: types.Time(time.Unix(1744150000, 0)),
			Status:       statusOpen,
			Market:       getPair(t, asset.Spot),
		}}
		assert.Equal(t, exp, result, "GetPriceTriggeredOrderList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPriceTriggeredOrderList should return a result")
}

func TestCancelMultipleSpotOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelMultipleSpotOpenOrders(t.Context(), getPair(t, asset.Spot), "normal")
	require.NoError(t, err, "CancelMultipleSpotOpenOrders must not error")
	if mockTests {
		exp := []*SpotPriceTriggeredOrderResponse{{
			Trigger:      TriggerPriceInfo{Price: 100, Rule: "<=", Expiration: 3600},
			Put:          PutOrder{Type: "market", Side: "sell", Price: 99, Amount: 2, Account: "normal", TimeInForce: iocTIF, Text: "api"},
			AutoOrderID:  1283294,
			UserID:       1234,
			CreationTime: types.Time(time.Unix(1744150000, 0)),
			FireTime:     types.Time(time.Unix(1744150500, 0)),
			Status:       "cancelled",
			Reason:       "cancelled by user",
			Market:       getPair(t, asset.Spot),
		}}
		assert.Equal(t, exp, result, "CancelMultipleSpotOpenOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelMultipleSpotOpenOrders should return a result")
}

func TestGetSinglePriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetSinglePriceTriggeredOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetSinglePriceTriggeredOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSinglePriceTriggeredOrder(t.Context(), "1234")
	require.NoError(t, err, "GetSinglePriceTriggeredOrder must not error")
	if mockTests {
		exp := &SpotPriceTriggeredOrderResponse{
			Trigger:      TriggerPriceInfo{Price: 100, Rule: ">=", Expiration: 3600},
			Put:          PutOrder{Type: "limit", Side: "buy", Price: 2.15, Amount: 2, Account: "normal", TimeInForce: gtcTIF, Text: "api"},
			AutoOrderID:  1234,
			UserID:       1234,
			CreationTime: types.Time(time.Unix(1744150000, 0)),
			FireTime:     types.Time(time.Unix(1744150001, 0)),
			FiredOrderID: 18523,
			Status:       "finish",
			Market:       getPair(t, asset.Spot),
		}
		assert.Equal(t, exp, result, "GetSinglePriceTriggeredOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSinglePriceTriggeredOrder should return a result")
}

func TestCancelPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelPriceTriggeredOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelPriceTriggeredOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelPriceTriggeredOrder(t.Context(), "1234")
	require.NoError(t, err, "CancelPriceTriggeredOrder must not error")
	if mockTests {
		exp := &SpotPriceTriggeredOrderResponse{
			Trigger:      TriggerPriceInfo{Price: 100, Rule: ">=", Expiration: 3600},
			Put:          PutOrder{Type: "limit", Side: "buy", Price: 2.15, Amount: 2, Account: "normal", TimeInForce: gtcTIF, Text: "api"},
			AutoOrderID:  1234,
			UserID:       1234,
			CreationTime: types.Time(time.Unix(1744150000, 0)),
			FireTime:     types.Time(time.Unix(1744150500, 0)),
			Status:       "cancelled",
			Market:       getPair(t, asset.Spot),
		}
		assert.Equal(t, exp, result, "CancelPriceTriggeredOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelPriceTriggeredOrder should return a result")
}

func TestCreateSpotPOVOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CreateSpotPOVOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateSpotPOVOrder must reject a nil request")

	arg := &SpotPOVOrderRequest{}
	_, err = e.CreateSpotPOVOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CreateSpotPOVOrder must reject an empty currency pair")

	arg.CurrencyPair = getPair(t, asset.Spot)
	_, err = e.CreateSpotPOVOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "CreateSpotPOVOrder must reject an invalid side")

	arg.Side = order.Buy.Lower()
	_, err = e.CreateSpotPOVOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "CreateSpotPOVOrder must reject an empty amount")

	arg.Amount = 1
	_, err = e.CreateSpotPOVOrder(t.Context(), arg)
	require.ErrorIs(t, err, errParticipationRateRequired, "CreateSpotPOVOrder must reject an empty participation rate")

	arg.ParticipationRate = 5
	_, err = e.CreateSpotPOVOrder(t.Context(), arg)
	require.ErrorIs(t, err, errTimeToLiveRequired, "CreateSpotPOVOrder must reject an empty time to live")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateSpotPOVOrder(t.Context(), &SpotPOVOrderRequest{
		CurrencyPair:      getPair(t, asset.Spot),
		Side:              order.Buy.Lower(),
		Amount:            1,
		ParticipationRate: 5,
		TimeToLive:        "1h",
		LimitPrice:        63000,
		TriggerPrice:      63000,
		Text:              "t-pov1",
	})
	require.NoError(t, err, "CreateSpotPOVOrder must not error")
	if mockTests {
		exp := &SpotPOVOrderResponse{
			ID:                "1216",
			CurrencyPair:      getPair(t, asset.Spot),
			Side:              "buy",
			Amount:            1,
			ParticipationRate: 5,
			TimeToLive:        "1h",
			LimitPrice:        63000,
			TriggerPrice:      63000,
			Status:            "CREATED",
			ExpireTime:        types.Time(time.UnixMilli(1744153600074)),
			CreateTime:        types.Time(time.UnixMilli(1744150000074)),
			UpdateTime:        types.Time(time.UnixMilli(1744150000074)),
			Text:              "t-pov1",
		}
		assert.Equal(t, exp, result, "CreateSpotPOVOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CreateSpotPOVOrder should return a result")
}

func TestGetSpotPOVOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotPOVOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpotPOVOrders must reject a nil request")

	_, err = e.GetSpotPOVOrders(t.Context(), &GetSpotPOVOrdersRequest{})
	require.ErrorIs(t, err, errInvalidOrderStatus, "GetSpotPOVOrders must reject an empty status")

	_, err = e.GetSpotPOVOrders(t.Context(), &GetSpotPOVOrdersRequest{Status: statusOpen, Side: order.Short})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "GetSpotPOVOrders must reject an invalid side")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpotPOVOrders(t.Context(), &GetSpotPOVOrdersRequest{
		CurrencyPair: getPair(t, asset.Spot),
		Status:       statusOpen,
		Side:         order.Buy,
		Page:         1,
		Limit:        10,
	})
	require.NoError(t, err, "GetSpotPOVOrders must not error")
	if mockTests {
		exp := []*SpotPOVOrderResponse{{
			ID:                "1216",
			CurrencyPair:      getPair(t, asset.Spot),
			Side:              "buy",
			Amount:            0.01,
			ParticipationRate: 10,
			TimeToLive:        "1h",
			LimitPrice:        63000,
			TriggerPrice:      63000,
			Status:            "RUNNING",
			StartTime:         types.Time(time.UnixMilli(1744150001074)),
			ExpireTime:        types.Time(time.UnixMilli(1744153600074)),
			CreateTime:        types.Time(time.UnixMilli(1744150000074)),
			UpdateTime:        types.Time(time.UnixMilli(1744150001074)),
		}}
		assert.Equal(t, exp, result, "GetSpotPOVOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSpotPOVOrders should return a result")
}

func TestCancelSpotPOVOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelSpotPOVOrders(t.Context(), getPair(t, asset.Spot))
	require.NoError(t, err, "CancelSpotPOVOrders must not error")
	if mockTests {
		exp := []*SpotPOVOrderResponse{{
			ID:                "1216",
			CurrencyPair:      getPair(t, asset.Spot),
			Side:              "buy",
			Amount:            0.01,
			ParticipationRate: 10,
			TimeToLive:        "1h",
			LimitPrice:        63000,
			TriggerPrice:      63000,
			Status:            "CANCELING",
			StartTime:         types.Time(time.UnixMilli(1744150001074)),
			ExpireTime:        types.Time(time.UnixMilli(1744153600074)),
			CreateTime:        types.Time(time.UnixMilli(1744150000074)),
			UpdateTime:        types.Time(time.UnixMilli(1744150002074)),
		}}
		assert.Equal(t, exp, result, "CancelSpotPOVOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelSpotPOVOrders should return a result")
}

func TestGetSpotPOVOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotPOVOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetSpotPOVOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpotPOVOrder(t.Context(), "1216")
	require.NoError(t, err, "GetSpotPOVOrder must not error")
	if mockTests {
		exp := &SpotPOVOrderResponse{
			ID:                "1216",
			CurrencyPair:      getPair(t, asset.Spot),
			Side:              "sell",
			Amount:            0.01,
			ParticipationRate: 20,
			TimeToLive:        "1d",
			Status:            "COMPLETED",
			StartTime:         types.Time(time.UnixMilli(1744150001074)),
			EndTime:           types.Time(time.UnixMilli(1744150901074)),
			ExpireTime:        types.Time(time.UnixMilli(1744236400074)),
			CreateTime:        types.Time(time.UnixMilli(1744150000074)),
			UpdateTime:        types.Time(time.UnixMilli(1744150901074)),
			Text:              "t-pov2",
		}
		assert.Equal(t, exp, result, "GetSpotPOVOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSpotPOVOrder should return a result")
}

func TestCancelSpotPOVOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelSpotPOVOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelSpotPOVOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelSpotPOVOrder(t.Context(), "1216")
	require.NoError(t, err, "CancelSpotPOVOrder must not error")
	if mockTests {
		exp := &SpotPOVOrderResponse{
			ID:                "1216",
			CurrencyPair:      getPair(t, asset.Spot),
			Side:              "buy",
			Amount:            0.01,
			ParticipationRate: 40,
			TimeToLive:        "6h",
			LimitPrice:        63000,
			Status:            "CANCELING",
			StartTime:         types.Time(time.UnixMilli(1744150001074)),
			ExpireTime:        types.Time(time.UnixMilli(1744171600074)),
			CreateTime:        types.Time(time.UnixMilli(1744150000074)),
			UpdateTime:        types.Time(time.UnixMilli(1744150002074)),
		}
		assert.Equal(t, exp, result, "CancelSpotPOVOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelSpotPOVOrder should return a result")
}

func TestGenerateSignature(t *testing.T) {
	t.Parallel()
	// Expected values follow Gate's documented scheme: HMAC-SHA512 over method, path, query, the SHA512 of the body and
	// the Unix timestamp, each separated by a newline
	ts := time.Unix(1700000000, 0)
	for _, tc := range []struct {
		method, path, query string
		body                any
		exp                 string
	}{
		{
			method: http.MethodGet,
			path:   "/api/v4/spot/orders",
			query:  "currency_pair=BTC_USDT&status=open",
			exp:    "777f9b7385b58e88af6ce1aaae82f217f3e3dd0b9406e98522498001de299a9636bc797e3291aa1c49df79d8decf51eca463bd6082937c4c8742175c938eee56",
		},
		{
			method: http.MethodPost,
			path:   "/api/v4/spot/orders",
			body:   map[string]string{"currency_pair": "BTC_USDT"},
			exp:    "dac4c0d44277469bbb4d2dd7297111b89ce2039ebc9b4cebfaa8cd8a361738ca6904a75ebc28df1a1c041af50c104884dddeeae9bbbe89064513a3837363c886",
		},
	} {
		sig, err := e.GenerateSignature("mock-api-secret", tc.method, tc.path, tc.query, tc.body, ts)
		require.NoErrorf(t, err, "GenerateSignature must not error for %s %s", tc.method, tc.path)
		assert.Equalf(t, tc.exp, sig, "GenerateSignature should sign %s %s as documented", tc.method, tc.path)
	}
}

func TestCurrencySupportedByCrossMargin(t *testing.T) {
	t.Parallel()
	result, err := e.CurrencySupportedByCrossMargin(t.Context())
	require.NoError(t, err, "CurrencySupportedByCrossMargin must not error")
	if mockTests {
		require.Len(t, result, 7, "CurrencySupportedByCrossMargin must return every recorded currency")
		exp := &CrossMarginCurrenciesResponse{
			Name:                 currency.BTC,
			Rate:                 0.0002,
			CurrencyPrecision:    0.000001,
			Discount:             1,
			MinBorrowAmount:      0.01,
			UserMaxBorrowAmount:  1000000,
			TotalMaxBorrowAmount: 10000000,
			Price:                1,
			Loanable:             true,
			Status:               1,
		}
		assert.Equal(t, exp, result[0], "CurrencySupportedByCrossMargin should decode every field")
		return
	}
	assert.NotEmpty(t, result, "CurrencySupportedByCrossMargin should return currencies")
}

func TestGetCrossMarginSupportedCurrencyDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossMarginSupportedCurrencyDetail(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetCrossMarginSupportedCurrencyDetail must reject an empty currency")

	result, err := e.GetCrossMarginSupportedCurrencyDetail(t.Context(), currency.BTC)
	require.NoError(t, err, "GetCrossMarginSupportedCurrencyDetail must not error")
	if mockTests {
		exp := &CrossMarginCurrenciesResponse{
			Name:                 currency.BTC,
			CurrencyPrecision:    0.000001,
			Discount:             1,
			MinBorrowAmount:      0.00001,
			UserMaxBorrowAmount:  120000000,
			TotalMaxBorrowAmount: 300000000,
			Price:                82139.13,
			Loanable:             true,
			Status:               1,
		}
		assert.Equal(t, exp, result, "GetCrossMarginSupportedCurrencyDetail should decode every field")
		return
	}
	assert.Equal(t, currency.BTC, result.Name, "GetCrossMarginSupportedCurrencyDetail should return the requested currency")
}

func TestGetCrossMarginAccounts(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossMarginAccounts(t.Context())
	require.NoError(t, err, "GetCrossMarginAccounts must not error")
	if mockTests {
		exp := &CrossMarginAccountResponse{
			UserID: 10001,
			Balances: map[string]CrossMarginCurrencyBalance{
				"BTC": {Available: 0.5, Freeze: 0.1, Borrowed: 0.02, Interest: 0.0001},
			},
			Total:                       48050,
			Borrowed:                    1600,
			Interest:                    8,
			Risk:                        25.6,
			TotalInitialMargin:          160,
			TotalMarginBalance:          48050,
			TotalMaintenanceMargin:      80,
			TotalInitialMarginRate:      300.31,
			TotalMaintenanceMarginRate:  600.62,
			TotalAvailableMargin:        47890,
			TotalPortfolioMarginAccount: 48050,
		}
		assert.Equal(t, exp, result, "GetCrossMarginAccounts should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCrossMarginAccounts should return an account")
}

func TestGetCrossMarginAccountChangeHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetCrossMarginAccountChangeHistory(t.Context(), currency.BTC, endTime, startTime, 0, 6, "in")
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetCrossMarginAccountChangeHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossMarginAccountChangeHistory(t.Context(), currency.BTC, startTime, endTime, 1, 6, "in")
	require.NoError(t, err, "GetCrossMarginAccountChangeHistory must not error")
	if mockTests {
		exp := []*CrossMarginAccountHistoryItemResponse{{
			ID:       "123456",
			Time:     types.Time(time.UnixMilli(1744150000000)),
			Currency: currency.BTC,
			Change:   1.03,
			Balance:  4.59316525194,
			Type:     "in",
		}}
		assert.Equal(t, exp, result, "GetCrossMarginAccountChangeHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCrossMarginAccountChangeHistory should return a result")
}

func TestCreateCrossMarginBorrowLoanNullReply(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v4/margin/cross/loans", r.URL.Path, "request path should be the cross margin loans")
		_, err := w.Write([]byte(`null`))
		assert.NoError(t, err, "writing the response should not error")
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")

	_, err := ex.CreateCrossMarginBorrowLoan(t.Context(), &CrossMarginBorrowLoanRequest{Currency: currency.BTC, Amount: 3})
	assert.ErrorIs(t, err, common.ErrNoResponse, "CreateCrossMarginBorrowLoan should reject a null reply")
}

func TestCreateCrossMarginBorrowLoan(t *testing.T) {
	t.Parallel()
	_, err := e.CreateCrossMarginBorrowLoan(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateCrossMarginBorrowLoan must reject a nil request")

	_, err = e.CreateCrossMarginBorrowLoan(t.Context(), &CrossMarginBorrowLoanRequest{Currency: currency.EMPTYCODE, Amount: 3})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateCrossMarginBorrowLoan must reject an empty currency")

	_, err = e.CreateCrossMarginBorrowLoan(t.Context(), &CrossMarginBorrowLoanRequest{Currency: currency.BTC, Amount: 0})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "CreateCrossMarginBorrowLoan must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateCrossMarginBorrowLoan(t.Context(), &CrossMarginBorrowLoanRequest{Currency: currency.BTC, Amount: 3})
	require.NoError(t, err, "CreateCrossMarginBorrowLoan must not error")
	if mockTests {
		exp := &CrossMarginLoanResponse{
			ID:             "17",
			CreateTime:     types.Time(time.UnixMilli(1744150000000)),
			UpdateTime:     types.Time(time.UnixMilli(1744150000000)),
			Currency:       currency.BTC,
			Amount:         3,
			Status:         2,
			UnpaidInterest: 0.0001,
		}
		assert.Equal(t, exp, result, "CreateCrossMarginBorrowLoan should decode every field")
		return
	}
	assert.NotNil(t, result, "CreateCrossMarginBorrowLoan should return the loan")
}

func TestGetCrossMarginBorrowHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossMarginBorrowHistory(t.Context(), 6, currency.BTC, 0, 0, false)
	require.ErrorIs(t, err, errInvalidOrderStatus, "GetCrossMarginBorrowHistory must reject an unknown status")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossMarginBorrowHistory(t.Context(), 1, currency.BTC, 0, 0, false)
	require.NoError(t, err, "GetCrossMarginBorrowHistory must not error")
	if mockTests {
		exp := []*CrossMarginLoanResponse{{
			ID:         "19",
			CreateTime: types.Time(time.UnixMilli(1744150000000)),
			UpdateTime: types.Time(time.UnixMilli(1744150000000)),
			Currency:   currency.BTC,
			Amount:     100,
			Text:       "web",
			Status:     1,
		}}
		assert.Equal(t, exp, result, "GetCrossMarginBorrowHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCrossMarginBorrowHistory should return a result")
}

func TestGetSingleBorrowLoanDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleBorrowLoanDetail(t.Context(), "")
	require.ErrorIs(t, err, errInvalidLoanID, "GetSingleBorrowLoanDetail must reject an empty loan ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSingleBorrowLoanDetail(t.Context(), "1234")
	require.NoError(t, err, "GetSingleBorrowLoanDetail must not error")
	if mockTests {
		exp := &CrossMarginLoanResponse{
			ID:             "1234",
			CreateTime:     types.Time(time.UnixMilli(1744103854944)),
			UpdateTime:     types.Time(time.UnixMilli(1744150000000)),
			Currency:       currency.BTC,
			Amount:         0.5,
			Text:           "web",
			Status:         2,
			Repaid:         0.1,
			RepaidInterest: 0.00001,
			UnpaidInterest: 0.00004,
		}
		assert.Equal(t, exp, result, "GetSingleBorrowLoanDetail should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSingleBorrowLoanDetail should return the loan")
}

func TestExecuteRepayment(t *testing.T) {
	t.Parallel()
	_, err := e.ExecuteRepayment(t.Context(), CurrencyAndAmountResponse{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "ExecuteRepayment must reject an empty currency")

	_, err = e.ExecuteRepayment(t.Context(), CurrencyAndAmountResponse{Currency: currency.BTC})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "ExecuteRepayment must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ExecuteRepayment(t.Context(), CurrencyAndAmountResponse{Currency: currency.USDT, Amount: 1234.55})
	require.NoError(t, err, "ExecuteRepayment must not error")
	if mockTests {
		exp := []*CrossMarginLoanResponse{{
			ID:             "18",
			CreateTime:     types.Time(time.UnixMilli(1744103854944)),
			UpdateTime:     types.Time(time.UnixMilli(1744150000000)),
			Currency:       currency.USDT,
			Amount:         1234.55,
			Text:           "web",
			Status:         3,
			Repaid:         1234.55,
			RepaidInterest: 0.04,
		}}
		assert.Equal(t, exp, result, "ExecuteRepayment should decode every field")
		return
	}
	assert.NotNil(t, result, "ExecuteRepayment should return the repaid loans")
}

func TestGetCrossMarginRepayments(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossMarginRepayments(t.Context(), currency.BTC, "123", 1000, 10, true)
	require.NoError(t, err, "GetCrossMarginRepayments must not error")
	if mockTests {
		exp := []*CrossMarginLoanResponse{{
			ID:             "123",
			CreateTime:     types.Time(time.UnixMilli(1744103854944)),
			UpdateTime:     types.Time(time.UnixMilli(1744150000000)),
			Currency:       currency.BTC,
			Amount:         0.5,
			Text:           "web",
			Status:         3,
			Repaid:         0.5,
			RepaidInterest: 0.00002,
		}}
		assert.Equal(t, exp, result, "GetCrossMarginRepayments should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCrossMarginRepayments should return a result")
}

func TestGetMaxTransferableAmountForSpecificCrossMarginCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetMaxTransferableAmountForSpecificCrossMarginCurrency(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMaxTransferableAmountForSpecificCrossMarginCurrency must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMaxTransferableAmountForSpecificCrossMarginCurrency(t.Context(), currency.BTC)
	require.NoError(t, err, "GetMaxTransferableAmountForSpecificCrossMarginCurrency must not error")
	if mockTests {
		exp := &CurrencyAndAmountResponse{Currency: currency.BTC, Amount: 1.5}
		assert.Equal(t, exp, result, "GetMaxTransferableAmountForSpecificCrossMarginCurrency should decode every field")
		return
	}
	assert.Equal(t, currency.BTC, result.Currency, "GetMaxTransferableAmountForSpecificCrossMarginCurrency should return the requested currency")
}

func TestGetMaxBorrowableAmountForSpecificCrossMarginCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetMaxBorrowableAmountForSpecificCrossMarginCurrency(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMaxBorrowableAmountForSpecificCrossMarginCurrency must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMaxBorrowableAmountForSpecificCrossMarginCurrency(t.Context(), currency.BTC)
	require.NoError(t, err, "GetMaxBorrowableAmountForSpecificCrossMarginCurrency must not error")
	if mockTests {
		exp := &CurrencyAndAmountResponse{Currency: currency.BTC, Amount: 0.8}
		assert.Equal(t, exp, result, "GetMaxBorrowableAmountForSpecificCrossMarginCurrency should decode every field")
		return
	}
	assert.Equal(t, currency.BTC, result.Currency, "GetMaxBorrowableAmountForSpecificCrossMarginCurrency should return the requested currency")
}

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

func TestWalletAccountType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		asset       asset.Item
		expected    string
		expectedErr error
	}{
		{asset: asset.Spot, expected: "spot"},
		{asset: asset.Margin, expected: "margin"},
		{asset: asset.Futures, expected: "futures"},
		{asset: asset.USDTMarginedFutures, expected: "futures"},
		{asset: asset.CoinMarginedFutures, expected: "futures"},
		{asset: asset.DeliveryFutures, expected: "delivery"},
		{asset: asset.Options, expected: "options"},
		{asset: asset.Empty, expectedErr: asset.ErrInvalidAsset},
		{asset: asset.CrossMargin, expectedErr: asset.ErrNotSupported},
	} {
		t.Run(tc.asset.String(), func(t *testing.T) {
			t.Parallel()
			got, err := walletAccountType(tc.asset)
			require.ErrorIs(t, err, tc.expectedErr, "walletAccountType must return the expected error")
			assert.Equal(t, tc.expected, got, "walletAccountType should return the account Gate expects")
		})
	}
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
			Currency:                currency.GT,
			CurrencyName:            "GateToken",
			CurrencyNameChinese:     "GateToken",
			DepositFee:              0,
			WithdrawPercent:         "0%",
			FixedWithdrawalFee:      0.01,
			WithdrawDayLimit:        20000,
			WithdrawDayLimitRemain:  20000,
			WithdrawAmountMini:      0.11,
			WithdrawEachTimeLimit:   20000,
			WithdrawFixOnChains:     map[string]types.Number{"BTC": 20, "ETH": 15, "TRX": 0, "EOS": 2.5},
			WithdrawPercentOnChains: map[string]string{"ETH": "0%", "GTEVM": "0.1%"},
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
				Interest:                   0.5,
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
		exp := []*WalletSavedAddressResponse{{Currency: currency.BTC, Chain: "BTC", Address: "1HkxtBAMrA3tP5ENnYY2CZortjZvFDH5Cs", Name: "gate", Verified: "1"}}
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
			UserID:              10001,
			TakerFee:            0.002,
			MakerFee:            0.002,
			RPIMakerFee:         -0.00175,
			GTDiscount:          true,
			GTTakerFee:          0.0015,
			GTMakerFee:          0.0015,
			LoanFee:             0.18,
			PointType:           "1",
			FuturesTakerFee:     -0.00025,
			FuturesMakerFee:     0.00075,
			FuturesRPIMakerFee:  -0.00175,
			DeliveryTakerFee:    0.00016,
			DeliveryMakerFee:    -0.00015,
			FeeDeductionType:    3,
			RPIMarketMakerLevel: 2,
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

// futuresLocalServerExchange returns an authenticated exchange whose REST requests reach handler. Mock tests use it for
// endpoints whose JSON bodies the VCR mock server cannot match, which compares nested arrays and objects in a
// recorded body as Go formatted text but in a request body as JSON
func futuresLocalServerExchange(t *testing.T, handler http.HandlerFunc) *Exchange {
	t.Helper()
	server := httptest.NewTestServer(t, handler)
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	return ex
}

// futuresFixturePosition returns the position the futures fixtures record for a contract, position mode and size
func futuresFixturePosition(contract currency.Pair, mode string, size types.Number) *PositionResponse {
	return &PositionResponse{
		User:                       10000,
		Contract:                   contract,
		Size:                       size,
		HedgeStatus:                "partial_hedged",
		HedgedSize:                 12,
		UnhedgedSize:               9428,
		RiskLimit:                  100,
		LeverageMax:                100,
		MaintenanceRate:            0.005,
		Value:                      3568.62,
		Margin:                     4.431548146258,
		EntryPrice:                 3779.55,
		LiquidationPrice:           99999999,
		MarkPrice:                  3780.32,
		InitialMargin:              1.5,
		MaintenanceMargin:          0.75,
		UnrealisedPNL:              -0.000507486844,
		RealisedPNL:                0.045543982432,
		RealisedPNLPosition:        0.045543982432,
		RealisedPNLFundingFee:      0.1,
		RealisedPNLTransactionFees: 0.2,
		HistoryPNL:                 0.3,
		LastClosePNL:               0.4,
		RealisedPNLPoint:           0.5,
		RealisedPNLHistoryPoint:    0.6,
		ADLRanking:                 5,
		PendingOrders:              16,
		CloseOrder:                 CloseOrder{ID: 232323, Price: 3779, IsLiquidation: true},
		Mode:                       mode,
		CrossLeverageLimit:         10,
		UpdateTime:                 types.Time(time.Unix(1684994406, 0)),
		OpenTime:                   types.Time(time.Unix(1684990000, 0)),
		UpdateID:                   1,
		RiskLimitTable:             "BIG_HOT_COIN_50X_V2",
		AverageMaintenanceRate:     0.005,
		PositionID:                 7,
		PositionMarginMode:         "isolated",
		PositionLeverage:           30,
	}
}

// futuresFixtureAccount returns the account the futures fixtures record for a settlement currency
func futuresFixtureAccount(settle currency.Code) *FuturesAccountResponse {
	return &FuturesAccountResponse{
		User:                          1666,
		Currency:                      settle,
		Total:                         9707.803567115145,
		UnrealisedPNL:                 3371.248828,
		PositionMargin:                38.712189181,
		OrderMargin:                   0.5,
		Available:                     9669.091377934145,
		Point:                         1,
		Bonus:                         2,
		EnabledCredit:                 true,
		InDualMode:                    true,
		PositionInitialMargin:         10.5,
		MaintenanceMargin:             2.25,
		MarginMode:                    1,
		EnabledEvolvedClassic:         true,
		CrossOrderMargin:              3,
		CrossInitialMargin:            61855.56788525,
		CrossMaintenanceMargin:        682.04678105,
		CrossUnrealisedPNL:            1501.178222634128,
		CrossAvailable:                27549.406108813951,
		CrossMarginBalance:            10371.77306201952,
		CrossMaintenanceMarginRate:    797.2134,
		CrossInitialMarginRate:        116.6097,
		IsolatedPositionMargin:        4,
		EnableNewDualMode:             true,
		EnableTieredMaintenanceMargin: true,
		EnableDualPlus:                true,
		PositionMode:                  "dual_plus",
		History: FuturesAccountHistory{
			DepositAndWithdrawal:      10000,
			ProfitAndLoss:             68.3685,
			Fee:                       -1.645812875,
			ReferrerRebate:            5,
			Funding:                   -358.919120009855,
			PointDepositAndWithdrawal: 6,
			PointFee:                  7,
			PointReferrerRebate:       8,
			BonusDepositAndWithdrawal: 9,
			BonusOffset:               10,
			CrossSettle:               12.5,
		},
	}
}

// futuresFixtureOrder returns the order the futures fixtures record for a contract
func futuresFixtureOrder(contract currency.Pair) *FuturesOrderResponse {
	return &FuturesOrderResponse{
		ID:                        15675394,
		User:                      "100000",
		Contract:                  contract,
		CreateTime:                types.Time(time.Unix(1546569968, 0)),
		UpdateTime:                types.Time(time.UnixMilli(1546569970123)),
		Size:                      6024,
		RemainingAmount:           6024,
		Price:                     3765,
		MakerFeeRate:              -0.00025,
		TakerFeeRate:              0.00075,
		TimeInForce:               "gtc",
		Text:                      "t-my-custom-id",
		Status:                    "finished",
		FinishTime:                types.Time(time.Unix(1514764900, 0)),
		FinishAs:                  "cancelled",
		SelfTradePreventionAction: "-",
		AmendText:                 "-",
		OrderValue:                64112.2099,
		TradeValue:                64112.2099,
		MarketOrderSlipRatio:      0.03,
		PositionMarginMode:        "isolated",
		ActionMode:                "FULL",
		TakeProfitTriggerPrice:    3800,
		StopLossTriggerPrice:      3700,
	}
}

// futuresFixtureTrailOrder returns the trail order the futures fixtures record
func futuresFixtureTrailOrder() *TrailOrderResponse {
	return &TrailOrderResponse{
		ID:                 "63585",
		UserID:             "2124438083",
		User:               "2124438083",
		Contract:           currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter),
		Settle:             "usdt",
		Amount:             10,
		IsGreaterThanEqual: true,
		ActivationPrice:    50000,
		PriceType:          1,
		PriceOffset:        "5%",
		Text:               "apiv4",
		ReduceOnly:         true,
		PositionRelated:    true,
		CreatedAt:          types.Time(time.Unix(1769569837, 0)),
		ActivatedAt:        types.Time(time.Unix(1769569838, 0)),
		FinishedAt:         types.Time(time.Unix(1769578529, 0)),
		CreateTime:         types.Time(time.Unix(1769569837, 0)),
		ActiveTime:         types.Time(time.Unix(1769569838, 0)),
		FinishTime:         types.Time(time.Unix(1769578529, 0)),
		Reason:             "success",
		SuborderText:       "apiv4-auto-trail-o-1d29",
		IsDualMode:         true,
		TriggerPrice:       91047.4,
		SuborderID:         "94294117233225616",
		SideLabel:          "buy",
		OriginalStatus:     4,
		Status:             "finished",
		PositionSideOutput: "buy",
		UpdatedAt:          types.Time(time.Unix(1769578529, 0)),
		ExtremumPrice:      86711.9,
		StatusCode:         "success",
		CreatedAtPrecise:   types.Time(time.UnixMicro(1769569837778000)),
		FinishedAtPrecise:  types.Time(time.UnixMicro(1769578529853294)),
		ActivatedAtPrecise: types.Time(time.UnixMicro(1769569837976010)),
		StatusLabel:        "finished",
		PositionMarginMode: "cross",
		PositionMode:       "single",
		Leverage:           10,
	}
}

// futuresFixtureChaseOrder returns the chase order the futures fixtures record
func futuresFixtureChaseOrder() *ChaseOrderResponse {
	return &ChaseOrderResponse{
		ID:                 "1001",
		User:               "2124438083",
		Contract:           currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter),
		Settle:             "usdt",
		Amount:             1,
		PriceLimit:         68000,
		ReduceOnly:         true,
		Text:               "t-chase",
		CreateTime:         types.Time(time.Unix(1774991579, 0)),
		FinishTime:         types.Time(time.Unix(1774991679, 0)),
		OriginalStatus:     3,
		Status:             "finished",
		Reason:             "filled",
		FillAmount:         1,
		AverageFillPrice:   67900.1,
		SuborderID:         "94294117233225616",
		IsDualMode:         true,
		SideLabel:          "buy",
		PositionSideOutput: "buy",
		ChasePrice:         67900.1,
		IntervalSeconds:    5,
		UpdatedAt:          types.Time(time.Unix(1774991679, 0)),
		SuborderPrice:      67900.1,
		SuborderFinishAs:   "filled",
		PriceType:          2,
		PriceGapType:       "1",
		PriceGapValue:      0.5,
		StatusCode:         "success",
		CreateTimePrecise:  types.Time(time.UnixMicro(1774991579123456)),
		FinishTimePrecise:  types.Time(time.UnixMicro(1774991679654321)),
		PositionMarginMode: "cross",
		PositionMode:       "dual",
		Leverage:           10,
	}
}

// futuresFixturePriceTriggeredOrder returns the price triggered order the futures fixtures record
func futuresFixturePriceTriggeredOrder() *PriceTriggeredOrderResponse {
	return &PriceTriggeredOrderResponse{
		Initial: PriceTriggeredOrderInitial{
			Contract:     currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter),
			Size:         2,
			Amount:       2,
			Price:        1234,
			TimeInForce:  "gtc",
			Text:         "api",
			ReduceOnly:   true,
			IsReduceOnly: true,
			IsClose:      true,
		},
		Trigger: PriceTriggeredOrderTrigger{
			PriceType:  1,
			Price:      3000,
			Rule:       1,
			Expiration: 86400,
		},
		ID:                 1432329,
		IDString:           "1432329",
		User:               1666,
		CreateTime:         types.Time(time.Unix(1514764800, 0)),
		FinishTime:         types.Time(time.Unix(1514764900, 0)),
		TradeID:            13566,
		Status:             "finished",
		FinishAs:           "succeeded",
		Reason:             "triggered",
		OrderType:          "close-short-position",
		MatchingOrderID:    15675394,
		PositionMarginMode: "isolated",
	}
}

func TestGetAllFutureContracts(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllFutureContracts(t.Context(), currency.EMPTYCODE, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAllFutureContracts must reject an empty settlement currency")

	for _, c := range []currency.Code{currency.BTC, currency.USDT} {
		result, err := e.GetAllFutureContracts(t.Context(), c, 0, 0)
		require.NoErrorf(t, err, "GetAllFutureContracts must not error for %s", c)
		assert.NotEmptyf(t, result, "GetAllFutureContracts should return contracts for %s", c)
	}

	result, err := e.GetAllFutureContracts(t.Context(), currency.USDT, 1, 1)
	require.NoError(t, err, "GetAllFutureContracts must not error with a limit and offset")
	require.Len(t, result, 1, "GetAllFutureContracts must honour the limit")
	if mockTests {
		exp := &FuturesContractResponse{
			Name:                  currency.NewPairWithDelimiter("BTC", "USDT", "_"),
			Type:                  "direct",
			QuantoMultiplier:      0.0001,
			OrderPriceDeviate:     0.5,
			MaintenanceRate:       0.005,
			MarkType:              "index",
			LastPrice:             38026,
			MarkPrice:             37985.6,
			IndexPrice:            37954.92,
			FundingRateIndicative: 0.000219,
			MarkPriceRound:        0.01,
			RiskLimitBase:         1000000,
			InterestRate:          0.0003,
			OrderPriceRound:       0.1,
			OrderSizeMin:          1,
			RefRebateRate:         0.2,
			FundingInterval:       28800,
			RiskLimitStep:         1000000,
			LeverageMin:           1,
			LeverageMax:           100,
			RiskLimitMax:          8000000,
			MakerFeeRate:          -0.00025,
			TakerFeeRate:          0.00075,
			FundingRate:           0.002053,
			OrderSizeMax:          1000000,
			FundingNextApply:      types.Time(time.Unix(1610035200, 0)),
			ShortUsers:            977,
			ConfigChangeTime:      types.Time(time.Unix(1609899548, 0)),
			TradeSize:             28530850594,
			PositionSize:          5223816,
			LongUsers:             455,
			FundingImpactValue:    60000,
			OrdersLimit:           50,
			TradeID:               10851092,
			OrderbookID:           2129638396,
			EnableBonus:           true,
			EnableCredit:          true,
			CreateTime:            types.Time(time.Unix(1669688556, 0)),
			FundingCapRatio:       0.75,
			Status:                "trading",
			LaunchTime:            types.Time(time.Unix(1609899548, 0)),
			DelistingTime:         types.Time(time.Unix(1609899549, 0)),
			DelistedTime:          types.Time(time.Unix(1609899550, 0)),
			EnableDecimal:         true,
			MarketOrderSlipRatio:  0.05,
			MarketOrderSizeMax:    500000,
			FundingRateLimit:      0.003,
			ContractType:          "indices",
			EnableCircuitBreaker:  true,
			CrossLeverageDefault:  10,
			VoucherLeverage:       2,
			IsPreMarket:           true,
		}
		assert.Equal(t, exp, result[0], "GetAllFutureContracts should decode every field")
	}
}

func TestGetAllFutureContractsIncludingDelisted(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllFutureContractsIncludingDelisted(t.Context(), currency.EMPTYCODE, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAllFutureContractsIncludingDelisted must reject an empty settlement currency")

	result, err := e.GetAllFutureContractsIncludingDelisted(t.Context(), currency.USDT, 1, 0)
	require.NoError(t, err, "GetAllFutureContractsIncludingDelisted must not error")
	require.Len(t, result, 1, "GetAllFutureContractsIncludingDelisted must honour the limit")
	if mockTests {
		exp := &FuturesContractResponse{
			Name:                  currency.NewPairWithDelimiter("LUNA", "USDT", "_"),
			Type:                  "direct",
			QuantoMultiplier:      0.0001,
			OrderPriceDeviate:     0.5,
			MaintenanceRate:       0.005,
			MarkType:              "index",
			LastPrice:             0.0001,
			MarkPrice:             0.0001,
			IndexPrice:            0.0001,
			FundingRateIndicative: 0.000219,
			MarkPriceRound:        0.01,
			Delisting:             true,
			RiskLimitBase:         1000000,
			InterestRate:          0.0003,
			OrderPriceRound:       0.1,
			OrderSizeMin:          1,
			RefRebateRate:         0.2,
			FundingInterval:       28800,
			RiskLimitStep:         1000000,
			LeverageMin:           1,
			LeverageMax:           100,
			RiskLimitMax:          8000000,
			MakerFeeRate:          -0.00025,
			TakerFeeRate:          0.00075,
			FundingRate:           0.002053,
			OrderSizeMax:          1000000,
			FundingNextApply:      types.Time(time.Unix(1610035200, 0)),
			ShortUsers:            977,
			ConfigChangeTime:      types.Time(time.Unix(1609899548, 0)),
			TradeSize:             28530850594,
			LongUsers:             455,
			FundingImpactValue:    60000,
			OrdersLimit:           50,
			TradeID:               10851092,
			OrderbookID:           2129638396,
			EnableBonus:           true,
			EnableCredit:          true,
			CreateTime:            types.Time(time.Unix(1669688556, 0)),
			FundingCapRatio:       0.75,
			Status:                "delisted",
			LaunchTime:            types.Time(time.Unix(1609899548, 0)),
			DelistingTime:         types.Time(time.Unix(1609899549, 0)),
			DelistedTime:          types.Time(time.Unix(1609899550, 0)),
			MarketOrderSlipRatio:  0.05,
			MarketOrderSizeMax:    500000,
			FundingRateLimit:      0.003,
			CrossLeverageDefault:  10,
			VoucherLeverage:       2,
		}
		assert.Equal(t, exp, result[0], "GetAllFutureContractsIncludingDelisted should decode every field")
	}
}

func TestGetFuturesContract(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContract(t.Context(), currency.USDT, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesContract must reject an empty contract")

	_, err = e.GetFuturesContract(t.Context(), currency.EMPTYCODE, getPair(t, asset.USDTMarginedFutures).String())
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesContract must reject an empty settlement currency")

	result, err := e.GetFuturesContract(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures).String())
	require.NoError(t, err, "GetFuturesContract must not error for a USDT contract")
	assert.NotNil(t, result, "GetFuturesContract should return a USDT contract")

	result, err = e.GetFuturesContract(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures).String())
	require.NoError(t, err, "GetFuturesContract must not error for a BTC contract")
	if mockTests {
		exp := &FuturesContractResponse{
			Name:                  currency.NewPairWithDelimiter("BTC", "USD", "_"),
			Type:                  "inverse",
			OrderPriceDeviate:     0.5,
			MaintenanceRate:       0.005,
			MarkType:              "index",
			LastPrice:             67295.7,
			MarkPrice:             67501.33,
			IndexPrice:            67499.05,
			FundingRateIndicative: 0.0001,
			MarkPriceRound:        0.01,
			RiskLimitBase:         3,
			InterestRate:          0.0003,
			OrderPriceRound:       0.1,
			OrderSizeMin:          1,
			RefRebateRate:         0.2,
			FundingInterval:       28800,
			RiskLimitStep:         2497,
			LeverageMin:           1,
			LeverageMax:           100,
			RiskLimitMax:          2500,
			MakerFeeRate:          -0.0002,
			TakerFeeRate:          0.00075,
			FundingRate:           0.0001,
			OrderSizeMax:          600000,
			FundingNextApply:      types.Time(time.Unix(1775433600, 0)),
			ShortUsers:            224,
			ConfigChangeTime:      types.Time(time.Unix(1769580511, 0)),
			TradeSize:             61312185044,
			PositionSize:          8701256,
			LongUsers:             349,
			FundingImpactValue:    0.5,
			OrdersLimit:           50,
			TradeID:               48366685,
			OrderbookID:           5235273987,
			EnableCredit:          true,
			CreateTime:            types.Time(time.Unix(1545235200, 0)),
			FundingCapRatio:       0.75,
			Status:                "trading",
			LaunchTime:            types.Time(time.Unix(1545235200, 0)),
			MarketOrderSlipRatio:  0.02,
			MarketOrderSizeMax:    400000,
			FundingRateLimit:      0.00375,
			CrossLeverageDefault:  10,
		}
		assert.Equal(t, exp, result, "GetFuturesContract should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFuturesContract should return a BTC contract")
}

func TestGetFuturesADLRiskStates(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesADLRiskStates(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesADLRiskStates must reject an empty settlement currency")

	result, err := e.GetFuturesADLRiskStates(t.Context(), currency.USDT)
	require.NoError(t, err, "GetFuturesADLRiskStates must not error")
	if mockTests {
		exp := &FuturesADLRiskStatesResponse{
			Settle: "usdt",
			States: map[string]*FuturesADLRiskState{
				"BTC_USDT": {
					State:        "normal",
					CalculatedAt: types.Time(time.UnixMilli(1786695185345)),
				},
				"HYPE_USDT": {
					State:        "adl_risk",
					CalculatedAt: types.Time(time.UnixMilli(1786695185346)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetFuturesADLRiskStates should decode every field")
		return
	}
	require.NotNil(t, result, "GetFuturesADLRiskStates must return a result")
	assert.NotEmpty(t, result.States, "GetFuturesADLRiskStates should return market states")
}

func TestGetFuturesOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrderbook(t.Context(), currency.BTC, currency.EMPTYPAIR, "", 10, false)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesOrderbook must reject an empty contract")

	_, err = e.GetFuturesOrderbook(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures), "", 10, false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesOrderbook must reject an empty settlement currency")

	result, err := e.GetFuturesOrderbook(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), "", 10, false)
	require.NoError(t, err, "GetFuturesOrderbook must not error for a USDT contract")
	assert.NotNil(t, result, "GetFuturesOrderbook should return a USDT order book")

	result, err = e.GetFuturesOrderbook(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), "", 10, false)
	require.NoError(t, err, "GetFuturesOrderbook must not error for a BTC contract")
	if mockTests {
		exp := &OrderbookResponse{
			ID:      123456,
			Current: types.Time(time.UnixMilli(1623898993123)),
			Update:  types.Time(time.UnixMilli(1623898993121)),
			Bids: OrderbookLevels{
				{
					Amount: 150,
					Price:  1.17,
				},
				{
					Amount: 203,
					Price:  1.16,
				},
			},
			Asks: OrderbookLevels{
				{
					Amount: 100,
					Price:  1.52,
				},
				{
					Amount: 40,
					Price:  1.53,
				},
			},
		}
		assert.Equal(t, exp, result, "GetFuturesOrderbook should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFuturesOrderbook should return a BTC order book")
}

func TestGetFuturesTradingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesTradingHistory(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures), 0, 0, "", time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesTradingHistory must reject an empty settlement currency")

	_, err = e.GetFuturesTradingHistory(t.Context(), currency.BTC, currency.EMPTYPAIR, 0, 0, "", time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesTradingHistory must reject an empty contract")

	_, err = e.GetFuturesTradingHistory(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0, 0, "", time.Time{}, time.Time{})
	require.NoError(t, err, "GetFuturesTradingHistory must not error for a BTC contract")

	result, err := e.GetFuturesTradingHistory(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 0, 0, "", time.Time{}, time.Time{})
	require.NoError(t, err, "GetFuturesTradingHistory must not error for a USDT contract")
	require.NotEmpty(t, result, "GetFuturesTradingHistory must return trades")
	if mockTests {
		require.Len(t, result, 5, "GetFuturesTradingHistory must return every recorded trade")
		exp := &TradingHistoryItemResponse{
			ID:                     724267229,
			CreateTime:             types.Time(time.UnixMilli(1775423970950)),
			CreateTimeMilliseconds: types.Time(time.UnixMilli(1775423970950)),
			Contract:               "BTC_USDT",
			Size:                   -4,
			Price:                  67477.9,
		}
		assert.Equal(t, exp, result[0], "GetFuturesTradingHistory should decode every field")
	}
}

func TestGetFuturesCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesCandlesticks(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures), time.Time{}, time.Time{}, 0, kline.OneWeek, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesCandlesticks must reject an empty settlement currency")

	_, err = e.GetFuturesCandlesticks(t.Context(), currency.BTC, currency.EMPTYPAIR, time.Time{}, time.Time{}, 0, kline.OneWeek, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesCandlesticks must reject an empty contract")

	startTime, endTime := getTime()
	_, err = e.GetFuturesCandlesticks(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), startTime, endTime, 10, kline.OneWeek, "")
	require.ErrorIs(t, err, errLimitWithTimeRange, "GetFuturesCandlesticks must reject a limit combined with a time range")

	_, err = e.GetFuturesCandlesticks(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), time.Time{}, time.Time{}, 0, kline.OneWeek, "utc3")
	require.ErrorIs(t, err, errInvalidTimezone, "GetFuturesCandlesticks must reject an unsupported timezone")

	_, err = e.GetFuturesCandlesticks(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), time.Time{}, time.Time{}, 0, kline.OneWeek, "")
	require.NoError(t, err, "GetFuturesCandlesticks must not error for a BTC contract")

	result, err := e.GetFuturesCandlesticks(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), time.Time{}, time.Time{}, 0, kline.OneWeek, "")
	require.NoError(t, err, "GetFuturesCandlesticks must not error for a USDT contract")
	require.NotEmpty(t, result, "GetFuturesCandlesticks must return candles")
	if mockTests {
		require.Len(t, result, 5, "GetFuturesCandlesticks must return every recorded candle")
		exp := &FuturesCandlestickResponse{
			Timestamp:    types.Time(time.Unix(1715212800, 0)),
			Volume:       1035191038,
			ClosePrice:   66173.3,
			HighestPrice: 66398.8,
			LowestPrice:  60166.2,
			OpenPrice:    61187,
			Sum:          6443459137.427661,
		}
		assert.Equal(t, exp, result[0], "GetFuturesCandlesticks should decode every field")
	}
}

func TestPremiumIndexKline(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	_, err := e.PremiumIndexKline(t.Context(), currency.EMPTYCODE, contract, time.Time{}, time.Time{}, 0, kline.OneWeek)
	require.ErrorIs(t, err, errEmptyOrInvalidSettlementCurrency, "PremiumIndexKline must reject an empty settlement currency")

	_, err = e.PremiumIndexKline(t.Context(), currency.USDT, currency.EMPTYPAIR, time.Time{}, time.Time{}, 0, kline.OneWeek)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "PremiumIndexKline must reject an empty contract")

	_, err = e.PremiumIndexKline(t.Context(), currency.USDT, contract, time.Time{}, time.Time{}, 0, kline.FiveDay)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "PremiumIndexKline must reject an unsupported interval")

	startTime, endTime := getTime()
	_, err = e.PremiumIndexKline(t.Context(), currency.USDT, contract, startTime, endTime, 1, kline.OneWeek)
	require.ErrorIs(t, err, errLimitWithTimeRange, "PremiumIndexKline must reject a limit combined with a time range")

	result, err := e.PremiumIndexKline(t.Context(), currency.USDT, contract, time.Time{}, time.Time{}, 1, kline.OneWeek)
	require.NoError(t, err, "PremiumIndexKline must not error with a limit")
	if mockTests {
		exp := []*FuturesPremiumIndexKlineResponse{
			{
				UnixTimestamp: types.Time(time.Unix(1780876800, 0)),
				ClosePrice:    -0.0005988142412787689,
				HighestPrice:  -0.0005638765430773542,
				LowestPrice:   -0.0013547770658499465,
				OpenPrice:     0.0030031738532652376,
			},
			{
				UnixTimestamp: types.Time(time.Unix(1781481600, 0)),
				ClosePrice:    -0.0004602259855446214,
				HighestPrice:  -0.0007128946261382114,
				LowestPrice:   -0.0015641449473647465,
				OpenPrice:     0.00025202506014303526,
			},
		}
		assert.Equal(t, exp, result, "PremiumIndexKline should decode every field")
		return
	}
	assert.NotEmpty(t, result, "PremiumIndexKline should return data points")
}

func TestGetFutureTickers(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesTickers(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesTickers must reject an empty settlement currency")

	result, err := e.GetFuturesTickers(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures))
	require.NoError(t, err, "GetFuturesTickers must not error for a BTC contract")
	assert.Len(t, result, 1, "GetFuturesTickers should return the BTC contract's ticker")

	result, err = e.GetFuturesTickers(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures))
	require.NoError(t, err, "GetFuturesTickers must not error for a USDT contract")
	require.Len(t, result, 1, "GetFuturesTickers must return the USDT contract's ticker")
	if mockTests {
		exp := &FuturesTickerResponse{
			Contract:              currency.NewPairWithDelimiter("BTC", "USDT", "_"),
			ChangePercentage:      0.37,
			Last:                  67477.9,
			Low24Hour:             66580.1,
			High24Hour:            67815.7,
			TotalSize:             577702480,
			Volume24Hour:          327232988,
			Volume24HourBase:      32723,
			Volume24HourQuote:     2196541426,
			Volume24HourSettle:    2196541426,
			QuantoMultiplier:      0.0001,
			MarkPrice:             67478,
			FundingRate:           0.000083,
			FundingRateIndicative: 0.000083,
			IndexPrice:            67503.92,
			LowestAsk:             67478,
			LowestAskSize:         57437,
			HighestBid:            67477.9,
			HighestBidSize:        95272,
			ChangePrice:           248.1,
		}
		assert.Equal(t, exp, result[0], "GetFuturesTickers should decode every field")
	}
}

func TestGetFutureFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetFutureFundingRates(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures), 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFutureFundingRates must reject an empty settlement currency")

	_, err = e.GetFutureFundingRates(t.Context(), currency.BTC, currency.EMPTYPAIR, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFutureFundingRates must reject an empty contract")

	startTime, endTime := getTime()
	_, err = e.GetFutureFundingRates(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 10, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFutureFundingRates must reject a start after the end")

	_, err = e.GetFutureFundingRates(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0, time.Time{}, time.Time{})
	require.NoError(t, err, "GetFutureFundingRates must not error for a BTC contract")

	result, err := e.GetFutureFundingRates(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 10, startTime, endTime)
	require.NoError(t, err, "GetFutureFundingRates must not error for a USDT contract within a time range")
	if mockTests {
		exp := []*FuturesFundingRateResponse{
			{
				Timestamp: types.Time(time.Unix(1744185600, 0)),
				Rate:      0.000157,
			},
			{
				Timestamp: types.Time(time.Unix(1744156800, 0)),
				Rate:      -0.000145,
			},
		}
		assert.Equal(t, exp, result, "GetFutureFundingRates should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetFutureFundingRates should return rates")
}

func TestGetBatchFuturesFundingRates(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	_, err := e.GetBatchFuturesFundingRates(t.Context(), currency.EMPTYCODE, currency.Pairs{contract})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetBatchFuturesFundingRates must reject an empty settlement currency")

	_, err = e.GetBatchFuturesFundingRates(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "GetBatchFuturesFundingRates must reject an empty contract list")

	_, err = e.GetBatchFuturesFundingRates(t.Context(), currency.USDT, currency.Pairs{currency.EMPTYPAIR})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetBatchFuturesFundingRates must reject an empty contract")

	ex := e
	if mockTests {
		ex = futuresLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method, "batch funding rates should be requested with POST")
			assert.Equal(t, "/api/v4/futures/usdt/funding_rates", r.URL.Path, "batch funding rates should be requested from the funding_rates endpoint")
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err, "reading the request body should not error")
			assert.JSONEq(t, `{"contracts":["BTC_USDT"]}`, string(body), "batch funding rates should send the contract list in the body")
			_, err = w.Write([]byte(`[{"contract":"BTC_USDT","data":[{"t":1543968000,"r":"0.000157"},{"t":1544054400,"r":"-0.000145"}]}]`))
			assert.NoError(t, err, "writing the funding rates should not error")
		})
	}
	result, err := ex.GetBatchFuturesFundingRates(t.Context(), currency.USDT, currency.Pairs{contract})
	require.NoError(t, err, "GetBatchFuturesFundingRates must not error")
	if mockTests {
		exp := []*BatchFundingRatesResponse{
			{
				Contract: "BTC_USDT",
				Data: []*FuturesFundingRateResponse{
					{
						Timestamp: types.Time(time.Unix(1543968000, 0)),
						Rate:      0.000157,
					},
					{
						Timestamp: types.Time(time.Unix(1544054400, 0)),
						Rate:      -0.000145,
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetBatchFuturesFundingRates should decode every field")
		return
	}
	require.Len(t, result, 1, "GetBatchFuturesFundingRates must return the requested contract")
	assert.NotEmpty(t, result[0].Data, "GetBatchFuturesFundingRates should return funding rates")
}

func TestGetFuturesInsuranceBalanceHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesInsuranceBalanceHistory(t.Context(), currency.EMPTYCODE, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesInsuranceBalanceHistory must reject an empty settlement currency")

	result, err := e.GetFuturesInsuranceBalanceHistory(t.Context(), currency.USDT, 10)
	require.NoError(t, err, "GetFuturesInsuranceBalanceHistory must not error")
	require.NotEmpty(t, result, "GetFuturesInsuranceBalanceHistory must return balances")
	if mockTests {
		require.Len(t, result, 5, "GetFuturesInsuranceBalanceHistory must return every recorded balance")
		exp := &InsuranceBalanceResponse{
			Timestamp: types.Time(time.Unix(1775347200, 0)),
			Balance:   132672449.967292,
		}
		assert.Equal(t, exp, result[0], "GetFuturesInsuranceBalanceHistory should decode every field")
	}
}

func TestGetFutureStats(t *testing.T) {
	t.Parallel()
	_, err := e.GetFutureStats(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures), time.Time{}, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFutureStats must reject an empty settlement currency")

	_, err = e.GetFutureStats(t.Context(), currency.BTC, currency.EMPTYPAIR, time.Time{}, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFutureStats must reject an empty contract")

	_, err = e.GetFutureStats(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), time.Time{}, kline.FiveDay, 0)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "GetFutureStats must reject an unsupported interval")

	_, err = e.GetFutureStats(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), time.Time{}, 0, 0)
	require.NoError(t, err, "GetFutureStats must not error for a BTC contract")

	result, err := e.GetFutureStats(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), time.UnixMilli(1780846985008), kline.OneMin, 100)
	require.NoError(t, err, "GetFutureStats must not error for a USDT contract")
	if mockTests {
		require.Len(t, result, 5, "GetFutureStats must return every recorded stat")
		exp := &ContractStatResponse{
			Time:                types.Time(time.Unix(1780847040, 0)),
			LongShortTaker:      0.4883055642879598,
			LongShortAccount:    1.5497060395510422,
			OpenInterest:        643307912,
			MarkPrice:           62048.59,
			TopLongShortSize:    1.060266307572198,
			OpenInterestUSD:     3991634887.544408,
			TopLongShortAccount: 0.5662650602409639,
			TopLongSize:         36769850,
			TopShortSize:        34679825,
			TopLongAccount:      47,
			TopShortAccount:     83,
			LongTakerSize:       155330,
			ShortTakerSize:      318100,
			LongUsers:           17397,
			ShortUsers:          11226,
		}
		assert.Equal(t, exp, result[0], "GetFutureStats should decode every field")
	}
}

func TestGetIndexConstituent(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexConstituent(t.Context(), currency.EMPTYCODE, getPair(t, asset.USDTMarginedFutures).String())
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetIndexConstituent must reject an empty settlement currency")

	_, err = e.GetIndexConstituent(t.Context(), currency.USDT, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetIndexConstituent must reject an empty index")

	result, err := e.GetIndexConstituent(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures).String())
	require.NoError(t, err, "GetIndexConstituent must not error")
	if mockTests {
		exp := &IndexConstituentResponse{
			Index: "BTC_USDT",
			Constituents: []*IndexConstituentSources{
				{
					Exchange: "Binance",
					Symbols: []string{
						"BTC_USDT",
					},
				},
				{
					Exchange: "Bitget",
					Symbols: []string{
						"BTC_USDT",
					},
				},
				{
					Exchange: "Bybit",
					Symbols: []string{
						"BTC_USDT",
					},
				},
				{
					Exchange: "Gate",
					Symbols: []string{
						"BTC_USDT",
					},
				},
				{
					Exchange: "MEXC",
					Symbols: []string{
						"BTC_USDT",
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetIndexConstituent should decode every field")
		return
	}
	require.NotNil(t, result, "GetIndexConstituent must return a result")
	assert.NotEmpty(t, result.Constituents, "GetIndexConstituent should return constituents")
}

func TestGetLiquidationHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetLiquidationHistory(t.Context(), currency.EMPTYCODE, getPair(t, asset.USDTMarginedFutures), time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetLiquidationHistory must reject an empty settlement currency")

	startTime, endTime := getTimeWithInterval(kline.OneHour)
	if !mockTests {
		endTime = startTime.Add(time.Hour) // live ranges span two days, beyond this endpoint's one hour limit
	}
	_, err = e.GetLiquidationHistory(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), endTime, startTime, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetLiquidationHistory must reject a start after the end")

	_, err = e.GetLiquidationHistory(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), startTime, endTime.Add(time.Second), 0)
	require.ErrorIs(t, err, errLiquidationTimeRangeTooLarge, "GetLiquidationHistory must reject a time range beyond an hour")

	result, err := e.GetLiquidationHistory(t.Context(), currency.USDT, currency.EMPTYPAIR, startTime, endTime, 10)
	require.NoError(t, err, "GetLiquidationHistory must not error without a contract")
	if mockTests {
		exp := []*LiquidationHistoryResponse{
			{
				Time:       types.Time(time.Unix(1744104000, 0)),
				Contract:   "BTC_USDT",
				Size:       600,
				OrderSize:  -600,
				OrderPrice: 3405,
				FillPrice:  3424,
			},
		}
		assert.Equal(t, exp, result, "GetLiquidationHistory should decode every field")
	}
}

func TestQueryFuturesAccount(t *testing.T) {
	t.Parallel()
	_, err := e.QueryFuturesAccount(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "QueryFuturesAccount must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.QueryFuturesAccount(t.Context(), currency.USDT)
	require.NoError(t, err, "QueryFuturesAccount must not error")
	if mockTests {
		exp := futuresFixtureAccount(currency.USDT)
		assert.Equal(t, exp, result, "QueryFuturesAccount should decode every field")
		return
	}
	assert.NotNil(t, result, "QueryFuturesAccount should return an account")
}

func TestGetFuturesAccountBooks(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesAccountBooks(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR, 0, 0, time.Time{}, time.Time{}, "dnw")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesAccountBooks must reject an empty settlement currency")

	startTime, endTime := getTime()
	_, err = e.GetFuturesAccountBooks(t.Context(), currency.USDT, currency.EMPTYPAIR, 0, 0, endTime, startTime, "dnw")
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesAccountBooks must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesAccountBooks(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 10, 1, startTime, endTime, "fee")
	require.NoError(t, err, "GetFuturesAccountBooks must not error")
	if mockTests {
		exp := []*AccountBookItemResponse{
			{
				Time:     types.Time(time.UnixMicro(1682294400123456)),
				Change:   0.000010152188,
				Balance:  4.59316525194,
				Text:     "BTC_USDT:6086261",
				Type:     "fee",
				Contract: "BTC_USDT",
				TradeID:  "1",
				ID:       "2",
			},
		}
		assert.Equal(t, exp, result, "GetFuturesAccountBooks should decode every field")
	}
}

func TestGetAllFuturesPositionsOfUsers(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllFuturesPositionsOfUsers(t.Context(), currency.EMPTYCODE, true, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAllFuturesPositionsOfUsers must reject an empty settlement currency")

	_, err = e.GetAllFuturesPositionsOfUsers(t.Context(), currency.USDT, true, 101, 0)
	require.ErrorIs(t, err, errInvalidLimit, "GetAllFuturesPositionsOfUsers must reject a limit above 100")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAllFuturesPositionsOfUsers(t.Context(), currency.USDT, true, 10, 0)
	require.NoError(t, err, "GetAllFuturesPositionsOfUsers must not error")
	if mockTests {
		exp := []*PositionResponse{futuresFixturePosition(getPair(t, asset.USDTMarginedFutures), "single", -9440)}
		assert.Equal(t, exp, result, "GetAllFuturesPositionsOfUsers should decode every field")
	}
}

func TestGetFuturesHistoricalPositions(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesHistoricalPositions(t.Context(), currency.EMPTYCODE, getPair(t, asset.USDTMarginedFutures), 0, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesHistoricalPositions must reject an empty settlement currency")

	_, err = e.GetFuturesHistoricalPositions(t.Context(), currency.USDT, currency.EMPTYPAIR, 0, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesHistoricalPositions must reject an empty contract")

	startTime, endTime := getTime()
	_, err = e.GetFuturesHistoricalPositions(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 0, 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesHistoricalPositions must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesHistoricalPositions(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 10, 0, startTime, endTime)
	require.NoError(t, err, "GetFuturesHistoricalPositions must not error")
	if mockTests {
		exp := []*FuturesHistoricalPositionResponse{
			{
				Contract:                currency.NewPairWithDelimiter("BTC", "USDT", "_"),
				Size:                    -9440,
				RiskLimit:               100,
				LeverageMax:             100,
				MaintenanceRate:         0.005,
				Margin:                  4.43,
				LiquidationPrice:        99999999,
				RealisedPNL:             0.04,
				HistoryPNL:              1.5,
				LastClosePNL:            0.25,
				RealisedPNLPoint:        0.1,
				RealisedPNLHistoryPoint: 0.2,
				Mode:                    "single",
				CrossLeverageLimit:      10,
				EntryPrice:              3779.55,
				Time:                    types.Time(time.Unix(1744110000, 0)),
			},
		}
		assert.Equal(t, exp, result, "GetFuturesHistoricalPositions should decode every field")
	}
}

func TestGetSinglePosition(t *testing.T) {
	t.Parallel()
	_, err := e.GetSinglePosition(t.Context(), currency.EMPTYCODE, getPair(t, asset.USDTMarginedFutures))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetSinglePosition must reject an empty settlement currency")

	_, err = e.GetSinglePosition(t.Context(), currency.USDT, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSinglePosition must reject an empty contract")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSinglePosition(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures))
	require.NoError(t, err, "GetSinglePosition must not error")
	if mockTests {
		exp := futuresFixturePosition(getPair(t, asset.USDTMarginedFutures), "single", -9440)
		assert.Equal(t, exp, result, "GetSinglePosition should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSinglePosition should return a position")
}

func TestGetFuturesLeverage(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	_, err := e.GetFuturesLeverage(t.Context(), currency.EMPTYCODE, contract, "isolated", "dual_long")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesLeverage must reject an empty settlement currency")

	_, err = e.GetFuturesLeverage(t.Context(), currency.USDT, currency.EMPTYPAIR, "isolated", "dual_long")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesLeverage must reject an empty contract")

	_, err = e.GetFuturesLeverage(t.Context(), currency.USDT, contract, "portfolio", "dual_long")
	require.ErrorIs(t, err, errInvalidPositionMarginMode, "GetFuturesLeverage must reject an unknown position margin mode")

	_, err = e.GetFuturesLeverage(t.Context(), currency.USDT, contract, "isolated", "long")
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "GetFuturesLeverage must reject an unknown dual side")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesLeverage(t.Context(), currency.USDT, contract, "isolated", "dual_long")
	require.NoError(t, err, "GetFuturesLeverage must not error")
	if mockTests {
		exp := &FuturesLeverageResponse{
			Leverage: 10,
		}
		assert.Equal(t, exp, result, "GetFuturesLeverage should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFuturesLeverage should return a leverage")
}

func TestUpdateFuturesPositionMargin(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateFuturesPositionMargin(t.Context(), currency.EMPTYCODE, 0.01, getPair(t, asset.CoinMarginedFutures))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "UpdateFuturesPositionMargin must reject an empty settlement currency")

	_, err = e.UpdateFuturesPositionMargin(t.Context(), currency.BTC, 0.01, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateFuturesPositionMargin must reject an empty contract")

	_, err = e.UpdateFuturesPositionMargin(t.Context(), currency.BTC, 0, getPair(t, asset.CoinMarginedFutures))
	require.ErrorIs(t, err, errInvalidMarginChange, "UpdateFuturesPositionMargin must reject a zero change")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err = e.UpdateFuturesPositionMargin(t.Context(), currency.BTC, 0.01, getPair(t, asset.CoinMarginedFutures))
	require.NoError(t, err, "UpdateFuturesPositionMargin must not error adding margin")

	result, err := e.UpdateFuturesPositionMargin(t.Context(), currency.USDT, -0.01, getPair(t, asset.USDTMarginedFutures))
	require.NoError(t, err, "UpdateFuturesPositionMargin must not error removing margin")
	if mockTests {
		exp := futuresFixturePosition(getPair(t, asset.USDTMarginedFutures), "single", -9440)
		assert.Equal(t, exp, result, "UpdateFuturesPositionMargin should decode every field")
		return
	}
	assert.NotNil(t, result, "UpdateFuturesPositionMargin should return the position")
}

func TestUpdateFuturesPositionLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateFuturesPositionLeverage(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures), 1, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "UpdateFuturesPositionLeverage must reject an empty settlement currency")

	_, err = e.UpdateFuturesPositionLeverage(t.Context(), currency.BTC, currency.EMPTYPAIR, 1, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateFuturesPositionLeverage must reject an empty contract")

	_, err = e.UpdateFuturesPositionLeverage(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), -1, 0, 0)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported, "UpdateFuturesPositionLeverage must reject a negative leverage")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.UpdateFuturesPositionLeverage(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 1, 0, 0)
	require.NoError(t, err, "UpdateFuturesPositionLeverage must not error")
	if mockTests {
		exp := futuresFixturePosition(getPair(t, asset.CoinMarginedFutures), "single", -9440)
		assert.Equal(t, exp, result, "UpdateFuturesPositionLeverage should decode every field")
		return
	}
	assert.NotNil(t, result, "UpdateFuturesPositionLeverage should return the position")
}

func TestPositionReplyUnmarshalJSON(t *testing.T) {
	t.Parallel()
	exp := &PositionResponse{Contract: currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter), Size: 5, Mode: "single"}
	for _, payload := range []string{
		`{"contract":"BTC_USDT","size":"5","mode":"single"}`,
		`[{"contract":"BTC_USDT","size":"5","mode":"single"}]`,
	} {
		var reply positionReply
		require.NoErrorf(t, json.Unmarshal([]byte(payload), &reply), "Unmarshal must not error for %s", payload)
		assert.Equalf(t, exp, reply.Position, "positionReply should decode the position from %s", payload)
	}
	for _, payload := range []string{`[]`, `[{"contract":"BTC_USDT"},{"contract":"ETH_USDT"}]`} {
		var reply positionReply
		assert.ErrorIsf(t, json.Unmarshal([]byte(payload), &reply), common.ErrInvalidResponse, "positionReply should reject %s, which does not hold exactly one position", payload)
	}
}

func TestSetFuturesLeverage(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	_, err := e.SetFuturesLeverage(t.Context(), currency.EMPTYCODE, contract, 10, "cross", "dual_long")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SetFuturesLeverage must reject an empty settlement currency")

	_, err = e.SetFuturesLeverage(t.Context(), currency.USDT, currency.EMPTYPAIR, 10, "cross", "dual_long")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "SetFuturesLeverage must reject an empty contract")

	_, err = e.SetFuturesLeverage(t.Context(), currency.USDT, contract, 0, "cross", "dual_long")
	require.ErrorIs(t, err, errInvalidLeverage, "SetFuturesLeverage must reject a zero leverage")

	_, err = e.SetFuturesLeverage(t.Context(), currency.USDT, contract, 10, "CROSS", "dual_long")
	require.ErrorIs(t, err, errInvalidPositionMarginMode, "SetFuturesLeverage must reject an unknown margin mode")

	_, err = e.SetFuturesLeverage(t.Context(), currency.USDT, contract, 10, "cross", "short")
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "SetFuturesLeverage must reject an unknown dual side")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetFuturesLeverage(t.Context(), currency.USDT, contract, 10, "cross", "dual_long")
	require.NoError(t, err, "SetFuturesLeverage must not error")
	if mockTests {
		exp := futuresFixturePosition(getPair(t, asset.USDTMarginedFutures), "dual_long", -9440)
		assert.Equal(t, exp, result, "SetFuturesLeverage should decode every field")
		return
	}
	assert.NotNil(t, result, "SetFuturesLeverage should return the position")
}

func TestSwitchFuturesPositionMarginMode(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	_, err := e.SwitchFuturesPositionMarginMode(t.Context(), currency.EMPTYCODE, &FuturesPositionMarginModeRequest{Mode: "ISOLATED", Contract: contract})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SwitchFuturesPositionMarginMode must reject an empty settlement currency")

	_, err = e.SwitchFuturesPositionMarginMode(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SwitchFuturesPositionMarginMode must reject a nil request")

	_, err = e.SwitchFuturesPositionMarginMode(t.Context(), currency.USDT, &FuturesPositionMarginModeRequest{Mode: "isolated", Contract: contract})
	require.ErrorIs(t, err, errInvalidPositionMarginMode, "SwitchFuturesPositionMarginMode must reject a margin mode in the wrong case")

	_, err = e.SwitchFuturesPositionMarginMode(t.Context(), currency.USDT, &FuturesPositionMarginModeRequest{Mode: "ISOLATED"})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "SwitchFuturesPositionMarginMode must reject an empty contract")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SwitchFuturesPositionMarginMode(t.Context(), currency.USDT, &FuturesPositionMarginModeRequest{Mode: "ISOLATED", Contract: contract})
	require.NoError(t, err, "SwitchFuturesPositionMarginMode must not error")
	if mockTests {
		exp := futuresFixturePosition(getPair(t, asset.USDTMarginedFutures), "single", -9440)
		assert.Equal(t, exp, result, "SwitchFuturesPositionMarginMode should decode every field")
		return
	}
	assert.NotNil(t, result, "SwitchFuturesPositionMarginMode should return the position")
}

func TestSwitchFuturesDualModePositionMarginMode(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	_, err := e.SwitchFuturesDualModePositionMarginMode(t.Context(), currency.EMPTYCODE, &FuturesPositionMarginModeRequest{Mode: "CROSS", Contract: contract})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SwitchFuturesDualModePositionMarginMode must reject an empty settlement currency")

	_, err = e.SwitchFuturesDualModePositionMarginMode(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SwitchFuturesDualModePositionMarginMode must reject a nil request")

	_, err = e.SwitchFuturesDualModePositionMarginMode(t.Context(), currency.USDT, &FuturesPositionMarginModeRequest{Mode: "PORTFOLIO", Contract: contract})
	require.ErrorIs(t, err, errInvalidPositionMarginMode, "SwitchFuturesDualModePositionMarginMode must reject an unknown margin mode")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SwitchFuturesDualModePositionMarginMode(t.Context(), currency.USDT, &FuturesPositionMarginModeRequest{Mode: "CROSS", Contract: contract})
	require.NoError(t, err, "SwitchFuturesDualModePositionMarginMode must not error")
	if mockTests {
		exp := []*PositionResponse{futuresFixturePosition(getPair(t, asset.USDTMarginedFutures), "dual_long", 10), futuresFixturePosition(getPair(t, asset.USDTMarginedFutures), "dual_short", -10)}
		assert.Equal(t, exp, result, "SwitchFuturesDualModePositionMarginMode should decode every field")
	}
}

func TestFuturesPositionMarginModeRequestValidate(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, (*FuturesPositionMarginModeRequest)(nil).validate(), common.ErrNilPointer, "validate should reject a nil request")
	assert.ErrorIs(t, (&FuturesPositionMarginModeRequest{Mode: "cross", Contract: currency.NewBTCUSDT()}).validate(), errInvalidPositionMarginMode, "validate should reject a margin mode in the wrong case")
	assert.ErrorIs(t, (&FuturesPositionMarginModeRequest{Mode: "CROSS"}).validate(), currency.ErrCurrencyPairEmpty, "validate should reject an empty contract")
	for _, mode := range []string{"ISOLATED", "CROSS"} {
		assert.NoErrorf(t, (&FuturesPositionMarginModeRequest{Mode: mode, Contract: currency.NewBTCUSDT()}).validate(), "validate should accept the %s margin mode", mode)
	}
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()

	err := e.SetLeverage(t.Context(), asset.Spot, getPair(t, asset.Spot), margin.Isolated, 1, order.UnknownSide)
	require.ErrorIs(t, err, asset.ErrNotSupported, "SetLeverage must reject spot")

	err = e.SetLeverage(t.Context(), asset.CoinMarginedFutures, getPair(t, asset.CoinMarginedFutures), margin.NoMargin, 1, order.UnknownSide)
	require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported, "SetLeverage must reject an unsupported margin type")

	err = e.SetLeverage(t.Context(), asset.CoinMarginedFutures, getPair(t, asset.CoinMarginedFutures), margin.Isolated, 0, order.UnknownSide)
	require.ErrorIs(t, err, errInvalidLeverage, "SetLeverage must reject a leverage of 0")

	err = e.SetLeverage(t.Context(), asset.CoinMarginedFutures, currency.EMPTYPAIR, margin.Isolated, 1, order.UnknownSide)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "SetLeverage must reject an empty pair")

	err = e.SetLeverage(t.Context(), asset.DeliveryFutures, getPair(t, asset.DeliveryFutures), margin.Isolated, 0, order.UnknownSide)
	require.ErrorIs(t, err, errInvalidLeverage, "SetLeverage must reject a delivery leverage of 0")

	err = e.SetLeverage(t.Context(), asset.DeliveryFutures, getPair(t, asset.DeliveryFutures), margin.Multi, 1, order.UnknownSide)
	require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported, "SetLeverage must reject cross margin for delivery")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		a        asset.Item
		mt       margin.Type
		leverage float64
	}{
		{a: asset.CoinMarginedFutures, mt: margin.Isolated, leverage: 1},
		{a: asset.USDTMarginedFutures, mt: margin.Multi, leverage: 5},
		{a: asset.DeliveryFutures, mt: margin.Isolated, leverage: 1},
	} {
		err = e.SetLeverage(t.Context(), tc.a, getPair(t, tc.a), tc.mt, tc.leverage, order.UnknownSide)
		assert.NoErrorf(t, err, "SetLeverage should not error for %s %s", tc.a, tc.mt)
	}
}

// TestSetLeverageRequests checks the leverage each margin type sends, which the mock fixtures cannot tell apart since
// they match a POST by its empty body
func TestSetLeverageRequests(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	require.NoError(t, storeTestPairs(ex), "storeTestPairs must not error")
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "test-api-key", Secret: "test-api-secret"})
	var mu sync.Mutex
	var gotPath string
	var gotQuery url.Values
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "leverage request method should be POST")
		mu.Lock()
		gotPath, gotQuery = r.URL.Path, r.URL.Query()
		mu.Unlock()
		_, err := fmt.Fprint(w, `{"contract":"BTC_USDT","leverage":"0"}`)
		assert.NoError(t, err, "writing the position should not error")
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")

	pairs := mockTestPairs()
	for _, tc := range []struct {
		a        asset.Item
		mt       margin.Type
		expPath  string
		expQuery url.Values
	}{
		// Isolated margin sets the leverage itself, and cross margin a zero leverage capped by the cross leverage limit
		{a: asset.USDTMarginedFutures, mt: margin.Isolated, expPath: "/api/v4/futures/usdt/positions/BTC_USDT/leverage", expQuery: url.Values{"leverage": {"5"}}},
		{a: asset.USDTMarginedFutures, mt: margin.Unset, expPath: "/api/v4/futures/usdt/positions/BTC_USDT/leverage", expQuery: url.Values{"leverage": {"5"}}},
		{a: asset.USDTMarginedFutures, mt: margin.Multi, expPath: "/api/v4/futures/usdt/positions/BTC_USDT/leverage", expQuery: url.Values{"leverage": {"0"}, "cross_leverage_limit": {"5"}}},
		{a: asset.CoinMarginedFutures, mt: margin.Multi, expPath: "/api/v4/futures/btc/positions/BTC_USD/leverage", expQuery: url.Values{"leverage": {"0"}, "cross_leverage_limit": {"5"}}},
		{a: asset.DeliveryFutures, mt: margin.Isolated, expPath: "/api/v4/delivery/usdt/positions/BTC_USDT_20260626/leverage", expQuery: url.Values{"leverage": {"5"}}},
	} {
		require.NoErrorf(t, ex.SetLeverage(t.Context(), tc.a, pairs[tc.a], tc.mt, 5, order.UnknownSide), "SetLeverage must not error for %s %s", tc.a, tc.mt)
		mu.Lock()
		assert.Equalf(t, tc.expPath, gotPath, "SetLeverage should set the %s %s leverage of the contract", tc.a, tc.mt)
		assert.Equalf(t, tc.expQuery, gotQuery, "SetLeverage should send the %s %s leverage", tc.a, tc.mt)
		mu.Unlock()
	}
}

func TestPlaceDeliveryOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceDeliveryOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceDeliveryOrder must reject a nil request")

	_, err = e.PlaceDeliveryOrder(t.Context(), &DeliveryOrderCreateRequest{Size: 1, Price: 1, Settle: currency.USDT})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "PlaceDeliveryOrder must reject an empty contract")

	pair := getPair(t, asset.DeliveryFutures)
	_, err = e.PlaceDeliveryOrder(t.Context(), &DeliveryOrderCreateRequest{Contract: pair, Size: 1, Price: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PlaceDeliveryOrder must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PlaceDeliveryOrder(t.Context(), &DeliveryOrderCreateRequest{
		Contract:    pair,
		Size:        6024,
		Iceberg:     0,
		Price:       3765,
		Text:        "t-my-custom-id",
		Settle:      currency.USDT,
		TimeInForce: gtcTIF,
	})
	require.NoError(t, err, "PlaceDeliveryOrder must not error")
	if !mockTests {
		assert.NotZero(t, result.ID, "PlaceDeliveryOrder should return an order ID")
		return
	}
	assert.Equal(t, deliveryOpenOrderFixture(pair, 15675394), result, "PlaceDeliveryOrder should decode every field")

	// The recorded request body only matches a market order sending its zero price as "0"
	result, err = e.PlaceDeliveryOrder(t.Context(), &DeliveryOrderCreateRequest{
		Contract:    pair,
		Size:        -1,
		Text:        "t-market-order",
		Settle:      currency.USDT,
		TimeInForce: iocTIF,
	})
	require.NoError(t, err, "PlaceDeliveryOrder must not error for a market order")
	exp := &FuturesOrderResponse{
		ID:                        15675395,
		Contract:                  pair,
		CreateTime:                types.Time(time.Unix(1546569968, 0)),
		Size:                      -1,
		FillPrice:                 67850,
		MakerFeeRate:              -0.00025,
		TakerFeeRate:              0.00075,
		TimeInForce:               iocTIF,
		Text:                      "t-market-order",
		Status:                    "finished",
		FinishTime:                types.Time(time.Unix(1546569969, 0)),
		FinishAs:                  "filled",
		SelfTradePreventionAction: "-",
		AmendText:                 "-",
	}
	assert.Equal(t, exp, result, "PlaceDeliveryOrder should decode a filled market order")
}

func TestDeliveryOrderCreateParamsMarshalJSON(t *testing.T) {
	t.Parallel()
	pair := currency.NewPairWithDelimiter("BTC", "USDT_20260626", currency.UnderscoreDelimiter)
	for _, tc := range []struct {
		name   string
		params *DeliveryOrderCreateRequest
		exp    string
	}{
		{
			name:   "market order",
			params: &DeliveryOrderCreateRequest{Contract: pair, Size: -1, TimeInForce: iocTIF, Settle: currency.USDT},
			exp:    `{"contract":"BTC_USDT_20260626","size":-1,"iceberg":0,"price":"0","tif":"ioc"}`,
		},
		{
			name:   "close position",
			params: &DeliveryOrderCreateRequest{Contract: pair, ClosePosition: true, TimeInForce: iocTIF, Settle: currency.USDT},
			exp:    `{"contract":"BTC_USDT_20260626","size":0,"iceberg":0,"price":"0","tif":"ioc","close":true}`,
		},
		{
			name:   "close hedge mode side",
			params: &DeliveryOrderCreateRequest{Contract: pair, AutoSize: "close_long", TimeInForce: iocTIF, Settle: currency.USDT},
			exp:    `{"contract":"BTC_USDT_20260626","size":0,"iceberg":0,"price":"0","tif":"ioc","auto_size":"close_long"}`,
		},
		{
			name: "limit order",
			params: &DeliveryOrderCreateRequest{
				Contract:                  pair,
				Size:                      6024,
				Iceberg:                   10,
				Price:                     3765.5,
				TimeInForce:               gtcTIF,
				Text:                      "t-my-custom-id",
				ReduceOnly:                true,
				Settle:                    currency.USDT,
				SelfTradePreventionAction: "cn",
			},
			exp: `{"contract":"BTC_USDT_20260626","size":6024,"iceberg":10,"price":"3765.5","tif":"gtc","text":"t-my-custom-id","reduce_only":true,"stp_act":"cn"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tc.params)
			require.NoError(t, err, "Marshal must not error")
			assert.JSONEq(t, tc.exp, string(got), "Marshal should send the documented encoding")
		})
	}
}

func TestGetDeliveryOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryOrders(t.Context(), currency.EMPTYPAIR, statusOpen, currency.EMPTYCODE, "", 0, 0, true)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDeliveryOrders must reject an empty settlement currency")

	_, err = e.GetDeliveryOrders(t.Context(), getPair(t, asset.DeliveryFutures), "", currency.USDT, "", 0, 0, true)
	require.ErrorIs(t, err, errInvalidOrderStatus, "GetDeliveryOrders must reject an empty status")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	pair := getPair(t, asset.DeliveryFutures)
	result, err := e.GetDeliveryOrders(t.Context(), pair, statusOpen, currency.USDT, "", 0, 0, true)
	require.NoError(t, err, "GetDeliveryOrders must not error")
	if mockTests {
		assert.Equal(t, []*FuturesOrderResponse{deliveryOpenOrderFixture(pair, 15675394)}, result, "GetDeliveryOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDeliveryOrders should return a list")
}

func TestCancelMultipleDeliveryOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMultipleDeliveryOrders(t.Context(), currency.EMPTYPAIR, "ask", currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CancelMultipleDeliveryOrders must reject an empty settlement currency")

	_, err = e.CancelMultipleDeliveryOrders(t.Context(), currency.EMPTYPAIR, "ask", currency.USDT)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelMultipleDeliveryOrders must reject an empty contract")

	pair := getPair(t, asset.DeliveryFutures)
	_, err = e.CancelMultipleDeliveryOrders(t.Context(), pair, "buy", currency.USDT)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "CancelMultipleDeliveryOrders must reject a side other than ask or bid")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelMultipleDeliveryOrders(t.Context(), pair, "ask", currency.USDT)
	require.NoError(t, err, "CancelMultipleDeliveryOrders must not error")
	if mockTests {
		assert.Equal(t, []*FuturesOrderResponse{deliveryCancelledOrderFixture(pair)}, result, "CancelMultipleDeliveryOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelMultipleDeliveryOrders should return a list")
}

func TestGetSingleDeliveryOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleDeliveryOrder(t.Context(), currency.EMPTYCODE, "123456")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetSingleDeliveryOrder must reject an empty settlement currency")

	_, err = e.GetSingleDeliveryOrder(t.Context(), currency.USDT, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetSingleDeliveryOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSingleDeliveryOrder(t.Context(), currency.USDT, "123456")
	require.NoError(t, err, "GetSingleDeliveryOrder must not error")
	if mockTests {
		assert.Equal(t, deliveryOpenOrderFixture(getPair(t, asset.DeliveryFutures), 123456), result, "GetSingleDeliveryOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSingleDeliveryOrder should return an order")
}

func TestCancelSingleDeliveryOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelSingleDeliveryOrder(t.Context(), currency.EMPTYCODE, "123456")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CancelSingleDeliveryOrder must reject an empty settlement currency")

	_, err = e.CancelSingleDeliveryOrder(t.Context(), currency.USDT, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelSingleDeliveryOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelSingleDeliveryOrder(t.Context(), currency.USDT, "123456")
	require.NoError(t, err, "CancelSingleDeliveryOrder must not error")
	if mockTests {
		assert.Equal(t, deliveryCancelledOrderFixture(getPair(t, asset.DeliveryFutures)), result, "CancelSingleDeliveryOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelSingleDeliveryOrder should return an order")
}

// deliveryOpenOrderFixture returns the open delivery order recorded in testdata/http.json
func deliveryOpenOrderFixture(contract currency.Pair, id uint64) *FuturesOrderResponse {
	return &FuturesOrderResponse{
		ID:                        id,
		Contract:                  contract,
		CreateTime:                types.Time(time.Unix(1546569968, 0)),
		Size:                      6024,
		RemainingAmount:           6024,
		Price:                     3765,
		MakerFeeRate:              -0.00025,
		TakerFeeRate:              0.00075,
		TimeInForce:               gtcTIF,
		Text:                      "t-my-custom-id",
		Status:                    statusOpen,
		SelfTradePreventionAction: "-",
		AmendText:                 "-",
	}
}

// deliveryCancelledOrderFixture returns the cancelled delivery order recorded in testdata/http.json
func deliveryCancelledOrderFixture(contract currency.Pair) *FuturesOrderResponse {
	return &FuturesOrderResponse{
		ID:                        123456,
		Contract:                  contract,
		CreateTime:                types.Time(time.Unix(1546569968, 0)),
		Size:                      -100,
		RemainingAmount:           -100,
		Price:                     3800,
		MakerFeeRate:              -0.00025,
		TakerFeeRate:              0.00075,
		TimeInForce:               gtcTIF,
		ReferenceUserID:           1001,
		IsReduceOnly:              true,
		Text:                      "t-my-custom-id",
		Status:                    statusFinished,
		FinishTime:                types.Time(time.Unix(1546570000, 0)),
		FinishAs:                  "cancelled",
		SelfTradePreventionID:     7,
		SelfTradePreventionAction: "cn",
		AmendText:                 "t-amended",
	}
}

func TestGetMyDeliveryTradingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetMyDeliveryTradingHistory(t.Context(), currency.EMPTYCODE, "", getPair(t, asset.DeliveryFutures), 0, 0, 1, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMyDeliveryTradingHistory must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMyDeliveryTradingHistory(t.Context(), currency.USDT, "", getPair(t, asset.DeliveryFutures), 0, 0, 1, "")
	require.NoError(t, err, "GetMyDeliveryTradingHistory must not error")
	if mockTests {
		exp := []*TradingHistoryItemResponse{{
			ID:       121234231,
			Contract: "BTC_USDT_20260626",
			Text:     "t-123456",
			Size:     100,
			Price:    100.123,
			Fee:      0.01,
			PointFee: 0.002,
			Role:     "taker",
		}}
		assert.Equal(t, exp, result, "GetMyDeliveryTradingHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMyDeliveryTradingHistory should return a list")
}

func TestGetDeliveryPositionCloseHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryPositionCloseHistory(t.Context(), currency.EMPTYCODE, getPair(t, asset.DeliveryFutures), 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDeliveryPositionCloseHistory must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDeliveryPositionCloseHistory(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures), 0)
	require.NoError(t, err, "GetDeliveryPositionCloseHistory must not error")
	if mockTests {
		exp := []*PositionCloseHistoryResponse{{
			Time:          types.Time(time.Unix(1546487347, 0)),
			ProfitAndLoss: 0.00013,
			Side:          "long",
			Contract:      "BTC_USDT_20260626",
			Text:          "web",
		}}
		assert.Equal(t, exp, result, "GetDeliveryPositionCloseHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDeliveryPositionCloseHistory should return a list")
}

func TestGetDeliveryLiquidationHistory(t *testing.T) {
	t.Parallel()
	at, _ := getTime()
	_, err := e.GetDeliveryLiquidationHistory(t.Context(), currency.EMPTYCODE, getPair(t, asset.DeliveryFutures), 0, at)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDeliveryLiquidationHistory must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	pair := getPair(t, asset.DeliveryFutures)
	result, err := e.GetDeliveryLiquidationHistory(t.Context(), currency.USDT, pair, 0, at)
	require.NoError(t, err, "GetDeliveryLiquidationHistory must not error")
	if mockTests {
		exp := []*LiquidationHistoryItemResponse{{
			Time:             types.Time(time.Unix(1744103854, 0)),
			Contract:         pair,
			Size:             600,
			Leverage:         25,
			Margin:           0.006705256878,
			EntryPrice:       3536.123,
			MarkPrice:        3420.27,
			OrderPrice:       3405,
			FillPrice:        3424,
			LiquidationPrice: 3421.54,
			OrderID:          317393847,
			RemainingAmount:  4,
		}}
		assert.Equal(t, exp, result, "GetDeliveryLiquidationHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDeliveryLiquidationHistory should return a list")
}

func TestGetDeliverySettlementHistory(t *testing.T) {
	t.Parallel()
	at, _ := getTime()
	_, err := e.GetDeliverySettlementHistory(t.Context(), currency.EMPTYCODE, getPair(t, asset.DeliveryFutures), 0, at)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDeliverySettlementHistory must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	pair := getPair(t, asset.DeliveryFutures)
	result, err := e.GetDeliverySettlementHistory(t.Context(), currency.USDT, pair, 0, at)
	require.NoError(t, err, "GetDeliverySettlementHistory must not error")
	if mockTests {
		exp := []*SettlementHistoryItemResponse{{
			Time:        types.Time(time.Unix(1744103854, 0)),
			Contract:    pair,
			Size:        600,
			Leverage:    25,
			Margin:      0.006705256878,
			EntryPrice:  3536.123,
			SettlePrice: 3421.54,
			Profit:      -6.87498,
			Fee:         0.03079386,
		}}
		assert.Equal(t, exp, result, "GetDeliverySettlementHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDeliverySettlementHistory should return a list")
}

func TestCreatePriceTriggeredDeliveryOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CreatePriceTriggeredDeliveryOrder(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreatePriceTriggeredDeliveryOrder must reject a nil request")

	pair := getPair(t, asset.DeliveryFutures)
	newRequest := func() *FuturesPriceTriggeredOrderRequest {
		return &FuturesPriceTriggeredOrderRequest{
			Initial:   FuturesInitial{Contract: pair, Size: 12, Price: 1234},
			Trigger:   FuturesTrigger{Rule: 1, Price: 123400},
			OrderType: "close-short-position",
		}
	}
	for _, tc := range []struct {
		name   string
		settle currency.Code
		modify func(*FuturesPriceTriggeredOrderRequest)
		err    error
	}{
		{name: "settlement currency", settle: currency.EMPTYCODE, modify: func(*FuturesPriceTriggeredOrderRequest) {}, err: currency.ErrCurrencyCodeEmpty},
		{name: "contract", settle: currency.USDT, modify: func(p *FuturesPriceTriggeredOrderRequest) { p.Initial.Contract = currency.EMPTYPAIR }, err: currency.ErrCurrencyPairEmpty},
		{name: "price", settle: currency.USDT, modify: func(p *FuturesPriceTriggeredOrderRequest) { p.Initial.Price = -1 }, err: limits.ErrPriceBelowMin},
		{name: "time in force", settle: currency.USDT, modify: func(p *FuturesPriceTriggeredOrderRequest) { p.Initial.TimeInForce = pocTIF }, err: order.ErrUnsupportedTimeInForce},
		{name: "strategy type", settle: currency.USDT, modify: func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.StrategyType = 2 }, err: errInvalidTriggerStrategyType},
		{name: "price type", settle: currency.USDT, modify: func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.PriceType = 3 }, err: errInvalidTriggerPriceType},
		{name: "rule", settle: currency.USDT, modify: func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.Rule = 0 }, err: errInvalidTriggerRule},
		{name: "trigger price", settle: currency.USDT, modify: func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.Price = 0 }, err: limits.ErrPriceBelowMin},
		{name: "read only order type", settle: currency.USDT, modify: func(p *FuturesPriceTriggeredOrderRequest) { p.OrderType = "close-long-order" }, err: errInvalidAutoOrderType},
	} {
		arg := newRequest()
		tc.modify(arg)
		_, err = e.CreatePriceTriggeredDeliveryOrder(t.Context(), tc.settle, arg)
		require.ErrorIsf(t, err, tc.err, "CreatePriceTriggeredDeliveryOrder must reject an invalid %s", tc.name)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		result, err := e.CreatePriceTriggeredDeliveryOrder(t.Context(), currency.USDT, newRequest())
		require.NoError(t, err, "CreatePriceTriggeredDeliveryOrder must not error")
		assert.NotZero(t, result.ID, "CreatePriceTriggeredDeliveryOrder should return an order ID")
		return
	}

	// The initial order's wire encoding belongs to the shared FuturesInitial type, so the request is checked here
	// rather than against a recorded body
	var requests atomic.Int64
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		assert.Equal(t, http.MethodPost, r.Method, "request method should be POST")
		assert.Equal(t, "/api/v4/delivery/usdt/price_orders", r.URL.Path, "request path should be the delivery price orders endpoint")
		var body struct {
			Initial   map[string]any `json:"initial"`
			Trigger   map[string]any `json:"trigger"`
			OrderType string         `json:"order_type"`
		}
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "request body should decode")
		assert.Equal(t, "close-short-position", body.OrderType, "order type should be sent at the top level")
		assert.Equal(t, map[string]any{"price": "123400", "rule": float64(1)}, body.Trigger, "trigger should carry the price and rule")
		assert.Len(t, body.Initial, 3, "initial order should carry the contract, size and price")
		assert.Equal(t, "BTC_USDT_20260626", body.Initial["contract"], "initial order contract should match")
		assert.Equal(t, "1234", body.Initial["price"], "initial order price should be a decimal string")
		assert.Equal(t, "12", fmt.Sprint(body.Initial["size"]), "initial order size should match")
		_, err := w.Write([]byte(`{"id":1432329}`))
		assert.NoError(t, err, "writing the price order response should not error")
	}))
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	result, err := ex.CreatePriceTriggeredDeliveryOrder(t.Context(), currency.USDT, newRequest())
	require.NoError(t, err, "CreatePriceTriggeredDeliveryOrder must not error")
	assert.Equal(t, &TriggerOrderIDResponse{ID: 1432329}, result, "CreatePriceTriggeredDeliveryOrder should decode the order ID")
	assert.Equal(t, int64(1), requests.Load(), "CreatePriceTriggeredDeliveryOrder should send one request")
}

func TestGetDeliveryAllAutoOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryAllAutoOrder(t.Context(), "", currency.USDT, getPair(t, asset.DeliveryFutures), 0, 1)
	require.ErrorIs(t, err, errInvalidOrderStatus, "GetDeliveryAllAutoOrder must reject an empty status")

	_, err = e.GetDeliveryAllAutoOrder(t.Context(), statusOpen, currency.EMPTYCODE, getPair(t, asset.DeliveryFutures), 0, 1)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDeliveryAllAutoOrder must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	pair := getPair(t, asset.DeliveryFutures)
	result, err := e.GetDeliveryAllAutoOrder(t.Context(), statusOpen, currency.USDT, pair, 0, 1)
	require.NoError(t, err, "GetDeliveryAllAutoOrder must not error")
	if mockTests {
		assert.Equal(t, []*PriceTriggeredOrderResponse{deliveryOpenAutoOrderFixture(pair, 1283293)}, result, "GetDeliveryAllAutoOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDeliveryAllAutoOrder should return a list")
}

func TestCancelAllDeliveryPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllDeliveryPriceTriggeredOrder(t.Context(), currency.EMPTYCODE, getPair(t, asset.DeliveryFutures))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CancelAllDeliveryPriceTriggeredOrder must reject an empty settlement currency")

	_, err = e.CancelAllDeliveryPriceTriggeredOrder(t.Context(), currency.USDT, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelAllDeliveryPriceTriggeredOrder must reject an empty contract")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	pair := getPair(t, asset.DeliveryFutures)
	result, err := e.CancelAllDeliveryPriceTriggeredOrder(t.Context(), currency.USDT, pair)
	require.NoError(t, err, "CancelAllDeliveryPriceTriggeredOrder must not error")
	if mockTests {
		assert.Equal(t, []*PriceTriggeredOrderResponse{deliveryCancelledAutoOrderFixture(pair)}, result, "CancelAllDeliveryPriceTriggeredOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelAllDeliveryPriceTriggeredOrder should return a list")
}

func TestGetSingleDeliveryPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleDeliveryPriceTriggeredOrder(t.Context(), currency.EMPTYCODE, "12345")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetSingleDeliveryPriceTriggeredOrder must reject an empty settlement currency")

	_, err = e.GetSingleDeliveryPriceTriggeredOrder(t.Context(), currency.USDT, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetSingleDeliveryPriceTriggeredOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSingleDeliveryPriceTriggeredOrder(t.Context(), currency.USDT, "12345")
	require.NoError(t, err, "GetSingleDeliveryPriceTriggeredOrder must not error")
	if mockTests {
		assert.Equal(t, deliveryOpenAutoOrderFixture(getPair(t, asset.DeliveryFutures), 12345), result, "GetSingleDeliveryPriceTriggeredOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSingleDeliveryPriceTriggeredOrder should return an order")
}

func TestCancelDeliveryPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelDeliveryPriceTriggeredOrder(t.Context(), currency.EMPTYCODE, "12345")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CancelDeliveryPriceTriggeredOrder must reject an empty settlement currency")

	_, err = e.CancelDeliveryPriceTriggeredOrder(t.Context(), currency.USDT, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelDeliveryPriceTriggeredOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelDeliveryPriceTriggeredOrder(t.Context(), currency.USDT, "12345")
	require.NoError(t, err, "CancelDeliveryPriceTriggeredOrder must not error")
	if mockTests {
		assert.Equal(t, deliveryCancelledAutoOrderFixture(getPair(t, asset.DeliveryFutures)), result, "CancelDeliveryPriceTriggeredOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelDeliveryPriceTriggeredOrder should return an order")
}

// deliveryOpenAutoOrderFixture returns the open delivery price-triggered order recorded in testdata/http.json
func deliveryOpenAutoOrderFixture(contract currency.Pair, id uint64) *PriceTriggeredOrderResponse {
	o := &PriceTriggeredOrderResponse{
		ID:         id,
		User:       1234,
		CreateTime: types.Time(time.Unix(1514764800, 0)),
		Status:     statusOpen,
		OrderType:  "close-long-position",
	}
	o.Initial.Contract = contract
	o.Initial.Size = 100
	o.Initial.Price = 5.03
	o.Trigger.PriceType = 1
	o.Trigger.Price = 3000
	o.Trigger.Rule = 1
	o.Trigger.Expiration = 86400
	return o
}

// deliveryCancelledAutoOrderFixture returns the cancelled delivery price-triggered order recorded in testdata/http.json
func deliveryCancelledAutoOrderFixture(contract currency.Pair) *PriceTriggeredOrderResponse {
	o := deliveryOpenAutoOrderFixture(contract, 12345)
	o.FinishTime = types.Time(time.Unix(1514764900, 0))
	o.TradeID = 13566
	o.Status = statusFinished
	o.FinishAs = "cancelled"
	o.Reason = "cancelled by user"
	return o
}

func TestEnableOrDisableDualMode(t *testing.T) {
	t.Parallel()
	_, err := e.EnableOrDisableDualMode(t.Context(), currency.EMPTYCODE, true)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "EnableOrDisableDualMode must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.EnableOrDisableDualMode(t.Context(), currency.BTC, true)
	require.NoError(t, err, "EnableOrDisableDualMode must not error")
	if mockTests {
		exp := futuresFixtureAccount(currency.BTC)
		assert.Equal(t, exp, result, "EnableOrDisableDualMode should decode every field")
		return
	}
	assert.NotNil(t, result, "EnableOrDisableDualMode should return the account")
}

func TestSetFuturesPositionMode(t *testing.T) {
	t.Parallel()
	_, err := e.SetFuturesPositionMode(t.Context(), currency.EMPTYCODE, "dual_plus")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SetFuturesPositionMode must reject an empty settlement currency")

	_, err = e.SetFuturesPositionMode(t.Context(), currency.USDT, "hedge")
	require.ErrorIs(t, err, errInvalidPositionMode, "SetFuturesPositionMode must reject an unknown position mode")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetFuturesPositionMode(t.Context(), currency.USDT, "dual_plus")
	require.NoError(t, err, "SetFuturesPositionMode must not error")
	if mockTests {
		exp := futuresFixtureAccount(currency.USDT)
		assert.Equal(t, exp, result, "SetFuturesPositionMode should decode every field")
		return
	}
	assert.NotNil(t, result, "SetFuturesPositionMode should return the account")
}

func TestGetPositionDetailInDualMode(t *testing.T) {
	t.Parallel()
	_, err := e.GetPositionDetailInDualMode(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetPositionDetailInDualMode must reject an empty settlement currency")

	_, err = e.GetPositionDetailInDualMode(t.Context(), currency.BTC, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetPositionDetailInDualMode must reject an empty contract")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err = e.GetPositionDetailInDualMode(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures))
	require.NoError(t, err, "GetPositionDetailInDualMode must not error for a BTC contract")

	result, err := e.GetPositionDetailInDualMode(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures))
	require.NoError(t, err, "GetPositionDetailInDualMode must not error for a USDT contract")
	if mockTests {
		exp := []*PositionResponse{futuresFixturePosition(getPair(t, asset.USDTMarginedFutures), "dual_long", 10), futuresFixturePosition(getPair(t, asset.USDTMarginedFutures), "dual_short", -10)}
		assert.Equal(t, exp, result, "GetPositionDetailInDualMode should decode every field")
	}
}

func TestUpdatePositionMarginInDualMode(t *testing.T) {
	t.Parallel()
	_, err := e.UpdatePositionMarginInDualMode(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures), 0.001, "dual_long")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "UpdatePositionMarginInDualMode must reject an empty settlement currency")

	_, err = e.UpdatePositionMarginInDualMode(t.Context(), currency.BTC, currency.EMPTYPAIR, 0.001, "dual_long")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdatePositionMarginInDualMode must reject an empty contract")

	_, err = e.UpdatePositionMarginInDualMode(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0, "dual_long")
	require.ErrorIs(t, err, errInvalidMarginChange, "UpdatePositionMarginInDualMode must reject a zero change")

	_, err = e.UpdatePositionMarginInDualMode(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0.001, "")
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "UpdatePositionMarginInDualMode must reject an empty dual side")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err = e.UpdatePositionMarginInDualMode(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0.001, "dual_long")
	require.NoError(t, err, "UpdatePositionMarginInDualMode must not error for a BTC contract")

	result, err := e.UpdatePositionMarginInDualMode(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 0.001, "dual_long")
	require.NoError(t, err, "UpdatePositionMarginInDualMode must not error for a USDT contract")
	if mockTests {
		exp := []*PositionResponse{futuresFixturePosition(getPair(t, asset.USDTMarginedFutures), "dual_long", 10)}
		assert.Equal(t, exp, result, "UpdatePositionMarginInDualMode should decode every field")
	}
}

func TestUpdatePositionLeverageInDualMode(t *testing.T) {
	t.Parallel()
	_, err := e.UpdatePositionLeverageInDualMode(t.Context(), currency.EMPTYCODE, getPair(t, asset.USDTMarginedFutures), 1, 0.001)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "UpdatePositionLeverageInDualMode must reject an empty settlement currency")

	_, err = e.UpdatePositionLeverageInDualMode(t.Context(), currency.USDT, currency.EMPTYPAIR, 1, 0.001)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdatePositionLeverageInDualMode must reject an empty contract")

	_, err = e.UpdatePositionLeverageInDualMode(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), -1, 0.001)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported, "UpdatePositionLeverageInDualMode must reject a negative leverage")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err = e.UpdatePositionLeverageInDualMode(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 1, 0.001)
	require.NoError(t, err, "UpdatePositionLeverageInDualMode must not error for a BTC contract")

	result, err := e.UpdatePositionLeverageInDualMode(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 10, 0.001)
	require.NoError(t, err, "UpdatePositionLeverageInDualMode must not error for a USDT contract")
	if mockTests {
		exp := []*PositionResponse{futuresFixturePosition(getPair(t, asset.USDTMarginedFutures), "dual_long", 10), futuresFixturePosition(getPair(t, asset.USDTMarginedFutures), "dual_short", -10)}
		assert.Equal(t, exp, result, "UpdatePositionLeverageInDualMode should decode every field")
	}
}

func TestUpdatePositionRiskLimitInDualMode(t *testing.T) {
	t.Parallel()
	_, err := e.UpdatePositionRiskLimitInDualMode(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures), 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "UpdatePositionRiskLimitInDualMode must reject an empty settlement currency")

	_, err = e.UpdatePositionRiskLimitInDualMode(t.Context(), currency.BTC, currency.EMPTYPAIR, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdatePositionRiskLimitInDualMode must reject an empty contract")

	_, err = e.UpdatePositionRiskLimitInDualMode(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0)
	require.ErrorIs(t, err, errInvalidRiskLimit, "UpdatePositionRiskLimitInDualMode must reject a zero risk limit")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err = e.UpdatePositionRiskLimitInDualMode(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 20000)
	require.NoError(t, err, "UpdatePositionRiskLimitInDualMode must not error for a BTC contract")

	result, err := e.UpdatePositionRiskLimitInDualMode(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 20000)
	require.NoError(t, err, "UpdatePositionRiskLimitInDualMode must not error for a USDT contract")
	if mockTests {
		exp := []*PositionResponse{futuresFixturePosition(getPair(t, asset.USDTMarginedFutures), "dual_long", 10), futuresFixturePosition(getPair(t, asset.USDTMarginedFutures), "dual_short", -10)}
		assert.Equal(t, exp, result, "UpdatePositionRiskLimitInDualMode should decode every field")
	}
}

func TestPlaceFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceFuturesOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceFuturesOrder must reject a nil order")

	arg := &FuturesOrderCreateRequest{}
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "PlaceFuturesOrder must reject an empty contract")

	arg.Contract = getPair(t, asset.USDTMarginedFutures)
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidOrderSize, "PlaceFuturesOrder must reject an empty size")

	arg.Size = 1
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "PlaceFuturesOrder must reject a market order without an ioc or fok time in force")

	arg.TimeInForce = order.GoodTillCancel.Lower()
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "PlaceFuturesOrder must reject a market order with a gtc time in force")

	arg.Price = 123
	arg.Text = "client-order-id"
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidTextPrefix, "PlaceFuturesOrder must reject text without the t- prefix")

	arg.Text = "t-client-order-id"
	arg.AutoSize = "long"
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidAutoSize, "PlaceFuturesOrder must reject an unknown auto size")

	arg.AutoSize = ""
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PlaceFuturesOrder must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PlaceFuturesOrder(t.Context(), &FuturesOrderCreateRequest{
		Contract:    getPair(t, asset.CoinMarginedFutures),
		Size:        6024,
		Price:       3765,
		TimeInForce: "gtc",
		Text:        "t-my-custom-id",
		Settle:      currency.BTC,
	})
	require.NoError(t, err, "PlaceFuturesOrder must not error")
	if mockTests {
		exp := futuresFixtureOrder(getPair(t, asset.CoinMarginedFutures))
		assert.Equal(t, exp, result, "PlaceFuturesOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "PlaceFuturesOrder should return the order")
}

func TestFuturesOrderCreateParamsMarshalJSON(t *testing.T) {
	t.Parallel()
	contract := currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter)
	for _, tc := range []struct {
		name string
		arg  FuturesOrderCreateRequest
		exp  string
	}{
		{
			name: "market order sends a zero price and iceberg",
			arg:  FuturesOrderCreateRequest{Contract: contract, Size: 10, TimeInForce: iocTIF, Text: "t-x"},
			exp:  `{"contract":"BTC_USDT","size":"10","iceberg":"0","price":"0","tif":"ioc","text":"t-x"}`,
		},
		{
			name: "close order sends a zero size",
			arg:  FuturesOrderCreateRequest{Contract: contract, TimeInForce: iocTIF, ClosePosition: true, ReduceOnly: true},
			exp:  `{"contract":"BTC_USDT","size":"0","iceberg":"0","price":"0","tif":"ioc","close":true,"reduce_only":true}`,
		},
		{
			name: "limit order sends every optional field",
			arg: FuturesOrderCreateRequest{
				Contract: contract, Size: -1.5, Iceberg: 0.5, Price: 31503.3, TimeInForce: gtcTIF, Text: "t-y", AutoSize: "close_long",
				Settle: currency.USDT, SelfTradePreventionAction: "cn", PositionID: 7, MarketOrderSlipRatio: 0.03, PositionMarginMode: "isolated",
				ActionMode: "ACK", TakeProfitTriggerPrice: 32000, StopLossTriggerPrice: 31000,
			},
			exp: `{"contract":"BTC_USDT","size":"-1.5","iceberg":"0.5","price":"31503.3","tif":"gtc","text":"t-y","auto_size":"close_long","stp_act":"cn","pid":7,"market_order_slip_ratio":"0.03","pos_margin_mode":"isolated","action_mode":"ACK","tpsl_tp_trigger_price":"32000","tpsl_sl_trigger_price":"31000"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			payload, err := json.Marshal(tc.arg)
			require.NoError(t, err, "Marshal must not error")
			assert.Equal(t, tc.exp, string(payload), "FuturesOrderCreateRequest should marshal to the documented body")
			pointerPayload, err := json.Marshal(&tc.arg)
			require.NoError(t, err, "Marshal must not error for a pointer")
			assert.Equal(t, tc.exp, string(pointerPayload), "a FuturesOrderCreateRequest pointer should marshal like a value")
		})
	}
}

func TestNumericStringUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		payload string
		exp     NumericString
	}{
		{payload: `"110xxxxx"`, exp: "110xxxxx"},
		{payload: `"94294117233225616"`, exp: "94294117233225616"},
		{payload: `94294117233225616`, exp: "94294117233225616"},
		{payload: `-12`, exp: "-12"},
		{payload: `null`, exp: ""},
	} {
		var n NumericString
		require.NoErrorf(t, json.Unmarshal([]byte(tc.payload), &n), "Unmarshal must not error for %s", tc.payload)
		assert.Equalf(t, tc.exp, n, "NumericString should keep the text of %s", tc.payload)
		assert.Equalf(t, string(tc.exp), n.String(), "NumericString.String should return the text of %s", tc.payload)
	}
	for _, payload := range []string{`1.5`, `true`} {
		var n NumericString
		assert.ErrorIsf(t, json.Unmarshal([]byte(payload), &n), errInvalidNumericString, "NumericString should reject %s", payload)
	}
}

func TestGetFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrders(t.Context(), getPair(t, asset.CoinMarginedFutures), statusOpen, "", currency.EMPTYCODE, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesOrders must reject an empty settlement currency")

	_, err = e.GetFuturesOrders(t.Context(), getPair(t, asset.CoinMarginedFutures), "", "", currency.BTC, 0, 0)
	require.ErrorIs(t, err, errInvalidOrderStatus, "GetFuturesOrders must reject an empty status")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesOrders(t.Context(), getPair(t, asset.CoinMarginedFutures), statusOpen, "", currency.BTC, 0, 0)
	require.NoError(t, err, "GetFuturesOrders must not error")
	if mockTests {
		openOrder := futuresFixtureOrder(getPair(t, asset.CoinMarginedFutures))
		openOrder.Status = statusOpen
		exp := []*FuturesOrderResponse{openOrder}
		assert.Equal(t, exp, result, "GetFuturesOrders should decode every field")
	}
}

func TestCancelMultipleFuturesOpenOrders(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	_, err := e.CancelMultipleFuturesOpenOrders(t.Context(), currency.EMPTYCODE, &CancelFuturesOrdersRequest{Contract: contract, Side: "ask"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CancelMultipleFuturesOpenOrders must reject an empty settlement currency")

	_, err = e.CancelMultipleFuturesOpenOrders(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelMultipleFuturesOpenOrders must reject a nil request")

	_, err = e.CancelMultipleFuturesOpenOrders(t.Context(), currency.USDT, &CancelFuturesOrdersRequest{Contract: contract, Side: "sell"})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "CancelMultipleFuturesOpenOrders must reject a side other than bid or ask")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelMultipleFuturesOpenOrders(t.Context(), currency.USDT, &CancelFuturesOrdersRequest{Contract: contract, Side: "ask"})
	require.NoError(t, err, "CancelMultipleFuturesOpenOrders must not error")
	if mockTests {
		exp := []*FuturesOrderResponse{futuresFixtureOrder(getPair(t, asset.USDTMarginedFutures))}
		assert.Equal(t, exp, result, "CancelMultipleFuturesOpenOrders should decode every field")
	}
}

func TestGetFuturesOrdersByTimeRange(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrdersByTimeRange(t.Context(), currency.EMPTYCODE, getPair(t, asset.USDTMarginedFutures), 0, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesOrdersByTimeRange must reject an empty settlement currency")

	startTime, endTime := getTime()
	_, err = e.GetFuturesOrdersByTimeRange(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 0, 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesOrdersByTimeRange must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesOrdersByTimeRange(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 10, 0, startTime, endTime)
	require.NoError(t, err, "GetFuturesOrdersByTimeRange must not error")
	if mockTests {
		exp := []*FuturesOrderResponse{futuresFixtureOrder(getPair(t, asset.USDTMarginedFutures))}
		assert.Equal(t, exp, result, "GetFuturesOrdersByTimeRange should decode every field")
	}
}

func TestPlaceBatchFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceBatchFuturesOrders(t.Context(), currency.EMPTYCODE, []FuturesOrderCreateRequest{{}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PlaceBatchFuturesOrders must reject an empty settlement currency")

	_, err = e.PlaceBatchFuturesOrders(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, errOrdersEmpty, "PlaceBatchFuturesOrders must reject an empty batch")

	_, err = e.PlaceBatchFuturesOrders(t.Context(), currency.USDT, make([]FuturesOrderCreateRequest, 11))
	require.ErrorIs(t, err, errTooManyOrderRequest, "PlaceBatchFuturesOrders must reject more than 10 orders")

	arg := FuturesOrderCreateRequest{}
	_, err = e.PlaceBatchFuturesOrders(t.Context(), currency.USDT, []FuturesOrderCreateRequest{arg})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "PlaceBatchFuturesOrders must reject an empty contract")

	arg.Contract = getPair(t, asset.USDTMarginedFutures)
	_, err = e.PlaceBatchFuturesOrders(t.Context(), currency.USDT, []FuturesOrderCreateRequest{arg})
	require.ErrorIs(t, err, errInvalidOrderSize, "PlaceBatchFuturesOrders must reject an empty size")

	arg.Size = 1
	_, err = e.PlaceBatchFuturesOrders(t.Context(), currency.USDT, []FuturesOrderCreateRequest{arg})
	require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "PlaceBatchFuturesOrders must reject a market order without an ioc or fok time in force")

	arg.TimeInForce = order.GoodTillCancel.Lower()
	_, err = e.PlaceBatchFuturesOrders(t.Context(), currency.USDT, []FuturesOrderCreateRequest{arg})
	require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "PlaceBatchFuturesOrders must reject a market order with a gtc time in force")

	arg.Price = 123
	arg.Text = "client-order-id"
	_, err = e.PlaceBatchFuturesOrders(t.Context(), currency.USDT, []FuturesOrderCreateRequest{arg})
	require.ErrorIs(t, err, errInvalidTextPrefix, "PlaceBatchFuturesOrders must reject text without the t- prefix")

	arg.Text = "t-client-order-id"
	arg.AutoSize = "long"
	_, err = e.PlaceBatchFuturesOrders(t.Context(), currency.USDT, []FuturesOrderCreateRequest{arg})
	require.ErrorIs(t, err, errInvalidAutoSize, "PlaceBatchFuturesOrders must reject an unknown auto size")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PlaceBatchFuturesOrders(t.Context(), currency.BTC, []FuturesOrderCreateRequest{
		{
			Contract:    getPair(t, asset.CoinMarginedFutures),
			Size:        6024,
			Price:       3765,
			TimeInForce: "gtc",
			Text:        "t-my-custom-id",
			Settle:      currency.BTC,
		},
		{
			Contract:    getPair(t, asset.CoinMarginedFutures),
			Size:        232,
			Price:       376225,
			TimeInForce: "gtc",
			Text:        "t-my-custom-id",
			Settle:      currency.BTC,
		},
	})
	require.NoError(t, err, "PlaceBatchFuturesOrders must not error")
	if mockTests {
		exp := []*BatchFuturesOrderResponse{
			{Succeeded: true, FuturesOrderResponse: *futuresFixtureOrder(getPair(t, asset.CoinMarginedFutures))},
			{Label: "INSUFFICIENT_AVAILABLE", Detail: "available balance not enough"},
		}
		assert.Equal(t, exp, result, "PlaceBatchFuturesOrders should decode every field")
		return
	}
	assert.Len(t, result, 2, "PlaceBatchFuturesOrders should report each order")
}

func TestGetSingleFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleFuturesOrder(t.Context(), currency.EMPTYCODE, "12345")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetSingleFuturesOrder must reject an empty settlement currency")

	_, err = e.GetSingleFuturesOrder(t.Context(), currency.BTC, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetSingleFuturesOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSingleFuturesOrder(t.Context(), currency.BTC, "12345")
	require.NoError(t, err, "GetSingleFuturesOrder must not error")
	if mockTests {
		exp := futuresFixtureOrder(getPair(t, asset.CoinMarginedFutures))
		assert.Equal(t, exp, result, "GetSingleFuturesOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSingleFuturesOrder should return the order")
}

func TestCancelSingleFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelSingleFuturesOrder(t.Context(), currency.EMPTYCODE, "12345", "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CancelSingleFuturesOrder must reject an empty settlement currency")

	_, err = e.CancelSingleFuturesOrder(t.Context(), currency.BTC, "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelSingleFuturesOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelSingleFuturesOrder(t.Context(), currency.BTC, "12345", "ACK")
	require.NoError(t, err, "CancelSingleFuturesOrder must not error")
	if mockTests {
		exp := futuresFixtureOrder(getPair(t, asset.CoinMarginedFutures))
		assert.Equal(t, exp, result, "CancelSingleFuturesOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelSingleFuturesOrder should return the order")
}

func TestAmendFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := e.AmendFuturesOrder(t.Context(), currency.EMPTYCODE, "1234", &AmendFuturesOrderRequest{Price: 12345.990})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "AmendFuturesOrder must reject an empty settlement currency")

	_, err = e.AmendFuturesOrder(t.Context(), currency.BTC, "", &AmendFuturesOrderRequest{Price: 12345.990})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "AmendFuturesOrder must reject an empty order ID")

	_, err = e.AmendFuturesOrder(t.Context(), currency.BTC, "1234", nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AmendFuturesOrder must reject a nil amendment")

	_, err = e.AmendFuturesOrder(t.Context(), currency.BTC, "1234", &AmendFuturesOrderRequest{AmendText: "no change"})
	require.ErrorIs(t, err, errNoOrderAmendment, "AmendFuturesOrder must reject an amendment without a size or price")

	_, err = e.AmendFuturesOrder(t.Context(), currency.BTC, "1234", &AmendFuturesOrderRequest{Price: -1})
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "AmendFuturesOrder must reject a negative price")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AmendFuturesOrder(t.Context(), currency.BTC, "1234", &AmendFuturesOrderRequest{Price: 12345.990})
	require.NoError(t, err, "AmendFuturesOrder must not error")
	if mockTests {
		exp := futuresFixtureOrder(getPair(t, asset.CoinMarginedFutures))
		exp.Price = 12345.99
		assert.Equal(t, exp, result, "AmendFuturesOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "AmendFuturesOrder should return the order")
}

func TestGetMyFuturesTradingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetMyFuturesTradingHistory(t.Context(), currency.EMPTYCODE, "", "", getPair(t, asset.CoinMarginedFutures), 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMyFuturesTradingHistory must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMyFuturesTradingHistory(t.Context(), currency.BTC, "", "", getPair(t, asset.CoinMarginedFutures), 0, 0)
	require.NoError(t, err, "GetMyFuturesTradingHistory must not error")
	if mockTests {
		exp := []*TradingHistoryItemResponse{
			{
				ID:         121234231,
				CreateTime: types.Time(time.UnixMilli(1514764800123)),
				Contract:   "BTC_USD",
				Text:       "t-123456",
				Size:       100,
				Price:      100.123,
				OrderID:    "21893289839",
				CloseSize:  -50,
				Fee:        0.01,
				PointFee:   0.02,
				Role:       "taker",
				TradeValue: 28601.83,
			},
		}
		assert.Equal(t, exp, result, "GetMyFuturesTradingHistory should decode every field")
	}
}

func TestGetMyFuturesTradingHistoryByTimeRange(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	_, err := e.GetMyFuturesTradingHistoryByTimeRange(t.Context(), currency.EMPTYCODE, contract, "", 0, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMyFuturesTradingHistoryByTimeRange must reject an empty settlement currency")

	_, err = e.GetMyFuturesTradingHistoryByTimeRange(t.Context(), currency.USDT, contract, "liquidity", 0, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidTradeRole, "GetMyFuturesTradingHistoryByTimeRange must reject a role other than maker or taker")

	startTime, endTime := getTime()
	_, err = e.GetMyFuturesTradingHistoryByTimeRange(t.Context(), currency.USDT, contract, "", 0, 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMyFuturesTradingHistoryByTimeRange must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMyFuturesTradingHistoryByTimeRange(t.Context(), currency.USDT, contract, "taker", 10, 0, startTime, endTime)
	require.NoError(t, err, "GetMyFuturesTradingHistoryByTimeRange must not error")
	if mockTests {
		exp := []*FuturesTradeByTimeRangeResponse{
			{
				TradeID:    "121234231",
				CreateTime: types.Time(time.UnixMilli(1744110000123)),
				Contract:   "BTC_USDT",
				OrderID:    "21893289839",
				Size:       100,
				CloseSize:  -50,
				Price:      100.123,
				Role:       "taker",
				Text:       "t-123456",
				Fee:        0.01,
				PointFee:   0.02,
			},
		}
		assert.Equal(t, exp, result, "GetMyFuturesTradingHistoryByTimeRange should decode every field")
	}
}

func TestGetFuturesPositionCloseHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPositionCloseHistory(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures), 0, 0, time.Time{}, time.Time{}, "", "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesPositionCloseHistory must reject an empty settlement currency")

	_, err = e.GetFuturesPositionCloseHistory(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0, 0, time.Time{}, time.Time{}, "buy", "")
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "GetFuturesPositionCloseHistory must reject a side other than long or short")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesPositionCloseHistory(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0, 0, time.Time{}, time.Time{}, "long", "")
	require.NoError(t, err, "GetFuturesPositionCloseHistory must not error")
	if mockTests {
		exp := []*PositionCloseHistoryResponse{
			{
				Time:                  types.Time(time.UnixMilli(1546487347123)),
				ProfitAndLoss:         0.00013,
				ProfitAndLossPosition: 0.00011,
				ProfitAndLossFunding:  0.00001,
				ProfitAndLossFee:      0.00001,
				Side:                  "long",
				Contract:              "BTC_USD",
				Text:                  "web",
				MaxSize:               100,
				AccumulatedClosedSize: 100,
				FirstOpenTime:         types.Time(time.Unix(1546487300, 0)),
				LongAveragePrice:      2026.87,
				ShortAveragePrice:     2544.4,
			},
		}
		assert.Equal(t, exp, result, "GetFuturesPositionCloseHistory should decode every field")
	}
}

func TestGetFuturesLiquidationHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesLiquidationHistory(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures), 0, 0, time.Time{}, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesLiquidationHistory must reject an empty settlement currency")

	startTime, endTime := getTime()
	_, err = e.GetFuturesLiquidationHistory(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0, 0, endTime, startTime, time.Time{})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesLiquidationHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesLiquidationHistory(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0, 0, time.Time{}, time.Time{}, time.Time{})
	require.NoError(t, err, "GetFuturesLiquidationHistory must not error")
	if mockTests {
		exp := []*LiquidationHistoryItemResponse{
			{
				Time:             types.Time(time.Unix(1548654951, 0)),
				Contract:         currency.NewPairWithDelimiter("BTC", "USD", "_"),
				Size:             600,
				Leverage:         25,
				Margin:           0.006705256878,
				EntryPrice:       3536.123,
				MarkPrice:        3420.27,
				OrderPrice:       3405,
				FillPrice:        3424,
				LiquidationPrice: 3421.54,
				OrderID:          317393847,
			},
		}
		assert.Equal(t, exp, result, "GetFuturesLiquidationHistory should decode every field")
	}
}

func TestGetFuturesAutoDeleverages(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesAutoDeleverages(t.Context(), currency.EMPTYCODE, getPair(t, asset.USDTMarginedFutures), 0, 0, time.Time{}, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesAutoDeleverages must reject an empty settlement currency")

	startTime, endTime := getTime()
	_, err = e.GetFuturesAutoDeleverages(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 0, 0, endTime, startTime, time.Time{})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesAutoDeleverages must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesAutoDeleverages(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 10, 0, startTime, endTime, time.Time{})
	require.NoError(t, err, "GetFuturesAutoDeleverages must not error")
	if mockTests {
		exp := []*FuturesAutoDeleverageResponse{
			{
				Time:               types.Time(time.Unix(1675841679, 0)),
				User:               1666,
				OrderID:            73873128,
				Contract:           currency.NewPairWithDelimiter("BTC", "USDT", "_"),
				CrossLeverageLimit: 10,
				EntryPrice:         2649.648633636364,
				FillPrice:          2790.8082,
				TradeSize:          -10,
				PositionSize:       1,
			},
		}
		assert.Equal(t, exp, result, "GetFuturesAutoDeleverages should decode every field")
	}
}

func TestCountdownCancelFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CountdownCancelFuturesOrders(t.Context(), currency.EMPTYCODE, &CountdownRequest{Timeout: 8})
	require.ErrorIs(t, err, errEmptyOrInvalidSettlementCurrency, "CountdownCancelFuturesOrders must reject an empty settlement currency")

	_, err = e.CountdownCancelFuturesOrders(t.Context(), currency.BTC, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CountdownCancelFuturesOrders must reject a nil request")

	for _, timeout := range []int64{-1, 1, 4} {
		_, err = e.CountdownCancelFuturesOrders(t.Context(), currency.BTC, &CountdownRequest{Timeout: timeout})
		require.ErrorIsf(t, err, errInvalidTimeout, "CountdownCancelFuturesOrders must reject a timeout of %d seconds", timeout)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CountdownCancelFuturesOrders(t.Context(), currency.BTC, &CountdownRequest{Timeout: 8})
	require.NoError(t, err, "CountdownCancelFuturesOrders must not error")
	if mockTests {
		exp := &TriggerTimeResponse{
			TriggerTime: types.Time(time.Unix(1660039145, 0)),
		}
		assert.Equal(t, exp, result, "CountdownCancelFuturesOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "CountdownCancelFuturesOrders should return the trigger time")
}

func TestGetFuturesTradingFees(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesTradingFees(t.Context(), currency.EMPTYCODE, getPair(t, asset.USDTMarginedFutures))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesTradingFees must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesTradingFees(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures))
	require.NoError(t, err, "GetFuturesTradingFees must not error")
	if mockTests {
		exp := map[string]*FuturesTradingFeeResponse{
			"BTC_USDT": {
				TakerFee: 0.00075,
				MakerFee: -0.0001,
			},
		}
		assert.Equal(t, exp, result, "GetFuturesTradingFees should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetFuturesTradingFees should return fee rates")
}

func TestCancelBatchFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBatchFuturesOrders(t.Context(), currency.EMPTYCODE, []string{"1"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CancelBatchFuturesOrders must reject an empty settlement currency")

	_, err = e.CancelBatchFuturesOrders(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, errNoOrdersToCancel, "CancelBatchFuturesOrders must reject an empty order list")

	_, err = e.CancelBatchFuturesOrders(t.Context(), currency.USDT, make([]string, 21))
	require.ErrorIs(t, err, errTooManyOrders, "CancelBatchFuturesOrders must reject more than 20 orders")

	_, err = e.CancelBatchFuturesOrders(t.Context(), currency.USDT, []string{"1", ""})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelBatchFuturesOrders must reject an empty order ID")

	ex := e
	if mockTests {
		ex = futuresLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method, "batch cancellation should be requested with POST")
			assert.Equal(t, "/api/v4/futures/usdt/batch_cancel_orders", r.URL.Path, "batch cancellation should be requested from the batch_cancel_orders endpoint")
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err, "reading the request body should not error")
			assert.JSONEq(t, `["1","2"]`, string(body), "batch cancellation should send the order IDs as the body")
			_, err = w.Write([]byte(`[{"user_id":111,"id":"1","succeeded":true,"message":""},{"user_id":111,"id":"2","succeeded":false,"message":"ORDER_NOT_FOUND"}]`))
			assert.NoError(t, err, "writing the cancellation results should not error")
		})
	} else {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := ex.CancelBatchFuturesOrders(t.Context(), currency.USDT, []string{"1", "2"})
	require.NoError(t, err, "CancelBatchFuturesOrders must not error")
	if mockTests {
		exp := []*CancelFuturesOrderResultResponse{
			{
				UserID:    111,
				ID:        "1",
				Succeeded: true,
			},
			{
				UserID:  111,
				ID:      "2",
				Message: "ORDER_NOT_FOUND",
			},
		}
		assert.Equal(t, exp, result, "CancelBatchFuturesOrders should decode every field")
		return
	}
	assert.Len(t, result, 2, "CancelBatchFuturesOrders should report each order")
}

func TestAmendBatchFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.AmendBatchFuturesOrders(t.Context(), currency.EMPTYCODE, []*BatchAmendFuturesOrderRequest{{OrderID: 1, Price: 1}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "AmendBatchFuturesOrders must reject an empty settlement currency")

	_, err = e.AmendBatchFuturesOrders(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, errOrdersEmpty, "AmendBatchFuturesOrders must reject an empty batch")

	_, err = e.AmendBatchFuturesOrders(t.Context(), currency.USDT, make([]*BatchAmendFuturesOrderRequest, 11))
	require.ErrorIs(t, err, errTooManyOrders, "AmendBatchFuturesOrders must reject more than 10 orders")

	_, err = e.AmendBatchFuturesOrders(t.Context(), currency.USDT, []*BatchAmendFuturesOrderRequest{nil})
	require.ErrorIs(t, err, common.ErrNilPointer, "AmendBatchFuturesOrders must reject a nil amendment")

	_, err = e.AmendBatchFuturesOrders(t.Context(), currency.USDT, []*BatchAmendFuturesOrderRequest{{Price: 1}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "AmendBatchFuturesOrders must reject an amendment without an order ID or text")

	_, err = e.AmendBatchFuturesOrders(t.Context(), currency.USDT, []*BatchAmendFuturesOrderRequest{{OrderID: 1}})
	require.ErrorIs(t, err, errNoOrderAmendment, "AmendBatchFuturesOrders must reject an amendment without a size or price")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AmendBatchFuturesOrders(t.Context(), currency.USDT, []*BatchAmendFuturesOrderRequest{{OrderID: 121212, AmendText: "batch amend text", Size: 100, Price: 54321}})
	require.NoError(t, err, "AmendBatchFuturesOrders must not error")
	if mockTests {
		amendedOrder := futuresFixtureOrder(getPair(t, asset.USDTMarginedFutures))
		amendedOrder.Size, amendedOrder.Price, amendedOrder.AmendText = 100, 54321, "batch amend text"
		exp := []*BatchFuturesOrderResponse{{Succeeded: true, FuturesOrderResponse: *amendedOrder}}
		assert.Equal(t, exp, result, "AmendBatchFuturesOrders should decode every field")
		return
	}
	assert.Len(t, result, 1, "AmendBatchFuturesOrders should report the order")
}

func TestPlaceFuturesBBOOrder(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	_, err := e.PlaceFuturesBBOOrder(t.Context(), currency.EMPTYCODE, &FuturesBBOOrderRequest{Contract: contract, Size: 1, Direction: "sell", Level: 8})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PlaceFuturesBBOOrder must reject an empty settlement currency")

	_, err = e.PlaceFuturesBBOOrder(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceFuturesBBOOrder must reject a nil order")

	_, err = e.PlaceFuturesBBOOrder(t.Context(), currency.USDT, &FuturesBBOOrderRequest{Size: 1, Direction: "sell", Level: 8})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "PlaceFuturesBBOOrder must reject an empty contract")

	_, err = e.PlaceFuturesBBOOrder(t.Context(), currency.USDT, &FuturesBBOOrderRequest{Contract: contract, Direction: "sell", Level: 8})
	require.ErrorIs(t, err, errInvalidOrderSize, "PlaceFuturesBBOOrder must reject an empty size")

	_, err = e.PlaceFuturesBBOOrder(t.Context(), currency.USDT, &FuturesBBOOrderRequest{Contract: contract, Size: 1, Direction: "ask", Level: 8})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "PlaceFuturesBBOOrder must reject a direction other than buy or sell")

	for _, level := range []uint64{0, 21} {
		_, err = e.PlaceFuturesBBOOrder(t.Context(), currency.USDT, &FuturesBBOOrderRequest{Contract: contract, Size: 1, Direction: "sell", Level: level})
		require.ErrorIsf(t, err, errInvalidOrderBookLevel, "PlaceFuturesBBOOrder must reject order book level %d", level)
	}

	_, err = e.PlaceFuturesBBOOrder(t.Context(), currency.USDT, &FuturesBBOOrderRequest{Contract: contract, Size: 1, Direction: "sell", Level: 8, TimeInForce: "day"})
	require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "PlaceFuturesBBOOrder must reject an unknown time in force")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PlaceFuturesBBOOrder(t.Context(), currency.USDT, &FuturesBBOOrderRequest{Contract: contract, Size: 1, Direction: "sell", Level: 8})
	require.NoError(t, err, "PlaceFuturesBBOOrder must not error")
	if mockTests {
		exp := futuresFixtureOrder(getPair(t, asset.USDTMarginedFutures))
		exp.Size, exp.RemainingAmount, exp.TimeInForce = 1, 1, pocTIF
		assert.Equal(t, exp, result, "PlaceFuturesBBOOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "PlaceFuturesBBOOrder should return the order")
}

func TestCreateTrailOrder(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	_, err := e.CreateTrailOrder(t.Context(), currency.EMPTYCODE, &CreateTrailOrderRequest{Contract: contract, Amount: 10})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateTrailOrder must reject an empty settlement currency")

	_, err = e.CreateTrailOrder(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateTrailOrder must reject a nil order")

	_, err = e.CreateTrailOrder(t.Context(), currency.USDT, &CreateTrailOrderRequest{Amount: 10})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CreateTrailOrder must reject an empty contract")

	_, err = e.CreateTrailOrder(t.Context(), currency.USDT, &CreateTrailOrderRequest{Contract: contract})
	require.ErrorIs(t, err, errInvalidOrderSize, "CreateTrailOrder must reject an empty amount")

	_, err = e.CreateTrailOrder(t.Context(), currency.USDT, &CreateTrailOrderRequest{Contract: contract, Amount: 10, PriceType: 4})
	require.ErrorIs(t, err, errInvalidTriggerPriceType, "CreateTrailOrder must reject an unknown price type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateTrailOrder(t.Context(), currency.USDT, &CreateTrailOrderRequest{
		Contract:           contract,
		Amount:             10,
		ActivationPrice:    50000,
		IsGreaterThanEqual: true,
		PriceType:          1,
		PriceOffset:        "0.1%",
		Text:               "apiv4",
	})
	require.NoError(t, err, "CreateTrailOrder must not error")
	if mockTests {
		exp := &CreateTrailOrderResponse{
			ID: NumericString("63648"),
		}
		assert.Equal(t, exp, result, "CreateTrailOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CreateTrailOrder should return the order ID")
}

func TestStopTrailOrder(t *testing.T) {
	t.Parallel()
	_, err := e.StopTrailOrder(t.Context(), currency.EMPTYCODE, &StopTrailOrderRequest{ID: 63585})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "StopTrailOrder must reject an empty settlement currency")

	_, err = e.StopTrailOrder(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "StopTrailOrder must reject a nil request")

	_, err = e.StopTrailOrder(t.Context(), currency.USDT, &StopTrailOrderRequest{})
	require.ErrorIs(t, err, errTrailOrderIDOrTextRequired, "StopTrailOrder must reject a request without an ID or text")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.StopTrailOrder(t.Context(), currency.USDT, &StopTrailOrderRequest{ID: 63585})
	require.NoError(t, err, "StopTrailOrder must not error")
	if mockTests {
		exp := futuresFixtureTrailOrder()
		assert.Equal(t, exp, result, "StopTrailOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "StopTrailOrder should return the order")
}

func TestStopAllTrailOrders(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	_, err := e.StopAllTrailOrders(t.Context(), currency.EMPTYCODE, &StopAllTrailOrdersRequest{Contract: contract})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "StopAllTrailOrders must reject an empty settlement currency")

	_, err = e.StopAllTrailOrders(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "StopAllTrailOrders must reject a nil request")

	_, err = e.StopAllTrailOrders(t.Context(), currency.USDT, &StopAllTrailOrdersRequest{RelatedPosition: 3})
	require.ErrorIs(t, err, errInvalidTrailOrderFilter, "StopAllTrailOrders must reject an unknown related position")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.StopAllTrailOrders(t.Context(), currency.USDT, &StopAllTrailOrdersRequest{Contract: contract})
	require.NoError(t, err, "StopAllTrailOrders must not error")
	if mockTests {
		exp := []*TrailOrderResponse{futuresFixtureTrailOrder()}
		assert.Equal(t, exp, result, "StopAllTrailOrders should decode every field")
	}
}

func TestGetTrailOrders(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	_, err := e.GetTrailOrders(t.Context(), currency.EMPTYCODE, &GetTrailOrdersRequest{Contract: contract})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetTrailOrders must reject an empty settlement currency")

	_, err = e.GetTrailOrders(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTrailOrders must reject a nil request")

	for _, arg := range []*GetTrailOrdersRequest{{SortBy: 3}, {RelatedPosition: 3}, {ReduceOnly: 3}, {Side: 3}} {
		_, err = e.GetTrailOrders(t.Context(), currency.USDT, arg)
		require.ErrorIsf(t, err, errInvalidTrailOrderFilter, "GetTrailOrders must reject the filter %+v", arg)
	}

	startTime, endTime := getTime()
	_, err = e.GetTrailOrders(t.Context(), currency.USDT, &GetTrailOrdersRequest{StartAt: endTime, EndAt: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetTrailOrders must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTrailOrders(t.Context(), currency.USDT, &GetTrailOrdersRequest{Contract: contract, IsFinished: true, PageNumber: 1, PageSize: 10, SortBy: 1})
	require.NoError(t, err, "GetTrailOrders must not error")
	if mockTests {
		exp := []*TrailOrderResponse{futuresFixtureTrailOrder()}
		assert.Equal(t, exp, result, "GetTrailOrders should decode every field")
	}
}

func TestGetTrailOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetTrailOrderDetail(t.Context(), currency.EMPTYCODE, 63585)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetTrailOrderDetail must reject an empty settlement currency")

	_, err = e.GetTrailOrderDetail(t.Context(), currency.USDT, 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetTrailOrderDetail must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTrailOrderDetail(t.Context(), currency.USDT, 63585)
	require.NoError(t, err, "GetTrailOrderDetail must not error")
	if mockTests {
		exp := futuresFixtureTrailOrder()
		assert.Equal(t, exp, result, "GetTrailOrderDetail should decode every field")
		return
	}
	assert.NotNil(t, result, "GetTrailOrderDetail should return the order")
}

func TestUpdateTrailOrder(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTrailOrder(t.Context(), currency.EMPTYCODE, &UpdateTrailOrderRequest{ID: 63585, Amount: 20})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "UpdateTrailOrder must reject an empty settlement currency")

	_, err = e.UpdateTrailOrder(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UpdateTrailOrder must reject a nil request")

	_, err = e.UpdateTrailOrder(t.Context(), currency.USDT, &UpdateTrailOrderRequest{Amount: 20})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "UpdateTrailOrder must reject an empty order ID")

	_, err = e.UpdateTrailOrder(t.Context(), currency.USDT, &UpdateTrailOrderRequest{ID: 63585, IsGreaterThanEqual: "yes"})
	require.ErrorIs(t, err, errInvalidTrailOrderFilter, "UpdateTrailOrder must reject an is_gte_str other than true or false")

	_, err = e.UpdateTrailOrder(t.Context(), currency.USDT, &UpdateTrailOrderRequest{ID: 63585, PriceType: 4})
	require.ErrorIs(t, err, errInvalidTriggerPriceType, "UpdateTrailOrder must reject an unknown price type")

	_, err = newFixedResponseExchange(t, "/api/v4/futures/usdt/autoorder/v1/trail/update", `null`).UpdateTrailOrder(t.Context(), currency.USDT, &UpdateTrailOrderRequest{ID: 63585, Amount: 20})
	require.ErrorIs(t, err, common.ErrNoResponse, "UpdateTrailOrder must reject a response without the order")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.UpdateTrailOrder(t.Context(), currency.USDT, &UpdateTrailOrderRequest{ID: 63585, Amount: 20, ActivationPrice: "51000", PriceOffset: "0.2%"})
	require.NoError(t, err, "UpdateTrailOrder must not error")
	if mockTests {
		exp := futuresFixtureTrailOrder()
		assert.Equal(t, exp, result, "UpdateTrailOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "UpdateTrailOrder should return the order")
}

func TestGetTrailOrderChangeLog(t *testing.T) {
	t.Parallel()
	_, err := e.GetTrailOrderChangeLog(t.Context(), currency.EMPTYCODE, 63585, 1, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetTrailOrderChangeLog must reject an empty settlement currency")

	_, err = e.GetTrailOrderChangeLog(t.Context(), currency.USDT, 0, 1, 10)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetTrailOrderChangeLog must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTrailOrderChangeLog(t.Context(), currency.USDT, 63585, 1, 10)
	require.NoError(t, err, "GetTrailOrderChangeLog must not error")
	if mockTests {
		exp := []*TrailOrderChangeResponse{
			{
				UpdatedAt:          types.Time(time.Unix(1546569968, 0)),
				Amount:             10,
				IsGreaterThanEqual: true,
				ActivationPrice:    50000,
				PriceType:          1,
				PriceOffset:        "0.1%",
				IsCreate:           true,
			},
			{
				UpdatedAt:       types.Time(time.Unix(1546570000, 0)),
				Amount:          20,
				ActivationPrice: 51000,
				PriceType:       2,
				PriceOffset:     "0.2%",
			},
		}
		assert.Equal(t, exp, result, "GetTrailOrderChangeLog should decode every field")
	}
}

func TestTrailOrderReplyUnmarshalJSON(t *testing.T) {
	t.Parallel()
	exp := &TrailOrderResponse{ID: "63585", Contract: currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter), Amount: 10}
	for _, payload := range []string{
		`{"order":{"id":"63585","contract":"BTC_USDT","amount":"10"}}`,
		`{"id":63585,"contract":"BTC_USDT","amount":"10"}`,
	} {
		var reply trailOrderReply
		require.NoErrorf(t, json.Unmarshal([]byte(payload), &reply), "Unmarshal must not error for %s", payload)
		assert.Equalf(t, exp, reply.Order, "trailOrderReply should decode the order from %s", payload)
	}
}

func TestFuturesEmptyOrderReplies(t *testing.T) {
	t.Parallel()
	ex := futuresLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		reply := `{}`
		switch r.URL.Path {
		case "/api/v4/futures/btc/positions/BTC_USD/leverage", "/api/v4/futures/usdt/autoorder/v1/trail/stop":
			reply = `null`
		case "/api/v4/futures/usdt/autoorder/v1/trail/detail":
			reply = `{"code":0,"message":"ok","data":null,"timestamp":1769584936814}`
		}
		_, err := w.Write([]byte(reply))
		assert.NoError(t, err, "writing the empty reply should not error")
	})
	for name, call := range map[string]func() (any, error){
		"UpdateFuturesPositionLeverage": func() (any, error) {
			return ex.UpdateFuturesPositionLeverage(t.Context(), currency.BTC, currency.NewPairWithDelimiter("BTC", "USD", currency.UnderscoreDelimiter), 1, 0, 0)
		},
		"StopTrailOrder": func() (any, error) {
			return ex.StopTrailOrder(t.Context(), currency.USDT, &StopTrailOrderRequest{ID: 63585})
		},
		"GetTrailOrderDetail": func() (any, error) { return ex.GetTrailOrderDetail(t.Context(), currency.USDT, 63585) },
		"StopChaseOrder": func() (any, error) {
			return ex.StopChaseOrder(t.Context(), currency.USDT, &StopChaseOrderRequest{ID: "1001"})
		},
		"GetChaseOrderDetail": func() (any, error) { return ex.GetChaseOrderDetail(t.Context(), currency.USDT, "1001") },
	} {
		_, err := call()
		assert.ErrorIsf(t, err, common.ErrNoResponse, "%s should report an empty reply", name)
	}
}

func TestCreateChaseOrder(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	_, err := e.CreateChaseOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateChaseOrder must reject a nil order")

	_, err = e.CreateChaseOrder(t.Context(), &CreateChaseOrderRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateChaseOrder must reject an empty settlement currency")

	_, err = e.CreateChaseOrder(t.Context(), &CreateChaseOrderRequest{Settle: currency.USDT})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CreateChaseOrder must reject an empty contract")

	_, err = e.CreateChaseOrder(t.Context(), &CreateChaseOrderRequest{Settle: currency.USDT, Contract: contract})
	require.ErrorIs(t, err, errInvalidOrderSize, "CreateChaseOrder must reject an empty amount")

	_, err = e.CreateChaseOrder(t.Context(), &CreateChaseOrderRequest{Settle: currency.USDT, Contract: contract, Amount: 1, PriceLimit: 68000, OffsetLimit: 50})
	require.ErrorIs(t, err, errChasePriceAndOffsetLimit, "CreateChaseOrder must reject both a price limit and an offset limit")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateChaseOrder(t.Context(), &CreateChaseOrderRequest{Settle: currency.USDT, Contract: contract, Amount: 1, PriceLimit: 68000})
	require.NoError(t, err, "CreateChaseOrder must not error")
	if mockTests {
		exp := &CreateChaseOrderResponse{
			ID: NumericString("1001"),
		}
		assert.Equal(t, exp, result, "CreateChaseOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CreateChaseOrder should return the order ID")
}

func TestCreateChaseOrderRequestMarshalJSON(t *testing.T) {
	t.Parallel()
	payload, err := json.Marshal(&CreateChaseOrderRequest{
		Contract:    currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter),
		Amount:      -2,
		OffsetLimit: 50,
		PriceType:   2,
		Settle:      currency.USDT,
	})
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"contract":"BTC_USDT","amount":"-2","price_limit":"0","offset_limit":"50","price_type":2}`, string(payload), "CreateChaseOrderRequest should send a zero price limit when chasing by offset")
}

func TestStopChaseOrder(t *testing.T) {
	t.Parallel()
	_, err := e.StopChaseOrder(t.Context(), currency.EMPTYCODE, &StopChaseOrderRequest{ID: "1001"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "StopChaseOrder must reject an empty settlement currency")

	_, err = e.StopChaseOrder(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "StopChaseOrder must reject a nil request")

	_, err = e.StopChaseOrder(t.Context(), currency.USDT, &StopChaseOrderRequest{})
	require.ErrorIs(t, err, errChaseOrderIDOrTextRequired, "StopChaseOrder must reject a request without an ID or text")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.StopChaseOrder(t.Context(), currency.USDT, &StopChaseOrderRequest{ID: "1001"})
	require.NoError(t, err, "StopChaseOrder must not error")
	if mockTests {
		exp := futuresFixtureChaseOrder()
		assert.Equal(t, exp, result, "StopChaseOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "StopChaseOrder should return the order")
}

func TestStopAllChaseOrders(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	_, err := e.StopAllChaseOrders(t.Context(), currency.EMPTYCODE, &StopAllChaseOrdersRequest{Contract: contract})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "StopAllChaseOrders must reject an empty settlement currency")

	_, err = e.StopAllChaseOrders(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "StopAllChaseOrders must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.StopAllChaseOrders(t.Context(), currency.USDT, &StopAllChaseOrdersRequest{Contract: contract})
	require.NoError(t, err, "StopAllChaseOrders must not error")
	if mockTests {
		exp := []*ChaseOrderResponse{futuresFixtureChaseOrder()}
		assert.Equal(t, exp, result, "StopAllChaseOrders should decode every field")
	}
}

func TestStopAllChaseOrdersRequestMarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		arg StopAllChaseOrdersRequest
		exp string
	}{
		{arg: StopAllChaseOrdersRequest{}, exp: `{}`},
		{arg: StopAllChaseOrdersRequest{PositionMarginMode: "cross"}, exp: `{"pos_margin_mode":"cross"}`},
		{arg: StopAllChaseOrdersRequest{Contract: currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter), PositionMarginMode: "isolated"}, exp: `{"contract":"BTC_USDT","pos_margin_mode":"isolated"}`},
	} {
		payload, err := json.Marshal(&tc.arg)
		require.NoError(t, err, "Marshal must not error")
		assert.Equal(t, tc.exp, string(payload), "StopAllChaseOrdersRequest should omit an empty contract and margin mode")
	}
}

func TestGetChaseOrders(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	startTime, endTime := getTime()
	_, err := e.GetChaseOrders(t.Context(), currency.EMPTYCODE, &GetChaseOrdersRequest{SortBy: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetChaseOrders must reject an empty settlement currency")

	_, err = e.GetChaseOrders(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetChaseOrders must reject a nil request")

	_, err = e.GetChaseOrders(t.Context(), currency.USDT, &GetChaseOrdersRequest{})
	require.ErrorIs(t, err, errInvalidChaseSortBy, "GetChaseOrders must reject a missing sort field")

	_, err = e.GetChaseOrders(t.Context(), currency.USDT, &GetChaseOrdersRequest{SortBy: 1, IsFinished: true})
	require.ErrorIs(t, err, errChaseTimeRangeRequired, "GetChaseOrders must reject finished orders without a time range")

	_, err = e.GetChaseOrders(t.Context(), currency.USDT, &GetChaseOrdersRequest{SortBy: 1, StartAt: endTime, EndAt: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetChaseOrders must reject a start after the end")

	_, err = e.GetChaseOrders(t.Context(), currency.USDT, &GetChaseOrdersRequest{SortBy: 1, PageSize: 101})
	require.ErrorIs(t, err, errInvalidLimit, "GetChaseOrders must reject a page size above 100")

	for _, arg := range []*GetChaseOrdersRequest{{SortBy: 1, ReduceOnly: 3}, {SortBy: 1, Side: 3}} {
		_, err = e.GetChaseOrders(t.Context(), currency.USDT, arg)
		require.ErrorIsf(t, err, errInvalidChaseOrderFilter, "GetChaseOrders must reject the filter %+v", arg)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetChaseOrders(t.Context(), currency.USDT, &GetChaseOrdersRequest{Contract: contract, IsFinished: true, StartAt: startTime, EndAt: endTime, PageNumber: 1, PageSize: 100, SortBy: 1})
	require.NoError(t, err, "GetChaseOrders must not error")
	if mockTests {
		exp := []*ChaseOrderResponse{futuresFixtureChaseOrder()}
		assert.Equal(t, exp, result, "GetChaseOrders should decode every field")
	}
}

func TestGetChaseOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetChaseOrderDetail(t.Context(), currency.EMPTYCODE, "1001")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetChaseOrderDetail must reject an empty settlement currency")

	_, err = e.GetChaseOrderDetail(t.Context(), currency.USDT, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetChaseOrderDetail must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetChaseOrderDetail(t.Context(), currency.USDT, "1001")
	require.NoError(t, err, "GetChaseOrderDetail must not error")
	if mockTests {
		exp := futuresFixtureChaseOrder()
		assert.Equal(t, exp, result, "GetChaseOrderDetail should decode every field")
		return
	}
	assert.NotNil(t, result, "GetChaseOrderDetail should return the order")
}

func TestCreatePriceTriggeredFuturesOrder(t *testing.T) {
	t.Parallel()
	arg := &FuturesPriceTriggeredOrderRequest{
		Initial: FuturesInitial{Price: 1234, Size: 2, Contract: getPair(t, asset.USDTMarginedFutures)},
		Trigger: FuturesTrigger{Rule: 1, Price: 3000},
	}
	_, err := e.CreatePriceTriggeredFuturesOrder(t.Context(), currency.EMPTYCODE, arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreatePriceTriggeredFuturesOrder must reject an empty settlement currency")

	_, err = e.CreatePriceTriggeredFuturesOrder(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreatePriceTriggeredFuturesOrder must reject a nil order")

	for _, tc := range []struct {
		name        string
		mutate      func(*FuturesPriceTriggeredOrderRequest)
		expectedErr error
	}{
		{"empty contract", func(p *FuturesPriceTriggeredOrderRequest) { p.Initial.Contract = currency.EMPTYPAIR }, currency.ErrCurrencyPairEmpty},
		{"negative price", func(p *FuturesPriceTriggeredOrderRequest) { p.Initial.Price = -1 }, limits.ErrPriceBelowMin},
		{"fok time in force", func(p *FuturesPriceTriggeredOrderRequest) { p.Initial.TimeInForce = fokTIF }, order.ErrInvalidTimeInForce},
		{"unknown strategy type", func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.StrategyType = 2 }, errInvalidTriggerStrategyType},
		{"unknown price type", func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.PriceType = 3 }, errInvalidTriggerPriceType},
		{"missing trigger price", func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.Price = 0 }, limits.ErrPriceBelowMin},
		{"unknown rule", func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.Rule = 3 }, errInvalidTriggerRule},
		{"unknown order type", func(p *FuturesPriceTriggeredOrderRequest) { p.OrderType = "close-all" }, errInvalidAutoOrderType},
	} {
		invalid := *arg
		tc.mutate(&invalid)
		_, err = e.CreatePriceTriggeredFuturesOrder(t.Context(), currency.USDT, &invalid)
		require.ErrorIsf(t, err, tc.expectedErr, "CreatePriceTriggeredFuturesOrder must reject an order with a %s", tc.name)
	}

	ex := e
	if mockTests {
		ex = futuresLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method, "price triggered orders should be created with POST")
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err, "reading the request body should not error")
			switch r.URL.Path {
			case "/api/v4/futures/btc/price_orders":
				assert.JSONEq(t, `{"initial":{"contract":"BTC_USD","size":2,"price":"1234"},"trigger":{"price":"3000","rule":1},"order_type":"close-short-position"}`, string(body), "the BTC price triggered order should send the documented body")
			case "/api/v4/futures/usdt/price_orders":
				assert.JSONEq(t, `{"initial":{"contract":"BTC_USDT","size":0,"price":"0","tif":"ioc","reduce_only":true,"auto_size":"close_long"},"trigger":{"price_type":1,"price":"3000","rule":2,"expiration":86400},"order_type":"plan-close-long-position","pos_margin_mode":"isolated"}`, string(body), "a market close price triggered order should send a zero size and price")
			default:
				assert.Failf(t, "unexpected price triggered order path", "price triggered orders should only be created on the BTC and USDT paths, not %s", r.URL.Path)
			}
			_, err = w.Write([]byte(`{"id":1432329,"id_string":"1432329"}`))
			assert.NoError(t, err, "writing the order ID should not error")
		})
	} else {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err = ex.CreatePriceTriggeredFuturesOrder(t.Context(), currency.BTC, &FuturesPriceTriggeredOrderRequest{
		Initial:   FuturesInitial{Price: 1234, Size: 2, Contract: getPair(t, asset.CoinMarginedFutures)},
		Trigger:   FuturesTrigger{Rule: 1, Price: 3000},
		OrderType: "close-short-position",
	})
	require.NoError(t, err, "CreatePriceTriggeredFuturesOrder must not error for a BTC contract")

	result, err := ex.CreatePriceTriggeredFuturesOrder(t.Context(), currency.USDT, &FuturesPriceTriggeredOrderRequest{
		Initial:            FuturesInitial{Contract: getPair(t, asset.USDTMarginedFutures), TimeInForce: iocTIF, ReduceOnly: true, AutoSize: "close_long"},
		Trigger:            FuturesTrigger{PriceType: 1, Price: 3000, Rule: 2, Expiration: 86400},
		OrderType:          "plan-close-long-position",
		PositionMarginMode: "isolated",
	})
	require.NoError(t, err, "CreatePriceTriggeredFuturesOrder must not error for a market close")
	if mockTests {
		exp := &TriggerOrderIDResponse{
			ID:       1432329,
			IDString: "1432329",
		}
		assert.Equal(t, exp, result, "CreatePriceTriggeredFuturesOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CreatePriceTriggeredFuturesOrder should return the order ID")
}

func TestFuturesPriceTriggeredOrderParamValidate(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, (*FuturesPriceTriggeredOrderRequest)(nil).validate(), common.ErrNilPointer, "validate should reject a nil order")

	valid := FuturesPriceTriggeredOrderRequest{
		Initial: FuturesInitial{Price: 1234, Size: 2, Contract: currency.NewBTCUSDT()},
		Trigger: FuturesTrigger{Rule: 1, Price: 3000},
	}
	for _, tc := range []struct {
		name   string
		mutate func(*FuturesPriceTriggeredOrderRequest)
		err    error
	}{
		{"zero price for a market order", func(p *FuturesPriceTriggeredOrderRequest) { p.Initial.Price = 0 }, nil},
		{"gtc time in force", func(p *FuturesPriceTriggeredOrderRequest) { p.Initial.TimeInForce = gtcTIF }, nil},
		{"ioc time in force", func(p *FuturesPriceTriggeredOrderRequest) { p.Initial.TimeInForce = iocTIF }, nil},
		{"price gap strategy", func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.StrategyType = 1 }, nil},
		{"mark price trigger", func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.PriceType = 1 }, nil},
		{"index price trigger", func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.PriceType = 2 }, nil},
		{"at or below rule", func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.Rule = 2 }, nil},
		{"read only close long order type", func(p *FuturesPriceTriggeredOrderRequest) { p.OrderType = "close-long-order" }, errInvalidAutoOrderType},
		{"read only close short order type", func(p *FuturesPriceTriggeredOrderRequest) { p.OrderType = "close-short-order" }, errInvalidAutoOrderType},
		{"close long position type", func(p *FuturesPriceTriggeredOrderRequest) { p.OrderType = "close-long-position" }, nil},
		{"close short position type", func(p *FuturesPriceTriggeredOrderRequest) { p.OrderType = "close-short-position" }, nil},
		{"plan close long position type", func(p *FuturesPriceTriggeredOrderRequest) { p.OrderType = "plan-close-long-position" }, nil},
		{"plan close short position type", func(p *FuturesPriceTriggeredOrderRequest) { p.OrderType = "plan-close-short-position" }, nil},
		{"empty contract", func(p *FuturesPriceTriggeredOrderRequest) { p.Initial.Contract = currency.EMPTYPAIR }, currency.ErrCurrencyPairEmpty},
		{"negative price", func(p *FuturesPriceTriggeredOrderRequest) { p.Initial.Price = -1 }, limits.ErrPriceBelowMin},
		{"fok time in force", func(p *FuturesPriceTriggeredOrderRequest) { p.Initial.TimeInForce = fokTIF }, order.ErrInvalidTimeInForce},
		{"unknown strategy type", func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.StrategyType = 2 }, errInvalidTriggerStrategyType},
		{"unknown price type", func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.PriceType = 3 }, errInvalidTriggerPriceType},
		{"missing trigger price", func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.Price = 0 }, limits.ErrPriceBelowMin},
		{"unknown rule", func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.Rule = 3 }, errInvalidTriggerRule},
		{"unknown order type", func(p *FuturesPriceTriggeredOrderRequest) { p.OrderType = "close-all" }, errInvalidAutoOrderType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := valid
			tc.mutate(&p)
			assert.ErrorIs(t, p.validate(), tc.err, "validate should accept only documented values")
		})
	}
}

func TestListAllFuturesAutoOrders(t *testing.T) {
	t.Parallel()
	_, err := e.ListAllFuturesAutoOrders(t.Context(), "pending", currency.BTC, currency.EMPTYPAIR, 0, 0)
	require.ErrorIs(t, err, errInvalidOrderStatus, "ListAllFuturesAutoOrders must reject an unknown status")

	_, err = e.ListAllFuturesAutoOrders(t.Context(), statusOpen, currency.EMPTYCODE, currency.EMPTYPAIR, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "ListAllFuturesAutoOrders must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.ListAllFuturesAutoOrders(t.Context(), statusOpen, currency.BTC, currency.EMPTYPAIR, 0, 0)
	require.NoError(t, err, "ListAllFuturesAutoOrders must not error")
	if mockTests {
		exp := []*PriceTriggeredOrderResponse{futuresFixturePriceTriggeredOrder()}
		assert.Equal(t, exp, result, "ListAllFuturesAutoOrders should decode every field")
	}
}

func TestCancelAllFuturesOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllFuturesOpenOrders(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CancelAllFuturesOpenOrders must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err = e.CancelAllFuturesOpenOrders(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures))
	require.NoError(t, err, "CancelAllFuturesOpenOrders must not error for a contract")

	result, err := e.CancelAllFuturesOpenOrders(t.Context(), currency.USDT, currency.EMPTYPAIR)
	require.NoError(t, err, "CancelAllFuturesOpenOrders must not error for every contract")
	if mockTests {
		exp := []*PriceTriggeredOrderResponse{futuresFixturePriceTriggeredOrder()}
		assert.Equal(t, exp, result, "CancelAllFuturesOpenOrders should decode every field")
	}
}

func TestGetSingleFuturesPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleFuturesPriceTriggeredOrder(t.Context(), currency.EMPTYCODE, "12345")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetSingleFuturesPriceTriggeredOrder must reject an empty settlement currency")

	_, err = e.GetSingleFuturesPriceTriggeredOrder(t.Context(), currency.BTC, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetSingleFuturesPriceTriggeredOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSingleFuturesPriceTriggeredOrder(t.Context(), currency.BTC, "12345")
	require.NoError(t, err, "GetSingleFuturesPriceTriggeredOrder must not error")
	if mockTests {
		exp := futuresFixturePriceTriggeredOrder()
		assert.Equal(t, exp, result, "GetSingleFuturesPriceTriggeredOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSingleFuturesPriceTriggeredOrder should return the order")
}

func TestCancelFuturesPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelFuturesPriceTriggeredOrder(t.Context(), currency.EMPTYCODE, "12345")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CancelFuturesPriceTriggeredOrder must reject an empty settlement currency")

	_, err = e.CancelFuturesPriceTriggeredOrder(t.Context(), currency.BTC, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelFuturesPriceTriggeredOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelFuturesPriceTriggeredOrder(t.Context(), currency.USDT, "12345")
	require.NoError(t, err, "CancelFuturesPriceTriggeredOrder must not error")
	if mockTests {
		exp := futuresFixturePriceTriggeredOrder()
		assert.Equal(t, exp, result, "CancelFuturesPriceTriggeredOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelFuturesPriceTriggeredOrder should return the order")
}

func TestAmendFuturesPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	zeroSize, marketPrice := int64(0), 0.0
	_, err := e.AmendFuturesPriceTriggeredOrder(t.Context(), currency.EMPTYCODE, &AmendFuturesPriceTriggeredOrderRequest{OrderID: 1432329, TriggerPrice: 3100})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "AmendFuturesPriceTriggeredOrder must reject an empty settlement currency")

	_, err = e.AmendFuturesPriceTriggeredOrder(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AmendFuturesPriceTriggeredOrder must reject a nil amendment")

	_, err = e.AmendFuturesPriceTriggeredOrder(t.Context(), currency.USDT, &AmendFuturesPriceTriggeredOrderRequest{TriggerPrice: 3100})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "AmendFuturesPriceTriggeredOrder must reject an empty order ID")

	_, err = e.AmendFuturesPriceTriggeredOrder(t.Context(), currency.USDT, &AmendFuturesPriceTriggeredOrderRequest{OrderID: 1432329})
	require.ErrorIs(t, err, errNoOrderAmendment, "AmendFuturesPriceTriggeredOrder must reject an amendment that changes nothing")

	negativePrice, unknownPriceType := -1.0, int64(3)
	_, err = e.AmendFuturesPriceTriggeredOrder(t.Context(), currency.USDT, &AmendFuturesPriceTriggeredOrderRequest{OrderID: 1432329, Price: &negativePrice})
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "AmendFuturesPriceTriggeredOrder must reject a negative price")

	_, err = e.AmendFuturesPriceTriggeredOrder(t.Context(), currency.USDT, &AmendFuturesPriceTriggeredOrderRequest{OrderID: 1432329, PriceType: &unknownPriceType})
	require.ErrorIs(t, err, errInvalidTriggerPriceType, "AmendFuturesPriceTriggeredOrder must reject an unknown price type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AmendFuturesPriceTriggeredOrder(t.Context(), currency.USDT, &AmendFuturesPriceTriggeredOrderRequest{OrderID: 1432329, Size: &zeroSize, Price: &marketPrice, TriggerPrice: 3100})
	require.NoError(t, err, "AmendFuturesPriceTriggeredOrder must not error")
	if mockTests {
		exp := &TriggerOrderIDResponse{
			ID:       1432329,
			IDString: "1432329",
		}
		assert.Equal(t, exp, result, "AmendFuturesPriceTriggeredOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "AmendFuturesPriceTriggeredOrder should return the order ID")
}

func TestAmendFuturesPriceTriggeredOrderRequestMarshalJSON(t *testing.T) {
	t.Parallel()
	price, priceType := 3050.5, int64(0)
	for _, tc := range []struct {
		name string
		arg  AmendFuturesPriceTriggeredOrderRequest
		exp  string
	}{
		{name: "unset fields are left unchanged", arg: AmendFuturesPriceTriggeredOrderRequest{OrderID: 1, TriggerPrice: 3100}, exp: `{"order_id":1,"trigger_price":"3100"}`},
		{name: "set fields are sent even when zero", arg: AmendFuturesPriceTriggeredOrderRequest{OrderID: 1, Price: &price, PriceType: &priceType, Amount: 1.5, AutoSize: "close_short", Close: true}, exp: `{"order_id":1,"amount":"1.5","price":"3050.5","price_type":0,"auto_size":"close_short","close":true}`},
	} {
		payload, err := json.Marshal(&tc.arg)
		require.NoErrorf(t, err, "Marshal must not error when %s", tc.name)
		assert.Equalf(t, tc.exp, string(payload), "AmendFuturesPriceTriggeredOrderRequest should marshal as documented when %s", tc.name)
	}
}

func TestFuturesInitialMarshalJSON(t *testing.T) {
	t.Parallel()
	payload, err := json.Marshal(&FuturesInitial{Contract: currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter), Amount: 0.5, TimeInForce: iocTIF})
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"contract":"BTC_USDT","size":0,"amount":"0.5","price":"0","tif":"ioc"}`, string(payload), "FuturesInitial should send a zero size and price for a market order")
}

func TestGetAllDeliveryContracts(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllDeliveryContracts(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAllDeliveryContracts must reject an empty settlement currency")

	r, err := e.GetAllDeliveryContracts(t.Context(), currency.USDT)
	require.NoError(t, err, "GetAllDeliveryContracts must not error")
	if mockTests {
		require.Len(t, r, 6, "GetAllDeliveryContracts must return every recorded contract")
		exp := &DeliveryContractResponse{
			Name:                "BTC_USDT_20260626",
			Underlying:          "BTC_USDT",
			Cycle:               "BI-WEEKLY",
			Type:                "direct",
			QuantoMultiplier:    10,
			MarkType:            "index",
			LastPrice:           0.09043,
			MarkPrice:           0.0904,
			IndexPrice:          0.090395,
			BasisRate:           0.005653,
			BasisValue:          0.00001,
			BasisImpactValue:    1000,
			SettlePriceInterval: 60,
			SettlePriceDuration: 1800,
			SettleFeeRate:       0.00015,
			OrderPriceRound:     0.00001,
			MarkPriceRound:      0.00001,
			LeverageMin:         1,
			LeverageMax:         50,
			MaintenanceRate:     0.01,
			RiskLimitBase:       500000,
			RiskLimitStep:       500000,
			RiskLimitMax:        5000000,
			MakerFeeRate:        -0.00015,
			TakerFeeRate:        0.00025,
			RefRebateRate:       0.2,
			OrderPriceDeviate:   0.5,
			OrderSizeMin:        1,
			OrderSizeMax:        1000000,
			OrdersLimit:         50,
			OrderbookID:         2981897,
			TradeID:             74916,
			TradeSize:           530433,
			PositionSize:        106,
			ExpireTime:          types.Time(time.Unix(1775808000, 0)),
			ConfigChangeTime:    types.Time(time.Unix(1774597501, 0)),
		}
		assert.Equal(t, exp, r[0], "GetAllDeliveryContracts should decode every field")
	} else {
		assert.NotEmpty(t, r, "GetAllDeliveryContracts should return data")
	}
	r, err = e.GetAllDeliveryContracts(t.Context(), currency.BTC)
	require.NoError(t, err, "GetAllDeliveryContracts must not error")
	// The test below will fail if support for BTC settlement is added. This is intentional, as it ensures we are alerted when it's time to reintroduce support
	if !assert.Empty(t, r, "GetAllDeliveryContracts should not return any data with unsupported settlement currency BTC") {
		t.Error("BTC settlement for delivery futures appears to be supported again by the API. Please raise an issue to reintroduce BTC support for this exchange")
	}
}

func TestGetDeliveryContract(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryContract(t.Context(), currency.EMPTYCODE, getPair(t, asset.DeliveryFutures))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDeliveryContract must reject an empty settlement currency")

	_, err = e.GetDeliveryContract(t.Context(), currency.USDT, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetDeliveryContract must reject an empty contract")

	result, err := e.GetDeliveryContract(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures))
	require.NoError(t, err, "GetDeliveryContract must not error")
	if mockTests {
		exp := &DeliveryContractResponse{
			Name:                "BTC_USDT_20260626",
			Underlying:          "BTC_USDT",
			Cycle:               "BI-QUARTERLY",
			Type:                "direct",
			QuantoMultiplier:    0.0001,
			MarkType:            "index",
			LastPrice:           67857.5,
			MarkPrice:           67852.21,
			IndexPrice:          67503.92,
			BasisRate:           0.022966,
			BasisValue:          348.29,
			BasisImpactValue:    1000,
			SettlePriceInterval: 60,
			SettlePriceDuration: 1800,
			SettleFeeRate:       0.00015,
			OrderPriceRound:     0.1,
			MarkPriceRound:      0.01,
			LeverageMin:         1,
			LeverageMax:         100,
			MaintenanceRate:     0.005,
			RiskLimitBase:       1000000,
			RiskLimitStep:       1000000,
			RiskLimitMax:        8000000,
			MakerFeeRate:        -0.00015,
			TakerFeeRate:        0.00025,
			RefRebateRate:       0.2,
			OrderPriceDeviate:   0.5,
			OrderSizeMin:        1,
			OrderSizeMax:        1000000,
			OrdersLimit:         50,
			OrderbookID:         175093732,
			TradeID:             1017842,
			TradeSize:           6032055,
			PositionSize:        1257,
			ExpireTime:          types.Time(time.Unix(1782460800, 0)),
			ConfigChangeTime:    types.Time(time.Unix(1766735467, 0)),
		}
		assert.Equal(t, exp, result, "GetDeliveryContract should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDeliveryContract should return a contract")
}

func TestGetDeliveryOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryOrderbook(t.Context(), currency.EMPTYCODE, "0", getPair(t, asset.DeliveryFutures), 0, false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDeliveryOrderbook must reject an empty settlement currency")

	_, err = e.GetDeliveryOrderbook(t.Context(), currency.USDT, "0", currency.EMPTYPAIR, 0, false)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetDeliveryOrderbook must reject an empty contract")

	result, err := e.GetDeliveryOrderbook(t.Context(), currency.USDT, "0", getPair(t, asset.DeliveryFutures), 0, false)
	require.NoError(t, err, "GetDeliveryOrderbook must not error")
	if mockTests {
		exp := &OrderbookResponse{
			Current: types.Time(time.UnixMilli(1775423963401)),
			Update:  types.Time(time.UnixMilli(1775423963338)),
			Bids:    OrderbookLevels{{Price: 67853.9, Amount: 1}, {Price: 67848.6, Amount: 3}, {Price: 67848.1, Amount: 1007}, {Price: 67837.9, Amount: 2706}, {Price: 67828.2, Amount: 3}},
			Asks:    OrderbookLevels{{Price: 67859.1, Amount: 3}, {Price: 67868.6, Amount: 915}, {Price: 67876.1, Amount: 3}, {Price: 67878.8, Amount: 2714}, {Price: 67890, Amount: 3}},
		}
		assert.Equal(t, exp, result, "GetDeliveryOrderbook should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDeliveryOrderbook should return an orderbook")
}

func TestGetDeliveryTradingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryTradingHistory(t.Context(), currency.EMPTYCODE, "", getPair(t, asset.DeliveryFutures), 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDeliveryTradingHistory must reject an empty settlement currency")

	_, err = e.GetDeliveryTradingHistory(t.Context(), currency.USDT, "", currency.EMPTYPAIR, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetDeliveryTradingHistory must reject an empty contract")

	result, err := e.GetDeliveryTradingHistory(t.Context(), currency.USDT, "", getPair(t, asset.DeliveryFutures), 0, time.Time{}, time.Time{})
	require.NoError(t, err, "GetDeliveryTradingHistory must not error")
	if mockTests {
		require.Len(t, result, 5, "GetDeliveryTradingHistory must return every recorded trade")
		exp := &TradingHistoryItemResponse{
			ID:                     1017843,
			CreateTime:             types.Time(time.UnixMilli(1775423963713)),
			CreateTimeMilliseconds: types.Time(time.UnixMilli(1775423963713)),
			Contract:               "BTC_USDT_20260626",
			Size:                   -1,
			Price:                  67854,
		}
		assert.Equal(t, exp, result[0], "GetDeliveryTradingHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDeliveryTradingHistory should return a list")
}

func TestGetDeliveryFuturesCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryFuturesCandlesticks(t.Context(), currency.EMPTYCODE, getPair(t, asset.DeliveryFutures), time.Time{}, time.Time{}, 0, kline.OneWeek)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDeliveryFuturesCandlesticks must reject an empty settlement currency")

	_, err = e.GetDeliveryFuturesCandlesticks(t.Context(), currency.USDT, currency.EMPTYPAIR, time.Time{}, time.Time{}, 0, kline.OneWeek)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetDeliveryFuturesCandlesticks must reject an empty contract")

	startTime, endTime := getTime()
	_, err = e.GetDeliveryFuturesCandlesticks(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures), startTime, time.Time{}, 10, kline.OneWeek)
	require.ErrorIs(t, err, errLimitWithTimeRange, "GetDeliveryFuturesCandlesticks must reject a limit with a start time")

	_, err = e.GetDeliveryFuturesCandlesticks(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures), time.Time{}, endTime, 10, kline.OneWeek)
	require.ErrorIs(t, err, errLimitWithTimeRange, "GetDeliveryFuturesCandlesticks must reject a limit with an end time")

	_, err = e.GetDeliveryFuturesCandlesticks(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures), endTime, startTime, 0, kline.OneWeek)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetDeliveryFuturesCandlesticks must reject a start after the end")

	_, err = e.GetDeliveryFuturesCandlesticks(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures), startTime, endTime, 0, kline.Interval(5))
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "GetDeliveryFuturesCandlesticks must reject an unsupported interval")

	deliveryFuturesPair := getPair(t, asset.DeliveryFutures)
	if !mockTests {
		endTime = time.Now().UTC()
		startTime = endTime.Add(-time.Hour)
	}
	result, err := e.GetDeliveryFuturesCandlesticks(t.Context(), currency.USDT, deliveryFuturesPair, startTime, endTime, 0, kline.OneMin)
	require.NoError(t, err, "GetDeliveryFuturesCandlesticks must not error")
	if mockTests {
		require.Len(t, result, 5, "GetDeliveryFuturesCandlesticks must return every recorded candle")
		exp := &FuturesCandlestickResponse{
			Timestamp:    types.Time(time.Unix(1744103820, 0)),
			Volume:       42,
			ClosePrice:   79897.8,
			HighestPrice: 79915.1,
			LowestPrice:  79897.5,
			OpenPrice:    79915.1,
		}
		assert.Equal(t, exp, result[0], "GetDeliveryFuturesCandlesticks should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDeliveryFuturesCandlesticks should return a list")
}

func TestGetDeliveryFutureTickers(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryFutureTickers(t.Context(), currency.EMPTYCODE, getPair(t, asset.DeliveryFutures))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDeliveryFutureTickers must reject an empty settlement currency")

	results, err := e.GetDeliveryFutureTickers(t.Context(), currency.USDT, currency.EMPTYPAIR)
	require.NoError(t, err, "GetDeliveryFutureTickers must not error")
	if mockTests {
		require.Len(t, results, 5, "GetDeliveryFutureTickers must return every recorded ticker")
		exp := &FuturesTickerResponse{
			Contract:           currency.NewPairWithDelimiter("ARB", "USDT_20260410", currency.UnderscoreDelimiter),
			ChangePercentage:   -2.36,
			Last:               0.091,
			Low24Hour:          0.0893,
			High24Hour:         0.0933,
			TotalSize:          655,
			Volume24Hour:       103971,
			Volume24HourBase:   103971,
			Volume24HourQuote:  9461,
			Volume24HourSettle: 9461,
			MarkPrice:          0.091,
			IndexPrice:         0.0911,
			LowestAsk:          0.0911,
			HighestBid:         0.0909,
		}
		assert.Equal(t, exp, results[0], "GetDeliveryFutureTickers should decode every field")
		return
	}
	assert.NotNil(t, results, "GetDeliveryFutureTickers should return a list")
}

func TestGetDeliveryInsuranceBalanceHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryInsuranceBalanceHistory(t.Context(), currency.EMPTYCODE, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDeliveryInsuranceBalanceHistory must reject an empty settlement currency")

	result, err := e.GetDeliveryInsuranceBalanceHistory(t.Context(), currency.USDT, 2)
	require.NoError(t, err, "GetDeliveryInsuranceBalanceHistory must not error")
	if mockTests {
		exp := []*InsuranceBalanceResponse{
			{Timestamp: types.Time(time.Unix(1791849600, 0)), Balance: 103102.59316754},
			{Timestamp: types.Time(time.Unix(1791763200, 0)), Balance: 103102.41956754},
		}
		assert.Equal(t, exp, result, "GetDeliveryInsuranceBalanceHistory should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetDeliveryInsuranceBalanceHistory should return balances")
}

func TestGetDeliveryFuturesAccounts(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryFuturesAccounts(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDeliveryFuturesAccounts must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDeliveryFuturesAccounts(t.Context(), currency.USDT)
	require.NoError(t, err, "GetDeliveryFuturesAccounts must not error")
	if mockTests {
		exp := &FuturesAccountResponse{
			User:                   1666,
			Currency:               currency.USDT,
			Total:                  9707.803567115145,
			UnrealisedPNL:          3371.248828,
			PositionMargin:         38.712189181,
			OrderMargin:            0.5,
			Available:              9669.091377934145,
			Point:                  0.25,
			EnabledEvolvedClassic:  true,
			CrossInitialMargin:     61855.56788525,
			CrossUnrealisedPNL:     1501.178222634128,
			IsolatedPositionMargin: 38.712189181,
		}
		assert.Equal(t, exp, result, "GetDeliveryFuturesAccounts should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDeliveryFuturesAccounts should return an account")
}

func TestGetDeliveryAccountBooks(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetDeliveryAccountBooks(t.Context(), currency.EMPTYCODE, 0, startTime, endTime, "dnw")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDeliveryAccountBooks must reject an empty settlement currency")

	_, err = e.GetDeliveryAccountBooks(t.Context(), currency.USDT, 0, endTime, startTime, "dnw")
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetDeliveryAccountBooks must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDeliveryAccountBooks(t.Context(), currency.USDT, 0, startTime, endTime, "dnw")
	require.NoError(t, err, "GetDeliveryAccountBooks must not error")
	if mockTests {
		exp := []*AccountBookItemResponse{{
			Time:    types.Time(time.UnixMicro(1744110000123456)),
			Change:  10000,
			Balance: 10004.59316525194,
			Text:    "deposit",
			Type:    "dnw",
		}}
		assert.Equal(t, exp, result, "GetDeliveryAccountBooks should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDeliveryAccountBooks should return a list")
}

func TestGetAllDeliveryPositionsOfUser(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllDeliveryPositionsOfUser(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAllDeliveryPositionsOfUser must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAllDeliveryPositionsOfUser(t.Context(), currency.USDT)
	require.NoError(t, err, "GetAllDeliveryPositionsOfUser must not error")
	if mockTests {
		assert.Equal(t, []*PositionResponse{deliveryPositionFixture(getPair(t, asset.DeliveryFutures))}, result, "GetAllDeliveryPositionsOfUser should decode every position")
		return
	}
	assert.NotNil(t, result, "GetAllDeliveryPositionsOfUser should return a list")
}

func TestGetSingleDeliveryPosition(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleDeliveryPosition(t.Context(), currency.EMPTYCODE, getPair(t, asset.DeliveryFutures))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetSingleDeliveryPosition must reject an empty settlement currency")

	_, err = e.GetSingleDeliveryPosition(t.Context(), currency.USDT, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSingleDeliveryPosition must reject an empty contract")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	pair := getPair(t, asset.DeliveryFutures)
	result, err := e.GetSingleDeliveryPosition(t.Context(), currency.USDT, pair)
	require.NoError(t, err, "GetSingleDeliveryPosition must not error")
	if mockTests {
		assert.Equal(t, deliveryPositionFixture(pair), result, "GetSingleDeliveryPosition should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSingleDeliveryPosition should return a position")
}

func TestUpdateDeliveryPositionMargin(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateDeliveryPositionMargin(t.Context(), currency.EMPTYCODE, 0.001, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "UpdateDeliveryPositionMargin must reject an empty settlement currency")

	_, err = e.UpdateDeliveryPositionMargin(t.Context(), currency.USDT, 0.001, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateDeliveryPositionMargin must reject an empty contract")

	pair := getPair(t, asset.DeliveryFutures)
	_, err = e.UpdateDeliveryPositionMargin(t.Context(), currency.USDT, 0, pair)
	require.ErrorIs(t, err, errInvalidMarginChange, "UpdateDeliveryPositionMargin must reject a zero change")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	// A negative change removes margin, which Gate documents alongside a positive change adding it
	result, err := e.UpdateDeliveryPositionMargin(t.Context(), currency.USDT, -0.001, pair)
	require.NoError(t, err, "UpdateDeliveryPositionMargin must not error")
	if mockTests {
		exp := deliveryPositionFixture(pair)
		exp.Margin = 4.430548146258
		assert.Equal(t, exp, result, "UpdateDeliveryPositionMargin should decode every field")
		return
	}
	assert.NotNil(t, result, "UpdateDeliveryPositionMargin should return a position")
}

func TestUpdateDeliveryPositionLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateDeliveryPositionLeverage(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR, 0.001)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "UpdateDeliveryPositionLeverage must reject an empty settlement currency")

	_, err = e.UpdateDeliveryPositionLeverage(t.Context(), currency.USDT, currency.EMPTYPAIR, 0.001)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateDeliveryPositionLeverage must reject an empty contract")

	_, err = e.UpdateDeliveryPositionLeverage(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures), -1)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported, "UpdateDeliveryPositionLeverage must reject a negative leverage")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	pair := getPair(t, asset.DeliveryFutures)
	result, err := e.UpdateDeliveryPositionLeverage(t.Context(), currency.USDT, pair, 1)
	require.NoError(t, err, "UpdateDeliveryPositionLeverage must not error")
	if mockTests {
		exp := deliveryPositionFixture(pair)
		exp.Leverage = 1
		assert.Equal(t, exp, result, "UpdateDeliveryPositionLeverage should decode every field")
		return
	}
	assert.NotNil(t, result, "UpdateDeliveryPositionLeverage should return a position")
}

// deliveryPositionFixture returns the delivery position recorded in testdata/http.json
func deliveryPositionFixture(contract currency.Pair) *PositionResponse {
	return &PositionResponse{
		User:                       10000,
		Contract:                   contract,
		Size:                       -9440,
		RiskLimit:                  100,
		LeverageMax:                100,
		MaintenanceRate:            0.005,
		Value:                      3568.62,
		Margin:                     4.431548146258,
		EntryPrice:                 3779.55,
		LiquidationPrice:           99999999,
		MarkPrice:                  3780.32,
		InitialMargin:              35.7,
		MaintenanceMargin:          17.84,
		UnrealisedPNL:              -0.000507486844,
		RealisedPNL:                0.045497982432,
		RealisedPNLPosition:        0.045543982432,
		RealisedPNLFundingFee:      -0.000012,
		RealisedPNLTransactionFees: -0.000034,
		HistoryPNL:                 1.5,
		LastClosePNL:               0.5,
		ADLRanking:                 5,
		PendingOrders:              16,
		CloseOrder:                 CloseOrder{ID: 232323, Price: 3779},
		Mode:                       "single",
		CrossLeverageLimit:         10,
		UpdateTime:                 types.Time(time.Unix(1684994406, 0)),
		OpenTime:                   types.Time(time.Unix(1684990000, 0)),
		UpdateID:                   1,
		RiskLimitTable:             "BIG_HOT_COIN_50X_V2",
		AverageMaintenanceRate:     0.005,
	}
}

func TestGetAllOptionsUnderlyings(t *testing.T) {
	t.Parallel()
	underlyings, err := e.GetAllOptionsUnderlyings(t.Context())
	require.NoError(t, err, "GetAllOptionsUnderlyings must not error")
	if mockTests {
		require.Len(t, underlyings, 5, "GetAllOptionsUnderlyings must return every recorded underlying")
		exp := &OptionUnderlyingResponse{Name: "BTC_USDT", IndexPrice: 67504.01, IndexTime: types.Time(time.Unix(1775423976, 0))}
		assert.Equal(t, exp, underlyings[0], "GetAllOptionsUnderlyings should decode every field")
		return
	}
	assert.NotEmpty(t, underlyings, "GetAllOptionsUnderlyings should return underlyings")
}

func TestGetExpirationTimes(t *testing.T) {
	t.Parallel()
	_, err := e.GetExpirationTimes(t.Context(), "")
	require.ErrorIs(t, err, errInvalidUnderlying, "GetExpirationTimes must reject an empty underlying")

	result, err := e.GetExpirationTimes(t.Context(), getPair(t, asset.Spot).String())
	require.NoError(t, err, "GetExpirationTimes must not error")
	if mockTests {
		exp := make([]time.Time, 0, 10)
		for _, ts := range []int64{1775635200, 1775548800, 1775808000, 1798185600, 1777017600, 1775462400, 1780041600, 1782460800, 1776412800, 1790323200} {
			exp = append(exp, time.Unix(ts, 0))
		}
		assert.Equal(t, exp, result, "GetExpirationTimes should return every expiration in the order listed")
		return
	}
	assert.NotEmpty(t, result, "GetExpirationTimes should return expirations")
}

func TestGetAllContractOfUnderlyingWithinExpiryDate(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllContractOfUnderlyingWithinExpiryDate(t.Context(), "", time.Time{})
	require.ErrorIs(t, err, errInvalidUnderlying, "GetAllContractOfUnderlyingWithinExpiryDate must reject an empty underlying")

	result, err := e.GetAllContractOfUnderlyingWithinExpiryDate(t.Context(), "BTC_USDT", time.Time{})
	require.NoError(t, err, "GetAllContractOfUnderlyingWithinExpiryDate must not error")
	require.NotEmpty(t, result, "GetAllContractOfUnderlyingWithinExpiryDate must return contracts")
	exp := &OptionContractResponse{
		Name:              "BTC_USDT-20260408-73000-P",
		Tag:               "day",
		StrikePrice:       73000,
		MarkPrice:         5533.2,
		OrderbookID:       34336,
		Underlying:        "BTC_USDT",
		UnderlyingPrice:   67488.22,
		Multiplier:        0.01,
		OrderPriceRound:   1,
		MarkPriceRound:    0.1,
		MakerFeeRate:      0.0003,
		TakerFeeRate:      0.0003,
		PriceLimitFeeRate: 0.125,
		OrderPriceDeviate: 0.15,
		OrderSizeMin:      1,
		OrderSizeMax:      10000,
		OrdersLimit:       10,
		CreateTime:        types.Time(time.Unix(1775375281, 0)),
		ExpirationTime:    types.Time(time.Unix(1775635200, 0)),
	}
	if mockTests {
		require.Len(t, result, 6, "GetAllContractOfUnderlyingWithinExpiryDate must return every recorded contract")
		assert.Equal(t, exp, result[0], "GetAllContractOfUnderlyingWithinExpiryDate should decode every field")
	}

	expirations, err := e.GetExpirationTimes(t.Context(), "BTC_USDT")
	require.NoError(t, err, "GetExpirationTimes must not error")
	require.NotEmpty(t, expirations, "GetExpirationTimes must return expirations")
	result, err = e.GetAllContractOfUnderlyingWithinExpiryDate(t.Context(), "BTC_USDT", expirations[0])
	require.NoError(t, err, "GetAllContractOfUnderlyingWithinExpiryDate must not error for an expiration")
	if mockTests {
		exp.IndexPrice = 67503.71
		assert.Equal(t, []*OptionContractResponse{exp}, result, "GetAllContractOfUnderlyingWithinExpiryDate should decode every field")
		return
	}
	require.NotEmpty(t, result, "GetAllContractOfUnderlyingWithinExpiryDate must return contracts for an expiration")
	for _, contract := range result {
		assert.Truef(t, contract.ExpirationTime.Time().Equal(expirations[0]), "contract %s should expire at the requested expiration", contract.Name)
	}
}

func TestGetOptionsSpecifiedContractDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionsSpecifiedContractDetail(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOptionsSpecifiedContractDetail must reject an empty contract")

	result, err := e.GetOptionsSpecifiedContractDetail(t.Context(), getPair(t, asset.Options))
	require.NoError(t, err, "GetOptionsSpecifiedContractDetail must not error")
	if mockTests {
		exp := &OptionContractResponse{
			Name:              "BTC_USDT-20260925-75000-P",
			Tag:               "month",
			StrikePrice:       75000,
			LastPrice:         12988,
			MarkPrice:         12368.9,
			OrderbookID:       3087433,
			TradeID:           3,
			TradeSize:         3,
			PositionSize:      2,
			Underlying:        "BTC_USDT",
			UnderlyingPrice:   68398.81,
			Multiplier:        0.01,
			OrderPriceRound:   1,
			MarkPriceRound:    0.1,
			MakerFeeRate:      0.0003,
			TakerFeeRate:      0.0003,
			PriceLimitFeeRate: 0.125,
			OrderPriceDeviate: 0.15,
			OrderSizeMin:      1,
			OrderSizeMax:      10000,
			OrdersLimit:       10,
			CreateTime:        types.Time(time.Unix(1770450500, 0)),
			ExpirationTime:    types.Time(time.Unix(1790323200, 0)),
		}
		assert.Equal(t, exp, result, "GetOptionsSpecifiedContractDetail should decode every field")
		return
	}
	assert.NotNil(t, result, "GetOptionsSpecifiedContractDetail should return a contract")
}

func TestGetSettlementHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSettlementHistory(t.Context(), "", 0, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidUnderlying, "GetSettlementHistory must reject an empty underlying")

	startTime, endTime := getTime()
	_, err = e.GetSettlementHistory(t.Context(), "BTC_USDT", 0, 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSettlementHistory must reject a start after the end")

	result, err := e.GetSettlementHistory(t.Context(), "BTC_USDT", 0, 0, time.Time{}, time.Time{})
	require.NoError(t, err, "GetSettlementHistory must not error")
	if mockTests {
		require.Len(t, result, 5, "GetSettlementHistory must return every recorded settlement")
		exp := &OptionSettlementResponse{
			Timestamp:   types.Time(time.Unix(1775376000, 0)),
			SettlePrice: 66774.71479977753,
			Contract:    "BTC_USDT-20260405-60000-P",
			StrikePrice: 60000,
		}
		assert.Equal(t, exp, result[0], "GetSettlementHistory should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetSettlementHistory should return settlements")
}

func TestGetOptionsSpecifiedContractsSettlement(t *testing.T) {
	t.Parallel()
	contract := currency.NewPairWithDelimiter("BTC", "USDT-20260405-60000-P", currency.UnderscoreDelimiter)
	_, err := e.GetOptionsSpecifiedContractsSettlement(t.Context(), contract, "", time.Unix(1775376000, 0))
	require.ErrorIs(t, err, errInvalidUnderlying, "GetOptionsSpecifiedContractsSettlement must reject an empty underlying")

	_, err = e.GetOptionsSpecifiedContractsSettlement(t.Context(), currency.EMPTYPAIR, "BTC_USDT", time.Unix(1775376000, 0))
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOptionsSpecifiedContractsSettlement must reject an empty contract")

	_, err = e.GetOptionsSpecifiedContractsSettlement(t.Context(), contract, "BTC_USDT", time.Time{})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetOptionsSpecifiedContractsSettlement must reject an unset settlement time")

	underlying := "BTC_USDT"
	optionsSettlement, err := e.GetSettlementHistory(t.Context(), underlying, 0, 1, time.Time{}, time.Time{})
	require.NoError(t, err, "GetSettlementHistory must not error")
	require.NotEmpty(t, optionsSettlement, "GetSettlementHistory must return a settlement")

	cp, err := currency.NewPairFromString(optionsSettlement[0].Contract)
	require.NoError(t, err, "NewPairFromString must not error")

	result, err := e.GetOptionsSpecifiedContractsSettlement(t.Context(), cp, underlying, optionsSettlement[0].Timestamp.Time())
	require.NoError(t, err, "GetOptionsSpecifiedContractsSettlement must not error")
	if mockTests {
		exp := &OptionSettlementResponse{
			Timestamp:   types.Time(time.Unix(1774944000, 0)),
			Profit:      73.431430405781,
			SettlePrice: 67343.1430405781,
			Contract:    "BTC_USDT-20260405-60000-P",
			StrikePrice: 60000,
		}
		assert.Equal(t, exp, result, "GetOptionsSpecifiedContractsSettlement should decode every field")
		return
	}
	assert.Equal(t, optionsSettlement[0].Contract, result.Contract, "GetOptionsSpecifiedContractsSettlement should return the requested contract")
}

func TestGetSupportedFlashSwapCurrencyPairs(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		ccy         currency.Code
		limit, page uint64
		exp         *FlashSwapCurrencyPairResponse
	}{
		{
			ccy:   currency.BTC,
			limit: 10,
			exp: &FlashSwapCurrencyPairResponse{
				CurrencyPair:  getPair(t, asset.Spot),
				SellCurrency:  currency.BTC,
				BuyCurrency:   currency.USDT,
				SellMinAmount: 0.000001,
				SellMaxAmount: 250,
				BuyMinAmount:  1,
				BuyMaxAmount:  20000000,
			},
		},
		{
			limit: 10,
			page:  1,
			exp: &FlashSwapCurrencyPairResponse{
				CurrencyPair:  currency.NewPairWithDelimiter("PIRATE", "USDT", currency.UnderscoreDelimiter),
				SellCurrency:  currency.NewCode("PIRATE"),
				BuyCurrency:   currency.USDT,
				SellMinAmount: 1,
				SellMaxAmount: 842600,
				BuyMinAmount:  1,
				BuyMaxAmount:  1710,
			},
		},
	} {
		result, err := e.GetSupportedFlashSwapCurrencyPairs(t.Context(), tc.ccy, tc.limit, tc.page)
		require.NoErrorf(t, err, "GetSupportedFlashSwapCurrencyPairs must not error for currency %q", tc.ccy)
		if mockTests {
			require.Lenf(t, result, 5, "GetSupportedFlashSwapCurrencyPairs must return every recorded pair for currency %q", tc.ccy)
			assert.Equalf(t, tc.exp, result[0], "GetSupportedFlashSwapCurrencyPairs should decode every field for currency %q", tc.ccy)
			continue
		}
		assert.NotEmptyf(t, result, "GetSupportedFlashSwapCurrencyPairs should return pairs for currency %q", tc.ccy)
	}
}

func TestCreateFlashSwapOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CreateFlashSwapOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateFlashSwapOrder must reject a nil request")

	arg := &FlashSwapOrderRequest{}
	_, err = e.CreateFlashSwapOrder(t.Context(), arg)
	require.ErrorIs(t, err, errMissingPreviewID, "CreateFlashSwapOrder must reject an empty preview ID")

	arg.PreviewID = "1234"
	_, err = e.CreateFlashSwapOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateFlashSwapOrder must reject an empty buy currency")

	arg.BuyCurrency = currency.BTC
	_, err = e.CreateFlashSwapOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateFlashSwapOrder must reject an empty sell currency")

	arg.SellCurrency = currency.USDT
	_, err = e.CreateFlashSwapOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "CreateFlashSwapOrder must reject an empty sell amount")

	arg.SellAmount = 34234
	_, err = e.CreateFlashSwapOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "CreateFlashSwapOrder must reject an empty buy amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateFlashSwapOrder(t.Context(), &FlashSwapOrderRequest{
		PreviewID:    "1234",
		SellCurrency: currency.USDT,
		BuyCurrency:  currency.BTC,
		BuyAmount:    34234,
		SellAmount:   34234,
	})
	require.NoError(t, err, "CreateFlashSwapOrder must not error")
	if mockTests {
		exp := &FlashSwapOrderResponse{
			ID:           54646,
			CreateTime:   types.Time(time.UnixMilli(1744150000378)),
			UpdateTime:   types.Time(time.UnixMilli(1744150000378)),
			UserID:       11135567,
			SellCurrency: currency.USDT,
			SellAmount:   34234,
			BuyCurrency:  currency.BTC,
			BuyAmount:    34234,
			Price:        1,
			Status:       1,
		}
		assert.Equal(t, exp, result, "CreateFlashSwapOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CreateFlashSwapOrder should return a result")
}

func TestGetAllFlashSwapOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAllFlashSwapOrders(t.Context(), 1, currency.EMPTYCODE, currency.EMPTYCODE, true, 0, 0)
	require.NoError(t, err, "GetAllFlashSwapOrders must not error")
	if mockTests {
		exp := []*FlashSwapOrderResponse{{
			ID:           54646,
			CreateTime:   types.Time(time.UnixMilli(1744150000378)),
			UpdateTime:   types.Time(time.UnixMilli(1744150000378)),
			UserID:       11135567,
			SellCurrency: currency.BTC,
			SellAmount:   0.01,
			BuyCurrency:  currency.USDT,
			BuyAmount:    650,
			Price:        65000,
			Status:       1,
		}}
		assert.Equal(t, exp, result, "GetAllFlashSwapOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAllFlashSwapOrders should return a result")
}

func TestGetSingleFlashSwapOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleFlashSwapOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetSingleFlashSwapOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSingleFlashSwapOrder(t.Context(), "1234")
	require.NoError(t, err, "GetSingleFlashSwapOrder must not error")
	if mockTests {
		exp := &FlashSwapOrderResponse{
			ID:           1234,
			CreateTime:   types.Time(time.UnixMilli(1744150000378)),
			UpdateTime:   types.Time(time.UnixMilli(1744150000378)),
			UserID:       11135567,
			SellCurrency: currency.BTC,
			SellAmount:   0.01,
			BuyCurrency:  currency.USDT,
			BuyAmount:    650,
			Price:        65000,
			Status:       2,
		}
		assert.Equal(t, exp, result, "GetSingleFlashSwapOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSingleFlashSwapOrder should return a result")
}

func TestInitiateFlashSwapOrderReview(t *testing.T) {
	t.Parallel()
	_, err := e.InitiateFlashSwapOrderReview(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "InitiateFlashSwapOrderReview must reject a nil request")

	arg := &FlashSwapPreviewRequest{SellAmount: 100}
	_, err = e.InitiateFlashSwapOrderReview(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "InitiateFlashSwapOrderReview must reject an empty sell currency")

	arg.SellCurrency = currency.USDT
	_, err = e.InitiateFlashSwapOrderReview(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "InitiateFlashSwapOrderReview must reject an empty buy currency")

	arg.BuyCurrency = currency.BTC
	arg.SellAmount = 0
	_, err = e.InitiateFlashSwapOrderReview(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "InitiateFlashSwapOrderReview must reject a request without sell or buy amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.InitiateFlashSwapOrderReview(t.Context(), &FlashSwapPreviewRequest{
		SellCurrency: currency.USDT,
		BuyCurrency:  currency.BTC,
		SellAmount:   100,
	})
	require.NoError(t, err, "InitiateFlashSwapOrderReview must not error")
	if mockTests {
		exp := &InitFlashSwapOrderPreviewResponse{
			PreviewID:    "3453434",
			SellCurrency: currency.USDT,
			SellAmount:   100,
			BuyCurrency:  currency.BTC,
			BuyAmount:    0.0015,
			Price:        66666.66,
		}
		assert.Equal(t, exp, result, "InitiateFlashSwapOrderReview should decode every field")
		return
	}
	assert.NotNil(t, result, "InitiateFlashSwapOrderReview should return a result")
}

func TestGetDualInvestmentProductList(t *testing.T) {
	t.Parallel()
	_, err := e.GetDualInvestmentProductList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetDualInvestmentProductList must reject a nil request")

	putPlan := &DualInvestmentPlanResponse{
		ID:               123123412341,
		InstrumentName:   "BTC 81,000",
		Type:             "put",
		InvestCurrency:   currency.USDT,
		ExerciseCurrency: currency.BTC,
		ExercisePrice:    81000,
		DeliveryTime:     types.Time(time.Unix(1744272000, 0)),
		APYDisplay:       0.9401,
		MinAmount:        2,
		EndTime:          types.Time(time.Unix(1744268400, 0)),
		Status:           "ONGOING",
	}
	callPlan := *putPlan
	callPlan.InstrumentName = "BTC 81,500"
	callPlan.Type = "call"
	callPlan.InvestCurrency = currency.BTC
	callPlan.ExerciseCurrency = currency.USDT
	callPlan.ExercisePrice = 81500.5
	callPlan.APYDisplay = 8.3846
	callPlan.MinAmount = 0.00002435
	for _, tc := range []struct {
		request *DualInvestmentProductListRequest
		exp     *DualInvestmentPlanResponse
	}{
		{&DualInvestmentProductListRequest{Sort: "apy", Type: "put", QuoteCurrency: currency.USDT, PlanID: 123123412341, Page: 1, PageSize: 1000}, putPlan},
		{&DualInvestmentProductListRequest{Sort: "short-period", Type: "put", Coin: currency.BTC, QuoteCurrency: currency.USDT, PlanID: 123123412341, Page: 1, PageSize: 1000}, putPlan},
		{&DualInvestmentProductListRequest{Sort: "apy", Type: "call", QuoteCurrency: currency.USDT, PlanID: 123123412341, Page: 1, PageSize: 1000}, &callPlan},
	} {
		result, err := e.GetDualInvestmentProductList(t.Context(), tc.request)
		require.NoErrorf(t, err, "GetDualInvestmentProductList must not error for sort %s type %s", tc.request.Sort, tc.request.Type)
		if mockTests {
			assert.Equalf(t, []*DualInvestmentPlanResponse{tc.exp}, result, "GetDualInvestmentProductList should decode every field for sort %s type %s", tc.request.Sort, tc.request.Type)
			continue
		}
		assert.NotNilf(t, result, "GetDualInvestmentProductList should return a result for sort %s type %s", tc.request.Sort, tc.request.Type)
	}
}

func TestGetDualInvestmentOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.GetDualInvestmentOrderList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetDualInvestmentOrderList must reject a nil request")

	startTime, endTime := getTime()
	_, err = e.GetDualInvestmentOrderList(t.Context(), &DualInvestmentOrderListRequest{From: endTime, To: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetDualInvestmentOrderList must reject a reversed time range")

	t.Run("future settlement window", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.API.AuthenticatedSupport = true
		ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
		ex.SkipAuthCheck = true
		require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
		var query url.Values
		server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v4/earn/dual/orders", r.URL.Path, "request path should be the dual investment orders endpoint")
			query = r.URL.Query()
			_, err := fmt.Fprint(w, `[]`)
			assert.NoError(t, err, "writing dual investment orders should not error")
		}))
		require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
		require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")
		from, to := time.Now().Add(time.Hour*24), time.Now().Add(time.Hour*24*30)
		for _, tc := range []struct {
			name     string
			from, to time.Time
		}{
			{name: "future window", from: from, to: to},
			{name: "single settlement time", from: to, to: to},
		} {
			result, err := ex.GetDualInvestmentOrderList(t.Context(), &DualInvestmentOrderListRequest{From: tc.from, To: tc.to, Page: 2})
			require.NoErrorf(t, err, "GetDualInvestmentOrderList must accept a %s", tc.name)
			assert.Emptyf(t, result, "GetDualInvestmentOrderList should return the empty listing for a %s", tc.name)
			assert.Equalf(t, url.Values{"from": {strconv.FormatInt(tc.from.Unix(), 10)}, "to": {strconv.FormatInt(tc.to.Unix(), 10)}, "page": {"2"}}, query, "GetDualInvestmentOrderList should send the %s", tc.name)
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDualInvestmentOrderList(t.Context(), &DualInvestmentOrderListRequest{From: startTime, To: endTime, Type: "call", Status: "HOLD", Coin: currency.ETH, Limit: 100})
	require.NoError(t, err, "GetDualInvestmentOrderList must not error")
	if mockTests {
		exp := []*DualInvestmentOrderDetailResponse{{
			ID:               373,
			PlanID:           176,
			InvestAmount:     1.5,
			CreateTime:       types.Time(time.Unix(1744103900, 0)),
			CompleteTime:     types.Time(time.Unix(1744190000, 0)),
			Status:           "INIT",
			InvestCurrency:   currency.ETH,
			ExerciseCurrency: currency.USDT,
			ExercisePrice:    2500,
			DeliveryTime:     types.Time(time.Unix(1744272000, 0)),
			APYDisplay:       0.68,
			Text:             "t-custom-text",
		}}
		assert.Equal(t, exp, result, "GetDualInvestmentOrderList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDualInvestmentOrderList should return a result")
}

func TestPlaceDualInvestmentOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceDualInvestmentOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceDualInvestmentOrder must reject a nil request")

	_, err = e.PlaceDualInvestmentOrder(t.Context(), &DualInvestmentOrderRequest{Text: "t-abc"})
	require.ErrorIs(t, err, errPlanIDRequired, "PlaceDualInvestmentOrder must require a plan ID")

	_, err = e.PlaceDualInvestmentOrder(t.Context(), &DualInvestmentOrderRequest{PlanID: "12321"})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "PlaceDualInvestmentOrder must require an amount")

	_, err = e.PlaceDualInvestmentOrder(t.Context(), &DualInvestmentOrderRequest{PlanID: "12321", Amount: 1, Text: "abc"})
	require.ErrorIs(t, err, errInvalidTextPrefix, "PlaceDualInvestmentOrder must require custom text to start with t-")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PlaceDualInvestmentOrder(t.Context(), &DualInvestmentOrderRequest{PlanID: "12321", Amount: 1223.213})
	require.NoError(t, err, "PlaceDualInvestmentOrder must not error")
	if mockTests {
		exp := &DualInvestmentOrderDetailResponse{
			ID:               374,
			PlanID:           12321,
			InvestAmount:     1223.213,
			CreateTime:       types.Time(time.Unix(1744103900, 0)),
			Status:           "INIT",
			InvestCurrency:   currency.USDT,
			ExerciseCurrency: currency.BTC,
			ExercisePrice:    81000,
			DeliveryTime:     types.Time(time.Unix(1744272000, 0)),
			APYDisplay:       0.9401,
		}
		assert.Equal(t, exp, result, "PlaceDualInvestmentOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "PlaceDualInvestmentOrder should return the order")
}

func TestGetDualCurrencyEarningAssets(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDualCurrencyEarningAssets(t.Context())
	require.NoError(t, err, "GetDualCurrencyEarningAssets must not error")
	if mockTests {
		exp := &DualCurrencyEarningAssetResponse{
			UserAssetUSDT:         30.13,
			UserAssetBTC:          0.00032798,
			UserTotalInterestUSDT: 581.19003892,
			UserTotalInterestBTC:  0.00632655,
		}
		assert.Equal(t, exp, result, "GetDualCurrencyEarningAssets should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDualCurrencyEarningAssets should return a result")
}

func TestGetDualCurrencyEarlyRedemptionPreview(t *testing.T) {
	t.Parallel()
	_, err := e.GetDualCurrencyEarlyRedemptionPreview(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetDualCurrencyEarlyRedemptionPreview must require an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDualCurrencyEarlyRedemptionPreview(t.Context(), "12345678")
	require.NoError(t, err, "GetDualCurrencyEarlyRedemptionPreview must not error")
	if mockTests {
		exp := &DualCurrencyEarlyRedemptionResponse{
			CreateTime:          types.Time(time.Unix(1775027285, 0)),
			DeliveryTime:        types.Time(time.Unix(1775289600, 0)),
			ExercisePrice:       71303.44279,
			InvestAmount:        1,
			InvestCurrency:      currency.BTC,
			Name:                "BTC_USDT-20260404-71303.44279-C",
			OrderID:             12345678,
			RequestID:           "tIvdY7nh",
			SettlePrice:         68781.0063,
			SettlementAmount:    0.99486528,
			SettlementCurrency:  currency.BTC,
			SettlementPrincipal: 0.99486528,
			Type:                "call",
			MoneyBackTime:       types.Time(time.Unix(1775027958, 0)),
		}
		assert.Equal(t, exp, result, "GetDualCurrencyEarlyRedemptionPreview should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDualCurrencyEarlyRedemptionPreview should return a result")
}

func TestRedeemDualCurrencyOrder(t *testing.T) {
	t.Parallel()
	err := e.RedeemDualCurrencyOrder(t.Context(), "", "OepRSEfv")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "RedeemDualCurrencyOrder must require an order ID")

	err = e.RedeemDualCurrencyOrder(t.Context(), "9487", "")
	require.ErrorIs(t, err, errOrderRefundRequestIDRequired, "RedeemDualCurrencyOrder must require a request ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.RedeemDualCurrencyOrder(t.Context(), "9487", "OepRSEfv")
	assert.NoError(t, err, "RedeemDualCurrencyOrder should not error")
}

func TestModifyDualCurrencyOrderReinvest(t *testing.T) {
	t.Parallel()
	err := e.ModifyDualCurrencyOrderReinvest(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ModifyDualCurrencyOrderReinvest must reject a nil request")

	err = e.ModifyDualCurrencyOrderReinvest(t.Context(), &DualModifyOrderReinvestRequest{Status: 1})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "ModifyDualCurrencyOrderReinvest must require an order ID")

	err = e.ModifyDualCurrencyOrderReinvest(t.Context(), &DualModifyOrderReinvestRequest{OrderID: 9497, Status: 2})
	require.ErrorIs(t, err, errInvalidReinvestStatus, "ModifyDualCurrencyOrderReinvest must reject a status other than 0 or 1")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	// Status 0 switches reinvestment off, so it must reach the API rather than be dropped as empty
	for _, status := range []uint8{1, 0} {
		err = e.ModifyDualCurrencyOrderReinvest(t.Context(), &DualModifyOrderReinvestRequest{OrderID: 9497, Status: status})
		assert.NoErrorf(t, err, "ModifyDualCurrencyOrderReinvest should not error for status %d", status)
	}
}

func TestGetDualCurrencyRecommendedProjects(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	project := &DualRecommendedProjectResponse{
		ID:               110669,
		Category:         1,
		Type:             "call",
		InvestCurrency:   currency.BTC,
		ExerciseCurrency: currency.USDT,
		APYDisplay:       2.6,
		ExercisePrice:    71303.44279,
		DeliveryTime:     types.Time(time.Unix(1775289600, 0)),
		MinAmount:        1,
		MaxAmount:        100000,
		MinCopies:        1,
		MaxCopies:        100000,
		InvestDays:       2,
		InvestHours:      70.43458031,
	}
	result, err := e.GetDualCurrencyRecommendedProjects(t.Context(), "senior", currency.EMPTYCODE, "", nil)
	require.NoError(t, err, "GetDualCurrencyRecommendedProjects must not error")
	if mockTests {
		assert.Equal(t, []*DualRecommendedProjectResponse{project}, result, "GetDualCurrencyRecommendedProjects should decode every field")
	} else {
		assert.NotNil(t, result, "GetDualCurrencyRecommendedProjects should return a result")
	}

	result, err = e.GetDualCurrencyRecommendedProjects(t.Context(), "normal", currency.BTC, "call", []uint64{110669, 110670})
	require.NoError(t, err, "GetDualCurrencyRecommendedProjects must not error when excluding products")
	if mockTests {
		exp := *project
		exp.ID = 110671
		assert.Equal(t, []*DualRecommendedProjectResponse{&exp}, result, "GetDualCurrencyRecommendedProjects should decode every field when excluding products")
		return
	}
	assert.NotNil(t, result, "GetDualCurrencyRecommendedProjects should return a result when excluding products")
}

func TestCreateAutoInvestPlan(t *testing.T) {
	t.Parallel()
	_, err := e.CreateAutoInvestPlan(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateAutoInvestPlan must reject a nil request")

	arg := &CreateAutoInvestPlanRequest{}
	_, err = e.CreateAutoInvestPlan(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateAutoInvestPlan must require a pricing currency")

	arg.PlanMoney = currency.USDT
	_, err = e.CreateAutoInvestPlan(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "CreateAutoInvestPlan must require a plan amount")

	arg.PlanAmount = 10
	_, err = e.CreateAutoInvestPlan(t.Context(), arg)
	require.ErrorIs(t, err, errPeriodTypeRequired, "CreateAutoInvestPlan must require a period type")

	arg.PlanPeriodType = "monthly"
	_, err = e.CreateAutoInvestPlan(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidPlanPeriodDay, "CreateAutoInvestPlan must require a day of the month for monthly plans")

	arg.PlanPeriodType = "weekly"
	arg.PlanPeriodDay = 8
	_, err = e.CreateAutoInvestPlan(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidPlanPeriodDay, "CreateAutoInvestPlan must require a weekday for weekly plans")

	arg.PlanPeriodType = "daily"
	arg.PlanPeriodHour = 24
	_, err = e.CreateAutoInvestPlan(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidPlanPeriodHour, "CreateAutoInvestPlan must reject an hour beyond 23")

	arg.PlanPeriodHour = 0
	_, err = e.CreateAutoInvestPlan(t.Context(), arg)
	require.ErrorIs(t, err, errNoValidParameterPassed, "CreateAutoInvestPlan must require portfolio items")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateAutoInvestPlan(t.Context(), &CreateAutoInvestPlanRequest{
		PlanMoney:      currency.USDT,
		PlanAmount:     10,
		PlanPeriodType: "monthly",
		PlanPeriodDay:  1,
		Items:          []*AutoInvestPlanItem{{Asset: currency.BTC, Ratio: 100}},
	})
	require.NoError(t, err, "CreateAutoInvestPlan must not error")
	if mockTests {
		exp := &AutoInvestPlanResponse{
			ID:         142579,
			Amount:     10,
			Money:      currency.USDT,
			NextTime:   types.Time(time.Unix(1775005200, 0)),
			PeriodType: "monthly",
			PeriodDay:  1,
			FundFlow:   "auto_invest",
			FundSource: "spot",
		}
		assert.Equal(t, exp, result, "CreateAutoInvestPlan should decode every field")
		return
	}
	assert.NotNil(t, result, "CreateAutoInvestPlan should return the plan")
}

func TestValidateAutoInvestPlanItems(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		items []*AutoInvestPlanItem
		err   error
	}{
		{name: "no items", err: errNoValidParameterPassed},
		{name: "nil item", items: []*AutoInvestPlanItem{nil}, err: common.ErrNilPointer},
		{name: "empty asset", items: []*AutoInvestPlanItem{{Ratio: 100}}, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing ratio", items: []*AutoInvestPlanItem{{Asset: currency.BTC}}, err: order.ErrAmountMustBeSet},
		{name: "valid", items: []*AutoInvestPlanItem{{Asset: currency.BTC, Ratio: 50}, {Asset: currency.ETH, Ratio: 50}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateAutoInvestPlanItems(tc.items)
			if tc.err == nil {
				assert.NoError(t, err, "validateAutoInvestPlanItems should accept a complete portfolio")
				return
			}
			assert.ErrorIs(t, err, tc.err, "validateAutoInvestPlanItems should return the expected error")
		})
	}
}

func TestUpdateAutoInvestPlan(t *testing.T) {
	t.Parallel()
	err := e.UpdateAutoInvestPlan(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UpdateAutoInvestPlan must reject a nil request")

	err = e.UpdateAutoInvestPlan(t.Context(), &AutoInvestPlanUpdateRequest{FundSource: "earn"})
	require.ErrorIs(t, err, errPlanIDRequired, "UpdateAutoInvestPlan must require a plan ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.UpdateAutoInvestPlan(t.Context(), &AutoInvestPlanUpdateRequest{PlanID: 142582, FundSource: "earn", FundFlow: "auto_invest"})
	assert.NoError(t, err, "UpdateAutoInvestPlan should not error")
}

func TestStopAutoInvestPlan(t *testing.T) {
	t.Parallel()
	err := e.StopAutoInvestPlan(t.Context(), 0)
	require.ErrorIs(t, err, errPlanIDRequired, "StopAutoInvestPlan must require a plan ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.StopAutoInvestPlan(t.Context(), 142582)
	assert.NoError(t, err, "StopAutoInvestPlan should not error")
}

func TestAddAutoInvestPlanPosition(t *testing.T) {
	t.Parallel()
	err := e.AddAutoInvestPlanPosition(t.Context(), 0, 10)
	require.ErrorIs(t, err, errPlanIDRequired, "AddAutoInvestPlanPosition must require a plan ID")

	err = e.AddAutoInvestPlanPosition(t.Context(), 142583, 0)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "AddAutoInvestPlanPosition must require an amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.AddAutoInvestPlanPosition(t.Context(), 142583, 12.345)
	assert.NoError(t, err, "AddAutoInvestPlanPosition should not error")
}

func TestGetAutoInvestSupportedCoins(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAutoInvestSupportedCoins(t.Context(), currency.USDT)
	require.NoError(t, err, "GetAutoInvestSupportedCoins must not error")
	if mockTests {
		exp := []*AutoInvestCoinItemResponse{
			{Key: currency.BTC, Value: "BTC", AssetIconURL: "https://icon.staticimgs.com/images/coin_icon/64/btc.png?v=1743408000"},
			{Key: currency.ETH, Value: "ETH", AssetIconURL: "https://icon.staticimgs.com/images/coin_icon/64/eth.png?v=1743408000", Sort: 1},
		}
		assert.Equal(t, exp, result, "GetAutoInvestSupportedCoins should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAutoInvestSupportedCoins should return a result")
}

func TestGetAutoInvestMinimumAmount(t *testing.T) {
	t.Parallel()
	_, err := e.GetAutoInvestMinimumAmount(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAutoInvestMinimumAmount must reject a nil request")

	_, err = e.GetAutoInvestMinimumAmount(t.Context(), &AutoInvestMinAmountRequest{Items: []*AutoInvestPlanItem{{Asset: currency.BTC, Ratio: 33}}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAutoInvestMinimumAmount must require a pricing currency")

	_, err = e.GetAutoInvestMinimumAmount(t.Context(), &AutoInvestMinAmountRequest{Money: currency.USDT})
	require.ErrorIs(t, err, errNoValidParameterPassed, "GetAutoInvestMinimumAmount must require portfolio items")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAutoInvestMinimumAmount(t.Context(), &AutoInvestMinAmountRequest{
		Money: currency.USDT,
		Items: []*AutoInvestPlanItem{
			{Asset: currency.BTC, Ratio: 33},
			{Asset: currency.ETH, Ratio: 33},
			{Asset: currency.SOL, Ratio: 34},
		},
	})
	require.NoError(t, err, "GetAutoInvestMinimumAmount must not error")
	if mockTests {
		assert.Equal(t, &AutoInvestMinAmountResponse{MinAmount: 7.06}, result, "GetAutoInvestMinimumAmount should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAutoInvestMinimumAmount should return a result")
}

func TestGetAutoInvestPlanExecutionRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetAutoInvestPlanExecutionRecords(t.Context(), 0, 0, 0)
	require.ErrorIs(t, err, errPlanIDRequired, "GetAutoInvestPlanExecutionRecords must require a plan ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAutoInvestPlanExecutionRecords(t.Context(), 12345, 0, 0)
	require.NoError(t, err, "GetAutoInvestPlanExecutionRecords must not error")
	if mockTests {
		exp := &AutoInvestPlanExecutionRecordsResponse{
			Page:      1,
			PageSize:  10,
			TotalPage: 30,
			Total:     299,
			List: []*AutoInvestPlanExecutionRecord{{
				ID:            1770805384918111,
				Type:          "Auto Invest",
				Money:         currency.USDT,
				Asset:         currency.BTC,
				UserID:        21245786,
				PlanID:        12345,
				PlanVersion:   2,
				Amount:        32.60292,
				CreateTime:    types.Time(time.Unix(1773734410, 0)),
				UpdateTime:    types.Time(time.Unix(1773734410, 0)),
				Status:        "Failed",
				StatusType:    2,
				Side:          2,
				StatusMessage: "put order fail",
				Detail:        "insufficient balance",
			}},
		}
		assert.Equal(t, exp, result, "GetAutoInvestPlanExecutionRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAutoInvestPlanExecutionRecords should return a result")
}

func TestGetAutoInvestPlanOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetAutoInvestPlanOrderDetails(t.Context(), 0, 1)
	require.ErrorIs(t, err, errPlanIDRequired, "GetAutoInvestPlanOrderDetails must require a plan ID")

	_, err = e.GetAutoInvestPlanOrderDetails(t.Context(), 1, 0)
	require.ErrorIs(t, err, errRecordIDRequired, "GetAutoInvestPlanOrderDetails must require a record ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAutoInvestPlanOrderDetails(t.Context(), 12345, 67890)
	require.NoError(t, err, "GetAutoInvestPlanOrderDetails must not error")
	if mockTests {
		exp := []*AutoInvestOrderItemResponse{{
			ID:         6787627,
			Type:       "Buy",
			Amount:     0.031,
			PlanID:     12345,
			Side:       2,
			Asset:      currency.USDT,
			RecordID:   67890,
			TotalMoney: 2.72583,
			Market:     currency.NewPairWithDelimiter("SOL", "USDT", "/"),
			Price:      87.93,
			CreateTime: types.Time(time.Unix(1773656162, 0)),
			Total:      2.72583,
			FundFlow:   "auto_invest",
			Status:     1,
		}}
		assert.Equal(t, exp, result, "GetAutoInvestPlanOrderDetails should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAutoInvestPlanOrderDetails should return a result")
}

func TestGetAutoInvestCurrencyConfig(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAutoInvestCurrencyConfig(t.Context())
	require.NoError(t, err, "GetAutoInvestCurrencyConfig must not error")
	if mockTests {
		exp := []*AutoInvestConfigItemResponse{{Coin: currency.BTC, MaxLimit: 10}, {Coin: currency.USDT, MaxLimit: 100000}}
		assert.Equal(t, exp, result, "GetAutoInvestCurrencyConfig should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAutoInvestCurrencyConfig should return a result")
}

// autoInvestPlanDetailsFixture returns the auto invest plan recorded for the plan detail and plan list fixtures
func autoInvestPlanDetailsFixture() *AutoInvestPlanDetailsResponse {
	return &AutoInvestPlanDetailsResponse{
		ID:         12345,
		Version:    1,
		Name:       "Auto Invest BTC",
		CreateTime: types.Time(time.Unix(1773733133, 0)),
		UpdateTime: types.Time(time.Unix(1773733133, 0)),
		UserID:     2124569786,
		Money:      currency.USDT,
		Amount:     100,
		PeriodType: "monthly",
		PeriodDay:  1,
		PeriodHour: 1,
		Portfolio: []*AutoInvestPortfolioItem{
			{Asset: currency.BTC, Ratio: 66, CumulativeInvestment: 200, CumulativeHolding: 0.0025, AveragePrice: 80000},
			{Asset: currency.ETH, Ratio: 34, CumulativeInvestment: 100, CumulativeHolding: 0.04, AveragePrice: 2500},
		},
		NextTime:   types.Time(time.Unix(1775005200, 0)),
		Period:     3,
		FundSource: "spot",
		FundFlow:   "auto_invest",
	}
}

func TestGetAutoInvestPlanDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetAutoInvestPlanDetails(t.Context(), 0)
	require.ErrorIs(t, err, errPlanIDRequired, "GetAutoInvestPlanDetails must require a plan ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAutoInvestPlanDetails(t.Context(), 12345)
	require.NoError(t, err, "GetAutoInvestPlanDetails must not error")
	if mockTests {
		assert.Equal(t, autoInvestPlanDetailsFixture(), result, "GetAutoInvestPlanDetails should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAutoInvestPlanDetails should return a result")
}

func TestGetAutoInvestPlanList(t *testing.T) {
	t.Parallel()
	_, err := e.GetAutoInvestPlanList(t.Context(), "", 0, 0)
	require.ErrorIs(t, err, errPlanStatusRequired, "GetAutoInvestPlanList must require a status")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAutoInvestPlanList(t.Context(), "active", 1, 100)
	require.NoError(t, err, "GetAutoInvestPlanList must not error")
	if mockTests {
		exp := &AutoInvestPlanListResponse{
			Page:       1,
			PageSize:   100,
			PageCount:  1,
			TotalCount: 1,
			List:       []*AutoInvestPlanDetailsResponse{autoInvestPlanDetailsFixture()},
		}
		assert.Equal(t, exp, result, "GetAutoInvestPlanList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAutoInvestPlanList should return a result")
}

func TestGetFixedTermProducts(t *testing.T) {
	t.Parallel()
	_, err := e.GetFixedTermProducts(t.Context(), currency.EMPTYCODE, 0, 0, 10)
	require.ErrorIs(t, err, errPageRequired, "GetFixedTermProducts must require a page")

	_, err = e.GetFixedTermProducts(t.Context(), currency.EMPTYCODE, 0, 1, 0)
	require.ErrorIs(t, err, errInvalidLimit, "GetFixedTermProducts must require a limit")

	result, err := e.GetFixedTermProducts(t.Context(), currency.EMPTYCODE, 0, 1, 10)
	require.NoError(t, err, "GetFixedTermProducts must not error")
	if mockTests {
		product := &FixedTermProduct{
			ID:                11,
			Name:              "USDT-TEST",
			Asset:             currency.USDT,
			LockUpPeriodDays:  16,
			MinLendAmount:     1,
			UserMaxLendAmount: 100000,
			TotalLendAmount:   100000000,
			YearRate:          0.03,
			Status:            2,
			CreateTime:        time.Date(2024, 12, 13, 3, 29, 41, 103000000, time.UTC),
		}
		otherProduct := *product
		otherProduct.ID = 10
		otherProduct.Name = "\u793a\u4f8b"
		otherProduct.LockUpPeriodDays = 15
		assert.Equal(t, &FixedTermProductsResponse{List: []*FixedTermProduct{product, &otherProduct}, Total: 12}, result, "GetFixedTermProducts should decode every field")
	} else {
		require.NotNil(t, result, "GetFixedTermProducts must return a result")
		assert.NotEmpty(t, result.List, "GetFixedTermProducts should return products")
	}

	result, err = e.GetFixedTermProducts(t.Context(), currency.USDT, 2, 1, 10)
	require.NoError(t, err, "GetFixedTermProducts must not error for VIP products")
	if mockTests {
		exp := &FixedTermProductsResponse{
			List: []*FixedTermProduct{{
				ID:                33,
				Name:              "USDT - V5-7",
				Asset:             currency.USDT,
				LockUpPeriodDays:  7,
				MinLendAmount:     500,
				UserMaxLendAmount: 30000000,
				TotalLendAmount:   30000000,
				YearRate:          0.037,
				Type:              2,
				Reinvest:          true,
				RedeemAccount:     1,
				MinVIP:            5,
				MaxVIP:            16,
				Status:            2,
				CreateTime:        time.Date(2024, 12, 30, 2, 55, 35, 950000000, time.UTC),
				UserMaxLendVolume: 30000000,
				SaleStatus:        1,
			}},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetFixedTermProducts should decode every field for VIP products")
		return
	}
	assert.NotNil(t, result, "GetFixedTermProducts should return a result for VIP products")
}

func TestGetFixedTermProductsAPIError(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	// Fixed-term endpoints report failures inside a 200 response envelope
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v4/earn/fixed-term/product", r.URL.Path, "request path should be the fixed-term product list")
		_, err := fmt.Fprint(w, `{"code":-1,"message":"Service is busy. Please try again later.","data":null,"timestamp":1791854276}`)
		assert.NoError(t, err, "writing the fixed-term error envelope should not error")
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")

	_, err := ex.GetFixedTermProducts(t.Context(), currency.EMPTYCODE, 0, 1, 10)
	assert.ErrorContains(t, err, "Service is busy", "GetFixedTermProducts should surface the envelope error")
}

func TestGetFixedTermProductsByAsset(t *testing.T) {
	t.Parallel()
	_, err := e.GetFixedTermProductsByAsset(t.Context(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFixedTermProductsByAsset must require a currency")

	regular := &FixedTermProductSimple{
		ID:               68,
		Asset:            currency.USDT,
		LockUpPeriodDays: 1,
		YearRate:         0.0289,
		Type:             1,
		PreRedeem:        true,
		Reinvest:         true,
		SimpleEarn:       true,
		RedeemAccount:    1,
		SaleStatus:       1,
	}
	result, err := e.GetFixedTermProductsByAsset(t.Context(), currency.USDT, "")
	require.NoError(t, err, "GetFixedTermProductsByAsset must not error")
	if mockTests {
		assert.Equal(t, &FixedTermProductsByAssetResponse{List: []*FixedTermProductSimple{regular}}, result, "GetFixedTermProductsByAsset should decode every field")
	} else {
		assert.NotNil(t, result, "GetFixedTermProductsByAsset should return a result")
	}

	result, err = e.GetFixedTermProductsByAsset(t.Context(), currency.USDT, "0")
	require.NoError(t, err, "GetFixedTermProductsByAsset must not error for every product type")
	if mockTests {
		vip := &FixedTermProductSimple{
			ID:               284,
			Asset:            currency.USDT,
			LockUpPeriodDays: 7,
			YearRate:         0.037,
			Type:             2,
			Reinvest:         true,
			RedeemAccount:    1,
			MinVIP:           3,
			MaxVIP:           14,
			SaleStatus:       2,
		}
		assert.Equal(t, &FixedTermProductsByAssetResponse{List: []*FixedTermProductSimple{regular, vip}}, result, "GetFixedTermProductsByAsset should decode every field for every product type")
		return
	}
	assert.NotNil(t, result, "GetFixedTermProductsByAsset should return a result for every product type")
}

func TestSetFixedTermSubscriptionOrder(t *testing.T) {
	t.Parallel()
	_, err := e.SetFixedTermSubscriptionOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetFixedTermSubscriptionOrder must reject a nil request")

	_, err = e.SetFixedTermSubscriptionOrder(t.Context(), &FixedTermSubscriptionRequest{Amount: 1})
	require.ErrorIs(t, err, errProductIDRequired, "SetFixedTermSubscriptionOrder must require a product ID")

	_, err = e.SetFixedTermSubscriptionOrder(t.Context(), &FixedTermSubscriptionRequest{ProductID: 1})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "SetFixedTermSubscriptionOrder must require an amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetFixedTermSubscriptionOrder(t.Context(), &FixedTermSubscriptionRequest{ProductID: 1, Amount: 1})
	require.NoError(t, err, "SetFixedTermSubscriptionOrder must not error")
	if mockTests {
		assert.Equal(t, &OrderIDResponse{OrderID: 5846120275}, result, "SetFixedTermSubscriptionOrder should decode the order ID")
		return
	}
	assert.NotNil(t, result, "SetFixedTermSubscriptionOrder should return the order ID")
}

func TestGetFixedTermSubscriptionOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetFixedTermSubscriptionOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFixedTermSubscriptionOrders must reject a nil request")

	arg := &FixedTermSubscriptionOrdersRequest{ProductID: 11}
	_, err = e.GetFixedTermSubscriptionOrders(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "GetFixedTermSubscriptionOrders must require an order type")

	arg.OrderType = 1
	_, err = e.GetFixedTermSubscriptionOrders(t.Context(), arg)
	require.ErrorIs(t, err, errPageRequired, "GetFixedTermSubscriptionOrders must require a page")

	arg.Page = 1
	_, err = e.GetFixedTermSubscriptionOrders(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidLimit, "GetFixedTermSubscriptionOrders must require a limit")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	arg.Limit = 10
	result, err := e.GetFixedTermSubscriptionOrders(t.Context(), arg)
	require.NoError(t, err, "GetFixedTermSubscriptionOrders must not error")
	if mockTests {
		settlement := time.Date(2025, 8, 8, 0, 0, 0, 0, time.UTC)
		exp := &FixedTermSubscriptionOrdersResponse{
			List: []*FixedTermLendOrder{{
				ID:                   83402,
				Business:             1,
				OrderID:              5846120275,
				UserID:               2024103749,
				Asset:                currency.USDT,
				ProductID:            11,
				LockUpPeriodDays:     23,
				Principal:            44.12,
				YearRate:             0.3,
				ProductType:          2,
				Interest:             0.83404931,
				Status:               1,
				ReinvestStatus:       1,
				RedeemAccountType:    1,
				RedeemType:           2,
				RedeemTime:           settlement,
				FinishTime:           settlement,
				CreateTime:           time.Date(2025, 7, 15, 3, 55, 7, 304000000, time.UTC),
				YearRatePercent:      30,
				TotalYearRatePercent: 63.13,
				TotalInterest:        0.83996629,
				ProductInfo:          &FixedTermProductInfo{PreRedeem: true, Reinvest: true, RedeemAccount: 1, MinVIP: 1, MaxVIP: 16},
				BonusInfo: &FixedTermBonusInfo{
					ID:                   142,
					ProductID:            11,
					Asset:                currency.USDT,
					BonusAsset:           currency.GT,
					KYCLimit:             "1,2,3,4",
					LadderAPR:            []*FixedTermLadderAPR{{APR: 0.1, Right: 1}, {APR: 0.2, Left: 1, Right: 2}},
					TotalBonusAmount:     100000,
					UserTotalBonusAmount: 100000,
					Status:               2,
					StartTime:            time.Date(2025, 6, 20, 6, 57, 46, 0, time.UTC),
					EndTime:              time.Date(2026, 6, 29, 16, 0, 0, 0, time.UTC),
					CreateTime:           time.Date(2025, 6, 20, 6, 57, 54, 796000000, time.UTC),
					StartAt:              types.Time(time.Unix(1750402666, 0)),
					EndAt:                types.Time(time.Unix(1782748800, 0)),
					TotalIssuedAmount:    19.49,
					BonusAssetPrice:      17.66,
					ProductAssetPrice:    1,
					ProductYearRate:      0.3,
				},
				CouponInfo: &FixedTermCouponInfo{
					ID:              63,
					Business:        1,
					UserID:          2024103749,
					Asset:           currency.USDT,
					OrderID:         5846120275,
					FinancialRateID: 1506,
					BuyLimitLow:     1,
					BuyLimitHigh:    3,
					RateDays:        30,
					RateRatio:       3.13,
					CouponDays:      23,
					CouponPrincipal: 3,
					CouponYearRate:  0.0313,
					CouponInterest:  0.00591698,
					Status:          2,
					FinishTime:      settlement,
					CreateTime:      time.Date(2025, 7, 15, 3, 55, 7, 285000000, time.UTC),
				},
				RedeemAt: types.Time(time.Unix(1754611200, 0)),
				FinishAt: types.Time(time.Unix(1754611200, 0)),
				CreateAt: types.Time(time.Unix(1752551707, 0)),
				Icon:     "https://icon.gateimg.com/images/coin_icon/64/usdt.png",
			}},
			Total: 51,
		}
		assert.Equal(t, exp, result, "GetFixedTermSubscriptionOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFixedTermSubscriptionOrders should return a result")
}

func TestRedeemFixedTermOrder(t *testing.T) {
	t.Parallel()
	err := e.RedeemFixedTermOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "RedeemFixedTermOrder must require an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.RedeemFixedTermOrder(t.Context(), "5862476630")
	assert.NoError(t, err, "RedeemFixedTermOrder should not error")
}

func TestGetFixedTermSubscriptionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFixedTermSubscriptionHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFixedTermSubscriptionHistory must reject a nil request")

	arg := &FixedTermHistoryRequest{}
	_, err = e.GetFixedTermSubscriptionHistory(t.Context(), arg)
	require.ErrorIs(t, err, errHistoryTypeRequired, "GetFixedTermSubscriptionHistory must require a history type")

	arg.Type = 5
	_, err = e.GetFixedTermSubscriptionHistory(t.Context(), arg)
	require.ErrorIs(t, err, errHistoryTypeRequired, "GetFixedTermSubscriptionHistory must reject an unknown history type")

	arg.Type = 1
	_, err = e.GetFixedTermSubscriptionHistory(t.Context(), arg)
	require.ErrorIs(t, err, errPageRequired, "GetFixedTermSubscriptionHistory must require a page")

	arg.Page = 1
	_, err = e.GetFixedTermSubscriptionHistory(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidLimit, "GetFixedTermSubscriptionHistory must require a limit")

	startTime, endTime := getTime()
	_, err = e.GetFixedTermSubscriptionHistory(t.Context(), &FixedTermHistoryRequest{Type: 1, Page: 1, Limit: 10, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFixedTermSubscriptionHistory must reject a reversed time range")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	record := &FixedTermHistoryRecord{
		ID:               10001,
		OrderID:          5846120275,
		UserID:           2024103749,
		Asset:            currency.USDT,
		UniqueTime:       "2025-07-15",
		ProductID:        11,
		TotalPrincipal:   44.12,
		Amount:           44.12,
		AssetPrice:       1,
		Status:           1,
		Detail:           "subscribe",
		CreateTime:       time.Date(2025, 7, 15, 3, 55, 7, 304000000, time.UTC),
		CreateAt:         types.Time(time.Unix(1752551707, 0)),
		LockUpPeriodDays: 23,
	}
	arg.Limit = 10
	result, err := e.GetFixedTermSubscriptionHistory(t.Context(), arg)
	require.NoError(t, err, "GetFixedTermSubscriptionHistory must not error")
	if mockTests {
		assert.Equal(t, &FixedTermHistoryResponse{List: []*FixedTermHistoryRecord{record}, Total: 1}, result, "GetFixedTermSubscriptionHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetFixedTermSubscriptionHistory should return a result")
	}

	result, err = e.GetFixedTermSubscriptionHistory(t.Context(), &FixedTermHistoryRequest{
		Type:           3,
		Page:           1,
		Limit:          10,
		ProductID:      11,
		Asset:          currency.USDT,
		StartTime:      startTime,
		EndTime:        endTime,
		BusinessFilter: []FixedTermBusinessFilterRequest{{Business: 1}},
	})
	require.NoError(t, err, "GetFixedTermSubscriptionHistory must not error with filters")
	if mockTests {
		bonus := *record
		bonus.ID = 10002
		bonus.UniqueTime = "2025-04-09"
		bonus.BonusID = 142
		bonus.BonusAsset = currency.GT
		bonus.Amount = 0.00091
		bonus.AssetPrice = 17.66
		bonus.Detail = "bonus"
		bonus.CreateTime = time.Date(2025, 4, 9, 0, 0, 0, 0, time.UTC)
		bonus.CreateAt = types.Time(time.Unix(1744156800, 0))
		assert.Equal(t, &FixedTermHistoryResponse{List: []*FixedTermHistoryRecord{&bonus}, Total: 1}, result, "GetFixedTermSubscriptionHistory should decode every field with filters")
		return
	}
	assert.NotNil(t, result, "GetFixedTermSubscriptionHistory should return a result with filters")
}

func TestSetFixedTermBusinessFilter(t *testing.T) {
	t.Parallel()
	params := url.Values{}
	require.NoError(t, setFixedTermBusinessFilter(params, nil), "setFixedTermBusinessFilter must not error without filters")
	assert.Empty(t, params, "setFixedTermBusinessFilter should not set a parameter without filters")

	require.NoError(t, setFixedTermBusinessFilter(params, []FixedTermBusinessFilterRequest{{Business: 1}, {Business: 2, SubBusiness: 13}}), "setFixedTermBusinessFilter must not error")
	assert.Equal(t, `[{"business":1,"sub_business":0},{"business":2,"sub_business":13}]`, params.Get("business_filter"), "setFixedTermBusinessFilter should encode the filters as a JSON array")
}

func TestGetMyOptionsSettlements(t *testing.T) {
	t.Parallel()
	_, err := e.GetMyOptionsSettlements(t.Context(), "", currency.EMPTYPAIR, 0, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidUnderlying, "GetMyOptionsSettlements must reject an empty underlying")

	startTime, endTime := getTime()
	_, err = e.GetMyOptionsSettlements(t.Context(), "BTC_USDT", currency.EMPTYPAIR, 0, 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMyOptionsSettlements must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMyOptionsSettlements(t.Context(), "BTC_USDT", currency.EMPTYPAIR, 0, 0, time.Time{}, time.Time{})
	require.NoError(t, err, "GetMyOptionsSettlements must not error")
	if mockTests {
		exp := []*MyOptionSettlementResponse{{
			Size:        -1,
			Contract:    currency.NewPairWithDelimiter("BTC", "USDT-20220624-26000-C", currency.UnderscoreDelimiter),
			StrikePrice: 26000,
			Time:        types.Time(time.Unix(1656057600, 0)),
			SettlePrice: 20917.461281337048,
			Underlying:  "BTC_USDT",
			RealisedPNL: -0.00116042,
			Fee:         0.0001,
		}}
		assert.Equal(t, exp, result, "GetMyOptionsSettlements should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMyOptionsSettlements should return a list")
}

func TestGetOptionAccount(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOptionAccount(t.Context())
	require.NoError(t, err, "GetOptionAccount must not error")
	if mockTests {
		exp := &OptionAccountResponse{
			UserID:                       666,
			Total:                        1650.443022,
			PositionValue:                -40.1136,
			Equity:                       1610.329422,
			ShortEnabled:                 true,
			MarketMakerProtectionEnabled: true,
			MarginMode:                   1,
			UnrealisedPNL:                -0.7811,
			InitialMargin:                0.5,
			MaintenanceMargin:            135.541485,
			OrderMargin:                  139.74496,
			AskOrderMargin:               139.74496,
			BidOrderMargin:               0.25,
			Available:                    1514.901537,
			Point:                        0.1,
			Currency:                     currency.USDT,
			MaxOrderLimit:                10,
			PositionNotionalLimit:        1000000,
		}
		assert.Equal(t, exp, result, "GetOptionAccount should decode every field")
		return
	}
	assert.NotNil(t, result, "GetOptionAccount should return an account")
}

func TestGetAccountChangingHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetAccountChangingHistory(t.Context(), 0, 0, endTime, startTime, "")
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAccountChangingHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountChangingHistory(t.Context(), 1, 100, startTime, endTime, "prem")
	require.NoError(t, err, "GetAccountChangingHistory must not error")
	if mockTests {
		exp := []*AccountBookResponse{{
			ChangeTime:    types.Time(time.Unix(1744110000, 0)),
			AccountChange: -0.16,
			Balance:       7378.189,
			CustomText:    "BTC_USDT-20260925-75000-P:25",
			ChangingType:  "prem",
		}}
		assert.Equal(t, exp, result, "GetAccountChangingHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAccountChangingHistory should return a list")
}

func TestGetUsersPositionSpecifiedUnderlying(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUsersPositionSpecifiedUnderlying(t.Context(), "")
	require.NoError(t, err, "GetUsersPositionSpecifiedUnderlying must not error for every underlying")
	if mockTests {
		withoutCloseOrder := optionsPositionFixture(currency.NewPairWithDelimiter("BTC", "USDT-20260925-80000-C", currency.UnderscoreDelimiter))
		withoutCloseOrder.Size = -2
		withoutCloseOrder.PendingOrders = 0
		withoutCloseOrder.CloseOrder = CloseOrder{}
		exp := []*UsersPositionForUnderlyingResponse{optionsPositionFixture(getPair(t, asset.Options)), withoutCloseOrder}
		assert.Equal(t, exp, result, "GetUsersPositionSpecifiedUnderlying should decode every position, including one without a close order")
	}

	result, err = e.GetUsersPositionSpecifiedUnderlying(t.Context(), "BTC_USDT")
	require.NoError(t, err, "GetUsersPositionSpecifiedUnderlying must not error")
	if mockTests {
		assert.Equal(t, []*UsersPositionForUnderlyingResponse{optionsPositionFixture(getPair(t, asset.Options))}, result, "GetUsersPositionSpecifiedUnderlying should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUsersPositionSpecifiedUnderlying should return a list")
}

func TestGetSpecifiedContractPosition(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpecifiedContractPosition(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSpecifiedContractPosition must reject an empty contract")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpecifiedContractPosition(t.Context(), getPair(t, asset.Options))
	require.NoError(t, err, "GetSpecifiedContractPosition must not error")
	if mockTests {
		assert.Equal(t, optionsPositionFixture(getPair(t, asset.Options)), result, "GetSpecifiedContractPosition should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSpecifiedContractPosition should return a position")
}

// optionsPositionFixture returns the options position recorded in testdata/http.json
func optionsPositionFixture(contract currency.Pair) *UsersPositionForUnderlyingResponse {
	return &UsersPositionForUnderlyingResponse{
		User:                  11027586,
		Underlying:            "BTC_USDT",
		UnderlyingPrice:       70000,
		Contract:              contract,
		Size:                  10,
		EntryPrice:            1234,
		MarkPrice:             6000,
		MarkImpliedVolatility: 0.9638,
		RealisedPNL:           120,
		UnrealisedPNL:         -320,
		PendingOrders:         1,
		CloseOrder:            CloseOrder{ID: 232323, Price: 5779, IsLiquidation: true},
		Delta:                 -0.0046,
		Gamma:                 0.0001,
		Vega:                  2.87656,
		Theta:                 -1.00247,
	}
}

func TestGetUsersLiquidationHistoryForSpecifiedUnderlying(t *testing.T) {
	t.Parallel()
	_, err := e.GetUsersLiquidationHistoryForSpecifiedUnderlying(t.Context(), "", currency.EMPTYPAIR)
	require.ErrorIs(t, err, errInvalidUnderlying, "GetUsersLiquidationHistoryForSpecifiedUnderlying must reject an empty underlying")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUsersLiquidationHistoryForSpecifiedUnderlying(t.Context(), "BTC_USDT", currency.EMPTYPAIR)
	require.NoError(t, err, "GetUsersLiquidationHistoryForSpecifiedUnderlying must not error")
	if mockTests {
		exp := []*ContractClosePositionResponse{{
			PositionCloseTime: types.Time(time.Unix(1631764800, 0)),
			Contract:          "BTC_USDT-20210916-5000-C",
			Side:              "short",
			PNL:               -42914.291,
			CloseOrderText:    "settled",
			SettleSize:        -10001,
		}}
		assert.Equal(t, exp, result, "GetUsersLiquidationHistoryForSpecifiedUnderlying should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUsersLiquidationHistoryForSpecifiedUnderlying should return a list")
}

func TestPlaceOptionOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceOptionOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceOptionOrder must reject a nil request")

	arg := &OptionOrderRequest{
		OrderSize:   -1,
		Iceberg:     0,
		Text:        "-",
		TimeInForce: "gtc",
		Price:       100,
	}
	_, err = e.PlaceOptionOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "PlaceOptionOrder must reject an empty contract")

	pair := getPair(t, asset.Options)
	_, err = e.PlaceOptionOrder(t.Context(), &OptionOrderRequest{Contract: pair, Price: 100})
	require.ErrorIs(t, err, errInvalidOrderSize, "PlaceOptionOrder must reject a zero size without closing the position")

	_, err = e.PlaceOptionOrder(t.Context(), &OptionOrderRequest{Contract: pair, OrderSize: 1, Close: true})
	require.ErrorIs(t, err, errInvalidOrderSize, "PlaceOptionOrder must reject a non-zero size when closing the position")

	_, err = e.PlaceOptionOrder(t.Context(), &OptionOrderRequest{Contract: pair, OrderSize: 1, Price: -1})
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "PlaceOptionOrder must reject a negative price")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.Contract = pair
	result, err := e.PlaceOptionOrder(t.Context(), arg)
	require.NoError(t, err, "PlaceOptionOrder must not error")
	if !mockTests {
		assert.NotZero(t, result.OptionOrderID, "PlaceOptionOrder should return an order ID")
		return
	}
	assert.Equal(t, optionsOrderFixture(pair, 2), result, "PlaceOptionOrder should decode every field")

	// The recorded request body only matches a market order sending its zero price as "0"
	result, err = e.PlaceOptionOrder(t.Context(), &OptionOrderRequest{Contract: pair, OrderSize: 1, TimeInForce: iocTIF})
	require.NoError(t, err, "PlaceOptionOrder must not error for a market order")
	exp := &OptionOrderResponse{
		OptionOrderID: 3,
		User:          666,
		CreateTime:    types.Time(time.Unix(1631763361, 0)),
		FinishTime:    types.Time(time.Unix(1631763362, 0)),
		FinishAs:      "filled",
		Status:        statusFinished,
		Contract:      "BTC_USDT-20260925-75000-P",
		Size:          1,
		TimeInForce:   iocTIF,
		FillPrice:     12988,
		Text:          "api",
		TakerFeeRate:  0.0003,
		MakerFeeRate:  0.0003,
	}
	assert.Equal(t, exp, result, "PlaceOptionOrder should decode a filled market order")
}

func TestOptionOrderRequestMarshalJSON(t *testing.T) {
	t.Parallel()
	pair := currency.NewPairWithDelimiter("BTC", "USDT-20260925-75000-P", currency.UnderscoreDelimiter)
	for _, tc := range []struct {
		name    string
		request *OptionOrderRequest
		exp     string
	}{
		{
			name:    "market order",
			request: &OptionOrderRequest{Contract: pair, OrderSize: 1, TimeInForce: iocTIF},
			exp:     `{"contract":"BTC_USDT-20260925-75000-P","size":1,"price":"0","tif":"ioc"}`,
		},
		{
			name:    "close position",
			request: &OptionOrderRequest{Contract: pair, Close: true, TimeInForce: iocTIF},
			exp:     `{"contract":"BTC_USDT-20260925-75000-P","size":0,"price":"0","close":true,"tif":"ioc"}`,
		},
		{
			name: "limit order",
			request: &OptionOrderRequest{
				Contract:              pair,
				OrderSize:             -2,
				Iceberg:               1,
				Price:                 100.5,
				ReduceOnly:            true,
				MarketMakerProtection: true,
				TimeInForce:           pocTIF,
				Text:                  "t-my-custom-id",
			},
			exp: `{"contract":"BTC_USDT-20260925-75000-P","size":-2,"iceberg":1,"price":"100.5","reduce_only":true,"mmp":true,"tif":"poc","text":"t-my-custom-id"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tc.request)
			require.NoError(t, err, "Marshal must not error")
			assert.JSONEq(t, tc.exp, string(got), "Marshal should send the documented encoding")
		})
	}
}

func TestGetOptionFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionFuturesOrders(t.Context(), currency.EMPTYPAIR, "", "", 0, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidOrderStatus, "GetOptionFuturesOrders must reject an empty status")

	startTime, endTime := getTime()
	_, err = e.GetOptionFuturesOrders(t.Context(), currency.EMPTYPAIR, "", statusOpen, 0, 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetOptionFuturesOrders must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOptionFuturesOrders(t.Context(), currency.EMPTYPAIR, "", statusOpen, 0, 0, time.Time{}, time.Time{})
	require.NoError(t, err, "GetOptionFuturesOrders must not error")
	if mockTests {
		exp := optionsOrderFixture(getPair(t, asset.Options), 2)
		exp.IsMarketMakerProtection = true
		assert.Equal(t, []*OptionOrderResponse{exp}, result, "GetOptionFuturesOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetOptionFuturesOrders should return a list")
}

func TestCancelMultipleOptionOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMultipleOptionOpenOrders(t.Context(), getPair(t, asset.Options), "", "buy")
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "CancelMultipleOptionOpenOrders must reject a side other than ask or bid")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelMultipleOptionOpenOrders(t.Context(), getPair(t, asset.Options), "", "")
	require.NoError(t, err, "CancelMultipleOptionOpenOrders must not error")
	if mockTests {
		assert.Equal(t, []*OptionOrderResponse{optionsCancelledOrderFixture()}, result, "CancelMultipleOptionOpenOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelMultipleOptionOpenOrders should return a list")
}

func TestAmendOptionOrder(t *testing.T) {
	t.Parallel()
	pair := getPair(t, asset.Options)
	_, err := e.AmendOptionOrder(t.Context(), "", &AmendOptionOrderRequest{Contract: pair, Price: 1661, Size: 10})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "AmendOptionOrder must reject an empty order ID")

	_, err = e.AmendOptionOrder(t.Context(), "1234", nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AmendOptionOrder must reject a nil request")

	_, err = e.AmendOptionOrder(t.Context(), "1234", &AmendOptionOrderRequest{Price: 1661, Size: 10})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "AmendOptionOrder must reject an empty contract")

	_, err = e.AmendOptionOrder(t.Context(), "1234", &AmendOptionOrderRequest{Contract: pair, Size: 10})
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "AmendOptionOrder must reject a zero price")

	_, err = e.AmendOptionOrder(t.Context(), "1234", &AmendOptionOrderRequest{Contract: pair, Price: 1661})
	require.ErrorIs(t, err, errInvalidOrderSize, "AmendOptionOrder must reject a zero size")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AmendOptionOrder(t.Context(), "1234", &AmendOptionOrderRequest{Contract: pair, Price: 1661, Size: 10})
	require.NoError(t, err, "AmendOptionOrder must not error")
	if mockTests {
		exp := &OptionOrderResponse{
			OptionOrderID:   1234,
			User:            666,
			CreateTime:      types.Time(time.Unix(1631763361, 0)),
			Status:          statusOpen,
			Contract:        "BTC_USDT-20260925-75000-P",
			Size:            10,
			Price:           1661,
			TimeInForce:     gtcTIF,
			RemainingAmount: 10,
			Text:            "t-amended",
			TakerFeeRate:    0.0003,
			MakerFeeRate:    0.0003,
		}
		assert.Equal(t, exp, result, "AmendOptionOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "AmendOptionOrder should return an order")
}

func TestGetSingleOptionOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleOptionOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetSingleOptionOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSingleOptionOrder(t.Context(), "1234")
	require.NoError(t, err, "GetSingleOptionOrder must not error")
	if mockTests {
		assert.Equal(t, optionsOrderFixture(getPair(t, asset.Options), 1234), result, "GetSingleOptionOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSingleOptionOrder should return an order")
}

func TestCancelOptionSingleOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelOptionSingleOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelOptionSingleOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelOptionSingleOrder(t.Context(), "1234")
	require.NoError(t, err, "CancelOptionSingleOrder must not error")
	if mockTests {
		assert.Equal(t, optionsCancelledOrderFixture(), result, "CancelOptionSingleOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelOptionSingleOrder should return an order")
}

// optionsOrderFixture returns the open options order recorded in testdata/http.json
func optionsOrderFixture(contract currency.Pair, id uint64) *OptionOrderResponse {
	return &OptionOrderResponse{
		OptionOrderID:   id,
		User:            666,
		CreateTime:      types.Time(time.Unix(1631763361, 0)),
		Status:          statusOpen,
		Contract:        contract.String(),
		Size:            -1,
		Price:           100,
		TimeInForce:     gtcTIF,
		RemainingAmount: -1,
		Text:            "-",
		TakerFeeRate:    0.0003,
		MakerFeeRate:    0.0003,
	}
}

// optionsCancelledOrderFixture returns the cancelled options order recorded in testdata/http.json
func optionsCancelledOrderFixture() *OptionOrderResponse {
	return &OptionOrderResponse{
		OptionOrderID:   1234,
		User:            666,
		CreateTime:      types.Time(time.Unix(1631763361, 0)),
		FinishTime:      types.Time(time.Unix(1631763397, 0)),
		FinishAs:        "cancelled",
		Status:          statusFinished,
		Contract:        "BTC_USDT-20260925-75000-P",
		Size:            -1,
		Price:           100,
		TimeInForce:     gtcTIF,
		RemainingAmount: -1,
		Text:            "-",
		TakerFeeRate:    0.0003,
		MakerFeeRate:    0.0003,
		ReferenceUserID: 123,
		ReferrerRebate:  0.1,
	}
}

func TestCountdownCancelOptionOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CountdownCancelOptionOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CountdownCancelOptionOrders must reject a nil request")

	_, err = e.CountdownCancelOptionOrders(t.Context(), &CountdownCancelOptionOrdersRequest{Timeout: 4 * time.Second})
	require.ErrorIs(t, err, errInvalidTimeout, "CountdownCancelOptionOrders must reject a timeout under 5 seconds")

	_, err = e.CountdownCancelOptionOrders(t.Context(), &CountdownCancelOptionOrdersRequest{Timeout: -time.Second})
	require.ErrorIs(t, err, errInvalidTimeout, "CountdownCancelOptionOrders must reject a negative timeout")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CountdownCancelOptionOrders(t.Context(), &CountdownCancelOptionOrdersRequest{Timeout: 30 * time.Second, Underlying: "BTC_USDT"})
	require.NoError(t, err, "CountdownCancelOptionOrders must not error")
	if mockTests {
		assert.Equal(t, &TriggerTimeResponse{TriggerTime: types.Time(time.UnixMilli(1660039145000))}, result, "CountdownCancelOptionOrders should decode the trigger time")
	} else {
		assert.NotZero(t, result.TriggerTime, "CountdownCancelOptionOrders should return a trigger time")
	}

	result, err = e.CountdownCancelOptionOrders(t.Context(), &CountdownCancelOptionOrdersRequest{})
	require.NoError(t, err, "CountdownCancelOptionOrders must not error when clearing the countdown")
	assert.Zero(t, result.TriggerTime, "CountdownCancelOptionOrders should return no trigger time once cleared")
}

func TestCountdownCancelOptionOrdersRequestMarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		request *CountdownCancelOptionOrdersRequest
		exp     string
	}{
		{
			name: "scoped countdown",
			request: &CountdownCancelOptionOrdersRequest{
				Timeout:    30 * time.Second,
				Contract:   currency.NewPairWithDelimiter("BTC", "USDT-20260925-75000-P", currency.UnderscoreDelimiter),
				Underlying: "BTC_USDT",
			},
			exp: `{"timeout":30,"contract":"BTC_USDT-20260925-75000-P","underlying":"BTC_USDT"}`,
		},
		{
			name:    "fractional seconds",
			request: &CountdownCancelOptionOrdersRequest{Timeout: 5500 * time.Millisecond},
			exp:     `{"timeout":5}`,
		},
		{
			name:    "cleared countdown",
			request: &CountdownCancelOptionOrdersRequest{},
			exp:     `{"timeout":0}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tc.request)
			require.NoError(t, err, "Marshal must not error")
			assert.JSONEq(t, tc.exp, string(got), "Marshal should send the documented encoding")
		})
	}
}

func TestGetMyOptionsTradingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetMyOptionsTradingHistory(t.Context(), "", currency.EMPTYPAIR, 0, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidUnderlying, "GetMyOptionsTradingHistory must reject an empty underlying")

	startTime, endTime := getTime()
	_, err = e.GetMyOptionsTradingHistory(t.Context(), "BTC_USDT", currency.EMPTYPAIR, 0, 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMyOptionsTradingHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMyOptionsTradingHistory(t.Context(), "BTC_USDT", currency.EMPTYPAIR, 0, 0, time.Time{}, time.Time{})
	require.NoError(t, err, "GetMyOptionsTradingHistory must not error")
	if mockTests {
		exp := []*OptionTradingHistoryResponse{{
			ID:              1,
			UnderlyingPrice: 48000,
			Size:            1,
			Contract:        getPair(t, asset.Options),
			TradeRole:       "taker",
			CreateTime:      types.Time(time.Unix(1631763397, 0)),
			OrderID:         4,
			Price:           100,
		}}
		assert.Equal(t, exp, result, "GetMyOptionsTradingHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMyOptionsTradingHistory should return a list")
}

func TestSetOptionsMarketMakerProtection(t *testing.T) {
	t.Parallel()
	_, err := e.SetOptionsMarketMakerProtection(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetOptionsMarketMakerProtection must reject a nil request")

	newRequest := func() *OptionsMarketMakerProtectionRequest {
		return &OptionsMarketMakerProtectionRequest{
			Underlying:    "BTC_USDT",
			Window:        5 * time.Second,
			FrozenPeriod:  200 * time.Millisecond,
			QuantityLimit: 10,
			DeltaLimit:    10,
		}
	}
	for _, tc := range []struct {
		name   string
		modify func(*OptionsMarketMakerProtectionRequest)
		err    error
	}{
		{name: "underlying", modify: func(r *OptionsMarketMakerProtectionRequest) { r.Underlying = "" }, err: errInvalidUnderlying},
		{name: "negative window", modify: func(r *OptionsMarketMakerProtectionRequest) { r.Window = -time.Millisecond }, err: errInvalidMMPWindow},
		{name: "window", modify: func(r *OptionsMarketMakerProtectionRequest) { r.Window = 5*time.Second + time.Millisecond }, err: errInvalidMMPWindow},
		{name: "fractional window", modify: func(r *OptionsMarketMakerProtectionRequest) { r.Window = 500 * time.Microsecond }, err: errInvalidMMPWindow},
		{name: "negative frozen period", modify: func(r *OptionsMarketMakerProtectionRequest) { r.FrozenPeriod = -time.Millisecond }, err: errInvalidMMPFrozenPeriod},
		{name: "fractional frozen period", modify: func(r *OptionsMarketMakerProtectionRequest) { r.FrozenPeriod = 1500 * time.Microsecond }, err: errInvalidMMPFrozenPeriod},
		{name: "quantity limit", modify: func(r *OptionsMarketMakerProtectionRequest) { r.QuantityLimit = 0 }, err: errInvalidMMPLimit},
		{name: "delta limit", modify: func(r *OptionsMarketMakerProtectionRequest) { r.DeltaLimit = -1 }, err: errInvalidMMPLimit},
	} {
		arg := newRequest()
		tc.modify(arg)
		_, err = e.SetOptionsMarketMakerProtection(t.Context(), arg)
		require.ErrorIsf(t, err, tc.err, "SetOptionsMarketMakerProtection must reject an invalid %s", tc.name)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetOptionsMarketMakerProtection(t.Context(), newRequest())
	require.NoError(t, err, "SetOptionsMarketMakerProtection must not error")
	if mockTests {
		assert.Equal(t, optionsMarketMakerProtectionFixture(), result, "SetOptionsMarketMakerProtection should decode every field")
		return
	}
	assert.Equal(t, "BTC_USDT", result.Underlying, "SetOptionsMarketMakerProtection should return the underlying's settings")
}

func TestOptionsMarketMakerProtectionRequestMarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		request *OptionsMarketMakerProtectionRequest
		exp     string
	}{
		{
			name: "enabled",
			request: &OptionsMarketMakerProtectionRequest{
				Underlying:    "BTC_USDT",
				Window:        5 * time.Second,
				FrozenPeriod:  200 * time.Millisecond,
				QuantityLimit: 10,
				DeltaLimit:    2.5,
			},
			exp: `{"underlying":"BTC_USDT","window":5000,"frozen_period":200,"qty_limit":"10","delta_limit":"2.5"}`,
		},
		{
			name:    "disabled until reset",
			request: &OptionsMarketMakerProtectionRequest{Underlying: "BTC_USDT", QuantityLimit: 1, DeltaLimit: 1},
			exp:     `{"underlying":"BTC_USDT","window":0,"frozen_period":0,"qty_limit":"1","delta_limit":"1"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tc.request)
			require.NoError(t, err, "Marshal must not error")
			assert.JSONEq(t, tc.exp, string(got), "Marshal should send the documented encoding")
		})
	}
}

func TestOptionsMarketMakerProtectionResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var result OptionsMarketMakerProtectionResponse
	err := json.Unmarshal([]byte(`{"underlying":"BTC_USDT","window":5000,"frozen_period":200,"qty_limit":"10","delta_limit":"2.5","trigger_time_ms":1744110000123,"frozen_until_ms":1744110000323}`), &result)
	require.NoError(t, err, "Unmarshal must not error")
	exp := OptionsMarketMakerProtectionResponse{
		Underlying:    "BTC_USDT",
		Window:        5 * time.Second,
		FrozenPeriod:  200 * time.Millisecond,
		QuantityLimit: 10,
		DeltaLimit:    2.5,
		TriggerTime:   types.Time(time.UnixMilli(1744110000123)),
		FrozenUntil:   types.Time(time.UnixMilli(1744110000323)),
	}
	assert.Equal(t, exp, result, "Unmarshal should decode the window and frozen period from milliseconds")

	err = json.Unmarshal([]byte(`{"window":"5000"}`), &result)
	assert.Error(t, err, "Unmarshal should reject a window that is not a JSON number")
}

func TestGetOptionsMarketMakerProtection(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOptionsMarketMakerProtection(t.Context(), "BTC_USDT")
	require.NoError(t, err, "GetOptionsMarketMakerProtection must not error")
	if mockTests {
		exp := optionsMarketMakerProtectionFixture()
		exp.TriggerTime = types.Time(time.UnixMilli(1744110000123))
		exp.FrozenUntil = types.Time(time.UnixMilli(1744110000323))
		assert.Equal(t, []*OptionsMarketMakerProtectionResponse{exp}, result, "GetOptionsMarketMakerProtection should decode every field")
		return
	}
	assert.NotNil(t, result, "GetOptionsMarketMakerProtection should return a list")
}

func TestResetOptionsMarketMakerProtection(t *testing.T) {
	t.Parallel()
	_, err := e.ResetOptionsMarketMakerProtection(t.Context(), "")
	require.ErrorIs(t, err, errInvalidUnderlying, "ResetOptionsMarketMakerProtection must reject an empty underlying")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ResetOptionsMarketMakerProtection(t.Context(), "BTC_USDT")
	require.NoError(t, err, "ResetOptionsMarketMakerProtection must not error")
	if mockTests {
		assert.Equal(t, optionsMarketMakerProtectionFixture(), result, "ResetOptionsMarketMakerProtection should decode every field")
		return
	}
	assert.Equal(t, "BTC_USDT", result.Underlying, "ResetOptionsMarketMakerProtection should return the underlying's settings")
}

// optionsMarketMakerProtectionFixture returns the untriggered market maker protection recorded in testdata/http.json
func optionsMarketMakerProtectionFixture() *OptionsMarketMakerProtectionResponse {
	return &OptionsMarketMakerProtectionResponse{
		Underlying:    "BTC_USDT",
		Window:        5 * time.Second,
		FrozenPeriod:  200 * time.Millisecond,
		QuantityLimit: 10,
		DeltaLimit:    10,
	}
}

func TestWithdrawCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawCurrency(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WithdrawCurrency must reject a nil request")

	_, err = e.WithdrawCurrency(t.Context(), &WithdrawalRequest{})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "WithdrawCurrency must reject a zero amount")

	_, err = e.WithdrawCurrency(t.Context(), &WithdrawalRequest{Amount: 0.000000001})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "WithdrawCurrency must reject an empty currency")

	_, err = e.WithdrawCurrency(t.Context(), &WithdrawalRequest{Amount: 1, Currency: currency.BTC})
	require.ErrorIs(t, err, errInvalidCurrencyChain, "WithdrawCurrency must reject an empty chain")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.WithdrawCurrency(t.Context(), &WithdrawalRequest{
		Currency: currency.BTC,
		Amount:   0.00000001,
		Chain:    "ERC20",
		Address:  core.BitcoinDonationAddress,
	})
	require.NoError(t, err, "WithdrawCurrency must not error")
	if mockTests {
		exp := &WithdrawalResponse{
			ID:                "210496",
			WithdrawalID:      "w1879219868",
			Timestamp:         types.Time(time.Unix(1542000000, 0)),
			Currency:          currency.BTC,
			WithdrawalAddress: core.BitcoinDonationAddress,
			Amount:            0.00000001,
			Status:            "REQUEST",
			Chain:             "ERC20",
			AssetClass:        "SPOT",
		}
		assert.Equal(t, exp, result, "WithdrawCurrency should decode every field")
	}
}

func TestCancelWithdrawalWithSpecifiedID(t *testing.T) {
	t.Parallel()
	_, err := e.CancelWithdrawalWithSpecifiedID(t.Context(), "")
	require.ErrorIs(t, err, errMissingWithdrawalID, "CancelWithdrawalWithSpecifiedID must reject an empty withdrawal ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelWithdrawalWithSpecifiedID(t.Context(), "1234567")
	require.NoError(t, err, "CancelWithdrawalWithSpecifiedID must not error")
	if mockTests {
		exp := &WithdrawalResponse{
			ID:                "1234567",
			Timestamp:         types.Time(time.Unix(1542000000, 0)),
			Currency:          currency.USDT,
			WithdrawalAddress: "1HkxtBAMrA3tP5ENnYY2CZortjZvFDH5Cs",
			TransactionID:     "128988928203223323290",
			BlockNumber:       18217349,
			Amount:            222.61,
			Status:            "CANCELPEND",
			Chain:             "TRX",
		}
		assert.Equal(t, exp, result, "CancelWithdrawalWithSpecifiedID should decode every field")
	}
}

func TestGetOptionsOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionsOrderbook(t.Context(), currency.EMPTYPAIR, "0.1", 9, true)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOptionsOrderbook must reject an empty contract")

	result, err := e.GetOptionsOrderbook(t.Context(), getPair(t, asset.Options), "0.1", 9, true)
	require.NoError(t, err, "GetOptionsOrderbook must not error")
	if mockTests {
		exp := &OrderbookResponse{
			ID:      2836984,
			Current: types.Time(time.UnixMilli(1774970474039)),
			Update:  types.Time(time.UnixMilli(1774970470308)),
			Bids:    OrderbookLevels{{Price: 13588, Amount: 102}, {Price: 13487, Amount: 144}, {Price: 11639, Amount: 244}, {Price: 11611, Amount: 182}, {Price: 11584, Amount: 173}},
			Asks:    OrderbookLevels{{Price: 13676, Amount: 600}, {Price: 13817, Amount: 102}, {Price: 14577, Amount: 214}, {Price: 14605, Amount: 230}, {Price: 14632, Amount: 197}},
		}
		assert.Equal(t, exp, result, "GetOptionsOrderbook should decode every field")
		return
	}
	assert.NotNil(t, result, "GetOptionsOrderbook should return an orderbook")
}

func TestGetOptionsTickers(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionsTickers(t.Context(), "")
	require.ErrorIs(t, err, errInvalidUnderlying, "GetOptionsTickers must reject an empty underlying")

	result, err := e.GetOptionsTickers(t.Context(), "BTC_USDT")
	require.NoError(t, err, "GetOptionsTickers must not error")
	if mockTests {
		require.Len(t, result, 5, "GetOptionsTickers must return every recorded ticker")
		exp := &OptionsTickerResponse{
			Name:                  currency.NewPairWithDelimiter("BTC", "USDT-20260408-73000-P", currency.UnderscoreDelimiter),
			MarkPrice:             5532.7,
			IndexPrice:            67503.71,
			Ask1Size:              2000,
			Ask1Price:             5672,
			Bid1Size:              2000,
			Bid1Price:             5348,
			Vega:                  2.92983,
			Theta:                 -28.33781,
			Gamma:                 0.00002,
			Delta:                 -0.97772,
			MarkImpliedVolatility: 0.4729,
			AskImpliedVolatility:  0.7037,
			Leverage:              11.926376189058,
		}
		assert.Equal(t, exp, result[0], "GetOptionsTickers should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetOptionsTickers should return tickers")
}

func TestGetOptionUnderlyingTickers(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionUnderlyingTickers(t.Context(), "")
	require.ErrorIs(t, err, errInvalidUnderlying, "GetOptionUnderlyingTickers must reject an empty underlying")

	result, err := e.GetOptionUnderlyingTickers(t.Context(), "BTC_USDT")
	require.NoError(t, err, "GetOptionUnderlyingTickers must not error")
	if mockTests {
		assert.Equal(t, &OptionsUnderlyingTickerResponse{TradePut: 278, TradeCall: 1345, IndexPrice: 67503.71}, result, "GetOptionUnderlyingTickers should decode every field")
		return
	}
	assert.Positive(t, result.IndexPrice.Float64(), "GetOptionUnderlyingTickers should return an index price")
}

func TestGetOptionFuturesCandlesticks(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetOptionFuturesCandlesticks(t.Context(), currency.EMPTYPAIR, kline.ThirtyMin, startTime, endTime, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOptionFuturesCandlesticks must reject an empty contract")

	_, err = e.GetOptionFuturesCandlesticks(t.Context(), getPair(t, asset.Options), kline.ThirtyMin, endTime, startTime, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetOptionFuturesCandlesticks must reject a start after the end")

	_, err = e.GetOptionFuturesCandlesticks(t.Context(), getPair(t, asset.Options), kline.Interval(5), startTime, endTime, 100)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "GetOptionFuturesCandlesticks must reject an unsupported interval")

	result, err := e.GetOptionFuturesCandlesticks(t.Context(), getPair(t, asset.Options), kline.ThirtyMin, startTime, endTime, 1000)
	require.NoError(t, err, "GetOptionFuturesCandlesticks must not error")
	if mockTests {
		assert.Empty(t, result, "GetOptionFuturesCandlesticks should return the recorded empty range")
		return
	}
	assert.NotNil(t, result, "GetOptionFuturesCandlesticks should return a list")
}

func TestGetOptionFuturesMarkPriceCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionFuturesMarkPriceCandlesticks(t.Context(), "", 0, time.Time{}, time.Time{}, kline.OneMonth)
	require.ErrorIs(t, err, errInvalidUnderlying, "GetOptionFuturesMarkPriceCandlesticks must reject an empty underlying")

	startTime, endTime := getTime()
	_, err = e.GetOptionFuturesMarkPriceCandlesticks(t.Context(), "BTC_USDT", 0, endTime, startTime, kline.OneMonth)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetOptionFuturesMarkPriceCandlesticks must reject a start after the end")

	result, err := e.GetOptionFuturesMarkPriceCandlesticks(t.Context(), "BTC_USDT", 0, time.Time{}, time.Time{}, kline.OneMonth)
	require.NoError(t, err, "GetOptionFuturesMarkPriceCandlesticks must not error")
	if mockTests {
		exp := []*FuturesCandlestickResponse{
			{Timestamp: types.Time(time.Unix(1770336000, 0)), ClosePrice: 67264.82, HighestPrice: 74027.87, LowestPrice: 65056.81, OpenPrice: 67126.36},
			{Timestamp: types.Time(time.Unix(1772928000, 0)), ClosePrice: 67116.1, HighestPrice: 75991.85, LowestPrice: 65005.28, OpenPrice: 67264.15},
		}
		assert.Equal(t, exp, result, "GetOptionFuturesMarkPriceCandlesticks should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetOptionFuturesMarkPriceCandlesticks should return candles")
}

func TestGetOptionsTradeHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetOptionsTradeHistory(t.Context(), getPair(t, asset.Options), "C", 0, 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetOptionsTradeHistory must reject a start after the end")

	_, err = e.GetOptionsTradeHistory(t.Context(), getPair(t, asset.Options), "C", 0, 0, startTime, endTime)
	require.NoError(t, err, "GetOptionsTradeHistory must not error for a time range")

	pair := getPair(t, asset.Options)
	result, err := e.GetOptionsTradeHistory(t.Context(), pair, "", 0, 0, time.Time{}, time.Time{})
	require.NoError(t, err, "GetOptionsTradeHistory must not error")
	if mockTests {
		require.Len(t, result, 3, "GetOptionsTradeHistory must return every recorded trade")
		exp := &OptionTradeResponse{
			ID:         3,
			CreateTime: types.Time(time.Unix(1775386206, 0)),
			Contract:   pair,
			Size:       -1,
			Price:      12988,
		}
		assert.Equal(t, exp, result[0], "GetOptionsTradeHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetOptionsTradeHistory should return a list")
}

// Wrapper test functions

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.FetchTradablePairs(t.Context(), a)
		require.NoErrorf(t, err, "FetchTradablePairs must not error for %s", a)
		require.NotEmptyf(t, pairs, "FetchTradablePairs must return some pairs for %s", a)

		if a == asset.USDTMarginedFutures || a == asset.CoinMarginedFutures {
			for _, p := range pairs {
				_, err := getSettlementCurrency(p, a)
				require.NoErrorf(t, err, "Fetched pair %s %s must not error on getSettlementCurrency", a, p)
			}
		}
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.UpdateTickers(t.Context(), asset.Binary), asset.ErrNotSupported, "UpdateTickers must reject an unsupported asset")

	t.Run("all supported assets", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			ex := newIsolatedExchange(t)
			ratsSpot := func() *ticker.Price {
				return &ticker.Price{Last: 0.00003885, High: 0.00004023, Low: 0.00003663, Bid: 0.00003874, Ask: 0.00003879, BaseVolume: 5504390622, QuoteVolume: 209618.0648051, Pair: currency.NewPairWithDelimiter("RATS", "USDT", currency.UnderscoreDelimiter)}
			}
			// Every ticker Gate returns is stored, enabled or not, so the first of each recorded list is checked
			for _, tc := range []struct {
				a   asset.Item
				exp *ticker.Price
			}{
				{a: asset.Spot, exp: ratsSpot()},
				{a: asset.Margin, exp: ratsSpot()},
				{a: asset.CrossMargin, exp: ratsSpot()},
				{
					a:   asset.USDTMarginedFutures,
					exp: &ticker.Price{Last: 84.8, High: 85.24, Low: 84.59, BaseVolume: 37.19, QuoteVolume: 3157, MarkPrice: 84.8, IndexPrice: 85.049, Pair: currency.NewPairWithDelimiter("EWJ", "USDT", currency.UnderscoreDelimiter)},
				},
				{
					a:   asset.CoinMarginedFutures,
					exp: &ticker.Price{Last: 67295.7, High: 67691.7, Low: 66490.4, BaseVolume: 14, QuoteVolume: 964612, MarkPrice: 67501.33, IndexPrice: 67499.05, Pair: getPair(t, asset.CoinMarginedFutures)},
				},
				{
					a:   asset.DeliveryFutures,
					exp: &ticker.Price{Last: 0.091, High: 0.0933, Low: 0.0893, BaseVolume: 103971, QuoteVolume: 9461, MarkPrice: 0.091, IndexPrice: 0.0911, Pair: currency.NewPairWithDelimiter("ARB", "USDT_20260410", currency.UnderscoreDelimiter)},
				},
				{
					a:   asset.Options,
					exp: &ticker.Price{Bid: 3408, BidSize: 455, Ask: 3752, AskSize: 2000, MarkPrice: 3597.4, IndexPrice: 67503.71, Pair: getPair(t, asset.Options)},
				},
			} {
				t.Run(tc.a.String(), func(t *testing.T) {
					t.Parallel()
					started := time.Now()
					require.NoError(t, ex.UpdateTickers(t.Context(), tc.a), "UpdateTickers must not error")
					result, err := ticker.GetTicker(ex.Name, tc.exp.Pair, tc.a)
					require.NoError(t, err, "UpdateTickers must store the ticker")
					assert.WithinRange(t, result.LastUpdated, started, time.Now(), "UpdateTickers should stamp the ticker when storing it")
					tc.exp.LastUpdated = result.LastUpdated
					tc.exp.ExchangeName = ex.Name
					tc.exp.AssetType = tc.a
					assert.Equal(t, tc.exp, result, "UpdateTickers should store every field of the ticker")
				})
			}
			return
		}
		for _, a := range e.GetAssetTypes(false) {
			t.Run(a.String(), func(t *testing.T) {
				t.Parallel()
				switch a {
				case asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.DeliveryFutures, asset.Options:
					pair := getPair(t, a)
					ex := new(Exchange)
					require.NoError(t, testexch.Setup(ex), "Setup must not error")
					if a == asset.Options {
						testexch.UpdatePairsOnce(t, ex)
					}
					// Isolate the cache so another test cannot supply the ticker being checked.
					ex.Name = t.Name()
					require.NoError(t, ex.UpdateTickers(t.Context(), a), "live UpdateTickers must not error")
					got, err := ticker.GetTicker(ex.Name, pair, a)
					require.NoError(t, err, "live UpdateTickers must cache the requested ticker")
					require.NotNil(t, got, "live ticker must not be nil")
					if a == asset.Options {
						// Out-of-the-money options can have a zero mark price near expiry.
						assert.GreaterOrEqual(t, got.MarkPrice, 0.0, "live options mark price should be non-negative")
					} else {
						assert.Positive(t, got.MarkPrice, "live ticker mark price should be positive")
					}
					assert.Positive(t, got.IndexPrice, "live ticker index price should be positive")
				default:
					assert.NoError(t, e.UpdateTickers(t.Context(), a), "UpdateTickers should not error")
				}
			})
		}
	})

	for _, tc := range mockedTickerTestCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			ex.Name = t.Name()
			if tc.asset == asset.Options {
				require.NoError(t, ex.UpdatePairs(currency.Pairs{tc.pair}, tc.asset, true), "mocked options pair must be enabled")
			}

			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method, "ticker request method should be GET")
				assert.Equal(t, tc.expectedPath, r.URL.Path, "ticker request path should match the asset endpoint")
				if tc.asset == asset.Options {
					assert.Equal(t, "BTC_USDT", r.URL.Query().Get("underlying"), "options underlying should match")
					_, err := fmt.Fprintf(w, `[{"name":%q,"last_price":"118.4","mark_price":"118.35","index_price":"100000.25"}]`, tc.pair.String())
					assert.NoError(t, err, "mocked response should be written")
					return
				}
				assert.Empty(t, r.URL.Query().Get("contract"), "bulk ticker request should not filter by contract")
				_, err := fmt.Fprintf(w, `[{"contract":%q,"last":"118.4","low_24h":"99.2","high_24h":"132.5","volume_24h":"745487577","volume_24h_quote":"1665006","mark_price":"118.35","index_price":"118.36"}]`, tc.pair.String())
				assert.NoError(t, err, "mocked ticker response should be written")
			}))

			require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")

			require.NoError(t, ex.UpdateTickers(t.Context(), tc.asset), "mocked bulk ticker update must succeed")
			got, err := ticker.GetTicker(ex.Name, tc.pair, tc.asset)
			require.NoError(t, err, "mocked ticker retrieval must succeed")
			assert.Equal(t, 118.35, got.MarkPrice, "ticker mark price should match the mocked response")
			if tc.asset == asset.Options {
				assert.Equal(t, 100000.25, got.IndexPrice, "options index price should match")
			} else {
				assert.Equal(t, 118.36, got.IndexPrice, "ticker index price should match the mocked response")
			}
		})
	}

	if mockTests {
		return // The base volume check below needs live tickers from an unmocked instance
	}
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
	ex.Name = t.Name()
	require.NoError(t, ex.UpdateTickers(t.Context(), asset.USDTMarginedFutures), "UpdateTickers must not error")

	// a multiplier of 1 reports both fields identically, and truncation to a whole number can
	// cost a full unit, so pick a contract that differs and carries enough base
	tickers, err := ex.GetFuturesTickers(t.Context(), currency.USDT, currency.EMPTYPAIR)
	require.NoError(t, err, "GetFuturesTickers must not error")
	require.NotEmpty(t, tickers, "GetFuturesTickers must return tickers")
	var checked bool
	for i := range tickers {
		tk := tickers[i]
		if tk.Volume24HourBase.Float64() < 1000 || tk.Volume24HourBase.Float64() == tk.Volume24Hour.Float64() {
			continue
		}
		p := tk.Contract
		stored, err := ticker.GetTicker(ex.Name, p, asset.USDTMarginedFutures)
		if err != nil {
			continue
		}
		assert.InEpsilonf(t, tk.Volume24HourBase.Float64(), stored.BaseVolume, 0.05,
			"UpdateTickers should report the base volume for %s, not the contract count in volume_24h", p)
		checked = true
		break
	}
	require.True(t, checked, "must find a futures contract whose base volume and contract count differ")
}

// TestUpdateTickersRequestsEachOptionsUnderlyingOnce checks the options tickers are requested once per underlying,
// since each request returns every contract of the underlying however many of them are enabled
func TestUpdateTickersRequestsEachOptionsUnderlyingOnce(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	put := currency.NewPairWithDelimiter("BTC", "USDT-20261225-100000-P", currency.UnderscoreDelimiter)
	call := currency.NewPairWithDelimiter("BTC", "USDT-20261225-100000-C", currency.UnderscoreDelimiter)
	require.NoError(t, ex.UpdatePairs(currency.Pairs{put, call}, asset.Options, true), "UpdatePairs must not error")

	var requests atomic.Int64
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		assert.Equal(t, "/api/v4/options/tickers", r.URL.Path, "request should reach the options tickers endpoint")
		assert.Equal(t, "BTC_USDT", r.URL.Query().Get("underlying"), "request should name the underlying")
		_, err := fmt.Fprintf(w, `[{"name":%q,"mark_price":"118.35","index_price":"100000.25"},{"name":%q,"mark_price":"7.5","index_price":"100000.25"}]`, put, call)
		assert.NoError(t, err, "writing the tickers should not error")
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")

	require.NoError(t, ex.UpdateTickers(t.Context(), asset.Options), "UpdateTickers must not error")
	assert.Equal(t, int64(1), requests.Load(), "UpdateTickers should request the tickers of an underlying once")
	for _, p := range []currency.Pair{put, call} {
		_, err := ticker.GetTicker(ex.Name, p, asset.Options)
		assert.NoErrorf(t, err, "UpdateTickers should store the %s ticker", p)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	ex := newIsolatedExchange(t)
	_, err := ex.UpdateOrderbook(t.Context(), currency.EMPTYPAIR, 1336)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateOrderbook must reject an empty pair")

	btcusdtSpot := func() *orderbook.Book {
		return &orderbook.Book{
			Bids:         orderbook.Levels{{Price: 67502.6, Amount: 1.705211}, {Price: 67502.4, Amount: 0.0005}, {Price: 67500.1, Amount: 0.013209}, {Price: 67500, Amount: 0.01}, {Price: 67499.9, Amount: 0.0005}},
			Asks:         orderbook.Levels{{Price: 67502.7, Amount: 0.586481}, {Price: 67506, Amount: 0.01}, {Price: 67507.4, Amount: 0.0005}, {Price: 67507.7, Amount: 0.014813}, {Price: 67507.9, Amount: 0.012953}},
			LastUpdated:  time.UnixMilli(1775423974345),
			LastPushed:   time.UnixMilli(1775423974347),
			LastUpdateID: 34890909828,
		}
	}
	for _, tc := range []struct {
		a   asset.Item
		exp *orderbook.Book
	}{
		{a: asset.Spot, exp: btcusdtSpot()},
		{a: asset.Margin, exp: btcusdtSpot()},
		{a: asset.CrossMargin, exp: btcusdtSpot()},
		{
			a: asset.USDTMarginedFutures,
			exp: &orderbook.Book{
				Bids:         orderbook.Levels{{Price: 67477.9, Amount: 19507}, {Price: 67477.5, Amount: 362}, {Price: 67477, Amount: 362}, {Price: 67476.9, Amount: 2}, {Price: 67476.5, Amount: 362}},
				Asks:         orderbook.Levels{{Price: 67478, Amount: 16139}, {Price: 67478.5, Amount: 362}, {Price: 67479, Amount: 362}, {Price: 67479.5, Amount: 362}, {Price: 67479.7, Amount: 222}},
				LastUpdated:  time.UnixMilli(1775423973949),
				LastPushed:   time.UnixMilli(1775423973950),
				LastUpdateID: 116388268171,
			},
		},
		{
			a: asset.CoinMarginedFutures,
			exp: &orderbook.Book{
				Bids:         orderbook.Levels{{Price: 1.17, Amount: 150}, {Price: 1.16, Amount: 203}},
				Asks:         orderbook.Levels{{Price: 1.52, Amount: 100}, {Price: 1.53, Amount: 40}},
				LastUpdated:  time.UnixMilli(1623898993121),
				LastPushed:   time.UnixMilli(1623898993123),
				LastUpdateID: 123456,
			},
		},
		{
			a: asset.DeliveryFutures,
			exp: &orderbook.Book{
				Bids:         orderbook.Levels{{Price: 67848.1, Amount: 1007}, {Price: 67847.2, Amount: 3}, {Price: 67837.9, Amount: 2706}, {Price: 67828.2, Amount: 3}, {Price: 67827.7, Amount: 2979}},
				Asks:         orderbook.Levels{{Price: 67863.8, Amount: 3}, {Price: 67868.6, Amount: 915}, {Price: 67876.1, Amount: 3}, {Price: 67878.8, Amount: 2714}, {Price: 67890, Amount: 3}},
				LastUpdated:  time.UnixMilli(1775423976249),
				LastPushed:   time.UnixMilli(1775423976701),
				LastUpdateID: 175093772,
			},
		},
		{
			a: asset.Options,
			exp: &orderbook.Book{
				Bids:         orderbook.Levels{{Price: 13588, Amount: 102}, {Price: 13487, Amount: 144}, {Price: 11639, Amount: 244}, {Price: 11611, Amount: 182}, {Price: 11584, Amount: 173}},
				Asks:         orderbook.Levels{{Price: 13676, Amount: 600}, {Price: 13817, Amount: 102}, {Price: 14577, Amount: 214}, {Price: 14605, Amount: 230}, {Price: 14632, Amount: 197}},
				LastUpdated:  time.UnixMilli(1774970470308),
				LastPushed:   time.UnixMilli(1774970474039),
				LastUpdateID: 2836984,
			},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			marketEx, pair := getMarketDataPair(t, ex, tc.a, hasTwoSidedDeliveryBook)
			started := time.Now()
			result, err := marketEx.UpdateOrderbook(t.Context(), pair, tc.a)
			require.NoErrorf(t, err, "UpdateOrderbook must not error for %s", pair)
			if !mockTests {
				if tc.a != asset.Options { // Options orderbooks can be empty
					assert.NotEmptyf(t, result.Bids, "UpdateOrderbook should return bids for %s", pair)
					assert.NotEmptyf(t, result.Asks, "UpdateOrderbook should return asks for %s", pair)
				}
				return
			}
			assert.WithinRange(t, result.InsertedAt, started, time.Now(), "UpdateOrderbook should stamp the book when storing it")
			tc.exp.InsertedAt = result.InsertedAt
			tc.exp.Exchange = ex.Name
			tc.exp.Pair = pair
			tc.exp.Asset = tc.a
			tc.exp.ValidateOrderbook = true
			tc.exp.RestSnapshot = true
			assert.Equal(t, tc.exp, result, "UpdateOrderbook should return every level of the book")
		})
	}
}

func TestUpdateOrderbookWithLimit(t *testing.T) {
	t.Parallel()
	ex := newIsolatedExchange(t)
	_, err := ex.UpdateOrderbookWithLimit(t.Context(), currency.EMPTYPAIR, asset.Spot, 20)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateOrderbookWithLimit must reject an empty pair")

	started := time.Now()
	result, err := ex.UpdateOrderbookWithLimit(t.Context(), getPair(t, asset.Spot), asset.Spot, 20)
	require.NoError(t, err, "UpdateOrderbookWithLimit must not error")
	if !mockTests {
		assert.LessOrEqual(t, len(result.Bids), 20, "UpdateOrderbookWithLimit should return no more bids than the limit")
		return
	}
	assert.WithinRange(t, result.InsertedAt, started, time.Now(), "UpdateOrderbookWithLimit should stamp the book when storing it")
	exp := &orderbook.Book{
		Bids:              orderbook.Levels{{Price: 67197, Amount: 0.442669}},
		Asks:              orderbook.Levels{{Price: 67197.1, Amount: 0.485259}},
		Exchange:          ex.Name,
		Pair:              getPair(t, asset.Spot),
		Asset:             asset.Spot,
		LastUpdated:       time.UnixMilli(1774968317359),
		LastPushed:        time.UnixMilli(1774968317360),
		InsertedAt:        result.InsertedAt,
		LastUpdateID:      27596272446,
		ValidateOrderbook: true,
		RestSnapshot:      true,
	}
	assert.Equal(t, exp, result, "UpdateOrderbookWithLimit should return the limited book")
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Empty)
	require.NoError(t, err, "GetWithdrawalsHistory must not error")
	if !mockTests {
		assert.NotNil(t, result, "GetWithdrawalsHistory should return withdrawals")
		return
	}
	exp := []exchange.WithdrawalHistory{
		{
			Status:          "DONE",
			TransferID:      "w1879219868",
			Timestamp:       time.Unix(1744150000, 0),
			Currency:        "BTC",
			Amount:          0.5,
			Fee:             0.0001,
			TransferType:    "withdrawal",
			CryptoToAddress: "1HkxtBAMrA3tP5ENnYY2CZortjZvFDH5Cs",
			CryptoTxID:      "Internal transaction 260594131",
			CryptoChain:     "BTC",
		},
		{
			Status:          "CANCEL",
			TransferID:      "w1879219870",
			Description:     "Withdrawal address is not valid",
			Timestamp:       time.Unix(1744155000, 0),
			Currency:        "BTC",
			Amount:          0.25,
			Fee:             0.0002,
			TransferType:    "withdrawal",
			CryptoToAddress: "bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc",
			CryptoChain:     "BTC",
		},
	}
	assert.Equal(t, exp, result, "GetWithdrawalsHistory should return every withdrawal of the currency")
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountFundingHistory(t.Context())
	require.NoError(t, err, "GetAccountFundingHistory must not error")
	if !mockTests {
		assert.NotNil(t, result, "GetAccountFundingHistory should return records")
		return
	}
	// Deposits and withdrawals are merged in time order, a deposit first where both share a time
	exp := []exchange.FundingHistory{
		{
			ExchangeName:    e.Name,
			Status:          "BLOCKED",
			TransferID:      "d123456",
			Timestamp:       time.Unix(1744150000, 0),
			Currency:        "BTC",
			Amount:          10,
			TransferType:    "deposit",
			CryptoToAddress: "bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc",
			CryptoTxID:      "0xabcdef",
			CryptoChain:     "BTC",
		},
		{
			ExchangeName:    e.Name,
			Status:          "DONE",
			TransferID:      "w1879219868",
			Timestamp:       time.Unix(1744150000, 0),
			Currency:        "BTC",
			Amount:          0.5,
			Fee:             0.0001,
			TransferType:    "withdrawal",
			CryptoToAddress: "1HkxtBAMrA3tP5ENnYY2CZortjZvFDH5Cs",
			CryptoTxID:      "Internal transaction 260594131",
			CryptoChain:     "BTC",
		},
		{
			ExchangeName:    e.Name,
			Status:          "CANCEL",
			TransferID:      "w1879219870",
			Description:     "Withdrawal address is not valid",
			Timestamp:       time.Unix(1744155000, 0),
			Currency:        "BTC",
			Amount:          0.25,
			Fee:             0.0002,
			TransferType:    "withdrawal",
			CryptoToAddress: "bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc",
			CryptoChain:     "BTC",
		},
		{
			ExchangeName:    e.Name,
			Status:          "DONE",
			TransferID:      "210496",
			Timestamp:       time.Unix(1744160000, 0),
			Currency:        "BTC",
			Amount:          222.61,
			TransferType:    "deposit",
			CryptoToAddress: "1HkxtBAMrA3tP5ENnYY2CZortjZvFDH5Cs",
			CryptoTxID:      "128988928203223323290",
			CryptoChain:     "BTC",
		},
	}
	assert.Equal(t, exp, result, "GetAccountFundingHistory should return every deposit and withdrawal")
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecentTrades(t.Context(), getPair(t, asset.Spot), asset.Binary)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetRecentTrades must reject an unsupported asset")

	spotTrades := func() []trade.Data {
		return []trade.Data{
			{TID: "192741997", Side: order.Sell, Price: 67510, Amount: 0.002623, Timestamp: time.UnixMicro(1775423953817400)},
			{TID: "192741998", Side: order.Sell, Price: 67510, Amount: 0.059324, Timestamp: time.UnixMicro(1775423954285137)},
			{TID: "192741999", Side: order.Sell, Price: 67510, Amount: 0.001481, Timestamp: time.UnixMicro(1775423954285142)},
			{TID: "192742000", Side: order.Sell, Price: 67503.9, Amount: 0.000296, Timestamp: time.UnixMicro(1775423954292325)},
			{TID: "192742001", Side: order.Buy, Price: 67502.7, Amount: 0.100977, Timestamp: time.UnixMicro(1775423957197522)},
		}
	}
	for _, tc := range []struct {
		a   asset.Item
		exp []trade.Data
	}{
		{a: asset.Spot, exp: spotTrades()},
		{a: asset.Margin, exp: spotTrades()},
		{a: asset.CrossMargin, exp: spotTrades()},
		{
			// A negative size is a trade whose taker sold, and trades sharing a time keep Gate's order
			a: asset.USDTMarginedFutures,
			exp: []trade.Data{
				{TID: "724267225", Side: order.Buy, Price: 67478, Amount: 1467, Timestamp: time.UnixMilli(1775423969219)},
				{TID: "724267226", Side: order.Buy, Price: 67478, Amount: 105, Timestamp: time.UnixMilli(1775423969612)},
				{TID: "724267228", Side: order.Sell, Price: 67477.9, Amount: 19, Timestamp: time.UnixMilli(1775423970266)},
				{TID: "724267227", Side: order.Sell, Price: 67477.9, Amount: 1, Timestamp: time.UnixMilli(1775423970266)},
				{TID: "724267229", Side: order.Sell, Price: 67477.9, Amount: 4, Timestamp: time.UnixMilli(1775423970950)},
			},
		},
		{
			a: asset.CoinMarginedFutures,
			exp: []trade.Data{
				{TID: "49573344", Side: order.Buy, Price: 67438, Amount: 448, Timestamp: time.UnixMilli(1775423940134)},
				{TID: "49573345", Side: order.Sell, Price: 67296.7, Amount: 530, Timestamp: time.UnixMilli(1775423970135)},
			},
		},
		{
			a: asset.DeliveryFutures,
			exp: []trade.Data{
				{TID: "1017839", Side: order.Sell, Price: 67859.2, Amount: 10, Timestamp: time.UnixMilli(1775423940596)},
				{TID: "1017840", Side: order.Buy, Price: 67858.4, Amount: 1, Timestamp: time.UnixMilli(1775423948920)},
				{TID: "1017841", Side: order.Sell, Price: 67857.5, Amount: 5, Timestamp: time.UnixMilli(1775423954620)},
				{TID: "1017842", Side: order.Sell, Price: 67857.5, Amount: 3, Timestamp: time.UnixMilli(1775423959637)},
				{TID: "1017843", Side: order.Sell, Price: 67854, Amount: 1, Timestamp: time.UnixMilli(1775423963713)},
			},
		},
		{
			a: asset.Options,
			exp: []trade.Data{
				{TID: "1", Side: order.Sell, Price: 10460, Amount: 1, Timestamp: time.Unix(1773735690, 0)},
				{TID: "2", Side: order.Buy, Price: 12227, Amount: 1, Timestamp: time.Unix(1774074656, 0)},
				{TID: "3", Side: order.Sell, Price: 12988, Amount: 1, Timestamp: time.Unix(1775386206, 0)},
			},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			result, err := e.GetRecentTrades(t.Context(), getPair(t, tc.a), tc.a)
			require.NoError(t, err, "GetRecentTrades must not error")
			if !mockTests {
				for i := range result {
					assert.Positivef(t, result[i].Amount, "GetRecentTrades should return an unsigned amount for trade %d", i)
				}
				return
			}
			for i := range tc.exp {
				tc.exp[i].Exchange = e.Name
				tc.exp[i].CurrencyPair = getPair(t, tc.a)
				tc.exp[i].AssetType = tc.a
			}
			assert.Equal(t, tc.exp, result, "GetRecentTrades should return every trade")
		})
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricTrades(t.Context(), getPair(t, asset.Spot), asset.Spot, time.Now().Add(-time.Hour), time.Now())
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetHistoricTrades should not be supported")
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil, "SubmitOrder must reject a nil submission")
	for _, tc := range []struct {
		name   string
		submit *order.Submit
		err    error
	}{
		{"unsupported asset", &order.Submit{AssetType: asset.Binary, Pair: currency.NewBTCUSDT(), Type: order.Limit, Side: order.Buy, Price: 1, Amount: 1}, asset.ErrNotSupported},
		{"futures good till time", &order.Submit{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), Type: order.Limit, Side: order.Buy, Price: 1, Amount: 1, TimeInForce: order.GoodTillTime}, order.ErrUnsupportedTimeInForce},
		{"USDT-margined futures settled in USD", &order.Submit{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSD(), Type: order.Limit, Side: order.Buy, Price: 1, Amount: 1}, errInvalidSettlementQuote},
		{"options fill or kill", &order.Submit{AssetType: asset.Options, Pair: currency.NewBTCUSDT(), Type: order.Limit, Side: order.Buy, Price: 1, Amount: 1, TimeInForce: order.FillOrKill}, order.ErrUnsupportedTimeInForce},
		{"options stop limit", &order.Submit{AssetType: asset.Options, Pair: currency.NewBTCUSDT(), Type: order.StopLimit, Side: order.Buy, Price: 1, TriggerPrice: 2, Amount: 1}, order.ErrUnsupportedOrderType},
		{"options good till cancel market", &order.Submit{AssetType: asset.Options, Pair: currency.NewBTCUSDT(), Type: order.Market, Side: order.Buy, Amount: 1, TimeInForce: order.GoodTillCancel}, order.ErrUnsupportedTimeInForce},
		{"futures stop limit", &order.Submit{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), Type: order.StopLimit, Side: order.Buy, Price: 1, TriggerPrice: 2, Amount: 1}, order.ErrUnsupportedOrderType},
		{"delivery stop limit", &order.Submit{AssetType: asset.DeliveryFutures, Pair: getPair(t, asset.DeliveryFutures), Type: order.StopLimit, Side: order.Buy, Price: 1, TriggerPrice: 2, Amount: 1}, order.ErrUnsupportedOrderType},
		{"margin market buy without quote amount", &order.Submit{AssetType: asset.Margin, Pair: currency.NewBTCUSDT(), Type: order.Market, Side: order.Buy, Amount: 1}, order.ErrAmountMustBeSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.submit.Exchange = e.Name
			_, err := e.SubmitOrder(t.Context(), tc.submit)
			assert.ErrorIs(t, err, tc.err, "SubmitOrder should reject the submission")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	// Prices are far from the market, so a live run leaves resting orders or, for IOC, unfilled orders
	for _, tc := range []struct {
		name   string
		submit *order.Submit
		exp    *order.SubmitResponse
	}{
		{
			name:   "spot",
			submit: &order.Submit{AssetType: asset.Spot, Type: order.Limit, Side: order.Buy, Price: 10000, Amount: 0.001, ClientOrderID: "spot-limit"},
			exp: &order.SubmitResponse{
				Exchange:      e.Name,
				Type:          order.Limit,
				Side:          order.Buy,
				Pair:          getPair(t, asset.Spot),
				AssetType:     asset.Spot,
				Price:         10000,
				Amount:        0.001,
				ClientOrderID: "t-spot-limit",
				Date:          time.UnixMilli(1744150000073),
				LastUpdated:   time.UnixMilli(1744150000074),
				Status:        order.Open,
				OrderID:       "1852454425",
				FeeAsset:      currency.BTC,
			},
		},
		{
			name:   "margin",
			submit: &order.Submit{AssetType: asset.Margin, Type: order.Limit, Side: order.Sell, Price: 200000, Amount: 0.001, ClientOrderID: "margin-limit"},
			exp: &order.SubmitResponse{
				Exchange:      e.Name,
				Type:          order.Limit,
				Side:          order.Sell,
				Pair:          getPair(t, asset.Margin),
				AssetType:     asset.Margin,
				Price:         200000,
				Amount:        0.001,
				ClientOrderID: "t-margin-limit",
				Date:          time.UnixMilli(1744150000173),
				LastUpdated:   time.UnixMilli(1744150000174),
				Status:        order.Open,
				OrderID:       "1852454426",
				FeeAsset:      currency.USDT,
			},
		},
		{
			name:   "cross margin",
			submit: &order.Submit{AssetType: asset.CrossMargin, Type: order.Limit, Side: order.Buy, Price: 10000, Amount: 0.001, TimeInForce: order.PostOnly},
			exp: &order.SubmitResponse{
				Exchange:    e.Name,
				Type:        order.Limit,
				Side:        order.Buy,
				Pair:        getPair(t, asset.CrossMargin),
				AssetType:   asset.CrossMargin,
				TimeInForce: order.PostOnly,
				Price:       10000,
				Amount:      0.001,
				Date:        time.UnixMilli(1744150000273),
				LastUpdated: time.UnixMilli(1744150000274),
				Status:      order.Open,
				OrderID:     "1852454427",
				FeeAsset:    currency.BTC,
			},
		},
		{
			// Gate finishes an unfilled IOC order as ioc, a status the order package has no name for
			name:   "USDT-margined futures IOC finishing unfilled",
			submit: &order.Submit{AssetType: asset.USDTMarginedFutures, Type: order.Limit, Side: order.Buy, Price: 10000, Amount: 1, TimeInForce: order.ImmediateOrCancel, ClientOrderID: "usdt-ioc"},
			exp: &order.SubmitResponse{
				Exchange:        e.Name,
				Type:            order.Limit,
				Side:            order.Buy,
				Pair:            getPair(t, asset.USDTMarginedFutures),
				AssetType:       asset.USDTMarginedFutures,
				TimeInForce:     order.ImmediateOrCancel,
				Price:           10000,
				Amount:          1,
				RemainingAmount: 1,
				ClientOrderID:   "t-usdt-ioc",
				Date:            time.UnixMilli(1744150000123),
				LastUpdated:     time.UnixMilli(1744150000125),
				Status:          order.Cancelled,
				OrderID:         "15675401",
			},
		},
		{
			name:   "coin-margined futures",
			submit: &order.Submit{AssetType: asset.CoinMarginedFutures, Type: order.Limit, Side: order.Buy, Price: 3765, Amount: 6024, ClientOrderID: "my-custom-id"},
			exp: &order.SubmitResponse{
				Exchange:        e.Name,
				Type:            order.Limit,
				Side:            order.Buy,
				Pair:            getPair(t, asset.CoinMarginedFutures),
				AssetType:       asset.CoinMarginedFutures,
				Price:           3765,
				Amount:          6024,
				RemainingAmount: 6024,
				ClientOrderID:   "t-my-custom-id",
				Date:            time.Unix(1546569968, 0),
				LastUpdated:     time.UnixMilli(1546569970123),
				Status:          order.Cancelled,
				OrderID:         "15675394",
			},
		},
		{
			name:   "delivery futures",
			submit: &order.Submit{AssetType: asset.DeliveryFutures, Type: order.Limit, Side: order.Buy, Price: 3765, Amount: 6024, ClientOrderID: "my-custom-id"},
			exp: &order.SubmitResponse{
				Exchange:        e.Name,
				Type:            order.Limit,
				Side:            order.Buy,
				Pair:            getPair(t, asset.DeliveryFutures),
				AssetType:       asset.DeliveryFutures,
				Price:           3765,
				Amount:          6024,
				RemainingAmount: 6024,
				ClientOrderID:   "t-my-custom-id",
				Date:            time.Unix(1546569968, 0),
				LastUpdated:     time.Unix(1546569968, 0),
				Status:          order.Open,
				OrderID:         "15675394",
			},
		},
		{
			// Gate reports a finished options order's status as finished, which only its finish_as explains
			name:   "options IOC finishing unfilled",
			submit: &order.Submit{AssetType: asset.Options, Type: order.Limit, Side: order.Buy, Price: 1, Amount: 1, TimeInForce: order.ImmediateOrCancel, ClientOrderID: "options-ioc"},
			exp: &order.SubmitResponse{
				Exchange:        e.Name,
				Type:            order.Limit,
				Side:            order.Buy,
				Pair:            getPair(t, asset.Options),
				AssetType:       asset.Options,
				TimeInForce:     order.ImmediateOrCancel,
				Price:           1,
				Amount:          1,
				RemainingAmount: 1,
				ClientOrderID:   "t-options-ioc",
				Date:            time.Unix(1744150000, 0),
				LastUpdated:     time.Unix(1744150001, 0),
				Status:          order.Cancelled,
				OrderID:         "5",
			},
		},
		{
			// Gate sells options with a negative size
			name:   "options post only sell",
			submit: &order.Submit{AssetType: asset.Options, Type: order.Limit, Side: order.Sell, Price: 100, Amount: 1, TimeInForce: order.PostOnly, ClientOrderID: "options-sell"},
			exp: &order.SubmitResponse{
				Exchange:        e.Name,
				Type:            order.Limit,
				Side:            order.Sell,
				Pair:            getPair(t, asset.Options),
				AssetType:       asset.Options,
				TimeInForce:     order.PostOnly,
				Price:           100,
				Amount:          1,
				RemainingAmount: 1,
				ClientOrderID:   "t-options-sell",
				Date:            time.Unix(1744150000, 0),
				LastUpdated:     time.Unix(1744150000, 0),
				Status:          order.Open,
				OrderID:         "6",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.submit.Exchange = e.Name
			tc.submit.Pair = getPair(t, tc.submit.AssetType)
			got, err := e.SubmitOrder(t.Context(), tc.submit)
			require.NoError(t, err, "SubmitOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, got, "SubmitOrder should return the placed order")
				return
			}
			assert.NotEmpty(t, got.OrderID, "SubmitOrder should return the order ID")
		})
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyOrder(t.Context(), &order.Modify{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "ModifyOrder should report that it is not supported")
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.CancelOrder(t.Context(), nil), order.ErrCancelOrderIsNil, "CancelOrder must reject a nil cancellation")
	cancel := &order.Cancel{AssetType: asset.Spot, Pair: currency.NewBTCUSDT()}
	require.ErrorIs(t, e.CancelOrder(t.Context(), cancel), order.ErrOrderIDNotSet, "CancelOrder must reject a cancellation without an order ID")
	cancel.OrderID, cancel.Pair = "1234", currency.EMPTYPAIR
	require.ErrorIs(t, e.CancelOrder(t.Context(), cancel), currency.ErrCurrencyPairEmpty, "CancelOrder must reject a cancellation without a pair")
	cancel.Pair, cancel.AssetType = currency.NewBTCUSDT(), asset.Binary
	require.ErrorIs(t, e.CancelOrder(t.Context(), cancel), asset.ErrNotSupported, "CancelOrder must reject an unsupported asset")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for a, orderID := range map[asset.Item]string{
		asset.Spot:                "1234",
		asset.Margin:              "1234",
		asset.CrossMargin:         "1234",
		asset.USDTMarginedFutures: "15675402",
		asset.CoinMarginedFutures: "15675394",
		asset.DeliveryFutures:     "123456",
		asset.Options:             "1234",
	} {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			err := e.CancelOrder(t.Context(), &order.Cancel{OrderID: orderID, Pair: getPair(t, a), AssetType: a})
			assert.NoError(t, err, "CancelOrder should not error")
		})
	}
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelBatchOrders must reject an empty batch")
	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: asset.Binary, Pair: currency.NewBTCUSDT(), OrderID: "1"}})
	require.ErrorIs(t, err, asset.ErrNotSupported, "CancelBatchOrders must reject an unsupported asset")
	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{
		{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), OrderID: "1"},
		{AssetType: asset.DeliveryFutures, Pair: currency.NewBTCUSDT(), OrderID: "2"},
	})
	require.ErrorIs(t, err, errSingleAssetRequired, "CancelBatchOrders must reject a batch of several assets")
	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: asset.Spot, Pair: currency.NewBTCUSDT()}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelBatchOrders must reject an order without an ID")
	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: asset.Spot, OrderID: "1"}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelBatchOrders must reject an order without a pair")
	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSD(), OrderID: "1"}})
	require.ErrorIs(t, err, errInvalidSettlementQuote, "CancelBatchOrders must reject a pair the asset does not settle")

	t.Run("failed requests", func(t *testing.T) {
		t.Parallel()
		// A rejected cancellation of one order must not stop the others from being cancelled
		ex := futuresLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodDelete, r.Method, "each delivery order should be cancelled on its own")
			if r.URL.Path == "/api/v4/delivery/usdt/orders/1" {
				w.WriteHeader(http.StatusBadRequest)
				_, err := w.Write([]byte(`{"label":"ORDER_NOT_FOUND","message":"Order not found"}`))
				assert.NoError(t, err, "Write should not error")
				return
			}
			assert.Equal(t, "/api/v4/delivery/usdt/orders/2", r.URL.Path, "the second order should be cancelled after the first is rejected")
			_, err := w.Write([]byte(`{"id":2,"contract":"BTC_USDT_20260626","size":1,"left":1,"price":"3765","tif":"gtc","status":"finished","finish_as":"cancelled"}`))
			assert.NoError(t, err, "Write should not error")
		})
		pair := currency.NewPairWithDelimiter("BTC", "USDT_20260626", currency.UnderscoreDelimiter)
		got, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{
			{OrderID: "1", Pair: pair, AssetType: asset.DeliveryFutures},
			{OrderID: "2", Pair: pair, AssetType: asset.DeliveryFutures},
		})
		require.ErrorIs(t, err, request.ErrAuthRequestFailed, "CancelBatchOrders must return the rejected request's error")
		require.NotNil(t, got, "CancelBatchOrders must report the outcomes alongside the error")
		assert.Contains(t, got.Status["1"], "ORDER_NOT_FOUND", "the rejected order should report Gate's reason")
		assert.Equal(t, order.Cancelled.String(), got.Status["2"], "the order after the rejected one should still be cancelled")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	spotOutcomes := &order.CancelBatchResponse{Status: map[string]string{"1234567": order.Cancelled.String(), "123453": "ORDER_NOT_FOUND: Order not found"}}
	for _, tc := range []struct {
		a        asset.Item
		orderIDs []string
		exp      *order.CancelBatchResponse
	}{
		{a: asset.Spot, orderIDs: []string{"1234567", "123453"}, exp: spotOutcomes},
		{a: asset.Margin, orderIDs: []string{"1234567", "123453"}, exp: spotOutcomes},
		{a: asset.CrossMargin, orderIDs: []string{"1234567", "123453"}, exp: spotOutcomes},
		{a: asset.USDTMarginedFutures, orderIDs: []string{"1", "2"}, exp: &order.CancelBatchResponse{Status: map[string]string{"1": order.Cancelled.String(), "2": "ORDER_NOT_FOUND"}}},
		{a: asset.CoinMarginedFutures, orderIDs: []string{"15675394"}, exp: &order.CancelBatchResponse{Status: map[string]string{"15675394": order.Cancelled.String()}}},
		{a: asset.DeliveryFutures, orderIDs: []string{"123456", "123457"}, exp: &order.CancelBatchResponse{Status: map[string]string{"123456": order.Cancelled.String(), "123457": order.Cancelled.String()}}},
		{a: asset.Options, orderIDs: []string{"1234", "1235"}, exp: &order.CancelBatchResponse{Status: map[string]string{"1234": order.Cancelled.String(), "1235": order.Cancelled.String()}}},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			cancels := make([]order.Cancel, len(tc.orderIDs))
			for i, id := range tc.orderIDs {
				cancels[i] = order.Cancel{OrderID: id, Pair: getPair(t, tc.a), AssetType: tc.a}
			}
			got, err := e.CancelBatchOrders(t.Context(), cancels)
			require.NoError(t, err, "CancelBatchOrders must not error")
			if mockTests {
				assert.Equal(t, tc.exp, got, "CancelBatchOrders should report each order's own outcome")
				return
			}
			assert.Len(t, got.Status, len(tc.orderIDs), "CancelBatchOrders should report each order's outcome")
		})
	}
}

func TestRecordCancelOutcomes(t *testing.T) {
	t.Parallel()
	requested := []order.Cancel{{OrderID: "1"}, {OrderID: "t-custom"}, {OrderID: "3"}, {OrderID: "4"}}
	resp := &order.CancelBatchResponse{Status: map[string]string{}}
	err := recordCancelOutcomes(resp, requested, []cancelOutcome{
		{orderID: "1", succeeded: true},
		// Gate reports the order ID of an order cancelled by its custom text
		{orderID: "2", text: "t-custom", succeeded: true},
		{orderID: "3", reason: "ORDER_NOT_FOUND: Order not found"},
	}, nil)
	require.ErrorIs(t, err, errCancelOutcomeMissing, "recordCancelOutcomes must report an order Gate returned no outcome for")
	exp := map[string]string{
		"1":        order.Cancelled.String(),
		"t-custom": order.Cancelled.String(),
		"3":        "ORDER_NOT_FOUND: Order not found",
		"4":        errCancelOutcomeMissing.Error(),
	}
	assert.Equal(t, exp, resp.Status, "recordCancelOutcomes should report each requested order's outcome")

	resp = &order.CancelBatchResponse{Status: map[string]string{}}
	err = recordCancelOutcomes(resp, requested[:2], nil, common.ErrNoResponse)
	require.ErrorIs(t, err, common.ErrNoResponse, "recordCancelOutcomes must return a failed request's error")
	assert.Equal(t, map[string]string{"1": common.ErrNoResponse.Error(), "t-custom": common.ErrNoResponse.Error()}, resp.Status, "recordCancelOutcomes should report a failed request against every order it carried")
}

func TestJoinNonEmpty(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "ORDER_NOT_FOUND: Order not found", joinNonEmpty("ORDER_NOT_FOUND", "Order not found"), "joinNonEmpty should join a label and a message")
	assert.Equal(t, "ORDER_NOT_FOUND", joinNonEmpty("ORDER_NOT_FOUND", ""), "joinNonEmpty should return a label without a message")
	assert.Equal(t, "Order not found", joinNonEmpty("", "Order not found"), "joinNonEmpty should return a message without a label")
	assert.Empty(t, joinNonEmpty("", ""), "joinNonEmpty should return nothing for nothing")
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddress(t.Context(), currency.EMPTYCODE, "", "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDepositAddress must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err = e.GetDepositAddress(t.Context(), currency.BTC, "", "GCTNOTACHAIN")
	require.ErrorIs(t, err, deposit.ErrAddressNotFound, "GetDepositAddress must reject a chain without an address")
	if !mockTests {
		chains, err := e.GetAvailableTransferChains(t.Context(), currency.BTC)
		require.NoError(t, err, "GetAvailableTransferChains must not error")
		for _, chain := range chains {
			result, err := e.GetDepositAddress(t.Context(), currency.BTC, "", chain)
			require.NoErrorf(t, err, "GetDepositAddress must not error for chain %s", chain)
			assert.NotEmptyf(t, result.Address, "GetDepositAddress should return an address for chain %s", chain)
		}
		return
	}
	// The fixture records an XLM address Gate failed to obtain
	_, err = e.GetDepositAddress(t.Context(), currency.BTC, "", "XLM")
	require.ErrorIs(t, err, deposit.ErrAddressNotFound, "GetDepositAddress must reject a chain whose address Gate failed to obtain")
	for _, tc := range []struct {
		ccy   currency.Code
		chain string
		exp   *deposit.Address
	}{
		{ccy: currency.BTC, exp: &deposit.Address{Address: "1HkxtBAMrA3tP5ENnYY2CZortjZvFDH5Cs"}},
		{ccy: currency.BTC, chain: "BSC", exp: &deposit.Address{Address: "0x1234567890abcdef1234567890abcdef12345678", Chain: "BSC"}},
		// The tag is the memo a deposit must carry, not the name of its kind
		{ccy: currency.XRP, chain: "XRP", exp: &deposit.Address{Address: "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", Tag: "100156837", Chain: "XRP"}},
	} {
		result, err := e.GetDepositAddress(t.Context(), tc.ccy, "", tc.chain)
		require.NoErrorf(t, err, "GetDepositAddress must not error for %s chain %q", tc.ccy, tc.chain)
		assert.Equalf(t, tc.exp, result, "GetDepositAddress should return the %s address of chain %q", tc.ccy, tc.chain)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetActiveOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil, "GetActiveOrders must reject a nil request")
	_, err = e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Binary, Side: order.AnySide, Type: order.AnyType})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetActiveOrders must reject an unsupported asset")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		a   asset.Item
		exp order.FilteredOrders
	}{
		{
			a: asset.Spot,
			exp: order.FilteredOrders{{
				Exchange:             e.Name,
				OrderID:              "12332324",
				ClientOrderID:        "t-123456",
				Side:                 order.Buy,
				Type:                 order.Limit,
				Status:               order.Open,
				Pair:                 getPair(t, asset.Spot),
				AssetType:            asset.Spot,
				Price:                5.00032,
				Amount:               1,
				ExecutedAmount:       0.5,
				RemainingAmount:      0.5,
				AverageExecutedPrice: 5.00032,
				Cost:                 2.50016,
				CostAsset:            currency.USDT,
				Fee:                  0.005,
				FeeAsset:             currency.BTC,
				Date:                 time.UnixMilli(1744150000123),
				LastUpdated:          time.UnixMilli(1744150100123),
			}},
		},
		// Gate lists every account's open orders together, and the only one recorded is a spot account order
		{a: asset.Margin, exp: order.FilteredOrders{}},
		{a: asset.CrossMargin, exp: order.FilteredOrders{}},
		{
			// An order of another contract is left out
			a: asset.USDTMarginedFutures,
			exp: order.FilteredOrders{{
				Exchange:             e.Name,
				OrderID:              "15675403",
				ClientOrderID:        "t-usdt-open",
				Side:                 order.Short,
				Type:                 order.Limit,
				Status:               order.Open,
				TimeInForce:          order.GoodTillCancel,
				ReduceOnly:           true,
				Pair:                 getPair(t, asset.USDTMarginedFutures),
				AssetType:            asset.USDTMarginedFutures,
				SettlementCurrency:   currency.USDT,
				Price:                31503.3,
				Amount:               10,
				ContractAmount:       10,
				ExecutedAmount:       6,
				RemainingAmount:      4,
				AverageExecutedPrice: 31510,
				Date:                 time.UnixMilli(1744150000500),
				LastUpdated:          time.UnixMilli(1744150100250),
			}},
		},
		{
			a: asset.CoinMarginedFutures,
			exp: order.FilteredOrders{{
				Exchange:           e.Name,
				OrderID:            "15675404",
				Side:               order.Long,
				Type:               order.Limit,
				Status:             order.Open,
				TimeInForce:        order.GoodTillCancel,
				Pair:               getPair(t, asset.CoinMarginedFutures),
				AssetType:          asset.CoinMarginedFutures,
				SettlementCurrency: currency.BTC,
				Price:              50000,
				Amount:             100,
				ContractAmount:     100,
				RemainingAmount:    100,
				Date:               time.Unix(1744150000, 0),
				LastUpdated:        time.Unix(1744150000, 0),
			}},
		},
		{
			a: asset.DeliveryFutures,
			exp: order.FilteredOrders{{
				Exchange:           e.Name,
				OrderID:            "15675394",
				ClientOrderID:      "t-my-custom-id",
				Side:               order.Long,
				Type:               order.Limit,
				Status:             order.Open,
				TimeInForce:        order.GoodTillCancel,
				Pair:               getPair(t, asset.DeliveryFutures),
				AssetType:          asset.DeliveryFutures,
				SettlementCurrency: currency.USDT,
				Price:              3765,
				Amount:             6024,
				ContractAmount:     6024,
				RemainingAmount:    6024,
				Date:               time.Unix(1546569968, 0),
				LastUpdated:        time.Unix(1546569968, 0),
			}},
		},
		{
			// Gate sizes a sell negative
			a: asset.Options,
			exp: order.FilteredOrders{{
				Exchange:        e.Name,
				OrderID:         "2",
				Side:            order.Short,
				Type:            order.Limit,
				Status:          order.Open,
				TimeInForce:     order.GoodTillCancel,
				Pair:            getPair(t, asset.Options),
				AssetType:       asset.Options,
				Price:           100,
				Amount:          1,
				RemainingAmount: 1,
				Date:            time.Unix(1631763361, 0),
				LastUpdated:     time.Unix(1631763361, 0),
			}},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			got, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
				Pairs:     currency.Pairs{getPair(t, tc.a)},
				Type:      order.AnyType,
				Side:      order.AnySide,
				AssetType: tc.a,
			})
			require.NoError(t, err, "GetActiveOrders must not error")
			if mockTests {
				assert.Equal(t, tc.exp, got, "GetActiveOrders should return every open order of the pair")
				return
			}
			for i := range got {
				assert.Equal(t, tc.a, got[i].AssetType, "GetActiveOrders should return orders of the requested asset")
			}
		})
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil, "GetOrderHistory must reject a nil request")
	_, err = e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Binary, Side: order.AnySide, Type: order.AnyType})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOrderHistory must reject an unsupported asset")
	_, err = e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Options, Side: order.AnySide, Type: order.AnyType})
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "GetOrderHistory must require options pairs")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	// spotFill is the spot fill the spot, margin and cross margin requests return, as Gate's trade history covers every
	// account
	spotFill := func(a asset.Item) order.FilteredOrders {
		return order.FilteredOrders{{
			Exchange:             e.Name,
			OrderID:              "987345678235",
			Side:                 order.Buy,
			Pair:                 getPair(t, a),
			AssetType:            a,
			Price:                65000,
			Amount:               0.5,
			ExecutedAmount:       0.5,
			AverageExecutedPrice: 65000,
			Cost:                 32500,
			CostAsset:            currency.USDT,
			Fee:                  0.0005,
			FeeAsset:             currency.BTC,
			Date:                 time.UnixMicro(1744150000123456),
			LastUpdated:          time.UnixMicro(1744150000123456),
			Trades: []order.TradeHistory{{
				TID:       "1232893233",
				Price:     65000,
				Amount:    0.5,
				Fee:       0.0005,
				FeeAsset:  "BTC",
				Side:      order.Buy,
				IsMaker:   true,
				Timestamp: time.UnixMicro(1744150000123456),
				Exchange:  e.Name,
			}},
		}}
	}
	// futuresFill is a futures fill, which Gate sizes negative for a sell, in contracts and identified by its own fill
	// ID besides the ID of the order it filled
	futuresFill := func(a asset.Item, orderID, fillID string, side order.Side, price, amount, fee float64, isMaker bool, date time.Time) order.FilteredOrders {
		settle, err := getSettlementCurrency(currency.EMPTYPAIR, a)
		require.NoErrorf(t, err, "getSettlementCurrency must not error for %s", a)
		return order.FilteredOrders{{
			Exchange:    e.Name,
			OrderID:     orderID,
			Side:        side,
			Pair:        getPair(t, a),
			AssetType:   a,
			Price:       price,
			Amount:      amount,
			CostAsset:   settle,
			Date:        date,
			LastUpdated: date,
			Trades: []order.TradeHistory{{
				TID:       fillID,
				Price:     price,
				Amount:    amount,
				Fee:       fee,
				Side:      side,
				IsMaker:   isMaker,
				Timestamp: date,
				Exchange:  e.Name,
			}},
		}}
	}
	usdtFill := futuresFill(asset.USDTMarginedFutures, "21893289840", "121234235", order.Short, 31500.5, 30, 0.0473, true, time.UnixMilli(1744150000123))
	coinFill := futuresFill(asset.CoinMarginedFutures, "21893289839", "121234231", order.Long, 100.123, 100, 0.01, false, time.UnixMilli(1514764800123))
	deliveryFill := futuresFill(asset.DeliveryFutures, "21893289839", "121234231", order.Short, 100.123, 100, 0.01, true, time.UnixMilli(1514764800123))
	optionsFill := futuresFill(asset.Options, "6", "7", order.Short, 100, 1, 0, true, time.Unix(1631763397, 0))
	for _, tc := range []struct {
		a         asset.Item
		withPairs bool
		exp       order.FilteredOrders
	}{
		{a: asset.Spot, withPairs: true, exp: spotFill(asset.Spot)},
		{a: asset.Spot, exp: spotFill(asset.Spot)},
		{a: asset.Margin, withPairs: true, exp: spotFill(asset.Margin)},
		{a: asset.Margin, exp: spotFill(asset.Margin)},
		{a: asset.CrossMargin, withPairs: true, exp: spotFill(asset.CrossMargin)},
		{a: asset.CrossMargin, exp: spotFill(asset.CrossMargin)},
		{a: asset.USDTMarginedFutures, withPairs: true, exp: usdtFill},
		{a: asset.USDTMarginedFutures, exp: usdtFill},
		{a: asset.CoinMarginedFutures, withPairs: true, exp: coinFill},
		{a: asset.CoinMarginedFutures, exp: coinFill},
		{a: asset.DeliveryFutures, withPairs: true, exp: deliveryFill},
		{a: asset.DeliveryFutures, exp: deliveryFill},
		{a: asset.Options, withPairs: true, exp: optionsFill},
	} {
		name := tc.a.String() + "/without_pairs"
		if tc.withPairs {
			name = tc.a.String() + "/with_pairs"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			req := &order.MultiOrderRequest{Type: order.AnyType, Side: order.AnySide, AssetType: tc.a}
			if tc.withPairs {
				req.Pairs = currency.Pairs{getPair(t, tc.a)}
			}
			got, err := e.GetOrderHistory(t.Context(), req)
			require.NoError(t, err, "GetOrderHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, got, "GetOrderHistory should return every fill")
				return
			}
			for i := range got {
				assert.Equal(t, tc.a, got[i].AssetType, "GetOrderHistory should return fills of the requested asset")
				assert.True(t, got[i].Pair.IsPopulated(), "GetOrderHistory should populate each fill's pair")
			}
		})
	}
}

func TestGetOrderHistoryRequestImmutability(t *testing.T) {
	t.Parallel()
	if !mockTests {
		testexch.UpdatePairsOnce(t, e)
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	enabledPairs := getPairs(t, asset.Spot)
	enabledPairs = enabledPairs[:min(2, len(enabledPairs))]

	type testCase struct {
		name    string
		request order.MultiOrderRequest
	}

	testCases := []testCase{
		{
			name: "nil_pairs",
			request: order.MultiOrderRequest{
				Type:      order.AnyType,
				Side:      order.Buy,
				AssetType: asset.Spot,
			},
		},
		{
			name: "provided_pairs",
			request: order.MultiOrderRequest{
				Type:      order.AnyType,
				Side:      order.Buy,
				Pairs:     slices.Clone(enabledPairs),
				AssetType: asset.Spot,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			expectedPairs := slices.Clone(tc.request.Pairs)
			_, err := e.GetOrderHistory(t.Context(), &tc.request)
			require.NoError(t, err)
			assert.Equal(t, expectedPairs, tc.request.Pairs)
		})
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetHistoricCandles(t.Context(), getPair(t, asset.Options), asset.Options, kline.OneDay, startTime, endTime)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricCandles must reject options")

	spotCandles := func() []kline.Candle {
		return []kline.Candle{
			{Time: time.Unix(1744070400, 0), Open: 79167.3, High: 80866.2, Low: 76249.4, Close: 76324.7, Volume: 3446.19726},
			{Time: time.Unix(1744156800, 0), Open: 76325.7, High: 83585, Low: 74633.5, Close: 82613.5, Volume: 9707.25837},
		}
	}
	for _, tc := range []struct {
		a   asset.Item
		exp []kline.Candle
	}{
		{a: asset.Spot, exp: spotCandles()},
		{a: asset.Margin, exp: spotCandles()},
		{a: asset.CrossMargin, exp: spotCandles()},
		{
			// Futures volumes count contracts
			a: asset.USDTMarginedFutures,
			exp: []kline.Candle{
				{Time: time.Unix(1744070400, 0), Open: 79145.8, High: 80829.6, Low: 76209.7, Close: 76303.1, Volume: 367739297},
				{Time: time.Unix(1744156800, 0), Open: 76303, High: 83598, Low: 74594.3, Close: 82586, Volume: 566153127},
			},
		},
		{
			a: asset.CoinMarginedFutures,
			exp: []kline.Candle{
				{Time: time.Unix(1744070400, 0), Open: 79173.1, High: 80689.2, Low: 76088, Close: 76095.5, Volume: 8333969},
				{Time: time.Unix(1744156800, 0), Open: 76183.3, High: 83450, Low: 74531, Close: 82394.2, Volume: 12502932},
			},
		},
		{
			a: asset.DeliveryFutures,
			exp: []kline.Candle{
				{Time: time.Unix(1744070400, 0), Open: 80020.4, High: 81726.2, Low: 77203.3, Close: 77242, Volume: 66260},
				{Time: time.Unix(1744156800, 0), Open: 77266.7, High: 84575.4, Low: 75565.3, Close: 83650.2, Volume: 65609},
			},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			marketEx, pair := getMarketDataPair(t, e, tc.a, hasDeliveryCandles(kline.OneDay, startTime, endTime))
			result, err := marketEx.GetHistoricCandles(t.Context(), pair, tc.a, kline.OneDay, startTime, endTime)
			require.NoErrorf(t, err, "GetHistoricCandles must not error for %s", pair)
			if !mockTests {
				assert.NotEmptyf(t, result.Candles, "GetHistoricCandles should return candles for %s", pair)
				return
			}
			// The fixtures only match a request spanning whole days, and the candle opening at its end, which Gate also
			// returns, is left out
			exp := &kline.Item{Exchange: e.Name, Pair: getPair(t, tc.a), Asset: tc.a, Interval: kline.OneDay, Candles: tc.exp}
			assert.Equal(t, exp, result, "GetHistoricCandles should return every candle of the window")
		})
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetHistoricCandlesExtended(t.Context(), getPair(t, asset.Options), asset.Options, kline.FourHour, startTime, endTime)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricCandlesExtended must reject options")

	// The fixtures return no candles after midnight, so the last two periods of the window are padded
	padding := []kline.Candle{{Time: time.Unix(1744171200, 0).UTC()}, {Time: time.Unix(1744185600, 0).UTC()}}
	spotCandles := func() []kline.Candle {
		return append([]kline.Candle{
			{Time: time.Unix(1744099200, 0), Open: 79494.3, High: 80065.6, Low: 78830.1, Close: 79981.3, Volume: 367.39632},
			{Time: time.Unix(1744113600, 0), Open: 79987, High: 80371.6, Low: 78185.5, Close: 78497.9, Volume: 788.95289},
			{Time: time.Unix(1744128000, 0), Open: 78497.8, High: 78524.3, Low: 76342.4, Close: 76857.5, Volume: 1039.09048},
			{Time: time.Unix(1744142400, 0), Open: 76857.8, High: 77285.1, Low: 76249.4, Close: 76324.7, Volume: 448.57618},
			{Time: time.Unix(1744156800, 0), Open: 76325.7, High: 76870.7, Low: 74633.5, Close: 75089.6, Volume: 1221.24348},
		}, padding...)
	}
	for _, tc := range []struct {
		a   asset.Item
		exp []kline.Candle
	}{
		{a: asset.Spot, exp: spotCandles()},
		{a: asset.Margin, exp: spotCandles()},
		{a: asset.CrossMargin, exp: spotCandles()},
		{
			a: asset.USDTMarginedFutures,
			exp: append([]kline.Candle{
				{Time: time.Unix(1744099200, 0), Open: 79456.8, High: 80032.3, Low: 78800, Close: 79948.1, Volume: 40040576},
				{Time: time.Unix(1744113600, 0), Open: 79943.7, High: 80342, Low: 78126, Close: 78463.6, Volume: 101353903},
				{Time: time.Unix(1744128000, 0), Open: 78461.2, High: 78493.8, Low: 76302.1, Close: 76824, Volume: 94027904},
				{Time: time.Unix(1744142400, 0), Open: 76828.3, High: 77260.6, Low: 76209.7, Close: 76303.1, Volume: 38795519},
				{Time: time.Unix(1744156800, 0), Open: 76303, High: 76842, Low: 74594.3, Close: 75048.5, Volume: 94167913},
			}, padding...),
		},
		{
			a: asset.CoinMarginedFutures,
			exp: append([]kline.Candle{
				{Time: time.Unix(1744099200, 0), Open: 79262.8, High: 79927.5, Low: 78780, Close: 79861.2, Volume: 759064},
				{Time: time.Unix(1744113600, 0), Open: 79927.5, High: 80249.7, Low: 78126.4, Close: 78460.7, Volume: 1733264},
				{Time: time.Unix(1744128000, 0), Open: 78450, High: 78450, Low: 76247, Close: 76646.4, Volume: 3369527},
				{Time: time.Unix(1744142400, 0), Open: 76742.5, High: 77088.2, Low: 76088, Close: 76095.5, Volume: 1173958},
				{Time: time.Unix(1744156800, 0), Open: 76183.3, High: 76867.5, Low: 74531, Close: 74976.3, Volume: 2822737},
			}, padding...),
		},
		{
			a: asset.DeliveryFutures,
			exp: append([]kline.Candle{
				{Time: time.Unix(1744099200, 0), Open: 80323.9, High: 80872.8, Low: 79664.1, Close: 80824.4, Volume: 10500},
				{Time: time.Unix(1744113600, 0), Open: 80870.9, High: 81285.1, Low: 79096.2, Close: 79406.8, Volume: 11067},
				{Time: time.Unix(1744128000, 0), Open: 79365.6, High: 79415.8, Low: 77285, Close: 77798.5, Volume: 11525},
				{Time: time.Unix(1744142400, 0), Open: 77795.1, High: 78235.5, Low: 77203.3, Close: 77242, Volume: 10375},
				{Time: time.Unix(1744156800, 0), Open: 77266.7, High: 77798.8, Low: 75565.3, Close: 75987.8, Volume: 11106},
			}, padding...),
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			marketEx, pair := getMarketDataPair(t, e, tc.a, hasDeliveryCandles(kline.FourHour, startTime, endTime))
			result, err := marketEx.GetHistoricCandlesExtended(t.Context(), pair, tc.a, kline.FourHour, startTime, endTime)
			require.NoErrorf(t, err, "GetHistoricCandlesExtended must not error for %s", pair)
			if !mockTests {
				assert.NotEmptyf(t, result.Candles, "GetHistoricCandlesExtended should return candles for %s", pair)
				return
			}
			// Spot volumes are the base amounts, as GetHistoricCandles reports them
			exp := &kline.Item{Exchange: e.Name, Pair: getPair(t, tc.a), Asset: tc.a, Interval: kline.FourHour, Candles: tc.exp}
			assert.Equal(t, exp, result, "GetHistoricCandlesExtended should return every candle of the window")
		})
	}

	t.Run("window beyond one request", func(t *testing.T) {
		t.Parallel()
		ex := newIsolatedExchange(t)
		// A limit of four candles a request splits the seven of the window into two requests
		ex.Features.Enabled.Kline.GlobalResultLimit = 4
		result, err := ex.GetHistoricCandlesExtended(t.Context(), getPair(t, asset.Spot), asset.Spot, kline.FourHour, startTime, endTime)
		require.NoError(t, err, "GetHistoricCandlesExtended must not error")
		if !mockTests {
			assert.NotEmpty(t, result.Candles, "GetHistoricCandlesExtended should return candles")
			return
		}
		exp := &kline.Item{Exchange: ex.Name, Pair: getPair(t, asset.Spot), Asset: asset.Spot, Interval: kline.FourHour, Candles: spotCandles()}
		assert.Equal(t, exp, result, "GetHistoricCandlesExtended should merge the candles of every request")
	})
}

// TestGetHistoricCandlesBuildsUnservedIntervals checks an interval Gate does not serve is built from the candles of a
// shorter one it does, requested over whole periods of the interval asked for
func TestGetHistoricCandlesBuildsUnservedIntervals(t *testing.T) {
	t.Parallel()
	type hourlyCandle struct {
		time                                time.Time
		open, high, low, closePrice, volume float64
	}
	// Gate also returns the candle opening at 12:00, the end of the window
	hourly := []hourlyCandle{
		{time.Unix(1744092000, 0), 100, 110, 95, 105, 1},
		{time.Unix(1744095600, 0), 105, 120, 100, 115, 2},
		{time.Unix(1744099200, 0), 115, 118, 90, 92, 3},
		{time.Unix(1744102800, 0), 92, 99, 91, 98, 4},
		{time.Unix(1744106400, 0), 98, 130, 97, 125, 5},
		{time.Unix(1744110000, 0), 125, 126, 110, 111, 6},
		{time.Unix(1744113600, 0), 111, 112, 108, 109, 7},
	}
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	require.NoError(t, storeTestPairs(ex), "storeTestPairs must not error")
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		// The window opening at 08:00 widens to the three hour period it falls in, which opens at 06:00
		assert.Equal(t, "1h", query.Get("interval"), "request should ask for the interval the candles are built from")
		assert.Equal(t, "1744092000", query.Get("from"), "request should start at the period the window starts in")
		assert.Equal(t, "1744113600", query.Get("to"), "request should end at the end of the window")
		candles := make([]string, len(hourly))
		for i, c := range hourly {
			if r.URL.Path == "/api/v4/spot/candlesticks" {
				candles[i] = fmt.Sprintf(`["%d","1","%v","%v","%v","%v","%v","true"]`, c.time.Unix(), c.closePrice, c.high, c.low, c.open, c.volume)
				continue
			}
			candles[i] = fmt.Sprintf(`{"t":%d,"o":"%v","h":"%v","l":"%v","c":"%v","v":%v}`, c.time.Unix(), c.open, c.high, c.low, c.closePrice, c.volume)
		}
		_, err := fmt.Fprint(w, "["+strings.Join(candles, ",")+"]")
		assert.NoError(t, err, "writing the candles should not error")
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")

	start, end := time.Unix(1744099200, 0), time.Unix(1744113600, 0)
	for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures, asset.DeliveryFutures} {
		pair := mockTestPairs()[a]
		exp := &kline.Item{
			Exchange: ex.Name,
			Pair:     pair,
			Asset:    a,
			Interval: kline.ThreeHour,
			Candles: []kline.Candle{
				{Time: time.Unix(1744092000, 0), Open: 100, High: 120, Low: 90, Close: 92, Volume: 6},
				{Time: time.Unix(1744102800, 0), Open: 92, High: 130, Low: 91, Close: 111, Volume: 15},
			},
		}
		result, err := ex.GetHistoricCandles(t.Context(), pair, a, kline.ThreeHour, start, end)
		require.NoErrorf(t, err, "GetHistoricCandles must not error for %s", a)
		assert.Equalf(t, exp, result, "GetHistoricCandles should build the %s candles from hourly ones", a)
		result, err = ex.GetHistoricCandlesExtended(t.Context(), pair, a, kline.ThreeHour, start, end)
		require.NoErrorf(t, err, "GetHistoricCandlesExtended must not error for %s", a)
		assert.Equalf(t, exp, result, "GetHistoricCandlesExtended should build the %s candles from hourly ones", a)
	}
}

func TestGetUnderlyingFromCurrencyPair(t *testing.T) {
	t.Parallel()
	uly, err := e.GetUnderlyingFromCurrencyPair(currency.Pair{Delimiter: currency.UnderscoreDelimiter, Base: currency.BTC, Quote: currency.NewCode("USDT_LLK")})
	require.NoError(t, err)
	assert.True(t, uly.Equal(currency.NewBTCUSDT()))
}

func TestTickerHandlersSkipEmptyBatches(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	require.NoError(t, ex.processTicker(t.Context(), []byte(`{"currency_pair":"UNTRACKED_USDT","last":"1"}`), time.Now()), "processTicker must not error for an untracked pair")
	require.NoError(t, ex.processFuturesTickers(t.Context(), []byte(`{"time":1541659086,"channel":"futures.tickers","event":"update","result":[]}`), asset.USDTMarginedFutures), "processFuturesTickers must not error for an empty result")
	assert.Empty(t, ex.Websocket.DataHandler.C, "ticker handlers should not relay an empty batch")
}

var mapSpotPublicPushData = map[string]string{
	"spot.tickers":           `{"time": 1606291803,	"channel": "spot.tickers",	"event": "update",	"result": {	  "currency_pair": "BTC_USDT",	  "last": "19106.55",	  "lowest_ask": "19108.71",	  "highest_bid": "19106.55",	  "change_percentage": "3.66",	  "base_volume": "2811.3042155865",	  "quote_volume": "53441606.52411221454674732293",	  "high_24h": "19417.74",	  "low_24h": "18434.21"	}}`,
	"spot.trades":            `{"time": 1606292218,	"channel": "spot.trades",	"event": "update",	"result": {	  "id": 309143071,	  "create_time": 1606292218,	  "create_time_ms": "1606292218213.4578",	  "side": "sell",	  "currency_pair": "BTC_USDT",	  "amount": "16.4700000000",	  "price": "0.4705000000"}}`,
	"spot.candlesticks":      `{"time": 1606292600,	"channel": "spot.candlesticks",	"event": "update",	"result": {	  "t": "1606292580",	  "v": "2362.32035",	  "c": "19128.1",	  "h": "19128.1",	  "l": "19128.1",	  "o": "19128.1","n": "1m_BTC_USDT"}}`,
	"spot.book_ticker":       `{"time": 1606293275,	"channel": "spot.book_ticker",	"event": "update",	"result": {	  "t": 1606293275123,	  "u": 48733182,	  "s": "BTC_USDT",	  "b": "19177.79",	  "B": "0.0003341504",	  "a": "19179.38",	  "A": "0.09"	}}`,
	"spot.order_book_update": `{"time": 1606294781,	"channel": "spot.order_book_update",	"event": "update",	"result": {	  "t": 1606294781123,	  "e": "depthUpdate",	  "E": 1606294781,"s": "BTC_USDT","U": 48776301,"u": 48776306,"b": [["19137.74","0.0001"],["19088.37","0"]],"a": [["19137.75","0.6135"]]	}}`,
	"spot.order_book":        `{"time":1606295412,"channel": "spot.order_book",	"event": "update",	"result": {	  "t": 1606295412123,	  "lastUpdateId": 48791820,	  "s": "BTC_USDT",	  "bids": [		[		  "19079.55",		  "0.0195"		],		[		  "19079.07",		  "0.7341"],["19076.23",		  "0.00011808"		],		[		  "19073.9",		  "0.105"		],		[		  "19068.83",		  "0.1009"		]	  ],	  "asks": [		[		  "19080.24",		  "0.1638"		],		[		  "19080.91","0.1366"],["19080.92","0.01"],["19081.29","0.01"],["19083.8","0.097"]]}}`,
	"spot.orders":            `{"time": 1605175506,	"channel": "spot.orders",	"event": "update",	"result": [	  {		"id": "30784435",		"user": 123456,		"text": "t-abc",		"create_time": "1605175506",		"create_time_ms": "1605175506123",		"update_time": "1605175506",		"update_time_ms": "1605175506123",		"event": "put",		"currency_pair": "BTC_USDT",		"type": "limit",		"account": "spot",		"side": "sell",		"amount": "1",		"price": "10001",		"time_in_force": "gtc",		"left": "1",		"filled_total": "0",		"fee": "0",		"fee_currency": "USDT",		"point_fee": "0",		"gt_fee": "0",		"gt_discount": true,		"rebated_fee": "0",		"rebated_fee_currency": "USDT"}	]}`,
	"spot.usertrades":        `{"time": 1605176741,	"channel": "spot.usertrades",	"event": "update",	"result": [	  {		"id": 5736713,		"user_id": 1000001,		"order_id": "30784428",		"currency_pair": "BTC_USDT",		"create_time": 1605176741,		"create_time_ms": "1605176741123.456",		"side": "sell",		"amount": "1.00000000",		"role": "taker",		"price": "10000.00000000",		"fee": "0.00200000000000",		"point_fee": "0",		"gt_fee": "0",		"text": "apiv4"	  }	]}`,
	"spot.cross_loan":        `{"time":1658289372,	"channel":"spot.cross_loan",	"event":"update",	"result":{	  "timestamp":1658289372338,	  "user":"1000001",	  "currency":"BTC",	  "change":"0.01",	  "total":"4.992341029566",	  "available":"0.078054772536",	  "borrowed":"0.01",	  "interest":"0.00001375"	}}`,
}

func TestWebsocketHandleSpotData(t *testing.T) {
	t.Parallel()
	for k, v := range mapSpotPublicPushData {
		t.Run(k, func(t *testing.T) {
			t.Parallel()
			err := e.WebsocketHandleSpotData(t.Context(), nil, []byte(v))
			require.NoError(t, err)
		})
	}
}

var mapSpotPrivatePushData = map[string]string{
	"spot.balances":         `{"time": 1605248616,	"channel": "spot.balances",	"event": "update",	"result": [	  {		"timestamp": "1605248616",		"timestamp_ms": "1605248616123",		"user": "1000001",		"currency": "USDT",		"change": "100",		"total": "1032951.325075926","available": "1022943.325075926"}	]}`,
	"spot.funding_balances": `{"time": 1605248616,	"channel": "spot.funding_balances",	"event": "update",	"result": [	  {"timestamp": "1605248616","timestamp_ms": "1605248616123","user": "1000001","currency": "USDT","change": "100","freeze": "100","lent": "0"}	]}`,
	"spot.cross_balances":   `{"time": 1605248616,"channel": "spot.cross_balances","event": "update",	"result": [{"timestamp": "1605248616","timestamp_ms": "1605248616123","user": "1000001","currency": "USDT",	"change": "100","total": "1032951.325075926","available": "1022943.325075926"}]}`,
}

func TestWebsocketHandleSpotDataPrivate(t *testing.T) {
	t.Parallel()
	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "test", Secret: "test"})
	for x := range mapSpotPrivatePushData {
		err := e.WebsocketHandleSpotData(ctx, nil, []byte(mapSpotPrivatePushData[x]))
		require.NoError(t, err)
	}
}

// TestFuturesDataHandler ensures that messages from various futures channels do not error
func TestFuturesDataHandler(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	testexch.FixtureToDataHandler(t, "testdata/wsFutures.json", func(ctx context.Context, m []byte) error {
		if strings.Contains(string(m), "futures.balances") {
			ctx = accounts.DeployCredentialsToContext(ctx, &accounts.Credentials{Key: "test", Secret: "test"})
		}
		return e.WebsocketHandleFuturesData(ctx, nil, m, asset.CoinMarginedFutures)
	})
	e.Websocket.DataHandler.Close()
	assert.Len(t, e.Websocket.DataHandler.C, 15, "Should see the correct number of messages")
	var sawPosition, sawPositionClose bool
	for resp := range e.Websocket.DataHandler.C {
		if err, isErr := resp.Data.(error); isErr {
			assert.NoError(t, err, "Should not get any errors down the data handler")
		}
		if positions, ok := resp.Data.([]futures.Position); ok {
			require.Len(t, positions, 1, "position update must contain one position")
			assert.Equal(t, asset.CoinMarginedFutures, positions[0].Asset, "asset should match the websocket")
			assert.Equal(t, "BTC_USD", positions[0].Pair.String(), "pair should be normalised")
			assert.Equal(t, currency.BTC, positions[0].Underlying, "underlying should be normalised")
			assert.Equal(t, currency.BTC, positions[0].CollateralCurrency, "collateral currency should be normalised")
			assert.Equal(t, order.Long, positions[0].LatestDirection, "direction should be normalised")
			if positions[0].CloseDate.IsZero() {
				sawPosition = true
				assert.Equal(t, order.Open, positions[0].Status, "position status should be open")
				assert.Equal(t, "3", positions[0].LatestSize.String(), "size should be normalised")
				assert.Equal(t, "5", positions[0].Leverage.String(), "replacement cross-margin leverage should take precedence")
				assert.True(t, positions[0].PositionMargin.Equal(decimal.MustFromFloat(49.999890611186)), "position margin should be populated")
				assert.True(t, positions[0].MaintenanceMarginFraction.Equal(decimal.MustFromFloat(0.005)), "maintenance margin rate should be populated")
				assert.True(t, positions[0].EstimatedLiquidationPrice.Equal(decimal.MustFromFloat(0.1)), "liquidation price should be populated")
				assert.Equal(t, int64(42), positions[0].UpdateID, "update ID should be populated")
			} else {
				sawPositionClose = true
				assert.Equal(t, order.Closed, positions[0].Status, "position close status should be closed")
				assert.True(t, positions[0].LatestSize.IsZero(), "close size should be zero")
			}
		}
	}
	require.True(t, sawPosition, "futures fixture must emit a normalised position")
	require.True(t, sawPositionClose, "futures fixture must emit a normalised position close")
}

func TestFuturesPositionBOBCapturedPayloads(t *testing.T) {
	t.Parallel()
	payloads := [][]byte{
		[]byte(`{"time":1788148634,"time_ms":1788148634648,"channel":"futures.positions","event":"update","result":[{"time":1788148634,"time_ms":1788148634648,"size":"112","user":"12870774","leverage_max":10,"mode":"single","update_id":2498,"cross_leverage_limit":20,"liq_price":0,"maintenance_rate":0.9,"risk_limit":2500000,"last_close_pnl":-0.2543647646,"entry_price":0.005032,"leverage":1,"realised_pnl":-0.027052032,"contract":"BOB_USDT","history_point":0,"margin":56.4006688,"realised_point":0,"history_pnl":-274.6956184238,"pos_margin_mode":"isolated","lever":"1"}]}`),
		[]byte(`{"time":1788148634,"time_ms":1788148634648,"channel":"futures.positions","event":"update","result":[{"time":1788148634,"time_ms":1788148634648,"size":"126","user":"12870774","leverage_max":10,"mode":"single","update_id":2499,"cross_leverage_limit":20,"liq_price":0,"maintenance_rate":0.9,"risk_limit":2500000,"last_close_pnl":-0.2543647646,"entry_price":0.005032444444,"leverage":1,"realised_pnl":-0.030436224,"contract":"BOB_USDT","history_point":0,"margin":63.4563566,"realised_point":0,"history_pnl":-274.6956184238,"pos_margin_mode":"isolated","lever":"1"}]}`),
	}
	wantSizes := []string{"112", "126"}
	wantUpdateIDs := []int64{2498, 2499}
	wantMargins := []float64{56.4006688, 63.4563566}
	wantOpeningPrices := []float64{0.005032, 0.005032444444}
	wantRealisedPNLs := []float64{-0.027052032, -0.030436224}

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.USDTMarginedFutures,
		currency.Pairs{currency.NewPair(currency.NewCode("BOB"), currency.USDT)}, false),
		"StorePairs must register the captured contract")
	for i := range payloads {
		response := struct {
			Time      types.Time                 `json:"time"`
			TimeMilli types.Time                 `json:"time_ms"`
			Channel   string                     `json:"channel"`
			Event     string                     `json:"event"`
			Result    []WebsocketFuturesPosition `json:"result"`
		}{}
		decoder := json.NewDecoder(bytes.NewReader(payloads[i]))
		decoder.DisallowUnknownFields()
		require.NoError(t, decoder.Decode(&response), "captured BOB position payload must unmarshal without unknown fields")
		require.Len(t, response.Result, 1, "captured BOB payload must contain one position")
		assert.Equal(t, "BOB_USDT", response.Result[0].Contract, "captured contract should be preserved")

		require.NoError(t, ex.WebsocketHandleFuturesData(t.Context(), nil, payloads[i], asset.USDTMarginedFutures),
			"WebsocketHandleFuturesData must process the captured position update")
		message := <-ex.Websocket.DataHandler.C
		positions, ok := message.Data.([]futures.Position)
		require.True(t, ok, "captured payload must emit canonical futures positions")
		require.Len(t, positions, 1, "captured payload must emit one position")
		position := positions[0]
		assert.Equal(t, order.Open, position.Status, "position status should be open")
		assert.Equal(t, order.Long, position.LatestDirection, "positive position size should be long")
		assert.Equal(t, wantSizes[i], position.LatestSize.String(), "position size should be preserved")
		assert.Equal(t, "1", position.Leverage.String(), "isolated position should use the quoted lever field")
		assert.Equal(t, wantUpdateIDs[i], position.UpdateID, "position update ID should be preserved")
		assert.Equal(t, time.UnixMilli(1788148634648), position.LastUpdated, "position update time should preserve milliseconds")
		assert.True(t, position.OpeningPrice.Equal(decimal.MustFromFloat(wantOpeningPrices[i])), "position opening price should be correct")
		assert.True(t, position.PositionMargin.Equal(decimal.MustFromFloat(wantMargins[i])), "position margin should be correct")
		assert.True(t, position.RealisedPNL.Equal(decimal.MustFromFloat(wantRealisedPNLs[i])), "position realised PNL should be correct")
	}
}

func TestProcessPositionCloseData(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name       string
		text       string
		riskStatus order.Status
	}{
		{name: "web", text: "web"},
		{name: "liquidation", text: "liquidation", riskStatus: order.Liquidated},
		{name: "liquidation prefix", text: "liq-123", riskStatus: order.Liquidated},
		{name: "hedge liquidation prefix", text: "hedge-liq-123", riskStatus: order.Liquidated},
		{name: "portfolio margin liquidation", text: "pm_liquidate", riskStatus: order.Liquidated},
		{name: "combined margin liquidation", text: "comb_margin_liquidate", riskStatus: order.Liquidated},
		{name: "single currency margin liquidation", text: "scm_liquidate", riskStatus: order.Liquidated},
		{name: "insurance", text: "insurance", riskStatus: order.Liquidated},
		{name: "auto deleveraging", text: "auto_deleveraging", riskStatus: order.AutoDeleverage},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
			payload := []byte(`{"time":1541505434,"channel":"futures.position_closes","event":"update","result":[{"contract":"BTC_USD","pnl":-0.000624354791,"side":"long","text":"` + tt.text + `","time":1547198562,"time_ms":1547198562123,"user":"211xxxx"}]}`)
			require.NoError(t, ex.processPositionCloseData(t.Context(), payload, asset.CoinMarginedFutures),
				"processPositionCloseData must process the close update")
			message := <-ex.Websocket.DataHandler.C
			positions, ok := message.Data.([]futures.Position)
			require.True(t, ok, "position close must emit canonical futures positions")
			require.Len(t, positions, 1, "position close must emit one position")
			assert.Equal(t, order.Closed, positions[0].Status, "position close status should identify a completed lifecycle")
			assert.Equal(t, tt.riskStatus, positions[0].RiskStatus, "position risk status should preserve its close cause")
			assert.Equal(t, order.Long, positions[0].OpeningDirection, "opening direction should be preserved")
			assert.Equal(t, order.Long, positions[0].LatestDirection, "latest direction should be preserved")
		})
	}
}

func TestFuturesPositionCapturedPayload(t *testing.T) {
	t.Parallel()
	payload := []byte(`{"time":1787206135,"time_ms":1787206135040,"channel":"futures.positions","event":"update","result":[{"time":1787206135,"time_ms":1787206135038,"size":"-55","user":"12870774","leverage_max":25,"mode":"single","update_id":267,"cross_leverage_limit":40,"liq_price":0.023554,"maintenance_rate":0.9,"risk_limit":2500000,"last_close_pnl":-0.802254544,"entry_price":0.012071295652,"leverage":1,"realised_pnl":0.079661896,"contract":"GPS_USDT","history_point":0,"margin":66.491714276087,"realised_point":0,"history_pnl":17.1379873471,"pos_margin_mode":"isolated","lever":"1"}]}`)
	response := struct {
		Time      types.Time                 `json:"time"`
		TimeMilli types.Time                 `json:"time_ms"`
		Channel   string                     `json:"channel"`
		Event     string                     `json:"event"`
		Result    []WebsocketFuturesPosition `json:"result"`
	}{}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	require.NoError(t, decoder.Decode(&response), "captured position payload must unmarshal without unknown fields")
	require.Len(t, response.Result, 1, "captured payload must contain one position")
	assert.Equal(t, time.UnixMilli(1787206135038), response.Result[0].Time.Time(), "position timestamp should preserve millisecond precision")

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	require.NoError(t, e.processFuturesPositionsNotification(t.Context(), payload, asset.USDTMarginedFutures))
	message := <-e.Websocket.DataHandler.C
	positions, ok := message.Data.([]futures.Position)
	require.True(t, ok, "captured payload must emit canonical futures positions")
	require.Len(t, positions, 1, "captured payload must emit one position")
	position := positions[0]
	assert.Equal(t, "GPS_USDT", position.Pair.String(), "position pair should be normalised")
	assert.Equal(t, order.Open, position.Status, "position status should be open")
	assert.Equal(t, order.Short, position.LatestDirection, "position direction should be short")
	assert.Equal(t, "55", position.LatestSize.String(), "position size should be absolute")
	assert.Equal(t, "1", position.Leverage.String(), "position leverage should use the isolated leverage field")
	assert.Equal(t, int64(267), position.UpdateID, "position update ID should be preserved")
	assert.Equal(t, time.UnixMilli(1787206135038), position.LastUpdated, "position update time should be preserved")
	assert.True(t, position.OpeningPrice.Equal(decimal.MustFromFloat(0.012071295652)), "position opening price should be correct")
	assert.True(t, position.PositionMargin.Equal(decimal.MustFromFloat(66.491714276087)), "position margin should be correct")
	assert.True(t, position.MaintenanceMarginFraction.Equal(decimal.MustFromFloat(0.9)), "position maintenance rate should be correct")
	assert.True(t, position.EstimatedLiquidationPrice.Equal(decimal.MustFromFloat(0.023554)), "position liquidation price should be correct")
	assert.True(t, position.RealisedPNL.Equal(decimal.MustFromFloat(0.079661896)), "position realised PNL should be correct")

	classicCrossPayload := []byte(strings.NewReplacer(
		`"leverage":1`, `"leverage":0`,
		`"pos_margin_mode":"isolated"`, `"pos_margin_mode":""`,
		`"lever":"1"`, `"lever":"0"`,
	).Replace(string(payload)))
	require.NoError(t, e.processFuturesPositionsNotification(t.Context(), classicCrossPayload, asset.USDTMarginedFutures),
		"processFuturesPositionsNotification must process classic cross margin")
	message = <-e.Websocket.DataHandler.C
	positions, ok = message.Data.([]futures.Position)
	require.True(t, ok, "classic cross-margin payload must emit canonical futures positions")
	require.Len(t, positions, 1, "classic cross-margin payload must emit one position")
	assert.Equal(t, "40", positions[0].Leverage.String(),
		"classic cross-margin leverage should use the cross leverage limit")

	closedPayload := []byte(strings.Replace(string(payload), `"size":"-55"`, `"size":"0"`, 1))
	require.NoError(t, e.processFuturesPositionsNotification(t.Context(), closedPayload, asset.USDTMarginedFutures),
		"processFuturesPositionsNotification must process a zero-size update")
	message = <-e.Websocket.DataHandler.C
	positions, ok = message.Data.([]futures.Position)
	require.True(t, ok, "zero-size update must emit canonical futures positions")
	require.Len(t, positions, 1, "zero-size update must emit one position")
	assert.Equal(t, order.Closed, positions[0].Status, "zero-size position status should be closed")
	assert.Equal(t, order.UnknownSide, positions[0].LatestDirection, "zero-size single-mode direction should be unknown")
	assert.Equal(t, time.UnixMilli(1787206135038), positions[0].CloseDate, "zero-size position close time should be preserved")

	for _, tt := range []struct {
		mode      string
		direction order.Side
	}{
		{mode: "dual_long", direction: order.Long},
		{mode: "dual_short", direction: order.Short},
	} {
		t.Run(tt.mode, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
			dualPayload := []byte(strings.Replace(string(closedPayload), `"mode":"single"`, `"mode":"`+tt.mode+`"`, 1))
			require.NoError(t, ex.processFuturesPositionsNotification(t.Context(), dualPayload, asset.USDTMarginedFutures),
				"processFuturesPositionsNotification must process a zero-size dual-mode update")
			message := <-ex.Websocket.DataHandler.C
			positions, ok := message.Data.([]futures.Position)
			require.True(t, ok, "zero-size dual-mode update must emit canonical futures positions")
			require.Len(t, positions, 1, "zero-size dual-mode update must emit one position")
			assert.Equal(t, tt.direction, positions[0].LatestDirection, "zero-size dual-mode direction should be preserved")
		})
	}

	t.Run("delivery collateral", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
		deliveryPairs, err := ex.GetEnabledPairs(asset.DeliveryFutures)
		require.NoError(t, err, "GetEnabledPairs must return delivery pairs")
		require.NotEmpty(t, deliveryPairs, "delivery pairs must not be empty")
		deliveryPair := deliveryPairs[0]
		formattedPair, err := ex.FormatExchangeCurrency(deliveryPair, asset.DeliveryFutures)
		require.NoError(t, err, "FormatExchangeCurrency must format the delivery pair")
		deliveryPayload := []byte(strings.Replace(string(payload), "GPS_USDT", formattedPair.String(), 1))
		require.NoError(t, ex.processFuturesPositionsNotification(t.Context(), deliveryPayload, asset.DeliveryFutures),
			"processFuturesPositionsNotification must process a delivery position")
		message := <-ex.Websocket.DataHandler.C
		positions, ok := message.Data.([]futures.Position)
		require.True(t, ok, "delivery update must emit canonical futures positions")
		require.Len(t, positions, 1, "delivery update must emit one position")
		assert.Equal(t, currency.USDT, positions[0].CollateralCurrency,
			"delivery collateral currency should exclude the dated contract suffix")
	})
}

func TestProcessFuturesCandlesticksIntervalMapping(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")

	payload := []byte(`{"time":1606292600,"channel":"futures.candlesticks","event":"update","result":[{"t":1606292580,"v":"2362.32035","c":"19128.1","h":"19128.1","l":"19128.1","o":"19128.1","n":"1m_BTC_USDT"}]}`)
	require.NoError(t, ex.processFuturesCandlesticks(t.Context(), payload, asset.USDTMarginedFutures))

	select {
	case msg := <-ex.Websocket.DataHandler.C:
		got, ok := msg.Data.([]kline.Item)
		expectedPair := currency.NewPairWithDelimiter("BTC", "USDT", "_")
		require.True(t, ok, "expected []kline.Item")
		assert.Equal(t, []kline.Item{{
			Pair:     expectedPair,
			Asset:    asset.USDTMarginedFutures,
			Exchange: ex.Name,
			Interval: kline.OneMin,
			Candles: []kline.Candle{{
				Time:   time.Unix(1606292580, 0),
				Open:   19128.1,
				Close:  19128.1,
				High:   19128.1,
				Low:    19128.1,
				Volume: 2362.32035,
			}},
		}}, got)
	default:
		require.Fail(t, "expected futures websocket candle payload")
	}
}

// ******************************************** Options web-socket unit test funcs ********************

var wsOptionsPushDataHandler = map[string]string{
	"options.contract_tickers":      `{"time": 1630576352,	"channel": "options.contract_tickers",	"event": "update",	"result": {"name": "BTC_USDT-20211231-59800-P",    "last_price": "11349.5",    "mark_price": "11170.19",    "index_price": "",    "position_size": 993,    "bid1_price": "10611.7",    "bid1_size": 100,    "ask1_price": "11728.7",    "ask1_size": 100,    "vega": "34.8731",    "theta": "-72.80588",    "rho": "-28.53331",    "gamma": "0.00003",    "delta": "-0.78311",    "mark_iv": "0.86695",    "bid_iv": "0.65481",    "ask_iv": "0.88145",    "leverage": "3.5541112718136"	}}`,
	"options.ul_tickers":            `{"time": 1630576352,	"channel": "options.ul_tickers",	"event": "update",	"result": {"trade_put": 800,	   "trade_call": 41700,	   "index_price": "50695.43",	   "name": "BTC_USDT"	}}`,
	"options.trades":                `{"time": 1630576356,	"channel": "options.trades",	"event": "update",	"result": [{"contract": "BTC_USDT-20211231-59800-C",        "create_time": 1639144526,        "id": 12279,        "price": 997.8,        "size": -100,        "create_time_ms": 1639144526597,        "underlying": "BTC_USDT"    }	]}`,
	"options.ul_trades":             `{"time": 1630576356,	"channel": "options.ul_trades",	"event": "update",	"result": [{"contract": "BTC_USDT-20211231-59800-C","create_time": 1639144526,"id": 12279,"price": 997.8,"size": -100,"create_time_ms": 1639144526597,"underlying": "BTC_USDT","is_call": true}	]}`,
	"options.ul_price":              `{"time": 1630576356,	"channel": "options.ul_price",	"event": "update",	"result": {"underlying": "BTC_USDT",	   "price": 49653.24,"time": 1639143988,"time_ms": 1639143988931}}`,
	"options.mark_price":            `{"time": 1630576356,	"channel": "options.mark_price",	"event": "update",	"result": {"contract": "BTC_USDT-20211231-59800-P",    "price": 11021.27,    "time": 1639143401,    "time_ms": 1639143401676}}`,
	"options.settlements":           `{"time": 1630576356,	"channel": "options.settlements",	"event": "update",	"result": {"contract": "BTC_USDT-20211130-55000-P",	   "orderbook_id": 2,	   "position_size": 1,	   "profit": 0.5,"settle_price": 70000,	   "strike_price": 65000,	   "tag": "WEEK",	   "trade_id": 1,	   "trade_size": 1,	   "underlying": "BTC_USDT",	   "time": 1639051907,	   "time_ms": 1639051907000}}`,
	"options.contracts":             `{"time": 1630576356,	"channel": "options.contracts",	"event": "update",	"result": {	   "contract": "BTC_USDT-20211130-50000-P",	   "create_time": 1637917026,	   "expiration_time": 1638230400,"init_margin_high": 0.15,	   "init_margin_low": 0.1,	   "is_call": false,	   "maint_margin_base": 0.075,	   "maker_fee_rate": 0.0004,	   "mark_price_round": 0.1,	   "min_balance_short": 0.5,	   "min_order_margin": 0.1,	   "multiplier": 0.0001,	   "order_price_deviate": 0,	   "order_price_round": 0.1,	   "order_size_max": 1,	   "order_size_min": 10,	   "orders_limit": 100000,	   "ref_discount_rate": 0.1,	   "ref_rebate_rate": 0,	   "strike_price": 50000,	   "tag": "WEEK",	   "taker_fee_rate": 0.0004,	   "underlying": "BTC_USDT",	   "time": 1639051907,	   "time_ms": 1639051907000}}`,
	"options.contract_candlesticks": `{	"time": 1630650451,	"channel": "options.contract_candlesticks",	"event": "update",	"result": [{"t": 1639039260,       "v": 100,       "c": "1041.4",       "h": "1041.4",       "l": "1041.4","o": "1041.4",       "a": "0",       "n": "10s_BTC_USDT-20211231-59800-C"   }	]}`,
	"options.ul_candlesticks":       `{"time": 1630650451,	"channel": "options.ul_candlesticks",	"event": "update",	"result": [{"t": 1639039260,        "v": 100,        "c": "1041.4",        "h": "1041.4",        "l": "1041.4","o": "1041.4",        "a": "0",        "n": "10s_BTC_USDT"    }	]}`,
	"options.book_ticker":           `{"time": 1630650452,	"channel": "options.book_ticker",	"event": "update",	"result": {    "t": 1615366379123,    "u": 2517661076,    "s": "BTC_USDT-20211130-50000-C",    "b": "54696.6",    "B": 37000,    "a": "54696.7",    "A": 47061	}}`,
	"options.order_book_update":     `{"time": 1630650445,	"channel": "options.order_book_update",	"event": "update",	"result": {    "t": 1615366381417,    "s": "BTC_USDT-20211130-50000-C",    "U": 2517661101,    "u": 2517661113,    "b": [{"p": "54672.1","s": 95        },        {            "p": "54664.5",            "s": 58794        }    ],    "a": [        {            "p": "54743.6",            "s": 95        },        {            "p": "54742",            "s": 95        }    ]	}}`,
	"options.order_book":            `{"time": 1630650445,	"channel": "options.order_book",	"event": "all",	"result": {"t": 1541500161123,    "contract": "BTC_USDT-20211130-50000-C",    "id": 93973511,    "asks": [{"p": "97.1","s": 2245},{"p": "97.2",            "s": 2245        }    ],    "bids": [		{            "p": "97.2",            "s": 2245        },        {            "p": "97.1",            "s": 2245        }    ]	}}`,
	"options.order_book-1":          `{"channel": "options.order_book",	"event": "update",	"time": 1630650445,	"result": [{"p": "49525.6",		"s": 7726,		"c": "BTC_USDT-20211130-50000-C",		"id": 93973511	  }	]}`,
	"options.book_ticker-2":         `{"time": 1630650452,	"channel": "options.book_ticker",	"event": "update",	"result": {"t": 1615366379123,    "u": 2517661076,    "s": "BTC_USDT-20211130-50000-C",    "b": "54696.6",    "B": 37000,    "a": "54696.7",    "A": 47061	}}`,
	"options.order_book_update-3":   `{"time": 1630650445,	"channel": "options.order_book_update",	"event": "update",	"result": {"t": 1615366381417,    "s": "BTC_USDT-20211130-50000-C",    "U": 2517661101,"u": 2517661113,    "b": [{"p": "54672.1","s": 95},{"p": "54664.5","s": 58794}],"a": [{"p": "54743.6","s": 95},{"p": "54742","s": 95}]}}`,
	"options.order_book-4":          `{"time": 1630650445,	"channel": "options.order_book",	"event": "all",	"result": {"t": 1541500161123,    "contract": "BTC_USDT-20211130-50000-C",    "id": 93973511,    "asks": [{"p": "97.1","s": 2245},{"p": "97.2","s": 2245}    ],    "bids": [{"p": "97.2",            "s": 2245        },        {            "p": "97.1",            "s": 2245        }    ]	}}`,
	"options.order_book-5":          `{"channel": "options.order_book",	"event": "update",	"time": 1630650445,	"result": [{"p": "49525.6",		"s": 7726,		"c": "BTC_USDT-20211130-50000-C",		"id": 93973511	  }	]}`,
	"options.liquidates":            `{"channel": "options.liquidates",	"event": "update",	"time": 1630654851,	"result": [{"user": "1xxxx",		  "init_margin": 1190,		  "maint_margin": 1042.5,		  "order_margin": 0,		  "time": 1639051907,		  "time_ms": 1639051907000}	]}`,
	"options.positions":             `{"time": 1630654851,	"channel": "options.positions",	"event": "update",	"error": null,	"result": [	   {		  "entry_price": 0,		  "realised_pnl": -13.028,		  "size": 0,		  "contract": "BTC_USDT-20211130-65000-C",		  "user": "9010",		  "time": 1639051907,		  "time_ms": 1639051907000}	]}`,
	"options.pong":                  `{"time":1756700469,"channel":"options.pong","event":"","result":null}`,
}

func TestWebsocketHandleOptionsData(t *testing.T) {
	t.Parallel()
	for k, v := range wsOptionsPushDataHandler {
		t.Run(k, func(t *testing.T) {
			t.Parallel()
			err := e.WebsocketHandleOptionsData(t.Context(), nil, []byte(v))
			require.NoError(t, err)
		})
	}
}

var wsOptionsAuthenticatedDataHandler = map[string]string{
	"options.orders":           `{"time": 1630654851,"channel": "options.orders",	"event": "update",	"result": [{"contract": "BTC_USDT-20211130-65000-C",		  "create_time": 1637897000,		  "fill_price": 0,"finish_as": "cancelled",		  "iceberg": 0,		  "id": 106,		  "is_close": false,		  "is_liq": false,		  "is_reduce_only": false,		  "left": -10,		  "mkfr": 0.0004,		  "price": 15000,		  "refr": 0,		  "refu": 0,		  "size": -10,		  "status": "finished",		  "text": "web",		  "tif": "gtc",		  "tkfr": 0.0004,		  "underlying": "BTC_USDT",		  "user": "9xxx",		  "time": 1639051907,"time_ms": 1639051907000}]}`,
	"options.usertrades":       `{"time": 1639144214,	"channel": "options.usertrades",	"event": "update",	"result": [{"id": "1","underlying": "BTC_USDT","order": "557940","contract": "BTC_USDT-20211216-44800-C","create_time": 1639144214,"create_time_ms": 1639144214583,"price": "4999","role": "taker","size": -1}]}`,
	"options.user_settlements": `{"channel": "options.user_settlements","event": "update",	"time": 1639051907,	"result": [{"contract": "BTC_USDT-20211130-65000-C","realised_pnl": -13.028,"settle_price": 70000,"settle_profit": 5,"size": 10,"strike_price": 65000,"underlying": "BTC_USDT","user": "9xxx","time": 1639051907,"time_ms": 1639051907000}]}`,
	"options.position_closes":  `{"channel": "options.position_closes","event": "update",	"time": 1630654851,	"result": [{"contract": "BTC_USDT-20211130-50000-C","pnl": -0.0056,"settle_size": 0,"side": "long","text": "web","underlying": "BTC_USDT","user": "11xxxxx","time": 1639051907,"time_ms": 1639051907000}]}`,
	"options.balances":         `{"channel": "options.balances",	"event": "update",	"time": 1630654851,	"result": [{"balance": 60.79009,"change": -0.5,"text": "BTC_USDT-20211130-55000-P","type": "set","user": "11xxxx","time": 1639051907,"time_ms": 1639051907000}]}`,
}

func TestPrivateWebsocketHandleOptionsData(t *testing.T) {
	t.Parallel()
	for k, v := range wsOptionsAuthenticatedDataHandler {
		t.Run(k, func(t *testing.T) {
			t.Parallel()
			ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "test", Secret: "test"})
			err := e.WebsocketHandleOptionsData(ctx, nil, []byte(v))
			require.NoError(t, err)
		})
	}
}

func TestGenerateSubscriptionsSpot(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	subs, err := e.generateSubscriptionsSpot()
	require.NoError(t, err, "generateSubscriptions must not error")
	var spotV2Count int
	for _, sub := range subs {
		if sub.Asset == asset.Spot && sub.Channel == subscription.OrderbookChannel {
			assert.Fail(t, "legacy spot orderbook subscription should be disabled")
		}
		if sub.Asset == asset.Spot && sub.Channel == spotOrderbookV2 {
			spotV2Count++
			assert.Equal(t, 50, sub.Levels, "V2 spot orderbook subscription should request 50 levels")
		}
	}
	spotPairs, err := e.GetEnabledPairs(asset.Spot)
	require.NoError(t, err, "GetEnabledPairs must not error")
	assert.Equal(t, len(spotPairs), spotV2Count, "each enabled spot pair should generate one V2 orderbook subscription")
	exp := subscription.List{}
	assets := slices.DeleteFunc(e.GetAssetTypes(true), func(a asset.Item) bool { return !e.IsAssetWebsocketSupported(a) })
	for _, s := range e.Features.Subscriptions {
		for _, a := range assets {
			if s.Asset != asset.All && s.Asset != a {
				continue
			}
			pairs, err := e.GetEnabledPairs(a)
			require.NoErrorf(t, err, "GetEnabledPairs %s must not error", a)
			pairs = common.SortStrings(pairs).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"})
			s := s.Clone() //nolint:govet // Intentional lexical scope shadow
			s.Asset = a
			if singleSymbolChannel(channelName(s)) {
				for i := range pairs {
					s := s.Clone() //nolint:govet // Intentional lexical scope shadow
					switch s.Channel {
					case subscription.CandlesChannel:
						s.QualifiedChannel = "5m," + pairs[i].String()
					case subscription.OrderbookChannel:
						s.QualifiedChannel = pairs[i].String() + ",100ms"
					case spotOrderbookChannel:
						s.QualifiedChannel = pairs[i].String() + ",5,1000ms"
					case spotOrderbookV2:
						s.QualifiedChannel = fmt.Sprintf("ob.%s.%d", pairs[i].String(), s.Levels)
					}
					s.Pairs = pairs[i : i+1]
					exp = append(exp, s)
				}
			} else {
				s.Pairs = pairs
				s.QualifiedChannel = pairs.Join()
				exp = append(exp, s)
			}
		}
	}
	testsubs.EqualLists(t, exp, subs)
}

func TestSubscribe(t *testing.T) {
	t.Parallel()
	subs, err := e.Features.Subscriptions.ExpandTemplates(e)
	require.NoError(t, err, "ExpandTemplates must not error")
	e.Features.Subscriptions = subscription.List{}
	err = e.Subscribe(t.Context(), &FixtureConnection{}, subs)
	require.NoError(t, err, "Subscribe must not error")
}

func TestGenerateDeliveryFuturesDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	_, err := e.GenerateDeliveryFuturesDefaultSubscriptions(asset.DeliveryFutures)
	assert.NoError(t, err)
}

func TestGenerateFuturesDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	_, err := e.GenerateFuturesDefaultSubscriptions(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "Options must not use the perpetual futures subscription generator")
	subs, err := e.GenerateFuturesDefaultSubscriptions(t.Context(), asset.USDTMarginedFutures)
	require.NoError(t, err)
	require.NotEmpty(t, subs)
	var orderbooks int
	for _, sub := range subs {
		if sub.Channel != futuresOrderbookV2 {
			continue
		}
		orderbooks++
		assert.Equal(t, "ob."+sub.Pairs[0].String()+".50", sub.QualifiedChannel, "V2 recovery key should identify the pair and depth")
		for _, event := range []string{subscribeEvent, unsubscribeEvent} {
			payload, err := e.generateFuturesPayload(t.Context(), event, subscription.List{sub})
			require.NoError(t, err, "V2 payload generation must succeed")
			require.Len(t, payload, 1, "V2 subscription must generate one request")
			assert.Equal(t, []string{sub.QualifiedChannel}, payload[0].Payload, "V2 payload should match the recovery key")
		}
	}
	require.Positive(t, orderbooks, "futures defaults must include V2 orderbooks")
	subs, err = e.GenerateFuturesDefaultSubscriptions(t.Context(), asset.CoinMarginedFutures)
	require.NoError(t, err)
	require.NotEmpty(t, subs)
	var accountRequests atomic.Int64
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		accountRequests.Add(1)
		if _, err := w.Write([]byte(`{"user":20011}`)); err != nil {
			t.Errorf("Mock futures account response should be written: %v", err)
		}
	}))
	require.NoError(t, e.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	for endpoint := range e.API.Endpoints.GetURLMap() {
		require.NoError(t, e.API.Endpoints.SetRunningURL(endpoint, server.URL+"/"), "SetRunningURL must not error")
	}
	e.API.AuthenticatedSupport = true
	e.API.AuthenticatedWebsocketSupport = true
	e.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	e.setFuturesUserID("key", currency.USDT, "20011")
	continueBootstrap, err := e.Bootstrap(t.Context())
	require.NoError(t, err, "Bootstrap must not error")
	assert.True(t, continueBootstrap, "Bootstrap should continue with common startup actions")
	assert.Equal(t, int64(1), accountRequests.Load(), "Bootstrap should request the uncached futures account ID")
	subs, err = e.GenerateFuturesDefaultSubscriptions(t.Context(), asset.CoinMarginedFutures)
	require.NoError(t, err)
	assert.Equal(t, int64(1), accountRequests.Load(), "Subscription generation should not perform another account request")
	require.NoError(t, e.Websocket.Disable(), "Websocket.Disable must not error")
	e.SetCredentials(&accounts.Credentials{Key: "rotated-key", Secret: "secret"})
	continueBootstrap, err = e.Bootstrap(t.Context())
	require.NoError(t, err, "Bootstrap must not error")
	assert.True(t, continueBootstrap, "Bootstrap should continue with common startup actions")
	assert.Equal(t, int64(1), accountRequests.Load(), "Bootstrap should not request futures account IDs when websocket support is disabled")
	authenticatedChannels := map[string]bool{
		futuresOrdersChannel:            false,
		futuresUserTradesChannel:        false,
		futuresLiquidatesChannel:        false,
		futuresAutoDeleveragesChannel:   false,
		futuresBalancesChannel:          false,
		futuresReduceRiskLimitsChannel:  false,
		futuresPositionsChannel:         false,
		futuresAutoPositionCloseChannel: false,
		futuresAutoOrdersChannel:        false,
	}
	accountSubscriptions := make(map[string]*subscription.Subscription)
	for _, sub := range subs {
		if _, ok := authenticatedChannels[sub.Channel]; ok {
			authenticatedChannels[sub.Channel] = true
			accountSubscriptions[sub.Channel] = sub
		}
	}
	for channel, found := range authenticatedChannels {
		assert.Truef(t, found, "%s should be generated for authenticated futures", channel)
	}
	require.Len(t, accountSubscriptions, len(authenticatedChannels),
		"authenticated futures channels must each have one account-wide subscription")
	positionsSubscription := accountSubscriptions[futuresPositionsChannel]
	require.NotNil(t, positionsSubscription, "positions subscription must be generated")
	require.Len(t, positionsSubscription.Pairs, 1, "account-wide subscription must retain one identity pair")
	assert.NotEqual(t, allFuturesContracts, positionsSubscription.Pairs[0].String(),
		"account-wide subscription should not register the selector as a currency")
	assert.Equal(t, allFuturesContracts, positionsSubscription.Params[contractPayloadOverrideParam],
		"account-wide subscription should serialise the documented selector")
	requiresUserPlaceholder, ok := positionsSubscription.Params[requiresUserPlaceholderParam].(bool)
	require.True(t, ok, "user placeholder parameter must be a boolean")
	assert.True(t, requiresUserPlaceholder,
		"positions subscription should retain the deprecated user placeholder")
	require.Equal(t, asset.CoinMarginedFutures, positionsSubscription.Asset,
		"account-wide subscription must retain its futures asset")
	closeSubscription := accountSubscriptions[futuresAutoPositionCloseChannel]
	require.NotNil(t, closeSubscription, "position closes subscription must be generated")
	assert.Equal(t, "20011", closeSubscription.Params["user"],
		"position closes subscription should use the futures account user ID")
	accountSubscriptionList := make(subscription.List, 0, len(accountSubscriptions))
	for _, sub := range accountSubscriptions {
		accountSubscriptionList = append(accountSubscriptionList, sub)
	}
	payloads, err := e.generateFuturesPayload(t.Context(), subscribeEvent, accountSubscriptionList)
	require.NoError(t, err, "generateFuturesPayload must generate account-wide payloads")
	require.Len(t, payloads, len(authenticatedChannels),
		"account-wide subscriptions must each generate one payload")
	expectedPayloads := map[string][]string{
		futuresOrdersChannel:            {"20011", allFuturesContracts},
		futuresUserTradesChannel:        {"20011", allFuturesContracts},
		futuresLiquidatesChannel:        {"20011", allFuturesContracts},
		futuresAutoDeleveragesChannel:   {"20011", allFuturesContracts},
		futuresAutoPositionCloseChannel: {"20011", allFuturesContracts},
		futuresBalancesChannel:          {"20011"},
		futuresReduceRiskLimitsChannel:  {"20011", allFuturesContracts},
		futuresPositionsChannel:         {"", allFuturesContracts},
		futuresAutoOrdersChannel:        {allFuturesContracts},
	}
	for i := range payloads {
		assert.Equalf(t, expectedPayloads[payloads[i].Channel], payloads[i].Payload,
			"%s payload should match Gate's documented parameters", payloads[i].Channel)
		require.NotNilf(t, payloads[i].Auth, "%s payload must be authenticated", payloads[i].Channel)
	}
	require.NoError(t, e.CurrencyPairs.StorePairs(asset.CoinMarginedFutures, nil, true),
		"StorePairs must clear enabled coin-margined futures pairs")
	subs, err = e.GenerateFuturesDefaultSubscriptions(t.Context(), asset.CoinMarginedFutures)
	require.NoError(t, err, "Enabled asset without pairs must not tear down other Gate connections")
	require.Empty(t, subs, "Enabled asset without pairs must return no subscriptions")
	require.NoError(t, e.CurrencyPairs.SetAssetEnabled(asset.USDTMarginedFutures, false), "SetAssetEnabled must not error")
	subs, err = e.GenerateFuturesDefaultSubscriptions(t.Context(), asset.USDTMarginedFutures)
	require.NoError(t, err, "Disabled asset must not error")
	require.Empty(t, subs, "Disabled asset must return no pairs")
}

func TestGenerateFuturesDefaultSubscriptionsAccountIDCache(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`{"user":20011}`))
		assert.NoError(t, err, "Mock futures account response should be written")
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	for endpoint := range ex.API.Endpoints.GetURLMap() {
		require.NoError(t, ex.API.Endpoints.SetRunningURL(endpoint, server.URL+"/"), "SetRunningURL must not error")
	}
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)

	err := ex.cacheFuturesUserID(t.Context(), "key", currency.USDT)
	require.NoError(t, err, "Initial account lookup must populate the futures user ID cache")
	subs, err := ex.GenerateFuturesDefaultSubscriptions(t.Context(), asset.USDTMarginedFutures)
	require.NoError(t, err, "Subscription generation must reuse the matching cached user ID")
	privateChannels := map[string]bool{
		futuresOrdersChannel:            false,
		futuresUserTradesChannel:        false,
		futuresLiquidatesChannel:        false,
		futuresAutoDeleveragesChannel:   false,
		futuresAutoPositionCloseChannel: false,
		futuresBalancesChannel:          false,
		futuresReduceRiskLimitsChannel:  false,
		futuresPositionsChannel:         false,
		futuresAutoOrdersChannel:        false,
	}
	for _, sub := range subs {
		if _, ok := privateChannels[sub.Channel]; ok {
			privateChannels[sub.Channel] = true
		}
	}
	for channel, generated := range privateChannels {
		assert.Truef(t, generated, "%s should remain generated from the cached account ID", channel)
	}

	_, err = ex.GenerateFuturesDefaultSubscriptions(t.Context(), asset.CoinMarginedFutures)
	require.NoError(t, err, "A user ID cached for another settlement account must not affect generation")
	ex.SetCredentials(&accounts.Credentials{Key: "rotated-key", Secret: "secret"})
	_, err = ex.GenerateFuturesDefaultSubscriptions(t.Context(), asset.USDTMarginedFutures)
	require.NoError(t, err, "A user ID cached for another API key must not affect generation")
}

func TestGenerateFuturesDefaultSubscriptionsColdCache(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
	var accountRequests atomic.Int64
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		accountRequests.Add(1)
		_, err := w.Write([]byte(`{"user":20011}`))
		assert.NoError(t, err, "Mock futures account response should be written")
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	for endpoint := range ex.API.Endpoints.GetURLMap() {
		require.NoError(t, ex.API.Endpoints.SetRunningURL(endpoint, server.URL+"/"), "SetRunningURL must not error")
	}
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)

	subs, err := ex.GenerateFuturesDefaultSubscriptions(t.Context(), asset.CoinMarginedFutures)
	require.NoError(t, err, "Missing cached account ID must not prevent futures subscription generation")
	assert.Zero(t, accountRequests.Load(), "subscription generation should not perform an account request")

	authenticatedChannels := map[string]bool{
		futuresOrdersChannel:            false,
		futuresUserTradesChannel:        false,
		futuresLiquidatesChannel:        false,
		futuresAutoDeleveragesChannel:   false,
		futuresAutoPositionCloseChannel: false,
		futuresBalancesChannel:          false,
		futuresReduceRiskLimitsChannel:  false,
		futuresPositionsChannel:         false,
		futuresAutoOrdersChannel:        false,
	}
	for _, sub := range subs {
		if _, ok := authenticatedChannels[sub.Channel]; ok {
			authenticatedChannels[sub.Channel] = true
		}
	}
	assert.True(t, authenticatedChannels[futuresPositionsChannel], "Positions subscription should not require a user ID")
	assert.True(t, authenticatedChannels[futuresAutoOrdersChannel], "Auto-orders subscription should not require a user ID")
	for channel, generated := range authenticatedChannels {
		if channel == futuresPositionsChannel || channel == futuresAutoOrdersChannel {
			continue
		}
		assert.Falsef(t, generated, "%s should be skipped without a valid user ID", channel)
	}
}

func TestGenerateFuturesDefaultSubscriptionsMissingCredentials(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)

	subs, err := ex.GenerateFuturesDefaultSubscriptions(t.Context(), asset.CoinMarginedFutures)
	require.NoError(t, err, "Missing credentials must degrade to public futures subscriptions")
	assert.False(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "authenticated websocket support should be disabled")
	for _, sub := range subs {
		assert.NotContains(t, []string{
			futuresOrdersChannel,
			futuresUserTradesChannel,
			futuresLiquidatesChannel,
			futuresAutoDeleveragesChannel,
			futuresAutoPositionCloseChannel,
			futuresBalancesChannel,
			futuresReduceRiskLimitsChannel,
			futuresPositionsChannel,
			futuresAutoOrdersChannel,
		}, sub.Channel, "private channel should not be generated without credentials")
	}
}

func TestPreConnectWiring(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")

	var accountRequests atomic.Int64
	var healthy atomic.Bool
	rest := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		accountRequests.Add(1)
		if !healthy.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err := w.Write([]byte(`{"user":20011}`))
		assert.NoError(t, err, "Mock futures account response should be written")
	}))
	require.NoError(t, ex.SetHTTPClient(rest.Client()), "SetHTTPClient must not error")
	for endpoint := range ex.API.Endpoints.GetURLMap() {
		require.NoError(t, ex.API.Endpoints.SetRunningURL(endpoint, rest.URL+"/"), "SetRunningURL must not error")
	}

	// Refuses the upgrade, so Connect fails locally after preparation has run.
	ws := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	ws.Start()
	require.NoError(t, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(ws.URL, "http")), "SetAllConnectionURLs must not error")

	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)

	require.Error(t, ex.Websocket.Connect(t.Context()), "Connect must fail against the rejecting mock")
	assert.Equal(t, int64(2), accountRequests.Load(), "Connect should run futures account preparation for both settlement accounts")
	assert.Empty(t, ex.getFuturesUserID("key", currency.USDT), "a failed lookup should leave the cache cold")

	healthy.Store(true)
	require.Error(t, ex.Websocket.Connect(t.Context()), "Connect must fail against the rejecting mock")
	assert.Equal(t, int64(4), accountRequests.Load(), "a later connection attempt should retry the failed lookup")
	assert.Equal(t, "20011", ex.getFuturesUserID("key", currency.USDT), "a successful retry should populate the cache")

	require.Error(t, ex.Websocket.Connect(t.Context()), "Connect must fail against the rejecting mock")
	assert.Equal(t, int64(4), accountRequests.Load(), "a warm cache should not perform further account requests")
}

func TestPrepareFuturesUserIDsLookupFailure(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
	var accountRequests atomic.Int64
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		accountRequests.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	for endpoint := range ex.API.Endpoints.GetURLMap() {
		require.NoError(t, ex.API.Endpoints.SetRunningURL(endpoint, server.URL+"/"), "SetRunningURL must not error")
	}
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)

	ex.prepareFuturesUserIDs(t.Context())

	assert.Equal(t, int64(2), accountRequests.Load(), "preparation should try each enabled futures settlement account")
	assert.Empty(t, ex.getFuturesUserID("key", currency.USDT), "failed preparation should leave the USDT account ID uncached")
	assert.Empty(t, ex.getFuturesUserID("key", currency.BTC), "failed preparation should leave the BTC account ID uncached")
}

func TestGenerateDeliveryFuturesDefaultSubscriptionsSkipsUnsupportedAsset(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	ex.SetDefaults()

	got, err := ex.GenerateDeliveryFuturesDefaultSubscriptions(asset.Futures)
	require.NoError(t, err, "GenerateDeliveryFuturesDefaultSubscriptions must not error for unsupported delivery asset")
	require.Empty(t, got, "unsupported delivery asset must not generate subscriptions")
}

func TestGenerateOptionsDefaultSubscriptionsSkipsDisabledAssetBeforeAuth(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	ex.SetDefaults()
	ex.API.AuthenticatedWebsocketSupport = true
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)

	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Options, false), "SetAssetEnabled must not error")
	got, err := ex.GenerateOptionsDefaultSubscriptions()
	require.NoError(t, err, "GenerateOptionsDefaultSubscriptions must not error for disabled asset")
	require.Empty(t, got, "disabled options asset must not generate subscriptions")
	require.True(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "authenticated endpoints must not be changed when disabled options are skipped before auth")
}

func TestParseGateioMilliSecTimeUnmarshal(t *testing.T) {
	t.Parallel()
	var timeWhenTesting int64 = 1684981731098
	timeWhenTestingString := `"1684981731098"` // Normal string
	integerJSON := `{"number": 1684981731098}`
	float64JSON := `{"number": 1684981731.098}`

	tm := time.UnixMilli(timeWhenTesting)
	var in types.Time
	err := json.Unmarshal([]byte(timeWhenTestingString), &in)
	require.NoError(t, err)
	require.Equalf(t, tm, in.Time(), "found %v, but expected %v", in.Time(), tm)

	var inInteger struct {
		Number types.Time `json:"number"`
	}
	err = json.Unmarshal([]byte(integerJSON), &inInteger)
	require.NoError(t, err)
	require.Equalf(t, tm, inInteger.Number.Time(), "found %v, but expected %v", inInteger.Number.Time(), tm)

	var inFloat64 struct {
		Number types.Time `json:"number"`
	}
	err = json.Unmarshal([]byte(float64JSON), &inFloat64)
	require.NoError(t, err)
	require.Equalf(t, tm, inFloat64.Number.Time(), "found %v, but expected %v", inFloat64.Number.Time(), tm)
}

func TestParseTimeUnmarshal(t *testing.T) {
	t.Parallel()
	var timeWhenTesting int64 = 1684981731
	timeWhenTestingString := `"1684981731"`
	integerJSON := `{"number": 1684981731}`
	float64JSON := `{"number": 1684981731.234}`
	timeWhenTestingStringMicroSecond := `"1691122380942.173000"`

	whenTime := time.Unix(timeWhenTesting, 0)
	var in types.Time
	err := json.Unmarshal([]byte(timeWhenTestingString), &in)
	require.NoError(t, err)
	require.Equalf(t, whenTime, in.Time(), "found %v, but expected %v", in.Time(), whenTime)

	var inInteger struct {
		Number types.Time `json:"number"`
	}
	err = json.Unmarshal([]byte(integerJSON), &inInteger)
	require.NoError(t, err)
	require.Equalf(t, whenTime, inInteger.Number.Time(), "found %v, but expected %v", inInteger.Number.Time(), whenTime)

	var inFloat64 struct {
		Number types.Time `json:"number"`
	}
	err = json.Unmarshal([]byte(float64JSON), &inFloat64)
	require.NoError(t, err)

	msTime := time.UnixMilli(1684981731234)
	require.Equalf(t, time.UnixMilli(1684981731234), inFloat64.Number.Time(), "found %v, but expected %v", inFloat64.Number.Time(), msTime)

	var microSeconds types.Time
	err = json.Unmarshal([]byte(timeWhenTestingStringMicroSecond), &microSeconds)
	require.NoError(t, err)
	require.Equalf(t, time.UnixMicro(1691122380942173), microSeconds.Time(), "found %v, but expected %v", microSeconds.Time(), time.UnixMicro(1691122380942173))
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	ex := newIsolatedExchange(t)
	require.ErrorIs(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.Binary), asset.ErrNotSupported, "UpdateOrderExecutionLimits must reject an unsupported asset")

	if mockTests {
		for _, tc := range []struct {
			a   asset.Item
			exp limits.MinMaxLevel
		}{
			{
				// The earlier of the buy and sell start times is the listing time
				a: asset.Spot,
				exp: limits.MinMaxLevel{
					PriceStepIncrementSize:  0.00001,
					MultiplierUp:            0.4,
					MultiplierDown:          0.4,
					MinimumBaseAmount:       0.01,
					MinimumQuoteAmount:      3,
					MaximumQuoteAmount:      5000000,
					AmountStepIncrementSize: 0.01,
					MarketMaxQty:            7228044.81,
					Listed:                  time.Unix(1608264000, 0),
				},
			},
			{
				// Margin limits are the spot limits with the market's own borrow minimums
				a: asset.Margin,
				exp: limits.MinMaxLevel{
					PriceStepIncrementSize:   0.00001,
					MinimumBaseAmount:        0.01,
					MinimumQuoteAmount:       3,
					AmountStepIncrementSize:  0.01,
					MinimumBorrowAmountBase:  0.00001,
					MinimumBorrowAmountQuote: 1,
				},
			},
			{
				// Cross margin borrow minimums are those of each currency
				a: asset.CrossMargin,
				exp: limits.MinMaxLevel{
					PriceStepIncrementSize:   0.00001,
					MinimumBaseAmount:        0.01,
					MinimumQuoteAmount:       3,
					AmountStepIncrementSize:  0.01,
					MinimumBorrowAmountBase:  0.01,
					MinimumBorrowAmountQuote: 3,
				},
			},
			{
				// Futures amounts count whole contracts
				a: asset.USDTMarginedFutures,
				exp: limits.MinMaxLevel{
					PriceStepIncrementSize:  0.01,
					PriceDivisor:            1,
					MultiplierDecimal:       0.01,
					MinimumBaseAmount:       1,
					MaximumBaseAmount:       32000,
					AmountStepIncrementSize: 1,
					Listed:                  time.Unix(1769416200, 0),
				},
			},
			{
				a: asset.CoinMarginedFutures,
				exp: limits.MinMaxLevel{
					PriceStepIncrementSize:  0.1,
					PriceDivisor:            1,
					MinimumBaseAmount:       1,
					MaximumBaseAmount:       600000,
					AmountStepIncrementSize: 1,
					Listed:                  time.Unix(1545235200, 0),
				},
			},
			{
				a: asset.DeliveryFutures,
				exp: limits.MinMaxLevel{
					PriceStepIncrementSize:  0.00001,
					PriceDivisor:            1,
					MultiplierDecimal:       10,
					MinimumBaseAmount:       1,
					MaximumBaseAmount:       1000000,
					AmountStepIncrementSize: 1,
					Expiry:                  time.Unix(1775808000, 0),
				},
			},
			{
				a: asset.Options,
				exp: limits.MinMaxLevel{
					PriceStepIncrementSize:  1,
					PriceDivisor:            1,
					MultiplierDecimal:       0.01,
					MinimumBaseAmount:       1,
					MaximumBaseAmount:       10000,
					AmountStepIncrementSize: 1,
				},
			},
		} {
			t.Run(tc.a.String(), func(t *testing.T) {
				t.Parallel()
				started := time.Now()
				require.NoError(t, ex.UpdateOrderExecutionLimits(t.Context(), tc.a), "UpdateOrderExecutionLimits must not error")
				result, err := ex.GetOrderExecutionLimits(tc.a, getPair(t, tc.a))
				require.NoError(t, err, "GetOrderExecutionLimits must not error")
				assert.WithinRange(t, result.UpdatedAt, started, time.Now(), "UpdateOrderExecutionLimits should stamp the limits when loading them")
				tc.exp.UpdatedAt = result.UpdatedAt
				tc.exp.Key = key.NewExchangeAssetPair(ex.Name, tc.a, getPair(t, tc.a))
				assert.Equal(t, tc.exp, result, "UpdateOrderExecutionLimits should load the pair's limits")
			})
		}
		return
	}
	for _, a := range ex.GetAssetTypes(false) {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			require.NoError(t, ex.UpdateOrderExecutionLimits(t.Context(), a), "UpdateOrderExecutionLimits must not error")
			pairs, err := ex.GetAvailablePairs(a)
			require.NoError(t, err, "GetAvailablePairs must not error")
			for _, p := range pairs {
				l, err := ex.GetOrderExecutionLimits(a, p)
				require.NoErrorf(t, err, "GetOrderExecutionLimits must not error for %s", p)
				require.NotNilf(t, l, "GetOrderExecutionLimits must return limits for %s", p)
				assert.Equalf(t, a, l.Key.Asset, "asset should equal for %s", p)
				assert.Truef(t, p.Equal(l.Key.Pair()), "pair should equal for %s", p)
				switch a {
				case asset.Options:
					assert.Positivef(t, l.MinimumBaseAmount, "MinimumBaseAmount should be positive for %s", p)
					assert.Positivef(t, l.MaximumBaseAmount, "MaximumBaseAmount should be positive for %s", p)
					assert.Positivef(t, l.PriceStepIncrementSize, "PriceStepIncrementSize should be positive for %s", p)
					assert.Positivef(t, l.AmountStepIncrementSize, "AmountStepIncrementSize should be positive for %s", p)
					assert.Positivef(t, l.MultiplierDecimal, "MultiplierDecimal should be positive for %s", p)
				case asset.USDTMarginedFutures:
					assert.Positivef(t, l.MultiplierDecimal, "MultiplierDecimal should be positive for %s", p)
					assert.NotZerof(t, l.Listed, "Listed should be populated for %s", p)
					fallthrough
				case asset.CoinMarginedFutures:
					if !l.Delisted.IsZero() {
						assert.Truef(t, l.Delisted.After(l.Delisting), "Delisted should be after Delisting for %s", p)
					}
					assert.Positivef(t, l.AmountStepIncrementSize, "AmountStepIncrementSize should be positive for %s", p)
				case asset.Spot:
					assert.Positivef(t, l.MinimumQuoteAmount, "MinimumQuoteAmount should be positive for %s", p)
					assert.Positivef(t, l.PriceStepIncrementSize, "PriceStepIncrementSize should be positive for %s", p)
					assert.Positivef(t, l.MinimumBaseAmount, "MinimumBaseAmount should be positive for %s", p)
					assert.Positivef(t, l.AmountStepIncrementSize, "AmountStepIncrementSize should be positive for %s", p)
				case asset.Margin, asset.CrossMargin:
					assert.Positivef(t, l.MinimumQuoteAmount, "MinimumQuoteAmount should be positive for %s", p)
					assert.Positivef(t, l.MinimumBaseAmount, "MinimumBaseAmount should be positive for %s", p)
					assert.Positivef(t, l.PriceStepIncrementSize, "PriceStepIncrementSize should be positive for %s", p)
					invalidPrice := l.PriceStepIncrementSize / 2
					assert.ErrorIsf(t, l.Validate(invalidPrice, l.MinimumBaseAmount, order.Limit), limits.ErrPriceExceedsStep, "Validate should reject an invalid price tick for %s", p)
					if l.AmountStepIncrementSize != 0 {
						assert.Positivef(t, l.AmountStepIncrementSize, "AmountStepIncrementSize should be positive for %s when set", p)
					}
					assert.Positivef(t, l.MinimumBorrowAmountBase, "MinimumBorrowAmountBase should be positive for %s", p)
					assert.Positivef(t, l.MinimumBorrowAmountQuote, "MinimumBorrowAmountQuote should be positive for %s", p)
				case asset.DeliveryFutures:
					assert.NotZerof(t, l.Expiry, "Expiry should be populated for %s", p)
					assert.Positivef(t, l.MinimumBaseAmount, "MinimumBaseAmount should be positive for %s", p)
					assert.Positivef(t, l.AmountStepIncrementSize, "AmountStepIncrementSize should be positive for %s", p)
					assert.Positivef(t, l.MultiplierDecimal, "MultiplierDecimal should be positive for %s", p)
				}
			}
		})
	}
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset, "GetFuturesContractDetails must reject a spot asset")

	_, err = e.GetFuturesContractDetails(t.Context(), asset.PerpetualContract)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetFuturesContractDetails must reject an unsupported futures asset")

	for _, tc := range []struct {
		a     asset.Item
		count int
		exp   futures.Contract
	}{
		{
			// The latest rate is the one applying since the previous funding time
			a:     asset.USDTMarginedFutures,
			count: 6,
			exp: futures.Contract{
				Exchange:           e.Name,
				Name:               getPair(t, asset.USDTMarginedFutures),
				Underlying:         getPair(t, asset.USDTMarginedFutures),
				Asset:              asset.USDTMarginedFutures,
				IsActive:           true,
				Type:               futures.Perpetual,
				SettlementType:     futures.Linear,
				SettlementCurrency: currency.USDT,
				Multiplier:         0.01,
				MaxLeverage:        25,
				LatestRate:         fundingrate.Rate{Time: time.Unix(1775404800, 0), Rate: decimal.MustFromFloat(0)},
			},
		},
		{
			a:     asset.CoinMarginedFutures,
			count: 1,
			exp: futures.Contract{
				Exchange:           e.Name,
				Name:               getPair(t, asset.CoinMarginedFutures),
				Underlying:         getPair(t, asset.CoinMarginedFutures),
				Asset:              asset.CoinMarginedFutures,
				IsActive:           true,
				Type:               futures.Perpetual,
				SettlementType:     futures.Inverse,
				SettlementCurrency: currency.BTC,
				MaxLeverage:        100,
				LatestRate:         fundingrate.Rate{Time: time.Unix(1775404800, 0), Rate: decimal.MustFromFloat(0.0001)},
			},
		},
		{
			// A bi-weekly contract is taken to start two weeks before it expires
			a:     asset.DeliveryFutures,
			count: 6,
			exp: futures.Contract{
				Exchange:           e.Name,
				Name:               getPair(t, asset.DeliveryFutures),
				Underlying:         currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter),
				Asset:              asset.DeliveryFutures,
				StartDate:          time.Unix(1774598400, 0),
				EndDate:            time.Unix(1775808000, 0),
				IsActive:           true,
				Type:               futures.Fortnightly,
				SettlementType:     futures.Linear,
				SettlementCurrency: currency.USDT,
				Multiplier:         10,
				MaxLeverage:        50,
			},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			result, err := e.GetFuturesContractDetails(t.Context(), tc.a)
			require.NoError(t, err, "GetFuturesContractDetails must not error")
			if !mockTests {
				assert.NotEmpty(t, result, "GetFuturesContractDetails should return contracts")
				return
			}
			require.Len(t, result, tc.count, "GetFuturesContractDetails must return every recorded contract")
			assert.Equal(t, tc.exp, result[0], "GetFuturesContractDetails should return every field of the contract")
		})
	}
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetLatestFundingRates(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLatestFundingRates must reject a nil request")
	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.DeliveryFutures})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetLatestFundingRates must reject an asset without funding rates")

	usdtPair := currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter)
	coinPair := currency.NewPairWithDelimiter("BTC", "USD", currency.UnderscoreDelimiter)
	// Each contract's rate applies from one funding interval before its next funding time
	rate := func(a asset.Item, p currency.Pair, r float64) fundingrate.LatestRateResponse {
		return fundingrate.LatestRateResponse{
			Exchange:       e.Name,
			Asset:          a,
			Pair:           p,
			LatestRate:     fundingrate.Rate{Time: time.Unix(1775404800, 0), Rate: decimal.MustFromFloat(r)},
			TimeOfNextRate: time.Unix(1775433600, 0),
		}
	}
	predicted := rate(asset.USDTMarginedFutures, usdtPair, 0.000083)
	predicted.PredictedUpcomingRate = fundingrate.Rate{Time: time.Unix(1775433600, 0), Rate: decimal.MustFromFloat(0.000083)}
	for _, tc := range []struct {
		name string
		req  *fundingrate.LatestRateRequest
		exp  []fundingrate.LatestRateResponse
	}{
		{
			name: "USDT-M pair with the predicted rate",
			req:  &fundingrate.LatestRateRequest{Asset: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), IncludePredictedRate: true},
			exp:  []fundingrate.LatestRateResponse{predicted},
		},
		{
			name: "COIN-M pair",
			req:  &fundingrate.LatestRateRequest{Asset: asset.CoinMarginedFutures, Pair: currency.NewBTCUSD()},
			exp:  []fundingrate.LatestRateResponse{rate(asset.CoinMarginedFutures, coinPair, 0.0001)},
		},
		{
			// Contracts that are not enabled are left out
			name: "COIN-M enabled pairs",
			req:  &fundingrate.LatestRateRequest{Asset: asset.CoinMarginedFutures},
			exp:  []fundingrate.LatestRateResponse{rate(asset.CoinMarginedFutures, coinPair, 0.0001)},
		},
		{
			// The recorded contract list holds a rate of 0 for the pair
			name: "USDT-M enabled pairs",
			req:  &fundingrate.LatestRateRequest{Asset: asset.USDTMarginedFutures},
			exp:  []fundingrate.LatestRateResponse{rate(asset.USDTMarginedFutures, usdtPair, 0)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			started := time.Now()
			result, err := e.GetLatestFundingRates(t.Context(), tc.req)
			require.NoError(t, err, "GetLatestFundingRates must not error")
			if !mockTests {
				assert.NotEmpty(t, result, "GetLatestFundingRates should return rates")
				return
			}
			require.Len(t, result, len(tc.exp), "GetLatestFundingRates must return a rate for every contract")
			for i := range result {
				assert.WithinRange(t, result[i].TimeChecked, started, time.Now(), "GetLatestFundingRates should stamp the time each rate is checked")
				tc.exp[i].TimeChecked = result[i].TimeChecked
			}
			assert.Equal(t, tc.exp, result, "GetLatestFundingRates should return every field of the rates")
		})
	}
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricalFundingRates(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetHistoricalFundingRates must reject a nil request")

	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricalFundingRates must reject an unset asset")

	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.CoinMarginedFutures})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetHistoricalFundingRates must reject an empty pair")

	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.Futures})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricalFundingRates must reject an asset without funding rates")

	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset: asset.USDTMarginedFutures,
		Pair:  getPair(t, asset.USDTMarginedFutures),
	})
	require.ErrorIs(t, err, fundingrate.ErrPaymentCurrencyCannotBeEmpty, "GetHistoricalFundingRates must reject an empty payment currency")

	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:           asset.USDTMarginedFutures,
		Pair:            getPair(t, asset.USDTMarginedFutures),
		PaymentCurrency: currency.USDT,
		IncludePayments: true,
	})
	require.ErrorIs(t, err, common.ErrNotYetImplemented, "GetHistoricalFundingRates must reject a request for payments")

	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:                asset.USDTMarginedFutures,
		Pair:                 getPair(t, asset.USDTMarginedFutures),
		PaymentCurrency:      currency.USDT,
		IncludePredictedRate: true,
	})
	require.ErrorIs(t, err, common.ErrNotYetImplemented, "GetHistoricalFundingRates must reject a request for the predicted rate")

	startTime, endTime := getTime()
	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:           asset.USDTMarginedFutures,
		Pair:            getPair(t, asset.USDTMarginedFutures),
		PaymentCurrency: currency.USDT,
		StartDate:       endTime,
		EndDate:         startTime,
	})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetHistoricalFundingRates must reject a start after the end")

	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:           asset.USDTMarginedFutures,
		Pair:            getPair(t, asset.USDTMarginedFutures),
		PaymentCurrency: currency.USDT,
		StartDate:       time.Now().Add(-time.Hour * 8008),
		EndDate:         time.Now(),
	})
	require.ErrorIs(t, err, fundingrate.ErrFundingRateOutsideLimits, "GetHistoricalFundingRates must reject a start before the rates Gate returns")

	result, err := e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:           asset.USDTMarginedFutures,
		Pair:            getPair(t, asset.USDTMarginedFutures),
		PaymentCurrency: currency.USDT,
	})
	require.NoError(t, err, "GetHistoricalFundingRates must not error")
	if !mockTests {
		assert.NotEmpty(t, result.FundingRates, "GetHistoricalFundingRates should return rates")
		return
	}
	// Gate lists the latest rate first
	exp := &fundingrate.HistoricalRates{
		Exchange:   e.Name,
		Asset:      asset.USDTMarginedFutures,
		Pair:       getPair(t, asset.USDTMarginedFutures),
		StartDate:  time.Unix(1774954800, 0),
		EndDate:    time.Unix(1774969200, 0),
		LatestRate: fundingrate.Rate{Time: time.Unix(1774969200, 0), Rate: decimal.MustFromFloat(-0.000449)},
		FundingRates: []fundingrate.Rate{
			{Time: time.Unix(1774969200, 0), Rate: decimal.MustFromFloat(-0.000449)},
			{Time: time.Unix(1774965600, 0), Rate: decimal.MustFromFloat(0.000012)},
			{Time: time.Unix(1774962000, 0), Rate: decimal.MustFromFloat(-0.000236)},
			{Time: time.Unix(1774958401, 0), Rate: decimal.MustFromFloat(0.000012)},
			{Time: time.Unix(1774954800, 0), Rate: decimal.MustFromFloat(-0.000002)},
		},
		PaymentCurrency: currency.USDT,
	}
	assert.Equal(t, exp, result, "GetHistoricalFundingRates should return every rate")
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.getOpenInterestContracts(t.Context(), asset.Options, currency.EMPTYPAIR)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "options should remain unsupported by futures open interest endpoints")
	_, err = e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.NewCode("GOLDFISH").Item,
		Quote: currency.USDT.Item,
		Asset: asset.USDTMarginedFutures,
	})
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "GetOpenInterest should error correctly")

	if mockTests {
		pairKey := func(a asset.Item) key.PairAsset {
			p := getPair(t, a)
			return key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}
		}
		openInterest := func(a asset.Item, amount float64) futures.OpenInterest {
			return futures.OpenInterest{Key: key.NewExchangeAssetPair(e.Name, a, getPair(t, a)), OpenInterest: amount}
		}
		// A pair asked for on its own is read from its contract and several from the contract list, which the fixtures
		// record at different times. Open interest is sized in the quote currency, which an inverse contract's
		// position already is
		for _, tc := range []struct {
			name string
			keys []key.PairAsset
			exp  []futures.OpenInterest
		}{
			{
				name: "USDT-M pair",
				keys: []key.PairAsset{pairKey(asset.USDTMarginedFutures)},
				exp:  []futures.OpenInterest{openInterest(asset.USDTMarginedFutures, 1949859099.6860802)},
			},
			{
				name: "COIN-M pair",
				keys: []key.PairAsset{pairKey(asset.CoinMarginedFutures)},
				exp:  []futures.OpenInterest{openInterest(asset.CoinMarginedFutures, 8701256)},
			},
			{
				name: "delivery pair",
				keys: []key.PairAsset{pairKey(asset.DeliveryFutures)},
				exp:  []futures.OpenInterest{openInterest(asset.DeliveryFutures, 8485.242744000001)},
			},
			{
				name: "perpetual pairs",
				keys: []key.PairAsset{pairKey(asset.CoinMarginedFutures), pairKey(asset.USDTMarginedFutures)},
				exp:  []futures.OpenInterest{openInterest(asset.CoinMarginedFutures, 8701256), openInterest(asset.USDTMarginedFutures, 118106.768)},
			},
			{
				name: "enabled pairs",
				exp:  []futures.OpenInterest{openInterest(asset.DeliveryFutures, 95.8187), openInterest(asset.CoinMarginedFutures, 8701256), openInterest(asset.USDTMarginedFutures, 118106.768)},
			},
		} {
			result, err := e.GetOpenInterest(t.Context(), tc.keys...)
			require.NoErrorf(t, err, "GetOpenInterest must not error for %s", tc.name)
			assert.Equalf(t, tc.exp, result, "GetOpenInterest should return the open interest of %s", tc.name)
		}
		return
	}
	var resp []futures.OpenInterest
	validPairs := make(map[asset.Item]key.PairAsset, 3)
	for _, a := range []asset.Item{asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures} {
		pairs := getPairs(t, a)
		if a == asset.DeliveryFutures {
			// The enabled delivery contract is picked at random and may have no open positions
			pairs, err = e.GetAvailablePairs(a)
			require.NoErrorf(t, err, "GetAvailablePairs must not error for %s", a)
		}
		pair, response := getPairWithOpenInterest(t, a, pairs)
		validPairs[a] = pair
		resp = response
		require.Lenf(t, resp, 1, "GetOpenInterest must return 1 item for %s asset", a)
		assert.Positivef(t, resp[0].OpenInterest, "GetOpenInterest should return positive open interest for %s asset", a)
	}

	coinPair := validPairs[asset.CoinMarginedFutures]
	usdtPair := validPairs[asset.USDTMarginedFutures]
	resp, err = e.GetOpenInterest(
		t.Context(),
		coinPair,
		usdtPair,
	)
	assert.NoError(t, err, "GetOpenInterest should not error for multiple explicit perpetual pairs")
	require.Len(t, resp, 2, "GetOpenInterest returns exactly the requested perpetual pairs")

	expected := map[asset.Item]currency.Pair{
		asset.CoinMarginedFutures: coinPair.Pair(),
		asset.USDTMarginedFutures: usdtPair.Pair(),
	}
	found := make(map[asset.Item]bool, len(expected))
	for _, oi := range resp {
		expPair, ok := expected[oi.Key.Asset]
		require.Truef(t, ok, "unexpected asset in OpenInterest response: %v", oi.Key.Asset)
		assert.Truef(t, expPair.Equal(oi.Key.Pair()), "OpenInterest pair mismatch for asset %v", oi.Key.Asset)
		assert.Positivef(t, oi.OpenInterest, "OpenInterest should return positive open interest for asset %v", oi.Key.Asset)
		found[oi.Key.Asset] = true
	}
	require.Len(t, found, len(expected), "OpenInterest response missing expected assets")

	resp, err = e.GetOpenInterest(t.Context())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, "GetOpenInterest should return some items")
}

func TestGetClientOrderIDFromText(t *testing.T) {
	t.Parallel()
	assert.Empty(t, getClientOrderIDFromText("api"), "should not return anything")
	assert.Equal(t, "t-123", getClientOrderIDFromText("t-123"), "should return t-123")
}

func TestFormatClientOrderID(t *testing.T) {
	t.Parallel()
	assert.Empty(t, formatClientOrderID(""), "should not return anything")
	assert.Equal(t, "t-123", formatClientOrderID("t-123"), "should return t-123")
	assert.Equal(t, "t-456", formatClientOrderID("456"), "should return t-456")
}

func TestGetSideAndAmountFromSize(t *testing.T) {
	t.Parallel()
	side, amount, remaining := getSideAndAmountFromSize(1, 1)
	assert.Equal(t, order.Long, side, "should be a buy order")
	assert.Equal(t, 1.0, amount, "should be 1.0")
	assert.Equal(t, 1.0, remaining, "should be 1.0")

	side, amount, remaining = getSideAndAmountFromSize(-1, -1)
	assert.Equal(t, order.Short, side, "should be a sell order")
	assert.Equal(t, 1.0, amount, "should be 1.0")
	assert.Equal(t, 1.0, remaining, "should be 1.0")
}

func TestTradeSideAndAmount(t *testing.T) {
	t.Parallel()
	for size, exp := range map[float64]struct {
		side   order.Side
		amount float64
	}{
		4:  {side: order.Buy, amount: 4},
		-4: {side: order.Sell, amount: 4},
	} {
		side, amount := tradeSideAndAmount(size)
		assert.Equalf(t, exp.side, side, "tradeSideAndAmount should return the taker side of size %v", size)
		assert.Equalf(t, exp.amount, amount, "tradeSideAndAmount should return the unsigned amount of size %v", size)
	}
}

func TestGetFutureOrderSize(t *testing.T) {
	t.Parallel()
	_, err := getFutureOrderSize(&order.Submit{Side: order.CouldNotCloseShort, Amount: 1})
	assert.ErrorIs(t, err, order.ErrSideIsInvalid)

	ret, err := getFutureOrderSize(&order.Submit{Side: order.Buy, Amount: 1})
	require.NoError(t, err)
	assert.Equal(t, 1.0, ret)

	ret, err = getFutureOrderSize(&order.Submit{Side: order.Sell, Amount: 1})
	require.NoError(t, err)
	assert.Equal(t, -1.0, ret)
}

func TestProcessFuturesOrdersPushData(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		incoming string
		status   order.Status
	}{
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"open","stp_id":"123456","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.Open},
	}
	processed, err := e.processFuturesOrdersPushData([]byte(testCases[0].incoming), asset.CoinMarginedFutures)
	require.NoError(t, err)
	require.Len(t, processed, 1, "futures order update must contain one order")
	assert.Equal(t, testCases[0].status, processed[0].Status, "futures order status should be open")

	incomingWithoutTIF := strings.Replace(testCases[0].incoming, `,"tif":"gtc"`, "", 1)
	processed, err = e.processFuturesOrdersPushData([]byte(incomingWithoutTIF), asset.CoinMarginedFutures)
	require.NoError(t, err, "futures order update without TIF must use Gate's default")
	require.Len(t, processed, 1, "futures order update without TIF must contain one order")
	assert.Equal(t, order.GoodTillCancel, processed[0].TimeInForce, "futures order without TIF should default to GTC")
}

func TestProcessFinishedFuturesOrdersPushData(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		incoming string
		status   order.Status
	}{
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"filled","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.Filled},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"cancelled","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.Cancelled},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"liquidated","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.Liquidated},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"ioc","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.Cancelled},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"auto_deleveraged","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.AutoDeleverage},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"reduce_only","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.Cancelled},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"position_closed","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.Closed},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"stp","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.STP},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			t.Parallel()
			processed, err := e.processFuturesOrdersPushData([]byte(tc.incoming), asset.CoinMarginedFutures)
			require.NoError(t, err)
			require.Len(t, processed, 1, "futures order update must contain one order")
			for i := range processed {
				assert.Equal(t, tc.status.String(), processed[i].Status.String())
				assert.Equal(t, order.Long, processed[i].Side, "positive futures order size should map to long")
				assert.Equal(t, 1.0, processed[i].Amount, "futures order amount should be absolute")
				assert.Equal(t, order.Limit, processed[i].Type, "priced GTC futures order should be a limit order")
				assert.Equal(t, order.GoodTillCancel, processed[i].TimeInForce, "GTC futures order should preserve time in force")
				assert.Equal(t, 40000.4, processed[i].AverageExecutedPrice, "futures order should preserve its fill price")
			}
		})
	}

	t.Run("reduce out among other orders", func(t *testing.T) {
		t.Parallel()
		order1 := `{"contract":"BTC_USD","create_time":1628736847,"fill_price":0,"finish_as":"reduce_out","finish_time":1628736848,"id":4872461,"is_reduce_only":true,"left":2,"price":40000.4,"size":-2,"status":"finished","text":"-","tif":"gtc","user":"110xxxxx"}`
		order2 := `{"contract":"BTC_USD","create_time":1628736847,"fill_price":0,"finish_as":"","id":4872462,"left":1,"price":39000,"size":1,"status":"open","text":"-","tif":"gtc","user":"110xxxxx"}`
		incoming := `{"channel":"futures.orders","event":"update","time":1541505434,"result":[` + order1 + `,` + order2 + `]}`
		processed, err := e.processFuturesOrdersPushData([]byte(incoming), asset.CoinMarginedFutures)
		require.NoError(t, err, "processFuturesOrdersPushData must accept an order finished as reduce_out")
		require.Len(t, processed, 2, "processFuturesOrdersPushData must keep every order of the push")
		assert.Equal(t, order.Cancelled, processed[0].Status, "an order finished as reduce_out should be cancelled")
		assert.Equal(t, order.Open, processed[1].Status, "an open order should stay open")
	})

	t.Run("finish_as spellings Gate's futures websocket page lists", func(t *testing.T) {
		t.Parallel()
		order1 := `{"contract":"BTC_USD","create_time":1628736847,"finish_as":"position_close","finish_time":1628736848,"id":4872463,"left":2,"price":40000.4,"size":-2,"status":"finished","text":"-","tif":"gtc","user":"110xxxxx"}`
		order2 := `{"contract":"BTC_USD","create_time":1628736847,"finish_as":"auto_deleveraging","finish_time":1628736848,"id":4872464,"left":0,"price":40000.4,"size":3,"status":"finished","text":"auto_deleveraging","tif":"gtc","user":"110xxxxx"}`
		order3 := `{"contract":"BTC_USD","create_time":1628736847,"finish_as":"filled","finish_time":1628736848,"id":4872465,"left":0,"price":39000,"size":1,"status":"finished","text":"-","tif":"gtc","user":"110xxxxx"}`
		incoming := `{"channel":"futures.orders","event":"update","time":1541505434,"result":[` + order1 + `,` + order2 + `,` + order3 + `]}`
		processed, err := e.processFuturesOrdersPushData([]byte(incoming), asset.CoinMarginedFutures)
		require.NoError(t, err, "processFuturesOrdersPushData must accept both spellings Gate documents for each finish_as value")
		require.Len(t, processed, 3, "processFuturesOrdersPushData must keep every order of the push")
		assert.Equal(t, order.Closed, processed[0].Status, "an order finished as position_close should be closed")
		assert.Equal(t, order.AutoDeleverage, processed[1].Status, "an order finished as auto_deleveraging should be auto-deleveraged")
		assert.Equal(t, order.Filled, processed[2].Status, "a filled order in the same push should be filled")
	})

	t.Run("malformed push", func(t *testing.T) {
		t.Parallel()
		_, err := e.processFuturesOrdersPushData([]byte(`{`), asset.CoinMarginedFutures)
		assert.Error(t, err, "processFuturesOrdersPushData should reject malformed JSON")
	})

	t.Run("unknown time in force", func(t *testing.T) {
		t.Parallel()
		incoming := strings.Replace(testCases[0].incoming, `"tif":"gtc"`, `"tif":"gtd"`, 1)
		_, err := e.processFuturesOrdersPushData([]byte(incoming), asset.CoinMarginedFutures)
		assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "processFuturesOrdersPushData should reject an unknown time in force")
	})

	t.Run("finished without finish_as", func(t *testing.T) {
		t.Parallel()
		incoming := strings.Replace(testCases[0].incoming, `"finish_as":"filled"`, `"finish_as":""`, 1)
		_, err := e.processFuturesOrdersPushData([]byte(incoming), asset.CoinMarginedFutures)
		assert.ErrorContains(t, err, `"FINISHED"`, "processFuturesOrdersPushData should not guess how a finished order finished")
	})

	t.Run("negative size", func(t *testing.T) {
		t.Parallel()
		incoming := strings.Replace(testCases[0].incoming, `"size":1`, `"size":-1`, 1)
		processed, err := e.processFuturesOrdersPushData([]byte(incoming), asset.CoinMarginedFutures)
		require.NoError(t, err)
		require.Len(t, processed, 1, "futures order update must contain one order")
		assert.Equal(t, order.Short, processed[0].Side, "negative futures order size should map to short")
		assert.Equal(t, 1.0, processed[0].Amount, "futures order amount should be absolute")
		assert.Equal(t, 1.0, processed[0].ExecutedAmount, "executed futures order amount should be absolute")
	})

	t.Run("market order", func(t *testing.T) {
		t.Parallel()
		incoming := strings.NewReplacer(`"price":40000.4`, `"price":0`, `"tif":"gtc"`, `"tif":"ioc"`).Replace(testCases[1].incoming)
		processed, err := e.processFuturesOrdersPushData([]byte(incoming), asset.USDTMarginedFutures)
		require.NoError(t, err)
		require.Len(t, processed, 1, "futures order update must contain one order")
		assert.Equal(t, order.Market, processed[0].Type, "zero-price IOC futures order should be a market order")
		assert.Equal(t, order.ImmediateOrCancel, processed[0].TimeInForce, "IOC futures order should preserve time in force")
		assert.Zero(t, processed[0].Price, "market order request price should remain zero")
		assert.Equal(t, 40000.4, processed[0].AverageExecutedPrice, "market order should preserve its fill price")
	})
}

// Captured from live GateIO websocket data on 2026-09-03.
// Treat these payloads as semi-trusted until independently confirmed.
func TestFuturesLiveCapturedPayloads(t *testing.T) {
	t.Parallel()

	t.Run("orders", func(t *testing.T) {
		t.Parallel()
		for _, tt := range []struct {
			name             string
			payload          string
			direction        order.Side
			amount           float64
			price            float64
			averageFillPrice float64
			timeInForce      order.TimeInForce
			orderType        order.Type
			reduceOnly       bool
		}{
			{
				name:      "reduce-only IOC close",
				payload:   `{"time":1788387249,"time_ms":1788387249995,"channel":"futures.orders","event":"update","result":[{"create_time":1788387249,"create_time_ms":1788387249995,"fill_price":0.242073333333,"finish_as":"filled","iceberg":"0","id":294985777215626819,"id_string":"294985777215626819","is_close":false,"is_liq":false,"left":"0","mkfr":0.0002,"is_reduce_only":true,"status":"finished","tkfr":0.00048,"price":0,"refu":0,"refr":0,"size":"21","text":"apiv4-ws","tif":"ioc","finish_time":1788387249,"finish_time_ms":1788387249995,"user":"12870774","contract":"RAVE_USDT","stop_profit_price":"","stop_loss_price":"","amend_text":"-","biz_info":"-","stp_act":"-","stp_id":"0","update_id":1,"is_voucher":false,"update_time":1788387249995,"fee":0.024400992,"point_fee":0,"role":"taker","bbo":"-","market_order_slip_ratio":"0.025","pos_margin_mode":"isolated","leverage":"1"}]}`,
				direction: order.Long, orderType: order.Market, amount: 21, averageFillPrice: 0.242073333333,
				timeInForce: order.ImmediateOrCancel, reduceOnly: true,
			},
			{
				name:      "short FOK open",
				payload:   `{"time":1788387300,"time_ms":1788387300728,"channel":"futures.orders","event":"update","result":[{"create_time":1788387300,"create_time_ms":1788387300727,"fill_price":60.97,"finish_as":"filled","iceberg":"0","id":296956100605608003,"id_string":"296956100605608003","is_close":false,"is_liq":false,"left":"0","mkfr":0.0002,"is_reduce_only":false,"status":"finished","tkfr":0.00048,"price":60.91,"refu":0,"refr":0,"size":"-3","text":"apiv4-ws","tif":"fok","finish_time":1788387300,"finish_time_ms":1788387300727,"user":"12870774","contract":"SLVON_USDT","stop_profit_price":"","stop_loss_price":"","amend_text":"-","biz_info":"-","stp_act":"-","stp_id":"0","update_id":1,"is_voucher":false,"update_time":1788387300727,"fee":0.00877968,"point_fee":0,"role":"taker","bbo":"-","market_order_slip_ratio":"0","pos_margin_mode":"isolated","leverage":"1"}]}`,
				direction: order.Short, orderType: order.Limit, amount: 3, price: 60.91, averageFillPrice: 60.97,
				timeInForce: order.FillOrKill,
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				processed, err := e.processFuturesOrdersPushData([]byte(tt.payload), asset.USDTMarginedFutures)
				require.NoError(t, err, "processFuturesOrdersPushData must process captured live data")
				require.Len(t, processed, 1, "captured order payload must contain one order")
				assert.Equal(t, order.Filled, processed[0].Status, "captured order status should be filled")
				assert.Equal(t, tt.orderType, processed[0].Type, "captured order type should follow the request price")
				assert.Equal(t, tt.direction, processed[0].Side, "captured order direction should follow signed size")
				assert.Equal(t, tt.amount, processed[0].Amount, "captured order amount should be absolute")
				assert.Equal(t, tt.amount, processed[0].ExecutedAmount, "captured order should be fully executed")
				assert.Zero(t, processed[0].RemainingAmount, "captured order should have no remaining amount")
				assert.Equal(t, tt.price, processed[0].Price, "captured order price should be preserved")
				assert.Equal(t, tt.averageFillPrice, processed[0].AverageExecutedPrice, "captured fill price should be preserved")
				assert.Equal(t, tt.timeInForce, processed[0].TimeInForce, "captured time in force should be preserved")
				assert.Equal(t, tt.reduceOnly, processed[0].ReduceOnly, "captured reduce-only flag should be preserved")
			})
		}
	})

	t.Run("user trade", func(t *testing.T) {
		t.Parallel()
		for _, tt := range []struct {
			name      string
			payload   string
			direction order.Side
			amount    float64
			tradeID   string
			orderID   string
			timestamp time.Time
		}{
			{
				name: "long", direction: order.Long, amount: 13.4,
				tradeID: "27640330", orderID: "294985777215626819", timestamp: time.UnixMilli(1788387249995),
				payload: `{"time":1788387249,"time_ms":1788387249995,"channel":"futures.usertrades","event":"update","result":[{"id":"27640330","order_id":"294985777215626819","contract":"RAVE_USDT","create_time":1788387249,"create_time_ms":1788387249995,"size":"13.4","role":"taker","price":"0.242","text":"apiv4-ws","fee":0.01556544,"point_fee":0,"amend_text":"-","biz_info":"-","close_size":"13.4"}]}`,
			},
			{
				name: "short", direction: order.Short, amount: 6,
				tradeID: "1762690", orderID: "296956100605606387", timestamp: time.UnixMilli(1788387276248),
				payload: `{"time":1788387276,"time_ms":1788387276249,"channel":"futures.usertrades","event":"update","result":[{"id":"1762690","order_id":"296956100605606387","contract":"SLVON_USDT","create_time":1788387276,"create_time_ms":1788387276248,"size":"-6","role":"taker","price":"61.13","text":"apiv4-ws","fee":0.01760544,"point_fee":0,"amend_text":"-","biz_info":"-","close_size":"0"}]}`,
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
				ex.Websocket.Fills.Setup(true, ex.Websocket.DataHandler)
				require.NoError(t, ex.processFuturesUserTrades([]byte(tt.payload), asset.USDTMarginedFutures),
					"processFuturesUserTrades must process captured live data")
				message := <-ex.Websocket.DataHandler.C
				fills, ok := message.Data.([]fill.Data)
				require.True(t, ok, "captured user trade must emit fills")
				require.Len(t, fills, 1, "captured user trade must contain one fill")
				assert.Equal(t, tt.direction, fills[0].Side, "captured trade direction should follow signed size")
				assert.Equal(t, tt.amount, fills[0].Amount, "captured trade amount should be absolute")
				assert.Equal(t, tt.tradeID, fills[0].TradeID, "captured trade ID should be preserved")
				assert.Equal(t, tt.orderID, fills[0].OrderID, "captured order ID should be preserved")
				assert.Equal(t, tt.timestamp, fills[0].Timestamp, "captured trade timestamp should preserve millisecond precision")
			})
		}
	})

	t.Run("position", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
		require.NoError(t, ex.CurrencyPairs.StorePairs(asset.USDTMarginedFutures,
			currency.Pairs{currency.NewPair(currency.NewCode("RAVE"), currency.USDT)}, false),
			"StorePairs must register the captured contract")
		payload := []byte(`{"time":1788387249,"time_ms":1788387249995,"channel":"futures.positions","event":"update","result":[{"time":1788387249,"time_ms":1788387249995,"size":"-37","user":"12870774","leverage_max":25,"mode":"single","update_id":4815,"cross_leverage_limit":20,"liq_price":0.4942,"maintenance_rate":0.8,"risk_limit":10000000,"last_close_pnl":0.00672032,"entry_price":0.252077758621,"leverage":1,"realised_pnl":2.0063498704,"contract":"RAVE_USDT","history_point":0,"margin":93.40867384569,"realised_point":0,"history_pnl":-55.48692517441,"pos_margin_mode":"isolated","lever":"1"}]}`)
		require.NoError(t, ex.processFuturesPositionsNotification(t.Context(), payload, asset.USDTMarginedFutures),
			"processFuturesPositionsNotification must process captured live data")
		message := <-ex.Websocket.DataHandler.C
		positions, ok := message.Data.([]futures.Position)
		require.True(t, ok, "captured position must emit canonical positions")
		require.Len(t, positions, 1, "captured position payload must contain one position")
		assert.Equal(t, order.Short, positions[0].LatestDirection, "negative captured position size should map to short")
		assert.Equal(t, "37", positions[0].LatestSize.String(), "captured position size should be absolute")
		assert.Equal(t, int64(4815), positions[0].UpdateID, "captured position update ID should be preserved")
		assert.True(t, positions[0].RealisedPNL.Equal(decimal.MustFromFloat(2.0063498704)), "captured realised PNL should be preserved")
	})
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	if !mockTests {
		testexch.UpdatePairsOnce(t, e)
	}
	for a, page := range map[asset.Item]string{
		asset.Spot:                "trade/",
		asset.Margin:              "trade/",
		asset.CrossMargin:         "trade/",
		asset.USDTMarginedFutures: "futures/USDT/",
		asset.CoinMarginedFutures: "futures/BTC/",
		asset.DeliveryFutures:     "futures-delivery/USDT/",
	} {
		p := getPair(t, a)
		resp, err := e.GetCurrencyTradeURL(t.Context(), a, p)
		require.NoErrorf(t, err, "GetCurrencyTradeURL must not error for %s", a)
		p.Delimiter = currency.UnderscoreDelimiter
		assert.Equalf(t, "https://www.gate.io/"+page+p.Upper().String(), resp, "GetCurrencyTradeURL should return the %s trade page", a)
	}
	_, err := e.GetCurrencyTradeURL(t.Context(), asset.Options, getPair(t, asset.Options))
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetCurrencyTradeURL should reject options")
	_, err = e.GetCurrencyTradeURL(t.Context(), asset.Binary, getPair(t, asset.Spot))
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetCurrencyTradeURL should reject an asset the exchange does not trade")

	ex := newIsolatedExchange(t)
	ethusd := currency.NewPairWithDelimiter("ETH", "USD", currency.UnderscoreDelimiter)
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.CoinMarginedFutures, currency.Pairs{ethusd}, false), "StorePairs must add the contract")
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.CoinMarginedFutures, currency.Pairs{ethusd}, true), "StorePairs must enable the contract")
	_, err = ex.GetCurrencyTradeURL(t.Context(), asset.CoinMarginedFutures, ethusd)
	assert.ErrorIs(t, err, errInvalidSettlementBase, "GetCurrencyTradeURL should reject a coin-margined contract without a settlement")
}

func TestGetUnifiedAccount(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUnifiedAccount(t.Context(), currency.EMPTYCODE, "")
	require.NoError(t, err, "GetUnifiedAccount must not error")
	if mockTests {
		exp := &UnifiedUserAccountResponse{
			Mode:        "multi_currency",
			UserID:      10001,
			RefreshTime: types.Time(time.UnixMilli(1744190254944)),
			Balances: map[currency.Code]BalanceDetails{
				currency.BTC: {
					Available:         0.5,
					Freeze:            0.1,
					Borrowed:          0.02,
					Equity:            0.58,
					TotalFreeze:       0.1,
					TotalLiabilities:  0.02,
					Funding:           0.01,
					FundingVersion:    3,
					CrossBalance:      0.5,
					EnabledCollateral: true,
					BalanceVersion:    42,
				},
				currency.USDT: {Available: 1000, Equity: 1000, CrossBalance: 1000, BalanceVersion: 7},
			},
			Total:                          48050,
			Borrowed:                       1600,
			TotalInitialMargin:             1025.05,
			TotalMarginBalance:             48050,
			TotalMaintenanceMargin:         205.01,
			TotalInitialMarginRate:         46.87,
			TotalMaintenanceMarginRate:     234.37,
			TotalAvailableMargin:           47024.95,
			UnifiedAccountTotal:            48050,
			UnifiedAccountTotalLiabilities: 1600,
			UnifiedAccountTotalEquity:      46450,
			Leverage:                       2,
			SpotOrderLoss:                  12,
			OptionsOrderLoss:               3,
			UseFunding:                     true,
		}
		assert.Equal(t, exp, result, "GetUnifiedAccount should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUnifiedAccount should return the account")
}

func TestGetMaximumBorrowableAmountUnifiedAccount(t *testing.T) {
	t.Parallel()
	_, err := e.GetMaximumBorrowableAmountUnifiedAccount(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMaximumBorrowableAmountUnifiedAccount must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMaximumBorrowableAmountUnifiedAccount(t.Context(), currency.ETH)
	require.NoError(t, err, "GetMaximumBorrowableAmountUnifiedAccount must not error")
	if mockTests {
		exp := &CurrencyAndAmountResponse{Currency: currency.ETH, Amount: 10000}
		assert.Equal(t, exp, result, "GetMaximumBorrowableAmountUnifiedAccount should decode every field")
		return
	}
	assert.Equal(t, currency.ETH, result.Currency, "GetMaximumBorrowableAmountUnifiedAccount should return the requested currency")
}

func TestGetUnifiedAccountMaximumTransferableAmount(t *testing.T) {
	t.Parallel()
	_, err := e.GetUnifiedAccountMaximumTransferableAmount(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetUnifiedAccountMaximumTransferableAmount must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUnifiedAccountMaximumTransferableAmount(t.Context(), currency.ETH)
	require.NoError(t, err, "GetUnifiedAccountMaximumTransferableAmount must not error")
	if mockTests {
		exp := &CurrencyAndAmountResponse{Currency: currency.ETH, Amount: 250.5}
		assert.Equal(t, exp, result, "GetUnifiedAccountMaximumTransferableAmount should decode every field")
		return
	}
	assert.Equal(t, currency.ETH, result.Currency, "GetUnifiedAccountMaximumTransferableAmount should return the requested currency")
}

func TestCurrencyListParam(t *testing.T) {
	t.Parallel()
	_, err := currencyListParam(nil, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodesEmpty, "currencyListParam must reject no currencies")

	_, err = currencyListParam(make([]currency.Code, 11), 10)
	require.ErrorIs(t, err, errTooManyCurrencyCodes, "currencyListParam must reject more currencies than the maximum")

	_, err = currencyListParam([]currency.Code{currency.BTC, currency.EMPTYCODE}, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "currencyListParam must reject an empty currency")

	got, err := currencyListParam([]currency.Code{currency.BTC, currency.USDT, currency.ETH}, 3)
	require.NoError(t, err, "currencyListParam must accept the maximum number of currencies")
	assert.Equal(t, "BTC,USDT,ETH", got, "currencyListParam should join the currencies with commas")
}

func TestGetMultipleTransferableAmountForUnifiedAccounts(t *testing.T) {
	t.Parallel()
	_, err := e.GetMultipleTransferableAmountForUnifiedAccounts(t.Context())
	require.ErrorIs(t, err, currency.ErrCurrencyCodesEmpty, "GetMultipleTransferableAmountForUnifiedAccounts must reject no currencies")

	_, err = e.GetMultipleTransferableAmountForUnifiedAccounts(t.Context(), currency.BTC, currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMultipleTransferableAmountForUnifiedAccounts must reject an empty currency")

	_, err = e.GetMultipleTransferableAmountForUnifiedAccounts(t.Context(), make([]currency.Code, 101)...)
	require.ErrorIs(t, err, errTooManyCurrencyCodes, "GetMultipleTransferableAmountForUnifiedAccounts must reject more than 100 currencies")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMultipleTransferableAmountForUnifiedAccounts(t.Context(), currency.BTC, currency.ETH)
	require.NoError(t, err, "GetMultipleTransferableAmountForUnifiedAccounts must not error")
	if mockTests {
		exp := []*CurrencyAndAmountResponse{{Currency: currency.BTC, Amount: 1.25}, {Currency: currency.ETH, Amount: 250.5}}
		assert.Equal(t, exp, result, "GetMultipleTransferableAmountForUnifiedAccounts should decode every field")
		return
	}
	assert.Len(t, result, 2, "GetMultipleTransferableAmountForUnifiedAccounts should return each requested currency")
}

func TestGetBatchUnifiedAccountMaximumBorrowableAmount(t *testing.T) {
	t.Parallel()
	_, err := e.GetBatchUnifiedAccountMaximumBorrowableAmount(t.Context())
	require.ErrorIs(t, err, currency.ErrCurrencyCodesEmpty, "GetBatchUnifiedAccountMaximumBorrowableAmount must reject no currencies")

	_, err = e.GetBatchUnifiedAccountMaximumBorrowableAmount(t.Context(), currency.BTC, currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetBatchUnifiedAccountMaximumBorrowableAmount must reject an empty currency")

	_, err = e.GetBatchUnifiedAccountMaximumBorrowableAmount(t.Context(), make([]currency.Code, 11)...)
	require.ErrorIs(t, err, errTooManyCurrencyCodes, "GetBatchUnifiedAccountMaximumBorrowableAmount must reject more than 10 currencies")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBatchUnifiedAccountMaximumBorrowableAmount(t.Context(), currency.BTC, currency.ETH)
	require.NoError(t, err, "GetBatchUnifiedAccountMaximumBorrowableAmount must not error")
	if mockTests {
		exp := []*CurrencyAndAmountResponse{{Currency: currency.BTC, Amount: 0.5}, {Currency: currency.ETH, Amount: 12}}
		assert.Equal(t, exp, result, "GetBatchUnifiedAccountMaximumBorrowableAmount should decode every field")
		return
	}
	assert.Len(t, result, 2, "GetBatchUnifiedAccountMaximumBorrowableAmount should return each requested currency")
}

func TestBorrowOrRepay(t *testing.T) {
	t.Parallel()
	_, err := e.BorrowOrRepay(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "BorrowOrRepay must reject a nil request")

	_, err = e.BorrowOrRepay(t.Context(), &BorrowOrRepayRequest{Amount: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "BorrowOrRepay must reject an empty currency")

	_, err = e.BorrowOrRepay(t.Context(), &BorrowOrRepayRequest{Currency: currency.ETH})
	require.ErrorIs(t, err, errLoanTypeIsRequired, "BorrowOrRepay must reject an empty type")

	_, err = e.BorrowOrRepay(t.Context(), &BorrowOrRepayRequest{Currency: currency.ETH, Type: "lend", Amount: 1})
	require.ErrorIs(t, err, errInvalidLoanType, "BorrowOrRepay must reject a type other than borrow or repay")

	_, err = e.BorrowOrRepay(t.Context(), &BorrowOrRepayRequest{Currency: currency.ETH, Type: "borrow"})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "BorrowOrRepay must reject a missing amount")

	_, err = e.BorrowOrRepay(t.Context(), &BorrowOrRepayRequest{Currency: currency.ETH, Type: "borrow", RepaidAll: true})
	require.ErrorIs(t, err, errInvalidRepaidAllOperation, "BorrowOrRepay must reject a full repayment of a borrow")

	_, err = e.BorrowOrRepay(t.Context(), &BorrowOrRepayRequest{Currency: currency.ETH, Type: "repay", Amount: 1, RepaidAll: true})
	require.ErrorIs(t, err, errAmountOverriddenByRepaidAll, "BorrowOrRepay must reject an amount with a full repayment")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.BorrowOrRepay(t.Context(), &BorrowOrRepayRequest{Amount: 2, Currency: currency.ETH, Type: "borrow"})
	require.NoError(t, err, "BorrowOrRepay must not error")
	if mockTests {
		assert.Equal(t, &BorrowOrRepayResponse{TransactionID: 9527}, result, "BorrowOrRepay should decode the transaction ID")
	}

	result, err = e.BorrowOrRepay(t.Context(), &BorrowOrRepayRequest{Currency: currency.ETH, Type: "repay", RepaidAll: true})
	require.NoError(t, err, "BorrowOrRepay must accept a full repayment without an amount")
	if mockTests {
		assert.Equal(t, &BorrowOrRepayResponse{TransactionID: 9528}, result, "BorrowOrRepay should decode the full repayment's transaction ID")
	}
}

func TestGetLoans(t *testing.T) {
	t.Parallel()
	_, err := e.GetLoans(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLoans must reject a nil request")

	_, err = e.GetLoans(t.Context(), &UnifiedLoansRequest{Currency: currency.ETH, Type: "repay"})
	require.ErrorIs(t, err, errInvalidLoanType, "GetLoans must reject a loan record type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLoans(t.Context(), &UnifiedLoansRequest{Currency: currency.USDT, Type: "margin", Page: 1, Limit: 100})
	require.NoError(t, err, "GetLoans must not error")
	if mockTests {
		exp := []*UnifiedLoanResponse{{
			Currency:     currency.USDT,
			CurrencyPair: BTCUSDT,
			Amount:       100,
			Type:         "margin",
			CreateTime:   types.Time(time.UnixMilli(1744103854944)),
			UpdateTime:   types.Time(time.UnixMilli(1744190254944)),
		}}
		assert.Equal(t, exp, result, "GetLoans should decode every field")
		return
	}
	assert.NotNil(t, result, "GetLoans should return a result")
}

func TestGetLoanRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetLoanRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLoanRecords must reject a nil request")

	_, err = e.GetLoanRecords(t.Context(), &UnifiedLoansRequest{Currency: currency.ETH, Type: "margin"})
	require.ErrorIs(t, err, errInvalidLoanType, "GetLoanRecords must reject a loan type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLoanRecords(t.Context(), &UnifiedLoansRequest{Currency: currency.ETH, Type: "repay", Page: 1, Limit: 100})
	require.NoError(t, err, "GetLoanRecords must not error")
	if mockTests {
		exp := []*UnifiedLoanRecordResponse{{
			ID:            16442,
			Type:          "repay",
			RepaymentType: "auto_repay",
			BorrowType:    "auto_borrow",
			MarginMode:    "cross",
			CurrencyPair:  currency.NewPairWithDelimiter("ETH", "USDT", currency.UnderscoreDelimiter),
			Currency:      currency.ETH,
			Amount:        0.5,
			CreateTime:    types.Time(time.UnixMilli(1744150000000)),
		}}
		assert.Equal(t, exp, result, "GetLoanRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetLoanRecords should return a result")
}

func TestGetLoansHelper(t *testing.T) {
	t.Parallel()
	var result []*UnifiedLoanResponse
	err := e.getLoans(t.Context(), unifiedLoansEPL, "unified/loans", &UnifiedLoansRequest{Limit: 101}, &result)
	require.ErrorIs(t, err, errInvalidLimit, "getLoans must reject a limit above 100")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	err = e.getLoans(t.Context(), unifiedLoansEPL, "unified/loans", &UnifiedLoansRequest{Currency: currency.USDT, Type: "margin", Page: 1, Limit: 100}, &result)
	require.NoError(t, err, "getLoans must not error")
	if mockTests {
		require.Len(t, result, 1, "getLoans must decode the recorded loan")
		assert.Equal(t, currency.USDT, result[0].Currency, "getLoans should decode into the supplied result")
		return
	}
	assert.NotNil(t, result, "getLoans should decode a result")
}

func TestGetInterestDeductionRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetInterestDeductionRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetInterestDeductionRecords must reject a nil request")

	startTime, endTime := getTime()
	_, err = e.GetInterestDeductionRecords(t.Context(), &UnifiedInterestRecordsRequest{Currency: currency.BTC, Type: "platform", StartTime: endTime, EndTime: startTime, Limit: 10})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetInterestDeductionRecords must reject a start after the end")

	_, err = e.GetInterestDeductionRecords(t.Context(), &UnifiedInterestRecordsRequest{Currency: currency.BTC, Limit: 101})
	require.ErrorIs(t, err, errInvalidLimit, "GetInterestDeductionRecords must reject a limit above 100")

	_, err = e.GetInterestDeductionRecords(t.Context(), &UnifiedInterestRecordsRequest{Currency: currency.BTC, Type: "invalid"})
	require.ErrorIs(t, err, errInvalidLoanType, "GetInterestDeductionRecords must reject an unknown loan type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetInterestDeductionRecords(t.Context(), &UnifiedInterestRecordsRequest{Currency: currency.BTC, Type: "platform", StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "GetInterestDeductionRecords must not error")
	if mockTests {
		exp := []*InterestDeductionRecordResponse{{
			Status:       1,
			CurrencyPair: BTCUSDT,
			Currency:     currency.BTC,
			ActualRate:   0.00000236,
			Interest:     0.00006136,
			LoanType:     "platform",
			CreateTime:   types.Time(time.UnixMilli(1744150000000)),
		}}
		assert.Equal(t, exp, result, "GetInterestDeductionRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetInterestDeductionRecords should return a result")
}

func TestSetUnifiedAccountMode(t *testing.T) {
	t.Parallel()
	err := e.SetUnifiedAccountMode(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetUnifiedAccountMode must reject a nil request")

	err = e.SetUnifiedAccountMode(t.Context(), &UnifiedAccountModeRequest{Settings: &UnifiedAccountModeSettingsRequest{}})
	require.ErrorIs(t, err, errMissingUnifiedAccountMode, "SetUnifiedAccountMode must reject an empty mode")

	arg := &UnifiedAccountModeRequest{Mode: "portfolio", Settings: &UnifiedAccountModeSettingsRequest{SpotHedge: new(true), USDTFutures: new(false)}}
	payload, err := json.Marshal(arg)
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"mode":"portfolio","settings":{"spot_hedge":true,"usdt_futures":false}}`, string(payload), "SetUnifiedAccountMode should send a disabled switch and omit unset ones")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.SetUnifiedAccountMode(t.Context(), arg), "SetUnifiedAccountMode should not error")
}

func TestGetUnifiedAccountMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUnifiedAccountMode(t.Context())
	require.NoError(t, err, "GetUnifiedAccountMode must not error")
	if mockTests {
		exp := &UnifiedAccountModeResponse{
			Mode:     "portfolio",
			Settings: UnifiedAccountModeSettings{USDTFutures: true, SpotHedge: true, Options: true},
		}
		assert.Equal(t, exp, result, "GetUnifiedAccountMode should decode every field")
		return
	}
	assert.NotEmpty(t, result.Mode, "GetUnifiedAccountMode should return the mode")
}

func TestGetUnifiedAccountEstimatedInterestRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetUnifiedAccountEstimatedInterestRate(t.Context(), nil)
	require.ErrorIs(t, err, currency.ErrCurrencyCodesEmpty, "GetUnifiedAccountEstimatedInterestRate must reject no currencies")

	_, err = e.GetUnifiedAccountEstimatedInterestRate(t.Context(), []currency.Code{currency.BTC, currency.USDT, currency.EMPTYCODE})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetUnifiedAccountEstimatedInterestRate must reject an empty currency")

	_, err = e.GetUnifiedAccountEstimatedInterestRate(t.Context(), make([]currency.Code, 11))
	require.ErrorIs(t, err, errTooManyCurrencyCodes, "GetUnifiedAccountEstimatedInterestRate must reject more than 10 currencies")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUnifiedAccountEstimatedInterestRate(t.Context(), []currency.Code{currency.BTC, currency.USDT, currency.ETH})
	require.NoError(t, err, "GetUnifiedAccountEstimatedInterestRate must not error")
	if mockTests {
		exp := map[currency.Code]types.Number{currency.BTC: 0.000002, currency.USDT: 0.000001, currency.ETH: 0}
		assert.Equal(t, exp, result, "GetUnifiedAccountEstimatedInterestRate should decode every rate, an unsupported currency's as zero")
		return
	}
	assert.NotEmpty(t, result, "GetUnifiedAccountEstimatedInterestRate should return rates")
}

func TestTierUpperLimit(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		input string
		exp   TierUpperLimit
	}{
		{input: `"+"`, exp: TierUpperLimit(math.Inf(1))},
		{input: `""`, exp: TierUpperLimit(math.Inf(1))},
		{input: `"100000"`, exp: 100000},
		{input: `500000.5`, exp: 500000.5},
	} {
		var got TierUpperLimit
		require.NoErrorf(t, got.UnmarshalJSON([]byte(tc.input)), "UnmarshalJSON must not error for %s", tc.input)
		assert.Equalf(t, tc.exp, got, "UnmarshalJSON should decode %s", tc.input)
		assert.Equalf(t, float64(tc.exp), got.Float64(), "Float64 should return the decoded boundary for %s", tc.input)
	}

	var got TierUpperLimit
	assert.Error(t, got.UnmarshalJSON([]byte(`"unbounded"`)), "UnmarshalJSON should reject a non-numeric boundary")

	payload, err := TierUpperLimit(math.Inf(1)).MarshalJSON()
	require.NoError(t, err, "MarshalJSON must not error for an unbounded tier")
	assert.Equal(t, `"+"`, string(payload), "MarshalJSON should encode an unbounded tier as Gate does")

	payload, err = json.Marshal([]TierUpperLimit{100000})
	require.NoError(t, err, "Marshal must not error for a bounded tier")
	assert.Equal(t, `["100000"]`, string(payload), "Marshal should encode a bounded tier as a quoted number")
}

func TestGetUnifiedAccountTiered(t *testing.T) {
	t.Parallel()
	result, err := e.GetUnifiedAccountTiered(t.Context())
	require.NoError(t, err, "GetUnifiedAccountTiered must not error")
	if mockTests {
		exp := []*UnifiedAccountTieredDetailResponse{
			{
				Currency:      currency.USDT,
				DiscountTiers: []*UnifiedDiscountTier{{Tier: "1", Discount: 1, UpperLimit: TierUpperLimit(math.Inf(1))}},
			},
			{
				Currency: currency.USDC,
				DiscountTiers: []*UnifiedDiscountTier{
					{Tier: "1", Discount: 1, UpperLimit: 10000000},
					{Tier: "2", Discount: 0.98, LowerLimit: 10000000, UpperLimit: 15000000},
					{Tier: "3", Discount: 0.95, LowerLimit: 15000000, UpperLimit: 20000000},
					{Tier: "4", Discount: 0.925, LowerLimit: 20000000, UpperLimit: 50000000},
					{Tier: "5", Discount: 0.9, LowerLimit: 50000000, UpperLimit: 100000000},
					{Tier: "6", LowerLimit: 100000000, UpperLimit: TierUpperLimit(math.Inf(1))},
				},
			},
		}
		assert.Equal(t, exp, result, "GetUnifiedAccountTiered should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetUnifiedAccountTiered should return tiers")
}

func TestGetUnifiedAccountTieredLoanMargin(t *testing.T) {
	t.Parallel()
	result, err := e.GetUnifiedAccountTieredLoanMargin(t.Context())
	require.NoError(t, err, "GetUnifiedAccountTieredLoanMargin must not error")
	if mockTests {
		exp := []*UnifiedAccountLoanMarginResponse{{
			Currency: currency.USDT,
			MarginTiers: []*UnifiedMarginTier{
				{Tier: "1", MarginRate: 0.01, UpperLimit: 100000, Leverage: 20},
				{Tier: "2", MarginRate: 0.02, LowerLimit: 100000, UpperLimit: 500000, Leverage: 10},
				{Tier: "3", MarginRate: 0.04, LowerLimit: 500000, UpperLimit: 1000000, Leverage: 8},
				{Tier: "4", MarginRate: 0.06, LowerLimit: 1000000, UpperLimit: 2000000, Leverage: 5},
				{Tier: "5", MarginRate: 0.08, LowerLimit: 2000000, UpperLimit: 5000000, Leverage: 4},
				{Tier: "6", MarginRate: 0.1, LowerLimit: 5000000, UpperLimit: 10000000, Leverage: 2.5},
				{Tier: "7", MarginRate: 0.15, LowerLimit: 10000000, UpperLimit: 20000000, Leverage: 1.5},
				{Tier: "8", MarginRate: 0.3, LowerLimit: 20000000, UpperLimit: TierUpperLimit(math.Inf(1))},
			},
		}}
		assert.Equal(t, exp, result, "GetUnifiedAccountTieredLoanMargin should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetUnifiedAccountTieredLoanMargin should return tiers")
}

func TestCalculatePortfolioMargin(t *testing.T) {
	t.Parallel()
	spotPair, futuresPair, optionsPair := getPair(t, asset.Spot), getPair(t, asset.USDTMarginedFutures), getPair(t, asset.Options)
	for _, tc := range []struct {
		name string
		arg  *PortfolioMarginCalculatorRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "nil spot balance", arg: &PortfolioMarginCalculatorRequest{SpotBalances: []*SpotBalanceParams{nil}}, err: common.ErrNilPointer},
		{name: "spot balance without currency", arg: &PortfolioMarginCalculatorRequest{SpotBalances: []*SpotBalanceParams{{Equity: 1000}}}, err: currency.ErrCurrencyCodeEmpty},
		{name: "spot balance without equity", arg: &PortfolioMarginCalculatorRequest{SpotBalances: []*SpotBalanceParams{{Currency: currency.BTC}}}, err: errInvalidEquity},
		{name: "nil spot order", arg: &PortfolioMarginCalculatorRequest{SpotOrders: []*SpotOrderParams{nil}}, err: common.ErrNilPointer},
		{name: "spot order without pair", arg: &PortfolioMarginCalculatorRequest{SpotOrders: []*SpotOrderParams{{OrderPrice: 60000, Left: 0.1, Side: order.Buy}}}, err: currency.ErrCurrencyPairEmpty},
		{name: "spot order without price", arg: &PortfolioMarginCalculatorRequest{SpotOrders: []*SpotOrderParams{{CurrencyPair: spotPair, Left: 0.1, Side: order.Buy}}}, err: limits.ErrPriceBelowMin},
		{name: "spot order without unfilled quantity", arg: &PortfolioMarginCalculatorRequest{SpotOrders: []*SpotOrderParams{{CurrencyPair: spotPair, OrderPrice: 60000, Side: order.Buy}}}, err: errInvalidOrderSize},
		{name: "spot order without side", arg: &PortfolioMarginCalculatorRequest{SpotOrders: []*SpotOrderParams{{CurrencyPair: spotPair, OrderPrice: 60000, Left: 0.1}}}, err: order.ErrSideIsInvalid},
		{name: "nil futures position", arg: &PortfolioMarginCalculatorRequest{FuturesPositions: []*FuturesOrderInfo{nil}}, err: common.ErrNilPointer},
		{name: "futures position without contract", arg: &PortfolioMarginCalculatorRequest{FuturesPositions: []*FuturesOrderInfo{{Size: -1}}}, err: currency.ErrCurrencyPairEmpty},
		{name: "futures position without size", arg: &PortfolioMarginCalculatorRequest{FuturesPositions: []*FuturesOrderInfo{{Contract: futuresPair}}}, err: errInvalidOrderSize},
		{name: "nil futures order", arg: &PortfolioMarginCalculatorRequest{FuturesOrders: []*FuturesOrderInfo{nil}}, err: common.ErrNilPointer},
		{name: "futures order without contract", arg: &PortfolioMarginCalculatorRequest{FuturesOrders: []*FuturesOrderInfo{{Size: 1, Left: 0.5}}}, err: currency.ErrCurrencyPairEmpty},
		{name: "futures order without size", arg: &PortfolioMarginCalculatorRequest{FuturesOrders: []*FuturesOrderInfo{{Contract: futuresPair, Left: 0.5}}}, err: errInvalidOrderSize},
		{name: "futures order without unfilled quantity", arg: &PortfolioMarginCalculatorRequest{FuturesOrders: []*FuturesOrderInfo{{Contract: futuresPair, Size: 1}}}, err: errInvalidOrderSize},
		{name: "nil options position", arg: &PortfolioMarginCalculatorRequest{OptionsPositions: []*OptionsOrderInfo{nil}}, err: common.ErrNilPointer},
		{name: "options position without name", arg: &PortfolioMarginCalculatorRequest{OptionsPositions: []*OptionsOrderInfo{{Size: 1}}}, err: currency.ErrCurrencyPairEmpty},
		{name: "options position without size", arg: &PortfolioMarginCalculatorRequest{OptionsPositions: []*OptionsOrderInfo{{OptionsName: optionsPair}}}, err: errInvalidOrderSize},
		{name: "nil options order", arg: &PortfolioMarginCalculatorRequest{OptionsOrders: []*OptionsOrderInfo{nil}}, err: common.ErrNilPointer},
		{name: "options order without name", arg: &PortfolioMarginCalculatorRequest{OptionsOrders: []*OptionsOrderInfo{{Size: 1, Left: 0.5}}}, err: currency.ErrCurrencyPairEmpty},
		{name: "options order without size", arg: &PortfolioMarginCalculatorRequest{OptionsOrders: []*OptionsOrderInfo{{OptionsName: optionsPair, Left: 0.5}}}, err: errInvalidOrderSize},
		{name: "options order without unfilled quantity", arg: &PortfolioMarginCalculatorRequest{OptionsOrders: []*OptionsOrderInfo{{OptionsName: optionsPair, Size: 1}}}, err: errInvalidOrderSize},
	} {
		_, err := e.CalculatePortfolioMargin(t.Context(), tc.arg)
		require.ErrorIsf(t, err, tc.err, "CalculatePortfolioMargin must reject a request with a %s", tc.name)
	}

	// Short positions and sell orders carry negative sizes
	arg := &PortfolioMarginCalculatorRequest{
		SpotBalances:     []*SpotBalanceParams{{Currency: currency.BTC, Equity: -1}},
		SpotOrders:       []*SpotOrderParams{{CurrencyPair: spotPair, OrderPrice: 60000, Count: 1, Left: 1, Side: order.Sell}},
		FuturesPositions: []*FuturesOrderInfo{{Contract: futuresPair, Size: -100}},
		FuturesOrders:    []*FuturesOrderInfo{{Contract: futuresPair, Size: -10, Left: -8}},
	}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.CalculatePortfolioMargin(t.Context(), arg)
	require.NoError(t, err, "CalculatePortfolioMargin must not error")
	if mockTests {
		exp := &PortfolioMarginCalculationResponse{
			MaintainMarginTotal: 123.171795,
			InitialMarginTotal:  160.1233335,
			CalculateTime:       types.Time(time.Unix(1791854278, 0)),
			RiskUnits: []*PortfolioMarginRiskUnit{{
				Symbol:         currency.BTC,
				MaintainMargin: 123.171795,
				InitialMargin:  160.1233335,
				MarginResults: []*PortfolioMarginResult{{
					Type: "original_position",
					ProfitLossRanges: []*VolatilityInfo{
						{PricePercentage: -0.15, ImpliedVolatilityPercentage: -0.25, ProfitLoss: 123.171795},
						{PricePercentage: -0.12, ImpliedVolatilityPercentage: -0.25, ProfitLoss: 98.537436},
					},
					MaxLoss:    &VolatilityInfo{PricePercentage: 0.15, ImpliedVolatilityPercentage: -0.25, ProfitLoss: -123.171795},
					StressTest: 123.171795,
				}},
				Delta: -0.01,
			}},
		}
		assert.Equal(t, exp, result, "CalculatePortfolioMargin should decode every field")
		return
	}
	assert.NotEmpty(t, result.RiskUnits, "CalculatePortfolioMargin should return risk units")
}

func TestSpotOrderParamsMarshalJSON(t *testing.T) {
	t.Parallel()
	payload, err := (&SpotOrderParams{CurrencyPair: BTCUSDT, OrderPrice: 60000, Count: 1, Left: 0.5, Side: order.Buy}).MarshalJSON()
	require.NoError(t, err, "MarshalJSON must not error")
	assert.JSONEq(t, `{"currency_pairs":"BTC_USDT","order_price":"60000","count":"1","left":"0.5","type":"buy"}`, string(payload), "MarshalJSON should send the order side in lower case as the type")
}

func TestSetAllLoanCurrenciesLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.SetAllLoanCurrenciesLeverage(t.Context(), 0)
	require.ErrorIs(t, err, errInvalidLeverage, "SetAllLoanCurrenciesLeverage must reject a zero leverage")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetAllLoanCurrenciesLeverage(t.Context(), 10)
	require.NoError(t, err, "SetAllLoanCurrenciesLeverage must not error")
	if mockTests {
		exp := []*LeverageFailedCurrencyResponse{{Currency: currency.USDT, Reason: "leverage cannot be changed with an outstanding loan"}}
		assert.Equal(t, exp, result, "SetAllLoanCurrenciesLeverage should decode every failed currency")
		return
	}
	assert.NotNil(t, result, "SetAllLoanCurrenciesLeverage should return the failed currencies")
}

func TestGetMaxMinCurrencyLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.GetMaxMinCurrencyLeverage(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMaxMinCurrencyLeverage must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMaxMinCurrencyLeverage(t.Context(), currency.BTC)
	require.NoError(t, err, "GetMaxMinCurrencyLeverage must not error")
	if mockTests {
		exp := &UnifiedCurrencyLeverageConfigResponse{CurrentLeverage: 2, MaxLeverage: 8}
		assert.Equal(t, exp, result, "GetMaxMinCurrencyLeverage should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMaxMinCurrencyLeverage should return the leverage limits")
}

func TestGetUserCurrencyLeverage(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUserCurrencyLeverage(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err, "GetUserCurrencyLeverage must not error")
	if mockTests {
		exp := []*UnifiedLeverageSettingResponse{{Currency: currency.BTC, Leverage: 3}, {Currency: currency.ETH, Leverage: 2}}
		assert.Equal(t, exp, result, "GetUserCurrencyLeverage should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetUserCurrencyLeverage should return currency leverages")
}

func TestSetLoanCurrencyLeverage(t *testing.T) {
	t.Parallel()
	err := e.SetLoanCurrencyLeverage(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetLoanCurrencyLeverage must reject a nil request")

	err = e.SetLoanCurrencyLeverage(t.Context(), &UnifiedLeverageSettingResponse{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SetLoanCurrencyLeverage must reject an empty currency")

	err = e.SetLoanCurrencyLeverage(t.Context(), &UnifiedLeverageSettingResponse{Currency: currency.BTC})
	require.ErrorIs(t, err, errInvalidLeverage, "SetLoanCurrencyLeverage must reject a zero leverage")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.SetLoanCurrencyLeverage(t.Context(), &UnifiedLeverageSettingResponse{Currency: currency.BTC, Leverage: 3}), "SetLoanCurrencyLeverage should not error")
}

func TestGetUnifiedLoanCurrencies(t *testing.T) {
	t.Parallel()
	result, err := e.GetUnifiedLoanCurrencies(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err, "GetUnifiedLoanCurrencies must not error")
	if mockTests {
		exp := []*UnifiedLoanCurrencyResponse{{
			Name:                 currency.BTC,
			Precision:            0.000001,
			MinBorrowAmount:      0.01,
			UserMaxBorrowAmount:  1000000,
			TotalMaxBorrowAmount: 1000000,
			LoanStatus:           "enable",
		}}
		assert.Equal(t, exp, result, "GetUnifiedLoanCurrencies should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetUnifiedLoanCurrencies should return currencies")
}

func TestGetUnifiedHistoricalLendingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetUnifiedHistoricalLendingRates(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUnifiedHistoricalLendingRates must reject a nil request")

	_, err = e.GetUnifiedHistoricalLendingRates(t.Context(), &UnifiedHistoricalLendingRatesRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetUnifiedHistoricalLendingRates must reject an empty currency")

	_, err = e.GetUnifiedHistoricalLendingRates(t.Context(), &UnifiedHistoricalLendingRatesRequest{Currency: currency.USDT, Limit: 101})
	require.ErrorIs(t, err, errInvalidLimit, "GetUnifiedHistoricalLendingRates must reject a limit above 100")

	result, err := e.GetUnifiedHistoricalLendingRates(t.Context(), &UnifiedHistoricalLendingRatesRequest{Currency: currency.USDT})
	require.NoError(t, err, "GetUnifiedHistoricalLendingRates must not error")
	if mockTests {
		exp := &UnifiedHistoricalLendingRateResponse{
			Currency:   currency.USDT,
			Tier:       "1",
			TierUpRate: 1.18,
			Rates:      []*UnifiedLendingRate{{Time: types.Time(time.UnixMilli(1729476160000)), Rate: 0.00010287}},
		}
		assert.Equal(t, exp, result, "GetUnifiedHistoricalLendingRates should decode every field")
		return
	}
	assert.NotEmpty(t, result.Rates, "GetUnifiedHistoricalLendingRates should return rates")
}

func TestSetUnifiedCollateralCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.SetUnifiedCollateralCurrency(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetUnifiedCollateralCurrency must reject a nil request")

	_, err = e.SetUnifiedCollateralCurrency(t.Context(), &UnifiedSetCollateralRequest{CollateralType: 2})
	require.ErrorIs(t, err, errInvalidCollateralType, "SetUnifiedCollateralCurrency must reject an unknown collateral type")

	payload, err := json.Marshal(&UnifiedSetCollateralRequest{})
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"collateral_type":0}`, string(payload), "SetUnifiedCollateralCurrency should send the all currencies collateral type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetUnifiedCollateralCurrency(t.Context(), &UnifiedSetCollateralRequest{CollateralType: 1, EnableList: []currency.Code{currency.BTC, currency.ETH}})
	require.NoError(t, err, "SetUnifiedCollateralCurrency must not error")
	if mockTests {
		assert.Equal(t, &UnifiedSetCollateralResponse{IsSuccess: true}, result, "SetUnifiedCollateralCurrency should decode the result")
		return
	}
	assert.True(t, result.IsSuccess, "SetUnifiedCollateralCurrency should succeed")
}

func TestEstimateQuickRepaymentDetails(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.EstimateQuickRepaymentDetails(t.Context())
	require.NoError(t, err, "EstimateQuickRepaymentDetails must not error")
	if mockTests {
		exp := &UnifiedQuickEstimateRepaymentResponse{
			DebtCurrencies:      []*UnifiedDebtCurrency{{Currency: currency.ETH, DebtAmount: 1.5, EstimatedUSD: 4500, Borrowed: 1.5}},
			AvailableCurrencies: []*UnifiedAvailableCurrency{{Currency: currency.USDT, Available: 5000, EstimatedUSD: 5000}},
		}
		assert.Equal(t, exp, result, "EstimateQuickRepaymentDetails should decode every field")
		return
	}
	assert.NotNil(t, result, "EstimateQuickRepaymentDetails should return an estimate")
}

func TestQuickRepayment(t *testing.T) {
	t.Parallel()
	_, err := e.QuickRepayment(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "QuickRepayment must reject a nil request")

	_, err = e.QuickRepayment(t.Context(), &UnifiedQuickRepaymentRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodesEmpty, "QuickRepayment must reject missing debt currencies")

	_, err = e.QuickRepayment(t.Context(), &UnifiedQuickRepaymentRequest{DebtCurrencies: []currency.Code{currency.USDT}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodesEmpty, "QuickRepayment must reject missing available currencies")

	_, err = e.QuickRepayment(t.Context(), &UnifiedQuickRepaymentRequest{DebtCurrencies: []currency.Code{currency.USDT}, AvailableCurrencies: []currency.Code{currency.EMPTYCODE}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "QuickRepayment must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.QuickRepayment(t.Context(), &UnifiedQuickRepaymentRequest{
		DebtCurrencies:      []currency.Code{currency.USDT, currency.ETH},
		AvailableCurrencies: []currency.Code{currency.USDT, currency.BTC},
	})
	require.NoError(t, err, "QuickRepayment must not error")
	if mockTests {
		exp := &UnifiedQuickRepaymentResponse{
			OrderID:     "qr_123456",
			RepaidInfos: []*UnifiedRepaidInfo{{Currency: currency.USDT, Repaid: 100}, {Currency: currency.ETH, Repaid: 0.5, Left: 1}},
			UsedInfos:   []*UnifiedUsedInfo{{Currency: currency.USDT, Used: 1600}},
		}
		assert.Equal(t, exp, result, "QuickRepayment should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderID, "QuickRepayment should return an order ID")
}

func TestSetUnifiedDeltaNeutral(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetUnifiedDeltaNeutral(t.Context(), true)
	require.NoError(t, err, "SetUnifiedDeltaNeutral must not error")
	assert.Equal(t, &DeltaNeutralResponse{Enabled: true}, result, "SetUnifiedDeltaNeutral should return the enabled state")
}

func TestGetUnifiedDeltaNeutral(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUnifiedDeltaNeutral(t.Context())
	require.NoError(t, err, "GetUnifiedDeltaNeutral must not error")
	if mockTests {
		assert.Equal(t, &DeltaNeutralResponse{Enabled: true}, result, "GetUnifiedDeltaNeutral should decode the setting")
		return
	}
	assert.NotNil(t, result, "GetUnifiedDeltaNeutral should return the setting")
}

func TestGetSettlementCurrency(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		a   asset.Item
		p   currency.Pair
		exp currency.Code
		err error
	}{
		{asset.Futures, currency.EMPTYPAIR, currency.EMPTYCODE, asset.ErrNotSupported},
		{asset.DeliveryFutures, currency.EMPTYPAIR, currency.USDT, nil},
		{asset.DeliveryFutures, getPair(t, asset.DeliveryFutures), currency.USDT, nil},
		{asset.Options, currency.EMPTYPAIR, currency.USDT, nil},
		{asset.Options, currency.NewBTCUSDT(), currency.USDT, nil},
		{asset.USDTMarginedFutures, currency.EMPTYPAIR, currency.USDT, nil},
		{asset.USDTMarginedFutures, getPair(t, asset.USDTMarginedFutures), currency.USDT, nil},
		{asset.USDTMarginedFutures, getPair(t, asset.DeliveryFutures), currency.EMPTYCODE, errInvalidSettlementQuote},

		{asset.CoinMarginedFutures, currency.EMPTYPAIR, currency.BTC, nil},
		{asset.CoinMarginedFutures, currency.Pair{Base: currency.BTC, Quote: currency.USD}, currency.BTC, nil},
		{asset.CoinMarginedFutures, currency.Pair{Base: currency.ETH, Quote: currency.USD}, currency.EMPTYCODE, errInvalidSettlementBase},
	} {
		t.Run(tt.a.String()+"_"+tt.p.String()+"_"+tt.exp.String(), func(t *testing.T) {
			t.Parallel()
			c, err := getSettlementCurrency(tt.p, tt.a)
			if tt.err == nil {
				require.NoErrorf(t, err, "getSettlementCurrency must not error for %s %s", tt.a, tt.p)
			} else {
				assert.ErrorIsf(t, err, tt.err, "getSettlementCurrency should return correct error for %s %s", tt.a, tt.p)
			}
			assert.Equalf(t, tt.exp, c, "getSettlementCurrency should return correct settlement currency for %s %s", tt.a, tt.p)
		})
	}
}

type FixtureConnection struct{ websocket.Connection }

func (d *FixtureConnection) SendMessageReturnResponse(context.Context, request.EndpointLimit, any, any) ([]byte, error) {
	return []byte(`{"time":1726121320,"time_ms":1726121320745,"id":1,"conn_id":"f903779a148987ca","trace_id":"d8ee37cd14347e4ed298d44e69aedaa7","channel":"spot.tickers","event":"subscribe","payload":["BRETT_USDT"],"result":{"status":"success"},"requestId":"d8ee37cd14347e4ed298d44e69aedaa7"}`), nil
}

func (d *FixtureConnection) GetURL() string { return "wss://test" }

func TestHandleSubscriptions(t *testing.T) {
	t.Parallel()
	subs := subscription.List{{Channel: subscription.OrderbookChannel}}
	err := e.handleSubscription(t.Context(), &FixtureConnection{}, subscribeEvent, subs, func(context.Context, string, subscription.List) ([]*WebsocketChannelRequest, error) {
		return []*WebsocketChannelRequest{{}}, nil
	})
	require.NoError(t, err)

	err = e.handleSubscription(t.Context(), &FixtureConnection{}, unsubscribeEvent, subs, func(context.Context, string, subscription.List) ([]*WebsocketChannelRequest, error) {
		return []*WebsocketChannelRequest{{}}, nil
	})
	require.NoError(t, err)

	err = e.handleSubscription(t.Context(), &replyConnection{reply: []byte(`null`)}, subscribeEvent, subs, func(context.Context, string, subscription.List) ([]*WebsocketChannelRequest, error) {
		return []*WebsocketChannelRequest{{}}, nil
	})
	assert.ErrorIs(t, err, common.ErrNoResponse, "handleSubscription should reject a null reply")
}

func TestManageSubsNullReply(t *testing.T) {
	t.Parallel()
	subs := subscription.List{{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{getPair(t, asset.Spot)}}}
	err := e.manageSubs(t.Context(), subscribeEvent, &replyConnection{reply: []byte(`null`)}, subs)
	assert.ErrorIs(t, err, common.ErrNoResponse, "manageSubs should reject a null reply")
}

func TestParseWebsocketHeader(t *testing.T) {
	in := []string{
		`{"time":1726121320,"time_ms":1726121320745,"id":1,"channel":"spot.tickers","event":"subscribe","result":{"status":"success"},"request_id":"a4"}`,
		`{"time_ms":1726121320746,"id":2,"channel":"spot.tickers","event":"subscribe","result":{"status":"success"},"request_id":"a4"}`,
		`{"time":1726121321,"id":3,"channel":"spot.tickers","event":"subscribe","result":{"status":"success"},"request_id":"a4"}`,
	}
	for _, i := range in {
		h, err := parseWSHeader([]byte(i))
		require.NoError(t, err)
		require.NotEmpty(t, h.ID)
		assert.Equal(t, "a4", h.RequestID)
		assert.Equal(t, "spot.tickers", h.Channel)
		assert.Equal(t, "subscribe", h.Event)
		assert.NotEmpty(t, h.Result)
		switch h.ID {
		case 1:
			assert.Equal(t, int64(1726121320745), h.Time.UnixMilli())
		case 2:
			assert.Equal(t, int64(1726121320746), h.Time.UnixMilli())
		case 3:
			assert.Equal(t, int64(1726121321), h.Time.Unix())
		}
	}
}

func TestDeriveSpotWebsocketOrderResponse(t *testing.T) {
	t.Parallel()
	var resp *WebsocketOrderResponse
	require.NoError(t, json.Unmarshal([]byte(`{"left":"0","update_time":"1735720637","amount":"0.0001","create_time":"1735720637","price":"0","finish_as":"filled","time_in_force":"ioc","currency_pair":"BTC_USDT","type":"market","account":"spot","side":"sell","amend_text":"-","text":"t-1735720637181634009","status":"closed","iceberg":"0","avg_deal_price":"93503.3","filled_total":"9.35033","id":"766075454481","fill_price":"9.35033","stp_id":"123456","update_time_ms":1735720637188,"create_time_ms":1735720637188}`), &resp), "unmarshal must not error")

	got, err := e.deriveSpotWebsocketOrderResponse(resp)
	require.NoError(t, err)
	assert.Equal(t, &order.SubmitResponse{
		Exchange:             e.Name,
		OrderID:              "766075454481",
		AssetType:            asset.Spot,
		Pair:                 currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
		ClientOrderID:        "t-1735720637181634009",
		Date:                 time.UnixMilli(1735720637188),
		LastUpdated:          time.UnixMilli(1735720637188),
		Amount:               0.0001,
		AverageExecutedPrice: 93503.3,
		Type:                 order.Market,
		Side:                 order.Sell,
		Status:               order.Filled,
		TimeInForce:          order.ImmediateOrCancel,
		Cost:                 0.0001,
		Purchased:            9.35033,
	}, got)
}

func TestDeriveSpotWebsocketOrderResponses(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		orders   [][]byte
		error    error
		expected []*order.SubmitResponse
	}{
		{
			name:   "no response",
			orders: [][]byte{},
			error:  common.ErrNoResponse,
		},
		{
			name: "assortment of spot orders",
			orders: [][]byte{
				[]byte(`{"left":"0","update_time":"1735720637","amount":"0.0001","create_time":"1735720637","price":"0","finish_as":"filled","time_in_force":"ioc","currency_pair":"BTC_USDT","type":"market","account":"spot","side":"sell","amend_text":"-","text":"t-1735720637181634009","status":"closed","iceberg":"0","avg_deal_price":"93503.3","filled_total":"9.35033","id":"766075454481","fill_price":"9.35033","update_time_ms":1735720637188,"create_time_ms":1735720637188}`),
				[]byte(`{"left":"0.000008","update_time":"1735720637","amount":"9.99152","create_time":"1735720637","price":"0","finish_as":"filled","time_in_force":"ioc","currency_pair":"HNS_USDT","type":"market","account":"spot","side":"buy","amend_text":"-","text":"t-1735720637126962151","status":"closed","iceberg":"0","avg_deal_price":"0.01224","filled_total":"9.991512","id":"766075454188","fill_price":"9.991512","update_time_ms":1735720637142,"create_time_ms":1735720637142}`),
				[]byte(`{"left":"0","update_time":"1735778597","amount":"200","create_time":"1735778597","price":"0.03673","finish_as":"filled","time_in_force":"fok","currency_pair":"REX_USDT","type":"limit","account":"spot","side":"buy","amend_text":"-","text":"t-1364","status":"closed","iceberg":"0","avg_deal_price":"0.03673","filled_total":"7.346","id":"766488882062","fill_price":"7.346","update_time_ms":1735778597363,"create_time_ms":1735778597363}`),
				[]byte(`{"left":"0.0003","update_time":"1735780321","amount":"0.0003","create_time":"1735780321","price":"20000","finish_as":"open","time_in_force":"poc","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"buy","amend_text":"-","text":"t-1735780321603944400","status":"open","iceberg":"0","filled_total":"0","id":"766504537761","fill_price":"0","update_time_ms":1735780321729,"create_time_ms":1735780321729}`),
				[]byte(`{"left":"1","update_time":"1735784755","amount":"1","create_time":"1735784755","price":"100","finish_as":"open","time_in_force":"gtc","currency_pair":"GT_USDT","type":"limit","account":"spot","side":"sell","amend_text":"-","text":"t-1735784754905434100","status":"open","iceberg":"0","filled_total":"0","id":"766536556747","fill_price":"0","update_time_ms":1735784755068,"create_time_ms":1735784755068}`),
			},
			expected: []*order.SubmitResponse{
				{
					Exchange:             e.Name,
					OrderID:              "766075454481",
					AssetType:            asset.Spot,
					Pair:                 currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:        "t-1735720637181634009",
					Date:                 time.UnixMilli(1735720637188),
					LastUpdated:          time.UnixMilli(1735720637188),
					Amount:               0.0001,
					AverageExecutedPrice: 93503.3,
					Type:                 order.Market,
					Side:                 order.Sell,
					Status:               order.Filled,
					TimeInForce:          order.ImmediateOrCancel,
					Cost:                 0.0001,
					Purchased:            9.35033,
				},
				{
					Exchange:             e.Name,
					OrderID:              "766075454188",
					AssetType:            asset.Spot,
					Pair:                 currency.NewPair(currency.HNS, currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:        "t-1735720637126962151",
					Date:                 time.UnixMilli(1735720637142),
					LastUpdated:          time.UnixMilli(1735720637142),
					RemainingAmount:      0.000008,
					Amount:               9.99152,
					AverageExecutedPrice: 0.01224,
					Type:                 order.Market,
					Side:                 order.Buy,
					Status:               order.Filled,
					TimeInForce:          order.ImmediateOrCancel,
					Cost:                 9.991512,
					Purchased:            816.3,
				},
				{
					Exchange:             e.Name,
					OrderID:              "766488882062",
					AssetType:            asset.Spot,
					Pair:                 currency.NewPair(currency.NewCode("REX"), currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:        "t-1364",
					Date:                 time.UnixMilli(1735778597363),
					LastUpdated:          time.UnixMilli(1735778597363),
					Amount:               200,
					Price:                0.03673,
					AverageExecutedPrice: 0.03673,
					Type:                 order.Limit,
					Side:                 order.Buy,
					Status:               order.Filled,
					TimeInForce:          order.FillOrKill,
					Cost:                 7.346,
					Purchased:            200,
				},
				{
					Exchange:        e.Name,
					OrderID:         "766504537761",
					AssetType:       asset.Spot,
					Pair:            currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:   "t-1735780321603944400",
					Date:            time.UnixMilli(1735780321729),
					LastUpdated:     time.UnixMilli(1735780321729),
					RemainingAmount: 0.0003,
					Amount:          0.0003,
					Price:           20000,
					Type:            order.Limit,
					Side:            order.Buy,
					Status:          order.Open,
					TimeInForce:     order.PostOnly,
				},
				{
					Exchange:        e.Name,
					OrderID:         "766536556747",
					AssetType:       asset.Spot,
					Pair:            currency.NewPair(currency.NewCode("GT"), currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:   "t-1735784754905434100",
					Date:            time.UnixMilli(1735784755068),
					LastUpdated:     time.UnixMilli(1735784755068),
					RemainingAmount: 1,
					Amount:          1,
					Price:           100,
					Type:            order.Limit,
					Side:            order.Sell,
					Status:          order.Open,
					TimeInForce:     order.GoodTillCancel,
				},
			},
		},
		{
			// Gate's documented market buy reply, which gives the BTC bought as filled_amount, and a sell
			name: "orders with their filled amounts",
			orders: [][]byte{
				[]byte(`{"id":"12332324","text":"t-123456","amend_text":"-","create_time":"1734081653","update_time":"1734081653","create_time_ms":1734081653247,"update_time_ms":1734081653249,"status":"closed","currency_pair":"BTC_USDT","type":"market","account":"spot","side":"buy","amount":"70.0","price":"0.0","time_in_force":"ioc","iceberg":"0.0","left":"0.143511","filled_amount":"0.00299","fill_price":"69.856489","filled_total":"69.856489","avg_deal_price":"23363.3742475","fee":"0.0000028704","fee_currency":"BTC","finish_as":"filled"}`),
				[]byte(`{"id":"12332325","text":"t-123457","create_time_ms":1734081653250,"update_time_ms":1734081653251,"status":"open","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"sell","amount":"1","price":"60000","time_in_force":"gtc","left":"0.4","filled_amount":"0.6","filled_total":"36000","avg_deal_price":"60000","fee":"36","fee_currency":"USDT","finish_as":"open"}`),
			},
			expected: []*order.SubmitResponse{
				{
					Exchange:             e.Name,
					OrderID:              "12332324",
					AssetType:            asset.Spot,
					Pair:                 currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:        "t-123456",
					Date:                 time.UnixMilli(1734081653247),
					LastUpdated:          time.UnixMilli(1734081653249),
					RemainingAmount:      0.143511,
					Amount:               70,
					AverageExecutedPrice: 23363.3742475,
					Type:                 order.Market,
					Side:                 order.Buy,
					Status:               order.Filled,
					TimeInForce:          order.ImmediateOrCancel,
					Fee:                  0.0000028704,
					FeeAsset:             currency.BTC,
					Cost:                 69.856489,
					Purchased:            0.00299,
				},
				{
					Exchange:             e.Name,
					OrderID:              "12332325",
					AssetType:            asset.Spot,
					Pair:                 currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:        "t-123457",
					Date:                 time.UnixMilli(1734081653250),
					LastUpdated:          time.UnixMilli(1734081653251),
					RemainingAmount:      0.4,
					Amount:               1,
					Price:                60000,
					AverageExecutedPrice: 60000,
					Type:                 order.Limit,
					Side:                 order.Sell,
					Status:               order.Open,
					TimeInForce:          order.GoodTillCancel,
					Fee:                  36,
					FeeAsset:             currency.USDT,
					Cost:                 1,
					Purchased:            36000, // A sell's purchase is in the quote currency, so filled_amount does not apply
				},
			},
		},
		{
			name: "batch of spot orders with error at end",
			// This is specifically testing the return responses of WebsocketSpotSubmitOrders
			// AverageDealPrice is not returned when using this endpoint so purchased and cost fields cannot be set.
			orders: [][]byte{
				[]byte(`{"account":"spot","status":"closed","side":"buy","amount":"9.98","id":"775453816782","create_time":"1736980695","update_time":"1736980695","text":"t-740","left":"0.047239","currency_pair":"BTC_USDT","type":"market","finish_as":"filled","price":"0","time_in_force":"fok","iceberg":"0","filled_total":"9.932761","fill_price":"9.932761","create_time_ms":1736980695949,"update_time_ms":1736980695949,"succeeded":true}`),
				[]byte(`{"account":"spot","status":"closed","side":"buy","amount":"0.00289718","id":"775453816824","create_time":"1736980695","update_time":"1736980695","text":"t-741","left":"0.00000000962","currency_pair":"BTC_USDT","type":"market","finish_as":"filled","price":"0","time_in_force":"fok","iceberg":"0","filled_total":"0.00289717038","fill_price":"0.00289717038","create_time_ms":1736980695956,"update_time_ms":1736980695956,"succeeded":true}`),
				[]byte(`{"text":"t-742","label":"BALANCE_NOT_ENOUGH","message":"Not enough balance"}`),
			},
			expected: []*order.SubmitResponse{
				{
					Exchange:        e.Name,
					OrderID:         "775453816782",
					AssetType:       asset.Spot,
					Pair:            getPair(t, asset.Spot).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:   "t-740",
					Date:            time.UnixMilli(1736980695949),
					LastUpdated:     time.UnixMilli(1736980695949),
					Amount:          9.98,
					RemainingAmount: 0.047239,
					Type:            order.Market,
					Side:            order.Buy,
					Status:          order.Filled,
					TimeInForce:     order.FillOrKill,
				},
				{
					Exchange:        e.Name,
					OrderID:         "775453816824",
					AssetType:       asset.Spot,
					Pair:            getPair(t, asset.Spot).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:   "t-741",
					Date:            time.UnixMilli(1736980695956),
					LastUpdated:     time.UnixMilli(1736980695956),
					RemainingAmount: 0.00000000962,
					Amount:          0.00289718,
					Type:            order.Market,
					Side:            order.Buy,
					Status:          order.Filled,
					TimeInForce:     order.FillOrKill,
				},
				{
					Exchange:        e.Name,
					ClientOrderID:   "t-742",
					SubmissionError: order.ErrUnableToPlaceOrder,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			orders := bytes.Join(tc.orders, []byte(","))
			orders = append([]byte("["), append(orders, []byte("]")...)...)

			var resp []*WebsocketOrderResponse
			require.NoError(t, json.Unmarshal(orders, &resp), "unmarshal must not error")

			got, err := e.deriveSpotWebsocketOrderResponses(resp)
			require.ErrorIs(t, err, tc.error)

			require.Len(t, got, len(tc.expected))
			for i := range got {
				if tc.expected[i].SubmissionError != nil {
					assert.ErrorIs(t, got[i].SubmissionError, tc.expected[i].SubmissionError)
					assert.Equal(t, tc.expected[i].Exchange, got[i].Exchange)
					assert.Equal(t, tc.expected[i].ClientOrderID, got[i].ClientOrderID)
					continue
				}
				assert.Equal(t, tc.expected[i], got[i])
			}
		})
	}
}

func TestDeriveFuturesWebsocketOrderResponse(t *testing.T) {
	t.Parallel()

	var resp *WebsocketFuturesOrderResponse
	require.NoError(t, json.Unmarshal([]byte(`{"text":"t-1337","price":"0","biz_info":"-","tif":"ioc","amend_text":"-","status":"finished","contract":"CWIF_USDT","stp_act":"-","stp_id":"123456","finish_as":"filled","fill_price":"0.0000002625","id":596729318437,"create_time":1735787107.449,"size":2,"finish_time":1735787107.45,"update_time":1735787107.45,"left":0,"user":12870774,"is_reduce_only":true}`), &resp), "unmarshal must not error")

	got, err := e.deriveFuturesWebsocketOrderResponse(resp, asset.USDTMarginedFutures)
	require.NoError(t, err)
	assert.Equal(t, &order.SubmitResponse{
		Exchange:             e.Name,
		OrderID:              "596729318437",
		AssetType:            asset.USDTMarginedFutures,
		Pair:                 currency.NewPair(currency.NewCode("CWIF"), currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
		ClientOrderID:        "t-1337",
		Date:                 time.UnixMilli(1735787107449),
		LastUpdated:          time.UnixMilli(1735787107450),
		Amount:               2,
		AverageExecutedPrice: 0.0000002625,
		Type:                 order.Market,
		Side:                 order.Long,
		Status:               order.Filled,
		TimeInForce:          order.ImmediateOrCancel,
		ReduceOnly:           true,
	}, got)

	// Pairing a USDT contract with a coin-margined caller proves the asset comes from the caller, not the contract.
	got, err = e.deriveFuturesWebsocketOrderResponse(resp, asset.CoinMarginedFutures)
	require.NoError(t, err, "deriveFuturesWebsocketOrderResponse must not error")
	require.NotNil(t, got, "response must not be nil")
	assert.Equal(t, asset.CoinMarginedFutures, got.AssetType, "AssetType should follow the caller")
}

func TestDeriveFuturesWebsocketOrderResponses(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		orders   [][]byte
		error    error
		expected []*order.SubmitResponse
	}{
		{
			name:   "no response",
			orders: [][]byte{},
			error:  common.ErrNoResponse,
		},
		{
			name: "assortment of futures orders",
			orders: [][]byte{
				[]byte(`{"text":"t-1337","price":"0","biz_info":"-","tif":"ioc","amend_text":"-","status":"finished","contract":"CWIF_USDT","stp_act":"-","finish_as":"filled","fill_price":"0.0000002625","id":596729318437,"create_time":1735787107.449,"size":2,"finish_time":1735787107.45,"update_time":1735787107.45,"left":0,"user":12870774,"is_reduce_only":true}`),
				[]byte(`{"text":"t-1336","price":"0","biz_info":"-","tif":"ioc","amend_text":"-","status":"finished","contract":"REX_USDT","stp_act":"-","finish_as":"filled","fill_price":"0.03654","id":596662040388,"create_time":1735778597.374,"size":-2,"finish_time":1735778597.374,"update_time":1735778597.374,"left":0,"user":12870774}`),
				[]byte(`{"text":"apiv4-ws","price":"40000","biz_info":"-","tif":"gtc","amend_text":"-","status":"open","contract":"BTC_USDT","stp_act":"-","fill_price":"0","id":596746193678,"create_time":1735789790.476,"size":1,"update_time":1735789790.476,"left":1,"user":2365748}`),
				[]byte(`{"text":"apiv4-ws","price":"200000","biz_info":"-","tif":"gtc","amend_text":"-","status":"open","contract":"BTC_USDT","stp_act":"-","fill_price":"0","id":596748780649,"create_time":1735790222.185,"size":-1,"update_time":1735790222.185,"left":-1,"user":2365748}`),
				[]byte(`{"text":"apiv4-ws","price":"0","biz_info":"-","tif":"ioc","amend_text":"-","status":"finished","contract":"BTC_USDT","stp_act":"-","finish_as":"filled","fill_price":"98172.9","id":36028797827161124,"create_time":1740108860.761,"size":1,"finish_time":1740108860.761,"update_time":1740108860.761,"left":0,"user":2365748}`),
				[]byte(`{"text":"apiv4-ws","price":"0","biz_info":"-","tif":"ioc","amend_text":"-","status":"finished","contract":"BTC_USDT","stp_act":"-","finish_as":"filled","fill_price":"98113.1","id":36028797827225781,"create_time":1740109172.06,"size":-1,"finish_time":1740109172.06,"update_time":1740109172.06,"left":0,"user":2365748,"is_reduce_only":true}`),
			},
			expected: []*order.SubmitResponse{
				{
					Exchange:             e.Name,
					OrderID:              "596729318437",
					AssetType:            asset.USDTMarginedFutures,
					Pair:                 currency.NewPair(currency.NewCode("CWIF"), currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:        "t-1337",
					Date:                 time.UnixMilli(1735787107449),
					LastUpdated:          time.UnixMilli(1735787107450),
					Amount:               2,
					AverageExecutedPrice: 0.0000002625,
					Type:                 order.Market,
					Side:                 order.Long,
					Status:               order.Filled,
					TimeInForce:          order.ImmediateOrCancel,
					ReduceOnly:           true,
				},
				{
					Exchange:             e.Name,
					OrderID:              "596662040388",
					AssetType:            asset.USDTMarginedFutures,
					Pair:                 currency.NewPair(currency.NewCode("REX"), currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:        "t-1336",
					Date:                 time.UnixMilli(1735778597374),
					LastUpdated:          time.UnixMilli(1735778597374),
					Amount:               2,
					AverageExecutedPrice: 0.03654,
					Type:                 order.Market,
					Side:                 order.Short,
					Status:               order.Filled,
					TimeInForce:          order.ImmediateOrCancel,
				},
				{
					Exchange:        e.Name,
					OrderID:         "596746193678",
					AssetType:       asset.USDTMarginedFutures,
					Pair:            currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					Date:            time.UnixMilli(1735789790476),
					LastUpdated:     time.UnixMilli(1735789790476),
					RemainingAmount: 1,
					Amount:          1,
					Price:           40000,
					Type:            order.Limit,
					Side:            order.Long,
					Status:          order.Open,
					TimeInForce:     order.GoodTillCancel,
				},
				{
					Exchange:        e.Name,
					OrderID:         "596748780649",
					AssetType:       asset.USDTMarginedFutures,
					Pair:            currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					Date:            time.UnixMilli(1735790222185),
					LastUpdated:     time.UnixMilli(1735790222185),
					RemainingAmount: 1,
					Amount:          1,
					Price:           200000,
					Type:            order.Limit,
					Side:            order.Short,
					Status:          order.Open,
					TimeInForce:     order.GoodTillCancel,
				},
				{
					Exchange:             e.Name,
					OrderID:              "36028797827161124",
					AssetType:            asset.USDTMarginedFutures,
					Pair:                 currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					Date:                 time.UnixMilli(1740108860761),
					LastUpdated:          time.UnixMilli(1740108860761),
					Amount:               1,
					AverageExecutedPrice: 98172.9,
					Type:                 order.Market,
					Side:                 order.Long,
					Status:               order.Filled,
					TimeInForce:          order.ImmediateOrCancel,
				},
				{
					Exchange:             e.Name,
					OrderID:              "36028797827225781",
					AssetType:            asset.USDTMarginedFutures,
					Pair:                 currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					Date:                 time.UnixMilli(1740109172060),
					LastUpdated:          time.UnixMilli(1740109172060),
					Amount:               1,
					AverageExecutedPrice: 98113.1,
					Type:                 order.Market,
					Side:                 order.Short,
					Status:               order.Filled,
					TimeInForce:          order.ImmediateOrCancel,
					ReduceOnly:           true,
				},
			},
		},
		{
			// Gate rejects an order of a batch on its own, reporting why in its place
			name: "batch of futures orders with a rejection",
			orders: [][]byte{
				[]byte(`{"succeeded":true,"id":74046545,"user":6790020,"create_time":1681196536.592,"status":"open","contract":"BTC_USDT","size":"10","price":"31403.2","tif":"gtc","left":"10","fill_price":"0","text":"t-my-custom-id","tkfr":"0.0003","mkfr":"0"}`),
				[]byte(`{"succeeded":false,"label":"INSUFFICIENT_AVAILABLE","detail":"insufficient available balance"}`),
			},
			expected: []*order.SubmitResponse{
				{
					Exchange:        e.Name,
					OrderID:         "74046545",
					AssetType:       asset.USDTMarginedFutures,
					Pair:            currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:   "t-my-custom-id",
					Date:            time.UnixMilli(1681196536592),
					LastUpdated:     time.UnixMilli(1681196536592),
					RemainingAmount: 10,
					Amount:          10,
					Price:           31403.2,
					Type:            order.Limit,
					Side:            order.Long,
					Status:          order.Open,
					TimeInForce:     order.GoodTillCancel,
				},
				{
					Exchange:        e.Name,
					AssetType:       asset.USDTMarginedFutures,
					SubmissionError: fmt.Errorf("%w reason label:%q detail:%q", order.ErrUnableToPlaceOrder, "INSUFFICIENT_AVAILABLE", "insufficient available balance"),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			orders := bytes.Join(tc.orders, []byte(","))
			orders = append([]byte("["), append(orders, []byte("]")...)...)

			var resp []*WebsocketFuturesOrderResponse
			require.NoError(t, json.Unmarshal(orders, &resp), "unmarshal must not error")

			got, err := e.deriveFuturesWebsocketOrderResponses(resp, asset.USDTMarginedFutures)
			require.ErrorIs(t, err, tc.error)

			require.Len(t, got, len(tc.expected))
			for i := range got {
				assert.Equal(t, tc.expected[i], got[i])
			}
		})
	}
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

func TestGetAccountDetails(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountDetails(t.Context())
	require.NoError(t, err, "GetAccountDetails must not error")
	if mockTests {
		exp := &AccountDetailsResponse{
			IPWhitelist:     []string{"127.0.0.1"},
			CurrencyPairs:   []currency.Pair{currency.NewPairWithDelimiter("USDT", "BTC", currency.UnderscoreDelimiter)},
			UserID:          1667201533,
			VIPTier:         2,
			Key:             AccountKeyDetails{Mode: 1},
			CopyTradingRole: 1,
		}
		assert.Equal(t, exp, result, "GetAccountDetails should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAccountDetails should return details")
}

func TestGetAccountMainKeys(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountMainKeys(t.Context())
	require.NoError(t, err, "GetAccountMainKeys must not error")
	if mockTests {
		exp := []*AccountMainKeyResponse{{
			State:         1,
			Mode:          1,
			Name:          "test1",
			CurrencyPairs: []currency.Pair{currency.NewPairWithDelimiter("BTC", "USD", currency.UnderscoreDelimiter), currency.NewPairWithDelimiter("ETH", "USD", currency.UnderscoreDelimiter)},
			UserID:        1,
			IPWhitelist:   []string{},
			Permissions:   []*SubAccountKeyPerm{{Name: "account"}, {Name: "spot", ReadOnly: true}},
			Key:           "c5dcfbf1f3a7*****",
			CreatedAt:     types.Time(time.Unix(1753030929, 0)),
			UpdateAt:      types.Time(time.Unix(1756300567, 0)),
			LastAccess:    types.Time(time.Unix(1753030930, 0)),
		}}
		assert.Equal(t, exp, result, "GetAccountMainKeys should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAccountMainKeys should return keys")
}

func TestGetUserTransactionRateLimitInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUserTransactionRateLimitInfo(t.Context())
	require.NoError(t, err, "GetUserTransactionRateLimitInfo must not error")
	if mockTests {
		exp := []*UserTransactionRateLimitInfoResponse{
			{Type: "spot", Tier: 1, Ratio: 0.5, MainRatio: 0.25, UpdatedAt: types.Time(time.Unix(1728230400, 0))},
			{Type: "futures", Tier: 2, Ratio: 0, MainRatio: 0, UpdatedAt: types.Time(time.Unix(1728230400, 0))},
		}
		assert.Equal(t, exp, result, "GetUserTransactionRateLimitInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUserTransactionRateLimitInfo should return rate limits")
}

func TestCreateSelfTradePreventionUserGroup(t *testing.T) {
	t.Parallel()
	_, err := e.CreateSelfTradePreventionUserGroup(t.Context(), "")
	require.ErrorIs(t, err, errSTPGroupNameRequired, "CreateSelfTradePreventionUserGroup must reject an empty name")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateSelfTradePreventionUserGroup(t.Context(), "stp_groups")
	require.NoError(t, err, "CreateSelfTradePreventionUserGroup must not error")
	if mockTests {
		exp := &STPUserGroupResponse{ID: 123435, Name: "stp_groups", CreatorID: 10000, CreateTime: types.Time(time.Unix(1548000000, 0))}
		assert.Equal(t, exp, result, "CreateSelfTradePreventionUserGroup should decode every field")
	}
}

func TestGetUserSelfTradePreventionGroups(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUserSelfTradePreventionGroups(t.Context(), "thrasher")
	require.NoError(t, err, "GetUserSelfTradePreventionGroups must not error")
	if mockTests {
		exp := []*STPUserGroupResponse{{ID: 123435, Name: "thrasher", CreatorID: 10000, CreateTime: types.Time(time.Unix(1548000000, 0))}}
		assert.Equal(t, exp, result, "GetUserSelfTradePreventionGroups should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUserSelfTradePreventionGroups should return groups")
}

func TestGetUsersInSTPUserGroup(t *testing.T) {
	t.Parallel()
	_, err := e.GetUsersInSTPUserGroup(t.Context(), 0)
	require.ErrorIs(t, err, errSTPGroupIDRequired, "GetUsersInSTPUserGroup must reject an empty group ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUsersInSTPUserGroup(t.Context(), 1)
	require.NoError(t, err, "GetUsersInSTPUserGroup must not error")
	if mockTests {
		exp := []*STPUserGroupMemberResponse{{UserID: 10000, STPID: 1, CreateTime: types.Time(time.Unix(1548000000, 0))}}
		assert.Equal(t, exp, result, "GetUsersInSTPUserGroup should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUsersInSTPUserGroup should return users")
}

func TestAddUsersToSTPUserGroup(t *testing.T) {
	t.Parallel()
	_, err := e.AddUsersToSTPUserGroup(t.Context(), 0, []uint64{12312312})
	require.ErrorIs(t, err, errSTPGroupIDRequired, "AddUsersToSTPUserGroup must reject an empty group ID")

	_, err = e.AddUsersToSTPUserGroup(t.Context(), 2123, nil)
	require.ErrorIs(t, err, errUserIDRequired, "AddUsersToSTPUserGroup must reject an empty user list")

	_, err = e.AddUsersToSTPUserGroup(t.Context(), 2123, []uint64{12312312, 0})
	require.ErrorIs(t, err, errUserIDRequired, "AddUsersToSTPUserGroup must reject an empty user ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		result, err := e.AddUsersToSTPUserGroup(t.Context(), 1, []uint64{12312312, 2132234232})
		require.NoError(t, err, "AddUsersToSTPUserGroup must not error")
		assert.NotEmpty(t, result, "AddUsersToSTPUserGroup should return the group's users")
		return
	}

	// The recorded mock server cannot match a body that is an array of numbers, so a test server checks the request
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "request method should be POST")
		assert.Equal(t, "/api/v4/account/stp_groups/1/users", r.URL.Path, "request path should name the STP group")
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "reading the request body should not error")
		assert.JSONEq(t, `[12312312,2132234232]`, string(body), "request body should list the user IDs")
		_, err = w.Write([]byte(`[{"user_id":12312312,"stp_id":1,"create_time":1548000000},{"user_id":2132234232,"stp_id":1,"create_time":1548000001}]`))
		assert.NoError(t, err, "writing the response should not error")
	}))
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})

	result, err := ex.AddUsersToSTPUserGroup(t.Context(), 1, []uint64{12312312, 2132234232})
	require.NoError(t, err, "AddUsersToSTPUserGroup must not error")
	exp := []*STPUserGroupMemberResponse{
		{UserID: 12312312, STPID: 1, CreateTime: types.Time(time.Unix(1548000000, 0))},
		{UserID: 2132234232, STPID: 1, CreateTime: types.Time(time.Unix(1548000001, 0))},
	}
	assert.Equal(t, exp, result, "AddUsersToSTPUserGroup should decode every field")
}

func TestDeleteUserFromSTPUserGroup(t *testing.T) {
	t.Parallel()
	_, err := e.DeleteUserFromSTPUserGroup(t.Context(), 0, []uint64{12312312})
	require.ErrorIs(t, err, errSTPGroupIDRequired, "DeleteUserFromSTPUserGroup must reject an empty group ID")

	_, err = e.DeleteUserFromSTPUserGroup(t.Context(), 2123, nil)
	require.ErrorIs(t, err, errUserIDRequired, "DeleteUserFromSTPUserGroup must reject an empty user list")

	_, err = e.DeleteUserFromSTPUserGroup(t.Context(), 2123, []uint64{0})
	require.ErrorIs(t, err, errUserIDRequired, "DeleteUserFromSTPUserGroup must reject an empty user ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.DeleteUserFromSTPUserGroup(t.Context(), 1, []uint64{12312312, 2132234232})
	require.NoError(t, err, "DeleteUserFromSTPUserGroup must not error")
	if mockTests {
		exp := []*STPUserGroupMemberResponse{{UserID: 10000, STPID: 1, CreateTime: types.Time(time.Unix(1548000000, 0))}}
		assert.Equal(t, exp, result, "DeleteUserFromSTPUserGroup should decode every field")
	}
}

func TestConfigureGTFeeDeduction(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err := e.ConfigureGTFeeDeduction(t.Context(), false)
	assert.NoError(t, err, "ConfigureGTFeeDeduction should not error")
}

func TestGetGTFeeDeductionConfiguration(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	enabled, err := e.GetGTFeeDeductionConfiguration(t.Context())
	require.NoError(t, err, "GetGTFeeDeductionConfiguration must not error")
	if mockTests {
		assert.True(t, enabled, "GetGTFeeDeductionConfiguration should decode the setting")
	}
}

func TestPlaceMultiCollateralLoanOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceMultiCollateralLoanOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceMultiCollateralLoanOrder must reject a nil request")

	arg := &MultiCollateralLoanOrderRequest{BorrowAmount: 123}
	_, err = e.PlaceMultiCollateralLoanOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PlaceMultiCollateralLoanOrder must require a borrow currency")

	_, err = e.PlaceMultiCollateralLoanOrder(t.Context(), &MultiCollateralLoanOrderRequest{BorrowCurrency: currency.ETH})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "PlaceMultiCollateralLoanOrder must require a borrow amount")

	arg.BorrowCurrency = currency.ETH
	arg.OrderType = "fixed"
	_, err = e.PlaceMultiCollateralLoanOrder(t.Context(), arg)
	require.ErrorIs(t, err, errFixedRateTermRequired, "PlaceMultiCollateralLoanOrder must require a term for fixed rate orders")

	arg.FixedType = "7d"
	_, err = e.PlaceMultiCollateralLoanOrder(t.Context(), arg)
	require.ErrorIs(t, err, errFixedRateRequired, "PlaceMultiCollateralLoanOrder must require a rate for fixed rate orders")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PlaceMultiCollateralLoanOrder(t.Context(), &MultiCollateralLoanOrderRequest{BorrowCurrency: currency.ETH, BorrowAmount: 123})
	require.NoError(t, err, "PlaceMultiCollateralLoanOrder must not error")
	if mockTests {
		assert.Equal(t, &OrderIDResponse{OrderID: 10005578}, result, "PlaceMultiCollateralLoanOrder should decode the order ID")
		return
	}
	assert.NotNil(t, result, "PlaceMultiCollateralLoanOrder should return the order ID")
}

// multiCollateralLoanOrderFixture returns the multi-collateral loan order recorded for the order list and detail fixtures
func multiCollateralLoanOrderFixture() *MultiCollateralLoanOrderDetailResponse {
	return &MultiCollateralLoanOrderDetailResponse{
		OrderID:                 "10005578",
		OrderType:               "fixed",
		FixedType:               "7d",
		FixedRate:               0.00001,
		ExpireTime:              types.Time(time.Unix(1703820105, 0)),
		AutoRenew:               true,
		AutoRepay:               true,
		CurrentLTV:              0.0001004349664281,
		Status:                  "lent",
		BorrowTime:              types.Time(time.Unix(1702615021, 0)),
		TotalLeftRepayUSDT:      106.491212982,
		TotalLeftCollateralUSDT: 1060300.18,
		BorrowCurrencies: []*BorrowCurrencyInfo{
			{Currency: currency.GT, IndexPrice: 10.6491, LeftRepayPrincipal: 10, LeftRepayInterest: 0.00002, LeftRepayUSDT: 106.491212982},
		},
		CollateralCurrencies: []*CollateralCurrencyInfo{
			{Currency: currency.BTC, IndexPrice: 112794.7, LeftCollateral: 9.4, LeftCollateralUSDT: 1060270.18},
			{Currency: currency.USDT, IndexPrice: 1, LeftCollateral: 30, LeftCollateralUSDT: 30},
		},
	}
}

func TestGetMultiCollateralLoanOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct{ sortType, orderType string }{{"", "current"}, {"time_desc", ""}, {"time_desc", "current"}} {
		result, err := e.GetMultiCollateralLoanOrders(t.Context(), 1, 100, tc.sortType, tc.orderType)
		require.NoErrorf(t, err, "GetMultiCollateralLoanOrders must not error for sort %q and order type %q", tc.sortType, tc.orderType)
		if mockTests {
			assert.Equalf(t, []*MultiCollateralLoanOrderDetailResponse{multiCollateralLoanOrderFixture()}, result, "GetMultiCollateralLoanOrders should decode every field for sort %q and order type %q", tc.sortType, tc.orderType)
			continue
		}
		assert.NotNilf(t, result, "GetMultiCollateralLoanOrders should return a result for sort %q and order type %q", tc.sortType, tc.orderType)
	}
}

var (
	pairMap    = map[asset.Item]currency.Pairs{}
	pairsGuard sync.RWMutex
)

func getPair(tb testing.TB, a asset.Item) currency.Pair {
	tb.Helper()
	ensureTestPairs(tb)
	return enabledAssetPair[a]
}

func getPairWithOpenInterest(t *testing.T, a asset.Item, pairs currency.Pairs) (key.PairAsset, []futures.OpenInterest) {
	t.Helper()
	var lastErr error
	var receivedResponse bool
	for _, pair := range pairs {
		pairAsset := key.PairAsset{Base: pair.Base.Item, Quote: pair.Quote.Item, Asset: a}
		response, err := e.GetOpenInterest(t.Context(), pairAsset)
		if err != nil {
			lastErr = err
			continue
		}
		receivedResponse = true
		if len(response) == 1 && response[0].OpenInterest > 0 {
			return pairAsset, response
		}
	}
	switch {
	case !receivedResponse:
		t.Fatalf("GetOpenInterest must find a live pair for %s asset: every request failed, last error: %v", a, lastErr)
	case lastErr != nil:
		t.Fatalf("GetOpenInterest must find a live pair with positive open interest for %s asset: no pair returned positive open interest and some requests failed, last error: %v", a, lastErr)
	default:
		t.Fatalf("GetOpenInterest must find a live pair with positive open interest for %s asset: no request errored, but no pair returned positive open interest", a)
	}
	return key.PairAsset{}, nil
}

// getMarketDataPair returns the instance and pair a market data test reads. Live delivery tests use the most active
// contract with the data the test asserts on, rather than the enabled one, because most of Gate's delivery contracts can
// go hours without a trade, often with one side of the book empty, and the enabled contract is picked at random once the
// configured one expires. That contract gets an instance of its own because candle requests need it enabled, and
// enabling it on e would not last: testexch.UpdatePairsOnce reloads e's cached pairs. Mock tests read the recorded pair
func getMarketDataPair(t *testing.T, ex *Exchange, a asset.Item, hasData deliveryDataCheck) (*Exchange, currency.Pair) {
	t.Helper()
	if mockTests || a != asset.DeliveryFutures {
		return ex, getPair(t, a)
	}
	pair := getActiveDeliveryPair(t, hasData)
	deliveryEx := new(Exchange)
	require.NoError(t, testexch.Setup(deliveryEx), "Setup must not error")
	deliveryEx.Name = t.Name()
	require.NoError(t, deliveryEx.CurrencyPairs.StorePairs(a, currency.Pairs{pair}, false), "StorePairs must not error for available pairs")
	require.NoError(t, deliveryEx.CurrencyPairs.StorePairs(a, currency.Pairs{pair}, true), "StorePairs must not error for enabled pairs")
	return deliveryEx, pair
}

// deliveryDataCheck returns why a delivery contract lacks the data a test asserts on, or nil if it has it. Each test
// checks only its own need, because a contract with a two-sided book can go hours without a trade while another trades
// with one side of its book empty
type deliveryDataCheck func(context.Context, currency.Pair) error

func hasTwoSidedDeliveryBook(ctx context.Context, p currency.Pair) error {
	book, err := e.GetDeliveryOrderbook(ctx, currency.USDT, "", p, 1, false)
	switch {
	case err != nil:
		return err
	case book == nil || len(book.Bids) == 0:
		return errors.New("no bids")
	case len(book.Asks) == 0:
		return errors.New("no asks")
	}
	return nil
}

// hasDeliveryCandles checks with the test's own interval and window, because Gate returns no candles unless a trade
// falls in a candle the window overlaps, and a contract's 24-hour volume does not show when it traded
func hasDeliveryCandles(interval kline.Interval, start, end time.Time) deliveryDataCheck {
	return func(ctx context.Context, p currency.Pair) error {
		candles, err := e.GetDeliveryFuturesCandlesticks(ctx, currency.USDT, p, start, end, 0, interval)
		if err == nil && len(candles) == 0 {
			return errors.New("no candles")
		}
		return err
	}
}

// getActiveDeliveryPair returns the contract with the most 24-hour volume that passes hasData. It selects afresh on
// each call, as each test checks for different data and a contract's book and recent trades can change between tests
func getActiveDeliveryPair(tb testing.TB, hasData deliveryDataCheck) currency.Pair {
	tb.Helper()
	tickers, err := e.GetDeliveryFutureTickers(tb.Context(), currency.USDT, currency.EMPTYPAIR)
	require.NoError(tb, err, "GetDeliveryFutureTickers must not error")
	slices.SortFunc(tickers, func(a, b *FuturesTickerResponse) int {
		return cmp.Compare(b.Volume24HourQuote.Float64(), a.Volume24HourQuote.Float64())
	})
	// A contract can expire mid-selection, so a failed request moves on to the next one
	var errs error
	for i := range tickers {
		if err := hasData(tb.Context(), tickers[i].Contract); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%s: %w", tickers[i].Contract, err))
			continue
		}
		return tickers[i].Contract
	}
	require.FailNowf(tb, "must find a delivery contract with the data this test asserts on", "%d contracts checked: %v", len(tickers), errs)
	return currency.EMPTYPAIR
}

func getPairs(tb testing.TB, a asset.Item) currency.Pairs {
	tb.Helper()
	pairsGuard.RLock()
	p, ok := pairMap[a]
	pairsGuard.RUnlock()
	if ok {
		return p
	}
	pairsGuard.Lock()
	defer pairsGuard.Unlock()
	p, ok = pairMap[a] // Protect Race if we blocked on Lock and another RW populated
	if ok {
		return p
	}

	if !mockTests {
		testexch.UpdatePairsOnce(tb, e)
	}
	enabledPairs, err := e.GetEnabledPairs(a)
	assert.NoErrorf(tb, err, "%s GetEnabledPairs should not error", a)
	if !assert.NotEmptyf(tb, enabledPairs, "%s GetEnabledPairs should not be empty", a) {
		tb.Fatalf("No pair available for asset %s", a)
		return nil
	}
	pairMap[a] = enabledPairs

	return enabledPairs
}

// BenchmarkTimeInForceFromString-8         1528101               767.3 ns/op             0 B/op          0 allocs/op
func BenchmarkTimeInForceFromString(b *testing.B) {
	for b.Loop() {
		for _, tifString := range []string{gtcTIF, iocTIF, pocTIF, fokTIF} {
			_, err := timeInForceFromString(tifString)
			require.NoError(b, err)
		}
	}
}

func TestTimeInForceFromString(t *testing.T) {
	t.Parallel()
	_, err := timeInForceFromString("abcdef")
	assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce)

	for k, v := range map[string]order.TimeInForce{gtcTIF: order.GoodTillCancel, iocTIF: order.ImmediateOrCancel, pocTIF: order.PostOnly, fokTIF: order.FillOrKill} {
		t.Run(k, func(t *testing.T) {
			t.Parallel()
			tif, err := timeInForceFromString(k)
			require.NoError(t, err)
			assert.Equal(t, v, tif)
		})
	}
}

func TestGetTypeFromTimeInForce(t *testing.T) {
	t.Parallel()
	typeResp := getTypeFromTimeInForce("gtc", 0)
	assert.Equal(t, order.Limit, typeResp)

	typeResp = getTypeFromTimeInForce("ioc", 0)
	assert.Equal(t, order.Market, typeResp, "should be market order")

	typeResp = getTypeFromTimeInForce("poc", 123)
	assert.Equal(t, order.Limit, typeResp, "should be limit order")

	typeResp = getTypeFromTimeInForce("fok", 0)
	assert.Equal(t, order.Market, typeResp, "should be market order")
}

// TestOrderStatusFromFinishAs covers every finish_as value Gate documents for spot, futures, delivery and options orders
func TestOrderStatusFromFinishAs(t *testing.T) {
	t.Parallel()
	for finishAs, exp := range map[string]order.Status{
		"":                        order.Open,
		"open":                    order.Open,
		"filled":                  order.Filled,
		"cancelled":               order.Cancelled,
		"liquidated":              order.Liquidated,
		"liquidate_cancelled":     order.Cancelled,
		"auto_deleveraged":        order.AutoDeleverage,
		"auto_deleveraging":       order.AutoDeleverage,
		"position_closed":         order.Closed,
		"position_close":          order.Closed,
		"unified_check_failed":    order.Cancelled,
		"-":                       order.UnknownStatus,
		"stp":                     order.STP,
		"ioc":                     order.Cancelled,
		"poc":                     order.Cancelled,
		"fok":                     order.Cancelled,
		"small":                   order.Cancelled,
		"depth_not_enough":        order.Cancelled,
		"trader_not_enough":       order.Cancelled,
		"price_protect_cancelled": order.Cancelled,
		"reduce_only":             order.Cancelled,
		"reduce_out":              order.Cancelled,
		"mmp_cancelled":           order.Cancelled,
		"unknown":                 order.UnknownStatus,
	} {
		got, err := orderStatusFromFinishAs(finishAs)
		require.NoErrorf(t, err, "orderStatusFromFinishAs must not error for %q", finishAs)
		assert.Equalf(t, exp, got, "orderStatusFromFinishAs should return the status for %q", finishAs)
	}
	_, err := orderStatusFromFinishAs("stopped")
	assert.ErrorContains(t, err, "STOPPED", "orderStatusFromFinishAs should reject an undocumented value")
}

func TestFuturesOrderStatus(t *testing.T) {
	t.Parallel()
	got, err := futuresOrderStatus(statusOpen, "filled")
	require.NoError(t, err, "futuresOrderStatus must not error for an open order")
	assert.Equal(t, order.Open, got, "futuresOrderStatus should report an open order as open")

	got, err = futuresOrderStatus(statusFinished, "ioc")
	require.NoError(t, err, "futuresOrderStatus must not error for a finished order")
	assert.Equal(t, order.Cancelled, got, "futuresOrderStatus should report how a finished order finished")
}

func TestOrderLastUpdated(t *testing.T) {
	t.Parallel()
	updated, finished, created := types.Time(time.Unix(3, 0)), types.Time(time.Unix(2, 0)), types.Time(time.Unix(1, 0))
	assert.Equal(t, time.Unix(3, 0), orderLastUpdated(updated, finished, created), "orderLastUpdated should prefer the update time")
	assert.Equal(t, time.Unix(2, 0), orderLastUpdated(types.Time{}, finished, created), "orderLastUpdated should fall back to the finish time")
	assert.Equal(t, time.Unix(1, 0), orderLastUpdated(types.Time{}, types.Time{}, created), "orderLastUpdated should fall back to the creation time")
	assert.Zero(t, orderLastUpdated(types.Time{}, types.Time{}, types.Time{}), "orderLastUpdated should return a zero time without any time")
}

func TestSpotOrderDetail(t *testing.T) {
	t.Parallel()
	created, updated := time.UnixMilli(1744103854944), time.UnixMilli(1744103855944)
	pair := currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter)
	newOrder := func() *SpotOrderResponse {
		return &SpotOrderResponse{
			OrderID:          "1852454420",
			Text:             "t-1852454420",
			Side:             "buy",
			Type:             "limit",
			Status:           "open",
			Price:            65000,
			Amount:           0.75,
			RemainingAmount:  0.25,
			AverageFillPrice: 64990,
			FilledTotal:      32495,
			FeeDeducted:      0.0005,
			FeeCurrency:      currency.BTC,
			CreateTime:       types.Time(created),
			UpdateTime:       types.Time(updated),
		}
	}
	got, err := e.spotOrderDetail(newOrder(), pair, asset.Margin)
	require.NoError(t, err, "spotOrderDetail must not error")
	exp := &order.Detail{
		Exchange:             e.Name,
		OrderID:              "1852454420",
		ClientOrderID:        "t-1852454420",
		Side:                 order.Buy,
		Type:                 order.Limit,
		Status:               order.Open,
		Pair:                 pair,
		AssetType:            asset.Margin,
		Price:                65000,
		Amount:               0.75,
		ExecutedAmount:       0.5,
		RemainingAmount:      0.25,
		AverageExecutedPrice: 64990,
		Cost:                 32495,
		CostAsset:            currency.USDT,
		Fee:                  0.0005,
		FeeAsset:             currency.BTC,
		Date:                 created,
		LastUpdated:          updated,
	}
	assert.Equal(t, exp, got, "spotOrderDetail should convert every field")

	unified := newOrder()
	unified.TradeQuote = currency.USDC
	got, err = e.spotOrderDetail(unified, pair, asset.Margin)
	require.NoError(t, err, "spotOrderDetail must not error for a unified account order")
	exp.CostAsset = currency.USDC
	assert.Equal(t, exp, got, "spotOrderDetail should report the cost in the quote currency the order traded in")

	badSide := newOrder()
	badSide.Side = "sideways"
	_, err = e.spotOrderDetail(badSide, pair, asset.Spot)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "spotOrderDetail should reject an unknown side")

	badType := newOrder()
	badType.Type = "pegged"
	_, err = e.spotOrderDetail(badType, pair, asset.Spot)
	assert.ErrorIs(t, err, order.ErrUnrecognisedOrderType, "spotOrderDetail should reject an unknown type")

	badStatus := newOrder()
	badStatus.Status = "stopped"
	_, err = e.spotOrderDetail(badStatus, pair, asset.Spot)
	assert.ErrorContains(t, err, "STOPPED", "spotOrderDetail should reject an unknown status")
}

func TestFuturesOrderDetail(t *testing.T) {
	t.Parallel()
	created, updated, finished := time.Unix(1744103854, 0), time.Unix(1744103855, 0), time.Unix(1744103856, 0)
	contract := currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter)
	format := currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	for _, tc := range []struct {
		name   string
		a      asset.Item
		settle currency.Code
		order  *FuturesOrderResponse
		exp    *order.Detail
	}{
		{
			name:   "cancelled short limit",
			a:      asset.USDTMarginedFutures,
			settle: currency.USDT,
			order: &FuturesOrderResponse{
				ID:              15675394,
				Contract:        contract,
				CreateTime:      types.Time(created),
				FinishTime:      types.Time(finished),
				Size:            -100,
				RemainingAmount: -25,
				Price:           65000,
				FillPrice:       64990,
				TimeInForce:     gtcTIF,
				IsReduceOnly:    true,
				Text:            "t-15675394",
				Status:          statusFinished,
				FinishAs:        "cancelled",
			},
			exp: &order.Detail{
				Exchange:             e.Name,
				OrderID:              "15675394",
				ClientOrderID:        "t-15675394",
				Side:                 order.Short,
				Type:                 order.Limit,
				Status:               order.Cancelled,
				TimeInForce:          order.GoodTillCancel,
				ReduceOnly:           true,
				Pair:                 currency.NewPairWithDelimiter("BTC", "USDT", currency.DashDelimiter),
				AssetType:            asset.USDTMarginedFutures,
				SettlementCurrency:   currency.USDT,
				Price:                65000,
				Amount:               100,
				ContractAmount:       100,
				ExecutedAmount:       75,
				RemainingAmount:      25,
				AverageExecutedPrice: 64990,
				Date:                 created,
				LastUpdated:          finished,
			},
		},
		{
			name:   "open long market",
			a:      asset.DeliveryFutures,
			settle: currency.USDT,
			order: &FuturesOrderResponse{
				ID:              15675395,
				Contract:        contract,
				CreateTime:      types.Time(created),
				UpdateTime:      types.Time(updated),
				Size:            10,
				RemainingAmount: 10,
				TimeInForce:     iocTIF,
				Text:            "web",
				Status:          statusOpen,
			},
			exp: &order.Detail{
				Exchange:           e.Name,
				OrderID:            "15675395",
				Side:               order.Long,
				Type:               order.Market,
				Status:             order.Open,
				TimeInForce:        order.ImmediateOrCancel,
				Pair:               currency.NewPairWithDelimiter("BTC", "USDT", currency.DashDelimiter),
				AssetType:          asset.DeliveryFutures,
				SettlementCurrency: currency.USDT,
				Amount:             10,
				ContractAmount:     10,
				RemainingAmount:    10,
				Date:               created,
				LastUpdated:        updated,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := e.futuresOrderDetail(tc.order, tc.a, tc.settle, format)
			require.NoError(t, err, "futuresOrderDetail must not error")
			assert.Equal(t, tc.exp, got, "futuresOrderDetail should convert every field")
		})
	}

	_, err := e.futuresOrderDetail(&FuturesOrderResponse{Status: statusFinished, FinishAs: "stopped"}, asset.USDTMarginedFutures, currency.USDT, format)
	assert.ErrorContains(t, err, "STOPPED", "futuresOrderDetail should reject an unknown finish_as")

	_, err = e.futuresOrderDetail(&FuturesOrderResponse{Status: statusOpen, TimeInForce: "gtd"}, asset.USDTMarginedFutures, currency.USDT, format)
	assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "futuresOrderDetail should reject an unknown time in force")
}

func TestOptionsOrderDetail(t *testing.T) {
	t.Parallel()
	created, finished := time.Unix(1744103854, 0), time.Unix(1744103856, 0)
	got, err := e.optionsOrderDetail(&OptionOrderResponse{
		OptionOrderID: 2187,
		Contract:      "BTC_USDT-20260925-75000-P",
		CreateTime:    types.Time(created),
		FinishTime:    types.Time(finished),
		Status:        statusFinished,
		FinishAs:      "filled",
		Size:          -2,
		Price:         1500,
		FillPrice:     1490,
		TimeInForce:   iocTIF,
		IsReduceOnly:  true,
		Text:          "t-2187",
	})
	require.NoError(t, err, "optionsOrderDetail must not error")
	exp := &order.Detail{
		Exchange:             e.Name,
		OrderID:              "2187",
		ClientOrderID:        "t-2187",
		Side:                 order.Short,
		Type:                 order.Limit,
		Status:               order.Filled,
		TimeInForce:          order.ImmediateOrCancel,
		ReduceOnly:           true,
		Pair:                 currency.NewPairWithDelimiter("BTC", "USDT-20260925-75000-P", currency.UnderscoreDelimiter),
		AssetType:            asset.Options,
		Price:                1500,
		Amount:               2,
		ExecutedAmount:       2,
		AverageExecutedPrice: 1490,
		Date:                 created,
		LastUpdated:          finished,
	}
	assert.Equal(t, exp, got, "optionsOrderDetail should convert every field")

	_, err = e.optionsOrderDetail(&OptionOrderResponse{Status: statusOpen})
	assert.ErrorIs(t, err, currency.ErrCreatingPair, "optionsOrderDetail should reject a missing contract")

	_, err = e.optionsOrderDetail(&OptionOrderResponse{Contract: "BTC_USDT-20260925-75000-P", Status: statusFinished, FinishAs: "stopped"})
	assert.ErrorContains(t, err, "STOPPED", "optionsOrderDetail should reject an unknown finish_as")

	_, err = e.optionsOrderDetail(&OptionOrderResponse{Contract: "BTC_USDT-20260925-75000-P", Status: statusOpen, TimeInForce: "gtd"})
	assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "optionsOrderDetail should reject an unknown time in force")
}

func TestDeriveFuturesSubmitResponse(t *testing.T) {
	t.Parallel()
	created, updated := time.Unix(1744103854, 0), time.Unix(1744103855, 0)
	pair := currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter)
	s := &order.Submit{
		Exchange:      e.Name,
		Type:          order.Limit,
		Side:          order.Short,
		Pair:          pair,
		AssetType:     asset.USDTMarginedFutures,
		TimeInForce:   order.GoodTillCancel,
		Price:         65000,
		Amount:        100,
		ClientOrderID: "t-15675394",
	}
	placed := &FuturesOrderResponse{
		ID:              15675394,
		Contract:        pair,
		CreateTime:      types.Time(created),
		UpdateTime:      types.Time(updated),
		Size:            -100,
		RemainingAmount: -40,
		Price:           65000,
		FillPrice:       64995,
		TimeInForce:     gtcTIF,
		Text:            "t-15675394",
		Status:          statusOpen,
	}
	got, err := deriveFuturesSubmitResponse(s, placed)
	require.NoError(t, err, "deriveFuturesSubmitResponse must not error")
	exp := &order.SubmitResponse{
		Exchange:             e.Name,
		Type:                 order.Limit,
		Side:                 order.Short,
		Pair:                 pair,
		AssetType:            asset.USDTMarginedFutures,
		TimeInForce:          order.GoodTillCancel,
		Price:                65000,
		Amount:               100,
		RemainingAmount:      40,
		AverageExecutedPrice: 64995,
		ClientOrderID:        "t-15675394",
		OrderID:              "15675394",
		Status:               order.Open,
		Date:                 created,
		LastUpdated:          updated,
	}
	assert.Equal(t, exp, got, "deriveFuturesSubmitResponse should report the placed order")

	_, err = deriveFuturesSubmitResponse(s, &FuturesOrderResponse{ID: 15675394, Status: statusFinished, FinishAs: "stopped"})
	assert.ErrorContains(t, err, "STOPPED", "deriveFuturesSubmitResponse should reject an unknown finish_as")
}

func TestToExchangeTIF(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		tif      order.TimeInForce
		price    float64
		expected string
		err      error
	}{
		{price: 0, expected: iocTIF}, // market orders default to IOC
		{price: 0, tif: order.FillOrKill, expected: fokTIF},
		{price: 420, expected: gtcTIF}, // limit orders default to GTC
		{price: 420, tif: order.GoodTillCancel, expected: gtcTIF},
		{price: 420, tif: order.ImmediateOrCancel, expected: iocTIF},
		{price: 420, tif: order.PostOnly, expected: pocTIF},
		{price: 420, tif: order.FillOrKill, expected: fokTIF},
		{tif: order.GoodTillTime, err: order.ErrUnsupportedTimeInForce},
	} {
		t.Run(fmt.Sprintf("TIF:%q Price:'%v'", tc.tif, tc.price), func(t *testing.T) {
			t.Parallel()
			got, err := toExchangeTIF(tc.tif, tc.price)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestIsSingleOrderbookChannel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		channel  string
		expected bool
	}{
		{channel: spotOrderbookUpdateChannel, expected: true},
		{channel: spotOrderbookChannel, expected: true},
		{channel: spotOrderbookTickerChannel, expected: true},
		{channel: futuresOrderbookChannel, expected: true},
		{channel: futuresOrderbookTickerChannel, expected: true},
		{channel: futuresOrderbookUpdateChannel, expected: true},
		{channel: optionsOrderbookChannel, expected: true},
		{channel: optionsOrderbookTickerChannel, expected: true},
		{channel: optionsOrderbookUpdateChannel, expected: true},
		{channel: spotTickerChannel, expected: false},
		{channel: "sad", expected: false},
	} {
		assert.Equal(t, tc.expected, isSingleOrderbookChannel(tc.channel))
	}
}

func TestOrderCreateParamsValidateNilGuard(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, (*FuturesOrderCreateRequest)(nil).validate(true), common.ErrNilPointer)
	require.ErrorIs(t, (*DeliveryOrderCreateRequest)(nil).validate(true), common.ErrNilPointer)
}

// TestOrderCreateParamsValidateClosePosition checks that a futures order may close a position with a zero size,
// which Gate documents for single position mode as close with a size of 0
func TestOrderCreateParamsValidateClosePosition(t *testing.T) {
	t.Parallel()
	futuresContract := currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter)
	deliveryContract := currency.NewPairWithDelimiter("BTC", "USDT_20260626", currency.UnderscoreDelimiter)
	for _, tc := range []struct {
		name      string
		close     bool
		autoSize  string
		size      types.Number
		expectErr error
	}{
		{name: "close with a zero size", close: true},
		{name: "close one side of a dual mode position", autoSize: "close_long"},
		{name: "close with a size", close: true, size: 1, expectErr: errInvalidOrderSize},
		{name: "zero size without closing", expectErr: errInvalidOrderSize},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for _, isRest := range []bool{true, false} {
				perpetual := &FuturesOrderCreateRequest{Contract: futuresContract, Size: tc.size, TimeInForce: iocTIF, ClosePosition: tc.close, AutoSize: tc.autoSize, ReduceOnly: true, Settle: currency.USDT}
				assert.ErrorIsf(t, perpetual.validate(isRest), tc.expectErr, "futures validate should return the expected error when isRest is %t", isRest)
				delivery := &DeliveryOrderCreateRequest{Contract: deliveryContract, Size: tc.size, TimeInForce: iocTIF, ClosePosition: tc.close, AutoSize: tc.autoSize, ReduceOnly: true, Settle: currency.USDT}
				assert.ErrorIsf(t, delivery.validate(isRest), tc.expectErr, "delivery validate should return the expected error when isRest is %t", isRest)
			}
		})
	}
}

func TestValidateSubscriptions(t *testing.T) {
	t.Parallel()
	require.NoError(t, e.ValidateSubscriptions(nil))
	require.NoError(t, e.ValidateSubscriptions([]*subscription.Subscription{{Channel: spotTickerChannel, Pairs: []currency.Pair{currency.NewBTCUSDT()}}}))
	require.NoError(t, e.ValidateSubscriptions([]*subscription.Subscription{
		{Channel: spotTickerChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
		{Channel: spotOrderbookUpdateChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
	}))
	require.NoError(t, e.ValidateSubscriptions([]*subscription.Subscription{
		{Channel: spotTickerChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
		{Channel: spotOrderbookUpdateChannel, Pairs: []currency.Pair{currency.NewBTCUSD(), currency.NewBTCUSDT()}},
	}))
	require.NoError(t, e.ValidateSubscriptions([]*subscription.Subscription{
		{Channel: spotTickerChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
		{Channel: spotOrderbookUpdateChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
		{Channel: spotOrderbookUpdateChannel, Pairs: []currency.Pair{currency.NewBTCUSDT()}},
	}))
	require.ErrorIs(t, e.ValidateSubscriptions([]*subscription.Subscription{
		{Channel: spotTickerChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
		{Channel: spotOrderbookUpdateChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
		{Channel: spotOrderbookChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
	}), subscription.ErrExclusiveSubscription)
}

func TestCandlesChannelIntervals(t *testing.T) {
	t.Parallel()
	s := &subscription.Subscription{Channel: subscription.CandlesChannel, Asset: asset.Spot, Interval: 0}
	_, err := candlesChannelInterval(s)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "candlestickChannelInterval must error correctly with a 0 interval")
	s.Interval = kline.ThousandMilliseconds
	i, err := candlesChannelInterval(s)
	require.NoError(t, err)
	assert.Equal(t, "1000ms", i)
}

func TestOrderbookChannelIntervals(t *testing.T) {
	t.Parallel()

	s := &subscription.Subscription{Channel: futuresOrderbookUpdateChannel, Interval: kline.TwentyMilliseconds, Levels: 100}
	_, err := orderbookChannelInterval(s, asset.Futures)
	require.ErrorIs(t, err, subscription.ErrInvalidInterval)
	require.ErrorContains(t, err, "20ms only valid with Levels 20")
	s.Levels = 20
	i, err := orderbookChannelInterval(s, asset.Futures)
	require.NoError(t, err)
	assert.Equal(t, "20ms", i)

	for s, exp := range map[*subscription.Subscription]error{
		{Asset: asset.Binary, Channel: "unknown_channel", Interval: kline.OneYear}:                    nil,
		{Asset: asset.Spot, Channel: spotOrderbookTickerChannel, Interval: kline.OneDay}:              subscription.ErrInvalidInterval,
		{Asset: asset.Spot, Channel: spotOrderbookTickerChannel, Interval: 0}:                         nil,
		{Asset: asset.Spot, Channel: spotOrderbookChannel, Interval: kline.OneDay}:                    subscription.ErrInvalidInterval,
		{Asset: asset.Spot, Channel: spotOrderbookChannel, Interval: kline.HundredMilliseconds}:       nil,
		{Asset: asset.Spot, Channel: spotOrderbookChannel, Interval: kline.ThousandMilliseconds}:      nil,
		{Asset: asset.Spot, Channel: spotOrderbookUpdateChannel, Interval: kline.OneDay}:              subscription.ErrInvalidInterval,
		{Asset: asset.Spot, Channel: spotOrderbookUpdateChannel, Interval: kline.HundredMilliseconds}: nil,

		{Asset: asset.Futures, Channel: futuresOrderbookTickerChannel, Interval: kline.TenMilliseconds}:     subscription.ErrInvalidInterval,
		{Asset: asset.Futures, Channel: futuresOrderbookTickerChannel, Interval: 0}:                         nil,
		{Asset: asset.Futures, Channel: futuresOrderbookChannel, Interval: kline.TenMilliseconds}:           subscription.ErrInvalidInterval,
		{Asset: asset.Futures, Channel: futuresOrderbookChannel, Interval: 0}:                               nil,
		{Asset: asset.Futures, Channel: futuresOrderbookUpdateChannel, Interval: kline.OneDay}:              subscription.ErrInvalidInterval,
		{Asset: asset.Futures, Channel: futuresOrderbookUpdateChannel, Interval: kline.HundredMilliseconds}: nil,

		{Asset: asset.DeliveryFutures, Channel: futuresOrderbookTickerChannel, Interval: kline.TenMilliseconds}:      subscription.ErrInvalidInterval,
		{Asset: asset.DeliveryFutures, Channel: futuresOrderbookTickerChannel, Interval: 0}:                          nil,
		{Asset: asset.DeliveryFutures, Channel: futuresOrderbookChannel, Interval: kline.TenMilliseconds}:            subscription.ErrInvalidInterval,
		{Asset: asset.DeliveryFutures, Channel: futuresOrderbookChannel, Interval: 0}:                                nil,
		{Asset: asset.DeliveryFutures, Channel: futuresOrderbookUpdateChannel, Interval: kline.OneDay}:               subscription.ErrInvalidInterval,
		{Asset: asset.DeliveryFutures, Channel: futuresOrderbookUpdateChannel, Interval: kline.HundredMilliseconds}:  nil,
		{Asset: asset.DeliveryFutures, Channel: futuresOrderbookUpdateChannel, Interval: kline.ThousandMilliseconds}: nil,

		{Asset: asset.Options, Channel: optionsOrderbookTickerChannel, Interval: kline.TenMilliseconds}:          subscription.ErrInvalidInterval,
		{Asset: asset.Options, Channel: optionsOrderbookTickerChannel, Interval: 0}:                              nil,
		{Asset: asset.Options, Channel: optionsOrderbookChannel, Interval: kline.TwoHundredAndFiftyMilliseconds}: subscription.ErrInvalidInterval,
		{Asset: asset.Options, Channel: optionsOrderbookChannel, Interval: 0}:                                    nil,
		{Asset: asset.Options, Channel: optionsOrderbookUpdateChannel, Interval: kline.OneDay}:                   subscription.ErrInvalidInterval,
		{Asset: asset.Options, Channel: optionsOrderbookUpdateChannel, Interval: kline.HundredMilliseconds}:      nil,
		{Asset: asset.Options, Channel: optionsOrderbookUpdateChannel, Interval: kline.ThousandMilliseconds}:     nil,
	} {
		t.Run(s.Asset.String()+"/"+s.Channel+"/"+s.Interval.Short(), func(t *testing.T) {
			t.Parallel()
			i, err := orderbookChannelInterval(s, s.Asset)
			if exp != nil {
				require.ErrorIs(t, err, exp)
			} else {
				switch {
				case s.Channel == "unknown_channel":
					assert.Empty(t, i, "orderbookChannelInterval should return empty for unknown channels")
				case strings.HasSuffix(s.Channel, "_ticker"):
					assert.Empty(t, i)
				case s.Interval == 0:
					assert.Equal(t, "0", i)
				default:
					exp, err2 := getIntervalString(s.Interval)
					require.NoError(t, err2, "getIntervalString must not error for validating expected value")
					require.Equal(t, exp, i)
				}
			}
		})
	}
}

func TestChannelLevels(t *testing.T) {
	t.Parallel()

	for s, exp := range map[*subscription.Subscription]error{
		{Channel: "unknown_channel", Asset: asset.Binary}:                                   nil,
		{Channel: spotOrderbookTickerChannel, Asset: asset.Spot}:                            nil,
		{Channel: spotOrderbookTickerChannel, Asset: asset.Spot, Levels: 1}:                 subscription.ErrInvalidLevel,
		{Channel: spotOrderbookUpdateChannel, Asset: asset.Spot}:                            nil,
		{Channel: spotOrderbookUpdateChannel, Asset: asset.Spot, Levels: 100}:               subscription.ErrInvalidLevel,
		{Channel: spotOrderbookChannel, Asset: asset.Spot}:                                  subscription.ErrInvalidLevel,
		{Channel: spotOrderbookChannel, Asset: asset.Spot, Levels: 5}:                       nil,
		{Channel: spotOrderbookChannel, Asset: asset.Spot, Levels: 10}:                      nil,
		{Channel: spotOrderbookChannel, Asset: asset.Spot, Levels: 20}:                      nil,
		{Channel: spotOrderbookChannel, Asset: asset.Spot, Levels: 50}:                      nil,
		{Channel: spotOrderbookChannel, Asset: asset.Spot, Levels: 100}:                     nil,
		{Channel: futuresOrderbookChannel, Asset: asset.Futures}:                            subscription.ErrInvalidLevel,
		{Channel: futuresOrderbookChannel, Asset: asset.Futures, Levels: 1}:                 nil,
		{Channel: futuresOrderbookChannel, Asset: asset.Futures, Levels: 5}:                 nil,
		{Channel: futuresOrderbookChannel, Asset: asset.Futures, Levels: 10}:                nil,
		{Channel: futuresOrderbookChannel, Asset: asset.Futures, Levels: 20}:                nil,
		{Channel: futuresOrderbookChannel, Asset: asset.Futures, Levels: 50}:                nil,
		{Channel: futuresOrderbookChannel, Asset: asset.Futures, Levels: 100}:               nil,
		{Channel: futuresOrderbookTickerChannel, Asset: asset.Futures}:                      nil,
		{Channel: futuresOrderbookTickerChannel, Asset: asset.Futures, Levels: 1}:           subscription.ErrInvalidLevel,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.Futures}:                      subscription.ErrInvalidLevel,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.Futures, Levels: 20}:          nil,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.Futures, Levels: 50}:          nil,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.DeliveryFutures}:              subscription.ErrInvalidLevel,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.DeliveryFutures, Levels: 5}:   nil,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.DeliveryFutures, Levels: 10}:  nil,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.DeliveryFutures, Levels: 20}:  nil,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.DeliveryFutures, Levels: 50}:  nil,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.DeliveryFutures, Levels: 100}: nil,
		{Channel: optionsOrderbookTickerChannel, Asset: asset.Options}:                      nil,
		{Channel: optionsOrderbookTickerChannel, Asset: asset.Options, Levels: 1}:           subscription.ErrInvalidLevel,
		{Channel: optionsOrderbookUpdateChannel, Asset: asset.Options}:                      subscription.ErrInvalidLevel,
		{Channel: optionsOrderbookUpdateChannel, Asset: asset.Options, Levels: 5}:           nil,
		{Channel: optionsOrderbookUpdateChannel, Asset: asset.Options, Levels: 10}:          nil,
		{Channel: optionsOrderbookUpdateChannel, Asset: asset.Options, Levels: 20}:          nil,
		{Channel: optionsOrderbookUpdateChannel, Asset: asset.Options, Levels: 50}:          nil,
		{Channel: optionsOrderbookChannel, Asset: asset.Options}:                            subscription.ErrInvalidLevel,
		{Channel: optionsOrderbookChannel, Asset: asset.Options, Levels: 5}:                 nil,
		{Channel: optionsOrderbookChannel, Asset: asset.Options, Levels: 10}:                nil,
		{Channel: optionsOrderbookChannel, Asset: asset.Options, Levels: 20}:                nil,
		{Channel: optionsOrderbookChannel, Asset: asset.Options, Levels: 50}:                nil,
	} {
		t.Run(s.Asset.String()+"/"+s.Channel+"/"+strconv.Itoa(s.Levels), func(t *testing.T) {
			t.Parallel()
			l, err := channelLevels(s, s.Asset)
			switch {
			case exp != nil:
				require.ErrorIs(t, err, exp)
			case s.Levels == 0:
				assert.Empty(t, l)
			default:
				require.NoError(t, err)
				require.NotEmpty(t, l)
			}
		})
	}
}

func TestGetIntervalString(t *testing.T) {
	t.Parallel()
	for key, exp := range map[kline.Interval]string{
		kline.TenMilliseconds:                "10ms",
		kline.TwentyMilliseconds:             "20ms",
		kline.HundredMilliseconds:            "100ms",
		kline.TwoHundredAndFiftyMilliseconds: "250ms",
		kline.ThousandMilliseconds:           "1000ms",
		kline.TenSecond:                      "10s",
		kline.ThirtySecond:                   "30s",
		kline.OneMin:                         "1m",
		kline.FiveMin:                        "5m",
		kline.FifteenMin:                     "15m",
		kline.ThirtyMin:                      "30m",
		kline.OneHour:                        "1h",
		kline.TwoHour:                        "2h",
		kline.FourHour:                       "4h",
		kline.SixHour:                        "6h",
		kline.EightHour:                      "8h",
		kline.TwelveHour:                     "12h",
		kline.OneDay:                         "1d",
		kline.SevenDay:                       "7d",
		kline.OneMonth:                       "30d",
	} {
		t.Run(exp, func(t *testing.T) {
			t.Parallel()
			s, err := getIntervalString(key)
			require.NoError(t, err, "getIntervalString must not error")
			assert.Equal(t, exp, s, "getIntervalString should return Gate's name for the interval")
		})
	}
	_, err := getIntervalString(0)
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "0 should be an invalid interval")

	_, err = getIntervalString(kline.FiveDay)
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "Any other random interval should also be invalid")
}

func TestGetIntervalFromString(t *testing.T) {
	t.Parallel()
	for str, exp := range map[string]kline.Interval{
		"10ms":  kline.TenMilliseconds,
		"20ms":  kline.TwentyMilliseconds,
		"100ms": kline.HundredMilliseconds,
		"250ms": kline.TwoHundredAndFiftyMilliseconds,
		"10s":   kline.TenSecond,
		"30s":   kline.ThirtySecond,
		"1m":    kline.OneMin,
		"5m":    kline.FiveMin,
		"15m":   kline.FifteenMin,
		"30m":   kline.ThirtyMin,
		"1h":    kline.OneHour,
		"2h":    kline.TwoHour,
		"4h":    kline.FourHour,
		"6h":    kline.SixHour,
		"8h":    kline.EightHour,
		"12h":   kline.TwelveHour,
		"1d":    kline.OneDay,
		"7d":    kline.SevenDay,
		"30d":   kline.OneMonth,
		// Input is lowercased before matching, so mixed case must resolve too
		"1H": kline.OneHour,
	} {
		t.Run(str, func(t *testing.T) {
			t.Parallel()
			interval, err := e.GetIntervalFromString(str)
			require.NoError(t, err, "GetIntervalFromString must not error")
			assert.Equal(t, exp, interval, "GetIntervalFromString should return the interval Gate's name stands for")
		})
	}

	_, err := e.GetIntervalFromString("")
	assert.ErrorIs(t, err, kline.ErrInvalidInterval, "empty interval string should be invalid")

	_, err = e.GetIntervalFromString("3w")
	assert.ErrorIs(t, err, kline.ErrInvalidInterval, "unknown interval string should be invalid")
}

func TestUIDTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.UIDTransfer(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UIDTransfer must reject a nil request")
	_, err = e.UIDTransfer(t.Context(), &UIDTransferRequest{})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "UIDTransfer must reject a zero amount")
	_, err = e.UIDTransfer(t.Context(), &UIDTransferRequest{Amount: 1.})
	require.ErrorIs(t, err, errUserIDRequired, "UIDTransfer must reject an empty recipient")

	_, err = e.UIDTransfer(t.Context(), &UIDTransferRequest{Amount: 1., RecipientUserID: 2231234})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "UIDTransfer must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.UIDTransfer(t.Context(), &UIDTransferRequest{
		Amount:          1.,
		RecipientUserID: 2231234,
		Currency:        currency.ETH,
	})
	require.NoError(t, err, "UIDTransfer must not error")
	if mockTests {
		assert.Equal(t, &UIDTransferResponse{ID: "111"}, result, "UIDTransfer should decode the order ID")
	}
}

func TestGetMultiCollateralLoanOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetMultiCollateralLoanOrderDetail(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetMultiCollateralLoanOrderDetail must require an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMultiCollateralLoanOrderDetail(t.Context(), "1234556")
	require.NoError(t, err, "GetMultiCollateralLoanOrderDetail must not error")
	if mockTests {
		exp := multiCollateralLoanOrderFixture()
		exp.OrderID = "1234556"
		assert.Equal(t, exp, result, "GetMultiCollateralLoanOrderDetail should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMultiCollateralLoanOrderDetail should return a result")
}

func TestRepayMultiCollateralLoan(t *testing.T) {
	t.Parallel()
	_, err := e.RepayMultiCollateralLoan(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "RepayMultiCollateralLoan must reject a nil request")

	_, err = e.RepayMultiCollateralLoan(t.Context(), &MultiCollateralLoanRepaymentRequest{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "RepayMultiCollateralLoan must require an order ID")

	arg := &MultiCollateralLoanRepaymentRequest{OrderID: 1234}
	_, err = e.RepayMultiCollateralLoan(t.Context(), arg)
	require.ErrorIs(t, err, errNoValidParameterPassed, "RepayMultiCollateralLoan must require repay items")

	arg.RepayItems = []LoanRepaymentMultiCollateralAssetItem{{RepaidAll: true}}
	_, err = e.RepayMultiCollateralLoan(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "RepayMultiCollateralLoan must require each repay item's currency")

	arg.RepayItems = []LoanRepaymentMultiCollateralAssetItem{{Currency: currency.USDT}}
	_, err = e.RepayMultiCollateralLoan(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "RepayMultiCollateralLoan must require an amount for partial repayments")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.RepayItems = []LoanRepaymentMultiCollateralAssetItem{{Currency: currency.USDT, RepaidAll: true}, {Currency: currency.BTC, Amount: 1}}
	result, err := e.RepayMultiCollateralLoan(t.Context(), arg)
	require.NoError(t, err, "RepayMultiCollateralLoan must not error")
	if mockTests {
		exp := &MultiCollateralLoanRepaymentResponse{
			OrderID: 1234,
			RepaidCurrencies: []*MultiCollateralRepaidCurrency{
				{Succeeded: true, Currency: currency.USDT, RepaidPrincipal: 10, RepaidInterest: 0.0001},
				{Label: "INVALID_PARAM_VALUE", Message: "Invalid parameter value", Currency: currency.BTC},
			},
		}
		assert.Equal(t, exp, result, "RepayMultiCollateralLoan should decode every field")
		return
	}
	assert.NotNil(t, result, "RepayMultiCollateralLoan should return a result")
}

func TestGetMultiCurrencyCollateralRepaymentRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetMultiCurrencyCollateralRepaymentRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMultiCurrencyCollateralRepaymentRecords must reject a nil request")

	_, err = e.GetMultiCurrencyCollateralRepaymentRecords(t.Context(), &MultiCollateralRepaymentRecordsRequest{BorrowCurrency: currency.USDT, Limit: 10})
	require.ErrorIs(t, err, errLoanTypeIsRequired, "GetMultiCurrencyCollateralRepaymentRecords must require an operation type")

	startTime, endTime := getTime()
	_, err = e.GetMultiCurrencyCollateralRepaymentRecords(t.Context(), &MultiCollateralRepaymentRecordsRequest{Type: "repay", From: endTime, To: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMultiCurrencyCollateralRepaymentRecords must reject a reversed time range")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMultiCurrencyCollateralRepaymentRecords(t.Context(), &MultiCollateralRepaymentRecordsRequest{Type: "repay", BorrowCurrency: currency.USDT, Limit: 10})
	require.NoError(t, err, "GetMultiCurrencyCollateralRepaymentRecords must not error")
	if mockTests {
		exp := []*MultiCurrencyCollateralRepaymentResponse{{
			OrderID:    10005679,
			RecordID:   1348,
			InitLTV:    0.2141,
			BeforeLTV:  0.215,
			AfterLTV:   0.312,
			BorrowTime: types.Time(time.Unix(1702995889, 0)),
			RepayTime:  types.Time(time.Unix(1703053927, 0)),
			BorrowCurrencies: []*CurrencyDetail{
				{Currency: currency.USDT, IndexPrice: 1, BeforeAmount: 1, BeforeAmountUSDT: 1, AfterAmount: 0.999017, AfterAmountUSDT: 0.999017},
			},
			CollateralCurrencies: []*CurrencyDetail{
				{Currency: currency.ETC, IndexPrice: 0.6014228107, BeforeAmount: 1000, BeforeAmountUSDT: 601.4228107, AfterAmount: 1000, AfterAmountUSDT: 601.4228107},
			},
			RepaidCurrencies: []*RepayRecordRepaidCurrency{
				{Currency: currency.USDT, IndexPrice: 1, RepaidAmount: 0.001, RepaidPrincipal: 0.000983, RepaidInterest: 0.000017, RepaidAmountUSDT: 0.001},
			},
			TotalInterestList: []*RepayRecordTotalInterest{
				{Currency: currency.USDT, IndexPrice: 1, Amount: 0.000017, AmountUSDT: 0.000017},
			},
			LeftRepayInterestList: []*CurrencyDetail{
				{Currency: currency.USDT, IndexPrice: 1, BeforeAmount: 0.000017, BeforeAmountUSDT: 0.000017},
			},
		}}
		assert.Equal(t, exp, result, "GetMultiCurrencyCollateralRepaymentRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMultiCurrencyCollateralRepaymentRecords should return a result")
}

func TestAddOrWithdrawCollateral(t *testing.T) {
	t.Parallel()
	_, err := e.AddOrWithdrawCollateral(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AddOrWithdrawCollateral must reject a nil request")

	_, err = e.AddOrWithdrawCollateral(t.Context(), &AddOrWithdrawCollateralRequest{OperationType: "redeem"})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "AddOrWithdrawCollateral must require an order ID")

	_, err = e.AddOrWithdrawCollateral(t.Context(), &AddOrWithdrawCollateralRequest{OrderID: 12344})
	require.ErrorIs(t, err, errLoanTypeIsRequired, "AddOrWithdrawCollateral must require an operation type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AddOrWithdrawCollateral(t.Context(), &AddOrWithdrawCollateralRequest{
		OrderID:       12344,
		OperationType: "redeem",
		Collaterals:   []CurrencyAndAmountResponse{{Currency: currency.BTC, Amount: 0.5}},
	})
	require.NoError(t, err, "AddOrWithdrawCollateral must not error")
	if mockTests {
		exp := &CollateralAddOrRemoveResponse{
			OrderID:              12344,
			CollateralCurrencies: []*CollateralOperationResult{{Succeeded: true, Currency: currency.BTC, Amount: 0.5}},
		}
		assert.Equal(t, exp, result, "AddOrWithdrawCollateral should decode every field")
		return
	}
	assert.NotNil(t, result, "AddOrWithdrawCollateral should return a result")
}

func TestGetMultiCollateralAdjustmentRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetMultiCollateralAdjustmentRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMultiCollateralAdjustmentRecords must reject a nil request")

	startTime, endTime := getTime()
	_, err = e.GetMultiCollateralAdjustmentRecords(t.Context(), &MultiCollateralAdjustmentRecordsRequest{Page: 1, Limit: 10, From: endTime, To: startTime, CollateralCurrency: currency.ETH})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMultiCollateralAdjustmentRecords must reject a reversed time range")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMultiCollateralAdjustmentRecords(t.Context(), &MultiCollateralAdjustmentRecordsRequest{Page: 1, Limit: 10, From: startTime, To: endTime, CollateralCurrency: currency.ETH})
	require.NoError(t, err, "GetMultiCollateralAdjustmentRecords must not error")
	if mockTests {
		exp := []*MultiCollateralAdjustmentRecordResponse{{
			OrderID:     10000417,
			RecordID:    10000452,
			BeforeLTV:   0.0003934555562148,
			AfterLTV:    0.0001967277781074,
			OperateTime: types.Time(time.Unix(1744150000, 0)),
			BorrowCurrencies: []*CurrencyDetail{
				{Currency: currency.USDT, IndexPrice: 1, BeforeAmount: 1000, BeforeAmountUSDT: 1000, AfterAmount: 1000, AfterAmountUSDT: 1000},
			},
			CollateralCurrencies: []*CurrencyDetail{
				{Currency: currency.ETH, IndexPrice: 1500, BeforeAmount: 0.5, BeforeAmountUSDT: 750, AfterAmount: 1, AfterAmountUSDT: 1500},
			},
		}}
		assert.Equal(t, exp, result, "GetMultiCollateralAdjustmentRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMultiCollateralAdjustmentRecords should return a result")
}

func TestGetMultiCollateralCurrencyQuota(t *testing.T) {
	t.Parallel()
	_, err := e.GetMultiCollateralCurrencyQuota(t.Context(), "", []currency.Code{currency.BTC})
	require.ErrorIs(t, err, errCurrencyTypeRequired, "GetMultiCollateralCurrencyQuota must require a quota type")

	_, err = e.GetMultiCollateralCurrencyQuota(t.Context(), "collateral", nil)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMultiCollateralCurrencyQuota must require a currency")

	_, err = e.GetMultiCollateralCurrencyQuota(t.Context(), "borrow", []currency.Code{currency.BTC, currency.ETH})
	require.ErrorIs(t, err, errTooManyCurrencyCodes, "GetMultiCollateralCurrencyQuota must accept a single borrow currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMultiCollateralCurrencyQuota(t.Context(), "collateral", []currency.Code{currency.BTC})
	require.NoError(t, err, "GetMultiCollateralCurrencyQuota must not error")
	if mockTests {
		exp := []*MultiCollateralCurrencyQuotaResponse{{
			Currency:           currency.BTC,
			IndexPrice:         35306.1,
			LeftQuota:          2768152.4958445218723677,
			LeftQuotaUSDT:      97732668833.536273678,
			LeftQuotaFixed:     1000,
			LeftQuotaUSDTFixed: 35306100,
		}}
		assert.Equal(t, exp, result, "GetMultiCollateralCurrencyQuota should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMultiCollateralCurrencyQuota should return a result")
}

func TestGetMultiCollateralSupportedCurrencies(t *testing.T) {
	t.Parallel()
	result, err := e.GetMultiCollateralSupportedCurrencies(t.Context())
	require.NoError(t, err, "GetMultiCollateralSupportedCurrencies must not error")
	if mockTests {
		exp := &MultiCollateralSupportedCurrenciesResponse{
			LoanCurrencies:       []*MultiLoanItem{{Currency: currency.BTC, Price: 1212}, {Currency: currency.GT, Price: 12}},
			CollateralCurrencies: []*MultiCollateralItem{{Currency: currency.BTC, IndexPrice: 1212, Discount: 0.7}},
		}
		assert.Equal(t, exp, result, "GetMultiCollateralSupportedCurrencies should decode every field")
		return
	}
	require.NotNil(t, result, "GetMultiCollateralSupportedCurrencies must return a result")
	assert.NotEmpty(t, result.LoanCurrencies, "GetMultiCollateralSupportedCurrencies should return loan currencies")
}

func TestGetMultiCollateralisationRatio(t *testing.T) {
	t.Parallel()
	result, err := e.GetMultiCollateralisationRatio(t.Context())
	require.NoError(t, err, "GetMultiCollateralisationRatio must not error")
	if mockTests {
		assert.Equal(t, &MultiCollateralisationRatioResponse{InitLTV: 0.7, AlertLTV: 0.8, LiquidateLTV: 0.9}, result, "GetMultiCollateralisationRatio should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMultiCollateralisationRatio should return a result")
}

func TestGetMultiCollateralFixedRates(t *testing.T) {
	t.Parallel()
	result, err := e.GetMultiCollateralFixedRates(t.Context())
	require.NoError(t, err, "GetMultiCollateralFixedRates must not error")
	if mockTests {
		exp := []*CollateralFixRateResponse{{Currency: currency.BTC, Rate7Day: 0.000023, Rate30Day: 0.1, UpdateTime: types.Time(time.Unix(1703820105, 0))}}
		assert.Equal(t, exp, result, "GetMultiCollateralFixedRates should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetMultiCollateralFixedRates should return rates")
}

func TestGetMultiCollateralCurrentRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetMultiCollateralCurrentRates(t.Context(), nil, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMultiCollateralCurrentRates must require a currency")

	_, err = e.GetMultiCollateralCurrentRates(t.Context(), make([]currency.Code, 101), "")
	require.ErrorIs(t, err, errTooManyCurrencyCodes, "GetMultiCollateralCurrentRates must reject more than 100 currencies")

	result, err := e.GetMultiCollateralCurrentRates(t.Context(), []currency.Code{currency.BTC, currency.GT}, "")
	require.NoError(t, err, "GetMultiCollateralCurrentRates must not error")
	if mockTests {
		exp := []*CollateralCurrentRateResponse{{Currency: currency.BTC, CurrentRate: 0.000023}, {Currency: currency.GT, CurrentRate: 0.000015}}
		assert.Equal(t, exp, result, "GetMultiCollateralCurrentRates should decode every field")
		return
	}
	assert.Len(t, result, 2, "GetMultiCollateralCurrentRates should return a rate for each currency")
}

func TestWebsocketSubmitOrders(t *testing.T) {
	t.Parallel()

	_, err := e.WebsocketSubmitOrders(t.Context(), nil)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sub := &order.Submit{
		Exchange:    e.Name,
		AssetType:   asset.Spot,
		Side:        order.Buy,
		Type:        order.Market,
		QuoteAmount: 10,
	}
	_, err = e.WebsocketSubmitOrders(t.Context(), []*order.Submit{sub})
	require.ErrorIs(t, err, order.ErrPairIsEmpty)

	sub.Pair = currency.NewBTCUSD()
	cpy := *sub
	cpy.AssetType = asset.Futures
	_, err = e.WebsocketSubmitOrders(t.Context(), []*order.Submit{sub, &cpy})
	require.ErrorIs(t, err, errSingleAssetRequired)

	cpy.AssetType = asset.Spread
	sub.AssetType = asset.Spread
	_, err = e.WebsocketSubmitOrders(t.Context(), []*order.Submit{sub, &cpy})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sub.AssetType = asset.USDTMarginedFutures
	cpy.AssetType = asset.USDTMarginedFutures
	_, err = e.WebsocketSubmitOrders(t.Context(), []*order.Submit{sub, &cpy})
	require.ErrorIs(t, err, errInvalidOrderSize)

	// Prices are far from the market, so a live run leaves resting orders
	for _, tc := range []struct {
		a      asset.Item
		orders []*order.Submit
		exp    []*order.SubmitResponse
	}{
		{
			// Gate rejects an order of a batch on its own, reporting why in its place
			a: asset.Spot,
			orders: []*order.Submit{
				{Type: order.Limit, Side: order.Buy, Price: 10000, Amount: 0.001, ClientOrderID: "ws-spot-buy"},
				{Type: order.Limit, Side: order.Sell, Price: 200000, Amount: 0.001, ClientOrderID: "ws-spot-sell"},
			},
			exp: []*order.SubmitResponse{
				{
					Exchange:        "GateIO",
					OrderID:         "1700664332",
					AssetType:       asset.Spot,
					Pair:            BTCUSDT,
					ClientOrderID:   "t-ws-spot-buy",
					Date:            time.UnixMilli(1681986204832),
					LastUpdated:     time.UnixMilli(1681986204832),
					RemainingAmount: 0.001,
					Amount:          0.001,
					Price:           10000,
					Type:            order.Limit,
					Side:            order.Buy,
					FeeAsset:        currency.BTC,
					TimeInForce:     order.GoodTillCancel,
					Status:          order.Open,
				},
				{
					Exchange:        "GateIO",
					ClientOrderID:   "t-ws-spot-sell",
					SubmissionError: fmt.Errorf("%w reason label:%q message:%q", order.ErrUnableToPlaceOrder, "BALANCE_NOT_ENOUGH", "Not enough balance"),
				},
			},
		},
		{
			a: asset.USDTMarginedFutures,
			orders: []*order.Submit{
				{Type: order.Limit, Side: order.Buy, Price: 10000, Amount: 1, ClientOrderID: "ws-usdt-buy"},
				{Type: order.Limit, Side: order.Sell, Price: 200000, Amount: 1, ClientOrderID: "ws-usdt-sell"},
			},
			exp: []*order.SubmitResponse{
				{
					Exchange:        "GateIO",
					OrderID:         "74046545",
					AssetType:       asset.USDTMarginedFutures,
					Pair:            BTCUSDT,
					ClientOrderID:   "t-ws-usdt-buy",
					Date:            time.UnixMilli(1681196536592),
					LastUpdated:     time.UnixMilli(1681196536592),
					RemainingAmount: 1,
					Amount:          1,
					Price:           10000,
					Type:            order.Limit,
					Side:            order.Long,
					Status:          order.Open,
					TimeInForce:     order.GoodTillCancel,
				},
				{
					Exchange:        "GateIO",
					AssetType:       asset.USDTMarginedFutures,
					SubmissionError: fmt.Errorf("%w reason label:%q detail:%q", order.ErrUnableToPlaceOrder, "INSUFFICIENT_AVAILABLE", "insufficient available balance"),
				},
			},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			ex := newWsAPITestExchange(t, tc.a, canManipulateRealOrders)
			for _, o := range tc.orders {
				o.Exchange, o.AssetType, o.Pair = ex.Name, tc.a, getPair(t, tc.a)
			}
			got, err := ex.WebsocketSubmitOrders(t.Context(), tc.orders)
			require.NoError(t, err, "WebsocketSubmitOrders must not error")
			if mockTests {
				assert.Equal(t, tc.exp, got, "WebsocketSubmitOrders should return each order's outcome")
				return
			}
			assert.Len(t, got, len(tc.orders), "WebsocketSubmitOrders should return each order's outcome")
		})
	}
}

func TestWebsocketSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketSubmitOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil, "WebsocketSubmitOrder must reject a nil submission")
	_, err = e.WebsocketSubmitOrder(t.Context(), &order.Submit{Exchange: e.Name, AssetType: asset.DeliveryFutures, Pair: currency.NewBTCUSDT(), Type: order.Limit, Side: order.Buy, Price: 1, Amount: 1})
	require.ErrorIs(t, err, asset.ErrNotSupported, "WebsocketSubmitOrder must reject an asset Gate cannot trade over its WebSocket API")

	// Prices are far from the market, so a live run leaves resting orders
	for _, tc := range []struct {
		a      asset.Item
		submit *order.Submit
		exp    *order.SubmitResponse
	}{
		{
			a:      asset.Spot,
			submit: &order.Submit{Type: order.Limit, Side: order.Buy, Price: 10000, Amount: 0.001, ClientOrderID: "ws-spot"},
			exp: &order.SubmitResponse{
				Exchange:        "GateIO",
				OrderID:         "1700664333",
				AssetType:       asset.Spot,
				Pair:            BTCUSDT,
				ClientOrderID:   "t-ws-spot",
				Date:            time.UnixMilli(1681986204832),
				LastUpdated:     time.UnixMilli(1681986204832),
				RemainingAmount: 0.001,
				Amount:          0.001,
				Price:           10000,
				Type:            order.Limit,
				Side:            order.Buy,
				FeeAsset:        currency.BTC,
				TimeInForce:     order.GoodTillCancel,
				Status:          order.Open,
			},
		},
		{
			// Margin orders are placed through the spot connection
			a:      asset.Margin,
			submit: &order.Submit{Type: order.Limit, Side: order.Sell, Price: 200000, Amount: 0.001, ClientOrderID: "ws-margin"},
			exp: &order.SubmitResponse{
				Exchange:        "GateIO",
				OrderID:         "1700664334",
				AssetType:       asset.Margin,
				Pair:            BTCUSDT,
				ClientOrderID:   "t-ws-margin",
				Date:            time.UnixMilli(1681986204832),
				LastUpdated:     time.UnixMilli(1681986204832),
				RemainingAmount: 0.001,
				Amount:          0.001,
				Price:           200000,
				Type:            order.Limit,
				Side:            order.Sell,
				FeeAsset:        currency.USDT,
				TimeInForce:     order.GoodTillCancel,
				Status:          order.Open,
			},
		},
		{
			// Gate finishes an unfilled IOC order as ioc, and its futures order results carry no update time
			a:      asset.USDTMarginedFutures,
			submit: &order.Submit{Type: order.Limit, Side: order.Sell, Price: 200000, Amount: 10, TimeInForce: order.ImmediateOrCancel, ClientOrderID: "ws-usdt"},
			exp: &order.SubmitResponse{
				Exchange:        "GateIO",
				OrderID:         "74046514",
				AssetType:       asset.USDTMarginedFutures,
				Pair:            BTCUSDT,
				ClientOrderID:   "t-ws-usdt",
				Date:            time.UnixMilli(1681195484462),
				LastUpdated:     time.UnixMilli(1681195484462),
				RemainingAmount: 10,
				Amount:          10,
				Price:           200000,
				Type:            order.Limit,
				Side:            order.Short,
				Status:          order.Cancelled,
				TimeInForce:     order.ImmediateOrCancel,
			},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			connAsset := tc.a
			if connAsset == asset.Margin {
				connAsset = asset.Spot
			}
			ex := newWsAPITestExchange(t, connAsset, canManipulateRealOrders)
			tc.submit.Exchange, tc.submit.AssetType, tc.submit.Pair = ex.Name, tc.a, getPair(t, tc.a)
			got, err := ex.WebsocketSubmitOrder(t.Context(), tc.submit)
			require.NoError(t, err, "WebsocketSubmitOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, got, "WebsocketSubmitOrder should return the placed order")
				return
			}
			assert.NotEmpty(t, got.OrderID, "WebsocketSubmitOrder should return the order ID")
		})
	}
}

func TestUnmarshalJSONOrderbookLevels(t *testing.T) {
	t.Parallel()
	var ob OrderbookLevels
	require.NoError(t, ob.UnmarshalJSON([]byte(`[{"p":"123.45","s":"0.001"}]`)))
	assert.Equal(t, 123.45, ob[0].Price, "Price should be correct")
	assert.Equal(t, 0.001, ob[0].Amount, "Amount should be correct")
	require.Error(t, ob.UnmarshalJSON([]byte(`["p":"123.45","s":"0.001"]`)))
}

func TestGetLendingCurrencyList(t *testing.T) {
	t.Parallel()
	result, err := e.GetLendingCurrencyList(t.Context())
	require.NoError(t, err, "GetLendingCurrencyList must not error")
	if mockTests {
		require.Len(t, result, 5, "GetLendingCurrencyList must return every recorded currency")
		exp := &LendingCurrencyDetailResponse{Currency: currency.USDT, MinLendAmount: 1, MaxLendAmount: 100000000, MaxRate: 0.00057, MinRate: 0.00000011}
		assert.Equal(t, exp, result[0], "GetLendingCurrencyList should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetLendingCurrencyList should return currencies")
}

func TestGetLendingCurrencyDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetLendingCurrencyDetail(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetLendingCurrencyDetail must require a currency")

	result, err := e.GetLendingCurrencyDetail(t.Context(), currency.ETH)
	require.NoError(t, err, "GetLendingCurrencyDetail must not error")
	if mockTests {
		exp := &LendingCurrencyDetailResponse{Currency: currency.ETH, MinLendAmount: 0.001, MaxLendAmount: 500000000, MaxRate: 0.00057, MinRate: 0.00000011}
		assert.Equal(t, exp, result, "GetLendingCurrencyDetail should decode every field")
		return
	}
	require.NotNil(t, result, "GetLendingCurrencyDetail must return a result")
	assert.True(t, result.Currency.Equal(currency.ETH), "GetLendingCurrencyDetail should return the requested currency")
}

func TestCreateLendingOrRedemption(t *testing.T) {
	t.Parallel()
	err := e.CreateLendingOrRedemption(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateLendingOrRedemption must reject a nil request")

	arg := &LendingOrRedemptionRequest{}
	err = e.CreateLendingOrRedemption(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateLendingOrRedemption must require a currency")

	arg.Currency = currency.USDT
	err = e.CreateLendingOrRedemption(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "CreateLendingOrRedemption must require an amount")

	arg.Amount = 1
	err = e.CreateLendingOrRedemption(t.Context(), arg)
	require.ErrorIs(t, err, errLoanTypeIsRequired, "CreateLendingOrRedemption must require an operation type")

	arg.Type = "lend"
	err = e.CreateLendingOrRedemption(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "CreateLendingOrRedemption must require a minimum rate for lending")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.MinRate = 0.00000211
	err = e.CreateLendingOrRedemption(t.Context(), arg)
	require.NoError(t, err, "CreateLendingOrRedemption must not error when lending")

	err = e.CreateLendingOrRedemption(t.Context(), &LendingOrRedemptionRequest{Currency: currency.USDT, Amount: 1, Type: "redeem"})
	assert.NoError(t, err, "CreateLendingOrRedemption should not require a minimum rate when redeeming")
}

func TestGetUserLendingOrderList(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUserLendingOrderList(t.Context(), currency.USDT, 10, 100)
	require.NoError(t, err, "GetUserLendingOrderList must not error")
	if mockTests {
		exp := []*LendOrderDetailResponse{{
			Currency:       currency.USDT,
			CurrentAmount:  20.999992,
			Amount:         20.999992,
			MinRate:        0.1,
			InterestStatus: "interest_dividend",
			CreateTime:     types.Time(time.UnixMilli(1673247054000)),
			UpdateTime:     types.Time(time.UnixMilli(1673247054000)),
		}}
		assert.Equal(t, exp, result, "GetUserLendingOrderList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUserLendingOrderList should return a result")
}

func TestAmendUserLendingInformation(t *testing.T) {
	t.Parallel()
	err := e.AmendUserLendingInformation(t.Context(), currency.EMPTYCODE, 123.45)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "AmendUserLendingInformation must require a currency")

	err = e.AmendUserLendingInformation(t.Context(), currency.USDT, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "AmendUserLendingInformation must require a minimum rate")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.AmendUserLendingInformation(t.Context(), currency.USDT, 0.0001)
	assert.NoError(t, err, "AmendUserLendingInformation should not error")
}

func TestGetLendingTransactionRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetLendingTransactionRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLendingTransactionRecords must reject a nil request")

	startTime, endTime := getTime()
	_, err = e.GetLendingTransactionRecords(t.Context(), &LendingTransactionRecordsRequest{Currency: currency.USDT, Page: 1, Limit: 100, From: endTime, To: startTime, Type: "lend"})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetLendingTransactionRecords must reject a reversed time range")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLendingTransactionRecords(t.Context(), &LendingTransactionRecordsRequest{Currency: currency.USDT, Page: 10, Limit: 100, From: startTime, To: endTime, Type: "redeem"})
	require.NoError(t, err, "GetLendingTransactionRecords must not error")
	if mockTests {
		exp := []*LendingTransactionRecordResponse{{
			Type:             "redeem",
			Currency:         currency.USDT,
			Amount:           1,
			LastWalletAmount: 0.2,
			CreateTime:       types.Time(time.UnixMilli(1744150000000)),
		}}
		assert.Equal(t, exp, result, "GetLendingTransactionRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetLendingTransactionRecords should return a result")
}

func TestGetUserTotalInterestIncomePerCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserTotalInterestIncomePerCurrency(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetUserTotalInterestIncomePerCurrency must require a currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUserTotalInterestIncomePerCurrency(t.Context(), currency.USDT)
	require.NoError(t, err, "GetUserTotalInterestIncomePerCurrency must not error")
	if mockTests {
		assert.Equal(t, &CurrencyAndInterestIncomeResponse{Currency: currency.USDT, Interest: 123.345}, result, "GetUserTotalInterestIncomePerCurrency should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUserTotalInterestIncomePerCurrency should return a result")
}

func TestGetUserDividendRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserDividendRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUserDividendRecords must reject a nil request")

	startTime, endTime := getTime()
	_, err = e.GetUserDividendRecords(t.Context(), &UserDividendRecordsRequest{Currency: currency.USDT, Page: 1, Limit: 100, From: endTime, To: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUserDividendRecords must reject a reversed time range")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUserDividendRecords(t.Context(), &UserDividendRecordsRequest{Currency: currency.AE, From: startTime, To: endTime})
	require.NoError(t, err, "GetUserDividendRecords must not error")
	if mockTests {
		exp := []*UserDividendRecordsResponse{{
			Status:         1,
			Currency:       currency.AE,
			ActualRate:     0.0005,
			Interest:       0.05,
			InterestStatus: "interest_dividend",
			CreateTime:     types.Time(time.UnixMilli(1744150000000)),
		}}
		assert.Equal(t, exp, result, "GetUserDividendRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUserDividendRecords should return a result")
}

func TestGetCurrencyInterestCompoundingStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrencyInterestCompoundingStatus(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetCurrencyInterestCompoundingStatus must require a currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCurrencyInterestCompoundingStatus(t.Context(), currency.USDT)
	require.NoError(t, err, "GetCurrencyInterestCompoundingStatus must not error")
	if mockTests {
		assert.Equal(t, &CurrencyInterestStatusResponse{Currency: currency.USDT, InterestStatus: "interest_dividend"}, result, "GetCurrencyInterestCompoundingStatus should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCurrencyInterestCompoundingStatus should return a result")
}

func TestGetUniLoanCurrencyAnnualisedTrendChart(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetUniLoanCurrencyAnnualisedTrendChart(t.Context(), time.Time{}, endTime, currency.AE)
	require.ErrorIs(t, err, common.ErrDateUnset, "GetUniLoanCurrencyAnnualisedTrendChart must require a start time")

	_, err = e.GetUniLoanCurrencyAnnualisedTrendChart(t.Context(), endTime, startTime, currency.AE)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUniLoanCurrencyAnnualisedTrendChart must reject a reversed time range")

	_, err = e.GetUniLoanCurrencyAnnualisedTrendChart(t.Context(), endTime.Add(-31*24*time.Hour), endTime, currency.AE)
	require.ErrorIs(t, err, errUniChartRangeExceeded, "GetUniLoanCurrencyAnnualisedTrendChart must reject a range beyond 30 days")

	_, err = e.GetUniLoanCurrencyAnnualisedTrendChart(t.Context(), startTime, endTime, currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetUniLoanCurrencyAnnualisedTrendChart must require a currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUniLoanCurrencyAnnualisedTrendChart(t.Context(), startTime, endTime, currency.AE)
	require.NoError(t, err, "GetUniLoanCurrencyAnnualisedTrendChart must not error")
	if mockTests {
		assert.Equal(t, []*UniLoanAssetResponse{{Time: types.Time(time.Unix(1744156800, 0)), Value: 0.01}}, result, "GetUniLoanCurrencyAnnualisedTrendChart should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUniLoanCurrencyAnnualisedTrendChart should return a result")
}

func TestGetCurrencyEstimatedAnnualisedInterestRate(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCurrencyEstimatedAnnualisedInterestRate(t.Context())
	require.NoError(t, err, "GetCurrencyEstimatedAnnualisedInterestRate must not error")
	if mockTests {
		assert.Equal(t, []*CurrencyEstimatedAnnualInterestRateResponse{{Currency: currency.USDT, EstimatedRate: 0.0226}}, result, "GetCurrencyEstimatedAnnualisedInterestRate should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCurrencyEstimatedAnnualisedInterestRate should return a result")
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	_, err := e.GetAvailableTransferChains(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAvailableTransferChains must reject an empty currency")

	for _, tc := range []struct {
		ccy currency.Code
		exp []string
	}{
		// Chains Gate has disabled are left out
		{ccy: currency.USDT, exp: []string{"ETH", "APT", "ARBEVM"}},
		{ccy: currency.ETH, exp: []string{"ETH", "ABS", "ARBEVM", "ARBNOVA", "BASEEVM"}},
	} {
		result, err := e.GetAvailableTransferChains(t.Context(), tc.ccy)
		require.NoErrorf(t, err, "GetAvailableTransferChains must not error for %s", tc.ccy)
		if mockTests {
			assert.Equalf(t, tc.exp, result, "GetAvailableTransferChains should return every enabled %s chain", tc.ccy)
			continue
		}
		assert.NotEmptyf(t, result, "GetAvailableTransferChains should return %s chains", tc.ccy)
	}
}

func TestResponseError(t *testing.T) {
	t.Parallel()

	assert.NoError(t, responseError(nil), "nil result should not error")
	assert.NoError(t, responseError(&struct{ Field string }{}), "result without AsError should not error")

	clean := &BotCreateResponse{Code: 200}
	assert.NoError(t, responseError(clean), "successful response should not error")

	failed := &BotCreateResponse{Code: 4, Message: "boom", TraceID: "trace-1"}
	err := responseError(failed)
	require.Error(t, err, "failed response must error")
	assert.Contains(t, err.Error(), "trace-1", "error should carry the trace ID")

	// Results are commonly unmarshalled into a pointer to a pointer, whose method set is empty
	doubled := &AlphaCurrencyQuoteDetailResponse{Label: "BALANCE_NOT_ENOUGH", Message: "insufficient"}
	err = responseError(&doubled)
	require.Error(t, err, "double pointer result must still surface the embedded error")
	assert.Contains(t, err.Error(), "BALANCE_NOT_ENOUGH", "error should carry the alpha label")

	var null *BotCreateResponse
	require.NoError(t, json.Unmarshal([]byte("null"), &null), "Unmarshal must not error")
	assert.NoError(t, responseError(&null), "a null response should not error")
}
