package exchange

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
)

// FeeType custom type for calculating fees based on method
type FeeType uint8

// Const declarations for fee types
const (
	BankFee FeeType = iota
	InternationalBankDepositFee
	InternationalBankWithdrawalFee
	CryptocurrencyTradeFee
	CyptocurrencyDepositFee
	CryptocurrencyWithdrawalFee
	OfflineTradeFee
)

// InternationalBankTransactionType custom type for calculating fees based on fiat transaction types
type InternationalBankTransactionType uint8

// Const declarations for international transaction types
const (
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
)

// SubmitOrderResponse is what is returned after submitting an order to an exchange
type SubmitOrderResponse struct {
	IsOrderPlaced bool
	OrderID       string
}

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
	NoFiatWithdrawals                       uint32 = (1 << 18)
	NoFiatWithdrawalsText                   string = "NO FIAT WITHDRAWAL"

	UnknownWithdrawalTypeText string = "UNKNOWN"
)

// ModifyOrder is a an order modifyer
// ModifyOrder is a an order modifyer
type ModifyOrder struct {
	OrderID string
	OrderType
	OrderSide
	Price           float64
	Amount          float64
	LimitPriceUpper float64
	LimitPriceLower float64
	CurrencyPair    currency.Pair

	ImmediateOrCancel bool
	HiddenOrder       bool
	FillOrKill        bool
	PostOnly          bool
}

// ModifyOrderResponse is an order modifying return type
type ModifyOrderResponse struct {
	OrderID string
}

// CancelAllOrdersResponse returns the status from attempting to cancel all orders on an exchagne
type CancelAllOrdersResponse struct {
	OrderStatus map[string]string
}

// OrderType enforces a standard for Ordertypes across the code base
type OrderType string

// OrderType ...types
const (
	AnyOrderType               OrderType = "ANY"
	LimitOrderType             OrderType = "LIMIT"
	MarketOrderType            OrderType = "MARKET"
	ImmediateOrCancelOrderType OrderType = "IMMEDIATE_OR_CANCEL"
	StopOrderType              OrderType = "STOP"
	TrailingStopOrderType      OrderType = "TRAILINGSTOP"
	UnknownOrderType           OrderType = "UNKNOWN"
)

// ToLower changes the ordertype to lower case
func (o OrderType) ToLower() OrderType {
	return OrderType(common.StringToLower(string(o)))
}

// ToString changes the ordertype to the exchange standard and returns a string
func (o OrderType) ToString() string {
	return fmt.Sprintf("%v", o)
}

// OrderSide enforces a standard for OrderSides across the code base
type OrderSide string

// OrderSide types
const (
	AnyOrderSide  OrderSide = "ANY"
	BuyOrderSide  OrderSide = "BUY"
	SellOrderSide OrderSide = "SELL"
	BidOrderSide  OrderSide = "BID"
	AskOrderSide  OrderSide = "ASK"
)

// ToLower changes the ordertype to lower case
func (o OrderSide) ToLower() OrderSide {
	return OrderSide(common.StringToLower(string(o)))
}

// ToString changes the ordertype to the exchange standard and returns a string
func (o OrderSide) ToString() string {
	return fmt.Sprintf("%v", o)
}

// AccountInfo is a Generic type to hold each exchange's holdings in
// all enabled currencies
type AccountInfo struct {
	Exchange string
	Accounts []Account
}

// Account defines a singular account type with asocciated currencies
type Account struct {
	ID         string
	Currencies []AccountCurrencyInfo
}

// AccountCurrencyInfo is a sub type to store currency name and value
type AccountCurrencyInfo struct {
	CurrencyName currency.Code
	TotalValue   float64
	Hold         float64
}

// TradeHistory holds exchange history data
type TradeHistory struct {
	Timestamp   time.Time
	TID         int64
	Price       float64
	Amount      float64
	Exchange    string
	Type        string
	Fee         float64
	Description string
}

