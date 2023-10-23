package coinbasepro

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
)

var (
	c        = &CoinbasePro{}
	testPair = currency.NewPairWithDelimiter(currency.BTC.String(), currency.USD.String(), "-")
)

// Please supply your APIKeys here for better testing
const (
	apiKey                  = ""
	apiSecret               = ""
	clientID                = "" // passphrase you made at API CREATION
	canManipulateRealOrders = true
	testingInSandbox        = true
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
		log.Fatal("coinbasepro load config error", err)
	}
	gdxConfig, err := cfg.GetExchangeConfig("CoinbasePro")
	if err != nil {
		log.Fatal("coinbasepro Setup() init error")
	}
	if apiKey != "" {
		gdxConfig.API.Credentials.Key = apiKey
		gdxConfig.API.Credentials.Secret = apiSecret
		gdxConfig.API.Credentials.ClientID = clientID
		gdxConfig.API.AuthenticatedSupport = true
		gdxConfig.API.AuthenticatedWebsocketSupport = true
	}
	c.Websocket = sharedtestvalues.NewTestWebsocket()
	err = c.Setup(gdxConfig)
	if err != nil {
		log.Fatal("CoinbasePro setup error", err)
	}
	c.Verbose = true
	os.Exit(m.Run())
}

func TestStart(t *testing.T) {
	t.Parallel()
	err := c.Start(context.Background(), nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrNilPointer)
	}
	var testWg sync.WaitGroup
	err = c.Start(context.Background(), &testWg)
	if err != nil {
		t.Fatal(err)
	}
	testWg.Wait()
}

func TestGetAllAccounts(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetAllAccounts(context.Background())
	if err != nil {
		t.Error("CoinBasePro GetAllAccounts() error", err)
	}
	assert.NotEmpty(t, resp, "CoinBasePro GetAllAccounts() error, expected a non-empty response")
}

func TestGetAccountByID(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	longResp, err := c.GetAllAccounts(context.Background())
	if err != nil {
		t.Error("CoinBasePro GetAllAccounts() error", err)
	}
	shortResp, err := c.GetAccountByID(context.Background(), longResp[0].ID)
	if err != nil {
		t.Error("CoinBasePro GetAccountByID() error", err)
	}
	if *shortResp != longResp[0] {
		t.Error("CoinBasePro GetAccountByID() error, mismatched responses")
	}
}

func TestGetHolds(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	accID, err := c.GetAllAccounts(context.Background())
	if err != nil {
		t.Error("CoinBasePro GetAllAccounts() error", err)
	}
	resp, err := c.GetHolds(context.Background(), accID[1].ID, pageNone, "1", 2)
	if err != nil {
		t.Error("CoinBasePro GetHolds() error", err)
	}
	log.Printf("%+v", resp)
}

func TestGetAccountLedger(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	accID, err := c.GetAllAccounts(context.Background())
	if err != nil {
		t.Error("CoinBasePro GetAllAccounts() error", err)
	}
	_, err = c.GetAccountLedger(context.Background(), accID[0].ID, pageBefore, "", "a",
		time.Unix(1, 1), time.Now(), 3)
	if err != nil {
		t.Error("CoinBasePro GetAccountLedger() error", err)
	}
}

