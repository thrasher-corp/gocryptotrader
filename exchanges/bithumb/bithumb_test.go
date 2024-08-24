package bithumb

import (
	"context"
	"log"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own keys here for due diligence testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
	testCurrency            = "btc"
)

var b = &Bithumb{}

func TestMain(m *testing.M) {
	b.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("Bithumb load config error", err)
	}
	bitConfig, err := cfg.GetExchangeConfig("Bithumb")
	if err != nil {
		log.Fatal("Bithumb Setup() init error")
	}

	bitConfig.API.AuthenticatedSupport = true
	if apiKey != "" {
		bitConfig.API.Credentials.Key = apiKey
	}
	if apiSecret != "" {
		bitConfig.API.Credentials.Secret = apiSecret
	}

	err = b.Setup(bitConfig)
	if err != nil {
		log.Fatal("Bithumb setup error", err)
	}

	err = b.UpdateTradablePairs(context.Background(), false)
	if err != nil {
		log.Fatal("Bithumb Setup() init error", err)
	}

	os.Exit(m.Run())
}

func TestGetTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := b.GetTradablePairs(context.Background())
	require.NoError(t, err, "GetTradablePairs must not error")
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	tick, err := b.GetTicker(context.Background(), testCurrency)
	require.NoError(t, err, "GetTicker must not error")
	assert.Positive(t, tick.OpeningPrice, "OpeningPrice should be positive")
	assert.Positive(t, tick.ClosingPrice, "ClosingPrice should be positive")
	assert.Positive(t, tick.MinPrice, "MinPrice should be positive")
	assert.Positive(t, tick.MaxPrice, "MaxPrice should be positive")
	assert.Positive(t, tick.UnitsTraded, "UnitsTraded should be positive")
	assert.Positive(t, tick.AccumulatedTradeValue, "AccumulatedTradeValue should be positive")
	assert.Positive(t, tick.PreviousClosingPrice, "PreviousClosingPrice should be positive")
	assert.Positive(t, tick.UnitsTraded24Hr, "UnitsTraded24Hr should be positive")
	assert.Positive(t, tick.AccumulatedTradeValue24hr, "AccumulatedTradeValue24hr should be positive")
	assert.NotEmpty(t, tick.Fluctuate24Hr, "Fluctuate24Hr should not be empty")
	assert.NotEmpty(t, tick.FluctuateRate24hr, "FluctuateRate24hr should not be empty")
	assert.Positive(t, tick.Date, "Date should be positive")
}

// not all currencies have dates and fluctuation rates
func TestGetAllTickers(t *testing.T) {
	t.Parallel()
	tick, err := b.GetAllTickers(context.Background())
	require.NoError(t, err, "GetAllTickers must not error")
	for _, res := range tick {
		assert.Positive(t, res.OpeningPrice, "OpeningPrice should be positive")
		assert.Positive(t, res.ClosingPrice, "ClosingPrice should be positive")
		assert.Positive(t, res.MinPrice, "MinPrice should be positive")
		assert.Positive(t, res.MaxPrice, "MaxPrice should be positive")
		assert.Positive(t, res.UnitsTraded, "UnitsTraded should be positive")
		assert.Positive(t, res.AccumulatedTradeValue, "AccumulatedTradeValue should be positive")
		assert.Positive(t, res.PreviousClosingPrice, "PreviousClosingPrice should be positive")
		assert.Positive(t, res.UnitsTraded24Hr, "UnitsTraded24Hr should be positive")
		assert.Positive(t, res.AccumulatedTradeValue24hr, "AccumulatedTradeValue24hr should be positive")
	}
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()
	ob, err := b.GetOrderBook(context.Background(), testCurrency)
	require.NoError(t, err, "GetOrderBook must not error")
	assert.NotEmpty(t, ob.Status, "Status should not be empty")
	assert.NotEmpty(t, ob.Data.Timestamp, "Timestamp should not be empty")
	assert.NotEmpty(t, ob.Data.OrderCurrency, "OrderCurrency should not be empty")
	assert.NotEmpty(t, ob.Data.PaymentCurrency, "PaymentCurrency should not be empty")
}

func TestGetTransactionHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetTransactionHistory(context.Background(), testCurrency)
	require.NoError(t, err, "GetTransactionHistory must not error")
}

func TestGetAccountInformation(t *testing.T) {
	t.Parallel()

	// Offline test
	_, err := b.GetAccountInformation(context.Background(), "", "")
	assert.Error(t, err, "expected error when no order currency is specified")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err = b.GetAccountInformation(context.Background(),
		testCurrency,
		currency.KRW.String())
	require.NoError(t, err, "GetAccountInformation should not error")
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetAccountBalance(context.Background(), testCurrency)
	require.NoError(t, err, "GetAccountBalance must not error")
}

func TestGetWalletAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	a, err := b.GetWalletAddress(context.Background(), currency.BTC)
	require.NoError(t, err, "GetWalletAddress must not error")
	assert.NotEmpty(t, a.Data.Currency, "Currency should not be empty")
	assert.NotEmpty(t, a.Data.Tag, "Tag should not be empty")
	assert.NotEmpty(t, a.Data.WalletAddress, "WalletAddress should not be empty")
}

func TestGetLastTransaction(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetLastTransaction(context.Background())
	require.NoError(t, err, "GetLastTransaction must not error")
}

func TestGetOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetOrders(context.Background(),
		"1337", order.Bid.Lower(), 100, time.Time{}, currency.BTC, currency.KRW)
	require.NoError(t, err, "GetOrders must not error")
}

func TestGetUserTransactions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetUserTransactions(context.Background(), 0, 0, 0, currency.EMPTYCODE, currency.EMPTYCODE)
	require.NoError(t, err, "GetUserTransactions must not error")
}

func TestPlaceTrade(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	_, err := b.PlaceTrade(context.Background(),
		testCurrency, order.Bid.Lower(), 0, 0)
	require.NoError(t, err, "PlaceTrade must not error")
}

func TestGetOrderDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetOrderDetails(context.Background(),
		"1337", order.Bid.Lower(), testCurrency)
	require.NoError(t, err, "GetOrderDetails must not error")
}

func TestCancelTrade(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	_, err := b.CancelTrade(context.Background(), "", "", "")
	require.NoError(t, err, "CancelTrade must not error")
}

func TestWithdrawCrypto(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	_, err := b.WithdrawCrypto(context.Background(),
		"LQxiDhKU7idKiWQhx4ALKYkBx8xKEQVxJR", "", "ltc", 0)
	require.NoError(t, err, "WithdrawCrypto must not error")
}

func TestRequestKRWDepositDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.RequestKRWDepositDetails(context.Background())
	require.NoError(t, err, "RequestKRWDepositDetails must not error")
}

func TestRequestKRWWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	_, err := b.RequestKRWWithdraw(context.Background(),
		"102_bank", "1337", 1000)
	require.NoError(t, err, "RequestKRWWithdraw must not error")
}

func TestMarketBuyOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	p := currency.NewPair(currency.BTC, currency.KRW)
	_, err := b.MarketBuyOrder(context.Background(), p, 0)
	require.NoError(t, err, "MarketBuyOrder must not error")
}

func TestMarketSellOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	p := currency.NewPair(currency.BTC, currency.KRW)
	_, err := b.MarketSellOrder(context.Background(), p, 0)
	require.NoError(t, err, "MarketSellOrder must not error")
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	cp := currency.NewPairWithDelimiter("QTUM", "KRW", "-")
	tick, err := b.UpdateTicker(context.Background(), cp, asset.Spot)
	require.NoError(t, err, "UpdateTicker must not error")
	assert.Positive(t, tick.High, "High should be positive")
	assert.Positive(t, tick.Low, "Low should be positive")
	assert.Positive(t, tick.Open, "Open should be positive")
	assert.Positive(t, tick.Volume, "Volume should be positive")
	assert.NotEmpty(t, tick.Pair, "Pair should not be empty")
	assert.NotEmpty(t, tick.ExchangeName, "ExchangeName should not be empty")
	assert.NotEmpty(t, tick.LastUpdated, "LastUpdated should not be empty")
	assert.Equal(t, cp, tick.Pair, "Pair should be correct")
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := b.UpdateTickers(context.Background(), asset.Spot)
	require.NoError(t, err, "UpdateTickers must not error")
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:        1,
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          currency.NewPair(currency.BTC, currency.LTC),
		PurchasePrice: 1,
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	t.Parallel()
	var feeBuilder = setFeeBuilder()
	_, err := b.GetFeeByType(context.Background(), feeBuilder)
	require.NoError(t, err, "GetFeeByType must not error")

	if !sharedtestvalues.AreAPICredentialsSet(b) {
		assert.Equal(t, exchange.OfflineTradeFee, feeBuilder.FeeType, "FeeType should be correct")
	} else {
		assert.Equal(t, exchange.CryptocurrencyTradeFee, feeBuilder.FeeType, "FeeType should be correct")
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	var feeBuilder = setFeeBuilder()
	// CryptocurrencyTradeFee Basic
	_, err := b.GetFee(feeBuilder)
	require.NoError(t, err, "GetFee must not error")

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	_, err = b.GetFee(feeBuilder)
	require.NoError(t, err, "GetFee must not error")

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	_, err = b.GetFee(feeBuilder)
	require.NoError(t, err, "GetFee must not error")

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	_, err = b.GetFee(feeBuilder)
	require.NoError(t, err, "GetFee must not error")

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	_, err = b.GetFee(feeBuilder)
	require.NoError(t, err, "GetFee must not error")

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	_, err = b.GetFee(feeBuilder)
	require.NoError(t, err, "GetFee must not error")

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.HKD
	_, err = b.GetFee(feeBuilder)
	require.NoError(t, err, "GetFee must not error")

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.HKD
	_, err = b.GetFee(feeBuilder)
	require.NoError(t, err, "GetFee must not error")
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expectedResult := exchange.AutoWithdrawCryptoText + " & " + exchange.AutoWithdrawFiatText
	withdrawPermissions := b.FormatWithdrawPermissions()
	assert.Equal(t, expectedResult, withdrawPermissions, "withdrawPermissions should be correct")
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	var getOrdersRequest = order.MultiOrderRequest{
		Type:      order.AnyType,
		Side:      order.Sell,
		AssetType: asset.Spot,
	}

	_, err := b.GetActiveOrders(context.Background(), &getOrdersRequest)
	require.NoError(t, err, "GetActiveOrders must not error")
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	var getOrdersRequest = order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.AnySide,
		Pairs:     currency.Pairs{currency.NewPair(currency.BTC, currency.KRW)},
	}

	_, err := b.GetOrderHistory(context.Background(), &getOrdersRequest)
	require.NoError(t, err, "GetOrderHistory must not error")
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	var orderSubmission = &order.Submit{
		Exchange: b.Name,
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
	_, err := b.SubmitOrder(context.Background(), orderSubmission)
	require.NoError(t, err, "SubmitOrder must not error")
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
		AssetType:     asset.Spot,
	}

	err := b.CancelOrder(context.Background(), orderCancellation)
	require.NoError(t, err, "CancelOrder must not error")
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
		AssetType:     asset.Spot,
	}

	resp, err := b.CancelAllOrders(context.Background(), orderCancellation)
	require.NoError(t, err, "CancelAllOrders must not error")

	assert.Empty(t, resp.Status, "%v orders failed to cancel", len(resp.Status))
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.UpdateAccountInfo(context.Background(), asset.Spot)
	require.NoError(t, err, "UpdateAccountInfo must not error")
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	curr, err := currency.NewPairFromString("BTCUSD")
	require.NoError(t, err, "Issue setting currency")

	_, err = b.ModifyOrder(context.Background(), &order.Modify{
		OrderID:   "1337",
		Price:     100,
		Amount:    1000,
		Side:      order.Sell,
		Pair:      curr,
		AssetType: asset.Spot,
	})
	require.NoError(t, err, "ModifyOrder must not error")
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	t.Skip("TestWithdraw not allowed for live tests")
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	var withdrawFiatRequest = withdraw.Request{
		Type:     withdraw.Fiat,
		Exchange: b.Name,
		Fiat: withdraw.FiatRequest{
			Bank: banking.Account{
				SupportedExchanges:  b.Name,
				Enabled:             true,
				AccountName:         "Satoshi Nakamoto",
				AccountNumber:       "12345",
				BankAddress:         "123 Fake St",
				BankPostalCity:      "Tarry Town",
				BankCountry:         "AU",
				BankName:            "Federal Reserve Bank",
				SWIFTCode:           "CTBAAU2S",
				BankPostalCode:      "2088",
				IBAN:                "IT60X0542811101000000123456",
				SupportedCurrencies: "KRW",
			},
			WireCurrency:             currency.KRW.String(),
			RequiresIntermediaryBank: false,
			IsExpressWire:            false,
		},
		Amount:      10,
		Currency:    currency.KRW,
		Description: "WITHDRAW IT ALL",
	}

	_, err := b.WithdrawFiatFunds(context.Background(), &withdrawFiatRequest)
	require.NoError(t, err, "WithdrawFiatFunds must not error")
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	var withdrawFiatRequest = withdraw.Request{}
	_, err := b.WithdrawFiatFundsToInternationalBank(context.Background(),
		&withdrawFiatRequest)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetDepositAddress(context.Background(), currency.BTC, "", "")
	require.NoError(t, err, "GetDepositAddress must not error")
}

