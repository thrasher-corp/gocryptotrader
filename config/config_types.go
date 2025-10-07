package config

import (
	"errors"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	gctscript "github.com/thrasher-corp/gocryptotrader/gctscript/vm"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
)

// Constants declared here are filename strings and test strings
const (
	FXProviderFixer                      = "fixer"
	EncryptedFile                        = "config.dat"
	File                                 = "config.json"
	TestFile                             = "../testdata/configtest.json"
	fileEncryptionPrompt                 = 0
	fileEncryptionEnabled                = 1
	fileEncryptionDisabled               = -1
	pairsLastUpdatedWarningThreshold     = 30 // 30 days
	defaultHTTPTimeout                   = time.Second * 15
	defaultWebsocketOrderbookBufferLimit = 5
	DefaultConnectionMonitorDelay        = time.Second * 2
	maxAuthFailures                      = 3
	defaultNTPAllowedDifference          = 50000000
	defaultNTPAllowedNegativeDifference  = 50000000
	DefaultAPIKey                        = "Key"
	DefaultAPISecret                     = "Secret"
	DefaultAPIClientID                   = "ClientID"
	defaultDataHistoryMonitorCheckTimer  = time.Minute
	defaultCurrencyStateManagerDelay     = time.Minute
	defaultMaxJobsPerCycle               = 5
	// DefaultSyncerWorkers limits the number of sync workers
	DefaultSyncerWorkers = 15
	// DefaultSyncerTimeoutREST the default time to switch from REST to websocket protocols without a response
	DefaultSyncerTimeoutREST = time.Second * 15
	// DefaultSyncerTimeoutWebsocket the default time to switch from websocket to REST protocols without a response
	DefaultSyncerTimeoutWebsocket = time.Minute
	// DefaultWebsocketResponseCheckTimeout is the default timeout for
	// websocket responses.
	DefaultWebsocketResponseCheckTimeout = time.Millisecond * 30
	// DefaultWebsocketResponseMaxLimit is the default maximum time for
	// websocket responses.
	DefaultWebsocketResponseMaxLimit = time.Second * 7
	// DefaultWebsocketTrafficTimeout is the default timeout for websocket
	// traffic.
	DefaultWebsocketTrafficTimeout = time.Second * 30
)

// Constants here hold some messages
const (
	warningExchangeAuthAPIDefaultOrEmptyValues = "exchange %s authenticated API support disabled due to default/empty APIKey/Secret/ClientID values"
	warningPairsLastUpdatedThresholdExceeded   = "exchange %s last manual update of available currency pairs has exceeded %d days. Manual update required!"
)

// Constants here define unset default values displayed in the config.json
// file
const (
	APIURLNonDefaultMessage       = "NON_DEFAULT_HTTP_LINK_TO_EXCHANGE_API"
	WebsocketURLNonDefaultMessage = "NON_DEFAULT_HTTP_LINK_TO_WEBSOCKET_EXCHANGE_API"
	DefaultUnsetAPIKey            = "Key"
	DefaultUnsetAPISecret         = "Secret"
	DefaultUnsetAccountPlan       = "accountPlan"
	DefaultGRPCUsername           = "admin"
	DefaultGRPCPassword           = "Password"
)

// Public errors exported by this package
var (
	ErrExchangeNotFound     = errors.New("exchange not found")
	ErrFailureOpeningConfig = errors.New("fatal error opening file")
)

var (
	cfg Config
	m   sync.Mutex

	errNoEnabledExchanges   = errors.New("no exchanges enabled")
	errCheckingConfigValues = errors.New("fatal error checking config values")
)

