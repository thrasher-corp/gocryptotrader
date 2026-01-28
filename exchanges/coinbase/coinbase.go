package coinbase

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"
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
	apiURL                   = "https://api.coinbase.com"
	v1APIURL                 = "https://api.exchange.coinbase.com/"
	sandboxAPIURL            = "https://api-sandbox.coinbase.com"
	tradeBaseURL             = "https://www.coinbase.com/advanced-trade/spot/"
	v3Path                   = "/api/v3/brokerage/"
	accountsPath             = "accounts"
	convertPath              = "convert"
	tradePath                = "trade"
	quotePath                = "quote"
	keyPermissionsPath       = "key_permissions"
	transactionSummaryPath   = "transaction_summary"
	futuresPath              = "cfm" // Coinbase Financial Markets is the legal name for the Coinbase futures company
	sweepsPath               = "sweeps"
	intradayPath             = "intraday"
	currentMarginWindowPath  = "current_margin_window"
	balanceSummaryPath       = "balance_summary"
	positionsPath            = "positions"
	marginSettingPath        = "margin_setting"
	schedulePath             = "schedule"
	ordersPath               = "orders"
	batchCancelpath          = "batch_cancel"
	closePositionPath        = "close_position"
	editPath                 = "edit"
	editPreviewPath          = "edit_preview"
	historicalPath           = "historical"
	fillsPath                = "fills"
	batchPath                = "batch"
	bestBidAskPath           = "best_bid_ask"
	productBookPath          = "product_book"
	productsPath             = "products"
	candlesPath              = "candles"
	tickerPath               = "ticker"
	previewPath              = "preview"
	portfoliosPath           = "portfolios"
	moveFundsPath            = "move_funds"
	intxPath                 = "intx"
	balancesPath             = "balances"
	multiAssetCollateralPath = "multi_asset_collateral"
	allocatePath             = "allocate"
	portfolioPath            = "portfolio"
	paymentMethodsPath       = "payment_methods"
	v2Path                   = "/v2/"
	userPath                 = "user"
	addressesPath            = "addresses"
	transactionsPath         = "transactions"
	depositsPath             = "deposits"
	commitPath               = "commit"
	withdrawalsPath          = "withdrawals"
	currenciesPath           = "currencies"
	cryptoPath               = "crypto"
	exchangeRatesPath        = "exchange-rates"
	pricesPath               = "prices"
	timePath                 = "time"
	volumeSummaryPath        = "volume-summary"
	bookPath                 = "book"
	statsPath                = "stats"
	tradesPath               = "trades"
	wrappedAssetsPath        = "wrapped-assets"
	conversionRatePath       = "conversion-rate"
	marketPath               = "market"

	startDateString = "start_date"
	endDateString   = "end_date"

	defaultOrderFillCount = 3000       // Largest number of fills the exchange will let one retrieve in a request, found through experimentation
	defaultOrderCount     = 2147483647 // int32 limit, largest number of orders the exchange will let one retrieve in a request, found through experimentation
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
	errProductIDEmpty           = errors.New("product id cannot be empty")
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
	errDecodingPrivateKey       = errors.New("error decoding private key")
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
	errMarginProfileTypeEmpty   = errors.New("margin profile type cannot be empty")
	errSettingEmpty             = errors.New("setting cannot be empty")
	errUnknownTransferType      = errors.New("unknown transfer type")
	errOutOfSequence            = errors.New("out of order sequence number")

	closedStatuses = []string{"FILLED", "CANCELLED", "EXPIRED", "FAILED"}
	openStatus     = []string{"OPEN"}

	allowedGranularities = map[kline.Interval]string{
		kline.OneMin:     "ONE_MINUTE",
		kline.FiveMin:    "FIVE_MINUTE",
		kline.FifteenMin: "FIFTEEN_MINUTE",
		kline.ThirtyMin:  "THIRTY_MINUTE",
		kline.OneHour:    "ONE_HOUR",
		kline.TwoHour:    "TWO_HOUR",
		kline.SixHour:    "SIX_HOUR",
		kline.OneDay:     "ONE_DAY",
	}
)

// GetAccountByID returns information for a single account
func (e *Exchange) GetAccountByID(ctx context.Context, accountID string) (*Account, error) {
	if accountID == "" {
		return nil, errAccountIDEmpty
	}
	path := v3Path + accountsPath + "/" + accountID
	resp := struct {
		Account Account `json:"account"`
	}{}
	return &resp.Account, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, true, &resp)
}

// ListAccounts returns information on all trading accounts associated with the API key
func (e *Exchange) ListAccounts(ctx context.Context, limit uint8, cursor int64) (*AllAccountsResponse, error) {
	vals := url.Values{}
	if limit != 0 {
		vals.Set("limit", strconv.FormatUint(uint64(limit), 10))
	}
	if cursor != 0 {
		vals.Set("cursor", strconv.FormatInt(cursor, 10))
	}
	var resp *AllAccountsResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, v3Path+accountsPath, vals, nil, true, &resp)
}

// CommitConvertTrade commits a conversion between two currencies, using the trade_id returned from CreateConvertQuote
func (e *Exchange) CommitConvertTrade(ctx context.Context, tradeID, from, to string) (*ConvertResponse, error) {
	if tradeID == "" {
		return nil, errTransactionIDEmpty
	}
	if from == "" || to == "" {
		return nil, errAccountIDEmpty
	}
	path := v3Path + convertPath + "/" + tradePath + "/" + tradeID
	var resp ConvertWrapper
	return &resp.Trade, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, convertTradeReqBase{FromAccount: from, ToAccount: to}, true, &resp)
}

// CreateConvertQuote creates a quote for a conversion between two currencies. The trade_id returned can be used to commit the trade, but that must be done within 10 minutes of the quote's creation
func (e *Exchange) CreateConvertQuote(ctx context.Context, from, to, userIncentiveID, codeVal string, amount float64) (*ConvertResponse, error) {
	if from == "" || to == "" {
		return nil, errAccountIDEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	path := v3Path + convertPath + "/" + quotePath
	req := convertQuoteReqBase{
		FromAccount: from,
		ToAccount:   to,
		Amount:      amount,
		Metadata: tradeIncentiveMetadata{
			UserIncentiveID: userIncentiveID,
			CodeVal:         codeVal,
		},
	}
	var resp ConvertWrapper
	return &resp.Trade, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, req, true, &resp)
}

// GetConvertTradeByID returns information on a conversion between two currencies
func (e *Exchange) GetConvertTradeByID(ctx context.Context, tradeID, from, to string) (*ConvertResponse, error) {
	if tradeID == "" {
		return nil, errTransactionIDEmpty
	}
	if from == "" || to == "" {
		return nil, errAccountIDEmpty
	}
	path := v3Path + convertPath + "/" + tradePath + "/" + tradeID
	var resp ConvertWrapper
	return &resp.Trade, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, convertTradeReqBase{FromAccount: from, ToAccount: to}, true, &resp)
}

// GetPermissions returns the permissions associated with the API key
func (e *Exchange) GetPermissions(ctx context.Context) (*PermissionsResponse, error) {
	var resp PermissionsResponse
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, v3Path+keyPermissionsPath, nil, nil, true, &resp)
}

// GetTransactionSummary returns a summary of transactions with fee tiers, total volume, and fees
func (e *Exchange) GetTransactionSummary(ctx context.Context, startDate, endDate time.Time, productVenue, productType, contractExpiryType string) (*TransactionSummary, error) {
	vals, err := urlValsFromDateRange(startDate, endDate, startDateString, endDateString)
	if err != nil {
		return nil, err
	}
	if contractExpiryType != "" {
		vals.Set("contract_expiry_type", contractExpiryType)
	}
	if productType != "" {
		vals.Set("product_type", productType)
	}
	if productVenue != "" {
		vals.Set("product_venue", productVenue)
	}
	var resp *TransactionSummary
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, v3Path+transactionSummaryPath, vals, nil, true, &resp)
}

