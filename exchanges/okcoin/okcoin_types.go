package okcoin

import "github.com/thrasher-/gocryptotrader/decimal"

// Ticker holds ticker data
type Ticker struct {
	Buy  decimal.Decimal `json:",string"`
	High decimal.Decimal `json:",string"`
	Last decimal.Decimal `json:",string"`
	Low  decimal.Decimal `json:",string"`
	Sell decimal.Decimal `json:",string"`
	Vol  decimal.Decimal `json:",string"`
}

// TickerResponse is the response type for ticker
type TickerResponse struct {
	Date   string
	Ticker Ticker
}

// FuturesTicker holds futures ticker data
type FuturesTicker struct {
	Last       decimal.Decimal
	Buy        decimal.Decimal
	Sell       decimal.Decimal
	High       decimal.Decimal
	Low        decimal.Decimal
	Vol        decimal.Decimal
	ContractID int64
	UnitAmount decimal.Decimal
}

// Orderbook holds orderbook data
type Orderbook struct {
	Asks [][]decimal.Decimal `json:"asks"`
	Bids [][]decimal.Decimal `json:"bids"`
}

// FuturesTickerResponse is a response type
type FuturesTickerResponse struct {
	Date   string
	Ticker FuturesTicker
}

// BorrowInfo holds borrowing amount data
type BorrowInfo struct {
	BorrowBTC        decimal.Decimal `json:"borrow_btc"`
	BorrowLTC        decimal.Decimal `json:"borrow_ltc"`
	BorrowCNY        decimal.Decimal `json:"borrow_cny"`
	CanBorrow        decimal.Decimal `json:"can_borrow"`
	InterestBTC      decimal.Decimal `json:"interest_btc"`
	InterestLTC      decimal.Decimal `json:"interest_ltc"`
	Result           bool            `json:"result"`
	DailyInterestBTC decimal.Decimal `json:"today_interest_btc"`
	DailyInterestLTC decimal.Decimal `json:"today_interest_ltc"`
	DailyInterestCNY decimal.Decimal `json:"today_interest_cny"`
}

// BorrowOrder holds order data
type BorrowOrder struct {
	Amount      decimal.Decimal `json:"amount"`
	BorrowDate  int64           `json:"borrow_date"`
	BorrowID    int64           `json:"borrow_id"`
	Days        int64           `json:"days"`
	TradeAmount decimal.Decimal `json:"deal_amount"`
	Rate        decimal.Decimal `json:"rate"`
	Status      int64           `json:"status"`
	Symbol      string          `json:"symbol"`
}

// Record hold record data
type Record struct {
	Address            string          `json:"addr"`
	Account            int64           `json:"account,string"`
	Amount             decimal.Decimal `json:"amount"`
	Bank               string          `json:"bank"`
	BenificiaryAddress string          `json:"benificiary_addr"`
	TransactionValue   decimal.Decimal `json:"transaction_value"`
	Fee                decimal.Decimal `json:"fee"`
	Date               decimal.Decimal `json:"date"`
}

// AccountRecords holds account record data
type AccountRecords struct {
	Records []Record `json:"records"`
	Symbol  string   `json:"symbol"`
}

// FuturesOrder holds information about a futures order
type FuturesOrder struct {
	Amount       decimal.Decimal `json:"amount"`
	ContractName string          `json:"contract_name"`
	DateCreated  decimal.Decimal `json:"create_date"`
	TradeAmount  decimal.Decimal `json:"deal_amount"`
	Fee          decimal.Decimal `json:"fee"`
	LeverageRate decimal.Decimal `json:"lever_rate"`
	OrderID      int64           `json:"order_id"`
	Price        decimal.Decimal `json:"price"`
	AvgPrice     decimal.Decimal `json:"avg_price"`
	Status       decimal.Decimal `json:"status"`
	Symbol       string          `json:"symbol"`
	Type         int64           `json:"type"`
	UnitAmount   int64           `json:"unit_amount"`
}

// FuturesHoldAmount contains futures hold amount data
type FuturesHoldAmount struct {
	Amount       decimal.Decimal `json:"amount"`
	ContractName string          `json:"contract_name"`
}

