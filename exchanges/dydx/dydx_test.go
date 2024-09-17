package dydx

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey     = ""
	apiSecret  = ""
	passphrase = ""

	privateKey = ""

	demoEthereumAddress = ""

	canManipulateRealOrders = false
)

var dy = &DYDX{}

func TestMain(m *testing.M) {
	dy = new(DYDX)
	if err := testexch.Setup(dy); err != nil {
		log.Fatal(err)
	}
	if apiKey != "" && apiSecret != "" && passphrase != "" && privateKey != "" {
		dy.API.AuthenticatedSupport = true
		dy.API.AuthenticatedWebsocketSupport = true
		dy.Websocket.SetCanUseAuthenticatedEndpoints(true)
		dy.SetCredentials(apiKey, apiSecret, passphrase, "", "", "")
	}
	setupWS()
	os.Exit(m.Run())
}

const instrumentJSON = `{	"markets": {"LINK-USD": {"market": "LINK-USD","status": "ONLINE","baseAsset": "LINK","quoteAsset": "USD","stepSize": "0.1","tickSize": "0.01","indexPrice": "12","oraclePrice": "101","priceChange24H": "0","nextFundingRate": "0.0000125000","nextFundingAt": "2021-03-01T18:00:00.000Z","minOrderSize": "1","type": "PERPETUAL","initialMarginFraction": "0.10","maintenanceMarginFraction": "0.05","baselinePositionSize": "1000","incrementalPositionSize": "1000","incrementalInitialMarginFraction": "0.2","volume24H": "0","trades24H": "0","openInterest": "0","maxPositionSize": "10000",	  "assetResolution": "10000000","syntheticAssetId": "0x4c494e4b2d37000000000000000000"}}}`

func TestGetInstruments(t *testing.T) {
	t.Parallel()
	var instrumentData InstrumentDatas
	err := json.Unmarshal([]byte(instrumentJSON), &instrumentData)
	assert.NoError(t, err)

	_, err = dy.GetMarkets(context.Background(), "")
	assert.NoError(t, err)
}

