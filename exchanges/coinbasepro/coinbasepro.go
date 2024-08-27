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

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	coinbaseAPIURL             = "https://api.coinbase.com"
	coinbaseV1APIURL           = "https://api.exchange.coinbase.com/"
	coinbaseproSandboxAPIURL   = "https://api-public.sandbox.exchange.coinbase.com/"
	tradeBaseURL               = "https://www.coinbase.com/advanced-trade/spot/"
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
	coinbasePreview            = "preview"
	coinbasePortfolios         = "portfolios"
	coinbaseMoveFunds          = "move_funds"
	coinbaseCFM                = "cfm"
	coinbaseBalanceSummary     = "balance_summary"
	coinbasePositions          = "positions"
	coinbaseSweeps             = "sweeps"
	coinbaseSchedule           = "schedule"
	coinbaseIntx               = "intx"
	coinbaseAllocate           = "allocate"
	coinbasePortfolio          = "portfolio"
	coinbaseTransactionSummary = "transaction_summary"
	coinbaseConvert            = "convert"
	coinbaseQuote              = "quote"
	coinbaseTrade              = "trade"
	coinbasePaymentMethods     = "payment_methods"
	coinbaseV2                 = "/v2/"
	coinbaseNotifications      = "notifications"
	coinbaseUser               = "user"
	coinbaseUsers              = "users"
	coinbaseAuth               = "auth"
	coinbaseAddresses          = "addresses"
	coinbaseTransactions       = "transactions"
	coinbaseDeposits           = "deposits"
	coinbaseCommit             = "commit"
	coinbaseWithdrawals        = "withdrawals"
	coinbaseCurrencies         = "currencies"
	coinbaseCrypto             = "crypto"
	coinbaseExchangeRates      = "exchange-rates"
	coinbasePrices             = "prices"
	coinbaseTime               = "time"
	coinbaseVolumeSummary      = "volume-summary"
	coinbaseBook               = "book"
	coinbaseStats              = "stats"
	coinbaseTrades             = "trades"
	coinbaseWrappedAssets      = "wrapped-assets"
	coinbaseConversionRate     = "conversion-rate"
	coinbaseMarket             = "market"

	pageNone        = ""
	pageBefore      = "before"
	pageAfter       = "after"
	unknownContract = "UNKNOWN_CONTRACT_EXPIRY_TYPE"
	granUnknown     = "UNKNOWN_GRANULARITY"
	granOneMin      = "ONE_MINUTE"
	granFiveMin     = "FIVE_MINUTE"
	granFifteenMin  = "FIFTEEN_MINUTE"
	granThirtyMin   = "THIRTY_MINUTE"
	granOneHour     = "ONE_HOUR"
	granTwoHour     = "TWO_HOUR"
	granSixHour     = "SIX_HOUR"
	granOneDay      = "ONE_DAY"
	startDateString = "start_date"
	endDateString   = "end_date"

	errIntervalNotSupported = "interval not supported"
	warnSequenceIssue       = "Out of order sequence number. Received %v, expected %v"
	warnAuth                = "%v authenticated request failed, attempting unauthenticated"
)

// Constants defining whether a transfer is a deposit or withdrawal, used to simplify
// interactions with a few endpoints
const (
	FiatDeposit    FiatTransferType = false
	FiatWithdrawal FiatTransferType = true
)

// While the exchange's fee pages say the worst taker/maker fees are lower than the ones listed
// here, the data returned by the GetTransactionsSummary endpoint are consistent with these worst
// case scenarios. The best case scenarios are untested, and assumed to be in line with the fee pages
const (
	WorstCaseTakerFee           = 0.012
	WorstCaseMakerFee           = 0.006
	BestCaseTakerFee            = 0.0005
	BestCaseMakerFee            = 0
	StablePairMakerFee          = 0
	WorstCaseStablePairTakerFee = 0.000045
	BestCaseStablePairTakerFee  = 0.00001
)

var (
	errAccountIDEmpty            = errors.New("account id cannot be empty")
	errClientOrderIDEmpty        = errors.New("client order id cannot be empty")
	errProductIDEmpty            = errors.New("product id cannot be empty")
	errOrderIDEmpty              = errors.New("order ids cannot be empty")
	errOpenPairWithOtherTypes    = errors.New("cannot pair open orders with other order types")
	errSizeAndPriceZero          = errors.New("size and price cannot both be 0")
	errCurrencyEmpty             = errors.New("currency cannot be empty")
	errCurrWalletConflict        = errors.New("exactly one of walletID and currency must be specified")
	errWalletIDEmpty             = errors.New("wallet id cannot be empty")
	errAddressIDEmpty            = errors.New("address id cannot be empty")
	errTransactionTypeEmpty      = errors.New("transaction type cannot be empty")
	errToEmpty                   = errors.New("to cannot be empty")
	errAmountEmpty               = errors.New("amount cannot be empty")
	errTransactionIDEmpty        = errors.New("transaction id cannot be empty")
	errPaymentMethodEmpty        = errors.New("payment method cannot be empty")
	errDepositIDEmpty            = errors.New("deposit id cannot be empty")
	errInvalidPriceType          = errors.New("price type must be spot, buy, or sell")
	errInvalidOrderType          = errors.New("order type must be market, limit, or stop")
	errNoMatchingWallets         = errors.New("no matching wallets returned")
	errOrderModFailNoRet         = errors.New("order modification failed but no error returned")
	errNameEmpty                 = errors.New("name cannot be empty")
	errPortfolioIDEmpty          = errors.New("portfolio id cannot be empty")
	errFeeTypeNotSupported       = errors.New("fee type not supported")
	errCantDecodePrivKey         = errors.New("cannot decode private key")
	errNoWalletForCurrency       = errors.New("no wallet found for currency, address creation impossible")
	errChannelNameUnknown        = errors.New("unknown channel name")
	errNoWalletsReturned         = errors.New("no wallets returned")
	errPayMethodNotFound         = errors.New("payment method not found")
	errUnknownL2DataType         = errors.New("unknown l2update data type")
	errUnknownSide               = errors.New("unknown side")
	errInvalidGranularity        = errors.New("invalid granularity")
	errOrderFailedToCancel       = errors.New("failed to cancel order")
	errUnrecognisedStatusType    = errors.New("unrecognised status type")
	errPairEmpty                 = errors.New("pair cannot be empty")
	errStringConvert             = errors.New("unable to convert into string value")
	errFloatConvert              = errors.New("unable to convert into float64 value")
	errWrappedAssetEmpty         = errors.New("wrapped asset cannot be empty")
	errExpectedOneTickerReturned = errors.New("expected one ticker to be returned")
)

// GetAllAccounts returns information on all trading accounts associated with the API key
func (c *CoinbasePro) GetAllAccounts(ctx context.Context, limit uint8, cursor string) (*AllAccountsResponse, error) {
	vals := url.Values{}
	if limit != 0 {
		vals.Set("limit", strconv.FormatUint(uint64(limit), 10))
	}
	if cursor != "" {
		vals.Set("cursor", cursor)
	}
	var resp AllAccountsResponse
	return &resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		coinbaseV3+coinbaseAccounts, vals, nil, true, &resp, nil)
}

// GetAccountByID returns information for a single account
func (c *CoinbasePro) GetAccountByID(ctx context.Context, accountID string) (*Account, error) {
	if accountID == "" {
		return nil, errAccountIDEmpty
	}
	path := coinbaseV3 + coinbaseAccounts + "/" + accountID
	resp := OneAccountResponse{}
	return &resp.Account, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, nil, nil, true, &resp, nil)
}

