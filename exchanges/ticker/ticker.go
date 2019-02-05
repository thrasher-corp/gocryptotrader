package ticker

import (
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
)

// Const values for the ticker package
const (
	ErrTickerForExchangeNotFound = "ticker for exchange does not exist"
	ErrPrimaryCurrencyNotFound   = "primary currency for ticker not found"
	ErrSecondaryCurrencyNotFound = "secondary currency for ticker not found"

	Spot = "SPOT"
)

// Vars for the ticker package
var (
	Tickers []Ticker
	m       sync.Mutex
)

// Price struct stores the currency pair and pricing information
type Price struct {
	Pair         currency.Pair `json:"Pair"`
	LastUpdated  time.Time     `json:"LastUpdated"`
	CurrencyPair string        `json:"CurrencyPair"`
	Last         float64       `json:"Last"`
	High         float64       `json:"High"`
	Low          float64       `json:"Low"`
	Bid          float64       `json:"Bid"`
	Ask          float64       `json:"Ask"`
	Volume       float64       `json:"Volume"`
	PriceATH     float64       `json:"PriceATH"`
}

// Ticker struct holds the ticker information for a currency pair and type
type Ticker struct {
	Price        map[currency.Code]map[currency.Code]map[string]Price
	ExchangeName string
}

// PriceToString returns the string version of a stored price field
func (t *Ticker) PriceToString(p currency.Pair, priceType, tickerType string) string {
	priceType = common.StringToLower(priceType)

	switch priceType {
	case "last":
		return strconv.FormatFloat(t.Price[p.Base][p.Quote][tickerType].Last, 'f', -1, 64)
	case "high":
		return strconv.FormatFloat(t.Price[p.Base][p.Quote][tickerType].High, 'f', -1, 64)
	case "low":
		return strconv.FormatFloat(t.Price[p.Base][p.Quote][tickerType].Low, 'f', -1, 64)
	case "bid":
		return strconv.FormatFloat(t.Price[p.Base][p.Quote][tickerType].Bid, 'f', -1, 64)
	case "ask":
		return strconv.FormatFloat(t.Price[p.Base][p.Quote][tickerType].Ask, 'f', -1, 64)
	case "volume":
		return strconv.FormatFloat(t.Price[p.Base][p.Quote][tickerType].Volume, 'f', -1, 64)
	case "ath":
		return strconv.FormatFloat(t.Price[p.Base][p.Quote][tickerType].PriceATH, 'f', -1, 64)
	default:
		return ""
	}
}

// GetTicker checks and returns a requested ticker if it exists
func GetTicker(exchange string, p currency.Pair, tickerType string) (Price, error) {
	ticker, err := GetTickerByExchange(exchange)
	if err != nil {
		return Price{}, err
	}

	if !FirstCurrencyExists(exchange, p.Base) {
		return Price{}, errors.New(ErrPrimaryCurrencyNotFound)
	}

	if !SecondCurrencyExists(exchange, p) {
		return Price{}, errors.New(ErrSecondaryCurrencyNotFound)
	}

	return ticker.Price[p.Base][p.Quote][tickerType], nil
}

// GetTickerByExchange returns an exchange Ticker
func GetTickerByExchange(exchange string) (*Ticker, error) {
	m.Lock()
	defer m.Unlock()
	for x := range Tickers {
		if Tickers[x].ExchangeName == exchange {
			return &Tickers[x], nil
		}
	}
	return nil, errors.New(ErrTickerForExchangeNotFound)
}

// FirstCurrencyExists checks to see if the first currency of the Price map
// exists
func FirstCurrencyExists(exchange string, currency currency.Code) bool {
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
func SecondCurrencyExists(exchange string, p currency.Pair) bool {
	m.Lock()
	defer m.Unlock()
	for _, y := range Tickers {
		if y.ExchangeName == exchange {
			if _, ok := y.Price[p.Base]; ok {
				if _, ok := y.Price[p.Base][p.Quote]; ok {
					return true
				}
			}
		}
	}
	return false
}

// CreateNewTicker creates a new Ticker
func CreateNewTicker(exchangeName string, p currency.Pair, tickerNew Price, tickerType string) Ticker {
	m.Lock()
	defer m.Unlock()
	ticker := Ticker{}
	ticker.ExchangeName = exchangeName
	ticker.Price = make(map[currency.Code]map[currency.Code]map[string]Price)
	a := make(map[currency.Code]map[string]Price)
	b := make(map[string]Price)
	b[tickerType] = tickerNew
	a[p.Quote] = b
	ticker.Price[p.Base] = a
	Tickers = append(Tickers, ticker)
	return ticker
}

// ProcessTicker processes incoming tickers, creating or updating the Tickers
// list
func ProcessTicker(exchangeName string, p currency.Pair, tickerNew Price, tickerType string) {
	if tickerNew.Pair.String() == "" {
		// set Pair if not set
		tickerNew.Pair = p
	}

	tickerNew.CurrencyPair = p.String()
	tickerNew.LastUpdated = time.Now()

	ticker, err := GetTickerByExchange(exchangeName)
	if err != nil {
		CreateNewTicker(exchangeName, p, tickerNew, tickerType)
		return
	}

	if FirstCurrencyExists(exchangeName, p.Base) {
		m.Lock()
		a := make(map[string]Price)
		a[tickerType] = tickerNew
		ticker.Price[p.Base][p.Quote] = a
		m.Unlock()
		return
	}

	m.Lock()
	a := make(map[currency.Code]map[string]Price)
	b := make(map[string]Price)
	b[tickerType] = tickerNew
	a[p.Quote] = b
	ticker.Price[p.Base] = a
	m.Unlock()
}
