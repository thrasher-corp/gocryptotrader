package deribit

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// WSRetrieveBookBySummary retrieves book summary data for currency requested through websocket connection.
func (d *Deribit) WSRetrieveBookBySummary(ccy currency.Code, kind string) ([]BookSummaryData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	input := &struct {
		Currency currency.Code `json:"currency"`
		Kind     string        `json:"kind,omitempty"`
	}{
		Currency: ccy,
	}
	if kind != "" {
		input.Kind = kind
	}
	var resp []BookSummaryData
	return resp, d.SendWSRequest(nonMatchingEPL, getBookByCurrency, input, &resp, false)
}

// WSRetrieveBookSummaryByInstrument retrieves book summary data for instrument requested through the websocket connection.
func (d *Deribit) WSRetrieveBookSummaryByInstrument(instrument string) ([]BookSummaryData, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	input := &struct {
		Instrument string `json:"instrument_name,omitempty"`
	}{
		Instrument: instrument,
	}
	var resp []BookSummaryData
	return resp, d.SendWSRequest(nonMatchingEPL, getBookByInstrument, input, &resp, false)
}

// WSRetrieveContractSize retrieves contract size for instrument requested through the websocket connection.
func (d *Deribit) WSRetrieveContractSize(instrument string) (*ContractSizeData, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	input := &struct {
		Instrument string `json:"instrument_name"`
	}{
		Instrument: instrument,
	}
	var resp *ContractSizeData
	return resp, d.SendWSRequest(nonMatchingEPL, getContractSize, input, &resp, false)
}

// WSRetrieveCurrencies retrieves all cryptocurrencies supported by the API through the websocket connection.
func (d *Deribit) WSRetrieveCurrencies() ([]CurrencyData, error) {
	var resp []CurrencyData
	return resp, d.SendWSRequest(nonMatchingEPL, getCurrencies, nil, &resp, false)
}

// WSRetrieveDeliveryPrices retrieves delivery prices using index name through the websocket connection.
func (d *Deribit) WSRetrieveDeliveryPrices(indexName string, offset, count int64) (*IndexDeliveryPrice, error) {
	if indexName == "" {
		return nil, errUnsupportedIndexName
	}
	input := &struct {
		IndexName string `json:"index_name"`
		Offset    int64  `json:"offset,omitempty"`
		Count     int64  `json:"count,omitempty"`
	}{
		IndexName: indexName,
		Offset:    offset,
		Count:     count,
	}
	var resp *IndexDeliveryPrice
	return resp, d.SendWSRequest(nonMatchingEPL, getDeliveryPrices, input, &resp, false)
}

// WSRetrieveFundingChartData retrieves funding chart data for the requested instrument and time length through the websocket connection.
// supported lengths: 8h, 24h, 1m <-(1month)
func (d *Deribit) WSRetrieveFundingChartData(instrument, length string) (*FundingChartData, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	if length == "" {
		return nil, errIntervalNotSupported
	}
	input := &struct {
		InstrumentName string `json:"instrument_name"`
		Length         string `json:"length"`
	}{
		InstrumentName: instrument,
		Length:         length,
	}
	var resp *FundingChartData
	return resp, d.SendWSRequest(nonMatchingEPL, getFundingChartData, input, &resp, false)
}

// WSRetrieveFundingRateHistory retrieves hourly historical interest rate for requested PERPETUAL instrument through the websocket connection.
func (d *Deribit) WSRetrieveFundingRateHistory(instrumentName string, startTime, endTime time.Time) ([]FundingRateHistory, error) {
	if instrumentName == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	err := common.StartEndTimeCheck(startTime, endTime)
	if err != nil {
		return nil, err
	}
	input := &struct {
		InstrumentName string `json:"instrument_name"`
		StartTime      int64  `json:"start_timestamp"`
		EndTime        int64  `json:"end_timestamp"`
	}{
		InstrumentName: instrumentName,
		StartTime:      startTime.UnixMilli(),
		EndTime:        endTime.UnixMilli(),
	}
	var resp []FundingRateHistory
	return resp, d.SendWSRequest(nonMatchingEPL, getFundingRateHistory, input, &resp, false)
}

// WSRetrieveFundingRateValue retrieves funding rate value data through the websocket connection.
func (d *Deribit) WSRetrieveFundingRateValue(instrument string, startTime, endTime time.Time) (float64, error) {
	if instrument == "" {
		return 0, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
		return 0, err
	}
	input := &struct {
		Instrument     string `json:"instrument_name"`
		StartTimestamp int64  `json:"start_timestamp"`
		EndTimestamp   int64  `json:"end_timestamp"`
	}{
		Instrument:     instrument,
		StartTimestamp: startTime.UnixMilli(),
		EndTimestamp:   endTime.UnixMilli(),
	}
	var resp float64
	return resp, d.SendWSRequest(nonMatchingEPL, getFundingRateValue, input, &resp, false)
}

// WSRetrieveHistoricalVolatility retrieves historical volatility data
func (d *Deribit) WSRetrieveHistoricalVolatility(ccy currency.Code) ([]HistoricalVolatilityData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	input := &struct {
		Currency currency.Code `json:"currency"`
	}{
		Currency: ccy,
	}
	var data [][2]any
	err := d.SendWSRequest(nonMatchingEPL, getHistoricalVolatility, input, &data, false)
	if err != nil {
		return nil, err
	}
	resp := make([]HistoricalVolatilityData, len(data))
	for x := range data {
		timeData, ok := data[x][0].(float64)
		if !ok {
			return resp, common.GetTypeAssertError("float64", data[x][0], "time data")
		}
		val, ok := data[x][1].(float64)
		if !ok {
			return resp, common.GetTypeAssertError("float64", data[x][1], "volatility value")
		}
		resp[x] = HistoricalVolatilityData{
			Timestamp: timeData,
			Value:     val,
		}
	}
	return resp, nil
}

// WSRetrieveCurrencyIndexPrice the current index price for the instruments, for the selected currency through the websocket connection.
func (d *Deribit) WSRetrieveCurrencyIndexPrice(ccy currency.Code) (map[string]float64, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	input := &struct {
		Currency currency.Code `json:"currency"`
	}{
		Currency: ccy,
	}
	var resp map[string]float64
	return resp, d.SendWSRequest(nonMatchingEPL, getCurrencyIndexPrice, input, &resp, false)
}

// WSRetrieveIndexPrice retrieves price data for the requested index through the websocket connection.
func (d *Deribit) WSRetrieveIndexPrice(index string) (*IndexPriceData, error) {
	if index == "" {
		return nil, fmt.Errorf("%w index can not be empty", errUnsupportedIndexName)
	}
	input := &struct {
		IndexName string `json:"index_name"`
	}{
		IndexName: index,
	}
	var resp *IndexPriceData
	return resp, d.SendWSRequest(nonMatchingEPL, getIndexPrice, input, &resp, false)
}

// WSRetrieveIndexPriceNames names of indexes through the websocket connection.
func (d *Deribit) WSRetrieveIndexPriceNames() ([]string, error) {
	var resp []string
	return resp, d.SendWSRequest(nonMatchingEPL, getIndexPriceNames, nil, &resp, false)
}

// WSRetrieveInstrumentData retrieves data for a requested instrument through the websocket connection.
func (d *Deribit) WSRetrieveInstrumentData(instrument string) (*InstrumentData, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	input := &struct {
		Instrument string `json:"instrument_name"`
	}{
		Instrument: instrument,
	}
	var resp *InstrumentData
	return resp, d.SendWSRequest(nonMatchingEPL, getInstrument, input, &resp, false)
}

// WSRetrieveInstrumentsData gets data for all available instruments
func (d *Deribit) WSRetrieveInstrumentsData(ccy currency.Code, kind string, expired bool) ([]InstrumentData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	input := &struct {
		Currency currency.Code `json:"currency"`
		Expired  bool          `json:"expired"`
		Kind     string        `json:"kind,omitempty"`
	}{
		Currency: ccy,
		Expired:  expired,
		Kind:     kind,
	}
	var resp []InstrumentData
	return resp, d.SendWSRequest(nonMatchingEPL, getInstruments, input, &resp, false)
}

// WSRetrieveLastSettlementsByCurrency retrieves last settlement data by currency through the websocket connection.
func (d *Deribit) WSRetrieveLastSettlementsByCurrency(ccy currency.Code, settlementType, continuation string, count int64, startTime time.Time) (*SettlementsData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	input := &struct {
		Currency             string `json:"currency,omitempty"`
		Type                 string `json:"type,omitempty"`
		Continuation         string `json:"continuation,omitempty"`
		Count                int64  `json:"count,omitempty"`
		SearchStartTimestamp int64  `json:"search_start_timestamp,omitempty"`
	}{
		Currency:             ccy.String(),
		Type:                 settlementType,
		Continuation:         continuation,
		Count:                count,
		SearchStartTimestamp: startTime.UnixMilli(),
	}
	var resp *SettlementsData
	return resp, d.SendWSRequest(nonMatchingEPL, getLastSettlementsByCurrency, input, &resp, false)
}

// WSRetrieveLastSettlementsByInstrument retrieves last settlement data for requested instrument through the websocket connection.
func (d *Deribit) WSRetrieveLastSettlementsByInstrument(instrument, settlementType, continuation string, count int64, startTime time.Time) (*SettlementsData, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	input := &struct {
		Instrument           string `json:"instrument_name"`
		SettlementType       string `json:"type,omitempty"`
		Continuation         string `json:"continuation,omitempty"`
		Count                int64  `json:"count,omitempty"`
		SearchStartTimestamp int64  `json:"search_start_timestamp,omitempty"`
	}{
		Instrument:     instrument,
		SettlementType: settlementType,
		Continuation:   continuation,
		Count:          count,
	}
	if !startTime.IsZero() {
		input.SearchStartTimestamp = startTime.UnixMilli()
	}
	var resp *SettlementsData
	return resp, d.SendWSRequest(nonMatchingEPL, getLastSettlementsByInstrument, input, &resp, false)
}

// WSRetrieveLastTradesByCurrency retrieves last trades for requested currency through the websocket connection.
func (d *Deribit) WSRetrieveLastTradesByCurrency(ccy currency.Code, kind, startID, endID, sorting string, count int64, includeOld bool) (*PublicTradesData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	input := &struct {
		Currency   currency.Code `json:"currency"`
		Kind       string        `json:"kind,omitempty"`
		StartID    string        `json:"start_id,omitempty"`
		EndID      string        `json:"end_id,omitempty"`
		Count      int64         `json:"count,omitempty"`
		IncludeOld bool          `json:"include_old,omitempty"`
		Sorting    string        `json:"sorting,omitempty"`
	}{
		Currency:   ccy,
		Kind:       kind,
		StartID:    startID,
		EndID:      endID,
		Count:      count,
		IncludeOld: includeOld,
		Sorting:    sorting,
	}
	var resp *PublicTradesData
	return resp, d.SendWSRequest(nonMatchingEPL, getLastTradesByCurrency, input, &resp, false)
}

