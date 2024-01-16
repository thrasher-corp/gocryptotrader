package coinbasepro

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	coinbaseAPIURL             = "https://api.coinbase.com"
	coinbaseproSandboxAPIURL   = "https://api-public.sandbox.exchange.coinbase.com/"
	coinbaseproAPIVersion      = "0"
	coinbaseV3                 = "/api/v3/brokerage/"
	coinbaseAccounts           = "accounts"
	coinbaseBestBidAsk         = "best_bid_ask"
	coinbaseProductBook        = "product_book"
	coinbaseProducts           = "products"
	coinbaseOrders             = "orders"
	coinbaseBatchCancel        = "batch_cancel"
	coinbaseHistorical         = "historical"
	coinbaseBatch              = "batch"
	coinbaseEdit               = "edit"
	coinbaseEditPreview        = "edit_preview"
	coinbaseFills              = "fills"
	coinbaseCandles            = "candles"
	coinbaseTicker             = "ticker"
	coinbasePortfolios         = "portfolios"
	coinbaseMoveFunds          = "move_funds"
	coinbaseCFM                = "cfm"
	coinbaseBalanceSummary     = "balance_summary"
	coinbasePositions          = "positions"
	coinbaseSweeps             = "sweeps"
	coinbaseSchedule           = "schedule"
	coinbaseTransactionSummary = "transaction_summary"
	coinbaseConvert            = "convert"
	coinbaseQuote              = "quote"
	coinbaseTrade              = "trade"
	coinbaseV2                 = "/v2/"
	coinbaseNotifications      = "notifications"
	coinbaseUser               = "user"
	coinbaseUsers              = "users"
	coinbaseAuth               = "auth"
	coinbaseAddresses          = "addresses"
	coinbaseTransactions       = "transactions"
	coinbaseDeposits           = "deposits"
	coinbaseCommit             = "commit"
	coinbasePaymentMethods     = "payment-methods"
	coinbaseWithdrawals        = "withdrawals"
	coinbaseCurrencies         = "currencies"
	coinbaseCrypto             = "crypto"
	coinbaseExchangeRates      = "exchange-rates"
	coinbasePrices             = "prices"
	coinbaseTime               = "time"

	FiatDeposit     FiatTransferType = false
	FiatWithdrawal  FiatTransferType = true
	pageNone                         = ""
	pageBefore                       = "before"
	pageAfter                        = "after"
	unknownContract                  = "UNKNOWN_CONTRACT_EXPIRY_TYPE"
	granUnknown                      = "UNKNOWN_GRANULARITY"
	granOneMin                       = "ONE_MINUTE"
	granFiveMin                      = "FIVE_MINUTE"
	granFifteenMin                   = "FIFTEEN_MINUTE"
	granThirtyMin                    = "THIRTY_MINUTE"
	granOneHour                      = "ONE_HOUR"
	granTwoHour                      = "TWO_HOUR"
	granSixHour                      = "SIX_HOUR"
	granOneDay                       = "ONE_DAY"
	startDateString                  = "start_date"
	endDateString                    = "end_date"

	errPayMethodNotFound    = "payment method '%v' not found"
	errIntervalNotSupported = "interval not supported"
)

var (
	errAccountIDEmpty         = errors.New("account id cannot be empty")
	errClientOrderIDEmpty     = errors.New("client order id cannot be empty")
	errProductIDEmpty         = errors.New("product id cannot be empty")
	errOrderIDEmpty           = errors.New("order ids cannot be empty")
	errOpenPairWithOtherTypes = errors.New("cannot pair open orders with other order types")
	errUserIDEmpty            = errors.New("user id cannot be empty")
	errSizeAndPriceZero       = errors.New("size and price cannot both be 0")
	errCurrencyEmpty          = errors.New("currency cannot be empty")
	errCurrWalletConflict     = errors.New("exactly one of walletID and currency must be specified")
	errWalletIDEmpty          = errors.New("wallet id cannot be empty")
	errAddressIDEmpty         = errors.New("address id cannot be empty")
	errTransactionTypeEmpty   = errors.New("transaction type cannot be empty")
	errToEmpty                = errors.New("to cannot be empty")
	errAmountEmpty            = errors.New("amount cannot be empty")
	errTransactionIDEmpty     = errors.New("transaction id cannot be empty")
	errPaymentMethodEmpty     = errors.New("payment method cannot be empty")
	errDepositIDEmpty         = errors.New("deposit id cannot be empty")
	errInvalidPriceType       = errors.New("price type must be spot, buy, or sell")
	errInvalidOrderType       = errors.New("order type must be market, limit, or stop")
	errNoMatchingWallets      = errors.New("no matching wallets returned")
	errOrderModFailNoErr      = errors.New("order modification failed but no error returned")
	errNoMatchingOrders       = errors.New("no matching orders returned")
	errPointerNil             = errors.New("relevant pointer is nil")
	errNameEmpty              = errors.New("name cannot be empty")
	errPortfolioIDEmpty       = errors.New("portfolio id cannot be empty")
	errFeeTypeNotSupported    = errors.New("fee type not supported")
	errUnknownEndpointLimit   = errors.New("unknown endpoint limit")
	errNoEventsWS             = errors.New("no events returned from websocket")
)

// GetAllAccounts returns information on all trading accounts associated with the API key
func (c *CoinbasePro) GetAllAccounts(ctx context.Context, limit uint8, cursor string) (AllAccountsResponse, error) {
	var resp AllAccountsResponse

	var params Params
	params.urlVals = url.Values{}
	params.urlVals.Set("limit", strconv.FormatInt(int64(limit), 10))
	params.urlVals.Set("cursor", cursor)

	pathParams := common.EncodeURLValues("", params.urlVals)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		coinbaseV3+coinbaseAccounts, pathParams, nil, true, &resp, nil)
}

// GetAccountByID returns information for a single account
func (c *CoinbasePro) GetAccountByID(ctx context.Context, accountID string) (*Account, error) {
	if accountID == "" {
		return nil, errAccountIDEmpty
	}
	path := fmt.Sprintf("%s%s/%s", coinbaseV3, coinbaseAccounts, accountID)
	resp := OneAccountResponse{}

	return &resp.Account, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, "", nil, true, &resp, nil)
}

// GetBestBidAsk returns the best bid/ask for all products. Can be filtered to certain products
// by passing through additional strings
func (c *CoinbasePro) GetBestBidAsk(ctx context.Context, products []string) (BestBidAsk, error) {
	var params Params
	params.urlVals = url.Values{}
	if len(products) > 0 {
		for x := range products {
			params.urlVals.Add("product_ids", products[x])
		}
	}

	pathParams := common.EncodeURLValues("", params.urlVals)

	var resp BestBidAsk

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		coinbaseV3+coinbaseBestBidAsk, pathParams, nil, true, &resp, nil)
}

// GetProductBook returns a list of bids/asks for a single product
func (c *CoinbasePro) GetProductBook(ctx context.Context, productID string, limit uint16) (ProductBook, error) {
	if productID == "" {
		return ProductBook{}, errProductIDEmpty
	}
	var params Params
	params.urlVals = url.Values{}
	params.urlVals.Set("product_id", productID)
	params.urlVals.Set("limit", strconv.FormatInt(int64(limit), 10))

	pathParams := common.EncodeURLValues("", params.urlVals)

	var resp ProductBookResponse

	return resp.Pricebook, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		coinbaseV3+coinbaseProductBook, pathParams, nil, true, &resp, nil)
}

