package deribit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Deribit is the overarching type across this package
type Deribit struct {
	exchange.Base
}

const (
	deribitAPIURL     = "https://www.deribit.com"
	deribitTestAPIURL = "https://test.deribit.com"
	deribitAPIVersion = "/api/v2"

	// Public endpoints

	// Market Data
	getBookByCurrency                = "public/get_book_summary_by_currency"
	getBookByInstrument              = "public/get_book_summary_by_instrument"
	getContractSize                  = "public/get_contract_size"
	getCurrencies                    = "public/get_currencies"
	getDeliveryPrices                = "public/get_delivery_prices"
	getFundingChartData              = "public/get_funding_chart_data"
	getFundingRateHistory            = "public/get_funding_rate_history"
	getFundingRateValue              = "public/get_funding_rate_value"
	getHistoricalVolatility          = "public/get_historical_volatility"
	getCurrencyIndexPrice            = "public/get_index"
	getIndexPrice                    = "public/get_index_price"
	getIndexPriceNames               = "public/get_index_price_names"
	getInstrument                    = "public/get_instrument"
	getInstruments                   = "public/get_instruments"
	getLastSettlementsByCurrency     = "public/get_last_settlements_by_currency"
	getLastSettlementsByInstrument   = "public/get_last_settlements_by_instrument"
	getLastTradesByCurrency          = "public/get_last_trades_by_currency"
	getLastTradesByCurrencyAndTime   = "public/get_last_trades_by_currency_and_time"
	getLastTradesByInstrument        = "public/get_last_trades_by_instrument"
	getLastTradesByInstrumentAndTime = "public/get_last_trades_by_instrument_and_time"
	getMarkPriceHistory              = "public/get_mark_price_history"
	getOrderbook                     = "public/get_order_book"
	getOrderbookByInstrumentID       = "public/get_order_book_by_instrument_id"
	getRFQ                           = "public/get_rfqs"
	getTradeVolumes                  = "public/get_trade_volumes"
	getTradingViewChartData          = "public/get_tradingview_chart_data"
	getVolatilityIndexData           = "public/get_volatility_index_data"
	getTicker                        = "public/ticker"

	// Authenticated endpoints

	// wallet eps
	cancelTransferByID         = "private/cancel_transfer_by_id"
	cancelWithdrawal           = "private/cancel_withdrawal"
	createDepositAddress       = "private/create_deposit_address"
	getCurrentDepositAddress   = "private/get_current_deposit_address"
	getDeposits                = "private/get_deposits"
	getTransfers               = "private/get_transfers"
	getWithdrawals             = "private/get_withdrawals"
	submitTransferToSubaccount = "private/submit_transfer_to_subaccount"
	submitTransferToUser       = "private/submit_transfer_to_user"
	submitWithdraw             = "private/withdraw"

	// trading endpoints
	submitBuy                        = "private/buy"
	submitSell                       = "private/sell"
	submitEdit                       = "private/edit"
	editByLabel                      = "private/edit_by_label"
	submitCancel                     = "private/cancel"
	submitCancelAll                  = "private/cancel_all"
	submitCancelAllByCurrency        = "private/cancel_all_by_currency"
	submitCancelAllByInstrument      = "private/cancel_all_by_instrument"
	submitCancelByLabel              = "private/cancel_by_label"
	submitClosePosition              = "private/close_position"
	getMargins                       = "private/get_margins"
	getMMPConfig                     = "private/get_mmp_config"
	getOpenOrdersByCurrency          = "private/get_open_orders_by_currency"
	getOpenOrdersByInstrument        = "private/get_open_orders_by_instrument"
	getOrderHistoryByCurrency        = "private/get_order_history_by_currency"
	getOrderHistoryByInstrument      = "private/get_order_history_by_instrument"
	getOrderMarginByIDs              = "private/get_order_margin_by_ids"
	getOrderState                    = "private/get_order_state"
	getTriggerOrderHistory           = "private/get_trigger_order_history"
	getUserTradesByCurrency          = "private/get_user_trades_by_currency"
	getUserTradesByCurrencyAndTime   = "private/get_user_trades_by_currency_and_time"
	getUserTradesByInstrument        = "private/get_user_trades_by_instrument"
	getUserTradesByInstrumentAndTime = "private/get_user_trades_by_instrument_and_time"
	getUserTradesByOrder             = "private/get_user_trades_by_order"
	resetMMP                         = "private/reset_mmp"
	sendRFQ                          = "private/send_rfq"
	setMMPConfig                     = "private/set_mmp_config"
	getSettlementHistoryByInstrument = "private/get_settlement_history_by_instrument"
	getSettlementHistoryByCurrency   = "private/get_settlement_history_by_currency"

	// account management eps
	getAnnouncements                  = "public/get_announcements"
	getPublicPortfolioMargins         = "public/get_portfolio_margins"
	changeAPIKeyName                  = "private/change_api_key_name"
	changeScopeInAPIKey               = "private/change_scope_in_api_key"
	changeSubAccountName              = "private/change_subaccount_name"
	createAPIKey                      = "private/create_api_key"
	createSubAccount                  = "private/create_subaccount"
	disableAPIKey                     = "private/disable_api_key"
	disableTFAForSubaccount           = "private/disable_tfa_for_subaccount"
	enableAffiliateProgram            = "private/enable_affiliate_program"
	enableAPIKey                      = "private/enable_api_key"
	getAccessLog                      = "private/get_access_log"
	getAccountSummary                 = "private/get_account_summary"
	getAffiliateProgramInfo           = "private/get_affiliate_program_info"
	getEmailLanguage                  = "private/get_email_language"
	getNewAnnouncements               = "private/get_new_announcements"
	getPrivatePortfolioMargins        = "private/get_portfolio_margins"
	getPosition                       = "private/get_position"
	getPositions                      = "private/get_positions"
	getSubAccounts                    = "private/get_subaccounts"
	getSubAccountDetails              = "private/get_subaccounts_details"
	getTransactionLog                 = "private/get_transaction_log"
	getUserLocks                      = "private/get_user_locks"
	listAPIKeys                       = "private/list_api_keys"
	removeAPIKey                      = "private/remove_api_key"
	removeSubAccount                  = "private/remove_subaccount"
	resetAPIKey                       = "private/reset_api_key"
	setAnnouncementAsRead             = "private/set_announcement_as_read"
	setAPIKeyAsDefault                = "private/set_api_key_as_default"
	setEmailForSubAccount             = "private/set_email_for_subaccount"
	setEmailLanguage                  = "private/set_email_language"
	setPasswordForSubAccount          = "private/set_password_for_subaccount"
	toggleNotificationsFromSubAccount = "private/toggle_notifications_from_subaccount"
	togglePortfolioMargining          = "private/toggle_portfolio_margining"
	toggleSubAccountLogin             = "private/toggle_subaccount_login"

	// Combo Books Endpoints
	getComboDetails = "public/get_combo_details"
	getComboIDS     = "public/get_combo_ids"
	getCombos       = "public/get_combos"
	createCombos    = "private/create_combo"

	// Block Trades Endpoints
	executeBlockTrades             = "private/execute_block_trade"
	getBlockTrades                 = "private/get_block_trade"
	getLastBlockTradesByCurrency   = "private/get_last_block_trades_by_currency"
	invalidateBlockTradesSignature = "private/invalidate_block_trade_signature"
	movePositions                  = "private/move_positions"
	verifyBlockTrades              = "private/verify_block_trade"
)

// Start implementing public and private exchange API funcs below

// GetBookSummaryByCurrency gets book summary data for currency requested
func (d *Deribit) GetBookSummaryByCurrency(ctx context.Context, currency, kind string) ([]BookSummaryData, error) {
	currency = strings.ToUpper(currency)
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	if kind != "" {
		params.Set("kind", kind)
	}
	var resp []BookSummaryData
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures,
		common.EncodeURLValues(getBookByCurrency, params), &resp)
}

// GetBookSummaryByInstrument gets book summary data for instrument requested
func (d *Deribit) GetBookSummaryByInstrument(ctx context.Context, instrument string) ([]BookSummaryData, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	params := url.Values{}
	params.Set("instrument_name", instrument)
	var resp []BookSummaryData
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures,
		common.EncodeURLValues(getBookByInstrument, params), &resp)
}

// GetContractSize gets contract size for instrument requested
func (d *Deribit) GetContractSize(ctx context.Context, instrument string) (*ContractSizeData, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	params := url.Values{}
	params.Set("instrument_name", instrument)
	var resp ContractSizeData
	return &resp, d.SendHTTPRequest(ctx, exchange.RestFutures,
		common.EncodeURLValues(getContractSize, params), &resp)
}

// GetCurrencies gets all cryptocurrencies supported by the API
func (d *Deribit) GetCurrencies(ctx context.Context) ([]CurrencyData, error) {
	var resp []CurrencyData
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, getCurrencies, &resp)
}

// GetDeliveryPrices gets all delivery prices for the given inde name
func (d *Deribit) GetDeliveryPrices(ctx context.Context, indexName string, offset, count int64) (*IndexDeliveryPrice, error) {
	indexNames := map[string]bool{"ada_usd": true, "avax_usd": true, "btc_usd": true, "eth_usd": true, "dot_usd": true, "luna_usd": true, "matic_usd": true, "sol_usd": true, "usdc_usd": true, "xrp_usd": true, "ada_usdc": true, "avax_usdc": true, "btc_usdc": true, "eth_usdc": true, "dot_usdc": true, "luna_usdc": true, "matic_usdc": true, "sol_usdc": true, "xrp_usdc": true, "btcdvol_usdc": true, "ethdvol_usdc": true}
	if !indexNames[indexName] {
		return nil, errUnsupportedIndexName
	}
	params := url.Values{}
	params.Set("index_name", indexName)
	if offset > 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	var resp IndexDeliveryPrice
	return &resp, d.SendHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues(getDeliveryPrices, params), &resp)
}

