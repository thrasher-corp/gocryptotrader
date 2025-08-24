package bitflyer

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own keys here for due diligence testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var e *Exchange

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("Bitflyer Setup error: %s", err)
	}

	if apiKey != "" && apiSecret != "" {
		e.API.AuthenticatedSupport = true
		e.SetCredentials(apiKey, apiSecret, "", "", "", "")
	}

	os.Exit(m.Run())
}

func TestGetLatestBlockCA(t *testing.T) {
	t.Parallel()
	_, err := e.GetLatestBlockCA(t.Context())
	if err != nil {
		t.Error("Bitflyer - GetLatestBlockCA() error:", err)
	}
}

func TestGetBlockCA(t *testing.T) {
	t.Parallel()
	_, err := e.GetBlockCA(t.Context(), "000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f")
	if err != nil {
		t.Error("Bitflyer - GetBlockCA() error:", err)
	}
}

func TestGetBlockbyHeightCA(t *testing.T) {
	t.Parallel()
	_, err := e.GetBlockbyHeightCA(t.Context(), 0)
	if err != nil {
		t.Error("Bitflyer - GetBlockbyHeightCA() error:", err)
	}
}

func TestGetTransactionByHashCA(t *testing.T) {
	t.Parallel()
	_, err := e.GetTransactionByHashCA(t.Context(), "0562d1f063cd4127053d838b165630445af5e480ceb24e1fd9ecea52903cb772")
	if err != nil {
		t.Error("Bitflyer - GetTransactionByHashCA() error:", err)
	}
}

func TestGetAddressInfoCA(t *testing.T) {
	t.Parallel()
	v, err := e.GetAddressInfoCA(t.Context(), core.BitcoinDonationAddress)
	if err != nil {
		t.Error("Bitflyer - GetAddressInfoCA() error:", err)
	}
	if v.UnconfirmedBalance == 0 || v.ConfirmedBalance == 0 {
		t.Log("Donation wallet is empty :( - please consider donating")
	}
}

func TestGetMarkets(t *testing.T) {
	t.Parallel()
	markets, err := e.GetMarkets(t.Context())
	if err != nil {
		t.Error("Bitflyer - GetMarkets() error:", err)
	}
	for _, market := range markets {
		if market.ProductCode == "" {
			t.Error("Bitflyer - ProductCode is empty in GetMarkets()")
		}
		if market.MarketType == "" {
			t.Error("Bitflyer - MarketType is empty in GetMarkets()")
		}
	}
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderBook(t.Context(), "BTC_JPY")
	if err != nil {
		t.Error("Bitflyer - GetOrderBook() error:", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetTicker(t.Context(), "BTC_JPY")
	if err != nil {
		t.Error("Bitflyer - GetTicker() error:", err)
	}
}

func TestGetExecutionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetExecutionHistory(t.Context(), "BTC_JPY")
	if err != nil {
		t.Error("Bitflyer - GetExecutionHistory() error:", err)
	}
}

func TestGetExchangeStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetExchangeStatus(t.Context())
	if err != nil {
		t.Error("Bitflyer - GetExchangeStatus() error:", err)
	}
}

func TestCheckFXString(t *testing.T) {
	t.Parallel()
	p, err := currency.NewPairDelimiter("FXBTC_JPY", "_")
	if err != nil {
		t.Fatal(err)
	}
	p = e.CheckFXString(p)
	if p.Base.String() != "FX_BTC" {
		t.Error("Bitflyer - CheckFXString() error")
	}
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:              1,
		FeeType:             exchange.CryptocurrencyTradeFee,
		Pair:                currency.NewPair(currency.BTC, currency.LTC),
		PurchasePrice:       1,
		FiatCurrency:        currency.JPY,
		BankTransactionType: exchange.WireTransfer,
	}
}

