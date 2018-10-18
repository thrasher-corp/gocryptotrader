package poloniex

import "github.com/thrasher-/gocryptotrader/currency/symbol"

// Ticker holds ticker data
type Ticker struct {
	Last          float64 `json:"last,string"`
	LowestAsk     float64 `json:"lowestAsk,string"`
	HighestBid    float64 `json:"highestBid,string"`
	PercentChange float64 `json:"percentChange,string"`
	BaseVolume    float64 `json:"baseVolume,string"`
	QuoteVolume   float64 `json:"quoteVolume,string"`
	IsFrozen      int     `json:"isFrozen,string"`
	High24Hr      float64 `json:"high24hr,string"`
	Low24Hr       float64 `json:"low24hr,string"`
}

// OrderbookResponseAll holds the full response type orderbook
type OrderbookResponseAll struct {
	Data map[string]OrderbookResponse
}

// CompleteBalances holds the full balance data
type CompleteBalances struct {
	Currency map[string]CompleteBalance
}

// OrderbookResponse is a sub-type for orderbooks
type OrderbookResponse struct {
	Asks     [][]interface{} `json:"asks"`
	Bids     [][]interface{} `json:"bids"`
	IsFrozen string          `json:"isFrozen"`
	Error    string          `json:"error"`
}

// OrderbookItem holds data on an individual item
type OrderbookItem struct {
	Price  float64
	Amount float64
}

// OrderbookAll contains the full range of orderbooks
type OrderbookAll struct {
	Data map[string]Orderbook
}

// Orderbook is a generic type golding orderbook information
type Orderbook struct {
	Asks []OrderbookItem `json:"asks"`
	Bids []OrderbookItem `json:"bids"`
}

// TradeHistory holds trade history data
type TradeHistory struct {
	GlobalTradeID int64   `json:"globalTradeID"`
	TradeID       int64   `json:"tradeID"`
	Date          string  `json:"date"`
	Type          string  `json:"type"`
	Rate          float64 `json:"rate,string"`
	Amount        float64 `json:"amount,string"`
	Total         float64 `json:"total,string"`
}

// ChartData holds kline data
type ChartData struct {
	Date            int     `json:"date"`
	High            float64 `json:"high"`
	Low             float64 `json:"low"`
	Open            float64 `json:"open"`
	Close           float64 `json:"close"`
	Volume          float64 `json:"volume"`
	QuoteVolume     float64 `json:"quoteVolume"`
	WeightedAverage float64 `json:"weightedAverage"`
	Error           string  `json:"error"`
}

// Currencies contains currency information
type Currencies struct {
	Name               string      `json:"name"`
	MaxDailyWithdrawal string      `json:"maxDailyWithdrawal"`
	TxFee              float64     `json:"txFee,string"`
	MinConfirmations   int         `json:"minConf"`
	DepositAddresses   interface{} `json:"depositAddress"`
	Disabled           int         `json:"disabled"`
	Delisted           int         `json:"delisted"`
	Frozen             int         `json:"frozen"`
}

// LoanOrder holds loan order information
type LoanOrder struct {
	Rate     float64 `json:"rate,string"`
	Amount   float64 `json:"amount,string"`
	RangeMin int     `json:"rangeMin"`
	RangeMax int     `json:"rangeMax"`
}

// LoanOrders holds loan order information range
type LoanOrders struct {
	Offers  []LoanOrder `json:"offers"`
	Demands []LoanOrder `json:"demands"`
}

// Balance holds data for a range of currencies
type Balance struct {
	Currency map[string]float64
}

// CompleteBalance contains the complete balance with a btcvalue
type CompleteBalance struct {
	Available float64
	OnOrders  float64
	BTCValue  float64
}

// DepositAddresses holds the full address per crypto-currency
type DepositAddresses struct {
	Addresses map[string]string
}

// DepositsWithdrawals holds withdrawal information
type DepositsWithdrawals struct {
	Deposits []struct {
		Currency      string  `json:"currency"`
		Address       string  `json:"address"`
		Amount        float64 `json:"amount,string"`
		Confirmations int     `json:"confirmations"`
		TransactionID string  `json:"txid"`
		Timestamp     int64   `json:"timestamp"`
		Status        string  `json:"status"`
	} `json:"deposits"`
	Withdrawals []struct {
		WithdrawalNumber int64   `json:"withdrawalNumber"`
		Currency         string  `json:"currency"`
		Address          string  `json:"address"`
		Amount           float64 `json:"amount,string"`
		Confirmations    int     `json:"confirmations"`
		TransactionID    string  `json:"txid"`
		Timestamp        int64   `json:"timestamp"`
		Status           string  `json:"status"`
		IPAddress        string  `json:"ipAddress"`
	} `json:"withdrawals"`
}