// Config is the overarching object that holds all the information for
// prestart management of Portfolio, Communications, Webserver and Enabled Exchanges
type Config struct {
	Name                 string                    `json:"name"`
	Version              int                       `json:"version"`
	DataDirectory        string                    `json:"dataDirectory"`
	EncryptConfig        int                       `json:"encryptConfig"`
	GlobalHTTPTimeout    time.Duration             `json:"globalHTTPTimeout"`
	Database             database.Config           `json:"database"`
	Logging              log.Config                `json:"logging"`
	SyncManagerConfig    SyncManagerConfig         `json:"syncManager"`
	ConnectionMonitor    ConnectionMonitorConfig   `json:"connectionMonitor"`
	OrderManager         OrderManager              `json:"orderManager"`
	DataHistoryManager   DataHistoryManager        `json:"dataHistoryManager"`
	CurrencyStateManager CurrencyStateManager      `json:"currencyStateManager"`
	Profiler             Profiler                  `json:"profiler"`
	NTPClient            NTPClientConfig           `json:"ntpclient"`
	GCTScript            gctscript.Config          `json:"gctscript"`
	Currency             currency.Config           `json:"currencyConfig"`
	Communications       base.CommunicationsConfig `json:"communications"`
	RemoteControl        RemoteControlConfig       `json:"remoteControl"`
	Portfolio            *portfolio.Base           `json:"portfolioAddresses"`
	Exchanges            []Exchange                `json:"exchanges"`
	BankAccounts         []banking.Account         `json:"bankAccounts"`

	// Deprecated config settings, will be removed at a future date
	CurrencyPairFormat  *currency.PairFormat  `json:"currencyPairFormat,omitempty"`
	FiatDisplayCurrency *currency.Code        `json:"fiatDispayCurrency,omitempty"`
	Cryptocurrencies    *currency.Currencies  `json:"cryptocurrencies,omitempty"`
	SMS                 *base.SMSGlobalConfig `json:"smsGlobal,omitempty"`
	// encryption session values
	storedSalt            []byte
	sessionDK             []byte
	EncryptionKeyProvider EncryptionKeyProvider `json:"-"`
}

// EncryptionKeyProvider is a function config can use to prompt the user for an encryption key
type EncryptionKeyProvider func(confirmKey bool) ([]byte, error)

// OrderManager holds settings used for the order manager
type OrderManager struct {
	Enabled                       bool          `json:"enabled"`
	Verbose                       bool          `json:"verbose"`
	ActivelyTrackFuturesPositions bool          `json:"activelyTrackFuturesPositions"`
	FuturesTrackingSeekDuration   time.Duration `json:"futuresTrackingSeekDuration"`
	RespectOrderHistoryLimits     bool          `json:"respectOrderHistoryLimits"`
	CancelOrdersOnShutdown        bool          `json:"cancelOrdersOnShutdown"`
}

// DataHistoryManager holds all information required for the data history manager
type DataHistoryManager struct {
	Enabled             bool          `json:"enabled"`
	CheckInterval       time.Duration `json:"checkInterval"`
	MaxJobsPerCycle     int64         `json:"maxJobsPerCycle"`
	MaxResultInsertions int64         `json:"maxResultInsertions"`
	Verbose             bool          `json:"verbose"`
}

// CurrencyStateManager defines a set of configuration options for the currency
// state manager
type CurrencyStateManager struct {
	Enabled *bool         `json:"enabled"`
	Delay   time.Duration `json:"delay"`
}

// SyncManagerConfig stores the currency pair synchronization manager config
type SyncManagerConfig struct {
	Enabled                 bool                 `json:"enabled"`
	SynchronizeTicker       bool                 `json:"synchronizeTicker"`
	SynchronizeOrderbook    bool                 `json:"synchronizeOrderbook"`
	SynchronizeTrades       bool                 `json:"synchronizeTrades"`
	SynchronizeContinuously bool                 `json:"synchronizeContinuously"`
	TimeoutREST             time.Duration        `json:"timeoutREST"`
	TimeoutWebsocket        time.Duration        `json:"timeoutWebsocket"`
	NumWorkers              int                  `json:"numWorkers"`
	FiatDisplayCurrency     currency.Code        `json:"fiatDisplayCurrency"`
	PairFormatDisplay       *currency.PairFormat `json:"pairFormatDisplay,omitempty"`
	// log events
	Verbose                 bool `json:"verbose"`
	LogSyncUpdateEvents     bool `json:"logSyncUpdateEvents"`
	LogSwitchProtocolEvents bool `json:"logSwitchProtocolEvents"`
	LogInitialSyncEvents    bool `json:"logInitialSyncEvents"`
}

