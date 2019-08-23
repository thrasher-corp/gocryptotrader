package bitfinex

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

const (
	bitfinexAPIURLBase         = "https://api.bitfinex.com"
	bitfinexAPIVersion         = "/v1/"
	bitfinexAPIVersion2        = "2"
	bitfinexTickerV2           = "ticker"
	bitfinexTickersV2          = "tickers"
	bitfinexTicker             = "pubticker/"
	bitfinexStats              = "stats/"
	bitfinexLendbook           = "lendbook/"
	bitfinexOrderbookV2        = "book"
	bitfinexOrderbook          = "book/"
	bitfinexTrades             = "trades/"
	bitfinexTradesV2           = "https://api.bitfinex.com/v2/trades/%s/hist?limit=1000&start=%s&end=%s"
	bitfinexKeyPermissions     = "key_info"
	bitfinexLends              = "lends/"
	bitfinexSymbols            = "symbols/"
	bitfinexSymbolsDetails     = "symbols_details/"
	bitfinexAccountInfo        = "account_infos"
	bitfinexAccountFees        = "account_fees"
	bitfinexAccountSummary     = "summary"
	bitfinexDeposit            = "deposit/new"
	bitfinexOrderNew           = "order/new"
	bitfinexOrderNewMulti      = "order/new/multi"
	bitfinexOrderCancel        = "order/cancel"
	bitfinexOrderCancelMulti   = "order/cancel/multi"
	bitfinexOrderCancelAll     = "order/cancel/all"
	bitfinexOrderCancelReplace = "order/cancel/replace"
	bitfinexOrderStatus        = "order/status"
	bitfinexOrders             = "orders"
	bitfinexInactiveOrders     = "orders/hist"
	bitfinexPositions          = "positions"
	bitfinexClaimPosition      = "position/claim"
	bitfinexHistory            = "history"
	bitfinexHistoryMovements   = "history/movements"
	bitfinexTradeHistory       = "mytrades"
	bitfinexOfferNew           = "offer/new"
	bitfinexOfferCancel        = "offer/cancel"
	bitfinexOfferStatus        = "offer/status"
	bitfinexOffers             = "offers"
	bitfinexMarginActiveFunds  = "taken_funds"
	bitfinexMarginTotalFunds   = "total_taken_funds"
	bitfinexMarginUnusedFunds  = "unused_taken_funds"
	bitfinexMarginClose        = "funding/close"
	bitfinexBalances           = "balances"
	bitfinexMarginInfo         = "margin_infos"
	bitfinexTransfer           = "transfer"
	bitfinexWithdrawal         = "withdraw"
	bitfinexActiveCredits      = "credits"
	bitfinexPlatformStatus     = "platform/status"

	// requests per minute
	bitfinexAuthRate   = 10
	bitfinexUnauthRate = 10

	// Bitfinex platform status values
	// When the platform is marked in maintenance mode bots should stop trading
	// activity. Cancelling orders will be still possible.
	bitfinexMaintenanceMode = 0
	bitfinexOperativeMode   = 1
)

// Bitfinex is the overarching type across the bitfinex package
// Notes: Bitfinex has added a rate limit to the number of REST requests.
// Rate limit policy can vary in a range of 10 to 90 requests per minute
// depending on some factors (e.g. servers load, endpoint, etc.).
type Bitfinex struct {
	exchange.Base
	WebsocketConn         *wshandler.WebsocketConnection
	WebsocketSubdChannels map[int]WebsocketChanInfo
}