// GetFundingChartData gets funding chart data for the requested instrument and time length
// supported lengths: 8h, 24h, 1m <-(1month)
func (d *Deribit) GetFundingChartData(ctx context.Context, instrument, length string) (*FundingChartData, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	params := url.Values{}
	params.Set("instrument_name", instrument)
	if length != "8h" && length != "24h" && length != "12h" && length != "1m" {
		return nil, fmt.Errorf("%w, only 8h, 12h, 1m, and 24h are supported", errIntervalNotSupported)
	}
	params.Set("length", length)
	var resp FundingChartData
	return &resp, d.SendHTTPRequest(ctx, exchange.RestFutures,
		common.EncodeURLValues(getFundingChartData, params), &resp)
}

// GetFundingRateValue gets funding rate value data
func (d *Deribit) GetFundingRateValue(ctx context.Context, instrument string, startTime, endTime time.Time) (float64, error) {
	if instrument == "" {
		return 0, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	params := url.Values{}
	params.Set("instrument_name", instrument)
	if startTime.IsZero() {
		return 0, fmt.Errorf("%w, start time can not be zero", errInvalidTimestamp)
	} else if startTime.After(endTime) {
		return 0, errStartTimeCannotBeAfterEndTime
	}
	params.Set("start_timestamp", strconv.FormatInt(startTime.UnixMilli(), 10))
	if endTime.IsZero() {
		endTime = time.Now()
	}
	params.Set("end_timestamp", strconv.FormatInt(endTime.UnixMilli(), 10))
	var resp float64
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures,
		common.EncodeURLValues(getFundingRateValue, params), &resp)
}

// GetHistoricalVolatility gets historical volatility data
func (d *Deribit) GetHistoricalVolatility(ctx context.Context, currency string) ([]HistoricalVolatilityData, error) {
	currency = strings.ToUpper(currency)
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	var data [][2]interface{}
	err := d.SendHTTPRequest(ctx, exchange.RestFutures,
		common.EncodeURLValues(getHistoricalVolatility, params), &data)
	if err != nil {
		return nil, err
	}
	resp := make([]HistoricalVolatilityData, len(data))
	for x := range data {
		timeData, ok := data[x][0].(float64)
		if !ok {
			return resp, fmt.Errorf("%v GetHistoricalVolatility: %w for time", d.Name, errTypeAssert)
		}
		val, ok := data[x][1].(float64)
		if !ok {
			return resp, fmt.Errorf("%v GetHistoricalVolatility: %w for val", d.Name, errTypeAssert)
		}
		resp[x] = HistoricalVolatilityData{
			Timestamp: timeData,
			Value:     val,
		}
	}
	return resp, nil
}

// GetCurrencyIndexPrice retrives the current index price for the instruments, for the selected currency.
func (d *Deribit) GetCurrencyIndexPrice(ctx context.Context, currency string) (map[string]float64, error) {
	currency = strings.ToUpper(currency)
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidIndexPriceCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	var resp map[string]float64
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues(getCurrencyIndexPrice, params), &resp)
}

// GetIndexPrice gets price data for the requested index
func (d *Deribit) GetIndexPrice(ctx context.Context, index string) (*IndexPriceData, error) {
	indexNames := map[string]bool{"ada_usd": true, "avax_usd": true, "btc_usd": true, "eth_usd": true, "dot_usd": true, "luna_usd": true, "matic_usd": true, "sol_usd": true, "usdc_usd": true, "xrp_usd": true, "ada_usdc": true, "avax_usdc": true, "btc_usdc": true, "eth_usdc": true, "dot_usdc": true, "luna_usdc": true, "matic_usdc": true, "sol_usdc": true, "xrp_usdc": true, "btcdvol_usdc": true, "ethdvol_usdc": true}
	if !indexNames[index] {
		return nil, errUnsupportedIndexName
	}
	params := url.Values{}
	params.Set("index_name", index)
	var resp IndexPriceData
	return &resp, d.SendHTTPRequest(ctx, exchange.RestFutures,
		common.EncodeURLValues(getIndexPrice, params), &resp)
}

// GetIndexPriceNames gets names of indexes
func (d *Deribit) GetIndexPriceNames(ctx context.Context) ([]string, error) {
	var resp []string
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, getIndexPriceNames, &resp)
}

// GetInstrumentData gets data for a requested instrument
func (d *Deribit) GetInstrumentData(ctx context.Context, instrument string) (*InstrumentData, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	params := url.Values{}
	params.Set("instrument_name", instrument)
	var resp InstrumentData
	return &resp, d.SendHTTPRequest(ctx, exchange.RestFutures,
		common.EncodeURLValues(getInstrument, params), &resp)
}

// GetInstrumentsData gets data for all available instruments
func (d *Deribit) GetInstrumentsData(ctx context.Context, currency, kind string, expired bool) ([]InstrumentData, error) {
	currency = strings.ToUpper(currency)
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidIndexPriceCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	if kind != "" {
		params.Set("kind", kind)
	}
	params.Set("expired", strconv.FormatBool(expired))
	var resp []InstrumentData
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures,
		common.EncodeURLValues(getInstruments, params), &resp)
}

// GetLastSettlementsByCurrency gets last settlement data by currency
func (d *Deribit) GetLastSettlementsByCurrency(ctx context.Context, currency, settlementType, continuation string, count int64, startTime time.Time) (*SettlementsData, error) {
	currency = strings.ToUpper(currency)
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidIndexPriceCurrency
	}
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
		params.Set("search_start_timestamp", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	var resp SettlementsData
	return &resp, d.SendHTTPRequest(ctx, exchange.RestFutures,
		common.EncodeURLValues(getLastSettlementsByCurrency, params), &resp)
}

// GetLastSettlementsByInstrument gets last settlement data for requested instrument
func (d *Deribit) GetLastSettlementsByInstrument(ctx context.Context, instrument, settlementType, continuation string, count int64, startTime time.Time) (*SettlementsData, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
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
		params.Set("search_start_timestamp", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	var resp SettlementsData
	return &resp, d.SendHTTPRequest(ctx, exchange.RestFutures,
		common.EncodeURLValues(getLastSettlementsByInstrument, params), &resp)
}

// GetLastTradesByCurrency gets last trades for requested currency
func (d *Deribit) GetLastTradesByCurrency(ctx context.Context, currency, kind, startID, endID, sorting string, count int64, includeOld bool) (*PublicTradesData, error) {
	currency = strings.ToUpper(currency)
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidIndexPriceCurrency
	}
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
	params.Set("include_old", strconv.FormatBool(includeOld))
	var resp PublicTradesData
	return &resp, d.SendHTTPRequest(ctx, exchange.RestFutures,
		common.EncodeURLValues(getLastTradesByCurrency, params), &resp)
}

// GetLastTradesByCurrencyAndTime gets last trades for requested currency and time intervals
func (d *Deribit) GetLastTradesByCurrencyAndTime(ctx context.Context, currency, kind, sorting string, count int64, includeOld bool, startTime, endTime time.Time) (*PublicTradesData, error) {
	currency = strings.ToUpper(currency)
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidIndexPriceCurrency
	}
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
	if startTime.IsZero() {
		return nil, fmt.Errorf("%w, start time can not be zero", errInvalidTimestamp)
	} else if startTime.After(endTime) {
		return nil, errStartTimeCannotBeAfterEndTime
	}
	params.Set("start_timestamp", strconv.FormatInt(startTime.UnixMilli(), 10))
	if endTime.IsZero() {
		endTime = time.Now()
	}
	params.Set("end_timestamp", strconv.FormatInt(endTime.UnixMilli(), 10))
	var resp PublicTradesData
	return &resp, d.SendHTTPRequest(ctx, exchange.RestFutures,
		common.EncodeURLValues(getLastTradesByCurrencyAndTime, params), &resp)
}

// GetLastTradesByInstrument gets last trades for requested instrument requested
func (d *Deribit) GetLastTradesByInstrument(ctx context.Context, instrument, startSeq, endSeq, sorting string, count int64, includeOld bool) (*PublicTradesData, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	params := url.Values{}
	params.Set("instrument_name", instrument)
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
	params.Set("include_old", strconv.FormatBool(includeOld))
	var resp PublicTradesData
	return &resp, d.SendHTTPRequest(ctx, exchange.RestFutures,
		common.EncodeURLValues(getLastTradesByInstrument, params), &resp)
}

// GetLastTradesByInstrumentAndTime gets last trades for requested instrument requested and time intervals
func (d *Deribit) GetLastTradesByInstrumentAndTime(ctx context.Context, instrument, sorting string, count int64, includeOld bool, startTime, endTime time.Time) (*PublicTradesData, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	params := url.Values{}
	params.Set("instrument_name", instrument)
	if sorting != "" {
		params.Set("sorting", sorting)
	}
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	if startTime.IsZero() {
		return nil, fmt.Errorf("%w, start time can not be zero", errInvalidTimestamp)
	} else if startTime.After(endTime) {
		return nil, errStartTimeCannotBeAfterEndTime
	}
	params.Set("start_timestamp", strconv.FormatInt(startTime.UnixMilli(), 10))
	if endTime.IsZero() {
		endTime = time.Now()
	}
	params.Set("end_timestamp", strconv.FormatInt(endTime.UnixMilli(), 10))
	var resp PublicTradesData
	return &resp, d.SendHTTPRequest(ctx, exchange.RestFutures,
		common.EncodeURLValues(getLastTradesByInstrumentAndTime, params), &resp)
}

