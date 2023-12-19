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
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
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

	Version3        Version          = true
	Version2        Version          = false
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
		coinbaseV3+coinbaseAccounts, pathParams, nil, Version3, &resp, nil)
}

// GetAccountByID returns information for a single account
func (c *CoinbasePro) GetAccountByID(ctx context.Context, accountID string) (*Account, error) {
	if accountID == "" {
		return nil, errAccountIDEmpty
	}
	path := fmt.Sprintf("%s%s/%s", coinbaseV3, coinbaseAccounts, accountID)
	resp := OneAccountResponse{}

	return &resp.Account, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, "", nil, Version3, &resp, nil)
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
		coinbaseV3+coinbaseBestBidAsk, pathParams, nil, Version3, &resp, nil)
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
		coinbaseV3+coinbaseProductBook, pathParams, nil, Version3, &resp, nil)
}

// GetAllProducts returns information on all currency pairs that are available for trading
func (c *CoinbasePro) GetAllProducts(ctx context.Context, limit, offset int32, productType, contractExpiryType string, productIDs []string) (AllProducts, error) {
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
		coinbaseV3+coinbaseProducts, pathParams, nil, Version3, &products, nil)
}

// GetProductByID returns information on a single specified currency pair
func (c *CoinbasePro) GetProductByID(ctx context.Context, productID string) (*Product, error) {
	if productID == "" {
		return nil, errProductIDEmpty
	}

	path := fmt.Sprintf("%s%s/%s", coinbaseV3, coinbaseProducts, productID)

	resp := Product{}

	return &resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, "", nil, Version3, &resp, nil)
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
		path, pathParams, nil, Version3, &resp, nil)

	return resp, err
}

// GetTicker returns snapshot information about the last trades (ticks) and best bid/ask.
// Contrary to documentation, this does not tell you the 24h volume
func (c *CoinbasePro) GetTicker(ctx context.Context, productID string, limit uint16) (*Ticker, error) {
	if productID == "" {
		return nil, errProductIDEmpty
	}
	path := fmt.Sprintf(
		"%s%s/%s/%s", coinbaseV3, coinbaseProducts, productID, coinbaseTicker)

	var params Params
	params.urlVals = url.Values{}
	params.urlVals.Set("limit", strconv.FormatInt(int64(limit), 10))

	pathParams := common.EncodeURLValues("", params.urlVals)

	var resp Ticker

	return &resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, pathParams, nil, Version3, &resp, nil)
}

// PlaceOrder places either a limit, market, or stop order
func (c *CoinbasePro) PlaceOrder(ctx context.Context, clientOID, productID, side, stopDirection, orderType string, amount, limitPrice, stopPrice float64, postOnly bool, endTime time.Time) (*PlaceOrderResp, error) {
	if clientOID == "" {
		return nil, errClientOrderIDEmpty
	}
	if productID == "" {
		return nil, errProductIDEmpty
	}
	if amount == 0 {
		return nil, errAmountEmpty
	}

	var resp PlaceOrderResp

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
		return &resp, errInvalidOrderType
	}

	req := map[string]interface{}{"client_order_id": clientOID, "product_id": productID,
		"side": side, "order_configuration": orderConfig}

	return &resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
			coinbaseV3+coinbaseOrders, "", req, Version3, &resp, nil)
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
		req, Version3, &resp, nil)
}

// EditOrder edits an order to a new size or price. Only limit orders with a good-till-cancelled time
// in force can be edited
func (c *CoinbasePro) EditOrder(ctx context.Context, orderID string, size, price float64) (bool, error) {
	if orderID == "" {
		return false, errOrderIDEmpty
	}
	if size == 0 && price == 0 {
		return false, errSizeAndPriceZero
	}

	path := fmt.Sprintf("%s%s/%s", coinbaseV3, coinbaseOrders, coinbaseEdit)

	req := map[string]interface{}{"order_id": orderID, "size": strconv.FormatFloat(size, 'f', -1, 64),
		"price": strconv.FormatFloat(price, 'f', -1, 64)}

	var resp bool

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, "",
		req, Version3, &resp, nil)
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
		req, Version3, &resp, nil)
}

// GetAllOrders lists orders, filtered by their status
func (c *CoinbasePro) GetAllOrders(ctx context.Context, productID, userNativeCurrency, orderType, orderSide, cursor, productType, orderPlacementSource, contractExpiryType string, orderStatus []string, limit int32, startDate, endDate time.Time) (GetAllOrdersResp, error) {
	var resp GetAllOrdersResp

	var params Params
	params.urlVals = make(url.Values)
	err := params.PrepareDateString(startDate, endDate, startDateString, endDateString)
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
		pathParams, nil, Version3, &resp, nil)
}

// GetFills returns information of recent fills on the specified profile
func (c *CoinbasePro) GetFills(ctx context.Context, orderID, productID, cursor string, limit uint16, startDate, endDate time.Time) (FillResponse, error) {
	var resp FillResponse
	var params Params
	params.urlVals = url.Values{}
	err := params.PrepareDateString(startDate, endDate, "start_sequence_timestamp",
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
		pathParams, nil, Version3, &resp, nil)
}

// GetOrderByID returns a single order by order id.
func (c *CoinbasePro) GetOrderByID(ctx context.Context, orderID, userNativeCurrency, clientID string) (*GetOrderResponse, error) {
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

	return &resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, pathParams, nil, Version3, &resp, nil)
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
		coinbaseV3+coinbasePortfolios, pathParams, nil, Version3, &resp, nil)
}

// CreatePortfolio creates a new portfolio
func (c *CoinbasePro) CreatePortfolio(ctx context.Context, name string) (SimplePortfolioResponse, error) {
	var resp SimplePortfolioResponse

	if name == "" {
		return resp, errNameEmpty
	}

	req := map[string]interface{}{"name": name}

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		coinbaseV3+coinbasePortfolios, "", req, Version3, &resp, nil)
}

