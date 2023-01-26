package cryptodotcom

import (
	"encoding/json"
	"errors"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
)

var (
	errSymbolIsRequired              = errors.New("symbol is required")
	errInvalidOrderCancellationScope = errors.New("invalid order cancellation scope, only ACCOUNT or CONNECTION is supported")
	errInvalidCurrency               = errors.New("invalid currency")
	errInvalidAmount                 = errors.New("amount has to be greater than zero")
	errNoArgumentPassed              = errors.New("no argument passed")
	errInvalidResponseFromServer     = errors.New("invalid response from server")
)

// InstrumentsResponse represents instruments response.
type InstrumentsResponse struct {
	ID     int    `json:"id"`
	Method string `json:"method"`
	Code   int    `json:"code"`
	Result struct {
		Instruments []Instrument `json:"instruments"`
	} `json:"result"`
}

// Instrument represents an details.
type Instrument struct {
	InstrumentName          string  `json:"instrument_name"`
	QuoteCurrency           string  `json:"quote_currency"`
	BaseCurrency            string  `json:"base_currency"`
	PriceDecimals           int     `json:"price_decimals"`
	QuantityDecimals        int     `json:"quantity_decimals"`
	MarginTradingEnabled    bool    `json:"margin_trading_enabled"`
	MarginTradingEnabled5X  bool    `json:"margin_trading_enabled_5x"`
	MarginTradingEnabled10X bool    `json:"margin_trading_enabled_10x"`
	MaxQuantity             string  `json:"max_quantity"`
	MinQuantity             string  `json:"min_quantity"`
	MaxPrice                float64 `json:"max_price,string"`
	MinPrice                float64 `json:"min_price,string"`
	LastUpdateDate          int64   `json:"last_update_date"`
	QuantityTickSize        float64 `json:"quantity_tick_size,string"`
	PriceTickSize           float64 `json:"price_tick_size,string"`
}

// OrderbookDetail public order book detail.
type OrderbookDetail struct {
	Depth int `json:"depth"`
	Data  []struct {
		Asks [][3]string `json:"asks"`
		Bids [][3]string `json:"bids"`
	} `json:"data"`
	InstrumentName string `json:"instrument_name"`
}

// CandlestickDetail candlesticks (k-line data history).
type CandlestickDetail struct {
	InstrumentName string            `json:"instrument_name"`
	Interval       string            `json:"interval"`
	Data           []CandlestickItem `json:"data"`
}

// CandlestickItem candlesticks (k-line data history) item.
type CandlestickItem struct {
	EndTime cryptoDotComMilliSec `json:"t"` // this represents Start Time for websocket push data.
	Open    float64              `json:"o,string"`
	High    float64              `json:"h,string"`
	Low     float64              `json:"l,string"`
	Close   float64              `json:"c,string"`
	Volume  float64              `json:"v,string"`

	// Added for websocket push data
	UpdateTime cryptoDotComMilliSec `json:"ut"` // this represents Update Time for websocket push data.
}

// TickersResponse represents a list of tickers.
type TickersResponse struct {
	Data []TickerItem `json:"data"`
}

// TickerItem represents a ticker item.
type TickerItem struct {
	HighestTradePrice    float64              `json:"h,string"` // Price of the 24h highest trade
	LowestTradePrice     float64              `json:"l,string"` // Price of the 24h lowest trade, null if there weren't any trades
	LatestTradePrice     float64              `json:"a,string"` // The price of the latest trade, null if there weren't any trades
	InstrumentName       string               `json:"i"`
	TradedVolume         float64              `json:"v,string"`  // The total 24h traded volume
	TradedVolumeInUSD24H float64              `json:"vv,string"` // The total 24h traded volume value (in USD)
	OpenInterest         float64              `json:"oi,string"`
	PriceChange24H       float64              `json:"c,string"` // 24-hour price change, null if there weren't any trades
	BestBidPrice         float64              `json:"b,string"` // The current best bid price, null if there aren't any bids
	BestAskPrice         float64              `json:"k,string"` // The current best ask price, null if there aren't any asks
	TradeTimestamp       cryptoDotComMilliSec `json:"t"`

	// Added for websocket push datas.
	BestBidSize float64 `json:"bs,string"`
	BestAskSize float64 `json:"ks,string"`
}

// TradesResponse represents public trades for a particular instrument.
type TradesResponse struct {
	Data []TradeItem `json:"data"`
}

