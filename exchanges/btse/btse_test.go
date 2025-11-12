package btse

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
)

// Please supply your own keys here to do better tests
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var (
	e           *Exchange
	futuresPair = currency.NewPair(currency.ENJ, currency.PFC)
	spotPair    = currency.NewPairWithDelimiter("BTC", "USD", "-")
)

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("BTSE Setup error: %s", err)
	}

	if apiKey != "" && apiSecret != "" {
		e.API.AuthenticatedSupport = true
		e.SetCredentials(apiKey, apiSecret, "", "", "", "")
	}

	os.Exit(m.Run())
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
	expected := map[asset.Item][]string{
		asset.Spot:    {"BTCUSD", "BTCUSDT", "ETHBTC"},
		asset.Futures: {"BTCPFC", "ETHPFC"},
	}
	for a, pairs := range expected {
		for _, symb := range pairs {
			_, err := e.CurrencyPairs.Match(symb, a)
			assert.NoErrorf(t, err, "Should find pair %s for %s", symb, a)
		}
	}
}

func TestFetchFundingHistory(t *testing.T) {
	_, err := e.FetchFundingHistory(t.Context(), "")
	assert.NoError(t, err, "FetchFundingHistory should not error")
}

func TestGetMarketsSummary(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarketSummary(t.Context(), "", true)
	assert.NoError(t, err, "GetMarketSummary should not error")

	ret, err := e.GetMarketSummary(t.Context(), spotPair.String(), true)
	assert.NoError(t, err, "GetMarketSummary should not error")
	assert.Len(t, ret, 1, "expected only one result when requesting BTC-USD data received")

	_, err = e.GetMarketSummary(t.Context(), "", false)
	assert.NoError(t, err, "GetMarketSummary should not error")
}

func TestFetchOrderBook(t *testing.T) {
	t.Parallel()
	_, err := e.FetchOrderbook(t.Context(), spotPair.String(), 0, 1, 1, true)
	assert.NoError(t, err, "FetchOrderBook should not error")

	_, err = e.FetchOrderbook(t.Context(), futuresPair.String(), 0, 1, 1, false)
	assert.NoError(t, err, "FetchOrderBook should not error")

	_, err = e.FetchOrderbook(t.Context(), spotPair.String(), 1, 1, 1, true)
	assert.NoError(t, err, "FetchOrderBook should not error")
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateOrderbook(t.Context(), spotPair, asset.Spot)
	assert.NoError(t, err, "UpdateOrderbook should not error")

	_, err = e.UpdateOrderbook(t.Context(), futuresPair, asset.Futures)
	assert.NoError(t, err, "UpdateOrderbook should not error")
}

func TestFetchOrderBookL2(t *testing.T) {
	t.Parallel()
	_, err := e.FetchOrderbookL2(t.Context(), spotPair.String(), 20)
	assert.NoError(t, err, "FetchOrderBookL2 should not error")
}

func TestOHLCV(t *testing.T) {
	t.Parallel()
	_, err := e.GetOHLCV(t.Context(), spotPair.String(), time.Now().AddDate(0, 0, -1), time.Now(), 60, asset.Spot)
	assert.NoError(t, err, "GetOHLCV should not error")

	_, err = e.GetOHLCV(t.Context(), spotPair.String(), time.Now(), time.Now().AddDate(0, 0, -1), 60, asset.Spot)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd, "GetOHLCV should error if start date after end date")

	_, err = e.GetOHLCV(t.Context(), futuresPair.String(), time.Now().AddDate(0, 0, -1), time.Now(), 60, asset.Futures)
	assert.NoError(t, err, "GetOHLCV should not error")
}

func TestGetPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetPrice(t.Context(), spotPair.String())
	assert.NoError(t, err, "GetPrice should not error")
}

