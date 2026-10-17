package deribit

import (
	"context"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

// GetAPRHistory returns a yield-bearing token's daily APRs as Exchange.GetAPRHistory does, over the websocket
func (w WebsocketAPI) GetAPRHistory(ctx context.Context, arg *APRHistoryRequest) (*APRHistoryResponse, error) {
	return w.e.getAPRHistory(ctx, arg, true)
}

// GetBasketIndexComposition returns a basket index's latest composition as Exchange.GetBasketIndexComposition does,
// over the websocket
func (w WebsocketAPI) GetBasketIndexComposition(ctx context.Context, indexName string) (*BasketIndexCompositionResponse, error) {
	return w.e.getBasketIndexComposition(ctx, indexName, true)
}

// GetBookSummaryByCurrency returns the market summaries of a currency's instruments as
// Exchange.GetBookSummaryByCurrency does, over the websocket
func (w WebsocketAPI) GetBookSummaryByCurrency(ctx context.Context, arg *BookSummaryByCurrencyRequest) ([]BookSummary, error) {
	return w.e.getBookSummaryByCurrency(ctx, arg, true)
}

// GetBookSummaryByInstrument returns an instrument's market summary as Exchange.GetBookSummaryByInstrument does, over
// the websocket
func (w WebsocketAPI) GetBookSummaryByInstrument(ctx context.Context, instrumentName string) ([]BookSummary, error) {
	return w.e.getBookSummaryByInstrument(ctx, instrumentName, true)
}

// GetContractSize returns an instrument's contract size as Exchange.GetContractSize does, over the websocket
func (w WebsocketAPI) GetContractSize(ctx context.Context, instrumentName string) (*ContractSizeResponse, error) {
	return w.e.getContractSize(ctx, instrumentName, true)
}

// GetCurrencies returns every currency Deribit supports as Exchange.GetCurrencies does, over the websocket
func (w WebsocketAPI) GetCurrencies(ctx context.Context) ([]CurrencyDetail, error) {
	return w.e.getCurrencies(ctx, true)
}

// GetDeliveryPrices returns an index's delivery prices as Exchange.GetDeliveryPrices does, over the websocket
func (w WebsocketAPI) GetDeliveryPrices(ctx context.Context, arg *DeliveryPricesRequest) (*DeliveryPricesResponse, error) {
	return w.e.getDeliveryPrices(ctx, arg, true)
}

// GetExpirations returns expiration labels as Exchange.GetExpirations does, over the websocket
func (w WebsocketAPI) GetExpirations(ctx context.Context, arg *ExpirationsRequest) (*ExpirationsResponse, error) {
	return w.e.getExpirations(ctx, arg, true)
}

// GetFundingChartData returns a perpetual's funding chart as Exchange.GetFundingChartData does, over the websocket
func (w WebsocketAPI) GetFundingChartData(ctx context.Context, arg *FundingChartDataRequest) (*FundingChartDataResponse, error) {
	return w.e.getFundingChartData(ctx, arg, true)
}

// GetFundingRateHistory returns a perpetual's hourly funding as Exchange.GetFundingRateHistory does, over the websocket
func (w WebsocketAPI) GetFundingRateHistory(ctx context.Context, arg *InstrumentTimeRangeRequest) ([]FundingRateHistory, error) {
	return w.e.getFundingRateHistory(ctx, arg, true)
}

// GetFundingRateValue returns a perpetual's funding rate over a time range as Exchange.GetFundingRateValue does, over
// the websocket
func (w WebsocketAPI) GetFundingRateValue(ctx context.Context, arg *InstrumentTimeRangeRequest) (float64, error) {
	return w.e.getFundingRateValue(ctx, arg, true)
}

// GetHistoricalVolatility returns a currency's realised volatility as Exchange.GetHistoricalVolatility does, over the
// websocket
func (w WebsocketAPI) GetHistoricalVolatility(ctx context.Context, ccy currency.Code) ([]HistoricalVolatilityPoint, error) {
	return w.e.getHistoricalVolatility(ctx, ccy, true)
}

// GetIndexChartData returns an index's price chart as Exchange.GetIndexChartData does, over the websocket
func (w WebsocketAPI) GetIndexChartData(ctx context.Context, arg *IndexChartDataRequest) ([]IndexChartPoint, error) {
	return w.e.getIndexChartData(ctx, arg, true)
}

// GetIndexPrice returns an index's current price as Exchange.GetIndexPrice does, over the websocket
func (w WebsocketAPI) GetIndexPrice(ctx context.Context, indexName string) (*IndexPriceResponse, error) {
	return w.e.getIndexPrice(ctx, indexName, true)
}

// GetIndexPriceNames returns the names of every price index as Exchange.GetIndexPriceNames does, over the websocket
func (w WebsocketAPI) GetIndexPriceNames(ctx context.Context, extended bool) ([]IndexPriceName, error) {
	return w.e.getIndexPriceNames(ctx, extended, true)
}

// GetInstrument returns an instrument's specification as Exchange.GetInstrument does, over the websocket
func (w WebsocketAPI) GetInstrument(ctx context.Context, instrumentName string) (*Instrument, error) {
	return w.e.getInstrument(ctx, instrumentName, true)
}

// GetInstruments returns a settlement currency's instruments as Exchange.GetInstruments does, over the websocket
func (w WebsocketAPI) GetInstruments(ctx context.Context, arg *InstrumentsRequest) ([]Instrument, error) {
	return w.e.getInstruments(ctx, arg, true)
}

// GetLastSettlementsByCurrency returns a currency's settlement events as Exchange.GetLastSettlementsByCurrency does,
// over the websocket
func (w WebsocketAPI) GetLastSettlementsByCurrency(ctx context.Context, arg *LastSettlementsByCurrencyRequest) (*SettlementsResponse, error) {
	return w.e.getLastSettlementsByCurrency(ctx, arg, true)
}

// GetLastSettlementsByInstrument returns an instrument's settlement events as Exchange.GetLastSettlementsByInstrument
// does, over the websocket
func (w WebsocketAPI) GetLastSettlementsByInstrument(ctx context.Context, arg *LastSettlementsByInstrumentRequest) (*SettlementsResponse, error) {
	return w.e.getLastSettlementsByInstrument(ctx, arg, true)
}

// GetLastTradesByCurrency returns a currency's latest trades as Exchange.GetLastTradesByCurrency does, over the
// websocket
func (w WebsocketAPI) GetLastTradesByCurrency(ctx context.Context, arg *LastTradesByCurrencyRequest) (*LastTradesResponse, error) {
	return w.e.getLastTradesByCurrency(ctx, arg, true)
}

// GetLastTradesByCurrencyAndTime returns a currency's latest trades within a time range as
// Exchange.GetLastTradesByCurrencyAndTime does, over the websocket
func (w WebsocketAPI) GetLastTradesByCurrencyAndTime(ctx context.Context, arg *LastTradesByCurrencyAndTimeRequest) (*LastTradesResponse, error) {
	return w.e.getLastTradesByCurrencyAndTime(ctx, arg, true)
}

// GetLastTradesByInstrument returns an instrument's latest trades as Exchange.GetLastTradesByInstrument does, over the
// websocket
func (w WebsocketAPI) GetLastTradesByInstrument(ctx context.Context, arg *LastTradesByInstrumentRequest) (*LastTradesResponse, error) {
	return w.e.getLastTradesByInstrument(ctx, arg, true)
}

// GetLastTradesByInstrumentAndTime returns an instrument's latest trades within a time range as
// Exchange.GetLastTradesByInstrumentAndTime does, over the websocket
func (w WebsocketAPI) GetLastTradesByInstrumentAndTime(ctx context.Context, arg *LastTradesByInstrumentAndTimeRequest) (*LastTradesResponse, error) {
	return w.e.getLastTradesByInstrumentAndTime(ctx, arg, true)
}

// GetMarkPriceHistory returns an option's mark prices as Exchange.GetMarkPriceHistory does, over the websocket
func (w WebsocketAPI) GetMarkPriceHistory(ctx context.Context, arg *InstrumentTimeRangeRequest) ([]MarkPricePoint, error) {
	return w.e.getMarkPriceHistory(ctx, arg, true)
}

// GetOrderBook returns an instrument's order book as Exchange.GetOrderBook does, over the websocket
func (w WebsocketAPI) GetOrderBook(ctx context.Context, arg *OrderBookRequest) (*OrderBookResponse, error) {
	return w.e.getOrderBook(ctx, arg, true)
}

// GetOrderBookByInstrumentID returns the order book of an instrument identified by its ID as
// Exchange.GetOrderBookByInstrumentID does, over the websocket
func (w WebsocketAPI) GetOrderBookByInstrumentID(ctx context.Context, arg *OrderBookByInstrumentIDRequest) (*OrderBookResponse, error) {
	return w.e.getOrderBookByInstrumentID(ctx, arg, true)
}

// GetSupportedIndexNames returns the names of the price indexes of a type as Exchange.GetSupportedIndexNames does, over
// the websocket
func (w WebsocketAPI) GetSupportedIndexNames(ctx context.Context, indexType string) ([]string, error) {
	return w.e.getSupportedIndexNames(ctx, indexType, true)
}

// GetTradeVolumes returns each currency's traded volumes as Exchange.GetTradeVolumes does, over the websocket
func (w WebsocketAPI) GetTradeVolumes(ctx context.Context, extended bool) ([]TradeVolume, error) {
	return w.e.getTradeVolumes(ctx, extended, true)
}

// GetTradingViewChartData returns an instrument's candles as Exchange.GetTradingViewChartData does, over the websocket
func (w WebsocketAPI) GetTradingViewChartData(ctx context.Context, arg *TradingViewChartDataRequest) (*TradingViewChartDataResponse, error) {
	return w.e.getTradingViewChartData(ctx, arg, true)
}

// GetVolatilityIndexData returns a currency's volatility index candles as Exchange.GetVolatilityIndexData does, over
// the websocket
func (w WebsocketAPI) GetVolatilityIndexData(ctx context.Context, arg *VolatilityIndexDataRequest) (*VolatilityIndexDataResponse, error) {
	return w.e.getVolatilityIndexData(ctx, arg, true)
}

// GetTicker returns an instrument's ticker as Exchange.GetTicker does, over the websocket
func (w WebsocketAPI) GetTicker(ctx context.Context, instrumentName string) (*TickerResponse, error) {
	return w.e.getTicker(ctx, instrumentName, true)
}
