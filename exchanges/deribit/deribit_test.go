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

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
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
)

var (
	d                                                                                                            = &Deribit{}
	futuresTradablePair, optionsTradablePair, optionComboTradablePair, futureComboTradablePair, spotTradablePair currency.Pair
	fetchTradablePairChan                                                                                        chan struct{}
	tradablePairsFetchedStatusLock                                                                               = sync.Mutex{}
	tradablePairsFetched                                                                                         bool
)

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
	d.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	d.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
	fetchTradablePairChan = make(chan struct{})
	instantiateTradablePairs()
	setupWs()
	os.Exit(m.Run())
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	_, err := d.UpdateTicker(context.Background(), spotTradablePair, asset.Spot)
	assert.NoError(t, err)
	_, err = d.UpdateTicker(context.Background(), futuresTradablePair, asset.Futures)
	assert.NoError(t, err)
	sleepUntilTradablePairsUpdated()
	_, err = d.UpdateTicker(context.Background(), optionsTradablePair, asset.Options)
	assert.NoError(t, err)
	_, err = d.UpdateTicker(context.Background(), optionComboTradablePair, asset.OptionCombo)
	assert.NoError(t, err)
	_, err = d.UpdateTicker(context.Background(), futureComboTradablePair, asset.FutureCombo)
	assert.NoError(t, err)
	_, err = d.UpdateTicker(context.Background(), currency.Pair{}, asset.Margin)
	assert.Falsef(t, err != nil && !errors.Is(err, asset.ErrNotSupported), "expected: %v, received %v", asset.ErrNotSupported, err)
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	_, err := d.UpdateOrderbook(context.Background(), spotTradablePair, asset.Spot)
	assert.NoError(t, err)
	_, err = d.UpdateOrderbook(context.Background(), futuresTradablePair, asset.Futures)
	assert.NoError(t, err)
	sleepUntilTradablePairsUpdated()
	_, err = d.UpdateOrderbook(context.Background(), optionsTradablePair, asset.Options)
	assert.NoError(t, err)
	_, err = d.UpdateOrderbook(context.Background(), futureComboTradablePair, asset.FutureCombo)
	assert.NoError(t, err)
	_, err = d.UpdateOrderbook(context.Background(), optionComboTradablePair, asset.OptionCombo)
	assert.NoError(t, err)
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := d.GetHistoricTrades(context.Background(), futuresTradablePair, asset.Futures, time.Now().Add(-time.Minute*10), time.Now())
	assert.NoError(t, err)
	_, err = d.GetHistoricTrades(context.Background(), spotTradablePair, asset.Spot, time.Now().Add(-time.Minute*10), time.Now())
	assert.NoError(t, err)
}

func TestFetchRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := d.GetRecentTrades(context.Background(), futuresTradablePair, asset.Futures)
	assert.NoError(t, err)
	_, err = d.GetRecentTrades(context.Background(), spotTradablePair, asset.Spot)
	assert.NoError(t, err)
	sleepUntilTradablePairsUpdated()
	_, err = d.GetRecentTrades(context.Background(), optionsTradablePair, asset.Options)
	assert.NoError(t, err)
	_, err = d.GetRecentTrades(context.Background(), optionComboTradablePair, asset.OptionCombo)
	assert.NoError(t, err)
	_, err = d.GetRecentTrades(context.Background(), futureComboTradablePair, asset.FutureCombo)
	assert.NoError(t, err)
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	_, err := d.GetHistoricCandles(context.Background(), futuresTradablePair, asset.Futures, kline.FifteenMin, time.Now().Add(-time.Minute*5), time.Now())
	assert.NoError(t, err)
	_, err = d.GetHistoricCandles(context.Background(), spotTradablePair, asset.Spot, kline.FifteenMin, time.Now().Add(-time.Minute*5), time.Now())
	assert.NoError(t, err)
	sleepUntilTradablePairsUpdated()
	_, err = d.GetHistoricCandles(context.Background(), optionsTradablePair, asset.Options, kline.FifteenMin, time.Now().Add(-time.Minute*5), time.Now())
	assert.Truef(t, errors.Is(err, asset.ErrNotSupported), "expected %v, but found %v", asset.ErrNotSupported, err)
	_, err = d.GetHistoricCandles(context.Background(), futureComboTradablePair, asset.FutureCombo, kline.FifteenMin, time.Now().Add(-time.Hour), time.Now())
	assert.Truef(t, errors.Is(err, asset.ErrNotSupported), "expected %v, but found %v", asset.ErrNotSupported, err)
	_, err = d.GetHistoricCandles(context.Background(), optionComboTradablePair, asset.OptionCombo, kline.FifteenMin, time.Now().Add(-time.Hour), time.Now())
	assert.Falsef(t, !errors.Is(err, asset.ErrNotSupported), "expected %v, but found %v", asset.ErrNotSupported, err)
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	_, err := d.GetHistoricCandlesExtended(context.Background(), futuresTradablePair, asset.Futures, kline.FifteenMin, time.Now().Add(-time.Hour*550), time.Now().Add(-time.Hour*150))
	assert.NoError(t, err)
	_, err = d.GetHistoricCandlesExtended(context.Background(), spotTradablePair, asset.Spot, kline.FifteenMin, time.Now().Add(-time.Hour*550), time.Now().Add(-time.Hour*150))
	assert.NoError(t, err)
	sleepUntilTradablePairsUpdated()
	_, err = d.GetHistoricCandlesExtended(context.Background(), optionsTradablePair, asset.Options, kline.FifteenMin, time.Now().Add(-time.Hour*550), time.Now().Add(-time.Hour*150))
	assert.True(t, errors.Is(err, asset.ErrNotSupported), "expected %v, but found %v", asset.ErrNotSupported, err)
	_, err = d.GetHistoricCandlesExtended(context.Background(), futureComboTradablePair, asset.FutureCombo, kline.FifteenMin, time.Now().Add(-time.Hour*550), time.Now().Add(-time.Hour*150))
	assert.Truef(t, errors.Is(err, asset.ErrNotSupported), "expected %v, but found %v", asset.ErrNotSupported, err)
	_, err = d.GetHistoricCandlesExtended(context.Background(), optionComboTradablePair, asset.OptionCombo, kline.FifteenMin, time.Now().Add(-time.Hour*550), time.Now().Add(-time.Hour*150))
	assert.Truef(t, errors.Is(err, asset.ErrNotSupported),
		"expected %v, but found %v", asset.ErrNotSupported, err)
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	info, err := d.GetInstrument(context.Background(), d.formatFuturesTradablePair(futuresTradablePair))
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
	assert.NoError(t, err)
	sleepUntilTradablePairsUpdated()
	info, err = d.GetInstrument(context.Background(), optionsTradablePair.String())
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
	assert.NoError(t, err)
	info, err = d.GetInstrument(context.Background(), futureComboTradablePair.String())
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
	assert.NoError(t, err)
	info, err = d.GetInstrument(context.Background(), optionComboTradablePair.String())
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
	assert.NoError(t, err)
}

func TestGetMarkPriceHistory(t *testing.T) {
	t.Parallel()
	var resp []MarkPriceHistory
	err := json.Unmarshal([]byte(`[[1608142381229,0.5165791606037885],[1608142380231,0.5165737855432504],[1608142379227,0.5165768236356326]]`), &resp)
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.GetMarkPriceHistory(context.Background(), btcPerpInstrument, time.Now().Add(-5*time.Minute), time.Now())
	assert.NoError(t, err)
	_, err = d.WSRetrieveMarkPriceHistory(btcPerpInstrument, time.Now().Add(-4*time.Hour), time.Now())
	assert.NoError(t, err)
	sleepUntilTradablePairsUpdated()
	_, err = d.GetMarkPriceHistory(context.Background(), optionsTradablePair.String(), time.Now().Add(-5*time.Minute), time.Now())
	assert.NoError(t, err)
	_, err = d.WSRetrieveMarkPriceHistory(optionsTradablePair.String(), time.Now().Add(-4*time.Hour), time.Now())
	assert.NoError(t, err)
	_, err = d.GetMarkPriceHistory(context.Background(), spotTradablePair.String(), time.Now().Add(-5*time.Minute), time.Now())
	assert.NoError(t, err)
	_, err = d.WSRetrieveMarkPriceHistory(spotTradablePair.String(), time.Now().Add(-4*time.Hour), time.Now())
	assert.NoError(t, err)
	_, err = d.GetMarkPriceHistory(context.Background(), futureComboTradablePair.String(), time.Now().Add(-5*time.Minute), time.Now())
	assert.NoError(t, err)
	_, err = d.WSRetrieveMarkPriceHistory(futureComboTradablePair.String(), time.Now().Add(-4*time.Hour), time.Now())
	assert.NoError(t, err)
	_, err = d.GetMarkPriceHistory(context.Background(), optionComboTradablePair.String(), time.Now().Add(-5*time.Minute), time.Now())
	assert.NoError(t, err)
	_, err = d.WSRetrieveMarkPriceHistory(optionComboTradablePair.String(), time.Now().Add(-4*time.Hour), time.Now())
	assert.NoError(t, err)
}

func TestGetBookSummaryByCurrency(t *testing.T) {
	t.Parallel()
	var response BookSummaryData
	err := json.Unmarshal([]byte(`{	"volume_usd": 0,	"volume": 0,	"quote_currency": "USD",	
	"price_change": -11.1896349,	"open_interest": 0,	"mid_price": null,	"mark_price": 3579.73,	"low": null,	
	"last": null,	"instrument_name": "BTC-22FEB19",	"high": null,	"estimated_delivery_price": 3579.73,	"creation_timestamp": 1550230036440,	
	"bid_price": null,	"base_currency": "BTC",	"ask_price": null}`), &response)
	assert.NoError(t, err)
	_, err = d.GetBookSummaryByCurrency(context.Background(), currencyBTC, "")
	assert.NoError(t, err)
	_, err = d.WSRetrieveBookBySummary(currencySOL, "")
	assert.NoError(t, err)
}

func TestGetBookSummaryByInstrument(t *testing.T) {
	t.Parallel()
	_, err := d.GetBookSummaryByInstrument(context.Background(), btcPerpInstrument)
	assert.NoError(t, err)
	_, err = d.WSRetrieveBookSummaryByInstrument(btcPerpInstrument)
	assert.NoError(t, err)
	_, err = d.GetBookSummaryByInstrument(context.Background(), spotTradablePair.String())
	assert.NoError(t, err)
	_, err = d.WSRetrieveBookSummaryByInstrument(spotTradablePair.String())
	assert.NoError(t, err)
	sleepUntilTradablePairsUpdated()
	_, err = d.GetBookSummaryByInstrument(context.Background(), optionsTradablePair.String())
	assert.NoError(t, err)
	_, err = d.WSRetrieveBookSummaryByInstrument(optionsTradablePair.String())
	assert.NoError(t, err)
	_, err = d.GetBookSummaryByInstrument(context.Background(), optionComboTradablePair.String())
	assert.NoError(t, err)
	_, err = d.WSRetrieveBookSummaryByInstrument(optionComboTradablePair.String())
	assert.NoError(t, err)
	_, err = d.GetBookSummaryByInstrument(context.Background(), futureComboTradablePair.String())
	assert.NoError(t, err)
	_, err = d.WSRetrieveBookSummaryByInstrument(futureComboTradablePair.String())
	assert.NoError(t, err)
}

func TestGetContractSize(t *testing.T) {
	t.Parallel()
	_, err := d.GetContractSize(context.Background(), btcPerpInstrument)
	assert.NoError(t, err)
	_, err = d.WSRetrieveContractSize(btcPerpInstrument)
	assert.NoError(t, err)
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	_, err := d.GetCurrencies(context.Background())
	assert.NoError(t, err)
	_, err = d.WSRetrieveCurrencies()
	assert.NoError(t, err)
}

func TestGetDeliveryPrices(t *testing.T) {
	t.Parallel()
	_, err := d.GetDeliveryPrices(context.Background(), "btc_usd", 0, 5)
	assert.NoError(t, err)
	_, err = d.WSRetrieveDeliveryPrices("btc_usd", 0, 5)
	assert.NoError(t, err)
}

func TestGetFundingChartData(t *testing.T) {
	t.Parallel()
	// only for perpetual instruments
	_, err := d.GetFundingChartData(context.Background(), btcPerpInstrument, "8h")
	assert.NoError(t, err)
	_, err = d.WSRetrieveFundingChartData(btcPerpInstrument, "8h")
	assert.NoError(t, err)
}

func TestGetFundingRateHistory(t *testing.T) {
	t.Parallel()
	_, err := d.GetFundingRateHistory(context.Background(), btcPerpInstrument, time.Now().Add(-time.Hour), time.Now())
	assert.NoError(t, err)
	_, err = d.WSRetrieveFundingRateHistory(btcPerpInstrument, time.Now().Add(-time.Hour), time.Now())
	assert.NoError(t, err)
}

func TestGetFundingRateValue(t *testing.T) {
	t.Parallel()
	_, err := d.GetFundingRateValue(context.Background(), btcPerpInstrument, time.Now().Add(-time.Hour*8), time.Now())
	assert.NoError(t, err)
	_, err = d.GetFundingRateValue(context.Background(), btcPerpInstrument, time.Now(), time.Now().Add(-time.Hour*8))
	assert.Falsef(t, err != nil && !errors.Is(err, common.ErrStartAfterEnd), "expected: %v, received %v", errStartTimeCannotBeAfterEndTime, err)
	_, err = d.WSRetrieveFundingRateValue(btcPerpInstrument, time.Now(), time.Now().Add(-time.Hour*8))
	assert.False(t, err != nil && !errors.Is(err, common.ErrStartAfterEnd), "expected: %v, received %v", errStartTimeCannotBeAfterEndTime, err)
	_, err = d.WSRetrieveFundingRateValue(btcPerpInstrument, time.Now().Add(-time.Hour*8), time.Now())
	assert.NoError(t, err)
}

func TestGetHistoricalVolatility(t *testing.T) {
	t.Parallel()
	_, err := d.GetHistoricalVolatility(context.Background(), currencyBTC)
	assert.NoError(t, err)
	_, err = d.WSRetrieveHistoricalVolatility(currencySOL)
	assert.NoError(t, err)
}

func TestGetCurrencyIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := d.GetCurrencyIndexPrice(context.Background(), currencyBTC)
	assert.NoError(t, err)
	_, err = d.WSRetrieveCurrencyIndexPrice(currencyBTC)
	assert.NoError(t, err)
}

func TestGetIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := d.GetIndexPrice(context.Background(), "ada_usd")
	assert.NoError(t, err)
	_, err = d.WSRetrieveIndexPrice("ada_usd")
	assert.NoError(t, err)
}

func TestGetIndexPriceNames(t *testing.T) {
	t.Parallel()
	_, err := d.GetIndexPriceNames(context.Background())
	assert.NoError(t, err)
	_, err = d.WSRetrieveIndexPriceNames()
	assert.NoError(t, err)
}

