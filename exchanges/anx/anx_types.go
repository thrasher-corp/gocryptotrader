package anx

import "github.com/thrasher-/gocryptotrader/currency/symbol"

// List of strings
const (
	CancelOrderNotFound    string = "ORDER_NOT_FOUND"
	CancelRequestSubmitted string = "CANCEL_REQUEST_SUBMITTED"
	CancelOrderWrongState  string = "ORDER_CANCEL_WRONG_STATE"
)

// Currency holds the currency information
type Currency struct {
	Decimals               int     `json:"decimals"`
	MinOrderSize           float64 `json:"minOrderSize"`
	MaxOrderSize           float64 `json:"maxOrderSize"`
	DisplayDenominator     float64 `json:"displayDenominator"`
	SummaryDecimals        int     `json:"summaryDecimals"`
	DisplayUnit            string  `json:"displayUnit"`
	Symbol                 string  `json:"symbol"`
	Type                   string  `json:"type"`
	ConfirmationThresholds []struct {
		ConfosRequired int `json:"confosRequired"`
	} `json:"confirmationThresholds"`
	NetworkFee     float64 `json:"networkFee"`
	EngineSettings struct {
		DepositsEnabled     bool `json:"depositsEnabled"`
		WithdrawalsEnabled  bool `json:"withdrawalsEnabled"`
		DisplayEnabled      bool `json:"displayEnabled"`
		MobileAccessEnabled bool `json:"mobileAccessEnabled"`
	} `json:"engineSettings"`
	MinOrderValue       float64 `json:"minOrderValue"`
	MaxOrderValue       float64 `json:"maxOrderValue"`
	MaxMarketOrderValue float64 `json:"maxMarketOrderValue"`
	MaxMarketOrderSize  float64 `json:"maxMarketOrderSize"`
	DigitalCurrencyType string  `json:"digitalCurrencyType"`
	AssetName           string  `json:"assetName"`
	AssetDivisibility   int     `json:"assetDivisibility"`
	AssetIcon           string  `json:"assetIcon"`
	AssetIssueQuantity  string  `json:"assetIssueQuantity"`
}

// Currencies stores a list of currencies
type Currencies map[string]Currency

// CurrencyPair holds the currency information
type CurrencyPair struct {
	PriceDecimals  int `json:"priceDecimals"`
	EngineSettings struct {
		TradingEnabled bool `json:"tradingEnabled"`
		DisplayEnabled bool `json:"displayEnabled"`
		CancelOnly     bool `json:"cancelOnly"`
		VerifyRequired bool `json:"verifyRequired"`
		RestrictedBuy  bool `json:"restrictedBuy"`
		RestrictedSell bool `json:"restrictedSell"`
	} `json:"engineSettings"`
	MinOrderRate         float64 `json:"minOrderRate"`
	MaxOrderRate         float64 `json:"maxOrderRate"`
	DisplayPriceDecimals int     `json:"displayPriceDecimals"`
	TradedCcy            string  `json:"tradedCcy"`
	SettlementCcy        string  `json:"settlementCcy"`
	PreferredMarket      string  `json:"preferredMarket"`
	ChartEnabled         bool    `json:"chartEnabled"`
	SimpleTradeEnabled   bool    `json:"simpleTradeEnabled"`
}

// AccountInformation Used by Get account information
// Retrieves details of the account and api's
type AccountInformation struct {
	UserUUID   string   `json:"userUuid"`
	Rights     []string `json:"Rights"`
	ResultCode string   `json:"resultCode"`
	Wallets    map[string]struct {
		Balance              Amount `json:"Balance"`
		AvailableBalance     Amount `json:"Available_Balance"`
		DailyWithdrawalLimit Amount `json:"Daily_Withdrawal_Limit"`
		MaxWithdraw          Amount `json:"Max_Withdraw"`
	} `json:"Wallets"`
}

// Amount basic storage of wallet details
type Amount struct {
	DisplayShort string  `json:"displayShort"`
	ValueInt     int64   `json:"valueInt"`
	Currency     string  `json:"currency"`
	Display      string  `json:"display"`
	Value        float64 `json:"value"`
}

// CurrencyPairs stores currency pair info
type CurrencyPairs map[string]CurrencyPair

// CurrenciesStore stores the available cryptocurrencies
// and fiat currencies
type CurrenciesStore struct {
	Currencies    Currencies    `json:"currencies"`
	CurrencyPairs CurrencyPairs `json:"currencyPairs"`
	Timestamp     string        `json:"timestamp"`
	ResultCode    string        `json:"resultCode"`
}

