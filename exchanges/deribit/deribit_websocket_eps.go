package deribit

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// WSRetriveBookBySummary retrives book summary data for currency requested through websocket connection.
func (d *Deribit) WSRetriveBookBySummary(symbol, kind string) ([]BookSummaryData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	input := &struct {
		Currency string `json:"currency"`
		Kind     string `json:"kind,omitempty"`
	}{
		Currency: symbol,
	}
	if kind != "" {
		input.Kind = kind
	}
	var resp []BookSummaryData
	return resp, d.SendWSRequest(getBookByCurrency, input, &resp, false)
}

// WSRetriveBookSummaryByInstrument retrives book summary data for instrument requested through the websocket connection.
func (d *Deribit) WSRetriveBookSummaryByInstrument(instrument string) ([]BookSummaryData, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	input := &struct {
		Instrument string `json:"instrument_name,omitempty"`
	}{
		Instrument: instrument,
	}
	var resp []BookSummaryData
	return resp, d.SendWSRequest(getBookByInstrument, input, &resp, false)
}

// WSRetriveContractSize retrives contract size for instrument requested through the websocket connection.
func (d *Deribit) WSRetriveContractSize(instrument string) (*ContractSizeData, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	input := &struct {
		Instrument string `json:"instrument_name"`
	}{
		Instrument: instrument,
	}
	var resp *ContractSizeData
	return resp, d.SendWSRequest(getContractSize, input, &resp, false)
}

// WSRetriveCurrencies retrives all cryptocurrencies supported by the API through the websocket connection.
func (d *Deribit) WSRetriveCurrencies() ([]CurrencyData, error) {
	var resp []CurrencyData
	return resp, d.SendWSRequest(getCurrencies, nil, &resp, false)
}

// WSRetriveDeliveryPrices retrives delivery prices using index name through the websocket connection.
func (d *Deribit) WSRetriveDeliveryPrices(indexName string, offset, count int64) (*IndexDeliveryPrice, error) {
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
	return resp, d.SendWSRequest(getDeliveryPrices, input, &resp, false)
}

// WSRetriveFundingChartData retrives funding chart data for the requested instrument and time length through the websocket connection.
// supported lengths: 8h, 24h, 1m <-(1month)
func (d *Deribit) WSRetriveFundingChartData(instrument, length string) (*FundingChartData, error) {
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
	return resp, d.SendWSRequest(getFundingChartData, input, &resp, false)
}

// WSRetriveFundingRateValue retrives funding rate value data through the websocket connection.
func (d *Deribit) WSRetriveFundingRateValue(instrument string, startTime, endTime time.Time) (float64, error) {
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
	return resp, d.SendWSRequest(getFundingRateValue, input, &resp, false)
}

// WSRetriveHistoricalVolatility retrives historical volatility data
func (d *Deribit) WSRetriveHistoricalVolatility(symbol string) ([]HistoricalVolatilityData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	input := &struct {
		Currency string `json:"currency"`
	}{
		Currency: symbol,
	}
	var data [][2]interface{}
	err := d.SendWSRequest(getHistoricalVolatility, input, &data, false)
	if err != nil {
		return nil, err
	}
	resp := make([]HistoricalVolatilityData, len(data))
	for x := range data {
		timeData, ok := data[x][0].(float64)
		if !ok {
			return resp, fmt.Errorf("%v WSRetriveHistoricalVolatility: %w for time", d.Name, errTypeAssert)
		}
		val, ok := data[x][1].(float64)
		if !ok {
			return resp, fmt.Errorf("%v WSRetriveHistoricalVolatility: %w for val", d.Name, errTypeAssert)
		}
		resp[x] = HistoricalVolatilityData{
			Timestamp: timeData,
			Value:     val,
		}
	}
	return resp, nil
}

// WSRetriveCurrencyIndexPrice the current index price for the instruments, for the selected currency through the websocket connection.
func (d *Deribit) WSRetriveCurrencyIndexPrice(symbol string) (map[string]float64, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	input := &struct {
		Currency string `json:"currency"`
	}{
		Currency: symbol,
	}
	var resp map[string]float64
	return resp, d.SendWSRequest(getCurrencyIndexPrice, input, &resp, false)
}

// WSRetriveIndexPrice retrives price data for the requested index through the websocket connection.
func (d *Deribit) WSRetriveIndexPrice(index string) (*IndexPriceData, error) {
	if index == "" {
		return nil, fmt.Errorf("%w index can not be empty", errUnsupportedIndexName)
	}
	input := &struct {
		IndexName string `json:"index_name"`
	}{
		IndexName: index,
	}
	var resp *IndexPriceData
	return resp, d.SendWSRequest(getIndexPrice, input, &resp, false)
}

// WSRetriveIndexPriceNames names of indexes through the websocket connection.
func (d *Deribit) WSRetriveIndexPriceNames() ([]string, error) {
	var resp []string
	return resp, d.SendWSRequest(getIndexPriceNames, nil, &resp, false)
}

// WSRetriveInstrumentData retrives data for a requested instrument through the websocket connection.
func (d *Deribit) WSRetriveInstrumentData(instrument string) (*InstrumentData, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	input := &struct {
		Instrument string `json:"instrument_name"`
	}{
		Instrument: instrument,
	}
	var resp *InstrumentData
	return resp, d.SendWSRequest(getInstrument, input, &resp, false)
}

// WSRetriveInstrumentsData gets data for all available instruments
func (d *Deribit) WSRetriveInstrumentsData(symbol, kind string, expired bool) ([]InstrumentData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	input := &struct {
		Currency string `json:"currency"`
		Expired  bool   `json:"expired"`
		Kind     string `json:"kind,omitempty"`
	}{
		Currency: symbol,
		Expired:  expired,
		Kind:     kind,
	}
	var resp []InstrumentData
	return resp, d.SendWSRequest(getInstruments, input, &resp, false)
}

// WSRetriveLastSettlementsByCurrency retrives last settlement data by currency through the websocket connection.
func (d *Deribit) WSRetriveLastSettlementsByCurrency(symbol, settlementType, continuation string, count int64, startTime time.Time) (*SettlementsData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	input := &struct {
		Currency             string `json:"currency,omitempty"`
		Type                 string `json:"type,omitempty"`
		Continuation         string `json:"continuation,omitempty"`
		Count                int64  `json:"count,omitempty"`
		SearchStartTimestamp int64  `json:"search_start_timestamp,omitempty"`
	}{
		Currency:             symbol,
		Type:                 settlementType,
		Continuation:         continuation,
		Count:                count,
		SearchStartTimestamp: startTime.UnixMilli(),
	}
	var resp *SettlementsData
	return resp, d.SendWSRequest(getLastSettlementsByCurrency, input, &resp, false)
}

// WSRetriveLastSettlementsByInstrument retrives last settlement data for requested instrument through the websocket connection.
func (d *Deribit) WSRetriveLastSettlementsByInstrument(instrument, settlementType, continuation string, count int64, startTime time.Time) (*SettlementsData, error) {
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
	return resp, d.SendWSRequest(getLastSettlementsByInstrument, input, &resp, false)
}