func TestGetOrderbooks(t *testing.T) {
	t.Parallel()
	_, err := dy.GetOrderbooks(context.Background(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = dy.GetOrderbooks(context.Background(), "CRV-USD")
	assert.NoError(t, err)
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := dy.GetTrades(context.Background(), "CRV-USD", time.Time{}, 5)
	assert.NoError(t, err)
}

func TestGetFastWithdrawalLiquidity(t *testing.T) {
	t.Parallel()
	_, err := dy.GetFastWithdrawalLiquidity(context.Background(), FastWithdrawalRequestParam{})
	assert.NoError(t, err)
}

func TestGetMarketStats(t *testing.T) {
	t.Parallel()
	_, err := dy.GetMarketStats(context.Background(), "", 7)
	assert.NoError(t, err)
}

func TestGetHistoricalFunding(t *testing.T) {
	t.Parallel()
	_, err := dy.GetHistoricalFunding(context.Background(), "CRV-USD", time.Time{})
	assert.NoError(t, err)
}

func TestGetCandlesForMarket(t *testing.T) {
	t.Parallel()
	_, err := dy.GetCandlesForMarket(context.Background(), "CRV-USD", kline.FiveMin, "", "", 10)
	assert.NoError(t, err)
}

func TestGetGlobalConfigurationVariables(t *testing.T) {
	t.Parallel()
	_, err := dy.GetGlobalConfigurationVariables(context.Background())
	assert.NoError(t, err)
}

func TestCheckIfUserExists(t *testing.T) {
	t.Parallel()
	_, err := dy.CheckIfUserExists(context.Background(), "")
	assert.NoError(t, err)
}

func TestCheckIfUsernameExists(t *testing.T) {
	t.Parallel()
	_, err := dy.CheckIfUsernameExists(context.Background(), "sam")
	assert.NoError(t, err)
}

func TestGetAPIServerTime(t *testing.T) {
	t.Parallel()
	_, err := dy.GetAPIServerTime(context.Background())
	assert.NoError(t, err)
}

func TestGetPublicLeaderboardPNLs(t *testing.T) {
	t.Parallel()
	_, err := dy.GetPublicLeaderboardPNLs(context.Background(), "DAILY", "ABSOLUTE", time.Time{}, 2)
	assert.NoError(t, err)
}

func TestGetPublicRetroactiveMiningReqards(t *testing.T) {
	t.Parallel()
	_, err := dy.GetPublicRetroactiveMiningReqards(context.Background(), "")
	assert.NoError(t, err)
}

func TestVerifyEmailAddress(t *testing.T) {
	t.Parallel()
	_, err := dy.VerifyEmailAddress(context.Background(), "1234")
	assert.NoError(t, err)
}

func TestGetCurrentlyRevealedHedgies(t *testing.T) {
	t.Parallel()
	_, err := dy.GetCurrentlyRevealedHedgies(context.Background(), "", "")
	assert.NoError(t, err)
}

func TestGetHistoricallyRevealedHedgies(t *testing.T) {
	t.Parallel()
	_, err := dy.GetHistoricallyRevealedHedgies(context.Background(), "daily", 1, 10)
	assert.NoError(t, err)
}

func TestGetInsuranceFundBalance(t *testing.T) {
	t.Parallel()
	_, err := dy.GetInsuranceFundBalance(context.Background())
	assert.NoError(t, err)
}

func TestGetPublicProfile(t *testing.T) {
	t.Parallel()
	_, err := dy.GetPublicProfile(context.Background(), "")
	assert.ErrorIs(t, err, errMissingPublicID)

	_, err = dy.GetPublicProfile(context.Background(), "some_public_profile")
	assert.NoError(t, err)
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := dy.FetchTradablePairs(context.Background(), asset.Spot)
	assert.NoError(t, err)
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	pair := currency.NewPair(currency.BTC, currency.USD)
	_, err := dy.GetHistoricCandles(context.Background(), pair, asset.Spot, kline.Interval(time.Minute*4), time.Now().Add(-time.Minute*20), time.Now())
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	_, err = dy.GetHistoricCandles(context.Background(), pair, asset.Spot, kline.FiveMin, time.Now().Add(-time.Hour), time.Now().Add(-time.Minute*10))
	assert.NoError(t, err)
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := dy.GetHistoricTrades(context.Background(), currency.NewPair(currency.BTC, currency.USD), asset.Spot, time.Time{}, time.Now().Add(-time.Minute*2))
	assert.NoError(t, err)
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := dy.GetRecentTrades(context.Background(), currency.NewPair(currency.BTC, currency.USD), asset.Spot)
	assert.NoError(t, err)
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	_, err := dy.UpdateOrderbook(context.Background(), currency.NewPair(currency.BTC, currency.NewCode("USD")), asset.Spot)
	assert.NoError(t, err)
}

func TestFetchOrderbook(t *testing.T) {
	t.Parallel()
	_, err := dy.FetchOrderbook(context.Background(), currency.NewPair(currency.BTC, currency.USD), asset.Spot)
	assert.NoError(t, err)
}

func TestFetchTicker(t *testing.T) {
	t.Parallel()
	_, err := dy.FetchTicker(context.Background(), currency.NewPair(currency.BTC, currency.USD), asset.Spot)
	assert.NoError(t, err)
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := dy.UpdateTickers(context.Background(), asset.Spot)
	assert.NoError(t, err)
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	_, err := dy.UpdateTicker(context.Background(), currency.NewPair(currency.BTC, currency.USD), asset.Spot)
	assert.NoError(t, err)
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	err := dy.UpdateTradablePairs(context.Background(), true)
	assert.NoError(t, err)
}

func TestWsConnect(t *testing.T) {
	t.Parallel()
	err := dy.WsConnect()
	assert.NoError(t, err)
}

func setupWS() {
	if !dy.Websocket.IsEnabled() {
		return
	}
	if !sharedtestvalues.AreAPICredentialsSet(dy) {
		dy.Websocket.SetCanUseAuthenticatedEndpoints(false)
	}
	err := dy.WsConnect()
	if err != nil {
		log.Fatal(err)
	}
}

func TestGenerateDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	_, err := dy.GenerateDefaultSubscriptions()
	assert.NoError(t, err)
}

func TestSubscribe(t *testing.T) {
	t.Parallel()
	err := dy.Subscribe(subscription.List{
		{
			Channel: "v3_orderbook",
			Pairs: []currency.Pair{{
				Base:      currency.LTC,
				Delimiter: currency.DashDelimiter,
				Quote:     currency.USD,
			}},
		},
	})
	assert.NoError(t, err)
}

// func TestRecoverStarkKeyQuoteBalanceAndOpenPosition(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
// 	_, err := dy.RecoverStarkKeyQuoteBalanceAndOpenPosition(context.Background())
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestGetRegistration(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
// 	_, err := dy.GetRegistration(context.Background())
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestRegisterAPIKey(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
// 	_, err := dy.RegisterAPIKey(context.Background())
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestGetAPIKeys(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
// 	_, err := dy.GetAPIKeys(context.Background())
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestDeleteAPIKeys(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
// 	_, err := dy.DeleteAPIKeys(context.Background(), "publicKey")
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

func TestGetPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
	_, err := dy.GetPositions(context.Background(), "", "", "", 0)
	assert.NoError(t, err)
}

func TestGetUsers(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
	_, err := dy.GetUsers(context.Background())
	assert.NoError(t, err)
}

func TestUpdateusers(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
	_, err := dy.Updateusers(context.Background(), &UpdateUserParams{
		IsSharingUsername: true,
		IsSharingAddress:  true,
	})
	assert.NoError(t, err)
}

func TestGetUserActiveLinks(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
	_, err := dy.GetUserActiveLinks(context.Background(), "PRIMARY", "", "")
	assert.NoError(t, err)
}

// func TestSendUserLinkRequest(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
// 	_, err := dy.SendUserLinkRequest(context.Background(), UserLinkParams{Action: "CREATE_SECONDARY_REQUEST", Address: "0xb794f5ea0ba39494ce839613fffba74279579268"})
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

func TestGetUserPendingLinkRequest(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
	_, err := dy.GetUserPendingLinkRequest(context.Background(), "", "", "")
	assert.NoError(t, err)
}

func TestCreateAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy, canManipulateRealOrders)
	_, err := dy.CreateAccount(context.Background(), "starkKey", "ycoordinate")
	assert.NoError(t, err)
}

func TestGetAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
	_, err := dy.GetAccount(context.Background(), demoEthereumAddress)
	assert.NoError(t, err)
}

func TestGetAccountLeaderboardPNLs(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
	_, err := dy.GetAccountLeaderboardPNLs(context.Background(), "WEEKLY", time.Time{})
	assert.NoError(t, err)
}

func TestGetAccountHistoricalLeaderboardPNLs(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
	_, err := dy.GetAccountHistoricalLeaderboardPNLs(context.Background(), "DAILY", 0)
	assert.NoError(t, err)
}

func TestGetAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
	_, err := dy.GetAccounts(context.Background())
	assert.NoError(t, err)
}

