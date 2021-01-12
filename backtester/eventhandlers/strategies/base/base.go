package base

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
)

type Strategy struct {
	multiCurrency bool
}

func (s *Strategy) GetBase(d data.Handler) (signal.Signal, error) {
	if d == nil {
		return signal.Signal{}, errors.New("nil data handler received")
	}
	latest := d.Latest()
	if latest == nil {
		return signal.Signal{}, errors.New("could not retrieve latest data for strategy")
	}
	return signal.Signal{
		Event: event.Event{
			Exchange:     latest.GetExchange(),
			Time:         latest.GetTime(),
			CurrencyPair: latest.Pair(),
			AssetType:    latest.GetAssetType(),
			Interval:     latest.GetInterval(),
		},
	}, nil
}

func (s *Strategy) IsMultiCurrency() bool {
	return s.multiCurrency
}

func (s *Strategy) SetMultiCurrency(b bool) {
	s.multiCurrency = b
}
