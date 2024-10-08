package coinbaseinternational

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	result, err := co.ListAssets(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAssetDetails(t *testing.T) {
	t.Parallel()
	result, err := co.GetAssetDetails(context.Background(), currency.EMPTYCODE, "", "207597618027560960")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSupportedNetworksPerAsset(t *testing.T) {
	t.Parallel()
	result, err := co.GetSupportedNetworksPerAsset(context.Background(), currency.USDC, "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInstruments(t *testing.T) {
	t.Parallel()
	result, err := co.GetInstruments(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInstrumentDetails(t *testing.T) {
	t.Parallel()
	result, err := co.GetInstrumentDetails(context.Background(), "BTC-PERP", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetQuotePerInstrument(t *testing.T) {
	t.Parallel()
	result, err := co.GetQuotePerInstrument(context.Background(), "BTC-PERP", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateOrder(t *testing.T) {
	t.Parallel()
	orderType, err := orderTypeString(order.Limit)
	require.NoError(t, err)
	_, err = co.CreateOrder(context.Background(), &OrderRequestParams{
		Side:       "BUY",
		BaseSize:   1,
		Instrument: "BTC-PERP",
		OrderType:  orderType,
	})
	require.ErrorIs(t, err, order.ErrPriceBelowMin)
	_, err = co.CreateOrder(context.Background(), &OrderRequestParams{
		Side:       "BUY",
		BaseSize:   1,
		Instrument: "BTC-PERP",
		OrderType:  orderType,
		Price:      12345.67,
	})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = co.CreateOrder(context.Background(), &OrderRequestParams{
		Side:       "BUY",
		BaseSize:   1,
		Instrument: "BTC-PERP",
		OrderType:  orderType,
	})
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.CreateOrder(context.Background(), &OrderRequestParams{
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
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetOpenOrders(context.Background(), "", "", "BTC-PERP", "", "", time.Time{}, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.CancelOrders(context.Background(), "1234", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestModifyOpenOrder(t *testing.T) {
	t.Parallel()
	_, err := co.ModifyOpenOrder(context.Background(), "1234", &ModifyOrderParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.ModifyOpenOrder(context.Background(), "1234", &ModifyOrderParam{
		Price:     1234,
		StopPrice: 1239,
		Size:      1,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetOrderDetail(context.Background(), "1234")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelTradeOrder(t *testing.T) {
	t.Parallel()
	_, err := co.CancelTradeOrder(context.Background(), "", "", "", "")
	require.ErrorIsf(t, err, order.ErrOrderIDNotSet, "expected %v, got %v", order.ErrOrderIDNotSet, err)
	_, err = co.CancelTradeOrder(context.Background(), "order-id", "", "", "")
	require.ErrorIsf(t, err, errMissingPortfolioID, "expected %v, got %v", errMissingPortfolioID, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.CancelTradeOrder(context.Background(), "1234", "", "12344232", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestListAllUserPortfolios(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetAllUserPortfolios(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetPortfolioDetails(context.Background(), "", "1234")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioSummary(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetPortfolioSummary(context.Background(), "", "5189861793641175")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestListPortfolioBalances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.ListPortfolioBalances(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioAssetBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetPortfolioAssetBalance(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPortfolioPosition(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.ListPortfolioPositions(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioInstrumentPosition(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetPortfolioInstrumentPosition(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", btcPerp)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestListPortfolioFills(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.ListPortfolioFills(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestListMatchingTransfers(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.ListMatchingTransfers(context.Background(), "", "", "", "ALL", 10, 0, time.Now().Add(-time.Hour*24*10), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetTransfer(context.Background(), "12345")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawToCryptoAddress(t *testing.T) {
	t.Parallel()
	_, err := co.WithdrawToCryptoAddress(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.WithdrawToCryptoAddress(context.Background(), &WithdrawCryptoParams{
		Portfolio:       "892e8c7c-e979-4cad-b61b-55a197932cf1",
		AssetIdentifier: "291efb0f-2396-4d41-ad03-db3b2311cb2c",
		Amount:          1200,
		Address:         "1234HGJHGHGHGJ",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateCryptoAddress(t *testing.T) {
	t.Parallel()
	_, err := co.CreateCryptoAddress(context.Background(), nil)
	assert.ErrorIsf(t, err, common.ErrNilPointer, "expected %v, got %v", common.ErrNilPointer, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.CreateCryptoAddress(context.Background(), &CryptoAddressParam{
		Portfolio:       "892e8c7c-e979-4cad-b61b-55a197932cf1",
		AssetIdentifier: "291efb0f-2396-4d41-ad03-db3b2311cb2c",
		NetworkArnID:    "networks/ethereum-mainnet/assets/313ef8a9-ae5a-5f2f-8a56-572c0e2a4d5a",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	result, err := co.FetchTradablePairs(context.Background(), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	err := co.UpdateTradablePairs(context.Background(), true)
	assert.NoError(t, err)
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	result, err := co.UpdateTicker(context.Background(), btcPerp, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := co.UpdateTickers(context.Background(), asset.Spot)
	assert.NoError(t, err)
}

func TestWsConnect(t *testing.T) {
	t.Parallel()
	err := co.WsConnect()
	assert.NoError(t, err)
}

func TestGenerateSubscriptionPayload(t *testing.T) {
	t.Parallel()
	_, err := co.GenerateSubscriptionPayload(subscription.List{}, "SUBSCRIBE")
	require.ErrorIs(t, err, errEmptyArgument)

	payload, err := co.GenerateSubscriptionPayload(subscription.List{
		{Channel: cnlFunding, Pairs: currency.Pairs{{Base: currency.BTC, Delimiter: "-", Quote: currency.USDT}}},
		{Channel: cnlFunding, Pairs: currency.Pairs{{Base: currency.BTC, Delimiter: "-", Quote: currency.USDC}}},
		{Channel: cnlFunding, Pairs: currency.Pairs{{Base: currency.BTC, Delimiter: "-", Quote: currency.USDC}}},
		{Channel: cnlInstruments, Pairs: currency.Pairs{{Base: currency.BTC, Delimiter: "-", Quote: currency.USDT}}},
		{Channel: cnlInstruments, Pairs: currency.Pairs{{Base: currency.BTC, Delimiter: "-", Quote: currency.USDC}}},
		{Channel: cnlMatch, Pairs: currency.Pairs{{Base: currency.BTC, Delimiter: "-", Quote: currency.USDT}}},
	}, "SUBSCRIBE")
	require.NoError(t, err)
	assert.Len(t, payload, 2)
}

func TestFetchOrderBook(t *testing.T) {
	t.Parallel()
	result, err := co.FetchOrderbook(context.Background(), btcPerp, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	result, err := co.UpdateOrderbook(context.Background(), btcPerp, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	_, err := co.UpdateAccountInfo(context.Background(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.UpdateAccountInfo(context.Background(), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFetchAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.FetchAccountInfo(context.Background(), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetAccountFundingHistory(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetFeeByType(context.Background(), &exchange.FeeBuilder{
		IsMaker: true,
		Pair:    btcPerp,
		FeeType: exchange.CryptocurrencyTradeFee,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = co.GetFeeByType(context.Background(), &exchange.FeeBuilder{
		IsMaker: true,
		Pair:    btcPerp,
		FeeType: exchange.CryptocurrencyWithdrawalFee,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	result, err := co.GetAvailableTransferChains(context.Background(), currency.USDC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.SubmitOrder(context.Background(), &order.Submit{
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
	require.NoError(t, err)
	assert.NotNil(t, result)
}
func TestModifyOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.ModifyOrder(context.Background(), &order.Modify{
		Exchange:  "CoinbaseInternational",
		OrderID:   "1337",
		Price:     10000,
		Amount:    10,
		Side:      order.Sell,
		Pair:      btcPerp,
		AssetType: asset.CoinMarginedFutures,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
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
	_, err := co.CancelAllOrders(context.Background(),
		&order.Cancel{AssetType: asset.Spot})
	assert.ErrorIs(t, err, errMissingPortfolioID, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.CancelAllOrders(context.Background(),
		&order.Cancel{
			Exchange:  "CoinbaseInternational",
			AssetType: asset.Spot,
			AccountID: "Sub-account Samuael",
			Pair:      btcPerp,
		})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetOrderInfo(context.Background(), "12234", btcPerp, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}
func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.WithdrawCryptocurrencyFunds(context.Background(), &withdraw.Request{
		Exchange:    co.Name,
		Amount:      10,
		Currency:    currency.LTC,
		PortfolioID: "1234564",
		Crypto: withdraw.CryptoRequest{
			Chain:      currency.LTC.String(),
			Address:    "3CDJNfdWX8m2NwuGUV3nhXHXEeLygMXoAj",
			AddressTag: "",
		}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetActiveOrders(context.Background(), &order.MultiOrderRequest{
		AssetType: asset.Spot,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := co.UpdateOrderExecutionLimits(context.Background(), asset.Spot)
	require.NoError(t, err)

	pairs, err := co.FetchTradablePairs(context.Background(), asset.Spot)
	require.NoError(t, err)
	for y := range pairs {
		lim, err := co.GetOrderExecutionLimits(asset.Spot, pairs[y])
		require.NoErrorf(t, err, "%v %s %v", err, pairs[y], asset.Spot)
		require.NotEmpty(t, lim, "limit cannot be empty")
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	_, err := co.GetCurrencyTradeURL(context.Background(), asset.Spot, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = co.GetCurrencyTradeURL(context.Background(), asset.Futures, currency.NewPair(currency.BTC, currency.USDC))
	require.ErrorIs(t, err, asset.ErrNotSupported)

	pairs, err := co.CurrencyPairs.GetPairs(asset.Spot, false)
	require.NoError(t, err)
	require.NotEmpty(t, pairs)

	resp, err := co.GetCurrencyTradeURL(context.Background(), asset.Spot, currency.NewPair(currency.BTC, currency.USDC))
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetFeeRateTiers(t *testing.T) {
	t.Parallel()
	result, err := co.GetFeeRateTiers(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}