// SetDefaults sets the basic defaults for bitfinex
func (b *Bitfinex) SetDefaults() {
	b.Name = "Bitfinex"
	b.Enabled = false
	b.Verbose = false
	b.RESTPollingDelay = 10
	b.WebsocketSubdChannels = make(map[int]WebsocketChanInfo)
	b.APIWithdrawPermissions = exchange.AutoWithdrawCryptoWithAPIPermission |
		exchange.AutoWithdrawFiatWithAPIPermission
	b.RequestCurrencyPairFormat.Delimiter = ""
	b.RequestCurrencyPairFormat.Uppercase = true
	b.ConfigCurrencyPairFormat.Delimiter = ""
	b.ConfigCurrencyPairFormat.Uppercase = true
	b.AssetTypes = []string{ticker.Spot}
	b.SupportsAutoPairUpdating = true
	b.SupportsRESTTickerBatching = true
	b.Requester = request.New(b.Name,
		request.NewRateLimit(time.Second*60, bitfinexAuthRate),
		request.NewRateLimit(time.Second*60, bitfinexUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	b.APIUrlDefault = bitfinexAPIURLBase
	b.APIUrl = b.APIUrlDefault
	b.Websocket = wshandler.New()
	b.Websocket.Functionality = wshandler.WebsocketTickerSupported |
		wshandler.WebsocketTradeDataSupported |
		wshandler.WebsocketOrderbookSupported |
		wshandler.WebsocketSubscribeSupported |
		wshandler.WebsocketUnsubscribeSupported |
		wshandler.WebsocketAuthenticatedEndpointsSupported
	b.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	b.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	b.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (b *Bitfinex) Setup(exch *config.ExchangeConfig) {
	if !exch.Enabled {
		b.SetEnabled(false)
	} else {
		b.Enabled = true
		b.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		b.AuthenticatedWebsocketAPISupport = exch.AuthenticatedWebsocketAPISupport
		b.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		b.SetHTTPClientTimeout(exch.HTTPTimeout)
		b.SetHTTPClientUserAgent(exch.HTTPUserAgent)
		b.RESTPollingDelay = exch.RESTPollingDelay
		b.Verbose = exch.Verbose
		b.HTTPDebugging = exch.HTTPDebugging
		b.Websocket.SetWsStatusAndConnection(exch.Websocket)
		b.BaseCurrencies = exch.BaseCurrencies
		b.AvailablePairs = exch.AvailablePairs
		b.EnabledPairs = exch.EnabledPairs
		err := b.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = b.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
		err = b.SetAutoPairDefaults()
		if err != nil {
			log.Fatal(err)
		}
		err = b.SetAPIURL(exch)
		if err != nil {
			log.Fatal(err)
		}
		err = b.SetClientProxyAddress(exch.ProxyAddress)
		if err != nil {
			log.Fatal(err)
		}
		err = b.Websocket.Setup(b.WsConnect,
			b.Subscribe,
			b.Unsubscribe,
			exch.Name,
			exch.Websocket,
			exch.Verbose,
			bitfinexWebsocket,
			exch.WebsocketURL,
			exch.AuthenticatedWebsocketAPISupport)
		if err != nil {
			log.Fatal(err)
		}
		b.WebsocketConn = &wshandler.WebsocketConnection{
			ExchangeName:         b.Name,
			URL:                  b.Websocket.GetWebsocketURL(),
			ProxyURL:             b.Websocket.GetProxyAddress(),
			Verbose:              b.Verbose,
			ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		}
		b.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
		b.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
		b.Websocket.Orderbook.Setup(
			exch.WebsocketOrderbookBufferLimit,
			true,
			false,
			false,
			false,
			exch.Name)
	}
}

// GetPlatformStatus returns the Bifinex platform status
func (b *Bitfinex) GetPlatformStatus() (int, error) {
	var response []interface{}
	path := fmt.Sprintf("%s/v%s/%s", b.APIUrl, bitfinexAPIVersion2,
		bitfinexPlatformStatus)

	err := b.SendHTTPRequest(path, &response, b.Verbose)
	if err != nil {
		return 0, err
	}

	if (len(response)) != 1 {
		return 0, errors.New("unexpected platform status value")
	}

	return int(response[0].(float64)), nil
}

// GetLatestSpotPrice returns latest spot price of symbol
//
// symbol: string of currency pair
func (b *Bitfinex) GetLatestSpotPrice(symbol string) (float64, error) {
	res, err := b.GetTicker(symbol)
	if err != nil {
		return 0, err
	}
	return res.Mid, nil
}

// GetTicker returns ticker information
func (b *Bitfinex) GetTicker(symbol string) (Ticker, error) {
	response := Ticker{}
	path := common.EncodeURLValues(b.APIUrl+bitfinexAPIVersion+bitfinexTicker+symbol,
		url.Values{})

	if err := b.SendHTTPRequest(path, &response, b.Verbose); err != nil {
		return response, err
	}

	if response.Message != "" {
		return response, errors.New(response.Message)
	}

	return response, nil
}

// GetTickerV2 returns ticker information
func (b *Bitfinex) GetTickerV2(symb string) (Tickerv2, error) {
	var response []interface{}
	var tick Tickerv2

	path := fmt.Sprintf("%s/v%s/%s/%s",
		b.APIUrl,
		bitfinexAPIVersion2,
		bitfinexTickerV2,
		symb)
	err := b.SendHTTPRequest(path, &response, b.Verbose)
	if err != nil {
		return tick, err
	}

	if len(response) > 10 {
		tick.FlashReturnRate = response[0].(float64)
		tick.Bid = response[1].(float64)
		tick.BidSize = response[2].(float64)
		tick.BidPeriod = int64(response[3].(float64))
		tick.Ask = response[4].(float64)
		tick.AskSize = response[5].(float64)
		tick.AskPeriod = int64(response[6].(float64))
		tick.DailyChange = response[7].(float64)
		tick.DailyChangePerc = response[8].(float64)
		tick.Last = response[9].(float64)
		tick.Volume = response[10].(float64)
		tick.High = response[11].(float64)
		tick.Low = response[12].(float64)
	} else {
		tick.Bid = response[0].(float64)
		tick.BidSize = response[1].(float64)
		tick.Ask = response[2].(float64)
		tick.AskSize = response[3].(float64)
		tick.DailyChange = response[4].(float64)
		tick.DailyChangePerc = response[5].(float64)
		tick.Last = response[6].(float64)
		tick.Volume = response[7].(float64)
		tick.High = response[8].(float64)
		tick.Low = response[9].(float64)
	}
	return tick, nil
}

// GetTickersV2 returns ticker information for multiple symbols
func (b *Bitfinex) GetTickersV2(symbols string) ([]Tickersv2, error) {
	var response [][]interface{}
	var tickers []Tickersv2

	v := url.Values{}
	v.Set("symbols", symbols)

	path := common.EncodeURLValues(fmt.Sprintf("%s/v%s/%s",
		b.APIUrl,
		bitfinexAPIVersion2,
		bitfinexTickersV2), v)

	err := b.SendHTTPRequest(path, &response, b.Verbose)
	if err != nil {
		return nil, err
	}

	for x := range response {
		var tick Tickersv2
		data := response[x]
		if len(data) > 11 {
			tick.Symbol = data[0].(string)
			tick.FlashReturnRate = data[1].(float64)
			tick.Bid = data[2].(float64)
			tick.BidSize = data[3].(float64)
			tick.BidPeriod = int64(data[4].(float64))
			tick.Ask = data[5].(float64)
			tick.AskSize = data[6].(float64)
			tick.AskPeriod = int64(data[7].(float64))
			tick.DailyChange = data[8].(float64)
			tick.DailyChangePerc = data[9].(float64)
			tick.Last = data[10].(float64)
			tick.Volume = data[11].(float64)
			tick.High = data[12].(float64)
			tick.Low = data[13].(float64)
		} else {
			tick.Symbol = data[0].(string)
			tick.Bid = data[1].(float64)
			tick.BidSize = data[2].(float64)
			tick.Ask = data[3].(float64)
			tick.AskSize = data[4].(float64)
			tick.DailyChange = data[5].(float64)
			tick.DailyChangePerc = data[6].(float64)
			tick.Last = data[7].(float64)
			tick.Volume = data[8].(float64)
			tick.High = data[9].(float64)
			tick.Low = data[10].(float64)
		}
		tickers = append(tickers, tick)
	}
	return tickers, nil
}

// GetStats returns various statistics about the requested pair
func (b *Bitfinex) GetStats(symbol string) ([]Stat, error) {
	var response []Stat
	path := fmt.Sprint(b.APIUrl + bitfinexAPIVersion + bitfinexStats + symbol)

	return response, b.SendHTTPRequest(path, &response, b.Verbose)
}

// GetFundingBook the entire margin funding book for both bids and asks sides
// per currency string
// symbol - example "USD"
func (b *Bitfinex) GetFundingBook(symbol string) (FundingBook, error) {
	response := FundingBook{}
	path := fmt.Sprint(b.APIUrl + bitfinexAPIVersion + bitfinexLendbook + symbol)

	if err := b.SendHTTPRequest(path, &response, b.Verbose); err != nil {
		return response, err
	}

	if response.Message != "" {
		return response, errors.New(response.Message)
	}

	return response, nil
}

// GetOrderbook retieves the orderbook bid and ask price points for a currency
// pair - By default the response will return 25 bid and 25 ask price points.
// CurrencyPair - Example "BTCUSD"
// Values can contain limit amounts for both the asks and bids - Example
// "limit_bids" = 1000
func (b *Bitfinex) GetOrderbook(currencyPair string, values url.Values) (Orderbook, error) {
	response := Orderbook{}
	path := common.EncodeURLValues(
		b.APIUrl+bitfinexAPIVersion+bitfinexOrderbook+currencyPair,
		values,
	)
	return response, b.SendHTTPRequest(path, &response, b.Verbose)
}

// GetOrderbookV2 retieves the orderbook bid and ask price points for a currency
// pair - By default the response will return 25 bid and 25 ask price points.
// symbol - Example "tBTCUSD"
// precision - P0,P1,P2,P3,R0
// Values can contain limit amounts for both the asks and bids - Example
// "len" = 1000
func (b *Bitfinex) GetOrderbookV2(symbol, precision string, values url.Values) (OrderbookV2, error) {
	var response [][]interface{}
	var book OrderbookV2
	path := common.EncodeURLValues(fmt.Sprintf("%s/v%s/%s/%s/%s", b.APIUrl,
		bitfinexAPIVersion2, bitfinexOrderbookV2, symbol, precision), values)
	err := b.SendHTTPRequest(path, &response, b.Verbose)
	if err != nil {
		return book, err
	}

	for x := range response {
		data := response[x]
		bookItem := BookV2{}

		if len(data) > 3 {
			bookItem.Rate = data[0].(float64)
			bookItem.Price = data[1].(float64)
			bookItem.Count = int64(data[2].(float64))
			bookItem.Amount = data[3].(float64)
		} else {
			bookItem.Price = data[0].(float64)
			bookItem.Count = int64(data[1].(float64))
			bookItem.Amount = data[2].(float64)
		}

		if symbol[0] == 't' {
			if bookItem.Amount > 0 {
				book.Bids = append(book.Bids, bookItem)
			} else {
				book.Asks = append(book.Asks, bookItem)
			}
		} else {
			if bookItem.Amount > 0 {
				book.Asks = append(book.Asks, bookItem)
			} else {
				book.Bids = append(book.Bids, bookItem)
			}
		}
	}
	return book, nil
}

// GetTrades returns a list of the most recent trades for the given curencyPair
// By default the response will return 100 trades
// CurrencyPair - Example "BTCUSD"
// Values can contain limit amounts for the number of trades returned - Example
// "limit_trades" = 1000
func (b *Bitfinex) GetTrades(currencyPair string, values url.Values) ([]TradeStructure, error) {
	var response []TradeStructure
	path := common.EncodeURLValues(
		b.APIUrl+bitfinexAPIVersion+bitfinexTrades+currencyPair,
		values,
	)
	return response, b.SendHTTPRequest(path, &response, b.Verbose)
}

// GetTradesV2 uses the V2 API to get historic trades that occurred on the
// exchange
//
// currencyPair e.g. "tBTCUSD" v2 prefixes currency pairs with t. (?)
// timestampStart is an int64 unix epoch time
// timestampEnd is an int64 unix epoch time, make sure this is always there or
// you will get the most recent trades.
// reOrderResp reorders the returned data.
func (b *Bitfinex) GetTradesV2(currencyPair string, timestampStart, timestampEnd int64, reOrderResp bool) ([]TradeStructureV2, error) {
	var resp [][]interface{}
	var actualHistory []TradeStructureV2

	path := fmt.Sprintf(bitfinexTradesV2,
		currencyPair,
		strconv.FormatInt(timestampStart, 10),
		strconv.FormatInt(timestampEnd, 10))

	err := b.SendHTTPRequest(path, &resp, b.Verbose)
	if err != nil {
		return actualHistory, err
	}

	var tempHistory TradeStructureV2
	for _, data := range resp {
		tempHistory.TID = int64(data[0].(float64))
		tempHistory.Timestamp = int64(data[1].(float64))
		tempHistory.Amount = data[2].(float64)
		tempHistory.Price = data[3].(float64)
		tempHistory.Exchange = b.Name
		tempHistory.Type = "BUY"

		if tempHistory.Amount < 0 {
			tempHistory.Type = "SELL"
			tempHistory.Amount *= -1
		}

		actualHistory = append(actualHistory, tempHistory)
	}

	// re-order index
	if reOrderResp {
		orderedHistory := make([]TradeStructureV2, len(actualHistory))
		for i, quickRange := range actualHistory {
			orderedHistory[len(actualHistory)-i-1] = quickRange
		}
		return orderedHistory, nil
	}
	return actualHistory, nil
}

// GetLendbook returns a list of the most recent funding data for the given
// currency: total amount provided and Flash Return Rate (in % by 365 days) over
// time
// Symbol - example "USD"
func (b *Bitfinex) GetLendbook(symbol string, values url.Values) (Lendbook, error) {
	response := Lendbook{}
	if len(symbol) == 6 {
		symbol = symbol[:3]
	}
	path := common.EncodeURLValues(b.APIUrl+bitfinexAPIVersion+bitfinexLendbook+symbol,
		values)

	return response, b.SendHTTPRequest(path, &response, b.Verbose)
}

// GetLends returns a list of the most recent funding data for the given
// currency: total amount provided and Flash Return Rate (in % by 365 days)
// over time
// Symbol - example "USD"
func (b *Bitfinex) GetLends(symbol string, values url.Values) ([]Lends, error) {
	var response []Lends
	path := common.EncodeURLValues(b.APIUrl+bitfinexAPIVersion+bitfinexLends+symbol,
		values)

	return response, b.SendHTTPRequest(path, &response, b.Verbose)
}

// GetSymbols returns the available currency pairs on the exchange
func (b *Bitfinex) GetSymbols() ([]string, error) {
	var products []string
	path := fmt.Sprint(b.APIUrl + bitfinexAPIVersion + bitfinexSymbols)

	return products, b.SendHTTPRequest(path, &products, b.Verbose)
}

// GetSymbolsDetails a list of valid symbol IDs and the pair details
func (b *Bitfinex) GetSymbolsDetails() ([]SymbolDetails, error) {
	var response []SymbolDetails
	path := fmt.Sprint(b.APIUrl + bitfinexAPIVersion + bitfinexSymbolsDetails)

	return response, b.SendHTTPRequest(path, &response, b.Verbose)
}

// GetAccountInformation returns information about your account incl. trading
// fees
func (b *Bitfinex) GetAccountInformation() ([]AccountInfo, error) {
	var responses []AccountInfo
	return responses, b.SendAuthenticatedHTTPRequest(http.MethodPost,
		bitfinexAccountInfo, nil, &responses)
}

// GetAccountFees - Gets all fee rates for all currencies
func (b *Bitfinex) GetAccountFees() (AccountFees, error) {
	response := AccountFees{}
	return response, b.SendAuthenticatedHTTPRequest(http.MethodPost,
		bitfinexAccountFees, nil, &response)
}

// GetAccountSummary returns a 30-day summary of your trading volume and return
// on margin funding
func (b *Bitfinex) GetAccountSummary() (AccountSummary, error) {
	response := AccountSummary{}

	return response,
		b.SendAuthenticatedHTTPRequest(
			http.MethodPost, bitfinexAccountSummary, nil, &response,
		)
}

// NewDeposit returns a new deposit address
// Method - Example methods accepted: “bitcoin”, “litecoin”, “ethereum”,
// “tethers", "ethereumc", "zcash", "monero", "iota", "bcash"
// WalletName - accepted: “trading”, “exchange”, “deposit”
// renew - Default is 0. If set to 1, will return a new unused deposit address
func (b *Bitfinex) NewDeposit(method, walletName string, renew int) (DepositResponse, error) {
	response := DepositResponse{}
	req := make(map[string]interface{})
	req["method"] = method
	req["wallet_name"] = walletName
	req["renew"] = renew

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost,
			bitfinexDeposit,
			req,
			&response)
}

// GetKeyPermissions checks the permissions of the key being used to generate
// this request.
func (b *Bitfinex) GetKeyPermissions() (KeyPermissions, error) {
	response := KeyPermissions{}

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost,
			bitfinexKeyPermissions, nil, &response)
}

// GetMarginInfo shows your trading wallet information for margin trading
func (b *Bitfinex) GetMarginInfo() ([]MarginInfo, error) {
	var response []MarginInfo

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost,
			bitfinexMarginInfo, nil, &response)
}

