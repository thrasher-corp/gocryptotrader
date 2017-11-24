package itbit

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

const (
	itbitAPIURL         = "https://api.itbit.com/v1"
	itbitAPIVersion     = "1"
	itbitMarkets        = "markets"
	itbitOrderbook      = "order_book"
	itbitTicker         = "ticker"
	itbitWallets        = "wallets"
	itbitBalances       = "balances"
	itbitTrades         = "trades"
	itbitFundingHistory = "funding_history"
	itbitOrders         = "orders"
	itbitCryptoDeposits = "cryptocurrency_deposits"
	itbitWalletTransfer = "wallet_transfers"
)

// ItBit is the overarching type across the ItBit package
type ItBit struct {
	exchange.Base
}

// SetDefaults sets the defaults for the exchange
func (i *ItBit) SetDefaults() {
	i.Name = "ITBIT"
	i.Enabled = false
	i.MakerFee = -0.10
	i.TakerFee = 0.50
	i.Verbose = false
	i.Websocket = false
	i.RESTPollingDelay = 10
	i.RequestCurrencyPairFormat.Delimiter = ""
	i.RequestCurrencyPairFormat.Uppercase = true
	i.ConfigCurrencyPairFormat.Delimiter = ""
	i.ConfigCurrencyPairFormat.Uppercase = true
	i.AssetTypes = []string{ticker.Spot}
}

// Setup sets the exchange parameters from exchange config
func (i *ItBit) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		i.SetEnabled(false)
	} else {
		i.Enabled = true
		i.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		i.SetAPIKeys(exch.APIKey, exch.APISecret, exch.ClientID, false)
		i.RESTPollingDelay = exch.RESTPollingDelay
		i.Verbose = exch.Verbose
		i.Websocket = exch.Websocket
		i.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		i.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		i.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
		err := i.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = i.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
	}
}

// GetFee returns the maker or taker fee
func (i *ItBit) GetFee(maker bool) float64 {
	if maker {
		return i.MakerFee
	}
	return i.TakerFee
}

// GetTicker returns ticker info for a specified market.
// currencyPair - example "XBTUSD" "XBTSGD" "XBTEUR"
func (i *ItBit) GetTicker(currencyPair string) (Ticker, error) {
	var response Ticker
	path := fmt.Sprintf("%s/%s/%s/%s", itbitAPIURL, itbitMarkets, currencyPair, itbitTicker)

	return response,
		common.SendHTTPGetRequest(path, true, i.Verbose, &response)
}

// GetOrderbook returns full order book for the specified market.
// currencyPair - example "XBTUSD" "XBTSGD" "XBTEUR"
func (i *ItBit) GetOrderbook(currencyPair string) (OrderbookResponse, error) {
	response := OrderbookResponse{}
	path := fmt.Sprintf("%s/%s/%s/%s", itbitAPIURL, itbitMarkets, currencyPair, itbitOrderbook)

	return response,
		common.SendHTTPGetRequest(path, true, i.Verbose, &response)
}

// GetTradeHistory returns recent trades for a specified market.
//
// currencyPair - example "XBTUSD" "XBTSGD" "XBTEUR"
// timestamp - matchNumber, only executions after this will be returned
func (i *ItBit) GetTradeHistory(currencyPair, timestamp string) (Trades, error) {
	response := Trades{}
	req := "trades?since=" + timestamp
	path := fmt.Sprintf("%s/%s/%s/%s", itbitAPIURL, itbitMarkets, currencyPair, req)

	return response,
		common.SendHTTPGetRequest(path, true, i.Verbose, &response)
}

// GetWallets returns information about all wallets associated with the account.
//
// params --
// 					page - [optional] page to return example 1. default 1
//					perPage - [optional] items per page example 50, default 50 max 50
func (i *ItBit) GetWallets(params url.Values) ([]Wallet, error) {
	resp := []Wallet{}
	params.Set("userId", i.ClientID)
	path := fmt.Sprintf("/%s?%s", itbitWallets, params.Encode())

	return resp, i.SendAuthenticatedHTTPRequest("GET", path, nil, &resp)
}

// CreateWallet creates a new wallet with a specified name.
func (i *ItBit) CreateWallet(walletName string) (Wallet, error) {
	resp := Wallet{}
	params := make(map[string]interface{})
	params["userId"] = i.ClientID
	params["name"] = walletName

	return resp,
		i.SendAuthenticatedHTTPRequest("POST", "/"+itbitWallets, params, &resp)
}

// GetWallet returns wallet information by walletID
func (i *ItBit) GetWallet(walletID string) (Wallet, error) {
	resp := Wallet{}
	path := fmt.Sprintf("/%s/%s", itbitWallets, walletID)

	return resp, i.SendAuthenticatedHTTPRequest("GET", path, nil, &resp)
}

// GetWalletBalance returns balance information for a specific currency in a
// wallet.
func (i *ItBit) GetWalletBalance(walletID, currency string) (Balance, error) {
	resp := Balance{}
	path := fmt.Sprintf("/%s/%s/%s/%s", itbitWallets, walletID, itbitBalances, currency)

	return resp, i.SendAuthenticatedHTTPRequest("GET", path, nil, &resp)
}

