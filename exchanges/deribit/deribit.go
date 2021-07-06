package deribit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Deribit is the overarching type across this package
type Deribit struct {
	exchange.Base
}

const (
	deribitAPIURL     = "https://www.deribit.com"
	deribitAPIVersion = "/api/v2"

	// Public endpoints
	getBookByCurrency                = "/public/get_book_summary_by_currency"
	getBookByInstrument              = "/public/get_book_summary_by_instrument"
	getContractSize                  = "/public/get_contract_size"
	getCurrencies                    = "/public/get_currencies"
	getFundingChartData              = "/public/get_funding_chart_data"
	getFundingRateHistory            = "/public/get_funding_rate_history"
	getFundingRateValue              = "/public/get_funding_rate_value"
	getHistoricalVolatility          = "/public/get_historical_volatility"
	getIndexPrice                    = "/public/get_index_price"
	getIndexPriceNames               = "/public/get_index_price_names"
	getInstrument                    = "/public/get_instrument"
	getInstruments                   = "/public/get_instruments"
	getLastSettlementsByCurrency     = "/public/get_last_settlements_by_currency"
	getLastSettlementsByInstrument   = "/public/get_last_settlements_by_instrument"
	getLastTradesByCurrency          = "/public/get_last_trades_by_currency"
	getLastTradesByCurrencyAndTime   = "/public/get_last_trades_by_currency_and_time"
	getLastTradesByInstrument        = "/public/get_last_trades_by_instrument"
	getLastTradesByInstrumentAndTime = "/public/get_last_trades_by_instrument_and_time"
	getMarkPriceHistory              = "/public/get_mark_price_history"
	getOrderbook                     = "/public/get_order_book"
	getTradeVolumes                  = "/public/get_trade_volumes"
	getTradingViewChartData          = "/public/get_tradingview_chart_data"
	getVolatilityIndexData           = "/public/get_volatility_index_data"
	getTicker                        = "/public/ticker"
	getAnnouncements                 = "/public/get_announcements"

	// Authenticated endpoints

	// wallet eps
	cancelTransferByID         = "/private/cancel_transfer_by_id"
	cancelWithdrawal           = "/private/cancel_withdrawal"
	createDepositAddress       = "/private/create_deposit_address"
	getCurrentDepositAddress   = "/private/get_current_deposit_address"
	getDeposits                = "/private/get_deposits"
	getTransfers               = "/private/get_transfers"
	getWithdrawals             = "/private/get_withdrawals"
	submitTransferToSubaccount = "/private/submit_transfer_to_subaccount"
	submitTransferToUser       = "/private/submit_transfer_to_user"
	submitWithdraw             = "/private/withdraw"

	// trading eps
	submitBuy                        = "/private/buy"
	submitSell                       = "/private/sell"
	submitEdit                       = "/private/edit"
	editByLabel                      = "/private/edit_by_label"
	submitCancel                     = "/private/cancel"
	submitCancelAll                  = "/private/cancel_all"
	submitCancelAllByCurrency        = "/private/cancel_all_by_currency"
	submitCancelAllByInstrument      = "/private/cancel_all_by_instrument"
	submitCancelByLabel              = "/private/cancel_by_label"
	submitClosePosition              = "/private/close_position"
	getMargins                       = "/private/get_margins"
	getMMPConfig                     = "/private/get_mmp_config"
	getOpenOrdersByCurrency          = "/private/get_open_orders_by_currency"
	getOpenOrdersByInstrument        = "/private/get_open_orders_by_instrument"
	getOrderHistoryByCurrency        = "/private/get_order_history_by_currency"
	getOrderHistoryByInstrument      = "/private/get_order_history_by_instrument"
	getOrderMarginByIDs              = "/private/get_order_margin_by_ids"
	getOrderState                    = "/private/get_order_state"
	getTriggerOrderHistory           = "/private/get_trigger_order_history"
	getUserTradesByCurrency          = "/private/get_user_trades_by_currency"
	getUserTradesByCurrencyAndTime   = "/private/get_user_trades_by_currency_and_time"
	getUserTradesByInstrument        = "/private/get_user_trades_by_instrument"
	getUserTradesByInstrumentAndTime = "/private/get_user_trades_by_instrument_and_time"
	getUserTradesByOrder             = "/private/get_user_trades_by_order"
	resetMMP                         = "/private/reset_mmp"
	setMMPConfig                     = "/private/set_mmp_config"
	getSettlementHistoryByInstrument = "/private/get_settlement_history_by_instrument"
	getSettlementHistoryByCurrency   = "/private/get_settlement_history_by_currency"

	// account management eps
	changeAPIKeyName                  = "/private/change_api_key_name"
	changeScopeInAPIKey               = "/private/change_scope_in_api_key"
	changeSubAccountName              = "/private/change_subaccount_name"
	createAPIKey                      = "/private/create_api_key"
	createSubAccount                  = "/private/create_subaccount"
	disableAPIKey                     = "/private/disable_api_key"
	disableTFAForSubaccount           = "/private/disable_tfa_for_subaccount"
	enableAffiliateProgram            = "/private/enable_affiliate_program"
	enableAPIKey                      = "/private/enable_api_key"
	getAccountSummary                 = "/private/get_account_summary"
	getAffiliateProgramInfo           = "/private/get_affiliate_program_info"
	getEmailLanguage                  = "/private/get_email_language"
	getNewAnnouncements               = "/private/get_new_announcements"
	getPosition                       = "/private/get_position"
	getPositions                      = "/private/get_positions"
	getSubAccounts                    = "/private/get_subaccounts"
	getTransactionLog                 = "/private/get_transaction_log"
	listAPIKeys                       = "/private/list_api_keys"
	removeAPIKey                      = "/private/remove_api_key"
	removeSubAccount                  = "/private/remove_subaccount"
	resetAPIKey                       = "/private/reset_api_key"
	setAnnouncementAsRead             = "/private/set_announcement_as_read"
	setAPIKeyAsDefault                = "/private/set_api_key_as_default"
	setEmailForSubAccount             = "/private/set_email_for_subaccount"
	setEmailLanguage                  = "/private/set_email_language"
	setPasswordForSubAccount          = "/private/set_password_for_subaccount"
	toggleNotificationsFromSubAccount = "/private/toggle_notifications_from_subaccount"
	toggleSubAccountLogin             = "/private/toggle_subaccount_login"
)