func TestGetInstrumentData(t *testing.T) {
	t.Parallel()
	_, err := d.GetInstrument(context.Background(), btcPerpInstrument)
	assert.NoError(t, err)
	_, err = d.WSRetrieveInstrumentData(btcPerpInstrument)
	assert.NoError(t, err)
	_, err = d.GetInstrument(context.Background(), spotTradablePair.String())
	assert.NoError(t, err)
	_, err = d.WSRetrieveInstrumentData(spotTradablePair.String())
	assert.NoError(t, err)
	sleepUntilTradablePairsUpdated()
	_, err = d.GetInstrument(context.Background(), optionsTradablePair.String())
	assert.NoError(t, err)
	_, err = d.WSRetrieveInstrumentData(optionsTradablePair.String())
	assert.NoError(t, err)
	_, err = d.GetInstrument(context.Background(), optionComboTradablePair.String())
	assert.NoError(t, err)
	_, err = d.WSRetrieveInstrumentData(optionComboTradablePair.String())
	assert.NoError(t, err)
	_, err = d.GetInstrument(context.Background(), futureComboTradablePair.String())
	assert.NoError(t, err)
	_, err = d.WSRetrieveInstrumentData(futureComboTradablePair.String())
	assert.NoError(t, err)
}

func TestGetInstrumentsData(t *testing.T) {
	t.Parallel()
	_, err := d.GetInstruments(context.Background(), currencyBTC, "", false)
	assert.NoError(t, err)
	_, err = d.WSRetrieveInstrumentsData(currencyBTC, "", false)
	assert.NoError(t, err)
}

func TestGetLastSettlementsByCurrency(t *testing.T) {
	t.Parallel()
	_, err := d.GetLastSettlementsByCurrency(context.Background(), currencyBTC, "delivery", "5", 0, time.Now().Add(-time.Hour))
	assert.NoError(t, err)
	_, err = d.WSRetrieveLastSettlementsByCurrency(currencyBTC, "delivery", "5", 0, time.Now().Add(-time.Hour))
	assert.NoError(t, err)
}

func TestGetLastSettlementsByInstrument(t *testing.T) {
	t.Parallel()
	_, err := d.GetLastSettlementsByInstrument(context.Background(), d.formatFuturesTradablePair(futuresTradablePair), "settlement", "5", 0, time.Now().Add(-2*time.Hour))
	assert.NoError(t, err)
	_, err = d.WSRetrieveLastSettlementsByInstrument(d.formatFuturesTradablePair(futuresTradablePair), "settlement", "5", 0, time.Now().Add(-2*time.Hour))
	assert.NoError(t, err)
}

func TestGetLastTradesByCurrency(t *testing.T) {
	t.Parallel()
	_, err := d.GetLastTradesByCurrency(context.Background(), currencyBTC, "option", "36798", "36799", "asc", 0, true)
	assert.NoError(t, err)
	_, err = d.WSRetrieveLastTradesByCurrency(currencyBTC, "option", "36798", "36799", "asc", 0, true)
	assert.NoError(t, err)
}

func TestGetLastTradesByCurrencyAndTime(t *testing.T) {
	t.Parallel()
	_, err := d.GetLastTradesByCurrencyAndTime(context.Background(), currencyBTC, "", "", 0,
		time.Now().Add(-8*time.Hour), time.Now())
	assert.NoError(t, err)
	_, err = d.GetLastTradesByCurrencyAndTime(context.Background(), currencyBTC, "option", "asc", 25,
		time.Now().Add(-8*time.Hour), time.Now())
	assert.NoError(t, err)
	_, err = d.WSRetrieveLastTradesByCurrencyAndTime(currencyBTC, "", "", 0, false, time.Now().Add(-8*time.Hour), time.Now())
	assert.NoError(t, err)
	_, err = d.WSRetrieveLastTradesByCurrencyAndTime(currencyBTC, "option", "asc", 25, false, time.Now().Add(-8*time.Hour), time.Now())
	assert.NoError(t, err)
}

func TestGetLastTradesByInstrument(t *testing.T) {
	t.Parallel()
	_, err := d.GetLastTradesByInstrument(context.Background(), btcPerpInstrument, "", "", "", 0, false)
	assert.NoError(t, err)
	_, err = d.GetLastTradesByInstrument(context.Background(), btcPerpInstrument, "30500", "31500", "desc", 0, true)
	assert.NoError(t, err)
	_, err = d.WSRetrieveLastTradesByInstrument(btcPerpInstrument, "", "", "", 0, false)
	assert.NoError(t, err)
	_, err = d.WSRetrieveLastTradesByInstrument(btcPerpInstrument, "30500", "31500", "desc", 0, true)
	assert.NoError(t, err)
	_, err = d.GetLastTradesByInstrument(context.Background(), spotTradablePair.String(), "30500", "31500", "desc", 0, true)
	assert.NoError(t, err)
	_, err = d.WSRetrieveLastTradesByInstrument(spotTradablePair.String(), "", "", "", 0, false)
	assert.NoError(t, err)
	sleepUntilTradablePairsUpdated()
	_, err = d.GetLastTradesByInstrument(context.Background(), optionsTradablePair.String(), "30500", "31500", "desc", 0, true)
	assert.NoError(t, err)
	_, err = d.WSRetrieveLastTradesByInstrument(optionsTradablePair.String(), "", "", "", 0, false)
	assert.NoError(t, err)
	_, err = d.GetLastTradesByInstrument(context.Background(), optionComboTradablePair.String(), "30500", "31500", "desc", 0, true)
	assert.NoError(t, err)
	_, err = d.WSRetrieveLastTradesByInstrument(optionComboTradablePair.String(), "", "", "", 0, false)
	assert.NoError(t, err)
	_, err = d.GetLastTradesByInstrument(context.Background(), futureComboTradablePair.String(), "30500", "31500", "desc", 0, true)
	assert.NoError(t, err)
	_, err = d.WSRetrieveLastTradesByInstrument(futureComboTradablePair.String(), "", "", "", 0, false)
	assert.NoError(t, err)
}

func TestGetLastTradesByInstrumentAndTime(t *testing.T) {
	t.Parallel()
	_, err := d.GetLastTradesByInstrumentAndTime(context.Background(), btcPerpInstrument, "", 0,
		time.Now().Add(-8*time.Hour), time.Now())
	assert.NoError(t, err)
	_, err = d.GetLastTradesByInstrumentAndTime(context.Background(), btcPerpInstrument, "asc", 0,
		time.Now().Add(-8*time.Hour), time.Now())
	assert.NoError(t, err)
	_, err = d.WSRetrieveLastTradesByInstrumentAndTime(btcPerpInstrument, "", 0, false, time.Now().Add(-8*time.Hour), time.Now())
	assert.NoError(t, err)
	_, err = d.WSRetrieveLastTradesByInstrumentAndTime(btcPerpInstrument, "asc", 0, false, time.Now().Add(-8*time.Hour), time.Now())
	assert.NoError(t, err)
	_, err = d.GetLastTradesByInstrumentAndTime(context.Background(), spotTradablePair.String(), "", 0, time.Now().Add(-8*time.Hour), time.Now())
	assert.NoError(t, err)
	_, err = d.WSRetrieveLastTradesByInstrumentAndTime(spotTradablePair.String(), "asc", 0, false, time.Now().Add(-8*time.Hour), time.Now())
	assert.NoError(t, err)
	sleepUntilTradablePairsUpdated()
	_, err = d.GetLastTradesByInstrumentAndTime(context.Background(), optionsTradablePair.String(), "", 0, time.Now().Add(-8*time.Hour), time.Now())
	assert.NoError(t, err)
	_, err = d.WSRetrieveLastTradesByInstrumentAndTime(optionsTradablePair.String(), "asc", 0, false, time.Now().Add(-8*time.Hour), time.Now())
	assert.NoError(t, err)
	_, err = d.GetLastTradesByInstrumentAndTime(context.Background(), optionsTradablePair.String(), "", 0, time.Now().Add(-8*time.Hour), time.Now())
	assert.NoError(t, err)
	_, err = d.WSRetrieveLastTradesByInstrumentAndTime(optionsTradablePair.String(), "asc", 0, false, time.Now().Add(-8*time.Hour), time.Now())
	assert.NoError(t, err)
	_, err = d.GetLastTradesByInstrumentAndTime(context.Background(), optionComboTradablePair.String(), "", 0, time.Now().Add(-8*time.Hour), time.Now())
	assert.NoError(t, err)
	_, err = d.WSRetrieveLastTradesByInstrumentAndTime(optionComboTradablePair.String(), "asc", 0, false, time.Now().Add(-8*time.Hour), time.Now())
	assert.NoError(t, err)
	_, err = d.GetLastTradesByInstrumentAndTime(context.Background(), futureComboTradablePair.String(), "", 0, time.Now().Add(-8*time.Hour), time.Now())
	assert.NoError(t, err)
	_, err = d.WSRetrieveLastTradesByInstrumentAndTime(futureComboTradablePair.String(), "asc", 0, false, time.Now().Add(-8*time.Hour), time.Now())
	assert.NoError(t, err)
}

func TestGetOrderbookData(t *testing.T) {
	t.Parallel()
	_, err := d.GetOrderbook(context.Background(), btcPerpInstrument, 0)
	assert.NoError(t, err)
	_, err = d.WSRetrieveOrderbookData(btcPerpInstrument, 0)
	assert.NoError(t, err)
	_, err = d.GetOrderbook(context.Background(), spotTradablePair.String(), 0)
	assert.NoError(t, err)
	_, err = d.GetOrderbook(context.Background(), spotTradablePair.String(), 0)
	assert.NoError(t, err)
	sleepUntilTradablePairsUpdated()
	_, err = d.GetOrderbook(context.Background(), futureComboTradablePair.String(), 0)
	assert.NoError(t, err)
	_, err = d.GetOrderbook(context.Background(), optionComboTradablePair.String(), 0)
	assert.NoError(t, err)
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
	assert.NoError(t, err)
	_, err = d.GetOrderbookByInstrumentID(context.Background(), comboD.InstrumentID, 50)
	assert.NoError(t, err)
	_, err = d.WSRetrieveOrderbookByInstrumentID(comboD.InstrumentID, 50)
	assert.NoError(t, err)
}

func TestGetSupportedIndexNames(t *testing.T) {
	t.Parallel()
	_, err := d.GetSupportedIndexNames(context.Background(), "derivative")
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.WsRetrieveSupportedIndexNames("derivative")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetRequestForQuote(t *testing.T) {
	t.Parallel()
	_, err := d.GetRequestForQuote(context.Background(), currencyBTC, d.GetAssetKind(asset.Futures))
	assert.NoError(t, err)
	_, err = d.WSRetrieveRequestForQuote(currencyBTC, d.GetAssetKind(asset.Futures))
	assert.NoError(t, err)
}

func TestGetTradeVolumes(t *testing.T) {
	t.Parallel()
	_, err := d.GetTradeVolumes(context.Background(), false)
	assert.NoError(t, err)
	_, err = d.WSRetrieveTradeVolumes(false)
	assert.NoError(t, err)
}

func TestGetTradingViewChartData(t *testing.T) {
	t.Parallel()
	_, err := d.GetTradingViewChart(context.Background(), btcPerpInstrument, "60", time.Now().Add(-time.Hour), time.Now())
	assert.NoError(t, err)
	_, err = d.WSRetrievesTradingViewChartData(btcPerpInstrument, "60", time.Now().Add(-time.Hour), time.Now())
	assert.NoError(t, err)
	_, err = d.WSRetrievesTradingViewChartData(spotTradablePair.String(), "60", time.Now().Add(-time.Hour), time.Now())
	assert.NoError(t, err)
	_, err = d.GetTradingViewChart(context.Background(), spotTradablePair.String(), "60", time.Now().Add(-time.Hour), time.Now())
	assert.NoError(t, err)
}

func TestGetVolatilityIndexData(t *testing.T) {
	t.Parallel()
	_, err := d.GetVolatilityIndex(context.Background(), currencyBTC, "60", time.Now().Add(-time.Hour), time.Now())
	assert.NoError(t, err)
	_, err = d.WSRetrieveVolatilityIndexData(currencyBTC, "60", time.Now().Add(-time.Hour), time.Now())
	assert.NoError(t, err)
}

func TestGetPublicTicker(t *testing.T) {
	t.Parallel()
	_, err := d.GetPublicTicker(context.Background(), btcPerpInstrument)
	assert.NoError(t, err)
	_, err = d.WSRetrievePublicTicker(btcPerpInstrument)
	assert.NoError(t, err)
}

func TestGetAccountSummary(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetAccountSummary(context.Background(), currencyBTC, false)
	assert.NoError(t, err)
	_, err = d.WSRetrieveAccountSummary(currencyBTC, false)
	assert.NoError(t, err)
}

func TestCancelTransferByID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err := d.CancelTransferByID(context.Background(), currencyBTC, "", 23487)
	assert.NoError(t, err)
	_, err = d.WSCancelTransferByID(currencyBTC, "", 23487)
	assert.NoError(t, err)
}

const getTransferResponseJSON = `{"count": 2, "data":[{"amount": 0.2, "created_timestamp": 1550579457727, "currency": "BTC", "direction": "payment", "id": 2, "other_side": "2MzyQc5Tkik61kJbEpJV5D5H9VfWHZK9Sgy", "state": "prepared", "type": "user", "updated_timestamp": 1550579457727 }, { "amount": 0.3, "created_timestamp": 1550579255800, "currency": "BTC", "direction": "payment", "id": 1, "other_side": "new_user_1_1", "state": "confirmed", "type": "subaccount", "updated_timestamp": 1550579255800 } ] }`