// CancelPendingFuturesSweep cancels a pending sweep request
func (e *Exchange) CancelPendingFuturesSweep(ctx context.Context) (bool, error) {
	path := v3Path + futuresPath + "/" + sweepsPath
	var resp SuccessResp
	return resp.Success, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, nil, nil, true, &resp)
}

// GetCurrentMarginWindow returns the futures current margin window
func (e *Exchange) GetCurrentMarginWindow(ctx context.Context, marginProfileType string) (*CurrentMarginWindow, error) {
	if marginProfileType == "" {
		return nil, errMarginProfileTypeEmpty
	}
	vals := url.Values{}
	if marginProfileType != "" {
		vals.Set("margin_profile_type", marginProfileType)
	}
	path := v3Path + futuresPath + "/" + intradayPath + "/" + currentMarginWindowPath
	var resp *CurrentMarginWindow
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, vals, nil, true, &resp)
}

// GetFuturesBalanceSummary returns information on balances related to Coinbase Financial Markets futures trading
func (e *Exchange) GetFuturesBalanceSummary(ctx context.Context) (*FuturesBalanceSummary, error) {
	resp := struct {
		BalanceSummary FuturesBalanceSummary `json:"balance_summary"`
	}{}
	path := v3Path + futuresPath + "/" + balanceSummaryPath
	return &resp.BalanceSummary, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, true, &resp)
}

// GetFuturesPositionByID returns information on an open futures position
func (e *Exchange) GetFuturesPositionByID(ctx context.Context, productID currency.Pair) (*FuturesPosition, error) {
	if productID.IsEmpty() {
		return nil, errProductIDEmpty
	}
	path := v3Path + futuresPath + "/" + positionsPath + "/" + productID.String()
	resp := struct {
		Position FuturesPosition `json:"position"`
	}{}
	return &resp.Position, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, true, &resp)
}

// GetIntradayMarginSetting returns the futures intraday margin setting
func (e *Exchange) GetIntradayMarginSetting(ctx context.Context) (string, error) {
	resp := struct {
		Setting string `json:"setting"`
	}{}
	path := v3Path + futuresPath + "/" + intradayPath + "/" + marginSettingPath
	return resp.Setting, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, true, &resp)
}

// ListFuturesPositions returns a list of all open futures positions
func (e *Exchange) ListFuturesPositions(ctx context.Context) ([]FuturesPosition, error) {
	resp := struct {
		Positions []FuturesPosition `json:"positions"`
	}{}
	path := v3Path + futuresPath + "/" + positionsPath
	return resp.Positions, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, true, &resp)
}

// ListFuturesSweeps returns information on pending and/or processing requests to sweep funds
func (e *Exchange) ListFuturesSweeps(ctx context.Context) ([]SweepData, error) {
	resp := struct {
		Sweeps []SweepData `json:"sweeps"`
	}{}
	path := v3Path + futuresPath + "/" + sweepsPath
	return resp.Sweeps, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, true, &resp)
}

// ScheduleFuturesSweep schedules a sweep of funds from a CFTC-regulated futures account to a Coinbase USD Spot wallet. Request submitted before 5 pm ET are processed the following business day, requests submitted after are processed in 2 business days. Only one sweep request can be pending at a time. Funds transferred depend on the excess available in the futures account. An amount of 0 will sweep all available excess funds
func (e *Exchange) ScheduleFuturesSweep(ctx context.Context, amount float64) (bool, error) {
	path := v3Path + futuresPath + "/" + sweepsPath + "/" + schedulePath
	var resp SuccessResp
	return resp.Success, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, futuresSweepReqBase{USDAmount: amount}, true, &resp)
}

// SetIntradayMarginSetting sets the futures intraday margin setting
func (e *Exchange) SetIntradayMarginSetting(ctx context.Context, setting string) error {
	if setting == "" {
		return errSettingEmpty
	}
	path := v3Path + futuresPath + "/" + intradayPath + "/" + marginSettingPath
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, marginSettingReqBase{Setting: setting}, true, nil)
}

// CancelOrders cancels orders by orderID. Can only cancel 100 orders per request
func (e *Exchange) CancelOrders(ctx context.Context, orderIDs []string) ([]OrderCancelDetail, error) {
	if len(orderIDs) == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if len(orderIDs) > 100 {
		return nil, errCancelLimitExceeded
	}
	path := v3Path + ordersPath + "/" + batchCancelpath
	resp := struct {
		Results []OrderCancelDetail `json:"results"`
	}{}
	return resp.Results, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, cancelOrdersReqBase{OrderIDs: orderIDs}, true, &resp)
}

// ClosePosition closes a position by client order ID, product ID, and size
func (e *Exchange) ClosePosition(ctx context.Context, clientOrderID string, productID currency.Pair, size float64) (*SuccessFailureConfig, error) {
	if clientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	if productID.IsEmpty() {
		return nil, errProductIDEmpty
	}
	if size <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	path := v3Path + ordersPath + "/" + closePositionPath
	req := closePositionReqBase{
		ClientOrderID: clientOrderID,
		ProductID:     productID,
		Size:          size,
	}
	var resp *SuccessFailureConfig
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, req, true, &resp)
}

// PlaceOrder places either a limit, market, or stop order
func (e *Exchange) PlaceOrder(ctx context.Context, ord *PlaceOrderInfo) (*SuccessFailureConfig, error) {
	if ord.ClientOID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	if ord.ProductID == "" {
		return nil, errProductIDEmpty
	}
	if ord.BaseAmount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	orderConfig, err := createOrderConfig(&ord.OrderInfo)
	if err != nil {
		return nil, err
	}
	req := placeOrderReqbase{
		ClientOID:                  ord.ClientOID,
		ProductID:                  ord.ProductID,
		Side:                       ord.Side,
		OrderConfiguration:         &orderConfig,
		RetailPortfolioID:          ord.RetailPortfolioID,
		PreviewID:                  ord.PreviewID,
		AttachedOrderConfiguration: &ord.AttachedOrderConfiguration,
		MarginType:                 FormatMarginType(ord.MarginType),
	}
	if ord.Leverage != 1 {
		req.Leverage = ord.Leverage
	}
	var resp *SuccessFailureConfig
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, v3Path+ordersPath, nil, req, true, &resp)
}

// EditOrder edits an order to a new size or price. Only limit orders with a good-till-cancelled time in force can be edited
func (e *Exchange) EditOrder(ctx context.Context, orderID string, size, price float64) (bool, error) {
	if orderID == "" {
		return false, order.ErrOrderIDNotSet
	}
	if size <= 0 && price <= 0 {
		return false, errSizeAndPriceZero
	}
	path := v3Path + ordersPath + "/" + editPath
	req := editOrderReqBase{
		OrderID: orderID,
		Size:    size,
		Price:   price,
	}
	var resp SuccessResp
	return resp.Success, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, req, true, &resp)
}

// EditOrderPreview simulates an edit order request, to preview the result. Only limit orders with a good-till-cancelled time in force can be edited.
func (e *Exchange) EditOrderPreview(ctx context.Context, orderID string, size, price float64) (*EditOrderPreviewResp, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if size <= 0 && price <= 0 {
		return nil, errSizeAndPriceZero
	}
	path := v3Path + ordersPath + "/" + editPreviewPath
	req := editOrderReqBase{
		OrderID: orderID,
		Size:    size,
		Price:   price,
	}
	var resp *EditOrderPreviewResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, req, true, &resp)
}