// WSRetrieveLastTradesByCurrencyAndTime retrieves last trades for requested currency and time intervals through the websocket connection.
func (d *Deribit) WSRetrieveLastTradesByCurrencyAndTime(ccy currency.Code, kind, sorting string, count int64, includeOld bool, startTime, endTime time.Time) (*PublicTradesData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
		return nil, err
	}
	input := &struct {
		Currency       currency.Code `json:"currency"`
		Kind           string        `json:"kind,omitempty"`
		Sorting        string        `json:"sorting,omitempty"`
		Count          int64         `json:"count,omitempty"`
		StartTimestamp int64         `json:"start_timestamp,omitempty"`
		EndTimestamp   int64         `json:"end_timestamp,omitempty"`
		IncludeOld     bool          `json:"include_old,omitempty"`
	}{
		Currency:       ccy,
		Kind:           kind,
		Count:          count,
		StartTimestamp: startTime.UnixMilli(),
		EndTimestamp:   endTime.UnixMilli(),
		IncludeOld:     includeOld,
		Sorting:        sorting,
	}
	var resp *PublicTradesData
	return resp, d.SendWSRequest(nonMatchingEPL, getLastTradesByCurrencyAndTime, input, &resp, false)
}

// WSRetrieveLastTradesByInstrument retrieves last trades for requested instrument requested through the websocket connection.
func (d *Deribit) WSRetrieveLastTradesByInstrument(instrument, startSeq, endSeq, sorting string, count int64, includeOld bool) (*PublicTradesData, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	input := &struct {
		Instrument    string `json:"instrument_name,omitempty"`
		StartSequence string `json:"start_seq,omitempty"`
		EndSequence   string `json:"end_seq,omitempty"`
		Sorting       string `json:"sorting,omitempty"`
		Count         int64  `json:"count,omitempty"`
		IncludeOld    bool   `json:"include_old,omitempty"`
	}{
		Instrument:    instrument,
		StartSequence: startSeq,
		EndSequence:   endSeq,
		Sorting:       sorting,
		Count:         count,
		IncludeOld:    includeOld,
	}
	var resp *PublicTradesData
	return resp, d.SendWSRequest(nonMatchingEPL, getLastTradesByInstrument, input, &resp, false)
}

// WSRetrieveLastTradesByInstrumentAndTime retrieves last trades for requested instrument requested and time intervals through the websocket connection.
func (d *Deribit) WSRetrieveLastTradesByInstrumentAndTime(instrument, sorting string, count int64, includeOld bool, startTime, endTime time.Time) (*PublicTradesData, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
		return nil, err
	}
	input := &struct {
		Instrument     string `json:"instrument_name,omitempty"`
		StartTimestamp int64  `json:"start_timestamp,omitempty"`
		EndTimestamp   int64  `json:"end_timestamp,omitempty"`
		Sorting        string `json:"sorting,omitempty"`
		Count          int64  `json:"count,omitempty"`
		IncludeOld     bool   `json:"include_old,omitempty"`
	}{
		Instrument: instrument,
		Sorting:    sorting,
		Count:      count,
		IncludeOld: includeOld,
	}
	input.StartTimestamp = startTime.UnixMilli()
	input.EndTimestamp = endTime.UnixMilli()
	var resp *PublicTradesData
	return resp, d.SendWSRequest(nonMatchingEPL, getLastTradesByInstrumentAndTime, input, &resp, false)
}

// WSRetrieveMarkPriceHistory retrieves data for mark price history through the websocket connection.
func (d *Deribit) WSRetrieveMarkPriceHistory(instrument string, startTime, endTime time.Time) ([]MarkPriceHistory, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
		return nil, err
	}
	input := &struct {
		Instrument     string `json:"instrument_name,omitempty"`
		StartTimestamp int64  `json:"start_timestamp,omitempty"`
		EndTimestamp   int64  `json:"end_timestamp,omitempty"`
	}{
		Instrument:     instrument,
		StartTimestamp: startTime.UnixMilli(),
		EndTimestamp:   endTime.UnixMilli(),
	}
	var resp []MarkPriceHistory
	return resp, d.SendWSRequest(nonMatchingEPL, getMarkPriceHistory, input, &resp, false)
}

// WSRetrieveOrderbookData retrieves data orderbook of requested instrument through the web-socket connection.
func (d *Deribit) WSRetrieveOrderbookData(instrument string, depth int64) (*Orderbook, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	input := &struct {
		Instrument string `json:"instrument_name"`
		Depth      int64  `json:"depth,omitempty"`
	}{
		Instrument: instrument,
		Depth:      depth,
	}
	var resp *Orderbook
	return resp, d.SendWSRequest(nonMatchingEPL, getOrderbook, input, &resp, false)
}

// WSRetrieveOrderbookByInstrumentID retrieves orderbook by instrument ID through websocket connection.
func (d *Deribit) WSRetrieveOrderbookByInstrumentID(instrumentID int64, depth float64) (*Orderbook, error) {
	if instrumentID == 0 {
		return nil, errInvalidInstrumentID
	}
	input := &struct {
		InstrumentID int64   `json:"instrument_id"`
		Depth        float64 `json:"depth,omitempty"`
	}{
		InstrumentID: instrumentID,
		Depth:        depth,
	}
	var resp *Orderbook
	return resp, d.SendWSRequest(nonMatchingEPL, getOrderbookByInstrumentID, input, &resp, false)
}

// WsRetrieveSupportedIndexNames retrieves the identifiers of all supported Price Indexes
// 'type' represents Type of a cryptocurrency price index. possible 'all', 'spot', 'derivative'
func (d *Deribit) WsRetrieveSupportedIndexNames(priceIndexType string) ([]string, error) {
	input := &struct {
		PriceIndexType string `json:"type,omitempty"`
	}{
		PriceIndexType: priceIndexType,
	}
	var resp []string
	return resp, d.SendWSRequest(nonMatchingEPL, "public/get_supported_index_names", input, &resp, false)
}

// WSRetrieveRequestForQuote retrieves RFQ information.
func (d *Deribit) WSRetrieveRequestForQuote(ccy currency.Code, kind string) ([]RequestForQuote, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	input := &struct {
		Currency currency.Code `json:"currency"`
		Kind     string        `json:"kind,omitempty"`
	}{
		Currency: ccy,
		Kind:     kind,
	}
	var resp []RequestForQuote
	return resp, d.SendWSRequest(nonMatchingEPL, getRFQ, input, &resp, false)
}

// WSRetrieveTradeVolumes retrieves trade volumes' data of all instruments through the websocket connection.
func (d *Deribit) WSRetrieveTradeVolumes(extended bool) ([]TradeVolumesData, error) {
	input := &struct {
		Extended bool `json:"extended,omitempty"`
	}{
		Extended: extended,
	}
	var resp []TradeVolumesData
	return resp, d.SendWSRequest(nonMatchingEPL, getTradeVolumes, input, &resp, false)
}

// WSRetrievesTradingViewChartData retrieves volatility index data for the requested instrument through the websocket connection.
func (d *Deribit) WSRetrievesTradingViewChartData(instrument, resolution string, startTime, endTime time.Time) (*TVChartData, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
		return nil, err
	}
	if resolution == "" {
		return nil, errors.New("unsupported resolution, resolution can not be empty")
	}
	input := &struct {
		Instrument     string `json:"instrument_name,omitempty"`
		StartTimestamp int64  `json:"start_timestamp,omitempty"`
		EndTimestamp   int64  `json:"end_timestamp,omitempty"`
		Resolution     string `json:"resolution,omitempty"`
	}{
		Instrument:     instrument,
		Resolution:     resolution,
		StartTimestamp: startTime.UnixMilli(),
		EndTimestamp:   endTime.UnixMilli(),
	}
	var resp *TVChartData
	return resp, d.SendWSRequest(nonMatchingEPL, getTradingViewChartData, input, &resp, false)
}

// WSRetrieveVolatilityIndexData retrieves volatility index data for the requested currency through the websocket connection.
func (d *Deribit) WSRetrieveVolatilityIndexData(ccy currency.Code, resolution string, startTime, endTime time.Time) ([]VolatilityIndexData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if resolution == "" {
		return nil, errResolutionNotSet
	}
	err := common.StartEndTimeCheck(startTime, endTime)
	if err != nil {
		return nil, err
	}
	input := &struct {
		Currency       string `json:"currency,omitempty"`
		StartTimestamp int64  `json:"start_timestamp,omitempty"`
		EndTimestamp   int64  `json:"end_timestamp,omitempty"`
		Resolution     string `json:"resolution,omitempty"`
	}{
		Currency:       ccy.String(),
		Resolution:     resolution,
		StartTimestamp: startTime.UnixMilli(),
		EndTimestamp:   endTime.UnixMilli(),
	}
	var resp VolatilityIndexRawData
	err = d.SendWSRequest(nonMatchingEPL, getVolatilityIndex, input, &resp, false)
	if err != nil {
		return nil, err
	}
	response := make([]VolatilityIndexData, len(resp.Data))
	for x := range resp.Data {
		response[x] = VolatilityIndexData{
			TimestampMS: time.UnixMilli(int64(resp.Data[x][0])),
			Open:        resp.Data[x][1],
			High:        resp.Data[x][2],
			Low:         resp.Data[x][3],
			Close:       resp.Data[x][4],
		}
	}
	return response, nil
}

// WSRetrievePublicTicker retrieves public ticker data of the instrument requested through the websocket connection.
func (d *Deribit) WSRetrievePublicTicker(instrument string) (*TickerData, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	input := &struct {
		Instrument string `json:"instrument_name,omitempty"`
	}{
		Instrument: instrument,
	}
	var resp *TickerData
	return resp, d.SendWSRequest(nonMatchingEPL, getTicker, input, &resp, false)
}

// WSRetrieveAccountSummary retrieves account summary data for the requested instrument through the websocket connection.
func (d *Deribit) WSRetrieveAccountSummary(ccy currency.Code, extended bool) (*AccountSummaryData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	input := &struct {
		Currency currency.Code `json:"currency"`
		Extended bool          `json:"extended"`
	}{
		Currency: ccy,
		Extended: extended,
	}
	var resp *AccountSummaryData
	return resp, d.SendWSRequest(nonMatchingEPL, getAccountSummary, input, &resp, true)
}

// WSCancelWithdrawal cancels withdrawal request for a given currency by its id through the websocket connection.
func (d *Deribit) WSCancelWithdrawal(ccy currency.Code, id int64) (*CancelWithdrawalData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if id <= 0 {
		return nil, fmt.Errorf("%w, withdrawal id has to be positive integer", errInvalidID)
	}
	input := &struct {
		Currency currency.Code `json:"currency"`
		ID       int64         `json:"id"`
	}{
		Currency: ccy,
		ID:       id,
	}
	var resp *CancelWithdrawalData
	return resp, d.SendWSRequest(nonMatchingEPL, cancelWithdrawal, input, &resp, true)
}

// WSCancelTransferByID cancels transfer by ID through the websocket connection.
func (d *Deribit) WSCancelTransferByID(ccy currency.Code, tfa string, id int64) (*AccountSummaryData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if id <= 0 {
		return nil, fmt.Errorf("%w, transfer id has to be positive integer", errInvalidID)
	}
	input := &struct {
		Currency                    string `json:"currency"`
		TwoFactorAuthenticationCode string `json:"tfa,omitempty"`
		ID                          int64  `json:"id"`
	}{
		Currency:                    ccy.String(),
		ID:                          id,
		TwoFactorAuthenticationCode: tfa,
	}
	var resp *AccountSummaryData
	return resp, d.SendWSRequest(nonMatchingEPL, cancelTransferByID, input, &resp, true)
}

