package bitstamp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please add your private keys and customerID for better tests
const (
	apiKey                  = ""
	apiSecret               = ""
	customerID              = "" // This is the customer id you use to log in
	canManipulateRealOrders = false
)

var (
	b          = &Bitstamp{}
	btcusdPair = currency.NewBTCUSD()
)

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:        5,
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          currency.NewPair(currency.LTC, currency.BTC),
		PurchasePrice: 1800,
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	t.Parallel()

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	feeBuilder := setFeeBuilder()
	_, err := b.GetFeeByType(t.Context(), feeBuilder)
	require.NoError(t, err, "GetFeeByType must not error")
	if mockTests {
		assert.Equal(t, exchange.OfflineTradeFee, feeBuilder.FeeType, "TradeFee should be correct")
	} else {
		assert.Equal(t, exchange.CryptocurrencyTradeFee, feeBuilder.FeeType, "TradeFee should be correct")
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()

	feeBuilder := setFeeBuilder()

	// CryptocurrencyTradeFee Basic
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	fee, err := b.GetFee(t.Context(), feeBuilder)
	require.NoError(t, err, "GetFee must not error")
	if mockTests {
		assert.NotEmpty(t, fee, "Fee should not be empty")
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	_, err = b.GetFee(t.Context(), feeBuilder)
	require.NoError(t, err, "GetFee must not error")

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	fee, err = b.GetFee(t.Context(), feeBuilder)
	require.NoError(t, err, "GetFee must not error")
	if mockTests {
		assert.Positive(t, fee, "Maker fee should be positive")
	}

	// CryptocurrencyTradeFee IsTaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = false
	fee, err = b.GetFee(t.Context(), feeBuilder)
	require.NoError(t, err, "GetFee must not error")
	if mockTests {
		assert.Positive(t, fee, "Taker fee should be positive")
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	_, err = b.GetFee(t.Context(), feeBuilder)
	require.NoError(t, err, "GetFee must not error")

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	_, err = b.GetFee(t.Context(), feeBuilder)
	require.NoError(t, err, "GetFee must not error")

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	_, err = b.GetFee(t.Context(), feeBuilder)
	require.NoError(t, err, "GetFee must not error")

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.HKD
	_, err = b.GetFee(t.Context(), feeBuilder)
	require.NoError(t, err, "GetFee must not error")

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.HKD
	fee, err = b.GetFee(t.Context(), feeBuilder)
	require.NoError(t, err, "GetFee must not error")
	assert.NotEmpty(t, fee, "Fee should not be empty")
}

func TestGetAccountTradingFee(t *testing.T) {
	t.Parallel()

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}

	fee, err := b.GetAccountTradingFee(t.Context(), currency.NewPair(currency.LTC, currency.BTC))
	require.NoError(t, err, "GetAccountTradingFee must not error")
	if mockTests {
		assert.Positive(t, fee.Fees.Maker, "Maker should be positive")
		assert.Positive(t, fee.Fees.Taker, "Taker should be positive")
	}
	assert.NotEmpty(t, fee.Symbol, "Symbol should not be empty")
	assert.Equal(t, "ltcbtc", fee.Symbol, "Symbol should be correct")

	_, err = b.GetAccountTradingFee(t.Context(), currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "Should get back the right error")
}

func TestGetAccountTradingFees(t *testing.T) {
	t.Parallel()

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}

	fees, err := b.GetAccountTradingFees(t.Context())
	require.NoError(t, err, "GetAccountTradingFee must not error")
	if assert.NotEmpty(t, fees, "Should get back multiple fees") {
		fee := fees[0]
		assert.NotEmpty(t, fee.Symbol, "Should get back a symbol")
		if mockTests {
			assert.Positive(t, fee.Fees.Maker, "Maker should be positive")
			assert.Positive(t, fee.Fees.Taker, "Taker should be positive")
		}
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()

	tick, err := b.GetTicker(t.Context(),
		currency.BTC.String()+currency.USD.String(), false)
	require.NoError(t, err, "GetTicker must not error")
	assert.Positive(t, tick.Ask, "Ask should be positive")
	assert.Positive(t, tick.Bid, "Bid should be positive")
	assert.Positive(t, tick.High, "High should be positive")
	assert.Positive(t, tick.Low, "Low should be positive")
	assert.Positive(t, tick.Last, "Last should be positive")
	assert.Positive(t, tick.Open, "Open should be positive")
	assert.Positive(t, tick.Volume, "Volume should be positive")
	assert.Positive(t, tick.Vwap, "Vwap should be positive")
	assert.Positive(t, tick.Open24, "Open24 should be positive")
	assert.NotEmpty(t, tick.PercentChange24, "PercentChange24 should be positive")
	assert.NotEmpty(t, tick.Timestamp, "Timestamp should not be empty")
	assert.Contains(t, []order.Side{order.Buy, order.Sell}, tick.Side.Side(), "Side should be either Buy or Sell")
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	ob, err := b.GetOrderbook(t.Context(), currency.BTC.String()+currency.USD.String())
	require.NoError(t, err, "GetOrderbook must not error")
	assert.NotEmpty(t, ob.Timestamp, "Timestamp should not be empty")
	for i, o := range [][]OrderbookBase{ob.Asks, ob.Bids} {
		s := []string{"Ask", "Bid"}[i]
		if assert.NotEmptyf(t, o, "Should have items in %ss", s) {
			a := o[0]
			assert.Positivef(t, a.Price, "%ss Price should be positive", s)
			assert.Positivef(t, a.Amount, "%ss Amount should be positive", s)
		}
	}
}

func TestGetTradingPairs(t *testing.T) {
	t.Parallel()

	p, err := b.GetTradingPairs(t.Context())
	require.NoError(t, err, "GetTradingPairs must not error")
	assert.NotEmpty(t, p, "Pairs should not be empty")
	for _, res := range p {
		if mockTests {
			assert.Positive(t, res.BaseDecimals, "BaseDecimals should be positive")
			assert.Positive(t, res.CounterDecimals, "CounterDecimals should be positive")
		}
		assert.NotEmpty(t, res.Name, "Name should not be empty")
		assert.Positive(t, res.MinimumOrder, "MinimumOrder should be positive")
		assert.NotEmpty(t, res.URLSymbol, "URLSymbol should not be empty")
		assert.NotEmpty(t, res.Description, "Description should not be empty")
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()

	p, err := b.FetchTradablePairs(t.Context(), asset.Spot)
	require.NoError(t, err, "FetchTradablePairs must not error")
	assert.True(t, p.Contains(currency.NewBTCUSD(), true), "Pairs should contain BTC/USD")
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	err := b.UpdateTradablePairs(t.Context(), true)
	require.NoError(t, err, "UpdateTradablePairs must not error")
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()

	type limitTest struct {
		pair currency.Pair
		step float64
		min  float64
	}

	tests := map[asset.Item][]limitTest{
		asset.Spot: {
			{currency.NewPair(currency.ETH, currency.USDT), 0.01, 20},
			{currency.NewBTCUSDT(), 0.01, 20},
		},
	}
	for assetItem, limitTests := range tests {
		if err := b.UpdateOrderExecutionLimits(t.Context(), assetItem); err != nil {
			t.Errorf("Error fetching %s pairs for test: %v", assetItem, err)
		}
		for _, limitTest := range limitTests {
			limits, err := b.GetOrderExecutionLimits(assetItem, limitTest.pair)
			if err != nil {
				t.Errorf("Bitstamp GetOrderExecutionLimits() error during TestExecutionLimits; Asset: %s Pair: %s Err: %v", assetItem, limitTest.pair, err)
				continue
			}
			assert.NotEmpty(t, limits.Pair, "Pair should not be empty")
			assert.Positive(t, limits.PriceStepIncrementSize, "PriceStepIncrementSize should be positive")
			assert.Positive(t, limits.AmountStepIncrementSize, "AmountStepIncrementSize should be positive")
			assert.Positive(t, limits.MinimumQuoteAmount, "MinimumQuoteAmount should be positive")
			if mockTests {
				if got := limits.PriceStepIncrementSize; got != limitTest.step {
					t.Errorf("Bitstamp UpdateOrderExecutionLimits wrong PriceStepIncrementSize; Asset: %s Pair: %s Expected: %v Got: %v", assetItem, limitTest.pair, limitTest.step, got)
				}
				if got := limits.MinimumQuoteAmount; got != limitTest.min {
					t.Errorf("Bitstamp UpdateOrderExecutionLimits wrong MinAmount; Pair: %s Expected: %v Got: %v", limitTest.pair, limitTest.min, got)
				}
			}
		}
	}
}

func TestGetTransactions(t *testing.T) {
	t.Parallel()

	tr, err := b.GetTransactions(t.Context(),
		currency.BTC.String()+currency.USD.String(), "hour")
	require.NoError(t, err, "GetTransactions must not error")
	assert.NotEmpty(t, tr, "Transactions should not be empty")
	for _, res := range tr {
		assert.NotEmpty(t, res.Date, "Date should not be empty")
		assert.Positive(t, res.Amount, "Amount should be positive")
		assert.Positive(t, res.Price, "Price should be positive")
		assert.NotEmpty(t, res.TradeID, "TradeID should not be empty")
	}
}

func TestGetEURUSDConversionRate(t *testing.T) {
	t.Parallel()

	c, err := b.GetEURUSDConversionRate(t.Context())
	require.NoError(t, err, "GetEURUSDConversionRate must not error")
	assert.Positive(t, c.Sell, "Sell should be positive")
	assert.Positive(t, c.Buy, "Buy should be positive")
}

func TestGetBalance(t *testing.T) {
	t.Parallel()

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	bal, err := b.GetBalance(t.Context())
	require.NoError(t, err, "GetBalance must not error")
	if mockTests {
		for k, e := range map[string]Balance{
			"USDT": {
				Available:     42.42,
				Balance:       1337.42,
				Reserved:      1295.00,
				WithdrawalFee: 5.0,
			},
			"BTC": {
				Available:     9.1,
				Balance:       11.2,
				Reserved:      2.1,
				WithdrawalFee: 0.00050000,
			},
		} {
			assert.Equal(t, e.Available, bal[k].Available, "Available balance should match")
			assert.Equal(t, e.Balance, bal[k].Balance, "Balance should match")
			assert.Equal(t, e.Reserved, bal[k].Reserved, "Reserved balance should match")
			assert.Equal(t, e.WithdrawalFee, bal[k].WithdrawalFee, "WithdrawalFee should match")
		}
	}
}

func TestGetUserTransactions(t *testing.T) {
	t.Parallel()

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	tr, err := b.GetUserTransactions(t.Context(), "btcusd")
	require.NoError(t, err, "GetUserTransactions must not error")
	if mockTests {
		assert.NotEmpty(t, tr, "Transactions should not be empty")
		for _, res := range tr {
			assert.NotEmpty(t, res.OrderID, "OrderID should not be empty")
			assert.NotEmpty(t, res.Date, "Date should not be empty")
		}
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	o, err := b.GetOpenOrders(t.Context(), "btcusd")
	require.NoError(t, err, "GetOpenOrders must not error")
	if mockTests {
		assert.NotEmpty(t, o, "Orders should not be empty")
		for _, res := range o {
			assert.Equal(t, time.Date(2022, 1, 31, 14, 43, 15, 0, time.UTC), res.DateTime.Time(), "DateTime should match")
			assert.Equal(t, int64(1234123412341234), res.ID, "ID should match")
			assert.Equal(t, 0.50000000, res.Amount, "Amount should match")
			assert.Equal(t, 100.00, res.Price, "Price should match")
			assert.Equal(t, int64(0), res.Type, "Type should match")
			assert.Equal(t, 0.50000000, res.AmountAtCreate, "AmountAtCreate should match")
			assert.Equal(t, 110.00, res.LimitPrice, "LimitPrice should match")
			assert.Equal(t, "1234123412341234", res.ClientOrderID, "ClientOrderID should match")
			assert.Equal(t, "BTC/USD", res.Market, "Market should match")
		}
	}
}

func TestGetOrderStatus(t *testing.T) {
	t.Parallel()

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	o, err := b.GetOrderStatus(t.Context(), 1458532827766784)
	if !mockTests {
		assert.ErrorContains(t, err, "Order not found")
	} else {
		require.NoError(t, err, "GetOrderStatus must not error")
		assert.Equal(t, time.Date(2022, 1, 31, 14, 43, 15, 0, time.UTC), o.DateTime.Time(), "DateTime should match")
		assert.Equal(t, "1458532827766784", o.ID, "OrderID should match")
		assert.Equal(t, 200.00, o.AmountRemaining, "AmountRemaining should match")
		assert.Equal(t, int64(0), o.Type, "Type should match")
		assert.Equal(t, "0.50000000", o.ClientOrderID, "ClientOrderID should match")
		assert.Equal(t, "BTC/USD", o.Market, "Market should match")
		for _, tr := range o.Transactions {
			assert.Equal(t, time.Date(2022, 1, 31, 14, 43, 15, 0, time.UTC), tr.DateTime.Time(), "DateTime should match")
			assert.Equal(t, 50.00, tr.Price, "Price should match")
			assert.Equal(t, 101.00, tr.FromCurrency, "FromCurrency should match")
			assert.Equal(t, 1.0, tr.ToCurrency, "ToCurrency should match")
			assert.Equal(t, int64(0), o.Type, "Type should match")
		}
	}
}

func TestGetWithdrawalRequests(t *testing.T) {
	t.Parallel()

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	r, err := b.GetWithdrawalRequests(t.Context(), 1)
	require.NoError(t, err, "GetWithdrawalRequests must not error")
	if mockTests {
		assert.NotEmpty(t, r, "GetWithdrawalRequests should return a withdrawal request")
		for _, req := range r {
			assert.Equal(t, int64(1), req.OrderID, "OrderId should match")
			assert.Equal(t, "aMDHooGmAkyrsaQiKhAORhSNTmoRzxqWIO", req.Address, "Address should match")
			assert.Equal(t, time.Date(2022, 1, 31, 16, 7, 32, 0, time.UTC), req.Date.Time(), "Date should match")
			assert.Equal(t, currency.BTC, req.Currency, "Currency should match")
			assert.Equal(t, 0.00006000, req.Amount, "Amount should match")
			assert.Equal(t, "NsOeFbQhRnpGzNIThWGBTkQwRJqTNOGPVhYavrVyMfkAyMUmIlUpFIwGTzSvpeOP", req.TransactionID, "TransactionID should match")
			assert.Equal(t, int64(2), req.Status, "Status should match")
			assert.Equal(t, int64(0), req.Type, "Type should match")
			assert.Equal(t, "bitcoin", req.Network, "Network should match")
			assert.Equal(t, int64(1), req.TxID, "TxID should match")
		}
	}
}

func TestGetUnconfirmedBitcoinDeposits(t *testing.T) {
	t.Parallel()

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	d, err := b.GetUnconfirmedBitcoinDeposits(t.Context())
	require.NoError(t, err, "GetUnconfirmedBitcoinDeposits must not error")
	if mockTests {
		assert.NotEmpty(t, d, "Deposits should not be empty")
		for _, res := range d {
			assert.Equal(t, "0x6a56f5b80f04b4fd70d64d72e1396698635e5436", res.Address, "Address should match")
			assert.Equal(t, 89473951, res.DestinationTag, "DestinationTag should match")
			assert.Equal(t, "299576079", res.MemoID, "MemoID should match")
		}
	}
}

func TestTransferAccountBalance(t *testing.T) {
	t.Parallel()

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	err := b.TransferAccountBalance(t.Context(),
		10000, "BTC", "1234567", true)
	if !mockTests {
		assert.ErrorContains(t, err, "Sub account with identifier \"1234567\" does not exist.")
	} else {
		require.NoError(t, err, "TransferAccountBalance must not error")
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()

	expectedResult := exchange.AutoWithdrawCryptoText +
		" & " +
		exchange.AutoWithdrawFiatText
	withdrawPermissions := b.FormatWithdrawPermissions()
	assert.Equal(t, expectedResult, withdrawPermissions, "Permissions should be the same")
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	o, err := b.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.AnySide,
	})
	require.NoError(t, err, "GetActiveOrders must not error")
	if mockTests {
		assert.NotEmpty(t, o, "ActiveOrders should not be empty")
		for _, res := range o {
			assert.Equal(t, "1234123412341234", res.OrderID, "OrderID should be correct")
			assert.Equal(t, time.Date(2022, time.January, 31, 14, 43, 15, 0, time.UTC), res.Date, "Date should be correct")
			assert.Equal(t, order.Buy, res.Side, "Order Side should be correct")
			assert.Equal(t, 100.00, res.Price, "Price should be correct")
			assert.Equal(t, currency.NewPairWithDelimiter("BTC", "USD", "/"), res.Pair, "Pair should be correct")
			assert.Equal(t, 0.50000000, res.Amount, "Amount should be correct")
		}
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	o, err := b.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.AnySide,
	})
	require.NoError(t, err, "GetOrderHistory must not error")
	if mockTests {
		assert.NotEmpty(t, o, "OrderHistory should not be empty")
		for _, res := range o {
			assert.NotEmpty(t, res.OrderID, "OrderID should not be empty")
			assert.NotEmpty(t, res.Date, "Date should not be empty")
		}
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------

func TestSubmitOrder(t *testing.T) {
	t.Parallel()

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	}
	o, err := b.SubmitOrder(t.Context(), &order.Submit{
		Exchange: b.Name,
		Pair: currency.Pair{
			Base:  currency.BTC,
			Quote: currency.USD,
		},
		Side:          order.Buy,
		Type:          order.Limit,
		Price:         2211.00,
		Amount:        45,
		ClientOrderID: "123456789",
		AssetType:     asset.Spot,
	})
	if !mockTests {
		assert.ErrorContains(t, err, "You have only 0 USD available. Check your account balance for details.")
	} else {
		require.NoError(t, err, "SubmitOrder must not error")
		assert.Equal(t, 45.0, o.Amount, "Amount should be correct")
		assert.Equal(t, asset.Spot, o.AssetType, "AssetType should be correct")
		assert.Equal(t, "123456789", o.ClientOrderID, "ClientOrderID should be correct")
		assert.Equal(t, "1234123412341234", o.OrderID, "OrderID should be correct")
		assert.Equal(t, 2211.0, o.Price, "Price should be correct")
		assert.Equal(t, btcusdPair, o.Pair, "Pair should be correct")
		assert.WithinRange(t, o.Date, time.Now().Add(-24*time.Hour), time.Now(), "Date should be correct")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	}
	err := b.CancelOrder(t.Context(), &order.Cancel{
		OrderID: "1453282316578816",
	})
	if !mockTests {
		assert.ErrorContains(t, err, "Order not found")
	} else {
		require.NoError(t, err, "CancelExchangeOrder must not error")
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	}
	resp, err := b.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot})
	require.NoError(t, err, "CancelAllOrders must not error")
	if len(resp.Status) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.Status))
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()

	_, err := b.ModifyOrder(t.Context(), &order.Modify{AssetType: asset.Spot})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestWithdraw(t *testing.T) {
	t.Parallel()

	if !mockTests {
		t.Skip("TestWithdraw not allowed for live tests")
	}
	w, err := b.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
		Exchange:    b.Name,
		Amount:      6,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	})
	require.NoError(t, err, "WithdrawCryptocurrencyFunds must not error")
	assert.Equal(t, "1", w.ID, "Withdrawal ID should be correct")
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	}

	withdrawFiatRequest := withdraw.Request{
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
				SupportedCurrencies: "USD",
			},
			WireCurrency:             currency.USD.String(),
			RequiresIntermediaryBank: false,
			IsExpressWire:            false,
		},
		Amount:      10,
		Currency:    currency.USD,
		Description: "WITHDRAW IT ALL",
	}

	w, err := b.WithdrawFiatFunds(t.Context(), &withdrawFiatRequest)
	if mockTests {
		require.NoError(t, err, "WithdrawFiat must not error")
		assert.Equal(t, "1", w.ID, "Withdrawal ID should be correct")
	} else {
		assert.ErrorContains(t, err, "Check your account balance for details")
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	}

	withdrawFiatRequest := withdraw.Request{
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
				SupportedCurrencies: "USD",
			},
			WireCurrency:                  currency.USD.String(),
			RequiresIntermediaryBank:      false,
			IsExpressWire:                 false,
			IntermediaryBankAccountNumber: 12345,
			IntermediaryBankAddress:       "123 Fake St",
			IntermediaryBankCity:          "Tarry Town",
			IntermediaryBankCountry:       "AU",
			IntermediaryBankName:          "Federal Reserve Bank",
			IntermediaryBankPostalCode:    "2088",
		},
		Amount:      50,
		Currency:    currency.USD,
		Description: "WITHDRAW IT ALL",
	}

	w, err := b.WithdrawFiatFundsToInternationalBank(t.Context(),
		&withdrawFiatRequest)
	if mockTests {
		assert.Equal(t, "1", w.ID, "Withdrawal ID should be correct")
	} else {
		require.NoError(t, err, "WithdrawFiatFundsToInternationalBank must not error")
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	}
	a, err := b.GetDepositAddress(t.Context(), currency.XRP, "", "")
	require.NoError(t, err, "GetDepositAddress must not error")
	assert.NotEmpty(t, a.Address, "Address should not be empty")
	assert.NotEmpty(t, a.Tag, "Tag should not be empty")
}

