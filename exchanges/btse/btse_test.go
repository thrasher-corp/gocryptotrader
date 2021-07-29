package btse

import (
	"errors"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

// Please supply your own keys here to do better tests
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
	testSPOTPair            = "BTC-USD"
	testFUTURESPair         = "BTCPFC"
)

var b BTSE

func TestMain(m *testing.M) {
	b.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}
	btseConfig, err := cfg.GetExchangeConfig("BTSE")
	if err != nil {
		log.Fatal(err)
	}

	btseConfig.API.AuthenticatedSupport = true
	btseConfig.API.Credentials.Key = apiKey
	btseConfig.API.Credentials.Secret = apiSecret
	b.Websocket = sharedtestvalues.NewTestWebsocket()
	err = b.Setup(btseConfig)
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func areTestAPIKeysSet() bool {
	return b.ValidateAPICredentials()
}

func TestFetchFundingHistory(t *testing.T) {
	_, err := b.FetchFundingHistory("")
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarketsSummary(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarketSummary("", true)
	if err != nil {
		t.Error(err)
	}

	ret, err := b.GetMarketSummary(testSPOTPair, true)
	if err != nil {
		t.Error(err)
	}
	if len(ret) != 1 {
		t.Errorf("expected only one result when requesting BTC-USD data received: %v", len(ret))
	}
}

func TestFetchOrderBook(t *testing.T) {
	t.Parallel()
	_, err := b.FetchOrderBook(testSPOTPair, 0, 1, 1, true)
	if err != nil {
		t.Error(err)
	}

	_, err = b.FetchOrderBook(testFUTURESPair, 0, 1, 1, false)
	if err != nil {
		t.Error(err)
	}

	_, err = b.FetchOrderBook(testSPOTPair, 1, 1, 1, true)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()

	p, err := currency.NewPairFromString(testSPOTPair)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.UpdateOrderbook(p, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	f, err := currency.NewPairFromString(testFUTURESPair)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.UpdateOrderbook(f, asset.Futures)
	if err != nil {
		if !errors.Is(err, common.ErrNotYetImplemented) {
			t.Fatal(err)
		}
	}
}

func TestFetchOrderBookL2(t *testing.T) {
	t.Parallel()
	_, err := b.FetchOrderBookL2(testSPOTPair, 20)
	if err != nil {
		t.Error(err)
	}
}

func TestOHLCV(t *testing.T) {
	t.Parallel()
	_, err := b.OHLCV(testSPOTPair,
		time.Now().AddDate(0, 0, -1),
		time.Now(), 60)
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.OHLCV(testSPOTPair, time.Now(), time.Now().AddDate(0, 0, -1), 60)
	if err == nil {
		t.Fatal("expected error if start is after end date")
	}
}

func TestGetPrice(t *testing.T) {
	t.Parallel()
	_, err := b.GetPrice(testSPOTPair)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFormatExchangeKlineInterval(t *testing.T) {
	ret := b.FormatExchangeKlineInterval(kline.OneDay)
	if ret != "1440" {
		t.Fatalf("unexpected result received: %v", ret)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	curr, err := currency.NewPairFromString(testSPOTPair)
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetHistoricCandles(
		curr, asset.Spot,
		time.Time{}, time.Time{},
		kline.OneMin)
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetHistoricCandles(
		curr, asset.Spot,
		time.Time{}, time.Time{},
		kline.OneDay)
	if err != nil {
		t.Fatal(err)
	}

	curr.Quote = currency.XRP
	_, err = b.GetHistoricCandles(
		curr, asset.Spot,
		time.Time{}, time.Time{},
		kline.OneMin)
	if err == nil {
		t.Fatal("expected error when requesting with disabled pair")
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	curr, err := currency.NewPairFromString(testSPOTPair)
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetHistoricCandlesExtended(
		curr, asset.Spot,
		time.Time{}, time.Time{},
		kline.OneMin)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetTrades(testSPOTPair,
		time.Now().AddDate(0, 0, -1), time.Now(),
		0, 0, 50, false, true)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetTrades(testSPOTPair,
		time.Now(), time.Now().AddDate(0, -1, 0),
		0, 0, 50, false, true)
	if err == nil {
		t.Error("expected error if start time is after end time")
	}

	_, err = b.GetTrades(testFUTURESPair,
		time.Now().AddDate(0, 0, -1), time.Now(),
		0, 0, 50, false, false)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	curr, err := currency.NewPairFromString(testSPOTPair)
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.UpdateTicker(curr, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	curr, err = currency.NewPairFromString("BTC-PFC")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.UpdateTicker(curr, asset.Futures)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := b.GetServerTime()
	if err != nil {
		t.Error(err)
	}
}

func TestGetWalletInformation(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping test")
	}
	_, err := b.GetWalletInformation()
	if err != nil {
		t.Error(err)
	}
}

func TestGetFeeInformation(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping test")
	}
	_, err := b.GetFeeInformation("")
	if err != nil {
		t.Error(err)
	}
}

func TestGetWalletHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping test")
	}
	_, err := b.GetWalletHistory(testSPOTPair,
		time.Time{}, time.Time{},
		50)
	if err != nil {
		t.Error(err)
	}
}

func TestGetWalletAddress(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping test")
	}
	_, err := b.GetWalletAddress("XRP")
	if err != nil {
		t.Error(err)
	}
}

func TestCreateWalletAddress(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping test")
	}
	_, err := b.CreateWalletAddress("XRP")
	if err != nil {
		t.Error(err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping test")
	}
	_, err := b.GetDepositAddress(currency.XRP, "")
	if err != nil {
		t.Error(err)
	}
}

func TestCreateOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys are unset or canManipulateRealOrders is false")
	}
	_, err := b.CreateOrder("", 0.0,
		false,
		-1, "BUY", 100, 0, 0,
		testSPOTPair, "GTC",
		0.0, 0.0,
		"LIMIT", "LIMIT")
	if err != nil {
		t.Error(err)
	}
}

