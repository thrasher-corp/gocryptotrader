package base

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
)

// Strategy is base implementation of the Handler interface
type Strategy struct {
	useSimultaneousProcessing bool
	usingExchangeLevelFunding bool
}

// GetBaseData returns the non-interface version of the Handler
func (s *Strategy) GetBaseData(d data.Handler) (signal.Signal, error) {
	if d == nil {
		return signal.Signal{}, common.ErrNilArguments
	}
	latest := d.Latest()
	if latest == nil {
		return signal.Signal{}, common.ErrNilEvent
	}
	return signal.Signal{
		Base:       latest.GetBase(),
		ClosePrice: latest.GetClosePrice(),
		HighPrice:  latest.GetHighPrice(),
		OpenPrice:  latest.GetOpenPrice(),
		LowPrice:   latest.GetLowPrice(),
	}, nil
}

// UsingSimultaneousProcessing returns whether multiple currencies can be assessed in one go
func (s *Strategy) UsingSimultaneousProcessing() bool {
	return s.useSimultaneousProcessing
}

// SetSimultaneousProcessing sets whether multiple currencies can be assessed in one go
func (s *Strategy) SetSimultaneousProcessing(b bool) {
	s.useSimultaneousProcessing = b
}

// UsingExchangeLevelFunding returns whether funding is based on currency pairs or individual currencies at the exchange level
func (s *Strategy) UsingExchangeLevelFunding() bool {
	return s.usingExchangeLevelFunding
}

// SetExchangeLevelFunding sets whether funding is based on currency pairs or individual currencies at the exchange level
func (s *Strategy) SetExchangeLevelFunding(b bool) {
	s.usingExchangeLevelFunding = b
}
