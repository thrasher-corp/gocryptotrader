package coinbasepro

import (
	"net/http"
	"os"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	c        *CoinbasePro
	testPair = currency.NewPairWithDelimiter(currency.BTC.String(), currency.USD.String(), "-")
)

// Please supply your APIKeys here for better testing
const (
	apiKey                  = ""
	apiSecret               = ""
	clientID                = "" // passphrase you made at API CREATION
	canManipulateRealOrders = false
)

func TestMain(_ *testing.M) {
	os.Exit(0) // Disable full test suite until PR #1381 is merged as more API endpoints have been deprecated over time
}

func TestGetProducts(t *testing.T) {
	_, err := c.GetProducts(t.Context())
	if err != nil {
		t.Errorf("Coinbase, GetProducts() Error: %s", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	_, err := c.GetOrderbook(t.Context(), testPair.String(), 2)
	if err != nil {
		t.Error(err)
	}
	_, err = c.GetOrderbook(t.Context(), testPair.String(), 3)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTicker(t *testing.T) {
	_, err := c.GetTicker(t.Context(), testPair.String())
	if err != nil {
		t.Error("GetTicker() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	_, err := c.GetTrades(t.Context(), testPair.String())
	if err != nil {
		t.Error("GetTrades() error", err)
	}
}

func TestHistoryUnmarshalJSON(t *testing.T) {
	t.Parallel()
	data := []byte(`[[1746649200,96269.22,96307.18,96275.58,96307.18,1.85952049],[1746649140,96256.39,96297.31,96296,96273.29,3.41045323],[1746649080,96256.01,96365.73,96365.73,96299.99,3.56073877]]`)
	var resp []History
	err := json.Unmarshal(data, &resp)
	require.NoError(t, err)
	require.Len(t, resp, 3)
	assert.Equal(t, History{
		Time:   types.Time(time.Unix(1746649200, 0)),
		Low:    96269.22,
		High:   96307.18,
		Open:   96275.58,
		Close:  96307.18,
		Volume: 1.85952049,
	}, resp[0])
}

func TestGetHistoricRates(t *testing.T) {
	t.Parallel()
	result, err := c.GetHistoricRates(t.Context(), "BTC-USD", "", "", 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricRatesGranularityCheck(t *testing.T) {
	end := time.Now()
	start := end.Add(-time.Hour * 2)
	_, err := c.GetHistoricCandles(t.Context(),
		testPair, asset.Spot, kline.OneHour, start, end)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	_, err := c.GetHistoricCandlesExtended(t.Context(), testPair, asset.Spot, kline.OneDay, time.Unix(1546300800, 0), time.Unix(1577836799, 0))
	assert.NoError(t, err, "GetHistoricCandlesExtended should not error")
}

func TestGetStats(t *testing.T) {
	_, err := c.GetStats(t.Context(), testPair.String())
	if err != nil {
		t.Error("GetStats() error", err)
	}
}

func TestGetCurrencies(t *testing.T) {
	_, err := c.GetCurrencies(t.Context())
	if err != nil {
		t.Error("GetCurrencies() error", err)
	}
}

func TestGetCurrentServerTime(t *testing.T) {
	_, err := c.GetCurrentServerTime(t.Context())
	if err != nil {
		t.Error("GetServerTime() error", err)
	}
}

func TestWrapperGetServerTime(t *testing.T) {
	t.Parallel()
	st, err := c.GetServerTime(t.Context(), asset.Spot)
	require.NoError(t, err)

	if st.IsZero() {
		t.Fatal("expected a time")
	}
}

func TestAuthRequests(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)

	_, err := c.GetAccounts(t.Context())
	if err != nil {
		t.Error("GetAccounts() error", err)
	}
	accountResponse, err := c.GetAccount(t.Context(),
		"13371337-1337-1337-1337-133713371337")
	if accountResponse.ID != "" {
		t.Error("Expecting no data returned")
	}
	if err == nil {
		t.Error("Expecting error")
	}
	accountHistoryResponse, err := c.GetAccountHistory(t.Context(),
		"13371337-1337-1337-1337-133713371337")
	if len(accountHistoryResponse) > 0 {
		t.Error("Expecting no data returned")
	}
	if err == nil {
		t.Error("Expecting error")
	}
	getHoldsResponse, err := c.GetHolds(t.Context(),
		"13371337-1337-1337-1337-133713371337")
	if len(getHoldsResponse) > 0 {
		t.Error("Expecting no data returned")
	}
	if err == nil {
		t.Error("Expecting error")
	}
	orderResponse, err := c.PlaceLimitOrder(t.Context(),
		"", 0.001, 0.001,
		order.Buy.Lower(), "", "", testPair.String(), "", false)
	if orderResponse != "" {
		t.Error("Expecting no data returned")
	}
	if err == nil {
		t.Error("Expecting error")
	}
	marketOrderResponse, err := c.PlaceMarketOrder(t.Context(),
		"", 1, 0,
		order.Buy.Lower(), testPair.String(), "")
	if marketOrderResponse != "" {
		t.Error("Expecting no data returned")
	}
	if err == nil {
		t.Error("Expecting error")
	}
	fillsResponse, err := c.GetFills(t.Context(),
		"1337", testPair.String())
	if len(fillsResponse) > 0 {
		t.Error("Expecting no data returned")
	}
	if err == nil {
		t.Error("Expecting error")
	}
	_, err = c.GetFills(t.Context(), "", "")
	if err == nil {
		t.Error("Expecting error")
	}
	marginTransferResponse, err := c.MarginTransfer(t.Context(),
		1, "withdraw", "13371337-1337-1337-1337-133713371337", "BTC")
	if marginTransferResponse.ID != "" {
		t.Error("Expecting no data returned")
	}
	if err == nil {
		t.Error("Expecting error")
	}
	_, err = c.GetPosition(t.Context())
	if err == nil {
		t.Error("Expecting error")
	}
	_, err = c.ClosePosition(t.Context(), false)
	if err == nil {
		t.Error("Expecting error")
	}
	_, err = c.GetPayMethods(t.Context())
	if err != nil {
		t.Error("GetPayMethods() error", err)
	}
	_, err = c.GetCoinbaseAccounts(t.Context())
	if err != nil {
		t.Error("GetCoinbaseAccounts() error", err)
	}
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:        1,
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          testPair,
		PurchasePrice: 1,
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	feeBuilder := setFeeBuilder()
	_, err := c.GetFeeByType(t.Context(), feeBuilder)
	if err != nil {
		t.Fatal(err)
	}
	if !sharedtestvalues.AreAPICredentialsSet(c) {
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
	feeBuilder := setFeeBuilder()

	if sharedtestvalues.AreAPICredentialsSet(c) {
		// CryptocurrencyTradeFee Basic
		if _, err := c.GetFee(t.Context(), feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if _, err := c.GetFee(t.Context(), feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if _, err := c.GetFee(t.Context(), feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if _, err := c.GetFee(t.Context(), feeBuilder); err != nil {
			t.Error(err)
		}
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := c.GetFee(t.Context(), feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	if _, err := c.GetFee(t.Context(), feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.EUR
	if _, err := c.GetFee(t.Context(), feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if _, err := c.GetFee(t.Context(), feeBuilder); err != nil {
		t.Error(err)
	}
}

func TestCalculateTradingFee(t *testing.T) {
	t.Parallel()
	// uppercase
	volume := []Volume{
		{
			ProductID: "BTC_USD",
			Volume:    100,
		},
	}

	if resp := c.calculateTradingFee(volume, currency.BTC, currency.USD, "_", 1, 1, false); resp != float64(0.003) {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.003), resp)
	}

	// lowercase
	volume = []Volume{
		{
			ProductID: "btc_usd",
			Volume:    100,
		},
	}

	if resp := c.calculateTradingFee(volume, currency.BTC, currency.USD, "_", 1, 1, false); resp != float64(0.003) {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.003), resp)
	}

	// mixedCase
	volume = []Volume{
		{
			ProductID: "btc_USD",
			Volume:    100,
		},
	}

	if resp := c.calculateTradingFee(volume, currency.BTC, currency.USD, "_", 1, 1, false); resp != float64(0.003) {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.003), resp)
	}

	// medium volume
	volume = []Volume{
		{
			ProductID: "btc_USD",
			Volume:    10000001,
		},
	}

	if resp := c.calculateTradingFee(volume, currency.BTC, currency.USD, "_", 1, 1, false); resp != float64(0.002) {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.002), resp)
	}

	// high volume
	volume = []Volume{
		{
			ProductID: "btc_USD",
			Volume:    100000010000,
		},
	}

	if resp := c.calculateTradingFee(volume, currency.BTC, currency.USD, "_", 1, 1, false); resp != float64(0.001) {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.001), resp)
	}

	// no match
	volume = []Volume{
		{
			ProductID: "btc_beeteesee",
			Volume:    100000010000,
		},
	}

	if resp := c.calculateTradingFee(volume, currency.BTC, currency.USD, "_", 1, 1, false); resp != float64(0) {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
	}

	// taker
	volume = []Volume{
		{
			ProductID: "btc_USD",
			Volume:    100000010000,
		},
	}

	if resp := c.calculateTradingFee(volume, currency.BTC, currency.USD, "_", 1, 1, true); resp != float64(0) {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	expectedResult := exchange.AutoWithdrawCryptoWithAPIPermissionText + " & " + exchange.AutoWithdrawFiatWithAPIPermissionText
	withdrawPermissions := c.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Pairs:     []currency.Pair{testPair},
		Side:      order.AnySide,
	}

	_, err := c.GetActiveOrders(t.Context(), &getOrdersRequest)
	if sharedtestvalues.AreAPICredentialsSet(c) && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !sharedtestvalues.AreAPICredentialsSet(c) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Pairs:     []currency.Pair{testPair},
		Side:      order.AnySide,
	}

	_, err := c.GetOrderHistory(t.Context(), &getOrdersRequest)
	if sharedtestvalues.AreAPICredentialsSet(c) && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !sharedtestvalues.AreAPICredentialsSet(c) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}

	getOrdersRequest.Pairs = []currency.Pair{}
	_, err = c.GetOrderHistory(t.Context(), &getOrdersRequest)
	if sharedtestvalues.AreAPICredentialsSet(c) && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !sharedtestvalues.AreAPICredentialsSet(c) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}

	getOrdersRequest.Pairs = nil
	_, err = c.GetOrderHistory(t.Context(), &getOrdersRequest)
	if sharedtestvalues.AreAPICredentialsSet(c) && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !sharedtestvalues.AreAPICredentialsSet(c) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, c, canManipulateRealOrders)

	// limit order
	orderSubmission := &order.Submit{
		Exchange: c.Name,
		Pair: currency.Pair{
			Delimiter: "-",
			Base:      currency.BTC,
			Quote:     currency.USD,
		},
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    0.001,
		ClientID:  "meowOrder",
		AssetType: asset.Spot,
	}
	response, err := c.SubmitOrder(t.Context(), orderSubmission)
	if sharedtestvalues.AreAPICredentialsSet(c) && (err != nil || response.Status != order.New) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !sharedtestvalues.AreAPICredentialsSet(c) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}

	// market order from amount
	orderSubmission = &order.Submit{
		Exchange: c.Name,
		Pair: currency.Pair{
			Delimiter: "-",
			Base:      currency.BTC,
			Quote:     currency.USD,
		},
		Side:      order.Buy,
		Type:      order.Market,
		Amount:    0.001,
		ClientID:  "meowOrder",
		AssetType: asset.Spot,
	}
	response, err = c.SubmitOrder(t.Context(), orderSubmission)
	if sharedtestvalues.AreAPICredentialsSet(c) && (err != nil || response.Status != order.New) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !sharedtestvalues.AreAPICredentialsSet(c) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}

	// market order from quote amount
	orderSubmission = &order.Submit{
		Exchange: c.Name,
		Pair: currency.Pair{
			Delimiter: "-",
			Base:      currency.BTC,
			Quote:     currency.USD,
		},
		Side:        order.Buy,
		Type:        order.Market,
		QuoteAmount: 1,
		ClientID:    "meowOrder",
		AssetType:   asset.Spot,
	}
	response, err = c.SubmitOrder(t.Context(), orderSubmission)
	if sharedtestvalues.AreAPICredentialsSet(c) && (err != nil || response.Status != order.New) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !sharedtestvalues.AreAPICredentialsSet(c) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, c, canManipulateRealOrders)

	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      testPair,
		AssetType: asset.Spot,
	}

	err := c.CancelOrder(t.Context(), orderCancellation)
	if !sharedtestvalues.AreAPICredentialsSet(c) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(c) && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, c, canManipulateRealOrders)

	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      testPair,
		AssetType: asset.Spot,
	}

	resp, err := c.CancelAllOrders(t.Context(), orderCancellation)

	if !sharedtestvalues.AreAPICredentialsSet(c) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(c) && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}

	if len(resp.Status) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.Status))
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, c, canManipulateRealOrders)

	_, err := c.ModifyOrder(t.Context(),
		&order.Modify{AssetType: asset.Spot})
	if err == nil {
		t.Error("ModifyOrder() Expected error")
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, c, canManipulateRealOrders)

	withdrawCryptoRequest := withdraw.Request{
		Exchange:    c.Name,
		Amount:      -1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}

	_, err := c.WithdrawCryptocurrencyFunds(t.Context(),
		&withdrawCryptoRequest)
	if !sharedtestvalues.AreAPICredentialsSet(c) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(c) && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, c, canManipulateRealOrders)

	withdrawFiatRequest := withdraw.Request{
		Amount:   100,
		Currency: currency.USD,
		Fiat: withdraw.FiatRequest{
			Bank: banking.Account{
				BankName: "Federal Reserve Bank",
			},
		},
	}

	_, err := c.WithdrawFiatFunds(t.Context(), &withdrawFiatRequest)
	if !sharedtestvalues.AreAPICredentialsSet(c) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(c) && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, c, canManipulateRealOrders)

	withdrawFiatRequest := withdraw.Request{
		Amount:   100,
		Currency: currency.USD,
		Fiat: withdraw.FiatRequest{
			Bank: banking.Account{
				BankName: "Federal Reserve Bank",
			},
		},
	}

	_, err := c.WithdrawFiatFundsToInternationalBank(t.Context(),
		&withdrawFiatRequest)
	if !sharedtestvalues.AreAPICredentialsSet(c) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(c) && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	_, err := c.GetDepositAddress(t.Context(), currency.BTC, "", "")
	if err == nil {
		t.Error("GetDepositAddress() error", err)
	}
}