// TradeItem represents a public trade item.
type TradeItem struct {
	Side           string               `json:"s"`
	TradePrice     float64              `json:"p,string"`
	TradeQuantity  float64              `json:"q,string"`
	TradeTimestamp cryptoDotComMilliSec `json:"t"`
	TradeID        string               `json:"d"`
	InstrumentName string               `json:"i"`
	DataTime       cryptoDotComMilliSec `json:"dataTime"`
}

// CancelOnDisconnectScope represents a scope of cancellation.
type CancelOnDisconnectScope struct {
	Scope string `json:"scope"`
}

// PrivateRequestParam represents a generalized private request parameter.
type PrivateRequestParam struct {
	ID        int64                  `json:"id"`
	Method    string                 `json:"method"`
	APIKey    string                 `json:"api_key,omitempty"`
	Params    map[string]interface{} `json:"params"`
	Nonce     int64                  `json:"nonce"`
	Signature string                 `json:"sig"`
}

// CurrencyNetworkResponse retrives the symbol network mapping.
type CurrencyNetworkResponse struct {
	UpdateTime  int64 `json:"update_time"`
	CurrencyMap map[string]struct {
		FullName       string              `json:"full_name"`
		DefaultNetwork string              `json:"default_network"`
		NetworkList    []NetworkListDetail `json:"network_list"`
	} `json:"currency_map"`
}

// NetworkListDetail represents a network list detail.
type NetworkListDetail struct {
	NetworkID            string  `json:"network_id"`
	WithdrawalFee        float64 `json:"withdrawal_fee"`
	WithdrawEnabled      bool    `json:"withdraw_enabled"`
	MinWithdrawalAmount  float64 `json:"min_withdrawal_amount"`
	DepositEnabled       bool    `json:"deposit_enabled"`
	ConfirmationRequired int     `json:"confirmation_required"`
}

// WithdrawalResponse represents a list of withdrawal notifications.
type WithdrawalResponse struct {
	WithdrawalList []WithdrawalItem `json:"withdrawal_list"`
}

// WithdrawalItem represents a withdrawal instance item.
type WithdrawalItem struct {
	Currency      string               `json:"currency"`
	Fee           float64              `json:"fee"`
	ID            string               `json:"id"`
	UpdateTime    cryptoDotComMilliSec `json:"update_time"`
	Amount        float64              `json:"amount"`
	Address       string               `json:"address"`
	Status        string               `json:"status"`
	TransactionID string               `json:"txid"`
	NetworkID     string               `json:"network_id"`

	Symbol             string               `json:"symbol"`
	ClientWithdrawalID string               `json:"client_wid"` // client generated withdrawal id.
	CreateTime         cryptoDotComMilliSec `json:"create_time"`
}

// DepositResponse represents accounts list of deposit funds.
type DepositResponse struct {
	DepositList []DepositItem `json:"deposit_list"`
}

// DepositItem represents accounts deposit item
type DepositItem struct {
	Currency   string               `json:"currency"`
	Fee        float64              `json:"fee"`
	CreateTime cryptoDotComMilliSec `json:"create_time"`
	ID         string               `json:"id"`
	UpdateTime cryptoDotComMilliSec `json:"update_time"`
	Amount     float64              `json:"amount"`
	Address    string               `json:"address"`
	Status     string               `json:"status"`
}

// DepositAddresses represents a list of deposit address.
type DepositAddresses struct {
	DepositAddressList []DepositAddress `json:"deposit_address_list"`
}

// DepositAddress represents a single deposit address item.
type DepositAddress struct {
	Currency   string               `json:"currency"`
	CreateTime cryptoDotComMilliSec `json:"create_time"`
	ID         string               `json:"id"`
	Address    string               `json:"address"`
	Status     string               `json:"status"`
	Network    string               `json:"network"`
}

// Accounts represents list of currency account.
type Accounts struct {
	Accounts []AccountItem `json:"accounts"`
}

// AccountItem represents a single currency account and balance detailed information.
type AccountItem struct {
	Balance   float64 `json:"balance"`
	Available float64 `json:"available"`
	Order     float64 `json:"order"`
	Stake     float64 `json:"stake"`
	Currency  string  `json:"currency"`
}

// CreateOrderResponse represents a response for a new BUY or SELL order on the Exchange.
type CreateOrderResponse struct {
	OrderID   string `json:"order_id"`
	ClientOid string `json:"client_oid"`
}

