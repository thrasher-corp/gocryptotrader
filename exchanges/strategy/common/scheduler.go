package common

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// Scheduler defines scheduling assistance for strategies. NOTE: Acts as base
// for all potential strategies and all methods be overridable. e.g.
// GetSignal() chan interface{} core will return a time.Time and then signal
// can be strategy defined in methodOnSignal(ctx context.Context, sig interface{}).
// This can then build up to market making bots and standard TA wrappers.
type Scheduler struct {
	// start defines scheduled start time
	start time.Time
	// end defines scheduled end time
	end time.Time
	// alignmentToUTC allows the heartbeat of strategy to occur at actual
	// candle close
	alignmentToUTC bool
	// interval defines the actual lowest interval as heart beat to execute
	// strategy.
	interval kline.Interval
	// TODO: // offset allows for the shift in heartbeat
	// offset kline.Interval

	timer *time.Timer

	ender *time.Timer

	pipe chan interface{}
}

// GetSignal implements requirements interface
func (s *Scheduler) GetSignal() <-chan interface{} {
	return s.pipeTimer()
}

// pipeTimer converts channel to a <-chan interface{} for a future
// strategy signals interface.
// TODO: This might not be optimal for market making and orderbook
// change signaling. Should not be called in multiple routines.
// TODO: this will leak cause of input.C not closing gonna have to run a closer.
func (s *Scheduler) pipeTimer() <-chan interface{} {
	if s.pipe == nil {
		go func(input *time.Timer, output chan<- interface{}, interval kline.Interval) {
			for signal := range input.C {
				output <- signal
				input.Reset(interval.Duration()) // TODO: This will drift add truncation.
			}
			close(output)
		}(s.timer, s.pipe, s.interval)
	}
	return s.pipe
}

func (s *Scheduler) GetFinished() <-chan time.Time {
	return s.ender.C
}
