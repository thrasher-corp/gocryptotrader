package engine

import (
	"sync"
	"time"
)

// Settings stores engine params. Please define a settings struct for automatic
// display of instance settings. For example, if you define a struct named
// ManagerSettings, it will be displayed as a subheading "Manager Settings"
// and individual field names such as 'EnableManager' will be displayed
// as "Enable Manager: true/false".
type Settings struct {
	ConfigFile            string
	DataDir               string
	MigrationDir          string
	LogFile               string
	GoMaxProcs            int
	CheckParamInteraction bool

	CoreSettings
	ExchangeSyncerSettings
	ForexSettings
	ExchangeTuningSettings
	GCTScriptSettings
	WithdrawSettings

	// Main shutdown channel
	Shutdown chan struct{}
}

// CoreSettings defines settings related to core engine operations
type CoreSettings struct {
	EnableDryRun                bool
	EnableAllExchanges          bool
	EnableAllPairs              bool
	EnableCoinmarketcapAnalysis bool
	EnablePortfolioManager      bool
	EnableDataHistoryManager    bool
	PortfolioManagerDelay       time.Duration
	EnableGRPC                  bool
	EnableGRPCProxy             bool
	EnableGRPCShutdown          bool
	EnableCommsRelayer          bool
	EnableExchangeSyncManager   bool
	EnableDepositAddressManager bool
	EnableEventManager          bool
	EnableOrderManager          bool
	EnableConnectivityMonitor   bool
	EnableDatabaseManager       bool
	EnableGCTScriptManager      bool
	EnableNTPClient             bool
	EnableWebsocketRoutine      bool
	EnableCurrencyStateManager  bool
	EventManagerDelay           time.Duration
	EnableFuturesTracking       bool
	Verbose                     bool
	EnableDispatcher            bool
	DispatchMaxWorkerAmount     int
	DispatchJobsLimit           int
	Exchanges                   string
}

// ExchangeSyncerSettings defines settings for the exchange pair synchronisation
type ExchangeSyncerSettings struct {
	EnableTickerSyncing    bool
	EnableOrderbookSyncing bool
	EnableTradeSyncing     bool
	SyncWorkersCount       int
	SyncContinuously       bool
	SyncTimeoutREST        time.Duration
	SyncTimeoutWebsocket   time.Duration
}

// ForexSettings defines settings related to the foreign exchange services
type ForexSettings struct {
	EnableCurrencyConverter bool
	EnableCurrencyLayer     bool
	EnableExchangeRates     bool
	EnableFixer             bool
	EnableOpenExchangeRates bool
}

// ExchangeTuningSettings defines settings related to an exchange
type ExchangeTuningSettings struct {
	EnableExchangeHTTPRateLimiter       bool
	EnableExchangeHTTPDebugging         bool
	EnableExchangeVerbose               bool
	ExchangePurgeCredentials            bool
	EnableExchangeAutoPairUpdates       bool
	DisableExchangeAutoPairUpdates      bool
	EnableExchangeRESTSupport           bool
	EnableExchangeWebsocketSupport      bool
	TradeBufferProcessingInterval       time.Duration
	RequestMaxRetryAttempts             int
	AlertSystemPreAllocationCommsBuffer int // See exchanges/alert.go
	ExchangeShutdownTimeout             time.Duration
	HTTPTimeout                         time.Duration
	HTTPUserAgent                       string
	HTTPProxy                           string
	GlobalHTTPTimeout                   time.Duration
	GlobalHTTPUserAgent                 string
	GlobalHTTPProxy                     string
}

// GCTScriptSettings defines settings related to the GCTScript virtual machine
type GCTScriptSettings struct {
	MaxVirtualMachines uint64
}

// WithdrawSettings defines settings related to Withdrawing cryptocurrency
type WithdrawSettings struct {
	WithdrawCacheSize uint64
}

const (
	// MsgStatusOK message to display when status is "OK"
	MsgStatusOK string = "ok"
	// MsgStatusSuccess message to display when status is successful
	MsgStatusSuccess string = "success"
	// MsgStatusError message to display when failure occurs
	MsgStatusError string = "error"
	grpcName       string = "grpc"
	grpcProxyName  string = "grpc_proxy"
)

// newConfigMutex only locks and unlocks on engine creation functions
// as engine modifies global files, this protects the main bot creation
// functions from interfering with each other
var newEngineMutex sync.Mutex
