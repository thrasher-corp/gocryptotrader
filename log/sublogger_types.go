package log

import "io"

// Global vars related to the logger package
var (
	subLoggers = map[string]*subLogger{}

	Global           *subLogger
	BackTester       *subLogger
	ConnectionMgr    *subLogger
	CommunicationMgr *subLogger
	ConfigMgr        *subLogger
	DatabaseMgr      *subLogger
	GCTScriptMgr     *subLogger
	OrderMgr         *subLogger
	PortfolioMgr     *subLogger
	SyncMgr          *subLogger
	TimeMgr          *subLogger
	WebsocketMgr     *subLogger
	EventMgr         *subLogger
	DispatchMgr      *subLogger

	RequestSys  *subLogger
	ExchangeSys *subLogger
	GRPCSys     *subLogger
	RESTSys     *subLogger

	Ticker    *subLogger
	OrderBook *subLogger
	Trade     *subLogger
)

// logFields is used to store data in a non-global and thread-safe manner
// so logs cannot be modified mid-log causing a data-race issue
type logFields struct {
	info   bool
	warn   bool
	debug  bool
	error  bool
	name   string
	output io.Writer
	logger Logger
}
