package base

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/signal"
	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
)

type Strategy struct{}

func (s *Strategy) GetBase(d portfolio.DataHandler) signal.Signal {
	return signal.Signal{
		Event: event.Event{
			Exchange:     d.Latest().GetExchange(),
			Time:         d.Latest().GetTime(),
			CurrencyPair: d.Latest().Pair(),
			AssetType:    d.Latest().GetAssetType(),
		},
	}
}