func TestBTSEIndexOrderPeg(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys are unset or canManipulateRealOrders is false")
	}
	_, err := b.IndexOrderPeg("", 0.0,
		false,
		-1, "BUY", 100, 0, 0,
		testSPOTPair, "GTC",
		0.0, 0.0,
		"", "LIMIT")
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping test")
	}
	_, err := b.GetOrders(testSPOTPair, "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping test")
	}
	var getOrdersRequest = order.GetOrdersRequest{
		Pairs: []currency.Pair{
			{
				Delimiter: "-",
				Base:      currency.BTC,
				Quote:     currency.USD,
			},
			{
				Delimiter: "-",
				Base:      currency.XRP,
				Quote:     currency.USD,
			},
		},
		Type:      order.AnyType,
		AssetType: asset.Spot,
	}

	_, err := b.GetActiveOrders(&getOrdersRequest)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping test")
	}
	var getOrdersRequest = order.GetOrdersRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
	}
	_, err := b.GetOrderHistory(&getOrdersRequest)
	if err != nil {
		t.Error(err)
	}
}

func TestTradeHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping test")
	}
	_, err := b.TradeHistory("",
		time.Time{}, time.Time{},
		0, 0, 0,
		false,
		"", "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expected := exchange.NoAPIWithdrawalMethodsText
	actual := b.FormatWithdrawPermissions()
	if actual != expected {
		t.Errorf("Expected: %s, Received: %s", expected, actual)
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	feeBuilder := &exchange.FeeBuilder{
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          currency.NewPair(currency.BTC, currency.USD),
		IsMaker:       true,
		Amount:        1,
		PurchasePrice: 1000,
	}

	b.GetFeeByType(feeBuilder)
	if !areTestAPIKeysSet() {
		if feeBuilder.FeeType != exchange.OfflineTradeFee {
			t.Errorf("Expected %v, received %v", exchange.OfflineTradeFee, feeBuilder.FeeType)
		}
	} else {
		if feeBuilder.FeeType != exchange.CryptocurrencyTradeFee {
			t.Errorf("Expected %v, received %v", exchange.CryptocurrencyTradeFee, feeBuilder.FeeType)
		}
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()

	feeBuilder := &exchange.FeeBuilder{
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          currency.NewPair(currency.BTC, currency.USD),
		IsMaker:       true,
		Amount:        1,
		PurchasePrice: 1000,
	}

	if _, err := b.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	feeBuilder.IsMaker = false
	if _, err := b.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := b.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	feeBuilder.Pair.Base = currency.USDT
	if _, err := b.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if _, err := b.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	feeBuilder.Amount = 1000000
	if _, err := b.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	if _, err := b.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	feeBuilder.Amount = 1000
	if _, err := b.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
}

func TestParseOrderTime(t *testing.T) {
	expected := int64(1534792846)
	actual, err := parseOrderTime("2018-08-20 19:20:46")
	if err != nil {
		t.Fatal(err)
	}
	if expected != actual.Unix() {
		t.Errorf("TestParseOrderTime expected: %d, got %d", expected, actual.Unix())
	}
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys are unset or canManipulateRealOrders is false")
	}
	var orderSubmission = &order.Submit{
		Pair: currency.Pair{
			Base:  currency.BTC,
			Quote: currency.USD,
		},
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     -100000000,
		Amount:    1,
		ClientID:  "",
		AssetType: asset.Spot,
	}
	response, err := b.SubmitOrder(orderSubmission)
	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelAllAfter(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys are unset or canManipulateRealOrders is false")
	}

	err := b.CancelAllAfter(1)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys are unset or canManipulateRealOrders is false")
	}
	currencyPair := currency.NewPairWithDelimiter(currency.BTC.String(),
		currency.USD.String(),
		"-")
	var orderCancellation = &order.Cancel{
		ID:            "b334ecef-2b42-4998-b8a4-b6b14f6d2671",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
		AssetType:     asset.Spot,
	}
	err := b.CancelOrder(orderCancellation)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys are unset or canManipulateRealOrders is false")
	}
	_, err := b.CancelExistingOrder("", testSPOTPair, "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys are unset or canManipulateRealOrders is false")
	}
	currencyPair := currency.NewPairWithDelimiter(currency.BTC.String(),
		currency.USD.String(),
		"-")
	var orderCancellation = &order.Cancel{
		ID:            "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
		AssetType:     asset.Spot,
	}
	resp, err := b.CancelAllOrders(orderCancellation)

	if err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
	for k, v := range resp.Status {
		if strings.Contains(v, "Failed") {
			t.Errorf("order id: %s failed to cancel: %v", k, v)
		}
	}
}