// GetOrderByID returns a single order by order id.
func (e *Exchange) GetOrderByID(ctx context.Context, orderID, clientOID string, userNativeCurrency currency.Code) (*GetOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	vals := url.Values{}
	if clientOID != "" {
		vals.Set("client_order_id", clientOID)
	}
	if !userNativeCurrency.IsEmpty() {
		vals.Set("user_native_currency", userNativeCurrency.String())
	}
	path := v3Path + ordersPath + "/" + historicalPath + "/" + orderID
	resp := struct {
		Order GetOrderResponse `json:"order"`
	}{}
	return &resp.Order, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, vals, nil, true, &resp)
}

// ListFills returns information on recent order fills
func (e *Exchange) ListFills(ctx context.Context, orderIDs, tradeIDs []string, productIDs currency.Pairs, cursor int64, sortBy string, startDate, endDate time.Time, limit uint16) (*FillResponse, error) {
	vals, err := urlValsFromDateRange(startDate, endDate, "start_sequence_timestamp", "end_sequence_timestamp")
	if err != nil {
		return nil, err
	}
	for i := range orderIDs {
		vals.Add("order_ids", orderIDs[i])
	}
	for i := range tradeIDs {
		vals.Add("trade_ids", tradeIDs[i])
	}
	for i := range productIDs {
		vals.Add("product_ids", productIDs[i].String())
	}
	vals.Set("limit", strconv.FormatInt(int64(limit), 10))
	vals.Set("cursor", strconv.FormatInt(cursor, 10))
	if sortBy != "" {
		vals.Set("sort_by", sortBy)
	}
	path := v3Path + ordersPath + "/" + historicalPath + "/" + fillsPath
	var resp *FillResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, vals, nil, true, &resp)
}

// ListOrders lists orders, filtered by their status
func (e *Exchange) ListOrders(ctx context.Context, req *ListOrdersReq) (*ListOrdersResp, error) {
	vals, err := urlValsFromDateRange(req.StartDate, req.EndDate, startDateString, endDateString)
	if err != nil {
		return nil, err
	}
	for x := range req.OrderStatus {
		vals.Add("order_status", req.OrderStatus[x])
	}
	for x := range req.OrderIDs {
		vals.Add("order_ids", req.OrderIDs[x])
	}
	for x := range req.TimeInForces {
		vals.Add("time_in_forces", req.TimeInForces[x])
	}
	for x := range req.OrderTypes {
		vals.Add("order_types", req.OrderTypes[x])
	}
	for x := range req.AssetFilters {
		vals.Add("asset_filters", req.AssetFilters[x])
	}
	for x := range req.ProductIDs {
		vals.Add("product_ids", req.ProductIDs[x].String())
	}
	if req.ProductType != "" {
		vals.Set("product_type", req.ProductType)
	}
	if req.OrderSide != "" {
		vals.Set("order_side", req.OrderSide)
	}
	if req.OrderPlacementSource != "" {
		vals.Set("order_placement_source", req.OrderPlacementSource)
	}
	if req.ContractExpiryType != "" {
		vals.Set("contract_expiry_type", req.ContractExpiryType)
	}
	if req.SortBy != "" {
		vals.Set("sort_by", req.SortBy)
	}
	vals.Set("cursor", strconv.FormatInt(req.Cursor, 10))
	vals.Set("limit", strconv.FormatInt(int64(req.Limit), 10))
	vals.Set("user_native_currency", req.UserNativeCurrency.String())
	vals.Set("retail_portfolio_id", req.RetailPortfolioID) // deprecated and only works for legacy API keys
	path := v3Path + ordersPath + "/" + historicalPath + "/" + batchPath
	var resp *ListOrdersResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, vals, nil, true, &resp)
}

// PreviewOrder simulates the results of an order request
func (e *Exchange) PreviewOrder(ctx context.Context, inf *PreviewOrderInfo) (*PreviewOrderResp, error) {
	if inf.BaseAmount <= 0 && inf.QuoteAmount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	orderConfig, err := createOrderConfig(&inf.OrderInfo)
	if err != nil {
		return nil, err
	}
	req := previewOrderReqBase{
		ProductID:                  inf.ProductID,
		Side:                       inf.Side,
		OrderConfiguration:         &orderConfig,
		RetailPortfolioID:          inf.RetailPortfolioID,
		Leverage:                   inf.Leverage,
		AttachedOrderConfiguration: &inf.AttachedOrderConfiguration,
		MarginType:                 FormatMarginType(inf.MarginType),
	}
	var resp *PreviewOrderResp
	path := v3Path + ordersPath + "/" + previewPath
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, req, true, &resp)
}

// GetPaymentMethodByID returns information on a single payment method associated with the user's account
func (e *Exchange) GetPaymentMethodByID(ctx context.Context, paymentMethodID string) (*PaymentMethodData, error) {
	if paymentMethodID == "" {
		return nil, errPaymentMethodEmpty
	}
	path := v3Path + paymentMethodsPath + "/" + paymentMethodID
	resp := struct {
		PaymentMethod PaymentMethodData `json:"payment_method"`
	}{}
	return &resp.PaymentMethod, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, true, &resp)
}

// ListPaymentMethods returns a list of all payment methods associated with the user's account
func (e *Exchange) ListPaymentMethods(ctx context.Context) ([]PaymentMethodData, error) {
	resp := struct {
		PaymentMethods []PaymentMethodData `json:"payment_methods"`
	}{}
	path := v3Path + paymentMethodsPath
	return resp.PaymentMethods, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, paymentMethodReqBase{Currency: currency.BTC}, true, &resp)
}

// AllocatePortfolio allocates funds to a position in your perpetuals portfolio
func (e *Exchange) AllocatePortfolio(ctx context.Context, portfolioID, productID, cur string, amount float64) error {
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
	req := allocatePortfolioReqBase{
		PortfolioUUID: portfolioID,
		Symbol:        productID,
		Currency:      cur,
		Amount:        amount,
	}
	path := v3Path + intxPath + "/" + allocatePath
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, req, true, nil)
}

// GetPerpetualsPortfolioSummary returns a summary of your perpetuals portfolio
func (e *Exchange) GetPerpetualsPortfolioSummary(ctx context.Context, portfolioID string) (*PerpetualPortfolioResponse, error) {
	if portfolioID == "" {
		return nil, errPortfolioIDEmpty
	}
	path := v3Path + intxPath + "/" + portfolioPath + "/" + portfolioID
	resp := struct {
		Summary PerpetualPortfolioResponse `json:"summary"`
	}{}
	return &resp.Summary, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, true, &resp)
}

// GetPerpetualsPositionByID returns information on a single open position in your perpetuals portfolio
func (e *Exchange) GetPerpetualsPositionByID(ctx context.Context, portfolioID string, productID currency.Pair) (*PerpPositionDetail, error) {
	if portfolioID == "" {
		return nil, errPortfolioIDEmpty
	}
	if productID.IsEmpty() {
		return nil, errProductIDEmpty
	}
	path := v3Path + intxPath + "/" + positionsPath + "/" + portfolioID + "/" + productID.String()
	resp := struct {
		Position PerpPositionDetail `json:"position"`
	}{}
	return &resp.Position, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, true, &resp)
}

// GetPortfolioBalances returns the current balances for all assets in your portfolio
func (e *Exchange) GetPortfolioBalances(ctx context.Context, portfolioID string) ([]PortfolioBalancesResponse, error) {
	if portfolioID == "" {
		return nil, errPortfolioIDEmpty
	}
	path := v3Path + intxPath + "/" + balancesPath + "/" + portfolioID
	resp := struct {
		PortfolioBalances []PortfolioBalancesResponse `json:"portfolio_balances"`
	}{}
	return resp.PortfolioBalances, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, true, &resp)
}

// GetAllPerpetualsPositions returns a list of all open positions in your perpetuals portfolio
func (e *Exchange) GetAllPerpetualsPositions(ctx context.Context, portfolioID string) (*AllPerpPosResponse, error) {
	if portfolioID == "" {
		return nil, errPortfolioIDEmpty
	}
	path := v3Path + intxPath + "/" + positionsPath + "/" + portfolioID
	var resp *AllPerpPosResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, true, &resp)
}