// Order hold order information
type Order struct {
	OrderNumber int64   `json:"orderNumber,string"`
	Type        string  `json:"type"`
	Rate        float64 `json:"rate,string"`
	Amount      float64 `json:"amount,string"`
	Total       float64 `json:"total,string"`
	Date        string  `json:"date"`
	Margin      float64 `json:"margin"`
}

// OpenOrdersResponseAll holds all open order responses
type OpenOrdersResponseAll struct {
	Data map[string][]Order
}

// OpenOrdersResponse holds open response orders
type OpenOrdersResponse struct {
	Data []Order
}

// AuthentictedTradeHistory holds client trade history information
type AuthentictedTradeHistory struct {
	GlobalTradeID int64   `json:"globalTradeID"`
	TradeID       int64   `json:"tradeID,string"`
	Date          string  `json:"date"`
	Rate          float64 `json:"rate,string"`
	Amount        float64 `json:"amount,string"`
	Total         float64 `json:"total,string"`
	Fee           float64 `json:"fee,string"`
	OrderNumber   int64   `json:"orderNumber,string"`
	Type          string  `json:"type"`
	Category      string  `json:"category"`
}

// AuthenticatedTradeHistoryAll holds the full client trade history
type AuthenticatedTradeHistoryAll struct {
	Data map[string][]AuthentictedTradeHistory
}

// AuthenticatedTradeHistoryResponse is a response type for trade history
type AuthenticatedTradeHistoryResponse struct {
	Data []AuthentictedTradeHistory
}

// ResultingTrades holds resultant trade information
type ResultingTrades struct {
	Amount  float64 `json:"amount,string"`
	Date    string  `json:"date"`
	Rate    float64 `json:"rate,string"`
	Total   float64 `json:"total,string"`
	TradeID int64   `json:"tradeID,string"`
	Type    string  `json:"type"`
}

// OrderResponse is a response type of trades
type OrderResponse struct {
	OrderNumber int64             `json:"orderNumber,string"`
	Trades      []ResultingTrades `json:"resultingTrades"`
}

// GenericResponse is a response type for exchange generic responses
type GenericResponse struct {
	Success int    `json:"success"`
	Error   string `json:"error"`
}

// MoveOrderResponse is a response type for move order trades
type MoveOrderResponse struct {
	Success     int                          `json:"success"`
	Error       string                       `json:"error"`
	OrderNumber int64                        `json:"orderNumber,string"`
	Trades      map[string][]ResultingTrades `json:"resultingTrades"`
}

// Withdraw holds withdraw information
type Withdraw struct {
	Response string `json:"response"`
	Error    string `json:"error"`
}

// Fee holds fees for specific trades
type Fee struct {
	MakerFee        float64 `json:"makerFee,string"`
	TakerFee        float64 `json:"takerFee,string"`
	ThirtyDayVolume float64 `json:"thirtyDayVolume,string"`
}

// Margin holds margin information
type Margin struct {
	TotalValue    float64 `json:"totalValue,string"`
	ProfitLoss    float64 `json:"pl,string"`
	LendingFees   float64 `json:"lendingFees,string"`
	NetValue      float64 `json:"netValue,string"`
	BorrowedValue float64 `json:"totalBorrowedValue,string"`
	CurrentMargin float64 `json:"currentMargin,string"`
}

// MarginPosition holds margin positional information
type MarginPosition struct {
	Amount            float64 `json:"amount,string"`
	Total             float64 `json:"total,string"`
	BasePrice         float64 `json:"basePrice,string"`
	LiquidiationPrice float64 `json:"liquidiationPrice"`
	ProfitLoss        float64 `json:"pl,string"`
	LendingFees       float64 `json:"lendingFees,string"`
	Type              string  `json:"type"`
}

// LoanOffer holds loan offer information
type LoanOffer struct {
	ID        int64   `json:"id"`
	Rate      float64 `json:"rate,string"`
	Amount    float64 `json:"amount,string"`
	Duration  int     `json:"duration"`
	AutoRenew bool    `json:"autoRenew,int"`
	Date      string  `json:"date"`
}

