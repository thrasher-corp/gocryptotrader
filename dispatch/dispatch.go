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

// ErrNotRunning defines an error when the dispatcher is not running
var ErrNotRunning = errors.New("dispatcher not running")

var (
	errDispatcherNotInitialized          = errors.New("dispatcher not initialised")
	errDispatcherAlreadyRunning          = errors.New("dispatcher already running")
	errLeakedWorkers                     = errors.New("dispatcher leaked workers found")
	errDispatchShutdown                  = errors.New("dispatcher did not shutdown properly, routines failed to close")
	errWorkerCeilingReached              = errors.New("dispatcher cannot spawn more workers; ceiling reached")
	errDispatcherUUIDNotFoundInRouteList = errors.New("dispatcher uuid not found in route list")
	errTypeAssertionFailure              = errors.New("type assertion failure")
	errChannelNotFoundInUUIDRef          = errors.New("dispatcher channel not found in uuid reference slice")
	errUUIDCollision                     = errors.New("dispatcher collision detected, uuid already exists")
	errNoWorkers                         = errors.New("no workers")
	errDispatcherJobsAtLimit             = errors.New("dispatcher jobs at limit")
	errChannelIsNil                      = errors.New("channel is nil")

	limitMessage = "%w [%d] current worker count [%d]. Spawn more workers via --dispatchworkers=x, or increase the jobs limit via --dispatchjobslimit=x"
)

// Name is an exported subsystem name
const Name = "dispatch"

func init() {
	dispatcher = newDispatcher()
}

func newDispatcher() *Dispatcher {
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

// DropWorker drops a worker routine
func DropWorker() error {
	return dispatcher.dropWorker()
}

// SpawnWorker starts a new worker routine
func SpawnWorker() error {
	return dispatcher.spawnWorker()
}

// start compares atomic running value, sets defaults, overides with
// configuration, then spawns workers
func (d *Dispatcher) start(workers, channelCapacity int) error {
	if d == nil {
		return errDispatcherNotInitialized
	}

	if !atomic.CompareAndSwapUint32(&d.running, 0, 1) {
		return errDispatcherAlreadyRunning
	}

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
	d.jobs = make(chan *job, channelCapacity) // TODO: pass by value
	d.maxWorkers = int32(workers)
	d.shutdown = make(chan *sync.WaitGroup)

	if atomic.LoadInt32(&d.count) != 0 {
		atomic.SwapUint32(&d.running, 0)
		return errLeakedWorkers
	}

	for i := int32(0); i < d.maxWorkers; i++ {
		err := d.spawnWorker()
		if err != nil {
			atomic.SwapUint32(&d.running, 0)
			return err
		}
	}
	return nil
}

// stop stops the service and shuts down all worker routines
func (d *Dispatcher) stop() error {
	if d == nil {
		return errDispatcherNotInitialized
	}

	if !atomic.CompareAndSwapUint32(&d.running, 1, 0) {
		return ErrNotRunning
	}
	close(d.shutdown)
	ch := make(chan struct{})
	timer := time.NewTimer(time.Second)
	defer func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}()
	go func(ch chan struct{}) { d.wg.Wait(); ch <- struct{}{} }(ch)
	select {
	case <-ch:
		// close all routes
		for key, pipes := range d.routes {
			for i := range pipes {
				close(pipes[i])
			}
			delete(d.routes, key)
		}

		for len(d.jobs) != 0 { // drain jobs channel for old data
			<-d.jobs
		}

		log.Debugln(log.DispatchMgr, "Dispatch manager shutdown.")
		return nil
	case <-timer.C:
		return errDispatchShutdown
	}
}

// isRunning returns if the dispatch system is running
func (d *Dispatcher) isRunning() bool {
	return d != nil && atomic.LoadUint32(&d.running) == 1
}

// dropWorker deallocates a worker routine
func (d *Dispatcher) dropWorker() error {
	if d == nil {
		return errDispatcherNotInitialized
	}
	if atomic.LoadUint32(&d.running) != 1 {
		return ErrNotRunning
	}
	if atomic.LoadInt32(&d.count) == 0 {
		return errNoWorkers
	}
	wg := sync.WaitGroup{}
	wg.Add(1)
	d.shutdown <- &wg
	wg.Wait()
	return nil
}

// spawnWorker allocates a new worker for job processing
func (d *Dispatcher) spawnWorker() error {
	if d == nil {
		return errDispatcherNotInitialized
	}
	if atomic.LoadUint32(&d.running) != 1 {
		return ErrNotRunning
	}
	if atomic.LoadInt32(&d.count) >= d.maxWorkers {
		return errWorkerCeilingReached
	}
	atomic.AddInt32(&d.count, 1)
	d.wg.Add(1)
	go d.relayer()
	return nil
}

// relayer routine relays communications across the defined routes
func (d *Dispatcher) relayer() {
	for {
		select {
		case j := <-d.jobs:
			d.rMtx.RLock()
			if pipes, ok := d.routes[j.ID]; ok {
				for i := range pipes {
					select {
					case pipes[i] <- j.Data:
					default:
						// no receiver; don't wait. This limits complexity.
					}
				}
			}
			d.rMtx.RUnlock()
		case v := <-d.shutdown:
			atomic.AddInt32(&d.count, -1)
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
	if d == nil {
		return errDispatcherNotInitialized
	}

	if id.IsNil() {
		return errIDNotSet
	}

	if data == nil {
		return errNoData
	}

	if atomic.LoadUint32(&d.running) == 0 {
		return nil
	}

	select {
	case d.jobs <- &job{Data: data, ID: id}: // Push job into job channel.
		return nil
	default:
		return fmt.Errorf(limitMessage,
			errDispatcherJobsAtLimit,
			len(d.jobs),
			atomic.LoadInt32(&d.count))
	}
}

// Subscribe subscribes a system and returns a communication chan, this does not
// ensure initial push.
func (d *Dispatcher) subscribe(id uuid.UUID) (<-chan interface{}, error) {
	if d == nil {
		return nil, errDispatcherNotInitialized
	}

	if id.IsNil() {
		return nil, errIDNotSet
	}

	if atomic.LoadUint32(&d.running) == 0 {
		return nil, errDispatcherNotInitialized
	}

	d.rMtx.Lock()
	defer d.rMtx.Unlock()
	_, ok := d.routes[id] // TODO: Pointer to channel slice, benchmark heap issue.
	if !ok {
		return nil, errDispatcherUUIDNotFoundInRouteList
	}

	// Get an unused channel from the channel pool
	ch, ok := d.outbound.Get().(chan interface{})
	if !ok {
		return nil, errTypeAssertionFailure
	}

	d.routes[id] = append(d.routes[id], ch)
	return ch, nil
}

// Unsubscribe unsubs a routine from the dispatcher
func (d *Dispatcher) unsubscribe(id uuid.UUID, usedChan <-chan interface{}) error {
	if d == nil {
		return errDispatcherNotInitialized
	}

	if id.IsNil() {
		return errIDNotSet
	}

	if usedChan == nil {
		return errChannelIsNil
	}

	if atomic.LoadUint32(&d.running) == 0 {
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