type something struct {
	ContingencyType string `json:"contingency_type,omitempty"`
	OrderList       []struct {
		InstrumentName string `json:"instrument_name"`
		Side           string `json:"side"`
		Type           string `json:"type"`
		Price          int    `json:"price"`
		Quantity       int    `json:"quantity"`
		ClientOid      string `json:"client_oid"`
		TimeInForce    string `json:"time_in_force"`
		ExecInst       string `json:"exec_inst"`
	} `json:"order_list,omitempty"`
	ResultList []struct {
		Index     int    `json:"index"`
		Code      int    `json:"code"`
		OrderID   string `json:"order_id"`
		ClientOid string `json:"client_oid"`
	} `json:"result_list,omitempty"`
}

// PersonalTrades represents a personal trade list response.
type PersonalTrades struct {
	TradeList []PersonalTradeItem `json:"trade_list"`
}

// PersonalTradeItem represents a personal trade item instance.
type PersonalTradeItem struct {
	Side           string               `json:"side"`
	InstrumentName string               `json:"instrument_name"`
	Fee            float64              `json:"fee"`
	TradeID        string               `json:"trade_id"`
	CreateTime     cryptoDotComMilliSec `json:"create_time"`
	TradedPrice    float64              `json:"traded_price"`
	TradedQuantity float64              `json:"traded_quantity"`
	FeeCurrency    string               `json:"fee_currency"`
	OrderID        string               `json:"order_id"`
}

// OrderDetail represents an order detail.
type OrderDetail struct {
	TradeList []struct {
		Side           string  `json:"side"`
		InstrumentName string  `json:"instrument_name"`
		Fee            float64 `json:"fee"`
		TradeID        string  `json:"trade_id"`
		CreateTime     int64   `json:"create_time"`
		TradedPrice    int     `json:"traded_price"`
		TradedQuantity int     `json:"traded_quantity"`
		FeeCurrency    string  `json:"fee_currency"`
		OrderID        string  `json:"order_id"`
	} `json:"trade_list"`
	OrderInfo OrderItem `json:"order_info"`
}

// OrderItem represents order instance detail information.
type OrderItem struct {
	Status             string               `json:"status"`
	Side               string               `json:"side"`
	OrderID            string               `json:"order_id"`
	ClientOid          string               `json:"client_oid"`
	CreateTime         cryptoDotComMilliSec `json:"create_time"`
	UpdateTime         cryptoDotComMilliSec `json:"update_time"`
	Type               string               `json:"type"`
	InstrumentName     string               `json:"instrument_name"`
	CumulativeQuantity float64              `json:"cumulative_quantity"`
	CumulativeValue    float64              `json:"cumulative_value"`
	AvgPrice           float64              `json:"avg_price"`
	FeeCurrency        string               `json:"fee_currency"`
	TimeInForce        string               `json:"time_in_force"`
	ExecInst           string               `json:"exec_inst"`
	// --
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
}

// PersonalOrdersResponse represents a personal order.
type PersonalOrdersResponse struct {
	Count     int64       `json:"count,omitempty"`
	OrderList []OrderItem `json:"order_list"`
}

// CreateOrderParam represents a create order request parameter.
type CreateOrderParam struct {
	InstrumentName string     `json:"instrument_name"`
	Side           order.Side `json:"side"`
	OrderType      order.Type `json:"type"`
	Price          float64    `json:"price"`
	Quantity       float64    `json:"quantity"`
	Notional       float64    `json:"notional"`
	ClientOrderID  string     `json:"client_oid"`
	TimeInForce    string     `json:"time_in_force"`
	PostOnly       bool       `json:"exec_inst"`
	TriggerPrice   float64    `json:"trigger_price"`
}

// CancelOrderParam represents the parameters to cancel an existing order.
type CancelOrderParam struct {
	InstrumentName string
	OrderID        string
}

// OrderCreationResponse represents list of order creation result information.
type OrderCreationResponse struct {
	ResultList []OrderCreationResultItem `json:"result_list"`
}

// CancelOrdersResponse represents list of cancel orders response.
type CancelOrdersResponse struct {
	ResultList []struct {
		Index int `json:"index"`
		Code  int `json:"code"`
	} `json:"result_list"`
}