func TestFormatExchangeKlineInterval(t *testing.T) {
	ret := e.FormatExchangeKlineInterval(kline.OneDay)
	assert.Equal(t, "1440", ret, "FormatExchangeKlineInterval should return correct value")
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	r := e.Requester
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test exchange Setup must not error")
	e.Requester = r
	start := time.Now().AddDate(0, 0, -3)
	_, err := e.GetHistoricCandles(t.Context(), spotPair, asset.Spot, kline.OneHour, start, time.Now())
	assert.NoError(t, err, "GetHistoricCandles should not error")

	_, err = e.GetHistoricCandles(t.Context(), spotPair, asset.Spot, kline.OneDay, start, time.Now())
	assert.NoError(t, err, "GetHistoricCandles should not error")
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	r := e.Requester
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test exchange Setup must not error")
	e.Requester = r
	err := e.CurrencyPairs.StorePairs(asset.Futures, currency.Pairs{futuresPair}, true)
	assert.NoError(t, err, "StorePairs should not error")

	start := time.Now().AddDate(0, 0, -1)
	_, err = e.GetHistoricCandlesExtended(t.Context(), spotPair, asset.Spot, kline.OneHour, start, time.Now())
	assert.NoError(t, err, "GetHistoricCandlesExtended should not error")

	_, err = e.GetHistoricCandlesExtended(t.Context(), futuresPair, asset.Futures, kline.OneHour, start, time.Now())
	assert.NoError(t, err, "GetHistoricCandlesExtended should not error")
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetTrades(t.Context(), spotPair.String(), time.Now().AddDate(0, 0, -1), time.Now(), 0, 0, 50, false, true)
	assert.NoError(t, err, "GetTrades should not error")

	_, err = e.GetTrades(t.Context(), spotPair.String(), time.Now(), time.Now().AddDate(0, -1, 0), 0, 0, 50, false, true)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd, "GetTrades should error if start date after end date")

	_, err = e.GetTrades(t.Context(), futuresPair.String(), time.Now().AddDate(0, 0, -1), time.Now(), 0, 0, 50, false, false)
	assert.NoError(t, err, "GetTrades should not error")
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTicker(t.Context(), spotPair, asset.Spot)
	assert.NoError(t, common.ExcludeError(err, ticker.ErrBidEqualsAsk), "UpdateTickers may only error about locked markets")

	_, err = e.UpdateTicker(t.Context(), futuresPair, asset.Futures)
	assert.NoError(t, common.ExcludeError(err, ticker.ErrBidEqualsAsk), "UpdateTickers may only error about locked markets")
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := e.UpdateTickers(t.Context(), asset.Spot)
	assert.NoError(t, common.ExcludeError(err, ticker.ErrBidEqualsAsk), "UpdateTickers may only error about locked markets")

	err = e.UpdateTickers(t.Context(), asset.Futures)
	assert.NoError(t, common.ExcludeError(err, ticker.ErrBidEqualsAsk), "UpdateTickers may only error about locked markets")
}

func TestGetCurrentServerTime(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrentServerTime(t.Context())
	assert.NoError(t, err, "GetCurrentServerTime should not error")
}

func TestWrapperGetServerTime(t *testing.T) {
	t.Parallel()
	st, err := e.GetServerTime(t.Context(), asset.Spot)
	assert.NoError(t, err, "GetServerTime should not error")
	assert.WithinRange(t, st, time.Now().Add(-24*time.Hour), time.Now().Add(24*time.Hour), "Time should be within a day of now")
}

func TestGetWalletInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetWalletInformation(t.Context())
	assert.NoError(t, err, "GetWalletInformation should not error")
}

func TestGetFeeInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetFeeInformation(t.Context(), "")
	assert.NoError(t, err, "GetFeeInformation should not error")
}

func TestGetWalletHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetWalletHistory(t.Context(), spotPair.String(), time.Time{}, time.Time{}, 50)
	assert.NoError(t, err, "GetWalletHistory should not error")
}

func TestGetWalletAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetWalletAddress(t.Context(), "XRP")
	assert.NoError(t, err, "GetWalletAddress should not error")
}

func TestCreateWalletAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.CreateWalletAddress(t.Context(), "XRP")
	assert.NoError(t, err, "CreateWalletAddress should not error")
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetDepositAddress(t.Context(), currency.BTC, "", "")
	assert.NoError(t, err, "GetDepositAddress should not error")
}

func TestCreateOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CreateOrder(t.Context(), "", 0.0, false, -1, "BUY", 100, 0, 0, spotPair.String(), "GTC", 0.0, 0.0, "LIMIT", "LIMIT")
	assert.NoError(t, err, "CreateOrder should not error")
}

func TestBTSEIndexOrderPeg(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.IndexOrderPeg(t.Context(), "", 0.0, false, -1, "BUY", 100, 0, 0, spotPair.String(), "GTC", 0.0, 0.0, "", "LIMIT")
	assert.NoError(t, err, "IndexOrderPeg should not error")
}

func TestGetOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetOrders(t.Context(), spotPair.String(), "", "")
	assert.NoError(t, err, "GetOrders should not error")
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	getOrdersRequest := order.MultiOrderRequest{
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
		Side:      order.AnySide,
	}

	_, err := e.GetActiveOrders(t.Context(), &getOrdersRequest)
	assert.NoError(t, err, "GetActiveOrders should not error")
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}
	_, err := e.GetOrderHistory(t.Context(), &getOrdersRequest)
	assert.NoError(t, err, "GetOrderHistory should not error")
}

func TestTradeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.TradeHistory(t.Context(), "", time.Time{}, time.Time{}, 0, 0, 0, false, "", "")
	assert.NoError(t, err, "TradeHistory should not error")
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	assert.Equal(t, exchange.NoAPIWithdrawalMethodsText, e.FormatWithdrawPermissions(), "FormatWithdrawPermissions should return correct format")
}

func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	feeBuilder := &exchange.FeeBuilder{
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          spotPair,
		IsMaker:       true,
		Amount:        1,
		PurchasePrice: 1000,
	}

	_, err := e.GetFeeByType(t.Context(), feeBuilder)
	assert.NoError(t, err, "GetFeeByType should not error")
	if !sharedtestvalues.AreAPICredentialsSet(e) {
		assert.Equal(t, exchange.OfflineTradeFee, feeBuilder.FeeType, "FeeBuilder should give offline trade fee type")
	} else {
		assert.Equal(t, exchange.CryptocurrencyTradeFee, feeBuilder.FeeType, "FeeBuilder should give crypto trade fee type")
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()

	feeBuilder := &exchange.FeeBuilder{
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          spotPair,
		IsMaker:       true,
		Amount:        1,
		PurchasePrice: 1000,
	}

	_, err := e.GetFee(t.Context(), feeBuilder)
	assert.NoError(t, err, "fee builuder should not error for maker")

	feeBuilder.IsMaker = false
	_, err = e.GetFee(t.Context(), feeBuilder)
	assert.NoError(t, err, "fee builuder should not error for taker")

	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	_, err = e.GetFee(t.Context(), feeBuilder)
	assert.NoError(t, err, "fee builuder should not error for withdrawal")

	feeBuilder.Pair.Base = currency.USDT
	_, err = e.GetFee(t.Context(), feeBuilder)
	assert.NoError(t, err, "fee builuder should not error for USDT")

	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	_, err = e.GetFee(t.Context(), feeBuilder)
	assert.NoError(t, err, "fee builuder should not error for International deposits")

	feeBuilder.Amount = 1000000
	_, err = e.GetFee(t.Context(), feeBuilder)
	assert.NoError(t, err, "fee builuder should not error for a squillion")

	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	_, err = e.GetFee(t.Context(), feeBuilder)
	assert.NoError(t, err, "fee builuder should not error for International withdrawals")

	feeBuilder.Amount = 1000
	_, err = e.GetFee(t.Context(), feeBuilder)
	assert.NoError(t, err, "fee builuder should not error for a fraction of a squillion")
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	orderSubmission := &order.Submit{
		Exchange: e.Name,
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
	response, err := e.SubmitOrder(t.Context(), orderSubmission)
	assert.NoError(t, err, "SubmitOrder should not error")
	assert.Equal(t, order.New, response.Status, "Response Status should be New")
}

func TestCancelAllAfter(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	err := e.CancelAllAfter(t.Context(), 1)
	assert.NoError(t, err, "CancelAllAfter should not error")
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	// TODO: Place an order to make sure we can cancel it
	orderCancellation := &order.Cancel{
		OrderID:   "b334ecef-2b42-4998-b8a4-b6b14f6d2671",
		AccountID: "1",
		Pair:      spotPair,
		AssetType: asset.Spot,
	}
	err := e.CancelOrder(t.Context(), orderCancellation)
	assert.NoError(t, err, "CancelOrder should not error")
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	// TODO: Place an order to make sure we can cancel it
	_, err := e.CancelExistingOrder(t.Context(), "", spotPair.String(), "")
	assert.NoError(t, err, "CancelExistingOrder should not error")
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      spotPair,
		AssetType: asset.Spot,
	}
	resp, err := e.CancelAllOrders(t.Context(), orderCancellation)

	assert.NoError(t, err, "CancelAllOrders should not error")
	for k, v := range resp.Status {
		assert.NotContainsf(t, v, "Failed", "order %s should not fail to cancel", k)
	}
}