func TestWsOrderbook(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`{"topic":"orderBookL2Api:BTC-USD_0","data":{"buyQuote":[{"price":"9272.0","size":"0.077"},{"price":"9271.0","size":"1.122"},{"price":"9270.0","size":"2.548"},{"price":"9267.5","size":"1.015"},{"price":"9265.5","size":"0.930"},{"price":"9265.0","size":"0.475"},{"price":"9264.5","size":"2.216"},{"price":"9264.0","size":"9.709"},{"price":"9263.5","size":"3.667"},{"price":"9263.0","size":"8.481"},{"price":"9262.5","size":"7.660"},{"price":"9262.0","size":"9.689"},{"price":"9261.5","size":"4.213"},{"price":"9261.0","size":"1.491"},{"price":"9260.5","size":"6.264"},{"price":"9260.0","size":"1.690"},{"price":"9259.5","size":"5.718"},{"price":"9259.0","size":"2.706"},{"price":"9258.5","size":"0.192"},{"price":"9258.0","size":"1.592"},{"price":"9257.5","size":"1.749"},{"price":"9257.0","size":"8.104"},{"price":"9256.0","size":"0.161"},{"price":"9252.0","size":"1.544"},{"price":"9249.5","size":"1.462"},{"price":"9247.5","size":"1.833"},{"price":"9247.0","size":"0.168"},{"price":"9245.5","size":"1.941"},{"price":"9244.0","size":"1.423"},{"price":"9243.5","size":"0.175"}],"currency":"USD","sellQuote":[{"price":"9303.5","size":"1.839"},{"price":"9303.0","size":"2.067"},{"price":"9302.0","size":"0.117"},{"price":"9298.5","size":"1.569"},{"price":"9297.0","size":"1.527"},{"price":"9295.0","size":"0.184"},{"price":"9294.0","size":"1.785"},{"price":"9289.0","size":"1.673"},{"price":"9287.5","size":"4.194"},{"price":"9287.0","size":"6.622"},{"price":"9286.5","size":"2.147"},{"price":"9286.0","size":"3.348"},{"price":"9285.5","size":"5.655"},{"price":"9285.0","size":"10.423"},{"price":"9284.5","size":"6.233"},{"price":"9284.0","size":"8.860"},{"price":"9283.5","size":"9.441"},{"price":"9283.0","size":"3.455"},{"price":"9282.5","size":"11.033"},{"price":"9282.0","size":"11.471"},{"price":"9281.5","size":"4.742"},{"price":"9281.0","size":"14.789"},{"price":"9280.5","size":"11.117"},{"price":"9280.0","size":"0.807"},{"price":"9279.5","size":"1.651"},{"price":"9279.0","size":"0.244"},{"price":"9278.5","size":"0.533"},{"price":"9277.0","size":"1.447"},{"price":"9273.0","size":"1.976"},{"price":"9272.5","size":"0.093"}]}}`)
	err := b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsTrades(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`{"topic":"tradeHistory:BTC-USD","data":[{"amount":0.09,"gain":1,"newest":0,"price":9273.6,"serialId":0,"transactionUnixtime":1580349090693}]}`)
	err := b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsOrderNotification(t *testing.T) {
	t.Parallel()
	status := []string{"ORDER_INSERTED", "ORDER_CANCELLED", "TRIGGER_INSERTED", "ORDER_FULL_TRANSACTED", "ORDER_PARTIALLY_TRANSACTED", "INSUFFICIENT_BALANCE", "TRIGGER_ACTIVATED", "MARKET_UNAVAILABLE"}
	for i := range status {
		pressXToJSON := []byte(`{"topic": "notificationApi","data": [{"symbol": "BTC-USD","orderID": "1234","orderMode": "MODE_BUY","orderType": "TYPE_LIMIT","price": "1","size": "1","status": "` + status[i] + `","timestamp": "1580349090693","type": "STOP","triggerPrice": "1"}]}`)
		err := b.wsHandleData(pressXToJSON)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestStatusToStandardStatus(t *testing.T) {
	type TestCases struct {
		Case   string
		Result order.Status
	}
	testCases := []TestCases{
		{Case: "ORDER_INSERTED", Result: order.New},
		{Case: "TRIGGER_INSERTED", Result: order.New},
		{Case: "ORDER_CANCELLED", Result: order.Cancelled},
		{Case: "ORDER_FULL_TRANSACTED", Result: order.Filled},
		{Case: "ORDER_PARTIALLY_TRANSACTED", Result: order.PartiallyFilled},
		{Case: "TRIGGER_ACTIVATED", Result: order.Active},
		{Case: "INSUFFICIENT_BALANCE", Result: order.InsufficientBalance},
		{Case: "MARKET_UNAVAILABLE", Result: order.MarketUnavailable},
		{Case: "LOL", Result: order.UnknownStatus},
	}
	for i := range testCases {
		result, _ := stringToOrderStatus(testCases[i].Case)
		if result != testCases[i].Result {
			t.Errorf("Exepcted: %v, received: %v", testCases[i].Result, result)
		}
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	assets := b.GetAssetTypes(false)
	for i := range assets {
		data, err := b.FetchTradablePairs(assets[i])
		if err != nil {
			t.Fatal(err)
		}
		if len(data) == 0 {
			t.Fatal("data cannot be zero")
		}
	}
}

func TestMatchType(t *testing.T) {
	t.Parallel()
	ret := matchType(1, order.AnyType)
	if !ret {
		t.Fatal("expected true value")
	}

	ret = matchType(76, order.Market)
	if ret {
		t.Fatal("expected false match")
	}

	ret = matchType(76, order.Limit)
	if !ret {
		t.Fatal("expected true match")
	}

	ret = matchType(77, order.Market)
	if !ret {
		t.Fatal("expected true match")
	}
}

func TestSeedOrderSizeLimits(t *testing.T) {
	t.Parallel()
	err := b.seedOrderSizeLimits()
	if err != nil {
		t.Fatal(err)
	}
}

func TestOrderSizeLimits(t *testing.T) {
	t.Parallel()
	seedOrderSizeLimitMap()
	_, ok := OrderSizeLimits(testSPOTPair)
	if !ok {
		t.Fatal("expected BTC-USD to be found in map")
	}

	_, ok = OrderSizeLimits("XRP-GARBAGE")
	if ok {
		t.Fatal("expected false value for XRP-GARBAGE")
	}
}

func seedOrderSizeLimitMap() {
	testOrderSizeLimits := []struct {
		name string
		o    OrderSizeLimit
	}{
		{
			name: "XRP-USD",
			o: OrderSizeLimit{
				MinSizeIncrement: 1,
				MinOrderSize:     1,
				MaxOrderSize:     1000000,
			},
		},
		{
			name: "LTC-USD",
			o: OrderSizeLimit{
				MinSizeIncrement: 0.01,
				MinOrderSize:     0.01,
				MaxOrderSize:     5000,
			},
		},
		{
			name: "BTC-USD",
			o: OrderSizeLimit{
				MinSizeIncrement: 0.0001,
				MinOrderSize:     1,
				MaxOrderSize:     1000000,
			},
		},
	}

	orderSizeLimitMap.Range(func(key interface{}, _ interface{}) bool {
		orderSizeLimitMap.Delete(key)
		return true
	})

	for x := range testOrderSizeLimits {
		orderSizeLimitMap.Store(testOrderSizeLimits[x].name, testOrderSizeLimits[x].o)
	}
}

func TestWithinLimits(t *testing.T) {
	t.Parallel()
	seedOrderSizeLimitMap()
	p, _ := currency.NewPairDelimiter("XRP-USD", "-")
	v := b.withinLimits(p, 1.0)
	if !v {
		t.Fatal("expected valid limits")
	}
	v = b.withinLimits(p, 5.0000001)
	if v {
		t.Fatal("expected invalid limits")
	}
	v = b.withinLimits(p, 100)
	if !v {
		t.Fatal("expected valid limits")
	}
	v = b.withinLimits(p, 10.1)
	if v {
		t.Fatal("expected invalid limits")
	}

	p.Base = currency.LTC
	v = b.withinLimits(p, 10)
	if v {
		t.Fatal("expected valid limits")
	}

	v = b.withinLimits(p, 0.009)
	if !v {
		t.Fatal("expected invalid limits")
	}
	p.Base = currency.BTC
	v = b.withinLimits(p, 10)
	if v {
		t.Fatal("expected valid limits")
	}

	v = b.withinLimits(p, 0.001)
	if !v {
		t.Fatal("expected invalid limits")
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString(testSPOTPair)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetRecentTrades(currencyPair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	currencyPair, err = currency.NewPairFromString(testFUTURESPair)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetRecentTrades(currencyPair, asset.Futures)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	curr, _ := currency.NewPairFromString(testSPOTPair)

	_, err := b.GetHistoricTrades(curr, asset.Spot, time.Now().Add(-time.Minute), time.Now())
	if err == nil {
		t.Fatal("expected error")
	}
	if err != common.ErrFunctionNotSupported {
		t.Error("unexpected error")
	}
}

func TestOrderbookFilter(t *testing.T) {
	t.Parallel()
	if !b.orderbookFilter(0, 1) {
		t.Fatal("incorrect filtering")
	}
	if !b.orderbookFilter(1, 0) {
		t.Fatal("incorrect filtering")
	}
	if !b.orderbookFilter(0, 0) {
		t.Fatal("incorrect filtering")
	}
	if b.orderbookFilter(1, 1) {
		t.Fatal("incorrect filtering")
	}
}

func TestWsLogin(t *testing.T) {
	t.Parallel()
	data := []byte(`{"event":"login","success":true}`)
	err := b.wsHandleData(data)
	if err != nil {
		t.Error(err)
	}
	if !b.Websocket.CanUseAuthenticatedEndpoints() {
		t.Error("expected true")
	}

	data = []byte(`{"event":"login","success":false}`)
	err = b.wsHandleData(data)
	if err != nil {
		t.Error(err)
	}
	if b.Websocket.CanUseAuthenticatedEndpoints() {
		t.Error("expected false")
	}
}

func TestWsSubscription(t *testing.T) {
	t.Parallel()
	data := []byte(`{"event":"subscribe","channel":["orderBookL2Api:SFI-ETH_0","tradeHistory:SFI-ETH"]}`)
	err := b.wsHandleData(data)
	if err != nil {
		t.Error(err)
	}
}

func TestWsUnexpectedData(t *testing.T) {
	t.Parallel()
	data := []byte(`{}`)
	err := b.wsHandleData(data)
	if err != nil && err.Error() != "BTSE - Unhandled websocket message: {}" {
		t.Error(err)
	}
	if err == nil {
		t.Error("expected error response from bad data")
	}
}