// GetAllProducts returns information on all currency pairs that are available for trading
func (c *CoinbasePro) GetAllProducts(ctx context.Context, limit, offset int32, productType, contractExpiryType, expiringContractStatus string, productIDs []string) (AllProducts, error) {
	var params Params
	params.urlVals = url.Values{}
	params.urlVals.Set("limit", strconv.FormatInt(int64(limit), 10))
	params.urlVals.Set("offset", strconv.FormatInt(int64(offset), 10))

	if productType != "" {
		params.urlVals.Set("product_type", productType)
	}

	if contractExpiryType != "" {
		params.urlVals.Set("contract_expiry_type", contractExpiryType)
	}

	if len(productIDs) > 0 {
		for x := range productIDs {
			params.urlVals.Add("product_ids", productIDs[x])
		}
	}

	pathParams := common.EncodeURLValues("", params.urlVals)

	var products AllProducts

	return products, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		coinbaseV3+coinbaseProducts, pathParams, nil, true, &products, nil)
}

// GetProductByID returns information on a single specified currency pair
func (c *CoinbasePro) GetProductByID(ctx context.Context, productID string) (*Product, error) {
	if productID == "" {
		return nil, errProductIDEmpty
	}

	path := fmt.Sprintf("%s%s/%s", coinbaseV3, coinbaseProducts, productID)

	resp := Product{}

	return &resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, "", nil, true, &resp, nil)
}

// GetHistoricRates returns historic rates for a product. Rates are returned in
// grouped buckets based on requested granularity. Requests that return more than
// 300 data points are rejected
func (c *CoinbasePro) GetHistoricRates(ctx context.Context, productID, granularity string, startDate, endDate time.Time) (History, error) {
	var resp History

	if productID == "" {
		return resp, errProductIDEmpty
	}

	allowedGranularities := [8]string{granOneMin, granFiveMin, granFifteenMin,
		granThirtyMin, granOneHour, granTwoHour, granSixHour, granOneDay}
	validGran, _ := common.InArray(granularity, allowedGranularities)
	if !validGran {
		return resp, fmt.Errorf("invalid granularity %v, allowed granularities are: %+v",
			granularity, allowedGranularities)
	}

	var params Params
	params.urlVals = url.Values{}

	params.urlVals.Set("start", strconv.FormatInt(startDate.Unix(), 10))
	params.urlVals.Set("end", strconv.FormatInt(endDate.Unix(), 10))
	params.urlVals.Set("granularity", granularity)

	pathParams := common.EncodeURLValues("", params.urlVals)

	path := fmt.Sprintf("%s%s/%s/%s", coinbaseV3, coinbaseProducts, productID, coinbaseCandles)

	err := c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, pathParams, nil, true, &resp, nil)

	return resp, err
}

// GetTicker returns snapshot information about the last trades (ticks) and best bid/ask.
// Contrary to documentation, this does not tell you the 24h volume
func (c *CoinbasePro) GetTicker(ctx context.Context, productID string, limit uint16, startDate, endDate time.Time) (*Ticker, error) {
	if productID == "" {
		return nil, errProductIDEmpty
	}
	path := fmt.Sprintf(
		"%s%s/%s/%s", coinbaseV3, coinbaseProducts, productID, coinbaseTicker)

	var params Params
	params.urlVals = url.Values{}
	params.urlVals.Set("limit", strconv.FormatInt(int64(limit), 10))
	params.urlVals.Set("start", strconv.FormatInt(startDate.Unix(), 10))
	params.urlVals.Set("end", strconv.FormatInt(endDate.Unix(), 10))

	pathParams := common.EncodeURLValues("", params.urlVals)

	var resp Ticker

	return &resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, pathParams, nil, true, &resp, nil)
}

// PlaceOrder places either a limit, market, or stop order
func (c *CoinbasePro) PlaceOrder(ctx context.Context, clientOID, productID, side, stopDirection, orderType, stpID, marginType, rpID string, amount, limitPrice, stopPrice, leverage float64, postOnly bool, endTime time.Time) (*PlaceOrderResp, error) {
	if clientOID == "" {
		return nil, errClientOrderIDEmpty
	}
	if productID == "" {
		return nil, errProductIDEmpty
	}
	if amount == 0 {
		return nil, errAmountEmpty
	}

	var orderConfig OrderConfiguration

	switch orderType {
	case order.Market.String(), order.ImmediateOrCancel.String():
		orderConfig.MarketMarketIOC = &MarketMarketIOC{}
		if side == order.Buy.String() {
			orderConfig.MarketMarketIOC.QuoteSize = strconv.FormatFloat(amount, 'f', -1, 64)
		}
		if side == order.Sell.String() {
			orderConfig.MarketMarketIOC.BaseSize = strconv.FormatFloat(amount, 'f', -1, 64)
		}
	case order.Limit.String():
		if endTime == (time.Time{}) {
			orderConfig.LimitLimitGTC = &LimitLimitGTC{}
			orderConfig.LimitLimitGTC.BaseSize = strconv.FormatFloat(amount, 'f', -1, 64)
			orderConfig.LimitLimitGTC.LimitPrice = strconv.FormatFloat(limitPrice, 'f', -1, 64)
			orderConfig.LimitLimitGTC.PostOnly = postOnly
		} else {
			orderConfig.LimitLimitGTD = &LimitLimitGTD{}
			orderConfig.LimitLimitGTD.BaseSize = strconv.FormatFloat(amount, 'f', -1, 64)
			orderConfig.LimitLimitGTD.LimitPrice = strconv.FormatFloat(limitPrice, 'f', -1, 64)
			orderConfig.LimitLimitGTD.PostOnly = postOnly
			orderConfig.LimitLimitGTD.EndTime = endTime
		}
	case order.StopLimit.String():
		if endTime == (time.Time{}) {
			orderConfig.StopLimitStopLimitGTC = &StopLimitStopLimitGTC{}
			orderConfig.StopLimitStopLimitGTC.BaseSize = strconv.FormatFloat(amount, 'f', -1, 64)
			orderConfig.StopLimitStopLimitGTC.LimitPrice = strconv.FormatFloat(limitPrice, 'f', -1,
				64)
			orderConfig.StopLimitStopLimitGTC.StopPrice = strconv.FormatFloat(stopPrice, 'f', -1, 64)
			orderConfig.StopLimitStopLimitGTC.StopDirection = stopDirection
		} else {
			orderConfig.StopLimitStopLimitGTD = &StopLimitStopLimitGTD{}
			orderConfig.StopLimitStopLimitGTD.BaseSize = strconv.FormatFloat(amount, 'f', -1, 64)
			orderConfig.StopLimitStopLimitGTD.LimitPrice = strconv.FormatFloat(limitPrice, 'f', -1,
				64)
			orderConfig.StopLimitStopLimitGTD.StopPrice = strconv.FormatFloat(stopPrice, 'f', -1, 64)
			orderConfig.StopLimitStopLimitGTD.StopDirection = stopDirection
			orderConfig.StopLimitStopLimitGTD.EndTime = endTime
		}
	default:
		return nil, errInvalidOrderType
	}

	req := map[string]interface{}{"client_order_id": clientOID, "product_id": productID,
		"side": side, "order_configuration": orderConfig, "self_trade_prevention_id": stpID,
		"leverage": strconv.FormatFloat(leverage, 'f', -1, 64), "retail_portfolio_id": rpID}

	if marginType == "ISOLATED" || marginType == "CROSS" {
		req["margin_type"] = marginType
	}
	if marginType == "MULTI" {
		req["margin_type"] = "CROSS"
	}

	var resp PlaceOrderResp

	return &resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
			coinbaseV3+coinbaseOrders, "", req, true, &resp, nil)
}