// MultiAssetCollateralToggle allows for the toggling of multi-asset collateral on a portfolio
func (e *Exchange) MultiAssetCollateralToggle(ctx context.Context, portfolioID string, enabled bool) (bool, error) {
	if portfolioID == "" {
		return false, errPortfolioIDEmpty
	}
	path := v3Path + intxPath + "/" + multiAssetCollateralPath
	var resp struct {
		Enabled bool `json:"multi_asset_collateral_enabled"`
	}
	return resp.Enabled, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, assetCollateralToggleReqBase{PortfolioUUID: portfolioID, Enabled: enabled}, true, &resp)
}

// CreatePortfolio creates a new portfolio
func (e *Exchange) CreatePortfolio(ctx context.Context, name string) (*SimplePortfolioData, error) {
	if name == "" {
		return nil, errNameEmpty
	}
	resp := struct {
		Portfolio SimplePortfolioData `json:"portfolio"`
	}{}
	return &resp.Portfolio, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, v3Path+portfoliosPath, nil, nameReqBase{Name: name}, true, &resp)
}

// DeletePortfolio deletes a portfolio
func (e *Exchange) DeletePortfolio(ctx context.Context, portfolioID string) error {
	if portfolioID == "" {
		return errPortfolioIDEmpty
	}
	path := v3Path + portfoliosPath + "/" + portfolioID
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, nil, nil, true, nil)
}

// EditPortfolio edits the name of a portfolio
func (e *Exchange) EditPortfolio(ctx context.Context, portfolioID, name string) (*SimplePortfolioData, error) {
	if portfolioID == "" {
		return nil, errPortfolioIDEmpty
	}
	if name == "" {
		return nil, errNameEmpty
	}
	path := v3Path + portfoliosPath + "/" + portfolioID
	resp := struct {
		Portfolio SimplePortfolioData `json:"portfolio"`
	}{}
	return &resp.Portfolio, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPut, path, nil, nameReqBase{Name: name}, true, &resp)
}

// GetPortfolioByID provides detailed information on a single portfolio
func (e *Exchange) GetPortfolioByID(ctx context.Context, portfolioID string) (*DetailedPortfolioResponse, error) {
	if portfolioID == "" {
		return nil, errPortfolioIDEmpty
	}
	path := v3Path + portfoliosPath + "/" + portfolioID
	resp := struct {
		Breakdown DetailedPortfolioResponse `json:"breakdown"`
	}{}
	return &resp.Breakdown, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, true, &resp)
}

// GetAllPortfolios returns a list of portfolios associated with the user
func (e *Exchange) GetAllPortfolios(ctx context.Context, portfolioType string) ([]SimplePortfolioData, error) {
	resp := struct {
		Portfolios []SimplePortfolioData `json:"portfolios"`
	}{}
	vals := url.Values{}
	if portfolioType != "" {
		vals.Set("portfolio_type", portfolioType)
	}
	return resp.Portfolios, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, v3Path+portfoliosPath, vals, nil, true, &resp)
}

// MovePortfolioFunds transfers funds between portfolios
func (e *Exchange) MovePortfolioFunds(ctx context.Context, cur currency.Code, from, to string, amount float64) (*MovePortfolioFundsResponse, error) {
	if from == "" || to == "" {
		return nil, errPortfolioIDEmpty
	}
	if cur.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	req := movePortfolioFundsReqBase{
		SourcePortfolioUUID: from,
		TargetPortfolioUUID: to,
		Funds: fundsData{
			Value:    amount,
			Currency: cur,
		},
	}
	path := v3Path + portfoliosPath + "/" + moveFundsPath
	var resp *MovePortfolioFundsResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, req, true, &resp)
}

// GetBestBidAsk returns the best bid/ask for all products. Can be filtered to certain products by passing through additional strings
func (e *Exchange) GetBestBidAsk(ctx context.Context, products []string) ([]ProductBook, error) {
	vals := url.Values{}
	for x := range products {
		vals.Add("product_ids", products[x])
	}
	resp := struct {
		Pricebooks []ProductBook `json:"pricebooks"`
	}{}
	return resp.Pricebooks, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, v3Path+bestBidAskPath, vals, nil, true, &resp)
}

// GetTicker returns snapshot information about the last trades (ticks) and best bid/ask. Contrary to documentation, this does not tell you the 24h volume
func (e *Exchange) GetTicker(ctx context.Context, productID currency.Pair, limit uint16, startDate, endDate time.Time, authenticated bool) (*Ticker, error) {
	if productID.IsEmpty() {
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
	var resp *Ticker
	if authenticated {
		path := v3Path + productsPath + "/" + productID.String() + "/" + tickerPath
		return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, vals, nil, true, &resp)
	}
	path := v3Path + marketPath + "/" + productsPath + "/" + productID.String() + "/" + tickerPath
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, path, vals, &resp)
}

// GetProductByID returns information on a single specified currency pair
func (e *Exchange) GetProductByID(ctx context.Context, productID currency.Pair, authenticated bool) (*Product, error) {
	if productID.IsEmpty() {
		return nil, errProductIDEmpty
	}
	var resp *Product
	if authenticated {
		path := v3Path + productsPath + "/" + productID.String()
		return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, true, &resp)
	}
	path := v3Path + marketPath + "/" + productsPath + "/" + productID.String()
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, path, nil, &resp)
}

// GetProductBookV3 returns a list of bids/asks for a single product
func (e *Exchange) GetProductBookV3(ctx context.Context, productID currency.Pair, limit uint16, aggregationIncrement float64, authenticated bool) (*ProductBookResp, error) {
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
		return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, v3Path+productBookPath, vals, nil, true, &resp)
	}
	path := v3Path + marketPath + "/" + productBookPath
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, path, vals, &resp)
}

// GetHistoricKlines returns historic candles for a product. Candles are returned in grouped buckets based on requested granularity. Requests that return more than 300 data points are rejected
func (e *Exchange) GetHistoricKlines(ctx context.Context, productID string, granularity kline.Interval, startDate, endDate time.Time, authenticated bool) ([]kline.Candle, error) {
	if productID == "" {
		return nil, errProductIDEmpty
	}
	gran, ok := allowedGranularities[granularity]
	if !ok {
		return nil, fmt.Errorf("%w %v, allowed granularities are: %+v", kline.ErrUnsupportedInterval, granularity, allowedGranularities)
	}
	vals := url.Values{}
	vals.Set("start", strconv.FormatInt(startDate.Unix(), 10))
	vals.Set("end", strconv.FormatInt(endDate.Unix(), 10))
	vals.Set("granularity", gran)
	resp := struct {
		Candles []Klines `json:"candles"`
	}{}
	var err error
	if authenticated {
		path := v3Path + productsPath + "/" + productID + "/" + candlesPath
		err = e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, vals, nil, true, &resp)
	} else {
		path := v3Path + marketPath + "/" + productsPath + "/" + productID + "/" + candlesPath
		err = e.SendHTTPRequest(ctx, exchange.RestSpot, path, vals, &resp)
	}
	if err != nil {
		return nil, err
	}
	timeSeries := make([]kline.Candle, len(resp.Candles))
	for x := range resp.Candles {
		timeSeries[x] = kline.Candle{
			Time:   resp.Candles[x].Start.Time(),
			Low:    resp.Candles[x].Low.Float64(),
			High:   resp.Candles[x].High.Float64(),
			Open:   resp.Candles[x].Open.Float64(),
			Close:  resp.Candles[x].Close.Float64(),
			Volume: resp.Candles[x].Volume.Float64(),
		}
	}
	return timeSeries, nil
}

