package gateio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestGetP2PAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetP2PAccountInfo(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetP2PCounterpartyInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetP2PCounterpartyInfo(t.Context(), &GetCounterpartyInfoRequest{})
	require.ErrorIs(t, err, errBizUIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetP2PCounterpartyInfo(t.Context(), &GetCounterpartyInfoRequest{BizUID: "biz_uid_demo_0fbc1"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetP2PPaymentMethods(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetP2PPaymentMethods(t.Context(), &GetP2PPaymentMethodsRequest{})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetP2PPaymentMethods(t.Context(), &GetP2PPaymentMethodsRequest{Fiat: "USD"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetP2PPendingOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetP2PPendingOrders(t.Context(), &GetP2POrdersRequest{Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetP2PHistoricalOrders(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetP2PHistoricalOrders(t.Context(), endTime, startTime, 1, 10, nil)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetP2PHistoricalOrders(t.Context(), startTime, endTime, 1, 10, nil)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetP2POrderDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetP2POrderDetails(t.Context(), &GetP2POrderDetailsRequest{TransactionID: 0})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetP2POrderDetails(t.Context(), &GetP2POrderDetailsRequest{TransactionID: 40000001})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestConfirmP2PPayment(t *testing.T) {
	t.Parallel()
	err := e.ConfirmP2PPayment(t.Context(), &ConfirmP2PPaymentRequest{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.ConfirmP2PPayment(t.Context(), &ConfirmP2PPaymentRequest{TransactionID: "40000001", PaymentMethod: "bank"})
	require.NoError(t, err)
}

func TestConfirmP2PReceipt(t *testing.T) {
	t.Parallel()
	err := e.ConfirmP2PReceipt(t.Context(), &ConfirmP2PReceiptRequest{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.ConfirmP2PReceipt(t.Context(), &ConfirmP2PReceiptRequest{TransactionID: "40000001"})
	require.NoError(t, err)
}

func TestCancelP2POrder(t *testing.T) {
	t.Parallel()
	err := e.CancelP2POrder(t.Context(), &CancelP2POrderRequest{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.CancelP2POrder(t.Context(), &CancelP2POrderRequest{TransactionID: "100000", ReasonID: 6, ReasonMemo: "Cancelled after agreement with the counterparty"})
	require.NoError(t, err)
}

func TestPublishP2PAdOrder(t *testing.T) {
	t.Parallel()
	err := e.PublishP2PAdOrder(t.Context(), &PublishP2PAdRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg := &PublishP2PAdRequest{Asset: currency.USDT}
	err = e.PublishP2PAdOrder(t.Context(), arg)
	require.ErrorIs(t, err, errP2PFiatUnitRequired)

	arg.FiatUnit = "USD"
	err = e.PublishP2PAdOrder(t.Context(), arg)
	require.ErrorIs(t, err, errP2PTradeTypeRequired)

	arg.TradeType = "sell"
	err = e.PublishP2PAdOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	arg.PayIDs = "100002"
	err = e.PublishP2PAdOrder(t.Context(), arg)
	require.ErrorIs(t, err, errP2PPriceTypeInvalid)

	arg.PriceType = 2
	err = e.PublishP2PAdOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.MaxAmount = 500
	err = e.PublishP2PAdOrder(t.Context(), arg)
	require.ErrorIs(t, err, errP2PMinAmountRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	arg.MinAmount = 10
	arg.FixedPrice = "1.05"
	err = e.PublishP2PAdOrder(t.Context(), arg)
	require.NoError(t, err)
}

func TestUpdateP2PAdStatus(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateP2PAdStatus(t.Context(), &UpdateP2PAdStatusRequest{AdvNo: 0, AdvStatus: 1})
	require.ErrorIs(t, err, errP2PAdIDRequired)

	_, err = e.UpdateP2PAdStatus(t.Context(), &UpdateP2PAdStatusRequest{AdvNo: 2124000001, AdvStatus: 2})
	require.ErrorIs(t, err, errP2PAdStatusInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.UpdateP2PAdStatus(t.Context(), &UpdateP2PAdStatusRequest{AdvNo: 2124000001, AdvStatus: 3})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetP2PAdDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetP2PAdDetails(t.Context(), &GetP2PAdDetailsRequest{})
	require.ErrorIs(t, err, errP2PAdIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetP2PAdDetails(t.Context(), &GetP2PAdDetailsRequest{AdvNo: "2124000001"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMyP2PAds(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMyP2PAds(t.Context(), &GetMyP2PAdsRequest{})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetMyP2PAds(t.Context(), &GetMyP2PAdsRequest{Asset: currency.USDT, FiatUnit: "USD", TradeType: "sell"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetP2PAdList(t *testing.T) {
	t.Parallel()
	_, err := e.GetP2PAdList(t.Context(), &GetP2PAdsListRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetP2PAdList(t.Context(), &GetP2PAdsListRequest{Asset: currency.USDT})
	require.ErrorIs(t, err, errP2PFiatUnitRequired)

	_, err = e.GetP2PAdList(t.Context(), &GetP2PAdsListRequest{Asset: currency.USDT, FiatUnit: "USD"})
	require.ErrorIs(t, err, errP2PTradeTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetP2PAdList(t.Context(), &GetP2PAdsListRequest{Asset: currency.USDT, FiatUnit: "USD", TradeType: "sell"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetP2PChatHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetP2PChatHistory(t.Context(), 0, 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetP2PChatHistory(t.Context(), 40000001, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSendP2PChatMessage(t *testing.T) {
	t.Parallel()
	_, err := e.SendP2PChatMessage(t.Context(), &SendP2PChatMessageRequest{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	_, err = e.SendP2PChatMessage(t.Context(), &SendP2PChatMessageRequest{TransactionID: 40000001})
	require.ErrorIs(t, err, errP2PMessageRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SendP2PChatMessage(t.Context(), &SendP2PChatMessageRequest{TransactionID: 40000001, Message: "Payment completed, please check"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUploadP2PChatFile(t *testing.T) {
	t.Parallel()
	_, err := e.UploadP2PChatFile(t.Context(), &UploadP2PChatFileRequest{})
	require.ErrorIs(t, err, errP2PImageTypeRequired)

	_, err = e.UploadP2PChatFile(t.Context(), &UploadP2PChatFileRequest{ImageContentType: "image/png"})
	require.ErrorIs(t, err, errP2PImageDataRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.UploadP2PChatFile(t.Context(), &UploadP2PChatFileRequest{
		ImageContentType: "image/png",
		Base64Img:        "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}
