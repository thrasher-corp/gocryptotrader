package coinbaseinternational

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// CoinbaseInternational is the overarching type across this package
type CoinbaseInternational struct {
	exchange.Base
}

const (
	coinbaseInternationalAPIURL = "https://api.international.coinbase.com"
	coinbaseAPIVersion          = "/api/v1"

	portfolios = "portfolios/"
)

var (
	errArgumentMustBeInterface = errors.New("argument must be an interface")
	errMissingPortfolioID      = errors.New("missing portfolio identification")
	errNetworkArnID            = errors.New("identifies the blockchain network")
	errMissingTransferID       = errors.New("missing transfer ID")
	errAddressIsRequired       = errors.New("missing address")
	errAssetIdentifierRequired = errors.New("asset identified is required")
	errIndexNameRequired       = errors.New("index name required")
	errGranularityRequired     = errors.New("granularity value is required")
	errStartTimeRequired       = errors.New("start time required")
	errInstrumentIDRequired    = errors.New("instrument information is required")
	errInstrumentTypeRequired  = errors.New("instrument type required")
)

// ListAssets returns a list of all supported assets.
func (co *CoinbaseInternational) ListAssets(ctx context.Context) ([]AssetItemInfo, error) {
	var resp []AssetItemInfo
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "assets", nil, nil, &resp, false)
}

// GetAssetDetails retrieves information for a specific asset.
func (co *CoinbaseInternational) GetAssetDetails(ctx context.Context, assetName currency.Code, assetUUID, assetID string) (*AssetItemInfo, error) {
	path := "assets/"
	switch {
	case !assetName.IsEmpty():
		path += assetName.String()
	case assetUUID != "":
		path += assetUUID
	case assetID != "":
		path += assetID
	default:
		return nil, errAssetIdentifierRequired
	}
	var resp *AssetItemInfo
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &resp, false)
}

// GetSupportedNetworksPerAsset returns a list of supported networks and network information for a specific asset.
func (co *CoinbaseInternational) GetSupportedNetworksPerAsset(ctx context.Context, assetName currency.Code, assetUUID, assetID string) ([]AssetInfoWithSupportedNetwork, error) {
	path := "assets/"
	switch {
	case !assetName.IsEmpty():
		path += assetName.String()
	case assetUUID != "":
		path += assetUUID
	case assetID != "":
		path += assetID
	default:
		return nil, errAssetIdentifierRequired
	}
	var resp []AssetInfoWithSupportedNetwork
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path+"/networks", nil, nil, &resp, false)
}

// GetFeeRateTiers return all the fee rate tiers.
func (co *CoinbaseInternational) GetFeeRateTiers(ctx context.Context) ([]FeeRateInfo, error) {
	var resp []FeeRateInfo
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "fee-rate-tiers", nil, nil, &resp, true)
}

// GetIndexComposition retrieves the latest index composition (metadata) with an ordered set of constituents
func (co *CoinbaseInternational) GetIndexComposition(ctx context.Context, indexName string) (*IndexMetadata, error) {
	if indexName == "" {
		return nil, errIndexNameRequired
	}
	var resp *IndexMetadata
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "index/"+indexName+"/composition", nil, nil, &resp, true)
}

// GetIndexCompositionHistory retrieves a history of index composition records in a descending time order.
// The results are an array of index composition data recorded at different “timestamps”.
func (co *CoinbaseInternational) GetIndexCompositionHistory(ctx context.Context, indexName string, timeFrom time.Time, resultLimit, resultOffset int64) (*IndexMetadata, error) {
	if indexName == "" {
		return nil, errIndexNameRequired
	}
	params := url.Values{}
	if !timeFrom.IsZero() {
		params.Set("timeFrom", strconv.FormatInt(timeFrom.UnixMilli(), 10))
	}
	if resultOffset > 0 {
		params.Set("result_offset", strconv.FormatInt(resultOffset, 10))
	}
	if resultLimit > 0 {
		params.Set("result_limit", strconv.FormatInt(resultLimit, 10))
	}
	var resp *IndexMetadata
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "index/"+indexName+"/composition-history", params, nil, &resp, true)
}

// GetIndexPrice retrieves the latest index price
func (co *CoinbaseInternational) GetIndexPrice(ctx context.Context, indexName string) (*IndexPriceInfo, error) {
	if indexName == "" {
		return nil, errIndexNameRequired
	}
	var resp *IndexPriceInfo
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "index/"+indexName+"/price", nil, nil, &resp, true)
}

// GetIndexCandles retrieves the historical daily index prices in time descending order.
// The daily values are represented as aggregated entries for the day in typical OHLC format.
func (co *CoinbaseInternational) GetIndexCandles(ctx context.Context, indexName, granularity string, start, end time.Time) (*IndexPriceCandlesticks, error) {
	if indexName == "" {
		return nil, errIndexNameRequired
	}
	if granularity == "" {
		return nil, fmt.Errorf("%w, possible values are ONE_DAY and ONE_HOUR", errGranularityRequired)
	}
	if start.IsZero() {
		return nil, errStartTimeRequired
	}
	params := url.Values{}
	params.Set("granularity", granularity)
	params.Set("start", start.Format("2006-01-02T15:04:05Z"))
	if !end.IsZero() {
		params.Set("end", end.Format("2006-01-02T15:04:05Z"))
	}
	var resp *IndexPriceCandlesticks
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "index/"+indexName+"/candles", params, nil, &resp, true)
}