// CurrenciesStaticResponse stores the currencyStatic response
type CurrenciesStaticResponse struct {
	CurrenciesResponse CurrenciesStore `json:"CurrencyStatic"`
}

// Order holds order information
type Order struct {
	OrderType                      string  `json:"orderType"`
	BuyTradedCurrency              bool    `json:"buyTradedCurrency"`
	TradedCurrency                 string  `json:"tradedCurrency"`
	SettlementCurrency             string  `json:"settlementCurrency"`
	TradedCurrencyAmount           float64 `json:"tradedCurrencyAmount,string"`
	SettlementCurrencyAmount       float64 `json:"settlementCurrencyAmount,string"`
	LimitPriceInSettlementCurrency float64 `json:"limitPriceInSettlementCurrency,string"`
	ReplaceExistingOrderUUID       string  `json:"replaceExistingOrderUuid"`
	ReplaceOnlyIfActive            bool    `json:"replaceOnlyIfActive"`
}

// OrderResponse holds order response data
type OrderResponse struct {
	BuyTradedCurrency              bool    `json:"buyTradedCurrency"`
	ExecutedAverageRate            string  `json:"executedAverageRate"`
	LimitPriceInSettlementCurrency string  `json:"limitPriceInSettlementCurrency"`
	OrderID                        string  `json:"orderId"`
	OrderStatus                    string  `json:"orderStatus"`
	OrderType                      string  `json:"orderType"`
	ReplaceExistingOrderUUID       string  `json:"replaceExistingOrderId"`
	SettlementCurrency             string  `json:"settlementCurrency"`
	SettlementCurrencyAmount       float64 `json:"settlementCurrencyAmount,string"`
	SettlementCurrencyOutstanding  string  `json:"settlementCurrencyOutstanding"`
	Timestamp                      int64   `json:"timestamp"`
	TradedCurrency                 string  `json:"tradedCurrency"`
	TradedCurrencyAmount           float64 `json:"tradedCurrencyAmount,string"`
	TradedCurrencyOutstanding      string  `json:"tradedCurrencyOutstanding"`
}

// OrderCancelResponse returned when cancelling multiple orders
type OrderCancelResponse struct {
	OrderCancellationResponses []OrderCancellationResponse `json:"orderIds"`
	ResultCode                 string                      `json:"resultCode"`
	UUID                       int64                       `json:"uuid"`
	ErrorCode                  int64                       `json:"errorCode"`
}

// OrderCancellationResponse contains the orderID and error when cancelling multiple orders
type OrderCancellationResponse struct {
	UUID  string `json:"uuid"`
	Error string `json:"errorCode"`
}

// TickerComponent is a sub-type for ticker
type TickerComponent struct {
	Currency     string `json:"currency"`
	Display      string `json:"display"`
	DisplayShort string `json:"display_short"`
	Value        string `json:"value"`
}

// Ticker contains ticker data
type Ticker struct {
	Result string `json:"result"`
	Data   struct {
		High       TickerComponent `json:"high"`
		Low        TickerComponent `json:"low"`
		Avg        TickerComponent `json:"avg"`
		Vwap       TickerComponent `json:"vwap"`
		Vol        TickerComponent `json:"vol"`
		Last       TickerComponent `json:"last"`
		Buy        TickerComponent `json:"buy"`
		Sell       TickerComponent `json:"sell"`
		Now        string          `json:"now"`
		UpdateTime string          `json:"dataUpdateTime"`
	} `json:"data"`
}

// DepthItem contains depth information
type DepthItem struct {
	Price     float64 `json:"price,string"`
	PriceInt  float64 `json:"price_int,string"`
	Amount    float64 `json:"amount,string"`
	AmountInt int64   `json:"amount_int,string"`
}

// Depth contains full depth information
type Depth struct {
	Result string `json:"result"`
	Data   struct {
		Now            string      `json:"now"`
		DataUpdateTime string      `json:"dataUpdateTime"`
		Asks           []DepthItem `json:"asks"`
		Bids           []DepthItem `json:"bids"`
	} `json:"data"`
}

// WithdrawalFees the large list of predefined withdrawal fees
// Prone to change
var WithdrawalFees = map[string]float64{
	symbol.BTC:  0.002,
	symbol.DOGE: 0.1,
	symbol.ETH:  0.005,
	symbol.GNT:  0.001,
	symbol.LTC:  0.02,
	symbol.OAX:  0.001,
	symbol.XRP:  1,
	symbol.HKD:  0.01,
}
