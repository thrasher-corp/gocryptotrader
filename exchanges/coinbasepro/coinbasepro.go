package coinbasepro

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	coinbaseAPIURL              = "https://api.coinbase.com"
	coinbaseV1APIURL            = "https://api.exchange.coinbase.com/"
	coinbaseproSandboxAPIURL    = "https://api-sandbox.coinbase.com"
	tradeBaseURL                = "https://www.coinbase.com/advanced-trade/spot/"
	coinbaseV3                  = "/api/v3/brokerage/"
	coinbaseAccounts            = "accounts"
	coinbaseConvert             = "convert"
	coinbaseTrade               = "trade"
	coinbaseQuote               = "quote"
	coinbaseKeyPermissions      = "key_permissions"
	coinbaseTransactionSummary  = "transaction_summary"
	coinbaseCFM                 = "cfm"
	coinbaseSweeps              = "sweeps"
	coinbaseIntraday            = "intraday"
	coinbaseCurrentMarginWindow = "current_margin_window"
	coinbaseBalanceSummary      = "balance_summary"
	coinbasePositions           = "positions"
	coinbaseMarginSetting       = "margin_setting"
	coinbaseSchedule            = "schedule"
	coinbaseOrders              = "orders"
	coinbaseBatchCancel         = "batch_cancel"
	coinbaseClosePosition       = "close_position"
	coinbaseEdit                = "edit"
	coinbaseEditPreview         = "edit_preview"
	coinbaseHistorical          = "historical"
	coinbaseFills               = "fills"
	coinbaseBatch               = "batch"

	coinbaseBestBidAsk     = "best_bid_ask"
	coinbaseProductBook    = "product_book"
	coinbaseProducts       = "products"
	coinbaseCandles        = "candles"
	coinbaseTicker         = "ticker"
	coinbasePreview        = "preview"
	coinbasePortfolios     = "portfolios"
	coinbaseMoveFunds      = "move_funds"
	coinbaseIntx           = "intx"
	coinbaseAllocate       = "allocate"
	coinbasePortfolio      = "portfolio"
	coinbasePaymentMethods = "payment_methods"
	coinbaseV2             = "/v2/"
	coinbaseNotifications  = "notifications"
	coinbaseUser           = "user"
	coinbaseAddresses      = "addresses"
	coinbaseTransactions   = "transactions"
	coinbaseDeposits       = "deposits"
	coinbaseCommit         = "commit"
	coinbaseWithdrawals    = "withdrawals"
	coinbaseCurrencies     = "currencies"
	coinbaseCrypto         = "crypto"
	coinbaseExchangeRates  = "exchange-rates"
	coinbasePrices         = "prices"
	coinbaseTime           = "time"
	coinbaseVolumeSummary  = "volume-summary"
	coinbaseBook           = "book"
	coinbaseStats          = "stats"
	coinbaseTrades         = "trades"
	coinbaseWrappedAssets  = "wrapped-assets"
	coinbaseConversionRate = "conversion-rate"
	coinbaseMarket         = "market"

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

	warnSequenceIssue = "Out of order sequence number. Received %v, expected %v"
	warnAuth          = "%v authenticated request failed, attempting unauthenticated"

	manyFills = 3000
	manyOrds  = 2147483647
)

// Constants defining whether a transfer is a deposit or withdrawal, used to simplify interactions with a few endpoints
const (
	FiatDeposit    FiatTransferType = false
	FiatWithdrawal FiatTransferType = true
)

// While the exchange's fee pages say the worst taker/maker fees are lower than the ones listed here, the data returned by the GetTransactionsSummary endpoint are consistent with these worst case scenarios. The best case scenarios are untested, and assumed to be in line with the fee pages
const (
	WorstCaseTakerFee           = 0.012
	WorstCaseMakerFee           = 0.006
	BestCaseTakerFee            = 0.0005
	BestCaseMakerFee            = 0
	StablePairMakerFee          = 0
	WorstCaseStablePairTakerFee = 0.000045
)

var (
	errAccountIDEmpty           = errors.New("account id cannot be empty")
	errClientOrderIDEmpty       = errors.New("client order id cannot be empty")
	errProductIDEmpty           = errors.New("product id cannot be empty")
	errOrderIDEmpty             = errors.New("order ids cannot be empty")
	errCancelLimitExceeded      = errors.New("100 order cancel limit exceeded")
	errSizeAndPriceZero         = errors.New("size and price cannot both be 0")
	errCurrWalletConflict       = errors.New("exactly one of walletID and currency must be specified")
	errWalletIDEmpty            = errors.New("wallet id cannot be empty")
	errAddressIDEmpty           = errors.New("address id cannot be empty")
	errTransactionTypeEmpty     = errors.New("transaction type cannot be empty")
	errToEmpty                  = errors.New("to cannot be empty")
	errTransactionIDEmpty       = errors.New("transaction id cannot be empty")
	errPaymentMethodEmpty       = errors.New("payment method cannot be empty")
	errDepositIDEmpty           = errors.New("deposit id cannot be empty")
	errInvalidPriceType         = errors.New("price type must be spot, buy, or sell")
	errInvalidOrderType         = errors.New("order type must be market, limit, or stop")
	errEndTimeInPast            = errors.New("end time cannot be in the past")
	errNoMatchingWallets        = errors.New("no matching wallets returned")
	errOrderModFailNoRet        = errors.New("order modification failed but no error returned")
	errNameEmpty                = errors.New("name cannot be empty")
	errPortfolioIDEmpty         = errors.New("portfolio id cannot be empty")
	errFeeTypeNotSupported      = errors.New("fee type not supported")
	errCantDecodePrivKey        = errors.New("cannot decode private key")
	errNoWalletForCurrency      = errors.New("no wallet found for currency, address creation impossible")
	errChannelNameUnknown       = errors.New("unknown channel name")
	errNoWalletsReturned        = errors.New("no wallets returned")
	errPayMethodNotFound        = errors.New("payment method not found")
	errUnknownL2DataType        = errors.New("unknown l2update data type")
	errOrderFailedToCancel      = errors.New("failed to cancel order")
	errWrappedAssetEmpty        = errors.New("wrapped asset cannot be empty")
	errUnrecognisedStrategyType = errors.New("unrecognised strategy type")
	errEndpointPathInvalid      = errors.New("endpoint path invalid, should start with https://")
	errPairsDisabledOrErrored   = errors.New("pairs are either disabled or errored")
	errDateLabelEmpty           = errors.New("date label cannot be empty")
	errParamValuesNil           = errors.New("param values cannot be nil")
	errMarginProfileTypeEmpty   = errors.New("margin profile type cannot be empty")
	errSettingEmpty             = errors.New("setting cannot be empty")

	allowedGranularities = []string{granOneMin, granFiveMin, granFifteenMin, granThirtyMin, granOneHour, granTwoHour, granSixHour, granOneDay}
	closedStatuses       = []string{"FILLED", "CANCELLED", "EXPIRED", "FAILED"}
	openStatus           = []string{"OPEN"}
)

// GetAccountByID returns information for a single account
func (c *CoinbasePro) GetAccountByID(ctx context.Context, accountID string) (*Account, error) {
	if accountID == "" {
		return nil, errAccountIDEmpty
	}
	path := coinbaseV3 + coinbaseAccounts + "/" + accountID
	resp := struct {
		Account Account `json:"account"`
	}{}
	return &resp.Account, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, true, &resp)
}

// ListAccounts returns information on all trading accounts associated with the API key
func (c *CoinbasePro) ListAccounts(ctx context.Context, limit uint8, cursor string) (*AllAccountsResponse, error) {
	vals := url.Values{}
	if limit != 0 {
		vals.Set("limit", strconv.FormatUint(uint64(limit), 10))
	}
	if cursor != "" {
		vals.Set("cursor", cursor)
	}
	var resp AllAccountsResponse
	return &resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseV3+coinbaseAccounts, vals, nil, true, &resp)
}