// MovePortfolioFunds transfers funds between portfolios
func (c *CoinbasePro) MovePortfolioFunds(ctx context.Context, from, to, currency string, amount float64) (MovePortfolioFundsResponse, error) {
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
		path, "", req, Version3, &resp, nil)
}

// GetPortfolioByID provides detailed information on a single portfolio
func (c *CoinbasePro) GetPortfolioByID(ctx context.Context, portfolioID string) (*DetailedPortfolioResponse, error) {
	if portfolioID == "" {
		return nil, errPortfolioIDEmpty
	}

	path := fmt.Sprintf("%s%s/%s", coinbaseV3, coinbasePortfolios, portfolioID)

	var resp DetailedPortfolioResponse

	return &resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, "", nil, Version3, &resp, nil)
}

// DeletePortfolio deletes a portfolio
func (c *CoinbasePro) DeletePortfolio(ctx context.Context, portfolioID string) error {
	if portfolioID == "" {
		return errPortfolioIDEmpty
	}

	path := fmt.Sprintf("%s%s/%s", coinbaseV3, coinbasePortfolios, portfolioID)

	return c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, "", nil,
		Version3, nil, nil)
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
		path, "", req, Version3, &resp, nil)
}

// GetTransactionSummary returns a summary of transactions with fee tiers, total volume,
// and fees
func (c *CoinbasePro) GetTransactionSummary(ctx context.Context, startDate, endDate time.Time, userNativeCurrency, productType, contractExpiryType string) (*TransactionSummary, error) {
	var params Params
	params.urlVals = url.Values{}

	err := params.PrepareDateString(startDate, endDate, startDateString, endDateString)
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
		coinbaseV3+coinbaseTransactionSummary, pathParams, nil, Version3, &resp, nil)
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
		"", req, Version3, &resp, nil)
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
		"", req, Version3, &resp, nil)
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
		"", req, Version3, &resp, nil)
}

// GetV3Time returns the current server time, calling V3 of the API
func (c *CoinbasePro) GetV3Time(ctx context.Context) (ServerTimeV3, error) {
	var resp ServerTimeV3

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		coinbaseV3+coinbaseTime, "", nil, Version3, &resp, nil)
}

// ListNotifications lists the notifications the user is subscribed to
func (c *CoinbasePro) ListNotifications(ctx context.Context, pag PaginationInp) (ListNotificationsResponse, error) {
	var resp ListNotificationsResponse

	var params Params
	params.urlVals = url.Values{}
	params.PreparePagination(pag)

	pathParams := common.EncodeURLValues("", params.urlVals)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		coinbaseV2+coinbaseNotifications, pathParams, nil, Version2, &resp, nil)
}

func (c *CoinbasePro) GetUserByID(ctx context.Context, userID string) (*UserResponse, error) {
	if userID == "" {
		return nil, errUserIDEmpty
	}

	path := fmt.Sprintf("%s%s/%s", coinbaseV2, coinbaseUsers, userID)

	var resp *UserResponse

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, "", nil, Version2, &resp, nil)
}

// GetCurrentUser returns information about the user associated with the API key
func (c *CoinbasePro) GetCurrentUser(ctx context.Context) (*UserResponse, error) {
	var resp *UserResponse

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		coinbaseV2+coinbaseUser, "", nil, Version2, &resp, nil)
}

// GetAuthInfo returns information about the scopes granted to the API key
func (c *CoinbasePro) GetAuthInfo(ctx context.Context) (AuthResponse, error) {
	var resp AuthResponse

	path := fmt.Sprintf("%s%s/%s", coinbaseV2, coinbaseUser, coinbaseAuth)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, "", nil, Version2, &resp, nil)
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
		coinbaseV2+coinbaseUser, "", req, Version2, &resp, nil)
}

// CreateWallet creates a new wallet for the specified currency
func (c *CoinbasePro) CreateWallet(ctx context.Context, currency string) (*GenWalletResponse, error) {
	if currency == "" {
		return nil, errCurrencyEmpty
	}

	path := fmt.Sprintf("%s%s/%s", coinbaseV2, coinbaseAccounts, currency)

	var resp *GenWalletResponse

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, "", nil, Version2, &resp, nil)
}

// GetAllWallets lists all accounts associated with the API key
func (c *CoinbasePro) GetAllWallets(ctx context.Context, pag PaginationInp) (GetAllWalletsResponse, error) {
	var resp GetAllWalletsResponse

	var params Params
	params.urlVals = url.Values{}
	params.PreparePagination(pag)

	pathParams := common.EncodeURLValues("", params.urlVals)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		coinbaseV2+coinbaseAccounts, pathParams, nil, Version2, &resp, nil)
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
		path, "", nil, Version2, &resp, nil)
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
		path, "", req, Version2, &resp, nil)
}

// DeleteWallet deletes a wallet
func (c *CoinbasePro) DeleteWallet(ctx context.Context, walletID string) error {
	if walletID == "" {
		return errWalletIDEmpty
	}

	path := fmt.Sprintf("%s%s/%s", coinbaseV2, coinbaseAccounts, walletID)

	return c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, "", nil,
		Version2, nil, nil)
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
		path, "", req, Version2, &resp, nil)
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
	params.PreparePagination(pag)

	pathParams := common.EncodeURLValues("", params.urlVals)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, pathParams, nil, Version2, &resp, nil)
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
		path, "", nil, Version2, &resp, nil)
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
	params.PreparePagination(pag)

	pathParams := common.EncodeURLValues("", params.urlVals)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, pathParams, nil, Version2, &resp, nil)
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
		path, "", req, Version2, &resp, nil)
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
	params.PreparePagination(pag)

	pathParams := common.EncodeURLValues("", params.urlVals)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, pathParams, nil, Version2, &resp, nil)
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
		path, "", nil, Version2, &resp, nil)
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
		path, "", req, Version2, &resp, nil)
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
		path, "", nil, Version2, &resp, nil)
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
	params.PreparePagination(pag)

	pathParams := common.EncodeURLValues("", params.urlVals)

	err := c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, pathParams, nil, Version2, &resp, nil)

	if err != nil {
		return resp, err
	}

	for i := range resp.Data {
		resp.Data[i].TransferType = transferType
	}

	return resp, nil
}

