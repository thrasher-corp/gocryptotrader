package base

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
)

// Strategy is base implementation of the Handler interface
type Strategy struct {
	useSimultaneousProcessing bool
}

// GetBaseData returns the non-interface version of the Handler
func (s *Strategy) GetBaseData(d data.Handler) (signal.Signal, error) {
	if d == nil {
		return signal.Signal{}, gctcommon.ErrNilPointer
	}
	latest, err := d.Latest()
	if err != nil {
		return signal.Signal{}, err
	}
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

// CloseAllPositions sends a closing signal to supported
// strategies, allowing them to sell off any positions held
// default use-case is for when a user closes the application when running
// a live strategy
func (s *Strategy) CloseAllPositions([]holdings.Holding, []data.Event) ([]signal.Event, error) {
	return nil, gctcommon.ErrFunctionNotSupported
}