// FuturesExplosive holds inforamtion about explosive futures
type FuturesExplosive struct {
	Amount      decimal.Decimal `json:"amount,string"`
	DateCreated string          `json:"create_date"`
	Loss        decimal.Decimal `json:"loss,string"`
	Type        int64           `json:"type"`
}

// Trades holds trade data
type Trades struct {
	Amount  decimal.Decimal `json:"amount,string"`
	Date    int64           `json:"date"`
	DateMS  int64           `json:"date_ms"`
	Price   decimal.Decimal `json:"price,string"`
	TradeID int64           `json:"tid"`
	Type    string          `json:"type"`
}

// FuturesTrades holds trade data for the futures market
type FuturesTrades struct {
	Amount  decimal.Decimal `json:"amount"`
	Date    int64           `json:"date"`
	DateMS  int64           `json:"date_ms"`
	Price   decimal.Decimal `json:"price"`
	TradeID int64           `json:"tid"`
	Type    string          `json:"type"`
}

// UserInfo holds user account details
type UserInfo struct {
	Info struct {
		Funds struct {
			Asset struct {
				Net   decimal.Decimal `json:"net,string"`
				Total decimal.Decimal `json:"total,string"`
			} `json:"asset"`
			Borrow struct {
				BTC decimal.Decimal `json:"btc,string"`
				LTC decimal.Decimal `json:"ltc,string"`
				USD decimal.Decimal `json:"usd,string"`
				CNY decimal.Decimal `json:"cny,string"`
			} `json:"borrow"`
			Free struct {
				BTC decimal.Decimal `json:"btc,string"`
				LTC decimal.Decimal `json:"ltc,string"`
				USD decimal.Decimal `json:"usd,string"`
				CNY decimal.Decimal `json:"cny,string"`
			} `json:"free"`
			Freezed struct {
				BTC decimal.Decimal `json:"btc,string"`
				LTC decimal.Decimal `json:"ltc,string"`
				USD decimal.Decimal `json:"usd,string"`
				CNY decimal.Decimal `json:"cny,string"`
			} `json:"freezed"`
			UnionFund struct {
				BTC decimal.Decimal `json:"btc,string"`
				LTC decimal.Decimal `json:"ltc,string"`
			} `json:"union_fund"`
		} `json:"funds"`
	} `json:"info"`
	Result bool `json:"result"`
}

// BatchTrade holds data on a batch of trades
type BatchTrade struct {
	OrderInfo []struct {
		OrderID   int64 `json:"order_id"`
		ErrorCode int64 `json:"error_code"`
	} `json:"order_info"`
	Result bool `json:"result"`
}

// CancelOrderResponse is a response type for a cancelled order
type CancelOrderResponse struct {
	Success string
	Error   string
}

// OrderInfo holds data on an order
type OrderInfo struct {
	Amount     decimal.Decimal `json:"amount"`
	AvgPrice   decimal.Decimal `json:"avg_price"`
	Created    int64           `json:"create_date"`
	DealAmount decimal.Decimal `json:"deal_amount"`
	OrderID    int64           `json:"order_id"`
	OrdersID   int64           `json:"orders_id"`
	Price      decimal.Decimal `json:"price"`
	Status     int             `json:"status"`
	Symbol     string          `json:"symbol"`
	Type       string          `json:"type"`
}

// OrderHistory holds information on order history
type OrderHistory struct {
	CurrentPage int         `json:"current_page"`
	Orders      []OrderInfo `json:"orders"`
	PageLength  int         `json:"page_length"`
	Result      bool        `json:"result"`
	Total       int         `json:"total"`
}

// WithdrawalResponse is a response type for withdrawal
type WithdrawalResponse struct {
	WithdrawID int  `json:"withdraw_id"`
	Result     bool `json:"result"`
}

// WithdrawInfo holds data on a withdraw
type WithdrawInfo struct {
	Address    string          `json:"address"`
	Amount     decimal.Decimal `json:"amount"`
	Created    int64           `json:"created_date"`
	ChargeFee  decimal.Decimal `json:"chargefee"`
	Status     int             `json:"status"`
	WithdrawID int64           `json:"withdraw_id"`
}

// OrderFeeInfo holds data on order fees
type OrderFeeInfo struct {
	Fee     decimal.Decimal `json:"fee,string"`
	OrderID int64           `json:"order_id"`
	Type    string          `json:"type"`
}