func TestWsSubscription(t *testing.T) {
	pressXToJSON := []byte(`{
		"event": "bts:subscribe",
		"data": {
			"channel": "[channel_name]"
		}
	}`)
	err := b.wsHandleData(t.Context(), pressXToJSON)
	require.NoError(t, err, "TestWsSubscription must not error")
}

func TestWsUnsubscribe(t *testing.T) {
	pressXToJSON := []byte(`{
		"event": "bts:subscribe",
		"data": {
			"channel": "[channel_name]"
		}
	}`)
	err := b.wsHandleData(t.Context(), pressXToJSON)
	require.NoError(t, err, "WsUnsubscribe must not error")
}

func TestWsTrade(t *testing.T) {
	pressXToJSON := []byte(`{"data": {"microtimestamp": "1580336751488517", "amount": 0.00598803, "buy_order_id": 4621328909, "sell_order_id": 4621329035, "amount_str": "0.00598803", "price_str": "9334.73", "timestamp": "1580336751", "price": 9334.73, "type": 1, "id": 104007706}, "event": "trade", "channel": "live_trades_btcusd"}`)
	err := b.wsHandleData(t.Context(), pressXToJSON)
	require.NoError(t, err, "TestWsTrade must not error")
}

func TestWsOrderbook(t *testing.T) {
	pressXToJSON := []byte(`{"data": {"timestamp": "1580336834", "microtimestamp": "1580336834607546", "bids": [["9328.28", "0.05925332"], ["9327.34", "0.43120000"], ["9327.29", "0.63470860"], ["9326.59", "0.41114619"], ["9326.38", "1.06910000"], ["9323.91", "2.67930000"], ["9322.69", "0.80000000"], ["9322.57", "0.03000000"], ["9322.31", "1.36010820"], ["9319.54", "0.03090000"], ["9318.97", "0.28000000"], ["9317.61", "0.02910000"], ["9316.39", "1.08000000"], ["9316.20", "2.00000000"], ["9315.48", "1.00000000"], ["9314.72", "0.11197459"], ["9314.47", "0.32207398"], ["9312.53", "0.03961501"], ["9312.29", "1.00000000"], ["9311.78", "0.03060000"], ["9311.69", "0.32217221"], ["9310.98", "3.29000000"], ["9310.18", "0.01304192"], ["9310.13", "0.02500000"], ["9309.04", "1.00000000"], ["9309.00", "0.05000000"], ["9308.96", "0.03030000"], ["9308.91", "0.32227154"], ["9307.52", "0.32191362"], ["9307.25", "2.44280000"], ["9305.92", "3.00000000"], ["9305.62", "2.37600000"], ["9305.60", "0.21815312"], ["9305.54", "2.80000000"], ["9305.13", "0.05000000"], ["9305.02", "2.90917302"], ["9303.68", "0.02316372"], ["9303.53", "12.55000000"], ["9303.00", "0.02191430"], ["9302.94", "2.38250000"], ["9302.37", "0.01000000"], ["9301.85", "2.50000000"], ["9300.89", "0.02000000"], ["9300.40", "4.10000000"], ["9300.00", "0.33936139"], ["9298.48", "1.45200000"], ["9297.80", "0.42380000"], ["9295.44", "4.54689328"], ["9295.43", "3.20000000"], ["9295.00", "0.28669566"], ["9291.66", "14.09931321"], ["9290.13", "2.87254900"], ["9290.00", "0.67530840"], ["9285.37", "0.38033002"], ["9285.15", "5.37993528"], ["9285.00", "0.09419278"], ["9283.71", "0.15679830"], ["9280.33", "12.55000000"], ["9280.13", "3.20310000"], ["9280.00", "1.36477909"], ["9276.01", "0.00707488"], ["9275.75", "0.56974291"], ["9275.00", "5.88000000"], ["9274.00", "0.00754205"], ["9271.68", "0.01400000"], ["9271.11", "15.37188500"], ["9270.00", "0.06674325"], ["9268.79", "24.54320000"], ["9257.18", "12.55000000"], ["9256.30", "0.17876365"], ["9255.71", "13.82642967"], ["9254.79", "0.96329407"], ["9250.00", "0.78214958"], ["9245.34", "4.90200000"], ["9245.13", "0.10000000"], ["9240.00", "0.44383459"], ["9238.84", "13.16615207"], ["9234.11", "0.43317656"], ["9234.10", "12.55000000"], ["9231.28", "11.79290000"], ["9230.09", "4.15059441"], ["9227.69", "0.00791097"], ["9225.00", "0.44768346"], ["9224.49", "0.85857203"], ["9223.50", "5.61001041"], ["9216.01", "0.03222653"], ["9216.00", "0.05000000"], ["9213.54", "0.71253866"], ["9212.50", "2.86768195"], ["9211.07", "12.55000000"], ["9210.00", "0.54288817"], ["9208.00", "1.00000000"], ["9206.06", "2.62587578"], ["9205.98", "15.40000000"], ["9205.52", "0.01710603"], ["9205.37", "0.03524953"], ["9205.11", "0.15000000"], ["9205.00", "0.01534763"], ["9204.76", "7.00600000"], ["9203.00", "0.01090000"]], "asks": [["9337.10", "0.03000000"], ["9340.85", "2.67820000"], ["9340.95", "0.02900000"], ["9341.17", "1.00000000"], ["9341.41", "2.13966390"], ["9341.61", "0.20000000"], ["9341.97", "0.11199911"], ["9341.98", "3.00000000"], ["9342.26", "0.32112762"], ["9343.87", "1.00000000"], ["9344.17", "3.57250000"], ["9345.04", "0.32103450"], ["9345.41", "4.90000000"], ["9345.69", "1.03000000"], ["9345.80", "0.03000000"], ["9346.00", "0.10200000"], ["9346.69", "0.02397394"], ["9347.41", "1.00000000"], ["9347.82", "0.32094177"], ["9348.23", "0.02880000"], ["9348.62", "11.96287551"], ["9349.31", "2.44270000"], ["9349.47", "0.96000000"], ["9349.86", "4.50000000"], ["9350.37", "0.03300000"], ["9350.57", "0.34682266"], ["9350.60", "0.32085527"], ["9351.45", "0.31147923"], ["9352.31", "0.28000000"], ["9352.86", "9.80000000"], ["9353.73", "0.02360739"], ["9354.00", "0.45000000"], ["9354.12", "0.03000000"], ["9354.29", "3.82446861"], ["9356.20", "0.64000000"], ["9356.90", "0.02316372"], ["9357.30", "2.50000000"], ["9357.70", "2.38240000"], ["9358.92", "6.00000000"], ["9359.97", "0.34898075"], ["9359.98", "2.30000000"], ["9362.56", "2.37600000"], ["9365.00", "0.64000000"], ["9365.16", "1.70030306"], ["9365.27", "3.03000000"], ["9369.99", "2.47102665"], ["9370.00", "3.15688574"], ["9370.21", "2.32720000"], ["9371.78", "13.20000000"], ["9371.89", "0.96293482"], ["9375.08", "4.74762500"], ["9384.34", "1.45200000"], ["9384.49", "16.42310000"], ["9385.66", "0.34382112"], ["9388.19", "0.00268265"], ["9392.20", "0.20980000"], ["9392.40", "0.10320000"], ["9393.00", "0.20980000"], ["9395.40", "0.40000000"], ["9398.86", "24.54310000"], ["9400.00", "0.05489988"], ["9400.33", "0.00495100"], ["9400.45", "0.00484700"], ["9402.92", "17.20000000"], ["9404.18", "10.00000000"], ["9418.89", "16.38000000"], ["9419.41", "3.06700000"], ["9420.40", "12.50000000"], ["9421.11", "0.10500000"], ["9434.47", "0.03215805"], ["9434.48", "0.28285714"], ["9434.49", "15.83000000"], ["9435.13", "0.15000000"], ["9438.93", "0.00368800"], ["9439.19", "0.69343985"], ["9442.86", "0.10000000"], ["9443.96", "12.50000000"], ["9444.00", "0.06004471"], ["9444.97", "0.01494896"], ["9447.00", "0.01234000"], ["9448.97", "0.14500000"], ["9449.00", "0.05000000"], ["9450.00", "11.13426018"], ["9451.87", "15.90000000"], ["9452.00", "0.20000000"], ["9454.25", "0.01100000"], ["9454.51", "0.02409062"], ["9455.05", "0.00600063"], ["9456.00", "0.27965118"], ["9456.10", "0.17000000"], ["9459.00", "0.00320000"], ["9459.98", "0.02460685"], ["9459.99", "8.11000000"], ["9460.00", "0.08500000"], ["9464.36", "0.56957951"], ["9464.54", "0.69158059"], ["9465.00", "21.00002015"], ["9467.57", "12.50000000"], ["9468.00", "0.08800000"], ["9469.09", "13.94000000"]]}, "event": "data", "channel": "order_book_btcusd"}`)
	err := b.wsHandleData(t.Context(), pressXToJSON)
	require.NoError(t, err, "wsHandleData must not error")

	pressXToJSON = []byte(`{"data": {"timestamp": "1580336834", "microtimestamp": "1580336834607546", "bids": [["9328.28", "0.05925332"], ["9327.34", "0.43120000"], ["9327.29", "0.63470860"], ["9326.59", "0.41114619"], ["9326.38", "1.06910000"], ["9323.91", "2.67930000"], ["9322.69", "0.80000000"], ["9322.57", "0.03000000"], ["9322.31", "1.36010820"], ["9319.54", "0.03090000"], ["9318.97", "0.28000000"], ["9317.61", "0.02910000"], ["9316.39", "1.08000000"], ["9316.20", "2.00000000"], ["9315.48", "1.00000000"], ["9314.72", "0.11197459"], ["9314.47", "0.32207398"], ["9312.53", "0.03961501"], ["9312.29", "1.00000000"], ["9311.78", "0.03060000"], ["9311.69", "0.32217221"], ["9310.98", "3.29000000"], ["9310.18", "0.01304192"], ["9310.13", "0.02500000"], ["9309.04", "1.00000000"], ["9309.00", "0.05000000"], ["9308.96", "0.03030000"], ["9308.91", "0.32227154"], ["9307.52", "0.32191362"], ["9307.25", "2.44280000"], ["9305.92", "3.00000000"], ["9305.62", "2.37600000"], ["9305.60", "0.21815312"], ["9305.54", "2.80000000"], ["9305.13", "0.05000000"], ["9305.02", "2.90917302"], ["9303.68", "0.02316372"], ["9303.53", "12.55000000"], ["9303.00", "0.02191430"], ["9302.94", "2.38250000"], ["9302.37", "0.01000000"], ["9301.85", "2.50000000"], ["9300.89", "0.02000000"], ["9300.40", "4.10000000"], ["9300.00", "0.33936139"], ["9298.48", "1.45200000"], ["9297.80", "0.42380000"], ["9295.44", "4.54689328"], ["9295.43", "3.20000000"], ["9295.00", "0.28669566"], ["9291.66", "14.09931321"], ["9290.13", "2.87254900"], ["9290.00", "0.67530840"], ["9285.37", "0.38033002"], ["9285.15", "5.37993528"], ["9285.00", "0.09419278"], ["9283.71", "0.15679830"], ["9280.33", "12.55000000"], ["9280.13", "3.20310000"], ["9280.00", "1.36477909"], ["9276.01", "0.00707488"], ["9275.75", "0.56974291"], ["9275.00", "5.88000000"], ["9274.00", "0.00754205"], ["9271.68", "0.01400000"], ["9271.11", "15.37188500"], ["9270.00", "0.06674325"], ["9268.79", "24.54320000"], ["9257.18", "12.55000000"], ["9256.30", "0.17876365"], ["9255.71", "13.82642967"], ["9254.79", "0.96329407"], ["9250.00", "0.78214958"], ["9245.34", "4.90200000"], ["9245.13", "0.10000000"], ["9240.00", "0.44383459"], ["9238.84", "13.16615207"], ["9234.11", "0.43317656"], ["9234.10", "12.55000000"], ["9231.28", "11.79290000"], ["9230.09", "4.15059441"], ["9227.69", "0.00791097"], ["9225.00", "0.44768346"], ["9224.49", "0.85857203"], ["9223.50", "5.61001041"], ["9216.01", "0.03222653"], ["9216.00", "0.05000000"], ["9213.54", "0.71253866"], ["9212.50", "2.86768195"], ["9211.07", "12.55000000"], ["9210.00", "0.54288817"], ["9208.00", "1.00000000"], ["9206.06", "2.62587578"], ["9205.98", "15.40000000"], ["9205.52", "0.01710603"], ["9205.37", "0.03524953"], ["9205.11", "0.15000000"], ["9205.00", "0.01534763"], ["9204.76", "7.00600000"], ["9203.00", "0.01090000"]], "asks": [["9337.10", "0.03000000"], ["9340.85", "2.67820000"], ["9340.95", "0.02900000"], ["9341.17", "1.00000000"], ["9341.41", "2.13966390"], ["9341.61", "0.20000000"], ["9341.97", "0.11199911"], ["9341.98", "3.00000000"], ["9342.26", "0.32112762"], ["9343.87", "1.00000000"], ["9344.17", "3.57250000"], ["9345.04", "0.32103450"], ["9345.41", "4.90000000"], ["9345.69", "1.03000000"], ["9345.80", "0.03000000"], ["9346.00", "0.10200000"], ["9346.69", "0.02397394"], ["9347.41", "1.00000000"], ["9347.82", "0.32094177"], ["9348.23", "0.02880000"], ["9348.62", "11.96287551"], ["9349.31", "2.44270000"], ["9349.47", "0.96000000"], ["9349.86", "4.50000000"], ["9350.37", "0.03300000"], ["9350.57", "0.34682266"], ["9350.60", "0.32085527"], ["9351.45", "0.31147923"], ["9352.31", "0.28000000"], ["9352.86", "9.80000000"], ["9353.73", "0.02360739"], ["9354.00", "0.45000000"], ["9354.12", "0.03000000"], ["9354.29", "3.82446861"], ["9356.20", "0.64000000"], ["9356.90", "0.02316372"], ["9357.30", "2.50000000"], ["9357.70", "2.38240000"], ["9358.92", "6.00000000"], ["9359.97", "0.34898075"], ["9359.98", "2.30000000"], ["9362.56", "2.37600000"], ["9365.00", "0.64000000"], ["9365.16", "1.70030306"], ["9365.27", "3.03000000"], ["9369.99", "2.47102665"], ["9370.00", "3.15688574"], ["9370.21", "2.32720000"], ["9371.78", "13.20000000"], ["9371.89", "0.96293482"], ["9375.08", "4.74762500"], ["9384.34", "1.45200000"], ["9384.49", "16.42310000"], ["9385.66", "0.34382112"], ["9388.19", "0.00268265"], ["9392.20", "0.20980000"], ["9392.40", "0.10320000"], ["9393.00", "0.20980000"], ["9395.40", "0.40000000"], ["9398.86", "24.54310000"], ["9400.00", "0.05489988"], ["9400.33", "0.00495100"], ["9400.45", "0.00484700"], ["9402.92", "17.20000000"], ["9404.18", "10.00000000"], ["9418.89", "16.38000000"], ["9419.41", "3.06700000"], ["9420.40", "12.50000000"], ["9421.11", "0.10500000"], ["9434.47", "0.03215805"], ["9434.48", "0.28285714"], ["9434.49", "15.83000000"], ["9435.13", "0.15000000"], ["9438.93", "0.00368800"], ["9439.19", "0.69343985"], ["9442.86", "0.10000000"], ["9443.96", "12.50000000"], ["9444.00", "0.06004471"], ["9444.97", "0.01494896"], ["9447.00", "0.01234000"], ["9448.97", "0.14500000"], ["9449.00", "0.05000000"], ["9450.00", "11.13426018"], ["9451.87", "15.90000000"], ["9452.00", "0.20000000"], ["9454.25", "0.01100000"], ["9454.51", "0.02409062"], ["9455.05", "0.00600063"], ["9456.00", "0.27965118"], ["9456.10", "0.17000000"], ["9459.00", "0.00320000"], ["9459.98", "0.02460685"], ["9459.99", "8.11000000"], ["9460.00", "0.08500000"], ["9464.36", "0.56957951"], ["9464.54", "0.69158059"], ["9465.00", "21.00002015"], ["9467.57", "12.50000000"], ["9468.00", "0.08800000"], ["9469.09", "13.94000000"]]}, "event": "data", "channel": ""}`)
	err = b.wsHandleData(t.Context(), pressXToJSON)
	require.ErrorIs(t, err, errChannelUnderscores, "wsHandleData must error parsing channel")
}

