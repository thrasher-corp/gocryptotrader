package gateio

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

const (
	gateioTradeURL                      = "https://api.gateio.ws"
	gateioFuturesTestnetTrading         = "https://fx-api-testnet.gateio.ws"
	gateioFuturesLiveTradingAlternative = "https://fx-api.gateio.ws"
	gateioAPIVersion                    = "api/v4"

	// Spot
	spotCurrencies    = "spot/currencies"
	spotCurrencyPairs = "spot/currency_pairs"
	spotTickers       = "spot/tickers"
	spotMarketTrades  = "spot/trades"
	spotCandlesticks  = "spot/candlesticks"

	// Wallets
	walletCurrencyChain  = "wallet/currency_chains"
	walletDepositAddress = "wallet/deposit_address"
	walletWithdrawals    = "wallet/withdrawals"
	walletDeposits       = "wallet/deposits"
	walletTransfer       = "wallet/transfers"

	// Margin
	marginCurrencyPairs = "margin/currency_pairs"
	marginFundingBook   = "margin/funding_book"

	// Futures
	futuresSettleContracts    = "futures/%s/contracts"
	futuresOrderbook          = "futures/%s/order_book"
	futuresTrades             = "futures/%s/trades"
	futuresCandlesticks       = "futures/%s/candlesticks"
	futuresTicker             = "futures/%s/tickers"
	futuresFundingRate        = "futures/%s/funding_rate"
	futuresInsuranceBalance   = "futures/%s/insurance"
	futuresContractStats      = "futures/%s/contract_stats"
	futuresIndexConstituent   = "futures/%s/index_constituents/%s"
	futuresLiquidationHistory = "futures/%s/liq_orders"

	// Delivery
	deliveryContracts        = "delivery/%s/contracts"
	deliveryOrderbook        = "delivery/%s/order_book"
	deliveryTradeHistory     = "delivery/%s/trades"
	deliveryCandlesticks     = "delivery/%s/candlesticks"
	deliveryTicker           = "delivery/%s/tickers"
	deliveryInsuranceBalance = "delivery/%s/insurance"

	// Options
	optionUnderlyings            = "options/underlyings"
	optionExpiration             = "options/expirations"
	optionContracts              = "options/contracts"
	optionSettlement             = "options/settlements"
	optionMySettlements          = "options/my_settlements"
	optionsOrderbook             = "options/order_book"
	optionsTickers               = "options/tickers"
	optionsUnderlyingTickers     = "options/underlying/tickers/%s"
	optionCandlesticks           = "options/candlesticks"
	optionUnderlyingCandlesticks = "options/underlying/candlesticks"
	optionsTrades                = "options/trades"
	optionAccounts               = "options/accounts"
	optionsAccountbook           = "options/account_book"
	optionsPosition              = "options/positions"
	optionsPositionClose         = "options/position_close"
	optionsOrders                = "options/orders"
	optionsMyTrades              = "options/my_trades"

	// Flash Swap
	flashSwapCurrencies = "flash_swap/currencies"

	// Withdrawals
	withdrawal                     = "withdrawals"
	clientWithdrawalWithSpecificID = "withdrawals/%s"

	gateioSymbol          = "pairs"
	gateioMarketInfo      = "marketinfo"
	gateioKline           = "candlestick2"
	gateioOrder           = "private"
	gateioBalances        = "private/balances"
	gateioCancelOrder     = "private/cancelOrder"
	gateioCancelAllOrders = "private/cancelAllOrders"
	gateioWithdraw        = "private/withdraw"
	gateioOpenOrders      = "private/openOrders"
	gateioTradeHistory    = "private/tradeHistory"
	gateioDepositAddress  = "private/depositAddress"
	gateioTicker          = "ticker"
	gateioTrades          = "tradeHistory"
	spotOrderbook         = "spot/order_book"

	gateioGenerateAddress = "New address is being generated for you, please wait a moment and refresh this page. "
)

const (
	UTC0TimeZone = "utc0"
	UTC8TimeZone = "utc8"
)

var (
	errInvalidCurrency                     = errors.New("invalid or empty currency")
	errInvalidAssetType                    = errors.New("invalid asset type")
	errInvalidOrEmptyCurrencyPair          = errors.New("empty or invalid currency pair")
	errMissingSettleCurrency               = errors.New("missing settle currency")
	errInvalidOrMissingContractParam       = errors.New("invalid or empty contract")
	errNoValidResponseFromServer           = errors.New("no valid response from server")
	errInvalidUnderlying                   = errors.New("missing underlying")
	errInvalidOrderSize                    = errors.New("invalid order size")
	errInvalidOrderID                      = errors.New("invalid order id")
	errInvalidWithdrawalDestinationAddress = errors.New("invalid withdrawal destination addresss")
	errInvalidAmount                       = errors.New("invalid amount")
)

// Gateio is the overarching type across this package
type Gateio struct {
	exchange.Base
}

// GetSymbols returns all supported symbols
func (g *Gateio) GetSymbols(ctx context.Context) ([]string, error) {
	var result []string
	urlPath := fmt.Sprintf("/%s/%s", gateioAPIVersion, gateioSymbol)
	err := g.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, urlPath, &result)
	return result, err
}

// GetMarketInfo returns information about all trading pairs, including
// transaction fee, minimum order quantity, price accuracy and so on
func (g *Gateio) GetMarketInfo(ctx context.Context) (MarketInfoResponse, error) {
	type response struct {
		Result string        `json:"result"`
		Pairs  []interface{} `json:"pairs"`
	}

	urlPath := fmt.Sprintf("/%s/%s", gateioAPIVersion, gateioMarketInfo)
	var res response
	var result MarketInfoResponse
	err := g.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, urlPath, &res)
	if err != nil {
		return result, err
	}

	result.Result = res.Result
	for _, v := range res.Pairs {
		item, ok := v.(map[string]interface{})
		if !ok {
			return result, errors.New("unable to type assert item")
		}
		for itemk, itemv := range item {
			pairv, ok := itemv.(map[string]interface{})
			if !ok {
				return result, errors.New("unable to type assert pairv")
			}
			decimalPlaces, ok := pairv["decimal_places"].(float64)
			if !ok {
				return result, errors.New("unable to type assert decimal_places")
			}
			minAmount, ok := pairv["min_amount"].(float64)
			if !ok {
				return result, errors.New("unable to type assert min_amount")
			}
			fee, ok := pairv["fee"].(float64)
			if !ok {
				return result, errors.New("unable to type assert fee")
			}
			result.Pairs = append(result.Pairs, MarketInfoPairsResponse{
				Symbol:        itemk,
				DecimalPlaces: decimalPlaces,
				MinAmount:     minAmount,
				Fee:           fee,
			})
		}
	}
	return result, nil
}

// GetLatestSpotPrice returns latest spot price of symbol
// updated every 10 seconds
//
// symbol: string of currency pair
// func (g *Gateio) GetLatestSpotPrice(ctx context.Context, symbol string) (float64, error) {
// 	res, err := g.GetTicker(ctx, symbol)
// 	if err != nil {
// 		return 0, err
// 	}

// 	return res.Last, nil
// }

// // GetTicker returns a ticker for the supplied symbol
// // updated every 10 seconds
// func (g *Gateio) GetTicker(ctx context.Context, symbol string) (TickerResponse, error) {
// 	urlPath := fmt.Sprintf("/%s/%s/%s", gateioAPIVersion, gateioTicker, symbol)
// 	var res TickerResponse
// 	return res, g.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, urlPath, &res)
// }

// // GetTickers returns tickers for all symbols
// func (g *Gateio) GetTickers(ctx context.Context) (map[string]TickerResponse, error) {
// 	urlPath := fmt.Sprintf("/%s/%s", gateioAPIVersion, spotTickers)
// 	resp := make(map[string]TickerResponse)
// 	err := g.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, urlPath, &resp)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return resp, nil
// }

