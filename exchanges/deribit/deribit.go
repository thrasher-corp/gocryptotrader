package deribit

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/nonce"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Deribit is the overarching type across this package
type Deribit struct {
	exchange.Base
}

const (
	deribitAPIVersion = "/api/v2"
	tradeBaseURL      = "https://www.deribit.com/"
	tradeSpot         = "spot/"
	tradeFutures      = "futures/"
	tradeOptions      = "options/"
	tradeFuturesCombo = "futures-spreads/"
	tradeOptionsCombo = "combos/"

	perpString = "PERPETUAL"

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
	getVolatilityIndex               = "public/get_volatility_index_data"
	getTicker                        = "public/ticker"

	// Authenticated endpoints

	// wallet eps
	cancelTransferByID               = "private/cancel_transfer_by_id"
	cancelWithdrawal                 = "private/cancel_withdrawal"
	createDepositAddress             = "private/create_deposit_address"
	getCurrentDepositAddress         = "private/get_current_deposit_address"
	getDeposits                      = "private/get_deposits"
	getTransfers                     = "private/get_transfers"
	getWithdrawals                   = "private/get_withdrawals"
	submitTransferBetweenSubAccounts = "private/submit_transfer_between_subaccounts"
	submitTransferToSubaccount       = "private/submit_transfer_to_subaccount"
	submitTransferToUser             = "private/submit_transfer_to_user"
	submitWithdraw                   = "private/withdraw"

	// trading endpoints
	submitBuy                        = "private/buy"
	submitSell                       = "private/sell"
	submitEdit                       = "private/edit"
	editByLabel                      = "private/edit_by_label"
	submitCancel                     = "private/cancel"
	submitCancelAll                  = "private/cancel_all"
	submitCancelAllByCurrency        = "private/cancel_all_by_currency"
	submitCancelAllByKind            = "private/cancel_all_by_kind_or_type"
	submitCancelAllByInstrument      = "private/cancel_all_by_instrument"
	submitCancelByLabel              = "private/cancel_by_label"
	submitCancelQuotes               = "private/cancel_quotes"
	submitClosePosition              = "private/close_position"
	getMargins                       = "private/get_margins"
	getMMPConfig                     = "private/get_mmp_config"
	getOpenOrdersByCurrency          = "private/get_open_orders_by_currency"
	getOpenOrdersByLabel             = "private/get_open_orders_by_label"
	getOpenOrdersByInstrument        = "private/get_open_orders_by_instrument"
	getOrderHistoryByCurrency        = "private/get_order_history_by_currency"
	getOrderHistoryByInstrument      = "private/get_order_history_by_instrument"
	getOrderMarginByIDs              = "private/get_order_margin_by_ids"
	getOrderState                    = "private/get_order_state"
	getOrderStateByLabel             = "private/get_order_state_by_label"
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
	changeAPIKeyName                  = "private/change_api_key_name"
	changeMarginModel                 = "private/change_margin_model"
	changeScopeInAPIKey               = "private/change_scope_in_api_key"
	changeSubAccountName              = "private/change_subaccount_name"
	createAPIKey                      = "private/create_api_key"
	createSubAccount                  = "private/create_subaccount"
	disableAPIKey                     = "private/disable_api_key"
	editAPIKey                        = "private/edit_api_key"
	enableAffiliateProgram            = "private/enable_affiliate_program"
	enableAPIKey                      = "private/enable_api_key"
	getAccessLog                      = "private/get_access_log"
	getAccountSummary                 = "private/get_account_summary"
	getAffiliateProgramInfo           = "private/get_affiliate_program_info"
	getEmailLanguage                  = "private/get_email_language"
	getNewAnnouncements               = "private/get_new_announcements"
	getPosition                       = "private/get_position"
	getPositions                      = "private/get_positions"
	getSubAccounts                    = "private/get_subaccounts"
	getSubAccountDetails              = "private/get_subaccounts_details"
	getTransactionLog                 = "private/get_transaction_log"
	getUserLocks                      = "private/get_user_locks"
	listAPIKeys                       = "private/list_api_keys"
	listCustodyAccounts               = "private/list_custody_accounts"
	removeAPIKey                      = "private/remove_api_key"
	removeSubAccount                  = "private/remove_subaccount"
	resetAPIKey                       = "private/reset_api_key"
	setAnnouncementAsRead             = "private/set_announcement_as_read"
	setEmailForSubAccount             = "private/set_email_for_subaccount"
	setEmailLanguage                  = "private/set_email_language"
	setSelfTradingConfig              = "private/set_self_trading_config"
	toggleNotificationsFromSubAccount = "private/toggle_notifications_from_subaccount"
	togglePortfolioMargining          = "private/toggle_portfolio_margining"
	toggleSubAccountLogin             = "private/toggle_subaccount_login"

	// Combo Books Endpoints
	getComboDetails = "public/get_combo_details"
	getComboIDs     = "public/get_combo_ids"
	getCombos       = "public/get_combos"
	createCombos    = "private/create_combo"

	// Block Trades Endpoints
	executeBlockTrades             = "private/execute_block_trade"
	getBlockTrades                 = "private/get_block_trade"
	getLastBlockTradesByCurrency   = "private/get_last_block_trades_by_currency"
	invalidateBlockTradesSignature = "private/invalidate_block_trade_signature"
	movePositions                  = "private/move_positions"
	simulateBlockPosition          = "private/simulate_block_trade"
	verifyBlockTrades              = "private/verify_block_trade"

	// roles

	roleMaker = "maker"
	roleTaker = "taker"
)

// GetBookSummaryByCurrency gets book summary data for currency requested
func (d *Deribit) GetBookSummaryByCurrency(ctx context.Context, ccy currency.Code, kind string) ([]BookSummaryData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if kind != "" {
		params.Set("kind", kind)
	}
	var resp []BookSummaryData
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL,
		common.EncodeURLValues(getBookByCurrency, params), &resp)
}

// GetBookSummaryByInstrument gets book summary data for instrument requested
func (d *Deribit) GetBookSummaryByInstrument(ctx context.Context, instrument string) ([]BookSummaryData, error) {
	params, err := checkInstrument(instrument)
	if err != nil {
		return nil, err
	}
	var resp []BookSummaryData
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL,
		common.EncodeURLValues(getBookByInstrument, params), &resp)
}

// GetContractSize gets contract size for instrument requested
func (d *Deribit) GetContractSize(ctx context.Context, instrument string) (*ContractSizeData, error) {
	params, err := checkInstrument(instrument)
	if err != nil {
		return nil, err
	}
	var resp *ContractSizeData
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL,
		common.EncodeURLValues(getContractSize, params), &resp)
}

// GetCurrencies gets all cryptocurrencies supported by the API
func (d *Deribit) GetCurrencies(ctx context.Context) ([]CurrencyData, error) {
	var resp []CurrencyData
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL, getCurrencies, &resp)
}

// GetDeliveryPrices gets all delivery prices for the given inde name
func (d *Deribit) GetDeliveryPrices(ctx context.Context, indexName string, offset, count int64) (*IndexDeliveryPrice, error) {
	if indexName == "" {
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
	var resp *IndexDeliveryPrice
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL, common.EncodeURLValues(getDeliveryPrices, params), &resp)
}

// GetFundingChartData gets funding chart data for the requested instrument and time length
// supported lengths: 8h, 24h, 1m <-(1month)
func (d *Deribit) GetFundingChartData(ctx context.Context, instrument, length string) (*FundingChartData, error) {
	params, err := checkInstrument(instrument)
	if err != nil {
		return nil, err
	}
	params.Set("length", length)
	var resp *FundingChartData
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL,
		common.EncodeURLValues(getFundingChartData, params), &resp)
}

// GetFundingRateHistory retrieves hourly historical interest rate for requested PERPETUAL instrument.
func (d *Deribit) GetFundingRateHistory(ctx context.Context, instrumentName string, startTime, endTime time.Time) ([]FundingRateHistory, error) {
	params, err := checkInstrument(instrumentName)
	if err != nil {
		return nil, err
	}
	err = common.StartEndTimeCheck(startTime, endTime)
	if err != nil {
		return nil, err
	}
	params.Set("start_timestamp", strconv.FormatInt(startTime.UnixMilli(), 10))
	params.Set("end_timestamp", strconv.FormatInt(endTime.UnixMilli(), 10))
	var resp []FundingRateHistory
	return resp, d.SendHTTPRequest(ctx, exchange.RestSpot, nonMatchingEPL, common.EncodeURLValues(getFundingRateHistory, params), &resp)
}

// GetFundingRateValue gets funding rate value data.
func (d *Deribit) GetFundingRateValue(ctx context.Context, instrument string, startTime, endTime time.Time) (float64, error) {
	params, err := checkInstrument(instrument)
	if err != nil {
		return 0, err
	}
	err = common.StartEndTimeCheck(startTime, endTime)
	if err != nil {
		return 0, err
	}
	params.Set("start_timestamp", strconv.FormatInt(startTime.UnixMilli(), 10))
	params.Set("end_timestamp", strconv.FormatInt(endTime.UnixMilli(), 10))
	var resp float64
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL,
		common.EncodeURLValues(getFundingRateValue, params), &resp)
}

// GetHistoricalVolatility gets historical volatility data
func (d *Deribit) GetHistoricalVolatility(ctx context.Context, ccy currency.Code) ([]HistoricalVolatilityData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var data []HistoricalVolatilityData
	return data, d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL, common.EncodeURLValues(getHistoricalVolatility, params), &data)
}

// GetCurrencyIndexPrice retrieves the current index price for the instruments, for the selected currency.
func (d *Deribit) GetCurrencyIndexPrice(ctx context.Context, ccy currency.Code) (*IndexPrice, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var resp *IndexPrice
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL, common.EncodeURLValues(getCurrencyIndexPrice, params), &resp)
}

