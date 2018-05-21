package base

import (
	"fmt"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

//global vars contain staged update data that will be sent to the communication
// mediums
var (
	TickerStaged    map[string]map[string]map[string]ticker.Price
	OrderbookStaged map[string]map[string]map[string]Orderbook
	PortfolioStaged Portfolio
	SettingsStaged  Settings
	ServiceStarted  time.Time
	m               sync.Mutex
)

// Orderbook holds the minimal orderbook details to be sent to a communication
// medium
type Orderbook struct {
	CurrencyPair string
	AssetType    string
	TotalAsks    float64
	TotalBids    float64
	LastUpdated  string
}

// Ticker holds the minimal orderbook details to be sent to a communication
// medium
type Ticker struct {
	CurrencyPair string
	LastUpdated  string
}

// Portfolio holds the minimal portfolio details to be sent to a communication
// medium
type Portfolio struct {
	ProfitLoss string
}

// Settings holds the minimal setting details to be sent to a communication
// medium
type Settings struct {
	EnabledExchanges      string
	EnabledCommunications string
}

// Base enforces standard variables across communication packages
type Base struct {
	Name      string
	Enabled   bool
	Verbose   bool
	Connected bool
}

// Event is a generalise event type
type Event struct {
	Type         string
	GainLoss     string
	TradeDetails string
}

// IsEnabled returns if the comms package has been enabled in the configuration
func (b *Base) IsEnabled() bool {
	return b.Enabled
}

// IsConnected returns if the package is connected to a server and/or ready to
// send
func (b *Base) IsConnected() bool {
	return b.Connected
}

// GetName returns a package name
func (b *Base) GetName() string {
	return b.Name
}

// GetTicker returns staged ticker data
func (b *Base) GetTicker(exchangeName string) string {
	m.Lock()
	defer m.Unlock()

	tickerPrice, ok := TickerStaged[exchangeName]
	if !ok {
		return ""
	}

	var tickerPrices []ticker.Price
	for _, x := range tickerPrice {
		for _, y := range x {
			tickerPrices = append(tickerPrices, y)
		}
	}

	var packagedTickers []string
	for i := range tickerPrices {
		packagedTickers = append(packagedTickers, fmt.Sprintf(
			"Currency Pair: %s Ask: %f, Bid: %f High: %f Last: %f Low: %f ATH: %f Volume: %f",
			tickerPrices[i].CurrencyPair,
			tickerPrices[i].Ask,
			tickerPrices[i].Bid,
			tickerPrices[i].High,
			tickerPrices[i].Last,
			tickerPrices[i].Low,
			tickerPrices[i].PriceATH,
			tickerPrices[i].Volume))
	}
	return common.JoinStrings(packagedTickers, "\n")
}

// GetOrderbook returns staged orderbook data
func (b *Base) GetOrderbook(exchangeName string) string {
	m.Lock()
	defer m.Unlock()

	orderbook, ok := OrderbookStaged[exchangeName]
	if !ok {
		return ""
	}

	var orderbooks []Orderbook
	for _, x := range orderbook {
		for _, y := range x {
			orderbooks = append(orderbooks, y)
		}
	}

	var packagedOrderbooks []string
	for i := range orderbooks {
		packagedOrderbooks = append(packagedOrderbooks, fmt.Sprintf(
			"Currency Pair: %s AssetType: %s, LastUpdated: %s TotalAsks: %f TotalBids: %f",
			orderbooks[i].CurrencyPair,
			orderbooks[i].AssetType,
			orderbooks[i].LastUpdated,
			orderbooks[i].TotalAsks,
			orderbooks[i].TotalBids))
	}
	return common.JoinStrings(packagedOrderbooks, "\n")
}

// GetPortfolio returns staged portfolio info
func (b *Base) GetPortfolio() string {
	m.Lock()
	defer m.Unlock()
	return fmt.Sprintf("%v", PortfolioStaged)
}

// GetSettings returns stage setting info
func (b *Base) GetSettings() string {
	m.Lock()
	defer m.Unlock()
	return fmt.Sprintf("%v", SettingsStaged)
}

// GetStatus returns status data
func (b *Base) GetStatus() string {
	return `
	GoCryptoTrader Service: Online
	Service Started: ` + ServiceStarted.String()
}
