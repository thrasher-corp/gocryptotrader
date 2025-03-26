package cryptodotcom

import (
	"context"
	"errors"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var (
	cr           = &Cryptodotcom{}
	tradablePair currency.Pair
)

func TestMain(m *testing.M) {
	cr = new(Cryptodotcom)
	if err := testexch.Setup(cr); err != nil {
		log.Fatal(err)
	}

	if apiKey != "" && apiSecret != "" {
		cr.API.AuthenticatedSupport = true
		cr.API.AuthenticatedWebsocketSupport = true
		cr.API.CredentialsValidator.RequiresBase64DecodeSecret = false
		cr.SetCredentials(apiKey, apiSecret, "", "", "", "")
		cr.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}
	err := initTradablePair()
	if err != nil {
		log.Fatal(err)
	}
	setupWS()
	os.Exit(m.Run())
}

func initTradablePair() error {
	err := cr.UpdateTradablePairs(context.Background(), false)
	if err != nil {
		return err
	}
	enabledPairs, err := cr.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	} else if len(enabledPairs) == 0 {
		return errors.New("No enabled pairs found")
	}
	tradablePair = enabledPairs[0]
	return nil
}

func TestGetRiskParameters(t *testing.T) {
	t.Parallel()
	result, err := cr.GetRiskParameters(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	result, err := cr.GetInstruments(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	result, err := cr.GetOrderbook(context.Background(), tradablePair.String(), 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCandlestickDetail(t *testing.T) {
	t.Parallel()
	result, err := cr.GetCandlestickDetail(context.Background(), tradablePair.String(), kline.FiveMin)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	result, err := cr.GetTickers(context.Background(), tradablePair.String())
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = cr.GetTickers(context.Background(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := cr.GetTrades(context.Background(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := cr.GetTrades(context.Background(), tradablePair.String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetValuations(t *testing.T) {
	t.Parallel()
	_, err := cr.GetValuations(context.Background(), "", "index_price", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = cr.GetValuations(context.Background(), tradablePair.String(), "", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errValuationTypeUnset)

	result, err := cr.GetValuations(context.Background(), "BTCUSD-INDEX", "index_price", 0, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawFunds(t *testing.T) {
	t.Parallel()
	_, err := cr.WithdrawFunds(context.Background(), currency.EMPTYCODE, 10, core.BitcoinDonationAddress, "", "", "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = cr.WithdrawFunds(context.Background(), currency.BTC, 0, core.BitcoinDonationAddress, "", "", "")
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = cr.WithdrawFunds(context.Background(), currency.BTC, 10, "", "", "", "")
	require.ErrorIs(t, err, errAddressRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr, canManipulateRealOrders)
	result, err := cr.WithdrawFunds(context.Background(), currency.BTC, 10, core.BitcoinDonationAddress, "", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}
func TestWsCreateWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := cr.WsCreateWithdrawal(currency.EMPTYCODE, 10, core.BitcoinDonationAddress, "", "", "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = cr.WsCreateWithdrawal(currency.BTC, 0, core.BitcoinDonationAddress, "", "", "")
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = cr.WsCreateWithdrawal(currency.BTC, 10, "", "", "", "")
	require.ErrorIs(t, err, errAddressRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr, canManipulateRealOrders)
	result, err := cr.WsCreateWithdrawal(currency.BTC, 10, core.BitcoinDonationAddress, "", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrencyNetworks(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.GetCurrencyNetworks(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalHistory(t *testing.T) {
	t.Parallel()
	var resp *WithdrawalResponse
	err := json.Unmarshal([]byte(getWithdrawalHistory), &resp)
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.GetWithdrawalHistory(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}
func TestWsRetriveWithdrawalHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.WsRetriveWithdrawalHistory()
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.GetDepositHistory(context.Background(), currency.EMPTYCODE, time.Time{}, time.Time{}, 20, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPersonalDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := cr.GetPersonalDepositAddress(context.Background(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.GetPersonalDepositAddress(context.Background(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateExportRequest(t *testing.T) {
	t.Parallel()
	_, err := cr.CreateExportRequest(context.Background(), tradablePair.String(), "", time.Now().Add(-time.Hour*240), time.Now(), []string{})
	require.ErrorIs(t, err, errRequestedDataTypesRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr, canManipulateRealOrders)
	result, err := cr.CreateExportRequest(context.Background(), tradablePair.String(), "", time.Now().Add(-time.Hour*240), time.Now(), []string{"SPOT_ORDER", "SPOT_TRADE"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetExportRequests(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.GetExportRequests(context.Background(), tradablePair.String(), time.Time{}, time.Time{}, []string{"SPOT_ORDER"}, 10, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountSummary(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.GetAccountSummary(context.Background(), currency.USDT)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsRetriveAccountSummary(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.WsRetriveAccountSummary(currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateOrder(t *testing.T) {
	t.Parallel()
	_, err := cr.CreateOrder(context.Background(), &CreateOrderParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &CreateOrderParam{
		PostOnly: true,
	}
	_, err = cr.CreateOrder(context.Background(), arg)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	arg.Symbol = "BTC_USDT"
	_, err = cr.CreateOrder(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Buy
	_, err = cr.CreateOrder(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = order.StopLimit
	_, err = cr.CreateOrder(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	arg.Price = 123
	_, err = cr.CreateOrder(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.Quantity = 1
	_, err = cr.CreateOrder(context.Background(), arg)
	require.ErrorIs(t, err, errTriggerPriceRequired)

	arg.OrderType = order.Market
	arg.Quantity = 0
	arg.Side = order.Buy
	_, err = cr.CreateOrder(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrAmountMustBeSet)

	arg.Side = order.Sell
	_, err = cr.CreateOrder(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.OrderType = order.StopLoss
	_, err = cr.CreateOrder(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.Side = order.Sell
	_, err = cr.CreateOrder(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.Side = order.Buy
	arg.Notional = 1
	_, err = cr.CreateOrder(context.Background(), arg)
	require.ErrorIs(t, err, errTriggerPriceRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr, canManipulateRealOrders)
	result, err := cr.CreateOrder(context.Background(), &CreateOrderParam{
		Symbol:    tradablePair.String(),
		Side:      order.Buy,
		OrderType: order.Limit,
		Price:     123,
		Quantity:  12})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsPlaceOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr, canManipulateRealOrders)
	arg := &CreateOrderParam{Symbol: tradablePair.String(), Side: order.Buy, OrderType: order.Limit, Price: 123, Quantity: 12}
	result, err := cr.WsPlaceOrder(arg)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()
	err := cr.CancelExistingOrder(context.Background(), "", "1232412")
	assert.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	err = cr.CancelExistingOrder(context.Background(), tradablePair.String(), "")
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr, canManipulateRealOrders)
	err = cr.CancelExistingOrder(context.Background(), tradablePair.String(), "1232412")
	assert.NoError(t, err)
}
func TestWsCancelExistingOrder(t *testing.T) {
	t.Parallel()
	err := cr.WsCancelExistingOrder("", "1232412")
	assert.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	err = cr.WsCancelExistingOrder(tradablePair.String(), "")
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr, canManipulateRealOrders)
	err = cr.WsCancelExistingOrder(tradablePair.String(), "1232412")
	assert.NoError(t, err)
}

func TestGetPrivateTrades(t *testing.T) {
	t.Parallel()
	var resp *PersonalTrades
	err := json.Unmarshal([]byte(getPrivateTrades), &resp)
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.GetPrivateTrades(context.Background(), "", time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsRetrivePrivateTrades(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.WsRetrivePrivateTrades("", time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := cr.GetOrderDetail(context.Background(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.GetOrderDetail(context.Background(), "1234")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsRetriveOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := cr.WsRetriveOrderDetail("")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.WsRetriveOrderDetail("1234")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPersonalOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.GetPersonalOpenOrders(context.Background(), "", 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}
func TestWsRetrivePersonalOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.WsRetrivePersonalOpenOrders("", 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPersonalOrderHistory(t *testing.T) {
	t.Parallel()
	var resp *PersonalOrdersResponse
	err := json.Unmarshal([]byte(getPersonalOrderHistory), &resp)
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.GetPersonalOrderHistory(context.Background(), "", time.Time{}, time.Time{}, 0, 20)
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = cr.WsRetrivePersonalOrderHistory("", time.Time{}, time.Time{}, 0, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateOrderList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr, canManipulateRealOrders)
	result, err := cr.CreateOrderList(context.Background(), "LIST", []CreateOrderParam{
		{
			Symbol: tradablePair.String(), ClientOrderID: "", TimeInForce: "", Side: order.Buy, OrderType: order.Limit, PostOnly: false, TriggerPrice: 0, Price: 123, Quantity: 12, Notional: 0,
		}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}
func TestWsCreateOrderList(t *testing.T) {
	t.Parallel()
	_, err := cr.WsCreateOrderList("LIST", []CreateOrderParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr, canManipulateRealOrders)
	result, err := cr.WsCreateOrderList("LIST", []CreateOrderParam{
		{
			Symbol: tradablePair.String(), ClientOrderID: "", TimeInForce: "", Side: order.Buy, OrderType: order.Limit, PostOnly: false, TriggerPrice: 0, Price: 123, Quantity: 12, Notional: 0,
		}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOrderList(t *testing.T) {
	t.Parallel()
	_, err := cr.CancelOrderList(context.Background(), []CancelOrderParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = cr.CancelOrderList(context.Background(), []CancelOrderParam{{InstrumentName: "", OrderID: ""}})
	require.ErrorIs(t, err, errInstrumentNameOrOrderIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr, canManipulateRealOrders)
	result, err := cr.CancelOrderList(context.Background(), []CancelOrderParam{
		{InstrumentName: tradablePair.String(), OrderID: "1234567"}, {InstrumentName: tradablePair.String(),
			OrderID: "123450067"}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsCancelOrderList(t *testing.T) {
	t.Parallel()
	_, err := cr.WsCancelOrderList([]CancelOrderParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = cr.WsCancelOrderList([]CancelOrderParam{{InstrumentName: "", OrderID: ""}})
	require.ErrorIs(t, err, errInstrumentNameOrOrderIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr, canManipulateRealOrders)
	result, err := cr.WsCancelOrderList([]CancelOrderParam{
		{InstrumentName: tradablePair.String(), OrderID: "1234567"}, {InstrumentName: tradablePair.String(),
			OrderID: "123450067"}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllPersonalOrders(t *testing.T) {
	t.Parallel()
	err := cr.CancelAllPersonalOrders(context.Background(), "")
	assert.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr, canManipulateRealOrders)
	err = cr.CancelAllPersonalOrders(context.Background(), tradablePair.String())
	assert.NoError(t, err)
}

func TestWsCancelAllPersonalOrders(t *testing.T) {
	t.Parallel()
	err := cr.WsCancelAllPersonalOrders("")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr, canManipulateRealOrders)
	err = cr.WsCancelAllPersonalOrders(tradablePair.String())
	assert.NoError(t, err)
}

func TestGetAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.GetAccounts(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubAccountTransfer(t *testing.T) {
	t.Parallel()
	err := cr.SubAccountTransfer(context.Background(), "", "12345678-0000-0000-0000-000000000002", currency.BTC, 0.0000001)
	assert.ErrorIs(t, err, errSubAccountAddressRequired)
	err = cr.SubAccountTransfer(context.Background(), "12345678-0000-0000-0000-000000000001", "", currency.BTC, 0.0000001)
	assert.ErrorIs(t, err, errSubAccountAddressRequired)
	err = cr.SubAccountTransfer(context.Background(), "12345678-0000-0000-0000-000000000001", "12345678-0000-0000-0000-000000000002", currency.EMPTYCODE, 0.0000001)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	err = cr.SubAccountTransfer(context.Background(), "12345678-0000-0000-0000-000000000001", "12345678-0000-0000-0000-000000000002", currency.BTC, 0)
	assert.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr, canManipulateRealOrders)
	err = cr.SubAccountTransfer(context.Background(), "12345678-0000-0000-0000-000000000001", "12345678-0000-0000-0000-000000000002", currency.BTC, 0.0000001)
	assert.NoError(t, err)
}

func TestGetTransactions(t *testing.T) {
	t.Parallel()
	var resp *TransactionResponse
	err := json.Unmarshal([]byte(getTransactions), &resp)
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.GetTransactions(context.Background(), "BTCUSD-PERP", "", time.Time{}, time.Time{}, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserAccountFeeRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.GetUserAccountFeeRate(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFeeRateForUserAccount(t *testing.T) {
	t.Parallel()
	_, err := cr.GetInstrumentFeeRate(context.Background(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.GetInstrumentFeeRate(context.Background(), tradablePair.String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateSubAccountTransfer(t *testing.T) {
	t.Parallel()
	err := cr.CreateSubAccountTransfer(context.Background(), "", core.BitcoinDonationAddress, currency.USDT, 1232)
	assert.ErrorIs(t, err, errSubAccountAddressRequired)
	err = cr.CreateSubAccountTransfer(context.Background(), "destination_address", "", currency.USDT, 1232)
	assert.ErrorIs(t, err, errSubAccountAddressRequired)
	err = cr.CreateSubAccountTransfer(context.Background(), "destination_address", core.BitcoinDonationAddress, currency.EMPTYCODE, 1232)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	err = cr.CreateSubAccountTransfer(context.Background(), "destination_address", core.BitcoinDonationAddress, currency.USDT, 0)
	assert.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr, canManipulateRealOrders)
	err = cr.CreateSubAccountTransfer(context.Background(), "destination_address", core.BitcoinDonationAddress, currency.USDT, 1232)
	assert.NoError(t, err)
}

func TestGetOTCUser(t *testing.T) {
	t.Parallel()
	var resp *OTCTrade
	err := json.Unmarshal([]byte(getOtcUser), &resp)
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.GetOTCUser(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOTCInstruments(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.GetOTCInstruments(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRequestOTCQuote(t *testing.T) {
	t.Parallel()
	var resp *OTCQuoteResponse
	err := json.Unmarshal([]byte(requestOTCQuote), &resp)
	require.NoError(t, err)

	_, err = cr.RequestOTCQuote(context.Background(), currency.EMPTYPAIR, .001, 232, "BUY")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = cr.RequestOTCQuote(context.Background(), currency.NewPair(currency.BTC, currency.USDT), 0, 0, "BUY")
	require.ErrorIs(t, err, order.ErrAmountMustBeSet)
	_, err = cr.RequestOTCQuote(context.Background(), currency.NewPair(currency.BTC, currency.USDT), .001, 232, "")
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr, canManipulateRealOrders)
	result, err := cr.RequestOTCQuote(context.Background(), currency.NewPair(currency.BTC, currency.USDT), .001, 232, "BUY")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAcceptOTCQuote(t *testing.T) {
	t.Parallel()
	var resp *AcceptQuoteResponse
	err := json.Unmarshal([]byte(acceptOTCQuote), &resp)
	require.NoError(t, err)

	_, err = cr.AcceptOTCQuote(context.Background(), "", "")
	require.ErrorIs(t, err, errQuoteIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.AcceptOTCQuote(context.Background(), "12323123", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOTCQuoteHistory(t *testing.T) {
	t.Parallel()
	var resp *QuoteHistoryResponse
	err := json.Unmarshal([]byte(getOTCQuoteHistory), &resp)
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.GetOTCQuoteHistory(context.Background(), currency.EMPTYPAIR, time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOTCTradeHistory(t *testing.T) {
	t.Parallel()
	var resp *OTCTradeHistoryResponse
	err := json.Unmarshal([]byte(getOTCTradeHistory), &resp)
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.GetOTCTradeHistory(context.Background(), currency.NewPair(currency.BTC, currency.USDT), time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateOTCOrder(t *testing.T) {
	t.Parallel()
	_, err := cr.CreateOTCOrder(context.Background(), "", "BUY", "3427401068340147456", 0.0001, 12321, false)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = cr.CreateOTCOrder(context.Background(), tradablePair.String(), "BUY", "3427401068340147456", 0, 12321, false)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = cr.CreateOTCOrder(context.Background(), tradablePair.String(), "BUY", "3427401068340147456", 0.0001, 0, false)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)
	_, err = cr.CreateOTCOrder(context.Background(), tradablePair.String(), "", "3427401068340147456", 0.0001, 12321, false)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr, canManipulateRealOrders)
	result, err := cr.CreateOTCOrder(context.Background(), tradablePair.String(), "BUY", "3427401068340147456", 0.0001, 12321, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// wrapper test functions

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	result, err := cr.FetchTradablePairs(context.Background(), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	result, err := cr.UpdateTicker(context.Background(), tradablePair, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := cr.UpdateTickers(context.Background(), asset.Spot)
	assert.NoError(t, err)
}

func TestFetchTicker(t *testing.T) {
	t.Parallel()
	result, err := cr.FetchTicker(context.Background(), tradablePair, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFetchOrderbook(t *testing.T) {
	t.Parallel()
	result, err := cr.FetchOrderbook(context.Background(), tradablePair, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	result, err := cr.UpdateOrderbook(context.Background(), tradablePair, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.UpdateAccountInfo(context.Background(), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	result, err := cr.GetRecentTrades(context.Background(), currency.NewPair(currency.BTC, currency.USDT), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := cr.GetHistoricTrades(context.Background(), currency.NewPair(currency.BTC, currency.USDT), asset.Spot, time.Now().Add(-time.Hour*4), time.Now())
	assert.NoError(t, err)
}

func TestGetFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.GetAccountFundingHistory(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	startTime := time.Now().Add(-time.Minute * 40)
	endTime := time.Now()

	result, err := cr.GetHistoricCandles(context.Background(), tradablePair, asset.Spot, kline.OneDay, startTime, endTime)
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = cr.GetHistoricCandles(context.Background(), tradablePair, asset.Spot, kline.FiveMin, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	var getOrdersRequest = order.MultiOrderRequest{
		Type:      order.Limit,
		Pairs:     currency.Pairs{tradablePair, currency.NewPair(currency.USDT, currency.USD), currency.NewPair(currency.USD, currency.LTC)},
		AssetType: asset.Spot,
		Side:      order.Buy,
	}
	result, err := cr.GetActiveOrders(context.Background(), &getOrdersRequest)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	var getOrdersRequest = order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.Buy,
	}
	result, err := cr.GetOrderHistory(context.Background(), &getOrdersRequest)
	require.NoError(t, err)
	require.NotNil(t, result)

	getOrdersRequest.Pairs = []currency.Pair{currency.NewPair(currency.LTC, currency.BTC)}
	result, err = cr.GetOrderHistory(context.Background(), &getOrdersRequest)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr, canManipulateRealOrders)
	var orderSubmission = &order.Submit{
		Pair: currency.Pair{
			Base:  currency.LTC,
			Quote: currency.BTC,
		},
		Exchange:  cr.Name,
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1000000000,
		ClientID:  "myOwnOrder",
		AssetType: asset.Spot,
	}
	result, err := cr.SubmitOrder(context.Background(), orderSubmission)
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = cr.SubmitOrder(context.Background(), &order.Submit{
		Pair: currency.Pair{
			Base:  currency.LTC,
			Quote: currency.BTC,
		},
		Exchange:  cr.Name,
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1000000000,
		ClientID:  "myOwnOrder",
		AssetType: asset.Spot,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}
func TestCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr, canManipulateRealOrders)
	err := cr.CancelOrder(context.Background(), &order.Cancel{
		OrderID:   "1",
		Pair:      currency.NewPair(currency.LTC, currency.BTC),
		AssetType: asset.Spot,
	})
	assert.NoError(t, err)
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr, canManipulateRealOrders)
	result, err := cr.CancelBatchOrders(context.Background(), []order.Cancel{
		{
			OrderID: "1",
			Pair:    currency.NewPair(currency.LTC, currency.BTC),
		},
		{
			OrderID: "1",
			Pair:    currency.NewPair(currency.LTC, currency.BTC),
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr, canManipulateRealOrders)
	result, err := cr.CancelAllOrders(context.Background(), &order.Cancel{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.GetOrderInfo(context.Background(),
		"123", tradablePair, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.GetDepositAddress(context.Background(), currency.ETH, "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.WithdrawCryptocurrencyFunds(context.Background(), &withdraw.Request{
		Amount:   10,
		Currency: currency.BTC,
		Crypto: withdraw.CryptoRequest{
			Chain:      currency.BTC.String(),
			Address:    core.BitcoinDonationAddress,
			AddressTag: "",
		}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func setupWS() {
	if !cr.Websocket.IsEnabled() {
		return
	}
	if !sharedtestvalues.AreAPICredentialsSet(cr) {
		cr.Websocket.SetCanUseAuthenticatedEndpoints(false)
	}
	err := cr.WsConnect()
	if err != nil {
		log.Fatal(err)
	}
}

func TestGenerateDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	result, err := cr.GenerateDefaultSubscriptions()
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsRetriveCancelOnDisconnect(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.WsRetriveCancelOnDisconnect()
	require.NoError(t, err)
	assert.NotNil(t, result)
}
func TestWsSetCancelOnDisconnect(t *testing.T) {
	t.Parallel()
	_, err := cr.WsSetCancelOnDisconnect("")
	require.ErrorIs(t, err, errInvalidOrderCancellationScope)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr, canManipulateRealOrders)
	result, err := cr.WsSetCancelOnDisconnect("ACCOUNT")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCreateParamMap(t *testing.T) {
	t.Parallel()
	arg := &CreateOrderParam{Symbol: "", OrderType: order.Limit, Price: 123, Quantity: 12}
	_, err := arg.getCreateParamMap()
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	var newone *CreateOrderParam
	_, err = newone.getCreateParamMap()
	require.ErrorIs(t, err, common.ErrNilPointer)
	arg.Symbol = tradablePair.String()
	_, err = arg.getCreateParamMap()
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	arg.Side = order.Buy
	_, err = arg.getCreateParamMap()
	require.NoError(t, err)
	arg.OrderType = order.Market
	_, err = arg.getCreateParamMap()
	require.NoError(t, err)
	arg.OrderType = order.TakeProfit
	arg.Notional = 12
	_, err = arg.getCreateParamMap()
	require.ErrorIs(t, err, errTriggerPriceRequired)
	arg.OrderType = order.UnknownType
	_, err = arg.getCreateParamMap()
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)
	arg.OrderType = order.StopLimit
	_, err = arg.getCreateParamMap()
	require.ErrorIs(t, err, errTriggerPriceRequired)

	arg.TriggerPrice = .432423
	result, err := arg.getCreateParamMap()
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	feeBuilder := &exchange.FeeBuilder{
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          currency.NewPair(currency.BTC, currency.USD),
		IsMaker:       true,
		Amount:        1,
		PurchasePrice: 1000,
	}
	result, err := cr.GetFeeByType(context.Background(), feeBuilder)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	if !sharedtestvalues.AreAPICredentialsSet(cr) {
		assert.Equal(t, exchange.OfflineTradeFee, feeBuilder.FeeType)
	} else {
		assert.Equal(t, exchange.CryptocurrencyTradeFee, feeBuilder.FeeType)
	}
}

var pushDataMap = map[string]string{
	"Orderbook":     `{ "id": -1, "code": 0, "method": "subscribe", "result": { "channel": "book", "subscription": "book.RSR_USDT", "instrument_name": "RSR_USDT", "depth": 150, "data": [ { "asks": [ [ "0.0041045", "164840", "1" ], [ "0.0041057", "273330", "1" ], [ "0.0041116", "6440", "1" ], [ "0.0041159", "29490", "1" ], [ "0.0041185", "21940", "1" ], [ "0.0041238", "191790", "2" ], [ "0.0041317", "495840", "2" ], [ "0.0041396", "1117990", "1" ], [ "0.0041475", "1430830", "1" ], [ "0.0041528", "785220", "1" ], [ "0.0041554", "1409330", "1" ], [ "0.0041633", "1710820", "1" ], [ "0.0041712", "2399680", "1" ], [ "0.0041791", "2355400", "1" ], [ "0.0042500", "1500", "1" ], [ "0.0044000", "1000", "1" ], [ "0.0045000", "1000", "1" ], [ "0.0046600", "85770", "1" ], [ "0.0049230", "20660", "1" ], [ "0.0049380", "88520", "2" ], [ "0.0050000", "1120", "1" ], [ "0.0050203", "304960", "2" ], [ "0.0051026", "509200", "2" ], [ "0.0051849", "3452290", "1" ], [ "0.0052672", "10928750", "1" ], [ "0.0206000", "730", "1" ], [ "0.0406000", "370", "1" ] ], "bids": [ [ "0.0041013", "273330", "1" ], [ "0.0040975", "3750", "1" ], [ "0.0040974", "174120", "1" ], [ "0.0040934", "6440", "1" ], [ "0.0040922", "32200", "1" ], [ "0.0040862", "21940", "1" ], [ "0.0040843", "187900", "2" ], [ "0.0040764", "483650", "3" ], [ "0.0040686", "12280", "1" ], [ "0.0040685", "813180", "3" ], [ "0.0040607", "16020", "1" ], [ "0.0040606", "1123210", "3" ], [ "0.0040527", "1432240", "3" ], [ "0.0040482", "642210", "1" ], [ "0.0040448", "1441580", "2" ], [ "0.0040369", "2071370", "2" ], [ "0.0040290", "1453600", "1" ], [ "0.0037500", "29390", "1" ], [ "0.0033776", "80", "1" ], [ "0.0033740", "29630", "1" ], [ "0.0033000", "50", "1" ], [ "0.0032797", "30990", "1" ], [ "0.0032097", "175720", "2" ], [ "0.0032000", "50", "1" ], [ "0.0031274", "511460", "2" ], [ "0.0031000", "50", "1" ], [ "0.0030451", "793150", "2" ], [ "0.0030400", "750000", "1" ], [ "0.0030000", "100", "1" ], [ "0.0029628", "5620050", "2" ], [ "0.0029000", "50", "1" ], [ "0.0028805", "20567780", "2" ], [ "0.0018000", "500", "1" ], [ "0.0014500", "500", "1" ] ], "t": 1679082891435, "tt": 1679082890266, "u": 27043535761920, "cs": 723295208 } ] } }`,
	"Ticker":        `{ "id": -1, "code": 0, "method": "subscribe", "result": { "channel": "ticker", "instrument_name": "RSR_USDT", "subscription": "ticker.RSR_USDT", "id": -1, "data": [ { "h": "0.0041622", "l": "0.0037959", "a": "0.0040738", "c": "0.0721", "b": "0.0040738", "bs": "3680", "k": "0.0040796", "ks": "179780", "i": "RSR_USDT", "v": "45133400", "vv": "181223.95", "oi": "0","t": 1679087156318}]}}`,
	"Trade":         `{"id": 140466243, "code": 0, "method": "subscribe", "result": { "channel": "trade", "subscription": "trade.RSR_USDT", "instrument_name": "RSR_USDT", "data": [ { "d": "4611686018428182866", "t": 1679085786004, "p": "0.0040604", "q": "10", "s": "BUY", "i": "RSR_USDT" }, { "d": "4611686018428182865", "t": 1679085717204, "p": "0.0040671", "q": "10", "s": "BUY", "i": "RSR_USDT" }, { "d": "4611686018428182864", "t": 1679085672504, "p": "0.0040664", "q": "10", "s": "BUY", "i": "RSR_USDT" }, { "d": "4611686018428182863", "t": 1679085638806, "p": "0.0040674", "q": "10", "s": "BUY", "i": "RSR_USDT" }, { "d": "4611686018428182862", "t": 1679085568762, "p": "0.0040689", "q": "20", "s": "BUY", "i": "RSR_USDT" } ] } }`,
	"Candlestick":   `{"id": -1, "code": 0, "method": "subscribe", "result": { "channel": "candlestick", "instrument_name": "RSR_USDT", "subscription": "candlestick.5m.RSR_USDT", "interval": "5m", "data": [ { "o": "0.0040838", "h": "0.0040920", "l": "0.0040838", "c": "0.0040920", "v": "60.0000", "t": 1679087700000, "ut": 1679087959106 } ] } }`,
	"User Balance":  `{"id":3397447550047468012,"method":"subscribe","code":0,"result":{"subscription":"user.balance","channel":"user.balance","data":[{"stake":0,"balance":7.26648846,"available":7.26648846,"currency":"BOSON","order":0},{"stake":0,"balance":15.2782122,"available":15.2782122,"currency":"EFI","order":0},{"stake":0,"balance":90.63857968,"available":90.63857968,"currency":"ZIL","order":0},{"stake":0,"balance":16790279.87929312,"available":16790279.87929312,"currency":"SHIB","order":0},{"stake":0,"balance":1.79673318,"available":1.79673318,"currency":"NEAR","order":0},{"stake":0,"balance":307.29679422,"available":307.29679422,"currency":"DOGE","order":0},{"stake":0,"balance":0.00109125,"available":0.00109125,"currency":"BTC","order":0},{"stake":0,"balance":18634.17320776,"available":18634.17320776,"currency":"CRO-STAKE","order":0},{"stake":0,"balance":0.4312475,"available":0.4312475,"currency":"DOT","order":0},{"stake":0,"balance":924.07197632,"available":924.07197632,"currency":"CRO","order":0}]}}`,
	"User Order":    `{"method": "subscribe", "result": { "instrument_name": "ETH_CRO", "subscription": "user.order.ETH_CRO", "channel": "user.order", "data": [ { "status": "ACTIVE", "side": "BUY", "price": 1, "quantity": 1, "order_id": "366455245775097673", "client_oid": "my_order_0002", "create_time": 1588758017375, "update_time": 1588758017411, "type": "LIMIT", "instrument_name": "ETH_CRO", "cumulative_quantity": 0, "cumulative_value": 0, "avg_price": 0, "fee_currency": "CRO", "time_in_force":"GOOD_TILL_CANCEL" } ], "channel": "user.order.ETH_CRO"}}`,
	"User Trade":    `{"method": "subscribe", "code": 0, "result": { "instrument_name": "ETH_CRO", "subscription": "user.trade.ETH_CRO", "channel": "user.trade", "data": [ { "side": "SELL", "instrument_name": "ETH_CRO", "fee": 0.014, "trade_id": "367107655537806900", "create_time": "1588777459755", "traded_price": 7, "traded_quantity": 1, "fee_currency": "CRO", "order_id": "367107623521528450" } ], "channel": "user.trade.ETH_CRO" }}`,
	"OTC Orderbook": `{ "id": 1, "code": 0, "method": "subscribe", "result": { "channel": "otc_book", "subscription": "otc_book.BTC_USDT", "instrument_name": "BTC_USDT", "t": 1667800910315, "data": [ { "asks": [ ["8944.4", "1", "1", 1672502400000, 1510419685596942874], ["8955.1", "3", "1", 1672502400000, 1510419685596942875] ], "bids": [ ["8940.5", "1", "1", 1672502400000, 1510419685596942876], ["8918.7", "3", "1", 1672502400000, 1510419685596942877]]}]}}`,
}

const (
	getWithdrawalHistory    = `{ "withdrawal_list": [ { "currency": "XRP", "client_wid": "my_withdrawal_002", "fee": 1.0, "create_time": 1607063412000, "id": "2220", "update_time": 1607063460000, "amount": 100, "address": "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBf?1234567890", "status": "1", "txid": "", "network_id": null }]}`
	getPrivateTrades        = `{ "trade_list": [ { "side": "SELL", "instrument_name": "ETH_CRO", "fee": 0.014, "trade_id": "367107655537806900", "create_time": 1588777459755, "traded_price": 7, "traded_quantity": 1, "fee_currency": "CRO", "order_id": "367107623521528450"}]}`
	getPersonalOrderHistory = `{ "order_list": [ { "status": "FILLED", "side": "SELL", "price": 1, "quantity": 1, "order_id": "367107623521528457", "client_oid": "my_order_0002", "create_time": 1588777459755, "update_time": 1588777460700, "type": "LIMIT", "instrument_name": "ETH_CRO", "cumulative_quantity": 1, "cumulative_value": 1, "avg_price": 1, "fee_currency": "CRO", "time_in_force": "GOOD_TILL_CANCEL" }, { "status": "FILLED", "side": "SELL", "price": 1, "quantity": 1, "order_id": "367063282527104905", "client_oid": "my_order_0002", "create_time": 1588776138290, "update_time": 1588776138679, "type": "LIMIT", "instrument_name": "ETH_CRO", "cumulative_quantity": 1, "cumulative_value": 1, "avg_price": 1, "fee_currency": "CRO", "time_in_force": "GOOD_TILL_CANCEL"}]}`
	getTransactions         = `{ "data": [ { "account_id": "88888888-8888-8888-8888-000000000007", "event_date": "2021-02-18", "journal_type": "TRADING", "journal_id": "187078", "transaction_qty": "-0.0005", "transaction_cost": "-24.500000", "realized_pnl": "-0.006125", "order_id": "72062", "trade_id": "71497", "trade_match_id": "8625", "event_timestamp_ms": 1613640752166, "event_timestamp_ns": "1613640752166234567", "client_oid": "6ac2421d-5078-4ef6-a9d5-9680602ce123", "taker_side": "MAKER", "side": "SELL", "instrument_name": "BTCUSD-PERP" }, { "account_id": "9c72d8f1-583d-4b9d-b27c-55e695a2d116", "event_date": "2021-02-18", "journal_type": "SESSION_SETTLE", "journal_id": "186959", "transaction_qty": "0", "transaction_cost": "0.000000", "realized_pnl": "-0.007800", "trade_match_id": "0", "event_timestamp_ms": 1613638800001, "event_timestamp_ns": "1613638800001124563", "client_oid": "", "taker_side": "", "instrument_name": "BTCUSD-PERP" }]}`
	getOtcUser              = `{ "account_uuid": "00000000-00000000-00000000-00000000", "requests_per_minute": 30, "max_trade_value_usd": "5000000", "min_trade_value_usd": "50000", "accept_otc_tc_datetime": 1636512069509 }`
	requestOTCQuote         = `{"quote_id": "2412548678404715041", "quote_status": "ACTIVE", "quote_direction": "BUY", "base_currency": "BTC", "quote_currency": "USDT", "base_currency_size": null, "quote_currency_size": "100000.00", "quote_buy": "39708.24", "quote_buy_quantity": "2.51836898", "quote_buy_value": "100000.00", "quote_sell": "39677.18", "quote_sell_quantity": "2.52034040", "quote_sell_value": "100000.00", "quote_duration": 2, "quote_time": 1649736353489, "quote_expiry_time": 1649736363578 }`
	acceptOTCQuote          = `{"quote_id": "2412548678404715041", "quote_status": "FILLED", "quote_direction": "BUY", "base_currency": "BTC", "quote_currency": "USDT", "base_currency_size": null, "quote_currency_size": "100000.00", "quote_buy": "39708.24", "quote_sell": null, "quote_duration": 2, "quote_time": 1649743710146, "quote_expiry_time": 1649743720231, "trade_direction": "BUY", "trade_price": "39708.24", "trade_quantity": "2.51836898", "trade_value": "100000.00", "trade_time": 1649743718963 }`
	getOTCQuoteHistory      = `{"count": 1, "quote_list": [{ "quote_id": "2412795526826582752", "quote_status": "EXPIRED", "quote_direction": "BUY", "base_currency": "BTC", "quote_currency": "USDT", "base_currency_size": null, "quote_currency_size": "100000.00", "quote_buy": "39708.24", "quote_sell": null, "quote_duration": 2, "quote_time": 1649743710146, "quote_expiry_time": 1649743720231, "trade_direction": null, "trade_price": null, "trade_quantity": null, "trade_value": null, "trade_time": null } ] }`
	getOTCTradeHistory      = `{"count": 1, "trade_list": [{ "quote_id": "2412795526826582752", "quote_status": "FILLED", "quote_direction": "BUY", "base_currency": "BTC", "quote_currency": "USDT", "base_currency_size": null, "quote_currency_size": "100000.00", "quote_buy": "39708.24", "quote_sell": null, "quote_duration": 10, "quote_time": 1649743710146, "quote_expiry_time": 1649743720231, "trade_direction": "BUY", "trade_price": "39708.24", "trade_quantity": "2.51836898", "trade_value": "100000.00", "trade_time": 1649743718963 } ] }`
)

func TestPushData(t *testing.T) {
	t.Parallel()
	for x := range pushDataMap {
		err := cr.WsHandleData([]byte(pushDataMap[x]), true)
		assert.NoErrorf(t, err, "Received unexpected error: %v for asset type: %s", err, x)
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := cr.UpdateOrderExecutionLimits(context.Background(), asset.Binary)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	err = cr.UpdateOrderExecutionLimits(context.Background(), asset.Spot)
	assert.NoError(t, err)

	pairs, err := cr.FetchTradablePairs(context.Background(), asset.Spot)
	assert.NoError(t, err)
	assert.NotEmpty(t, pairs)

	for y := range pairs {
		lim, err := cr.GetOrderExecutionLimits(asset.Spot, pairs[y])
		assert.NoErrorf(t, err, "%v %s %v", err, pairs[y], asset.Spot)
		assert.NotEmpty(t, lim, "limit cannot be empty")
	}
}

func TestSafeNumberUnmarshal(t *testing.T) {
	t.Parallel()
	result := []byte(`{"value": null}`)
	resp := &struct {
		Value SafeNumber `json:"value"`
	}{}
	err := json.Unmarshal(result, &resp)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateStaking(t *testing.T) {
	t.Parallel()
	_, err := cr.CreateStaking(context.Background(), "", 123.45)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = cr.CreateStaking(context.Background(), tradablePair.String(), 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr, canManipulateRealOrders)
	result, err := cr.CreateStaking(context.Background(), tradablePair.String(), 123.45)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUnstake(t *testing.T) {
	t.Parallel()
	_, err := cr.Unstake(context.Background(), "", 123.45)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = cr.Unstake(context.Background(), tradablePair.String(), 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr, canManipulateRealOrders)
	result, err := cr.Unstake(context.Background(), tradablePair.String(), 123.45)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetStakingPosition(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.GetStakingPosition(context.Background(), tradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetStakingInstruments(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.GetStakingInstruments(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenStakeUnStakeRequests(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.GetOpenStakeUnStakeRequests(context.Background(), tradablePair.String(), time.Now().Add(-time.Hour*25*30), time.Now(), 10)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetStakingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.GetStakingHistory(context.Background(), tradablePair.String(), time.Now().Add(-time.Hour*25*30), time.Now(), 10)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetStakingReqardHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.GetStakingRewardHistory(context.Background(), tradablePair.String(), time.Now().Add(-time.Hour*25*30), time.Now(), 10)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestConvertStakedToken(t *testing.T) {
	t.Parallel()
	_, err := cr.ConvertStakedToken(context.Background(), "", "ETH_USDT", .5, 12.34, 3)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = cr.ConvertStakedToken(context.Background(), tradablePair.String(), "", .5, 12.34, 3)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = cr.ConvertStakedToken(context.Background(), tradablePair.String(), "ETH_USDT", 0, 12.34, 3)
	require.ErrorIs(t, err, errInvalidRate)
	_, err = cr.ConvertStakedToken(context.Background(), tradablePair.String(), "ETH_USDT", .5, 0, 3)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = cr.ConvertStakedToken(context.Background(), tradablePair.String(), "ETH_USDT", .5, 12.34, 0)
	require.ErrorIs(t, err, errInvalidSlippageToleraceBPs)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr, canManipulateRealOrders)
	result, err := cr.ConvertStakedToken(context.Background(), tradablePair.String(), "ETH_USDT", .5, 12.34, 3)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenStakingConverts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.GetOpenStakingConverts(context.Background(), time.Time{}, time.Time{}, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetStakingConvertHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.GetStakingConvertHistory(context.Background(), time.Time{}, time.Time{}, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestStakingConversionRate(t *testing.T) {
	t.Parallel()
	_, err := cr.StakingConversionRate(context.Background(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, cr)
	result, err := cr.StakingConversionRate(context.Background(), tradablePair.String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTimeInForceString(t *testing.T) {
	t.Parallel()
	var timeInForceStringMap = map[order.TimeInForce]struct {
		String string
		Error  error
	}{
		order.GoodTillDay | order.PostOnly: {"GOOD_TILL_CANCEL", nil},
		order.GoodTillCancel:               {"GOOD_TILL_CANCEL", nil},
		order.ImmediateOrCancel:            {"IMMEDIATE_OR_CANCEL", nil},
		order.FillOrKill:                   {"FILL_OR_KILL", nil},
		order.GoodTillDay:                  {"", order.ErrInvalidTimeInForce},
	}
	for k, v := range timeInForceStringMap {
		result, err := timeInForceString(k)
		assert.ErrorIs(t, err, v.Error)
		assert.Equal(t, result, v.String)
	}
}

func TestOrderTypeString(t *testing.T) {
	t.Parallel()
	var orderTypeStringMap = map[order.Type]string{
		order.Market:          "MARKET",
		order.Limit:           "LIMIT",
		order.StopLoss:        "STOP_LOSS",
		order.StopLimit:       "STOP_LIMIT",
		order.TakeProfit:      "TAKE_PROFIT",
		order.TakeProfitLimit: "TAKE_PROFIT_LIMI",
		order.OCO:             "",
	}
	for k, v := range orderTypeStringMap {
		oTypeString := OrderTypeToString(k)
		assert.Equal(t, oTypeString, v)
	}
}

func TestPriceTypeToString(t *testing.T) {
	t.Parallel()
	var priceTypeToStringMap = map[order.PriceType]struct {
		String string
		Error  error
	}{
		order.IndexPrice:     {"INDEX_PRICE", nil},
		order.MarkPrice:      {"MARK_PRICE", nil},
		order.LastPrice:      {"LAST_PRICE", nil},
		order.UnsetPriceType: {"", nil},
		order.PriceType(200): {"", order.ErrUnknownPriceType},
	}
	for k, v := range priceTypeToStringMap {
		priceTypeString, err := priceTypeToString(k)
		assert.ErrorIs(t, err, v.Error)
		assert.Equal(t, priceTypeString, v.String)
	}
}