// GetAllProducts returns information on all currency pairs that are available for trading
// The getTradabilityStatus parameter is only used for authenticated requests, and will return the tradability status of SPOT products in their view_only field
// The getAllProducts parameter overrides the set productType; with it set to true, it will return both SPOT and Futures products
func (e *Exchange) GetAllProducts(ctx context.Context, limit, offset int32, productType, contractExpiryType, expiringContractStatus, productsSortOrder string, productIDs []string, getTradabilityStatus, getAllProducts, authenticated bool) (*AllProducts, error) {
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
	if productsSortOrder != "" {
		vals.Set("products_sort_order", productsSortOrder)
	}
	for x := range productIDs {
		vals.Add("product_ids", productIDs[x])
	}
	vals.Set("get_tradability_status", strconv.FormatBool(getTradabilityStatus))
	vals.Set("get_all_products", strconv.FormatBool(getAllProducts))
	var resp *AllProducts
	if authenticated {
		return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, v3Path+productsPath, vals, nil, true, &resp)
	}
	path := v3Path + marketPath + "/" + productsPath
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, path, vals, &resp)
}

// GetV3Time returns the current server time, calling V3 of the API
func (e *Exchange) GetV3Time(ctx context.Context) (*ServerTimeV3, error) {
	var resp *ServerTimeV3
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, v3Path+timePath, nil, &resp)
}

// SendMoney can send funds to an email or cryptocurrency address (if "traType" is set to "send"), or to another one of the user's wallets or vaults (if "traType" is set to "transfer"). Coinbase may delay or cancel the transaction at their discretion. The "idem" parameter is an optional string for idempotency; a token with a max length of 100 characters, if a previous transaction included the same token as a parameter, the new transaction won't be processed, and information on the previous transaction will be returned instead
func (e *Exchange) SendMoney(ctx context.Context, traType, walletID, to, description, idem, destinationTag, network string, cur currency.Code, amount float64, skipNotifications bool, travelRuleData *TravelRule) (*TransactionData, error) {
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
	if cur.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	path := v2Path + accountsPath + "/" + walletID + "/" + transactionsPath
	req := sendMoneyReqBase{
		Type:              traType,
		To:                to,
		Amount:            amount,
		Currency:          cur,
		Description:       description,
		SkipNotifications: skipNotifications,
		Idem:              idem,
		DestinationTag:    destinationTag,
		Network:           network,
		TravelRuleData:    travelRuleData,
	}
	resp := struct {
		Data TransactionData `json:"data"`
	}{}
	return &resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, req, false, &resp)
}

// CreateAddress generates a crypto address for depositing to the specified wallet
func (e *Exchange) CreateAddress(ctx context.Context, walletID, name string) (*AddressData, error) {
	if walletID == "" {
		return nil, errWalletIDEmpty
	}
	path := v2Path + accountsPath + "/" + walletID + "/" + addressesPath
	resp := struct {
		Data AddressData `json:"data"`
	}{}
	return &resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, nameReqBase{Name: name}, false, &resp)
}

// GetAllAddresses returns information on all addresses associated with a wallet
func (e *Exchange) GetAllAddresses(ctx context.Context, walletID string, pag PaginationInp) (*GetAllAddrResponse, error) {
	if walletID == "" {
		return nil, errWalletIDEmpty
	}
	path := v2Path + accountsPath + "/" + walletID + "/" + addressesPath
	vals := urlValsFromPagination(pag)
	var resp *GetAllAddrResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, vals, nil, false, &resp)
}

// GetAddressByID returns information on a single address associated with the specified wallet
func (e *Exchange) GetAddressByID(ctx context.Context, walletID, addressID string) (*AddressData, error) {
	if walletID == "" {
		return nil, errWalletIDEmpty
	}
	if addressID == "" {
		return nil, errAddressIDEmpty
	}
	path := v2Path + accountsPath + "/" + walletID + "/" + addressesPath + "/" + addressID
	resp := struct {
		Data AddressData `json:"data"`
	}{}
	return &resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, false, &resp)
}

// GetAddressTransactions returns a list of transactions associated with the specified address
func (e *Exchange) GetAddressTransactions(ctx context.Context, walletID, addressID string, pag PaginationInp) (*ManyTransactionsResp, error) {
	if walletID == "" {
		return nil, errWalletIDEmpty
	}
	if addressID == "" {
		return nil, errAddressIDEmpty
	}
	path := v2Path + accountsPath + "/" + walletID + "/" + addressesPath + "/" + addressID + "/" + transactionsPath
	vals := urlValsFromPagination(pag)
	var resp *ManyTransactionsResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, vals, nil, false, &resp)
}

// FiatTransfer prepares and optionally processes a transfer of funds between the exchange and a fiat payment method. "Deposit" signifies funds going from exchange to bank, "withdraw" signifies funds going from bank to exchange
func (e *Exchange) FiatTransfer(ctx context.Context, walletID, cur, paymentMethod string, amount float64, commit bool, transferType FiatTransferType) (*DeposWithdrData, error) {
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
		path = v2Path + accountsPath + "/" + walletID + "/" + depositsPath
	case FiatWithdrawal:
		path = v2Path + accountsPath + "/" + walletID + "/" + withdrawalsPath
	}
	req := fiatTransferReqBase{
		Currency:      cur,
		PaymentMethod: paymentMethod,
		Amount:        amount,
		Commit:        commit,
	}
	resp := struct {
		Data DeposWithdrData `json:"transfer"`
	}{}
	return &resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, req, false, &resp)
}

// CommitTransfer processes a deposit/withdrawal that was created with the "commit" parameter set to false
func (e *Exchange) CommitTransfer(ctx context.Context, walletID, depositID string, transferType FiatTransferType) (*DeposWithdrData, error) {
	if walletID == "" {
		return nil, errWalletIDEmpty
	}
	if depositID == "" {
		return nil, errDepositIDEmpty
	}
	var path string
	switch transferType {
	case FiatDeposit:
		path = v2Path + accountsPath + "/" + walletID + "/" + depositsPath + "/" + depositID + "/" + commitPath
	case FiatWithdrawal:
		path = v2Path + accountsPath + "/" + walletID + "/" + withdrawalsPath + "/" + depositID + "/" + commitPath
	}
	resp := struct {
		Data DeposWithdrData `json:"data"`
	}{}
	return &resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, nil, false, &resp)
}

// GetAllFiatTransfers returns a list of transfers either to or from fiat payment methods and the specified wallet
func (e *Exchange) GetAllFiatTransfers(ctx context.Context, walletID string, pag PaginationInp, transferType FiatTransferType) (*ManyDeposWithdrResp, error) {
	if walletID == "" {
		return nil, errWalletIDEmpty
	}
	var path string
	switch transferType {
	case FiatDeposit:
		path = v2Path + accountsPath + "/" + walletID + "/" + depositsPath
	case FiatWithdrawal:
		path = v2Path + accountsPath + "/" + walletID + "/" + withdrawalsPath
	}
	vals := urlValsFromPagination(pag)
	var resp *ManyDeposWithdrResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, vals, nil, false, &resp)
}

// GetFiatTransferByID returns information on a single deposit/withdrawal associated with the specified wallet
func (e *Exchange) GetFiatTransferByID(ctx context.Context, walletID, depositID string, transferType FiatTransferType) (*DeposWithdrData, error) {
	if walletID == "" {
		return nil, errWalletIDEmpty
	}
	if depositID == "" {
		return nil, errDepositIDEmpty
	}
	var path string
	switch transferType {
	case FiatDeposit:
		path = v2Path + accountsPath + "/" + walletID + "/" + depositsPath + "/" + depositID
	case FiatWithdrawal:
		path = v2Path + accountsPath + "/" + walletID + "/" + withdrawalsPath + "/" + depositID
	}
	resp := struct {
		Data DeposWithdrData `json:"data"`
	}{}
	return &resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, false, &resp)
}

