package deribit

import (
	"errors"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
	btcInstrument           = "BTC-25MAR22"
	btcCurrency             = "BTC"
	ethCurrency             = "ETH"
)

var d Deribit

func TestMain(m *testing.M) {
	d.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}

	exchCfg, err := cfg.GetExchangeConfig("Deribit")
	if err != nil {
		log.Fatal(err)
	}

	exchCfg.API.AuthenticatedSupport = true
	exchCfg.API.AuthenticatedWebsocketSupport = true
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret
	d.API.Credentials.Key = apiKey
	d.API.Credentials.Secret = apiSecret

	err = d.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

func areTestAPIKeysSet() bool {
	return d.ValidateAPICredentials()
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	d.Verbose = true
	pairs, err := d.FetchTradablePairs(asset.Futures)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(pairs)

	_, err = d.FetchTradablePairs(asset.Spot)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("expected: %v, received %v", asset.ErrNotSupported, err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	d.Verbose = true
	pairs, err := d.UpdateTicker(currency.Pair{}, asset.Futures)
	if err != nil {
		t.Error(err)
	}

	_, err = d.UpdateTicker(currency.Pair{}, asset.Spot)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("expected: %v, received %v", asset.ErrNotSupported, err)
	}
	fmt.Println(pairs)
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	d.Verbose = true

	cp, err := currency.NewPairFromString(btcInstrument)
	if err != nil {
		t.Error(err)
	}

	fmtPair, err := d.FormatExchangeCurrency(cp, asset.Futures)
	if err != nil {
		t.Error(err)
	}

	obData, err := d.UpdateOrderbook(fmtPair, asset.Futures)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(obData)

	_, err = d.UpdateOrderbook(fmtPair, asset.Spot)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("expected: %v, received %v", asset.ErrNotSupported, err)
	}
}

// Implement tests for API endpoints below

func TestGetBookSummaryByCurrency(t *testing.T) {
	t.Parallel()
	_, err := d.GetBookSummaryByCurrency(btcCurrency, "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetBookSummaryByInstrument(t *testing.T) {
	t.Parallel()
	data, err := d.GetBookSummaryByInstrument(btcInstrument)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(data)
}

func TestGetContractSize(t *testing.T) {
	t.Parallel()
	data, err := d.GetContractSize(btcInstrument)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(data)
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	_, err := d.GetCurrencies()
	if err != nil {
		t.Error(err)
	}
}

func TestGetFundingChartData(t *testing.T) {
	t.Parallel()
	_, err := d.GetFundingChartData("BTC-PERPETUAL", "8h")
	if err != nil {
		t.Error(err)
	}
}

func TestGetFundingRateValue(t *testing.T) {
	t.Parallel()
	_, err := d.GetFundingRateValue("BTC-PERPETUAL", time.Now().Add(-time.Hour*8), time.Now())
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetFundingRateValue("BTC-PERPETUAL", time.Now(), time.Now().Add(-time.Hour*8))
	if !errors.Is(err, errStartTimeCannotBeAfterEndTime) {
		t.Errorf("expected: %v, received %v", errStartTimeCannotBeAfterEndTime, err)
	}
}

func TestGetHistoricalVolatility(t *testing.T) {
	t.Parallel()
	_, err := d.GetHistoricalVolatility(btcCurrency)
	if err != nil {
		t.Error(err)
	}
}

func TestGetIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := d.GetIndexPrice("btc_usd")
	if err != nil {
		t.Error(err)
	}
}

func TestGetIndexPriceNames(t *testing.T) {
	t.Parallel()
	_, err := d.GetIndexPriceNames()
	if err != nil {
		t.Error(err)
	}
}

func TestGetInstrumentData(t *testing.T) {
	t.Parallel()
	_, err := d.GetInstrumentData(btcInstrument)
	if err != nil {
		t.Error(err)
	}
}

func TestGetInstrumentsData(t *testing.T) {
	t.Parallel()
	_, err := d.GetInstrumentsData(btcCurrency, "", false)
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetInstrumentsData(btcCurrency, "option", true)
	if err != nil {
		t.Error(err)
	}
}

