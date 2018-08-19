package coinut

import "github.com/shopspring/decimal"

// GenericResponse is the generic response you will get from coinut
type GenericResponse struct {
	Nonce     int64    `json:"nonce"`
	Reply     string   `json:"reply"`
	Status    []string `json:"status"`
	TransID   int64    `json:"trans_id"`
	Timestamp int64    `json:"timestamp"`
}

// InstrumentBase holds information on base currency
type InstrumentBase struct {
	Base          string `json:"base"`
	DecimalPlaces int    `json:"decimal_places"`
	InstID        int    `json:"inst_id"`
	Quote         string `json:"quote"`
}

// Instruments holds the full information on base currencies
type Instruments struct {
	Instruments map[string][]InstrumentBase `json:"SPOT"`
}

// Ticker holds ticker information
type Ticker struct {
	HighestBuy   decimal.Decimal `json:"highest_buy,string"`
	InstrumentID int             `json:"inst_id"`
	Last         decimal.Decimal `json:"last,string"`
	LowestSell   decimal.Decimal `json:"lowest_sell,string"`
	OpenInterest decimal.Decimal `json:"open_interest,string"`
	Timestamp    decimal.Decimal `json:"timestamp"`
	TransID      int64           `json:"trans_id"`
	Volume       decimal.Decimal `json:"volume,string"`
	Volume24     decimal.Decimal `json:"volume24,string"`
}

// OrderbookBase is a sub-type holding price and quantity
type OrderbookBase struct {
	Count    int             `json:"count"`
	Price    decimal.Decimal `json:"price,string"`
	Quantity decimal.Decimal `json:"qty,string"`
}

// Orderbook is the full order book
type Orderbook struct {
	Buy          []OrderbookBase `json:"buy"`
	Sell         []OrderbookBase `json:"sell"`
	InstrumentID int             `json:"inst_id"`
	TotalBuy     decimal.Decimal `json:"total_buy,string"`
	TotalSell    decimal.Decimal `json:"total_sell,string"`
	TransID      int64           `json:"trans_id"`
}

// TradeBase is a sub-type holding information on trades
type TradeBase struct {
	Price     decimal.Decimal `json:"price,string"`
	Quantity  decimal.Decimal `json:"quantity,string"`
	Side      string          `json:"side"`
	Timestamp decimal.Decimal `json:"timestamp"`
	TransID   int64           `json:"trans_id"`
}

// Trades holds the full amount of trades associated with API keys
type Trades struct {
	Trades []TradeBase `json:"trades"`
}

// UserBalance holds user balances on the exchange
type UserBalance struct {
	BTC               decimal.Decimal `json:"btc,string"`
	ETC               decimal.Decimal `json:"etc,string"`
	ETH               decimal.Decimal `json:"eth,string"`
	LTC               decimal.Decimal `json:"ltc,string"`
	Equity            decimal.Decimal `json:"equity,string,string"`
	InitialMargin     decimal.Decimal `json:"initial_margin,string"`
	MaintenanceMargin decimal.Decimal `json:"maintenance_margin,string"`
	RealizedPL        decimal.Decimal `json:"realized_pl,string"`
	TransID           int64           `json:"trans_id"`
	UnrealizedPL      decimal.Decimal `json:"unrealized_pl,string"`
}

// Order holds order information
type Order struct {
	InstrumentID  int64           `json:"inst_id"`
	Price         decimal.Decimal `json:"price,string"`
	Quantity      decimal.Decimal `json:"qty,string"`
	ClientOrderID int             `json:"client_ord_id"`
	Side          string          `json:"side,string"`
}

// OrderResponse is a response for orders
type OrderResponse struct {
	OrderID       int64           `json:"order_id"`
	OpenQuantity  decimal.Decimal `json:"open_qty,string"`
	Price         decimal.Decimal `json:"price,string"`
	Quantity      decimal.Decimal `json:"qty,string"`
	InstrumentID  int64           `json:"inst_id"`
	ClientOrderID int64           `json:"client_ord_id"`
	Timestamp     int64           `json:"timestamp"`
	OrderPrice    decimal.Decimal `json:"order_price,string"`
	Side          string          `json:"side"`
}

// Commission holds trade commission structure
type Commission struct {
	Currency string          `json:"currency"`
	Amount   decimal.Decimal `json:"amount,string"`
}