// GetAllWallets lists all accounts associated with the API key
func (e *Exchange) GetAllWallets(ctx context.Context, pag PaginationInp) (*GetAllWalletsResponse, error) {
	vals := urlValsFromPagination(pag)
	var resp *GetAllWalletsResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, v2Path+accountsPath, vals, nil, false, &resp)
}

// GetWalletByID returns information about a single wallet. In lieu of a wallet ID, a currency can be provided to get the primary account for that currency
func (e *Exchange) GetWalletByID(ctx context.Context, walletID string, cur currency.Code) (*WalletData, error) {
	if (walletID == "" && cur.IsEmpty()) || (walletID != "" && !cur.IsEmpty()) {
		return nil, errCurrWalletConflict
	}
	var path string
	if walletID != "" {
		path = v2Path + accountsPath + "/" + walletID
	}
	if !cur.IsEmpty() {
		path = v2Path + accountsPath + "/" + cur.String()
	}
	resp := struct {
		Data WalletData `json:"data"`
	}{}
	return &resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, false, &resp)
}

// GetAllTransactions returns a list of transactions associated with the specified wallet
func (e *Exchange) GetAllTransactions(ctx context.Context, walletID string, pag PaginationInp) (*ManyTransactionsResp, error) {
	if walletID == "" {
		return nil, errWalletIDEmpty
	}
	vals := urlValsFromPagination(pag)
	path := v2Path + accountsPath + "/" + walletID + "/" + transactionsPath
	var resp *ManyTransactionsResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, vals, nil, false, &resp)
}

// GetTransactionByID returns information on a single transaction associated with the specified wallet
func (e *Exchange) GetTransactionByID(ctx context.Context, walletID, transactionID string) (*TransactionData, error) {
	if walletID == "" {
		return nil, errWalletIDEmpty
	}
	if transactionID == "" {
		return nil, errTransactionIDEmpty
	}
	path := v2Path + accountsPath + "/" + walletID + "/" + transactionsPath + "/" + transactionID
	resp := struct {
		Data TransactionData `json:"data"`
	}{}
	return &resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, false, &resp)
}

// GetFiatCurrencies lists currencies that Coinbase knows about
func (e *Exchange) GetFiatCurrencies(ctx context.Context) ([]FiatData, error) {
	resp := struct {
		Data []FiatData `json:"data"`
	}{}
	return resp.Data, e.SendHTTPRequest(ctx, exchange.RestSpot, v2Path+currenciesPath, nil, &resp)
}

// GetCryptocurrencies lists cryptocurrencies that Coinbase knows about
func (e *Exchange) GetCryptocurrencies(ctx context.Context) ([]CryptoData, error) {
	resp := struct {
		Data []CryptoData `json:"data"`
	}{}
	path := v2Path + currenciesPath + "/" + cryptoPath
	return resp.Data, e.SendHTTPRequest(ctx, exchange.RestSpot, path, nil, &resp)
}

// GetExchangeRates returns exchange rates for the specified currency. If none is specified, it defaults to USD
func (e *Exchange) GetExchangeRates(ctx context.Context, cur string) (*GetExchangeRatesResp, error) {
	resp := struct {
		Data GetExchangeRatesResp `json:"data"`
	}{}
	vals := url.Values{}
	vals.Set("currency", cur)
	return &resp.Data, e.SendHTTPRequest(ctx, exchange.RestSpot, v2Path+exchangeRatesPath, vals, &resp)
}

// GetPrice returns the price the spot/buy/sell price for the specified currency pair, including the standard Coinbase fee of 1%, but excluding any other fees
func (e *Exchange) GetPrice(ctx context.Context, currencyPair, priceType string) (*GetPriceResp, error) {
	var path string
	switch priceType {
	case "spot", "buy", "sell":
		path = v2Path + pricesPath + "/" + currencyPair + "/" + priceType
	default:
		return nil, errInvalidPriceType
	}
	resp := struct {
		Data GetPriceResp `json:"data"`
	}{}
	return &resp.Data, e.SendHTTPRequest(ctx, exchange.RestSpot, path, nil, &resp)
}

// GetV2Time returns the current server time, calling V2 of the API
func (e *Exchange) GetV2Time(ctx context.Context) (*ServerTimeV2, error) {
	resp := struct {
		Data ServerTimeV2 `json:"data"`
	}{}
	return &resp.Data, e.SendHTTPRequest(ctx, exchange.RestSpot, v2Path+timePath, nil, &resp)
}

// GetCurrentUser returns information about the user associated with the API key
func (e *Exchange) GetCurrentUser(ctx context.Context) (*UserResponse, error) {
	resp := struct {
		Data UserResponse `json:"data"`
	}{}
	return &resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, v2Path+userPath, nil, nil, false, &resp)
}

// GetAllCurrencies returns a list of all currencies that Coinbase knows about. These aren't necessarily tradable
func (e *Exchange) GetAllCurrencies(ctx context.Context) ([]CurrencyData, error) {
	var resp []CurrencyData
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, currenciesPath, nil, &resp)
}

// GetACurrency returns information on a single currency specified by the user
func (e *Exchange) GetACurrency(ctx context.Context, cur string) (*CurrencyData, error) {
	if cur == "" {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *CurrencyData
	path := currenciesPath + "/" + cur
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, nil, &resp)
}

// GetAllTradingPairs returns a list of currency pairs which are available for trading
func (e *Exchange) GetAllTradingPairs(ctx context.Context, pairType string) ([]PairData, error) {
	var resp []PairData
	vals := url.Values{}
	if pairType != "" {
		vals.Set("type", pairType)
	}
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, productsPath, vals, &resp)
}

// GetAllPairVolumes returns a list of currency pairs and their associated volumes
func (e *Exchange) GetAllPairVolumes(ctx context.Context) ([]PairVolumeData, error) {
	var resp []PairVolumeData
	path := productsPath + "/" + volumeSummaryPath
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, nil, &resp)
}

// GetPairDetails returns information on a single currency pair
func (e *Exchange) GetPairDetails(ctx context.Context, pair string) (*PairData, error) {
	if pair == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp *PairData
	path := productsPath + "/" + pair
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, nil, &resp)
}

// GetProductBookV1 returns the order book for the specified currency pair. Level 1 only returns the best bids and asks, Level 2 returns the full order book with orders at the same price aggregated, Level 3 returns the full non-aggregated order book.
func (e *Exchange) GetProductBookV1(ctx context.Context, pair string, level uint8) (*OrderBookResp, error) {
	if pair == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp *OrderBookResp
	vals := url.Values{}
	vals.Set("level", strconv.FormatUint(uint64(level), 10))
	path := productsPath + "/" + pair + "/" + bookPath
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, vals, &resp)
}

// GetProductCandles returns historical market data for the specified currency pair.
func (e *Exchange) GetProductCandles(ctx context.Context, pair string, granularity uint32, startTime, endTime time.Time) ([]Candle, error) {
	if pair == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	vals, err := urlValsFromDateRange(startTime, endTime, "start", "end")
	if err != nil {
		return nil, err
	}
	if granularity != 0 {
		vals.Set("granularity", strconv.FormatUint(uint64(granularity), 10))
	}
	path := productsPath + "/" + pair + "/" + candlesPath
	var resp []Candle
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, vals, &resp)
}

// GetProductStats returns information on a specific pair's price and volume
func (e *Exchange) GetProductStats(ctx context.Context, pair string) (*ProductStats, error) {
	if pair == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	path := productsPath + "/" + pair + "/" + statsPath
	var resp *ProductStats
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, nil, &resp)
}