// GetIndexPrice gets price data for the requested index
func (d *Deribit) GetIndexPrice(ctx context.Context, index string) (*IndexPriceData, error) {
	if index == "" {
		return nil, errUnsupportedIndexName
	}
	params := url.Values{}
	params.Set("index_name", index)
	var resp *IndexPriceData
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL,
		common.EncodeURLValues(getIndexPrice, params), &resp)
}

// GetIndexPriceNames gets names of indexes
func (d *Deribit) GetIndexPriceNames(ctx context.Context) ([]string, error) {
	var resp []string
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL, getIndexPriceNames, &resp)
}

// GetInstrument retrieves instrument detail
func (d *Deribit) GetInstrument(ctx context.Context, instrument string) (*InstrumentData, error) {
	params, err := checkInstrument(instrument)
	if err != nil {
		return nil, err
	}
	var resp *InstrumentData
	return resp, d.SendHTTPRequest(ctx, exchange.RestSpot, nonMatchingEPL,
		common.EncodeURLValues(getInstrument, params), &resp)
}

// GetInstruments gets data for all available instruments
func (d *Deribit) GetInstruments(ctx context.Context, ccy currency.Code, kind string, expired bool) ([]*InstrumentData, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if kind != "" {
		params.Set("kind", kind)
	}
	if expired {
		params.Set("expired", "true")
	}
	var resp []*InstrumentData
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL, common.EncodeURLValues(getInstruments, params), &resp)
}

// GetLastSettlementsByCurrency gets last settlement data by currency
func (d *Deribit) GetLastSettlementsByCurrency(ctx context.Context, ccy currency.Code, settlementType, continuation string, count int64, searchStartTime time.Time) (*SettlementsData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if settlementType != "" {
		params.Set("type", settlementType)
	}
	if continuation != "" {
		params.Set("continuation", continuation)
	}
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	if !searchStartTime.IsZero() {
		params.Set("search_start_timestamp", strconv.FormatInt(searchStartTime.UnixMilli(), 10))
	}
	var resp *SettlementsData
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL,
		common.EncodeURLValues(getLastSettlementsByCurrency, params), &resp)
}

// GetLastSettlementsByInstrument gets last settlement data for requested instrument
func (d *Deribit) GetLastSettlementsByInstrument(ctx context.Context, instrument, settlementType, continuation string, count int64, startTime time.Time) (*SettlementsData, error) {
	params, err := checkInstrument(instrument)
	if err != nil {
		return nil, err
	}
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
	var resp *SettlementsData
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL,
		common.EncodeURLValues(getLastSettlementsByInstrument, params), &resp)
}

// GetLastTradesByCurrency gets last trades for requested currency
func (d *Deribit) GetLastTradesByCurrency(ctx context.Context, ccy currency.Code, kind, startID, endID, sorting string, count int64, includeOld bool) (*PublicTradesData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
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
	if includeOld {
		params.Set("include_old", "true")
	}
	var resp *PublicTradesData
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL,
		common.EncodeURLValues(getLastTradesByCurrency, params), &resp)
}

// GetLastTradesByCurrencyAndTime gets last trades for requested currency and time intervals
func (d *Deribit) GetLastTradesByCurrencyAndTime(ctx context.Context, ccy currency.Code, kind, sorting string, count int64, startTime, endTime time.Time) (*PublicTradesData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if kind != "" {
		params.Set("kind", kind)
	}
	if sorting != "" {
		params.Set("sorting", sorting)
	}
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	err := common.StartEndTimeCheck(startTime, endTime)
	if err != nil {
		return nil, err
	}
	params.Set("start_timestamp", strconv.FormatInt(startTime.UnixMilli(), 10))
	params.Set("end_timestamp", strconv.FormatInt(endTime.UnixMilli(), 10))
	var resp *PublicTradesData
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL,
		common.EncodeURLValues(getLastTradesByCurrencyAndTime, params), &resp)
}

// GetLastTradesByInstrument gets last trades for requested instrument requested
func (d *Deribit) GetLastTradesByInstrument(ctx context.Context, instrument, startSeq, endSeq, sorting string, count int64, includeOld bool) (*PublicTradesData, error) {
	params, err := checkInstrument(instrument)
	if err != nil {
		return nil, err
	}
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
	if includeOld {
		params.Set("include_old", "true")
	}
	var resp *PublicTradesData
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL, common.EncodeURLValues(getLastTradesByInstrument, params), &resp)
}

// GetLastTradesByInstrumentAndTime gets last trades for requested instrument requested and time intervals
func (d *Deribit) GetLastTradesByInstrumentAndTime(ctx context.Context, instrument, sorting string, count int64, startTime, endTime time.Time) (*PublicTradesData, error) {
	params, err := checkInstrument(instrument)
	if err != nil {
		return nil, err
	}
	if sorting != "" {
		params.Set("sorting", sorting)
	}
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	err = common.StartEndTimeCheck(startTime, endTime)
	if err != nil {
		return nil, err
	}
	params.Set("start_timestamp", strconv.FormatInt(startTime.UnixMilli(), 10))
	params.Set("end_timestamp", strconv.FormatInt(endTime.UnixMilli(), 10))
	var resp *PublicTradesData
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL,
		common.EncodeURLValues(getLastTradesByInstrumentAndTime, params), &resp)
}

// GetMarkPriceHistory gets data for mark price history
func (d *Deribit) GetMarkPriceHistory(ctx context.Context, instrument string, startTime, endTime time.Time) ([]MarkPriceHistory, error) {
	params, err := checkInstrument(instrument)
	if err != nil {
		return nil, err
	}
	err = common.StartEndTimeCheck(startTime, endTime)
	if err != nil {
		return nil, err
	}
	params.Set("start_timestamp", strconv.FormatInt(startTime.UnixMilli(), 10))
	params.Set("end_timestamp", strconv.FormatInt(endTime.UnixMilli(), 10))
	var resp []MarkPriceHistory
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL,
		common.EncodeURLValues(getMarkPriceHistory, params), &resp)
}

func checkInstrument(instrumentName string) (url.Values, error) {
	if instrumentName == "" {
		return nil, fmt.Errorf("%w, instrument_name is missing", errInvalidInstrumentName)
	}
	params := url.Values{}
	params.Set("instrument_name", instrumentName)
	return params, nil
}

// GetOrderbook gets data orderbook of requested instrument
func (d *Deribit) GetOrderbook(ctx context.Context, instrument string, depth int64) (*Orderbook, error) {
	params, err := checkInstrument(instrument)
	if err != nil {
		return nil, err
	}
	if depth != 0 {
		params.Set("depth", strconv.FormatInt(depth, 10))
	}
	var resp *Orderbook
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL,
		common.EncodeURLValues(getOrderbook, params), &resp)
}

// GetOrderbookByInstrumentID retrieves orderbook by instrument ID
func (d *Deribit) GetOrderbookByInstrumentID(ctx context.Context, instrumentID int64, depth float64) (*Orderbook, error) {
	if instrumentID == 0 {
		return nil, errInvalidInstrumentID
	}
	params := url.Values{}
	params.Set("instrument_id", strconv.FormatInt(instrumentID, 10))
	if depth != 0 {
		params.Set("depth", strconv.FormatFloat(depth, 'f', -1, 64))
	}
	var resp *Orderbook
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL, common.EncodeURLValues(getOrderbookByInstrumentID, params), &resp)
}

// GetSupportedIndexNames retrieves the identifiers of all supported Price Indexes
// 'type' represents Type of a cryptocurrency price index. possible 'all', 'spot', 'derivative'
func (d *Deribit) GetSupportedIndexNames(ctx context.Context, priceIndexType string) ([]string, error) {
	params := url.Values{}
	if priceIndexType != "" {
		params.Set("type", priceIndexType)
	}
	var resp []string
	return resp, d.SendHTTPRequest(ctx, exchange.RestSpot, nonMatchingEPL, common.EncodeURLValues("public/get_supported_index_names", params), &resp)
}

// GetRequestForQuote retrieves RFQ information.
func (d *Deribit) GetRequestForQuote(ctx context.Context, ccy currency.Code, kind string) ([]RequestForQuote, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if kind != "" {
		params.Set("kind", kind)
	}
	var resp []RequestForQuote
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL, common.EncodeURLValues(getRFQ, params), &resp)
}

// GetTradeVolumes gets trade volumes' data of all instruments
func (d *Deribit) GetTradeVolumes(ctx context.Context, extended bool) ([]TradeVolumesData, error) {
	params := url.Values{}
	if extended {
		params.Set("extended", "true")
	}
	var resp []TradeVolumesData
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL,
		common.EncodeURLValues(getTradeVolumes, params), &resp)
}

// GetTradingViewChart gets volatility index data for the requested instrument
func (d *Deribit) GetTradingViewChart(ctx context.Context, instrument, resolution string, startTime, endTime time.Time) (*TVChartData, error) {
	params, err := checkInstrument(instrument)
	if err != nil {
		return nil, err
	}
	err = common.StartEndTimeCheck(startTime, endTime)
	if err != nil {
		return nil, err
	}
	if resolution == "" {
		return nil, errResolutionNotSet
	}
	params.Set("start_timestamp", strconv.FormatInt(startTime.UnixMilli(), 10))
	params.Set("end_timestamp", strconv.FormatInt(endTime.UnixMilli(), 10))
	params.Set("resolution", resolution)
	var resp *TVChartData
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL,
		common.EncodeURLValues(getTradingViewChartData, params), &resp)
}

// GetResolutionFromInterval returns the string representation of intervals given kline.Interval instance.
func (d *Deribit) GetResolutionFromInterval(interval kline.Interval) (string, error) {
	switch interval {
	case kline.HundredMilliseconds:
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
	case kline.Raw:
		return interval.Word(), nil
	default:
		return "", kline.ErrUnsupportedInterval
	}
}

