package deribit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/thrasher-corp/gocryptotrader/common"
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