// GetInstruments returns all of the instruments available for trading.
func (co *CoinbaseInternational) GetInstruments(ctx context.Context) ([]InstrumentInfo, error) {
	var resp []InstrumentInfo
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "instruments", nil, nil, &resp, false)
}

// GetInstrumentDetails retrieves market information for a specific instrument.
func (co *CoinbaseInternational) GetInstrumentDetails(ctx context.Context, instrumentName, instrumentUUID, instrumentID string) (*InstrumentInfo, error) {
	path := "instruments/"
	switch {
	case instrumentName != "":
		path += instrumentName
	case instrumentUUID != "":
		path += instrumentUUID
	case instrumentID != "":
		path += instrumentID
	default:
		return nil, errInstrumentIDRequired
	}
	var resp *InstrumentInfo
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &resp, false)
}

// GetQuotePerInstrument retrieves the current quote for a specific instrument.
func (co *CoinbaseInternational) GetQuotePerInstrument(ctx context.Context, instrumentName, instrumentUUID, instrumentID string) (*QuoteInformation, error) {
	path := "instruments/"
	switch {
	case instrumentName != "":
		path += instrumentName
	case instrumentUUID != "":
		path += instrumentUUID
	case instrumentID != "":
		path += instrumentID
	default:
		return nil, errInstrumentIDRequired
	}
	var resp *QuoteInformation
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path+"/quote", nil, nil, &resp, false)
}

// GetDailyTradingVolumes retrieves the trading volumes for each instrument separated by day
func (co *CoinbaseInternational) GetDailyTradingVolumes(ctx context.Context, instruments []string, resultLimit, resultOffset int64, timeFrom time.Time, showOther bool) (*InstrumentsTradingVolumeInfo, error) {
	if len(instruments) == 0 {
		return nil, errInstrumentIDRequired
	}
	params := url.Values{}
	params.Set("instruments", strings.Join(instruments, ","))
	if resultOffset > 0 {
		params.Set("result_offset", strconv.FormatInt(resultOffset, 10))
	}
	if resultLimit > 0 {
		params.Set("result_limit", strconv.FormatInt(resultLimit, 10))
	}
	if !timeFrom.IsZero() {
		params.Set("time_from", timeFrom.Format("2006-01-02T15:04:05Z"))
	}
	if showOther {
		params.Set("show_other", "true")
	}
	var resp *InstrumentsTradingVolumeInfo
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "instruments/volumes/daily", params, nil, &resp, false)
}

// GetAggregatedCandlesDataPerInstrument retrieves a list of aggregated candles data for a given instrument, granularity and time range
func (co *CoinbaseInternational) GetAggregatedCandlesDataPerInstrument(ctx context.Context, instrument string, granularity kline.Interval, start, end time.Time) (*CandlestickDataHistory, error) {
	if instrument == "" {
		return nil, errInstrumentIDRequired
	}
	if start.IsZero() {
		return nil, errStartTimeRequired
	}
	params := url.Values{}
	params.Set("start", start.Format("2006-01-02T15:04:05Z"))
	if granularity != kline.Interval(0) {
		intervalString, err := stringFromInterval(granularity)
		if err != nil {
			return nil, err
		}
		params.Set("granularity", intervalString)
	}
	if !end.IsZero() {
		params.Set("end", end.Format("2006-01-02T15:04:05Z"))
	}
	var resp *CandlestickDataHistory
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "instruments/"+instrument+"/candles", params, nil, &resp, false)
}

var intervalToStringList = []struct {
	Interval kline.Interval
	String   string
}{
	{kline.OneDay, "ONE_DAY"}, {kline.SixHour, "SIX_HOUR"}, {kline.TwoHour, "TWO_HOUR"}, {kline.OneHour, "ONE_HOUR"}, {kline.ThirtyMin, "THIRTY_MINUTE"}, {kline.FifteenMin, "FIFTEEN_MINUTE"}, {kline.FiveMin, "FIVE_MINUTE"}, {kline.OneMin, "ONE_MINUTE"},
}

func stringFromInterval(interval kline.Interval) (string, error) {
	for a := range intervalToStringList {
		if intervalToStringList[a].Interval == interval {
			return intervalToStringList[a].String, nil
		}
	}
	return "", kline.ErrUnsupportedInterval
}