// GetBestBidAsk returns the best bid/ask for all products. Can be filtered to certain products
// by passing through additional strings
func (c *CoinbasePro) GetBestBidAsk(ctx context.Context, products []string) ([]ProductBook, error) {
	vals := url.Values{}
	for x := range products {
		vals.Add("product_ids", products[x])
	}
	var resp BestBidAsk
	return resp.Pricebooks, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		coinbaseV3+coinbaseBestBidAsk, vals, nil, true, &resp, nil)
}

// GetProductBookV3 returns a list of bids/asks for a single product
func (c *CoinbasePro) GetProductBookV3(ctx context.Context, productID string, limit uint16, authenticated bool) (*ProductBook, error) {
	if productID == "" {
		return nil, errProductIDEmpty
	}
	vals := url.Values{}
	vals.Set("product_id", productID)
	if limit != 0 {
		vals.Set("limit", strconv.FormatInt(int64(limit), 10))
	}
	var resp ProductBookResponse
	if authenticated {
		return &resp.Pricebook, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseV3+coinbaseProductBook,
			vals, nil, true, &resp, nil)
	}
	path := coinbaseV3 + coinbaseMarket + "/" + coinbaseProductBook
	return &resp.Pricebook, c.SendHTTPRequest(ctx, exchange.RestSpot, path, vals, &resp)
}

// GetAllProducts returns information on all currency pairs that are available for trading
func (c *CoinbasePro) GetAllProducts(ctx context.Context, limit, offset int32, productType, contractExpiryType, expiringContractStatus string, productIDs []string, authenticated bool) (*AllProducts, error) {
	vals := url.Values{}
	if limit != 0 {
		vals.Set("limit", strconv.FormatInt(int64(limit), 10))
	}
	if offset != 0 {
		vals.Set("offset", strconv.FormatInt(int64(offset), 10))
	}
	if productType != "" {
		vals.Set("product_type", productType)
	}
	if contractExpiryType != "" {
		vals.Set("contract_expiry_type", contractExpiryType)
	}
	if expiringContractStatus != "" {
		vals.Set("expiring_contract_status", expiringContractStatus)
	}
	for x := range productIDs {
		vals.Add("product_ids", productIDs[x])
	}
	var resp AllProducts
	if authenticated {
		return &resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseV3+coinbaseProducts,
			vals, nil, true, &resp, nil)
	}
	path := coinbaseV3 + coinbaseMarket + "/" + coinbaseProducts
	return &resp, c.SendHTTPRequest(ctx, exchange.RestSpot, path, vals, &resp)
}

// GetProductByID returns information on a single specified currency pair
func (c *CoinbasePro) GetProductByID(ctx context.Context, productID string, authenticated bool) (*Product, error) {
	if productID == "" {
		return nil, errProductIDEmpty
	}
	var resp Product
	if authenticated {
		path := coinbaseV3 + coinbaseProducts + "/" + productID
		return &resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, true, &resp,
			nil)
	}
	path := coinbaseV3 + coinbaseMarket + "/" + coinbaseProducts + "/" + productID
	return &resp, c.SendHTTPRequest(ctx, exchange.RestSpot, path, nil, &resp)
}

// GetHistoricRates returns historic rates for a product. Rates are returned in
// grouped buckets based on requested granularity. Requests that return more than
// 300 data points are rejected
func (c *CoinbasePro) GetHistoricRates(ctx context.Context, productID, granularity string, startDate, endDate time.Time, authenticated bool) ([]CandleStruct, error) {
	var resp History
	if productID == "" {
		return nil, errProductIDEmpty
	}
	allowedGranularities := [8]string{granOneMin, granFiveMin, granFifteenMin,
		granThirtyMin, granOneHour, granTwoHour, granSixHour, granOneDay}
	validGran, _ := common.InArray(granularity, allowedGranularities)
	if !validGran {
		return nil, fmt.Errorf("%w %v, allowed granularities are: %+v", errInvalidGranularity,
			granularity, allowedGranularities)
	}
	vals := url.Values{}
	vals.Set("start", strconv.FormatInt(startDate.Unix(), 10))
	vals.Set("end", strconv.FormatInt(endDate.Unix(), 10))
	vals.Set("granularity", granularity)
	if authenticated {
		path := coinbaseV3 + coinbaseProducts + "/" + productID + "/" + coinbaseCandles
		return resp.Candles, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
			path, vals, nil, true, &resp, nil)
	}
	path := coinbaseV3 + coinbaseMarket + "/" + coinbaseProducts + "/" + productID + "/" + coinbaseCandles
	return resp.Candles, c.SendHTTPRequest(ctx, exchange.RestSpot, path, vals, &resp)
}

// GetTicker returns snapshot information about the last trades (ticks) and best bid/ask.
// Contrary to documentation, this does not tell you the 24h volume
func (c *CoinbasePro) GetTicker(ctx context.Context, productID string, limit uint16, startDate, endDate time.Time, authenticated bool) (*Ticker, error) {
	if productID == "" {
		return nil, errProductIDEmpty
	}
	vals := url.Values{}
	vals.Set("limit", strconv.FormatInt(int64(limit), 10))
	if !startDate.IsZero() && !startDate.Equal(time.Time{}) {
		vals.Set("start", strconv.FormatInt(startDate.Unix(), 10))
	}
	if !endDate.IsZero() && !endDate.Equal(time.Time{}) {
		vals.Set("end", strconv.FormatInt(endDate.Unix(), 10))
	}
	var resp Ticker
	if authenticated {
		path := coinbaseV3 + coinbaseProducts + "/" + productID + "/" + coinbaseTicker
		return &resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, vals, nil, true, &resp,
			nil)
	}
	path := coinbaseV3 + coinbaseMarket + "/" + coinbaseProducts + "/" + productID + "/" + coinbaseTicker
	return &resp, c.SendHTTPRequest(ctx, exchange.RestSpot, path, vals, &resp)
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
	orderConfig, err := prepareOrderConfig(orderType, side, stopDirection, amount, limitPrice, stopPrice, endTime,
		postOnly)
	if err != nil {
		return nil, err
	}
	mt := formatMarginType(marginType)
	req := map[string]interface{}{
		"client_order_id":          clientOID,
		"product_id":               productID,
		"side":                     side,
		"order_configuration":      orderConfig,
		"self_trade_prevention_id": stpID,
		"leverage":                 strconv.FormatFloat(leverage, 'f', -1, 64),
		"retail_portfolio_id":      rpID,
		"margin_type":              mt}
	var resp PlaceOrderResp
	return &resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
			coinbaseV3+coinbaseOrders, nil, req, true, &resp, nil)
}

// CancelOrders cancels orders by orderID. Can only cancel 100 orders per request
func (c *CoinbasePro) CancelOrders(ctx context.Context, orderIDs []string) ([]OrderCancelDetail, error) {
	if len(orderIDs) == 0 {
		return nil, errOrderIDEmpty
	}
	path := coinbaseV3 + coinbaseOrders + "/" + coinbaseBatchCancel
	req := map[string]interface{}{"order_ids": orderIDs}
	var resp CancelOrderResp
	return resp.Results, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil,
		req, true, &resp, nil)
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
	path := coinbaseV3 + coinbaseOrders + "/" + coinbaseEdit
	req := map[string]interface{}{
		"order_id": orderID,
		"size":     strconv.FormatFloat(size, 'f', -1, 64),
		"price":    strconv.FormatFloat(price, 'f', -1, 64)}
	var resp SuccessBool
	return resp.Success, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil,
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
	path := coinbaseV3 + coinbaseOrders + "/" + coinbaseEditPreview
	req := map[string]interface{}{
		"order_id": orderID,
		"size":     strconv.FormatFloat(size, 'f', -1, 64),
		"price":    strconv.FormatFloat(price, 'f', -1, 64)}
	var resp *EditOrderPreviewResp
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil,
		req, true, &resp, nil)
}

