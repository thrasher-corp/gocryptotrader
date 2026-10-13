package kucoin

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// GetAnnouncements returns a page of announcements, by default those published within the last month (Get Announcements)
func (e *Exchange) GetAnnouncements(ctx context.Context, req *AnnouncementsRequest) (*AnnouncementsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if req.CurrentPage != 0 {
		params.Set("currentPage", strconv.FormatUint(req.CurrentPage, 10))
	}
	if req.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	if req.AnnouncementType != "" {
		params.Set("annType", req.AnnouncementType)
	}
	if req.Language != "" {
		params.Set("lang", req.Language)
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	var resp *AnnouncementsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getAnnouncementsEPL, common.EncodeURLValues("/v3/announcements", params), &resp)
}

// GetCurrencyDetailV3 returns a currency and its chains, optionally narrowed to one chain by its chain ID (Get Currency)
func (e *Exchange) GetCurrencyDetailV3(ctx context.Context, ccy currency.Code, chain string) (*CurrencyResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	if chain != "" {
		params.Set("chain", chain)
	}
	var resp *CurrencyResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getCurrencyDetailV3EPL, common.EncodeURLValues("/v3/currencies/"+ccy.Upper().String(), params), &resp)
}

// GetCurrenciesV3 returns every currency and its chains (Get All Currencies)
func (e *Exchange) GetCurrenciesV3(ctx context.Context) ([]CurrencyResponse, error) {
	var resp []CurrencyResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getCurrenciesV3EPL, "/v3/currencies", &resp)
}

// GetSymbol returns a spot symbol's trading rules (Get Symbol)
func (e *Exchange) GetSymbol(ctx context.Context, symbol string) (*SymbolResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *SymbolResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getSymbolEPL, "/v2/symbols/"+symbol, &resp)
}

// GetSymbols returns the trading rules of every spot symbol, or of one market's symbols (Get All Symbols)
func (e *Exchange) GetSymbols(ctx context.Context, market string) ([]SymbolResponse, error) {
	params := url.Values{}
	if market != "" {
		params.Set("market", market)
	}
	var resp []SymbolResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getSymbolsEPL, common.EncodeURLValues("/v2/symbols", params), &resp)
}

// GetTicker returns a symbol's best bid and ask and its last trade (Get Ticker)
func (e *Exchange) GetTicker(ctx context.Context, symbol string) (*TickerResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *TickerResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getTickerEPL, common.EncodeURLValues("/v1/market/orderbook/level1", params), &resp); err != nil {
		return nil, err
	}
	// KuCoin answers an unknown symbol with null data
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetTickers returns every symbol's 24 hour ticker, which KuCoin snapshots every 2 seconds (Get All Tickers)
func (e *Exchange) GetTickers(ctx context.Context) (*TickersResponse, error) {
	var resp *TickersResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getTickersEPL, "/v1/market/allTickers", &resp)
}

// GetTradeHistory returns a symbol's last 100 trades (Get Trade History)
func (e *Exchange) GetTradeHistory(ctx context.Context, symbol string) ([]Trade, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp []Trade
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getTradeHistoryEPL, common.EncodeURLValues("/v1/market/histories", params), &resp)
}

// GetKlines returns at most 1500 of a symbol's candles, newest first; intervals without trades have no candle (Get
// Klines)
func (e *Exchange) GetKlines(ctx context.Context, req *KlinesRequest) ([]Kline, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if req.Type == "" {
		return nil, fmt.Errorf("%w: type is required", errInvalidPeriod)
	}
	if !req.StartAt.IsZero() && !req.EndAt.IsZero() {
		if err := common.StartEndTimeCheck(req.StartAt, req.EndAt); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("symbol", req.Symbol)
	params.Set("type", req.Type)
	if !req.StartAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(req.StartAt.Unix(), 10))
	}
	if !req.EndAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(req.EndAt.Unix(), 10))
	}
	var resp []Kline
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getKlinesEPL, common.EncodeURLValues("/v1/market/candles", params), &resp)
}