// WSRetriveLastTradesByCurrency retrives last trades for requested currency through the websocket connection.
func (d *Deribit) WSRetriveLastTradesByCurrency(symbol, kind, startID, endID, sorting string, count int64, includeOld bool) (*PublicTradesData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	input := &struct {
		Currency   string `json:"currency"`
		Kind       string `json:"kind,omitempty"`
		StartID    string `json:"start_id,omitempty"`
		EndID      string `json:"end_id,omitempty"`
		Count      int64  `json:"count,omitempty"`
		IncludeOld bool   `json:"include_old,omitempty"`
		Sorting    string `json:"sorting,omitempty"`
	}{
		Currency:   symbol,
		Kind:       kind,
		StartID:    startID,
		EndID:      endID,
		Count:      count,
		IncludeOld: includeOld,
		Sorting:    sorting,
	}
	var resp *PublicTradesData
	return resp, d.SendWSRequest(getLastTradesByCurrency, input, &resp, false)
}

// WSRetriveLastTradesByCurrencyAndTime retrives last trades for requested currency and time intervals through the websocket connection.
func (d *Deribit) WSRetriveLastTradesByCurrencyAndTime(symbol, kind, sorting string, count int64, includeOld bool, startTime, endTime time.Time) (*PublicTradesData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
		return nil, err
	}
	input := &struct {
		Currency       string `json:"currency"`
		Kind           string `json:"kind,omitempty"`
		Sorting        string `json:"sorting,omitempty"`
		Count          int64  `json:"count,omitempty"`
		StartTimestamp int64  `json:"start_timestamp,omitempty"`
		EndTimestamp   int64  `json:"end_timestamp,omitempty"`
		IncludeOld     bool   `json:"include_old,omitempty"`
	}{
		Currency:       symbol,
		Kind:           kind,
		Count:          count,
		StartTimestamp: startTime.UnixMilli(),
		EndTimestamp:   endTime.UnixMilli(),
		IncludeOld:     includeOld,
		Sorting:        sorting,
	}
	var resp *PublicTradesData
	return resp, d.SendWSRequest(getLastTradesByCurrencyAndTime, input, &resp, false)
}

// WSRetriveLastTradesByInstrument retrives last trades for requested instrument requested through the websocket connection.
func (d *Deribit) WSRetriveLastTradesByInstrument(instrument, startSeq, endSeq, sorting string, count int64, includeOld bool) (*PublicTradesData, error) {
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
	return resp, d.SendWSRequest(getLastTradesByInstrument, input, &resp, false)
}

// WSRetriveLastTradesByInstrumentAndTime retrives last trades for requested instrument requested and time intervals through the websocket connection.
func (d *Deribit) WSRetriveLastTradesByInstrumentAndTime(instrument, sorting string, count int64, includeOld bool, startTime, endTime time.Time) (*PublicTradesData, error) {
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
	return resp, d.SendWSRequest(getLastTradesByInstrumentAndTime, input, &resp, false)
}

// WSRetriveMarkPriceHistory retrives data for mark price history through the websocket connection.
func (d *Deribit) WSRetriveMarkPriceHistory(instrument string, startTime, endTime time.Time) ([]MarkPriceHistory, error) {
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
	return resp, d.SendWSRequest(getMarkPriceHistory, input, &resp, false)
}

// WSRetriveOrderbookData retrives data orderbook of requested instrument through the web-socket connection.
func (d *Deribit) WSRetriveOrderbookData(instrument string, depth int64) (*Orderbook, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	input := &struct {
		Instrument string `json:"instrument_name,omitempty"`
		Depth      int64  `json:"depth,omitempty"`
	}{
		Instrument: instrument,
		Depth:      depth,
	}
	var resp *Orderbook
	return resp, d.SendWSRequest(getOrderbook, input, &resp, false)
}

// WSRetriveOrderbookByInstrumentID retrives orderbook by instrument ID through websocket connection.
func (d *Deribit) WSRetriveOrderbookByInstrumentID(instrumentID int64, depth float64) (*Orderbook, error) {
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
	return resp, d.SendWSRequest(getOrderbookByInstrumentID, input, &resp, false)
}

// WSRetriveRequestForQuote retrives RFQ information.
func (d *Deribit) WSRetriveRequestForQuote(symbol, kind string) ([]RequestForQuote, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	input := &struct {
		Currency string `json:"currency"`
		Kind     string `json:"kind,omitempty"`
	}{
		Currency: symbol,
		Kind:     kind,
	}
	var resp []RequestForQuote
	return resp, d.SendWSRequest(getRFQ, input, &resp, false)
}

// WSRetriveTradeVolumes retrives trade volumes' data of all instruments through the websocket connection.
func (d *Deribit) WSRetriveTradeVolumes(extended bool) ([]TradeVolumesData, error) {
	input := &struct {
		Extended bool `json:"extended,omitempty"`
	}{
		Extended: extended,
	}
	var resp []TradeVolumesData
	return resp, d.SendWSRequest(getTradeVolumes, input, &resp, false)
}

// WSRetrivesTradingViewChartData retrives volatility index data for the requested instrument through the websocket connection.
func (d *Deribit) WSRetrivesTradingViewChartData(instrument, resolution string, startTime, endTime time.Time) (*TVChartData, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
		return nil, err
	}
	if resolution == "" {
		return nil, fmt.Errorf("unsupported resolution, resolution can not be empty")
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
	return resp, d.SendWSRequest(getTradingViewChartData, input, &resp, false)
}