// GetAllOrders lists orders, filtered by their status
func (c *CoinbasePro) GetAllOrders(ctx context.Context, productID, userNativeCurrency, orderType, orderSide, cursor, productType, orderPlacementSource, contractExpiryType, retailPortfolioID string, orderStatus, assetFilters []string, limit int32, startDate, endDate time.Time) (*GetAllOrdersResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startDate, endDate, startDateString, endDateString)
	if err != nil {
		return nil, err
	}
	for x := range orderStatus {
		if orderStatus[x] == "OPEN" && len(orderStatus) > 1 {
			return nil, errOpenPairWithOtherTypes
		}
		params.Values.Add("order_status", orderStatus[x])
	}
	for x := range assetFilters {
		params.Values.Add("asset_filters", assetFilters[x])
	}
	if productID != "" {
		params.Values.Set("product_id", productID)
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(int64(limit), 10))
	}
	if cursor != "" {
		params.Values.Set("cursor", cursor)
	}
	if userNativeCurrency != "" {
		params.Values.Set("user_native_currency", userNativeCurrency)
	}
	if orderPlacementSource != "" {
		params.Values.Set("order_placement_source", orderPlacementSource)
	}
	if productType != "" {
		params.Values.Set("product_type", productType)
	}
	if orderSide != "" {
		params.Values.Set("order_side", orderSide)
	}
	if contractExpiryType != "" {
		params.Values.Set("contract_expiry_type", contractExpiryType)
	}
	if retailPortfolioID != "" {
		params.Values.Set("retail_portfolio_id", retailPortfolioID)
	}
	if orderType != "" {
		params.Values.Set("order_type", orderType)
	}
	path := coinbaseV3 + coinbaseOrders + "/" + coinbaseHistorical + "/" + coinbaseBatch
	var resp GetAllOrdersResp
	return &resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path,
		params.Values, nil, true, &resp, nil)
}

// GetFills returns information of recent fills on the specified order
func (c *CoinbasePro) GetFills(ctx context.Context, orderID, productID, cursor string, startDate, endDate time.Time, limit uint16) (*FillResponse, error) {
	var params Params
	params.Values = url.Values{}
	err := params.prepareDateString(startDate, endDate, "start_sequence_timestamp",
		"end_sequence_timestamp")
	if err != nil {
		return nil, err
	}
	if orderID != "" {
		params.Values.Set("order_id", orderID)
	}
	if productID != "" {
		params.Values.Set("product_id", productID)
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(int64(limit), 10))
	}
	if cursor != "" {
		params.Values.Set("cursor", cursor)
	}
	path := coinbaseV3 + coinbaseOrders + "/" + coinbaseHistorical + "/" + coinbaseFills
	var resp FillResponse
	return &resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path,
		params.Values, nil, true, &resp, nil)
}

// GetOrderByID returns a single order by order id.
func (c *CoinbasePro) GetOrderByID(ctx context.Context, orderID, clientOID, userNativeCurrency string) (*GetOrderResponse, error) {
	if orderID == "" {
		return nil, errOrderIDEmpty
	}
	var resp GetOrderResponse
	vals := url.Values{}
	if clientOID != "" {
		vals.Set("client_order_id", clientOID)
	}
	if userNativeCurrency != "" {
		vals.Set("user_native_currency", userNativeCurrency)
	}
	path := coinbaseV3 + coinbaseOrders + "/" + coinbaseHistorical + "/" + orderID
	return &resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, vals, nil, true, &resp, nil)
}

// PreviewOrder simulates the results of an order request
func (c *CoinbasePro) PreviewOrder(ctx context.Context, productID, side, orderType, stopDirection, marginType string, commissionValue, amount, limitPrice, stopPrice, tradableBalance, leverage float64, postOnly, isMax, skipFCMRiskCheck bool, endTime time.Time) (*PreviewOrderResp, error) {
	if amount == 0 {
		return nil, errAmountEmpty
	}
	orderConfig, err := prepareOrderConfig(orderType, side, stopDirection, amount, limitPrice, stopPrice, endTime,
		postOnly)
	if err != nil {
		return nil, err
	}
	commissionRate := map[string]string{"value": strconv.FormatFloat(commissionValue, 'f', -1, 64)}
	mt := formatMarginType(marginType)
	req := map[string]interface{}{
		"product_id":          productID,
		"side":                side,
		"commission_rate":     commissionRate,
		"order_configuration": orderConfig,
		"is_max":              isMax,
		"tradable_balance":    strconv.FormatFloat(tradableBalance, 'f', -1, 64),
		"skip_fcm_risk_check": skipFCMRiskCheck,
		"leverage":            strconv.FormatFloat(leverage, 'f', -1, 64),
	}
	if mt != "" {
		req["margin_type"] = mt
	}
	var resp *PreviewOrderResp
	path := coinbaseV3 + coinbaseOrders + "/" + coinbasePreview
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, req, true,
		&resp, nil)
}

// GetAllPortfolios returns a list of portfolios associated with the user
func (c *CoinbasePro) GetAllPortfolios(ctx context.Context, portfolioType string) ([]SimplePortfolioData, error) {
	var resp AllPortfolioResponse
	vals := url.Values{}
	if portfolioType != "" {
		vals.Set("portfolio_type", portfolioType)
	}
	return resp.Portfolios, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		coinbaseV3+coinbasePortfolios, vals, nil, true, &resp, nil)
}

// CreatePortfolio creates a new portfolio
func (c *CoinbasePro) CreatePortfolio(ctx context.Context, name string) (*SimplePortfolioResponse, error) {
	if name == "" {
		return nil, errNameEmpty
	}
	req := map[string]interface{}{"name": name}
	var resp *SimplePortfolioResponse
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		coinbaseV3+coinbasePortfolios, nil, req, true, &resp, nil)
}

// MovePortfolioFunds transfers funds between portfolios
func (c *CoinbasePro) MovePortfolioFunds(ctx context.Context, currency, from, to string, amount float64) (*MovePortfolioFundsResponse, error) {
	if from == "" || to == "" {
		return nil, errPortfolioIDEmpty
	}
	if currency == "" {
		return nil, errCurrencyEmpty
	}
	if amount == 0 {
		return nil, errAmountEmpty
	}
	funds := FundsData{
		Value:    strconv.FormatFloat(amount, 'f', -1, 64),
		Currency: currency}
	req := map[string]interface{}{
		"source_portfolio_uuid": from,
		"target_portfolio_uuid": to,
		"funds":                 funds}
	path := coinbaseV3 + coinbasePortfolios + "/" + coinbaseMoveFunds
	var resp *MovePortfolioFundsResponse
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		path, nil, req, true, &resp, nil)
}

// GetPortfolioByID provides detailed information on a single portfolio
func (c *CoinbasePro) GetPortfolioByID(ctx context.Context, portfolioID string) (*DetailedPortfolioResponse, error) {
	if portfolioID == "" {
		return nil, errPortfolioIDEmpty
	}
	path := coinbaseV3 + coinbasePortfolios + "/" + portfolioID
	var resp DetailedPortfolioResponse
	return &resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, nil, nil, true, &resp, nil)
}