// GetMarkPriceHistory gets data for mark price history
func (d *Deribit) GetMarkPriceHistory(ctx context.Context, instrument string, startTime, endTime time.Time) ([]MarkPriceHistory, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	params := url.Values{}
	params.Set("instrument_name", instrument)
	if startTime.IsZero() {
		return nil, fmt.Errorf("%w, start time can not be zero", errInvalidTimestamp)
	} else if startTime.After(endTime) {
		return nil, errStartTimeCannotBeAfterEndTime
	}
	params.Set("start_timestamp", strconv.FormatInt(startTime.UnixMilli(), 10))
	if endTime.IsZero() {
		endTime = time.Now()
	}
	params.Set("end_timestamp", strconv.FormatInt(endTime.UnixMilli(), 10))
	var resp []MarkPriceHistory
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures,
		common.EncodeURLValues(getMarkPriceHistory, params), &resp)
}

// GetOrderbookData gets data orderbook of requested instrument
func (d *Deribit) GetOrderbookData(ctx context.Context, instrument string, depth int64) (*Orderbook, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	params := url.Values{}
	params.Set("instrument_name", instrument)
	if depth != 0 {
		params.Set("depth", strconv.FormatInt(depth, 10))
	}
	var resp Orderbook
	return &resp, d.SendHTTPRequest(ctx, exchange.RestFutures,
		common.EncodeURLValues(getOrderbook, params), &resp)
}

// GetOrderbookByInstrumentID retrives orderbook by instrument ID
func (d *Deribit) GetOrderbookByInstrumentID(ctx context.Context, instrumentID int64, depth float64) (*Orderbook, error) {
	if instrumentID == 0 {
		return nil, errInvalidInstrumentID
	}
	params := url.Values{}
	params.Set("instrument_id", strconv.FormatInt(instrumentID, 10))
	if depth != 0 {
		params.Set("depth", strconv.FormatFloat(depth, 'f', -1, 64))
	}
	var response Orderbook
	return &response, d.SendHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues(getOrderbookByInstrumentID, params), &response)
}

// GetRFQ retrives RFQ information.
func (d *Deribit) GetRFQ(ctx context.Context, currency, kind string) ([]RFQ, error) {
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidIndexPriceCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	if kind != "" {
		params.Set("kind", kind)
	}
	var resp []RFQ
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues(getRFQ, params), &resp)
}

// GetTradeVolumes gets trade volumes' data of all instruments
func (d *Deribit) GetTradeVolumes(ctx context.Context, extended bool) ([]TradeVolumesData, error) {
	params := url.Values{}
	params.Set("extended", strconv.FormatBool(extended))
	var resp []TradeVolumesData
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures,
		common.EncodeURLValues(getTradeVolumes, params), &resp)
}

// GetTradingViewChartData gets volatility index data for the requested instrument
func (d *Deribit) GetTradingViewChartData(ctx context.Context, instrument, resolution string, startTime, endTime time.Time) (*TVChartData, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	params := url.Values{}
	params.Set("instrument_name", instrument)
	if startTime.IsZero() {
		return nil, fmt.Errorf("%w, start time can not be zero", errInvalidTimestamp)
	} else if startTime.After(endTime) {
		return nil, errStartTimeCannotBeAfterEndTime
	}
	params.Set("start_timestamp", strconv.FormatInt(startTime.UnixMilli(), 10))
	if endTime.IsZero() {
		endTime = time.Now()
	}
	params.Set("end_timestamp", strconv.FormatInt(endTime.UnixMilli(), 10))
	resolutionMap := map[string]bool{"1": true, "3": true, "5": true, "10": true, "15": true, "30": true, "60": true, "120": true, "180": true, "360": true, "720": true, "1D": true}
	if !resolutionMap[resolution] {
		return nil, fmt.Errorf("unsupported resolution, only 1,3,5,10,15,30,60,120,180,360,720, and 1D are supported")
	}
	params.Set("resolution", resolution)
	var resp TVChartData
	return &resp, d.SendHTTPRequest(ctx, exchange.RestFutures,
		common.EncodeURLValues(getTradingViewChartData, params), &resp)
}

// GetResolutionFromInterval returns the string representation of intervals given kline.Interval instance.
func (d *Deribit) GetResolutionFromInterval(interval kline.Interval) (string, error) {
	switch interval {
	case kline.HundredMilliSec:
		return "100ms", nil
	case kline.OneMin:
		return "1", nil
	case kline.ThreeMin:
		return "3", nil
	case kline.FiveMin:
		return "5", nil
	case kline.TenMin:
		return "10", nil
	case kline.FifteenMin:
		return "15", nil
	case kline.ThirtyMin:
		return "30", nil
	case kline.OneHour:
		return "60", nil
	case kline.TwoHour:
		return "120", nil
	case kline.ThreeHour:
		return "180", nil
	case kline.SixHour:
		return "360", nil
	case kline.TwelveHour:
		return "720", nil
	case kline.OneDay:
		return "1D", nil
	default:
		return "", kline.ErrUnsupportedInterval
	}
}

// GetVolatilityIndexData gets volatility index data for the requested currency
func (d *Deribit) GetVolatilityIndexData(ctx context.Context, currency, resolution string, startTime, endTime time.Time) (*VolatilityIndexData, error) {
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidIndexPriceCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	if startTime.IsZero() {
		return nil, fmt.Errorf("%w, start time can not be zero", errInvalidTimestamp)
	} else if !endTime.IsZero() && startTime.After(endTime) {
		return nil, errStartTimeCannotBeAfterEndTime
	}
	params.Set("start_timestamp", strconv.FormatInt(startTime.UnixMilli(), 10))
	if endTime.IsZero() {
		endTime = time.Now()
	}
	params.Set("end_timestamp", strconv.FormatInt(endTime.UnixMilli(), 10))
	resolutionMap := map[string]bool{"1": true, "60": true, "3600": true, "43200": true, "1D": true}
	if !resolutionMap[resolution] {
		return nil, fmt.Errorf("unsupported resolution, only 1 ,60 ,3600 ,43200 and 1D are supported")
	}
	params.Set("resolution", resolution)
	var resp VolatilityIndexData
	return &resp, d.SendHTTPRequest(ctx, exchange.RestFutures,
		common.EncodeURLValues(getVolatilityIndexData, params), &resp)
}

// GetPublicTicker gets public ticker data of the instrument requested
func (d *Deribit) GetPublicTicker(ctx context.Context, instrument string) (*TickerData, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	params := url.Values{}
	params.Set("instrument_name", instrument)
	var resp TickerData
	return &resp, d.SendHTTPRequest(ctx, exchange.RestFutures,
		common.EncodeURLValues(getTicker, params), &resp)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (d *Deribit) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, result interface{}) error {
	endpoint, err := d.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	var data struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      int64           `json:"id"`
		Data    json.RawMessage `json:"result"`
	}
	err = d.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		return &request.Item{
			Method:        http.MethodGet,
			Path:          endpoint + deribitAPIVersion + "/" + path,
			Result:        &data,
			Verbose:       d.Verbose,
			HTTPDebugging: d.HTTPDebugging,
			HTTPRecording: d.HTTPRecording,
		}, nil
	})
	if err != nil {
		return err
	}
	return json.Unmarshal(data.Data, result)
}

// GetAccountSummary gets account summary data for the requested instrument
func (d *Deribit) GetAccountSummary(ctx context.Context, currency string, extended bool) (*AccountSummaryData, error) {
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidIndexPriceCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	params.Set("extended", strconv.FormatBool(extended))
	var resp AccountSummaryData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet, getAccountSummary, params, &resp)
}

// CancelWithdrawal cancels withdrawal request for a given currency by its id
func (d *Deribit) CancelWithdrawal(ctx context.Context, currency string, id int64) (*CancelWithdrawalData, error) {
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidIndexPriceCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	if id <= 0 {
		return nil, fmt.Errorf("%w, withdrawal id has to be positive integer", errInvalidID)
	}
	params.Set("id", strconv.FormatInt(id, 10))
	var resp CancelWithdrawalData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		cancelWithdrawal, params, &resp)
}

// CancelTransferByID cancels transfer by ID through the websocket connection.
func (d *Deribit) CancelTransferByID(ctx context.Context, currency, tfa string, id int64) (*AccountSummaryData, error) {
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidIndexPriceCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	if tfa != "" {
		params.Set("tfa", tfa)
	}
	if id <= 0 {
		return nil, fmt.Errorf("%w, transfer id has to be positive integer", errInvalidID)
	}
	params.Set("id", strconv.FormatInt(id, 10))
	var resp AccountSummaryData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet, cancelTransferByID, params, &resp)
}

// CreateDepositAddress creates a deposit address for the currency requested
func (d *Deribit) CreateDepositAddress(ctx context.Context, currency string) (*DepositAddressData, error) {
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidIndexPriceCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	var resp DepositAddressData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		createDepositAddress, params, &resp)
}

// GetCurrentDepositAddress gets the current deposit address for the requested currency
func (d *Deribit) GetCurrentDepositAddress(ctx context.Context, currency string) (*DepositAddressData, error) {
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidIndexPriceCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	var resp DepositAddressData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet, createDepositAddress, params, &resp)
}

// GetDeposits gets the deposits of a given currency
func (d *Deribit) GetDeposits(ctx context.Context, currency string, count, offset int64) (*DepositsData, error) {
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidIndexPriceCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	if offset != 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	var resp DepositsData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		getDeposits, params, &resp)
}

