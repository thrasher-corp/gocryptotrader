package exchange

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// change these if you wish to test another exchange and/or currency pair
const (
	exchName      = "BTC Markets" // change to test on another exchange
	exchAPIKEY    = ""
	exchAPISECRET = ""
	exchClientID  = ""
	pairs         = "BTC-AUD" // change to test another currency pair
	delimiter     = "-"
	assetType     = asset.Spot
	orderID       = "1234"
	orderType     = order.Limit
	orderSide     = order.Buy
	orderClientID = ""
	orderPrice    = 1
	orderAmount   = 1
)

var (
	settings = engine.Settings{
		CoreSettings: engine.CoreSettings{EnableDryRun: true},
		ConfigFile:   filepath.Join("..", "..", "..", "..", "testdata", "configtest.json"),
		DataDir:      filepath.Join("..", "..", "..", "..", "testdata", "gocryptotrader"),
	}
	exchangeTest = Exchange{}
)

func TestMain(m *testing.M) {
	var t int
	if err := setupEngine(); err != nil {
		fmt.Printf("Failed to configure exchange test cannot continue: %v", err)
		os.Exit(1)
	}
	t = m.Run()
	cleanup()
	os.Exit(t)
}

func TestExchange_Exchanges(t *testing.T) {
	t.Parallel()
	if x := exchangeTest.Exchanges(false); len(x) != 1 {
		t.Fatalf("expected 1 received %v", x)
	}
}

func TestExchange_GetExchange(t *testing.T) {
	t.Parallel()
	_, err := exchangeTest.GetExchange(exchName)
	if err != nil {
		t.Fatal(err)
	}
	_, err = exchangeTest.GetExchange("hello world")
	if err == nil {
		t.Fatal("unexpected error message received nil")
	}
}

func TestExchange_IsEnabled(t *testing.T) {
	t.Parallel()
	x := exchangeTest.IsEnabled(exchName)
	if !x {
		t.Fatal("expected return to be true")
	}
	x = exchangeTest.IsEnabled("fake_exchange")
	if x {
		t.Fatal("expected return to be false")
	}
}

func TestExchange_Ticker(t *testing.T) {
	t.Parallel()
	c, err := currency.NewPairDelimiter(pairs, delimiter)
	if err != nil {
		t.Fatal(err)
	}
	_, err = exchangeTest.Ticker(t.Context(), exchName, c, assetType)
	if err != nil {
		t.Fatal(err)
	}
}

func TestExchange_Orderbook(t *testing.T) {
	t.Parallel()
	c, err := currency.NewPairDelimiter(pairs, delimiter)
	if err != nil {
		t.Fatal(err)
	}
	_, err = exchangeTest.Orderbook(t.Context(), exchName, c, assetType)
	if err != nil {
		t.Fatal(err)
	}
}

func TestExchange_Pairs(t *testing.T) {
	t.Parallel()
	_, err := exchangeTest.Pairs(exchName, false, assetType)
	if err != nil {
		t.Fatal(err)
	}
	_, err = exchangeTest.Pairs(exchName, true, assetType)
	if err != nil {
		t.Fatal(err)
	}
}

func TestExchange_AccountBalances(t *testing.T) {
	if !configureExchangeKeys() {
		t.Skip("no exchange configured test skipped")
	}
	_, err := exchangeTest.AccountBalances(t.Context(),
		exchName, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
}

func TestExchange_QueryOrder(t *testing.T) {
	if !configureExchangeKeys() {
		t.Skip("no exchange configured test skipped")
	}
	t.Parallel()
	_, err := exchangeTest.QueryOrder(t.Context(),
		exchName, orderID, currency.EMPTYPAIR, assetType)
	if err != nil {
		t.Fatal(err)
	}
}

func TestExchange_SubmitOrder(t *testing.T) {
	if !configureExchangeKeys() {
		t.Skip("no exchange configured test skipped")
	}

	t.Parallel()
	c, err := currency.NewPairDelimiter(pairs, delimiter)
	if err != nil {
		t.Fatal(err)
	}
	tempOrder := &order.Submit{
		Pair:      c,
		Type:      orderType,
		Side:      orderSide,
		Price:     orderPrice,
		Amount:    orderAmount,
		ClientID:  orderClientID,
		Exchange:  exchName,
		AssetType: asset.Spot,
	}
	_, err = exchangeTest.SubmitOrder(t.Context(), tempOrder)
	if err != nil {
		t.Fatal(err)
	}
}

func TestExchange_CancelOrder(t *testing.T) {
	if !configureExchangeKeys() {
		t.Skip("no exchange configured test skipped")
	}
	t.Parallel()
	cp := currency.NewBTCUSD()
	a := asset.Spot
	_, err := exchangeTest.CancelOrder(t.Context(),
		exchName, orderID, cp, a)
	if err != nil {
		t.Fatal(err)
	}
}

func TestOHLCV(t *testing.T) {
	t.Parallel()
	cp := currency.NewPair(currency.BTC, currency.AUD)
	cp.Delimiter = currency.DashDelimiter
	calvinKline, err := exchangeTest.OHLCV(t.Context(), exchName, cp, assetType, time.Now().Add(-time.Hour*24).UTC(), time.Now().UTC(), kline.OneHour)
	if err != nil {
		t.Error(err)
	}
	if calvinKline.Exchange != exchName {
		t.Error("unexpected response")
	}
}

func setupEngine() (err error) {
	engine.Bot, err = engine.NewFromSettings(&settings, nil)
	if err != nil {
		return err
	}

	em := engine.NewExchangeManager()
	engine.Bot.ExchangeManager = em

	return engine.Bot.LoadExchange(exchName)
}

func cleanup() {
	err := os.RemoveAll(settings.DataDir)
	if err != nil {
		fmt.Printf("Clean up failed to remove file: %v manual removal may be required", err)
	}
}

func configureExchangeKeys() bool {
	ex, err := engine.Bot.GetExchangeByName(exchName)
	if err != nil {
		return false
	}
	b := ex.GetBase()
	b.SetCredentials(exchAPIKEY, exchAPISECRET, exchClientID, "", "", "")
	b.SkipAuthCheck = true
	return b.AreCredentialsValid(context.Background())
}
