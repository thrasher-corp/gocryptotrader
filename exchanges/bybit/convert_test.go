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

func TestGetConvertCoinList(t *testing.T) {
	t.Parallel()
	_, err := e.GetConvertCoinList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetConvertCoinList must reject a nil request")
	_, err = e.GetConvertCoinList(t.Context(), &ConvertCoinListRequest{})
	require.ErrorIs(t, err, errMissingAccountType, "GetConvertCoinList must reject an empty account type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		arg  *ConvertCoinListRequest
		exp  *ConvertCoinListResponse
	}{
		{
			name: "to coins",
			arg:  &ConvertCoinListRequest{AccountType: "eb_convert_uta", Coin: currency.USDT, Side: 1},
			exp: &ConvertCoinListResponse{
				Coins: []ConvertCoin{
					{
						Coin:               currency.BTC,
						FullName:           "Bitcoin",
						Icon:               "https://t1.bycsi.com/app/assets/token/0717b8c28c2373bf714c964195411d0f.svg",
						IconNight:          "https://t1.bycsi.com/app/assets/token/9504b4c841194cc38f04041003ffbfdb.svg",
						AccuracyLength:     8,
						CoinType:           "crypto",
						Balance:            0.0105,
						USDTBalance:        1250.36,
						SingleFromMinLimit: 0.001,
						SingleFromMaxLimit: 1,
						DisableTo:          true,
						TimePeriod:         1,
						SingleToMinLimit:   0.0001,
						SingleToMaxLimit:   5,
						DailyFromMinLimit:  0.001,
						DailyFromMaxLimit:  10,
						DailyToMinLimit:    0.0001,
						DailyToMaxLimit:    50,
					},
					{
						Coin:               currency.XRP,
						FullName:           "XRP",
						Icon:               "https://s1.bycsi.com/app/assets/token/7fd3a6f3b1d3e2f3f1d9b9b6c1f6e1a6.svg",
						IconNight:          "https://s1.bycsi.com/app/assets/token/2c6b2d8b5a3a3e2f1d9b9b6c1f6e1a6c.svg",
						AccuracyLength:     6,
						CoinType:           "crypto",
						SingleFromMinLimit: 10,
						SingleFromMaxLimit: 500000,
						DisableFrom:        true,
					},
				},
			},
		},
		{
			name: "from coins",
			arg:  &ConvertCoinListRequest{AccountType: "eb_convert_uta"},
			exp: &ConvertCoinListResponse{
				Coins: []ConvertCoin{
					{
						Coin:               currency.SOL,
						FullName:           "SOL",
						Icon:               "https://s1.bycsi.com/app/assets/token/87ca5f1ca7229bdf0d9a16435653007c.svg",
						IconNight:          "https://t1.bycsi.com/app/assets/token/383a834046655ffe5ef1be1a025791cc.svg",
						AccuracyLength:     8,
						CoinType:           "crypto",
						Balance:            18.05988133,
						USDTBalance:        2458.4699021177503,
						SingleFromMinLimit: 0.1,
						SingleFromMaxLimit: 1250,
						TimePeriod:         1,
						SingleToMinLimit:   0.01,
						SingleToMaxLimit:   2500,
						DailyFromMinLimit:  0.1,
						DailyFromMaxLimit:  25000,
						DailyToMinLimit:    0.01,
						DailyToMaxLimit:    50000,
					},
					{
						Coin:               currency.ETH,
						FullName:           "ETH",
						Icon:               "https://s1.bycsi.com/app/assets/token/d6c17c9e767e1810875c702d86ac9f32.svg",
						IconNight:          "https://t1.bycsi.com/app/assets/token/9613ac8e7d62081f4ca20488ae5b168d.svg",
						AccuracyLength:     8,
						CoinType:           "crypto",
						Balance:            0.80264489,
						USDTBalance:        2596.097516500328,
						SingleFromMinLimit: 0.01,
						SingleFromMaxLimit: 250,
						DisableFrom:        true,
						DisableTo:          true,
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetConvertCoinList(t.Context(), tc.arg)
			require.NoError(t, err, "GetConvertCoinList must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetConvertCoinList should decode every field")
				return
			}
			assert.NotEmpty(t, result.Coins, "GetConvertCoinList should return coins")
		})
	}
}

func TestRequestQuote(t *testing.T) {
	t.Parallel()
	_, err := e.RequestQuote(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "RequestQuote must reject a nil request")
	arg := &RequestQuoteRequest{}
	_, err = e.RequestQuote(t.Context(), arg)
	require.ErrorIs(t, err, errMissingAccountType, "RequestQuote must reject an empty account type")
	arg.AccountType = "eb_convert_uta"
	_, err = e.RequestQuote(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "RequestQuote must reject an empty from coin")
	arg.FromCoin = currency.XRP
	_, err = e.RequestQuote(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "RequestQuote must reject an empty to coin")
	arg.ToCoin = currency.USDT
	_, err = e.RequestQuote(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "RequestQuote must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	arg.RequestAmount = 0.0088
	arg.FromCoinType = "crypto"
	arg.ToCoinType = "crypto"
	arg.ParamType = "opFrom"
	arg.ParamValue = "broker-id-001"
	arg.RequestID = "test-00002"
	result, err := e.RequestQuote(t.Context(), arg)
	require.NoError(t, err, "RequestQuote must not error")
	if mockTests {
		exp := &RequestQuoteResponse{
			QuoteTransactionID: "10175108571334212336947200",
			ExchangeRate:       2.84849953775,
			FromCoin:           currency.XRP,
			FromCoinType:       "crypto",
			ToCoin:             currency.USDT,
			ToCoinType:         "crypto",
			FromAmount:         0.0088,
			ToAmount:           0.0250667959322,
			ExpiredTime:        types.Time(time.UnixMilli(1758693506127)),
			RequestID:          "test-00002",
			ExtendedTaxAndFee:  []json.RawMessage{},
		}
		assert.Equal(t, exp, result, "RequestQuote should decode every field")
		return
	}
	assert.NotEmpty(t, result.QuoteTransactionID, "RequestQuote should return a quote")
}

func TestConfirmQuote(t *testing.T) {
	t.Parallel()
	_, err := e.ConfirmQuote(t.Context(), "")
	require.ErrorIs(t, err, errQuoteTransactionIDEmpty, "ConfirmQuote must reject an empty quote transaction ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ConfirmQuote(t.Context(), "10175108571334212336947200")
	require.NoError(t, err, "ConfirmQuote must not error")
	if mockTests {
		exp := &ConfirmQuoteResponse{
			QuoteTransactionID: "10175108571334212336947200",
			ExchangeStatus:     "processing",
		}
		assert.Equal(t, exp, result, "ConfirmQuote should decode every field")
		return
	}
	assert.NotEmpty(t, result.QuoteTransactionID, "ConfirmQuote should return the quote transaction ID")
}

func TestGetConvertStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetConvertStatus(t.Context(), "", "10414247553864074960678912")
	require.ErrorIs(t, err, errMissingAccountType, "GetConvertStatus must reject an empty account type")
	_, err = e.GetConvertStatus(t.Context(), "eb_convert_uta", "")
	require.ErrorIs(t, err, errQuoteTransactionIDEmpty, "GetConvertStatus must reject an empty quote transaction ID")

	accountType, quoteTransactionID := "eb_convert_uta", "10414247553864074960678912"
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		// A quote belongs to its account, so query one from the account's own history
		history, err := e.GetConvertHistory(t.Context(), &ConvertHistoryRequest{AccountTypes: []string{"eb_convert_uta", "eb_convert_funding"}})
		require.NoError(t, err, "GetConvertHistory must not error")
		if len(history.List) == 0 {
			t.Skip("no conversion to query")
		}
		accountType, quoteTransactionID = history.List[0].AccountType, history.List[0].ExchangeTransactionID
	}
	result, err := e.GetConvertStatus(t.Context(), accountType, quoteTransactionID)
	require.NoError(t, err, "GetConvertStatus must not error")
	if mockTests {
		exp := &ConvertStatusResponse{
			Result: ConvertRecord{
				AccountType:           "eb_convert_uta",
				ExchangeTransactionID: "10414247553864074960678912",
				UserID:                "100000001",
				FromCoin:              currency.NewCode("PIRATE"),
				FromCoinType:          "crypto",
				FromAmount:            49.96,
				ToCoin:                currency.USDT,
				ToCoinType:            "crypto",
				ToAmount:              2.3364201371342,
				ExchangeStatus:        "success",
				ExtendedInfo: ConvertExtendedInfo{
					ParamType:  "opFrom",
					ParamValue: "broker-id-001",
				},
				ConvertRate: 0.046765815395,
				CreatedAt:   types.Time(time.UnixMilli(1754528298061)),
			},
		}
		assert.Equal(t, exp, result, "GetConvertStatus should decode every field")
		return
	}
	assert.Equal(t, quoteTransactionID, result.Result.ExchangeTransactionID, "GetConvertStatus should return the requested conversion")
}

func TestGetConvertHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetConvertHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetConvertHistory must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetConvertHistory(t.Context(), &ConvertHistoryRequest{AccountTypes: []string{"eb_convert_uta"}})
	require.NoError(t, err, "GetConvertHistory must not error")
	t.Run("second page of two wallets", func(t *testing.T) {
		t.Parallel()
		// Bybit takes several wallet types comma separated, and numbers pages from 1
		page, err := e.GetConvertHistory(t.Context(), &ConvertHistoryRequest{AccountTypes: []string{"eb_convert_uta", "eb_convert_funding"}, Index: 2, Limit: 1})
		require.NoError(t, err, "GetConvertHistory must not error")
		if mockTests {
			exp := &ConvertHistoryResponse{
				List: []ConvertRecord{
					{
						AccountType:           "eb_convert_funding",
						ExchangeTransactionID: "10412345678901234567890123",
						UserID:                "100000001",
						FromCoin:              currency.USDT,
						FromCoinType:          "crypto",
						FromAmount:            100,
						ToCoin:                currency.BTC,
						ToCoinType:            "crypto",
						ToAmount:              0.0009433,
						ExchangeStatus:        "success",
						ConvertRate:           0.000009433,
						CreatedAt:             types.Time(time.UnixMilli(1754300000000)),
					},
				},
			}
			assert.Equal(t, exp, page, "GetConvertHistory should send the wallet types and page and decode the page")
			return
		}
		assert.NotNil(t, page, "GetConvertHistory should return a page")
	})
	if mockTests {
		exp := &ConvertHistoryResponse{
			List: []ConvertRecord{
				{
					AccountType:           "eb_convert_uta",
					ExchangeTransactionID: "104231555340214158196736",
					UserID:                "100000001",
					FromCoin:              currency.NewCode("UXLINK"),
					FromCoinType:          "crypto",
					FromAmount:            7.9952,
					ToCoin:                currency.USDT,
					ToCoinType:            "crypto",
					ToAmount:              2.84509740190888,
					ExchangeStatus:        "success",
					ConvertRate:           0.35585068565,
					CreatedAt:             types.Time(time.UnixMilli(1754880224953)),
				},
				{
					AccountType:           "eb_convert_uta",
					ExchangeTransactionID: "10414247553864074960678912",
					UserID:                "100000001",
					FromCoin:              currency.NewCode("PIRATE"),
					FromCoinType:          "crypto",
					FromAmount:            49.96,
					ToCoin:                currency.USDT,
					ToCoinType:            "crypto",
					ToAmount:              2.3364201371342,
					ExchangeStatus:        "success",
					ExtendedInfo: ConvertExtendedInfo{
						ParamType:  "opFrom",
						ParamValue: "broker-id-001",
					},
					ConvertRate: 0.046765815395,
					CreatedAt:   types.Time(time.UnixMilli(1754528298061)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetConvertHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetConvertHistory should return a page")
}
