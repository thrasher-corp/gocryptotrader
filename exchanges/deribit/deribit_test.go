package deribit

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey    = ""
	apiSecret = ""

	canManipulateRealOrders = true
	btcPerpInstrument       = "BTC-PERPETUAL"
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
	d.Config = exchCfg
	exchCfg.API.AuthenticatedSupport = true
	exchCfg.API.AuthenticatedWebsocketSupport = true
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret
	d.Websocket = sharedtestvalues.NewTestWebsocket()
	err = d.Setup(exchCfg)
	if err != nil {
		log.Fatal("Deribit setup error", err)
	}
	os.Exit(m.Run())
}

func TestStart(t *testing.T) {
	t.Parallel()
	err := d.Start(nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrNilPointer)
	}
	var testWg sync.WaitGroup
	err = d.Start(&testWg)
	if err != nil {
		t.Fatal(err)
	}
	testWg.Wait()
}

func areTestAPIKeysSet() bool {
	return d.ValidateAPICredentials(d.GetDefaultCredentials()) == nil
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := d.FetchTradablePairs(context.Background(), asset.Futures)
	if err != nil {
		t.Error(err)
	}
	_, err = d.FetchTradablePairs(context.Background(), asset.OptionCombo)
	if err != nil {
		t.Error(err)
	}
	_, err = d.FetchTradablePairs(context.Background(), asset.Options)
	if err != nil {
		t.Error(err)
	}
	_, err = d.FetchTradablePairs(context.Background(), asset.Spot)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("expected: %v, received %v", asset.ErrNotSupported, err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString(btcPerpInstrument)
	if err != nil {
		t.Error(err)
	}
	_, err = d.UpdateTicker(context.Background(), cp, asset.Futures)
	if err != nil {
		t.Error(err)
	}
	_, err = d.UpdateTicker(context.Background(), currency.Pair{}, asset.Spot)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("expected: %v, received %v", asset.ErrNotSupported, err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString(btcPerpInstrument)
	if err != nil {
		t.Error(err)
	}
	fmtPair, err := d.FormatExchangeCurrency(cp, asset.Futures)
	if err != nil {
		t.Error(err)
	}
	_, err = d.UpdateOrderbook(context.Background(), fmtPair, asset.Futures)
	if err != nil {
		t.Error(err)
	}
	_, err = d.UpdateOrderbook(context.Background(), fmtPair, asset.Spot)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("expected: %v, received %v", asset.ErrNotSupported, err)
	}
}

func TestFetchRecentTrades(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString(btcPerpInstrument)
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetRecentTrades(context.Background(), cp, asset.Futures)
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetRecentTrades(context.Background(), cp, asset.Spot)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("expected: %v, received %v", asset.ErrNotSupported, err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString(btcPerpInstrument)
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetHistoricTrades(
		context.Background(),
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
	cp, err := currency.NewPairFromString(btcPerpInstrument)
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetHistoricCandles(context.Background(), cp,
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
	cp, err := currency.NewPairFromString(btcPerpInstrument)
	if err != nil {
		t.Error(err)
	}
	info, err := d.GetInstrumentData(context.Background(), btcPerpInstrument)
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.SubmitOrder(
		context.Background(),
		&order.Submit{
			Exchange:  d.Name,
			Price:     10,
			Amount:    info.ContractSize * 3,
			Type:      order.Limit,
			AssetType: asset.Futures,
			Side:      order.Buy,
			Pair:      cp,
		},
	)
	if err != nil && !strings.Contains(err.Error(), "not_enough_funds") {
		t.Error(err)
	}
}

func TestGetMarkPriceHistory(t *testing.T) {
	t.Parallel()
	_, err := d.GetMarkPriceHistory(context.Background(), btcPerpInstrument, time.Now().Add(-24*time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
}

var bookSummaryByCurrencyJSON = `{	"volume_usd": 0,	"volume": 0,	"quote_currency": "USD",	"price_change": -11.1896349,	"open_interest": 0,	"mid_price": null,	"mark_price": 3579.73,	"low": null,	"last": null,	"instrument_name": "BTC-22FEB19",	"high": null,	"estimated_delivery_price": 3579.73,	"creation_timestamp": 1550230036440,	"bid_price": null,	"base_currency": "BTC",	"ask_price": null}`

func TestGetBookSummaryByCurrency(t *testing.T) {
	t.Parallel()
	var response BookSummaryData
	if err := json.Unmarshal([]byte(bookSummaryByCurrencyJSON), &response); err != nil {
		t.Error(err)
	}
	_, err := d.GetBookSummaryByCurrency(context.Background(), currencyBTC, "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetBookSummaryByInstrument(t *testing.T) {
	t.Parallel()
	_, err := d.GetBookSummaryByInstrument(context.Background(), btcPerpInstrument)
	if err != nil {
		t.Error(err)
	}
}

func TestGetContractSize(t *testing.T) {
	t.Parallel()
	_, err := d.GetContractSize(context.Background(), btcPerpInstrument)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	_, err := d.GetCurrencies(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetDeliveryPrices(t *testing.T) {
	t.Parallel()
	_, err := d.GetDeliveryPrices(context.Background(), "btc_usd", 0, 5)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFundingChartData(t *testing.T) {
	t.Parallel()
	_, err := d.GetFundingChartData(context.Background(), btcPerpInstrument, "8h")
	if err != nil {
		t.Error(err)
	}
}

func TestGetFundingRateValue(t *testing.T) {
	t.Parallel()
	_, err := d.GetFundingRateValue(context.Background(), btcPerpInstrument, time.Now().Add(-time.Hour*8), time.Now())
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetFundingRateValue(context.Background(), btcPerpInstrument, time.Now(), time.Now().Add(-time.Hour*8))
	if !errors.Is(err, errStartTimeCannotBeAfterEndTime) {
		t.Errorf("expected: %v, received %v", errStartTimeCannotBeAfterEndTime, err)
	}
}

func TestGetHistoricalVolatility(t *testing.T) {
	t.Parallel()
	_, err := d.GetHistoricalVolatility(context.Background(), currencyBTC)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCurrencyIndexPrice(t *testing.T) {
	t.Parallel()
	if _, err := d.GetCurrencyIndexPrice(context.Background(), currencyBTC); err != nil {
		t.Error(err)
	}
}

func TestGetIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := d.GetIndexPrice(context.Background(), "ada_usd")
	if err != nil {
		t.Error(err)
	}
}

func TestGetIndexPriceNames(t *testing.T) {
	t.Parallel()
	_, err := d.GetIndexPriceNames(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetInstrumentData(t *testing.T) {
	t.Parallel()
	_, err := d.GetInstrumentData(context.Background(), btcPerpInstrument)
	if err != nil {
		t.Error(err)
	}
}

func TestGetInstrumentsData(t *testing.T) {
	t.Parallel()
	_, err := d.GetInstrumentsData(context.Background(), currencyBTC, "", false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetLastSettlementsByCurrency(t *testing.T) {
	t.Parallel()
	_, err := d.GetLastSettlementsByCurrency(context.Background(), currencyBTC, "", "", 0, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetLastSettlementsByCurrency(context.Background(), currencyBTC, "delivery", "5", 0, time.Now().Add(-time.Hour))
	if err != nil {
		t.Error(err)
	}
}

func TestGetLastSettlementsByInstrument(t *testing.T) {
	t.Parallel()
	_, err := d.GetLastSettlementsByInstrument(context.Background(), btcPerpInstrument, "", "", 0, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetLastSettlementsByInstrument(context.Background(), btcPerpInstrument, "settlement", "5", 0, time.Now().Add(-2*time.Hour))
	if err != nil {
		t.Error(err)
	}
}

func TestGetLastTradesByCurrency(t *testing.T) {
	t.Parallel()
	_, err := d.GetLastTradesByCurrency(context.Background(), currencyBTC, "", "", "", "", 0, false)
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetLastTradesByCurrency(context.Background(), currencyBTC, "option", "36798", "36799", "asc", 0, true)
	if err != nil {
		t.Error(err)
	}
}

func TestGetLastTradesByCurrencyAndTime(t *testing.T) {
	t.Parallel()
	_, err := d.GetLastTradesByCurrencyAndTime(context.Background(), currencyBTC, "", "", 0, false,
		time.Now().Add(-8*time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetLastTradesByCurrencyAndTime(context.Background(), currencyBTC, "option", "asc", 25, false,
		time.Now().Add(-8*time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestGetLastTradesByInstrument(t *testing.T) {
	t.Parallel()
	_, err := d.GetLastTradesByInstrument(context.Background(), btcPerpInstrument, "", "", "", 0, false)
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetLastTradesByInstrument(context.Background(), btcPerpInstrument, "30500", "31500", "desc", 0, true)
	if err != nil {
		t.Error(err)
	}
}

func TestGetLastTradesByInstrumentAndTime(t *testing.T) {
	t.Parallel()
	_, err := d.GetLastTradesByInstrumentAndTime(context.Background(), btcPerpInstrument, "", 0, false,
		time.Now().Add(-8*time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetLastTradesByInstrumentAndTime(context.Background(), btcPerpInstrument, "asc", 0, false,
		time.Now().Add(-8*time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderbookData(t *testing.T) {
	t.Parallel()
	_, err := d.GetOrderbookData(context.Background(), btcPerpInstrument, 0)
	if err != nil {
		t.Error(err)
	}
}

var orderbookJSON = `{	"timestamp":1550757626706,	"stats":{"volume":93.35589552,"price_change": 0.6913,		"low":3940.75,		"high":3976.25	},	"state":"open",	"settlement_price":3925.85,	"open_interest":45.27600333464605,	"min_price":3932.22,	"max_price":3971.74,	"mark_price":3931.97,	"last_price":3955.75,	"instrument_name":"BTC-PERPETUAL",	"index_price":3910.46,	"funding_8h":0.00455263,	"current_funding":0.00500063,	"change_id":474988,	"bids":[		[			3955.75,			30.0		],		[			3940.75,			102020.0		],		[			3423.0,			42840.0		]	],	"best_bid_price":3955.75,	"best_bid_amount":30.0,	"best_ask_price":0.0,	"best_ask_amount":0.0,	"asks":[	]}`

func TestGetOrderbookByInstrumentID(t *testing.T) {
	t.Parallel()
	var response Orderbook
	err := json.Unmarshal([]byte(orderbookJSON), &response)
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetOrderbookByInstrumentID(context.Background(), 12, 50)
	if err != nil && !strings.Contains(err.Error(), "not_found") {
		t.Error(err)
	}
}

var rfqJSON = `{"traded_volume": 0,	"amount": 10,"side": "buy",	"last_rfq_tstamp": 1634816611595,"instrument_name": "BTC-PERPETUAL"}`

func TestGetRFQ(t *testing.T) {
	t.Parallel()
	var response RFQ
	err := json.Unmarshal([]byte(rfqJSON), &response)
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetRFQ(context.Background(), currencyBTC, "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetTradeVolumes(t *testing.T) {
	t.Parallel()
	_, err := d.GetTradeVolumes(context.Background(), false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTradingViewChartData(t *testing.T) {
	t.Parallel()
	_, err := d.GetTradingViewChartData(context.Background(), btcPerpInstrument, "60", time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestGetVolatilityIndexData(t *testing.T) {
	t.Parallel()
	_, err := d.GetVolatilityIndexData(context.Background(), currencyBTC, "60", time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestGetPublicTicker(t *testing.T) {
	t.Parallel()
	_, err := d.GetPublicTicker(context.Background(), btcPerpInstrument)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountSummary(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetAccountSummary(context.Background(), currencyBTC, false)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelTransferByID(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.CancelTransferByID(context.Background(), currencyBTC, "", 23487)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTransfers(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetTransfers(context.Background(), currencyBTC, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelWithdrawal(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.CancelWithdrawal(context.Background(), currencyBTC, 123844)
	if err != nil && !strings.Contains(err.Error(), "withdrawal with given id and currency not found") {
		t.Error(err)
	}
}

func TestCreateDepositAddress(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.CreateDepositAddress(context.Background(), currencySOL)
	if err != nil && !strings.Contains(err.Error(), "max_addr_count_exceeded") {
		t.Error(err)
	}
}

func TestGetCurrentDepositAddress(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetCurrentDepositAddress(context.Background(), currencyETH)
	if err != nil && !strings.Contains(err.Error(), "max_addr_count_exceeded") {
		t.Error(err)
	}
}

func TestGetDeposits(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetDeposits(context.Background(), currencyBTC, 25, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawals(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetWithdrawals(context.Background(), currencyBTC, 25, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitTransferToSubAccount(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.SubmitTransferToSubAccount(context.Background(), currencyBTC, 0.01, 13434)
	if err != nil && !strings.Contains(err.Error(), "transfer_not_allowed") {
		t.Error(err)
	}
}

func TestSubmitTransferToUser(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.SubmitTransferToUser(context.Background(), currencyBTC, "", 0.001, 13434)
	if err != nil && !strings.Contains(err.Error(), "transfer_not_allowed") {
		t.Error(err)
	}
}

func TestSubmitWithdraw(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.SubmitWithdraw(context.Background(), currencyBTC, core.BitcoinDonationAddress, "", 0.001)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAnnouncements(t *testing.T) {
	t.Parallel()
	_, err := d.GetAnnouncements(context.Background(), time.Now(), 5)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPublicPortfolioMargins(t *testing.T) {
	t.Parallel()
	info, err := d.GetInstrumentData(context.Background(), "BTC-PERPETUAL")
	if err != nil {
		t.Skip(err)
	}
	_, err = d.GetPublicPortfolioMargins(context.Background(), currencyBTC, map[string]float64{
		"BTC-PERPETUAL": info.ContractSize * 2,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccessLog(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	_, err := d.GetAccessLog(context.Background(), 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestChangeAPIKeyName(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.ChangeAPIKeyName(context.Background(), 1, "TestKey123")
	if err != nil {
		t.Error(err)
	}
}

func TestChangeScopeInAPIKey(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.ChangeScopeInAPIKey(context.Background(), 1, "account:read_write")
	if err != nil {
		t.Error(err)
	}
}

func TestChangeSubAccountName(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.ChangeSubAccountName(context.Background(), 1, "new_sub")
	if err != nil && !strings.Contains(err.Error(), "unauthorized") {
		t.Error(err)
	}
}

func TestCreateAPIKey(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.CreateAPIKey(context.Background(), "account:read_write", "new_sub", false)
	if err != nil {
		t.Error(err)
	}
}

func TestCreateSubAccount(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.CreateSubAccount(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestDisableAPIKey(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.DisableAPIKey(context.Background(), 1)
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
	_, err := d.DisableTFAForSubAccount(context.Background(), 1)
	if err != nil && !strings.Contains(err.Error(), "Method not found") { // this functionality is removed by now.
		t.Error(err)
	}
}

func TestEnableAffiliateProgram(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := d.EnableAffiliateProgram(context.Background())
	if err != nil && !strings.Contains(err.Error(), "not_allowed_to_enable_affiliate_program") {
		t.Error(err)
	}
}

func TestEnableAPIKey(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	t.Parallel()
	_, err := d.EnableAPIKey(context.Background(), 1)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAffiliateProgramInfo(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetAffiliateProgramInfo(context.Background(), 1)
	if err != nil {
		t.Error(err)
	}
}

func TestGetEmailLanguage(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetEmailLanguage(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetNewAnnouncements(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetNewAnnouncements(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetPricatePortfolioMargins(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	if _, err := d.GetPricatePortfolioMargins(context.Background(), currencyBTC, false, nil); err != nil {
		t.Error(err)
	}
}

func TestGetPosition(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetPosition(context.Background(), btcPerpInstrument)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSubAccounts(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetSubAccounts(context.Background(), false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSubAccountDetails(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetSubAccountDetails(context.Background(), currencyBTC, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPositions(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetPositions(context.Background(), currencyBTC, "option")
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetPositions(context.Background(), currencyETH, "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetTransactionLog(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.GetTransactionLog(context.Background(), currencyBTC, "trade", time.Now().Add(-24*time.Hour), time.Now(), 5, 0)
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetTransactionLog(context.Background(), currencyBTC, "trade", time.Now().Add(-24*time.Hour), time.Now(), 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUserLocks(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	_, err := d.GetUserLocks(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestListAPIKeys(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.ListAPIKeys(context.Background(), "")
	if err != nil {
		t.Error(err)
	}
}

func TestRemoveAPIKey(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := d.RemoveAPIKey(context.Background(), 1)
	if err != nil {
		t.Error(err)
	}
}

func TestRemoveSubAccount(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.RemoveSubAccount(context.Background(), 1)
	if err != nil && !strings.Contains(err.Error(), "unauthorized") {
		t.Error(err)
	}
}

func TestResetAPIKey(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	t.Parallel()
	_, err := d.ResetAPIKey(context.Background(), 1)
	if err != nil {
		t.Error(err)
	}
}

func TestSetAnnouncementAsRead(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := d.SetAnnouncementAsRead(context.Background(), 1)
	if err != nil {
		t.Error(err)
	}
}

func TestSetEmailForSubAccount(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := d.SetEmailForSubAccount(context.Background(), 1, "wrongemail@wrongemail.com")
	if err != nil && !strings.Contains(err.Error(), "could not link email (wrongemail@wrongemail.com) to subaccount 1") {
		t.Error(err)
	}
}

func TestSetEmailLanguage(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := d.SetEmailLanguage(context.Background(), "en")
	if err != nil {
		t.Error(err)
	}
}

func TestSetPasswordForSubAccount(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	// Caution! This may reduce the security of the subaccount
	_, err := d.SetPasswordForSubAccount(context.Background(), 1, "randompassword123")
	if err != nil {
		t.Error(err)
	}
}

func TestToggleNotificationsFromSubAccount(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := d.ToggleNotificationsFromSubAccount(context.Background(), 1, false)
	if err != nil && !strings.Contains(err.Error(), "unauthorized") {
		t.Error(err)
	}
}

func TestTogglePortfolioMargining(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	subaccount, err := d.GetSubAccountDetails(context.Background(), currencyBTC, false)
	if err != nil {
		t.Skip(err)
	}
	if len(subaccount) == 0 {
		t.SkipNow()
	}
	_, err = d.TogglePortfolioMargining(context.Background(), int64(subaccount[0].UID), false, false)
	if err != nil && !strings.Contains(err.Error(), "account is already on SM") {
		t.Error(err)
	}
}

func TestToggleSubAccountLogin(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := d.ToggleSubAccountLogin(context.Background(), 1, false)
	if err != nil && !strings.Contains(err.Error(), "unauthorized") {
		t.Error(err)
	}
}

func TestSubmitSell(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	info, err := d.GetInstrumentData(context.Background(), btcPerpInstrument)
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.SubmitSell(context.Background(), btcPerpInstrument, "limit", "testOrder", "", "", "", info.ContractSize*3, 500000, 0, 0, false, false, false, false)
	if err != nil && !strings.Contains(err.Error(), "not_enough_funds") {
		t.Error(err)
	}
}

func TestEditOrderByLabel(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := d.EditOrderByLabel(context.Background(), "incorrectUserLabel", btcPerpInstrument, "",
		1, 30000, 0, false, false, false, false)
	if err != nil && !strings.Contains(err.Error(), "order_not_found") {
		t.Error(err)
	}
}

func TestSubmitCancel(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := d.SubmitCancel(context.Background(), "incorrectID")
	if err != nil && !strings.Contains(err.Error(), "order_not_found") {
		t.Error(err)
	}
}

func TestSubmitCancelAll(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := d.SubmitCancelAll(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitCancelAllByCurrency(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := d.SubmitCancelAllByCurrency(context.Background(), currencyBTC, "option", "")
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitCancelAllByInstrument(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := d.SubmitCancelAllByInstrument(context.Background(), btcPerpInstrument, "all")
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitCancelByLabel(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := d.SubmitCancelByLabel(context.Background(), "incorrectOrderLabel", "")
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitClosePosition(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := d.SubmitClosePosition(context.Background(), btcPerpInstrument, "limit", 35000)
	if err != nil && !strings.Contains(err.Error(), "already_closed") {
		t.Error(err)
	}
}

func TestGetMargins(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := d.GetMargins(context.Background(), btcPerpInstrument, 5, 35000)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMMPConfig(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := d.GetMMPConfig(context.Background(), currencyETH)
	if err != nil && !strings.Contains(err.Error(), "MMP disabled") {
		t.Error(err)
	}
}

func TestGetOpenOrdersByCurrency(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := d.GetOpenOrdersByCurrency(context.Background(), currencyBTC, "option", "all")
	if err != nil {
		t.Error(err)
	}
}

func TestGetOpenOrdersByInstrument(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := d.GetOpenOrdersByInstrument(context.Background(), btcPerpInstrument, "all")
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderHistoryByCurrency(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := d.GetOrderHistoryByCurrency(context.Background(), currencyBTC, "future", 0, 0, false, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderHistoryByInstrument(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := d.GetOrderHistoryByInstrument(context.Background(), btcPerpInstrument, 0, 0, false, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderMarginsByID(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := d.GetOrderMarginsByID(context.Background(), []string{"ETH-349280", "ETH-349279", "ETH-349278"})
	if err != nil && strings.Contains(err.Error(), "value must be a list") {
		t.Skip(err)
	} else if err != nil {
		t.Error(err)
	}
}

func TestGetOrderState(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := d.GetOrderState(context.Background(), "brokenid123")
	if err != nil && !strings.Contains(err.Error(), "order_not_found") {
		t.Error(err)
	}
}

func TestGetTriggerOrderHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := d.GetTriggerOrderHistory(context.Background(), currencyETH, "", "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUserTradesByCurrency(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := d.GetUserTradesByCurrency(context.Background(), currencyETH, "future", "", "", "asc", 0, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUserTradesByCurrencyAndTime(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := d.GetUserTradesByCurrencyAndTime(context.Background(), currencyETH, "future", "default", 5, false, "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetUserTradesByInstrument(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := d.GetUserTradesByInstrument(context.Background(), btcPerpInstrument, "asc", 5, 10, 4, true)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUserTradesByInstrumentAndTime(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := d.GetUserTradesByInstrumentAndTime(context.Background(), btcPerpInstrument, "asc", 10, false, time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestGetUserTradesByOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := d.GetUserTradesByOrder(context.Background(), "wrongOrderID", "default")
	if err != nil && !strings.Contains(err.Error(), "order_not_found") {
		t.Error(err)
	}
}

func TestResetMMP(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := d.ResetMMP(context.Background(), currencyBTC)
	if err != nil && !strings.Contains(err.Error(), "MMP disabled") {
		t.Error(err)
	}
}

func TestSendRFQ(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := d.SendRFQ(context.Background(), "BTC-PERPETUAL", 1000, order.Buy)
	if err != nil && !strings.Contains(err.Error(), "not_enough_funds") {
		t.Error(err)
	}
}

func TestSetMMPConfig(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := d.SetMMPConfig(context.Background(), currencyBTC, 5, 5, 0, 0)
	if err != nil && !strings.Contains(err.Error(), "MMP disabled") {
		t.Error(err)
	}
}

func TestGetSettlementHistoryByCurency(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := d.GetSettlementHistoryByCurency(context.Background(), currencyBTC, "settlement", "", 10, time.Now().Add(-time.Hour))
	if err != nil {
		t.Error(err)
	}
}

func TestGetSettlementHistoryByInstrument(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := d.GetSettlementHistoryByInstrument(context.Background(), btcPerpInstrument, "settlement", "", 10, time.Now().Add(-time.Hour))
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitEdit(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := d.SubmitEdit(context.Background(), "incorrectID", "", 0.001, 100000, 123, false, false, false, false)
	if err != nil && !strings.Contains(err.Error(), "order_not_found") {
		t.Error(err)
	}
}

// Combo Books Endpoints

func TestGetComboIDS(t *testing.T) {
	t.Parallel()
	_, err := d.GetComboIDS(context.Background(), currencyBTC, "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetComboDetails(t *testing.T) {
	t.Parallel()
	combos, err := d.GetComboIDS(context.Background(), currencyBTC, "")
	if err != nil {
		t.Skip(err)
	}
	if len(combos) == 0 {
		t.Skip("no combo instance found for currency BTC")
	}
	_, err = d.GetComboDetails(context.Background(), combos[0])
	if err != nil {
		t.Error(err)
	}
}

func TestGetCombos(t *testing.T) {
	t.Parallel()
	_, err := d.GetCombos(context.Background(), currencyBTC)
	if err != nil {
		t.Error(err)
	}
}

func TestCreateCombo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	_, err := d.CreateCombo(context.Background(), []ComboParam{})
	if err != nil && !errors.Is(errNoArgumentPassed, err) {
		t.Errorf("expecting %v, but found %v", errNoArgumentPassed, err)
	}
	instruments, err := d.FetchTradablePairs(context.Background(), asset.Futures)
	if err != nil {
		t.Skip(err)
	}
	if len(instruments) < 2 {
		t.Skip("no enough instrument found")
	}
	_, err = d.CreateCombo(context.Background(), []ComboParam{
		{
			InstrumentName: instruments[0],
			Direction:      "sell",
		},
		{
			InstrumentName: instruments[1],
			Direction:      "sell",
			Amount:         1200,
		},
	})
	if err != nil && !errors.Is(errInvalidAmount, err) {
		t.Errorf("expecting %v, but found %v", errInvalidAmount, err)
	}
	_, err = d.CreateCombo(context.Background(), []ComboParam{
		{
			InstrumentName: instruments[0],
			Amount:         123,
		},
		{
			InstrumentName: instruments[1],
			Direction:      "sell",
			Amount:         1200,
		},
	})
	if err != nil && !strings.Contains(err.Error(), "invalid direction") {
		t.Errorf("expecting error message 'invalid direction', but found %v", err)
	}
	_, err = d.CreateCombo(context.Background(), []ComboParam{
		{
			InstrumentName: instruments[0],
			Direction:      "buy",
			Amount:         123,
		},
		{
			InstrumentName: instruments[1],
			Direction:      "buy",
			Amount:         1200,
		},
	})
	if err != nil && !strings.Contains(err.Error(), "not_enough_funds") {
		t.Error(err)
	}
}

func TestVerifyBlockTrade(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	info, err := d.GetInstrumentData(context.Background(), "BTC-PERPETUAL")
	if err != nil {
		t.Skip(err)
	}
	_, err = d.VerifyBlockTrade(context.Background(), time.Now(), "sdjkafdad", "maker", "", []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: "BTC-PERPETUAL",
			Direction:      "buy",
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	})
	if err != nil && !strings.Contains(err.Error(), "not_enough_funds") {
		t.Error(err)
	}
}

var blockTradeResponseJSON = `[	{	  "trade_seq":30289730,	  "trade_id":"48079573",	  "timestamp":1590485535978,	  "tick_direction":0,	  "state":"filled",	  "self_trade":false,	  "reduce_only":false,	  "price":8900.0,	  "post_only":false,	  "order_type":"limit",	  "order_id":"4009043192",	  "matching_id":"None",	  "mark_price":8895.19,	  "liquidity":"M",	  "instrument_name":"BTC-PERPETUAL",	  "index_price":8900.45,	  "fee_currency":"BTC",	  "fee":-0.00561798,	  "direction":"sell",	  "block_trade_id":"6165",	  "amount":200000.0	},	{	  "underlying_price":8902.86,	  "trade_seq":1,	  "trade_id":"48079574",	  "timestamp":1590485535979,	  "tick_direction":1,	  "state":"filled",	  "self_trade":false,	  "reduce_only":false,	  "price":0.0133,	  "post_only":false,	  "order_type":"limit",	  "order_id":"4009043194",	  "matching_id":"None",	  "mark_price":0.01831619,	  "liquidity":"M",	  "iv":62.44,	  "instrument_name":"BTC-28MAY20-9000-C",	  "index_price":8900.45,	  "fee_currency":"BTC",	  "fee":0.002,	  "direction":"sell",	  "block_trade_id":"6165",	  "amount":5.0	}]`

func TestExecuteBlockTrade(t *testing.T) {
	t.Parallel()
	var response []BlockTradeResponse
	err := json.Unmarshal([]byte(blockTradeResponseJSON), &response)
	if err != nil {
		t.Error(err)
	}
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	info, err := d.GetInstrumentData(context.Background(), "BTC-PERPETUAL")
	if err != nil {
		t.Skip(err)
	}
	_, err = d.ExecuteBlockTrade(context.Background(), time.Now(), "sdjkafdad", "maker", "", []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: "BTC-PERPETUAL",
			Direction:      "buy",
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	})
	if err != nil && !strings.Contains(err.Error(), "not_enough_funds") {
		t.Error(err)
	}
}

func TestGetUserBlocTrade(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	_, err := d.GetUserBlocTrade(context.Background(), "12345567")
	if err != nil && !strings.Contains(err.Error(), "block_trade_not_found") {
		t.Error(err)
	}
}

var blockTradeResponsesJSON = `[	{	  "trade_seq": 4,	  "trade_id": "92462",	  "timestamp": 1565093070164,	  "tick_direction": 2,	  "state": "filled",	  "self_trade": false,	  "price": 0.0151,	  "order_type": "limit",	  "order_id": "343121",	  "matching_id": null,	  "liquidity": "M",	  "iv": 72.38,	  "instrument_name": "BTC-9AUG19-11500-P",	  "index_price": 11758.65,	  "fee_currency": "BTC",	  "fee": 0,	  "direction": "sell",	  "block_trade_id": "66",	  "amount": 2.3	},	{	  "trade_seq": 41,	  "trade_id": "92460",	  "timestamp": 1565093070164,	  "tick_direction": 2,	  "state": "filled",	  "self_trade": false,	  "price": 11753,	  "order_type": "limit",	  "order_id": "343117",	  "matching_id": null,	  "liquidity": "M",	  "instrument_name": "BTC-9AUG19",	  "index_price": 11758.65,	  "fee_currency": "BTC",	  "fee": 0,	  "direction": "sell",	  "block_trade_id": "66",	  "amount": 50	}]`

func TestGetLastBlockTradesbyCurrency(t *testing.T) {
	t.Parallel()
	var resp []BlockTradeData
	err := json.Unmarshal([]byte(blockTradeResponsesJSON), &resp)
	if err != nil {
		t.Error(err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	_, err = d.GetLastBlockTradesbyCurrency(context.Background(), "SOL", "", "", 5)
	if err != nil {
		t.Error(err)
	}
}

func TestMovePositions(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	info, err := d.GetInstrumentData(context.Background(), "BTC-PERPETUAL")
	if err != nil {
		t.Skip(err)
	}
	_, err = d.MovePositions(context.Background(), currencyBTC, 123, 345, []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: "BTC-PERPETUAL",
			Direction:      "buy",
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	})
	if err != nil && !strings.Contains(err.Error(), "must be one of the subaccounts") {
		t.Error(err)
	}
}