// OrderCreationResultItem represents order creation result Item.
// This represents single order information.
type OrderCreationResultItem struct {
	Index     int    `json:"index"`
	Code      int    `json:"code"`
	OrderID   string `json:"order_id"`
	ClientOid string `json:"client_oid"`
}

type AccountResponse struct {
	MasterAccount  AccountInfo   `json:"master_account"`
	SubAccountList []AccountInfo `json:"sub_account_list"`
}

// AccountInfo represents the account information.
type AccountInfo struct {
	UUID              string               `json:"uuid"`
	MasterAccountUUID string               `json:"master_account_uuid"`
	MarginAccountUUID string               `json:"margin_account_uuid"`
	Enabled           bool                 `json:"enabled"`
	Tradable          bool                 `json:"tradable"`
	Name              string               `json:"name"`
	Email             string               `json:"email"`
	MobileNumber      string               `json:"mobile_number"`
	CountryCode       string               `json:"country_code"`
	Address           string               `json:"address"`
	MarginAccess      string               `json:"margin_access"`
	DerivativesAccess string               `json:"derivatives_access"`
	CreateTime        cryptoDotComMilliSec `json:"create_time"`
	UpdateTime        cryptoDotComMilliSec `json:"update_time"`
	TwoFaEnabled      bool                 `json:"two_fa_enabled"`
	KycLevel          string               `json:"kyc_level"`
	Suspended         bool                 `json:"suspended"`
	Terminated        bool                 `json:"terminated"`
	Label             string               `json:"label"`
}

// TransactionResponse represents a transaction response.
type TransactionResponse struct {
	Data []TransactionItem `json:"data"`
}

// TransactionItem represents a transaction instance.
type TransactionItem struct {
	AccountID        string `json:"account_id"`
	EventDate        string `json:"event_date"`
	JournalType      string `json:"journal_type"`
	JournalID        string `json:"journal_id"`
	TransactionQty   string `json:"transaction_qty"`
	TransactionCost  string `json:"transaction_cost"`
	RealizedPnl      string `json:"realized_pnl"`
	OrderID          string `json:"order_id,omitempty"`
	TradeID          string `json:"trade_id,omitempty"`
	TradeMatchID     string `json:"trade_match_id"`
	EventTimestampMs int64  `json:"event_timestamp_ms"`
	EventTimestampNs string `json:"event_timestamp_ns"`
	ClientOid        string `json:"client_oid"`
	TakerSide        string `json:"taker_side"`
	Side             string `json:"side,omitempty"`
	InstrumentName   string `json:"instrument_name"`
}

// OTCTrade represents an OTC trade.
type OTCTrade struct {
	AccountUUID         string               `json:"account_uuid"`
	RequestsPerMinute   int                  `json:"requests_per_minute"`
	MaxTradeValueUsd    string               `json:"max_trade_value_usd"`
	MinTradeValueUsd    string               `json:"min_trade_value_usd"`
	AcceptOtcTcDatetime cryptoDotComMilliSec `json:"accept_otc_tc_datetime"`
}

// OTCInstrumentsResponse represents an OTC instruments instance.
type OTCInstrumentsResponse struct {
	InstrumentList []OTCInstrument `json:"instrument_list"`
}

// OTCInstrument represents an OTC instrument.
type OTCInstrument struct {
	InstrumentName               string `json:"instrument_name"`
	BaseCurrency                 string `json:"base_currency"`
	QuoteCurrency                string `json:"quote_currency"`
	BaseCurrencyDecimals         int    `json:"base_currency_decimals"`
	QuoteCurrencyDecimals        int    `json:"quote_currency_decimals"`
	BaseCurrencyDisplayDecimals  int    `json:"base_currency_display_decimals"`
	QuoteCurrencyDisplayDecimals int    `json:"quote_currency_display_decimals"`
	Tradable                     bool   `json:"tradable"`
}