// CommitConvertTrade commits a conversion between two currencies, using the trade_id returned from CreateConvertQuote
func (c *CoinbasePro) CommitConvertTrade(ctx context.Context, tradeID, from, to string) (*ConvertResponse, error) {
	if tradeID == "" {
		return nil, errTransactionIDEmpty
	}
	if from == "" || to == "" {
		return nil, errAccountIDEmpty
	}
	path := coinbaseV3 + coinbaseConvert + "/" + coinbaseTrade + "/" + tradeID
	req := map[string]any{
		"from_account": from,
		"to_account":   to,
	}
	resp := struct {
		Trade ConvertResponse `json:"trade"`
	}{}
	return &resp.Trade, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, req, true, &resp)
}

// CreateConvertQuote creates a quote for a conversion between two currencies. The trade_id returned can be used to commit the trade, but that must be done within 10 minutes of the quote's creation
func (c *CoinbasePro) CreateConvertQuote(ctx context.Context, from, to, userIncentiveID, codeVal string, amount float64) (*ConvertResponse, error) {
	if from == "" || to == "" {
		return nil, errAccountIDEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	path := coinbaseV3 + coinbaseConvert + "/" + coinbaseQuote
	tIM := map[string]any{
		"user_incentive_id": userIncentiveID,
		"code_val":          codeVal,
	}
	req := map[string]any{
		"from_account":             from,
		"to_account":               to,
		"amount":                   strconv.FormatFloat(amount, 'f', -1, 64),
		"trade_incentive_metadata": tIM,
	}
	resp := struct {
		Trade ConvertResponse `json:"trade"`
	}{}
	return &resp.Trade, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, req, true, &resp)
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
	req := map[string]any{
		"from_account": from,
		"to_account":   to,
	}
	resp := struct {
		Trade ConvertResponse `json:"trade"`
	}{}
	return &resp.Trade, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, req, true, &resp)
}

// GetPermissions returns the permissions associated with the API key
func (c *CoinbasePro) GetPermissions(ctx context.Context) (*PermissionsResponse, error) {
	var resp PermissionsResponse
	return &resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseV3+coinbaseKeyPermissions, nil, nil, true, &resp)
}

// GetTransactionSummary returns a summary of transactions with fee tiers, total volume, and fees
func (c *CoinbasePro) GetTransactionSummary(ctx context.Context, startDate, endDate time.Time, productVenue, productType, contractExpiryType string) (*TransactionSummary, error) {
	var params Params
	params.Values = url.Values{}
	err := params.encodeDateRange(startDate, endDate, startDateString, endDateString)
	if err != nil {
		return nil, err
	}
	if contractExpiryType != "" {
		params.Values.Set("contract_expiry_type", contractExpiryType)
	}
	if productType != "" {
		params.Values.Set("product_type", productType)
	}
	if productVenue != "" {
		params.Values.Set("product_venue", productVenue)
	}
	var resp TransactionSummary
	return &resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseV3+coinbaseTransactionSummary, params.Values, nil, true, &resp)
}

// CancelPendingFuturesSweep cancels a pending sweep request
func (c *CoinbasePro) CancelPendingFuturesSweep(ctx context.Context) (bool, error) {
	path := coinbaseV3 + coinbaseCFM + "/" + coinbaseSweeps
	resp := struct {
		Success bool `json:"success"`
	}{}
	return resp.Success, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, nil, nil, true, &resp)
}

// GetCurrentMarginWindow returns the futures current margin window
func (c *CoinbasePro) GetCurrentMarginWindow(ctx context.Context, marginProfileType string) (*CurrentMarginWindow, error) {
	if marginProfileType == "" {
		return nil, errMarginProfileTypeEmpty
	}
	vals := url.Values{}
	if marginProfileType != "" {
		vals.Set("margin_profile_type", marginProfileType)
	}
	path := coinbaseV3 + coinbaseCFM + "/" + coinbaseIntraday + "/" + coinbaseCurrentMarginWindow
	var resp CurrentMarginWindow
	return &resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, vals, nil, true, &resp)
}

// GetFuturesBalanceSummary returns information on balances related to Coinbase Financial Markets futures trading
func (c *CoinbasePro) GetFuturesBalanceSummary(ctx context.Context) (*FuturesBalanceSummary, error) {
	resp := struct {
		BalanceSummary FuturesBalanceSummary `json:"balance_summary"`
	}{}
	path := coinbaseV3 + coinbaseCFM + "/" + coinbaseBalanceSummary
	return &resp.BalanceSummary, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, true, &resp)
}

// GetFuturesPositionByID returns information on a single open position in CFM futures products
func (c *CoinbasePro) GetFuturesPositionByID(ctx context.Context, productID string) (*FuturesPosition, error) {
	if productID == "" {
		return nil, errProductIDEmpty
	}
	path := coinbaseV3 + coinbaseCFM + "/" + coinbasePositions + "/" + productID
	resp := struct {
		Position FuturesPosition `json:"position"`
	}{}
	return &resp.Position, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, true, &resp)
}

// GetIntradayMarginSetting returns the futures intraday margin setting
func (c *CoinbasePro) GetIntradayMarginSetting(ctx context.Context) (string, error) {
	resp := struct {
		Setting string `json:"setting"`
	}{}
	path := coinbaseV3 + coinbaseCFM + "/" + coinbaseIntraday + "/" + coinbaseMarginSetting
	return resp.Setting, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, true, &resp)
}

// ListFuturesPositions returns a list of all open positions in CFM futures products
func (c *CoinbasePro) ListFuturesPositions(ctx context.Context) ([]FuturesPosition, error) {
	resp := struct {
		Positions []FuturesPosition `json:"positions"`
	}{}
	path := coinbaseV3 + coinbaseCFM + "/" + coinbasePositions
	return resp.Positions, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, true, &resp)
}

// ListFuturesSweeps returns information on pending and/or processing requests to sweep funds
func (c *CoinbasePro) ListFuturesSweeps(ctx context.Context) ([]SweepData, error) {
	resp := struct {
		Sweeps []SweepData `json:"sweeps"`
	}{}
	path := coinbaseV3 + coinbaseCFM + "/" + coinbaseSweeps
	return resp.Sweeps, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, true, &resp)
}

// ScheduleFuturesSweep schedules a sweep of funds from a CFTC-regulated futures account to a Coinbase USD Spot wallet. Request submitted before 5 pm ET are processed the following business day, requests submitted after are processed in 2 business days. Only one sweep request can be pending at a time. Funds transferred depend on the excess available in the futures account. An amount of 0 will sweep all available excess funds
func (c *CoinbasePro) ScheduleFuturesSweep(ctx context.Context, amount float64) (bool, error) {
	path := coinbaseV3 + coinbaseCFM + "/" + coinbaseSweeps + "/" + coinbaseSchedule
	var req map[string]any
	if amount != 0 {
		req = map[string]any{"usd_amount": strconv.FormatFloat(amount, 'f', -1, 64)}
	}
	resp := struct {
		Success bool `json:"success"`
	}{}
	return resp.Success, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, req, true, &resp)
}

// SetIntradayMarginSetting sets the futures intraday margin setting
func (c *CoinbasePro) SetIntradayMarginSetting(ctx context.Context, setting string) error {
	if setting == "" {
		return errSettingEmpty
	}
	path := coinbaseV3 + coinbaseCFM + "/" + coinbaseIntraday + "/" + coinbaseMarginSetting
	req := map[string]any{"setting": setting}
	return c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, req, true, nil)
}

// CancelOrders cancels orders by orderID. Can only cancel 100 orders per request
func (c *CoinbasePro) CancelOrders(ctx context.Context, orderIDs []string) ([]OrderCancelDetail, error) {
	if len(orderIDs) == 0 {
		return nil, errOrderIDEmpty
	}
	if len(orderIDs) > 100 {
		return nil, errCancelLimitExceeded
	}
	path := coinbaseV3 + coinbaseOrders + "/" + coinbaseBatchCancel
	req := map[string]any{"order_ids": orderIDs}
	resp := struct {
		Results []OrderCancelDetail `json:"results"`
	}{}
	return resp.Results, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, req, true, &resp)
}

