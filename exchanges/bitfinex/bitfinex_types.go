package bitfinex

import "github.com/kempeng/gocryptotrader/decimal"

// Ticker holds basic ticker information from the exchange
type Ticker struct {
	Mid       decimal.Decimal `json:"mid,string"`
	Bid       decimal.Decimal `json:"bid,string"`
	Ask       decimal.Decimal `json:"ask,string"`
	Last      decimal.Decimal `json:"last_price,string"`
	Low       decimal.Decimal `json:"low,string"`
	High      decimal.Decimal `json:"high,string"`
	Volume    decimal.Decimal `json:"volume,string"`
	Timestamp string          `json:"timestamp"`
	Message   string          `json:"message"`
}

// Tickerv2 holds the version 2 ticker information
type Tickerv2 struct {
	FlashReturnRate decimal.Decimal
	Bid             decimal.Decimal
	BidPeriod       int64
	BidSize         decimal.Decimal
	Ask             decimal.Decimal
	AskPeriod       int64
	AskSize         decimal.Decimal
	DailyChange     decimal.Decimal
	DailyChangePerc decimal.Decimal
	Last            decimal.Decimal
	Volume          decimal.Decimal
	High            decimal.Decimal
	Low             decimal.Decimal
}

// Tickersv2 holds the version 2 tickers information
type Tickersv2 struct {
	Symbol string
	Tickerv2
}

// Stat holds individual statistics from exchange
type Stat struct {
	Period int64           `json:"period"`
	Volume decimal.Decimal `json:"volume,string"`
}

// FundingBook holds current the full margin funding book
type FundingBook struct {
	Bids    []Book `json:"bids"`
	Asks    []Book `json:"asks"`
	Message string `json:"message"`
}

// Orderbook holds orderbook information from bid and ask sides
type Orderbook struct {
	Bids []Book
	Asks []Book
}

// BookV2 holds the orderbook item
type BookV2 struct {
	Price  decimal.Decimal
	Rate   decimal.Decimal
	Period decimal.Decimal
	Count  int64
	Amount decimal.Decimal
}

// OrderbookV2 holds orderbook information from bid and ask sides
type OrderbookV2 struct {
	Bids []BookV2
	Asks []BookV2
}

// TradeStructure holds executed trade information
type TradeStructure struct {
	Timestamp int64           `json:"timestamp"`
	Tid       int64           `json:"tid"`
	Price     decimal.Decimal `json:"price,string"`
	Amount    decimal.Decimal `json:"amount,string"`
	Exchange  string          `json:"exchange"`
	Type      string          `json:"sell"`
}

// TradeStructureV2 holds resp information
type TradeStructureV2 struct {
	Timestamp int64
	TID       int64
	Price     decimal.Decimal
	Amount    decimal.Decimal
	Exchange  string
	Type      string
}

// Lendbook holds most recent funding data for a relevant currency
type Lendbook struct {
	Bids []Book `json:"bids"`
	Asks []Book `json:"asks"`
}

// Book is a generalised sub-type to hold book information
type Book struct {
	Price           decimal.Decimal `json:"price,string"`
	Rate            decimal.Decimal `json:"rate,string"`
	Amount          decimal.Decimal `json:"amount,string"`
	Period          int             `json:"period"`
	Timestamp       string          `json:"timestamp"`
	FlashReturnRate string          `json:"frr"`
}

// Lends holds the lent information by currency
type Lends struct {
	Rate       decimal.Decimal `json:"rate,string"`
	AmountLent decimal.Decimal `json:"amount_lent,string"`
	AmountUsed decimal.Decimal `json:"amount_used,string"`
	Timestamp  int64           `json:"timestamp"`
}

// SymbolDetails holds currency pair information
type SymbolDetails struct {
	Pair             string          `json:"pair"`
	PricePrecision   int             `json:"price_precision"`
	InitialMargin    decimal.Decimal `json:"initial_margin,string"`
	MinimumMargin    decimal.Decimal `json:"minimum_margin,string"`
	MaximumOrderSize decimal.Decimal `json:"maximum_order_size,string"`
	MinimumOrderSize decimal.Decimal `json:"minimum_order_size,string"`
	Expiration       string          `json:"expiration"`
}

