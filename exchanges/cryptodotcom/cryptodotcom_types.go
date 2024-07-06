package cryptodotcom

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	errInvalidOrderCancellationScope = errors.New("invalid order cancellation scope, only ACCOUNT or CONNECTION is supported")
	errInvalidResponseFromServer     = errors.New("invalid response from server")
	errInvalidQuantity               = errors.New("quantity must be non-zero positive decimal value for order")
	errTriggerPriceRequired          = errors.New("trigger price is required")
	errSubAccountAddressRequired     = errors.New("sub-account address is required")
)

// Instrument represents an details.
type Instrument struct {
	InstrumentName          string               `json:"instrument_name"`
	QuoteCurrency           string               `json:"quote_currency"`
	BaseCurrency            string               `json:"base_currency"`
	PriceDecimals           int64                `json:"price_decimals"`
	QuantityDecimals        int64                `json:"quantity_decimals"`
	MarginTradingEnabled    bool                 `json:"margin_trading_enabled"`
	MarginTradingEnabled5X  bool                 `json:"margin_trading_enabled_5x"`
	MarginTradingEnabled10X bool                 `json:"margin_trading_enabled_10x"`
	MaxQuantity             SafeNumber           `json:"max_quantity"`
	MinQuantity             SafeNumber           `json:"min_quantity"`
	MaxPrice                SafeNumber           `json:"max_price"`
	MinPrice                SafeNumber           `json:"min_price"`
	LastUpdateDate          convert.ExchangeTime `json:"last_update_date"`
	QuantityTickSize        SafeNumber           `json:"quantity_tick_size"`
	PriceTickSize           SafeNumber           `json:"price_tick_size"`
}

// OrderbookDetail public order book detail.
type OrderbookDetail struct {
	Depth int64 `json:"depth"`
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
	EndTime convert.ExchangeTime `json:"t"` // this represents Start Time for websocket push data.
	Open    float64              `json:"o,string"`
	High    float64              `json:"h,string"`
	Low     float64              `json:"l,string"`
	Close   float64              `json:"c,string"`
	Volume  float64              `json:"v,string"`
}

// WsCandlestickItem represents candlestick (k-line data history) item pushed through the websocket connection.
type WsCandlestickItem struct {
	CandlestickItem
	// Added for websocket push data
	UpdateTime convert.ExchangeTime `json:"ut"` // this represents Update Time for websocket push data.
}

// OTCBook represents an orderbook data for OTC instrument.
type OTCBook struct {
	Asks [][5]types.Number `json:"asks"` // Price, Total Size, Number of Orders in the level, Expiry Time, Unique ID
	Bids [][5]types.Number `json:"bids"` // Price, Total Size, Number of Orders in the level, Expiry Time, Unique ID
}

// TickersResponse represents a list of tickers.
type TickersResponse struct {
	Data []TickerItem `json:"data"`
}

// InstrumentValuation represents a particular instrument valuation.
type InstrumentValuation struct {
	Data []struct {
		Value     types.Number         `json:"v"`
		Timestamp convert.ExchangeTime `json:"t"`
	} `json:"data"`
	InstrumentName string `json:"instrument_name"`
}

// TickerItem represents a ticker item.
type TickerItem struct {
	InstrumentName       string               `json:"i"`
	HighestTradePrice    SafeNumber           `json:"h"`  // Price of the 24h highest trade
	LowestTradePrice     SafeNumber           `json:"l"`  // Price of the 24h lowest trade, null if there weren't any trades
	LatestTradePrice     SafeNumber           `json:"a"`  // The price of the latest trade, null if there weren't any trades
	TradedVolume         SafeNumber           `json:"v"`  // The total 24h traded volume
	TradedVolumeInUSD24H SafeNumber           `json:"vv"` // The total 24h traded volume value (in USD)
	OpenInterest         string               `json:"oi"`
	PriceChange24H       SafeNumber           `json:"c"` // 24-hour price change, null if there weren't any trades
	BestBidPrice         SafeNumber           `json:"b"` // The current best bid price, null if there aren't any bids
	BestAskPrice         SafeNumber           `json:"k"` // The current best ask price, null if there aren't any asks
	TradeTimestamp       convert.ExchangeTime `json:"t"`

	// Added for websocket push data.
	BestBidSize SafeNumber `json:"bs"`
	BestAskSize SafeNumber `json:"ks"`
}