// ClosePosition closes a position by client order ID, product ID, and size
func (c *CoinbasePro) ClosePosition(ctx context.Context, clientOrderID string, productID currency.Pair, size float64) (*SuccessFailureConfig, error) {
	if clientOrderID == "" {
		return nil, errClientOrderIDEmpty
	}
	if productID.IsEmpty() {
		return nil, errProductIDEmpty
	}
	if size <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	path := coinbaseV3 + coinbaseOrders + "/" + coinbaseClosePosition
	req := map[string]any{
		"client_order_id": clientOrderID,
		"product_id":      productID.String(),
		"size":            strconv.FormatFloat(size, 'f', -1, 64),
	}
	var resp SuccessFailureConfig
	return &resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, req, true, &resp)
}

// PlaceOrder places either a limit, market, or stop order
func (c *CoinbasePro) PlaceOrder(ctx context.Context, ord *PlaceOrderInfo) (*SuccessFailureConfig, error) {
	if ord.ClientOID == "" {
		return nil, errClientOrderIDEmpty
	}
	if ord.ProductID == "" {
		return nil, errProductIDEmpty
	}
	if ord.BaseAmount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	orderConfig, err := createOrderConfig(ord.OrderType, ord.TimeInForce, ord.StopDirection, ord.BaseAmount, ord.QuoteAmount, ord.LimitPrice, ord.StopPrice, ord.BucketSize, ord.EndTime, ord.PostOnly, ord.BucketNumber, ord.BucketDuration)
	if err != nil {
		return nil, err
	}
	req := map[string]any{
		"client_order_id":              ord.ClientOID,
		"product_id":                   ord.ProductID,
		"side":                         ord.Side,
		"order_configuration":          orderConfig,
		"retail_portfolio_id":          ord.RetailPortfolioID,
		"preview_id":                   ord.PreviewID,
		"attached_order_configuration": ord.AttachedOrderConfiguration,
	}
	if ord.MarginType != "" {
		req["margin_type"] = FormatMarginType(ord.MarginType)
	}
	if ord.Leverage != 0 && ord.Leverage != 1 {
		req["leverage"] = strconv.FormatFloat(ord.Leverage, 'f', -1, 64)
	}
	var resp SuccessFailureConfig
	return &resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseV3+coinbaseOrders, nil, req, true, &resp)
}

// EditOrder edits an order to a new size or price. Only limit orders with a good-till-cancelled time in force can be edited
func (c *CoinbasePro) EditOrder(ctx context.Context, orderID string, size, price float64) (bool, error) {
	if orderID == "" {
		return false, errOrderIDEmpty
	}
	if size <= 0 && price <= 0 {
		return false, errSizeAndPriceZero
	}
	path := coinbaseV3 + coinbaseOrders + "/" + coinbaseEdit
	req := map[string]any{
		"order_id": orderID,
		"size":     strconv.FormatFloat(size, 'f', -1, 64),
		"price":    strconv.FormatFloat(price, 'f', -1, 64),
	}
	resp := struct {
		Success bool `json:"success"`
	}{}
	return resp.Success, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, req, true, &resp)
}

// EditOrderPreview simulates an edit order request, to preview the result. Only limit orders with a good-till-cancelled time in force can be edited.
func (c *CoinbasePro) EditOrderPreview(ctx context.Context, orderID string, size, price float64) (*EditOrderPreviewResp, error) {
	if orderID == "" {
		return nil, errOrderIDEmpty
	}
	if size <= 0 && price <= 0 {
		return nil, errSizeAndPriceZero
	}
	path := coinbaseV3 + coinbaseOrders + "/" + coinbaseEditPreview
	req := map[string]any{
		"order_id": orderID,
		"size":     strconv.FormatFloat(size, 'f', -1, 64),
		"price":    strconv.FormatFloat(price, 'f', -1, 64),
	}
	var resp *EditOrderPreviewResp
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, req, true, &resp)
}

// GetOrderByID returns a single order by order id.
func (c *CoinbasePro) GetOrderByID(ctx context.Context, orderID, clientOID string, userNativeCurrency currency.Code) (*GetOrderResponse, error) {
	if orderID == "" {
		return nil, errOrderIDEmpty
	}
	vals := url.Values{}
	if clientOID != "" {
		vals.Set("client_order_id", clientOID)
	}
	if !userNativeCurrency.IsEmpty() {
		vals.Set("user_native_currency", userNativeCurrency.String())
	}
	path := coinbaseV3 + coinbaseOrders + "/" + coinbaseHistorical + "/" + orderID
	resp := struct {
		Order GetOrderResponse `json:"order"`
	}{}
	return &resp.Order, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, vals, nil, true, &resp)
}

// ListFills returns information on recent fills
func (c *CoinbasePro) ListFills(ctx context.Context, orderIDs, tradeIDs []string, productIDs currency.Pairs, cursor, sortBy string, startDate, endDate time.Time, limit uint16) (*FillResponse, error) {
	var params Params
	params.Values = url.Values{}
	err := params.encodeDateRange(startDate, endDate, "start_sequence_timestamp", "end_sequence_timestamp")
	if err != nil {
		return nil, err
	}
	for i := range orderIDs {
		params.Values.Add("order_ids", orderIDs[i])
	}
	for i := range tradeIDs {
		params.Values.Add("trade_ids", tradeIDs[i])
	}
	for i := range productIDs {
		params.Values.Add("product_ids", productIDs[i].String())
	}
	params.Values.Set("limit", strconv.FormatInt(int64(limit), 10))
	params.Values.Set("cursor", cursor)
	if sortBy != "" {
		params.Values.Set("sort_by", sortBy)
	}
	path := coinbaseV3 + coinbaseOrders + "/" + coinbaseHistorical + "/" + coinbaseFills
	var resp FillResponse
	return &resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params.Values, nil, true, &resp)
}

// old signature ListOrders(ctx context.Context, productID, userNativeCurrency, orderType, orderSide, cursor, productType, orderPlacementSource, contractExpiryType, retailPortfolioID string, orderStatus, assetFilters []string, limit int32, startDate, endDate time.Time) (*ListOrdersResp, error) {

// ListOrders lists orders, filtered by their status
func (c *CoinbasePro) ListOrders(ctx context.Context, orderIDs, orderStatus, timeInForces, orderTypes, assetFilters []string, productIDs currency.Pairs, productType, orderSide, orderPlacementSource, contractExpiryType, retailPortfolioID, cursor, sortBy string, limit int32, startDate, endDate time.Time, userNativeCurrency currency.Code) (*ListOrdersResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.encodeDateRange(startDate, endDate, startDateString, endDateString)
	if err != nil {
		return nil, err
	}
	for x := range orderStatus {
		params.Values.Add("order_status", orderStatus[x])
	}
	for x := range orderIDs {
		params.Values.Add("order_ids", orderIDs[x])
	}
	for x := range timeInForces {
		params.Values.Add("time_in_forces", timeInForces[x])
	}
	for x := range orderTypes {
		params.Values.Add("order_types", orderTypes[x])
	}
	for x := range assetFilters {
		params.Values.Add("asset_filters", assetFilters[x])
	}
	for x := range productIDs {
		params.Values.Add("product_ids", productIDs[x].String())
	}
	if productType != "" {
		params.Values.Set("product_type", productType)
	}
	if orderSide != "" {
		params.Values.Set("order_side", orderSide)
	}
	if orderPlacementSource != "" {
		params.Values.Set("order_placement_source", orderPlacementSource)
	}
	if contractExpiryType != "" {
		params.Values.Set("contract_expiry_type", contractExpiryType)
	}
	if sortBy != "" {
		params.Values.Set("sort_by", sortBy)
	}
	params.Values.Set("cursor", cursor)
	params.Values.Set("limit", strconv.FormatInt(int64(limit), 10))
	params.Values.Set("user_native_currency", userNativeCurrency.String())
	// This functionality has been deprecated, and only works for legacy API keys
	params.Values.Set("retail_portfolio_id", retailPortfolioID)
	path := coinbaseV3 + coinbaseOrders + "/" + coinbaseHistorical + "/" + coinbaseBatch
	var resp ListOrdersResp
	return &resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params.Values, nil, true, &resp)
}

