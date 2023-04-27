package deribit

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
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
var futuresTradablePair, optionsTradablePair, optionComboTradablePair, futureComboTradablePair currency.Pair

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
	exchCfg.API.AuthenticatedWebsocketSupport = false
	if apiKey != "" && apiSecret != "" {
		exchCfg.API.Credentials.Key = apiKey
		exchCfg.API.Credentials.Secret = apiSecret
		d.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}
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
	err = instantiateTradablePairs()
	if err != nil {
		log.Fatalf("%v, generating sample tradable pairs", err)
	}
	setupWs()
	os.Exit(m.Run())
}

func TestStart(t *testing.T) {
	t.Parallel()
	err := d.Start(context.Background(), nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrNilPointer)
	}
	var testWg sync.WaitGroup
	err = d.Start(context.Background(), &testWg)
	if err != nil {
		t.Fatal(err)
	}
	testWg.Wait()
}

func areTestAPIKeysSet() bool {
	return d.ValidateAPICredentials(d.GetDefaultCredentials()) == nil
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	_, err := d.UpdateTicker(context.Background(), futuresTradablePair, asset.Futures)
	if err != nil {
		t.Error(err)
	}
	_, err = d.UpdateTicker(context.Background(), optionsTradablePair, asset.Options)
	if err != nil {
		t.Error(err)
	}
	_, err = d.UpdateTicker(context.Background(), optionComboTradablePair, asset.OptionCombo)
	if err != nil {
		t.Error(err)
	}
	_, err = d.UpdateTicker(context.Background(), futureComboTradablePair, asset.FutureCombo)
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
	_, err := d.UpdateOrderbook(context.Background(), futuresTradablePair, asset.Futures)
	if err != nil {
		t.Error(err)
	}
	_, err = d.UpdateOrderbook(context.Background(), optionsTradablePair, asset.Options)
	if err != nil {
		t.Error(err)
	}
	_, err = d.UpdateOrderbook(context.Background(), futureComboTradablePair, asset.FutureCombo)
	if err != nil {
		t.Error(err)
	}
	_, err = d.UpdateOrderbook(context.Background(), optionComboTradablePair, asset.OptionCombo)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := d.GetRecentTrades(context.Background(), futuresTradablePair, asset.Futures)
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetRecentTrades(context.Background(), optionsTradablePair, asset.Options)
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetRecentTrades(context.Background(), optionComboTradablePair, asset.OptionCombo)
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetRecentTrades(context.Background(), futureComboTradablePair, asset.FutureCombo)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := d.GetHistoricTrades(context.Background(), futuresTradablePair, asset.Futures, time.Now().Add(-time.Minute*10), time.Now())
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetHistoricTrades(context.Background(), optionsTradablePair, asset.Options, time.Now().Add(-time.Minute*10), time.Now())
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetHistoricTrades(context.Background(), futureComboTradablePair, asset.FutureCombo, time.Now().Add(-time.Minute*10), time.Now())
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetHistoricTrades(context.Background(), optionComboTradablePair, asset.OptionCombo, time.Now().Add(-time.Minute*10), time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	_, err := d.GetHistoricCandles(context.Background(), futuresTradablePair, asset.Futures, kline.FifteenMin, time.Now().Add(-time.Minute*5), time.Now())
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetHistoricCandles(context.Background(), optionsTradablePair, asset.Options, kline.FifteenMin, time.Now().Add(-time.Minute*5), time.Now())
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetHistoricCandles(context.Background(), futureComboTradablePair, asset.FutureCombo, kline.FifteenMin, time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetHistoricCandles(context.Background(), futureComboTradablePair, asset.OptionCombo, kline.FifteenMin, time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	_, err := d.GetHistoricCandlesExtended(context.Background(), futuresTradablePair, asset.Futures, kline.FifteenMin, time.Now().Add(-time.Hour*5), time.Now())
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetHistoricCandlesExtended(context.Background(), optionsTradablePair, asset.Options, kline.FifteenMin, time.Now().Add(-time.Hour*5), time.Now())
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetHistoricCandlesExtended(context.Background(), futureComboTradablePair, asset.FutureCombo, kline.FifteenMin, time.Now().Add(-time.Hour*5), time.Now())
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetHistoricCandlesExtended(context.Background(), optionComboTradablePair, asset.OptionCombo, kline.FifteenMin, time.Now().Add(-time.Hour*5), time.Now())
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("expected %v, but found %v", asset.ErrNotSupported, err)
	}
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	info, err := d.GetInstrumentData(context.Background(), futuresTradablePair.String())
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
			Pair:      futuresTradablePair,
		},
	)
	if err != nil {
		t.Error(err)
	}
	info, err = d.GetInstrumentData(context.Background(), optionsTradablePair.String())
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
			AssetType: asset.Options,
			Side:      order.Buy,
			Pair:      optionsTradablePair,
		},
	)
	if err != nil {
		t.Error(err)
	}
	info, err = d.GetInstrumentData(context.Background(), futureComboTradablePair.String())
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
			AssetType: asset.FutureCombo,
			Side:      order.Buy,
			Pair:      futureComboTradablePair,
		},
	)
	if err != nil {
		t.Error(err)
	}
	info, err = d.GetInstrumentData(context.Background(), optionComboTradablePair.String())
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
			AssetType: asset.OptionCombo,
			Side:      order.Buy,
			Pair:      optionComboTradablePair,
		},
	)
	if err != nil {
		t.Error(err)
	}
}

const markPriceHistoryPushDataJSON = `[ [1608142381229,0.5165791606037885], [1608142380231,0.5165737855432504], [1608142379227,0.5165768236356326] ]`

