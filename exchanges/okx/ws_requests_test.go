package okx

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestWSPlaceOrder(t *testing.T) {
	t.Parallel()

	_, err := e.WSPlaceOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	testexch.SetupWs(t, e)

	out := &PlaceOrderRequestParam{
		InstrumentID: mainPair.String(),
		TradeMode:    TradeModeIsolated, // depending on portfolio settings this can also be TradeModeCash
		Side:         "Buy",
		OrderType:    "post_only",
		Amount:       0.0001,
		Price:        20000,
		Currency:     "USDT",
	}

	got, err := e.WSPlaceOrder(request.WithVerbose(t.Context()), out)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWSPlaceMultipleOrders(t *testing.T) {
	t.Parallel()

	_, err := e.WSPlaceMultipleOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil)

	_, err = e.WSPlaceMultipleOrders(t.Context(), []PlaceOrderRequestParam{{}})
	require.ErrorIs(t, err, errMissingInstrumentID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	testexch.SetupWs(t, e)

	out := PlaceOrderRequestParam{
		InstrumentID: mainPair.String(),
		TradeMode:    TradeModeIsolated, // depending on portfolio settings this can also be TradeModeCash
		Side:         "Buy",
		OrderType:    "post_only",
		Amount:       0.0001,
		Price:        20000,
		Currency:     "USDT",
	}

	got, err := e.WSPlaceMultipleOrders(request.WithVerbose(t.Context()), []PlaceOrderRequestParam{out})
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWSCancelOrder(t *testing.T) {
	t.Parallel()

	_, err := e.WSCancelOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.WSCancelOrder(t.Context(), &CancelOrderRequestParam{})
	require.ErrorIs(t, err, errMissingInstrumentID)

	_, err = e.WSCancelOrder(t.Context(), &CancelOrderRequestParam{InstrumentID: mainPair.String()})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	testexch.SetupWs(t, e)

	got, err := e.WSCancelOrder(request.WithVerbose(t.Context()), &CancelOrderRequestParam{InstrumentID: mainPair.String(), OrderID: "2341161427393388544"})
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWSCancelMultipleOrders(t *testing.T) {
	t.Parallel()

	_, err := e.WSCancelMultipleOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil)

	_, err = e.WSCancelMultipleOrders(t.Context(), []CancelOrderRequestParam{{}})
	require.ErrorIs(t, err, errMissingInstrumentID)

	_, err = e.WSCancelMultipleOrders(t.Context(), []CancelOrderRequestParam{{InstrumentID: mainPair.String()}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	testexch.SetupWs(t, e)

	got, err := e.WSCancelMultipleOrders(request.WithVerbose(t.Context()), []CancelOrderRequestParam{{InstrumentID: mainPair.String(), OrderID: "2341184920998715392"}})
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWSAmendOrder(t *testing.T) {
	t.Parallel()

	_, err := e.WSAmendOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	out := &AmendOrderRequestParams{}
	_, err = e.WSAmendOrder(t.Context(), out)
	require.ErrorIs(t, err, errMissingInstrumentID)

	out.InstrumentID = mainPair.String()
	_, err = e.WSAmendOrder(t.Context(), out)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	out.OrderID = "2341200629875154944"
	_, err = e.WSAmendOrder(t.Context(), out)
	require.ErrorIs(t, err, errInvalidNewSizeOrPriceInformation)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	testexch.SetupWs(t, e)

	out.NewPrice = 21000
	got, err := e.WSAmendOrder(request.WithVerbose(t.Context()), out)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWSAmendMultipleOrders(t *testing.T) {
	t.Parallel()

	_, err := e.WSAmendMultipleOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil)

	out := AmendOrderRequestParams{}
	_, err = e.WSAmendMultipleOrders(t.Context(), []AmendOrderRequestParams{out})
	require.ErrorIs(t, err, errMissingInstrumentID)

	out.InstrumentID = mainPair.String()
	_, err = e.WSAmendMultipleOrders(t.Context(), []AmendOrderRequestParams{out})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	out.OrderID = "2341200629875154944"
	_, err = e.WSAmendMultipleOrders(t.Context(), []AmendOrderRequestParams{out})
	require.ErrorIs(t, err, errInvalidNewSizeOrPriceInformation)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	testexch.SetupWs(t, e)
	out.NewPrice = 20000

	got, err := e.WSAmendMultipleOrders(request.WithVerbose(t.Context()), []AmendOrderRequestParams{out})
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWSMassCancelOrders(t *testing.T) {
	t.Parallel()
	err := e.WSMassCancelOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil)

	err = e.WSMassCancelOrders(t.Context(), []CancelMassReqParam{{}})
	require.ErrorIs(t, err, errInvalidInstrumentType)

	err = e.WSMassCancelOrders(t.Context(), []CancelMassReqParam{{InstrumentType: "OPTION"}})
	require.ErrorIs(t, err, errInstrumentFamilyRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	testexch.SetupWs(t, e)
	err = e.WSMassCancelOrders(request.WithVerbose(t.Context()), []CancelMassReqParam{
		{
			InstrumentType:   "OPTION",
			InstrumentFamily: optionsPair.String(),
		},
	})
	require.NoError(t, err)
}

func TestWSPlaceSpreadOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WSPlaceSpreadOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	testexch.SetupWs(t, e)
	result, err := e.WSPlaceSpreadOrder(request.WithVerbose(t.Context()), &SpreadOrderParam{
		SpreadID:      spreadPair.String(),
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
	_, err := e.WSAmendSpreadOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.WSAmendSpreadOrder(t.Context(), &AmendSpreadOrderParam{NewSize: 2})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.WSAmendSpreadOrder(t.Context(), &AmendSpreadOrderParam{OrderID: "2510789768709120"})
	require.ErrorIs(t, err, errSizeOrPriceIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	testexch.SetupWs(t, e)
	result, err := e.WSAmendSpreadOrder(request.WithVerbose(t.Context()), &AmendSpreadOrderParam{
		OrderID: "2510789768709120",
		NewSize: 2,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSCancelSpreadOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WSCancelSpreadOrder(t.Context(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	testexch.SetupWs(t, e)
	result, err := e.WSCancelSpreadOrder(request.WithVerbose(t.Context()), "1234", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSCancelAllSpreadOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	testexch.SetupWs(t, e)
	err := e.WSCancelAllSpreadOrders(request.WithVerbose(t.Context()), spreadPair.String())
	require.NoError(t, err)
}

type mockHasError struct {
	err error
}

func (m *mockHasError) Error() error {
	return m.err
}

func TestParseWSResponseErrors(t *testing.T) {
	t.Parallel()

	require.Panics(t, func() { _ = parseWSResponseErrors(123, nil) }, "result must be a pointer")
	require.Panics(t, func() { _ = parseWSResponseErrors(&mockHasError{}, nil) }, "result must be a slice")

	var emptySlice []*mockHasError
	require.NoError(t, parseWSResponseErrors(&emptySlice, nil))
	require.ErrorIs(t, parseWSResponseErrors(&emptySlice, errOperationFailed), errOperationFailed)

	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	mockSlice := []*mockHasError{{err: nil}, {err: err1}, {err: err2}}
	err := parseWSResponseErrors(&mockSlice, errPartialSuccess)
	require.ErrorIs(t, err, errPartialSuccess)
	require.ErrorIs(t, err, err1)
	require.ErrorIs(t, err, err2)
}

func TestSingleItem(t *testing.T) {
	t.Parallel()

	_, err := singleItem([]*any(nil))
	require.ErrorIs(t, err, common.ErrNoResponse)
	_, err = singleItem([]*mockHasError{{}, {}})
	require.ErrorIs(t, err, errMultipleItemsReturned)

	got, err := singleItem([]*mockHasError{{}})
	require.NoError(t, err)
	require.NotNil(t, got)
}