// GetBestBidAsk returns the best bid/ask for all products. Can be filtered to certain products by passing through additional strings
func (c *CoinbasePro) GetBestBidAsk(ctx context.Context, products []string) ([]ProductBook, error) {
	vals := url.Values{}
	for x := range products {
		vals.Add("product_ids", products[x])
	}
	resp := struct {
		Pricebooks []ProductBook `json:"pricebooks"`
	}{}
	return resp.Pricebooks, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseV3+coinbaseBestBidAsk, vals, nil, true, &resp)
}

// GetProductBookV3 returns a list of bids/asks for a single product
func (c *CoinbasePro) GetProductBookV3(ctx context.Context, productID currency.Pair, limit uint16, aggregationIncrement float64, authenticated bool) (*ProductBookResp, error) {
	if productID.IsEmpty() {
		return nil, errProductIDEmpty
	}
	vals := url.Values{}
	vals.Set("product_id", productID.String())
	if limit != 0 {
		vals.Set("limit", strconv.FormatInt(int64(limit), 10))
	}
	if aggregationIncrement != 0 {
		vals.Set("aggregation_price_increment", strconv.FormatFloat(aggregationIncrement, 'f', -1, 64))
	}
	var resp *ProductBookResp
	if authenticated {
		return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseV3+coinbaseProductBook, vals, nil, true, &resp)
	}
	path := coinbaseV3 + coinbaseMarket + "/" + coinbaseProductBook
	return resp, c.SendHTTPRequest(ctx, exchange.RestSpot, path, vals, &resp)
}

// GetAllProducts returns information on all currency pairs that are available for trading
func (c *CoinbasePro) GetAllProducts(ctx context.Context, limit, offset int32, productType, contractExpiryType, expiringContractStatus string, productIDs []string, authenticated bool) (*AllProducts, error) {
	vals := url.Values{}
	vals.Set("limit", strconv.FormatInt(int64(limit), 10))
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
		return &resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseV3+coinbaseProducts, vals, nil, true, &resp)
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
		return &resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, true, &resp)
	}
	path := coinbaseV3 + coinbaseMarket + "/" + coinbaseProducts + "/" + productID
	return &resp, c.SendHTTPRequest(ctx, exchange.RestSpot, path, nil, &resp)
}

// GetHistoricKlines returns historic candles for a product. Candles are returned in grouped buckets based on requested granularity. Requests that return more than 300 data points are rejected
func (c *CoinbasePro) GetHistoricKlines(ctx context.Context, productID, granularity string, startDate, endDate time.Time, authenticated bool) ([]Klines, error) {
	if productID == "" {
		return nil, errProductIDEmpty
	}
	if !common.StringSliceContains(allowedGranularities, granularity) {
		return nil, fmt.Errorf("%w %v, allowed granularities are: %+v", kline.ErrUnsupportedInterval, granularity, allowedGranularities)
	}
	vals := url.Values{}
	vals.Set("start", strconv.FormatInt(startDate.Unix(), 10))
	vals.Set("end", strconv.FormatInt(endDate.Unix(), 10))
	vals.Set("granularity", granularity)
	resp := struct {
		Candles []Klines `json:"candles"`
	}{}
	if authenticated {
		path := coinbaseV3 + coinbaseProducts + "/" + productID + "/" + coinbaseCandles
		return resp.Candles, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, vals, nil, true, &resp)
	}
	path := coinbaseV3 + coinbaseMarket + "/" + coinbaseProducts + "/" + productID + "/" + coinbaseCandles
	return resp.Candles, c.SendHTTPRequest(ctx, exchange.RestSpot, path, vals, &resp)
}

// GetTicker returns snapshot information about the last trades (ticks) and best bid/ask. Contrary to documentation, this does not tell you the 24h volume
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
		return &resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, vals, nil, true, &resp)
	}
	path := coinbaseV3 + coinbaseMarket + "/" + coinbaseProducts + "/" + productID + "/" + coinbaseTicker
	return &resp, c.SendHTTPRequest(ctx, exchange.RestSpot, path, vals, &resp)
}

// PreviewOrder simulates the results of an order request
func (c *CoinbasePro) PreviewOrder(ctx context.Context, inf *PreviewOrderInfo) (*PreviewOrderResp, error) {
	if inf.BaseAmount <= 0 && inf.QuoteAmount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	orderConfig, err := createOrderConfig(inf.OrderType, inf.TimeInForce, inf.StopDirection, inf.BaseAmount, inf.QuoteAmount, inf.LimitPrice, inf.StopPrice, 0, inf.EndTime, inf.PostOnly, 0, 0)
	if err != nil {
		return nil, err
	}
	req := map[string]any{
		"product_id":          inf.ProductID,
		"side":                inf.Side,
		"commission_rate":     map[string]string{"value": strconv.FormatFloat(inf.CommissionValue, 'f', -1, 64)},
		"order_configuration": orderConfig,
		"is_max":              inf.IsMax,
		"tradable_balance":    strconv.FormatFloat(inf.TradableBalance, 'f', -1, 64),
		"skip_fcm_risk_check": inf.SkipFCMRiskCheck,
		"leverage":            strconv.FormatFloat(inf.Leverage, 'f', -1, 64),
	}
	if mt := FormatMarginType(inf.MarginType); mt != "" {
		req["margin_type"] = mt
	}
	var resp *PreviewOrderResp
	path := coinbaseV3 + coinbaseOrders + "/" + coinbasePreview
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, req, true, &resp)
}

// GetAllPortfolios returns a list of portfolios associated with the user
func (c *CoinbasePro) GetAllPortfolios(ctx context.Context, portfolioType string) ([]SimplePortfolioData, error) {
	resp := struct {
		Portfolios []SimplePortfolioData `json:"portfolios"`
	}{}
	vals := url.Values{}
	if portfolioType != "" {
		vals.Set("portfolio_type", portfolioType)
	}
	return resp.Portfolios, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseV3+coinbasePortfolios, vals, nil, true, &resp)
}

// CreatePortfolio creates a new portfolio
func (c *CoinbasePro) CreatePortfolio(ctx context.Context, name string) (*SimplePortfolioData, error) {
	if name == "" {
		return nil, errNameEmpty
	}
	req := map[string]any{"name": name}
	resp := struct {
		Portfolio SimplePortfolioData `json:"portfolio"`
	}{}
	return &resp.Portfolio, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseV3+coinbasePortfolios, nil, req, true, &resp)
}

// MovePortfolioFunds transfers funds between portfolios
func (c *CoinbasePro) MovePortfolioFunds(ctx context.Context, cur, from, to string, amount float64) (*MovePortfolioFundsResponse, error) {
	if from == "" || to == "" {
		return nil, errPortfolioIDEmpty
	}
	if cur == "" {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	funds := FundsData{
		Value:    strconv.FormatFloat(amount, 'f', -1, 64),
		Currency: cur,
	}
	req := map[string]any{
		"source_portfolio_uuid": from,
		"target_portfolio_uuid": to,
		"funds":                 funds,
	}
	path := coinbaseV3 + coinbasePortfolios + "/" + coinbaseMoveFunds
	var resp *MovePortfolioFundsResponse
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, req, true, &resp)
}

