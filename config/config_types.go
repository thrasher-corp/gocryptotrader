package config

import (
	"time"

	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/currency/forexprovider/base"
	log "github.com/thrasher-/gocryptotrader/logger"
	"github.com/thrasher-/gocryptotrader/portfolio"
)

// Config is the overarching object that holds all the information for
// prestart management of Portfolio, Communications, Webserver and Enabled
// Exchanges
type Config struct {
	Name              string                  `json:"name"`
	EncryptConfig     int                     `json:"encryptConfig"`
	GlobalHTTPTimeout time.Duration           `json:"globalHTTPTimeout"`
	Logging           log.Logging             `json:"logging"`
	ConnectionMonitor ConnectionMonitorConfig `json:"connectionMonitor"`
	Profiler          ProfilerConfig          `json:"profiler"`
	NTPClient         NTPClientConfig         `json:"ntpclient"`
	Currency          CurrencyConfig          `json:"currencyConfig"`
	Communications    CommunicationsConfig    `json:"communications"`
	RemoteControl     RemoteControlConfig     `json:"remoteControl"`
	Portfolio         portfolio.Base          `json:"portfolioAddresses"`
	Exchanges         []ExchangeConfig        `json:"exchanges"`
	BankAccounts      []BankAccount           `json:"bankAccounts"`

	// Deprecated config settings, will be removed at a future date
	Webserver           *WebserverConfig          `json:"webserver,omitempty"`
	CurrencyPairFormat  *CurrencyPairFormatConfig `json:"currencyPairFormat,omitempty"`
	FiatDisplayCurrency currency.Code             `json:"fiatDispayCurrency,omitempty"`
	Cryptocurrencies    currency.Currencies       `json:"cryptocurrencies,omitempty"`
	SMS                 *SMSGlobalConfig          `json:"smsGlobal,omitempty"`
}

// ConnectionMonitorConfig defines the connection monitor variables to ensure
// that there is internet connectivity
type ConnectionMonitorConfig struct {
	DNSList          []string      `json:"preferredDNSList"`
	PublicDomainList []string      `json:"preferredDomainList"`
	CheckInterval    time.Duration `json:"checkInterval"`
}

// ExchangeConfig holds all the information needed for each enabled Exchange.
type ExchangeConfig struct {
	Name            string                 `json:"name"`
	Enabled         bool                   `json:"enabled"`
	Verbose         bool                   `json:"verbose"`
	UseSandbox      bool                   `json:"useSandbox,omitempty"`
	HTTPTimeout     time.Duration          `json:"httpTimeout"`
	HTTPUserAgent   string                 `json:"httpUserAgent,omitempty"`
	HTTPRateLimiter *HTTPRateLimitConfig   `json:"httpRateLimiter,omitempty"`
	ProxyAddress    string                 `json:"proxyAddress,omitempty"`
	BaseCurrencies  currency.Currencies    `json:"baseCurrencies"`
	CurrencyPairs   *currency.PairsManager `json:"currencyPairs"`
	API             APIConfig              `json:"api"`
	Features        *FeaturesConfig        `json:"features"`
	BankAccounts    []BankAccount          `json:"bankAccounts,omitempty"`

	// Deprecated settings which will be removed in a future update
	AvailablePairs            *currency.Pairs      `json:"availablePairs,omitempty"`
	EnabledPairs              *currency.Pairs      `json:"enabledPairs,omitempty"`
	AssetTypes                *string              `json:"assetTypes,omitempty"`
	PairsLastUpdated          *int64               `json:"pairsLastUpdated,omitempty"`
	ConfigCurrencyPairFormat  *currency.PairFormat `json:"configCurrencyPairFormat,omitempty"`
	RequestCurrencyPairFormat *currency.PairFormat `json:"requestCurrencyPairFormat,omitempty"`
	AuthenticatedAPISupport   *bool                `json:"authenticatedApiSupport,omitempty"`
	APIKey                    *string              `json:"apiKey,omitempty"`
	APISecret                 *string              `json:"apiSecret,omitempty"`
	APIAuthPEMKeySupport      *bool                `json:"apiAuthPemKeySupport,omitempty"`
	APIAuthPEMKey             *string              `json:"apiAuthPemKey,omitempty"`
	APIURL                    *string              `json:"apiUrl,omitempty"`
	APIURLSecondary           *string              `json:"apiUrlSecondary,omitempty"`
	ClientID                  *string              `json:"clientId,omitempty"`
	SupportsAutoPairUpdates   *bool                `json:"supportsAutoPairUpdates,omitempty"`
	Websocket                 *bool                `json:"websocket,omitempty"`
	WebsocketURL              *string              `json:"websocketUrl,omitempty"`
}

// ProfilerConfig defines the profiler configuration to enable pprof
type ProfilerConfig struct {
	Enabled bool `json:"enabled"`
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
}

