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
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	gctlog "github.com/thrasher-corp/gocryptotrader/log"
)

var (
	c        = &CoinbasePro{}
	testPair = currency.NewPairWithDelimiter(currency.BTC.String(), currency.USD.String(), "-")
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

	errExpectMismatch     = "received: '%v' but expected: '%v'"
	errExpectedNonEmpty   = "expected non-empty response"
	errOrder0CancelFail   = "order 0 failed to cancel"
	errIDNotSet           = "ID not set"
	skipPayMethodNotFound = "no payment methods found, skipping"
	skipInsufSuitableAccs = "insufficient suitable accounts found, skipping"
	errx7f                = "setting proxy address error parse \"\\x7f\": net/url: invalid control character in URL"
)

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
	gdxConfig.API.AuthenticatedSupport = true
	c.Websocket = sharedtestvalues.NewTestWebsocket()
	err = c.Setup(gdxConfig)
	if err != nil {
		log.Fatal("CoinbasePro setup error", err)
	}
	c.GetBase().API.AuthenticatedSupport = true
	c.GetBase().API.AuthenticatedWebsocketSupport = true
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetAccountByID(context.Background(), "")
	if !errors.Is(err, errAccountIDEmpty) {
		t.Errorf(errExpectMismatch, err, errAccountIDEmpty)
	}
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
		t.Error("mismatched responses")
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetProductBook(context.Background(), "", 0)
	if !errors.Is(err, errProductIDEmpty) {
		t.Errorf(errExpectMismatch, err, errProductIDEmpty)
	}
	resp, err := c.GetProductBook(context.Background(), testPair.String(), 2)
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetAllProducts(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	// testPairs := []string{testPair.String(), "ETH-USD"}
	var testPairs []string
	resp, err := c.GetAllProducts(context.Background(), 30000, 0, "", "", testPairs)
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
	// log.Printf("%+v\n%+v", resp.NumProducts, len(resp.Products))
}

