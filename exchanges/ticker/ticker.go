package ticker

import (
	"errors"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/decimal"
)

// Const values for the ticker package
const (
	ErrTickerForExchangeNotFound = "Ticker for exchange does not exist."
	ErrPrimaryCurrencyNotFound   = "Error primary currency for ticker not found."
	ErrSecondaryCurrencyNotFound = "Error secondary currency for ticker not found."

	Spot = "SPOT"
)

// Vars for the ticker package
var (
	Tickers []Ticker
	m       sync.Mutex
)

// Price struct stores the currency pair and pricing information
type Price struct {
	Pair         pair.CurrencyPair `json:"Pair"`
	LastUpdated  time.Time         `json:"LastUpdated"`
	CurrencyPair string            `json:"CurrencyPair"`
	Last         decimal.Decimal   `json:"Last"`
	High         decimal.Decimal   `json:"High"`
	Low          decimal.Decimal   `json:"Low"`
	Bid          decimal.Decimal   `json:"Bid"`
	Ask          decimal.Decimal   `json:"Ask"`
	Volume       decimal.Decimal   `json:"Volume"`
	PriceATH     decimal.Decimal   `json:"PriceATH"`
}

// Ticker struct holds the ticker information for a currency pair and type
type Ticker struct {
	Price        map[pair.CurrencyItem]map[pair.CurrencyItem]map[string]Price
	ExchangeName string
}

// PriceToString returns the string version of a stored price field
func (t *Ticker) PriceToString(p pair.CurrencyPair, priceType, tickerType string) string {
	priceType = common.StringToLower(priceType)

	switch priceType {
	case "last":
		//return strconv.FormatFloat(t.Price[p.FirstCurrency][p.SecondCurrency][tickerType].Last, 'f', -1, 64)
		return t.Price[p.FirstCurrency][p.SecondCurrency][tickerType].Last.String()
	case "high":
		//return strconv.FormatFloat(t.Price[p.FirstCurrency][p.SecondCurrency][tickerType].High, 'f', -1, 64)
		return t.Price[p.FirstCurrency][p.SecondCurrency][tickerType].High.String()
	case "low":
		//return strconv.FormatFloat(t.Price[p.FirstCurrency][p.SecondCurrency][tickerType].Low, 'f', -1, 64)
		return t.Price[p.FirstCurrency][p.SecondCurrency][tickerType].Low.String()
	case "bid":
		//return strconv.FormatFloat(t.Price[p.FirstCurrency][p.SecondCurrency][tickerType].Bid, 'f', -1, 64)
		return t.Price[p.FirstCurrency][p.SecondCurrency][tickerType].Bid.String()
	case "ask":
		//return strconv.FormatFloat(t.Price[p.FirstCurrency][p.SecondCurrency][tickerType].Ask, 'f', -1, 64)
		return t.Price[p.FirstCurrency][p.SecondCurrency][tickerType].Ask.String()
	case "volume":
		return t.Price[p.FirstCurrency][p.SecondCurrency][tickerType].Volume.String()
	case "ath":
		//return strconv.FormatFloat(t.Price[p.FirstCurrency][p.SecondCurrency][tickerType].PriceATH, 'f', -1, 64)
		return t.Price[p.FirstCurrency][p.SecondCurrency][tickerType].PriceATH.String()
	default:
		return ""
	}
}

// GetTicker checks and returns a requested ticker if it exists
func GetTicker(exchange string, p pair.CurrencyPair, tickerType string) (Price, error) {
	ticker, err := GetTickerByExchange(exchange)
	if err != nil {
		return Price{}, err
	}

	if !FirstCurrencyExists(exchange, p.FirstCurrency) {
		return Price{}, errors.New(ErrPrimaryCurrencyNotFound)
	}

	if !SecondCurrencyExists(exchange, p) {
		return Price{}, errors.New(ErrSecondaryCurrencyNotFound)
	}

	return ticker.Price[p.FirstCurrency][p.SecondCurrency][tickerType], nil
}

// GetTickerByExchange returns an exchange Ticker
func GetTickerByExchange(exchange string) (*Ticker, error) {
	m.Lock()
	defer m.Unlock()
	for _, y := range Tickers {
		if y.ExchangeName == exchange {
			return &y, nil
		}
	}
	return nil, errors.New(ErrTickerForExchangeNotFound)
}

// FirstCurrencyExists checks to see if the first currency of the Price map
// exists
func FirstCurrencyExists(exchange string, currency pair.CurrencyItem) bool {
	m.Lock()
	defer m.Unlock()
	for _, y := range Tickers {
		if y.ExchangeName == exchange {
			if _, ok := y.Price[currency]; ok {
				return true
			}
		}
	}
	return false
}

// SecondCurrencyExists checks to see if the second currency of the Price map
// exists
func SecondCurrencyExists(exchange string, p pair.CurrencyPair) bool {
	m.Lock()
	defer m.Unlock()
	for _, y := range Tickers {
		if y.ExchangeName == exchange {
			if _, ok := y.Price[p.GetFirstCurrency()]; ok {
				if _, ok := y.Price[p.GetFirstCurrency()][p.GetSecondCurrency()]; ok {
					return true
				}
			}
		}
	}
	return false
}

// CreateNewTicker creates a new Ticker
func CreateNewTicker(exchangeName string, p pair.CurrencyPair, tickerNew Price, tickerType string) Ticker {
	m.Lock()
	defer m.Unlock()
	ticker := Ticker{}
	ticker.ExchangeName = exchangeName
	ticker.Price = make(map[pair.CurrencyItem]map[pair.CurrencyItem]map[string]Price)
	a := make(map[pair.CurrencyItem]map[string]Price)
	b := make(map[string]Price)
	b[tickerType] = tickerNew
	a[p.SecondCurrency] = b
	ticker.Price[p.FirstCurrency] = a
	Tickers = append(Tickers, ticker)
	return ticker
}

// ProcessTicker processes incoming tickers, creating or updating the Tickers
// list
func ProcessTicker(exchangeName string, p pair.CurrencyPair, tickerNew Price, tickerType string) {
	if tickerNew.Pair.Pair() == "" {
		// set Pair if not set
		tickerNew.Pair = p
	}

	tickerNew.CurrencyPair = p.Pair().String()
	tickerNew.LastUpdated = time.Now()
	if len(Tickers) == 0 {
		CreateNewTicker(exchangeName, p, tickerNew, tickerType)
		return
	}

	ticker, err := GetTickerByExchange(exchangeName)
	if err != nil {
		CreateNewTicker(exchangeName, p, tickerNew, tickerType)
		return
	}

	if FirstCurrencyExists(exchangeName, p.FirstCurrency) {
		if !SecondCurrencyExists(exchangeName, p) {
			m.Lock()
			a := ticker.Price[p.FirstCurrency]
			b := make(map[string]Price)
			b[tickerType] = tickerNew
			a[p.SecondCurrency] = b
			ticker.Price[p.FirstCurrency] = a
			m.Unlock()
			return
		}
	}

	m.Lock()
	a := make(map[pair.CurrencyItem]map[string]Price)
	b := make(map[string]Price)
	b[tickerType] = tickerNew
	a[p.SecondCurrency] = b
	ticker.Price[p.FirstCurrency] = a
	m.Unlock()
}
