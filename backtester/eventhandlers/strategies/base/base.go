package base

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
)

// Strategy is base implementation of the Handler interface
type Strategy struct {
	multiCurrency bool
}

// GetBase returns the non-interface version of the Handler
func (s *Strategy) GetBase(d data.Handler) (signal.Signal, error) {
	if d == nil {
		return signal.Signal{}, errors.New("nil data handler received")
	}
	latest := d.Latest()
	if latest == nil {
		return signal.Signal{}, errors.New("could not retrieve latest data for strategy")
	}
	return signal.Signal{
		Base: event.Base{
			Exchange:     latest.GetExchange(),
			Time:         latest.GetTime(),
			CurrencyPair: latest.Pair(),
			AssetType:    latest.GetAssetType(),
			Interval:     latest.GetInterval(),
		},
	}, nil
}

// IsMultiCurrency returns whether multiple currencies can be assessed in one go
func (s *Strategy) IsMultiCurrency() bool {
	return s.multiCurrency
}

// SetMultiCurrency sets whether multiple currencies can be assessed in one go
func (s *Strategy) SetMultiCurrency(b bool) {
	s.multiCurrency = b
}