// LendDepth hold lend depths
type LendDepth struct {
	Amount decimal.Decimal `json:"amount"`
	Days   string          `json:"days"`
	Num    int64           `json:"num"`
	Rate   decimal.Decimal `json:"rate,string"`
}

// BorrowResponse is a response type for borrow
type BorrowResponse struct {
	Result   bool `json:"result"`
	BorrowID int  `json:"borrow_id"`
}

// WebsocketFutureIndex holds future index data for websocket
type WebsocketFutureIndex struct {
	FutureIndex decimal.Decimal `json:"futureIndex"`
	Timestamp   int64           `json:"timestamp,string"`
}

// WebsocketTicker holds ticker data for websocket
type WebsocketTicker struct {
	Timestamp decimal.Decimal
	Vol       string
	Buy       decimal.Decimal
	High      decimal.Decimal
	Last      decimal.Decimal
	Low       decimal.Decimal
	Sell      decimal.Decimal
}

// WebsocketFuturesTicker holds futures ticker data for websocket
type WebsocketFuturesTicker struct {
	Buy        decimal.Decimal `json:"buy"`
	ContractID string          `json:"contractId"`
	High       decimal.Decimal `json:"high"`
	HoldAmount decimal.Decimal `json:"hold_amount"`
	Last       decimal.Decimal `json:"last,string"`
	Low        decimal.Decimal `json:"low"`
	Sell       decimal.Decimal `json:"sell"`
	UnitAmount decimal.Decimal `json:"unitAmount"`
	Volume     decimal.Decimal `json:"vol,string"`
}

// WebsocketOrderbook holds orderbook data for websocket
type WebsocketOrderbook struct {
	Asks      [][]decimal.Decimal `json:"asks"`
	Bids      [][]decimal.Decimal `json:"bids"`
	Timestamp int64               `json:"timestamp,string"`
}

// WebsocketUserinfo holds user info for websocket
type WebsocketUserinfo struct {
	Info struct {
		Funds struct {
			Asset struct {
				Net   decimal.Decimal `json:"net,string"`
				Total decimal.Decimal `json:"total,string"`
			} `json:"asset"`
			Free struct {
				BTC decimal.Decimal `json:"btc,string"`
				LTC decimal.Decimal `json:"ltc,string"`
				USD decimal.Decimal `json:"usd,string"`
				CNY decimal.Decimal `json:"cny,string"`
			} `json:"free"`
			Frozen struct {
				BTC decimal.Decimal `json:"btc,string"`
				LTC decimal.Decimal `json:"ltc,string"`
				USD decimal.Decimal `json:"usd,string"`
				CNY decimal.Decimal `json:"cny,string"`
			} `json:"freezed"`
		} `json:"funds"`
	} `json:"info"`
	Result bool `json:"result"`
}

// WebsocketFuturesContract holds futures contract information for websocket
type WebsocketFuturesContract struct {
	Available    decimal.Decimal `json:"available"`
	Balance      decimal.Decimal `json:"balance"`
	Bond         decimal.Decimal `json:"bond"`
	ContractID   decimal.Decimal `json:"contract_id"`
	ContractType string          `json:"contract_type"`
	Frozen       decimal.Decimal `json:"freeze"`
	Profit       decimal.Decimal `json:"profit"`
	Loss         decimal.Decimal `json:"unprofit"`
}

// WebsocketFuturesUserInfo holds futures user information for websocket
type WebsocketFuturesUserInfo struct {
	Info struct {
		BTC struct {
			Balance   decimal.Decimal            `json:"balance"`
			Contracts []WebsocketFuturesContract `json:"contracts"`
			Rights    decimal.Decimal            `json:"rights"`
		} `json:"btc"`
		LTC struct {
			Balance   decimal.Decimal            `json:"balance"`
			Contracts []WebsocketFuturesContract `json:"contracts"`
			Rights    decimal.Decimal            `json:"rights"`
		} `json:"ltc"`
	} `json:"info"`
	Result bool `json:"result"`
}

