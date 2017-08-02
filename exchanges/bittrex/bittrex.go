package bittrex

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges"
)

const (
	bittrexAPIURL              = "https://bittrex.com/api"
	bittrexAPIVersion          = "v1.1"
	bittrexMaxOpenOrders       = 500
	bittrexMaxOrderCountPerDay = 200000

	// Returned messages from Bittrex API
	bittrexAddressGenerating      = "ADDRESS_GENERATING"
	bittrexErrorMarketNotProvided = "MARKET_NOT_PROVIDED"
	bittrexErrorInvalidMarket     = "INVALID_MARKET"
	bittrexErrorAPIKeyInvalid     = "APIKEY_INVALID"

	// Public requests
	bittrexAPIGetMarkets         = "public/getmarkets"
	bittrexAPIGetCurrencies      = "public/getcurrencies"
	bittrexAPIGetTicker          = "public/getticker"
	bittrexAPIGetMarketSummaries = "public/getmarketsummaries"
	bittrexAPIGetMarketSummary   = "public/getmarketsummary"
	bittrexAPIGetOrderbook       = "public/getorderbook"
	bittrexAPIGetMarketHistory   = "public/getmarkethistory"

	// Market requests
	bittrexAPIBuyLimit      = "market/buylimit"
	bittrexAPISellLimit     = "market/selllimit"
	bittrexAPICancel        = "market/cancel"
	bittrexAPIGetOpenOrders = "market/getopenorders"

	// Account requests
	bittrexAPIGetBalances          = "account/getbalances"
	bittrexAPIGetBalance           = "account/getbalance"
	bittrexAPIGetDepositAddress    = "account/getdepositaddress"
	bittrexAPIWithdraw             = "account/withdraw"
	bittrexAPIGetOrder             = "account/getorder"
	bittrexAPIGetOrderHistory      = "account/getorderhistory"
	bittrexAPIGetWithdrawalHistory = "account/getwithdrawalhistory"
	bittrexAPIGetDepositHistory    = "account/getdeposithistory"
)

// Bittrex is amazeballs
type Bittrex struct {
	exchange.Base
}

// SetDefaults method assignes the default values for Bittrex
func (b *Bittrex) SetDefaults() {
	b.Name = "Bittrex"
	b.Enabled = false
	b.Verbose = false
	b.Websocket = false
	b.RESTPollingDelay = 10
}

// Setup method sets current configuration details if enabled
func (b *Bittrex) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		b.SetEnabled(false)
	} else {
		b.Enabled = true
		b.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		b.SetAPIKeys(exch.APIKey, exch.APISecret, exch.ClientID, false)
		b.RESTPollingDelay = exch.RESTPollingDelay
		b.Verbose = exch.Verbose
		b.Websocket = exch.Websocket
		b.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		b.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		b.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
	}
}

// GetMarkets is used to get the open and available trading markets at Bittrex
// along with other meta data.
func (b *Bittrex) GetMarkets() ([]Market, error) {
	var markets []Market
	path := fmt.Sprintf(
		"%s/%s/%s/", bittrexAPIURL, bittrexAPIVersion, bittrexAPIGetMarkets,
	)
	resp := Response{}
	err := common.SendHTTPGetRequest(path, true, &resp)
	if err != nil {
		return markets, err
	}
	if resp.Success {
		if err = json.Unmarshal(resp.Result, &markets); err != nil {
			return markets, err
		}
		return markets, nil
	}
	return markets, errors.New(resp.Message)
}

// GetCurrencies is used to get all supported currencies at Bittrex
func (b *Bittrex) GetCurrencies() ([]Currency, error) {
	var currencies []Currency
	path := fmt.Sprintf(
		"%s/%s/%s/", bittrexAPIURL, bittrexAPIVersion, bittrexAPIGetCurrencies,
	)
	resp := Response{}
	err := common.SendHTTPGetRequest(path, true, &resp)
	if err != nil {
		return currencies, err
	}
	if resp.Success {
		if err = json.Unmarshal(resp.Result, &currencies); err != nil {
			return currencies, err
		}
		return currencies, nil
	}
	return currencies, errors.New(resp.Message)
}

