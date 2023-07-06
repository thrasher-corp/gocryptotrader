package dispatch

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var (
	// ErrNotRunning defines an error when the dispatcher is not running
	ErrNotRunning = errors.New("dispatcher not running")

	errDispatcherNotInitialized          = errors.New("dispatcher not initialised")
	errDispatcherAlreadyRunning          = errors.New("dispatcher already running")
	errDispatchShutdown                  = errors.New("dispatcher did not shutdown properly, routines failed to close")
	errDispatcherUUIDNotFoundInRouteList = errors.New("dispatcher uuid not found in route list")
	errTypeAssertionFailure              = errors.New("type assertion failure")
	errChannelNotFoundInUUIDRef          = errors.New("dispatcher channel not found in uuid reference slice")
	errUUIDCollision                     = errors.New("dispatcher collision detected, uuid already exists")
	errDispatcherJobsAtLimit             = errors.New("dispatcher jobs at limit")
	errChannelIsNil                      = errors.New("channel is nil")
	errUUIDGeneratorFunctionIsNil        = errors.New("UUID generator function is nil")

	limitMessage = "%w [%d] current worker count [%d]. Spawn more workers via --dispatchworkers=x, or increase the jobs limit via --dispatchjobslimit=x"
)

// Name is an exported subsystem name
const Name = "dispatch"

func init() {
	dispatcher = NewDispatcher()
}

// NewDispatcher creates a new Dispatcher for relaying data.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		routes: make(map[uuid.UUID][]chan interface{}),
		outbound: sync.Pool{
			New: getChan,
		},
	}
}

func getChan() interface{} {
	// Create unbuffered channel for data pass
	return make(chan interface{})
}

// Start starts the dispatch system by spawning workers and allocating memory
func Start(workers, jobsLimit int) error {
	return dispatcher.start(workers, jobsLimit)
}

// Stop attempts to stop the dispatch service, this will close all pipe channels
// flush job list and drop all workers
func Stop() error {
	log.Debugln(log.DispatchMgr, "Dispatch manager shutting down...")
	return dispatcher.stop()
}

// IsRunning checks to see if the dispatch service is running
func IsRunning() bool {
	return dispatcher.isRunning()
}

// start compares atomic running value, sets defaults, overrides with
// configuration, then spawns workers
func (d *Dispatcher) start(workers, channelCapacity int) error {
	if d == nil {
		return errDispatcherNotInitialized
	}

	d.m.Lock()
	defer d.m.Unlock()

	if d.running {
		return errDispatcherAlreadyRunning
	}

	d.running = true

	if workers < 1 {
		log.Warnf(log.DispatchMgr,
			"workers cannot be zero, using default value %d\n",
			DefaultMaxWorkers)
		workers = DefaultMaxWorkers
	}
	if channelCapacity < 1 {
		log.Warnf(log.DispatchMgr,
			"jobs limit cannot be zero, using default values %d\n",
			DefaultJobsLimit)
		channelCapacity = DefaultJobsLimit
	}
	d.jobs = make(chan job, channelCapacity)
	d.maxWorkers = workers
	d.shutdown = make(chan struct{})

	for i := 0; i < d.maxWorkers; i++ {
		d.wg.Add(1)
		go d.relayer()
	}
	return nil
}

// stop stops the service and shuts down all worker routines
func (d *Dispatcher) stop() error {
	if d == nil {
		return errDispatcherNotInitialized
	}

	d.m.Lock()
	defer d.m.Unlock()

	if !d.running {
		return ErrNotRunning
	}

	d.running = false

	// Stop all jobs
	close(d.jobs)

	// Release finished workers
	close(d.shutdown)

	d.rMtx.Lock()
	for key, pipes := range d.routes {
		for i := range pipes {
			// Boot off receivers waiting on pipes.
			close(pipes[i])
		}
		// Flush all pipes, re-subscription will need to occur.
		d.routes[key] = nil
	}
	d.rMtx.Unlock()

	ch := make(chan struct{})
	timer := time.NewTimer(time.Second)
	go func(ch chan<- struct{}) { d.wg.Wait(); ch <- struct{}{} }(ch)
	select {
	case <-ch:
		log.Debugln(log.DispatchMgr, "Dispatch manager shutdown.")
		return nil
	case <-timer.C:
		return errDispatchShutdown
	}
}

// isRunning returns if the dispatch system is running
func (d *Dispatcher) isRunning() bool {
	if d == nil {
		return false
	}

	d.m.RLock()
	defer d.m.RUnlock()
	return d.running
}

