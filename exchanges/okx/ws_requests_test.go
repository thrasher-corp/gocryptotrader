package okx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestWSPlaceOrder(t *testing.T) {
	t.Parallel()

	_, err := ok.WSPlaceOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	out := &PlaceOrderRequestParam{
		InstrumentID: btcusdt,
		TradeMode:    TradeModeCash,
		Side:         "Buy",
		OrderType:    "limit",
		Amount:       0.0001,
		Price:        20000,
		Currency:     "USDT",
	}

	got, err := ok.WSPlaceOrder(request.WithVerbose(t.Context()), out)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWSPlaceMultipleOrder(t *testing.T) {
	t.Parallel()

	_, err := ok.WSPlaceMultipleOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil)

	_, err = ok.WSPlaceMultipleOrder(t.Context(), []PlaceOrderRequestParam{{}})
	require.ErrorIs(t, err, errMissingInstrumentID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	out := PlaceOrderRequestParam{
		InstrumentID: btcusdt,
		TradeMode:    TradeModeCash,
		Side:         "Buy",
		OrderType:    "limit",
		Amount:       0.0001,
		Price:        20000,
		Currency:     "USDT",
	}

	got, err := ok.WSPlaceMultipleOrder(request.WithVerbose(t.Context()), []PlaceOrderRequestParam{out})
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWSCancelOrder(t *testing.T) {
	t.Parallel()

	_, err := ok.WSCancelOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = ok.WSCancelOrder(t.Context(), &CancelOrderRequestParam{})
	require.ErrorIs(t, err, errMissingInstrumentID)

	_, err = ok.WSCancelOrder(t.Context(), &CancelOrderRequestParam{InstrumentID: btcusdt})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	got, err := ok.WSCancelOrder(request.WithVerbose(t.Context()), &CancelOrderRequestParam{InstrumentID: btcusdt, OrderID: "2341161427393388544"})
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWSCancleMultipleOrder(t *testing.T) {
	t.Parallel()

	_, err := ok.WSCancelMultipleOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil)

	_, err = ok.WSCancelMultipleOrder(t.Context(), []CancelOrderRequestParam{{}})
	require.ErrorIs(t, err, errMissingInstrumentID)

	_, err = ok.WSCancelMultipleOrder(t.Context(), []CancelOrderRequestParam{{InstrumentID: btcusdt}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	got, err := ok.WSCancelMultipleOrder(request.WithVerbose(t.Context()), []CancelOrderRequestParam{{InstrumentID: btcusdt, OrderID: "2341184920998715392"}})
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWSAmendOrder(t *testing.T) {
	t.Parallel()

	_, err := ok.WSAmendOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	out := &AmendOrderRequestParams{}
	_, err = ok.WSAmendOrder(t.Context(), out)
	require.ErrorIs(t, err, errMissingInstrumentID)

	out.InstrumentID = btcusdt
	_, err = ok.WSAmendOrder(t.Context(), out)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	out.OrderID = "2341200629875154944"
	_, err = ok.WSAmendOrder(t.Context(), out)
	require.ErrorIs(t, err, errInvalidNewSizeOrPriceInformation)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	out.NewPrice = 21000
	got, err := ok.WSAmendOrder(request.WithVerbose(t.Context()), out)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWSAmendMultipleOrders(t *testing.T) {
	t.Parallel()

	_, err := ok.WSAmendMultipleOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil)

	out := AmendOrderRequestParams{}
	_, err = ok.WSAmendMultipleOrders(t.Context(), []AmendOrderRequestParams{out})
	require.ErrorIs(t, err, errMissingInstrumentID)

	out.InstrumentID = btcusdt
	_, err = ok.WSAmendMultipleOrders(t.Context(), []AmendOrderRequestParams{out})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	out.OrderID = "2341200629875154944"
	_, err = ok.WSAmendMultipleOrders(t.Context(), []AmendOrderRequestParams{out})
	require.ErrorIs(t, err, errInvalidNewSizeOrPriceInformation)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	out.NewPrice = 20000

	got, err := ok.WSAmendMultipleOrders(request.WithVerbose(t.Context()), []AmendOrderRequestParams{out})
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWSMassCancelOrders(t *testing.T) {
	t.Parallel()
	_, err := ok.WSMassCancelOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil)

	_, err = ok.WSMassCancelOrders(t.Context(), []CancelMassReqParam{{}})
	require.ErrorIs(t, err, errInvalidInstrumentType)

	_, err = ok.WSMassCancelOrders(t.Context(), []CancelMassReqParam{{InstrumentType: "OPTION"}})
	require.ErrorIs(t, err, errInstrumentFamilyRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.WSMassCancelOrders(request.WithVerbose(t.Context()), []CancelMassReqParam{
		{
			InstrumentType:   "OPTION",
			InstrumentFamily: "BTC-USD",
		},
	})
	require.NoError(t, err)
	assert.True(t, result)
}

func TestWSPlaceSpreadOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.WSPlaceSpreadOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.WSPlaceSpreadOrder(request.WithVerbose(t.Context()), &SpreadOrderParam{
		SpreadID:      "BTC-USDT_BTC-USDT-SWAP",
		ClientOrderID: "b15",
		Side:          order.Buy.Lower(),
		OrderType:     "limit",
		Price:         2.15,
		Size:          2,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSAmendSpreadOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.WSAmendSpreadOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = ok.WSAmendSpreadOrder(t.Context(), &AmendSpreadOrderParam{NewSize: 2})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = ok.WSAmendSpreadOrder(t.Context(), &AmendSpreadOrderParam{OrderID: "2510789768709120"})
	require.ErrorIs(t, err, errSizeOrPriceIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.WSAmendSpreadOrder(request.WithVerbose(t.Context()), &AmendSpreadOrderParam{
		OrderID: "2510789768709120",
		NewSize: 2,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsCancelSpreadOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.WsCancelSpreadOrder(t.Context(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.WsCancelSpreadOrder(request.WithVerbose(t.Context()), "1234", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSCancelAllSpreadOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.WSCancelAllSpreadOrders(request.WithVerbose(t.Context()), "BTC-USDT_BTC-USDT-SWAP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}
