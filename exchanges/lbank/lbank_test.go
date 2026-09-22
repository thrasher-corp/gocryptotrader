package lbank

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/md5" //nolint:gosec // Used for this exchange
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

// Please supply your own keys here for due diligence testing
const canManipulateRealOrders = false

var apiCredentials = &accounts.Credentials{
	Key:    "",
	Secret: "",
}

var (
	e        *Exchange
	testPair = currency.NewBTCUSDT().Format(currency.PairFormat{Delimiter: "_"})
)

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("Lbank Setup error: %s", err)
	}
	if apiCredentials.Key != "" && apiCredentials.Secret != "" {
		e.API.AuthenticatedSupport = true
		e.SetCredentials(apiCredentials)
	}
	os.Exit(m.Run())
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetTicker(t.Context(), testPair.String())
	assert.NoError(t, err, "GetTicker should not error")
}

func TestGetTimestamp(t *testing.T) {
	t.Parallel()
	ts, err := e.GetTimestamp(t.Context())
	require.NoError(t, err, "GetTimestamp must not error")
	assert.NotZero(t, ts, "GetTimestamp should return a non-zero time")
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	tickers, err := e.GetTickers(t.Context())
	require.NoError(t, err, "GetTickers must not error")
	assert.Greater(t, len(tickers), 1, "GetTickers should return more than 1 ticker")
}

func TestGetCurrencyPairs(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrencyPairs(t.Context())
	assert.NoError(t, err, "GetCurrencyPairs should not error")
}

func TestGetMarketDepths(t *testing.T) {
	t.Parallel()
	d, err := e.GetMarketDepths(t.Context(), testPair.String(), 4)
	require.NoError(t, err, "GetMarketDepths must not error")
	require.NotEmpty(t, d, "GetMarketDepths must return a non-empty response")
	assert.Len(t, d.Asks, 4, "GetMarketDepths should return 4 asks")
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	r, err := e.GetTrades(t.Context(), testPair.String(), 420, time.Now())
	require.NoError(t, err, "GetTrades must not error")
	require.NotEmpty(t, r, "GetTrades must return a non-empty response")
	assert.Len(t, r, 420, "GetTrades should return 420 trades")
}

func TestGetKlines(t *testing.T) {
	t.Parallel()
	_, err := e.GetKlines(t.Context(), testPair.String(), "600", "minute1", time.Now())
	assert.NoError(t, err, "GetKlines should not error")
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateOrderbook(t.Context(), currency.EMPTYPAIR, asset.Spot)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.UpdateOrderbook(t.Context(), testPair, asset.Options)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	_, err = e.UpdateOrderbook(t.Context(), testPair, asset.Spot)
	assert.NoError(t, err, "UpdateOrderbook should not error")
}

func TestGetUserInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.GetUserInfo(t.Context())
	require.NoError(t, err, "GetUserInfo must not error")
}

func TestCreateOrder(t *testing.T) {
	t.Parallel()

	_, err := e.CreateOrder(t.Context(), testPair.String(), "what", 1231, 12314)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = e.CreateOrder(t.Context(), testPair.String(), order.Buy.String(), 0, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.CreateOrder(t.Context(), testPair.String(), order.Sell.String(), 1231, 0)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	_, err = e.CreateOrder(t.Context(), testPair.String(), order.Buy.String(), 58, 681)
	assert.NoError(t, err, "CreateOrder should not error")
}

func TestRemoveOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	_, err := e.RemoveOrder(t.Context(), testPair.String(), "24f7ce27-af1d-4dca-a8c1-ef1cbeec1b23")
	assert.NoError(t, err, "RemoveOrder should not error")
}

func TestQueryOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.QueryOrder(t.Context(), testPair.String(), "1")
	assert.NoError(t, err, "QueryOrder should not error")
}

func TestQueryOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.QueryOrderHistory(t.Context(), testPair.String(), "1", "100")
	assert.NoError(t, err, "QueryOrderHistory should not error")
}

func TestGetPairInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetPairInfo(t.Context())
	assert.NoError(t, err, "GetPairInfo should not error")
}

func TestOrderTransactionDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.OrderTransactionDetails(t.Context(), testPair.String(), "24f7ce27-af1d-4dca-a8c1-ef1cbeec1b23")
	assert.NoError(t, err, "OrderTransactionDetails should not error")
}

func TestTransactionHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.TransactionHistory(t.Context(), testPair.String(), "", "", "", "", "", "")
	assert.NoError(t, err, "TransactionHistory should not error")
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.GetOpenOrders(t.Context(), testPair.String(), "1", "50")
	assert.NoError(t, err, "GetOpenOrders should not error")
}

func TestUSD2RMBRate(t *testing.T) {
	t.Parallel()
	_, err := e.USD2RMBRate(t.Context())
	assert.NoError(t, err, "USD2RMBRate should not error")
}

func TestGetWithdrawConfig(t *testing.T) {
	t.Parallel()
	c, err := e.GetWithdrawConfig(t.Context(), currency.ETH)
	require.NoError(t, err, "GetWithdrawConfig must not error")
	assert.NotEmpty(t, c)
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	_, err := e.Withdraw(t.Context(), "", "", "", "", "", "")
	require.NoError(t, err, "Withdraw must not error")
}

func TestGetWithdrawRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.GetWithdrawalRecords(t.Context(), currency.ETH.String(), 0, 1, 20)
	assert.NoError(t, err, "GetWithdrawRecords should not error")
}

func TestLoadPrivKey(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	e.SetDefaults()
	require.ErrorIs(t, e.loadPrivKey(t.Context()), exchange.ErrCredentialsAreEmpty)

	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "test", Secret: "errortest"})
	assert.ErrorIs(t, e.loadPrivKey(ctx), errPEMBlockIsNil)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der := x509.MarshalPKCS1PrivateKey(key)
	ctx = accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "test", Secret: base64.StdEncoding.EncodeToString(der)})
	require.ErrorIs(t, e.loadPrivKey(ctx), errUnableToParsePrivateKey)

	ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err = x509.MarshalPKCS8PrivateKey(ecdsaKey)
	require.NoError(t, err)
	ctx = accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "test", Secret: base64.StdEncoding.EncodeToString(der)})
	require.ErrorIs(t, e.loadPrivKey(ctx), common.ErrTypeAssertFailure)

	key, err = rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err = x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	ctx = accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "test", Secret: base64.StdEncoding.EncodeToString(der)})
	assert.NoError(t, e.loadPrivKey(ctx), "loadPrivKey should not error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	assert.NoError(t, e.loadPrivKey(t.Context()), "loadPrivKey should not error")
}

func TestSign(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	e.SetDefaults()
	_, err := e.sign("hello123")
	require.ErrorIs(t, err, errPrivateKeyNotLoaded)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "GenerateKey must not error")
	e.privateKey = key

	targetMessage := "hello123"
	msg, err := e.sign(targetMessage)
	require.NoError(t, err, "sign must not error")

	md5sum := md5.Sum([]byte(targetMessage)) //nolint:gosec // Used for this exchange
	shasum := sha256.Sum256([]byte(strings.ToUpper(hex.EncodeToString(md5sum[:]))))
	sigBytes, err := base64.StdEncoding.DecodeString(msg)
	require.NoError(t, err)
	err = rsa.VerifyPKCS1v15(&e.privateKey.PublicKey, crypto.SHA256, shasum[:], sigBytes)
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	require.NoError(t, e.loadPrivKey(t.Context()), "loadPrivKey must not error")

	_, err = e.sign("hello123")
	assert.NoError(t, err, "sign should not error")
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	r, err := e.SubmitOrder(t.Context(), &order.Submit{
		Exchange:  e.Name,
		Pair:      testPair,
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		ClientID:  "meowOrder",
		AssetType: asset.Spot,
	})
	if sharedtestvalues.AreAPICredentialsSet(e) {
		require.NoError(t, err, "SubmitOrder must not error")
		assert.Equal(t, order.New, r.Status, "SubmitOrder should return order status New")
	} else {
		assert.Error(t, err, "SubmitOrder should error when credentials are not set")
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	err := e.CancelOrder(t.Context(), &order.Cancel{
		Pair:      testPair,
		AssetType: asset.Spot,
		OrderID:   "24f7ce27-af1d-4dca-a8c1-ef1cbeec1b23",
	})
	assert.NoError(t, err, "CancelOrder should not error")
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.GetOrderInfo(t.Context(), "9ead39f5-701a-400b-b635-d7349eb0f6b", currency.EMPTYPAIR, asset.Spot)
	assert.NoError(t, err, "GetOrderInfo should not error")
}

