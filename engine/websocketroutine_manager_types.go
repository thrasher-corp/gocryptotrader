package engine

import (
	"errors"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var (
	errNilCurrencyPairSyncer           = errors.New("nil currency pair syncer received")
	errNilCurrencyConfig               = errors.New("nil currency config received")
	errNilCurrencyPairFormat           = errors.New("nil currency pair format received")
	errNilWebsocketDataHandlerFunction = errors.New("websocket data handler function is nil")
	errNilWebsocket                    = errors.New("websocket is nil")
	errRoutineManagerNotStarted        = errors.New("websocket routine manager not started")
	errUseAPointer                     = errors.New("could not process, pass to websocket routine manager as a pointer")
)

const (
	stoppedState int32 = iota
	startingState
	readyState
)

// WebsocketRoutineManager is used to process websocket updates from a unified location
type WebsocketRoutineManager struct {
	state           int32
	verbose         bool
	exchangeManager iExchangeManager
	orderManager    iOrderManager
	syncer          iCurrencyPairSyncer
	currencyConfig  *currency.Config
	shutdown        chan struct{}
	dataHandlers    []WebsocketDataHandler
	wg              sync.WaitGroup
	mu              sync.RWMutex
	logSequence     chan logCaller
}

// WebsocketDataHandler defines a function signature for a function that handles
// data coming from websocket connections.
type WebsocketDataHandler func(service string, incoming any) error

// Logf sends a formatted log call to the logSequencer channel
func (m *WebsocketRoutineManager) Logf(f func(sl *log.SubLogger, format string, a ...any), subLogger *log.SubLogger, format string, args ...any) {
	m.logSequence <- &formattedSubLogger{
		Func:      f,
		SubLogger: subLogger,
		Format:    format,
		Args:      args,
	}
}

// Logln sends a log call to the logSequencer channel
func (m *WebsocketRoutineManager) Logln(f func(sl *log.SubLogger, a ...any), subLogger *log.SubLogger, args ...any) {
	m.logSequence <- &newLineSubLogger{
		Func:      f,
		SubLogger: subLogger,
		Args:      args,
	}
}

type logCaller interface {
	Call()
}

// formattedSubLogger takes in a SubLogger instance with the log format and corresponding arguments.
type formattedSubLogger struct {
	Func      func(sl *log.SubLogger, format string, a ...any)
	SubLogger *log.SubLogger
	Format    string
	Args      []any
}

// Call implements the logCaller interface
func (f *formattedSubLogger) Call() {
	f.Func(f.SubLogger, f.Format, f.Args...)
}

// newLineSubLogger takes in a SubLogger instance with the log with arguments for log
type newLineSubLogger struct {
	Func      func(sl *log.SubLogger, a ...any)
	SubLogger *log.SubLogger
	Args      []any
}

// Call executes the logCaller interface
func (n *newLineSubLogger) Call() {
	n.Func(n.SubLogger, n.Args...)
}
