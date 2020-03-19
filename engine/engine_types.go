package engine

import (
	"fmt"
	"strings"
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
	PortfolioManagerDelay       time.Duration
	EnableGRPC                  bool
	EnableGRPCProxy             bool
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
	EventManagerDelay           time.Duration
	Verbose                     bool

	// Synchronisation settings
	SyncerSettings SynchronisationSettings

	SyncTimeout time.Duration

	// Forex settings
	EnableCurrencyConverter bool
	EnableCurrencyLayer     bool
	EnableFixer             bool
	EnableOpenExchangeRates bool

	// Exchange tuning settings
	EnableExchangeHTTPRateLimiter  bool
	EnableExchangeHTTPDebugging    bool
	EnableExchangeVerbose          bool
	ExchangePurgeCredentials       bool
	EnableExchangeAutoPairUpdates  bool
	DisableExchangeAutoPairUpdates bool
	EnableExchangeRESTSupport      bool
	EnableExchangeWebsocketSupport bool
	MaxHTTPRequestJobsLimit        int
	RequestTimeoutRetryAttempts    int

	// Global HTTP related settings
	GlobalHTTPTimeout   time.Duration
	GlobalHTTPUserAgent string
	GlobalHTTPProxy     string

	// Exchange HTTP related settings
	ExchangeHTTPTimeout   time.Duration
	ExchangeHTTPUserAgent string
	ExchangeHTTPProxy     string

	// Dispatch system settings
	EnableDispatcher        bool
	DispatchMaxWorkerAmount int
	DispatchJobsLimit       int

	// GCTscript settings
	MaxVirtualMachines uint

	// Withdraw settings
	WithdrawCacheSize uint64
}

const (
	// ErrSubSystemAlreadyStarted message to return when a subsystem is already started
	ErrSubSystemAlreadyStarted = "manager already started"
	// ErrSubSystemAlreadyStopped message to return when a subsystem is already stopped
	ErrSubSystemAlreadyStopped = "already stopped"
	// ErrSubSystemNotStarted message to return when subsystem not started
	ErrSubSystemNotStarted = "not started"

	// ErrScriptFailedValidation message to display when a script fails its validation
	ErrScriptFailedValidation string = "validation failed"
	// MsgSubSystemStarting message to return when subsystem is starting up
	MsgSubSystemStarting = "manager starting..."
	// MsgSubSystemStarted message to return when subsystem has started
	MsgSubSystemStarted = "started."

	// MsgSubSystemShuttingDown message to return when a subsystem is shutting down
	MsgSubSystemShuttingDown = "shutting down..."
	// MsgSubSystemShutdown message to return when a subsystem has shutdown
	MsgSubSystemShutdown = "manager shutdown."

	// MsgStatusOK message to display when status is "OK"
	MsgStatusOK string = "ok"
	// MsgStatusSuccess message to display when status is successful
	MsgStatusSuccess string = "success"
	// MsgStatusError message to display when failure occurs
	MsgStatusError string = "error"
)

// SynchronisationSettings implements the flag.Value interface
type SynchronisationSettings struct {
	EnableExchangeTicker         bool
	EnableExchangeOrderbook      bool
	EnableExchangeTrade          bool
	EnableExchangeSupportedPairs bool
	EnableAccountBalance         bool
}

// String method returns a string
func (i *SynchronisationSettings) String() string { return "" }

// Set method takes in a comma delimitered string turns off what is not
// needed
func (i *SynchronisationSettings) Set(value string) error {
	vals := strings.Split(value, ",")
	for x := range vals {
		switch vals[x] {
		case "balance":
			i.EnableAccountBalance = false
		case "trade":
			i.EnableExchangeTrade = false
		case "orderbook":
			i.EnableExchangeOrderbook = false
		case "supportedpairs":
			i.EnableExchangeSupportedPairs = false
		case "ticker":
			i.EnableExchangeTicker = false
		default:
			return fmt.Errorf("cannot disable sync agent value: %s not found ",
				vals[x])
		}
	}
	return nil
}