// GetAccountBalance returns full wallet balance information
func (b *Bitfinex) GetAccountBalance() ([]Balance, error) {
	var response []Balance

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost,
			bitfinexBalances, nil, &response)
}

// WalletTransfer move available balances between your wallets
// Amount - Amount to move
// Currency -  example "BTC"
// WalletFrom - example "exchange"
// WalletTo -  example "deposit"
func (b *Bitfinex) WalletTransfer(amount float64, currency, walletFrom, walletTo string) ([]WalletTransfer, error) {
	var response []WalletTransfer
	req := make(map[string]interface{})
	req["amount"] = amount
	req["currency"] = currency
	req["walletfrom"] = walletFrom
	req["walletTo"] = walletTo

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost,
			bitfinexTransfer,
			req,
			&response)
}

// WithdrawCryptocurrency requests a withdrawal from one of your wallets.
// For FIAT, use WithdrawFIAT
func (b *Bitfinex) WithdrawCryptocurrency(withdrawType, wallet, address, paymentID string, amount float64, c currency.Code) ([]Withdrawal, error) {
	var response []Withdrawal
	req := make(map[string]interface{})
	req["withdraw_type"] = withdrawType
	req["walletselected"] = wallet
	req["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	req["address"] = address
	if c == currency.XMR {
		req["paymend_id"] = paymentID
	}

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost,
			bitfinexWithdrawal,
			req,
			&response)
}