// GetTicker sends a public get request and returns current ticker information
// on the supplied currency. Example currency input param "btc-ltc".
func (b *Bittrex) GetTicker(currencyPair string) (Ticker, error) {
	ticker := Ticker{}
	path := fmt.Sprintf(
		"%s/%s/%s?market=%s", bittrexAPIURL, bittrexAPIVersion, bittrexAPIGetTicker,
		common.StringToLower(currencyPair),
	)
	resp := Response{}
	err := common.SendHTTPGetRequest(path, true, &resp)
	if err != nil {
		return ticker, err
	}
	if resp.Success {
		if err = json.Unmarshal(resp.Result, &ticker); err != nil {
			return Ticker{}, err
		}
		return ticker, nil
	}
	return ticker, errors.New(resp.Message)
}

// GetMarketSummaries is used to get the last 24 hour summary of all active
// exchanges
func (b *Bittrex) GetMarketSummaries() ([]MarketSummary, error) {
	var summaries []MarketSummary
	path := fmt.Sprintf(
		"%s/%s/%s/", bittrexAPIURL, bittrexAPIVersion, bittrexAPIGetMarketSummaries,
	)
	resp := Response{}
	err := common.SendHTTPGetRequest(path, true, &resp)
	if err != nil {
		return summaries, err
	}
	if resp.Success {
		if err = json.Unmarshal(resp.Result, &summaries); err != nil {
			return summaries, err
		}
		return summaries, nil
	}
	return summaries, errors.New(resp.Message)
}

// GetMarketSummary is used to get the last 24 hour summary of all active
// exchanges by currency pair (btc-ltc).
func (b *Bittrex) GetMarketSummary(currencyPair string) ([]MarketSummary, error) {
	var summary []MarketSummary
	path := fmt.Sprintf(
		"%s/%s/%s?market=%s", bittrexAPIURL, bittrexAPIVersion,
		bittrexAPIGetMarketSummary, common.StringToLower(currencyPair),
	)
	resp := Response{}
	err := common.SendHTTPGetRequest(path, true, &resp)
	if err != nil {
		return summary, err
	}
	if resp.Success {
		if err = json.Unmarshal(resp.Result, &summary); err != nil {
			return summary, err
		}
		return summary, nil
	}
	return summary, errors.New(resp.Message)
}

// GetOrderbook method returns current order book information by currency, type
// & depth.
// "Currency Pair" ie btc-ltc
// "category" either "buy", "sell" or "both"
// "Depth" is 1 -> 50, 50 is max.
func (b *Bittrex) GetOrderbook(currencyPair, category string, depth int) (OrderBooks, error) {
	var orderbooks OrderBooks
	path := fmt.Sprintf(
		"%s/%s/%s?market=%s&type=%s&depth=%d", bittrexAPIURL, bittrexAPIVersion,
		bittrexAPIGetOrderbook, common.StringToUpper(currencyPair),
		common.StringToLower(category), depth,
	)
	resp := Response{}
	err := common.SendHTTPGetRequest(path, true, &resp)
	if err != nil {
		return orderbooks, err
	}

	if resp.Success {
		if category == "buy" {
			if err = json.Unmarshal(resp.Result, &orderbooks.Buy); err != nil {
				return orderbooks, err
			}
		} else if category == "sell" {
			if err = json.Unmarshal(resp.Result, &orderbooks.Sell); err != nil {
				return orderbooks, err
			}
		} else if category == "both" {
			if err = json.Unmarshal(resp.Result, &orderbooks); err != nil {
				return orderbooks, err
			}
		}
		return orderbooks, nil
	}
	return orderbooks, errors.New(resp.Message)
}

