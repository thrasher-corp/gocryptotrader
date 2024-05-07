package yobit

import (
	"context"
	"errors"
	"log"
	"math"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

var y = &Yobit{}

// Please supply your own keys for better unit testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

func TestMain(m *testing.M) {
	y.SetDefaults()
	yobitConfig := config.GetConfig()
	err := yobitConfig.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("Yobit load config error", err)
	}
	conf, err := yobitConfig.GetExchangeConfig("Yobit")
	if err != nil {
		log.Fatal("Yobit init error", err)
	}
	conf.API.Credentials.Key = apiKey
	conf.API.Credentials.Secret = apiSecret
	conf.API.AuthenticatedSupport = true

	err = y.Setup(conf)
	if err != nil {
		log.Fatal("Yobit setup error", err)
	}

	os.Exit(m.Run())
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := y.FetchTradablePairs(context.Background(), asset.Spot)
	if err != nil {
		t.Errorf("FetchTradablePairs err: %s", err)
	}
}

func TestGetInfo(t *testing.T) {
	t.Parallel()
	_, err := y.GetInfo(context.Background())
	if err != nil {
		t.Error("GetInfo() error")
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := y.GetTicker(context.Background(), "btc_usd")
	if err != nil {
		t.Error("GetTicker() error", err)
	}
}

func TestGetDepth(t *testing.T) {
	t.Parallel()
	_, err := y.GetDepth(context.Background(), "btc_usd")
	if err != nil {
		t.Error("GetDepth() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := y.GetTrades(context.Background(), "btc_usd")
	if err != nil {
		t.Error("GetTrades() error", err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	_, err := y.UpdateAccountInfo(context.Background(), asset.Spot)
	if err == nil {
		t.Error("GetAccountInfo() Expected error")
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := y.GetOpenOrders(context.Background(), "")
	if err == nil {
		t.Error("GetOpenOrders() Expected error")
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, y)
	_, err := y.GetOrderInfo(context.Background(), "1337", currency.NewPair(currency.BTC, currency.USD), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCryptoDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, y)

	_, err := y.GetCryptoDepositAddress(context.Background(), "bTc", false)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	err := y.CancelExistingOrder(context.Background(), 1337)
	if err == nil {
		t.Error("CancelOrder() Expected error")
	}
}

func TestTrade(t *testing.T) {
	t.Parallel()
	_, err := y.Trade(context.Background(), "", order.Buy.String(), 0, 0)
	if err == nil {
		t.Error("Trade() Expected error")
	}
}

func TestWithdrawCoinsToAddress(t *testing.T) {
	t.Parallel()
	_, err := y.WithdrawCoinsToAddress(context.Background(), "", 0, "")
	if err == nil {
		t.Error("WithdrawCoinsToAddress() Expected error")
	}
}

func TestCreateYobicode(t *testing.T) {
	t.Parallel()
	_, err := y.CreateCoupon(context.Background(), "bla", 0)
	if err == nil {
		t.Error("CreateYobicode() Expected error")
	}
}

func TestRedeemYobicode(t *testing.T) {
	t.Parallel()
	_, err := y.RedeemCoupon(context.Background(), "bla2")
	if err == nil {
		t.Error("RedeemYobicode() Expected error")
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
	var feeBuilder = setFeeBuilder()
	_, err := y.GetFeeByType(context.Background(), feeBuilder)
	if err != nil {
		t.Fatal(err)
	}
	if !sharedtestvalues.AreAPICredentialsSet(y) {
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
	var feeBuilder = setFeeBuilder()

	// CryptocurrencyTradeFee Basic
	if _, err := y.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if _, err := y.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if _, err := y.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if _, err := y.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := y.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := y.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	if _, err := y.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if _, err := y.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if _, err := y.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// InternationalBankWithdrawalFee QIWI
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	feeBuilder.BankTransactionType = exchange.Qiwi
	if _, err := y.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// InternationalBankWithdrawalFee Wire
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	feeBuilder.BankTransactionType = exchange.WireTransfer
	if _, err := y.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// InternationalBankWithdrawalFee Payeer
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	feeBuilder.BankTransactionType = exchange.Payeer
	if _, err := y.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// InternationalBankWithdrawalFee Capitalist
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.RUR
	feeBuilder.BankTransactionType = exchange.Capitalist
	if _, err := y.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// InternationalBankWithdrawalFee AdvCash
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	feeBuilder.BankTransactionType = exchange.AdvCash
	if _, err := y.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// InternationalBankWithdrawalFee PerfectMoney
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.RUR
	feeBuilder.BankTransactionType = exchange.PerfectMoney
	if _, err := y.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expectedResult := exchange.AutoWithdrawCryptoWithAPIPermissionText + " & " + exchange.WithdrawFiatViaWebsiteOnlyText
	withdrawPermissions := y.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	var getOrdersRequest = order.MultiOrderRequest{
		Type:      order.AnyType,
		Pairs:     []currency.Pair{currency.NewPair(currency.LTC, currency.BTC)},
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}

	_, err := y.GetActiveOrders(context.Background(), &getOrdersRequest)
	if sharedtestvalues.AreAPICredentialsSet(y) && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !sharedtestvalues.AreAPICredentialsSet(y) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	var getOrdersRequest = order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Pairs:     []currency.Pair{currency.NewPair(currency.LTC, currency.BTC)},
		StartTime: time.Unix(0, 0),
		EndTime:   time.Unix(math.MaxInt64, 0),
		Side:      order.AnySide,
	}

	_, err := y.GetOrderHistory(context.Background(), &getOrdersRequest)
	if sharedtestvalues.AreAPICredentialsSet(y) && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !sharedtestvalues.AreAPICredentialsSet(y) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, y, canManipulateRealOrders)

	var orderSubmission = &order.Submit{
		Exchange: y.Name,
		Pair: currency.Pair{
			Delimiter: "_",
			Base:      currency.BTC,
			Quote:     currency.USD,
		},
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		ClientID:  "meowOrder",
		AssetType: asset.Spot,
	}
	response, err := y.SubmitOrder(context.Background(), orderSubmission)
	if sharedtestvalues.AreAPICredentialsSet(y) && (err != nil || response.Status != order.New) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !sharedtestvalues.AreAPICredentialsSet(y) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, y, canManipulateRealOrders)

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
		AssetType:     asset.Spot,
	}

	err := y.CancelOrder(context.Background(), orderCancellation)
	if !sharedtestvalues.AreAPICredentialsSet(y) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(y) && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, y, canManipulateRealOrders)

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
		AssetType:     asset.Spot,
	}

	resp, err := y.CancelAllOrders(context.Background(), orderCancellation)

	if !sharedtestvalues.AreAPICredentialsSet(y) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(y) && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}

	if len(resp.Status) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.Status))
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, y, canManipulateRealOrders)

	_, err := y.ModifyOrder(context.Background(),
		&order.Modify{AssetType: asset.Spot})
	if err == nil {
		t.Error("ModifyOrder() Expected error")
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, y, canManipulateRealOrders)

	withdrawCryptoRequest := withdraw.Request{
		Exchange:    y.Name,
		Amount:      -1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}

	_, err := y.WithdrawCryptocurrencyFunds(context.Background(),
		&withdrawCryptoRequest)
	if !sharedtestvalues.AreAPICredentialsSet(y) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(y) && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, y, canManipulateRealOrders)

	var withdrawFiatRequest = withdraw.Request{}
	_, err := y.WithdrawFiatFunds(context.Background(), &withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'",
			common.ErrFunctionNotSupported,
			err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, y, canManipulateRealOrders)

	var withdrawFiatRequest = withdraw.Request{}
	_, err := y.WithdrawFiatFundsToInternationalBank(context.Background(),
		&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'",
			common.ErrFunctionNotSupported,
			err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	if sharedtestvalues.AreAPICredentialsSet(y) {
		_, err := y.GetDepositAddress(context.Background(), currency.BTC, "", "")
		if err != nil {
			t.Error(err)
		}
	} else {
		_, err := y.GetDepositAddress(context.Background(), currency.BTC, "", "")
		if err == nil {
			t.Error("GetDepositAddress() error")
		}
	}
}

func TestGetRecentTrades(t *testing.T) {
	currencyPair, err := currency.NewPairFromString("btc_usd")
	if err != nil {
		t.Fatal(err)
	}
	_, err = y.GetRecentTrades(context.Background(), currencyPair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	currencyPair, err := currency.NewPairFromString("btc_usd")
	if err != nil {
		t.Fatal(err)
	}
	_, err = y.GetHistoricTrades(context.Background(),
		currencyPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	if err != nil && err != common.ErrFunctionNotSupported {
		t.Error(err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("ETH_BTC")
	if err != nil {
		t.Fatal(err)
	}
	_, err = y.UpdateTicker(context.Background(), cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := y.UpdateTickers(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestWrapperGetServerTime(t *testing.T) {
	t.Parallel()
	st, err := y.GetServerTime(context.Background(), asset.Spot)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if st.IsZero() {
		t.Fatal("expected a time")
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, y)
	for _, a := range y.GetAssetTypes(false) {
		pairs, err := y.CurrencyPairs.GetPairs(a, false)
		require.NoError(t, err, "cannot get pairs for %s", a)
		require.NotEmpty(t, pairs, "no pairs for %s", a)
		resp, err := y.GetCurrencyTradeURL(context.Background(), a, pairs[0])
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}
