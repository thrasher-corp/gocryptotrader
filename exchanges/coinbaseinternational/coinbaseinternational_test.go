package coinbaseinternational

import (
	"context"
	"errors"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	passphrase              = ""
	canManipulateRealOrders = false
)

var co = &CoinbaseInternational{}
var btcPerp = currency.Pair{Base: currency.BTC, Delimiter: currency.DashDelimiter, Quote: currency.PERP}

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
	os.Exit(m.Run())
}

func TestListAssets(t *testing.T) {
	t.Parallel()
	_, err := co.ListAssets(context.Background())
	assert.NoError(t, err)
}

func TestGetAssetDetails(t *testing.T) {
	t.Parallel()
	_, err := co.GetAssetDetails(context.Background(), currency.EMPTYCODE, "", "207597618027560960")
	assert.NoError(t, err)
}

func TestGetSupportedNetworksPerAsset(t *testing.T) {
	t.Parallel()
	_, err := co.GetSupportedNetworksPerAsset(context.Background(), currency.USDC, "", "")
	assert.NoError(t, err)
}

func TestGetInstruments(t *testing.T) {
	t.Parallel()
	_, err := co.GetInstruments(context.Background())
	assert.NoError(t, err)
}

func TestGetInstrumentDetails(t *testing.T) {
	t.Parallel()
	_, err := co.GetInstrumentDetails(context.Background(), "BTC-PERP", "", "")
	assert.NoError(t, err)
}

func TestGetQuotePerInstrument(t *testing.T) {
	t.Parallel()
	_, err := co.GetQuotePerInstrument(context.Background(), "BTC-PERP", "", "")
	assert.NoError(t, err)
}

func TestCreateOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	orderType, err := orderTypeString(order.Limit)
	if err != nil {
		t.Fatal(err)
	}
	_, err = co.CreateOrder(context.Background(), &OrderRequestParams{
		Side:       "BUY",
		BaseSize:   1,
		Instrument: "BTC-PERP",
		OrderType:  orderType,
	})
	if !errors.Is(err, order.ErrPriceBelowMin) {
		t.Fatalf("expected %v, got %v", order.ErrAmountBelowMin, err)
	}
	_, err = co.CreateOrder(context.Background(), &OrderRequestParams{
		Side:       "BUY",
		BaseSize:   1,
		Instrument: "BTC-PERP",
		OrderType:  orderType,
		Price:      12345.67,
	})
	if !errors.Is(err, order.ErrOrderIDNotSet) {
		t.Fatalf("expected %v, got %v", order.ErrOrderIDNotSet, err)
	}
	_, err = co.CreateOrder(context.Background(), &OrderRequestParams{
		Side:       "BUY",
		BaseSize:   1,
		Instrument: "BTC-PERP",
		OrderType:  orderType,
	})
	if !errors.Is(err, order.ErrPriceBelowMin) {
		t.Fatalf("expected %v, got %v", order.ErrPriceBelowMin, err)
	}
	_, err = co.CreateOrder(context.Background(), &OrderRequestParams{
		ClientOrderID: "123442",
		Side:          "BUY",
		BaseSize:      1,
		Instrument:    "BTC-PERP",
		OrderType:     orderType,
		Price:         12345.67,
		ExpireTime:    "",
		PostOnly:      true,
		TimeInForce:   "GTC",
	})
	assert.NoError(t, err)
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	_, err := co.GetOpenOrders(context.Background(), "", "", "BTC-PERP", "", "", time.Time{}, 0, 0)
	assert.NoError(t, err)
}

func TestCancelOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	_, err := co.CancelOrders(context.Background(), "1234", "", "")
	assert.NoError(t, err)
}

func TestModifyOpenOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	_, err := co.ModifyOpenOrder(context.Background(), "1234", &ModifyOrderParam{})
	assert.ErrorIsf(t, err, common.ErrNilPointer, "expected %v, got %v", common.ErrNilPointer, err)
	_, err = co.ModifyOpenOrder(context.Background(), "1234", &ModifyOrderParam{
		Price:     1234,
		StopPrice: 1239,
		Size:      1,
	})
	assert.NoError(t, err)
}

func TestGetOrderDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	_, err := co.GetOrderDetail(context.Background(), "1234")
	assert.NoError(t, err)
}

func TestCancelTradeOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	_, err := co.CancelTradeOrder(context.Background(), "", "", "", "")
	assert.ErrorIsf(t, err, order.ErrOrderIDNotSet, "expected %v, got %v", order.ErrOrderIDNotSet, err)
	_, err = co.CancelTradeOrder(context.Background(), "order-id", "", "", "")
	assert.ErrorIsf(t, err, errMissingPortfolioID, "expected %v, got %v", errMissingPortfolioID, err)
	_, err = co.CancelTradeOrder(context.Background(), "1234", "", "12344232", "")
	assert.NoError(t, err)
}