// OrderFilledResponse contains order filled response
type OrderFilledResponse struct {
	GenericResponse
	Commission   Commission      `json:"commission"`
	FillPrice    decimal.Decimal `json:"fill_price,string"`
	FillQuantity decimal.Decimal `json:"fill_qty,string"`
	Order        OrderResponse   `json:"order"`
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

// OrdersResponse holds the full data range on orders
type OrdersResponse struct {
	Data []OrdersBase
}

// CancelOrders holds information about a cancelled order
type CancelOrders struct {
	InstrumentID int   `json:"int"`
	OrderID      int64 `json:"order_id"`
}

// CancelOrdersResponse is response for a cancelled order
type CancelOrdersResponse struct {
	GenericResponse
	Results []struct {
		OrderID      int64  `json:"order_id"`
		Status       string `json:"status"`
		InstrumentID int    `json:"inst_id"`
	} `json:"results"`
}

// TradeHistory holds trade history information
type TradeHistory struct {
	TotalNumber int64                 `json:"total_number"`
	Trades      []OrderFilledResponse `json:"trades"`
}

// IndexTicker holds indexed ticker inforamtion
type IndexTicker struct {
	Asset string          `json:"asset"`
	Price decimal.Decimal `json:"price,string"`
}

// Option holds options information
type Option struct {
	HighestBuy   decimal.Decimal `json:"highest_buy,string"`
	InstrumentID int             `json:"inst_id"`
	Last         decimal.Decimal `json:"last,string"`
	LowestSell   decimal.Decimal `json:"lowest_sell,string"`
	OpenInterest decimal.Decimal `json:"open_interest,string"`
}

// OptionChainResponse is the response type for options
type OptionChainResponse struct {
	ExpiryTime   int64  `json:"expiry_time"`
	SecurityType string `json:"sec_type"`
	Asset        string `json:"asset"`
	Entries      []struct {
		Call   Option          `json:"call"`
		Put    Option          `json:"put"`
		Strike decimal.Decimal `json:"strike,string"`
	}
}

// OptionChainUpdate contains information on the chain update options
type OptionChainUpdate struct {
	Option
	GenericResponse
	Asset        string          `json:"asset"`
	ExpiryTime   int64           `json:"expiry_time"`
	SecurityType string          `json:"sec_type"`
	Volume       decimal.Decimal `json:"volume,string"`
}

// PositionHistory holds the complete position history
type PositionHistory struct {
	Positions []struct {
		PositionID int `json:"position_id"`
		Records    []struct {
			Commission    Commission      `json:"commission"`
			FillPrice     decimal.Decimal `json:"fill_price,string,omitempty"`
			TransactionID int             `json:"trans_id"`
			FillQuantity  decimal.Decimal `json:"fill_qty,omitempty"`
			Position      struct {
				Commission Commission      `json:"commission"`
				Timestamp  int64           `json:"timestamp"`
				OpenPrice  decimal.Decimal `json:"open_price,string"`
				RealizedPL decimal.Decimal `json:"realized_pl,string"`
				Quantity   decimal.Decimal `json:"qty,string"`
			} `json:"position"`
			AssetAtExpiry decimal.Decimal `json:"asset_at_expiry,string,omitempty"`
		} `json:"records"`
		Instrument struct {
			ExpiryTime     int64           `json:"expiry_time"`
			ContractSize   decimal.Decimal `json:"contract_size,string"`
			ConversionRate decimal.Decimal `json:"conversion_rate,string"`
			OptionType     string          `json:"option_type"`
			InstrumentID   int             `json:"inst_id"`
			SecType        string          `json:"sec_type"`
			Asset          string          `json:"asset"`
			Strike         decimal.Decimal `json:"strike,string"`
		} `json:"inst"`
		OpenTimestamp int64 `json:"open_timestamp"`
	} `json:"positions"`
	TotalNumber int `json:"total_number"`
}

// OpenPosition holds information on an open position
type OpenPosition struct {
	PositionID    int             `json:"position_id"`
	Commission    Commission      `json:"commission"`
	OpenPrice     decimal.Decimal `json:"open_price,string"`
	RealizedPL    decimal.Decimal `json:"realized_pl,string"`
	Quantity      decimal.Decimal `json:"qty,string"`
	OpenTimestamp int64           `json:"open_timestamp"`
	InstrumentID  int             `json:"inst_id"`
}