// WebsocketOrder holds order data for websocket
type WebsocketOrder struct {
	Amount      decimal.Decimal `json:"amount"`
	AvgPrice    decimal.Decimal `json:"avg_price"`
	DateCreated decimal.Decimal `json:"create_date"`
	TradeAmount decimal.Decimal `json:"deal_amount"`
	OrderID     decimal.Decimal `json:"order_id"`
	OrdersID    decimal.Decimal `json:"orders_id"`
	Price       decimal.Decimal `json:"price"`
	Status      int64           `json:"status"`
	Symbol      string          `json:"symbol"`
	OrderType   string          `json:"type"`
}

// WebsocketFuturesOrder holds futures order data for websocket
type WebsocketFuturesOrder struct {
	Amount         decimal.Decimal `json:"amount"`
	ContractName   string          `json:"contract_name"`
	DateCreated    decimal.Decimal `json:"createdDate"`
	TradeAmount    decimal.Decimal `json:"deal_amount"`
	Fee            decimal.Decimal `json:"fee"`
	LeverageAmount int             `json:"lever_rate"`
	OrderID        decimal.Decimal `json:"order_id"`
	Price          decimal.Decimal `json:"price"`
	AvgPrice       decimal.Decimal `json:"avg_price"`
	Status         int             `json:"status"`
	Symbol         string          `json:"symbol"`
	TradeType      int             `json:"type"`
	UnitAmount     decimal.Decimal `json:"unit_amount"`
}

// WebsocketRealtrades holds real trade data for WebSocket
type WebsocketRealtrades struct {
	AveragePrice         decimal.Decimal `json:"averagePrice,string"`
	CompletedTradeAmount decimal.Decimal `json:"completedTradeAmount,string"`
	DateCreated          decimal.Decimal `json:"createdDate"`
	ID                   decimal.Decimal `json:"id"`
	OrderID              decimal.Decimal `json:"orderId"`
	SigTradeAmount       decimal.Decimal `json:"sigTradeAmount,string"`
	SigTradePrice        decimal.Decimal `json:"sigTradePrice,string"`
	Status               int64           `json:"status"`
	Symbol               string          `json:"symbol"`
	TradeAmount          decimal.Decimal `json:"tradeAmount,string"`
	TradePrice           decimal.Decimal `json:"buy,string"`
	TradeType            string          `json:"tradeType"`
	TradeUnitPrice       decimal.Decimal `json:"tradeUnitPrice,string"`
	UnTrade              decimal.Decimal `json:"unTrade,string"`
}

// WebsocketFuturesRealtrades holds futures real trade data for websocket
type WebsocketFuturesRealtrades struct {
	Amount         decimal.Decimal `json:"amount,string"`
	ContractID     decimal.Decimal `json:"contract_id,string"`
	ContractName   string          `json:"contract_name"`
	ContractType   string          `json:"contract_type"`
	TradeAmount    decimal.Decimal `json:"deal_amount,string"`
	Fee            decimal.Decimal `json:"fee,string"`
	OrderID        decimal.Decimal `json:"orderid"`
	Price          decimal.Decimal `json:"price,string"`
	AvgPrice       decimal.Decimal `json:"price_avg,string"`
	Status         int             `json:"status,string"`
	TradeType      int             `json:"type,string"`
	UnitAmount     decimal.Decimal `json:"unit_amount,string"`
	LeverageAmount int             `json:"lever_rate,string"`
}

// WebsocketEvent holds websocket events
type WebsocketEvent struct {
	Event   string `json:"event"`
	Channel string `json:"channel"`
}

// WebsocketResponse holds websocket responses
type WebsocketResponse struct {
	Channel string      `json:"channel"`
	Data    interface{} `json:"data"`
}

// WebsocketEventAuth holds websocket authenticated events
type WebsocketEventAuth struct {
	Event      string            `json:"event"`
	Channel    string            `json:"channel"`
	Parameters map[string]string `json:"parameters"`
}

// WebsocketEventAuthRemove holds websocket remove authenticated events
type WebsocketEventAuthRemove struct {
	Event      string            `json:"event"`
	Channel    string            `json:"channel"`
	Parameters map[string]string `json:"parameters"`
}

// WebsocketTradeOrderResponse holds trade order responses for websocket
type WebsocketTradeOrderResponse struct {
	OrderID int64 `json:"order_id,string"`
	Result  bool  `json:"result,string"`
}
