package poloniex

import (
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own APIKEYS here for due diligence testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var p Poloniex

func areTestAPIKeysSet() bool {
	return p.ValidateAPICredentials()
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := p.GetTicker()
	if err != nil {
		t.Error("Poloniex GetTicker() error", err)
	}
}

func TestGetVolume(t *testing.T) {
	t.Parallel()
	_, err := p.GetVolume()
	if err != nil {
		t.Error("Test faild - Poloniex GetVolume() error")
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := p.GetOrderbook("BTC_XMR", 50)
	if err != nil {
		t.Error("Test faild - Poloniex GetOrderbook() error", err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := p.GetTradeHistory("BTC_XMR", 0, 0)
	if err != nil {
		t.Error("Test faild - Poloniex GetTradeHistory() error", err)
	}
}

func TestGetChartData(t *testing.T) {
	t.Parallel()
	_, err := p.GetChartData("BTC_XMR",
		time.Unix(1405699200, 0), time.Unix(1405699400, 0), "300")
	if err != nil {
		t.Error("Test faild - Poloniex GetChartData() error", err)
	}
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	_, err := p.GetCurrencies()
	if err != nil {
		t.Error("Test faild - Poloniex GetCurrencies() error", err)
	}
}

func TestGetLoanOrders(t *testing.T) {
	t.Parallel()
	_, err := p.GetLoanOrders("BTC")
	if err != nil {
		t.Error("Test faild - Poloniex GetLoanOrders() error", err)
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

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	t.Parallel()

	var feeBuilder = setFeeBuilder()
	p.GetFeeByType(feeBuilder)
	if !areTestAPIKeysSet() {
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
	var feeBuilder = setFeeBuilder()

	if areTestAPIKeysSet() || mockTests {
		// CryptocurrencyTradeFee Basic
		if resp, err := p.GetFee(feeBuilder); resp != float64(0.0025) || err != nil {
			t.Error(err)
			t.Errorf("GetFee() error. Expected: %f, Received: %f",
				float64(0.0025), resp)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if resp, err := p.GetFee(feeBuilder); resp != float64(2500) || err != nil {
			t.Errorf("GetFee() error. Expected: %f, Received: %f",
				float64(2500), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if resp, err := p.GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Errorf("GetFee() error. Expected: %f, Received: %f",
				float64(0), resp)
			t.Error(err)
		}
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := p.GetFee(feeBuilder); resp != float64(0.001) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f",
			float64(0.001), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := p.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f",
			float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := p.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f",
			float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if resp, err := p.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f",
			float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if resp, err := p.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f",
			float64(0), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expectedResult := exchange.AutoWithdrawCryptoWithAPIPermissionText +
		" & " +
		exchange.NoFiatWithdrawalsText
	withdrawPermissions := p.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s",
			expectedResult,
			withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	var getOrdersRequest = order.GetOrdersRequest{
		Type: order.AnyType,
	}

	_, err := p.GetActiveOrders(&getOrdersRequest)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("GetActiveOrders() error", err)
	case !areTestAPIKeysSet() && !mockTests && err == nil:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock GetActiveOrders() err", err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	var getOrdersRequest = order.GetOrdersRequest{
		Type: order.AnyType,
	}

	_, err := p.GetOrderHistory(&getOrdersRequest)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Errorf("Could not get order history: %s", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Errorf("Could not mock get order history: %s", err)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var orderSubmission = &order.Submit{
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

	response, err := p.SubmitOrder(orderSubmission)
	switch {
	case areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced):
		t.Errorf("Order failed to be placed: %v", err)
	case !areTestAPIKeysSet() && !mockTests && err == nil:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock SubmitOrder() err", err)
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	var orderCancellation = &order.Cancel{
		ID:            "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currency.NewPair(currency.LTC, currency.BTC),
		AssetType:     asset.Spot,
	}

	err := p.CancelOrder(orderCancellation)
	switch {
	case !areTestAPIKeysSet() && !mockTests && err == nil:
		t.Error("Expecting an error when no keys are set")
	case areTestAPIKeysSet() && err != nil:
		t.Errorf("Could not cancel orders: %v", err)
	case mockTests && err != nil:
		t.Error("Mock CancelExchangeOrder() err", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	var orderCancellation = &order.Cancel{
		ID:            "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
		AssetType:     asset.Spot,
	}

	resp, err := p.CancelAllOrders(orderCancellation)
	switch {
	case !areTestAPIKeysSet() && !mockTests && err == nil:
		t.Error("Expecting an error when no keys are set")
	case areTestAPIKeysSet() && err != nil:
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
	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	_, err := p.ModifyOrder(&order.Modify{ID: "1337",
		Price:     1337,
		AssetType: asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USDT)})
	switch {
	case areTestAPIKeysSet() && err != nil && mockTests:
		t.Error("ModifyOrder() error", err)
	case !areTestAPIKeysSet() && !mockTests && err == nil:
		t.Error("ModifyOrder() error cannot be nil")
	case mockTests && err != nil:
		t.Error("Mock ModifyOrder() err", err)
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	withdrawCryptoRequest := withdraw.Request{
		Exchange: p.Name,
		Crypto: withdraw.CryptoRequest{
			Address:   core.BitcoinDonationAddress,
			FeeAmount: 1,
		},
		Amount:        0,
		Currency:      currency.LTC,
		Description:   "WITHDRAW IT ALL",
		TradePassword: "Password",
	}
	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	_, err := p.WithdrawCryptocurrencyFunds(&withdrawCryptoRequest)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Errorf("Withdraw failed to be placed: %v", err)
	case !areTestAPIKeysSet() && !mockTests && err == nil:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err == nil:
		t.Error("Mock Withdraw() err cannot be nil")
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest withdraw.Request
	_, err := p.WithdrawFiatFunds(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'",
			common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest withdraw.Request
	_, err := p.WithdrawFiatFundsToInternationalBank(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'",
			common.ErrFunctionNotSupported, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := p.GetDepositAddress(currency.DASH, "")
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("GetDepositAddress()", err)
	case !areTestAPIKeysSet() && !mockTests && err == nil:
		t.Error("GetDepositAddress() cannot be nil")
	case mockTests && err != nil:
		t.Error("Mock GetDepositAddress() err", err)
	}
}

// TestWsAuth dials websocket, sends login request.
// Will receive a message only on failure
func TestWsAuth(t *testing.T) {
	t.Parallel()
	if !p.Websocket.IsEnabled() && !p.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := p.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	go p.wsReadData()
	err = p.wsSendAuthorisedCommand("subscribe")
	if err != nil {
		t.Fatal(err)
	}
	timer := time.NewTimer(sharedtestvalues.WebsocketResponseDefaultTimeout)
	select {
	case response := <-p.Websocket.DataHandler:
		t.Error(response)
	case <-timer.C:
	}
	timer.Stop()
}

func TestWsSubAck(t *testing.T) {
	pressXToJSON := []byte(`[1002, 1]`)
	err := p.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsTicker(t *testing.T) {
	err := p.getCurrencyIDMap()
	if err != nil {
		t.Error(err)
	}
	pressXToJSON := []byte(`[1002, null, [ 50, "382.98901522", "381.99755898", "379.41296309", "-0.04312950", "14969820.94951828", "38859.58435407", 0, "412.25844455", "364.56122072" ] ]`)
	err = p.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsExchangeVolume(t *testing.T) {
	err := p.getCurrencyIDMap()
	if err != nil {
		t.Error(err)
	}
	pressXToJSON := []byte(`[1003,null,["2018-11-07 16:26",5804,{"BTC":"3418.409","ETH":"2645.921","USDT":"10832502.689","USDC":"1578020.908"}]]`)
	err = p.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsTrades(t *testing.T) {
	err := p.getCurrencyIDMap()
	if err != nil {
		t.Error(err)
	}
	pressXToJSON := []byte(`[14, 8768, [["t", "42706057", 1, "0.05567134", "0.00181421", 1522877119]]]`)
	err = p.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsPriceAggregateOrderbook(t *testing.T) {
	err := p.getCurrencyIDMap()
	if err != nil {
		t.Error(err)
	}
	pressXToJSON := []byte(`[148,827987828,[["i",{"currencyPair":"BTC_ETH","orderBook":[{"0.02311264":"2.20557811","1000.02022945":"1.00000000","1000.17618025":"0.00100000","1148.00000000":"0.04594689","1997.00000000":"2.00000000","2000.00000000":"0.00000206","3000.00000000":"0.00000137","3772.00000000":"0.65977073","4000.00000000":"0.00000103","5000.00000000":"0.10284089"},{"0.02310611":"21.20361406","0.00010000":"2052.10260000","0.00009726":"17.85554185","0.00009170":"10.00000000","0.00008800":"8.00000000","0.00008000":"2.02050000","0.00007186":"6.95811300","0.00006060":"130.00000000","0.00005126":"1070.00000000","0.00005120":"195.31250000","0.00005000":"2120.00000000","0.00004295":"202.34435389","0.00004168":"95.96928983","0.00004000":"200.00000000","0.00003638":"137.43815283","0.00003500":"114.28657143","0.00003492":"6.90074951","0.00003101":"500.00000000","0.00003100":"1000.00000000","0.00002560":"390.62500000","0.00002500":"20000.00000000","0.00002000":"55.00000000","0.00001280":"781.25000000","0.00001010":"50.00000000","0.00001005":"146.26965174","0.00001000":"12109.99999999","0.00000640":"1562.50000000","0.00000550":"800.00000000","0.00000500":"200.00000000","0.00000331":"1000.00000000","0.00000330":"11479.02727273","0.00000320":"3125.00000000","0.00000200":"1000.00000001","0.00000178":"65.00000000","0.00000170":"100.00000000","0.00000164":"210.17073171","0.00000160":"6250.00000000","0.00000100":"1999.00000000","0.00000095":"1612.31578947","0.00000090":"1111.11111111","0.00000080":"12500.00000000","0.00000054":"557.96296296","0.00000040":"25000.00000000","0.00000020":"50000.00000000","0.00000010":"200000.00000000","0.00000005":"200000.00000000","0.00000004":"2500.00000000","0.00000002":"556100.00000000","0.00000001":"1182263.00000000"}]}]]]`)
	err = p.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`[148,827984670,[["o",0,"0.02328500","0.00000000"],["o",0,"0.02328498","0.04303557"]]]`)
	err = p.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}
func TestWsHandleAccountData(t *testing.T) {
	t.Parallel()
	err := p.getCurrencyIDMap()
	if err != nil {
		t.Error(err)
	}
	jsons := []string{
		`[1000,"",[["o",807230187,"0.00000000", "f"],["b",267,"e","0.10000000"]]]`,
		`[1000,"",[["n",50,807230187,0,"1000.00000000","0.10000000","2018-11-07 16:42:42"],["b",267,"e","-0.10000000"]]]`,
		`[1000,"",[["t", 12345, "0.03000000", "0.50000000", "0.00250000", 0, 6083059, "0.00000375", "2018-09-08 05:54:09", "12345"]]]`,
		`[1000,"",[["k", 1337, ""]]]`,
	}
	for i := range jsons {
		err := p.wsHandleData([]byte(jsons[i]))
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGetHistoricCandles(t *testing.T) {
	currencyPair, err := currency.NewPairFromString("BTC_LTC")
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.GetHistoricCandles(currencyPair, asset.Spot, time.Unix(1588741402, 0), time.Unix(1588745003, 0), kline.FiveMin)
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.GetHistoricCandles(currencyPair, asset.Spot, time.Unix(1588741402, 0), time.Unix(1588745003, 0), kline.Interval(time.Hour*7))
	if err == nil {
		t.Fatal("unexpected result")
	}

	currencyPair.Quote = currency.NewCode("LTCC")
	_, err = p.GetHistoricCandles(currencyPair, asset.Spot, time.Unix(1588741402, 0), time.Unix(1588745003, 0), kline.FiveMin)
	if err == nil {
		t.Fatal(err)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	currencyPair, err := currency.NewPairFromString("BTC_LTC")
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.GetHistoricCandlesExtended(currencyPair, asset.Spot, time.Unix(1588741402, 0), time.Unix(1588745003, 0), kline.FiveMin)
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.GetHistoricCandlesExtended(currencyPair, asset.Spot, time.Unix(1588741402, 0), time.Unix(1588745003, 0), kline.Interval(time.Hour*7))
	if err == nil {
		t.Fatal("unexpected result")
	}

	currencyPair.Quote = currency.NewCode("LTCC")
	_, err = p.GetHistoricCandlesExtended(currencyPair, asset.Spot, time.Unix(1588741402, 0), time.Unix(1588745003, 0), kline.FiveMin)
	if err == nil {
		t.Fatal(err)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("BTC_XMR")
	if err != nil {
		t.Fatal(err)
	}
	if mockTests {
		t.Skip("relies on time.Now()")
	}
	_, err = p.GetRecentTrades(currencyPair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("BTC_XMR")
	if err != nil {
		t.Fatal(err)
	}
	tStart := time.Date(2020, 6, 6, 0, 0, 0, 0, time.UTC)
	tEnd := time.Date(2020, 6, 6, 1, 0, 0, 0, time.UTC)
	if !mockTests {
		tStart = time.Date(2020, time.Now().Month()-3, 6, 0, 0, 0, 0, time.UTC)
		tEnd = time.Date(2020, time.Now().Month()-3, 7, 0, 0, 0, 0, time.UTC)
	}
	_, err = p.GetHistoricTrades(currencyPair, asset.Spot, tStart, tEnd)
	if err != nil {
		t.Error(err)
	}
}