func TestGetLastSettlementsByCurrency(t *testing.T) {
	t.Parallel()
	_, err := d.GetLastSettlementsByCurrency(btcCurrency, "", "", 0, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetLastSettlementsByCurrency(btcCurrency, "delivery", "5", 0, time.Now().Add(-time.Hour))
	if err != nil {
		t.Error(err)
	}
}

func TestGetLastSettlementsByInstrument(t *testing.T) {
	t.Parallel()
	_, err := d.GetLastSettlementsByInstrument(btcInstrument, "", "", 0, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetLastSettlementsByInstrument(btcInstrument, "settlement", "5", 0, time.Now().Add(-2*time.Hour))
	if err != nil {
		t.Error(err)
	}
}

func TestGetLastTradesByCurrency(t *testing.T) {
	t.Parallel()
	_, err := d.GetLastTradesByCurrency(btcCurrency, "", "", "", "", 0, false)
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetLastTradesByCurrency(btcCurrency, "option", "36798", "36799", "asc", 0, true)
	if err != nil {
		t.Error(err)
	}
}

func TestGetLastTradesByCurrencyAndTime(t *testing.T) {
	t.Parallel()
	_, err := d.GetLastTradesByCurrencyAndTime(btcCurrency, "", "", 0, false,
		time.Now().Add(-8*time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetLastTradesByCurrencyAndTime(btcCurrency, "option", "asc", 25, false,
		time.Now().Add(-8*time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestGetLastTradesByInstrument(t *testing.T) {
	t.Parallel()
	_, err := d.GetLastTradesByInstrument(btcInstrument, "", "", "", 0, false)
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetLastTradesByInstrument(btcInstrument, "30500", "31500", "desc", 0, true)
	if err != nil {
		t.Error(err)
	}
}

func TestGetLastTradesByInstrumentAndTime(t *testing.T) {
	t.Parallel()
	_, err := d.GetLastTradesByInstrumentAndTime(btcInstrument, "", 0, false,
		time.Now().Add(-8*time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetLastTradesByInstrumentAndTime(btcInstrument, "asc", 0, false,
		time.Now().Add(-8*time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderbookData(t *testing.T) {
	t.Parallel()
	_, err := d.GetOrderbookData(btcInstrument, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTradeVolumes(t *testing.T) {
	t.Parallel()
	_, err := d.GetTradeVolumes(false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTradingViewChartData(t *testing.T) {
	t.Parallel()
	_, err := d.GetTradingViewChartData(btcInstrument, "60", time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestGetVolatilityIndexData(t *testing.T) {
	t.Parallel()
	_, err := d.GetVolatilityIndexData(btcCurrency, "60", time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestGetPublicTicker(t *testing.T) {
	t.Parallel()
	_, err := d.GetPublicTicker(btcInstrument)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountSummary(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.GetAccountSummary(btcCurrency, false)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestCancelTransferByID(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.CancelTransferByID(btcCurrency, "", 23487)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestGetTransfers(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.GetTransfers(btcCurrency, 0, 0)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestCancelWithdrawal(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.CancelWithdrawal(btcCurrency, 123844)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestCreateDepositAddress(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.CreateDepositAddress(btcCurrency)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestGetCurrentDepositAddress(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.GetCurrentDepositAddress(btcCurrency)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestGetDeposits(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.GetDeposits(btcCurrency, 25, 0)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestGetWithdrawals(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.GetWithdrawals(btcCurrency, 25, 0)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestSubmitTransferToSubAccount(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.SubmitTransferToSubAccount(btcCurrency, 0.01, 13434)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestSubmitTransferToUser(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.SubmitTransferToUser(btcCurrency, "", 0.001, 13434)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestSubmitWithdraw(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.SubmitWithdraw(btcCurrency, "incorrectAddress", "", "", 0.001)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestChangeAPIKeyName(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.ChangeAPIKeyName(1, "TestKey123")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestChangeScopeInAPIKey(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.ChangeScopeInAPIKey(1, "account:read_write")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestChangeSubAccountName(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.ChangeSubAccountName(1, "TestingSubAccount")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestCreateAPIKey(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.CreateAPIKey("account:read_write", "TestingSubAccount", false)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestCreateSubAccount(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.CreateSubAccount()
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestDisableAPIKey(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.DisableAPIKey(1)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestDisableTFAForSubAccount(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	// Use with caution will reduce the security of the account
	a, err := d.DisableTFAForSubAccount(1)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestEnableAffiliateProgram(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.EnableAffiliateProgram()
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestEnableAPIKey(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.EnableAPIKey(1)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestGetAffiliateProgramInfo(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.GetAffiliateProgramInfo(1)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestGetEmailLanguage(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.GetEmailLanguage()
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestGetNewAnnouncements(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.GetNewAnnouncements()
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestGetPosition(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.GetPosition(btcInstrument)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestGetSubAccounts(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.GetSubAccounts(false)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestGetPositions(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.GetPositions(btcCurrency, "option")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
	_, err = d.GetPositions(ethCurrency, "")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestGetTransactionLog(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.GetTransactionLog(btcCurrency, "trade", time.Now().Add(-24*time.Hour), time.Now(), 5, 0)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
	a, err = d.GetTransactionLog(btcCurrency, "trade", time.Now().Add(-24*time.Hour), time.Now(), 0, 0)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestListAPIKeys(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.ListAPIKeys("")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestRemoveAPIKey(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.RemoveAPIKey(1)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestRemoveSubAccount(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.RemoveSubAccount(1)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestResetAPIKey(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.ResetAPIKey(1)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestSetAnnouncementAsRead(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.SetAnnouncementAsRead(1)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestSetEmailForSubAccount(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.SetEmailForSubAccount(1, "wrongemail@wrongemail.com")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestSetEmailLanguage(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.SetEmailLanguage("ja")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestSetPasswordForSubAccount(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	// Caution! This may reduce the security of the subaccount
	a, err := d.SetPasswordForSubAccount(1, "randompassword123")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestToggleNotificationsFromSubAccount(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.ToggleNotificationsFromSubAccount(1, false)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestToggleSubAccountLogin(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.ToggleSubAccountLogin(1, false)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestSubmitSell(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.SubmitSell(btcInstrument, "limit", "testOrder", "", "", "", 1, 500000, 0, 0, false, false, false, false)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestEditOrderByLabel(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.EditOrderByLabel("incorrectUserLabel", btcInstrument, "",
		1, 30000, 0, false, false, false, false)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestSubmitCancel(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.SubmitCancel("incorrectID")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestSubmitCancelAll(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.SubmitCancelAll()
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestSubmitCancelAllByCurrency(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.SubmitCancelAllByCurrency(btcCurrency, "option", "")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestSubmitCancelAllByInstrument(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.SubmitCancelAllByInstrument(btcInstrument, "all")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestSubmitCancelByLabel(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.SubmitCancelByLabel("incorrectOrderLabel")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestSubmitClosePosition(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.SubmitClosePosition(btcInstrument, "limit", 35000)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestGetMargins(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.GetMargins(btcInstrument, 5, 35000)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestGetMMPConfig(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.GetMMPConfig(ethCurrency)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestGetOpenOrdersByCurrency(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.GetOpenOrdersByCurrency(btcCurrency, "option", "all")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestGetOpenOrdersByInstrument(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.GetOpenOrdersByInstrument(btcInstrument, "all")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestGetOrderHistoryByCurrency(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.GetOrderHistoryByCurrency(btcCurrency, "future", 0, 0, false, false)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestGetOrderHistoryByInstrument(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.GetOrderHistoryByInstrument(btcInstrument, 0, 0, false, false)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestGetOrderMarginsByID(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	// NOTE TO SELF: UPDATE THIS
	a, err := d.GetOrderMarginsByID([]string{""})
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestGetOrderState(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.GetOrderState("brokenid123")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestGetTriggerOrderHistory(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.GetTriggerOrderHistory(ethCurrency, "", "", 0)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestGetUserTradesByCurrency(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.GetUserTradesByCurrency(ethCurrency, "future", "5000", "5005", "asc", 0, false)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestGetUserTradesByCurrencyAndTime(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.GetUserTradesByCurrencyAndTime(ethCurrency, "future", "default", 5, false, time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestGetUserTradesByInstrument(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.GetUserTradesByInstrument(btcInstrument, "asc", 5, 10, 4, true)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestGetUserTradesByInstrumentAndTime(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.GetUserTradesByInstrumentAndTime(btcInstrument, "asc", 10, false, time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestGetUserTradesByOrder(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.GetUserTradesByOrder("wrongOrderID", "default")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestResetMMP(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.ResetMMP(btcCurrency)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestSetMMPConfig(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.SetMMPConfig(btcCurrency, 5, 5, 0, 0)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestGetSettlementHistoryByCurency(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.GetSettlementHistoryByCurency(btcCurrency, "settlement", "", 10, time.Now().Add(-time.Hour))
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}

func TestGetSettlementHistoryByInstrument(t *testing.T) {
	d.Verbose = true
	t.Parallel()
	a, err := d.GetSettlementHistoryByInstrument(btcInstrument, "settlement", "", 10, time.Now().Add(-time.Hour))
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)
}