// WSRetriveVolatilityIndexData retrives volatility index data for the requested currency through the websocket connection.
func (d *Deribit) WSRetriveVolatilityIndexData(symbol, resolution string, startTime, endTime time.Time) ([]VolatilityIndexData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	err := common.StartEndTimeCheck(startTime, endTime)
	if err != nil {
		return nil, err
	}
	if resolution == "" {
		return nil, fmt.Errorf("unsupported resolution, resolution can not be empty")
	}
	input := &struct {
		Currency       string `json:"currency,omitempty"`
		StartTimestamp int64  `json:"start_timestamp,omitempty"`
		EndTimestamp   int64  `json:"end_timestamp,omitempty"`
		Resolution     string `json:"resolution,omitempty"`
	}{
		Currency:       symbol,
		Resolution:     resolution,
		StartTimestamp: startTime.UnixMilli(),
		EndTimestamp:   endTime.UnixMilli(),
	}
	var resp VolatilityIndexRawData
	err = d.SendWSRequest(getVolatilityIndexData, input, &resp, false)
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

// WSRetrivePublicTicker retrives public ticker data of the instrument requested through the websocket connection.
func (d *Deribit) WSRetrivePublicTicker(instrument string) (*TickerData, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	input := &struct {
		Instrument string `json:"instrument_name,omitempty"`
	}{
		Instrument: instrument,
	}
	var resp *TickerData
	return resp, d.SendWSRequest(getTicker, input, &resp, false)
}

// WSRetriveAccountSummary retrives account summary data for the requested instrument through the websocket connection.
func (d *Deribit) WSRetriveAccountSummary(symbol string, extended bool) (*AccountSummaryData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	input := &struct {
		Currency string `json:"currency"`
		Extended bool   `json:"extended"`
	}{
		Currency: symbol,
		Extended: extended,
	}
	var resp *AccountSummaryData
	return resp, d.SendWSRequest(getAccountSummary, input, &resp, true)
}

// WSCancelWithdrawal cancels withdrawal request for a given currency by its id through the websocket connection.
func (d *Deribit) WSCancelWithdrawal(symbol string, id int64) (*CancelWithdrawalData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	if id <= 0 {
		return nil, fmt.Errorf("%w, withdrawal id has to be positive integer", errInvalidID)
	}
	input := &struct {
		Currency string `json:"currency"`
		ID       int64  `json:"id"`
	}{
		Currency: symbol,
		ID:       id,
	}
	var resp *CancelWithdrawalData
	return resp, d.SendWSRequest(cancelWithdrawal, input, &resp, true)
}

// WSCancelTransferByID cancels transfer by ID through the websocket connection.
func (d *Deribit) WSCancelTransferByID(symbol, tfa string, id int64) (*AccountSummaryData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	if id <= 0 {
		return nil, fmt.Errorf("%w, transfer id has to be positive integer", errInvalidID)
	}
	input := &struct {
		Currency                    string `json:"currency"`
		TwoFactorAuthenticationCode string `json:"tfa,omitempty"`
		ID                          int64  `json:"id"`
	}{
		Currency:                    symbol,
		ID:                          id,
		TwoFactorAuthenticationCode: tfa,
	}
	var resp *AccountSummaryData
	return resp, d.SendWSRequest(cancelTransferByID, input, &resp, true)
}

// WSCreateDepositAddress creates a deposit address for the currency requested through the websocket connection.
func (d *Deribit) WSCreateDepositAddress(symbol string) (*DepositAddressData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	input := &struct {
		Currency string `json:"currency"`
	}{
		Currency: symbol,
	}
	var resp *DepositAddressData
	return resp, d.SendWSRequest(createDepositAddress, input, &resp, true)
}

// WSRetriveDeposits retrives the deposits of a given currency through the websocket connection.
func (d *Deribit) WSRetriveDeposits(symbol string, count, offset int64) (*DepositsData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	input := &struct {
		Currency string `json:"currency"`
		Count    int64  `json:"count,omitempty"`
		Offset   int64  `json:"offset,omitempty"`
	}{
		Currency: symbol,
		Count:    count,
		Offset:   offset,
	}
	var resp *DepositsData
	return resp, d.SendWSRequest(getDeposits, input, &resp, true)
}

// WSRetriveTransfers retrives data for the requested currency through the websocket connection.
func (d *Deribit) WSRetriveTransfers(symbol string, count, offset int64) (*TransferData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	input := &struct {
		Currency string `json:"currency,omitempty"`
		Count    int64  `json:"count,omitempty"`
		Offset   int64  `json:"offset,omitempty"`
	}{
		Currency: symbol,
		Count:    count,
		Offset:   offset,
	}
	var resp *TransferData
	return resp, d.SendWSRequest(getTransfers, input, &resp, true)
}

// WSRetriveCurrentDepositAddress retrives the current deposit address for the requested currency through the websocket connection.
func (d *Deribit) WSRetriveCurrentDepositAddress(symbol string) (*DepositAddressData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	input := &struct {
		Currency string `json:"currency"`
	}{
		Currency: symbol,
	}
	var resp *DepositAddressData
	return resp, d.SendWSRequest(getCurrentDepositAddress, input, &resp, true)
}

// WSRetriveWithdrawals retrives withdrawals data for a requested currency through the websocket connection.
func (d *Deribit) WSRetriveWithdrawals(symbol string, count, offset int64) (*WithdrawalsData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	input := &struct {
		Currency string `json:"currency,omitempty"`
		Count    int64  `json:"count,omitempty"`
		Offset   int64  `json:"offset,omitempty"`
	}{
		Currency: symbol,
		Count:    count,
		Offset:   offset,
	}
	var resp *WithdrawalsData
	return resp, d.SendWSRequest(getWithdrawals, input, &resp, true)
}

// WSSubmitTransferToSubAccount submits a request to transfer a currency to a subaccount
func (d *Deribit) WSSubmitTransferToSubAccount(symbol string, amount float64, destinationID int64) (*TransferData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
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
		Currency:    symbol,
		Destination: destinationID,
		Amount:      amount,
	}
	var resp *TransferData
	return resp, d.SendWSRequest(submitTransferToSubaccount, input, &resp, true)
}

// WSSubmitTransferToUser submits a request to transfer a currency to another user through the websocket connection.
func (d *Deribit) WSSubmitTransferToUser(symbol, tfa, destinationAddress string, amount float64) (*TransferData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	if amount <= 0 {
		return nil, errInvalidAmount
	}
	if destinationAddress == "" {
		return nil, errInvalidDestinationID
	}
	input := &struct {
		Currency                    string  `json:"currency"`
		TwoFactorAuthenticationCode string  `json:"tfa,omitempty"`
		DestinationID               string  `json:"destination"`
		Amount                      float64 `json:"amount"`
	}{
		Currency:                    symbol,
		TwoFactorAuthenticationCode: tfa,
		DestinationID:               destinationAddress,
		Amount:                      amount,
	}
	var resp *TransferData
	return resp, d.SendWSRequest(submitTransferToUser, input, &resp, true)
}

// ----------------------------------------------------------------------------

// WSSubmitWithdraw submits a withdrawal request to the exchange for the requested currency through the websocket connection.
func (d *Deribit) WSSubmitWithdraw(symbol, address, priority string, amount float64) (*WithdrawData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
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
		Currency: symbol,
		Address:  address,
		Priority: priority,
		Amount:   amount,
	}
	var resp *WithdrawData
	return resp, d.SendWSRequest(submitWithdraw, input, &resp, true)
}

// WSRetriveAnnouncements retrieves announcements through the websocket connection. Default "start_timestamp" parameter value is current timestamp, "count" parameter value must be between 1 and 50, default is 5.
func (d *Deribit) WSRetriveAnnouncements(startTime time.Time, count int64) ([]Announcement, error) {
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
	return resp, d.SendWSRequest(getAnnouncements, input, &resp, false)
}

// WSRetrivePublicPortfolioMargins public version of the method calculates portfolio margin info for simulated position. For concrete user position, the private version of the method must be used. The public version of the request has special restricted rate limit (not more than once per a second for the IP).
func (d *Deribit) WSRetrivePublicPortfolioMargins(symbol string, simulatedPositions map[string]float64) (*PortfolioMargin, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	input := &struct {
		Currency           string             `json:"currency"`
		SimulatedPositions map[string]float64 `json:"simulated_positions"`
	}{
		Currency: symbol,
	}
	if len(simulatedPositions) != 0 {
		input.SimulatedPositions = simulatedPositions
	}
	var resp *PortfolioMargin
	return resp, d.SendWSRequest(getPublicPortfolioMargins, input, &resp, false)
}