// GetPartOrderbook returns the top size levels of a symbol's orderbook, where size is 20 or 100 (Get Part OrderBook)
func (e *Exchange) GetPartOrderbook(ctx context.Context, symbol string, size uint64) (*OrderbookResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if size == 0 {
		return nil, errOrderbookSizeRequired
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *OrderbookResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getPartOrderbookEPL, common.EncodeURLValues("/v1/market/orderbook/level2_"+strconv.FormatUint(size, 10), params), &resp); err != nil {
		return nil, err
	}
	// KuCoin answers an unknown symbol with null data
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetOrderbook returns a symbol's full orderbook (Get Full OrderBook)
func (e *Exchange) GetOrderbook(ctx context.Context, symbol string) (*OrderbookResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *OrderbookResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getOrderbookEPL, http.MethodGet, common.EncodeURLValues("/v3/market/orderbook/level2", params), nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetCallAuctionPartOrderbook returns the top size levels of a symbol's call auction orderbook, where size is 20 or
// 100; outside an auction it is empty with sequence -1 (Get Call Auction Part OrderBook)
func (e *Exchange) GetCallAuctionPartOrderbook(ctx context.Context, symbol string, size uint64) (*OrderbookResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if size == 0 {
		return nil, errOrderbookSizeRequired
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *OrderbookResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getCallAuctionPartOrderbookEPL, common.EncodeURLValues("/v1/market/orderbook/callauction/level2_"+strconv.FormatUint(size, 10), params), &resp)
}

// GetCallAuctionInfo returns a symbol's call auction estimated price and size and its order price ranges, which are
// all empty outside an auction (Get Call Auction Info)
func (e *Exchange) GetCallAuctionInfo(ctx context.Context, symbol string) (*CallAuctionInfoResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *CallAuctionInfoResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getCallAuctionInfoEPL, common.EncodeURLValues("/v1/market/callauctionData", params), &resp)
}

// GetFiatPrice returns currencies' prices in a fiat base currency, by default every currency in USD (Get Fiat Price)
func (e *Exchange) GetFiatPrice(ctx context.Context, base currency.Code, currencies currency.Currencies) (map[currency.Code]types.Number, error) {
	params := url.Values{}
	if !base.IsEmpty() {
		params.Set("base", base.Upper().String())
	}
	if len(currencies) != 0 {
		codes := make([]string, len(currencies))
		for i := range currencies {
			codes[i] = currencies[i].Upper().String()
		}
		params.Set("currencies", strings.Join(codes, ","))
	}
	var resp map[currency.Code]types.Number
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getFiatPriceEPL, common.EncodeURLValues("/v1/prices", params), &resp)
}

// Get24HourStats returns a symbol's statistics over the last 24 hours (Get 24hr Stats)
func (e *Exchange) Get24HourStats(ctx context.Context, symbol string) (*Stats24HourResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *Stats24HourResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, get24HourStatsEPL, common.EncodeURLValues("/v1/market/stats", params), &resp)
}

// GetMarketList returns the names of the spot trading markets, which GetSymbols accepts as a filter (Get Market List)
func (e *Exchange) GetMarketList(ctx context.Context) ([]string, error) {
	var resp []string
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getMarketListEPL, "/v1/markets", &resp)
}

// GetClientIPAddress returns the IP address KuCoin sees the request come from (Get Client IP Address)
func (e *Exchange) GetClientIPAddress(ctx context.Context) (string, error) {
	var resp string
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getClientIPAddressEPL, "/v1/my-ip", &resp)
}

// GetCurrentServerTime returns KuCoin's server time (Get Server Time)
func (e *Exchange) GetCurrentServerTime(ctx context.Context) (time.Time, error) {
	var resp types.Time
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getCurrentServerTimeEPL, "/v1/timestamp", &resp); err != nil {
		return time.Time{}, err
	}
	return resp.Time(), nil
}

// GetServiceStatus returns whether spot trading is open, closed or cancel only (Get Service Status)
func (e *Exchange) GetServiceStatus(ctx context.Context) (*ServiceStatusResponse, error) {
	var resp *ServiceStatusResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getServiceStatusEPL, "/v1/status", &resp)
}

// GetKYCRegions returns the regions users can verify their identity in (Get KYC Regions)
func (e *Exchange) GetKYCRegions(ctx context.Context) ([]KYCRegion, error) {
	var resp []KYCRegion
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getKYCRegionsEPL, "/kyc/regions/v4", &resp)
}
