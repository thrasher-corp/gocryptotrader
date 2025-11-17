package dispatch

import (
	"sync"
	"time"

	"github.com/gofrs/uuid"
)

const (
	// DefaultJobsLimit defines a maximum amount of jobs allowed in channel
	DefaultJobsLimit = 100

	// DefaultMaxWorkers is the package default worker ceiling amount
	DefaultMaxWorkers = 10

	// DefaultHandshakeTimeout defines a workers max length of time to wait on a
	// an unbuffered channel for a receiver before moving on to next route
	DefaultHandshakeTimeout = 200 * time.Nanosecond
)

// dispatcher is our main in memory instance with a stop/start mtx below
var dispatcher *Dispatcher

// Dispatcher defines an internal subsystem communication/change state publisher
type Dispatcher struct {
	// routes refers to a subsystem uuid ticket map with associated publish
	// channels, a relayer will be given a unique id through its job channel,
	// then publish the data across the full registered channels for that uuid.
	// See relayer() method below.
	routes map[uuid.UUID][]chan any
	// routesMtx protects the routes variable ensuring acceptable read/write access
	routesMtx sync.Mutex

	// Persistent buffered job queue for relayers
	jobs chan job

	// Dynamic channel pool; returns an unbuffered channel for routes map
	outbound sync.Pool

	// MaxWorkers defines max worker ceiling
	maxWorkers int

	// Dispatch status
	running bool

	// Unbufferd shutdown chan, sync wg for ensuring concurrency when only
	// dropping a single relayer routine
	shutdown chan struct{}

	// Relayer shutdown tracking
	wg sync.WaitGroup

	// dispatcher write protection
	m sync.RWMutex
	// subscriberCount atomically stores the amount of subscription endpoints
	// to verify whether to send out any jobs
	subscriberCount int32
}

// job defines a relaying job associated with a ticket which allows routing to
// routines that require specific data
type job struct {
	Data any
	ID   uuid.UUID
}

// Mux defines a new multiplexer for the dispatch system, these are generated
// per subsystem
type Mux struct {
	// Reference to the main running dispatch service
	d *Dispatcher
}

// Pipe defines an outbound object to the desired routine
type Pipe struct {
	// Channel to get all our lovely information
	c chan any
	// ID to tracked system
	id uuid.UUID
	// Reference to multiplexer
	m *Mux
}
