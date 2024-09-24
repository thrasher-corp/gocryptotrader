package coinbasepro

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
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
	testPair         = currency.NewPairWithDelimiter(testCrypto.String(), testFiat.String(), "-")
)

// Constants used within tests
const (
	testAddress = "fake address"
	testAmount  = 1e-08
	testPrice   = 1e+09

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

	expectedTimestamp = "1970-01-01 00:20:34 +0000 UTC"
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
	var dialer websocket.Dialer
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
	assert.ErrorIs(t, err, stream.ErrAlreadyDisabled)
	err = exch.WsConnect()
	assert.ErrorIs(t, err, stream.ErrWebsocketNotEnabled)
	exch.SetDefaults()
	err = exchangeBaseHelper(exch)
	require.NoError(t, err)
	err = exch.Websocket.Enable()
	assert.NoError(t, err)
}

func TestGetAllAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetAllAccounts(context.Background(), 50, "")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAccountByID(t *testing.T) {
	t.Parallel()
	_, err := c.GetAccountByID(context.Background(), "")
	assert.ErrorIs(t, err, errAccountIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	longResp, err := c.GetAllAccounts(context.Background(), 49, "")
	assert.NoError(t, err)
	if longResp == nil || len(longResp.Accounts) == 0 {
		t.Fatal(errExpectedNonEmpty)
	}
	shortResp, err := c.GetAccountByID(context.Background(), longResp.Accounts[0].UUID)
	assert.NoError(t, err)
	if *shortResp != longResp.Accounts[0] {
		t.Errorf(errExpectMismatch, shortResp, longResp.Accounts[0])
	}
}

func TestGetBestBidAsk(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	testPairs := []string{testPair.String(), "ETH-USD"}
	resp, err := c.GetBestBidAsk(context.Background(), testPairs)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetProductBookV3(t *testing.T) {
	t.Parallel()
	_, err := c.GetProductBookV3(context.Background(), "", 0, false)
	assert.ErrorIs(t, err, errProductIDEmpty)
	resp, err := c.GetProductBookV3(context.Background(), testPair.String(), 2, false)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err = c.GetProductBookV3(context.Background(), testPair.String(), 2, true)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAllProducts(t *testing.T) {
	t.Parallel()
	testPairs := []string{testPair.String(), "ETH-USD"}
	resp, err := c.GetAllProducts(context.Background(), 30000, 1, "SPOT", "PERPETUAL", "STATUS_ALL", testPairs, false)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err = c.GetAllProducts(context.Background(), 0, 1, "SPOT", "PERPETUAL", "STATUS_ALL", nil, true)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetProductByID(t *testing.T) {
	t.Parallel()
	_, err := c.GetProductByID(context.Background(), "", false)
	assert.ErrorIs(t, err, errProductIDEmpty)
	resp, err := c.GetProductByID(context.Background(), testPair.String(), false)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err = c.GetProductByID(context.Background(), testPair.String(), true)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetHistoricRates(t *testing.T) {
	t.Parallel()
	_, err := c.GetHistoricRates(context.Background(), "", granUnknown, time.Time{}, time.Time{}, false)
	assert.ErrorIs(t, err, errProductIDEmpty)
	_, err = c.GetHistoricRates(context.Background(), testPair.String(), "blorbo", time.Time{}, time.Time{}, false)
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval)
	resp, err := c.GetHistoricRates(context.Background(), testPair.String(), granOneMin, time.Now().Add(-5*time.Minute), time.Now(), false)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err = c.GetHistoricRates(context.Background(), testPair.String(), granOneMin, time.Now().Add(-5*time.Minute), time.Now(), true)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := c.GetTicker(context.Background(), "", 1, time.Time{}, time.Time{}, false)
	assert.ErrorIs(t, err, errProductIDEmpty)
	resp, err := c.GetTicker(context.Background(), testPair.String(), 5, time.Now().Add(-time.Minute*5), time.Now(), false)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err = c.GetTicker(context.Background(), testPair.String(), 5, time.Now().Add(-time.Minute*5), time.Now(), true)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestPlaceOrder(t *testing.T) {
	t.Parallel()
	_, err := c.PlaceOrder(context.Background(), "", "", "", "", "", "", "", "", 0, 0, 0, 0, false, time.Time{})
	assert.ErrorIs(t, err, errClientOrderIDEmpty)
	_, err = c.PlaceOrder(context.Background(), "meow", "", "", "", "", "", "", "", 0, 0, 0, 0, false, time.Time{})
	assert.ErrorIs(t, err, errProductIDEmpty)
	_, err = c.PlaceOrder(context.Background(), "meow", testPair.String(), order.Sell.String(), "", "", "", "", "", 0, 0, 0, 0, false, time.Time{})
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	skipTestIfLowOnFunds(t)
	id, err := uuid.NewV4()
	assert.NoError(t, err)
	resp, err := c.PlaceOrder(context.Background(), id.String(), testPair.String(), order.Sell.String(), "", order.Limit.String(), "", "CROSS", "", testAmount, testPrice, 0, 9999, false, time.Now().Add(time.Hour))
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
	id, err = uuid.NewV4()
	assert.NoError(t, err)
	resp, err = c.PlaceOrder(context.Background(), id.String(), testPair.String(), order.Sell.String(), "", order.Limit.String(), "", "MULTI", "", testAmount, testPrice, 0, 9999, false, time.Now().Add(time.Hour))
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func orderTestHelper(t *testing.T, orderSide string) *GetAllOrdersResp {
	t.Helper()
	ordIDs, err := c.GetAllOrders(context.Background(), "", "", "", orderSide, "", "", "", "", "", []string{}, []string{}, 1000, time.Time{}, time.Time{})
	assert.NoError(t, err)
	if ordIDs == nil || len(ordIDs.Orders) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	for i := range ordIDs.Orders {
		if ordIDs.Orders[i].Status == order.Open.String() {
			ordIDs.Orders = ordIDs.Orders[i : i+1]
			return ordIDs
		}
	}
	t.Skip(skipInsufficientOrders)
	return nil
}

func TestCancelOrders(t *testing.T) {
	t.Parallel()
	var orderSlice []string
	_, err := c.CancelOrders(context.Background(), orderSlice)
	assert.ErrorIs(t, err, errOrderIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	ordIDs := orderTestHelper(t, "")
	orderSlice = append(orderSlice, ordIDs.Orders[0].OrderID)
	resp, err := c.CancelOrders(context.Background(), orderSlice)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestEditOrder(t *testing.T) {
	t.Parallel()
	_, err := c.EditOrder(context.Background(), "", 0, 0)
	assert.ErrorIs(t, err, errOrderIDEmpty)
	_, err = c.EditOrder(context.Background(), "meow", 0, 0)
	assert.ErrorIs(t, err, errSizeAndPriceZero)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	ordIDs := orderTestHelper(t, "SELL")
	resp, err := c.EditOrder(context.Background(), ordIDs.Orders[0].OrderID, testAmount, testPrice*10)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestEditOrderPreview(t *testing.T) {
	t.Parallel()
	_, err := c.EditOrderPreview(context.Background(), "", 0, 0)
	assert.ErrorIs(t, err, errOrderIDEmpty)
	_, err = c.EditOrderPreview(context.Background(), "meow", 0, 0)
	assert.ErrorIs(t, err, errSizeAndPriceZero)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	ordIDs := orderTestHelper(t, "")
	resp, err := c.EditOrderPreview(context.Background(), ordIDs.Orders[0].OrderID, testAmount, testPrice*10)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAllOrders(t *testing.T) {
	t.Parallel()
	assets := []string{testFiat.String()}
	status := make([]string, 2)
	_, err := c.GetAllOrders(context.Background(), "", "", "", "", "", "", "", "", "", status, assets, 0, time.Unix(2, 2), time.Unix(1, 1))
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	status[0] = "CANCELLED"
	status[1] = "OPEN"
	_, err = c.GetAllOrders(context.Background(), "", "", "", "", "", "", "", "", "", status, assets, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errOpenPairWithOtherTypes)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	status = make([]string, 0)
	assets = make([]string, 1)
	assets[0] = testCrypto.String()
	_, err = c.GetAllOrders(context.Background(), "", testFiat.String(), "LIMIT", "SELL", "", "SPOT", "RETAIL_ADVANCED", "UNKNOWN_CONTRACT_EXPIRY_TYPE", "2", status, assets, 10, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetFills(t *testing.T) {
	t.Parallel()
	_, err := c.GetFills(context.Background(), "", "", "", time.Unix(2, 2), time.Unix(1, 1), 0)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err = c.GetFills(context.Background(), "", testPair.String(), "", time.Unix(1, 1), time.Now(), 5)
	assert.NoError(t, err)
	status := []string{"OPEN"}
	ordID, err := c.GetAllOrders(context.Background(), "", "", "", "", "", "", "", "", "", status, nil, 3, time.Time{}, time.Time{})
	assert.NoError(t, err)
	if ordID == nil || len(ordID.Orders) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = c.GetFills(context.Background(), ordID.Orders[0].OrderID, "", "", time.Time{}, time.Time{}, 5)
	assert.NoError(t, err)
}

func TestGetOrderByID(t *testing.T) {
	t.Parallel()
	_, err := c.GetOrderByID(context.Background(), "", "", "")
	assert.ErrorIs(t, err, errOrderIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	ordID, err := c.GetAllOrders(context.Background(), "", "", "", "", "", "", "", "", "", nil, nil, 10, time.Time{}, time.Time{})
	assert.NoError(t, err)
	if ordID == nil || len(ordID.Orders) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	resp, err := c.GetOrderByID(context.Background(), ordID.Orders[0].OrderID, ordID.Orders[0].ClientOID, testFiat.String())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestPreviewOrder(t *testing.T) {
	t.Parallel()
	_, err := c.PreviewOrder(context.Background(), "", "", "", "", "", 0, 0, 0, 0, 0, 0, false, false, false, time.Time{})
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid)
	_, err = c.PreviewOrder(context.Background(), "", "", "", "", "", 0, 1, 0, 0, 0, 0, false, false, false, time.Time{})
	assert.ErrorIs(t, err, errInvalidOrderType)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	skipTestIfLowOnFunds(t)
	resp, err := c.PreviewOrder(context.Background(), testPair.String(), "BUY", "MARKET", "", "ISOLATED", 0, testAmount, 0, 0, 0, 0, false, false, false, time.Time{})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAllPortfolios(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetAllPortfolios(context.Background(), "DEFAULT")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestCreatePortfolio(t *testing.T) {
	t.Parallel()
	_, err := c.CreatePortfolio(context.Background(), "")
	assert.ErrorIs(t, err, errNameEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	_, err = c.CreatePortfolio(context.Background(), "GCT Test Portfolio")
	if err != nil && err.Error() != errPortfolioNameDuplicate {
		t.Error(err)
	}
}

func TestMovePortfolioFunds(t *testing.T) {
	t.Parallel()
	_, err := c.MovePortfolioFunds(context.Background(), "", "", "", 0)
	assert.ErrorIs(t, err, errPortfolioIDEmpty)
	_, err = c.MovePortfolioFunds(context.Background(), "", "meowPort", "woofPort", 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = c.MovePortfolioFunds(context.Background(), testCrypto.String(), "meowPort", "woofPort", 0)
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	portID, err := c.GetAllPortfolios(context.Background(), "")
	assert.NoError(t, err)
	if len(portID) < 2 {
		t.Skip(skipInsufficientPortfolios)
	}
	_, err = c.MovePortfolioFunds(context.Background(), testCrypto.String(), portID[0].UUID, portID[1].UUID, testAmount)
	assert.NoError(t, err)
}

func TestGetPortfolioByID(t *testing.T) {
	t.Parallel()
	_, err := c.GetPortfolioByID(context.Background(), "")
	assert.ErrorIs(t, err, errPortfolioIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	portID, err := c.GetAllPortfolios(context.Background(), "")
	assert.NoError(t, err)
	if len(portID) == 0 {
		t.Fatal(errExpectedNonEmpty)
	}
	resp, err := c.GetPortfolioByID(context.Background(), portID[0].UUID)
	assert.NoError(t, err)
	if resp.Portfolio != portID[0] {
		t.Errorf(errExpectMismatch, resp.Portfolio, portID[0])
	}
}

func TestDeletePortfolio(t *testing.T) {
	t.Parallel()
	err := c.DeletePortfolio(context.Background(), "")
	assert.ErrorIs(t, err, errPortfolioIDEmpty)
	pID := portfolioIDFromName(t, "GCT Test Portfolio To-Delete")
	err = c.DeletePortfolio(context.Background(), pID)
	assert.NoError(t, err)
}

func TestEditPortfolio(t *testing.T) {
	t.Parallel()
	_, err := c.EditPortfolio(context.Background(), "", "")
	assert.ErrorIs(t, err, errPortfolioIDEmpty)
	_, err = c.EditPortfolio(context.Background(), "meow", "")
	assert.ErrorIs(t, err, errNameEmpty)
	pID := portfolioIDFromName(t, "GCT Test Portfolio To-Edit")
	_, err = c.EditPortfolio(context.Background(), pID, "GCT Test Portfolio Edited")
	if err != nil && err.Error() != errPortfolioNameDuplicate {
		t.Error(err)
	}
}

func TestGetFuturesBalanceSummary(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetFuturesBalanceSummary(context.Background())
	assert.NoError(t, err)
}

func TestGetAllFuturesPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetAllFuturesPositions(context.Background())
	assert.NoError(t, err)
}

func TestGetFuturesPositionByID(t *testing.T) {
	t.Parallel()
	_, err := c.GetFuturesPositionByID(context.Background(), "")
	assert.ErrorIs(t, err, errProductIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err = c.GetFuturesPositionByID(context.Background(), "meow")
	assert.NoError(t, err)
}

func TestListFuturesSweeps(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.ListFuturesSweeps(context.Background())
	assert.NoError(t, err)
}

func TestAllocatePortfolio(t *testing.T) {
	t.Parallel()
	err := c.AllocatePortfolio(context.Background(), "", "", "", 0)
	assert.ErrorIs(t, err, errPortfolioIDEmpty)
	err = c.AllocatePortfolio(context.Background(), "meow", "", "", 0)
	assert.ErrorIs(t, err, errProductIDEmpty)
	err = c.AllocatePortfolio(context.Background(), "meow", "bark", "", 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	pID := getINTXPortfolio(t)
	err = c.AllocatePortfolio(context.Background(), pID, testCrypto.String(), testFiat.String(), 0.001337)
	assert.NoError(t, err)
}

func TestGetPerpetualsPortfolioSummary(t *testing.T) {
	t.Parallel()
	_, err := c.GetPerpetualsPortfolioSummary(context.Background(), "")
	assert.ErrorIs(t, err, errPortfolioIDEmpty)
	pID := getINTXPortfolio(t)
	resp, err := c.GetPerpetualsPortfolioSummary(context.Background(), pID)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAllPerpetualsPositions(t *testing.T) {
	t.Parallel()
	_, err := c.GetAllPerpetualsPositions(context.Background(), "")
	assert.ErrorIs(t, err, errPortfolioIDEmpty)
	pID := getINTXPortfolio(t)
	_, err = c.GetAllPerpetualsPositions(context.Background(), pID)
	assert.NoError(t, err)
}

func TestGetPerpetualsPositionByID(t *testing.T) {
	t.Parallel()
	_, err := c.GetPerpetualsPositionByID(context.Background(), "", "")
	assert.ErrorIs(t, err, errPortfolioIDEmpty)
	_, err = c.GetPerpetualsPositionByID(context.Background(), "meow", "")
	assert.ErrorIs(t, err, errProductIDEmpty)
	pID := getINTXPortfolio(t)
	_, err = c.GetPerpetualsPositionByID(context.Background(), pID, testPair.String())
	assert.NoError(t, err)
}

func TestGetTransactionSummary(t *testing.T) {
	t.Parallel()
	_, err := c.GetTransactionSummary(context.Background(), time.Unix(2, 2), time.Unix(1, 1), "", "", "")
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetTransactionSummary(context.Background(), time.Unix(1, 1), time.Now(), testFiat.String(), asset.Spot.Upper(), "UNKNOWN_CONTRACT_EXPIRY_TYPE")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestCreateConvertQuote(t *testing.T) {
	t.Parallel()
	_, err := c.CreateConvertQuote(context.Background(), "", "", "", "", 0)
	assert.ErrorIs(t, err, errAccountIDEmpty)
	_, err = c.CreateConvertQuote(context.Background(), "meow", "123", "", "", 0)
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	fromAccID, toAccID := convertTestHelper(t)
	resp, err := c.CreateConvertQuote(context.Background(), fromAccID, toAccID, "", "", 0.01)
	assert.NoError(t, err)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	testGetNoArgs(t, c.GetV3Time)
}

func TestGetAllPaymentMethods(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	testGetNoArgs(t, c.GetAllPaymentMethods)
}

func TestGetPaymentMethodByID(t *testing.T) {
	t.Parallel()
	_, err := c.GetPaymentMethodByID(context.Background(), "")
	assert.ErrorIs(t, err, errPaymentMethodEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	pmID, err := c.GetAllPaymentMethods(context.Background())
	assert.NoError(t, err)
	if pmID == nil || len(pmID) == 0 {
		t.Skip(skipPayMethodNotFound)
	}
	resp, err := c.GetPaymentMethodByID(context.Background(), pmID[0].ID)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestListNotifications(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.ListNotifications(context.Background(), PaginationInp{})
	assert.NoError(t, err)
}

func TestGetCurrentUser(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	testGetNoArgs(t, c.GetCurrentUser)
}

func TestGetAllWallets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	pagIn := PaginationInp{Limit: 2}
	resp, err := c.GetAllWallets(context.Background(), pagIn)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
	if resp.Pagination.NextStartingAfter == "" {
		t.Skip(skipInsufficientWallets)
	}
	pagIn.StartingAfter = resp.Pagination.NextStartingAfter
	resp, err = c.GetAllWallets(context.Background(), pagIn)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetWalletByID(t *testing.T) {
	t.Parallel()
	_, err := c.GetWalletByID(context.Background(), "", "")
	assert.ErrorIs(t, err, errCurrWalletConflict)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetWalletByID(context.Background(), "", testCrypto.String())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
	resp, err = c.GetWalletByID(context.Background(), resp.ID, "")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestCreateAddress(t *testing.T) {
	t.Parallel()
	_, err := c.CreateAddress(context.Background(), "", "")
	assert.ErrorIs(t, err, errWalletIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	wID, err := c.GetWalletByID(context.Background(), "", testCrypto.String())
	assert.NoError(t, err)
	assert.NotEmpty(t, wID, errExpectedNonEmpty)
	resp, err := c.CreateAddress(context.Background(), wID.ID, "")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAllAddresses(t *testing.T) {
	t.Parallel()
	var pag PaginationInp
	_, err := c.GetAllAddresses(context.Background(), "", pag)
	assert.ErrorIs(t, err, errWalletIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	wID, err := c.GetWalletByID(context.Background(), "", testCrypto.String())
	assert.NoError(t, err)
	assert.NotEmpty(t, wID, errExpectedNonEmpty)
	resp, err := c.GetAllAddresses(context.Background(), wID.ID, pag)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAddressByID(t *testing.T) {
	t.Parallel()
	_, err := c.GetAddressByID(context.Background(), "", "")
	assert.ErrorIs(t, err, errWalletIDEmpty)
	_, err = c.GetAddressByID(context.Background(), "123", "")
	assert.ErrorIs(t, err, errAddressIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	wID, err := c.GetWalletByID(context.Background(), "", testCrypto.String())
	assert.NoError(t, err)
	assert.NotEmpty(t, wID, errExpectedNonEmpty)
	addID, err := c.GetAllAddresses(context.Background(), wID.ID, PaginationInp{})
	assert.NoError(t, err)
	require.NotEmpty(t, addID, errExpectedNonEmpty)
	resp, err := c.GetAddressByID(context.Background(), wID.ID, addID.Data[0].ID)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAddressTransactions(t *testing.T) {
	t.Parallel()
	_, err := c.GetAddressTransactions(context.Background(), "", "", PaginationInp{})
	assert.ErrorIs(t, err, errWalletIDEmpty)
	_, err = c.GetAddressTransactions(context.Background(), "123", "", PaginationInp{})
	assert.ErrorIs(t, err, errAddressIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	wID, err := c.GetWalletByID(context.Background(), "", testCrypto.String())
	assert.NoError(t, err)
	assert.NotEmpty(t, wID, errExpectedNonEmpty)
	addID, err := c.GetAllAddresses(context.Background(), wID.ID, PaginationInp{})
	assert.NoError(t, err)
	require.NotEmpty(t, addID, errExpectedNonEmpty)
	_, err = c.GetAddressTransactions(context.Background(), wID.ID, addID.Data[0].ID, PaginationInp{})
	assert.NoError(t, err)
}

func TestSendMoney(t *testing.T) {
	t.Parallel()
	_, err := c.SendMoney(context.Background(), "", "", "", "", "", "", "", "", 0, false, false)
	assert.ErrorIs(t, err, errTransactionTypeEmpty)
	_, err = c.SendMoney(context.Background(), "123", "", "", "", "", "", "", "", 0, false, false)
	assert.ErrorIs(t, err, errWalletIDEmpty)
	_, err = c.SendMoney(context.Background(), "123", "123", "", "", "", "", "", "", 0, false, false)
	assert.ErrorIs(t, err, errToEmpty)
	_, err = c.SendMoney(context.Background(), "123", "123", "123", "", "", "", "", "", 0, false, false)
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid)
	_, err = c.SendMoney(context.Background(), "123", "123", "123", "", "", "", "", "", 1, false, false)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	wID, err := c.GetAllWallets(context.Background(), PaginationInp{})
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
	resp, err := c.SendMoney(context.Background(), "transfer", wID.Data[0].ID, wID.Data[1].ID, testCrypto.String(), "GCT Test", "123", "", "", testAmount, false, false)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAllTransactions(t *testing.T) {
	t.Parallel()
	var pag PaginationInp
	_, err := c.GetAllTransactions(context.Background(), "", pag)
	assert.ErrorIs(t, err, errWalletIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	wID, err := c.GetWalletByID(context.Background(), "", testCrypto.String())
	assert.NoError(t, err)
	assert.NotEmpty(t, wID, errExpectedNonEmpty)
	_, err = c.GetAllTransactions(context.Background(), wID.ID, pag)
	assert.NoError(t, err)
}

func TestGetTransactionByID(t *testing.T) {
	t.Parallel()
	_, err := c.GetTransactionByID(context.Background(), "", "")
	assert.ErrorIs(t, err, errWalletIDEmpty)
	_, err = c.GetTransactionByID(context.Background(), "123", "")
	assert.ErrorIs(t, err, errTransactionIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	wID, err := c.GetWalletByID(context.Background(), "", testCrypto.String())
	assert.NoError(t, err)
	assert.NotEmpty(t, wID, errExpectedNonEmpty)
	tID, err := c.GetAllTransactions(context.Background(), wID.ID, PaginationInp{})
	assert.NoError(t, err)
	if tID == nil || len(tID.Data) == 0 {
		t.Skip(skipInsufficientTransactions)
	}
	resp, err := c.GetTransactionByID(context.Background(), wID.ID, tID.Data[0].ID)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestFiatTransfer(t *testing.T) {
	t.Parallel()
	_, err := c.FiatTransfer(context.Background(), "", "", "", 0, false, FiatDeposit)
	assert.ErrorIs(t, err, errWalletIDEmpty)
	_, err = c.FiatTransfer(context.Background(), "123", "", "", 0, false, FiatDeposit)
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid)
	_, err = c.FiatTransfer(context.Background(), "123", "", "", 1, false, FiatDeposit)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = c.FiatTransfer(context.Background(), "123", "123", "", 1, false, FiatDeposit)
	assert.ErrorIs(t, err, errPaymentMethodEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	wallets, err := c.GetAllWallets(context.Background(), PaginationInp{})
	assert.NoError(t, err)
	assert.NotEmpty(t, wallets, errExpectedNonEmpty)
	wID, pmID := transferTestHelper(t, wallets)
	resp, err := c.FiatTransfer(context.Background(), wID, testFiat.String(), pmID, testAmount, false, FiatDeposit)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
	resp, err = c.FiatTransfer(context.Background(), wID, testFiat.String(), pmID, testAmount, false, FiatWithdrawal)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestCommitTransfer(t *testing.T) {
	t.Parallel()
	_, err := c.CommitTransfer(context.Background(), "", "", FiatDeposit)
	assert.ErrorIs(t, err, errWalletIDEmpty)
	_, err = c.CommitTransfer(context.Background(), "123", "", FiatDeposit)
	assert.ErrorIs(t, err, errDepositIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	wallets, err := c.GetAllWallets(context.Background(), PaginationInp{})
	assert.NoError(t, err)
	assert.NotEmpty(t, wallets, errExpectedNonEmpty)
	wID, pmID := transferTestHelper(t, wallets)
	depID, err := c.FiatTransfer(context.Background(), wID, testFiat.String(), pmID, testAmount, false, FiatDeposit)
	require.NoError(t, err)
	resp, err := c.CommitTransfer(context.Background(), wID, depID.ID, FiatDeposit)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
	depID, err = c.FiatTransfer(context.Background(), wID, testFiat.String(), pmID, testAmount, false, FiatWithdrawal)
	require.NoError(t, err)
	resp, err = c.CommitTransfer(context.Background(), wID, depID.ID, FiatWithdrawal)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAllFiatTransfers(t *testing.T) {
	t.Parallel()
	var pag PaginationInp
	_, err := c.GetAllFiatTransfers(context.Background(), "", pag, FiatDeposit)
	assert.ErrorIs(t, err, errWalletIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	wID, err := c.GetWalletByID(context.Background(), "", "AUD")
	require.NoError(t, err)
	assert.NotEmpty(t, wID, errExpectedNonEmpty)
	// Fiat deposits/withdrawals aren't accepted for fiat currencies for Australian business accounts; the error
	// "id not found" possibly reflects this
	_, err = c.GetAllFiatTransfers(context.Background(), wID.ID, pag, FiatDeposit)
	assert.NoError(t, err)
	_, err = c.GetAllFiatTransfers(context.Background(), wID.ID, pag, FiatWithdrawal)
	assert.NoError(t, err)
}

func TestGetFiatTransferByID(t *testing.T) {
	t.Parallel()
	_, err := c.GetFiatTransferByID(context.Background(), "", "", FiatDeposit)
	assert.ErrorIs(t, err, errWalletIDEmpty)
	_, err = c.GetFiatTransferByID(context.Background(), "123", "", FiatDeposit)
	assert.ErrorIs(t, err, errDepositIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	wID, err := c.GetWalletByID(context.Background(), "", "AUD")
	require.NoError(t, err)
	assert.NotEmpty(t, wID, errExpectedNonEmpty)
	// Fiat deposits/withdrawals aren't accepted for fiat currencies for Australian business accounts; the error
	// "id not found" possibly reflects this
	dID, err := c.GetAllFiatTransfers(context.Background(), wID.ID, PaginationInp{}, FiatDeposit)
	assert.NoError(t, err)
	if dID == nil || len(dID.Data) == 0 {
		t.Skip(skipInsufficientTransactions)
	}
	resp, err := c.GetFiatTransferByID(context.Background(), wID.ID, dID.Data[0].ID, FiatDeposit)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
	resp, err = c.GetFiatTransferByID(context.Background(), wID.ID, dID.Data[0].ID, FiatWithdrawal)
	assert.NoError(t, err)
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
	resp, err := c.GetExchangeRates(context.Background(), "")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetPrice(t *testing.T) {
	t.Parallel()
	_, err := c.GetPrice(context.Background(), "", "")
	assert.ErrorIs(t, err, errInvalidPriceType)
	resp, err := c.GetPrice(context.Background(), testPair.String(), asset.Spot.String())
	assert.NoError(t, err)
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
	_, err := c.GetAllTradingPairs(context.Background(), "")
	assert.NoError(t, err)
}

func TestGetAllPairVolumes(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, c.GetAllPairVolumes)
}

func TestGetPairDetails(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, c.GetPairDetails, testPair.String(), currency.ErrCurrencyPairEmpty)
}

func TestGetProductBookV1(t *testing.T) {
	t.Parallel()
	_, err := c.GetProductBookV1(context.Background(), "", 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	resp, err := c.GetProductBookV1(context.Background(), testPair.String(), 2)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
	resp, err = c.GetProductBookV1(context.Background(), testPair.String(), 3)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetProductCandles(t *testing.T) {
	t.Parallel()
	_, err := c.GetProductCandles(context.Background(), "", 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	resp, err := c.GetProductCandles(context.Background(), testPair.String(), 300, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetProductStats(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, c.GetProductStats, testPair.String(), currency.ErrCurrencyPairEmpty)
}

func TestGetProductTicker(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, c.GetProductTicker, testPair.String(), currency.ErrCurrencyPairEmpty)
}

func TestGetProductTrades(t *testing.T) {
	t.Parallel()
	_, err := c.GetProductTrades(context.Background(), "", "", "", 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	resp, err := c.GetProductTrades(context.Background(), testPair.String(), "1", "before", 0)
	assert.NoError(t, err)
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
	err := c.SendHTTPRequest(context.Background(), exchange.EdgeCase3, "", nil, nil)
	assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound)
}

func TestSendAuthenticatedHTTPRequest(t *testing.T) {
	t.Parallel()
	fc := &CoinbasePro{}
	err := fc.SendAuthenticatedHTTPRequest(context.Background(), exchange.EdgeCase3, "", "", nil, nil, false, nil, nil)
	assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	err = c.SendAuthenticatedHTTPRequest(context.Background(), exchange.EdgeCase3, "", "", nil, nil, false, nil, nil)
	assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound)
	ch := make(chan struct{})
	body := map[string]interface{}{"Unmarshalable": ch}
	err = c.SendAuthenticatedHTTPRequest(context.Background(), exchange.RestSpot, "", "", nil, body, false, nil, nil)
	var targetErr *json.UnsupportedTypeError
	assert.ErrorAs(t, err, &targetErr)
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	_, err := c.GetFee(context.Background(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)
	feeBuilder := exchange.FeeBuilder{
		FeeType:       exchange.OfflineTradeFee,
		Amount:        1,
		PurchasePrice: 1,
	}
	resp, err := c.GetFee(context.Background(), &feeBuilder)
	assert.NoError(t, err)
	if resp != WorstCaseTakerFee {
		t.Errorf(errExpectMismatch, resp, WorstCaseTakerFee)
	}
	feeBuilder.IsMaker = true
	resp, err = c.GetFee(context.Background(), &feeBuilder)
	assert.NoError(t, err)
	if resp != WorstCaseMakerFee {
		t.Errorf(errExpectMismatch, resp, WorstCaseMakerFee)
	}
	feeBuilder.Pair = currency.NewPair(currency.USDT, currency.USD)
	resp, err = c.GetFee(context.Background(), &feeBuilder)
	assert.NoError(t, err)
	if resp != 0 {
		t.Errorf(errExpectMismatch, resp, StablePairMakerFee)
	}
	feeBuilder.IsMaker = false
	resp, err = c.GetFee(context.Background(), &feeBuilder)
	assert.NoError(t, err)
	if resp != WorstCaseStablePairTakerFee {
		t.Errorf(errExpectMismatch, resp, WorstCaseStablePairTakerFee)
	}
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	_, err = c.GetFee(context.Background(), &feeBuilder)
	assert.ErrorIs(t, err, errFeeTypeNotSupported)
	feeBuilder.Pair = currency.Pair{}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	feeBuilder.FeeType = exchange.CryptocurrencyTradeFee
	resp, err = c.GetFee(context.Background(), &feeBuilder)
	assert.NoError(t, err)
	if !(resp <= WorstCaseTakerFee && resp >= BestCaseTakerFee) {
		t.Errorf(errExpectedFeeRange, BestCaseTakerFee, WorstCaseTakerFee, resp)
	}
	feeBuilder.IsMaker = true
	resp, err = c.GetFee(context.Background(), &feeBuilder)
	assert.NoError(t, err)
	if !(resp <= WorstCaseMakerFee && resp >= BestCaseMakerFee) {
		t.Errorf(errExpectedFeeRange, BestCaseMakerFee, WorstCaseMakerFee, resp)
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := c.FetchTradablePairs(context.Background(), asset.Options)
	assert.EqualValues(t, errOptionInvalid, err.Error())
	resp, err := c.FetchTradablePairs(context.Background(), asset.Spot)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
	resp, err = c.FetchTradablePairs(context.Background(), asset.Futures)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	err := c.UpdateTradablePairs(context.Background(), false)
	assert.NoError(t, err)
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.UpdateAccountInfo(context.Background(), asset.Spot)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestFetchAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.FetchAccountInfo(context.Background(), asset.Spot)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	_, err := c.UpdateTicker(context.Background(), currency.Pair{}, asset.Spot)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	resp, err := c.UpdateTicker(context.Background(), testPair, asset.Spot)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestFetchTicker(t *testing.T) {
	t.Parallel()
	resp, err := c.FetchTicker(context.Background(), testPair, asset.Spot)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestFetchOrderbook(t *testing.T) {
	t.Parallel()
	resp, err := c.FetchOrderbook(context.Background(), testPair, asset.Spot)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	_, err := c.UpdateOrderbook(context.Background(), currency.Pair{}, asset.Empty)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = c.UpdateOrderbook(context.Background(), testPair, asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	_, err = c.UpdateOrderbook(context.Background(), currency.NewPairWithDelimiter("meow", "woof", "-"), asset.Spot)
	assert.EqualValues(t, errInvalidProductID, err.Error())
	resp, err := c.UpdateOrderbook(context.Background(), testPair, asset.Spot)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetAccountFundingHistory(context.Background())
	assert.NoError(t, err)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetWithdrawalsHistory(context.Background(), currency.NewCode("meow"), asset.Spot)
	assert.ErrorIs(t, err, errNoMatchingWallets)
	_, err = c.GetWithdrawalsHistory(context.Background(), testCrypto, asset.Spot)
	assert.NoError(t, err)
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := c.SubmitOrder(context.Background(), nil)
	assert.ErrorIs(t, err, order.ErrSubmissionIsNil)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	skipTestIfLowOnFunds(t)
	ord := order.Submit{
		Exchange:      c.Name,
		Pair:          testPair,
		AssetType:     asset.Spot,
		Side:          order.Buy,
		Type:          order.Market,
		StopDirection: order.StopUp,
		Amount:        testAmount,
		Price:         testPrice,
		RetrieveFees:  true,
		ClientOrderID: strconv.FormatInt(time.Now().UnixMilli(), 18) + "GCTSubmitOrderTest",
	}
	resp, err := c.SubmitOrder(context.Background(), &ord)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
	ord.StopDirection = order.StopDown
	ord.Side = order.Buy
	resp, err = c.SubmitOrder(context.Background(), &ord)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
	ord.Type = order.Market
	ord.QuoteAmount = testAmount
	resp, err = c.SubmitOrder(context.Background(), &ord)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := c.ModifyOrder(context.Background(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)
	var ord order.Modify
	_, err = c.ModifyOrder(context.Background(), &ord)
	assert.ErrorIs(t, err, order.ErrPairIsEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	skipTestIfLowOnFunds(t)
	ordIDs := orderTestHelper(t, "SELL")
	ord.OrderID = ordIDs.Orders[0].OrderID
	ord.Price = testPrice + 1
	resp2, err := c.ModifyOrder(context.Background(), &ord)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp2, errExpectedNonEmpty)
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	err := c.CancelOrder(context.Background(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)
	var can order.Cancel
	err = c.CancelOrder(context.Background(), &can)
	assert.ErrorIs(t, err, order.ErrIDNotSet)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	can.OrderID = "0"
	err = c.CancelOrder(context.Background(), &can)
	assert.ErrorIs(t, err, errOrderFailedToCancel)
	ordIDs := orderTestHelper(t, "")
	can.OrderID = ordIDs.Orders[0].OrderID
	err = c.CancelOrder(context.Background(), &can)
	assert.NoError(t, err)
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := c.CancelBatchOrders(context.Background(), nil)
	assert.ErrorIs(t, err, errOrderIDEmpty)
	can := make([]order.Cancel, 1)
	_, err = c.CancelBatchOrders(context.Background(), can)
	assert.ErrorIs(t, err, order.ErrIDNotSet)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	ordIDs := orderTestHelper(t, "")
	can[0].OrderID = ordIDs.Orders[0].OrderID
	resp2, err := c.CancelBatchOrders(context.Background(), can)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp2, errExpectedNonEmpty)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	ordID, err := c.GetAllOrders(context.Background(), testPair.String(), "", "", "", "", asset.Spot.Upper(), "", "", "", nil, nil, 2, time.Time{}, time.Now())
	require.NoError(t, err)
	if ordID == nil || len(ordID.Orders) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	resp, err := c.GetOrderInfo(context.Background(), ordID.Orders[0].OrderID, testPair, asset.Spot)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetDepositAddress(context.Background(), currency.NewCode("fake currency that doesn't exist"), "", "")
	assert.ErrorIs(t, err, errNoWalletForCurrency)
	resp, err := c.GetDepositAddress(context.Background(), testCrypto, "", "")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	req := withdraw.Request{}
	_, err := c.WithdrawCryptocurrencyFunds(context.Background(), &req)
	assert.ErrorIs(t, err, common.ErrExchangeNameUnset)
	req.Exchange = c.Name
	req.Currency = testCrypto
	req.Amount = testAmount
	req.Type = withdraw.Crypto
	req.Crypto.Address = testAddress
	_, err = c.WithdrawCryptocurrencyFunds(context.Background(), &req)
	assert.ErrorIs(t, err, errWalletIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	wallets, err := c.GetAllWallets(context.Background(), PaginationInp{})
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
	resp, err := c.WithdrawCryptocurrencyFunds(context.Background(), &req)
	assert.NoError(t, err)
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
	_, err := c.GetFeeByType(context.Background(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)
	var feeBuilder exchange.FeeBuilder
	feeBuilder.FeeType = exchange.OfflineTradeFee
	feeBuilder.Amount = 1
	feeBuilder.PurchasePrice = 1
	resp, err := c.GetFeeByType(context.Background(), &feeBuilder)
	assert.NoError(t, err)
	if resp != WorstCaseTakerFee {
		t.Errorf(errExpectMismatch, resp, WorstCaseTakerFee)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	_, err := c.GetActiveOrders(context.Background(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)
	var req order.MultiOrderRequest
	_, err = c.GetActiveOrders(context.Background(), &req)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	req.AssetType = asset.Spot
	req.Side = order.AnySide
	req.Type = order.AnyType
	_, err = c.GetActiveOrders(context.Background(), &req)
	assert.NoError(t, err)
	req.Pairs = req.Pairs.Add(currency.NewPair(testCrypto, testFiat))
	_, err = c.GetActiveOrders(context.Background(), &req)
	assert.NoError(t, err)
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := c.GetOrderHistory(context.Background(), nil)
	assert.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	var req order.MultiOrderRequest
	req.AssetType = asset.Spot
	req.Side = order.AnySide
	req.Type = order.AnyType
	_, err = c.GetOrderHistory(context.Background(), &req)
	assert.NoError(t, err)
	req.Pairs = req.Pairs.Add(testPair)
	_, err = c.GetOrderHistory(context.Background(), &req)
	assert.NoError(t, err)
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	_, err := c.GetHistoricCandles(context.Background(), currency.Pair{}, asset.Empty, kline.OneYear, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	resp, err := c.GetHistoricCandles(context.Background(), testPair, asset.Spot, kline.SixHour, time.Now().Add(-time.Hour*60), time.Now())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	_, err := c.GetHistoricCandlesExtended(context.Background(), currency.Pair{}, asset.Empty, kline.OneYear, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	resp, err := c.GetHistoricCandlesExtended(context.Background(), testPair, asset.Spot, kline.OneMin, time.Now().Add(-time.Hour*9), time.Now())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	err := c.ValidateAPICredentials(context.Background(), asset.Spot)
	assert.NoError(t, err)
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := c.GetServerTime(context.Background(), 0)
	assert.NoError(t, err)
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := c.GetLatestFundingRates(context.Background(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)
	req := fundingrate.LatestRateRequest{Asset: asset.UpsideProfitContract}
	_, err = c.GetLatestFundingRates(context.Background(), &req)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	req.Asset = asset.Futures
	resp, err := c.GetLatestFundingRates(context.Background(), &req)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := c.GetFuturesContractDetails(context.Background(), asset.Empty)
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset)
	_, err = c.GetFuturesContractDetails(context.Background(), asset.UpsideProfitContract)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetFuturesContractDetails(context.Background(), asset.Futures)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := c.UpdateOrderExecutionLimits(context.Background(), asset.Options)
	assert.EqualValues(t, errOptionInvalid, err.Error())
	err = c.UpdateOrderExecutionLimits(context.Background(), asset.Spot)
	assert.NoError(t, err)
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

func TestUnixTimestampUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var u UnixTimestamp
	err := u.UnmarshalJSON([]byte("0"))
	var targetErr *json.UnmarshalTypeError
	assert.ErrorAs(t, err, &targetErr)
	err = u.UnmarshalJSON([]byte("\"922337203685477580700\""))
	assert.ErrorIs(t, err, strconv.ErrRange)
	err = u.UnmarshalJSON([]byte("\"1234\""))
	assert.NoError(t, err)
}

func TestUnixTimestampString(t *testing.T) {
	t.Parallel()
	var u UnixTimestamp
	err := u.UnmarshalJSON([]byte("\"1234\""))
	assert.NoError(t, err)
	s := u.String()
	if s != expectedTimestamp {
		t.Errorf(errExpectMismatch, s, expectedTimestamp)
	}
}

func TestFormatExchangeKlineIntervalV3(t *testing.T) {
	t.Parallel()
	testSequence := map[kline.Interval]string{
		kline.OneMin:     granOneMin,
		kline.FiveMin:    granFiveMin,
		kline.FifteenMin: granFifteenMin,
		kline.ThirtyMin:  granThirtyMin,
		kline.TwoHour:    granTwoHour,
		kline.SixHour:    granSixHour,
		kline.OneDay:     granOneDay,
		kline.OneWeek:    errIntervalNotSupported}
	for k := range testSequence {
		resp := FormatExchangeKlineIntervalV3(k)
		if resp != testSequence[k] {
			t.Errorf(errExpectMismatch, resp, testSequence[k])
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
		resp, err := c.GetCurrencyTradeURL(context.Background(), a, pairs[0])
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}

func TestScheduleFuturesSweep(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	curSweeps, err := c.ListFuturesSweeps(context.Background())
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
		_, err = c.CancelPendingFuturesSweep(context.Background())
		assert.NoError(t, err)
	}
	_, err = c.ScheduleFuturesSweep(context.Background(), 0.001337)
	assert.NoError(t, err)
}

func TestCancelPendingFuturesSweep(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	curSweeps, err := c.ListFuturesSweeps(context.Background())
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
		_, err = c.ScheduleFuturesSweep(context.Background(), 0.001337)
		assert.NoError(t, err)
	}
	_, err = c.CancelPendingFuturesSweep(context.Background())
	assert.NoError(t, err)
}

// TestWsAuth dials websocket, sends login request.
func TestWsAuth(t *testing.T) {
	c.Verbose = true
	t.Parallel()
	p := currency.Pairs{testPair}
	for _, a := range c.GetAssetTypes(true) {
		require.NoError(t, c.CurrencyPairs.StorePairs(a, p, false))
		require.NoError(t, c.CurrencyPairs.StorePairs(a, p, true))
	}
	if c.Websocket.IsEnabled() && !c.API.AuthenticatedWebsocketSupport || !sharedtestvalues.AreAPICredentialsSet(c) {
		t.Skip(stream.ErrWebsocketNotEnabled.Error())
	}
	var dialer websocket.Dialer
	err := c.Websocket.Conn.Dial(&dialer, http.Header{})
	require.NoError(t, err)
	go c.wsReadData()
	err = c.Subscribe(subscription.List{
		{
			Channel: "account",
			Asset:   asset.All,
			Pairs:   p,
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

func TestStatusToStandardStatus(t *testing.T) {
	t.Parallel()
	type TestCases struct {
		Case   string
		Result order.Status
	}
	testCases := []TestCases{
		{Case: "received", Result: order.New},
		{Case: "open", Result: order.Active},
		{Case: "done", Result: order.Filled},
		{Case: "match", Result: order.PartiallyFilled},
		{Case: "change", Result: order.Active},
		{Case: "activate", Result: order.Active},
		{Case: "LOL", Result: order.UnknownStatus},
	}
	for i := range testCases {
		result, _ := statusToStandardStatus(testCases[i].Case)
		if result != testCases[i].Result {
			t.Errorf("Expected: %v, received: %v", testCases[i].Result, result)
		}
	}
}

func TestWsHandleData(t *testing.T) {
	go func() {
		for range c.Websocket.DataHandler {
			continue
		}
	}()
	mockJSON := []byte(`{"type": "error"}`)
	_, err := c.wsHandleData(mockJSON, 0)
	assert.Error(t, err)
	_, err = c.wsHandleData(nil, 0)
	assert.ErrorIs(t, err, jsonparser.KeyPathNotFoundError)
	mockJSON = []byte(`{"sequence_num": "l"}`)
	_, err = c.wsHandleData(mockJSON, 0)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	mockJSON = []byte(`{"sequence_num": 1, /\\/"""}`)
	_, err = c.wsHandleData(mockJSON, 0)
	assert.ErrorIs(t, err, jsonparser.KeyPathNotFoundError)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "subscriptions"}`)
	_, err = c.wsHandleData(mockJSON, 0)
	assert.NoError(t, err)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "", "events":}`)
	_, err = c.wsHandleData(mockJSON, 0)
	assert.ErrorIs(t, err, jsonparser.UnknownValueTypeError)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "status", "events": ["type": 1234]}`)
	_, err = c.wsHandleData(mockJSON, 0)
	var targetErr *json.SyntaxError
	assert.ErrorAs(t, err, &targetErr)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "status", "events": [{"type": "moo"}]}`)
	_, err = c.wsHandleData(mockJSON, 0)
	assert.NoError(t, err)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "ticker", "events": ["type": ""}]}`)
	_, err = c.wsHandleData(mockJSON, 0)
	assert.ErrorAs(t, err, &targetErr)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "ticker", "events": [{"type": "moo", "tickers": [{"price": "1.1"}]}]}`)
	_, err = c.wsHandleData(mockJSON, 0)
	assert.ErrorIs(t, err, jsonparser.KeyPathNotFoundError)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "ticker", "timestamp": "2006-01-02T15:04:05Z", "events": [{"type": "moo", "tickers": [{"price": "1.1"}]}]}`)
	_, err = c.wsHandleData(mockJSON, 0)
	assert.NoError(t, err)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "candles", "events": ["type": ""}]}`)
	_, err = c.wsHandleData(mockJSON, 0)
	assert.ErrorAs(t, err, &targetErr)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "candles", "events": [{"type": "moo", "candles": [{"low": "1.1"}]}]}`)
	_, err = c.wsHandleData(mockJSON, 0)
	assert.ErrorIs(t, err, jsonparser.KeyPathNotFoundError)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "candles", "timestamp": "2006-01-02T15:04:05Z", "events": [{"type": "moo", "candles": [{"low": "1.1"}]}]}`)
	_, err = c.wsHandleData(mockJSON, 0)
	assert.NoError(t, err)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "market_trades", "events": ["type": ""}]}`)
	_, err = c.wsHandleData(mockJSON, 0)
	assert.ErrorAs(t, err, &targetErr)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "market_trades", "events": [{"type": "moo", "trades": [{"price": "1.1"}]}]}`)
	_, err = c.wsHandleData(mockJSON, 0)
	assert.NoError(t, err)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "l2_data", "events": ["type": ""}]}`)
	_, err = c.wsHandleData(mockJSON, 0)
	assert.ErrorAs(t, err, &targetErr)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "l2_data", "events": [{"type": "moo", "updates": [{"price_level": "1.1"}]}]}`)
	_, err = c.wsHandleData(mockJSON, 0)
	assert.ErrorIs(t, err, jsonparser.KeyPathNotFoundError)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "l2_data", "timestamp": "2006-01-02T15:04:05Z", "events": [{"type": "moo", "updates": [{"price_level": "1.1"}]}]}`)
	_, err = c.wsHandleData(mockJSON, 0)
	assert.ErrorIs(t, err, errUnknownL2DataType)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "l2_data", "timestamp": "2006-01-02T15:04:05Z", "events": [{"type": "snapshot", "product_id": "BTC-USD", "updates": [{"side": "bid", "price_level": "1.1", "new_quantity": "2.2"}]}]}`)
	_, err = c.wsHandleData(mockJSON, 0)
	assert.NoError(t, err)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "l2_data", "timestamp": "2006-01-02T15:04:05Z", "events": [{"type": "update", "product_id": "BTC-USD", "updates": [{"side": "bid", "price_level": "1.1", "new_quantity": "2.2"}]}]}`)
	_, err = c.wsHandleData(mockJSON, 0)
	assert.NoError(t, err)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "user", "events": ["type": ""}]}`)
	_, err = c.wsHandleData(mockJSON, 0)
	assert.ErrorAs(t, err, &targetErr)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "user", "events": [{"type": "moo", "orders": [{"total_fees": "1.1"}]}]}`)
	_, err = c.wsHandleData(mockJSON, 0)
	assert.ErrorIs(t, err, order.ErrUnrecognisedOrderType)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "user", "events": [{"type": "moo", "orders": [{"total_fees": "1.1", "order_type": "ioc"}]}]}`)
	_, err = c.wsHandleData(mockJSON, 0)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "user", "events": [{"type": "moo", "orders": [{"total_fees": "1.1", "order_type": "ioc", "order_side": "buy"}]}]}`)
	_, err = c.wsHandleData(mockJSON, 0)
	assert.ErrorIs(t, err, errUnrecognisedStatusType)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "user", "events": [{"type": "moo", "orders": [{"total_fees": "1.1", "order_type": "ioc", "order_side": "buy", "status": "done"}]}]}`)
	_, err = c.wsHandleData(mockJSON, 0)
	assert.NoError(t, err)
	mockJSON = []byte(`{"sequence_num": 0, "channel": "fakechan", "events": ["type": ""}]}`)
	_, err = c.wsHandleData(mockJSON, 0)
	assert.ErrorIs(t, err, errChannelNameUnknown)
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
	p, err := c.GetEnabledPairs(asset.Spot)
	require.NoError(t, err)
	exp := subscription.List{}
	for _, baseSub := range defaultSubscriptions.Enabled() {
		s := baseSub.Clone()
		s.QualifiedChannel = subscriptionNames[s.Channel]
		if s.Asset != asset.Empty {
			s.Pairs = p
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
	_, err := c.GetJWT(context.Background(), "")
	assert.NoError(t, err)
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

func skipTestIfLowOnFunds(t *testing.T) {
	t.Helper()
	accounts, err := c.GetAllAccounts(context.Background(), 250, "")
	assert.NoError(t, err)
	if accounts == nil || len(accounts.Accounts) == 0 {
		t.Fatal(errExpectedNonEmpty)
	}
	var hasValidFunds bool
	for i := range accounts.Accounts {
		if accounts.Accounts[i].Currency == testCrypto.String() &&
			accounts.Accounts[i].AvailableBalance.Value > testAmount*100 {
			hasValidFunds = true
		}
	}
	if !hasValidFunds {
		t.Skip(skipInsufficientFunds)
	}
}

func portfolioIDFromName(t *testing.T, targetName string) string {
	t.Helper()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	createResp, err := c.CreatePortfolio(context.Background(), targetName)
	var targetID string
	if err != nil {
		assert.EqualValues(t, errPortfolioNameDuplicate, err.Error())
		getResp, err := c.GetAllPortfolios(context.Background(), "")
		assert.NoError(t, err)
		if len(getResp) == 0 {
			t.Fatal(errExpectedNonEmpty)
		}
		for i := range getResp {
			if getResp[i].Name == targetName {
				targetID = getResp[i].UUID
				break
			}
		}
	} else {
		targetID = createResp.UUID
	}
	return targetID
}

func getINTXPortfolio(t *testing.T) string {
	t.Helper()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetAllPortfolios(context.Background(), "")
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
	accIDs, err := c.GetAllAccounts(context.Background(), 250, "")
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
	pmID, err := c.GetAllPaymentMethods(context.Background())
	assert.NoError(t, err)
	if pmID == nil || len(pmID) == 0 {
		t.Skip(skipPayMethodNotFound)
	}
	return srcWalletID, pmID[0].ID
}

type withdrawFiatFunc func(context.Context, *withdraw.Request) (*withdraw.ExchangeResponse, error)

func withdrawFiatFundsHelper(t *testing.T, fn withdrawFiatFunc) {
	t.Helper()
	t.Parallel()
	req := withdraw.Request{}
	_, err := fn(context.Background(), &req)
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
	_, err = fn(context.Background(), &req)
	assert.ErrorIs(t, err, errWalletIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	req.WalletID = "meow"
	req.Fiat.Bank.BankName = "GCT's Officially Fake and Not Real Test Bank"
	_, err = fn(context.Background(), &req)
	assert.ErrorIs(t, err, errPayMethodNotFound)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	wallets, err := c.GetAllWallets(context.Background(), PaginationInp{})
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
	resp, err := fn(context.Background(), &req)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

type getNoArgsResp interface {
	*ServerTimeV3 | []PaymentMethodData | *UserResponse | []FiatData | []CryptoData | *ServerTimeV2 | []CurrencyData | []PairVolumeData | *AllWrappedAssets
}

type getNoArgsAssertNotEmpty[G getNoArgsResp] func(context.Context) (G, error)

func testGetNoArgs[G getNoArgsResp](t *testing.T, f getNoArgsAssertNotEmpty[G]) {
	t.Helper()
	resp, err := f(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

type genConvertTestFunc func(context.Context, string, string, string) (*ConvertResponse, error)

func convertTestShared(t *testing.T, f genConvertTestFunc) {
	t.Helper()
	t.Parallel()
	_, err := f(context.Background(), "", "", "")
	assert.ErrorIs(t, err, errTransactionIDEmpty)
	_, err = f(context.Background(), "meow", "", "")
	assert.ErrorIs(t, err, errAccountIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	fromAccID, toAccID := convertTestHelper(t)
	resp, err := c.CreateConvertQuote(context.Background(), fromAccID, toAccID, "", "", 0.01)
	assert.NoError(t, err)
	require.NotNil(t, resp)
	resp, err = f(context.Background(), resp.ID, fromAccID, toAccID)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

type getOneArgResp interface {
	*CurrencyData | *PairData | *ProductStats | *ProductTicker | *WrappedAsset | *WrappedAssetConversionRate
}

type getOneArgAssertNotEmpty[G getOneArgResp] func(context.Context, string) (G, error)

func testGetOneArg[G getOneArgResp](t *testing.T, f getOneArgAssertNotEmpty[G], arg string, tarErr error) {
	t.Helper()
	_, err := f(context.Background(), "")
	assert.ErrorIs(t, err, tarErr)
	resp, err := f(context.Background(), arg)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}