// GetTransfers gets transfers data for the requested currency
func (d *Deribit) GetTransfers(ctx context.Context, currency string, count, offset int64) (*TransferData, error) {
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidIndexPriceCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	if offset != 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	var resp TransferData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		getTransfers, params, &resp)
}

// GetWithdrawals gets withdrawals data for a requested currency
func (d *Deribit) GetWithdrawals(ctx context.Context, currency string, count, offset int64) (*WithdrawalsData, error) {
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidIndexPriceCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	if offset != 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	var resp WithdrawalsData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		getWithdrawals, params, &resp)
}

// SubmitTransferToSubAccount submits a request to transfer a currency to a subaccount
func (d *Deribit) SubmitTransferToSubAccount(ctx context.Context, currency string, amount float64, destinationID int64) (*TransferData, error) {
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidIndexPriceCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	if amount <= 0 {
		return nil, errInvalidAmount
	}
	if destinationID <= 0 {
		return nil, errors.New("invalid destination address")
	}
	params.Set("destination", strconv.FormatInt(destinationID, 10))
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp TransferData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		submitTransferToSubaccount, params, &resp)
}

// SubmitTransferToUser submits a request to transfer a currency to another user
func (d *Deribit) SubmitTransferToUser(ctx context.Context, currency, tfa, destinationAddress string, amount float64) (*TransferData, error) {
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidIndexPriceCurrency
	}
	if amount <= 0 {
		return nil, errInvalidAmount
	}
	params := url.Values{}
	params.Set("currency", currency)
	if tfa != "" {
		params.Set("tfa", tfa)
	}
	if destinationAddress == "" {
		return nil, errors.New("invalid destination address")
	}
	params.Set("destination", destinationAddress)
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp TransferData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		submitTransferToUser, params, &resp)
}

// SubmitWithdraw submits a withdrawal request to the exchange for the requested currency
func (d *Deribit) SubmitWithdraw(ctx context.Context, currency, address, priority string, amount float64) (*WithdrawData, error) {
	currency = strings.ToUpper(currency)
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidCurrency
	}
	if amount <= 0 {
		return nil, errInvalidAmount
	}
	params := url.Values{}
	if address == "" {
		return nil, errInvalidCryptoAddress
	}
	params.Set("currency", currency)
	params.Set("address", address)
	if priority != "" && priority != "insane" && priority != "extreme_high" && priority != "very_high" && priority != "high" && priority != "mid" && priority != "low" && priority != "very_low" {
		return nil, errors.New("unsupported priority '%s', only insane ,extreme_high ,very_high ,high ,mid ,low ,and very_low")
	} else if priority != "" {
		params.Set("priority", priority)
	}
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp WithdrawData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		submitWithdraw, params, &resp)
}

// GetAnnouncements retrieves announcements. Default "start_timestamp" parameter value is current timestamp, "count" parameter value must be between 1 and 50, default is 5.
func (d *Deribit) GetAnnouncements(ctx context.Context, startTime time.Time, count int64) ([]Announcement, error) {
	params := url.Values{}
	if !startTime.IsZero() {
		params.Set("start_time", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if count > 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	var resp []Announcement
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues(getAnnouncements, params), &resp)
}

// GetPublicPortfolioMargins public version of the method calculates portfolio margin info for simulated position. For concrete user position, the private version of the method must be used. The public version of the request has special restricted rate limit (not more than once per a second for the IP).
func (d *Deribit) GetPublicPortfolioMargins(ctx context.Context, currency string, simulatedPositions map[string]float64) (*PortfolioMargin, error) {
	currency = strings.ToUpper(currency)
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	if len(simulatedPositions) != 0 {
		values, err := json.Marshal(simulatedPositions)
		if err != nil {
			return nil, err
		}
		params.Set("simulated_positions", string(values))
	}
	var resp PortfolioMargin
	return &resp, d.SendHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues(getPublicPortfolioMargins, params), &resp)
}

// ChangeAPIKeyName changes the name of the api key requested
func (d *Deribit) ChangeAPIKeyName(ctx context.Context, id int64, name string) (*APIKeyData, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w, invalid api key id", errInvalidID)
	}
	if !regexp.MustCompile("^[a-zA-Z0-9_]*$").MatchString(name) {
		return nil, errors.New("unacceptable api key name")
	}
	params := url.Values{}
	params.Set("id", strconv.FormatInt(id, 10))
	params.Set("name", name)
	var resp APIKeyData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		changeAPIKeyName, params, &resp)
}

// ChangeScopeInAPIKey changes the scope of the api key requested
func (d *Deribit) ChangeScopeInAPIKey(ctx context.Context, id int64, maxScope string) (*APIKeyData, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w, invalid api key id", errInvalidID)
	}
	params := url.Values{}
	params.Set("id", strconv.FormatInt(id, 10))
	params.Set("max_scope", maxScope)
	var resp APIKeyData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		changeScopeInAPIKey, params, &resp)
}

// ChangeSubAccountName changes the name of the requested subaccount id
func (d *Deribit) ChangeSubAccountName(ctx context.Context, sid int64, name string) (string, error) {
	if sid <= 0 {
		return "", fmt.Errorf("%w, invalid subaccount user id", errInvalidID)
	}
	if name == "" {
		return "", errors.New("new username has to be specified")
	}
	params := url.Values{}
	params.Set("sid", strconv.FormatInt(sid, 10))
	params.Set("name", name)
	var resp string
	err := d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		changeSubAccountName, params, &resp)
	if err != nil {
		return "", err
	}
	if resp != "ok" {
		return "", fmt.Errorf("subaccount name change failed")
	}
	return resp, nil
}

// CreateAPIKey creates an api key based on the provided settings
func (d *Deribit) CreateAPIKey(ctx context.Context, maxScope, name string, defaultKey bool) (*APIKeyData, error) {
	params := url.Values{}
	params.Set("max_scope", maxScope)
	if name != "" {
		params.Set("name", name)
	}
	params.Set("default", strconv.FormatBool(defaultKey))
	var resp APIKeyData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		createAPIKey, params, &resp)
}

// CreateSubAccount creates a new subaccount
func (d *Deribit) CreateSubAccount(ctx context.Context) (*SubAccountData, error) {
	var resp SubAccountData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		createSubAccount, nil, &resp)
}

// DisableAPIKey disables the api key linked to the provided id
func (d *Deribit) DisableAPIKey(ctx context.Context, id int64) (*APIKeyData, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w, invalid api key id", errInvalidID)
	}
	params := url.Values{}
	params.Set("id", strconv.FormatInt(id, 10))
	var resp APIKeyData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		disableAPIKey, params, &resp)
}

// DisableTFAForSubAccount disables two factor authentication for the subaccount linked to the requested id
func (d *Deribit) DisableTFAForSubAccount(ctx context.Context, sid int64) (string, error) {
	if sid <= 0 {
		return "", fmt.Errorf("%w, invalid subaccount user id", errInvalidID)
	}
	params := url.Values{}
	params.Set("sid", strconv.FormatInt(sid, 10))
	var resp string
	err := d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		disableTFAForSubaccount, params, &resp)
	if err != nil {
		return "", err
	}
	if resp != "ok" {
		return "", fmt.Errorf("disabling 2fa for subaccount %v failed", sid)
	}
	return resp, nil
}

// EnableAffiliateProgram enables the affiliate program
func (d *Deribit) EnableAffiliateProgram(ctx context.Context) (string, error) {
	var resp string
	err := d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		enableAffiliateProgram, nil, &resp)
	if err != nil {
		return "", err
	}
	if resp != "ok" {
		return "", fmt.Errorf("could not enable affiliate program")
	}
	return resp, nil
}

// EnableAPIKey enables the api key linked to the provided id
func (d *Deribit) EnableAPIKey(ctx context.Context, id int64) (*APIKeyData, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w, invalid api key id", errInvalidID)
	}
	params := url.Values{}
	params.Set("id", strconv.FormatInt(id, 10))
	var resp APIKeyData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		enableAPIKey, params, &resp)
}

// GetAccessLog lists access logs for the user
func (d *Deribit) GetAccessLog(ctx context.Context, offset, count int64) (*AccessLog, error) {
	params := url.Values{}
	if offset > 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	if count > 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	var resp AccessLog
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet, getAccessLog, params, &resp)
}

// [[ TODO ]]

// GetAffiliateProgramInfo gets the affiliate program info
func (d *Deribit) GetAffiliateProgramInfo(ctx context.Context, id int64) (*AffiliateProgramInfo, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w, invalid api key id", errInvalidID)
	}
	var resp AffiliateProgramInfo
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		getAffiliateProgramInfo, nil, &resp)
}

// GetEmailLanguage gets the current language set for the email
func (d *Deribit) GetEmailLanguage(ctx context.Context) (string, error) {
	var resp string
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		getEmailLanguage, nil, &resp)
}

// GetNewAnnouncements gets new announcements
func (d *Deribit) GetNewAnnouncements(ctx context.Context) ([]Announcement, error) {
	var resp []Announcement
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		getNewAnnouncements, nil, &resp)
}

// GetPricatePortfolioMargins calculates portfolio margin info for simulated position or current position of the user. This request has special restricted rate limit (not more than once per a second).
func (d *Deribit) GetPricatePortfolioMargins(ctx context.Context, currency string, accPositions bool, simulatedPositions map[string]float64) (*PortfolioMargin, error) {
	currency = strings.ToUpper(currency)
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	if accPositions {
		params.Set("acc_positions", strconv.FormatBool(accPositions))
	}
	if len(simulatedPositions) != 0 {
		values, err := json.Marshal(simulatedPositions)
		if err != nil {
			return nil, err
		}
		params.Set("simulated_positions", string(values))
	}
	var resp PortfolioMargin
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet, getPrivatePortfolioMargins, params, &resp)
}

