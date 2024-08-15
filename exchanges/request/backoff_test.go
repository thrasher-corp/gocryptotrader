package request_test

import (
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestLinearBackoff(t *testing.T) {
	t.Parallel()
	type args struct {
		Backoff request.Backoff
	}
	type want struct {
		Delays map[int]time.Duration
	}
	testTable := map[string]struct {
		Args args
		Want want
	}{
		"Default": {
			Args: args{Backoff: request.DefaultBackoff()},
			Want: want{Delays: map[int]time.Duration{
				1:  100 * time.Millisecond,
				2:  200 * time.Millisecond,
				3:  300 * time.Millisecond,
				4:  400 * time.Millisecond,
				9:  900 * time.Millisecond,
				10: time.Second,
				11: time.Second,
			}},
		},
		"Fixed": {
			Args: args{Backoff: request.LinearBackoff(100*time.Millisecond, 100*time.Millisecond)},
			Want: want{Delays: map[int]time.Duration{
				1: 100 * time.Millisecond,
				2: 100 * time.Millisecond,
				3: 100 * time.Millisecond,
			}},
		},
		"Quick Cap": {
			Args: args{Backoff: request.LinearBackoff(400*time.Millisecond, time.Second)},
			Want: want{Delays: map[int]time.Duration{
				1: 400 * time.Millisecond,
				2: 800 * time.Millisecond,
				3: time.Second,
				4: time.Second,
			}},
		},
		"Slow Cap": {
			Args: args{Backoff: request.LinearBackoff(50*time.Millisecond, time.Minute)},
			Want: want{Delays: map[int]time.Duration{
				1:    50 * time.Millisecond,
				2:    100 * time.Millisecond,
				3:    150 * time.Millisecond,
				19:   time.Second - 50*time.Millisecond,
				20:   time.Second,
				21:   time.Second + 50*time.Millisecond,
				1199: time.Minute - 50*time.Millisecond,
				1200: time.Minute,
				1201: time.Minute,
			}},
		},
	}

	for name, tt := range testTable {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			for n, exp := range tt.Want.Delays {
				got := tt.Args.Backoff(n)
				if got != exp {
					t.Errorf("incorrect backoff duration\nexp: %s\ngot: %s", exp, got)
				}
			}
		})
	}
}