// relayer routine relays communications across the defined routes
func (d *Dispatcher) relayer() {
	for {
		select {
		case j := <-d.jobs:
			if j.ID.IsNil() {
				// empty jobs from `channelCapacity` length are sent upon shutdown
				// every real job created has an ID set
				continue
			}
			d.rMtx.RLock()
			pipes, ok := d.routes[j.ID]
			if !ok {
				log.Warnf(log.DispatchMgr, "%v: %v\n", errDispatcherUUIDNotFoundInRouteList, j.ID)
				d.rMtx.RUnlock()
				continue
			}
			for i := range pipes {
				select {
				case pipes[i] <- j.Data:
				default:
					// no receiver; don't wait. This limits complexity.
				}
			}
			d.rMtx.RUnlock()
		case <-d.shutdown:
			d.wg.Done()
			return
		}
	}
}

// publish relays data to the subscribed subsystems
func (d *Dispatcher) publish(id uuid.UUID, data interface{}) error {
	if d == nil {
		return errDispatcherNotInitialized
	}

	if id.IsNil() {
		return errIDNotSet
	}

	if data == nil {
		return errNoData
	}

	d.m.RLock()
	defer d.m.RUnlock()

	if !d.running {
		return nil
	}

	select {
	case d.jobs <- job{data, id}: // Push job into job channel.
		return nil
	default:
		return fmt.Errorf(limitMessage,
			errDispatcherJobsAtLimit,
			len(d.jobs),
			d.maxWorkers)
	}
}

// Subscribe subscribes a system and returns a communication chan, this does not
// ensure initial push.
func (d *Dispatcher) subscribe(id uuid.UUID) (chan interface{}, error) {
	if d == nil {
		return nil, errDispatcherNotInitialized
	}

	if id.IsNil() {
		return nil, errIDNotSet
	}

	d.m.RLock()
	defer d.m.RUnlock()

	if !d.running {
		return nil, ErrNotRunning
	}

	d.rMtx.Lock()
	defer d.rMtx.Unlock()
	if _, ok := d.routes[id]; !ok {
		return nil, errDispatcherUUIDNotFoundInRouteList
	}

	// Get an unused channel from the channel pool
	ch, ok := d.outbound.Get().(chan interface{})
	if !ok {
		return nil, errTypeAssertionFailure
	}

	d.routes[id] = append(d.routes[id], ch)
	atomic.AddInt32(&d.subscriberCount, 1)
	return ch, nil
}

// Unsubscribe unsubs a routine from the dispatcher
func (d *Dispatcher) unsubscribe(id uuid.UUID, usedChan chan interface{}) error {
	if d == nil {
		return errDispatcherNotInitialized
	}

	if id.IsNil() {
		return errIDNotSet
	}

	if usedChan == nil {
		return errChannelIsNil
	}

	d.m.RLock()
	defer d.m.RUnlock()

	if !d.running {
		// reference will already be released in the stop function
		return nil
	}

	d.rMtx.Lock()
	defer d.rMtx.Unlock()
	pipes, ok := d.routes[id]
	if !ok {
		return errDispatcherUUIDNotFoundInRouteList
	}

	for i := range pipes {
		if pipes[i] != usedChan {
			continue
		}
		// Delete individual reference
		pipes[i] = pipes[len(pipes)-1]
		pipes[len(pipes)-1] = nil
		d.routes[id] = pipes[:len(pipes)-1]
		atomic.AddInt32(&d.subscriberCount, -1)

		// Drain and put the used chan back in pool; only if it is not closed.
		select {
		case _, ok = <-usedChan:
		default:
		}

		if ok {
			d.outbound.Put(usedChan)
		}
		return nil
	}
	return errChannelNotFoundInUUIDRef
}

// GetNewID returns a new ID
func (d *Dispatcher) getNewID(genFn func() (uuid.UUID, error)) (uuid.UUID, error) {
	if d == nil {
		return uuid.Nil, errDispatcherNotInitialized
	}

	if genFn == nil {
		return uuid.Nil, errUUIDGeneratorFunctionIsNil
	}

	// Continue to allow the generation, input and return of UUIDs even if
	// service is not currently enabled.

	d.m.RLock()
	defer d.m.RUnlock()

	// Generate new uuid
	newID, err := genFn()
	if err != nil {
		return uuid.Nil, err
	}

	d.rMtx.Lock()
	defer d.rMtx.Unlock()
	// Check to see if it already exists
	if _, ok := d.routes[newID]; ok {
		return uuid.Nil, errUUIDCollision
	}
	// Write the key into system
	d.routes[newID] = nil
	return newID, nil
}
