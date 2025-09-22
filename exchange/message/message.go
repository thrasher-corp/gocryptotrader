package message

import (
	"context"
	"errors"
	"fmt"
)

var errChannelBufferFull = errors.New("channel buffer is full")

// Relay defines a channel relay for messages
type Relay struct {
	comm chan Payload
}

// Payload represents a relayed message with a context
type Payload struct {
	Ctx  context.Context //nolint:containedctx // context needed for tracing/metrics
	Data any
}

// NewRelay creates a new Relay instance with a specified buffer size
func NewRelay(buffer uint) *Relay {
	if buffer == 0 {
		panic("buffer size must be greater than 0")
	}
	return &Relay{comm: make(chan Payload, buffer)}
}

// Send sends a message to the channel receiver
// This is non-blocking and returns an error if the channel buffer is full
func (r *Relay) Send(ctx context.Context, data any) error {
	select {
	case r.comm <- Payload{Ctx: ctx, Data: data}:
		return nil
	default:
		return fmt.Errorf("%w: failed to relay <%T>", errChannelBufferFull, data)
	}
}

// Read returns the channel to receive messages
func (r *Relay) Read() <-chan Payload {
	return r.comm
}

// Close closes the relay channel
func (r *Relay) Close() {
	close(r.comm)
}
