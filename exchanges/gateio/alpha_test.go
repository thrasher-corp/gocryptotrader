package gateio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// alphaTestCurrency is a mixed-case alpha symbol, as Gate matches alpha symbols case sensitively
const alphaTestCurrency = "memeboxELON"

func TestAlphaStatusErrorAsError(t *testing.T) {
	t.Parallel()
	assert.NoError(t, (&alphaStatusError{}).AsError(), "an empty label should not error")
	err := (&alphaStatusError{Label: "BALANCE_NOT_ENOUGH", Message: "insufficient balance"}).AsError()
	require.Error(t, err, "a label must error")
	assert.ErrorContains(t, err, "BALANCE_NOT_ENOUGH", "error should carry the label")
	assert.ErrorContains(t, err, "insufficient balance", "error should carry the message")
}

func TestGetAlphaAccounts(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAlphaAccounts(t.Context())
	require.NoError(t, err, "GetAlphaAccounts must not error")
	if mockTests {
		exp := []*AlphaAccountResponse{{Currency: alphaTestCurrency, Available: 1, TokenAddress: "0x6952c5408b", Chain: "SOL"}}
		assert.Equal(t, exp, result, "GetAlphaAccounts should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAlphaAccounts should return accounts")
}

func TestGetAlphaAccountTransactionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetAlphaAccountTransactionHistory(t.Context(), time.Time{}, time.Time{}, 1, 10)
	require.ErrorIs(t, err, errStartTimeRequired, "GetAlphaAccountTransactionHistory must require a start time")
	startTime, endTime := getTime()
	_, err = e.GetAlphaAccountTransactionHistory(t.Context(), endTime, startTime, 1, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAlphaAccountTransactionHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAlphaAccountTransactionHistory(t.Context(), startTime, endTime, 1, 10)
	require.NoError(t, err, "GetAlphaAccountTransactionHistory must not error")
	if mockTests {
		exp := []*AlphaAccountTransactionItemResponse{{
			ID:       "123456",
			Time:     types.Time(time.Unix(1747827868, 0)),
			Currency: alphaTestCurrency,
			Change:   1.03,
			Balance:  4.59316525194,
		}}
		assert.Equal(t, exp, result, "GetAlphaAccountTransactionHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAlphaAccountTransactionHistory should return transactions")
}

func TestCreateAlphaCurrencyQuoteID(t *testing.T) {
	t.Parallel()
	_, err := e.CreateAlphaCurrencyQuoteID(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateAlphaCurrencyQuoteID must reject a nil request")
	_, err = e.CreateAlphaCurrencyQuoteID(t.Context(), &AlphaCurrencyQuoteInfoRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateAlphaCurrencyQuoteID must require a currency")

	arg := &AlphaCurrencyQuoteInfoRequest{Currency: alphaTestCurrency}
	_, err = e.CreateAlphaCurrencyQuoteID(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "CreateAlphaCurrencyQuoteID must require a side")
	arg.Side = order.Buy
	_, err = e.CreateAlphaCurrencyQuoteID(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "CreateAlphaCurrencyQuoteID must require an amount")
	arg.Amount = 324
	_, err = e.CreateAlphaCurrencyQuoteID(t.Context(), arg)
	require.ErrorIs(t, err, errGasModeRequired, "CreateAlphaCurrencyQuoteID must require a gas mode")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.GasMode = "custom"
	arg.Slippage = 10
	result, err := e.CreateAlphaCurrencyQuoteID(t.Context(), arg)
	require.NoError(t, err, "CreateAlphaCurrencyQuoteID must not error")
	if !mockTests {
		assert.NotEmpty(t, result.QuoteID, "CreateAlphaCurrencyQuoteID should return a quote ID")
		return
	}
	exp := &AlphaCurrencyQuoteDetailResponse{
		QuoteID:                  "12345678",
		MinAmount:                0.1,
		MaxAmount:                "1000:0.0",
		Price:                    11.666,
		Slippage:                 10,
		TargetTokenMinAmount:     500.6,
		TargetTokenMaxAmount:     666.6,
		EstimateGasFeeAmountUSDT: DisplayNumber{Text: "$0.04", Value: 0.04},
		OrderFee:                 DisplayNumber{Text: "$0"},
	}
	assert.Equal(t, exp, result, "CreateAlphaCurrencyQuoteID should decode every field")

	_, err = e.CreateAlphaCurrencyQuoteID(t.Context(), &AlphaCurrencyQuoteInfoRequest{Currency: alphaTestCurrency, Side: order.Sell, Amount: 1, GasMode: "speed"})
	assert.ErrorContains(t, err, "BALANCE_NOT_ENOUGH", "CreateAlphaCurrencyQuoteID should return the response's error label")
}

func TestPlaceAlphaTradeOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceAlphaTradeOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceAlphaTradeOrder must reject a nil request")
	_, err = e.PlaceAlphaTradeOrder(t.Context(), &AlphaCurrencyQuoteInfoRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PlaceAlphaTradeOrder must require a currency")

	arg := &AlphaCurrencyQuoteInfoRequest{Currency: alphaTestCurrency}
	_, err = e.PlaceAlphaTradeOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "PlaceAlphaTradeOrder must require a side")
	arg.Side = order.Buy
	_, err = e.PlaceAlphaTradeOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "PlaceAlphaTradeOrder must require an amount")
	arg.Amount = 324
	_, err = e.PlaceAlphaTradeOrder(t.Context(), arg)
	require.ErrorIs(t, err, errGasModeRequired, "PlaceAlphaTradeOrder must require a gas mode")
	arg.GasMode = "custom"
	arg.Slippage = 10
	_, err = e.PlaceAlphaTradeOrder(t.Context(), arg)
	require.ErrorIs(t, err, errQuoteIDRequired, "PlaceAlphaTradeOrder must require a quote ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.QuoteID = "12345678"
	result, err := e.PlaceAlphaTradeOrder(t.Context(), arg)
	require.NoError(t, err, "PlaceAlphaTradeOrder must not error")
	if mockTests {
		exp := &AlphaPlaceOrderResponse{
			OrderID:      "12345678",
			Status:       1,
			Side:         "buy",
			GasMode:      "custom",
			CreateTime:   types.Time(time.Unix(1749468580, 0)),
			Amount:       324,
			TokenAddress: "0x6952c5408b",
			Chain:        "ETH",
		}
		assert.Equal(t, exp, result, "PlaceAlphaTradeOrder should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderID, "PlaceAlphaTradeOrder should return an order ID")
}

func TestGetAlphaOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetAlphaOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAlphaOrders must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetAlphaOrders(t.Context(), &GetAlphaOrdersRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAlphaOrders must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAlphaOrders(t.Context(), &GetAlphaOrdersRequest{
		Currency:  alphaTestCurrency,
		Side:      order.Buy,
		Status:    2,
		StartTime: startTime,
		EndTime:   endTime,
		Page:      1,
		Limit:     10,
	})
	require.NoError(t, err, "GetAlphaOrders must not error")
	if mockTests {
		exp := []*AlphaOrderDetailResponse{{
			OrderID:         "12345678",
			TransactionHash: "0xaaaaaaa",
			Side:            "buy",
			USDTAmount:      324,
			Currency:        alphaTestCurrency,
			CurrencyAmount:  565455643.64,
			Status:          2,
			GasMode:         "custom",
			Chain:           "ETH",
			GasFee:          0.3,
			TransactionFee:  0.1,
			CreateTime:      types.Time(time.Unix(1742972931, 0)),
		}}
		assert.Equal(t, exp, result, "GetAlphaOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAlphaOrders should return orders")
}

func TestGetAlphaOrderByID(t *testing.T) {
	t.Parallel()
	_, err := e.GetAlphaOrderByID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetAlphaOrderByID must require an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAlphaOrderByID(t.Context(), "12345678")
	require.NoError(t, err, "GetAlphaOrderByID must not error")
	if mockTests {
		exp := &AlphaOrderDetailResponse{
			OrderID:         "12345678",
			TransactionHash: "0xaaaaaaa",
			Side:            "sell",
			Currency:        alphaTestCurrency,
			CurrencyAmount:  565455643.64,
			Status:          3,
			GasMode:         "speed",
			Chain:           "ETH",
			GasFee:          0.3,
			CreateTime:      types.Time(time.Unix(1742972931, 0)),
			FailedReason:    "slippage exceeded",
		}
		assert.Equal(t, exp, result, "GetAlphaOrderByID should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAlphaOrderByID should return the order")
}

func TestGetAlphaCurrenciesDetail(t *testing.T) {
	t.Parallel()
	result, err := e.GetAlphaCurrenciesDetail(t.Context(), "", 100, 10)
	require.NoError(t, err, "GetAlphaCurrenciesDetail must not error")
	require.NotEmpty(t, result, "GetAlphaCurrenciesDetail must return currencies")
	if mockTests {
		require.Len(t, result, 5, "GetAlphaCurrenciesDetail must return every recorded currency")
		exp := &AlphaCurrencyDetailResponse{Currency: "memeboxacs", Name: "ACS", Chain: "SOLANA", Address: "5MAYDfq5yxtudAhtfyuMBuHZjgAbaS9tbEyEQYAhDS5y", Status: 1, Precision: 7, AmountPrecision: 2}
		assert.Equal(t, exp, result[0], "GetAlphaCurrenciesDetail should decode every field")
	}

	// Gate matches alpha symbols case sensitively, so a mixed-case symbol must be sent unchanged
	result, err = e.GetAlphaCurrenciesDetail(t.Context(), "memeboxTUT", 0, 0)
	require.NoError(t, err, "GetAlphaCurrenciesDetail must not error for a mixed-case symbol")
	require.Len(t, result, 1, "GetAlphaCurrenciesDetail must return the requested currency")
	assert.Equal(t, "memeboxTUT", result[0].Currency, "currency symbol should keep its case")
	if mockTests {
		exp := &AlphaCurrencyDetailResponse{Currency: "memeboxTUT", Name: "TUT", Chain: "BSC", Address: "0xcaae2a2f939f51d97cdfa9a86e79e3f085b799f3", Status: 1, Precision: 5, AmountPrecision: 4}
		assert.Equal(t, exp, result[0], "GetAlphaCurrenciesDetail should decode every field")
	}
}

func TestGetAlphaCurrencyTicker(t *testing.T) {
	t.Parallel()
	result, err := e.GetAlphaCurrencyTicker(t.Context(), "", 100, 10)
	require.NoError(t, err, "GetAlphaCurrencyTicker must not error")
	require.NotEmpty(t, result, "GetAlphaCurrencyTicker must return tickers")
	if mockTests {
		require.Len(t, result, 5, "GetAlphaCurrencyTicker must return every recorded ticker")
		exp := &AlphaCurrencyTickerInfoResponse{Currency: "memeboxnura", PercentageChange24Hour: 0.27, Last: 0.00001617, USDTVolume24Hour: 2379.22, MarketCap: 144024.57}
		assert.Equal(t, exp, result[0], "GetAlphaCurrencyTicker should decode every field")
	}

	// Gate matches alpha symbols case sensitively, so a mixed-case symbol must be sent unchanged
	result, err = e.GetAlphaCurrencyTicker(t.Context(), "memeboxTUT", 0, 0)
	require.NoError(t, err, "GetAlphaCurrencyTicker must not error for a mixed-case symbol")
	require.Len(t, result, 1, "GetAlphaCurrencyTicker must return the requested ticker")
	assert.Equal(t, "memeboxTUT", result[0].Currency, "currency symbol should keep its case")
	if mockTests {
		exp := &AlphaCurrencyTickerInfoResponse{Currency: "memeboxTUT", PercentageChange24Hour: -6.5, Last: 0.02047, USDTVolume24Hour: 108404.86, MarketCap: 17010203.55}
		assert.Equal(t, exp, result[0], "GetAlphaCurrencyTicker should decode every field")
	}
}

func TestGetAlphaTokens(t *testing.T) {
	t.Parallel()
	const contractAddress = "FtTSDNLD5mMLn3anqEQpy44cRdrtAJRrLX2MKXxfpump"
	for _, tc := range []struct {
		name           string
		chain          string
		launchPlatform string
		address        string
		page           uint64
		expected       *AlphaCurrencyDetailResponse
	}{
		{
			name:     "without filters",
			expected: &AlphaCurrencyDetailResponse{Currency: "memeboxtrump", Name: "TRUMP", Chain: "SOLANA", Address: "6p6xgHyF7AeE6TZkSmFsko444wqoP15icUSqi2jfGiPN", Status: 1, Precision: 4, AmountPrecision: 4},
		},
		{
			name:           "with filters",
			chain:          "solana",
			launchPlatform: "pump",
			address:        contractAddress,
			page:           1,
			expected:       &AlphaCurrencyDetailResponse{Currency: "memeboxretard", Name: "retard", Chain: "SOLANA", Address: contractAddress, Status: 1, Precision: 8, AmountPrecision: 1},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetAlphaTokens(t.Context(), tc.chain, tc.launchPlatform, tc.address, tc.page)
			require.NoError(t, err, "GetAlphaTokens must not error")
			if mockTests {
				require.Len(t, result, 1, "GetAlphaTokens must return one mocked token")
				assert.Equal(t, tc.expected, result[0], "GetAlphaTokens should decode every field")
				return
			}
			assert.NotNil(t, result, "GetAlphaTokens should return tokens")
		})
	}
}

func TestAlphaCurrencyQuoteInfoRequestMarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		side order.Side
		exp  string
	}{
		{order.Buy, "buy"},
		{order.Sell, "sell"},
	} {
		got, err := (&AlphaCurrencyQuoteInfoRequest{Currency: alphaTestCurrency, Side: tc.side, Amount: 1, GasMode: "speed"}).MarshalJSON()
		require.NoErrorf(t, err, "MarshalJSON must not error for %s", tc.side)
		assert.JSONEqf(t, `{"currency":"memeboxELON","side":"`+tc.exp+`","amount":"1","gas_mode":"speed"}`, string(got), "side should be sent lower-case and the currency unchanged for %s", tc.side)
	}
}