// DeletePortfolio deletes a portfolio
func (c *CoinbasePro) DeletePortfolio(ctx context.Context, portfolioID string) error {
	if portfolioID == "" {
		return errPortfolioIDEmpty
	}
	path := coinbaseV3 + coinbasePortfolios + "/" + portfolioID
	return c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, nil, nil,
		true, nil, nil)
}

// EditPortfolio edits the name of a portfolio
func (c *CoinbasePro) EditPortfolio(ctx context.Context, portfolioID, name string) (*SimplePortfolioResponse, error) {
	if portfolioID == "" {
		return nil, errPortfolioIDEmpty
	}
	if name == "" {
		return nil, errNameEmpty
	}
	req := map[string]interface{}{"name": name}
	path := coinbaseV3 + coinbasePortfolios + "/" + portfolioID
	var resp *SimplePortfolioResponse
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPut,
		path, nil, req, true, &resp, nil)
}

// GetFuturesBalanceSummary returns information on balances related to Coinbase Financial Markets
// futures trading
func (c *CoinbasePro) GetFuturesBalanceSummary(ctx context.Context) (*FuturesBalanceSummary, error) {
	var resp *FuturesBalanceSummary
	path := coinbaseV3 + coinbaseCFM + "/" + coinbaseBalanceSummary
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, nil, nil, true, &resp, nil)
}

// GetAllFuturesPositions returns a list of all open positions in CFM futures products
func (c *CoinbasePro) GetAllFuturesPositions(ctx context.Context) ([]FuturesPosition, error) {
	var resp AllFuturesPositions
	path := coinbaseV3 + coinbaseCFM + "/" + coinbasePositions
	return resp.Positions, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, nil, nil, true, &resp, nil)
}

// GetFuturesPositionByID returns information on a single open position in CFM futures products
func (c *CoinbasePro) GetFuturesPositionByID(ctx context.Context, productID string) (*FuturesPosition, error) {
	if productID == "" {
		return nil, errProductIDEmpty
	}
	path := coinbaseV3 + coinbaseCFM + "/" + coinbasePositions + "/" + productID
	var resp *FuturesPosition
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, nil, nil, true, &resp, nil)
}

// ScheduleFuturesSweep schedules a sweep of funds from a CFTC-regulated futures account to a
// Coinbase USD Spot wallet. Request submitted before 5 pm ET are processed the following
// business day, requests submitted after are processed in 2 business days. Only one
// sweep request can be pending at a time. Funds transferred depend on the excess available
// in the futures account. An amount of 0 will sweep all available excess funds
func (c *CoinbasePro) ScheduleFuturesSweep(ctx context.Context, amount float64) (bool, error) {
	path := coinbaseV3 + coinbaseCFM + "/" + coinbaseSweeps + "/" + coinbaseSchedule
	req := make(map[string]interface{})
	if amount != 0 {
		req["usd_amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	}
	var resp SuccessBool
	return resp.Success, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		path, nil, req, true, &resp, nil)
}

// ListFuturesSweeps returns information on pending and/or processing requests to sweep funds
func (c *CoinbasePro) ListFuturesSweeps(ctx context.Context) ([]SweepData, error) {
	var resp ListFuturesSweepsResponse
	path := coinbaseV3 + coinbaseCFM + "/" + coinbaseSweeps
	return resp.Sweeps, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, nil, nil, true, &resp, nil)
}

// CancelPendingFuturesSweep cancels a pending sweep request
func (c *CoinbasePro) CancelPendingFuturesSweep(ctx context.Context) (bool, error) {
	path := coinbaseV3 + coinbaseCFM + "/" + coinbaseSweeps
	var resp SuccessBool
	return resp.Success, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete,
		path, nil, nil, true, &resp, nil)
}

// AllocatePortfolio allocates funds to a position in your perpetuals portfolio
func (c *CoinbasePro) AllocatePortfolio(ctx context.Context, portfolioID, productID, currency string, amount float64) error {
	if portfolioID == "" {
		return errPortfolioIDEmpty
	}
	if productID == "" {
		return errProductIDEmpty
	}
	if currency == "" {
		return errCurrencyEmpty
	}
	if amount == 0 {
		return errAmountEmpty
	}
	req := map[string]interface{}{
		"portfolio_uuid": portfolioID,
		"symbol":         productID,
		"currency":       currency,
		"amount":         strconv.FormatFloat(amount, 'f', -1, 64)}
	path := coinbaseV3 + coinbaseIntx + "/" + coinbaseAllocate
	return c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		path, nil, req, true, nil, nil)
}

// GetPerpetualsPortfolioSummary returns a summary of your perpetuals portfolio
func (c *CoinbasePro) GetPerpetualsPortfolioSummary(ctx context.Context, portfolioID string) (*PerpetualPortResponse, error) {
	if portfolioID == "" {
		return nil, errPortfolioIDEmpty
	}
	path := coinbaseV3 + coinbaseIntx + "/" + coinbasePortfolio + "/" + portfolioID
	var resp *PerpetualPortResponse
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, nil, nil, true, &resp, nil)
}

// GetAllPerpetualsPositions returns a list of all open positions in your perpetuals portfolio
func (c *CoinbasePro) GetAllPerpetualsPositions(ctx context.Context, portfolioID string) (*AllPerpPosResponse, error) {
	if portfolioID == "" {
		return nil, errPortfolioIDEmpty
	}
	path := coinbaseV3 + coinbaseIntx + "/" + coinbasePositions + "/" + portfolioID
	var resp *AllPerpPosResponse
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, nil, nil, true, &resp, nil)
}

// GetPerpetualsPositionByID returns information on a single open position in your perpetuals portfolio
func (c *CoinbasePro) GetPerpetualsPositionByID(ctx context.Context, portfolioID, productID string) (*OnePerpPosResponse, error) {
	if portfolioID == "" {
		return nil, errPortfolioIDEmpty
	}
	if productID == "" {
		return nil, errProductIDEmpty
	}
	path := coinbaseV3 + coinbaseIntx + "/" + coinbasePositions + "/" + portfolioID + "/" + productID
	var resp *OnePerpPosResponse
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, nil, nil, true, &resp, nil)
}

// GetTransactionSummary returns a summary of transactions with fee tiers, total volume,
// and fees
func (c *CoinbasePro) GetTransactionSummary(ctx context.Context, startDate, endDate time.Time, userNativeCurrency, productType, contractExpiryType string) (*TransactionSummary, error) {
	var params Params
	params.Values = url.Values{}
	err := params.prepareDateString(startDate, endDate, startDateString, endDateString)
	if err != nil {
		return nil, err
	}
	if contractExpiryType != "" {
		params.Values.Set("contract_expiry_type", contractExpiryType)
	}
	if productType != "" {
		params.Values.Set("product_type", productType)
	}
	if userNativeCurrency != "" {
		params.Values.Set("user_native_currency", userNativeCurrency)
	}
	var resp TransactionSummary
	return &resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		coinbaseV3+coinbaseTransactionSummary, params.Values, nil, true, &resp, nil)
}

