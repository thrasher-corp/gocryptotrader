package engine

import "time"

// Settings stores engine params
type Settings struct {
	ConfigFile string
	DataDir    string
	LogFile    string
	GoMaxProcs int

	// Core Settings
	EnableDryRun           bool
	EnableAllExchanges     bool
	EnableAllPairs         bool
	EnablePortfolioWatcher bool
	EnableWebsocketServer  bool
	EnableRESTServer       bool
	EnableTickerRoutine    bool
	EnableOrderbookRoutine bool
	EnableWebsocketRoutine bool
	EnableCommsRelayer     bool
	Verbose                bool

	// Exchange tuning settings
	EnableHTTPRateLimiter          bool
	EnableExchangeVerbose          bool
	EnableExchangeAutoPairUpdates  bool
	EnableExchangeRESTSupport      bool
	EnableExchangeWebsocketSupport bool
	MaxHTTPRequestJobsLimit        int

	// Global HTTP related settings
	GlobalHTTPTimeout   time.Duration
	GlobalHTTPUserAgent string
	GlobalHTTPProxy     string

	// Exchange HTTP related settings
	ExchangeHTTPTimeout   time.Duration
	ExchangeHTTPUserAgent string
	ExchangeHTTPProxy     string
}