// GetWalletTrades returns all trades for a specified wallet.
func (i *ItBit) GetWalletTrades(walletID string, params url.Values) (Records, error) {
	resp := Records{}
	url := fmt.Sprintf("/%s/%s/%s", itbitWallets, walletID, itbitTrades)
	path := common.EncodeURLValues(url, params)

	return resp, i.SendAuthenticatedHTTPRequest("GET", path, nil, &resp)
}

// GetFundingHistory returns all funding history for a specified wallet.
func (i *ItBit) GetFundingHistory(walletID string, params url.Values) (FundingRecords, error) {
	resp := FundingRecords{}
	url := fmt.Sprintf("/%s/%s/%s", itbitWallets, walletID, itbitFundingHistory)
	path := common.EncodeURLValues(url, params)

	return resp, i.SendAuthenticatedHTTPRequest("GET", path, nil, &resp)
}

// PlaceOrder places a new order
func (i *ItBit) PlaceOrder(walletID, side, orderType, currency string, amount, price float64, instrument, clientRef string) (Order, error) {
	resp := Order{}
	path := fmt.Sprintf("/%s/%s/%s", itbitWallets, walletID, itbitOrders)

	params := make(map[string]interface{})
	params["side"] = side
	params["type"] = orderType
	params["currency"] = currency
	params["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	params["price"] = strconv.FormatFloat(price, 'f', -1, 64)
	params["instrument"] = instrument

	if clientRef != "" {
		params["clientOrderIdentifier"] = clientRef
	}

	return resp, i.SendAuthenticatedHTTPRequest("POST", path, params, &resp)
}

// GetOrder returns an order by id.
func (i *ItBit) GetOrder(walletID string, params url.Values) (Order, error) {
	resp := Order{}
	url := fmt.Sprintf("/%s/%s/%s", itbitWallets, walletID, itbitOrders)
	path := common.EncodeURLValues(url, params)

	return resp, i.SendAuthenticatedHTTPRequest("GET", path, nil, &resp)
}

// CancelOrder cancels and open order. *This is not a guarantee that the order
// has been cancelled!*
func (i *ItBit) CancelOrder(walletID, orderID string) error {
	path := fmt.Sprintf("/%s/%s/%s/%s", itbitWallets, walletID, itbitOrders, orderID)

	return i.SendAuthenticatedHTTPRequest("DELETE", path, nil, nil)
}

// GetDepositAddress returns a deposit address to send cryptocurrency to.
func (i *ItBit) GetDepositAddress(walletID, currency string) (CryptoCurrencyDeposit, error) {
	resp := CryptoCurrencyDeposit{}
	path := fmt.Sprintf("/%s/%s/%s", itbitWallets, walletID, itbitCryptoDeposits)
	params := make(map[string]interface{})
	params["currency"] = currency

	return resp, i.SendAuthenticatedHTTPRequest("POST", path, params, &resp)
}

// WalletTransfer transfers funds between wallets.
func (i *ItBit) WalletTransfer(walletID, sourceWallet, destWallet string, amount float64, currency string) (WalletTransfer, error) {
	resp := WalletTransfer{}
	path := fmt.Sprintf("/%s/%s/%s", itbitWallets, walletID, itbitWalletTransfer)

	params := make(map[string]interface{})
	params["sourceWalletId"] = sourceWallet
	params["destinationWalletId"] = destWallet
	params["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	params["currencyCode"] = currency

	return resp, i.SendAuthenticatedHTTPRequest("POST", path, params, &resp)
}

// SendAuthenticatedHTTPRequest sends an authenticated request to itBit
func (i *ItBit) SendAuthenticatedHTTPRequest(method string, path string, params map[string]interface{}, result interface{}) error {
	if !i.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, i.Name)
	}

	request := make(map[string]interface{})
	url := itbitAPIURL + path

	if params != nil {
		for key, value := range params {
			request[key] = value
		}
	}

	PayloadJSON := []byte("")
	var err error

	if params != nil {

		PayloadJSON, err = common.JSONEncode(request)
		if err != nil {
			return err
		}

		if i.Verbose {
			log.Printf("Request JSON: %s\n", PayloadJSON)
		}
	}

	nonce := i.Nonce.GetValue(i.Name, false).String()
	timestamp := strconv.FormatInt(time.Now().UnixNano()/1000000, 10)

	message, err := common.JSONEncode([]string{method, url, string(PayloadJSON), nonce, timestamp})
	if err != nil {
		return err
	}

	hash := common.GetSHA256([]byte(nonce + string(message)))
	hmac := common.GetHMAC(common.HashSHA512, []byte(url+string(hash)), []byte(i.APISecret))
	signature := common.Base64Encode(hmac)

	headers := make(map[string]string)
	headers["Authorization"] = i.ClientID + ":" + signature
	headers["X-Auth-Timestamp"] = timestamp
	headers["X-Auth-Nonce"] = nonce
	headers["Content-Type"] = "application/json"

	resp, err := common.SendHTTPRequest(method, url, headers, bytes.NewBuffer([]byte(PayloadJSON)))
	if err != nil {
		return err
	}

	if i.Verbose {
		log.Printf("Received raw: \n%s\n", resp)
	}

	errCapture := GeneralReturn{}
	if err := common.JSONDecode([]byte(resp), &errCapture); err == nil {
		return errors.New(errCapture.Description)
	}

	return common.JSONDecode([]byte(resp), result)
}
