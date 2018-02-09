package bitflyer

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

const (
	// Bitflyer chain analysis endpoints
	// APIURL
	chainAnalysis = "https://chainflyer.bitflyer.jp/v1/"

	// Public endpoints for chain analysis
	latestBlock        = "block/latest"
	blockByBlockHash   = "block/"
	blockByBlockHeight = "block/height/"
	transaction        = "tx/"
	address            = "address/"

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

	// Autheticated Endpoints
	privGetPermissions             = "/me/getpermissions"
	privGetBalance                 = "/me/getbalance"
	privMarginStatus               = "/me/getcollateral"
	privGetCollateralAcc           = "/me/getcollateralaccounts"
	privGetDepositAddress          = "/me/getaddresses"
	privDepositHistory             = "/me/getcoinins"
	privTransactionHistory         = "/me/getcoinouts"
	privBankAccSummary             = "/me/getbankaccounts"
	privGetDeposits                = "/me/getdeposits"
	privWithdraw                   = "/me/withdraw"
	privDepositCancellationHistory = "/me/getwithdrawals"
	privSendOrder                  = "/me/sendchildorder"
	privCancelOrder                = "/me/cancelchildorder"
	privParentOrder                = "/me/sendparentorder"
	privCancelParentOrder          = "/me/cancelparentorder"
	privCancelOrders               = "/me/cancelallchildorders"
	privListOrders                 = "/me/getchildorders"
	privListParentOrders           = "/me/getparentorders"
	privParentOrderDetails         = "/me/getparentorder"
	privExecutions                 = "/me/getexecutions"
	privOpenInterest               = "/me/getpositions"
	privMarginChange               = "/me/getcollateralhistory"
	privTradingCommission          = "/me/gettradingcommission"
)

// Bitflyer is the overarching type across this package
type Bitflyer struct {
	exchange.Base
}

// SetDefaults sets the basic defaults for Bitflyer
func (b *Bitflyer) SetDefaults() {
	b.Name = "Bitflyer"
	b.Enabled = false
	b.Verbose = false
	b.Websocket = false
	b.RESTPollingDelay = 10
	b.RequestCurrencyPairFormat.Delimiter = "_"
	b.RequestCurrencyPairFormat.Uppercase = true
	b.ConfigCurrencyPairFormat.Delimiter = "_"
	b.ConfigCurrencyPairFormat.Uppercase = true
	b.AssetTypes = []string{ticker.Spot}
}

// Setup takes in the supplied exchange configuration details and sets params
func (b *Bitflyer) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		b.SetEnabled(false)
	} else {
		b.Enabled = true
		b.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		b.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		b.RESTPollingDelay = exch.RESTPollingDelay
		b.Verbose = exch.Verbose
		b.Websocket = exch.Websocket
		b.APIUrl = japanURL
		b.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		b.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		b.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
		err := b.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = b.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
	}
}

// GetLatestBlockCA returns the latest block information from bitflyer chain
// analysis system
func (b *Bitflyer) GetLatestBlockCA() (ChainAnalysisBlock, error) {
	var resp ChainAnalysisBlock
	path := fmt.Sprintf("%s%s", chainAnalysis, latestBlock)

	return resp, common.SendHTTPGetRequest(path, true, b.Verbose, &resp)
}

// GetBlockCA returns block information by blockhash from bitflyer chain
// analysis system
func (b *Bitflyer) GetBlockCA(blockhash string) (ChainAnalysisBlock, error) {
	var resp ChainAnalysisBlock
	path := fmt.Sprintf("%s%s%s", chainAnalysis, blockByBlockHash, blockhash)

	return resp, common.SendHTTPGetRequest(path, true, b.Verbose, &resp)
}

// GetBlockbyHeightCA returns the block information by height from bitflyer chain
// analysis system
func (b *Bitflyer) GetBlockbyHeightCA(height int64) (ChainAnalysisBlock, error) {
	var resp ChainAnalysisBlock
	path := fmt.Sprintf("%s%s%s", chainAnalysis, blockByBlockHeight, strconv.FormatInt(height, 10))

	return resp, common.SendHTTPGetRequest(path, true, b.Verbose, &resp)
}

// GetTransactionByHashCA returns transaction information by txHash from
// bitflyer chain analysis system
func (b *Bitflyer) GetTransactionByHashCA(txHash string) (ChainAnalysisTransaction, error) {
	var resp ChainAnalysisTransaction
	path := fmt.Sprintf("%s%s%s", chainAnalysis, transaction, txHash)

	return resp, common.SendHTTPGetRequest(path, true, b.Verbose, &resp)
}

// GetAddressInfoCA returns balance information for address by addressln string
// from bitflyer chain analysis system
func (b *Bitflyer) GetAddressInfoCA(addressln string) (ChainAnalysisAddress, error) {
	var resp ChainAnalysisAddress
	path := fmt.Sprintf("%s%s%s", chainAnalysis, address, addressln)

	return resp, common.SendHTTPGetRequest(path, true, b.Verbose, &resp)
}

// GetMarkets returns market information
func (b *Bitflyer) GetMarkets() ([]MarketInfo, error) {
	var resp []MarketInfo
	path := fmt.Sprintf("%s%s", b.APIUrl, pubGetMarkets)

	return resp, common.SendHTTPGetRequest(path, true, b.Verbose, &resp)
}