// GetPosition gets the data of all positions in the requested instrument name
func (d *Deribit) GetPosition(ctx context.Context, instrument string) (*PositionData, error) {
	if instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	params := url.Values{}
	params.Set("instrument_name", instrument)
	var resp PositionData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		getPosition, params, &resp)
}

// GetSubAccounts gets all subaccounts' data
func (d *Deribit) GetSubAccounts(ctx context.Context, withPortfolio bool) ([]SubAccountData, error) {
	params := url.Values{}
	params.Set("with_portfolio", strconv.FormatBool(withPortfolio))
	var resp []SubAccountData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		getSubAccounts, params, &resp)
}

// GetSubAccountDetails retrives sub accounts detail information.
func (d *Deribit) GetSubAccountDetails(ctx context.Context, currency string, withOpenOrders bool) ([]SubAccountDetail, error) {
	currency = strings.ToUpper(currency)
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	if withOpenOrders {
		params.Set("with_open_orders", strconv.FormatBool(withOpenOrders))
	}
	var resp []SubAccountDetail
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet, getSubAccountDetails, params, &resp)
}

// GetPositions gets positions data of the user account
func (d *Deribit) GetPositions(ctx context.Context, currency, kind string) ([]PositionData, error) {
	currency = strings.ToUpper(currency)
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	if kind != "" {
		params.Set("kind", kind)
	}
	var resp []PositionData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		getPositions, params, &resp)
}

// GetTransactionLog gets transaction logs' data
func (d *Deribit) GetTransactionLog(ctx context.Context, currency, query string, startTime, endTime time.Time, count, continuation int64) (*TransactionsData, error) {
	currency = strings.ToUpper(currency)
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	if query != "" {
		params.Set("query", query)
	}
	if startTime.IsZero() {
		return nil, fmt.Errorf("%w, start time can not be zero", errInvalidTimestamp)
	} else if startTime.After(endTime) {
		return nil, errStartTimeCannotBeAfterEndTime
	}
	params.Set("start_timestamp", strconv.FormatInt(startTime.UnixMilli(), 10))
	if endTime.IsZero() {
		endTime = time.Now()
	}
	params.Set("end_timestamp", strconv.FormatInt(endTime.UnixMilli(), 10))
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	if continuation != 0 {
		params.Set("continuation", strconv.FormatInt(continuation, 10))
	}
	var resp TransactionsData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet, getTransactionLog, params, &resp)
}

// GetUserLocks retrieves information about locks on user account.
func (d *Deribit) GetUserLocks(ctx context.Context) ([]UserLock, error) {
	var resp []UserLock
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet, getUserLocks, nil, &resp)
}

// ListAPIKeys lists all the api keys associated with a user account
func (d *Deribit) ListAPIKeys(ctx context.Context, tfa string) ([]APIKeyData, error) {
	params := url.Values{}
	if tfa != "" {
		params.Set("tfa", tfa)
	}
	var resp []APIKeyData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		listAPIKeys, params, &resp)
}

// RemoveAPIKey removes api key vid ID
func (d *Deribit) RemoveAPIKey(ctx context.Context, id int64) (string, error) {
	if id <= 0 {
		return "", fmt.Errorf("%w, invalid api key id", errInvalidID)
	}
	params := url.Values{}
	params.Set("id", strconv.FormatInt(id, 10))
	var resp interface{}
	err := d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet, removeAPIKey, params, &resp)
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
			return "", err
		}
		return string(data), nil
	}
	if resp != "ok" {
		return "", fmt.Errorf("removal of the api key requested failed")
	}
	return "ok", nil
}

// RemoveSubAccount removes a subaccount given its id
func (d *Deribit) RemoveSubAccount(ctx context.Context, subAccountID int64) (string, error) {
	params := url.Values{}
	params.Set("subaccount_id", strconv.FormatInt(subAccountID, 10))
	var resp string
	err := d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		removeSubAccount, params, &resp)
	if err != nil {
		return "", err
	}
	if resp != "ok" {
		return "", fmt.Errorf("removal of sub account %v failed", subAccountID)
	}
	return resp, nil
}

// ResetAPIKey resets the api key to its default settings
func (d *Deribit) ResetAPIKey(ctx context.Context, id int64) (*APIKeyData, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w, invalid api key id", errInvalidID)
	}
	params := url.Values{}
	params.Set("id", strconv.FormatInt(id, 10))
	var resp APIKeyData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		resetAPIKey, params, &resp)
}

// SetAnnouncementAsRead sets an announcement as read
func (d *Deribit) SetAnnouncementAsRead(ctx context.Context, id int64) (string, error) {
	if id <= 0 {
		return "", fmt.Errorf("%w, invalid announcement id", errInvalidID)
	}
	params := url.Values{}
	params.Set("announcement_id", strconv.FormatInt(id, 10))
	var resp string
	err := d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		setAnnouncementAsRead, params, &resp)
	if err != nil {
		return "", err
	}
	if resp != "ok" {
		return "", fmt.Errorf("setting announcement %v as read failed", id)
	}
	return resp, nil
}

// SetEmailForSubAccount links an email given to the designated subaccount
func (d *Deribit) SetEmailForSubAccount(ctx context.Context, sid int64, email string) (string, error) {
	if sid <= 0 {
		return "", fmt.Errorf("%w, invalid subaccount user id", errInvalidID)
	}
	if !common.MatchesEmailPattern(email) {
		return "", errInvalidEmailAddress
	}
	params := url.Values{}
	params.Set("sid", strconv.FormatInt(sid, 10))
	params.Set("email", email)
	var resp interface{}
	err := d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		setEmailForSubAccount, params, &resp)
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
			return "", err
		}
		return string(data), nil
	}
	if resp != "ok" {
		return "", fmt.Errorf("could not link email (%v) to subaccount %v", email, sid)
	}
	return "resp", nil
}

// SetEmailLanguage sets a requested language for an email
func (d *Deribit) SetEmailLanguage(ctx context.Context, language string) (string, error) {
	if language != "en" && language != "ko" && language != "zh" && language != "ja" && language != "ru" {
		return "", errors.New("invalid language, only 'en', 'ko', 'zh', 'ja' and 'ru' are supported")
	}
	params := url.Values{}
	params.Set("language", language)
	var resp string
	err := d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		setEmailLanguage, params, &resp)
	if err != nil {
		return "", err
	}
	if resp != "ok" {
		return "", fmt.Errorf("could not set the email language to %v", language)
	}
	return resp, nil
}

// SetPasswordForSubAccount sets a password for subaccount usage
func (d *Deribit) SetPasswordForSubAccount(ctx context.Context, sid int64, password string) (string, error) {
	if sid <= 0 {
		return "", fmt.Errorf("%w, invalid subaccount user id", errInvalidID)
	}
	if password == "" {
		return "", errors.New("subaccount password must not be empty")
	}
	params := url.Values{}
	params.Set("password", password)
	params.Set("sid", strconv.FormatInt(sid, 10))
	var resp interface{}
	err := d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		setPasswordForSubAccount, params, &resp)
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
			return "", err
		}
		return string(data), nil
	}
	if resp != "ok" {
		return "", fmt.Errorf("could not set the provided password to subaccount %v", sid)
	}
	return "ok", nil
}

// ToggleNotificationsFromSubAccount toggles the notifications from a subaccount specified
func (d *Deribit) ToggleNotificationsFromSubAccount(ctx context.Context, sid int64, state bool) (string, error) {
	if sid <= 0 {
		return "", fmt.Errorf("%w, invalid subaccount user id", errInvalidID)
	}
	params := url.Values{}
	params.Set("sid", strconv.FormatInt(sid, 10))
	params.Set("state", strconv.FormatBool(state))
	var resp string
	err := d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		toggleNotificationsFromSubAccount, params, &resp)
	if err != nil {
		return "", err
	}
	if resp != "ok" {
		return "", fmt.Errorf("toggling notifications for subaccount %v to %v failed", sid, state)
	}
	return resp, nil
}

// TogglePortfolioMargining toggle between SM and PM models.
func (d *Deribit) TogglePortfolioMargining(ctx context.Context, userID int64, enabled, dryRun bool) ([]TogglePortfolioMarginResponse, error) {
	if userID == 0 {
		return nil, errors.New("missing user id")
	}
	params := url.Values{}
	params.Set("user_id", strconv.FormatInt(userID, 10))
	params.Set("enabled", strconv.FormatBool(enabled))
	if dryRun {
		params.Set("dry_run", strconv.FormatBool(dryRun))
	}
	var resp []TogglePortfolioMarginResponse
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet, togglePortfolioMargining, params, &resp)
}

// ToggleSubAccountLogin toggles access for subaccount login
func (d *Deribit) ToggleSubAccountLogin(ctx context.Context, sid int64, state bool) (string, error) {
	if sid <= 0 {
		return "", fmt.Errorf("%w, invalid subaccount user id", errInvalidID)
	}
	params := url.Values{}
	params.Set("sid", strconv.FormatInt(sid, 10))
	params.Set("state", strconv.FormatBool(state))
	var resp string
	err := d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		toggleSubAccountLogin, params, &resp)
	if err != nil {
		return "", err
	}
	if resp != "ok" {
		return "", fmt.Errorf("toggling login access for subaccount %v to %v failed", sid, state)
	}
	return resp, nil
}