// GetProductTicker returns the ticker for the specified currency pair
func (e *Exchange) GetProductTicker(ctx context.Context, pair string) (*ProductTicker, error) {
	if pair == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	path := productsPath + "/" + pair + "/" + tickerPath
	var resp *ProductTicker
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, nil, &resp)
}

// GetProductTrades returns a list of the latest traides for a pair
func (e *Exchange) GetProductTrades(ctx context.Context, pair, step, direction string, limit int64) ([]ProductTrades, error) {
	if pair == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	vals := url.Values{}
	if step != "" {
		vals.Set(direction, step)
	}
	vals.Set("limit", strconv.FormatInt(limit, 10))
	path := productsPath + "/" + pair + "/" + tradesPath
	var resp []ProductTrades
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, vals, &resp)
}

// GetAllWrappedAssets returns a list of supported wrapped assets
func (e *Exchange) GetAllWrappedAssets(ctx context.Context) (*AllWrappedAssets, error) {
	var resp *AllWrappedAssets
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, wrappedAssetsPath, nil, &resp)
}

// GetWrappedAssetDetails returns information on a single wrapped asset
func (e *Exchange) GetWrappedAssetDetails(ctx context.Context, wrappedAsset string) (*WrappedAsset, error) {
	if wrappedAsset == "" {
		return nil, errWrappedAssetEmpty
	}
	var resp *WrappedAsset
	path := wrappedAssetsPath + "/" + wrappedAsset
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, nil, &resp)
}

// GetWrappedAssetConversionRate returns the conversion rate for the specified wrapped asset
func (e *Exchange) GetWrappedAssetConversionRate(ctx context.Context, wrappedAsset string) (*WrappedAssetConversionRate, error) {
	if wrappedAsset == "" {
		return nil, errWrappedAssetEmpty
	}
	var resp *WrappedAssetConversionRate
	path := wrappedAssetsPath + "/" + wrappedAsset + "/" + conversionRatePath
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, nil, &resp)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (e *Exchange) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, vals url.Values, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	rLim := PubRate
	if strings.Contains(path, v2Path) {
		rLim = V2Rate
	}
	path = common.EncodeURLValues(path, vals)
	item := &request.Item{
		Method:                 http.MethodGet,
		Path:                   endpoint + path,
		Result:                 result,
		Verbose:                e.Verbose,
		HTTPDebugging:          e.HTTPDebugging,
		HTTPRecording:          e.HTTPRecording,
		HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
	}
	return e.SendPayload(ctx, rLim, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request
func (e *Exchange) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, method, path string, queryParams url.Values, payload any, isVersion3 bool, result any) (err error) {
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	if len(endpoint) < 8 {
		return errEndpointPathInvalid
	}
	interim := json.RawMessage{}
	newRequest := func() (*request.Item, error) {
		payloadBytes := []byte("")
		if payload != nil {
			if payloadBytes, err = json.Marshal(payload); err != nil {
				return nil, err
			}
		}
		var jwt string
		if jwt, _, err = e.GetJWT(ctx, method+" "+endpoint[8:]+path); err != nil {
			return nil, err
		}
		headers := make(map[string]string)
		headers["Content-Type"] = "application/json"
		headers["CB-VERSION"] = "2025-03-26"
		headers["Authorization"] = "Bearer " + jwt
		return &request.Item{
			Method:                 method,
			Path:                   endpoint + common.EncodeURLValues(path, queryParams),
			Headers:                headers,
			Body:                   bytes.NewBuffer(payloadBytes),
			Result:                 &interim,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}
	rateLim := V2Rate
	if isVersion3 {
		rateLim = V3Rate
	}
	if err := e.SendPayload(ctx, rateLim, newRequest, request.AuthenticatedRequest); err != nil {
		return err
	}
	// Doing this error handling because the docs indicate that errors can be returned even with a 200 status code, and that these errors can be buried in the JSON returned
	singleErrCap := struct {
		ErrorResponse ErrorResponse `json:"error_response"`
	}{}
	if err := json.Unmarshal(interim, &singleErrCap); err == nil {
		if singleErrCap.ErrorResponse.ErrorType != "" {
			return fmt.Errorf("message: %s, error type: %s, error details: %s, edit failure reason: %s, preview failure reason: %s, new order failure reason: %s", singleErrCap.ErrorResponse.Message, singleErrCap.ErrorResponse.ErrorType, singleErrCap.ErrorResponse.ErrorDetails, singleErrCap.ErrorResponse.EditFailureReason, singleErrCap.ErrorResponse.PreviewFailureReason, singleErrCap.ErrorResponse.NewOrderFailureReason)
		}
	}
	manyErrCap := struct {
		Results []ManyErrors `json:"results"`
		Errors  []ManyErrors `json:"errors"`
	}{}
	if err := json.Unmarshal(interim, &manyErrCap); err == nil {
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
func (e *Exchange) GetJWT(ctx context.Context, uri string) (string, time.Time, error) {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return "", time.Time{}, err
	}
	block, _ := pem.Decode([]byte(creds.Secret))
	if block == nil {
		return "", time.Time{}, errDecodingPrivateKey
	}
	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return "", time.Time{}, err
	}
	nonce, err := common.GenerateRandomString(16, "1234567890ABCDEF")
	if err != nil {
		return "", time.Time{}, err
	}
	head := map[string]any{
		"kid":   creds.Key,
		"typ":   "JWT",
		"alg":   "ES256",
		"nonce": nonce,
	}
	headJSON, err := json.Marshal(head)
	if err != nil {
		return "", time.Time{}, err
	}
	headEnc := base64.RawURLEncoding.EncodeToString(headJSON)
	regTime := time.Now()
	body := map[string]any{
		"iss": "cdp",
		"nbf": regTime.Unix(),
		// As per documentation, the JWT expires after two minutes, with the exchange expecting this expiry time to be set accordingly
		"exp": regTime.Add(2 * time.Minute).Unix(),
		"sub": creds.Key,
	}
	if uri != "" {
		body["uri"] = uri
	}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return "", time.Time{}, err
	}
	bodyEnc := base64.RawURLEncoding.EncodeToString(bodyJSON)
	signingInput := headEnc + "." + bodyEnc
	hash := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		return "", time.Time{}, err
	}
	n := privateKey.Params().N
	halfN := new(big.Int).Rsh(n, 1)
	if s.Cmp(halfN) == 1 {
		s.Sub(n, s)
	}
	rb := r.Bytes()
	sb := s.Bytes()
	sig := make([]byte, 64)
	copy(sig[32-len(rb):32], rb)
	copy(sig[64-len(sb):], sb)
	sigEnc := base64.RawURLEncoding.EncodeToString(sig)
	return signingInput + "." + sigEnc, regTime.Add(2 * time.Minute), nil
}