// ConnectionMonitorConfig defines the connection monitor variables to ensure
// that there is internet connectivity
type ConnectionMonitorConfig struct {
	DNSList          []string      `json:"preferredDNSList"`
	PublicDomainList []string      `json:"preferredDomainList"`
	CheckInterval    time.Duration `json:"checkInterval"`
}

// Exchange holds all the information needed for each enabled Exchange.
type Exchange struct {
	Name                          string                 `json:"name"`
	Enabled                       bool                   `json:"enabled"`
	Verbose                       bool                   `json:"verbose"`
	UseSandbox                    bool                   `json:"useSandbox,omitempty"`
	HTTPTimeout                   time.Duration          `json:"httpTimeout"`
	HTTPUserAgent                 string                 `json:"httpUserAgent,omitempty"`
	HTTPDebugging                 bool                   `json:"httpDebugging,omitempty"`
	WebsocketResponseCheckTimeout time.Duration          `json:"websocketResponseCheckTimeout"`
	WebsocketResponseMaxLimit     time.Duration          `json:"websocketResponseMaxLimit"`
	WebsocketTrafficTimeout       time.Duration          `json:"websocketTrafficTimeout"`
	ConnectionMonitorDelay        time.Duration          `json:"connectionMonitorDelay"`
	ProxyAddress                  string                 `json:"proxyAddress,omitempty"`
	BaseCurrencies                currency.Currencies    `json:"baseCurrencies"`
	CurrencyPairs                 *currency.PairsManager `json:"currencyPairs"`
	API                           APIConfig              `json:"api"`
	Features                      *FeaturesConfig        `json:"features"`
	BankAccounts                  []banking.Account      `json:"bankAccounts,omitempty"`
	Orderbook                     Orderbook              `json:"orderbook"`

	// Deprecated settings which will be removed in a future update
	AuthenticatedAPISupport          *bool   `json:"authenticatedApiSupport,omitempty"`
	AuthenticatedWebsocketAPISupport *bool   `json:"authenticatedWebsocketApiSupport,omitempty"`
	APIKey                           *string `json:"apiKey,omitempty"`
	APISecret                        *string `json:"apiSecret,omitempty"`
	APIAuthPEMKeySupport             *bool   `json:"apiAuthPemKeySupport,omitempty"`
	APIAuthPEMKey                    *string `json:"apiAuthPemKey,omitempty"`
	APIURL                           *string `json:"apiUrl,omitempty"`
	APIURLSecondary                  *string `json:"apiUrlSecondary,omitempty"`
	ClientID                         *string `json:"clientId,omitempty"`
	SupportsAutoPairUpdates          *bool   `json:"supportsAutoPairUpdates,omitempty"`
	Websocket                        *bool   `json:"websocket,omitempty"`
	WebsocketURL                     *string `json:"websocketUrl,omitempty"`
}

// Profiler defines the profiler configuration to enable pprof
type Profiler struct {
	Enabled              bool   `json:"enabled"`
	MutexProfileFraction int    `json:"mutex_profile_fraction"`
	ListenAddress        string `json:"listen_address"`
	BlockProfileRate     int    `json:"block_profile_rate"`
}

// NTPClientConfig defines a network time protocol configuration to allow for
// positive and negative differences
type NTPClientConfig struct {
	Level                     int            `json:"enabled"`
	Pool                      []string       `json:"pool"`
	AllowedDifference         *time.Duration `json:"allowedDifference"`
	AllowedNegativeDifference *time.Duration `json:"allowedNegativeDifference"`
}

// GRPCConfig stores the gRPC settings
type GRPCConfig struct {
	Enabled                bool   `json:"enabled"`
	ListenAddress          string `json:"listenAddress"`
	GRPCProxyEnabled       bool   `json:"grpcProxyEnabled"`
	GRPCProxyListenAddress string `json:"grpcProxyListenAddress"`
	GRPCAllowBotShutdown   bool   `json:"grpcAllowBotShutdown"`
	TimeInNanoSeconds      bool   `json:"timeInNanoSeconds"`
}

// RemoteControlConfig stores the RPC services config
type RemoteControlConfig struct {
	Username string     `json:"username"`
	Password string     `json:"password"`
	GRPC     GRPCConfig `json:"gRPC"`
}