func TestGetAccountTransfers(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	accID, err := c.GetAllAccounts(context.Background())
	if err != nil {
		t.Error("CoinBasePro GetAllAccounts() error", err)
	}
	_, err = c.GetAccountTransfers(context.Background(), accID[0].ID, "", "", "", 3)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAddressBook(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetAddressBook(context.Background())
	if err != nil {
		t.Error("CoinBasePro GetAddressBook() error", err)
	}
}

func TestAddAddresses(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	var req [1]AddAddressRequest
	var err error
	req[0], err = PrepareAddAddress("this test", "is not", "properly", "implemented", "Coinbase", false)
	if err != nil {
		t.Error(err)
	}
	_, err = c.AddAddresses(context.Background(), req[:])
	if err != nil {
		t.Error("CoinBasePro AddAddresses() error", err)
	}
}

func TestDeleteAddress(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	err := c.DeleteAddress(context.Background(), "Test not properly implemented")
	if err != nil {
		t.Error("CoinBasePro DeleteAddress() error", err)
	}
}

func TestGetCoinbaseWallets(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetCoinbaseWallets(context.Background())
	if err != nil {
		t.Error("CoinBasePro GetCoinbaseAccounts() error", err)
	}
}

func TestGenerateCryptoAddress(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	accID, err := c.GetCoinbaseWallets(context.Background())
	if err != nil {
		t.Error("CoinBasePro GetAllAccounts() error", err)
	}
	_, err = c.GenerateCryptoAddress(context.Background(), accID[0].ID, "", "")
	if err != nil {
		t.Error("CoinBasePro GenerateCryptoAddress() error", err)
	}
}

func TestConvertCurrency(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	_, err := c.ConvertCurrency(context.Background(), "This test", " is not", "implemented", "quite yet", 0)
	if err != nil {
		t.Error("CoinBasePro ConvertCurrency() error", err)
	}
}

func TestGetConversionByID(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetConversionByID(context.Background(), "Test not", "implemented yet")
	if err == nil {
		t.Error("This really should have failed since a proper ID wasn't supplied.")
	}
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
	_, err := c.DepositViaCoinbase(context.Background(), "This test", "is not", "yet implemented", 1)
	if err != nil {
		t.Error("CoinBasePro DepositViaCoinbase() error", err)
	}
}

func TestDepositViaPaymentMethod(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	_, err := c.DepositViaPaymentMethod(context.Background(), "This test", "is not", "yet implemented", 1)
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
	_, err := c.GetAllTransfers(context.Background(), "", "", "", "", 3)
	if err != nil {
		t.Error("CoinBasePro GetAllTransfers() error", err)
	}
}

func TestGetTransferByID(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetTransferByID(context.Background(), "Not yet implemented")
	if err == nil {
		t.Error("This really should have failed since a proper ID wasn't supplied.")
	}
}

func TestSendTravelInfoForTransfer(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	_, err := c.SendTravelInfoForTransfer(context.Background(), "This test", "is not", "yet implemented")
	if err != nil {
		t.Error("CoinBasePro SendTravelInfoForTransfer() error", err)
	}
}

func TestWithdrawViaCoinbase(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	_, err := c.WithdrawViaCoinbase(context.Background(), "This test", "is not", "yet implemented", 1)
	if err != nil {
		t.Error("CoinBasePro WithdrawViaCoinbase() error", err)
	}
}

func TestWithdrawCrypto(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	_, err := c.WithdrawCrypto(context.Background(), "This", "test", "is", "not", "implemented", "yet", 1,
		false, false, 2)
	if err != nil {
		t.Error("CoinBasePro WithdrawCrypto() error", err)
	}
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
	_, err = c.GetWithdrawalFeeEstimate(context.Background(), "This test is not", "yet implemented",
		"due to not knowing a valid network string")
	if err == nil {
		t.Error("This should have errored out due to an improper network string")
	}
}

func TestWithdrawViaPaymentMethod(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	_, err := c.WithdrawViaPaymentMethod(context.Background(), "This test", "is not", "yet implemented", 1)
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
	if resp.TakerFeeRate == 0 {
		t.Error("CoinBasePro GetFees() error, expected non-zero value for taker fee rate")
	}
}

func TestGetFills(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetFills(context.Background(), "", "", "", "", "", 0, time.Time{}, time.Time{})
	if err == nil {
		t.Error("CoinBasePro GetFills() error, expected error due to empty order and product ID")
	}

	_, err = c.GetFills(context.Background(), "1", "", "", "", "", 0, time.Time{}, time.Time{})
	if err == nil {
		t.Error("CoinBasePro GetFills() error, expected error due to null time range")
	}

	_, err = c.GetFills(context.Background(), "", testPair.String(), "", "", "spot", 0, time.Unix(1, 1), time.Now())
	if err != nil {
		t.Error("CoinBasePro GetFills() error", err)
	}
}

func TestGetAllOrders(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	status := []string{"open", "pending", "active", "done"}

	_, err := c.GetAllOrders(context.Background(), "", "", "", "", "", "", "", time.Unix(1, 1), time.Now(), 5, status)
	if err != nil {
		t.Error("CoinBasePro GetAllOrders() error", err)
	}
}

func TestCancelAllExistingOrders(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	_, err := c.CancelAllExistingOrders(context.Background(), "This test is not", "yet implemented")
	if err != nil {
		t.Error("CoinBasePro CancelAllExistingOrders() error", err)
	}
}

func TestPlaceOrder(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)

	for x := 0; x < 550; x++ {
		_, _ = c.PlaceOrder(context.Background(), "this", "", "sell", "BTC-USD", "", "implemented", "", "sandbox",
			"testing", 0, 2<<30, 1, 0, false)

	}

	// log.Printf("Response: %+v\nError: %+v", resp, err)
}