func TestWsOrderbook(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`{"topic":"orderBookL2Api:BTC-USD_0","data":{"buyQuote":[{"price":"9272.0","size":"0.077"},{"price":"9271.0","size":"1.122"},{"price":"9270.0","size":"2.548"},{"price":"9267.5","size":"1.015"},{"price":"9265.5","size":"0.930"},{"price":"9265.0","size":"0.475"},{"price":"9264.5","size":"2.216"},{"price":"9264.0","size":"9.709"},{"price":"9263.5","size":"3.667"},{"price":"9263.0","size":"8.481"},{"price":"9262.5","size":"7.660"},{"price":"9262.0","size":"9.689"},{"price":"9261.5","size":"4.213"},{"price":"9261.0","size":"1.491"},{"price":"9260.5","size":"6.264"},{"price":"9260.0","size":"1.690"},{"price":"9259.5","size":"5.718"},{"price":"9259.0","size":"2.706"},{"price":"9258.5","size":"0.192"},{"price":"9258.0","size":"1.592"},{"price":"9257.5","size":"1.749"},{"price":"9257.0","size":"8.104"},{"price":"9256.0","size":"0.161"},{"price":"9252.0","size":"1.544"},{"price":"9249.5","size":"1.462"},{"price":"9247.5","size":"1.833"},{"price":"9247.0","size":"0.168"},{"price":"9245.5","size":"1.941"},{"price":"9244.0","size":"1.423"},{"price":"9243.5","size":"0.175"}],"currency":"USD","sellQuote":[{"price":"9303.5","size":"1.839"},{"price":"9303.0","size":"2.067"},{"price":"9302.0","size":"0.117"},{"price":"9298.5","size":"1.569"},{"price":"9297.0","size":"1.527"},{"price":"9295.0","size":"0.184"},{"price":"9294.0","size":"1.785"},{"price":"9289.0","size":"1.673"},{"price":"9287.5","size":"4.194"},{"price":"9287.0","size":"6.622"},{"price":"9286.5","size":"2.147"},{"price":"9286.0","size":"3.348"},{"price":"9285.5","size":"5.655"},{"price":"9285.0","size":"10.423"},{"price":"9284.5","size":"6.233"},{"price":"9284.0","size":"8.860"},{"price":"9283.5","size":"9.441"},{"price":"9283.0","size":"3.455"},{"price":"9282.5","size":"11.033"},{"price":"9282.0","size":"11.471"},{"price":"9281.5","size":"4.742"},{"price":"9281.0","size":"14.789"},{"price":"9280.5","size":"11.117"},{"price":"9280.0","size":"0.807"},{"price":"9279.5","size":"1.651"},{"price":"9279.0","size":"0.244"},{"price":"9278.5","size":"0.533"},{"price":"9277.0","size":"1.447"},{"price":"9273.0","size":"1.976"},{"price":"9272.5","size":"0.093"}]}}`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	assert.NoError(t, err, "wsHandleData orderBookL2Api should not error")
	// TODO: Meaningful test of data parsing
}