// GetPortfolioByID provides detailed information on a single portfolio
func (c *CoinbasePro) GetPortfolioByID(ctx context.Context, portfolioID string) (*DetailedPortfolioResponse, error) {
	if portfolioID == "" {
		return nil, errPortfolioIDEmpty
	}
	path := coinbaseV3 + coinbasePortfolios + "/" + portfolioID
	resp := struct {
		Breakdown DetailedPortfolioResponse `json:"breakdown"`
	}{}
	return &resp.Breakdown, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, true, &resp)
}

// DeletePortfolio deletes a portfolio
func (c *CoinbasePro) DeletePortfolio(ctx context.Context, portfolioID string) error {
	if portfolioID == "" {
		return errPortfolioIDEmpty
	}
	path := coinbaseV3 + coinbasePortfolios + "/" + portfolioID
	return c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, nil, nil, true, nil)
}

// EditPortfolio edits the name of a portfolio
func (c *CoinbasePro) EditPortfolio(ctx context.Context, portfolioID, name string) (*SimplePortfolioData, error) {
	if portfolioID == "" {
		return nil, errPortfolioIDEmpty
	}
	if name == "" {
		return nil, errNameEmpty
	}
	req := map[string]any{"name": name}
	path := coinbaseV3 + coinbasePortfolios + "/" + portfolioID
	resp := struct {
		Portfolio SimplePortfolioData `json:"portfolio"`
	}{}
	return &resp.Portfolio, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPut, path, nil, req, true, &resp)
}

// AllocatePortfolio allocates funds to a position in your perpetuals portfolio
func (c *CoinbasePro) AllocatePortfolio(ctx context.Context, portfolioID, productID, cur string, amount float64) error {
	if portfolioID == "" {
		return errPortfolioIDEmpty
	}
	if productID == "" {
		return errProductIDEmpty
	}
	if cur == "" {
		return currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return order.ErrAmountIsInvalid
	}
	req := map[string]any{
		"portfolio_uuid": portfolioID,
		"symbol":         productID,
		"currency":       cur,
		"amount":         strconv.FormatFloat(amount, 'f', -1, 64),
	}
	path := coinbaseV3 + coinbaseIntx + "/" + coinbaseAllocate
	return c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, req, true, nil)
}

// GetPerpetualsPortfolioSummary returns a summary of your perpetuals portfolio
func (c *CoinbasePro) GetPerpetualsPortfolioSummary(ctx context.Context, portfolioID string) (*PerpetualsPortfolioSummary, error) {
	if portfolioID == "" {
		return nil, errPortfolioIDEmpty
	}
	path := coinbaseV3 + coinbaseIntx + "/" + coinbasePortfolio + "/" + portfolioID
	resp := struct {
		Summary PerpetualsPortfolioSummary `json:"summary"`
	}{}
	return &resp.Summary, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, true, &resp)
}

// GetAllPerpetualsPositions returns a list of all open positions in your perpetuals portfolio
func (c *CoinbasePro) GetAllPerpetualsPositions(ctx context.Context, portfolioID string) (*AllPerpPosResponse, error) {
	if portfolioID == "" {
		return nil, errPortfolioIDEmpty
	}
	path := coinbaseV3 + coinbaseIntx + "/" + coinbasePositions + "/" + portfolioID
	var resp *AllPerpPosResponse
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, true, &resp)
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
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, true, &resp)
}

// GetV3Time returns the current server time, calling V3 of the API
func (c *CoinbasePro) GetV3Time(ctx context.Context) (*ServerTimeV3, error) {
	var resp *ServerTimeV3
	return resp, c.SendHTTPRequest(ctx, exchange.RestSpot, coinbaseV3+coinbaseTime, nil, &resp)
}

// ListPaymentMethods returns a list of all payment methods associated with the user's account
func (c *CoinbasePro) ListPaymentMethods(ctx context.Context) ([]PaymentMethodData, error) {
	resp := struct {
		PaymentMethods []PaymentMethodData `json:"payment_methods"`
	}{}
	req := map[string]any{"currency": "BTC"}
	path := coinbaseV3 + coinbasePaymentMethods
	return resp.PaymentMethods, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, req, true, &resp)
}

// GetPaymentMethodByID returns information on a single payment method associated with the user's account
func (c *CoinbasePro) GetPaymentMethodByID(ctx context.Context, paymentMethodID string) (*PaymentMethodData, error) {
	if paymentMethodID == "" {
		return nil, errPaymentMethodEmpty
	}
	path := coinbaseV3 + coinbasePaymentMethods + "/" + paymentMethodID
	resp := struct {
		PaymentMethod PaymentMethodData `json:"payment_method"`
	}{}
	return &resp.PaymentMethod, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, true, &resp)
}

// GetCurrentUser returns information about the user associated with the API key
func (c *CoinbasePro) GetCurrentUser(ctx context.Context) (*UserResponse, error) {
	resp := struct {
		Data UserResponse `json:"data"`
	}{}
	return &resp.Data, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseV2+coinbaseUser, nil, nil, false, &resp)
}

// GetAllWallets lists all accounts associated with the API key
func (c *CoinbasePro) GetAllWallets(ctx context.Context, pag PaginationInp) (*GetAllWalletsResponse, error) {
	var resp *GetAllWalletsResponse
	var params Params
	params.Values = url.Values{}
	if err := params.encodePagination(pag); err != nil {
		return nil, err
	}
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseV2+coinbaseAccounts, params.Values, nil, false, &resp)
}

// GetWalletByID returns information about a single wallet. In lieu of a wallet ID, a currency can be provided to get the primary account for that currency
func (c *CoinbasePro) GetWalletByID(ctx context.Context, walletID, currency string) (*WalletData, error) {
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
	resp := struct {
		Data WalletData `json:"data"`
	}{}
	return &resp.Data, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, false, &resp)
}

// CreateAddress generates a crypto address for depositing to the specified wallet
func (c *CoinbasePro) CreateAddress(ctx context.Context, walletID, name string) (*AddressData, error) {
	if walletID == "" {
		return nil, errWalletIDEmpty
	}
	path := coinbaseV2 + coinbaseAccounts + "/" + walletID + "/" + coinbaseAddresses
	req := map[string]any{"name": name}
	resp := struct {
		Data AddressData `json:"data"`
	}{}
	return &resp.Data, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, req, false, &resp)
}

// GetAllAddresses returns information on all addresses associated with a wallet
func (c *CoinbasePro) GetAllAddresses(ctx context.Context, walletID string, pag PaginationInp) (*GetAllAddrResponse, error) {
	if walletID == "" {
		return nil, errWalletIDEmpty
	}
	path := coinbaseV2 + coinbaseAccounts + "/" + walletID + "/" + coinbaseAddresses
	var params Params
	params.Values = url.Values{}
	if err := params.encodePagination(pag); err != nil {
		return nil, err
	}
	var resp *GetAllAddrResponse
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params.Values, nil, false, &resp)
}

// GetAddressByID returns information on a single address associated with the specified wallet
func (c *CoinbasePro) GetAddressByID(ctx context.Context, walletID, addressID string) (*AddressData, error) {
	if walletID == "" {
		return nil, errWalletIDEmpty
	}
	if addressID == "" {
		return nil, errAddressIDEmpty
	}
	path := coinbaseV2 + coinbaseAccounts + "/" + walletID + "/" + coinbaseAddresses + "/" + addressID
	resp := struct {
		Data AddressData `json:"data"`
	}{}
	return &resp.Data, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, false, &resp)
}

// GetAddressTransactions returns a list of transactions associated with the specified address
func (c *CoinbasePro) GetAddressTransactions(ctx context.Context, walletID, addressID string, pag PaginationInp) (*ManyTransactionsResp, error) {
	if walletID == "" {
		return nil, errWalletIDEmpty
	}
	if addressID == "" {
		return nil, errAddressIDEmpty
	}
	path := coinbaseV2 + coinbaseAccounts + "/" + walletID + "/" + coinbaseAddresses + "/" + addressID + "/" + coinbaseTransactions
	var params Params
	params.Values = url.Values{}
	if err := params.encodePagination(pag); err != nil {
		return nil, err
	}
	var resp *ManyTransactionsResp
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params.Values, nil, false, &resp)
}