func TestGetProductByID(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetProductByID(context.Background(), "")
	if !errors.Is(err, errProductIDEmpty) {
		t.Errorf(errExpectMismatch, err, errProductIDEmpty)
	}
	resp, err := c.GetProductByID(context.Background(), testPair.String())
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetHistoricRates(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetHistoricRates(context.Background(), "", granUnknown, time.Time{}, time.Time{})
	if !errors.Is(err, errProductIDEmpty) {
		t.Errorf(errExpectMismatch, err, errProductIDEmpty)
	}
	_, err = c.GetHistoricRates(context.Background(), testPair.String(), "blorbo", time.Time{}, time.Time{})
	if err == nil {
		t.Error("expected error due to invalid granularity")
	}
	resp, err := c.GetHistoricRates(context.Background(), testPair.String(), granOneMin,
		time.Now().Add(-5*time.Minute), time.Now())
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetTicker(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetTicker(context.Background(), "", 1)
	if !errors.Is(err, errProductIDEmpty) {
		t.Errorf(errExpectMismatch, err, errProductIDEmpty)
	}
	resp, err := c.GetTicker(context.Background(), testPair.String(), 5)
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestPlaceOrder(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	_, err := c.PlaceOrder(context.Background(), "", "", "", "", "", 0, 0, 0, false, time.Time{})
	if !errors.Is(err, errClientOrderIDEmpty) {
		t.Errorf(errExpectMismatch, err, errClientOrderIDEmpty)
	}
	_, err = c.PlaceOrder(context.Background(), "meow", "", "", "", "", 0, 0, 0, false, time.Time{})
	if !errors.Is(err, errProductIDEmpty) {
		t.Errorf(errExpectMismatch, err, errProductIDEmpty)
	}
	_, err = c.PlaceOrder(context.Background(), "meow", testPair.String(), order.Sell.String(), "", "", 0,
		0, 0, false, time.Time{})
	if !errors.Is(err, errAmountZero) {
		t.Errorf(errExpectMismatch, err, errAmountZero)
	}
	id, _ := uuid.NewV4()
	_, err = c.PlaceOrder(context.Background(), id.String(), testPair.String(), order.Sell.String(), "",
		order.Limit.String(), 0.0000001, 1000000000000, 0, false, time.Now().Add(time.Hour))
	if err != nil {
		t.Error(err)
	}
}

func TestCancelOrders(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	var OrderSlice []string
	_, err := c.CancelOrders(context.Background(), OrderSlice)
	if !errors.Is(err, errOrderIDEmpty) {
		t.Errorf(errExpectMismatch, err, errOrderIDEmpty)
	}
	ordID, err := c.PlaceOrder(context.Background(), "meow", testPair.String(), order.Sell.String(), "",
		order.Limit.Lower(), 0.0000001, 1000000000000, 0, false, time.Time{})
	if err != nil {
		t.Error(err)
	}
	OrderSlice = append(OrderSlice, ordID.OrderID)
	_, err = c.CancelOrders(context.Background(), OrderSlice)
	if err != nil {
		t.Error(err)
	}
}

func TestEditOrder(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	_, err := c.EditOrder(context.Background(), "", 0, 0)
	if !errors.Is(err, errOrderIDEmpty) {
		t.Errorf(errExpectMismatch, err, errOrderIDEmpty)
	}
	_, err = c.EditOrder(context.Background(), "meow", 0, 0)
	if !errors.Is(err, errSizeAndPriceZero) {
		t.Errorf(errExpectMismatch, err, errSizeAndPriceZero)
	}
	id, _ := uuid.NewV4()
	ordID, err := c.PlaceOrder(context.Background(), id.String(), testPair.String(), order.Sell.String(), "",
		order.Limit.Lower(), 0.0000001, 1000000000000, 0, false, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = c.EditOrder(context.Background(), ordID.OrderID, 0, 10000000000000)
	if err != nil {
		t.Error(err)
	}
}

func TestEditOrderPreview(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	_, err := c.EditOrderPreview(context.Background(), "", 0, 0)
	if !errors.Is(err, errOrderIDEmpty) {
		t.Errorf(errExpectMismatch, err, errOrderIDEmpty)
	}
	_, err = c.EditOrderPreview(context.Background(), "meow", 0, 0)
	if !errors.Is(err, errSizeAndPriceZero) {
		t.Errorf(errExpectMismatch, err, errSizeAndPriceZero)
	}
	id, _ := uuid.NewV4()
	_, err = c.EditOrderPreview(context.Background(), id.String(), 0.0000001, 10000000000000)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllOrders(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	status := make([]string, 2)
	_, err := c.GetAllOrders(context.Background(), "", "", "", "", "", "", "", "", status, 0, time.Unix(2, 2),
		time.Unix(1, 1))
	if !errors.Is(err, common.ErrStartAfterEnd) {
		t.Errorf(errExpectMismatch, err, common.ErrStartAfterEnd)
	}
	status[0] = "CANCELLED"
	status[1] = "OPEN"
	_, err = c.GetAllOrders(context.Background(), "", "", "", "", "", "", "", "", status, 0, time.Time{},
		time.Time{})
	if !errors.Is(err, errOpenPairWithOtherTypes) {
		t.Errorf(errExpectMismatch, err, errOpenPairWithOtherTypes)
	}
	status = make([]string, 0)
	_, err = c.GetAllOrders(context.Background(), "", "USD", "LIMIT", "SELL", "", "SPOT", "RETAIL_ADVANCED",
		"UNKNOWN_CONTRACT_EXPIRY_TYPE", status, 10, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
}

func TestGetFills(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetFills(context.Background(), "", "", "", 0, time.Unix(2, 2), time.Unix(1, 1))
	if !errors.Is(err, common.ErrStartAfterEnd) {
		t.Errorf(errExpectMismatch, err, common.ErrStartAfterEnd)
	}
	_, err = c.GetFills(context.Background(), "", "", "", 5, time.Unix(1, 1), time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderByID(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetOrderByID(context.Background(), "", "", "")
	if !errors.Is(err, errOrderIDEmpty) {
		t.Errorf(errExpectMismatch, err, errOrderIDEmpty)
	}
	ordID, err := c.GetAllOrders(context.Background(), "", "", "", "", "", "", "", "", nil, 10,
		time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	if len(ordID.Orders) == 0 {
		t.Skip("no orders found, skipping")
	}
	_, err = c.GetOrderByID(context.Background(), ordID.Orders[0].OrderID, ordID.Orders[0].ClientOID, "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetTransactionSummary(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetTransactionSummary(context.Background(), time.Unix(2, 2), time.Unix(1, 1), "", "", "")
	if !errors.Is(err, common.ErrStartAfterEnd) {
		t.Errorf(errExpectMismatch, err, common.ErrStartAfterEnd)
	}
	_, err = c.GetTransactionSummary(context.Background(), time.Unix(1, 1), time.Now(), "", "SPOT",
		"UNKNOWN_CONTRACT_EXPIRY_TYPE")
	if err != nil {
		t.Error(err)
	}
}

func TestCreateConvertQuote(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	_, err := c.CreateConvertQuote(context.Background(), "", "", "", "", 0)
	if !errors.Is(err, errAccountIDEmpty) {
		t.Errorf(errExpectMismatch, err, errAccountIDEmpty)
	}
	_, err = c.CreateConvertQuote(context.Background(), "meow", "123", "", "", 0)
	if !errors.Is(err, errAmountEmpty) {
		t.Errorf(errExpectMismatch, err, errAmountEmpty)
	}
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
	_, err = c.CreateConvertQuote(context.Background(), fromAccID, toAccID, "", "", 0.01)
	if err != nil {
		t.Error(err)
	}
}

func TestCommitConvertTrade(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	_, err := c.CommitConvertTrade(context.Background(), "", "", "")
	if !errors.Is(err, errTransactionIDEmpty) {
		t.Errorf(errExpectMismatch, err, errTransactionIDEmpty)
	}
	_, err = c.CommitConvertTrade(context.Background(), "meow", "", "")
	if !errors.Is(err, errAccountIDEmpty) {
		t.Errorf(errExpectMismatch, err, errAccountIDEmpty)
	}
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
	resp, err := c.CreateConvertQuote(context.Background(), fromAccID, toAccID, "", "", 0.01)
	if err != nil {
		t.Error(err)
	}
	_, err = c.CommitConvertTrade(context.Background(), resp.Trade.ID, fromAccID, toAccID)
	if err != nil {
		t.Error(err)
	}
}

func TestGetConvertTradeByID(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	_, err := c.GetConvertTradeByID(context.Background(), "", "", "")
	if !errors.Is(err, errTransactionIDEmpty) {
		t.Errorf(errExpectMismatch, err, errTransactionIDEmpty)
	}
	_, err = c.GetConvertTradeByID(context.Background(), "meow", "", "")
	if !errors.Is(err, errAccountIDEmpty) {
		t.Errorf(errExpectMismatch, err, errAccountIDEmpty)
	}
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
	resp, err := c.CreateConvertQuote(context.Background(), fromAccID, toAccID, "", "", 0.01)
	if err != nil {
		t.Error(err)
	}
	_, err = c.GetConvertTradeByID(context.Background(), resp.Trade.ID, fromAccID, toAccID)
	if err != nil {
		t.Error(err)
	}
}

func TestGetV3Time(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetV3Time(context.Background())
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
	log.Printf("%+v", resp)
}

func TestListNotifications(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.ListNotifications(context.Background(), PaginationInp{})
	if err != nil {
		t.Error(err)
	}
}

func TestGetUserByID(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetCurrentUser(context.Background())
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
	resp2, err := c.GetUserByID(context.Background(), resp.Data.ID)
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp2, errExpectedNonEmpty)
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
	resp, err := c.UpdateUser(context.Background(), "", "", "AUD")
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestCreateWallet(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	resp, err := c.CreateWallet(context.Background(), "BTC")
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
		t.Skip("fewer than 3 wallets found, skipping pagination test")
	}
	pagIn.StartingAfter = resp.Pagination.NextStartingAfter
	resp, err = c.GetAllWallets(context.Background(), pagIn)
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestGetWalletByID(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetWalletByID(context.Background(), "", "")
	if !errors.Is(err, errCurrWalletConflict) {
		t.Errorf(errExpectMismatch, err, errCurrWalletConflict)
	}
	resp, err := c.GetWalletByID(context.Background(), "", "BTC")
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestUpdateWalletName(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	_, err := c.UpdateWalletName(context.Background(), "", "")
	if !errors.Is(err, errWalletIDEmpty) {
		t.Errorf(errExpectMismatch, err, errWalletIDEmpty)
	}
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	err := c.DeleteWallet(context.Background(), "")
	if !errors.Is(err, errWalletIDEmpty) {
		t.Errorf(errExpectMismatch, err, errWalletIDEmpty)
	}
	wID, err := c.CreateWallet(context.Background(), "LTC")
	if err != nil {
		t.Error(err)
	}
	// As of now, it seems like this next step always fails. DeleteWallet only lets you delete non-primary
	// non-fiat wallets, but the only non-primary wallet is fiat. Trying to create a secondary wallet for
	// any cryptocurrency using CreateWallet simply returns the details of the existing primary wallet.
	err = c.DeleteWallet(context.Background(), wID.Data.ID)
	if err != nil {
		t.Error(err)
	}
}

func TestCreateAddress(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	_, err := c.CreateAddress(context.Background(), "", "")
	if !errors.Is(err, errWalletIDEmpty) {
		t.Errorf(errExpectMismatch, err, errWalletIDEmpty)
	}
	wID, err := c.GetWalletByID(context.Background(), "", "BTC")
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	var pag PaginationInp
	_, err := c.GetAllAddresses(context.Background(), "", pag)
	if !errors.Is(err, errWalletIDEmpty) {
		t.Errorf(errExpectMismatch, err, errWalletIDEmpty)
	}
	wID, err := c.GetWalletByID(context.Background(), "", "BTC")
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetAddressByID(context.Background(), "", "")
	if !errors.Is(err, errWalletIDEmpty) {
		t.Errorf(errExpectMismatch, err, errWalletIDEmpty)
	}
	_, err = c.GetAddressByID(context.Background(), "123", "")
	if !errors.Is(err, errAddressIDEmpty) {
		t.Errorf(errExpectMismatch, err, errAddressIDEmpty)
	}
	wID, err := c.GetWalletByID(context.Background(), "", "BTC")
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetAddressTransactions(context.Background(), "", "", PaginationInp{})
	if !errors.Is(err, errWalletIDEmpty) {
		t.Errorf(errExpectMismatch, err, errWalletIDEmpty)
	}
	_, err = c.GetAddressTransactions(context.Background(), "123", "", PaginationInp{})
	if !errors.Is(err, errAddressIDEmpty) {
		t.Errorf(errExpectMismatch, err, errAddressIDEmpty)
	}
	wID, err := c.GetWalletByID(context.Background(), "", "BTC")
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	_, err := c.SendMoney(context.Background(), "", "", "", "", "", "", "", "", "", false, false)
	if !errors.Is(err, errTransactionTypeEmpty) {
		t.Errorf(errExpectMismatch, err, errTransactionTypeEmpty)
	}
	_, err = c.SendMoney(context.Background(), "123", "", "", "", "", "", "", "", "", false, false)
	if !errors.Is(err, errWalletIDEmpty) {
		t.Errorf(errExpectMismatch, err, errWalletIDEmpty)
	}
	_, err = c.SendMoney(context.Background(), "123", "123", "", "", "", "", "", "", "", false, false)
	if !errors.Is(err, errToEmpty) {
		t.Errorf(errExpectMismatch, err, errToEmpty)
	}
	_, err = c.SendMoney(context.Background(), "123", "123", "123", "", "", "", "", "", "", false, false)
	if !errors.Is(err, errAmountEmpty) {
		t.Errorf(errExpectMismatch, err, errAmountEmpty)
	}
	_, err = c.SendMoney(context.Background(), "123", "123", "123", "123", "", "", "", "", "", false, false)
	if !errors.Is(err, errCurrencyEmpty) {
		t.Errorf(errExpectMismatch, err, errCurrencyEmpty)
	}
	wID, err := c.GetAllWallets(context.Background(), PaginationInp{})
	if err != nil {
		t.Error(err)
	}
	if len(wID.Data) < 2 {
		t.Skip("fewer than 2 wallets found, skipping test")
	}
	_, err = c.SendMoney(context.Background(), "transfer", wID.Data[0].ID, wID.Data[1].ID, "0.00000001",
		"BTC", "GCT Test", "123", "", "", false, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllTransactions(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	var pag PaginationInp
	_, err := c.GetAllTransactions(context.Background(), "", pag)
	if !errors.Is(err, errWalletIDEmpty) {
		t.Errorf(errExpectMismatch, err, errWalletIDEmpty)
	}
	wID, err := c.GetWalletByID(context.Background(), "", "BTC")
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetTransactionByID(context.Background(), "", "")
	if !errors.Is(err, errWalletIDEmpty) {
		t.Errorf(errExpectMismatch, err, errWalletIDEmpty)
	}
	_, err = c.GetTransactionByID(context.Background(), "123", "")
	if !errors.Is(err, errTransactionIDEmpty) {
		t.Errorf(errExpectMismatch, err, errTransactionIDEmpty)
	}
	wID, err := c.GetWalletByID(context.Background(), "", "BTC")
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, wID, errExpectedNonEmpty)
	tID, err := c.GetAllTransactions(context.Background(), wID.Data.ID, PaginationInp{})
	if err != nil {
		t.Error(err)
	}
	if len(tID.Data) == 0 {
		t.Skip("no transactions found, skipping")
	}
	resp, err := c.GetTransactionByID(context.Background(), wID.Data.ID, tID.Data[0].ID)
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, resp, errExpectedNonEmpty)
}

func TestFiatTransfer(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
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
	wID, err := c.GetWalletByID(context.Background(), "", "AUD")
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, wID, errExpectedNonEmpty)
	pmID, err := c.GetAllPaymentMethods(context.Background(), PaginationInp{})
	if err != nil {
		t.Error(err)
	}
	if len(pmID.Data) == 0 {
		t.Skip(skipPayMethodNotFound)
	}
	_, err = c.FiatTransfer(context.Background(), wID.Data.ID, "AUD", pmID.Data[0].FiatAccount.ID, 1, false, FiatDeposit)
	if err != nil {
		t.Error(err)
	}
	_, err = c.FiatTransfer(context.Background(), wID.Data.ID, "AUD", pmID.Data[0].FiatAccount.ID, 1, false, FiatWithdrawal)
	if err != nil {
		t.Error(err)
	}
}

func TestCommitTransfer(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	_, err := c.CommitTransfer(context.Background(), "", "", FiatDeposit)
	if !errors.Is(err, errWalletIDEmpty) {
		t.Errorf(errExpectMismatch, err, errWalletIDEmpty)
	}
	_, err = c.CommitTransfer(context.Background(), "123", "", FiatDeposit)
	if !errors.Is(err, errDepositIDEmpty) {
		t.Errorf(errExpectMismatch, err, errDepositIDEmpty)
	}
	wID, err := c.GetWalletByID(context.Background(), "", "AUD")
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, wID, errExpectedNonEmpty)
	pmID, err := c.GetAllPaymentMethods(context.Background(), PaginationInp{})
	if err != nil {
		t.Error(err)
	}
	if len(pmID.Data) == 0 {
		t.Skip(skipPayMethodNotFound)
	}
	depID, err := c.FiatTransfer(context.Background(), wID.Data.ID, "AUD", pmID.Data[0].FiatAccount.ID, 1,
		false, FiatDeposit)
	if err != nil {
		t.Error(err)
	}
	_, err = c.CommitTransfer(context.Background(), wID.Data.ID, depID.Data.ID, FiatDeposit)
	if err != nil {
		t.Error(err)
	}
	_, err = c.CommitTransfer(context.Background(), wID.Data.ID, depID.Data.ID, FiatWithdrawal)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllFiatTransfers(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	var pag PaginationInp
	_, err := c.GetAllFiatTransfers(context.Background(), "", pag, FiatDeposit)
	if !errors.Is(err, errWalletIDEmpty) {
		t.Errorf(errExpectMismatch, err, errWalletIDEmpty)
	}
	wID, err := c.GetWalletByID(context.Background(), "", "AUD")
	if err != nil {
		t.Error(err)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetFiatTransferByID(context.Background(), "", "", FiatDeposit)
	if !errors.Is(err, errWalletIDEmpty) {
		t.Errorf(errExpectMismatch, err, errWalletIDEmpty)
	}
	_, err = c.GetFiatTransferByID(context.Background(), "123", "", FiatDeposit)
	if !errors.Is(err, errDepositIDEmpty) {
		t.Errorf(errExpectMismatch, err, errDepositIDEmpty)
	}
	wID, err := c.GetWalletByID(context.Background(), "", "AUD")
	if err != nil {
		t.Error(err)
	}
	assert.NotEmpty(t, wID, errExpectedNonEmpty)
	dID, err := c.GetAllFiatTransfers(context.Background(), wID.Data.ID, PaginationInp{}, FiatDeposit)
	if err != nil {
		t.Error(err)
	}
	if len(dID.Data) == 0 {
		t.Skip("no deposits found, skipping")
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetPaymentMethodByID(context.Background(), "")
	if !errors.Is(err, errPaymentMethodEmpty) {
		t.Errorf(errExpectMismatch, err, errPaymentMethodEmpty)
	}
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
	resp, err := c.GetPrice(context.Background(), testPair.String(), "spot")
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

/*

func TestGetHolds(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	accID, err := c.GetAllAccounts(context.Background(), 49, "")
	if err != nil {
		t.Error("CoinBasePro GetAllAccounts() error", err)
	}
	if len(accID.Accounts) == 0 {
		t.Fatal("CoinBasePro GetAllAccounts() error, expected a non-empty response")
	}
	_, _, err = c.GetHolds(context.Background(), accID.Accounts[1].UUID, pageNone, "2", 2)
	if err != nil {
		t.Error("CoinBasePro GetHolds() error", err)
	}
}

func TestGetAccountLedger(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	accID, err := c.GetAllAccounts(context.Background(), 49, "")
	if err != nil {
		t.Error("CoinBasePro GetAllAccounts() error", err)
	}
	_, _, err = c.GetAccountLedger(context.Background(), "", pageNone, "", "", time.Unix(2, 2), time.Unix(1, 1), 0)
	if err == nil {
		t.Error("CoinBasePro GetAccountLedger() error, expected an error due to invalid times")
	}
	if len(accID.Accounts) == 0 {
		t.Fatal("CoinBasePro GetAllAccounts() error, expected a non-empty response")
	}
	_, _, err = c.GetAccountLedger(context.Background(), accID.Accounts[1].UUID, pageNone, "1177507600", "a",
		time.Unix(1, 1), time.Now(), 1000)
	if err != nil {
		t.Error("CoinBasePro GetAccountLedger() error", err)
	}
}

func TestGetAccountTransfers(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	accID, err := c.GetAllAccounts(context.Background(), 49, "")
	if err != nil {
		t.Error("CoinBasePro GetAllAccounts() error", err)
	}
	if len(accID.Accounts) == 0 {
		t.Fatal("CoinBasePro GetAllAccounts() error, expected a non-empty response")
	}
	_, _, err = c.GetAccountTransfers(context.Background(), accID.Accounts[1].UUID, "", "", "", 3)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAddressBook(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetAddressBook(context.Background())
	if err != nil {
		t.Error("CoinBasePro GetAddressBook() error", err)
	}
	assert.NotEmpty(t, resp, "CoinBasePro GetAddressBook() error, expected a non-empty response")
}

func TestAddAddresses(t *testing.T) {
	// sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	// var req [1]AddAddressRequest
	// var err error
	// req[0], err = PrepareAddAddress("BTC", testAddress, "", "implemented", "Coinbase", false)
	// if err != nil {
	// 	t.Error(err)
	// }
	// resp, err := c.AddAddresses(context.Background(), req[:])
	// if err != nil {
	// 	t.Error("CoinBasePro AddAddresses() error", err)
	// }
	// assert.NotEmpty(t, resp, "CoinBasePro AddAddresses() error, expected a non-empty response")
}

func TestDeleteAddress(t *testing.T) {
	// sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	// var req [1]AddAddressRequest
	// var err error
	// req[0], err = PrepareAddAddress("BTC", testAddress, "", "implemented", "Coinbase", false)
	// if err != nil {
	// 	t.Error(err)
	// }
	// resp, err := c.AddAddresses(context.Background(), req[:])
	// if err != nil {
	// 	t.Error("CoinBasePro AddAddresses() error", err)
	// }
	// if len(resp) == 0 {
	// 	t.Fatal("CoinBasePro AddAddresses() error, expected a non-empty response")
	// }

	// err = c.DeleteAddress(context.Background(), resp[0].ID)
	// if err != nil {
	// 	t.Error("CoinBasePro DeleteAddress() error", err)
	// }
}

func TestGetCoinbaseWallets(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetCoinbaseWallets(context.Background())
	if err != nil {
		t.Error("CoinBasePro GetCoinbaseAccounts() error", err)
	}
	assert.NotEmpty(t, resp, "CoinBasePro GetCoinbaseWallets() error, expected a non-empty response")
}

func TestConvertCurrency(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	actBool := true
	profID, err := c.GetAllProfiles(context.Background(), &actBool)
	if err != nil {
		t.Error("CoinBasePro GetAllProfiles() error", err)
	}
	if len(profID) == 0 {
		t.Fatal("CoinBasePro GetAllProfiles() error, expected a non-empty response")
	}
	resp, err := c.ConvertCurrency(context.Background(), profID[0].ID, "USD", "USDC", "", 1)
	if err != nil {
		t.Error("CoinBasePro ConvertCurrency() error", err)
	}
	assert.NotEmpty(t, resp, "CoinBasePro ConvertCurrency() error, expected a non-empty response")
}

func TestGetConversionByID(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	actBool := true
	profID, err := c.GetAllProfiles(context.Background(), &actBool)
	if err != nil {
		t.Error("CoinBasePro GetAllProfiles() error", err)
	}
	if len(profID) == 0 {
		t.Fatal("CoinBasePro GetAllProfiles() error, expected a non-empty response")
	}
	resp, err := c.ConvertCurrency(context.Background(), profID[0].ID, "USD", "USDC", "", 1)
	if err != nil {
		t.Error("CoinBasePro ConvertCurrency() error", err)
	}
	resp2, err := c.GetConversionByID(context.Background(), resp.ID, profID[0].ID)
	if err != nil {
		t.Error("CoinBasePro GetConversionByID() error", err)
	}
	assert.NotEmpty(t, resp2, "CoinBasePro GetConversionByID() error, expected a non-empty response")
}

func TestGetAllCurrencies(t *testing.T) {
	_, err := c.GetAllCurrencies(context.Background())
	if err != nil {
		t.Error("GetAllCurrencies() error", err)
	}
}

func TestGetCurrencyByID(t *testing.T) {
	resp, err := c.GetCurrencyByID(context.Background(), "BTC")
	if err != nil {
		t.Error("GetCurrencyByID() error", err)
	}
	if resp.Name != "Bitcoin" {
		t.Errorf("GetCurrencyByID() error, incorrect name returned, expected 'Bitcoin', got '%s'",
			resp.Name)
	}
}

func TestDepositViaCoinbase(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	accID, err := c.GetCoinbaseWallets(context.Background())
	if err != nil {
		t.Error("CoinBasePro GetCoinbaseWallets() error", err)
	}
	if len(accID) == 0 {
		t.Fatal("CoinBasePro GetCoinbaseWallets() error, expected a non-empty response")
	}
	resp, err := c.DepositViaCoinbase(context.Background(), "", "BTC", accID[1].ID, 1)
	if err != nil {
		t.Error("CoinBasePro DepositViaCoinbase() error", err)
	}
	log.Printf("%+v", resp)
}

func TestDepositViaPaymentMethod(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	actBool := true
	profID, err := c.GetAllProfiles(context.Background(), &actBool)
	if err != nil {
		t.Error("CoinBasePro GetAllProfiles() error", err)
	}
	if len(profID) == 0 {
		t.Fatal("CoinBasePro GetAllProfiles() error, expected a non-empty response")
	}
	payID, err := c.GetPayMethods(context.Background())
	if err != nil {
		t.Error("CoinBasePro GetPayMethods() error", err)
	}
	var success bool
	i := 0
	for i = range payID {
		if payID[i].Type == "ach_bank_account" {
			success = true
			break
		}
	}
	if !success {
		t.Skip("Skipping test due to no ACH bank account found")
	}
	_, err = c.DepositViaPaymentMethod(context.Background(), profID[0].ID, payID[i].ID, payID[i].Currency, 1)
	if err != nil {
		t.Error("CoinBasePro DepositViaPaymentMethod() error", err)
	}
}

func TestGetPayMethods(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetPayMethods(context.Background())
	if err != nil {
		t.Error("CoinBasePro GetPayMethods() error", err)
	}
}

func TestGetAllTransfers(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, _, err := c.GetAllTransfers(context.Background(), "", "", "", "", 3)
	if err != nil {
		t.Error("CoinBasePro GetAllTransfers() error", err)
	}
}

func TestGetTransferByID(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, _, err := c.GetAllTransfers(context.Background(), "", "", "", "", 3)
	if err != nil {
		t.Error("CoinBasePro GetAllTransfers() error", err)
	}
	if len(resp) == 0 {
		t.Skip("TestGetTransferByID skipped due to there being zero transfers.")
	}
	_, err = c.GetTransferByID(context.Background(), resp[0].ID)
	if err != nil {
		t.Error("CoinBasePro GetTransferByID() error", err)
	}

}

func TestSendTravelInfoForTransfer(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	resp, _, err := c.GetAllTransfers(context.Background(), "", "", "", "", 1000)
	if err != nil {
		t.Error("CoinBasePro GetAllTransfers() error", err)
	}
	if len(resp) == 0 {
		t.Skip("TestSendTravelInfoForTransfer skipped due to there being zero pending transfers.")
	}
	var tID string
	var zeroValue ExchTime
	for i := range resp {
		if resp[i].CompletedAt == zeroValue && resp[i].CanceledAt == zeroValue &&
			resp[i].ProcessedAt == zeroValue {
			tID = resp[i].ID
			break
		}
	}
	if tID == "" {
		t.Log("TestSendTravelInfoForTransfer skipped due to there being zero pending transfers.")
	} else {
		_, err = c.SendTravelInfoForTransfer(context.Background(), tID, "GoCryptoTrader", "AU")
		if err != nil {
			t.Error("CoinBasePro SendTravelInfoForTransfer() error", err)
		}
	}

}

func TestWithdrawViaCoinbase(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	accID, err := c.GetCoinbaseWallets(context.Background())
	if err != nil {
		t.Error("CoinBasePro GetCoinbaseWallets() error", err)
	}
	if len(accID) == 0 {
		t.Fatal("CoinBasePro GetCoinbaseWallets() error, expected a non-empty response")
	}
	_, err = c.WithdrawViaCoinbase(context.Background(), "", accID[1].ID, "BTC", 1)
	if err != nil {
		t.Error("CoinBasePro WithdrawViaCoinbase() error", err)
	}
}

func TestWithdrawCrypto(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	actBool := true
	profID, err := c.GetAllProfiles(context.Background(), &actBool)
	if err != nil {
		t.Error("CoinBasePro GetAllProfiles() error", err)
	}
	if len(profID) == 0 {
		t.Fatal("CoinBasePro GetAllProfiles() error, expected a non-empty response")
	}
	resp, err := c.WithdrawCrypto(context.Background(), profID[0].ID, "BTC", testAddress, "", "", "bitcoin", 1,
		false, false, 2)
	if err != nil {
		t.Error("CoinBasePro WithdrawCrypto() error", err)
	}
	log.Printf("%+v", resp)
}

func TestGetWithdrawalFeeEstimate(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetWithdrawalFeeEstimate(context.Background(), "", "", "")
	if err == nil {
		t.Error("CoinBasePro GetWithdrawalFeeEstimate() error, expected error due to empty field")
	}
	_, err = c.GetWithdrawalFeeEstimate(context.Background(), "BTC", "", "")
	if err == nil {
		t.Error("CoinBasePro GetWithdrawalFeeEstimate() error, expected error due to empty field")
	}
	_, err = c.GetWithdrawalFeeEstimate(context.Background(), "BTC", testAddress, "bitcoin")
	if err != nil {
		t.Error("CoinBasePro GetWithdrawalFeeEstimate() error", err)
	}
}

func TestWithdrawViaPaymentMethod(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	actBool := true
	profID, err := c.GetAllProfiles(context.Background(), &actBool)
	if err != nil {
		t.Error("CoinBasePro GetAllProfiles() error", err)
	}
	if len(profID) == 0 {
		t.Fatal("CoinBasePro GetAllProfiles() error, expected a non-empty response")
	}
	payID, err := c.GetPayMethods(context.Background())
	if err != nil {
		t.Error("CoinBasePro GetPayMethods() error", err)
	}
	var success bool
	i := 0
	for i = range payID {
		if payID[i].Type == "ach_bank_account" {
			success = true
			break
		}
	}
	if !success {
		t.Skip("Skipping test due to no ACH bank account found")
	}
	_, err = c.WithdrawViaPaymentMethod(context.Background(), profID[0].ID, payID[i].ID, payID[i].Currency, 1)
	if err != nil {
		t.Error("CoinBasePro WithdrawViaPaymentMethod() error", err)
	}
}

func TestGetFees(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetFees(context.Background())
	if err != nil {
		t.Error("CoinBasePro GetFees() error", err)
	}
	assert.NotEmpty(t, resp, "CoinBasePro GetFees() error, expected a non-empty response")
}

func TestGetSignedPrices(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetSignedPrices(context.Background())
	if err != nil {
		t.Error("CoinBasePro GetSignedPrices() error", err)
	}
	assert.NotEmpty(t, resp, "CoinBasePro GetSignedPrices() error, expected a non-empty response")
}

func TestGetOrderbook(t *testing.T) {
	_, err := c.GetOrderbook(context.Background(), "", 1)
	if err == nil {
		t.Error("Coinbase, GetOrderbook() Error, expected an error due to nonexistent pair")
	}
	_, err = c.GetOrderbook(context.Background(), "There's no way this doesn't cause an error", 1)
	if err == nil {
		t.Error("Coinbase, GetOrderbook() Error, expected an error due to invalid pair")
	}
	resp, err := c.GetOrderbook(context.Background(), testPair.String(), 1)
	if err != nil {
		t.Error("Coinbase, GetOrderbook() Error", err)
	}
	assert.NotEmpty(t, resp, "Coinbase, GetOrderbook() Error, expected a non-empty response")
}

func TestGetStats(t *testing.T) {
	_, err := c.GetStats(context.Background(), "")
	if err == nil {
		t.Error("Coinbase, GetStats() Error, expected an error due to nonexistent pair")
	}
	resp, err := c.GetStats(context.Background(), testPair.String())
	if err != nil {
		t.Error("GetStats() error", err)
	}
	assert.NotEmpty(t, resp, "Coinbase, GetStats() Error, expected a non-empty response")
}

func TestGetTrades(t *testing.T) {
	_, err := c.GetTrades(context.Background(), "", "", "", 1)
	if err == nil {
		t.Error("Coinbase, GetTrades() Error, expected an error due to nonexistent pair")
	}
	resp, err := c.GetTrades(context.Background(), testPair.String(), "", "", 1)
	if err != nil {
		t.Error("GetTrades() error", err)
	}
	assert.NotEmpty(t, resp, "Coinbase, GetTrades() Error, expected a non-empty response")
}

func TestGetAllProfiles(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	active := true
	resp, err := c.GetAllProfiles(context.Background(), &active)
	if err != nil {
		t.Error("GetAllProfiles() error", err)
	}
	assert.NotEmpty(t, resp, "Coinbase, GetAllProfiles() Error, expected a non-empty response")
}

func TestCreateAProfile(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	_, err := c.CreateAProfile(context.Background(), "")
	if err == nil {
		t.Error("Coinbase, CreateAProfile() Error, expected an error due to empty name")
	}
	_, err = c.CreateAProfile(context.Background(), "default")
	if err == nil {
		t.Error("CreateAProfile() error, expected an error due to reserved name")
	}
	t.Skip("Skipping test; seems to always return an internal server error when a non-reserved profile name is sent")
	resp, err := c.CreateAProfile(context.Background(), "GCTTestProfile")
	if err != nil {
		t.Error("CreateAProfile() error", err)
	}
	assert.NotEmpty(t, resp, "Coinbase, CreateAProfile() Error, expected a non-empty response")
}

// Cannot d due to there only being one profile, and CreateAProfile not working
func TestTransferBetweenProfiles(t *testing.T) {
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, c, canManipulateRealOrders)
	_, err := c.TransferBetweenProfiles(context.Background(), "", "", "", 0)
	if err == nil {
		t.Error("Coinbase, TransferBetweenProfiles() Error, expected an error due to empty fields")
	}
	_, err = c.TransferBetweenProfiles(context.Background(), "this test", "has not", "been implemented", 0)
	if err == nil {
		t.Error("Coinbase, TransferBetweenProfiles() Error, expected an error due to un-implemented test")
	}
}

func TestGetProfileByID(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetAllProfiles(context.Background(), nil)
	if err != nil {
		t.Error("GetAllProfiles() error", err)
	}
	active := true
	resp2, err := c.GetProfileByID(context.Background(), resp[0].ID, &active)
	if err != nil {
		t.Error("GetProfileByID() error", err)
	}
	assert.NotEmpty(t, resp2, "Coinbase, GetProfileByID() Error, expected a non-empty response")
}

func TestRenameProfile(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	_, err := c.RenameProfile(context.Background(), "", "")
	if err == nil {
		t.Error("Coinbase, RenameProfile() Error, expected an error due to empty fields")
	}
	profID, err := c.GetAllProfiles(context.Background(), nil)
	if err != nil {
		t.Error("GetAllProfiles() error", err)
	}
	if len(profID) == 0 {
		t.Fatal("CoinBasePro GetAllProfiles() error, expected a non-empty response")
	}
	_, err = c.RenameProfile(context.Background(), profID[0].ID, "margin")
	if err == nil {
		t.Error("RenameProfile() error, expected an error due to reserved name")
	}
	t.Skip("Skipping test; seems to always return an internal server error when a non-reserved profile name is sent")
	resp, err := c.RenameProfile(context.Background(), profID[0].ID, "GCTTestProfile2")
	if err != nil {
		t.Error("Coinbase, RenameProfile() Error", err)
	}
	assert.NotEmpty(t, resp, "Coinbase, RenameProfile() Error, expected a non-empty response")
}

// Cannot be tested due to there only being one profile, and CreateAProfile not working
func TestDeleteProfile(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	_, err := c.DeleteProfile(context.Background(), "", "")
	if err == nil {
		t.Error("Coinbase, DeleteProfile() Error, expected an error due to empty fields")
	}
	profID, err := c.GetAllProfiles(context.Background(), nil)
	if err != nil {
		t.Error("GetAllProfiles() error", err)
	}
	if len(profID) == 0 {
		t.Fatal("CoinBasePro GetAllProfiles() error, expected a non-empty response")
	}
	_, err = c.DeleteProfile(context.Background(), "profID[0].ID", "this test hasn't been implemented")
	if err == nil {
		t.Error("Coinbase, DeleteProfile() Error, expected an error due to un-implemented test")
	}
}

func TestGetAllReports(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	profID, err := c.GetAllProfiles(context.Background(), nil)
	if err != nil {
		t.Error("GetAllProfiles() error", err)
	}
	if len(profID) == 0 {
		t.Fatal("CoinBasePro GetAllProfiles() error, expected a non-empty response")
	}
	_, err = c.GetAllReports(context.Background(), profID[0].ID, "account", time.Time{}, 1000, false)
	if err != nil {
		t.Error("GetAllReports() error", err)
	}
}

func TestCreateReport(t *testing.T) {
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, c, canManipulateRealOrders)
	profID, err := c.GetAllProfiles(context.Background(), nil)
	if err != nil {
		t.Error("GetAllProfiles() error", err)
	}
	if len(profID) == 0 {
		t.Fatal("CoinBasePro GetAllProfiles() error, expected a non-empty response")
	}
	accID, err := c.GetAllAccounts(context.Background(), 49, "")
	if err != nil {
		t.Error("CoinBasePro GetAllAccounts() error", err)
	}
	if len(accID.Accounts) == 0 {
		t.Fatal("CoinBasePro GetAllAccounts() error, expected a non-empty response")
	}
	// _, err = c.CreateReport(context.Background(), "account", "", "pdf", "testemail@thrasher.io", profID[0].ID,
	// 	"", accID[0].ID, time.Time{}, time.Unix(1, 1), time.Now())
	// if err != nil {
	// 	t.Error("Coinbase, CreateReport() error", err)
	// }
	resp, err := c.CreateReport(context.Background(), "balance", "", "csv", "testemail@thrasher.io", "",
		"", "", time.Now(), time.Unix(1, 1), time.Now())
	if err != nil {
		t.Error("Coinbase, CreateReport() error", err)
	}
	assert.NotEmpty(t, resp, "Coinbase, CreateReport() Error, expected a non-empty response")
}

func TestGetReportByID(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetReportByID(context.Background(), "")
	if err == nil {
		t.Error("Coinbase, GetReportByID() Error, expected an error due to empty fields")
	}
	prof, err := c.GetAllProfiles(context.Background(), nil)
	if err != nil {
		t.Error("GetAllProfiles() error", err)
	}
	if len(prof) == 0 {
		t.Fatal("CoinBasePro GetAllProfiles() error, expected a non-empty response")
	}
	resp, err := c.GetAllReports(context.Background(), prof[0].ID, "account", time.Time{}, 1000, false)
	if err != nil {
		t.Error("GetAllReports() error", err)
	}
	if len(resp) == 0 {
		t.Skip("Skipping test due to no reports found")
	}
	resp2, err := c.GetReportByID(context.Background(), resp[0].ID)
	if err != nil {
		t.Error("GetReportByID() error", err)
	}
	assert.NotEmpty(t, resp2, "Coinbase, GetReportByID() Error, expected a non-empty response")
}

func TestGetTravelRules(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetTravelRules(context.Background(), "", "", "", 0)
	if err != nil {
		t.Error("GetTravelRules() error", err)
	}
}

func TestCreateTravelRule(t *testing.T) {
	// sharedtestvalues.SkipTestIfCannotManipulateOrders(t, c, canManipulateRealOrders)
	// resp, err := c.CreateTravelRule(context.Background(), "GCT Travel Rule Test", "GoCryptoTrader", "AU")
	// if err != nil && err.Error() != travelRuleDuplicateError {
	// 	t.Error("Coinbase, CreateTravelRule() error", err)
	// }
	// assert.NotEmpty(t, resp, "Coinbase, CreateTravelRule() Error, expected a non-empty response")
}

func TestDeleteTravelRule(t *testing.T) {
	// sharedtestvalues.SkipTestIfCannotManipulateOrders(t, c, canManipulateRealOrders)
	// err := c.DeleteTravelRule(context.Background(), "")
	// if err == nil {
	// 	t.Error("Coinbase, DeleteTravelRule() Error, expected an error due to empty ID")
	// }
	// _, err = c.CreateTravelRule(context.Background(), "GCT Travel Rule Test", "GoCryptoTrader", "AU")
	// if err != nil && err.Error() != travelRuleDuplicateError {
	// 	t.Error("Coinbase, CreateTravelRule() error", err)
	// }
	// resp, err := c.GetTravelRules(context.Background(), "", "", "", 0)
	// if err != nil {
	// 	t.Error("GetTravelRules() error", err)
	// }
	// if len(resp) == 0 {
	// 	t.Fatal("GetTravelRules() error, expected a non-empty response")
	// }
	// for i := range resp {
	// 	if resp[i].Address == "GCT Travel Rule Test" {
	// 		err = c.DeleteTravelRule(context.Background(), resp[i].ID)
	// 		if err != nil {
	// 			t.Error("Coinbase, DeleteTravelRule() error", err)
	// 		}
	// 	}
	// }
}

func TestGetExchangeLimits(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	accID, err := c.GetAllAccounts(context.Background(), 49, "")
	if err != nil {
		t.Error("GetAllAccounts() error", err)
	}
	if len(accID.Accounts) == 0 {
		t.Fatal("CoinBasePro GetAllAccounts() error, expected a non-empty response")
	}
	resp, err := c.GetExchangeLimits(context.Background(), accID.Accounts[0].UUID)
	if err != nil {
		t.Error("GetExchangeLimits() error", err)
	}
	assert.NotEmpty(t, resp, "Coinbase, GetExchangeLimits() Error, expected a non-empty response")
}

func TestUpdateSettlementPreference(t *testing.T) {
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, c, canManipulateRealOrders)
	uID, err := c.GetAllProfiles(context.Background(), nil)
	if err != nil {
		t.Error("GetAllProfiles() error", err)
	}
	if len(uID) == 0 {
		t.Fatal("CoinBasePro GetAllProfiles() error, expected a non-empty response")
	}
	resp, err := c.UpdateSettlementPreference(context.Background(), uID[0].UserID, "USD")
	if err != nil {
		t.Error("Coinbase, UpdateSettlementPreference() error", err)
	}
	log.Printf("%+v", resp)
}

func TestGetAllWrappedAssets(t *testing.T) {
	resp, err := c.GetAllWrappedAssets(context.Background())
	if err != nil {
		t.Error("GetAllWrappedAssets() error", err)
	}
	assert.NotEmpty(t, resp, "Coinbase, GetAllWrappedAssets() Error, expected a non-empty response")
}

func TestGetAllStakeWraps(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetAllStakeWraps(context.Background(), "", "ETH", "CBETH", "", time.Time{}, 1)
	if err != nil {
		t.Error("GetAllStakeWraps() error", err)
	}
	_, err = c.GetAllStakeWraps(context.Background(), "after", "ETH", "CBETH", "", time.Unix(1, 1), 1000)
	if err != nil {
		t.Error("GetAllStakeWraps() error", err)
	}
}

func TestCreateStakeWrap(t *testing.T) {
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, c, canManipulateRealOrders)
	_, err := c.CreateStakeWrap(context.Background(), "", "", 0)
	if err == nil {
		t.Error("Coinbase, CreateStakeWrap() Error, expected an error due to empty fields")
	}
	resp, err := c.CreateStakeWrap(context.Background(), "ETH", "CBETH", 1)
	if err != nil {
		t.Error("Coinbase, CreateStakeWrap() error", err)
	}
	log.Printf("%+v", resp)
}

func TestGetStakeWrapByID(t *testing.T) {
	resp, err := c.GetAllStakeWraps(context.Background(), "", "ETH", "CBETH", "", time.Time{}, 1)
	if err != nil {
		t.Error("GetAllStakeWraps() error", err)
	}
	if len(resp) == 0 {
		t.Skip("No stake wraps found, skipping test")
	}
	resp2, err := c.GetStakeWrapByID(context.Background(), resp[0].ID)
	if err != nil {
		t.Error("GetStakeWrapByID() error", err)
	}
	assert.NotEmpty(t, resp2, "Coinbase, GetStakeWrapByID() Error, expected a non-empty response")

}

func TestGetWrappedAssetByID(t *testing.T) {
	_, err := c.GetWrappedAssetByID(context.Background(), "")
	if err == nil {
		t.Error("Coinbase, GetWrappedAssetByID() Error, expected an error due to empty fields")
	}
	resp, err := c.GetWrappedAssetByID(context.Background(), "CBETH")
	if err != nil {
		t.Error("GetWrappedAssetByID() error", err)
	}
	assert.NotEmpty(t, resp, "Coinbase, GetWrappedAssetByID() Error, expected a non-empty response")
}

func TestGetWrappedAssetConversionRate(t *testing.T) {
	_, err := c.GetWrappedAssetConversionRate(context.Background(), "")
	if err == nil {
		t.Error("Coinbase, GetWrappedAssetConversionRate() Error, expected an error due to empty fields")
	}
	resp, err := c.GetWrappedAssetConversionRate(context.Background(), "CBETH")
	if err != nil {
		t.Error("GetWrappedAssetConversionRate() error", err)
	}
	assert.NotEmpty(t, resp, "Coinbase, GetWrappedAssetConversionRate() Error, expected a non-empty response")
}

*/

// func TestGetCurrentServerTime(t *testing.T) {
// 	_, err := c.GetCurrentServerTime(context.Background())
// 	if err != nil {
// 		t.Error("GetServerTime() error", err)
// 	}
// }

// func TestWrapperGetServerTime(t *testing.T) {
// 	t.Parallel()
// 	st, err := c.GetServerTime(context.Background(), asset.Spot)
// 	if !errors.Is(err, nil) {
// 		t.Fatalf(errExpectMismatch, err, nil)
// 	}

// 	if st.IsZero() {
// 		t.Fatal("expected a time")
// 	}
// }

// func TestAuthRequests(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)

// 	_, err := c.GetAllAccounts(context.Background(), 49, "")
// 	if err != nil {
// 		t.Error("GetAllAccounts() error", err)
// 	}
// 	accountResponse, err := c.GetAccountByID(context.Background(),
// 		"13371337-1337-1337-1337-133713371337")
// 	if accountResponse.ID != "" {
// 		t.Error("Expecting no data returned")
// 	}
// 	if err == nil {
// 		t.Error("Expecting error")
// 	}
// 	if err == nil {
// 		t.Error("Expecting error")
// 	}
// 	// getHoldsResponse, err := c.GetHolds(context.Background(),
// 	// 	"13371337-1337-1337-1337-133713371337")
// 	// if len(getHoldsResponse) > 0 {
// 	// 	t.Error("Expecting no data returned")
// 	// }
// 	if err == nil {
// 		t.Error("Expecting error")
// 	}
// 	marginTransferResponse, err := c.MarginTransfer(context.Background(),
// 		1, "withdraw", "13371337-1337-1337-1337-133713371337", "BTC")
// 	if marginTransferResponse.ID != "" {
// 		t.Error("Expecting no data returned")
// 	}
// 	if err == nil {
// 		t.Error("Expecting error")
// 	}
// 	_, err = c.GetPosition(context.Background())
// 	if err == nil {
// 		t.Error("Expecting error")
// 	}
// 	_, err = c.ClosePosition(context.Background(), false)
// 	if err == nil {
// 		t.Error("Expecting error")
// 	}
// }

// func setFeeBuilder() *exchange.FeeBuilder {
// 	return &exchange.FeeBuilder{
// 		Amount:        1,
// 		FeeType:       exchange.CryptocurrencyTradeFee,
// 		Pair:          testPair,
// 		PurchasePrice: 1,
// 	}
// }

// // TestGetFeeByTypeOfflineTradeFee logic test
// func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
// 	var feeBuilder = setFeeBuilder()
// 	_, err := c.GetFeeByType(context.Background(), feeBuilder)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	if !sharedtestvalues.AreAPICredentialsSet(c) {
// 		if feeBuilder.FeeType != exchange.OfflineTradeFee {
// 			t.Errorf("Expected %v, received %v", exchange.OfflineTradeFee, feeBuilder.FeeType)
// 		}
// 	} else {
// 		if feeBuilder.FeeType != exchange.CryptocurrencyTradeFee {
// 			t.Errorf("Expected %v, received %v", exchange.CryptocurrencyTradeFee, feeBuilder.FeeType)
// 		}
// 	}
// }

// func TestGetFee(t *testing.T) {
// 	var feeBuilder = setFeeBuilder()

// 	if sharedtestvalues.AreAPICredentialsSet(c) {
// 		// CryptocurrencyTradeFee Basic
// 		if _, err := c.GetFee(context.Background(), feeBuilder); err != nil {
// 			t.Error(err)
// 		}

// 		// CryptocurrencyTradeFee High quantity
// 		feeBuilder = setFeeBuilder()
// 		feeBuilder.Amount = 1000
// 		feeBuilder.PurchasePrice = 1000
// 		if _, err := c.GetFee(context.Background(), feeBuilder); err != nil {
// 			t.Error(err)
// 		}

// 		// CryptocurrencyTradeFee IsMaker
// 		feeBuilder = setFeeBuilder()
// 		feeBuilder.IsMaker = true
// 		if _, err := c.GetFee(context.Background(), feeBuilder); err != nil {
// 			t.Error(err)
// 		}

// 		// CryptocurrencyTradeFee Negative purchase price
// 		feeBuilder = setFeeBuilder()
// 		feeBuilder.PurchasePrice = -1000
// 		if _, err := c.GetFee(context.Background(), feeBuilder); err != nil {
// 			t.Error(err)
// 		}
// 	}

// 	// CryptocurrencyWithdrawalFee Basic
// 	feeBuilder = setFeeBuilder()
// 	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
// 	if _, err := c.GetFee(context.Background(), feeBuilder); err != nil {
// 		t.Error(err)
// 	}

// 	// CryptocurrencyDepositFee Basic
// 	feeBuilder = setFeeBuilder()
// 	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
// 	if _, err := c.GetFee(context.Background(), feeBuilder); err != nil {
// 		t.Error(err)
// 	}

// 	// InternationalBankDepositFee Basic
// 	feeBuilder = setFeeBuilder()
// 	feeBuilder.FeeType = exchange.InternationalBankDepositFee
// 	feeBuilder.FiatCurrency = currency.EUR
// 	if _, err := c.GetFee(context.Background(), feeBuilder); err != nil {
// 		t.Error(err)
// 	}

// 	// InternationalBankWithdrawalFee Basic
// 	feeBuilder = setFeeBuilder()
// 	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
// 	feeBuilder.FiatCurrency = currency.USD
// 	if _, err := c.GetFee(context.Background(), feeBuilder); err != nil {
// 		t.Error(err)
// 	}
// }

// func TestCalculateTradingFee(t *testing.T) {
// 	t.Parallel()
// 	// uppercase
// 	var volume = []Volume{
// 		{
// 			ProductID: "BTC_USD",
// 			Volume:    100,
// 		},
// 	}

// 	if resp := c.calculateTradingFee(volume, currency.BTC, currency.USD, "_", 1, 1, false); resp != float64(0.003) {
// 		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.003), resp)
// 	}

// 	// lowercase
// 	volume = []Volume{
// 		{
// 			ProductID: "btc_usd",
// 			Volume:    100,
// 		},
// 	}

// 	if resp := c.calculateTradingFee(volume, currency.BTC, currency.USD, "_", 1, 1, false); resp != float64(0.003) {
// 		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.003), resp)
// 	}

// 	// mixedCase
// 	volume = []Volume{
// 		{
// 			ProductID: "btc_USD",
// 			Volume:    100,
// 		},
// 	}

// 	if resp := c.calculateTradingFee(volume, currency.BTC, currency.USD, "_", 1, 1, false); resp != float64(0.003) {
// 		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.003), resp)
// 	}

// 	// medium volume
// 	volume = []Volume{
// 		{
// 			ProductID: "btc_USD",
// 			Volume:    10000001,
// 		},
// 	}

// 	if resp := c.calculateTradingFee(volume, currency.BTC, currency.USD, "_", 1, 1, false); resp != float64(0.002) {
// 		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.002), resp)
// 	}

// 	// high volume
// 	volume = []Volume{
// 		{
// 			ProductID: "btc_USD",
// 			Volume:    100000010000,
// 		},
// 	}

// 	if resp := c.calculateTradingFee(volume, currency.BTC, currency.USD, "_", 1, 1, false); resp != float64(0.001) {
// 		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.001), resp)
// 	}

// 	// no match
// 	volume = []Volume{
// 		{
// 			ProductID: "btc_beeteesee",
// 			Volume:    100000010000,
// 		},
// 	}

// 	if resp := c.calculateTradingFee(volume, currency.BTC, currency.USD, "_", 1, 1, false); resp != float64(0) {
// 		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
// 	}

// 	// taker
// 	volume = []Volume{
// 		{
// 			ProductID: "btc_USD",
// 			Volume:    100000010000,
// 		},
// 	}

// 	if resp := c.calculateTradingFee(volume, currency.BTC, currency.USD, "_", 1, 1, true); resp != float64(0) {
// 		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
// 	}
// }

// func TestFormatWithdrawPermissions(t *testing.T) {
// 	expectedResult := exchange.AutoWithdrawCryptoWithAPIPermissionText + " & " + exchange.AutoWithdrawFiatWithAPIPermissionText
// 	withdrawPermissions := c.FormatWithdrawPermissions()
// 	if withdrawPermissions != expectedResult {
// 		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
// 	}
// }

// func TestGetActiveOrders(t *testing.T) {
// 	var getOrdersRequest = order.MultiOrderRequest{
// 		Type:      order.AnyType,
// 		AssetType: asset.Spot,
// 		Pairs:     []currency.Pair{testPair},
// 		Side:      order.AnySide,
// 	}

// 	_, err := c.GetActiveOrders(context.Background(), &getOrdersRequest)
// 	if sharedtestvalues.AreAPICredentialsSet(c) && err != nil {
// 		t.Errorf("Could not get open orders: %s", err)
// 	} else if !sharedtestvalues.AreAPICredentialsSet(c) && err == nil {
// 		t.Error("Expecting an error when no keys are set")
// 	}
// }

// func TestGetOrderHistory(t *testing.T) {
// 	var getOrdersRequest = order.MultiOrderRequest{
// 		Type:      order.AnyType,
// 		AssetType: asset.Spot,
// 		Pairs:     []currency.Pair{testPair},
// 		Side:      order.AnySide,
// 	}

// 	_, err := c.GetOrderHistory(context.Background(), &getOrdersRequest)
// 	if sharedtestvalues.AreAPICredentialsSet(c) && err != nil {
// 		t.Errorf("Could not get order history: %s", err)
// 	} else if !sharedtestvalues.AreAPICredentialsSet(c) && err == nil {
// 		t.Error("Expecting an error when no keys are set")
// 	}

// 	getOrdersRequest.Pairs = []currency.Pair{}
// 	_, err = c.GetOrderHistory(context.Background(), &getOrdersRequest)
// 	if sharedtestvalues.AreAPICredentialsSet(c) && err != nil {
// 		t.Errorf("Could not get order history: %s", err)
// 	} else if !sharedtestvalues.AreAPICredentialsSet(c) && err == nil {
// 		t.Error("Expecting an error when no keys are set")
// 	}

// 	getOrdersRequest.Pairs = nil
// 	_, err = c.GetOrderHistory(context.Background(), &getOrdersRequest)
// 	if sharedtestvalues.AreAPICredentialsSet(c) && err != nil {
// 		t.Errorf("Could not get order history: %s", err)
// 	} else if !sharedtestvalues.AreAPICredentialsSet(c) && err == nil {
// 		t.Error("Expecting an error when no keys are set")
// 	}
// }

// // Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// // ----------------------------------------------------------------------------------------------------------------------------

// func TestSubmitOrder(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, c, canManipulateRealOrders)

// 	// limit order
// 	var orderSubmission = &order.Submit{
// 		Exchange: c.Name,
// 		Pair: currency.Pair{
// 			Delimiter: "-",
// 			Base:      currency.BTC,
// 			Quote:     currency.USD,
// 		},
// 		Side:      order.Buy,
// 		Type:      order.Limit,
// 		Price:     1,
// 		Amount:    0.001,
// 		ClientID:  "meowOrder",
// 		AssetType: asset.Spot,
// 	}
// 	response, err := c.SubmitOrder(context.Background(), orderSubmission)
// 	if sharedtestvalues.AreAPICredentialsSet(c) && (err != nil || response.Status != order.New) {
// 		t.Errorf("Order failed to be placed: %v", err)
// 	} else if !sharedtestvalues.AreAPICredentialsSet(c) && err == nil {
// 		t.Error("Expecting an error when no keys are set")
// 	}

// 	// market order from amount
// 	orderSubmission = &order.Submit{
// 		Exchange: c.Name,
// 		Pair: currency.Pair{
// 			Delimiter: "-",
// 			Base:      currency.BTC,
// 			Quote:     currency.USD,
// 		},
// 		Side:      order.Buy,
// 		Type:      order.Market,
// 		Amount:    0.001,
// 		ClientID:  "meowOrder",
// 		AssetType: asset.Spot,
// 	}
// 	response, err = c.SubmitOrder(context.Background(), orderSubmission)
// 	if sharedtestvalues.AreAPICredentialsSet(c) && (err != nil || response.Status != order.New) {
// 		t.Errorf("Order failed to be placed: %v", err)
// 	} else if !sharedtestvalues.AreAPICredentialsSet(c) && err == nil {
// 		t.Error("Expecting an error when no keys are set")
// 	}

// 	// market order from quote amount
// 	orderSubmission = &order.Submit{
// 		Exchange: c.Name,
// 		Pair: currency.Pair{
// 			Delimiter: "-",
// 			Base:      currency.BTC,
// 			Quote:     currency.USD,
// 		},
// 		Side:        order.Buy,
// 		Type:        order.Market,
// 		QuoteAmount: 1,
// 		ClientID:    "meowOrder",
// 		AssetType:   asset.Spot,
// 	}
// 	response, err = c.SubmitOrder(context.Background(), orderSubmission)
// 	if sharedtestvalues.AreAPICredentialsSet(c) && (err != nil || response.Status != order.New) {
// 		t.Errorf("Order failed to be placed: %v", err)
// 	} else if !sharedtestvalues.AreAPICredentialsSet(c) && err == nil {
// 		t.Error("Expecting an error when no keys are set")
// 	}
// }

// func TestCancelExchangeOrder(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, c, canManipulateRealOrders)

// 	var orderCancellation = &order.Cancel{
// 		OrderID:       "1",
// 		WalletAddress: core.BitcoinDonationAddress,
// 		AccountID:     "1",
// 		Pair:          testPair,
// 		AssetType:     asset.Spot,
// 	}

// 	err := c.CancelOrder(context.Background(), orderCancellation)
// 	if !sharedtestvalues.AreAPICredentialsSet(c) && err == nil {
// 		t.Error("Expecting an error when no keys are set")
// 	}
// 	if sharedtestvalues.AreAPICredentialsSet(c) && err != nil {
// 		t.Errorf("Could not cancel orders: %v", err)
// 	}
// }

// func TestCancelAllExchangeOrders(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, c, canManipulateRealOrders)

// 	var orderCancellation = &order.Cancel{
// 		OrderID:       "1",
// 		WalletAddress: core.BitcoinDonationAddress,
// 		AccountID:     "1",
// 		Pair:          testPair,
// 		AssetType:     asset.Spot,
// 	}

// 	resp, err := c.CancelAllOrders(context.Background(), orderCancellation)

// 	if !sharedtestvalues.AreAPICredentialsSet(c) && err == nil {
// 		t.Error("Expecting an error when no keys are set")
// 	}
// 	if sharedtestvalues.AreAPICredentialsSet(c) && err != nil {
// 		t.Errorf("Could not cancel orders: %v", err)
// 	}

// 	if len(resp.Status) > 0 {
// 		t.Errorf("%v orders failed to cancel", len(resp.Status))
// 	}
// }

// func TestModifyOrder(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, c, canManipulateRealOrders)

// 	_, err := c.ModifyOrder(context.Background(),
// 		&order.Modify{AssetType: asset.Spot})
// 	if err == nil {
// 		t.Error("ModifyOrder() Expected error")
// 	}
// }

// func TestWithdraw(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, c, canManipulateRealOrders)

// 	withdrawCryptoRequest := withdraw.Request{
// 		Exchange:    c.Name,
// 		Amount:      -1,
// 		Currency:    currency.BTC,
// 		Description: "WITHDRAW IT ALL",
// 		Crypto: withdraw.CryptoRequest{
// 			Address: core.BitcoinDonationAddress,
// 		},
// 	}

// 	_, err := c.WithdrawCryptocurrencyFunds(context.Background(),
// 		&withdrawCryptoRequest)
// 	if !sharedtestvalues.AreAPICredentialsSet(c) && err == nil {
// 		t.Error("Expecting an error when no keys are set")
// 	}
// 	if sharedtestvalues.AreAPICredentialsSet(c) && err != nil {
// 		t.Errorf("Withdraw failed to be placed: %v", err)
// 	}
// }

// func TestWithdrawFiat(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, c, canManipulateRealOrders)

// 	var withdrawFiatRequest = withdraw.Request{
// 		Amount:   100,
// 		Currency: currency.USD,
// 		Fiat: withdraw.FiatRequest{
// 			Bank: banking.Account{
// 				BankName: "Federal Reserve Bank",
// 			},
// 		},
// 	}

// 	_, err := c.WithdrawFiatFunds(context.Background(), &withdrawFiatRequest)
// 	if !sharedtestvalues.AreAPICredentialsSet(c) && err == nil {
// 		t.Error("Expecting an error when no keys are set")
// 	}
// 	if sharedtestvalues.AreAPICredentialsSet(c) && err != nil {
// 		t.Errorf("Withdraw failed to be placed: %v", err)
// 	}
// }

// func TestWithdrawInternationalBank(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, c, canManipulateRealOrders)

// 	var withdrawFiatRequest = withdraw.Request{
// 		Amount:   100,
// 		Currency: currency.USD,
// 		Fiat: withdraw.FiatRequest{
// 			Bank: banking.Account{
// 				BankName: "Federal Reserve Bank",
// 			},
// 		},
// 	}

// 	_, err := c.WithdrawFiatFundsToInternationalBank(context.Background(),
// 		&withdrawFiatRequest)
// 	if !sharedtestvalues.AreAPICredentialsSet(c) && err == nil {
// 		t.Error("Expecting an error when no keys are set")
// 	}
// 	if sharedtestvalues.AreAPICredentialsSet(c) && err != nil {
// 		t.Errorf("Withdraw failed to be placed: %v", err)
// 	}
// }

// func TestGetDepositAddress(t *testing.T) {
// 	_, err := c.GetDepositAddress(context.Background(), currency.BTC, "", "")
// 	if err == nil {
// 		t.Error("GetDepositAddress() error", err)
// 	}
// }

/*

// TestWsAuth dials websocket, sends login request.
func TestWsAuth(t *testing.T) {
	if !c.Websocket.IsEnabled() && !c.API.AuthenticatedWebsocketSupport || !sharedtestvalues.AreAPICredentialsSet(c) {
		t.Skip(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := c.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	go c.wsReadData()

	err = c.Subscribe([]stream.ChannelSubscription{
		{
			Channel:  "user",
			Currency: testPair,
		},
	})
	if err != nil {
		t.Error(err)
	}
	timer := time.NewTimer(sharedtestvalues.WebsocketResponseDefaultTimeout)
	select {
	case badResponse := <-c.Websocket.DataHandler:
		t.Error(badResponse)
	case <-timer.C:
	}
	timer.Stop()
}

func TestWsSubscribe(t *testing.T) {
	pressXToJSON := []byte(`{
		"type": "subscriptions",
		"channels": [
			{
				"name": "level2",
				"product_ids": [
					"ETH-USD",
					"ETH-EUR"
				]
			},
			{
				"name": "heartbeat",
				"product_ids": [
					"ETH-USD",
					"ETH-EUR"
				]
			},
			{
				"name": "ticker",
				"product_ids": [
					"ETH-USD",
					"ETH-EUR",
					"ETH-BTC"
				]
			}
		]
	}`)
	err := c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsHeartbeat(t *testing.T) {
	pressXToJSON := []byte(`{
		"type": "heartbeat",
		"sequence": 90,
		"last_trade_id": 20,
		"product_id": BTC-USD,
		"time": "2014-11-07T08:19:28.464459Z"
	}`)
	err := c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsStatus(t *testing.T) {
	pressXToJSON := []byte(`{
    "type": "status",
    "products": [
        {
            "id": BTC-USD,
            "base_currency": "BTC",
            "quote_currency": "USD",
            "base_min_size": "0.001",
            "base_max_size": "70",
            "base_increment": "0.00000001",
            "quote_increment": "0.01",
            "display_name": "BTC/USD",
            "status": "online",
            "status_message": null,
            "min_market_funds": "10",
            "max_market_funds": "1000000",
            "post_only": false,
            "limit_only": false,
            "cancel_only": false
        }
    ],
    "currencies": [
        {
            "id": "USD",
            "name": "United States Dollar",
            "min_size": "0.01000000",
            "status": "online",
            "status_message": null,
            "max_precision": "0.01",
            "convertible_to": ["USDC"], "details": {}
        },
        {
            "id": "USDC",
            "name": "USD Coin",
            "min_size": "0.00000100",
            "status": "online",
            "status_message": null,
            "max_precision": "0.000001",
            "convertible_to": ["USD"], "details": {}
        },
        {
            "id": "BTC",
            "name": "Bitcoin",
            "min_size": "0.00000001",
            "status": "online",
            "status_message": null,
            "max_precision": "0.00000001",
            "convertible_to": []
        }
    ]
}`)
	err := c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsTicker(t *testing.T) {
	pressXToJSON := []byte(`{
    "type": "ticker",
    "trade_id": 20153558,
    "sequence": 3262786978,
    "time": "2017-09-02T17:05:49.250000Z",
    "product_id": "BTC-USD",
    "price": "4388.01000000",
    "side": "buy",
    "last_size": "0.03000000",
    "best_bid": "4388",
    "best_ask": "4388.01"
}`)
	err := c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsOrderbook(t *testing.T) {
	pressXToJSON := []byte(`{
    "type": "snapshot",
    "product_id": "BTC-USD",
    "bids": [["10101.10", "0.45054140"]],
    "asks": [["10102.55", "0.57753524"]],
	"time":"2023-08-15T06:46:55.376250Z"
}`)
	err := c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`{
  "type": "l2update",
  "product_id": "BTC-USD",
  "time": "2023-08-15T06:46:57.933713Z",
  "changes": [
    [
      "buy",
      "10101.80000000",
      "0.162567"
    ]
  ]
}`)
	err = c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsOrders(t *testing.T) {
	pressXToJSON := []byte(`{
    "type": "received",
    "time": "2014-11-07T08:19:27.028459Z",
    "product_id": "BTC-USD",
    "sequence": 10,
    "order_id": "d50ec984-77a8-460a-b958-66f114b0de9b",
    "size": "1.34",
    "price": "502.1",
    "side": "buy",
    "order_type": "limit"
}`)
	err := c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`{
    "type": "received",
    "time": "2014-11-09T08:19:27.028459Z",
    "product_id": "BTC-USD",
    "sequence": 12,
    "order_id": "dddec984-77a8-460a-b958-66f114b0de9b",
    "funds": "3000.234",
    "side": "buy",
    "order_type": "market"
}`)
	err = c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`{
    "type": "open",
    "time": "2014-11-07T08:19:27.028459Z",
    "product_id": "BTC-USD",
    "sequence": 10,
    "order_id": "d50ec984-77a8-460a-b958-66f114b0de9b",
    "price": "200.2",
    "remaining_size": "1.00",
    "side": "sell"
}`)
	err = c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`{
    "type": "done",
    "time": "2014-11-07T08:19:27.028459Z",
    "product_id": "BTC-USD",
    "sequence": 10,
    "price": "200.2",
    "order_id": "d50ec984-77a8-460a-b958-66f114b0de9b",
    "reason": "filled",
    "side": "sell",
    "remaining_size": "0"
}`)
	err = c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`{
    "type": "match",
    "trade_id": 10,
    "sequence": 50,
    "maker_order_id": "ac928c66-ca53-498f-9c13-a110027a60e8",
    "taker_order_id": "132fb6ae-456b-4654-b4e0-d681ac05cea1",
    "time": "2014-11-07T08:19:27.028459Z",
    "product_id": "BTC-USD",
    "size": "5.23512",
    "price": "400.23",
    "side": "sell"
}`)
	err = c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`{
    "type": "change",
    "time": "2014-11-07T08:19:27.028459Z",
    "sequence": 80,
    "order_id": "ac928c66-ca53-498f-9c13-a110027a60e8",
    "product_id": "BTC-USD",
    "new_size": "5.23512",
    "old_size": "12.234412",
    "price": "400.23",
    "side": "sell"
}`)
	err = c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
	pressXToJSON = []byte(`{
    "type": "change",
    "time": "2014-11-07T08:19:27.028459Z",
    "sequence": 80,
    "order_id": "ac928c66-ca53-498f-9c13-a110027a60e8",
    "product_id": "BTC-USD",
    "new_funds": "5.23512",
    "old_funds": "12.234412",
    "price": "400.23",
    "side": "sell"
}`)
	err = c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
	pressXToJSON = []byte(`{
  "type": "activate",
  "product_id": "BTC-USD",
  "timestamp": "1483736448.299000",
  "user_id": "12",
  "profile_id": "30000727-d308-cf50-7b1c-c06deb1934fc",
  "order_id": "7b52009b-64fd-0a2a-49e6-d8a939753077",
  "stop_type": "entry",
  "side": "buy",
  "stop_price": "80",
  "size": "2",
  "funds": "50",
  "taker_fee_rate": "0.0025",
  "private": true
}`)
	err = c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestStatusToStandardStatus(t *testing.T) {
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

// func TestParseTime(t *testing.T) {
// 	// Rest examples use 2014-11-07T22:19:28.578544Z" and can be safely
// 	// unmarhsalled into time.Time

// 	// All events except for activate use the above, in the below test
// 	// we'll use their API docs example
// 	r := convert.TimeFromUnixTimestampDecimal(1483736448.299000).UTC()
// 	if r.Year() != 2017 ||
// 		r.Month().String() != "January" ||
// 		r.Day() != 6 {
// 		t.Error("unexpected result")
// 	}
// }

// func TestGetRecentTrades(t *testing.T) {
// 	t.Parallel()
// 	_, err := c.GetRecentTrades(context.Background(), testPair, asset.Spot)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestGetHistoricTrades(t *testing.T) {
// 	t.Parallel()
// 	_, err := c.GetHistoricTrades(context.Background(),
// 		testPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
// 	if err != nil && err != common.ErrFunctionNotSupported {
// 		t.Error(err)
// 	}
// }

*/

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

	err := result.PrepareDateString(time.Unix(1, 1).UTC(), time.Unix(2, 2).UTC(), labelStart, labelEnd)
	if err != nil {
		t.Error(err)
	}
	if fmt.Sprint(expectedResult) != fmt.Sprint(result) {
		t.Errorf(errExpectMismatch, result, expectedResult)
	}

	var newTime time.Time
	err = result.PrepareDateString(newTime, newTime, labelStart, labelEnd)
	if err != nil {
		t.Error(err)
	}

	err = result.PrepareDateString(time.Unix(2, 2).UTC(), time.Unix(1, 1).UTC(), labelStart, labelEnd)
	if err == nil {
		t.Error("expected startafterend error")
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

	result.PreparePagination(pagIn)

	if fmt.Sprint(expectedResult) != fmt.Sprint(result) {
		t.Errorf(errExpectMismatch, result, expectedResult)
	}
}

func TestOrderbookHelper(t *testing.T) {
	t.Parallel()
	req := make(InterOrderDetail, 1)
	req[0][0] = true
	req[0][1] = false
	req[0][2] = true

	_, err := OrderbookHelper(req, 2)
	if err == nil {
		t.Error("expected unable to type assert price error")
	}
	req[0][0] = "egg"
	_, err = OrderbookHelper(req, 2)
	if err == nil {
		t.Error("expected invalid ParseFloat error")
	}
	req[0][0] = "1.1"
	_, err = OrderbookHelper(req, 2)
	if err == nil {
		t.Error("expected unable to type assert amount error")
	}
	req[0][1] = "meow"
	_, err = OrderbookHelper(req, 2)
	if err == nil {
		t.Error("expected invalid ParseFloat error")
	}
	req[0][1] = "2.2"
	_, err = OrderbookHelper(req, 2)
	if err == nil {
		t.Error("expected unable to type assert number of orders error")
	}
	req[0][2] = 3.3
	_, err = OrderbookHelper(req, 2)
	if err != nil {
		t.Error(err)
	}
	_, err = OrderbookHelper(req, 3)
	if err == nil {
		t.Error("expected unable to type assert order ID error")
	}
	req[0][2] = "woof"
	_, err = OrderbookHelper(req, 3)
	if err != nil {
		t.Error(err)
	}
}

func TestPrepareDSL(t *testing.T) {
	t.Parallel()
	var expectedResult Params
	expectedResult.urlVals = map[string][]string{
		"before": {"1"},
		"limit":  {"2"},
	}
	var result Params

	result.urlVals = make(url.Values)

	result.PrepareDSL("before", "1", 2)
	if fmt.Sprint(expectedResult) != fmt.Sprint(result) {
		t.Errorf(errExpectMismatch, result, expectedResult)
	}
}

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

func TestUpdateTickersS(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	err := c.UpdateTickers(context.Background(), asset.Empty)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf(errExpectMismatch, err, asset.ErrNotSupported)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	err = c.UpdateTickers(context.Background(), asset.Futures)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.UpdateOrderbook(context.Background(), currency.Pair{}, asset.Empty)
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Errorf(errExpectMismatch, err, currency.ErrCurrencyPairEmpty)
	}
	_, err = c.UpdateOrderbook(context.Background(), currency.NewPairWithDelimiter("meow", "woof", "-"), asset.Spot)
	if err != nil {
		t.Error(err)
	}
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

	resp := c.ProcessFundingData(accHist, cryptHist)

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

func TestGetRecentTrades(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetRecentTrades(context.Background(), testPair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetHistoricTrades(context.Background(), testPair, asset.Spot, time.Time{}, time.Now())
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
	resp, err := c.PlaceOrder(context.Background(), strconv.FormatInt(time.Now().UnixMilli(), 18)+"GCTModifyOrderTest",
		testPair.String(), order.Sell.String(), "", order.Limit.String(), 0.0000001, 1000000000000, 0, false, time.Time{})
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
	resp, err := c.PlaceOrder(context.Background(), strconv.FormatInt(time.Now().UnixMilli(), 18)+"GCTCancelOrderTest",
		testPair.String(), order.Sell.String(), "", order.Limit.String(), 0.0000001, 1000000000000, 0, false, time.Time{})
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
	resp, err := c.PlaceOrder(context.Background(),
		strconv.FormatInt(time.Now().UnixMilli(), 18)+"GCTCancelBatchOrdersTest", testPair.String(),
		order.Sell.String(), "", order.Limit.String(), 0.0000001, 1000000000000, 0, false, time.Time{})
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
	_, err = c.PlaceOrder(context.Background(),
		strconv.FormatInt(time.Now().UnixMilli(), 18)+"GCTCancelAllOrdersTest", testPair.String(),
		order.Sell.String(), "", order.Limit.String(), 0.0000001, 1000000000000, 0, false, time.Time{})
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

// 8837708	       143.0 ns/op	      24 B/op	       5 allocs/op
// func BenchmarkXxx(b *testing.B) {
// 	for x := 0; x < b.N; x++ {
// 		_ = strconv.FormatInt(60, 10)
// 		_ = strconv.FormatInt(300, 10)
// 		_ = strconv.FormatInt(900, 10)
// 		_ = strconv.FormatInt(3600, 10)
// 		_ = strconv.FormatInt(21600, 10)
// 		_ = strconv.FormatInt(86400, 10)
// 	}
// }

// 8350056	       154.3 ns/op	      24 B/op	       5 allocs/op
// func BenchmarkXxx2(b *testing.B) {
// 	for x := 0; x < b.N; x++ {
// 		_ = strconv.Itoa(60)
// 		_ = strconv.Itoa(300)
// 		_ = strconv.Itoa(900)
// 		_ = strconv.Itoa(3600)
// 		_ = strconv.Itoa(21600)
// 		_ = strconv.Itoa(86400)
// 	}
// }

// const reportType = "rfq-fills"

// // Benchmark3Ifs-8   	1000000000	         0.2556 ns/op	       0 B/op	       0 allocs/op
// // 1000000000	         0.2879 ns/op	       0 B/op	       0 allocs/op
// // Benchmark3Ifs-8   	1000000000	         0.2945 ns/op	       0 B/op	       0 allocs/op
// func Benchmark3Ifs(b *testing.B) {
// 	a := 0
// 	for x := 0; x < b.N; x++ {
// 		if reportType == "fills" || reportType == "otc-fills" || reportType == "rfq-fills" {
// 			a++
// 		}
// 	}
// 	log.Print(a)
// }

// // BenchmarkNeedle-8   	322462062	         3.670 ns/op	       0 B/op	       0 allocs/op
// // BenchmarkNeedle-8   	295766910	         4.467 ns/op	       0 B/op	       0 allocs/op
// // BenchmarkNeedle-8   	137813607	         9.496 ns/op	       0 B/op	       0 allocs/op
// func BenchmarkNeedle(b *testing.B) {
// 	a := 0
// 	for x := 0; x < b.N; x++ {
// 		rTCheck := []string{"fills", "otc-fills", "rfq-fills"}
// 		if common.StringDataCompare(rTCheck, reportType) {
// 			a++
// 		}
// 	}
// 	log.Print(a)
// }

// // BenchmarkIfPrevar-8 		537492	      2807 ns/op	     155 B/op	       2 allocs/op
// // BenchmarkIfPrevar-8		534740	      2674 ns/op	     155 B/op	       2 allocs/op
// func BenchmarkIfPrevar(b *testing.B) {
// 	var str1 string
// 	for x := 0; x < b.N; x++ {
// 		if x%2 == 0 {
// 			str1 = coinbaseDeposits
// 		}
// 		if x%2 == 1 {
// 			str1 = coinbaseWithdrawals
// 		}
// 		str2 := fmt.Sprintf("%s%s/%s/%s", coinbaseV2, coinbaseAccounts, "67e0eaec-07d7-54c4-a72c-2e92826897df",
// 			str1)
// 		_ = str2
// 	}
// }

// // BenchmarkIfDirSet-8		459709	      2634 ns/op	     139 B/op	       1 allocs/op
// // BenchmarkIfDirSet-8		494604	      2576 ns/op	     139 B/op	       1 allocs/op
// func BenchmarkIfDirSet(b *testing.B) {
// 	for x := 0; x < b.N; x++ {
// 		var str2 string
// 		if x%2 == 0 {
// 			str2 = fmt.Sprintf("%s%s/%s/%s", coinbaseV2, coinbaseAccounts, "67e0eaec-07d7-54c4-a72c-2e92826897df",
// 				coinbaseDeposits)
// 		}
// 		if x%2 == 1 {
// 			str2 = fmt.Sprintf("%s%s/%s/%s", coinbaseV2, coinbaseAccounts, "67e0eaec-07d7-54c4-a72c-2e92826897df",
// 				coinbaseWithdrawals)
// 		}
// 		_ = str2
// 	}
// }

// // BenchmarkSwitchPrevar-8	453200	      2623 ns/op	     155 B/op	       2 allocs/op
// // BenchmarkSwitchPrevar-8 	556477	      2077 ns/op	     156 B/op	       3 allocs/op
// func BenchmarkSwitchPrevar(b *testing.B) {
// 	var str1 string
// 	for x := 0; x < b.N; x++ {
// 		switch x % 2 {
// 		case 0:
// 			str1 = coinbaseDeposits
// 		case 1:
// 			str1 = coinbaseWithdrawals
// 		}
// 		str2 := fmt.Sprintf("%s%s/%s/%s", coinbaseV2, coinbaseAccounts, "67e0eaec-07d7-54c4-a72c-2e92826897df",
// 			str1)
// 		_ = str2
// 	}
// }

// // BenchmarkSwitchDirSet-8	432816	      2371 ns/op	     139 B/op	       1 allocs/op
// // BenchmarkSwitchDirSet-8 	544873	      2071 ns/op	     140 B/op	       2 allocs/op
// func BenchmarkSwitchDirSet(b *testing.B) {
// 	for x := 0; x < b.N; x++ {
// 		var str2 string
// 		switch x % 2 {
// 		case 0:
// 			str2 = fmt.Sprintf("%s%s/%s/%s", coinbaseV2, coinbaseAccounts, "67e0eaec-07d7-54c4-a72c-2e92826897df",
// 				coinbaseDeposits)
// 		case 1:
// 			str2 = fmt.Sprintf("%s%s/%s/%s", coinbaseV2, coinbaseAccounts, "67e0eaec-07d7-54c4-a72c-2e92826897df",
// 				coinbaseWithdrawals)
// 		}
// 		_ = str2
// 	}
// }
