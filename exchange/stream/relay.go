package stream

import (
	"context"
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
)

var errChannelBufferFull = errors.New("channel buffer is full")

// Relay defines a channel relay for messages
type Relay struct {
	C    <-chan Payload
	comm chan Payload
}

// Payload represents a relayed message with a context
type Payload struct {
	Ctx  common.FrozenContext
	Data any
}

// NewRelay creates a new Relay instance with a specified buffer size
func NewRelay(buffer uint) *Relay {
	if buffer == 0 {
		panic("buffer size must be greater than 0")
	}
	comm := make(chan Payload, buffer)
	return &Relay{comm: comm, C: comm}
}

// Send sends a message to the channel receiver
// This is non-blocking and returns an error if the channel buffer is full
func (r *Relay) Send(ctx context.Context, data any) error {
	select {
	case r.comm <- Payload{Ctx: common.FreezeContext(ctx), Data: data}:
		return nil
	default:
		return fmt.Errorf("%w: failed to relay <%T>", errChannelBufferFull, data)
	}
}

// Close closes the relay channel
func (r *Relay) Close() {
	close(r.comm)
}
