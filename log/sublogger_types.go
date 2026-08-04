package log

// Global vars related to the logger package
var (
	SubLoggers = map[string]*SubLogger{}

	Global           *SubLogger
	ConnectionMgr    *SubLogger
	CommunicationMgr *SubLogger
	APIServerMgr     *SubLogger
	ConfigMgr        *SubLogger
	DatabaseMgr      *SubLogger
	DataHistory      *SubLogger
	GCTScriptMgr     *SubLogger
	OrderMgr         *SubLogger
	PortfolioMgr     *SubLogger
	SyncMgr          *SubLogger
	TimeMgr          *SubLogger
	WebsocketMgr     *SubLogger
	EventMgr         *SubLogger
	DispatchMgr      *SubLogger

	RequestSys  *SubLogger
	ExchangeSys *SubLogger
	GRPCSys     *SubLogger
	RESTSys     *SubLogger

	Ticker    *SubLogger
	OrderBook *SubLogger
	Trade     *SubLogger
	Fill      *SubLogger
	Currency  *SubLogger
)

// SubLogger defines a sub logger can be used externally for packages wanted to
// leverage GCT library logger features.
type SubLogger struct {
	name              string
	levels            Levels
	output            *multiWriterHolder
	botName           string
	structuredLogging bool
}

// fields stores per-call data while logger references configuration protected
// by mu. Callers hold mu's read lock for the lifetime of a checked-out value.
type fields struct {
	info              bool
	warn              bool
	debug             bool
	error             bool
	structuredLogging bool
	name              string
	output            *multiWriterHolder
	logger            *Logger
	botName           string
	structuredFields  ExtraFields
}