// WSChangeAPIKeyName changes the name of the api key requested through the websocket connection.
func (d *Deribit) WSChangeAPIKeyName(id int64, name string) (*APIKeyData, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w, invalid api key id", errInvalidID)
	}
	if !alphaNumericRegExp.MatchString(name) {
		return nil, errors.New("unacceptable api key name")
	}
	input := &struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}{
		ID:   id,
		Name: name,
	}
	var resp *APIKeyData
	return resp, d.SendWSRequest(changeAPIKeyName, input, &resp, true)
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
	return resp, d.SendWSRequest(changeScopeInAPIKey, input, &resp, true)
}

// WSChangeSubAccountName retrives changes the name of the requested subaccount id through the websocket connection.
func (d *Deribit) WSChangeSubAccountName(sid int64, name string) error {
	if sid <= 0 {
		return fmt.Errorf("%w, invalid subaccount user id", errInvalidID)
	}
	if name == "" {
		return errors.New("new username has to be specified")
	}
	input := &struct {
		SID  int64  `json:"sid"`
		Name string `json:"name"`
	}{
		SID:  sid,
		Name: name,
	}
	var resp string
	err := d.SendWSRequest(changeSubAccountName, input, &resp, true)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return fmt.Errorf("subaccount name change failed")
	}
	return nil
}

// WSCreateAPIKey creates an api key based on the provided settings through the websocket connection.
func (d *Deribit) WSCreateAPIKey(maxScope, name string, defaultKey bool) (interface{}, error) {
	input := &struct {
		MaxScope string `json:"max_scope"`
		Name     string `json:"name,omitempty"`
		Default  bool   `json:"default"`
	}{
		MaxScope: maxScope,
		Name:     name,
		Default:  defaultKey,
	}
	var result json.RawMessage
	err := d.SendWSRequest(createAPIKey, input, &result, true)
	if err != nil {
		return nil, err
	}
	challenge := &TFAChallenge{}
	err = json.Unmarshal(result, challenge)
	if err == nil && challenge.SecurityKeyAuthorizationRequired {
		return challenge, nil
	}
	var resp APIKeyData
	err = json.Unmarshal(result, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// WSCreateSubAccount creates a new subaccount through the websocket connection.
func (d *Deribit) WSCreateSubAccount() (*SubAccountData, error) {
	var resp *SubAccountData
	return resp, d.SendWSRequest(createSubAccount, nil, &resp, true)
}

// WSDisableAPIKey disables the api key linked to the provided id through the websocket connection.
func (d *Deribit) WSDisableAPIKey(id int64) (interface{}, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w, invalid api key id", errInvalidID)
	}
	input := &struct {
		ID int64 `json:"id"`
	}{
		ID: id,
	}
	var response json.RawMessage
	err := d.SendWSRequest(disableAPIKey, input, &response, true)
	if err != nil {
		return nil, err
	}
	challenge := &TFAChallenge{}
	err = json.Unmarshal(response, challenge)
	if err == nil && challenge.SecurityKeyAuthorizationRequired {
		return challenge, nil
	}
	var resp APIKeyData
	err = json.Unmarshal(response, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// WSDisableTFAForSubAccount disables two factor authentication for the subaccount linked to the requested id through the websocket connection.
func (d *Deribit) WSDisableTFAForSubAccount(sid int64) error {
	if sid <= 0 {
		return fmt.Errorf("%w, invalid subaccount user id", errInvalidID)
	}
	input := &struct {
		SID int64 `json:"sid"`
	}{
		SID: sid,
	}
	var resp string
	err := d.SendWSRequest(disableTFAForSubaccount, input, &resp, true)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return fmt.Errorf("disabling 2fa for subaccount %v failed", sid)
	}
	return nil
}

// WSEnableAffiliateProgram enables the affiliate program through the websocket connection.
func (d *Deribit) WSEnableAffiliateProgram() error {
	var resp string
	err := d.SendWSRequest(enableAffiliateProgram, nil, &resp, true)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return fmt.Errorf("could not enable affiliate program")
	}
	return nil
}

// WSEnableAPIKey enables the api key linked to the provided id through the websocket connection.
func (d *Deribit) WSEnableAPIKey(id int64) (interface{}, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w, invalid api key id", errInvalidID)
	}
	var response json.RawMessage
	err := d.SendWSRequest(enableAPIKey, map[string]int64{"id": id}, &response, true)
	if err != nil {
		return nil, err
	}
	challenge := &TFAChallenge{}
	err = json.Unmarshal(response, challenge)
	if err == nil && challenge.SecurityKeyAuthorizationRequired {
		return challenge, nil
	}
	var resp APIKeyData
	err = json.Unmarshal(response, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// WSRetriveAccessLog lists access logs for the user through the websocket connection.
func (d *Deribit) WSRetriveAccessLog(offset, count int64) (*AccessLog, error) {
	input := &struct {
		Offset int64 `json:"offset,omitempty"`
		Count  int64 `json:"count,omitempty"`
	}{
		Offset: offset,
		Count:  count,
	}
	var resp *AccessLog
	return resp, d.SendWSRequest(getAccessLog, input, &resp, true)
}

// WSRetriveAffiliateProgramInfo retrives the affiliate program info through the websocket connection.
func (d *Deribit) WSRetriveAffiliateProgramInfo(id int64) (*AffiliateProgramInfo, error) {
	var resp *AffiliateProgramInfo
	return resp, d.SendWSRequest(getAffiliateProgramInfo, nil, &resp, true)
}

// WSRetriveEmailLanguage retrives the current language set for the email through the websocket connection.
func (d *Deribit) WSRetriveEmailLanguage() (string, error) {
	var resp string
	return resp, d.SendWSRequest(getEmailLanguage, nil, &resp, true)
}

// WSRetriveNewAnnouncements retrives new announcements through the websocket connection.
func (d *Deribit) WSRetriveNewAnnouncements() ([]Announcement, error) {
	var resp []Announcement
	return resp, d.SendWSRequest(getNewAnnouncements, nil, &resp, true)
}

// WSRetrivePricatePortfolioMargins alculates portfolio margin info for simulated position or current position of the user through the websocket connection. This request has special restricted rate limit (not more than once per a second).
func (d *Deribit) WSRetrivePricatePortfolioMargins(symbol string, accPositions bool, simulatedPositions map[string]float64) (*PortfolioMargin, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	input := &struct {
		Currency           string             `json:"currency"`
		AccountPositions   bool               `json:"acc_positions,omitempty"`
		SimulatedPositions map[string]float64 `json:"simulated_positions,omitempty"`
	}{
		Currency:         symbol,
		AccountPositions: accPositions,
	}
	if len(simulatedPositions) != 0 {
		input.SimulatedPositions = simulatedPositions
	}
	var resp *PortfolioMargin
	return resp, d.SendWSRequest(getPrivatePortfolioMargins, input, &resp, true)
}

// WSRetrivePosition retrives the data of all positions in the requested instrument name through the websocket connection.
func (d *Deribit) WSRetrivePosition(instrument string) (*PositionData, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	var resp *PositionData
	return resp, d.SendWSRequest(getPosition, map[string]string{"instrument_name": instrument}, &resp, true)
}

// WSRetriveSubAccounts retrives all subaccounts' data through the websocket connection.
func (d *Deribit) WSRetriveSubAccounts(withPortfolio bool) ([]SubAccountData, error) {
	var resp []SubAccountData
	return resp, d.SendWSRequest(getSubAccounts, map[string]bool{"with_portfolio": withPortfolio}, &resp, true)
}

// WSRetriveSubAccountDetails retrives sub-account detail information through the websocket connection.
func (d *Deribit) WSRetriveSubAccountDetails(symbol string, withOpenOrders bool) ([]SubAccountDetail, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	input := &struct {
		Currency       string `json:"currency"`
		WithOpenOrders bool   `json:"with_open_orders,omitempty"`
	}{
		Currency:       symbol,
		WithOpenOrders: withOpenOrders,
	}
	var resp []SubAccountDetail
	return resp, d.SendWSRequest(getSubAccountDetails, input, &resp, true)
}

// WSRetrivePositions retrives positions data of the user account through the websocket connection.
func (d *Deribit) WSRetrivePositions(symbol, kind string) ([]PositionData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	input := &struct {
		Currency string `json:"currency"`
		Kind     string `json:"kind,omitempty"`
	}{
		Currency: symbol,
		Kind:     kind,
	}
	var resp []PositionData
	return resp, d.SendWSRequest(getPositions, input, &resp, true)
}

// WSRetriveTransactionLog retrives transaction logs data through the websocket connection.
func (d *Deribit) WSRetriveTransactionLog(symbol, query string, startTime, endTime time.Time, count, continuation int64) (*TransactionsData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
		return nil, err
	}
	input := &struct {
		Currency       string `json:"currency"`
		Query          string `json:"query,omitempty"`
		StartTimestamp int64  `json:"start_timestamp,omitempty"`
		EndTimestamp   int64  `json:"end_timestamp,omitempty"`
		Count          int64  `json:"count,omitempty"`
		Continuation   int64  `json:"continuation,omitempty"`
	}{
		Currency:       symbol,
		Query:          query,
		StartTimestamp: startTime.UnixMilli(),
		EndTimestamp:   endTime.UnixMilli(),
		Count:          count,
		Continuation:   continuation,
	}
	var resp *TransactionsData
	return resp, d.SendWSRequest(getTransactionLog, input, &resp, true)
}

// WSRetriveUserLocks retrieves information about locks on user account through the websocket connecton.
func (d *Deribit) WSRetriveUserLocks() ([]UserLock, error) {
	var resp []UserLock
	return resp, d.SendWSRequest(getUserLocks, nil, &resp, true)
}

// WSListAPIKeys retrives all the api keys associated with a user account through the websocket connection.
func (d *Deribit) WSListAPIKeys(tfa string) ([]APIKeyData, error) {
	var resp []APIKeyData
	return resp, d.SendWSRequest(listAPIKeys, map[string]string{"tfa": tfa}, &resp, true)
}

// WSRemoveAPIKey removes api key vid ID through the websocket connection.
func (d *Deribit) WSRemoveAPIKey(id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w, invalid api key id", errInvalidID)
	}
	var resp interface{}
	err := d.SendWSRequest(removeAPIKey, map[string]int64{"id": id}, &resp, true)
	if err != nil {
		return err
	}
	_, ok := resp.(map[string]interface{})
	if ok {
		data, err := json.Marshal(resp)
		if err != nil {
			return err
		}
		var respo TFAChallenge
		err = json.Unmarshal(data, &respo)
		if err != nil {
			return err
		}
		return nil
	}
	if resp != "ok" {
		return fmt.Errorf("removal of the api key requested failed")
	}
	return nil
}