func TestGetTransfers(t *testing.T) {
	t.Parallel()
	var resp *TransfersData
	err := json.Unmarshal([]byte(getTransferResponseJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err = d.GetTransfers(context.Background(), currencyBTC, 0, 0)
	assert.NoError(t, err)
	_, err = d.WSRetrieveTransfers(currencyBTC, 0, 0)
	assert.NoError(t, err)
}

const cancelWithdrawlPushDataJSON = `{"address": "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBz", "amount": 0.5, "confirmed_timestamp": null, "created_timestamp": 1550571443070, "currency": "BTC", "fee": 0.0001, "id": 1, "priority": 0.15, "state": "cancelled", "transaction_id": null, "updated_timestamp": 1550571443070 }`

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	var resp *CancelWithdrawalData
	err := json.Unmarshal([]byte(cancelWithdrawlPushDataJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err = d.CancelWithdrawal(context.Background(), currencyBTC, 123844)
	assert.NoError(t, err)
	_, err = d.WSCancelWithdrawal(currencyBTC, 123844)
	assert.NoError(t, err)
}

func TestCreateDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err := d.CreateDepositAddress(context.Background(), currencySOL)
	assert.NoError(t, err)
	_, err = d.WSCreateDepositAddress(currencySOL)
	assert.NoError(t, err)
}

func TestGetCurrentDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetCurrentDepositAddress(context.Background(), currencyETH)
	assert.NoError(t, err)
	_, err = d.WSRetrieveCurrentDepositAddress(currencyETH)
	assert.NoError(t, err)
}

const getDepositPushDataJSON = `{"count": 1, "data": [ { "address": "2N35qDKDY22zmJq9eSyiAerMD4enJ1xx6ax", "amount": 5, "currency": "BTC", "received_timestamp": 1549295017670, "state": "completed", "transaction_id": "230669110fdaf0a0dbcdc079b6b8b43d5af29cc73683835b9bc6b3406c065fda", "updated_timestamp": 1549295130159 } ] }`

func TestGetDeposits(t *testing.T) {
	t.Parallel()
	var resp *DepositsData
	err := json.Unmarshal([]byte(getDepositPushDataJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err = d.GetDeposits(context.Background(), currencyBTC, 25, 0)
	assert.NoError(t, err)
	_, err = d.WSRetrieveDeposits(currencyBTC, 25, 0)
	assert.NoError(t, err)
}

const getWithdrawalResponseJSON = `{"count": 1, "data": [ { "address": "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBz", "amount": 0.5, "confirmed_timestamp": null, "created_timestamp": 1550571443070, "currency": "BTC", "fee": 0.0001, "id": 1, "priority": 0.15, "state": "unconfirmed", "transaction_id": null, "updated_timestamp": 1550571443070 } ] }`

func TestGetWithdrawals(t *testing.T) {
	t.Parallel()
	var resp *WithdrawalsData
	err := json.Unmarshal([]byte(getWithdrawalResponseJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err = d.GetWithdrawals(context.Background(), currencyBTC, 25, 0)
	assert.NoError(t, err)
	_, err = d.WSRetrieveWithdrawals(currencyBTC, 25, 0)
	assert.NoError(t, err)
}

func TestSubmitTransferBetweenSubAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err := d.SubmitTransferBetweenSubAccounts(context.Background(), currency.EURR, 12345, 4, "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.WsSubmitTransferBetweenSubAccounts(currency.EURR, 12345, 2, "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestSubmitTransferToSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err := d.SubmitTransferToSubAccount(context.Background(), currencyBTC, 0.01, 13434)
	assert.NoError(t, err)
	_, err = d.WSSubmitTransferToSubAccount(currencyBTC, 0.01, 13434)
	assert.NoError(t, err)
}

func TestSubmitTransferToUser(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err := d.SubmitTransferToUser(context.Background(), currencyBTC, "", "13434", 0.001)
	assert.NoError(t, err)
	_, err = d.WSSubmitTransferToUser(currencyBTC, "", "0x4aa0753d798d668056920094d65321a8e8913e26", 0.001)
	assert.NoError(t, err)
}

const submitWithdrawalResponseJSON = `{"address": "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBz", "amount": 0.4, "confirmed_timestamp": null, "created_timestamp": 1550574558607, "currency": "BTC", "fee": 0.0001, "id": 4, "priority": 1, "state": "unconfirmed", "transaction_id": null, "updated_timestamp": 1550574558607 }`

func TestSubmitWithdraw(t *testing.T) {
	t.Parallel()
	var resp *WithdrawData
	err := json.Unmarshal([]byte(submitWithdrawalResponseJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err = d.SubmitWithdraw(context.Background(), currencyBTC, core.BitcoinDonationAddress, "", 0.001)
	assert.NoError(t, err)
	_, err = d.WSSubmitWithdraw(currencyBTC, core.BitcoinDonationAddress, "", 0.001)
	assert.NoError(t, err)
}

func TestGetAnnouncements(t *testing.T) {
	t.Parallel()
	_, err := d.GetAnnouncements(context.Background(), time.Now(), 5)
	assert.NoError(t, err)
	_, err = d.WSRetrieveAnnouncements(time.Now(), 5)
	assert.NoError(t, err)
}

func TestGetPublicPortfolioMargins(t *testing.T) {
	info, err := d.GetInstrument(context.Background(), "BTC-PERPETUAL")
	if err != nil {
		t.Skip(err)
	}
	_, err = d.GetPublicPortfolioMargins(context.Background(), currencyBTC, map[string]float64{
		"BTC-PERPETUAL": info.ContractSize * 2,
	})
	assert.NoError(t, err)
}

func TestGetAccessLog(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetAccessLog(context.Background(), 0, 0)
	assert.NoError(t, err)
	_, err = d.WSRetrieveAccessLog(0, 0)
	assert.NoError(t, err)
}

func TestChangeAPIKeyName(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.ChangeAPIKeyName(context.Background(), 1, "TestKey123")
	assert.NoError(t, err)
	_, err = d.WSChangeAPIKeyName(1, "TestKey123")
	assert.NoError(t, err)
}

func TestChangeMarginModel(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err := d.ChangeMarginModel(context.Background(), 2, "segregated_pm", false)
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.WsChangeMarginModel(2, "segregated_pm", false)
	if err != nil {
		t.Fatal(err)
	}
}

func TestChangeScopeInAPIKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err := d.ChangeScopeInAPIKey(context.Background(), 1, "account:read_write")
	assert.NoError(t, err)
	_, err = d.WSChangeScopeInAPIKey(1, "account:read_write")
	assert.NoError(t, err)
}

func TestChangeSubAccountName(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	err := d.ChangeSubAccountName(context.Background(), 1, "new_sub")
	assert.NoError(t, err)
	err = d.WSChangeSubAccountName(1, "new_sub")
	assert.NoError(t, err)
}

func TestCreateAPIKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err := d.CreateAPIKey(context.Background(), "account:read_write", "new_sub", false)
	assert.NoError(t, err)
	_, err = d.WSCreateAPIKey("account:read_write", "new_sub", false)
	assert.NoError(t, err)
}

func TestCreateSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err := d.CreateSubAccount(context.Background())
	assert.NoError(t, err)
	_, err = d.WSCreateSubAccount()
	assert.NoError(t, err)
}

func TestDisableAPIKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err := d.DisableAPIKey(context.Background(), 1)
	assert.NoError(t, err)
	_, err = d.WSDisableAPIKey(1)
	assert.NoError(t, err)
}

func TestEditAPIKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err := d.EditAPIKey(context.Background(), 1234, "trade", "", false, []string{"read", "read_write"}, []string{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.WsEditAPIKey(1234, "trade", "", false, []string{"read", "read_write"}, []string{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestEnableAffiliateProgram(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	err := d.EnableAffiliateProgram(context.Background())
	assert.NoError(t, err)
	err = d.WSEnableAffiliateProgram()
	assert.NoError(t, err)
}

func TestEnableAPIKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err := d.EnableAPIKey(context.Background(), 1)
	assert.NoError(t, err)
	_, err = d.WSEnableAPIKey(1)
	assert.NoError(t, err)
}

func TestGetAffiliateProgramInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetAffiliateProgramInfo(context.Background())
	assert.NoError(t, err)
	_, err = d.WSRetrieveAffiliateProgramInfo()
	assert.NoError(t, err)
}

func TestGetEmailLanguage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetEmailLanguage(context.Background())
	assert.NoError(t, err)
	_, err = d.WSRetrieveEmailLanguage()
	assert.NoError(t, err)
}

func TestGetNewAnnouncements(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetNewAnnouncements(context.Background())
	assert.NoError(t, err)
	_, err = d.WSRetrieveNewAnnouncements()
	assert.NoError(t, err)
}

func TestGetPrivatePortfolioMargins(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetPrivatePortfolioMargins(context.Background(), currencyBTC, false, nil)
	assert.NoError(t, err)
}

func TestWsRetrivePricatePortfolioMargins(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.WSRetrievePrivatePortfolioMargins(currencyBTC, false, nil)
	assert.NoError(t, err)
}

func TestGetPosition(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetPosition(context.Background(), btcPerpInstrument)
	assert.NoError(t, err)
	_, err = d.WSRetrievePosition(btcPerpInstrument)
	assert.NoError(t, err)
}

func TestGetSubAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetSubAccounts(context.Background(), false)
	assert.NoError(t, err)
	_, err = d.WSRetrieveSubAccounts(false)
	assert.NoError(t, err)
}

func TestGetSubAccountDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetSubAccountDetails(context.Background(), currencyBTC, false)
	assert.NoError(t, err)
	_, err = d.WSRetrieveSubAccountDetails(currencyBTC, false)
	assert.NoError(t, err)
}

func TestGetPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetPositions(context.Background(), currencyBTC, "option")
	assert.NoError(t, err)
	_, err = d.GetPositions(context.Background(), currencyETH, "")
	assert.NoError(t, err)
	_, err = d.WSRetrievePositions(currencyBTC, "option")
	assert.NoError(t, err)
	_, err = d.WSRetrievePositions(currencyETH, "")
	assert.NoError(t, err)
}

const getTransactionLogResponseJSON = `{"logs": [ { "username": "TestUser", "user_seq": 6009, "user_id": 7, "type": "transfer", "trade_id": null, "timestamp": 1613659830333, "side": "-", "price": null, "position": null, "order_id": null, "interest_pl": null, "instrument_name": null, "info": { "transfer_type": "subaccount", "other_user_id": 27, "other_user": "Subaccount" }, "id": 61312, "equity": 3000.9275869, "currency": "BTC", "commission": 0, "change": -2.5, "cashflow": -2.5, "balance": 3001.22270418 } ], "continuation": 61282 }`

func TestGetTransactionLog(t *testing.T) {
	t.Parallel()
	var resp *TransactionsData
	err := json.Unmarshal([]byte(getTransactionLogResponseJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err = d.GetTransactionLog(context.Background(), currencyBTC, "trade", time.Now().Add(-24*time.Hour), time.Now(), 5, 0)
	assert.NoError(t, err)
	_, err = d.WSRetrieveTransactionLog(currencyBTC, "trade", time.Now().Add(-24*time.Hour), time.Now(), 5, 0)
	assert.NoError(t, err)
}

func TestGetUserLocks(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetUserLocks(context.Background())
	assert.NoError(t, err)
	_, err = d.WSRetrieveUserLocks()
	assert.NoError(t, err)
}

func TestListAPIKeys(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.ListAPIKeys(context.Background(), "")
	assert.NoError(t, err)
	_, err = d.WSListAPIKeys("")
	assert.NoError(t, err)
}

func TestGetCustodyAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetCustodyAccounts(context.Background(), currency.BTC)
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.WsRetrieveCustodyAccounts(currency.BTC)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRemoveAPIKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	err := d.RemoveAPIKey(context.Background(), 1)
	assert.NoError(t, err)
	err = d.WSRemoveAPIKey(1)
	assert.NoError(t, err)
}

func TestRemoveSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	err := d.RemoveSubAccount(context.Background(), 1)
	assert.NoError(t, err)
	err = d.WSRemoveSubAccount(1)
	assert.NoError(t, err)
}

func TestResetAPIKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err := d.ResetAPIKey(context.Background(), 1)
	assert.NoError(t, err)
	err = d.WSResetAPIKey(1)
	assert.NoError(t, err)
}

func TestSetAnnouncementAsRead(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	err := d.SetAnnouncementAsRead(context.Background(), 1)
	assert.NoError(t, err)
}

func TestSetEmailForSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	err := d.SetEmailForSubAccount(context.Background(), 1, "wrongemail@wrongemail.com")
	assert.NoError(t, err)
	err = d.WSSetEmailForSubAccount(1, "wrongemail@wrongemail.com")
	assert.NoError(t, err)
}

func TestSetEmailLanguage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	err := d.SetEmailLanguage(context.Background(), "en")
	assert.NoError(t, err)
	err = d.WSSetEmailLanguage("en")
	assert.NoError(t, err)
}

func TestSetSelfTradingConfig(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err := d.SetSelfTradingConfig(context.Background(), "reject_taker", false)
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.WsSetSelfTradingConfig("reject_taker", false)
	if err != nil {
		t.Fatal(err)
	}
}

func TestToggleNotificationsFromSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	err := d.ToggleNotificationsFromSubAccount(context.Background(), 1, false)
	assert.NoError(t, err)
	err = d.WSToggleNotificationsFromSubAccount(1, false)
	assert.NoError(t, err)
}

func TestTogglePortfolioMargining(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.TogglePortfolioMargining(context.Background(), 1234, false, false)
	assert.NoError(t, err)
	_, err = d.WSTogglePortfolioMargining(1234, false, false)
	assert.NoError(t, err)
}

func TestToggleSubAccountLogin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	err := d.ToggleSubAccountLogin(context.Background(), 1, false)
	assert.NoError(t, err)
	err = d.WSToggleSubAccountLogin(1, false)
	assert.NoError(t, err)
}

func TestSubmitBuy(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	pairs, err := d.GetEnabledPairs(asset.Futures)
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
	assert.NoError(t, err)
	_, err = d.WSSubmitBuy(&OrderBuyAndSellParams{
		Instrument: btcPerpInstrument, OrderType: "limit",
		Label: "testOrder", TimeInForce: "",
		Trigger: "", Advanced: "",
		Amount: 30, Price: 500000,
		MaxShow: 0, TriggerPrice: 0,
		PostOnly: false, RejectPostOnly: false,
		ReduceOnly: false, MMP: false})
	assert.NoError(t, err)
}

func TestSubmitSell(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	info, err := d.GetInstrument(context.Background(), btcPerpInstrument)
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.SubmitSell(context.Background(), &OrderBuyAndSellParams{Instrument: btcPerpInstrument, OrderType: "limit", Label: "testOrder", TimeInForce: "", Trigger: "", Advanced: "", Amount: info.ContractSize * 3, Price: 500000, MaxShow: 0, TriggerPrice: 0, PostOnly: false, RejectPostOnly: false, ReduceOnly: false, MMP: false})
	assert.NoError(t, err)
	_, err = d.WSSubmitSell(&OrderBuyAndSellParams{
		Instrument: btcPerpInstrument, OrderType: "limit",
		Label: "testOrder", TimeInForce: "",
		Trigger: "", Advanced: "", Amount: info.ContractSize * 3,
		Price: 500000, MaxShow: 0, TriggerPrice: 0, PostOnly: false,
		RejectPostOnly: false, ReduceOnly: false, MMP: false})
	assert.NoError(t, err)
}

func TestEditOrderByLabel(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err := d.EditOrderByLabel(context.Background(), &OrderBuyAndSellParams{Label: "incorrectUserLabel", Instrument: btcPerpInstrument,
		Advanced: "", Amount: 1, Price: 30000, TriggerPrice: 0, PostOnly: false, ReduceOnly: false, RejectPostOnly: false, MMP: false})
	assert.NoError(t, err)
	_, err = d.WSEditOrderByLabel(&OrderBuyAndSellParams{Label: "incorrectUserLabel", Instrument: btcPerpInstrument,
		Advanced: "", Amount: 1, Price: 30000, TriggerPrice: 0, PostOnly: false, ReduceOnly: false, RejectPostOnly: false, MMP: false})
	assert.NoError(t, err)
}

