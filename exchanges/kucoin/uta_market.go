package kucoin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Unified trading account market data errors
var (
	errUTAOrderbookLevelLength     = errors.New("UTA orderbook level must hold a price and a size, and optionally an RPI size")
	errPositionTierDataRequired    = errors.New("position tier data type is required")
	errHistoricOpenInterestSymbols = errors.New("historic open interest takes exactly one symbol")
)

// GetUTAAnnouncements returns a page of KuCoin's announcements (Get Announcements)
func (e *Exchange) GetUTAAnnouncements(ctx context.Context, req *UTAAnnouncementsRequest) (*UTAAnnouncementsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.Language != "" {
		params.Set("language", req.Language)
	}
	if req.Type != "" {
		params.Set("type", req.Type)
	}
	if req.PageNumber != 0 {
		params.Set("pageNumber", strconv.FormatUint(req.PageNumber, 10))
	}
	if req.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	if err := setTimeRange(params, "startTime", "endTime", req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	var resp *UTAAnnouncementsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getUTAAnnouncementsEPL, common.EncodeURLValues("/ua/v2/market/announcement", params), &resp)
}

// GetUTACurrency returns a currency's configuration on each chain, or on the chain given (Get Currency)
func (e *Exchange) GetUTACurrency(ctx context.Context, ccy currency.Code, chain string) (*UTACurrencyResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.Upper().String())
	if chain != "" {
		params.Set("chain", chain)
	}
	var resp *UTACurrencyResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getUTACurrencyEPL, common.EncodeURLValues("/ua/v2/market/currency", params), &resp)
}

// GetUTACurrencies returns the configuration of the given currencies, or of every currency, on each chain, or on the
// chain given (Get Currencies)
func (e *Exchange) GetUTACurrencies(ctx context.Context, chain string, currencies []currency.Code) ([]UTACurrencyResponse, error) {
	params := url.Values{}
	if chain != "" {
		params.Set("chain", chain)
	}
	for _, c := range currencies {
		params.Add("currencyList", c.Upper().String())
	}
	var resp []UTACurrencyResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getUTACurrenciesEPL, common.EncodeURLValues("/ua/v2/asset/currencies", params), &resp)
}

// GetUTATicker returns the 24 hour ticker of a symbol, or of every symbol, of a product type (Get Ticker)
func (e *Exchange) GetUTATicker(ctx context.Context, tradeType, symbol string) (*UTATickerResponse, error) {
	if tradeType == "" {
		return nil, errTradeTypeMissing
	}
	params := url.Values{}
	params.Set("tradeType", tradeType)
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *UTATickerResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getUTATickerEPL, common.EncodeURLValues("/ua/v2/market/ticker", params), &resp)
}

// GetUTASymbol returns the trading rules of a symbol, or of every symbol, of a product type (Get Symbol)
func (e *Exchange) GetUTASymbol(ctx context.Context, tradeType, symbol string) (*UTASymbolResponse, error) {
	if tradeType == "" {
		return nil, errTradeTypeMissing
	}
	params := url.Values{}
	params.Set("tradeType", tradeType)
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *UTASymbolResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getUTASymbolEPL, common.EncodeURLValues("/ua/v2/market/instrument", params), &resp)
}

// GetUTAKlines returns a symbol's candles, newest first. Spot returns up to 1500 candles a request and futures up to
// 200; futures candles shorter than 8 hours start from 2025-01-01 (Get Klines)
func (e *Exchange) GetUTAKlines(ctx context.Context, req *UTAKlinesRequest) (*UTAKlinesResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if req.TradeType == "" {
		return nil, errTradeTypeMissing
	}
	if req.Interval == "" {
		return nil, kline.ErrInvalidInterval
	}
	params := url.Values{}
	params.Set("symbol", req.Symbol)
	params.Set("tradeType", req.TradeType)
	if req.KlineType != "" {
		params.Set("klineType", req.KlineType)
	}
	params.Set("interval", req.Interval)
	if !req.StartAt.IsZero() && !req.EndAt.IsZero() {
		if err := common.StartEndTimeCheck(req.StartAt, req.EndAt); err != nil {
			return nil, err
		}
	}
	// Kline windows are in seconds, unlike the other UTA market windows
	if !req.StartAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(req.StartAt.Unix(), 10))
	}
	if !req.EndAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(req.EndAt.Unix(), 10))
	}
	var resp *UTAKlinesResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getUTAKlinesEPL, common.EncodeURLValues("/ua/v2/market/kline", params), &resp)
}

// GetUTAOrderbook returns a symbol's orderbook, aggregated by price (Get OrderBook)
func (e *Exchange) GetUTAOrderbook(ctx context.Context, req *UTAOrderbookRequest) (*UTAOrderbookResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.TradeType == "" {
		return nil, errTradeTypeMissing
	}
	if req.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if req.Limit == "" {
		return nil, fmt.Errorf("%w: limit is required", errInvalidLimit)
	}
	params := url.Values{}
	params.Set("tradeType", req.TradeType)
	params.Set("symbol", req.Symbol)
	params.Set("limit", req.Limit)
	if req.RPIFilter {
		params.Set("rpiFilter", "1")
	}
	var resp *UTAOrderbookResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAOrderbookEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/market/orderbook", params), nil, &resp)
}