// OTCQuoteResponse represents quote to buy or sell with either base currency or quote currency.
type OTCQuoteResponse struct {
	QuoteID           string               `json:"quote_id"`
	QuoteStatus       string               `json:"quote_status"`
	QuoteDirection    string               `json:"quote_direction"`
	BaseCurrency      string               `json:"base_currency"`
	QuoteCurrency     string               `json:"quote_currency"`
	BaseCurrencySize  float64              `json:"base_currency_size"`
	QuoteCurrencySize string               `json:"quote_currency_size"`
	QuoteBuy          string               `json:"quote_buy"`
	QuoteBuyQuantity  string               `json:"quote_buy_quantity"`
	QuoteBuyValue     string               `json:"quote_buy_value"`
	QuoteSell         string               `json:"quote_sell"`
	QuoteSellQuantity string               `json:"quote_sell_quantity"`
	QuoteSellValue    string               `json:"quote_sell_value"`
	QuoteDuration     int                  `json:"quote_duration"`
	QuoteTime         cryptoDotComMilliSec `json:"quote_time"`
	QuoteExpiryTime   cryptoDotComMilliSec `json:"quote_expiry_time"`
}

// AcceptQuoteResponse represents response param for accepting quote.
type AcceptQuoteResponse struct {
	QuoteID           string      `json:"quote_id"`
	QuoteStatus       string      `json:"quote_status"`
	QuoteDirection    string      `json:"quote_direction"`
	BaseCurrency      string      `json:"base_currency"`
	QuoteCurrency     string      `json:"quote_currency"`
	BaseCurrencySize  interface{} `json:"base_currency_size"`
	QuoteCurrencySize string      `json:"quote_currency_size"`
	QuoteBuy          string      `json:"quote_buy"`
	QuoteSell         interface{} `json:"quote_sell"`
	QuoteDuration     int         `json:"quote_duration"`
	QuoteTime         int64       `json:"quote_time"`
	QuoteExpiryTime   int64       `json:"quote_expiry_time"`
	TradeDirection    string      `json:"trade_direction"`
	TradePrice        string      `json:"trade_price"`
	TradeQuantity     string      `json:"trade_quantity"`
	TradeValue        string      `json:"trade_value"`
	TradeTime         int64       `json:"trade_time"`
}

// QuoteHistoryResponse represents a quote history instance.
type QuoteHistoryResponse struct {
	Count     int `json:"count"`
	QuoteList []struct {
		QuoteID           string               `json:"quote_id"`
		QuoteStatus       string               `json:"quote_status"`
		QuoteDirection    string               `json:"quote_direction"`
		BaseCurrency      string               `json:"base_currency"`
		QuoteCurrency     string               `json:"quote_currency"`
		BaseCurrencySize  float64              `json:"base_currency_size"`
		QuoteCurrencySize string               `json:"quote_currency_size"`
		QuoteBuy          string               `json:"quote_buy"`
		QuoteSell         float64              `json:"quote_sell"`
		QuoteDuration     int                  `json:"quote_duration"`
		QuoteTime         cryptoDotComMilliSec `json:"quote_time"`
		QuoteExpiryTime   int64                `json:"quote_expiry_time"`
		TradeDirection    string               `json:"trade_direction"`
		TradePrice        float64              `json:"trade_price"`
		TradeQuantity     float64              `json:"trade_quantity"`
		TradeValue        float64              `json:"trade_value"`
		TradeTime         cryptoDotComMilliSec `json:"trade_time"`
	} `json:"quote_list"`
}

// OTCTradeHistoryResponse represents an OTC trade history response.
type OTCTradeHistoryResponse struct {
	Count     int `json:"count"`
	TradeList []struct {
		QuoteID           string               `json:"quote_id"`
		QuoteStatus       string               `json:"quote_status"`
		QuoteDirection    string               `json:"quote_direction"`
		BaseCurrency      string               `json:"base_currency"`
		QuoteCurrency     string               `json:"quote_currency"`
		BaseCurrencySize  string               `json:"base_currency_size"`
		QuoteCurrencySize string               `json:"quote_currency_size"`
		QuoteBuy          string               `json:"quote_buy"`
		QuoteSell         string               `json:"quote_sell"`
		QuoteDuration     int                  `json:"quote_duration"`
		QuoteTime         cryptoDotComMilliSec `json:"quote_time"`
		QuoteExpiryTime   int64                `json:"quote_expiry_time"`
		TradeDirection    string               `json:"trade_direction"`
		TradePrice        string               `json:"trade_price"`
		TradeQuantity     string               `json:"trade_quantity"`
		TradeValue        string               `json:"trade_value"`
		TradeTime         cryptoDotComMilliSec `json:"trade_time"`
	} `json:"trade_list"`
}

// SubscriptionPayload represents a subscription payload
type SubscriptionPayload struct {
	ID            int                 `json:"id"`
	Method        string              `json:"method"`
	Params        map[string][]string `json:"params"`
	Nonce         int64               `json:"nonce"`
	Authenticated bool                `json:"-"`
}