// TradesResponse represents public trades for a particular instrument.
type TradesResponse struct {
	Data []TradeItem `json:"data"`
}

// TradeItem represents a public trade item.
type TradeItem struct {
	Side           string               `json:"s"`
	TradePrice     SafeNumber           `json:"p"`
	TradeQuantity  SafeNumber           `json:"q"`
	TradeTimestamp convert.ExchangeTime `json:"t"`
	TradeID        string               `json:"d"`
	InstrumentName string               `json:"i"`
	DataTime       convert.ExchangeTime `json:"dataTime"`
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

// CurrencyNetworkResponse retrieves the symbol network mapping.
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
	ConfirmationRequired int64   `json:"confirmation_required"`
}

// WithdrawalResponse represents a list of withdrawal notifications.
type WithdrawalResponse struct {
	WithdrawalList []WithdrawalItem `json:"withdrawal_list"`
}

// WithdrawalItem represents a withdrawal instance item.
type WithdrawalItem struct {
	Currency           string               `json:"currency"`
	Fee                float64              `json:"fee"`
	ID                 string               `json:"id"`
	UpdateTime         convert.ExchangeTime `json:"update_time"`
	Amount             float64              `json:"amount"`
	Address            string               `json:"address"`
	Status             string               `json:"status"`
	TransactionID      string               `json:"txid"`
	NetworkID          string               `json:"network_id"`
	Symbol             string               `json:"symbol"`
	ClientWithdrawalID string               `json:"client_wid"` // client generated withdrawal id.
	CreateTime         convert.ExchangeTime `json:"create_time"`
}

// DepositResponse represents accounts list of deposit funds.
type DepositResponse struct {
	DepositList []DepositItem `json:"deposit_list"`
}

// DepositItem represents accounts deposit item
type DepositItem struct {
	Currency   string               `json:"currency"`
	Fee        float64              `json:"fee"`
	ID         string               `json:"id"`
	CreateTime convert.ExchangeTime `json:"create_time"`
	UpdateTime convert.ExchangeTime `json:"update_time"`
	Amount     float64              `json:"amount"`
	Address    string               `json:"address"`
	Status     string               `json:"status"`
}

// DepositAddresses represents a list of deposit address.
type DepositAddresses struct {
	DepositAddressList []DepositAddress `json:"deposit_address_list"`
}

// ExportRequestResponse represents a response after creating an instrument export request.
type ExportRequestResponse struct {
	ID              int64  `json:"id"`
	Status          string `json:"status"`
	ClientRequestID string `json:"client_request_id"`
}

// ExportRequests represents a list of export requests
type ExportRequests struct {
	UserBatchList []struct {
		ID              string               `json:"id"`
		StartTime       convert.ExchangeTime `json:"start_ts"`
		EndTime         convert.ExchangeTime `json:"end_ts"`
		InstrumentNames []string             `json:"instrument_names"`
		RequestedData   []string             `json:"requested_data"`
		ClientRequestID string               `json:"client_request_id"`
		Status          string               `json:"status"`
	} `json:"user_batch_list"`
}

// DepositAddress represents a single deposit address item.
type DepositAddress struct {
	Currency   string               `json:"currency"`
	CreateTime convert.ExchangeTime `json:"create_time"`
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
	CreateTime     convert.ExchangeTime `json:"create_time"`
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
		TradedPrice    float64 `json:"traded_price"`
		TradedQuantity float64 `json:"traded_quantity"`
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
	CreateTime         convert.ExchangeTime `json:"create_time"`
	UpdateTime         convert.ExchangeTime `json:"update_time"`
	Type               string               `json:"type"`
	InstrumentName     string               `json:"instrument_name"`
	CumulativeQuantity float64              `json:"cumulative_quantity"`
	CumulativeValue    float64              `json:"cumulative_value"`
	AvgPrice           float64              `json:"avg_price"`
	FeeCurrency        string               `json:"fee_currency"`
	TimeInForce        string               `json:"time_in_force"`
	ExecInst           string               `json:"exec_inst"`
	Price              float64              `json:"price"`
	Quantity           float64              `json:"quantity"`
}

