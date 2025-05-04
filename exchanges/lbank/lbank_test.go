package lbank

import (
	"context"
	"log"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
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
	testAPIKey              = ""
	testAPISecret           = ""
	canManipulateRealOrders = false
	testCurrencyPair        = "btc_usdt"
)

var l = &Lbank{}

func TestMain(m *testing.M) {
	l.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}
	lbankConfig, err := cfg.GetExchangeConfig("Lbank")
	if err != nil {
		log.Fatal(err)
	}
	lbankConfig.API.AuthenticatedSupport = true
	lbankConfig.API.Credentials.Key = testAPIKey
	lbankConfig.API.Credentials.Secret = testAPISecret
	err = l.Setup(lbankConfig)
	if err != nil {
		log.Fatal(err)
	}
	err = l.UpdateTradablePairs(context.Background(), true)
	if err != nil {
		log.Fatal(err)
	}
	cp, err := currency.NewPairFromString(testCurrencyPair)
	if err != nil {
		log.Fatal(err)
	}
	err = l.CurrencyPairs.EnablePair(asset.Spot, cp)
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := l.GetTicker(t.Context(), testCurrencyPair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	tickers, err := l.GetTickers(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(tickers) <= 1 {
		t.Errorf("expected multiple tickers, received %v", len(tickers))
	}
}

func TestGetCurrencyPairs(t *testing.T) {
	t.Parallel()
	_, err := l.GetCurrencyPairs(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarketDepths(t *testing.T) {
	t.Parallel()
	_, err := l.GetMarketDepths(t.Context(), testCurrencyPair, "600", "1")
	if err != nil {
		t.Fatal(err)
	}
	a, _ := l.GetMarketDepths(t.Context(), testCurrencyPair, "4", "0")
	if len(a.Data.Asks) != 4 {
		t.Errorf("asks length requested doesn't match the output")
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := l.GetTrades(t.Context(), testCurrencyPair, 600, time.Now().Unix())
	if err != nil {
		t.Error(err)
	}
	a, err := l.GetTrades(t.Context(), testCurrencyPair, 600, 0)
	if len(a) != 600 && err != nil {
		t.Error(err)
	}
}

func TestGetKlines(t *testing.T) {
	t.Parallel()
	_, err := l.GetKlines(t.Context(),
		testCurrencyPair, "600", "minute1",
		strconv.FormatInt(time.Now().Unix(), 10))
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	p := currency.Pair{
		Delimiter: "_",
		Base:      currency.ETH,
		Quote:     currency.BTC,
	}

	_, err := l.UpdateOrderbook(t.Context(), p.Lower(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUserInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	_, err := l.GetUserInfo(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestCreateOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l, canManipulateRealOrders)

	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_")
	_, err := l.CreateOrder(t.Context(), cp.Lower().String(), "what", 1231, 12314)
	if err == nil {
		t.Error("CreateOrder error cannot be nil")
	}
	_, err = l.CreateOrder(t.Context(), cp.Lower().String(), order.Buy.Lower(), 0, 0)
	if err == nil {
		t.Error("CreateOrder error cannot be nil")
	}
	_, err = l.CreateOrder(t.Context(), cp.Lower().String(), order.Sell.Lower(), 1231, 0)
	if err == nil {
		t.Error("CreateOrder error cannot be nil")
	}
	_, err = l.CreateOrder(t.Context(), cp.Lower().String(), order.Buy.Lower(), 58, 681)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRemoveOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l, canManipulateRealOrders)

	cp := currency.NewPairWithDelimiter(currency.ETH.String(), currency.BTC.String(), "_")
	_, err := l.RemoveOrder(t.Context(),
		cp.Lower().String(), "24f7ce27-af1d-4dca-a8c1-ef1cbeec1b23")
	if err != nil {
		t.Error(err)
	}
}

func TestQueryOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_")
	_, err := l.QueryOrder(t.Context(), cp.Lower().String(), "1")
	if err != nil {
		t.Error(err)
	}
}

func TestQueryOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_")
	_, err := l.QueryOrderHistory(t.Context(),
		cp.Lower().String(), "1", "100")
	if err != nil {
		t.Error(err)
	}
}

func TestGetPairInfo(t *testing.T) {
	t.Parallel()
	_, err := l.GetPairInfo(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestOrderTransactionDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	_, err := l.OrderTransactionDetails(t.Context(),
		testCurrencyPair, "24f7ce27-af1d-4dca-a8c1-ef1cbeec1b23")
	if err != nil {
		t.Error(err)
	}
}

func TestTransactionHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	_, err := l.TransactionHistory(t.Context(),
		testCurrencyPair, "", "", "", "", "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_")
	_, err := l.GetOpenOrders(t.Context(), cp.Lower().String(), "1", "50")
	if err != nil {
		t.Error(err)
	}
}

func TestUSD2RMBRate(t *testing.T) {
	t.Parallel()
	_, err := l.USD2RMBRate(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawConfig(t *testing.T) {
	t.Parallel()
	_, err := l.GetWithdrawConfig(t.Context(),
		currency.ETH.Lower().String())
	if err != nil {
		t.Error(err)
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l, canManipulateRealOrders)

	_, err := l.Withdraw(t.Context(), "", "", "", "", "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	_, err := l.GetWithdrawalRecords(t.Context(), currency.ETH.Lower().String(), 0, 1, 20)
	if err != nil {
		t.Error(err)
	}
}

func TestLoadPrivKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	err := l.loadPrivKey(t.Context())
	if err != nil {
		t.Error(err)
	}

	ctx := account.DeployCredentialsToContext(t.Context(),
		&account.Credentials{Secret: "errortest"})
	err = l.loadPrivKey(ctx)
	if err == nil {
		t.Errorf("Expected error due to pemblock nil")
	}
}

func TestSign(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	err := l.loadPrivKey(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	_, err = l.sign("hello123")
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, l, canManipulateRealOrders)

	orderSubmission := &order.Submit{
		Exchange: l.Name,
		Pair: currency.Pair{
			Base:      currency.BTC,
			Quote:     currency.USDT,
			Delimiter: "_",
		},
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		ClientID:  "meowOrder",
		AssetType: asset.Spot,
	}
	response, err := l.SubmitOrder(t.Context(), orderSubmission)
	if sharedtestvalues.AreAPICredentialsSet(l) && (err != nil || response.Status != order.New) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !sharedtestvalues.AreAPICredentialsSet(l) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l, canManipulateRealOrders)

	cp := currency.NewPairWithDelimiter(currency.ETH.String(), currency.BTC.String(), "_")
	var a order.Cancel
	a.Pair = cp
	a.AssetType = asset.Spot
	a.OrderID = "24f7ce27-af1d-4dca-a8c1-ef1cbeec1b23"
	err := l.CancelOrder(t.Context(), &a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	_, err := l.GetOrderInfo(t.Context(),
		"9ead39f5-701a-400b-b635-d7349eb0f6b", currency.EMPTYPAIR, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllOpenOrderID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	_, err := l.getAllOpenOrderID(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_")
	var input exchange.FeeBuilder
	input.Amount = 2
	input.FeeType = exchange.CryptocurrencyWithdrawalFee
	input.Pair = cp
	_, err := l.GetFeeByType(t.Context(), &input)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	_, err := l.UpdateAccountInfo(t.Context(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	var input order.MultiOrderRequest
	input.Side = order.Buy
	input.AssetType = asset.Spot
	input.Type = order.AnyType
	input.Side = order.AnySide
	_, err := l.GetActiveOrders(t.Context(), &input)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	var input order.MultiOrderRequest
	input.Side = order.Buy
	input.AssetType = asset.Spot
	input.Type = order.AnyType
	input.Side = order.AnySide
	_, err := l.GetOrderHistory(t.Context(), &input)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString(testCurrencyPair)
	if err != nil {
		t.Fatal(err)
	}
	_, err = l.GetHistoricCandles(t.Context(), cp, asset.Spot, kline.OneMin, time.Now().Add(-24*time.Hour), time.Now())
	if err != nil {
		t.Fatal(err)
	}

	_, err = l.GetHistoricCandles(t.Context(), cp, asset.Spot, kline.OneHour, time.Now().Add(-24*time.Hour), time.Now())
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	startTime := time.Now().Add(-time.Minute * 2)
	end := time.Now()
	cp, err := currency.NewPairFromString(testCurrencyPair)
	if err != nil {
		t.Fatal(err)
	}
	_, err = l.GetHistoricCandlesExtended(t.Context(), cp, asset.Spot, kline.OneMin, startTime, end)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_FormatExchangeKlineInterval(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		interval kline.Interval
		output   string
	}{
		{
			"OneMin",
			kline.OneMin,
			"minute1",
		},
		{
			"OneHour",
			kline.OneHour,
			"hour1",
		},
		{
			"OneDay",
			kline.OneDay,
			"day1",
		},
		{
			"OneWeek",
			kline.OneWeek,
			"week1",
		},
		{
			"AllOther",
			kline.FifteenDay,
			"",
		},
	}

	for x := range testCases {
		test := testCases[x]

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ret := l.FormatExchangeKlineInterval(test.interval)

			if ret != test.output {
				t.Fatalf("unexpected result return expected: %v received: %v", test.output, ret)
			}
		})
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString(testCurrencyPair)
	if err != nil {
		t.Fatal(err)
	}
	_, err = l.GetRecentTrades(t.Context(), currencyPair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString(testCurrencyPair)
	if err != nil {
		t.Fatal(err)
	}
	_, err = l.GetHistoricTrades(t.Context(),
		currencyPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	if err != nil {
		t.Error(err)
	}
	// longer term
	_, err = l.GetHistoricTrades(t.Context(),
		currencyPair, asset.Spot, time.Now().Add(-time.Minute*60*200), time.Now().Add(-time.Minute*60*199))
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString(testCurrencyPair)
	if err != nil {
		t.Fatal(err)
	}
	_, err = l.UpdateTicker(t.Context(), cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := l.UpdateTickers(t.Context(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
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
		t.Run("", func(t *testing.T) {
			t.Parallel()
			resp := l.GetStatus(tt.status)
			if resp != tt.resp {
				t.Fatalf("received: '%v' but expected: '%v'", resp, tt.resp)
			}
		})
	}
}

func TestGetTimestamp(t *testing.T) {
	t.Parallel()
	tt, err := l.GetTimestamp(t.Context())
	if err != nil {
		t.Error(err)
	}
	if tt.IsZero() {
		t.Error("expected time")
	}
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	tt, err := l.GetServerTime(t.Context(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
	if tt.IsZero() {
		t.Error("expected time")
	}
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, l)

	_, err := l.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, l)
	for _, a := range l.GetAssetTypes(false) {
		pairs, err := l.CurrencyPairs.GetPairs(a, false)
		require.NoError(t, err, "cannot get pairs for %s", a)
		require.NotEmpty(t, pairs, "no pairs for %s", a)
		resp, err := l.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}