// TestWsAuth dials websocket, sends login request.
func TestWsAuth(t *testing.T) {
	if !c.Websocket.IsEnabled() && !c.API.AuthenticatedWebsocketSupport || !sharedtestvalues.AreAPICredentialsSet(c) {
		t.Skip(websocket.ErrWebsocketNotEnabled.Error())
	}
	var dialer gws.Dialer
	err := c.Websocket.Conn.Dial(t.Context(), &dialer, http.Header{})
	require.NoError(t, err, "Dial must not error")
	go c.wsReadData(t.Context())

	err = c.Subscribe(subscription.List{{Channel: "user", Pairs: currency.Pairs{testPair}}})
	require.NoError(t, err, "Subscribe must not error")
	timer := time.NewTimer(sharedtestvalues.WebsocketResponseDefaultTimeout)
	select {
	case badResponse := <-c.Websocket.DataHandler:
		t.Error(badResponse)
	case <-timer.C:
	}
	timer.Stop()
}

func TestWsSubscribe(t *testing.T) {
	pressXToJSON := []byte(`{
		"type": "subscriptions",
		"channels": [
			{
				"name": "level2",
				"product_ids": [
					"ETH-USD",
					"ETH-EUR"
				]
			},
			{
				"name": "heartbeat",
				"product_ids": [
					"ETH-USD",
					"ETH-EUR"
				]
			},
			{
				"name": "ticker",
				"product_ids": [
					"ETH-USD",
					"ETH-EUR",
					"ETH-BTC"
				]
			}
		]
	}`)
	err := c.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsHeartbeat(t *testing.T) {
	pressXToJSON := []byte(`{
		"type": "heartbeat",
		"sequence": 90,
		"last_trade_id": 20,
		"product_id": "BTC-USD",
		"time": "2014-11-07T08:19:28.464459Z"
	}`)
	err := c.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsStatus(t *testing.T) {
	pressXToJSON := []byte(`{
    "type": "status",
    "products": [
        {
            "id": "BTC-USD",
            "base_currency": "BTC",
            "quote_currency": "USD",
            "base_min_size": "0.001",
            "base_max_size": "70",
            "base_increment": "0.00000001",
            "quote_increment": "0.01",
            "display_name": "BTC/USD",
            "status": "online",
            "status_message": null,
            "min_market_funds": "10",
            "max_market_funds": "1000000",
            "post_only": false,
            "limit_only": false,
            "cancel_only": false
        }
    ],
    "currencies": [
        {
            "id": "USD",
            "name": "United States Dollar",
            "min_size": "0.01000000",
            "status": "online",
            "status_message": null,
            "max_precision": "0.01",
            "convertible_to": ["USDC"], "details": {}
        },
        {
            "id": "USDC",
            "name": "USD Coin",
            "min_size": "0.00000100",
            "status": "online",
            "status_message": null,
            "max_precision": "0.000001",
            "convertible_to": ["USD"], "details": {}
        },
        {
            "id": "BTC",
            "name": "Bitcoin",
            "min_size": "0.00000001",
            "status": "online",
            "status_message": null,
            "max_precision": "0.00000001",
            "convertible_to": []
        }
    ]
}`)
	err := c.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsTicker(t *testing.T) {
	pressXToJSON := []byte(`{
    "type": "ticker",
    "trade_id": 20153558,
    "sequence": 3262786978,
    "time": "2017-09-02T17:05:49.250000Z",
    "product_id": "BTC-USD",
    "price": "4388.01000000",
    "side": "buy", 
    "last_size": "0.03000000",
    "best_bid": "4388",
    "best_ask": "4388.01"
}`)
	err := c.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsOrderbook(t *testing.T) {
	pressXToJSON := []byte(`{
    "type": "snapshot",
    "product_id": "BTC-USD",
    "bids": [["10101.10", "0.45054140"]],
    "asks": [["10102.55", "0.57753524"]],
	"time":"2023-08-15T06:46:55.376250Z"
}`)
	err := c.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`{
  "type": "l2update",
  "product_id": "BTC-USD",
  "time": "2023-08-15T06:46:57.933713Z",
  "changes": [
    [
      "buy",
      "10101.80000000",
      "0.162567"
    ]
  ]
}`)
	err = c.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsOrders(t *testing.T) {
	pressXToJSON := []byte(`{
    "type": "received",
    "time": "2014-11-07T08:19:27.028459Z",
    "product_id": "BTC-USD",
    "sequence": 10,
    "order_id": "d50ec984-77a8-460a-b958-66f114b0de9b",
    "size": "1.34",
    "price": "502.1",
    "side": "buy",
    "order_type": "limit"
}`)
	err := c.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`{
    "type": "received",
    "time": "2014-11-09T08:19:27.028459Z",
    "product_id": "BTC-USD",
    "sequence": 12,
    "order_id": "dddec984-77a8-460a-b958-66f114b0de9b",
    "funds": "3000.234",
    "side": "buy",
    "order_type": "market"
}`)
	err = c.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`{
    "type": "open",
    "time": "2014-11-07T08:19:27.028459Z",
    "product_id": "BTC-USD",
    "sequence": 10,
    "order_id": "d50ec984-77a8-460a-b958-66f114b0de9b",
    "price": "200.2",
    "remaining_size": "1.00",
    "side": "sell"
}`)
	err = c.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`{
    "type": "done",
    "time": "2014-11-07T08:19:27.028459Z",
    "product_id": "BTC-USD",
    "sequence": 10,
    "price": "200.2",
    "order_id": "d50ec984-77a8-460a-b958-66f114b0de9b",
    "reason": "filled", 
    "side": "sell",
    "remaining_size": "0"
}`)
	err = c.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`{
    "type": "match",
    "trade_id": 10,
    "sequence": 50,
    "maker_order_id": "ac928c66-ca53-498f-9c13-a110027a60e8",
    "taker_order_id": "132fb6ae-456b-4654-b4e0-d681ac05cea1",
    "time": "2014-11-07T08:19:27.028459Z",
    "product_id": "BTC-USD",
    "size": "5.23512",
    "price": "400.23",
    "side": "sell"
}`)
	err = c.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`{
    "type": "change",
    "time": "2014-11-07T08:19:27.028459Z",
    "sequence": 80,
    "order_id": "ac928c66-ca53-498f-9c13-a110027a60e8",
    "product_id": "BTC-USD",
    "new_size": "5.23512",
    "old_size": "12.234412",
    "price": "400.23",
    "side": "sell"
}`)
	err = c.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
	pressXToJSON = []byte(`{
    "type": "change",
    "time": "2014-11-07T08:19:27.028459Z",
    "sequence": 80,
    "order_id": "ac928c66-ca53-498f-9c13-a110027a60e8",
    "product_id": "BTC-USD",
    "new_funds": "5.23512",
    "old_funds": "12.234412",
    "price": "400.23",
    "side": "sell"
}`)
	err = c.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
	pressXToJSON = []byte(`{
  "type": "activate",
  "product_id": "BTC-USD",
  "timestamp": "1483736448.299000",
  "user_id": "12",
  "profile_id": "30000727-d308-cf50-7b1c-c06deb1934fc",
  "order_id": "7b52009b-64fd-0a2a-49e6-d8a939753077",
  "stop_type": "entry",
  "side": "buy",
  "stop_price": "80",
  "size": "2",
  "funds": "50",
  "taker_fee_rate": "0.0025",
  "private": true
}`)
	err = c.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestStatusToStandardStatus(t *testing.T) {
	type TestCases struct {
		Case   string
		Result order.Status
	}
	testCases := []TestCases{
		{Case: "received", Result: order.New},
		{Case: "open", Result: order.Active},
		{Case: "done", Result: order.Filled},
		{Case: "match", Result: order.PartiallyFilled},
		{Case: "change", Result: order.Active},
		{Case: "activate", Result: order.Active},
		{Case: "LOL", Result: order.UnknownStatus},
	}
	for i := range testCases {
		result, _ := statusToStandardStatus(testCases[i].Case)
		if result != testCases[i].Result {
			t.Errorf("Expected: %v, received: %v", testCases[i].Result, result)
		}
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := c.GetRecentTrades(t.Context(), testPair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := c.GetHistoricTrades(t.Context(),
		testPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	if err != nil && err != common.ErrFunctionNotSupported {
		t.Error(err)
	}
}

func TestGetTransfers(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetTransfers(t.Context(), "", "", 100, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, c)
	for _, a := range c.GetAssetTypes(false) {
		pairs, err := c.CurrencyPairs.GetPairs(a, false)
		require.NoErrorf(t, err, "cannot get pairs for %s", a)
		require.NotEmptyf(t, pairs, "no pairs for %s", a)
		resp, err := c.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}