// CancelOrders cancels orders by orderID
func (c *CoinbasePro) CancelOrders(ctx context.Context, orderIDs []string) (CancelOrderResp, error) {
	var resp CancelOrderResp
	if len(orderIDs) == 0 {
		return resp, errOrderIDEmpty
	}

	path := fmt.Sprintf("%s%s/%s", coinbaseV3, coinbaseOrders, coinbaseBatchCancel)

	req := map[string]interface{}{"order_ids": orderIDs}

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, "",
		req, true, &resp, nil)
}

// EditOrder edits an order to a new size or price. Only limit orders with a good-till-cancelled time
// in force can be edited
func (c *CoinbasePro) EditOrder(ctx context.Context, orderID string, size, price float64) (SuccessBool, error) {
	var resp SuccessBool

	if orderID == "" {
		return resp, errOrderIDEmpty
	}
	if size == 0 && price == 0 {
		return resp, errSizeAndPriceZero
	}

	path := fmt.Sprintf("%s%s/%s", coinbaseV3, coinbaseOrders, coinbaseEdit)

	req := map[string]interface{}{"order_id": orderID, "size": strconv.FormatFloat(size, 'f', -1, 64),
		"price": strconv.FormatFloat(price, 'f', -1, 64)}

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, "",
		req, true, &resp, nil)
}

// EditOrderPreview simulates an edit order request, to preview the result. Only limit orders with a
// good-till-cancelled time in force can be edited.
func (c *CoinbasePro) EditOrderPreview(ctx context.Context, orderID string, size, price float64) (*EditOrderPreviewResp, error) {
	if orderID == "" {
		return nil, errOrderIDEmpty
	}
	if size == 0 && price == 0 {
		return nil, errSizeAndPriceZero
	}

	path := fmt.Sprintf("%s%s/%s", coinbaseV3, coinbaseOrders, coinbaseEditPreview)

	req := map[string]interface{}{"order_id": orderID, "size": strconv.FormatFloat(size, 'f', -1, 64),
		"price": strconv.FormatFloat(price, 'f', -1, 64)}

	var resp *EditOrderPreviewResp

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, "",
		req, true, &resp, nil)
}

// GetAllOrders lists orders, filtered by their status
func (c *CoinbasePro) GetAllOrders(ctx context.Context, productID, userNativeCurrency, orderType, orderSide, cursor, productType, orderPlacementSource, contractExpiryType, retailPortfolioID string, orderStatus, assetFilters []string, limit int32, startDate, endDate time.Time) (GetAllOrdersResp, error) {
	var resp GetAllOrdersResp

	var params Params
	params.urlVals = make(url.Values)
	err := params.prepareDateString(startDate, endDate, startDateString, endDateString)
	if err != nil {
		return resp, err
	}
	if len(orderStatus) != 0 {
		for x := range orderStatus {
			if orderStatus[x] == "OPEN" && len(orderStatus) > 1 {
				return resp, errOpenPairWithOtherTypes
			}
			params.urlVals.Add("order_status", orderStatus[x])
		}
	}
	if len(assetFilters) != 0 {
		for x := range assetFilters {
			params.urlVals.Add("asset_filters", assetFilters[x])
		}
	}

	params.urlVals.Set("product_id", productID)
	params.urlVals.Set("limit", strconv.FormatInt(int64(limit), 10))
	params.urlVals.Set("cursor", cursor)

	if userNativeCurrency != "" {
		params.urlVals.Set("user_native_currency", userNativeCurrency)
	}
	if orderPlacementSource != "" {
		params.urlVals.Set("order_placement_source", orderPlacementSource)
	}
	if productType != "" {
		params.urlVals.Set("product_type", productType)
	}
	if orderSide != "" {
		params.urlVals.Set("order_side", orderSide)
	}
	if contractExpiryType != "" {
		params.urlVals.Set("contract_expiry_type", contractExpiryType)
	}
	if orderType != "" {
		params.urlVals.Set("order_type", orderType)
	}

	pathParams := common.EncodeURLValues("", params.urlVals)
	path := fmt.Sprintf("%s%s/%s/%s", coinbaseV3, coinbaseOrders, coinbaseHistorical, coinbaseBatch)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path,
		pathParams, nil, true, &resp, nil)
}

// GetFills returns information of recent fills on the specified profile
func (c *CoinbasePro) GetFills(ctx context.Context, orderID, productID, cursor string, startDate, endDate time.Time, limit uint16) (FillResponse, error) {
	var resp FillResponse
	var params Params
	params.urlVals = url.Values{}
	err := params.prepareDateString(startDate, endDate, "start_sequence_timestamp",
		"end_sequence_timestamp")
	if err != nil {
		return resp, err
	}

	if orderID != "" {
		params.urlVals.Set("order_id", orderID)
	}
	if productID != "" {
		params.urlVals.Set("product_id", productID)
	}

	params.urlVals.Set("limit", strconv.FormatInt(int64(limit), 10))
	params.urlVals.Set("cursor", cursor)

	pathParams := common.EncodeURLValues("", params.urlVals)
	path := fmt.Sprintf("%s%s/%s/%s", coinbaseV3, coinbaseOrders, coinbaseHistorical, coinbaseFills)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path,
		pathParams, nil, true, &resp, nil)
}

// GetOrderByID returns a single order by order id.
func (c *CoinbasePro) GetOrderByID(ctx context.Context, orderID, clientID, userNativeCurrency string) (*GetOrderResponse, error) {
	if orderID == "" {
		return nil, errOrderIDEmpty
	}
	var resp GetOrderResponse
	var params Params
	params.urlVals = url.Values{}
	params.urlVals.Set("client_order_id", clientID)
	params.urlVals.Set("user_native_currency", userNativeCurrency)

	path := fmt.Sprintf("%s%s/%s/%s", coinbaseV3, coinbaseOrders, coinbaseHistorical, orderID)
	pathParams := common.EncodeURLValues("", params.urlVals)

	return &resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, pathParams, nil, true, &resp, nil)
}

// GetAllPortfolios returns a list of portfolios associated with the user
func (c *CoinbasePro) GetAllPortfolios(ctx context.Context, portfolioType string) (AllPortfolioResponse, error) {
	var resp AllPortfolioResponse

	var params Params
	params.urlVals = url.Values{}

	if portfolioType != "" {
		params.urlVals.Set("portfolio_type", portfolioType)
	}

	pathParams := common.EncodeURLValues("", params.urlVals)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		coinbaseV3+coinbasePortfolios, pathParams, nil, true, &resp, nil)
}