// AccountInfoFull adds the error message to Account info
type AccountInfoFull struct {
	Info    []AccountInfo
	Message string `json:"message"`
}

// AccountInfo general account information with fees
type AccountInfo struct {
	MakerFees string `json:"maker_fees"`
	TakerFees string `json:"taker_fees"`
	Fees      []struct {
		Pairs     string `json:"pairs"`
		MakerFees string `json:"maker_fees"`
		TakerFees string `json:"taker_fees"`
	} `json:"fees"`
	Message string `json:"message"`
}

// AccountFees stores withdrawal account fee data from Bitfinex
type AccountFees struct {
	Withdraw struct {
		BTC decimal.Decimal `json:"BTC,string"`
		LTC decimal.Decimal `json:"LTC,string"`
		ETH decimal.Decimal `json:"ETH,string"`
		ETC decimal.Decimal `json:"ETC,string"`
		ZEC decimal.Decimal `json:"ZEC,string"`
		XMR decimal.Decimal `json:"XMR,string"`
		DSH decimal.Decimal `json:"DSH,string"`
		XRP decimal.Decimal `json:"XRP,string"`
		IOT decimal.Decimal `json:"IOT"`
		EOS decimal.Decimal `json:"EOS,string"`
		SAN decimal.Decimal `json:"SAN,string"`
		OMG decimal.Decimal `json:"OMG,string"`
		BCH decimal.Decimal `json:"BCH,string"`
	} `json:"withdraw"`
}

// AccountSummary holds account summary data
type AccountSummary struct {
	TradeVolumePer30D []Currency      `json:"trade_vol_30d"`
	FundingProfit30D  []Currency      `json:"funding_profit_30d"`
	MakerFee          decimal.Decimal `json:"maker_fee"`
	TakerFee          decimal.Decimal `json:"taker_fee"`
}

// Currency is a sub-type for AccountSummary data
type Currency struct {
	Currency string          `json:"curr"`
	Volume   decimal.Decimal `json:"vol,string"`
	Amount   decimal.Decimal `json:"amount,string"`
}

// DepositResponse holds deposit address information
type DepositResponse struct {
	Result   string `json:"string"`
	Method   string `json:"method"`
	Currency string `json:"currency"`
	Address  string `json:"address"`
}

// KeyPermissions holds the key permissions for the API key set
type KeyPermissions struct {
	Account   Permission `json:"account"`
	History   Permission `json:"history"`
	Orders    Permission `json:"orders"`
	Positions Permission `json:"positions"`
	Funding   Permission `json:"funding"`
	Wallets   Permission `json:"wallets"`
	Withdraw  Permission `json:"withdraw"`
}

// Permission sub-type for KeyPermissions
type Permission struct {
	Read  bool `json:"read"`
	Write bool `json:"write"`
}

// MarginInfo holds metadata for margin information from bitfinex
type MarginInfo struct {
	Info    MarginData
	Message string `json:"message"`
}

// MarginData holds wallet information for margin trading
type MarginData struct {
	MarginBalance     decimal.Decimal `json:"margin_balance,string"`
	TradableBalance   decimal.Decimal `json:"tradable_balance,string"`
	UnrealizedPL      int64           `json:"unrealized_pl"`
	UnrealizedSwap    int64           `json:"unrealized_swap"`
	NetValue          decimal.Decimal `json:"net_value,string"`
	RequiredMargin    int64           `json:"required_margin"`
	Leverage          decimal.Decimal `json:"leverage,string"`
	MarginRequirement decimal.Decimal `json:"margin_requirement,string"`
	MarginLimits      []MarginLimits  `json:"margin_limits"`
}