// WSCreateDepositAddress creates a deposit address for the currency requested through the websocket connection.
func (d *Deribit) WSCreateDepositAddress(ccy currency.Code) (*DepositAddressData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	input := &struct {
		Currency currency.Code `json:"currency"`
	}{
		Currency: ccy,
	}
	var resp *DepositAddressData
	return resp, d.SendWSRequest(nonMatchingEPL, createDepositAddress, input, &resp, true)
}

// WSRetrieveDeposits retrieves the deposits of a given currency through the websocket connection.
func (d *Deribit) WSRetrieveDeposits(ccy currency.Code, count, offset int64) (*DepositsData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	input := &struct {
		Currency currency.Code `json:"currency"`
		Count    int64         `json:"count,omitempty"`
		Offset   int64         `json:"offset,omitempty"`
	}{
		Currency: ccy,
		Count:    count,
		Offset:   offset,
	}
	var resp *DepositsData
	return resp, d.SendWSRequest(nonMatchingEPL, getDeposits, input, &resp, true)
}

// WSRetrieveTransfers retrieves data for the requested currency through the websocket connection.
func (d *Deribit) WSRetrieveTransfers(ccy currency.Code, count, offset int64) (*TransfersData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	input := &struct {
		Currency string `json:"currency,omitempty"`
		Count    int64  `json:"count,omitempty"`
		Offset   int64  `json:"offset,omitempty"`
	}{
		Currency: ccy.String(),
		Count:    count,
		Offset:   offset,
	}
	var resp *TransfersData
	return resp, d.SendWSRequest(nonMatchingEPL, getTransfers, input, &resp, true)
}

// WSRetrieveCurrentDepositAddress retrieves the current deposit address for the requested currency through the websocket connection.
func (d *Deribit) WSRetrieveCurrentDepositAddress(ccy currency.Code) (*DepositAddressData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	input := &struct {
		Currency currency.Code `json:"currency"`
	}{
		Currency: ccy,
	}
	var resp *DepositAddressData
	err := d.SendWSRequest(nonMatchingEPL, getCurrentDepositAddress, input, &resp, true)
	if err != nil {
		return nil, err
	} else if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// WSRetrieveWithdrawals retrieves withdrawals data for a requested currency through the websocket connection.
func (d *Deribit) WSRetrieveWithdrawals(ccy currency.Code, count, offset int64) (*WithdrawalsData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	input := &struct {
		Currency currency.Code `json:"currency"`
		Count    int64         `json:"count,omitempty"`
		Offset   int64         `json:"offset,omitempty"`
	}{
		Currency: ccy,
		Count:    count,
		Offset:   offset,
	}
	var resp *WithdrawalsData
	return resp, d.SendWSRequest(nonMatchingEPL, getWithdrawals, input, &resp, true)
}

// WsSubmitTransferBetweenSubAccounts transfer funds between two (sub)accounts.
func (d *Deribit) WsSubmitTransferBetweenSubAccounts(ccy currency.Code, amount float64, destinationID int64, source string) (*TransferData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, fmt.Errorf("%w, amount : %f", errInvalidAmount, amount)
	}
	if destinationID <= 0 {
		return nil, errInvalidDestinationID
	}
	input := &struct {
		Currency    string  `json:"currency"`
		Amount      float64 `json:"amount"`
		Destination int64   `json:"destination"`
		Source      string  `json:"source,omitempty"`
	}{
		Currency:    ccy.String(),
		Amount:      amount,
		Destination: destinationID,
		Source:      source,
	}
	var resp *TransferData
	return resp, d.SendWSRequest(nonMatchingEPL, submitTransferBetweenSubAccounts, input, &resp, true)
}

// WSSubmitTransferToSubAccount submits a request to transfer a currency to a subaccount
func (d *Deribit) WSSubmitTransferToSubAccount(ccy currency.Code, amount float64, destinationID int64) (*TransferData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, errInvalidAmount
	}
	if destinationID <= 0 {
		return nil, errInvalidDestinationID
	}
	input := &struct {
		Currency    string  `json:"currency"`
		Destination int64   `json:"destination"`
		Amount      float64 `json:"amount"`
	}{
		Currency:    ccy.String(),
		Destination: destinationID,
		Amount:      amount,
	}
	var resp *TransferData
	return resp, d.SendWSRequest(nonMatchingEPL, submitTransferToSubaccount, input, &resp, true)
}

// WSSubmitTransferToUser submits a request to transfer a currency to another user through the websocket connection.
func (d *Deribit) WSSubmitTransferToUser(ccy currency.Code, tfa, destinationAddress string, amount float64) (*TransferData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, errInvalidAmount
	}
	if destinationAddress == "" {
		return nil, errInvalidCryptoAddress
	}
	input := &struct {
		Currency                    string  `json:"currency"`
		TwoFactorAuthenticationCode string  `json:"tfa,omitempty"`
		DestinationID               string  `json:"destination"`
		Amount                      float64 `json:"amount"`
	}{
		Currency:                    ccy.String(),
		TwoFactorAuthenticationCode: tfa,
		DestinationID:               destinationAddress,
		Amount:                      amount,
	}
	var resp *TransferData
	return resp, d.SendWSRequest(nonMatchingEPL, submitTransferToUser, input, &resp, true)
}

// ----------------------------------------------------------------------------

// WSSubmitWithdraw submits a withdrawal request to the exchange for the requested currency through the websocket connection.
func (d *Deribit) WSSubmitWithdraw(ccy currency.Code, address, priority string, amount float64) (*WithdrawData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, errInvalidAmount
	}
	if address == "" {
		return nil, errInvalidCryptoAddress
	}
	input := &struct {
		Currency string  `json:"currency"`
		Address  string  `json:"address"`
		Priority string  `json:"priority,omitempty"`
		Amount   float64 `json:"amount"`
	}{
		Currency: ccy.String(),
		Address:  address,
		Priority: priority,
		Amount:   amount,
	}
	var resp *WithdrawData
	return resp, d.SendWSRequest(nonMatchingEPL, submitWithdraw, input, &resp, true)
}

// WSRetrieveAnnouncements retrieves announcements through the websocket connection. Default "start_timestamp" parameter value is current timestamp, "count" parameter value must be between 1 and 50, default is 5.
func (d *Deribit) WSRetrieveAnnouncements(startTime time.Time, count int64) ([]Announcement, error) {
	input := &struct {
		StartTime int64 `json:"start_time,omitempty"`
		Count     int64 `json:"count,omitempty"`
	}{}
	if !startTime.IsZero() {
		input.StartTime = startTime.UnixMilli()
	}
	if count > 0 {
		input.Count = count
	}
	var resp []Announcement
	return resp, d.SendWSRequest(nonMatchingEPL, getAnnouncements, input, &resp, false)
}

// WSRetrievePublicPortfolioMargins public version of the method calculates portfolio margin info for simulated position. For concrete user position, the private version of the method must be used. The public version of the request has special restricted rate limit (not more than once per a second for the IP).
func (d *Deribit) WSRetrievePublicPortfolioMargins(ccy currency.Code, simulatedPositions map[string]float64) (*PortfolioMargin, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	input := &struct {
		Currency           currency.Code      `json:"currency"`
		SimulatedPositions map[string]float64 `json:"simulated_positions"`
	}{
		Currency: ccy,
	}
	if len(simulatedPositions) != 0 {
		input.SimulatedPositions = simulatedPositions
	}
	var resp *PortfolioMargin
	return resp, d.SendWSRequest(nonMatchingEPL, getPublicPortfolioMargins, input, &resp, false)
}

// WSChangeAPIKeyName changes the name of the api key requested through the websocket connection.
func (d *Deribit) WSChangeAPIKeyName(id int64, name string) (*APIKeyData, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w, invalid api key id", errInvalidID)
	}
	if !alphaNumericRegExp.MatchString(name) {
		return nil, errUnacceptableAPIKey
	}
	input := &struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}{
		ID:   id,
		Name: name,
	}
	var resp *APIKeyData
	return resp, d.SendWSRequest(nonMatchingEPL, changeAPIKeyName, input, &resp, true)
}

// WsChangeMarginModel change margin model
// Margin model: 'cross_pm', 'cross_sm', 'segregated_pm', 'segregated_sm'
// 'dry_run': If true request returns the result without switching the margining model. Default: false
func (d *Deribit) WsChangeMarginModel(userID int64, marginModel string, dryRun bool) ([]TogglePortfolioMarginResponse, error) {
	if marginModel == "" {
		return nil, errInvalidMarginModel
	}
	input := &struct {
		MarginModel string `json:"margin_model"`
		UserID      int64  `json:"user_id"`
		DryRun      bool   `json:"dry_run,omitempty"`
	}{
		MarginModel: marginModel,
		UserID:      userID,
		DryRun:      dryRun,
	}
	var resp []TogglePortfolioMarginResponse
	return resp, d.SendWSRequest(nonMatchingEPL, changeMarginModel, input, &resp, true)
}

// WSChangeScopeInAPIKey changes the name of the requested subaccount id through the websocket connection.
func (d *Deribit) WSChangeScopeInAPIKey(id int64, maxScope string) (*APIKeyData, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w, invalid api key id", errInvalidID)
	}
	input := &struct {
		ID       int64  `json:"id"`
		MaxScope string `json:"max_scope"`
	}{
		ID:       id,
		MaxScope: maxScope,
	}
	var resp *APIKeyData
	return resp, d.SendWSRequest(nonMatchingEPL, changeScopeInAPIKey, input, &resp, true)
}

// WSChangeSubAccountName retrieves changes the name of the requested subaccount id through the websocket connection.
func (d *Deribit) WSChangeSubAccountName(sid int64, name string) error {
	if sid <= 0 {
		return fmt.Errorf("%w, invalid subaccount user id", errInvalidID)
	}
	if name == "" {
		return errInvalidUsername
	}
	input := &struct {
		SID  int64  `json:"sid"`
		Name string `json:"name"`
	}{
		SID:  sid,
		Name: name,
	}
	var resp string
	err := d.SendWSRequest(nonMatchingEPL, changeSubAccountName, input, &resp, true)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return errSubAccountNameChangeFailed
	}
	return nil
}

// WSCreateAPIKey creates an api key based on the provided settings through the websocket connection.
func (d *Deribit) WSCreateAPIKey(maxScope, name string, defaultKey bool) (*APIKeyData, error) {
	input := &struct {
		MaxScope string `json:"max_scope"`
		Name     string `json:"name,omitempty"`
		Default  bool   `json:"default"`
	}{
		MaxScope: maxScope,
		Name:     name,
		Default:  defaultKey,
	}

	var resp *APIKeyData
	return resp, d.SendWSRequest(nonMatchingEPL, createAPIKey, input, &resp, true)
}

// WSCreateSubAccount creates a new subaccount through the websocket connection.
func (d *Deribit) WSCreateSubAccount() (*SubAccountData, error) {
	var resp *SubAccountData
	return resp, d.SendWSRequest(nonMatchingEPL, createSubAccount, nil, &resp, true)
}

// WSDisableAPIKey disables the api key linked to the provided id through the websocket connection.
func (d *Deribit) WSDisableAPIKey(id int64) (*APIKeyData, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w, invalid api key id", errInvalidID)
	}
	input := &struct {
		ID int64 `json:"id"`
	}{
		ID: id,
	}
	var resp *APIKeyData
	return resp, d.SendWSRequest(nonMatchingEPL, disableAPIKey, input, &resp, true)
}