// PersonalOrdersResponse represents a personal order.
type PersonalOrdersResponse struct {
	Count     int64       `json:"count,omitempty"`
	OrderList []OrderItem `json:"order_list"`
}

// CreateOrderParam represents a create order request parameter.
type CreateOrderParam struct {
	Symbol        string     `json:"instrument_name"`
	Side          order.Side `json:"side"`
	OrderType     string     `json:"type"`
	Price         float64    `json:"price"`
	Quantity      float64    `json:"quantity"`
	Notional      float64    `json:"notional"`
	ClientOrderID string     `json:"client_oid"`
	TimeInForce   string     `json:"time_in_force"`
	PostOnly      bool       `json:"exec_inst"`
	TriggerPrice  float64    `json:"trigger_price"`
}

func (arg *CreateOrderParam) getCreateParamMap() (map[string]interface{}, error) {
	if arg == nil {
		return nil, fmt.Errorf("%w, CreateOrderParam can not be nil", common.ErrNilPointer)
	}
	if arg.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.Side != order.Sell && arg.Side != order.Buy {
		return nil, fmt.Errorf("%w, side: %s", order.ErrSideIsInvalid, arg.Side)
	}
	oType, err := stringToOrderType(arg.OrderType)
	if err != nil {
		return nil, err
	}
	switch oType {
	case order.Limit, order.StopLimit, order.TakeProfitLimit:
		if arg.Price <= 0 { // Unit price
			return nil, fmt.Errorf("%w, price must be non-zero positive decimal value", order.ErrPriceBelowMin)
		}
		if arg.Quantity <= 0 {
			return nil, errInvalidQuantity
		}
		switch oType {
		case order.StopLimit, order.TakeProfitLimit:
			if arg.TriggerPrice <= 0 {
				return nil, fmt.Errorf("%w for Order Type: %v", errTriggerPriceRequired, arg.OrderType)
			}
		}
	case order.Market:
		if arg.Side == order.Buy {
			if arg.Notional <= 0 && arg.Quantity <= 0 {
				return nil, fmt.Errorf("either notional or quantity must be non-zero value for order type: %v and order side: %v", arg.OrderType, arg.Side)
			}
		} else {
			if arg.Quantity <= 0 {
				return nil, fmt.Errorf("%w order type: %v and order side: %v", errInvalidQuantity, arg.OrderType, arg.Side)
			}
		}
	case order.StopLoss, order.TakeProfit:
		if arg.Side == order.Sell {
			if arg.Quantity <= 0 {
				return nil, fmt.Errorf("%w order type: %v and order side: %v", errInvalidQuantity, arg.OrderType, arg.Side)
			}
		} else {
			if arg.Notional <= 0 {
				return nil, fmt.Errorf("notional must be non-zero positive decimal value for order type: %v", arg.OrderType)
			}
		}
		if arg.TriggerPrice <= 0 {
			return nil, fmt.Errorf("%w for Order Type: %s", errTriggerPriceRequired, arg.OrderType)
		}
	default:
		return nil, fmt.Errorf("unsupported order type: %v", arg.OrderType)
	}
	params := make(map[string]interface{})
	params["instrument_name"] = arg.Symbol
	params["side"] = arg.Side.String()
	params["type"] = arg.OrderType
	params["price"] = arg.Price
	if arg.Quantity > 0 {
		params["quantity"] = arg.Quantity
	}
	if arg.Notional > 0 {
		params["notional"] = arg.Notional
	}
	if arg.ClientOrderID != "" {
		params["client_oid"] = arg.ClientOrderID
	}
	if arg.TimeInForce != "" {
		params["time_in_force"] = arg.TimeInForce
	}
	if arg.PostOnly {
		params["exec_inst"] = "POST_ONLY"
	}
	if arg.TriggerPrice > 0 {
		params["trigger_price"] = arg.TriggerPrice
	}
	return params, nil
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
		Index int64 `json:"index"`
		Code  int64 `json:"code"`
	} `json:"result_list"`
}