// MarginLimits holds limit data per pair
type MarginLimits struct {
	OnPair            string          `json:"on_pair"`
	InitialMargin     decimal.Decimal `json:"initial_margin,string"`
	MarginRequirement decimal.Decimal `json:"margin_requirement,string"`
	TradableBalance   decimal.Decimal `json:"tradable_balance,string"`
}

// Balance holds current balance data
type Balance struct {
	Type      string          `json:"type"`
	Currency  string          `json:"currency"`
	Amount    decimal.Decimal `json:"amount,string"`
	Available decimal.Decimal `json:"available,string"`
}

// WalletTransfer holds status of wallet to wallet content transfer on exchange
type WalletTransfer struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Withdrawal holds withdrawal status information
type Withdrawal struct {
	Status       string `json:"status"`
	Message      string `json:"message"`
	WithdrawalID int64  `json:"withdrawal_id,string"`
}

// Order holds order information when an order is in the market
type Order struct {
	ID                    int64           `json:"id"`
	Symbol                string          `json:"symbol"`
	Exchange              string          `json:"exchange"`
	Price                 decimal.Decimal `json:"price,string"`
	AverageExecutionPrice decimal.Decimal `json:"avg_execution_price,string"`
	Side                  string          `json:"side"`
	Type                  string          `json:"type"`
	Timestamp             string          `json:"timestamp"`
	IsLive                bool            `json:"is_live"`
	IsCancelled           bool            `json:"is_cancelled"`
	IsHidden              bool            `json:"is_hidden"`
	WasForced             bool            `json:"was_forced"`
	OriginalAmount        decimal.Decimal `json:"original_amount,string"`
	RemainingAmount       decimal.Decimal `json:"remaining_amount,string"`
	ExecutedAmount        decimal.Decimal `json:"executed_amount,string"`
	OrderID               int64           `json:"order_id"`
}

// OrderMultiResponse holds order information on the executed orders
type OrderMultiResponse struct {
	Orders []Order `json:"order_ids"`
	Status string  `json:"status"`
}

// PlaceOrder is used for order placement
type PlaceOrder struct {
	Symbol   string          `json:"symbol"`
	Amount   decimal.Decimal `json:"amount,string"`
	Price    decimal.Decimal `json:"price,string"`
	Exchange string          `json:"exchange"`
	Side     string          `json:"side"`
	Type     string          `json:"type"`
}

// GenericResponse holds the result for a generic response
type GenericResponse struct {
	Result string `json:"result"`
}

// Position holds position information
type Position struct {
	ID        int64           `json:"id"`
	Symbol    string          `json:"string"`
	Status    string          `json:"active"`
	Base      decimal.Decimal `json:"base,string"`
	Amount    decimal.Decimal `json:"amount,string"`
	Timestamp string          `json:"timestamp"`
	Swap      decimal.Decimal `json:"swap,string"`
	PL        decimal.Decimal `json:"pl,string"`
}

// BalanceHistory holds balance history information
type BalanceHistory struct {
	Currency    string          `json:"currency"`
	Amount      decimal.Decimal `json:"amount,string"`
	Balance     decimal.Decimal `json:"balance,string"`
	Description string          `json:"description"`
	Timestamp   string          `json:"timestamp"`
}

// MovementHistory holds deposit and withdrawal history data
type MovementHistory struct {
	ID               int64           `json:"id"`
	TxID             int64           `json:"txid"`
	Currency         string          `json:"currency"`
	Method           string          `json:"method"`
	Type             string          `json:"withdrawal"`
	Amount           decimal.Decimal `json:"amount,string"`
	Description      string          `json:"description"`
	Address          string          `json:"address"`
	Status           string          `json:"status"`
	Timestamp        string          `json:"timestamp"`
	TimestampCreated string          `json:"timestamp_created"`
	Fee              decimal.Decimal `json:"fee"`
}

