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
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

// Please supply your own keys here for due diligence testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var (
	l        = &Lbank{}
	testPair = currency.NewBTCUSDT().Format(currency.PairFormat{Delimiter: "_"})
)

func TestMain(m *testing.M) {
	l = new(Lbank)
	if err := testexch.Setup(l); err != nil {
		log.Fatalf("Lbank Setup error: %s", err)
	}
	if apiKey != "" && apiSecret != "" {
		l.API.AuthenticatedSupport = true
		l.SetCredentials(apiKey, apiSecret, "", "", "", "")
	}
	os.Exit(m.Run())
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := l.GetTicker(t.Context(), testPair.String())
	assert.NoError(t, err, "GetTicker should not error")
}

func TestGetTimestamp(t *testing.T) {
	t.Parallel()
	ts, err := l.GetTimestamp(t.Context())
	require.NoError(t, err, "GetTimestamp must not error")
	assert.NotZero(t, ts, "GetTimestamp should return a non-zero time")
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	tickers, err := l.GetTickers(t.Context())
	require.NoError(t, err, "GetTickers must not error")
	assert.Greater(t, len(tickers), 1, "GetTickers should return more than 1 ticker")
}

func TestGetCurrencyPairs(t *testing.T) {
	t.Parallel()
	_, err := l.GetCurrencyPairs(t.Context())
	assert.NoError(t, err, "GetCurrencyPairs should not error")
}

func TestGetMarketDepths(t *testing.T) {
	t.Parallel()
	d, err := l.GetMarketDepths(t.Context(), testPair.String(), 4)
	require.NoError(t, err, "GetMarketDepths must not error")
	require.NotEmpty(t, d, "GetMarketDepths must return a non-empty response")
	assert.Len(t, d.Data.Asks, 4, "GetMarketDepths should return 4 asks")
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	r, err := l.GetTrades(t.Context(), testPair.String(), 420, time.Now())
	require.NoError(t, err, "GetTrades must not error")
	require.NotEmpty(t, r, "GetTrades must return a non-empty response")
	assert.Len(t, r, 420, "GetTrades should return 420 trades")
}

func TestGetKlines(t *testing.T) {
	t.Parallel()
	_, err := l.GetKlines(t.Context(), testPair.String(), "600", "minute1", time.Now())
	assert.NoError(t, err, "GetKlines should not error")
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	_, err := l.UpdateOrderbook(t.Context(), currency.EMPTYPAIR, asset.Spot)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = l.UpdateOrderbook(t.Context(), testPair, asset.Options)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	_, err = l.UpdateOrderbook(t.Context(), testPair, asset.Spot)
	assert.NoError(t, err, "UpdateOrderbook should not error")
}

func TestGetUserInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	_, err := l.GetUserInfo(t.Context())
	require.NoError(t, err, "GetUserInfo must not error")
}

func TestCreateOrder(t *testing.T) {
	t.Parallel()

	_, err := l.CreateOrder(t.Context(), testPair.String(), "what", 1231, 12314)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = l.CreateOrder(t.Context(), testPair.String(), order.Buy.String(), 0, 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = l.CreateOrder(t.Context(), testPair.String(), order.Sell.String(), 1231, 0)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, l, canManipulateRealOrders)

	_, err = l.CreateOrder(t.Context(), testPair.String(), order.Buy.String(), 58, 681)
	assert.NoError(t, err, "CreateOrder should not error")
}

func TestRemoveOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l, canManipulateRealOrders)

	_, err := l.RemoveOrder(t.Context(), testPair.String(), "24f7ce27-af1d-4dca-a8c1-ef1cbeec1b23")
	assert.NoError(t, err, "RemoveOrder should not error")
}

func TestQueryOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	_, err := l.QueryOrder(t.Context(), testPair.String(), "1")
	assert.NoError(t, err, "QueryOrder should not error")
}

func TestQueryOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	_, err := l.QueryOrderHistory(t.Context(), testPair.String(), "1", "100")
	assert.NoError(t, err, "QueryOrderHistory should not error")
}

func TestGetPairInfo(t *testing.T) {
	t.Parallel()
	_, err := l.GetPairInfo(t.Context())
	assert.NoError(t, err, "GetPairInfo should not error")
}

func TestOrderTransactionDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	_, err := l.OrderTransactionDetails(t.Context(), testPair.String(), "24f7ce27-af1d-4dca-a8c1-ef1cbeec1b23")
	assert.NoError(t, err, "OrderTransactionDetails should not error")
}

func TestTransactionHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	_, err := l.TransactionHistory(t.Context(), testPair.String(), "", "", "", "", "", "")
	assert.NoError(t, err, "TransactionHistory should not error")
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	_, err := l.GetOpenOrders(t.Context(), testPair.String(), "1", "50")
	assert.NoError(t, err, "GetOpenOrders should not error")
}

func TestUSD2RMBRate(t *testing.T) {
	t.Parallel()
	_, err := l.USD2RMBRate(t.Context())
	assert.NoError(t, err, "USD2RMBRate should not error")
}

func TestGetWithdrawConfig(t *testing.T) {
	t.Parallel()
	c, err := l.GetWithdrawConfig(t.Context(), currency.ETH)
	require.NoError(t, err, "GetWithdrawConfig must not error")
	assert.NotEmpty(t, c)
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l, canManipulateRealOrders)

	_, err := l.Withdraw(t.Context(), "", "", "", "", "", "")
	require.NoError(t, err, "Withdraw must not error")
}

func TestGetWithdrawRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	_, err := l.GetWithdrawalRecords(t.Context(), currency.ETH.String(), 0, 1, 20)
	assert.NoError(t, err, "GetWithdrawRecords should not error")
}

func TestLoadPrivKey(t *testing.T) {
	t.Parallel()

	l2 := new(Lbank)
	l2.SetDefaults()
	require.ErrorIs(t, l2.loadPrivKey(t.Context()), exchange.ErrCredentialsAreEmpty)

	ctx := account.DeployCredentialsToContext(t.Context(), &account.Credentials{Key: "test", Secret: "errortest"})
	assert.ErrorIs(t, l2.loadPrivKey(ctx), errPEMBlockIsNil)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der := x509.MarshalPKCS1PrivateKey(key)
	ctx = account.DeployCredentialsToContext(t.Context(), &account.Credentials{Key: "test", Secret: base64.StdEncoding.EncodeToString(der)})
	require.ErrorIs(t, l2.loadPrivKey(ctx), errUnableToParsePrivateKey)

	ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err = x509.MarshalPKCS8PrivateKey(ecdsaKey)
	require.NoError(t, err)
	ctx = account.DeployCredentialsToContext(t.Context(), &account.Credentials{Key: "test", Secret: base64.StdEncoding.EncodeToString(der)})
	require.ErrorIs(t, l2.loadPrivKey(ctx), common.ErrTypeAssertFailure)

	key, err = rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err = x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	ctx = account.DeployCredentialsToContext(t.Context(), &account.Credentials{Key: "test", Secret: base64.StdEncoding.EncodeToString(der)})
	assert.NoError(t, l2.loadPrivKey(ctx), "loadPrivKey should not error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	assert.NoError(t, l.loadPrivKey(t.Context()), "loadPrivKey should not error")
}

func TestSign(t *testing.T) {
	t.Parallel()

	l2 := new(Lbank)
	l2.SetDefaults()
	_, err := l2.sign("hello123")
	require.ErrorIs(t, err, errPrivateKeyNotLoaded)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "GenerateKey must not error")
	l2.privateKey = key

	targetMessage := "hello123"
	msg, err := l2.sign(targetMessage)
	require.NoError(t, err, "sign must not error")

	md5sum := md5.Sum([]byte(targetMessage)) //nolint:gosec // Used for this exchange
	shasum := sha256.Sum256([]byte(strings.ToUpper(hex.EncodeToString(md5sum[:]))))
	sigBytes, err := base64.StdEncoding.DecodeString(msg)
	require.NoError(t, err)
	err = rsa.VerifyPKCS1v15(&l2.privateKey.PublicKey, crypto.SHA256, shasum[:], sigBytes)
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	require.NoError(t, l.loadPrivKey(t.Context()), "loadPrivKey must not error")

	_, err = l.sign("hello123")
	assert.NoError(t, err, "sign should not error")
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, l, canManipulateRealOrders)

	r, err := l.SubmitOrder(t.Context(), &order.Submit{
		Exchange:  l.Name,
		Pair:      testPair,
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		ClientID:  "meowOrder",
		AssetType: asset.Spot,
	})
	if sharedtestvalues.AreAPICredentialsSet(l) {
		require.NoError(t, err, "SubmitOrder must not error")
		assert.Equal(t, order.New, r.Status, "SubmitOrder should return order status New")
	} else {
		assert.Error(t, err, "SubmitOrder should error when credentials are not set")
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l, canManipulateRealOrders)

	err := l.CancelOrder(t.Context(), &order.Cancel{
		Pair:      testPair,
		AssetType: asset.Spot,
		OrderID:   "24f7ce27-af1d-4dca-a8c1-ef1cbeec1b23",
	})
	assert.NoError(t, err, "CancelOrder should not error")
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	_, err := l.GetOrderInfo(t.Context(), "9ead39f5-701a-400b-b635-d7349eb0f6b", currency.EMPTYPAIR, asset.Spot)
	assert.NoError(t, err, "GetOrderInfo should not error")
}