// WithdrawFIAT Sends an authenticated request to withdraw FIAT currency
func (b *Bitfinex) WithdrawFIAT(withdrawalType, walletType string, withdrawRequest *exchange.WithdrawRequest) ([]Withdrawal, error) {
	var response []Withdrawal
	req := make(map[string]interface{})

	req["withdraw_type"] = withdrawalType
	req["walletselected"] = walletType
	req["amount"] = strconv.FormatFloat(withdrawRequest.Amount, 'f', -1, 64)
	req["account_name"] = withdrawRequest.BankAccountName
	req["account_number"] = strconv.FormatFloat(withdrawRequest.BankAccountNumber, 'f', -1, 64)
	req["bank_name"] = withdrawRequest.BankName
	req["bank_address"] = withdrawRequest.BankAddress
	req["bank_city"] = withdrawRequest.BankCity
	req["bank_country"] = withdrawRequest.BankCountry
	req["expressWire"] = withdrawRequest.IsExpressWire
	req["swift"] = withdrawRequest.SwiftCode
	req["detail_payment"] = withdrawRequest.Description
	req["currency"] = withdrawRequest.WireCurrency
	req["account_address"] = withdrawRequest.BankAddress

	if withdrawRequest.RequiresIntermediaryBank {
		req["intermediary_bank_name"] = withdrawRequest.IntermediaryBankName
		req["intermediary_bank_address"] = withdrawRequest.IntermediaryBankAddress
		req["intermediary_bank_city"] = withdrawRequest.IntermediaryBankCity
		req["intermediary_bank_country"] = withdrawRequest.IntermediaryBankCountry
		req["intermediary_bank_account"] = strconv.FormatFloat(withdrawRequest.IntermediaryBankAccountNumber, 'f', -1, 64)
		req["intermediary_bank_swift"] = withdrawRequest.IntermediarySwiftCode
	}

	return response, b.SendAuthenticatedHTTPRequest(http.MethodPost, bitfinexWithdrawal, req, &response)
}