// SendMoney can send funds to an email or cryptocurrency address (if "traType" is set to "send"), or to another one of the user's wallets or vaults (if "traType" is set to "transfer"). Coinbase may delay or cancel the transaction at their discretion. The "idem" parameter is an optional string for idempotency; a token with a max length of 100 characters, if a previous transaction included the same token as a parameter, the new transaction won't be processed, and information on the previous transaction will be returned instead
func (c *CoinbasePro) SendMoney(ctx context.Context, traType, walletID, to, cur, description, idem, financialInstitutionWebsite, destinationTag string, amount float64, skipNotifications, toFinancialInstitution bool) (*TransactionData, error) {
	if traType == "" {
		return nil, errTransactionTypeEmpty
	}
	if walletID == "" {
		return nil, errWalletIDEmpty
	}
	if to == "" {
		return nil, errToEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if cur == "" {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	path := coinbaseV2 + coinbaseAccounts + "/" + walletID + "/" + coinbaseTransactions
	req := map[string]any{
		"type":                          traType,
		"to":                            to,
		"amount":                        strconv.FormatFloat(amount, 'f', -1, 64),
		"currency":                      cur,
		"description":                   description,
		"skip_notifications":            skipNotifications,
		"idem":                          idem,
		"to_financial_institution":      toFinancialInstitution,
		"financial_institution_website": financialInstitutionWebsite,
		"destination_tag":               destinationTag,
	}
	resp := struct {
		Data TransactionData `json:"data"`
	}{}
	return &resp.Data, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, req, false, &resp)
}

// GetAllTransactions returns a list of transactions associated with the specified wallet
func (c *CoinbasePro) GetAllTransactions(ctx context.Context, walletID string, pag PaginationInp) (*ManyTransactionsResp, error) {
	if walletID == "" {
		return nil, errWalletIDEmpty
	}
	path := coinbaseV2 + coinbaseAccounts + "/" + walletID + "/" + coinbaseTransactions
	var params Params
	params.Values = url.Values{}
	if err := params.encodePagination(pag); err != nil {
		return nil, err
	}
	var resp *ManyTransactionsResp
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params.Values, nil, false, &resp)
}

// GetTransactionByID returns information on a single transaction associated with the specified wallet
func (c *CoinbasePro) GetTransactionByID(ctx context.Context, walletID, transactionID string) (*TransactionData, error) {
	if walletID == "" {
		return nil, errWalletIDEmpty
	}
	if transactionID == "" {
		return nil, errTransactionIDEmpty
	}
	path := coinbaseV2 + coinbaseAccounts + "/" + walletID + "/" + coinbaseTransactions + "/" + transactionID
	resp := struct {
		Data TransactionData `json:"data"`
	}{}
	return &resp.Data, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, false, &resp)
}

// FiatTransfer prepares and optionally processes a transfer of funds between the exchange and a fiat payment method. "Deposit" signifies funds going from exchange to bank, "withdraw" signifies funds going from bank to exchange
func (c *CoinbasePro) FiatTransfer(ctx context.Context, walletID, cur, paymentMethod string, amount float64, commit bool, transferType FiatTransferType) (*DeposWithdrData, error) {
	if walletID == "" {
		return nil, errWalletIDEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if cur == "" {
		return nil, currency.ErrCurrencyCodeEmpty
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
	req := map[string]any{
		"currency":       cur,
		"payment_method": paymentMethod,
		"amount":         strconv.FormatFloat(amount, 'f', -1, 64),
		"commit":         commit,
	}
	resp := struct {
		Data DeposWithdrData `json:"data"`
	}{}
	return &resp.Data, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, req, false, &resp)
}

// CommitTransfer processes a deposit/withdrawal that was created with the "commit" parameter set to false
func (c *CoinbasePro) CommitTransfer(ctx context.Context, walletID, depositID string, transferType FiatTransferType) (*DeposWithdrData, error) {
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
	resp := struct {
		Data DeposWithdrData `json:"data"`
	}{}
	return &resp.Data, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, nil, false, &resp)
}

// GetAllFiatTransfers returns a list of transfers either to or from fiat payment methods and the specified wallet
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
	if err := params.encodePagination(pag); err != nil {
		return nil, err
	}
	var resp *ManyDeposWithdrResp
	err := c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params.Values, nil, false, &resp)
	if err != nil {
		return nil, err
	}
	for i := range resp.Data {
		resp.Data[i].TransferType = transferType
	}
	return resp, nil
}

// GetFiatTransferByID returns information on a single deposit/withdrawal associated with the specified wallet
func (c *CoinbasePro) GetFiatTransferByID(ctx context.Context, walletID, depositID string, transferType FiatTransferType) (*DeposWithdrData, error) {
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
	resp := struct {
		Data DeposWithdrData `json:"data"`
	}{}
	return &resp.Data, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, false, &resp)
}

// GetFiatCurrencies lists currencies that Coinbase knows about
func (c *CoinbasePro) GetFiatCurrencies(ctx context.Context) ([]FiatData, error) {
	resp := struct {
		Data []FiatData `json:"data"`
	}{}
	return resp.Data, c.SendHTTPRequest(ctx, exchange.RestSpot, coinbaseV2+coinbaseCurrencies, nil, &resp)
}

// GetCryptocurrencies lists cryptocurrencies that Coinbase knows about
func (c *CoinbasePro) GetCryptocurrencies(ctx context.Context) ([]CryptoData, error) {
	resp := struct {
		Data []CryptoData `json:"data"`
	}{}
	path := coinbaseV2 + coinbaseCurrencies + "/" + coinbaseCrypto
	return resp.Data, c.SendHTTPRequest(ctx, exchange.RestSpot, path, nil, &resp)
}

// GetExchangeRates returns exchange rates for the specified currency. If none is specified, it defaults to USD
func (c *CoinbasePro) GetExchangeRates(ctx context.Context, currency string) (*GetExchangeRatesResp, error) {
	resp := struct {
		Data GetExchangeRatesResp `json:"data"`
	}{}
	vals := url.Values{}
	vals.Set("currency", currency)
	return &resp.Data, c.SendHTTPRequest(ctx, exchange.RestSpot, coinbaseV2+coinbaseExchangeRates, vals, &resp)
}

// GetPrice returns the price the spot/buy/sell price for the specified currency pair, including the standard Coinbase fee of 1%, but excluding any other fees
func (c *CoinbasePro) GetPrice(ctx context.Context, currencyPair, priceType string) (*GetPriceResp, error) {
	var path string
	switch priceType {
	case "spot", "buy", "sell":
		path = coinbaseV2 + coinbasePrices + "/" + currencyPair + "/" + priceType
	default:
		return nil, errInvalidPriceType
	}
	resp := struct {
		Data GetPriceResp `json:"data"`
	}{}
	return &resp.Data, c.SendHTTPRequest(ctx, exchange.RestSpot, path, nil, &resp)
}

// GetV2Time returns the current server time, calling V2 of the API
func (c *CoinbasePro) GetV2Time(ctx context.Context) (*ServerTimeV2, error) {
	resp := struct {
		Data ServerTimeV2 `json:"data"`
	}{}
	return &resp.Data, c.SendHTTPRequest(ctx, exchange.RestSpot, coinbaseV2+coinbaseTime, nil, &resp)
}

// GetAllCurrencies returns a list of all currencies that Coinbase knows about. These aren't necessarily tradable
func (c *CoinbasePro) GetAllCurrencies(ctx context.Context) ([]CurrencyData, error) {
	var resp []CurrencyData
	return resp, c.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, coinbaseCurrencies, nil, &resp)
}

// GetACurrency returns information on a single currency specified by the user
func (c *CoinbasePro) GetACurrency(ctx context.Context, cur string) (*CurrencyData, error) {
	if cur == "" {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *CurrencyData
	path := coinbaseCurrencies + "/" + cur
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
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp *PairData
	path := coinbaseProducts + "/" + pair
	return resp, c.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, nil, &resp)
}

