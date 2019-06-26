package kline

import (
	"sync"
	"time"
)

// const values for the ticker package
const (
	errExchangeTickerNotFound = "ticker for exchange does not exist"
	errPairNotSet             = "ticker currency pair not set"
	errAssetTypeNotSet        = "ticker asset type not set"
	errBaseCurrencyNotFound   = "ticker base currency not found"
	errQuoteCurrencyNotFound  = "ticker quote currency not found"

	Spot = "SPOT"
)

// Vars for the ticker package
var (
	Klines []Kline
	m      sync.Mutex
)

// Kline K线的映射
type Kline struct {
	Amount    float64   `json:"amount" description:"成交量"`
	Count     int       `json:"count" description:"成交笔数"`
	Open      float64   `json:"open" description:"开盘价"`
	Close     float64   `json:"close" description:"收盘价"`
	Low       float64   `json:"low" description:"最低价"`
	High      float64   `json:"high" description:"最高价"`
	Vol       float64   `json:"vol" description:"成交额,即SUM(每一笔成交价 * 该笔的成交数量)"`
	OpenTime  time.Time `json:"opentime" description:"开盘时间"`
	CloseTime time.Time `json:"closetime" description:"收盘时间"`
}

// // GetKlines  checks and returns a requested kline if it exists
// func GetKlines(exchange string, arg interface{}, p currency.Pair, tickerType string) (Klines, error) {
// 	if strings.EqualFold("Binance", exchange){

// 	}
// 	ticker, err := GetTickerByExchange(exchange)
// 	if err != nil {
// 		return Price{}, err
// 	}

// 	if !BaseCurrencyExists(exchange, p.Base) {
// 		return Price{}, errors.New(errBaseCurrencyNotFound)
// 	}

// 	if !QuoteCurrencyExists(exchange, p) {
// 		return Price{}, errors.New(errQuoteCurrencyNotFound)
// 	}

// 	return ticker.Price[p.Base.Upper().String()][p.Quote.Upper().String()][tickerType], nil
// }
