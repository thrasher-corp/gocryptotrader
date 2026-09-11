package yobit

import (
	"log"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

var e *Exchange

// Please supply your own keys for better unit testing
const canManipulateRealOrders = false

var apiCredentials = &accounts.Credentials{
	Key:    "",
	Secret: "",
}

var testPair = currency.NewBTCUSD().Format(currency.PairFormat{Delimiter: "_"})

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("Yobit Setup error: %s", err)
	}

	if apiCredentials.Key != "" && apiCredentials.Secret != "" {
		e.API.AuthenticatedSupport = true
		e.SetCredentials(apiCredentials)
	}

	os.Exit(m.Run())
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := e.FetchTradablePairs(t.Context(), asset.Spot)
	if err != nil {
		t.Errorf("FetchTradablePairs err: %s", err)
	}
}

func TestGetInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetInfo(t.Context())
	if err != nil {
		t.Error("GetInfo() error")
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetTicker(t.Context(), testPair.String())
	assert.NoError(t, err, "GetTicker should not error")
}

func TestGetTickerResponseHandling(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		body      string
		want      map[string]Ticker
		errIs     error
		errString string
	}{
		{name: "defensive metadata alongside entries", body: `{"success":1,"error":"Pair is off: bcc_btc","btc_usd":{"last":80001}}`, want: map[string]Ticker{"btc_usd": {Last: 80001}}},
		{name: "entries without metadata", body: `{"btc_usd":{"last":80001}}`, want: map[string]Ticker{"btc_usd": {Last: 80001}}},
		{name: "malformed ticker", body: `{"btc_usd":1}`, errString: "error decoding ticker for btc_usd"},
		{name: "failed request", body: `{"success":0,"error":"Invalid pair name: btc_usd"}`, errIs: errTickerRequestFailed, errString: "Invalid pair name: btc_usd"},
		{name: "malformed error", body: `{"success":0,"error":{}}`, errString: "error decoding ticker error field"},
		{name: "malformed success", body: `{"success":"invalid"}`, errString: "error decoding ticker success field"},
		{name: "failure without error", body: `{"success":0}`, errIs: errTickerRequestFailed},
		{name: "failure with empty error", body: `{"success":0,"error":""}`, errIs: errTickerRequestFailed},
		{name: "failure with null ticker", body: `{"success":0,"error":"Pair is off: btc_usd","btc_usd":null}`, errIs: errTickerRequestFailed},
		{name: "failure with empty ticker", body: `{"success":0,"error":"Pair is off: btc_usd","btc_usd":{}}`, errIs: errTickerRequestFailed},
		{name: "failure with populated ticker", body: `{"success":0,"btc_usd":{"last":80001}}`, errIs: errTickerRequestFailed},
		{name: "error without success", body: `{"error":"Pair is off: btc_usd"}`, errIs: errTickerRequestFailed},
		{name: "failure metadata alongside a populated ticker", body: `{"success":0,"error":"Pair is off: eth_btc","btc_usd":{"last":80001}}`, errIs: errTickerRequestFailed, errString: "Pair is off: eth_btc"},
		{name: "empty response", body: `{}`, want: map[string]Ticker{}},
		{name: "error with null ticker", body: `{"error":"Pair is off: btc_usd","btc_usd":null}`, errIs: errTickerRequestFailed, errString: "empty ticker for btc_usd"},
		{name: "error with empty ticker", body: `{"error":"Pair is off: btc_usd","btc_usd":{}}`, errIs: errTickerRequestFailed},
		{name: "success with null ticker", body: `{"success":1,"btc_usd":null}`, errIs: errTickerRequestFailed},
		{name: "success with empty ticker", body: `{"success":1,"btc_usd":{}}`, errIs: errTickerRequestFailed},
		{name: "null ticker without metadata", body: `{"btc_usd":null}`, errIs: errTickerRequestFailed},
		{name: "empty ticker without metadata", body: `{"btc_usd":{}}`, errIs: errTickerRequestFailed},
		{name: "zero last with populated ticker", body: `{"btc_usd":{"last":0,"buy":80001}}`, want: map[string]Ticker{"btc_usd": {Buy: 80001}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				assert.Equal(t, "1", req.URL.Query().Get("ignore_invalid"), "ticker requests should ignore invalid pairs")
				_, err := w.Write([]byte(test.body))
				assert.NoError(t, err, "writing ticker response should not error")
			}))
			t.Cleanup(server.Close)
			require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")
			result, err := ex.GetTicker(t.Context(), "btc_usd")
			if test.errIs != nil || test.errString != "" {
				if test.errIs != nil {
					assert.ErrorIs(t, err, test.errIs, "GetTicker should return the expected sentinel")
				}
				if test.errString != "" {
					assert.ErrorContains(t, err, test.errString, "GetTicker should preserve error context")
				}
				assert.Nil(t, result, "GetTicker should not return tickers on failure")
				return
			}
			require.NoError(t, err, "GetTicker must decode successful responses")
			assert.Equal(t, test.want, result, "GetTicker should return only ticker entries")
		})
	}
}