// GetMarketHistory retrieves the latest trades that have occured for a specific
// market
func (b *Bittrex) GetMarketHistory(currencyPair string) ([]MarketHistory, error) {
	var marketHistoriae []MarketHistory
	path := fmt.Sprintf(
		"%s/%s/%s?market=%s", bittrexAPIURL, bittrexAPIVersion,
		bittrexAPIGetMarketHistory, common.StringToUpper(currencyPair),
	)
	resp := Response{}
	err := common.SendHTTPGetRequest(path, true, &resp)
	if err != nil {
		return marketHistoriae, err
	}
	if resp.Success {
		if err = json.Unmarshal(resp.Result, &marketHistoriae); err != nil {
			return marketHistoriae, err
		}
		return marketHistoriae, nil
	}
	return marketHistoriae, errors.New(resp.Message)
}

// PlaceBuyLimit is used to place a buy order in a specific market. Use buylimit
// to place limit orders. Make sure you have the proper permissions set on your
// API keys for this call to work.
// "Currency" ie "btc-ltc"
// "Quantity" is the ammount to purchase
// "Rate" is the rate at which to purchase
func (b *Bittrex) PlaceBuyLimit(currencyPair string, quantity, rate float64) ([]UUID, error) {
	var id []UUID
	values := url.Values{}
	values.Set("market", currencyPair)
	values.Set("quantity", strconv.FormatFloat(quantity, 'E', -1, 64))
	values.Set("rate", strconv.FormatFloat(rate, 'E', -1, 64))

	path := fmt.Sprintf(
		"%s/%s/%s", bittrexAPIURL, bittrexAPIVersion, bittrexAPIGetBalances,
	)

	resp := Response{}
	err := b.SendAuthenticatedHTTPRequest(path, values, &resp)
	if err != nil {
		return id, err
	}
	if resp.Success {
		if err = json.Unmarshal(resp.Result, &id); err != nil {
			return id, err
		}
		return id, nil
	}
	return id, errors.New(resp.Message)
}

// PlaceSellLimit is used to place a sell order in a specific market. Use
// selllimit to place limit orders. Make sure you have the proper permissions
// set on your API keys for this call to work.
// "Currency" ie "btc-ltc"
// "Quantity" is the ammount to purchase
// "Rate" is the rate at which to purchase
func (b *Bittrex) PlaceSellLimit(currencyPair string, quantity, rate float64) ([]UUID, error) {
	var id []UUID
	values := url.Values{}
	values.Set("market", currencyPair)
	values.Set("quantity", strconv.FormatFloat(quantity, 'E', -1, 64))
	values.Set("rate", strconv.FormatFloat(rate, 'E', -1, 64))

	path := fmt.Sprintf(
		"%s/%s/%s", bittrexAPIURL, bittrexAPIVersion, bittrexAPIGetBalances,
	)

	resp := Response{}
	err := b.SendAuthenticatedHTTPRequest(path, values, &resp)
	if err != nil {
		return id, err
	}
	if resp.Success {
		if err = json.Unmarshal(resp.Result, &id); err != nil {
			return id, err
		}
		return id, nil
	}
	return id, errors.New(resp.Message)
}

// GetOpenOrders returns all orders that you currently have opened.
// A specific market can be requested for example "btc-ltc"
func (b *Bittrex) GetOpenOrders(currencyPair string) ([]Order, error) {
	var orders []Order
	values := url.Values{}

	path := fmt.Sprintf(
		"%s/%s/%s", bittrexAPIURL, bittrexAPIVersion, bittrexAPIGetBalances,
	)

	resp := Response{}
	err := b.SendAuthenticatedHTTPRequest(path, values, &resp)
	if err != nil {
		return orders, err
	}
	if resp.Success {
		if err = json.Unmarshal(resp.Result, &orders); err != nil {
			return orders, err
		}
		return orders, nil
	}
	return orders, errors.New(resp.Message)
}