// Start implementing public and private exchange API funcs below

// GetBookSummaryByCurrency gets book summary data for currency requested
func (d *Deribit) GetBookSummaryByCurrency(currency, kind string) ([]BookSummaryData, error) {
	var resp []BookSummaryData
	params := url.Values{}
	params.Set("currency", currency)
	if kind != "" {
		params.Set("kind", kind)
	}
	return resp, d.SendHTTPRequest(exchange.RestSpot,
		common.EncodeURLValues(getBookByCurrency, params), &resp)
}

// GetBookSummaryByInstrument gets book summary data for instrument requested
func (d *Deribit) GetBookSummaryByInstrument(instrument string) ([]BookSummaryData, error) {
	var resp []BookSummaryData
	params := url.Values{}
	params.Set("instrument_name", instrument)
	return resp, d.SendHTTPRequest(exchange.RestSpot,
		common.EncodeURLValues(getBookByInstrument, params), &resp)
}

// GetContractSize gets contract size for instrument requested
func (d *Deribit) GetContractSize(instrument string) (ContractSizeData, error) {
	var resp ContractSizeData
	params := url.Values{}
	params.Set("instrument_name", instrument)
	return resp, d.SendHTTPRequest(exchange.RestSpot,
		common.EncodeURLValues(getContractSize, params), &resp)
}

// GetCurrencies gets all cryptocurrencies supported by the API
func (d *Deribit) GetCurrencies() ([]CurrencyData, error) {
	var resp []CurrencyData
	return resp, d.SendHTTPRequest(exchange.RestSpot, getCurrencies, &resp)
}

