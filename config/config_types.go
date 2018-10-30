package config

import (
	"time"

	"github.com/thrasher-/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-/gocryptotrader/portfolio"
)

// Config is the overarching object that holds all the information for
// prestart management of Portfolio, Communications, Webserver and Enabled
// Exchanges
type Config struct {
	Name              string               `json:"name"`
	EncryptConfig     int                  `json:"encryptConfig"`
	GlobalHTTPTimeout time.Duration        `json:"globalHTTPTimeout"`
	Currency          CurrencyConfig       `json:"currencyConfig"`
	Communications    CommunicationsConfig `json:"communications"`
	Portfolio         portfolio.Base       `json:"portfolioAddresses"`
	RESTServer        RESTConfig           `json:"restServer"`
	WebsocketServer   WebsocketConfig      `json:"websocketServer"`
	Exchanges         []ExchangeConfig     `json:"exchanges"`
	BankAccounts      []BankAccount        `json:"bankAccounts"`

	// Deprecated config settings, will be removed at a future date
	Webserver           *Webserver                `json:"webserver,omitempty"`
	CurrencyPairFormat  *CurrencyPairFormatConfig `json:"currencyPairFormat,omitempty"`
	FiatDisplayCurrency string                    `json:"fiatDispayCurrency,omitempty"`
	Cryptocurrencies    string                    `json:"cryptocurrencies,omitempty"`
	SMS                 *SMSGlobalConfig          `json:"smsGlobal,omitempty"`
}

// ExchangeConfig holds all the information needed for each enabled Exchange.
type ExchangeConfig struct {
	Name                      string                    `json:"name"`
	Enabled                   bool                      `json:"enabled"`
	Verbose                   bool                      `json:"verbose"`
	Websocket                 bool                      `json:"websocket"`
	UseSandbox                bool                      `json:"useSandbox"`
	RESTPollingDelay          time.Duration             `json:"restPollingDelay"`
	HTTPTimeout               time.Duration             `json:"httpTimeout"`
	HTTPUserAgent             string                    `json:"httpUserAgent"`
	AuthenticatedAPISupport   bool                      `json:"authenticatedApiSupport"`
	APIKey                    string                    `json:"apiKey"`
	APISecret                 string                    `json:"apiSecret"`
	APIAuthPEMKeySupport      bool                      `json:"apiAuthPemKeySupport,omitempty"`
	APIAuthPEMKey             string                    `json:"apiAuthPemKey,omitempty"`
	APIURL                    string                    `json:"apiUrl"`
	APIURLSecondary           string                    `json:"apiUrlSecondary"`
	ProxyAddress              string                    `json:"proxyAddress"`
	WebsocketURL              string                    `json:"websocketUrl"`
	ClientID                  string                    `json:"clientId,omitempty"`
	AvailablePairs            string                    `json:"availablePairs"`
	EnabledPairs              string                    `json:"enabledPairs"`
	BaseCurrencies            string                    `json:"baseCurrencies"`
	AssetTypes                string                    `json:"assetTypes"`
	SupportsAutoPairUpdates   bool                      `json:"supportsAutoPairUpdates"`
	PairsLastUpdated          int64                     `json:"pairsLastUpdated,omitempty"`
	ConfigCurrencyPairFormat  *CurrencyPairFormatConfig `json:"configCurrencyPairFormat"`
	RequestCurrencyPairFormat *CurrencyPairFormatConfig `json:"requestCurrencyPairFormat"`
	BankAccounts              []BankAccount             `json:"bankAccounts"`
}

// RESTConfig struct holds the prestart variables for the webserver.
type RESTConfig struct {
	Enabled       bool   `json:"enabled"`
	AdminUsername string `json:"adminUsername"`
	AdminPassword string `json:"adminPassword"`
	ListenAddress string `json:"listenAddress"`
}

// WebsocketConfig struct holds the variables for the Websocket server.
type WebsocketConfig struct {
	RESTConfig
	WebsocketConnectionLimit     int  `json:"websocketConnectionLimit"`
	WebsocketMaxAuthFailures     int  `json:"websocketMaxAuthFailures"`
	WebsocketAllowInsecureOrigin bool `json:"websocketAllowInsecureOrigin"`
}

// Webserver stores the old webserver config
type Webserver WebsocketConfig

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
	Enabled             bool   `json:"enabled,omitempty"`
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
	ForexProviders      []base.Settings           `json:"forexProviders"`
	Cryptocurrencies    string                    `json:"cryptocurrencies"`
	CurrencyPairFormat  *CurrencyPairFormatConfig `json:"currencyPairFormat"`
	FiatDisplayCurrency string                    `json:"fiatDisplayCurrency"`
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
