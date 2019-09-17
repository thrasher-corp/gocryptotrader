package dispatch

import (
	"errors"
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

	// handshakeTimeout defines a workers max length of time to wait on a
	// channel before moving on to next channel route
	handshakeTimeout = 200 * time.Nanosecond
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
	// generic data
	outbound sync.Pool

	// Atomic worker count
	count int64
	// Atomic worker gateway
	gateway uint32
}

// Relayer routine relays communications across the defined routes
func (c *Communications) relayer() {
	atomic.AddInt64(&c.count, 1)
	tick := time.NewTicker(defaultGatewaySleep)
	chanHSTimeout := time.NewTimer(0)
	for {
		select {
		case j := <-c.jobs:
			c.rwMtx.RLock()
			if _, ok := c.Routing[j.ID]; !ok {
				c.rwMtx.RUnlock()
				continue
			}
			// Channel handshake timeout feature if a channel is blocked for any
			// period of time due to an issue with the receiving routine.
			// This will wait on channel then fall over to the next route when
			// the timer actuates and continue over the route list. Have to
			// iterate across full length of routes so every routine can get
			// their new info, cannot be buffered as we dont want to have an old
			// orderbook etc contained in a buffered channel when a routine
			// actually is ready for a receive.
			// TODO: Need to consider optimal timer length
			for i := range c.Routing[j.ID] {
				if !chanHSTimeout.Stop() { // Stop timer before reset
					// Drain channel if timer has already actuated
					select {
					case <-chanHSTimeout.C:
					default:
					}
				}

				chanHSTimeout.Reset(handshakeTimeout)
				select {
				case c.Routing[j.ID][i] <- j.Data:
				case <-chanHSTimeout.C:
				}
			}
			c.rwMtx.RUnlock()

		case <-tick.C:
			if atomic.CompareAndSwapUint32(&c.gateway, 0, 1) {
				if len(c.jobs) < (defaultJobBuffer / defaultWorkerRate) {
					if atomic.LoadInt64(&c.count) > 1 {
						atomic.AddInt64(&c.count, -1)
						atomic.SwapUint32(&c.gateway, 0)

						tick.Stop()
						if !chanHSTimeout.Stop() {
							select {
							case <-chanHSTimeout.C:
							default:
							}
						}
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

	// Create a new job to publish
	newJob := &job{
		Data: data,
		ID:   id,
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
		select {
		case c.jobs <- newJob:
		default:
			return errors.New("buffer at max cap, spawn more workers")
		}
	}

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
		return nil, errors.New("id not found in route list")
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
func SetMaxWorkers(w int64) {
	if w < 1 {
		log.Warnf(log.Global,
			"dispatch package: invalid worker amount, defaulting to %d",
			DefaultMaxWorkers)
		w = DefaultMaxWorkers
	}

	old := atomic.SwapInt64(&MaxWorkers, w)
	log.Debugf(log.Global, "dispatch worker ceiling updated from %d to %d max workers",
		old,
		w)
}