// WsEditAPIKey edits existing API key. At least one parameter is required.
// Describes maximal access for tokens generated with given key, possible values:
// trade:[read, read_write, none],
// wallet:[read, read_write, none],
// account:[read, read_write, none],
// block_trade:[read, read_write, none].
func (d *Deribit) WsEditAPIKey(id int64, maxScope, name string, enabled bool, enabledFeatures, ipWhitelist []string) (*APIKeyData, error) {
	if id == 0 {
		return nil, errInvalidAPIKeyID
	}
	if maxScope == "" {
		return nil, errMaxScopeIsRequired
	}
	input := &struct {
		ID              int64    `json:"id"`
		MaxScope        string   `json:"max_scope"`
		Name            string   `json:"name,omitempty"`
		Enabled         bool     `json:"enabled,omitempty"`
		EnabledFeatures []string `json:"enabled_features,omitempty"`
		IPWhitelist     []string `json:"ip_whitelist,omitempty"`
	}{
		ID:              id,
		MaxScope:        maxScope,
		Name:            name,
		Enabled:         enabled,
		EnabledFeatures: enabledFeatures,
		IPWhitelist:     ipWhitelist,
	}
	var resp *APIKeyData
	return resp, d.SendWSRequest(nonMatchingEPL, editAPIKey, input, &resp, true)
}

// WSEnableAffiliateProgram enables the affiliate program through the websocket connection.
func (d *Deribit) WSEnableAffiliateProgram() error {
	var resp string
	err := d.SendWSRequest(nonMatchingEPL, enableAffiliateProgram, nil, &resp, true)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return errors.New("could not enable affiliate program")
	}
	return nil
}

// WSEnableAPIKey enables the api key linked to the provided id through the websocket connection.
func (d *Deribit) WSEnableAPIKey(id int64) (*APIKeyData, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w, invalid api key id", errInvalidID)
	}
	var resp *APIKeyData
	return resp, d.SendWSRequest(nonMatchingEPL, enableAPIKey, map[string]int64{"id": id}, &resp, true)
}

// WSRetrieveAccessLog lists access logs for the user through the websocket connection.
func (d *Deribit) WSRetrieveAccessLog(offset, count int64) (*AccessLog, error) {
	input := &struct {
		Offset int64 `json:"offset,omitempty"`
		Count  int64 `json:"count,omitempty"`
	}{
		Offset: offset,
		Count:  count,
	}
	var resp *AccessLog
	return resp, d.SendWSRequest(nonMatchingEPL, getAccessLog, input, &resp, true)
}

// WSRetrieveAffiliateProgramInfo retrieves the affiliate program info through the websocket connection.
func (d *Deribit) WSRetrieveAffiliateProgramInfo() (*AffiliateProgramInfo, error) {
	var resp *AffiliateProgramInfo
	return resp, d.SendWSRequest(nonMatchingEPL, getAffiliateProgramInfo, nil, &resp, true)
}

// WSRetrieveEmailLanguage retrieves the current language set for the email through the websocket connection.
func (d *Deribit) WSRetrieveEmailLanguage() (string, error) {
	var resp string
	return resp, d.SendWSRequest(nonMatchingEPL, getEmailLanguage, nil, &resp, true)
}

// WSRetrieveNewAnnouncements retrieves new announcements through the websocket connection.
func (d *Deribit) WSRetrieveNewAnnouncements() ([]Announcement, error) {
	var resp []Announcement
	return resp, d.SendWSRequest(nonMatchingEPL, getNewAnnouncements, nil, &resp, true)
}

// WSRetrievePrivatePortfolioMargins alculates portfolio margin info for simulated position or current position of the user through the websocket connection. This request has special restricted rate limit (not more than once per a second).
func (d *Deribit) WSRetrievePrivatePortfolioMargins(ccy currency.Code, accPositions bool, simulatedPositions map[string]float64) (*PortfolioMargin, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	input := &struct {
		Currency           currency.Code      `json:"currency"`
		AccountPositions   bool               `json:"acc_positions,omitempty"`
		SimulatedPositions map[string]float64 `json:"simulated_positions,omitempty"`
	}{
		Currency:         ccy,
		AccountPositions: accPositions,
	}
	if len(simulatedPositions) != 0 {
		input.SimulatedPositions = simulatedPositions
	}
	var resp *PortfolioMargin
	return resp, d.SendWSRequest(portfolioMarginEPL, getPrivatePortfolioMargins, input, &resp, true)
}

// WSRetrievePosition retrieves the data of all positions in the requested instrument name through the websocket connection.
func (d *Deribit) WSRetrievePosition(instrument string) (*PositionData, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	var resp *PositionData
	return resp, d.SendWSRequest(nonMatchingEPL, getPosition, map[string]string{"instrument_name": instrument}, &resp, true)
}

// WSRetrieveSubAccounts retrieves all subaccounts' data through the websocket connection.
func (d *Deribit) WSRetrieveSubAccounts(withPortfolio bool) ([]SubAccountData, error) {
	var resp []SubAccountData
	return resp, d.SendWSRequest(nonMatchingEPL, getSubAccounts, map[string]bool{"with_portfolio": withPortfolio}, &resp, true)
}

// WSRetrieveSubAccountDetails retrieves sub-account detail information through the websocket connection.
func (d *Deribit) WSRetrieveSubAccountDetails(ccy currency.Code, withOpenOrders bool) ([]SubAccountDetail, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	input := &struct {
		Currency       currency.Code `json:"currency"`
		WithOpenOrders bool          `json:"with_open_orders,omitempty"`
	}{
		Currency:       ccy,
		WithOpenOrders: withOpenOrders,
	}
	var resp []SubAccountDetail
	return resp, d.SendWSRequest(nonMatchingEPL, getSubAccountDetails, input, &resp, true)
}

// WSRetrievePositions retrieves positions data of the user account through the websocket connection.
func (d *Deribit) WSRetrievePositions(ccy currency.Code, kind string) ([]PositionData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	input := &struct {
		Currency currency.Code `json:"currency"`
		Kind     string        `json:"kind,omitempty"`
	}{
		Currency: ccy,
		Kind:     kind,
	}
	var resp []PositionData
	return resp, d.SendWSRequest(nonMatchingEPL, getPositions, input, &resp, true)
}

// WSRetrieveTransactionLog retrieves transaction logs data through the websocket connection.
func (d *Deribit) WSRetrieveTransactionLog(ccy currency.Code, query string, startTime, endTime time.Time, count, continuation int64) (*TransactionsData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
		return nil, err
	}
	input := &struct {
		Currency       currency.Code `json:"currency"`
		Query          string        `json:"query,omitempty"`
		StartTimestamp int64         `json:"start_timestamp,omitempty"`
		EndTimestamp   int64         `json:"end_timestamp,omitempty"`
		Count          int64         `json:"count,omitempty"`
		Continuation   int64         `json:"continuation,omitempty"`
	}{
		Currency:       ccy,
		Query:          query,
		StartTimestamp: startTime.UnixMilli(),
		EndTimestamp:   endTime.UnixMilli(),
		Count:          count,
		Continuation:   continuation,
	}
	var resp *TransactionsData
	return resp, d.SendWSRequest(nonMatchingEPL, getTransactionLog, input, &resp, true)
}

// WSRetrieveUserLocks retrieves information about locks on user account through the websocket connection.
func (d *Deribit) WSRetrieveUserLocks() ([]UserLock, error) {
	var resp []UserLock
	return resp, d.SendWSRequest(nonMatchingEPL, getUserLocks, nil, &resp, true)
}

// WSListAPIKeys retrieves all the api keys associated with a user account through the websocket connection.
func (d *Deribit) WSListAPIKeys(tfa string) ([]APIKeyData, error) {
	var resp []APIKeyData
	return resp, d.SendWSRequest(nonMatchingEPL, listAPIKeys, map[string]string{"tfa": tfa}, &resp, true)
}

// WsRetrieveCustodyAccounts retrieves user custody accounts
func (d *Deribit) WsRetrieveCustodyAccounts(ccy currency.Code) ([]CustodyAccount, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp []CustodyAccount
	return resp, d.SendWSRequest(nonMatchingEPL, listCustodyAccounts, &map[string]string{"currency": ccy.String()}, &resp, true)
}

// WSRemoveAPIKey removes api key vid ID through the websocket connection.
func (d *Deribit) WSRemoveAPIKey(id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w, invalid api key id", errInvalidID)
	}
	var resp string
	err := d.SendWSRequest(nonMatchingEPL, removeAPIKey, map[string]int64{"id": id}, &resp, true)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return errors.New("removal of the api key requested failed")
	}
	return nil
}

// WSRemoveSubAccount removes a subaccount given its id through the websocket connection.
func (d *Deribit) WSRemoveSubAccount(subAccountID int64) error {
	var resp string
	err := d.SendWSRequest(nonMatchingEPL, removeSubAccount, map[string]int64{"subaccount_id": subAccountID}, &resp, true)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return fmt.Errorf("removal of sub account %v failed", subAccountID)
	}
	return nil
}

// WSResetAPIKey sets an announcement as read through the websocket connection.
func (d *Deribit) WSResetAPIKey(id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w, invalid announcement id", errInvalidID)
	}
	var resp string
	err := d.SendWSRequest(nonMatchingEPL, resetAPIKey, map[string]int64{"announcement_id": id}, &resp, true)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return fmt.Errorf("setting announcement %v as read failed", id)
	}
	return nil
}

// WSSetEmailForSubAccount links an email given to the designated subaccount through the websocket connection.
func (d *Deribit) WSSetEmailForSubAccount(sid int64, email string) error {
	if sid <= 0 {
		return fmt.Errorf("%w, invalid subaccount user id", errInvalidID)
	}
	if !common.MatchesEmailPattern(email) {
		return errInvalidEmailAddress
	}
	input := &struct {
		SID   int64  `json:"sid"`
		Email string `json:"email"`
	}{
		Email: email,
		SID:   sid,
	}
	var resp string
	err := d.SendWSRequest(nonMatchingEPL, setEmailForSubAccount, input, &resp, true)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return fmt.Errorf("could not link email (%v) to subaccount %v", email, sid)
	}
	return nil
}

// WSSetEmailLanguage sets a requested language for an email through the websocket connection.
func (d *Deribit) WSSetEmailLanguage(language string) error {
	if language == "" {
		return errLanguageIsRequired
	}
	var resp string
	err := d.SendWSRequest(nonMatchingEPL, setEmailLanguage, map[string]string{"language": language}, &resp, true)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return fmt.Errorf("could not set the email language to %v", language)
	}
	return nil
}

// WsSetSelfTradingConfig configure self trading behavior through the websocket connection.
// mode: Self trading prevention behavior. Possible values: 'reject_taker', 'cancel_maker'
// extended_to_subaccounts: If value is true trading is prevented between subaccounts of given account
func (d *Deribit) WsSetSelfTradingConfig(mode string, extendedToSubaccounts bool) (string, error) {
	if mode == "" {
		return "", errTradeModeIsRequired
	}
	input := &struct {
		Mode                  string `json:"mode"`
		ExtendedToSubAccounts bool   `json:"extended_to_subaccounts"`
	}{
		Mode:                  mode,
		ExtendedToSubAccounts: extendedToSubaccounts,
	}
	var resp string
	return resp, d.SendWSRequest(nonMatchingEPL, setSelfTradingConfig, input, &resp, true)
}