func TestGetPosition(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
	_, err := dy.GetPosition(context.Background(), "", "", 0, time.Time{})
	assert.NoError(t, err)
}

func TestGetTransfers(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
	_, err := dy.GetTransfers(context.Background(), "DEPOSIT", 10, time.Time{})
	assert.NoError(t, err)
}

func TestCreateFastWithdrawal(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy, canManipulateRealOrders)
	_, err := dy.CreateFastWithdrawal(context.Background(), &FastWithdrawalParam{
		CreditAsset:  currency.USDC.String(),
		CreditAmount: 497.95,
		DebitAmount:  505.10,
		LPPositionID: 1,
		Expiration:   dYdXTimeUTC(time.Now().Add(time.Hour * 8 * 24)),
		ClientID:     "123456",
		ToAddress:    demoEthereumAddress,
	})
	assert.NoError(t, err)
}

func TestCreateNewOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy, canManipulateRealOrders)
	_, err := dy.CreateNewOrder(context.Background(), &CreateOrderRequestParams{
		Market:       "BTC-USD",
		Side:         order.Buy.String(),
		Type:         order.Limit.String(),
		PostOnly:     true,
		Size:         1,
		Price:        123,
		LimitFee:     0,
		Expiration:   dYdXTimeUTC(time.Now().Add(time.Hour * 24 * 8)),
		TimeInForce:  "GTT",
		Cancelled:    true,
		TriggerPrice: 0,
	})
	assert.NoError(t, err)
}

