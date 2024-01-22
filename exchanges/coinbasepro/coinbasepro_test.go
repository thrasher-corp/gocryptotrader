package coinbasepro

import (
	"context"
	"errors"
	"strconv"

	"fmt"
	"log"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
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
	gctlog "github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

var (
	c          = &CoinbasePro{}
	testCrypto = currency.BTC
	testFiat   = currency.USD
	testPair   = currency.NewPairWithDelimiter(testCrypto.String(), testFiat.String(), "-")
)

// Please supply your APIKeys here for better testing
const (
	apiKey    = ""
	apiSecret = ""
	// clientID                = "" // passphrase you made at API CREATION, might not exist any more
	canManipulateRealOrders = false
	testingInSandbox        = false
)

// Constants used within tests
const (
	// Donation address
	testAddress = "bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc"

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
	errOrder0CancelFail        = "order 0 failed to cancel"
	errIDNotSet                = "ID not set"
	errx7f                     = "setting proxy address error parse \"\\x7f\": net/url: invalid control character in URL"
	errPortfolioNameDuplicate  = `CoinbasePro unsuccessful HTTP status code: 409 raw response: {"error":"CONFLICT","error_details":"[PORTFOLIO_ERROR_CODE_ALREADY_EXISTS] the requested portfolio name already exists","message":"[PORTFOLIO_ERROR_CODE_ALREADY_EXISTS] the requested portfolio name already exists"}, authenticated request failed`
	errPortTransferInsufFunds  = `CoinbasePro unsuccessful HTTP status code: 429 raw response: {"error":"unknown","error_details":"[PORTFOLIO_ERROR_CODE_INSUFFICIENT_FUNDS] insufficient funds in source account","message":"[PORTFOLIO_ERROR_CODE_INSUFFICIENT_FUNDS] insufficient funds in source account"}, authenticated request failed`
	errInvalidProductID        = `CoinbasePro unsuccessful HTTP status code: 400 raw response: {"error":"INVALID_ARGUMENT","error_details":"valid product_id is required","message":"valid product_id is required"}, authenticated request failed`
	errFeeBuilderNil           = "*exchange.FeeBuilder nil pointer"
	errUnsupportedAssetType    = " unsupported asset type"
	errUpsideUnsupported       = "unsupported asset type upsideprofitcontract"
	errBlorboGranularity       = "invalid granularity blorbo, allowed granularities are: [ONE_MINUTE FIVE_MINUTE FIFTEEN_MINUTE THIRTY_MINUTE ONE_HOUR TWO_HOUR SIX_HOUR ONE_DAY]"
	errNoEndpointPathEdgeCase3 = "no endpoint path found for the given key: EdgeCase3URL"
	errJsonUnsupportedChan     = "json: unsupported type: chan struct {}, authenticated request failed"
	errExpectedFeeRange        = "expected fee range of %v and %v, received %v"
	errJsonNumberIntoString    = "json: cannot unmarshal number into Go value of type string"
	errParseIntValueOutOfRange = `strconv.ParseInt: parsing "922337203685477580700": value out of range`

	expectedTimestamp = "1970-01-01 00:20:34 +0000 UTC"

	testAmount = 1e-08
	testPrice  = 1e+09
)

func TestGetDefaultConfig(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetDefaultConfig(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestSetup(t *testing.T) {
	err := c.Setup(nil)
	if !errors.Is(err, config.ErrExchangeConfigIsNil) {
		t.Errorf(errExpectMismatch, err, config.ErrExchangeConfigIsNil)
	}
	cfg, err := c.GetStandardConfig()
	if err != nil {
		t.Error(err)
	}
	cfg.API.AuthenticatedSupport = true
	cfg.API.Credentials.Key = apiKey
	cfg.API.Credentials.Secret = apiSecret
	cfg.Enabled = false
	_ = c.Setup(cfg)
	cfg.Enabled = true
	cfg.ProxyAddress = string(rune(0x7f))
	err = c.Setup(cfg)
	if err.Error() != errx7f {
		t.Errorf(errExpectMismatch, err, errx7f)
	}
}

func TestWrapperStart(t *testing.T) {
	wg := sync.WaitGroup{}
	err := c.Start(context.Background(), &wg)
	if err != nil {
		t.Error(err)
	}
}

func TestMain(m *testing.M) {
	c.SetDefaults()
	if testingInSandbox {
		c.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
			exchange.RestSpot: coinbaseproSandboxAPIURL,
		})
	}
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("load config error", err)
	}
	gdxConfig, err := cfg.GetExchangeConfig("CoinbasePro")
	if err != nil {
		log.Fatal("init error")
	}
	if apiKey != "" {
		gdxConfig.API.Credentials.Key = apiKey
		gdxConfig.API.Credentials.Secret = apiSecret
		// gdxConfig.API.Credentials.ClientID = clientID
		gdxConfig.API.AuthenticatedSupport = true
		gdxConfig.API.AuthenticatedWebsocketSupport = true
	}
	c.Websocket = sharedtestvalues.NewTestWebsocket()
	err = c.Setup(gdxConfig)
	if err != nil {
		log.Fatal("CoinbasePro setup error", err)
	}
	if apiKey != "" {
		c.GetBase().API.AuthenticatedSupport = true
		c.GetBase().API.AuthenticatedWebsocketSupport = true
	}
	c.Verbose = true
	err = gctlog.SetGlobalLogConfig(gctlog.GenDefaultSettings())
	fmt.Println(err)
	os.Exit(m.Run())
}

func TestStart(t *testing.T) {
	t.Parallel()
	err := c.Start(context.Background(), nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Fatalf(errExpectMismatch, err, common.ErrNilPointer)
	}
	var testWg sync.WaitGroup
	err = c.Start(context.Background(), &testWg)
	if err != nil {
		t.Fatal(err)
	}
	testWg.Wait()
}

// func TestSendAuthenticatedHTTPRequest(t *testing.T) {
// 	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
// 	var resp interface{}
// 	err := c.SendAuthenticatedHTTPRequest(context.Background(), exchange.RestSpot, "", "", "", nil, &resp, nil)
// 	if err != nil {
// 		t.Error("SendAuthenticatedHTTPRequest() error", err)
// 	}
// 	log.Printf("%+v", resp)
// }