// GetVolatilityIndex gets volatility index for the requested currency
func (d *Deribit) GetVolatilityIndex(ctx context.Context, ccy currency.Code, resolution string, startTime, endTime time.Time) ([]VolatilityIndexData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	err := common.StartEndTimeCheck(startTime, endTime)
	if err != nil {
		return nil, err
	}
	if resolution == "" {
		return nil, errResolutionNotSet
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	params.Set("start_timestamp", strconv.FormatInt(startTime.UnixMilli(), 10))
	params.Set("end_timestamp", strconv.FormatInt(endTime.UnixMilli(), 10))
	params.Set("resolution", resolution)
	var resp VolatilityIndexRawData
	err = d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL,
		common.EncodeURLValues(getVolatilityIndex, params), &resp)
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

// GetPublicTicker gets public ticker data of the instrument requested
func (d *Deribit) GetPublicTicker(ctx context.Context, instrument string) (*TickerData, error) {
	params, err := checkInstrument(instrument)
	if err != nil {
		return nil, err
	}
	var resp *TickerData
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL, common.EncodeURLValues(getTicker, params), &resp)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (d *Deribit) SendHTTPRequest(ctx context.Context, ep exchange.URL, epl request.EndpointLimit, path string, result any) error {
	endpoint, err := d.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	data := &struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int64  `json:"id"`
		Data    any    `json:"result"`
	}{
		Data: result,
	}
	return d.SendPayload(ctx, epl, func() (*request.Item, error) {
		return &request.Item{
			Method:        http.MethodGet,
			Path:          endpoint + deribitAPIVersion + "/" + path,
			Result:        data,
			Verbose:       d.Verbose,
			HTTPDebugging: d.HTTPDebugging,
			HTTPRecording: d.HTTPRecording,
		}, nil
	}, request.UnauthenticatedRequest)
}

// GetAccountSummary gets account summary data for the requested instrument
func (d *Deribit) GetAccountSummary(ctx context.Context, ccy currency.Code, extended bool) (*AccountSummaryData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if extended {
		params.Set("extended", "true")
	}
	var resp *AccountSummaryData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, getAccountSummary, params, &resp)
}

// CancelWithdrawal cancels withdrawal request for a given currency by its id
func (d *Deribit) CancelWithdrawal(ctx context.Context, ccy currency.Code, id int64) (*CancelWithdrawalData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if id <= 0 {
		return nil, fmt.Errorf("%w, withdrawal id has to be positive integer", errInvalidID)
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	params.Set("id", strconv.FormatInt(id, 10))
	var resp *CancelWithdrawalData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		cancelWithdrawal, params, &resp)
}

// CancelTransferByID cancels transfer by ID through the websocket connection.
func (d *Deribit) CancelTransferByID(ctx context.Context, ccy currency.Code, tfa string, id int64) (*AccountSummaryData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if id <= 0 {
		return nil, fmt.Errorf("%w, transfer id has to be positive integer", errInvalidID)
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if tfa != "" {
		params.Set("tfa", tfa)
	}
	params.Set("id", strconv.FormatInt(id, 10))
	var resp *AccountSummaryData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, cancelTransferByID, params, &resp)
}

// CreateDepositAddress creates a deposit address for the currency requested
func (d *Deribit) CreateDepositAddress(ctx context.Context, ccy currency.Code) (*DepositAddressData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var resp *DepositAddressData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		createDepositAddress, params, &resp)
}

// GetCurrentDepositAddress gets the current deposit address for the requested currency
func (d *Deribit) GetCurrentDepositAddress(ctx context.Context, ccy currency.Code) (*DepositAddressData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var resp *DepositAddressData
	err := d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, getCurrentDepositAddress, params, &resp)
	if err != nil {
		return nil, err
	} else if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetDeposits gets the deposits of a given currency
func (d *Deribit) GetDeposits(ctx context.Context, ccy currency.Code, count, offset int64) (*DepositsData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	if offset != 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	var resp *DepositsData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, getDeposits, params, &resp)
}

// GetTransfers gets transfers data for the requested currency
func (d *Deribit) GetTransfers(ctx context.Context, ccy currency.Code, count, offset int64) (*TransfersData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	if offset != 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	var resp *TransfersData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		getTransfers, params, &resp)
}

// GetWithdrawals gets withdrawals data for a requested currency
func (d *Deribit) GetWithdrawals(ctx context.Context, ccy currency.Code, count, offset int64) (*WithdrawalsData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	if offset != 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	var resp *WithdrawalsData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		getWithdrawals, params, &resp)
}

// SubmitTransferBetweenSubAccounts transfer funds between two (sub)accounts.
// Id of the source (sub)account. Can be found in My Account >> Subaccounts tab. By default, it is the Id of the account which made the request.
// However, if a different "source" is specified, the user must possess the mainaccount scope, and only other subaccounts can be designated as the source.
func (d *Deribit) SubmitTransferBetweenSubAccounts(ctx context.Context, ccy currency.Code, amount float64, destinationID int64, source string) (*TransferData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, errInvalidAmount
	}
	if destinationID <= 0 {
		return nil, errInvalidDestinationID
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	params.Set("destination", strconv.FormatInt(destinationID, 10))
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	if source != "" {
		params.Set("source", source)
	}
	var resp *TransferData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, submitTransferBetweenSubAccounts, params, &resp)
}

// SubmitTransferToSubAccount submits a request to transfer a currency to a subaccount
func (d *Deribit) SubmitTransferToSubAccount(ctx context.Context, ccy currency.Code, amount float64, destinationID int64) (*TransferData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, errInvalidAmount
	}
	if destinationID <= 0 {
		return nil, errInvalidDestinationID
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	params.Set("destination", strconv.FormatInt(destinationID, 10))
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp *TransferData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		submitTransferToSubaccount, params, &resp)
}

// SubmitTransferToUser submits a request to transfer a currency to another user
func (d *Deribit) SubmitTransferToUser(ctx context.Context, ccy currency.Code, tfa, destinationAddress string, amount float64) (*TransferData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, errInvalidAmount
	}
	if destinationAddress == "" {
		return nil, errInvalidCryptoAddress
	}
	params := url.Values{}
	if tfa != "" {
		params.Set("tfa", tfa)
	}
	params.Set("currency", ccy.String())
	params.Set("destination", destinationAddress)
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp *TransferData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, submitTransferToUser, params, &resp)
}

// SubmitWithdraw submits a withdrawal request to the exchange for the requested currency
func (d *Deribit) SubmitWithdraw(ctx context.Context, ccy currency.Code, address, priority string, amount float64) (*WithdrawData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, errInvalidAmount
	}
	if address == "" {
		return nil, errInvalidCryptoAddress
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	params.Set("address", address)
	if priority != "" {
		params.Set("priority", priority)
	}
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp *WithdrawData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
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
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL, common.EncodeURLValues(getAnnouncements, params), &resp)
}

// ChangeAPIKeyName changes the name of the api key requested
func (d *Deribit) ChangeAPIKeyName(ctx context.Context, id int64, name string) (*APIKeyData, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w, invalid api key id", errInvalidID)
	}
	if !alphaNumericRegExp.MatchString(name) {
		return nil, errUnacceptableAPIKey
	}
	params := url.Values{}
	params.Set("id", strconv.FormatInt(id, 10))
	params.Set("name", name)
	var resp *APIKeyData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		changeAPIKeyName, params, &resp)
}

// ChangeMarginModel change margin model
// Margin model: 'cross_pm', 'cross_sm', 'segregated_pm', 'segregated_sm'
// 'dry_run': If true request returns the result without switching the margining model. Default: false
func (d *Deribit) ChangeMarginModel(ctx context.Context, userID int64, marginModel string, dryRun bool) ([]TogglePortfolioMarginResponse, error) {
	if marginModel == "" {
		return nil, errInvalidMarginModel
	}
	params := url.Values{}
	params.Set("margin_model", marginModel)
	if userID != 0 {
		params.Set("user_id", strconv.FormatInt(userID, 10))
	}
	if dryRun {
		params.Set("dry_run", "true")
	}
	var resp []TogglePortfolioMarginResponse
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, changeMarginModel, params, &resp)
}

// ChangeScopeInAPIKey changes the scope of the api key requested
func (d *Deribit) ChangeScopeInAPIKey(ctx context.Context, id int64, maxScope string) (*APIKeyData, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w, invalid api key id", errInvalidID)
	}
	params := url.Values{}
	params.Set("id", strconv.FormatInt(id, 10))
	params.Set("max_scope", maxScope)
	var resp *APIKeyData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		changeScopeInAPIKey, params, &resp)
}

// ChangeSubAccountName changes the name of the requested subaccount id
func (d *Deribit) ChangeSubAccountName(ctx context.Context, sid int64, name string) error {
	if sid <= 0 {
		return fmt.Errorf("%w, invalid subaccount user id", errInvalidID)
	}
	if name == "" {
		return errInvalidUsername
	}
	params := url.Values{}
	params.Set("sid", strconv.FormatInt(sid, 10))
	params.Set("name", name)
	var resp string
	err := d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		changeSubAccountName, params, &resp)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return errSubAccountNameChangeFailed
	}
	return nil
}

// CreateAPIKey creates an api key based on the provided settings
func (d *Deribit) CreateAPIKey(ctx context.Context, maxScope, name string, defaultKey bool) (*APIKeyData, error) {
	params := url.Values{}
	params.Set("max_scope", maxScope)
	if name != "" {
		params.Set("name", name)
	}
	if defaultKey {
		params.Set("default", "true")
	}
	var resp *APIKeyData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		createAPIKey, params, &resp)
}

// CreateSubAccount creates a new subaccount
func (d *Deribit) CreateSubAccount(ctx context.Context) (*SubAccountData, error) {
	var resp *SubAccountData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		createSubAccount, nil, &resp)
}

// DisableAPIKey disables the api key linked to the provided id
func (d *Deribit) DisableAPIKey(ctx context.Context, id int64) (*APIKeyData, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w, invalid api key id", errInvalidID)
	}
	params := url.Values{}
	params.Set("id", strconv.FormatInt(id, 10))
	var resp *APIKeyData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		disableAPIKey, params, &resp)
}