// GetProductBookV1 returns the order book for the specified currency pair. Level 1 only returns the best bids and asks, Level 2 returns the full order book with orders at the same price aggregated, Level 3 returns the full non-aggregated order book.
func (c *CoinbasePro) GetProductBookV1(ctx context.Context, pair string, level uint8) (*OrderBookResp, error) {
	if pair == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp *OrderBookResp
	vals := url.Values{}
	vals.Set("level", strconv.FormatUint(uint64(level), 10))
	path := coinbaseProducts + "/" + pair + "/" + coinbaseBook
	return resp, c.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, vals, &resp)
}

// GetProductCandles returns historical market data for the specified currency pair.
func (c *CoinbasePro) GetProductCandles(ctx context.Context, pair string, granularity uint32, startTime, endTime time.Time) ([]Candle, error) {
	if pair == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var params Params
	params.Values = url.Values{}
	err := params.encodeDateRange(startTime, endTime, "start", "end")
	if err != nil {
		return nil, err
	}
	if granularity != 0 {
		params.Values.Set("granularity", strconv.FormatUint(uint64(granularity), 10))
	}
	path := coinbaseProducts + "/" + pair + "/" + coinbaseCandles
	var resp []Candle
	return resp, c.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, params.Values, &resp)
}

// GetProductStats returns information on a specific pair's price and volume
func (c *CoinbasePro) GetProductStats(ctx context.Context, pair string) (*ProductStats, error) {
	if pair == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	path := coinbaseProducts + "/" + pair + "/" + coinbaseStats
	var resp *ProductStats
	return resp, c.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, nil, &resp)
}

// GetProductTicker returns the ticker for the specified currency pair
func (c *CoinbasePro) GetProductTicker(ctx context.Context, pair string) (*ProductTicker, error) {
	if pair == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	path := coinbaseProducts + "/" + pair + "/" + coinbaseTicker
	var resp *ProductTicker
	return resp, c.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, nil, &resp)
}

