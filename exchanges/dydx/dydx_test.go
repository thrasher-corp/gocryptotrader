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
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	passphrase              = ""
	etheriumAddress         = ""
	starkPrivateKey         = ""
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

	if apiKey != "" && apiSecret != "" && passphrase != "" {
		exchCfg.API.AuthenticatedSupport = true
		exchCfg.API.AuthenticatedWebsocketSupport = true
	}

	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret
	exchCfg.API.Credentials.ClientID = etheriumAddress
	exchCfg.API.Credentials.PEMKey = passphrase
	exchCfg.API.Credentials.Subaccount = starkPrivateKey

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

var instrumentJSON = `{	"markets": {"LINK-USD": {"market": "LINK-USD","status": "ONLINE","baseAsset": "LINK","quoteAsset": "USD","stepSize": "0.1","tickSize": "0.01","indexPrice": "12","oraclePrice": "101","priceChange24H": "0","nextFundingRate": "0.0000125000","nextFundingAt": "2021-03-01T18:00:00.000Z","minOrderSize": "1","type": "PERPETUAL","initialMarginFraction": "0.10","maintenanceMarginFraction": "0.05","baselinePositionSize": "1000","incrementalPositionSize": "1000","incrementalInitialMarginFraction": "0.2","volume24H": "0","trades24H": "0","openInterest": "0","maxPositionSize": "10000",	  "assetResolution": "10000000","syntheticAssetId": "0x4c494e4b2d37000000000000000000"}}}`

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
	if _, err := dy.GetFastWithdrawalLiquidity(context.Background(), FastWithdrawalParam{}); err != nil {
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
	if _, err := dy.GenerateDefaultSubscriptions(); err != nil {
		t.Error(err)
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

func TestOnboarding(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	err := dy.Onboarding(context.Background(), "", "", "", "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetPositions(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	_, err := dy.GetPositions(context.Background(), "", "", "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUsers(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	_, err := dy.GetUsers(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateusers(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := dy.Updateusers(context.Background(), UpdateUserParams{
		IsSharingUsername: true,
		IsSharingAddress:  true,
	}); err != nil {
		t.Error(err)
	}
}

func TestGetUserActiveLinks(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	_, err := dy.GetUserActiveLinks(context.Background(), "PRIMARY", etheriumAddress, "")
	if err != nil {
		t.Error(err)
	}
}

func TestSendUserLinkRequest(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	_, err := dy.SendUserLinkRequest(context.Background(), UserLinkParams{Action: "CREATE_SECONDARY_REQUEST", Address: "0xb794f5ea0ba39494ce839613fffba74279579268"})
	if err != nil && !strings.Contains(err.Error(), "No receiving user found with address") {
		t.Error(err)
	}
}

func TestGetUserPendingLinkRequest(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	_, err := dy.GetUserPendingLinkRequest(context.Background(), "", "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestCreateAccount(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	_, err := dy.CreateAccount(context.Background(), "starkKey", "ycoordinate")
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccount(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := dy.GetAccount(context.Background(), etheriumAddress); err != nil {
		t.Error(err)
	}
}

func TestGetAccountLeaderboardPNLs(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := dy.GetAccountLeaderboardPNLs(context.Background(), "WEEKLY", time.Time{}); err != nil {
		t.Error(err)
	}
}

func TestGetAccountHistoricalLeaderboardPNLs(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	_, err := dy.GetAccountHistoricalLeaderboardPNLs(context.Background(), "DAILY", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccounts(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	_, err := dy.GetAccounts(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetPosition(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := dy.GetPosition(context.Background(), "", "", 0, time.Time{}); err != nil {
		t.Error(err)
	}
}

func TestTransferResponse(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := dy.GetTransfers(context.Background(), "DEPOSIT", 10, time.Time{}); err != nil {
		t.Error(err)
	}
}

func TestCreateTransfers(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := dy.CreateTransfer(context.Background(), TransferParam{
		Amount:             123,
		ClientID:           "141324",
		Expiration:         time.Now().Add(time.Hour * 24 * 4).UTC().Format(timeFormat),
		ReceiverAccountID:  "ec84385a-ad03-55a8-86bf-a8213571f0ee",
		Signature:          "",
		ReceiverPublicKey:  "037f9c7a8511ea61adf3074f3b60d3911f37bb95cd31cbc712629d992d13e109",
		ReceiverPositionID: "",
	}); err != nil {
		t.Error(err)
	}
}

func TestCreateFastWithdrawal(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	input := WithdrawalParam{
		CreditAsset:  currency.USDC.String(),
		CreditAmount: 123,
		DebitAmount:  100,
		LpPositionId: 1,
		Expiration:   time.Time{}.UTC().Format(timeFormat),
		ClientID:     "",
	}
	if _, err := dy.CreateFastWithdrawal(context.Background(), input); err != nil {
		t.Error(err)
	}
}

func TestCreateNewOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := dy.CreateNewOrder(context.Background(), CreateOrderRequestParams{
		Market:       "BTC-USD",
		Side:         order.Buy.String(),
		Type:         order.Limit.String(),
		PostOnly:     true,
		Size:         1,
		Price:        123,
		LimitFee:     0,
		Expiration:   time.Now().Add(time.Hour * 24 * 3).UTC().Format("2006-01-02T15:04:05.999Z"),
		TimeInForce:  "GTT",
		Cancelled:    true,
		TriggerPrice: 0,
	}); err != nil {
		t.Error(err)
	}
}

func TestCancelOrderByID(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := dy.CancelOrderByID(context.Background(), "1234"); err != nil && !strings.Contains(err.Error(), "No order exists with id: 1234") {
		t.Error(err)
	}
}

func TestCancelMultipleOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := dy.CancelMultipleOrders(context.Background(), ""); err != nil {
		t.Error(err)
	}
}

func TestCancelActiveOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	enabledPairs, err := dy.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dy.CancelActiveOrders(context.Background(), enabledPairs[0].String(), "buy", ""); err != nil {
		t.Error(err)
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	dy.Verbose = true
	enabledPairs, err := dy.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dy.GetOrders(context.Background(), enabledPairs[0].String(), "PENDING", "", "TRAILING_STOP", 90, time.Time{}, true); err != nil {
		t.Error(err)
	}
	if _, err := dy.GetOpenOrders(context.Background(), enabledPairs[0].String(), "", ""); err != nil {
		t.Error(err)
	}
}

func TestGetOrderByID(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := dy.GetOrderByID(context.Background(), "1234"); err != nil && !strings.Contains(err.Error(), "No order found with id: 1234") {
		t.Error(err)
	}
}

func TestGetOrderByClientID(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := dy.GetOrderByClientID(context.Background(), "1234"); err != nil && !strings.Contains(err.Error(), "No order found with clientId: 1234") {
		t.Error(err)
	}
}

func TestGetFills(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	enabledPairs, err := dy.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dy.GetFills(context.Background(), enabledPairs[0].String(), "", 10, time.Now().Add(time.Hour*4)); err != nil {
		t.Error(err)
	}
}
func TestGetFundingPayment(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	enabledPairs, err := dy.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dy.GetFundingPayment(context.Background(), enabledPairs[0].String(), 10, time.Time{}); err != nil {
		t.Error(err)
	}
}

func TestGetHistoricPNLTicks(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := dy.GetHistoricPNLTicks(context.Background(), time.Time{}, time.Time{}); err != nil {
		t.Error(err)
	}
}

func TestGetTradingRewards(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := dy.GetTradingRewards(context.Background(), 4, ""); err != nil {
		t.Error(err)
	}
}

func TestGetLiquidityProviderRewards(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := dy.GetLiquidityProviderRewards(context.Background(), 14); err != nil && !strings.Contains(err.Error(), "User is not a liquidity provider") {
		t.Error(err)
	}
	if _, err := dy.GetLiquidityRewards(context.Background(), 14, ""); err != nil {
		t.Error(err)
	}
}

func TestGetRetroactiveMiningRewards(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := dy.GetRetroactiveMiningRewards(context.Background()); err != nil {
		t.Error(err)
	}
}

func TestSendVerificationEmail(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := dy.SendVerificationEmail(context.Background()); err != nil {
		t.Error(err)
	}
}

func TestRequestTestnetTokens(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := dy.RequestTestnetTokens(context.Background()); err != nil {
		t.Error(err)
	}
}

func TestGetPrivateProfile(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := dy.GetPrivateProfile(context.Background()); err != nil {
		t.Error(err)
	}
}