func TestCancelOrderByID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy, canManipulateRealOrders)
	_, err := dy.CancelOrderByID(context.Background(), "1234")
	assert.NoError(t, err)

	_, err = dy.CancelOrderByID(context.Background(), "")
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)
}

func TestCancelMultipleOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy, canManipulateRealOrders)
	_, err := dy.CancelMultipleOrders(context.Background(), "")
	assert.NoError(t, err)
}

func TestCancelActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy, canManipulateRealOrders)
	enabledPairs, err := dy.GetEnabledPairs(asset.Spot)
	require.NoError(t, err)

	_, err = dy.CancelActiveOrders(context.Background(), enabledPairs[0].String(), "buy", "")
	assert.NoError(t, err)
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
	enabledPairs, err := dy.GetEnabledPairs(asset.Spot)
	require.NoError(t, err)

	_, err = dy.GetOrders(context.Background(), enabledPairs[0].String(), "PENDING", "", "TRAILING_STOP", 90, time.Time{}, true)
	assert.NoError(t, err)

	_, err = dy.GetOpenOrders(context.Background(), enabledPairs[0].String(), "", "")
	assert.NoError(t, err)
}

func TestGetOrderByID(t *testing.T) {
	t.Parallel()
	_, err := dy.GetOrderByID(context.Background(), "1234")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
	_, err = dy.GetOrderByID(context.Background(), "1234")
	assert.NoError(t, err)

}

func TestGetOrderByClientID(t *testing.T) {
	t.Parallel()
	_, err := dy.GetOrderByClientID(context.Background(), "")
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
	_, err = dy.GetOrderByClientID(context.Background(), "1234")
	assert.NoError(t, err)

}

func TestGetFills(t *testing.T) {
	t.Parallel()
	enabledPairs, err := dy.GetEnabledPairs(asset.Spot)
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
	_, err = dy.GetFills(context.Background(), enabledPairs[0].String(), "", 10, time.Now().Add(time.Hour*4))
	assert.NoError(t, err)
}
func TestGetFundingPayment(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
	enabledPairs, err := dy.GetEnabledPairs(asset.Spot)
	require.NoError(t, err)

	_, err = dy.GetFundingPayment(context.Background(), enabledPairs[0].String(), 10, time.Time{})
	assert.NoError(t, err)
}

func TestGetHistoricPNLTicks(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
	_, err := dy.GetHistoricPNLTicks(context.Background(), time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetTradingRewards(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
	_, err := dy.GetTradingRewards(context.Background(), 4, "")
	assert.NoError(t, err)
}

func TestGetLiquidityProviderRewards(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
	_, err := dy.GetLiquidityProviderRewards(context.Background(), 14)
	assert.NoError(t, err)

	_, err = dy.GetLiquidityRewards(context.Background(), 14, "")
	assert.NoError(t, err)
}

func TestGetRetroactiveMiningRewards(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
	_, err := dy.GetRetroactiveMiningRewards(context.Background())
	assert.NoError(t, err)
}

func TestSendVerificationEmail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
	_, err := dy.SendVerificationEmail(context.Background())
	assert.NoError(t, err)
}

func TestRequestTestnetTokens(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
	_, err := dy.RequestTestnetTokens(context.Background())
	assert.NoError(t, err)
}

func TestGetPrivateProfile(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
	_, err := dy.GetPrivateProfile(context.Background())
	assert.NoError(t, err)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
	_, err := dy.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Spot)
	assert.NoError(t, err)
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy, canManipulateRealOrders)
	_, err := dy.SubmitOrder(context.Background(), &order.Submit{
		Exchange: dy.Name,
		Pair: currency.Pair{
			Delimiter: privateKey,
			Base:      currency.LTC,
			Quote:     currency.BTC,
		},
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     0.0001,
		Amount:    10,
		ClientID:  "newOrder",
		AssetType: asset.Spot,
	})
	assert.NoError(t, err)
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy, canManipulateRealOrders)
	err := dy.CancelOrder(context.Background(), &order.Cancel{
		AssetType: asset.Spot,
		OrderID:   "1234",
	})
	assert.NoError(t, err)
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy, canManipulateRealOrders)
	enabledPair, err := dy.GetEnabledPairs(asset.Spot)
	require.NoError(t, err)

	_, err = dy.CancelBatchOrders(context.Background(), []order.Cancel{
		{
			OrderID:   "1",
			Side:      order.Sell,
			Pair:      enabledPair[0],
			AssetType: asset.Spot,
		},
		{
			OrderID:   "2",
			Side:      order.Buy,
			Pair:      enabledPair[1],
			AssetType: asset.PerpetualSwap,
		},
	})
	assert.NoError(t, err)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
	enabled, err := dy.GetEnabledPairs(asset.Spot)
	assert.NoError(t, err)
	require.NotEmpty(t, enabled)

	_, err = dy.GetOrderInfo(context.Background(), "123", enabled[0], asset.Spot)
	assert.NoError(t, err)
}