// GetTrades returns trades for symbols
func (g *Gateio) GetTrades(ctx context.Context, symbol string) (TradeHistory, error) {
	urlPath := fmt.Sprintf("/%s/%s/%s", gateioAPIVersion, gateioTrades, symbol)
	var resp TradeHistory
	err := g.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, urlPath, &resp)
	if err != nil {
		return TradeHistory{}, err
	}
	return resp, nil
}

// // GetOrderbook returns the orderbook data for a suppled symbol
// func (g *Gateio) GetOrderbook(ctx context.Context, symbol string) (*Orderbook, error) {
// 	urlPath := fmt.Sprintf("/%s/%s/%s", gateioAPIVersion, gateioOrderbook, symbol)
// 	var resp OrderbookResponse
// 	err := g.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, urlPath, &resp)
// 	if err != nil {
// 		return nil, err
// 	}

// 	switch {
// 	case resp.Result != "true":
// 		return nil, errors.New("result was not true")
// 	case len(resp.Asks) == 0:
// 		return nil, errors.New("asks are empty")
// 	case len(resp.Bids) == 0:
// 		return nil, errors.New("bids are empty")
// 	}

// 	// Asks are in reverse order
// 	ob := Orderbook{
// 		Result:  resp.Result,
// 		Elapsed: resp.Elapsed,
// 		Bids:    make([]OrderbookItem, len(resp.Bids)),
// 		Asks:    make([]OrderbookItem, 0, len(resp.Asks)),
// 	}

// 	for x := len(resp.Asks) - 1; x != 0; x-- {
// 		price, err := strconv.ParseFloat(resp.Asks[x][0], 64)
// 		if err != nil {
// 			return nil, err
// 		}

// 		amount, err := strconv.ParseFloat(resp.Asks[x][1], 64)
// 		if err != nil {
// 			return nil, err
// 		}

// 		ob.Asks = append(ob.Asks, OrderbookItem{Price: price, Amount: amount})
// 	}

// 	for x := range resp.Bids {
// 		price, err := strconv.ParseFloat(resp.Bids[x][0], 64)
// 		if err != nil {
// 			return nil, err
// 		}

// 		amount, err := strconv.ParseFloat(resp.Bids[x][1], 64)
// 		if err != nil {
// 			return nil, err
// 		}

// 		ob.Bids[x] = OrderbookItem{Price: price, Amount: amount}
// 	}
// 	return &ob, nil
// }

// GetSpotKline returns kline data for the most recent time period
func (g *Gateio) GetSpotKline(ctx context.Context, arg KlinesRequestParams) (kline.Item, error) {
	urlPath := fmt.Sprintf("/%s/%s/%s?group_sec=%s&range_hour=%d",
		gateioAPIVersion,
		gateioKline,
		arg.Symbol,
		arg.GroupSec,
		arg.HourSize)

	resp := struct {
		Data   [][]string `json:"data"`
		Result string     `json:"result"`
	}{}

	if err := g.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, urlPath, &resp); err != nil {
		return kline.Item{}, err
	}
	if resp.Result != "true" || len(resp.Data) == 0 {
		return kline.Item{}, errors.New("rawKlines unexpected data returned")
	}

	result := kline.Item{
		Exchange: g.Name,
	}

	for x := range resp.Data {
		if len(resp.Data[x]) < 6 {
			return kline.Item{}, fmt.Errorf("unexpected kline data length")
		}
		otString, err := strconv.ParseFloat(resp.Data[x][0], 64)
		if err != nil {
			return kline.Item{}, err
		}
		ot, err := convert.TimeFromUnixTimestampFloat(otString)
		if err != nil {
			return kline.Item{}, fmt.Errorf("cannot parse Kline.OpenTime. Err: %s", err)
		}
		_vol, err := convert.FloatFromString(resp.Data[x][1])
		if err != nil {
			return kline.Item{}, fmt.Errorf("cannot parse Kline.Volume. Err: %s", err)
		}
		_close, err := convert.FloatFromString(resp.Data[x][2])
		if err != nil {
			return kline.Item{}, fmt.Errorf("cannot parse Kline.Close. Err: %s", err)
		}
		_high, err := convert.FloatFromString(resp.Data[x][3])
		if err != nil {
			return kline.Item{}, fmt.Errorf("cannot parse Kline.High. Err: %s", err)
		}
		_low, err := convert.FloatFromString(resp.Data[x][4])
		if err != nil {
			return kline.Item{}, fmt.Errorf("cannot parse Kline.Low. Err: %s", err)
		}
		_open, err := convert.FloatFromString(resp.Data[x][5])
		if err != nil {
			return kline.Item{}, fmt.Errorf("cannot parse Kline.Open. Err: %s", err)
		}
		result.Candles = append(result.Candles, kline.Candle{
			Time:   ot,
			Volume: _vol,
			Close:  _close,
			High:   _high,
			Low:    _low,
			Open:   _open,
		})
	}
	return result, nil
}

// GetBalances obtains the users account balance
func (g *Gateio) GetBalances(ctx context.Context) (BalancesResponse, error) {
	var result BalancesResponse
	return result,
		g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, gateioBalances, nil, nil, &result)
}

// SpotNewOrder places a new order
func (g *Gateio) SpotNewOrder(ctx context.Context, arg SpotNewOrderRequestParams) (SpotNewOrderResponse, error) {
	var result SpotNewOrderResponse
	var params url.Values
	// Be sure to use the correct price precision before calling this
	params.Set("currencyPair",
		arg.Symbol)
	params.Set("rate",
		strconv.FormatFloat(arg.Price, 'f', -1, 64))
	params.Set("amount",
		strconv.FormatFloat(arg.Amount, 'f', -1, 64))

	urlPath := fmt.Sprintf("%s/%s", gateioOrder, arg.Type)
	return result, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, urlPath, params, nil, &result)
}

// CancelExistingOrder cancels an order given the supplied orderID and symbol
// orderID order ID number
// symbol trade pair (ltc_btc)
func (g *Gateio) CancelExistingOrder(ctx context.Context, orderID int64, symbol string) (bool, error) {
	type response struct {
		Result  bool   `json:"result"`
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	var result response
	var params url.Values
	// Be sure to use the correct price precision before calling this
	params.Set("orderNumber",
		strconv.FormatInt(orderID, 10))
	params.Set(
		"currencyPair",
		symbol)
	err := g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, gateioCancelOrder, params, nil, &result)
	if err != nil {
		return false, err
	}
	if !result.Result {
		return false, fmt.Errorf("code:%d message:%s", result.Code, result.Message)
	}

	return true, nil
}

// CancelAllExistingOrders all orders for a given symbol and side
// orderType (0: sell,1: buy,-1: unlimited)
func (g *Gateio) CancelAllExistingOrders(ctx context.Context, orderType int64, symbol string) error {
	type response struct {
		Result  bool   `json:"result"`
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	var result response
	params := url.Values{}
	params.Set("type", strconv.FormatInt(orderType, 10))
	params.Set("currencyPair", symbol)
	err := g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, gateioCancelAllOrders, params, nil, &result)
	if err != nil {
		return err
	}

	if !result.Result {
		return fmt.Errorf("code:%d message:%s", result.Code, result.Message)
	}

	return nil
}

// GetOpenOrders retrieves all open orders with an optional symbol filter
func (g *Gateio) GetOpenOrders(ctx context.Context, symbol string) (OpenOrdersResponse, error) {
	var params url.Values
	var result OpenOrdersResponse

	if symbol != "" {
		params.Set("currencyPair", symbol)
	}

	err := g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, gateioOpenOrders, params, nil, &result)
	if err != nil {
		return result, err
	}

	if result.Code > 0 {
		return result, fmt.Errorf("code:%d message:%s", result.Code, result.Message)
	}

	return result, nil
}

