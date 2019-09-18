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
	// DefaultJobBuffer defines a maxiumum amount of jobs allowed in channel
	DefaultJobBuffer = 100

	// DefaultMaxWorkers is the package default worker ceiling amount
	DefaultMaxWorkers = 10

	// DefaultHandshakeTimeout defines a workers max length of time to wait on a
	// an unbuffered channel for a receiver before moving on to next route
	DefaultHandshakeTimeout = 200 * time.Nanosecond

	errNotInitialised   = "dispatcher not initialised"
	errAlreadyStarted   = "dispatcher already started"
	errCannotShutdown   = "dispatcher cannot shutdown, already stopped"
	errShutdownRoutines = "dispatcher did not shutdown properly, routines failed to close"
)

func init() {
	dispatcher = &Dispatcher{
		routes: make(map[uuid.UUID][]chan interface{}),
		jobs:   make(chan *job, DefaultJobBuffer),
		outbound: sync.Pool{
			New: func() interface{} {
				// Create unbuffered channel for data pass
				return make(chan interface{})
			},
		},
	}
}

// dispatcher is our main in memory instance with a stop/start mtx below
var dispatcher *Dispatcher
var mtx sync.Mutex

// Start starts the dispatch system by spawning workers and allocating memory
func Start(workers int64) error {
	if dispatcher == nil {
		return errors.New(errNotInitialised)
	}

	mtx.Lock()
	defer mtx.Unlock()
	return dispatcher.start(workers)
}

// Stop attempts to stop the dispatch service, this will close all pipe channels
// flush job list and drop all workers
func Stop() error {
	if dispatcher == nil {
		return errors.New(errNotInitialised)
	}

	log.Debugln(log.DispatchMgr, "Dispatch manager shutting down...")

	mtx.Lock()
	defer mtx.Unlock()
	return dispatcher.stop()
}

// IsRunning checks to see if the dispatch service is running
func IsRunning() bool {
	if dispatcher == nil {
		return false
	}

	return dispatcher.isRunning()
}

// DropWorker drops a worker routine
func DropWorker() error {
	if dispatcher == nil {
		return errors.New(errNotInitialised)
	}

	return dispatcher.dropWorker()
}

// SpawnWorker starts a new worker routine
func SpawnWorker() error {
	if dispatcher == nil {
		return errors.New(errNotInitialised)
	}
	return dispatcher.spawnWorker()
}

// Dispatcher defines an internal subsystem communication/change state publisher
type Dispatcher struct {
	// routes refers to a subystem uuid ticket map with associated publish
	// channels, a relayer will be given a unique id through its job channel,
	// then publish the data across the full registered channels for that uuid.
	// See relayer() method below.
	routes map[uuid.UUID][]chan interface{}

	// rMtx protects the routes variable ensuring acceptable read/write access
	rMtx sync.RWMutex

	// Persistent buffered job queue for relayers
	jobs chan *job

	// Dynamic channel pool; returns an unbuffered channel for routes map
	outbound sync.Pool

	// Atomic values -----------------------
	// MaxWorkers defines max worker ceiling
	maxWorkers int64
	// Worker counter
	count int64
	// Dispatch status
	running uint32

	// Unbufferd shutdown chan, sync wg for ensuring concurrency when only
	// dropping a single relayer routine
	shutdown chan *sync.WaitGroup

	// Relayer shutdown tracking
	wg sync.WaitGroup
}

// start compares atomic running value, sets defaults, overides with
// configuration, then spawns workers
func (d *Dispatcher) start(workers int64) error {
	if atomic.LoadUint32(&d.running) == 1 {
		return errors.New(errAlreadyStarted)
	}

	if workers < 1 {
		log.Warn(log.DispatchMgr,
			"Dispatcher: workers cannot be zero using default values")
		workers = DefaultMaxWorkers
	}

	d.maxWorkers = workers
	d.shutdown = make(chan *sync.WaitGroup)

	if atomic.LoadInt64(&d.count) != 0 {
		return errors.New("dispatcher leaked workers found")
	}

	for i := int64(0); i < d.maxWorkers; i++ {
		err := d.spawnWorker()
		if err != nil {
			return err
		}
	}

	atomic.SwapUint32(&d.running, 1)
	return nil
}

// stop stops the service and shuts down all worker routines
func (d *Dispatcher) stop() error {
	if !atomic.CompareAndSwapUint32(&d.running, 1, 0) {
		return errors.New(errCannotShutdown)
	}
	close(d.shutdown)
	ch := make(chan struct{})
	timer := time.NewTimer(1 * time.Second)
	defer func() {
		timer.Stop()
		select {
		case <-timer.C:
		default:
		}
	}()
	go func(ch chan struct{}) { d.wg.Wait(); ch <- struct{}{} }(ch)
	select {
	case <-ch:
		// close all routes
		for key := range d.routes {
			for i := range d.routes[key] {
				close(d.routes[key][i])
			}

			d.routes[key] = nil
		}

		for len(d.jobs) != 0 { // drain jobs channel for old data
			<-d.jobs
		}

		log.Debugln(log.DispatchMgr, "Dispatch manager shutdown.")

		return nil
	case <-timer.C:
		return errors.New(errShutdownRoutines)
	}
}

// isRunning returns if the dispatch system is running
func (d *Dispatcher) isRunning() bool {
	return atomic.LoadUint32(&d.running) == 1
}

