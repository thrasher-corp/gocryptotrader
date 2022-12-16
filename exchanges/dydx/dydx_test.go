package dydx

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var dy DYDX

func TestMain(m *testing.M) {
	dy.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}

	exchCfg, err := cfg.GetExchangeConfig("DYDX")
	if err != nil {
		log.Fatal(err)
	}

	exchCfg.API.AuthenticatedSupport = true
	exchCfg.API.AuthenticatedWebsocketSupport = true
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret

	err = dy.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}
	setupWS()
	os.Exit(m.Run())
}

// Ensures that this exchange package is compatible with IBotExchange
func TestInterface(t *testing.T) {
	var e exchange.IBotExchange
	if e = new(DYDX); e == nil {
		t.Fatal("unable to allocate exchange")
	}
}

func areTestAPIKeysSet() bool {
	return dy.ValidateAPICredentials(dy.GetDefaultCredentials()) == nil
}

// Implement tests for API endpoints below

var instrumentJSON = `{	"markets": {	  "LINK-USD": {	  "market": "LINK-USD",	  "status": "ONLINE",	  "baseAsset": "LINK",	  "quoteAsset": "USD",	  "stepSize": "0.1",	  "tickSize": "0.01",	  "indexPrice": "12",	  "oraclePrice": "101",	  "priceChange24H": "0",	  "nextFundingRate": "0.0000125000",	  "nextFundingAt": "2021-03-01T18:00:00.000Z",	  "minOrderSize": "1",	  "type": "PERPETUAL",	  "initialMarginFraction": "0.10",	  "maintenanceMarginFraction": "0.05",	  "baselinePositionSize": "1000",	  "incrementalPositionSize": "1000",	  "incrementalInitialMarginFraction": "0.2",	  "volume24H": "0",	  "trades24H": "0",	  "openInterest": "0",	  "maxPositionSize": "10000",	  "assetResolution": "10000000",	  "syntheticAssetId": "0x4c494e4b2d37000000000000000000"	}	}}`

func TestGetInstruments(t *testing.T) {
	t.Parallel()
	var instrumentData InstrumentDatas
	err := json.Unmarshal([]byte(instrumentJSON), &instrumentData)
	if err != nil {
		t.Error(err)
	}
	if _, err := dy.GetMarkets(context.Background(), ""); err != nil {
		t.Error(err)
	}
}