// GetFee returns an estimate of fee based on type of transaction
func (e *Exchange) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	var fee float64
	switch {
	case !isStablePair(feeBuilder.Pair) && feeBuilder.FeeType == exchange.CryptocurrencyTradeFee:
		fees, err := e.GetTransactionSummary(ctx, time.Now().Add(-time.Hour*24*30), time.Now(), "", "", "")
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

// isStablePair returns true if the currency pair is considered a "stable pair" by Coinbase
func isStablePair(pair currency.Pair) bool {
	return stableMap[key.PairAsset{Base: pair.Base.Item, Quote: pair.Quote.Item}]
}

// urlValsFromDateRange encodes a set of parameters indicating start and end dates
func urlValsFromDateRange(startDate, endDate time.Time, labelStart, labelEnd string) (url.Values, error) {
	values := url.Values{}
	if err := common.StartEndTimeCheck(startDate, endDate); err != nil {
		if errors.Is(err, common.ErrDateUnset) {
			return values, nil
		}
		return nil, err
	}
	if labelStart == "" || labelEnd == "" {
		return nil, errDateLabelEmpty
	}
	values.Set(labelStart, startDate.Format(time.RFC3339))
	values.Set(labelEnd, endDate.Format(time.RFC3339))
	return values, nil
}

// urlValsFromPagination formats pagination information in the way the exchange expects
func urlValsFromPagination(pag PaginationInp) url.Values {
	values := url.Values{}
	if pag.Limit != 0 {
		values.Set("limit", strconv.FormatInt(int64(pag.Limit), 10))
	}
	if pag.OrderAscend {
		values.Set("order", "asc")
	}
	if pag.StartingAfter != "" {
		values.Set("starting_after", pag.StartingAfter)
	}
	if pag.EndingBefore != "" {
		values.Set("ending_before", pag.EndingBefore)
	}
	return values
}

// createOrderConfig populates the OrderConfiguration struct
func createOrderConfig(sharedParams *OrderInfo) (OrderConfiguration, error) {
	if sharedParams == nil {
		return OrderConfiguration{}, fmt.Errorf("%T %w", sharedParams, common.ErrNilPointer)
	}
	var orderConfig OrderConfiguration
	switch sharedParams.OrderType {
	case order.Market:
		if sharedParams.BaseAmount != 0 {
			orderConfig.MarketMarketIOC = &MarketMarketIOC{BaseSize: types.Number(sharedParams.BaseAmount), RFQDisabled: sharedParams.RFQDisabled}
		}
		if sharedParams.QuoteAmount != 0 {
			orderConfig.MarketMarketIOC = &MarketMarketIOC{QuoteSize: types.Number(sharedParams.QuoteAmount), RFQDisabled: sharedParams.RFQDisabled}
		}
	case order.Limit:
		switch {
		case sharedParams.TimeInForce == order.StopOrReduce:
			orderConfig.SORLimitIOC = &QuoteBaseLimit{BaseSize: types.Number(sharedParams.BaseAmount), QuoteSize: types.Number(sharedParams.QuoteAmount), LimitPrice: types.Number(sharedParams.LimitPrice), RFQDisabled: sharedParams.RFQDisabled}
		case sharedParams.TimeInForce == order.FillOrKill:
			orderConfig.LimitLimitFOK = &QuoteBaseLimit{BaseSize: types.Number(sharedParams.BaseAmount), QuoteSize: types.Number(sharedParams.QuoteAmount), LimitPrice: types.Number(sharedParams.LimitPrice), RFQDisabled: sharedParams.RFQDisabled}
		case sharedParams.EndTime.IsZero():
			orderConfig.LimitLimitGTC = &LimitLimitGTC{LimitPrice: types.Number(sharedParams.LimitPrice), PostOnly: sharedParams.PostOnly, RFQDisabled: sharedParams.RFQDisabled}
			if sharedParams.BaseAmount != 0 {
				orderConfig.LimitLimitGTC.BaseSize = types.Number(sharedParams.BaseAmount)
			}
			if sharedParams.QuoteAmount != 0 {
				orderConfig.LimitLimitGTC.QuoteSize = types.Number(sharedParams.QuoteAmount)
			}
		default:
			if sharedParams.EndTime.Before(time.Now()) {
				return orderConfig, errEndTimeInPast
			}
			orderConfig.LimitLimitGTD = &LimitLimitGTD{LimitPrice: types.Number(sharedParams.LimitPrice), PostOnly: sharedParams.PostOnly, EndTime: sharedParams.EndTime, RFQDisabled: sharedParams.RFQDisabled}
			if sharedParams.BaseAmount != 0 {
				orderConfig.LimitLimitGTD.BaseSize = types.Number(sharedParams.BaseAmount)
			}
			if sharedParams.QuoteAmount != 0 {
				orderConfig.LimitLimitGTD.QuoteSize = types.Number(sharedParams.QuoteAmount)
			}
		}
	case order.TWAP:
		if sharedParams.EndTime.Before(time.Now()) {
			return orderConfig, errEndTimeInPast
		}
		orderConfig.TWAPLimitGTD = &TWAPLimitGTD{StartTime: time.Now(), EndTime: sharedParams.EndTime, LimitPrice: types.Number(sharedParams.LimitPrice), NumberBuckets: sharedParams.BucketNumber, BucketSize: types.Number(sharedParams.BucketSize), BucketDuration: strconv.FormatFloat(sharedParams.BucketDuration.Seconds(), 'f', -1, 64) + "s"}
	case order.StopLimit:
		if sharedParams.EndTime.IsZero() {
			orderConfig.StopLimitStopLimitGTC = &StopLimitStopLimitGTC{LimitPrice: types.Number(sharedParams.LimitPrice), StopPrice: types.Number(sharedParams.StopPrice), StopDirection: sharedParams.StopDirection}
			if sharedParams.BaseAmount != 0 {
				orderConfig.StopLimitStopLimitGTC.BaseSize = types.Number(sharedParams.BaseAmount)
			}
			if sharedParams.QuoteAmount != 0 {
				orderConfig.StopLimitStopLimitGTC.QuoteSize = types.Number(sharedParams.QuoteAmount)
			}
		} else {
			if sharedParams.EndTime.Before(time.Now()) {
				return orderConfig, errEndTimeInPast
			}
			orderConfig.StopLimitStopLimitGTD = &StopLimitStopLimitGTD{LimitPrice: types.Number(sharedParams.LimitPrice), StopPrice: types.Number(sharedParams.StopPrice), StopDirection: sharedParams.StopDirection, EndTime: sharedParams.EndTime}
			if sharedParams.BaseAmount != 0 {
				orderConfig.StopLimitStopLimitGTD.BaseSize = types.Number(sharedParams.BaseAmount)
			}
			if sharedParams.QuoteAmount != 0 {
				orderConfig.StopLimitStopLimitGTD.QuoteSize = types.Number(sharedParams.QuoteAmount)
			}
		}
	case order.Bracket:
		if sharedParams.EndTime.IsZero() {
			orderConfig.TriggerBracketGTC = &TriggerBracketGTC{BaseSize: types.Number(sharedParams.BaseAmount), LimitPrice: types.Number(sharedParams.LimitPrice), StopTriggerPrice: types.Number(sharedParams.StopPrice)}
		} else {
			if sharedParams.EndTime.Before(time.Now()) {
				return orderConfig, errEndTimeInPast
			}
			orderConfig.TriggerBracketGTD = &TriggerBracketGTD{BaseSize: types.Number(sharedParams.BaseAmount), LimitPrice: types.Number(sharedParams.LimitPrice), StopTriggerPrice: types.Number(sharedParams.StopPrice), EndTime: sharedParams.EndTime}
		}
	default:
		return orderConfig, errInvalidOrderType
	}
	return orderConfig, nil
}

// FormatMarginType properly formats the margin type for the request
func FormatMarginType(marginType string) string {
	switch marginType {
	case "ISOLATED", "CROSS":
		return marginType
	case "MULTI":
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
	err := json.Unmarshal(data, &[3]any{&o.Price, &o.Size, &alias})
	if err != nil {
		return err
	}
	switch a := alias.(type) {
	case string:
		if o.OrderID, err = uuid.FromString(a); err != nil {
			return err
		}
		o.OrderCount = 1
	case float64:
		o.OrderCount = uint64(a)
	default:
		return common.GetTypeAssertError("string | float64", alias, "Orders[3]")
	}
	return nil
}

// UnmarshalJSON unmarshals the JSON data
func (c *Candle) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[6]any{&c.Time, &c.Low, &c.High, &c.Open, &c.Close, &c.Volume})
}

// UnmarshalJSON unmarshals the JSON data
func (i *Integer) UnmarshalJSON(data []byte) error {
	var temp string
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	if temp == "" {
		*i = 0
		return nil
	}
	value, err := strconv.ParseInt(temp, 10, 64)
	if err != nil {
		return err
	}
	*i = Integer(value)
	return nil
}