// GetFundingChartData gets funding chart data for the requested instrument and timelength
// supported lengths: 8h, 24h, 1m <-(1month)
func (d *Deribit) GetFundingChartData(instrument, length string) (FundingChartData, error) {
	var resp FundingChartData
	params := url.Values{}
	params.Set("instrument_name", instrument)
	params.Set("length", length)
	return resp, d.SendHTTPRequest(exchange.RestSpot,
		common.EncodeURLValues(getFundingChartData, params), &resp)
}

// GetFundingRateValue gets funding rate value data
func (d *Deribit) GetFundingRateValue(instrument string, startTime, endTime time.Time) (float64, error) {
	var resp float64
	params := url.Values{}
	params.Set("instrument_name", instrument)
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errStartTimeCannotBeAfterEndTime
		}
		params.Set("start_timestamp", strconv.FormatInt(startTime.Unix()*1000, 10))
		params.Set("end_timestamp", strconv.FormatInt(endTime.Unix()*1000, 10))
	}
	return resp, d.SendHTTPRequest(exchange.RestSpot,
		common.EncodeURLValues(getFundingRateValue, params), &resp)
}

// GetHistoricalVolatility gets historical volatility data
func (d *Deribit) GetHistoricalVolatility(currency string) ([]HistoricalVolatilityData, error) {
	var data [][2]interface{}
	params := url.Values{}
	params.Set("currency", currency)
	err := d.SendHTTPRequest(exchange.RestSpot,
		common.EncodeURLValues(getHistoricalVolatility, params), &data)
	if err != nil {
		return nil, err
	}
	var resp []HistoricalVolatilityData
	for x := range data {
		timeData, ok := data[x][0].(float64)
		if !ok {
			fmt.Println(data[x][0])
			return resp, fmt.Errorf("%v GetHistoricalVolatility: %w for time", d.Name, errTypeAssert)
		}
		val, ok := data[x][1].(float64)
		if !ok {
			return resp, fmt.Errorf("%v GetHistoricalVolatility: %w for val", d.Name, errTypeAssert)
		}
		resp = append(resp, HistoricalVolatilityData{
			Timestamp: timeData,
			Value:     val,
		})
	}
	return resp, nil
}

// GetIndexPrice gets price data for the requested index
func (d *Deribit) GetIndexPrice(index string) (IndexPriceData, error) {
	var resp IndexPriceData
	params := url.Values{}
	params.Set("index_name", index)
	return resp, d.SendHTTPRequest(exchange.RestSpot,
		common.EncodeURLValues(getIndexPrice, params), &resp)
}

// GetIndexPriceNames gets names of indexes
func (d *Deribit) GetIndexPriceNames() ([]string, error) {
	var resp []string
	return resp, d.SendHTTPRequest(exchange.RestSpot, getIndexPriceNames, &resp)
}

// GetInstrumentData gets data for a requested instrument
func (d *Deribit) GetInstrumentData(instrument string) (InstrumentData, error) {
	var resp InstrumentData
	params := url.Values{}
	params.Set("instrument_name", instrument)
	return resp, d.SendHTTPRequest(exchange.RestSpot,
		common.EncodeURLValues(getInstrument, params), &resp)
}

// GetInstrumentsData gets data for all available instruments
func (d *Deribit) GetInstrumentsData(currency, kind string, expired bool) ([]InstrumentData, error) {
	var resp []InstrumentData
	params := url.Values{}
	params.Set("currency", currency)
	if kind != "" {
		params.Set("kind", kind)
	}
	expiredString := "false"
	if expired {
		expiredString = "true"
	}
	params.Set("expired", expiredString)
	return resp, d.SendHTTPRequest(exchange.RestSpot,
		common.EncodeURLValues(getInstruments, params), &resp)
}

// GetLastSettlementsByCurrency gets last settlement data by currency
func (d *Deribit) GetLastSettlementsByCurrency(currency, settlementType, continuation string, count int64, startTime time.Time) (SettlementsData, error) {
	var resp SettlementsData
	params := url.Values{}
	params.Set("currency", currency)
	if settlementType != "" {
		params.Set("type", settlementType)
	}
	if continuation != "" {
		params.Set("continuation", continuation)
	}
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	if !startTime.IsZero() {
		params.Set("search_start_timestamp", strconv.FormatInt(startTime.Unix()*1000, 10))
	}
	return resp, d.SendHTTPRequest(exchange.RestSpot,
		common.EncodeURLValues(getLastSettlementsByCurrency, params), &resp)
}