// CancelOrder is used to cancel a buy or sell order.
func (b *Bittrex) CancelOrder(uuid string) ([]Balance, error) {
	var balances []Balance
	values := url.Values{}

	path := fmt.Sprintf(
		"%s/%s/%s", bittrexAPIURL, bittrexAPIVersion, bittrexAPIGetBalances,
	)

	resp := Response{}
	err := b.SendAuthenticatedHTTPRequest(path, values, &resp)
	if err != nil {
		return balances, err
	}
	if resp.Success {
		if err = json.Unmarshal(resp.Result, &balances); err != nil {
			return balances, err
		}
		return balances, nil
	}
	return balances, errors.New(resp.Message)
}

// GetAccountBalances is used to retrieve all balances from your account
func (b *Bittrex) GetAccountBalances() ([]Balance, error) {
	var balances []Balance
	values := url.Values{}

	path := fmt.Sprintf(
		"%s/%s/%s", bittrexAPIURL, bittrexAPIVersion, bittrexAPIGetBalances,
	)

	resp := Response{}
	err := b.SendAuthenticatedHTTPRequest(path, values, &resp)
	if err != nil {
		return balances, err
	}
	if resp.Success {
		if err = json.Unmarshal(resp.Result, &balances); err != nil {
			return balances, err
		}
		return balances, nil
	}
	return balances, errors.New(resp.Message)
}

// GetAccountBalanceByCurrency is used to retrieve the balance from your account
// for a specific currency. ie. "btc" or "ltc"
func (b *Bittrex) GetAccountBalanceByCurrency(currency string) (Balance, error) {
	var balance Balance
	values := url.Values{}
	values.Set("currency", currency)

	path := fmt.Sprintf(
		"%s/%s/%s", bittrexAPIURL, bittrexAPIVersion, bittrexAPIGetBalance,
	)

	resp := Response{}
	err := b.SendAuthenticatedHTTPRequest(path, values, &resp)
	if err != nil {
		return balance, err
	}
	if resp.Success {
		if err = json.Unmarshal(resp.Result, &balance); err != nil {
			return balance, err
		}
		return balance, nil
	}
	return balance, errors.New(resp.Message)
}

// GetDepositAddress is used to retrieve or generate an address for a specific
// currency. If one does not exist, the call will fail and return
// ADDRESS_GENERATING until one is available.
func (b *Bittrex) GetDepositAddress(currency string) (DepositAddress, error) {
	var address DepositAddress
	values := url.Values{}
	values.Set("currency", currency)

	path := fmt.Sprintf(
		"%s/%s/%s", bittrexAPIURL, bittrexAPIVersion, bittrexAPIGetDepositAddress,
	)

	resp := Response{}
	err := b.SendAuthenticatedHTTPRequest(path, values, &resp)
	if err != nil {
		return address, err
	}
	if resp.Success {
		if err = json.Unmarshal(resp.Result, &address); err != nil {
			return address, err
		}
		return address, nil
	}
	return address, errors.New(resp.Message)
}

// Withdraw is used to withdraw funds from your account.
// note: Please account for transaction fee.
func (b *Bittrex) Withdraw(currency, paymentID, address string, quantity float64) (UUID, error) {
	var id UUID
	values := url.Values{}
	values.Set("currency", currency)
	values.Set("quantity", strconv.FormatFloat(quantity, 'E', -1, 64))
	values.Set("address", address)

	path := fmt.Sprintf(
		"%s/%s/%s", bittrexAPIURL, bittrexAPIVersion, bittrexAPIWithdraw,
	)

	resp := Response{}
	err := b.SendAuthenticatedHTTPRequest(path, values, &resp)
	if err != nil {
		return id, err
	}
	if resp.Success {
		if err = json.Unmarshal(resp.Result, &id); err != nil {
			return id, err
		}
		return id, nil
	}
	return id, errors.New(resp.Message)
}

// GetOrder is used to retrieve a single order by UUID.
func (b *Bittrex) GetOrder(uuid string) (Order, error) {
	var order Order
	values := url.Values{}
	values.Set("uuid", uuid)
	path := fmt.Sprintf(
		"%s/%s/%s", bittrexAPIURL, bittrexAPIVersion, bittrexAPIGetOrder,
	)
	resp := Response{}
	err := b.SendAuthenticatedHTTPRequest(path, values, &resp)
	if err != nil {
		return order, err
	}
	if resp.Success {
		if err = json.Unmarshal(resp.Result, &order); err != nil {
			return order, err
		}
		return order, nil
	}
	return order, errors.New(resp.Message)
}

