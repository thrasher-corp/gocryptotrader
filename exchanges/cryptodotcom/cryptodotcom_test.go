package cryptodotcom

import (
	"context"
	"errors"
	"log"
	"os"
	"strconv"
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
	e                   *Exchange
	mainTP, perpetualTP currency.Pair
)

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatal(err)
	}

	if apiKey != "" && apiSecret != "" {
		e.API.AuthenticatedSupport = true
		e.API.AuthenticatedWebsocketSupport = true
		e.API.CredentialsValidator.RequiresBase64DecodeSecret = false
		e.SetCredentials(apiKey, apiSecret, "", "", "", "")
		e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}
	err := initmainTP()
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func initmainTP() error {
	err := e.UpdateTradablePairs(context.Background(), false)
	if err != nil {
		return err
	}
	enabledPairs, err := e.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	} else if len(enabledPairs) == 0 {
		return errors.New("No enabled pairs found")
	}
	mainTP = enabledPairs[0]

	enabledPairs, err = e.GetEnabledPairs(asset.PerpetualSwap)
	if err != nil {
		return err
	} else if len(enabledPairs) == 0 {
		return errors.New("No enabled pairs found")
	}
	perpetualTP = enabledPairs[0]
	return nil
}

func TestGetRiskParameters(t *testing.T) {
	t.Parallel()
	result, err := e.GetRiskParameters(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	result, err := e.GetInstruments(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	result, err := e.GetOrderbook(t.Context(), mainTP.String(), 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCandlestickDetail(t *testing.T) {
	t.Parallel()
	result, err := e.GetCandlestickDetail(t.Context(), mainTP.String(), kline.FiveMin)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	result, err := e.GetTickers(t.Context(), mainTP.String())
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = e.GetTickers(t.Context(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetTrades(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetTrades(t.Context(), mainTP.String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetValuations(t *testing.T) {
	t.Parallel()
	_, err := e.GetValuations(t.Context(), "", "index_price", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.GetValuations(t.Context(), mainTP.String(), "", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errValuationTypeUnset)

	result, err := e.GetValuations(t.Context(), "BTCUSD-INDEX", "index_price", 0, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawFunds(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFunds(t.Context(), currency.EMPTYCODE, 10, core.BitcoinDonationAddress, "", "", "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.WithdrawFunds(t.Context(), currency.BTC, 0, core.BitcoinDonationAddress, "", "", "")
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = e.WithdrawFunds(t.Context(), currency.BTC, 10, "", "", "", "")
	require.ErrorIs(t, err, errAddressRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WithdrawFunds(t.Context(), currency.BTC, 10, core.BitcoinDonationAddress, "", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsCreateWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := e.WsCreateWithdrawal(currency.EMPTYCODE, 10, core.BitcoinDonationAddress, "", "", "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.WsCreateWithdrawal(currency.BTC, 0, core.BitcoinDonationAddress, "", "", "")
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = e.WsCreateWithdrawal(currency.BTC, 10, "", "", "", "")
	require.ErrorIs(t, err, errAddressRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WsCreateWithdrawal(currency.BTC, 10, core.BitcoinDonationAddress, "", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrencyNetworks(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCurrencyNetworks(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalHistory(t *testing.T) {
	t.Parallel()
	var resp *WithdrawalResponse
	err := json.Unmarshal([]byte(getWithdrawalHistory), &resp)
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetWithdrawalHistory(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsRetriveWithdrawalHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WsRetriveWithdrawalHistory()
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetDepositHistory(t.Context(), currency.EMPTYCODE, time.Time{}, time.Time{}, 20, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPersonalDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetPersonalDepositAddress(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPersonalDepositAddress(t.Context(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateExportRequest(t *testing.T) {
	t.Parallel()
	_, err := e.CreateExportRequest(t.Context(), mainTP.String(), "", time.Now().Add(-time.Hour*240), time.Now(), []string{})
	require.ErrorIs(t, err, errRequestedDataTypesRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateExportRequest(t.Context(), mainTP.String(), "", time.Now().Add(-time.Hour*240), time.Now(), []string{"SPOT_ORDER", "SPOT_TRADE"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetExportRequests(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetExportRequests(t.Context(), mainTP.String(), time.Time{}, time.Time{}, []string{"SPOT_ORDER"}, 10, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountSummary(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccountSummary(t.Context(), currency.USDT)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsRetriveAccountSummary(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WsRetriveAccountSummary(currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CreateOrder(t.Context(), &CreateOrderParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &CreateOrderParam{
		PostOnly: true,
	}
	_, err = e.CreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	arg.Symbol = "BTC_USDT"
	_, err = e.CreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Buy
	_, err = e.CreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = order.StopLimit
	_, err = e.CreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	arg.Price = 123
	_, err = e.CreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.Quantity = 1
	_, err = e.CreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, errTriggerPriceRequired)

	arg.OrderType = order.Market
	arg.Quantity = 0
	arg.Side = order.Buy
	_, err = e.CreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountMustBeSet)

	arg.Side = order.Sell
	_, err = e.CreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.OrderType = order.Stop
	_, err = e.CreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.Side = order.Sell
	_, err = e.CreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.Side = order.Buy
	arg.Notional = 1
	_, err = e.CreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, errTriggerPriceRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateOrder(t.Context(), &CreateOrderParam{
		Symbol:    mainTP.String(),
		Side:      order.Buy,
		OrderType: order.Limit,
		Price:     123,
		Quantity:  12,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsPlaceOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	arg := &CreateOrderParam{Symbol: mainTP.String(), Side: order.Buy, OrderType: order.Limit, Price: 123, Quantity: 12}
	result, err := e.WsPlaceOrder(arg)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()
	err := e.CancelExistingOrder(t.Context(), "", "1232412")
	assert.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	err = e.CancelExistingOrder(t.Context(), mainTP.String(), "")
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.CancelExistingOrder(t.Context(), mainTP.String(), "1232412")
	assert.NoError(t, err)
}

func TestWsCancelExistingOrder(t *testing.T) {
	t.Parallel()
	err := e.WsCancelExistingOrder("", "1232412")
	assert.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	err = e.WsCancelExistingOrder(mainTP.String(), "")
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.WsCancelExistingOrder(mainTP.String(), "1232412")
	assert.NoError(t, err)
}

func TestGetPrivateTrades(t *testing.T) {
	t.Parallel()
	var resp *PersonalTrades
	err := json.Unmarshal([]byte(getPrivateTrades), &resp)
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPrivateTrades(t.Context(), "", time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsRetrivePrivateTrades(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WsRetrivePrivateTrades("", time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderDetail(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOrderDetail(t.Context(), "1234")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsRetriveOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := e.WsRetriveOrderDetail("")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WsRetriveOrderDetail("1234")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPersonalOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPersonalOpenOrders(t.Context(), "", 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsRetrivePersonalOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WsRetrivePersonalOpenOrders("", 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPersonalOrderHistory(t *testing.T) {
	t.Parallel()
	var resp *PersonalOrdersResponse
	err := json.Unmarshal([]byte(getPersonalOrderHistory), &resp)
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPersonalOrderHistory(t.Context(), "", time.Time{}, time.Time{}, 0, 20)
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = e.WsRetrivePersonalOrderHistory("", time.Time{}, time.Time{}, 0, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateOrderList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateOrderList(t.Context(), "LIST", []CreateOrderParam{
		{
			Symbol: mainTP.String(), ClientOrderID: "", TimeInForce: "", Side: order.Buy, OrderType: order.Limit, PostOnly: false, TriggerPrice: 0, Price: 123, Quantity: 12, Notional: 0,
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsCreateOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.WsCreateOrderList("LIST", []CreateOrderParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WsCreateOrderList("LIST", []CreateOrderParam{
		{
			Symbol: mainTP.String(), ClientOrderID: "", TimeInForce: "", Side: order.Buy, OrderType: order.Limit, PostOnly: false, TriggerPrice: 0, Price: 123, Quantity: 12, Notional: 0,
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.CancelOrderList(t.Context(), []CancelOrderParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.CancelOrderList(t.Context(), []CancelOrderParam{{InstrumentName: "", OrderID: ""}})
	require.ErrorIs(t, err, errInstrumentNameOrOrderIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelOrderList(t.Context(), []CancelOrderParam{{InstrumentName: mainTP.String(), OrderID: "1234567"}, {InstrumentName: mainTP.String(), OrderID: "123450067"}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsCancelOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.WsCancelOrderList([]CancelOrderParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.WsCancelOrderList([]CancelOrderParam{{InstrumentName: "", OrderID: ""}})
	require.ErrorIs(t, err, errInstrumentNameOrOrderIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WsCancelOrderList([]CancelOrderParam{{InstrumentName: mainTP.String(), OrderID: "1234567"}, {InstrumentName: mainTP.String(), OrderID: "123450067"}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllPersonalOrders(t *testing.T) {
	t.Parallel()
	err := e.CancelAllPersonalOrders(t.Context(), "")
	assert.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.CancelAllPersonalOrders(t.Context(), mainTP.String())
	assert.NoError(t, err)
}

func TestWsCancelAllPersonalOrders(t *testing.T) {
	t.Parallel()
	err := e.WsCancelAllPersonalOrders("")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.WsCancelAllPersonalOrders(mainTP.String())
	assert.NoError(t, err)
}

func TestGetAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccounts(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubAccountTransfer(t *testing.T) {
	t.Parallel()
	err := e.SubAccountTransfer(t.Context(), "", "12345678-0000-0000-0000-000000000002", currency.BTC, 0.0000001)
	assert.ErrorIs(t, err, errSubAccountAddressRequired)
	err = e.SubAccountTransfer(t.Context(), "12345678-0000-0000-0000-000000000001", "", currency.BTC, 0.0000001)
	assert.ErrorIs(t, err, errSubAccountAddressRequired)
	err = e.SubAccountTransfer(t.Context(), "12345678-0000-0000-0000-000000000001", "12345678-0000-0000-0000-000000000002", currency.EMPTYCODE, 0.0000001)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	err = e.SubAccountTransfer(t.Context(), "12345678-0000-0000-0000-000000000001", "12345678-0000-0000-0000-000000000002", currency.BTC, 0)
	assert.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.SubAccountTransfer(t.Context(), "12345678-0000-0000-0000-000000000001", "12345678-0000-0000-0000-000000000002", currency.BTC, 0.0000001)
	assert.NoError(t, err)
}

func TestGetTransactions(t *testing.T) {
	t.Parallel()
	var resp *TransactionResponse
	err := json.Unmarshal([]byte(getTransactions), &resp)
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetTransactions(t.Context(), perpetualTP.String(), "", time.Time{}, time.Time{}, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserAccountFeeRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserAccountFeeRate(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFeeRateForUserAccount(t *testing.T) {
	t.Parallel()
	_, err := e.GetInstrumentFeeRate(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetInstrumentFeeRate(t.Context(), mainTP.String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateSubAccountTransfer(t *testing.T) {
	t.Parallel()
	err := e.CreateSubAccountTransfer(t.Context(), "", core.BitcoinDonationAddress, currency.USDT, 1232)
	assert.ErrorIs(t, err, errSubAccountAddressRequired)
	err = e.CreateSubAccountTransfer(t.Context(), "destination_address", "", currency.USDT, 1232)
	assert.ErrorIs(t, err, errSubAccountAddressRequired)
	err = e.CreateSubAccountTransfer(t.Context(), "destination_address", core.BitcoinDonationAddress, currency.EMPTYCODE, 1232)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	err = e.CreateSubAccountTransfer(t.Context(), "destination_address", core.BitcoinDonationAddress, currency.USDT, 0)
	assert.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.CreateSubAccountTransfer(t.Context(), "destination_address", core.BitcoinDonationAddress, currency.USDT, 1232)
	assert.NoError(t, err)
}

func TestGetOTCUser(t *testing.T) {
	t.Parallel()
	var resp *OTCTrade
	err := json.Unmarshal([]byte(getOtcUser), &resp)
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOTCUser(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOTCInstruments(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOTCInstruments(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRequestOTCQuote(t *testing.T) {
	t.Parallel()
	var resp *OTCQuoteResponse
	err := json.Unmarshal([]byte(requestOTCQuote), &resp)
	require.NoError(t, err)

	_, err = e.RequestOTCQuote(t.Context(), currency.EMPTYPAIR, .001, 232, "BUY")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.RequestOTCQuote(t.Context(), currency.NewPair(currency.BTC, currency.USDT), 0, 0, "BUY")
	require.ErrorIs(t, err, order.ErrAmountMustBeSet)
	_, err = e.RequestOTCQuote(t.Context(), currency.NewPair(currency.BTC, currency.USDT), .001, 232, "")
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.RequestOTCQuote(t.Context(), currency.NewPair(currency.BTC, currency.USDT), .001, 232, "BUY")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAcceptOTCQuote(t *testing.T) {
	t.Parallel()
	var resp *AcceptQuoteResponse
	err := json.Unmarshal([]byte(acceptOTCQuote), &resp)
	require.NoError(t, err)

	_, err = e.AcceptOTCQuote(t.Context(), "", "")
	require.ErrorIs(t, err, errQuoteIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.AcceptOTCQuote(t.Context(), "12323123", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOTCQuoteHistory(t *testing.T) {
	t.Parallel()
	var resp *QuoteHistoryResponse
	err := json.Unmarshal([]byte(getOTCQuoteHistory), &resp)
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOTCQuoteHistory(t.Context(), currency.EMPTYPAIR, time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOTCTradeHistory(t *testing.T) {
	t.Parallel()
	var resp *OTCTradeHistoryResponse
	err := json.Unmarshal([]byte(getOTCTradeHistory), &resp)
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOTCTradeHistory(t.Context(), currency.NewPair(currency.BTC, currency.USDT), time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateOTCOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CreateOTCOrder(t.Context(), "", "BUY", "3427401068340147456", 0.0001, 12321, false)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.CreateOTCOrder(t.Context(), mainTP.String(), "BUY", "3427401068340147456", 0, 12321, false)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = e.CreateOTCOrder(t.Context(), mainTP.String(), "BUY", "3427401068340147456", 0.0001, 0, false)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)
	_, err = e.CreateOTCOrder(t.Context(), mainTP.String(), "", "3427401068340147456", 0.0001, 12321, false)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateOTCOrder(t.Context(), mainTP.String(), "BUY", "3427401068340147456", 0.0001, 12321, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// wrapper test functions

func TestFetchmainTPs(t *testing.T) {
	t.Parallel()
	assetTypes := e.GetAssetTypes(true)
	for a := range assetTypes {
		result, err := e.FetchTradablePairs(t.Context(), assetTypes[a])
		require.NoError(t, err)
		assert.NotNil(t, result)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	result, err := e.UpdateTicker(t.Context(), mainTP, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.UpdateTicker(t.Context(), perpetualTP, asset.PerpetualSwap)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := e.UpdateTickers(t.Context(), asset.Spot)
	assert.NoError(t, err)

	err = e.UpdateTickers(t.Context(), asset.PerpetualSwap)
	assert.NoError(t, err)
}

func TestFetchTicker(t *testing.T) {
	t.Parallel()
	result, err := e.FetchTicker(t.Context(), perpetualTP, asset.PerpetualSwap)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFetchOrderbook(t *testing.T) {
	t.Parallel()
	result, err := e.FetchOrderbook(t.Context(), mainTP, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.FetchOrderbook(t.Context(), perpetualTP, asset.PerpetualSwap)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	result, err := e.UpdateOrderbook(t.Context(), mainTP, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.UpdateOrderbook(t.Context(), perpetualTP, asset.PerpetualSwap)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.UpdateAccountInfo(t.Context(), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.UpdateAccountInfo(t.Context(), asset.PerpetualSwap)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.PerpetualSwap)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	result, err := e.GetRecentTrades(t.Context(), currency.NewPair(currency.BTC, currency.USDT), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetRecentTrades(t.Context(), currency.NewPair(currency.BTC, currency.USDT), asset.PerpetualSwap)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	result, err := e.GetHistoricTrades(t.Context(), mainTP, asset.Spot, time.Now().Add(-time.Hour*4), time.Now())
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetHistoricTrades(t.Context(), perpetualTP, asset.PerpetualSwap, time.Now().Add(-time.Hour*4), time.Now())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccountFundingHistory(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	startTime := time.Now().Add(-time.Minute * 40)
	endTime := time.Now()

	result, err := e.GetHistoricCandles(t.Context(), mainTP, asset.Spot, kline.OneDay, startTime, endTime)
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = e.GetHistoricCandles(t.Context(), mainTP, asset.Spot, kline.FiveMin, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetHistoricCandles(t.Context(), perpetualTP, asset.PerpetualSwap, kline.FiveMin, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{Type: order.Limit, Pairs: currency.Pairs{mainTP, currency.NewPair(currency.USDT, currency.USD), currency.NewPair(currency.USD, currency.LTC)}, AssetType: asset.Spot, Side: order.Buy})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.Buy,
	}
	result, err := e.GetOrderHistory(t.Context(), &getOrdersRequest)
	require.NoError(t, err)
	require.NotNil(t, result)

	getOrdersRequest.Pairs = []currency.Pair{mainTP}
	result, err = e.GetOrderHistory(t.Context(), &getOrdersRequest)
	require.NoError(t, err)
	assert.NotNil(t, result)

	getOrdersRequest.Pairs = []currency.Pair{perpetualTP}
	getOrdersRequest.AssetType = asset.PerpetualSwap
	result, err = e.GetOrderHistory(t.Context(), &getOrdersRequest)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	orderSubmission := &order.Submit{
		Pair: currency.Pair{
			Base:  currency.LTC,
			Quote: currency.BTC,
		},
		Exchange:  e.Name,
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1000000000,
		ClientID:  "myOwnOrder",
		AssetType: asset.Spot,
	}
	result, err := e.SubmitOrder(t.Context(), orderSubmission)
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = e.SubmitOrder(t.Context(), &order.Submit{
		Pair:      mainTP,
		Exchange:  e.Name,
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1000000000,
		ClientID:  "myOwnOrder",
		AssetType: asset.Spot,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.SubmitOrder(t.Context(), &order.Submit{
		Pair:      perpetualTP,
		Exchange:  e.Name,
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1000000000,
		ClientID:  "myOwnOrder",
		AssetType: asset.PerpetualSwap,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err := e.CancelOrder(t.Context(), &order.Cancel{
		OrderID:   "1",
		Pair:      mainTP,
		AssetType: asset.Spot,
	})
	assert.NoError(t, err)

	err = e.CancelOrder(t.Context(), &order.Cancel{
		OrderID:   "1",
		Pair:      perpetualTP,
		AssetType: asset.PerpetualSwap,
	})
	assert.NoError(t, err)
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelBatchOrders(t.Context(), []order.Cancel{
		{
			OrderID: "1",
			Pair:    mainTP,
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelAllOrders(t.Context(), &order.Cancel{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOrderInfo(t.Context(), "123", mainTP, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetOrderInfo(t.Context(), "123", perpetualTP, asset.PerpetualSwap)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetDepositAddress(t.Context(), currency.ETH, "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
		Amount:   10,
		Currency: currency.BTC,
		Crypto: withdraw.CryptoRequest{
			Chain:      currency.BTC.String(),
			Address:    core.BitcoinDonationAddress,
			AddressTag: "",
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGenerateDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	result, err := e.GenerateDefaultSubscriptions()
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsRetriveCancelOnDisconnect(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.WsRetriveCancelOnDisconnect()
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsSetCancelOnDisconnect(t *testing.T) {
	t.Parallel()
	_, err := e.WsSetCancelOnDisconnect("")
	require.ErrorIs(t, err, errInvalidOrderCancellationScope)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WsSetCancelOnDisconnect("ACCOUNT")
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
	arg.Symbol = mainTP.String()
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
	result, err := e.GetFeeByType(t.Context(), feeBuilder)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	if !sharedtestvalues.AreAPICredentialsSet(e) {
		assert.Equal(t, exchange.OfflineTradeFee, feeBuilder.FeeType)
	} else {
		assert.Equal(t, exchange.CryptocurrencyTradeFee, feeBuilder.FeeType)
	}
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
	pushDataMap := map[string]string{
		"Orderbook":     `{ "id": -1, "code": 0, "method": "subscribe", "result": { "channel": "book", "subscription": "book.RSR_USDT", "instrument_name": "RSR_USDT", "depth": 150, "data": [ { "asks": [ [ "0.0041045", "164840", "1" ], [ "0.0041057", "273330", "1" ], [ "0.0041116", "6440", "1" ], [ "0.0041159", "29490", "1" ], [ "0.0041185", "21940", "1" ], [ "0.0041238", "191790", "2" ], [ "0.0041317", "495840", "2" ], [ "0.0041396", "1117990", "1" ], [ "0.0041475", "1430830", "1" ], [ "0.0041528", "785220", "1" ], [ "0.0041554", "1409330", "1" ], [ "0.0041633", "1710820", "1" ], [ "0.0041712", "2399680", "1" ], [ "0.0041791", "2355400", "1" ], [ "0.0042500", "1500", "1" ], [ "0.0044000", "1000", "1" ], [ "0.0045000", "1000", "1" ], [ "0.0046600", "85770", "1" ], [ "0.0049230", "20660", "1" ], [ "0.0049380", "88520", "2" ], [ "0.0050000", "1120", "1" ], [ "0.0050203", "304960", "2" ], [ "0.0051026", "509200", "2" ], [ "0.0051849", "3452290", "1" ], [ "0.0052672", "10928750", "1" ], [ "0.0206000", "730", "1" ], [ "0.0406000", "370", "1" ] ], "bids": [ [ "0.0041013", "273330", "1" ], [ "0.0040975", "3750", "1" ], [ "0.0040974", "174120", "1" ], [ "0.0040934", "6440", "1" ], [ "0.0040922", "32200", "1" ], [ "0.0040862", "21940", "1" ], [ "0.0040843", "187900", "2" ], [ "0.0040764", "483650", "3" ], [ "0.0040686", "12280", "1" ], [ "0.0040685", "813180", "3" ], [ "0.0040607", "16020", "1" ], [ "0.0040606", "1123210", "3" ], [ "0.0040527", "1432240", "3" ], [ "0.0040482", "642210", "1" ], [ "0.0040448", "1441580", "2" ], [ "0.0040369", "2071370", "2" ], [ "0.0040290", "1453600", "1" ], [ "0.0037500", "29390", "1" ], [ "0.0033776", "80", "1" ], [ "0.0033740", "29630", "1" ], [ "0.0033000", "50", "1" ], [ "0.0032797", "30990", "1" ], [ "0.0032097", "175720", "2" ], [ "0.0032000", "50", "1" ], [ "0.0031274", "511460", "2" ], [ "0.0031000", "50", "1" ], [ "0.0030451", "793150", "2" ], [ "0.0030400", "750000", "1" ], [ "0.0030000", "100", "1" ], [ "0.0029628", "5620050", "2" ], [ "0.0029000", "50", "1" ], [ "0.0028805", "20567780", "2" ], [ "0.0018000", "500", "1" ], [ "0.0014500", "500", "1" ] ], "t": 1679082891435, "tt": 1679082890266, "u": 27043535761920, "cs": 723295208 } ] } }`,
		"Ticker":        `{ "id": -1, "code": 0, "method": "subscribe", "result": { "channel": "ticker", "instrument_name": "RSR_USDT", "subscription": "ticker.RSR_USDT", "id": -1, "data": [ { "h": "0.0041622", "l": "0.0037959", "a": "0.0040738", "c": "0.0721", "b": "0.0040738", "bs": "3680", "k": "0.0040796", "ks": "179780", "i": "RSR_USDT", "v": "45133400", "vv": "181223.95", "oi": "0","t": 1679087156318}]}}`,
		"Trade":         `{"id": 140466243, "code": 0, "method": "subscribe", "result": { "channel": "trade", "subscription": "trade.RSR_USDT", "instrument_name": "RSR_USDT", "data": [ { "d": "4611686018428182866", "t": 1679085786004, "p": "0.0040604", "q": "10", "s": "BUY", "i": "RSR_USDT" }, { "d": "4611686018428182865", "t": 1679085717204, "p": "0.0040671", "q": "10", "s": "BUY", "i": "RSR_USDT" }, { "d": "4611686018428182864", "t": 1679085672504, "p": "0.0040664", "q": "10", "s": "BUY", "i": "RSR_USDT" }, { "d": "4611686018428182863", "t": 1679085638806, "p": "0.0040674", "q": "10", "s": "BUY", "i": "RSR_USDT" }, { "d": "4611686018428182862", "t": 1679085568762, "p": "0.0040689", "q": "20", "s": "BUY", "i": "RSR_USDT" } ] } }`,
		"Candlestick":   `{"id": -1, "code": 0, "method": "subscribe", "result": { "channel": "candlestick", "instrument_name": "RSR_USDT", "subscription": "candlestick.5m.RSR_USDT", "interval": "5m", "data": [ { "o": "0.0040838", "h": "0.0040920", "l": "0.0040838", "c": "0.0040920", "v": "60.0000", "t": 1679087700000, "ut": 1679087959106 } ] } }`,
		"User Balance":  `{"id":3397447550047468012,"method":"subscribe","code":0,"result":{"subscription":"user.balance","channel":"user.balance","data":[{"stake":0,"balance":7.26648846,"available":7.26648846,"currency":"BOSON","order":0},{"stake":0,"balance":15.2782122,"available":15.2782122,"currency":"EFI","order":0},{"stake":0,"balance":90.63857968,"available":90.63857968,"currency":"ZIL","order":0},{"stake":0,"balance":16790279.87929312,"available":16790279.87929312,"currency":"SHIB","order":0},{"stake":0,"balance":1.79673318,"available":1.79673318,"currency":"NEAR","order":0},{"stake":0,"balance":307.29679422,"available":307.29679422,"currency":"DOGE","order":0},{"stake":0,"balance":0.00109125,"available":0.00109125,"currency":"BTC","order":0},{"stake":0,"balance":18634.17320776,"available":18634.17320776,"currency":"CRO-STAKE","order":0},{"stake":0,"balance":0.4312475,"available":0.4312475,"currency":"DOT","order":0},{"stake":0,"balance":924.07197632,"available":924.07197632,"currency":"CRO","order":0}]}}`,
		"User Order":    `{"method": "subscribe", "result": { "instrument_name": "ETH_CRO", "subscription": "user.order.ETH_CRO", "channel": "user.order", "data": [ { "status": "ACTIVE", "side": "BUY", "price": 1, "quantity": 1, "order_id": "366455245775097673", "client_oid": "my_order_0002", "create_time": 1588758017375, "update_time": 1588758017411, "type": "LIMIT", "instrument_name": "ETH_CRO", "cumulative_quantity": 0, "cumulative_value": 0, "avg_price": 0, "fee_currency": "CRO", "time_in_force":"GOOD_TILL_CANCEL" } ], "channel": "user.order.ETH_CRO"}}`,
		"User Trade":    `{"method": "subscribe", "code": 0, "result": { "instrument_name": "ETH_CRO", "subscription": "user.trade.ETH_CRO", "channel": "user.trade", "data": [ { "side": "SELL", "instrument_name": "ETH_CRO", "fee": 0.014, "trade_id": "367107655537806900", "create_time": "1588777459755", "traded_price": 7, "traded_quantity": 1, "fee_currency": "CRO", "order_id": "367107623521528450" } ], "channel": "user.trade.ETH_CRO" }}`,
		"OTC Orderbook": `{ "id": 1, "code": 0, "method": "subscribe", "result": { "channel": "otc_book", "subscription": "otc_book.BTC_USDT", "instrument_name": "BTC_USDT", "t": 1667800910315, "data": [ { "asks": [ ["8944.4", "1", "1", 1672502400000, 1510419685596942874], ["8955.1", "3", "1", 1672502400000, 1510419685596942875] ], "bids": [ ["8940.5", "1", "1", 1672502400000, 1510419685596942876], ["8918.7", "3", "1", 1672502400000, 1510419685596942877]]}]}}`,
	}
	for key, val := range pushDataMap {
		t.Run(key, func(t *testing.T) {
			t.Parallel()
			err := e.WsHandleData([]byte(val), true)
			assert.NoErrorf(t, err, "Received unexpected error: %v for asset type: %s", err, val)
		})
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := e.UpdateOrderExecutionLimits(t.Context(), asset.Binary)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	err = e.UpdateOrderExecutionLimits(t.Context(), asset.Spot)
	assert.NoError(t, err)

	err = e.UpdateOrderExecutionLimits(t.Context(), asset.PerpetualSwap)
	assert.NoError(t, err)

	pairs, err := e.FetchTradablePairs(t.Context(), asset.Spot)
	assert.NoError(t, err)
	assert.NotEmpty(t, pairs)

	pairs, err = e.FetchTradablePairs(t.Context(), asset.PerpetualSwap)
	assert.NoError(t, err)
	assert.NotEmpty(t, pairs)

	for y := range pairs {
		lim, err := e.GetOrderExecutionLimits(asset.Spot, pairs[y])
		assert.NoErrorf(t, err, "%v %s %v", err, pairs[y], asset.Spot)
		assert.NotEmpty(t, lim, "limit cannot be empty")
	}

	for y := range pairs {
		lim, err := e.GetOrderExecutionLimits(asset.PerpetualSwap, pairs[y])
		assert.NoErrorf(t, err, "%v %s %v", err, pairs[y], asset.PerpetualSwap)
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
	_, err := e.CreateStaking(t.Context(), "", 123.45)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.CreateStaking(t.Context(), mainTP.String(), 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateStaking(t.Context(), mainTP.String(), 123.45)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUnstake(t *testing.T) {
	t.Parallel()
	_, err := e.Unstake(t.Context(), "", 123.45)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.Unstake(t.Context(), mainTP.String(), 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.Unstake(t.Context(), mainTP.String(), 123.45)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetStakingPosition(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetStakingPosition(t.Context(), mainTP.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetStakingInstruments(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetStakingInstruments(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenStakeUnStakeRequests(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOpenStakeUnStakeRequests(t.Context(), mainTP.String(), time.Now().Add(-time.Hour*25*30), time.Now(), 10)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetOpenStakeUnStakeRequests(t.Context(), perpetualTP.String(), time.Now().Add(-time.Hour*25*30), time.Now(), 10)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetStakingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetStakingHistory(t.Context(), mainTP.String(), time.Now().Add(-time.Hour*25*30), time.Now(), 10)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetStakingReqardHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetStakingRewardHistory(t.Context(), mainTP.String(), time.Now().Add(-time.Hour*25*30), time.Now(), 10)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestConvertStakedToken(t *testing.T) {
	t.Parallel()
	_, err := e.ConvertStakedToken(t.Context(), "", "ETH_USDT", .5, 12.34, 3)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.ConvertStakedToken(t.Context(), mainTP.String(), "", .5, 12.34, 3)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.ConvertStakedToken(t.Context(), mainTP.String(), "ETH_USDT", 0, 12.34, 3)
	require.ErrorIs(t, err, errInvalidRate)
	_, err = e.ConvertStakedToken(t.Context(), mainTP.String(), "ETH_USDT", .5, 0, 3)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = e.ConvertStakedToken(t.Context(), mainTP.String(), "ETH_USDT", .5, 12.34, 0)
	require.ErrorIs(t, err, errInvalidSlippageToleraceBPs)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ConvertStakedToken(t.Context(), mainTP.String(), "ETH_USDT", .5, 12.34, 3)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenStakingConverts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOpenStakingConverts(t.Context(), time.Time{}, time.Time{}, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetStakingConvertHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetStakingConvertHistory(t.Context(), time.Time{}, time.Time{}, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestStakingConversionRate(t *testing.T) {
	t.Parallel()
	_, err := e.StakingConversionRate(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.StakingConversionRate(t.Context(), mainTP.String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTimeInForceString(t *testing.T) {
	t.Parallel()
	timeInForceStringMap := map[order.TimeInForce]struct {
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
	orderTypeStringMap := map[order.Type]string{
		order.Market:          "MARKET",
		order.Limit:           "LIMIT",
		order.Stop:            "STOP_LOSS",
		order.StopLimit:       "STOP_LIMIT",
		order.TakeProfit:      "TAKE_PROFIT",
		order.TakeProfitLimit: "TAKE_PROFIT_LIMIT",
		order.OCO:             "OCO",
	}
	for k, v := range orderTypeStringMap {
		t.Run(v, func(t *testing.T) {
			t.Parallel()
			oTypeString := OrderTypeToString(k)
			assert.Equal(t, oTypeString, v)
		})
	}
}

func TestPriceTypeToString(t *testing.T) {
	t.Parallel()
	priceTypeToStringMap := map[order.PriceType]struct {
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

func TestGetUserBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserBalance(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserBalanceHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserBalanceHistory(t.Context(), "H1", time.Time{}, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountBalances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountBalances(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPositions(t.Context(), perpetualTP.String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetExpiredSettlementPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetExpiredSettlementPrice(t.Context(), asset.Empty, 0)
	require.ErrorIs(t, err, asset.ErrInvalidAsset)

	result, err := e.GetExpiredSettlementPrice(t.Context(), asset.Futures, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangeAccountLeverage(t *testing.T) {
	t.Parallel()
	err := e.ChangeAccountLeverage(t.Context(), "", 100)
	require.ErrorIs(t, err, errAccountIDMissing)

	err = e.ChangeAccountLeverage(t.Context(), perpetualTP.String(), 0)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	err = e.ChangeAccountLeverage(t.Context(), perpetualTP.String(), 100)
	assert.NoError(t, err)
}

func TestGetAllExecutableTradesForInstrument(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllExecutableTradesForInstrument(t.Context(), perpetualTP.String(), time.Time{}, time.Time{}, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestClosePosition(t *testing.T) {
	t.Parallel()
	_, err := e.ClosePosition(t.Context(), "", "MARKET", 23123)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.ClosePosition(t.Context(), perpetualTP.String(), "", 23123)
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	_, err = e.ClosePosition(t.Context(), perpetualTP.String(), "LIMIT", 0)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ClosePosition(t.Context(), perpetualTP.String(), "LIMIT", 23123)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrderList(t.Context(), "", "6498090546073120100", perpetualTP.String())
	require.ErrorIs(t, err, errContingencyTypeRequired)
	_, err = e.GetFuturesOrderList(t.Context(), "OCO", "", perpetualTP.String())
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.GetFuturesOrderList(t.Context(), "OCO", "6498090546073120100", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFuturesOrderList(t.Context(), "OCO", "6498090546073120100", perpetualTP.String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInsurance(t *testing.T) {
	t.Parallel()
	_, err := e.GetInsurance(t.Context(), "", 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetInsurance(t.Context(), perpetualTP.String(), 10, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTimeInForceFromString(t *testing.T) {
	t.Parallel()
	timesInForceList := []struct {
		String   string
		PostOnly bool
		TIF      order.TimeInForce
	}{
		{"GOOD_TILL_CANCEL", true, order.GoodTillCancel | order.PostOnly},
		{"GOOD_TILL_CANCEL", false, order.GoodTillCancel},
		{"IMMEDIATE_OR_CANCEL", false, order.ImmediateOrCancel},
		{"FILL_OR_KILL", false, order.FillOrKill},
		{"", true, order.PostOnly},
		{"", false, order.UnknownTIF},
	}
	for i, v := range timesInForceList {
		t.Run(v.String+"#"+strconv.FormatBool(v.PostOnly), func(t *testing.T) {
			t.Parallel()
			tif := timeInForceFromString(timesInForceList[i].String, timesInForceList[i].PostOnly)
			assert.Equal(t, timesInForceList[i].TIF, tif)
		})
	}
}

func TestStringToInterval(t *testing.T) {
	t.Parallel()
	intervalsList := []struct {
		String   string
		Interval kline.Interval
		Err      error
	}{
		{"5m", kline.FiveMin, nil},
		{"15m", kline.FifteenMin, nil},
		{"30m", kline.ThirtyMin, nil},
		{"1h", kline.OneHour, nil},
		{"4h", kline.FourHour, nil},
		{"6h", kline.SixHour, nil},
		{"12h", kline.TwelveHour, nil},
		{"abcd", 0, kline.ErrUnsupportedInterval},
		{"", 0, kline.ErrUnsupportedInterval},
	}
	for i := range intervalsList {
		t.Run(intervalsList[i].String, func(t *testing.T) {
			t.Parallel()
			result, err := stringToInterval(intervalsList[i].String)
			assert.ErrorIs(t, err, intervalsList[i].Err)
			assert.Equal(t, intervalsList[i].Interval, result)
		})
	}
}
