package poloniex

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own APIKEYS here for due diligence testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var p = &Poloniex{}

func TestStart(t *testing.T) {
	t.Parallel()
	err := p.Start(context.Background(), nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrNilPointer)
	}
	var testWg sync.WaitGroup
	err = p.Start(context.Background(), &testWg)
	if err != nil {
		t.Fatal(err)
	}
	testWg.Wait()
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
	_, err := p.GetFeeByType(context.Background(), feeBuilder)
	if err != nil {
		t.Fatal(err)
	}
	if !sharedtestvalues.AreAPICredentialsSet(p) {
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

// TODO: update
func TestGetFee(t *testing.T) {
	t.Parallel()
	var feeBuilder = setFeeBuilder()

	if sharedtestvalues.AreAPICredentialsSet(p) || mockTests {
		// CryptocurrencyTradeFee Basic
		if _, err := p.GetFee(context.Background(), feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if _, err := p.GetFee(context.Background(), feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if _, err := p.GetFee(context.Background(), feeBuilder); err != nil {
			t.Error(err)
		}
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := p.GetFee(context.Background(), feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := p.GetFee(context.Background(), feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	if _, err := p.GetFee(context.Background(), feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if _, err := p.GetFee(context.Background(), feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if _, err := p.GetFee(context.Background(), feeBuilder); err != nil {
		t.Error(err)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	var getOrdersRequest = order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}

	_, err := p.GetActiveOrders(context.Background(), &getOrdersRequest)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(p) && err != nil:
		t.Error("GetActiveOrders() error", err)
	case !sharedtestvalues.AreAPICredentialsSet(p) && !mockTests && err == nil:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock GetActiveOrders() err", err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	var getOrdersRequest = order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}
	results, err := p.GetOrderHistory(context.Background(), &getOrdersRequest)
	if err != nil {
		t.Error(err)
	} else {
		val, _ := json.Marshal(results)
		println(string(val))
	}
}

// func TestGetOrderStatus(t *testing.T) {
// 	t.Parallel()

// 	tests := []struct {
// 		name           string
// 		mock           bool
// 		orderID        string
// 		errExpected    bool
// 		errMsgExpected string
// 	}{
// 		{
// 			name:           "correct order ID",
// 			mock:           true,
// 			orderID:        "96238912841",
// 			errExpected:    false,
// 			errMsgExpected: "",
// 		},
// 		{
// 			name:           "wrong order ID",
// 			mock:           true,
// 			orderID:        "96238912842",
// 			errExpected:    true,
// 			errMsgExpected: "Order not found",
// 		},
// 	}

// 	for _, tt := range tests {
// 		tt := tt
// 		t.Run(tt.name, func(t *testing.T) {
// 			t.Parallel()
// 			if tt.mock != mockTests {
// 				t.Skip("mock mismatch, skipping")
// 			}

// 			_, err := p.GetAuthenticatedOrderStatus(context.Background(),
// 				tt.orderID)
// 			switch {
// 			case sharedtestvalues.AreAPICredentialsSet(p) && err != nil:
// 				t.Errorf("Could not get order status: %s", err)
// 			case !sharedtestvalues.AreAPICredentialsSet(p) && err == nil && !mockTests:
// 				t.Error("Expecting an error when no keys are set")
// 			case mockTests && err != nil:
// 				if !tt.errExpected {
// 					t.Errorf("Could not mock get order status: %s", err.Error())
// 				} else if !(strings.Contains(err.Error(), tt.errMsgExpected)) {
// 					t.Errorf("Could not mock get order status: %s", err.Error())
// 				}
// 			case mockTests:
// 				if tt.errExpected {
// 					t.Errorf("Mock get order status expect an error '%s', get no error", tt.errMsgExpected)
// 				}
// 			}
// 		})
// 	}
// }

// func TestGetOrderTrades(t *testing.T) {
// 	t.Parallel()

// 	tests := []struct {
// 		name           string
// 		mock           bool
// 		orderID        string
// 		errExpected    bool
// 		errMsgExpected string
// 	}{
// 		{
// 			name:           "correct order ID",
// 			mock:           true,
// 			orderID:        "96238912841",
// 			errExpected:    false,
// 			errMsgExpected: "",
// 		},
// 		{
// 			name:           "wrong order ID",
// 			mock:           true,
// 			orderID:        "96238912842",
// 			errExpected:    true,
// 			errMsgExpected: "Order not found",
// 		},
// 	}

// 	for _, tt := range tests {
// 		tt := tt
// 		t.Run(tt.name, func(t *testing.T) {
// 			t.Parallel()
// 			if tt.mock != mockTests {
// 				t.Skip("mock mismatch, skipping")
// 			}

// 			_, err := p.GetAuthenticatedOrderTrades(context.Background(), tt.orderID)
// 			switch {
// 			case sharedtestvalues.AreAPICredentialsSet(p) && err != nil:
// 				t.Errorf("Could not get order trades: %s", err)
// 			case !sharedtestvalues.AreAPICredentialsSet(p) && err == nil && !mockTests:
// 				t.Error("Expecting an error when no keys are set")
// 			case mockTests && err != nil:
// 				if !(tt.errExpected && strings.Contains(err.Error(), tt.errMsgExpected)) {
// 					t.Errorf("Could not mock get order trades: %s", err)
// 				}
// 			}
// 		})
// 	}
// }

// // Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// // ----------------------------------------------------------------------------------------------------------------------------

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, p, canManipulateRealOrders)
	}

	var orderSubmission = &order.Submit{
		Exchange: p.Name,
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

	response, err := p.SubmitOrder(context.Background(), orderSubmission)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(p) && (err != nil || response.Status != order.Filled):
		t.Errorf("Order failed to be placed: %v", err)
	case !sharedtestvalues.AreAPICredentialsSet(p) && !mockTests && err == nil:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock SubmitOrder() err", err)
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, p, canManipulateRealOrders)
	}
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currency.NewPair(currency.LTC, currency.BTC),
		AssetType:     asset.Spot,
	}

	err := p.CancelOrder(context.Background(), orderCancellation)
	switch {
	case !sharedtestvalues.AreAPICredentialsSet(p) && !mockTests && err == nil:
		t.Error("Expecting an error when no keys are set")
	case sharedtestvalues.AreAPICredentialsSet(p) && err != nil:
		t.Errorf("Could not cancel orders: %v", err)
	case mockTests && err != nil:
		t.Error("Mock CancelExchangeOrder() err", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, p, canManipulateRealOrders)
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
		AssetType:     asset.Spot,
	}

	resp, err := p.CancelAllOrders(context.Background(), orderCancellation)
	switch {
	case !sharedtestvalues.AreAPICredentialsSet(p) && !mockTests && err == nil:
		t.Error("Expecting an error when no keys are set")
	case sharedtestvalues.AreAPICredentialsSet(p) && err != nil:
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
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, p, canManipulateRealOrders)
	}

	_, err := p.ModifyOrder(context.Background(), &order.Modify{
		OrderID:   "1337",
		Price:     1337,
		AssetType: asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
	})
	switch {
	case sharedtestvalues.AreAPICredentialsSet(p) && err != nil && mockTests:
		t.Error("ModifyOrder() error", err)
	case !sharedtestvalues.AreAPICredentialsSet(p) && !mockTests && err == nil:
		t.Error("ModifyOrder() error cannot be nil")
	case mockTests && err != nil:
		t.Error("Mock ModifyOrder() err", err)
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	// if !mockTests {
	// 	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, p, canManipulateRealOrders)
	// }
	withdrawCryptoRequest := withdraw.Request{
		Exchange: p.Name,
		Crypto: withdraw.CryptoRequest{
			Address:   core.BitcoinDonationAddress,
			FeeAmount: 0,
		},
		Amount:        1,
		Currency:      currency.LTC,
		Description:   "WITHDRAW IT ALL",
		TradePassword: "Password",
	}
	_, err := p.WithdrawCryptocurrencyFunds(context.Background(), &withdrawCryptoRequest)
	if err != nil {
		t.Error(err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, p, canManipulateRealOrders)
	}

	var withdrawFiatRequest withdraw.Request
	_, err := p.WithdrawFiatFunds(context.Background(), &withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'",
			common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, p, canManipulateRealOrders)
	}

	var withdrawFiatRequest withdraw.Request
	_, err := p.WithdrawFiatFundsToInternationalBank(context.Background(),
		&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'",
			common.ErrFunctionNotSupported, err)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC_USDT")
	if err != nil {
		t.Fatal(err)
	}
	start := time.Unix(1588741402, 0)
	_, err = p.GetHistoricCandles(context.Background(), pair, asset.Spot, kline.FiveMin, start, time.Unix(1588745003, 0))
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC_USDT")
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.GetHistoricCandlesExtended(context.Background(), pair, asset.Spot, kline.FiveMin, time.Unix(1588741402, 0), time.Unix(1588745003, 0))
	if !errors.Is(err, nil) {
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
	_, err = p.GetRecentTrades(context.Background(), currencyPair, asset.Spot)
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
		tmNow := time.Now()
		tStart = time.Date(tmNow.Year(), tmNow.Month()-3, 6, 0, 0, 0, 0, time.UTC)
		tEnd = time.Date(tmNow.Year(), tmNow.Month()-3, 7, 0, 0, 0, 0, time.UTC)
	}
	_, err = p.GetHistoricTrades(context.Background(),
		currencyPair, asset.Spot, tStart, tEnd)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC_LTC")
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.UpdateTicker(context.Background(), cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := p.UpdateTickers(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	_, err := p.GetAvailableTransferChains(context.Background(), currency.USDT)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.GetAccountFundingHistory(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)

	_, err := p.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	_, err := p.CancelBatchOrders(context.Background(), []order.Cancel{
		{
			OrderID:   "1234",
			AssetType: asset.Spot,
			Pair:      currency.NewPair(currency.BTC, currency.USD),
		},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	st, err := p.GetServerTime(context.Background(), asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if st.IsZero() {
		t.Error("expected a valid time")
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := p.FetchTradablePairs(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSymbolInformation(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("ETH_USDT")
	if err != nil {
		t.Error(err)
	}
	_, err = p.GetSymbolInformation(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
	_, err = p.GetSymbolInformation(context.Background(), currency.EMPTYPAIR)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCurrencyInformations(t *testing.T) {
	t.Parallel()
	_, err := p.GetCurrencyInformations(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetCurrencyInformation(t *testing.T) {
	t.Parallel()
	_, err := p.GetCurrencyInformation(context.Background(), currency.BTC)
	if err != nil {
		t.Error(err)
	}
}

func TestGetV2CurrencyInformations(t *testing.T) {
	t.Parallel()
	_, err := p.GetV2CurrencyInformations(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetV2CurrencyInformation(t *testing.T) {
	t.Parallel()
	p.Verbose = true
	_, err := p.GetV2CurrencyInformation(context.Background(), currency.BTC)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSystemTimestamp(t *testing.T) {
	t.Parallel()
	_, err := p.GetSystemTimestamp(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarketPrices(t *testing.T) {
	t.Parallel()
	_, err := p.GetMarketPrices(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarketPrice(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("TRX_USDC")
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.GetMarketPrice(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarkPrices(t *testing.T) {
	t.Parallel()
	_, err := p.GetMarkPrices(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarkPrice(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC_USDT")
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.GetMarkPrice(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestMarkPriceComponents(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC_USDT")
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.MarkPriceComponents(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC_USDT")
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.GetOrderbook(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCandlesticks(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC_USDT")
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.GetCandlesticks(context.Background(), pair, kline.FiveMin, time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC_USDT")
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.GetTrades(context.Background(), pair, 10)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, err := p.GetTickers(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC_USDT")
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.GetTicker(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCollateralInfos(t *testing.T) {
	t.Parallel()
	_, err := p.GetCollateralInfos(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetCollateralInfo(t *testing.T) {
	t.Parallel()
	_, err := p.GetCollateralInfo(context.Background(), currency.BTC)
	if err != nil {
		t.Error(err)
	}
}

func TestGetBorrowRateInfo(t *testing.T) {
	t.Parallel()
	_, err := p.GetBorrowRateInfo(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.GetAccountInformation(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllBalances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.GetAllBalances(context.Background(), "")
	if err != nil {
		t.Error(err)
	}
	results, err := p.GetAllBalances(context.Background(), "SPOT")
	if err != nil {
		t.Error(err)
	} else {
		val, _ := json.Marshal(results)
		println(string(val))
	}
}

func TestGetAllBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.GetAllBalance(context.Background(), "219961623421431808", "")
	if err != nil {
		t.Error(err)
	}
	_, err = p.GetAllBalance(context.Background(), "219961623421431808", "SPOT")
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllAccountActivities(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.GetAllAccountActivities(context.Background(), time.Time{}, time.Time{}, 0, 0, 0, "", currency.EMPTYCODE)
	if err != nil {
		t.Error(err)
	}
}

func TestAccountsTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.AccountsTransfer(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Errorf("expected %v, got %v", errNilArgument, err)
	}
	_, err = p.AccountsTransfer(context.Background(), &AccountTransferParams{})
	if !errors.Is(err, currency.ErrCurrencyCodeEmpty) {
		t.Errorf("expected %v, got %v", currency.ErrCurrencyCodeEmpty, err)
	}
	_, err = p.AccountsTransfer(context.Background(), &AccountTransferParams{
		Ccy: currency.BTC,
	})
	if !errors.Is(err, order.ErrAmountIsInvalid) {
		t.Errorf("expected %v, got %v", order.ErrAmountIsInvalid, err)
	}
	_, err = p.AccountsTransfer(context.Background(), &AccountTransferParams{
		Amount:      1,
		Ccy:         currency.BTC,
		FromAccount: "219961623421431808",
	})
	if !errors.Is(err, errAddressRequired) {
		t.Errorf("expected %v, got %v", errAddressRequired, err)
	}
	_, err = p.AccountsTransfer(context.Background(), &AccountTransferParams{
		Amount:      1,
		Ccy:         currency.BTC,
		FromAccount: "219961623421431808",
		ToAccount:   "219961623421431890",
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountTransferRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetAccountTransferRecords(context.Background(), time.Time{}, time.Time{}, "", currency.BTC, 0, 0)
	if err != nil {
		t.Error(err)
	} else {
		val, _ := json.Marshal(result)
		println(string(val))
	}
}

func TestGetAccountTransferRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetAccountTransferRecord(context.Background(), "23123123120")
	if err != nil {
		t.Error(err)
	} else {
		val, _ := json.Marshal(result)
		println(string(val))
	}
}

func TestGetFeeInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.GetFeeInfo(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetInterestHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.GetInterestHistory(context.Background(), time.Time{}, time.Time{}, "", 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSubAccountInformations(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.GetSubAccountInformations(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetSubAccountBalances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.GetSubAccountBalances(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetSubAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.GetSubAccountBalance(context.Background(), "2d45301d-5f08-4a2b-a763-f9199778d854")
	if err != nil {
		t.Error(err)
	}
}

func TestSubAccountTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.SubAccountTransfer(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Errorf("expected %v, got %v", errNilArgument, err)
	}
	_, err = p.SubAccountTransfer(context.Background(), &SubAccountTransferParam{})
	if !errors.Is(err, currency.ErrCurrencyCodeEmpty) {
		t.Errorf("expected %v, got %v", currency.ErrCurrencyCodeEmpty, err)
	}
	_, err = p.SubAccountTransfer(context.Background(), &SubAccountTransferParam{
		Currency: currency.BTC,
	})
	if !errors.Is(err, order.ErrAmountIsInvalid) {
		t.Errorf("expected %v, got %v", order.ErrAmountIsInvalid, err)
	}
	_, err = p.SubAccountTransfer(context.Background(), &SubAccountTransferParam{
		Currency: currency.BTC,
		Amount:   1,
	})
	if !errors.Is(err, errAccountIDRequired) {
		t.Errorf("expected %v, got %v", errAccountIDRequired, err)
	}
	_, err = p.SubAccountTransfer(context.Background(), &SubAccountTransferParam{
		Currency:      currency.BTC,
		Amount:        1,
		FromAccountID: "1234568",
		ToAccountID:   "1234567",
	})
	if !errors.Is(err, errAccountTypeRequired) {
		t.Errorf("expected %v, got %v", errAccountTypeRequired, err)
	}
	_, err = p.SubAccountTransfer(context.Background(), &SubAccountTransferParam{
		Currency:        currency.BTC,
		Amount:          1,
		FromAccountID:   "1234568",
		ToAccountID:     "1234567",
		FromAccountType: "SPOT",
		ToAccountType:   "SPOT",
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetSubAccountTransferRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.GetSubAccountTransferRecords(context.Background(), currency.BTC, time.Time{}, time.Now(), "", "", "", "", "", 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSubAccountTransferRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.GetSubAccountTransferRecord(context.Background(), "1234567")
	if err != nil {
		t.Error(err)
	}
}

func TestGetDepositAddresses(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.GetDepositAddresses(context.Background(), currency.USDT)
	if err != nil {
		t.Error(err)
	}
}
func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.GetDepositAddress(context.Background(), currency.USDT, "", "USDT")
	if err != nil {
		t.Error(err)
	}
}

func TestWalletActivity(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	var start, end time.Time
	if mockTests {
		start = time.UnixMilli(1693741163970)
		end = time.UnixMilli(1693748363970)
	} else {
		start = time.Now().Add(-time.Hour * 2)
		end = time.Now()
	}
	_, err := p.WalletActivity(context.Background(), start, end, "")
	if err != nil {
		t.Error(err)
	}
}

func TestNewCurrencyDepoditAddress(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	}
	_, err := p.NewCurrencyDepoditAddress(context.Background(), currency.BTC)
	if err != nil {
		t.Error(err)
	}
}

func TestWithdrawCurrency(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	}
	_, err := p.WithdrawCurrency(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Errorf("expected %v, got %v", errNilArgument, err)
	}
	_, err = p.WithdrawCurrency(context.Background(), &WithdrawCurrencyParam{
		Currency: currency.BTC,
	})
	if !errors.Is(err, order.ErrAmountBelowMin) {
		t.Errorf("expected %v, got %v", order.ErrAmountBelowMin, err)
	}
	_, err = p.WithdrawCurrency(context.Background(), &WithdrawCurrencyParam{
		Currency: currency.BTC,
		Amount:   1,
	})
	if !errors.Is(err, errAddressRequired) {
		t.Errorf("expected %v, got %v", errAddressRequired, err)
	}
	_, err = p.WithdrawCurrency(context.Background(), &WithdrawCurrencyParam{
		Currency: currency.BTC,
		Amount:   1,
		Address:  "0xbb8d0d7c346daecc2380dabaa91f3ccf8ae232fb4",
	})
	if err != nil {
		t.Error(err)
	}
}

func TestWithdrawCurrencyV2(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	}
	_, err := p.WithdrawCurrencyV2(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Errorf("expected %v, got %v", errNilArgument, err)
	}
	if _, err = p.WithdrawCurrencyV2(context.Background(), &WithdrawCurrencyV2Param{
		Coin: currency.BTC}); !errors.Is(err, order.ErrAmountBelowMin) {
		t.Errorf("expected %v, got %v", order.ErrAmountBelowMin, err)
	}
	if _, err = p.WithdrawCurrencyV2(context.Background(), &WithdrawCurrencyV2Param{Coin: currency.BTC, Amount: 1}); !errors.Is(err, errInvalidWithdrawalChain) {
		t.Errorf("expected %v, got %v", errInvalidWithdrawalChain, err)
	}
	if _, err = p.WithdrawCurrencyV2(context.Background(), &WithdrawCurrencyV2Param{
		Coin: currency.BTC, Amount: 1, Network: "BTC"}); !errors.Is(err, errAddressRequired) {
		t.Errorf("expected %v, got %v", errAddressRequired, err)
	}
	if _, err = p.WithdrawCurrencyV2(context.Background(), &WithdrawCurrencyV2Param{
		Network: "BTC", Coin: currency.BTC, Amount: 1, Address: "0xbb8d0d7c346daecc2380dabaa91f3ccf8ae232fb4"}); err != nil {
		t.Error(err)
	}
}

func TestGetAccountMarginInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.GetAccountMarginInformation(context.Background(), "SPOT")
	if err != nil {
		t.Error(err)
	}
}

func TestGetBorrowStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.GetBorrowStatus(context.Background(), currency.USDT)
	if err != nil {
		t.Error(err)
	}
}

func TestMaximumBuySellAmount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	pair, err := currency.NewPairFromString("BTC_USDT")
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.MaximumBuySellAmount(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestPlaceOrder(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	}
	_, err := p.PlaceOrder(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Errorf("expected %v, got %v", errNilArgument, err)
	}
	_, err = p.PlaceOrder(context.Background(), &PlaceOrderParams{})
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Errorf("expected %v, got %v", currency.ErrCurrencyPairEmpty, err)
	}
	pair, err := currency.NewPairFromString("BTC_USDT")
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.PlaceOrder(context.Background(), &PlaceOrderParams{
		Symbol: pair,
	})
	if !errors.Is(err, order.ErrSideIsInvalid) {
		t.Errorf("expected %v, got %v", order.ErrSideIsInvalid, err)
	}
	_, err = p.PlaceOrder(context.Background(), &PlaceOrderParams{
		Symbol:        pair,
		Side:          order.Buy.String(),
		Type:          order.Market.String(),
		Quantity:      100,
		Price:         40000.50000,
		TimeInForce:   "GTC",
		ClientOrderID: "1234Abc",
	})
	if err != nil {
		t.Error(err)
	}
}

func TestPlaceBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := p.PlaceBatchOrders(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Errorf("expected %v, got %v", errNilArgument, err)
	}
	_, err = p.PlaceBatchOrders(context.Background(), []PlaceOrderParams{{}})
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Errorf("expected %v, got %v", currency.ErrCurrencyPairEmpty, err)
	}
	pair, err := currency.NewPairFromString("BTC_USDT")
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.PlaceBatchOrders(context.Background(), []PlaceOrderParams{
		{
			Symbol: pair,
		},
	})
	if !errors.Is(err, order.ErrSideIsInvalid) {
		t.Errorf("expected %v, got %v", order.ErrSideIsInvalid, err)
	}
	getPairFromString := func(pairString string) currency.Pair {
		pair, err := currency.NewPairFromString(pairString)
		if err != nil {
			return currency.EMPTYPAIR
		}
		return pair
	}
	result, err := p.PlaceBatchOrders(context.Background(), []PlaceOrderParams{
		{
			Symbol:        pair,
			Side:          order.Buy.String(),
			Type:          order.Market.String(),
			Quantity:      100,
			Price:         40000.50000,
			TimeInForce:   "GTC",
			ClientOrderID: "1234Abc",
		},
		{
			Symbol: getPairFromString("BTC_USDT"),
			Amount: 100,
			Side:   "BUY",
		},
		{
			Symbol:        getPairFromString("BTC_USDT"),
			Type:          "LIMIT",
			Quantity:      100,
			Side:          "BUY",
			Price:         40000.50000,
			TimeInForce:   "IOC",
			ClientOrderID: "1234Abc",
		},
		{
			Symbol: getPairFromString("ETH_USDT"),
			Amount: 1000,
			Side:   "BUY",
		},
		{
			Symbol:        getPairFromString("TRX_USDT"),
			Type:          "LIMIT",
			Quantity:      15000,
			Side:          "SELL",
			Price:         0.0623423423,
			TimeInForce:   "IOC",
			ClientOrderID: "456Xyz",
		},
	})
	if err != nil {
		t.Error(err)
	} else {
		val, _ := json.Marshal(result)
		println(string(val))
	}
}

func TestCancelReplaceOrder(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	}
	_, err := p.CancelReplaceOrder(context.Background(), &CancelReplaceOrderParam{})
	if !errors.Is(err, errNilArgument) {
		t.Errorf("expected %v, got %v", errNilArgument, err)
	}
	_, err = p.CancelReplaceOrder(context.Background(), &CancelReplaceOrderParam{
		ID:            "29772698821328896",
		ClientOrderID: "1234Abc",
		Price:         18000,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	pair, err := currency.NewPairFromString("BTC_USDT")
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.GetOpenOrders(context.Background(), pair, "", "NEXT", "", 10)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.GetOrderDetail(context.Background(), "12345536545645", "")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelOrderByID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.CancelOrderByID(context.Background(), "12345536545645")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelMultipleOrdersByIDs(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.CancelMultipleOrdersByIDs(context.Background(), &OrderCancellationParams{OrderIds: []string{"1234"}, ClientOrderIds: []string{"5678"}})
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAllTradeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.CancelAllTradeOrders(context.Background(), []string{"BTC_USDT", "ETH_USDT"}, []string{"SPOT"})
	if err != nil {
		t.Error(err)
	}
}

func TestKillSwitch(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.KillSwitch(context.Background(), "30")
	if err != nil {
		t.Error(err)
	}
}

func TestGetKillSwitchStatus(t *testing.T) {
	t.Parallel()
	_, err := p.GetKillSwitchStatus(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestCreateSmartOrder(t *testing.T) {
	t.Parallel()
	_, err := p.CreateSmartOrder(context.Background(), &SmartOrderRequestParam{})
	if !errors.Is(err, errNilArgument) {
		t.Errorf("expected %v, got %v", errNilArgument, err)
	}
	_, err = p.CreateSmartOrder(context.Background(), &SmartOrderRequestParam{
		Side: "BUY",
	})
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Errorf("expected %v, got %v", currency.ErrCurrencyPairEmpty, err)
	}
	pair, err := currency.NewPairFromString("BTC_USDT")
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.CreateSmartOrder(context.Background(), &SmartOrderRequestParam{
		Symbol: pair,
	})
	if !errors.Is(err, order.ErrSideIsInvalid) {
		t.Errorf("expected %v, got %v", order.ErrSideIsInvalid, err)
	}
	_, err = p.CreateSmartOrder(context.Background(), &SmartOrderRequestParam{
		Symbol:        pair,
		Side:          "BUY",
		Type:          orderTypeString(order.StopLimit),
		Quantity:      100,
		Price:         40000.50000,
		TimeInForce:   "GTC",
		ClientOrderID: "1234Abc",
	})
	if err != nil {
		t.Error(err)
	}
}

func TestCancelReplaceSmartOrder(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	}
	_, err := p.CancelReplaceSmartOrder(context.Background(), &CancelReplaceSmartOrderParam{})
	if !errors.Is(err, errNilArgument) {
		t.Errorf("expected %v, got %v", errNilArgument, err)
	}
	_, err = p.CancelReplaceSmartOrder(context.Background(), &CancelReplaceSmartOrderParam{
		ID:            "29772698821328896",
		ClientOrderID: "1234Abc",
		Price:         18000,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetSmartOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.GetSmartOpenOrders(context.Background(), 10)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSmartOrderDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.GetSmartOrderDetail(context.Background(), "123313413", "")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelSmartOrderByID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.CancelSmartOrderByID(context.Background(), "123313413", "")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelMultipleSmartOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.CancelMultipleSmartOrders(context.Background(), &OrderCancellationParams{OrderIds: []string{"1234"}, ClientOrderIds: []string{"5678"}})
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAllSmartOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.CancelAllSmartOrders(context.Background(), []string{"BTC_USDT", "ETH_USDT"}, []string{"SPOT"})
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrdersHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	pair, err := currency.NewPairFromString("BTC_USDT")
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.GetOrdersHistory(context.Background(), pair, "SPOT", "", "", "", "", 0, 10, time.Time{}, time.Time{}, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSmartOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	pair, err := currency.NewPairFromString("BTC_USDT")
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.GetSmartOrderHistory(context.Background(), pair, "SPOT", "", "", "", "", 0, 10, time.Time{}, time.Time{}, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC_USDT")
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.GetTradeHistory(context.Background(), currency.Pairs{pair}, "", 0, 0, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
}

func TestGetTradeOrderID(t *testing.T) {
	t.Parallel()
	_, err := p.GetTradeOrderID(context.Background(), "13123242323")
	if err != nil {
		t.Error(err)
	}
}
