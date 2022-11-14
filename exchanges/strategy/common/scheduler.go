package common

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// NewScheduler returns a new schedular to assist in timing strategy heartbeat.
func NewScheduler(start, end time.Time, aligned bool, heartbeat kline.Interval) (*Scheduler, error) {
	schedule := &Scheduler{
		start:          start,
		end:            end,
		alignmentToUTC: aligned,
		interval:       heartbeat,
	}
	// NOTE: Pre-set timer and routine.
	schedule.setTimer()
	schedule.setEndTimer()
	return schedule, nil
}

// Scheduler defines scheduling assistance for strategies. NOTE: Acts as base
// for all potential strategies and all methods can be overridden. e.g.
// GetSignal() <-chan interface{} core will return a time.Time and then signal
// can be strategy defined in method OnSignal(ctx context.Context, sig interface{}).
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
	// timer defines the next firing sequence for the wake up signal
	timer *time.Timer
	// ender defines the end of life for the strategy
	ender *time.Timer
	// the pipe defines a common channel to return to implement the GetSignal
	// method, theres now way to type cast a channel in go and requires a
	// routine
	pipe chan interface{}
}

// GetSignal implements requirements interface
func (s *Scheduler) GetSignal() <-chan interface{} {
	return s.pipeTimer()
}

// GetEnd returns when the strategy will cease to operate at a certain defined
// time.
func (s *Scheduler) GetEnd() <-chan time.Time {
	if s.ender == nil {
		return nil
	}
	return s.ender.C
}

// pipeTimer converts channel to a <-chan interface{} for a future
// strategy signals interface.
// TODO: This might not be optimal for market making and orderbook
// change signaling. Should not be called in multiple routines.
// TODO: this will leak cause of input.C not closing gonna have to run a closer.
func (s *Scheduler) pipeTimer() <-chan interface{} {
	if s.pipe == nil {
		s.pipe = make(chan interface{})
		go s.piper()
	}
	return s.pipe
}

// piper routine takes the timer signal and sends it to the OnSignal channel it
// then resets the timer to the appropriate next heartbeat.
func (s *Scheduler) piper() {
	for signal := range s.timer.C {
		s.pipe <- signal
		s.setTimer()
	}
}

// setTimer automatically resets timer to next heartbeat interval
func (s *Scheduler) setTimer() {
	tn := time.Now()
	intDur := s.interval.Duration()
	if s.alignmentToUTC {
		tn = tn.Truncate(intDur)
	}
	fireAt := time.Until(tn.Add(intDur))
	if s.timer == nil {
		s.timer = time.NewTimer(fireAt)
		return
	}
	s.timer.Reset(fireAt)
}

// setEndTimer sets when the strategy will end
func (s *Scheduler) setEndTimer() {
	fireAt := time.Until(s.end)
	s.ender = time.NewTimer(fireAt)
}
