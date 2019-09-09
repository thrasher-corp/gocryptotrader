package dispatch

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofrs/uuid"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

const (
	// defaultGatewaySleep defines our sleep time in between worker generation
	// to limit worker production
	defaultGatewaySleep = time.Millisecond * 100

	// defaultJobBuffer defines a maxiumum amount of jobs allowed in channel
	// before we spill over and spawn more workers
	defaultJobBuffer = 10

	// defaultWorkerRate defines a rate at which we determine the efficient
	// release of workers by dividing how many jobs are in the job queue/channel
	// with this number
	defaultWorkerRate = 4

	// DefaultMaxWorkers is the package default worker ceiling amount
	DefaultMaxWorkers = 10
)

// MaxWorkers define a ceiling for the amount of workers spawned
var MaxWorkers int64

// comms is our main instance
var comms *Communications

// init initial startup (✿◠‿◠)
func init() {
	comms = &Communications{
		Routing: make(map[uuid.UUID][]chan interface{}),
		jobs:    make(chan *job, defaultJobBuffer),
		outbound: sync.Pool{
			New: func() interface{} {
				// Create unbuffered channel for data pass
				return make(chan interface{})
			},
		},
		inbound: sync.Pool{
			New: func() interface{} {
				// Create buffered channel for error return, buffered to free up
				// worker
				return make(chan interface{}, 1)
			},
		},
	}

	MaxWorkers = DefaultMaxWorkers
	// TODO: Might drop this worker in the future and just allocate and
	// de-allocate as workers are needed
	go comms.relayer()
}

// job defines a relaying job associated with a ticket which allows routing to
// routines that require specific data
type job struct {
	Data interface{}
	ID   uuid.UUID
	Err  chan interface{}
}

// Communications defines inner-subsystem communication systems
type Communications struct {
	// Outbound subystem with a ticket association so this could be routes will
	// be cleaned when sub-systems have no use with them
	// TODO: limit slice doubling
	Routing map[uuid.UUID][]chan interface{}
	rwMtx   sync.RWMutex

	// Persistent job channel see job struct
	jobs chan *job

	// Dynamic channel communication pools; unbuffered outbound channels for
	// generic data and buffered inbound channels for general errors
	outbound sync.Pool
	inbound  sync.Pool

	// Atomic worker count
	count int64
	// Atomic worker gateway
	gateway uint32
}

// Relayer routine relays communications across the defined routes
func (c *Communications) relayer() {
	atomic.AddInt64(&c.count, 1)
	tick := time.NewTicker(defaultGatewaySleep)
	for {
		select {
		case j := <-c.jobs:
			c.rwMtx.RLock()
			if _, ok := c.Routing[j.ID]; !ok {
				c.rwMtx.RUnlock()
				j.Err <- fmt.Errorf("relay failure ID: %s not found in routes",
					j.ID)
				continue
			}
			for i := range c.Routing[j.ID] {
				c.Routing[j.ID][i] <- j.Data
			}
			c.rwMtx.RUnlock()
			j.Err <- nil

		case <-tick.C:
			if atomic.CompareAndSwapUint32(&c.gateway, 0, 1) {
				if len(c.jobs) < (defaultJobBuffer / defaultWorkerRate) {
					if atomic.LoadInt64(&c.count) > 1 {
						atomic.AddInt64(&c.count, -1)
						atomic.SwapUint32(&c.gateway, 0)
						return
					}
				}
			}
		}
	}
}

// publish relays data to the subscribed subsystems
func (c *Communications) publish(id uuid.UUID, data interface{}) error {
	if data == nil {
		return errors.New("data cannot be nil")
	}

	if id == (uuid.UUID{}) {
		return errors.New("id not set")
	}

	// Get a buffered error channel link
	err := c.inbound.Get().(chan interface{})

	// Create a new job to publish
	newJob := &job{
		// TODO: possibly change data to pointer from here, so we dont reference
		// our main copy in the subsystem which might race when we read and
		// write to it.
		Data: data,
		ID:   id,
		Err:  err,
	}

	// Push to job stack, this is buffered, when it reaches its buffered limit
	// it will overflow the job stack and spawn a worker up until MaxWorkers
	// (ノಠ益ಠ)
	select {
	case c.jobs <- newJob:
	default:
		if atomic.CompareAndSwapUint32(&c.gateway, 0, 1) {
			go func() {
				// Adds in an artificial time buffer between worker generation
				// so we limit the scale up
				time.Sleep(defaultGatewaySleep)
				atomic.SwapUint32(&c.gateway, 0)
			}()
			if atomic.LoadInt64(&c.count) < atomic.LoadInt64(&MaxWorkers) {
				go c.relayer()
			}
		}
		c.jobs <- newJob
	}

	newErr := <-err

	// Put error channel back in to pool when finished
	c.inbound.Put(err)

	if newErr != nil {
		return newErr.(error)
	}

	return nil
}