// GetHistoricalFundingRate retrieves the historical funding rates for a specific instrument.
func (co *CoinbaseInternational) GetHistoricalFundingRate(ctx context.Context, instrument string, resultOffset, resultLimit int64) (*FundingRateHistory, error) {
	if instrument == "" {
		return nil, errInstrumentIDRequired
	}
	params := url.Values{}
	if resultOffset > 0 {
		params.Set("result_offset", strconv.FormatInt(resultOffset, 10))
	}
	if resultLimit > 0 {
		params.Set("result_limit", strconv.FormatInt(resultLimit, 10))
	}
	var resp *FundingRateHistory
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "instruments/"+instrument+"/funding", params, nil, &resp, false)
}

// GetPositionOffsets returns all active position offsets
func (co *CoinbaseInternational) GetPositionOffsets(ctx context.Context) (*PositionsOffset, error) {
	var resp *PositionsOffset
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "position-offsets", nil, nil, &resp, false)
}

// CreateOrder creates a new order.
func (co *CoinbaseInternational) CreateOrder(ctx context.Context, arg *OrderRequestParams) (*TradeOrder, error) {
	if arg == nil || *arg == (OrderRequestParams{}) {
		return nil, common.ErrNilPointer
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.BaseSize <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if arg.Price <= 0 {
		return nil, order.ErrPriceBelowMin
	}
	if arg.OrderType == "" {
		return nil, order.ErrUnsupportedOrderType
	}
	if arg.ClientOrderID == "" {
		return nil, fmt.Errorf("%w, client_order_id is required", order.ErrOrderIDNotSet)
	}
	if arg.TimeInForce == "" {
		return nil, fmt.Errorf("%w: time-in-force is missing", order.ErrInvalidTimeInForce)
	}
	var resp *TradeOrder
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "orders", nil, arg, &resp, true)
}

// GetOpenOrders returns a list of active orders resting on the order book matching the requested criteria. Does not return any rejected, cancelled, or fully filled orders as they are not active.
func (co *CoinbaseInternational) GetOpenOrders(ctx context.Context, portfolioUUID, portfolioID, instrument, clientOrderID, eventType string, refDateTime time.Time, resultOffset, resultLimit int64) (*OrderItemDetail, error) {
	params := url.Values{}
	switch {
	case portfolioID != "":
		params.Set("portfolio", portfolioID)
	case portfolioUUID != "":
		params.Set("portfolio", portfolioUUID)
	}
	if instrument != "" {
		params.Set("instrument", instrument)
	}
	if clientOrderID != "" {
		params.Set("client_order_id", clientOrderID)
	}
	if eventType != "" {
		params.Set("event_type", eventType)
	}
	if !refDateTime.IsZero() {
		params.Set("ref_datetime", refDateTime.String())
	}
	if resultOffset > 0 {
		params.Set("result_offset", strconv.FormatInt(resultOffset, 10))
	}
	if resultLimit > 0 {
		params.Set("result_limit", strconv.FormatInt(resultLimit, 10))
	}
	var resp *OrderItemDetail
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "orders", params, nil, &resp, true)
}

// CancelOrders cancels all orders matching the requested criteria.
func (co *CoinbaseInternational) CancelOrders(ctx context.Context, portfolioID, portfolioUUID, instrument string) ([]OrderItem, error) {
	params := url.Values{}
	switch {
	case portfolioID != "":
		params.Set("portfolio", portfolioID)
	case portfolioUUID != "":
		params.Set("portfolio", portfolioUUID)
	default:
		return nil, fmt.Errorf("%w %w", request.ErrAuthRequestFailed, errMissingPortfolioID)
	}
	if instrument != "" {
		params.Set("instrument", instrument)
	}
	var resp []OrderItem
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, "orders", params, nil, &resp, true)
}

// ModifyOpenOrder modifies an open order.
func (co *CoinbaseInternational) ModifyOpenOrder(ctx context.Context, orderID string, arg *ModifyOrderParam) (*OrderItem, error) {
	if arg == nil || *arg == (ModifyOrderParam{}) {
		return nil, common.ErrNilPointer
	}
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *OrderItem
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPut, "orders/"+orderID, nil, arg, &resp, true)
}

// GetOrderDetail retrieves a single order. The order retrieved can be either active or inactive.
func (co *CoinbaseInternational) GetOrderDetail(ctx context.Context, orderID string) (*OrderItem, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *OrderItem
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "orders/"+orderID, nil, nil, &resp, true)
}

// CancelTradeOrder cancels a single open order.
func (co *CoinbaseInternational) CancelTradeOrder(ctx context.Context, orderID, clientOrderID, portfolioID, portfolioUUID string) (*OrderItem, error) {
	switch {
	case orderID != "":
	case clientOrderID != "":
		orderID = clientOrderID
	default:
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	switch {
	case portfolioID != "":
		params.Set("portfolio", portfolioID)
	case portfolioUUID != "":
		params.Set("portfolio", portfolioUUID)
	default:
		return nil, fmt.Errorf("%w %w", request.ErrAuthRequestFailed, errMissingPortfolioID)
	}
	var resp *OrderItem
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, "orders/"+orderID, params, nil, &resp, true)
}

// GetAllUserPortfolios returns all of the user's portfolios.
func (co *CoinbaseInternational) GetAllUserPortfolios(ctx context.Context) ([]PortfolioItem, error) {
	var resp []PortfolioItem
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "portfolios", nil, nil, &resp, true)
}