// CreateConvertQuote creates a quote for a conversion between two currencies. The trade_id returned
// can be used to commit the trade, but that must be done within 10 minutes of the quote's creation
func (c *CoinbasePro) CreateConvertQuote(ctx context.Context, from, to, userIncentiveID, codeVal string, amount float64) (*ConvertResponse, error) {
	if from == "" || to == "" {
		return nil, errAccountIDEmpty
	}
	if amount == 0 {
		return nil, errAmountEmpty
	}
	path := coinbaseV3 + coinbaseConvert + "/" + coinbaseQuote
	tIM := map[string]interface{}{
		"user_incentive_id": userIncentiveID,
		"code_val":          codeVal}
	req := map[string]interface{}{
		"from_account":             from,
		"to_account":               to,
		"amount":                   strconv.FormatFloat(amount, 'f', -1, 64),
		"trade_incentive_metadata": tIM}
	var resp *ConvertResponse
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path,
		nil, req, true, &resp, nil)
}

// CommitConvertTrade commits a conversion between two currencies, using the trade_id returned
// from CreateConvertQuote
func (c *CoinbasePro) CommitConvertTrade(ctx context.Context, tradeID, from, to string) (*ConvertResponse, error) {
	if tradeID == "" {
		return nil, errTransactionIDEmpty
	}
	if from == "" || to == "" {
		return nil, errAccountIDEmpty
	}
	path := coinbaseV3 + coinbaseConvert + "/" + coinbaseTrade + "/" + tradeID
	req := map[string]interface{}{
		"from_account": from,
		"to_account":   to}
	var resp *ConvertResponse
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path,
		nil, req, true, &resp, nil)
}

// GetConvertTradeByID returns information on a conversion between two currencies
func (c *CoinbasePro) GetConvertTradeByID(ctx context.Context, tradeID, from, to string) (*ConvertResponse, error) {
	if tradeID == "" {
		return nil, errTransactionIDEmpty
	}
	if from == "" || to == "" {
		return nil, errAccountIDEmpty
	}
	path := coinbaseV3 + coinbaseConvert + "/" + coinbaseTrade + "/" + tradeID
	req := map[string]interface{}{
		"from_account": from,
		"to_account":   to}
	var resp *ConvertResponse
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path,
		nil, req, true, &resp, nil)
}

// GetV3Time returns the current server time, calling V3 of the API
func (c *CoinbasePro) GetV3Time(ctx context.Context) (*ServerTimeV3, error) {
	var resp *ServerTimeV3
	return resp, c.SendHTTPRequest(ctx, exchange.RestSpot, coinbaseV3+coinbaseTime, nil, &resp)
}

// GetAllPaymentMethods returns a list of all payment methods associated with the user's account
func (c *CoinbasePro) GetAllPaymentMethods(ctx context.Context) (*GetAllPaymentMethodsResp, error) {
	var resp *GetAllPaymentMethodsResp
	req := map[string]interface{}{"currency": "BTC"}
	path := coinbaseV3 + coinbasePaymentMethods
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, nil, req, true, &resp, nil)
}

// GetPaymentMethodByID returns information on a single payment method associated with the user's
// account
func (c *CoinbasePro) GetPaymentMethodByID(ctx context.Context, paymentMethodID string) (*GenPaymentMethodResp, error) {
	if paymentMethodID == "" {
		return nil, errPaymentMethodEmpty
	}
	path := coinbaseV3 + coinbasePaymentMethods + "/" + paymentMethodID
	var resp *GenPaymentMethodResp
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, nil, nil, true, &resp, nil)
}

// ListNotifications lists the notifications the user is subscribed to
func (c *CoinbasePro) ListNotifications(ctx context.Context, pag PaginationInp) (*ListNotificationsResponse, error) {
	var resp *ListNotificationsResponse
	var params Params
	params.Values = url.Values{}
	params.preparePagination(pag)
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		coinbaseV2+coinbaseNotifications, params.Values, nil, false, &resp, nil)
}

// GetCurrentUser returns information about the user associated with the API key
func (c *CoinbasePro) GetCurrentUser(ctx context.Context) (*UserResponse, error) {
	var resp *UserResponse
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		coinbaseV2+coinbaseUser, nil, nil, false, &resp, nil)
}

// GetAllWallets lists all accounts associated with the API key
func (c *CoinbasePro) GetAllWallets(ctx context.Context, pag PaginationInp) (*GetAllWalletsResponse, error) {
	var resp *GetAllWalletsResponse
	var params Params
	params.Values = url.Values{}
	params.preparePagination(pag)
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		coinbaseV2+coinbaseAccounts, params.Values, nil, false, &resp, nil)
}

// GetWalletByID returns information about a single wallet. In lieu of a wallet ID,
// a currency can be provided to get the primary account for that currency
func (c *CoinbasePro) GetWalletByID(ctx context.Context, walletID, currency string) (*GenWalletResponse, error) {
	if (walletID == "" && currency == "") || (walletID != "" && currency != "") {
		return nil, errCurrWalletConflict
	}
	var path string
	if walletID != "" {
		path = coinbaseV2 + coinbaseAccounts + "/" + walletID
	}
	if currency != "" {
		path = coinbaseV2 + coinbaseAccounts + "/" + currency
	}
	var resp *GenWalletResponse
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, nil, nil, false, &resp, nil)
}

// CreateAddress generates a crypto address for depositing to the specified wallet
func (c *CoinbasePro) CreateAddress(ctx context.Context, walletID, name string) (*GenAddrResponse, error) {
	if walletID == "" {
		return nil, errWalletIDEmpty
	}
	path := coinbaseV2 + coinbaseAccounts + "/" + walletID + "/" + coinbaseAddresses
	req := map[string]interface{}{"name": name}
	var resp *GenAddrResponse
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		path, nil, req, false, &resp, nil)
}

// GetAllAddresses returns information on all addresses associated with a wallet
func (c *CoinbasePro) GetAllAddresses(ctx context.Context, walletID string, pag PaginationInp) (*GetAllAddrResponse, error) {
	if walletID == "" {
		return nil, errWalletIDEmpty
	}
	path := coinbaseV2 + coinbaseAccounts + "/" + walletID + "/" + coinbaseAddresses
	var params Params
	params.Values = url.Values{}
	params.preparePagination(pag)
	var resp *GetAllAddrResponse
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, params.Values, nil, false, &resp, nil)
}

// GetAddressByID returns information on a single address associated with the specified wallet
func (c *CoinbasePro) GetAddressByID(ctx context.Context, walletID, addressID string) (*GenAddrResponse, error) {
	if walletID == "" {
		return nil, errWalletIDEmpty
	}
	if addressID == "" {
		return nil, errAddressIDEmpty
	}
	path := coinbaseV2 + coinbaseAccounts + "/" + walletID + "/" + coinbaseAddresses + "/" + addressID
	var resp *GenAddrResponse
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, nil, nil, false, &resp, nil)
}

// GetAddressTransactions returns a list of transactions associated with the specified address
func (c *CoinbasePro) GetAddressTransactions(ctx context.Context, walletID, addressID string, pag PaginationInp) (*ManyTransactionsResp, error) {
	if walletID == "" {
		return nil, errWalletIDEmpty
	}
	if addressID == "" {
		return nil, errAddressIDEmpty
	}
	path := coinbaseV2 + coinbaseAccounts + "/" + walletID + "/" + coinbaseAddresses + "/" + addressID + "/" +
		coinbaseTransactions
	var params Params
	params.Values = url.Values{}
	params.preparePagination(pag)
	var resp *ManyTransactionsResp
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, params.Values, nil, false, &resp, nil)
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
	path := coinbaseV2 + coinbaseAccounts + "/" + walletID + "/" + coinbaseTransactions
	req := map[string]interface{}{
		"type":                          traType,
		"to":                            to,
		"amount":                        strconv.FormatFloat(amount, 'f', -1, 64),
		"currency":                      currency,
		"description":                   description,
		"skip_notifications":            skipNotifications,
		"idem":                          idem,
		"to_financial_institution":      toFinancialInstitution,
		"financial_institution_website": financialInstitutionWebsite,
		"destination_tag":               destinationTag}
	var resp *GenTransactionResp
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		path, nil, req, false, &resp, nil)
}