// OrderCreationResultItem represents order creation result Item.
// This represents single order information.
type OrderCreationResultItem struct {
	Index     int64  `json:"index"`
	Code      int64  `json:"code"`
	OrderID   string `json:"order_id"`
	ClientOid string `json:"client_oid"`
}

// AccountResponse represents main and sub account detail information
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
	CreateTime        convert.ExchangeTime `json:"create_time"`
	UpdateTime        convert.ExchangeTime `json:"update_time"`
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
	OrderID             string               `json:"order_id,omitempty"`
	AccountID           string               `json:"account_id"`
	TradeMatchID        string               `json:"trade_match_id"`
	TradeID             string               `json:"trade_id,omitempty"`
	EventDate           string               `json:"event_date"` // format 2021-02-18
	JournalType         string               `json:"journal_type"`
	JournalID           string               `json:"journal_id"`
	TransactionCost     SafeNumber           `json:"transaction_cost"`
	TransactionQuantity SafeNumber           `json:"transaction_qty"`
	RealizedPnl         SafeNumber           `json:"realized_pnl"`
	EventTimestampMs    convert.ExchangeTime `json:"event_timestamp_ms"` // Event timestamp in milliseconds
	EventTimestampNs    convert.ExchangeTime `json:"event_timestamp_ns"` // Event timestamp in nanoseconds
	ClientOrderID       string               `json:"client_oid"`
	TakerSide           string               `json:"taker_side"`
	Side                string               `json:"side,omitempty"`
	InstrumentName      string               `json:"instrument_name"`
}

// OTCTrade represents an OTC trade.
type OTCTrade struct {
	AccountUUID         string               `json:"account_uuid"`
	RequestsPerMinute   int64                `json:"requests_per_minute"`
	MaxTradeValueUSD    SafeNumber           `json:"max_trade_value_usd"`
	MinTradeValueUSD    SafeNumber           `json:"min_trade_value_usd"`
	AcceptOtcTcDatetime convert.ExchangeTime `json:"accept_otc_tc_datetime"`
}

// OTCInstrumentsResponse represents an OTC instruments instance.
type OTCInstrumentsResponse struct {
	InstrumentList []OTCInstrument `json:"instrument_list"`
}

// OTCInstrument represents an OTC instrument.
type OTCInstrument struct {
	InstrumentName               string  `json:"instrument_name"`
	BaseCurrency                 string  `json:"base_currency"`
	QuoteCurrency                string  `json:"quote_currency"`
	BaseCurrencyDecimals         float64 `json:"base_currency_decimals"`
	QuoteCurrencyDecimals        float64 `json:"quote_currency_decimals"`
	BaseCurrencyDisplayDecimals  float64 `json:"base_currency_display_decimals"`
	QuoteCurrencyDisplayDecimals float64 `json:"quote_currency_display_decimals"`
	Tradable                     bool    `json:"tradable"`
}

// OTCQuoteResponse represents quote to buy or sell with either base currency or quote currency.
type OTCQuoteResponse struct {
	QuoteID           string               `json:"quote_id"`
	QuoteStatus       string               `json:"quote_status"`
	QuoteDirection    string               `json:"quote_direction"`
	BaseCurrency      string               `json:"base_currency"`
	QuoteCurrency     string               `json:"quote_currency"`
	BaseCurrencySize  SafeNumber           `json:"base_currency_size"`
	QuoteCurrencySize SafeNumber           `json:"quote_currency_size"`
	QuoteBuy          SafeNumber           `json:"quote_buy"`
	QuoteBuyQuantity  SafeNumber           `json:"quote_buy_quantity"`
	QuoteBuyValue     SafeNumber           `json:"quote_buy_value"`
	QuoteSell         SafeNumber           `json:"quote_sell"`
	QuoteSellQuantity SafeNumber           `json:"quote_sell_quantity"`
	QuoteSellValue    SafeNumber           `json:"quote_sell_value"`
	QuoteDuration     int64                `json:"quote_duration"`
	QuoteTime         convert.ExchangeTime `json:"quote_time"`
	QuoteExpiryTime   convert.ExchangeTime `json:"quote_expiry_time"`
}