func TestWSTrades(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	testexch.FixtureToDataHandler(t, "testdata/wsAllTrades.json", e.wsHandleData)
	e.Websocket.DataHandler.Close()

	exp := []trade.Data{
		{
			Exchange:     e.Name,
			CurrencyPair: spotPair,
			Timestamp:    time.UnixMilli(1741836562893).UTC(),
			Price:        83894.01,
			Amount:       0.00067,
			Side:         order.Buy,
			TID:          "74040596",
			AssetType:    asset.Spot,
		},
		{
			Exchange:     e.Name,
			CurrencyPair: spotPair,
			Timestamp:    time.UnixMilli(1741836562687).UTC(),
			Price:        83894.87,
			Amount:       0.0035,
			Side:         order.Sell,
			TID:          "74040529",
			AssetType:    asset.Spot,
		},
	}
	require.Len(t, e.Websocket.DataHandler.C, 2, "Must see the correct number of trades")
	for resp := range e.Websocket.DataHandler.C {
		switch v := resp.Data.(type) {
		case trade.Data:
			i := 1 - len(e.Websocket.DataHandler.C)
			require.Equalf(t, exp[i], v, "Trade [%d] must be correct", i)
		case error:
			t.Error(v)
		default:
			t.Errorf("Unexpected type in DataHandler: %T(%s)", v, v)
		}
	}
}

func TestWsOrderNotification(t *testing.T) {
	t.Parallel()
	status := []string{"ORDER_INSERTED", "ORDER_CANCELLED", "TRIGGER_INSERTED", "ORDER_FULL_TRANSACTED", "ORDER_PARTIALLY_TRANSACTED", "INSUFFICIENT_BALANCE", "TRIGGER_ACTIVATED", "MARKET_UNAVAILABLE"}
	for i := range status {
		pressXToJSON := []byte(`{"topic": "notificationApi","data": [{"symbol": "BTC-USD","orderID": "1234","orderMode": "MODE_BUY","orderType": "TYPE_LIMIT","price": "1","size": "1","status": "` + status[i] + `","timestamp": "1580349090693","type": "STOP","triggerPrice": "1"}]}`)
		err := e.wsHandleData(t.Context(), pressXToJSON)
		assert.NoErrorf(t, err, "wsHandleData notificationApi should not error on %s", status[i])
		// TODO: Meaningful test of data parsing
	}
}