// WSRemoveSubAccount removes a subaccount given its id through the websocket connection.
func (d *Deribit) WSRemoveSubAccount(subAccountID int64) error {
	var resp string
	err := d.SendWSRequest(removeSubAccount, map[string]int64{"subaccount_id": subAccountID}, &resp, true)
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
	err := d.SendWSRequest(setAnnouncementAsRead, map[string]int64{"announcement_id": id}, &resp, true)
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
	var resp interface{}
	err := d.SendWSRequest(setEmailForSubAccount, input, &resp, true)
	if err != nil {
		return err
	}
	_, ok := resp.(map[string]interface{})
	if ok {
		data, err := json.Marshal(resp)
		if err != nil {
			return err
		}
		var respo TFAChallenge
		err = json.Unmarshal(data, &respo)
		if err != nil {
			return err
		}
		return nil
	}
	if resp != "ok" {
		return fmt.Errorf("could not link email (%v) to subaccount %v", email, sid)
	}
	return nil
}

// WSSetEmailLanguage sets a requested language for an email through the websocket connecton.
func (d *Deribit) WSSetEmailLanguage(language string) error {
	if language == "" {
		return errors.New("language is required")
	}
	var resp string
	err := d.SendWSRequest(setEmailLanguage, map[string]string{"language": language}, &resp, true)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return fmt.Errorf("could not set the email language to %v", language)
	}
	return nil
}

// WSSetPasswordForSubAccount sets a password for subaccount usage through the websocket connection.
func (d *Deribit) WSSetPasswordForSubAccount(sid int64, password string) (interface{}, error) {
	if sid <= 0 {
		return "", fmt.Errorf("%w, invalid subaccount user id", errInvalidID)
	}
	if password == "" {
		return "", errors.New("subaccount password must not be empty")
	}
	input := &struct {
		Password string `json:"password"`
		SID      int64  `json:"sid"`
	}{
		Password: password,
		SID:      sid,
	}
	var resp interface{}
	err := d.SendWSRequest(setPasswordForSubAccount, input, &resp, true)
	if err != nil {
		return "", err
	}
	_, ok := resp.(map[string]interface{})
	if ok {
		data, err := json.Marshal(resp)
		if err != nil {
			return "", err
		}
		var respo TFAChallenge
		err = json.Unmarshal(data, &respo)
		if err != nil {
			return nil, err
		}
		return respo, nil
	}
	if resp != "ok" {
		return "", fmt.Errorf("could not set the provided password to subaccount %v", sid)
	}
	return "ok", nil
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
	err := d.SendWSRequest(toggleNotificationsFromSubAccount, input, &resp, true)
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
		return nil, errors.New("missing user id")
	}
	input := &struct {
		UserID  int64 `json:"user_id"`
		Enabled bool  `json:"enabled"`
		DryRun  bool  `json:"dry_run,omitempty"`
	}{
		UserID:  userID,
		Enabled: enabled,
		DryRun:  dryRun,
	}
	var resp []TogglePortfolioMarginResponse
	return resp, d.SendWSRequest(togglePortfolioMargining, input, &resp, true)
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
	err := d.SendWSRequest(toggleSubAccountLogin, input, &resp, true)
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
	if arg == nil {
		return nil, fmt.Errorf("%s parameter is required", common.ErrNilPointer)
	}
	if arg.Instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	var resp *PrivateTradeData
	return resp, d.SendWSRequest(submitBuy, &arg, &resp, true)
}