func TestGetAllOpenOrderID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.getAllOpenOrderID(t.Context())
	assert.NoError(t, err, "getAllOpenOrderID should not error")
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	_, err := e.GetFeeByType(t.Context(), &exchange.FeeBuilder{
		Amount:  2,
		FeeType: exchange.CryptocurrencyWithdrawalFee,
		Pair:    testPair,
	})
	assert.NoError(t, err, "GetFeeByType should not error")
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.UpdateAccountBalances(t.Context(), asset.Spot)
	assert.NoError(t, err, "UpdateAccountBalances should not error")
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		Side:      order.AnySide,
		AssetType: asset.Spot,
		Type:      order.AnyType,
	})
	assert.NoError(t, err, "GetActiveOrders should not error")
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		Side:      order.AnySide,
		AssetType: asset.Spot,
		Type:      order.AnyType,
	})
	assert.NoError(t, err, "GetOrderHistory should not error")
}

// TestGetActiveOrdersEmptyResponse ensures an order the exchange no longer
// reports is skipped instead of panicking on an empty order slice.
func TestGetActiveOrdersEmptyResponse(t *testing.T) {
	t.Parallel()
	ex := setupOrderGuard(t, orderGuardHandler(t, orderGuardNoOrders))

	got, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		Pairs:     currency.Pairs{testPair},
		Side:      order.AnySide,
		AssetType: asset.Spot,
		Type:      order.AnyType,
	})
	require.NoError(t, err, "GetActiveOrders must not error")
	assert.Empty(t, got, "GetActiveOrders should skip orders the exchange no longer reports")
}

// TestGetActiveOrdersCompoundOrderType ensures a compound order type LBank
// documents, such as buy_maker, is mapped to its side instead of aborting the
// rest of the open order listing.
func TestGetActiveOrdersCompoundOrderType(t *testing.T) {
	t.Parallel()
	ex := setupOrderGuard(t, orderGuardHandler(t, orderGuardSingleOrder("buy_maker")))

	got, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		Pairs:     currency.Pairs{testPair},
		Side:      order.AnySide,
		AssetType: asset.Spot,
		Type:      order.AnyType,
	})
	require.NoError(t, err, "GetActiveOrders must not error for a compound order type")
	require.Len(t, got, 1, "GetActiveOrders must keep the resting order")
	assert.Equal(t, order.Buy, got[0].Side, "GetActiveOrders should map buy_maker to the buy side")
}

// TestGetActiveOrdersUnknownSide ensures an order type LBank does not document
// surfaces as an error instead of being reported as a sell.
func TestGetActiveOrdersUnknownSide(t *testing.T) {
	t.Parallel()
	ex := setupOrderGuard(t, orderGuardHandler(t, orderGuardSingleOrder("hold")))

	_, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		Pairs:     currency.Pairs{testPair},
		Side:      order.AnySide,
		AssetType: asset.Spot,
		Type:      order.AnyType,
	})
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "GetActiveOrders should reject an order type it cannot map")
}

// TestGetOrderInfoEmptyResponse ensures an order the exchange no longer reports
// is reported as missing instead of panicking on an empty order slice.
func TestGetOrderInfoEmptyResponse(t *testing.T) {
	t.Parallel()
	ex := setupOrderGuard(t, orderGuardHandler(t, orderGuardNoOrders))

	_, err := ex.GetOrderInfo(t.Context(), "1", testPair, asset.Spot)
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "GetOrderInfo should report an order the exchange no longer holds")
}

// TestGetOrderInfoUnknownSide ensures an order type LBank does not document
// surfaces as an error instead of being reported as a sell.
func TestGetOrderInfoUnknownSide(t *testing.T) {
	t.Parallel()
	ex := setupOrderGuard(t, orderGuardHandler(t, orderGuardSingleOrder("hold")))

	_, err := ex.GetOrderInfo(t.Context(), "1", testPair, asset.Spot)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "GetOrderInfo should reject an order type it cannot map")
}

