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
	"sync"
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

// TestGetOrderHistoryPagination ensures order history is crawled page by page.
// orders_info_history.do paginates through the current_page body parameter, so
// the page counter must advance once per page and not once per order.
func TestGetOrderHistoryPagination(t *testing.T) {
	t.Parallel()
	var (
		pagesMu sync.Mutex
		pages   []string
	)
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, r.ParseForm(), "the request body should parse as a form")
		assert.True(t, strings.HasSuffix(r.URL.Path, lbankQueryHistoryOrder), "GetOrderHistory should only request order history")

		page := r.Form.Get("current_page")
		pagesMu.Lock()
		pages = append(pages, page)
		pagesMu.Unlock()

		body := `{"result":"true","error_code":0,"current_page":3,"orders":[]}`
		switch page {
		case "1":
			body = `{"result":"true","error_code":0,"current_page":1,"orders":[` + orderHistoryEntry(orderHistoryFixture[0][0]) + "," + orderHistoryEntry(orderHistoryFixture[0][1]) + `]}`
		case "2":
			body = `{"result":"true","error_code":0,"current_page":2,"orders":[` + orderHistoryEntry(orderHistoryFixture[1][0]) + `]}`
		}
		_, err := fmt.Fprint(w, body)
		assert.NoError(t, err, "writing the order history response should not error")
	}))

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.API.AuthenticatedSupport = true
	ex.SkipAuthCheck = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-key", Secret: "mock-secret"})
	var err error
	ex.privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "the RSA test key must generate")
	require.NoError(t, ex.SetHTTPClient(orderHistoryClient(server)), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")

	got, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		Pairs:     currency.Pairs{testPair},
		Side:      order.AnySide,
		AssetType: asset.Spot,
		Type:      order.AnyType,
	})
	require.NoError(t, err, "GetOrderHistory must not error")

	pagesMu.Lock()
	gotPages := append([]string(nil), pages...)
	pagesMu.Unlock()
	assert.Equal(t, []string{"1", "2", "3"}, gotPages, "GetOrderHistory should request each page once, in order")
	assert.Equal(t, expectedOrderHistory(t, ex.Name), got, "GetOrderHistory should return the orders the exchange sent, mapped order for order")
}

// TestGetOrderHistoryPageLimit ensures a server that never serves an empty page
// cannot spin GetOrderHistory forever.
func TestGetOrderHistoryPageLimit(t *testing.T) {
	t.Parallel()
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, r.ParseForm(), "the request body should parse as a form")
		_, err := fmt.Fprintf(w, `{"result":"true","error_code":0,"current_page":1,"orders":[%s]}`, orderHistoryEntry(orderHistoryFixture[0][0]))
		assert.NoError(t, err, "writing the order history response should not error")
	}))

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.API.AuthenticatedSupport = true
	ex.SkipAuthCheck = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-key", Secret: "mock-secret"})
	var err error
	ex.privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "the RSA test key must generate")
	require.NoError(t, ex.SetHTTPClient(orderHistoryClient(server)), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")

	_, err = ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		Pairs:     currency.Pairs{testPair},
		Side:      order.AnySide,
		AssetType: asset.Spot,
		Type:      order.AnyType,
	})
	assert.ErrorIs(t, err, errOrderHistoryPageLimit, "GetOrderHistory should stop once the page limit is reached")
}

// orderHistoryClient returns the test server's client with the relative URL
// adapter installed.
func orderHistoryClient(server *httptest.Server) *http.Client {
	client := server.Client()
	client.Transport = &relativeURLTransport{base: client.Transport, serverURL: server.URL}
	return client
}

// relativeURLTransport resolves the relative request URLs SendAuthHTTPRequest
// builds against the test server, so this test exercises a real http.Client
// against a real server rather than a transport that answers on its behalf.
//
// SendAuthHTTPRequest assigns the caller's path straight to request.Item.Path
// without prefixing the configured endpoint URL, so a relative request never
// leaves the client at all:
//
//	Post "/v2/orders_info_history.do": unsupported protocol scheme ""
//
// That pre-existing bug is being fixed separately in #2272. Requests that
// already carry a host are passed through untouched, so this adapter turns into
// a no-op once #2272 lands and can be deleted with it.
type relativeURLTransport struct {
	base      http.RoundTripper
	serverURL string
}

func (r *relativeURLTransport) RoundTrip(req *http.Request) (*http.Response, error) {
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

// orderHistoryOrder describes one order the mocked LBank server serves. Every
// entry differs so a mapping that returns the wrong order, or that carries
// fields over from the previous one, cannot pass.
type orderHistoryOrder struct {
	OrderID    string
	Price      float64
	Amount     float64
	DealAmount float64
}

// orderHistoryFixture is the two page order history the mocked server serves.
// The third page, and every page after it, is empty.
var orderHistoryFixture = [][]orderHistoryOrder{
	{{"1", 10, 2, 1}, {"2", 20, 3, 2}},
	{{"3", 30, 4, 3}},
}

// orderHistoryTimestamp is the 2025-09-22T00:00:00Z created_time every fixture
// order carries. It is typed int64 so the millisecond value stays representable
// where the fixtures format it on 32-bit platforms.
const orderHistoryTimestamp int64 = 1758499200000

// orderHistoryEntry returns the JSON of a single LBank order history entry
func orderHistoryEntry(o orderHistoryOrder) string {
	return fmt.Sprintf(`{"order_id":%q,"symbol":"btc_usdt","type":"buy","price":%v,"amount":%v,"deal_amount":%v,"avg_price":%v,"status":0,"created_time":%d}`,
		o.OrderID, o.Price, o.Amount, o.DealAmount, o.Price, orderHistoryTimestamp)
}

// expectedOrderHistory returns the order.Detail values the fixture is expected
// to map onto, so the assertion pins content and not just the row count.
func expectedOrderHistory(t *testing.T, name string) order.FilteredOrders {
	t.Helper()
	pair, err := currency.NewPairFromString("btc_usdt")
	require.NoError(t, err, "the fixture symbol must parse")
	date := time.UnixMilli(orderHistoryTimestamp)
	var exp order.FilteredOrders
	for x := range orderHistoryFixture {
		for y := range orderHistoryFixture[x] {
			o := orderHistoryFixture[x][y]
			exp = append(exp, order.Detail{
				Price:                o.Price,
				Amount:               o.Amount,
				AverageExecutedPrice: o.Price,
				ExecutedAmount:       o.DealAmount,
				RemainingAmount:      o.Amount - o.DealAmount,
				Cost:                 o.Price * o.DealAmount,
				CostAsset:            pair.Quote,
				Fee:                  o.Amount * o.Price * 0.002,
				Exchange:             name,
				OrderID:              o.OrderID,
				Side:                 order.Buy,
				Status:               order.Active,
				Date:                 date,
				LastUpdated:          date,
				Pair:                 pair,
			})
		}
	}
	return exp
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