// WSSubmitSell submits a sell request with the parameters provided through the websocket connection.
func (d *Deribit) WSSubmitSell(arg *OrderBuyAndSellParams) (*PrivateTradeData, error) {
	if arg == nil {
		return nil, fmt.Errorf("%s parameter is required", common.ErrNilPointer)
	}
	if arg.Instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	var resp *PrivateTradeData
	return resp, d.SendWSRequest(submitSell, &arg, &resp, true)
}

// WSSubmitEdit submits an edit order request through the websocket connection.
func (d *Deribit) WSSubmitEdit(arg *OrderBuyAndSellParams) (*PrivateTradeData, error) {
	if arg.OrderID == "" {
		return nil, fmt.Errorf("%w, order id is required", errInvalidID)
	}
	if arg.Amount <= 0 {
		return nil, errInvalidAmount
	}
	var resp *PrivateTradeData
	return resp, d.SendWSRequest(submitEdit, &arg, &resp, true)
}

// WSEditOrderByLabel submits an edit order request sorted via label through the websocket connection.
func (d *Deribit) WSEditOrderByLabel(arg *OrderBuyAndSellParams) (*PrivateTradeData, error) {
	if arg == nil {
		return nil, fmt.Errorf("%w argument cannot be null", common.ErrNilPointer)
	}
	if arg.Instrument == "" {
		return nil, errInvalidInstrumentName
	}
	if arg.Amount <= 0 {
		return nil, errInvalidAmount
	}
	var resp *PrivateTradeData
	return resp, d.SendWSRequest(editByLabel, &arg, &resp, true)
}

// WSSubmitCancel sends a request to cancel the order via its orderID through the websocket connection.
func (d *Deribit) WSSubmitCancel(orderID string) (*PrivateCancelData, error) {
	if orderID == "" {
		return nil, fmt.Errorf("%w, no order ID specified", errInvalidID)
	}
	var resp *PrivateCancelData
	return resp, d.SendWSRequest(submitCancel, map[string]string{"order_id": orderID}, &resp, true)
}

// WSSubmitCancelAll sends a request to cancel all user orders in all currencies and instruments
func (d *Deribit) WSSubmitCancelAll() (int64, error) {
	var resp int64
	return resp, d.SendWSRequest(submitCancelAll, nil, &resp, true)
}

// WSSubmitCancelAllByCurrency sends a request to cancel all user orders for the specified currency through the websocket connection.
func (d *Deribit) WSSubmitCancelAllByCurrency(symbol, kind, orderType string) (int64, error) {
	if symbol == "" {
		return 0, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	input := &struct {
		Currency  string `json:"currency"`
		Kind      string `json:"kind"`
		OrderType string `json:"order_type,omitempty"`
	}{
		Currency:  symbol,
		Kind:      kind,
		OrderType: orderType,
	}
	var resp int64
	return resp, d.SendWSRequest(submitCancelAllByCurrency, input, &resp, true)
}

// WSSubmitCancelAllByInstrument sends a request to cancel all user orders for the specified instrument through the websocket connection.
func (d *Deribit) WSSubmitCancelAllByInstrument(instrument, orderType string) (int64, error) {
	if instrument == "" {
		return 0, errInvalidInstrumentName
	}
	input := &struct {
		Instrument string `json:"instrument_name"`
		OrderType  string `json:"order_type,omitempty"`
	}{
		Instrument: instrument,
		OrderType:  orderType,
	}
	var resp int64
	return resp, d.SendWSRequest(submitCancelAllByInstrument, input, &resp, true)
}

// WSSubmitCancelByLabel sends a request to cancel all user orders for the specified label through the websocket connection.
func (d *Deribit) WSSubmitCancelByLabel(label, symbol string) (int64, error) {
	input := &struct {
		Label    string `json:"label"`
		Currency string `json:"currency,omitempty"`
	}{
		Label:    label,
		Currency: symbol,
	}
	var resp int64
	return resp, d.SendWSRequest(submitCancelByLabel, input, &resp, true)
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
	return resp, d.SendWSRequest(submitClosePosition, input, &resp, true)
}

// WSRetriveMargins sends a request to fetch account margins data through the websocket connection.
func (d *Deribit) WSRetriveMargins(instrument string, amount, price float64) (*MarginsData, error) {
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
	return resp, d.SendWSRequest(getMargins, input, &resp, true)
}

// WSRetriveMMPConfig sends a request to fetch the config for MMP of the requested currency through the websocket connection.
func (d *Deribit) WSRetriveMMPConfig(symbol string) (*MMPConfigData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	var resp *MMPConfigData
	return resp, d.SendWSRequest(getMMPConfig, map[string]string{"currency": symbol}, &resp, true)
}

// WSRetriveOpenOrdersByCurrency sends a request to fetch open orders data sorted by requested params
func (d *Deribit) WSRetriveOpenOrdersByCurrency(symbol, kind, orderType string) ([]OrderData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	input := &struct {
		Currency  string `json:"currency"`
		Kind      string `json:"kind,omitempty"`
		OrderType string `json:"type,omitempty"`
	}{
		Currency:  symbol,
		Kind:      kind,
		OrderType: orderType,
	}
	var resp []OrderData
	return resp, d.SendWSRequest(getOpenOrdersByCurrency, input, &resp, true)
}

// WSRetriveOpenOrdersByInstrument sends a request to fetch open orders data sorted by requested params through the websocket connection.
func (d *Deribit) WSRetriveOpenOrdersByInstrument(instrument, orderType string) ([]OrderData, error) {
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
	return resp, d.SendWSRequest(getOpenOrdersByInstrument, input, &resp, true)
}

// WSRetriveOrderHistoryByCurrency sends a request to fetch order history according to given params and currency through the websocket connection.
func (d *Deribit) WSRetriveOrderHistoryByCurrency(symbol, kind string, count, offset int64, includeOld, includeUnfilled bool) ([]OrderData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	input := &struct {
		Currency        string `json:"currency"`
		Kind            string `json:"kind,omitempty"`
		Count           int64  `json:"count,omitempty"`
		Offset          int64  `json:"offset,omitempty"`
		IncludeOld      bool   `json:"include_old,omitempty"`
		IncludeUnfilled bool   `json:"include_unfilled,omitempty"`
	}{
		Currency:        symbol,
		Kind:            kind,
		Count:           count,
		Offset:          offset,
		IncludeOld:      includeOld,
		IncludeUnfilled: includeUnfilled,
	}
	var resp []OrderData
	return resp, d.SendWSRequest(getOrderHistoryByCurrency, input, &resp, true)
}

// WSRetriveOrderHistoryByInstrument sends a request to fetch order history according to given params and instrument through the websocket connection.
func (d *Deribit) WSRetriveOrderHistoryByInstrument(instrument string, count, offset int64, includeOld, includeUnfilled bool) ([]OrderData, error) {
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
	return resp, d.SendWSRequest(getOrderHistoryByInstrument, input, &resp, true)
}

// WSRetriveOrderMarginsByID sends a request to fetch order margins data according to their ids through the websocket connection.
func (d *Deribit) WSRetriveOrderMarginsByID(ids []string) ([]OrderData, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("%w, order ids cannot be empty", errInvalidID)
	}
	var resp []OrderData
	return resp, d.SendWSRequest(getOrderMarginByIDs, map[string][]string{"ids": ids}, &resp, true)
}