// GetFiatTransferByID returns information on a single deposit associated with the specified wallet
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
		path, "", nil, Version2, &resp, nil)
}

// GetAllPaymentMethods returns a list of all payment methods associated with the user's account
func (c *CoinbasePro) GetAllPaymentMethods(ctx context.Context, pag PaginationInp) (GetAllPaymentMethodsResp, error) {
	var resp GetAllPaymentMethodsResp

	path := fmt.Sprintf("%s%s", coinbaseV2, coinbasePaymentMethods)

	var params Params
	params.urlVals = url.Values{}
	params.PreparePagination(pag)

	pathParams := common.EncodeURLValues("", params.urlVals)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, pathParams, nil, Version2, &resp, nil)
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
		path, "", nil, Version2, &resp, nil)
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

/*

// GetHolds returns information on the holds of an account
func (c *CoinbasePro) GetHolds(ctx context.Context, accountID, direction, step string, limit int64) ([]AccountHolds, ReturnedPaginationHeaders, error) {
	path := fmt.Sprintf("%s/%s/%s", coinbaseAccounts, accountID, coinbaseproHolds)

	var params Params
	params.urlVals = url.Values{}
	// Warning: This endpoint doesn't seem to properly support pagination, the headers
	// indicating the cursor position are never actually present. Still, it's handled
	// as if it works, in case it gets fixed.
	params.PrepareDSL(direction, step, limit)

	path = common.EncodeURLValues(path, params.urlVals)

	var resp []AccountHolds
	retH := http.Header{}

	err := c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, "", nil, Version3, &resp, &retH)

	rph := ReturnedPaginationHeaders{before: retH.Get("CB-BEFORE"), after: retH.Get("CB-AFTER")}

	return resp, rph, err
}

// GetAccountLedger returns a list of ledger activity
func (c *CoinbasePro) GetAccountLedger(ctx context.Context, accountID, direction, step, pID string, startDate, endDate time.Time, limit int64) ([]AccountLedgerResponse, ReturnedPaginationHeaders, error) {
	var params Params
	params.urlVals = url.Values{}
	var rph ReturnedPaginationHeaders

	err := params.PrepareDateString(startDate, endDate, startDateString, endDateString)
	if err != nil {
		return nil, rph, err
	}

	path := fmt.Sprintf("%s/%s/%s", coinbaseAccounts, accountID, coinbaseproLedger)

	params.PrepareDSL(direction, step, limit)

	if pID != "" {
		params.urlVals.Set("profile_id", pID)
	}

	path = common.EncodeURLValues(path, params.urlVals)

	var resp []AccountLedgerResponse
	retH := http.Header{}

	err = c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, "", nil, Version3, &resp, &retH)

	rph = ReturnedPaginationHeaders{before: retH.Get("CB-BEFORE"), after: retH.Get("CB-AFTER")}

	return resp, rph, err
}

// GetAccountTransfers returns a history of withdrawal and or deposit
// transactions for a single account
func (c *CoinbasePro) GetAccountTransfers(ctx context.Context, accountID, direction, step, transferType string, limit int64) ([]TransferResponse, ReturnedPaginationHeaders, error) {
	path := fmt.Sprintf("%s/%s/%s", coinbaseAccounts, accountID, coinbaseproTransfers)

	var params Params
	params.urlVals = url.Values{}

	params.PrepareDSL(direction, step, limit)
	params.urlVals.Set("type", transferType)

	path = common.EncodeURLValues(path, params.urlVals)

	var resp []TransferResponse
	retH := http.Header{}

	err := c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, "", nil, Version3, &resp, &retH)

	rph := ReturnedPaginationHeaders{before: retH.Get("CB-BEFORE"), after: retH.Get("CB-AFTER")}

	return resp, rph, err
}

// GetAddressBook returns all addresses stored in the address book
func (c *CoinbasePro) GetAddressBook(ctx context.Context) ([]GetAddressResponse, error) {
	var resp []GetAddressResponse

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproAddressBook, "", nil, Version3, &resp, nil)
}

// AddAddresses adds new addresses to the address book
func (c *CoinbasePro) AddAddresses(ctx context.Context, req []AddAddressRequest) ([]AddAddressResponse, error) {
	params := make(map[string]interface{})
	params["addresses"] = req
	// The documentation also prompts us to add in an arbitrary amount of strings
	// into the parameters, without specifying what they're for. Adding some seemed
	// to do nothing

	var resp []AddAddressResponse

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproAddressBook, "", params, Version3, &resp, nil)
}

// DeleteAddress deletes an address from the address book
func (c *CoinbasePro) DeleteAddress(ctx context.Context, addressID string) error {
	path := fmt.Sprintf("%s/%s", coinbaseproAddressBook, addressID)

	return c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, "", nil, Version3, nil, nil)
}

// GetCoinbaseWallets returns all of the user's available Coinbase wallets
func (c *CoinbasePro) GetCoinbaseWallets(ctx context.Context) ([]CoinbaseAccounts, error) {
	var resp []CoinbaseAccounts

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproCoinbaseAccounts, "", nil, Version3, &resp, nil)
}

// GetAllCurrencies returns a list of currencies known by the exchange
// Warning: Currencies won't necessarily be available for trading
func (c *CoinbasePro) GetAllCurrencies(ctx context.Context) ([]Currency, error) {
	var currencies []Currency

	return currencies,
		c.SendHTTPRequest(ctx, exchange.RestSpot, coinbaseproCurrencies, &currencies)
}

// GetCurrencyByID returns info on a single currency given its ID in ISO 4217, or
// in a custom code for currencies which lack an ISO 4217 code
func (c *CoinbasePro) GetCurrencyByID(ctx context.Context, currencyID string) (*Currency, error) {
	path := fmt.Sprintf("%s/%s", coinbaseproCurrencies, currencyID)

	resp := Currency{}

	return &resp, c.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// DepositViaCoinbase deposits funds from a Coinbase account
func (c *CoinbasePro) DepositViaCoinbase(ctx context.Context, profileID, currency, coinbaseAccountID string, amount float64) (DepositWithdrawalInfo, error) {
	params := map[string]interface{}{"profile_id": profileID,
		"amount":              strconv.FormatFloat(amount, 'f', -1, 64),
		"coinbase_account_id": coinbaseAccountID, "currency": currency}

	resp := DepositWithdrawalInfo{}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproDepositCoinbase, "", params, Version3, &resp, nil)
}

// DepositViaPaymentMethod deposits funds from a payment method. SEPA is not allowed
func (c *CoinbasePro) DepositViaPaymentMethod(ctx context.Context, profileID, paymentID, currency string, amount float64) (DepositWithdrawalInfo, error) {
	params := map[string]interface{}{"profile_id": profileID,
		"amount":            strconv.FormatFloat(amount, 'f', -1, 64),
		"payment_method_id": paymentID, "currency": currency}

	resp := DepositWithdrawalInfo{}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproPaymentMethodDeposit, "", params, Version3, &resp, nil)
}

// GetPayMethods returns a full list of payment methods
func (c *CoinbasePro) GetPayMethods(ctx context.Context) ([]PaymentMethod, error) {
	var resp []PaymentMethod

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproPaymentMethod, "", nil, Version3, &resp, nil)
}

// GetAllTransfers returns all in-progress and completed transfers in and out of any
// of the user's accounts
func (c *CoinbasePro) GetAllTransfers(ctx context.Context, profileID, direction, step, transferType string, limit int64) ([]TransferResponse, ReturnedPaginationHeaders, error) {
	var params Params
	params.urlVals = url.Values{}
	params.urlVals.Set("profile_id", profileID)
	params.PrepareDSL(direction, step, limit)
	params.urlVals.Set("type", transferType)
	path := common.EncodeURLValues(coinbaseproTransfers, params.urlVals)

	resp := []TransferResponse{}
	retH := http.Header{}

	err := c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, "", nil, Version3, &resp, &retH)

	rph := ReturnedPaginationHeaders{before: retH.Get("CB-BEFORE"), after: retH.Get("CB-AFTER")}

	return resp, rph, err
}

// GetTransferByID returns information on a single transfer when provided with its ID
func (c *CoinbasePro) GetTransferByID(ctx context.Context, transferID string) (*TransferResponse, error) {
	path := fmt.Sprintf("%s/%s", coinbaseproTransfers, transferID)
	resp := TransferResponse{}

	return &resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, "", nil, Version3, &resp, nil)
}

// SendTravelInfoForTransfer sends travel rule information for a transfer
func (c *CoinbasePro) SendTravelInfoForTransfer(ctx context.Context, transferID, originName, originCountry string) (string, error) {
	path := fmt.Sprintf("%s/%s/%s", coinbaseproTransfers, transferID,
		coinbaseproTravelRules)
	params := map[string]interface{}{"transfer_id": transferID,
		"originator_name": originName, "originator_country": originCountry}

	var resp string

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, "", params, Version3, &resp, nil)
}

// WithdrawViaCoinbase withdraws funds to a coinbase account.
func (c *CoinbasePro) WithdrawViaCoinbase(ctx context.Context, profileID, accountID, currency string, amount float64) (DepositWithdrawalInfo, error) {
	req := map[string]interface{}{"profile_id": profileID,
		"amount":              strconv.FormatFloat(amount, 'f', -1, 64),
		"coinbase_account_id": accountID, "currency": currency}

	resp := DepositWithdrawalInfo{}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproWithdrawalCoinbase, "", req, Version3, &resp, nil)
}

// WithdrawCrypto withdraws funds to a crypto address
func (c *CoinbasePro) WithdrawCrypto(ctx context.Context, profileID, currency, cryptoAddress, destinationTag, twoFactorCode, network string, amount float64, noDestinationTag, addNetworkFee bool, nonce int32) (DepositWithdrawalInfo, error) {
	req := map[string]interface{}{"profile_id": profileID,
		"amount":   strconv.FormatFloat(amount, 'f', -1, 64),
		"currency": currency, "crypto_address": cryptoAddress,
		"destination_tag": destinationTag, "no_destination_tag": noDestinationTag,
		"two_factor_code": twoFactorCode, "nonce": nonce, "network": network,
		"add_network_fee": addNetworkFee}

	resp := DepositWithdrawalInfo{}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproWithdrawalCrypto, "", req, Version3, &resp, nil)
}

// GetWithdrawalFeeEstimate has Coinbase estimate the fee for withdrawing in a certain
// network to a certain address
func (c *CoinbasePro) GetWithdrawalFeeEstimate(ctx context.Context, currency, cryptoAddress, network string) (WithdrawalFeeEstimate, error) {
	resp := WithdrawalFeeEstimate{}
	if currency == "" {
		return resp, errors.New("currency cannot be empty")
	}
	if cryptoAddress == "" {
		return resp, errors.New("cryptoAddress cannot be empty")
	}
	var params Params
	params.urlVals = url.Values{}
	params.urlVals.Set("currency", currency)
	params.urlVals.Set("crypto_address", cryptoAddress)
	params.urlVals.Set("network", network)
	path := common.EncodeURLValues(coinbaseproFeeEstimate, params.urlVals)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, "", nil, Version3, &resp, nil)
}

// WithdrawViaPaymentMethod withdraws funds to a payment method
func (c *CoinbasePro) WithdrawViaPaymentMethod(ctx context.Context, profileID, paymentID, currency string, amount float64) (DepositWithdrawalInfo, error) {
	req := map[string]interface{}{"profile_id": profileID,
		"amount":            strconv.FormatFloat(amount, 'f', -1, 64),
		"payment_method_id": paymentID, "currency": currency}

	resp := DepositWithdrawalInfo{}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproWithdrawalPaymentMethod, "", req, Version3, &resp, nil)
}

// GetFees returns your current maker & taker fee rates, as well as your 30-day
// trailing volume. Quoted rates are subject to change.
func (c *CoinbasePro) GetFees(ctx context.Context) (FeeResponse, error) {
	resp := FeeResponse{}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproFees, "", nil, Version3, &resp, nil)
}

// GetSignedPrices returns some cryptographically signed prices ready to be
// posted on-chain using Compound's Open Oracle smart contract
func (c *CoinbasePro) GetSignedPrices(ctx context.Context) (SignedPrices, error) {
	resp := SignedPrices{}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproOracle, "", nil, Version3, &resp, nil)
}

// GetOrderbook returns orderbook by currency pair and level
func (c *CoinbasePro) GetOrderbook(ctx context.Context, symbol string, level int32) (*OrderbookFinalResponse, error) {
	if symbol == "" {
		return nil, errors.New("symbol cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/%s", coinbaseProducts, symbol, coinbaseproOrderbook)
	if level > 0 {
		var params Params
		params.urlVals = url.Values{}
		params.urlVals.Set("level", strconv.Itoa(int(level)))

		path = common.EncodeURLValues(path, params.urlVals)
	}

	data := OrderbookIntermediaryResponse{}
	err := c.SendHTTPRequest(ctx, exchange.RestSpot, path, &data)
	if err != nil {
		return nil, err
	}

	obF := OrderbookFinalResponse{
		Sequence:    data.Sequence,
		AuctionMode: data.AuctionMode,
		Auction:     data.Auction,
		Time:        data.Time,
	}

	obF.Bids, err = OrderbookHelper(data.Bids, level)
	if err != nil {
		return nil, err
	}
	obF.Asks, err = OrderbookHelper(data.Asks, level)
	if err != nil {
		return nil, err
	}

	return &obF, nil
}

// GetStats returns 30 day and 24 hour stats for the product. Volume is in base currency
// units. open, high, low are in quote currency units.
func (c *CoinbasePro) GetStats(ctx context.Context, currencyPair string) (Stats, error) {
	stats := Stats{}
	if currencyPair == "" {
		return stats, errors.New("currency pair cannot be empty")
	}

	path := fmt.Sprintf(
		"%s/%s/%s", coinbaseProducts, currencyPair, coinbaseproStats)

	return stats, c.SendHTTPRequest(ctx, exchange.RestSpot, path, &stats)
}

// GetTrades lists information on the latest trades for a product
func (c *CoinbasePro) GetTrades(ctx context.Context, currencyPair, direction, step string, limit int64) ([]Trade, error) {
	if currencyPair == "" {
		return nil, errors.New("currency pair cannot be empty")
	}

	path := fmt.Sprintf(
		"%s/%s/%s", coinbaseProducts, currencyPair, coinbaseproTrades)

	var params Params
	params.urlVals = url.Values{}
	params.PrepareDSL(direction, step, limit)

	path = common.EncodeURLValues(path, params.urlVals)

	var trades []Trade

	return trades, c.SendHTTPRequest(ctx, exchange.RestSpot, path, &trades)
}

// GetAllProfiles returns information on all of the current user's profiles
func (c *CoinbasePro) GetAllProfiles(ctx context.Context, active *bool) ([]Profile, error) {
	var params Params
	params.urlVals = url.Values{}

	if active != nil {
		params.urlVals.Set("active", strconv.FormatBool(*active))
	}

	var resp []Profile

	path := common.EncodeURLValues(coinbaseproProfiles, params.urlVals)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, "", nil, Version3, &resp, nil)
}

// CreateAProfile creates a new profile, failing if no name is provided,
// or if the user already has the max number of profiles
func (c *CoinbasePro) CreateAProfile(ctx context.Context, name string) (Profile, error) {
	var resp Profile
	if name == "" {
		return resp, errors.New("name cannot be empty")
	}

	req := map[string]interface{}{"name": name}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproProfiles, "", req, Version3, &resp, nil)
}

// TransferBetweenProfiles transfers an amount of currency from one profile to another
func (c *CoinbasePro) TransferBetweenProfiles(ctx context.Context, from, to, currency string, amount float64) (string, error) {
	var resp string
	if from == "" || to == "" || currency == "" {
		return resp, errors.New("from, to, and currency must all not be empty")
	}

	req := map[string]interface{}{"from": from, "to": to, "currency": currency,
		"amount": strconv.FormatFloat(amount, 'f', -1, 64)}

	path := fmt.Sprintf("%s/%s", coinbaseproProfiles, coinbaseproTransfer)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, "", req, Version3, &resp, nil)
}

// GetProfileByID returns information on a single profile, provided its ID
func (c *CoinbasePro) GetProfileByID(ctx context.Context, profileID string, active *bool) (Profile, error) {
	var params Params
	params.urlVals = url.Values{}
	if active != nil {
		params.urlVals.Set("active", strconv.FormatBool(*active))
	}

	var resp Profile
	path := fmt.Sprintf("%s/%s", coinbaseproProfiles, profileID)
	path = common.EncodeURLValues(path, params.urlVals)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, "", nil, Version3, &resp, nil)
}

// RenameProfile renames a profile, provided its ID
func (c *CoinbasePro) RenameProfile(ctx context.Context, profileID, newName string) (Profile, error) {
	var resp Profile
	if newName == "" {
		return resp, errors.New("new name cannot be empty")
	}

	req := map[string]interface{}{"profile_id": profileID, "name": newName}

	path := fmt.Sprintf("%s/%s", coinbaseproProfiles, profileID)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPut, path, "", req, Version3, &resp, nil)
}

// DeleteProfile deletes a profile and transfers its funds to a specified
// profile. Fails if there are any open orders on the profile facing deletion
func (c *CoinbasePro) DeleteProfile(ctx context.Context, profileID, transferTo string) (string, error) {
	var resp string
	if profileID == "" || transferTo == "" {
		return resp, errors.New("neither profileID nor transferTo can be empty")
	}

	req := map[string]interface{}{"profile_id": profileID, "to": transferTo}

	path := fmt.Sprintf("%s/%s/%s", coinbaseproProfiles, profileID, coinbaseproDeactivate)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, "", req, Version3, &resp, nil)
}

// GetAllReports returns a list of all user-generated reports
func (c *CoinbasePro) GetAllReports(ctx context.Context, profileID string, reportType string, after time.Time, limit int64, ignoreExpired bool) ([]Report, error) {
	var resp []Report

	var params Params
	params.urlVals = url.Values{}

	params.urlVals.Set("profile_id", profileID)
	params.urlVals.Set("after", after.Format(time.RFC3339))
	if limit != 0 {
		params.urlVals.Set("limit", strconv.FormatInt(limit, 10))
	}

	params.urlVals.Set("type", reportType)
	params.urlVals.Set("ignore_expired", strconv.FormatBool(ignoreExpired))

	path := common.EncodeURLValues(coinbaseproReports, params.urlVals)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, "", nil, Version3, &resp, nil)
}

// CreateReport creates a new report
func (c *CoinbasePro) CreateReport(ctx context.Context, reportType, year, format, email, profileID, productID, accountID string, balanceDate, startDate, endDate time.Time) (CreateReportResponse, error) {
	var resp CreateReportResponse

	if reportType == "" {
		return resp, errors.New("report type cannot be empty")
	}
	if reportType == "1099k-transaction-history" && year == "" {
		return resp, errors.New("year cannot be empty for 1099k-transaction-history reports")
	}
	if reportType != "balance" {
		err := common.StartEndTimeCheck(startDate, endDate)
		if err != nil {
			return resp, err
		}
	}

	req := map[string]interface{}{"type": reportType, "year": year, "format": format,
		"email": email, "profile_id": profileID}

	if reportType == "account" {
		req["account"] = ReportAccountStruct{StartDate: startDate.Format(time.RFC3339),
			EndDate: endDate.Format(time.RFC3339), AccountID: accountID}
	}
	if reportType == "balance" {
		req["balance"] = ReportBalanceStruct{DateTime: balanceDate.Format(time.RFC3339)}
	}
	if reportType == "fills" || reportType == "otc-fills" || reportType == "rfq-fills" ||
		reportType == "tax-invoice" {
		req[reportType] = ReportFillsTaxStruct{StartDate: startDate.Format(time.RFC3339),
			EndDate: endDate.Format(time.RFC3339), ProductID: productID}
	}
	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproReports, "", req, Version3, &resp, nil)
}

// GetReportByID returns a single report, provided its ID
func (c *CoinbasePro) GetReportByID(ctx context.Context, reportID string) (Report, error) {
	var resp Report
	if reportID == "" {
		return resp, errors.New("report id cannot be empty")
	}

	path := fmt.Sprintf("%s/%s", coinbaseproReports, reportID)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, "", nil, Version3, &resp, nil)
}

// GetTravelRules returns a list of all travel rule information
func (c *CoinbasePro) GetTravelRules(ctx context.Context, direction, step, address string, limit int64) ([]TravelRule, error) {
	var resp []TravelRule
	var params Params
	params.urlVals = url.Values{}

	params.PrepareDSL(direction, step, limit)
	params.urlVals.Set("address", address)

	path := common.EncodeURLValues(coinbaseproTravelRules, params.urlVals)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, "", nil, Version3, &resp, nil)
}

// CreateTravelRule creates a travel rule entry
func (c *CoinbasePro) CreateTravelRule(ctx context.Context, address, originName, originCountry string) (TravelRule, error) {
	var resp TravelRule

	req := map[string]interface{}{"address": address, "originator_name": originName,
		"originator_country": originCountry}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproTravelRules, "", req, Version3, &resp, nil)
}

// DeleteTravelRule deletes a travel rule entry
func (c *CoinbasePro) DeleteTravelRule(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("id cannot be empty")
	}

	path := fmt.Sprintf("%s/%s", coinbaseproTravelRules, id)

	return c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, "", nil, Version3, nil, nil)
}

// GetExchangeLimits returns information on payment method transfer limits,
// as well as buy/sell limits per currency
func (c *CoinbasePro) GetExchangeLimits(ctx context.Context, userID string) (ExchangeLimits, error) {
	var resp ExchangeLimits

	path := fmt.Sprintf("%s/%s/%s", coinbaseUsers, userID, coinbaseproExchangeLimits)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, "", nil, Version3, &resp, nil)
}

// UpdateSettlementPreference updates whether one wants their funds to
// automatically convert to USD, USDC, or to remain in the currency received
func (c *CoinbasePro) UpdateSettlementPreference(ctx context.Context, userID, preference string) (string, error) {
	if userID == "" || preference == "" {
		return "", errors.New("neither userID nor preference can be empty")
	}

	req := map[string]interface{}{"settlement_preference": preference}

	path := fmt.Sprintf("%s/%s/%s", coinbaseUsers, userID, coinbaseproSettlementPreferences)

	var resp string

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPut, path, "", req, Version3, &resp, nil)
}

// GetAllWrappedAssets returns information on all supported wrapped assets
func (c *CoinbasePro) GetAllWrappedAssets(ctx context.Context) (AllWrappedAssetResponse, error) {
	var resp AllWrappedAssetResponse

	return resp,
		c.SendHTTPRequest(ctx, exchange.RestSpot, coinbaseproWrappedAssets, &resp)
}

// GetAllStakeWraps returns details of all stake-wraps under the profile associated
// with the API key
func (c *CoinbasePro) GetAllStakeWraps(ctx context.Context, direction, from, to, status string, timestamp time.Time, limit int64) ([]StakeWrap, error) {
	var resp []StakeWrap

	var params Params
	params.urlVals = url.Values{}

	if !timestamp.IsZero() && !timestamp.Equal(time.Unix(0, 0)) {
		params.PrepareDSL(direction, timestamp.Format(time.RFC3339), limit)
	} else {
		params.urlVals.Set("limit", strconv.FormatInt(limit, 10))
	}

	params.urlVals.Set("from", from)
	params.urlVals.Set("to", to)
	params.urlVals.Set("status", status)

	path := fmt.Sprintf("%s/%s", coinbaseproWrappedAssets, coinbaseproStakeWraps)

	path = common.EncodeURLValues(path, params.urlVals)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, "", nil, Version3, &resp, nil)
}

// CreateStakeWrap stakes and wraps from one currency to another, under the profile
// associated with the API key
func (c *CoinbasePro) CreateStakeWrap(ctx context.Context, from, to string, amount float64) (StakeWrap, error) {
	if from == "" || to == "" || amount == 0 {
		return StakeWrap{}, errors.New("none of from, to, or amount can be empty or zero")
	}
	var resp StakeWrap

	req := map[string]interface{}{"from": from, "to": to,
		"amount": strconv.FormatFloat(amount, 'f', -1, 64)}

	path := fmt.Sprintf("%s/%s", coinbaseproWrappedAssets, coinbaseproStakeWraps)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, "", req, Version3, &resp, nil)
}

// GetStakeWrapByID returns details of a single stake-wrap
func (c *CoinbasePro) GetStakeWrapByID(ctx context.Context, stakeWrapID string) (StakeWrap, error) {
	var resp StakeWrap

	if stakeWrapID == "" {
		return resp, errors.New("stake wrap id cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/%s", coinbaseproWrappedAssets, coinbaseproStakeWraps, stakeWrapID)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, "", nil, Version3, &resp, nil)
}

// GetWrappedAssetByID returns details of a single wrapped asset
func (c *CoinbasePro) GetWrappedAssetByID(ctx context.Context, wrappedAssetID string) (WrappedAssetResponse, error) {
	var resp WrappedAssetResponse

	if wrappedAssetID == "" {
		return resp, errors.New("wrapped asset id cannot be empty")
	}

	path := fmt.Sprintf("%s/%s", coinbaseproWrappedAssets, wrappedAssetID)

	return resp, c.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GetWrappedAssetConversionRate returns the conversion rate for a wrapped asset
func (c *CoinbasePro) GetWrappedAssetConversionRate(ctx context.Context, wrappedAssetID string) (WrappedAssetConversionRate, error) {
	var resp WrappedAssetConversionRate

	if wrappedAssetID == "" {
		return resp, errors.New("wrapped asset id cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/%s", coinbaseproWrappedAssets, wrappedAssetID, coinbaseproConversionRate)

	return resp, c.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// // MarginTransfer sends funds between a standard/default profile and a margin
// // profile.
// // A deposit will transfer funds from the default profile into the margin
// // profile. A withdraw will transfer funds from the margin profile to the
// // default profile. Withdraws will fail if they would set your margin ratio
// // below the initial margin ratio requirement.
// //
// // amount - the amount to transfer between the default and margin profile
// // transferType - either "deposit" or "withdraw"
// // profileID - The id of the margin profile to deposit or withdraw from
// // currency - currency to transfer, currently on "BTC" or "USD"
// func (c *CoinbasePro) MarginTransfer(ctx context.Context, amount float64, transferType, profileID, currency string) (MarginTransfer, error) {
// 	resp := MarginTransfer{}
// 	req := make(map[string]interface{})
// 	req["type"] = transferType
// 	req["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
// 	req["currency"] = currency
// 	req["margin_profile_id"] = profileID

// 	return resp,
// 		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproMarginTransfer, req, &resp)
// }

// // GetPosition returns an overview of account profile.
// func (c *CoinbasePro) GetPosition(ctx context.Context) (AccountOverview, error) {
// 	resp := AccountOverview{}

// 	return resp,
// 		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproPosition, nil, &resp)
// }

// // ClosePosition closes a position and allowing you to repay position as well
// // repayOnly -  allows the position to be repaid
// func (c *CoinbasePro) ClosePosition(ctx context.Context, repayOnly bool) (AccountOverview, error) {
// 	resp := AccountOverview{}
// 	req := make(map[string]interface{})
// 	req["repay_only"] = repayOnly

// 	return resp,
// 		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproPositionClose, req, &resp)
// }

// // GetReportStatus once a report request has been accepted for processing, the
// // status is available by polling the report resource endpoint.
// func (c *CoinbasePro) GetReportStatus(ctx context.Context, reportID string) (Report, error) {
// 	resp := Report{}
// 	path := fmt.Sprintf("%s/%s", coinbaseproReports, reportID)

// 	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp)
// }

// // GetTrailingVolume this request will return your 30-day trailing volume for
// // all products.
// func (c *CoinbasePro) GetTrailingVolume(ctx context.Context) ([]Volume, error) {
// 	var resp []Volume

// 	return resp,
// 		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproTrailingVolume, nil, &resp)
// }

*/

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
func (c *CoinbasePro) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, method, path, queryParams string, bodyParams map[string]interface{}, version Version, result interface{}, returnHead *http.Header) (err error) {
	creds, err := c.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpoint, err := c.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	// Version 2 wants query params in the path during signing
	if version == Version2 {
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
		if version == Version3 {
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
	if version == Version3 {
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

// // GetFee returns an estimate of fee based on type of transaction
// func (c *CoinbasePro) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
// 	var fee float64
// 	switch feeBuilder.FeeType {
// 	case exchange.CryptocurrencyTradeFee:
// 		fees, err := c.GetFees(ctx)
// 		if err != nil {
// 			fee = fees.TakerFeeRate
// 		} else {
// 			fee = 0.006
// 		}
// 	case exchange.InternationalBankWithdrawalFee:
// 		fee = getInternationalBankWithdrawalFee(feeBuilder.FiatCurrency)
// 	case exchange.InternationalBankDepositFee:
// 		fee = getInternationalBankDepositFee(feeBuilder.FiatCurrency)
// 	case exchange.OfflineTradeFee:
// 		fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
// 	}

// 	if fee < 0 {
// 		fee = 0
// 	}

// 	return fee, nil
// }

// // getOfflineTradeFee calculates the worst case-scenario trading fee
// func getOfflineTradeFee(price, amount float64) float64 {
// 	return 0.0025 * price * amount
// }

// func (c *CoinbasePro) calculateTradingFee(trailingVolume []Volume, base, quote currency.Code, delimiter string, purchasePrice, amount float64, isMaker bool) float64 {
// 	var fee float64
// 	for _, i := range trailingVolume {
// 		if strings.EqualFold(i.ProductID, base.String()+delimiter+quote.String()) {
// 			switch {
// 			case isMaker:
// 				fee = 0
// 			case i.Volume <= 10000000:
// 				fee = 0.003
// 			case i.Volume > 10000000 && i.Volume <= 100000000:
// 				fee = 0.002
// 			case i.Volume > 100000000:
// 				fee = 0.001
// 			}
// 			break
// 		}
// 	}
// 	return fee * amount * purchasePrice
// }

// func getInternationalBankWithdrawalFee(c currency.Code) float64 {
// 	var fee float64

// 	if c.Equal(currency.USD) {
// 		fee = 25
// 	} else if c.Equal(currency.EUR) {
// 		fee = 0.15
// 	}

// 	return fee
// }

// func getInternationalBankDepositFee(c currency.Code) float64 {
// 	var fee float64

// 	if c.Equal(currency.USD) {
// 		fee = 10
// 	} else if c.Equal(currency.EUR) {
// 		fee = 0.15
// 	}

// 	return fee
// }

// PrepareDateString encodes a set of parameters indicating start & end dates
func (p *Params) PrepareDateString(startDate, endDate time.Time, labelStart, labelEnd string) error {
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

func (p *Params) PreparePagination(pag PaginationInp) {
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

// OrderbookHelper handles the transfer of bids and asks of unclear levels, to a
// generalised format
func OrderbookHelper(iOD InterOrderDetail, level int32) ([]GenOrderDetail, error) {
	gOD := make([]GenOrderDetail, len(iOD))

	for i := range iOD {
		priceConv, ok := iOD[i][0].(string)
		if !ok {
			return nil, errors.New("unable to type assert price")
		}
		price, err := strconv.ParseFloat(priceConv, 64)
		if err != nil {
			return nil, err
		}
		gOD[i].Price = price

		amountConv, ok := iOD[i][1].(string)
		if !ok {
			return nil, errors.New("unable to type assert amount")
		}
		amount, err := strconv.ParseFloat(amountConv, 64)
		if err != nil {
			return nil, err
		}
		gOD[i].Amount = amount

		if level == 3 {
			orderID, ok := iOD[i][2].(string)
			if !ok {
				return nil, errors.New("unable to type assert order ID")
			}
			gOD[i].OrderID = orderID
		} else {
			numOrders, ok := iOD[i][2].(float64)
			if !ok {
				return nil, errors.New("unable to type assert number of orders")
			}
			gOD[i].NumOrders = numOrders
		}

	}
	return gOD, nil

}

// PrepareDSL adds the direction, step, and limit queries for pagination
func (p *Params) PrepareDSL(direction, step string, limit int64) {
	p.urlVals.Set(direction, step)
	if limit >= 0 {
		p.urlVals.Set("limit", strconv.FormatInt(limit, 10))
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
	*t = UnixTimestamp(time.Unix(timestamp, 0))
	return nil
}

func (t UnixTimestamp) String() string {
	return time.Time(t).String()
}

func (t *ExchTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == " " || s == "null" {
		return nil
	}
	tt, err := time.Parse("2006-01-02 15:04:05.999999-07", s)
	if err != nil {
		return err
	}
	*t = ExchTime(tt)
	return nil
}

func (t ExchTime) String() string {
	return time.Time(t).String()
}

func (pm *PriceMap) UnmarshalJSON(data []byte) error {
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*pm = make(PriceMap)
	for k, v := range m {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return err
		}
		(*pm)[k] = f
	}
	return nil
}
