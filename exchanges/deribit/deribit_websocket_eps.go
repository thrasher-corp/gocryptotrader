package deribit

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// WSRetriveBookBySummary retrives book summary data for currency requested through websocket connection.
func (d *Deribit) WSRetriveBookBySummary(currency, kind string) ([]BookSummaryData, error) {
	currency = strings.ToUpper(currency)
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidCurrency
	}
	input := &struct {
		Currency string `json:"currency"`
		Kind     string `json:"kind,omitempty"`
	}{
		Currency: currency,
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
	var resp ContractSizeData
	return &resp, d.SendWSRequest(getContractSize, input, &resp, false)
}

// WSRetriveCurrencies retrives all cryptocurrencies supported by the API through the websocket connection.
func (d *Deribit) WSRetriveCurrencies() ([]CurrencyData, error) {
	var resp []CurrencyData
	return resp, d.SendWSRequest(getCurrencies, nil, &resp, false)
}

// WSRetriveDeliveryPrices retrives delivery prices using index name through the websocket connection.
func (d *Deribit) WSRetriveDeliveryPrices(indexName string, offset, count int64) (*IndexDeliveryPrice, error) {
	indexNames := map[string]bool{"ada_usd": true, "avax_usd": true, "btc_usd": true, "eth_usd": true, "dot_usd": true, "luna_usd": true, "matic_usd": true, "sol_usd": true, "usdc_usd": true, "xrp_usd": true, "ada_usdc": true, "avax_usdc": true, "btc_usdc": true, "eth_usdc": true, "dot_usdc": true, "luna_usdc": true, "matic_usdc": true, "sol_usdc": true, "xrp_usdc": true, "btcdvol_usdc": true, "ethdvol_usdc": true}
	if !indexNames[indexName] {
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
	var resp IndexDeliveryPrice
	return &resp, d.SendWSRequest(getDeliveryPrices, input, &resp, false)
}

// WSRetriveFundingChartData retrives funding chart data for the requested instrument and time length through the websocket connection.
// supported lengths: 8h, 24h, 1m <-(1month)
func (d *Deribit) WSRetriveFundingChartData(instrument, length string) (*FundingChartData, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	if length != "8h" && length != "24h" && length != "12h" && length != "1m" {
		return nil, fmt.Errorf("%w, only 8h, 12h, 1m, and 24h are supported", errIntervalNotSupported)
	}
	input := &struct {
		InstrumentName string `json:"instrument_name"`
		Length         string `json:"length"`
	}{
		InstrumentName: instrument,
		Length:         length,
	}
	var resp FundingChartData
	return &resp, d.SendWSRequest(getFundingChartData, input, &resp, false)
}

// WSRetriveFundingRateValue retrives funding rate value data through the websocket connection.
func (d *Deribit) WSRetriveFundingRateValue(instrument string, startTime, endTime time.Time) (float64, error) {
	if instrument == "" {
		return 0, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	if startTime.IsZero() {
		return 0, fmt.Errorf("%w, start time can not be zero", errInvalidTimestamp)
	} else if startTime.After(endTime) {
		return 0, errStartTimeCannotBeAfterEndTime
	}
	if endTime.IsZero() {
		endTime = time.Now()
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
func (d *Deribit) WSRetriveHistoricalVolatility(currency string) ([]HistoricalVolatilityData, error) {
	currency = strings.ToUpper(currency)
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidCurrency
	}
	input := &struct {
		Currency string `json:"currency"`
	}{
		Currency: currency,
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
func (d *Deribit) WSRetriveCurrencyIndexPrice(currency string) (map[string]float64, error) {
	currency = strings.ToUpper(currency)
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidIndexPriceCurrency
	}
	input := &struct {
		Currency string `json:"currency"`
	}{
		Currency: currency,
	}
	var resp map[string]float64
	return resp, d.SendWSRequest(getCurrencyIndexPrice, input, &resp, false)
}

// WSRetriveIndexPrice retrives price data for the requested index through the websocket connection.
func (d *Deribit) WSRetriveIndexPrice(index string) (*IndexPriceData, error) {
	indexNames := map[string]bool{"ada_usd": true, "avax_usd": true, "btc_usd": true, "eth_usd": true, "dot_usd": true, "luna_usd": true, "matic_usd": true, "sol_usd": true, "usdc_usd": true, "xrp_usd": true, "ada_usdc": true, "avax_usdc": true, "btc_usdc": true, "eth_usdc": true, "dot_usdc": true, "luna_usdc": true, "matic_usdc": true, "sol_usdc": true, "xrp_usdc": true, "btcdvol_usdc": true, "ethdvol_usdc": true}
	if !indexNames[index] {
		return nil, errUnsupportedIndexName
	}
	input := &struct {
		IndexName string `json:"index_name"`
	}{
		IndexName: index,
	}
	var resp IndexPriceData
	return &resp, d.SendWSRequest(getIndexPrice, input, &resp, false)
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
	var resp InstrumentData
	return &resp, d.SendWSRequest(getInstrument, input, &resp, false)
}

// WSRetriveInstrumentsData gets data for all available instruments
func (d *Deribit) WSRetriveInstrumentsData(currency, kind string, expired bool) ([]InstrumentData, error) {
	currency = strings.ToUpper(currency)
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidIndexPriceCurrency
	}
	input := &struct {
		Currency string `json:"currency"`
		Expired  bool   `json:"expired"`
		Kind     string `json:"kind,omitempty"`
	}{
		Currency: currency,
		Expired:  expired,
		Kind:     kind,
	}
	var resp []InstrumentData
	return resp, d.SendWSRequest(getInstruments, input, &resp, false)
}

// WSRetriveLastSettlementsByCurrency retrives last settlement data by currency through the websocket connection.
func (d *Deribit) WSRetriveLastSettlementsByCurrency(currency, settlementType, continuation string, count int64, startTime time.Time) (*SettlementsData, error) {
	currency = strings.ToUpper(currency)
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidIndexPriceCurrency
	}
	input := &struct {
		Currency             string `json:"currency"`
		Type                 string `json:"type,omitempty"`
		Continuation         string `json:"continuation"`
		Count                int64  `json:"count,omitempty"`
		SearchStartTimestamp int64  `json:"search_start_timestamp,omitempty"`
	}{
		Currency:             currency,
		Type:                 settlementType,
		Continuation:         continuation,
		Count:                count,
		SearchStartTimestamp: startTime.UnixMilli(),
	}
	if !startTime.IsZero() {
		input.SearchStartTimestamp = startTime.UnixMilli()
	}
	var resp SettlementsData
	return &resp, d.SendWSRequest(getLastSettlementsByCurrency, input, resp, false)
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
	var resp SettlementsData
	return &resp, d.SendWSRequest(getLastSettlementsByInstrument, input, &resp, false)
}

// WSRetriveLastTradesByCurrency retrives last trades for requested currency through the websocket connection.
func (d *Deribit) WSRetriveLastTradesByCurrency(currency, kind, startID, endID, sorting string, count int64, includeOld bool) (*PublicTradesData, error) {
	currency = strings.ToUpper(currency)
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidIndexPriceCurrency
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
		Currency:   currency,
		Kind:       kind,
		StartID:    startID,
		EndID:      endID,
		Count:      count,
		IncludeOld: includeOld,
		Sorting:    sorting,
	}
	var resp PublicTradesData
	return &resp, d.SendWSRequest(getLastTradesByCurrency, input, &resp, false)
}

// SendWSRequest sends a request through the websocket connection.
// both authenticated and public endpoints are allowed.
func (d *Deribit) SendWSRequest(method string, params interface{}, response interface{}, authenticated bool) error {
	respVal := reflect.ValueOf(response)
	if respVal.Kind() != reflect.Pointer || respVal.IsNil() {
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
	resp := &wsResponse{Result: response}
	err = json.Unmarshal(result, resp)
	if err != nil {
		return err
	}
	if resp.Error.Code != 0 || resp.Error.Message != "" {
		return fmt.Errorf("Code: %d Message: %s; %s: %s", resp.Error.Code, resp.Error.Message, resp.Error.Data.Param, resp.Error.Data.Reason)
	}
	return nil
}