func TestListAllUserPortfolios(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	_, err := co.GetAllUserPortfolios(context.Background())
	assert.NoError(t, err)
}

func TestGetPortfolioDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	_, err := co.GetPortfolioDetails(context.Background(), "", "1234")
	assert.NoError(t, err)
}

func TestGetPortfolioSummary(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	_, err := co.GetPortfolioSummary(context.Background(), "", "5189861793641175")
	assert.NoError(t, err)
}

func TestListPortfolioBalances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	_, err := co.ListPortfolioBalances(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetPortfolioAssetBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	_, err := co.GetPortfolioAssetBalance(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", currency.BTC)
	assert.NoError(t, err)
}

func TestPortfolioPosition(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	_, err := co.ListPortfolioPositions(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "")
	assert.NoError(t, err)
}

func TestGetPortfolioInstrumentPosition(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	_, err := co.GetPortfolioInstrumentPosition(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", btcPerp)
	assert.NoError(t, err)
}

func TestListPortfolioFills(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	_, err := co.ListPortfolioFills(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestListMatchingTransfers(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	_, err := co.ListMatchingTransfers(context.Background(), "", "", "", "ALL", 10, 0, time.Now().Add(-time.Hour*24*10), time.Now())
	assert.NoError(t, err)
}

func TestGetTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	_, err := co.GetTransfer(context.Background(), "12345")
	assert.NoError(t, err)
}

func TestWithdrawToCryptoAddress(t *testing.T) {
	t.Parallel()
	_, err := co.WithdrawToCryptoAddress(context.Background(), nil)
	assert.ErrorIsf(t, err, common.ErrNilPointer, "expected %v, got %v", common.ErrNilPointer, err)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	_, err = co.WithdrawToCryptoAddress(context.Background(), &WithdrawCryptoParams{
		Portfolio:       "892e8c7c-e979-4cad-b61b-55a197932cf1",
		AssetIdentifier: "291efb0f-2396-4d41-ad03-db3b2311cb2c",
		Amount:          1200,
		Address:         "1234HGJHGHGHGJ",
	})
	assert.ErrorIsf(t, err, common.ErrNilPointer, "expected %v, got %v", common.ErrNilPointer, err)
}

func TestCreateCryptoAddress(t *testing.T) {
	t.Parallel()
	_, err := co.CreateCryptoAddress(context.Background(), nil)
	assert.ErrorIsf(t, err, common.ErrNilPointer, "expected %v, got %v", common.ErrNilPointer, err)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	_, err = co.CreateCryptoAddress(context.Background(), &CryptoAddressParam{
		Portfolio:       "892e8c7c-e979-4cad-b61b-55a197932cf1",
		AssetIdentifier: "291efb0f-2396-4d41-ad03-db3b2311cb2c",
		NetworkArnID:    "networks/ethereum-mainnet/assets/313ef8a9-ae5a-5f2f-8a56-572c0e2a4d5a",
	})
	assert.NoError(t, err)
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
	_, err := co.UpdateTicker(context.Background(), btcPerp, asset.Spot)
	assert.NoError(t, err)
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := co.UpdateTickers(context.Background(), asset.Spot)
	assert.NoError(t, err)
}

func TestWsConnect(t *testing.T) {
	t.Parallel()
	err := co.WsConnect()
	if err != nil {
		t.Fatal(err)
	}
}

func TestGenerateSubscriptionPayload(t *testing.T) {
	t.Parallel()
	_, err := co.GenerateSubscriptionPayload([]subscription.Subscription{}, "SUBSCRIBE")
	if !errors.Is(err, errEmptyArgument) {
		t.Fatalf("expected %v, got %v", errEmptyArgument, err)
	}
	payload, err := co.GenerateSubscriptionPayload([]subscription.Subscription{
		{Channel: cnlFunding, Pair: currency.Pair{Base: currency.BTC, Delimiter: "-", Quote: currency.USDT}},
		{Channel: cnlFunding, Pair: currency.Pair{Base: currency.BTC, Delimiter: "-", Quote: currency.USDC}},
		{Channel: cnlFunding, Pair: currency.Pair{Base: currency.BTC, Delimiter: "-", Quote: currency.USDC}},
		{Channel: cnlInstruments, Pair: currency.Pair{Base: currency.BTC, Delimiter: "-", Quote: currency.USDT}},
		{Channel: cnlInstruments, Pair: currency.Pair{Base: currency.BTC, Delimiter: "-", Quote: currency.USDC}},
		{Channel: cnlMatch, Pair: currency.Pair{Base: currency.BTC, Delimiter: "-", Quote: currency.USDT}},
	}, "SUBSCRIBE")
	if err != nil {
		t.Fatal(err)
	} else if len(payload) != 2 {
		t.Fatalf("expected payload of length %d, got %d", 2, len(payload))
	}
}

func TestFetchOrderBook(t *testing.T) {
	t.Parallel()
	_, err := co.FetchOrderbook(context.Background(), btcPerp, asset.Spot)
	assert.NoError(t, err)
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	_, err := co.UpdateOrderbook(context.Background(), btcPerp, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	_, err := co.UpdateAccountInfo(context.Background(), asset.Futures)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("expected %v, got %v", asset.ErrNotSupported, err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	_, err = co.UpdateAccountInfo(context.Background(), asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFetchAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	_, err := co.FetchAccountInfo(context.Background(), asset.Spot)
	assert.NoError(t, err)
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	_, err := co.GetAccountFundingHistory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	_, err := co.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	_, err := co.GetFeeByType(context.Background(), &exchange.FeeBuilder{
		IsMaker: true,
		Pair:    btcPerp,
		FeeType: exchange.CryptocurrencyTradeFee,
	})
	assert.NoError(t, err)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	_, err = co.GetFeeByType(context.Background(), &exchange.FeeBuilder{
		IsMaker: true,
		Pair:    btcPerp,
		FeeType: exchange.CryptocurrencyWithdrawalFee,
	})
	assert.NoError(t, err)
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	_, err := co.GetAvailableTransferChains(context.Background(), currency.USDC)
	assert.NoError(t, err)
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	_, err := co.SubmitOrder(context.Background(), &order.Submit{
		Exchange:      co.Name,
		Pair:          btcPerp,
		Side:          order.Buy,
		Type:          order.Limit,
		Price:         0.0001,
		Amount:        10,
		ClientID:      "newOrder",
		ClientOrderID: "my-new-order-id",
		AssetType:     asset.Spot,
	})
	assert.NoError(t, err)
}
func TestModifyOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	_, err := co.ModifyOrder(context.Background(), &order.Modify{
		Exchange:  "CoinbaseInternational",
		OrderID:   "1337",
		Price:     10000,
		Amount:    10,
		Side:      order.Sell,
		Pair:      btcPerp,
		AssetType: asset.CoinMarginedFutures,
	})
	assert.NoError(t, err)
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	err := co.CancelOrder(context.Background(), &order.Cancel{
		Exchange:  "CoinbaseInternational",
		AssetType: asset.Spot,
		Pair:      btcPerp,
		OrderID:   "1234",
		AccountID: "Someones SubAccount",
	})
	assert.NoError(t, err)
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	_, err := co.CancelAllOrders(context.Background(),
		&order.Cancel{AssetType: asset.Spot})
	assert.ErrorIs(t, err, errMissingPortfolioID, err)
	_, err = co.CancelAllOrders(context.Background(),
		&order.Cancel{
			Exchange:  "CoinbaseInternational",
			AssetType: asset.Spot,
			AccountID: "Sub-account Samuael",
			Pair:      btcPerp,
		})
	assert.NoError(t, err)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	_, err := co.GetOrderInfo(context.Background(), "12234", btcPerp, asset.Spot)
	assert.NoError(t, err)
}
func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	_, err := co.WithdrawCryptocurrencyFunds(context.Background(), &withdraw.Request{
		Exchange:    co.Name,
		Amount:      10,
		Currency:    currency.LTC,
		PortfolioID: "1234564",
		Crypto: withdraw.CryptoRequest{
			Chain:      currency.LTC.String(),
			Address:    "3CDJNfdWX8m2NwuGUV3nhXHXEeLygMXoAj",
			AddressTag: "",
		}})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	_, err := co.GetActiveOrders(context.Background(), &order.MultiOrderRequest{
		AssetType: asset.Spot,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := co.UpdateOrderExecutionLimits(context.Background(), asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	pairs, err := co.FetchTradablePairs(context.Background(), asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	for y := range pairs {
		lim, err := co.GetOrderExecutionLimits(asset.Spot, pairs[y])
		if err != nil {
			t.Fatalf("%v %s %v", err, pairs[y], asset.Spot)
		}
		assert.NotEmpty(t, lim, "limit cannot be empty")
	}
}
