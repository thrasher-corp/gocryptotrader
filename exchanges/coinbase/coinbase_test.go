package coinbase

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your APIKeys here for better testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
	// Sandbox functionality only works for certain endpoints https://docs.cdp.coinbase.com/coinbase-app/advanced-trade-apis/sandbox
	testingInSandbox = false
)

var (
	e                = &Exchange{}
	testCrypto       = currency.BTC
	testFiat         = currency.USD
	testStable       = currency.USDC
	testWrappedAsset = currency.CBETH
	testPairFiat     = currency.NewPairWithDelimiter(testCrypto.String(), testFiat.String(), "-")
	testPairStable   = currency.NewPairWithDelimiter(testCrypto.String(), testStable.String(), "-")
)

// Constants used within tests
const (
	testAddress = "fake address"
	testAmount  = 1e-08
	testAmount2 = 1e-02
	testAmount3 = 1
	testPrice   = 1.5e+05

	skipPayMethodNotFound          = "no payment methods found, skipping"
	skipInsufSuitableAccs          = "insufficient suitable accounts for test, skipping"
	skipInsufficientFunds          = "insufficient funds for test, skipping"
	skipInsufficientOrders         = "insufficient orders for test, skipping"
	skipInsufficientPortfolios     = "insufficient portfolios for test, skipping"
	skipInsufficientWallets        = "insufficient wallets for test, skipping"
	skipInsufficientFundsOrWallets = "insufficient funds or wallets for test, skipping"
	skipInsufficientTransactions   = "insufficient transactions for test, skipping"

	errExpectMismatch          = "received: '%v' but expected: '%v'"
	errExpectedNonEmpty        = "expected non-empty response"
	errInvalidProductID        = `Coinbase unsuccessful HTTP status code: 404 raw response: {"error":"NOT_FOUND","error_details":"valid product_id is required","message":"valid product_id is required"}`
	errExpectedFeeRange        = "expected fee range of %v and %v, received %v"
	errOptionInvalid           = `Coinbase unsuccessful HTTP status code: 400 raw response: {"error":"unknown","error_details":"parsing field \"product_type\": \"OPTIONS\" is not a valid value","message":"parsing field \"product_type\": \"OPTIONS\" is not a valid value"}`
	errJSONUnmarshalUnexpected = "JSON umarshalling did not return expected error"
)

func TestMain(m *testing.M) {
	if err := exchangeBaseHelper(e); err != nil {
		log.Fatal(err)
	}
	if testingInSandbox {
		err := e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
			exchange.RestSpot: sandboxAPIURL,
		})
		if err != nil {
			log.Fatalf("Coinbase SetDefaultEndpoints sandbox error: %s", err)
		}
	}
	os.Exit(m.Run())
}

func TestSetup(t *testing.T) {
	cfg, err := e.GetStandardConfig()
	assert.NoError(t, err)
	exch := &Exchange{}
	err = exchangeBaseHelper(exch)
	require.NoError(t, err)
	cfg.ProxyAddress = string(rune(0x7f))
	err = exch.Setup(cfg)
	assert.ErrorIs(t, err, exchange.ErrSettingProxyAddress)
}

func TestWsConnect(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	exch := &Exchange{}
	exch.Websocket = sharedtestvalues.NewTestWebsocket()
	err := exch.WsConnect()
	assert.ErrorIs(t, err, websocket.ErrWebsocketNotEnabled)
	err = exchangeBaseHelper(exch)
	require.NoError(t, err)
	err = exch.Websocket.Enable(t.Context())
	assert.NoError(t, err)
}

