package exchange

import (
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/currencystate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// Endpoint authentication types
const (
	// Repeated exchange strings
	// FeeType custom type for calculating fees based on method
	WireTransfer InternationalBankTransactionType = iota
	PerfectMoney
	Neteller
	AdvCash
	Payeer
	Skrill
	Simplex
	SEPA
	Swift
	RapidTransfer
	MisterTangoSEPA
	Qiwi
	VisaMastercard
	WebMoney
	Capitalist
	WesternUnion
	MoneyGram
	Contact
	// Const declarations for fee types
	BankFee FeeType = iota
	InternationalBankDepositFee
	InternationalBankWithdrawalFee
	CryptocurrencyTradeFee
	CryptocurrencyDepositFee
	CryptocurrencyWithdrawalFee
	OfflineTradeFee
	// Definitions for each type of withdrawal method for a given exchange
	NoAPIWithdrawalMethods                  uint32 = 0
	NoAPIWithdrawalMethodsText              string = "NONE, WEBSITE ONLY"
	AutoWithdrawCrypto                      uint32 = (1 << 0)
	AutoWithdrawCryptoWithAPIPermission     uint32 = (1 << 1)
	AutoWithdrawCryptoWithSetup             uint32 = (1 << 2)
	AutoWithdrawCryptoText                  string = "AUTO WITHDRAW CRYPTO"
	AutoWithdrawCryptoWithAPIPermissionText string = "AUTO WITHDRAW CRYPTO WITH API PERMISSION"
	AutoWithdrawCryptoWithSetupText         string = "AUTO WITHDRAW CRYPTO WITH SETUP"
	WithdrawCryptoWith2FA                   uint32 = (1 << 3)
	WithdrawCryptoWithSMS                   uint32 = (1 << 4)
	WithdrawCryptoWithEmail                 uint32 = (1 << 5)
	WithdrawCryptoWithWebsiteApproval       uint32 = (1 << 6)
	WithdrawCryptoWithAPIPermission         uint32 = (1 << 7)
	WithdrawCryptoWith2FAText               string = "WITHDRAW CRYPTO WITH 2FA"
	WithdrawCryptoWithSMSText               string = "WITHDRAW CRYPTO WITH SMS"
	WithdrawCryptoWithEmailText             string = "WITHDRAW CRYPTO WITH EMAIL"
	WithdrawCryptoWithWebsiteApprovalText   string = "WITHDRAW CRYPTO WITH WEBSITE APPROVAL"
	WithdrawCryptoWithAPIPermissionText     string = "WITHDRAW CRYPTO WITH API PERMISSION"
	AutoWithdrawFiat                        uint32 = (1 << 8)
	AutoWithdrawFiatWithAPIPermission       uint32 = (1 << 9)
	AutoWithdrawFiatWithSetup               uint32 = (1 << 10)
	AutoWithdrawFiatText                    string = "AUTO WITHDRAW FIAT"
	AutoWithdrawFiatWithAPIPermissionText   string = "AUTO WITHDRAW FIAT WITH API PERMISSION"
	AutoWithdrawFiatWithSetupText           string = "AUTO WITHDRAW FIAT WITH SETUP"
	WithdrawFiatWith2FA                     uint32 = (1 << 11)
	WithdrawFiatWithSMS                     uint32 = (1 << 12)
	WithdrawFiatWithEmail                   uint32 = (1 << 13)
	WithdrawFiatWithWebsiteApproval         uint32 = (1 << 14)
	WithdrawFiatWithAPIPermission           uint32 = (1 << 15)
	WithdrawFiatWith2FAText                 string = "WITHDRAW FIAT WITH 2FA"
	WithdrawFiatWithSMSText                 string = "WITHDRAW FIAT WITH SMS"
	WithdrawFiatWithEmailText               string = "WITHDRAW FIAT WITH EMAIL"
	WithdrawFiatWithWebsiteApprovalText     string = "WITHDRAW FIAT WITH WEBSITE APPROVAL"
	WithdrawFiatWithAPIPermissionText       string = "WITHDRAW FIAT WITH API PERMISSION"
	WithdrawCryptoViaWebsiteOnly            uint32 = (1 << 16)
	WithdrawFiatViaWebsiteOnly              uint32 = (1 << 17)
	WithdrawCryptoViaWebsiteOnlyText        string = "WITHDRAW CRYPTO VIA WEBSITE ONLY"
	WithdrawFiatViaWebsiteOnlyText          string = "WITHDRAW FIAT VIA WEBSITE ONLY"
	NoFiatWithdrawals                       uint32 = (1 << 18)
	NoFiatWithdrawalsText                   string = "NO FIAT WITHDRAWAL"
	UnknownWithdrawalTypeText               string = "UNKNOWN"
)

// FeeType is the type for holding a custom fee type (International withdrawal fee)
type FeeType uint8

// InternationalBankTransactionType custom type for calculating fees based on fiat transaction types
type InternationalBankTransactionType uint8

// FeeBuilder is the type which holds all parameters required to calculate a fee
// for an exchange
type FeeBuilder struct {
	FeeType FeeType
	// Used for calculating crypto trading fees, deposits & withdrawals
	Pair    currency.Pair
	IsMaker bool
	// Fiat currency used for bank deposits & withdrawals
	FiatCurrency        currency.Code
	BankTransactionType InternationalBankTransactionType
	// Used to multiply for fee calculations
	PurchasePrice float64
	Amount        float64
}

// FundingHistory holds exchange funding history data
type FundingHistory struct {
	ExchangeName      string
	Status            string
	TransferID        string
	Description       string
	Timestamp         time.Time
	Currency          string
	Amount            float64
	Fee               float64
	TransferType      string
	CryptoToAddress   string
	CryptoFromAddress string
	CryptoTxID        string
	CryptoChain       string
	BankTo            string
	BankFrom          string
}

// WithdrawalHistory holds exchange Withdrawal history data
type WithdrawalHistory struct {
	Status          string
	TransferID      string
	Description     string
	Timestamp       time.Time
	Currency        string
	Amount          float64
	Fee             float64
	TransferType    string
	CryptoToAddress string
	CryptoTxID      string
	CryptoChain     string
	BankTo          string
}

// Features stores the supported and enabled features
// for the exchange
type Features struct {
	Supports             FeaturesSupported
	Enabled              FeaturesEnabled
	Subscriptions        subscription.List
	CurrencyTranslations currency.Translations
	TradingRequirements  protocol.TradingRequirements
}

// FeaturesEnabled stores the exchange enabled features
type FeaturesEnabled struct {
	AutoPairUpdates bool
	Kline           kline.ExchangeCapabilitiesEnabled
	SaveTradeData   bool
	TradeFeed       bool
	FillsFeed       bool
}

// FeaturesSupported stores the exchanges supported features
type FeaturesSupported struct {
	REST                       bool
	RESTCapabilities           protocol.Features
	Websocket                  bool
	WebsocketCapabilities      protocol.Features
	WithdrawPermissions        uint32
	Kline                      kline.ExchangeCapabilitiesSupported
	MaximumOrderHistory        time.Duration
	FuturesCapabilities        FuturesCapabilities
	OfflineFuturesCapabilities FuturesCapabilities
}

// FuturesCapabilities stores the exchange's futures capabilities
type FuturesCapabilities struct {
	FundingRates                    bool
	MaximumFundingRateHistory       time.Duration
	FundingRateBatching             map[asset.Item]bool
	SupportedFundingRateFrequencies map[kline.Interval]bool
	Positions                       bool
	OrderManagerPositionTracking    bool
	Collateral                      bool
	CollateralMode                  bool
	Leverage                        bool
	OpenInterest                    OpenInterestSupport
}

// OpenInterestSupport helps breakdown a feature and how it is supported
type OpenInterestSupport struct {
	Supported          bool
	SupportedViaTicker bool
	SupportsRestBatch  bool
}

// MarginCapabilities stores the exchange's margin capabilities
type MarginCapabilities struct {
	SetMarginType        bool
	ChangePositionMargin bool
	GetMarginRateHistory bool
}

// Endpoints stores running url endpoints for exchanges
type Endpoints struct {
	Exchange string
	defaults map[string]string
	mu       sync.RWMutex
}

// API stores the exchange API settings
type API struct {
	AuthenticatedSupport          bool
	AuthenticatedWebsocketSupport bool
	PEMKeySupport                 bool

	Endpoints *Endpoints

	credentials accounts.Credentials
	credMu      sync.RWMutex

	CredentialsValidator config.APICredentialsValidatorConfig
}

// Base stores the individual exchange information
type Base struct {
	Name                          string
	Enabled                       bool
	Verbose                       bool
	LoadedByConfig                bool
	SkipAuthCheck                 bool
	API                           API
	BaseCurrencies                currency.Currencies
	CurrencyPairs                 currency.PairsManager
	Features                      Features
	HTTPTimeout                   time.Duration
	HTTPRecording                 bool
	HTTPMockDataSliceLimit        int // Use with HTTPRecording to reduce the size of recorded mock data
	HTTPDebugging                 bool
	BypassConfigFormatUpgrades    bool
	WebsocketResponseCheckTimeout time.Duration
	WebsocketResponseMaxLimit     time.Duration
	WebsocketOrderbookBufferLimit int64
	Websocket                     *websocket.Manager
	Accounts                      *accounts.Accounts
	*request.Requester
	Config        *config.Exchange
	settingsMutex sync.RWMutex
	// ValidateOrderbook determines if the orderbook verification can be bypassed,
	// increasing potential update speed but decreasing confidence in orderbook
	// integrity.
	ValidateOrderbook bool

	AssetWebsocketSupport
	*currencystate.States
	messageSequence common.Counter
}

// url lookup consts
const (
	Invalid URL = iota
	RestSpot
	RestSpotSupplementary
	RestUSDTMargined
	RestCoinMargined
	RestFutures
	RestFuturesSupplementary
	RestUSDCMargined
	RestSwap
	RestSandbox
	WebsocketSpot
	WebsocketCoinMargined
	WebsocketUSDTMargined
	WebsocketUSDCMargined
	WebsocketOptions
	WebsocketTrade
	WebsocketPrivate
	WebsocketSpotSupplementary
	ChainAnalysis
	EdgeCase1
	EdgeCase2
	EdgeCase3

	restSpotURL                   = "RestSpotURL"
	restSpotSupplementaryURL      = "RestSpotSupplementaryURL"
	restUSDTMarginedFuturesURL    = "RestUSDTMarginedFuturesURL"
	restCoinMarginedFuturesURL    = "RestCoinMarginedFuturesURL"
	restUSDCMarginedFuturesURL    = "RestUSDCMarginedFuturesURL"
	restFuturesURL                = "RestFuturesURL"
	restFuturesSupplementaryURL   = "RestFuturesSupplementaryURL"
	restSandboxURL                = "RestSandboxURL"
	restSwapURL                   = "RestSwapURL"
	websocketSpotURL              = "WebsocketSpotURL"
	websocketCoinMarginedURL      = "WebsocketCoinMarginedURL"
	websocketUSDTMarginedURL      = "WebsocketUSDTMarginedURL"
	websocketUSDCMarginedURL      = "WebsocketUSDCMarginedURL"
	websocketOptionsURL           = "WebsocketOptionsURL"
	websocketTradeURL             = "WebsocketTradeURL"
	websocketPrivateURL           = "WebsocketPrivateURL"
	websocketSpotSupplementaryURL = "WebsocketSpotSupplementaryURL"
	chainAnalysisURL              = "ChainAnalysisURL"
	edgeCase1URL                  = "EdgeCase1URL"
	edgeCase2URL                  = "EdgeCase2URL"
	edgeCase3URL                  = "EdgeCase3URL"
)

var keyURLs = []URL{
	RestSpot,
	RestSpotSupplementary,
	RestUSDTMargined,
	RestCoinMargined,
	RestFutures,
	RestFuturesSupplementary,
	RestUSDCMargined,
	RestSwap,
	RestSandbox,
	WebsocketSpot,
	WebsocketCoinMargined,
	WebsocketUSDTMargined,
	WebsocketUSDCMargined,
	WebsocketOptions,
	WebsocketTrade,
	WebsocketPrivate,
	WebsocketSpotSupplementary,
	ChainAnalysis,
	EdgeCase1,
	EdgeCase2,
	EdgeCase3,
}

// URL stores uint conversions
type URL uint16

// AssetWebsocketSupport defines the availability of websocket functionality to
// the specific asset type. TODO: Deprecate as this is a temp item to address
// certain limitations quickly.
type AssetWebsocketSupport struct {
	unsupported map[asset.Item]bool
	m           sync.RWMutex
}
