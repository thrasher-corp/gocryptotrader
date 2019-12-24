package engine

import (
	"container/heap"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofrs/uuid"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

var (
	newErrorChan = func() interface{} { return make(chan error, 1) }
)

// Get gets a new instance of the work mananager
func Get(workers int32, verbose bool) *WorkManager {
	if workers < 1 {
		if verbose {
			log.Warnln(log.WorkMgr, "Invalid worker count using defaults")
		}
		workers = defaultWorkerCount
	}
	workManager := &WorkManager{
		p: &sync.Pool{
			New: newErrorChan,
		},
		workerCount: workers,
		verbose:     verbose,
	}

	heap.Init(&workManager.Jobs)
	return workManager
}

// Start starts the work manager
func (w *WorkManager) Start() error {
	if !atomic.CompareAndSwapInt32(&w.started, 0, 1) {
		return errWorkManagerStarted
	}

	w.shutdown = make(chan struct{})
	w.workAvailable = make(chan struct{}, 1000)

	var engagement sync.WaitGroup
	for i := int32(0); i < w.workerCount; i++ {
		id, err := uuid.NewV4()
		if err != nil {
			return err
		}
		engagement.Add(1)
		go w.worker(id, &engagement)
	}

	// Ensures workers are at the gate before running
	engagement.Wait()

	if atomic.SwapInt32(&w.running, 1) == 1 {
		return errors.New("running can not be set to 1")
	}
	return nil
}

// Stop stops the work manager
func (w *WorkManager) Stop() error {
	if !atomic.CompareAndSwapInt32(&w.running, 1, 0) {
		return errWorkManagerStopped
	}

	// Initiate shutdown
	close(w.shutdown)
	w.wg.Wait()

	// Drain available jobs to free up calling functions
	for i := 0; i < len(w.Jobs); i++ {
		j := heap.Pop(&w.Jobs).(*Job)
		j.err <- errWorkManagerStopped
	}

	if atomic.SwapInt32(&w.started, 0) == 0 {
		return errors.New("started cannot be set to 0")
	}

	return nil
}

// worker defines our worker for job stack manipulation
func (w *WorkManager) worker(id uuid.UUID, engagement *sync.WaitGroup) {
	w.wg.Add(1)
	if w.verbose {
		fmt.Printf("worker: %s started\n", id)
	}

	engagement.Done()

	for {
		select {
		case <-w.workAvailable:
			w.jobsMtx.Lock()
			job := heap.Pop(&w.Jobs).(*Job)
			w.jobsMtx.Unlock()
			if w.verbose {
				log.Debugf(log.WorkMgr,
					"Job recieved %v, by worker: %s, priority: %d\n",
					job.function,
					id,
					job.priority)
			}

			if job.priority != int(cancel) {
				go func(job *Job) {
					// Execute function
					if reserved := job.function.GetReservation(); reserved != nil {
						// If time spent on stack, just sleep for the delta
						delay := reserved.Delay()
						if delay != 0 {
							go func(t time.Duration, j *Job) {
								time.Sleep(t)
								select {
								case j.cancel <- 0:
								default:
								}
							}(delay, job)
							if <-job.cancel == 1 {
								reserved.Cancel()
								job.err <- errors.New("cancelled")
								return
							}
						}
					}
					job.function.Execute()
					job.err <- nil
				}(job)
			} else {
				// Job cancelled
				job.err <- errJobCancelled
			}
		case <-w.shutdown:
			if w.verbose {
				fmt.Printf("worker %s stopped\n", id)
			}
			w.wg.Done()
			return
		}

	}
}

// Exchange initiates a coupling to exchange functionality
func (w *WorkManager) Exchange(callingsystem uuid.UUID, e exchange.IBotExchange) *Exchange {
	// TODO: fetch and set client for system
	return &Exchange{e: e, wm: w}
}

// ExecuteJob validates and checks potential job and inserts it on the stack to
// be then executed via a worker pool and returns an executed channel error
func (w *WorkManager) ExecuteJob(c Command, p Priority, cancel chan int) error {
	if atomic.LoadInt32(&w.running) != 1 {
		return errors.New("system offline")
	}

	job := &Job{function: c, priority: int(p), cancel: cancel}
	job.err = w.p.Get().(chan error)

	w.jobsMtx.Lock()
	heap.Push(&w.Jobs, job)
	w.jobsMtx.Unlock()
	w.workAvailable <- struct{}{} // Send availability

	err := <-job.err
	w.p.Put(job.err)
	return err
}

// Cancel sets cancellation for job on the stack
func (w *WorkManager) Cancel(j *Job) {
	w.jobsMtx.Lock()
	w.Jobs.update(j, j.function, int(cancel))
	w.jobsMtx.Unlock()
}

// Job individual job
type Job struct {
	err      chan error
	cancel   chan int
	function Command
	priority int
	index    int
}

// PriorityJobQueue smoooo
type PriorityJobQueue []*Job

// Len returns length of the total job queue
func (pj PriorityJobQueue) Len() int { return len(pj) }

// Less derives the if i job has the higher priority of the subsequent j job
func (pj PriorityJobQueue) Less(i, j int) bool {
	return pj[i].priority > pj[j].priority
}

// Swap swaps the items over
func (pj PriorityJobQueue) Swap(i, j int) {
	pj[i], pj[j] = pj[j], pj[i]
	pj[i].index = i
	pj[j].index = j
}

// Push adds new job to the stack
func (pj *PriorityJobQueue) Push(x interface{}) {
	n := len(*pj)
	item := x.(*Job)
	item.index = n
	*pj = append(*pj, item)
}

// Pop pops job off stack and returns its val
func (pj *PriorityJobQueue) Pop() interface{} {
	old := *pj
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*pj = old[0 : n-1]
	return item
}

// update modifies the priority and value of an Item in the queue.
func (pj *PriorityJobQueue) update(j *Job, c Command, priority int) {
	j.function = c
	j.priority = priority
	heap.Fix(pj, j.index)
}
