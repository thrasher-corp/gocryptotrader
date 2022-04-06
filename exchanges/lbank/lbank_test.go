package lbank

import (
	"context"
	"errors"
	"log"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Please supply your own keys here for due diligence testing
const (
	testAPIKey              = ""
	testAPISecret           = ""
	canManipulateRealOrders = false
	testCurrencyPair        = "btc_usdt"
)

var l Lbank

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
	os.Exit(m.Run())
}

func areTestAPIKeysSet() bool {
	return l.ValidateAPICredentials(l.GetDefaultCredentials()) == nil
}

func TestStart(t *testing.T) {
	t.Parallel()
	err := l.Start(nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrNilPointer)
	}
	var testWg sync.WaitGroup
	err = l.Start(&testWg)
	if err != nil {
		t.Fatal(err)
	}
	testWg.Wait()
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := l.GetTicker(context.Background(), testCurrencyPair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	tickers, err := l.GetTickers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(tickers) <= 1 {
		t.Errorf("expected multiple tickers, received %v", len(tickers))
	}
}

func TestGetCurrencyPairs(t *testing.T) {
	t.Parallel()
	_, err := l.GetCurrencyPairs(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarketDepths(t *testing.T) {
	t.Parallel()
	_, err := l.GetMarketDepths(context.Background(), testCurrencyPair, "600", "1")
	if err != nil {
		t.Fatal(err)
	}
	a, _ := l.GetMarketDepths(context.Background(), testCurrencyPair, "4", "0")
	if len(a.Data.Asks) != 4 {
		t.Errorf("asks length requested doesnt match the output")
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := l.GetTrades(context.Background(), testCurrencyPair, 600, time.Now().Unix())
	if err != nil {
		t.Error(err)
	}
	a, err := l.GetTrades(context.Background(), testCurrencyPair, 600, 0)
	if len(a) != 600 && err != nil {
		t.Error(err)
	}
}

func TestGetKlines(t *testing.T) {
	t.Parallel()
	_, err := l.GetKlines(context.Background(),
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
		Quote:     currency.BTC}

	_, err := l.UpdateOrderbook(context.Background(), p.Lower(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUserInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := l.GetUserInfo(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestCreateOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_")
	_, err := l.CreateOrder(context.Background(), cp.Lower().String(), "what", 1231, 12314)
	if err == nil {
		t.Error("CreateOrder error cannot be nil")
	}
	_, err = l.CreateOrder(context.Background(), cp.Lower().String(), order.Buy.Lower(), 0, 0)
	if err == nil {
		t.Error("CreateOrder error cannot be nil")
	}
	_, err = l.CreateOrder(context.Background(), cp.Lower().String(), order.Sell.Lower(), 1231, 0)
	if err == nil {
		t.Error("CreateOrder error cannot be nil")
	}
	_, err = l.CreateOrder(context.Background(), cp.Lower().String(), order.Buy.Lower(), 58, 681)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRemoveOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	cp := currency.NewPairWithDelimiter(currency.ETH.String(), currency.BTC.String(), "_")
	_, err := l.RemoveOrder(context.Background(),
		cp.Lower().String(), "24f7ce27-af1d-4dca-a8c1-ef1cbeec1b23")
	if err != nil {
		t.Error(err)
	}
}

func TestQueryOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_")
	_, err := l.QueryOrder(context.Background(), cp.Lower().String(), "1")
	if err != nil {
		t.Error(err)
	}
}

func TestQueryOrderHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_")
	_, err := l.QueryOrderHistory(context.Background(),
		cp.Lower().String(), "1", "100")
	if err != nil {
		t.Error(err)
	}
}

func TestGetPairInfo(t *testing.T) {
	t.Parallel()
	_, err := l.GetPairInfo(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestOrderTransactionDetails(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := l.OrderTransactionDetails(context.Background(),
		testCurrencyPair, "24f7ce27-af1d-4dca-a8c1-ef1cbeec1b23")
	if err != nil {
		t.Error(err)
	}
}

func TestTransactionHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := l.TransactionHistory(context.Background(),
		testCurrencyPair, "", "", "", "", "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_")
	_, err := l.GetOpenOrders(context.Background(), cp.Lower().String(), "1", "50")
	if err != nil {
		t.Error(err)
	}
}

func TestUSD2RMBRate(t *testing.T) {
	t.Parallel()
	_, err := l.USD2RMBRate(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawConfig(t *testing.T) {
	t.Parallel()
	_, err := l.GetWithdrawConfig(context.Background(),
		currency.ETH.Lower().String())
	if err != nil {
		t.Error(err)
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	_, err := l.Withdraw(context.Background(), "", "", "", "", "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawRecords(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := l.GetWithdrawalRecords(context.Background(),
		currency.ETH.Lower().String(),
		"0", "1", "20")
	if err != nil {
		t.Error(err)
	}
}

func TestLoadPrivKey(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	err := l.loadPrivKey(context.Background())
	if err != nil {
		t.Error(err)
	}

	ctx := exchange.DeployCredentialsToContext(context.Background(), &exchange.Credentials{Secret: "errortest"})
	err = l.loadPrivKey(ctx)
	if err == nil {
		t.Errorf("Expected error due to pemblock nil")
	}
}

func TestSign(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	err := l.loadPrivKey(context.Background())
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
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var orderSubmission = &order.Submit{
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
	response, err := l.SubmitOrder(context.Background(), orderSubmission)
	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	cp := currency.NewPairWithDelimiter(currency.ETH.String(), currency.BTC.String(), "_")
	var a order.Cancel
	a.Pair = cp
	a.AssetType = asset.Spot
	a.ID = "24f7ce27-af1d-4dca-a8c1-ef1cbeec1b23"
	err := l.CancelOrder(context.Background(), &a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := l.GetOrderInfo(context.Background(),
		"9ead39f5-701a-400b-b635-d7349eb0f6b", currency.EMPTYPAIR, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllOpenOrderID(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := l.getAllOpenOrderID(context.Background())
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
	_, err := l.GetFeeByType(context.Background(), &input)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := l.UpdateAccountInfo(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	var input order.GetOrdersRequest
	input.Side = order.Buy
	input.AssetType = asset.Spot
	_, err := l.GetOrderHistory(context.Background(), &input)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("eth_btc")
	if err != nil {
		t.Fatal(err)
	}
	_, err = l.GetHistoricCandles(context.Background(),
		pair, asset.Spot, time.Now().Add(-24*time.Hour), time.Now(), kline.OneMin)
	if err != nil {
		t.Fatal(err)
	}

	_, err = l.GetHistoricCandles(context.Background(),
		pair, asset.Spot, time.Now().Add(-24*time.Hour), time.Now(), kline.OneHour)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()

	startTime := time.Now().Add(-time.Minute * 2)
	end := time.Now()
	pair, err := currency.NewPairFromString("eth_btc")
	if err != nil {
		t.Fatal(err)
	}
	_, err = l.GetHistoricCandlesExtended(context.Background(),
		pair, asset.Spot, startTime, end, kline.OneMin)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_FormatExchangeKlineInterval(t *testing.T) {
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
	_, err = l.GetRecentTrades(context.Background(), currencyPair, asset.Spot)
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
	_, err = l.GetHistoricTrades(context.Background(),
		currencyPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	if err != nil {
		t.Error(err)
	}
	// longer term
	_, err = l.GetHistoricTrades(context.Background(),
		currencyPair, asset.Spot, time.Now().Add(-time.Minute*60*200), time.Now().Add(-time.Minute*60*199))
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("eth_btc")
	if err != nil {
		t.Fatal(err)
	}
	_, err = l.UpdateTicker(context.Background(), cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := l.UpdateTickers(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}
