package ticker

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/dispatch"
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
	service *Service
)

func init() {
	service = new(Service)
	service.Tickers = make(map[string]map[*currency.Item]map[*currency.Item]map[asset.Item]*Ticker)
	service.Exchange = make(map[string]uuid.UUID)
	service.mux = dispatch.GetNewMux()
}

// Service holds ticker information for each individual exchange
type Service struct {
	Tickers  map[string]map[*currency.Item]map[*currency.Item]map[asset.Item]*Ticker
	Exchange map[string]uuid.UUID
	mux      *dispatch.Mux
	sync.RWMutex
}

// Price struct stores the currency pair and pricing information
type Price struct {
	Last         float64       `json:"Last"`
	High         float64       `json:"High"`
	Low          float64       `json:"Low"`
	Bid          float64       `json:"Bid"`
	Ask          float64       `json:"Ask"`
	Volume       float64       `json:"Volume"`
	QuoteVolume  float64       `json:"QuoteVolume"`
	PriceATH     float64       `json:"PriceATH"`
	Open         float64       `json:"Open"`
	Close        float64       `json:"Close"`
	Pair         currency.Pair `json:"Pair"`
	ExchangeName string        `json:"exchangeName"`
	AssetType    asset.Item    `json:"assetType"`
	LastUpdated  time.Time
}

// Ticker struct holds the ticker information for a currency pair and type
type Ticker struct {
	Price
	Main  uuid.UUID
	Assoc []uuid.UUID
}

// Update updates ticker price
func (s *Service) Update(p *Price) error {
	var ids []uuid.UUID

	s.Lock()
	switch {
	case s.Tickers[p.ExchangeName] == nil:
		s.Tickers[p.ExchangeName] = make(map[*currency.Item]map[*currency.Item]map[asset.Item]*Ticker)
		s.Tickers[p.ExchangeName][p.Pair.Base.Item] = make(map[*currency.Item]map[asset.Item]*Ticker)
		s.Tickers[p.ExchangeName][p.Pair.Base.Item][p.Pair.Quote.Item] = make(map[asset.Item]*Ticker)
		err := s.SetNewData(p)
		if err != nil {
			s.Unlock()
			return err
		}

	case s.Tickers[p.ExchangeName][p.Pair.Base.Item] == nil:
		s.Tickers[p.ExchangeName][p.Pair.Base.Item] = make(map[*currency.Item]map[asset.Item]*Ticker)
		s.Tickers[p.ExchangeName][p.Pair.Base.Item][p.Pair.Quote.Item] = make(map[asset.Item]*Ticker)
		err := s.SetNewData(p)
		if err != nil {
			s.Unlock()
			return err
		}

	case s.Tickers[p.ExchangeName][p.Pair.Base.Item][p.Pair.Quote.Item] == nil:
		s.Tickers[p.ExchangeName][p.Pair.Base.Item][p.Pair.Quote.Item] = make(map[asset.Item]*Ticker)
		err := s.SetNewData(p)
		if err != nil {
			s.Unlock()
			return err
		}

	case s.Tickers[p.ExchangeName][p.Pair.Base.Item][p.Pair.Quote.Item][p.AssetType] == nil:
		err := s.SetNewData(p)
		if err != nil {
			s.Unlock()
			return err
		}

	default:
		s.Tickers[p.ExchangeName][p.Pair.Base.Item][p.Pair.Quote.Item][p.AssetType].Last = p.Last
		s.Tickers[p.ExchangeName][p.Pair.Base.Item][p.Pair.Quote.Item][p.AssetType].High = p.High
		s.Tickers[p.ExchangeName][p.Pair.Base.Item][p.Pair.Quote.Item][p.AssetType].Low = p.Low
		s.Tickers[p.ExchangeName][p.Pair.Base.Item][p.Pair.Quote.Item][p.AssetType].Bid = p.Bid
		s.Tickers[p.ExchangeName][p.Pair.Base.Item][p.Pair.Quote.Item][p.AssetType].Ask = p.Ask
		s.Tickers[p.ExchangeName][p.Pair.Base.Item][p.Pair.Quote.Item][p.AssetType].Volume = p.Volume
		s.Tickers[p.ExchangeName][p.Pair.Base.Item][p.Pair.Quote.Item][p.AssetType].QuoteVolume = p.QuoteVolume
		s.Tickers[p.ExchangeName][p.Pair.Base.Item][p.Pair.Quote.Item][p.AssetType].PriceATH = p.PriceATH
		s.Tickers[p.ExchangeName][p.Pair.Base.Item][p.Pair.Quote.Item][p.AssetType].Open = p.Open
		s.Tickers[p.ExchangeName][p.Pair.Base.Item][p.Pair.Quote.Item][p.AssetType].Close = p.Close
		s.Tickers[p.ExchangeName][p.Pair.Base.Item][p.Pair.Quote.Item][p.AssetType].LastUpdated = p.LastUpdated
		ids = s.Tickers[p.ExchangeName][p.Pair.Base.Item][p.Pair.Quote.Item][p.AssetType].Assoc
		ids = append(ids, s.Tickers[p.ExchangeName][p.Pair.Base.Item][p.Pair.Quote.Item][p.AssetType].Main)
	}
	s.Unlock()
	return s.mux.Publish(ids, p)
}