// WSRetrivesOrderState sends a request to fetch order state of the order id provided
func (d *Deribit) WSRetrivesOrderState(orderID string) (*OrderData, error) {
	if orderID == "" {
		return nil, fmt.Errorf("%w, no order ID specified", errInvalidID)
	}
	var resp *OrderData
	return resp, d.SendWSRequest(getOrderState, map[string]string{"order_id": orderID}, &resp, true)
}

// WSRetriveTriggerOrderHistory sends a request to fetch order state of the order id provided through the websocket connection.
func (d *Deribit) WSRetriveTriggerOrderHistory(symbol, instrumentName, continuation string, count int64) (*OrderData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	input := &struct {
		Currency     string `json:"currency,omitempty"`
		Instrument   string `json:"instrument,omitempty"`
		Continuation string `json:"continuation,omitempty"`
		Count        int64  `json:"count,omitempty"`
	}{
		Currency:     symbol,
		Instrument:   instrumentName,
		Continuation: continuation,
		Count:        count,
	}
	var resp *OrderData
	return resp, d.SendWSRequest(getTriggerOrderHistory, input, &resp, true)
}

// WSRetriveUserTradesByCurrency sends a request to fetch user trades sorted by currency through the websocket connection.
func (d *Deribit) WSRetriveUserTradesByCurrency(symbol, kind, startID, endID, sorting string, count int64, includeOld bool) (*UserTradesData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	input := &struct {
		Currency   string `json:"currency"`
		Kind       string `json:"kind"`
		StartID    string `json:"start_id,omitempty"`
		EndID      string `json:"end_id,omitempty"`
		Sorting    string `json:"sorting,omitempty"`
		Count      int64  `json:"count,omitempty"`
		IncludeOld bool   `json:"include_old,omitempty"`
	}{
		Currency:   symbol,
		Kind:       kind,
		StartID:    startID,
		EndID:      endID,
		Sorting:    sorting,
		Count:      count,
		IncludeOld: includeOld,
	}
	var resp *UserTradesData
	return resp, d.SendWSRequest(getUserTradesByCurrency, input, &resp, true)
}

// WSRetriveUserTradesByCurrencyAndTime retrives user trades sorted by currency and time through the websocket connection.
func (d *Deribit) WSRetriveUserTradesByCurrencyAndTime(symbol, kind, sorting string, count int64, includeOld bool, startTime, endTime time.Time) (*UserTradesData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	input := &struct {
		Currency   string `json:"currency"`
		Kind       string `json:"kind,omitempty"`
		StartTime  int64  `json:"start_time,omitempty"`
		EndTime    int64  `json:"end_time,omitempty"`
		Sorting    string `json:"sorting,omitempty"`
		Count      int64  `json:"count,omitempty"`
		IncludeOld bool   `json:"include_old,omitempty"`
	}{
		Currency:   symbol,
		Kind:       kind,
		Sorting:    sorting,
		Count:      count,
		IncludeOld: includeOld,
	}
	if !startTime.IsZero() {
		input.StartTime = startTime.UnixMilli()
	}
	if !endTime.IsZero() {
		input.EndTime = endTime.UnixMilli()
	}
	var resp *UserTradesData
	return resp, d.SendWSRequest(getUserTradesByCurrency, input, &resp, true)
}

// WSRetriveUserTradesByInstrument retrives user trades sorted by instrument through the websocket connection.
func (d *Deribit) WSRetriveUserTradesByInstrument(instrument, sorting string, startSeq, endSeq, count int64, includeOld bool) (*UserTradesData, error) {
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
	return resp, d.SendWSRequest(getUserTradesByInstrument, input, &resp, true)
}

// WSRetriveUserTradesByInstrumentAndTime retrives user trades sorted by instrument and time through the websocket connection.
func (d *Deribit) WSRetriveUserTradesByInstrumentAndTime(instrument, sorting string, count int64, includeOld bool, startTime, endTime time.Time) (*UserTradesData, error) {
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
	return resp, d.SendWSRequest(getUserTradesByInstrumentAndTime, input, &resp, true)
}

// WSRetriveUserTradesByOrder retrives user trades fetched by orderID through the web socket connection.
func (d *Deribit) WSRetriveUserTradesByOrder(orderID, sorting string) (*UserTradesData, error) {
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
	return resp, d.SendWSRequest(getUserTradesByOrder, input, &resp, true)
}

// WSResetMMP sends a request to reset MMP for a currency provided through the websocket connection.
func (d *Deribit) WSResetMMP(symbol string) error {
	if symbol == "" {
		return fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	var resp string
	err := d.SendWSRequest(resetMMP, map[string]string{"currency": symbol}, &resp, true)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return fmt.Errorf("mmp could not be reset for %v", symbol)
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
	err := d.SendWSRequest(sendRFQ, input, &resp, true)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return fmt.Errorf("rfq couldn't send for %v", instrumentName)
	}
	return nil
}

// WSSetMMPConfig sends a request to set the given parameter values to the mmp config for the provided currency through the websocket connection.
func (d *Deribit) WSSetMMPConfig(symbol string, interval, frozenTime int64, quantityLimit, deltaLimit float64) error {
	if symbol == "" {
		return fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	var resp string
	err := d.SendWSRequest(resetMMP, map[string]string{"currency": symbol}, &resp, true)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return fmt.Errorf("mmp data could not be set for %v", symbol)
	}
	return nil
}

// WSRetriveSettlementHistoryByInstrument sends a request to fetch settlement history data sorted by instrument through the websocket connection.
func (d *Deribit) WSRetriveSettlementHistoryByInstrument(instrument, settlementType, continuation string, count int64, searchStartTimeStamp time.Time) (*PrivateSettlementsHistoryData, error) {
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
	return resp, d.SendWSRequest(getSettlementHistoryByInstrument, input, &resp, true)
}

// WSRetriveSettlementHistoryByCurency sends a request to fetch settlement history data sorted by currency through the websocket connection.
func (d *Deribit) WSRetriveSettlementHistoryByCurency(symbol, settlementType, continuation string, count int64, searchStartTimeStamp time.Time) (*PrivateSettlementsHistoryData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	input := &struct {
		Currency             string `json:"currency"`
		SettlementType       string `json:"settlement_type,omitempty"`
		Continuation         string `json:"continuation,omitempty"`
		Count                int64  `json:"count,omitempty"`
		SearchStartTimestamp int64  `json:"search_start_timestamp,omitempty"`
	}{
		Currency:       symbol,
		SettlementType: settlementType,
		Continuation:   continuation,
		Count:          count,
	}
	if !searchStartTimeStamp.IsZero() {
		input.SearchStartTimestamp = searchStartTimeStamp.UnixMilli()
	}
	var resp *PrivateSettlementsHistoryData
	return resp, d.SendWSRequest(getSettlementHistoryByCurrency, input, &resp, true)
}