// GetTradeHistory retrieves all orders with an optional symbol filter
func (g *Gateio) GetTradeHistory(ctx context.Context, symbol string) (TradHistoryResponse, error) {
	var params url.Values
	var result TradHistoryResponse
	params.Set("currencyPair", symbol)

	err := g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, gateioTradeHistory, params, nil, &result)
	if err != nil {
		return result, err
	}

	if result.Code > 0 {
		return result, fmt.Errorf("code:%d message:%s", result.Code, result.Message)
	}

	return result, nil
}

// GetFee returns an estimate of fee based on type of transaction
func (g *Gateio) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (fee float64, err error) {
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		feePairs, err := g.GetMarketInfo(ctx)
		if err != nil {
			return 0, err
		}

		currencyPair := feeBuilder.Pair.Base.String() +
			feeBuilder.Pair.Delimiter +
			feeBuilder.Pair.Quote.String()

		var feeForPair float64
		for _, i := range feePairs.Pairs {
			if strings.EqualFold(currencyPair, i.Symbol) {
				feeForPair = i.Fee
			}
		}

		if feeForPair == 0 {
			return 0, fmt.Errorf("currency '%s' failed to find fee data",
				currencyPair)
		}

		fee = calculateTradingFee(feeForPair,
			feeBuilder.PurchasePrice,
			feeBuilder.Amount)

	case exchange.CryptocurrencyWithdrawalFee:
		fee = getCryptocurrencyWithdrawalFee(feeBuilder.Pair.Base)
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}

	if fee < 0 {
		fee = 0
	}

	return fee, nil
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(price, amount float64) float64 {
	return 0.002 * price * amount
}

func calculateTradingFee(feeForPair, purchasePrice, amount float64) float64 {
	return (feeForPair / 100) * purchasePrice * amount
}

func getCryptocurrencyWithdrawalFee(c currency.Code) float64 {
	return WithdrawalFees[c]
}

// WithdrawCrypto withdraws cryptocurrency to your selected wallet
func (g *Gateio) WithdrawCrypto(ctx context.Context, curr, address, memo, chain string, amount float64) (*withdraw.ExchangeResponse, error) {
	if curr == "" || address == "" || amount <= 0 {
		return nil, errors.New("currency, address and amount must be set")
	}

	resp := struct {
		Result  bool   `json:"result"`
		Message string `json:"message"`
		Code    int    `json:"code"`
	}{}

	vals := url.Values{}
	vals.Set("currency", strings.ToUpper(curr))
	vals.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))

	// Transaction MEMO has to be entered after the address separated by a space
	if memo != "" {
		address += " " + memo
	}
	vals.Set("address", address)

	if chain != "" {
		vals.Set("chain", strings.ToUpper(chain))
	}

	err := g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, gateioWithdraw, vals, nil, &resp)
	if err != nil {
		return nil, err
	}
	if !resp.Result {
		return nil, fmt.Errorf("code:%d message:%s", resp.Code, resp.Message)
	}

	return &withdraw.ExchangeResponse{
		Status: resp.Message,
	}, nil
}

// GetCryptoDepositAddress returns a deposit address for a cryptocurrency
func (g *Gateio) GetCryptoDepositAddress(ctx context.Context, curr string) (*DepositAddr, error) {
	var result DepositAddr
	params := url.Values{}
	params.Set("currency", curr)

	err := g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, gateioDepositAddress, params, nil, &result)
	if err != nil {
		return nil, err
	}

	if !result.Result {
		return nil, fmt.Errorf("code:%d message:%s", result.Code, result.Message)
	}

	// For memo/payment ID currencies
	if strings.Contains(result.Address, " ") {
		split := strings.Split(result.Address, " ")
		result.Address = split[0]
		result.Tag = split[1]
	}
	return &result, nil
}

// *****************************************  Spot **************************************

// ListAllCurrencies to retrive detailed list of each currency.
func (g *Gateio) ListAllCurrencies(ctx context.Context) ([]CurrencyInfo, error) {
	var resp []CurrencyInfo
	return resp, g.SendHTTPRequest(ctx, exchange.RestSpot, spotCurrencies, &resp)
}

// GetCurrencyDetail details of a specific currency
func (g *Gateio) GetCurrencyDetail(ctx context.Context, ccy currency.Code) (*CurrencyInfo, error) {
	var resp CurrencyInfo
	if ccy.IsEmpty() {
		return nil, errInvalidCurrency
	}
	path := fmt.Sprintf("%s/%s", spotCurrencies, ccy.String())
	return &resp, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// ListAllCurrencyPairs retrive all currency pairs supported by the exchange.
func (g *Gateio) ListAllCurrencyPairs(ctx context.Context) ([]CurrencyPairDetail, error) {
	var resp []CurrencyPairDetail
	return resp, g.SendHTTPRequest(ctx, exchange.RestSpot, spotCurrencyPairs, &resp)
}

// GetCurrencyPairDetal to get details of a specifc order
func (g *Gateio) GetCurrencyPairDetail(ctx context.Context, currencyPair currency.Pair) (*CurrencyPairDetail, error) {
	var resp CurrencyPairDetail
	if currencyPair.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	pair := currencyPair.Base.String() + UnderscoreDelimiter + currencyPair.Quote.String()
	path := fmt.Sprintf("%s/%s", spotCurrencyPairs, pair)
	return &resp, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GetTickers retrieve ticker information
// Return only related data if currency_pair is specified; otherwise return all of them
func (g *Gateio) GetTickers(ctx context.Context, currencyPair currency.Pair, timezone string) ([]Ticker, error) {
	var tickers []Ticker
	var pair string
	params := url.Values{}
	if !currencyPair.IsEmpty() {
		pair = currencyPair.Base.String() + UnderscoreDelimiter + currencyPair.Quote.String()
		params.Set("currency_pair", pair)
	}
	if timezone == UTC8TimeZone || timezone == UTC0TimeZone {
		params.Set("timezone", timezone)
	} else {
		params.Set("timezone", "all")
	}
	path := common.EncodeURLValues(spotTickers, params)
	return tickers, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &tickers)
}

// GetTicker retrives a single ticker information for a currency pair.
func (g *Gateio) GetTicker(ctx context.Context, currencyPair currency.Pair, timezone string) (*Ticker, error) {
	if currencyPair.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	tickers, er := g.GetTickers(ctx, currencyPair, timezone)
	if er != nil {
		return nil, er
	}
	if len(tickers) > 0 {
		return &tickers[0], er
	}
	return nil, fmt.Errorf("no ticker data found for currency pair %v", currencyPair)
}

func (g *Gateio) GetIntervalString(interval kline.Interval) string {
	switch interval {
	case kline.TenSecond:
		return "10s"
	case kline.ThirtySecond:
		return "30s"
	case kline.OneMin:
		return "1m"
	case kline.FiveMin:
		return "5m"
	case kline.FifteenMin:
		return "15m"
	case kline.ThirtyMin:
		return "30m"
	case kline.OneHour:
		return "1h"
	case kline.TwoHour:
		return "2h"
	case kline.FourHour:
		return "4h"
	case kline.EightHour:
		return "8h"
	case kline.TwelveHour:
		return "12h"
	case kline.OneDay:
		return "1d"
	case kline.OneWeek:
		return "1w"
	case kline.ThirtyDay:
		return "30d"
	default:
		return ""
	}
}

// GetOrderbook returns the orderbook data for a suppled currency pair
func (g *Gateio) GetOrderbook(ctx context.Context, currencyPair currency.Pair, interval string, limit uint, withOrderbookID bool) (*Orderbook, error) {
	var response OrderbookData
	if currencyPair.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	params := url.Values{}
	fPair, er := g.GetPairFormat(asset.Spot, true)
	if er != nil {
		fPair = currency.PairFormat{
			Delimiter: UnderscoreDelimiter,
			Uppercase: true,
		}
	}
	pairString := strings.ToUpper(fPair.Format(currencyPair))
	params.Set("currency_pair", pairString)
	if interval == OrderbookIntervalZero || interval == OrderbookIntervalZeroPt1 || interval == OrderbookIntervalZeroPtZero1 {
		params.Set("interval", interval)
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%v", limit))
	}
	params.Set("with_id", strconv.FormatBool(withOrderbookID))
	path := common.EncodeURLValues(spotOrderbook, params)
	er = g.SendHTTPRequest(ctx, exchange.RestSpot, path, &response)
	if er != nil {
		return nil, er
	}
	return response.MakeOrderbook()
}

// GetMarketTrades retrieve market trades
func (g *Gateio) GetMarketTrades(ctx context.Context, currencyPair currency.Pair, limit uint, lastID string, reverse bool, from, to time.Time, page int) ([]Trade, error) {
	var response []Trade
	params := url.Values{}
	if currencyPair.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	fPair, er := g.GetPairFormat(asset.Spot, true)
	if er != nil {
		fPair = currency.PairFormat{
			Delimiter: UnderscoreDelimiter,
			Uppercase: true,
		}
	}
	pairString := strings.ToUpper(fPair.Format(currencyPair))
	params.Set("currency_pair", pairString)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(uint64(limit), 10))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	if reverse {
		params.Set("reverse", strconv.FormatBool(reverse))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if page != 0 {
		params.Set("page", strconv.Itoa(page))
	}
	path := common.EncodeURLValues(spotMarketTrades, params)
	return response, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &response)
}