// SubmitBuy submits a private buy request through the websocket connection.
func (d *Deribit) SubmitBuy(ctx context.Context, arg *OrderBuyAndSellParams) (*PrivateTradeData, error) {
	if arg == nil {
		return nil, fmt.Errorf("%w parameter is required", common.ErrNilPointer)
	}
	if arg.Instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	params := url.Values{}
	params.Set("instrument_name", arg.Instrument)
	params.Set("amount", strconv.FormatFloat(arg.Amount, 'f', -1, 64))
	if arg.OrderType != "" {
		params.Set("type", arg.OrderType)
	}
	if arg.Price != 0 {
		params.Set("price", strconv.FormatFloat(arg.Amount, 'f', -1, 64))
	}
	if arg.Label != "" {
		params.Set("label", arg.Label)
	}
	if arg.TimeInForce != "" {
		params.Set("time_in_force", arg.TimeInForce)
	}
	if arg.MaxShow != 0 {
		params.Set("max_show", strconv.FormatFloat(arg.MaxShow, 'f', -1, 64))
	}
	params.Set("post_only", strconv.FormatBool(arg.PostOnly))
	params.Set("reject_post_only", strconv.FormatBool(arg.RejectPostOnly))
	params.Set("reduce_only", strconv.FormatBool(arg.ReduceOnly))
	params.Set("mmp", strconv.FormatBool(arg.MMP))
	if arg.TriggerPrice != 0 {
		params.Set("trigger_price", strconv.FormatFloat(arg.TriggerPrice, 'f', -1, 64))
	}
	if arg.Trigger != "" {
		params.Set("trigger", arg.Trigger)
	}
	if arg.Advanced != "" {
		params.Set("advanced", arg.Advanced)
	}
	var resp PrivateTradeData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		submitBuy, params, &resp)
}

// SubmitSell submits a sell request with the parameters provided
func (d *Deribit) SubmitSell(ctx context.Context, arg *OrderBuyAndSellParams) (*PrivateTradeData, error) {
	if arg == nil {
		return nil, fmt.Errorf("%s argument is required", common.ErrNilPointer)
	}
	if arg.Instrument == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	params := url.Values{}
	params.Set("instrument_name", arg.Instrument)
	params.Set("amount", strconv.FormatFloat(arg.Amount, 'f', -1, 64))
	if arg.OrderType != "" {
		params.Set("type", arg.OrderType)
	}
	if arg.Label != "" {
		params.Set("label", arg.Label)
	}
	if arg.TimeInForce != "" {
		params.Set("time_in_force", arg.TimeInForce)
	}
	if arg.MaxShow != 0 {
		params.Set("max_show", strconv.FormatFloat(arg.MaxShow, 'f', -1, 64))
	}
	if arg.Price <= 0 {
		return nil, errInvalidPrice
	}
	params.Set("price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	params.Set("post_only", strconv.FormatBool(arg.PostOnly))
	params.Set("reject_post_only", strconv.FormatBool(arg.RejectPostOnly))
	params.Set("reduce_only", strconv.FormatBool(arg.ReduceOnly))
	params.Set("mmp", strconv.FormatBool(arg.MMP))
	if arg.TriggerPrice != 0 {
		params.Set("trigger_price", strconv.FormatFloat(arg.TriggerPrice, 'f', -1, 64))
	}
	if arg.Trigger != "" {
		params.Set("trigger", arg.Trigger)
	}
	if arg.Advanced != "" {
		params.Set("advanced", arg.Advanced)
	}
	var resp PrivateTradeData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		submitSell, params, &resp)
}

// SubmitEdit submits an edit order request
func (d *Deribit) SubmitEdit(ctx context.Context, arg *OrderBuyAndSellParams) (*PrivateTradeData, error) {
	if arg == nil {
		return nil, fmt.Errorf("%w parameter is required", common.ErrNilPointer)
	}
	if arg.OrderID == "" {
		return nil, fmt.Errorf("%w, order id is required", errInvalidID)
	}
	if arg.Amount <= 0 {
		return nil, errInvalidAmount
	}
	params := url.Values{}
	params.Set("order_id", arg.OrderID)
	params.Set("amount", strconv.FormatFloat(arg.Amount, 'f', -1, 64))
	params.Set("post_only", strconv.FormatBool(arg.PostOnly))
	params.Set("reject_post_only", strconv.FormatBool(arg.RejectPostOnly))
	params.Set("reduce_only", strconv.FormatBool(arg.ReduceOnly))
	params.Set("mmp", strconv.FormatBool(arg.MMP))
	if arg.TriggerPrice != 0 {
		params.Set("trigger_price", strconv.FormatFloat(arg.TriggerPrice, 'f', -1, 64))
	}
	if arg.Advanced != "" {
		params.Set("advanced", arg.Advanced)
	}
	if arg.Price > 0 {
		params.Set("price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	}
	var resp PrivateTradeData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet, submitEdit, params, &resp)
}

// EditOrderByLabel submits an edit order request sorted via label
func (d *Deribit) EditOrderByLabel(ctx context.Context, arg *OrderBuyAndSellParams) (*PrivateTradeData, error) {
	if arg == nil {
		return nil, fmt.Errorf("%w parameter is required", common.ErrNilPointer)
	}
	if arg.Instrument == "" {
		return nil, errInvalidInstrumentName
	}
	if arg.Amount <= 0 {
		return nil, errInvalidAmount
	}
	params := url.Values{}
	if arg.Label != "" {
		params.Set("label", arg.Label)
	}
	params.Set("instrument_name", arg.Instrument)
	params.Set("amount", strconv.FormatFloat(arg.Amount, 'f', -1, 64))
	params.Set("post_only", strconv.FormatBool(arg.PostOnly))
	params.Set("reject_post_only", strconv.FormatBool(arg.RejectPostOnly))
	params.Set("reduce_only", strconv.FormatBool(arg.ReduceOnly))
	params.Set("mmp", strconv.FormatBool(arg.MMP))
	if arg.TriggerPrice != 0 {
		params.Set("trigger_price", strconv.FormatFloat(arg.TriggerPrice, 'f', -1, 64))
	}
	if arg.Advanced != "" {
		params.Set("advanced", arg.Advanced)
	}
	var resp PrivateTradeData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		editByLabel, params, &resp)
}

// SubmitCancel sends a request to cancel the order via its orderID
func (d *Deribit) SubmitCancel(ctx context.Context, orderID string) (*PrivateCancelData, error) {
	if orderID == "" {
		return nil, fmt.Errorf("%w, no order ID specified", errInvalidID)
	}
	params := url.Values{}
	params.Set("order_id", orderID)
	var resp PrivateCancelData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		submitCancel, params, &resp)
}

// SubmitCancelAll sends a request to cancel all user orders in all currencies and instruments
func (d *Deribit) SubmitCancelAll(ctx context.Context) (int64, error) {
	var resp int64
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		submitCancelAll, nil, &resp)
}

// SubmitCancelAllByCurrency sends a request to cancel all user orders for the specified currency
func (d *Deribit) SubmitCancelAllByCurrency(ctx context.Context, currency, kind, orderType string) (int64, error) {
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return 0, errInvalidCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	if kind != "" {
		params.Set("kind", kind)
	}
	if orderType != "" {
		params.Set("order_type", orderType)
	}
	var resp int64
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		submitCancelAllByCurrency, params, &resp)
}

// SubmitCancelAllByInstrument sends a request to cancel all user orders for the specified instrument
func (d *Deribit) SubmitCancelAllByInstrument(ctx context.Context, instrument, orderType string) (int64, error) {
	if instrument == "" {
		return 0, errInvalidInstrumentName
	}
	params := url.Values{}
	params.Set("instrument_name", instrument)
	if orderType != "" {
		params.Set("order_type", orderType)
	}
	var resp int64
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		submitCancelAllByInstrument, params, &resp)
}

// SubmitCancelByLabel sends a request to cancel all user orders for the specified label
func (d *Deribit) SubmitCancelByLabel(ctx context.Context, label, currency string) (int64, error) {
	params := url.Values{}
	params.Set("label", label)
	if currency != "" {
		params.Set("currency", currency)
	}
	var resp int64
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		submitCancelByLabel, params, &resp)
}

// SubmitClosePosition sends a request to cancel all user orders for the specified label
func (d *Deribit) SubmitClosePosition(ctx context.Context, instrument, orderType string, price float64) (*PrivateTradeData, error) {
	if instrument == "" {
		return nil, errInvalidInstrumentName
	}
	params := url.Values{}
	params.Set("instrument_name", instrument)
	if orderType != "" {
		params.Set("type", orderType)
	}
	params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	var resp PrivateTradeData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		submitClosePosition, params, &resp)
}

// GetMargins sends a request to fetch account margins data
func (d *Deribit) GetMargins(ctx context.Context, instrument string, amount, price float64) (*MarginsData, error) {
	if instrument == "" {
		return nil, errInvalidInstrumentName
	}
	if amount <= 0 {
		return nil, errInvalidAmount
	}
	if price <= 0 {
		return nil, errInvalidPrice
	}
	params := url.Values{}
	params.Set("instrument_name", instrument)
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	var resp MarginsData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		getMargins, params, &resp)
}

// GetMMPConfig sends a request to fetch the config for MMP of the requested currency
func (d *Deribit) GetMMPConfig(ctx context.Context, currency string) (*MMPConfigData, error) {
	currency = strings.ToUpper(currency)
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	var resp MMPConfigData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		getMMPConfig, params, &resp)
}

// GetOpenOrdersByCurrency sends a request to fetch open orders data sorted by requested params
func (d *Deribit) GetOpenOrdersByCurrency(ctx context.Context, currency, kind, orderType string) ([]OrderData, error) {
	currency = strings.ToUpper(currency)
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	if kind != "" {
		params.Set("kind", kind)
	}
	if orderType != "" {
		params.Set("type", orderType)
	}
	var resp []OrderData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		getOpenOrdersByCurrency, params, &resp)
}

