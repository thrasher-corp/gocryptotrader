package bitfinex

type BitfinexStats struct {
	Period int64
	Volume float64 `json:",string"`
}

type BitfinexTicker struct {
	Mid       float64 `json:",string"`
	Bid       float64 `json:",string"`
	Ask       float64 `json:",string"`
	Last      float64 `json:"Last_price,string"`
	Low       float64 `json:",string"`
	High      float64 `json:",string"`
	Volume    float64 `json:",string"`
	Timestamp string
}

type BitfinexMarginLimits struct {
	On_Pair           string
	InitialMargin     float64 `json:"initial_margin,string"`
	MarginRequirement float64 `json:"margin_requirement,string"`
	TradableBalance   float64 `json:"tradable_balance,string"`
}

type BitfinexMarginInfo struct {
	MarginBalance     float64                `json:"margin_balance,string"`
	TradableBalance   float64                `json:"tradable_balance,string"`
	UnrealizedPL      int64                  `json:"unrealized_pl"`
	UnrealizedSwap    int64                  `json:"unrealized_swap"`
	NetValue          float64                `json:"net_value,string"`
	RequiredMargin    int64                  `json:"required_margin"`
	Leverage          float64                `json:"leverage,string"`
	MarginRequirement float64                `json:"margin_requirement,string"`
	MarginLimits      []BitfinexMarginLimits `json:"margin_limits"`
	Message           string
}

type BitfinexOrder struct {
	ID                    int64
	Symbol                string
	Exchange              string
	Price                 float64 `json:"price,string"`
	AverageExecutionPrice float64 `json:"avg_execution_price,string"`
	Side                  string
	Type                  string
	Timestamp             string
	IsLive                bool    `json:"is_live"`
	IsCancelled           bool    `json:"is_cancelled"`
	IsHidden              bool    `json:"is_hidden"`
	WasForced             bool    `json:"was_forced"`
	OriginalAmount        float64 `json:"original_amount,string"`
	RemainingAmount       float64 `json:"remaining_amount,string"`
	ExecutedAmount        float64 `json:"executed_amount,string"`
	OrderID               int64   `json:"order_id"`
}

type BitfinexPlaceOrder struct {
	Symbol   string  `json:"symbol"`
	Amount   float64 `json:"amount,string"`
	Price    float64 `json:"price,string"`
	Exchange string  `json:"exchange"`
	Side     string  `json:"side"`
	Type     string  `json:"type"`
}

type BitfinexBalance struct {
	Type      string
	Currency  string
	Amount    float64 `json:"amount,string"`
	Available float64 `json:"available,string"`
}

type BitfinexOffer struct {
	ID              int64
	Currency        string
	Rate            float64 `json:"rate,string"`
	Period          int64
	Direction       string
	Timestamp       string
	Type            string
	IsLive          bool    `json:"is_live"`
	IsCancelled     bool    `json:"is_cancelled"`
	OriginalAmount  float64 `json:"original_amount,string"`
	RemainingAmount float64 `json:"remaining_amount,string"`
	ExecutedAmount  float64 `json:"remaining_amount,string"`
}

type BitfinexBookStructure struct {
	Price, Amount, Timestamp string
}

type BitfinexFee struct {
	Currency  string
	TakerFees float64
	MakerFees float64
}

type BitfinexOrderbook struct {
	Bids []BitfinexBookStructure
	Asks []BitfinexBookStructure
}

type BitfinexTradeStructure struct {
	Timestamp, Tid                int64
	Price, Amount, Exchange, Type string
}

type BitfinexSymbolDetails struct {
	Pair             string  `json:"pair"`
	PricePrecision   int     `json:"price_precision"`
	InitialMargin    float64 `json:"initial_margin,string"`
	MinimumMargin    float64 `json:"minimum_margin,string"`
	MaximumOrderSize float64 `json:"maximum_order_size,string"`
	MinimumOrderSize float64 `json:"minimum_order_size,string"`
	Expiration       string  `json:"expiration"`
}

type BitfinexLends struct {
	Rate       float64 `json:"rate,string"`
	AmountLent float64 `json:"amount_lent,string"`
	AmountUsed float64 `json:"amount_used,string"`
	Timestamp  int64   `json:"timestamp"`
}

type BitfinexAccountInfo struct {
	MakerFees string `json:"maker_fees"`
	TakerFees string `json:"taker_fees"`
	Fees      []struct {
		Pairs     string `json:"pairs"`
		MakerFees string `json:"maker_fees"`
		TakerFees string `json:"taker_fees"`
	} `json:"fees"`
}