// GetAllTransactions returns a list of transactions associated with the specified wallet
func (c *CoinbasePro) GetAllTransactions(ctx context.Context, walletID string, pag PaginationInp) (*ManyTransactionsResp, error) {
	if walletID == "" {
		return nil, errWalletIDEmpty
	}
	path := coinbaseV2 + coinbaseAccounts + "/" + walletID + "/" + coinbaseTransactions
	var params Params
	params.Values = url.Values{}
	params.preparePagination(pag)
	var resp *ManyTransactionsResp
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, params.Values, nil, false, &resp, nil)
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
	path := coinbaseV2 + coinbaseAccounts + "/" + walletID + "/" + coinbaseTransactions + "/" + transactionID
	var resp *GenTransactionResp
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, nil, nil, false, &resp, nil)
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
		path = coinbaseV2 + coinbaseAccounts + "/" + walletID + "/" + coinbaseDeposits
	case FiatWithdrawal:
		path = coinbaseV2 + coinbaseAccounts + "/" + walletID + "/" + coinbaseWithdrawals
	}
	req := map[string]interface{}{
		"currency":       currency,
		"payment_method": paymentMethod,
		"amount":         strconv.FormatFloat(amount, 'f', -1, 64),
		"commit":         commit}
	var resp *GenDeposWithdrResp
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		path, nil, req, false, &resp, nil)
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
		path = coinbaseV2 + coinbaseAccounts + "/" + walletID + "/" + coinbaseDeposits + "/" + depositID + "/" + coinbaseCommit
	case FiatWithdrawal:
		path = coinbaseV2 + coinbaseAccounts + "/" + walletID + "/" + coinbaseWithdrawals + "/" + depositID + "/" + coinbaseCommit
	}
	var resp *GenDeposWithdrResp
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		path, nil, nil, false, &resp, nil)
}

// GetAllFiatTransfers returns a list of transfers either to or from fiat payment methods and
// the specified wallet
func (c *CoinbasePro) GetAllFiatTransfers(ctx context.Context, walletID string, pag PaginationInp, transferType FiatTransferType) (*ManyDeposWithdrResp, error) {
	if walletID == "" {
		return nil, errWalletIDEmpty
	}
	var path string
	switch transferType {
	case FiatDeposit:
		path = coinbaseV2 + coinbaseAccounts + "/" + walletID + "/" + coinbaseDeposits
	case FiatWithdrawal:
		path = coinbaseV2 + coinbaseAccounts + "/" + walletID + "/" + coinbaseWithdrawals
	}
	var params Params
	params.Values = url.Values{}
	params.preparePagination(pag)
	var resp *ManyDeposWithdrResp
	err := c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, params.Values, nil, false, &resp, nil)
	if err != nil {
		return nil, err
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
	var path string
	switch transferType {
	case FiatDeposit:
		path = coinbaseV2 + coinbaseAccounts + "/" + walletID + "/" + coinbaseDeposits + "/" + depositID
	case FiatWithdrawal:
		path = coinbaseV2 + coinbaseAccounts + "/" + walletID + "/" + coinbaseWithdrawals + "/" + depositID
	}
	var resp *GenDeposWithdrResp
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		path, nil, nil, false, &resp, nil)
}

// GetFiatCurrencies lists currencies that Coinbase knows about
func (c *CoinbasePro) GetFiatCurrencies(ctx context.Context) ([]FiatData, error) {
	var resp GetFiatCurrenciesResp
	return resp.Data, c.SendHTTPRequest(ctx, exchange.RestSpot, coinbaseV2+coinbaseCurrencies, nil, &resp)
}

// GetCryptocurrencies lists cryptocurrencies that Coinbase knows about
func (c *CoinbasePro) GetCryptocurrencies(ctx context.Context) ([]CryptoData, error) {
	var resp GetCryptocurrenciesResp
	path := coinbaseV2 + coinbaseCurrencies + "/" + coinbaseCrypto
	return resp.Data, c.SendHTTPRequest(ctx, exchange.RestSpot, path, nil, &resp)
}

// GetExchangeRates returns exchange rates for the specified currency. If none is specified,
// it defaults to USD
func (c *CoinbasePro) GetExchangeRates(ctx context.Context, currency string) (*GetExchangeRatesResp, error) {
	var resp *GetExchangeRatesResp
	vals := url.Values{}
	vals.Set("currency", currency)
	return resp, c.SendHTTPRequest(ctx, exchange.RestSpot, coinbaseV2+coinbaseExchangeRates, vals, &resp)
}

// GetPrice returns the price the spot/buy/sell price for the specified currency pair,
// including the standard Coinbase fee of 1%, but excluding any other fees
func (c *CoinbasePro) GetPrice(ctx context.Context, currencyPair, priceType string) (*GetPriceResp, error) {
	var path string
	switch priceType {
	case "spot", "buy", "sell":
		path = coinbaseV2 + coinbasePrices + "/" + currencyPair + "/" + priceType
	default:
		return nil, errInvalidPriceType
	}
	var resp *GetPriceResp
	return resp, c.SendHTTPRequest(ctx, exchange.RestSpot, path, nil, &resp)
}

// GetV2Time returns the current server time, calling V2 of the API
func (c *CoinbasePro) GetV2Time(ctx context.Context) (*ServerTimeV2, error) {
	var resp *ServerTimeV2
	return resp, c.SendHTTPRequest(ctx, exchange.RestSpot, coinbaseV2+coinbaseTime, nil, &resp)
}

// GetAllCurrencies returns a list of all currencies that Coinbase knows about. These aren't necessarily tradable
func (c *CoinbasePro) GetAllCurrencies(ctx context.Context) ([]CurrencyData, error) {
	var resp []CurrencyData
	return resp, c.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, coinbaseCurrencies, nil, &resp)
}

// GetACurrency returns information on a single currency specified by the user
func (c *CoinbasePro) GetACurrency(ctx context.Context, currency string) (*CurrencyData, error) {
	if currency == "" {
		return nil, errCurrencyEmpty
	}
	var resp *CurrencyData
	path := coinbaseCurrencies + "/" + currency
	return resp, c.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, nil, &resp)
}

// GetAllTradingPairs returns a list of currency pairs which are available for trading
func (c *CoinbasePro) GetAllTradingPairs(ctx context.Context, pairType string) ([]PairData, error) {
	var resp []PairData
	vals := url.Values{}
	if pairType != "" {
		vals.Set("type", pairType)
	}
	return resp, c.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, coinbaseProducts, vals, &resp)
}

// GetAllPairVolumes returns a list of currency pairs and their associated volumes
func (c *CoinbasePro) GetAllPairVolumes(ctx context.Context) ([]PairVolumeData, error) {
	var resp []PairVolumeData
	path := coinbaseProducts + "/" + coinbaseVolumeSummary
	return resp, c.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, nil, &resp)
}

// GetPairDetails returns information on a single currency pair
func (c *CoinbasePro) GetPairDetails(ctx context.Context, pair string) (*PairData, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	var resp *PairData
	path := coinbaseProducts + "/" + pair
	return resp, c.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, nil, &resp)
}