// Release releases all chans associated with ticket, will release channels to
// the pool
func (c *Communications) release(id uuid.UUID) error {
	c.rwMtx.Lock()
	channels, ok := c.Routing[id]
	if !ok {
		c.rwMtx.Unlock()
		return errors.New("ticket not found in routing map")
	}

	// Put excess channels back into the pool
	for i := range channels {
		c.outbound.Put(channels[i])
	}

	c.Routing[id] = nil // Release reference TODO: Actually check garbage
	// collection ¯\_(ツ)_/¯

	// Delete key and associations
	delete(c.Routing, id)
	c.rwMtx.Unlock()
	return nil
}

// Subscribe subscribes a system and returns a communication chan, this does not
// ensure initial push. If your routine is out of sync with heartbeat and the
// system does not get a change, its up to you to in turn get initial state.
func (c *Communications) subscribe(id uuid.UUID) (chan interface{}, error) {
	// Read lock to read route list
	c.rwMtx.RLock()
	_, ok := c.Routing[id]
	c.rwMtx.RUnlock()
	if !ok {
		newChan := make(chan interface{})
		close(newChan)
		return newChan, errors.New("id not found in route list")
	}

	// Get an unused channel from the channel pool
	unusedChan := c.outbound.Get().(chan interface{})

	// Lock for writing to the route list
	c.rwMtx.Lock()
	c.Routing[id] = append(c.Routing[id], unusedChan)
	c.rwMtx.Unlock()

	return unusedChan, nil
}

// Unsubscribe unsubs a routine from the dispatcher
func (c *Communications) unsubscribe(id uuid.UUID, usedChan chan interface{}) error {
	// Read lock to read route list
	c.rwMtx.RLock()
	_, ok := c.Routing[id]
	c.rwMtx.RUnlock()
	if !ok {
		return errors.New("ticket does not reference any channels")
	}

	// Lock for write to delete references
	c.rwMtx.Lock()
	for i := range c.Routing[id] {
		if c.Routing[id][i] != usedChan {
			continue
		}
		// Delete individual reference
		c.Routing[id][i] = c.Routing[id][len(c.Routing[id])-1]
		c.Routing[id][len(c.Routing[id])-1] = nil
		c.Routing[id] = c.Routing[id][:len(c.Routing[id])-1]

		// Put the used chan back in pool
		c.outbound.Put(usedChan)
		c.rwMtx.Unlock()
		return nil
	}
	c.rwMtx.Unlock()
	return errors.New("channel not found in uuid reference slice")
}

// GetNewID returns a new ID
func (c *Communications) getNewID() (uuid.UUID, error) {
	// Generate new uuid
	newID, err := uuid.NewV4()
	if err != nil {
		return uuid.UUID{}, err
	}

	// Check to see if it already exists
	c.rwMtx.RLock()
	_, ok := c.Routing[newID]
	c.rwMtx.RUnlock()
	if ok {
		return newID, errors.New("collision detected, uuid already exists")
	}

	// Write the key into system
	c.rwMtx.Lock()
	c.Routing[newID] = nil
	c.rwMtx.Unlock()

	return newID, nil
}

// SetMaxWorkers sets worker generation ceiling
func SetMaxWorkers(w int64) error {
	if w < 1 {
		return errors.New("cannot set dispatch workers less than one")
	}

	old := atomic.SwapInt64(&MaxWorkers, w)
	log.Debugf(log.Global, "dispatch worker ceiling updated from %d to %d max workers",
		old,
		w)
	return nil
}