func TestWsOrderbook2(t *testing.T) {
	pressXToJSON := []byte(`{"data":{"timestamp":"1606965727","microtimestamp":"1606965727403931","bids":[["19133.97","0.01000000"],["19131.58","0.39200000"],["19131.18","0.69581810"],["19131.17","0.48139054"],["19129.72","0.48164130"],["19129.71","0.65400000"],["19128.80","1.04500000"],["19128.59","0.65400000"],["19128.12","0.00259236"],["19127.81","0.19784245"],["19126.66","1.04500000"],["19125.74","0.26020000"],["19124.68","0.22000000"],["19122.01","0.39777840"],["19122.00","1.04600000"],["19121.27","0.16741000"],["19121.10","1.56390000"],["19119.90","1.60000000"],["19119.58","0.15593238"],["19117.70","1.14600000"],["19115.36","2.61300000"],["19114.60","1.19570000"],["19113.88","0.07500000"],["19113.86","0.15668522"],["19113.70","1.00000000"],["19113.69","1.60000000"],["19112.27","0.00166667"],["19111.00","0.15464628"],["19108.80","0.70000000"],["19108.77","0.16300000"],["19108.38","1.10000000"],["19107.53","0.10000000"],["19106.83","0.21377991"],["19106.78","3.45938881"],["19104.24","1.30000000"],["19100.81","0.00166667"],["19100.21","0.49770000"],["19099.54","2.40971961"],["19099.53","0.51223189"],["19097.40","1.55000000"],["19095.55","2.61300000"],["19092.94","0.27402906"],["19092.20","1.60000000"],["19089.36","0.00166667"],["19086.32","1.62000000"],["19085.23","1.65670000"],["19080.88","1.40000000"],["19075.45","1.16000000"],["19071.24","1.20000000"],["19065.09","1.51000000"],["19059.38","1.57000000"],["19058.11","0.37393556"],["19052.98","0.01000000"],["19052.90","0.33000000"],["19049.55","6.89000000"],["19047.61","6.03623432"],["19030.16","16.60260000"],["19026.76","23.90800000"],["19024.78","2.16656212"],["19022.11","0.02628500"],["19020.37","6.03000000"],["19000.00","0.00132020"],["18993.52","2.22000000"],["18979.21","6.03240000"],["18970.20","0.01500000"],["18969.14","7.42000000"],["18956.46","6.03240000"],["18950.22","42.37500000"],["18950.00","0.00132019"],["18949.94","0.52650000"],["18946.00","0.00791700"],["18933.74","6.03240000"],["18932.21","8.21000000"],["18926.99","0.00150000"],["18926.98","0.02641500"],["18925.00","0.02000000"],["18909.99","0.00133000"],["18908.47","7.15000000"],["18905.99","0.00133000"],["18905.20","0.00190000"],["18901.00","0.10000000"],["18900.67","0.24430000"],["18900.00","7.56529933"],["18895.99","0.00178450"],["18890.00","0.10000000"],["18889.90","0.10580000"],["18888.00","0.00362564"],["18887.00","4.00000000"],["18881.62","0.20583403"],["18880.08","5.72198740"],["18880.05","8.33480000"],["18879.09","7.33000000"],["18875.99","0.00132450"],["18875.00","0.02000000"],["18873.47","0.25934200"],["18871.99","0.00132600"],["18870.93","0.36463225"],["18864.10","43.56800000"],["18853.11","0.00540000"],["18850.01","0.38925549"]],"asks":[["19141.75","0.39300000"],["19141.78","0.10204700"],["19143.05","1.99685100"],["19143.08","0.05777900"],["19143.09","1.60700800"],["19143.10","0.48282909"],["19143.36","0.11250000"],["19144.06","0.26040000"],["19145.97","0.65400000"],["19146.02","0.22000000"],["19146.56","0.45061841"],["19147.45","0.15877831"],["19148.92","0.70431840"],["19148.93","0.78400000"],["19150.32","0.78400000"],["19151.55","0.07500000"],["19152.64","3.11400000"],["19153.32","1.04600000"],["19153.84","0.15626630"],["19155.57","3.10000000"],["19156.40","0.13438213"],["19156.92","0.16300000"],["19157.54","1.38970000"],["19158.18","0.00166667"],["19158.41","0.15317000"],["19158.78","0.15888798"],["19160.14","0.10000000"],["19160.34","1.60000000"],["19160.70","1.21590000"],["19162.17","0.00352761"],["19162.67","1.04500000"],["19163.61","0.15000000"],["19163.80","1.18050000"],["19164.62","0.86919692"],["19165.36","0.15674424"],["19166.75","1.40000000"],["19167.47","2.61300000"],["19169.68","0.00166667"],["19171.08","0.15452025"],["19171.69","0.54308236"],["19172.12","0.49000000"],["19173.47","1.34000000"],["19174.49","1.07436448"],["19175.37","0.01200000"],["19178.25","1.50000000"],["19178.80","0.49770000"],["19181.18","0.00166667"],["19182.75","1.77297176"],["19182.76","2.61099999"],["19183.03","1.20000000"],["19185.17","6.00352761"],["19189.56","0.05797137"],["19189.72","1.17000000"],["19193.94","1.60000000"],["19197.15","0.26961100"],["19200.00","0.03107838"],["19200.06","1.29000000"],["19202.73","1.65670000"],["19206.06","1.30000000"],["19208.19","6.00352761"],["19209.00","0.00132021"],["19210.70","1.20000000"],["19213.77","0.02615500"],["19217.40","8.50000000"],["19217.57","1.29000000"],["19222.61","1.19000000"],["19230.00","0.00193480"],["19231.24","6.00000000"],["19237.91","6.89152278"],["19240.13","6.90000000"],["19242.16","0.00336000"],["19243.38","0.00299103"],["19244.48","14.79300000"],["19248.25","0.01300000"],["19250.00","1.95802492"],["19251.00","0.45000000"],["19254.20","0.00366102"],["19254.32","6.00000000"],["19259.00","0.00131022"],["19266.43","0.00917191"],["19267.63","0.05000000"],["19267.79","7.10000000"],["19268.72","16.60260000"],["19277.42","6.00000000"],["19286.64","0.00916230"],["19295.49","7.77000000"],["19300.00","0.19668172"],["19306.00","0.06000000"],["19307.00","3.00000000"],["19307.40","0.19000000"],["19309.00","0.00262046"],["19310.33","0.02602500"],["19319.33","0.00213688"],["19320.00","0.00171242"],["19321.02","48.47300000"],["19322.74","0.00250000"],["19324.00","0.36983571"],["19325.54","0.02314521"],["19325.73","7.22000000"],["19326.50","0.00915272"]]},"channel":"order_book_btcusd","event":"data"}`)
	err := b.wsHandleData(t.Context(), pressXToJSON)
	require.NoError(t, err, "WsOrderbook2 must not error")
}

