package poloniex

import (
	"net/http"
	"strings"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own APIKEYS here for due diligence testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var testPair = currency.NewPair(currency.BTC, currency.LTC)

var e *Exchange

func TestTimestamp(t *testing.T) {
	t.Parallel()
	_, err := e.GetTimestamp(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetTicker(t.Context())
	if err != nil {
		t.Error("Poloniex GetTicker() error", err)
	}
}

func TestGetVolume(t *testing.T) {
	t.Parallel()
	_, err := e.GetVolume(t.Context())
	if err != nil {
		t.Error("Test failed - Poloniex GetVolume() error")
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderbook(t.Context(), "BTC_XMR", 50)
	if err != nil {
		t.Error("Test failed - Poloniex GetOrderbook() error", err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradeHistory(t.Context(), "BTC_XMR", 0, 0)
	if err != nil {
		t.Error("Test failed - Poloniex GetTradeHistory() error", err)
	}
}

func TestGetChartData(t *testing.T) {
	t.Parallel()
	_, err := e.GetChartData(t.Context(),
		"BTC_XMR",
		time.Unix(1405699200, 0), time.Unix(1405699400, 0), "300")
	if err != nil {
		t.Error("Test failed - Poloniex GetChartData() error", err)
	}
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrencies(t.Context())
	if err != nil {
		t.Error("Test failed - Poloniex GetCurrencies() error", err)
	}
}

func TestGetLoanOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetLoanOrders(t.Context(), "BTC")
	if err != nil {
		t.Error("Test failed - Poloniex GetLoanOrders() error", err)
	}
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:  1,
		FeeType: exchange.CryptocurrencyTradeFee,
		Pair: currency.NewPairWithDelimiter(currency.LTC.String(),
			currency.BTC.String(),
			"-"),
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	t.Parallel()

	feeBuilder := setFeeBuilder()
	_, err := e.GetFeeByType(t.Context(), feeBuilder)
	if err != nil {
		t.Fatal(err)
	}
	if !sharedtestvalues.AreAPICredentialsSet(e) {
		if feeBuilder.FeeType != exchange.OfflineTradeFee {
			t.Errorf("Expected %v, received %v",
				exchange.OfflineTradeFee,
				feeBuilder.FeeType)
		}
	} else {
		if feeBuilder.FeeType != exchange.CryptocurrencyTradeFee {
			t.Errorf("Expected %v, received %v",
				exchange.CryptocurrencyTradeFee,
				feeBuilder.FeeType)
		}
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	feeBuilder := setFeeBuilder()

	if sharedtestvalues.AreAPICredentialsSet(e) || mockTests {
		// CryptocurrencyTradeFee Basic
		if _, err := e.GetFee(t.Context(), feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if _, err := e.GetFee(t.Context(), feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if _, err := e.GetFee(t.Context(), feeBuilder); err != nil {
			t.Error(err)
		}
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := e.GetFee(t.Context(), feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := e.GetFee(t.Context(), feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	if _, err := e.GetFee(t.Context(), feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if _, err := e.GetFee(t.Context(), feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if _, err := e.GetFee(t.Context(), feeBuilder); err != nil {
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expectedResult := exchange.AutoWithdrawCryptoWithAPIPermissionText +
		" & " +
		exchange.NoFiatWithdrawalsText
	withdrawPermissions := e.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s",
			expectedResult,
			withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}

	_, err := e.GetActiveOrders(t.Context(), &getOrdersRequest)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(e) && err != nil:
		t.Error("GetActiveOrders() error", err)
	case !sharedtestvalues.AreAPICredentialsSet(e) && !mockTests && err == nil:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock GetActiveOrders() err", err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}

	_, err := e.GetOrderHistory(t.Context(), &getOrdersRequest)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(e) && err != nil:
		t.Errorf("Could not get order history: %s", err)
	case !sharedtestvalues.AreAPICredentialsSet(e) && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Errorf("Could not mock get order history: %s", err)
	}
}

func TestGetOrderStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mock           bool
		orderID        string
		errExpected    bool
		errMsgExpected string
	}{
		{
			name:           "correct order ID",
			mock:           true,
			orderID:        "96238912841",
			errExpected:    false,
			errMsgExpected: "",
		},
		{
			name:           "wrong order ID",
			mock:           true,
			orderID:        "96238912842",
			errExpected:    true,
			errMsgExpected: "Order not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.mock != mockTests {
				t.Skip("mock mismatch, skipping")
			}

			_, err := e.GetAuthenticatedOrderStatus(t.Context(),
				tt.orderID)
			switch {
			case sharedtestvalues.AreAPICredentialsSet(e) && err != nil:
				t.Errorf("Could not get order status: %s", err)
			case !sharedtestvalues.AreAPICredentialsSet(e) && err == nil && !mockTests:
				t.Error("Expecting an error when no keys are set")
			case mockTests && err != nil:
				if !tt.errExpected {
					t.Errorf("Could not mock get order status: %s", err.Error())
				} else if !(strings.Contains(err.Error(), tt.errMsgExpected)) {
					t.Errorf("Could not mock get order status: %s", err.Error())
				}
			case mockTests:
				if tt.errExpected {
					t.Errorf("Mock get order status expect an error %q, get no error", tt.errMsgExpected)
				}
			}
		})
	}
}

func TestGetOrderTrades(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mock           bool
		orderID        string
		errExpected    bool
		errMsgExpected string
	}{
		{
			name:           "correct order ID",
			mock:           true,
			orderID:        "96238912841",
			errExpected:    false,
			errMsgExpected: "",
		},
		{
			name:           "wrong order ID",
			mock:           true,
			orderID:        "96238912842",
			errExpected:    true,
			errMsgExpected: "Order not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.mock != mockTests {
				t.Skip("mock mismatch, skipping")
			}

			_, err := e.GetAuthenticatedOrderTrades(t.Context(), tt.orderID)
			switch {
			case sharedtestvalues.AreAPICredentialsSet(e) && err != nil:
				t.Errorf("Could not get order trades: %s", err)
			case !sharedtestvalues.AreAPICredentialsSet(e) && err == nil && !mockTests:
				t.Error("Expecting an error when no keys are set")
			case mockTests && err != nil:
				assert.ErrorContains(t, err, tt.errMsgExpected)
			}
		})
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
	}

	orderSubmission := &order.Submit{
		Exchange: e.Name,
		Pair: currency.Pair{
			Delimiter: currency.UnderscoreDelimiter,
			Base:      currency.BTC,
			Quote:     currency.LTC,
		},
		Side:      order.Buy,
		Type:      order.Market,
		Price:     10,
		Amount:    10000000,
		ClientID:  "hi",
		AssetType: asset.Spot,
	}

	response, err := e.SubmitOrder(t.Context(), orderSubmission)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(e) && (err != nil || response.Status != order.Filled):
		t.Errorf("Order failed to be placed: %v", err)
	case !sharedtestvalues.AreAPICredentialsSet(e) && !mockTests && err == nil:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock SubmitOrder() err", err)
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
	}
	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      currency.NewPair(currency.LTC, currency.BTC),
		AssetType: asset.Spot,
	}

	err := e.CancelOrder(t.Context(), orderCancellation)
	switch {
	case !sharedtestvalues.AreAPICredentialsSet(e) && !mockTests && err == nil:
		t.Error("Expecting an error when no keys are set")
	case sharedtestvalues.AreAPICredentialsSet(e) && err != nil:
		t.Errorf("Could not cancel orders: %v", err)
	case mockTests && err != nil:
		t.Error("Mock CancelExchangeOrder() err", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      currencyPair,
		AssetType: asset.Spot,
	}

	resp, err := e.CancelAllOrders(t.Context(), orderCancellation)
	switch {
	case !sharedtestvalues.AreAPICredentialsSet(e) && !mockTests && err == nil:
		t.Error("Expecting an error when no keys are set")
	case sharedtestvalues.AreAPICredentialsSet(e) && err != nil:
		t.Errorf("Could not cancel orders: %v", err)
	case mockTests && err != nil:
		t.Error("Mock CancelAllExchangeOrders() err", err)
	}
	if len(resp.Status) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.Status))
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
	}

	_, err := e.ModifyOrder(t.Context(), &order.Modify{
		OrderID:   "1337",
		Price:     1337,
		AssetType: asset.Spot,
		Pair:      currency.NewBTCUSDT(),
	})
	switch {
	case sharedtestvalues.AreAPICredentialsSet(e) && err != nil && mockTests:
		t.Error("ModifyOrder() error", err)
	case !sharedtestvalues.AreAPICredentialsSet(e) && !mockTests && err == nil:
		t.Error("ModifyOrder() error cannot be nil")
	case mockTests && err != nil:
		t.Error("Mock ModifyOrder() err", err)
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	withdrawCryptoRequest := withdraw.Request{
		Exchange: e.Name,
		Crypto: withdraw.CryptoRequest{
			Address:   core.BitcoinDonationAddress,
			FeeAmount: 0,
		},
		Amount:        -1,
		Currency:      currency.LTC,
		Description:   "WITHDRAW IT ALL",
		TradePassword: "Password",
	}
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
	}

	_, err := e.WithdrawCryptocurrencyFunds(t.Context(),
		&withdrawCryptoRequest)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(e) && err != nil:
		t.Errorf("Withdraw failed to be placed: %v", err)
	case !sharedtestvalues.AreAPICredentialsSet(e) && !mockTests && err == nil:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err == nil:
		t.Error("should error due to invalid amount")
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
	}

	var withdrawFiatRequest withdraw.Request
	_, err := e.WithdrawFiatFunds(t.Context(), &withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'",
			common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
	}

	var withdrawFiatRequest withdraw.Request
	_, err := e.WithdrawFiatFundsToInternationalBank(t.Context(),
		&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'",
			common.ErrFunctionNotSupported, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddress(t.Context(), currency.USDT, "", "USDTETH")
	switch {
	case sharedtestvalues.AreAPICredentialsSet(e) && err != nil:
		t.Error("GetDepositAddress()", err)
	case !sharedtestvalues.AreAPICredentialsSet(e) && !mockTests && err == nil:
		t.Error("GetDepositAddress() cannot be nil")
	case mockTests && err != nil:
		t.Error("Mock GetDepositAddress() err", err)
	}
}

func TestGenerateNewAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.GenerateNewAddress(t.Context(), currency.XRP.String())
	if err != nil {
		t.Fatal(err)
	}
}

// TestWsAuth receives a message only on failure
func TestWsAuth(t *testing.T) {
	t.Parallel()
	if !e.Websocket.IsEnabled() && !e.API.AuthenticatedWebsocketSupport || !sharedtestvalues.AreAPICredentialsSet(e) {
		t.Skip(websocket.ErrWebsocketNotEnabled.Error())
	}
	var dialer gws.Dialer
	err := e.Websocket.Conn.Dial(t.Context(), &dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	go e.wsReadData(t.Context())
	creds, err := e.GetCredentials(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	err = e.wsSendAuthorisedCommand(t.Context(), creds.Secret, creds.Key, "subscribe")
	if err != nil {
		t.Fatal(err)
	}
	timer := time.NewTimer(sharedtestvalues.WebsocketResponseDefaultTimeout)
	select {
	case response := <-e.Websocket.DataHandler.C:
		t.Error(response)
	case <-timer.C:
	}
	timer.Stop()
}

func TestWsSubAck(t *testing.T) {
	pressXToJSON := []byte(`[1002, 1]`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsTicker(t *testing.T) {
	err := e.loadCurrencyDetails(t.Context())
	if err != nil {
		t.Error(err)
	}
	pressXToJSON := []byte(`[1002, null, [ 50, "382.98901522", "381.99755898", "379.41296309", "-0.04312950", "14969820.94951828", "38859.58435407", 0, "412.25844455", "364.56122072" ] ]`)
	err = e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsExchangeVolume(t *testing.T) {
	err := e.loadCurrencyDetails(t.Context())
	if err != nil {
		t.Error(err)
	}
	pressXToJSON := []byte(`[1003,null,["2018-11-07 16:26",5804,{"BTC":"3418.409","ETH":"2645.921","USDT":"10832502.689","USDC":"1578020.908"}]]`)
	err = e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsTrades(t *testing.T) {
	e.SetSaveTradeDataStatus(true)
	err := e.loadCurrencyDetails(t.Context())
	if err != nil {
		t.Error(err)
	}
	pressXToJSON := []byte(`[14, 8768, [["t", "42706057", 1, "0.05567134", "0.00181421", 1522877119]]]`)
	err = e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsPriceAggregateOrderbook(t *testing.T) {
	err := e.loadCurrencyDetails(t.Context())
	if err != nil {
		t.Error(err)
	}
	pressXToJSON := []byte(`[50,141160924,[["i",{"currencyPair":"BTC_LTC","orderBook":[{"0.002784":"17.55","0.002786":"1.47","0.002792":"13.25","0.0028":"0.21","0.002804":"0.02","0.00281":"1.5","0.002811":"258.82","0.002812":"3.81","0.002817":"0.06","0.002824":"3","0.002825":"0.02","0.002836":"18.01","0.002837":"0.03","0.00284":"0.03","0.002842":"12.7","0.00285":"0.02","0.002852":"0.02","0.002855":"1.3","0.002857":"15.64","0.002864":"0.01"},{"0.002782":"45.93","0.002781":"1.46","0.002774":"13.34","0.002773":"0.04","0.002771":"0.05","0.002765":"6.21","0.002764":"3","0.00276":"10.77","0.002758":"3.11","0.002754":"0.02","0.002751":"288.94","0.00275":"24.06","0.002745":"187.27","0.002743":"0.04","0.002742":"0.96","0.002731":"0.06","0.00273":"12.13","0.002727":"0.02","0.002725":"0.03","0.002719":"1.09"}]}, "1692080077892"]]]`)
	err = e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`[50,141160925,[["o",1,"0.002742","0", "1692080078806"],["o",1,"0.002718","0.02", "1692080078806"]]]`)
	err = e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()

	_, err := e.GetHistoricCandles(t.Context(), testPair, asset.Spot, kline.FiveMin, time.Unix(1588741402, 0), time.Unix(1588745003, 0))
	assert.NoError(t, err)
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()

	_, err := e.GetHistoricCandlesExtended(t.Context(), testPair, asset.Spot, kline.FiveMin, time.Unix(1588741402, 0), time.Unix(1588745003, 0))
	assert.NoError(t, err)
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip("relies on time.Now()")
	}
	_, err := e.GetRecentTrades(t.Context(), currency.NewPair(currency.BTC, currency.XMR), asset.Spot)
	assert.NoError(t, err)
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()

	tStart := time.Date(2020, 6, 6, 0, 0, 0, 0, time.UTC)
	tEnd := time.Date(2020, 6, 6, 1, 0, 0, 0, time.UTC)
	if !mockTests {
		tmNow := time.Now()
		tStart = time.Date(tmNow.Year(), tmNow.Month()-3, 6, 0, 0, 0, 0, time.UTC)
		tEnd = time.Date(tmNow.Year(), tmNow.Month()-3, 7, 0, 0, 0, 0, time.UTC)
	}
	_, err := e.GetHistoricTrades(t.Context(), currency.NewPair(currency.BTC, currency.XMR), asset.Spot, tStart, tEnd)
	assert.NoError(t, err)
}

func TestProcessAccountMarginPosition(t *testing.T) {
	err := e.loadCurrencyDetails(t.Context())
	if err != nil {
		t.Error(err)
	}

	margin := []byte(`[1000,"",[["m", 23432933, 28, "-0.06000000"]]]`)
	err = e.wsHandleData(t.Context(), margin)
	require.ErrorIs(t, err, errNotEnoughData)

	margin = []byte(`[1000,"",[["m", "23432933", 28, "-0.06000000", null]]]`)
	err = e.wsHandleData(t.Context(), margin)
	require.ErrorIs(t, err, errTypeAssertionFailure)

	margin = []byte(`[1000,"",[["m", 23432933, "28", "-0.06000000", null]]]`)
	err = e.wsHandleData(t.Context(), margin)
	require.ErrorIs(t, err, errTypeAssertionFailure)

	margin = []byte(`[1000,"",[["m", 23432933, 28, -0.06000000, null]]]`)
	err = e.wsHandleData(t.Context(), margin)
	require.ErrorIs(t, err, errTypeAssertionFailure)

	margin = []byte(`[1000,"",[["m", 23432933, 28, "-0.06000000", null]]]`)
	err = e.wsHandleData(t.Context(), margin)
	if err != nil {
		t.Fatal(err)
	}
}

func TestProcessAccountPendingOrder(t *testing.T) {
	err := e.loadCurrencyDetails(t.Context())
	if err != nil {
		t.Error(err)
	}

	pending := []byte(`[1000,"",[["p",431682155857,127,"1000.00000000","1.00000000","0"]]]`)
	err = e.wsHandleData(t.Context(), pending)
	require.ErrorIs(t, err, errNotEnoughData)

	pending = []byte(`[1000,"",[["p","431682155857",127,"1000.00000000","1.00000000","0",null]]]`)
	err = e.wsHandleData(t.Context(), pending)
	require.ErrorIs(t, err, errTypeAssertionFailure)

	pending = []byte(`[1000,"",[["p",431682155857,"127","1000.00000000","1.00000000","0",null]]]`)
	err = e.wsHandleData(t.Context(), pending)
	require.ErrorIs(t, err, errTypeAssertionFailure)

	pending = []byte(`[1000,"",[["p",431682155857,127,1000.00000000,"1.00000000","0",null]]]`)
	err = e.wsHandleData(t.Context(), pending)
	require.ErrorIs(t, err, errTypeAssertionFailure)

	pending = []byte(`[1000,"",[["p",431682155857,127,"1000.00000000",1.00000000,"0",null]]]`)
	err = e.wsHandleData(t.Context(), pending)
	require.ErrorIs(t, err, errTypeAssertionFailure)

	pending = []byte(`[1000,"",[["p",431682155857,127,"1000.00000000","1.00000000",0,null]]]`)
	err = e.wsHandleData(t.Context(), pending)
	require.ErrorIs(t, err, errTypeAssertionFailure)

	pending = []byte(`[1000,"",[["p",431682155857,127,"1000.00000000","1.00000000","0",null]]]`)
	err = e.wsHandleData(t.Context(), pending)
	if err != nil {
		t.Fatal(err)
	}

	// Unmatched pair in system
	pending = []byte(`[1000,"",[["p",431682155857,666,"1000.00000000","1.00000000","0",null]]]`)
	err = e.wsHandleData(t.Context(), pending)
	if err != nil {
		t.Fatal(err)
	}
}

func TestProcessAccountOrderUpdate(t *testing.T) {
	orderUpdate := []byte(`[1000,"",[["o",431682155857,"0.00000000","f"]]]`)
	err := e.wsHandleData(t.Context(), orderUpdate)
	require.ErrorIs(t, err, errNotEnoughData)

	orderUpdate = []byte(`[1000,"",[["o","431682155857","0.00000000","f",null]]]`)
	err = e.wsHandleData(t.Context(), orderUpdate)
	require.ErrorIs(t, err, errTypeAssertionFailure)

	orderUpdate = []byte(`[1000,"",[["o",431682155857,0.00000000,"f",null]]]`)
	err = e.wsHandleData(t.Context(), orderUpdate)
	require.ErrorIs(t, err, errTypeAssertionFailure)

	orderUpdate = []byte(`[1000,"",[["o",431682155857,"0.00000000",123,null]]]`)
	err = e.wsHandleData(t.Context(), orderUpdate)
	require.ErrorIs(t, err, errTypeAssertionFailure)

	orderUpdate = []byte(`[1000,"",[["o",431682155857,"0.00000000","c",null]]]`)
	err = e.wsHandleData(t.Context(), orderUpdate)
	require.ErrorIs(t, err, errNotEnoughData)

	orderUpdate = []byte(`[1000,"",[["o",431682155857,"0.50000000","c",null,"0.50000000"]]]`)
	err = e.wsHandleData(t.Context(), orderUpdate)
	if err != nil {
		t.Fatal(err)
	}

	orderUpdate = []byte(`[1000,"",[["o",431682155857,"0.00000000","c",null,"1.00000000"]]]`)
	err = e.wsHandleData(t.Context(), orderUpdate)
	if err != nil {
		t.Fatal(err)
	}

	orderUpdate = []byte(`[1000,"",[["o",431682155857,"0.50000000","f",null]]]`)
	err = e.wsHandleData(t.Context(), orderUpdate)
	if err != nil {
		t.Fatal(err)
	}

	orderUpdate = []byte(`[1000,"",[["o",431682155857,"0.00000000","s",null]]]`)
	err = e.wsHandleData(t.Context(), orderUpdate)
	if err != nil {
		t.Fatal(err)
	}
}

func TestProcessAccountOrderLimit(t *testing.T) {
	err := e.loadCurrencyDetails(t.Context())
	if err != nil {
		t.Error(err)
	}

	accountTrade := []byte(`[1000,"",[["n",127,431682155857,"0","1000.00000000","1.00000000","2021-04-13 07:19:56","1.00000000"]]]`)
	err = e.wsHandleData(t.Context(), accountTrade)
	require.ErrorIs(t, err, errNotEnoughData)

	accountTrade = []byte(`[1000,"",[["n","127",431682155857,"0","1000.00000000","1.00000000","2021-04-13 07:19:56","1.00000000",null]]]`)
	err = e.wsHandleData(t.Context(), accountTrade)
	require.ErrorIs(t, err, errTypeAssertionFailure)

	accountTrade = []byte(`[1000,"",[["n",127,"431682155857","0","1000.00000000","1.00000000","2021-04-13 07:19:56","1.00000000",null]]]`)
	err = e.wsHandleData(t.Context(), accountTrade)
	require.ErrorIs(t, err, errTypeAssertionFailure)

	accountTrade = []byte(`[1000,"",[["n",127,431682155857,0,"1000.00000000","1.00000000","2021-04-13 07:19:56","1.00000000",null]]]`)
	err = e.wsHandleData(t.Context(), accountTrade)
	require.ErrorIs(t, err, errTypeAssertionFailure)

	accountTrade = []byte(`[1000,"",[["n",127,431682155857,"0",1000.00000000,"1.00000000","2021-04-13 07:19:56","1.00000000",null]]]`)
	err = e.wsHandleData(t.Context(), accountTrade)
	require.ErrorIs(t, err, errTypeAssertionFailure)

	accountTrade = []byte(`[1000,"",[["n",127,431682155857,"0","1000.00000000",1.00000000,"2021-04-13 07:19:56","1.00000000",null]]]`)
	err = e.wsHandleData(t.Context(), accountTrade)
	require.ErrorIs(t, err, errTypeAssertionFailure)

	accountTrade = []byte(`[1000,"",[["n",127,431682155857,"0","1000.00000000","1.00000000",1234,"1.00000000",null]]]`)
	err = e.wsHandleData(t.Context(), accountTrade)
	require.ErrorIs(t, err, errTypeAssertionFailure)

	accountTrade = []byte(`[1000,"",[["n",127,431682155857,"0","1000.00000000","1.00000000","2021-04-13 07:19:56",1.00000000,null]]]`)
	err = e.wsHandleData(t.Context(), accountTrade)
	require.ErrorIs(t, err, errTypeAssertionFailure)

	accountTrade = []byte(`[1000,"",[["n",127,431682155857,"0","1000.00000000","1.00000000","2021-04-13 07:19:56","1.00000000",null]]]`)
	err = e.wsHandleData(t.Context(), accountTrade)
	if err != nil {
		t.Fatal(err)
	}
}

func TestProcessAccountBalanceUpdate(t *testing.T) {
	err := e.loadCurrencyDetails(t.Context())
	if err != nil {
		t.Error(err)
	}

	balance := []byte(`[1000,"",[["b",243,"e"]]]`)
	err = e.wsHandleData(t.Context(), balance)
	require.ErrorIs(t, err, errNotEnoughData)

	balance = []byte(`[1000,"",[["b","243","e","-1.00000000"]]]`)
	err = e.wsHandleData(t.Context(), balance)
	require.ErrorIs(t, err, errTypeAssertionFailure)

	balance = []byte(`[1000,"",[["b",243,1234,"-1.00000000"]]]`)
	err = e.wsHandleData(t.Context(), balance)
	require.ErrorIs(t, err, errTypeAssertionFailure)

	balance = []byte(`[1000,"",[["b",243,"e",-1.00000000]]]`)
	err = e.wsHandleData(t.Context(), balance)
	require.ErrorIs(t, err, errTypeAssertionFailure)

	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "test", Secret: "test"})
	balance = []byte(`[1000,"",[["b",243,"e","-1.00000000"]]]`)
	err = e.wsHandleData(ctx, balance)
	require.NoError(t, err, "wsHandleData must not error")
}

func TestProcessAccountTrades(t *testing.T) {
	accountTrades := []byte(`[1000,"",[["t", 12345, "0.03000000", "0.50000000", "0.00250000", 0, 6083059, "0.00000375", "2018-09-08 05:54:09", "12345"]]]`)
	err := e.wsHandleData(t.Context(), accountTrades)
	require.ErrorIs(t, err, errNotEnoughData)

	accountTrades = []byte(`[1000,"",[["t", "12345", "0.03000000", "0.50000000", "0.00250000", 0, 6083059, "0.00000375", "2018-09-08 05:54:09", "12345", "0.015"]]]`)
	err = e.wsHandleData(t.Context(), accountTrades)
	require.ErrorIs(t, err, errTypeAssertionFailure)

	accountTrades = []byte(`[1000,"",[["t", 12345, 0.03000000, "0.50000000", "0.00250000", 0, 6083059, "0.00000375", "2018-09-08 05:54:09", "12345", "0.015"]]]`)
	err = e.wsHandleData(t.Context(), accountTrades)
	require.ErrorIs(t, err, errTypeAssertionFailure)

	accountTrades = []byte(`[1000,"",[["t", 12345, "0.03000000", 0.50000000, "0.00250000", 0, 6083059, "0.00000375", "2018-09-08 05:54:09", "12345", "0.015"]]]`)
	err = e.wsHandleData(t.Context(), accountTrades)
	require.ErrorIs(t, err, errTypeAssertionFailure)

	accountTrades = []byte(`[1000,"",[["t", 12345, "0.03000000", "0.50000000", "0.00250000", 0, 6083059, 0.00000375, "2018-09-08 05:54:09", "12345", "0.015"]]]`)
	err = e.wsHandleData(t.Context(), accountTrades)
	require.ErrorIs(t, err, errTypeAssertionFailure)

	accountTrades = []byte(`[1000,"",[["t", 12345, "0.03000000", "0.50000000", "0.00250000", 0, 6083059, 0.0000037, "2018-09-08 05:54:09", "12345", "0.015"]]]`)
	err = e.wsHandleData(t.Context(), accountTrades)
	require.ErrorIs(t, err, errTypeAssertionFailure)

	accountTrades = []byte(`[1000,"",[["t", 12345, "0.03000000", "0.50000000", "0.00250000", 0, 6083059, "0.00000375", 12345, "12345", 0.015]]]`)
	err = e.wsHandleData(t.Context(), accountTrades)
	require.ErrorIs(t, err, errTypeAssertionFailure)

	accountTrades = []byte(`[1000,"",[["t", 12345, "0.03000000", "0.50000000", "0.00250000", 0, 6083059, "0.00000375", "2018-09-08 05:54:09", "12345", "0.015"]]]`)
	err = e.wsHandleData(t.Context(), accountTrades)
	if err != nil {
		t.Fatal(err)
	}
}

func TestProcessAccountKilledOrder(t *testing.T) {
	kill := []byte(`[1000,"",[["k", 1337]]]`)
	err := e.wsHandleData(t.Context(), kill)
	require.ErrorIs(t, err, errNotEnoughData)

	kill = []byte(`[1000,"",[["k", "1337", null]]]`)
	err = e.wsHandleData(t.Context(), kill)
	require.ErrorIs(t, err, errTypeAssertionFailure)

	kill = []byte(`[1000,"",[["k", 1337, null]]]`)
	err = e.wsHandleData(t.Context(), kill)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetCompleteBalances(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetCompleteBalances(t.Context())
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTicker(t.Context(), testPair, asset.Spot)
	assert.NoError(t, err)
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := e.UpdateTickers(t.Context(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	_, err := e.GetAvailableTransferChains(t.Context(), currency.USDT)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWalletActivity(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.WalletActivity(t.Context(), time.Now().Add(-time.Minute), time.Now(), "")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelMultipleOrdersByIDs(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.CancelMultipleOrdersByIDs(t.Context(), []string{"1234"}, []string{"5678"})
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetAccountFundingHistory(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelBatchOrders(t.Context(), []order.Cancel{
		{
			OrderID:   "1234",
			AssetType: asset.Spot,
			Pair:      currency.NewBTCUSD(),
		},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetTimestamp(t *testing.T) {
	t.Parallel()
	st, err := e.GetTimestamp(t.Context())
	require.NoError(t, err)

	if st.IsZero() {
		t.Error("expected a time")
	}
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	st, err := e.GetServerTime(t.Context(), asset.Spot)
	require.NoError(t, err)

	if st.IsZero() {
		t.Error("expected a time")
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := e.FetchTradablePairs(t.Context(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
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
