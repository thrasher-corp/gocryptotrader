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
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey    = ""
	apiSecret = ""

	canManipulateRealOrders = false
	btcPerpInstrument       = "BTC-PERPETUAL"

	authenticationSkipMessage         = "missing API credentials"
	endpointAuthorizationToManipulate = "endpoint requires API credentials and 'canManipulateRealOrders' to be enabled"
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
	d.Websocket = sharedtestvalues.NewTestWebsocket()
	d.Base.Config = exchCfg
	err = d.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}
	d.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	d.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
	setupWs()
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
	if err != nil && !errors.Is(err, asset.ErrNotSupported) {
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
	_, err = d.GetHistoricCandles(context.Background(), cp, asset.Futures, kline.FifteenMin, time.Now().Add(-time.Minute*5), time.Now())
	if err != nil {
		t.Error(err)
	}
}
func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString(btcPerpInstrument)
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetHistoricCandlesExtended(context.Background(), cp,
		asset.Futures,
		kline.FifteenMin,
		time.Now().Add(-time.Hour*10),
		time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
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
	pairs, err := d.FetchTradablePairs(context.Background(), asset.Futures)
	if err != nil || len(pairs) == 0 {
		t.Skip("tradable pairs not found")
	}
	_, err = d.GetMarkPriceHistory(context.Background(), pairs[0].String(), time.Now().Add(-5*time.Minute), time.Now())
	if err != nil {
		t.Error(err)
	}
	if _, err := d.WSRetrieveMarkPriceHistory(pairs[0].String(), time.Now().Add(-4*time.Hour), time.Now()); err != nil {
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
	if _, err = d.WSRetrieveBookBySummary(currencySOL, ""); err != nil {
		t.Error(err)
	}
}

func TestGetBookSummaryByInstrument(t *testing.T) {
	t.Parallel()
	_, err := d.GetBookSummaryByInstrument(context.Background(), btcPerpInstrument)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveBookSummaryByInstrument(btcPerpInstrument); err != nil {
		t.Error(err)
	}
}

func TestGetContractSize(t *testing.T) {
	t.Parallel()
	_, err := d.GetContractSize(context.Background(), btcPerpInstrument)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveContractSize(btcPerpInstrument); err != nil {
		t.Error(err)
	}
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	_, err := d.GetCurrencies(context.Background())
	if err != nil {
		t.Error(err)
	}
	if _, err := d.WSRetrieveCurrencies(); err != nil {
		t.Error(err)
	}
}

func TestGetDeliveryPrices(t *testing.T) {
	t.Parallel()
	_, err := d.GetDeliveryPrices(context.Background(), "btc_usd", 0, 5)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveDeliveryPrices("btc_usd", 0, 5); err != nil {
		t.Error(err)
	}
}

func TestGetFundingChartData(t *testing.T) {
	t.Parallel()
	_, err := d.GetFundingChartData(context.Background(), btcPerpInstrument, "8h")
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveFundingChartData(btcPerpInstrument, "8h"); err != nil {
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
	if err != nil && !errors.Is(err, common.ErrStartAfterEnd) {
		t.Errorf("expected: %v, received %v", errStartTimeCannotBeAfterEndTime, err)
	}
	_, err = d.WSRetrieveFundingRateValue(btcPerpInstrument, time.Now(), time.Now().Add(-time.Hour*8))
	if err != nil && !errors.Is(err, common.ErrStartAfterEnd) {
		t.Errorf("expected: %v, received %v", errStartTimeCannotBeAfterEndTime, err)
	}
	if _, err = d.WSRetrieveFundingRateValue(btcPerpInstrument, time.Now().Add(-time.Hour*8), time.Now()); err != nil {
		t.Error(err)
	}
}

func TestGetHistoricalVolatility(t *testing.T) {
	t.Parallel()
	_, err := d.GetHistoricalVolatility(context.Background(), currencyBTC)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveHistoricalVolatility(currencySOL); err != nil {
		t.Error(err)
	}
}

func TestGetCurrencyIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := d.GetCurrencyIndexPrice(context.Background(), currencyBTC)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveCurrencyIndexPrice(currencyBTC); err != nil {
		t.Error(err)
	}
}

func TestGetIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := d.GetIndexPrice(context.Background(), "ada_usd")
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveIndexPrice("ada_usd"); err != nil {
		t.Error(err)
	}
}

func TestGetIndexPriceNames(t *testing.T) {
	t.Parallel()
	_, err := d.GetIndexPriceNames(context.Background())
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveIndexPriceNames(); err != nil {
		t.Error(err)
	}
}

func TestGetInstrumentData(t *testing.T) {
	t.Parallel()
	_, err := d.GetInstrumentData(context.Background(), btcPerpInstrument)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveInstrumentData(btcPerpInstrument); err != nil {
		t.Error(err)
	}
}