// GetProductBookV1 returns the order book for the specified currency pair. Level 1 only returns the best bids and asks,
// Level 2 returns the full order book with orders at the same price aggregated, Level 3 returns the full
// non-aggregated order book.
func (c *CoinbasePro) GetProductBookV1(ctx context.Context, pair string, level uint8) (*OrderBook, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	var resp OrderBookResp
	vals := url.Values{}
	vals.Set("level", strconv.FormatUint(uint64(level), 10))
	path := coinbaseProducts + "/" + pair + "/" + coinbaseBook
	err := c.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, vals, &resp)
	if err != nil {
		return nil, err
	}
	ob := &OrderBook{
		Sequence:    resp.Sequence,
		Bids:        make([]Orders, len(resp.Bids)),
		Asks:        make([]Orders, len(resp.Asks)),
		AuctionMode: resp.AuctionMode,
		Auction:     resp.Auction,
		Time:        resp.Time,
	}
	for i := range resp.Bids {
		tempS1, ok := resp.Bids[i][0].(string)
		if !ok {
			return nil, fmt.Errorf("%w, %v", errStringConvert, resp.Bids[i][0])
		}
		tempF1, err := strconv.ParseFloat(tempS1, 64)
		if err != nil {
			return nil, err
		}
		tempS2, ok := resp.Bids[i][1].(string)
		if !ok {
			return nil, fmt.Errorf("%w, %v", errStringConvert, resp.Bids[i][1])
		}
		tempF2, err := strconv.ParseFloat(tempS2, 64)
		if err != nil {
			return nil, err
		}
		switch tempV := resp.Bids[i][2].(type) {
		case string:
			tempU, err := uuid.FromString(tempV)
			if err != nil {
				return nil, err
			}
			ob.Bids[i] = Orders{Price: tempF1, Size: tempF2, OrderCount: 1, OrderID: tempU}
		case float64:
			tempU := uint64(tempV)
			ob.Bids[i] = Orders{Price: tempF1, Size: tempF2, OrderCount: tempU}
		}
	}
	for i := range resp.Asks {
		tempS1, ok := resp.Asks[i][0].(string)
		if !ok {
			return nil, fmt.Errorf("%w, %v", errStringConvert, resp.Asks[i][0])
		}
		tempF1, err := strconv.ParseFloat(tempS1, 64)
		if err != nil {
			return nil, err
		}
		tempS2, ok := resp.Asks[i][1].(string)
		if !ok {
			return nil, fmt.Errorf("%w, %v", errStringConvert, resp.Asks[i][1])
		}
		tempF2, err := strconv.ParseFloat(tempS2, 64)
		if err != nil {
			return nil, err
		}
		switch tempV := resp.Asks[i][2].(type) {
		case string:
			tempU, err := uuid.FromString(tempV)
			if err != nil {
				return nil, err
			}
			ob.Asks[i] = Orders{Price: tempF1, Size: tempF2, OrderCount: 1, OrderID: tempU}
		case float64:
			tempU := uint64(tempV)
			ob.Asks[i] = Orders{Price: tempF1, Size: tempF2, OrderCount: tempU}
		}
	}
	return ob, nil
}

// GetProductCandles returns historical market data for the specified currency pair.
func (c *CoinbasePro) GetProductCandles(ctx context.Context, pair string, granularity uint32, startTime, endTime time.Time) ([]Candle, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	var params Params
	params.Values = url.Values{}
	err := params.prepareDateString(startTime, endTime, "start", "end")
	if err != nil {
		return nil, err
	}
	if granularity != 0 {
		params.Values.Set("granularity", strconv.FormatUint(uint64(granularity), 10))
	}
	path := coinbaseProducts + "/" + pair + "/" + coinbaseCandles
	var resp []RawCandles
	err = c.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, params.Values, &resp)
	if err != nil {
		return nil, err
	}
	candles := make([]Candle, len(resp))
	for i := range resp {
		f1, ok := resp[i][0].(float64)
		if !ok {
			return nil, fmt.Errorf("%w, %v", errFloatConvert, resp[i][0])
		}
		ti := int64(f1)
		t := time.Unix(ti, 0)
		f2, ok := resp[i][1].(float64)
		if !ok {
			return nil, fmt.Errorf("%w, %v", errFloatConvert, resp[i][1])
		}
		f3, ok := resp[i][2].(float64)
		if !ok {
			return nil, fmt.Errorf("%w, %v", errFloatConvert, resp[i][2])
		}
		f4, ok := resp[i][3].(float64)
		if !ok {
			return nil, fmt.Errorf("%w, %v", errFloatConvert, resp[i][3])
		}
		f5, ok := resp[i][4].(float64)
		if !ok {
			return nil, fmt.Errorf("%w, %v", errFloatConvert, resp[i][4])
		}
		f6, ok := resp[i][5].(float64)
		if !ok {
			return nil, fmt.Errorf("%w, %v", errFloatConvert, resp[i][5])
		}
		candles[i] = Candle{
			Time:   t,
			Low:    f2,
			High:   f3,
			Open:   f4,
			Close:  f5,
			Volume: f6,
		}
	}
	return candles, nil
}

// GetProductStats returns information on a specific pair's price and volume
func (c *CoinbasePro) GetProductStats(ctx context.Context, pair string) (*ProductStats, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	path := coinbaseProducts + "/" + pair + "/" + coinbaseStats
	var resp *ProductStats
	return resp, c.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, nil, &resp)
}

// GetProductTicker returns the ticker for the specified currency pair
func (c *CoinbasePro) GetProductTicker(ctx context.Context, pair string) (*ProductTicker, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	path := coinbaseProducts + "/" + pair + "/" + coinbaseTicker
	var resp *ProductTicker
	return resp, c.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, nil, &resp)
}

// GetProductTrades returns a list of the latest traides for a pair
func (c *CoinbasePro) GetProductTrades(ctx context.Context, pair, step, direction string, limit int64) ([]ProductTrades, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	vals := url.Values{}
	if step != "" {
		vals.Set(direction, step)
	}
	vals.Set("limit", strconv.FormatInt(limit, 10))
	path := coinbaseProducts + "/" + pair + "/" + coinbaseTrades
	var resp []ProductTrades
	return resp, c.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, vals, &resp)
}

// GetAllWrappedAssets returns a list of supported wrapped assets
func (c *CoinbasePro) GetAllWrappedAssets(ctx context.Context) (*AllWrappedAssets, error) {
	var resp *AllWrappedAssets
	return resp, c.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, coinbaseWrappedAssets, nil, &resp)
}

// GetWrappedAssetDetails returns information on a single wrapped asset
func (c *CoinbasePro) GetWrappedAssetDetails(ctx context.Context, wrappedAsset string) (*WrappedAsset, error) {
	if wrappedAsset == "" {
		return nil, errWrappedAssetEmpty
	}
	var resp *WrappedAsset
	path := coinbaseWrappedAssets + "/" + wrappedAsset
	return resp, c.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, nil, &resp)
}

