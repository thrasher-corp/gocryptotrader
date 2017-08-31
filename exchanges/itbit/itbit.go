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
	ITBIT_API_URL     = "https://api.itbit.com/v1"
	ITBIT_API_VERSION = "1"
)

type ItBit struct {
	exchange.Base
}

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

func (i *ItBit) GetFee(maker bool) float64 {
	if maker {
		return i.MakerFee
	}
	return i.TakerFee
}

func (i *ItBit) GetTicker(currency string) (Ticker, error) {
	path := ITBIT_API_URL + "/markets/" + currency + "/ticker"
	var itbitTicker Ticker
	err := common.SendHTTPGetRequest(path, true, &itbitTicker)
	if err != nil {
		return Ticker{}, err
	}
	return itbitTicker, nil
}

func (i *ItBit) GetOrderbook(currency string) (OrderbookResponse, error) {
	response := OrderbookResponse{}
	path := ITBIT_API_URL + "/markets/" + currency + "/order_book"
	err := common.SendHTTPGetRequest(path, true, &response)
	if err != nil {
		return OrderbookResponse{}, err
	}
	return response, nil
}

func (i *ItBit) GetTradeHistory(currency, timestamp string) bool {
	req := "/trades?since=" + timestamp
	err := common.SendHTTPGetRequest(ITBIT_API_URL+"markets/"+currency+req, true, nil)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func (i *ItBit) GetWallets(params url.Values) {
	params.Set("userId", i.ClientID)
	path := "/wallets?" + params.Encode()

	err := i.SendAuthenticatedHTTPRequest("GET", path, nil)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) CreateWallet(walletName string) {
	path := "/wallets"
	params := make(map[string]interface{})
	params["userId"] = i.ClientID
	params["name"] = walletName

	err := i.SendAuthenticatedHTTPRequest("POST", path, params)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) GetWallet(walletID string) {
	path := "/wallets/" + walletID
	err := i.SendAuthenticatedHTTPRequest("GET", path, nil)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) GetWalletBalance(walletID, currency string) {
	path := "/wallets/ " + walletID + "/balances/" + currency
	err := i.SendAuthenticatedHTTPRequest("GET", path, nil)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) GetWalletTrades(walletID string, params url.Values) {
	path := common.EncodeURLValues("/wallets/"+walletID+"/trades", params)
	err := i.SendAuthenticatedHTTPRequest("GET", path, nil)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) GetWalletOrders(walletID string, params url.Values) {
	path := common.EncodeURLValues("/wallets/"+walletID+"/orders", params)
	err := i.SendAuthenticatedHTTPRequest("GET", path, nil)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) PlaceWalletOrder(walletID, side, orderType, currency string, amount, price float64, instrument string, clientRef string) {
	path := "/wallets/" + walletID + "/orders"
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

	err := i.SendAuthenticatedHTTPRequest("POST", path, params)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) GetWalletOrder(walletID, orderID string) {
	path := "/wallets/" + walletID + "/orders/" + orderID
	err := i.SendAuthenticatedHTTPRequest("GET", path, nil)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) CancelWalletOrder(walletID, orderID string) {
	path := "/wallets/" + walletID + "/orders/" + orderID
	err := i.SendAuthenticatedHTTPRequest("DELETE", path, nil)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) PlaceWithdrawalRequest(walletID, currency, address string, amount float64) {
	path := "/wallets/" + walletID + "/cryptocurrency_withdrawals"
	params := make(map[string]interface{})
	params["currency"] = currency
	params["amount"] = amount
	params["address"] = address

	err := i.SendAuthenticatedHTTPRequest("POST", path, params)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) GetDepositAddress(walletID, currency string) {
	path := "/wallets/" + walletID + "/cryptocurrency_deposits"
	params := make(map[string]interface{})
	params["currency"] = currency

	err := i.SendAuthenticatedHTTPRequest("POST", path, params)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) WalletTransfer(walletID, sourceWallet, destWallet string, amount float64, currency string) {
	path := "/wallets/" + walletID + "/wallet_transfers"
	params := make(map[string]interface{})
	params["sourceWalletId"] = sourceWallet
	params["destinationWalletId"] = destWallet
	params["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	params["currencyCode"] = currency

	err := i.SendAuthenticatedHTTPRequest("POST", path, params)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) SendAuthenticatedHTTPRequest(method string, path string, params map[string]interface{}) (err error) {
	if !i.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, i.Name)
	}

	if i.Nonce.Get() == 0 {
		i.Nonce.Set(time.Now().UnixNano())
	} else {
		i.Nonce.Inc()
	}

	request := make(map[string]interface{})
	url := ITBIT_API_URL + path

	if params != nil {
		for key, value := range params {
			request[key] = value
		}
	}

	PayloadJSON := []byte("")

	if params != nil {
		PayloadJSON, err = common.JSONEncode(request)

		if err != nil {
			return errors.New("SendAuthenticatedHTTPRequest: Unable to JSON Marshal request")
		}

		if i.Verbose {
			log.Printf("Request JSON: %s\n", PayloadJSON)
		}
	}

	message, err := common.JSONEncode([]string{method, url, string(PayloadJSON), i.Nonce.String(), i.Nonce.String()[0:13]})
	if err != nil {
		log.Println(err)
		return
	}

	hash := common.GetSHA256([]byte(i.Nonce.String() + string(message)))
	hmac := common.GetHMAC(common.HashSHA512, []byte(url+string(hash)), []byte(i.APISecret))
	signature := common.Base64Encode(hmac)

	headers := make(map[string]string)
	headers["Authorization"] = i.ClientID + ":" + signature
	headers["X-Auth-Timestamp"] = i.Nonce.String()[0:13]
	headers["X-Auth-Nonce"] = i.Nonce.String()
	headers["Content-Type"] = "application/json"

	resp, err := common.SendHTTPRequest(method, url, headers, bytes.NewBuffer([]byte(PayloadJSON)))

	if i.Verbose {
		log.Printf("Received raw: \n%s\n", resp)
	}
	return nil
}
