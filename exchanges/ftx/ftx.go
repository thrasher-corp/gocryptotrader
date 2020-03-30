package ftx

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Ftx is the overarching type across this package
type Ftx struct {
	exchange.Base
}

const (
	ftxAPIURL     = "https://ftx.com/api"
	ftxAPIVersion = ""

	// Public endpoints

	getMarkets           = "/markets"
	getMarket            = "/markets/%s"
	getOrderbook         = "/markets/%s/orderbook?depth=%s"
	getTrades            = "/markets/%s/trades?"
	getHistoricalData    = "/markets/%s/candles?"
	getFutures           = "/futures"
	getFuture            = "/futures/%s"
	getFutureStats       = "/futures/%s/stats"
	getFundingRates      = "/funding_rates"
	getAllWalletBalances = "/wallet/all_balances"

	// Authenticated endpoints
	getAccountInfo       = "/account"
	getPositions         = "/positions"
	setLeverage          = "/account/leverage"
	getCoins             = "/wallet/coins"
	getBalances          = "/wallet/balances"
	getDepositAddress    = "/wallet/deposit_address/%s"
	getDepositHistory    = "/wallet/deposits"
	getWithdrawalHistory = "/wallet/withdrawals"
	withdrawRequest      = "/wallet/withdrawals"
	ftxRateInterval      = time.Minute
	ftxRequestRate       = 180
)

// Start implementing public and private exchange API funcs below

// GetMarkets gets market data
func (f *Ftx) GetMarkets() (Markets, error) {
	var resp Markets
	return resp, f.SendHTTPRequest(ftxAPIURL+getMarkets, &resp)
}

// GetMarket gets market data for a provided asset type
func (f *Ftx) GetMarket(marketName string) (Market, error) {
	var market Market
	log.Println(fmt.Sprintf(ftxAPIURL+getMarket, marketName))
	return market, f.SendHTTPRequest(fmt.Sprintf(ftxAPIURL+getMarket, marketName),
		&market)
}

// GetOrderbook gets orderbook for a given market with a given depth (default depth 20)
func (f *Ftx) GetOrderbook(marketName string, depth int64) (OrderbookData, error) {
	var resp OrderbookData
	var tempOB TempOrderbook
	strDepth := strconv.FormatInt(depth, 10)
	err := f.SendHTTPRequest(fmt.Sprintf(ftxAPIURL+getOrderbook, marketName, strDepth), &tempOB)
	if err != nil {
		return resp, err
	}
	resp.MarketName = marketName
	for x := range tempOB.Result.Asks {
		resp.Asks = append(resp.Asks, OrderData{Price: tempOB.Result.Asks[x][0],
			Size: tempOB.Result.Bids[x][1],
		})
	}
	for y := range tempOB.Result.Bids {
		resp.Bids = append(resp.Bids, OrderData{Price: tempOB.Result.Bids[y][0],
			Size: tempOB.Result.Bids[y][1],
		})
	}
	return resp, nil
}

// GetTrades gets trades based on the conditions specified
func (f *Ftx) GetTrades(marketName, startTime, endTime string, limit int64) (Trades, error) {
	var resp Trades
	var sTime, eTime int64
	var err error
	strLimit := strconv.FormatInt(limit, 10)
	params := url.Values{}
	params.Set("limit", strLimit)
	if startTime != "" {
		sTime, err = strconv.ParseInt(startTime, 10, 64)
		if err != nil {
			return resp, err
		}
		params.Set("start_time", startTime)
	}
	if endTime != "" {
		eTime, err = strconv.ParseInt(endTime, 10, 64)
		if err != nil {
			return resp, err
		}
		params.Set("end_time", endTime)
	}
	if startTime != "" && endTime != "" {
		if sTime > eTime {
			return resp, errors.New("startTime cannot be bigger than endTime")
		}
	}
	log.Println(fmt.Sprintf(ftxAPIURL+getTrades, marketName) + params.Encode())
	return resp, f.SendHTTPRequest((fmt.Sprintf(ftxAPIURL+getTrades, marketName) + params.Encode()),
		&resp)
}

// GetFundingRates gets funding rates for

// GetHistoricalData gets historical OHLCV data for a given market pair
func (f *Ftx) GetHistoricalData(marketName, timeInterval, limit, startTime, endTime string) (HistoricalData, error) {
	var resp HistoricalData
	params := url.Values{}
	params.Set("resolution", timeInterval)
	if limit != "" {
		params.Set("limit", limit)
	}
	if startTime != "" && endTime != "" {
		var sTime, eTime int64
		var err error
		sTime, err = strconv.ParseInt(startTime, 10, 64)
		if err != nil {
			return resp, err
		}
		eTime, err = strconv.ParseInt(endTime, 10, 64)
		if err != nil {
			return resp, err
		}
		if sTime > eTime {
			return resp, errors.New("startTime cannot be bigger than endTime")
		}
	}
	if startTime != "" {
		params.Set("start_time", startTime)
	}
	if endTime != "" {
		params.Set("end_time", endTime)
	}
	return resp, f.SendHTTPRequest(fmt.Sprintf(ftxAPIURL+getHistoricalData, marketName)+params.Encode(), &resp)
}

