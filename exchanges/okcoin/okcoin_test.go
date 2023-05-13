package okcoin

import (
	"context"
	"errors"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

// Please supply you own test keys here for due diligence testing.
const (
	apiKey                  = ""
	apiSecret               = ""
	passphrase              = ""
	canManipulateRealOrders = false
)

var (
	o                    OKCoin
	spotCurrency         = currency.NewPairWithDelimiter(currency.BTC.String(), currency.USD.String(), "-")
	spotCurrencyLowerStr = spotCurrency.Lower().String()
	spotCurrencyUpperStr = spotCurrency.Upper().String()
)

func TestMain(m *testing.M) {
	o.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("Okcoin load config error", err)
	}
	okcoinConfig, err := cfg.GetExchangeConfig(o.Name)
	if err != nil {
		log.Fatalf("%v Setup() init error", o.Name)
	}

	okcoinConfig.API.AuthenticatedSupport = true
	okcoinConfig.API.AuthenticatedWebsocketSupport = true
	okcoinConfig.API.Credentials.Key = apiKey
	okcoinConfig.API.Credentials.Secret = apiSecret
	okcoinConfig.API.Credentials.ClientID = passphrase
	o.Websocket = sharedtestvalues.NewTestWebsocket()
	err = o.Setup(okcoinConfig)
	if err != nil {
		log.Fatal("OKCoin setup error", err)
	}
	os.Exit(m.Run())
}

func areTestAPIKeysSet() bool {
	return o.ValidateAPICredentials(o.GetDefaultCredentials()) == nil
}

func TestStart(t *testing.T) {
	t.Parallel()
	err := o.Start(context.Background(), nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrNilPointer)
	}
	var testWg sync.WaitGroup
	err = o.Start(context.Background(), &testWg)
	if err != nil {
		t.Fatal(err)
	}
	testWg.Wait()
}

func TestFetchTradablePair(t *testing.T) {
	t.Parallel()
	o.Verbose = true
	_, err := o.GetInstruments(context.Background(), "SPOT", "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetSystemStatus(t *testing.T) {
	t.Parallel()
	o.Verbose = true
	// allowed state value: ongoing, scheduled, processing, pre_open, completed, canceled
	_, err := o.GetSystemStatus(context.Background(), "scheduled")
	if err != nil {
		t.Fatal(err)
	}
	_, err = o.GetSystemStatus(context.Background(), "ongoing")
	if err != nil {
		t.Fatal(err)
	}
	_, err = o.GetSystemStatus(context.Background(), "processing")
	if err != nil {
		t.Fatal(err)
	}
	_, err = o.GetSystemStatus(context.Background(), "pre_open")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetSystemTime(t *testing.T) {
	t.Parallel()
	systemTime, err := o.GetSystemTime(context.Background())
	if err != nil {
		t.Fatal(err)
	} else {
		println(systemTime.String())
	}
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, err := o.GetTickers(context.Background(), "SPOT")
	if err != nil {
		t.Error(err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := o.GetTicker(context.Background(), "USDT-USD")
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderbooks(t *testing.T) {
	t.Parallel()
	_, err := o.GetOrderbook(context.Background(), "BTC-USD", 200)
	if err != nil {
		t.Error(err)
	}
}

func TestGetLiteOrderbook(t *testing.T) {
	t.Parallel()
	_, err := o.GetOrderbookLitebook(context.Background(), "BTC-USD")
	if err != nil {
		t.Error(err)
	}
}

func TestGetCandlestick(t *testing.T) {
	t.Parallel()
	_, err := o.GetCandlesticks(context.Background(), "BTC-USD", kline.FiveMin, time.Now().Add(-time.Hour*3), time.Now(), 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCandlestickHistory(t *testing.T) {
	t.Parallel()
	_, err := o.GetCandlestickHistory(context.Background(), "BTC-USD", time.Now().Add(-time.Minute*30), time.Now(), kline.FiveMin, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := o.GetTrades(context.Background(), "BTC-USD", 10)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := o.GetTradeHistory(context.Background(), "BTC-USD", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGet24HourTradingVolume(t *testing.T) {
	t.Parallel()
	_, err := o.Get24HourTradingVolume(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetOracle(t *testing.T) {
	t.Parallel()
	_, err := o.GetOracle(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetExchangeRate(t *testing.T) {
	t.Parallel()
	_, err := o.GetExchangeRate(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGenerateDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	_, err := o.GenerateDefaultSubscriptions()
	if err != nil {
		t.Error(err)
	}
}

func TestWsConnect(t *testing.T) {
	err := o.WsConnect()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second * 25)
}