// NewOrder submits a new order and returns a order information
// Major Upgrade needed on this function to include all query params
func (b *Bitfinex) NewOrder(currencyPair string, amount, price float64, buy bool, orderType string, hidden bool) (Order, error) {
	response := Order{}
	req := make(map[string]interface{})
	req["symbol"] = currencyPair
	req["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	req["price"] = strconv.FormatFloat(price, 'f', -1, 64)
	req["exchange"] = "bitfinex"
	req["type"] = orderType
	req["is_hidden"] = hidden

	if buy {
		req["side"] = "buy"
	} else {
		req["side"] = "sell"
	}

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost,
			bitfinexOrderNew,
			req,
			&response)
}

// NewOrderMulti allows several new orders at once
func (b *Bitfinex) NewOrderMulti(orders []PlaceOrder) (OrderMultiResponse, error) {
	response := OrderMultiResponse{}
	req := make(map[string]interface{})
	req["orders"] = orders

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost,
			bitfinexOrderNewMulti,
			req,
			&response)
}

// CancelExistingOrder cancels a single order by OrderID
func (b *Bitfinex) CancelExistingOrder(orderID int64) (Order, error) {
	response := Order{}
	req := make(map[string]interface{})
	req["order_id"] = orderID

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost,
			bitfinexOrderCancel,
			req,
			&response)
}