// GetOpenOrdersByInstrument sends a request to fetch open orders data sorted by requested params
func (d *Deribit) GetOpenOrdersByInstrument(ctx context.Context, instrument, orderType string) ([]OrderData, error) {
	if instrument == "" {
		return nil, errInvalidInstrumentName
	}
	params := url.Values{}
	params.Set("instrument_name", instrument)
	if orderType != "" {
		params.Set("type", orderType)
	}
	var resp []OrderData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		getOpenOrdersByInstrument, params, &resp)
}

// GetOrderHistoryByCurrency sends a request to fetch order history according to given params and currency
func (d *Deribit) GetOrderHistoryByCurrency(ctx context.Context, currency, kind string, count, offset int64, includeOld, includeUnfilled bool) ([]OrderData, error) {
	currency = strings.ToUpper(currency)
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	if kind != "" {
		params.Set("kind", kind)
	}
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	if offset != 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	params.Set("include_old", strconv.FormatBool(includeOld))
	params.Set("include_unfilled", strconv.FormatBool(includeUnfilled))
	var resp []OrderData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		getOrderHistoryByCurrency, params, &resp)
}

// GetOrderHistoryByInstrument sends a request to fetch order history according to given params and instrument
func (d *Deribit) GetOrderHistoryByInstrument(ctx context.Context, instrument string, count, offset int64, includeOld, includeUnfilled bool) ([]OrderData, error) {
	if instrument == "" {
		return nil, errInvalidInstrumentName
	}
	params := url.Values{}
	params.Set("instrument_name", instrument)
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	if offset != 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	params.Set("include_old", strconv.FormatBool(includeOld))
	params.Set("include_unfilled", strconv.FormatBool(includeUnfilled))
	var resp []OrderData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		getOrderHistoryByInstrument, params, &resp)
}

// GetOrderMarginsByID sends a request to fetch order margins data according to their ids
func (d *Deribit) GetOrderMarginsByID(ctx context.Context, ids []string) ([]OrderData, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("%w, order ids cannot be empty", errInvalidID)
	}
	params := url.Values{}
	values, err := json.Marshal(ids)
	if err != nil {
		return nil, err
	}
	params.Set("ids", string(values))
	var resp []OrderData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		getOrderMarginByIDs, params, &resp)
}

// GetOrderState sends a request to fetch order state of the order id provided
func (d *Deribit) GetOrderState(ctx context.Context, orderID string) (*OrderData, error) {
	if orderID == "" {
		return nil, fmt.Errorf("%w, no order ID specified", errInvalidID)
	}
	params := url.Values{}
	params.Set("order_id", orderID)
	var resp OrderData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		getOrderState, params, &resp)
}

// GetTriggerOrderHistory sends a request to fetch order state of the order id provided
func (d *Deribit) GetTriggerOrderHistory(ctx context.Context, currency, instrumentName, continuation string, count int64) (*OrderData, error) {
	currency = strings.ToUpper(currency)
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	if instrumentName != "" {
		params.Set("instrument_name", instrumentName)
	}
	if continuation != "" {
		params.Set("continuation", continuation)
	}
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	var resp OrderData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		getTriggerOrderHistory, params, &resp)
}

// GetUserTradesByCurrency sends a request to fetch user trades sorted by currency
func (d *Deribit) GetUserTradesByCurrency(ctx context.Context, currency, kind, startID, endID, sorting string, count int64, includeOld bool) (*UserTradesData, error) {
	currency = strings.ToUpper(currency)
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidCurrency
	}
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
	params.Set("include_old", strconv.FormatBool(includeOld))
	var resp UserTradesData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		getUserTradesByCurrency, params, &resp)
}

// [[ TODO ]]

// GetUserTradesByCurrencyAndTime sends a request to fetch user trades sorted by currency and time
func (d *Deribit) GetUserTradesByCurrencyAndTime(ctx context.Context, currency, kind, sorting string, count int64, includeOld bool, startTime, endTime time.Time) (*UserTradesData, error) {
	currency = strings.ToUpper(currency)
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidCurrency
	}
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
	if !startTime.IsZero() {
		params.Set("start_timestamp", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("end_timestamp", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp UserTradesData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		getUserTradesByCurrencyAndTime, params, &resp)
}

// GetUserTradesByInstrument sends a request to fetch user trades sorted by instrument
func (d *Deribit) GetUserTradesByInstrument(ctx context.Context, instrument, sorting string, startSeq, endSeq, count int64, includeOld bool) (*UserTradesData, error) {
	if instrument == "" {
		return nil, errInvalidInstrumentName
	}
	params := url.Values{}
	params.Set("instrument_name", instrument)
	if startSeq != 0 {
		params.Set("start_seq", strconv.FormatInt(startSeq, 10))
	}
	if endSeq != 0 {
		params.Set("end_seq", strconv.FormatInt(endSeq, 10))
	}
	if sorting != "" {
		params.Set("sorting", sorting)
	}
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	params.Set("include_old", strconv.FormatBool(includeOld))
	var resp UserTradesData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		getUserTradesByInstrument, params, &resp)
}

// GetUserTradesByInstrumentAndTime sends a request to fetch user trades sorted by instrument and time
func (d *Deribit) GetUserTradesByInstrumentAndTime(ctx context.Context, instrument, sorting string, count int64, includeOld bool, startTime, endTime time.Time) (*UserTradesData, error) {
	if instrument == "" {
		return nil, errInvalidInstrumentName
	}
	params := url.Values{}
	params.Set("instrument_name", instrument)
	if sorting != "" {
		params.Set("sorting", sorting)
	}
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	if startTime.IsZero() {
		return nil, fmt.Errorf("%w, start time can not be zero", errInvalidTimestamp)
	} else if startTime.After(endTime) {
		return nil, errStartTimeCannotBeAfterEndTime
	}
	params.Set("start_timestamp", strconv.FormatInt(startTime.UnixMilli(), 10))
	if endTime.IsZero() {
		endTime = time.Now()
	}
	params.Set("end_timestamp", strconv.FormatInt(endTime.UnixMilli(), 10))
	var resp UserTradesData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		getUserTradesByInstrumentAndTime, params, &resp)
}

// GetUserTradesByOrder sends a request to get user trades fetched by orderID
func (d *Deribit) GetUserTradesByOrder(ctx context.Context, orderID, sorting string) (*UserTradesData, error) {
	if orderID == "" {
		return nil, fmt.Errorf("%w, no order ID specified", errInvalidID)
	}
	params := url.Values{}
	params.Set("order_id", orderID)
	if sorting != "" {
		params.Set("sorting", sorting)
	}
	var resp UserTradesData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		getUserTradesByOrder, params, &resp)
}

// ResetMMP sends a request to reset MMP for a currency provided
func (d *Deribit) ResetMMP(ctx context.Context, currency string) (string, error) {
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return "", errInvalidCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	var resp string
	err := d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		resetMMP, params, &resp)
	if err != nil {
		return "", err
	}
	if resp != "ok" {
		return "", fmt.Errorf("mmp could not be reset for %v", currency)
	}
	return resp, nil
}

// SendRFQ sends RFQ on a given instrument.
func (d *Deribit) SendRFQ(ctx context.Context, instrumentName string, amount float64, side order.Side) (string, error) {
	if instrumentName == "" {
		return "", errInvalidInstrumentName
	}
	params := url.Values{}
	params.Set("instrument_name", instrumentName)
	if amount > 0 {
		params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	}
	if side != order.UnknownSide {
		params.Set("side", side.String())
	}
	var resp string
	err := d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet, sendRFQ, params, &resp)
	if err != nil {
		return "", err
	}
	if resp != "ok" {
		return "", fmt.Errorf("rfq couldn't send for %v", instrumentName)
	}
	return resp, nil
}

// SetMMPConfig sends a request to set the given parameter values to the mmp config for the provided currency
func (d *Deribit) SetMMPConfig(ctx context.Context, currency string, interval, frozenTime int64, quantityLimit, deltaLimit float64) (string, error) {
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return "", errInvalidCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	var resp string
	err := d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		resetMMP, params, &resp)
	if err != nil {
		return "", err
	}
	if resp != "ok" {
		return "", fmt.Errorf("mmp data could not be set for %v", currency)
	}
	return resp, nil
}

// GetSettlementHistoryByInstrument sends a request to fetch settlement history data sorted by instrument
func (d *Deribit) GetSettlementHistoryByInstrument(ctx context.Context, instrument, settlementType, continuation string, count int64, searchStartTimeStamp time.Time) (*PrivateSettlementsHistoryData, error) {
	if instrument == "" {
		return nil, errInvalidInstrumentName
	}
	params := url.Values{}
	params.Set("instrument_name", instrument)
	if settlementType != "" {
		params.Set("settlement_type", settlementType)
	}
	if continuation != "" {
		params.Set("continuation", continuation)
	}
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	if !searchStartTimeStamp.IsZero() {
		params.Set("search_start_timestamp", strconv.FormatInt(searchStartTimeStamp.UnixMilli(), 10))
	}
	var resp PrivateSettlementsHistoryData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		getSettlementHistoryByInstrument, params, &resp)
}