// TestGetOrderInfoCompoundOrderType ensures a compound order type LBank
// documents, such as sell_market, is mapped to its side instead of erroring.
func TestGetOrderInfoCompoundOrderType(t *testing.T) {
	t.Parallel()
	ex := setupOrderGuard(t, orderGuardHandler(t, orderGuardSingleOrder("sell_market")))

	got, err := ex.GetOrderInfo(t.Context(), "1", testPair, asset.Spot)
	require.NoError(t, err, "GetOrderInfo must not error for a compound order type")
	assert.Equal(t, order.Sell, got.Side, "GetOrderInfo should map sell_market to the sell side")
}

// TestGetOrderInfoOrderIDNotFound ensures an order ID the exchange does not
// report is reported as missing instead of as a zero-valued order.
func TestGetOrderInfoOrderIDNotFound(t *testing.T) {
	t.Parallel()
	ex := setupOrderGuard(t, orderGuardHandler(t, orderGuardSingleOrder("buy")))

	_, err := ex.GetOrderInfo(t.Context(), "2", testPair, asset.Spot)
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "GetOrderInfo should report an order ID the exchange does not return")
}

// orderGuardNoOrders is an empty LBank order query response.
const orderGuardNoOrders = `{"result":"true","error_code":0,"orders":[]}`

// orderGuardOpenOrder returns an open order listing carrying a single order.
func orderGuardOpenOrder(orderID string) string {
	return fmt.Sprintf(`{"result":"true","error_code":0,"orders":[{"order_id":%q,"symbol":"btc_usdt","type":"buy","price":10,"amount":2,"deal_amount":1,"avg_price":10,"status":0,"created_time":1758499200000}]}`, orderID)
}

// orderGuardSingleOrder returns an order query response carrying one order of
// the supplied type.
func orderGuardSingleOrder(orderType string) string {
	return fmt.Sprintf(`{"result":"true","error_code":0,"orders":[{"order_id":"1","symbol":"btc_usdt","type":%q,"price":10,"amount":2,"deal_amount":1,"avg_price":10,"status":0,"created_time":1758499200000}]}`, orderType)
}

// orderGuardHandler answers the open order listing with a single order on the
// first page and nothing after it, and every order query with queryResponse.
func orderGuardHandler(t *testing.T, queryResponse string) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, r.ParseForm(), "the request body should parse as a form")
		var body string
		switch {
		case strings.HasSuffix(r.URL.Path, lbankOpeningOrders):
			if r.Form.Get("current_page") == "1" {
				body = orderGuardOpenOrder("1")
			} else {
				body = orderGuardNoOrders
			}
		case strings.HasSuffix(r.URL.Path, lbankQueryOrder):
			body = queryResponse
		default:
			assert.Failf(t, "unexpected endpoint", "the wrapper should only request open orders and order queries, got %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, err := fmt.Fprint(w, body)
		assert.NoError(t, err, "writing the response should not error")
	}
}

// setupOrderGuard returns an exchange served by handler, with a single enabled
// spot pair so the open order crawl only asks about the pair under test.
func setupOrderGuard(t *testing.T, handler http.Handler) *Exchange {
	t.Helper()
	server := httptest.NewTestServer(t, handler)
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.API.AuthenticatedSupport = true
	ex.SkipAuthCheck = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-key", Secret: "mock-secret"})
	var err error
	ex.privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "the RSA test key must generate")
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{testPair}, false), "StorePairs must not error for available pairs")
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{testPair}, true), "StorePairs must not error for enabled pairs")
	require.NoError(t, ex.SetHTTPClient(orderGuardClient(server)), "SetHTTPClient must not error")
	return ex
}

// orderGuardClient returns the test server's client with the relative URL
// adapter installed.
func orderGuardClient(server *httptest.Server) *http.Client {
	client := server.Client()
	client.Transport = &orderGuardTransport{base: client.Transport, serverURL: server.URL}
	return client
}