const submitCancelResponseJSON = `{"triggered": false, "trigger": "index_price", "time_in_force": "good_til_cancelled", "trigger_price": 144.73, "reduce_only": false, "profit_loss": 0, "price": "market_price", "post_only": false, "order_type": "stop_market", "order_state": "untriggered", "order_id": "ETH-SLIS-12", "max_show": 5, "last_update_timestamp": 1550575961291, "label": "", "is_liquidation": false, "instrument_name": "ETH-PERPETUAL", "direction": "sell", "creation_timestamp": 1550575961291, "api": false, "amount": 5 }`

func TestSubmitCancel(t *testing.T) {
	t.Parallel()
	var resp *PrivateCancelData
	err := json.Unmarshal([]byte(submitCancelResponseJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err = d.SubmitCancel(context.Background(), "incorrectID")
	assert.NoError(t, err)
	_, err = d.WSSubmitCancel("incorrectID")
	assert.NoError(t, err)
}

func TestSubmitCancelAll(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err := d.SubmitCancelAll(context.Background())
	assert.NoError(t, err)
	_, err = d.WSSubmitCancelAll()
	assert.NoError(t, err)
}

func TestSubmitCancelAllByCurrency(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err := d.SubmitCancelAllByCurrency(context.Background(), currency.BTC, "option", "")
	assert.NoError(t, err)
	_, err = d.WSSubmitCancelAllByCurrency(currencyBTC, "option", "")
	assert.NoError(t, err)
}

func TestSubmitCancelAllByKind(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err := d.SubmitCancelAllByKind(context.Background(), currency.ETH, "option_combo", "trigger_all", true)
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.WsSubmitCancelAllByKind(currency.ETH, "option_combo", "trigger_all", true)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSubmitCancelAllByInstrument(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err := d.SubmitCancelAllByInstrument(context.Background(), btcPerpInstrument, "all", true, true)
	assert.NoError(t, err)
	_, err = d.WSSubmitCancelAllByInstrument(btcPerpInstrument, "all", true, true)
	assert.NoError(t, err)
}

func TestSubmitCancelByLabel(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err := d.SubmitCancelByLabel(context.Background(), "incorrectOrderLabel", "")
	assert.NoError(t, err)
	_, err = d.WSSubmitCancelByLabel("incorrectOrderLabel", "")
	assert.NoError(t, err)
}

func TestSubmitClosePosition(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err := d.SubmitClosePosition(context.Background(), d.formatFuturesTradablePair(futuresTradablePair), "limit", 35000)
	assert.NoError(t, err)
	_, err = d.WSSubmitClosePosition(d.formatFuturesTradablePair(futuresTradablePair), "limit", 35000)
	assert.NoError(t, err)
}

func TestGetMargins(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetMargins(context.Background(), d.formatFuturesTradablePair(futuresTradablePair), 5, 35000)
	assert.NoError(t, err)
	_, err = d.WSRetrieveMargins(d.formatFuturesTradablePair(futuresTradablePair), 5, 35000)
	assert.NoError(t, err)
}

func TestGetMMPConfig(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetMMPConfig(context.Background(), currencyETH)
	assert.NoError(t, err)
	_, err = d.WSRetrieveMMPConfig(currencyETH)
	assert.NoError(t, err)
}

const getOpenOrdersByCurrencyResponseJSON = `[{ "time_in_force": "good_til_cancelled", "reduce_only": false, "profit_loss": 0, "price": 0.0028, "post_only": false, "order_type": "limit", "order_state": "open", "order_id": "146062", "max_show": 10, "last_update_timestamp": 1550050597036, "label": "", "is_liquidation": false, "instrument_name": "BTC-15FEB19-3250-P", "filled_amount": 0, "direction": "buy", "creation_timestamp": 1550050597036, "commission": 0, "average_price": 0, "api": true, "amount": 10 } ]`

func TestGetOpenOrdersByCurrency(t *testing.T) {
	t.Parallel()
	var resp []OrderData
	err := json.Unmarshal([]byte(getOpenOrdersByCurrencyResponseJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err = d.GetOpenOrdersByCurrency(context.Background(), currencyBTC, "option", "all")
	assert.NoError(t, err)
	_, err = d.WSRetrieveOpenOrdersByCurrency(currencyBTC, "option", "all")
	assert.NoError(t, err)
}

func TestGetOpenOrdersByLabel(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err := d.GetOpenOrdersByLabel(context.Background(), currency.EURR, "the-label")
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.WSRetrieveOpenOrdersByLabel(currency.EURR, "the-label")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetOpenOrdersByInstrument(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetOpenOrdersByInstrument(context.Background(), btcPerpInstrument, "all")
	assert.NoError(t, err)
	_, err = d.WSRetrieveOpenOrdersByInstrument(btcPerpInstrument, "all")
	assert.NoError(t, err)
}

func TestGetOrderHistoryByCurrency(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetOrderHistoryByCurrency(context.Background(), currencyBTC, "future", 0, 0, false, false)
	assert.NoError(t, err)
	_, err = d.WSRetrieveOrderHistoryByCurrency(currencyBTC, "future", 0, 0, false, false)
	assert.NoError(t, err)
}

func TestGetOrderHistoryByInstrument(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetOrderHistoryByInstrument(context.Background(), btcPerpInstrument, 0, 0, false, false)
	assert.NoError(t, err)
	_, err = d.WSRetrieveOrderHistoryByInstrument(btcPerpInstrument, 0, 0, false, false)
	assert.NoError(t, err)
}

func TestGetOrderMarginsByID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetOrderMarginsByID(context.Background(), []string{"ETH-349280", "ETH-349279", "ETH-349278"})
	assert.NoError(t, err)
	_, err = d.WSRetrieveOrderMarginsByID([]string{"ETH-349280", "ETH-349279", "ETH-349278"})
	assert.NoError(t, err)
}

func TestGetOrderState(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetOrderState(context.Background(), "brokenid123")
	assert.NoError(t, err)
	_, err = d.WSRetrievesOrderState("brokenid123")
	assert.NoError(t, err)
}

func TestGetOrderStateByLabel(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetOrderStateByLabel(context.Background(), currency.EURR, "the-label")
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.WsRetrieveOrderStateByLabel(currency.EURR, "the-label")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetTriggerOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetTriggerOrderHistory(context.Background(), currencyETH, "", "", 0)
	assert.NoError(t, err)
	_, err = d.WSRetrieveTriggerOrderHistory(currencyETH, "", "", 0)
	assert.NoError(t, err)
}

const getUserTradesByCurrencyResponseJSON = `{"trades": [ { "underlying_price": 204.5, "trade_seq": 3, "trade_id": "ETH-2696060", "timestamp": 1590480363130, "tick_direction": 2, "state": "filled", "reduce_only": false, "price": 0.361, "post_only": false, "order_type": "limit", "order_id": "ETH-584827850", "matching_id": null, "mark_price": 0.364585, "liquidity": "T", "iv": 0, "instrument_name": "ETH-29MAY20-130-C", "index_price": 203.72, "fee_currency": "ETH", "fee": 0.002, "direction": "sell", "amount": 5 }, { "underlying_price": 204.82, "trade_seq": 3, "trade_id": "ETH-2696062", "timestamp": 1590480416119, "tick_direction": 0, "state": "filled", "reduce_only": false, "price": 0.015, "post_only": false, "order_type": "limit", "order_id": "ETH-584828229", "matching_id": null, "mark_price": 0.000596, "liquidity": "T", "iv": 352.91, "instrument_name": "ETH-29MAY20-140-P", "index_price": 204.06, "fee_currency": "ETH", "fee": 0.002, "direction": "buy", "amount": 5 } ], "has_more": true }`

func TestGetUserTradesByCurrency(t *testing.T) {
	t.Parallel()
	var resp *UserTradesData
	err := json.Unmarshal([]byte(getUserTradesByCurrencyResponseJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err = d.GetUserTradesByCurrency(context.Background(), currencyETH, "future", "", "", "asc", 0, false)
	assert.NoError(t, err)
	_, err = d.WSRetrieveUserTradesByCurrency(currencyETH, "future", "", "", "asc", 0, false)
	assert.NoError(t, err)
}

func TestGetUserTradesByCurrencyAndTime(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetUserTradesByCurrencyAndTime(context.Background(), currencyETH, "future", "default", 5, time.Now().Add(-time.Hour*10), time.Now().Add(-time.Hour*1))
	assert.NoError(t, err)
	_, err = d.WSRetrieveUserTradesByCurrencyAndTime(currencyETH, "future", "default", 5, time.Now().Add(-time.Hour*4), time.Now())
	assert.NoError(t, err)
}

func TestGetUserTradesByInstrument(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetUserTradesByInstrument(context.Background(), btcPerpInstrument, "asc", 5, 10, 4, true)
	assert.NoError(t, err)
	_, err = d.WSRetrieveUserTradesByInstrument(btcPerpInstrument, "asc", 5, 10, 4, true)
	assert.NoError(t, err)
}

func TestGetUserTradesByInstrumentAndTime(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetUserTradesByInstrumentAndTime(context.Background(), btcPerpInstrument, "asc", 10, time.Now().Add(-time.Hour), time.Now())
	assert.NoError(t, err)
	_, err = d.WSRetrieveUserTradesByInstrumentAndTime(btcPerpInstrument, "asc", 10, false, time.Now().Add(-time.Hour), time.Now())
	assert.NoError(t, err)
}

func TestGetUserTradesByOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetUserTradesByOrder(context.Background(), "wrongOrderID", "default")
	assert.NoError(t, err)
	_, err = d.WSRetrieveUserTradesByOrder("wrongOrderID", "default")
	assert.NoError(t, err)
}

func TestResetMMP(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	err := d.ResetMMP(context.Background(), currencyBTC)
	assert.NoError(t, err)
	err = d.WSResetMMP(currencyBTC)
	assert.NoError(t, err)
}

func TestSendRequestForQuote(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	err := d.SendRequestForQuote(context.Background(), d.formatFuturesTradablePair(futuresTradablePair), 1000, order.Buy)
	assert.NoError(t, err)
	err = d.WSSendRequestForQuote(d.formatFuturesTradablePair(futuresTradablePair), 1000, order.Buy)
	assert.NoError(t, err)
}

func TestSetMMPConfig(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	err := d.SetMMPConfig(context.Background(), currencyBTC, kline.FiveMin, 5, 0, 0)
	assert.NoError(t, err)
	err = d.WSSetMMPConfig(currencyBTC, kline.FiveMin, 5, 0, 0)
	assert.NoError(t, err)
}

func TestGetSettlementHistoryByCurency(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetSettlementHistoryByCurency(context.Background(), currencyBTC, "settlement", "", 10, time.Now().Add(-time.Hour))
	assert.NoError(t, err)
	_, err = d.WSRetrieveSettlementHistoryByCurency(currencyBTC, "settlement", "", 10, time.Now().Add(-time.Hour))
	assert.NoError(t, err)
}

const getSettlementHistoryByInstrumentResponseJSON = `{"settlements": [ { "type": "settlement", "timestamp": 1550475692526, "session_profit_loss": 0.038358299, "profit_loss": -0.001783937, "position": -66, "mark_price": 121.67, "instrument_name": "ETH-22FEB19", "index_price": 119.8 } ], "continuation": "xY7T6cusbMBNpH9SNmKb94jXSBxUPojJEdCPL4YociHBUgAhWQvEP" }`

func TestGetSettlementHistoryByInstrument(t *testing.T) {
	t.Parallel()
	var resp *PrivateSettlementsHistoryData
	err := json.Unmarshal([]byte(getSettlementHistoryByInstrumentResponseJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	resp, err = d.GetSettlementHistoryByInstrument(context.Background(), btcPerpInstrument, "settlement", "", 10, time.Now().Add(-time.Hour))
	assert.NoError(t, err)
	_, err = d.WSRetrieveSettlementHistoryByInstrument(btcPerpInstrument, "settlement", "", 10, time.Now().Add(-time.Hour))
	assert.NoError(t, err)
}

func TestSubmitEdit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err := d.SubmitEdit(context.Background(), &OrderBuyAndSellParams{OrderID: "incorrectID", Advanced: "", TriggerPrice: 0.001, Price: 100000, Amount: 123})
	assert.NoError(t, err)
	_, err = d.WSSubmitEdit(&OrderBuyAndSellParams{
		OrderID:      "incorrectID",
		Advanced:     "",
		TriggerPrice: 0.001,
		Price:        100000,
		Amount:       123,
	})
	assert.NoError(t, err)
}

// Combo Books Endpoints

func TestGetComboIDS(t *testing.T) {
	t.Parallel()
	_, err := d.GetComboIDS(context.Background(), currencyBTC, "")
	assert.NoError(t, err)
	combos, err := d.WSRetrieveComboIDS(currencyBTC, "")
	assert.NoError(t, err)
	assert.False(t, len(combos) == 0, "no combo instance found for currency BTC")
}

func TestGetComboDetails(t *testing.T) {
	t.Parallel()
	sleepUntilTradablePairsUpdated()
	_, err := d.GetComboDetails(context.Background(), futureComboTradablePair.String())
	assert.NoError(t, err)
	_, err = d.WSRetrieveComboDetails(futureComboTradablePair.String())
	assert.NoError(t, err)
}

func TestGetCombos(t *testing.T) {
	t.Parallel()
	_, err := d.GetCombos(context.Background(), currencyBTC)
	assert.NoError(t, err)
}

func TestCreateCombo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err := d.CreateCombo(context.Background(), []ComboParam{})
	assert.Falsef(t, err != nil && !errors.Is(errNoArgumentPassed, err), "expecting %v, but found %v", errNoArgumentPassed, err)
	instruments, err := d.GetEnabledPairs(asset.Futures)
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
	assert.Falsef(t, err != nil && !errors.Is(errInvalidAmount, err), "expecting %v, but found %v", errInvalidAmount, err)
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
	assert.NoError(t, err, "expecting error message 'invalid direction', but found %v", err)
	sleepUntilTradablePairsUpdated()
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
	assert.NoError(t, err)
	_, err = d.WSCreateCombo([]ComboParam{})
	assert.Falsef(t, err != nil && !errors.Is(errNoArgumentPassed, err), "expecting %v, but found %v", errNoArgumentPassed, err)
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
	assert.Falsef(t, err != nil && !errors.Is(errInvalidAmount, err), "expecting %v, but found %v", errInvalidAmount, err)
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
	assert.NoErrorf(t, err, "expecting error message 'invalid direction', but found %v", err)
	_, err = d.WSCreateCombo([]ComboParam{
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
	assert.NoError(t, err)
}

func TestVerifyBlockTrade(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	info, err := d.GetInstrument(context.Background(), btcPerpInstrument)
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
	assert.NoError(t, err)
	_, err = d.WSVerifyBlockTrade(time.Now(), "sdjkafdad", "maker", "", []BlockTradeParam{
		{
			Price:          0.777 * 28000,
			InstrumentName: btcPerpInstrument,
			Direction:      "buy",
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	})
	assert.NoError(t, err)
}

func TestInvalidateBlockTradeSignature(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	err := d.InvalidateBlockTradeSignature(context.Background(), "verified_signature_string")
	assert.NoError(t, err)
	err = d.WsInvalidateBlockTradeSignature("verified_signature_string")
	assert.NoError(t, err)
}

func TestExecuteBlockTrade(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	info, err := d.GetInstrument(context.Background(), btcPerpInstrument)
	if err != nil {
		t.Skip(err)
	}
	_, err = d.ExecuteBlockTrade(context.Background(), time.Now(), "something", "maker", "", []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: btcPerpInstrument,
			Direction:      "buy",
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	})
	assert.NoError(t, err)
	_, err = d.WSExecuteBlockTrade(time.Now(), "sdjkafdad", "maker", "", []BlockTradeParam{
		{
			Price:          0.777 * 22000,
			InstrumentName: btcPerpInstrument,
			Direction:      "buy",
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	})
	assert.NoError(t, err)
}

const getUserBlocTradeResponseJSON = `[ { "trade_seq": 37, "trade_id": "92437", "timestamp": 1565089523719, "tick_direction": 3, "state": "filled", "price": 0.0001, "order_type": "limit", "order_id": "343062", "matching_id": null, "liquidity": "T", "iv": 0, "instrument_name": "BTC-9AUG19-10250-C", "index_price": 11738, "fee_currency": "BTC", "fee": 0.00025, "direction": "sell", "block_trade_id": "61", "amount": 10 }, { "trade_seq": 25350, "trade_id": "92435", "timestamp": 1565089523719, "tick_direction": 3, "state": "filled", "price": 11590, "order_type": "limit", "order_id": "343058", "matching_id": null, "liquidity": "T", "instrument_name": "BTC-PERPETUAL", "index_price": 11737.98, "fee_currency": "BTC", "fee": 0.00000164, "direction": "buy", "block_trade_id": "61", "amount": 190 } ]`

func TestGetUserBlocTrade(t *testing.T) {
	t.Parallel()
	var resp []BlockTradeData
	err := json.Unmarshal([]byte(getUserBlocTradeResponseJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err = d.GetUserBlockTrade(context.Background(), "12345567")
	assert.NoError(t, err)
	_, err = d.WSRetrieveUserBlockTrade("12345567")
	assert.NoError(t, err)
}

func TestGetLastBlockTradesbyCurrency(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetLastBlockTradesByCurrency(context.Background(), "SOL", "", "", 5)
	assert.NoError(t, err)
	_, err = d.WSRetrieveLastBlockTradesByCurrency("SOL", "", "", 5)
	assert.NoError(t, err)
}

func TestMovePositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	info, err := d.GetInstrument(context.Background(), "BTC-PERPETUAL")
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
	assert.NoError(t, err)
	_, err = d.WSMovePositions(currencyBTC, 123, 345, []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: btcPerpInstrument,
			Direction:      "buy",
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	})
	assert.NoError(t, err)
}

func TestSimulateBlockTrade(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	info, err := d.GetInstrument(context.Background(), "BTC-PERPETUAL")
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.SimulateBlockTrade(context.Background(), "maker", []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: btcPerpInstrument,
			Direction:      "buy",
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	})
	assert.ErrorIs(t, err, nil)
	_, err = d.WsSimulateBlockTrade("taker", []BlockTradeParam{
		{
			Price:          0.777 * 25000,
			InstrumentName: btcPerpInstrument,
			Direction:      "buy",
			Amount:         info.MinimumTradeAmount*5 + (200000 - info.MinimumTradeAmount*5) + 10,
		},
	})
	assert.NoError(t, err)
}

func setupWs() {
	if !d.Websocket.IsEnabled() {
		return
	}
	if !sharedtestvalues.AreAPICredentialsSet(d) {
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
	assert.NoError(t, err)
}

func TestFetchTicker(t *testing.T) {
	t.Parallel()
	_, err := d.FetchTicker(context.Background(), futuresTradablePair, asset.Futures)
	assert.NoError(t, err)
	_, err = d.FetchTicker(context.Background(), spotTradablePair, asset.Spot)
	assert.NoError(t, err)
	sleepUntilTradablePairsUpdated()
	_, err = d.FetchTicker(context.Background(), optionsTradablePair, asset.Options)
	assert.NoError(t, err)
	_, err = d.FetchTicker(context.Background(), optionComboTradablePair, asset.OptionCombo)
	assert.NoError(t, err)
	_, err = d.FetchTicker(context.Background(), futureComboTradablePair, asset.FutureCombo)
	assert.NoError(t, err)
}

func TestFetchOrderbook(t *testing.T) {
	t.Parallel()
	_, err := d.FetchOrderbook(context.Background(), futuresTradablePair, asset.Futures)
	assert.NoError(t, err)
	_, err = d.FetchOrderbook(context.Background(), spotTradablePair, asset.Spot)
	assert.NoError(t, err)
	sleepUntilTradablePairsUpdated()
	_, err = d.FetchOrderbook(context.Background(), futureComboTradablePair, asset.FutureCombo)
	assert.NoError(t, err)
	_, err = d.FetchOrderbook(context.Background(), optionComboTradablePair, asset.OptionCombo)
	assert.NoError(t, err)
	_, err = d.FetchOrderbook(context.Background(), optionsTradablePair, asset.Options)
	assert.NoError(t, err)
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.UpdateAccountInfo(context.Background(), asset.Futures)
	assert.NoError(t, err)
}

func TestFetchAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.FetchAccountInfo(context.Background(), asset.Spot)
	assert.NoError(t, err)
	_, err = d.FetchAccountInfo(context.Background(), asset.Futures)
	assert.NoError(t, err)
	_, err = d.FetchAccountInfo(context.Background(), asset.Options)
	assert.NoError(t, err)
	_, err = d.FetchAccountInfo(context.Background(), asset.OptionCombo)
	assert.NoError(t, err)
	_, err = d.FetchAccountInfo(context.Background(), asset.FutureCombo)
	assert.NoError(t, err)
}

func TestGetFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetAccountFundingHistory(context.Background())
	assert.NoError(t, err)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Empty)
	assert.NoError(t, err)
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := d.GetRecentTrades(context.Background(), futuresTradablePair, asset.Futures)
	assert.NoError(t, err)
	_, err = d.GetRecentTrades(context.Background(), spotTradablePair, asset.Spot)
	assert.NoError(t, err)
}

func TestWSRetrievePublicPortfolioMargins(t *testing.T) {
	t.Parallel()
	info, err := d.GetInstrument(context.Background(), btcPerpInstrument)
	if err != nil {
		t.Skip(err)
	}
	_, err = d.WSRetrievePublicPortfolioMargins(currencyBTC, map[string]float64{btcPerpInstrument: info.ContractSize * 2})
	assert.NoError(t, err)
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          futuresTradablePair,
		AssetType:     asset.Futures,
	}
	_, err := d.CancelAllOrders(context.Background(), orderCancellation)
	assert.False(t, err != nil && !errors.Is(err, errNoOrderDeleted), err)
	orderCancellation.AssetType = asset.FutureCombo
	orderCancellation.Pair = futureComboTradablePair
	_, err = d.CancelAllOrders(context.Background(), orderCancellation)
	assert.False(t, err != nil && !errors.Is(err, errNoOrderDeleted), err)
	sleepUntilTradablePairsUpdated()
	orderCancellation.AssetType = asset.Options
	orderCancellation.Pair = optionsTradablePair
	_, err = d.CancelAllOrders(context.Background(), orderCancellation)
	assert.False(t, err != nil && !errors.Is(err, errNoOrderDeleted), err)
	orderCancellation.AssetType = asset.OptionCombo
	orderCancellation.Pair = optionComboTradablePair
	_, err = d.CancelAllOrders(context.Background(), orderCancellation)
	assert.False(t, err != nil && !errors.Is(err, errNoOrderDeleted), err)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetOrderInfo(context.Background(), "1234", spotTradablePair, asset.Spot)
	assert.NoError(t, err)
	_, err = d.GetOrderInfo(context.Background(), "1234", futuresTradablePair, asset.Futures)
	assert.NoError(t, err)
	sleepUntilTradablePairsUpdated()
	_, err = d.GetOrderInfo(context.Background(), "1234", futureComboTradablePair, asset.FutureCombo)
	assert.NoError(t, err)
	_, err = d.GetOrderInfo(context.Background(), "1234", optionsTradablePair, asset.Options)
	assert.NoError(t, err)
	_, err = d.GetOrderInfo(context.Background(), "1234", optionComboTradablePair, asset.OptionCombo)
	assert.NoError(t, err)
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetDepositAddress(context.Background(), currency.BTC, "", "")
	assert.False(t, err != nil && !errors.Is(err, common.ErrNoResponse), err)
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err := d.WithdrawCryptocurrencyFunds(context.Background(), &withdraw.Request{
		Exchange:    d.Name,
		Amount:      1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: "0x1nv4l1d",
			Chain:   "tetheruse",
		},
	})
	assert.NoErrorf(t, err, "Withdraw failed to be placed: %v", err)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	var getOrdersRequest = order.MultiOrderRequest{
		Type: order.AnyType, AssetType: asset.Futures,
		Side: order.AnySide, Pairs: currency.Pairs{futuresTradablePair},
	}
	sleepUntilTradablePairsUpdated()
	_, err := d.GetActiveOrders(context.Background(), &getOrdersRequest)
	assert.NoError(t, err)
	getOrdersRequest.AssetType = asset.Options
	getOrdersRequest.Pairs = currency.Pairs{optionsTradablePair}
	_, err = d.GetActiveOrders(context.Background(), &getOrdersRequest)
	assert.NoError(t, err)
	getOrdersRequest.AssetType = asset.OptionCombo
	getOrdersRequest.Pairs = currency.Pairs{optionComboTradablePair}
	_, err = d.GetActiveOrders(context.Background(), &getOrdersRequest)
	assert.NoError(t, err)
	getOrdersRequest.AssetType = asset.FutureCombo
	getOrdersRequest.Pairs = currency.Pairs{futureComboTradablePair}
	_, err = d.GetActiveOrders(context.Background(), &getOrdersRequest)
	assert.NoError(t, err)
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetOrderHistory(context.Background(), &order.MultiOrderRequest{
		Type: order.AnyType, AssetType: asset.Futures,
		Side: order.AnySide, Pairs: []currency.Pair{futuresTradablePair},
	})
	assert.NoError(t, err)
	_, err = d.GetOrderHistory(context.Background(), &order.MultiOrderRequest{
		Type: order.AnyType, AssetType: asset.Spot,
		Side: order.AnySide, Pairs: []currency.Pair{spotTradablePair},
	})
	assert.NoError(t, err)
	sleepUntilTradablePairsUpdated()
	_, err = d.GetOrderHistory(context.Background(), &order.MultiOrderRequest{
		Type: order.AnyType, AssetType: asset.Options,
		Side: order.AnySide, Pairs: []currency.Pair{optionsTradablePair},
	})
	assert.NoError(t, err)
	_, err = d.GetOrderHistory(context.Background(), &order.MultiOrderRequest{
		Type: order.AnyType, AssetType: asset.FutureCombo,
		Side: order.AnySide, Pairs: []currency.Pair{futureComboTradablePair},
	})
	assert.NoError(t, err)
	_, err = d.GetOrderHistory(context.Background(), &order.MultiOrderRequest{Type: order.AnyType, AssetType: asset.OptionCombo, Side: order.AnySide, Pairs: []currency.Pair{optionComboTradablePair}})
	assert.NoError(t, err)
}

func TestGuessAssetTypeFromInstrument(t *testing.T) {
	availablePairs, err := d.GetEnabledPairs(asset.Futures)
	if err != nil {
		t.Fatal(err)
	}
	var assetType asset.Item
	for id, cp := range availablePairs {
		t.Run(strconv.Itoa(id), func(t *testing.T) {
			assetType, err = guessAssetTypeFromInstrument(cp)
			assert.False(t, assetType != asset.Futures, "expected %v, but found %v", asset.Futures, assetType)
			assert.NoError(t, err)
		})
	}
	availablePairs, err = d.GetEnabledPairs(asset.Options)
	if err != nil {
		t.Fatal(err)
	}
	for id, cp := range availablePairs {
		t.Run(strconv.Itoa(id), func(t *testing.T) {
			assetType, err = guessAssetTypeFromInstrument(cp)
			assert.Falsef(t, assetType != asset.Options, "expected %v, but found %v", asset.Options, assetType)
			assert.NoError(t, err)
		})
	}
	availablePairs, err = d.GetEnabledPairs(asset.OptionCombo)
	if err != nil {
		t.Fatal(err)
	}
	for id, cp := range availablePairs {
		t.Run(strconv.Itoa(id), func(t *testing.T) {
			assetType, err = guessAssetTypeFromInstrument(cp)
			assert.Falsef(t, assetType != asset.OptionCombo, "expected %v, but found %v", asset.OptionCombo, assetType)
			assert.NoError(t, err)
		})
	}
	availablePairs, err = d.GetEnabledPairs(asset.FutureCombo)
	if err != nil {
		t.Fatal(err)
	}
	for id, cp := range availablePairs {
		t.Run(strconv.Itoa(id), func(t *testing.T) {
			assetType, err = guessAssetTypeFromInstrument(cp)
			assert.Falsef(t, assetType != asset.FutureCombo, "expected %v, but found %v", asset.FutureCombo, assetType)
			assert.NoError(t, err)
		})
	}
	cp, err := currency.NewPairFromString("some_thing_else")
	if err != nil {
		t.Fatal(err)
	}
	_, err = guessAssetTypeFromInstrument(cp)
	assert.ErrorIsf(t, err, errUnsupportedInstrumentFormat, "expected %v, but found %v", errUnsupportedInstrumentFormat, err)
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
	if !sharedtestvalues.AreAPICredentialsSet(d) {
		assert.False(t, feeBuilder.FeeType != exchange.OfflineTradeFee, "Expected %v, received %v", exchange.OfflineTradeFee, feeBuilder.FeeType)
	} else {
		assert.Falsef(t, feeBuilder.FeeType != exchange.CryptocurrencyTradeFee, "Expected %v, received %v", exchange.CryptocurrencyTradeFee, feeBuilder.FeeType)
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
	assert.NoError(t, err)
	assert.Falsef(t, result != 1e-1, "expected result %f, got %f", 1e-1, result)
	// futures
	feeBuilder.Pair, err = currency.NewPairFromString("BTC-21OCT22")
	if err != nil {
		t.Fatal(err)
	}
	result, err = calculateTradingFee(feeBuilder)
	assert.NoError(t, err)
	assert.Falsef(t, result != 0.1, "expected 0.1 but found %f", result)
	// options
	feeBuilder.Pair, err = currency.NewPairFromString("SOL-21OCT22-20-C")
	if err != nil {
		t.Fatal(err)
	}
	feeBuilder.IsMaker = false
	result, err = calculateTradingFee(feeBuilder)
	assert.NoError(t, err)
	assert.Falsef(t, result != 0.3, "expected 0.3 but found %f", result)
	// options
	feeBuilder.Pair, err = currency.NewPairFromString("SOL-21OCT22-20-C,SOL-21OCT22-20-P")
	if err != nil {
		t.Fatal(err)
	}
	feeBuilder.IsMaker = true
	_, err = calculateTradingFee(feeBuilder)
	assert.NoError(t, err)
	assert.Falsef(t, result != 0.3, "expected 0.3 but found %f", result)
	// option_combo
	feeBuilder.Pair, err = currency.NewPairFromString("BTC-STRG-21OCT22-19000_21000")
	if err != nil {
		t.Fatal(err)
	}
	feeBuilder.IsMaker = false
	_, err = calculateTradingFee(feeBuilder)
	assert.NoError(t, err)
	// future_combo
	feeBuilder.Pair, err = currency.NewPairFromString("SOL-FS-30DEC22_28OCT22")
	if err != nil {
		t.Fatal(err)
	}
	feeBuilder.IsMaker = false
	_, err = calculateTradingFee(feeBuilder)
	assert.NoError(t, err)
	feeBuilder.Pair, err = currency.NewPairFromString("some_instrument_builder")
	if err != nil {
		t.Fatal(err)
	}
	_, err = calculateTradingFee(feeBuilder)
	assert.ErrorIs(t, err, errUnsupportedInstrumentFormat, err)
}

func TestGetTime(t *testing.T) {
	t.Parallel()
	_, err := d.GetTime(context.Background())
	assert.NoError(t, err)
}

func TestWrapperGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := d.GetServerTime(context.Background(), asset.Empty)
	assert.NoError(t, err)
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := d.ModifyOrder(context.Background(), &order.Modify{OrderID: "1234"})
	assert.False(t, err != nil && !errors.Is(err, order.ErrPairIsEmpty))
	_, err = d.ModifyOrder(context.Background(), &order.Modify{AssetType: asset.Spot})
	assert.False(t, err != nil && !errors.Is(err, order.ErrPairIsEmpty), err)
	_, err = d.ModifyOrder(context.Background(), &order.Modify{AssetType: asset.Margin, OrderID: "1234", Pair: spotTradablePair})
	assert.False(t, err != nil && !errors.Is(err, asset.ErrNotSupported), err)
	_, err = d.ModifyOrder(context.Background(), &order.Modify{AssetType: asset.Futures, OrderID: "1234", Pair: futuresTradablePair})
	assert.False(t, err != nil && !errors.Is(err, errInvalidAmount), err)
	_, err = d.ModifyOrder(context.Background(), &order.Modify{AssetType: asset.Futures, Pair: futuresTradablePair, Amount: 2})
	assert.False(t, err != nil && !errors.Is(err, order.ErrOrderIDNotSet), err)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err = d.ModifyOrder(context.Background(), &order.Modify{AssetType: asset.Futures, OrderID: "1234", Pair: futuresTradablePair, Amount: 2})
	assert.NoError(t, err)
	sleepUntilTradablePairsUpdated()
	_, err = d.ModifyOrder(context.Background(), &order.Modify{AssetType: asset.Options, OrderID: "1234", Pair: optionsTradablePair, Amount: 2})
	assert.NoError(t, err)
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          futuresTradablePair,
		AssetType:     asset.Futures,
	}
	err := d.CancelOrder(context.Background(), orderCancellation)
	assert.NoError(t, err)
	sleepUntilTradablePairsUpdated()
	orderCancellation.AssetType = asset.Options
	orderCancellation.Pair = optionsTradablePair
	err = d.CancelOrder(context.Background(), orderCancellation)
	assert.NoError(t, err)
	orderCancellation.AssetType = asset.FutureCombo
	orderCancellation.Pair = futureComboTradablePair
	err = d.CancelOrder(context.Background(), orderCancellation)
	assert.NoError(t, err)
	orderCancellation.AssetType = asset.OptionCombo
	orderCancellation.Pair = optionComboTradablePair
	err = d.CancelOrder(context.Background(), orderCancellation)
	assert.NoError(t, err)
}

var websocketPushData = map[string]string{
	"Announcement":                           `{"jsonrpc": "2.0","method": "subscription","params": {"channel": "announcements","data": {            "action": "new",            "body": "Lorem ipsum dolor sit amet, consectetur adipiscing elit.",            "id": 1532593832021,            "important": true,            "publication_timestamp": 1532593832021,            "title": "Example announcement"        }    }}`,
	"Orderbook":                              `{"jsonrpc": "2.0", "method": "subscription", "params": { "channel": "book.BTC-PERPETUAL.100ms", "data": { "type": "snapshot", "timestamp": 1677589058217, "instrument_name": "BTC-PERPETUAL", "change_id": 53639437695, "bids": [ [ "new", 23461.0, 47800.0 ], [ "new", 23460.5, 37820.0 ], [ "new", 23460.0, 45720.0 ], [ "new", 23459.5, 24030.0 ], [ "new", 23459.0, 63600.0 ], [ "new", 23458.5, 60480.0 ], [ "new", 23458.0, 7960.0 ], [ "new", 23457.5, 2310.0 ], [ "new", 23457.0, 4270.0 ], [ "new", 23456.5, 44070.0 ], [ "new", 23456.0, 88690.0 ], [ "new", 23455.5, 5650.0 ], [ "new", 23455.0, 13420.0 ], [ "new", 23454.5, 116710.0 ], [ "new", 23454.0, 2010.0 ], [ "new", 23453.5, 200000.0 ], [ "new", 23452.5, 19950.0 ], [ "new", 23452.0, 39360.0 ], [ "new", 23451.5, 10000.0 ], [ "new", 23451.0, 239510.0 ], [ "new", 23450.5, 6250.0 ], [ "new", 23450.0, 40080.0 ], [ "new", 23449.5, 2000.0 ], [ "new", 23448.5, 500.0 ], [ "new", 23447.5, 179810.0 ], [ "new", 23447.0, 11000.0 ], [ "new", 23446.0, 57730.0 ], [ "new", 23445.0, 3640.0 ], [ "new", 23444.0, 17640.0 ], [ "new", 23443.5, 50000.0 ], [ "new", 23443.0, 6250.0 ], [ "new", 23441.5, 30330.0 ], [ "new", 23440.5, 76990.0 ], [ "new", 23440.0, 23910.0 ], [ "new", 23439.5, 3000.0 ], [ "new", 23439.0, 990.0 ], [ "new", 23438.0, 20760.0 ], [ "new", 23437.5, 500.0 ], [ "new", 23437.0, 84970.0 ], [ "new", 23436.5, 30040.0 ], [ "new", 23435.5, 322380.0 ], [ "new", 23434.0, 86280.0 ], [ "new", 23433.5, 187860.0 ], [ "new", 23433.0, 102360.0 ], [ "new", 23432.5, 48250.0 ], [ "new", 23432.0, 29070.0 ], [ "new", 23430.0, 119780.0 ], [ "new", 23429.0, 10.0 ], [ "new", 23428.0, 1510.0 ], [ "new", 23427.5, 2000.0 ], [ "new", 23427.0, 10.0 ], [ "new", 23426.5, 1840.0 ], [ "new", 23425.5, 2000.0 ], [ "new", 23425.0, 2250.0 ], [ "new", 23424.5, 600000.0 ], [ "new", 23424.0, 40870.0 ], [ "new", 23423.0, 117200.0 ], [ "new", 23422.0, 5000.0 ], [ "new", 23421.5, 80970.0 ], [ "new", 23420.0, 2420.0 ], [ "new", 23419.5, 200.0 ], [ "new", 23418.5, 40000.0 ], [ "new", 23415.0, 8020.0 ], [ "new", 23414.5, 57730.0 ], [ "new", 23413.5, 133250.0 ], [ "new", 23412.0, 40000.0 ], [ "new", 23410.5, 24000.0 ], [ "new", 23410.0, 80.0 ], [ "new", 23408.0, 36000.0 ], [ "new", 23407.0, 550000.0 ], [ "new", 23406.0, 30.0 ], [ "new", 23404.5, 230.0 ], [ "new", 23402.5, 57730.0 ], [ "new", 23401.0, 300010.0 ], [ "new", 23400.0, 520.0 ], [ "new", 23398.0, 28980.0 ], [ "new", 23394.5, 10.0 ], [ "new", 23391.5, 200.0 ], [ "new", 23391.0, 150000.0 ], [ "new", 23390.0, 80.0 ], [ "new", 23387.0, 403640.0 ], [ "new", 23385.5, 110.0 ], [ "new", 23385.0, 50.0 ], [ "new", 23384.5, 4690.0 ], [ "new", 23381.0, 200.0 ], [ "new", 23101.0, 9240.0 ], [ "new", 23100.5, 2320.0 ], [ "new", 23100.0, 15360.0 ], [ "new", 23096.0, 3000.0 ], [ "new", 23090.0, 90.0 ], [ "new", 23088.0, 3000.0 ], [ "new", 23087.0, 60.0 ], [ "new", 23081.5, 100.0 ], [ "new", 23080.0, 5400.0 ], [ "new", 23072.0, 3000.0 ], [ "new", 23070.0, 80.0 ], [ "new", 23064.0, 3000.0 ], [ "new", 23062.0, 3270.0 ], [ "new", 23060.0, 80.0 ], [ "new", 23056.0, 98000.0 ], [ "new", 23053.0, 3500.0 ], [ "new", 23050.5, 2370.0 ], [ "new", 23050.0, 32510.0 ], [ "new", 23048.0, 3000.0 ], [ "new", 23040.0, 3080.0 ], [ "new", 23038.0, 1000.0 ], [ "new", 23032.0, 5310.0 ], [ "new", 23030.0, 100.0 ], [ "new", 23024.0, 29000.0 ], [ "new", 23021.0, 2080.0 ], [ "new", 23020.0, 80.0 ], [ "new", 23016.0, 4150.0 ], [ "new", 23010.0, 80.0 ], [ "new", 23008.0, 3000.0 ], [ "new", 23005.0, 80.0 ], [ "new", 23004.5, 79200.0 ], [ "new", 23002.0, 20470.0 ], [ "new", 23001.0, 1000.0 ], [ "new", 23000.0, 8940.0 ], [ "new", 22992.0, 3000.0 ], [ "new", 22990.0, 2080.0 ], [ "new", 22984.0, 3000.0 ], [ "new", 22980.5, 2320.0 ], [ "new", 22980.0, 80.0 ], [ "new", 22976.0, 3000.0 ], [ "new", 22975.0, 52000.0 ], [ "new", 22971.0, 3600.0 ], [ "new", 22970.0, 2400.0 ], [ "new", 22968.0, 3000.0 ], [ "new", 22965.0, 270.0 ], [ "new", 22960.0, 3080.0 ], [ "new", 22956.0, 1000.0 ], [ "new", 22952.0, 3000.0 ], [ "new", 22951.0, 60.0 ], [ "new", 22950.0, 40200.0 ], [ "new", 22949.0, 1500.0 ], [ "new", 22944.0, 3000.0 ], [ "new", 22936.0, 3000.0 ], [ "new", 22934.0, 3000.0 ], [ "new", 22928.0, 3000.0 ], [ "new", 22925.0, 2370.0 ], [ "new", 22922.0, 80.0 ], [ "new", 22920.0, 3000.0 ], [ "new", 22916.0, 1150.0 ], [ "new", 22912.0, 3000.0 ], [ "new", 22904.5, 220.0 ], [ "new", 22904.0, 3000.0 ], [ "new", 22900.0, 273290.0 ], [ "new", 22896.0, 3000.0 ], [ "new", 22889.5, 100.0 ], [ "new", 22888.0, 7580.0 ], [ "new", 22880.0, 683400.0 ], [ "new", 22875.0, 400.0 ], [ "new", 22872.0, 3000.0 ], [ "new", 22870.0, 100.0 ], [ "new", 22864.0, 3000.0 ], [ "new", 22860.0, 2320.0 ], [ "new", 22856.0, 3000.0 ], [ "new", 22854.0, 10.0 ], [ "new", 22853.0, 500.0 ], [ "new", 22850.0, 1020.0 ], [ "new", 22848.0, 3000.0 ], [ "new", 22844.0, 25730.0 ], [ "new", 22840.0, 3000.0 ], [ "new", 22834.0, 3000.0 ], [ "new", 22832.0, 3000.0 ], [ "new", 22831.0, 200.0 ], [ "new", 22827.0, 40120.0 ], [ "new", 22824.0, 3000.0 ], [ "new", 22816.0, 4140.0 ], [ "new", 22808.0, 3000.0 ], [ "new", 22804.5, 220.0 ], [ "new", 22802.0, 50.0 ], [ "new", 22801.0, 1150.0 ], [ "new", 22800.0, 14050.0 ], [ "new", 22797.0, 10.0 ], [ "new", 22792.0, 3000.0 ], [ "new", 22789.0, 3000.0 ], [ "new", 22787.5, 5000.0 ], [ "new", 22784.0, 3000.0 ], [ "new", 22776.0, 3000.0 ], [ "new", 22775.0, 10000.0 ], [ "new", 22770.0, 200.0 ], [ "new", 22768.0, 14380.0 ], [ "new", 22760.0, 3000.0 ], [ "new", 22756.5, 2370.0 ], [ "new", 22752.0, 3000.0 ], [ "new", 22751.0, 47780.0 ], [ "new", 22750.0, 59970.0 ], [ "new", 22749.0, 50.0 ], [ "new", 22744.0, 3000.0 ], [ "new", 22736.0, 3000.0 ], [ "new", 22728.0, 3000.0 ], [ "new", 22726.0, 2320.0 ], [ "new", 22725.0, 20000.0 ], [ "new", 22720.0, 3000.0 ], [ "new", 22713.5, 250.0 ], [ "new", 22712.0, 3000.0 ], [ "new", 22709.0, 25000.0 ], [ "new", 22704.5, 220.0 ], [ "new", 22704.0, 3000.0 ], [ "new", 22702.0, 50.0 ], [ "new", 22700.0, 10230.0 ], [ "new", 22697.5, 10.0 ], [ "new", 22696.0, 3000.0 ], [ "new", 22688.0, 3000.0 ], [ "new", 22684.0, 10.0 ], [ "new", 22680.0, 3000.0 ], [ "new", 22672.0, 3000.0 ], [ "new", 22667.0, 2270.0 ], [ "new", 22664.0, 3000.0 ], [ "new", 22662.5, 2320.0 ], [ "new", 22657.5, 2340.0 ], [ "new", 22656.0, 3000.0 ], [ "new", 22655.0, 50.0 ], [ "new", 22653.0, 500.0 ], [ "new", 22650.0, 360120.0 ], [ "new", 22648.0, 3000.0 ], [ "new", 22640.0, 5320.0 ], [ "new", 22635.5, 2350.0 ], [ "new", 22632.0, 3000.0 ], [ "new", 22628.5, 2000.0 ], [ "new", 22626.5, 2350.0 ], [ "new", 22625.0, 400.0 ], [ "new", 22624.0, 3000.0 ], [ "new", 22616.0, 3000.0 ], [ "new", 22608.0, 3000.0 ], [ "new", 22604.5, 220.0 ], [ "new", 22601.0, 22600.0 ], [ "new", 22600.0, 696120.0 ], [ "new", 22598.5, 2320.0 ], [ "new", 22592.0, 3000.0 ], [ "new", 22584.0, 3000.0 ], [ "new", 22576.0, 3000.0 ], [ "new", 22568.0, 3000.0 ], [ "new", 22560.0, 25560.0 ], [ "new", 22552.5, 20.0 ], [ "new", 22550.0, 35760.0 ], [ "new", 22533.0, 2320.0 ], [ "new", 22530.0, 2320.0 ], [ "new", 22520.0, 1000.0 ], [ "new", 22505.0, 20000.0 ], [ "new", 22504.5, 220.0 ], [ "new", 22501.0, 45000.0 ], [ "new", 22500.0, 27460.0 ], [ "new", 22497.5, 1500.0 ], [ "new", 22485.0, 810.0 ], [ "new", 22481.0, 300.0 ], [ "new", 22465.5, 2320.0 ], [ "new", 22456.0, 2350.0 ], [ "new", 22453.0, 500.0 ], [ "new", 22450.0, 25000.0 ], [ "new", 22433.0, 141000.0 ], [ "new", 22431.0, 1940.0 ], [ "new", 22420.0, 2320.0 ], [ "new", 22419.5, 1000000.0 ], [ "new", 22400.0, 14280.0 ], [ "new", 22388.5, 30.0 ], [ "new", 22381.0, 100.0 ] ] } } }`,
	"Orderbook Update":                       `{"params" : {"data" : {"type" : "snapshot","timestamp" : 1554373962454,"instrument_name" : "BTC-PERPETUAL","change_id" : 297217,"bids" : [["new",5042.34,30],["new",5041.94,20]],"asks" : [["new",5042.64,40],["new",5043.3,40]]},"channel" : "book.BTC-PERPETUAL.100ms"},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"Candlestick":                            `{"params" : {"data" : {"volume" : 0.05219351,"tick" : 1573645080000,"open" : 8869.79,"low" : 8788.25,"high" : 8870.31,"cost" : 460,"close" : 8791.25},"channel" : "chart.trades.BTC-PERPETUAL.1"},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"Index Price":                            `{"params" : {"data" : {"timestamp" : 1550588002899,"price" : 3937.89,"index_name" : "btc_usd"},"channel" : "deribit_price_index.btc_usd"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"Price Ranking":                          `{"params" : {"data" :[{"weight" : 14.29,"timestamp" : 1573202284040,"price" : 9109.35,"original_price" : 9109.35,"identifier" : "bitfinex","enabled" : true},{"weight" : 14.29,"timestamp" : 1573202284055,"price" : 9084.83,"original_price" : 9084.83,"identifier" : "bitstamp","enabled" : true		},		{		  "weight" : 14.29,"timestamp" : 1573202283191,"price" : 9079.91,"original_price" : 9079.91,"identifier" : "bittrex","enabled" : true		},		{		  "weight" : 14.29,"timestamp" : 1573202284094,"price" : 9085.81,"original_price" : 9085.81,"identifier" : "coinbase","enabled" : true		},		{		  "weight" : 14.29,"timestamp" : 1573202283881,"price" : 9086.27,"original_price" : 9086.27,"identifier" : "gemini","enabled" : true		},		{		  "weight" : 14.29,"timestamp" : 1573202283420,"price" : 9088.38,"original_price" : 9088.38,"identifier" : "itbit","enabled" : true		},		{		  "weight" : 14.29,"timestamp" : 1573202283459,"price" : 9083.6,"original_price" : 9083.6,"identifier" : "kraken","enabled" : true		},		{		  "weight" : 0,"timestamp" : 0,"price" : null,"original_price" : null,"identifier" : "lmax","enabled" : false		}	  ],	  "channel" : "deribit_price_ranking.btc_usd"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"Price Statistics":                       `{"params" : {"data" : {"low24h" : 58012.08,"index_name" : "btc_usd","high_volatility" : false,"high24h" : 59311.42,"change24h" : 1009.61},"channel" : "deribit_price_statistics.btc_usd"},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"Volatility Index":                       `{"params" : {"data" : {"volatility" : 129.36,"timestamp" : 1619777946007,"index_name" : "btc_usd","estimated_delivery" : 129.36},"channel" : "deribit_volatility_index.btc_usd"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"Estimated Expiration Price":             `{"params" : {"data" : {"seconds" : 180929,"price" : 3939.73,"is_estimated" : false},"channel" : "estimated_expiration_price.btc_usd"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"Incremental Ticker":                     `{"jsonrpc": "2.0", "method": "subscription", "params": { "channel": "incremental_ticker.BTC-PERPETUAL", "data": { "type": "snapshot", "timestamp": 1677592580023, "stats": { "volume_usd": 224579520.0, "volume": 9581.70741368, "price_change": -1.2945, "low": 23123.5, "high": 23900.0 }, "state": "open", "settlement_price": 23240.71, "open_interest": 333091400, "min_price": 23057.4, "max_price": 23759.65, "mark_price": 23408.41, "last_price": 23409.0, "interest_value": 0.0, "instrument_name": "BTC-PERPETUAL", "index_price": 23406.85, "funding_8h": 0.0, "estimated_delivery_price": 23406.85, "current_funding": 0.0, "best_bid_price": 23408.5, "best_bid_amount": 53270.0, "best_ask_price": 23409.0, "best_ask_amount": 46990.0 } } }`,
	"Instrument State":                       `{"params" : {"data" : {"timestamp" : 1553080940000,"state" : "created","instrument_name" : "BTC-22MAR19"},"channel" : "instrument.state.any.any"},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"Currency Trades":                        `{"params":{"data":[{"trade_seq":2,"trade_id" : "48079289","timestamp" : 1590484589306,"tick_direction" : 2,"price" : 0.0075,"mark_price" : 0.01062686,"iv" : 47.58,"instrument_name" : "BTC-27MAY20-9000-C","index_price" : 8956.17,"direction" : "sell","amount" : 3}],"channel" : "trades.option.BTC.raw"},"method":"subscription","jsonrpc":"2.0"}`,
	"Change Updates":                         `{"params" : {"data" : {"trades" : [{"trade_seq" : 866638,"trade_id" : "1430914","timestamp" : 1605780344032,"tick_direction" : 1,"state" : "filled","self_trade" : false,"reduce_only" : false,"profit_loss" : 0.00004898,"price" : 17391,"post_only" : false,"order_type" : "market","order_id" : "3398016","matching_id" : null,"mark_price" : 17391,"liquidity" : "T","instrument_name" : "BTC-PERPETUAL","index_price" : 17501.88,"fee_currency" : "BTC","fee" : 1.6e-7,"direction" : "sell","amount" : 10		  }		],"positions" : [		  {			"total_profit_loss" : 1.69711368,			"size_currency" : 10.646886321,			"size" : 185160,			"settlement_price" : 16025.83,			"realized_profit_loss" : 0.012454598,			"realized_funding" : 0.01235663,			"open_orders_margin" : 0,			"mark_price" : 17391,			"maintenance_margin" : 0.234575865,			"leverage" : 33,			"kind" : "future",			"interest_value" : 1.7362511643080387,			"instrument_name" : "BTC-PERPETUAL",			"initial_margin" : 0.319750953,			"index_price" : 17501.88,			"floating_profit_loss" : 0.906961435,			"direction" : "buy",			"delta" : 10.646886321,			"average_price" : 15000		  }		],"orders" : [		  {			"web" : true,			"time_in_force" : "good_til_cancelled",			"replaced" : false,			"reduce_only" : false,			"profit_loss" : 0.00009166,			"price" : 15665.5,			"post_only" : false,			"order_type" : "market",			"order_state" : "filled",			"order_id" : "3398016",			"max_show" : 10,			"last_update_timestamp" : 1605780344032,			"label" : "",			"is_liquidation" : false,			"instrument_name" : "BTC-PERPETUAL",			"filled_amount" : 10,			"direction" : "sell",			"creation_timestamp" : 1605780344032,			"commission" : 1.6e-7,			"average_price" : 17391,			"api" : false,			"amount" : 10}],"instrument_name" : "BTC-PERPETUAL"	  },	  "channel" : "user.changes.BTC-PERPETUAL.raw"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"Currency Changes Updates":               `{"params" : {"data" : {"trades" : [{"trade_seq" : 866638,"trade_id" : "1430914","timestamp" : 1605780344032,"tick_direction" : 1,"state" : "filled","self_trade" : false,"reduce_only" : false,"profit_loss" : 0.00004898,"price" : 17391,"post_only" : false,"order_type" : "market","order_id" : "3398016","matching_id" : null,"mark_price" : 17391,"liquidity" : "T","instrument_name" : "BTC-PERPETUAL","index_price" : 17501.88,"fee_currency" : "BTC","fee" : 1.6e-7,"direction" : "sell","amount" : 10		  }		],"positions" : [		  {			"total_profit_loss" : 1.69711368,			"size_currency" : 10.646886321,			"size" : 185160,			"settlement_price" : 16025.83,			"realized_profit_loss" : 0.012454598,			"realized_funding" : 0.01235663,			"open_orders_margin" : 0,			"mark_price" : 17391,			"maintenance_margin" : 0.234575865,			"leverage" : 33,			"kind" : "future",			"interest_value" : 1.7362511643080387,			"instrument_name" : "BTC-PERPETUAL",			"initial_margin" : 0.319750953,			"index_price" : 17501.88,			"floating_profit_loss" : 0.906961435,			"direction" : "buy",			"delta" : 10.646886321,			"average_price" : 15000		  }		],"orders" : [		  {			"web" : true,			"time_in_force" : "good_til_cancelled",			"replaced" : false,			"reduce_only" : false,			"profit_loss" : 0.00009166,			"price" : 15665.5,			"post_only" : false,			"order_type" : "market",			"order_state" : "filled",			"order_id" : "3398016",			"max_show" : 10,			"last_update_timestamp" : 1605780344032,			"label" : "",			"is_liquidation" : false,			"instrument_name" : "BTC-PERPETUAL",			"filled_amount" : 10,			"direction" : "sell",			"creation_timestamp" : 1605780344032,			"commission" : 1.6e-7,			"average_price" : 17391,			"api" : false,			"amount" : 10		  }		],"instrument_name" : "BTC-PERPETUAL"	  },	  "channel" : "user.changes.future.BTC.raw"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"User Orders Raw Instrument":             `{"params" : {"data" : {"time_in_force" : "good_til_cancelled","replaced" : false,"reduce_only" : false,"profit_loss" : 0,"price" : 10502.52,"post_only" : false,"original_order_type" : "market","order_type" : "limit","order_state" : "open","order_id" : "5","max_show" : 200,"last_update_timestamp" : 1581507423789,"label" : "","is_liquidation" : false,"instrument_name" : "BTC-PERPETUAL","filled_amount" : 0,"direction" : "buy","creation_timestamp" : 1581507423789,"commission" : 0,"average_price" : 0,"api" : false,"amount" : 200	  },	  "channel" : "user.orders.BTC-PERPETUAL.raw"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"User Orders By Instrument WithInterval": `{"params" : {"data" : [{"time_in_force" : "good_til_cancelled","replaced" : false,"reduce_only" : false,"profit_loss" : 0,"price" : 10460.43,"post_only" : false,"original_order_type" : "market","order_type" : "limit","order_state" : "open","order_id" : "4","max_show" : 200,"last_update_timestamp" : 1581507159533,"label" : "","is_liquidation" : false,"instrument_name" : "BTC-PERPETUAL","filled_amount" : 0,"direction" : "buy","creation_timestamp" : 1581507159533,"commission" : 0,"average_price" : 0,"api" : false,"amount" : 200		}	  ],	  "channel" : "user.orders.BTC-PERPETUAL.100ms"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"User Order By Currency Raw":             `{"params" : {"data" : {"time_in_force" : "good_til_cancelled","replaced" : false,"reduce_only" : false,"profit_loss" : 0,"price" : 10542.68,"post_only" : false,"original_order_type" : "market","order_type" : "limit","order_state" : "open","order_id" : "6","max_show" : 200,"last_update_timestamp" : 1581507583024,"label" : "","is_liquidation" : false,"instrument_name" : "BTC-PERPETUAL","filled_amount" : 0,"direction" : "buy","creation_timestamp" : 1581507583024,"commission" : 0,"average_price" : 0,"api" : false,"amount" : 200	  },	  "channel" : "user.orders.any.any.raw"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"User Order By Currency WithInterval":    `{"params" : {"data" : [{"time_in_force" : "good_til_cancelled","reduce_only" : false,"profit_loss" : 0,"price" : 3928.5,"post_only" : false,"order_type" : "limit","order_state" : "open","order_id" : "476137","max_show" : 120,"last_update_timestamp" : 1550826337209,"label" : "","is_liquidation" : false,"instrument_name" : "BTC-PERPETUAL","filled_amount" : 0,"direction" : "buy","creation_timestamp" : 1550826337209,"commission" : 0,"average_price" : 0,"api" : false,"amount" : 120		}	  ],	  "channel" : "user.orders.future.BTC.100ms"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"User Portfolio":                         `{"params" : {"data" : {"total_pl" : 0.00000425,"session_upl" : 0.00000425,"session_rpl" : -2e-8,"projected_maintenance_margin" : 0.00009141,"projected_initial_margin" : 0.00012542,"projected_delta_total" : 0.0043,"portfolio_margining_enabled" : false,"options_vega" : 0,"options_value" : 0,"options_theta" : 0,"options_session_upl" : 0,"options_session_rpl" : 0,"options_pl" : 0,"options_gamma" : 0,"options_delta" : 0,"margin_balance" : 0.2340038,"maintenance_margin" : 0.00009141,"initial_margin" : 0.00012542,"futures_session_upl" : 0.00000425,"futures_session_rpl" : -2e-8,"futures_pl" : 0.00000425,"estimated_liquidation_ratio" : 0.01822795,"equity" : 0.2340038,"delta_total" : 0.0043,"currency" : "BTC","balance" : 0.23399957,"available_withdrawal_funds" : 0.23387415,"available_funds" : 0.23387838	  },	  "channel" : "user.portfolio.btc"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"User Trades":                            `{"params" : {"data" : [{"trade_seq" :30289432,"trade_id":"48079254","timestamp":1590484156350,"tick_direction" : 0,"state" : "filled","self_trade" : false,"reduce_only" : false,"price" : 8954,"post_only" : false,"order_type" : "market","order_id" : "4008965646","matching_id" : null,"mark_price" : 8952.86,"liquidity" : "T","instrument_name" : "BTC-PERPETUAL","index_price" : 8956.73,"fee_currency" : "BTC","fee" : 0.00000168,"direction" : "sell","amount" : 20		},		{		  "trade_seq" : 30289433,"trade_id" : "48079255","timestamp" : 1590484156350,"tick_direction" : 1,"state" : "filled","self_trade" : false,"reduce_only" : false,"price" : 8954,"post_only" : false,"order_type" : "market","order_id" : "4008965646","matching_id" : null,"mark_price" : 8952.86,"liquidity" : "T","instrument_name" : "BTC-PERPETUAL","index_price" : 8956.73,"fee_currency" : "BTC","fee" : 0.00000168,"direction" : "sell","amount" : 20	}],"channel" : "user.trades.BTC-PERPETUAL.raw"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"User Trades With Currency":              `{"params" : {"data" : [{"trade_seq" :74405,	"trade_id":"48079262","timestamp":1590484255886,"tick_direction" : 2,"state" : "filled","self_trade" : false,"reduce_only" : false,"price" : 8947,"post_only" : false,"order_type" : "limit","order_id" : "4008978075","matching_id" : null,"mark_price" : 8970.03,"liquidity" : "T","instrument_name" : "BTC-25SEP20","index_price" : 8953.53,"fee_currency" : "BTC","fee" : 0.00049961,"direction" : "sell","amount" : 8940		}	  ],	  "channel" : "user.trades.future.BTC.100ms"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"Mark Price Options":                     `{"params" : {"data" : [{"timestamp" : 1622470378005,"mark_price" : 0.0333,"iv" : 0.9,"instrument_name" : "BTC-2JUN21-37000-P"},{"timestamp" : 1622470378005,"mark_price" : 0.117,"iv" : 0.9,"instrument_name" : "BTC-4JUN21-40500-P"},{"timestamp" : 1622470378005,"mark_price" : 0.0177,"iv" : 0.9,"instrument_name" : "BTC-4JUN21-38250-C"},{"timestamp" : 1622470378005,"mark_price" : 0.0098,"iv" : 0.9,"instrument_name" : "BTC-1JUN21-37000-C"		},		{		  "timestamp" : 1622470378005,"mark_price" : 0.0371,"iv" : 0.9,"instrument_name" : "BTC-4JUN21-36500-P"		}	  ],	  "channel" : "markprice.options.btc_usd"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"Platform State":                         `{"params" : {"data" : {"allow_unauthenticated_public_requests" : true},"channel" : "platform_state.public_methods_state"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"Quote Ticker":                           `{"params" : {"data" : {"timestamp" : 1550658624149,"instrument_name" : "BTC-PERPETUAL","best_bid_price" : 3914.97,"best_bid_amount" : 40,"best_ask_price" : 3996.61,"best_ask_amount" : 50},"channel" : "quote.BTC-PERPETUAL"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"Rfq":                                    `{"params" : {"data" : {"state" : true,"side" : null,"last_rfq_tstamp" : 1634816143836,"instrument_name" : "BTC-PERPETUAL","amount" : null	  },"channel" : "rfq.btc"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
	"Instrument Trades":                      `{"params":{"data":[{"trade_seq":30289442,"trade_id" : "48079269","timestamp" : 1590484512188,"tick_direction" : 2,"price" : 8950,"mark_price" : 8948.9,"instrument_name" : "BTC-PERPETUAL","index_price" : 8955.88,"direction" : "sell","amount" : 10}],"channel" : "trades.BTC-PERPETUAL.raw"},"method":"subscription","jsonrpc":"2.0"}`,
	"Instruments Ticker":                     `{"params" : {"data" : {"timestamp" : 1623060194301,"stats" : {"volume_usd" : 284061480,"volume" : 7871.02139035,"price_change" : 0.7229,"low" : 35213.5,"high" : 36824.5},"state" : "open","settlement_price" : 36169.49,"open_interest" : 502097590,"min_price" : 35898.37,"max_price" : 36991.72,"mark_price" : 36446.51,"last_price" : 36457.5,"interest_value" : 1.7362511643080387,"instrument_name" : "BTC-PERPETUAL","index_price" : 36441.64,"funding_8h" : 0.0000211,"estimated_delivery_price" : 36441.64,"current_funding" : 0,"best_bid_price" : 36442.5,"best_bid_amount" : 5000,"best_ask_price" : 36443,"best_ask_amount" : 100	  },	  "channel" : "ticker.BTC-PERPETUAL.raw"	},	"method" : "subscription",	"jsonrpc" : "2.0"  }`,
}

func TestProcessPushData(t *testing.T) {
	t.Parallel()
	for x := range websocketPushData {
		err := d.wsHandleData([]byte(websocketPushData[x]))
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestFormatFuturesTradablePair(t *testing.T) {
	t.Parallel()
	futuresInstrumentsOutputList := map[currency.Pair]string{
		{Delimiter: currency.DashDelimiter, Base: currency.BTC, Quote: currency.NewCode("PERPETUAL")}:                   "BTC-PERPETUAL",
		{Delimiter: currency.DashDelimiter, Base: currency.AVAX, Quote: currency.NewCode("USDC-PERPETUAL")}:             "AVAX_USDC-PERPETUAL",
		{Delimiter: currency.DashDelimiter, Base: currency.ETH, Quote: currency.NewCode("30DEC22")}:                     "ETH-30DEC22",
		{Delimiter: currency.DashDelimiter, Base: currency.SOL, Quote: currency.NewCode("30DEC22")}:                     "SOL-30DEC22",
		{Delimiter: currency.DashDelimiter, Base: currency.NewCode("BTCDVOL"), Quote: currency.NewCode("USDC-28JUN23")}: "BTCDVOL_USDC-28JUN23",
	}
	for pair, instrumentID := range futuresInstrumentsOutputList {
		instrument := d.formatFuturesTradablePair(pair)
		if instrument != instrumentID {
			assert.FailNow(t, "found %s, but expected %s", instrument, instrumentID)
		}
	}
}

func TestWSRetrieveCombos(t *testing.T) {
	t.Parallel()
	sleepUntilTradablePairsUpdated()
	_, err := d.WSRetrieveCombos(futureComboTradablePair.Base.String())
	assert.NoError(t, err)
}

func TestWSSetPasswordForSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err := d.WSSetPasswordForSubAccount(123, "PassMe123@#")
	assert.NoError(t, err)
}

func instantiateTradablePairs() {
	d.Websocket.Wg.Add(1)
	go func(tpfChan chan struct{}) {
		defer d.Websocket.Wg.Done()
		var err error
		futuresTradablePair, err = currency.NewPairFromString(btcPerpInstrument)
		if err != nil {
			close(tpfChan)
			log.Fatal(err)
		}

		spotTradablePair, err = currency.NewPairFromString("BTC_USDC")
		if err != nil {
			close(tpfChan)
			log.Fatal(err)
		}
		assets := []asset.Item{asset.Futures, asset.Options, asset.OptionCombo, asset.FutureCombo}
		for x := range assets {
			// This loop only fetches tradable pairs of selected assets.
			var pairs currency.Pairs
			pairs, err = d.FetchTradablePairs(context.Background(), assets[x])
			if err != nil {
				close(tpfChan)
				log.Fatalf("%v, while fetching tradable pairs of asset type %v", err, assets[x])
			}
			err = d.UpdatePairs(pairs, assets[x], false, true)
			if err != nil {
				close(tpfChan)
				log.Fatalf("%v, while updating tradable pairs of asset type %v", err, assets[x])
			}
		}
		var tradablePair currency.Pairs
		tradablePair, err = d.GetEnabledPairs(asset.Options)
		if err != nil {
			close(tpfChan)
			log.Fatalf("failed to update tradable pairs. Err: %v", err)
		} else if len(tradablePair) == 0 {
			close(tpfChan)
			log.Fatalf("enabled %v for asset type %v", currency.ErrCurrencyPairsEmpty, asset.Options)
		}
		optionsTradablePair = tradablePair[0]
		tradablePair, err = d.GetEnabledPairs(asset.OptionCombo)
		if err != nil {
			close(tpfChan)
			log.Fatalf("failed to update tradable pairs. Err: %v", err)
		} else if len(tradablePair) == 0 {
			close(tpfChan)
			log.Fatalf("enabled %v for asset type %v", currency.ErrCurrencyPairsEmpty, asset.OptionCombo)
		}
		optionComboTradablePair = tradablePair[0]
		tradablePair, err = d.GetEnabledPairs(asset.FutureCombo)
		if err != nil {
			close(tpfChan)
			log.Fatalf("failed to update tradable pairs. Err: %v", err)
		} else if len(tradablePair) == 0 {
			close(tpfChan)
			log.Fatalf("enabled %v for asset type %v", currency.ErrCurrencyPairsEmpty, asset.FutureCombo)
		}
		futureComboTradablePair = tradablePair[0]
		tradablePairsFetchedStatusLock.Lock()
		tradablePairsFetched = true
		tradablePairsFetchedStatusLock.Unlock()
		close(tpfChan)
	}(fetchTradablePairChan)
}

func sleepUntilTradablePairsUpdated() {
	tradablePairsFetchedStatusLock.Lock()
	if tradablePairsFetched {
		tradablePairsFetchedStatusLock.Unlock()
		return
	}
	tradablePairsFetchedStatusLock.Unlock()
	<-fetchTradablePairChan
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := d.GetLatestFundingRates(context.Background(), &fundingrate.LatestRateRequest{
		Asset:                asset.USDTMarginedFutures,
		Pair:                 currency.NewPair(currency.BTC, currency.USDT),
		IncludePredictedRate: true,
	})
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	_, err = d.GetLatestFundingRates(context.Background(), &fundingrate.LatestRateRequest{
		Asset: asset.Futures,
		Pair:  futuresTradablePair,
	})
	assert.NoError(t, err)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := d.UpdateOrderExecutionLimits(context.Background(), asset.Spot)
	assert.NoErrorf(t, err, "Error fetching %s pairs for test: %v", asset.Spot, err)
	instrumentInfo, err := d.GetInstruments(context.Background(), "BTC", d.GetAssetKind(asset.Spot), false)
	assert.NoError(t, err)
	assert.False(t, len(instrumentInfo) == 0, "invalid instrument information found")
	limits, err := d.GetOrderExecutionLimits(asset.Spot, spotTradablePair)
	assert.NoErrorf(t, err, "Asset: %s Pair: %s Err: %v", asset.Spot, spotTradablePair, err)
	assert.Falsef(t, limits.PriceStepIncrementSize != instrumentInfo[0].TickSize, "Asset: %s Pair: %s Expected: %v Got: %v", asset.Spot, spotTradablePair, instrumentInfo[0].TickSize, limits.MinimumBaseAmount)
	assert.Falsef(t, limits.MinimumBaseAmount != instrumentInfo[0].MinimumTradeAmount, "Pair: %s Expected: %v Got: %v", spotTradablePair, instrumentInfo[0].MinimumTradeAmount, limits.MinimumBaseAmount)
}

func TestUnmarshalCancelResponse(t *testing.T) {
	t.Parallel()
	data := `{ "currency": "BTC", "type": "trigger", "instrument_name": "ETH-PERPETUAL", "result": [{ "web": true, "triggered": false, "trigger_price": 1628.7, "trigger": "last_price", "time_in_force": "good_til_cancelled", "stop_price": 1628.7, "replaced": false, "reduce_only": false, "price": "market_price", "post_only": false, "order_type": "stop_market", "order_state": "untriggered", "order_id": "ETH-SLTS-250756", "max_show": 100, "last_update_timestamp": 1634206091071, "label": "", "is_rebalance": false, "is_liquidation": false, "instrument_name": "ETH-PERPETUAL", "direction": "sell", "creation_timestamp": 1634206000230, "api": false, "amount": 100 }] }`
	var resp OrderCancelationResponse
	err := json.Unmarshal([]byte(data), &resp)
	assert.NoError(t, err)
}

func TestGetLockedStatus(t *testing.T) {
	t.Parallel()
	_, err := d.GetLockedStatus(context.Background())
	assert.NoError(t, err)
}

func TestSayHello(t *testing.T) {
	t.Parallel()
	_, err := d.SayHello("Sami", "")
	assert.NoError(t, err)
}

func TestGetCancelOnDisconnect(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.GetCancelOnDisconnect(context.Background(), "account")
	assert.NoError(t, err)
	_, err = d.WsRetrieveCancelOnDisconnect("connection")
	assert.NoError(t, err)
}

func TestDisableCancelOnDisconnect(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err := d.DisableCancelOnDisconnect(context.Background(), "connection")
	assert.NoError(t, err)
	_, err = d.WsDisableCancelOnDisconnect("connection")
	assert.NoError(t, err)
}

func TestEnableCancelOnDisconnect(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	_, err := d.EnableCancelOnDisconnect(context.Background(), "account")
	assert.NoError(t, err)
	_, err = d.WsEnableCancelOnDisconnect("connection")
	assert.NoError(t, err)
}

func TestLogout(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d, canManipulateRealOrders)
	err := d.WsLogout(true)
	assert.NoError(t, err)
}

func TestExchangeToken(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.ExchangeToken(context.Background(), "1568800656974.1CWcuzUS.MGy49NK4hpTwvR1OYWfpqMEkH4T4oDg4tNIcrM7KdeyxXRcSFqiGzA_D4Cn7mqWocHmlS89FFmUYcmaN2H7lNKKTnhRg5EtrzsFCCiuyN0Wv9y-LbGLV3-Ojv_kbD50FoScQ8BDXS5b_w6Ir1MqEdQ3qFZ3MLcvlPiIgG2BqyJX3ybYnVpIlrVrrdYD1-lkjLcjxOBNJvvUKNUAzkQ",
		1234)
	assert.NoError(t, err)
	_, err = d.WsExchangeToken("1568800656974.1CWcuzUS.MGy49NK4hpTwvR1OYWfpqMEkH4T4oDg4tNIcrM7KdeyxXRcSFqiGzA_D4Cn7mqWocHmlS89FFmUYcmaN2H7lNKKTnhRg5EtrzsFCCiuyN0Wv9y-LbGLV3-Ojv_kbD50FoScQ8BDXS5b_w6Ir1MqEdQ3qFZ3MLcvlPiIgG2BqyJX3ybYnVpIlrVrrdYD1-lkjLcjxOBNJvvUKNUAzkQ",
		1234)
	assert.NoError(t, err)
}

func TestForkToken(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, d)
	_, err := d.ForkToken(context.Background(), "1568800656974.1CWcuzUS.MGy49NK4hpTwvR1OYWfpqMEkH4T4oDg4tNIcrM7KdeyxXRcSFqiGzA_D4Cn7mqWocHmlS89FFmUYcmaN2H7lNKKTnhRg5EtrzsFCCiuyN0Wv9y-LbGLV3-Ojv_kbD50FoScQ8BDXS5b_w6Ir1MqEdQ3qFZ3MLcvlPiIgG2BqyJX3ybYnVpIlrVrrdYD1-lkjLcjxOBNJvvUKNUAzkQ", "Sami")
	assert.NoError(t, err)
	_, err = d.WsForkToken("1568800656974.1CWcuzUS.MGy49NK4hpTwvR1OYWfpqMEkH4T4oDg4tNIcrM7KdeyxXRcSFqiGzA_D4Cn7mqWocHmlS89FFmUYcmaN2H7lNKKTnhRg5EtrzsFCCiuyN0Wv9y-LbGLV3-Ojv_kbD50FoScQ8BDXS5b_w6Ir1MqEdQ3qFZ3MLcvlPiIgG2BqyJX3ybYnVpIlrVrrdYD1-lkjLcjxOBNJvvUKNUAzkQ", "Sami")
	assert.NoError(t, err)
}
