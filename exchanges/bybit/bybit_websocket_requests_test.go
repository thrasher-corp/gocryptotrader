package bybit

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testutils "github.com/thrasher-corp/gocryptotrader/internal/testing/utils"
)

func TestWSCreateOrder(t *testing.T) {
	t.Parallel()

	arg := &PlaceOrderParams{}
	_, err := b.WSCreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, errCategoryNotSet)

	arg.Category = cSpot
	_, err = b.WSCreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = currency.NewBTCUSDT()
	arg.IsLeverage = 69
	_, err = b.WSCreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidLeverageValue)

	arg.IsLeverage = 0
	_, err = b.WSCreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "Buy"
	_, err = b.WSCreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = "Limit"
	_, err = b.WSCreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.OrderQuantity = 0.0001
	arg.TriggerDirection = 69
	_, err = b.WSCreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidTriggerDirection)

	arg.TriggerDirection = 0
	arg.OrderFilter = "dodgy"
	_, err = b.WSCreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidOrderFilter)

	arg.OrderFilter = "Order"
	arg.TriggerPriceType = "dodgy"
	_, err = b.WSCreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidTriggerPriceType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	b = getWebsocketInstance(t, b)
	got, err := b.WSCreateOrder(t.Context(), &PlaceOrderParams{
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

	// Test quote amount needs to be used due to protocol trade requirements
	s := &order.Submit{
		Exchange:  b.Name,
		Pair:      currency.NewBTCUSDT(),
		AssetType: asset.Spot,
		Side:      order.Buy,
		Type:      order.Market,
		Amount:    0.0001,
	}

	_, err := b.WebsocketSubmitOrder(t.Context(), s)
	require.ErrorIs(t, err, order.ErrAmountMustBeSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	b = getWebsocketInstance(t, b)

	s.Type = order.Limit
	s.Price = 55000
	s.Amount = -0.0001 // Replace with a valid quantity
	got, err := b.WebsocketSubmitOrder(t.Context(), s)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWSAmendOrder(t *testing.T) {
	t.Parallel()
	arg := &AmendOrderParams{}
	_, err := b.WSAmendOrder(t.Context(), arg)
	require.ErrorIs(t, err, errCategoryNotSet)

	arg.Category = cSpot
	_, err = b.WSAmendOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = currency.NewBTCUSDT()
	_, err = b.WSAmendOrder(t.Context(), arg)
	require.ErrorIs(t, err, errEitherOrderIDOROrderLinkIDRequired)

	arg.OrderID = "1793353687809485568" // Replace with a valid order ID
	_, err = b.WSAmendOrder(t.Context(), arg)
	require.ErrorIs(t, err, errAmendArgumentsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	b = getWebsocketInstance(t, b)
	arg.OrderQuantity = 0.0002
	got, err := b.WSAmendOrder(t.Context(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketModifyOrder(t *testing.T) {
	t.Parallel()
	mod := &order.Modify{
		Pair:      currency.NewBTCUSDT(),
		AssetType: asset.Spot,
		Amount:    0.0001,
		OrderID:   "1793388409122024192", // Replace with a valid order ID
	}

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	b = getWebsocketInstance(t, b)

	got, err := b.WebsocketModifyOrder(t.Context(), mod)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWSCancelOrder(t *testing.T) {
	t.Parallel()
	arg := &CancelOrderParams{}
	_, err := b.WSCancelOrder(t.Context(), arg)
	require.ErrorIs(t, err, errCategoryNotSet)

	arg.Category = cSpot
	_, err = b.WSCancelOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = currency.NewBTCUSDT()
	_, err = b.WSCancelOrder(t.Context(), arg)
	require.ErrorIs(t, err, errEitherOrderIDOROrderLinkIDRequired)

	arg.OrderID = "1793353687809485568" // Replace with a valid order ID

	arg.OrderFilter = "dodgy"
	_, err = b.WSCancelOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidOrderFilter)

	arg.Category = cLinear
	_, err = b.WSCancelOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidCategory)

	arg.Category = cSpot
	arg.OrderFilter = "Order"

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	b = getWebsocketInstance(t, b)
	got, err := b.WSCancelOrder(t.Context(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWebsocketCancelOrder(t *testing.T) {
	t.Parallel()
	cancel := &order.Cancel{
		OrderID:   "1793388409122024192", // Replace with a valid order ID
		Pair:      currency.NewBTCUSDT(),
		AssetType: asset.Spot,
	}

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	b = getWebsocketInstance(t, b)

	err := b.WebsocketCancelOrder(t.Context(), cancel)
	require.NoError(t, err)
}

// getWebsocketInstance returns a websocket instance copy for testing.
// This restricts the pairs to a single pair per asset type to reduce test time.
func getWebsocketInstance(t *testing.T, by *Bybit) *Bybit {
	t.Helper()
	cfg := &config.Config{}
	root, err := testutils.RootPathFromCWD()
	require.NoError(t, err)

	err = cfg.LoadConfig(filepath.Join(root, "testdata", "configtest.json"), true)
	require.NoError(t, err)

	cpy := new(Bybit)
	cpy.SetDefaults()
	bConf, err := cfg.GetExchangeConfig("Bybit")
	require.NoError(t, err)
	bConf.API.AuthenticatedSupport = true
	bConf.API.AuthenticatedWebsocketSupport = true
	bConf.API.Credentials.Key = apiKey
	bConf.API.Credentials.Secret = apiSecret

	require.NoError(t, cpy.Setup(bConf), "Test instance Setup must not error")
	cpy.CurrencyPairs.Load(&by.CurrencyPairs)

assetLoader:
	for _, a := range cpy.GetAssetTypes(true) {
		var avail currency.Pairs
		switch a {
		case asset.Spot:
			avail, err = cpy.GetAvailablePairs(a)
			require.NoError(t, err)
			if len(avail) > 1 { // reduce pairs to 1 to speed up tests
				avail = avail[:1]
			}
		default:
			require.NoError(t, cpy.CurrencyPairs.SetAssetEnabled(a, false))
			continue assetLoader
		}
		require.NoError(t, cpy.SetPairs(avail, a, true))
	}
	require.NoError(t, cpy.Websocket.Connect())
	return cpy
}
