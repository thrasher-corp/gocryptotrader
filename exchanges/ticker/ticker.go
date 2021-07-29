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

var (
	errInvalidTicker       = errors.New("invalid ticker")
	errTickerNotFound      = errors.New("ticker not found")
	errExchangeNameIsEmpty = errors.New("exchange name is empty")
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
	service.Lock()
	defer service.Unlock()

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
	service.Lock()
	defer service.Unlock()
	id, ok := service.Exchange[exchange]
	if !ok {
		return dispatch.Pipe{}, fmt.Errorf("%s exchange tickers not found",
			exchange)
	}

	return service.mux.Subscribe(id)
}

// GetTicker checks and returns a requested ticker if it exists
func GetTicker(exchange string, p currency.Pair, a asset.Item) (*Price, error) {
	exchange = strings.ToLower(exchange)
	service.Lock()
	defer service.Unlock()
	m1, ok := service.Tickers[exchange]
	if !ok {
		return nil, fmt.Errorf("no tickers for %s exchange", exchange)
	}

	m2, ok := m1[p.Base.Item]
	if !ok {
		return nil, fmt.Errorf("no tickers associated with base currency %s",
			p.Base)
	}

	m3, ok := m2[p.Quote.Item]
	if !ok {
		return nil, fmt.Errorf("no tickers associated with quote currency %s",
			p.Quote)
	}

	t, ok := m3[a]
	if !ok {
		return nil, fmt.Errorf("no tickers associated with asset type %s",
			a)
	}

	cpy := t.Price // Don't let external functions have access to underlying
	return &cpy, nil
}

// FindLast searches for a currency pair and returns the first available
func FindLast(p currency.Pair, a asset.Item) (float64, error) {
	service.Lock()
	defer service.Unlock()
	for _, m1 := range service.Tickers {
		m2, ok := m1[p.Base.Item]
		if !ok {
			continue
		}
		m3, ok := m2[p.Quote.Item]
		if !ok {
			continue
		}
		t, ok := m3[a]
		if !ok {
			continue
		}

		if t.Last == 0 {
			return 0, errInvalidTicker
		}
		return t.Last, nil
	}
	return 0, fmt.Errorf("%w %s %s", errTickerNotFound, p, a)
}

// ProcessTicker processes incoming tickers, creating or updating the Tickers
// list
func ProcessTicker(p *Price) error {
	if p == nil {
		return errors.New(errTickerPriceIsNil)
	}

	if p.ExchangeName == "" {
		return fmt.Errorf(ErrExchangeNameUnset)
	}

	if p.Pair.IsEmpty() {
		return fmt.Errorf("%s %s", p.ExchangeName, errPairNotSet)
	}

	if p.AssetType == "" {
		return fmt.Errorf("%s %s %s",
			p.ExchangeName,
			p.Pair,
			errAssetTypeNotSet)
	}

	if p.LastUpdated.IsZero() {
		p.LastUpdated = time.Now()
	}

	return service.update(p)
}

// update updates ticker price
func (s *Service) update(p *Price) error {
	name := strings.ToLower(p.ExchangeName)
	s.Lock()

	m1, ok := service.Tickers[name]
	if !ok {
		m1 = make(map[*currency.Item]map[*currency.Item]map[asset.Item]*Ticker)
		service.Tickers[name] = m1
	}

	m2, ok := m1[p.Pair.Base.Item]
	if !ok {
		m2 = make(map[*currency.Item]map[asset.Item]*Ticker)
		m1[p.Pair.Base.Item] = m2
	}

	m3, ok := m2[p.Pair.Quote.Item]
	if !ok {
		m3 = make(map[asset.Item]*Ticker)
		m2[p.Pair.Quote.Item] = m3
	}

	t, ok := m3[p.AssetType]
	if !ok || t == nil {
		newTicker := &Ticker{}
		err := s.setItemID(newTicker, p, name)
		if err != nil {
			s.Unlock()
			return err
		}
		m3[p.AssetType] = newTicker
		s.Unlock()
		return nil
	}

	t.Price = *p
	ids := append(t.Assoc, t.Main)
	s.Unlock()
	return s.mux.Publish(ids, p)
}

// setItemID retrieves and sets dispatch mux publish IDs
func (s *Service) setItemID(t *Ticker, p *Price, exch string) error {
	ids, err := s.getAssociations(exch)
	if err != nil {
		return err
	}
	singleID, err := s.mux.GetID()
	if err != nil {
		return err
	}

	t.Price = *p
	t.Main = singleID
	t.Assoc = ids
	return nil
}

// getAssociations links a singular book with it's dispatch associations
func (s *Service) getAssociations(exch string) ([]uuid.UUID, error) {
	if exch == "" {
		return nil, errExchangeNameIsEmpty
	}
	var ids []uuid.UUID
	exchangeID, ok := s.Exchange[exch]
	if !ok {
		var err error
		exchangeID, err = s.mux.GetID()
		if err != nil {
			return nil, err
		}
		s.Exchange[exch] = exchangeID
	}
	ids = append(ids, exchangeID)
	return ids, nil
}
