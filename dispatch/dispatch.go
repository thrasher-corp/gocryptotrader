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

// Name is an exported subsystem name
const Name = "dispatch"

func init() {
	dispatcher = &Dispatcher{
		routes: make(map[uuid.UUID][]chan interface{}),
		outbound: sync.Pool{
			New: func() interface{} {
				// Create unbuffered channel for data pass
				return make(chan interface{})
			},
		},
	}
}

// Start starts the dispatch system by spawning workers and allocating memory
func Start(workers, jobsLimit int) error {
	if dispatcher == nil {
		return errors.New(errNotInitialised)
	}

	mtx.Lock()
	defer mtx.Unlock()
	return dispatcher.start(workers, jobsLimit)
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

	dispatcher.dropWorker()
	return nil
}

// SpawnWorker starts a new worker routine
func SpawnWorker() error {
	if dispatcher == nil {
		return errors.New(errNotInitialised)
	}
	return dispatcher.spawnWorker()
}

// start compares atomic running value, sets defaults, overides with
// configuration, then spawns workers
func (d *Dispatcher) start(workers, channelCapacity int) error {
	if atomic.LoadUint32(&d.running) == 1 {
		return errors.New("dispatcher already running")
	}

	if workers < 1 {
		log.Warn(log.DispatchMgr,
			"Dispatcher: workers cannot be zero, using default values")
		workers = DefaultMaxWorkers
	}
	if channelCapacity < 1 {
		log.Warn(log.DispatchMgr,
			"Dispatcher: jobs limit cannot be zero, using default values")
		channelCapacity = DefaultJobsLimit
	}
	d.jobs = make(chan *job, channelCapacity)
	d.maxWorkers = int32(workers)
	d.shutdown = make(chan *sync.WaitGroup)

	if atomic.LoadInt32(&d.count) != 0 {
		return errors.New("dispatcher leaked workers found")
	}

	for i := int32(0); i < d.maxWorkers; i++ {
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
		return errors.New("dispatcher not running")
	}
	close(d.shutdown)
	ch := make(chan struct{})
	timer := time.NewTimer(1 * time.Second)
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
	if d == nil {
		return false
	}
	return atomic.LoadUint32(&d.running) == 1
}

// dropWorker deallocates a worker routine
func (d *Dispatcher) dropWorker() {
	wg := sync.WaitGroup{}
	wg.Add(1)
	d.shutdown <- &wg
	wg.Wait()
}

// spawnWorker allocates a new worker for job processing
func (d *Dispatcher) spawnWorker() error {
	if atomic.LoadInt32(&d.count) >= d.maxWorkers {
		return errors.New("dispatcher cannot spawn more workers; ceiling reached")
	}
	var spawnWg sync.WaitGroup
	spawnWg.Add(1)
	go d.relayer(&spawnWg)
	spawnWg.Wait()
	return nil
}

// relayer routine relays communications across the defined routes
func (d *Dispatcher) relayer(i *sync.WaitGroup) {
	atomic.AddInt32(&d.count, 1)
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
		return fmt.Errorf("dispatcher jobs at limit [%d] current worker count [%d]. Spawn more workers via --dispatchworkers=x"+
			", or increase the jobs limit via --dispatchjobslimit=x",
			len(d.jobs),
			atomic.LoadInt32(&d.count))
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
		// reference will already be released in the stop function
		return nil
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