// Post holds the bot configuration data
type Post struct {
	Data Config `json:"data"`
}

// CurrencyPairFormatConfig stores the users preferred currency pair display
type CurrencyPairFormatConfig struct {
	Uppercase bool   `json:"uppercase"`
	Delimiter string `json:"delimiter,omitempty"`
	Separator string `json:"separator,omitempty"`
	Index     string `json:"index,omitempty"`
}

// BankTransaction defines a related banking transaction
type BankTransaction struct {
	ReferenceNumber     string `json:"referenceNumber"`
	TransactionNumber   string `json:"transactionNumber"`
	PaymentInstructions string `json:"paymentInstructions"`
}

// FeaturesSupportedConfig stores the exchanges supported features
type FeaturesSupportedConfig struct {
	REST                  bool              `json:"restAPI"`
	RESTCapabilities      protocol.Features `json:"restCapabilities,omitzero"`
	Websocket             bool              `json:"websocketAPI"`
	WebsocketCapabilities protocol.Features `json:"websocketCapabilities,omitzero"`
}

// FeaturesEnabledConfig stores the exchanges enabled features
type FeaturesEnabledConfig struct {
	AutoPairUpdates bool `json:"autoPairUpdates"`
	Websocket       bool `json:"websocketAPI"`
	SaveTradeData   bool `json:"saveTradeData"`
	TradeFeed       bool `json:"tradeFeed"`
	FillsFeed       bool `json:"fillsFeed"`
}

// FeaturesConfig stores the exchanges supported and enabled features
type FeaturesConfig struct {
	Supports      FeaturesSupportedConfig `json:"supports"`
	Enabled       FeaturesEnabledConfig   `json:"enabled"`
	Subscriptions subscription.List       `json:"subscriptions,omitempty"`
}

// APIEndpointsConfig stores the API endpoint addresses
type APIEndpointsConfig struct {
	URL          string `json:"url"`
	URLSecondary string `json:"urlSecondary"`
	WebsocketURL string `json:"websocketURL"`
}

// APICredentialsConfig stores the API credentials
type APICredentialsConfig struct {
	Key           string `json:"key,omitempty"`
	Secret        string `json:"secret,omitempty"`
	ClientID      string `json:"clientID,omitempty"`
	Subaccount    string `json:"subaccount,omitempty"`
	PEMKey        string `json:"pemKey,omitempty"`
	OTPSecret     string `json:"otpSecret,omitempty"`
	TradePassword string `json:"tradePassword,omitempty"`
	PIN           string `json:"pin,omitempty"`
}

// APICredentialsValidatorConfig stores the API credentials validator settings
type APICredentialsValidatorConfig struct {
	// For Huobi (optional)
	RequiresPEM bool `json:"requiresPEM,omitempty"`

	RequiresKey                bool `json:"requiresKey,omitempty"`
	RequiresSecret             bool `json:"requiresSecret,omitempty"`
	RequiresClientID           bool `json:"requiresClientID,omitempty"`
	RequiresBase64DecodeSecret bool `json:"requiresBase64DecodeSecret,omitempty"`
}

// APIConfig stores the exchange API config
type APIConfig struct {
	AuthenticatedSupport          bool `json:"authenticatedSupport"`
	AuthenticatedWebsocketSupport bool `json:"authenticatedWebsocketApiSupport"`
	PEMKeySupport                 bool `json:"pemKeySupport,omitempty"`

	Credentials          APICredentialsConfig           `json:"credentials"`
	CredentialsValidator *APICredentialsValidatorConfig `json:"credentialsValidator,omitempty"`
	OldEndPoints         *APIEndpointsConfig            `json:"endpoints,omitempty"`
	Endpoints            map[string]string              `json:"urlEndpoints"`
}

// Orderbook stores the orderbook configuration variables
type Orderbook struct {
	VerificationBypass     bool `json:"verificationBypass"`
	WebsocketBufferLimit   int  `json:"websocketBufferLimit"`
	WebsocketBufferEnabled bool `json:"websocketBufferEnabled"`
}