// EditAPIKey edits existing API key. At least one parameter is required.
// Describes maximal access for tokens generated with given key, possible values:
// trade:[read, read_write, none],
// wallet:[read, read_write, none],
// account:[read, read_write, none],
// block_trade:[read, read_write, none].
func (d *Deribit) EditAPIKey(ctx context.Context, id int64, maxScope, name string, enabled bool, enabledFeatures, ipWhitelist []string) (*APIKeyData, error) {
	if id == 0 {
		return nil, errInvalidAPIKeyID
	}
	if maxScope == "" {
		return nil, errMaxScopeIsRequired
	}
	params := url.Values{}
	if name != "" {
		params.Set("name", name)
	}
	if enabled {
		params.Set("enabled", "true")
	}
	if len(enabledFeatures) > 0 {
		enabledFeaturesByte, err := json.Marshal(enabledFeatures)
		if err != nil {
			return nil, err
		}
		params.Set("enabled_features", string(enabledFeaturesByte))
	}
	if len(ipWhitelist) > 0 {
		ipWhitelistByte, err := json.Marshal(ipWhitelist)
		if err != nil {
			return nil, err
		}
		params.Set("ip_whitelist", string(ipWhitelistByte))
	}
	var resp *APIKeyData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, editAPIKey, params, &resp)
}

// EnableAffiliateProgram enables the affiliate program
func (d *Deribit) EnableAffiliateProgram(ctx context.Context) error {
	var resp string
	err := d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		enableAffiliateProgram, nil, &resp)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return errors.New("could not enable affiliate program")
	}
	return nil
}

// EnableAPIKey enables the api key linked to the provided id
func (d *Deribit) EnableAPIKey(ctx context.Context, id int64) (*APIKeyData, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w, invalid api key id", errInvalidID)
	}
	params := url.Values{}
	params.Set("id", strconv.FormatInt(id, 10))
	var resp *APIKeyData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
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
	var resp *AccessLog
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, getAccessLog, params, &resp)
}

// GetAffiliateProgramInfo gets the affiliate program info
func (d *Deribit) GetAffiliateProgramInfo(ctx context.Context) (*AffiliateProgramInfo, error) {
	var resp *AffiliateProgramInfo
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, getAffiliateProgramInfo, nil, &resp)
}

// GetEmailLanguage gets the current language set for the email
func (d *Deribit) GetEmailLanguage(ctx context.Context) (string, error) {
	var resp string
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		getEmailLanguage, nil, &resp)
}

// GetNewAnnouncements gets new announcements
func (d *Deribit) GetNewAnnouncements(ctx context.Context) ([]Announcement, error) {
	var resp []Announcement
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		getNewAnnouncements, nil, &resp)
}

// GetPosition gets the data of all positions in the requested instrument name
func (d *Deribit) GetPosition(ctx context.Context, instrument string) (*PositionData, error) {
	params, err := checkInstrument(instrument)
	if err != nil {
		return nil, err
	}
	var resp *PositionData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		getPosition, params, &resp)
}

// GetSubAccounts gets all subaccounts' data
func (d *Deribit) GetSubAccounts(ctx context.Context, withPortfolio bool) ([]SubAccountData, error) {
	params := url.Values{}
	if withPortfolio {
		params.Set("with_portfolio", "true")
	}
	var resp []SubAccountData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		getSubAccounts, params, &resp)
}

// GetSubAccountDetails retrieves sub accounts detail information.
func (d *Deribit) GetSubAccountDetails(ctx context.Context, ccy currency.Code, withOpenOrders bool) ([]SubAccountDetail, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if withOpenOrders {
		params.Set("with_open_orders", "true")
	}
	var resp []SubAccountDetail
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, getSubAccountDetails, params, &resp)
}

// GetPositions gets positions data of the user account
func (d *Deribit) GetPositions(ctx context.Context, ccy currency.Code, kind string) ([]PositionData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if kind != "" {
		params.Set("kind", kind)
	}
	var resp []PositionData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		getPositions, params, &resp)
}

// GetTransactionLog gets transaction logs' data
func (d *Deribit) GetTransactionLog(ctx context.Context, ccy currency.Code, query string, startTime, endTime time.Time, count, continuation int64) (*TransactionsData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if query != "" {
		params.Set("query", query)
	}
	err := common.StartEndTimeCheck(startTime, endTime)
	if err != nil {
		return nil, err
	}
	params.Set("start_timestamp", strconv.FormatInt(startTime.UnixMilli(), 10))
	params.Set("end_timestamp", strconv.FormatInt(endTime.UnixMilli(), 10))
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	if continuation != 0 {
		params.Set("continuation", strconv.FormatInt(continuation, 10))
	}
	var resp *TransactionsData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, getTransactionLog, params, &resp)
}

// GetUserLocks retrieves information about locks on user account.
func (d *Deribit) GetUserLocks(ctx context.Context) ([]UserLock, error) {
	var resp []UserLock
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, getUserLocks, nil, &resp)
}

// ListAPIKeys lists all the api keys associated with a user account
func (d *Deribit) ListAPIKeys(ctx context.Context, tfa string) ([]APIKeyData, error) {
	params := url.Values{}
	if tfa != "" {
		params.Set("tfa", tfa)
	}
	var resp []APIKeyData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		listAPIKeys, params, &resp)
}

// GetCustodyAccounts retrieves user custody accounts list.
func (d *Deribit) GetCustodyAccounts(ctx context.Context, ccy currency.Code) ([]CustodyAccount, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var resp []CustodyAccount
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, listCustodyAccounts, params, &resp)
}

// RemoveAPIKey removes api key vid ID
func (d *Deribit) RemoveAPIKey(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w, invalid api key id", errInvalidID)
	}
	params := url.Values{}
	params.Set("id", strconv.FormatInt(id, 10))
	var resp any
	err := d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, removeAPIKey, params, &resp)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return errors.New("removal of the api key requested failed")
	}
	return nil
}

// RemoveSubAccount removes a subaccount given its id
func (d *Deribit) RemoveSubAccount(ctx context.Context, subAccountID int64) error {
	params := url.Values{}
	params.Set("subaccount_id", strconv.FormatInt(subAccountID, 10))
	var resp string
	err := d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, removeSubAccount, params, &resp)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return fmt.Errorf("removal of sub account %v failed", subAccountID)
	}
	return nil
}

// ResetAPIKey resets the api key to its default settings
func (d *Deribit) ResetAPIKey(ctx context.Context, id int64) (*APIKeyData, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w, invalid api key id", errInvalidID)
	}
	params := url.Values{}
	params.Set("id", strconv.FormatInt(id, 10))
	var resp *APIKeyData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		resetAPIKey, params, &resp)
}

// SetAnnouncementAsRead sets an announcement as read
func (d *Deribit) SetAnnouncementAsRead(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w, invalid announcement id", errInvalidID)
	}
	params := url.Values{}
	params.Set("announcement_id", strconv.FormatInt(id, 10))
	var resp string
	err := d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		setAnnouncementAsRead, params, &resp)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return fmt.Errorf("setting announcement %v as read failed", id)
	}
	return nil
}

// SetEmailForSubAccount links an email given to the designated subaccount
func (d *Deribit) SetEmailForSubAccount(ctx context.Context, sid int64, email string) error {
	if sid <= 0 {
		return fmt.Errorf("%w, invalid subaccount user id", errInvalidID)
	}
	if !common.MatchesEmailPattern(email) {
		return errInvalidEmailAddress
	}
	params := url.Values{}
	params.Set("sid", strconv.FormatInt(sid, 10))
	params.Set("email", email)
	var resp any
	err := d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		setEmailForSubAccount, params, &resp)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return fmt.Errorf("could not link email (%v) to subaccount %v", email, sid)
	}
	return nil
}

// SetEmailLanguage sets a requested language for an email
func (d *Deribit) SetEmailLanguage(ctx context.Context, language string) error {
	if language == "" {
		return errLanguageIsRequired
	}
	params := url.Values{}
	params.Set("language", language)
	var resp string
	err := d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, setEmailLanguage, params, &resp)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return fmt.Errorf("could not set the email language to %v", language)
	}
	return nil
}

// SetSelfTradingConfig configure self trading behavior
// mode: Self trading prevention behavior. Possible values: 'reject_taker', 'cancel_maker'
// extended_to_subaccounts: If value is true trading is prevented between subaccounts of given account
func (d *Deribit) SetSelfTradingConfig(ctx context.Context, mode string, extendedToSubaccounts bool) (string, error) {
	if mode == "" {
		return "", errTradeModeIsRequired
	}
	params := url.Values{}
	params.Set("mode", mode)
	if extendedToSubaccounts {
		params.Set("extended_to_subaccounts", "true")
	} else {
		params.Set("extended_to_subaccounts", "false")
	}
	var resp string
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, setSelfTradingConfig, params, &resp)
}

// ToggleNotificationsFromSubAccount toggles the notifications from a subaccount specified
func (d *Deribit) ToggleNotificationsFromSubAccount(ctx context.Context, sid int64, state bool) error {
	if sid <= 0 {
		return fmt.Errorf("%w, invalid subaccount user id", errInvalidID)
	}
	params := url.Values{}
	params.Set("sid", strconv.FormatInt(sid, 10))
	params.Set("state", strconv.FormatBool(state))
	var resp string
	err := d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		toggleNotificationsFromSubAccount, params, &resp)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return fmt.Errorf("toggling notifications for subaccount %v to %v failed", sid, state)
	}
	return nil
}

// TogglePortfolioMargining toggle between SM and PM models.
func (d *Deribit) TogglePortfolioMargining(ctx context.Context, userID int64, enabled, dryRun bool) ([]TogglePortfolioMarginResponse, error) {
	if userID == 0 {
		return nil, errUserIDRequired
	}
	params := url.Values{}
	params.Set("user_id", strconv.FormatInt(userID, 10))
	params.Set("enabled", strconv.FormatBool(enabled))
	if dryRun {
		params.Set("dry_run", "true")
	}
	var resp []TogglePortfolioMarginResponse
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, togglePortfolioMargining, params, &resp)
}