// GetLastSettlementsByInstrument gets last settlement data for requested instrument
func (d *Deribit) GetLastSettlementsByInstrument(instrument, settlementType, continuation string, count int64, startTime time.Time) (SettlementsData, error) {
	var resp SettlementsData
	params := url.Values{}
	params.Set("instrument_name", instrument)
	if settlementType != "" {
		params.Set("type", settlementType)
	}
	if continuation != "" {
		params.Set("continuation", continuation)
	}
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	if !startTime.IsZero() {
		params.Set("search_start_timestamp", strconv.FormatInt(startTime.Unix()*1000, 10))
	}
	return resp, d.SendHTTPRequest(exchange.RestSpot,
		common.EncodeURLValues(getLastSettlementsByInstrument, params), &resp)
}

// GetLastTradesByCurrency gets last trades for requested currency
func (d *Deribit) GetLastTradesByCurrency(currency, kind, startID, endID, sorting string, count int64, includeOld bool) (PublicTradesData, error) {
	var resp PublicTradesData
	params := url.Values{}
	params.Set("currency", currency)
	if kind != "" {
		params.Set("kind", kind)
	}
	if startID != "" {
		params.Set("start_id", startID)
	}
	if endID != "" {
		params.Set("end_id", endID)
	}
	if sorting != "" {
		params.Set("sorting", sorting)
	}
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	includeOldString := "false"
	if includeOld {
		includeOldString = "true"
	}
	params.Set("include_old", includeOldString)
	return resp, d.SendHTTPRequest(exchange.RestSpot,
		common.EncodeURLValues(getLastTradesByCurrency, params), &resp)
}

// GetLastTradesByCurrencyAndTime gets last trades for requested currency and time intervals
func (d *Deribit) GetLastTradesByCurrencyAndTime(currency, kind, sorting string, count int64, includeOld bool, startTime, endTime time.Time) (PublicTradesData, error) {
	var resp PublicTradesData
	params := url.Values{}
	params.Set("currency", currency)
	if kind != "" {
		params.Set("kind", kind)
	}
	if sorting != "" {
		params.Set("sorting", sorting)
	}
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errStartTimeCannotBeAfterEndTime
		}
		params.Set("start_timestamp", strconv.FormatInt(startTime.Unix()*1000, 10))
		params.Set("end_timestamp", strconv.FormatInt(endTime.Unix()*1000, 10))
	}
	return resp, d.SendHTTPRequest(exchange.RestSpot,
		common.EncodeURLValues(getLastTradesByCurrencyAndTime, params), &resp)
}

// GetLastTradesByInstrument gets last trades for requested instrument requested
func (d *Deribit) GetLastTradesByInstrument(currency, startSeq, endSeq, sorting string, count int64, includeOld bool) (PublicTradesData, error) {
	var resp PublicTradesData
	params := url.Values{}
	params.Set("instrument_name", currency)
	if startSeq != "" {
		params.Set("start_seq", startSeq)
	}
	if endSeq != "" {
		params.Set("end_seq", endSeq)
	}
	if sorting != "" {
		params.Set("sorting", sorting)
	}
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	includeOldString := "false"
	if includeOld {
		includeOldString = "true"
	}
	params.Set("include_old", includeOldString)
	return resp, d.SendHTTPRequest(exchange.RestSpot,
		common.EncodeURLValues(getLastTradesByInstrument, params), &resp)
}