func TestCreateWithdrawal(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy, canManipulateRealOrders)
	_, err := dy.CreateWithdrawal(context.Background(), privateKey, &WithdrawalParam{
		Asset:      currency.USDC.String(),
		Expiration: dYdXTimeUTC(time.Now().Add(time.Hour * 24 * 10)),
		Amount:     10,
	})
	assert.NoError(t, err)
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy, canManipulateRealOrders)
	_, err := dy.WithdrawCryptocurrencyFunds(context.Background(), &withdraw.Request{
		Exchange: dy.Name,
		Amount:   100,
		Currency: currency.BTC,
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	})
	assert.NoError(t, err)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
	enabledPairs, err := dy.GetEnabledPairs(asset.Spot)
	require.NoError(t, err)

	_, err = dy.GetActiveOrders(context.Background(), &order.MultiOrderRequest{
		Type:      order.Limit,
		Pairs:     enabledPairs[:3],
		AssetType: asset.Spot,
		Side:      order.Buy,
	})
	assert.NoError(t, err)
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
	enabledPairs, err := dy.GetEnabledPairs(asset.Spot)
	require.NoError(t, err)

	var getOrdersRequest = order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.Buy,
	}
	_, err = dy.GetOrderHistory(context.Background(), &getOrdersRequest)
	assert.NoError(t, err)

	getOrdersRequest.Pairs = enabledPairs[:3]
	_, err = dy.GetOrderHistory(context.Background(), &getOrdersRequest)
	assert.NoError(t, err)
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, dy)
	_, err := dy.GetFeeByType(context.Background(), &exchange.FeeBuilder{
		Amount:        10000,
		FeeType:       exchange.CryptocurrencyTradeFee,
		PurchasePrice: 1000000,
		IsMaker:       true,
	})
	assert.NoError(t, err)
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := dy.GetServerTime(context.Background(), asset.Empty)
	assert.NoError(t, err)
}

// func TestGenerateAddress(t *testing.T) {
// 	t.Parallel()
// 	var privateKey string
// 	var privKey *ecdsa.PrivateKey
// 	if privateKey == "" {
// 		var err error
// 		privKey, err = crypto.GenerateKey()
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		privateKey = hexutil.Encode(crypto.FromECDSA(privKey))
// 	}
// 	_, _, err := GeneratePublicKeyAndAddress(privateKey)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// }