func TestStatusToStandardStatus(t *testing.T) {
	type TestCases struct {
		Case   string
		Result order.Status
	}
	testCases := []*TestCases{
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
	for _, tt := range testCases {
		result, err := stringToOrderStatus(tt.Case)
		if tt.Result != order.UnknownStatus {
			assert.NoErrorf(t, err, "stringToOrderStatus should not error for %s", tt.Case)
		}
		assert.Equalf(t, tt.Result, result, "stringToOrderStatus should return correct value for %s", tt.Case)
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	assets := e.GetAssetTypes(false)
	for i := range assets {
		data, err := e.FetchTradablePairs(t.Context(), assets[i])
		assert.NoErrorf(t, err, "FetchTradablePairs should not error for %s", assets[i])
		assert.NotEmpty(t, data, "FetchTradablePairs should return some pairs")
	}
}

func TestMatchType(t *testing.T) {
	t.Parallel()
	ret := matchType(1, order.AnyType)
	assert.True(t, ret, "matchType should match AnyType")

	ret = matchType(76, order.Market)
	assert.False(t, ret, "matchType should not false positive")

	ret = matchType(76, order.Limit)
	assert.True(t, ret, "matchType should match")

	ret = matchType(77, order.Market)
	assert.True(t, ret, "matchType should match")
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	for _, a := range e.GetAssetTypes(false) {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			require.NoError(t, e.UpdateOrderExecutionLimits(t.Context(), a), "UpdateOrderExecutionLimits must not error")
			pairs, err := e.CurrencyPairs.GetPairs(a, true)
			require.NoError(t, err, "GetPairs must not error")
			l, err := e.GetOrderExecutionLimits(a, pairs[0])
			require.NoError(t, err, "GetOrderExecutionLimits must not error")
			assert.Positive(t, l.MinimumBaseAmount, "MinimumBaseAmount should be positive")
			assert.Positive(t, l.MaximumBaseAmount, "MaximumBaseAmount should be positive")
			assert.Positive(t, l.AmountStepIncrementSize, "AmountStepIncrementSize should be positive")
			assert.Positive(t, l.MinPrice, "MinPrice should be positive")
			assert.Positive(t, l.PriceStepIncrementSize, "PriceStepIncrementSize should be positive")
		})
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecentTrades(t.Context(), spotPair, asset.Spot)
	assert.NoError(t, err, "GetRecentTrades Should not error")

	_, err = e.GetRecentTrades(t.Context(), futuresPair, asset.Futures)
	assert.NoError(t, err, "GetRecentTrades Should not error")
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricTrades(t.Context(), spotPair, asset.Spot, time.Now().Add(-time.Minute), time.Now())
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetHistoricTrades should not be supported")
}

func TestOrderbookFilter(t *testing.T) {
	t.Parallel()
	assert.True(t, e.orderbookFilter(0, 1), "orderbookFilter should return correctly")
	assert.True(t, e.orderbookFilter(1, 0), "orderbookFilter should return correctly")
	assert.True(t, e.orderbookFilter(0, 0), "orderbookFilter should return correctly")
	assert.False(t, e.orderbookFilter(1, 1), "orderbookFilter should return correctly")
}

func TestWsLogin(t *testing.T) {
	t.Parallel()
	data := []byte(`{"event":"login","success":true}`)
	err := e.wsHandleData(t.Context(), data)
	assert.NoError(t, err, "wsHandleData login should not error")
	assert.True(t, e.Websocket.CanUseAuthenticatedEndpoints(), "CanUseAuthenticatedEndpoints should be true after login")

	data = []byte(`{"event":"login","success":false}`)
	err = e.wsHandleData(t.Context(), data)
	assert.NoError(t, err, "wsHandleData login should not error")
	assert.False(t, e.Websocket.CanUseAuthenticatedEndpoints(), "CanUseAuthenticatedEndpoints should be false failed login")
}

func TestWsSubscription(t *testing.T) {
	t.Parallel()
	data := []byte(`{"event":"subscribe","channel":["orderBookL2Api:SFI-ETH_0","tradeHistory:SFI-ETH"]}`)
	err := e.wsHandleData(t.Context(), data)
	assert.NoError(t, err, "wsHandleData subscribe should not error")
}

func TestWsUnexpectedData(t *testing.T) {
	t.Parallel()
	data := []byte(`{}`)
	err := e.wsHandleData(t.Context(), data)
	assert.ErrorContains(t, err, websocket.UnhandledMessage, "wsHandleData should error on empty message")
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Margin)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetFuturesContractDetails should error correctly on Margin")

	_, err = e.GetFuturesContractDetails(t.Context(), asset.Futures)
	assert.NoError(t, err, "GetFuturesContractDetails should not error on Futures")
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset:                asset.USDTMarginedFutures,
		Pair:                 currency.NewBTCUSDT(),
		IncludePredictedRate: true,
	})
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetLatestFundingRates should error on Margin")

	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset: asset.Futures,
	})
	assert.NoError(t, err, "GetLatestFundingRates should not error on futures")

	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset: asset.Futures,
		Pair:  futuresPair,
	})
	assert.NoError(t, err, "GetLatestFundingRates should not error on futures")
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	isPerp, err := e.IsPerpetualFutureCurrency(asset.CoinMarginedFutures, currency.NewBTCUSD())
	assert.NoError(t, err, "IsPerpetualFutureCurrency should not error")
	assert.False(t, isPerp, "IsPerpetualFutureCurrency should return true for a Margin pair")

	isPerp, err = e.IsPerpetualFutureCurrency(asset.Futures, futuresPair)
	assert.NoError(t, err, "IsPerpetualFutureCurrency should not error")
	assert.True(t, isPerp, "IsPerpetualFutureCurrency should return true for a futures pair")

	isPerp, err = e.IsPerpetualFutureCurrency(asset.Futures, spotPair)
	assert.NoError(t, err, "IsPerpetualFutureCurrency should not error")
	assert.False(t, isPerp, "IsPerpetualFutureCurrency should return false for a spot pair")
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	r := e.Requester
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test exchange Setup must not error")
	testexch.UpdatePairsOnce(t, e)
	e.Requester = r
	cp1 := currency.NewPair(currency.BTC, currency.PFC)
	cp2 := currency.NewPair(currency.ETH, currency.PFC)
	sharedtestvalues.SetupCurrencyPairsForExchangeAsset(t, e, asset.Futures, futuresPair, cp1, cp2)

	resp, err := e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  cp1.Base.Item,
		Quote: cp1.Quote.Item,
		Asset: asset.Futures,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = e.GetOpenInterest(t.Context(),
		key.PairAsset{
			Base:  cp1.Base.Item,
			Quote: cp1.Quote.Item,
			Asset: asset.Futures,
		},
		key.PairAsset{
			Base:  cp2.Base.Item,
			Quote: cp2.Quote.Item,
			Asset: asset.Futures,
		})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = e.GetOpenInterest(t.Context())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	_, err = e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.BTC.Item,
		Quote: currency.USDT.Item,
		Asset: asset.Spot,
	})
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.CurrencyPairs.GetPairs(a, false)
		require.NoErrorf(t, err, "cannot get pairs for %s", a)
		require.NotEmptyf(t, pairs, "no pairs for %s", a)
		resp, err := e.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}

