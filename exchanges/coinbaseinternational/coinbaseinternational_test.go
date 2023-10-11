package coinbaseinternational

import (
	"context"
	"errors"
	"log"
	"os"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	passphrase              = ""
	canManipulateRealOrders = false
)

var co = &CoinbaseInternational{}

func TestMain(m *testing.M) {
	co.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}

	exchCfg, err := cfg.GetExchangeConfig("Coinbaseinternational")
	if err != nil {
		log.Fatal(err)
	}

	exchCfg.Enabled = true
	exchCfg.API.AuthenticatedSupport = true
	exchCfg.API.AuthenticatedWebsocketSupport = true
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret
	exchCfg.API.Credentials.ClientID = passphrase
	co.Websocket = sharedtestvalues.NewTestWebsocket()
	err = co.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}
	co.SetClientProxyAddress("http://ci:bE8OJv9gknuYbuiseFLL@35.77.58.161:3128")
	os.Exit(m.Run())
}

// Ensures that this exchange package is compatible with IBotExchange
func TestInterface(t *testing.T) {
	var e exchange.IBotExchange
	if e = new(CoinbaseInternational); e == nil {
		t.Fatal("unable to allocate exchange")
	}
}

// Implement tests for API endpoints below

func TestListAssets(t *testing.T) {
	t.Parallel()
	_, err := co.ListAssets(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetAssetDetails(t *testing.T) {
	t.Parallel()
	_, err := co.GetAssetDetails(context.Background(), currency.EMPTYCODE, "", "207597618027560960")
	if err != nil {
		t.Error(err)
	}
}

func TestGetSupportedNetworksPerAsset(t *testing.T) {
	t.Parallel()
	_, err := co.GetSupportedNetworksPerAsset(context.Background(), currency.BTC, "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetInstruments(t *testing.T) {
	t.Parallel()
	_, err := co.GetInstruments(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetInstrumentDetails(t *testing.T) {
	t.Parallel()
	_, err := co.GetInstrumentDetails(context.Background(), "BTC-PERP", "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetQuotePerInstrument(t *testing.T) {
	t.Parallel()
	_, err := co.GetQuotePerInstrument(context.Background(), "BTC-PERP", "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestCreateOrder(t *testing.T) {
	t.Parallel()
	orderType, err := orderTypeString(order.Limit)
	if err != nil {
		t.Fatal(err)
	}
	co.Verbose = true
	_, err = co.CreateOrder(context.Background(), &OrderRequestParams{
		Side:       "BUY",
		BaseSize:   1,
		Instrument: "BTC-USDT",
		OrderType:  orderType,
		Price:      12345.67,
		ExpireTime: "",
		PostOnly:   true,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	// sharedtestvalues.SkipTestIfCredentialsUnset(t, )
	_, err := co.GetOpenOrders(context.Background(), "", "", "BTC-PERP", "", "", time.Time{}, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelOrders(t *testing.T) {
	t.Parallel()
	_, err := co.CancelOrders(context.Background(), "1234", "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestModifyOpenOrder(t *testing.T) {
	t.Parallel()
	_, err := co.ModifyOpenOrder(context.Background(), "1234")
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := co.GetOrderDetails(context.Background(), "1234")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	_, err := co.CancelTradeOrder(context.Background(), "1234")
	if err != nil {
		t.Error(err)
	}
}

func TestListAllUserPortfolios(t *testing.T) {
	t.Parallel()
	_, err := co.GetAllUserPortfolios(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetPortfolioDetails(t *testing.T) {
	t.Parallel()
	_, err := co.GetPortfolioDetails(context.Background(), "", "1234")
	if err != nil {
		t.Error(err)
	}
}

func TestGetPortfolioSummary(t *testing.T) {
	t.Parallel()
	co.Verbose = true
	_, err := co.GetPortfolioSummary(context.Background(), "", "5189861793641175")
	if err != nil {
		t.Error(err)
	}
}

func TestListPortfolioBalances(t *testing.T) {
	t.Parallel()
	co.Verbose = true
	_, err := co.ListPortfolioBalances(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetPortfolioAssetBalance(t *testing.T) {
	t.Parallel()
	co.Verbose = true
	_, err := co.GetPortfolioAssetBalance(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", currency.BTC)
	if err != nil {
		t.Error(err)
	}
}

func TestPortfolioPosition(t *testing.T) {
	t.Parallel()
	co.Verbose = true
	_, err := co.ListPortfolioPositions(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetPortfolioInstrumentPosition(t *testing.T) {
	t.Parallel()
	co.Verbose = true
	cp, err := currency.NewPairFromString("BTC-PERP")
	if err != nil {
		t.Fatal(err)
	}
	_, err = co.GetPortfolioInstrumentPosition(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", cp)
	if err != nil {
		t.Error(err)
	}
}

func TestListPortfolioFills(t *testing.T) {
	t.Parallel()
	_, err := co.ListPortfolioFills(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestListMatchingTransfers(t *testing.T) {
	t.Parallel()
	_, err := co.ListMatchingTransfers(context.Background(), "", "", "", "ALL", 10, 0, time.Now().Add(-time.Hour*24*10), time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestGetTransfer(t *testing.T) {
	t.Parallel()
	_, err := co.GetTransfer(context.Background(), "12345")
	if err != nil {
		t.Error(err)
	}
}

func TestWithdrawToCryptoAddress(t *testing.T) {
	t.Parallel()
	_, err := co.WithdrawToCryptoAddress(context.Background(), nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf("expected %v, got %v", common.ErrNilPointer, err)
	}
	_, err = co.WithdrawToCryptoAddress(context.Background(), &WithdrawCryptoParams{
		Portfolio:       "892e8c7c-e979-4cad-b61b-55a197932cf1",
		AssetIdentifier: "291efb0f-2396-4d41-ad03-db3b2311cb2c",
		Amount:          1200,
		Address:         "1234HGJHGHGHGJ",
	})
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf("expected %v, got %v", common.ErrNilPointer, err)
	}
}

func TestCreateCryptoAddress(t *testing.T) {
	t.Parallel()
	_, err := co.CreateCryptoAddress(context.Background(), nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf("expected %v, got %v", common.ErrNilPointer, err)
	}
	_, err = co.CreateCryptoAddress(context.Background(), &CryptoAddressParam{
		Portfolio:       "892e8c7c-e979-4cad-b61b-55a197932cf1",
		AssetIdentifier: "291efb0f-2396-4d41-ad03-db3b2311cb2c",
		NetworkArnID:    "networks/ethereum-mainnet/assets/313ef8a9-ae5a-5f2f-8a56-572c0e2a4d5a",
	})
	if err != nil {
		t.Error(err)
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := co.FetchTradablePairs(context.Background(), asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	err := co.UpdateTradablePairs(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC-PERP")
	if err != nil {
		t.Fatal(err)
	}
	_, err = co.UpdateTicker(context.Background(), pair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestWsConnect(t *testing.T) {
	t.Parallel()
	err := co.WsConnect()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second * 23)
}

func TestGenerateSubscriptionPayload(t *testing.T) {
	t.Parallel()
	payload, err := co.GenerateSubscriptionPayload([]stream.ChannelSubscription{}, "SUBSCRIBE")
	if !errors.Is(err, errEmptyArgument) {
		t.Fatalf("expected %v, got %v", errEmptyArgument, err)
	}
	payload, err = co.GenerateSubscriptionPayload([]stream.ChannelSubscription{
		{Channel: cnlFunding, Currency: currency.Pair{Base: currency.BTC, Delimiter: "-", Quote: currency.USDT}},
		{Channel: cnlFunding, Currency: currency.Pair{Base: currency.BTC, Delimiter: "-", Quote: currency.USDC}},
		{Channel: cnlInstruments, Currency: currency.Pair{Base: currency.BTC, Delimiter: "-", Quote: currency.USDT}},
		{Channel: cnlInstruments, Currency: currency.Pair{Base: currency.BTC, Delimiter: "-", Quote: currency.USDC}},
		{Channel: cnlMatch, Currency: currency.Pair{Base: currency.BTC, Delimiter: "-", Quote: currency.USDT}},
	}, "SUBSCRIBE")
	if err != nil {
		t.Fatal(err)
	} else if len(payload) != 2 {
		t.Fatalf("expected payload of length %d, got %d", 2, len(payload))
	}
}
