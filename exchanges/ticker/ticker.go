package ticker

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// const values for the ticker package
const (
	errExchangeTickerNotFound = "ticker for exchange does not exist"
	errPairNotSet             = "ticker currency pair not set"
	errAssetTypeNotSet        = "ticker asset type not set"
	errBaseCurrencyNotFound   = "ticker base currency not found"
	errQuoteCurrencyNotFound  = "ticker quote currency not found"
)

// Vars for the ticker package
var (
	Tickers []Ticker
	m       sync.Mutex
)

// Price struct stores the currency pair and pricing information
type Price struct {
	Last        float64       `json:"Last"`
	High        float64       `json:"High"`
	Low         float64       `json:"Low"`
	Bid         float64       `json:"Bid"`
	Ask         float64       `json:"Ask"`
	Volume      float64       `json:"Volume"`
	PriceATH    float64       `json:"PriceATH"`
	Open        float64       `json:"Open"`
	Close       float64       `json:"Close"`
	Pair        currency.Pair `json:"Pair"`
	LastUpdated time.Time
}

// Ticker struct holds the ticker information for a currency pair and type
type Ticker struct {
	Price        map[string]map[string]map[asset.Item]Price
	ExchangeName string
}

// PriceToString returns the string version of a stored price field
func (t *Ticker) PriceToString(p currency.Pair, priceType string, tickerType asset.Item) string {
	priceType = strings.ToLower(priceType)

	switch priceType {
	case "last":
		return strconv.FormatFloat(t.Price[p.Base.Upper().String()][p.Quote.Upper().String()][tickerType].Last, 'f', -1, 64)
	case "high":
		return strconv.FormatFloat(t.Price[p.Base.Upper().String()][p.Quote.Upper().String()][tickerType].High, 'f', -1, 64)
	case "low":
		return strconv.FormatFloat(t.Price[p.Base.Upper().String()][p.Quote.Upper().String()][tickerType].Low, 'f', -1, 64)
	case "bid":
		return strconv.FormatFloat(t.Price[p.Base.Upper().String()][p.Quote.Upper().String()][tickerType].Bid, 'f', -1, 64)
	case "ask":
		return strconv.FormatFloat(t.Price[p.Base.Upper().String()][p.Quote.Upper().String()][tickerType].Ask, 'f', -1, 64)
	case "volume":
		return strconv.FormatFloat(t.Price[p.Base.Upper().String()][p.Quote.Upper().String()][tickerType].Volume, 'f', -1, 64)
	case "ath":
		return strconv.FormatFloat(t.Price[p.Base.Upper().String()][p.Quote.Upper().String()][tickerType].PriceATH, 'f', -1, 64)
	default:
		return ""
	}
}

// GetTicker checks and returns a requested ticker if it exists
func GetTicker(exchange string, p currency.Pair, tickerType asset.Item) (Price, error) {
	ticker, err := GetTickerByExchange(exchange)
	if err != nil {
		return Price{}, err
	}

	if !BaseCurrencyExists(exchange, p.Base) {
		return Price{}, errors.New(errBaseCurrencyNotFound)
	}

	if !QuoteCurrencyExists(exchange, p) {
		return Price{}, errors.New(errQuoteCurrencyNotFound)
	}

	return ticker.Price[p.Base.Upper().String()][p.Quote.Upper().String()][tickerType], nil
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
	return nil, errors.New(errExchangeTickerNotFound)
}

// BaseCurrencyExists checks to see if the base currency of the ticker map
// exists
func BaseCurrencyExists(exchange string, currency currency.Code) bool {
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

// QuoteCurrencyExists checks to see if the quote currency of the ticker map
// exists
func QuoteCurrencyExists(exchange string, p currency.Pair) bool {
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
func CreateNewTicker(exchangeName string, tickerNew *Price, tickerType asset.Item) Ticker {
	m.Lock()
	defer m.Unlock()
	ticker := Ticker{}
	ticker.ExchangeName = exchangeName
	ticker.Price = make(map[string]map[string]map[asset.Item]Price)
	a := make(map[string]map[asset.Item]Price)
	b := make(map[asset.Item]Price)
	b[tickerType] = *tickerNew
	a[tickerNew.Pair.Quote.Upper().String()] = b
	ticker.Price[tickerNew.Pair.Base.Upper().String()] = a
	Tickers = append(Tickers, ticker)
	return ticker
}

// ProcessTicker processes incoming tickers, creating or updating the Tickers
// list
func ProcessTicker(exchangeName string, tickerNew *Price, assetType asset.Item) error {
	if tickerNew.Pair.IsEmpty() {
		return errors.New(errPairNotSet)
	}

	if assetType == "" {
		return errors.New(errAssetTypeNotSet)
	}

	if tickerNew.LastUpdated.IsZero() {
		tickerNew.LastUpdated = time.Now()
	}

	ticker, err := GetTickerByExchange(exchangeName)
	if err != nil {
		CreateNewTicker(exchangeName, tickerNew, assetType)
		return nil
	}

	if BaseCurrencyExists(exchangeName, tickerNew.Pair.Base) {
		m.Lock()
		a := make(map[asset.Item]Price)
		a[assetType] = *tickerNew
		ticker.Price[tickerNew.Pair.Base.Upper().String()][tickerNew.Pair.Quote.Upper().String()] = a
		m.Unlock()
		return nil
	}

	m.Lock()
	a := make(map[string]map[asset.Item]Price)
	b := make(map[asset.Item]Price)
	b[assetType] = *tickerNew
	a[tickerNew.Pair.Quote.Upper().String()] = b
	ticker.Price[tickerNew.Pair.Base.Upper().String()] = a
	m.Unlock()
	return nil
}
