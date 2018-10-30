package engine

import "time"

// Settings stores engine params
type Settings struct {
	ConfigFile string
	DataDir    string
	LogFile    string
	GoMaxProcs int

	// Core Settings
	EnableDryRun                bool
	EnableAllExchanges          bool
	EnableAllPairs              bool
	EnableCoinmarketcapAnalysis bool
	EnablePortfolioWatcher      bool
	EnableGRPC                  bool
	EnableGRPCProxy             bool
	EnableWebsocketRPC          bool
	EnableDeprecatedRPC         bool
	EnableTickerRoutine         bool
	EnableOrderbookRoutine      bool
	EnableWebsocketRoutine      bool
	EnableCommsRelayer          bool
	EnableEventManager          bool
	EnableNTPClient             bool
	EventManagerDelay           time.Duration
	Verbose                     bool

	// Forex settings
	EnableCurrencyConverter bool
	EnableCurrencyLayer     bool
	EnableFixer             bool
	EnableOpenExchangeRates bool

	// Exchange tuning settings
	EnableHTTPRateLimiter          bool
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
}