// CreatePortfolio creates a new portfolio
func (c *CoinbasePro) CreatePortfolio(ctx context.Context, name string) (SimplePortfolioResponse, error) {
	var resp SimplePortfolioResponse

	if name == "" {
		return resp, errNameEmpty
	}

	req := map[string]interface{}{"name": name}

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		coinbaseV3+coinbasePortfolios, "", req, true, &resp, nil)
}

// MovePortfolioFunds transfers funds between portfolios
func (c *CoinbasePro) MovePortfolioFunds(ctx context.Context, currency, from, to string, amount float64) (MovePortfolioFundsResponse, error) {
	var resp MovePortfolioFundsResponse

	if from == "" || to == "" {
		return resp, errPortfolioIDEmpty
	}
	if currency == "" {
		return resp, errCurrencyEmpty
	}
	if amount == 0 {
		return resp, errAmountEmpty
	}

	funds := FundsData{Value: strconv.FormatFloat(amount, 'f', -1, 64), Currency: currency}

	req := map[string]interface{}{"source_portfolio_uuid": from, "target_portfolio_uuid": to, "funds": funds}

	path := fmt.Sprintf("%s%s/%s", coinbaseV3, coinbasePortfolios, coinbaseMoveFunds)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		path, "", req, true, &resp, nil)
}

// GetPortfolioByID provides detailed information on a single portfolio
func (c *CoinbasePro) GetPortfolioByID(ctx context.Context, portfolioID string) (*DetailedPortfolioResponse, error) {
	if portfolioID == "" {
		return nil, errPortfolioIDEmpty
	}

	path := fmt.Sprintf("%s%s/%s", coinbaseV3, coinbasePortfolios, portfolioID)

	var resp DetailedPortfolioResponse

	return &resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, "", nil, true, &resp, nil)
}

// DeletePortfolio deletes a portfolio
func (c *CoinbasePro) DeletePortfolio(ctx context.Context, portfolioID string) error {
	if portfolioID == "" {
		return errPortfolioIDEmpty
	}

	path := fmt.Sprintf("%s%s/%s", coinbaseV3, coinbasePortfolios, portfolioID)

	return c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, "", nil,
		true, nil, nil)
}

// EditPortfolio edits the name of a portfolio
func (c *CoinbasePro) EditPortfolio(ctx context.Context, portfolioID, name string) (SimplePortfolioResponse, error) {
	var resp SimplePortfolioResponse

	if portfolioID == "" {
		return resp, errPortfolioIDEmpty
	}
	if name == "" {
		return resp, errNameEmpty
	}

	req := map[string]interface{}{"name": name}

	path := fmt.Sprintf("%s%s/%s", coinbaseV3, coinbasePortfolios, portfolioID)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPut,
		path, "", req, true, &resp, nil)
}

// GetFuturesBalanceSummary returns information on balances related to Coinbase Financial Markets
// futures trading
func (c *CoinbasePro) GetFuturesBalanceSummary(ctx context.Context) (FuturesBalanceSummary, error) {
	var resp FuturesBalanceSummary

	path := fmt.Sprintf("%s%s/%s", coinbaseV3, coinbaseCFM, coinbaseBalanceSummary)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, "", nil, true, &resp, nil)
}

// GetAllFuturesPositions returns a list of all open positions in CFM futures products
func (c *CoinbasePro) GetAllFuturesPositions(ctx context.Context) (AllFuturesPositions, error) {
	var resp AllFuturesPositions

	path := fmt.Sprintf("%s%s/%s", coinbaseV3, coinbaseCFM, coinbasePositions)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, "", nil, true, &resp, nil)
}

// GetFuturesPositionByID returns information on a single open position in CFM futures products
func (c *CoinbasePro) GetFuturesPositionByID(ctx context.Context, productID string) (FuturesPosition, error) {
	var resp FuturesPosition

	if productID == "" {
		return resp, errProductIDEmpty
	}

	path := fmt.Sprintf("%s%s/%s/%s", coinbaseV3, coinbaseCFM, coinbasePositions, productID)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, "", nil, true, &resp, nil)
}

// ScheduleFuturesSweep schedules a sweep of funds from a CFTC-regulated futures account to a
// Coinbase USD Spot wallet. Request submitted before 5 pm ET are processed the following
// business day, requests submitted after are processed in 2 business days. Only one
// sweep request can be pending at a time. Funds transferred depend on the excess available
// in the futures account. An amount of 0 will sweep all available excess funds
func (c *CoinbasePro) ScheduleFuturesSweep(ctx context.Context, amount float64) (SuccessBool, error) {
	path := fmt.Sprintf("%s%s/%s/%s", coinbaseV3, coinbaseCFM, coinbaseSweeps, coinbaseSchedule)

	req := make(map[string]interface{})

	if amount != 0 {
		req["usd_amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	}

	var resp SuccessBool

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		path, "", req, true, &resp, nil)
}

// ListFuturesSweeps returns information on pending and/or processing requests to sweep funds
func (c *CoinbasePro) ListFuturesSweeps(ctx context.Context) (ListFuturesSweepsResponse, error) {
	var resp ListFuturesSweepsResponse

	path := fmt.Sprintf("%s%s/%s", coinbaseV3, coinbaseCFM, coinbaseSweeps)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, "", nil, true, &resp, nil)
}

// CancelPendingFuturesSweep cancels a pending sweep request
func (c *CoinbasePro) CancelPendingFuturesSweep(ctx context.Context) (SuccessBool, error) {
	path := fmt.Sprintf("%s%s/%s", coinbaseV3, coinbaseCFM, coinbaseSweeps)

	var resp SuccessBool

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete,
		path, "", nil, true, &resp, nil)
}

// GetTransactionSummary returns a summary of transactions with fee tiers, total volume,
// and fees
func (c *CoinbasePro) GetTransactionSummary(ctx context.Context, startDate, endDate time.Time, userNativeCurrency, productType, contractExpiryType string) (*TransactionSummary, error) {
	var params Params
	params.urlVals = url.Values{}

	err := params.prepareDateString(startDate, endDate, startDateString, endDateString)
	if err != nil {
		return nil, err
	}

	if contractExpiryType != "" {
		params.urlVals.Set("contract_expiry_type", contractExpiryType)
	}
	if productType != "" {
		params.urlVals.Set("product_type", productType)
	}

	params.urlVals.Set("user_native_currency", userNativeCurrency)

	pathParams := common.EncodeURLValues("", params.urlVals)

	var resp TransactionSummary

	return &resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		coinbaseV3+coinbaseTransactionSummary, pathParams, nil, true, &resp, nil)
}

// CreateConvertQuote creates a quote for a conversion between two currencies. The trade_id returned
// can be used to commit the trade, but that must be done within 10 minutes of the quote's creation
func (c *CoinbasePro) CreateConvertQuote(ctx context.Context, from, to, userIncentiveID, codeVal string, amount float64) (ConvertResponse, error) {
	var resp ConvertResponse
	if from == "" || to == "" {
		return resp, errAccountIDEmpty
	}
	if amount == 0 {
		return resp, errAmountEmpty
	}

	path := fmt.Sprintf("%s%s/%s", coinbaseV3, coinbaseConvert, coinbaseQuote)

	tIM := map[string]interface{}{"user_incentive_id": userIncentiveID, "code_val": codeVal}

	req := map[string]interface{}{"from_account": from, "to_account": to,
		"amount": strconv.FormatFloat(amount, 'f', -1, 64), "trade_incentive_metadata": tIM}

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path,
		"", req, true, &resp, nil)
}