// WSToggleNotificationsFromSubAccount toggles the notifications from a subaccount specified through the websocket connection.
func (d *Deribit) WSToggleNotificationsFromSubAccount(sid int64, state bool) error {
	if sid <= 0 {
		return fmt.Errorf("%w, invalid subaccount user id", errInvalidID)
	}
	input := &struct {
		SID   int64 `json:"sid"`
		State bool  `json:"state"`
	}{
		SID:   sid,
		State: state,
	}
	var resp string
	err := d.SendWSRequest(nonMatchingEPL, toggleNotificationsFromSubAccount, input, &resp, true)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return fmt.Errorf("toggling notifications for subaccount %v to %v failed", sid, state)
	}
	return nil
}

// WSTogglePortfolioMargining toggle between SM and PM models through the websocket connection.
func (d *Deribit) WSTogglePortfolioMargining(userID int64, enabled, dryRun bool) ([]TogglePortfolioMarginResponse, error) {
	if userID == 0 {
		return nil, errUserIDRequired
	}
	input := &struct {
		UserID  int64 `json:"user_id"`
		Enabled bool  `json:"enabled"`
		DryRun  bool  `json:"dry_run"`
	}{
		UserID:  userID,
		Enabled: enabled,
		DryRun:  dryRun,
	}
	var resp []TogglePortfolioMarginResponse
	return resp, d.SendWSRequest(nonMatchingEPL, togglePortfolioMargining, input, &resp, true)
}

// WSToggleSubAccountLogin toggles access for subaccount login through the websocket connection.
func (d *Deribit) WSToggleSubAccountLogin(sid int64, state bool) error {
	if sid <= 0 {
		return fmt.Errorf("%w, invalid subaccount user id", errInvalidID)
	}
	input := &struct {
		SID   int64 `json:"sid"`
		State bool  `json:"state"`
	}{
		SID:   sid,
		State: state,
	}
	var resp string
	err := d.SendWSRequest(nonMatchingEPL, toggleSubAccountLogin, input, &resp, true)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return fmt.Errorf("toggling login access for subaccount %v to %v failed", sid, state)
	}
	return nil
}

// WSSubmitBuy submits a private buy request through the websocket connection.
func (d *Deribit) WSSubmitBuy(arg *OrderBuyAndSellParams) (*PrivateTradeData, error) {
	if arg == nil || *arg == (OrderBuyAndSellParams{}) {
		return nil, fmt.Errorf("%w parameter is required", common.ErrNilPointer)
	}
	if arg.Instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	var resp *PrivateTradeData
	return resp, d.SendWSRequest(matchingEPL, submitBuy, &arg, &resp, true)
}

// WSSubmitSell submits a sell request with the parameters provided through the websocket connection.
func (d *Deribit) WSSubmitSell(arg *OrderBuyAndSellParams) (*PrivateTradeData, error) {
	if arg == nil || *arg == (OrderBuyAndSellParams{}) {
		return nil, fmt.Errorf("%w parameter is required", common.ErrNilPointer)
	}
	if arg.Instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	var resp *PrivateTradeData
	return resp, d.SendWSRequest(matchingEPL, submitSell, &arg, &resp, true)
}

// WSSubmitEdit submits an edit order request through the websocket connection.
func (d *Deribit) WSSubmitEdit(arg *OrderBuyAndSellParams) (*PrivateTradeData, error) {
	if arg == nil || *arg == (OrderBuyAndSellParams{}) {
		return nil, common.ErrNilPointer
	}
	if arg.OrderID == "" {
		return nil, fmt.Errorf("%w, order id is required", errInvalidID)
	}
	if arg.Amount <= 0 {
		return nil, errInvalidAmount
	}
	var resp *PrivateTradeData
	return resp, d.SendWSRequest(matchingEPL, submitEdit, &arg, &resp, true)
}

// WSEditOrderByLabel submits an edit order request sorted via label through the websocket connection.
func (d *Deribit) WSEditOrderByLabel(arg *OrderBuyAndSellParams) (*PrivateTradeData, error) {
	if arg == nil || *arg == (OrderBuyAndSellParams{}) {
		return nil, fmt.Errorf("%w argument cannot be null", common.ErrNilPointer)
	}
	if arg.Instrument == "" {
		return nil, errInvalidInstrumentName
	}
	if arg.Amount <= 0 {
		return nil, errInvalidAmount
	}
	var resp *PrivateTradeData
	return resp, d.SendWSRequest(nonMatchingEPL, editByLabel, &arg, &resp, true)
}

// WSSubmitCancel sends a request to cancel the order via its orderID through the websocket connection.
func (d *Deribit) WSSubmitCancel(orderID string) (*PrivateCancelData, error) {
	if orderID == "" {
		return nil, fmt.Errorf("%w, no order ID specified", errInvalidID)
	}
	var resp *PrivateCancelData
	return resp, d.SendWSRequest(matchingEPL, submitCancel, map[string]string{"order_id": orderID}, &resp, true)
}

// WSSubmitCancelAll sends a request to cancel all user orders in all currencies and instruments
func (d *Deribit) WSSubmitCancelAll(detailed bool) (*MultipleCancelResponse, error) {
	var resp *MultipleCancelResponse
	return resp, d.SendWSRequest(matchingEPL, submitCancelAll, map[string]bool{"detailed": detailed}, &resp, true)
}

// WSSubmitCancelAllByCurrency sends a request to cancel all user orders for the specified currency through the websocket connection.
func (d *Deribit) WSSubmitCancelAllByCurrency(ccy currency.Code, kind, orderType string, detailed bool) (*MultipleCancelResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	input := &struct {
		Currency  currency.Code `json:"currency"`
		Kind      string        `json:"kind,omitempty"`
		OrderType string        `json:"order_type,omitempty"`
		Detailed  bool          `json:"detailed"`
	}{
		Currency:  ccy,
		Kind:      kind,
		OrderType: orderType,
		Detailed:  detailed,
	}
	var resp *MultipleCancelResponse
	return resp, d.SendWSRequest(matchingEPL, submitCancelAllByCurrency, input, &resp, true)
}

// WSSubmitCancelAllByInstrument sends a request to cancel all user orders for the specified instrument through the websocket connection.
func (d *Deribit) WSSubmitCancelAllByInstrument(instrument, orderType string, detailed, includeCombos bool) (*MultipleCancelResponse, error) {
	if instrument == "" {
		return nil, errInvalidInstrumentName
	}
	input := &struct {
		Instrument    string `json:"instrument_name"`
		OrderType     string `json:"type,omitempty"`
		Detailed      bool   `json:"detailed,omitempty"`
		IncludeCombos bool   `json:"include_combos,omitempty"`
	}{
		Instrument:    instrument,
		OrderType:     orderType,
		Detailed:      detailed,
		IncludeCombos: includeCombos,
	}
	var resp *MultipleCancelResponse
	return resp, d.SendWSRequest(matchingEPL, submitCancelAllByInstrument, input, &resp, true)
}

// WsSubmitCancelAllByKind cancels all orders in currency(currencies), optionally filtered by instrument kind and/or order type.
// 'kind' Instrument kind. Possible values: 'future', 'option', 'spot', 'future_combo', 'option_combo', 'combo', 'any'
func (d *Deribit) WsSubmitCancelAllByKind(ccy currency.Code, kind, orderType string, detailed bool) (*MultipleCancelResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	input := &struct {
		Currency  string `json:"currency"`
		Kind      string `json:"kind,omitempty"`
		OrderType string `json:"type,omitempty"`
		Detailed  bool   `json:"detailed,omitempty"`
	}{
		Currency:  ccy.String(),
		Kind:      kind,
		OrderType: orderType,
		Detailed:  detailed,
	}
	var resp *MultipleCancelResponse
	return resp, d.SendWSRequest(matchingEPL, submitCancelAllByKind, input, &resp, true)
}

// WSSubmitCancelByLabel sends a request to cancel all user orders for the specified label through the websocket connection.
func (d *Deribit) WSSubmitCancelByLabel(label string, ccy currency.Code, detailed bool) (*MultipleCancelResponse, error) {
	input := &struct {
		Label    string `json:"label"`
		Currency string `json:"currency,omitempty"`
		Detailed bool   `json:"detailed,omitempty"`
	}{
		Label:    label,
		Currency: ccy.String(),
		Detailed: detailed,
	}
	var resp *MultipleCancelResponse
	return resp, d.SendWSRequest(matchingEPL, submitCancelByLabel, input, &resp, true)
}

// WSSubmitCancelQuotes cancels quotes based on the provided type.
//
// possible cancel_type values are delta, 'quote_set_id', 'instrument', 'instrument_kind', 'currency', and 'all'
// possible kind values are future 'option', 'spot', 'future_combo', 'option_combo', 'combo', and 'any'
func (d *Deribit) WSSubmitCancelQuotes(ccy currency.Code, minDelta, maxDelta float64, cancelType, quoteSetID, instrumentName, kind string, detailed bool) (*MultipleCancelResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if cancelType == "" {
		return nil, errors.New("cancel type is required")
	}
	input := &struct {
		CancelType     string  `json:"cancel_type"`
		Currency       string  `json:"currency"`
		Detailed       bool    `json:"detailed,omitempty"`
		MinDelta       float64 `json:"min_delta,omitempty"`
		MaxDelta       float64 `json:"max_delta,omitempty"`
		InstrumentName string  `json:"instrument_name,omitempty"`
		QuoteSetID     string  `json:"quote_set_id,omitempty"`
		Kind           string  `json:"kind,omitempty"`
	}{
		CancelType:     cancelType,
		Currency:       ccy.String(),
		Detailed:       detailed,
		MinDelta:       minDelta,
		MaxDelta:       maxDelta,
		InstrumentName: instrumentName,
		Kind:           kind,
		QuoteSetID:     quoteSetID,
	}
	var resp *MultipleCancelResponse
	return resp, d.SendWSRequest(matchingEPL, submitCancelQuotes, input, &resp, true)
}

// WSSubmitClosePosition sends a request to cancel all user orders for the specified label through the websocket connection.
func (d *Deribit) WSSubmitClosePosition(instrument, orderType string, price float64) (*PrivateTradeData, error) {
	if instrument == "" {
		return nil, errInvalidInstrumentName
	}
	input := &struct {
		Instrument string  `json:"instrument_name"`
		Type       string  `json:"type,omitempty"`
		Price      float64 `json:"price"`
	}{
		Instrument: instrument,
		Type:       orderType,
		Price:      price,
	}
	var resp *PrivateTradeData
	return resp, d.SendWSRequest(matchingEPL, submitClosePosition, input, &resp, true)
}

// WSRetrieveMargins sends a request to fetch account margins data through the websocket connection.
func (d *Deribit) WSRetrieveMargins(instrument string, amount, price float64) (*MarginsData, error) {
	if instrument == "" {
		return nil, errInvalidInstrumentName
	}
	if amount <= 0 {
		return nil, errInvalidAmount
	}
	if price <= 0 {
		return nil, errInvalidPrice
	}
	input := &struct {
		Instrument string  `json:"instrument_name"`
		Amount     float64 `json:"amount"`
		Price      float64 `json:"price"`
	}{
		Instrument: instrument,
		Amount:     amount,
		Price:      price,
	}
	var resp *MarginsData
	return resp, d.SendWSRequest(nonMatchingEPL, getMargins, input, &resp, true)
}

