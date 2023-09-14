package stream

import (
	"errors"
	"fmt"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

const (
	// Book is a key type for websocket orderbook data processing
	Unset UpdateType = iota
	Book
	defaultChannelBufferSize = 10
)

var (
	errMaxChanBufferSizeInvalid   = errors.New("max channel buffer must be greater than 0")
	errDataHandlerMustNotBeNil    = errors.New("data handler cannot be nil")
	errChannelFull                = errors.New("channel full")
	errKeyEmpty                   = errors.New("key is empty")
	errNoFunctionalityToProcess   = errors.New("no functionality to process")
	errUpdateTypeUnset            = errors.New("update type unset")
	errdUpdateTypeNotYetSupported = errors.New("update type not yet supported")
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
	Type  UpdateType
	Asset asset.Item
	Base  *currency.Item
	Quote *currency.Item
}

// UpdateType is a unique key for an individual type of work.
type UpdateType uint8

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

// QueueFunction spawns a new worker for a key if one does not already exist. It
// will then queue the function to be processed. If the channel is full, it
// will block until a slot is available. This tries to alleviate websocket
// reader blocking issues.
func (w *Processor) QueueFunction(key Key, fn func() error) error {
	if key == (Key{}) {
		return errKeyEmpty
	}
	switch key.Type {
	case Unset:
		return fmt.Errorf("%w for %+v", errUpdateTypeUnset, key)
	case Book:
	default:
		return fmt.Errorf("%w for %+v", errdUpdateTypeNotYetSupported, key)
	}
	if fn == nil {
		return fmt.Errorf("%w for %+v", errNoFunctionalityToProcess, key)
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
			for processWebsocketTask := range ch {
				err := processWebsocketTask()
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