// CommitConvertTrade commits a conversion between two currencies, using the trade_id returned
// from CreateConvertQuote
func (c *CoinbasePro) CommitConvertTrade(ctx context.Context, tradeID, from, to string) (ConvertResponse, error) {
	var resp ConvertResponse
	if tradeID == "" {
		return resp, errTransactionIDEmpty
	}
	if from == "" || to == "" {
		return resp, errAccountIDEmpty
	}

	path := fmt.Sprintf("%s%s/%s/%s", coinbaseV3, coinbaseConvert, coinbaseTrade, tradeID)

	req := map[string]interface{}{"from_account": from, "to_account": to}

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path,
		"", req, true, &resp, nil)
}

// GetConvertTradeByID returns information on a conversion between two currencies
func (c *CoinbasePro) GetConvertTradeByID(ctx context.Context, tradeID, from, to string) (ConvertResponse, error) {
	var resp ConvertResponse
	if tradeID == "" {
		return resp, errTransactionIDEmpty
	}
	if from == "" || to == "" {
		return resp, errAccountIDEmpty
	}

	path := fmt.Sprintf("%s%s/%s/%s", coinbaseV3, coinbaseConvert, coinbaseTrade, tradeID)

	req := map[string]interface{}{"from_account": from, "to_account": to}

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path,
		"", req, true, &resp, nil)
}

// GetV3Time returns the current server time, calling V3 of the API
func (c *CoinbasePro) GetV3Time(ctx context.Context) (ServerTimeV3, error) {
	var resp ServerTimeV3

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		coinbaseV3+coinbaseTime, "", nil, true, &resp, nil)
}

// ListNotifications lists the notifications the user is subscribed to
func (c *CoinbasePro) ListNotifications(ctx context.Context, pag PaginationInp) (ListNotificationsResponse, error) {
	var resp ListNotificationsResponse

	var params Params
	params.urlVals = url.Values{}
	params.preparePagination(pag)

	pathParams := common.EncodeURLValues("", params.urlVals)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		coinbaseV2+coinbaseNotifications, pathParams, nil, false, &resp, nil)
}

func (c *CoinbasePro) GetUserByID(ctx context.Context, userID string) (*UserResponse, error) {
	if userID == "" {
		return nil, errUserIDEmpty
	}

	path := fmt.Sprintf("%s%s/%s", coinbaseV2, coinbaseUsers, userID)

	var resp *UserResponse

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, "", nil, false, &resp, nil)
}

// GetCurrentUser returns information about the user associated with the API key
func (c *CoinbasePro) GetCurrentUser(ctx context.Context) (*UserResponse, error) {
	var resp *UserResponse

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		coinbaseV2+coinbaseUser, "", nil, false, &resp, nil)
}

// GetAuthInfo returns information about the scopes granted to the API key
func (c *CoinbasePro) GetAuthInfo(ctx context.Context) (AuthResponse, error) {
	var resp AuthResponse

	path := fmt.Sprintf("%s%s/%s", coinbaseV2, coinbaseUser, coinbaseAuth)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, "", nil, false, &resp, nil)
}

// UpdateUser modifies certain user preferences
func (c *CoinbasePro) UpdateUser(ctx context.Context, name, timeZone, nativeCurrency string) (*UserResponse, error) {
	var resp *UserResponse

	req := map[string]interface{}{}

	if name != "" {
		req["name"] = name
	}
	if timeZone != "" {
		req["time_zone"] = timeZone
	}
	if nativeCurrency != "" {
		req["native_currency"] = nativeCurrency
	}

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPut,
		coinbaseV2+coinbaseUser, "", req, false, &resp, nil)
}

// CreateWallet creates a new wallet for the specified currency
func (c *CoinbasePro) CreateWallet(ctx context.Context, currency string) (*GenWalletResponse, error) {
	if currency == "" {
		return nil, errCurrencyEmpty
	}

	path := fmt.Sprintf("%s%s/%s", coinbaseV2, coinbaseAccounts, currency)

	var resp *GenWalletResponse

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, "", nil, false, &resp, nil)
}

// GetAllWallets lists all accounts associated with the API key
func (c *CoinbasePro) GetAllWallets(ctx context.Context, pag PaginationInp) (GetAllWalletsResponse, error) {
	var resp GetAllWalletsResponse

	var params Params
	params.urlVals = url.Values{}
	params.preparePagination(pag)

	pathParams := common.EncodeURLValues("", params.urlVals)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		coinbaseV2+coinbaseAccounts, pathParams, nil, false, &resp, nil)
}

// GetWalletByID returns information about a single wallet. In lieu of a wallet ID,
// a currency can be provided to get the primary account for that currency
func (c *CoinbasePro) GetWalletByID(ctx context.Context, walletID, currency string) (*GenWalletResponse, error) {
	if (walletID == "" && currency == "") || (walletID != "" && currency != "") {
		return nil, errCurrWalletConflict
	}

	var path string

	if walletID != "" {
		path = fmt.Sprintf("%s%s/%s", coinbaseV2, coinbaseAccounts, walletID)
	}
	if currency != "" {
		path = fmt.Sprintf("%s%s/%s", coinbaseV2, coinbaseAccounts, currency)
	}

	var resp *GenWalletResponse

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, "", nil, false, &resp, nil)
}

// UpdateWalletName updates the name of a wallet
func (c *CoinbasePro) UpdateWalletName(ctx context.Context, walletID, newName string) (*GenWalletResponse, error) {
	if walletID == "" {
		return nil, errWalletIDEmpty
	}

	path := fmt.Sprintf("%s%s/%s", coinbaseV2, coinbaseAccounts, walletID)

	req := map[string]interface{}{"name": newName}

	var resp *GenWalletResponse

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPut,
		path, "", req, false, &resp, nil)
}

// DeleteWallet deletes a wallet
func (c *CoinbasePro) DeleteWallet(ctx context.Context, walletID string) error {
	if walletID == "" {
		return errWalletIDEmpty
	}

	path := fmt.Sprintf("%s%s/%s", coinbaseV2, coinbaseAccounts, walletID)

	return c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, "", nil,
		false, nil, nil)
}

// CreateAddress generates a crypto address for depositing to the specified wallet
func (c *CoinbasePro) CreateAddress(ctx context.Context, walletID, name string) (*GenAddrResponse, error) {
	if walletID == "" {
		return nil, errWalletIDEmpty
	}

	path := fmt.Sprintf("%s%s/%s/%s", coinbaseV2, coinbaseAccounts, walletID, coinbaseAddresses)

	req := map[string]interface{}{"name": name}

	var resp *GenAddrResponse

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		path, "", req, false, &resp, nil)
}