// GetUTAIndexPrice returns a page of a futures contract's index prices, newest first, with the exchange prices each
// one weighs (Get Index Price)
func (e *Exchange) GetUTAIndexPrice(ctx context.Context, req *UTAIndexPriceRequest) (*UTAIndexPriceResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", req.Symbol)
	if err := setTimeRange(params, "startAt", "endAt", req.StartAt, req.EndAt); err != nil {
		return nil, err
	}
	if req.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	if req.LastID != 0 {
		params.Set("lastId", strconv.FormatUint(req.LastID, 10))
	}
	var resp *UTAIndexPriceResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getUTAIndexPriceEPL, common.EncodeURLValues("/ua/v2/market/index-price", params), &resp)
}

// GetUTATrades returns a symbol's latest 100 public trades (Get Trades)
func (e *Exchange) GetUTATrades(ctx context.Context, tradeType, symbol string) (*UTATradesResponse, error) {
	if tradeType == "" {
		return nil, errTradeTypeMissing
	}
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("tradeType", tradeType)
	params.Set("symbol", symbol)
	var resp *UTATradesResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getUTATradesEPL, common.EncodeURLValues("/ua/v2/market/trade", params), &resp)
}

// GetUTAPositionTiers returns the unified account's margin borrowing tiers or futures risk limit tiers (Get Position
// Tiers)
func (e *Exchange) GetUTAPositionTiers(ctx context.Context, req *UTAPositionTiersRequest) ([]UTAPositionTier, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.TradeType == "" {
		return nil, errTradeTypeMissing
	}
	if req.MarginMode == "" {
		return nil, fmt.Errorf("%w: margin mode is required", margin.ErrInvalidMarginType)
	}
	if req.Data == "" {
		return nil, errPositionTierDataRequired
	}
	if req.AccountType == "" {
		return nil, errAccountTypeMissing
	}
	if req.Data == "BORROW" && len(req.Currencies) == 0 {
		return nil, currency.ErrCurrencyCodesEmpty
	}
	if req.Data == "RISK_LIMIT" && len(req.Symbols) == 0 {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("tradeType", req.TradeType)
	if len(req.Currencies) != 0 {
		codes := make([]string, len(req.Currencies))
		for i, c := range req.Currencies {
			codes[i] = c.Upper().String()
		}
		params.Set("currency", strings.Join(codes, ","))
	}
	params.Set("marginMode", req.MarginMode)
	params.Set("data", req.Data)
	params.Set("accountType", req.AccountType)
	if len(req.Symbols) != 0 {
		params.Set("symbol", strings.Join(req.Symbols, ","))
	}
	var resp []UTAPositionTier
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getUTAPositionTiersEPL, common.EncodeURLValues("/ua/v2/market/position-tiers", params), &resp)
}

// GetUTACollateralRatio returns each currency's collateral discount tiers in the unified account (Get Collateral Ratio)
func (e *Exchange) GetUTACollateralRatio(ctx context.Context) ([]UTACollateralRatio, error) {
	var resp []UTACollateralRatio
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getUTACollateralRatioEPL, "/ua/v2/market/collateral-discount-ratio", &resp)
}

// GetUTABorrowableCurrencies returns the currencies the unified account can borrow (Get Borrowable Currencies)
func (e *Exchange) GetUTABorrowableCurrencies(ctx context.Context) ([]UTABorrowableCurrency, error) {
	var resp []UTABorrowableCurrency
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getUTABorrowableCurrenciesEPL, "/ua/v2/market/borrowable-currency", &resp)
}

// GetUTACurrentFundingRate returns the current funding rate of a futures contract, or of every contract of a product
// type, or of every contract when both are empty; the symbol takes precedence (Get Current Funding Rate)
func (e *Exchange) GetUTACurrentFundingRate(ctx context.Context, symbol, productType string) ([]UTACurrentFundingRate, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if productType != "" {
		params.Set("productType", productType)
	}
	var resp []UTACurrentFundingRate
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getUTACurrentFundingRateEPL, common.EncodeURLValues("/ua/v2/market/funding-rate", params), &resp)
}

// GetUTAFundingRateHistory returns a futures contract's funding rate at each settlement between two times, newest
// first (Get History Funding Rate)
func (e *Exchange) GetUTAFundingRateHistory(ctx context.Context, symbol string, startAt, endAt time.Time) (*UTAFundingRateHistoryResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if err := common.StartEndTimeCheck(startAt, endAt); err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	var resp *UTAFundingRateHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getUTAFundingRateHistoryEPL, common.EncodeURLValues("/ua/v2/market/funding-rate-history", params), &resp)
}