func TestGetInstrumentsData(t *testing.T) {
	t.Parallel()
	_, err := d.GetInstrumentsData(context.Background(), currencyBTC, "", false)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveInstrumentsData(currencyBTC, "", false); err != nil {
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
	_, err = d.WSRetrieveLastSettlementsByCurrency(currencyBTC, "", "", 0, time.Now().Add(-time.Hour))
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveLastSettlementsByCurrency(currencyBTC, "delivery", "5", 0, time.Now().Add(-time.Hour)); err != nil {
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
	_, err = d.WSRetrieveLastSettlementsByInstrument(btcPerpInstrument, "", "", 0, time.Time{})
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveLastSettlementsByInstrument(btcPerpInstrument, "settlement", "5", 0, time.Now().Add(-2*time.Hour)); err != nil {
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
	_, err = d.WSRetrieveLastTradesByCurrency(currencyBTC, "", "", "", "", 0, false)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveLastTradesByCurrency(currencyBTC, "option", "36798", "36799", "asc", 0, true); err != nil {
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
	_, err = d.WSRetrieveLastTradesByCurrencyAndTime(currencyBTC, "", "", 0, false, time.Now().Add(-8*time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
	if _, err := d.WSRetrieveLastTradesByCurrencyAndTime(currencyBTC, "option", "asc", 25, false, time.Now().Add(-8*time.Hour), time.Now()); err != nil {
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
	_, err = d.WSRetrieveLastTradesByInstrument(btcPerpInstrument, "", "", "", 0, false)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveLastTradesByInstrument(btcPerpInstrument, "30500", "31500", "desc", 0, true); err != nil {
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
	_, err = d.WSRetrieveLastTradesByInstrumentAndTime(btcPerpInstrument, "", 0, false, time.Now().Add(-8*time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveLastTradesByInstrumentAndTime(btcPerpInstrument, "asc", 0, false, time.Now().Add(-8*time.Hour), time.Now()); err != nil {
		t.Error(err)
	}
}

func TestGetOrderbookData(t *testing.T) {
	t.Parallel()
	_, err := d.GetOrderbookData(context.Background(), btcPerpInstrument, 0)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveOrderbookData(btcPerpInstrument, 0); err != nil {
		t.Error(err)
	}
}

var orderbookJSON = `{"underlying_price":16582.3,"underlying_index":"index_price","timestamp":1668792897465,"stats":{"volume":null,"price_change":null,"low":null,"high":null},"state":"inactive","settlement_price":0.18292605,"open_interest":0.0,"min_price":0.1678,"max_price":0.1977,"mark_price":0.1827,"mark_iv":0.0,"last_price":0.18,"interest_rate":0.0,"instrument_name":"BTC-BOX-30DEC22-17000_20000","index_price":16582.3,"implied_bid":0.1775,"implied_ask":0.193,"greeks":{"vega":0.00192,"theta":-0.00154,"rho":-3.4168,"gamma":0.0,"delta":0.00013},"estimated_delivery_price":16582.3,"combo_state":"inactive","change_id":51602345369,"bids":[],"bid_iv":0.0,"best_bid_price":0.0,"best_bid_amount":0.0,"best_ask_price":0.0,"best_ask_amount":0.0,"asks":[],"ask_iv":0.0}`

func TestGetOrderbookByInstrumentID(t *testing.T) {
	t.Parallel()
	var response Orderbook
	err := json.Unmarshal([]byte(orderbookJSON), &response)
	if err != nil {
		t.Error(err)
	}
	combos, err := d.WSRetrieveComboIDS(currencyBTC, "")
	if err != nil {
		t.Skip(err)
	}
	if len(combos) == 0 {
		t.Skip("no combo instance found for currency BTC")
	}
	comboD, err := d.WSRetrieveComboDetails(combos[0])
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetOrderbookByInstrumentID(context.Background(), comboD.InstrumentID, 50)
	if err != nil && !strings.Contains(err.Error(), "not_found") {
		t.Error(err)
	}
	if _, err := d.WSRetrieveOrderbookByInstrumentID(comboD.InstrumentID, 50); err != nil && !strings.Contains(err.Error(), "not_found") {
		t.Error(err)
	}
}

var rfqJSON = `{"traded_volume": 0,	"amount": 10,"side": "buy",	"last_rfq_tstamp": 1634816611595,"instrument_name": "BTC-PERPETUAL"}`

func TestGetRequestForQuote(t *testing.T) {
	t.Parallel()
	var response RequestForQuote
	err := json.Unmarshal([]byte(rfqJSON), &response)
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetRequestForQuote(context.Background(), currencyBTC, d.GetAssetKind(asset.Futures))
	if err != nil {
		t.Error(err)
	}
	if _, err := d.WSRetrieveRequestForQuote(currencyBTC, d.GetAssetKind(asset.Futures)); err != nil {
		t.Error(err)
	}
}

func TestGetTradeVolumes(t *testing.T) {
	t.Parallel()
	_, err := d.GetTradeVolumes(context.Background(), false)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveTradeVolumes(false); err != nil {
		t.Error(err)
	}
}

func TestGetTradingViewChartData(t *testing.T) {
	t.Parallel()
	_, err := d.GetTradingViewChartData(context.Background(), btcPerpInstrument, "60", time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrievesTradingViewChartData(btcPerpInstrument, "60", time.Now().Add(-time.Hour), time.Now()); err != nil {
		t.Error(err)
	}
}

func TestGetVolatilityIndexData(t *testing.T) {
	t.Parallel()
	_, err := d.GetVolatilityIndexData(context.Background(), currencyBTC, "60", time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveVolatilityIndexData(currencyBTC, "60", time.Now().Add(-time.Hour), time.Now()); err != nil {
		t.Error(err)
	}
}

func TestGetPublicTicker(t *testing.T) {
	t.Parallel()
	_, err := d.GetPublicTicker(context.Background(), btcPerpInstrument)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrievePublicTicker(btcPerpInstrument); err != nil {
		t.Error(err)
	}
}

func TestGetAccountSummary(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetAccountSummary(context.Background(), currencyBTC, false)
	if err != nil {
		t.Error(err)
	}
	if _, err := d.WSRetrieveAccountSummary(currencyBTC, false); err != nil {
		t.Error(err)
	}
}

func TestCancelTransferByID(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	_, err := d.CancelTransferByID(context.Background(), currencyBTC, "", 23487)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSCancelTransferByID(currencyBTC, "", 23487); err != nil {
		t.Error(err)
	}
}

func TestGetTransfers(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetTransfers(context.Background(), currencyBTC, 0, 0)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveTransfers(currencyBTC, 0, 0); err != nil {
		t.Error(err)
	}
}

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	_, err := d.CancelWithdrawal(context.Background(), currencyBTC, 123844)
	if err != nil && !strings.Contains(err.Error(), "withdrawal with given id and currency not found") {
		t.Error(err)
	}
	if _, err = d.WSCancelWithdrawal(currencyBTC, 123844); err != nil && !strings.Contains(err.Error(), "withdrawal with given id and currency not found") {
		t.Error(err)
	}
}

func TestCreateDepositAddress(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	_, err := d.CreateDepositAddress(context.Background(), currencySOL)
	if err != nil && !strings.Contains(err.Error(), "max_addr_count_exceeded") {
		t.Error(err)
	}
	if _, err = d.WSCreateDepositAddress(currencySOL); err != nil && !strings.Contains(err.Error(), "max_addr_count_exceeded") {
		t.Error(err)
	}
}

func TestGetCurrentDepositAddress(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetCurrentDepositAddress(context.Background(), currencyETH)
	if err != nil {
		t.Error(err)
	}
	if _, err := d.WSRetrieveCurrentDepositAddress(currencyETH); err != nil {
		t.Error(err)
	}
}

func TestGetDeposits(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetDeposits(context.Background(), currencyBTC, 25, 0)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveDeposits(currencyBTC, 25, 0); err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawals(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetWithdrawals(context.Background(), currencyBTC, 25, 0)
	if err != nil {
		t.Error(err)
	}
	if _, err := d.WSRetrieveWithdrawals(currencyBTC, 25, 0); err != nil {
		t.Error(err)
	}
}

func TestSubmitTransferToSubAccount(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	_, err := d.SubmitTransferToSubAccount(context.Background(), currencyBTC, 0.01, 13434)
	if err != nil && !strings.Contains(err.Error(), "transfer_not_allowed") {
		t.Error(err)
	}
	if _, err = d.WSSubmitTransferToSubAccount(currencyBTC, 0.01, 13434); err != nil && !strings.Contains(err.Error(), "transfer_not_allowed") {
		t.Error(err)
	}
}

func TestSubmitTransferToUser(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	_, err := d.SubmitTransferToUser(context.Background(), currencyBTC, "", "13434", 0.001)
	if err != nil && !strings.Contains(err.Error(), "transfer_not_allowed") {
		t.Error(err)
	}
	if _, err = d.WSSubmitTransferToUser(currencyBTC, "", "0x4aa0753d798d668056920094d65321a8e8913e26", 0.001); err != nil {
		t.Error(err)
	}
}

func TestSubmitWithdraw(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	_, err := d.SubmitWithdraw(context.Background(), currencyBTC, core.BitcoinDonationAddress, "", 0.001)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSSubmitWithdraw(currencyBTC, core.BitcoinDonationAddress, "", 0.001); err != nil {
		t.Error(err)
	}
}

func TestGetAnnouncements(t *testing.T) {
	t.Parallel()
	_, err := d.GetAnnouncements(context.Background(), time.Now(), 5)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveAnnouncements(time.Now(), 5); err != nil {
		t.Error(err)
	}
}

func TestGetPublicPortfolioMargins(t *testing.T) {
	info, err := d.GetInstrumentData(context.Background(), "BTC-PERPETUAL")
	if err != nil {
		t.Skip(err)
	}
	time.Sleep(time.Second)
	if _, err = d.GetPublicPortfolioMargins(context.Background(), currencyBTC, map[string]float64{
		"BTC-PERPETUAL": info.ContractSize * 2,
	}); err != nil {
		t.Error(err)
	}
}

func TestGetAccessLog(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetAccessLog(context.Background(), 0, 0)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveAccessLog(0, 0); err != nil {
		t.Error(err)
	}
}

func TestChangeAPIKeyName(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.ChangeAPIKeyName(context.Background(), 1, "TestKey123")
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSChangeAPIKeyName(1, "TestKey123"); err != nil {
		t.Error(err)
	}
}

func TestChangeScopeInAPIKey(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	_, err := d.ChangeScopeInAPIKey(context.Background(), 1, "account:read_write")
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSChangeScopeInAPIKey(1, "account:read_write"); err != nil {
		t.Error(err)
	}
}

func TestChangeSubAccountName(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	err := d.ChangeSubAccountName(context.Background(), 1, "new_sub")
	if err != nil && !strings.Contains(err.Error(), "unauthorized") {
		t.Error(err)
	}
	if err = d.WSChangeSubAccountName(1, "new_sub"); err != nil && !strings.Contains(err.Error(), "unauthorized") {
		t.Error(err)
	}
}

func TestCreateAPIKey(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	_, err := d.CreateAPIKey(context.Background(), "account:read_write", "new_sub", false)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSCreateAPIKey("account:read_write", "new_sub", false); err != nil {
		t.Error(err)
	}
}

func TestCreateSubAccount(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	_, err := d.CreateSubAccount(context.Background())
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSCreateSubAccount(); err != nil {
		t.Error(err)
	}
}

func TestDisableAPIKey(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	_, err := d.DisableAPIKey(context.Background(), 1)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSDisableAPIKey(1); err != nil {
		t.Error(err)
	}
}

func TestDisableTFAForSubAccount(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	// Use with caution will reduce the security of the account
	err := d.DisableTFAForSubAccount(context.Background(), 1)
	if err != nil && !strings.Contains(err.Error(), "Method not found") { // this functionality is removed by now.
		t.Error(err)
	}
	if err = d.WSDisableTFAForSubAccount(1); err != nil && !strings.Contains(err.Error(), "Method not found") { // this functionality is removed by now.
		t.Error(err)
	}
}

func TestEnableAffiliateProgram(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	err := d.EnableAffiliateProgram(context.Background())
	if err != nil && !strings.Contains(err.Error(), "not_allowed_to_enable_affiliate_program") {
		t.Error(err)
	}
	if err = d.WSEnableAffiliateProgram(); err != nil && !strings.Contains(err.Error(), "not_allowed_to_enable_affiliate_program") {
		t.Error(err)
	}
}

func TestEnableAPIKey(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	_, err := d.EnableAPIKey(context.Background(), 1)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSEnableAPIKey(1); err != nil {
		t.Error(err)
	}
}

func TestGetAffiliateProgramInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetAffiliateProgramInfo(context.Background())
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveAffiliateProgramInfo(1); err != nil {
		t.Error(err)
	}
}

func TestGetEmailLanguage(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetEmailLanguage(context.Background())
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveEmailLanguage(); err != nil {
		t.Error(err)
	}
}

func TestGetNewAnnouncements(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetNewAnnouncements(context.Background())
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveNewAnnouncements(); err != nil {
		t.Error(err)
	}
}

func TestGetPrivatePortfolioMargins(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetPrivatePortfolioMargins(context.Background(), currencyBTC, false, nil)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrievePricatePortfolioMargins(currencyBTC, false, nil); err != nil {
		t.Error(err)
	}
}

func TestGetPosition(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetPosition(context.Background(), btcPerpInstrument)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrievePosition(btcPerpInstrument); err != nil {
		t.Error(err)
	}
}

func TestGetSubAccounts(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetSubAccounts(context.Background(), false)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveSubAccounts(false); err != nil {
		t.Error(err)
	}
}

func TestGetSubAccountDetails(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetSubAccountDetails(context.Background(), currencyBTC, false)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveSubAccountDetails(currencyBTC, false); err != nil {
		t.Error(err)
	}
}

func TestGetPositions(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetPositions(context.Background(), currencyBTC, "option")
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetPositions(context.Background(), currencyETH, "")
	if err != nil {
		t.Error(err)
	}
	_, err = d.WSRetrievePositions(currencyBTC, "option")
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrievePositions(currencyETH, ""); err != nil {
		t.Error(err)
	}
}

func TestGetTransactionLog(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetTransactionLog(context.Background(), currencyBTC, "trade", time.Now().Add(-24*time.Hour), time.Now(), 5, 0)
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetTransactionLog(context.Background(), currencyBTC, "trade", time.Now().Add(-24*time.Hour), time.Now(), 0, 0)
	if err != nil {
		t.Error(err)
	}
	_, err = d.WSRetrieveTransactionLog(currencyBTC, "trade", time.Now().Add(-24*time.Hour), time.Now(), 5, 0)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveTransactionLog(currencyBTC, "trade", time.Now().Add(-24*time.Hour), time.Now(), 0, 0); err != nil {
		t.Error(err)
	}
}

func TestGetUserLocks(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetUserLocks(context.Background())
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveUserLocks(); err != nil {
		t.Error(err)
	}
}

func TestListAPIKeys(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.ListAPIKeys(context.Background(), "")
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSListAPIKeys(""); err != nil {
		t.Error(err)
	}
}

func TestRemoveAPIKey(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	err := d.RemoveAPIKey(context.Background(), 1)
	if err != nil {
		t.Error(err)
	}
	if err = d.WSRemoveAPIKey(1); err != nil {
		t.Error(err)
	}
}

func TestRemoveSubAccount(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	err := d.RemoveSubAccount(context.Background(), 1)
	if err != nil && !strings.Contains(err.Error(), "unauthorized") {
		t.Error(err)
	}
	if err = d.WSRemoveSubAccount(1); err != nil && !strings.Contains(err.Error(), "unauthorized") {
		t.Error(err)
	}
}

func TestResetAPIKey(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.ResetAPIKey(context.Background(), 1)
	if err != nil {
		t.Error(err)
	}
	if err = d.WSResetAPIKey(1); err != nil {
		t.Error(err)
	}
}

func TestSetAnnouncementAsRead(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	err := d.SetAnnouncementAsRead(context.Background(), 1)
	if err != nil {
		t.Error(err)
	}
}

func TestSetEmailForSubAccount(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	err := d.SetEmailForSubAccount(context.Background(), 1, "wrongemail@wrongemail.com")
	if err != nil && !strings.Contains(err.Error(), "could not link email (wrongemail@wrongemail.com) to subaccount 1") {
		t.Error(err)
	}
	if err = d.WSSetEmailForSubAccount(1, "wrongemail@wrongemail.com"); err != nil && !strings.Contains(err.Error(), "could not link email (wrongemail@wrongemail.com) to subaccount 1") {
		t.Error(err)
	}
}

func TestSetEmailLanguage(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	err := d.SetEmailLanguage(context.Background(), "en")
	if err != nil {
		t.Error(err)
	}
	if err := d.WSSetEmailLanguage("en"); err != nil {
		t.Error(err)
	}
}

func TestSetPasswordForSubAccount(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	// Caution! This may reduce the security of the subaccount
	err := d.SetPasswordForSubAccount(context.Background(), 1, "randompassword123")
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSSetPasswordForSubAccount(1, "randompassword123"); err != nil {
		t.Error(err)
	}
}

func TestToggleNotificationsFromSubAccount(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	err := d.ToggleNotificationsFromSubAccount(context.Background(), 1, false)
	if err != nil && !strings.Contains(err.Error(), "unauthorized") {
		t.Error(err)
	}
	if err = d.WSToggleNotificationsFromSubAccount(1, false); err != nil && !strings.Contains(err.Error(), "unauthorized") {
		t.Error(err)
	}
}

func TestTogglePortfolioMargining(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	subaccount, err := d.GetSubAccountDetails(context.Background(), currencyBTC, false)
	if err != nil {
		t.Skip(err)
	}
	if len(subaccount) == 0 {
		t.Skip("no subaccount record found")
	}
	_, err = d.TogglePortfolioMargining(context.Background(), subaccount[0].UID, false, false)
	if err != nil && !strings.Contains(err.Error(), "account is already on SM") {
		t.Error(err)
	}
	if _, err = d.WSTogglePortfolioMargining(subaccount[0].UID, false, false); err != nil && !strings.Contains(err.Error(), "account is already on SM") {
		t.Error(err)
	}
}

func TestToggleSubAccountLogin(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	err := d.ToggleSubAccountLogin(context.Background(), 1, false)
	if err != nil && !strings.Contains(err.Error(), "unauthorized") {
		t.Error(err)
	}
	if err = d.WSToggleSubAccountLogin(1, false); err != nil && !strings.Contains(err.Error(), "unauthorized") {
		t.Error(err)
	}
}

func TestSubmitBuy(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	pairs, err := d.FetchTradablePairs(context.Background(), asset.Futures)
	if err != nil {
		t.Skip(err)
	}
	_, err = d.SubmitBuy(context.Background(), &OrderBuyAndSellParams{
		Instrument: pairs[0].String(), OrderType: "limit",
		Label: "testOrder", TimeInForce: "",
		Trigger: "", Advanced: "",
		Amount: 30, Price: 500000,
		MaxShow: 0, TriggerPrice: 0,
		PostOnly: false, RejectPostOnly: false,
		ReduceOnly: false, MMP: false})
	if err != nil && !strings.Contains(err.Error(), "not_enough_funds") {
		t.Error(err)
	}
	if _, err = d.WSSubmitBuy(&OrderBuyAndSellParams{
		Instrument: btcPerpInstrument, OrderType: "limit",
		Label: "testOrder", TimeInForce: "",
		Trigger: "", Advanced: "",
		Amount: 30, Price: 500000,
		MaxShow: 0, TriggerPrice: 0,
		PostOnly: false, RejectPostOnly: false,
		ReduceOnly: false, MMP: false}); err != nil && !strings.Contains(err.Error(), "not_enough_funds") {
		t.Error(err)
	}
}

func TestSubmitSell(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	info, err := d.GetInstrumentData(context.Background(), btcPerpInstrument)
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.SubmitSell(context.Background(), &OrderBuyAndSellParams{Instrument: btcPerpInstrument, OrderType: "limit", Label: "testOrder", TimeInForce: "", Trigger: "", Advanced: "", Amount: info.ContractSize * 3, Price: 500000, MaxShow: 0, TriggerPrice: 0, PostOnly: false, RejectPostOnly: false, ReduceOnly: false, MMP: false})
	if err != nil && !strings.Contains(err.Error(), "not_enough_funds") {
		t.Error(err)
	}
	if _, err = d.WSSubmitSell(&OrderBuyAndSellParams{
		Instrument: btcPerpInstrument, OrderType: "limit",
		Label: "testOrder", TimeInForce: "",
		Trigger: "", Advanced: "", Amount: info.ContractSize * 3,
		Price: 500000, MaxShow: 0, TriggerPrice: 0, PostOnly: false,
		RejectPostOnly: false, ReduceOnly: false, MMP: false}); err != nil && !strings.Contains(err.Error(), "not_enough_funds") {
		t.Error(err)
	}
}

func TestEditOrderByLabel(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	_, err := d.EditOrderByLabel(context.Background(), &OrderBuyAndSellParams{Label: "incorrectUserLabel", Instrument: btcPerpInstrument,
		Advanced: "", Amount: 1, Price: 30000, TriggerPrice: 0, PostOnly: false, ReduceOnly: false, RejectPostOnly: false, MMP: false})
	if err != nil && !strings.Contains(err.Error(), "order_not_found") {
		t.Error(err)
	}
	if _, err = d.WSEditOrderByLabel(&OrderBuyAndSellParams{Label: "incorrectUserLabel", Instrument: btcPerpInstrument,
		Advanced: "", Amount: 1, Price: 30000, TriggerPrice: 0, PostOnly: false, ReduceOnly: false, RejectPostOnly: false, MMP: false}); err != nil && !strings.Contains(err.Error(), "order_not_found") {
		t.Error(err)
	}
}

func TestSubmitCancel(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	_, err := d.SubmitCancel(context.Background(), "incorrectID")
	if err != nil && !strings.Contains(err.Error(), "order_not_found") {
		t.Error(err)
	}
	if _, err = d.WSSubmitCancel("incorrectID"); err != nil && !strings.Contains(err.Error(), "order_not_found") {
		t.Error(err)
	}
}

func TestSubmitCancelAll(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	_, err := d.SubmitCancelAll(context.Background())
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSSubmitCancelAll(); err != nil {
		t.Error(err)
	}
}

func TestSubmitCancelAllByCurrency(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	_, err := d.SubmitCancelAllByCurrency(context.Background(), currencyBTC, "option", "")
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSSubmitCancelAllByCurrency(currencyBTC, "option", ""); err != nil {
		t.Error(err)
	}
}

func TestSubmitCancelAllByInstrument(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	_, err := d.SubmitCancelAllByInstrument(context.Background(), btcPerpInstrument, "all")
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSSubmitCancelAllByInstrument(btcPerpInstrument, "all"); err != nil {
		t.Error(err)
	}
}

func TestSubmitCancelByLabel(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	_, err := d.SubmitCancelByLabel(context.Background(), "incorrectOrderLabel", "")
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSSubmitCancelByLabel("incorrectOrderLabel", ""); err != nil {
		t.Error(err)
	}
}

func TestSubmitClosePosition(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	_, err := d.SubmitClosePosition(context.Background(), btcPerpInstrument, "limit", 35000)
	if err != nil && !strings.Contains(err.Error(), "already_closed") {
		t.Error(err)
	}
	if _, err = d.WSSubmitClosePosition(btcPerpInstrument, "limit", 35000); err != nil && !strings.Contains(err.Error(), "already_closed") {
		t.Error(err)
	}
}

func TestGetMargins(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetMargins(context.Background(), btcPerpInstrument, 5, 35000)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveMargins(btcPerpInstrument, 5, 35000); err != nil {
		t.Error(err)
	}
}

func TestGetMMPConfig(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetMMPConfig(context.Background(), currencyETH)
	if err != nil && !strings.Contains(err.Error(), "MMP disabled") {
		t.Error(err)
	}
	if _, err = d.WSRetrieveMMPConfig(currencyETH); err != nil && !strings.Contains(err.Error(), "MMP disabled") {
		t.Error(err)
	}
}

func TestGetOpenOrdersByCurrency(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetOpenOrdersByCurrency(context.Background(), currencyBTC, "option", "all")
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveOpenOrdersByCurrency(currencyBTC, "option", "all"); err != nil {
		t.Error(err)
	}
}

func TestGetOpenOrdersByInstrument(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetOpenOrdersByInstrument(context.Background(), btcPerpInstrument, "all")
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveOpenOrdersByInstrument(btcPerpInstrument, "all"); err != nil {
		t.Error(err)
	}
}

func TestGetOrderHistoryByCurrency(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetOrderHistoryByCurrency(context.Background(), currencyBTC, "future", 0, 0, false, false)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveOrderHistoryByCurrency(currencyBTC, "future", 0, 0, false, false); err != nil {
		t.Error(err)
	}
}

func TestGetOrderHistoryByInstrument(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetOrderHistoryByInstrument(context.Background(), btcPerpInstrument, 0, 0, false, false)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveOrderHistoryByInstrument(btcPerpInstrument, 0, 0, false, false); err != nil {
		t.Error(err)
	}
}

func TestGetOrderMarginsByID(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetOrderMarginsByID(context.Background(), []string{"ETH-349280", "ETH-349279", "ETH-349278"})
	if err != nil && strings.Contains(err.Error(), "value must be a list") {
		t.Skip(err)
	} else if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveOrderMarginsByID([]string{"ETH-349280", "ETH-349279", "ETH-349278"}); err != nil && strings.Contains(err.Error(), "value must be a list") {
		t.Skip(err)
	} else if err != nil {
		t.Error(err)
	}
}

func TestGetOrderState(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetOrderState(context.Background(), "brokenid123")
	if err != nil && !strings.Contains(err.Error(), "order_not_found") {
		t.Error(err)
	}
	if _, err = d.WSRetrievesOrderState("brokenid123"); err != nil && !strings.Contains(err.Error(), "order_not_found") {
		t.Error(err)
	}
}

func TestGetTriggerOrderHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetTriggerOrderHistory(context.Background(), currencyETH, "", "", 0)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveTriggerOrderHistory(currencyETH, "", "", 0); err != nil {
		t.Error(err)
	}
}

func TestGetUserTradesByCurrency(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetUserTradesByCurrency(context.Background(), currencyETH, "future", "", "", "asc", 0, false)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveUserTradesByCurrency(currencyETH, "future", "", "", "asc", 0, false); err != nil {
		t.Error(err)
	}
}

func TestGetUserTradesByCurrencyAndTime(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetUserTradesByCurrencyAndTime(context.Background(), currencyETH, "future", "default", 5, false, time.Now().Add(-time.Hour*10), time.Now().Add(-time.Hour*1))
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveUserTradesByCurrencyAndTime(currencyETH, "future", "default", 5, false, time.Now().Add(-time.Hour*10), time.Now().Add(-time.Hour*1)); err != nil {
		t.Error(err)
	}
}

func TestGetUserTradesByInstrument(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetUserTradesByInstrument(context.Background(), btcPerpInstrument, "asc", 5, 10, 4, true)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveUserTradesByInstrument(btcPerpInstrument, "asc", 5, 10, 4, true); err != nil {
		t.Error(err)
	}
}

func TestGetUserTradesByInstrumentAndTime(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetUserTradesByInstrumentAndTime(context.Background(), btcPerpInstrument, "asc", 10, false, time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveUserTradesByInstrumentAndTime(btcPerpInstrument, "asc", 10, false, time.Now().Add(-time.Hour), time.Now()); err != nil {
		t.Error(err)
	}
}

func TestGetUserTradesByOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetUserTradesByOrder(context.Background(), "wrongOrderID", "default")
	if err != nil && !strings.Contains(err.Error(), "order_not_found") {
		t.Error(err)
	}
	if _, err = d.WSRetrieveUserTradesByOrder("wrongOrderID", "default"); err != nil && !strings.Contains(err.Error(), "order_not_found") {
		t.Error(err)
	}
}

func TestResetMMP(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	err := d.ResetMMP(context.Background(), currencyBTC)
	if err != nil && !strings.Contains(err.Error(), "MMP disabled") {
		t.Error(err)
	}
	if err = d.WSResetMMP(currencyBTC); err != nil && !strings.Contains(err.Error(), "MMP disabled") {
		t.Error(err)
	}
}

func TestSendRequestForQuote(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	err := d.SendRequestForQuote(context.Background(), "BTC-PERPETUAL", 1000, order.Buy)
	if err != nil {
		t.Error(err)
	}
	if err = d.WSSendRequestForQuote("BTC-PERPETUAL", 1000, order.Buy); err != nil {
		t.Error(err)
	}
}

func TestSetMMPConfig(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	err := d.SetMMPConfig(context.Background(), currencyBTC, 5, 5, 0, 0)
	if err != nil && !strings.Contains(err.Error(), "MMP disabled") {
		t.Error(err)
	}
	if err = d.WSSetMMPConfig(currencyBTC, 5, 5, 0, 0); err != nil && !strings.Contains(err.Error(), "MMP disabled") {
		t.Error(err)
	}
}

func TestGetSettlementHistoryByCurency(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetSettlementHistoryByCurency(context.Background(), currencyBTC, "settlement", "", 10, time.Now().Add(-time.Hour))
	if err != nil {
		t.Error(err)
	}
	_, err = d.WSRetrieveSettlementHistoryByCurency(currencyBTC, "settlement", "", 10, time.Now().Add(-time.Hour))
	if err != nil {
		t.Error(err)
	}
}

func TestGetSettlementHistoryByInstrument(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetSettlementHistoryByInstrument(context.Background(), btcPerpInstrument, "settlement", "", 10, time.Now().Add(-time.Hour))
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveSettlementHistoryByInstrument(btcPerpInstrument, "settlement", "", 10, time.Now().Add(-time.Hour)); err != nil {
		t.Error(err)
	}
}

func TestSubmitEdit(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	_, err := d.SubmitEdit(context.Background(), &OrderBuyAndSellParams{OrderID: "incorrectID", Advanced: "", TriggerPrice: 0.001, Price: 100000, Amount: 123})
	if err != nil && !strings.Contains(err.Error(), "order_not_found") {
		t.Error(err)
	}
	if _, err = d.WSSubmitEdit(&OrderBuyAndSellParams{
		OrderID:      "incorrectID",
		Advanced:     "",
		TriggerPrice: 0.001,
		Price:        100000,
		Amount:       123,
	}); err != nil && !strings.Contains(err.Error(), "order_not_found") {
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
	combos, err := d.WSRetrieveComboIDS(currencyBTC, "")
	if err != nil {
		t.Skip(err)
	}
	if len(combos) == 0 {
		t.Skip("no combo instance found for currency BTC")
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
	if _, err := d.WSRetrieveComboDetails(combos[0]); err != nil {
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
		t.Skip(endpointAuthorizationToManipulate)
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
			InstrumentName: instruments[0].String(),
			Direction:      "sell",
		},
		{
			InstrumentName: instruments[1].String(),
			Direction:      "sell",
			Amount:         1200,
		},
	})
	if err != nil && !errors.Is(errInvalidAmount, err) {
		t.Errorf("expecting %v, but found %v", errInvalidAmount, err)
	}
	_, err = d.CreateCombo(context.Background(), []ComboParam{
		{
			InstrumentName: instruments[0].String(),
			Amount:         123,
		},
		{
			InstrumentName: instruments[1].String(),
			Direction:      "sell",
			Amount:         1200,
		},
	})
	if err != nil && !strings.Contains(err.Error(), "invalid direction") {
		t.Errorf("expecting error message 'invalid direction', but found %v", err)
	}
	_, err = d.CreateCombo(context.Background(), []ComboParam{
		{
			InstrumentName: instruments[0].String(),
			Direction:      "buy",
			Amount:         123,
		},
		{
			InstrumentName: instruments[1].String(),
			Direction:      "buy",
			Amount:         1200,
		},
	})
	if err != nil && !strings.Contains(err.Error(), "not_enough_funds") {
		t.Error(err)
	}
	_, err = d.WSCreateCombo([]ComboParam{})
	if err != nil && !errors.Is(errNoArgumentPassed, err) {
		t.Errorf("expecting %v, but found %v", errNoArgumentPassed, err)
	}
	_, err = d.WSCreateCombo([]ComboParam{
		{
			InstrumentName: instruments[0].String(),
			Direction:      "sell",
		},
		{
			InstrumentName: instruments[1].String(),
			Direction:      "sell",
			Amount:         1200,
		},
	})
	if err != nil && !errors.Is(errInvalidAmount, err) {
		t.Errorf("expecting %v, but found %v", errInvalidAmount, err)
	}
	_, err = d.WSCreateCombo([]ComboParam{
		{
			InstrumentName: instruments[0].String(),
			Amount:         123,
		},
		{
			InstrumentName: instruments[1].String(),
			Direction:      "sell",
			Amount:         1200,
		},
	})
	if err != nil && !strings.Contains(err.Error(), "invalid direction") {
		t.Errorf("expecting error message 'invalid direction', but found %v", err)
	}
	if _, err = d.WSCreateCombo([]ComboParam{
		{
			InstrumentName: instruments[0].String(),
			Direction:      "buy",
			Amount:         123,
		},
		{
			InstrumentName: instruments[1].String(),
			Direction:      "buy",
			Amount:         1200,
		},
	}); err != nil && !strings.Contains(err.Error(), "not_enough_funds") {
		t.Error(err)
	}
}

func TestVerifyBlockTrade(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	info, err := d.GetInstrumentData(context.Background(), btcPerpInstrument)
	if err != nil {
		t.Skip(err)
	}
	_, err = d.VerifyBlockTrade(context.Background(), time.Now(), "sdjkafdad", "maker", "", []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: btcPerpInstrument,
			Direction:      order.Buy.Lower(),
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	})
	if err != nil && !strings.Contains(err.Error(), "not_enough_funds") {
		t.Error(err)
	}
	_, err = d.WSVerifyBlockTrade(time.Now(), "sdjkafdad", "maker", "", []BlockTradeParam{
		{
			Price:          0.777 * 22000,
			InstrumentName: btcPerpInstrument,
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
		t.Skip(endpointAuthorizationToManipulate)
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
	if _, err = d.WSExecuteBlockTrade(time.Now(), "sdjkafdad", "maker", "", []BlockTradeParam{
		{
			Price:          0.777 * 22000,
			InstrumentName: btcPerpInstrument,
			Direction:      "buy",
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	}); err != nil && !strings.Contains(err.Error(), "not_enough_funds") {
		t.Error(err)
	}
}

func TestGetUserBlocTrade(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetUserBlockTrade(context.Background(), "12345567")
	if err != nil && !strings.Contains(err.Error(), "block_trade_not_found") {
		t.Error(err)
	}
	if _, err = d.WSRetrieveUserBlockTrade("12345567"); err != nil && !strings.Contains(err.Error(), "block_trade_not_found") {
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
		t.Skip(authenticationSkipMessage)
	}
	_, err = d.GetLastBlockTradesByCurrency(context.Background(), "SOL", "", "", 5)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveLastBlockTradesByCurrency("SOL", "", "", 5); err != nil {
		t.Error(err)
	}
}

func TestMovePositions(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
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
	_, err = d.WSMovePositions(currencyBTC, 123, 345, []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: btcPerpInstrument,
			Direction:      "buy",
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	})
	if err != nil && !strings.Contains(err.Error(), "must be one of the subaccounts") {
		t.Error(err)
	}
}

var announcementPushDataJSON = `{    "jsonrpc": "2.0",    "method": "subscription",    "params": {         "channel": "announcements",         "data": {            "action": "new",            "body": "Lorem ipsum dolor sit amet, consectetur adipiscing elit.",            "id": 1532593832021,            "important": true,            "publication_timestamp": 1532593832021,            "title": "Example announcement"        }    }}`

func TestHandlerAnnouncementPushData(t *testing.T) {
	t.Parallel()
	if err := d.wsHandleData([]byte(announcementPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var (
	orderbookPushDataJSON       = `{	"params" : {"data" : {"timestamp" : 1554375447971,"instrument_name" : "ETH-PERPETUAL","change_id" : 109615,"bids" : [[160,40]],"asks" : [[161,20]]},	  "channel" : "book.ETH-PERPETUAL.100.1.100ms"	},	"method" : "subscription",	"jsonrpc" : "2.0"}`
	orderbookUpdatePushDataJSON = `{"params" : {"data" : {"type" : "snapshot","timestamp" : 1554373962454,"instrument_name" : "BTC-PERPETUAL","change_id" : 297217,"bids" : [["new",5042.34,30],["new",5041.94,20]],"asks" : [["new",5042.64,40],["new",5043.3,40]]},"channel" : "book.BTC-PERPETUAL.100ms"},	"method" : "subscription",	"jsonrpc" : "2.0"  }`
)

func TestOrderbookPushData(t *testing.T) {
	t.Parallel()
	if err := d.wsHandleData([]byte(orderbookPushDataJSON)); err != nil {
		t.Error(err)
	}
	if err := d.wsHandleData([]byte(orderbookUpdatePushDataJSON)); err != nil {
		t.Error(err)
	}
}

var candlestickPushDataJSON = `{"params" : {"data" : {"volume" : 0.05219351,"tick" : 1573645080000,"open" : 8869.79,"low" : 8788.25,"high" : 8870.31,"cost" : 460,"close" : 8791.25},"channel" : "chart.trades.BTC-PERPETUAL.1"},	"method" : "subscription",	"jsonrpc" : "2.0"  }`

func TestChartPushData(t *testing.T) {
	t.Parallel()
	if err := d.wsHandleData([]byte(candlestickPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var indexPricePushDataJSON = `{	"params" : {"data" : {"timestamp" : 1550588002899,"price" : 3937.89,"index_name" : "btc_usd"},"channel" : "deribit_price_index.btc_usd"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`

func TestIndexPricePushData(t *testing.T) {
	t.Parallel()
	if err := d.wsHandleData([]byte(indexPricePushDataJSON)); err != nil {
		t.Error(err)
	}
}

var priceRankingPushDataJSON = `{"params" : {"data" : [{"weight" : 14.29,		  "timestamp" : 1573202284040,		  "price" : 9109.35,		  "original_price" : 9109.35,		  "identifier" : "bitfinex",		  "enabled" : true		},		{		  "weight" : 14.29,		  "timestamp" : 1573202284055,		  "price" : 9084.83,		  "original_price" : 9084.83,		  "identifier" : "bitstamp",		  "enabled" : true		},		{		  "weight" : 14.29,		  "timestamp" : 1573202283191,		  "price" : 9079.91,		  "original_price" : 9079.91,		  "identifier" : "bittrex",		  "enabled" : true		},		{		  "weight" : 14.29,		  "timestamp" : 1573202284094,		  "price" : 9085.81,		  "original_price" : 9085.81,		  "identifier" : "coinbase",		  "enabled" : true		},		{		  "weight" : 14.29,		  "timestamp" : 1573202283881,		  "price" : 9086.27,		  "original_price" : 9086.27,		  "identifier" : "gemini",		  "enabled" : true		},		{		  "weight" : 14.29,		  "timestamp" : 1573202283420,		  "price" : 9088.38,		  "original_price" : 9088.38,		  "identifier" : "itbit",		  "enabled" : true		},		{		  "weight" : 14.29,		  "timestamp" : 1573202283459,		  "price" : 9083.6,		  "original_price" : 9083.6,		  "identifier" : "kraken",		  "enabled" : true		},		{		  "weight" : 0,		  "timestamp" : 0,		  "price" : null,		  "original_price" : null,		  "identifier" : "lmax",		  "enabled" : false		}	  ],	  "channel" : "deribit_price_ranking.btc_usd"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`

func TestPriceRakingPushData(t *testing.T) {
	t.Parallel()
	if err := d.wsHandleData([]byte(priceRankingPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var priceStatisticsPushDataJSON = `{"params" : {"data" : {"low24h" : 58012.08,"index_name" : "btc_usd","high_volatility" : false,"high24h" : 59311.42,"change24h" : 1009.61},"channel" : "deribit_price_statistics.btc_usd"},	"method" : "subscription",	"jsonrpc" : "2.0"  }`

func TestPriceStatisticsPushData(t *testing.T) {
	t.Parallel()
	if err := d.wsHandleData([]byte(priceStatisticsPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var volatilityIndexPushDataJSON = `{"params" : {"data" : {"volatility" : 129.36,"timestamp" : 1619777946007,"index_name" : "btc_usd","estimated_delivery" : 129.36},"channel" : "deribit_volatility_index.btc_usd"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`

func TestVolatilityIndexPushData(t *testing.T) {
	t.Parallel()
	if err := d.wsHandleData([]byte(volatilityIndexPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var estimatedExpirationPricePushDataJSON = `{"params" : {"data" : {"seconds" : 180929,"price" : 3939.73,"is_estimated" : false},"channel" : "estimated_expiration_price.btc_usd"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`

func TestEstimatedExpirationPricePushData(t *testing.T) {
	t.Parallel()
	if err := d.wsHandleData([]byte(estimatedExpirationPricePushDataJSON)); err != nil {
		t.Error(err)
	}
}

var incrementalTickerPushDataJSON = `{"params" : {"data" : {"timestamp" : 1623060194301,"stats" : {"volume_usd" : 284061480,"volume" : 7871.02139035,"price_change" : 0.7229,"low" : 35213.5,"high" : 36824.5},"state" : "open","settlement_price" : 36169.49,		"open_interest" : 502097590,		"min_price" : 35898.37,"max_price" : 36991.72,"mark_price" : 36446.51,"last_price" : 36457.5,"instrument_name" : "BTC-PERPETUAL","index_price" : 36441.64,		"funding_8h" : 0.0000211,"estimated_delivery_price" : 36441.64,"current_funding" : 0,"best_bid_price" : 36442.5,"best_bid_amount" : 5000,"best_ask_price" : 36443,"best_ask_amount" : 100},"channel" : "incremental_ticker.BTC-PERPETUAL"},	"method" : "subscription","jsonrpc" : "2.0"  }`

func TestIncrementalTickerPushData(t *testing.T) {
	t.Parallel()
	if err := d.wsHandleData([]byte(incrementalTickerPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var instrumentStatePushDataJSON = `{"params" : {"data" : {"timestamp" : 1553080940000,"state" : "created","instrument_name" : "BTC-22MAR19"},"channel" : "instrument.state.any.any"},	"method" : "subscription",	"jsonrpc" : "2.0"  }`

func TestInstrumentStatePushData(t *testing.T) {
	t.Parallel()
	if err := d.wsHandleData([]byte(instrumentStatePushDataJSON)); err != nil {
		t.Error(err)
	}
}

var markPriceOptionsPushDataJSON = `{"params" : {"data" : [{"timestamp" : 1622470378005,"mark_price" : 0.0333,"iv" : 0.9,"instrument_name" : "BTC-2JUN21-37000-P"},{"timestamp" : 1622470378005,"mark_price" : 0.117,"iv" : 0.9,"instrument_name" : "BTC-4JUN21-40500-P"},{"timestamp" : 1622470378005,"mark_price" : 0.0177,"iv" : 0.9,"instrument_name" : "BTC-4JUN21-38250-C"},{"timestamp" : 1622470378005,		  "mark_price" : 0.0098,		  "iv" : 0.9,		  "instrument_name" : "BTC-1JUN21-37000-C"		},		{		  "timestamp" : 1622470378005,		  "mark_price" : 0.0371,		  "iv" : 0.9,		  "instrument_name" : "BTC-4JUN21-36500-P"		}	  ],	  "channel" : "markprice.options.btc_usd"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`

func TestMarkPriceOption(t *testing.T) {
	t.Parallel()
	if err := d.wsHandleData([]byte(markPriceOptionsPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var platformStatePushDataJSON = `{"params" : {"data" : {"allow_unauthenticated_public_requests" : true},"channel" : "platform_state.public_methods_state"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`

func TestPlatformStatsPushData(t *testing.T) {
	t.Parallel()
	if err := d.wsHandleData([]byte(platformStatePushDataJSON)); err != nil {
		t.Error(err)
	}
}

var quoteTickerPushDataJSON = `{"params" : {"data" : {"timestamp" : 1550658624149,"instrument_name" : "BTC-PERPETUAL","best_bid_price" : 3914.97,"best_bid_amount" : 40,"best_ask_price" : 3996.61,"best_ask_amount" : 50},"channel" : "quote.BTC-PERPETUAL"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`

func TestQuotePushData(t *testing.T) {
	t.Parallel()
	if err := d.wsHandleData([]byte(quoteTickerPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var rfqPushDataJSON = `{"params" : {"data" : {"state" : true,"side" : null,"last_rfq_tstamp" : 1634816143836,"instrument_name" : "BTC-PERPETUAL","amount" : null	  },"channel" : "rfq.btc"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`

func TestRFQPushData(t *testing.T) {
	t.Parallel()
	if err := d.wsHandleData([]byte(rfqPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var instrumentsTickerPushDataJSON = `{"params" : {"data" : {"timestamp" : 1623060194301,"stats" : {"volume_usd" : 284061480,"volume" : 7871.02139035,"price_change" : 0.7229,"low" : 35213.5,"high" : 36824.5},"state" : "open","settlement_price" : 36169.49,"open_interest" : 502097590,"min_price" : 35898.37,		"max_price" : 36991.72,		"mark_price" : 36446.51,		"last_price" : 36457.5,		"interest_value" : 1.7362511643080387,		"instrument_name" : "BTC-PERPETUAL",		"index_price" : 36441.64,		"funding_8h" : 0.0000211,		"estimated_delivery_price" : 36441.64,		"current_funding" : 0,		"best_bid_price" : 36442.5,		"best_bid_amount" : 5000,		"best_ask_price" : 36443,		"best_ask_amount" : 100	  },	  "channel" : "ticker.BTC-PERPETUAL.raw"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`

func TestInstrumentsTickerPushData(t *testing.T) {
	t.Parallel()
	if err := d.wsHandleData([]byte(instrumentsTickerPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var (
	instrumentTradesPushDataJSON = `{"params":{"data":[{"trade_seq":30289442,"trade_id" : "48079269","timestamp" : 1590484512188,"tick_direction" : 2,"price" : 8950,"mark_price" : 8948.9,"instrument_name" : "BTC-PERPETUAL","index_price" : 8955.88,"direction" : "sell","amount" : 10}],"channel" : "trades.BTC-PERPETUAL.raw"},"method":"subscription","jsonrpc":"2.0"}`
	currencyTradesPushDataJSON   = `{"params":{"data":[{"trade_seq":2,"trade_id" : "48079289","timestamp" : 1590484589306,"tick_direction" : 2,"price" : 0.0075,"mark_price" : 0.01062686,"iv" : 47.58,"instrument_name" : "BTC-27MAY20-9000-C",		  "index_price" : 8956.17,"direction" : "sell","amount" : 3}],"channel" : "trades.option.BTC.raw"},"method":"subscription","jsonrpc":"2.0"}`
)

func TestInstrumentTradesPushData(t *testing.T) {
	t.Parallel()
	if err := d.wsHandleData([]byte(instrumentTradesPushDataJSON)); err != nil {
		t.Error(err)
	}
	if err := d.wsHandleData([]byte(currencyTradesPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var (
	changeUpdatesPushDataJSON          = `{"params" : {"data" : {"trades" : [{"trade_seq" : 866638,"trade_id" : "1430914","timestamp" : 1605780344032,"tick_direction" : 1,"state" : "filled","self_trade" : false,"reduce_only" : false,"profit_loss" : 0.00004898,"price" : 17391,"post_only" : false,"order_type" : "market",			"order_id" : "3398016",			"matching_id" : null,			"mark_price" : 17391,			"liquidity" : "T",			"instrument_name" : "BTC-PERPETUAL",			"index_price" : 17501.88,			"fee_currency" : "BTC",			"fee" : 1.6e-7,			"direction" : "sell",			"amount" : 10		  }		],		"positions" : [		  {			"total_profit_loss" : 1.69711368,			"size_currency" : 10.646886321,			"size" : 185160,			"settlement_price" : 16025.83,			"realized_profit_loss" : 0.012454598,			"realized_funding" : 0.01235663,			"open_orders_margin" : 0,			"mark_price" : 17391,			"maintenance_margin" : 0.234575865,			"leverage" : 33,			"kind" : "future",			"interest_value" : 1.7362511643080387,			"instrument_name" : "BTC-PERPETUAL",			"initial_margin" : 0.319750953,			"index_price" : 17501.88,			"floating_profit_loss" : 0.906961435,			"direction" : "buy",			"delta" : 10.646886321,			"average_price" : 15000		  }		],		"orders" : [		  {			"web" : true,			"time_in_force" : "good_til_cancelled",			"replaced" : false,			"reduce_only" : false,			"profit_loss" : 0.00009166,			"price" : 15665.5,			"post_only" : false,			"order_type" : "market",			"order_state" : "filled",			"order_id" : "3398016",			"max_show" : 10,			"last_update_timestamp" : 1605780344032,			"label" : "",			"is_liquidation" : false,			"instrument_name" : "BTC-PERPETUAL",			"filled_amount" : 10,			"direction" : "sell",			"creation_timestamp" : 1605780344032,			"commission" : 1.6e-7,			"average_price" : 17391,			"api" : false,			"amount" : 10		  }		],		"instrument_name" : "BTC-PERPETUAL"	  },	  "channel" : "user.changes.BTC-PERPETUAL.raw"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`
	currencyChangesUpdatesPushDataJSON = `{"params" : {"data" : {"trades" : [{"trade_seq" : 866638,"trade_id" : "1430914","timestamp" : 1605780344032,"tick_direction" : 1,"state" : "filled","self_trade" : false,"reduce_only" : false,"profit_loss" : 0.00004898,"price" : 17391,"post_only" : false,"order_type" : "market","order_id" : "3398016",			"matching_id" : null,			"mark_price" : 17391,			"liquidity" : "T",			"instrument_name" : "BTC-PERPETUAL",			"index_price" : 17501.88,			"fee_currency" : "BTC",			"fee" : 1.6e-7,			"direction" : "sell",			"amount" : 10		  }		],		"positions" : [		  {			"total_profit_loss" : 1.69711368,			"size_currency" : 10.646886321,			"size" : 185160,			"settlement_price" : 16025.83,			"realized_profit_loss" : 0.012454598,			"realized_funding" : 0.01235663,			"open_orders_margin" : 0,			"mark_price" : 17391,			"maintenance_margin" : 0.234575865,			"leverage" : 33,			"kind" : "future",			"interest_value" : 1.7362511643080387,			"instrument_name" : "BTC-PERPETUAL",			"initial_margin" : 0.319750953,			"index_price" : 17501.88,			"floating_profit_loss" : 0.906961435,			"direction" : "buy",			"delta" : 10.646886321,			"average_price" : 15000		  }		],		"orders" : [		  {			"web" : true,			"time_in_force" : "good_til_cancelled",			"replaced" : false,			"reduce_only" : false,			"profit_loss" : 0.00009166,			"price" : 15665.5,			"post_only" : false,			"order_type" : "market",			"order_state" : "filled",			"order_id" : "3398016",			"max_show" : 10,			"last_update_timestamp" : 1605780344032,			"label" : "",			"is_liquidation" : false,			"instrument_name" : "BTC-PERPETUAL",			"filled_amount" : 10,			"direction" : "sell",			"creation_timestamp" : 1605780344032,			"commission" : 1.6e-7,			"average_price" : 17391,			"api" : false,			"amount" : 10		  }		],		"instrument_name" : "BTC-PERPETUAL"	  },	  "channel" : "user.changes.future.BTC.raw"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`
)

func TestChangeUpdatesPushData(t *testing.T) {
	t.Parallel()
	if err := d.wsHandleData([]byte(changeUpdatesPushDataJSON)); err != nil {
		t.Error(err)
	}
	if err := d.wsHandleData([]byte(currencyChangesUpdatesPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var (
	userOrdersRawInstrumentPushDataJSON            = `{	"params" : {"data" : {"time_in_force" : "good_til_cancelled","replaced" : false,		"reduce_only" : false,		"profit_loss" : 0,		"price" : 10502.52,		"post_only" : false,		"original_order_type" : "market",		"order_type" : "limit",		"order_state" : "open",		"order_id" : "5",		"max_show" : 200,		"last_update_timestamp" : 1581507423789,		"label" : "",		"is_liquidation" : false,		"instrument_name" : "BTC-PERPETUAL",		"filled_amount" : 0,		"direction" : "buy",		"creation_timestamp" : 1581507423789,		"commission" : 0,		"average_price" : 0,		"api" : false,		"amount" : 200	  },	  "channel" : "user.orders.BTC-PERPETUAL.raw"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`
	userOrdersByInstrumentWithIntervalPushDataJSON = `{	"params" : {"data" : [{"time_in_force" : "good_til_cancelled","replaced" : false,		  "reduce_only" : false,		  "profit_loss" : 0,		  "price" : 10460.43,		  "post_only" : false,		  "original_order_type" : "market",		  "order_type" : "limit",		  "order_state" : "open",		  "order_id" : "4",		  "max_show" : 200,		  "last_update_timestamp" : 1581507159533,		  "label" : "",		  "is_liquidation" : false,		  "instrument_name" : "BTC-PERPETUAL",		  "filled_amount" : 0,		  "direction" : "buy",		  "creation_timestamp" : 1581507159533,		  "commission" : 0,		  "average_price" : 0,		  "api" : false,		  "amount" : 200		}	  ],	  "channel" : "user.orders.BTC-PERPETUAL.100ms"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`
	userOrderByCurrencyRawPushDataJSON             = `{	"params" : {"data" : {"time_in_force" : "good_til_cancelled",		"replaced" : false,"reduce_only" : false,		"profit_loss" : 0,		"price" : 10542.68,		"post_only" : false,		"original_order_type" : "market",		"order_type" : "limit",		"order_state" : "open",		"order_id" : "6",		"max_show" : 200,		"last_update_timestamp" : 1581507583024,		"label" : "",		"is_liquidation" : false,		"instrument_name" : "BTC-PERPETUAL",		"filled_amount" : 0,		"direction" : "buy",		"creation_timestamp" : 1581507583024,		"commission" : 0,		"average_price" : 0,		"api" : false,		"amount" : 200	  },	  "channel" : "user.orders.any.any.raw"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`
	userOrderByCurrencyWithIntervalPushDataJSON    = `{"params" : {"data" : [{"time_in_force" : "good_til_cancelled","reduce_only" : false,		  "profit_loss" : 0,		  "price" : 3928.5,		  "post_only" : false,		  "order_type" : "limit",		  "order_state" : "open",		  "order_id" : "476137",		  "max_show" : 120,		  "last_update_timestamp" : 1550826337209,		  "label" : "",		  "is_liquidation" : false,		  "instrument_name" : "BTC-PERPETUAL",		  "filled_amount" : 0,		  "direction" : "buy",		  "creation_timestamp" : 1550826337209,		  "commission" : 0,		  "average_price" : 0,		  "api" : false,		  "amount" : 120		}	  ],	  "channel" : "user.orders.future.BTC.100ms"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`
)

func TestUserOrdersPushData(t *testing.T) {
	t.Parallel()
	if err := d.wsHandleData([]byte(userOrdersRawInstrumentPushDataJSON)); err != nil {
		t.Error(err)
	}
	if err := d.wsHandleData([]byte(userOrdersByInstrumentWithIntervalPushDataJSON)); err != nil {
		t.Error(err)
	}
	if err := d.wsHandleData([]byte(userOrderByCurrencyRawPushDataJSON)); err != nil {
		t.Error(err)
	}
	if err := d.wsHandleData([]byte(userOrderByCurrencyWithIntervalPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var userPortfolioPushDataJSON = `{	"params" : {	  "data" : {		"total_pl" : 0.00000425,		"session_upl" : 0.00000425,		"session_rpl" : -2e-8,		"projected_maintenance_margin" : 0.00009141,		"projected_initial_margin" : 0.00012542,		"projected_delta_total" : 0.0043,		"portfolio_margining_enabled" : false,		"options_vega" : 0,		"options_value" : 0,		"options_theta" : 0,		"options_session_upl" : 0,		"options_session_rpl" : 0,		"options_pl" : 0,		"options_gamma" : 0,		"options_delta" : 0,		"margin_balance" : 0.2340038,		"maintenance_margin" : 0.00009141,		"initial_margin" : 0.00012542,		"futures_session_upl" : 0.00000425,		"futures_session_rpl" : -2e-8,		"futures_pl" : 0.00000425,		"estimated_liquidation_ratio" : 0.01822795,		"equity" : 0.2340038,		"delta_total" : 0.0043,		"currency" : "BTC",		"balance" : 0.23399957,		"available_withdrawal_funds" : 0.23387415,		"available_funds" : 0.23387838	  },	  "channel" : "user.portfolio.btc"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`

func TestUserPortfolioPushData(t *testing.T) {
	t.Parallel()
	if err := d.wsHandleData([]byte(userPortfolioPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var (
	userTradesPushDataJSON             = `{"params" : {"data" : [{"trade_seq" :30289432,"trade_id":"48079254","timestamp":1590484156350,		  "tick_direction" : 0,		  "state" : "filled",		  "self_trade" : false,		  "reduce_only" : false,		  "price" : 8954,		  "post_only" : false,		  "order_type" : "market",		  "order_id" : "4008965646",		  "matching_id" : null,		  "mark_price" : 8952.86,		  "liquidity" : "T",		  "instrument_name" : "BTC-PERPETUAL",		  "index_price" : 8956.73,		  "fee_currency" : "BTC",		  "fee" : 0.00000168,		  "direction" : "sell",		  "amount" : 20		},		{		  "trade_seq" : 30289433,		  "trade_id" : "48079255",		  "timestamp" : 1590484156350,		  "tick_direction" : 1,		  "state" : "filled",		  "self_trade" : false,		  "reduce_only" : false,		  "price" : 8954,		  "post_only" : false,		  "order_type" : "market",		  "order_id" : "4008965646",		  "matching_id" : null,		  "mark_price" : 8952.86,"liquidity" : "T","instrument_name" : "BTC-PERPETUAL","index_price" : 8956.73,"fee_currency" : "BTC","fee" : 0.00000168,"direction" : "sell","amount" : 20	}],"channel" : "user.trades.BTC-PERPETUAL.raw"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`
	userTradesWithCurrencyPushDataJSON = `{"params" : {"data" : [{"trade_seq" :74405,	"trade_id":"48079262","timestamp":1590484255886,		  "tick_direction" : 2,		  "state" : "filled",		  "self_trade" : false,		  "reduce_only" : false,		  "price" : 8947,		  "post_only" : false,		  "order_type" : "limit",		  "order_id" : "4008978075",		  "matching_id" : null,		  "mark_price" : 8970.03,		  "liquidity" : "T",		  "instrument_name" : "BTC-25SEP20",		  "index_price" : 8953.53,		  "fee_currency" : "BTC",		  "fee" : 0.00049961,		  "direction" : "sell",		  "amount" : 8940		}	  ],	  "channel" : "user.trades.future.BTC.100ms"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`
)

func TestUserTradesPushData(t *testing.T) {
	t.Parallel()
	if err := d.wsHandleData([]byte(userTradesPushDataJSON)); err != nil {
		t.Error(err)
	}
	if err := d.wsHandleData([]byte(userTradesWithCurrencyPushDataJSON)); err != nil {
		t.Error(err)
	}
}

func TestQuoteTickerPushData(t *testing.T) {
	t.Parallel()
	if err := d.wsHandleData([]byte(quoteTickerPushDataJSON)); err != nil {
		t.Error(err)
	}
}

func setupWs() {
	if !d.Websocket.IsEnabled() {
		return
	}
	if !areTestAPIKeysSet() {
		d.Websocket.SetCanUseAuthenticatedEndpoints(false)
	}
	err := d.WsConnect()
	if err != nil {
		log.Fatal(err)
	}
}

func TestGenerateDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	_, err := d.GenerateDefaultSubscriptions()
	if err != nil {
		t.Error(err)
	}
}

func TestFetchTicker(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString(btcPerpInstrument)
	if err != nil {
		t.Skip(err)
	}
	if _, err := d.FetchTicker(context.Background(), pair, asset.Futures); err != nil {
		t.Error(err)
	}
}

func TestFetchOrderbook(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString(btcPerpInstrument)
	if err != nil {
		t.Skip(err)
	}
	if _, err := d.FetchOrderbook(context.Background(), pair, asset.Futures); err != nil {
		t.Error(err)
	}
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	if _, err := d.UpdateAccountInfo(context.Background(), asset.Futures); err != nil {
		t.Error(err)
	}
}

func TestFetchAccountInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	if _, err := d.FetchAccountInfo(context.Background(), asset.Futures); err != nil {
		t.Error(err)
	}
}

func TestGetFundingHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	if _, err := d.GetFundingHistory(context.Background()); err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	if _, err := d.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Empty); err != nil {
		t.Error(err)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString(btcPerpInstrument)
	if err != nil {
		t.Error(err)
	}
	if _, err := d.GetRecentTrades(context.Background(), pair, asset.Futures); err != nil {
		t.Error(err)
	}
}

func TestSubscribe(t *testing.T) {
	t.Parallel()
	tradablePairs, err := d.FetchTradablePairs(context.Background(), asset.OptionCombo)
	if err != nil {
		t.Fatal(err)
	}
	err = d.Subscribe([]stream.ChannelSubscription{
		{
			Channel:  chartTradesChannel,
			Asset:    asset.Futures,
			Currency: tradablePairs[0],
		}})
	if err != nil {
		t.Error(err)
	}
}

func TestWSRetrievePublicPortfolioMargins(t *testing.T) {
	info, err := d.GetInstrumentData(context.Background(), "BTC-PERPETUAL")
	if err != nil {
		t.Skip(err)
	}
	time.Sleep(4 * time.Second)
	if _, err = d.WSRetrievePublicPortfolioMargins(currencyBTC, map[string]float64{btcPerpInstrument: info.ContractSize * 2}); err != nil {
		t.Error(err)
	}
}
