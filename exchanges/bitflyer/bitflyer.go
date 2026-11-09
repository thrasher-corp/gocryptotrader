package bitflyer

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	// Bitflyer chain analysis endpoints
	// APIURL
	chainAnalysis = "https://chainflyer.bitflyer.jp/v1/"
	tradeBaseURL  = "https://lightning.bitflyer.com/trade/"

	// APIURL
	japanURL  = "https://api.bitflyer.jp/v1"
	usURL     = "https://api.bitflyer.com/v1"
	europeURL = "https://api.bitflyer.com/v1"

	// Public Endpoints
	pubGetMarkets          = "/getmarkets/"
	pubGetBoard            = "/getboard"
	pubGetTicker           = "/getticker"
	pubGetExecutionHistory = "/getexecutions"
	pubGetHealth           = "/gethealth"
	pubGetChats            = "/getchats"

	orders request.EndpointLimit = iota
	lowVolume
)

// Exchange implements exchange.IBotExchange and contains additional specific api methods for interacting with Bitflyer
type Exchange struct {
	exchange.Base
}

// GetLatestBlockChainAnalysis returns the latest block information from bitflyer chain
// analysis system
func (e *Exchange) GetLatestBlockChainAnalysis(ctx context.Context) (*ChainAnalysisBlockResponse, error) {
	var resp *ChainAnalysisBlockResponse
	return resp, e.SendHTTPRequest(ctx, exchange.ChainAnalysis, "block/latest", &resp)
}

// GetBlockChainAnalysis returns block information by block hash from bitflyer chain
// analysis system
func (e *Exchange) GetBlockChainAnalysis(ctx context.Context, blockHash string) (*ChainAnalysisBlockResponse, error) {
	var resp *ChainAnalysisBlockResponse
	return resp, e.SendHTTPRequest(ctx, exchange.ChainAnalysis, "block/"+blockHash, &resp)
}

// GetBlockByHeightChainAnalysis returns the block information by height from bitflyer chain
// analysis system
func (e *Exchange) GetBlockByHeightChainAnalysis(ctx context.Context, height uint64) (*ChainAnalysisBlockResponse, error) {
	var resp *ChainAnalysisBlockResponse
	return resp, e.SendHTTPRequest(ctx, exchange.ChainAnalysis, "block/height/"+strconv.FormatUint(height, 10), &resp)
}

// GetTransactionByHashChainAnalysis returns transaction information by txHash from
// bitflyer chain analysis system
func (e *Exchange) GetTransactionByHashChainAnalysis(ctx context.Context, txHash string) (*ChainAnalysisTransactionResponse, error) {
	var resp *ChainAnalysisTransactionResponse
	return resp, e.SendHTTPRequest(ctx, exchange.ChainAnalysis, "tx/"+txHash, &resp)
}

// GetAddressInfoChainAnalysis returns balance information for an address from bitflyer
// chain analysis system
func (e *Exchange) GetAddressInfoChainAnalysis(ctx context.Context, address string) (*ChainAnalysisAddressResponse, error) {
	var resp *ChainAnalysisAddressResponse
	return resp, e.SendHTTPRequest(ctx, exchange.ChainAnalysis, "address/"+address, &resp)
}

// GetMarkets returns market information
func (e *Exchange) GetMarkets(ctx context.Context) ([]MarketInfo, error) {
	var resp []MarketInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, pubGetMarkets, &resp)
}

// GetOrderBook returns market orderbook depth
func (e *Exchange) GetOrderBook(ctx context.Context, symbol string) (Orderbook, error) {
	var resp Orderbook
	v := url.Values{}
	v.Set("product_code", symbol)

	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, pubGetBoard+"?"+v.Encode(), &resp)
}

// GetTicker returns ticker information
func (e *Exchange) GetTicker(ctx context.Context, symbol string) (Ticker, error) {
	var resp Ticker
	v := url.Values{}
	v.Set("product_code", symbol)
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, pubGetTicker+"?"+v.Encode(), &resp)
}

// GetExecutionHistory returns past trades that were executed on the market
func (e *Exchange) GetExecutionHistory(ctx context.Context, symbol string) ([]ExecutedTrade, error) {
	var resp []ExecutedTrade
	v := url.Values{}
	v.Set("product_code", symbol)

	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, pubGetExecutionHistory+"?"+v.Encode(), &resp)
}

// GetExchangeStatus returns exchange status information
func (e *Exchange) GetExchangeStatus(ctx context.Context) (string, error) {
	resp := make(map[string]string)
	err := e.SendHTTPRequest(ctx, exchange.RestSpot, pubGetHealth, &resp)
	if err != nil {
		return "", err
	}

	switch resp["status"] {
	case "BUSY":
		return "the exchange is experiencing high traffic", nil
	case "VERY BUSY":
		return "the exchange is experiencing heavy traffic", nil
	case "SUPER BUSY":
		return "the exchange is experiencing extremely heavy traffic. There is a possibility that orders will fail or be processed after a delay.", nil
	case "STOP":
		return "STOP", errors.New("the exchange has been stopped. Orders will not be accepted")
	}

	return "NORMAL", nil
}

// GetChats returns trollbox chat log
// Note: returns vary from instant to infinity
func (e *Exchange) GetChats(ctx context.Context, fromDate string) ([]ChatLog, error) {
	var resp []ChatLog
	v := url.Values{}
	v.Set("from_date", fromDate)
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, pubGetChats+"?"+v.Encode(), &resp)
}

