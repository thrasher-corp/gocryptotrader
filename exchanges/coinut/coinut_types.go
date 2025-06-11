package coinut

import (
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// GenericResponse is the generic response you will get from coinut
type GenericResponse struct {
	Nonce         int64    `json:"nonce"`
	Reply         string   `json:"reply"`
	Status        []string `json:"status"`
	TransactionID int64    `json:"trans_id"`
}

// InstrumentBase holds information on base currency
type InstrumentBase struct {
	Base          string `json:"base"`
	DecimalPlaces int    `json:"decimal_places"`
	InstrumentID  int64  `json:"inst_id"`
	Quote         string `json:"quote"`
}

// Instruments holds the full information on base currencies
type Instruments struct {
	Instruments map[string][]InstrumentBase `json:"SPOT"`
}

// Ticker holds ticker information
type Ticker struct {
	High24                float64    `json:"high24,string"`
	HighestBuy            float64    `json:"highest_buy,string"`
	InstrumentID          int        `json:"inst_id"`
	Last                  float64    `json:"last,string"`
	Low24                 float64    `json:"low24,string"`
	LowestSell            float64    `json:"lowest_sell,string"`
	PreviousTransactionID int64      `json:"prev_trans_id"`
	PriceChange24         float64    `json:"price_change_24,string"`
	Reply                 string     `json:"reply"`
	OpenInterest          float64    `json:"open_interest,string"`
	Timestamp             types.Time `json:"timestamp"`
	TransactionID         int64      `json:"trans_id"`
	Volume                float64    `json:"volume,string"`
	Volume24              float64    `json:"volume24,string"`
	Volume24Quote         float64    `json:"volume24_quote,string"`
	VolumeQuote           float64    `json:"volume_quote,string"`
}

// OrderbookBase is a sub-type holding price and quantity
type OrderbookBase struct {
	Count    int     `json:"count"`
	Price    float64 `json:"price,string"`
	Quantity float64 `json:"qty,string"`
}

// Orderbook is the full order book
type Orderbook struct {
	Buy           []OrderbookBase `json:"buy"`
	Sell          []OrderbookBase `json:"sell"`
	InstrumentID  int             `json:"inst_id"`
	TotalBuy      float64         `json:"total_buy,string"`
	TotalSell     float64         `json:"total_sell,string"`
	TransactionID int64           `json:"trans_id"`
}

// TradeBase is a sub-type holding information on trades
type TradeBase struct {
	Price         float64    `json:"price,string"`
	Quantity      float64    `json:"qty,string"`
	Side          string     `json:"side"`
	Timestamp     types.Time `json:"timestamp"`
	TransactionID int64      `json:"trans_id"`
}

// Trades holds the full amount of trades associated with API keys
type Trades struct {
	Trades []TradeBase `json:"trades"`
}

// UserBalance holds user balances on the exchange
type UserBalance struct {
	BCH     float64  `json:"BCH,string"`
	BTC     float64  `json:"BTC,string"`
	BTG     float64  `json:"BTG,string"`
	CAD     float64  `json:"CAD,string"`
	ETC     float64  `json:"ETC,string"`
	ETH     float64  `json:"ETH,string"`
	LCH     float64  `json:"LCH,string"`
	LTC     float64  `json:"LTC,string"`
	MYR     float64  `json:"MYR,string"`
	SGD     float64  `json:"SGD,string"`
	USD     float64  `json:"USD,string"`
	USDT    float64  `json:"USDT,string"`
	XMR     float64  `json:"XMR,string"`
	ZEC     float64  `json:"ZEC,string"`
	Nonce   int64    `json:"nonce"`
	Reply   string   `json:"reply"`
	Status  []string `json:"status"`
	TransID int64    `json:"trans_id"`
}

// Order holds order information
type Order struct {
	InstrumentID  int64   `json:"inst_id"`
	Price         float64 `json:"price,string"`
	Quantity      float64 `json:"qty,string"`
	ClientOrderID int     `json:"client_ord_id"`
	Side          string  `json:"side,string"`
}

// OrderResponse is a response for orders
type OrderResponse struct {
	OrderID       int64      `json:"order_id"`
	OpenQuantity  float64    `json:"open_qty,string"`
	Price         float64    `json:"price,string"`
	Quantity      float64    `json:"qty,string"`
	InstrumentID  int64      `json:"inst_id"`
	ClientOrderID int64      `json:"client_ord_id"`
	Timestamp     types.Time `json:"timestamp"`
	OrderPrice    float64    `json:"order_price,string"`
	Side          string     `json:"side"`
}

// Commission holds trade commission structure
type Commission struct {
	Currency string  `json:"currency"`
	Amount   float64 `json:"amount,string"`
}

// OrderFilledResponse contains order filled response
type OrderFilledResponse struct {
	GenericResponse
	Commission   Commission    `json:"commission"`
	FillPrice    float64       `json:"fill_price,string"`
	FillQuantity float64       `json:"fill_qty,string"`
	Order        OrderResponse `json:"order"`
}

// OrderRejectResponse holds information on a rejected order
type OrderRejectResponse struct {
	OrderResponse
	Reasons []string `json:"reasons"`
}

// OrdersBase contains generic response and order responses
type OrdersBase struct {
	GenericResponse
	OrderResponse
}

// GetOpenOrdersResponse holds all order data from GetOpenOrders request
type GetOpenOrdersResponse struct {
	Nonce         int             `json:"nonce"`
	Orders        []OrderResponse `json:"orders"`
	Reply         string          `json:"reply"`
	Status        []string        `json:"status"`
	TransactionID int             `json:"trans_id"`
}

// OrdersResponse holds the full data range on orders
type OrdersResponse struct {
	Data []OrdersBase
}

// CancelOrders holds information about a cancelled order
type CancelOrders struct {
	InstrumentID int64 `json:"inst_id"`
	OrderID      int64 `json:"order_id"`
}

// CancelOrdersResponse is response for a cancelled order
type CancelOrdersResponse struct {
	GenericResponse
	Results []struct {
		OrderID      int64  `json:"order_id"`
		Status       string `json:"status"`
		InstrumentID int64  `json:"inst_id"`
	} `json:"results"`
}

// TradeHistory holds trade history information
type TradeHistory struct {
	TotalNumber int64                 `json:"total_number"`
	Trades      []OrderFilledResponse `json:"trades"`
}

// IndexTicker holds indexed ticker information
type IndexTicker struct {
	Asset string  `json:"asset"`
	Price float64 `json:"price,string"`
}

// Option holds options information
type Option struct {
	HighestBuy   float64 `json:"highest_buy,string"`
	InstrumentID int     `json:"inst_id"`
	Last         float64 `json:"last,string"`
	LowestSell   float64 `json:"lowest_sell,string"`
	OpenInterest float64 `json:"open_interest,string"`
}

// OptionChainResponse is the response type for options
type OptionChainResponse struct {
	ExpiryTime   types.Time `json:"expiry_time"`
	SecurityType string     `json:"sec_type"`
	Asset        string     `json:"asset"`
	Entries      []struct {
		Call   Option  `json:"call"`
		Put    Option  `json:"put"`
		Strike float64 `json:"strike,string"`
	}
}

// OptionChainUpdate contains information on the chain update options
type OptionChainUpdate struct {
	Option
	GenericResponse
	Asset        string     `json:"asset"`
	ExpiryTime   types.Time `json:"expiry_time"`
	SecurityType string     `json:"sec_type"`
	Volume       float64    `json:"volume,string"`
}

// PositionHistory holds the complete position history
type PositionHistory struct {
	Positions []struct {
		PositionID int `json:"position_id"`
		Records    []struct {
			Commission    Commission `json:"commission"`
			FillPrice     float64    `json:"fill_price,string,omitempty"`
			TransactionID int        `json:"trans_id"`
			FillQuantity  float64    `json:"fill_qty,omitempty"`
			Position      struct {
				Commission Commission `json:"commission"`
				Timestamp  types.Time `json:"timestamp"`
				OpenPrice  float64    `json:"open_price,string"`
				RealizedPL float64    `json:"realized_pl,string"`
				Quantity   float64    `json:"qty,string"`
			} `json:"position"`
			AssetAtExpiry float64 `json:"asset_at_expiry,string,omitempty"`
		} `json:"records"`
		Instrument struct {
			ExpiryTime     types.Time `json:"expiry_time"`
			ContractSize   float64    `json:"contract_size,string"`
			ConversionRate float64    `json:"conversion_rate,string"`
			OptionType     string     `json:"option_type"`
			InstrumentID   int        `json:"inst_id"`
			SecType        string     `json:"sec_type"`
			Asset          string     `json:"asset"`
			Strike         float64    `json:"strike,string"`
		} `json:"inst"`
		OpenTimestamp types.Time `json:"open_timestamp"`
	} `json:"positions"`
	TotalNumber int `json:"total_number"`
}

// OpenPosition holds information on an open position
type OpenPosition struct {
	PositionID    int        `json:"position_id"`
	Commission    Commission `json:"commission"`
	OpenPrice     float64    `json:"open_price,string"`
	RealizedPL    float64    `json:"realized_pl,string"`
	Quantity      float64    `json:"qty,string"`
	OpenTimestamp types.Time `json:"open_timestamp"`
	InstrumentID  int        `json:"inst_id"`
}

type wsRequest struct {
	Request      string `json:"request"`
	SecurityType string `json:"sec_type,omitempty"`
	InstrumentID int64  `json:"inst_id,omitempty"`
	TopN         int64  `json:"top_n,omitempty"`
	Subscribe    bool   `json:"subscribe,omitempty"`
	Nonce        int64  `json:"nonce,omitempty"`
}

type wsResponse struct {
	Nonce int64  `json:"nonce,omitempty"`
	Reply string `json:"reply"`
}

// WsTicker defines the resp for ticker updates from the websocket connection
type WsTicker struct {
	High24        float64    `json:"high24,string"`
	HighestBuy    float64    `json:"highest_buy,string"`
	InstID        int64      `json:"inst_id"`
	Last          float64    `json:"last,string"`
	Low24         float64    `json:"low24,string"`
	LowestSell    float64    `json:"lowest_sell,string"`
	Nonce         int64      `json:"nonce"`
	PrevTransID   int64      `json:"prev_trans_id"`
	PriceChange24 float64    `json:"price_change_24,string"`
	Reply         string     `json:"reply"`
	Status        []string   `json:"status"`
	Timestamp     types.Time `json:"timestamp"`
	TransID       int64      `json:"trans_id"`
	Volume        float64    `json:"volume,string"`
	Volume24      float64    `json:"volume24,string"`
	Volume24Quote float64    `json:"volume24_quote,string"`
	VolumeQuote   float64    `json:"volume_quote,string"`
}

// WsOrderbookSnapshot defines the resp for orderbook snapshot updates from
// the websocket connection
type WsOrderbookSnapshot struct {
	Buy       []WsOrderbookData `json:"buy"`
	Sell      []WsOrderbookData `json:"sell"`
	InstID    int64             `json:"inst_id"`
	Nonce     int64             `json:"nonce"`
	TotalBuy  float64           `json:"total_buy,string"`
	TotalSell float64           `json:"total_sell,string"`
	Reply     string            `json:"reply"`
	Status    []any             `json:"status"`
}

// WsOrderbookData defines singular orderbook data
type WsOrderbookData struct {
	Count  int64   `json:"count"`
	Price  float64 `json:"price,string"`
	Volume float64 `json:"qty,string"`
}

// WsOrderbookUpdate defines orderbook update response from the websocket
// connection
type WsOrderbookUpdate struct {
	Count    int64   `json:"count"`
	InstID   int64   `json:"inst_id"`
	Price    float64 `json:"price,string"`
	Volume   float64 `json:"qty,string"`
	TotalBuy float64 `json:"total_buy,string"`
	Reply    string  `json:"reply"`
	Side     string  `json:"side"`
	TransID  int64   `json:"trans_id"`
}

// WsTradeSnapshot defines Market trade response from the websocket
// connection
type WsTradeSnapshot struct {
	InstrumentID int64         `json:"inst_id"`
	Nonce        int64         `json:"nonce"`
	Reply        string        `json:"reply"`
	Status       []any         `json:"status"`
	Trades       []WsTradeData `json:"trades"`
}

// WsTradeData defines market trade data
type WsTradeData struct {
	InstID    int64      `json:"inst_id"`
	TransID   int64      `json:"trans_id"`
	Price     float64    `json:"price,string"`
	Quantity  float64    `json:"qty,string"`
	Side      string     `json:"side"`
	Timestamp types.Time `json:"timestamp"`
}

// WsTradeUpdate defines trade update response from the websocket connection
type WsTradeUpdate struct {
	InstID    int64      `json:"inst_id"`
	TransID   int64      `json:"trans_id"`
	Price     float64    `json:"price,string"`
	Quantity  float64    `json:"qty,string"`
	Side      string     `json:"side"`
	Timestamp types.Time `json:"timestamp"`
	Reply     string     `json:"reply"`
}

// WsInstrumentList defines instrument list
type WsInstrumentList struct {
	Spot   map[string][]InstrumentBase `json:"SPOT"`
	Nonce  int64                       `json:"nonce,omitempty"`
	Reply  string                      `json:"inst_list,omitempty"`
	Status []any                       `json:"status,omitempty"`
}

// WsSupportedCurrency defines supported currency on the exchange
type WsSupportedCurrency struct {
	Base          string `json:"base"`
	InstID        int64  `json:"inst_id"`
	DecimalPlaces int64  `json:"decimal_places"`
	Quote         string `json:"quote"`
}

// WsRequest base request
type WsRequest struct {
	Request string `json:"request"`
	Nonce   int64  `json:"nonce"`
}

// WsTradeHistoryRequest ws request
type WsTradeHistoryRequest struct {
	InstID int64 `json:"inst_id"`
	Start  int64 `json:"start,omitempty"`
	Limit  int64 `json:"limit,omitempty"`
	WsRequest
}

// WsCancelOrdersRequest ws request
type WsCancelOrdersRequest struct {
	Entries []WsCancelOrdersRequestEntry `json:"entries"`
	WsRequest
}

// WsCancelOrdersRequestEntry ws request entry
type WsCancelOrdersRequestEntry struct {
	InstID  int64 `json:"inst_id"`
	OrderID int64 `json:"order_id"`
}

// WsCancelOrderParameters ws request parameters
type WsCancelOrderParameters struct {
	Currency currency.Pair
	OrderID  int64
}

// WsCancelOrderRequest data required for cancelling an order
type WsCancelOrderRequest struct {
	InstrumentID int64 `json:"inst_id"`
	OrderID      int64 `json:"order_id"`
	WsRequest
}

// WsCancelOrderResponse contains cancelled order data
type WsCancelOrderResponse struct {
	Nonce         int64    `json:"nonce"`
	Reply         string   `json:"reply"`
	OrderID       int64    `json:"order_id"`
	ClientOrderID int64    `json:"client_ord_id"`
	Status        []string `json:"status"`
}

// WsCancelOrdersResponse contains all cancelled order data
type WsCancelOrdersResponse struct {
	Nonce         int64                        `json:"nonce"`
	Reply         string                       `json:"reply"`
	Results       []WsCancelOrdersResponseData `json:"results"`
	Status        []string                     `json:"status"`
	TransactionID int64                        `json:"trans_id"`
}

// WsCancelOrdersResponseData individual cancellation response data
type WsCancelOrdersResponseData struct {
	InstrumentID int64  `json:"inst_id"`
	OrderID      int64  `json:"order_id"`
	Status       string `json:"status"`
}

// WsGetOpenOrdersRequest ws request
type WsGetOpenOrdersRequest struct {
	InstrumentID int64 `json:"inst_id"`
	WsRequest
}

// WsSubmitOrdersRequest ws request
type WsSubmitOrdersRequest struct {
	Orders []WsSubmitOrdersRequestData `json:"orders"`
	WsRequest
}

// WsSubmitOrdersRequestData ws request data
type WsSubmitOrdersRequestData struct {
	InstrumentID  int64   `json:"inst_id"`
	Price         float64 `json:"price,string"`
	Quantity      float64 `json:"qty,string"`
	ClientOrderID int     `json:"client_ord_id"`
	Side          string  `json:"side"`
}

// WsSubmitOrderRequest ws request
type WsSubmitOrderRequest struct {
	InstrumentID int64   `json:"inst_id"`
	Price        float64 `json:"price,string"`
	Quantity     float64 `json:"qty,string"`
	OrderID      int64   `json:"client_ord_id"`
	Side         string  `json:"side"`
	WsRequest
}

// WsSubmitOrderParameters ws request parameters
type WsSubmitOrderParameters struct {
	Currency      currency.Pair
	Side          order.Side
	Amount, Price float64
	OrderID       int64
}

// WsUserBalanceResponse ws response
type WsUserBalanceResponse struct {
	Nonce              int64    `json:"nonce"`
	Status             []string `json:"status"`
	Btc                float64  `json:"BTC,string"`
	Ltc                float64  `json:"LTC,string"`
	Etc                float64  `json:"ETC,string"`
	Eth                float64  `json:"ETH,string"`
	FloatingProfitLoss float64  `json:"floating_pl,string"`
	InitialMargin      float64  `json:"initial_margin,string"`
	RealisedProfitLoss float64  `json:"realized_pl,string"`
	MaintenanceMargin  float64  `json:"maintenance_margin,string"`
	Equity             float64  `json:"equity,string"`
	Reply              string   `json:"reply"`
	TransactionID      int64    `json:"trans_id"`
}

// WsOrderAcceptedResponse ws response
type WsOrderAcceptedResponse struct {
	Nonce         int64    `json:"nonce"`
	Status        []string `json:"status"`
	OrderID       int64    `json:"order_id"`
	OpenQuantity  float64  `json:"open_qty,string"`
	InstrumentID  int64    `json:"inst_id"`
	Quantity      float64  `json:"qty,string"`
	ClientOrderID int64    `json:"client_ord_id"`
	OrderPrice    float64  `json:"order_price,string"`
	Reply         string   `json:"reply"`
	Side          string   `json:"side"`
	TransactionID int64    `json:"trans_id"`
}

// WsOrderFilledResponse ws response
type WsOrderFilledResponse struct {
	Commission    WsOrderFilledCommissionData `json:"commission"`
	FillPrice     float64                     `json:"fill_price,string"`
	FillQuantity  float64                     `json:"fill_qty,string"`
	Nonce         int64                       `json:"nonce"`
	Order         WsOrderData                 `json:"order"`
	Reply         string                      `json:"reply"`
	Status        []string                    `json:"status"`
	Timestamp     types.Time                  `json:"timestamp"`
	TransactionID int64                       `json:"trans_id"`
}

// WsOrderData ws response data
type WsOrderData struct {
	ClientOrderID int64      `json:"client_ord_id"`
	InstrumentID  int64      `json:"inst_id"`
	OpenQuantity  float64    `json:"open_qty,string"`
	OrderID       int64      `json:"order_id"`
	Price         float64    `json:"price,string"`
	Quantity      float64    `json:"qty,string"`
	Side          string     `json:"side"`
	Timestamp     types.Time `json:"timestamp"`
	Status        []string   `json:"status"`
}

// WsOrderFilledCommissionData ws response data
type WsOrderFilledCommissionData struct {
	Amount   float64 `json:"amount,string"`
	Currency string  `json:"currency"`
}

// WsOrderRejectedResponse ws response
type WsOrderRejectedResponse struct {
	Nonce         int64      `json:"nonce"`
	Status        []string   `json:"status"`
	OrderID       int64      `json:"order_id"`
	OpenQuantity  float64    `json:"open_qty,string"`
	Price         float64    `json:"price,string"`
	InstrumentID  int64      `json:"inst_id"`
	Reasons       []string   `json:"reasons"`
	ClientOrderID int64      `json:"client_ord_id"`
	Timestamp     types.Time `json:"timestamp"`
	Reply         string     `json:"reply"`
	Quantity      float64    `json:"qty,string"`
	Side          string     `json:"side"`
	TransactionID int64      `json:"trans_id"`
}

type wsInstList struct {
	Spot map[string][]struct {
		Base          string `json:"base"`
		DecimalPlaces int64  `json:"decimal_places"`
		InstrumentID  int64  `json:"inst_id"`
		Quote         string `json:"quote"`
	} `json:"spot"`
}

// WsUserOpenOrdersResponse ws response
type WsUserOpenOrdersResponse struct {
	Nonce  int64         `json:"nonce"`
	Reply  string        `json:"reply"`
	Status []string      `json:"status"`
	Orders []WsOrderData `json:"orders"`
}

// WsTradeHistoryResponse ws response
type WsTradeHistoryResponse struct {
	Nonce       int64         `json:"nonce"`
	Reply       string        `json:"reply"`
	Status      []string      `json:"status"`
	TotalNumber int64         `json:"total_number"`
	Trades      []WsOrderData `json:"trades"`
}

// WsTradeHistoryCommissionData ws response data
type WsTradeHistoryCommissionData struct {
	Amount   float64 `json:"amount,string"`
	Currency string  `json:"currency"`
}

// WsTradeHistoryTradeData ws response data
type WsTradeHistoryTradeData struct {
	Commission    WsTradeHistoryCommissionData `json:"commission"`
	Order         WsOrderData                  `json:"order"`
	FillPrice     float64                      `json:"fill_price,string"`
	FillQuantity  float64                      `json:"fill_qty,string"`
	Timestamp     types.Time                   `json:"timestamp"`
	TransactionID int64                        `json:"trans_id"`
}

// WsLoginReq Login request message
type WsLoginReq struct {
	Request   string `json:"request"`
	Username  string `json:"username"`
	Nonce     int64  `json:"nonce"`
	Hmac      string `json:"hmac_sha256"`
	Timestamp int64  `json:"timestamp"`
}

// WsLoginResponse ws response data
type WsLoginResponse struct {
	APIKey          string     `json:"api_key"`
	Country         string     `json:"country"`
	DepositEnabled  bool       `json:"deposit_enabled"`
	Deposited       bool       `json:"deposited"`
	Email           string     `json:"email"`
	FailedTimes     types.Time `json:"failed_times"`
	KycPassed       bool       `json:"kyc_passed"`
	Language        string     `json:"lang"`
	Nonce           int64      `json:"nonce"`
	OTPEnabled      bool       `json:"otp_enabled"`
	PhoneNumber     string     `json:"phone_number"`
	ProductsEnabled []string   `json:"products_enabled"`
	Referred        bool       `json:"referred"`
	Reply           string     `json:"reply"`
	SessionID       string     `json:"session_id"`
	Status          []string   `json:"status"`
	Timezone        string     `json:"timezone"`
	Traded          bool       `json:"traded"`
	UnverifiedEmail string     `json:"unverified_email"`
	Username        string     `json:"username"`
	WithdrawEnabled bool       `json:"withdraw_enabled"`
}

// WsNewOrderResponse returns if new_order response fails
type WsNewOrderResponse struct {
	Message string   `json:"msg"`
	Nonce   int64    `json:"nonce"`
	Reply   string   `json:"reply"`
	Status  []string `json:"status"`
}

// WsGetAccountBalanceResponse contains values of each currency
type WsGetAccountBalanceResponse struct {
	BCH     float64  `json:"BCH,string"`
	BTC     float64  `json:"BTC,string"`
	BTG     float64  `json:"BTG,string"`
	CAD     float64  `json:"CAD,string"`
	ETC     float64  `json:"ETC,string"`
	ETH     float64  `json:"ETH,string"`
	LCH     float64  `json:"LCH,string"`
	LTC     float64  `json:"LTC,string"`
	MYR     float64  `json:"MYR,string"`
	SGD     float64  `json:"SGD,string"`
	USD     float64  `json:"USD,string"`
	USDT    float64  `json:"USDT,string"`
	XMR     float64  `json:"XMR,string"`
	ZEC     float64  `json:"ZEC,string"`
	Nonce   int64    `json:"nonce"`
	Reply   string   `json:"reply"`
	Status  []string `json:"status"`
	TransID int64    `json:"trans_id"`
}

type instrumentMap struct {
	Instruments map[string]int64
	Loaded      bool
	m           sync.Mutex
}

type wsOrderContainer struct {
	OrderID       int64      `json:"order_id"`
	ClientOrderID int64      `json:"client_ord_id"`
	InstrumentID  int64      `json:"inst_id"`
	Nonce         int64      `json:"nonce"`
	Timestamp     types.Time `json:"timestamp"`
	TransactionID int64      `json:"trans_id"`
	OpenQuantity  float64    `json:"open_qty,string"`
	OrderPrice    float64    `json:"order_price,string"`
	Quantity      float64    `json:"qty,string"`
	FillPrice     float64    `json:"fill_price,string"`
	FillQuantity  float64    `json:"fill_qty,string"`
	Price         float64    `json:"price,string"`
	Reply         string     `json:"reply"`
	Side          string     `json:"side"`
	Status        []string   `json:"status"`
	Reasons       []string   `json:"reasons"`
	Order         struct {
		ClientOrderID int64      `json:"client_ord_id"`
		InstrumentID  int64      `json:"inst_id"`
		OrderID       int64      `json:"order_id"`
		Timestamp     types.Time `json:"timestamp"`
		Price         float64    `json:"price,string"`
		Quantity      float64    `json:"qty,string"`
		OpenQuantity  float64    `json:"open_qty,string"`
		Side          string     `json:"side"`
	} `json:"order"`
	Commission struct {
		Amount   float64 `json:"amount,string"`
		Currency string  `json:"currency"`
	} `json:"commission"`
}
