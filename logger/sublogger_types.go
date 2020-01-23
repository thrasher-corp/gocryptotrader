package logger

//nolint
var (
	subLoggers = map[string]*subLogger{}

	Global           *subLogger
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

	ExchangeSys *subLogger
	GRPCSys     *subLogger
	RESTSys     *subLogger

	Ticker    *subLogger
	OrderBook *subLogger
)
