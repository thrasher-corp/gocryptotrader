package base

import (
	"time"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// IComm is the main interface array across the communication packages
type IComm []ICommunicate

// ICommunicate enforces standard functions across communication packages
type ICommunicate interface {
	Setup(config *config.CommunicationsConfig)
	Connect() error
	PushEvent(Event) error
	IsEnabled() bool
	IsConnected() bool
	GetName() string
}

// Setup sets up communication variables and intiates a connection to the
// communication mediums
func (c IComm) Setup() {
	TickerStaged = make(map[string]map[assets.AssetType]map[string]ticker.Price)
	OrderbookStaged = make(map[string]map[assets.AssetType]map[string]Orderbook)
	ServiceStarted = time.Now()

	for i := range c {
		if c[i].IsEnabled() && !c[i].IsConnected() {
			err := c[i].Connect()
			if err != nil {
				log.Errorf("Communications: %s failed to connect. Err: %s", c[i].GetName(), err)
			}
		}
	}
}

// PushEvent pushes triggered events to all enabled communication links
func (c IComm) PushEvent(event Event) {
	for i := range c {
		if c[i].IsEnabled() && c[i].IsConnected() {
			err := c[i].PushEvent(event)
			if err != nil {
				log.Errorf("Communications error - PushEvent() in package %s with %v",
					c[i].GetName(), event)
			}
		}
	}
}

// GetEnabledCommunicationMediums prints out enabled and connected communication
// packages
func (c IComm) GetEnabledCommunicationMediums() {
	var count int
	for i := range c {
		if c[i].IsEnabled() && c[i].IsConnected() {
			log.Debugf("Communications: Medium %s is enabled.", c[i].GetName())
			count++
		}
	}
	if count == 0 {
		log.Warnf("Communications: No communication mediums are enabled.")
	}
}

// StageTickerData stages updated ticker data for the communications package
func (c IComm) StageTickerData(exchangeName string, assetType assets.AssetType, tickerPrice *ticker.Price) {
	m.Lock()
	defer m.Unlock()

	if _, ok := TickerStaged[exchangeName]; !ok {
		TickerStaged[exchangeName] = make(map[assets.AssetType]map[string]ticker.Price)
	}

	if _, ok := TickerStaged[exchangeName][assetType]; !ok {
		TickerStaged[exchangeName][assetType] = make(map[string]ticker.Price)
	}

	TickerStaged[exchangeName][assetType][tickerPrice.Pair.String()] = *tickerPrice
}

// StageOrderbookData stages updated orderbook data for the communications
// package
func (c IComm) StageOrderbookData(exchangeName string, assetType assets.AssetType, ob *orderbook.Base) {
	m.Lock()
	defer m.Unlock()

	if _, ok := OrderbookStaged[exchangeName]; !ok {
		OrderbookStaged[exchangeName] = make(map[assets.AssetType]map[string]Orderbook)
	}

	if _, ok := OrderbookStaged[exchangeName][assetType]; !ok {
		OrderbookStaged[exchangeName][assetType] = make(map[string]Orderbook)
	}

	_, totalAsks := ob.TotalAsksAmount()
	_, totalBids := ob.TotalBidsAmount()

	OrderbookStaged[exchangeName][assetType][ob.Pair.String()] = Orderbook{
		CurrencyPair: ob.Pair.String(),
		TotalAsks:    totalAsks,
		TotalBids:    totalBids}
}
