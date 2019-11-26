package exchange

import (
	"fmt"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	settings = engine.Settings{
		ConfigFile:          "../../../testdata/gctscript/config.json",
		EnableDryRun:        true,
		DataDir:             "../../../testdata/gocryptotrader",
		Verbose:             false,
		EnableGRPC:          false,
		EnableDeprecatedRPC: false,
		EnableWebsocketRPC:  false,
	}
	exchangeTest = Exchange{}
)

const (
	exchName  = "BTC Markets" // change to test on another exchange
	pairs     = "BTC-AUD"     // change to test another currency pair
	delimiter = "-"
	assetType = asset.Spot
	orderID   = "1234"
)

func TestMain(m *testing.M) {
	var t int
	err := setupEngine()
	if err != nil {
		fmt.Println("Failed to configure exchange test cannot continue")
		os.Exit(1)
	}
	t = m.Run()
	err = cleanup()
	if err != nil {
		fmt.Printf("Clean up failed %v", err)
	}
	os.Exit(t)
}

func TestExchange_Exchanges(t *testing.T) {
	t.Parallel()
	x := exchangeTest.Exchanges(false)
	y := len(x)
	if y != 2 {
		t.Fatalf("expected only one result received %v", y)
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
	x = exchangeTest.IsEnabled("yobit")
	if x {
		t.Fatal("expected return to be false")
	}
}

func TestExchange_Ticker(t *testing.T) {
	t.Parallel()
	c := currency.NewPairDelimiter(pairs, delimiter)
	_, err := exchangeTest.Ticker(exchName, c, assetType)
	if err != nil {
		t.Fatal(err)
	}
}

func TestExchange_Orderbook(t *testing.T) {
	t.Parallel()
	c := currency.NewPairDelimiter(pairs, delimiter)
	_, err := exchangeTest.Orderbook(exchName, c, assetType)
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

// func TestExchange_AccountInformation(t *testing.T) {
// 	t.Parallel()
// 	_, err := exchangeTest.AccountInformation(exchName)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// }
//
// func TestExchange_QueryOrder(t *testing.T) {
// 	t.Parallel()
// 	_, err := exchangeTest.QueryOrder(exchName, orderID)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// }
//
// func TestExchange_SubmitOrder(t *testing.T) {
// 	t.Parallel()
// 	tempOrder := &order.Submit{}
// 	_, err := exchangeTest.SubmitOrder(exchName, tempOrder)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// }
//
// func TestExchange_CancelOrder(t *testing.T) {
// 	t.Parallel()
// 	_, err := exchangeTest.CancelOrder(exchName, orderID)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// }

func setupEngine() (err error) {
	engine.Bot, err = engine.NewFromSettings(&settings)
	if engine.Bot == nil || err != nil {
		return err
	}

	return engine.Bot.Start()
}

func cleanup() (err error) {
	err = os.RemoveAll(settings.DataDir)
	if err != nil {
		return
	}
	return nil
}