// CancelMultipleOrders cancels multiple orders
func (b *Bitfinex) CancelMultipleOrders(orderIDs []int64) (string, error) {
	response := GenericResponse{}
	req := make(map[string]interface{})
	req["order_ids"] = orderIDs

	return response.Result,
		b.SendAuthenticatedHTTPRequest(http.MethodPost,
			bitfinexOrderCancelMulti,
			req,
			nil)
}

// CancelAllExistingOrders cancels all active and open orders
func (b *Bitfinex) CancelAllExistingOrders() (string, error) {
	response := GenericResponse{}

	return response.Result,
		b.SendAuthenticatedHTTPRequest(http.MethodPost,
			bitfinexOrderCancelAll,
			nil,
			nil)
}

// ReplaceOrder replaces an older order with a new order
func (b *Bitfinex) ReplaceOrder(orderID int64, symbol string, amount, price float64, buy bool, orderType string, hidden bool) (Order, error) {
	response := Order{}
	req := make(map[string]interface{})
	req["order_id"] = orderID
	req["symbol"] = symbol
	req["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	req["price"] = strconv.FormatFloat(price, 'f', -1, 64)
	req["exchange"] = "bitfinex"
	req["type"] = orderType
	req["is_hidden"] = hidden

	if buy {
		req["side"] = "buy"
	} else {
		req["side"] = "sell"
	}

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost,
			bitfinexOrderCancelReplace,
			req,
			&response)
}

// GetOrderStatus returns order status information
func (b *Bitfinex) GetOrderStatus(orderID int64) (Order, error) {
	orderStatus := Order{}
	req := make(map[string]interface{})
	req["order_id"] = orderID

	return orderStatus,
		b.SendAuthenticatedHTTPRequest(http.MethodPost,
			bitfinexOrderStatus,
			req,
			&orderStatus)
}

// GetInactiveOrders returns order status information
func (b *Bitfinex) GetInactiveOrders() ([]Order, error) {
	var response []Order
	req := make(map[string]interface{})
	req["limit"] = "100"

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost,
			bitfinexInactiveOrders,
			req,
			&response)
}

// GetOpenOrders returns all active orders and statuses
func (b *Bitfinex) GetOpenOrders() ([]Order, error) {
	var response []Order

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost,
			bitfinexOrders,
			nil,
			&response)
}

// GetActivePositions returns an array of active positions
func (b *Bitfinex) GetActivePositions() ([]Position, error) {
	var response []Position

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost,
			bitfinexPositions,
			nil,
			&response)
}

// ClaimPosition allows positions to be claimed
func (b *Bitfinex) ClaimPosition(positionID int) (Position, error) {
	response := Position{}
	req := make(map[string]interface{})
	req["position_id"] = positionID

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost,
			bitfinexClaimPosition,
			nil,
			nil)
}

// GetBalanceHistory returns balance history for the account
func (b *Bitfinex) GetBalanceHistory(symbol string, timeSince, timeUntil time.Time, limit int, wallet string) ([]BalanceHistory, error) {
	var response []BalanceHistory
	req := make(map[string]interface{})
	req["currency"] = symbol

	if !timeSince.IsZero() {
		req["since"] = timeSince
	}
	if !timeUntil.IsZero() {
		req["until"] = timeUntil
	}
	if limit > 0 {
		req["limit"] = limit
	}
	if len(wallet) > 0 {
		req["wallet"] = wallet
	}

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost,
			bitfinexHistory,
			req,
			&response)
}

// GetMovementHistory returns an array of past deposits and withdrawals
func (b *Bitfinex) GetMovementHistory(symbol, method string, timeSince, timeUntil time.Time, limit int) ([]MovementHistory, error) {
	var response []MovementHistory
	req := make(map[string]interface{})
	req["currency"] = symbol

	if len(method) > 0 {
		req["method"] = method
	}
	if !timeSince.IsZero() {
		req["since"] = timeSince
	}
	if !timeUntil.IsZero() {
		req["until"] = timeUntil
	}
	if limit > 0 {
		req["limit"] = limit
	}

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost,
			bitfinexHistoryMovements,
			req,
			&response)
}