func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	feeBuilder := setFeeBuilder()
	_, err := e.GetFeeByType(t.Context(), feeBuilder)
	if err != nil {
		t.Fatal(err)
	}
	if !sharedtestvalues.AreAPICredentialsSet(e) {
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
	feeBuilder := setFeeBuilder()

	if sharedtestvalues.AreAPICredentialsSet(e) {
		// CryptocurrencyTradeFee Basic
		if _, err := e.GetFee(feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if _, err := e.GetFee(feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if _, err := e.GetFee(feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if _, err := e.GetFee(feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyWithdrawalFee Basic
		feeBuilder = setFeeBuilder()
		feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
		if _, err := e.GetFee(feeBuilder); err != nil {
			t.Error(err)
		}
	}

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.JPY
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.JPY
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expectedResult := exchange.AutoWithdrawFiatText + " & " + exchange.WithdrawCryptoViaWebsiteOnlyText
	withdrawPermissions := e.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
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
	if sharedtestvalues.AreAPICredentialsSet(e) && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !sharedtestvalues.AreAPICredentialsSet(e) && err == nil {
		t.Error("Expecting an error when no keys are set")
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
	if err != common.ErrNotYetImplemented {
		t.Errorf("Expected '%v', received '%v'", common.ErrNotYetImplemented, err)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	orderSubmission := &order.Submit{
		Exchange: e.Name,
		Pair: currency.Pair{
			Base:  currency.BTC,
			Quote: currency.LTC,
		},
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		ClientID:  "meowOrder",
		AssetType: asset.Spot,
	}
	_, err := e.SubmitOrder(t.Context(), orderSubmission)
	if err != common.ErrNotYetImplemented {
		t.Errorf("Expected 'Not Yet Implemented', received %v", err)
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      currencyPair,
		AssetType: asset.Spot,
	}

	err := e.CancelOrder(t.Context(), orderCancellation)

	if err != common.ErrNotYetImplemented {
		t.Errorf("Expected 'Not Yet Implemented', received %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      currencyPair,
		AssetType: asset.Spot,
	}

	_, err := e.CancelAllOrders(t.Context(), orderCancellation)

	if err != common.ErrNotYetImplemented {
		t.Errorf("Expected 'Not Yet Implemented', received %v", err)
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	withdrawCryptoRequest := withdraw.Request{
		Exchange:    e.Name,
		Amount:      -1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}

	_, err := e.WithdrawCryptocurrencyFunds(t.Context(),
		&withdrawCryptoRequest)
	if err != common.ErrNotYetImplemented {
		t.Errorf("Expected 'Not Yet Implemented', received %v", err)
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.ModifyOrder(t.Context(),
		&order.Modify{AssetType: asset.Spot})
	if err == nil {
		t.Error("ModifyOrder() Expected error")
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	withdrawFiatRequest := withdraw.Request{}

	_, err := e.WithdrawFiatFunds(t.Context(), &withdrawFiatRequest)
	if err != common.ErrNotYetImplemented {
		t.Errorf("Expected '%v', received: '%v'", common.ErrNotYetImplemented, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	withdrawFiatRequest := withdraw.Request{}

	_, err := e.WithdrawFiatFundsToInternationalBank(t.Context(),
		&withdrawFiatRequest)
	if err != common.ErrNotYetImplemented {
		t.Errorf("Expected '%v', received: '%v'", common.ErrNotYetImplemented, err)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("BTC_JPY")
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.GetRecentTrades(t.Context(), currencyPair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("BTC_JPY")
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.GetHistoricTrades(t.Context(),
		currencyPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	if err != nil && err != common.ErrFunctionNotSupported {
		t.Fatal(err)
	}
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
	err := e.CurrencyPairs.SetAssetEnabled(asset.Futures, false)
	require.NoError(t, err, "SetAssetEnabled must not error")
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.CurrencyPairs.GetPairs(a, false)
		require.NoErrorf(t, err, "cannot get pairs for %s", a)
		require.NotEmptyf(t, pairs, "no pairs for %s", a)
		resp, err := e.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoError(t, err, "GetCurrencyTradeURL must not error")
		assert.NotEmpty(t, resp, "GetCurrencyTradeURL should return an url")
	}
}
