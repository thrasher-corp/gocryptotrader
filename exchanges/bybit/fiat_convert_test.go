package bybit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetFiatTradingPairList(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFiatTradingPairList(t.Context(), "0")
	require.NoError(t, err, "GetFiatTradingPairList must not error")
	if mockTests {
		exp := &FiatTradingPairListResponse{
			Fiats: []FiatConvertCoin{
				{
					Coin:               currency.NewCode("GEL"),
					FullName:           "Georgian Lari",
					Icon:               "https://s1.bycsi.com/common-static/wove/fiat-admin/2023-5-4/GEL.svg",
					IconNight:          "https://s1.bycsi.com/common-static/wove/fiat-admin/2023-5-4/GEL.svg",
					Precision:          2,
					SingleFromMinLimit: 10,
					SingleFromMaxLimit: 100000,
				},
				{
					Coin:               currency.NewCode("KZT"),
					FullName:           "Kazakhstani Tenge",
					Icon:               "https://s1.bycsi.com/common-static/wove/fiat-admin/2023-5-4/KZT.svg",
					IconNight:          "https://s1.bycsi.com/common-static/wove/fiat-admin/2023-5-4/KZT.svg",
					Precision:          2,
					Disable:            true,
					SingleFromMinLimit: 2000,
					SingleFromMaxLimit: 20000000,
				},
			},
			Cryptos: []FiatConvertCoin{
				{
					Coin:               currency.USDT,
					FullName:           "Tether USDT",
					Icon:               "https://s1.bycsi.com/common-static/wove/fiat-admin/2024-8-5/8e50959d5f3e45bebf522e0cad456439_1726814031848.svg",
					IconNight:          "https://s1.bycsi.com/common-static/wove/fiat-admin/2024-8-5/8e50959d5f3e45bebf522e0cad456439_1726814031848.svg",
					Precision:          4,
					SingleFromMinLimit: 10,
					SingleFromMaxLimit: 10000,
				},
				{
					Coin:               currency.BTC,
					FullName:           "Bitcoin",
					Icon:               "https://s1.bycsi.com/common-static/wove/fiat-admin/20d09e76a0ab401f80bd545ae874c6a3_48x48.svg",
					IconNight:          "https://s1.bycsi.com/common-static/wove/fiat-admin/20d09e76a0ab401f80bd545ae874c6a3_48x48.svg",
					Precision:          8,
					Disable:            true,
					SingleFromMinLimit: 0.0001,
					SingleFromMaxLimit: 1,
				},
			},
		}
		assert.Equal(t, exp, result, "GetFiatTradingPairList should decode every field")
		return
	}
	assert.NotEmpty(t, result.Fiats, "GetFiatTradingPairList should return fiat coins")
}

func TestGetFiatReferencePrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetFiatReferencePrice(t.Context(), "")
	require.ErrorIs(t, err, errSymbolMissing, "GetFiatReferencePrice must reject an empty symbol")

	result, err := e.GetFiatReferencePrice(t.Context(), "EUR-USDT")
	require.NoError(t, err, "GetFiatReferencePrice must not error")
	if mockTests {
		exp := &FiatReferencePriceResponse{
			Symbol:    "EUR-USDT",
			Fiat:      currency.EUR,
			Crypto:    currency.USDT,
			Timestamp: types.Time(time.Unix(1791534382, 0)),
			Buys: []FiatReferencePriceItem{
				{
					UnitPrice:     0.8941268,
					PaymentMethod: "Credit Card",
				},
				{
					UnitPrice:     0.8950237,
					PaymentMethod: "My Bank SWIFT",
				},
			},
			Sells: []FiatReferencePriceItem{
				{
					UnitPrice:     0.8870396,
					PaymentMethod: "Card",
				},
				{
					UnitPrice:     0.8860914,
					PaymentMethod: "Cash Balance",
				},
			},
		}
		assert.Equal(t, exp, result, "GetFiatReferencePrice should decode every field")
		return
	}
	assert.Equal(t, "EUR-USDT", result.Symbol, "GetFiatReferencePrice should return the requested symbol")
	assert.True(t, result.Fiat.Equal(currency.EUR), "GetFiatReferencePrice should return the pair's fiat")
	assert.NotEmpty(t, result.Buys, "GetFiatReferencePrice should return buy prices")
}