func TestWsOrderUpdate(t *testing.T) {
	t.Parallel()

	b := new(Bitstamp) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(b), "Test instance Setup must not error")
	testexch.FixtureToDataHandler(t, "testdata/wsMyOrders.json", b.wsHandleData)
	close(b.Websocket.DataHandler)
	assert.Len(t, b.Websocket.DataHandler, 8, "Should see 8 orders")
	for resp := range b.Websocket.DataHandler {
		switch v := resp.(type) {
		case *order.Detail:
			switch len(b.Websocket.DataHandler) {
			case 7:
				assert.Equal(t, "1658864794234880", v.OrderID, "OrderID")
				assert.Equal(t, time.UnixMicro(1693831262313000), v.Date, "Date")
				assert.Equal(t, "test_market_buy", v.ClientOrderID, "ClientOrderID")
				assert.Equal(t, order.New, v.Status, "Status")
				assert.Equal(t, order.Buy, v.Side, "Side")
				assert.Equal(t, asset.Spot, v.AssetType, "AssetType")
				assert.Equal(t, currency.NewPairWithDelimiter("BTC", "USD", "/"), v.Pair, "Pair")
				assert.Equal(t, 0.0, v.ExecutedAmount, "ExecutedAmount")
				assert.Equal(t, 999999999.0, v.Price, "Price") // Market Buy Price
				// Note: Amount is 0 for market order create messages, oddly
			case 6:
				assert.Equal(t, "1658864794234880", v.OrderID, "OrderID")
				assert.Equal(t, order.PartiallyFilled, v.Status, "Status")
				assert.Equal(t, 0.00038667, v.Amount, "Amount")
				assert.Equal(t, 0.00000001, v.RemainingAmount, "RemainingAmount") // During live tests we consistently got back this Sat remaining
				assert.Equal(t, 0.00038666, v.ExecutedAmount, "ExecutedAmount")
				assert.Equal(t, 25862.0, v.Price, "Price")
			case 5:
				assert.Equal(t, "1658864794234880", v.OrderID, "OrderID")
				assert.Equal(t, order.Cancelled, v.Status, "Status") // Even though they probably consider it filled, Deleted + PartialFill = Cancelled
				assert.Equal(t, 0.00038667, v.Amount, "Amount")
				assert.Equal(t, 0.00000001, v.RemainingAmount, "RemainingAmount")
				assert.Equal(t, 0.00038666, v.ExecutedAmount, "ExecutedAmount")
				assert.Equal(t, 25862.0, v.Price, "Price")
			case 4:
				assert.Equal(t, "1658870500933632", v.OrderID, "OrderID")
				assert.Equal(t, order.New, v.Status, "Status")
				assert.Equal(t, order.Sell, v.Side, "Side")
				assert.Equal(t, 0.0, v.Price, "Price") // Market Sell Price
			case 3:
				assert.Equal(t, "1658870500933632", v.OrderID, "OrderID")
				assert.Equal(t, order.PartiallyFilled, v.Status, "Status")
				assert.Equal(t, 0.00038679, v.Amount, "Amount")
				assert.Equal(t, 0.00000001, v.RemainingAmount, "RemainingAmount")
				assert.Equal(t, 0.00038678, v.ExecutedAmount, "ExecutedAmount")
				assert.Equal(t, 25854.0, v.Price, "Price")
			case 2:
				assert.Equal(t, "1658870500933632", v.OrderID, "OrderID")
				assert.Equal(t, order.Cancelled, v.Status, "Status")
				assert.Equal(t, 0.00038679, v.Amount, "Amount")
				assert.Equal(t, 0.00000001, v.RemainingAmount, "RemainingAmount")
				assert.Equal(t, 0.00038678, v.ExecutedAmount, "ExecutedAmount")
				assert.Equal(t, 25854.0, v.Price, "Price")
			case 1:
				assert.Equal(t, "1658869033291777", v.OrderID, "OrderID")
				assert.Equal(t, order.New, v.Status, "Status")
				assert.Equal(t, order.Sell, v.Side, "Side")
				assert.Equal(t, 25845.0, v.Price, "Price")
				assert.Equal(t, 0.00038692, v.Amount, "Amount")
			case 0:
				assert.Equal(t, "1658869033291777", v.OrderID, "OrderID")
				assert.Equal(t, order.Filled, v.Status, "Status")
				assert.Equal(t, 25845.0, v.Price, "Price")
				assert.Equal(t, 0.00038692, v.Amount, "Amount")
				assert.Equal(t, 0.0, v.RemainingAmount, "RemainingAmount")
				assert.Equal(t, 0.00038692, v.ExecutedAmount, "ExecutedAmount")
			}
		case error:
			t.Error(v)
		default:
			t.Errorf("Got unexpected data: %T %v", v, v)
		}
	}
}