func TestGetCandleStick(t *testing.T) {
	t.Parallel()
	c, err := b.GetCandleStick(context.Background(), "BTC_KRW", "1m")
	require.NoError(t, err, "GetCandleStick must not error")
	assert.NotEmpty(t, c.Status, "Status should not be empty")
	assert.NotEmpty(t, c.Data, "Data should not be empty")
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCKRW")
	require.NoError(t, err, "Issue setting currency")
	startTime := time.Now().AddDate(0, -1, 0)
	c, err := b.GetHistoricCandles(context.Background(), pair, asset.Spot, kline.OneDay, startTime, time.Now())
	require.NoError(t, err, "GetHistoricCandles must not error")
	assert.NotEmpty(t, c.Exchange, "Exchange should not be empty")
	assert.NotEmpty(t, c.Candles, "Candles should not be empty")
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCKRW")
	require.NoError(t, err, "Issue setting currency")

	startTime := time.Now().Add(-time.Hour * 24)
	_, err = b.GetHistoricCandlesExtended(context.Background(), pair, asset.Spot, kline.OneDay, startTime, time.Now())
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("BTC_KRW")
	require.NoError(t, err, "Issue setting currency")

	tr, err := b.GetRecentTrades(context.Background(), currencyPair, asset.Spot)
	require.NoError(t, err, "GetRecentTrades must not error")
	assert.NotEmpty(t, tr, "Trades should not be empty")
	for _, req := range tr {
		assert.Positive(t, req.Amount, "Amount should be positive")
		assert.Equal(t, currencyPair, req.CurrencyPair, "Pair should be correct")
		assert.Equal(t, asset.Spot, req.AssetType, "AssetType should be set")
		assert.NotEmpty(t, req.Timestamp, "Timestamp should not be empty")
		assert.Positive(t, req.Price, "Price should be positive")
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("BTC_KRW")
	require.NoError(t, err, "Issue setting currency")

	_, err = b.GetHistoricTrades(context.Background(),
		currencyPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := b.UpdateOrderExecutionLimits(context.Background(), asset.Empty)
	require.NoError(t, err, "UpdateOrderExecutionLimits must not error")

	cp := currency.NewPair(currency.BTC, currency.KRW)
	limit, err := b.GetOrderExecutionLimits(asset.Spot, cp)
	require.NoError(t, err, "GetOrderExecutionLimits must not error")

	err = limit.Conforms(46241000, 0.00001, order.Limit)
	assert.ErrorIs(t, err, order.ErrAmountBelowMin, "expected error: %v, but got: %v", order.ErrAmountBelowMin, err)

	err = limit.Conforms(46241000, 0.0001, order.Limit)
	assert.NoError(t, err, "expected no error, but got: %v", err)
}

func TestGetAmountMinimum(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name      string
		unitprice float64
		expected  float64
	}{
		{
			name:      "ETH-KRW",
			unitprice: 2638000.0,
			expected:  0.0002,
		},
		{
			name:      "DOGE-KRW",
			unitprice: 236.5,
			expected:  2.1142,
		},
		{
			name:      "XRP-KRW",
			unitprice: 818.8,
			expected:  0.6107,
		},
		{
			name:      "LTC-KRW",
			unitprice: 160100,
			expected:  0.0032,
		},
		{
			name:      "BTC-KRW",
			unitprice: 46079000,
			expected:  0.0001,
		},
		{
			name:      "nonsense",
			unitprice: 0,
			expected:  0,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			minAmount := getAmountMinimum(tt.unitprice)
			assert.Equal(t, tt.expected, minAmount, "minAmount should be correct for %s", tt.unitprice)
		})
	}
}

func TestGetAssetStatus(t *testing.T) {
	t.Parallel()
	_, err := b.GetAssetStatus(context.Background(), "")
	assert.ErrorIs(t, err, errSymbolIsEmpty)

	s, err := b.GetAssetStatus(context.Background(), "sol")
	require.NoError(t, err, "GetAssetStatus must not error")
	assert.NotEmpty(t, s.Status, "Status should not be empty")
	assert.NotEmpty(t, s.Data.DepositStatus, "DepositStatus should not be empty")
	assert.NotEmpty(t, s.Data.WithdrawalStatus, "WithdrawalStatus should not be empty")
}

func TestGetAssetStatusAll(t *testing.T) {
	t.Parallel()
	s, err := b.GetAssetStatusAll(context.Background())
	require.NoError(t, err, "GetAssetStatusAll must not error")
	require.NoError(t, err, "GetAssetStatus must not error")
	assert.NotEmpty(t, s.Status, "Status should not be empty")
}

func TestUpdateCurrencyStates(t *testing.T) {
	t.Parallel()
	err := b.UpdateCurrencyStates(context.Background(), asset.Spot)
	require.NoError(t, err, "UpdateCurrencyStates must not error")
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Spot)
	require.NoError(t, err, "TestGetWithdrawalsHistory must not error")
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetOrderInfo(context.Background(), "1234", currency.NewPair(currency.BTC, currency.USDT), asset.Spot)
	require.NoError(t, err, "GetOrderInfo must not error")
}

func TestGetWithdrawalHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Spot)
	require.NoError(t, err, "GetWithdrawalsHistory must not error")
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, b)
	for _, a := range b.GetAssetTypes(false) {
		pairs, err := b.CurrencyPairs.GetPairs(a, false)
		require.NoError(t, err, "cannot get pairs for %s", a)
		require.NotEmpty(t, pairs, "no pairs for %s", a)
		resp, err := b.GetCurrencyTradeURL(context.Background(), a, pairs[0])
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}