// GetAllAddresses returns information on all addresses associated with a wallet
func (c *CoinbasePro) GetAllAddresses(ctx context.Context, walletID string, pag PaginationInp) (GetAllAddrResponse, error) {
	var resp GetAllAddrResponse

	if walletID == "" {
		return resp, errWalletIDEmpty
	}

	path := fmt.Sprintf("%s%s/%s/%s", coinbaseV2, coinbaseAccounts, walletID, coinbaseAddresses)

	var params Params
	params.urlVals = url.Values{}
	params.preparePagination(pag)

	pathParams := common.EncodeURLValues("", params.urlVals)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, pathParams, nil, false, &resp, nil)
}

// GetAddressByID returns information on a single address associated with the specified wallet
func (c *CoinbasePro) GetAddressByID(ctx context.Context, walletID, addressID string) (*GenAddrResponse, error) {
	if walletID == "" {
		return nil, errWalletIDEmpty
	}
	if addressID == "" {
		return nil, errAddressIDEmpty
	}

	path := fmt.Sprintf("%s%s/%s/%s/%s", coinbaseV2, coinbaseAccounts, walletID, coinbaseAddresses,
		addressID)

	var resp *GenAddrResponse

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, "", nil, false, &resp, nil)
}

// GetAddressTransactions returns a list of transactions associated with the specified address
func (c *CoinbasePro) GetAddressTransactions(ctx context.Context, walletID, addressID string, pag PaginationInp) (ManyTransactionsResp, error) {
	var resp ManyTransactionsResp

	if walletID == "" {
		return resp, errWalletIDEmpty
	}
	if addressID == "" {
		return resp, errAddressIDEmpty
	}

	path := fmt.Sprintf("%s%s/%s/%s/%s/%s", coinbaseV2, coinbaseAccounts, walletID,
		coinbaseAddresses, addressID, coinbaseTransactions)

	var params Params
	params.urlVals = url.Values{}
	params.preparePagination(pag)

	pathParams := common.EncodeURLValues("", params.urlVals)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, pathParams, nil, false, &resp, nil)
}

// SendMoney can send funds to an email or cryptocurrency address (if "traType" is set to "send"),
// or to another one of the user's wallets or vaults (if "traType" is set to "transfer"). Coinbase
// may delay or cancel the transaction at their discretion. The "idem" parameter is an optional
// string for idempotency; a token with a max length of 100 characters, if a previous
// transaction included the same token as a parameter, the new transaction won't be processed,
// and information on the previous transaction will be returned instead
func (c *CoinbasePro) SendMoney(ctx context.Context, traType, walletID, to, currency, description, idem, financialInstitutionWebsite, destinationTag string, amount float64, skipNotifications, toFinancialInstitution bool) (*GenTransactionResp, error) {
	if traType == "" {
		return nil, errTransactionTypeEmpty
	}
	if walletID == "" {
		return nil, errWalletIDEmpty
	}
	if to == "" {
		return nil, errToEmpty
	}
	if amount == 0 {
		return nil, errAmountEmpty
	}
	if currency == "" {
		return nil, errCurrencyEmpty
	}

	path := fmt.Sprintf("%s%s/%s/%s", coinbaseV2, coinbaseAccounts, walletID, coinbaseTransactions)

	req := map[string]interface{}{"type": traType, "to": to,
		"amount": strconv.FormatFloat(amount, 'f', -1, 64), "currency": currency,
		"description": description, "skip_notifications": skipNotifications, "idem": idem,
		"to_financial_institution":      toFinancialInstitution,
		"financial_institution_website": financialInstitutionWebsite,
		"destination_tag":               destinationTag}

	var resp *GenTransactionResp

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		path, "", req, false, &resp, nil)
}

// GetAllTransactions returns a list of transactions associated with the specified wallet
func (c *CoinbasePro) GetAllTransactions(ctx context.Context, walletID string, pag PaginationInp) (ManyTransactionsResp, error) {
	var resp ManyTransactionsResp

	if walletID == "" {
		return resp, errWalletIDEmpty
	}

	path := fmt.Sprintf("%s%s/%s/%s", coinbaseV2, coinbaseAccounts, walletID, coinbaseTransactions)

	var params Params
	params.urlVals = url.Values{}
	params.preparePagination(pag)

	pathParams := common.EncodeURLValues("", params.urlVals)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, pathParams, nil, false, &resp, nil)
}

// GetTransactionByID returns information on a single transaction associated with the
// specified wallet
func (c *CoinbasePro) GetTransactionByID(ctx context.Context, walletID, transactionID string) (*GenTransactionResp, error) {
	if walletID == "" {
		return nil, errWalletIDEmpty
	}
	if transactionID == "" {
		return nil, errTransactionIDEmpty
	}

	path := fmt.Sprintf("%s%s/%s/%s/%s", coinbaseV2, coinbaseAccounts, walletID,
		coinbaseTransactions, transactionID)

	var resp *GenTransactionResp

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, "", nil, false, &resp, nil)
}

// FiatTransfer prepares and optionally processes a transfer of funds between the exchange and a
// fiat payment method. "Deposit" signifies funds going from exchange to bank, "withdraw"
// signifies funds going from bank to exchange
func (c *CoinbasePro) FiatTransfer(ctx context.Context, walletID, currency, paymentMethod string, amount float64, commit bool, transferType FiatTransferType) (*GenDeposWithdrResp, error) {
	if walletID == "" {
		return nil, errWalletIDEmpty
	}
	if amount == 0 {
		return nil, errAmountEmpty
	}
	if currency == "" {
		return nil, errCurrencyEmpty
	}
	if paymentMethod == "" {
		return nil, errPaymentMethodEmpty
	}

	var path string
	switch transferType {
	case FiatDeposit:
		path = fmt.Sprintf("%s%s/%s/%s", coinbaseV2, coinbaseAccounts, walletID, coinbaseDeposits)
	case FiatWithdrawal:
		path = fmt.Sprintf("%s%s/%s/%s", coinbaseV2, coinbaseAccounts, walletID, coinbaseWithdrawals)
	}

	req := map[string]interface{}{"currency": currency, "payment_method": paymentMethod,
		"amount": strconv.FormatFloat(amount, 'f', -1, 64), "commit": commit}

	var resp *GenDeposWithdrResp

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		path, "", req, false, &resp, nil)
}

// CommitTransfer processes a deposit/withdrawal that was created with the "commit" parameter set
// to false
func (c *CoinbasePro) CommitTransfer(ctx context.Context, walletID, depositID string, transferType FiatTransferType) (*GenDeposWithdrResp, error) {
	if walletID == "" {
		return nil, errWalletIDEmpty
	}
	if depositID == "" {
		return nil, errDepositIDEmpty
	}

	var path string
	switch transferType {
	case FiatDeposit:
		path = fmt.Sprintf("%s%s/%s/%s/%s/%s", coinbaseV2, coinbaseAccounts, walletID,
			coinbaseDeposits, depositID, coinbaseCommit)
	case FiatWithdrawal:
		path = fmt.Sprintf("%s%s/%s/%s/%s/%s", coinbaseV2, coinbaseAccounts, walletID,
			coinbaseWithdrawals, depositID, coinbaseCommit)
	}

	var resp *GenDeposWithdrResp

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		path, "", nil, false, &resp, nil)
}

