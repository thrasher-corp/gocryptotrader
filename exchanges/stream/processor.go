package stream

import (
	"errors"
	"fmt"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

const DefaultChannelBufferSize = 10

var (
	errMaxChanBufferSizeInvalid = errors.New("max channel buffer must be greater than 0")
	errDataHandlerMustNotBeNil  = errors.New("data handler cannot be nil")
	errChannelFull              = errors.New("channel full")
	errKeyEmpty                 = errors.New("key is empty")
	errNoFunctionalityToProcess = errors.New("no functionality to process")
)

// Processor is a stream processor that handles incoming data from a stream,
// spawns workers for each unique key, and processes the data in a concurrent
// manner.
type Processor struct {
	routes         map[Key]chan func() error
	chanBufferSize int
	dataHandler    chan interface{}
	mtx            sync.RWMutex
}

// Key is a unique key for an individual type of work.
type Key struct {
	Asset asset.Item
	Base  *currency.Item
	Quote *currency.Item
}

// NewProcessor returns a new stream processor. This handles incoming data
// from a stream and processes it in a concurrent manner. It will spawn a
// new worker for each unique key. If a worker already exists for a key, it
// will use that worker. Functions are queued and processed in the order
// they are received. Function closures are used for ease of implementation
// with current systems.
func NewProcessor(maxChanBuffer int, dataHandler chan interface{}) (*Processor, error) {
	if maxChanBuffer <= 0 {
		return nil, errMaxChanBufferSizeInvalid
	}
	if dataHandler == nil {
		return nil, errDataHandlerMustNotBeNil
	}
	return &Processor{
		routes:         make(map[Key]chan func() error),
		chanBufferSize: maxChanBuffer,
		dataHandler:    dataHandler,
	}, nil
}

// Process spawns a new worker for a key if one does not already exist. It
// will then queue the function to be processed. If the channel is full, it
// will block until a slot is available. This tries to alleviate websocket
// reader blocking issues.
func (w *Processor) Process(key Key, fn func() error) error {
	if key == (Key{}) {
		return errKeyEmpty
	}
	if fn == nil {
		return fmt.Errorf("%w for %v", errNoFunctionalityToProcess, key)
	}

	w.mtx.RLock()
	ch, ok := w.routes[key]
	if !ok {
		wait := make(chan struct{})
		go func() {
			w.mtx.Lock()
			// re-check under main lock to ensure we don't spawn multiple
			// workers.
			if ch, ok = w.routes[key]; ok {
				close(wait)
				w.mtx.Unlock()
				return // already spawned
			}

			ch = make(chan func() error, w.chanBufferSize)
			w.routes[key] = ch
			close(wait)
			w.mtx.Unlock()
			for pending := range ch {
				err := pending()
				if err != nil {
					w.dataHandler <- err
				}
			}
		}()
		w.mtx.RUnlock()
		<-wait
	} else {
		w.mtx.RUnlock()
	}

	select {
	case ch <- fn:
		return nil
	default:
		ch <- fn
		return fmt.Errorf("%w for %v", errChannelFull, key)
	}
}