func TestGetAllAccounts(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetAllAccounts(context.Background(), 50, "")
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAccountByID(t *testing.T) {
	_, err := c.GetAccountByID(context.Background(), "")
	if !errors.Is(err, errAccountIDEmpty) {
		t.Errorf(errExpectMismatch, err, errAccountIDEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	longResp, err := c.GetAllAccounts(context.Background(), 49, "")
	if err != nil {
		t.Error(err)
	}
	if len(longResp.Accounts) == 0 {
		t.Fatal(errExpectedNonEmpty)
	}
	shortResp, err := c.GetAccountByID(context.Background(), longResp.Accounts[0].UUID)
	if err != nil {
		t.Error(err)
	}
	if *shortResp != longResp.Accounts[0] {
		t.Errorf(errExpectMismatch, shortResp, longResp.Accounts[0])
	}
}

func TestGetBestBidAsk(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	testPairs := []string{testPair.String(), "ETH-USD"}
	resp, err := c.GetBestBidAsk(context.Background(), testPairs)
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetProductBook(t *testing.T) {
	_, err := c.GetProductBook(context.Background(), "", 0)
	if !errors.Is(err, errProductIDEmpty) {
		t.Errorf(errExpectMismatch, err, errProductIDEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetProductBook(context.Background(), testPair.String(), 2)
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAllProducts(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	testPairs := []string{testPair.String(), "ETH-USD"}
	// var testPairs []string
	resp, err := c.GetAllProducts(context.Background(), 30000, 0, "SPOT", "PERPETUAL", "",
		testPairs)
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
	// log.Printf("%+v\n%+v", resp.NumProducts, len(resp.Products))
}

func TestGetProductByID(t *testing.T) {
	_, err := c.GetProductByID(context.Background(), "")
	if !errors.Is(err, errProductIDEmpty) {
		t.Errorf(errExpectMismatch, err, errProductIDEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetProductByID(context.Background(), testPair.String())
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetHistoricRates(t *testing.T) {
	_, err := c.GetHistoricRates(context.Background(), "", granUnknown, time.Time{}, time.Time{})
	if !errors.Is(err, errProductIDEmpty) {
		t.Errorf(errExpectMismatch, err, errProductIDEmpty)
	}
	_, err = c.GetHistoricRates(context.Background(), testPair.String(), "blorbo", time.Time{}, time.Time{})
	if err.Error() != errBlorboGranularity {
		t.Errorf(errExpectMismatch, err, errBlorboGranularity)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetHistoricRates(context.Background(), testPair.String(), granOneMin,
		time.Now().Add(-5*time.Minute), time.Now())
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetTicker(t *testing.T) {
	_, err := c.GetTicker(context.Background(), "", 1, time.Time{}, time.Time{})
	if !errors.Is(err, errProductIDEmpty) {
		t.Errorf(errExpectMismatch, err, errProductIDEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetTicker(context.Background(), testPair.String(), 5, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestPlaceOrder(t *testing.T) {
	_, err := c.PlaceOrder(context.Background(), "", "", "", "", "", "", "", "", 0, 0, 0, 0, false, time.Time{})
	if !errors.Is(err, errClientOrderIDEmpty) {
		t.Errorf(errExpectMismatch, err, errClientOrderIDEmpty)
	}
	_, err = c.PlaceOrder(context.Background(), "meow", "", "", "", "", "", "", "", 0, 0, 0, 0, false, time.Time{})
	if !errors.Is(err, errProductIDEmpty) {
		t.Errorf(errExpectMismatch, err, errProductIDEmpty)
	}
	_, err = c.PlaceOrder(context.Background(), "meow", testPair.String(), order.Sell.String(), "", "", "", "", "", 0,
		0, 0, 0, false, time.Time{})
	if !errors.Is(err, errAmountEmpty) {
		t.Errorf(errExpectMismatch, err, errAmountEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	skipTestIfLowOnFunds(t)
	id, err := uuid.NewV4()
	if err != nil {
		t.Error(err)
	}
	resp, err := c.PlaceOrder(context.Background(), id.String(), testPair.String(), order.Sell.String(), "",
		order.Limit.String(), "", "CROSS", "", testAmount, testPrice, 0, 9999, false, time.Now().Add(time.Hour))
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
	id, err = uuid.NewV4()
	if err != nil {
		t.Error(err)
	}
	resp, err = c.PlaceOrder(context.Background(), id.String(), testPair.String(), order.Sell.String(), "",
		order.Limit.String(), "", "MULTI", "", testAmount, testPrice, 0, 9999, false, time.Now().Add(time.Hour))
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestCancelOrders(t *testing.T) {
	var OrderSlice []string
	_, err := c.CancelOrders(context.Background(), OrderSlice)
	if !errors.Is(err, errOrderIDEmpty) {
		t.Errorf(errExpectMismatch, err, errOrderIDEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	skipTestIfLowOnFunds(t)
	ordID, err := c.PlaceOrder(context.Background(), "meow", testPair.String(), order.Sell.String(), "",
		order.Limit.String(), "", "", "", testPrice, testAmount, 0, 9999, false, time.Time{})
	if err != nil {
		t.Error(err)
	}
	OrderSlice = append(OrderSlice, ordID.OrderID)
	resp, err := c.CancelOrders(context.Background(), OrderSlice)
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestEditOrder(t *testing.T) {
	_, err := c.EditOrder(context.Background(), "", 0, 0)
	if !errors.Is(err, errOrderIDEmpty) {
		t.Errorf(errExpectMismatch, err, errOrderIDEmpty)
	}
	_, err = c.EditOrder(context.Background(), "meow", 0, 0)
	if !errors.Is(err, errSizeAndPriceZero) {
		t.Errorf(errExpectMismatch, err, errSizeAndPriceZero)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	skipTestIfLowOnFunds(t)
	id, err := uuid.NewV4()
	if err != nil {
		t.Error(err)
	}
	ordID, err := c.PlaceOrder(context.Background(), id.String(), testPair.String(), order.Sell.String(), "",
		order.Limit.String(), "", "", "", testAmount, testPrice, 0, 9999, false, time.Time{})
	if err != nil {
		t.Error(err)
	}
	resp, err := c.EditOrder(context.Background(), ordID.OrderID, testAmount, testPrice*10)
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestEditOrderPreview(t *testing.T) {
	_, err := c.EditOrderPreview(context.Background(), "", 0, 0)
	if !errors.Is(err, errOrderIDEmpty) {
		t.Errorf(errExpectMismatch, err, errOrderIDEmpty)
	}
	_, err = c.EditOrderPreview(context.Background(), "meow", 0, 0)
	if !errors.Is(err, errSizeAndPriceZero) {
		t.Errorf(errExpectMismatch, err, errSizeAndPriceZero)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	skipTestIfLowOnFunds(t)
	id, err := uuid.NewV4()
	if err != nil {
		t.Error(err)
	}
	ordID, err := c.PlaceOrder(context.Background(), id.String(), testPair.String(), order.Sell.String(), "",
		order.Limit.String(), "", "", "", testAmount, testPrice, 0, 9999, false, time.Time{})
	if err != nil {
		t.Error(err)
	}
	resp, err := c.EditOrderPreview(context.Background(), ordID.OrderID, testAmount, testPrice*10)
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAllOrders(t *testing.T) {
	assets := []string{"USD"}
	status := make([]string, 2)
	_, err := c.GetAllOrders(context.Background(), "", "", "", "", "", "", "", "", "", status, assets, 0,
		time.Unix(2, 2), time.Unix(1, 1))
	if !errors.Is(err, common.ErrStartAfterEnd) {
		t.Errorf(errExpectMismatch, err, common.ErrStartAfterEnd)
	}
	status[0] = "CANCELLED"
	status[1] = "OPEN"
	_, err = c.GetAllOrders(context.Background(), "", "", "", "", "", "", "", "", "", status, assets, 0, time.Time{},
		time.Time{})
	if !errors.Is(err, errOpenPairWithOtherTypes) {
		t.Errorf(errExpectMismatch, err, errOpenPairWithOtherTypes)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	status = make([]string, 0)
	assets = make([]string, 1)
	assets[0] = testCrypto.String()
	_, err = c.GetAllOrders(context.Background(), "", "USD", "LIMIT", "SELL", "", "SPOT", "RETAIL_ADVANCED",
		"UNKNOWN_CONTRACT_EXPIRY_TYPE", "2", status, assets, 10, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
}

func TestGetFills(t *testing.T) {
	_, err := c.GetFills(context.Background(), "", "", "", time.Unix(2, 2), time.Unix(1, 1), 0)
	if !errors.Is(err, common.ErrStartAfterEnd) {
		t.Errorf(errExpectMismatch, err, common.ErrStartAfterEnd)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err = c.GetFills(context.Background(), "", testPair.String(), "", time.Unix(1, 1), time.Now(), 5)
	if err != nil {
		t.Error(err)
	}
	status := []string{"OPEN"}
	ordID, err := c.GetAllOrders(context.Background(), "", "", "", "", "", "", "", "", "", status, nil, 3, time.Time{},
		time.Time{})
	if err != nil {
		t.Error(err)
	}
	if len(ordID.Orders) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = c.GetFills(context.Background(), ordID.Orders[0].OrderID, "", "", time.Time{}, time.Time{}, 5)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderByID(t *testing.T) {
	_, err := c.GetOrderByID(context.Background(), "", "", "")
	if !errors.Is(err, errOrderIDEmpty) {
		t.Errorf(errExpectMismatch, err, errOrderIDEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	ordID, err := c.GetAllOrders(context.Background(), "", "", "", "", "", "", "", "", "", nil, nil, 10,
		time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	if len(ordID.Orders) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = c.GetOrderByID(context.Background(), ordID.Orders[0].OrderID, ordID.Orders[0].ClientOID, "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllPortfolios(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetAllPortfolios(context.Background(), "DEFAULT")
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestCreatePortfolio(t *testing.T) {
	_, err := c.CreatePortfolio(context.Background(), "")
	if !errors.Is(err, errNameEmpty) {
		t.Errorf(errExpectMismatch, err, errNameEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	_, err = c.CreatePortfolio(context.Background(), "GCT Test Portfolio")
	if err != nil && err.Error() != errPortfolioNameDuplicate {
		t.Error(err)
	}
}

func TestMovePortfolioFunds(t *testing.T) {
	_, err := c.MovePortfolioFunds(context.Background(), "", "", "", 0)
	if !errors.Is(err, errPortfolioIDEmpty) {
		t.Errorf(errExpectMismatch, err, errPortfolioIDEmpty)
	}
	_, err = c.MovePortfolioFunds(context.Background(), "", "meowPort", "woofPort", 0)
	if !errors.Is(err, errCurrencyEmpty) {
		t.Errorf(errExpectMismatch, err, errCurrencyEmpty)
	}
	_, err = c.MovePortfolioFunds(context.Background(), testCrypto.String(), "meowPort", "woofPort", 0)
	if !errors.Is(err, errAmountEmpty) {
		t.Errorf(errExpectMismatch, err, errAmountEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	portID, err := c.GetAllPortfolios(context.Background(), "")
	if err != nil {
		t.Error(err)
	}
	if len(portID.Portfolios) < 2 {
		t.Skip(skipInsufficientPortfolios)
	}
	_, err = c.MovePortfolioFunds(context.Background(), testCrypto.String(), portID.Portfolios[0].UUID, portID.Portfolios[1].UUID,
		testAmount)
	if err != nil && err.Error() != errPortTransferInsufFunds {
		t.Error(err)
	}
}

func TestGetPortfolioByID(t *testing.T) {
	_, err := c.GetPortfolioByID(context.Background(), "")
	if !errors.Is(err, errPortfolioIDEmpty) {
		t.Errorf(errExpectMismatch, err, errPortfolioIDEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	portID, err := c.GetAllPortfolios(context.Background(), "")
	if err != nil {
		t.Error(err)
	}
	if len(portID.Portfolios) == 0 {
		t.Fatal(errExpectedNonEmpty)
	}
	resp, err := c.GetPortfolioByID(context.Background(), portID.Portfolios[0].UUID)
	if err != nil {
		t.Error(err)
	}
	if resp.Breakdown.Portfolio != portID.Portfolios[0] {
		t.Errorf(errExpectMismatch, resp.Breakdown.Portfolio, portID.Portfolios[0])
	}
}

func TestDeletePortfolio(t *testing.T) {
	err := c.DeletePortfolio(context.Background(), "")
	if !errors.Is(err, errPortfolioIDEmpty) {
		t.Errorf(errExpectMismatch, err, errPortfolioIDEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)

	pID := portfolioTestHelper(t, "GCT Test Portfolio To-Delete")

	err = c.DeletePortfolio(context.Background(), pID)
	if err != nil {
		t.Error(err)
	}
}

func TestEditPortfolio(t *testing.T) {
	_, err := c.EditPortfolio(context.Background(), "", "")
	if !errors.Is(err, errPortfolioIDEmpty) {
		t.Errorf(errExpectMismatch, err, errPortfolioIDEmpty)
	}
	_, err = c.EditPortfolio(context.Background(), "meow", "")
	if !errors.Is(err, errNameEmpty) {
		t.Errorf(errExpectMismatch, err, errNameEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)

	pID := portfolioTestHelper(t, "GCT Test Portfolio To-Edit")

	_, err = c.EditPortfolio(context.Background(), pID, "GCT Test Portfolio Edited")
	if err != nil && err.Error() != errPortfolioNameDuplicate {
		t.Error(err)
	}
}

func TestGetFuturesBalanceSummary(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetFuturesBalanceSummary(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllFuturesPositions(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetAllFuturesPositions(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuturesPositionByID(t *testing.T) {
	_, err := c.GetFuturesPositionByID(context.Background(), "")
	if !errors.Is(err, errProductIDEmpty) {
		t.Errorf(errExpectMismatch, err, errProductIDEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err = c.GetFuturesPositionByID(context.Background(), "meow")
	if err != nil {
		t.Error(err)
	}
}

func TestScheduleFuturesSweep(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	curSweeps, err := c.ListFuturesSweeps(context.Background())
	if err != nil {
		t.Error(err)
	}
	preCancel := false
	if len(curSweeps.Sweeps) > 0 {
		for i := range curSweeps.Sweeps {
			if curSweeps.Sweeps[i].Status == "PENDING" {
				preCancel = true

			}
		}
	}
	if preCancel {
		_, err = c.CancelPendingFuturesSweep(context.Background())
		if err != nil {
			t.Error(err)
		}
	}
	_, err = c.ScheduleFuturesSweep(context.Background(), 0.001337)
	if err != nil {
		t.Error(err)
	}
}

func TestListFuturesSweeps(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.ListFuturesSweeps(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestCancelPendingFuturesSweep(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	curSweeps, err := c.ListFuturesSweeps(context.Background())
	if err != nil {
		t.Error(err)
	}
	partialSkip := false
	if len(curSweeps.Sweeps) > 0 {
		for i := range curSweeps.Sweeps {
			if curSweeps.Sweeps[i].Status == "PENDING" {
				partialSkip = true

			}
		}
	}
	if !partialSkip {
		_, err = c.ScheduleFuturesSweep(context.Background(), 0.001337)
		if err != nil {
			t.Error(err)
		}

	}
	_, err = c.CancelPendingFuturesSweep(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetTransactionSummary(t *testing.T) {
	_, err := c.GetTransactionSummary(context.Background(), time.Unix(2, 2), time.Unix(1, 1), "", "", "")
	if !errors.Is(err, common.ErrStartAfterEnd) {
		t.Errorf(errExpectMismatch, err, common.ErrStartAfterEnd)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetTransactionSummary(context.Background(), time.Unix(1, 1), time.Now(), "", asset.Spot.Upper(),
		"UNKNOWN_CONTRACT_EXPIRY_TYPE")
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestCreateConvertQuote(t *testing.T) {
	_, err := c.CreateConvertQuote(context.Background(), "", "", "", "", 0)
	if !errors.Is(err, errAccountIDEmpty) {
		t.Errorf(errExpectMismatch, err, errAccountIDEmpty)
	}
	_, err = c.CreateConvertQuote(context.Background(), "meow", "123", "", "", 0)
	if !errors.Is(err, errAmountEmpty) {
		t.Errorf(errExpectMismatch, err, errAmountEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	fromAccID, toAccID := convertTestHelper(t)
	resp, err := c.CreateConvertQuote(context.Background(), fromAccID, toAccID, "", "", 0.01)
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestCommitConvertTrade(t *testing.T) {
	_, err := c.CommitConvertTrade(context.Background(), "", "", "")
	if !errors.Is(err, errTransactionIDEmpty) {
		t.Errorf(errExpectMismatch, err, errTransactionIDEmpty)
	}
	_, err = c.CommitConvertTrade(context.Background(), "meow", "", "")
	if !errors.Is(err, errAccountIDEmpty) {
		t.Errorf(errExpectMismatch, err, errAccountIDEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	fromAccID, toAccID := convertTestHelper(t)
	resp, err := c.CreateConvertQuote(context.Background(), fromAccID, toAccID, "", "", 0.01)
	if err != nil {
		t.Error(err)
	}
	resp, err = c.CommitConvertTrade(context.Background(), resp.Trade.ID, fromAccID, toAccID)
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetConvertTradeByID(t *testing.T) {
	_, err := c.GetConvertTradeByID(context.Background(), "", "", "")
	if !errors.Is(err, errTransactionIDEmpty) {
		t.Errorf(errExpectMismatch, err, errTransactionIDEmpty)
	}
	_, err = c.GetConvertTradeByID(context.Background(), "meow", "", "")
	if !errors.Is(err, errAccountIDEmpty) {
		t.Errorf(errExpectMismatch, err, errAccountIDEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	fromAccID, toAccID := convertTestHelper(t)
	resp, err := c.CreateConvertQuote(context.Background(), fromAccID, toAccID, "", "", 0.01)
	if err != nil {
		t.Error(err)
	}
	resp, err = c.GetConvertTradeByID(context.Background(), resp.Trade.ID, fromAccID, toAccID)
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetV3Time(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetV3Time(context.Background())
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestListNotifications(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.ListNotifications(context.Background(), PaginationInp{})
	if err != nil {
		t.Error(err)
	}
}

func TestGetUserByID(t *testing.T) {
	_, err := c.GetUserByID(context.Background(), "")
	if !errors.Is(err, errUserIDEmpty) {
		t.Errorf(errExpectMismatch, err, errUserIDEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetCurrentUser(context.Background())
	if err != nil {
		t.Error(err)
	}
	if resp == nil {
		t.Fatal(errExpectedNonEmpty)
	}
	resp, err = c.GetUserByID(context.Background(), resp.Data.ID)
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetCurrentUser(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetCurrentUser(context.Background())
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAuthInfo(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetAuthInfo(context.Background())
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestUpdateUser(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	oldData, err := c.GetCurrentUser(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.UpdateUser(context.Background(), "Name changed as per GCT testing", "Sydney", testFiat.String())
	if err != nil {
		t.Error(err)
	}
	resp, err := c.UpdateUser(context.Background(), oldData.Data.Name, oldData.Data.TimeZone, oldData.Data.NativeCurrency)
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestCreateWallet(t *testing.T) {
	_, err := c.CreateWallet(context.Background(), "")
	if !errors.Is(err, errCurrencyEmpty) {
		t.Errorf(errExpectMismatch, err, errCurrencyEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	resp, err := c.CreateWallet(context.Background(), testCrypto.String())
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAllWallets(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	pagIn := PaginationInp{Limit: 2}
	resp, err := c.GetAllWallets(context.Background(), pagIn)
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
	if resp.Pagination.NextStartingAfter == "" {
		t.Skip(skipInsufficientWallets)
	}
	pagIn.StartingAfter = resp.Pagination.NextStartingAfter
	resp, err = c.GetAllWallets(context.Background(), pagIn)
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetWalletByID(t *testing.T) {
	_, err := c.GetWalletByID(context.Background(), "", "")
	if !errors.Is(err, errCurrWalletConflict) {
		t.Errorf(errExpectMismatch, err, errCurrWalletConflict)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetWalletByID(context.Background(), "", testCrypto.String())
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
	resp, err = c.GetWalletByID(context.Background(), resp.Data.ID, "")
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestUpdateWalletName(t *testing.T) {
	_, err := c.UpdateWalletName(context.Background(), "", "")
	if !errors.Is(err, errWalletIDEmpty) {
		t.Errorf(errExpectMismatch, err, errWalletIDEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	wID, err := c.GetAllWallets(context.Background(), PaginationInp{})
	if err != nil {
		t.Error(err)
	}
	if len(wID.Data) == 0 {
		t.Fatal(errExpectedNonEmpty)
	}
	resp, err := c.UpdateWalletName(context.Background(), wID.Data[len(wID.Data)-1].ID, "Wallet Tested by GCT")
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestDeleteWallet(t *testing.T) {
	err := c.DeleteWallet(context.Background(), "")
	if !errors.Is(err, errWalletIDEmpty) {
		t.Errorf(errExpectMismatch, err, errWalletIDEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	wID, err := c.CreateWallet(context.Background(), testCrypto.String())
	if err != nil {
		t.Error(err)
	}
	// As of now, it seems like this next step always fails. DeleteWallet only lets you delete non-primary
	// non-fiat wallets, but the only non-primary wallet is fiat. Trying to create a secondary wallet for
	// any cryptocurrency using CreateWallet simply returns the details of the existing primary wallet.
	t.Skip("endpoint bugged on their end, skipping")
	err = c.DeleteWallet(context.Background(), wID.Data.ID)
	if err != nil {
		t.Error(err)
	}
}

func TestCreateAddress(t *testing.T) {
	_, err := c.CreateAddress(context.Background(), "", "")
	if !errors.Is(err, errWalletIDEmpty) {
		t.Errorf(errExpectMismatch, err, errWalletIDEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	wID, err := c.GetWalletByID(context.Background(), "", testCrypto.String())
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, wID, errExpectedNonEmpty)
	resp, err := c.CreateAddress(context.Background(), wID.Data.ID, "")
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAllAddresses(t *testing.T) {
	var pag PaginationInp
	_, err := c.GetAllAddresses(context.Background(), "", pag)
	if !errors.Is(err, errWalletIDEmpty) {
		t.Errorf(errExpectMismatch, err, errWalletIDEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	wID, err := c.GetWalletByID(context.Background(), "", testCrypto.String())
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, wID, errExpectedNonEmpty)
	resp, err := c.GetAllAddresses(context.Background(), wID.Data.ID, pag)
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAddressByID(t *testing.T) {
	_, err := c.GetAddressByID(context.Background(), "", "")
	if !errors.Is(err, errWalletIDEmpty) {
		t.Errorf(errExpectMismatch, err, errWalletIDEmpty)
	}
	_, err = c.GetAddressByID(context.Background(), "123", "")
	if !errors.Is(err, errAddressIDEmpty) {
		t.Errorf(errExpectMismatch, err, errAddressIDEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	wID, err := c.GetWalletByID(context.Background(), "", testCrypto.String())
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, wID, errExpectedNonEmpty)
	addID, err := c.GetAllAddresses(context.Background(), wID.Data.ID, PaginationInp{})
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, addID, errExpectedNonEmpty)
	resp, err := c.GetAddressByID(context.Background(), wID.Data.ID, addID.Data[0].ID)
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAddressTransactions(t *testing.T) {
	_, err := c.GetAddressTransactions(context.Background(), "", "", PaginationInp{})
	if !errors.Is(err, errWalletIDEmpty) {
		t.Errorf(errExpectMismatch, err, errWalletIDEmpty)
	}
	_, err = c.GetAddressTransactions(context.Background(), "123", "", PaginationInp{})
	if !errors.Is(err, errAddressIDEmpty) {
		t.Errorf(errExpectMismatch, err, errAddressIDEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	wID, err := c.GetWalletByID(context.Background(), "", testCrypto.String())
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, wID, errExpectedNonEmpty)
	addID, err := c.GetAllAddresses(context.Background(), wID.Data.ID, PaginationInp{})
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, addID, errExpectedNonEmpty)
	_, err = c.GetAddressTransactions(context.Background(), wID.Data.ID, addID.Data[0].ID, PaginationInp{})
	if err != nil {
		t.Error(err)
	}
}

func TestSendMoney(t *testing.T) {
	_, err := c.SendMoney(context.Background(), "", "", "", "", "", "", "", "", 0, false, false)
	if !errors.Is(err, errTransactionTypeEmpty) {
		t.Errorf(errExpectMismatch, err, errTransactionTypeEmpty)
	}
	_, err = c.SendMoney(context.Background(), "123", "", "", "", "", "", "", "", 0, false, false)
	if !errors.Is(err, errWalletIDEmpty) {
		t.Errorf(errExpectMismatch, err, errWalletIDEmpty)
	}
	_, err = c.SendMoney(context.Background(), "123", "123", "", "", "", "", "", "", 0, false, false)
	if !errors.Is(err, errToEmpty) {
		t.Errorf(errExpectMismatch, err, errToEmpty)
	}
	_, err = c.SendMoney(context.Background(), "123", "123", "123", "", "", "", "", "", 0, false, false)
	if !errors.Is(err, errAmountEmpty) {
		t.Errorf(errExpectMismatch, err, errAmountEmpty)
	}
	_, err = c.SendMoney(context.Background(), "123", "123", "123", "", "", "", "", "", 1, false, false)
	if !errors.Is(err, errCurrencyEmpty) {
		t.Errorf(errExpectMismatch, err, errCurrencyEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	wID, err := c.GetAllWallets(context.Background(), PaginationInp{})
	if err != nil {
		t.Error(err)
	}
	if len(wID.Data) < 2 {
		t.Skip(skipInsufficientWallets)
	}
	var (
		fromID string
		toID   string
	)
	for i := range wID.Data {
		if wID.Data[i].Currency.Name == testCrypto.String() {
			if wID.Data[i].Balance.Amount > testAmount {
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
	_, err = c.SendMoney(context.Background(), "transfer", wID.Data[0].ID, wID.Data[1].ID,
		testCrypto.String(), "GCT Test", "123", "", "", testAmount, false, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllTransactions(t *testing.T) {
	var pag PaginationInp
	_, err := c.GetAllTransactions(context.Background(), "", pag)
	if !errors.Is(err, errWalletIDEmpty) {
		t.Errorf(errExpectMismatch, err, errWalletIDEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	wID, err := c.GetWalletByID(context.Background(), "", testCrypto.String())
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, wID, errExpectedNonEmpty)
	_, err = c.GetAllTransactions(context.Background(), wID.Data.ID, pag)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTransactionByID(t *testing.T) {
	_, err := c.GetTransactionByID(context.Background(), "", "")
	if !errors.Is(err, errWalletIDEmpty) {
		t.Errorf(errExpectMismatch, err, errWalletIDEmpty)
	}
	_, err = c.GetTransactionByID(context.Background(), "123", "")
	if !errors.Is(err, errTransactionIDEmpty) {
		t.Errorf(errExpectMismatch, err, errTransactionIDEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	wID, err := c.GetWalletByID(context.Background(), "", testCrypto.String())
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, wID, errExpectedNonEmpty)
	tID, err := c.GetAllTransactions(context.Background(), wID.Data.ID, PaginationInp{})
	if err != nil {
		t.Error(err)
	}
	if len(tID.Data) == 0 {
		t.Skip(skipInsufficientTransactions)
	}
	resp, err := c.GetTransactionByID(context.Background(), wID.Data.ID, tID.Data[0].ID)
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestFiatTransfer(t *testing.T) {
	_, err := c.FiatTransfer(context.Background(), "", "", "", 0, false, FiatDeposit)
	if !errors.Is(err, errWalletIDEmpty) {
		t.Errorf(errExpectMismatch, err, errWalletIDEmpty)
	}
	_, err = c.FiatTransfer(context.Background(), "123", "", "", 0, false, FiatDeposit)
	if !errors.Is(err, errAmountEmpty) {
		t.Errorf(errExpectMismatch, err, errAmountEmpty)
	}
	_, err = c.FiatTransfer(context.Background(), "123", "", "", 1, false, FiatDeposit)
	if !errors.Is(err, errCurrencyEmpty) {
		t.Errorf(errExpectMismatch, err, errCurrencyEmpty)
	}
	_, err = c.FiatTransfer(context.Background(), "123", "123", "", 1, false, FiatDeposit)
	if !errors.Is(err, errPaymentMethodEmpty) {
		t.Errorf(errExpectMismatch, err, errPaymentMethodEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	wallets, err := c.GetAllWallets(context.Background(), PaginationInp{})
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, wallets, errExpectedNonEmpty)
	wID, pmID := transferTestHelper(t, wallets)
	_, err = c.FiatTransfer(context.Background(), wID, testFiat.String(), pmID, testAmount, false, FiatDeposit)
	if err != nil {
		t.Error(err)
	}
	_, err = c.FiatTransfer(context.Background(), wID, testFiat.String(), pmID, testAmount, false, FiatWithdrawal)
	if err != nil {
		t.Error(err)
	}
}

func TestCommitTransfer(t *testing.T) {
	_, err := c.CommitTransfer(context.Background(), "", "", FiatDeposit)
	if !errors.Is(err, errWalletIDEmpty) {
		t.Errorf(errExpectMismatch, err, errWalletIDEmpty)
	}
	_, err = c.CommitTransfer(context.Background(), "123", "", FiatDeposit)
	if !errors.Is(err, errDepositIDEmpty) {
		t.Errorf(errExpectMismatch, err, errDepositIDEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	wallets, err := c.GetAllWallets(context.Background(), PaginationInp{})
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, wallets, errExpectedNonEmpty)
	wID, pmID := transferTestHelper(t, wallets)
	depID, err := c.FiatTransfer(context.Background(), wID, testFiat.String(), pmID, testAmount,
		false, FiatDeposit)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.CommitTransfer(context.Background(), wID, depID.Data.ID, FiatDeposit)
	if err != nil {
		t.Error(err)
	}
	depID, err = c.FiatTransfer(context.Background(), wID, testFiat.String(), pmID, testAmount,
		false, FiatWithdrawal)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.CommitTransfer(context.Background(), wID, depID.Data.ID, FiatWithdrawal)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllFiatTransfers(t *testing.T) {
	var pag PaginationInp
	_, err := c.GetAllFiatTransfers(context.Background(), "", pag, FiatDeposit)
	if !errors.Is(err, errWalletIDEmpty) {
		t.Errorf(errExpectMismatch, err, errWalletIDEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	wID, err := c.GetWalletByID(context.Background(), "", "AUD")
	if err != nil {
		t.Fatal(err)
	}
	assert.NotEmpty(t, wID, errExpectedNonEmpty)
	_, err = c.GetAllFiatTransfers(context.Background(), wID.Data.ID, pag, FiatDeposit)
	if err != nil {
		t.Error(err)
	}
	_, err = c.GetAllFiatTransfers(context.Background(), wID.Data.ID, pag, FiatWithdrawal)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFiatTransferByID(t *testing.T) {
	_, err := c.GetFiatTransferByID(context.Background(), "", "", FiatDeposit)
	if !errors.Is(err, errWalletIDEmpty) {
		t.Errorf(errExpectMismatch, err, errWalletIDEmpty)
	}
	_, err = c.GetFiatTransferByID(context.Background(), "123", "", FiatDeposit)
	if !errors.Is(err, errDepositIDEmpty) {
		t.Errorf(errExpectMismatch, err, errDepositIDEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	wID, err := c.GetWalletByID(context.Background(), "", "AUD")
	if err != nil {
		t.Fatal(err)
	}
	assert.NotEmpty(t, wID, errExpectedNonEmpty)
	dID, err := c.GetAllFiatTransfers(context.Background(), wID.Data.ID, PaginationInp{}, FiatDeposit)
	if err != nil {
		t.Error(err)
	}
	if len(dID.Data) == 0 {
		t.Skip(skipInsufficientTransactions)
	}
	resp, err := c.GetFiatTransferByID(context.Background(), wID.Data.ID, dID.Data[0].ID, FiatDeposit)
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
	resp, err = c.GetFiatTransferByID(context.Background(), wID.Data.ID, dID.Data[0].ID, FiatWithdrawal)
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAllPaymentMethods(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetAllPaymentMethods(context.Background(), PaginationInp{})
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetPaymentMethodByID(t *testing.T) {
	_, err := c.GetPaymentMethodByID(context.Background(), "")
	if !errors.Is(err, errPaymentMethodEmpty) {
		t.Errorf(errExpectMismatch, err, errPaymentMethodEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	pmID, err := c.GetAllPaymentMethods(context.Background(), PaginationInp{})
	if err != nil {
		t.Error(err)
	}
	if len(pmID.Data) == 0 {
		t.Skip(skipPayMethodNotFound)
	}
	resp, err := c.GetPaymentMethodByID(context.Background(), pmID.Data[0].ID)
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetFiatCurrencies(t *testing.T) {
	resp, err := c.GetFiatCurrencies(context.Background())
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetCryptocurrencies(t *testing.T) {
	resp, err := c.GetCryptocurrencies(context.Background())
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetExchangeRates(t *testing.T) {
	resp, err := c.GetExchangeRates(context.Background(), "")
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetPrice(t *testing.T) {
	_, err := c.GetPrice(context.Background(), "", "")
	if !errors.Is(err, errInvalidPriceType) {
		t.Errorf(errExpectMismatch, err, errInvalidPriceType)
	}
	resp, err := c.GetPrice(context.Background(), testPair.String(), asset.Spot.String())
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
	resp, err = c.GetPrice(context.Background(), testPair.String(), "buy")
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
	resp, err = c.GetPrice(context.Background(), testPair.String(), "sell")
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetV2Time(t *testing.T) {
	resp, err := c.GetV2Time(context.Background())
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestSendHTTPRequest(t *testing.T) {
	err := c.SendHTTPRequest(context.Background(), exchange.EdgeCase3, "", nil)
	if err.Error() != errNoEndpointPathEdgeCase3 {
		t.Errorf(errExpectMismatch, err, errNoEndpointPathEdgeCase3)
	}
}

func TestSendAuthenticatedHTTPRequest(t *testing.T) {
	fc := &CoinbasePro{}
	err := fc.SendAuthenticatedHTTPRequest(context.Background(), exchange.EdgeCase3, "", "", "", nil, false, nil, nil)
	if !errors.Is(err, exchange.ErrCredentialsAreEmpty) {
		t.Errorf(errExpectMismatch, err, exchange.ErrCredentialsAreEmpty)
	}
	err = c.SendAuthenticatedHTTPRequest(context.Background(), exchange.EdgeCase3, "", "", "", nil, false, nil, nil)
	if err.Error() != errNoEndpointPathEdgeCase3 {
		t.Errorf(errExpectMismatch, err, errNoEndpointPathEdgeCase3)
	}
	ch := make(chan struct{})
	body := map[string]interface{}{"Unmarshalable": ch}
	err = c.SendAuthenticatedHTTPRequest(context.Background(), exchange.RestSpot, "", "", "", body, false, nil, nil)
	if err.Error() != errJsonUnsupportedChan {
		t.Errorf(errExpectMismatch, err, errJsonUnsupportedChan)
	}
}

func TestGetFee(t *testing.T) {
	_, err := c.GetFee(context.Background(), nil)
	if err.Error() != errFeeBuilderNil {
		t.Errorf(errExpectMismatch, errFeeBuilderNil, err)
	}
	feeBuilder := exchange.FeeBuilder{
		FeeType:       exchange.OfflineTradeFee,
		Amount:        1,
		PurchasePrice: 1,
	}
	resp, err := c.GetFee(context.Background(), &feeBuilder)
	if err != nil {
		t.Error(err)
	}
	if resp != 0.008 {
		t.Errorf(errExpectMismatch, resp, 0.008)
	}
	feeBuilder.IsMaker = true
	resp, err = c.GetFee(context.Background(), &feeBuilder)
	if err != nil {
		t.Error(err)
	}
	if resp != 0.006 {
		t.Errorf(errExpectMismatch, resp, 0.006)
	}
	feeBuilder.Pair = currency.NewPair(currency.USDT, currency.USD)
	resp, err = c.GetFee(context.Background(), &feeBuilder)
	if err != nil {
		t.Error(err)
	}
	if resp != 0 {
		t.Errorf(errExpectMismatch, resp, 0)
	}
	feeBuilder.IsMaker = false
	resp, err = c.GetFee(context.Background(), &feeBuilder)
	if err != nil {
		t.Error(err)
	}
	if resp != 0.00001 {
		t.Errorf(errExpectMismatch, resp, 0.00001)
	}
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	_, err = c.GetFee(context.Background(), &feeBuilder)
	if err != errFeeTypeNotSupported {
		t.Errorf(errExpectMismatch, errFeeTypeNotSupported, err)
	}
	feeBuilder.Pair = currency.Pair{}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	feeBuilder.FeeType = exchange.CryptocurrencyTradeFee
	resp, err = c.GetFee(context.Background(), &feeBuilder)
	if err != nil {
		t.Error(err)
	}
	if !(resp <= 0.008 && resp >= 0.0005) {
		t.Errorf(errExpectedFeeRange, 0.0005, 0.008, resp)
	}
	feeBuilder.IsMaker = true
	resp, err = c.GetFee(context.Background(), &feeBuilder)
	if err != nil {
		t.Error(err)
	}
	if !(resp <= 0.006 && resp >= 0) {
		t.Errorf(errExpectedFeeRange, 0, 0.006, resp)
	}
}

func TestPrepareDateString(t *testing.T) {
	t.Parallel()
	var expectedResult Params
	expectedResult.urlVals = map[string][]string{
		"start_date": {"1970-01-01T00:00:01Z"},
		"end_date":   {"1970-01-01T00:00:02Z"},
	}
	var result Params

	result.urlVals = make(url.Values)

	labelStart := "start_date"
	labelEnd := "end_date"

	err := result.prepareDateString(time.Unix(1, 1).UTC(), time.Unix(2, 2).UTC(), labelStart, labelEnd)
	if err != nil {
		t.Error(err)
	}
	if fmt.Sprint(expectedResult) != fmt.Sprint(result) {
		t.Errorf(errExpectMismatch, result, expectedResult)
	}

	var newTime time.Time
	err = result.prepareDateString(newTime, newTime, labelStart, labelEnd)
	if err != nil {
		t.Error(err)
	}

	err = result.prepareDateString(time.Unix(2, 2).UTC(), time.Unix(1, 1).UTC(), labelStart, labelEnd)
	if !errors.Is(err, common.ErrStartAfterEnd) {
		t.Errorf(errExpectMismatch, err, common.ErrStartAfterEnd)
	}
}

func TestPreparePagination(t *testing.T) {
	t.Parallel()
	var expectedResult Params
	expectedResult.urlVals = map[string][]string{"limit": {"1"}, "order": {"asc"}, "starting_after": {"meow"},
		"ending_before": {"woof"}}

	var result Params
	result.urlVals = make(url.Values)

	pagIn := PaginationInp{Limit: 1, OrderAscend: true, StartingAfter: "meow", EndingBefore: "woof"}

	result.preparePagination(pagIn)

	if fmt.Sprint(expectedResult) != fmt.Sprint(result) {
		t.Errorf(errExpectMismatch, result, expectedResult)
	}
}

func TestFetchTradablePairs(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.FetchTradablePairs(context.Background(), asset.Empty)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf(errExpectMismatch, err, asset.ErrNotSupported)
	}
	_, err = c.FetchTradablePairs(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
	_, err = c.FetchTradablePairs(context.Background(), asset.Futures)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTradablePairs(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	err := c.UpdateTradablePairs(context.Background(), false)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateAccountInfo(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.UpdateAccountInfo(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchAccountInfo(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.FetchAccountInfo(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTickers(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	err := c.UpdateTickers(context.Background(), asset.Futures)
	if err != nil {
		t.Error(err)
	}
	err = c.UpdateTickers(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTicker(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.UpdateTicker(context.Background(), currency.Pair{}, asset.Empty)
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Errorf(errExpectMismatch, err, currency.ErrCurrencyPairEmpty)
	}
	_, err = c.UpdateTicker(context.Background(), testPair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchTicker(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.FetchTicker(context.Background(), testPair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchOrderbook(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.FetchOrderbook(context.Background(), testPair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	_, err := c.UpdateOrderbook(context.Background(), currency.Pair{}, asset.Empty)
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Errorf(errExpectMismatch, err, currency.ErrCurrencyPairEmpty)
	}
	_, err = c.UpdateOrderbook(context.Background(), currency.NewPairWithDelimiter("meow", "woof", "-"), asset.Spot)
	if err.Error() != errInvalidProductID {
		t.Errorf(errExpectMismatch, err, errInvalidProductID)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err = c.UpdateOrderbook(context.Background(), testPair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestProcessFundingData(t *testing.T) {
	accHist := make([]DeposWithdrData, 1)
	genAmcur := AmCur{Amount: 1, Currency: "DOGE"}
	accHist[0] = DeposWithdrData{Status: "meow", ID: "woof", PayoutAt: time.Unix(1, 1).UTC(), Amount: genAmcur,
		Fee: genAmcur, TransferType: FiatWithdrawal}
	genAmcur.Amount = 2
	cryptHist := make([]TransactionData, 2)
	cryptHist[0] = TransactionData{Status: "moo", ID: "oink", CreatedAt: time.Unix(2, 2).UTC(), Amount: genAmcur,
		Type: "receive"}
	cryptHist[0].Network.Name = "neigh"
	cryptHist[0].To.ID = "The Barnyard"
	cryptHist[1].Type = "send"

	expectedResult := make([]exchange.FundingHistory, 3)
	expectedResult[0] = exchange.FundingHistory{ExchangeName: "CoinbasePro", Status: "meow", TransferID: "woof",
		Timestamp: time.Unix(1, 1).UTC(), Currency: "DOGE", Amount: 1, Fee: 1, TransferType: "withdrawal"}
	expectedResult[1] = exchange.FundingHistory{ExchangeName: "CoinbasePro", Status: "moo", TransferID: "oink",
		Timestamp: time.Unix(2, 2).UTC(), Currency: "DOGE", Amount: 2, Fee: 0, TransferType: "deposit",
		CryptoFromAddress: "The Barnyard", CryptoChain: "neigh"}
	expectedResult[2] = exchange.FundingHistory{ExchangeName: "CoinbasePro", TransferType: "withdrawal"}

	resp := c.processFundingData(accHist, cryptHist)

	if fmt.Sprint(expectedResult) != fmt.Sprint(resp) {
		t.Errorf(errExpectMismatch, resp, expectedResult)
	}
}

func TestGetAccountFundingHistory(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetAccountFundingHistory(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawalsHistory(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetWithdrawalsHistory(context.Background(), currency.NewCode("meow"), asset.Spot)
	if !errors.Is(err, errNoMatchingWallets) {
		t.Errorf(errExpectMismatch, err, errNoMatchingWallets)
	}
	_, err = c.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitOrder(t *testing.T) {
	_, err := c.SubmitOrder(context.Background(), nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf(errExpectMismatch, err, common.ErrNilPointer)
	}
	var ord order.Submit
	_, err = c.SubmitOrder(context.Background(), &ord)
	if !errors.Is(err, common.ErrExchangeNameUnset) {
		t.Errorf(errExpectMismatch, err, common.ErrExchangeNameUnset)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	skipTestIfLowOnFunds(t)
	ord.Exchange = c.Name
	ord.Pair = testPair
	ord.AssetType = asset.Spot
	ord.Side = order.Sell
	ord.Type = order.StopLimit
	ord.StopDirection = order.StopUp
	ord.Amount = 0.0000001
	ord.Price = 1000000000000
	ord.RetrieveFees = true
	ord.ClientOrderID = strconv.FormatInt(time.Now().UnixMilli(), 18) + "GCTSubmitOrderTest"
	_, err = c.SubmitOrder(context.Background(), &ord)
	if err != nil {
		t.Error(err)
	}
	ord.StopDirection = order.StopDown
	ord.Side = order.Buy
	_, err = c.SubmitOrder(context.Background(), &ord)
	if err != nil {
		t.Error(err)
	}
	ord.Type = order.Market
	ord.QuoteAmount = 0.0000001
	_, err = c.SubmitOrder(context.Background(), &ord)
	if err != nil {
		t.Error(err)
	}
}

func TestModifyOrder(t *testing.T) {
	_, err := c.ModifyOrder(context.Background(), nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf(errExpectMismatch, err, common.ErrNilPointer)
	}
	var ord order.Modify
	_, err = c.ModifyOrder(context.Background(), &ord)
	if !errors.Is(err, order.ErrPairIsEmpty) {
		t.Errorf(errExpectMismatch, err, order.ErrPairIsEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	skipTestIfLowOnFunds(t)
	resp, err := c.PlaceOrder(context.Background(), strconv.FormatInt(time.Now().UnixMilli(), 18)+"GCTModifyOrderTest",
		testPair.String(), order.Sell.String(), "", order.Limit.String(), "", "", "", 0.0000001, 1000000000000, 0, 9999,
		false, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	ord.OrderID = resp.OrderID
	ord.Price = 1000000000001
	_, err = c.ModifyOrder(context.Background(), &ord)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelOrder(t *testing.T) {
	err := c.CancelOrder(context.Background(), nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf(errExpectMismatch, err, common.ErrNilPointer)
	}
	var can order.Cancel
	err = c.CancelOrder(context.Background(), &can)
	if err.Error() != errIDNotSet {
		t.Errorf(errExpectMismatch, err, errIDNotSet)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	can.OrderID = "0"
	c.Verbose = true
	err = c.CancelOrder(context.Background(), &can)
	if err.Error() != errOrder0CancelFail {
		t.Errorf(errExpectMismatch, err, errOrder0CancelFail)
	}
	skipTestIfLowOnFunds(t)
	resp, err := c.PlaceOrder(context.Background(), strconv.FormatInt(time.Now().UnixMilli(), 18)+"GCTCancelOrderTest",
		testPair.String(), order.Sell.String(), "", order.Limit.String(), "", "", "", 0.0000001, 1000000000000, 0, 9999,
		false, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	can.OrderID = resp.OrderID
	err = c.CancelOrder(context.Background(), &can)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelBatchOrders(t *testing.T) {
	_, err := c.CancelBatchOrders(context.Background(), nil)
	if !errors.Is(err, errOrderIDEmpty) {
		t.Errorf(errExpectMismatch, err, errOrderIDEmpty)
	}
	can := make([]order.Cancel, 1)
	_, err = c.CancelBatchOrders(context.Background(), can)
	if err.Error() != errIDNotSet {
		t.Errorf(errExpectMismatch, err, errIDNotSet)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	skipTestIfLowOnFunds(t)
	resp, err := c.PlaceOrder(context.Background(),
		strconv.FormatInt(time.Now().UnixMilli(), 18)+"GCTCancelBatchOrdersTest", testPair.String(),
		order.Sell.String(), "", order.Limit.String(), "", "", "", 0.0000001, 1000000000000, 0, 9999, false, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	can[0].OrderID = resp.OrderID
	_, err = c.CancelBatchOrders(context.Background(), can)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAllOrders(t *testing.T) {
	_, err := c.CancelAllOrders(context.Background(), nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf(errExpectMismatch, err, common.ErrNilPointer)
	}
	var can order.Cancel
	_, err = c.CancelAllOrders(context.Background(), &can)
	if !errors.Is(err, order.ErrPairIsEmpty) {
		t.Errorf(errExpectMismatch, err, order.ErrPairIsEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	skipTestIfLowOnFunds(t)
	_, err = c.PlaceOrder(context.Background(),
		strconv.FormatInt(time.Now().UnixMilli(), 18)+"GCTCancelAllOrdersTest", testPair.String(),
		order.Sell.String(), "", order.Limit.String(), "", "", "", 0.0000001, 1000000000000, 0, 9999, false, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	can.Pair = testPair
	can.AssetType = asset.Spot
	_, err = c.CancelAllOrders(context.Background(), &can)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderInfo(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	ordID, err := c.GetAllOrders(context.Background(), testPair.String(), "", "", "", "",
		asset.Spot.Upper(), "", "", "", nil, nil, 2, time.Time{}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(ordID.Orders) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = c.GetOrderInfo(context.Background(), ordID.Orders[0].OrderID, testPair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	_, err := c.GetDepositAddress(context.Background(), currency.BTC, "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	req := withdraw.Request{}
	_, err := c.WithdrawCryptocurrencyFunds(context.Background(), &req)
	if !errors.Is(err, common.ErrExchangeNameUnset) {
		t.Errorf(errExpectMismatch, err, common.ErrExchangeNameUnset)
	}
	req.Exchange = c.Name
	req.Currency = currency.BTC
	req.Amount = testAmount
	req.Type = withdraw.Crypto
	req.Crypto.Address = testAddress
	_, err = c.WithdrawCryptocurrencyFunds(context.Background(), &req)
	if !errors.Is(err, errWalletIDEmpty) {
		t.Errorf(errExpectMismatch, err, errWalletIDEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	wallets, err := c.GetAllWallets(context.Background(), PaginationInp{})
	if err != nil {
		t.Error(err)
	}
	if len(wallets.Data) == 0 {
		t.Fatal(errExpectedNonEmpty)
	}
	for i := range wallets.Data {
		if wallets.Data[i].Currency.Name == currency.BTC.String() && wallets.Data[i].Balance.Amount > testAmount {
			req.WalletID = wallets.Data[i].ID
			break
		}
	}
	if req.WalletID == "" {
		t.Skip(skipInsufficientFunds)
	}
	_, err = c.WithdrawCryptocurrencyFunds(context.Background(), &req)
	if err != nil {
		t.Error(err)
	}
}

func TestWithdrawFiatFunds(t *testing.T) {
	req := withdraw.Request{}
	_, err := c.WithdrawFiatFunds(context.Background(), &req)
	if !errors.Is(err, common.ErrExchangeNameUnset) {
		t.Errorf(errExpectMismatch, err, common.ErrExchangeNameUnset)
	}
	req.Exchange = c.Name
	req.Currency = currency.AUD
	req.Amount = 1
	req.Type = withdraw.Fiat
	req.Fiat.Bank.Enabled = true
	req.Fiat.Bank.SupportedExchanges = "CoinbasePro"
	req.Fiat.Bank.SupportedCurrencies = "AUD"
	req.Fiat.Bank.AccountNumber = "123"
	req.Fiat.Bank.SWIFTCode = "456"
	req.Fiat.Bank.BSBNumber = "789"
	_, err = c.WithdrawFiatFunds(context.Background(), &req)
	if !errors.Is(err, errWalletIDEmpty) {
		t.Errorf(errExpectMismatch, err, errWalletIDEmpty)
	}
	req.WalletID = "meow"
	req.Fiat.Bank.BankName = "GCT's Fake and Not Real Test Bank Meow Meow"
	expectedError := fmt.Sprintf(errPayMethodNotFound, req.Fiat.Bank.BankName)
	_, err = c.WithdrawFiatFunds(context.Background(), &req)
	if err.Error() != expectedError {
		t.Errorf(errExpectMismatch, err, expectedError)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	wallets, err := c.GetAllWallets(context.Background(), PaginationInp{})
	if err != nil {
		t.Error(err)
	}
	if len(wallets.Data) == 0 {
		t.Fatal(errExpectedNonEmpty)
	}
	req.WalletID = ""
	for i := range wallets.Data {
		if wallets.Data[i].Currency.Name == currency.AUD.String() && wallets.Data[i].Balance.Amount > testAmount {
			req.WalletID = wallets.Data[i].ID
			break
		}
	}
	if req.WalletID == "" {
		t.Skip(skipInsufficientFunds)
	}
	req.Fiat.Bank.BankName = "AUD Wallet"
	_, err = c.WithdrawFiatFunds(context.Background(), &req)
	if err != nil {
		t.Error(err)
	}
}

func TestWithdrawFiatFundsToInternationalBank(t *testing.T) {
	req := withdraw.Request{}
	_, err := c.WithdrawFiatFundsToInternationalBank(context.Background(), &req)
	if !errors.Is(err, common.ErrExchangeNameUnset) {
		t.Errorf(errExpectMismatch, err, common.ErrExchangeNameUnset)
	}
}

func TestGetFeeByType(t *testing.T) {
	_, err := c.GetFeeByType(context.Background(), nil)
	if err.Error() != errFeeBuilderNil {
		t.Errorf(errExpectMismatch, err, errFeeBuilderNil)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	var feeBuilder exchange.FeeBuilder
	feeBuilder.FeeType = exchange.OfflineTradeFee
	feeBuilder.Amount = 1
	feeBuilder.PurchasePrice = 1
	resp, err := c.GetFeeByType(context.Background(), &feeBuilder)
	if err != nil {
		t.Error(err)
	}
	if resp != 0.008 {
		t.Errorf(errExpectMismatch, resp, 0.008)
	}
}

func TestGetActiveOrders(t *testing.T) {
	_, err := c.GetActiveOrders(context.Background(), nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf(errExpectMismatch, err, common.ErrNilPointer)
	}
	var req order.MultiOrderRequest
	_, err = c.GetActiveOrders(context.Background(), &req)
	if err.Error() != errUnsupportedAssetType {
		t.Errorf(errExpectMismatch, err, errUnsupportedAssetType)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	req.AssetType = asset.Spot
	req.Side = order.AnySide
	req.Type = order.AnyType
	_, err = c.GetActiveOrders(context.Background(), &req)
	if err != nil {
		t.Error(err)
	}
	req.Pairs = req.Pairs.Add(currency.NewPair(currency.BTC, currency.USD))
	_, err = c.GetActiveOrders(context.Background(), &req)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	_, err := c.GetOrderHistory(context.Background(), nil)
	if !errors.Is(err, order.ErrGetOrdersRequestIsNil) {
		t.Errorf(errExpectMismatch, err, order.ErrGetOrdersRequestIsNil)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	var req order.MultiOrderRequest
	req.AssetType = asset.Spot
	req.Side = order.AnySide
	req.Type = order.AnyType
	_, err = c.GetOrderHistory(context.Background(), &req)
	if err != nil {
		t.Error(err)
	}
	req.Pairs = req.Pairs.Add(testPair)
	_, err = c.GetOrderHistory(context.Background(), &req)
	if err != nil {
		t.Error(err)
	}

}

func TestGetHistoricCandles(t *testing.T) {
	_, err := c.GetHistoricCandles(context.Background(), currency.Pair{}, asset.Empty, kline.OneYear, time.Time{},
		time.Time{})
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Errorf(errExpectMismatch, err, currency.ErrCurrencyPairEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err = c.GetHistoricCandles(context.Background(), testPair, asset.Spot, kline.ThreeHour,
		time.Now().Add(-time.Hour*30), time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	_, err := c.GetHistoricCandlesExtended(context.Background(), currency.Pair{}, asset.Empty, kline.OneYear,
		time.Time{}, time.Time{})
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Errorf(errExpectMismatch, err, currency.ErrCurrencyPairEmpty)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetHistoricCandlesExtended(context.Background(), testPair, asset.Spot, kline.OneMin,
		time.Now().Add(-time.Hour*9), time.Now())
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestValidateAPICredentials(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	err := c.ValidateAPICredentials(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetServerTime(t *testing.T) {
	_, err := c.GetServerTime(context.Background(), 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetLatestFundingRates(t *testing.T) {
	_, err := c.GetLatestFundingRates(context.Background(), nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf(errExpectMismatch, err, common.ErrNilPointer)
	}
	req := fundingrate.LatestRateRequest{Asset: asset.UpsideProfitContract}
	_, err = c.GetLatestFundingRates(context.Background(), &req)
	if err.Error() != errUpsideUnsupported {
		t.Errorf(errExpectMismatch, err, errUpsideUnsupported)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	req.Asset = asset.Futures
	_, err = c.GetLatestFundingRates(context.Background(), &req)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuturesContractDetails(t *testing.T) {
	_, err := c.GetFuturesContractDetails(context.Background(), asset.Empty)
	if !errors.Is(err, futures.ErrNotFuturesAsset) {
		t.Errorf(errExpectMismatch, err, futures.ErrNotFuturesAsset)
	}
	_, err = c.GetFuturesContractDetails(context.Background(), asset.UpsideProfitContract)
	if err.Error() != errUpsideUnsupported {
		t.Errorf(errExpectMismatch, err, errUpsideUnsupported)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err = c.GetFuturesContractDetails(context.Background(), asset.Futures)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	err := c.UpdateOrderExecutionLimits(context.Background(), asset.UpsideProfitContract)
	if err.Error() != errUpsideUnsupported {
		t.Errorf(errExpectMismatch, err, errUpsideUnsupported)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	err = c.UpdateOrderExecutionLimits(context.Background(), asset.Futures)
	if err != nil {
		t.Error(err)
	}
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
	if err.Error() != errJsonNumberIntoString {
		t.Errorf(errExpectMismatch, err, errJsonNumberIntoString)
	}
	err = u.UnmarshalJSON([]byte("\"922337203685477580700\""))
	if err.Error() != errParseIntValueOutOfRange {
		t.Errorf(errExpectMismatch, err, errParseIntValueOutOfRange)
	}
	err = u.UnmarshalJSON([]byte("\"1234\""))
	if err != nil {
		t.Error(err)
	}
}

func TestUnixTimestampString(t *testing.T) {
	t.Parallel()
	var u UnixTimestamp
	u.UnmarshalJSON([]byte("\"1234\""))
	s := u.String()
	if s != expectedTimestamp {
		t.Errorf(errExpectMismatch, s, expectedTimestamp)
	}
}

func TestFormatExchangeKlineInterval(t *testing.T) {
	t.Parallel()
	resp := formatExchangeKlineInterval(kline.FiveMin)
	if resp != granFiveMin {
		t.Errorf(errExpectMismatch, resp, granFiveMin)
	}
	resp = formatExchangeKlineInterval(kline.FifteenMin)
	if resp != granFifteenMin {
		t.Errorf(errExpectMismatch, resp, granFifteenMin)
	}
	resp = formatExchangeKlineInterval(kline.ThirtyMin)
	if resp != granThirtyMin {
		t.Errorf(errExpectMismatch, resp, granThirtyMin)
	}
	resp = formatExchangeKlineInterval(kline.TwoHour)
	if resp != granTwoHour {
		t.Errorf(errExpectMismatch, resp, granTwoHour)
	}
	resp = formatExchangeKlineInterval(kline.SixHour)
	if resp != granSixHour {
		t.Errorf(errExpectMismatch, resp, granSixHour)
	}
	resp = formatExchangeKlineInterval(kline.OneDay)
	if resp != granOneDay {
		t.Errorf(errExpectMismatch, resp, granOneDay)
	}
	resp = formatExchangeKlineInterval(kline.OneWeek)
	if resp != errIntervalNotSupported {
		t.Errorf(errExpectMismatch, resp, errIntervalNotSupported)
	}
}

func TestStringToFloatPtr(t *testing.T) {
	t.Parallel()
	err := stringToFloatPtr(nil, "")
	if err != errPointerNil {
		t.Errorf(errExpectMismatch, err, errPointerNil)
	}
	var fl float64
	err = stringToFloatPtr(&fl, "")
	if err != nil {
		t.Error(err)
	}
	err = stringToFloatPtr(&fl, "1.1")
	if err != nil {
		t.Error(err)
	}
}

func TestWsSomethingOrOther(t *testing.T) {

}

func skipTestIfLowOnFunds(t *testing.T) {
	accounts, err := c.GetAllAccounts(context.Background(), 250, "")
	if err != nil {
		t.Error(err)
	}
	if len(accounts.Accounts) == 0 {
		t.Fatal(errExpectedNonEmpty)
	}
	var hasValidFunds bool
	for i := range accounts.Accounts {
		if accounts.Accounts[i].Currency == testCrypto.String() && accounts.Accounts[i].AvailableBalance.Value > testAmount*100 {
			hasValidFunds = true
		}
	}
	if !hasValidFunds {
		t.Skip(skipInsufficientFunds)
	}
}

func portfolioTestHelper(t *testing.T, targetName string) string {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	createResp, err := c.CreatePortfolio(context.Background(), targetName)
	var targetID string
	if err != nil {
		if err.Error() != errPortfolioNameDuplicate {
			t.Error(err)
		}
		getResp, err := c.GetAllPortfolios(context.Background(), "")
		if err != nil {
			t.Error(err)
		}
		if len(getResp.Portfolios) == 0 {
			t.Fatal(errExpectedNonEmpty)
		}
		for i := range getResp.Portfolios {
			if getResp.Portfolios[i].Name == targetName {
				targetID = getResp.Portfolios[i].UUID
				break
			}
		}
	} else {
		targetID = createResp.Portfolio.UUID
	}
	return targetID
}

func convertTestHelper(t *testing.T) (string, string) {
	accIDs, err := c.GetAllAccounts(context.Background(), 250, "")
	if err != nil {
		t.Error(err)
	}
	if len(accIDs.Accounts) == 0 {
		t.Fatal(errExpectedNonEmpty)
	}
	var (
		fromAccID string
		toAccID   string
	)
	for x := range accIDs.Accounts {
		if accIDs.Accounts[x].Currency == "USDC" {
			fromAccID = accIDs.Accounts[x].UUID
		}
		if accIDs.Accounts[x].Currency == "USD" {
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

func transferTestHelper(t *testing.T, wallets GetAllWalletsResponse) (string, string) {
	var hasValidFunds bool
	var wID string
	for i := range wallets.Data {
		if wallets.Data[i].Currency.Code == testFiat.String() && wallets.Data[i].Balance.Amount > 10 {
			hasValidFunds = true
			wID = wallets.Data[i].ID
		}
	}
	if !hasValidFunds {
		t.Skip(skipInsufficientFunds)
	}
	pmID, err := c.GetAllPaymentMethods(context.Background(), PaginationInp{})
	if err != nil {
		t.Error(err)
	}
	if len(pmID.Data) == 0 {
		t.Skip(skipPayMethodNotFound)
	}
	return wID, pmID.Data[0].FiatAccount.ID
}