// CreatePortfolio create a new portfolio. Request will fail if no name is provided or if user already has max number of portfolios.
// Max number of portfolios is 20.
func (co *CoinbaseInternational) CreatePortfolio(ctx context.Context, portfolioName string) (*PortfolioItem, error) {
	var resp *PortfolioItem
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "portfolios", nil, &struct {
		Name string `json:"name,omitempty"`
	}{Name: portfolioName}, &resp, true)
}

// GetUserPortfolio retrieves the user's specified portfolio.
func (co *CoinbaseInternational) GetUserPortfolio(ctx context.Context, portfolioID string) (*PortfolioItem, error) {
	if portfolioID == "" {
		return nil, errMissingPortfolioID
	}
	var resp *PortfolioItem
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, portfolios+portfolioID, nil, nil, &resp, true)
}

// PatchPortfolio update parameters for existing portfolio
func (co *CoinbaseInternational) PatchPortfolio(ctx context.Context, arg *PatchPortfolioParams) (*PortfolioItem, error) {
	if arg == nil || *arg == (PatchPortfolioParams{}) {
		return nil, common.ErrEmptyParams
	}
	var resp *PortfolioItem
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPatch, "portfolios", nil, arg, &resp, true)
}

// UpdatePortfolio update existing user portfolio
func (co *CoinbaseInternational) UpdatePortfolio(ctx context.Context, portfolioID, portfolioUniqueName string) (*PortfolioItem, error) {
	if portfolioID == "" {
		return nil, errMissingPortfolioID
	}
	var resp *PortfolioItem
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPut, portfolios+portfolioID, nil, &struct {
		Name string `json:"name,omitempty"`
	}{Name: portfolioUniqueName}, &resp, true)
}

// GetPortfolioDetails retrieves the summary, positions, and balances of a portfolio.
func (co *CoinbaseInternational) GetPortfolioDetails(ctx context.Context, portfolioID, portfolioUUID string) (*PortfolioDetail, error) {
	if portfolioID == "" && portfolioUUID == "" {
		return nil, errMissingPortfolioID
	}
	var pID string
	if portfolioID != "" {
		pID = portfolioID
	}
	if portfolioUUID != "" {
		pID = portfolioUUID
	}
	var resp *PortfolioDetail
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, portfolios+pID+"/detail", nil, nil, &resp, true)
}

// GetPortfolioSummary retrieves the high level overview of a portfolio.
func (co *CoinbaseInternational) GetPortfolioSummary(ctx context.Context, portfolioUUID, portfolioID string) (*PortfolioSummary, error) {
	var path string
	switch {
	case portfolioUUID != "":
		path = portfolios + portfolioUUID + "/summary"
	case portfolioID != "":
		path = portfolios + portfolioID + "/summary"
	default:
		return nil, errMissingPortfolioID
	}
	var resp *PortfolioSummary
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &resp, true)
}

// ListPortfolioBalances returns all of the balances for a given portfolio.
func (co *CoinbaseInternational) ListPortfolioBalances(ctx context.Context, portfolioUUID, portfolioID string) ([]PortfolioBalance, error) {
	var path string
	switch {
	case portfolioUUID != "":
		path = portfolios + portfolioUUID + "/balances"
	case portfolioID != "":
		path = portfolios + portfolioID + "/balances"
	default:
		return nil, errMissingPortfolioID
	}
	var resp []PortfolioBalance
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &resp, true)
}

// GetPortfolioAssetBalance retrieves the balance for a given portfolio and asset.
func (co *CoinbaseInternational) GetPortfolioAssetBalance(ctx context.Context, portfolioUUID, portfolioID string, ccy currency.Code) (*PortfolioBalance, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var path string
	switch {
	case portfolioUUID != "":
		path = portfolios + portfolioUUID + "/balances/" + ccy.String()
	case portfolioID != "":
		path = portfolios + portfolioID + "/balances/" + ccy.String()
	default:
		return nil, errMissingPortfolioID
	}
	var resp *PortfolioBalance
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &resp, true)
}

// GetActiveLoansForPortfolio retrieves all loan info for a given portfolio.
func (co *CoinbaseInternational) GetActiveLoansForPortfolio(ctx context.Context, portfolioUUID, portfolioID string) (*PortfolioLoanDetail, error) {
	var path string
	switch {
	case portfolioUUID != "":
		path = portfolios + portfolioUUID + "/loans"
	case portfolioID != "":
		path = portfolios + portfolioID + "/loans"
	default:
		return nil, errMissingPortfolioID
	}
	var resp *PortfolioLoanDetail
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &resp, true)
}