var pushDataMap = map[string]string{
	"Orderbook Snapshoot": `{ "type": "subscribed", "connection_id": "e8e52585-a771-4dbd-af27-02ec98091ace", "message_id": 1, "channel": "v3_orderbook", "id": "BTC-USD", "contents": { "asks": [ { "size": "2.6103", "price": "23206" }, { "size": "4.5688", "price": "23207" }, { "size": "9.3683", "price": "23208" }, { "size": "1.7395", "price": "23209" }, { "size": "13.104", "price": "23210" }, { "size": "1.603", "price": "23211" }, { "size": "0.8061", "price": "23212" }, { "size": "10.7343", "price": "23213" }, { "size": "2.3616", "price": "23214" }, { "size": "10.4789", "price": "23215" }, { "size": "0.3159", "price": "23216" }, { "size": "2.6208", "price": "23217" }, { "size": "3.3611", "price": "23218" }, { "size": "3.2764", "price": "23219" }, { "size": "0.548", "price": "23220" }, { "size": "12.5066", "price": "23221" }, { "size": "4.2413", "price": "23222" }, { "size": "0.5673", "price": "23223" }, { "size": "11.0673", "price": "23224" }, { "size": "6.5205", "price": "23225" }, { "size": "1.2896", "price": "23226" }, { "size": "10.6912", "price": "23227" }, { "size": "0.237", "price": "23228" } ], "bids": [ { "size": "2.3172", "price": "23205" }, { "size": "4.4282", "price": "23204" }, { "size": "6.0995", "price": "23203" }, { "size": "3.4384", "price": "23202" }, { "size": "2.5459", "price": "23201" }, { "size": "1.65", "price": "23200" }, { "size": "7.1203", "price": "23199" }, { "size": "9.5934", "price": "23198" }, { "size": "1.0401", "price": "23197" }, { "size": "4.6515", "price": "23196" }, { "size": "1.1346", "price": "23195" }, { "size": "12.8036", "price": "23194" }, { "size": "2.2153", "price": "23193" }, { "size": "0.8208", "price": "23192" }, { "size": "4.3075", "price": "23191" }, { "size": "2.7285", "price": "23190" }, { "size": "2.4483", "price": "23189" }, { "size": "1.0121", "price": "23188" }, { "size": "17.9464", "price": "23187" }, { "size": "0.3", "price": "23186" }, { "size": "1.8373", "price": "23185" }, { "size": "4.366", "price": "23184" }, { "size": "0.237", "price": "23183" }, { "size": "0.9216", "price": "23182" }, { "size": "1.0132", "price": "23181" }, { "size": "9.059", "price": "23180" }, { "size": "6.9619", "price": "23179" }, { "size": "10.5055", "price": "23178" }, { "size": "0.2", "price": "23176" }, { "size": "15.478", "price": "23175" }, { "size": "9.061", "price": "23174" }, { "size": "6.4383", "price": "23173" }, { "size": "12.6089", "price": "23172" }, { "size": "0.2161", "price": "23171" }, { "size": "0.001", "price": "23170" }, { "size": "14.1518", "price": "23169" }, { "size": "1.2936", "price": "23167" }, { "size": "0.248", "price": "23165" }, { "size": "4.0656", "price": "23164" }, { "size": "2.3816", "price": "23162" }, { "size": "7.8154", "price": "23161" }, { "size": "0.6548", "price": "23160" }, { "size": "1.733", "price": "23157" }, { "size": "0.3", "price": "23156" }, { "size": "0.862", "price": "23155" }, { "size": "4.674", "price": "23154" }, { "size": "0.02", "price": "23153" }, { "size": "8.2945", "price": "23151" }, { "size": "40.0016", "price": "23149" }, { "size": "0.3", "price": "23148" }, { "size": "0.4", "price": "23142" }, { "size": "0.0022", "price": "23140" }, { "size": "0.3", "price": "23138" }, { "size": "0.0194", "price": "23131" }, { "size": "0.001", "price": "23130" }, { "size": "35.5889", "price": "23129" }, { "size": "8.804", "price": "23128" }, { "size": "14.041", "price": "23122" }, { "size": "0.055", "price": "23120" }, { "size": "0.1598", "price": "23115" }, { "size": "0.01", "price": "23110" }, { "size": "0.001", "price": "23106" }, { "size": "0.8616", "price": "23104" }, { "size": "47.9684", "price": "23101" }, { "size": "13.135", "price": "23094" }, { "size": "0.47", "price": "23091" }, { "size": "0.001", "price": "23090" }, { "size": "0.1", "price": "23084" }, { "size": "0.0022", "price": "23082" }, { "size": "2.334", "price": "23080" }, { "size": "0.0021", "price": "23079" }, { "size": "0.009", "price": "23069" }, { "size": "0.0011", "price": "23060" }, { "size": "0.001", "price": "23059" }, { "size": "13.2516", "price": "23055" }, { "size": "0.051", "price": "23050" }, { "size": "28.23", "price": "23048" }, { "size": "0.001", "price": "23042" }, { "size": "0.005", "price": "23040" }, { "size": "0.1", "price": "23035" }, { "size": "0.01", "price": "23030" }, { "size": "0.001", "price": "23028" }, { "size": "9.7571", "price": "23025" }, { "size": "0.009", "price": "23023" }, { "size": "0.001", "price": "23010" }, { "size": "31.031", "price": "23005" }, { "size": "1.2926", "price": "23004" }, { "size": "0.8", "price": "23000" }, { "size": "0.0638", "price": "22996" }, { "size": "0.3", "price": "22986" }, { "size": "0.025", "price": "22981" }, { "size": "0.01", "price": "22980" }, { "size": "0.009", "price": "22978" }, { "size": "39.565", "price": "22971" }, { "size": "0.001", "price": "22970" }, { "size": "0.446", "price": "22964" }, { "size": "0.8714", "price": "22949" }, { "size": "0.009", "price": "22933" }, { "size": "0.001", "price": "22930" }, { "size": "0.0418", "price": "22925" }, { "size": "0.025", "price": "22924" }, { "size": "0.01", "price": "22920" }, { "size": "0.001", "price": "22908" }, { "size": "0.106", "price": "22900" }, { "size": "0.001", "price": "22890" }, { "size": "0.009", "price": "22888" }, { "size": "0.1", "price": "22885" }, { "size": "4.2985", "price": "22884" }, { "size": "0.0558", "price": "22880" }, { "size": "0.001", "price": "22855" }, { "size": "1", "price": "22852" }, { "size": "0.001", "price": "22850" }, { "size": "0.009", "price": "22842" }, { "size": "2.2244", "price": "22834" }, { "size": "2.6292", "price": "22820" }, { "size": "0.025", "price": "22813" }, { "size": "0.001", "price": "22811" }, { "size": "0.3", "price": "22800" } ] } }`,
	"Orderbook Update":    `{"type": "channel_data","connection_id": "e8e52585-a771-4dbd-af27-02ec98091ace","message_id": 662,"id": "BTC-USD","channel": "v3_orderbook","contents": {"offset": "14449118423","bids": [],"asks": [["23205","1.1386"]]}}`,
	"Trade":               `{"type": "subscribed", "connection_id": "b4e8043e-f149-4019-ba1a-246831296196", "message_id": 1, "channel": "v3_trades", "id": "BTC-USD", "contents": { "trades": [ { "side": "BUY", "size": "0.0863", "price": "23192", "createdAt": "2023-02-26T17:48:14.273Z", "liquidation": false }, { "side": "BUY", "size": "0.0017", "price": "23190", "createdAt": "2023-02-26T17:48:14.129Z", "liquidation": false }, { "side": "BUY", "size": "0.16", "price": "23188", "createdAt": "2023-02-26T17:48:13.995Z", "liquidation": false }, { "side": "BUY", "size": "0.0017", "price": "23188", "createdAt": "2023-02-26T17:48:13.995Z", "liquidation": false }, { "side": "BUY", "size": "0.0225", "price": "23188", "createdAt": "2023-02-26T17:48:13.995Z", "liquidation": false }, { "side": "BUY", "size": "0.4081", "price": "23188", "createdAt": "2023-02-26T17:48:13.959Z", "liquidation": false } ] } }`,
	"Ticker":              `{"type": "subscribed", "connection_id": "06101634-2ffb-4f7d-ae6b-e0235723c264", "message_id": 11, "channel": "v3_markets", "contents": { "markets": { "CELO-USD": { "market": "CELO-USD", "status": "ONLINE", "baseAsset": "CELO", "quoteAsset": "USD", "stepSize": "1", "tickSize": "0.001", "indexPrice": "0.7790", "oraclePrice": "0.7791", "priceChange24H": "-0.006843", "nextFundingRate": "0.0000103132", "nextFundingAt": "2023-02-26T19:00:00.000Z", "minOrderSize": "10", "type": "PERPETUAL", "initialMarginFraction": "0.2", "maintenanceMarginFraction": "0.05", "transferMarginFraction": "0.006488", "volume24H": "889920.224000", "trades24H": "807", "openInterest": "338608", "incrementalInitialMarginFraction": "0.02", "incrementalPositionSize": "17700", "maxPositionSize": "355000", "baselinePositionSize": "35500", "assetResolution": "1000000", "syntheticAssetId": "0x43454c4f2d36000000000000000000" }, "LINK-USD": { "market": "LINK-USD", "status": "ONLINE", "baseAsset": "LINK", "quoteAsset": "USD", "stepSize": "0.1", "tickSize": "0.001", "indexPrice": "7.3733", "oraclePrice": "7.3563", "priceChange24H": "0.064351", "nextFundingRate": "0.0000014420", "nextFundingAt": "2023-02-26T19:00:00.000Z", "minOrderSize": "1", "type": "PERPETUAL", "initialMarginFraction": "0.10", "maintenanceMarginFraction": "0.05", "transferMarginFraction": "0.005675", "volume24H": "6812983.182700", "trades24H": "9291", "openInterest": "530820.8", "incrementalInitialMarginFraction": "0.02", "incrementalPositionSize": "14000", "maxPositionSize": "700000", "baselinePositionSize": "70000", "assetResolution": "10000000", "syntheticAssetId": "0x4c494e4b2d37000000000000000000" }, "DOGE-USD": { "market": "DOGE-USD", "status": "ONLINE", "baseAsset": "DOGE", "quoteAsset": "USD", "stepSize": "10", "tickSize": "0.0001", "indexPrice": "0.0815", "oraclePrice": "0.0813", "priceChange24H": "0.000218", "nextFundingRate": "0.0000125000", "nextFundingAt": "2023-02-26T19:00:00.000Z", "minOrderSize": "100", "type": "PERPETUAL", "initialMarginFraction": "0.10", "maintenanceMarginFraction": "0.05", "transferMarginFraction": "0.002480", "volume24H": "4757624.405000", "trades24H": "2438", "openInterest": "44008710", "incrementalInitialMarginFraction": "0.02", "incrementalPositionSize": "1400000", "maxPositionSize": "70000000", "baselinePositionSize": "7000000", "assetResolution": "100000", "syntheticAssetId": "0x444f47452d35000000000000000000" } } } }`,
}