// GetLastTradesByInstrumentAndTime gets last trades for requested instrument requested and time intervals
func (d *Deribit) GetLastTradesByInstrumentAndTime(instrument, sorting string, count int64, includeOld bool, startTime, endTime time.Time) (PublicTradesData, error) {
	var resp PublicTradesData
	params := url.Values{}
	params.Set("instrument_name", instrument)
	if sorting != "" {
		params.Set("sorting", sorting)
	}
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errStartTimeCannotBeAfterEndTime
		}
		params.Set("start_timestamp", strconv.FormatInt(startTime.Unix()*1000, 10))
		params.Set("end_timestamp", strconv.FormatInt(endTime.Unix()*1000, 10))
	}
	return resp, d.SendHTTPRequest(exchange.RestSpot,
		common.EncodeURLValues(getLastTradesByInstrumentAndTime, params), &resp)
}

// GetMarkPriceHistory gets data for mark price history
func (d *Deribit) GetMarkPriceHistory(instrument string, startTime, endTime time.Time) ([]MarkPriceHistory, error) {
	var resp []MarkPriceHistory
	params := url.Values{}
	params.Set("instrument_name", instrument)
	if startTime.After(endTime) {
		return resp, errStartTimeCannotBeAfterEndTime
	}
	params.Set("start_timestamp", strconv.FormatInt(startTime.Unix()*1000, 10))
	params.Set("end_timestamp", strconv.FormatInt(endTime.Unix()*1000, 10))
	return resp, d.SendHTTPRequest(exchange.RestSpot,
		common.EncodeURLValues(getMarkPriceHistory, params), &resp)
}

// GetOrderbookData gets data orderbook of requested instrument
func (d *Deribit) GetOrderbookData(instrument string, depth int64) (OBData, error) {
	var resp OBData
	params := url.Values{}
	params.Set("instrument_name", instrument)
	if depth != 0 {
		params.Set("depth", strconv.FormatInt(depth, 10))
	}
	return resp, d.SendHTTPRequest(exchange.RestSpot,
		common.EncodeURLValues(getOrderbook, params), &resp)
}

// GetTradeVolumes gets trade volumes' data of all instruments
func (d *Deribit) GetTradeVolumes(extended bool) ([]TradeVolumesData, error) {
	var resp []TradeVolumesData
	params := url.Values{}
	extendedStr := "false"
	if extended {
		extendedStr = "true"
	}
	params.Set("extended", extendedStr)
	return resp, d.SendHTTPRequest(exchange.RestSpot,
		common.EncodeURLValues(getTradeVolumes, params), &resp)
}

// GetTradingViewChartData gets volatility index data for the requested instrument
func (d *Deribit) GetTradingViewChartData(instrument, resolution string, startTime, endTime time.Time) (TVChartData, error) {
	var resp TVChartData
	params := url.Values{}
	params.Set("instrument_name", instrument)
	if startTime.After(endTime) {
		return resp, errStartTimeCannotBeAfterEndTime
	}
	params.Set("start_timestamp", strconv.FormatInt(startTime.Unix()*1000, 10))
	params.Set("end_timestamp", strconv.FormatInt(endTime.Unix()*1000, 10))
	params.Set("resolution", resolution)
	return resp, d.SendHTTPRequest(exchange.RestSpot,
		common.EncodeURLValues(getTradingViewChartData, params), &resp)
}

// GetVolatilityIndexData gets volatility index data for the requested currency
func (d *Deribit) GetVolatilityIndexData(currency, resolution string, startTime, endTime time.Time) (VolatilityIndexData, error) {
	var resp VolatilityIndexData
	params := url.Values{}
	params.Set("currency", currency)
	if startTime.After(endTime) {
		return resp, errStartTimeCannotBeAfterEndTime
	}
	params.Set("start_timestamp", strconv.FormatInt(startTime.Unix()*1000, 10))
	params.Set("end_timestamp", strconv.FormatInt(endTime.Unix()*1000, 10))
	params.Set("resolution", resolution)
	return resp, d.SendHTTPRequest(exchange.RestSpot,
		common.EncodeURLValues(getVolatilityIndexData, params), &resp)
}