// GetOrderHistory is used to retrieve your order history. If currencyPair
// ommited it will return the entire order History.
func (b *Bittrex) GetOrderHistory(currencyPair string) ([]Order, error) {
	var orders []Order
	values := url.Values{}

	if !(currencyPair == "" || currencyPair == " ") {
		values.Set("market", currencyPair)
	}

	path := fmt.Sprintf(
		"%s/%s/%s", bittrexAPIURL, bittrexAPIVersion, bittrexAPIGetOrderHistory,
	)
	resp := Response{}
	err := b.SendAuthenticatedHTTPRequest(path, values, &resp)
	if err != nil {
		return orders, err
	}
	if resp.Success {
		if err = json.Unmarshal(resp.Result, &orders); err != nil {
			return orders, err
		}
		return orders, nil
	}
	return orders, errors.New(resp.Message)
}

// GetWithdrawelHistory is used to retrieve your withdrawal history. If currency
// ommited it will return the entire history
func (b *Bittrex) GetWithdrawelHistory(currency string) ([]WithdrawalHistory, error) {
	var history []WithdrawalHistory
	values := url.Values{}

	if !(currency == "" || currency == " ") {
		values.Set("currency", currency)
	}

	path := fmt.Sprintf(
		"%s/%s/%s", bittrexAPIURL, bittrexAPIVersion, bittrexAPIGetOrderHistory,
	)
	resp := Response{}
	err := b.SendAuthenticatedHTTPRequest(path, values, &resp)
	if err != nil {
		return history, err
	}
	if resp.Success {
		if err = json.Unmarshal(resp.Result, &history); err != nil {
			return history, err
		}
		return history, nil
	}
	return history, errors.New(resp.Message)
}

// GetDepositHistory is used to retrieve your deposit history. If currency is
// is ommitted it will return the entire deposit history
func (b *Bittrex) GetDepositHistory(currency string) ([]WithdrawalHistory, error) {
	var history []WithdrawalHistory
	values := url.Values{}

	if !(currency == "" || currency == " ") {
		values.Set("currency", currency)
	}

	path := fmt.Sprintf(
		"%s/%s/%s", bittrexAPIURL, bittrexAPIVersion, bittrexAPIGetOrderHistory,
	)
	resp := Response{}
	err := b.SendAuthenticatedHTTPRequest(path, values, &resp)
	if err != nil {
		return history, err
	}
	if resp.Success {
		if err = json.Unmarshal(resp.Result, &history); err != nil {
			return history, err
		}
		return history, nil
	}
	return history, errors.New(resp.Message)
}

// SendAuthenticatedHTTPRequest sends an authenticated http request to a desired
// path
func (b *Bittrex) SendAuthenticatedHTTPRequest(path string, values url.Values, result interface{}) (err error) {
	nonce := strconv.FormatInt(time.Now().UnixNano(), 10)
	values.Set("apikey", b.APIKey)
	values.Set("apisecret", b.APISecret)
	values.Set("nonce", nonce)
	rawQuery := path + "?" + values.Encode()
	hmac := common.GetHMAC(
		common.HashSHA512, []byte(rawQuery), []byte(b.APISecret),
	)
	headers := make(map[string]string)
	headers["apisign"] = common.HexEncodeToString(hmac)

	resp, err := common.SendHTTPRequest(
		"GET", rawQuery, headers, strings.NewReader(""),
	)
	if err != nil {
		return err
	}

	if b.Verbose {
		log.Printf("Recieved raw: %s\n", resp)
	}

	err = common.JSONDecode([]byte(resp), &result)
	if err != nil {
		return errors.New("Unable to JSON Unmarshal response." + err.Error())
	}
	return nil
}