// SetNewData sets new data
func (s *Service) SetNewData(p *Price) error {
	ids, err := s.GetAssociations(p)
	if err != nil {
		return err
	}
	singleID, err := s.mux.GetID()
	if err != nil {
		return err
	}

	s.Tickers[p.ExchangeName][p.Pair.Base.Item][p.Pair.Quote.Item][p.AssetType] = &Ticker{Price: *p,
		Main:  singleID,
		Assoc: ids}
	return nil
}

// GetAssociations links a singular book with it's dispatch associations
func (s *Service) GetAssociations(p *Price) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	exchangeID, ok := s.Exchange[p.ExchangeName]
	if !ok {
		var err error
		exchangeID, err = s.mux.GetID()
		if err != nil {
			return nil, err
		}
		s.Exchange[p.ExchangeName] = exchangeID
	}

	ids = append(ids, exchangeID)
	return ids, nil
}

// SubscribeTicker subcribes to a ticker and returns a communication channel to
// stream new ticker updates
func SubscribeTicker(exchange string, p currency.Pair, a asset.Item) (dispatch.Pipe, error) {
	service.RLock()
	defer service.RUnlock()
	if service.Tickers[exchange][p.Base.Item][p.Quote.Item][a] == nil {
		return dispatch.Pipe{}, errors.New("orderbook item not found")
	}

	tick, ok := service.Tickers[exchange][p.Base.Item][p.Quote.Item][a]
	if !ok {
		return dispatch.Pipe{}, errors.New("orderbook item not found")
	}

	return service.mux.Subscribe(tick.Main)
}

// SubscribeToExchangeTickers subcribes to all tickers on an exchange
func SubscribeToExchangeTickers(exchange string) (dispatch.Pipe, error) {
	service.RLock()
	defer service.RUnlock()
	id, ok := service.Exchange[exchange]
	if !ok {
		return dispatch.Pipe{}, errors.New("exchange orderbooks not found")
	}

	return service.mux.Subscribe(id)
}

// GetTicker checks and returns a requested ticker if it exists
func GetTicker(exchange string, p currency.Pair, tickerType asset.Item) (Price, error) {
	service.RLock()
	defer service.RUnlock()
	if service.Tickers[exchange] == nil {
		return Price{}, errors.New("exchange tickers not found")
	}

	if service.Tickers[exchange][p.Base.Item] == nil {
		return Price{}, errors.New("base currency tickers not found")
	}

	if service.Tickers[exchange][p.Base.Item][p.Quote.Item] == nil {
		return Price{}, errors.New("quote currency tickers not found")
	}

	if service.Tickers[exchange][p.Base.Item][p.Quote.Item][tickerType] == nil {
		return Price{}, errors.New("asset type tickers not found")
	}

	return service.Tickers[exchange][p.Base.Item][p.Quote.Item][tickerType].Price, nil
}

// ProcessTicker processes incoming tickers, creating or updating the Tickers
// list
func ProcessTicker(exchangeName string, tickerNew *Price, assetType asset.Item) error {
	if exchangeName == "" {
		return fmt.Errorf("%s %s", exchangeName, "name not set")
	}

	tickerNew.ExchangeName = exchangeName

	if tickerNew.Pair.IsEmpty() {
		return fmt.Errorf("%s %s", exchangeName, errPairNotSet)
	}

	if assetType == "" {
		return fmt.Errorf("%s %s %s", exchangeName, tickerNew.Pair.String(), errAssetTypeNotSet)
	}

	tickerNew.AssetType = assetType

	if tickerNew.LastUpdated.IsZero() {
		tickerNew.LastUpdated = time.Now()
	}

	return service.Update(tickerNew)
}
