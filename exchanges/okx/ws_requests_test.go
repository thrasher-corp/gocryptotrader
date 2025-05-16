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

	_, err := ok.WSPlaceOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	testexch.SetupWs(t, ok)

	out := &PlaceOrderRequestParam{
		InstrumentID: mainPair.String(),
		TradeMode:    TradeModeIsolated, // depending on portfolio settings this can also be TradeModeCash
		Side:         "Buy",
		OrderType:    "post_only",
		Amount:       0.0001,
		Price:        20000,
		Currency:     "USDT",
	}

	got, err := ok.WSPlaceOrder(request.WithVerbose(t.Context()), out)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWSPlaceMultipleOrders(t *testing.T) {
	t.Parallel()

	_, err := ok.WSPlaceMultipleOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil)

	_, err = ok.WSPlaceMultipleOrders(t.Context(), []PlaceOrderRequestParam{{}})
	require.ErrorIs(t, err, errMissingInstrumentID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	testexch.SetupWs(t, ok)

	out := PlaceOrderRequestParam{
		InstrumentID: mainPair.String(),
		TradeMode:    TradeModeIsolated, // depending on portfolio settings this can also be TradeModeCash
		Side:         "Buy",
		OrderType:    "post_only",
		Amount:       0.0001,
		Price:        20000,
		Currency:     "USDT",
	}

	got, err := ok.WSPlaceMultipleOrders(request.WithVerbose(t.Context()), []PlaceOrderRequestParam{out})
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWSCancelOrder(t *testing.T) {
	t.Parallel()

	_, err := ok.WSCancelOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = ok.WSCancelOrder(t.Context(), &CancelOrderRequestParam{})
	require.ErrorIs(t, err, errMissingInstrumentID)

	_, err = ok.WSCancelOrder(t.Context(), &CancelOrderRequestParam{InstrumentID: mainPair.String()})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	testexch.SetupWs(t, ok)

	got, err := ok.WSCancelOrder(request.WithVerbose(t.Context()), &CancelOrderRequestParam{InstrumentID: mainPair.String(), OrderID: "2341161427393388544"})
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWSCancelMultipleOrders(t *testing.T) {
	t.Parallel()

	_, err := ok.WSCancelMultipleOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil)

	_, err = ok.WSCancelMultipleOrders(t.Context(), []CancelOrderRequestParam{{}})
	require.ErrorIs(t, err, errMissingInstrumentID)

	_, err = ok.WSCancelMultipleOrders(t.Context(), []CancelOrderRequestParam{{InstrumentID: mainPair.String()}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	testexch.SetupWs(t, ok)

	got, err := ok.WSCancelMultipleOrders(request.WithVerbose(t.Context()), []CancelOrderRequestParam{{InstrumentID: mainPair.String(), OrderID: "2341184920998715392"}})
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

	out.InstrumentID = mainPair.String()
	_, err = ok.WSAmendOrder(t.Context(), out)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	out.OrderID = "2341200629875154944"
	_, err = ok.WSAmendOrder(t.Context(), out)
	require.ErrorIs(t, err, errInvalidNewSizeOrPriceInformation)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	testexch.SetupWs(t, ok)

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

	out.InstrumentID = mainPair.String()
	_, err = ok.WSAmendMultipleOrders(t.Context(), []AmendOrderRequestParams{out})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	out.OrderID = "2341200629875154944"
	_, err = ok.WSAmendMultipleOrders(t.Context(), []AmendOrderRequestParams{out})
	require.ErrorIs(t, err, errInvalidNewSizeOrPriceInformation)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	testexch.SetupWs(t, ok)
	out.NewPrice = 20000

	got, err := ok.WSAmendMultipleOrders(request.WithVerbose(t.Context()), []AmendOrderRequestParams{out})
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWSMassCancelOrders(t *testing.T) {
	t.Parallel()
	err := ok.WSMassCancelOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil)

	err = ok.WSMassCancelOrders(t.Context(), []CancelMassReqParam{{}})
	require.ErrorIs(t, err, errInvalidInstrumentType)

	err = ok.WSMassCancelOrders(t.Context(), []CancelMassReqParam{{InstrumentType: "OPTION"}})
	require.ErrorIs(t, err, errInstrumentFamilyRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	testexch.SetupWs(t, ok)
	err = ok.WSMassCancelOrders(request.WithVerbose(t.Context()), []CancelMassReqParam{
		{
			InstrumentType:   "OPTION",
			InstrumentFamily: optionsPair.String(),
		},
	})
	require.NoError(t, err)
}

func TestWSPlaceSpreadOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.WSPlaceSpreadOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	testexch.SetupWs(t, ok)
	result, err := ok.WSPlaceSpreadOrder(request.WithVerbose(t.Context()), &SpreadOrderParam{
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
	_, err := ok.WSAmendSpreadOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = ok.WSAmendSpreadOrder(t.Context(), &AmendSpreadOrderParam{NewSize: 2})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = ok.WSAmendSpreadOrder(t.Context(), &AmendSpreadOrderParam{OrderID: "2510789768709120"})
	require.ErrorIs(t, err, errSizeOrPriceIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	testexch.SetupWs(t, ok)
	result, err := ok.WSAmendSpreadOrder(request.WithVerbose(t.Context()), &AmendSpreadOrderParam{
		OrderID: "2510789768709120",
		NewSize: 2,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSCancelSpreadOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.WSCancelSpreadOrder(t.Context(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	testexch.SetupWs(t, ok)
	result, err := ok.WSCancelSpreadOrder(request.WithVerbose(t.Context()), "1234", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSCancelAllSpreadOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	testexch.SetupWs(t, ok)
	err := ok.WSCancelAllSpreadOrders(request.WithVerbose(t.Context()), spreadPair.String())
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