// TradeHistory holds trade history data
type TradeHistory struct {
	Price       decimal.Decimal `json:"price,string"`
	Amount      decimal.Decimal `json:"amount,string"`
	Timestamp   string          `json:"timestamp"`
	Exchange    string          `json:"exchange"`
	Type        string          `json:"type"`
	FeeCurrency string          `json:"fee_currency"`
	FeeAmount   decimal.Decimal `json:"fee_amount,string"`
	TID         int64           `json:"tid"`
	OrderID     int64           `json:"order_id"`
}

// Offer holds offer information
type Offer struct {
	ID              int64           `json:"id"`
	Currency        string          `json:"currency"`
	Rate            decimal.Decimal `json:"rate,string"`
	Period          int64           `json:"period"`
	Direction       string          `json:"direction"`
	Timestamp       string          `json:"timestamp"`
	Type            string          `json:"type"`
	IsLive          bool            `json:"is_live"`
	IsCancelled     bool            `json:"is_cancelled"`
	OriginalAmount  decimal.Decimal `json:"original_amount,string"`
	RemainingAmount decimal.Decimal `json:"remaining_amount,string"`
	ExecutedAmount  decimal.Decimal `json:"executed_amount,string"`
}

// MarginFunds holds active funding information used in a margin position
type MarginFunds struct {
	ID         int64           `json:"id"`
	PositionID int64           `json:"position_id"`
	Currency   string          `json:"currency"`
	Rate       decimal.Decimal `json:"rate,string"`
	Period     int             `json:"period"`
	Amount     decimal.Decimal `json:"amount,string"`
	Timestamp  string          `json:"timestamp"`
	AutoClose  bool            `json:"auto_close"`
}

// MarginTotalTakenFunds holds position funding including sum of active backing
// as total swaps
type MarginTotalTakenFunds struct {
	PositionPair string          `json:"position_pair"`
	TotalSwaps   decimal.Decimal `json:"total_swaps,string"`
}

// Fee holds fee data for a specified currency
type Fee struct {
	Currency  string
	TakerFees decimal.Decimal
	MakerFees decimal.Decimal
}

// WebsocketChanInfo holds websocket channel information
type WebsocketChanInfo struct {
	Channel string
	Pair    string
}

// WebsocketBook holds booking information
type WebsocketBook struct {
	Price  decimal.Decimal
	Count  int
	Amount decimal.Decimal
}

// WebsocketTrade holds trade information
type WebsocketTrade struct {
	ID        int64
	Timestamp int64
	Price     decimal.Decimal
	Amount    decimal.Decimal
}

// WebsocketTicker holds ticker information
type WebsocketTicker struct {
	Bid             decimal.Decimal
	BidSize         decimal.Decimal
	Ask             decimal.Decimal
	AskSize         decimal.Decimal
	DailyChange     decimal.Decimal
	DialyChangePerc decimal.Decimal
	LastPrice       decimal.Decimal
	Volume          decimal.Decimal
}

// WebsocketPosition holds position information
type WebsocketPosition struct {
	Pair              string
	Status            string
	Amount            decimal.Decimal
	Price             decimal.Decimal
	MarginFunding     decimal.Decimal
	MarginFundingType int
}

// WebsocketWallet holds wallet information
type WebsocketWallet struct {
	Name              string
	Currency          string
	Balance           decimal.Decimal
	UnsettledInterest decimal.Decimal
}

// WebsocketOrder holds order data
type WebsocketOrder struct {
	OrderID    int64
	Pair       string
	Amount     decimal.Decimal
	OrigAmount decimal.Decimal
	OrderType  string
	Status     string
	Price      decimal.Decimal
	PriceAvg   decimal.Decimal
	Timestamp  string
	Notify     int
}

// WebsocketTradeExecuted holds executed trade data
type WebsocketTradeExecuted struct {
	TradeID        int64
	Pair           string
	Timestamp      int64
	OrderID        int64
	AmountExecuted decimal.Decimal
	PriceExecuted  decimal.Decimal
}

// ErrorCapture is a simple type for returned errors from Bitfinex
type ErrorCapture struct {
	Message string `json:"message"`
}
