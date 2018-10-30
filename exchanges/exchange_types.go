package exchange

import (
	"time"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges/nonce"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
)

// FeeType custom type for calculating fees based on method
type FeeType string

// InternationalBankTransactionType custom type for calculating fees based on fiat transaction types
type InternationalBankTransactionType string

// Const declarations for fee types
const (
	BankFee                        FeeType = "bankFee"
	InternationalBankDepositFee    FeeType = "internationalBankDepositFee"
	InternationalBankWithdrawalFee FeeType = "internationalBankWithdrawalFee"
	CryptocurrencyTradeFee         FeeType = "cryptocurrencyTradeFee"
	CyptocurrencyDepositFee        FeeType = "cyptocurrencyDepositFee"
	CryptocurrencyWithdrawalFee    FeeType = "cryptocurrencyWithdrawalFee"
)

// Const declarations for international transaction types
const (
	WireTransfer    InternationalBankTransactionType = "wireTransfer"
	PerfectMoney    InternationalBankTransactionType = "perfectMoney"
	Neteller        InternationalBankTransactionType = "neteller"
	AdvCash         InternationalBankTransactionType = "advCash"
	Payeer          InternationalBankTransactionType = "payeer"
	Skrill          InternationalBankTransactionType = "skrill"
	Simplex         InternationalBankTransactionType = "simplex"
	SEPA            InternationalBankTransactionType = "sepa"
	Swift           InternationalBankTransactionType = "swift"
	RapidTransfer   InternationalBankTransactionType = "rapidTransfer"
	MisterTangoSEPA InternationalBankTransactionType = "misterTangoSepa"
	Qiwi            InternationalBankTransactionType = "qiwi"
	VisaMastercard  InternationalBankTransactionType = "visaMastercard"
	WebMoney        InternationalBankTransactionType = "webMoney"
	Capitalist      InternationalBankTransactionType = "capitalist"
	WesternUnion    InternationalBankTransactionType = "westernUnion"
	MoneyGram       InternationalBankTransactionType = "moneyGram"
	Contact         InternationalBankTransactionType = "contact"
)

// FeeBuilder is the type which holds all parameters required to calculate a fee for an exchange
type FeeBuilder struct {
	FeeType FeeType
	//Used for calculating crypto trading fees, deposits & withdrawals
	FirstCurrency  string
	SecondCurrency string
	Delimiter      string
	IsMaker        bool
	// Fiat currency used for bank deposits & withdrawals
	CurrencyItem        string
	BankTransactionType InternationalBankTransactionType
	// Used to multiply for fee calculations
	PurchasePrice float64
	Amount        float64
}

// Definitions for each type of withdrawal method for a given exchange
const (
	// No withdraw
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

	UnknownWithdrawalTypeText string = "UNKNOWN"
)

// AccountInfo is a Generic type to hold each exchange's holdings in
// all enabled currencies
type AccountInfo struct {
	ExchangeName string
	Currencies   []AccountCurrencyInfo
}

// AccountCurrencyInfo is a sub type to store currency name and value
type AccountCurrencyInfo struct {
	CurrencyName string
	TotalValue   float64
	Hold         float64
}

// TradeHistory holds exchange history data
type TradeHistory struct {
	Timestamp int64
	TID       int64
	Price     float64
	Amount    float64
	Exchange  string
	Type      string
}

// OrderDetail holds order detail data
type OrderDetail struct {
	Exchange      string
	ID            int64
	BaseCurrency  string
	QuoteCurrency string
	OrderSide     string
	OrderType     string
	CreationTime  int64
	Status        string
	Price         float64
	Amount        float64
	OpenVolume    float64
}

// FundHistory holds exchange funding history data
type FundHistory struct {
	ExchangeName      string
	Status            string
	TransferID        int64
	Description       string
	Timestamp         int64
	Currency          string
	Amount            float64
	Fee               float64
	TransferType      string
	CryptoToAddress   string
	CryptoFromAddress string
	CryptoTxID        string
	BankTo            string
	BankFrom          string
}

// Base stores the individual exchange information
type Base struct {
	Name                                       string
	Enabled                                    bool
	Verbose                                    bool
	RESTPollingDelay                           time.Duration
	AuthenticatedAPISupport                    bool
	APIWithdrawPermissions                     uint32
	APIAuthPEMKeySupport                       bool
	APISecret, APIKey, APIAuthPEMKey, ClientID string
	Nonce                                      nonce.Nonce
	TakerFee, MakerFee, Fee                    float64
	BaseCurrencies                             []string
	AvailablePairs                             []string
	EnabledPairs                               []string
	AssetTypes                                 []string
	PairsLastUpdated                           int64
	SupportsAutoPairUpdating                   bool
	SupportsRESTTickerBatching                 bool
	SupportsWebsocketAPI                       bool
	SupportsRESTAPI                            bool
	HTTPTimeout                                time.Duration
	HTTPUserAgent                              string
	WebsocketURL                               string
	APIUrl                                     string
	APIUrlDefault                              string
	APIUrlSecondary                            string
	APIUrlSecondaryDefault                     string
	RequestCurrencyPairFormat                  config.CurrencyPairFormatConfig
	ConfigCurrencyPairFormat                   config.CurrencyPairFormatConfig
	Websocket                                  *Websocket
	*request.Requester
}