// GetLoanInfoForPortfolioAsset retrieves the loan info for a given portfolio and asset.
func (co *CoinbaseInternational) GetLoanInfoForPortfolioAsset(ctx context.Context, portfolioUUID, portfolioID string, asset currency.Code) (*PortfolioLoanDetail, error) {
	var path string
	switch {
	case portfolioUUID != "":
		path = portfolios + portfolioUUID + "/loans/"
	case portfolioID != "":
		path = portfolios + portfolioID + "/loans/"
	default:
		return nil, errMissingPortfolioID
	}
	if asset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *PortfolioLoanDetail
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path+asset.String(), nil, nil, &resp, true)
}

// AcquireRepayLoan acquire or repay loan for a given portfolio and asset.
// Action possible values: [ACQUIRE, REPAY]
func (co *CoinbaseInternational) AcquireRepayLoan(ctx context.Context, portfolioUUID, portfolioID string, asset currency.Code, arg *LoanActionAmountParam) (*AcquireRepayLoanResponse, error) {
	if arg == nil || *arg == (LoanActionAmountParam{}) {
		return nil, common.ErrEmptyParams
	}
	var path string
	switch {
	case portfolioUUID != "":
		path = portfolios + portfolioUUID + "/loans/"
	case portfolioID != "":
		path = portfolios + portfolioID + "/loans/"
	default:
		return nil, errMissingPortfolioID
	}
	if asset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *AcquireRepayLoanResponse
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path+asset.String(), nil, arg, &resp, true)
}

// PreviewLoanUpdate preview acquire or repay loan for a given portfolio and asset.
func (co *CoinbaseInternational) PreviewLoanUpdate(ctx context.Context, portfolioUUID, portfolioID string, asset currency.Code, arg *LoanActionAmountParam) (*LoanUpdate, error) {
	if arg == nil || *arg == (LoanActionAmountParam{}) {
		return nil, common.ErrEmptyParams
	}
	var path string
	switch {
	case portfolioUUID != "":
		path = portfolios + portfolioUUID + "/loans/"
	case portfolioID != "":
		path = portfolios + portfolioID + "/loans/"
	default:
		return nil, errMissingPortfolioID
	}
	if asset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *LoanUpdate
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path+asset.String()+"/preview", nil, arg, &resp, true)
}

// ViewMaxLoanAvailability view the maximum amount of loan that could be acquired now
func (co *CoinbaseInternational) ViewMaxLoanAvailability(ctx context.Context, portfolioUUID, portfolioID string, asset currency.Code) (*MaxLoanAvailability, error) {
	var path string
	switch {
	case portfolioUUID != "":
		path = portfolios + portfolioUUID + "/loans/"
	case portfolioID != "":
		path = portfolios + portfolioID + "/loans/"
	default:
		return nil, errMissingPortfolioID
	}
	if asset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *MaxLoanAvailability
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path+asset.String()+"/availability", nil, nil, &resp, true)
}

// ListPortfolioPositions returns all of the positions for a given portfolio.
func (co *CoinbaseInternational) ListPortfolioPositions(ctx context.Context, portfolioUUID, portfolioID string) ([]PortfolioPosition, error) {
	var path string
	switch {
	case portfolioUUID != "":
		path = portfolios + portfolioUUID + "/positions"
	case portfolioID != "":
		path = portfolios + portfolioID + "/positions"
	default:
		return nil, errMissingPortfolioID
	}
	var resp []PortfolioPosition
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &resp, true)
}

// GetPortfolioInstrumentPosition retrieves the position for a given portfolio and symbol.
func (co *CoinbaseInternational) GetPortfolioInstrumentPosition(ctx context.Context, portfolioUUID, portfolioID string, instrument currency.Pair) (*PortfolioPosition, error) {
	if instrument.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var path string
	switch {
	case portfolioUUID != "":
		path = portfolios + portfolioUUID + "/positions/" + instrument.String()
	case portfolioID != "":
		path = portfolios + portfolioID + "/positions/" + instrument.String()
	default:
		return nil, errMissingPortfolioID
	}
	var resp *PortfolioPosition
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &resp, true)
}

// GetTotalOpenPositionLimitPortfolio retrieves the total open position limit across instruments for a given portfolio.
func (co *CoinbaseInternational) GetTotalOpenPositionLimitPortfolio(ctx context.Context, portfolioID string) (*PortfolioPositionLimit, error) {
	if portfolioID == "" {
		return nil, errMissingPortfolioID
	}
	var resp *PortfolioPositionLimit
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, portfolios+portfolioID+"/position-limits", nil, nil, &resp, true)
}

// GetFillsByPortfolio returns fills for specified portfolios or fills for all portfolios if none are provided
func (co *CoinbaseInternational) GetFillsByPortfolio(ctx context.Context, portfolioUUID, orderID, clientOrderID string, resultLimit, resultOffset int64, refDateTime, timeFrom time.Time) (*PortfolioFill, error) {
	params := url.Values{}
	if portfolioUUID != "" {
		params.Set("portfolios", portfolioUUID)
	}
	if orderID != "" {
		params.Set("order_id", orderID)
	}
	if clientOrderID != "" {
		params.Set("client_order_id", clientOrderID)
	}
	if !refDateTime.IsZero() {
		params.Set("ref_datetime", refDateTime.Format("2006-01-02T15:04:05Z"))
	}
	if resultLimit > 0 {
		params.Set("result_limit", strconv.FormatInt(resultLimit, 10))
	}
	if resultOffset > 0 {
		params.Set("result_offset", strconv.FormatInt(resultOffset, 10))
	}
	if !timeFrom.IsZero() {
		params.Set("time_from", timeFrom.Format("2006-01-02T15:04:05Z"))
	}
	var resp *PortfolioFill
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "portfolios/fills", params, nil, &resp, true)
}

