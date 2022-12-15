package common

import (
	"fmt"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// NewScheduler returns a standard schedular to generate signals at a defined
// heartbeat within a scheduled start and end time. This can be set to align
// to the interval candle open or it will operate at set heartbeat intervals
// without truncation.
func NewScheduler(start, end time.Time, aligned bool, heartbeat kline.Interval) (*Scheduler, error) {
	if heartbeat < kline.OneMin {
		return nil, ErrIntervalNotSupported
	}
	// If there is an actual set start and end time for this strategy.
	if !end.IsZero() && start.After(time.Now()) {
		window := end.Sub(start)
		if (float64(window) / float64(heartbeat)) < 1 {
			return nil, fmt.Errorf("due to time window %s and heart beat size %s %w",
				window, heartbeat, ErrCannotGenerateSignal)
		}
	}
	schedule := &Scheduler{
		start:          start,
		end:            end,
		next:           start,
		alignmentToUTC: aligned,
		interval:       heartbeat,
	}
	// Pre-set schedule for starting and ceasing operations
	schedule.setTimer()
	schedule.setEndTimer()
	return schedule, nil
}

// Scheduler provides scheduling assistance for strategies. This acts as a base
// for potential strategies, and all methods can be overridden when embedded.
// For example, the `GetSignal()` method returns a `time.Time`, and the
// `OnSignal` method can be used to handle this signal in a strategy-defined
// way. This can be used to create market making bots and standard technical
// analysis wrappers.
type Scheduler struct {
	// start is the scheduled start time for the strategy
	start time.Time
	// end is the scheduled end time for the strategy
	end time.Time
	// next is the next time the signal will be fired
	next time.Time
	// alignmentToUTC determines whether the strategy's heartbeat should occur
	// at the actual candle close.
	alignmentToUTC bool
	// interval is the lowest interval at which the strategy's heartbeat
	// should execute.
	interval kline.Interval
	// timer is the next firing sequence for the wake up signal
	timer *time.Timer
	// ender is the end of life for the strategy
	ender *time.Timer
	// pipe is a common channel used to implement the `GetSignal` method.
	// There is no way to typecast a channel in Go, so a routine is required.
	pipe chan interface{}
	mtx  sync.Mutex
}

// GetSignal returns a channel to an unspecified signal generator which will
// be utilised in the `deploy()` method as defined in requirement.go.
func (s *Scheduler) GetSignal() <-chan interface{} {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	return s.pipeTimer()
}

// GetEnd returns the scheduled end time for the strategy. This indicates
// when the strategy will cease operations.
func (s *Scheduler) GetEnd(suppress bool) <-chan time.Time {
	if suppress {
		return nil
	}
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if s.ender == nil {
		return nil
	}
	return s.ender.C
}

// pipeTimer converts a channel to a `<-chan interface{}` for use with a
// future strategy signals interface. This method should not be called in
// multiple routines.
// TODO: This will cause a leak due to input.C not being closed. A closer
// will need to be run to avoid this.
func (s *Scheduler) pipeTimer() <-chan interface{} {
	if s.pipe == nil {
		s.pipe = make(chan interface{})
		go s.piper()
	}
	return s.pipe
}

// piper routine takes the timer signal and sends it to the `OnSignal` channel
// it then resets the timer to the appropriate next heartbeat.
func (s *Scheduler) piper() {
	for signal := range s.timer.C {
		s.pipe <- signal
		s.setTimer()
	}
}

// setTimer automatically resets timer to next heartbeat interval
func (s *Scheduler) setTimer() {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	duration := s.interval.Duration()
	if s.timer == nil {
		if s.start.IsZero() {
			s.start = time.Now()
		}
		if s.alignmentToUTC {
			// This adds duration after trunc so that a strategy execute trading
			// before start time.
			s.next = s.start.Truncate(duration).Add(duration)
		} else {
			// Don't need monotonic clock for now.
			s.next = s.start.Round(0)
		}
		s.timer = time.NewTimer(time.Until(s.next))
		return
	}
	// Push forward next time
	s.next = s.next.Add(duration)
	s.timer.Reset(time.Until(s.next))
}

// setEndTimer sets when the strategy will cease operations
func (s *Scheduler) setEndTimer() {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if s.end.IsZero() {
		return
	}
	fireAt := time.Until(s.end)
	s.ender = time.NewTimer(fireAt)
}

// GetNext will return when the strategy will generate a new signal at a set
// time.
func (s *Scheduler) GetNext() time.Time {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	return s.next
}

// Schedule defines schedule operating details for the strategy
type Schedule struct {
	Start      time.Time
	End        time.Time
	Next       time.Time
	UntilStart time.Duration
	SinceStart time.Duration
	Window     time.Duration
}

// GetSchedule returns the actual schedule details
func (s *Scheduler) GetSchedule() Schedule {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	return Schedule{
		Start:      s.start,
		End:        s.end,
		Next:       s.next,
		UntilStart: time.Until(s.start),
		SinceStart: time.Since(s.start),
		Window:     s.end.Sub(s.start),
	}
}
