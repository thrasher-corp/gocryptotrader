package deribit

import (
	"errors"
	"log"
	"os"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
	btcInstrument           = "BTC-25MAR22"
	btcPerpInstrument       = "BTC-PERPETUAL"
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
	_, err := d.FetchTradablePairs(asset.Futures)
	if err != nil {
		t.Error(err)
	}
	_, err = d.FetchTradablePairs(asset.Spot)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("expected: %v, received %v", asset.ErrNotSupported, err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString(btcInstrument)
	if err != nil {
		t.Error(err)
	}
	_, err = d.UpdateTicker(cp, asset.Futures)
	if err != nil {
		t.Error(err)
	}
	_, err = d.UpdateTicker(currency.Pair{}, asset.Spot)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("expected: %v, received %v", asset.ErrNotSupported, err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString(btcInstrument)
	if err != nil {
		t.Error(err)
	}
	fmtPair, err := d.FormatExchangeCurrency(cp, asset.Futures)
	if err != nil {
		t.Error(err)
	}
	_, err = d.UpdateOrderbook(fmtPair, asset.Futures)
	if err != nil {
		t.Error(err)
	}
	_, err = d.UpdateOrderbook(fmtPair, asset.Spot)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("expected: %v, received %v", asset.ErrNotSupported, err)
	}
}

func TestFetchRecentTrades(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString(btcInstrument)
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetRecentTrades(cp, asset.Futures)
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetRecentTrades(cp, asset.Spot)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("expected: %v, received %v", asset.ErrNotSupported, err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString(btcInstrument)
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetHistoricTrades(
		cp,
		asset.Futures,
		time.Now().Add(-time.Minute*10),
		time.Now(),
	)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString(btcInstrument)
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetHistoricCandles(cp,
		asset.Futures,
		time.Now().Add(-time.Hour),
		time.Now(),
		kline.FifteenMin)
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	cp, err := currency.NewPairFromString(btcInstrument)
	if err != nil {
		t.Error(err)
	}
	_, err = d.SubmitOrder(
		&order.Submit{
			Price:     10,
			Amount:    1,
			Type:      order.Limit,
			AssetType: asset.Futures,
			Side:      order.Buy,
			Pair:      cp,
		},
	)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarkPriceHistory(t *testing.T) {
	t.Parallel()
	_, err := d.GetMarkPriceHistory(btcPerpInstrument, time.Now().Add(-24*time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestGetBookSummaryByCurrency(t *testing.T) {
	t.Parallel()
	_, err := d.GetBookSummaryByCurrency(btcCurrency, "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetBookSummaryByInstrument(t *testing.T) {
	t.Parallel()
	_, err := d.GetBookSummaryByInstrument(btcInstrument)
	if err != nil {
		t.Error(err)
	}
}

func TestGetContractSize(t *testing.T) {
	t.Parallel()
	_, err := d.GetContractSize(btcInstrument)
	if err != nil {
		t.Error(err)
	}
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
	_, err := d.GetFundingChartData(btcPerpInstrument, "8h")
	if err != nil {
		t.Error(err)
	}
}

func TestGetFundingRateValue(t *testing.T) {
	t.Parallel()
	_, err := d.GetFundingRateValue(btcPerpInstrument, time.Now().Add(-time.Hour*8), time.Now())
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetFundingRateValue(btcPerpInstrument, time.Now(), time.Now().Add(-time.Hour*8))
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
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetAccountSummary(btcCurrency, false)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelTransferByID(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.CancelTransferByID(btcCurrency, "", 23487)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTransfers(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetTransfers(btcCurrency, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelWithdrawal(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.CancelWithdrawal(btcCurrency, 123844)
	if err != nil {
		t.Error(err)
	}
}

func TestCreateDepositAddress(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.CreateDepositAddress(btcCurrency)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCurrentDepositAddress(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetCurrentDepositAddress(btcCurrency)
	if err != nil {
		t.Error(err)
	}
}

func TestGetDeposits(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetDeposits(btcCurrency, 25, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawals(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetWithdrawals(btcCurrency, 25, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitTransferToSubAccount(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.SubmitTransferToSubAccount(btcCurrency, 0.01, 13434)
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitTransferToUser(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.SubmitTransferToUser(btcCurrency, "", 0.001, 13434)
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitWithdraw(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.SubmitWithdraw(btcCurrency, "incorrectAddress", "", "", 0.001)
	if err != nil {
		t.Error(err)
	}
}

func TestChangeAPIKeyName(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.ChangeAPIKeyName(1, "TestKey123")
	if err != nil {
		t.Error(err)
	}
}

func TestChangeScopeInAPIKey(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.ChangeScopeInAPIKey(1, "account:read_write")
	if err != nil {
		t.Error(err)
	}
}

func TestChangeSubAccountName(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.ChangeSubAccountName(1, "TestingSubAccount")
	if err != nil {
		t.Error(err)
	}
}

func TestCreateAPIKey(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.CreateAPIKey("account:read_write", "TestingSubAccount", false)
	if err != nil {
		t.Error(err)
	}
}

func TestCreateSubAccount(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.CreateSubAccount()
	if err != nil {
		t.Error(err)
	}
}

func TestDisableAPIKey(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.DisableAPIKey(1)
	if err != nil {
		t.Error(err)
	}
}

func TestDisableTFAForSubAccount(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	// Use with caution will reduce the security of the account
	_, err := d.DisableTFAForSubAccount(1)
	if err != nil {
		t.Error(err)
	}
}

func TestEnableAffiliateProgram(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.EnableAffiliateProgram()
	if err != nil {
		t.Error(err)
	}
}

func TestEnableAPIKey(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.EnableAPIKey(1)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAffiliateProgramInfo(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetAffiliateProgramInfo(1)
	if err != nil {
		t.Error(err)
	}
}

func TestGetEmailLanguage(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetEmailLanguage()
	if err != nil {
		t.Error(err)
	}
}

func TestGetNewAnnouncements(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetNewAnnouncements()
	if err != nil {
		t.Error(err)
	}
}

func TestGetPosition(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetPosition(btcInstrument)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSubAccounts(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetSubAccounts(false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPositions(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetPositions(btcCurrency, "option")
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetPositions(ethCurrency, "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetTransactionLog(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetTransactionLog(btcCurrency, "trade", time.Now().Add(-24*time.Hour), time.Now(), 5, 0)
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetTransactionLog(btcCurrency, "trade", time.Now().Add(-24*time.Hour), time.Now(), 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestListAPIKeys(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.ListAPIKeys("")
	if err != nil {
		t.Error(err)
	}
}

func TestRemoveAPIKey(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.RemoveAPIKey(1)
	if err != nil {
		t.Error(err)
	}
}

func TestRemoveSubAccount(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.RemoveSubAccount(1)
	if err != nil {
		t.Error(err)
	}
}

func TestResetAPIKey(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.ResetAPIKey(1)
	if err != nil {
		t.Error(err)
	}
}

func TestSetAnnouncementAsRead(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.SetAnnouncementAsRead(1)
	if err != nil {
		t.Error(err)
	}
}

func TestSetEmailForSubAccount(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.SetEmailForSubAccount(1, "wrongemail@wrongemail.com")
	if err != nil {
		t.Error(err)
	}
}

func TestSetEmailLanguage(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.SetEmailLanguage("ja")
	if err != nil {
		t.Error(err)
	}
}

func TestSetPasswordForSubAccount(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	// Caution! This may reduce the security of the subaccount
	_, err := d.SetPasswordForSubAccount(1, "randompassword123")
	if err != nil {
		t.Error(err)
	}
}

func TestToggleNotificationsFromSubAccount(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.ToggleNotificationsFromSubAccount(1, false)
	if err != nil {
		t.Error(err)
	}
}

func TestToggleSubAccountLogin(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.ToggleSubAccountLogin(1, false)
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitSell(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.SubmitSell(btcInstrument, "limit", "testOrder", "", "", "", 1, 500000, 0, 0, false, false, false, false)
	if err != nil {
		t.Error(err)
	}
}

func TestEditOrderByLabel(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.EditOrderByLabel("incorrectUserLabel", btcInstrument, "",
		1, 30000, 0, false, false, false, false)
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitCancel(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.SubmitCancel("incorrectID")
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitCancelAll(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.SubmitCancelAll()
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitCancelAllByCurrency(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.SubmitCancelAllByCurrency(btcCurrency, "option", "")
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitCancelAllByInstrument(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.SubmitCancelAllByInstrument(btcInstrument, "all")
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitCancelByLabel(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.SubmitCancelByLabel("incorrectOrderLabel")
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitClosePosition(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.SubmitClosePosition(btcInstrument, "limit", 35000)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMargins(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetMargins(btcInstrument, 5, 35000)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMMPConfig(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetMMPConfig(ethCurrency)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOpenOrdersByCurrency(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetOpenOrdersByCurrency(btcCurrency, "option", "all")
	if err != nil {
		t.Error(err)
	}
}

func TestGetOpenOrdersByInstrument(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetOpenOrdersByInstrument(btcInstrument, "all")
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderHistoryByCurrency(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetOrderHistoryByCurrency(btcCurrency, "future", 0, 0, false, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderHistoryByInstrument(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetOrderHistoryByInstrument(btcInstrument, 0, 0, false, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderMarginsByID(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetOrderMarginsByID([]string{"id1,id2,id3"})
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetOrderMarginsByID([]string{""})
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderState(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetOrderState("brokenid123")
	if err != nil {
		t.Error(err)
	}
}

func TestGetTriggerOrderHistory(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetTriggerOrderHistory(ethCurrency, "", "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUserTradesByCurrency(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetUserTradesByCurrency(ethCurrency, "future", "5000", "5005", "asc", 0, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUserTradesByCurrencyAndTime(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetUserTradesByCurrencyAndTime(ethCurrency, "future", "default", 5, false, time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestGetUserTradesByInstrument(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetUserTradesByInstrument(btcInstrument, "asc", 5, 10, 4, true)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUserTradesByInstrumentAndTime(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetUserTradesByInstrumentAndTime(btcInstrument, "asc", 10, false, time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestGetUserTradesByOrder(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetUserTradesByOrder("wrongOrderID", "default")
	if err != nil {
		t.Error(err)
	}
}

func TestResetMMP(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.ResetMMP(btcCurrency)
	if err != nil {
		t.Error(err)
	}
}

func TestSetMMPConfig(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.SetMMPConfig(btcCurrency, 5, 5, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSettlementHistoryByCurency(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetSettlementHistoryByCurency(btcCurrency, "settlement", "", 10, time.Now().Add(-time.Hour))
	if err != nil {
		t.Error(err)
	}
}

func TestGetSettlementHistoryByInstrument(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetSettlementHistoryByInstrument(btcInstrument, "settlement", "", 10, time.Now().Add(-time.Hour))
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitEdit(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.SubmitEdit("incorrectID",
		"",
		0.001,
		100000,
		0,
		false,
		false,
		false,
		false)
	if err != nil {
		t.Error(err)
	}
}
