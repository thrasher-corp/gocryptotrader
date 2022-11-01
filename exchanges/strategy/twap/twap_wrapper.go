package twap

import (
	"context"
	"time"

	strategy "github.com/thrasher-corp/gocryptotrader/exchanges/strategy/common"
)

const twapTag = "TWAP"

// Run inititates a TWAP allocation using the specified paramaters.
func (s *Strategy) Run(ctx context.Context) error {
	if s == nil {
		return strategy.ErrIsNil
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s.running {
		return strategy.ErrAlreadyRunning
	}

	if s.Config == nil {
		return strategy.ErrConfigIsNil
	}

	var start time.Duration
	if s.CandleStickAligned {
		// If aligned this will need to be truncated
		var err error
		start, err = s.GetNextSchedule(s.Start)
		if err != nil {
			return err
		}
	}
	s.wg.Add(1)
	go s.deploy(ctx, start)
	s.running = true
	return nil
}

// Stop stops the twap strategy.
func (s *Strategy) Stop() error {
	if s == nil {
		return strategy.ErrIsNil
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	if !s.running {
		return strategy.ErrNotRunning
	}

	close(s.shutdown)
	s.wg.Wait()
	s.running = false
	return nil
}

// IsRunning checks to see if the strategy is running
func (s *Strategy) GetReporter() (strategy.Reporter, error) {
	if s == nil {
		return nil, strategy.ErrIsNil
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()
	if s.Reporter == nil {
		s.Reporter = make(strategy.Reporter, 1)
	}
	return s.Reporter, nil
}

// GetState returns the state of the strategy
func (s *Strategy) GetState() (*strategy.State, error) {
	if s == nil {
		return nil, strategy.ErrIsNil
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	return &strategy.State{
		Exchange: s.Exchange.GetName(),
		Pair:     s.Pair,
		Asset:    s.Asset,
		Strategy: twapTag,
		Running:  s.running,
	}, nil
}
