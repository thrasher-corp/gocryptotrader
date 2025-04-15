package coinbasepro

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
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
	gctlog "github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your APIKeys here for better testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
	testingInSandbox        = false
)

var (
	c                = &CoinbasePro{}
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
	testPrice   = 1e+09
	testPrice2  = 1e+05

	skipPayMethodNotFound          = "no payment methods found, skipping"
	skipInsufSuitableAccs          = "insufficient suitable accounts for test, skipping"
	skipInsufficientFunds          = "insufficient funds for test, skipping"
	skipInsufficientOrders         = "insufficient orders for test, skipping"
	skipInsufficientPortfolios     = "insufficient portfolios for test, skipping"
	skipInsufficientWallets        = "insufficient wallets for test, skipping"
	skipInsufficientFundsOrWallets = "insufficient funds or wallets for test, skipping"
	skipInsufficientTransactions   = "insufficient transactions for test, skipping"

	errExpectMismatch         = "received: '%v' but expected: '%v'"
	errExpectedNonEmpty       = "expected non-empty response"
	errPortfolioNameDuplicate = `CoinbasePro unsuccessful HTTP status code: 409 raw response: {"error":"CONFLICT","error_details":"A portfolio with this name already exists.","message":"A portfolio with this name already exists."}, authenticated request failed`
	errPortTransferInsufFunds = `CoinbasePro unsuccessful HTTP status code: 429 raw response: {"error":"unknown","error_details":"[PORTFOLIO_ERROR_CODE_INSUFFICIENT_FUNDS] insufficient funds in source account","message":"[PORTFOLIO_ERROR_CODE_INSUFFICIENT_FUNDS] insufficient funds in source account"}, authenticated request failed`
	errInvalidProductID       = `CoinbasePro unsuccessful HTTP status code: 404 raw response: {"error":"NOT_FOUND","error_details":"valid product_id is required","message":"valid product_id is required"}`
	errExpectedFeeRange       = "expected fee range of %v and %v, received %v"
	errOptionInvalid          = `CoinbasePro unsuccessful HTTP status code: 400 raw response: {"error":"unknown","error_details":"parsing field \"product_type\": \"OPTIONS\" is not a valid value","message":"parsing field \"product_type\": \"OPTIONS\" is not a valid value"}`
)

func TestMain(m *testing.M) {
	c.SetDefaults()
	if testingInSandbox {
		err := c.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
			exchange.RestSpot: coinbaseproSandboxAPIURL,
		})
		if err != nil {
			log.Fatal("failed to set sandbox endpoint", err)
		}
	}
	err := exchangeBaseHelper(c)
	if err != nil {
		log.Fatal(err)
	}
	if apiKey != "" {
		c.GetBase().API.AuthenticatedSupport = true
		c.GetBase().API.AuthenticatedWebsocketSupport = true
	}
	err = gctlog.SetGlobalLogConfig(gctlog.GenDefaultSettings())
	if err != nil {
		log.Fatal(err)
	}
	var dialer gws.Dialer
	err = c.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		log.Fatal(err)
	}
	go c.wsReadData()
	os.Exit(m.Run())
}

func TestSetup(t *testing.T) {
	cfg, err := c.GetStandardConfig()
	assert.NoError(t, err)
	exch := &CoinbasePro{}
	exch.SetDefaults()
	err = exchangeBaseHelper(exch)
	require.NoError(t, err)
	cfg.ProxyAddress = string(rune(0x7f))
	err = exch.Setup(cfg)
	assert.ErrorIs(t, err, exchange.ErrSettingProxyAddress)
}

func TestWsConnect(t *testing.T) {
	exch := &CoinbasePro{}
	exch.Websocket = sharedtestvalues.NewTestWebsocket()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	err := exch.Websocket.Disable()
	assert.ErrorIs(t, err, websocket.ErrAlreadyDisabled)
	err = exch.WsConnect()
	assert.ErrorIs(t, err, websocket.ErrWebsocketNotEnabled)
	exch.SetDefaults()
	err = exchangeBaseHelper(exch)
	require.NoError(t, err)
	err = exch.Websocket.Enable()
	assert.NoError(t, err)
}

func TestGetAllAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetAllAccounts(t.Context(), 50, "")
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAccountByID(t *testing.T) {
	t.Parallel()
	_, err := c.GetAccountByID(t.Context(), "")
	assert.ErrorIs(t, err, errAccountIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	longResp, err := c.GetAllAccounts(t.Context(), 49, "")
	assert.NoError(t, err)
	if longResp == nil || len(longResp.Accounts) == 0 {
		t.Fatal(errExpectedNonEmpty)
	}
	shortResp, err := c.GetAccountByID(t.Context(), longResp.Accounts[0].UUID)
	assert.NoError(t, err)
	if *shortResp != longResp.Accounts[0] {
		t.Errorf(errExpectMismatch, shortResp, longResp.Accounts[0])
	}
}

func TestGetBestBidAsk(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	testPairs := []string{testPairFiat.String(), "ETH-USD"}
	resp, err := c.GetBestBidAsk(t.Context(), testPairs)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetProductBookV3(t *testing.T) {
	t.Parallel()
	_, err := c.GetProductBookV3(t.Context(), currency.Pair{}, 0, 0, false)
	assert.ErrorIs(t, err, errProductIDEmpty)
	resp, err := c.GetProductBookV3(t.Context(), testPairFiat, 4, -1, false)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, resp, errExpectedNonEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err = c.GetProductBookV3(t.Context(), testPairFiat, 4, -1, true)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAllProducts(t *testing.T) {
	t.Parallel()
	testPairs := []string{testPairFiat.String(), "ETH-USD"}
	resp, err := c.GetAllProducts(t.Context(), 30000, 1, "SPOT", "PERPETUAL", "STATUS_ALL", testPairs, false)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, resp, errExpectedNonEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err = c.GetAllProducts(t.Context(), 0, 1, "SPOT", "PERPETUAL", "STATUS_ALL", nil, true)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetProductByID(t *testing.T) {
	t.Parallel()
	_, err := c.GetProductByID(t.Context(), "", false)
	assert.ErrorIs(t, err, errProductIDEmpty)
	resp, err := c.GetProductByID(t.Context(), testPairFiat.String(), false)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, resp, errExpectedNonEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err = c.GetProductByID(t.Context(), testPairFiat.String(), true)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetHistoricRates(t *testing.T) {
	t.Parallel()
	_, err := c.GetHistoricKlines(t.Context(), "", granUnknown, time.Time{}, time.Time{}, false)
	assert.ErrorIs(t, err, errProductIDEmpty)
	_, err = c.GetHistoricKlines(t.Context(), testPairFiat.String(), "blorbo", time.Time{}, time.Time{}, false)
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval)
	resp, err := c.GetHistoricKlines(t.Context(), testPairFiat.String(), granOneMin, time.Now().Add(-5*time.Minute), time.Now(), false)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, resp, errExpectedNonEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err = c.GetHistoricKlines(t.Context(), testPairFiat.String(), granOneMin, time.Now().Add(-5*time.Minute), time.Now(), true)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := c.GetTicker(t.Context(), "", 1, time.Time{}, time.Time{}, false)
	assert.ErrorIs(t, err, errProductIDEmpty)
	resp, err := c.GetTicker(t.Context(), testPairFiat.String(), 5, time.Now().Add(-time.Minute*5), time.Now(), false)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, resp, errExpectedNonEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err = c.GetTicker(t.Context(), testPairFiat.String(), 5, time.Now().Add(-time.Minute*5), time.Now(), true)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestPlaceOrder(t *testing.T) {
	t.Parallel()
	ord := &PlaceOrderInfo{}
	_, err := c.PlaceOrder(t.Context(), ord)
	assert.ErrorIs(t, err, errClientOrderIDEmpty)
	ord.ClientOID = "meow"
	_, err = c.PlaceOrder(t.Context(), ord)
	assert.ErrorIs(t, err, errProductIDEmpty)
	ord.ProductID = testPairFiat.String()
	_, err = c.PlaceOrder(t.Context(), ord)
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid)
	ord.BaseAmount = testAmount
	_, err = c.PlaceOrder(t.Context(), ord)
	assert.ErrorIs(t, err, errInvalidOrderType)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	id, err := uuid.NewV4()
	assert.NoError(t, err)
	ord = &PlaceOrderInfo{
		ClientOID:  id.String(),
		ProductID:  testPairStable.String(),
		Side:       order.Buy.String(),
		OrderType:  order.Limit.String(),
		MarginType: "CROSS",
		BaseAmount: testAmount,
		LimitPrice: testPrice2,
		Leverage:   9999,
		PostOnly:   false,
		EndTime:    time.Now().Add(time.Hour),
	}
	resp, err := c.PlaceOrder(t.Context(), ord)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, resp, errExpectedNonEmpty)
	}
	id, err = uuid.NewV4()
	assert.NoError(t, err)
	ord.ClientOID = id.String()
	ord.MarginType = "MULTI"
	resp, err = c.PlaceOrder(t.Context(), ord)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestCancelOrders(t *testing.T) {
	t.Parallel()
	var orderSlice []string
	_, err := c.CancelOrders(t.Context(), orderSlice)
	assert.ErrorIs(t, err, errOrderIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	orderSlice = append(orderSlice, "1")
	resp, err := c.CancelOrders(t.Context(), orderSlice)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestEditOrder(t *testing.T) {
	t.Parallel()
	_, err := c.EditOrder(t.Context(), "", 0, 0)
	assert.ErrorIs(t, err, errOrderIDEmpty)
	_, err = c.EditOrder(t.Context(), "meow", 0, 0)
	assert.ErrorIs(t, err, errSizeAndPriceZero)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	resp, err := c.EditOrder(t.Context(), "1", testAmount, testPrice2-1)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestEditOrderPreview(t *testing.T) {
	t.Parallel()
	_, err := c.EditOrderPreview(t.Context(), "", 0, 0)
	assert.ErrorIs(t, err, errOrderIDEmpty)
	_, err = c.EditOrderPreview(t.Context(), "meow", 0, 0)
	assert.ErrorIs(t, err, errSizeAndPriceZero)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.EditOrderPreview(t.Context(), "1", testAmount, testPrice2+2)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAllOrders(t *testing.T) {
	t.Parallel()
	assets := []string{testFiat.String()}
	status := make([]string, 2)
	_, err := c.GetAllOrders(t.Context(), "", "", "", "", "", "", "", "", "", status, assets, 0, time.Unix(2, 2), time.Unix(1, 1))
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	status[0] = "CANCELLED"
	status[1] = "OPEN"
	_, err = c.GetAllOrders(t.Context(), "", "", "", "", "", "", "", "", "", status, assets, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errOpenPairWithOtherTypes)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	status = make([]string, 0)
	assets = make([]string, 1)
	assets[0] = testCrypto.String()
	_, err = c.GetAllOrders(t.Context(), "", testFiat.String(), "LIMIT", "SELL", "", "SPOT", "RETAIL_ADVANCED", "UNKNOWN_CONTRACT_EXPIRY_TYPE", "", status, assets, 10, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetFills(t *testing.T) {
	t.Parallel()
	_, err := c.GetFills(t.Context(), "", "", "", time.Unix(2, 2), time.Unix(1, 1), 0)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err = c.GetFills(t.Context(), "", testPairStable.String(), "", time.Unix(1, 1), time.Now(), 5)
	assert.NoError(t, err)
	status := []string{"OPEN"}
	ordID, err := c.GetAllOrders(t.Context(), "", "", "", "", "", "", "", "", "", status, nil, 3, time.Time{}, time.Time{})
	assert.NoError(t, err)
	if ordID == nil || len(ordID.Orders) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = c.GetFills(t.Context(), ordID.Orders[0].OrderID, "", "", time.Time{}, time.Time{}, 5)
	assert.NoError(t, err)
}

func TestGetOrderByID(t *testing.T) {
	t.Parallel()
	_, err := c.GetOrderByID(t.Context(), "", "", "")
	assert.ErrorIs(t, err, errOrderIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	ordID, err := c.GetAllOrders(t.Context(), "", "", "", "", "", "", "", "", "", nil, nil, 10, time.Time{}, time.Time{})
	assert.NoError(t, err)
	if ordID == nil || len(ordID.Orders) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	resp, err := c.GetOrderByID(t.Context(), ordID.Orders[0].OrderID, ordID.Orders[0].ClientOID, testFiat.String())
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestPreviewOrder(t *testing.T) {
	t.Parallel()
	inf := &PreviewOrderInfo{}
	_, err := c.PreviewOrder(t.Context(), inf)
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid)
	inf.BaseAmount = 1
	_, err = c.PreviewOrder(t.Context(), inf)
	assert.ErrorIs(t, err, errInvalidOrderType)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	inf.ProductID = testPairStable.String()
	inf.Side = "BUY"
	inf.OrderType = "MARKET"
	inf.MarginType = "ISOLATED"
	inf.BaseAmount = testAmount
	resp, err := c.PreviewOrder(t.Context(), inf)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAllPortfolios(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetAllPortfolios(t.Context(), "DEFAULT")
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestCreatePortfolio(t *testing.T) {
	t.Parallel()
	_, err := c.CreatePortfolio(t.Context(), "")
	assert.ErrorIs(t, err, errNameEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	_, err = c.CreatePortfolio(t.Context(), "GCT Test Portfolio")
	assert.NoError(t, err)
}

func TestMovePortfolioFunds(t *testing.T) {
	t.Parallel()
	_, err := c.MovePortfolioFunds(t.Context(), "", "", "", 0)
	assert.ErrorIs(t, err, errPortfolioIDEmpty)
	_, err = c.MovePortfolioFunds(t.Context(), "", "meowPort", "woofPort", 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = c.MovePortfolioFunds(t.Context(), testCrypto.String(), "meowPort", "woofPort", 0)
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	portID, err := c.GetAllPortfolios(t.Context(), "")
	assert.NoError(t, err)
	if len(portID) < 2 {
		t.Skip(skipInsufficientPortfolios)
	}
	_, err = c.MovePortfolioFunds(t.Context(), testCrypto.String(), portID[0].UUID, portID[1].UUID, testAmount)
	assert.NoError(t, err)
}

func TestGetPortfolioByID(t *testing.T) {
	t.Parallel()
	_, err := c.GetPortfolioByID(t.Context(), "")
	assert.ErrorIs(t, err, errPortfolioIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	portID, err := c.GetAllPortfolios(t.Context(), "")
	assert.NoError(t, err)
	if len(portID) == 0 {
		t.Fatal(errExpectedNonEmpty)
	}
	resp, err := c.GetPortfolioByID(t.Context(), portID[0].UUID)
	assert.NoError(t, err)
	if resp.Portfolio != portID[0] {
		t.Errorf(errExpectMismatch, resp.Portfolio, portID[0])
	}
}

func TestDeletePortfolio(t *testing.T) {
	t.Parallel()
	err := c.DeletePortfolio(t.Context(), "")
	assert.ErrorIs(t, err, errPortfolioIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	err = c.DeletePortfolio(t.Context(), "insert portfolio ID here")
	// The new JWT-based keys don't have permissions to delete portfolios they aren't assigned to, causing this to fail
	assert.NoError(t, err)
}

func TestEditPortfolio(t *testing.T) {
	t.Parallel()
	_, err := c.EditPortfolio(t.Context(), "", "")
	assert.ErrorIs(t, err, errPortfolioIDEmpty)
	_, err = c.EditPortfolio(t.Context(), "meow", "")
	assert.ErrorIs(t, err, errNameEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	_, err = c.EditPortfolio(t.Context(), "insert portfolio ID here", "GCT Test Portfolio Edited")
	// The new JWT-based keys don't have permissions to edit portfolios they aren't assigned to, causing this to fail
	assert.NoError(t, err)
}

func TestGetFuturesBalanceSummary(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetFuturesBalanceSummary(t.Context())
	assert.NoError(t, err)
}

func TestGetAllFuturesPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetAllFuturesPositions(t.Context())
	assert.NoError(t, err)
}

func TestGetFuturesPositionByID(t *testing.T) {
	t.Parallel()
	_, err := c.GetFuturesPositionByID(t.Context(), "")
	assert.ErrorIs(t, err, errProductIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err = c.GetFuturesPositionByID(t.Context(), "meow")
	assert.NoError(t, err)
}

func TestListFuturesSweeps(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.ListFuturesSweeps(t.Context())
	assert.NoError(t, err)
}

func TestAllocatePortfolio(t *testing.T) {
	t.Parallel()
	err := c.AllocatePortfolio(t.Context(), "", "", "", 0)
	assert.ErrorIs(t, err, errPortfolioIDEmpty)
	err = c.AllocatePortfolio(t.Context(), "meow", "", "", 0)
	assert.ErrorIs(t, err, errProductIDEmpty)
	err = c.AllocatePortfolio(t.Context(), "meow", "bark", "", 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	err = c.AllocatePortfolio(t.Context(), "meow", "bark", "woof", 0)
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	pID := getINTXPortfolio(t)
	err = c.AllocatePortfolio(t.Context(), pID, testCrypto.String(), testFiat.String(), 0.001337)
	assert.NoError(t, err)
}

func TestGetPerpetualsPortfolioSummary(t *testing.T) {
	t.Parallel()
	_, err := c.GetPerpetualsPortfolioSummary(t.Context(), "")
	assert.ErrorIs(t, err, errPortfolioIDEmpty)
	pID := getINTXPortfolio(t)
	resp, err := c.GetPerpetualsPortfolioSummary(t.Context(), pID)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAllPerpetualsPositions(t *testing.T) {
	t.Parallel()
	_, err := c.GetAllPerpetualsPositions(t.Context(), "")
	assert.ErrorIs(t, err, errPortfolioIDEmpty)
	pID := getINTXPortfolio(t)
	_, err = c.GetAllPerpetualsPositions(t.Context(), pID)
	assert.NoError(t, err)
}

func TestGetPerpetualsPositionByID(t *testing.T) {
	t.Parallel()
	_, err := c.GetPerpetualsPositionByID(t.Context(), "", "")
	assert.ErrorIs(t, err, errPortfolioIDEmpty)
	_, err = c.GetPerpetualsPositionByID(t.Context(), "meow", "")
	assert.ErrorIs(t, err, errProductIDEmpty)
	pID := getINTXPortfolio(t)
	_, err = c.GetPerpetualsPositionByID(t.Context(), pID, testPairFiat.String())
	assert.NoError(t, err)
}

func TestGetTransactionSummary(t *testing.T) {
	t.Parallel()
	_, err := c.GetTransactionSummary(t.Context(), time.Unix(2, 2), time.Unix(1, 1), "", "", "")
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetTransactionSummary(t.Context(), time.Unix(1, 1), time.Now(), testFiat.String(), asset.Spot.Upper(), "UNKNOWN_CONTRACT_EXPIRY_TYPE")
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestCreateConvertQuote(t *testing.T) {
	t.Parallel()
	_, err := c.CreateConvertQuote(t.Context(), "", "", "", "", 0)
	assert.ErrorIs(t, err, errAccountIDEmpty)
	_, err = c.CreateConvertQuote(t.Context(), "meow", "123", "", "", 0)
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	fromAccID, toAccID := convertTestHelper(t)
	resp, err := c.CreateConvertQuote(t.Context(), fromAccID, toAccID, "", "", 0.01)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestCommitConvertTrade(t *testing.T) {
	convertTestShared(t, c.CommitConvertTrade)
}

func TestGetConvertTradeByID(t *testing.T) {
	convertTestShared(t, c.GetConvertTradeByID)
}

func TestGetV3Time(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, c.GetV3Time)
}

func TestGetAllPaymentMethods(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	testGetNoArgs(t, c.GetAllPaymentMethods)
}

func TestGetPaymentMethodByID(t *testing.T) {
	t.Parallel()
	_, err := c.GetPaymentMethodByID(t.Context(), "")
	assert.ErrorIs(t, err, errPaymentMethodEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	pmID, err := c.GetAllPaymentMethods(t.Context())
	assert.NoError(t, err)
	if len(pmID) == 0 {
		t.Skip(skipPayMethodNotFound)
	}
	resp, err := c.GetPaymentMethodByID(t.Context(), pmID[0].ID)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetCurrentUser(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	// This intermittently fails with the message "Unauthorized", for no clear reason
	testGetNoArgs(t, c.GetCurrentUser)
}

func TestGetAllWallets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	pagIn := PaginationInp{Limit: 2}
	resp, err := c.GetAllWallets(t.Context(), pagIn)
	assert.NoError(t, err)
	require.NotEmpty(t, resp, errExpectedNonEmpty)
	if resp.Pagination.NextStartingAfter == "" {
		t.Skip(skipInsufficientWallets)
	}
	pagIn.StartingAfter = resp.Pagination.NextStartingAfter
	resp, err = c.GetAllWallets(t.Context(), pagIn)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetWalletByID(t *testing.T) {
	t.Parallel()
	_, err := c.GetWalletByID(t.Context(), "", "")
	assert.ErrorIs(t, err, errCurrWalletConflict)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetWalletByID(t.Context(), "", testCrypto.String())
	require.NoError(t, err)
	require.NotEmpty(t, resp, errExpectedNonEmpty)
	resp, err = c.GetWalletByID(t.Context(), resp.ID, "")
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestCreateAddress(t *testing.T) {
	t.Parallel()
	_, err := c.CreateAddress(t.Context(), "", "")
	assert.ErrorIs(t, err, errWalletIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	wID, err := c.GetWalletByID(t.Context(), "", testCrypto.String())
	require.NoError(t, err)
	require.NotEmpty(t, wID, errExpectedNonEmpty)
	resp, err := c.CreateAddress(t.Context(), wID.ID, "")
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAllAddresses(t *testing.T) {
	t.Parallel()
	var pag PaginationInp
	_, err := c.GetAllAddresses(t.Context(), "", pag)
	assert.ErrorIs(t, err, errWalletIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	wID, err := c.GetWalletByID(t.Context(), "", testCrypto.String())
	require.NoError(t, err)
	require.NotEmpty(t, wID, errExpectedNonEmpty)
	resp, err := c.GetAllAddresses(t.Context(), wID.ID, pag)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAddressByID(t *testing.T) {
	t.Parallel()
	_, err := c.GetAddressByID(t.Context(), "", "")
	assert.ErrorIs(t, err, errWalletIDEmpty)
	_, err = c.GetAddressByID(t.Context(), "123", "")
	assert.ErrorIs(t, err, errAddressIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	wID, err := c.GetWalletByID(t.Context(), "", testCrypto.String())
	require.NoError(t, err)
	require.NotEmpty(t, wID, errExpectedNonEmpty)
	addID, err := c.GetAllAddresses(t.Context(), wID.ID, PaginationInp{})
	require.NoError(t, err)
	require.NotEmpty(t, addID, errExpectedNonEmpty)
	resp, err := c.GetAddressByID(t.Context(), wID.ID, addID.Data[0].ID)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAddressTransactions(t *testing.T) {
	t.Parallel()
	_, err := c.GetAddressTransactions(t.Context(), "", "", PaginationInp{})
	assert.ErrorIs(t, err, errWalletIDEmpty)
	_, err = c.GetAddressTransactions(t.Context(), "123", "", PaginationInp{})
	assert.ErrorIs(t, err, errAddressIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	wID, err := c.GetWalletByID(t.Context(), "", testCrypto.String())
	require.NoError(t, err)
	require.NotEmpty(t, wID, errExpectedNonEmpty)
	addID, err := c.GetAllAddresses(t.Context(), wID.ID, PaginationInp{})
	require.NoError(t, err)
	require.NotEmpty(t, addID, errExpectedNonEmpty)
	_, err = c.GetAddressTransactions(t.Context(), wID.ID, addID.Data[0].ID, PaginationInp{})
	assert.NoError(t, err)
}

func TestSendMoney(t *testing.T) {
	t.Parallel()
	_, err := c.SendMoney(t.Context(), "", "", "", "", "", "", "", "", 0, false, false)
	assert.ErrorIs(t, err, errTransactionTypeEmpty)
	_, err = c.SendMoney(t.Context(), "123", "", "", "", "", "", "", "", 0, false, false)
	assert.ErrorIs(t, err, errWalletIDEmpty)
	_, err = c.SendMoney(t.Context(), "123", "123", "", "", "", "", "", "", 0, false, false)
	assert.ErrorIs(t, err, errToEmpty)
	_, err = c.SendMoney(t.Context(), "123", "123", "123", "", "", "", "", "", 0, false, false)
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid)
	_, err = c.SendMoney(t.Context(), "123", "123", "123", "", "", "", "", "", 1, false, false)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	wID, err := c.GetAllWallets(t.Context(), PaginationInp{})
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
	resp, err := c.SendMoney(t.Context(), "transfer", wID.Data[0].ID, wID.Data[1].ID, testCrypto.String(), "GCT Test", "123", "", "", testAmount, false, false)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAllTransactions(t *testing.T) {
	t.Parallel()
	var pag PaginationInp
	_, err := c.GetAllTransactions(t.Context(), "", pag)
	assert.ErrorIs(t, err, errWalletIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	wID, err := c.GetWalletByID(t.Context(), "", testCrypto.String())
	require.NoError(t, err)
	require.NotEmpty(t, wID, errExpectedNonEmpty)
	_, err = c.GetAllTransactions(t.Context(), wID.ID, pag)
	assert.NoError(t, err)
}

func TestGetTransactionByID(t *testing.T) {
	t.Parallel()
	_, err := c.GetTransactionByID(t.Context(), "", "")
	assert.ErrorIs(t, err, errWalletIDEmpty)
	_, err = c.GetTransactionByID(t.Context(), "123", "")
	assert.ErrorIs(t, err, errTransactionIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	wID, err := c.GetWalletByID(t.Context(), "", testCrypto.String())
	require.NoError(t, err)
	require.NotEmpty(t, wID, errExpectedNonEmpty)
	tID, err := c.GetAllTransactions(t.Context(), wID.ID, PaginationInp{})
	assert.NoError(t, err)
	if tID == nil || len(tID.Data) == 0 {
		t.Skip(skipInsufficientTransactions)
	}
	resp, err := c.GetTransactionByID(t.Context(), wID.ID, tID.Data[0].ID)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestFiatTransfer(t *testing.T) {
	t.Parallel()
	_, err := c.FiatTransfer(t.Context(), "", "", "", 0, false, FiatDeposit)
	assert.ErrorIs(t, err, errWalletIDEmpty)
	_, err = c.FiatTransfer(t.Context(), "123", "", "", 0, false, FiatDeposit)
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid)
	_, err = c.FiatTransfer(t.Context(), "123", "", "", 1, false, FiatDeposit)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = c.FiatTransfer(t.Context(), "123", "123", "", 1, false, FiatDeposit)
	assert.ErrorIs(t, err, errPaymentMethodEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	wallets, err := c.GetAllWallets(t.Context(), PaginationInp{})
	require.NoError(t, err)
	assert.NotEmpty(t, wallets, errExpectedNonEmpty)
	wID, pmID := transferTestHelper(t, wallets)
	resp, err := c.FiatTransfer(t.Context(), wID, testFiat.String(), pmID, testAmount, false, FiatDeposit)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, resp, errExpectedNonEmpty)
	}
	resp, err = c.FiatTransfer(t.Context(), wID, testFiat.String(), pmID, testAmount, false, FiatWithdrawal)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestCommitTransfer(t *testing.T) {
	t.Parallel()
	_, err := c.CommitTransfer(t.Context(), "", "", FiatDeposit)
	assert.ErrorIs(t, err, errWalletIDEmpty)
	_, err = c.CommitTransfer(t.Context(), "123", "", FiatDeposit)
	assert.ErrorIs(t, err, errDepositIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	wallets, err := c.GetAllWallets(t.Context(), PaginationInp{})
	require.NoError(t, err)
	assert.NotEmpty(t, wallets, errExpectedNonEmpty)
	wID, pmID := transferTestHelper(t, wallets)
	depID, err := c.FiatTransfer(t.Context(), wID, testFiat.String(), pmID, testAmount, false, FiatDeposit)
	require.NoError(t, err)
	resp, err := c.CommitTransfer(t.Context(), wID, depID.ID, FiatDeposit)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, resp, errExpectedNonEmpty)
	}
	depID, err = c.FiatTransfer(t.Context(), wID, testFiat.String(), pmID, testAmount, false, FiatWithdrawal)
	require.NoError(t, err)
	resp, err = c.CommitTransfer(t.Context(), wID, depID.ID, FiatWithdrawal)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAllFiatTransfers(t *testing.T) {
	t.Parallel()
	var pag PaginationInp
	_, err := c.GetAllFiatTransfers(t.Context(), "", pag, FiatDeposit)
	assert.ErrorIs(t, err, errWalletIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	wID, err := c.GetWalletByID(t.Context(), "", "AUD")
	require.NoError(t, err)
	assert.NotEmpty(t, wID, errExpectedNonEmpty)
	// Fiat deposits/withdrawals aren't accepted for fiat currencies for Australian business accounts; the error
	// "id not found" possibly reflects this
	_, err = c.GetAllFiatTransfers(t.Context(), wID.ID, pag, FiatDeposit)
	assert.NoError(t, err)
	_, err = c.GetAllFiatTransfers(t.Context(), wID.ID, pag, FiatWithdrawal)
	assert.NoError(t, err)
}

func TestGetFiatTransferByID(t *testing.T) {
	t.Parallel()
	_, err := c.GetFiatTransferByID(t.Context(), "", "", FiatDeposit)
	assert.ErrorIs(t, err, errWalletIDEmpty)
	_, err = c.GetFiatTransferByID(t.Context(), "123", "", FiatDeposit)
	assert.ErrorIs(t, err, errDepositIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	wID, err := c.GetWalletByID(t.Context(), "", "AUD")
	require.NoError(t, err)
	assert.NotEmpty(t, wID, errExpectedNonEmpty)
	// Fiat deposits/withdrawals aren't accepted for fiat currencies for Australian business accounts; the error
	// "id not found" possibly reflects this
	dID, err := c.GetAllFiatTransfers(t.Context(), wID.ID, PaginationInp{}, FiatDeposit)
	assert.NoError(t, err)
	if dID == nil || len(dID.Data) == 0 {
		t.Skip(skipInsufficientTransactions)
	}
	resp, err := c.GetFiatTransferByID(t.Context(), wID.ID, dID.Data[0].ID, FiatDeposit)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, resp, errExpectedNonEmpty)
	}
	resp, err = c.GetFiatTransferByID(t.Context(), wID.ID, dID.Data[0].ID, FiatWithdrawal)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetFiatCurrencies(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, c.GetFiatCurrencies)
}

func TestGetCryptocurrencies(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, c.GetCryptocurrencies)
}

func TestGetExchangeRates(t *testing.T) {
	t.Parallel()
	resp, err := c.GetExchangeRates(t.Context(), "")
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetPrice(t *testing.T) {
	t.Parallel()
	_, err := c.GetPrice(t.Context(), "", "")
	assert.ErrorIs(t, err, errInvalidPriceType)
	resp, err := c.GetPrice(t.Context(), testPairFiat.String(), asset.Spot.String())
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetV2Time(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, c.GetV2Time)
}

func TestGetAllCurrencies(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, c.GetAllCurrencies)
}

func TestGetACurrency(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, c.GetACurrency, testCrypto.String(), currency.ErrCurrencyCodeEmpty)
}

func TestGetAllTradingPairs(t *testing.T) {
	t.Parallel()
	_, err := c.GetAllTradingPairs(t.Context(), "")
	assert.NoError(t, err)
}

func TestGetAllPairVolumes(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, c.GetAllPairVolumes)
}

func TestGetPairDetails(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, c.GetPairDetails, testPairFiat.String(), currency.ErrCurrencyPairEmpty)
}

func TestGetProductBookV1(t *testing.T) {
	t.Parallel()
	_, err := c.GetProductBookV1(t.Context(), "", 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	resp, err := c.GetProductBookV1(t.Context(), testPairFiat.String(), 2)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, resp, errExpectedNonEmpty)
	}
	resp, err = c.GetProductBookV1(t.Context(), testPairFiat.String(), 3)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetProductCandles(t *testing.T) {
	t.Parallel()
	_, err := c.GetProductCandles(t.Context(), "", 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	resp, err := c.GetProductCandles(t.Context(), testPairFiat.String(), 300, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetProductStats(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, c.GetProductStats, testPairFiat.String(), currency.ErrCurrencyPairEmpty)
}

func TestGetProductTicker(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, c.GetProductTicker, testPairFiat.String(), currency.ErrCurrencyPairEmpty)
}

func TestGetProductTrades(t *testing.T) {
	t.Parallel()
	_, err := c.GetProductTrades(t.Context(), "", "", "", 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	resp, err := c.GetProductTrades(t.Context(), testPairFiat.String(), "1", "before", 0)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAllWrappedAssets(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, c.GetAllWrappedAssets)
}

func TestGetWrappedAssetDetails(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, c.GetWrappedAssetDetails, testWrappedAsset.String(), errWrappedAssetEmpty)
}

func TestGetWrappedAssetConversionRate(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, c.GetWrappedAssetConversionRate, testWrappedAsset.String(), errWrappedAssetEmpty)
}

func TestSendHTTPRequest(t *testing.T) {
	t.Parallel()
	err := c.SendHTTPRequest(t.Context(), exchange.EdgeCase3, "", nil, nil)
	assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound)
}

func TestSendAuthenticatedHTTPRequest(t *testing.T) {
	t.Parallel()
	err := c.SendAuthenticatedHTTPRequest(t.Context(), exchange.EdgeCase3, "", "", nil, nil, false, nil)
	assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound)
	ch := make(chan struct{})
	body := map[string]any{"Unmarshalable": ch}
	err = c.SendAuthenticatedHTTPRequest(t.Context(), exchange.RestSpot, "", "", nil, body, false, nil)
	// TODO: Implement this more rigorously once thrasher investigates the code further
	// var targetErr *json.UnsupportedTypeError
	// assert.ErrorAs(t, err, &targetErr)
	assert.ErrorContains(t, err, "json: unsupported type: chan struct {}")
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	_, err := c.GetFee(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)
	feeBuilder := exchange.FeeBuilder{
		FeeType:       exchange.OfflineTradeFee,
		Amount:        1,
		PurchasePrice: 1,
	}
	resp, err := c.GetFee(t.Context(), &feeBuilder)
	assert.NoError(t, err)
	if resp != WorstCaseTakerFee {
		t.Errorf(errExpectMismatch, resp, WorstCaseTakerFee)
	}
	feeBuilder.IsMaker = true
	resp, err = c.GetFee(t.Context(), &feeBuilder)
	assert.NoError(t, err)
	if resp != WorstCaseMakerFee {
		t.Errorf(errExpectMismatch, resp, WorstCaseMakerFee)
	}
	feeBuilder.Pair = currency.NewPair(currency.USDT, currency.USD)
	resp, err = c.GetFee(t.Context(), &feeBuilder)
	assert.NoError(t, err)
	if resp != 0 {
		t.Errorf(errExpectMismatch, resp, StablePairMakerFee)
	}
	feeBuilder.IsMaker = false
	resp, err = c.GetFee(t.Context(), &feeBuilder)
	assert.NoError(t, err)
	if resp != WorstCaseStablePairTakerFee {
		t.Errorf(errExpectMismatch, resp, WorstCaseStablePairTakerFee)
	}
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	_, err = c.GetFee(t.Context(), &feeBuilder)
	assert.ErrorIs(t, err, errFeeTypeNotSupported)
	feeBuilder.Pair = currency.Pair{}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	feeBuilder.FeeType = exchange.CryptocurrencyTradeFee
	resp, err = c.GetFee(t.Context(), &feeBuilder)
	assert.NoError(t, err)
	if !(resp <= WorstCaseTakerFee && resp >= BestCaseTakerFee) {
		t.Errorf(errExpectedFeeRange, BestCaseTakerFee, WorstCaseTakerFee, resp)
	}
	feeBuilder.IsMaker = true
	resp, err = c.GetFee(t.Context(), &feeBuilder)
	assert.NoError(t, err)
	if !(resp <= WorstCaseMakerFee && resp >= BestCaseMakerFee) {
		t.Errorf(errExpectedFeeRange, BestCaseMakerFee, WorstCaseMakerFee, resp)
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := c.FetchTradablePairs(t.Context(), asset.Options)
	assert.Equal(t, errOptionInvalid, err.Error())
	resp, err := c.FetchTradablePairs(t.Context(), asset.Spot)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, resp, errExpectedNonEmpty)
	}
	resp, err = c.FetchTradablePairs(t.Context(), asset.Futures)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, c)
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.UpdateAccountInfo(t.Context(), asset.Spot)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	_, err := c.UpdateTicker(t.Context(), currency.Pair{}, asset.Spot)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	resp, err := c.UpdateTicker(t.Context(), testPairFiat, asset.Spot)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

// Not parallel; being parallel causes intermittent errors with another test for no discernible reason
func TestUpdateOrderbook(t *testing.T) {
	testexch.UpdatePairsOnce(t, c)
	_, err := c.UpdateOrderbook(t.Context(), currency.Pair{}, asset.Empty)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = c.UpdateOrderbook(t.Context(), testPairFiat, asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	_, err = c.UpdateOrderbook(t.Context(), currency.NewPairWithDelimiter("meow", "woof", "-"), asset.Spot)
	assert.Equal(t, errInvalidProductID, err.Error())
	// There are no perpetual futures contracts, so I can only deterministically test spot
	resp, err := c.UpdateOrderbook(t.Context(), testPairFiat, asset.Spot)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetAccountFundingHistory(t.Context())
	assert.NoError(t, err)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetWithdrawalsHistory(t.Context(), currency.NewCode("meow"), asset.Spot)
	assert.ErrorIs(t, err, errNoMatchingWallets)
	_, err = c.GetWithdrawalsHistory(t.Context(), testCrypto, asset.Spot)
	assert.NoError(t, err)
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := c.SubmitOrder(t.Context(), nil)
	assert.ErrorIs(t, err, order.ErrSubmissionIsNil)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	ord := order.Submit{
		Exchange:      c.Name,
		Pair:          testPairStable,
		AssetType:     asset.Spot,
		Side:          order.Buy,
		Type:          order.Market,
		StopDirection: order.StopUp,
		Amount:        testAmount,
		Price:         testAmount,
		RetrieveFees:  true,
		ClientOrderID: strconv.FormatInt(time.Now().UnixMilli(), 18) + "GCTSubmitOrderTest",
	}
	resp, err := c.SubmitOrder(t.Context(), &ord)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, resp, errExpectedNonEmpty)
	}
	ord.StopDirection = order.StopDown
	resp, err = c.SubmitOrder(t.Context(), &ord)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, resp, errExpectedNonEmpty)
	}
	ord.Type = order.Market
	ord.QuoteAmount = testAmount
	resp, err = c.SubmitOrder(t.Context(), &ord)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := c.ModifyOrder(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)
	var ord order.Modify
	_, err = c.ModifyOrder(t.Context(), &ord)
	assert.ErrorIs(t, err, order.ErrPairIsEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	ord.OrderID = "a"
	ord.Price = testPrice2 + 1
	ord.Amount = testAmount
	ord.Pair = testPairStable
	ord.AssetType = asset.Spot
	resp2, err := c.ModifyOrder(t.Context(), &ord)
	require.NoError(t, err)
	assert.NotEmpty(t, resp2, errExpectedNonEmpty)
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	err := c.CancelOrder(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)
	var can order.Cancel
	err = c.CancelOrder(t.Context(), &can)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	can.OrderID = "0"
	err = c.CancelOrder(t.Context(), &can)
	assert.Error(t, err)
	can.OrderID = "2"
	err = c.CancelOrder(t.Context(), &can)
	assert.NoError(t, err)
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := c.CancelBatchOrders(t.Context(), nil)
	assert.ErrorIs(t, err, errOrderIDEmpty)
	can := make([]order.Cancel, 1)
	_, err = c.CancelBatchOrders(t.Context(), can)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	can[0].OrderID = "1"
	resp2, err := c.CancelBatchOrders(t.Context(), can)
	require.NoError(t, err)
	assert.NotEmpty(t, resp2, errExpectedNonEmpty)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	ordID, err := c.GetAllOrders(t.Context(), testPairStable.String(), "", "", "", "", asset.Spot.Upper(), "", "", "", nil, nil, 2, time.Time{}, time.Now())
	require.NoError(t, err)
	if ordID == nil || len(ordID.Orders) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	resp, err := c.GetOrderInfo(t.Context(), ordID.Orders[0].OrderID, testPairStable, asset.Spot)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetDepositAddress(t.Context(), currency.NewCode("fake currency that doesn't exist"), "", "")
	assert.ErrorIs(t, err, errNoWalletForCurrency)
	resp, err := c.GetDepositAddress(t.Context(), testCrypto, "", "")
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	req := withdraw.Request{}
	_, err := c.WithdrawCryptocurrencyFunds(t.Context(), &req)
	assert.ErrorIs(t, err, common.ErrExchangeNameUnset)
	req.Exchange = c.Name
	req.Currency = testCrypto
	req.Amount = testAmount
	req.Type = withdraw.Crypto
	req.Crypto.Address = testAddress
	_, err = c.WithdrawCryptocurrencyFunds(t.Context(), &req)
	assert.ErrorIs(t, err, errWalletIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	wallets, err := c.GetAllWallets(t.Context(), PaginationInp{})
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
	resp, err := c.WithdrawCryptocurrencyFunds(t.Context(), &req)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestWithdrawFiatFunds(t *testing.T) {
	withdrawFiatFundsHelper(t, c.WithdrawFiatFunds)
}

func TestWithdrawFiatFundsToInternationalBank(t *testing.T) {
	withdrawFiatFundsHelper(t, c.WithdrawFiatFundsToInternationalBank)
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	_, err := c.GetFeeByType(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)
	var feeBuilder exchange.FeeBuilder
	feeBuilder.FeeType = exchange.OfflineTradeFee
	feeBuilder.Amount = 1
	feeBuilder.PurchasePrice = 1
	resp, err := c.GetFeeByType(t.Context(), &feeBuilder)
	assert.NoError(t, err)
	if resp != WorstCaseTakerFee {
		t.Errorf(errExpectMismatch, resp, WorstCaseTakerFee)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	_, err := c.GetActiveOrders(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)
	var req order.MultiOrderRequest
	_, err = c.GetActiveOrders(t.Context(), &req)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	req.AssetType = asset.Spot
	req.Side = order.AnySide
	req.Type = order.AnyType
	_, err = c.GetActiveOrders(t.Context(), &req)
	assert.NoError(t, err)
	req.Pairs = req.Pairs.Add(currency.NewPair(testCrypto, testFiat))
	_, err = c.GetActiveOrders(t.Context(), &req)
	assert.NoError(t, err)
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := c.GetOrderHistory(t.Context(), nil)
	assert.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	var req order.MultiOrderRequest
	req.AssetType = asset.Spot
	req.Side = order.AnySide
	req.Type = order.AnyType
	_, err = c.GetOrderHistory(t.Context(), &req)
	assert.NoError(t, err)
	req.Pairs = req.Pairs.Add(testPairStable)
	_, err = c.GetOrderHistory(t.Context(), &req)
	assert.NoError(t, err)
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	_, err := c.GetHistoricCandles(t.Context(), currency.Pair{}, asset.Empty, kline.OneYear, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	resp, err := c.GetHistoricCandles(t.Context(), testPairFiat, asset.Spot, kline.SixHour, time.Now().Add(-time.Hour*60), time.Now())
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	_, err := c.GetHistoricCandlesExtended(t.Context(), currency.Pair{}, asset.Empty, kline.OneYear, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	resp, err := c.GetHistoricCandlesExtended(t.Context(), testPairFiat, asset.Spot, kline.OneMin, time.Now().Add(-time.Hour*9), time.Now())
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	err := c.ValidateAPICredentials(t.Context(), asset.Spot)
	assert.NoError(t, err)
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := c.GetServerTime(t.Context(), 0)
	assert.NoError(t, err)
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := c.GetLatestFundingRates(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)
	req := fundingrate.LatestRateRequest{Asset: asset.UpsideProfitContract}
	_, err = c.GetLatestFundingRates(t.Context(), &req)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	req.Asset = asset.Futures
	resp, err := c.GetLatestFundingRates(t.Context(), &req)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := c.GetFuturesContractDetails(t.Context(), asset.Empty)
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset)
	_, err = c.GetFuturesContractDetails(t.Context(), asset.UpsideProfitContract)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetFuturesContractDetails(t.Context(), asset.Futures)
	require.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := c.UpdateOrderExecutionLimits(t.Context(), asset.Options)
	assert.Equal(t, errOptionInvalid, err.Error())
	err = c.UpdateOrderExecutionLimits(t.Context(), asset.Spot)
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
	resp := c.getOrderRespToOrderDetail(mockData, testPairStable, asset.Spot)
	expected := &order.Detail{ImmediateOrCancel: true, Exchange: "CoinbasePro", Type: 0x40, Side: 0x2, Status: 0x8000, AssetType: 0x1, Date: time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC), CloseTime: time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC), LastUpdated: time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC), Pair: testPairStable}
	assert.Equal(t, expected, resp)
	mockData.Side = "SELL"
	mockData.Status = "FILLED"
	resp = c.getOrderRespToOrderDetail(mockData, testPairStable, asset.Spot)
	expected.Side = 0x4
	expected.Status = 0x80
	assert.Equal(t, expected, resp)
	mockData.Status = "CANCELLED"
	resp = c.getOrderRespToOrderDetail(mockData, testPairStable, asset.Spot)
	expected.Status = 0x100
	assert.Equal(t, expected, resp)
	mockData.Status = "EXPIRED"
	resp = c.getOrderRespToOrderDetail(mockData, testPairStable, asset.Spot)
	expected.Status = 0x2000
	assert.Equal(t, expected, resp)
	mockData.Status = "FAILED"
	resp = c.getOrderRespToOrderDetail(mockData, testPairStable, asset.Spot)
	expected.Status = 0x1000
	assert.Equal(t, expected, resp)
	mockData.Status = "UNKNOWN_ORDER_STATUS"
	resp = c.getOrderRespToOrderDetail(mockData, testPairStable, asset.Spot)
	expected.Status = 0x0
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

func TestFormatExchangeKlineIntervalV3(t *testing.T) {
	t.Parallel()
	testSequence := map[kline.Interval]string{
		kline.OneMin:     granOneMin,
		kline.FiveMin:    granFiveMin,
		kline.FifteenMin: granFifteenMin,
		kline.ThirtyMin:  granThirtyMin,
		kline.OneHour:    granOneHour,
		kline.TwoHour:    granTwoHour,
		kline.SixHour:    granSixHour,
		kline.OneDay:     granOneDay,
		kline.OneWeek:    "",
	}
	for k := range testSequence {
		resp, err := FormatExchangeKlineIntervalV3(k)
		if resp != testSequence[k] {
			t.Errorf(errExpectMismatch, resp, testSequence[k])
		}
		if resp == "" {
			assert.ErrorIs(t, err, kline.ErrUnsupportedInterval)
		}
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, c)
	for _, a := range c.GetAssetTypes(false) {
		pairs, err := c.CurrencyPairs.GetPairs(a, false)
		require.NoError(t, err, "cannot get pairs for %s", a)
		require.NotEmpty(t, pairs, "no pairs for %s", a)
		resp, err := c.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}

func TestScheduleFuturesSweep(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	curSweeps, err := c.ListFuturesSweeps(t.Context())
	assert.NoError(t, err)
	preCancel := false
	if len(curSweeps) > 0 {
		for i := range curSweeps {
			if curSweeps[i].Status == "PENDING" {
				preCancel = true
			}
		}
	}
	if preCancel {
		_, err = c.CancelPendingFuturesSweep(t.Context())
		assert.NoError(t, err)
	}
	_, err = c.ScheduleFuturesSweep(t.Context(), 0.001337)
	assert.NoError(t, err)
}

func TestCancelPendingFuturesSweep(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	curSweeps, err := c.ListFuturesSweeps(t.Context())
	assert.NoError(t, err)
	partialSkip := false
	if len(curSweeps) > 0 {
		for i := range curSweeps {
			if curSweeps[i].Status == "PENDING" {
				partialSkip = true
			}
		}
	}
	if !partialSkip {
		_, err = c.ScheduleFuturesSweep(t.Context(), 0.001337)
		require.NoError(t, err)
	}
	_, err = c.CancelPendingFuturesSweep(t.Context())
	assert.NoError(t, err)
}

// TestWsAuth dials websocket, sends login request.
func TestWsAuth(t *testing.T) {
	p := currency.Pairs{testPairFiat}
	if c.Websocket.IsEnabled() && !c.API.AuthenticatedWebsocketSupport || !sharedtestvalues.AreAPICredentialsSet(c) {
		t.Skip(websocket.ErrWebsocketNotEnabled.Error())
	}
	var dialer gws.Dialer
	err := c.Websocket.Conn.Dial(&dialer, http.Header{})
	require.NoError(t, err)
	c.Websocket.Wg.Add(1)
	go c.wsReadData()
	err = c.Subscribe(subscription.List{
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
	case badResponse := <-c.Websocket.DataHandler:
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
			case <-c.Websocket.DataHandler:
				continue
			case <-done:
				return
			}
		}
	}()
	_, err := c.wsHandleData(nil)
	var syntaxErr *json.SyntaxError
	if !assert.ErrorAs(t, err, &syntaxErr) {
		assert.ErrorContains(t, err, "Syntax error no sources available, the input json is empty")
	}
	mockJSON := []byte(`{"type": "error"}`)
	_, err = c.wsHandleData(mockJSON)
	assert.Error(t, err)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "subscriptions"}`)
	_, err = c.wsHandleData(mockJSON)
	assert.NoError(t, err)
	var unmarshalTypeErr *json.UnmarshalTypeError
	mockJSON = []byte(`{"sequence_num": 0, "channel": "status", "events": [{"type": 1234}]}`)
	_, err = c.wsHandleData(mockJSON)
	if !assert.ErrorAs(t, err, &unmarshalTypeErr) {
		assert.ErrorContains(t, err, "mismatched type with value")
	}
	mockJSON = []byte(`{"sequence_num": 0, "channel": "status", "events": [{"type": "moo"}]}`)
	_, err = c.wsHandleData(mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "ticker", "events": [{"type": "moo", "tickers": false}]}`)
	_, err = c.wsHandleData(mockJSON)
	if !assert.ErrorAs(t, err, &unmarshalTypeErr) {
		assert.ErrorContains(t, err, "mismatched type with value")
	}
	mockJSON = []byte(`{"sequence_num": 0, "channel": "candles", "events": [{"type": false}]}`)
	_, err = c.wsHandleData(mockJSON)
	if !assert.ErrorAs(t, err, &unmarshalTypeErr) {
		assert.ErrorContains(t, err, "mismatched type with value")
	}
	mockJSON = []byte(`{"sequence_num": 0, "channel": "candles", "events": [{"type": "moo", "candles": [{"low": "1.1"}]}]}`)
	_, err = c.wsHandleData(mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "market_trades", "events": [{"type": false}]}`)
	_, err = c.wsHandleData(mockJSON)
	if !assert.ErrorAs(t, err, &unmarshalTypeErr) {
		assert.ErrorContains(t, err, "mismatched type with value")
	}
	mockJSON = []byte(`{"sequence_num": 0, "channel": "market_trades", "events": [{"type": "moo", "trades": [{"price": "1.1"}]}]}`)
	_, err = c.wsHandleData(mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "l2_data", "events": [{"type": false, "updates": [{"price_level": "1.1"}]}]}`)
	_, err = c.wsHandleData(mockJSON)
	if !assert.ErrorAs(t, err, &unmarshalTypeErr) {
		assert.ErrorContains(t, err, "mismatched type with value")
	}
	mockJSON = []byte(`{"sequence_num": 0, "channel": "l2_data", "timestamp": "2006-01-02T15:04:05Z", "events": [{"type": "moo", "updates": [{"price_level": "1.1"}]}]}`)
	_, err = c.wsHandleData(mockJSON)
	assert.ErrorIs(t, err, errUnknownL2DataType)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "l2_data", "timestamp": "2006-01-02T15:04:05Z", "events": [{"type": "snapshot", "product_id": "BTC-USD", "updates": [{"side": "bid", "price_level": "1.1", "new_quantity": "2.2"}]}]}`)
	_, err = c.wsHandleData(mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "l2_data", "timestamp": "2006-01-02T15:04:05Z", "events": [{"type": "update", "product_id": "BTC-USD", "updates": [{"side": "bid", "price_level": "1.1", "new_quantity": "2.2"}]}]}`)
	_, err = c.wsHandleData(mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "user", "events": [{"type": false}]}`)
	_, err = c.wsHandleData(mockJSON)
	if !assert.ErrorAs(t, err, &unmarshalTypeErr) {
		assert.ErrorContains(t, err, "mismatched type with value")
	}
	mockJSON = []byte(`{"sequence_num": 0, "channel": "user", "events": [{"type": "moo", "orders": [{"limit_price": "2.2", "total_fees": "1.1"}], "positions": {"perpetual_futures_positions": [{"margin_type": "fakeMarginType"}], "expiring_futures_positions": [{}]}}]}`)
	_, err = c.wsHandleData(mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "fakechan", "events": [{"type": ""}]}`)
	_, err = c.wsHandleData(mockJSON)
	assert.ErrorIs(t, err, errChannelNameUnknown)
	p, err := c.FormatExchangeCurrency(currency.NewBTCUSD(), asset.Spot)
	require.NoError(t, err)
	c.pairAliases.Load(map[currency.Pair]currency.Pairs{
		p: {p},
	})
	mockJSON = []byte(`{"sequence_num": 0, "channel": "ticker", "events": [{"type": "moo", "tickers": [{"product_id": "BTC-USD", "price": "1.1"}]}]}`)
	_, err = c.wsHandleData(mockJSON)
	assert.NoError(t, err)
}

func TestProcessSnapshotUpdate(t *testing.T) {
	t.Parallel()
	req := WebsocketOrderbookDataHolder{Changes: []WebsocketOrderbookData{{Side: "fakeside", PriceLevel: 1.1, NewQuantity: 2.2}}, ProductID: currency.NewBTCUSD()}
	err := c.ProcessSnapshot(&req, time.Time{})
	assert.ErrorIs(t, err, order.ErrSideIsInvalid)
	err = c.ProcessUpdate(&req, time.Time{})
	assert.ErrorIs(t, err, order.ErrSideIsInvalid)
	req.Changes[0].Side = "offer"
	err = c.ProcessSnapshot(&req, time.Now())
	assert.NoError(t, err)
	err = c.ProcessUpdate(&req, time.Now())
	assert.NoError(t, err)
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()
	c := new(CoinbasePro) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	if err := testexch.Setup(c); err != nil {
		log.Fatal(err)
	}
	c.Websocket.SetCanUseAuthenticatedEndpoints(true)
	p1, err := c.GetEnabledPairs(asset.Spot)
	require.NoError(t, err)
	p2, err := c.GetEnabledPairs(asset.Futures)
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
	subs, err := c.generateSubscriptions()
	require.NoError(t, err)
	testsubs.EqualLists(t, exp, subs)
	_, err = subscription.List{{Channel: "wibble"}}.ExpandTemplates(c)
	assert.ErrorContains(t, err, "subscription channel not supported: wibble")
}

func TestSubscribeUnsubscribe(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	req := subscription.List{{Channel: "heartbeat", Asset: asset.Spot, Pairs: currency.Pairs{currency.NewPairWithDelimiter(testCrypto.String(), testFiat.String(), "-")}}}
	err := c.Subscribe(req)
	assert.NoError(t, err)
	err = c.Unsubscribe(req)
	assert.NoError(t, err)
}

func TestCheckSubscriptions(t *testing.T) {
	t.Parallel()
	c := &CoinbasePro{ //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
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
	c.checkSubscriptions()
	testsubs.EqualLists(t, defaultSubscriptions.Enabled(), c.Features.Subscriptions)
	testsubs.EqualLists(t, defaultSubscriptions, c.Config.Features.Subscriptions)
}

func TestGetJWT(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, _, err := c.GetJWT(t.Context(), "")
	assert.NoError(t, err)
}

func TestEncodeDateRange(t *testing.T) {
	t.Parallel()
	var p Params
	err := p.encodeDateRange(time.Time{}, time.Time{}, "", "")
	assert.NoError(t, err)
	err = p.encodeDateRange(time.Unix(1, 1), time.Unix(1, 1), "", "")
	assert.ErrorIs(t, err, common.ErrStartEqualsEnd)
	err = p.encodeDateRange(time.Unix(1, 1), time.Unix(2, 2), "", "")
	assert.ErrorIs(t, err, errDateLabelEmpty)
	err = p.encodeDateRange(time.Unix(1, 1), time.Unix(2, 2), "start", "end")
	assert.ErrorIs(t, err, errParamValuesNil)
	p.Values = url.Values{}
	err = p.encodeDateRange(time.Unix(1, 1), time.Unix(2, 2), "start", "end")
	assert.NoError(t, err)
}

func TestEncodePagination(t *testing.T) {
	t.Parallel()
	var p Params
	err := p.encodePagination(PaginationInp{})
	assert.ErrorIs(t, err, errParamValuesNil)
	p.Values = url.Values{}
	err = p.encodePagination(PaginationInp{
		Limit:         1,
		OrderAscend:   true,
		StartingAfter: "a",
		EndingBefore:  "b",
	})
	assert.NoError(t, err)
}

func TestCreateOrderConfig(t *testing.T) {
	t.Parallel()
	_, err := createOrderConfig("", "", 0, 0, 0, 0, time.Time{}, false)
	assert.ErrorIs(t, err, errInvalidOrderType)
	_, err = createOrderConfig(order.Market.String(), "", 1, 2, 0, 0, time.Time{}, false)
	assert.NoError(t, err)
	_, err = createOrderConfig(order.Limit.String(), "", 1, 2, 0, 0, time.Time{}, false)
	assert.NoError(t, err)
	_, err = createOrderConfig(order.Limit.String(), "", 0, 0, 0, 0, time.Unix(1, 1), false)
	assert.ErrorIs(t, err, errEndTimeInPast)
	_, err = createOrderConfig(order.Limit.String(), "", 1, 2, 0, 0, time.Now().Add(time.Hour), false)
	assert.NoError(t, err)
	_, err = createOrderConfig(order.StopLimit.String(), "", 1, 2, 0, 0, time.Time{}, false)
	assert.NoError(t, err)
	_, err = createOrderConfig(order.StopLimit.String(), "", 0, 0, 0, 0, time.Unix(1, 1), false)
	assert.ErrorIs(t, err, errEndTimeInPast)
	_, err = createOrderConfig(order.StopLimit.String(), "", 1, 2, 0, 0, time.Now().Add(time.Hour), false)
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
	resp1, resp2, _ := strategyDecoder("IMMEDIATE_OR_CANCEL")
	assert.True(t, resp1)
	assert.False(t, resp2)
	resp1, resp2, _ = strategyDecoder("FILL_OR_KILL")
	assert.False(t, resp1)
	assert.True(t, resp2)
	resp1, resp2, _ = strategyDecoder("GOOD_UNTIL_CANCELLED")
	assert.False(t, resp1)
	assert.False(t, resp2)
	_, _, err := strategyDecoder("")
	assert.ErrorIs(t, err, errUnrecognisedStrategyType)
}

func TestBase64URLEncode(t *testing.T) {
	t.Parallel()
	resp := base64URLEncode([]byte{byte(252), byte(253), byte(254), byte(255)})
	assert.Equal(t, "_P3-_w", resp)
}

func TestProcessFundingData(t *testing.T) {
	t.Parallel()
	resp := c.processFundingData([]DeposWithdrData{
		{},
	}, []TransactionData{
		{
			Type: "receive",
		},
		{
			Type: "send",
		},
	})
	assert.NotEmpty(t, resp)
}

func exchangeBaseHelper(c *CoinbasePro) error {
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		return err
	}
	gdxConfig, err := cfg.GetExchangeConfig("CoinbasePro")
	if err != nil {
		return err
	}
	if apiKey != "" {
		gdxConfig.API.Credentials.Key = apiKey
		gdxConfig.API.Credentials.Secret = apiSecret
		gdxConfig.API.AuthenticatedSupport = true
		gdxConfig.API.AuthenticatedWebsocketSupport = true
	}
	c.Websocket = sharedtestvalues.NewTestWebsocket()
	err = c.Setup(gdxConfig)
	if err != nil {
		return err
	}
	return nil
}

func getINTXPortfolio(t *testing.T) string {
	t.Helper()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetAllPortfolios(t.Context(), "")
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
	accIDs, err := c.GetAllAccounts(t.Context(), 250, "")
	assert.NoError(t, err)
	if accIDs == nil || len(accIDs.Accounts) == 0 {
		t.Fatal(errExpectedNonEmpty)
	}
	for x := range accIDs.Accounts {
		if accIDs.Accounts[x].Currency == testStable.String() {
			fromAccID = accIDs.Accounts[x].UUID
		}
		if accIDs.Accounts[x].Currency == testFiat.String() {
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
	pmID, err := c.GetAllPaymentMethods(t.Context())
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
	assert.ErrorIs(t, err, common.ErrExchangeNameUnset)
	req.Exchange = c.Name
	req.Currency = testFiat
	req.Amount = 1
	req.Type = withdraw.Fiat
	req.Fiat.Bank.Enabled = true
	req.Fiat.Bank.SupportedExchanges = "CoinbasePro"
	req.Fiat.Bank.SupportedCurrencies = testFiat.String()
	req.Fiat.Bank.AccountNumber = "123"
	req.Fiat.Bank.SWIFTCode = "456"
	req.Fiat.Bank.BSBNumber = "789"
	_, err = fn(t.Context(), &req)
	assert.ErrorIs(t, err, errWalletIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	req.WalletID = "meow"
	req.Fiat.Bank.BankName = "GCT's Officially Fake and Not Real Test Bank"
	_, err = fn(t.Context(), &req)
	assert.ErrorIs(t, err, errPayMethodNotFound)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	wallets, err := c.GetAllWallets(t.Context(), PaginationInp{})
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	fromAccID, toAccID := convertTestHelper(t)
	resp, err := c.CreateConvertQuote(t.Context(), fromAccID, toAccID, "", "", 0.01)
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