// GetTradeHistory returns past executed trades
func (b *Bitfinex) GetTradeHistory(currencyPair string, timestamp, until time.Time, limit, reverse int) ([]TradeHistory, error) {
	var response []TradeHistory
	req := make(map[string]interface{})
	req["currency"] = currencyPair
	req["timestamp"] = timestamp

	if !until.IsZero() {
		req["until"] = until
	}
	if limit > 0 {
		req["limit"] = limit
	}
	if reverse > 0 {
		req["reverse"] = reverse
	}

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost,
			bitfinexTradeHistory,
			req,
			&response)
}

// NewOffer submits a new offer
func (b *Bitfinex) NewOffer(symbol string, amount, rate float64, period int64, direction string) (Offer, error) {
	response := Offer{}
	req := make(map[string]interface{})
	req["currency"] = symbol
	req["amount"] = amount
	req["rate"] = rate
	req["period"] = period
	req["direction"] = direction

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost,
			bitfinexOfferNew,
			req,
			&response)
}

// CancelOffer cancels offer by offerID
func (b *Bitfinex) CancelOffer(offerID int64) (Offer, error) {
	response := Offer{}
	req := make(map[string]interface{})
	req["offer_id"] = offerID

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost,
			bitfinexOfferCancel,
			req,
			&response)
}

// GetOfferStatus checks offer status whether it has been cancelled, execute or
// is still active
func (b *Bitfinex) GetOfferStatus(offerID int64) (Offer, error) {
	response := Offer{}
	req := make(map[string]interface{})
	req["offer_id"] = offerID

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost,
			bitfinexOrderStatus,
			req,
			&response)
}

// GetActiveCredits returns all available credits
func (b *Bitfinex) GetActiveCredits() ([]Offer, error) {
	var response []Offer

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost,
			bitfinexActiveCredits,
			nil,
			&response)
}

// GetActiveOffers returns all current active offers
func (b *Bitfinex) GetActiveOffers() ([]Offer, error) {
	var response []Offer

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost,
			bitfinexOffers,
			nil,
			&response)
}

// GetActiveMarginFunding returns an array of active margin funds
func (b *Bitfinex) GetActiveMarginFunding() ([]MarginFunds, error) {
	var response []MarginFunds

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost,
			bitfinexMarginActiveFunds,
			nil,
			&response)
}

// GetUnusedMarginFunds returns an array of funding borrowed but not currently
// used
func (b *Bitfinex) GetUnusedMarginFunds() ([]MarginFunds, error) {
	var response []MarginFunds

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost,
			bitfinexMarginUnusedFunds,
			nil,
			&response)
}

// GetMarginTotalTakenFunds returns an array of active funding used in a
// position
func (b *Bitfinex) GetMarginTotalTakenFunds() ([]MarginTotalTakenFunds, error) {
	var response []MarginTotalTakenFunds

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost,
			bitfinexMarginTotalFunds,
			nil,
			&response)
}

// CloseMarginFunding closes an unused or used taken fund
func (b *Bitfinex) CloseMarginFunding(swapID int64) (Offer, error) {
	response := Offer{}
	req := make(map[string]interface{})
	req["swap_id"] = swapID

	return response,
		b.SendAuthenticatedHTTPRequest(http.MethodPost,
			bitfinexMarginClose,
			req,
			&response)
}

// SendHTTPRequest sends an unauthenticated request
func (b *Bitfinex) SendHTTPRequest(path string, result interface{}, verbose bool) error {
	return b.SendPayload(http.MethodGet,
		path,
		nil,
		nil,
		result,
		false,
		false,
		verbose,
		b.HTTPDebugging,
		b.HTTPRecording)
}

// SendAuthenticatedHTTPRequest sends an autheticated http request and json
// unmarshals result to a supplied variable
func (b *Bitfinex) SendAuthenticatedHTTPRequest(method, path string, params map[string]interface{}, result interface{}) error {
	if !b.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet,
			b.Name)
	}

	n := b.Requester.GetNonce(true)

	req := make(map[string]interface{})
	req["request"] = fmt.Sprintf("%s%s", bitfinexAPIVersion, path)
	req["nonce"] = n.String()

	for key, value := range params {
		req[key] = value
	}

	PayloadJSON, err := common.JSONEncode(req)
	if err != nil {
		return errors.New("sendAuthenticatedAPIRequest: unable to JSON request")
	}

	if b.Verbose {
		log.Debugf("Request JSON: %s\n", PayloadJSON)
	}

	PayloadBase64 := common.Base64Encode(PayloadJSON)
	hmac := common.GetHMAC(common.HashSHA512_384, []byte(PayloadBase64),
		[]byte(b.APISecret))
	headers := make(map[string]string)
	headers["X-BFX-APIKEY"] = b.APIKey
	headers["X-BFX-PAYLOAD"] = PayloadBase64
	headers["X-BFX-SIGNATURE"] = common.HexEncodeToString(hmac)

	return b.SendPayload(method,
		b.APIUrl+bitfinexAPIVersion+path,
		headers,
		nil,
		result,
		true,
		true,
		b.Verbose,
		b.HTTPDebugging,
		b.HTTPRecording)
}