// ToggleSubAccountLogin toggles access for subaccount login
func (d *Deribit) ToggleSubAccountLogin(ctx context.Context, subAccountUserID int64, state bool) error {
	if subAccountUserID <= 0 {
		return fmt.Errorf("%w, invalid subaccount user id", errInvalidID)
	}
	params := url.Values{}
	params.Set("sid", strconv.FormatInt(subAccountUserID, 10))
	params.Set("state", strconv.FormatBool(state))
	var resp string
	err := d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, toggleSubAccountLogin, params, &resp)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return fmt.Errorf("toggling login access for subaccount %v to %v failed", subAccountUserID, state)
	}
	return nil
}

// SubmitBuy submits a private buy request through the websocket connection.
func (d *Deribit) SubmitBuy(ctx context.Context, arg *OrderBuyAndSellParams) (*PrivateTradeData, error) {
	if arg == nil || *arg == (OrderBuyAndSellParams{}) {
		return nil, fmt.Errorf("%w parameter is required", common.ErrNilPointer)
	}
	params, err := checkInstrument(arg.Instrument)
	if err != nil {
		return nil, err
	}
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
	if arg.PostOnly {
		params.Set("post_only", "true")
	}
	if arg.RejectPostOnly {
		params.Set("reject_post_only", "true")
	}
	if arg.ReduceOnly {
		params.Set("reduce_only", "true")
	}
	if arg.MMP {
		params.Set("mmp", "true")
	}
	if arg.TriggerPrice != 0 {
		params.Set("trigger_price", strconv.FormatFloat(arg.TriggerPrice, 'f', -1, 64))
	}
	if arg.Trigger != "" {
		params.Set("trigger", arg.Trigger)
	}
	if arg.Advanced != "" {
		params.Set("advanced", arg.Advanced)
	}
	var resp *PrivateTradeData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, matchingEPL, http.MethodGet,
		submitBuy, params, &resp)
}

// SubmitSell submits a sell request with the parameters provided
func (d *Deribit) SubmitSell(ctx context.Context, arg *OrderBuyAndSellParams) (*PrivateTradeData, error) {
	if arg == nil || *arg == (OrderBuyAndSellParams{}) {
		return nil, fmt.Errorf("%w argument is required", common.ErrNilPointer)
	}
	params, err := checkInstrument(arg.Instrument)
	if err != nil {
		return nil, err
	}
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
	if (arg.OrderType == "limit" || arg.OrderType == "stop_limit") && arg.Price <= 0 {
		return nil, errInvalidPrice
	}
	params.Set("price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	if arg.PostOnly {
		params.Set("post_only", "true")
	}
	if arg.RejectPostOnly {
		params.Set("reject_post_only", "true")
	}
	if arg.ReduceOnly {
		params.Set("reduce_only", "true")
	}
	if arg.MMP {
		params.Set("mmp", "true")
	}
	if arg.TriggerPrice != 0 {
		params.Set("trigger_price", strconv.FormatFloat(arg.TriggerPrice, 'f', -1, 64))
	}
	if arg.Trigger != "" {
		params.Set("trigger", arg.Trigger)
	}
	if arg.Advanced != "" {
		params.Set("advanced", arg.Advanced)
	}
	var resp *PrivateTradeData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestSpot, matchingEPL, http.MethodGet, submitSell, params, &resp)
}

// SubmitEdit submits an edit order request
func (d *Deribit) SubmitEdit(ctx context.Context, arg *OrderBuyAndSellParams) (*PrivateTradeData, error) {
	if arg == nil || *arg == (OrderBuyAndSellParams{}) {
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
	if arg.PostOnly {
		params.Set("post_only", "true")
	}
	if arg.RejectPostOnly {
		params.Set("reject_post_only", "true")
	}
	if arg.ReduceOnly {
		params.Set("reduce_only", "true")
	}
	if arg.MMP {
		params.Set("mmp", "true")
	}
	if arg.TriggerPrice != 0 {
		params.Set("trigger_price", strconv.FormatFloat(arg.TriggerPrice, 'f', -1, 64))
	}
	if arg.Advanced != "" {
		params.Set("advanced", arg.Advanced)
	}
	if arg.Price > 0 {
		params.Set("price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	}
	var resp *PrivateTradeData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, matchingEPL, http.MethodGet, submitEdit, params, &resp)
}

// EditOrderByLabel submits an edit order request sorted via label
func (d *Deribit) EditOrderByLabel(ctx context.Context, arg *OrderBuyAndSellParams) (*PrivateTradeData, error) {
	if arg == nil || *arg == (OrderBuyAndSellParams{}) {
		return nil, fmt.Errorf("%w parameter is required", common.ErrNilPointer)
	}
	params, err := checkInstrument(arg.Instrument)
	if err != nil {
		return nil, err
	}
	if arg.Amount <= 0 {
		return nil, errInvalidAmount
	}
	if arg.Label != "" {
		params.Set("label", arg.Label)
	}
	params.Set("amount", strconv.FormatFloat(arg.Amount, 'f', -1, 64))
	if arg.PostOnly {
		params.Set("post_only", "true")
	}
	if arg.RejectPostOnly {
		params.Set("reject_post_only", "true")
	}
	if arg.ReduceOnly {
		params.Set("reduce_only", "true")
	}
	if arg.MMP {
		params.Set("mmp", "true")
	}
	if arg.TriggerPrice != 0 {
		params.Set("trigger_price", strconv.FormatFloat(arg.TriggerPrice, 'f', -1, 64))
	}
	if arg.Advanced != "" {
		params.Set("advanced", arg.Advanced)
	}
	var resp *PrivateTradeData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		editByLabel, params, &resp)
}

// SubmitCancel sends a request to cancel the order via its orderID
func (d *Deribit) SubmitCancel(ctx context.Context, orderID string) (*PrivateCancelData, error) {
	if orderID == "" {
		return nil, fmt.Errorf("%w, no order ID specified", errInvalidID)
	}
	params := url.Values{}
	params.Set("order_id", orderID)
	var resp *PrivateCancelData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, matchingEPL, http.MethodGet,
		submitCancel, params, &resp)
}

// SubmitCancelAll sends a request to cancel all user orders in all currencies and instruments
func (d *Deribit) SubmitCancelAll(ctx context.Context, detailed bool) (*MultipleCancelResponse, error) {
	params := url.Values{}
	if detailed {
		params.Set("detailed", "true")
	}
	var resp *MultipleCancelResponse
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, matchingEPL, http.MethodGet,
		submitCancelAll, params, &resp)
}

// SubmitCancelAllByCurrency sends a request to cancel all user orders for the specified currency
// returns the total number of successfully cancelled orders
func (d *Deribit) SubmitCancelAllByCurrency(ctx context.Context, ccy currency.Code, kind, orderType string, detailed bool) (*MultipleCancelResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if kind != "" {
		params.Set("kind", kind)
	}
	if orderType != "" {
		params.Set("type", orderType)
	}
	if detailed {
		params.Set("detailed", "true")
	}
	var resp *MultipleCancelResponse
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, matchingEPL, http.MethodGet,
		submitCancelAllByCurrency, params, &resp)
}

// SubmitCancelAllByKind cancels all orders in currency(currencies), optionally filtered by instrument kind and/or order type.
// 'kind' instrument kind . Possible values: 'future', 'option', 'spot', 'future_combo', 'option_combo', 'combo', 'any'
// returns the total number of successfully cancelled orders
func (d *Deribit) SubmitCancelAllByKind(ctx context.Context, ccy currency.Code, kind, orderType string, detailed bool) (*MultipleCancelResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if kind != "" {
		params.Set("kind", kind)
	}
	if orderType != "" {
		params.Set("type", orderType)
	}
	if detailed {
		params.Set("detailed", "true")
	}
	var resp *MultipleCancelResponse
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestSpot, matchingEPL, http.MethodGet, submitCancelAllByKind, params, &resp)
}

// SubmitCancelAllByInstrument sends a request to cancel all user orders for the specified instrument
// returns the total number of successfully cancelled orders
func (d *Deribit) SubmitCancelAllByInstrument(ctx context.Context, instrument, orderType string, detailed, includeCombos bool) (*MultipleCancelResponse, error) {
	params, err := checkInstrument(instrument)
	if err != nil {
		return nil, err
	}
	if orderType != "" {
		params.Set("type", orderType)
	}
	if detailed {
		params.Set("detailed", "true")
	}
	if includeCombos {
		params.Set("include_combos", "true")
	}
	var resp *MultipleCancelResponse
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, matchingEPL, http.MethodGet,
		submitCancelAllByInstrument, params, &resp)
}

// SubmitCancelByLabel sends a request to cancel all user orders for the specified label
// returns the total number of successfully cancelled orders
func (d *Deribit) SubmitCancelByLabel(ctx context.Context, label string, ccy currency.Code, detailed bool) (*MultipleCancelResponse, error) {
	params := url.Values{}
	params.Set("label", label)
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if detailed {
		params.Set("detailed", "true")
	}
	var resp *MultipleCancelResponse
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, matchingEPL, http.MethodGet,
		submitCancelByLabel, params, &resp)
}