// WSRetrieveMMPConfig sends a request to fetch the config for MMP of the requested currency through the websocket connection.
func (d *Deribit) WSRetrieveMMPConfig(ccy currency.Code) (*MMPConfigData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *MMPConfigData
	return resp, d.SendWSRequest(nonMatchingEPL, getMMPConfig, map[string]currency.Code{"currency": ccy}, &resp, true)
}

// WSRetrieveOpenOrdersByCurrency retrieves open order by symbol and kind
func (d *Deribit) WSRetrieveOpenOrdersByCurrency(ccy currency.Code, kind, orderType string) ([]OrderData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	input := &struct {
		Currency  currency.Code `json:"currency"`
		Kind      string        `json:"kind,omitempty"`
		OrderType string        `json:"type,omitempty"`
	}{
		Currency:  ccy,
		Kind:      kind,
		OrderType: orderType,
	}
	var resp []OrderData
	return resp, d.SendWSRequest(nonMatchingEPL, getOpenOrdersByCurrency, input, &resp, true)
}

// WSRetrieveOpenOrdersByLabel retrieves open order by label and currency
func (d *Deribit) WSRetrieveOpenOrdersByLabel(ccy currency.Code, label string) ([]OrderData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	input := &struct {
		Currency currency.Code `json:"currency"`
		Label    string        `json:"label"`
	}{
		Currency: ccy,
		Label:    label,
	}
	var resp []OrderData
	return resp, d.SendWSRequest(nonMatchingEPL, getOpenOrdersByLabel, input, &resp, true)
}

// WSRetrieveOpenOrdersByInstrument sends a request to fetch open orders data sorted by requested params through the websocket connection.
func (d *Deribit) WSRetrieveOpenOrdersByInstrument(instrument, orderType string) ([]OrderData, error) {
	if instrument == "" {
		return nil, errInvalidInstrumentName
	}
	input := &struct {
		Instrument string `json:"instrument_name"`
		Type       string `json:"type,omitempty"`
	}{
		Instrument: instrument,
		Type:       orderType,
	}
	var resp []OrderData
	return resp, d.SendWSRequest(nonMatchingEPL, getOpenOrdersByInstrument, input, &resp, true)
}

// WSRetrieveOrderHistoryByCurrency sends a request to fetch order history according to given params and currency through the websocket connection.
func (d *Deribit) WSRetrieveOrderHistoryByCurrency(ccy currency.Code, kind string, count, offset int64, includeOld, includeUnfilled bool) ([]OrderData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	input := &struct {
		Currency        currency.Code `json:"currency"`
		Kind            string        `json:"kind,omitempty"`
		Count           int64         `json:"count,omitempty"`
		Offset          int64         `json:"offset,omitempty"`
		IncludeOld      bool          `json:"include_old,omitempty"`
		IncludeUnfilled bool          `json:"include_unfilled,omitempty"`
	}{
		Currency:        ccy,
		Kind:            kind,
		Count:           count,
		Offset:          offset,
		IncludeOld:      includeOld,
		IncludeUnfilled: includeUnfilled,
	}
	var resp []OrderData
	return resp, d.SendWSRequest(nonMatchingEPL, getOrderHistoryByCurrency, input, &resp, true)
}

// WSRetrieveOrderHistoryByInstrument sends a request to fetch order history according to given params and instrument through the websocket connection.
func (d *Deribit) WSRetrieveOrderHistoryByInstrument(instrument string, count, offset int64, includeOld, includeUnfilled bool) ([]OrderData, error) {
	if instrument == "" {
		return nil, errInvalidInstrumentName
	}
	input := &struct {
		Instrument      string `json:"instrument_name"`
		Count           int64  `json:"count,omitempty"`
		Offset          int64  `json:"offset,omitempty"`
		IncludeOld      bool   `json:"include_old"`
		IncludeUnfilled bool   `json:"include_unfilled"`
	}{
		Instrument:      instrument,
		Count:           count,
		Offset:          offset,
		IncludeOld:      includeOld,
		IncludeUnfilled: includeUnfilled,
	}
	var resp []OrderData
	return resp, d.SendWSRequest(nonMatchingEPL, getOrderHistoryByInstrument, input, &resp, true)
}

// WSRetrieveOrderMarginsByID sends a request to fetch order margins data according to their ids through the websocket connection.
func (d *Deribit) WSRetrieveOrderMarginsByID(ids []string) ([]OrderData, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("%w, order ids cannot be empty", errInvalidID)
	}
	var resp []OrderData
	return resp, d.SendWSRequest(nonMatchingEPL, getOrderMarginByIDs, map[string][]string{"ids": ids}, &resp, true)
}

// WSRetrievesOrderState sends a request to fetch order state of the order id provided
func (d *Deribit) WSRetrievesOrderState(orderID string) (*OrderData, error) {
	if orderID == "" {
		return nil, fmt.Errorf("%w, no order ID specified", errInvalidID)
	}
	var resp *OrderData
	return resp, d.SendWSRequest(nonMatchingEPL, getOrderState, map[string]string{"order_id": orderID}, &resp, true)
}

// WsRetrieveOrderStateByLabel retrieves an order state by label and currency
func (d *Deribit) WsRetrieveOrderStateByLabel(ccy currency.Code, label string) ([]OrderData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	input := &struct {
		Currency currency.Code `json:"currency"`
		Label    string        `json:"label"`
	}{
		Currency: ccy,
		Label:    label,
	}
	var resp []OrderData
	return resp, d.SendWSRequest(nonMatchingEPL, getOrderStateByLabel, input, &resp, true)
}

// WSRetrieveTriggerOrderHistory sends a request to fetch order state of the order id provided through the websocket connection.
func (d *Deribit) WSRetrieveTriggerOrderHistory(ccy currency.Code, instrumentName, continuation string, count int64) (*OrderData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	input := &struct {
		Currency     string `json:"currency,omitempty"`
		Instrument   string `json:"instrument,omitempty"`
		Continuation string `json:"continuation,omitempty"`
		Count        int64  `json:"count,omitempty"`
	}{
		Currency:     ccy.String(),
		Instrument:   instrumentName,
		Continuation: continuation,
		Count:        count,
	}
	var resp *OrderData
	return resp, d.SendWSRequest(nonMatchingEPL, getTriggerOrderHistory, input, &resp, true)
}

// WSRetrieveUserTradesByCurrency sends a request to fetch user trades sorted by currency through the websocket connection.
func (d *Deribit) WSRetrieveUserTradesByCurrency(ccy currency.Code, kind, startID, endID, sorting string, count int64, includeOld bool) (*UserTradesData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	input := &struct {
		Currency   currency.Code `json:"currency"`
		Kind       string        `json:"kind"`
		StartID    string        `json:"start_id,omitempty"`
		EndID      string        `json:"end_id,omitempty"`
		Sorting    string        `json:"sorting,omitempty"`
		Count      int64         `json:"count,omitempty"`
		IncludeOld bool          `json:"include_old,omitempty"`
	}{
		Currency:   ccy,
		Kind:       kind,
		StartID:    startID,
		EndID:      endID,
		Sorting:    sorting,
		Count:      count,
		IncludeOld: includeOld,
	}
	var resp *UserTradesData
	return resp, d.SendWSRequest(nonMatchingEPL, getUserTradesByCurrency, input, &resp, true)
}

// WSRetrieveUserTradesByCurrencyAndTime retrieves user trades sorted by currency and time through the websocket connection.
func (d *Deribit) WSRetrieveUserTradesByCurrencyAndTime(ccy currency.Code, kind, sorting string, count int64, startTime, endTime time.Time) (*UserTradesData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	input := &struct {
		Currency  currency.Code `json:"currency"`
		Kind      string        `json:"kind,omitempty"`
		StartTime int64         `json:"start_timestamp,omitempty"`
		EndTime   int64         `json:"end_timestamp,omitempty"`
		Sorting   string        `json:"sorting,omitempty"`
		Count     int64         `json:"count,omitempty"`
	}{
		Currency: ccy,
		Kind:     kind,
		Sorting:  sorting,
		Count:    count,
	}
	if !startTime.IsZero() {
		input.StartTime = startTime.UnixMilli()
	}
	if !endTime.IsZero() {
		input.EndTime = endTime.UnixMilli()
	}
	var resp *UserTradesData
	return resp, d.SendWSRequest(nonMatchingEPL, getUserTradesByCurrencyAndTime, input, &resp, true)
}

// WsRetrieveUserTradesByInstrument retrieves user trades sorted by instrument through the websocket connection.
func (d *Deribit) WsRetrieveUserTradesByInstrument(instrument, sorting string, startSeq, endSeq, count int64, includeOld bool) (*UserTradesData, error) {
	if instrument == "" {
		return nil, errInvalidInstrumentName
	}
	input := &struct {
		Instrument string `json:"instrument_name"`
		StartSeq   int64  `json:"start_seq,omitempty"`
		EndSeq     int64  `json:"end_seq,omitempty"`
		Sorting    string `json:"sorting,omitempty"`
		Count      int64  `json:"count,omitempty"`
		IncludeOld bool   `json:"include_old,omitempty"`
	}{
		Instrument: instrument,
		StartSeq:   startSeq,
		EndSeq:     endSeq,
		Sorting:    sorting,
		Count:      count,
		IncludeOld: includeOld,
	}
	var resp *UserTradesData
	return resp, d.SendWSRequest(nonMatchingEPL, getUserTradesByInstrument, input, &resp, true)
}

// WSRetrieveUserTradesByInstrumentAndTime retrieves user trades sorted by instrument and time through the websocket connection.
func (d *Deribit) WSRetrieveUserTradesByInstrumentAndTime(instrument, sorting string, count int64, includeOld bool, startTime, endTime time.Time) (*UserTradesData, error) {
	if instrument == "" {
		return nil, errInvalidInstrumentName
	}
	if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
		return nil, err
	}
	input := &struct {
		Instrument string `json:"instrument_name"`
		StartTime  int64  `json:"start_timestamp,omitempty"`
		EndTime    int64  `json:"end_timestamp,omitempty"`
		Sorting    string `json:"sorting,omitempty"`
		Count      int64  `json:"count,omitempty"`
		IncludeOld bool   `json:"include_old,omitempty"`
	}{
		Instrument: instrument,
		StartTime:  startTime.UnixMilli(),
		EndTime:    endTime.UnixMilli(),
		Sorting:    sorting,
		Count:      count,
		IncludeOld: includeOld,
	}
	var resp *UserTradesData
	return resp, d.SendWSRequest(nonMatchingEPL, getUserTradesByInstrumentAndTime, input, &resp, true)
}

// WSRetrieveUserTradesByOrder retrieves user trades fetched by orderID through the web socket connection.
func (d *Deribit) WSRetrieveUserTradesByOrder(orderID, sorting string) (*UserTradesData, error) {
	if orderID == "" {
		return nil, fmt.Errorf("%w, no order ID specified", errInvalidID)
	}
	input := &struct {
		OrderID string `json:"order_id"`
		Sorting string `json:"sorting,omitempty"`
	}{
		OrderID: orderID,
		Sorting: sorting,
	}
	var resp *UserTradesData
	return resp, d.SendWSRequest(nonMatchingEPL, getUserTradesByOrder, input, &resp, true)
}

