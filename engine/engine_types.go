package engine

import (
	"sync"
	"time"
)

// Settings stores engine params
type Settings struct {
	ConfigFile            string
	DataDir               string
	MigrationDir          string
	LogFile               string
	GoMaxProcs            int
	CheckParamInteraction bool

	// Core Settings
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
	EnableWebsocketRPC          bool
	EnableDeprecatedRPC         bool
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

	// Exchange syncer settings
	EnableTickerSyncing    bool
	EnableOrderbookSyncing bool
	EnableTradeSyncing     bool
	SyncWorkersCount       int
	SyncContinuously       bool
	SyncTimeoutREST        time.Duration
	SyncTimeoutWebsocket   time.Duration

	// Forex settings
	EnableCurrencyConverter bool
	EnableCurrencyLayer     bool
	EnableExchangeRates     bool
	EnableFixer             bool
	EnableOpenExchangeRates bool
	EnableExchangeRateHost  bool

	// Exchange tuning settings
	EnableExchangeHTTPRateLimiter       bool
	EnableExchangeHTTPDebugging         bool
	EnableExchangeVerbose               bool
	ExchangePurgeCredentials            bool
	EnableExchangeAutoPairUpdates       bool
	DisableExchangeAutoPairUpdates      bool
	EnableExchangeRESTSupport           bool
	EnableExchangeWebsocketSupport      bool
	MaxHTTPRequestJobsLimit             int
	TradeBufferProcessingInterval       time.Duration
	RequestMaxRetryAttempts             int
	AlertSystemPreAllocationCommsBuffer int // See exchanges/alert.go

	// Global HTTP related settings
	GlobalHTTPTimeout   time.Duration
	GlobalHTTPUserAgent string
	GlobalHTTPProxy     string

	// Exchange HTTP related settings
	HTTPTimeout   time.Duration
	HTTPUserAgent string
	HTTPProxy     string

	// Dispatch system settings
	EnableDispatcher        bool
	DispatchMaxWorkerAmount int
	DispatchJobsLimit       int

	// GCTscript settings
	MaxVirtualMachines uint

	// Withdraw settings
	WithdrawCacheSize uint64

	// Main shutdown channel
	Shutdown chan struct{}
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
