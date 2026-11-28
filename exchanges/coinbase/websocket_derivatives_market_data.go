package coinbase

import (
	"context"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

// GetBookSummaryByCurrency returns the market summaries of a settlement currency's instruments as
// GlobalDerivativesAPI.GetBookSummaryByCurrency does, over the websocket
func (w WebsocketAPI) GetBookSummaryByCurrency(ctx context.Context, req *DerivativesBookSummaryByCurrencyRequest) ([]DerivativesBookSummaryResponse, error) {
	return w.e.GlobalDerivatives().getBookSummaryByCurrency(ctx, true, req)
}

// GetBookSummaryByInstrument returns an instrument's market summary as GlobalDerivativesAPI.GetBookSummaryByInstrument
// does, over the websocket
func (w WebsocketAPI) GetBookSummaryByInstrument(ctx context.Context, instrumentName string) ([]DerivativesBookSummaryResponse, error) {
	return w.e.GlobalDerivatives().getBookSummaryByInstrument(ctx, true, instrumentName)
}

// GetContractSize returns an instrument's contract size as GlobalDerivativesAPI.GetContractSize does, over the
// websocket
func (w WebsocketAPI) GetContractSize(ctx context.Context, instrumentName string) (*DerivativesContractSizeResponse, error) {
	return w.e.GlobalDerivatives().getContractSize(ctx, true, instrumentName)
}

// GetCurrencies returns every currency the gateway supports as GlobalDerivativesAPI.GetCurrencies does, over the
// websocket
func (w WebsocketAPI) GetCurrencies(ctx context.Context) ([]DerivativesCurrencyResponse, error) {
	return w.e.GlobalDerivatives().getCurrencies(ctx, true)
}

// GetDeliveryPrices returns a page of an index's delivery prices as GlobalDerivativesAPI.GetDeliveryPrices does, over
// the websocket
func (w WebsocketAPI) GetDeliveryPrices(ctx context.Context, req *DerivativesDeliveryPricesRequest) (*DerivativesDeliveryPricesResponse, error) {
	return w.e.GlobalDerivatives().getDeliveryPrices(ctx, true, req)
}

// GetExpirations returns expiration labels as GlobalDerivativesAPI.GetExpirations does, over the websocket
func (w WebsocketAPI) GetExpirations(ctx context.Context, req *DerivativesExpirationsRequest) (*DerivativesExpirationsResponse, error) {
	return w.e.GlobalDerivatives().getExpirations(ctx, true, req)
}

// GetFundingChartData returns a perpetual's funding chart as GlobalDerivativesAPI.GetFundingChartData does, over the
// websocket
func (w WebsocketAPI) GetFundingChartData(ctx context.Context, req *DerivativesFundingChartDataRequest) (*DerivativesFundingChartDataResponse, error) {
	return w.e.GlobalDerivatives().getFundingChartData(ctx, true, req)
}

// GetFundingRateHistory returns a perpetual's hourly funding as GlobalDerivativesAPI.GetFundingRateHistory does, over
// the websocket
func (w WebsocketAPI) GetFundingRateHistory(ctx context.Context, req *DerivativesInstrumentTimeRangeRequest) ([]DerivativesFundingRateHistoryResponse, error) {
	return w.e.GlobalDerivatives().getFundingRateHistory(ctx, true, req)
}

// GetFundingRateValue returns a perpetual's funding rate over a time range as GlobalDerivativesAPI.GetFundingRateValue
// does, over the websocket
func (w WebsocketAPI) GetFundingRateValue(ctx context.Context, req *DerivativesInstrumentTimeRangeRequest) (float64, error) {
	return w.e.GlobalDerivatives().getFundingRateValue(ctx, true, req)
}

// GetHistoricalVolatility returns a currency's realised volatility as GlobalDerivativesAPI.GetHistoricalVolatility
// does, over the websocket
func (w WebsocketAPI) GetHistoricalVolatility(ctx context.Context, ccy currency.Code) ([]DerivativesHistoricalVolatilityResponse, error) {
	return w.e.GlobalDerivatives().getHistoricalVolatility(ctx, true, ccy)
}

// GetIndexChartData returns an index's price chart as GlobalDerivativesAPI.GetIndexChartData does, over the websocket
func (w WebsocketAPI) GetIndexChartData(ctx context.Context, req *DerivativesIndexChartDataRequest) ([]DerivativesIndexChartPointResponse, error) {
	return w.e.GlobalDerivatives().getIndexChartData(ctx, true, req)
}

// GetIndexPrice returns an index's current price as GlobalDerivativesAPI.GetIndexPrice does, over the websocket
func (w WebsocketAPI) GetIndexPrice(ctx context.Context, indexName string) (*DerivativesIndexPriceResponse, error) {
	return w.e.GlobalDerivatives().getIndexPrice(ctx, true, indexName)
}

// GetIndexPriceNames returns the names of every price index as GlobalDerivativesAPI.GetIndexPriceNames does, over the
// websocket
func (w WebsocketAPI) GetIndexPriceNames(ctx context.Context, extended bool) ([]DerivativesIndexPriceNameResponse, error) {
	return w.e.GlobalDerivatives().getIndexPriceNames(ctx, true, extended)
}

// GetInstrument returns an instrument's specification as GlobalDerivativesAPI.GetInstrument does, over the websocket
func (w WebsocketAPI) GetInstrument(ctx context.Context, instrumentName string) (*DerivativesInstrumentResponse, error) {
	return w.e.GlobalDerivatives().getInstrument(ctx, true, instrumentName)
}

// GetInstruments returns a settlement currency's instruments as GlobalDerivativesAPI.GetInstruments does, over the
// websocket
func (w WebsocketAPI) GetInstruments(ctx context.Context, req *DerivativesInstrumentsRequest) ([]DerivativesInstrumentResponse, error) {
	return w.e.GlobalDerivatives().getInstruments(ctx, true, req)
}

// GetLastSettlementsByCurrency returns a page of a settlement currency's settlement events as
// GlobalDerivativesAPI.GetLastSettlementsByCurrency does, over the websocket
func (w WebsocketAPI) GetLastSettlementsByCurrency(ctx context.Context, req *DerivativesLastSettlementsByCurrencyRequest) (*DerivativesSettlementsResponse, error) {
	return w.e.GlobalDerivatives().getLastSettlementsByCurrency(ctx, true, req)
}

// GetLastSettlementsByInstrument returns a page of an instrument's settlement events as
// GlobalDerivativesAPI.GetLastSettlementsByInstrument does, over the websocket
func (w WebsocketAPI) GetLastSettlementsByInstrument(ctx context.Context, req *DerivativesLastSettlementsByInstrumentRequest) (*DerivativesSettlementsResponse, error) {
	return w.e.GlobalDerivatives().getLastSettlementsByInstrument(ctx, true, req)
}

// GetLastTradesByCurrency returns a settlement currency's latest trades as GlobalDerivativesAPI.GetLastTradesByCurrency
// does, over the websocket
func (w WebsocketAPI) GetLastTradesByCurrency(ctx context.Context, req *DerivativesLastTradesByCurrencyRequest) (*DerivativesLastTradesResponse, error) {
	return w.e.GlobalDerivatives().getLastTradesByCurrency(ctx, true, req)
}

// GetLastTradesByCurrencyAndTime returns a settlement currency's latest trades within a time range as
// GlobalDerivativesAPI.GetLastTradesByCurrencyAndTime does, over the websocket
func (w WebsocketAPI) GetLastTradesByCurrencyAndTime(ctx context.Context, req *DerivativesLastTradesByCurrencyAndTimeRequest) (*DerivativesLastTradesResponse, error) {
	return w.e.GlobalDerivatives().getLastTradesByCurrencyAndTime(ctx, true, req)
}

// GetLastTradesByInstrument returns an instrument's latest trades as GlobalDerivativesAPI.GetLastTradesByInstrument
// does, over the websocket
func (w WebsocketAPI) GetLastTradesByInstrument(ctx context.Context, req *DerivativesLastTradesByInstrumentRequest) (*DerivativesLastTradesResponse, error) {
	return w.e.GlobalDerivatives().getLastTradesByInstrument(ctx, true, req)
}

// GetLastTradesByInstrumentAndTime returns an instrument's latest trades within a time range as
// GlobalDerivativesAPI.GetLastTradesByInstrumentAndTime does, over the websocket
func (w WebsocketAPI) GetLastTradesByInstrumentAndTime(ctx context.Context, req *DerivativesLastTradesByInstrumentAndTimeRequest) (*DerivativesLastTradesResponse, error) {
	return w.e.GlobalDerivatives().getLastTradesByInstrumentAndTime(ctx, true, req)
}

// GetMarkPriceHistory returns an option's mark prices as GlobalDerivativesAPI.GetMarkPriceHistory does, over the
// websocket
func (w WebsocketAPI) GetMarkPriceHistory(ctx context.Context, req *DerivativesInstrumentTimeRangeRequest) ([]DerivativesMarkPriceResponse, error) {
	return w.e.GlobalDerivatives().getMarkPriceHistory(ctx, true, req)
}

// GetOrderBook returns an instrument's order book as GlobalDerivativesAPI.GetOrderBook does, over the websocket
func (w WebsocketAPI) GetOrderBook(ctx context.Context, req *DerivativesOrderBookRequest) (*DerivativesOrderBookResponse, error) {
	return w.e.GlobalDerivatives().getOrderBook(ctx, true, req)
}

// GetOrderBookByInstrumentID returns the order book of an instrument identified by its ID as
// GlobalDerivativesAPI.GetOrderBookByInstrumentID does, over the websocket
func (w WebsocketAPI) GetOrderBookByInstrumentID(ctx context.Context, req *DerivativesOrderBookByInstrumentIDRequest) (*DerivativesOrderBookResponse, error) {
	return w.e.GlobalDerivatives().getOrderBookByInstrumentID(ctx, true, req)
}

// GetSupportedIndexNames returns the names of the price indexes of a type as
// GlobalDerivativesAPI.GetSupportedIndexNames does, over the websocket
func (w WebsocketAPI) GetSupportedIndexNames(ctx context.Context, indexType string) ([]string, error) {
	return w.e.GlobalDerivatives().getSupportedIndexNames(ctx, true, indexType)
}

// GetTradeVolumes returns each currency's traded volumes as GlobalDerivativesAPI.GetTradeVolumes does, over the
// websocket
func (w WebsocketAPI) GetTradeVolumes(ctx context.Context, extended bool) ([]DerivativesTradeVolumeResponse, error) {
	return w.e.GlobalDerivatives().getTradeVolumes(ctx, true, extended)
}

// GetTradingViewChartData returns an instrument's candles as GlobalDerivativesAPI.GetTradingViewChartData does, over
// the websocket, where the gateway takes it as any other method
func (w WebsocketAPI) GetTradingViewChartData(ctx context.Context, req *DerivativesTradingViewChartDataRequest) (*DerivativesTradingViewChartDataResponse, error) {
	return w.e.GlobalDerivatives().getTradingViewChartData(ctx, true, req)
}

// GetVolatilityIndexData returns a currency's volatility index candles as GlobalDerivativesAPI.GetVolatilityIndexData
// does, over the websocket
func (w WebsocketAPI) GetVolatilityIndexData(ctx context.Context, req *DerivativesVolatilityIndexDataRequest) (*DerivativesVolatilityIndexDataResponse, error) {
	return w.e.GlobalDerivatives().getVolatilityIndexData(ctx, true, req)
}

// GetTicker returns an instrument's ticker as GlobalDerivativesAPI.GetTicker does, over the websocket
func (w WebsocketAPI) GetTicker(ctx context.Context, instrumentName string) (*DerivativesTickerResponse, error) {
	return w.e.GlobalDerivatives().getTicker(ctx, true, instrumentName)
}

// GetTickersByCurrency returns the tickers of a settlement currency's open instruments as
// GlobalDerivativesAPI.GetTickersByCurrency does, over the websocket
func (w WebsocketAPI) GetTickersByCurrency(ctx context.Context, req *DerivativesTickersByCurrencyRequest) ([]DerivativesOrderBookResponse, error) {
	return w.e.GlobalDerivatives().getTickersByCurrency(ctx, true, req)
}