// ListPortfolioFills returns all of the fills for a given portfolio.
func (co *CoinbaseInternational) ListPortfolioFills(ctx context.Context, portfolioUUID, portfolioID string) ([]PortfolioFill, error) {
	var path string
	switch {
	case portfolioUUID != "":
		path = portfolios + portfolioUUID + "/fills"
	case portfolioID != "":
		path = portfolios + portfolioID + "/fills"
	default:
		return nil, errMissingPortfolioID
	}
	var resp []PortfolioFill
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &resp, true)
}

// EnableDisablePortfolioCrossCollateral enable or disable the cross collateral feature for the portfolio, which allows the portfolio to use non-USDC assets as collateral for margin trading.
func (co *CoinbaseInternational) EnableDisablePortfolioCrossCollateral(ctx context.Context, portfolioUUID, portfolioID string, enabled bool) (*PortfolioItem, error) {
	var path string
	switch {
	case portfolioUUID != "":
		path = portfolios + portfolioUUID + "/cross-collateral-enabled"
	case portfolioID != "":
		path = portfolios + portfolioID + "/cross-collateral-enabled"
	default:
		return nil, errMissingPortfolioID
	}
	var resp *PortfolioItem
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, &struct {
		Enabled bool `json:"enabled,omitempty"`
	}{
		Enabled: enabled,
	}, &resp, true)
}

// EnableDisablePortfolioAutoMarginMode enable or disable the auto margin feature,
// which lets the portfolio automatically post margin amounts required to exceed the high leverage position restrictions.
func (co *CoinbaseInternational) EnableDisablePortfolioAutoMarginMode(ctx context.Context, portfolioUUID, portfolioID string, enabled bool) (*PortfolioItem, error) {
	var path string
	switch {
	case portfolioUUID != "":
		path = portfolios + portfolioUUID + "/auto-margin-enabled"
	case portfolioID != "":
		path = portfolios + portfolioID + "/auto-margin-enabled"
	default:
		return nil, errMissingPortfolioID
	}
	var resp *PortfolioItem
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, &struct {
		Enabled bool `json:"enabled,omitempty"`
	}{
		Enabled: enabled,
	}, &resp, true)
}

// SetPortfolioMarginOverride specify the margin override value for a portfolio to either increase notional requirements or opt-in to higher leverage.
func (co *CoinbaseInternational) SetPortfolioMarginOverride(ctx context.Context, arg *PortfolioMarginOverrideParams) (*PortfolioMarginOverrideResponse, error) {
	if arg == nil || *arg == (PortfolioMarginOverrideParams{}) {
		return nil, common.ErrEmptyParams
	}
	var resp *PortfolioMarginOverrideResponse
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "portfolios/margin", nil, arg, &resp, true)
}

// TransferFundsBetweenPortfolios transfer assets from one portfolio to another.
func (co *CoinbaseInternational) TransferFundsBetweenPortfolios(ctx context.Context, arg *TransferFundsBetweenPortfoliosParams) (bool, error) {
	if arg == nil || *arg == (TransferFundsBetweenPortfoliosParams{}) {
		return false, common.ErrEmptyParams
	}
	resp := &struct {
		Success bool `json:"success"`
	}{}
	return resp.Success, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "portfolios/transfer", nil, arg, &resp, true)
}

// TransferPositionsBetweenPortfolios transfer an existing position from one portfolio to another.
// The position transfer must fulfill the same portfolio-level margin requirements as submitting a new order on the opposite side for the sender's portfolio and a new order on the same side for the recipient's portfolio.
// Additionally, organization-level requirements must be satisfied when evaluating the outcome of the position transfer.
func (co *CoinbaseInternational) TransferPositionsBetweenPortfolios(ctx context.Context, arg *TransferPortfolioParams) (bool, error) {
	if arg == nil || *arg == (TransferPortfolioParams{}) {
		return false, common.ErrEmptyParams
	}
	resp := &struct {
		Success bool `json:"success"`
	}{}
	return resp.Success, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "portfolios/transfer-position", nil, arg, &resp, true)
}

// GetPortfolioFeeRates retrieves the Perpetual Future and Spot fee rate tiers for the user.
func (co *CoinbaseInternational) GetPortfolioFeeRates(ctx context.Context) ([]PortfolioFeeRate, error) {
	var resp []PortfolioFeeRate
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "portfolios/fee-rates", nil, nil, &resp, true)
}