// GetWrappedAssetConversionRate returns the conversion rate for the specified wrapped asset
func (c *CoinbasePro) GetWrappedAssetConversionRate(ctx context.Context, wrappedAsset string) (*WrappedAssetConversionRate, error) {
	if wrappedAsset == "" {
		return nil, errWrappedAssetEmpty
	}
	var resp *WrappedAssetConversionRate
	path := coinbaseWrappedAssets + "/" + wrappedAsset + "/" + coinbaseConversionRate
	return resp, c.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, nil, &resp)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (c *CoinbasePro) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, vals url.Values, result interface{}) error {
	endpoint, err := c.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	path = common.EncodeURLValues(path, vals)
	item := &request.Item{
		Method:        http.MethodGet,
		Path:          endpoint + path,
		Result:        result,
		Verbose:       c.Verbose,
		HTTPDebugging: c.HTTPDebugging,
		HTTPRecording: c.HTTPRecording,
	}
	rLim := PubRate
	if strings.Contains(path, coinbaseV2) {
		rLim = V2Rate
	}
	return c.SendPayload(ctx, rLim, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request
func (c *CoinbasePro) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, method, path string, queryParams url.Values, bodyParams map[string]interface{}, isVersion3 bool, result interface{}, returnHead *http.Header) (err error) {
	creds, err := c.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpoint, err := c.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	queryString := common.EncodeURLValues("", queryParams)
	// Version 2 wants query params in the path during signing
	if !isVersion3 {
		path += queryString
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
		var hmac []byte
		hmac, err = crypto.GetHMAC(crypto.HashSHA256,
			[]byte(message),
			[]byte(creds.Secret))
		if err != nil {
			return nil, err
		}
		// TODO: Implement JWT authentication once it's supported by all endpoints we care about
		headers := make(map[string]string)
		headers["CB-ACCESS-KEY"] = creds.Key
		headers["CB-ACCESS-SIGN"] = hex.EncodeToString(hmac)
		headers["CB-ACCESS-TIMESTAMP"] = n
		headers["Content-Type"] = "application/json"
		headers["CB-VERSION"] = "2024-02-27"
		// Version 3 only wants query params in the path when the request is sent
		if isVersion3 {
			path += queryString
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
	if isVersion3 {
		rateLim = V3Rate
	}
	err = c.SendPayload(ctx, rateLim, newRequest, request.AuthenticatedRequest)
	// Doing this error handling because the docs indicate that errors can be returned even with a 200 status
	// code, and that these errors can be buried in the JSON returned
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
	if err = json.Unmarshal(interim, &singleErrCap); err == nil {
		if singleErrCap.Message != "" {
			return fmt.Errorf("message: %s, error type: %s, error details: %s, edit failure reason: %s, preview failure reason: %s, new order failure reason: %s",
				singleErrCap.Message, singleErrCap.ErrorType, singleErrCap.ErrorDetails, singleErrCap.EditFailureReason,
				singleErrCap.PreviewFailureReason, singleErrCap.NewOrderFailureReason)
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
		fee = StablePairMakerFee
	case !feeBuilder.IsMaker && isStablePair(feeBuilder.Pair) &&
		(feeBuilder.FeeType == exchange.CryptocurrencyTradeFee || feeBuilder.FeeType == exchange.OfflineTradeFee):
		fee = WorstCaseStablePairTakerFee
	case feeBuilder.IsMaker && !isStablePair(feeBuilder.Pair) && feeBuilder.FeeType == exchange.OfflineTradeFee:
		fee = WorstCaseMakerFee
	case !feeBuilder.IsMaker && !isStablePair(feeBuilder.Pair) && feeBuilder.FeeType == exchange.OfflineTradeFee:
		fee = WorstCaseTakerFee
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
	if err != nil {
		if errors.Is(err, common.ErrDateUnset) {
			return nil
		}
		return err
	}
	p.Values.Set(labelStart, startDate.Format(time.RFC3339))
	p.Values.Set(labelEnd, endDate.Format(time.RFC3339))
	return nil
}

// PreparePagination formats pagination information in the way the exchange expects
func (p *Params) preparePagination(pag PaginationInp) {
	if pag.Limit != 0 {
		p.Values.Set("limit", strconv.FormatInt(int64(pag.Limit), 10))
	}
	if pag.OrderAscend {
		p.Values.Set("order", "asc")
	}
	if pag.StartingAfter != "" {
		p.Values.Set("starting_after", pag.StartingAfter)
	}
	if pag.EndingBefore != "" {
		p.Values.Set("ending_before", pag.EndingBefore)
	}
}

// prepareOrderConfig populates the OrderConfiguration struct
func prepareOrderConfig(orderType, side, stopDirection string, amount, limitPrice, stopPrice float64, endTime time.Time, postOnly bool) (OrderConfiguration, error) {
	var orderConfig OrderConfiguration
	switch orderType {
	case order.Market.String(), order.ImmediateOrCancel.String():
		orderConfig.MarketMarketIOC = &MarketMarketIOC{}
		if side == order.Buy.String() {
			orderConfig.MarketMarketIOC.QuoteSize = types.Number(amount)
		}
		if side == order.Sell.String() {
			orderConfig.MarketMarketIOC.BaseSize = types.Number(amount)
		}
	case order.Limit.String():
		if endTime == (time.Time{}) {
			orderConfig.LimitLimitGTC = &LimitLimitGTC{}
			orderConfig.LimitLimitGTC.BaseSize = types.Number(amount)
			orderConfig.LimitLimitGTC.LimitPrice = types.Number(limitPrice)
			orderConfig.LimitLimitGTC.PostOnly = postOnly
		} else {
			orderConfig.LimitLimitGTD = &LimitLimitGTD{}
			orderConfig.LimitLimitGTD.BaseSize = types.Number(amount)
			orderConfig.LimitLimitGTD.LimitPrice = types.Number(limitPrice)
			orderConfig.LimitLimitGTD.PostOnly = postOnly
			orderConfig.LimitLimitGTD.EndTime = endTime
		}
	case order.StopLimit.String():
		if endTime == (time.Time{}) {
			orderConfig.StopLimitStopLimitGTC = &StopLimitStopLimitGTC{}
			orderConfig.StopLimitStopLimitGTC.BaseSize = types.Number(amount)
			orderConfig.StopLimitStopLimitGTC.LimitPrice = types.Number(limitPrice)
			orderConfig.StopLimitStopLimitGTC.StopPrice = types.Number(stopPrice)
			orderConfig.StopLimitStopLimitGTC.StopDirection = stopDirection
		} else {
			orderConfig.StopLimitStopLimitGTD = &StopLimitStopLimitGTD{}
			orderConfig.StopLimitStopLimitGTD.BaseSize = types.Number(amount)
			orderConfig.StopLimitStopLimitGTD.LimitPrice = types.Number(limitPrice)
			orderConfig.StopLimitStopLimitGTD.StopPrice = types.Number(stopPrice)
			orderConfig.StopLimitStopLimitGTD.StopDirection = stopDirection
			orderConfig.StopLimitStopLimitGTD.EndTime = endTime
		}
	default:
		return orderConfig, errInvalidOrderType
	}
	return orderConfig, nil
}

// formatMarginType properly formats the margin type for the request
func formatMarginType(marginType string) string {
	if marginType == "ISOLATED" || marginType == "CROSS" {
		return marginType
	}
	if marginType == "MULTI" {
		return "CROSS"
	}
	return ""
}

// String implements the stringer interface
func (f FiatTransferType) String() string {
	if f {
		return "withdrawal"
	}
	return "deposit"
}

// UnmarshalJSON unmarshals the JSON input into a UnixTimestamp type
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

// String implements the stringer interface
func (t *UnixTimestamp) String() string {
	return t.Time().String()
}

// Time returns the time.Time representation of the UnixTimestamp
func (t *UnixTimestamp) Time() time.Time {
	return time.Time(*t)
}