// AcceptQuoteResponse represents response param for accepting quote.
type AcceptQuoteResponse struct {
	QuoteID           string               `json:"quote_id"`
	QuoteStatus       string               `json:"quote_status"`
	TradeDirection    string               `json:"trade_direction"`
	QuoteDirection    string               `json:"quote_direction"`
	BaseCurrency      string               `json:"base_currency"`
	QuoteCurrency     string               `json:"quote_currency"`
	BaseCurrencySize  SafeNumber           `json:"base_currency_size"`
	QuoteCurrencySize SafeNumber           `json:"quote_currency_size"`
	QuoteBuy          SafeNumber           `json:"quote_buy"`
	QuoteSell         SafeNumber           `json:"quote_sell"`
	QuoteDuration     int64                `json:"quote_duration"`
	QuoteTime         convert.ExchangeTime `json:"quote_time"`
	QuoteExpiryTime   convert.ExchangeTime `json:"quote_expiry_time"`
	TradePrice        SafeNumber           `json:"trade_price"`
	TradeQuantity     SafeNumber           `json:"trade_quantity"`
	TradedValue       SafeNumber           `json:"trade_value"`
	TradeTime         convert.ExchangeTime `json:"trade_time"`
}

// QuoteHistoryResponse represents a quote history instance.
type QuoteHistoryResponse struct {
	Count     int64 `json:"count"`
	QuoteList []struct {
		QuoteID           string               `json:"quote_id"`
		QuoteStatus       string               `json:"quote_status"`
		QuoteDirection    string               `json:"quote_direction"`
		BaseCurrency      string               `json:"base_currency"`
		QuoteCurrency     string               `json:"quote_currency"`
		BaseCurrencySize  float64              `json:"base_currency_size"`
		QuoteCurrencySize SafeNumber           `json:"quote_currency_size"`
		QuoteBuy          SafeNumber           `json:"quote_buy"`
		QuoteSell         SafeNumber           `json:"quote_sell"`
		QuoteDuration     int64                `json:"quote_duration"`
		QuoteTime         convert.ExchangeTime `json:"quote_time"`
		QuoteExpiryTime   convert.ExchangeTime `json:"quote_expiry_time"`
		TradeDirection    string               `json:"trade_direction"`
		TradePrice        float64              `json:"trade_price"`
		TradeQuantity     float64              `json:"trade_quantity"`
		TradeValue        float64              `json:"trade_value"`
		TradeTime         convert.ExchangeTime `json:"trade_time"`
	} `json:"quote_list"`
}

// OTCTradeHistoryResponse represents an OTC trade history response.
type OTCTradeHistoryResponse struct {
	Count     int64          `json:"count"`
	TradeList []OTCTradeItem `json:"trade_list"`
}

// OTCOrderResponse represents an OTC order response.
type OTCOrderResponse struct {
	ClientOid       string               `json:"client_oid"`
	OrderID         string               `json:"order_id"`
	Status          string               `json:"status"` // FILLED, REJECTED, UNSETTLED, PENDING
	InstrumentName  string               `json:"instrument_name"`
	Side            string               `json:"side"`
	Price           types.Number         `json:"price"`
	Quantity        types.Number         `json:"quantity"`
	Value           string               `json:"value"`
	CreateTime      convert.ExchangeTime `json:"create_time"`
	RejectionReason string               `json:"reject_reason"`
}

// OTCTradeItem represents an OTC trade item detail.
type OTCTradeItem struct {
	QuoteID           string               `json:"quote_id"`
	QuoteStatus       string               `json:"quote_status"`
	QuoteDirection    string               `json:"quote_direction"`
	BaseCurrency      string               `json:"base_currency"`
	QuoteCurrency     string               `json:"quote_currency"`
	BaseCurrencySize  SafeNumber           `json:"base_currency_size"`
	QuoteCurrencySize SafeNumber           `json:"quote_currency_size"`
	QuoteBuy          string               `json:"quote_buy"`
	QuoteSell         string               `json:"quote_sell"`
	QuoteDuration     int64                `json:"quote_duration"`
	QuoteTime         convert.ExchangeTime `json:"quote_time"`
	QuoteExpiryTime   convert.ExchangeTime `json:"quote_expiry_time"`
	TradeDirection    string               `json:"trade_direction"`
	TradePrice        SafeNumber           `json:"trade_price"`
	TradeQuantity     SafeNumber           `json:"trade_quantity"`
	TradeValue        SafeNumber           `json:"trade_value"`
	TradeTime         convert.ExchangeTime `json:"trade_time"`
}