// dropWorker deallocates a worker routine
func (d *Dispatcher) dropWorker() error {
	oldC := atomic.LoadInt64(&d.count)
	wg := sync.WaitGroup{}
	wg.Add(1)
	d.shutdown <- &wg
	wg.Wait()
	newC := atomic.LoadInt64(&d.count)

	if oldC == newC {
		return errors.New("dispatcher worker counts are off")
	}
	return nil
}

// spawnWorker allocates a new worker for job processing
func (d *Dispatcher) spawnWorker() error {
	if atomic.LoadInt64(&d.count) >= d.maxWorkers {
		return errors.New("dispatcher cannot spawn more workers; ceiling reached")
	}
	var spawnWg sync.WaitGroup
	spawnWg.Add(1)
	go d.relayer(&spawnWg)
	spawnWg.Wait()
	return nil
}

// Relayer routine relays communications across the defined routes
func (d *Dispatcher) relayer(i *sync.WaitGroup) {
	atomic.AddInt64(&d.count, 1)
	d.wg.Add(1)
	timeout := time.NewTimer(0)
	i.Done()
	for {
		select {
		case j := <-d.jobs:
			d.rMtx.RLock()
			if _, ok := d.routes[j.ID]; !ok {
				d.rMtx.RUnlock()
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
			for i := range d.routes[j.ID] {
				if !timeout.Stop() { // Stop timer before reset
					// Drain channel if timer has already actuated
					select {
					case <-timeout.C:
					default:
					}
				}

				timeout.Reset(DefaultHandshakeTimeout)
				select {
				case d.routes[j.ID][i] <- j.Data:
				case <-timeout.C:
				}
			}
			d.rMtx.RUnlock()

		case v := <-d.shutdown:
			if !timeout.Stop() {
				select {
				case <-timeout.C:
				default:
				}
			}
			atomic.AddInt64(&d.count, -1)
			if v != nil {
				v.Done()
			}
			d.wg.Done()
			return
		}
	}
}

// publish relays data to the subscribed subsystems
func (d *Dispatcher) publish(id uuid.UUID, data interface{}) error {
	if data == nil {
		return errors.New("dispatcher data cannot be nil")
	}

	if id == (uuid.UUID{}) {
		return errors.New("dispatcher uuid not set")
	}

	if atomic.LoadUint32(&d.running) == 0 {
		return nil
	}

	// Create a new job to publish
	newJob := &job{
		Data: data,
		ID:   id,
	}

	// Push job on stack here
	select {
	case d.jobs <- newJob:
	default:
		return fmt.Errorf("dispatcher buffer at max capacity [%d] current worker count [%d], spawn more workers via --dispatchworkers=x",
			len(d.jobs),
			atomic.LoadInt64(&d.count))
	}

	return nil
}

// Subscribe subscribes a system and returns a communication chan, this does not
// ensure initial push. If your routine is out of sync with heartbeat and the
// system does not get a change, its up to you to in turn get initial state.
func (d *Dispatcher) subscribe(id uuid.UUID) (chan interface{}, error) {
	if atomic.LoadUint32(&d.running) == 0 {
		return nil, errors.New(errNotInitialised)
	}

	// Read lock to read route list
	d.rMtx.RLock()
	_, ok := d.routes[id]
	d.rMtx.RUnlock()
	if !ok {
		return nil, errors.New("dispatcher uuid not found in route list")
	}

	// Get an unused channel from the channel pool
	unusedChan := d.outbound.Get().(chan interface{})

	// Lock for writing to the route list
	d.rMtx.Lock()
	d.routes[id] = append(d.routes[id], unusedChan)
	d.rMtx.Unlock()

	return unusedChan, nil
}

// Unsubscribe unsubs a routine from the dispatcher
func (d *Dispatcher) unsubscribe(id uuid.UUID, usedChan chan interface{}) error {
	if atomic.LoadUint32(&d.running) == 0 {
		return errors.New(errNotInitialised)
	}

	// Read lock to read route list
	d.rMtx.RLock()
	_, ok := d.routes[id]
	d.rMtx.RUnlock()
	if !ok {
		return errors.New("dispatcher uuid does not reference any channels")
	}

	// Lock for write to delete references
	d.rMtx.Lock()
	for i := range d.routes[id] {
		if d.routes[id][i] != usedChan {
			continue
		}
		// Delete individual reference
		d.routes[id][i] = d.routes[id][len(d.routes[id])-1]
		d.routes[id][len(d.routes[id])-1] = nil
		d.routes[id] = d.routes[id][:len(d.routes[id])-1]

		d.rMtx.Unlock()

		// Drain and put the used chan back in pool; only if it is not closed.
		select {
		case _, ok := <-usedChan:
			if !ok {
				return nil
			}
		default:
		}

		d.outbound.Put(usedChan)
		return nil
	}
	d.rMtx.Unlock()
	return errors.New("dispatcher channel not found in uuid reference slice")
}

// GetNewID returns a new ID
func (d *Dispatcher) getNewID() (uuid.UUID, error) {
	// Generate new uuid
	newID, err := uuid.NewV4()
	if err != nil {
		return uuid.UUID{}, err
	}

	// Check to see if it already exists
	d.rMtx.RLock()
	_, ok := d.routes[newID]
	d.rMtx.RUnlock()
	if ok {
		return newID, errors.New("dispatcher collision detected, uuid already exists")
	}

	// Write the key into system
	d.rMtx.Lock()
	d.routes[newID] = nil
	d.rMtx.Unlock()

	return newID, nil
}

// job defines a relaying job associated with a ticket which allows routing to
// routines that require specific data
type job struct {
	Data interface{}
	ID   uuid.UUID
}