func TestGetAllOpenOrderID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	_, err := l.getAllOpenOrderID(t.Context())
	assert.NoError(t, err, "getAllOpenOrderID should not error")
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	_, err := l.GetFeeByType(t.Context(), &exchange.FeeBuilder{
		Amount:  2,
		FeeType: exchange.CryptocurrencyWithdrawalFee,
		Pair:    testPair,
	})
	assert.NoError(t, err, "GetFeeByType should not error")
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	_, err := l.UpdateAccountInfo(t.Context(), asset.Spot)
	assert.NoError(t, err, "UpdateAccountInfo should not error")
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	_, err := l.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		Side:      order.AnySide,
		AssetType: asset.Spot,
		Type:      order.AnyType,
	})
	assert.NoError(t, err, "GetActiveOrders should not error")
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	_, err := l.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		Side:      order.AnySide,
		AssetType: asset.Spot,
		Type:      order.AnyType,
	})
	assert.NoError(t, err, "GetOrderHistory should not error")
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	_, err := l.GetHistoricCandles(t.Context(), currency.EMPTYPAIR, asset.Spot, kline.OneMin, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = l.GetHistoricCandles(t.Context(), testPair, asset.Spot, kline.OneMin, time.Now().Add(-24*time.Hour), time.Now())
	assert.NoError(t, err, "GetHistoricCandles should not error")
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	_, err := l.GetHistoricCandlesExtended(t.Context(), testPair, asset.Spot, kline.OneMin, time.Now().Add(-time.Minute*2), time.Now())
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
			ret := l.FormatExchangeKlineInterval(tc.interval)
			assert.Equalf(t, tc.output, ret, "FormatExchangeKlineInterval(%s) should return %q", tc.interval, tc.output)
		})
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := l.GetRecentTrades(t.Context(), testPair, asset.Spot)
	assert.NoError(t, err, "GetRecentTrades should not error")
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := l.GetHistoricTrades(t.Context(), testPair, asset.Spot, time.Now().AddDate(69, 0, 0), time.Now())
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	_, err = l.GetHistoricTrades(t.Context(), currency.EMPTYPAIR, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = l.GetHistoricTrades(t.Context(), testPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	assert.NoError(t, err, "GetHistoricTrades should not error")
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	_, err := l.UpdateTicker(t.Context(), testPair, asset.Spot)
	assert.NoError(t, err, "UpdateTicker should not error")
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := l.UpdateTickers(t.Context(), asset.Spot)
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
			assert.Equalf(t, tt.resp.String(), l.GetStatus(tt.status).String(), "GetStatus(%d) should return %s", tt.status, tt.resp)
		})
	}
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	ts, err := l.GetServerTime(t.Context(), asset.Spot)
	require.NoError(t, err, "GetServerTime must not error")
	assert.NotZero(t, ts, "GetServerTime should return a non-zero time")
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	_, err := l.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	assert.NoError(t, err, "GetWithdrawalsHistory should not error")
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, l)
	for _, a := range l.GetAssetTypes(false) {
		pairs, err := l.CurrencyPairs.GetPairs(a, false)
		require.NoErrorf(t, err, "GetPairs must not error for asset %s", a)
		require.NotEmptyf(t, pairs, "GetPairs for asset %s must return pairs", a)
		resp, err := l.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}