// WSResetMMP sends a request to reset MMP for a currency provided through the websocket connection.
func (d *Deribit) WSResetMMP(ccy currency.Code) error {
	if ccy.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	var resp string
	err := d.SendWSRequest(nonMatchingEPL, resetMMP, map[string]currency.Code{"currency": ccy}, &resp, true)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return fmt.Errorf("mmp could not be reset for %v", ccy.String())
	}
	return nil
}

// WSSendRequestForQuote sends RFQ on a given instrument through the websocket connection.
func (d *Deribit) WSSendRequestForQuote(instrumentName string, amount float64, side order.Side) error {
	if instrumentName == "" {
		return errInvalidInstrumentName
	}
	input := &struct {
		Instrument string  `json:"instrument_name"`
		Amount     float64 `json:"amount,omitempty"`
		Side       string  `json:"side,omitempty"`
	}{
		Instrument: instrumentName,
		Amount:     amount,
		Side:       side.String(),
	}
	var resp string
	err := d.SendWSRequest(nonMatchingEPL, sendRFQ, input, &resp, true)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return fmt.Errorf("rfq couldn't send for %v", instrumentName)
	}
	return nil
}

// WSSetMMPConfig sends a request to set the given parameter values to the mmp config for the provided currency through the websocket connection.
func (d *Deribit) WSSetMMPConfig(ccy currency.Code, interval kline.Interval, frozenTime int64, quantityLimit, deltaLimit float64) error {
	if ccy.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	params := make(map[string]any)
	params["currency"] = ccy
	intervalString, err := d.GetResolutionFromInterval(interval)
	if err != nil {
		return err
	}
	params["interval"] = intervalString
	params["frozen_time"] = frozenTime
	if quantityLimit != 0 {
		params["quantity_time"] = quantityLimit
	}
	if deltaLimit != 0 {
		params["delta_limit"] = deltaLimit
	}
	var resp string
	err = d.SendWSRequest(nonMatchingEPL, setMMPConfig, params, &resp, true)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return fmt.Errorf("mmp data could not be set for %v", ccy.String())
	}
	return nil
}

// WSRetrieveSettlementHistoryByInstrument sends a request to fetch settlement history data sorted by instrument through the websocket connection.
func (d *Deribit) WSRetrieveSettlementHistoryByInstrument(instrument, settlementType, continuation string, count int64, searchStartTimeStamp time.Time) (*PrivateSettlementsHistoryData, error) {
	if instrument == "" {
		return nil, errInvalidInstrumentName
	}
	input := &struct {
		Instrument           string `json:"instrument_name"`
		Continuation         string `json:"continuation,omitempty"`
		Count                int64  `json:"count,omitempty"`
		SearchStartTimestamp int64  `json:"search_start_timestamp,omitempty"`
		Type                 string `json:"type,omitempty"`
	}{
		Instrument:   instrument,
		Continuation: continuation,
		Count:        count,
		Type:         settlementType,
	}
	if !searchStartTimeStamp.IsZero() {
		input.SearchStartTimestamp = searchStartTimeStamp.UnixMilli()
	}
	var resp *PrivateSettlementsHistoryData
	return resp, d.SendWSRequest(nonMatchingEPL, getSettlementHistoryByInstrument, input, &resp, true)
}

// WSRetrieveSettlementHistoryByCurency sends a request to fetch settlement history data sorted by currency through the websocket connection.
func (d *Deribit) WSRetrieveSettlementHistoryByCurency(ccy currency.Code, settlementType, continuation string, count int64, searchStartTimeStamp time.Time) (*PrivateSettlementsHistoryData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	input := &struct {
		Currency             currency.Code `json:"currency"`
		SettlementType       string        `json:"settlement_type,omitempty"`
		Continuation         string        `json:"continuation,omitempty"`
		Count                int64         `json:"count,omitempty"`
		SearchStartTimestamp int64         `json:"search_start_timestamp,omitempty"`
	}{
		Currency:       ccy,
		SettlementType: settlementType,
		Continuation:   continuation,
		Count:          count,
	}
	if !searchStartTimeStamp.IsZero() {
		input.SearchStartTimestamp = searchStartTimeStamp.UnixMilli()
	}
	var resp *PrivateSettlementsHistoryData
	return resp, d.SendWSRequest(nonMatchingEPL, getSettlementHistoryByCurrency, input, &resp, true)
}

// WSRetrieveComboIDs Retrieves available combos.
// This method can be used to get the list of all combos, or only the list of combos in the given state.
func (d *Deribit) WSRetrieveComboIDs(ccy currency.Code, state string) ([]string, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	input := &struct {
		Currency currency.Code `json:"currency"`
		State    string        `json:"state,omitempty"`
	}{
		Currency: ccy,
		State:    state,
	}
	var resp []string
	return resp, d.SendWSRequest(nonMatchingEPL, getComboIDs, input, &resp, false)
}

// WSRetrieveComboDetails retrieves information about a combo through the websocket connection.
func (d *Deribit) WSRetrieveComboDetails(comboID string) (*ComboDetail, error) {
	if comboID == "" {
		return nil, errInvalidComboID
	}
	var resp *ComboDetail
	return resp, d.SendWSRequest(nonMatchingEPL, getComboDetails, map[string]string{"combo_id": comboID}, &resp, false)
}

// WSRetrieveCombos retrieves information about active combos through the websocket connection.
func (d *Deribit) WSRetrieveCombos(ccy currency.Code) ([]ComboDetail, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp []ComboDetail
	return resp, d.SendWSRequest(nonMatchingEPL, getCombos, map[string]currency.Code{"currency": ccy}, &resp, false)
}

// WSCreateCombo verifies and creates a combo book or returns an existing combo matching given trades through the websocket connection.
func (d *Deribit) WSCreateCombo(args []ComboParam) (*ComboDetail, error) {
	if len(args) == 0 {
		return nil, errNoArgumentPassed
	}
	for x := range args {
		if args[x].InstrumentName == "" {
			return nil, fmt.Errorf("%w, empty string", errInvalidInstrumentName)
		}
		args[x].Direction = strings.ToLower(args[x].Direction)
		if args[x].Direction != sideBUY && args[x].Direction != sideSELL {
			return nil, errInvalidOrderSideOrDirection
		}
		if args[x].Amount <= 0 {
			return nil, errInvalidAmount
		}
	}
	var resp *ComboDetail
	return resp, d.SendWSRequest(nonMatchingEPL, createCombos, map[string]any{"trades": args}, &resp, true)
}

// WsLogout gracefully close websocket connection, when COD (Cancel On Disconnect) is enabled orders are not cancelled
func (d *Deribit) WsLogout(invalidateToken bool) error {
	input := struct {
		InvalidateToken bool `json:"invalidate_token,omitempty"`
	}{
		InvalidateToken: invalidateToken,
	}
	return d.SendWSRequest(nonMatchingEPL, "private/logout", input, &struct{}{}, true)
}

// WsEnableCancelOnDisconnect enable Cancel On Disconnect for the connection.
// After enabling Cancel On Disconnect all orders created by the connection will be removed when the connection is closed.
func (d *Deribit) WsEnableCancelOnDisconnect(scope string) (string, error) {
	input := &struct {
		Scope string `json:"scope,omitempty"`
	}{
		Scope: scope,
	}
	var resp string
	return resp, d.SendWSRequest(nonMatchingEPL, "private/enable_cancel_on_disconnect", input, &resp, true)
}

// WsDisableCancelOnDisconnect isable Cancel On Disconnect for the connection.
// When change is applied for the account, then every newly opened connection will start with inactive Cancel on Disconnect.
// scope: possible values are 'connection', 'account'
func (d *Deribit) WsDisableCancelOnDisconnect(scope string) (string, error) {
	input := &struct {
		Scope string `json:"scope,omitempty"`
	}{
		Scope: scope,
	}
	var resp string
	return resp, d.SendWSRequest(nonMatchingEPL, "private/disable_cancel_on_disconnect", input, &resp, true)
}

// SayHello method used to introduce the client software connected to Deribit platform over websocket.
// It returns version information
func (d *Deribit) SayHello(clientName, clientVersion string) (*Info, error) {
	if clientName == "" {
		return nil, errors.New("client name is required")
	}
	input := &struct {
		ClientName    string `json:"client_name"`
		ClientVersion string `json:"client_version"`
	}{
		ClientName:    clientName,
		ClientVersion: clientVersion,
	}
	var resp *Info
	return resp, d.SendWSRequest(nonMatchingEPL, "public/hello", input, &resp, false)
}

// WsRetrieveCancelOnDisconnect read current Cancel On Disconnect configuration for the account.
// 'scope': Specifies if Cancel On Disconnect change should be applied/checked for the current connection or the account (default - connection)
// Scope connection can be used only when working via Websocket.
func (d *Deribit) WsRetrieveCancelOnDisconnect(scope string) (*CancelOnDisconnect, error) {
	input := &struct {
		Scope string `json:"scope,omitempty"`
	}{
		Scope: scope,
	}
	var resp *CancelOnDisconnect
	return resp, d.SendWSRequest(nonMatchingEPL, "private/get_cancel_on_disconnect", input, &resp, true)
}

// WsExchangeToken generates a token for a new subject id. This method can be used to switch between subaccounts.
func (d *Deribit) WsExchangeToken(refreshToken string, subjectID int64) (*RefreshTokenInfo, error) {
	if refreshToken == "" {
		return nil, errRefreshTokenRequired
	}
	if subjectID == 0 {
		return nil, errors.New("subject id is required")
	}
	input := &struct {
		RefreshToken string `json:"retresh_token"`
		SubjectID    int64  `json:"subject_id"`
	}{
		RefreshToken: refreshToken,
		SubjectID:    subjectID,
	}
	var resp *RefreshTokenInfo
	return resp, d.SendWSRequest(nonMatchingEPL, "public/exchange_token", input, &resp, true)
}

// WsForkToken generates a token for a new named session. This method can be used only with session scoped tokens.
func (d *Deribit) WsForkToken(refreshToken, sessionName string) (*RefreshTokenInfo, error) {
	if refreshToken == "" {
		return nil, errRefreshTokenRequired
	}
	if sessionName == "" {
		return nil, errSessionNameRequired
	}
	input := &struct {
		RefreshToken string `json:"refresh_token"`
		SessionName  string `json:"session_name"`
	}{
		RefreshToken: refreshToken,
		SessionName:  sessionName,
	}
	var resp *RefreshTokenInfo
	return resp, d.SendWSRequest(nonMatchingEPL, "public/fork_token", input, &resp, true)
}

// UnsubscribeAll unsubscribe from all the public channels subscribed so far.
func (d *Deribit) UnsubscribeAll() (string, error) {
	var resp string
	return resp, d.SendWSRequest(nonMatchingEPL, "public/unsubscribe_all", nil, &resp, false)
}

// UnsubscribeAllPrivateChannels sends an unsubscribe request to cancel all private channels subscriptions
func (d *Deribit) UnsubscribeAllPrivateChannels() (string, error) {
	var resp string
	return resp, d.SendWSRequest(nonMatchingEPL, "private/unsubscribe_all", nil, &resp, false)
}

// ------------------------------------------------------------------------------------------------

