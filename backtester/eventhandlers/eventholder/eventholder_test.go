package eventholder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
)

func TestReset(t *testing.T) {
	t.Parallel()
	e := &Holder{Queue: []common.Event{}}
	err := e.Reset()
	assert.NoError(t, err)

	if e.Queue != nil {
		t.Error("expected nil")
	}

	e = nil
	err = e.Reset()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestAppendEvent(t *testing.T) {
	t.Parallel()
	e := Holder{Queue: []common.Event{}}
	e.AppendEvent(&order.Order{})
	if len(e.Queue) != 1 {
		t.Error("expected 1")
	}
}

func TestNextEvent(t *testing.T) {
	t.Parallel()
	e := Holder{Queue: []common.Event{}}
	if ev := e.NextEvent(); ev != nil {
		t.Error("expected not ok")
	}

	e = Holder{Queue: []common.Event{
		&order.Order{},
		&order.Order{},
		&order.Order{},
	}}
	if len(e.Queue) != 3 {
		t.Error("expected 3")
	}
	e.NextEvent()
	if len(e.Queue) != 2 {
		t.Error("expected 2")
	}
}