// GetPublicTicker gets public ticker data of the instrument requested
func (d *Deribit) GetPublicTicker(instrument string) (TickerData, error) {
	var resp TickerData
	params := url.Values{}
	params.Set("instrument_name", instrument)
	return resp, d.SendHTTPRequest(exchange.RestSpot,
		common.EncodeURLValues(getTicker, params), &resp)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (d *Deribit) SendHTTPRequest(ep exchange.URL, path string, result interface{}) error {
	endpoint, err := d.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	var data struct {
		JsonRPC string          `json:"jsonrpc"`
		ID      int64           `json:"id"`
		Data    json.RawMessage `json:"result"`
	}
	err = d.SendPayload(context.Background(), &request.Item{
		Method:        http.MethodGet,
		Path:          endpoint + deribitAPIVersion + path,
		Result:        &data,
		Verbose:       d.Verbose,
		HTTPDebugging: d.HTTPDebugging,
		HTTPRecording: d.HTTPRecording,
	})
	if err != nil {
		return err
	}
	return json.Unmarshal(data.Data, result)
}

// GetAccountSummary gets account summary data for the requested instrument
func (d *Deribit) GetAccountSummary(currency string, extended bool) (AccountSummaryData, error) {
	var resp AccountSummaryData
	params := url.Values{}
	params.Set("currency", currency)
	extendedStr := "false"
	if extended {
		extendedStr = "true"
	}
	params.Set("extended", extendedStr)
	return resp, d.SendHTTPAuthRequest(exchange.RestSpot, http.MethodGet,
		common.EncodeURLValues(getAccountSummary, params), nil, &resp)
}

// CancelWithdrawal cancels withdrawal request for a given currency by its id
func (d *Deribit) CancelWithdrawal(currency string, id int64) (CancelWithdrawalData, error) {
	var resp CancelWithdrawalData
	params := url.Values{}
	params.Set("currency", currency)
	params.Set("id", strconv.FormatInt(id, 10))
	return resp, d.SendHTTPAuthRequest(exchange.RestSpot, http.MethodGet,
		cancelWithdrawal, params, &resp)
}

// CancelTransferByID gets volatility index data for the requested instrument
func (d *Deribit) CancelTransferByID(currency, tfa string, id int64) (AccountSummaryData, error) {
	var resp AccountSummaryData
	params := url.Values{}
	params.Set("currency", currency)
	if tfa != "" {
		params.Set("tfa", tfa)
	}
	params.Set("id", strconv.FormatInt(id, 10))
	return resp, d.SendHTTPAuthRequest(exchange.RestSpot, http.MethodGet,
		common.EncodeURLValues(cancelTransferByID, params), nil, &resp)
}

// CreateDepositAddress creates a deposit address for the currency requested
func (d *Deribit) CreateDepositAddress(currency string) (DepositAddressData, error) {
	var resp DepositAddressData
	params := url.Values{}
	params.Set("currency", currency)
	return resp, d.SendHTTPAuthRequest(exchange.RestSpot, http.MethodGet,
		createDepositAddress, params, &resp)
}

// GetCurrentDepositAddress gets the current deposit address for the requested currency
func (d *Deribit) GetCurrentDepositAddress(currency string) (DepositAddressData, error) {
	var resp DepositAddressData
	params := url.Values{}
	params.Set("currency", currency)
	return resp, d.SendHTTPAuthRequest(exchange.RestSpot, http.MethodGet,
		createDepositAddress, params, &resp)
}

// GetDeposits gets the deposits of a given currency
func (d *Deribit) GetDeposits(currency string, count, offset int64) (DepositsData, error) {
	var resp DepositsData
	params := url.Values{}
	params.Set("currency", currency)
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	if offset != 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	return resp, d.SendHTTPAuthRequest(exchange.RestSpot, http.MethodGet,
		getDeposits, params, &resp)
}

// GetTransfers gets transfers data for the requested currency
func (d *Deribit) GetTransfers(currency string, count, offset int64) (TransferData, error) {
	var resp TransferData
	params := url.Values{}
	params.Set("currency", currency)
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	if offset != 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	return resp, d.SendHTTPAuthRequest(exchange.RestSpot, http.MethodGet,
		getTransfers, params, &resp)
}

// GetWithdrawals gets withdrawals data for a requested currency
func (d *Deribit) GetWithdrawals(currency string, count, offset int64) (WithdrawalsData, error) {
	var resp WithdrawalsData
	params := url.Values{}
	params.Set("currency", currency)
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	if offset != 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	return resp, d.SendHTTPAuthRequest(exchange.RestSpot, http.MethodGet,
		getWithdrawals, params, &resp)
}

// SubmitTransferToSubAccount submits a request to transfer a currency to a subaccount
func (d *Deribit) SubmitTransferToSubAccount(currency string, amount float64, destinationID int64) (TransferData, error) {
	var resp TransferData
	params := url.Values{}
	params.Set("currency", currency)
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	return resp, d.SendHTTPAuthRequest(exchange.RestSpot, http.MethodGet,
		submitTransferToSubaccount, params, &resp)
}

// SubmitTransferToSubAccount submits a request to transfer a currency to another user
func (d *Deribit) SubmitTransferToUser(currency, tfa string, amount float64, destinationID int64) (TransferData, error) {
	var resp TransferData
	params := url.Values{}
	params.Set("currency", currency)
	if tfa != "" {
		params.Set("tfa", tfa)
	}
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	return resp, d.SendHTTPAuthRequest(exchange.RestSpot, http.MethodGet,
		submitTransferToSubaccount, params, &resp)
}

// SubmitWithdraw submits a withdrawal request to the exchange for the requested currency
func (d *Deribit) SubmitWithdraw(currency, address, priority, tfa string, amount float64) (WithdrawData, error) {
	var resp WithdrawData
	params := url.Values{}
	params.Set("currency", currency)
	params.Set("address", address)
	if priority != "" {
		params.Set("priority", priority)
	}
	if tfa != "" {
		params.Set("tfa", tfa)
	}
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	return resp, d.SendHTTPAuthRequest(exchange.RestSpot, http.MethodGet,
		submitWithdraw, params, &resp)
}

// SendAuthHTTPRequest sends an authenticated request to deribit api
func (d *Deribit) SendHTTPAuthRequest(ep exchange.URL, method, path string, data url.Values, result interface{}) error {
	kee := "" //key here
	see := "" //secret here
	endpoint, err := d.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	reqDataStr := method + "\n" + deribitAPIVersion + common.EncodeURLValues(path, data) + "\n" + "" + "\n"
	fmt.Printf("REQUEST DATA STRRRRRRRRR: %v\n\n\n", reqDataStr)

	n := d.Requester.GetNonce(true)

	strTS := strconv.FormatInt(time.Now().Unix()*1000, 10)

	str2Sign := fmt.Sprintf("%s\n%s\n%s", strTS,
		n, reqDataStr)

	fmt.Printf("STR 2 SIGN: %v\n\n\n", str2Sign)

	hmac := crypto.GetHMAC(crypto.HashSHA256,
		[]byte(str2Sign),
		[]byte(see))

	headers := make(map[string]string)
	headerString := fmt.Sprintf("deri-hmac-sha256 id=%s,ts=%s,sig=%s,nonce=%s",
		kee,
		strTS,
		crypto.HexEncodeToString(hmac),
		n)
	headers["Authorization"] = headerString
	// headers["id"] = kee
	// headers["ts"] = strTS
	// headers["sig"] = crypto.HexEncodeToString(hmac)
	// headers["nonce"] = n.String()
	headers["Content-Type"] = "application/json"

	var tempData struct {
		JsonRPC string          `json:"jsonrpc"`
		ID      int64           `json:"id"`
		Data    json.RawMessage `json:"result"`
	}

	err = d.SendPayload(context.Background(), &request.Item{
		Method:        method,
		Path:          endpoint + deribitAPIVersion + path,
		Headers:       headers,
		Body:          nil,
		Result:        &tempData,
		AuthRequest:   true,
		Verbose:       d.Verbose,
		HTTPDebugging: d.HTTPDebugging,
		HTTPRecording: d.HTTPRecording,
	})
	if err != nil {
		return err
	}
	return json.Unmarshal(tempData.Data, result)
}