// WSExecuteBlockTrade executes a block trade request
// The whole request have to be exact the same as in private/verify_block_trade, only role field should be set appropriately - it basically means that both sides have to agree on the same timestamp, nonce, trades fields and server will assure that role field is different between sides (each party accepted own role).
// Using the same timestamp and nonce by both sides in private/verify_block_trade assures that even if unintentionally both sides execute given block trade with valid counterparty_signature, the given block trade will be executed only once
func (d *Deribit) WSExecuteBlockTrade(timestampMS time.Time, nonce, role string, ccy currency.Code, trades []BlockTradeParam) ([]BlockTradeResponse, error) {
	if nonce == "" {
		return nil, errMissingNonce
	}
	if role != roleMaker && role != roleTaker {
		return nil, errInvalidTradeRole
	}
	if len(trades) == 0 {
		return nil, errNoArgumentPassed
	}
	for x := range trades {
		if trades[x].InstrumentName == "" {
			return nil, fmt.Errorf("%w, empty string", errInvalidInstrumentName)
		}
		trades[x].Direction = strings.ToLower(trades[x].Direction)
		if trades[x].Direction != sideBUY && trades[x].Direction != sideSELL {
			return nil, errInvalidOrderSideOrDirection
		}
		if trades[x].Amount <= 0 {
			return nil, errInvalidAmount
		}
		if trades[x].Price < 0 {
			return nil, fmt.Errorf("%w, trade price can't be negative", errInvalidPrice)
		}
	}
	signature, err := d.WSVerifyBlockTrade(timestampMS, nonce, role, ccy, trades)
	if err != nil {
		return nil, err
	}
	input := &struct {
		Nonce                 string            `json:"nonce"`
		Role                  string            `json:"role,omitempty"`
		CounterpartySignature string            `json:"counterparty_signature"`
		Trades                []BlockTradeParam `json:"trades"`
		Timestamp             int64             `json:"timestamp"`
		Currency              string            `json:"currency,omitempty"`
	}{
		Nonce:                 nonce,
		Role:                  role,
		CounterpartySignature: signature,
		Trades:                trades,
		Timestamp:             timestampMS.UnixMilli(),
		Currency:              ccy.String(),
	}
	var resp []BlockTradeResponse
	return resp, d.SendWSRequest(matchingEPL, executeBlockTrades, input, &resp, true)
}

// WSVerifyBlockTrade verifies and creates block trade signature through the websocket connection.
func (d *Deribit) WSVerifyBlockTrade(timestampMS time.Time, nonce, role string, ccy currency.Code, trades []BlockTradeParam) (string, error) {
	if nonce == "" {
		return "", errMissingNonce
	}
	if role != roleMaker && role != roleTaker {
		return "", errInvalidTradeRole
	}
	if len(trades) == 0 {
		return "", errNoArgumentPassed
	}
	for x := range trades {
		if trades[x].InstrumentName == "" {
			return "", fmt.Errorf("%w, empty string", errInvalidInstrumentName)
		}
		trades[x].Direction = strings.ToLower(trades[x].Direction)
		if trades[x].Direction != sideBUY && trades[x].Direction != sideSELL {
			return "", errInvalidOrderSideOrDirection
		}
		if trades[x].Amount <= 0 {
			return "", errInvalidAmount
		}
		if trades[x].Price < 0 {
			return "", fmt.Errorf("%w, trade price can't be negative", errInvalidPrice)
		}
	}
	if timestampMS.IsZero() {
		return "", errZeroTimestamp
	}
	input := &struct {
		Nonce                 string            `json:"nonce"`
		Role                  string            `json:"role,omitempty"`
		CounterpartySignature string            `json:"counterparty_signature"`
		Trades                []BlockTradeParam `json:"trades"`
		Timestamp             int64             `json:"timestamp"`
		Currency              string            `json:"currency,omitempty"`
	}{
		Nonce:     nonce,
		Role:      role,
		Trades:    trades,
		Timestamp: timestampMS.UnixMilli(),
		Currency:  ccy.String(),
	}
	resp := &struct {
		Signature string `json:"signature"`
	}{}
	return resp.Signature, d.SendWSRequest(matchingEPL, verifyBlockTrades, input, &resp, true)
}

// WsInvalidateBlockTradeSignature user at any time (before the private/execute_block_trade is called) can invalidate its own signature effectively cancelling block trade through the websocket connection.
func (d *Deribit) WsInvalidateBlockTradeSignature(signature string) error {
	if signature == "" {
		return errMissingSignature
	}
	params := url.Values{}
	params.Set("signature", signature)
	var resp string
	err := d.SendWSRequest(nonMatchingEPL, invalidateBlockTradesSignature, params, &resp, true)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return fmt.Errorf("server response: %s", resp)
	}
	return nil
}

// WSRetrieveUserBlockTrade returns information about users block trade through the websocket connection.
func (d *Deribit) WSRetrieveUserBlockTrade(id string) ([]BlockTradeData, error) {
	if id == "" {
		return nil, errMissingBlockTradeID
	}
	var resp []BlockTradeData
	return resp, d.SendWSRequest(nonMatchingEPL, getBlockTrades, map[string]string{"id": id}, &resp, true)
}

// WSRetrieveLastBlockTradesByCurrency returns list of last users block trades through the websocket connection.
func (d *Deribit) WSRetrieveLastBlockTradesByCurrency(ccy currency.Code, startID, endID string, count int64) ([]BlockTradeData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	input := &struct {
		Currency currency.Code `json:"currency"`
		StartID  string        `json:"start_id,omitempty"`
		EndID    string        `json:"end_id,omitempty"`
		Count    int64         `json:"count,omitempty"`
	}{
		Currency: ccy,
		StartID:  startID,
		EndID:    endID,
		Count:    count,
	}
	var resp []BlockTradeData
	return resp, d.SendWSRequest(nonMatchingEPL, getLastBlockTradesByCurrency, input, &resp, true)
}

// WSMovePositions moves positions from source subaccount to target subaccount through the websocket connection.
func (d *Deribit) WSMovePositions(ccy currency.Code, sourceSubAccountUID, targetSubAccountUID int64, trades []BlockTradeParam) ([]BlockTradeMoveResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if sourceSubAccountUID == 0 {
		return nil, fmt.Errorf("%w source sub-account id", errMissingSubAccountID)
	}
	if targetSubAccountUID == 0 {
		return nil, fmt.Errorf("%w target sub-account id", errMissingSubAccountID)
	}
	for x := range trades {
		if trades[x].InstrumentName == "" {
			return nil, fmt.Errorf("%w, empty string", errInvalidInstrumentName)
		}
		if trades[x].Amount <= 0 {
			return nil, errInvalidAmount
		}
		if trades[x].Price < 0 {
			return nil, fmt.Errorf("%w, trade price can't be negative", errInvalidPrice)
		}
	}
	input := &struct {
		Currency  currency.Code     `json:"currency"`
		Trades    []BlockTradeParam `json:"trades"`
		TargetUID int64             `json:"target_uid"`
		SourceUID int64             `json:"source_uid"`
	}{
		Currency:  ccy,
		Trades:    trades,
		TargetUID: targetSubAccountUID,
		SourceUID: sourceSubAccountUID,
	}
	var resp []BlockTradeMoveResponse
	return resp, d.SendWSRequest(nonMatchingEPL, movePositions, input, &resp, true)
}

// WsSimulateBlockTrade checks if a block trade can be executed through the websocket
func (d *Deribit) WsSimulateBlockTrade(role string, trades []BlockTradeParam) (bool, error) {
	if role != roleMaker && role != roleTaker {
		return false, errInvalidTradeRole
	}
	if len(trades) == 0 {
		return false, errNoArgumentPassed
	}
	for x := range trades {
		if trades[x].InstrumentName == "" {
			return false, fmt.Errorf("%w, empty string", errInvalidInstrumentName)
		}
		trades[x].Direction = strings.ToLower(trades[x].Direction)
		if trades[x].Direction != sideBUY && trades[x].Direction != sideSELL {
			return false, errInvalidOrderSideOrDirection
		}
		if trades[x].Amount <= 0 {
			return false, errInvalidAmount
		}
		if trades[x].Price < 0 {
			return false, fmt.Errorf("%w, trade price can't be negative", errInvalidPrice)
		}
	}
	input := &struct {
		Role   string            `json:"role"`
		Trades []BlockTradeParam `json:"trades"`
	}{
		Role:   role,
		Trades: trades,
	}
	var resp bool
	return resp, d.SendWSRequest(matchingEPL, simulateBlockPosition, input, resp, true)
}

// SendWSRequest sends a request through the websocket connection.
// both authenticated and public endpoints are allowed.
func (d *Deribit) SendWSRequest(epl request.EndpointLimit, method string, params, response any, authenticated bool) error {
	if authenticated && !d.Websocket.CanUseAuthenticatedEndpoints() {
		return errWebsocketConnectionNotAuthenticated
	}
	input := &WsRequest{
		JSONRPCVersion: rpcVersion,
		ID:             d.Websocket.Conn.GenerateMessageID(true),
		Method:         method,
		Params:         params,
	}
	resp := &wsResponse{Result: response}
	err := d.sendWsPayload(epl, input, resp)
	if err != nil {
		return err
	}
	if resp.Error.Code != 0 || resp.Error.Message != "" {
		var data string
		if resp.Error.Data != nil {
			value, err := json.Marshal(resp.Error.Data)
			if err == nil {
				data = string(value)
			}
		}
		return fmt.Errorf("code: %d message: %s %s", resp.Error.Code, resp.Error.Message, data)
	}
	return nil
}

// sendWsPayload handles sending Websocket requests
func (d *Deribit) sendWsPayload(ep request.EndpointLimit, input *WsRequest, response *wsResponse) error {
	if input == nil {
		return fmt.Errorf("%w, input can not be ", common.ErrNilPointer)
	}
	deadline := time.Now().Add(websocketRequestTimeout)
	ctx, cancelFunc := context.WithDeadline(context.Background(), deadline)
	defer func() {
		if time.Now().After(deadline) {
			cancelFunc()
		}
	}()
	for attempt := 1; ; attempt++ {
		// Initiate a rate limit reservation and sleep on requested endpoint
		err := d.Requester.InitiateRateLimit(ctx, ep)
		if err != nil {
			return fmt.Errorf("failed to rate limit Websocket request: %w", err)
		}

		if d.Verbose {
			log.Debugf(log.RequestSys, "%s attempt %d", d.Name, attempt)
		}
		var payload []byte
		payload, err = d.Websocket.Conn.SendMessageReturnResponse(context.TODO(), request.Unset, input.ID, input)
		if err != nil {
			return err
		}
		err = json.Unmarshal(payload, response)
		if err != nil {
			return err
		}
		switch response.Error.Code {
		case 10040:
			after := 100 * time.Millisecond // because all the request rate will be reset after 1 sec interval
			backoff := request.DefaultBackoff()(attempt)
			delay := max(after, backoff)

			if dl, ok := ctx.Deadline(); ok && dl.Before(time.Now().Add(delay)) {
				return errors.New("deadline would be exceeded by retry")
			}

			if d.Verbose {
				log.Errorf(log.RequestSys,
					"%s request has failed. Retrying request in %s, attempt %d",
					d.Name,
					delay,
					attempt)
			}
			time.Sleep(delay)
			continue
		default:
			return nil
		}
	}
}