// ActiveLoans shows the full active loans on the exchange
type ActiveLoans struct {
	Provided []LoanOffer `json:"provided"`
	Used     []LoanOffer `json:"used"`
}

// LendingHistory holds the full lending history data
type LendingHistory struct {
	ID       int64   `json:"id"`
	Currency string  `json:"currency"`
	Rate     float64 `json:"rate,string"`
	Amount   float64 `json:"amount,string"`
	Duration float64 `json:"duration,string"`
	Interest float64 `json:"interest,string"`
	Fee      float64 `json:"fee,string"`
	Earned   float64 `json:"earned,string"`
	Open     string  `json:"open"`
	Close    string  `json:"close"`
}

// WebsocketTicker holds ticker data for the websocket
type WebsocketTicker struct {
	CurrencyPair  string
	Last          float64
	LowestAsk     float64
	HighestBid    float64
	PercentChange float64
	BaseVolume    float64
	QuoteVolume   float64
	IsFrozen      bool
	High          float64
	Low           float64
}

// WebsocketTrollboxMessage holds trollbox messages and information for
// websocket
type WebsocketTrollboxMessage struct {
	MessageNumber float64
	Username      string
	Message       string
	Reputation    float64
}

// WsCommand defines the request params after a websocket connection has been
// established
type WsCommand struct {
	Command string      `json:"command"`
	Channel interface{} `json:"channel"`
	APIKey  string      `json:"key,omitempty"`
	Payload string      `json:"payload,omitempty"`
	Sign    string      `json:"sign,omitempty"`
}

// WsTicker defines the websocket ticker response
type WsTicker struct {
	LastPrice              float64
	LowestAsk              float64
	HighestBid             float64
	PercentageChange       float64
	BaseCurrencyVolume24H  float64
	QuoteCurrencyVolume24H float64
	IsFrozen               bool
	HighestTradeIn24H      float64
	LowestTradePrice24H    float64
}

// WsTrade defines the websocket trade response
type WsTrade struct {
	Symbol    string
	TradeID   int64
	Side      string
	Volume    float64
	Price     float64
	Timestamp int64
}

// WithdrawalFees the large list of predefined withdrawal fees
// Prone to change, using highest value
var WithdrawalFees = map[string]float64{
	symbol.ZRX:   5,
	symbol.ARDR:  2,
	symbol.REP:   0.1,
	symbol.BTC:   0.0005,
	symbol.BCH:   0.0001,
	symbol.XBC:   0.0001,
	symbol.BTCD:  0.01,
	symbol.BTM:   0.01,
	symbol.BTS:   5,
	symbol.BURST: 1,
	symbol.BCN:   1,
	symbol.CVC:   1,
	symbol.CLAM:  0.001,
	symbol.XCP:   1,
	symbol.DASH:  0.01,
	symbol.DCR:   0.1,
	symbol.DGB:   0.1,
	symbol.DOGE:  5,
	symbol.EMC2:  0.01,
	symbol.EOS:   0,
	symbol.ETH:   0.01,
	symbol.ETC:   0.01,
	symbol.EXP:   0.01,
	symbol.FCT:   0.01,
	symbol.GAME:  0.01,
	symbol.GAS:   0,
	symbol.GNO:   0.015,
	symbol.GNT:   1,
	symbol.GRC:   0.01,
	symbol.HUC:   0.01,
	symbol.LBC:   0.05,
	symbol.LSK:   0.1,
	symbol.LTC:   0.001,
	symbol.MAID:  10,
	symbol.XMR:   0.015,
	symbol.NMC:   0.01,
	symbol.NAV:   0.01,
	symbol.XEM:   15,
	symbol.NEOS:  0.0001,
	symbol.NXT:   1,
	symbol.OMG:   0.3,
	symbol.OMNI:  0.1,
	symbol.PASC:  0.01,
	symbol.PPC:   0.01,
	symbol.POT:   0.01,
	symbol.XPM:   0.01,
	symbol.XRP:   0.15,
	symbol.SC:    10,
	symbol.STEEM: 0.01,
	symbol.SBD:   0.01,
	symbol.XLM:   0.00001,
	symbol.STORJ: 1,
	symbol.STRAT: 0.01,
	symbol.AMP:   5,
	symbol.SYS:   0.01,
	symbol.USDT:  10,
	symbol.VRC:   0.01,
	symbol.VTC:   0.001,
	symbol.VIA:   0.01,
	symbol.ZEC:   0.001,
}