// WSRetriveComboIDS Retrieves available combos.
// This method can be used to get the list of all combos, or only the list of combos in the given state.
func (d *Deribit) WSRetriveComboIDS(symbol, state string) ([]string, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	input := &struct {
		Currency string `json:"currency"`
		State    string `json:"state,omitempty"`
	}{
		Currency: symbol,
		State:    state,
	}
	var resp []string
	return resp, d.SendWSRequest(getComboIDS, input, &resp, false)
}

// WSRetriveComboDetails retrieves information about a combo through the websocket connection.
func (d *Deribit) WSRetriveComboDetails(comboID string) (*ComboDetail, error) {
	if comboID == "" {
		return nil, errInvalidComboID
	}
	var resp *ComboDetail
	return resp, d.SendWSRequest(getComboDetails, map[string]string{"combo_id": comboID}, &resp, false)
}

// WSRetriveCombos retrieves information about active combos through the websocket connection.
func (d *Deribit) WSRetriveCombos(symbol string) ([]ComboDetail, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	var resp []ComboDetail
	return resp, d.SendWSRequest(getCombos, map[string]string{"currency": symbol}, &resp, false)
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
			return nil, errors.New("invalid direction, only 'buy' or 'sell' are supported")
		}
		if args[x].Amount <= 0 {
			return nil, errInvalidAmount
		}
	}
	var resp *ComboDetail
	return resp, d.SendWSRequest(createCombos, map[string]interface{}{"trades": args}, &resp, true)
}

// ------------------------------------------------------------------------------------------------

// WSExecuteBlockTrade executes a block trade request
// The whole request have to be exact the same as in private/verify_block_trade, only role field should be set appropriately - it basically means that both sides have to agree on the same timestamp, nonce, trades fields and server will assure that role field is different between sides (each party accepted own role).
// Using the same timestamp and nonce by both sides in private/verify_block_trade assures that even if unintentionally both sides execute given block trade with valid counterparty_signature, the given block trade will be executed only once
func (d *Deribit) WSExecuteBlockTrade(timestampMS time.Time, nonce, role, symbol string, trades []BlockTradeParam) ([]BlockTradeResponse, error) {
	if nonce == "" {
		return nil, errMissingNonce
	}
	if role != "maker" && role != "taker" {
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
			return nil, errors.New("invalid direction, only 'buy' or 'sell' are supported")
		}
		if trades[x].Amount <= 0 {
			return nil, errInvalidAmount
		}
		if trades[x].Price < 0 {
			return nil, fmt.Errorf("%w, trade price can't be negative", errInvalidPrice)
		}
	}
	if timestampMS.IsZero() {
		return nil, errors.New("zero timestamp")
	}
	signature, err := d.WSVerifyBlockTrade(timestampMS, nonce, role, symbol, trades)
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
		Currency:              symbol,
	}
	var resp []BlockTradeResponse
	return resp, d.SendWSRequest(executeBlockTrades, input, &resp, true)
}

// WSVerifyBlockTrade verifies and creates block trade signature through the websocket connection.
func (d *Deribit) WSVerifyBlockTrade(timestampMS time.Time, nonce, role, symbol string, trades []BlockTradeParam) (string, error) {
	if nonce == "" {
		return "", errMissingNonce
	}
	if role != "maker" && role != "taker" {
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
			return "", errors.New("invalid direction, only 'buy' or 'sell' are supported")
		}
		if trades[x].Amount <= 0 {
			return "", errInvalidAmount
		}
		if trades[x].Price < 0 {
			return "", fmt.Errorf("%w, trade price can't be negative", errInvalidPrice)
		}
	}
	if timestampMS.IsZero() {
		return "", errors.New("zero timestamp")
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
		Currency:  symbol,
	}
	resp := &struct {
		Signature string `json:"signature"`
	}{}
	return resp.Signature, d.SendWSRequest(verifyBlockTrades, input, &resp, true)
}

// WSRetriveUserBlocTrade returns information about users block trade through the websocket connection.
func (d *Deribit) WSRetriveUserBlocTrade(id string) ([]BlockTradeData, error) {
	if id == "" {
		return nil, errors.New("missing block trade id")
	}
	var resp []BlockTradeData
	return resp, d.SendWSRequest(getBlockTrades, map[string]string{"id": id}, &resp, true)
}

// WSRetriveLastBlockTradesByCurrency returns list of last users block trades through the websocket connection.
func (d *Deribit) WSRetriveLastBlockTradesByCurrency(symbol, startID, endID string, count int64) ([]BlockTradeData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	input := &struct {
		Currency string `json:"currency"`
		StartID  string `json:"start_id,omitempty"`
		EndID    string `json:"end_id,omitempty"`
		Count    int64  `json:"count,omitempty"`
	}{
		Currency: symbol,
		StartID:  startID,
		EndID:    endID,
		Count:    count,
	}
	var resp []BlockTradeData
	return resp, d.SendWSRequest(getLastBlockTradesByCurrency, input, &resp, true)
}

// WSMovePositions moves positions from source subaccount to target subaccount through the websocket connection.
func (d *Deribit) WSMovePositions(symbol string, sourceSubAccountUID, targetSubAccountUID int64, trades []BlockTradeParam) ([]BlockTradeMoveResponse, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w \"%s\"", errInvalidCurrency, symbol)
	}
	if sourceSubAccountUID == 0 {
		return nil, errors.New("missing source subaccount id")
	}
	if targetSubAccountUID == 0 {
		return nil, errors.New("missing target subaccount id")
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
		Currency  string            `json:"currency"`
		Trades    []BlockTradeParam `json:"trades"`
		TargetUID int64             `json:"target_uid"`
		SourceUID int64             `json:"source_uid"`
	}{
		Currency:  symbol,
		Trades:    trades,
		TargetUID: targetSubAccountUID,
		SourceUID: sourceSubAccountUID,
	}
	var resp []BlockTradeMoveResponse
	return resp, d.SendWSRequest(movePositions, input, &resp, true)
}

// SendWSRequest sends a request through the websocket connection.
// both authenticated and public endpoints are allowed.
func (d *Deribit) SendWSRequest(method string, params, response interface{}, authenticated bool) error {
	respVal := reflect.ValueOf(response)
	if respVal.Kind() != reflect.Pointer {
		return errInvalidResponseReceiver
	}
	if authenticated && !d.Websocket.CanUseAuthenticatedEndpoints() {
		return errWebsocketConnectionNotAuthenticated
	}
	input := &WsRequest{
		JSONRPCVersion: rpcVersion,
		ID:             d.Websocket.Conn.GenerateMessageID(true),
		Method:         method,
		Params:         params,
	}
	result, err := d.Websocket.Conn.SendMessageReturnResponse(input.ID, input)
	if err != nil {
		return err
	}
	println(string(result))
	resp := &wsResponse{Result: response}
	err = json.Unmarshal(result, resp)
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