// GetYourRanking retrieve your volume rankings for maker, taker, and total volume.
// Instrument type allowed values: SPOT, PERPETUAL_FUTURE
// period allowed values: YESTERDAY, LAST_7_DAYS, THIS_MONTH, LAST_30_DAYS, LAST_MONTH. Default: THIS_MONTH
func (co *CoinbaseInternational) GetYourRanking(ctx context.Context, instrumentType, period string, instruments []string) (*VolumeRankingInfo, error) {
	if instrumentType == "" {
		return nil, errInstrumentTypeRequired
	}
	params := url.Values{}
	params.Set("instrument_type", instrumentType)
	if period != "" {
		params.Set("period", period)
	}
	if len(instruments) > 0 {
		for i := range instruments {
			params.Add("instruments", instruments[i])
		}
	}
	var resp *VolumeRankingInfo
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "rankings/statistics", params, nil, &resp, true)
}

// ListMatchingTransfers represents a list of transfer based on the query
// type: possible values DEPOSIT, WITHDRAW, REBATE, STIPEND
// status: possible value PROCESSED, NEW, FAILED, STARTED
func (co *CoinbaseInternational) ListMatchingTransfers(ctx context.Context, portfolioUUID, portfolioID, status, transferType string, resultLimit, resultOffset int64, timeFrom, timeTo time.Time) (*Transfers, error) {
	params := url.Values{}
	switch {
	case portfolioUUID != "":
		params.Set("portfolio", portfolioUUID)
	case portfolioID != "":
		params.Set("portfolio", portfolioID)
	}
	if resultOffset > 0 {
		params.Set("result_offset", strconv.FormatInt(resultOffset, 10))
	}
	if resultLimit > 0 {
		params.Set("result_limit", strconv.FormatInt(resultLimit, 10))
	}
	if status != "" {
		params.Set("status", status)
	}
	if transferType != "" {
		params.Set("type", transferType)
	}
	if !timeFrom.IsZero() {
		params.Set("time_from", timeFrom.String())
	}
	if !timeTo.IsZero() {
		params.Set("time_to", timeTo.String())
	}
	var resp *Transfers
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "transfers", params, nil, &resp, true)
}

// GetTransfer returns a single transfer instance
func (co *CoinbaseInternational) GetTransfer(ctx context.Context, transferID string) (*FundTransfer, error) {
	if transferID == "" {
		return nil, errMissingTransferID
	}
	var resp *FundTransfer
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "transfers/"+transferID, nil, nil, &resp, true)
}

// WithdrawToCryptoAddress withdraws a crypto fund to crypto address
func (co *CoinbaseInternational) WithdrawToCryptoAddress(ctx context.Context, arg *WithdrawCryptoParams) (*WithdrawalResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Address == "" {
		return nil, errAddressIsRequired
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if arg.AssetIdentifier == "" {
		return nil, errAssetIdentifierRequired
	}
	var resp *WithdrawalResponse
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "transfers/withdraw", nil, arg, &resp, true)
}

// CreateCryptoAddress created a new crypto address
func (co *CoinbaseInternational) CreateCryptoAddress(ctx context.Context, arg *CryptoAddressParam) (*CryptoAddressInfo, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AssetIdentifier == "" {
		return nil, errAssetIdentifierRequired
	}
	if arg.Portfolio == "" {
		return nil, errMissingPortfolioID
	}
	if arg.NetworkArnID == "" {
		return nil, errNetworkArnID
	}
	var resp *CryptoAddressInfo
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "transfers/address", nil, arg, &resp, true)
}

// CreateCounterPartyID create counterparty Id
func (co *CoinbaseInternational) CreateCounterPartyID(ctx context.Context, portfolio string) (*CounterpartyIDCreationResponse, error) {
	if portfolio == "" {
		return nil, errMissingPortfolioID
	}
	var resp *CounterpartyIDCreationResponse
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "transfers/create-counterparty-id", nil, &struct {
		Portfolio string `json:"portfolio,omitempty"`
	}{Portfolio: portfolio}, &resp, true)
}

// ValidateCounterpartyID validate counterparty Id
func (co *CoinbaseInternational) ValidateCounterpartyID(ctx context.Context, counterpartyID string) (*CounterpartyValidationResponse, error) {
	if counterpartyID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *CounterpartyValidationResponse
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "transfers/validate-counterparty-id", nil, &struct {
		CounterpartyID string `json:"counterparty_id,omitempty"`
	}{CounterpartyID: counterpartyID}, &resp, true)
}

// WithdrawToCounterpartyID withdraw to counterparty Id
func (co *CoinbaseInternational) WithdrawToCounterpartyID(ctx context.Context, arg *AssetCounterpartyWithdrawalResponse) (*CounterpartyWithdrawalResponse, error) {
	if arg == nil || *arg == (AssetCounterpartyWithdrawalResponse{}) {
		return nil, common.ErrEmptyParams
	}
	var resp *CounterpartyWithdrawalResponse
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "transfers/withdraw/counterparty", nil, arg, &resp, true)
}