// OrderDetail holds order detail data
type OrderDetail struct {
	Exchange        string
	AccountID       string
	ID              string
	CurrencyPair    currency.Pair
	OrderSide       OrderSide
	OrderType       OrderType
	OrderDate       time.Time
	Status          string
	Price           float64
	Amount          float64
	ExecutedAmount  float64
	RemainingAmount float64
	Fee             float64
	Trades          []TradeHistory
}

// OrderCancellation type required when requesting to cancel an order
type OrderCancellation struct {
	AccountID     string
	OrderID       string
	CurrencyPair  currency.Pair
	AssetType     assets.AssetType
	WalletAddress string
	Side          OrderSide
}

// FundHistory holds exchange funding history data
type FundHistory struct {
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
	BankTo            string
	BankFrom          string
}

// WithdrawRequest used for wrapper crypto and FIAT withdraw methods
type WithdrawRequest struct {
	// General withdraw information
	Description     string
	OneTimePassword int64
	AccountID       string
	PIN             int64
	TradePassword   string
	Amount          float64
	Currency        currency.Code
	// Crypto related information
	Address    string
	AddressTag string
	FeeAmount  float64
	// FIAT related information
	BankAccountName   string
	BankAccountNumber float64
	BankName          string
	BankAddress       string
	BankCity          string
	BankCountry       string
	BankPostalCode    string
	SwiftCode         string
	IBAN              string
	BankCode          float64
	IsExpressWire     bool
	// Intermediary bank information
	RequiresIntermediaryBank      bool
	IntermediaryBankAccountNumber float64
	IntermediaryBankName          string
	IntermediaryBankAddress       string
	IntermediaryBankCity          string
	IntermediaryBankCountry       string
	IntermediaryBankPostalCode    string
	IntermediarySwiftCode         string
	IntermediaryBankCode          float64
	IntermediaryIBAN              string
	WireCurrency                  string
}

// Features stores the supported and enabled features
// for the exchange
type Features struct {
	Supports FeaturesSupported
	Enabled  FeaturesEnabled
}

// FeaturesEnabled stores the exchange enabled features
type FeaturesEnabled struct {
	AutoPairUpdates bool
}

// ProtocolFeatures holds all variables for the exchanges supported features
// for a protocol (e.g REST or Websocket)
type ProtocolFeatures struct {
	TickerBatching      bool
	TickerFetching      bool
	OrderbookFetching   bool
	AutoPairUpdates     bool
	AccountInfo         bool
	CryptoDeposit       bool
	CryptoWithdrawal    uint32
	FiatWithdraw        bool
	GetOrder            bool
	GetOrders           bool
	CancelOrders        bool
	CancelOrder         bool
	SubmitOrder         bool
	SubmitOrders        bool
	ModifyOrder         bool
	DepositHistory      bool
	WithdrawalHistory   bool
	TradeHistory        bool
	UserTradeHistory    bool
	TradeFee            bool
	FiatDepositFee      bool
	FiatWithdrawalFee   bool
	CryptoDepositFee    bool
	CryptoWithdrawalFee bool
}

// FeaturesSupported stores the exchanges supported features
type FeaturesSupported struct {
	REST                  bool
	RESTCapabilities      ProtocolFeatures
	Websocket             bool
	WebsocketCapabilities ProtocolFeatures
}

// API stores the exchange API settings
type API struct {
	AuthenticatedSupport bool
	PEMKeySupport        bool

	Endpoints struct {
		URL                 string
		URLDefault          string
		URLSecondary        string
		URLSecondaryDefault string
		WebsocketURL        string
	}

	Credentials struct {
		Key      string
		Secret   string
		ClientID string
		PEMKey   string
	}

	CredentialsValidator struct {
		// For Huobi (optional)
		RequiresPEM bool

		RequiresKey                bool
		RequiresSecret             bool
		RequiresClientID           bool
		RequiresBase64DecodeSecret bool
	}
}