// GetFee returns an estimate of fee based on type of transaction
func (b *Bitfinex) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64

	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		accountInfos, err := b.GetAccountInformation()
		if err != nil {
			return 0, err
		}
		fee, err = b.CalculateTradingFee(accountInfos,
			feeBuilder.PurchasePrice,
			feeBuilder.Amount,
			feeBuilder.Pair.Base,
			feeBuilder.IsMaker)
		if err != nil {
			return 0, err
		}
	case exchange.CyptocurrencyDepositFee:
		//TODO: fee is charged when < $1000USD is transferred, need to infer value in some way
		fee = 0
	case exchange.CryptocurrencyWithdrawalFee:
		accountFees, err := b.GetAccountFees()
		if err != nil {
			return 0, err
		}
		fee, err = b.GetCryptocurrencyWithdrawalFee(feeBuilder.Pair.Base,
			accountFees)
		if err != nil {
			return 0, err
		}
	case exchange.InternationalBankDepositFee:
		fee = getInternationalBankDepositFee(feeBuilder.Amount)
	case exchange.InternationalBankWithdrawalFee:
		fee = getInternationalBankWithdrawalFee(feeBuilder.Amount)
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	if fee < 0 {
		fee = 0
	}
	return fee, nil
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
// does not require an API request, requires manual updating
func getOfflineTradeFee(price, amount float64) float64 {
	return 0.001 * price * amount
}

// GetCryptocurrencyWithdrawalFee returns an estimate of fee based on type of transaction
func (b *Bitfinex) GetCryptocurrencyWithdrawalFee(c currency.Code, accountFees AccountFees) (fee float64, err error) {
	switch result := accountFees.Withdraw[c.String()].(type) {
	case string:
		fee, err = strconv.ParseFloat(result, 64)
		if err != nil {
			return 0, err
		}
	case float64:
		fee = result
	}

	return fee, nil
}

func getInternationalBankDepositFee(amount float64) float64 {
	return 0.001 * amount
}

func getInternationalBankWithdrawalFee(amount float64) float64 {
	return 0.001 * amount
}

// CalculateTradingFee returns an estimate of fee based on type of whether is maker or taker fee
func (b *Bitfinex) CalculateTradingFee(accountInfos []AccountInfo, purchasePrice, amount float64, c currency.Code, isMaker bool) (fee float64, err error) {
	for _, i := range accountInfos {
		for _, j := range i.Fees {
			if c.String() == j.Pairs {
				if isMaker {
					fee = j.MakerFees
				} else {
					fee = j.TakerFees
				}
				break
			}
		}
		if fee > 0 {
			break
		}
	}
	return (fee / 100) * purchasePrice * amount, err
}

// ConvertSymbolToWithdrawalType You need to have specific withdrawal types to withdraw from Bitfinex
func (b *Bitfinex) ConvertSymbolToWithdrawalType(c currency.Code) string {
	switch c {
	case currency.BTC:
		return "bitcoin"
	case currency.LTC:
		return "litecoin"
	case currency.ETH:
		return "ethereum"
	case currency.ETC:
		return "ethereumc"
	case currency.USDT:
		return "tetheruso"
	case currency.ZEC:
		return "zcash"
	case currency.XMR:
		return "monero"
	case currency.DSH:
		return "dash"
	case currency.XRP:
		return "ripple"
	case currency.SAN:
		return "santiment"
	case currency.OMG:
		return "omisego"
	case currency.BCH:
		return "bcash"
	case currency.ETP:
		return "metaverse"
	case currency.AVT:
		return "aventus"
	case currency.EDO:
		return "eidoo"
	case currency.BTG:
		return "bgold"
	case currency.DATA:
		return "datacoin"
	case currency.GNT:
		return "golem"
	case currency.SNT:
		return "status"
	default:
		return c.Lower().String()
	}
}

// ConvertSymbolToDepositMethod returns a converted currency deposit method
func (b *Bitfinex) ConvertSymbolToDepositMethod(c currency.Code) (method string, err error) {
	switch c {
	case currency.BTC:
		method = "bitcoin"
	case currency.LTC:
		method = "litecoin"
	case currency.ETH:
		method = "ethereum"
	case currency.ETC:
		method = "ethereumc"
	case currency.USDT:
		method = "tetheruso"
	case currency.ZEC:
		method = "zcash"
	case currency.XMR:
		method = "monero"
	case currency.BCH:
		method = "bcash"
	case currency.MIOTA:
		method = "iota"
	default:
		err = fmt.Errorf("currency %s not supported in method list",
			c)
	}
	return
}
