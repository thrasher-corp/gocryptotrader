package bybit

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestValidatePlaceOrderRequest(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		params PlaceOrderRequest
		err    error
	}{
		{err: errCategoryNotSet},
		{params: PlaceOrderRequest{Category: cSpot}, err: currency.ErrCurrencyPairEmpty},
		{
			params: PlaceOrderRequest{Category: cSpot, Symbol: currency.NewBTCUSDT(), EnableBorrow: true},
			err:    order.ErrSideIsInvalid,
		},
		{
			params: PlaceOrderRequest{
				Category:     cSpot,
				Symbol:       currency.NewBTCUSDT(),
				EnableBorrow: true,
				Side:         sideBuy,
			},
			err: order.ErrTypeIsInvalid,
		},
		{
			params: PlaceOrderRequest{
				Category:     cSpot,
				Symbol:       currency.NewBTCUSDT(),
				EnableBorrow: true,
				Side:         sideBuy,
				OrderType:    orderTypeToString(order.Limit),
			},
			err: limits.ErrAmountBelowMin,
		},
		{
			params: PlaceOrderRequest{
				Category:         cSpot,
				Symbol:           currency.NewBTCUSDT(),
				EnableBorrow:     true,
				Side:             sideBuy,
				OrderType:        orderTypeToString(order.Limit),
				OrderQuantity:    0.0001,
				TriggerDirection: 69,
			},
			err: errInvalidTriggerDirection,
		},
		{
			params: PlaceOrderRequest{
				Category:      cInverse,
				Symbol:        currency.NewBTCUSDT(),
				EnableBorrow:  true,
				Side:          sideBuy,
				OrderType:     orderTypeToString(order.Limit),
				OrderQuantity: 0.0001,
				OrderFilter:   "dodgy",
			},
			err: errInvalidCategory,
		},
		{
			params: PlaceOrderRequest{
				Category:      cSpot,
				Symbol:        currency.NewBTCUSDT(),
				EnableBorrow:  true,
				Side:          sideBuy,
				OrderType:     orderTypeToString(order.Limit),
				OrderQuantity: 0.0001,
				OrderFilter:   "dodgy",
			},
			err: errInvalidOrderFilter,
		},
		{
			params: PlaceOrderRequest{
				Category:         cSpot,
				Symbol:           currency.NewBTCUSDT(),
				EnableBorrow:     true,
				Side:             sideBuy,
				OrderType:        orderTypeToString(order.Limit),
				OrderQuantity:    0.0001,
				TriggerPriceType: "dodgy",
			},
			err: errInvalidTriggerPriceType,
		},
		{
			params: PlaceOrderRequest{
				Category:      cSpot,
				Symbol:        currency.NewBTCUSDT(),
				EnableBorrow:  true,
				Side:          sideBuy,
				OrderType:     orderTypeToString(order.Limit),
				OrderQuantity: 0.0001,
			},
			err: nil,
		},
	} {
		if tc.err != nil {
			require.ErrorIs(t, tc.params.Validate(), tc.err)
			continue
		}
		require.NoError(t, tc.params.Validate())
	}
}

func TestValidateAmendOrderRequest(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		params AmendOrderRequest
		err    error
	}{
		{err: errCategoryNotSet},
		{params: AmendOrderRequest{Category: cSpot}, err: currency.ErrCurrencyPairEmpty},
		{
			params: AmendOrderRequest{
				Category: cSpot,
				Symbol:   currency.NewBTCUSDT(),
			},
			err: errEitherOrderIDOROrderLinkIDRequired,
		},
		{
			params: AmendOrderRequest{
				Category: cSpot,
				Symbol:   currency.NewBTCUSDT(),
				OrderID:  "69420",
				TPSLMode: "TP",
			},
			err: nil,
		},
	} {
		if tc.err != nil {
			require.ErrorIs(t, tc.params.Validate(), tc.err)
			continue
		}
		require.NoError(t, tc.params.Validate())
	}
}

func TestValidateCancelOrderRequest(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		params CancelOrderRequest
		err    error
	}{
		{err: errCategoryNotSet},
		{params: CancelOrderRequest{Category: cSpot}, err: currency.ErrCurrencyPairEmpty},
		{
			params: CancelOrderRequest{
				Category: cSpot,
				Symbol:   currency.NewBTCUSDT(),
			},
			err: errEitherOrderIDOROrderLinkIDRequired,
		},
		{
			params: CancelOrderRequest{
				Category:    cLinear,
				Symbol:      currency.NewBTCUSDT(),
				OrderID:     "69420",
				OrderFilter: "dodgy",
			},
			err: errInvalidCategory,
		},
		{
			params: CancelOrderRequest{
				Category:    cSpot,
				Symbol:      currency.NewBTCUSDT(),
				OrderID:     "69420",
				OrderFilter: "dodgy",
			},
			err: errInvalidOrderFilter,
		},
		{
			params: CancelOrderRequest{
				Category: cSpot,
				Symbol:   currency.NewBTCUSDT(),
				OrderID:  "69420",
			},
			err: nil,
		},
	} {
		if tc.err != nil {
			require.ErrorIs(t, tc.params.Validate(), tc.err)
			continue
		}
		require.NoError(t, tc.params.Validate())
	}
}