// GetSettlementHistoryByCurency sends a request to fetch settlement history data sorted by currency
func (d *Deribit) GetSettlementHistoryByCurency(ctx context.Context, currency, settlementType, continuation string, count int64, searchStartTimeStamp time.Time) (*PrivateSettlementsHistoryData, error) {
	currency = strings.ToUpper(currency)
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	if settlementType != "" {
		params.Set("settlement_type", settlementType)
	}
	if continuation != "" {
		params.Set("continuation", continuation)
	}
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	if !searchStartTimeStamp.IsZero() {
		params.Set("search_start_timestamp", strconv.FormatInt(searchStartTimeStamp.UnixMilli(), 10))
	}
	var resp PrivateSettlementsHistoryData
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet,
		getSettlementHistoryByCurrency, params, &resp)
}

// SendHTTPAuthRequest sends an authenticated request to deribit api
func (d *Deribit) SendHTTPAuthRequest(ctx context.Context, ep exchange.URL, method, path string, data url.Values, result interface{}) error {
	endpoint, err := d.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	reqDataStr := method + "\n" + deribitAPIVersion + common.EncodeURLValues(path, data) + "\n" + "" + "\n"
	n := d.Requester.GetNonce(true)
	strTS := strconv.FormatInt(time.Now().UnixMilli(), 10)
	str2Sign := fmt.Sprintf("%s\n%s\n%s", strTS,
		n, reqDataStr)
	creds, err := d.GetCredentials(ctx)
	if err != nil {
		return err
	}
	hmac, err := crypto.GetHMAC(crypto.HashSHA256,
		[]byte(str2Sign),
		[]byte(creds.Secret))

	if err != nil {
		return err
	}
	headers := make(map[string]string)
	headerString := fmt.Sprintf("deri-hmac-sha256 id=%s,ts=%s,sig=%s,nonce=%s",
		creds.Key,
		strTS,
		crypto.HexEncodeToString(hmac),
		n)
	headers["Authorization"] = headerString
	headers["Content-Type"] = "application/json"
	var tempData struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      int64           `json:"id"`
		Data    json.RawMessage `json:"result"`
	}
	item := &request.Item{
		Method:        method,
		Path:          endpoint + deribitAPIVersion + common.EncodeURLValues(path, data),
		Headers:       headers,
		Body:          nil,
		Result:        &tempData,
		AuthRequest:   true,
		Verbose:       d.Verbose,
		HTTPDebugging: d.HTTPDebugging,
		HTTPRecording: d.HTTPRecording,
	}
	err = d.SendPayload(context.Background(), request.Unset, func() (*request.Item, error) {
		return item, nil
	})
	if err != nil {
		return err
	}
	return json.Unmarshal(tempData.Data, result)
}

// Combo Books endpoints'

// GetComboIDS Retrieves available combos.
// This method can be used to get the list of all combos, or only the list of combos in the given state.
func (d *Deribit) GetComboIDS(ctx context.Context, currency, state string) ([]string, error) {
	currency = strings.ToUpper(currency)
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	if state != "" {
		params.Set("state", state)
	}
	var resp []string
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues(getComboIDS, params), &resp)
}

// GetComboDetails retrieves information about a combo
func (d *Deribit) GetComboDetails(ctx context.Context, comboID string) (*ComboDetail, error) {
	if comboID == "" {
		return nil, errInvalidComboID
	}
	params := url.Values{}
	params.Set("combo_id", comboID)
	var resp ComboDetail
	return &resp, d.SendHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues(getComboDetails, params), &resp)
}

// GetCombos retrieves information about active combos
func (d *Deribit) GetCombos(ctx context.Context, currency string) ([]ComboDetail, error) {
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, fmt.Errorf("%w, only BTC, ETH, SOL, and USDC are supported", errInvalidCurrency)
	}
	params := url.Values{}
	params.Set("currency", currency)
	var resp []ComboDetail
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues(getCombos, params), &resp)
}

// CreateCombo verifies and creates a combo book or returns an existing combo matching given trades
func (d *Deribit) CreateCombo(ctx context.Context, args []ComboParam) (*ComboDetail, error) {
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
	argsByte, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("trades", string(argsByte))
	var resp ComboDetail
	return &resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet, createCombos, params, &resp)
}

// ExecuteBlockTrade executes a block trade request
// The whole request have to be exact the same as in private/verify_block_trade, only role field should be set appropriately - it basically means that both sides have to agree on the same timestamp, nonce, trades fields and server will assure that role field is different between sides (each party accepted own role).
// Using the same timestamp and nonce by both sides in private/verify_block_trade assures that even if unintentionally both sides execute given block trade with valid counterparty_signature, the given block trade will be executed only once
func (d *Deribit) ExecuteBlockTrade(ctx context.Context, timestampMS time.Time, nonce, role, currency string, trades []BlockTradeParam) ([]BlockTradeResponse, error) {
	params := url.Values{}
	if nonce == "" {
		return nil, errMissingNonce
	}
	params.Set("nonce", nonce)
	if role != "maker" && role != "taker" {
		return nil, errInvalidTradeRole
	}
	params.Set("role", role)
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
	signature, err := d.VerifyBlockTrade(ctx, timestampMS, nonce, role, currency, trades)
	if err != nil {
		return nil, err
	}
	params.Set("counterparty_signature", signature)
	values, err := json.Marshal(trades)
	if err != nil {
		return nil, err
	}
	params.Set("trades", string(values))
	if timestampMS.IsZero() {
		return nil, errors.New("zero timestamp")
	}
	params.Set("timestamp", strconv.FormatInt(timestampMS.UnixMilli(), 10))
	if currency != "" {
		params.Set("currency", currency)
	}
	var resp []BlockTradeResponse
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet, executeBlockTrades, params, &resp)
}

// VerifyBlockTrade verifies and creates block trade signature
func (d *Deribit) VerifyBlockTrade(ctx context.Context, timestampMS time.Time, nonce, role, currency string, trades []BlockTradeParam) (string, error) {
	params := url.Values{}
	if nonce == "" {
		return "", errMissingNonce
	}
	params.Set("nonce", nonce)
	if role != "maker" && role != "taker" {
		return "", errInvalidTradeRole
	}
	params.Set("role", role)
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
	values, err := json.Marshal(trades)
	if err != nil {
		return "", err
	}
	params.Set("trades", string(values))
	if timestampMS.IsZero() {
		return "", errors.New("zero timestamp")
	}
	params.Set("timestamp", strconv.FormatInt(timestampMS.UnixMilli(), 10))
	if currency != "" {
		params.Set("currency", currency)
	}
	resp := &struct {
		Signature string `json:"signature"`
	}{}
	return resp.Signature, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet, verifyBlockTrades, params, resp)
}

// GetUserBlocTrade returns information about users block trade
func (d *Deribit) GetUserBlocTrade(ctx context.Context, id string) ([]BlockTradeData, error) {
	if id == "" {
		return nil, errors.New("missing block trade id")
	}
	params := url.Values{}
	params.Set("id", id)
	var resp []BlockTradeData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet, getBlockTrades, params, &resp)
}

// GetLastBlockTradesByCurrency returns list of last users block trades
func (d *Deribit) GetLastBlockTradesByCurrency(ctx context.Context, currency, startID, endID string, count int64) ([]BlockTradeData, error) {
	currency = strings.ToUpper(currency)
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	if startID != "" {
		params.Set("start_id", startID)
	}
	if endID != "" {
		params.Set("end_id", endID)
	}
	if count > 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	var resp []BlockTradeData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet, getLastBlockTradesByCurrency, params, &resp)
}

// MovePositions moves positions from source subaccount to target subaccount
func (d *Deribit) MovePositions(ctx context.Context, currency string, sourceSubAccountUID, targetSubAccountUID int64, trades []BlockTradeParam) ([]BlockTradeMoveResponse, error) {
	currency = strings.ToUpper(currency)
	if currency != currencyBTC && currency != currencyETH && currency != currencySOL && currency != currencyUSDC {
		return nil, errInvalidCurrency
	}
	params := url.Values{}
	params.Set("currency", currency)
	if sourceSubAccountUID == 0 {
		return nil, errors.New("missing source subaccount id")
	}
	params.Set("source_uid", strconv.FormatInt(sourceSubAccountUID, 10))
	if targetSubAccountUID == 0 {
		return nil, errors.New("missing target subaccount id")
	}
	params.Set("target_uid", strconv.FormatInt(targetSubAccountUID, 10))
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
	values, err := json.Marshal(trades)
	if err != nil {
		return nil, err
	}
	params.Set("trades", string(values))
	var resp []BlockTradeMoveResponse
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, http.MethodGet, movePositions, params, &resp)
}

// GetAssetKind returns the asset type (kind) string representation.
func (d *Deribit) GetAssetKind(assetType asset.Item) string {
	switch assetType {
	case asset.Options:
		return "option"
	case asset.Futures:
		return "future"
	case asset.FutureCombo, asset.OptionCombo, asset.Combo:
		return assetType.String()
	default:
		return "any"
	}
}

// StringToAssetKind returns the asset type (kind) from a string representation.
func (d *Deribit) StringToAssetKind(assetType string) (asset.Item, error) {
	assetType = strings.ToLower(assetType)
	switch assetType {
	case "option":
		return asset.Options, nil
	case "future":
		return asset.Futures, nil
	case "any":
		return asset.Empty, nil
	default:
		return asset.New(assetType)
	}
}

func (d *Deribit) getFirstAssetTradablePair(t *testing.T, _ asset.Item) (currency.Pair, error) {
	t.Helper()
	instruments, err := d.FetchTradablePairs(context.Background(), asset.Futures)
	if err != nil {
		t.Skip(err)
	}
	if len(instruments) < 1 {
		t.Skip("no enough instrument found")
	}
	cp, err := currency.NewPairFromString(instruments[0])
	if err != nil {
		return currency.EMPTYPAIR, err
	}
	cp = cp.Upper()
	cp.Delimiter = currency.DashDelimiter
	return cp, nil
}