// SubmitCancelQuotes cancels quotes based on the provided type.
// Delta cancels quotes within a Delta range defined by MinDelta and MaxDelta.
// quote_set_id cancels quotes by a specific Quote Set identifier.
// instrument cancels all quotes associated with a particular instrument. kind cancels all quotes for a certain kind.
// currency cancels all quotes in a specified currency. "all" cancels all quotes.
//
// possible cancel_type values are delta, 'quote_set_id', 'instrument', 'instrument_kind', 'currency', and 'all'
// possible kind values are future 'option', 'spot', 'future_combo', 'option_combo', 'combo', and 'any'
func (d *Deribit) SubmitCancelQuotes(ctx context.Context, ccy currency.Code, minDelta, maxDelta float64, cancelType, quoteSetID, instrumentName, kind string, detailed bool) (*MultipleCancelResponse, error) {
	if cancelType == "" {
		return nil, errors.New("cancel type is required")
	}
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("cancel_type", cancelType)
	params.Set("currency", ccy.String())
	if detailed {
		params.Set("detailed", "true")
	}
	if minDelta > 0 {
		params.Set("min_delta", strconv.FormatFloat(minDelta, 'f', -1, 64))
	}
	if maxDelta > 0 {
		params.Set("max_delta", strconv.FormatFloat(maxDelta, 'f', -1, 64))
	}
	if quoteSetID != "" {
		params.Set("quote_set_id", quoteSetID)
	}
	if instrumentName != "" {
		params.Set("instrument_name", instrumentName)
	}
	if kind != "" {
		params.Set("kind", kind)
	}
	var resp *MultipleCancelResponse
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, matchingEPL, http.MethodGet, submitCancelQuotes, params, &resp)
}

// SubmitClosePosition sends a request to cancel all user orders for the specified label
func (d *Deribit) SubmitClosePosition(ctx context.Context, instrument, orderType string, price float64) (*PrivateTradeData, error) {
	params, err := checkInstrument(instrument)
	if err != nil {
		return nil, err
	}
	if orderType != "" {
		params.Set("type", orderType)
	}
	params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	var resp *PrivateTradeData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, matchingEPL, http.MethodGet,
		submitClosePosition, params, &resp)
}

// GetMargins sends a request to fetch account margins data
func (d *Deribit) GetMargins(ctx context.Context, instrument string, amount, price float64) (*MarginsData, error) {
	params, err := checkInstrument(instrument)
	if err != nil {
		return nil, err
	}
	if amount <= 0 {
		return nil, errInvalidAmount
	}
	if price <= 0 {
		return nil, errInvalidPrice
	}
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	var resp *MarginsData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		getMargins, params, &resp)
}

// GetMMPConfig sends a request to fetch the config for MMP of the requested currency
func (d *Deribit) GetMMPConfig(ctx context.Context, ccy currency.Code) (*MMPConfigData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var resp *MMPConfigData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		getMMPConfig, params, &resp)
}

// GetOpenOrdersByCurrency retrieves open orders data sorted by requested params
func (d *Deribit) GetOpenOrdersByCurrency(ctx context.Context, ccy currency.Code, kind, orderType string) ([]OrderData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if kind != "" {
		params.Set("kind", kind)
	}
	if orderType != "" {
		params.Set("type", orderType)
	}
	var resp []OrderData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		getOpenOrdersByCurrency, params, &resp)
}

// GetOpenOrdersByLabel retrieves open orders using label and currency
func (d *Deribit) GetOpenOrdersByLabel(ctx context.Context, ccy currency.Code, label string) ([]OrderData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if label != "" {
		params.Set("label", label)
	}
	var resp []OrderData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, getOpenOrdersByLabel, params, &resp)
}

// GetOpenOrdersByInstrument sends a request to fetch open orders data sorted by requested params
func (d *Deribit) GetOpenOrdersByInstrument(ctx context.Context, instrument, orderType string) ([]OrderData, error) {
	params, err := checkInstrument(instrument)
	if err != nil {
		return nil, err
	}
	if orderType != "" {
		params.Set("type", orderType)
	}
	var resp []OrderData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, getOpenOrdersByInstrument, params, &resp)
}

// GetOrderHistoryByCurrency sends a request to fetch order history according to given params and currency
func (d *Deribit) GetOrderHistoryByCurrency(ctx context.Context, ccy currency.Code, kind string, count, offset int64, includeOld, includeUnfilled bool) ([]OrderData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if kind != "" {
		params.Set("kind", kind)
	}
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	if offset != 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	if includeOld {
		params.Set("include_old", "true")
	}
	if includeUnfilled {
		params.Set("include_unfilled", "true")
	}
	var resp []OrderData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		getOrderHistoryByCurrency, params, &resp)
}

// GetOrderHistoryByInstrument sends a request to fetch order history according to given params and instrument
func (d *Deribit) GetOrderHistoryByInstrument(ctx context.Context, instrument string, count, offset int64, includeOld, includeUnfilled bool) ([]OrderData, error) {
	params, err := checkInstrument(instrument)
	if err != nil {
		return nil, err
	}
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	if offset != 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	if includeOld {
		params.Set("include_old", "true")
	}
	if includeUnfilled {
		params.Set("include_unfilled", "true")
	}
	var resp []OrderData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestSpot, nonMatchingEPL, http.MethodGet,
		getOrderHistoryByInstrument, params, &resp)
}

// GetOrderMarginsByID sends a request to fetch order margins data according to their ids
func (d *Deribit) GetOrderMarginsByID(ctx context.Context, ids []string) ([]InitialMarginInfo, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("%w, order ids cannot be empty", errInvalidID)
	}
	params := url.Values{}
	for a := range ids {
		params.Add("ids[]", ids[a])
	}
	var resp []InitialMarginInfo
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestSpot, nonMatchingEPL, http.MethodGet,
		getOrderMarginByIDs, params, &resp)
}

// GetOrderState sends a request to fetch order state of the order id provided
func (d *Deribit) GetOrderState(ctx context.Context, orderID string) (*OrderData, error) {
	if orderID == "" {
		return nil, fmt.Errorf("%w, no order ID specified", errInvalidID)
	}
	params := url.Values{}
	params.Set("order_id", orderID)
	var resp *OrderData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		getOrderState, params, &resp)
}

// GetOrderStateByLabel retrieves an order state by label and currency
func (d *Deribit) GetOrderStateByLabel(ctx context.Context, ccy currency.Code, label string) ([]OrderData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if label != "" {
		params.Set("label", label)
	}
	var resp []OrderData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, getOrderStateByLabel, params, &resp)
}

// GetTriggerOrderHistory sends a request to fetch order state of the order id provided
func (d *Deribit) GetTriggerOrderHistory(ctx context.Context, ccy currency.Code, instrumentName, continuation string, count int64) (*OrderData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if instrumentName != "" {
		params.Set("instrument_name", instrumentName)
	}
	if continuation != "" {
		params.Set("continuation", continuation)
	}
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	var resp *OrderData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		getTriggerOrderHistory, params, &resp)
}

// GetUserTradesByCurrency sends a request to fetch user trades sorted by currency
func (d *Deribit) GetUserTradesByCurrency(ctx context.Context, ccy currency.Code, kind, startID, endID, sorting string, count int64, includeOld bool) (*UserTradesData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
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
	if includeOld {
		params.Set("include_old", "true")
	}
	var resp *UserTradesData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		getUserTradesByCurrency, params, &resp)
}

// GetUserTradesByCurrencyAndTime sends a request to fetch user trades sorted by currency and time
func (d *Deribit) GetUserTradesByCurrencyAndTime(ctx context.Context, ccy currency.Code, kind, sorting string, count int64, startTime, endTime time.Time) (*UserTradesData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
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
	var resp *UserTradesData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, getUserTradesByCurrencyAndTime, params, &resp)
}

// GetUserTradesByInstrument sends a request to fetch user trades sorted by instrument
func (d *Deribit) GetUserTradesByInstrument(ctx context.Context, instrument, sorting string, startSeq, endSeq, count int64, includeOld bool) (*UserTradesData, error) {
	params, err := checkInstrument(instrument)
	if err != nil {
		return nil, err
	}
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
	if includeOld {
		params.Set("include_old", "true")
	}
	var resp *UserTradesData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		getUserTradesByInstrument, params, &resp)
}

// GetUserTradesByInstrumentAndTime sends a request to fetch user trades sorted by instrument and time
func (d *Deribit) GetUserTradesByInstrumentAndTime(ctx context.Context, instrument, sorting string, count int64, startTime, endTime time.Time) (*UserTradesData, error) {
	params, err := checkInstrument(instrument)
	if err != nil {
		return nil, err
	}
	if sorting != "" {
		params.Set("sorting", sorting)
	}
	if count != 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	err = common.StartEndTimeCheck(startTime, endTime)
	if err != nil {
		return nil, err
	}
	params.Set("start_timestamp", strconv.FormatInt(startTime.UnixMilli(), 10))
	params.Set("end_timestamp", strconv.FormatInt(endTime.UnixMilli(), 10))
	var resp *UserTradesData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
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
	var resp *UserTradesData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		getUserTradesByOrder, params, &resp)
}

// ResetMMP sends a request to reset MMP for a currency provided
func (d *Deribit) ResetMMP(ctx context.Context, ccy currency.Code) error {
	if ccy.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var resp string
	err := d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, resetMMP, params, &resp)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return fmt.Errorf("mmp could not be reset for %v", ccy.String())
	}
	return nil
}

// SendRequestForQuote sends RFQ on a given instrument.
func (d *Deribit) SendRequestForQuote(ctx context.Context, instrumentName string, amount float64, side order.Side) error {
	params, err := checkInstrument(instrumentName)
	if err != nil {
		return err
	}
	if amount > 0 {
		params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	}
	if side != order.UnknownSide {
		params.Set("side", side.String())
	}
	var resp string
	err = d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, sendRFQ, params, &resp)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return fmt.Errorf("rfq couldn't send for %v", instrumentName)
	}
	return nil
}

// SetMMPConfig sends a request to set the given parameter values to the mmp config for the provided currency
func (d *Deribit) SetMMPConfig(ctx context.Context, ccy currency.Code, interval kline.Interval, frozenTime int64, quantityLimit, deltaLimit float64) error {
	if ccy.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	intervalString, err := d.GetResolutionFromInterval(interval)
	if err != nil {
		return err
	}
	params.Set("interval", intervalString)
	params.Set("frozen_time", strconv.FormatInt(frozenTime, 10))
	if quantityLimit != 0 {
		params.Set("quantity_limit", strconv.FormatFloat(quantityLimit, 'f', -1, 64))
	}
	if deltaLimit != 0 {
		params.Set("delta_limit", strconv.FormatFloat(deltaLimit, 'f', -1, 64))
	}
	var resp string
	err = d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		setMMPConfig, params, &resp)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return fmt.Errorf("mmp data could not be set for %v", ccy.String())
	}
	return nil
}