// GetProductTrades returns a list of the latest traides for a pair
func (c *CoinbasePro) GetProductTrades(ctx context.Context, pair, step, direction string, limit int64) ([]ProductTrades, error) {
	if pair == "" {
		return nil, currency.ErrCurrencyPairEmpty
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
func (c *CoinbasePro) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, vals url.Values, result any) error {
	endpoint, err := c.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	rLim := PubRate
	if strings.Contains(path, coinbaseV2) {
		rLim = V2Rate
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
	return c.SendPayload(ctx, rLim, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request
func (c *CoinbasePro) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, method, path string, queryParams url.Values, bodyParams map[string]any, isVersion3 bool, result any) (err error) {
	endpoint, err := c.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	if len(endpoint) < 8 {
		return errEndpointPathInvalid
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
		var jwt string
		jwt, _, err = c.GetJWT(ctx, method+" "+endpoint[8:]+path)
		if err != nil {
			return nil, err
		}
		headers := make(map[string]string)
		headers["Content-Type"] = "application/json"
		headers["CB-VERSION"] = "2025-03-26"
		headers["Authorization"] = "Bearer " + jwt
		return &request.Item{
			Method:        method,
			Path:          endpoint + common.EncodeURLValues(path, queryParams),
			Headers:       headers,
			Body:          bytes.NewBuffer(payload),
			Result:        &interim,
			Verbose:       c.Verbose,
			HTTPDebugging: c.HTTPDebugging,
			HTTPRecording: c.HTTPRecording,
		}, nil
	}
	rateLim := V2Rate
	if isVersion3 {
		rateLim = V3Rate
	}
	err = c.SendPayload(ctx, rateLim, newRequest, request.AuthenticatedRequest)
	if err != nil {
		return err
	}
	// Doing this error handling because the docs indicate that errors can be returned even with a 200 status code, and that these errors can be buried in the JSON returned
	singleErrCap := struct {
		ErrorType             string `json:"error"`
		Message               string `json:"message"`
		ErrorDetails          string `json:"error_details"`
		EditFailureReason     string `json:"edit_failure_reason"`
		PreviewFailureReason  string `json:"preview_failure_reason"`
		NewOrderFailureReason string `json:"new_order_failure_reason"`
	}{}
	if err = json.Unmarshal(interim, &singleErrCap); err == nil {
		if singleErrCap.ErrorType != "" {
			return fmt.Errorf("message: %s, error type: %s, error details: %s, edit failure reason: %s, preview failure reason: %s, new order failure reason: %s", singleErrCap.Message, singleErrCap.ErrorType, singleErrCap.ErrorDetails, singleErrCap.EditFailureReason, singleErrCap.PreviewFailureReason, singleErrCap.NewOrderFailureReason)
		}
	}
	manyErrCap := struct {
		Results []ManyErrors `json:"results"`
		Errors  []ManyErrors `json:"errors"`
	}{}
	err = json.Unmarshal(interim, &manyErrCap)
	if err == nil {
		errMessage := ""
		for i := range manyErrCap.Errors {
			if !manyErrCap.Errors[i].Success && (manyErrCap.Errors[i].EditFailureReason != "" || manyErrCap.Errors[i].PreviewFailureReason != "") {
				errMessage += fmt.Sprintf("order id: %s, failure reason: %s, edit failure reason: %s, preview failure reason: %s", manyErrCap.Errors[i].OrderID, manyErrCap.Errors[i].FailureReason, manyErrCap.Errors[i].EditFailureReason, manyErrCap.Errors[i].PreviewFailureReason)
			}
		}
		for i := range manyErrCap.Results {
			if !manyErrCap.Results[i].Success && (manyErrCap.Results[i].EditFailureReason != "" || manyErrCap.Results[i].PreviewFailureReason != "") {
				errMessage += fmt.Sprintf("order id: %s, failure reason: %s, edit failure reason: %s, preview failure reason: %s", manyErrCap.Results[i].OrderID, manyErrCap.Results[i].FailureReason, manyErrCap.Results[i].EditFailureReason, manyErrCap.Results[i].PreviewFailureReason)
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

// GetJWT generates a new JWT
func (c *CoinbasePro) GetJWT(ctx context.Context, uri string) (string, time.Time, error) {
	creds, err := c.GetCredentials(ctx)
	if err != nil {
		return "", time.Time{}, err
	}
	block, _ := pem.Decode([]byte(creds.Secret))
	if block == nil {
		return "", time.Time{}, errCantDecodePrivKey
	}
	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return "", time.Time{}, err
	}
	nonce, err := common.GenerateRandomString(16, "1234567890ABCDEF")
	if err != nil {
		return "", time.Time{}, err
	}
	regTime := time.Now()
	mapClaims := jwt.MapClaims{
		"iss": "cdp",
		"nbf": regTime.Unix(),
		"exp": regTime.Add(time.Minute * 2).Unix(),
		"sub": creds.Key,
	}
	if uri != "" {
		mapClaims["uri"] = uri
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, mapClaims)
	tok.Header["kid"] = creds.Key
	tok.Header["nonce"] = nonce
	sign, err := tok.SignedString(key)
	return sign, regTime.Add(time.Minute * 2), err
	// The code below mostly works, but seems to lead to bad results on the signature step. Deferring until later
	// head := map[string]any{"kid": creds.Key, "typ": "JWT", "alg": "ES256", "nonce": nonce}
	// headJSON, err := json.Marshal(head)
	// if err != nil {
	// 	return "", time.Time{}, err
	// }
	// headEncode := base64URLEncode(headJSON)
	// regTime := time.Now()
	// body := map[string]any{"iss": "cdp", "nbf": regTime.Unix(), "exp": regTime.Add(time.Minute * 2).Unix(), "sub": creds.Key /*, "aud": "retail_rest_api_proxy"*/}
	// if uri != "" {
	// 	body["uri"] = uri
	// }
	// bodyJSON, err := json.Marshal(body)
	// if err != nil {
	// 	return "", time.Time{}, err
	// }
	// bodyEncode := base64URLEncode(bodyJSON)
	// hash := sha256.Sum256([]byte(headEncode + "." + bodyEncode))
	// sig, err := ecdsa.SignASN1(rand.Reader, key, hash[:])
	// if err != nil {
	// 	return "", time.Time{}, err
	// }
	// sigEncode := base64URLEncode(sig)
	// return headEncode + "." + bodyEncode + "." + sigEncode, regTime.Add(time.Minute * 2), nil
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
			fee = fees.FeeTier.MakerFeeRate.Float64()
		} else {
			fee = fees.FeeTier.TakerFeeRate.Float64()
		}
	case feeBuilder.IsMaker && isStablePair(feeBuilder.Pair) && (feeBuilder.FeeType == exchange.CryptocurrencyTradeFee || feeBuilder.FeeType == exchange.OfflineTradeFee):
		fee = StablePairMakerFee
	case !feeBuilder.IsMaker && isStablePair(feeBuilder.Pair) && (feeBuilder.FeeType == exchange.CryptocurrencyTradeFee || feeBuilder.FeeType == exchange.OfflineTradeFee):
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

// encodeDateRange encodes a set of parameters indicating start & end dates
func (p *Params) encodeDateRange(startDate, endDate time.Time, labelStart, labelEnd string) error {
	if err := common.StartEndTimeCheck(startDate, endDate); err != nil {
		if errors.Is(err, common.ErrDateUnset) {
			return nil
		}
		return err
	}
	if labelStart == "" || labelEnd == "" {
		return errDateLabelEmpty
	}
	if p.Values == nil {
		return errParamValuesNil
	}
	p.Values.Set(labelStart, startDate.Format(time.RFC3339))
	p.Values.Set(labelEnd, endDate.Format(time.RFC3339))
	return nil
}

// encodePagination formats pagination information in the way the exchange expects
func (p *Params) encodePagination(pag PaginationInp) error {
	if p.Values == nil {
		return errParamValuesNil
	}
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
	return nil
}

// marketIOCImplementation creates a MarketMarketIOC struct based on the provided base and quote amounts
func marketIOCImplementation(baseAmount, quoteAmount float64) *MarketMarketIOC {
	if baseAmount != 0 {
		return &MarketMarketIOC{BaseSize: types.Number(baseAmount)}
	}
	if quoteAmount != 0 {
		return &MarketMarketIOC{QuoteSize: types.Number(quoteAmount)}
	}
	return nil
}

// createOrderConfig populates the OrderConfiguration struct
func createOrderConfig(orderType order.Type, timeInForce order.TimeInForce, stopDirection string, baseAmount, quoteAmount, limitPrice, stopPrice, bucketSize float64, endTime time.Time, postOnly bool, bucketNumber int64, bucketDuration time.Duration) (OrderConfiguration, error) {
	var orderConfig OrderConfiguration
	switch orderType {
	case order.Market:
		orderConfig.MarketMarketIOC = marketIOCImplementation(baseAmount, quoteAmount)
	case order.Limit:
		switch {
		case timeInForce == order.StopOrReduce:
			orderConfig.SORLimitIOC = &QuoteBaseLimit{BaseSize: types.Number(baseAmount), QuoteSize: types.Number(quoteAmount), LimitPrice: types.Number(limitPrice)}
		case endTime.IsZero():
			orderConfig.LimitLimitGTC = &LimitLimitGTC{LimitPrice: types.Number(limitPrice), PostOnly: postOnly}
			if baseAmount != 0 {
				orderConfig.LimitLimitGTC.BaseSize = types.Number(baseAmount)
			}
			if quoteAmount != 0 {
				orderConfig.LimitLimitGTC.QuoteSize = types.Number(quoteAmount)
			}
		case timeInForce == order.FillOrKill:
			orderConfig.LimitLimitFOK = &QuoteBaseLimit{BaseSize: types.Number(baseAmount), QuoteSize: types.Number(quoteAmount), LimitPrice: types.Number(limitPrice)}
		default:
			if endTime.Before(time.Now()) {
				return orderConfig, errEndTimeInPast
			}
			orderConfig.LimitLimitGTD = &LimitLimitGTD{LimitPrice: types.Number(limitPrice), PostOnly: postOnly, EndTime: endTime}
			if baseAmount != 0 {
				orderConfig.LimitLimitGTD.BaseSize = types.Number(baseAmount)
			}
			if quoteAmount != 0 {
				orderConfig.LimitLimitGTD.QuoteSize = types.Number(quoteAmount)
			}
		}
	case order.TWAP:
		if endTime.Before(time.Now()) {
			return orderConfig, errEndTimeInPast
		}
		orderConfig.TWAPLimitGTD = &TWAPLimitGTD{StartTime: time.Now(), EndTime: endTime, LimitPrice: types.Number(limitPrice), NumberBuckets: bucketNumber, BucketSize: types.Number(bucketSize), BucketDuration: bucketDuration}
	case order.StopLimit:
		if endTime.IsZero() {
			orderConfig.StopLimitStopLimitGTC = &StopLimitStopLimitGTC{LimitPrice: types.Number(limitPrice), StopPrice: types.Number(stopPrice), StopDirection: stopDirection}
			if baseAmount != 0 {
				orderConfig.StopLimitStopLimitGTC.BaseSize = types.Number(baseAmount)
			}
			if quoteAmount != 0 {
				orderConfig.StopLimitStopLimitGTC.QuoteSize = types.Number(quoteAmount)
			}
		} else {
			if endTime.Before(time.Now()) {
				return orderConfig, errEndTimeInPast
			}
			orderConfig.StopLimitStopLimitGTD = &StopLimitStopLimitGTD{LimitPrice: types.Number(limitPrice), StopPrice: types.Number(stopPrice), StopDirection: stopDirection, EndTime: endTime}
			if baseAmount != 0 {
				orderConfig.StopLimitStopLimitGTD.BaseSize = types.Number(baseAmount)
			}
			if quoteAmount != 0 {
				orderConfig.StopLimitStopLimitGTD.QuoteSize = types.Number(quoteAmount)
			}
		}
	case order.Bracket:
		if endTime.IsZero() {
			orderConfig.TriggerBracketGTC = &TriggerBracketGTC{BaseSize: types.Number(baseAmount), LimitPrice: types.Number(limitPrice), StopTriggerPrice: types.Number(stopPrice)}
		} else {
			if endTime.Before(time.Now()) {
				return orderConfig, errEndTimeInPast
			}
			orderConfig.TriggerBracketGTD = &TriggerBracketGTD{BaseSize: types.Number(baseAmount), LimitPrice: types.Number(limitPrice), StopTriggerPrice: types.Number(stopPrice), EndTime: endTime}
		}
	default:
		return orderConfig, errInvalidOrderType
	}
	return orderConfig, nil
}

// FormatMarginType properly formats the margin type for the request
func FormatMarginType(marginType string) string {
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

// UnmarshalJSON unmarshals the JSON data
func (o *Orders) UnmarshalJSON(data []byte) error {
	var alias any
	temp := [3]any{&o.Price, &o.Size, &alias}
	err := json.Unmarshal(data, &temp)
	if err != nil {
		return err
	}
	switch a := alias.(type) {
	case string:
		o.OrderID, err = uuid.FromString(a)
		if err != nil {
			return err
		}
		o.OrderCount = 1
	case float64:
		o.OrderCount = uint64(a)
	default:
		return common.ErrTypeAssertFailure
	}
	return nil
}

// UnmarshalJSON unmarshals the JSON data
func (c *Candle) UnmarshalJSON(data []byte) error {
	temp := [6]any{&c.Time, &c.Low, &c.High, &c.Open, &c.Close, &c.Volume}
	return json.Unmarshal(data, &temp)
}