func TestGetOrderByID(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetOrderByID(context.Background(), "940d4bf3-933b-4714-a702-155f82c3e739", "spot", false)

	log.Printf("Response: %+v\nError: %+v", resp, err)
}

func TestCancelExistingOrder(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	_, err := c.CancelExistingOrder(context.Background(), "This test", "is not", "yet implemented", true)
	if err != nil {
		t.Error("CoinBasePro CancelExistingOrder() error", err)
	}
}

func TestGetSignedPrices(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	resp, err := c.GetSignedPrices(context.Background())
	if err != nil {
		t.Error("CoinBasePro GetSignedPrices() error", err)
	}
	if resp.Timestamp == "" {
		t.Error("CoinBasePro GetSignedPrices() error, expected non-empty timestamp")
	}
}

func TestGetAllProducts(t *testing.T) {
	resp, err := c.GetAllProducts(context.Background(), "")
	if err != nil {
		t.Error("Coinbase, GetAllProducts() Error:", err)
	}
	if resp[0].ID == "" {
		t.Error("Coinbase, GetAllProducts() Error, expected non-empty string")
	}
}

func TestGetProductByID(t *testing.T) {
	_, err := c.GetProductByID(context.Background(), "")
	if err == nil {
		t.Error("Coinbase, GetProductByID() Error, expected an error due to nonexistent pair")
	}
	resp, err := c.GetProductByID(context.Background(), "BTC-USD")
	if err != nil {
		t.Error("Coinbase, GetProductByID() Error:", err)
	}
	if resp.ID != "BTC-USD" {
		t.Error("Coinbase, GetProductByID() Error, expected BTC-USD")
	}
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
	_, err = c.GetOrderbook(context.Background(), testPair.String(), 1)
	if err != nil {
		t.Error("Coinbase, GetOrderbook() Error", err)
	}
	_, err = c.GetOrderbook(context.Background(), testPair.String(), 3)
	if err != nil {
		t.Error("Coinbase, GetOrderbook() Error", err)
	}
}

func TestGetHistoricRates(t *testing.T) {
	_, err := c.GetHistoricRates(context.Background(), "", 0, time.Time{}, time.Time{})
	if err == nil {
		t.Error("Coinbase, GetHistoricRates() Error, expected an error due to nonexistent pair")
	}
	_, err = c.GetHistoricRates(context.Background(), testPair.String(), 0, time.Now(), time.Unix(1, 1))
	if err == nil {
		t.Error("Coinbase, GetHistoricRates() Error, expected an error due to invalid time")
	}
	_, err = c.GetHistoricRates(context.Background(), testPair.String(), 2<<60-2<<20, time.Time{}, time.Time{})
	if err == nil {
		t.Error("Coinbase, GetHistoricRates() Error, expected an error due to invalid granularity")
	}
	_, err = c.GetHistoricRates(context.Background(), "Invalid pair.woof", 60, time.Time{}, time.Time{})
	if err == nil {
		t.Error("Coinbase, GetHistoricRates() Error, expected an error due to invalid pair")
	}
	resp, err := c.GetHistoricRates(context.Background(), testPair.String(), 0, time.Unix(1, 1), time.Now())
	if err != nil {
		t.Error("Coinbase, GetHistoricRates() Error", err)
	}
	if resp[0].High == 0 {
		t.Error("Coinbase, GetHistoricRates() Error, expected non-zero value")
	}
}