func TestGetMarkPriceHistory(t *testing.T) {
	t.Parallel()
	var resp []MarkPriceHistory
	err := json.Unmarshal([]byte(markPriceHistoryPushDataJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.GetMarkPriceHistory(context.Background(), optionsTradablePair.String(), time.Now().Add(-5*time.Minute), time.Now())
	if err != nil {
		t.Error(err)
	}
	if _, err := d.WSRetrieveMarkPriceHistory(futuresTradablePair.String(), time.Now().Add(-4*time.Hour), time.Now()); err != nil {
		t.Error(err)
	}
}

const bookSummaryByCurrencyJSON = `{	"volume_usd": 0,	"volume": 0,	"quote_currency": "USD",	"price_change": -11.1896349,	"open_interest": 0,	"mid_price": null,	"mark_price": 3579.73,	"low": null,	"last": null,	"instrument_name": "BTC-22FEB19",	"high": null,	"estimated_delivery_price": 3579.73,	"creation_timestamp": 1550230036440,	"bid_price": null,	"base_currency": "BTC",	"ask_price": null}`

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
	_, err := d.GetContractSize(context.Background(), futuresTradablePair.String())
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveContractSize(futuresTradablePair.String()); err != nil {
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
	_, err := d.GetFundingChartData(context.Background(), futuresTradablePair.String(), "8h")
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveFundingChartData(futuresTradablePair.String(), "8h"); err != nil {
		t.Error(err)
	}
}

func TestGetFundingRateHistory(t *testing.T) {
	t.Parallel()
	_, err := d.GetFundingRateHistory(context.Background(), futuresTradablePair.String(), time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
	_, err = d.WSRetrieveFundingRateHistory(futuresTradablePair.String(), time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestGetFundingRateValue(t *testing.T) {
	t.Parallel()
	_, err := d.GetFundingRateValue(context.Background(), futuresTradablePair.String(), time.Now().Add(-time.Hour*8), time.Now())
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetFundingRateValue(context.Background(), futuresTradablePair.String(), time.Now(), time.Now().Add(-time.Hour*8))
	if err != nil && !errors.Is(err, common.ErrStartAfterEnd) {
		t.Errorf("expected: %v, received %v", errStartTimeCannotBeAfterEndTime, err)
	}
	_, err = d.WSRetrieveFundingRateValue(futuresTradablePair.String(), time.Now(), time.Now().Add(-time.Hour*8))
	if err != nil && !errors.Is(err, common.ErrStartAfterEnd) {
		t.Errorf("expected: %v, received %v", errStartTimeCannotBeAfterEndTime, err)
	}
	if _, err = d.WSRetrieveFundingRateValue(futuresTradablePair.String(), time.Now().Add(-time.Hour*8), time.Now()); err != nil {
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
	_, err := d.GetLastSettlementsByInstrument(context.Background(), futuresTradablePair.String(), "", "", 0, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetLastSettlementsByInstrument(context.Background(), futuresTradablePair.String(), "settlement", "5", 0, time.Now().Add(-2*time.Hour))
	if err != nil {
		t.Error(err)
	}
	_, err = d.WSRetrieveLastSettlementsByInstrument(futuresTradablePair.String(), "", "", 0, time.Time{})
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveLastSettlementsByInstrument(futuresTradablePair.String(), "settlement", "5", 0, time.Now().Add(-2*time.Hour)); err != nil {
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
	_, err := d.GetLastTradesByCurrencyAndTime(context.Background(), currencyBTC, "", "", 0,
		time.Now().Add(-8*time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetLastTradesByCurrencyAndTime(context.Background(), currencyBTC, "option", "asc", 25,
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
	_, err := d.GetLastTradesByInstrumentAndTime(context.Background(), btcPerpInstrument, "", 0,
		time.Now().Add(-8*time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetLastTradesByInstrumentAndTime(context.Background(), btcPerpInstrument, "asc", 0,
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

func TestGetOrderbookByInstrumentID(t *testing.T) {
	t.Parallel()
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
	if err != nil {
		t.Error(err)
	}
	if _, err := d.WSRetrieveOrderbookByInstrumentID(comboD.InstrumentID, 50); err != nil {
		t.Error(err)
	}
}
func TestGetRequestForQuote(t *testing.T) {
	t.Parallel()
	_, err := d.GetRequestForQuote(context.Background(), currencyBTC, d.GetAssetKind(asset.Futures))
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

const getTransferResponseJSON = `{"count": 2, "data":[{"amount": 0.2, "created_timestamp": 1550579457727, "currency": "BTC", "direction": "payment", "id": 2, "other_side": "2MzyQc5Tkik61kJbEpJV5D5H9VfWHZK9Sgy", "state": "prepared", "type": "user", "updated_timestamp": 1550579457727 }, { "amount": 0.3, "created_timestamp": 1550579255800, "currency": "BTC", "direction": "payment", "id": 1, "other_side": "new_user_1_1", "state": "confirmed", "type": "subaccount", "updated_timestamp": 1550579255800 } ] }`

func TestGetTransfers(t *testing.T) {
	t.Parallel()
	var resp *TransfersData
	err := json.Unmarshal([]byte(getTransferResponseJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err = d.GetTransfers(context.Background(), currencyBTC, 0, 0)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveTransfers(currencyBTC, 0, 0); err != nil {
		t.Error(err)
	}
}

const cancelWithdrawlPushDataJSON = `{"address": "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBz", "amount": 0.5, "confirmed_timestamp": null, "created_timestamp": 1550571443070, "currency": "BTC", "fee": 0.0001, "id": 1, "priority": 0.15, "state": "cancelled", "transaction_id": null, "updated_timestamp": 1550571443070 }`

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	var resp *CancelWithdrawalData
	err := json.Unmarshal([]byte(cancelWithdrawlPushDataJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	_, err = d.CancelWithdrawal(context.Background(), currencyBTC, 123844)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSCancelWithdrawal(currencyBTC, 123844); err != nil {
		t.Error(err)
	}
}

func TestCreateDepositAddress(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	_, err := d.CreateDepositAddress(context.Background(), currencySOL)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSCreateDepositAddress(currencySOL); err != nil {
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

const getDepositPushDataJSON = `{"count": 1, "data": [ { "address": "2N35qDKDY22zmJq9eSyiAerMD4enJ1xx6ax", "amount": 5, "currency": "BTC", "received_timestamp": 1549295017670, "state": "completed", "transaction_id": "230669110fdaf0a0dbcdc079b6b8b43d5af29cc73683835b9bc6b3406c065fda", "updated_timestamp": 1549295130159 } ] }`

func TestGetDeposits(t *testing.T) {
	t.Parallel()
	var resp *DepositsData
	err := json.Unmarshal([]byte(getDepositPushDataJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err = d.GetDeposits(context.Background(), currencyBTC, 25, 0)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveDeposits(currencyBTC, 25, 0); err != nil {
		t.Error(err)
	}
}

const getWithdrawalResponseJSON = `{"count": 1, "data": [ { "address": "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBz", "amount": 0.5, "confirmed_timestamp": null, "created_timestamp": 1550571443070, "currency": "BTC", "fee": 0.0001, "id": 1, "priority": 0.15, "state": "unconfirmed", "transaction_id": null, "updated_timestamp": 1550571443070 } ] }`

func TestGetWithdrawals(t *testing.T) {
	t.Parallel()
	var resp *WithdrawalsData
	err := json.Unmarshal([]byte(getWithdrawalResponseJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err = d.GetWithdrawals(context.Background(), currencyBTC, 25, 0)
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
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSSubmitTransferToSubAccount(currencyBTC, 0.01, 13434); err != nil {
		t.Error(err)
	}
}

func TestSubmitTransferToUser(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	_, err := d.SubmitTransferToUser(context.Background(), currencyBTC, "", "13434", 0.001)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSSubmitTransferToUser(currencyBTC, "", "0x4aa0753d798d668056920094d65321a8e8913e26", 0.001); err != nil {
		t.Error(err)
	}
}

const submitWithdrawalResponseJSON = `{"address": "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBz", "amount": 0.4, "confirmed_timestamp": null, "created_timestamp": 1550574558607, "currency": "BTC", "fee": 0.0001, "id": 4, "priority": 1, "state": "unconfirmed", "transaction_id": null, "updated_timestamp": 1550574558607 }`

func TestSubmitWithdraw(t *testing.T) {
	t.Parallel()
	var resp *WithdrawData
	err := json.Unmarshal([]byte(submitWithdrawalResponseJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	_, err = d.SubmitWithdraw(context.Background(), currencyBTC, core.BitcoinDonationAddress, "", 0.001)
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
	if _, err := d.WSRetrieveAnnouncements(time.Now(), 5); err != nil {
		t.Error(err)
	}
}

func TestGetPublicPortfolioMargins(t *testing.T) {
	info, err := d.GetInstrumentData(context.Background(), "BTC-PERPETUAL")
	if err != nil {
		t.Skip(err)
	}
	if _, err = d.GetPublicPortfolioMargins(context.Background(), currencyBTC, map[string]float64{
		"BTC-PERPETUAL": info.ContractSize * 2,
	}); err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrievePublicPortfolioMargins(currencyBTC, map[string]float64{btcPerpInstrument: info.ContractSize * 2}); err != nil {
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
	if err != nil {
		t.Error(err)
	}
	if err = d.WSChangeSubAccountName(1, "new_sub"); err != nil {
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
	if err != nil {
		t.Error(err)
	}
	if err = d.WSDisableTFAForSubAccount(1); err != nil {
		t.Error(err)
	}
}

func TestEnableAffiliateProgram(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	err := d.EnableAffiliateProgram(context.Background())
	if err != nil {
		t.Error(err)
	}
	if err = d.WSEnableAffiliateProgram(); err != nil {
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
	if _, err = d.WSRetrieveAffiliateProgramInfo(); err != nil {
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
}

func TestWsRetrivePricatePortfolioMargins(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	if _, err := d.WSRetrievePrivatePortfolioMargins(currencyBTC, false, nil); err != nil {
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

const getTransactionLogResponseJSON = `{"logs": [ { "username": "TestUser", "user_seq": 6009, "user_id": 7, "type": "transfer", "trade_id": null, "timestamp": 1613659830333, "side": "-", "price": null, "position": null, "order_id": null, "interest_pl": null, "instrument_name": null, "info": { "transfer_type": "subaccount", "other_user_id": 27, "other_user": "Subaccount" }, "id": 61312, "equity": 3000.9275869, "currency": "BTC", "commission": 0, "change": -2.5, "cashflow": -2.5, "balance": 3001.22270418 } ], "continuation": 61282 }`

func TestGetTransactionLog(t *testing.T) {
	t.Parallel()
	var resp *TransactionsData
	err := json.Unmarshal([]byte(getTransactionLogResponseJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err = d.GetTransactionLog(context.Background(), currencyBTC, "trade", time.Now().Add(-24*time.Hour), time.Now(), 5, 0)
	if err != nil {
		t.Error(err)
	}
	_, err = d.WSRetrieveTransactionLog(currencyBTC, "trade", time.Now().Add(-24*time.Hour), time.Now(), 5, 0)
	if err != nil {
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
	if err != nil {
		t.Error(err)
	}
	if err = d.WSRemoveSubAccount(1); err != nil {
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
	if err != nil {
		t.Error(err)
	}
	if err = d.WSSetEmailForSubAccount(1, "wrongemail@wrongemail.com"); err != nil {
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

func TestToggleNotificationsFromSubAccount(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	err := d.ToggleNotificationsFromSubAccount(context.Background(), 1, false)
	if err != nil {
		t.Error(err)
	}
	if err = d.WSToggleNotificationsFromSubAccount(1, false); err != nil {
		t.Error(err)
	}
}

func TestTogglePortfolioMargining(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.TogglePortfolioMargining(context.Background(), 1234, false, false)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSTogglePortfolioMargining(1234, false, false); err != nil {
		t.Error(err)
	}
}

func TestToggleSubAccountLogin(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	err := d.ToggleSubAccountLogin(context.Background(), 1, false)
	if err != nil {
		t.Error(err)
	}
	if err = d.WSToggleSubAccountLogin(1, false); err != nil {
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
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSSubmitBuy(&OrderBuyAndSellParams{
		Instrument: btcPerpInstrument, OrderType: "limit",
		Label: "testOrder", TimeInForce: "",
		Trigger: "", Advanced: "",
		Amount: 30, Price: 500000,
		MaxShow: 0, TriggerPrice: 0,
		PostOnly: false, RejectPostOnly: false,
		ReduceOnly: false, MMP: false}); err != nil {
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
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSSubmitSell(&OrderBuyAndSellParams{
		Instrument: btcPerpInstrument, OrderType: "limit",
		Label: "testOrder", TimeInForce: "",
		Trigger: "", Advanced: "", Amount: info.ContractSize * 3,
		Price: 500000, MaxShow: 0, TriggerPrice: 0, PostOnly: false,
		RejectPostOnly: false, ReduceOnly: false, MMP: false}); err != nil {
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
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSEditOrderByLabel(&OrderBuyAndSellParams{Label: "incorrectUserLabel", Instrument: btcPerpInstrument,
		Advanced: "", Amount: 1, Price: 30000, TriggerPrice: 0, PostOnly: false, ReduceOnly: false, RejectPostOnly: false, MMP: false}); err != nil {
		t.Error(err)
	}
}

const submitCancelResponseJSON = `{"triggered": false, "trigger": "index_price", "time_in_force": "good_til_cancelled", "trigger_price": 144.73, "reduce_only": false, "profit_loss": 0, "price": "market_price", "post_only": false, "order_type": "stop_market", "order_state": "untriggered", "order_id": "ETH-SLIS-12", "max_show": 5, "last_update_timestamp": 1550575961291, "label": "", "is_liquidation": false, "instrument_name": "ETH-PERPETUAL", "direction": "sell", "creation_timestamp": 1550575961291, "api": false, "amount": 5 }`

func TestSubmitCancel(t *testing.T) {
	t.Parallel()
	var resp *PrivateCancelData
	err := json.Unmarshal([]byte(submitCancelResponseJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	_, err = d.SubmitCancel(context.Background(), "incorrectID")
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSSubmitCancel("incorrectID"); err != nil {
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
	_, err := d.SubmitCancelAllByInstrument(context.Background(), btcPerpInstrument, "all", true, true)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSSubmitCancelAllByInstrument(btcPerpInstrument, "all", true, true); err != nil {
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
	_, err := d.SubmitClosePosition(context.Background(), futuresTradablePair.String(), "limit", 35000)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSSubmitClosePosition(futuresTradablePair.String(), "limit", 35000); err != nil {
		t.Error(err)
	}
}

func TestGetMargins(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetMargins(context.Background(), futuresTradablePair.String(), 5, 35000)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveMargins(futuresTradablePair.String(), 5, 35000); err != nil {
		t.Error(err)
	}
}

func TestGetMMPConfig(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetMMPConfig(context.Background(), currencyETH)
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveMMPConfig(currencyETH); err != nil {
		t.Error(err)
	}
}

const getOpenOrdersByCurrencyResponseJSON = `[{ "time_in_force": "good_til_cancelled", "reduce_only": false, "profit_loss": 0, "price": 0.0028, "post_only": false, "order_type": "limit", "order_state": "open", "order_id": "146062", "max_show": 10, "last_update_timestamp": 1550050597036, "label": "", "is_liquidation": false, "instrument_name": "BTC-15FEB19-3250-P", "filled_amount": 0, "direction": "buy", "creation_timestamp": 1550050597036, "commission": 0, "average_price": 0, "api": true, "amount": 10 } ]`

func TestGetOpenOrdersByCurrency(t *testing.T) {
	t.Parallel()
	var resp []OrderData
	err := json.Unmarshal([]byte(getOpenOrdersByCurrencyResponseJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err = d.GetOpenOrdersByCurrency(context.Background(), currencyBTC, "option", "all")
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
	if err != nil {
		t.Error(err)
	}
	if _, err := d.WSRetrieveOrderMarginsByID([]string{"ETH-349280", "ETH-349279", "ETH-349278"}); err != nil {
		t.Error(err)
	}
}

func TestGetOrderState(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetOrderState(context.Background(), "brokenid123")
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrievesOrderState("brokenid123"); err != nil {
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

const getUserTradesByCurrencyResponseJSON = `{"trades": [ { "underlying_price": 204.5, "trade_seq": 3, "trade_id": "ETH-2696060", "timestamp": 1590480363130, "tick_direction": 2, "state": "filled", "reduce_only": false, "price": 0.361, "post_only": false, "order_type": "limit", "order_id": "ETH-584827850", "matching_id": null, "mark_price": 0.364585, "liquidity": "T", "iv": 0, "instrument_name": "ETH-29MAY20-130-C", "index_price": 203.72, "fee_currency": "ETH", "fee": 0.002, "direction": "sell", "amount": 5 }, { "underlying_price": 204.82, "trade_seq": 3, "trade_id": "ETH-2696062", "timestamp": 1590480416119, "tick_direction": 0, "state": "filled", "reduce_only": false, "price": 0.015, "post_only": false, "order_type": "limit", "order_id": "ETH-584828229", "matching_id": null, "mark_price": 0.000596, "liquidity": "T", "iv": 352.91, "instrument_name": "ETH-29MAY20-140-P", "index_price": 204.06, "fee_currency": "ETH", "fee": 0.002, "direction": "buy", "amount": 5 } ], "has_more": true }`

func TestGetUserTradesByCurrency(t *testing.T) {
	t.Parallel()
	var resp *UserTradesData
	err := json.Unmarshal([]byte(getUserTradesByCurrencyResponseJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err = d.GetUserTradesByCurrency(context.Background(), currencyETH, "future", "", "", "asc", 0, false)
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
	_, err := d.GetUserTradesByCurrencyAndTime(context.Background(), currencyETH, "future", "default", 5, time.Now().Add(-time.Hour*10), time.Now().Add(-time.Hour*1))
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveUserTradesByCurrencyAndTime(currencyETH, "future", "default", 5, false, time.Now().Add(-time.Hour*4), time.Now()); err != nil {
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
	_, err := d.GetUserTradesByInstrumentAndTime(context.Background(), btcPerpInstrument, "asc", 10, time.Now().Add(-time.Hour), time.Now())
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
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveUserTradesByOrder("wrongOrderID", "default"); err != nil {
		t.Error(err)
	}
}

func TestResetMMP(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	err := d.ResetMMP(context.Background(), currencyBTC)
	if err != nil {
		t.Error(err)
	}
	if err = d.WSResetMMP(currencyBTC); err != nil {
		t.Error(err)
	}
}

func TestSendRequestForQuote(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	err := d.SendRequestForQuote(context.Background(), futuresTradablePair.String(), 1000, order.Buy)
	if err != nil {
		t.Error(err)
	}
	if err = d.WSSendRequestForQuote(futuresTradablePair.String(), 1000, order.Buy); err != nil {
		t.Error(err)
	}
}

func TestSetMMPConfig(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	err := d.SetMMPConfig(context.Background(), currencyBTC, kline.FiveMin, 5, 0, 0)
	if err != nil {
		t.Error(err)
	}
	if err = d.WSSetMMPConfig(currencyBTC, kline.FiveMin, 5, 0, 0); err != nil {
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

const getSettlementHistoryByInstrumentResponseJSON = `{"settlements": [ { "type": "settlement", "timestamp": 1550475692526, "session_profit_loss": 0.038358299, "profit_loss": -0.001783937, "position": -66, "mark_price": 121.67, "instrument_name": "ETH-22FEB19", "index_price": 119.8 } ], "continuation": "xY7T6cusbMBNpH9SNmKb94jXSBxUPojJEdCPL4YociHBUgAhWQvEP" }`

func TestGetSettlementHistoryByInstrument(t *testing.T) {
	t.Parallel()
	var resp *PrivateSettlementsHistoryData
	err := json.Unmarshal([]byte(getSettlementHistoryByInstrumentResponseJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	resp, err = d.GetSettlementHistoryByInstrument(context.Background(), btcPerpInstrument, "settlement", "", 10, time.Now().Add(-time.Hour))
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
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSSubmitEdit(&OrderBuyAndSellParams{
		OrderID:      "incorrectID",
		Advanced:     "",
		TriggerPrice: 0.001,
		Price:        100000,
		Amount:       123,
	}); err != nil {
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
		t.Error(err)
	}
	if len(combos) == 0 {
		t.Skip("no combo instance found for currency BTC")
	}
}

func TestGetComboDetails(t *testing.T) {
	t.Parallel()
	_, err := d.GetComboDetails(context.Background(), futureComboTradablePair.String())
	if err != nil {
		t.Error(err)
	}
	if _, err := d.WSRetrieveComboDetails(futureComboTradablePair.String()); err != nil {
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
	if err != nil {
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
	if err != nil {
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
	if err != nil {
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
	}); err != nil {
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
	_, err = d.VerifyBlockTrade(context.Background(), time.Now(), "something", "maker", "", []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: btcPerpInstrument,
			Direction:      order.Buy.Lower(),
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	})
	if err != nil {
		t.Error(err)
	}
	_, err = d.WSVerifyBlockTrade(time.Now(), "sdjkafdad", "maker", "", []BlockTradeParam{
		{
			Price:          0.777 * 28000,
			InstrumentName: btcPerpInstrument,
			Direction:      "buy",
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestInvalidateBlockTradeSignature(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	err := d.InvalidateBlockTradeSignature(context.Background(), "verified_signature_string")
	if err != nil {
		t.Error(err)
	}
	err = d.WsInvalidateBlockTradeSignature("verified_signature_string")
	if err != nil {
		t.Error(err)
	}
}

func TestExecuteBlockTrade(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	info, err := d.GetInstrumentData(context.Background(), futuresTradablePair.String())
	if err != nil {
		t.Skip(err)
	}
	_, err = d.ExecuteBlockTrade(context.Background(), time.Now(), "something", "maker", "", []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: futuresTradablePair.String(),
			Direction:      "buy",
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	})
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSExecuteBlockTrade(time.Now(), "sdjkafdad", "maker", "", []BlockTradeParam{
		{
			Price:          0.777 * 22000,
			InstrumentName: btcPerpInstrument,
			Direction:      "buy",
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	}); err != nil {
		t.Error(err)
	}
}

const getUserBlocTradeResponseJSON = `[ { "trade_seq": 37, "trade_id": "92437", "timestamp": 1565089523719, "tick_direction": 3, "state": "filled", "price": 0.0001, "order_type": "limit", "order_id": "343062", "matching_id": null, "liquidity": "T", "iv": 0, "instrument_name": "BTC-9AUG19-10250-C", "index_price": 11738, "fee_currency": "BTC", "fee": 0.00025, "direction": "sell", "block_trade_id": "61", "amount": 10 }, { "trade_seq": 25350, "trade_id": "92435", "timestamp": 1565089523719, "tick_direction": 3, "state": "filled", "price": 11590, "order_type": "limit", "order_id": "343058", "matching_id": null, "liquidity": "T", "instrument_name": "BTC-PERPETUAL", "index_price": 11737.98, "fee_currency": "BTC", "fee": 0.00000164, "direction": "buy", "block_trade_id": "61", "amount": 190 } ]`

func TestGetUserBlocTrade(t *testing.T) {
	t.Parallel()
	var resp []BlockTradeData
	err := json.Unmarshal([]byte(getUserBlocTradeResponseJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err = d.GetUserBlockTrade(context.Background(), "12345567")
	if err != nil {
		t.Error(err)
	}
	if _, err = d.WSRetrieveUserBlockTrade("12345567"); err != nil {
		t.Error(err)
	}
}

func TestGetLastBlockTradesbyCurrency(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetLastBlockTradesByCurrency(context.Background(), "SOL", "", "", 5)
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
	if err != nil {
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
	if err != nil {
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
	if _, err := d.FetchTicker(context.Background(), futuresTradablePair, asset.Futures); err != nil {
		t.Error(err)
	}
	if _, err := d.FetchTicker(context.Background(), futuresTradablePair, asset.Options); err != nil {
		t.Error(err)
	}
	if _, err := d.FetchTicker(context.Background(), futuresTradablePair, asset.OptionCombo); err != nil {
		t.Error(err)
	}
	if _, err := d.FetchTicker(context.Background(), futuresTradablePair, asset.FutureCombo); err != nil {
		t.Error(err)
	}
}

func TestFetchOrderbook(t *testing.T) {
	t.Parallel()
	if _, err := d.FetchOrderbook(context.Background(), futuresTradablePair, asset.FutureCombo); err != nil {
		t.Error(err)
	}
	if _, err := d.FetchOrderbook(context.Background(), futuresTradablePair, asset.OptionCombo); err != nil {
		t.Error(err)
	}
	if _, err := d.FetchOrderbook(context.Background(), futuresTradablePair, asset.Futures); err != nil {
		t.Error(err)
	}
	if _, err := d.FetchOrderbook(context.Background(), futuresTradablePair, asset.Options); err != nil {
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
	if _, err := d.FetchAccountInfo(context.Background(), asset.Options); err != nil {
		t.Error(err)
	}
	if _, err := d.FetchAccountInfo(context.Background(), asset.OptionCombo); err != nil {
		t.Error(err)
	}
	if _, err := d.FetchAccountInfo(context.Background(), asset.FutureCombo); err != nil {
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
	subscriptions, err := d.GenerateDefaultSubscriptions()
	if err != nil {
		t.Fatal(err)
	}
	err = d.Subscribe(subscriptions)
	if err != nil {
		t.Error(err)
	}
}

func TestWSRetrievePublicPortfolioMargins(t *testing.T) {
	t.Parallel()
	info, err := d.GetInstrumentData(context.Background(), futuresTradablePair.String())
	if err != nil {
		t.Skip(err)
	}
	time.Sleep(4 * time.Second)
	if _, err = d.WSRetrievePublicPortfolioMargins(currencyBTC, map[string]float64{btcPerpInstrument: info.ContractSize * 2}); err != nil {
		t.Error(err)
	}
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          futuresTradablePair,
		AssetType:     asset.Futures,
	}
	_, err := d.CancelAllOrders(context.Background(), orderCancellation)
	if err != nil && !errors.Is(err, errNoOrderDeleted) {
		t.Error(err)
	}
	orderCancellation.AssetType = asset.FutureCombo
	orderCancellation.Pair = futureComboTradablePair
	_, err = d.CancelAllOrders(context.Background(), orderCancellation)
	if err != nil && !errors.Is(err, errNoOrderDeleted) {
		t.Error(err)
	}
	orderCancellation.AssetType = asset.Options
	orderCancellation.Pair = optionsTradablePair
	_, err = d.CancelAllOrders(context.Background(), orderCancellation)
	if err != nil && !errors.Is(err, errNoOrderDeleted) {
		t.Error(err)
	}
	orderCancellation.AssetType = asset.OptionCombo
	orderCancellation.Pair = optionComboTradablePair
	_, err = d.CancelAllOrders(context.Background(), orderCancellation)
	if err != nil && !errors.Is(err, errNoOrderDeleted) {
		t.Error(err)
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetOrderInfo(context.Background(), "1234", futuresTradablePair, asset.Futures)
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetOrderInfo(context.Background(), "1234", futuresTradablePair, asset.FutureCombo)
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetOrderInfo(context.Background(), "1234", futuresTradablePair, asset.Options)
	if err != nil {
		t.Error(err)
	}
	_, err = d.GetOrderInfo(context.Background(), "1234", futuresTradablePair, asset.OptionCombo)
	if err != nil {
		t.Error(err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	_, err := d.GetDepositAddress(context.Background(), currency.BTC, "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	withdrawCryptoRequest := withdraw.Request{
		Exchange:    d.Name,
		Amount:      1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: "0x1nv4l1d",
			Chain:   "tetheruse",
		},
	}
	_, err := d.WithdrawCryptocurrencyFunds(context.Background(),
		&withdrawCryptoRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	var getOrdersRequest = order.GetOrdersRequest{
		Type: order.AnyType, AssetType: asset.Futures,
		Side: order.AnySide, Pairs: currency.Pairs{futuresTradablePair},
	}
	_, err := d.GetActiveOrders(context.Background(), &getOrdersRequest)
	if err != nil {
		t.Error(err)
	}
	getOrdersRequest.AssetType = asset.Options
	getOrdersRequest.Pairs = currency.Pairs{optionsTradablePair}
	_, err = d.GetActiveOrders(context.Background(), &getOrdersRequest)
	if err != nil {
		t.Error(err)
	}
	getOrdersRequest.AssetType = asset.OptionCombo
	getOrdersRequest.Pairs = currency.Pairs{optionComboTradablePair}
	_, err = d.GetActiveOrders(context.Background(), &getOrdersRequest)
	if err != nil {
		t.Error(err)
	}
	getOrdersRequest.AssetType = asset.FutureCombo
	getOrdersRequest.Pairs = currency.Pairs{futureComboTradablePair}
	_, err = d.GetActiveOrders(context.Background(), &getOrdersRequest)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(authenticationSkipMessage)
	}
	if _, err := d.GetOrderHistory(context.Background(), &order.GetOrdersRequest{
		Type: order.AnyType, AssetType: asset.Futures,
		Side: order.AnySide, Pairs: []currency.Pair{futuresTradablePair},
	}); err != nil {
		t.Error(err)
	}
	if _, err := d.GetOrderHistory(context.Background(), &order.GetOrdersRequest{
		Type: order.AnyType, AssetType: asset.Options,
		Side: order.AnySide, Pairs: []currency.Pair{optionsTradablePair},
	}); err != nil {
		t.Error(err)
	}
	if _, err := d.GetOrderHistory(context.Background(), &order.GetOrdersRequest{
		Type: order.AnyType, AssetType: asset.FutureCombo,
		Side: order.AnySide, Pairs: []currency.Pair{futureComboTradablePair},
	}); err != nil {
		t.Error(err)
	}
	if _, err := d.GetOrderHistory(context.Background(), &order.GetOrdersRequest{
		Type: order.AnyType, AssetType: asset.OptionCombo,
		Side: order.AnySide, Pairs: []currency.Pair{optionComboTradablePair},
	}); err != nil {
		t.Error(err)
	}
}

func TestGuessAssetTypeFromInstrument(t *testing.T) {
	availablePairs, err := d.GetEnabledPairs(asset.Futures)
	if err != nil {
		t.Fatal(err)
	}
	var assetType asset.Item
	for id, cp := range availablePairs {
		t.Run(strconv.Itoa(id), func(t *testing.T) {
			if assetType, err = guessAssetTypeFromInstrument(cp); assetType != asset.Futures {
				t.Errorf("expected %v, but found %v", asset.Futures, assetType)
			} else if err != nil {
				t.Error(err)
			}
		})
	}
	availablePairs, err = d.GetEnabledPairs(asset.Options)
	if err != nil {
		t.Fatal(err)
	}
	for id, cp := range availablePairs {
		t.Run(strconv.Itoa(id), func(t *testing.T) {
			if assetType, err = guessAssetTypeFromInstrument(cp); assetType != asset.Options {
				t.Errorf("expected %v, but found %v", asset.Options, assetType)
			} else if err != nil {
				t.Error(err)
			}
		})
	}
	availablePairs, err = d.GetEnabledPairs(asset.OptionCombo)
	if err != nil {
		t.Fatal(err)
	}
	for id, cp := range availablePairs {
		t.Run(strconv.Itoa(id), func(t *testing.T) {
			if assetType, err = guessAssetTypeFromInstrument(cp); assetType != asset.OptionCombo {
				t.Fatalf("expected %v, but found %v", asset.OptionCombo, assetType)
			} else if err != nil {
				t.Error(err)
			}
		})
	}
	availablePairs, err = d.GetEnabledPairs(asset.FutureCombo)
	if err != nil {
		t.Fatal(err)
	}
	for id, cp := range availablePairs {
		t.Run(strconv.Itoa(id), func(t *testing.T) {
			if assetType, err = guessAssetTypeFromInstrument(cp); assetType != asset.FutureCombo {
				t.Errorf("expected %v, but found %v", asset.FutureCombo, assetType)
			} else if err != nil {
				t.Error(err)
			}
		})
	}
	cp, err := currency.NewPairFromString("something_else")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = guessAssetTypeFromInstrument(cp); !errors.Is(err, errUnsupportedInstrumentFormat) {
		t.Errorf("expected %v, but found %v", errUnsupportedInstrumentFormat, err)
	}
}

func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	var feeBuilder = &exchange.FeeBuilder{
		Amount:              1,
		FeeType:             exchange.CryptocurrencyTradeFee,
		Pair:                futuresTradablePair,
		IsMaker:             false,
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	}
	_, err := d.GetFeeByType(context.Background(), feeBuilder)
	if err != nil {
		t.Fatal(err)
	}
	if !areTestAPIKeysSet() {
		if feeBuilder.FeeType != exchange.OfflineTradeFee {
			t.Errorf("Expected %v, received %v", exchange.OfflineTradeFee, feeBuilder.FeeType)
		}
	} else {
		if feeBuilder.FeeType != exchange.CryptocurrencyTradeFee {
			t.Errorf("Expected %v, received %v", exchange.CryptocurrencyTradeFee, feeBuilder.FeeType)
		}
	}
}

func TestCalculateTradingFee(t *testing.T) {
	t.Parallel()
	feeBuilder := &exchange.FeeBuilder{
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          currency.Pair{Base: currency.BTC, Quote: currency.USD, Delimiter: currency.DashDelimiter},
		IsMaker:       true,
		Amount:        1,
		PurchasePrice: 1000,
	}
	var result float64
	result, err := calculateTradingFee(feeBuilder)
	if err != nil {
		t.Error(err)
	} else if result != 1e-1 {
		t.Errorf("expected result %f, got %f", 1e-1, result)
	}
	// futures
	feeBuilder.Pair, err = currency.NewPairFromString("BTC-21OCT22")
	if err != nil {
		t.Fatal(err)
	}
	result, err = calculateTradingFee(feeBuilder)
	if err != nil {
		t.Error(err)
	} else if result != 0.1 {
		t.Errorf("expected 0.0001 but found %f", result)
	}
	// options
	feeBuilder.Pair, err = currency.NewPairFromString("SOL-21OCT22-20-C")
	if err != nil {
		t.Fatal(err)
	}
	feeBuilder.IsMaker = false
	result, err = calculateTradingFee(feeBuilder)
	if err != nil {
		t.Error(err)
	} else if result != 0.3 {
		t.Errorf("expected 0.3 but found %f", result)
	}
	// options
	feeBuilder.Pair, err = currency.NewPairFromString("SOL-21OCT22-20-C,SOL-21OCT22-20-P")
	if err != nil {
		t.Fatal(err)
	}
	feeBuilder.IsMaker = true
	_, err = calculateTradingFee(feeBuilder)
	if err != nil {
		t.Error(err)
	} else if result != 0.3 {
		t.Errorf("expected 0.3 but found %f", result)
	}
	// option_combo
	feeBuilder.Pair, err = currency.NewPairFromString("BTC-STRG-21OCT22-19000_21000")
	if err != nil {
		t.Fatal(err)
	}
	feeBuilder.IsMaker = false
	_, err = calculateTradingFee(feeBuilder)
	if err != nil {
		t.Error(err)
	}
	// future_combo
	feeBuilder.Pair, err = currency.NewPairFromString("SOL-FS-30DEC22_28OCT22")
	if err != nil {
		t.Fatal(err)
	}
	feeBuilder.IsMaker = false
	_, err = calculateTradingFee(feeBuilder)
	if err != nil {
		t.Error(err)
	}
	feeBuilder.Pair, err = currency.NewPairFromString("some_instrument")
	if err != nil {
		t.Fatal(err)
	}
	_, err = calculateTradingFee(feeBuilder)
	if !errors.Is(err, errUnsupportedInstrumentFormat) {
		t.Error(err)
	}
}

func TestGetTime(t *testing.T) {
	t.Parallel()
	_, err := d.GetTime(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}

func TestWrapperGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := d.GetServerTime(context.Background(), asset.Empty)
	if err != nil {
		t.Error(err)
	}
}

func instantiateTradablePairs() error {
	err := d.UpdateTradablePairs(context.Background(), false)
	if err != nil {
		return err
	}
	var tradablePair currency.Pairs
	tradablePair, err = d.GetEnabledPairs(asset.Futures)
	if err != nil {
		return err
	}
	futuresTradablePair = tradablePair[0]
	tradablePair, err = d.GetEnabledPairs(asset.Options)
	if err != nil {
		return err
	}
	optionsTradablePair = tradablePair[0]
	tradablePair, err = d.GetEnabledPairs(asset.OptionCombo)
	if err != nil {
		return err
	}
	optionComboTradablePair = tradablePair[0]
	tradablePair, err = d.GetEnabledPairs(asset.FutureCombo)
	if err != nil {
		return err
	}
	futureComboTradablePair = tradablePair[0]
	return nil
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(endpointAuthorizationToManipulate)
	}
	_, err := d.ModifyOrder(context.Background(), &order.Modify{OrderID: "1234"})
	if err != nil && !errors.Is(err, order.ErrPairIsEmpty) {
		t.Error(err)
	}
	_, err = d.ModifyOrder(context.Background(), &order.Modify{AssetType: asset.Spot})
	if err != nil && !errors.Is(err, order.ErrPairIsEmpty) {
		t.Error(err)
	}
	_, err = d.ModifyOrder(context.Background(), &order.Modify{AssetType: asset.Spot, OrderID: "1234", Pair: futuresTradablePair})
	if err != nil && !errors.Is(err, asset.ErrNotSupported) {
		t.Error(err)
	}
	_, err = d.ModifyOrder(context.Background(), &order.Modify{AssetType: asset.Futures, OrderID: "1234", Pair: futuresTradablePair})
	if err != nil && !errors.Is(err, errInvalidAmount) {
		t.Error(err)
	}
	_, err = d.ModifyOrder(context.Background(), &order.Modify{AssetType: asset.Futures, Pair: futuresTradablePair, Amount: 2})
	if err != nil && !errors.Is(err, order.ErrOrderIDNotSet) {
		t.Error(err)
	}
	_, err = d.ModifyOrder(context.Background(), &order.Modify{AssetType: asset.Futures, OrderID: "1234", Pair: futuresTradablePair, Amount: 2})
	if err != nil {
		t.Error(err)
	}
	_, err = d.ModifyOrder(context.Background(), &order.Modify{AssetType: asset.Options, OrderID: "1234", Pair: futuresTradablePair, Amount: 2})
	if err != nil {
		t.Error(err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          futuresTradablePair,
		AssetType:     asset.Futures,
	}
	err := d.CancelOrder(context.Background(), orderCancellation)
	if err != nil {
		t.Error(err)
	}
	orderCancellation.AssetType = asset.Options
	orderCancellation.Pair = optionsTradablePair
	err = d.CancelOrder(context.Background(), orderCancellation)
	if err != nil {
		t.Error(err)
	}
	orderCancellation.AssetType = asset.FutureCombo
	orderCancellation.Pair = futureComboTradablePair
	if err := d.CancelOrder(context.Background(), orderCancellation); err != nil {
		t.Error(err)
	}
	orderCancellation.AssetType = asset.OptionCombo
	orderCancellation.Pair = optionComboTradablePair
	if err := d.CancelOrder(context.Background(), orderCancellation); err != nil {
		t.Error(err)
	}
}

const userChangeInstrumentNamePushDataJSON = `{ "trades" : [ { "trade_seq" : 866638, "trade_id" : "1430914", "timestamp" : 1605780344032, "tick_direction" : 1, "state" : "filled", "reduce_only" : false, "profit_loss" : 0.00004898, "price" : 17391, "post_only" : false, "order_type" : "market", "order_id" : "3398016", "matching_id" : null, "mark_price" : 17391, "liquidity" : "T", "instrument_name" : "BTC-PERPETUAL", "index_price" : 17501.88, "fee_currency" : "BTC", "fee" : 1.6e-7, "direction" : "sell", "amount" : 10 } ], "positions" : [ { "total_profit_loss" : 1.69711368, "size_currency" : 10.646886321, "size" : 185160, "settlement_price" : 16025.83, "realized_profit_loss" : 0.012454598, "realized_funding" : 0.01235663, "open_orders_margin" : 0, "mark_price" : 17391, "maintenance_margin" : 0.234575865, "leverage" : 33, "kind" : "future", "interest_value" : 1.7362511643080387, "instrument_name" : "BTC-PERPETUAL", "initial_margin" : 0.319750953, "index_price" : 17501.88, "floating_profit_loss" : 0.906961435, "direction" : "buy", "delta" : 10.646886321, "average_price" : 15000 } ], "orders" : [ { "web" : true, "time_in_force" : "good_til_cancelled", "replaced" : false, "reduce_only" : false, "profit_loss" : 0.00009166, "price" : 15665.5, "post_only" : false, "order_type" : "market", "order_state" : "filled", "order_id" : "3398016", "max_show" : 10, "last_update_timestamp" : 1605780344032, "label" : "", "is_liquidation" : false, "instrument_name" : "BTC-PERPETUAL", "filled_amount" : 10, "direction" : "sell", "creation_timestamp" : 1605780344032, "commission" : 1.6e-7, "average_price" : 17391, "api" : false, "amount" : 10 } ], "instrument_name" : "BTC-PERPETUAL" }`

func TestVolatilityIndexUnmarshal(t *testing.T) {
	t.Parallel()
	var resp *wsChanges
	err := json.Unmarshal([]byte(userChangeInstrumentNamePushDataJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
}
