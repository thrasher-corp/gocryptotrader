package ticker

import (
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
)

// Const values for the ticker package
const (
	ErrTickerForExchangeNotFound = "ticker for exchange does not exist"
	ErrPrimaryCurrencyNotFound   = "error primary currency for ticker not found"
	ErrSecondaryCurrencyNotFound = "error secondary currency for ticker not found"
)

// Vars for the ticker package
var (
	Tickers []Ticker
	m       sync.Mutex
)

// Price struct stores the currency pair and pricing information
type Price struct {
	Pair        currency.Pair `json:"Pair"`
	Last        float64       `json:"Last"`
	High        float64       `json:"High"`
	Low         float64       `json:"Low"`
	Bid         float64       `json:"Bid"`
	Ask         float64       `json:"Ask"`
	Volume      float64       `json:"Volume"`
	PriceATH    float64       `json:"PriceATH"`
	LastUpdated time.Time
}

// Ticker struct holds the ticker information for a currency pair and type
type Ticker struct {
	Price        map[string]map[string]map[string]Price
	ExchangeName string
}

// PriceToString returns the string version of a stored price field
func (t *Ticker) PriceToString(p currency.Pair, priceType string, tickerType assets.AssetType) string {
	priceType = common.StringToLower(priceType)

	switch priceType {
	case "last":
		return strconv.FormatFloat(t.Price[p.Base.Upper().String()][p.Quote.Upper().String()][tickerType.String()].Last, 'f', -1, 64)
	case "high":
		return strconv.FormatFloat(t.Price[p.Base.Upper().String()][p.Quote.Upper().String()][tickerType.String()].High, 'f', -1, 64)
	case "low":
		return strconv.FormatFloat(t.Price[p.Base.Upper().String()][p.Quote.Upper().String()][tickerType.String()].Low, 'f', -1, 64)
	case "bid":
		return strconv.FormatFloat(t.Price[p.Base.Upper().String()][p.Quote.Upper().String()][tickerType.String()].Bid, 'f', -1, 64)
	case "ask":
		return strconv.FormatFloat(t.Price[p.Base.Upper().String()][p.Quote.Upper().String()][tickerType.String()].Ask, 'f', -1, 64)
	case "volume":
		return strconv.FormatFloat(t.Price[p.Base.Upper().String()][p.Quote.Upper().String()][tickerType.String()].Volume, 'f', -1, 64)
	case "ath":
		return strconv.FormatFloat(t.Price[p.Base.Upper().String()][p.Quote.Upper().String()][tickerType.String()].PriceATH, 'f', -1, 64)
	default:
		return ""
	}
}

// GetTicker checks and returns a requested ticker if it exists
func GetTicker(exchange string, p currency.Pair, tickerType assets.AssetType) (Price, error) {
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

	return ticker.Price[p.Base.Upper().String()][p.Quote.Upper().String()][tickerType.String()], nil
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
			if _, ok := y.Price[currency.Upper().String()]; ok {
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
			if _, ok := y.Price[p.Base.Upper().String()]; ok {
				if _, ok := y.Price[p.Base.Upper().String()][p.Quote.Upper().String()]; ok {
					return true
				}
			}
		}
	}
	return false
}

// CreateNewTicker creates a new Ticker
func CreateNewTicker(exchangeName string, tickerNew *Price, tickerType assets.AssetType) Ticker {
	m.Lock()
	defer m.Unlock()
	ticker := Ticker{}
	ticker.ExchangeName = exchangeName
	ticker.Price = make(map[string]map[string]map[string]Price)
	a := make(map[string]map[string]Price)
	b := make(map[string]Price)
	b[tickerType.String()] = *tickerNew
	a[tickerNew.Pair.Quote.Upper().String()] = b
	ticker.Price[tickerNew.Pair.Base.Upper().String()] = a
	Tickers = append(Tickers, ticker)
	return ticker
}

// ProcessTicker processes incoming tickers, creating or updating the Tickers
// list
func ProcessTicker(exchangeName string, tickerNew *Price, tickerType assets.AssetType) error {
	if tickerNew.Pair.String() == "" {
		return errors.New("")
	}

	tickerNew.LastUpdated = time.Now()

	ticker, err := GetTickerByExchange(exchangeName)
	if err != nil {
		CreateNewTicker(exchangeName, tickerNew, tickerType)
		return nil
	}

	if FirstCurrencyExists(exchangeName, tickerNew.Pair.Base) {
		m.Lock()
		a := make(map[string]Price)
		a[tickerType.String()] = *tickerNew
		ticker.Price[tickerNew.Pair.Base.Upper().String()][tickerNew.Pair.Quote.Upper().String()] = a
		m.Unlock()
		return nil
	}

	m.Lock()
	a := make(map[string]map[string]Price)
	b := make(map[string]Price)
	b[tickerType.String()] = *tickerNew
	a[tickerNew.Pair.Quote.Upper().String()] = b
	ticker.Price[tickerNew.Pair.Base.Upper().String()] = a
	m.Unlock()
	return nil
}