func TestGetStats(t *testing.T) {
	_, err := c.GetStats(context.Background(), "")
	if err == nil {
		t.Error("Coinbase, GetStats() Error, expected an error due to nonexistent pair")
	}
	_, err = c.GetStats(context.Background(), testPair.String())
	if err != nil {
		t.Error("GetStats() error", err)
	}
}

func TestGetTicker(t *testing.T) {
	_, err := c.GetTicker(context.Background(), "")
	if err == nil {
		t.Error("Coinbase, GetTicker() Error, expected an error due to nonexistent pair")
	}
	_, err = c.GetTicker(context.Background(), testPair.String())
	if err != nil {
		t.Error("GetTicker() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	_, err := c.GetTrades(context.Background(), "", "", "", 1)
	if err == nil {
		t.Error("Coinbase, GetTrades() Error, expected an error due to nonexistent pair")
	}
	_, err = c.GetTrades(context.Background(), testPair.String(), "", "", 1)
	if err != nil {
		t.Error("GetTrades() error", err)
	}
}

func TestGetAllProfiles(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	active := true
	_, err := c.GetAllProfiles(context.Background(), &active)
	if err != nil {
		t.Error("GetAllProfiles() error", err)
	}
}

func TestCreateAProfile(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	_, err := c.CreateAProfile(context.Background(), "")
	if err == nil {
		t.Error("Coinbase, CreateAProfile() Error, expected an error due to empty name")
	}
	// The names 'default' and 'margin' are reserved, so consider using those for tests
	_, err = c.CreateAProfile(context.Background(), "GCT Test Profile")
	if err != nil {
		t.Error("CreateAProfile() error", err)
	}
}

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
	if resp2.ID != resp[0].ID {
		t.Error("GetProfileByID() error, expected matching ID's")
	}
}

func TestRenameProfile(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	_, err := c.RenameProfile(context.Background(), "", "")
	if err == nil {
		t.Error("Coinbase, RenameProfile() Error, expected an error due to empty fields")
	}
	_, err = c.RenameProfile(context.Background(), "this test has", "not been implemented")
	if err == nil {
		t.Error("Coinbase, RenameProfile() Error, expected an error due to un-implemented test")
	}
}

func TestDeleteProfile(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c, canManipulateRealOrders)
	_, err := c.DeleteProfile(context.Background(), "", "")
	if err == nil {
		t.Error("Coinbase, DeleteProfile() Error, expected an error due to empty fields")
	}
	_, err = c.DeleteProfile(context.Background(), "this test has", "not been implemented")
	if err == nil {
		t.Error("Coinbase, DeleteProfile() Error, expected an error due to un-implemented test")
	}
}

func TestGetReport(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	prof, err := c.GetAllProfiles(context.Background(), nil)
	if err != nil {
		t.Error("GetAllProfiles() error", err)
	}
	_, err = c.GetAllReports(context.Background(), prof[0].ID, "account", time.Time{}, 1000, false)
	if err != nil {
		t.Error("GetAllReports() error", err)
	}
}

func TestCreateReport(t *testing.T) {
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, c, canManipulateRealOrders)
	prof, err := c.GetAllProfiles(context.Background(), nil)
	if err != nil {
		t.Error("GetAllProfiles() error", err)
	}
	_, err = c.CreateReport(context.Background(), "this", "test", "is", "not", prof[0].ID, "yet", "implemented",
		time.Time{}, time.Time{}, time.Time{})
	if err == nil {
		t.Error("Coinbase, CreateReport() Error, expected an error due to un-implemented test")
	}
}

func TestGetReportByID(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	prof, err := c.GetAllProfiles(context.Background(), nil)
	if err != nil {
		t.Error("GetAllProfiles() error", err)
	}
	resp, err := c.GetAllReports(context.Background(), prof[0].ID, "account", time.Time{}, 1000, false)
	if err != nil {
		t.Error("GetAllReports() error", err)
	}
	if len(resp) == 0 {
		t.Log("No reports found, skipping test")
	} else {
		_, err = c.GetReportByID(context.Background(), resp[0].ID)
		if err != nil {
			t.Error("GetReportByID() error", err)
		}
	}
}