func TestOrderbookPushData(t *testing.T) {
	t.Parallel()
	for x := range pushDataMap {
		err := dy.wsHandleData([]byte(pushDataMap[x]))
		assert.NoError(t, err)
	}
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := dy.GetLatestFundingRates(context.Background(), &fundingrate.LatestRateRequest{
		Asset:                asset.USDTMarginedFutures,
		Pair:                 currency.NewPair(currency.BTC, currency.USD),
		IncludePredictedRate: true,
	})
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	enabled, err := dy.GetEnabledPairs(asset.Spot)
	assert.NoError(t, err)

	_, err = dy.GetLatestFundingRates(context.Background(), &fundingrate.LatestRateRequest{
		Asset: asset.Spot,
		Pair:  enabled[0],
	})
	assert.NoError(t, err)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := dy.UpdateOrderExecutionLimits(context.Background(), asset.USDCMarginedFutures)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	err = dy.UpdateOrderExecutionLimits(context.Background(), asset.Spot)
	assert.NoError(t, err)

	enabledPairs, err := dy.GetEnabledPairs(asset.Spot)
	require.NoError(t, err)

	for x := range enabledPairs {
		limits, err := dy.GetOrderExecutionLimits(asset.Spot, enabledPairs[x])
		require.NoError(t, err)
		assert.NotEmpty(t, limits)
	}
}