func TestUpdateTickersResponseHandling(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name         string
		body         string
		errIs        error
		btcLast      float64
		btcRefreshed bool
	}{
		{name: "partial batch", body: `{"btc_usd":{"last":80002}}`, errIs: ticker.ErrTickerNotFound, btcLast: 80002, btcRefreshed: true},
		{name: "all invalid", body: `{"success":0,"error":"Empty pair list"}`, errIs: errTickerRequestFailed, btcLast: 80001},
		{name: "failure without error", body: `{"success":0}`, errIs: errTickerRequestFailed, btcLast: 80001},
		{name: "failure with empty error", body: `{"success":0,"error":""}`, errIs: errTickerRequestFailed, btcLast: 80001},
		{name: "failure with null ticker", body: `{"success":0,"error":"Pair is off: btc_usd","btc_usd":null}`, errIs: errTickerRequestFailed, btcLast: 80001},
		{name: "failure with empty ticker", body: `{"success":0,"error":"Pair is off: btc_usd","btc_usd":{}}`, errIs: errTickerRequestFailed, btcLast: 80001},
		{name: "error with null ticker", body: `{"error":"Pair is off: btc_usd","btc_usd":null}`, errIs: errTickerRequestFailed, btcLast: 80001},
		{name: "error with empty ticker", body: `{"error":"Pair is off: btc_usd","btc_usd":{}}`, errIs: errTickerRequestFailed, btcLast: 80001},
		{name: "success with null ticker", body: `{"success":1,"btc_usd":null}`, errIs: errTickerRequestFailed, btcLast: 80001},
		{name: "success with empty ticker", body: `{"success":1,"btc_usd":{}}`, errIs: errTickerRequestFailed, btcLast: 80001},
		{name: "null ticker without metadata", body: `{"btc_usd":null}`, errIs: errTickerRequestFailed, btcLast: 80001},
		{name: "failure metadata alongside a populated ticker", body: `{"success":0,"error":"Pair is off: eth_btc","btc_usd":{"last":80002}}`, errIs: errTickerRequestFailed, btcLast: 80001},
		{name: "healthy ticker beside null entry", body: `{"btc_usd":{"last":80002},"eth_btc":null}`, errIs: errTickerRequestFailed, btcLast: 80001},
		{name: "healthy ticker beside empty entry", body: `{"btc_usd":{"last":80002},"eth_btc":{}}`, errIs: errTickerRequestFailed, btcLast: 80001},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			ex.Name = t.Name()
			bitcoin := currency.NewBTCUSD()
			ethereum := currency.NewPair(currency.ETH, currency.BTC)
			require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{bitcoin, ethereum}, true), "test pairs must be enabled")
			var response atomic.Pointer[string]
			initialResponse := `{"btc_usd":{"last":80001},"eth_btc":{"last":0.031}}`
			response.Store(&initialResponse)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				assert.Equal(t, "/3/ticker/btc_usd-eth_btc", req.URL.Path, "ticker request should include both enabled pairs")
				assert.Equal(t, "1", req.URL.Query().Get("ignore_invalid"), "ticker request should allow partial results")
				_, err := w.Write([]byte(*response.Load()))
				assert.NoError(t, err, "writing ticker response should not error")
			}))
			t.Cleanup(server.Close)
			require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")
			require.NoError(t, ex.UpdateTickers(t.Context(), asset.Spot), "initial poll must populate both tickers")
			beforeBitcoin, err := ticker.GetTicker(ex.Name, bitcoin, asset.Spot)
			require.NoError(t, err, "initial bitcoin ticker must be cached")
			before, err := ticker.GetTicker(ex.Name, ethereum, asset.Spot)
			require.NoError(t, err, "initial ethereum ticker must be cached")
			response.Store(&test.body)
			err = ex.UpdateTickers(t.Context(), asset.Spot)
			assert.ErrorIs(t, err, test.errIs, "second poll should report omitted pairs or request failure")
			if test.errIs == ticker.ErrTickerNotFound {
				assert.ErrorContains(t, err, "ETH_BTC", "partial result error should identify the omitted pair")
			}
			cached, err := ticker.GetTicker(ex.Name, bitcoin, asset.Spot)
			require.NoError(t, err, "bitcoin ticker must remain cached")
			assert.Equal(t, test.btcLast, cached.Last, "valid partial results should update while failures should not overwrite prices")
			if !test.btcRefreshed {
				assert.Equal(t, beforeBitcoin.LastUpdated, cached.LastUpdated, "failed ticker should not be marked fresh")
			}
			stale, err := ticker.GetTicker(ex.Name, ethereum, asset.Spot)
			require.NoError(t, err, "previous ethereum ticker must remain cached")
			assert.Equal(t, before.LastUpdated, stale.LastUpdated, "omitted ticker should not be marked fresh")
			assert.Equal(t, 0.031, stale.Last, "omitted ticker should retain its previous cached value")
			result, err := ex.UpdateTicker(t.Context(), ethereum, asset.Spot)
			assert.ErrorIs(t, err, test.errIs, "UpdateTicker should propagate the refresh failure")
			assert.Nil(t, result, "UpdateTicker should not return the stale cached ticker as a successful refresh")
			result, err = ex.UpdateTicker(t.Context(), bitcoin, asset.Spot)
			if test.btcRefreshed {
				require.NoError(t, err, "UpdateTicker must return the refreshed pair despite unrelated omissions")
				require.NotNil(t, result, "UpdateTicker must return the refreshed ticker")
				assert.Equal(t, test.btcLast, result.Last, "UpdateTicker should return the fresh price")
			} else {
				assert.ErrorIs(t, err, test.errIs, "UpdateTicker should propagate failure for a pair that did not refresh")
				assert.Nil(t, result, "UpdateTicker should not return a cached price after failure")
			}
		})
	}
}