// SubscriptionResponse represents a websocket subscription response.
type SubscriptionResponse struct {
	ID     int64    `json:"id"`
	Code   int64    `json:"code,omitempty"`
	Method string   `json:"method"`
	Result WsResult `json:"result"`
}

// SubscriptionRawData represents a subscription response raw data.
type SubscriptionRawData struct {
	Data          stream.Response
	Authenticated bool
}

// WsResult represents a subscriptions response result
type WsResult struct {
	Channel        string          `json:"channel"`
	Subscription   string          `json:"subscription"`
	Data           json.RawMessage `json:"data"`
	InstrumentName string          `json:"instrument_name"`

	Depth int64 `json:"depth"` // for orderbooks

	Interval string `json:"interval"` // for candlestick datas.
}

// UserOrderbook represents a user orderbook object.
type UserOrderbook struct {
	Status                     string               `json:"status"`
	Side                       string               `json:"side"`
	Price                      float64              `json:"price"`
	Quantity                   float64              `json:"quantity"`
	OrderID                    string               `json:"order_id"`
	ClientOrderID              string               `json:"client_oid"`
	CreateTime                 cryptoDotComMilliSec `json:"create_time"`
	UpdateTime                 cryptoDotComMilliSec `json:"update_time"`
	Type                       string               `json:"type"`
	InstrumentName             string               `json:"instrument_name"`
	CumulativeExecutedQuantity float64              `json:"cumulative_quantity"`
	CumulativeExecutedValue    float64              `json:"cumulative_value"`
	AvgPrice                   float64              `json:"avg_price"`
	FeeCurrency                string               `json:"fee_currency"`
	TimeInForce                string               `json:"time_in_force"`
}

// UserTrade represents a user trade instance.
type UserTrade struct {
	Side           string                     `json:"side"`
	InstrumentName string                     `json:"instrument_name"`
	Fee            float64                    `json:"fee"`
	TradeID        string                     `json:"trade_id"`
	CreateTime     cryptoDotComMilliSecString `json:"create_time"`
	TradedPrice    float64                    `json:"traded_price"`
	TradedQuantity float64                    `json:"traded_quantity"`
	FeeCurrency    string                     `json:"fee_currency"`
	OrderID        string                     `json:"order_id"`
}

// UserBalance represents a user balance information.
type UserBalance struct {
	Currency  string  `json:"currency"`
	Balance   float64 `json:"balance"`
	Available float64 `json:"available"`
	Order     float64 `json:"order"`
	Stake     int     `json:"stake"`
}

// WsOrderbook represents an orderbook websocket push data.
type WsOrderbook struct {
	Asks                [][3]string          `json:"asks"`
	Bids                [][3]string          `json:"bids"`
	PushTime            cryptoDotComMilliSec `json:"t"`
	OrderbookUpdateTime cryptoDotComMilliSec `json:"tt"`
	UpdateSequence      int64                `json:"u"`
	Cs                  int                  `json:"cs"`
}

// WsRequestPayload represents authentication and request sending payload
type WsRequestPayload struct {
	ID        int64                  `json:"id"`
	Method    string                 `json:"method"`
	APIKey    string                 `json:"api_key,omitempty"`
	Signature string                 `json:"sig,omitempty"`
	Nonce     int64                  `json:"nonce,omitempty"`
	Params    map[string]interface{} `json:"params,omitempty"`
}

// RespData represents a generalized object structure of responses.
type RespData struct {
	ID            int             `json:"id"`
	Method        string          `json:"method"`
	Code          int             `json:"code"`
	Message       string          `json:"message"`
	DetailCode    string          `json:"detail_code"`
	DetailMessage string          `json:"detail_message"`
	Result        json.RawMessage `json:"result"`
}

// WSRespData represents a generalized object structure of websocket responses.
type WSRespData struct {
	ID            int         `json:"id"`
	Method        string      `json:"method"`
	Code          int         `json:"code"`
	Message       string      `json:"message"`
	DetailCode    string      `json:"detail_code"`
	DetailMessage string      `json:"detail_message"`
	Result        interface{} `json:"result"`
}

// InstrumentList represents a list of instruments detail items.
type InstrumentList struct {
	Instruments []Instrument `json:"instruments"`
}