// GetAllFiatTransfers returns a list of transfers either to or from fiat payment methods and
// the specified wallet
func (c *CoinbasePro) GetAllFiatTransfers(ctx context.Context, walletID string, pag PaginationInp, transferType FiatTransferType) (ManyDeposWithdrResp, error) {
	var resp ManyDeposWithdrResp

	if walletID == "" {
		return resp, errWalletIDEmpty
	}

	var path string
	switch transferType {
	case FiatDeposit:
		path = fmt.Sprintf("%s%s/%s/%s", coinbaseV2, coinbaseAccounts, walletID, coinbaseDeposits)
	case FiatWithdrawal:
		path = fmt.Sprintf("%s%s/%s/%s", coinbaseV2, coinbaseAccounts, walletID, coinbaseWithdrawals)
	}

	var params Params
	params.urlVals = url.Values{}
	params.preparePagination(pag)

	pathParams := common.EncodeURLValues("", params.urlVals)

	err := c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, pathParams, nil, false, &resp, nil)

	if err != nil {
		return resp, err
	}

	for i := range resp.Data {
		resp.Data[i].TransferType = transferType
	}

	return resp, nil
}

// GetFiatTransferByID returns information on a single deposit/withdrawal associated with the specified wallet
func (c *CoinbasePro) GetFiatTransferByID(ctx context.Context, walletID, depositID string, transferType FiatTransferType) (*GenDeposWithdrResp, error) {
	if walletID == "" {
		return nil, errWalletIDEmpty
	}
	if depositID == "" {
		return nil, errDepositIDEmpty
	}

	path := fmt.Sprintf("%s%s/%s/%s/%s", coinbaseV2, coinbaseAccounts, walletID,
		coinbaseDeposits, depositID)

	var resp *GenDeposWithdrResp

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, "", nil, false, &resp, nil)
}

// GetAllPaymentMethods returns a list of all payment methods associated with the user's account
func (c *CoinbasePro) GetAllPaymentMethods(ctx context.Context, pag PaginationInp) (GetAllPaymentMethodsResp, error) {
	var resp GetAllPaymentMethodsResp

	path := fmt.Sprintf("%s%s", coinbaseV2, coinbasePaymentMethods)

	var params Params
	params.urlVals = url.Values{}
	params.preparePagination(pag)

	pathParams := common.EncodeURLValues("", params.urlVals)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, pathParams, nil, false, &resp, nil)
}

// GetPaymentMethodByID returns information on a single payment method associated with the user's
// account
func (c *CoinbasePro) GetPaymentMethodByID(ctx context.Context, paymentMethodID string) (*GenPaymentMethodResp, error) {
	if paymentMethodID == "" {
		return nil, errPaymentMethodEmpty
	}

	path := fmt.Sprintf("%s%s/%s", coinbaseV2, coinbasePaymentMethods, paymentMethodID)

	var resp *GenPaymentMethodResp

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, "", nil, false, &resp, nil)
}

// GetFiatCurrencies lists currencies that Coinbase knows about
func (c *CoinbasePro) GetFiatCurrencies(ctx context.Context) (GetFiatCurrenciesResp, error) {
	var resp GetFiatCurrenciesResp

	return resp, c.SendHTTPRequest(ctx, exchange.RestSpot, coinbaseV2+coinbaseCurrencies, &resp)
}