func TestWsRequestReconnect(t *testing.T) {
	pressXToJSON := []byte(`{
		"event": "bts:request_reconnect",
		"channel": "",
		"data": ""
	}`)
	err := b.wsHandleData(t.Context(), pressXToJSON)
	require.NoError(t, err, "WsRequestReconnect must not error")
}

func TestOHLC(t *testing.T) {
	t.Parallel()
	o, err := b.OHLC(t.Context(), "btcusd", time.Unix(1546300800, 0), time.Unix(1577836799, 0), "60", "10")
	require.NoError(t, err, "OHLC must not error")
	assert.Equal(t, "BTC/USD", o.Data.Pair, "Pair should be correct")
	for _, req := range o.Data.OHLCV {
		assert.Positive(t, req.Low, "Low should be positive")
		assert.Positive(t, req.Close, "Close should be positive")
		assert.Positive(t, req.Open, "Open should be positive")
		assert.Positive(t, req.Volume, "Volume should be positive")
		assert.NotEmpty(t, req.Timestamp, "Timestamp should not be empty")
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	c, err := b.GetHistoricCandles(t.Context(), btcusdPair, asset.Spot, kline.OneDay, time.Unix(1546300800, 0), time.Unix(1577836799, 0))
	require.NoError(t, err, "GetHistoricCandles must not error")
	assert.Equal(t, btcusdPair, c.Pair, "Pair should be correct")
	assert.NotEmpty(t, c, "Candles should not be empty")
	for _, req := range c.Candles {
		assert.Positive(t, req.High, "High should be positive")
		assert.Positive(t, req.Low, "Low should be positive")
		assert.Positive(t, req.Close, "Close should be positive")
		assert.Positive(t, req.Open, "Open should be positive")
		assert.Positive(t, req.Volume, "Volume should be positive")
		assert.NotEmpty(t, req.Time, "Time should not be empty")
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	c, err := b.GetHistoricCandlesExtended(t.Context(), btcusdPair, asset.Spot, kline.OneDay, time.Unix(1546300800, 0), time.Unix(1577836799, 0))
	require.NoError(t, err, "GetHistoricCandlesExtended must not error")
	assert.Equal(t, btcusdPair, c.Pair, "Pair should be correct")
	assert.NotEmpty(t, c, "Candles should not be empty")
	for _, req := range c.Candles {
		assert.Positive(t, req.High, "High should be positive")
		assert.Positive(t, req.Low, "Low should be positive")
		assert.Positive(t, req.Close, "Close should be positive")
		assert.Positive(t, req.Open, "Open should be positive")
		assert.Positive(t, req.Volume, "Volume should be positive")
		assert.NotEmpty(t, req.Time, "Time should not be empty")
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()

	currencyPair, err := currency.NewPairFromString("LTCUSD")
	require.NoError(t, err, "NewPairFromString must not error")

	tr, err := b.GetRecentTrades(t.Context(), currencyPair, asset.Spot)
	require.NoError(t, err, "GetRecentTrades must not error")
	assert.NotEmpty(t, tr, "Trades should not be empty")
	for _, req := range tr {
		assert.Positive(t, req.Amount, "Amount should be positive")
		assert.Equal(t, currency.NewPairWithDelimiter("ltc", "usd", ""), req.CurrencyPair, "Pair should be correct")
		assert.Equal(t, asset.Spot, req.AssetType, "AssetType should be set")
		assert.NotEmpty(t, req.Timestamp, "Timestamp should not be empty")
		assert.Positive(t, req.Price, "Price should be positive")
		assert.NotEmpty(t, req.TID, "TID should not be empty")
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()

	currencyPair, err := currency.NewPairFromString("LTCUSD")
	require.NoError(t, err, "NewPairFromString must not error")
	_, err = b.GetHistoricTrades(t.Context(),
		currencyPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestOrderbookZeroBidPrice(t *testing.T) {
	t.Parallel()

	ob := &orderbook.Book{
		Exchange: "Bitstamp",
		Pair:     btcusdPair,
		Asset:    asset.Spot,
	}
	filterOrderbookZeroBidPrice(ob)

	ob.Bids = orderbook.Levels{
		{Price: 69, Amount: 1337},
		{Price: 0, Amount: 69},
	}
	filterOrderbookZeroBidPrice(ob)
	if ob.Bids[0].Price != 69 || ob.Bids[0].Amount != 1337 || len(ob.Bids) != 1 {
		t.Error("invalid orderbook bid values")
	}

	ob.Bids = orderbook.Levels{
		{Price: 59, Amount: 1337},
		{Price: 42, Amount: 8595},
	}
	filterOrderbookZeroBidPrice(ob)
	if ob.Bids[0].Price != 59 || ob.Bids[0].Amount != 1337 ||
		ob.Bids[1].Price != 42 || ob.Bids[1].Amount != 8595 || len(ob.Bids) != 2 {
		t.Error("invalid orderbook bid values")
	}
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	h, err := b.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	require.NoError(t, err, "GetWithdrawalsHistory must not error")
	if mockTests {
		assert.NotEmpty(t, h, "WithdrawalHistory should not be empty")
		for _, req := range h {
			assert.Equal(t, time.Date(2022, time.January, 31, 16, 7, 32, 0, time.UTC), req.Timestamp, "Timestamp should match")
			assert.Equal(t, "BTC", req.Currency, "Currency should match")
			assert.Equal(t, 0.00006000, req.Amount, "Amount should match")
		}
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	o, err := b.GetOrderInfo(t.Context(), "1458532827766784", btcusdPair, asset.Spot)
	if mockTests {
		require.NoError(t, err, "GetOrderInfo must not error")
		assert.Equal(t, time.Date(2022, time.January, 31, 14, 43, 15, 0, time.UTC), o.Date, "Date should match")
		assert.Equal(t, "1458532827766784", o.OrderID, "OrderID should match")
		assert.Equal(t, order.Open, o.Status, "Status should match")
		assert.Equal(t, 200.00, o.RemainingAmount, "RemainingAmount should match")
		for _, tr := range o.Trades {
			assert.Equal(t, 50.00, tr.Price, "Price should match")
		}
	} else {
		assert.ErrorContains(t, err, "authenticated request failed Order not found")
	}
}

func TestFetchWSAuth(t *testing.T) {
	t.Parallel()

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	resp, err := b.FetchWSAuth(t.Context())
	require.NoError(t, err, "FetchWSAuth must not error")
	assert.NotNil(t, resp, "resp should not be nil")
	assert.Positive(t, resp.UserID, "UserID should be positive")
	assert.Len(t, resp.Token, 32, "Token should be 32 chars")
	assert.Positive(t, resp.ValidSecs, "ValidSecs should be positive")
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, b)
	for _, a := range b.GetAssetTypes(false) {
		pairs, err := b.CurrencyPairs.GetPairs(a, false)
		require.NoErrorf(t, err, "cannot get pairs for %s", a)
		require.NotEmptyf(t, pairs, "no pairs for %s", a)
		resp, err := b.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()
	b.Websocket.SetCanUseAuthenticatedEndpoints(true)
	require.True(t, b.Websocket.CanUseAuthenticatedEndpoints(), "CanUseAuthenticatedEndpoints must return true")
	subs, err := b.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	exp := subscription.List{}
	pairs, err := b.GetEnabledPairs(asset.Spot)
	require.NoError(t, err, "GetEnabledPairs must not error")
	for _, baseSub := range b.Features.Subscriptions {
		for _, p := range pairs.Format(currency.PairFormat{Uppercase: false}) {
			s := baseSub.Clone()
			s.Pairs = currency.Pairs{p}
			s.QualifiedChannel = channelName(s) + "_" + p.String()
			exp = append(exp, s)
		}
	}
	testsubs.EqualLists(t, exp, subs)
	assert.PanicsWithError(t,
		"subscription channel not supported: wibble",
		func() { channelName(&subscription.Subscription{Channel: "wibble"}) },
		"should panic on invalid channel",
	)
}

func TestSubscribe(t *testing.T) {
	t.Parallel()
	b := new(Bitstamp)
	require.NoError(t, testexch.Setup(b), "Test instance Setup must not error")
	subs, err := b.Features.Subscriptions.ExpandTemplates(b)
	require.NoError(t, err, "ExpandTemplates must not error")
	b.Features.Subscriptions = subscription.List{}
	testexch.SetupWs(t, b)
	err = b.Subscribe(subs)
	require.NoError(t, err, "Subscribe must not error")
	for _, s := range subs {
		assert.Equalf(t, subscription.SubscribedState, s.State(), "Subscription %s should be subscribed", s)
	}
	err = b.Unsubscribe(subs)
	require.NoError(t, err, "UnSubscribe must not error")
	for _, s := range subs {
		assert.Equalf(t, subscription.UnsubscribedState, s.State(), "Subscription %s should be subscribed", s)
	}
}