// GetCandlesticks retrives market candlesticks.
func (g *Gateio) GetCandlesticks(ctx context.Context, currencyPair currency.Pair, limit uint, from, to time.Time, interval kline.Interval) ([]Candlestick, error) {
	var candles [][7]string
	params := url.Values{}
	if currencyPair.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	fPair, er := g.GetPairFormat(asset.Spot, true)
	if er != nil {
		fPair = currency.PairFormat{
			Delimiter: UnderscoreDelimiter,
			Uppercase: true,
		}
	}
	pairString := strings.ToUpper(fPair.Format(currencyPair))
	params.Set("currency_pair", pairString)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(uint64(limit), 10))
	}
	if intervalString := g.GetIntervalString(interval); intervalString != "" {
		params.Set("interval", intervalString)
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	path := common.EncodeURLValues(spotCandlesticks, params)
	er = g.SendHTTPRequest(ctx, exchange.RestSpot, path, &candles)
	if er != nil {
		return nil, er
	}
	if len(candles) == 0 {
		return nil, errors.New("no candlesticks available")
	}
	candlesticks := make([]Candlestick, len(candles))
	for x := range candles {
		timestamp, er := strconv.ParseInt(candles[x][0], 10, 64)
		if er != nil {
			return nil, er
		}
		quoteTradingVolume, er := strconv.ParseFloat(candles[x][1], 64)
		if er != nil {
			return nil, er
		}
		closePrice, er := strconv.ParseFloat(candles[x][2], 64)
		if er != nil {
			return nil, er
		}
		highestPrice, er := strconv.ParseFloat(candles[x][3], 64)
		if er != nil {
			return nil, er
		}
		lowestPrice, er := strconv.ParseFloat(candles[x][4], 64)
		if er != nil {
			return nil, er
		}
		openPrice, er := strconv.ParseFloat(candles[x][5], 64)
		if er != nil {
			return nil, er
		}
		baseCurrencyAmount, er := strconv.ParseFloat(candles[x][6], 64)
		if er != nil {
			return nil, er
		}
		candlesticks[x] = Candlestick{
			Timestamp:      time.Unix(timestamp, 0),
			QuoteCcyVolume: quoteTradingVolume,
			ClosePrice:     closePrice,
			HighestPrice:   highestPrice,
			LowestPrice:    lowestPrice,
			OpenPrice:      openPrice,
			BaseCcyAmount:  baseCurrencyAmount,
		}
	}
	return candlesticks, nil
}

// TradingFeeRatio retrives user trading fee rates
// func (g *Gateio) TradingFeeRatio(ctx context.Context, currencyPair currency.Pair) ()