func TestGetTravelRules(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	_, err := c.GetTravelRules(context.Background(), "", "", "", 0)
	if err != nil {
		t.Error("GetTravelRules() error", err)
	}
}

func TestCreateTravelRule(t *testing.T) {
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, c, canManipulateRealOrders)
	_, err := c.CreateTravelRule(context.Background(), "this test", "not yet", "implemented")
	if err == nil {
		t.Error("Coinbase, CreateTravelRule() Error, expected an error due to unimplemented test")
	}
}

func TestDeleteTravelRule(t *testing.T) {
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, c, canManipulateRealOrders)
	err := c.DeleteTravelRule(context.Background(), "this test is not yet implemented")
	if err == nil {
		t.Error("Coinbase, DeleteTravelRule() Error, expected an error due to unimplemented test")
	}
}

func TestGetExchangeLimits(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)
	acc, err := c.GetAllAccounts(context.Background())
	if err != nil {
		t.Error("GetAllAccounts() error", err)
	}
	_, err = c.GetExchangeLimits(context.Background(), acc[0].ID)
	if err != nil {
		t.Error("GetExchangeLimits() error", err)
	}
}

func TestUpdateSettlementPreference(t *testing.T) {
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, c, canManipulateRealOrders)
	_, err := c.UpdateSettlementPreference(context.Background(), "this test", "not implemented")
	if err == nil {
		t.Error("Coinbase, UpdateSettlementPreference() Error, expected an error due to unimplemented test")
	}
}

func TestGetAllWrappedAssets(t *testing.T) {
	_, err := c.GetAllWrappedAssets(context.Background())
	if err != nil {
		t.Error("GetAllWrappedAssets() error", err)
	}
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
	_, err = c.CreateStakeWrap(context.Background(), "this test", "is not implemented", 1)
	if err == nil {
		t.Error("Coinbase, CreateStakeWrap() Error, expected an error due to unimplemented test")
	}
}

func TestGetStakeWrapByID(t *testing.T) {
	resp, err := c.GetAllStakeWraps(context.Background(), "", "ETH", "CBETH", "", time.Time{}, 1)
	if err != nil {
		t.Error("GetAllStakeWraps() error", err)
	}
	if len(resp) == 0 {
		t.Log("No stake wraps found, skipping test")
	} else {
		_, err = c.GetStakeWrapByID(context.Background(), resp[0].ID)
		if err != nil {
			t.Error("GetStakeWrapByID() error", err)
		}
	}
}

func TestGetWrappedAssetByID(t *testing.T) {
	_, err := c.GetWrappedAssetByID(context.Background(), "")
	if err == nil {
		t.Error("Coinbase, GetWrappedAssetByID() Error, expected an error due to empty fields")
	}
	_, err = c.GetWrappedAssetByID(context.Background(), "CBETH")
	if err != nil {
		t.Error("GetWrappedAssetByID() error", err)
	}
}

func TestGetWrappedAssetConversionRate(t *testing.T) {
	_, err := c.GetWrappedAssetConversionRate(context.Background(), "")
	if err == nil {
		t.Error("Coinbase, GetWrappedAssetConversionRate() Error, expected an error due to empty fields")
	}
	_, err = c.GetWrappedAssetConversionRate(context.Background(), "CBETH")
	if err != nil {
		t.Error("GetWrappedAssetConversionRate() error", err)
	}
}

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
// 		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
// 	}

// 	if st.IsZero() {
// 		t.Fatal("expected a time")
// 	}
// }

// func TestAuthRequests(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCredentialsUnset(t, c)

// 	_, err := c.GetAllAccounts(context.Background())
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
		"product_id": "BTC-USD",
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
            "id": "BTC-USD",
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

