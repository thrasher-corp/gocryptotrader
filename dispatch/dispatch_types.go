package dispatch

import (
	"sync"
	"time"

	"github.com/gofrs/uuid"
)

const (
	// DefaultJobsLimit defines a maxiumum amount of jobs allowed in channel
	DefaultJobsLimit = 100

	// DefaultMaxWorkers is the package default worker ceiling amount
	DefaultMaxWorkers = 10

	// DefaultHandshakeTimeout defines a workers max length of time to wait on a
	// an unbuffered channel for a receiver before moving on to next route
	DefaultHandshakeTimeout = 200 * time.Nanosecond

	errNotInitialised   = "dispatcher not initialised"
	errShutdownRoutines = "dispatcher did not shutdown properly, routines failed to close"
)

// dispatcher is our main in memory instance with a stop/start mtx below
var dispatcher *Dispatcher
var mtx sync.Mutex

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

	// MaxWorkers defines max worker ceiling
	maxWorkers int32
	// Atomic values -----------------------
	// Worker counter
	count int32
	// Dispatch status
	running uint32

	// Unbufferd shutdown chan, sync wg for ensuring concurrency when only
	// dropping a single relayer routine
	shutdown chan *sync.WaitGroup

	// Relayer shutdown tracking
	wg sync.WaitGroup
}

// job defines a relaying job associated with a ticket which allows routing to
// routines that require specific data
type job struct {
	Data interface{}
	ID   uuid.UUID
}

// Mux defines a new multiplexer for the dispatch system, these a generated
// per subsystem
type Mux struct {
	// Reference to the main running dispatch service
	d *Dispatcher
	sync.RWMutex
}

// Pipe defines an outbound object to the desired routine
type Pipe struct {
	// Channel to get all our lovely informations
	C chan interface{}
	// ID to tracked system
	id uuid.UUID
	// Reference to multiplexer
	m *Mux
}