// CurrencyPair stores a list of enable/available
// currency pairs and their storage/request format
type CurrencyPair struct {
	Enabled       currency.Pairs
	Available     currency.Pairs
	RequestFormat config.CurrencyPairFormatConfig
	ConfigFormat  config.CurrencyPairFormatConfig
}

// CurrencyPairs stores a list of tradable currency pair settings
type CurrencyPairs struct {
	RequestFormat       config.CurrencyPairFormatConfig
	ConfigFormat        config.CurrencyPairFormatConfig
	UseGlobalPairFormat bool
	LastUpdated         int64
	Pairs               map[assets.AssetType]CurrencyPair
	AssetTypes          assets.AssetTypes
}

// GetOrdersRequest used for GetOrderHistory and GetOpenOrders wrapper functions
type GetOrdersRequest struct {
	OrderType  OrderType
	OrderSide  OrderSide
	StartTicks time.Time
	EndTicks   time.Time
	// Currencies Empty array = all currencies. Some endpoints only support singular currency enquiries
	Currencies []currency.Pair
}

// OrderStatus defines order status types
type OrderStatus string

// All OrderStatus types
const (
	AnyOrderStatus             OrderStatus = "ANY"
	NewOrderStatus             OrderStatus = "NEW"
	ActiveOrderStatus          OrderStatus = "ACTIVE"
	PartiallyFilledOrderStatus OrderStatus = "PARTIALLY_FILLED"
	FilledOrderStatus          OrderStatus = "FILLED"
	CancelledOrderStatus       OrderStatus = "CANCELED"
	PendingCancelOrderStatus   OrderStatus = "PENDING_CANCEL"
	RejectedOrderStatus        OrderStatus = "REJECTED"
	ExpiredOrderStatus         OrderStatus = "EXPIRED"
	HiddenOrderStatus          OrderStatus = "HIDDEN"
	UnknownOrderStatus         OrderStatus = "UNKNOWN"
)

// FilterOrdersBySide removes any OrderDetails that don't match the orderStatus provided
func FilterOrdersBySide(orders *[]OrderDetail, orderSide OrderSide) {
	if orderSide == "" || orderSide == AnyOrderSide {
		return
	}

	var filteredOrders []OrderDetail
	for i := range *orders {
		if strings.EqualFold(string((*orders)[i].OrderSide), string(orderSide)) {
			filteredOrders = append(filteredOrders, (*orders)[i])
		}
	}

	*orders = filteredOrders
}

// FilterOrdersByType removes any OrderDetails that don't match the orderType provided
func FilterOrdersByType(orders *[]OrderDetail, orderType OrderType) {
	if orderType == "" || orderType == AnyOrderType {
		return
	}

	var filteredOrders []OrderDetail
	for i := range *orders {
		if strings.EqualFold(string((*orders)[i].OrderType), string(orderType)) {
			filteredOrders = append(filteredOrders, (*orders)[i])
		}
	}

	*orders = filteredOrders
}

// FilterOrdersByTickRange removes any OrderDetails outside of the tick range
func FilterOrdersByTickRange(orders *[]OrderDetail, startTicks, endTicks time.Time) {
	if startTicks.IsZero() || endTicks.IsZero() ||
		startTicks.Unix() == 0 || endTicks.Unix() == 0 || endTicks.Before(startTicks) {
		return
	}

	var filteredOrders []OrderDetail
	for i := range *orders {
		if (*orders)[i].OrderDate.Unix() >= startTicks.Unix() && (*orders)[i].OrderDate.Unix() <= endTicks.Unix() {
			filteredOrders = append(filteredOrders, (*orders)[i])
		}
	}

	*orders = filteredOrders
}