// GetOrderBook returns market orderbook depth
func (b *Bitflyer) GetOrderBook(symbol string) (Orderbook, error) {
	var resp Orderbook
	v := url.Values{}
	v.Set("product_code", symbol)
	path := fmt.Sprintf("%s%s?%s", japanURL, pubGetBoard, v.Encode())

	return resp, common.SendHTTPGetRequest(path, true, b.Verbose, &resp)
}

// GetTicker returns ticker information
func (b *Bitflyer) GetTicker(symbol string) (Ticker, error) {
	var resp Ticker
	v := url.Values{}
	v.Set("product_code", symbol)
	path := fmt.Sprintf("%s%s?%s", japanURL, pubGetTicker, v.Encode())

	return resp, common.SendHTTPGetRequest(path, true, b.Verbose, &resp)
}

// GetExecutionHistory returns past trades that were executed on the market
func (b *Bitflyer) GetExecutionHistory(symbol string) ([]ExecutedTrade, error) {
	var resp []ExecutedTrade
	v := url.Values{}
	v.Set("product_code", symbol)
	path := fmt.Sprintf("%s%s?%s", japanURL, pubGetExecutionHistory, v.Encode())

	return resp, common.SendHTTPGetRequest(path, true, b.Verbose, &resp)
}

// GetExchangeStatus returns exchange status information
func (b *Bitflyer) GetExchangeStatus() (string, error) {
	resp := make(map[string]string)

	path := fmt.Sprintf("%s%s", b.APIUrl, pubGetHealth)

	err := common.SendHTTPGetRequest(path, true, b.Verbose, &resp)
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
// Note: returns vary from instant to infinty
func (b *Bitflyer) GetChats(FromDate string) ([]ChatLog, error) {
	var resp []ChatLog
	v := url.Values{}
	v.Set("from_date", FromDate)
	path := fmt.Sprintf("%s%s?%s", b.APIUrl, pubGetChats, v.Encode())

	return resp, common.SendHTTPGetRequest(path, true, b.Verbose, &resp)
}

// GetPermissions returns current permissions for associated with your API
// keys
func (b *Bitflyer) GetPermissions() {
	// Needs to be updated
}

// GetAccountBalance returnsthe full list of account funds
func (b *Bitflyer) GetAccountBalance() {
	// Needs to be updated
}

// GetMarginStatus returns current margin status
func (b *Bitflyer) GetMarginStatus() {
	// Needs to be updated
}

// GetCollateralAccounts returns a full list of collateralised accounts
func (b *Bitflyer) GetCollateralAccounts() {
	// Needs to be updated
}

// GetDepositAddress returns an address for cryptocurrency deposits
func (b *Bitflyer) GetDepositAddress() {
	// Needs to be updated
}

// GetDepositHistory returns a full history of deposits
func (b *Bitflyer) GetDepositHistory() {
	// Needs to be updated
}

// GetTransactionHistory returns a full history of transactions
func (b *Bitflyer) GetTransactionHistory() {
	// Needs to be updated
}

// GetBankAccSummary returns a full list of bank accounts assoc. with your keys
func (b *Bitflyer) GetBankAccSummary() {
	// Needs to be updated
}

// GetCashDeposits returns a full list of cash deposits to the exchange
func (b *Bitflyer) GetCashDeposits() {
	// Needs to be updated
}

// WithdrawFunds withdraws funds to a certain bank
func (b *Bitflyer) WithdrawFunds() {
	// Needs to be updated
}

// GetDepositCancellationHistory returns the cancellation history of deposits
func (b *Bitflyer) GetDepositCancellationHistory() {
	// Needs to be updated
}

// SendOrder creates new order
func (b *Bitflyer) SendOrder() {
	// Needs to be updated
}

// CancelOrder cancels an order
func (b *Bitflyer) CancelOrder() {
	// Needs to be updated
}

// SendParentOrder sends a special order
func (b *Bitflyer) SendParentOrder() {
	// Needs to be updated
}

// CancelParentOrder cancels a special order
func (b *Bitflyer) CancelParentOrder() {
	// Needs to be updated
}

// CancelAllOrders cancels all orders on the exchange
func (b *Bitflyer) CancelAllOrders() {
	// Needs to be updated
}

// GetAllOrders returns a list of all orders
func (b *Bitflyer) GetAllOrders() {
	// Needs to be updated
}

// GetParentOrders returns a list of all parent orders
func (b *Bitflyer) GetParentOrders() {
	// Needs to be updated
}

// GetParentOrderDetails returns a detailing of a parent order
func (b *Bitflyer) GetParentOrderDetails() {
	// Needs to be updated
}

// GetExecutions returns execution details
func (b *Bitflyer) GetExecutions() {
	// Needs to be updated
}

// GetOpenInterest returns a summary of open interest
func (b *Bitflyer) GetOpenInterest() {
	// Needs to be updated
}

// GetMarginChange returns collateral history
func (b *Bitflyer) GetMarginChange() {
	// Needs to be updated
}

// GetTradingCommission returns trading commission
func (b *Bitflyer) GetTradingCommission() {
	// Needs to be updated
}

// SendAuthHTTPRequest sends an authenticated HTTP request
// Note: HTTP not done due to incorrect account privileges, please open a PR
// if you have access and update the authenticated requests
func (b *Bitflyer) SendAuthHTTPRequest(path string, params url.Values, result interface{}) {
	headers := make(map[string]string)
	headers["ACCESS-KEY"] = b.APIKey
	headers["ACCESS-TIMESTAMP"] = strconv.FormatInt(int64(time.Now().UnixNano()), 10)
}