func TestParseTime(t *testing.T) {
	// Rest examples use 2014-11-07T22:19:28.578544Z" and can be safely
	// unmarhsalled into time.Time

	// All events except for activate use the above, in the below test
	// we'll use their API docs example
	r := convert.TimeFromUnixTimestampDecimal(1483736448.299000).UTC()
	if r.Year() != 2017 ||
		r.Month().String() != "January" ||
		r.Day() != 6 {
		t.Error("unexpected result")
	}
}

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
		t.Errorf("CoinBasePro PrepareDSL(), Expected: %v, Received: %v", expectedResult, result)
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

	err := result.PrepareDateString(time.Unix(1, 1).UTC(), time.Unix(2, 2).UTC())
	if err != nil {
		t.Error("CoinBasePro PrepareDateString() error", err)
	}
	if fmt.Sprint(expectedResult) != fmt.Sprint(result) {
		t.Errorf("CoinBasePro PrepareDateString(), Expected: %v, Received: %v", expectedResult, result)
	}

	var newTime time.Time
	err = result.PrepareDateString(newTime, newTime)
	if err != nil {
		t.Error("CoinBasePro PrepareDateString() error", err)
	}

	err = result.PrepareDateString(time.Unix(2, 2).UTC(), time.Unix(1, 1).UTC())
	if err == nil {
		t.Error("CoinBasePro PrepareDateString() expected StartAfterEnd error")
	}
}

func TestPrepareProfIDAndProdID(t *testing.T) {
	t.Parallel()
	var expectedResult Params
	expectedResult.urlVals = map[string][]string{
		"profile_id": {"123"},
		"product_id": {"BTC-USD"},
	}
	var result Params
	result.urlVals = make(url.Values)

	result.PrepareProfIDAndProdID("123", "BTC-USD")
	if fmt.Sprint(expectedResult) != fmt.Sprint(result) {
		t.Errorf("CoinBasePro PrepareProfIDAndProdID(), Expected: %v, Received: %v", expectedResult, result)
	}
}

func TestPrepareAddAddress(t *testing.T) {
	t.Parallel()

	_, err := PrepareAddAddress("", "", "", "", "", false)
	if err == nil {
		t.Error("CoinBasePro PrepareAddAddress() Expected error for empty address")
	}
	_, err = PrepareAddAddress("", "test", "", "", "meow", false)
	if err == nil {
		t.Error("CoinBasePro PrepareAddAddress() Expected error for invalid vaspID")
	}

	expectedResult := AddAddressRequest{"test", To{"woof", "meow"}, "whinny", false, "Coinbase"}
	result, err := PrepareAddAddress("test", "woof", "meow", "whinny", "Coinbase", false)
	if err != nil {
		t.Error("CoinBasePro PrepareAddAddress() error", err)
	}
	if fmt.Sprint(expectedResult) != fmt.Sprint(result) {
		t.Errorf("CoinBasePro PrepareAddAddress(), Expected: %v, Received: %v", expectedResult, result)
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
		t.Error("CoinBasePro OrderbookHelper(), expected unable to type assert price error")
	}
	req[0][0] = "egg"
	_, err = OrderbookHelper(req, 2)
	if err == nil {
		t.Error("CoinBasePro OrderbookHelper(), expected invalid ParseFloat error")
	}
	req[0][0] = "1.1"
	_, err = OrderbookHelper(req, 2)
	if err == nil {
		t.Error("CoinBasePro OrderbookHelper(), expected unable to type assert amount error")
	}
	req[0][1] = "meow"
	_, err = OrderbookHelper(req, 2)
	if err == nil {
		t.Error("CoinBasePro OrderbookHelper(), expected invalid ParseFloat error")
	}
	req[0][1] = "2.2"
	_, err = OrderbookHelper(req, 2)
	if err == nil {
		t.Error("CoinBasePro OrderbookHelper(), expected unable to type assert number of orders error")
	}
	req[0][2] = 3.3
	_, err = OrderbookHelper(req, 2)
	if err != nil {
		t.Error("CoinBasePro OrderbookHelper() error", err)
	}
	_, err = OrderbookHelper(req, 3)
	if err == nil {
		t.Error("CoinBasePro OrderbookHelper(), expected unable to type assert order ID error")
	}
	req[0][2] = "woof"
	_, err = OrderbookHelper(req, 3)
	if err != nil {
		t.Error("CoinBasePro OrderbookHelper() error", err)
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