// GetFutures gets data on futures
func (f *Ftx) GetFutures() (Futures, error) {
	var resp Futures
	return resp, f.SendHTTPRequest(ftxAPIURL+getFutures, &resp)
}

// GetFuture gets data on a given future
func (f *Ftx) GetFuture(futureName string) (Future, error) {
	var resp Future
	return resp, f.SendHTTPRequest(fmt.Sprintf(ftxAPIURL+getFuture, futureName), &resp)
}

// GetFutureStats gets data on a given future's stats
func (f *Ftx) GetFutureStats(futureName string) (FutureStats, error) {
	var resp FutureStats
	return resp, f.SendHTTPRequest(fmt.Sprintf(ftxAPIURL+getFutureStats, futureName), &resp)
}

// GetFundingRates gets data on funding rates
func (f *Ftx) GetFundingRates() (FundingRates, error) {
	var resp FundingRates
	return resp, f.SendHTTPRequest(ftxAPIURL+getFundingRates, &resp)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (f *Ftx) SendHTTPRequest(path string, result interface{}) error {
	return f.SendPayload(&request.Item{
		Method:        http.MethodGet,
		Path:          path,
		Result:        result,
		Verbose:       f.Verbose,
		HTTPDebugging: f.HTTPDebugging,
		HTTPRecording: f.HTTPRecording,
	})
}

// GetAccountInfo gets account info
func (f *Ftx) GetAccountInfo() (AccountData, error) {
	var resp AccountData
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getAccountInfo, nil, &resp)
}

// GetPositions gets the users positions
func (f *Ftx) GetPositions() (Positions, error) {
	var resp Positions
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getPositions, nil, &resp)
}

// ChangeAccountLeverage changes default leverage used by account
func (f *Ftx) ChangeAccountLeverage(leverage float64) error {
	req := make(map[string]interface{})
	req["leverage"] = leverage
	return f.SendAuthHTTPRequest(http.MethodPost, setLeverage, req, nil)
}

// GetCoins gets coins' data in the account wallet
func (f *Ftx) GetCoins() (WalletCoins, error) {
	var resp WalletCoins
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getCoins, nil, &resp)
}

// GetBalances gets balances of the account
func (f *Ftx) GetBalances() (WalletBalances, error) {
	var resp WalletBalances
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getBalances, nil, &resp)
}

// GetAllWalletBalances gets all wallets' balances
func (f *Ftx) GetAllWalletBalances() (AllWalletBalances, error) {
	var resp AllWalletBalances
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getAllWalletBalances, nil, &resp)
}

// FetchDepositAddress gets deposit address for a given coin
func (f *Ftx) FetchDepositAddress(coin string) (DepositAddress, error) {
	var resp DepositAddress
	return resp, f.SendAuthHTTPRequest(http.MethodGet, fmt.Sprintf(getDepositAddress, coin), nil, &resp)
}

// FetchDepositHistory gets deposit history
func (f *Ftx) FetchDepositHistory() (DepositHistory, error) {
	var resp DepositHistory
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getDepositHistory, nil, &resp)
}

// FetchWithdrawalHistory gets withdrawal history
func (f *Ftx) FetchWithdrawalHistory() (WithdrawalHistory, error) {
	var resp WithdrawalHistory
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getWithdrawalHistory, nil, &resp)
}

// Withdraw sends a withdrawal request
func (f *Ftx) Withdraw(coin, address, tag, password string, size float64) (WithdrawData, error) {
	var resp WithdrawData
	req := make(map[string]interface{})
	req["coin"] = coin
	req["address"] = address
	req["tag"] = tag
	return resp, nil
}

// SendAuthHTTPRequest sends an authenticated request
func (f *Ftx) SendAuthHTTPRequest(method, path string, data, result interface{}) error {
	ts := strconv.FormatInt(int64(time.Now().UnixNano()/1000000), 10)
	var body io.Reader
	var hmac, payload []byte
	var err error
	switch data.(type) {
	case map[string]interface{}, []interface{}:
		payload, err = json.Marshal(data)
		if err != nil {
			return err
		}
		body = bytes.NewBuffer(payload)
		sigPayload := ts + method + "/api" + path + string(payload)
		hmac = crypto.GetHMAC(crypto.HashSHA256, []byte(sigPayload), []byte(f.API.Credentials.Secret))
	default:
		sigPayload := ts + method + "/api" + path
		hmac = crypto.GetHMAC(crypto.HashSHA256, []byte(sigPayload), []byte(f.API.Credentials.Secret))
	}
	headers := make(map[string]string)
	headers["FTX-KEY"] = f.API.Credentials.Key
	headers["FTX-SIGN"] = crypto.HexEncodeToString(hmac)
	headers["FTX-TS"] = ts
	headers["Content-Type"] = "application/json"
	return f.SendPayload(&request.Item{
		Method:        method,
		Path:          ftxAPIURL + path,
		Headers:       headers,
		Body:          body,
		Result:        result,
		AuthRequest:   true,
		NonceEnabled:  false,
		Verbose:       f.Verbose,
		HTTPDebugging: f.HTTPDebugging,
		HTTPRecording: f.HTTPRecording,
	})
}
