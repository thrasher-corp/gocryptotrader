package base

import (
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
)

type Strategy struct {
	multiCurrency bool
}

func (s *Strategy) GetBase(d data.Handler) signal.Signal {
	return signal.Signal{
		Event: event.Event{
			Exchange:     d.Latest().GetExchange(),
			Time:         d.Latest().GetTime(),
			CurrencyPair: d.Latest().Pair(),
			AssetType:    d.Latest().GetAssetType(),
			Interval:     d.Latest().GetInterval(),
		},
	}
}

func (s *Strategy) IsMultiCurrency() bool {
	return s.multiCurrency
}

func (s *Strategy) SetMultiCurrency(b bool) {
	s.multiCurrency = b
}

func (s *Strategy) HasDataAtPresentTime(d data.Handler) bool {
	es := s.GetBase(d)
	if !d.HasDataAtTime(d.Latest().GetTime()) {
		es.SetDirection(common.MissingData)
		es.AppendWhy(fmt.Sprintf("missing data at %v, cannot perform any actions", d.Latest().GetTime()))
		return false
	}
	return true
}