// SubscriptionPayload represents a subscription payload
type SubscriptionPayload struct {
	ID            int64               `json:"id"`
	Method        string              `json:"method"`
	Params        map[string][]string `json:"params"`
	Nonce         int64               `json:"nonce"`
	Authenticated bool                `json:"-"`
}

// SubscriptionResponse represents a websocket subscription response.
type SubscriptionResponse struct {
	ID     int64     `json:"id"`
	Code   int64     `json:"code,omitempty"`
	Method string    `json:"method"`
	Result *WsResult `json:"result,omitempty"`
}

// SubscriptionInput represents a public/heartbead response
type SubscriptionInput struct {
	ID     int64  `json:"id"`
	Code   int64  `json:"code,omitempty"`
	Method string `json:"method"`
}

// SubscriptionRawData represents a subscription response raw data.
type SubscriptionRawData struct {
	Data          []byte
	Authenticated bool
}

// WsResult represents a subscriptions response result
type WsResult struct {
	Channel        string               `json:"channel,omitempty"`
	Subscription   string               `json:"subscription,omitempty"`
	Data           json.RawMessage      `json:"data,omitempty"`
	InstrumentName string               `json:"instrument_name,omitempty"`
	Depth          int64                `json:"depth,omitempty"`    // for orderbooks
	Interval       string               `json:"interval,omitempty"` // for candlestick data.
	Timestamp      convert.ExchangeTime `json:"t"`                  // Timestamp of book publish (milliseconds since the Unix epoch)
}

// UserOrder represents a user orderbook object.
type UserOrder struct {
	Status                     string               `json:"status"`
	Side                       string               `json:"side"`
	Price                      float64              `json:"price"`
	Quantity                   float64              `json:"quantity"`
	OrderID                    string               `json:"order_id"`
	ClientOrderID              string               `json:"client_oid"`
	CreateTime                 convert.ExchangeTime `json:"create_time"`
	UpdateTime                 convert.ExchangeTime `json:"update_time"`
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
	Side           string               `json:"side"`
	InstrumentName string               `json:"instrument_name"`
	Fee            float64              `json:"fee"`
	TradeID        string               `json:"trade_id"`
	CreateTime     convert.ExchangeTime `json:"create_time"`
	TradedPrice    float64              `json:"traded_price"`
	TradedQuantity float64              `json:"traded_quantity"`
	FeeCurrency    string               `json:"fee_currency"`
	OrderID        string               `json:"order_id"`
}

// UserBalance represents a user balance information.
type UserBalance struct {
	Currency  string  `json:"currency"`
	Balance   float64 `json:"balance"`
	Available float64 `json:"available"`
	Order     float64 `json:"order"`
	Stake     int64   `json:"stake"`
}

// WsOrderbook represents an orderbook websocket push data.
type WsOrderbook struct {
	Asks                [][3]string          `json:"asks"`
	Bids                [][3]string          `json:"bids"`
	PushTime            convert.ExchangeTime `json:"t"`
	OrderbookUpdateTime convert.ExchangeTime `json:"tt"`
	UpdateSequence      int64                `json:"u"`
	Cs                  int64                `json:"cs"`
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
	ID            int64       `json:"id"`
	Method        string      `json:"method"`
	Code          int64       `json:"code"`
	Message       string      `json:"message"`
	DetailCode    string      `json:"detail_code"`
	DetailMessage string      `json:"detail_message"`
	Result        interface{} `json:"result"`
}

// WSRespData represents a generalized object structure of websocket responses.
type WSRespData struct {
	ID            int64       `json:"id"`
	Method        string      `json:"method"`
	Code          int64       `json:"code"`
	Message       string      `json:"message"`
	DetailCode    string      `json:"detail_code"`
	DetailMessage string      `json:"detail_message"`
	Result        interface{} `json:"result"`
}

// InstrumentList represents a list of instruments detail items.
type InstrumentList struct {
	Instruments []Instrument `json:"instruments"`
}
