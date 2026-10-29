package gate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetSupportedFlashSwapCurrencyPairs(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		ccy         currency.Code
		limit, page uint64
		exp         *FlashSwapCurrencyPairResponse
	}{
		{
			ccy:   currency.BTC,
			limit: 10,
			exp: &FlashSwapCurrencyPairResponse{
				CurrencyPair:  getPair(t, asset.Spot),
				SellCurrency:  currency.BTC,
				BuyCurrency:   currency.USDT,
				SellMinAmount: 0.000001,
				SellMaxAmount: 250,
				BuyMinAmount:  1,
				BuyMaxAmount:  20000000,
			},
		},
		{
			limit: 10,
			page:  1,
			exp: &FlashSwapCurrencyPairResponse{
				CurrencyPair:  currency.NewPairWithDelimiter("PIRATE", "USDT", currency.UnderscoreDelimiter),
				SellCurrency:  currency.NewCode("PIRATE"),
				BuyCurrency:   currency.USDT,
				SellMinAmount: 1,
				SellMaxAmount: 842600,
				BuyMinAmount:  1,
				BuyMaxAmount:  1710,
			},
		},
	} {
		result, err := e.GetSupportedFlashSwapCurrencyPairs(t.Context(), tc.ccy, tc.limit, tc.page)
		require.NoErrorf(t, err, "GetSupportedFlashSwapCurrencyPairs must not error for currency %q", tc.ccy)
		if mockTests {
			require.Lenf(t, result, 5, "GetSupportedFlashSwapCurrencyPairs must return every recorded pair for currency %q", tc.ccy)
			assert.Equalf(t, tc.exp, result[0], "GetSupportedFlashSwapCurrencyPairs should decode every field for currency %q", tc.ccy)
			continue
		}
		assert.NotEmptyf(t, result, "GetSupportedFlashSwapCurrencyPairs should return pairs for currency %q", tc.ccy)
	}
}

func TestCreateFlashSwapOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CreateFlashSwapOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateFlashSwapOrder must reject a nil request")

	arg := &FlashSwapOrderRequest{}
	_, err = e.CreateFlashSwapOrder(t.Context(), arg)
	require.ErrorIs(t, err, errMissingPreviewID, "CreateFlashSwapOrder must reject an empty preview ID")

	arg.PreviewID = "1234"
	_, err = e.CreateFlashSwapOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateFlashSwapOrder must reject an empty buy currency")

	arg.BuyCurrency = currency.BTC
	_, err = e.CreateFlashSwapOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateFlashSwapOrder must reject an empty sell currency")

	arg.SellCurrency = currency.USDT
	_, err = e.CreateFlashSwapOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "CreateFlashSwapOrder must reject an empty sell amount")

	arg.SellAmount = 34234
	_, err = e.CreateFlashSwapOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "CreateFlashSwapOrder must reject an empty buy amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateFlashSwapOrder(t.Context(), &FlashSwapOrderRequest{
		PreviewID:    "1234",
		SellCurrency: currency.USDT,
		BuyCurrency:  currency.BTC,
		BuyAmount:    34234,
		SellAmount:   34234,
	})
	require.NoError(t, err, "CreateFlashSwapOrder must not error")
	if mockTests {
		exp := &FlashSwapOrderResponse{
			ID:           54646,
			CreateTime:   types.Time(time.UnixMilli(1744150000378)),
			UpdateTime:   types.Time(time.UnixMilli(1744150000378)),
			UserID:       11135567,
			SellCurrency: currency.USDT,
			SellAmount:   34234,
			BuyCurrency:  currency.BTC,
			BuyAmount:    34234,
			Price:        1,
			Status:       1,
		}
		assert.Equal(t, exp, result, "CreateFlashSwapOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CreateFlashSwapOrder should return a result")
}

func TestGetAllFlashSwapOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAllFlashSwapOrders(t.Context(), 1, currency.EMPTYCODE, currency.EMPTYCODE, true, 0, 0)
	require.NoError(t, err, "GetAllFlashSwapOrders must not error")
	if mockTests {
		exp := []*FlashSwapOrderResponse{{
			ID:           54646,
			CreateTime:   types.Time(time.UnixMilli(1744150000378)),
			UpdateTime:   types.Time(time.UnixMilli(1744150000378)),
			UserID:       11135567,
			SellCurrency: currency.BTC,
			SellAmount:   0.01,
			BuyCurrency:  currency.USDT,
			BuyAmount:    650,
			Price:        65000,
			Status:       1,
		}}
		assert.Equal(t, exp, result, "GetAllFlashSwapOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAllFlashSwapOrders should return a result")
}

func TestGetSingleFlashSwapOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleFlashSwapOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetSingleFlashSwapOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSingleFlashSwapOrder(t.Context(), "1234")
	require.NoError(t, err, "GetSingleFlashSwapOrder must not error")
	if mockTests {
		exp := &FlashSwapOrderResponse{
			ID:           1234,
			CreateTime:   types.Time(time.UnixMilli(1744150000378)),
			UpdateTime:   types.Time(time.UnixMilli(1744150000378)),
			UserID:       11135567,
			SellCurrency: currency.BTC,
			SellAmount:   0.01,
			BuyCurrency:  currency.USDT,
			BuyAmount:    650,
			Price:        65000,
			Status:       2,
		}
		assert.Equal(t, exp, result, "GetSingleFlashSwapOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSingleFlashSwapOrder should return a result")
}

func TestInitiateFlashSwapOrderReview(t *testing.T) {
	t.Parallel()
	_, err := e.InitiateFlashSwapOrderReview(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "InitiateFlashSwapOrderReview must reject a nil request")

	arg := &FlashSwapPreviewRequest{SellAmount: 100}
	_, err = e.InitiateFlashSwapOrderReview(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "InitiateFlashSwapOrderReview must reject an empty sell currency")

	arg.SellCurrency = currency.USDT
	_, err = e.InitiateFlashSwapOrderReview(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "InitiateFlashSwapOrderReview must reject an empty buy currency")

	arg.BuyCurrency = currency.BTC
	arg.SellAmount = 0
	_, err = e.InitiateFlashSwapOrderReview(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "InitiateFlashSwapOrderReview must reject a request without sell or buy amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.InitiateFlashSwapOrderReview(t.Context(), &FlashSwapPreviewRequest{
		SellCurrency: currency.USDT,
		BuyCurrency:  currency.BTC,
		SellAmount:   100,
	})
	require.NoError(t, err, "InitiateFlashSwapOrderReview must not error")
	if mockTests {
		exp := &InitFlashSwapOrderPreviewResponse{
			PreviewID:    "3453434",
			SellCurrency: currency.USDT,
			SellAmount:   100,
			BuyCurrency:  currency.BTC,
			BuyAmount:    0.0015,
			Price:        66666.66,
		}
		assert.Equal(t, exp, result, "InitiateFlashSwapOrderReview should decode every field")
		return
	}
	assert.NotNil(t, result, "InitiateFlashSwapOrderReview should return a result")
}