func TestGetAccountByID(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountByID(t.Context(), "")
	assert.ErrorIs(t, err, errAccountIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	longResp, err := e.ListAccounts(t.Context(), 49, 0)
	require.NoError(t, err)
	require.True(t, longResp != nil && len(longResp.Accounts) > 0, errExpectedNonEmpty)
	shortResp, err := e.GetAccountByID(t.Context(), longResp.Accounts[0].UUID)
	require.NoError(t, err)
	assert.Equal(t, shortResp, longResp.Accounts[0])
}

func TestListAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.ListAccounts(t.Context(), 50, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestCreateConvertQuote(t *testing.T) {
	t.Parallel()
	_, err := e.CreateConvertQuote(t.Context(), "", "", "", "", 0)
	assert.ErrorIs(t, err, errAccountIDEmpty)
	_, err = e.CreateConvertQuote(t.Context(), "meow", "123", "", "", 0)
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	fromAccID, toAccID := convertTestHelper(t)
	resp, err := e.CreateConvertQuote(t.Context(), fromAccID, toAccID, "", "", 0.01)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestCommitConvertTrade(t *testing.T) {
	convertTestShared(t, e.CommitConvertTrade)
}

func TestGetConvertTradeByID(t *testing.T) {
	convertTestShared(t, e.GetConvertTradeByID)
}

func TestGetPermissions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetPermissions(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetTransactionSummary(t *testing.T) {
	t.Parallel()
	_, err := e.GetTransactionSummary(t.Context(), time.Unix(2, 2), time.Unix(1, 1), "", "", "")
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetTransactionSummary(t.Context(), time.Unix(1, 1), time.Now(), "UNKNOWN_VENUE_TYPE", asset.Spot.Upper(), "UNKNOWN_CONTRACT_EXPIRY_TYPE")
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestCancelPendingFuturesSweep(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelPendingFuturesSweep(t.Context())
	assert.NoError(t, err)
}

func TestGetCurrentMarginWindow(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrentMarginWindow(t.Context(), "")
	assert.ErrorIs(t, err, errMarginProfileTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetCurrentMarginWindow(t.Context(), "MARGIN_PROFILE_TYPE_RETAIL_REGULAR")
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetFuturesBalanceSummary(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetFuturesBalanceSummary(t.Context())
	assert.NoError(t, err)
}

func TestGetFuturesPositionByID(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPositionByID(t.Context(), currency.Pair{})
	assert.ErrorIs(t, err, errProductIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetFuturesPositionByID(t.Context(), currency.NewPairWithDelimiter("SLR-25NOV25", "CDE", "-"))
	assert.NoError(t, err)
}

func TestGetIntradayMarginSetting(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetIntradayMarginSetting(t.Context())
	assert.NoError(t, err)
}

func TestListFuturesPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.ListFuturesPositions(t.Context())
	assert.NoError(t, err)
}

func TestListFuturesSweeps(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.ListFuturesSweeps(t.Context())
	assert.NoError(t, err)
}

func TestScheduleFuturesSweep(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.ScheduleFuturesSweep(t.Context(), 0.001337)
	assert.NoError(t, err)
}

func TestSetIntradayMarginSetting(t *testing.T) {
	t.Parallel()
	err := e.SetIntradayMarginSetting(t.Context(), "")
	assert.ErrorIs(t, err, errSettingEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.SetIntradayMarginSetting(t.Context(), "INTRADAY_MARGIN_SETTING_STANDARD")
	assert.NoError(t, err)
}

func TestCancelOrders(t *testing.T) {
	t.Parallel()
	var orderSlice []string
	_, err := e.CancelOrders(t.Context(), orderSlice)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)
	orderSlice = make([]string, 200)
	for i := range 200 {
		orderSlice[i] = strconv.Itoa(i)
	}
	_, err = e.CancelOrders(t.Context(), orderSlice)
	assert.ErrorIs(t, err, errCancelLimitExceeded)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	orderSlice = []string{"1"}
	resp, err := e.CancelOrders(t.Context(), orderSlice)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestClosePosition(t *testing.T) {
	t.Parallel()
	_, err := e.ClosePosition(t.Context(), "", currency.Pair{}, 0)
	assert.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)
	_, err = e.ClosePosition(t.Context(), "meow", currency.Pair{}, 0)
	assert.ErrorIs(t, err, errProductIDEmpty)
	_, err = e.ClosePosition(t.Context(), "meow", testPairFiat, 0)
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	resp, err := e.ClosePosition(t.Context(), "1", currency.NewPairWithDelimiter("BIT", "31OCT25-CDE", "-"), testAmount)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestPlaceOrder(t *testing.T) {
	t.Parallel()
	ord := &PlaceOrderInfo{}
	_, err := e.PlaceOrder(t.Context(), ord)
	assert.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)
	ord.ClientOID = "meow"
	_, err = e.PlaceOrder(t.Context(), ord)
	assert.ErrorIs(t, err, errProductIDEmpty)
	ord.ProductID = testPairFiat.String()
	_, err = e.PlaceOrder(t.Context(), ord)
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid)
	ord.BaseAmount = testAmount
	_, err = e.PlaceOrder(t.Context(), ord)
	assert.ErrorIs(t, err, errInvalidOrderType)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	id, err := uuid.NewV4()
	assert.NoError(t, err)
	ord = &PlaceOrderInfo{
		ClientOID:  id.String(),
		ProductID:  testPairStable.String(),
		Side:       order.Buy.String(),
		MarginType: "CROSS",
		Leverage:   9999,
		OrderInfo: OrderInfo{
			PostOnly:   false,
			EndTime:    time.Now().Add(time.Hour),
			OrderType:  order.Limit,
			BaseAmount: testAmount,
			LimitPrice: testPrice,
		},
	}
	resp, err := e.PlaceOrder(t.Context(), ord)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, resp, errExpectedNonEmpty)
	}
	id, err = uuid.NewV4()
	assert.NoError(t, err)
	ord.ClientOID = id.String()
	ord.MarginType = "MULTI"
	resp, err = e.PlaceOrder(t.Context(), ord)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestEditOrder(t *testing.T) {
	t.Parallel()
	_, err := e.EditOrder(t.Context(), "", 0, 0)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.EditOrder(t.Context(), "meow", 0, 0)
	assert.ErrorIs(t, err, errSizeAndPriceZero)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	resp, err := e.EditOrder(t.Context(), "1", testAmount, testPrice-1)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestEditOrderPreview(t *testing.T) {
	t.Parallel()
	_, err := e.EditOrderPreview(t.Context(), "", 0, 0)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.EditOrderPreview(t.Context(), "meow", 0, 0)
	assert.ErrorIs(t, err, errSizeAndPriceZero)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.EditOrderPreview(t.Context(), "1", testAmount, testPrice+2)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetOrderByID(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderByID(t.Context(), "", "", currency.Code{})
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	ordID, err := e.ListOrders(t.Context(), &ListOrdersReq{Limit: 10})
	assert.NoError(t, err)
	if ordID == nil || len(ordID.Orders) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	resp, err := e.GetOrderByID(t.Context(), ordID.Orders[0].OrderID, ordID.Orders[0].ClientOID, testFiat)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestListFills(t *testing.T) {
	t.Parallel()
	_, err := e.ListFills(t.Context(), nil, nil, nil, 0, "", time.Unix(2, 2), time.Unix(1, 1), 0)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.ListFills(t.Context(), nil, nil, currency.Pairs{testPairFiat, testPairStable}, 0, "TRADE_TIME", time.Time{}, time.Time{}, 0)
	assert.NoError(t, err)
	_, err = e.ListFills(t.Context(), []string{"1", "2"}, nil, nil, 0, "", time.Time{}, time.Time{}, 0)
	assert.NoError(t, err)
	_, err = e.ListFills(t.Context(), nil, []string{"3", "4"}, nil, 0, "", time.Time{}, time.Time{}, 0)
	assert.NoError(t, err)
}

func TestListOrders(t *testing.T) {
	t.Parallel()
	_, err := e.ListOrders(t.Context(), &ListOrdersReq{
		StartDate: time.Unix(2, 2),
		EndDate:   time.Unix(1, 1),
	})
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	orderStatus := []string{"FILLED", "CANCELLED", "EXPIRED", "FAILED"}
	orderTypes := []string{"MARKET", "LIMIT", "STOP", "STOP_LIMIT", "BRACKET", "TWAP"}
	timeInForces := []string{"GOOD_UNTIL_DATE_TIME", "GOOD_UNTIL_CANCELLED", "IMMEDIATE_OR_CANCEL", "FILL_OR_KILL"}
	productIDs := currency.Pairs{testPairFiat, testPairStable}
	_, err = e.ListOrders(t.Context(), &ListOrdersReq{
		OrderStatus:          orderStatus,
		TimeInForces:         timeInForces,
		OrderTypes:           orderTypes,
		ProductIDs:           productIDs,
		OrderSide:            "BUY",
		OrderPlacementSource: "RETAIL_SIMPLE",
		ContractExpiryType:   "PERPETUAL",
		SortBy:               "LAST_FILL_TIME",
	})
	assert.NoError(t, err)
	resp, err := e.ListOrders(t.Context(), &ListOrdersReq{})
	assert.NoError(t, err)
	if resp == nil || len(resp.Orders) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	orderIDs := []string{resp.Orders[0].OrderID}
	_, err = e.ListOrders(t.Context(), &ListOrdersReq{
		OrderIDs: orderIDs,
	})
	assert.NoError(t, err)
}

func TestPreviewOrder(t *testing.T) {
	t.Parallel()
	inf := &PreviewOrderInfo{}
	_, err := e.PreviewOrder(t.Context(), inf)
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid)
	inf.BaseAmount = 1
	_, err = e.PreviewOrder(t.Context(), inf)
	assert.ErrorIs(t, err, errInvalidOrderType)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	inf.ProductID = testPairStable.String()
	inf.Side = "BUY"
	inf.OrderType = order.Market
	inf.MarginType = "ISOLATED"
	inf.BaseAmount = testAmount
	resp, err := e.PreviewOrder(t.Context(), inf)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetPaymentMethodByID(t *testing.T) {
	t.Parallel()
	_, err := e.GetPaymentMethodByID(t.Context(), "")
	assert.ErrorIs(t, err, errPaymentMethodEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	pmID, err := e.ListPaymentMethods(t.Context())
	assert.NoError(t, err)
	if len(pmID) == 0 {
		t.Skip(skipPayMethodNotFound)
	}
	resp, err := e.GetPaymentMethodByID(t.Context(), pmID[0].ID)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestListPaymentMethods(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	testGetNoArgs(t, e.ListPaymentMethods)
}

func TestAllocatePortfolio(t *testing.T) {
	t.Parallel()
	err := e.AllocatePortfolio(t.Context(), "", "", "", 0)
	assert.ErrorIs(t, err, errPortfolioIDEmpty)
	err = e.AllocatePortfolio(t.Context(), "meow", "", "", 0)
	assert.ErrorIs(t, err, errProductIDEmpty)
	err = e.AllocatePortfolio(t.Context(), "meow", "bark", "", 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	err = e.AllocatePortfolio(t.Context(), "meow", "bark", "woof", 0)
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	pID := getINTXPortfolio(t)
	err = e.AllocatePortfolio(t.Context(), pID, testCrypto.String(), testFiat.String(), 0.001337)
	assert.NoError(t, err)
}

func TestGetPerpetualsPortfolioSummary(t *testing.T) {
	t.Parallel()
	_, err := e.GetPerpetualsPortfolioSummary(t.Context(), "")
	assert.ErrorIs(t, err, errPortfolioIDEmpty)
	pID := getINTXPortfolio(t)
	resp, err := e.GetPerpetualsPortfolioSummary(t.Context(), pID)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetPerpetualsPositionByID(t *testing.T) {
	t.Parallel()
	_, err := e.GetPerpetualsPositionByID(t.Context(), "", currency.Pair{})
	assert.ErrorIs(t, err, errPortfolioIDEmpty)
	_, err = e.GetPerpetualsPositionByID(t.Context(), "meow", currency.Pair{})
	assert.ErrorIs(t, err, errProductIDEmpty)
	pID := getINTXPortfolio(t)
	_, err = e.GetPerpetualsPositionByID(t.Context(), pID, testPairFiat)
	assert.NoError(t, err)
}

func TestGetPortfolioBalances(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioBalances(t.Context(), "")
	assert.ErrorIs(t, err, errPortfolioIDEmpty)
	pID := getINTXPortfolio(t)
	resp, err := e.GetPortfolioBalances(t.Context(), pID)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAllPerpetualsPositions(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllPerpetualsPositions(t.Context(), "")
	assert.ErrorIs(t, err, errPortfolioIDEmpty)
	pID := getINTXPortfolio(t)
	_, err = e.GetAllPerpetualsPositions(t.Context(), pID)
	assert.NoError(t, err)
}

func TestMultiAssetCollateralToggle(t *testing.T) {
	t.Parallel()
	_, err := e.MultiAssetCollateralToggle(t.Context(), "", false)
	assert.ErrorIs(t, err, errPortfolioIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	pID := getINTXPortfolio(t)
	_, err = e.MultiAssetCollateralToggle(t.Context(), pID, false)
	assert.NoError(t, err)
}

func TestCreatePortfolio(t *testing.T) {
	t.Parallel()
	_, err := e.CreatePortfolio(t.Context(), "")
	assert.ErrorIs(t, err, errNameEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CreatePortfolio(t.Context(), "GCT Test Portfolio")
	assert.NoError(t, err)
}

func TestDeletePortfolio(t *testing.T) {
	t.Parallel()
	err := e.DeletePortfolio(t.Context(), "")
	assert.ErrorIs(t, err, errPortfolioIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.DeletePortfolio(t.Context(), "insert portfolio ID here")
	// The new JWT-based keys only have permissions to delete portfolios they're assigned to, causing this to fail
	assert.NoError(t, err)
}

func TestEditPortfolio(t *testing.T) {
	t.Parallel()
	_, err := e.EditPortfolio(t.Context(), "", "")
	assert.ErrorIs(t, err, errPortfolioIDEmpty)
	_, err = e.EditPortfolio(t.Context(), "meow", "")
	assert.ErrorIs(t, err, errNameEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.EditPortfolio(t.Context(), "insert portfolio ID here", "GCT Test Portfolio Edited")
	// The new JWT-based keys only have permissions to edit portfolios they're assigned to, causing this to fail
	assert.NoError(t, err)
}

func TestGetPortfolioByID(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioByID(t.Context(), "")
	assert.ErrorIs(t, err, errPortfolioIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	portID, err := e.GetAllPortfolios(t.Context(), "")
	assert.NoError(t, err)
	if len(portID) == 0 {
		t.Fatal(errExpectedNonEmpty)
	}
	resp, err := e.GetPortfolioByID(t.Context(), portID[0].UUID)
	assert.NoError(t, err)
	if resp.Portfolio != portID[0] {
		t.Errorf(errExpectMismatch, resp.Portfolio, portID[0])
	}
}

func TestGetAllPortfolios(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetAllPortfolios(t.Context(), "DEFAULT")
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestMovePortfolioFunds(t *testing.T) {
	t.Parallel()
	_, err := e.MovePortfolioFunds(t.Context(), currency.Code{}, "", "", 0)
	assert.ErrorIs(t, err, errPortfolioIDEmpty)
	_, err = e.MovePortfolioFunds(t.Context(), currency.Code{}, "meowPort", "woofPort", 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.MovePortfolioFunds(t.Context(), testCrypto, "meowPort", "woofPort", 0)
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	portID, err := e.GetAllPortfolios(t.Context(), "")
	assert.NoError(t, err)
	if len(portID) < 2 {
		t.Skip(skipInsufficientPortfolios)
	}
	_, err = e.MovePortfolioFunds(t.Context(), testCrypto, portID[0].UUID, portID[1].UUID, testAmount)
	assert.NoError(t, err)
}

func TestGetBestBidAsk(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	testPairs := []string{testPairFiat.String(), "ETH-USD"}
	resp, err := e.GetBestBidAsk(t.Context(), testPairs)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetTicker(t.Context(), currency.Pair{}, 1, time.Time{}, time.Time{}, false)
	assert.ErrorIs(t, err, errProductIDEmpty)
	resp, err := e.GetTicker(t.Context(), testPairFiat, 5, time.Now().Add(-time.Minute*5), time.Now(), false)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, resp, errExpectedNonEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err = e.GetTicker(t.Context(), testPairFiat, 5, time.Now().Add(-time.Minute*5), time.Now(), true)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetProductByID(t *testing.T) {
	t.Parallel()
	_, err := e.GetProductByID(t.Context(), currency.Pair{}, false)
	assert.ErrorIs(t, err, errProductIDEmpty)
	resp, err := e.GetProductByID(t.Context(), testPairFiat, false)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, resp, errExpectedNonEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err = e.GetProductByID(t.Context(), testPairFiat, true)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetProductBookV3(t *testing.T) {
	t.Parallel()
	_, err := e.GetProductBookV3(t.Context(), currency.Pair{}, 0, 0, false)
	assert.ErrorIs(t, err, errProductIDEmpty)
	resp, err := e.GetProductBookV3(t.Context(), testPairFiat, 4, -1, false)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, resp, errExpectedNonEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err = e.GetProductBookV3(t.Context(), testPairFiat, 4, -1, true)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetHistoricKlines(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricKlines(t.Context(), "", kline.Raw, time.Time{}, time.Time{}, false)
	assert.ErrorIs(t, err, errProductIDEmpty)
	_, err = e.GetHistoricKlines(t.Context(), testPairFiat.String(), kline.Raw, time.Time{}, time.Time{}, false)
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval)
	resp, err := e.GetHistoricKlines(t.Context(), testPairFiat.String(), kline.OneMin, time.Now().Add(-5*time.Minute), time.Now(), false)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, resp, errExpectedNonEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err = e.GetHistoricKlines(t.Context(), testPairFiat.String(), kline.OneMin, time.Now().Add(-5*time.Minute), time.Now(), true)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAllProducts(t *testing.T) {
	t.Parallel()
	testPairs := []string{testPairFiat.String(), "ETH-USD"}
	resp, err := e.GetAllProducts(t.Context(), 30000, 1, "SPOT", "PERPETUAL", "STATUS_ALL", "PRODUCTS_SORT_ORDER_UNDEFINED", testPairs, true, true, false)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, resp, errExpectedNonEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err = e.GetAllProducts(t.Context(), 0, 1, "SPOT", "PERPETUAL", "STATUS_ALL", "", nil, true, true, true)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetV3Time(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, e.GetV3Time)
}

func TestSendMoney(t *testing.T) {
	t.Parallel()
	_, err := e.SendMoney(t.Context(), "", "", "", "", "", "", "", currency.Code{}, 0, false, &TravelRule{})
	assert.ErrorIs(t, err, errTransactionTypeEmpty)
	_, err = e.SendMoney(t.Context(), "123", "", "", "", "", "", "", currency.Code{}, 0, false, &TravelRule{})
	assert.ErrorIs(t, err, errWalletIDEmpty)
	_, err = e.SendMoney(t.Context(), "123", "123", "", "", "", "", "", currency.Code{}, 0, false, &TravelRule{})
	assert.ErrorIs(t, err, errToEmpty)
	_, err = e.SendMoney(t.Context(), "123", "123", "123", "", "", "", "", currency.Code{}, 0, false, &TravelRule{})
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid)
	_, err = e.SendMoney(t.Context(), "123", "123", "123", "", "", "", "", currency.Code{}, 1, false, &TravelRule{})
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	wID, err := e.GetAllWallets(t.Context(), PaginationInp{})
	assert.NoError(t, err)
	if wID == nil || len(wID.Data) < 2 {
		t.Skip(skipInsufficientWallets)
	}
	var (
		fromID string
		toID   string
	)
	for i := range wID.Data {
		if wID.Data[i].Currency.Name == testCrypto.String() {
			if wID.Data[i].Balance.Amount > testAmount*100 {
				fromID = wID.Data[i].ID
			} else {
				toID = wID.Data[i].ID
			}
		}
		if fromID != "" && toID != "" {
			break
		}
	}
	if fromID == "" || toID == "" {
		t.Skip(skipInsufficientFundsOrWallets)
	}
	resp, err := e.SendMoney(t.Context(), "transfer", wID.Data[0].ID, wID.Data[1].ID, "GCT Test", "123", "", "", testCrypto, testAmount, false, &TravelRule{})
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestCreateAddress(t *testing.T) {
	t.Parallel()
	_, err := e.CreateAddress(t.Context(), "", "")
	assert.ErrorIs(t, err, errWalletIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	wID, err := e.GetWalletByID(t.Context(), "", testCrypto)
	require.NoError(t, err)
	require.NotEmpty(t, wID, errExpectedNonEmpty)
	resp, err := e.CreateAddress(t.Context(), wID.ID, "")
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAllAddresses(t *testing.T) {
	t.Parallel()
	var pag PaginationInp
	_, err := e.GetAllAddresses(t.Context(), "", pag)
	assert.ErrorIs(t, err, errWalletIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	wID, err := e.GetWalletByID(t.Context(), "", testCrypto)
	require.NoError(t, err)
	require.NotEmpty(t, wID, errExpectedNonEmpty)
	resp, err := e.GetAllAddresses(t.Context(), wID.ID, pag)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAddressByID(t *testing.T) {
	t.Parallel()
	_, err := e.GetAddressByID(t.Context(), "", "")
	assert.ErrorIs(t, err, errWalletIDEmpty)
	_, err = e.GetAddressByID(t.Context(), "123", "")
	assert.ErrorIs(t, err, errAddressIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	wID, err := e.GetWalletByID(t.Context(), "", testCrypto)
	require.NoError(t, err)
	require.NotEmpty(t, wID, errExpectedNonEmpty)
	addID, err := e.GetAllAddresses(t.Context(), wID.ID, PaginationInp{})
	require.NoError(t, err)
	require.NotEmpty(t, addID, errExpectedNonEmpty)
	resp, err := e.GetAddressByID(t.Context(), wID.ID, addID.Data[0].ID)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAddressTransactions(t *testing.T) {
	t.Parallel()
	_, err := e.GetAddressTransactions(t.Context(), "", "", PaginationInp{})
	assert.ErrorIs(t, err, errWalletIDEmpty)
	_, err = e.GetAddressTransactions(t.Context(), "123", "", PaginationInp{})
	assert.ErrorIs(t, err, errAddressIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	wID, err := e.GetWalletByID(t.Context(), "", testCrypto)
	require.NoError(t, err)
	require.NotEmpty(t, wID, errExpectedNonEmpty)
	addID, err := e.GetAllAddresses(t.Context(), wID.ID, PaginationInp{})
	require.NoError(t, err)
	require.NotEmpty(t, addID, errExpectedNonEmpty)
	_, err = e.GetAddressTransactions(t.Context(), wID.ID, addID.Data[0].ID, PaginationInp{})
	assert.NoError(t, err)
}

func TestFiatTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.FiatTransfer(t.Context(), "", "", "", 0, false, FiatDeposit)
	assert.ErrorIs(t, err, errWalletIDEmpty)
	_, err = e.FiatTransfer(t.Context(), "123", "", "", 0, false, FiatDeposit)
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid)
	_, err = e.FiatTransfer(t.Context(), "123", "", "", 1, false, FiatDeposit)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.FiatTransfer(t.Context(), "123", "123", "", 1, false, FiatDeposit)
	assert.ErrorIs(t, err, errPaymentMethodEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	wallets, err := e.GetAllWallets(t.Context(), PaginationInp{})
	require.NoError(t, err)
	assert.NotEmpty(t, wallets, errExpectedNonEmpty)
	wID, pmID := transferTestHelper(t, wallets)
	resp, err := e.FiatTransfer(t.Context(), wID, testFiat.String(), pmID, testAmount, false, FiatDeposit)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, resp, errExpectedNonEmpty)
	}
	resp, err = e.FiatTransfer(t.Context(), wID, testFiat.String(), pmID, testAmount, false, FiatWithdrawal)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestCommitTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.CommitTransfer(t.Context(), "", "", FiatDeposit)
	assert.ErrorIs(t, err, errWalletIDEmpty)
	_, err = e.CommitTransfer(t.Context(), "123", "", FiatDeposit)
	assert.ErrorIs(t, err, errDepositIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	wallets, err := e.GetAllWallets(t.Context(), PaginationInp{})
	require.NoError(t, err)
	assert.NotEmpty(t, wallets, errExpectedNonEmpty)
	wID, pmID := transferTestHelper(t, wallets)
	depID, err := e.FiatTransfer(t.Context(), wID, testFiat.String(), pmID, testAmount, false, FiatDeposit)
	require.NoError(t, err)
	resp, err := e.CommitTransfer(t.Context(), wID, depID.ID, FiatDeposit)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, resp, errExpectedNonEmpty)
	}
	depID, err = e.FiatTransfer(t.Context(), wID, testFiat.String(), pmID, testAmount, false, FiatWithdrawal)
	require.NoError(t, err)
	resp, err = e.CommitTransfer(t.Context(), wID, depID.ID, FiatWithdrawal)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAllFiatTransfers(t *testing.T) {
	t.Parallel()
	var pag PaginationInp
	_, err := e.GetAllFiatTransfers(t.Context(), "", pag, FiatDeposit)
	assert.ErrorIs(t, err, errWalletIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	wID, err := e.GetWalletByID(t.Context(), "", currency.AUD)
	require.NoError(t, err)
	assert.NotEmpty(t, wID, errExpectedNonEmpty)
	// Fiat deposits/withdrawals aren't accepted for fiat currencies for Australian business accounts; the error "id not found" possibly reflects this
	_, err = e.GetAllFiatTransfers(t.Context(), wID.ID, pag, FiatDeposit)
	assert.NoError(t, err)
	_, err = e.GetAllFiatTransfers(t.Context(), wID.ID, pag, FiatWithdrawal)
	assert.NoError(t, err)
}

func TestGetFiatTransferByID(t *testing.T) {
	t.Parallel()
	_, err := e.GetFiatTransferByID(t.Context(), "", "", FiatDeposit)
	assert.ErrorIs(t, err, errWalletIDEmpty)
	_, err = e.GetFiatTransferByID(t.Context(), "123", "", FiatDeposit)
	assert.ErrorIs(t, err, errDepositIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	wID, err := e.GetWalletByID(t.Context(), "", currency.AUD)
	require.NoError(t, err)
	assert.NotEmpty(t, wID, errExpectedNonEmpty)
	// Fiat deposits/withdrawals aren't accepted for fiat currencies for Australian business accounts; the error "id not found" possibly reflects this
	dID, err := e.GetAllFiatTransfers(t.Context(), wID.ID, PaginationInp{}, FiatDeposit)
	assert.NoError(t, err)
	if dID == nil || len(dID.Data) == 0 {
		t.Skip(skipInsufficientTransactions)
	}
	resp, err := e.GetFiatTransferByID(t.Context(), wID.ID, dID.Data[0].ID, FiatDeposit)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, resp, errExpectedNonEmpty)
	}
	resp, err = e.GetFiatTransferByID(t.Context(), wID.ID, dID.Data[0].ID, FiatWithdrawal)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAllWallets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	pagIn := PaginationInp{Limit: 2}
	resp, err := e.GetAllWallets(t.Context(), pagIn)
	assert.NoError(t, err)
	require.NotEmpty(t, resp, errExpectedNonEmpty)
	if resp.Pagination.NextStartingAfter == "" {
		t.Skip(skipInsufficientWallets)
	}
	pagIn.StartingAfter = resp.Pagination.NextStartingAfter
	resp, err = e.GetAllWallets(t.Context(), pagIn)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetWalletByID(t *testing.T) {
	t.Parallel()
	_, err := e.GetWalletByID(t.Context(), "", currency.Code{})
	assert.ErrorIs(t, err, errCurrWalletConflict)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetWalletByID(t.Context(), "", testCrypto)
	require.NoError(t, err)
	require.NotEmpty(t, resp, errExpectedNonEmpty)
	resp, err = e.GetWalletByID(t.Context(), resp.ID, currency.Code{})
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAllTransactions(t *testing.T) {
	t.Parallel()
	var pag PaginationInp
	_, err := e.GetAllTransactions(t.Context(), "", pag)
	assert.ErrorIs(t, err, errWalletIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	wID, err := e.GetWalletByID(t.Context(), "", testCrypto)
	require.NoError(t, err)
	require.NotEmpty(t, wID, errExpectedNonEmpty)
	_, err = e.GetAllTransactions(t.Context(), wID.ID, pag)
	assert.NoError(t, err)
}

func TestGetTransactionByID(t *testing.T) {
	t.Parallel()
	_, err := e.GetTransactionByID(t.Context(), "", "")
	assert.ErrorIs(t, err, errWalletIDEmpty)
	_, err = e.GetTransactionByID(t.Context(), "123", "")
	assert.ErrorIs(t, err, errTransactionIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	wID, err := e.GetWalletByID(t.Context(), "", testCrypto)
	require.NoError(t, err)
	require.NotEmpty(t, wID, errExpectedNonEmpty)
	tID, err := e.GetAllTransactions(t.Context(), wID.ID, PaginationInp{})
	assert.NoError(t, err)
	if tID == nil || len(tID.Data) == 0 {
		t.Skip(skipInsufficientTransactions)
	}
	resp, err := e.GetTransactionByID(t.Context(), wID.ID, tID.Data[0].ID)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetFiatCurrencies(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, e.GetFiatCurrencies)
}

func TestGetCryptocurrencies(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, e.GetCryptocurrencies)
}

func TestGetExchangeRates(t *testing.T) {
	t.Parallel()
	resp, err := e.GetExchangeRates(t.Context(), "")
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetPrice(t.Context(), "", "")
	assert.ErrorIs(t, err, errInvalidPriceType)
	resp, err := e.GetPrice(t.Context(), testPairFiat.String(), asset.Spot.String())
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetV2Time(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, e.GetV2Time)
}

func TestGetCurrentUser(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	// This intermittently fails with the message "Unauthorized", for no clear reason
	testGetNoArgs(t, e.GetCurrentUser)
}

func TestGetAllCurrencies(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, e.GetAllCurrencies)
}

func TestGetACurrency(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetACurrency, testCrypto.String(), currency.ErrCurrencyCodeEmpty)
}

func TestGetAllTradingPairs(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllTradingPairs(t.Context(), "")
	assert.NoError(t, err)
}

func TestGetAllPairVolumes(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, e.GetAllPairVolumes)
}

func TestGetPairDetails(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetPairDetails, testPairFiat.String(), currency.ErrCurrencyPairEmpty)
}

func TestGetProductBookV1(t *testing.T) {
	t.Parallel()
	_, err := e.GetProductBookV1(t.Context(), "", 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	resp, err := e.GetProductBookV1(t.Context(), testPairFiat.String(), 2)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, resp, errExpectedNonEmpty)
	}
	resp, err = e.GetProductBookV1(t.Context(), testPairFiat.String(), 3)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetProductCandles(t *testing.T) {
	t.Parallel()
	_, err := e.GetProductCandles(t.Context(), "", 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	resp, err := e.GetProductCandles(t.Context(), testPairFiat.String(), 300, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetProductStats(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetProductStats, testPairFiat.String(), currency.ErrCurrencyPairEmpty)
}

func TestGetProductTicker(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetProductTicker, testPairFiat.String(), currency.ErrCurrencyPairEmpty)
}

func TestGetProductTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetProductTrades(t.Context(), "", "", "", 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	resp, err := e.GetProductTrades(t.Context(), testPairFiat.String(), "1", "before", 0)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAllWrappedAssets(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, e.GetAllWrappedAssets)
}

func TestGetWrappedAssetDetails(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetWrappedAssetDetails, testWrappedAsset.String(), errWrappedAssetEmpty)
}

func TestGetWrappedAssetConversionRate(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetWrappedAssetConversionRate, testWrappedAsset.String(), errWrappedAssetEmpty)
}

func TestSendHTTPRequest(t *testing.T) {
	t.Parallel()
	err := e.SendHTTPRequest(t.Context(), exchange.EdgeCase3, "", nil, nil)
	assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound)
}

func TestSendAuthenticatedHTTPRequest(t *testing.T) {
	t.Parallel()
	err := e.SendAuthenticatedHTTPRequest(t.Context(), exchange.EdgeCase3, "", "", nil, nil, false, nil)
	assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound)
	ch := make(chan struct{})
	body := map[string]any{"Unmarshalable": ch}
	err = e.SendAuthenticatedHTTPRequest(t.Context(), exchange.RestSpot, "", "", nil, body, false, nil)
	// TODO: Implement this more rigorously once thrasher investigates the code further
	// var targetErr *json.UnsupportedTypeError
	// assert.ErrorAs(t, err, &targetErr)
	assert.ErrorContains(t, err, "json: unsupported type: chan struct {}")
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	_, err := e.GetFee(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)
	feeBuilder := exchange.FeeBuilder{
		FeeType:       exchange.OfflineTradeFee,
		Amount:        1,
		PurchasePrice: 1,
	}
	resp, err := e.GetFee(t.Context(), &feeBuilder)
	assert.NoError(t, err)
	if resp != WorstCaseTakerFee {
		t.Errorf(errExpectMismatch, resp, WorstCaseTakerFee)
	}
	feeBuilder.IsMaker = true
	resp, err = e.GetFee(t.Context(), &feeBuilder)
	assert.NoError(t, err)
	if resp != WorstCaseMakerFee {
		t.Errorf(errExpectMismatch, resp, WorstCaseMakerFee)
	}
	feeBuilder.Pair = currency.NewPair(currency.USDT, currency.USD)
	resp, err = e.GetFee(t.Context(), &feeBuilder)
	assert.NoError(t, err)
	if resp != 0 {
		t.Errorf(errExpectMismatch, resp, StablePairMakerFee)
	}
	feeBuilder.IsMaker = false
	resp, err = e.GetFee(t.Context(), &feeBuilder)
	assert.NoError(t, err)
	if resp != WorstCaseStablePairTakerFee {
		t.Errorf(errExpectMismatch, resp, WorstCaseStablePairTakerFee)
	}
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	_, err = e.GetFee(t.Context(), &feeBuilder)
	assert.ErrorIs(t, err, errFeeTypeNotSupported)
	feeBuilder.Pair = currency.Pair{}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	feeBuilder.FeeType = exchange.CryptocurrencyTradeFee
	resp, err = e.GetFee(t.Context(), &feeBuilder)
	assert.NoError(t, err)
	if !(resp <= WorstCaseTakerFee && resp >= BestCaseTakerFee) {
		t.Errorf(errExpectedFeeRange, BestCaseTakerFee, WorstCaseTakerFee, resp)
	}
	feeBuilder.IsMaker = true
	resp, err = e.GetFee(t.Context(), &feeBuilder)
	assert.NoError(t, err)
	if !(resp <= WorstCaseMakerFee && resp >= BestCaseMakerFee) {
		t.Errorf(errExpectedFeeRange, BestCaseMakerFee, WorstCaseMakerFee, resp)
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := e.FetchTradablePairs(t.Context(), asset.Options)
	assert.Equal(t, errOptionInvalid, err.Error())
	resp, err := e.FetchTradablePairs(t.Context(), asset.Spot)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, resp, errExpectedNonEmpty)
	}
	resp, err = e.FetchTradablePairs(t.Context(), asset.Futures)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
}

func TestUpdateAccountBalances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTicker(t.Context(), currency.Pair{}, asset.Spot)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	resp, err := e.UpdateTicker(t.Context(), testPairFiat, asset.Spot)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

// TestUpdateOrderbook does not run in parallel; being parallel causes intermittent errors with another test for no discernible reason
func TestUpdateOrderbook(t *testing.T) {
	testexch.UpdatePairsOnce(t, e)
	_, err := e.UpdateOrderbook(t.Context(), currency.Pair{}, asset.Empty)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.UpdateOrderbook(t.Context(), testPairFiat, asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	_, err = e.UpdateOrderbook(t.Context(), currency.NewPairWithDelimiter("meow", "woof", "-"), asset.Spot)
	assert.Equal(t, errInvalidProductID, err.Error())
	// There are no perpetual futures contracts, so I can only deterministically test spot
	resp, err := e.UpdateOrderbook(t.Context(), testPairFiat, asset.Spot)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetAccountFundingHistory(t.Context())
	assert.NoError(t, err)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetWithdrawalsHistory(t.Context(), currency.NewCode("meow"), asset.Spot)
	assert.ErrorIs(t, err, errNoMatchingWallets)
	_, err = e.GetWithdrawalsHistory(t.Context(), testCrypto, asset.Spot)
	assert.NoError(t, err)
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitOrder(t.Context(), nil)
	assert.ErrorIs(t, err, order.ErrSubmissionIsNil)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	ord := order.Submit{
		Exchange:      e.Name,
		Pair:          testPairStable,
		AssetType:     asset.Spot,
		Side:          order.Buy,
		Type:          order.StopLimit,
		StopDirection: order.StopUp,
		Amount:        testAmount2,
		Price:         testPrice,
		TriggerPrice:  testPrice + 1,
		RetrieveFees:  true,
		ClientOrderID: strconv.FormatInt(time.Now().UnixMilli(), 18) + "GCTSubmitOrderTest",
	}
	resp, err := e.SubmitOrder(t.Context(), &ord)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, resp, errExpectedNonEmpty)
	}
	ord.StopDirection = order.StopDown
	ord.TriggerPrice = testPrice/2 + 1
	resp, err = e.SubmitOrder(t.Context(), &ord)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, resp, errExpectedNonEmpty)
	}
	ord.Type = order.Market
	ord.QuoteAmount = testAmount3
	resp, err = e.SubmitOrder(t.Context(), &ord)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyOrder(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)
	var ord order.Modify
	_, err = e.ModifyOrder(t.Context(), &ord)
	assert.ErrorIs(t, err, order.ErrPairIsEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	ord.OrderID = "a"
	ord.Price = testPrice + 1
	ord.Amount = testAmount
	ord.Pair = testPairStable
	ord.AssetType = asset.Spot
	resp2, err := e.ModifyOrder(t.Context(), &ord)
	require.NoError(t, err)
	assert.NotEmpty(t, resp2, errExpectedNonEmpty)
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	err := e.CancelOrder(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)
	var can order.Cancel
	err = e.CancelOrder(t.Context(), &can)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	can.OrderID = "0"
	err = e.CancelOrder(t.Context(), &can)
	assert.Error(t, err)
	can.OrderID = "2"
	err = e.CancelOrder(t.Context(), &can)
	assert.NoError(t, err)
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBatchOrders(t.Context(), nil)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)
	can := make([]order.Cancel, 1)
	_, err = e.CancelBatchOrders(t.Context(), can)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	can[0].OrderID = "1"
	resp2, err := e.CancelBatchOrders(t.Context(), can)
	require.NoError(t, err)
	assert.NotEmpty(t, resp2, errExpectedNonEmpty)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	resp, err := e.GetOrderInfo(t.Context(), "17", testPairStable, asset.Spot)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetDepositAddress(t.Context(), currency.NewCode("fake currency that doesn't exist"), "", "")
	assert.ErrorIs(t, err, errNoWalletForCurrency)
	resp, err := e.GetDepositAddress(t.Context(), testCrypto, "", "")
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	req := withdraw.Request{}
	_, err := e.WithdrawCryptocurrencyFunds(t.Context(), &req)
	assert.ErrorIs(t, err, common.ErrExchangeNameNotSet)
	req.Exchange = e.Name
	req.Currency = testCrypto
	req.Amount = testAmount
	req.Type = withdraw.Crypto
	req.Crypto.Address = testAddress
	_, err = e.WithdrawCryptocurrencyFunds(t.Context(), &req)
	assert.ErrorIs(t, err, errWalletIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	req.Amount = -0.1
	wallets, err := e.GetAllWallets(t.Context(), PaginationInp{})
	assert.NoError(t, err)
	if wallets == nil || len(wallets.Data) == 0 {
		t.Fatal(errExpectedNonEmpty)
	}
	for i := range wallets.Data {
		if wallets.Data[i].Currency.Name == testCrypto.String() && wallets.Data[i].Balance.Amount > testAmount*100 {
			req.WalletID = wallets.Data[i].ID
			break
		}
	}
	if req.WalletID == "" {
		t.Skip(skipInsufficientFunds)
	}
	resp, err := e.WithdrawCryptocurrencyFunds(t.Context(), &req)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestWithdrawFiatFunds(t *testing.T) {
	withdrawFiatFundsHelper(t, e.WithdrawFiatFunds)
}

func TestWithdrawFiatFundsToInternationalBank(t *testing.T) {
	withdrawFiatFundsHelper(t, e.WithdrawFiatFundsToInternationalBank)
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	_, err := e.GetFeeByType(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)
	var feeBuilder exchange.FeeBuilder
	feeBuilder.FeeType = exchange.OfflineTradeFee
	feeBuilder.Amount = 1
	feeBuilder.PurchasePrice = 1
	resp, err := e.GetFeeByType(t.Context(), &feeBuilder)
	assert.NoError(t, err)
	if resp != WorstCaseTakerFee {
		t.Errorf(errExpectMismatch, resp, WorstCaseTakerFee)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetActiveOrders(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)
	var req order.MultiOrderRequest
	_, err = e.GetActiveOrders(t.Context(), &req)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	req.AssetType = asset.Spot
	req.Side = order.AnySide
	req.Type = order.AnyType
	_, err = e.GetActiveOrders(t.Context(), &req)
	assert.NoError(t, err)
	req.Pairs = req.Pairs.Add(currency.NewPair(testCrypto, testFiat))
	_, err = e.GetActiveOrders(t.Context(), &req)
	assert.NoError(t, err)
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderHistory(t.Context(), nil)
	assert.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	var req order.MultiOrderRequest
	req.AssetType = asset.Spot
	req.Side = order.AnySide
	req.Type = order.AnyType
	_, err = e.GetOrderHistory(t.Context(), &req)
	assert.NoError(t, err)
	req.Pairs = req.Pairs.Add(testPairStable)
	_, err = e.GetOrderHistory(t.Context(), &req)
	assert.NoError(t, err)
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricCandles(t.Context(), currency.Pair{}, asset.Empty, kline.OneYear, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	resp, err := e.GetHistoricCandles(t.Context(), testPairFiat, asset.Spot, kline.SixHour, time.Now().Add(-time.Hour*60), time.Now())
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricCandlesExtended(t.Context(), currency.Pair{}, asset.Empty, kline.OneYear, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	resp, err := e.GetHistoricCandlesExtended(t.Context(), testPairFiat, asset.Spot, kline.OneMin, time.Now().Add(-time.Hour*9), time.Now())
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	err := e.ValidateAPICredentials(t.Context(), asset.Spot)
	assert.NoError(t, err)
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := e.GetServerTime(t.Context(), 0)
	assert.NoError(t, err)
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetLatestFundingRates(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)
	req := fundingrate.LatestRateRequest{Asset: asset.UpsideProfitContract}
	_, err = e.GetLatestFundingRates(t.Context(), &req)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	req.Asset = asset.Futures
	resp, err := e.GetLatestFundingRates(t.Context(), &req)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Empty)
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset)
	_, err = e.GetFuturesContractDetails(t.Context(), asset.UpsideProfitContract)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetFuturesContractDetails(t.Context(), asset.Futures)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := e.UpdateOrderExecutionLimits(t.Context(), asset.Options)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	err = e.UpdateOrderExecutionLimits(t.Context(), asset.Spot)
	assert.NoError(t, err)
}

func TestGetOrderRespToOrderDetail(t *testing.T) {
	t.Parallel()
	mockData := &GetOrderResponse{
		OrderConfiguration: OrderConfiguration{
			MarketMarketIOC:       &MarketMarketIOC{},
			LimitLimitGTC:         &LimitLimitGTC{},
			LimitLimitGTD:         &LimitLimitGTD{},
			StopLimitStopLimitGTC: &StopLimitStopLimitGTC{},
			StopLimitStopLimitGTD: &StopLimitStopLimitGTD{},
		},
		SizeInQuote: false,
		Side:        "BUY",
		Status:      "OPEN",
		Settled:     true,
		EditHistory: []EditHistory{(EditHistory{})},
	}
	resp := e.getOrderRespToOrderDetail(mockData, testPairStable, asset.Spot)
	expected := &order.Detail{TimeInForce: order.ImmediateOrCancel, Exchange: "Coinbase", Type: order.StopLimit, Side: order.Buy, Status: order.Open, AssetType: asset.Spot, Date: time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC), CloseTime: time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC), LastUpdated: time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC), Pair: testPairStable}
	assert.Equal(t, expected, resp)
	mockData.Side = "SELL"
	mockData.Status = "FILLED"
	resp = e.getOrderRespToOrderDetail(mockData, testPairStable, asset.Spot)
	expected.Side = order.Sell
	expected.Status = order.Filled
	assert.Equal(t, expected, resp)
	mockData.Status = "CANCELLED"
	resp = e.getOrderRespToOrderDetail(mockData, testPairStable, asset.Spot)
	expected.Status = order.Cancelled
	assert.Equal(t, expected, resp)
	mockData.Status = "EXPIRED"
	resp = e.getOrderRespToOrderDetail(mockData, testPairStable, asset.Spot)
	expected.Status = order.Expired
	assert.Equal(t, expected, resp)
	mockData.Status = "FAILED"
	resp = e.getOrderRespToOrderDetail(mockData, testPairStable, asset.Spot)
	expected.Status = order.Rejected
	assert.Equal(t, expected, resp)
	mockData.Status = "UNKNOWN_ORDER_STATUS"
	resp = e.getOrderRespToOrderDetail(mockData, testPairStable, asset.Spot)
	expected.Status = order.UnknownStatus
	assert.Equal(t, expected, resp)
}

func TestFiatTransferTypeString(t *testing.T) {
	t.Parallel()
	var f FiatTransferType
	if f.String() != "deposit" {
		t.Errorf(errExpectMismatch, f.String(), "deposit")
	}
	f = FiatWithdrawal
	if f.String() != "withdrawal" {
		t.Errorf(errExpectMismatch, f.String(), "withdrawal")
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.CurrencyPairs.GetPairs(a, false)
		require.NoErrorf(t, err, "cannot get pairs for %s", a)
		require.NotEmptyf(t, pairs, "no pairs for %s", a)
		resp, err := e.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}

// TestWsAuth dials websocket, sends login request.
func TestWsAuth(t *testing.T) {
	p := currency.Pairs{testPairFiat}
	if e.Websocket.IsEnabled() && !e.API.AuthenticatedWebsocketSupport || !sharedtestvalues.AreAPICredentialsSet(e) {
		t.Skip(websocket.ErrWebsocketNotEnabled.Error())
	}
	var dialer gws.Dialer
	err := e.Websocket.Conn.Dial(t.Context(), &dialer, http.Header{})
	require.NoError(t, err)
	e.Websocket.Wg.Add(1)
	go e.wsReadData(t.Context())
	err = e.Subscribe(subscription.List{
		{
			Channel:       "myAccount",
			Asset:         asset.All,
			Pairs:         p,
			Authenticated: true,
		},
	})
	assert.NoError(t, err)
	timer := time.NewTimer(sharedtestvalues.WebsocketResponseDefaultTimeout)
	select {
	case badResponse := <-e.Websocket.DataHandler.C:
		assert.IsType(t, []order.Detail{}, badResponse)
	case <-timer.C:
	}
	timer.Stop()
}

func TestWsHandleData(t *testing.T) {
	done := make(chan struct{})
	t.Cleanup(func() {
		close(done)
	})
	go func() {
		for {
			select {
			case <-e.Websocket.DataHandler.C:
				continue
			case <-done:
				return
			}
		}
	}()
	_, err := e.wsHandleData(t.Context(), nil)
	var syntaxErr *json.SyntaxError
	assert.True(t, errors.As(err, &syntaxErr) || strings.Contains(err.Error(), "Syntax error no sources available, the input json is empty"), errJSONUnmarshalUnexpected)
	mockJSON := []byte(`{"type": "error"}`)
	_, err = e.wsHandleData(t.Context(), mockJSON)
	assert.Error(t, err)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "subscriptions"}`)
	_, err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
	var unmarshalTypeErr *json.UnmarshalTypeError
	mockJSON = []byte(`{"sequence_num": 0, "channel": "status", "events": [{"type": 1234}]}`)
	_, err = e.wsHandleData(t.Context(), mockJSON)
	assert.True(t, errors.As(err, &unmarshalTypeErr) || strings.Contains(err.Error(), "mismatched type with value"), errJSONUnmarshalUnexpected)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "status", "events": [{"type": "moo"}]}`)
	_, err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "ticker", "events": [{"type": "moo", "tickers": false}]}`)
	_, err = e.wsHandleData(t.Context(), mockJSON)
	assert.True(t, errors.As(err, &unmarshalTypeErr) || strings.Contains(err.Error(), "mismatched type with value"), errJSONUnmarshalUnexpected)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "candles", "events": [{"type": false}]}`)
	_, err = e.wsHandleData(t.Context(), mockJSON)
	assert.True(t, errors.As(err, &unmarshalTypeErr) || strings.Contains(err.Error(), "mismatched type with value"), errJSONUnmarshalUnexpected)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "candles", "events": [{"type": "moo", "candles": [{"low": "1.1"}]}]}`)
	_, err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "market_trades", "events": [{"type": false}]}`)
	_, err = e.wsHandleData(t.Context(), mockJSON)
	assert.True(t, errors.As(err, &unmarshalTypeErr) || strings.Contains(err.Error(), "mismatched type with value"), errJSONUnmarshalUnexpected)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "market_trades", "events": [{"type": "moo", "trades": [{"price": "1.1"}]}]}`)
	_, err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "l2_data", "events": [{"type": false, "updates": [{"price_level": "1.1"}]}]}`)
	_, err = e.wsHandleData(t.Context(), mockJSON)
	assert.True(t, errors.As(err, &unmarshalTypeErr) || strings.Contains(err.Error(), "mismatched type with value"), errJSONUnmarshalUnexpected)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "l2_data", "timestamp": "2006-01-02T15:04:05Z", "events": [{"type": "moo", "updates": [{"price_level": "1.1"}]}]}`)
	_, err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorIs(t, err, errUnknownL2DataType)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "l2_data", "timestamp": "2006-01-02T15:04:05Z", "events": [{"type": "snapshot", "product_id": "BTC-USD", "updates": [{"side": "bid", "price_level": "1.1", "new_quantity": "2.2"}]}]}`)
	_, err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "l2_data", "timestamp": "2006-01-02T15:04:05Z", "events": [{"type": "update", "product_id": "BTC-USD", "updates": [{"side": "bid", "price_level": "1.1", "new_quantity": "2.2"}]}]}`)
	_, err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "user", "events": [{"type": false}]}`)
	_, err = e.wsHandleData(t.Context(), mockJSON)
	assert.True(t, errors.As(err, &unmarshalTypeErr) || strings.Contains(err.Error(), "mismatched type with value"), errJSONUnmarshalUnexpected)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "user", "events": [{"type": "l", "orders": [{"limit_price": "2.2", "total_fees": "1.1", "post_only": true}], "positions": {"perpetual_futures_positions": [{"margin_type": "fakeMarginType"}], "expiring_futures_positions": [{}]}}]}`)
	_, err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorIs(t, err, order.ErrUnrecognisedOrderType)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "fakechan", "events": [{"type": ""}]}`)
	_, err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorIs(t, err, errChannelNameUnknown)
	p, err := e.FormatExchangeCurrency(currency.NewBTCUSD(), asset.Spot)
	require.NoError(t, err)
	e.pairAliases.Load(map[currency.Pair]currency.Pairs{
		p: {p},
	})
	mockJSON = []byte(`{"sequence_num": 0, "channel": "ticker", "events": [{"type": "moo", "tickers": [{"product_id": "BTC-USD", "price": "1.1"}]}]}`)
	_, err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
}

func TestProcessSnapshotUpdate(t *testing.T) {
	t.Parallel()
	req := WebsocketOrderbookDataHolder{Changes: []WebsocketOrderbookData{{Side: "fakeside", PriceLevel: 1.1, NewQuantity: 2.2}}, ProductID: currency.NewBTCUSD()}
	err := e.ProcessSnapshot(&req, time.Time{})
	assert.ErrorIs(t, err, order.ErrSideIsInvalid)
	err = e.ProcessUpdate(&req, time.Time{})
	assert.ErrorIs(t, err, order.ErrSideIsInvalid)
	req.Changes[0].Side = "offer"
	err = e.ProcessSnapshot(&req, time.Now())
	assert.NoError(t, err)
	err = e.ProcessUpdate(&req, time.Now())
	assert.NoError(t, err)
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatal(err)
	}
	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	p1, err := e.GetEnabledPairs(asset.Spot)
	require.NoError(t, err)
	p2, err := e.GetEnabledPairs(asset.Futures)
	require.NoError(t, err)
	exp := subscription.List{}
	for _, baseSub := range defaultSubscriptions.Enabled() {
		s := baseSub.Clone()
		s.QualifiedChannel = subscriptionNames[s.Channel]
		switch s.Asset {
		case asset.Spot:
			s.Pairs = p1
		case asset.Futures:
			s.Pairs = p2
		case asset.All:
			s2 := s.Clone()
			s2.Asset = asset.Futures
			s2.Pairs = p2
			exp = append(exp, s2)
			s.Asset = asset.Spot
			s.Pairs = p1
		}
		exp = append(exp, s)
	}
	subs, err := e.generateSubscriptions()
	require.NoError(t, err)
	testsubs.EqualLists(t, exp, subs)
	_, err = subscription.List{{Channel: "wibble"}}.ExpandTemplates(e)
	assert.ErrorContains(t, err, "subscription channel not supported: wibble")
}

func TestSubscribeUnsubscribe(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	req := subscription.List{{Channel: "heartbeat", Asset: asset.Spot, Pairs: currency.Pairs{currency.NewPairWithDelimiter(testCrypto.String(), testFiat.String(), "-")}}}
	err := e.Subscribe(req)
	assert.NoError(t, err)
	err = e.Unsubscribe(req)
	assert.NoError(t, err)
}

func TestCheckSubscriptions(t *testing.T) {
	t.Parallel()
	e := &Exchange{
		Base: exchange.Base{
			Config: &config.Exchange{
				Features: &config.FeaturesConfig{
					Subscriptions: subscription.List{
						{Enabled: true, Channel: "matches"},
					},
				},
			},
			Features: exchange.Features{},
		},
	}
	e.checkSubscriptions()
	testsubs.EqualLists(t, defaultSubscriptions.Enabled(), e.Features.Subscriptions)
	testsubs.EqualLists(t, defaultSubscriptions, e.Config.Features.Subscriptions)
}

func TestGetJWT(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, _, err := e.GetJWT(t.Context(), "a")
	assert.NoError(t, err)
}

func TestEncodeDateRange(t *testing.T) {
	t.Parallel()
	_, err := urlValsFromDateRange(time.Time{}, time.Time{}, "", "")
	assert.NoError(t, err)
	_, err = urlValsFromDateRange(time.Unix(1, 1), time.Unix(1, 1), "", "")
	assert.ErrorIs(t, err, common.ErrStartEqualsEnd)
	_, err = urlValsFromDateRange(time.Unix(1, 1), time.Unix(2, 2), "", "")
	assert.ErrorIs(t, err, errDateLabelEmpty)
	vals, err := urlValsFromDateRange(time.Unix(1, 1), time.Unix(2, 2), "start", "end")
	assert.NoError(t, err)
	assert.NotEmpty(t, vals)
}

func TestEncodePagination(t *testing.T) {
	t.Parallel()
	vals := urlValsFromPagination(PaginationInp{
		Limit:         1,
		OrderAscend:   true,
		StartingAfter: "a",
		EndingBefore:  "b",
	})
	assert.NotEmpty(t, vals)
}

func TestCreateOrderConfig(t *testing.T) {
	t.Parallel()
	_, err := createOrderConfig(nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)
	params := &OrderInfo{}
	_, err = createOrderConfig(params)
	assert.ErrorIs(t, err, errInvalidOrderType)
	params.BaseAmount = 1
	params.QuoteAmount = 2
	params.OrderType = order.Market
	_, err = createOrderConfig(params)
	assert.NoError(t, err)
	params.OrderType = order.Limit
	params.TimeInForce = order.StopOrReduce
	_, err = createOrderConfig(params)
	assert.NoError(t, err)
	params.TimeInForce = order.FillOrKill
	_, err = createOrderConfig(params)
	assert.NoError(t, err)
	params.TimeInForce = order.UnknownTIF
	_, err = createOrderConfig(params)
	assert.NoError(t, err)
	params.EndTime = time.Unix(1, 1)
	_, err = createOrderConfig(params)
	assert.ErrorIs(t, err, errEndTimeInPast)
	params.EndTime = time.Now().Add(time.Hour)
	_, err = createOrderConfig(params)
	assert.NoError(t, err)
	params.OrderType = order.TWAP
	_, err = createOrderConfig(params)
	assert.NoError(t, err)
	params.EndTime = time.Time{}
	_, err = createOrderConfig(params)
	assert.ErrorIs(t, err, errEndTimeInPast)
	params.OrderType = order.StopLimit
	_, err = createOrderConfig(params)
	assert.NoError(t, err)
	params.EndTime = time.Unix(1, 1)
	_, err = createOrderConfig(params)
	assert.ErrorIs(t, err, errEndTimeInPast)
	params.EndTime = time.Now().Add(time.Hour)
	_, err = createOrderConfig(params)
	assert.NoError(t, err)
	params.OrderType = order.Bracket
	_, err = createOrderConfig(params)
	assert.NoError(t, err)
	params.EndTime = time.Unix(1, 1)
	_, err = createOrderConfig(params)
	assert.ErrorIs(t, err, errEndTimeInPast)
	params.EndTime = time.Time{}
	_, err = createOrderConfig(params)
	assert.NoError(t, err)
}

func TestFormatMarginType(t *testing.T) {
	t.Parallel()
	resp := FormatMarginType("ISOLATED")
	assert.Equal(t, "ISOLATED", resp)
	resp = FormatMarginType("MULTI")
	assert.Equal(t, "CROSS", resp)
	resp = FormatMarginType("fake")
	assert.Empty(t, resp)
}

func TestStatusToStandardStatus(t *testing.T) {
	t.Parallel()
	resp, _ := statusToStandardStatus("PENDING")
	assert.Equal(t, order.New, resp)
	resp, _ = statusToStandardStatus("OPEN")
	assert.Equal(t, order.Active, resp)
	resp, _ = statusToStandardStatus("FILLED")
	assert.Equal(t, order.Filled, resp)
	resp, _ = statusToStandardStatus("CANCELLED")
	assert.Equal(t, order.Cancelled, resp)
	resp, _ = statusToStandardStatus("EXPIRED")
	assert.Equal(t, order.Expired, resp)
	resp, _ = statusToStandardStatus("FAILED")
	assert.Equal(t, order.Rejected, resp)
	_, err := statusToStandardStatus("")
	assert.ErrorIs(t, err, order.ErrUnsupportedStatusType)
}

func TestStringToStandardType(t *testing.T) {
	t.Parallel()
	resp, _ := stringToStandardType("LIMIT_ORDER_TYPE")
	assert.Equal(t, order.Limit, resp)
	resp, _ = stringToStandardType("MARKET_ORDER_TYPE")
	assert.Equal(t, order.Market, resp)
	resp, _ = stringToStandardType("STOP_LIMIT_ORDER_TYPE")
	assert.Equal(t, order.StopLimit, resp)
	_, err := stringToStandardType("")
	assert.ErrorIs(t, err, order.ErrUnrecognisedOrderType)
}

func TestStringToStandardAsset(t *testing.T) {
	t.Parallel()
	resp, _ := stringToStandardAsset("SPOT")
	assert.Equal(t, asset.Spot, resp)
	resp, _ = stringToStandardAsset("FUTURE")
	assert.Equal(t, asset.Futures, resp)
	_, err := stringToStandardAsset("")
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestStrategyDecoder(t *testing.T) {
	t.Parallel()
	resp, _ := strategyDecoder("IMMEDIATE_OR_CANCEL")
	assert.True(t, resp.Is(order.ImmediateOrCancel))
	resp, _ = strategyDecoder("FILL_OR_KILL")
	assert.True(t, resp.Is(order.FillOrKill))
	resp, _ = strategyDecoder("GOOD_UNTIL_CANCELLED")
	assert.True(t, resp.Is(order.GoodTillCancel))
	resp, _ = strategyDecoder("GOOD_UNTIL_DATE_TIME")
	assert.True(t, resp.Is(order.GoodTillDay|order.GoodTillTime))
	_, err := strategyDecoder("")
	assert.ErrorIs(t, err, errUnrecognisedStrategyType)
}

func TestProcessFundingData(t *testing.T) {
	t.Parallel()
	accHistory := []DeposWithdrData{
		{
			Type: "unknown",
		},
		{
			Type: "TRANSFER_TYPE_WITHDRAWAL",
		},
	}
	cryptoHistory := []TransactionData{
		{
			Type: "receive",
		},
		{
			Type: "send",
		},
	}
	_, err := e.processFundingData(accHistory, cryptoHistory)
	assert.ErrorIs(t, err, errUnknownTransferType)
	accHistory[0].Type = "TRANSFER_TYPE_DEPOSIT"
	resp, err := e.processFundingData(accHistory, cryptoHistory)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestChannelName(t *testing.T) {
	_, err := channelName(&subscription.Subscription{})
	assert.ErrorIs(t, err, subscription.ErrNotSupported)
	_, err = channelName(&subscription.Subscription{Channel: subscription.HeartbeatChannel})
	assert.NoError(t, err)
}

func exchangeBaseHelper(e *Exchange) error {
	if err := testexch.Setup(e); err != nil {
		return err
	}
	if apiKey != "" {
		e.SetCredentials(apiKey, apiSecret, "", "", "", "")
		e.API.AuthenticatedSupport = true
		e.API.AuthenticatedWebsocketSupport = true
	}
	return nil
}

func getINTXPortfolio(t *testing.T) string {
	t.Helper()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetAllPortfolios(t.Context(), "")
	assert.NoError(t, err)
	if len(resp) == 0 {
		t.Skip(skipInsufficientPortfolios)
	}
	var targetID string
	for i := range resp {
		if resp[i].Type == "INTX" {
			targetID = resp[i].UUID
			break
		}
	}
	if targetID == "" {
		t.Skip(skipInsufficientPortfolios)
	}
	return targetID
}

func convertTestHelper(t *testing.T) (fromAccID, toAccID string) {
	t.Helper()
	accIDs, err := e.ListAccounts(t.Context(), 250, 0)
	assert.NoError(t, err)
	if accIDs == nil || len(accIDs.Accounts) == 0 {
		t.Fatal(errExpectedNonEmpty)
	}
	for x := range accIDs.Accounts {
		if accIDs.Accounts[x].Currency == testStable {
			fromAccID = accIDs.Accounts[x].UUID
		}
		if accIDs.Accounts[x].Currency == testFiat {
			toAccID = accIDs.Accounts[x].UUID
		}
		if fromAccID != "" && toAccID != "" {
			break
		}
	}
	if fromAccID == "" || toAccID == "" {
		t.Skip(skipInsufSuitableAccs)
	}
	return fromAccID, toAccID
}

func transferTestHelper(t *testing.T, wallets *GetAllWalletsResponse) (srcWalletID, tarWalletID string) {
	t.Helper()
	var hasValidFunds bool
	for i := range wallets.Data {
		if wallets.Data[i].Currency.Code == testFiat.String() && wallets.Data[i].Balance.Amount > 10 {
			hasValidFunds = true
			srcWalletID = wallets.Data[i].ID
		}
	}
	if !hasValidFunds {
		t.Skip(skipInsufficientFunds)
	}
	pmID, err := e.ListPaymentMethods(t.Context())
	assert.NoError(t, err)
	if len(pmID) == 0 {
		t.Skip(skipPayMethodNotFound)
	}
	return srcWalletID, pmID[0].ID
}

type withdrawFiatFunc func(context.Context, *withdraw.Request) (*withdraw.ExchangeResponse, error)

func withdrawFiatFundsHelper(t *testing.T, fn withdrawFiatFunc) {
	t.Helper()
	t.Parallel()
	req := withdraw.Request{}
	_, err := fn(t.Context(), &req)
	assert.ErrorIs(t, err, common.ErrExchangeNameNotSet)
	req.Exchange = e.Name
	req.Currency = testFiat
	req.Amount = 1
	req.Type = withdraw.Fiat
	req.Fiat.Bank.Enabled = true
	req.Fiat.Bank.SupportedExchanges = "Coinbase"
	req.Fiat.Bank.SupportedCurrencies = testFiat.String()
	req.Fiat.Bank.AccountNumber = "123"
	req.Fiat.Bank.SWIFTCode = "456"
	req.Fiat.Bank.BSBNumber = "789"
	_, err = fn(t.Context(), &req)
	assert.ErrorIs(t, err, errWalletIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	req.Amount = -0.1
	req.WalletID = "meow"
	req.Fiat.Bank.BankName = "GCT's Officially Fake and Not Real Test Bank"
	_, err = fn(t.Context(), &req)
	assert.ErrorIs(t, err, errPayMethodNotFound)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	wallets, err := e.GetAllWallets(t.Context(), PaginationInp{})
	assert.NoError(t, err)
	if wallets == nil || len(wallets.Data) == 0 {
		t.Fatal(errExpectedNonEmpty)
	}
	req.WalletID = ""
	for i := range wallets.Data {
		if wallets.Data[i].Currency.Name == testFiat.String() && wallets.Data[i].Balance.Amount > testAmount*100 {
			req.WalletID = wallets.Data[i].ID
			break
		}
	}
	if req.WalletID == "" {
		t.Skip(skipInsufficientFunds)
	}
	req.Fiat.Bank.BankName = "AUD Wallet"
	resp, err := fn(t.Context(), &req)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

type getNoArgsResp interface {
	*ServerTimeV3 | []PaymentMethodData | *UserResponse | []FiatData | []CryptoData | *ServerTimeV2 | []CurrencyData | []PairVolumeData | *AllWrappedAssets
}

type getNoArgsAssertNotEmpty[G getNoArgsResp] func(context.Context) (G, error)

func testGetNoArgs[G getNoArgsResp](t *testing.T, f getNoArgsAssertNotEmpty[G]) {
	t.Helper()
	resp, err := f(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

type genConvertTestFunc func(context.Context, string, string, string) (*ConvertResponse, error)

func convertTestShared(t *testing.T, f genConvertTestFunc) {
	t.Helper()
	t.Parallel()
	_, err := f(t.Context(), "", "", "")
	assert.ErrorIs(t, err, errTransactionIDEmpty)
	_, err = f(t.Context(), "meow", "", "")
	assert.ErrorIs(t, err, errAccountIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	fromAccID, toAccID := convertTestHelper(t)
	resp, err := e.CreateConvertQuote(t.Context(), fromAccID, toAccID, "", "", 0.01)
	require.NoError(t, err)
	require.NotNil(t, resp)
	resp, err = f(t.Context(), resp.ID, fromAccID, toAccID)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

type getOneArgResp interface {
	*CurrencyData | *PairData | *ProductStats | *ProductTicker | *WrappedAsset | *WrappedAssetConversionRate
}

type getOneArgAssertNotEmpty[G getOneArgResp] func(context.Context, string) (G, error)

func testGetOneArg[G getOneArgResp](t *testing.T, f getOneArgAssertNotEmpty[G], arg string, tarErr error) {
	t.Helper()
	_, err := f(t.Context(), "")
	assert.ErrorIs(t, err, tarErr)
	resp, err := f(t.Context(), arg)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}
