package bybit

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testutils "github.com/thrasher-corp/gocryptotrader/internal/testing/utils"
)

func TestWSCreateOrder(t *testing.T) {
	t.Parallel()

	arg := &PlaceOrderRequest{}
	_, err := e.WSCreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, errCategoryNotSet)

	arg.Category = cSpot
	_, err = e.WSCreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = currency.NewBTCUSDT()
	_, err = e.WSCreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "Buy"
	_, err = e.WSCreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = "Limit"
	_, err = e.WSCreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.OrderQuantity = 0.0001
	arg.TriggerDirection = 69
	_, err = e.WSCreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidTriggerDirection)

	arg.TriggerDirection = 0
	arg.OrderFilter = "dodgy"
	_, err = e.WSCreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidOrderFilter)

	arg.OrderFilter = "Order"
	arg.TriggerPriceType = "dodgy"
	_, err = e.WSCreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidTriggerPriceType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	e := getWebsocketInstance(t)
	got, err := e.WSCreateOrder(t.Context(), &PlaceOrderRequest{
		Category:      cSpot,
		Symbol:        currency.NewBTCUSDT(),
		Side:          "Buy",
		OrderType:     "Limit",
		Price:         55000,
		OrderQuantity: -0.0001, // Replace with a valid quantity
		TimeInForce:   "FOK",   // Replace with GTC to submit a valid order if outside current trading price range.
	})
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketSubmitOrder(t *testing.T) {
	t.Parallel()

	t.Run("coverage", func(t *testing.T) {
		t.Parallel()

		// Spot market orders require quote amount; this keeps local wrapper validation covered.
		_, err := e.WebsocketSubmitOrder(t.Context(), &order.Submit{
			Exchange:  e.Name,
			Pair:      currency.NewBTCUSDT(),
			AssetType: asset.Spot,
			Side:      order.Buy,
			Type:      order.Market,
			Amount:    0.0001,
		})
		require.ErrorIs(t, err, order.ErrAmountMustBeSet)
	})

	t.Run("live validation passes but order is not placed", func(t *testing.T) {
		t.Parallel()

		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ex := getWebsocketInstance(t)

		_, err := ex.WebsocketSubmitOrder(t.Context(), &order.Submit{
			Exchange:    ex.Name,
			Pair:        currency.NewBTCUSDT(),
			AssetType:   asset.Spot,
			Side:        order.Buy,
			Type:        order.Limit,
			TimeInForce: order.FillOrKill,
			Amount:      0.0001,
			Price:       1,
		})
		require.ErrorIs(t, err, request.ErrAuthRequestFailed)
		require.NotErrorIs(t, err, order.ErrAmountMustBeSet)
	})
}

func TestWSAmendOrder(t *testing.T) {
	t.Parallel()
	arg := &AmendOrderRequest{}
	_, err := e.WSAmendOrder(t.Context(), arg)
	require.ErrorIs(t, err, errCategoryNotSet)

	arg.Category = cSpot
	_, err = e.WSAmendOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = currency.NewBTCUSDT()
	_, err = e.WSAmendOrder(t.Context(), arg)
	require.ErrorIs(t, err, errEitherOrderIDOROrderLinkIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	e := getWebsocketInstance(t)
	arg.OrderID = "1793353687809485568" // Replace with a valid order ID
	arg.OrderQuantity = 0.0002
	got, err := e.WSAmendOrder(t.Context(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketModifyOrder(t *testing.T) {
	t.Parallel()

	t.Run("coverage", func(t *testing.T) {
		t.Parallel()

		_, err := e.WebsocketModifyOrder(t.Context(), &order.Modify{})
		require.ErrorIs(t, err, order.ErrPairIsEmpty)
	})

	t.Run("live validation passes but order is not modified", func(t *testing.T) {
		t.Parallel()

		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ex := getWebsocketInstance(t)

		_, err := ex.WebsocketModifyOrder(t.Context(), &order.Modify{
			Pair:      currency.NewBTCUSDT(),
			AssetType: asset.Spot,
			Amount:    0.0001,
			Price:     1,
			OrderID:   "codex-do-not-fill-this-order-id",
		})
		require.ErrorIs(t, err, request.ErrAuthRequestFailed)
		require.NotErrorIs(t, err, order.ErrPairIsEmpty)
	})
}

func TestWSCancelOrder(t *testing.T) {
	t.Parallel()
	arg := &CancelOrderRequest{}
	_, err := e.WSCancelOrder(t.Context(), arg)
	require.ErrorIs(t, err, errCategoryNotSet)

	arg.Category = cSpot
	_, err = e.WSCancelOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = currency.NewBTCUSDT()
	_, err = e.WSCancelOrder(t.Context(), arg)
	require.ErrorIs(t, err, errEitherOrderIDOROrderLinkIDRequired)

	arg.OrderID = "1793353687809485568" // Replace with a valid order ID

	arg.OrderFilter = "dodgy"
	_, err = e.WSCancelOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidOrderFilter)

	arg.Category = cLinear
	_, err = e.WSCancelOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidCategory)

	arg.Category = cSpot
	arg.OrderFilter = "Order"

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	e := getWebsocketInstance(t)
	got, err := e.WSCancelOrder(t.Context(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketCancelOrder(t *testing.T) {
	t.Parallel()

	t.Run("coverage", func(t *testing.T) {
		t.Parallel()

		err := e.WebsocketCancelOrder(t.Context(), &order.Cancel{})
		require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	})

	t.Run("live validation passes but order is not cancelled", func(t *testing.T) {
		t.Parallel()

		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ex := getWebsocketInstance(t)

		err := ex.WebsocketCancelOrder(t.Context(), &order.Cancel{
			OrderID:   "codex-do-not-fill-this-order-id",
			Pair:      currency.NewBTCUSDT(),
			AssetType: asset.Spot,
		})
		require.ErrorIs(t, err, request.ErrAuthRequestFailed)
		require.NotErrorIs(t, err, order.ErrOrderIDNotSet)
	})
}

// getWebsocketInstance returns a websocket instance copy for live bi-directional testing
func getWebsocketInstance(t *testing.T) *Exchange {
	t.Helper()
	cfg := &config.Config{}
	root, err := testutils.RootPathFromCWD()
	require.NoError(t, err)

	err = cfg.LoadConfig(filepath.Join(root, "testdata", "configtest.json"), true)
	require.NoError(t, err)

	pairs := &e.CurrencyPairs
	e := new(Exchange)
	e.SetDefaults()
	bConf, err := cfg.GetExchangeConfig("Bybit")
	require.NoError(t, err)
	bConf.API.AuthenticatedSupport = true
	bConf.API.AuthenticatedWebsocketSupport = true
	bConf.API.Credentials.Key = apiKey
	bConf.API.Credentials.Secret = apiSecret

	require.NoError(t, e.Setup(bConf), "Setup must not error")
	e.CurrencyPairs.Load(pairs)
	require.NoError(t, e.Websocket.Connect(t.Context()))
	return e
}