// GetSettlementHistoryByInstrument sends a request to fetch settlement history data sorted by instrument
func (d *Deribit) GetSettlementHistoryByInstrument(ctx context.Context, instrument, settlementType, continuation string, count int64, searchStartTimeStamp time.Time) (*PrivateSettlementsHistoryData, error) {
	params, err := checkInstrument(instrument)
	if err != nil {
		return nil, err
	}
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
	var resp *PrivateSettlementsHistoryData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		getSettlementHistoryByInstrument, params, &resp)
}

// GetSettlementHistoryByCurency sends a request to fetch settlement history data sorted by currency
func (d *Deribit) GetSettlementHistoryByCurency(ctx context.Context, ccy currency.Code, settlementType, continuation string, count int64, searchStartTimeStamp time.Time) (*PrivateSettlementsHistoryData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
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
	var resp *PrivateSettlementsHistoryData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet,
		getSettlementHistoryByCurrency, params, &resp)
}

// SendHTTPAuthRequest sends an authenticated request to deribit api
func (d *Deribit) SendHTTPAuthRequest(ctx context.Context, ep exchange.URL, epl request.EndpointLimit, method, path string, params url.Values, result any) error {
	endpoint, err := d.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	creds, err := d.GetCredentials(ctx)
	if err != nil {
		return fmt.Errorf("%w, %v", request.ErrAuthRequestFailed, err)
	}
	req := method + "\n" + deribitAPIVersion + "/" + common.EncodeURLValues(path, params) + "\n\n"
	n := d.Requester.GetNonce(nonce.UnixNano).String()
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	tsReq := []byte(ts + "\n" + n + "\n" + req)
	hmac, err := crypto.GetHMAC(crypto.HashSHA256, tsReq, []byte(creds.Secret))
	if err != nil {
		return err
	}
	headers := make(map[string]string)
	headerString := "deri-hmac-sha256 id=" + creds.Key + ",ts=" + ts + ",sig=" + hex.EncodeToString(hmac) + ",nonce=" + n
	headers["Authorization"] = headerString
	headers["Content-Type"] = "application/json"
	var tempData struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      int64           `json:"id"`
		Data    json.RawMessage `json:"result"`
		Error   ErrInfo         `json:"error"`
	}
	err = d.SendPayload(ctx, epl, func() (*request.Item, error) {
		return &request.Item{
			Method:        method,
			Path:          endpoint + deribitAPIVersion + "/" + common.EncodeURLValues(path, params),
			Headers:       headers,
			Result:        &tempData,
			Verbose:       d.Verbose,
			HTTPDebugging: d.HTTPDebugging,
			HTTPRecording: d.HTTPRecording,
		}, nil
	}, request.AuthenticatedRequest)
	if err != nil {
		return fmt.Errorf("%w %v", request.ErrAuthRequestFailed, err)
	}
	if tempData.Error.Code != 0 {
		var errParamInfo string
		if tempData.Error.Data.Param != "" {
			errParamInfo = fmt.Sprintf(" param: %s reason: %s", tempData.Error.Data.Param, tempData.Error.Data.Reason)
		}
		return fmt.Errorf("%w code: %d msg: %s%s", request.ErrAuthRequestFailed, tempData.Error.Code, tempData.Error.Message, errParamInfo)
	}
	return json.Unmarshal(tempData.Data, result)
}

// Combo Books endpoints'

// GetComboIDs Retrieves available combos.
// This method can be used to get the list of all combos, or only the list of combos in the given state.
func (d *Deribit) GetComboIDs(ctx context.Context, ccy currency.Code, state string) ([]string, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if state != "" {
		params.Set("state", state)
	}
	var resp []string
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL, common.EncodeURLValues(getComboIDs, params), &resp)
}

// GetComboDetails retrieves information about a combo
func (d *Deribit) GetComboDetails(ctx context.Context, comboID string) (*ComboDetail, error) {
	if comboID == "" {
		return nil, errInvalidComboID
	}
	params := url.Values{}
	params.Set("combo_id", comboID)
	var resp *ComboDetail
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL, common.EncodeURLValues(getComboDetails, params), &resp)
}

// GetCombos retrieves information about active combos
func (d *Deribit) GetCombos(ctx context.Context, ccy currency.Code) ([]ComboDetail, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var resp []ComboDetail
	return resp, d.SendHTTPRequest(ctx, exchange.RestFutures, nonMatchingEPL, common.EncodeURLValues(getCombos, params), &resp)
}

// CreateCombo verifies and creates a combo book or returns an existing combo matching given trades
func (d *Deribit) CreateCombo(ctx context.Context, args []ComboParam) (*ComboDetail, error) {
	if len(args) == 0 {
		return nil, errNoArgumentPassed
	}
	instrument := args[0].InstrumentName
	for x := range args {
		if args[x].InstrumentName == "" {
			return nil, fmt.Errorf("%w, empty string", errInvalidInstrumentName)
		} else if instrument != args[x].InstrumentName {
			return nil, errDifferentInstruments
		}
		args[x].Direction = strings.ToLower(args[x].Direction)
		if args[x].Direction != sideBUY && args[x].Direction != sideSELL {
			return nil, errInvalidOrderSideOrDirection
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
	var resp *ComboDetail
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, createCombos, params, &resp)
}

// ExecuteBlockTrade executes a block trade request
// The whole request have to be exact the same as in private/verify_block_trade, only role field should be set appropriately - it basically means that both sides have to agree on the same timestamp, nonce, trades fields and server will assure that role field is different between sides (each party accepted own role).
// Using the same timestamp and nonce by both sides in private/verify_block_trade assures that even if unintentionally both sides execute given block trade with valid counterparty_signature, the given block trade will be executed only once
func (d *Deribit) ExecuteBlockTrade(ctx context.Context, timestampMS time.Time, nonce, role string, ccy currency.Code, trades []BlockTradeParam) ([]BlockTradeResponse, error) {
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
	signature, err := d.VerifyBlockTrade(ctx, timestampMS, nonce, role, ccy, trades)
	if err != nil {
		return nil, err
	}
	values, err := json.Marshal(trades)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	params.Set("trades", string(values))
	params.Set("nonce", nonce)
	params.Set("role", role)
	params.Set("counterparty_signature", signature)
	params.Set("timestamp", strconv.FormatInt(timestampMS.UnixMilli(), 10))
	var resp []BlockTradeResponse
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, matchingEPL, http.MethodGet, executeBlockTrades, params, &resp)
}

// VerifyBlockTrade verifies and creates block trade signature
func (d *Deribit) VerifyBlockTrade(ctx context.Context, timestampMS time.Time, nonce, role string, ccy currency.Code, trades []BlockTradeParam) (string, error) {
	if nonce == "" {
		return "", errMissingNonce
	}
	if role != roleMaker && role != roleTaker {
		return "", errInvalidTradeRole
	}
	if len(trades) == 0 {
		return "", errNoArgumentPassed
	}
	if timestampMS.IsZero() {
		return "", errZeroTimestamp
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
	values, err := json.Marshal(trades)
	if err != nil {
		return "", err
	}
	params := url.Values{}
	params.Set("timestamp", strconv.FormatInt(timestampMS.UnixMilli(), 10))
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	params.Set("nonce", nonce)
	params.Set("role", role)
	params.Set("trades", string(values))
	resp := &struct {
		Signature string `json:"signature"`
	}{}
	return resp.Signature, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, matchingEPL, http.MethodGet, verifyBlockTrades, params, resp)
}

// InvalidateBlockTradeSignature user at any time (before the private/execute_block_trade is called) can invalidate its own signature effectively cancelling block trade
func (d *Deribit) InvalidateBlockTradeSignature(ctx context.Context, signature string) error {
	if signature == "" {
		return errMissingSignature
	}
	params := url.Values{}
	params.Set("signature", signature)
	var resp string
	err := d.SendHTTPAuthRequest(ctx, exchange.RestSpot, nonMatchingEPL, http.MethodGet, invalidateBlockTradesSignature, params, &resp)
	if err != nil {
		return err
	}
	if resp != "ok" {
		return fmt.Errorf("server response: %s", resp)
	}
	return nil
}

// GetUserBlockTrade returns information about users block trade
func (d *Deribit) GetUserBlockTrade(ctx context.Context, id string) ([]BlockTradeData, error) {
	if id == "" {
		return nil, errMissingBlockTradeID
	}
	params := url.Values{}
	params.Set("id", id)
	var resp []BlockTradeData
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, getBlockTrades, params, &resp)
}

// GetTime retrieves the current time (in milliseconds). This API endpoint can be used to check the clock skew between your software and Deribit's systems.
func (d *Deribit) GetTime(ctx context.Context) (time.Time, error) {
	var timestamp types.Time
	if err := d.SendHTTPRequest(ctx, exchange.RestSpot, nonMatchingEPL, "public/get_time", &timestamp); err != nil {
		return time.Time{}, err
	}
	return timestamp.Time(), nil
}

// GetLastBlockTradesByCurrency returns list of last users block trades
func (d *Deribit) GetLastBlockTradesByCurrency(ctx context.Context, ccy currency.Code, startID, endID string, count int64) ([]BlockTradeData, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
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
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, getLastBlockTradesByCurrency, params, &resp)
}

