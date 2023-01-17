package cryptodotcom

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var cr Cryptodotcom

func TestMain(m *testing.M) {
	cr.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}
	exchCfg, err := cfg.GetExchangeConfig("Cryptodotcom")
	if err != nil {
		log.Fatal(err)
	}
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret
	if apiKey != "" && apiSecret != "" {
		exchCfg.API.AuthenticatedSupport = true
		exchCfg.API.AuthenticatedWebsocketSupport = true
	}
	cr.Websocket = sharedtestvalues.NewTestWebsocket()
	err = cr.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}
	cr.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	cr.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
	os.Exit(m.Run())
}

// Ensures that this exchange package is compatible with IBotExchange
func TestInterface(t *testing.T) {
	var e exchange.IBotExchange
	if e = new(Cryptodotcom); e == nil {
		t.Fatal("unable to allocate exchange")
	}
}

func areTestAPIKeysSet() bool {
	return cr.ValidateAPICredentials(cr.GetDefaultCredentials()) == nil
}

// Implement tests for API endpoints below

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	_, err := cr.GetInstruments(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := cr.GetOrderbook(context.Background(), "BTC_USDT", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCandlestickDetail(t *testing.T) {
	t.Parallel()
	_, err := cr.GetCandlestickDetail(context.Background(), "BTC_USDT", kline.FiveMin)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := cr.GetTicker(context.Background(), "BTC_USDT")
	if err != nil {
		t.Error(err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := cr.GetTrades(context.Background(), "BTC_USDT")
	if err != nil {
		t.Error(err)
	}
}

func TestWithdrawFunds(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	_, err := cr.WithdrawFunds(context.Background(), currency.BTC, 10, core.BitcoinDonationAddress, "", "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetCurrencyNetworks(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	_, err := cr.GetCurrencyNetworks(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawalHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Parallel()
	}
	_, err := cr.GetWithdrawalHistory(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetDepositHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Parallel()
	}
	_, err := cr.GetDepositHistory(context.Background(), currency.EMPTYCODE, time.Time{}, time.Time{}, 20, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPersonalDepositAddress(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	_, err := cr.GetPersonalDepositAddress(context.Background(), currency.BTC)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountSummary(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	_, err := cr.GetAccountSummary(context.Background(), currency.USDT)
	if err != nil {
		t.Error(err)
	}
}