// DepcrecatedRPCConfig stores the deprecatedRPCConfig settings
type DepcrecatedRPCConfig struct {
	Enabled       bool   `json:"enabled"`
	ListenAddress string `json:"listenAddress"`
}

// WebsocketRPCConfig stores the websocket config info
type WebsocketRPCConfig struct {
	Enabled             bool   `json:"enabled"`
	ListenAddress       string `json:"listenAddress"`
	ConnectionLimit     int    `json:"connectionLimit"`
	MaxAuthFailures     int    `json:"maxAuthFailures"`
	AllowInsecureOrigin bool   `json:"allowInsecureOrigin"`
}

// RemoteControlConfig stores the RPC services config
type RemoteControlConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`

	GRPC          GRPCConfig           `json:"gRPC"`
	DeprecatedRPC DepcrecatedRPCConfig `json:"deprecatedRPC"`
	WebsocketRPC  WebsocketRPCConfig   `json:"websocketRPC"`
}

// WebserverConfig stores the old webserver config
type WebserverConfig struct {
	Enabled                      bool   `json:"enabled"`
	AdminUsername                string `json:"adminUsername"`
	AdminPassword                string `json:"adminPassword"`
	ListenAddress                string `json:"listenAddress"`
	WebsocketConnectionLimit     int    `json:"websocketConnectionLimit"`
	WebsocketMaxAuthFailures     int    `json:"websocketMaxAuthFailures"`
	WebsocketAllowInsecureOrigin bool   `json:"websocketAllowInsecureOrigin"`
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

// BankAccount holds differing bank account details by supported funding
// currency
type BankAccount struct {
	Enabled             bool   `json:"enabled"`
	BankName            string `json:"bankName"`
	BankAddress         string `json:"bankAddress"`
	AccountName         string `json:"accountName"`
	AccountNumber       string `json:"accountNumber"`
	SWIFTCode           string `json:"swiftCode"`
	IBAN                string `json:"iban"`
	BSBNumber           string `json:"bsbNumber,omitempty"`
	SupportedCurrencies string `json:"supportedCurrencies"`
	SupportedExchanges  string `json:"supportedExchanges,omitempty"`
}

// BankTransaction defines a related banking transaction
type BankTransaction struct {
	ReferenceNumber     string `json:"referenceNumber"`
	TransactionNumber   string `json:"transactionNumber"`
	PaymentInstructions string `json:"paymentInstructions"`
}

// CurrencyConfig holds all the information needed for currency related manipulation
type CurrencyConfig struct {
	ForexProviders                []base.Settings           `json:"forexProviders"`
	CryptocurrencyProvider        CryptocurrencyProvider    `json:"cryptocurrencyProvider"`
	Cryptocurrencies              currency.Currencies       `json:"cryptocurrencies"`
	CurrencyPairFormat            *CurrencyPairFormatConfig `json:"currencyPairFormat"`
	FiatDisplayCurrency           currency.Code             `json:"fiatDisplayCurrency"`
	CurrencyFileUpdateDuration    time.Duration             `json:"currencyFileUpdateDuration"`
	ForeignExchangeUpdateDuration time.Duration             `json:"foreignExchangeUpdateDuration"`
}

// CryptocurrencyProvider defines coinmarketcap tools
type CryptocurrencyProvider struct {
	Name        string `json:"name"`
	Enabled     bool   `json:"enabled"`
	Verbose     bool   `json:"verbose"`
	APIkey      string `json:"apiKey"`
	AccountPlan string `json:"accountPlan"`
}

// CommunicationsConfig holds all the information needed for each
// enabled communication package
type CommunicationsConfig struct {
	SlackConfig     SlackConfig     `json:"slack"`
	SMSGlobalConfig SMSGlobalConfig `json:"smsGlobal"`
	SMTPConfig      SMTPConfig      `json:"smtp"`
	TelegramConfig  TelegramConfig  `json:"telegram"`
}

// SlackConfig holds all variables to start and run the Slack package
type SlackConfig struct {
	Name              string `json:"name"`
	Enabled           bool   `json:"enabled"`
	Verbose           bool   `json:"verbose"`
	TargetChannel     string `json:"targetChannel"`
	VerificationToken string `json:"verificationToken"`
}

// SMSContact stores the SMS contact info
type SMSContact struct {
	Name    string `json:"name"`
	Number  string `json:"number"`
	Enabled bool   `json:"enabled"`
}

// SMSGlobalConfig structure holds all the variables you need for instant
// messaging and broadcast used by SMSGlobal
type SMSGlobalConfig struct {
	Name     string       `json:"name"`
	Enabled  bool         `json:"enabled"`
	Verbose  bool         `json:"verbose"`
	Username string       `json:"username"`
	Password string       `json:"password"`
	Contacts []SMSContact `json:"contacts"`
}

// SMTPConfig holds all variables to start and run the SMTP package
type SMTPConfig struct {
	Name            string `json:"name"`
	Enabled         bool   `json:"enabled"`
	Verbose         bool   `json:"verbose"`
	Host            string `json:"host"`
	Port            string `json:"port"`
	AccountName     string `json:"accountName"`
	AccountPassword string `json:"accountPassword"`
	RecipientList   string `json:"recipientList"`
}

// TelegramConfig holds all variables to start and run the Telegram package
type TelegramConfig struct {
	Name              string `json:"name"`
	Enabled           bool   `json:"enabled"`
	Verbose           bool   `json:"verbose"`
	VerificationToken string `json:"verificationToken"`
}

// ProtocolFeaturesConfig holds all variables for the exchanges supported features
// for a protocol (e.g REST or Websocket)
type ProtocolFeaturesConfig struct {
	TickerBatching      bool   `json:"tickerBatching,omitempty"`
	AutoPairUpdates     bool   `json:"autoPairUpdates,omitempty"`
	AccountBalance      bool   `json:"accountBalance,omitempty"`
	CryptoDeposit       bool   `json:"cryptoDeposit,omitempty"`
	CryptoWithdrawal    uint32 `json:"cryptoWithdrawal,omitempty"`
	FiatWithdraw        bool   `json:"fiatWithdraw,omitempty"`
	GetOrder            bool   `json:"getOrder,omitempty"`
	GetOrders           bool   `json:"getOrders,omitempty"`
	CancelOrders        bool   `json:"cancelOrders,omitempty"`
	CancelOrder         bool   `json:"cancelOrder,omitempty"`
	SubmitOrder         bool   `json:"submitOrder,omitempty"`
	SubmitOrders        bool   `json:"submitOrders,omitempty"`
	ModifyOrder         bool   `json:"modifyOrder,omitempty"`
	DepositHistory      bool   `json:"depositHistory,omitempty"`
	WithdrawalHistory   bool   `json:"withdrawalHistory,omitempty"`
	TradeHistory        bool   `json:"tradeHistory,omitempty"`
	UserTradeHistory    bool   `json:"userTradeHistory,omitempty"`
	TradeFee            bool   `json:"tradeFee,omitempty"`
	FiatDepositFee      bool   `json:"fiatDepositFee,omitempty"`
	FiatWithdrawalFee   bool   `json:"fiatWithdrawalFee,omitempty"`
	CryptoDepositFee    bool   `json:"cryptoDepositFee,omitempty"`
	CryptoWithdrawalFee bool   `json:"cryptoWithdrawalFee,omitempty"`
}

// FeaturesSupportedConfig stores the exchanges supported features
type FeaturesSupportedConfig struct {
	REST                  bool                   `json:"restAPI"`
	RESTCapabilities      ProtocolFeaturesConfig `json:"restCapabilities,omitempty"`
	Websocket             bool                   `json:"websocketAPI"`
	WebsocketCapabilities ProtocolFeaturesConfig `json:"websocketCapabilities,omitempty"`
}

// FeaturesEnabledConfig stores the exchanges enabled features
type FeaturesEnabledConfig struct {
	AutoPairUpdates bool `json:"autoPairUpdates"`
	Websocket       bool `json:"websocketAPI"`
}

// FeaturesConfig stores the exchanges supported and enabled features
type FeaturesConfig struct {
	Supports FeaturesSupportedConfig `json:"supports"`
	Enabled  FeaturesEnabledConfig   `json:"enabled"`
}

// APIEndpointsConfig stores the API endpoint addresses
type APIEndpointsConfig struct {
	URL          string `json:"url"`
	URLSecondary string `json:"urlSecondary"`
	WebsocketURL string `json:"websocketURL"`
}

// APICredentialsConfig stores the API credentials
type APICredentialsConfig struct {
	Key       string `json:"key,omitempty"`
	Secret    string `json:"secret,omitempty"`
	ClientID  string `json:"clientID,omitempty"`
	PEMKey    string `json:"pemKey,omitempty"`
	OTPSecret string `json:"otpSecret,omitempty"`
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
	AuthenticatedSupport bool `json:"authenticatedSupport"`
	PEMKeySupport        bool `json:"pemKeySupport,omitempty"`

	Endpoints            APIEndpointsConfig            `json:"endpoints"`
	Credentials          APICredentialsConfig          `json:"credentials"`
	CredentialsValidator APICredentialsValidatorConfig `json:"credentialsValidator"`
}

// HTTPRateConfig stores the exchanges HTTP rate limiter config
type HTTPRateConfig struct {
	Duration time.Duration `json:"duration"`
	Rate     int           `json:"rate"`
}

// HTTPRateLimitConfig stores the rate limit config
type HTTPRateLimitConfig struct {
	Unauthenticated HTTPRateConfig `json:"unauthenticated"`
	Authenticated   HTTPRateConfig `json:"authenticated"`
}