// GetCounterpartyWithdrawalLimit retrieves counterparty withdrawal limit within coinbase transfer network
func (co *CoinbaseInternational) GetCounterpartyWithdrawalLimit(ctx context.Context, portfolio, assetIdentifier string) (*CounterpartyWithdrawalLimi, error) {
	if portfolio == "" {
		return nil, errMissingPortfolioID
	}
	if assetIdentifier == "" {
		return nil, errAssetIdentifierRequired
	}
	var resp *CounterpartyWithdrawalLimi
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "transfers/withdraw/"+portfolio+"/"+assetIdentifier, nil, nil, &resp, true)
}

// SendHTTPRequest sends a public HTTP request.
func (co *CoinbaseInternational) SendHTTPRequest(ctx context.Context, ep exchange.URL, method, path string, params url.Values, data, result interface{}, authenticated bool) error {
	endpoint, err := co.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	urlPath := endpoint + coinbaseAPIVersion + "/" + path
	if params != nil {
		urlPath = common.EncodeURLValues(urlPath, params)
	}
	requestType := request.AuthType(request.UnauthenticatedRequest)
	var creds *account.Credentials
	if authenticated {
		creds, err = co.GetCredentials(ctx)
		if err != nil {
			return err
		}
		requestType = request.AuthenticatedRequest
	}

	var payload []byte
	if data != nil {
		if reflect.ValueOf(data).Kind() != reflect.Ptr {
			return errArgumentMustBeInterface
		}
		payload, err = json.Marshal(data)
		if err != nil {
			return err
		}
	}
	intrim := json.RawMessage{}
	err = co.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		timestamp := time.Now()
		headers := make(map[string]string)
		headers["Content-Type"] = "application/json"
		headers["Accept"] = "application/json"
		if authenticated {
			headers["CB-ACCESS-KEY"] = creds.Key
			headers["CB-ACCESS-PASSPHRASE"] = creds.ClientID
			headers["CB-ACCESS-TIMESTAMP"] = strconv.FormatInt(timestamp.Unix(), 10)
			signatureString := headers["CB-ACCESS-TIMESTAMP"] + method + coinbaseAPIVersion + "/" + path + string(payload)
			var hmac []byte
			hmac, err = crypto.GetHMAC(crypto.HashSHA256,
				[]byte(signatureString),
				[]byte(creds.Secret))
			if err != nil {
				return nil, err
			}
			headers["CB-ACCESS-SIGN"] = crypto.Base64Encode(hmac)
		}
		return &request.Item{
			Method:        method,
			Path:          urlPath,
			Headers:       headers,
			Result:        &intrim,
			Body:          bytes.NewBuffer(payload),
			Verbose:       co.Verbose,
			HTTPDebugging: co.HTTPDebugging,
			HTTPRecording: co.HTTPRecording,
		}, nil
	}, requestType)
	if err != nil {
		return err
	}
	errorMessage := &struct {
		Title  string `json:"title,omitempty"`
		Status int64  `json:"status,omitempty"`
	}{}
	err = json.Unmarshal(intrim, errorMessage)
	if errorMessage.Status != 0 {
		if authenticated {
			return fmt.Errorf("%v %w status: %d title: %s", err, request.ErrAuthRequestFailed, errorMessage.Status, errorMessage.Title)
		}
		return fmt.Errorf("status: %d Title: %s", errorMessage.Status, errorMessage.Title)
	}
	if result == nil {
		return nil
	}
	return json.Unmarshal(intrim, result)
}

// OrderTypeString returns a string representation of order.Type
func OrderTypeString(oType order.Type) (string, error) {
	switch oType {
	case order.Limit, order.Market, order.Stop:
		return oType.String(), nil
	case order.StopLimit:
		return "STOP_LIMIT", nil
	default:
		return "", order.ErrUnsupportedOrderType
	}
}

// GetFee returns an estimate of fee based on type of transaction
func (co *CoinbaseInternational) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	var err error
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee, err = co.calculateTradingFee(
			ctx,
			feeBuilder.Pair.Base,
			feeBuilder.Pair.Quote,
			feeBuilder.PurchasePrice,
			feeBuilder.Amount,
			feeBuilder.IsMaker)
		if err != nil {
			return 0, err
		}
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	if fee < 0 {
		fee = 0
	}
	return fee, nil
}

func (co *CoinbaseInternational) calculateTradingFee(ctx context.Context, base, quote currency.Code, purchasePrice, amount float64, isMaker bool) (float64, error) {
	fees, err := co.GetAllUserPortfolios(ctx)
	if err != nil {
		return 0, err
	}
	for x := range fees {
		if strings.EqualFold(fees[x].Name, currency.Pair{Base: base, Delimiter: "-", Quote: quote}.String()) {
			if isMaker {
				return fees[x].MakerFeeRate.Float64() * amount * purchasePrice, nil
			}
			return fees[x].TakerFeeRate.Float64() * amount * purchasePrice, nil
		}
	}
	if isMaker {
		return 0.018 * amount * purchasePrice, nil
	}
	return 0.02 * amount * purchasePrice, nil
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(price, amount float64) float64 {
	return 0.02 * price * amount
}
