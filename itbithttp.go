package main

import (
	"net/http"
	"strconv"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"encoding/json"
	"io/ioutil"
	"log"
	"fmt"
)

const (
	ITBIT_API_URL = "https://api.itbit.com/v1/"
)

type ItBit struct {
	ClientKey, APISecret, UserID string
}

func (i *ItBit) GetTicker(currency string) (bool) {
	path := ITBIT_API_URL + "/markets/" + currency + "/ticker"
	err := SendHTTPRequest(path, true, nil)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func (i *ItBit) GetOrderbook(currency string) (bool) {
	path := ITBIT_API_URL + "/markets/" + currency + "/orders"
	err := SendHTTPRequest(path , true, nil)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func (i *ItBit) GetTradeHistory(currency, timestamp string) (bool) {
	req := "/trades?since=" + timestamp
	err := SendHTTPRequest(ITBIT_API_URL + "markets/" + currency + req, true, nil)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func (i *ItBit) GetWallets(page int64, perPage int64, userID string) {
	path := ITBIT_API_URL + "wallets/"
	params := make(map[string]interface{})
	params["page"] = strconv.FormatInt(page, 10)
	params["perPage"] = strconv.FormatInt(perPage, 10)
	params["userID"] = userID

	err := i.SendAuthenticatedHTTPRequest("GET", path, params)

	if err != nil {
		fmt.Println(err)
	}
}

func (i *ItBit) GetWallet(walletID string) {
	path := ITBIT_API_URL + "/wallets/" + walletID
	err := i.SendAuthenticatedHTTPRequest("GET", path, nil)

	if err != nil {
		fmt.Println(err)
	}
}

func (i *ItBit) GetWalletBalance(walletID, currency string) {
	path := ITBIT_API_URL + "/wallets/ " + walletID +  "/balances/" + currency
	err := i.SendAuthenticatedHTTPRequest("GET", path, nil)

	if err != nil {
		fmt.Println(err)
	}
}

func (i *ItBit) GetWalletTrades(walletID string, page int64, perPage int64, rangeEnd int64, rangeStart int64) {
	path := ITBIT_API_URL + "/wallets/" + walletID + "/trades"
	params := make(map[string]interface{})
	params["page"] = strconv.FormatInt(page, 10)
	params["perPage"] = strconv.FormatInt(perPage, 10)
	params["rangeEnd"] = strconv.FormatInt(page, 10)
	params["rangeStart"] = strconv.FormatInt(perPage, 10)

	err := i.SendAuthenticatedHTTPRequest("GET", path, params)

	if err != nil {
		fmt.Println(err)
	}
}

func (i *ItBit) GetWalletOrders(walletID string, instrument string, page int64, perPage int64, status string) {
	path := ITBIT_API_URL + "/wallets/" + walletID + "/orders"
	params := make(map[string]interface{})
	params["instrument"] = instrument
	params["page"] = strconv.FormatInt(page, 10)
	params["perPage"] = strconv.FormatInt(perPage, 10)
	params["status"] = status

	err := i.SendAuthenticatedHTTPRequest("GET", path, params)

	if err != nil {
		fmt.Println(err)
	}
}

func (i *ItBit) PlaceWalletOrder(walletID, side, orderType, currency string, amount, price float64, instrument string) {
	path := ITBIT_API_URL + "/wallets/" + walletID + "/orders"
	params := make(map[string]interface{})
	params["side"] = side
	params["type"] = orderType
	params["currency"] = currency
	params["amount"] = strconv.FormatFloat(amount, 'f', 8, 64)
	params["price"] = strconv.FormatFloat(price, 'f', 2, 64)
	params["instrument"] = instrument

	err := i.SendAuthenticatedHTTPRequest("POST", path, params)

	if err != nil {
		fmt.Println(err)
	}
}

func (i *ItBit) GetWalletOrder(walletID, orderID string) {
	path := ITBIT_API_URL + "/wallets/" + walletID + "/orders/" + orderID
	err := i.SendAuthenticatedHTTPRequest("GET", path, nil)

	if err != nil {
		fmt.Println(err)
	}
}

func (i *ItBit) CancelWalletOrder(walletID, orderID string) {
	path := ITBIT_API_URL + "/wallets/" + walletID + "/orders/" + orderID
	err := i.SendAuthenticatedHTTPRequest("DELETE", path, nil)

	if err != nil {
		fmt.Println(err)
	}
}

func (i *ItBit) PlaceWithdrawalRequest(walletID, currency, address string, amount float64) {
	path := ITBIT_API_URL + "/wallets/" + walletID + "/cryptocurrency_withdrawals"
	params := make(map[string]interface{})
	params["currency"] = currency
	params["amount"] = strconv.FormatFloat(amount, 'f', 8, 64)
	params["address"] = address

	err := i.SendAuthenticatedHTTPRequest("POST", path, params)

	if err != nil {
		fmt.Println(err)
	}
}

func (i *ItBit) GetDepositAddress(walletID, currency string) {
	path := ITBIT_API_URL + "/wallets/" + walletID + "/cryptocurrency_deposits"
	params := make(map[string]interface{})
	params["currency"] = currency

	err := i.SendAuthenticatedHTTPRequest("POST", path, params)

	if err != nil {
		fmt.Println(err)
	}
}

func (i *ItBit) WalletTransfer(walletID, sourceWallet, destWallet string, amount float64, currency string) {
	path := ITBIT_API_URL + "/wallets/" + walletID + "/wallet_transfers"
	params := make(map[string]interface{})
	params["sourceWalletId"] = sourceWallet
	params["destinationWalletId"] = destWallet
	params["amount"] = strconv.FormatFloat(amount, 'f', 8, 64)
	params["currencyCode"] = currency

	err := i.SendAuthenticatedHTTPRequest("POST", path, params)

	if err != nil {
		fmt.Println(err)
	}
}

func (i *ItBit) SendAuthenticatedHTTPRequest(method string, path string, params map[string]interface{}) (err error) {
	request := make(map[string]interface{})
	nonce := strconv.FormatInt(time.Now().UnixNano(), 10)
	request["nonce"] = nonce
	request["timestamp"] = nonce

	if params != nil {
		for key, value:= range params {
			request[key] = value
		}
	}

	PayloadJson, err := json.Marshal(request)
	log.Printf("Request JSON: %s\n", PayloadJson)

	if err != nil {
		return errors.New("SendAuthenticatedHTTPRequest: Unable to JSON request")
	}

	hmac := hmac.New(sha512.New, []byte(i.APISecret))
	hmac.Write([]byte(nonce + string(PayloadJson)))
	hex := hex.EncodeToString(hmac.Sum(nil))
	signature := base64.StdEncoding.EncodeToString([]byte(hex))
	req, err := http.NewRequest(method, path, strings.NewReader(""))

	req.Header.Add("Authorization", i.ClientKey + ":" + signature)
	req.Header.Add("X-Auth-Timestamp", nonce)
	req.Header.Add("X-Auth-Nonce", nonce)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return errors.New("SendAuthenticatedHTTPRequest: Unable to send request")
	}

	contents, _ := ioutil.ReadAll(resp.Body)
	log.Printf("Recieved raw: %s\n", string(contents))
	resp.Body.Close()
	return nil
}