// TestStripExponent exercises StripExponent
func TestStripExponent(t *testing.T) {
	t.Parallel()
	s, err := (&MarketPair{Symbol: "BTC-ETH"}).StripExponent()
	assert.NoError(t, err, "Should not error on a symbol without exponent")
	assert.Empty(t, s, "Should return an empty symbol without exponent")

	for _, p := range []string{"B", "M", "K"} {
		s, err = (&MarketPair{Symbol: p + "_BTC-ETH"}).StripExponent()
		assert.NoError(t, err, "Should not error on a symbol with exponent")
		assert.Equal(t, "BTC-ETH", s, "Should return the symbol without the exponent")
	}

	_, err = (&MarketPair{Symbol: "Z_BTC-ETH"}).StripExponent()
	assert.ErrorIs(t, err, errInvalidPairSymbol, "Should error on a symbol with unknown exponent")

	_, err = (&MarketPair{Symbol: "M_BTC_ETH"}).StripExponent()
	assert.ErrorIs(t, err, errInvalidPairSymbol, "Should error on a symbol with too many underscores")
}

func TestMarketPair(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		symbol         string
		base           currency.Code
		futures        bool
		expectedErr    error
		expectedSymbol string
	}{
		{symbol: "RUNEPFC", base: currency.RUNE, futures: true, expectedSymbol: "RUNEPFC"},
		{symbol: "TRUMPPFC", base: currency.NewCode("TRUMPSOL"), futures: true, expectedSymbol: "TRUMPPFC"},
		{symbol: "BTCUSD", base: currency.NewCode("NAUGHTYBASE"), futures: true, expectedErr: errInvalidPairSymbol},
		{symbol: "NAUGHTYSYMBOL", base: currency.BTC, expectedErr: errInvalidPairSymbol},
		{symbol: "BTC-USD", base: currency.BTC, expectedSymbol: "BTCUSD"},
	} {
		mp := MarketPair{Symbol: tt.symbol, Base: tt.base, Quote: currency.USD, Futures: tt.futures}
		p, err := mp.Pair()
		assert.ErrorIs(t, err, tt.expectedErr, "Pair should not error")
		assert.Equal(t, tt.expectedSymbol, p.String(), "Pair should return the expected symbol")
	}
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	exp := subscription.List{
		{Channel: subscription.AllTradesChannel, QualifiedChannel: "tradeHistoryApi:BTC-USD", Asset: asset.Spot, Pairs: currency.Pairs{spotPair}},
		{Channel: subscription.MyTradesChannel, QualifiedChannel: "notificationApi"},
	}

	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	subs, err := e.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	testsubs.EqualLists(t, exp, subs)

	_, err = subscription.List{{Channel: subscription.OrderbookChannel}}.ExpandTemplates(e)
	assert.ErrorContains(t, err, "Channel not supported", "Sub template should error on unsupported channels")
}