// orderGuardTransport resolves the relative request URLs SendAuthHTTPRequest
// builds against the test server, so these tests exercise a real http.Client
// against a real server rather than a transport answering on its behalf.
//
// SendAuthHTTPRequest assigns the caller's path straight to request.Item.Path
// without prefixing the configured endpoint URL, so a relative request never
// leaves the client at all:
//
//	Post "/v2/orders_info.do": unsupported protocol scheme ""
//
// That pre-existing bug is being fixed separately in #2272. Requests that
// already carry a host are passed through untouched, so this adapter turns into
// a no-op once #2272 lands and can be deleted with it.
type orderGuardTransport struct {
	base      http.RoundTripper
	serverURL string
}

func (r *orderGuardTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host != "" {
		return r.base.RoundTrip(req)
	}
	server, err := url.Parse(r.serverURL)
	if err != nil {
		return nil, err
	}
	clone := req.Clone(req.Context())
	clone.URL = &url.URL{Scheme: server.Scheme, Host: server.Host, Path: req.URL.Path, RawQuery: req.URL.RawQuery}
	return r.base.RoundTrip(clone)
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricCandles(t.Context(), currency.EMPTYPAIR, asset.Spot, kline.OneMin, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetHistoricCandles(t.Context(), testPair, asset.Spot, kline.OneMin, time.Now().Add(-24*time.Hour), time.Now())
	assert.NoError(t, err, "GetHistoricCandles should not error")
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricCandlesExtended(t.Context(), testPair, asset.Spot, kline.OneMin, time.Now().Add(-time.Minute*2), time.Now())
	assert.NoError(t, err, "GetHistoricCandlesExtended should not error")
}

func TestFormatExchangeKlineInterval(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		interval kline.Interval
		output   string
	}{
		{
			kline.OneMin,
			"minute1",
		},
		{
			kline.OneHour,
			"hour1",
		},
		{
			kline.OneDay,
			"day1",
		},
		{
			kline.OneWeek,
			"week1",
		},
		{
			kline.FifteenDay,
			"",
		},
	} {
		t.Run(tc.interval.String(), func(t *testing.T) {
			t.Parallel()
			ret := e.FormatExchangeKlineInterval(tc.interval)
			assert.Equalf(t, tc.output, ret, "FormatExchangeKlineInterval(%s) should return %q", tc.interval, tc.output)
		})
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecentTrades(t.Context(), testPair, asset.Spot)
	assert.NoError(t, err, "GetRecentTrades should not error")
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricTrades(t.Context(), testPair, asset.Spot, time.Now().AddDate(69, 0, 0), time.Now())
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	_, err = e.GetHistoricTrades(t.Context(), currency.EMPTYPAIR, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetHistoricTrades(t.Context(), testPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	assert.NoError(t, err, "GetHistoricTrades should not error")
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTicker(t.Context(), testPair, asset.Spot)
	assert.NoError(t, err, "UpdateTicker should not error")
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := e.UpdateTickers(t.Context(), asset.Spot)
	assert.NoError(t, err, "UpdateTickers should not error")
}

func TestGetStatus(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		status int64
		resp   order.Status
	}{
		{status: -1, resp: order.Cancelled},
		{status: 0, resp: order.Active},
		{status: 1, resp: order.PartiallyFilled},
		{status: 2, resp: order.Filled},
		{status: 4, resp: order.Cancelling},
		{status: 5, resp: order.UnknownStatus},
	} {
		t.Run(tt.resp.String(), func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tt.resp.String(), e.GetStatus(tt.status).String(), "GetStatus(%d) should return %s", tt.status, tt.resp)
		})
	}
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	ts, err := e.GetServerTime(t.Context(), asset.Spot)
	require.NoError(t, err, "GetServerTime must not error")
	assert.NotZero(t, ts, "GetServerTime should return a non-zero time")
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	assert.NoError(t, err, "GetWithdrawalsHistory should not error")
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.CurrencyPairs.GetPairs(a, false)
		require.NoErrorf(t, err, "GetPairs must not error for asset %s", a)
		require.NotEmptyf(t, pairs, "GetPairs for asset %s must return pairs", a)
		resp, err := e.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}
