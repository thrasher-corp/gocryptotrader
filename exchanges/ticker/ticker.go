package ticker

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// Public errors
var (
	ErrTickerNotFound = errors.New("no ticker found")
	ErrBidEqualsAsk   = errors.New("bid equals ask this is a crossed or locked market")
)

var (
	errInvalidTicker     = errors.New("invalid ticker")
	errBidGreaterThanAsk = errors.New("bid greater than ask this is a crossed or locked market")
	errExchangeNotFound  = errors.New("exchange not found")
)

func init() {
	service = new(Service)
	service.Tickers = make(map[key.ExchangeAssetPair]*Ticker)
	service.Exchange = make(map[string]uuid.UUID)
	service.mux = dispatch.GetNewMux(nil)
}

// SubscribeTicker subscribes to a ticker and returns a communication channel to
// stream new ticker updates
func SubscribeTicker(exchange string, p currency.Pair, a asset.Item) (dispatch.Pipe, error) {
	exchange = strings.ToLower(exchange)
	service.mu.Lock()
	defer service.mu.Unlock()
	tick, ok := service.Tickers[key.NewExchangeAssetPair(exchange, a, p)]
	if !ok {
		return dispatch.Pipe{}, fmt.Errorf("ticker item not found for %s %s %s",
			exchange,
			p,
			a)
	}
	return service.mux.Subscribe(tick.Main)
}

// SubscribeToExchangeTickers subscribes to all tickers on an exchange
func SubscribeToExchangeTickers(exchange string) (dispatch.Pipe, error) {
	exchange = strings.ToLower(exchange)
	service.mu.Lock()
	defer service.mu.Unlock()
	id, ok := service.Exchange[exchange]
	if !ok {
		return dispatch.Pipe{}, fmt.Errorf("%s exchange tickers not found",
			exchange)
	}

	return service.mux.Subscribe(id)
}

// GetTicker checks and returns a requested ticker if it exists
func GetTicker(exchange string, p currency.Pair, a asset.Item) (*Price, error) {
	if exchange == "" {
		return nil, common.ErrExchangeNameNotSet
	}
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !a.IsValid() {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	exchange = strings.ToLower(exchange)
	service.mu.Lock()
	defer service.mu.Unlock()
	tick, ok := service.Tickers[key.NewExchangeAssetPair(exchange, a, p)]
	if !ok {
		return nil, fmt.Errorf("%w %s %s %s", ErrTickerNotFound, exchange, p, a)
	}

	cpy := tick.Price // Don't let external functions have access to underlying
	return &cpy, nil
}

// GetExchangeTickers returns all tickers for a given exchange
func GetExchangeTickers(exchange string) ([]*Price, error) {
	return service.getExchangeTickers(exchange)
}

func (s *Service) getExchangeTickers(exchange string) ([]*Price, error) {
	if exchange == "" {
		return nil, common.ErrExchangeNameNotSet
	}
	exchange = strings.ToLower(exchange)
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.Exchange[exchange]
	if !ok {
		return nil, fmt.Errorf("%w %v", errExchangeNotFound, exchange)
	}
	tickers := make([]*Price, 0, len(s.Tickers))
	for k, v := range s.Tickers {
		if k.Exchange != exchange {
			continue
		}
		cpy := v.Price // Don't let external functions have access to underlying
		tickers = append(tickers, &cpy)
	}
	return tickers, nil
}

// FindLast searches for a currency pair and returns the first available
func FindLast(p currency.Pair, a asset.Item) (float64, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	for mapKey, t := range service.Tickers {
		if !mapKey.MatchesPairAsset(p, a) {
			continue
		}
		if t.Last == 0 {
			return 0, errInvalidTicker
		}
		return t.Last, nil
	}
	return 0, fmt.Errorf("%w %s %s", ErrTickerNotFound, p, a)
}

// ProcessTicker processes incoming tickers, creating or updating the Tickers list
func ProcessTicker(p *Price) error {
	if p == nil {
		return errors.New(errTickerPriceIsNil)
	}

	if p.ExchangeName == "" {
		return common.ErrExchangeNameNotSet
	}

	if p.Pair.IsEmpty() {
		return fmt.Errorf("%s %s", p.ExchangeName, errPairNotSet)
	}

	if p.Bid != 0 && p.Ask != 0 {
		switch {
		case p.ExchangeName == "Bitfinex" && p.AssetType == asset.MarginFunding:
		// Margin funding books can be crossed see Bitfinex.
		default:
			if p.Bid == p.Ask {
				return fmt.Errorf("%s %s %w",
					p.ExchangeName,
					p.Pair,
					ErrBidEqualsAsk)
			}

			if p.Bid > p.Ask {
				return fmt.Errorf("%s %s %w",
					p.ExchangeName,
					p.Pair,
					errBidGreaterThanAsk)
			}
		}
	}

	if p.AssetType == asset.Empty {
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
	mapKey := key.NewExchangeAssetPair(name, p.AssetType, p.Pair)
	s.mu.Lock()
	t, ok := service.Tickers[mapKey]
	if !ok || t == nil {
		newTicker := &Ticker{}
		err := s.setItemID(newTicker, p, name)
		if err != nil {
			s.mu.Unlock()
			return err
		}
		service.Tickers[mapKey] = newTicker
		s.mu.Unlock()
		return nil
	}

	t.Price = *p
	//nolint: gocritic
	ids := append(t.Assoc, t.Main)
	s.mu.Unlock()
	return s.mux.Publish(p, ids...)
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

// getAssociations links a singular book with its dispatch associations
func (s *Service) getAssociations(exch string) ([]uuid.UUID, error) {
	if exch == "" {
		return nil, common.ErrExchangeNameNotSet
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