type BitfinexDepositResponse struct {
	Result   string `json:"string"`
	Method   string `json:"method"`
	Currency string `json:"currency"`
	Address  string `json:"address"`
}

type BitfinexOrderMultiResponse struct {
	Orders []BitfinexOrder `json:"order_ids"`
	Status string          `json:"status"`
}

type BitfinexLendbookBidAsk struct {
	Rate            float64 `json:"rate,string"`
	Amount          float64 `json:"amount,string"`
	Period          int     `json:"period"`
	Timestamp       string  `json:"timestamp"`
	FlashReturnRate string  `json:"frr"`
}

type BitfinexLendbook struct {
	Bids []BitfinexLendbookBidAsk `json:"bids"`
	Asks []BitfinexLendbookBidAsk `json:"asks"`
}

type BitfinexPosition struct {
	ID        int64   `json:"id"`
	Symbol    string  `json:"string"`
	Status    string  `json:"active"`
	Base      float64 `json:"base,string"`
	Amount    float64 `json:"amount,string"`
	Timestamp string  `json:"timestamp"`
	Swap      float64 `json:"swap,string"`
	PL        float64 `json:"pl,string"`
}

type BitfinexBalanceHistory struct {
	Currency    string  `json:"currency"`
	Amount      float64 `json:"amount,string"`
	Balance     float64 `json:"balance,string"`
	Description string  `json:"description"`
	Timestamp   string  `json:"timestamp"`
}

type BitfinexMovementHistory struct {
	ID          int64   `json:"id"`
	Currency    string  `json:"currency"`
	Method      string  `json:"method"`
	Type        string  `json:"withdrawal"`
	Amount      float64 `json:"amount,string"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	Timestamp   string  `json:"timestamp"`
}

type BitfinexTradeHistory struct {
	Price       float64 `json:"price,string"`
	Amount      float64 `json:"amount,string"`
	Timestamp   string  `json:"timestamp"`
	Exchange    string  `json:"exchange"`
	Type        string  `json:"type"`
	FeeCurrency string  `json:"fee_currency"`
	FeeAmount   float64 `json:"fee_amount,string"`
	TID         int64   `json:"tid"`
	OrderID     int64   `json:"order_id"`
}

type BitfinexMarginFunds struct {
	ID         int64   `json:"id"`
	PositionID int64   `json:"position_id"`
	Currency   string  `json:"currency"`
	Rate       float64 `json:"rate,string"`
	Period     int     `json:"period"`
	Amount     float64 `json:"amount,string"`
	Timestamp  string  `json:"timestamp"`
}

type BitfinexMarginTotalTakenFunds struct {
	PositionPair string  `json:"position_pair"`
	TotalSwaps   float64 `json:"total_swaps,string"`
}

type BitfinexWalletTransfer struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type BitfinexWithdrawal struct {
	Status       string `json:"status"`
	Message      string `json:"message"`
	WithdrawalID int64  `json:"withdrawal_id"`
}

type BitfinexGenericResponse struct {
	Result string `json:"result"`
}

type BitfinexWebsocketChanInfo struct {
	Channel string
	Pair    string
}

type BitfinexWebsocketBook struct {
	Price  float64
	Count  int
	Amount float64
}

type BitfinexWebsocketTrade struct {
	ID        int64
	Timestamp int64
	Price     float64
	Amount    float64
}

type BitfinexWebsocketTicker struct {
	Bid             float64
	BidSize         float64
	Ask             float64
	AskSize         float64
	DailyChange     float64
	DialyChangePerc float64
	LastPrice       float64
	Volume          float64
}

type BitfinexWebsocketPosition struct {
	Pair              string
	Status            string
	Amount            float64
	Price             float64
	MarginFunding     float64
	MarginFundingType int
}

type BitfinexWebsocketWallet struct {
	Name              string
	Currency          string
	Balance           float64
	UnsettledInterest float64
}

type BitfinexWebsocketOrder struct {
	OrderID    int64
	Pair       string
	Amount     float64
	OrigAmount float64
	OrderType  string
	Status     string
	Price      float64
	PriceAvg   float64
	Timestamp  string
	Notify     int
}

type BitfinexWebsocketTradeExecuted struct {
	TradeID        int64
	Pair           string
	Timestamp      int64
	OrderID        int64
	AmountExecuted float64
	PriceExecuted  float64
}