// GetUTAFuturesOpenInterest returns the real time open interest of futures contracts, or, with an interval, a
// contract's historic open interest, newest first (Get Futures Open Interest)
func (e *Exchange) GetUTAFuturesOpenInterest(ctx context.Context, req *UTAFuturesOpenInterestRequest) ([]UTAFuturesOpenInterest, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Interval != "" && len(req.Symbols) != 1 {
		return nil, errHistoricOpenInterestSymbols
	}
	params := url.Values{}
	for _, s := range req.Symbols {
		params.Add("symbol", s)
	}
	if req.Interval != "" {
		params.Set("interval", req.Interval)
	}
	if err := setTimeRange(params, "startAt", "endAt", req.StartAt, req.EndAt); err != nil {
		return nil, err
	}
	if req.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	var resp []UTAFuturesOpenInterest
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getUTAFuturesOpenInterestEPL, common.EncodeURLValues("/ua/v2/market/open-interest", params), &resp)
}

// GetUTAInterestRateIndex returns a page of a futures contract's interest rates, newest first (Get Interest Rate Index)
func (e *Exchange) GetUTAInterestRateIndex(ctx context.Context, req *UTAInterestRateIndexRequest) (*UTAInterestRateIndexResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", req.Symbol)
	if err := setTimeRange(params, "startAt", "endAt", req.StartAt, req.EndAt); err != nil {
		return nil, err
	}
	if req.LastID != 0 {
		params.Set("lastId", strconv.FormatUint(req.LastID, 10))
	}
	if req.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	var resp *UTAInterestRateIndexResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getUTAInterestRateIndexEPL, common.EncodeURLValues("/ua/v2/market/interest-rate-index", params), &resp)
}

// GetUTAPlatform24HourMarketStatistics returns the platform's 24 hour spot and futures turnover (Platform 24h Market
// Statistics)
func (e *Exchange) GetUTAPlatform24HourMarketStatistics(ctx context.Context) (*UTAPlatform24HourMarketStatisticsResponse, error) {
	var resp *UTAPlatform24HourMarketStatisticsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getUTAPlatform24HourMarketStatisticsEPL, "/ua/v2/trade-statistics", &resp)
}

// GetUTACallAuctionInfo returns a spot symbol's estimated call auction price and size and its order price ranges (Get
// Call Auction Info)
func (e *Exchange) GetUTACallAuctionInfo(ctx context.Context, symbol string) (*UTACallAuctionInfoResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *UTACallAuctionInfoResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getUTACallAuctionInfoEPL, common.EncodeURLValues("/ua/v2/market/call-auction-info", params), &resp)
}

// GetUTAFiatPrice returns the prices of the given currencies, or of every currency, in a fiat currency (Get Fiat
// Price)
func (e *Exchange) GetUTAFiatPrice(ctx context.Context, base currency.Code, currencies []currency.Code) (map[currency.Code]types.Number, error) {
	if base.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("base", base.Upper().String())
	for _, c := range currencies {
		params.Add("currencies", c.Upper().String())
	}
	var resp map[currency.Code]types.Number
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getUTAFiatPriceEPL, common.EncodeURLValues("/ua/v2/market/fiat-price", params), &resp)
}

// GetUTAThirdPartyCustodyCurrencies returns the settlement currencies of a custodian, or of every custodian (Get
// Third-Party Custody Currencies)
func (e *Exchange) GetUTAThirdPartyCustodyCurrencies(ctx context.Context, custodian string, ccy currency.Code) ([]UTAThirdPartyCustodyCurrency, error) {
	params := url.Values{}
	if custodian != "" {
		params.Set("custodian", custodian)
	}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.Upper().String())
	}
	var resp []UTAThirdPartyCustodyCurrency
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getUTAThirdPartyCustodyCurrenciesEPL, common.EncodeURLValues("/ua/v2/oes/currency", params), &resp)
}

// GetUTAServiceStatus returns a product type's service status (Get Service Status)
func (e *Exchange) GetUTAServiceStatus(ctx context.Context, tradeType string) (*UTAServiceStatusResponse, error) {
	if tradeType == "" {
		return nil, errTradeTypeMissing
	}
	params := url.Values{}
	params.Set("tradeType", tradeType)
	var resp *UTAServiceStatusResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getUTAServiceStatusEPL, common.EncodeURLValues("/ua/v2/server/status", params), &resp)
}

// GetUTAKYCRegions returns the regions KuCoin accepts KYC from (Get KYC Regions)
func (e *Exchange) GetUTAKYCRegions(ctx context.Context) ([]UTAKYCRegion, error) {
	var resp []UTAKYCRegion
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getUTAKYCRegionsEPL, "/ua/v2/user/kyc-region", &resp)
}

// GetUTAClientIPAddress returns the IP address KuCoin sees the request come from (Get Client IP Address)
func (e *Exchange) GetUTAClientIPAddress(ctx context.Context) (string, error) {
	var resp string
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getUTAClientIPAddressEPL, "/ua/v2/user/my-ip", &resp)
}