// GetPermissions returns current permissions for associated with your API
// keys
func (e *Exchange) GetPermissions() {
	// Needs to be updated
}

// GetAccountBalance returnsthe full list of account funds
func (e *Exchange) GetAccountBalance() {
	// Needs to be updated
}

// GetMarginStatus returns current margin status
func (e *Exchange) GetMarginStatus() {
	// Needs to be updated
}

// GetCollateralAccounts returns a full list of collateralised accounts
func (e *Exchange) GetCollateralAccounts() {
	// Needs to be updated
}

// GetCryptoDepositAddress returns an address for cryptocurrency deposits
func (e *Exchange) GetCryptoDepositAddress() {
	// Needs to be updated
}

// GetDepositHistory returns a full history of deposits
func (e *Exchange) GetDepositHistory() {
	// Needs to be updated
}

// GetTransactionHistory returns a full history of transactions
func (e *Exchange) GetTransactionHistory() {
	// Needs to be updated
}

// GetBankAccSummary returns a full list of bank accounts assoc. with your keys
func (e *Exchange) GetBankAccSummary() {
	// Needs to be updated
}

// GetCashDeposits returns a full list of cash deposits to the exchange
func (e *Exchange) GetCashDeposits() {
	// Needs to be updated
}

// WithdrawFunds withdraws funds to a certain bank
func (e *Exchange) WithdrawFunds() {
	// Needs to be updated
}

// GetDepositCancellationHistory returns the cancellation history of deposits
func (e *Exchange) GetDepositCancellationHistory() {
	// Needs to be updated
}

// SendOrder creates new order
func (e *Exchange) SendOrder() {
	// Needs to be updated
}

// CancelExistingOrder cancels an order
func (e *Exchange) CancelExistingOrder() {
	// Needs to be updated
}

// SendParentOrder sends a special order
func (e *Exchange) SendParentOrder() {
	// Needs to be updated
}

// CancelParentOrder cancels a special order
func (e *Exchange) CancelParentOrder() {
	// Needs to be updated
}

// CancelAllExistingOrders cancels all orders on the exchange
func (e *Exchange) CancelAllExistingOrders() {
	// Needs to be updated
}

// GetAllOrders returns a list of all orders
func (e *Exchange) GetAllOrders() {
	// Needs to be updated
}

// GetParentOrders returns a list of all parent orders
func (e *Exchange) GetParentOrders() {
	// Needs to be updated
}

// GetParentOrderDetails returns a detailing of a parent order
func (e *Exchange) GetParentOrderDetails() {
	// Needs to be updated
}

// GetExecutions returns execution details
func (e *Exchange) GetExecutions() {
	// Needs to be updated
}

// GetOpenInterestData returns a summary of open interest
func (e *Exchange) GetOpenInterestData() {
	// Needs to be updated
}

// GetMarginChange returns collateral history
func (e *Exchange) GetMarginChange() {
	// Needs to be updated
}

// GetTradingCommission returns trading commission
func (e *Exchange) GetTradingCommission() {
	// Needs to be updated
}

// SendHTTPRequest sends an unauthenticated request
func (e *Exchange) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	item := &request.Item{
		Method:                 http.MethodGet,
		Path:                   endpoint + path,
		Result:                 result,
		Verbose:                e.Verbose,
		HTTPDebugging:          e.HTTPDebugging,
		HTTPRecording:          e.HTTPRecording,
		HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
	}
	return e.SendPayload(ctx, request.UnAuth, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest)
}

// SendAuthHTTPRequest sends an authenticated HTTP request
// Note: HTTP not done due to incorrect account privileges, please open a PR
// if you have access and update the authenticated requests
// TODO: Fill out this function once API access is obtained
func (e *Exchange) SendAuthHTTPRequest() {
	//nolint:gocritic // code example
	// headers := make(map[string]string)
	// headers["ACCESS-KEY"] = b.API.Credentials.Key
	// headers["ACCESS-TIMESTAMP"] = strconv.FormatInt(time.Now().UnixNano(), 10)
}

// GetFee returns an estimate of fee based on type of transaction
// TODO: Figure out the weird fee structure. Do we use Bitcoin Easy Exchange,Lightning Spot,Bitcoin Market,Lightning FX/Futures ???
func (e *Exchange) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64

	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	case exchange.InternationalBankDepositFee:
		fee = getDepositFee(feeBuilder.BankTransactionType, feeBuilder.FiatCurrency)
	case exchange.InternationalBankWithdrawalFee:
		fee = getWithdrawalFee(feeBuilder.BankTransactionType, feeBuilder.FiatCurrency, feeBuilder.Amount)
	case exchange.OfflineTradeFee:
		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	if fee < 0 {
		fee = 0
	}
	return fee, nil
}

// calculateTradingFee returns fee when performing a trade
func calculateTradingFee(price, amount float64) float64 {
	// bitflyer has fee tiers, but does not disclose them via API, so the largest has to be assumed
	return 0.0012 * price * amount
}

func getDepositFee(bankTransactionType exchange.InternationalBankTransactionType, c currency.Code) (fee float64) {
	if bankTransactionType == exchange.WireTransfer {
		if c.Item == currency.JPY.Item {
			fee = 324
		}
	}
	return fee
}

func getWithdrawalFee(bankTransactionType exchange.InternationalBankTransactionType, c currency.Code, amount float64) (fee float64) {
	if bankTransactionType == exchange.WireTransfer {
		if c.Item == currency.JPY.Item {
			if amount < 30000 {
				fee = 540
			} else {
				fee = 756
			}
		}
	}
	return fee
}