func TestGetOrderbooks(t *testing.T) {
	t.Parallel()
	if _, err := dy.GetOrderbooks(context.Background(), "CRV-USD"); err != nil {
		t.Error(err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	if _, err := dy.GetTrades(context.Background(), "CRV-USD", time.Time{}, 5); err != nil {
		t.Error(err)
	}
}

func TestGetFastWithdrawalLiquidity(t *testing.T) {
	t.Parallel()
	if _, err := dy.GetFastWithdrawalLiquidity(context.Background(), "", 0, 0); err != nil {
		t.Error(err)
	}
}

func TestGetMarketStats(t *testing.T) {
	t.Parallel()
	dy.Verbose = true
	if _, err := dy.GetMarketStats(context.Background(), "", 7); err != nil {
		t.Error(err)
	}
}

func TestGetHistoricalFunding(t *testing.T) {
	t.Parallel()
	if _, err := dy.GetHistoricalFunding(context.Background(), "CRV-USD", time.Time{}); err != nil {
		t.Error(err)
	}
}

func TestGetCandlesForMarket(t *testing.T) {
	t.Parallel()
	if _, err := dy.GetCandlesForMarket(context.Background(), "CRV-USD", kline.FiveMin, "", "", 10); err != nil {
		t.Error()
	}
}

func TestGetGlobalConfigurationVariables(t *testing.T) {
	t.Parallel()
	if _, err := dy.GetGlobalConfigurationVariables(context.Background()); err != nil {
		t.Error(err)
	}
}

func TestCheckIfUserExists(t *testing.T) {
	t.Parallel()
	if _, err := dy.CheckIfUserExists(context.Background(), ""); err != nil {
		t.Error(err)
	}
}

func TestCheckIfUsernameExists(t *testing.T) {
	t.Parallel()
	if _, err := dy.CheckIfUsernameExists(context.Background(), ""); err != nil {
		t.Error(err)
	}
}

func TestGetAPIServerTime(t *testing.T) {
	t.Parallel()
	if _, err := dy.GetAPIServerTime(context.Background()); err != nil {
		t.Error(err)
	}
}

func TestGetPublicLeaderboardPNLs(t *testing.T) {
	t.Parallel()
	if _, err := dy.GetPublicLeaderboardPNLs(context.Background(), "DAILY", "ABSOLUTE", time.Time{}, 2); err != nil {
		t.Error(err)
	}
}

func TestGetPublicRetroactiveMiningReqards(t *testing.T) {
	t.Parallel()
	if _, err := dy.GetPublicRetroactiveMiningReqards(context.Background(), ""); err != nil {
		t.Error(err)
	}
}

func TestVerifyEmailAddress(t *testing.T) {
	t.Parallel()
	if _, err := dy.VerifyEmailAddress(context.Background(), "1234"); err != nil && !strings.Contains(err.Error(), "Not Found") {
		t.Error(err)
	}
}

func TestGetCurrentlyRevealedHedgies(t *testing.T) {
	t.Parallel()
	if _, err := dy.GetCurrentlyRevealedHedgies(context.Background(), "", ""); err != nil {
		t.Error(err)
	}
}

func TestGetHistoricallyRevealedHedgies(t *testing.T) {
	t.Parallel()
	if _, err := dy.GetHistoricallyRevealedHedgies(context.Background(), "daily", 1, 10); err != nil {
		t.Error(err)
	}
}

func TestGetInsuranceFundBalance(t *testing.T) {
	t.Parallel()
	if _, err := dy.GetInsuranceFundBalance(context.Background()); err != nil {
		t.Error(err)
	}
}

func TestGetPublicProfile(t *testing.T) {
	t.Parallel()
	if _, err := dy.GetPublicProfile(context.Background(), "some_public_profile"); err != nil && !strings.Contains(err.Error(), "User not found") {
		t.Error(err)
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	if _, err := dy.FetchTradablePairs(context.Background(), asset.Spot); err != nil {
		t.Error(err)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	pair := currency.NewPair(currency.BTC, currency.USD)
	startTime := time.Date(2020, 9, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2021, 2, 15, 0, 0, 0, 0, time.UTC)
	_, err := dy.GetHistoricCandles(context.Background(), pair, asset.Spot, startTime, endTime, kline.Interval(time.Hour*5))
	if err != nil && !strings.Contains(err.Error(), "interval not supported") {
		t.Errorf("%s GetHistoricCandles() expected %s, but found %v", "interval not supported", dy.Name, err)
	}
	_, err = dy.GetHistoricCandles(context.Background(), pair, asset.Spot, time.Time{}, time.Time{}, kline.Interval(time.Hour*4))
	if err != nil {
		t.Errorf("%s GetHistoricCandles() error %s", err, dy.Name)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	if _, err := dy.GetHistoricTrades(context.Background(), currency.NewPair(currency.BTC, currency.USD), asset.Spot, time.Time{} /*Now().Add(-time.Minute*4)*/, time.Now().Add(-time.Minute*2)); err != nil {
		t.Errorf("%s GetHistoricTrades() error %v", dy.Name, err)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	if _, err := dy.GetRecentTrades(context.Background(), currency.NewPair(currency.BTC, currency.USD), asset.Spot); err != nil {
		t.Errorf("%s GetRecentTrades() error %s", dy.Name, err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	if _, err := dy.UpdateOrderbook(context.Background(), currency.NewPair(currency.BTC, currency.NewCode("USD")), asset.Spot); err != nil {
		t.Errorf("%s UpdateOrderbook() error %s", err, dy.Name)
	}
}

func TestFetchOrderbook(t *testing.T) {
	t.Parallel()
	if _, err := dy.FetchOrderbook(context.Background(), currency.NewPair(currency.BTC, currency.USD), asset.Spot); err != nil {
		t.Errorf("%v FetchOrderbook() error %v", dy.Name, err)
	}
}

func TestFetchTicker(t *testing.T) {
	t.Parallel()
	if _, err := dy.FetchTicker(context.Background(), currency.NewPair(currency.BTC, currency.USD), asset.Spot); err != nil {
		t.Errorf("%s FetchTicker() error %v", dy.Name, err)
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	if err := dy.UpdateTickers(context.Background(), asset.Spot); err != nil {
		t.Errorf("%s UpdateTicker() error %v", dy.Name, err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	if _, err := dy.UpdateTicker(context.Background(), currency.NewPair(currency.BTC, currency.USD), asset.Spot); err != nil {
		t.Errorf("%s UpdateTicker() error %v", dy.Name, err)
	}
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	if err := dy.UpdateTradablePairs(context.Background(), true); err != nil {
		t.Errorf("%s UpdateTradablePairs() error %v", dy.Name, err)
	}
}

func TestWsConnect(t *testing.T) {
	t.Parallel()
	dy.Verbose = true
	if err := dy.WsConnect(); err != nil {
		t.Error(err)
	}
}

func setupWS() {
	if !dy.Websocket.IsEnabled() {
		return
	}
	if !areTestAPIKeysSet() {
		dy.Websocket.SetCanUseAuthenticatedEndpoints(false)
	}
	err := dy.WsConnect()
	if err != nil {
		log.Fatal(err)
	}
}

func TestGenerateDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	if subscriptions, err := dy.GenerateDefaultSubscriptions(); err != nil {
		t.Error(err)
	} else {
		for x := range subscriptions {
			val, _ := json.Marshal(subscriptions[x])
			println(string(val))
		}
	}
}

func TestSubscribe(t *testing.T) {
	t.Parallel()
	if err := dy.Subscribe([]stream.ChannelSubscription{
		{
			Channel: "v3_orderbook",
			Currency: currency.Pair{
				Base:      currency.LTC,
				Delimiter: currency.DashDelimiter,
				Quote:     currency.USD,
			},
		},
	}); err != nil {
		t.Error(err)
	}
}
