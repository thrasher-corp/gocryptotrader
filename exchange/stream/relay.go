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

// var Fr = trace.NewFlightRecorder(trace.FlightRecorderConfig{
// 	MinAge:   1 * time.Second,
// 	MaxBytes: 1 << 23, // 8 MiB
// })

// var once sync.Once

// // captureSnapshot captures a flight recorder snapshot.
// func captureSnapshot(fr *trace.FlightRecorder) {
// 	// once.Do ensures that the provided function is executed only once.
// 	once.Do(func() {
// 		f, err := os.Create("snapshot.trace")
// 		if err != nil {
// 			log.Printf("opening snapshot file %s failed: %s", f.Name(), err)
// 			return
// 		}
// 		defer f.Close() // ignore error

// 		// WriteTo writes the flight recorder data to the provided io.Writer.
// 		_, err = fr.WriteTo(f)
// 		if err != nil {
// 			log.Printf("writing snapshot to file %s failed: %s", f.Name(), err)
// 			return
// 		}

// 		// Stop the flight recorder after the snapshot has been taken.
// 		fr.Stop()
// 		log.Printf("captured a flight recorder snapshot to %s", f.Name())
// 		time.Sleep(500 * time.Millisecond) // allow time for the snapshot to be written before the program exits
// 		runtime.Breakpoint()
// 	})
// }

// Send sends a message to the channel receiver
// This is non-blocking and returns an error if the channel buffer is full
func (r *Relay) Send(ctx context.Context, data any) error {
	select {
	case r.comm <- Payload{Ctx: common.FreezeContext(ctx), Data: data}:
		return nil
	default:
		// if Fr.Enabled() {
		// 	captureSnapshot(Fr)
		// }
		return fmt.Errorf("%w: failed to relay <%T>", errChannelBufferFull, data)
	}
}

// Close closes the relay channel
func (r *Relay) Close() {
	close(r.comm)
}