func TestRequestFiatQuote(t *testing.T) {
	t.Parallel()
	_, err := e.RequestFiatQuote(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "RequestFiatQuote must reject a nil request")
	arg := &FiatQuoteRequest{}
	_, err = e.RequestFiatQuote(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "RequestFiatQuote must reject an empty from coin")
	arg.FromCoin = currency.NewCode("GEL")
	_, err = e.RequestFiatQuote(t.Context(), arg)
	require.ErrorIs(t, err, errCoinTypeRequired, "RequestFiatQuote must reject an empty from coin type")
	arg.FromCoinType = "fiat"
	_, err = e.RequestFiatQuote(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "RequestFiatQuote must reject an empty to coin")
	arg.ToCoin = currency.USDT
	_, err = e.RequestFiatQuote(t.Context(), arg)
	require.ErrorIs(t, err, errCoinTypeRequired, "RequestFiatQuote must reject an empty to coin type")
	arg.ToCoinType = "crypto"
	_, err = e.RequestFiatQuote(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "RequestFiatQuote must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	arg.RequestAmount = 100
	arg.RequestCoinType = "fiat"
	result, err := e.RequestFiatQuote(t.Context(), arg)
	require.NoError(t, err, "RequestFiatQuote must not error")
	if mockTests {
		exp := &FiatQuoteResponse{
			QuoteTransactionID: "QuoteTaxId123456",
			ExchangeRate:       0.3685,
			FromCoin:           currency.NewCode("GEL"),
			FromCoinType:       "fiat",
			ToCoin:             currency.USDT,
			ToCoinType:         "crypto",
			FromAmount:         100,
			ToAmount:           36.85,
			ExpiredTime:        types.Time(time.UnixMilli(1764561045346)),
		}
		assert.Equal(t, exp, result, "RequestFiatQuote should decode every field")
		return
	}
	assert.NotEmpty(t, result.QuoteTransactionID, "RequestFiatQuote should return a quote")
}

func TestConfirmFiatQuote(t *testing.T) {
	t.Parallel()
	_, err := e.ConfirmFiatQuote(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ConfirmFiatQuote must reject a nil request")
	arg := &FiatConfirmQuoteRequest{}
	_, err = e.ConfirmFiatQuote(t.Context(), arg)
	require.ErrorIs(t, err, errQuoteTransactionIDEmpty, "ConfirmFiatQuote must reject an empty quote transaction ID")
	arg.QuoteTransactionID = "QuoteTaxId123456"
	_, err = e.ConfirmFiatQuote(t.Context(), arg)
	require.ErrorIs(t, err, errSubUserIDRequired, "ConfirmFiatQuote must reject an empty sub user ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.SubUserID = "123456"
	arg.WebhookURL = "https://example.com/bybit/notify"
	arg.MerchantRequestID = "test-merchant-0001"
	result, err := e.ConfirmFiatQuote(t.Context(), arg)
	require.NoError(t, err, "ConfirmFiatQuote must not error")
	if mockTests {
		exp := &FiatConfirmQuoteResponse{
			TradeNumber:       "TradeNo123456",
			MerchantRequestID: "test-merchant-0001",
		}
		assert.Equal(t, exp, result, "ConfirmFiatQuote should decode every field")
		return
	}
	assert.NotEmpty(t, result.TradeNumber, "ConfirmFiatQuote should return the trade number")
}

func TestGetFiatConvertStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetFiatConvertStatus(t.Context(), "", "")
	require.ErrorIs(t, err, errTradeNumberOrMerchantRequestIDRequired, "GetFiatConvertStatus must reject an empty trade number and merchant request ID")

	if mockTests {
		exp := &FiatConvertStatusResponse{
			TradeNumber:        "TradeNo123456",
			Status:             "success",
			QuoteTransactionID: "QuoteTaxId123456",
			ExchangeRate:       0.3685,
			FromCoin:           currency.NewCode("GEL"),
			FromCoinType:       "fiat",
			ToCoin:             currency.USDT,
			ToCoinType:         "crypto",
			FromAmount:         100,
			ToAmount:           36.85,
			CreatedAt:          types.Time(time.UnixMilli(1764558832014)),
			SubUserID:          "123456",
		}
		for _, tc := range []struct {
			name              string
			tradeNumber       string
			merchantRequestID string
		}{
			{name: "trade number", tradeNumber: "TradeNo123456"},
			{name: "merchant request ID", merchantRequestID: "test-merchant-0001"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				result, err := e.GetFiatConvertStatus(t.Context(), tc.tradeNumber, tc.merchantRequestID)
				require.NoError(t, err, "GetFiatConvertStatus must not error")
				assert.Equal(t, exp, result, "GetFiatConvertStatus should decode every field")
			})
		}
		return
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	trades, err := e.GetFiatConvertHistory(t.Context(), &FiatConvertHistoryRequest{})
	require.NoError(t, err, "GetFiatConvertHistory must not error")
	if len(trades) == 0 {
		t.Skip("no fiat conversion to query")
	}
	result, err := e.GetFiatConvertStatus(t.Context(), trades[0].TradeNumber, "")
	require.NoError(t, err, "GetFiatConvertStatus must not error")
	assert.Equal(t, trades[0].TradeNumber, result.TradeNumber, "GetFiatConvertStatus should return the requested trade")
}

func TestGetFiatConvertHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFiatConvertHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFiatConvertHistory must reject a nil request")
	_, err = e.GetFiatConvertHistory(t.Context(), &FiatConvertHistoryRequest{StartTime: time.UnixMilli(1764600000000), EndTime: time.UnixMilli(1764500000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFiatConvertHistory must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFiatConvertHistory(t.Context(), &FiatConvertHistoryRequest{Index: 1, Limit: 20, StartTime: time.UnixMilli(1764500000000), EndTime: time.UnixMilli(1764600000000)})
	require.NoError(t, err, "GetFiatConvertHistory must not error")
	if mockTests {
		exp := []FiatTrade{
			{
				TradeNumber:        "TradeNo123456",
				Status:             "success",
				QuoteTransactionID: "QuoteTaxId123456",
				ExchangeRate:       0.3685,
				FromCoin:           currency.NewCode("GEL"),
				FromCoinType:       "fiat",
				ToCoin:             currency.USDT,
				ToCoinType:         "crypto",
				FromAmount:         100,
				ToAmount:           36.85,
				CreatedAt:          types.Time(time.UnixMilli(1764560093588)),
				SubUserID:          "123456",
			},
			{
				TradeNumber:        "TradeNo123457",
				Status:             "processing",
				QuoteTransactionID: "QuoteTaxId123457",
				ExchangeRate:       2.7,
				FromCoin:           currency.USDT,
				FromCoinType:       "crypto",
				ToCoin:             currency.NewCode("GEL"),
				ToCoinType:         "fiat",
				FromAmount:         20,
				ToAmount:           54,
				CreatedAt:          types.Time(time.Unix(1764560200, 0)),
				SubUserID:          "123456",
			},
		}
		assert.Equal(t, exp, result, "GetFiatConvertHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFiatConvertHistory should return the trades")
}

func TestGetFiatBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name         string
		fiatCurrency currency.Code
		exp          FiatBalances
	}{
		{
			name: "all currencies",
			exp: FiatBalances{
				{TotalBalance: 100000, Balance: 99900, FrozenBalance: 100, Currency: currency.NewCode("GEL")},
				{TotalBalance: 5000000, Balance: 5000000, Currency: currency.NewCode("KZT")},
			},
		},
		{
			name:         "one currency",
			fiatCurrency: currency.NewCode("GEL"),
			exp:          FiatBalances{{TotalBalance: 100000, Balance: 99900, FrozenBalance: 100, Currency: currency.NewCode("GEL")}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetFiatBalance(t.Context(), tc.fiatCurrency)
			require.NoError(t, err, "GetFiatBalance must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetFiatBalance should decode every field")
				return
			}
			assert.NotNil(t, result, "GetFiatBalance should return the balances")
		})
	}
}

func TestFiatBalancesUnmarshalJSON(t *testing.T) {
	t.Parallel()
	exp := FiatBalances{{TotalBalance: 100000, Balance: 99900, FrozenBalance: 100, Currency: currency.NewCode("GEL")}}
	for _, tc := range []struct {
		name string
		data string
	}{
		{"object", `{"currency":"GEL","totalBalance":"100000","balance":"99900","frozenBalance":"100"}`},
		{"array", `[{"currency":"GEL","totalBalance":"100000","balance":"99900","frozenBalance":"100"}]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var got FiatBalances
			require.NoError(t, got.UnmarshalJSON([]byte(tc.data)), "UnmarshalJSON must not error")
			assert.Equal(t, exp, got, "UnmarshalJSON should decode the balances")
		})
	}
	var got FiatBalances
	assert.Error(t, got.UnmarshalJSON([]byte(`{"balance":true}`)), "UnmarshalJSON should reject an invalid balance object")
	assert.Error(t, got.UnmarshalJSON([]byte(`"GEL"`)), "UnmarshalJSON should reject a string")
}

func TestFiatQuoteResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	exp := FiatQuoteResponse{QuoteTransactionID: "QuoteTaxId123456", ExpiredTime: types.Time(time.UnixMilli(1764561045346))}
	for _, tc := range []struct {
		name string
		data string
	}{
		{"documented keys", `{"quoteTxId":"QuoteTaxId123456","expiredTime":"1764561045346"}`},
		{"sampled keys", `{"quoteTaxId":"QuoteTaxId123456","expireTime":"1764561045346"}`},
		{"mixed keys", `{"quoteTxId":"QuoteTaxId123456","expireTime":"1764561045346"}`},
		{"documented keys before sampled keys", `{"quoteTxId":"QuoteTaxId123456","expiredTime":"1764561045346","quoteTaxId":"QuoteTaxId654321","expireTime":"1764561099999"}`},
		{"documented keys after sampled keys", `{"quoteTaxId":"QuoteTaxId654321","expireTime":"1764561099999","quoteTxId":"QuoteTaxId123456","expiredTime":"1764561045346"}`},
		{"empty documented keys", `{"quoteTxId":"","expiredTime":"","quoteTaxId":"QuoteTaxId123456","expireTime":"1764561045346"}`},
		{"null documented keys", `{"quoteTxId":null,"expiredTime":null,"quoteTaxId":"QuoteTaxId123456","expireTime":"1764561045346"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var got FiatQuoteResponse
			require.NoError(t, got.UnmarshalJSON([]byte(tc.data)), "UnmarshalJSON must not error")
			assert.Equal(t, exp, got, "UnmarshalJSON should decode the quote ID and expiry")
		})
	}
	var got FiatQuoteResponse
	require.NoError(t, got.UnmarshalJSON([]byte(`{"exchangeRate":"0.3685"}`)), "UnmarshalJSON must not error without either key")
	assert.Equal(t, FiatQuoteResponse{ExchangeRate: 0.3685}, got, "UnmarshalJSON should leave the quote ID and expiry empty without either key")
	// A quote decoded into a used one takes the new quote's values, whichever keys they arrive under
	reused := FiatQuoteResponse{QuoteTransactionID: "old", ExpiredTime: types.Time(time.UnixMilli(1764561000000))}
	require.NoError(t, reused.UnmarshalJSON([]byte(`{"quoteTaxId":"QuoteTaxId123456","expireTime":"1764561045346"}`)), "UnmarshalJSON must not error for a used quote")
	assert.Equal(t, exp, reused, "UnmarshalJSON should replace a used quote's ID and expiry with the sampled keys' values")
	assert.Error(t, got.UnmarshalJSON([]byte(`{"quoteTaxId":5}`)), "UnmarshalJSON should reject a non-string sampled quote ID")
}

func TestFiatConvertStatusResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var got FiatConvertStatusResponse
	require.NoError(t, got.UnmarshalJSON([]byte(`{"tradeNo":"TradeNo123456","quoteTaxId":"QuoteTaxId123456"}`)), "UnmarshalJSON must not error")
	assert.Equal(t, FiatConvertStatusResponse{TradeNumber: "TradeNo123456", QuoteTransactionID: "QuoteTaxId123456"}, got, "UnmarshalJSON should read the sampled quote ID as FiatTrade does")
	assert.Error(t, got.UnmarshalJSON([]byte(`{"quoteTaxId":5}`)), "UnmarshalJSON should reject a non-string sampled quote ID")
}

func TestFiatTradeUnmarshalJSON(t *testing.T) {
	t.Parallel()
	exp := FiatTrade{TradeNumber: "TradeNo123456", QuoteTransactionID: "QuoteTaxId123456"}
	for _, tc := range []struct {
		name string
		data string
	}{
		{"documented key", `{"tradeNo":"TradeNo123456","quoteTxId":"QuoteTaxId123456"}`},
		{"sampled key", `{"tradeNo":"TradeNo123456","quoteTaxId":"QuoteTaxId123456"}`},
		{"documented key before sampled key", `{"tradeNo":"TradeNo123456","quoteTxId":"QuoteTaxId123456","quoteTaxId":"QuoteTaxId654321"}`},
		{"documented key after sampled key", `{"tradeNo":"TradeNo123456","quoteTaxId":"QuoteTaxId654321","quoteTxId":"QuoteTaxId123456"}`},
		{"empty documented key", `{"tradeNo":"TradeNo123456","quoteTxId":"","quoteTaxId":"QuoteTaxId123456"}`},
		{"null documented key", `{"tradeNo":"TradeNo123456","quoteTxId":null,"quoteTaxId":"QuoteTaxId123456"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var got FiatTrade
			require.NoError(t, got.UnmarshalJSON([]byte(tc.data)), "UnmarshalJSON must not error")
			assert.Equal(t, exp, got, "UnmarshalJSON should decode the quote ID")
		})
	}
	var got FiatTrade
	require.NoError(t, got.UnmarshalJSON([]byte(`{"tradeNo":"TradeNo123456"}`)), "UnmarshalJSON must not error without either key")
	assert.Equal(t, FiatTrade{TradeNumber: "TradeNo123456"}, got, "UnmarshalJSON should leave the quote ID empty without either key")
	// A conversion decoded into a used one takes the new conversion's quote ID, whichever key it arrives under
	reused := FiatTrade{QuoteTransactionID: "old"}
	require.NoError(t, reused.UnmarshalJSON([]byte(`{"tradeNo":"TradeNo123456","quoteTaxId":"QuoteTaxId123456"}`)), "UnmarshalJSON must not error for a used conversion")
	assert.Equal(t, exp, reused, "UnmarshalJSON should replace a used conversion's quote ID with the sampled key's value")
	assert.Error(t, got.UnmarshalJSON([]byte(`{"quoteTaxId":5}`)), "UnmarshalJSON should reject a non-string sampled quote ID")
}

func TestFiatTradeNotificationUnmarshal(t *testing.T) {
	t.Parallel()
	var got FiatTradeNotification
	require.NoError(t, json.Unmarshal([]byte(`{"tradeNo":"TradeNo123456","status":"success","quoteTxId":"QuoteTaxId123456","exchangeRate":"0.3685","fromCoin":"GEL","fromCoinType":"fiat","toCoin":"USDT","toCoinType":"crypto","fromAmount":"100","toAmount":"36.85","createdAt":"1764558832014"}`), &got), "Unmarshal must not error")
	exp := &FiatTradeNotification{
		TradeNumber:        "TradeNo123456",
		Status:             "success",
		QuoteTransactionID: "QuoteTaxId123456",
		ExchangeRate:       0.3685,
		FromCoin:           currency.NewCode("GEL"),
		FromCoinType:       "fiat",
		ToCoin:             currency.USDT,
		ToCoinType:         "crypto",
		FromAmount:         100,
		ToAmount:           36.85,
		CreatedAt:          types.Time(time.UnixMilli(1764558832014)),
	}
	assert.Equal(t, exp, &got, "FiatTradeNotificationUnmarshal should decode every field")
}