// GetCryptocurrencies lists cryptocurrencies that Coinbase knows about
func (c *CoinbasePro) GetCryptocurrencies(ctx context.Context) (GetCryptocurrenciesResp, error) {
	var resp GetCryptocurrenciesResp

	path := fmt.Sprintf("%s%s/%s", coinbaseV2, coinbaseCurrencies, coinbaseCrypto)

	return resp, c.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GetExchangeRates returns exchange rates for the specified currency. If none is specified,
// it defaults to USD
func (c *CoinbasePro) GetExchangeRates(ctx context.Context, currency string) (GetExchangeRatesResp, error) {
	var resp GetExchangeRatesResp

	var params Params
	params.urlVals = url.Values{}

	params.urlVals.Set("currency", currency)

	path := common.EncodeURLValues(coinbaseV2+coinbaseExchangeRates, params.urlVals)

	return resp, c.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GetPrice returns the price the spot/buy/sell price for the specified currency pair,
// including the standard Coinbase fee of 1%, but excluding any other fees
func (c *CoinbasePro) GetPrice(ctx context.Context, currencyPair, priceType string) (GetPriceResp, error) {
	var resp GetPriceResp

	var path string
	switch priceType {
	case "spot":
		path = fmt.Sprintf("%s%s/%s/spot", coinbaseV2, coinbasePrices, currencyPair)
	case "buy":
		path = fmt.Sprintf("%s%s/%s/buy", coinbaseV2, coinbasePrices, currencyPair)
	case "sell":
		path = fmt.Sprintf("%s%s/%s/sell", coinbaseV2, coinbasePrices, currencyPair)
	default:
		return resp, errInvalidPriceType
	}

	return resp, c.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GetV2Time returns the current server time, calling V2 of the API
func (c *CoinbasePro) GetV2Time(ctx context.Context) (ServerTimeV2, error) {
	resp := ServerTimeV2{}
	return resp, c.SendHTTPRequest(ctx, exchange.RestSpot, coinbaseV2+coinbaseTime, &resp)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (c *CoinbasePro) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, result interface{}) error {
	endpoint, err := c.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	item := &request.Item{
		Method:        http.MethodGet,
		Path:          endpoint + path,
		Result:        result,
		Verbose:       c.Verbose,
		HTTPDebugging: c.HTTPDebugging,
		HTTPRecording: c.HTTPRecording,
	}

	return c.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request
func (c *CoinbasePro) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, method, path, queryParams string, bodyParams map[string]interface{}, istrue bool, result interface{}, returnHead *http.Header) (err error) {
	creds, err := c.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpoint, err := c.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	// Version 2 wants query params in the path during signing
	if !istrue {
		path = path + queryParams
	}

	interim := json.RawMessage{}
	newRequest := func() (*request.Item, error) {
		payload := []byte("")
		if bodyParams != nil {
			payload, err = json.Marshal(bodyParams)
			if err != nil {
				return nil, err
			}
		}

		n := strconv.FormatInt(time.Now().Unix(), 10)

		message := n + method + path + string(payload)

		hmac, err := crypto.GetHMAC(crypto.HashSHA256,
			[]byte(message),
			[]byte(creds.Secret))
		if err != nil {
			return nil, err
		}

		headers := make(map[string]string)
		headers["CB-ACCESS-KEY"] = creds.Key
		headers["CB-ACCESS-SIGN"] = hex.EncodeToString(hmac)
		headers["CB-ACCESS-TIMESTAMP"] = n
		headers["Content-Type"] = "application/json"
		headers["CB-VERSION"] = "2023-11-13"

		// Version 3 only wants query params in the path when the request is sent
		if istrue {
			path = path + queryParams
		}

		return &request.Item{
			Method:         method,
			Path:           endpoint + path,
			Headers:        headers,
			Body:           bytes.NewBuffer(payload),
			Result:         &interim,
			Verbose:        c.Verbose,
			HTTPDebugging:  c.HTTPDebugging,
			HTTPRecording:  c.HTTPRecording,
			HeaderResponse: returnHead,
		}, nil
	}
	rateLim := V2Rate
	if istrue {
		rateLim = V3Rate
	}

	err = c.SendPayload(ctx, rateLim, newRequest, request.AuthenticatedRequest)

	// Doing this error handling because the docs indicate that errors can be returned even with a 200 status code,
	// and that these errors can be buried in the JSON returned
	if err != nil {
		return err
	}
	singleErrCap := struct {
		ErrorType             string `json:"error"`
		Message               string `json:"message"`
		ErrorDetails          string `json:"error_details"`
		EditFailureReason     string `json:"edit_failure_reason"`
		PreviewFailureReason  string `json:"preview_failure_reason"`
		NewOrderFailureReason string `json:"new_order_failure_reason"`
	}{}
	if err := json.Unmarshal(interim, &singleErrCap); err == nil {
		if singleErrCap.Message != "" {
			errMessage := fmt.Sprintf("message: %s, error type: %s, error details: %s, edit failure reason: %s, preview failure reason: %s, new order failure reason: %s",
				singleErrCap.Message, singleErrCap.ErrorType, singleErrCap.ErrorDetails, singleErrCap.EditFailureReason,
				singleErrCap.PreviewFailureReason, singleErrCap.NewOrderFailureReason)
			return errors.New(errMessage)
		}
	}
	manyErrCap := struct {
		Errors []struct {
			Success              bool   `json:"success"`
			FailureReason        string `json:"failure_reason"`
			OrderID              string `json:"order_id"`
			EditFailureReason    string `json:"edit_failure_reason"`
			PreviewFailureReason string `json:"preview_failure_reason"`
		}
	}{}
	err = json.Unmarshal(interim, &manyErrCap)
	if err == nil {
		if len(manyErrCap.Errors) > 0 {
			errMessage := ""
			for i := range manyErrCap.Errors {
				if !manyErrCap.Errors[i].Success || manyErrCap.Errors[i].EditFailureReason != "" ||
					manyErrCap.Errors[i].PreviewFailureReason != "" {
					errMessage += fmt.Sprintf("order id: %s, failure reason: %s, edit failure reason: %s, preview failure reason: %s",
						manyErrCap.Errors[i].OrderID, manyErrCap.Errors[i].FailureReason,
						manyErrCap.Errors[i].EditFailureReason, manyErrCap.Errors[i].PreviewFailureReason)
				}
			}
			if errMessage != "" {
				return errors.New(errMessage)
			}
		}
	}
	if result == nil {
		return nil
	}
	return json.Unmarshal(interim, result)
}

// GetFee returns an estimate of fee based on type of transaction
func (c *CoinbasePro) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	var fee float64
	switch {
	case !isStablePair(feeBuilder.Pair) && feeBuilder.FeeType == exchange.CryptocurrencyTradeFee:
		fees, err := c.GetTransactionSummary(ctx, time.Now().Add(-time.Hour*24*30), time.Now(), "", "", "")
		if err != nil {
			return 0, err
		}
		if feeBuilder.IsMaker {
			fee = fees.FeeTier.MakerFeeRate
		} else {
			fee = fees.FeeTier.TakerFeeRate
		}
	case feeBuilder.IsMaker && isStablePair(feeBuilder.Pair) &&
		(feeBuilder.FeeType == exchange.CryptocurrencyTradeFee || feeBuilder.FeeType == exchange.OfflineTradeFee):
		fee = 0
	case !feeBuilder.IsMaker && isStablePair(feeBuilder.Pair) &&
		(feeBuilder.FeeType == exchange.CryptocurrencyTradeFee || feeBuilder.FeeType == exchange.OfflineTradeFee):
		fee = 0.00001
	case feeBuilder.IsMaker && !isStablePair(feeBuilder.Pair) && feeBuilder.FeeType == exchange.OfflineTradeFee:
		fee = 0.006
	case !feeBuilder.IsMaker && !isStablePair(feeBuilder.Pair) && feeBuilder.FeeType == exchange.OfflineTradeFee:
		fee = 0.008
	default:
		return 0, errFeeTypeNotSupported
	}
	return fee * feeBuilder.Amount * feeBuilder.PurchasePrice, nil
}

var stableMap = map[key.PairAsset]bool{
	{Base: currency.USDT.Item, Quote: currency.USD.Item}:  true,
	{Base: currency.USDT.Item, Quote: currency.EUR.Item}:  true,
	{Base: currency.USDC.Item, Quote: currency.EUR.Item}:  true,
	{Base: currency.USDC.Item, Quote: currency.GBP.Item}:  true,
	{Base: currency.USDT.Item, Quote: currency.GBP.Item}:  true,
	{Base: currency.USDT.Item, Quote: currency.USDC.Item}: true,
	{Base: currency.DAI.Item, Quote: currency.USD.Item}:   true,
	{Base: currency.CBETH.Item, Quote: currency.ETH.Item}: true,
	{Base: currency.PYUSD.Item, Quote: currency.USD.Item}: true,
	{Base: currency.EUROC.Item, Quote: currency.USD.Item}: true,
	{Base: currency.GUSD.Item, Quote: currency.USD.Item}:  true,
	{Base: currency.EUROC.Item, Quote: currency.EUR.Item}: true,
	{Base: currency.WBTC.Item, Quote: currency.BTC.Item}:  true,
	{Base: currency.LSETH.Item, Quote: currency.ETH.Item}: true,
	{Base: currency.GYEN.Item, Quote: currency.USD.Item}:  true,
	{Base: currency.PAX.Item, Quote: currency.USD.Item}:   true,
}

// IsStablePair returns true if the currency pair is considered a "stable pair" by Coinbase
func isStablePair(pair currency.Pair) bool {
	return stableMap[key.PairAsset{Base: pair.Base.Item, Quote: pair.Quote.Item}]
}

// PrepareDateString encodes a set of parameters indicating start & end dates
func (p *Params) prepareDateString(startDate, endDate time.Time, labelStart, labelEnd string) error {
	err := common.StartEndTimeCheck(startDate, endDate)

	if err == nil {
		p.urlVals.Set(labelStart, startDate.Format(time.RFC3339))
		p.urlVals.Set(labelEnd, endDate.Format(time.RFC3339))
	}

	if err != nil {
		if err.Error() == "start date unset" || err.Error() == "end date unset" {
			return nil
		}
	}

	return err
}

func (p *Params) preparePagination(pag PaginationInp) {
	if pag.Limit != 0 {
		p.urlVals.Set("limit", strconv.FormatInt(int64(pag.Limit), 10))
	}
	if pag.OrderAscend {
		p.urlVals.Set("order", "asc")
	}
	if pag.StartingAfter != "" {
		p.urlVals.Set("starting_after", pag.StartingAfter)
	}
	if pag.EndingBefore != "" {
		p.urlVals.Set("ending_before", pag.EndingBefore)
	}
}

func (f FiatTransferType) String() string {
	if f {
		return "withdrawal"
	}
	return "deposit"
}

func (t *UnixTimestamp) UnmarshalJSON(b []byte) error {
	var timestampStr string
	err := json.Unmarshal(b, &timestampStr)
	if err != nil {
		return err
	}
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return err
	}
	*t = UnixTimestamp(time.Unix(timestamp, 0).UTC())
	return nil
}

func (t *UnixTimestamp) String() string {
	return t.Time().String()
}

func (t UnixTimestamp) Time() time.Time {
	return time.Time(t)
}