// GenerateSignature returns hash for authenticated requests
func (g *Gateio) GenerateSignature(secret, method, path, query string, body interface{}, dtime time.Time) (string, error) {
	h := sha512.New()
	if body != nil {
		val, er := json.Marshal(body)
		if er != nil {
			return "", er
		}
		h.Write(val)
	}
	h.Write(nil)
	hashedPayload := hex.EncodeToString(h.Sum(nil))
	t := strconv.FormatInt(dtime.Unix(), 10)
	rawQuery, err := url.QueryUnescape(query)
	if err != nil {
		return "", err
	}
	msg := fmt.Sprintf("%s\n%s\n%s\n%s\n%s", method, path, rawQuery, hashedPayload, t)
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// SendAuthenticatedHTTPRequest sends authenticated requests to the Gateio API
// To use this you must setup an APIKey and APISecret from the exchange
func (g *Gateio) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, method, endpoint string, param url.Values, data, result interface{}) error {
	creds, err := g.GetCredentials(ctx)
	if err != nil {
		return err
	}
	ePoint, err := g.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	headers := make(map[string]string)
	urlPath := fmt.Sprintf("/%s/%s", gateioAPIVersion, endpoint)
	timestamp := time.Now()
	if err != nil {
		return err
	}
	var paramValue string
	if param != nil {
		paramValue = param.Encode()
	}
	hmac, err := g.GenerateSignature(creds.Secret, method, urlPath, paramValue, data, timestamp)
	if err != nil {
		return err
	}
	headers["Content-Type"] = "application/json"
	headers["KEY"] = creds.Key
	headers["TIMESTAMP"] = strconv.FormatInt(timestamp.Unix(), 10)
	headers["Accept"] = "application/json"
	headers["SIGN"] = hmac
	var intermidiary json.RawMessage
	urlPath = fmt.Sprintf("%s%s", ePoint, urlPath)
	if param != nil {
		urlPath = common.EncodeURLValues(urlPath, param)
	}
	item := &request.Item{
		Method:        method,
		Path:          urlPath,
		Headers:       headers,
		Result:        &intermidiary,
		AuthRequest:   true,
		Verbose:       g.Verbose,
		HTTPDebugging: g.HTTPDebugging,
		HTTPRecording: g.HTTPRecording,
	}
	err = g.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		var body io.Reader
		if data != nil {
			payload, err := json.Marshal(data)
			if err != nil {
				return nil, err
			}
			body = bytes.NewBuffer(payload)
		}
		item.Body = body
		return item, nil
	})
	if err != nil {
		return err
	}
	errCap := struct {
		Label   string `json:"label"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}{}

	if err := json.Unmarshal(intermidiary, &errCap); err == nil && errCap.Code != "" {
		return fmt.Errorf("%s auth request error, code: %s message: %s",
			g.Name,
			errCap.Label,
			errCap.Message)
	}
	return json.Unmarshal(intermidiary, result)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (g *Gateio) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, result interface{}) error {
	endpoint, err := g.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	path = fmt.Sprintf("/%s/%s", gateioAPIVersion, path)
	item := &request.Item{
		Method:        http.MethodGet,
		Path:          endpoint + path,
		Result:        result,
		Verbose:       g.Verbose,
		HTTPDebugging: g.HTTPDebugging,
		HTTPRecording: g.HTTPRecording,
	}
	return g.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		return item, nil
	})
}

// *********************************** Withdrawals ******************************

// WithdrawCurrency to withdraw a currency.
func (g *Gateio) WithdrawCurrency(ctx context.Context, arg WithdrawalRequestParam) (*WithdrawalResponse, error) {
	var response WithdrawalResponse
	if arg.Amount <= 0 {
		return nil, errInvalidAmount
	}
	if arg.Currency.String() == "" {
		return nil, errInvalidCurrency
	}
	if arg.Address == "" {
		return nil, errInvalidWithdrawalDestinationAddress
	}
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, withdrawal, nil, &arg, &response)
}

func (g *Gateio) CancelWithdrawalWithSpecifiedID(ctx context.Context, withdrawalID string) (*WithdrawalResponse, error) {
	var response WithdrawalResponse
	path := fmt.Sprint(withdrawal, withdrawalID)
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, nil, nil, &response)
}

// *********************************** Wallet ***********************************

// ListCurrencyChain retrives a list of currency chain name
func (g *Gateio) ListCurrencyChain(ctx context.Context, ccy currency.Code) ([]CurrencyChain, error) {
	params := url.Values{}
	if ccy.IsEmpty() {
		return nil, errInvalidCurrency
	}
	params.Set("currency", ccy.String())
	var resp []CurrencyChain
	path := common.EncodeURLValues(walletCurrencyChain, params)
	return resp, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GenerateCurrencyDepositAddress generate currency deposit address
func (g *Gateio) GenerateCurrencyDepositAddress(ctx context.Context, ccy currency.Code) (*CurrencyDepositAddressInfo, error) {
	var response CurrencyDepositAddressInfo
	if ccy.IsEmpty() {
		return nil, errInvalidCurrency
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot,
		http.MethodGet, walletDepositAddress, params, nil, &response)
}

// GetWithdrawalRecords retrieves withdrawal records. Record time range cannot exceed 30 days
func (g *Gateio) GetWithdrawalRecords(ctx context.Context, ccy currency.Code, from, to time.Time, offset, limit int) ([]WithdrawalResponse, error) {
	var withdrawals []WithdrawalResponse
	if ccy.IsEmpty() {
		return nil, errInvalidCurrency
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%v", limit))
	}
	if offset > 0 {
		params.Set("offset", fmt.Sprintf("%v", offset))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() && to.Before(from.Add(time.Hour*720)) {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	return withdrawals, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot,
		http.MethodGet, walletWithdrawals, params, nil, &withdrawals)
}

// GetDepositRecords retrives deposit records. Record time range cannot exceed 30 days
func (g *Gateio) GetDepositRecords(ctx context.Context, ccy currency.Code, from, to time.Time, offset, limit int) ([]DepositRecord, error) {
	var depositHistories []DepositRecord
	if ccy.IsEmpty() {
		return nil, errInvalidCurrency
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%v", limit))
	}
	if offset > 0 {
		params.Set("offset", fmt.Sprintf("%v", offset))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() && to.Before(from.Add(time.Hour*720)) {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	return depositHistories, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot,
		http.MethodGet, walletDeposits, params, nil, &depositHistories)
}

// TransferCurrency Transfer between different accounts. Currently support transfers between the following:
// spot - margin, spot - futures(perpetual), spot - delivery
// spot - cross margin, spot - options
func (g *Gateio) TransferCurrency(ctx context.Context, arg TransferCurrencyParam) (*TransactionIDResponse, error) {
	var response TransactionIDResponse
	if arg.Currency.String() == "" {
		return nil, errInvalidCurrency
	}
	if !arg.CurrencyPair.IsEmpty() {
		arg.CurrencyPair.Delimiter = UnderscoreDelimiter
	}
	if arg.From != asset.Spot {
		return nil, fmt.Errorf("%v, only %s accounts can be used to transfer from", errInvalidAssetType, asset.Spot)
	}
	if !(arg.To == asset.Spot ||
		arg.To == asset.Margin ||
		arg.To == asset.Futures ||
		arg.To == asset.DeliveryFutures ||
		arg.To == asset.CrossMargin ||
		arg.To == asset.Options) {
		return nil, fmt.Errorf("%v, only %v,%v,%v,%v,%v,and %v", errInvalidAssetType, asset.Spot, asset.Margin, asset.Futures, asset.DeliveryFutures, asset.CrossMargin, asset.Options)
	}
	if arg.Amount < 0 {
		return nil, errInvalidAmount
	}
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, walletTransfer, nil, &arg, &response)
}

// *********************************Margin *******************************************

// GetMarginSupportedCurrencyPair retrives margin supported currency pairs.
func (g *Gateio) GetMarginSupportedCurrencyPairs(ctx context.Context) ([]MarginCurrencyPairInfo, error) {
	var currenciePairsInfo []MarginCurrencyPairInfo
	return currenciePairsInfo, g.SendHTTPRequest(ctx, exchange.RestSpot, marginCurrencyPairs, &currenciePairsInfo)
}

// GetMarginSupportedCurrencyPair retrives margin supported currency pair detail given the currency pair.
func (g *Gateio) GetMarginSupportedCurrencyPair(ctx context.Context, cp currency.Pair) (*MarginCurrencyPairInfo, error) {
	var currencyPairInfo MarginCurrencyPairInfo
	if cp.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	fPair, er := g.GetPairFormat(asset.Margin, true)
	if er != nil {
		fPair = currency.PairFormat{
			Delimiter: UnderscoreDelimiter,
			Uppercase: true,
		}
	}
	pairString := strings.ToUpper(fPair.Format(cp))
	path := fmt.Sprintf("%s/%s", marginCurrencyPairs, pairString)
	return &currencyPairInfo, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &currencyPairInfo)
}

// GetOrderbookOfLendingLoans retrives order book of lending loans for specific currency
func (g *Gateio) GetOrderbookOfLendingLoans(ctx context.Context, ccy currency.Code) ([]OrderbookOfLendingLoan, error) {
	var lendingLoans []OrderbookOfLendingLoan
	if ccy.IsEmpty() {
		return nil, errInvalidCurrency
	}
	path := fmt.Sprintf("%s?currency=%s", marginFundingBook, ccy.String())
	return lendingLoans, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &lendingLoans)
}

// *********************************Futures***************************************

// GetAllFutureContracts  retrives list all futures contracts
func (g *Gateio) GetAllFutureContracts(ctx context.Context, settle string) ([]FuturesContract, error) {
	var contracts []FuturesContract
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	path := fmt.Sprintf(futuresSettleContracts, settle)
	return contracts, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &contracts)
}

// GetSingleContract returns a single contract info for the specified settle and Currency Pair (contract << in this case)
func (g *Gateio) GetSingleContract(ctx context.Context, settle string, contract currency.Pair) (*FuturesContract, error) {
	var futureContract FuturesContract
	if contract.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	fPair, er := g.GetPairFormat(asset.Futures, true)
	if er != nil {
		fPair = currency.PairFormat{
			Delimiter: UnderscoreDelimiter,
			Uppercase: true,
		}
	}
	pairString := strings.ToUpper(fPair.Format(contract))
	path := fmt.Sprintf(futuresSettleContracts+"/%s", settle, pairString)
	return &futureContract, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &futureContract)
}

// GetFuturesOrderbook retrives futures order book data for.
func (g *Gateio) GetFuturesOrderbook(ctx context.Context, settle string, contract currency.Pair, interval string, limit uint, withOrderbookID bool) (*Orderbook, error) {
	var response Orderbook
	if contract.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	params := url.Values{}
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	fPair, er := g.GetPairFormat(asset.Spot, true)
	if er != nil {
		fPair = currency.PairFormat{
			Delimiter: UnderscoreDelimiter,
			Uppercase: true,
		}
	}
	pairString := strings.ToUpper(fPair.Format(contract))
	params.Set("contract", pairString)
	if interval == OrderbookIntervalZero || interval == OrderbookIntervalZeroPt1 || interval == OrderbookIntervalZeroPtZero1 {
		params.Set("interval", interval)
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%v", limit))
	}
	params.Set("with_id", strconv.FormatBool(withOrderbookID))
	path := common.EncodeURLValues(fmt.Sprintf(futuresOrderbook, settle), params)
	return &response, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &response)
}

// GetFuturesTradingHistory retrives futures trading history
func (g *Gateio) GetFuturesTradingHistory(ctx context.Context, settle string, contract currency.Pair, limit, offset uint, lastID string, from, to time.Time) ([]TradingHistoryItem, error) {
	var response []TradingHistoryItem
	params := url.Values{}
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if contract.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	fPair, er := g.GetPairFormat(asset.Futures, true)
	if er != nil {
		fPair = currency.PairFormat{
			Delimiter: UnderscoreDelimiter,
			Uppercase: true,
		}
	}
	pairString := strings.ToUpper(fPair.Format(contract))
	params.Set("contract", pairString)
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%v", limit))
	}
	if offset > 0 {
		params.Set("offset", fmt.Sprintf("%v", offset))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	path := common.EncodeURLValues(fmt.Sprintf(futuresTrades, settle), params)
	return response, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &response)
}

// GetFuturesCandlesticks retrives specified contract candlesticks.
func (g *Gateio) GetFuturesCandlesticks(ctx context.Context, settle string, contract currency.Pair, from, to time.Time, limit uint, interval kline.Interval) ([]FuturesCandlestick, error) {
	var candlesticks []FuturesCandlestick
	params := url.Values{}
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if contract.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	fPair, er := g.GetPairFormat(asset.Futures, true)
	if er != nil {
		fPair = currency.PairFormat{
			Delimiter: UnderscoreDelimiter,
			Uppercase: true,
		}
	}
	pairString := strings.ToUpper(fPair.Format(contract))
	params.Set("contract", pairString)
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%v", limit))
	}
	if intervalString := g.GetIntervalString(interval); intervalString != "" {
		params.Set("interval", intervalString)
	}
	path := common.EncodeURLValues(fmt.Sprintf(futuresCandlesticks, settle), params)
	return candlesticks, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &candlesticks)
}

// GetFutureTickers retrives futures ticker information for a specific settle and contract info.
func (g *Gateio) GetFutureTickers(ctx context.Context, settle string, contract currency.Pair) ([]FuturesTicker, error) {
	var tickers []FuturesTicker
	params := url.Values{}
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if !contract.IsEmpty() {
		fPair, er := g.GetPairFormat(asset.Futures, true)
		if er != nil {
			fPair = currency.PairFormat{
				Delimiter: UnderscoreDelimiter,
				Uppercase: true,
			}
		}
		pairString := strings.ToUpper(fPair.Format(contract))
		params.Set("contract", pairString)
	}
	path := common.EncodeURLValues(fmt.Sprintf(futuresTicker, settle), params)
	return tickers, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &tickers)
}

// GetFutureFundingRates retrives funding rate information.
func (g *Gateio) GetFutureFundingRates(ctx context.Context, settle string, contract currency.Pair, limit uint) ([]FuturesFundingRate, error) {
	var rates []FuturesFundingRate
	params := url.Values{}
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if contract.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	fPair, er := g.GetPairFormat(asset.Futures, true)
	if er != nil {
		fPair = currency.PairFormat{
			Delimiter: UnderscoreDelimiter,
			Uppercase: true,
		}
	}
	pairString := strings.ToUpper(fPair.Format(contract))
	params.Set("contract", pairString)
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%v", limit))
	}
	path := common.EncodeURLValues(fmt.Sprintf(futuresFundingRate, settle), params)
	return rates, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &rates)
}

// GetFuturesInsuranceBalanceHistory retrives futures insurance balance history
func (g *Gateio) GetFuturesInsuranceBalanceHistory(ctx context.Context, settle string, limit uint) ([]InsuranceBalance, error) {
	var balances []InsuranceBalance
	params := url.Values{}
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%v", limit))
	}
	path := common.EncodeURLValues(fmt.Sprintf(futuresInsuranceBalance, settle), params)
	return balances, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &balances)
}

// GetFutureStats retrives futures stats
func (g *Gateio) GetFutureStats(ctx context.Context, settle string, contract currency.Pair, from time.Time, interval kline.Interval, limit uint) ([]ContractStat, error) {
	var stats []ContractStat
	params := url.Values{}
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if contract.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	fPair, er := g.GetPairFormat(asset.Futures, true)
	if er != nil {
		fPair = currency.PairFormat{
			Delimiter: UnderscoreDelimiter,
			Uppercase: true,
		}
	}
	pairString := strings.ToUpper(fPair.Format(contract))
	params.Set("contract", pairString)
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if intervalString := g.GetIntervalString(interval); intervalString != "" {
		params.Set("interval", intervalString)
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%v", limit))
	}
	path := common.EncodeURLValues(fmt.Sprintf(futuresContractStats, settle), params)
	return stats, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &stats)
}

// GetIndexConstituent retrives index constituents
func (g *Gateio) GetIndexConstituent(ctx context.Context, settle string, index currency.Pair) (*IndexConstituent, error) {
	var constituents IndexConstituent
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if index.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	fPair, er := g.GetPairFormat(asset.Futures, true)
	if er != nil {
		fPair = currency.PairFormat{
			Delimiter: UnderscoreDelimiter,
			Uppercase: true,
		}
	}
	indexString := strings.ToUpper(fPair.Format(index))
	path := fmt.Sprintf(futuresIndexConstituent, settle, indexString)
	return &constituents, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &constituents)
}

// GetLiquidationHistory retrives liqudiation history
func (g *Gateio) GetLiquidationHistory(ctx context.Context, settle string, contract currency.Pair, from, to time.Time, limit uint) ([]LiquidationHistory, error) {
	var histories []LiquidationHistory
	params := url.Values{}
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	fPair, er := g.GetPairFormat(asset.Futures, true)
	if er != nil {
		fPair = currency.PairFormat{
			Delimiter: UnderscoreDelimiter,
			Uppercase: true,
		}
	}
	pairString := strings.ToUpper(fPair.Format(contract))
	params.Set("contract", pairString)
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%v", limit))
	}
	path := common.EncodeURLValues(fmt.Sprintf(futuresLiquidationHistory, settle), params)
	return histories, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &histories)
}

// ***************************************Delivery ***************************************

// GetAllDeliveryContracts retrives all futures contracts
func (g *Gateio) GetAllDeliveryContracts(ctx context.Context, settle string) ([]DeliveryContract, error) {
	var contracts []DeliveryContract
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	path := fmt.Sprintf(deliveryContracts, settle)
	return contracts, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &contracts)
}

// GetSingleDeliveryContracts retrives a single delivery contract instance.
func (g *Gateio) GetSingleDeliveryContracts(ctx context.Context, settle, contract string) (*DeliveryContract, error) {
	var deliveryContract DeliveryContract
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	path := fmt.Sprintf(deliveryContracts+"/%s", settle, contract)
	return &deliveryContract, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &deliveryContract)
}

// GetDeliveryOrderbook delivery orderbook
func (g *Gateio) GetDeliveryOrderbook(ctx context.Context, settle, contract, interval string, limit uint, withOrderbookID bool) (*Orderbook, error) {
	var orderbook Orderbook
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	params := url.Values{}
	if contract == "" {
		return nil, errInvalidOrMissingContractParam
	}
	params.Set("contract", contract)
	if interval == OrderbookIntervalZero || interval == OrderbookIntervalZeroPt1 || interval == OrderbookIntervalZeroPtZero1 {
		params.Set("interval", interval)
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%v", limit))
	}
	if withOrderbookID {
		params.Set("with_id", strconv.FormatBool(withOrderbookID))
	}
	path := common.EncodeURLValues(fmt.Sprintf(deliveryOrderbook, settle), params)
	return &orderbook, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &orderbook)
}

// GetDeliveryTradingHistory retrives futures trading history
func (g *Gateio) GetDeliveryTradingHistory(ctx context.Context, settle, contract string, limit uint, lastID string, from, to time.Time) ([]DeliveryTradingHistory, error) {
	var histories []DeliveryTradingHistory
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	params := url.Values{}
	if contract == "" {
		return nil, errInvalidOrMissingContractParam
	}
	params.Set("contract", contract)
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%v", limit))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	path := common.EncodeURLValues(fmt.Sprintf(deliveryTradeHistory, settle), params)
	return histories, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &histories)
}

// GetDeliveryFuturesCandlesticks retrives specified contract candlesticks
func (g *Gateio) GetDeliveryFuturesCandlesticks(ctx context.Context, settle, contract string, from, to time.Time, limit uint, interval kline.Interval) ([]FuturesCandlestick, error) {
	var candlesticks []FuturesCandlestick
	params := url.Values{}
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if contract == "" {
		return nil, errInvalidOrMissingContractParam
	}
	params.Set("contract", contract)
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%v", limit))
	}
	if intervalString := g.GetIntervalString(interval); intervalString != "" {
		params.Set("interval", intervalString)
	}
	path := common.EncodeURLValues(fmt.Sprintf(deliveryCandlesticks, settle), params)
	return candlesticks, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &candlesticks)
}

// GetDeliveryFutureTickers retrives futures ticker information for a specific settle and contract info.
func (g *Gateio) GetDeliveryFutureTickers(ctx context.Context, settle string, contract string) ([]FuturesTicker, error) {
	var tickers []FuturesTicker
	params := url.Values{}
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if contract == "" {
		return nil, errInvalidOrMissingContractParam
	}
	params.Set("contract", contract)
	path := common.EncodeURLValues(fmt.Sprintf(deliveryTicker, settle), params)
	return tickers, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &tickers)
}

// GetDeliveryInsuranceBalanceHistory retrives delivery futures insurance balance history
func (g *Gateio) GetDeliveryInsuranceBalanceHistory(ctx context.Context, settle string, limit uint) ([]InsuranceBalance, error) {
	var balances []InsuranceBalance
	params := url.Values{}
	settle = strings.ToLower(settle)
	if !(settle == "btc" || settle == "usd" || settle == "usdt") {
		return nil, errMissingSettleCurrency
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%v", limit))
	}
	path := common.EncodeURLValues(fmt.Sprintf(deliveryInsuranceBalance, settle), params)
	return balances, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &balances)
}

// ********************************** Options ***************************************************

// GetAllUnderlyings retrives all option underlyings
func (g *Gateio) GetAllUnderlyings(ctx context.Context) ([]OptionUnderlying, error) {
	var underlyings []OptionUnderlying
	return underlyings, g.SendHTTPRequest(ctx, exchange.RestSpot, optionUnderlyings, &underlyings)
}

// GetExpirationTime return the expiration time for the provided underlying.
func (g *Gateio) GetExpirationTime(ctx context.Context, underlying string) (time.Time, error) {
	var timestamp []float64
	path := optionExpiration + "?underlying=" + underlying
	er := g.SendHTTPRequest(ctx, exchange.RestSpot, path, &timestamp)
	if er != nil {
		return time.Time{}, er
	}
	if len(timestamp) == 0 {
		return time.Time{}, errNoValidResponseFromServer
	}
	return time.Unix(int64(timestamp[0]), 0), nil
}

// GetAllContractOfUnderlyingWithinExpiryDate retrives list of contracts of the specified underlying and expiry time.
func (g *Gateio) GetAllContractOfUnderlyingWithinExpiryDate(ctx context.Context, underlying string, expTime time.Time) ([]OptionContract, error) {
	var contracts []OptionContract
	params := url.Values{}
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params.Set("underlying", underlying)
	if !expTime.IsZero() {
		params.Set("expires", strconv.FormatInt(expTime.Unix(), 10))
	}
	path := common.EncodeURLValues(optionContracts, params)
	return contracts, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &contracts)
}

// GetSpecifiedContractDetail query specified contract detail
func (g *Gateio) GetSpecifiedContractDetail(ctx context.Context, contract string) (*OptionContract, error) {
	var contr OptionContract
	if contract == "" {
		return nil, errInvalidOrMissingContractParam
	}
	path := fmt.Sprintf(optionContracts+"/%s", contract)
	return &contr, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &contr)
}

// GetSettlementHistory retrives list of settlement history
func (g *Gateio) GetSettlementHistory(ctx context.Context, underlying string, offset, limit uint, from, to time.Time) ([]OptionSettlement, error) {
	var settlements []OptionSettlement
	params := url.Values{}
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params.Set("underlying", underlying)
	if offset > 0 {
		params.Set("offset", strconv.Itoa(int(offset)))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(int(limit)))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	path := common.EncodeURLValues(optionSettlement, params)
	return settlements, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &settlements)
}

// GetSpecifiedSettlementHistory retrive a single contract settlement detail passing the underlying and contract name
func (g *Gateio) GetSpecifiedSettlementHistory(ctx context.Context, contract, underlying string, at uint) (*OptionSettlement, error) {
	var settlement OptionSettlement
	params := url.Values{}
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	if contract == "" {
		return nil, errInvalidOrMissingContractParam
	}
	params.Set("underlying", underlying)
	params.Set("at", strconv.Itoa(int(at)))
	path := common.EncodeURLValues(fmt.Sprintf(optionSettlement+"/%s", contract), params)
	return &settlement, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &settlement)
}

// GetMyOptionsSettlements retrives accounts option settlements.
func (g *Gateio) GetMyOptionsSettlements(ctx context.Context, underlying, contract string, offset, limit uint, to time.Time) ([]MyOptionSettlement, error) {
	var settlements []MyOptionSettlement
	params := url.Values{}
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params.Set("underlying", underlying)
	if contract != "" {
		params.Set("contract", contract)
	}
	if to.After(time.Now()) {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(int(offset)))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(int(limit)))
	}
	return settlements, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, optionMySettlements, params, nil, &settlements)
}

// GetOptionAccounts lists option accounts
func (g *Gateio) GetOptionAccounts(ctx context.Context) (*OptionAccount, error) {
	var resp OptionAccount
	return &resp, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, optionAccounts, nil, nil, &resp)
}

// GetAccountChangingHistory retrives list of account changing history
func (g *Gateio) GetAccountChangingHistory(ctx context.Context, offset, limit int, from, to time.Time, changingType string) ([]AccountBook, error) {
	params := url.Values{}
	var accountBook []AccountBook
	if changingType == "dnw" || changingType == "prem" ||
		changingType == "fee" || changingType == "refr" ||
		changingType == "set" {
		params.Set("type", changingType)
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() && ((!from.IsZero() && to.After(from)) || to.Before(time.Now())) {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	return accountBook, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, optionsOrderbook, params, nil, &accountBook)
}

// GetUsersPositionSpecifiedUnderlying lists user's positions of specified underlying
func (g *Gateio) GetUsersPositionSpecifiedUnderlying(ctx context.Context, underlying string) ([]UsersPositionForUnderlying, error) {
	var response []UsersPositionForUnderlying
	params := url.Values{}
	if underlying != "" {
		params.Set("underlying", underlying)
	}
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, optionsPosition, params, nil, &response)
}

// GetSpecifiedContractPosition retrives specified contract position
func (g *Gateio) GetSpecifiedContractPosition(ctx context.Context, contract string) (*UsersPositionForUnderlying, error) {
	var response UsersPositionForUnderlying
	if contract == "" {
		return nil, errInvalidOrMissingContractParam
	}
	path := fmt.Sprintf("%s/%s", optionsPosition, contract)
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &response)
}

// GetUsersLiquidationHistoryForSpecifiedUnderlying retrives user's liquidation history of specified underlying
func (g *Gateio) GetUsersLiquidationHistoryForSpecifiedUnderlying(ctx context.Context, underlying, contract string) ([]ContractClosePosition, error) {
	params := url.Values{}
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params.Set("underlying", underlying)
	if contract != "" {
		params.Set("contract", contract)
	}
	var response []ContractClosePosition
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, optionsPositionClose, params, nil, &response)
}

// PlaceOptionOrder creates an options order
func (g *Gateio) PlaceOptionOrder(ctx context.Context, arg OptionOrderParam) (*OptionOrderResponse, error) {
	var response OptionOrderResponse
	if arg.Contract == "" {
		return nil, errInvalidOrMissingContractParam
	}
	if arg.OrderSize == 0 {
		return nil, errInvalidOrderSize
	}
	if arg.Iceberg < 0 {
		arg.Iceberg = 0
	}
	if !(arg.TimeInForce == "gtc" || arg.TimeInForce == "ioc" || arg.TimeInForce == "poc") {
		arg.TimeInForce = ""
	}
	if arg.TimeInForce == "ioc" || arg.Price < 0 {
		arg.Price = 0
	}
	if arg.Close {
		arg.OrderSize = 0
	}
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost,
		optionsOrders, nil, &arg, &response)
}

// GetOptionFuturesOrders retrives futures orders
func (g *Gateio) GetOptionFuturesOrders(ctx context.Context, contract, underlying, status string, offset, limit int, from, to time.Time) ([]OptionOrderResponse, error) {
	var response []OptionOrderResponse
	params := url.Values{}
	if contract != "" {
		params.Set("contract", contract)
	}
	if underlying != "" {
		params.Set("underlying", underlying)
	}
	status = strings.ToLower(status)
	if status == "open" || status == "finished" {
		params.Set("status", status)
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() && ((!from.IsZero() && to.After(from)) || to.Before(time.Now())) {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpotSupplementary,
		http.MethodGet, optionsOrders, params, nil, &response)
}

// OptionCancelOpenOrders cancels all open orders matched
func (g *Gateio) CancelOptionOpenOrders(ctx context.Context, contract, underlying, side string) ([]OptionOrderResponse, error) {
	var response []OptionOrderResponse
	params := url.Values{}
	if contract != "" {
		params.Set("contract", contract)
	}
	if underlying != "" {
		params.Set("underlying", underlying)
	}
	if side == "ask" || side == "bid" {
		params.Set("side", side)
	}
	return response, g.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpotSupplementary, http.MethodDelete, optionsOrders, params, nil, &response)
}

// GetSingleOptionorder retrives a single option order
func (g *Gateio) GetSingleOptionorder(ctx context.Context, orderID string) (*OptionOrderResponse, error) {
	var order OptionOrderResponse
	if orderID == "" {
		return nil, errInvalidOrderID
	}
	path := fmt.Sprintf("%s/%s", optionsOrders, orderID)
	return &order, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, path, nil, nil, &order)
}

// CancelSingleOrder cancel a single order.
func (g *Gateio) CancelOptionSingleOrder(ctx context.Context, orderID string) (*OptionOrderResponse, error) {
	var response OptionOrderResponse
	if orderID == "" {
		return nil, errInvalidOrderID
	}
	path := fmt.Sprintf("%s/%s", optionsOrders, orderID)
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodDelete, path, nil, nil, &response)
}

// GetOptionsPersonalTradingHistory retrives personal tradign histories given the underlying{Required}, contract, and other pagination params.
func (g *Gateio) GetOptionsPersonalTradingHistory(ctx context.Context, underlying, contract string, offset, limit int, from, to time.Time) ([]OptionTradingHistory, error) {
	var resp []OptionTradingHistory
	params := url.Values{}
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params.Set("underlying", underlying)
	if contract != "" {
		params.Set("contract", contract)
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() && ((!from.IsZero() && to.After(from)) || to.Before(time.Now())) {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	return resp, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, optionsMyTrades, params, nil, &resp)
}

// GetOptionsFuturesOrderbooks retrives futures order book
func (g *Gateio) GetOptionsFuturesOrderbooks(ctx context.Context, contract, interval string, limit int, withOrderbookID bool) (*Orderbook, error) {
	var orderbook Orderbook
	params := url.Values{}
	if contract == "" {
		return nil, errInvalidOrMissingContractParam
	}
	params.Set("contract", contract)
	if interval == OrderbookIntervalZero || interval == OrderbookIntervalZeroPt1 || interval == OrderbookIntervalZeroPtZero1 {
		params.Set("interval", interval)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if withOrderbookID {
		params.Set("with_id", strconv.FormatBool(withOrderbookID))
	}
	path := common.EncodeURLValues(optionsOrderbook, params)
	return &orderbook, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &orderbook)
}

// GetOptionsTickers lists  tickers of options contracts
func (g *Gateio) GetOptionsTickers(ctx context.Context, underlying string) ([]OptionsTicker, error) {
	var respos []OptionsTicker
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	path := optionsTickers + "?underlying=" + underlying
	return respos, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &respos)
}

// GetOptionUnderlyingTickers retrives options underlying ticker
func (g *Gateio) GetOptionUnderlyingTickers(ctx context.Context, underlying string) (*OptionsUnderlyingTicker, error) {
	var respos OptionsUnderlyingTicker
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	path := fmt.Sprintf(optionsUnderlyingTickers, underlying)
	println(path)
	return &respos, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &respos)
}

// GetOptionFuturesCandlesticks retrives option futures candlesticks
func (g *Gateio) GetOptionFuturesCandlesticks(ctx context.Context, contract string, limit int, from, to time.Time, interval kline.Interval) ([]FuturesCandlestick, error) {
	var candles []FuturesCandlestick
	params := url.Values{}
	if contract == "" {
		return nil, errInvalidOrMissingContractParam
	}
	params.Set("contract", contract)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if intervalString := g.GetIntervalString(interval); intervalString != "" {
		params.Set("interval", intervalString)
	}
	path := common.EncodeURLValues(optionCandlesticks, params)
	return candles, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &candles)
}

// GetOptionFuturesMarkPriceCandlesticks retrives mark price candlesticks of an underlying
func (g *Gateio) GetOptionFuturesMarkPriceCandlesticks(ctx context.Context, underlying string, limit int, from, to time.Time, interval kline.Interval) ([]FuturesCandlestick, error) {
	var candles []FuturesCandlestick
	params := url.Values{}
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params.Set("underlying", underlying)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if intervalString := g.GetIntervalString(interval); intervalString != "" {
		params.Set("interval", intervalString)
	}
	path := common.EncodeURLValues(optionUnderlyingCandlesticks, params)
	return candles, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &candles)
}

// GetOptionsTradeHistory retrives options trade history
func (g *Gateio) GetOptionsTradeHistory(ctx context.Context, contract /*C is call, while P is put*/, callType string,
	offset, limit int, from, to time.Time) ([]TradingHistoryItem, error) {
	var trades []TradingHistoryItem
	params := url.Values{}
	callType = strings.ToUpper(callType)
	if callType == "C" || callType == "P" {
		params.Set("type", callType)
	}
	if contract != "" {
		params.Set("contract", contract)
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	path := common.EncodeURLValues(optionsTrades, params)
	return trades, g.SendHTTPRequest(ctx, exchange.RestSpot, path, &trades)
}

// **********************************Flash_SWAP*************************

// GetSupportedFlashSwapCurrencies retrives all supported currencies in flash swap
func (g *Gateio) GetSupportedFlashSwapCurrencies(ctx context.Context) ([]SwapCurrencies, error) {
	var currencies []SwapCurrencies
	return currencies, g.SendHTTPRequest(ctx, exchange.RestSpot, flashSwapCurrencies, &currencies)
}
