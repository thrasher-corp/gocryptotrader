package ticker

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func init() {
	service = new(Service)
	service.Tickers = make(map[string]map[*currency.Item]map[*currency.Item]map[asset.Item]*Ticker)
	service.Exchange = make(map[string]uuid.UUID)
	service.mux = dispatch.GetNewMux()
}

// SubscribeTicker subcribes to a ticker and returns a communication channel to
// stream new ticker updates
func SubscribeTicker(exchange string, p currency.Pair, a asset.Item) (dispatch.Pipe, error) {
	exchange = strings.ToLower(exchange)
	service.RLock()
	defer service.RUnlock()

	tick, ok := service.Tickers[exchange][p.Base.Item][p.Quote.Item][a]
	if !ok {
		return dispatch.Pipe{}, fmt.Errorf("ticker item not found for %s %s %s",
			exchange,
			p,
			a)
	}
	return service.mux.Subscribe(tick.Main)
}

// SubscribeToExchangeTickers subcribes to all tickers on an exchange
func SubscribeToExchangeTickers(exchange string) (dispatch.Pipe, error) {
	exchange = strings.ToLower(exchange)
	service.RLock()
	defer service.RUnlock()
	id, ok := service.Exchange[exchange]
	if !ok {
		return dispatch.Pipe{}, fmt.Errorf("%s exchange tickers not found",
			exchange)
	}

	return service.mux.Subscribe(id)
}

// GetTicker checks and returns a requested ticker if it exists
func GetTicker(exchange string, p currency.Pair, tickerType asset.Item) (*Price, error) {
	exchange = strings.ToLower(exchange)
	service.RLock()
	defer service.RUnlock()
	if service.Tickers[exchange] == nil {
		return nil, fmt.Errorf("no tickers for %s exchange", exchange)
	}

	if service.Tickers[exchange][p.Base.Item] == nil {
		return nil, fmt.Errorf("no tickers associated with base currency %s",
			p.Base)
	}

	if service.Tickers[exchange][p.Base.Item][p.Quote.Item] == nil {
		return nil, fmt.Errorf("no tickers associated with quote currency %s",
			p.Quote)
	}

	if service.Tickers[exchange][p.Base.Item][p.Quote.Item][tickerType] == nil {
		return nil, fmt.Errorf("no tickers associated with asset type %s",
			tickerType)
	}

	return &service.Tickers[exchange][p.Base.Item][p.Quote.Item][tickerType].Price, nil
}

// ProcessTicker processes incoming tickers, creating or updating the Tickers
// list
func ProcessTicker(tickerNew *Price) error {
	if tickerNew.ExchangeName == "" {
		return fmt.Errorf(errExchangeNameUnset)
	}

	if tickerNew.Pair.IsEmpty() {
		return fmt.Errorf("%s %s", tickerNew.ExchangeName, errPairNotSet)
	}

	if tickerNew.AssetType == "" {
		return fmt.Errorf("%s %s %s",
			tickerNew.ExchangeName,
			tickerNew.Pair,
			errAssetTypeNotSet)
	}

	if tickerNew.LastUpdated.IsZero() {
		tickerNew.LastUpdated = time.Now()
	}

	return service.Update(tickerNew)
}

// Update updates ticker price
func (s *Service) Update(p *Price) error {
	name := strings.ToLower(p.ExchangeName)
	s.Lock()

	ticker, ok := s.Tickers[name][p.Pair.Base.Item][p.Pair.Quote.Item][p.AssetType]
	if ok {
		ticker.Last = p.Last
		ticker.High = p.High
		ticker.Low = p.Low
		ticker.Bid = p.Bid
		ticker.Ask = p.Ask
		ticker.Volume = p.Volume
		ticker.QuoteVolume = p.QuoteVolume
		ticker.PriceATH = p.PriceATH
		ticker.Open = p.Open
		ticker.Close = p.Close
		ticker.LastUpdated = p.LastUpdated
		ids := append(ticker.Assoc, ticker.Main)
		s.Unlock()
		return s.mux.Publish(ids, p)
	}

	switch {
	case s.Tickers[name] == nil:
		s.Tickers[name] = make(map[*currency.Item]map[*currency.Item]map[asset.Item]*Ticker)
		fallthrough
	case s.Tickers[name][p.Pair.Base.Item] == nil:
		s.Tickers[name][p.Pair.Base.Item] = make(map[*currency.Item]map[asset.Item]*Ticker)
		fallthrough
	case s.Tickers[name][p.Pair.Base.Item][p.Pair.Quote.Item] == nil:
		s.Tickers[name][p.Pair.Base.Item][p.Pair.Quote.Item] = make(map[asset.Item]*Ticker)
	}

	err := s.SetItemID(p, name)
	if err != nil {
		s.Unlock()
		return err
	}

	s.Unlock()
	return nil
}

// SetItemID retrieves and sets dispatch mux publish IDs
func (s *Service) SetItemID(p *Price, fmtName string) error {
	if p == nil {
		return errors.New(errTickerPriceIsNil)
	}

	ids, err := s.GetAssociations(p, fmtName)
	if err != nil {
		return err
	}
	singleID, err := s.mux.GetID()
	if err != nil {
		return err
	}

	s.Tickers[fmtName][p.Pair.Base.Item][p.Pair.Quote.Item][p.AssetType] = &Ticker{Price: *p,
		Main:  singleID,
		Assoc: ids}
	return nil
}

// GetAssociations links a singular book with it's dispatch associations
func (s *Service) GetAssociations(p *Price, fmtName string) ([]uuid.UUID, error) {
	if p == nil || *p == (Price{}) {
		return nil, errors.New(errTickerPriceIsNil)
	}
	var ids []uuid.UUID
	exchangeID, ok := s.Exchange[fmtName]
	if !ok {
		var err error
		exchangeID, err = s.mux.GetID()
		if err != nil {
			return nil, err
		}
		s.Exchange[fmtName] = exchangeID
	}

	ids = append(ids, exchangeID)
	return ids, nil
}