// FilterOrdersByCurrencies removes any OrderDetails that do not match the provided currency list
// It is forgiving in that the provided currencies can match quote or base currencies
func FilterOrdersByCurrencies(orders *[]OrderDetail, currencies []currency.Pair) {
	if len(currencies) == 0 {
		return
	}

	var filteredOrders []OrderDetail
	for i := range *orders {
		matchFound := false
		for _, c := range currencies {
			if !matchFound && (*orders)[i].CurrencyPair.EqualIncludeReciprocal(c) {
				matchFound = true
			}
		}

		if matchFound {
			filteredOrders = append(filteredOrders, (*orders)[i])
		}
	}

	*orders = filteredOrders
}

// ByPrice used for sorting orders by price
type ByPrice []OrderDetail

func (b ByPrice) Len() int {
	return len(b)
}

func (b ByPrice) Less(i, j int) bool {
	return b[i].Price < b[j].Price
}

func (b ByPrice) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// SortOrdersByPrice the caller function to sort orders
func SortOrdersByPrice(orders *[]OrderDetail, reverse bool) {
	if reverse {
		sort.Sort(sort.Reverse(ByPrice(*orders)))
	} else {
		sort.Sort(ByPrice(*orders))
	}
}

// ByOrderType used for sorting orders by order type
type ByOrderType []OrderDetail

func (b ByOrderType) Len() int {
	return len(b)
}

func (b ByOrderType) Less(i, j int) bool {
	return b[i].OrderType.ToString() < b[j].OrderType.ToString()
}

func (b ByOrderType) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// SortOrdersByType the caller function to sort orders
func SortOrdersByType(orders *[]OrderDetail, reverse bool) {
	if reverse {
		sort.Sort(sort.Reverse(ByOrderType(*orders)))
	} else {
		sort.Sort(ByOrderType(*orders))
	}
}

// ByCurrency used for sorting orders by order currency
type ByCurrency []OrderDetail

func (b ByCurrency) Len() int {
	return len(b)
}

func (b ByCurrency) Less(i, j int) bool {
	return b[i].CurrencyPair.String() < b[j].CurrencyPair.String()
}

func (b ByCurrency) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// SortOrdersByCurrency the caller function to sort orders
func SortOrdersByCurrency(orders *[]OrderDetail, reverse bool) {
	if reverse {
		sort.Sort(sort.Reverse(ByCurrency(*orders)))
	} else {
		sort.Sort(ByCurrency(*orders))
	}
}

// ByDate used for sorting orders by order date
type ByDate []OrderDetail

func (b ByDate) Len() int {
	return len(b)
}

func (b ByDate) Less(i, j int) bool {
	return b[i].OrderDate.Unix() < b[j].OrderDate.Unix()
}

func (b ByDate) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// SortOrdersByDate the caller function to sort orders
func SortOrdersByDate(orders *[]OrderDetail, reverse bool) {
	if reverse {
		sort.Sort(sort.Reverse(ByDate(*orders)))
	} else {
		sort.Sort(ByDate(*orders))
	}
}

// ByOrderSide used for sorting orders by order side (buy sell)
type ByOrderSide []OrderDetail

func (b ByOrderSide) Len() int {
	return len(b)
}

func (b ByOrderSide) Less(i, j int) bool {
	return b[i].OrderSide.ToString() < b[j].OrderSide.ToString()
}

func (b ByOrderSide) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// SortOrdersBySide the caller function to sort orders
func SortOrdersBySide(orders *[]OrderDetail, reverse bool) {
	if reverse {
		sort.Sort(sort.Reverse(ByOrderSide(*orders)))
	} else {
		sort.Sort(ByOrderSide(*orders))
	}
}

// Base stores the individual exchange information
type Base struct {
	Name    string
	Enabled bool
	Verbose bool

	APIWithdrawPermissions uint32
	API                    API
	BaseCurrencies         currency.Currencies
	CurrencyPairs          currency.PairsManager

	Features      Features
	HTTPTimeout   time.Duration
	HTTPUserAgent string
	Websocket     *Websocket
	*request.Requester

	LoadedByConfig bool
	Config         *config.ExchangeConfig
}