func TestGetDepth(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepth(t.Context(), testPair.String())
	assert.NoError(t, err, "GetDepth should not error")
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetTrades(t.Context(), testPair.String())
	assert.NoError(t, err, "GetTrades should not error")
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err)
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenOrders(t.Context(), "")
	if err == nil {
		t.Error("GetOpenOrders() Expected error")
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetOrderInfo(t.Context(), "1337", currency.NewBTCUSD(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCryptoDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.GetCryptoDepositAddress(t.Context(), "bTc", false)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	err := e.CancelExistingOrder(t.Context(), 1337)
	if err == nil {
		t.Error("CancelOrder() Expected error")
	}
}

func TestTrade(t *testing.T) {
	t.Parallel()
	_, err := e.Trade(t.Context(), "", order.Buy.String(), 0, 0)
	if err == nil {
		t.Error("Trade() Expected error")
	}
}

func TestWithdrawCoinsToAddress(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawCoinsToAddress(t.Context(), "", 0, "")
	if err == nil {
		t.Error("WithdrawCoinsToAddress() Expected error")
	}
}

func TestCreateYobicode(t *testing.T) {
	t.Parallel()
	_, err := e.CreateCoupon(t.Context(), "bla", 0)
	if err == nil {
		t.Error("CreateYobicode() Expected error")
	}
}

func TestRedeemYobicode(t *testing.T) {
	t.Parallel()
	_, err := e.RedeemCoupon(t.Context(), "bla2")
	if err == nil {
		t.Error("RedeemYobicode() Expected error")
	}
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:  1,
		FeeType: exchange.CryptocurrencyTradeFee,
		Pair: currency.NewPairWithDelimiter(currency.LTC.String(),
			currency.BTC.String(),
			"-"),
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	feeBuilder := setFeeBuilder()
	_, err := e.GetFeeByType(t.Context(), feeBuilder)
	if err != nil {
		t.Fatal(err)
	}
	if !sharedtestvalues.AreAPICredentialsSet(e) {
		if feeBuilder.FeeType != exchange.OfflineTradeFee {
			t.Errorf("Expected %v, received %v", exchange.OfflineTradeFee, feeBuilder.FeeType)
		}
	} else {
		if feeBuilder.FeeType != exchange.CryptocurrencyTradeFee {
			t.Errorf("Expected %v, received %v", exchange.CryptocurrencyTradeFee, feeBuilder.FeeType)
		}
	}
}

func TestGetFee(t *testing.T) {
	feeBuilder := setFeeBuilder()

	// CryptocurrencyTradeFee Basic
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// InternationalBankWithdrawalFee QIWI
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	feeBuilder.BankTransactionType = exchange.Qiwi
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// InternationalBankWithdrawalFee Wire
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	feeBuilder.BankTransactionType = exchange.WireTransfer
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// InternationalBankWithdrawalFee Payeer
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	feeBuilder.BankTransactionType = exchange.Payeer
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// InternationalBankWithdrawalFee Capitalist
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.RUR
	feeBuilder.BankTransactionType = exchange.Capitalist
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// InternationalBankWithdrawalFee AdvCash
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	feeBuilder.BankTransactionType = exchange.AdvCash
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// InternationalBankWithdrawalFee PerfectMoney
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.RUR
	feeBuilder.BankTransactionType = exchange.PerfectMoney
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expectedResult := exchange.AutoWithdrawCryptoWithAPIPermissionText + " & " + exchange.WithdrawFiatViaWebsiteOnlyText
	withdrawPermissions := e.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		Pairs:     []currency.Pair{currency.NewPair(currency.LTC, currency.BTC)},
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}

	_, err := e.GetActiveOrders(t.Context(), &getOrdersRequest)
	if sharedtestvalues.AreAPICredentialsSet(e) && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !sharedtestvalues.AreAPICredentialsSet(e) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Pairs:     []currency.Pair{currency.NewPair(currency.LTC, currency.BTC)},
		StartTime: time.Unix(0, 0),
		EndTime:   time.Unix(math.MaxInt64, 0),
		Side:      order.AnySide,
	}

	_, err := e.GetOrderHistory(t.Context(), &getOrdersRequest)
	if sharedtestvalues.AreAPICredentialsSet(e) && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !sharedtestvalues.AreAPICredentialsSet(e) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// TestSubmitOrder and below can impact your orders on the exchange. Enable canManipulateRealOrders to run them
func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	orderSubmission := &order.Submit{
		Exchange: e.Name,
		Pair: currency.Pair{
			Delimiter: "_",
			Base:      currency.BTC,
			Quote:     currency.USD,
		},
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		ClientID:  "meowOrder",
		AssetType: asset.Spot,
	}
	response, err := e.SubmitOrder(t.Context(), orderSubmission)
	if sharedtestvalues.AreAPICredentialsSet(e) && (err != nil || response.Status != order.New) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !sharedtestvalues.AreAPICredentialsSet(e) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      currencyPair,
		AssetType: asset.Spot,
	}

	err := e.CancelOrder(t.Context(), orderCancellation)
	if !sharedtestvalues.AreAPICredentialsSet(e) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(e) && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      currencyPair,
		AssetType: asset.Spot,
	}

	resp, err := e.CancelAllOrders(t.Context(), orderCancellation)

	if !sharedtestvalues.AreAPICredentialsSet(e) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(e) && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}

	if len(resp.Status) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.Status))
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	_, err := e.ModifyOrder(t.Context(),
		&order.Modify{AssetType: asset.Spot})
	if err == nil {
		t.Error("ModifyOrder() Expected error")
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	withdrawCryptoRequest := withdraw.Request{
		Exchange:    e.Name,
		Amount:      -1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}

	_, err := e.WithdrawCryptocurrencyFunds(t.Context(),
		&withdrawCryptoRequest)
	if !sharedtestvalues.AreAPICredentialsSet(e) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(e) && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	withdrawFiatRequest := withdraw.Request{}
	_, err := e.WithdrawFiatFunds(t.Context(), &withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'",
			common.ErrFunctionNotSupported,
			err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	withdrawFiatRequest := withdraw.Request{}
	_, err := e.WithdrawFiatFundsToInternationalBank(t.Context(),
		&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'",
			common.ErrFunctionNotSupported,
			err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	if sharedtestvalues.AreAPICredentialsSet(e) {
		_, err := e.GetDepositAddress(t.Context(), currency.BTC, "", "")
		if err != nil {
			t.Error(err)
		}
	} else {
		_, err := e.GetDepositAddress(t.Context(), currency.BTC, "", "")
		if err == nil {
			t.Error("GetDepositAddress() error")
		}
	}
}

func TestGetRecentTrades(t *testing.T) {
	_, err := e.GetRecentTrades(t.Context(), testPair, asset.Spot)
	assert.NoError(t, err, "GetRecentTrades should not error")
}

func TestGetHistoricTrades(t *testing.T) {
	_, err := e.GetHistoricTrades(t.Context(), testPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTicker(t.Context(), testPair, asset.Spot)
	assert.NoError(t, err, "UpdateTicker should not error")
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := e.UpdateTickers(t.Context(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestWrapperGetServerTime(t *testing.T) {
	t.Parallel()
	st, err := e.GetServerTime(t.Context(), asset.Spot)
	require.NoError(t, err)

	if st.IsZero() {
		t.Fatal("expected a time")
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.CurrencyPairs.GetPairs(a, false)
		require.NoErrorf(t, err, "cannot get pairs for %s", a)
		require.NotEmptyf(t, pairs, "no pairs for %s", a)
		resp, err := e.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}