// MovePositions moves positions from source subaccount to target subaccount
func (d *Deribit) MovePositions(ctx context.Context, ccy currency.Code, sourceSubAccountUID, targetSubAccountUID int64, trades []BlockTradeParam) ([]BlockTradeMoveResponse, error) {
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
	values, err := json.Marshal(trades)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	params.Set("source_uid", strconv.FormatInt(sourceSubAccountUID, 10))
	params.Set("target_uid", strconv.FormatInt(targetSubAccountUID, 10))
	params.Set("trades", string(values))
	var resp []BlockTradeMoveResponse
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, nonMatchingEPL, http.MethodGet, movePositions, params, &resp)
}

// SimulateBlockTrade checks if a block trade can be executed
func (d *Deribit) SimulateBlockTrade(ctx context.Context, role string, trades []BlockTradeParam) (bool, error) {
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
	values, err := json.Marshal(trades)
	if err != nil {
		return false, err
	}
	params := url.Values{}
	params.Set("role", role)
	params.Set("trades", string(values))
	var resp bool
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestFutures, matchingEPL, http.MethodGet, simulateBlockPosition, params, &resp)
}

// GetLockedStatus retrieves information about locked currencies
func (d *Deribit) GetLockedStatus(ctx context.Context) (*LockedCurrenciesStatus, error) {
	var resp *LockedCurrenciesStatus
	return resp, d.SendHTTPRequest(ctx, exchange.RestSpot, nonMatchingEPL, "public/status", &resp)
}

// EnableCancelOnDisconnect enable Cancel On Disconnect for the connection.
// After enabling Cancel On Disconnect all orders created by the connection will be removed when the connection is closed.
func (d *Deribit) EnableCancelOnDisconnect(ctx context.Context, scope string) (string, error) {
	params := url.Values{}
	if scope != "" {
		params.Set("scope", scope)
	}
	var resp string
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestSpot, nonMatchingEPL, http.MethodGet, "private/enable_cancel_on_disconnect", params, &resp)
}

// ExchangeToken generates a token for a new subject id. This method can be used to switch between subaccounts.
func (d *Deribit) ExchangeToken(ctx context.Context, refreshToken string, subjectID int64) (*RefreshTokenInfo, error) {
	if refreshToken == "" {
		return nil, errRefreshTokenRequired
	}
	if subjectID == 0 {
		return nil, errSubjectIDRequired
	}
	params := url.Values{}
	params.Set("refresh_token", refreshToken)
	params.Set("subject_id", strconv.FormatInt(subjectID, 10))
	var resp *RefreshTokenInfo
	return resp, d.SendHTTPAuthRequest(ctx, exchange.RestSpot, nonMatchingEPL, http.MethodGet, "public/exchange_token", params, &resp)
}

// ForkToken generates a token for a new named session. This method can be used only with session scoped tokens.
func (d *Deribit) ForkToken(ctx context.Context, refreshToken, sessionName string) (*RefreshTokenInfo, error) {
	if refreshToken == "" {
		return nil, errRefreshTokenRequired
	}
	if sessionName == "" {
		return nil, errSessionNameRequired
	}
	params := url.Values{}
	params.Set("refresh_token", refreshToken)
	params.Set("session_name", sessionName)
	var resp *RefreshTokenInfo
	return resp, d.SendHTTPRequest(ctx, exchange.RestSpot, nonMatchingEPL, common.EncodeURLValues("public/fork_token", params), &resp)
}

// GetAssetKind returns the asset type (kind) string representation.
func (d *Deribit) GetAssetKind(assetType asset.Item) string {
	switch assetType {
	case asset.Options:
		return "option"
	case asset.Futures:
		return "future"
	case asset.FutureCombo, asset.OptionCombo, asset.Spot:
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
	case "future_combo":
		return asset.FutureCombo, nil
	case "option_combo":
		return asset.OptionCombo, nil
	default:
		return asset.Empty, nil
	}
}

// getAssetPairByInstrument is able to determine the asset type and currency pair
// based on the received instrument ID
func (d *Deribit) getAssetPairByInstrument(instrument string) (currency.Pair, asset.Item, error) {
	if instrument == "" {
		return currency.EMPTYPAIR, asset.Empty, errInvalidInstrumentName
	}

	var item asset.Item
	// Find the first occurrence of the delimiter and split the instrument string accordingly
	parts := strings.Split(instrument, currency.DashDelimiter)
	switch {
	case len(parts) == 1:
		if i := strings.IndexAny(instrument, currency.UnderscoreDelimiter); i == -1 {
			return currency.EMPTYPAIR, asset.Empty, fmt.Errorf("%w %s", errUnsupportedInstrumentFormat, instrument)
		}
		item = asset.Spot
	case len(parts) == 2:
		item = asset.Futures
	case parts[len(parts)-1] == "C" || parts[len(parts)-1] == "P":
		item = asset.Options
	case len(parts) >= 3:
		// Check for options or other types
		switch parts[1] {
		case "USDC", "USDT":
			item = asset.Futures
		case "FS":
			item = asset.FutureCombo
		default:
			item = asset.OptionCombo
		}
	default:
		return currency.EMPTYPAIR, asset.Empty, fmt.Errorf("%w %s", errUnsupportedInstrumentFormat, instrument)
	}
	cp, err := currency.NewPairFromString(instrument)
	if err != nil {
		return currency.EMPTYPAIR, asset.Empty, err
	}

	return cp, item, nil
}

func getAssetFromPair(currencyPair currency.Pair) (asset.Item, error) {
	currencyPairString := currencyPair.String()
	vals := strings.Split(currencyPairString, currency.DashDelimiter)
	if strings.HasSuffix(currencyPairString, perpString) || len(vals) == 2 {
		return asset.Futures, nil
	} else if len(vals) == 1 {
		if vals = strings.Split(vals[0], currency.UnderscoreDelimiter); len(vals) == 2 {
			return asset.Spot, nil
		}
	}
	added := false
	if len(vals) >= 3 {
		for a := range vals {
			lastVals := strings.Split(vals[a], currency.UnderscoreDelimiter)
			if len(lastVals) > 1 {
				added = true
				if a < len(vals)-1 {
					lastVals = append(lastVals, vals[a+1:]...)
				}
				vals = append(vals[:a], lastVals...)
			}
		}
	}
	length := len(vals)
	if strings.EqualFold(vals[length-1], "C") || strings.EqualFold(vals[length-1], "P") {
		return asset.Options, nil
	}
	if length == 4 {
		if added {
			return asset.FutureCombo, nil
		}
		return asset.OptionCombo, nil
	} else if length >= 5 {
		return asset.OptionCombo, nil
	}
	return asset.Empty, fmt.Errorf("%w currency pair: %v", errUnsupportedInstrumentFormat, currencyPair)
}

func calculateTradingFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	assetType, err := getAssetFromPair(feeBuilder.Pair)
	if err != nil {
		return 0, err
	}
	switch assetType {
	case asset.Futures, asset.FutureCombo:
		switch {
		case strings.HasSuffix(feeBuilder.Pair.String(), "USDC-PERPETUAL"):
			if feeBuilder.IsMaker {
				return 0, nil
			}
			return feeBuilder.Amount * feeBuilder.PurchasePrice * 0.0005, nil
		case strings.HasPrefix(feeBuilder.Pair.String(), currencyBTC),
			strings.HasPrefix(feeBuilder.Pair.String(), currencyETH):
			if strings.HasSuffix(feeBuilder.Pair.String(), perpString) {
				if feeBuilder.IsMaker {
					return 0, nil
				}
				return feeBuilder.Amount * feeBuilder.PurchasePrice * 0.0005, nil
			}
			// weekly futures contracts
			if feeBuilder.IsMaker {
				return feeBuilder.Amount * feeBuilder.PurchasePrice * 0.0001, nil
			}
			return feeBuilder.Amount * feeBuilder.PurchasePrice * 0.0005, nil
		case strings.HasPrefix(feeBuilder.Pair.String(), "SOL"): // perpetual and weekly SOL contracts
			if feeBuilder.IsMaker {
				return feeBuilder.Amount * feeBuilder.PurchasePrice * 0.00002, nil
			}
			return feeBuilder.Amount * feeBuilder.PurchasePrice * 0.0005, nil
		}
	case asset.Options, asset.OptionCombo:
		switch {
		case strings.HasPrefix(feeBuilder.Pair.String(), currencyBTC),
			strings.HasPrefix(feeBuilder.Pair.String(), currencyETH):
			return feeBuilder.Amount * feeBuilder.PurchasePrice * 0.0003, nil
		case strings.HasPrefix(feeBuilder.Pair.String(), "SOL"):
			if feeBuilder.IsMaker {
				return 0, nil
			}
			return feeBuilder.Amount * feeBuilder.PurchasePrice * 0.0003, nil
		}
	}
	return 0, fmt.Errorf("%w asset: %s", asset.ErrNotSupported, assetType.String())
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(price, amount float64) float64 {
	return 0.0003 * price * amount
}

func (d *Deribit) formatFuturesTradablePair(pair currency.Pair) string {
	var instrumentID string
	if result := strings.Split(pair.String(), currency.DashDelimiter); len(result) == 3 {
		instrumentID = strings.Join(result[:2], currency.UnderscoreDelimiter) + currency.DashDelimiter + result[2]
	} else {
		instrumentID = pair.String()
	}
	return instrumentID
}

// optionPairToString formats an options pair as: SYMBOL-EXPIRE-STRIKE-TYPE
// SYMBOL may be a currency (BTC) or a pair (XRP_USDC)
// EXPIRE is DDMMMYY
// STRIKE may include a d for decimal point in linear options
// TYPE is Call or Put
func (d *Deribit) optionPairToString(pair currency.Pair) string {
	initialDelimiter := currency.DashDelimiter
	q := pair.Quote.String()
	if strings.HasPrefix(q, "USDC") && len(q) > 11 { // Linear option
		initialDelimiter = currency.UnderscoreDelimiter
		// Replace a capital D with d for decimal place in Strike price
		// Char 11 is either the hyphen before Strike price or first digit
		q = q[:11] + strings.Replace(q[11:], "D", "d", 1)
	}
	return pair.Base.String() + initialDelimiter + q
}
