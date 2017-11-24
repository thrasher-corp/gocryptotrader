package okcoin

type OKCoinTicker struct {
	Buy  float64 `json:",string"`
	High float64 `json:",string"`
	Last float64 `json:",string"`
	Low  float64 `json:",string"`
	Sell float64 `json:",string"`
	Vol  float64 `json:",string"`
}

type OKCoinTickerResponse struct {
	Date   string
	Ticker OKCoinTicker
}
type OKCoinFuturesTicker struct {
	Last        float64
	Buy         float64
	Sell        float64
	High        float64
	Low         float64
	Vol         float64
	Contract_ID int64
	Unit_Amount float64
}

type OKCoinOrderbook struct {
	Asks [][]float64 `json:"asks"`
	Bids [][]float64 `json:"bids"`
}

type OKCoinFuturesTickerResponse struct {
	Date   string
	Ticker OKCoinFuturesTicker
}

type OKCoinBorrowInfo struct {
	BorrowBTC        float64 `json:"borrow_btc"`
	BorrowLTC        float64 `json:"borrow_ltc"`
	BorrowCNY        float64 `json:"borrow_cny"`
	CanBorrow        float64 `json:"can_borrow"`
	InterestBTC      float64 `json:"interest_btc"`
	InterestLTC      float64 `json:"interest_ltc"`
	Result           bool    `json:"result"`
	DailyInterestBTC float64 `json:"today_interest_btc"`
	DailyInterestLTC float64 `json:"today_interest_ltc"`
	DailyInterestCNY float64 `json:"today_interest_cny"`
}

type OKCoinBorrowOrder struct {
	Amount      float64 `json:"amount"`
	BorrowDate  int64   `json:"borrow_date"`
	BorrowID    int64   `json:"borrow_id"`
	Days        int64   `json:"days"`
	TradeAmount float64 `json:"deal_amount"`
	Rate        float64 `json:"rate"`
	Status      int64   `json:"status"`
	Symbol      string  `json:"symbol"`
}

type OKCoinRecord struct {
	Address            string  `json:"addr"`
	Account            int64   `json:"account,string"`
	Amount             float64 `json:"amount"`
	Bank               string  `json:"bank"`
	BenificiaryAddress string  `json:"benificiary_addr"`
	TransactionValue   float64 `json:"transaction_value"`
	Fee                float64 `json:"fee"`
	Date               float64 `json:"date"`
}

type OKCoinAccountRecords struct {
	Records []OKCoinRecord `json:"records"`
	Symbol  string         `json:"symbol"`
}

type OKCoinFuturesOrder struct {
	Amount       float64 `json:"amount"`
	ContractName string  `json:"contract_name"`
	DateCreated  float64 `json:"create_date"`
	TradeAmount  float64 `json:"deal_amount"`
	Fee          float64 `json:"fee"`
	LeverageRate float64 `json:"lever_rate"`
	OrderID      int64   `json:"order_id"`
	Price        float64 `json:"price"`
	AvgPrice     float64 `json:"avg_price"`
	Status       float64 `json:"status"`
	Symbol       string  `json:"symbol"`
	Type         int64   `json:"type"`
	UnitAmount   int64   `json:"unit_amount"`
}

type OKCoinFuturesHoldAmount struct {
	Amount       float64 `json:"amount"`
	ContractName string  `json:"contract_name"`
}

type OKCoinFuturesExplosive struct {
	Amount      float64 `json:"amount,string"`
	DateCreated string  `json:"create_date"`
	Loss        float64 `json:"loss,string"`
	Type        int64   `json:"type"`
}

type OKCoinTrades struct {
	Amount  float64 `json:"amount,string"`
	Date    int64   `json:"date"`
	DateMS  int64   `json:"date_ms"`
	Price   float64 `json:"price,string"`
	TradeID int64   `json:"tid"`
	Type    string  `json:"type"`
}

type OKCoinFuturesTrades struct {
	Amount  float64 `json:"amount"`
	Date    int64   `json:"date"`
	DateMS  int64   `json:"date_ms"`
	Price   float64 `json:"price"`
	TradeID int64   `json:"tid"`
	Type    string  `json:"type"`
}

type OKCoinUserInfo struct {
	Info struct {
		Funds struct {
			Asset struct {
				Net   float64 `json:"net,string"`
				Total float64 `json:"total,string"`
			} `json:"asset"`
			Borrow struct {
				BTC float64 `json:"btc,string"`
				LTC float64 `json:"ltc,string"`
				USD float64 `json:"usd,string"`
				CNY float64 `json:"cny,string"`
			} `json:"borrow"`
			Free struct {
				BTC float64 `json:"btc,string"`
				LTC float64 `json:"ltc,string"`
				USD float64 `json:"usd,string"`
				CNY float64 `json:"cny,string"`
			} `json:"free"`
			Freezed struct {
				BTC float64 `json:"btc,string"`
				LTC float64 `json:"ltc,string"`
				USD float64 `json:"usd,string"`
				CNY float64 `json:"cny,string"`
			} `json:"freezed"`
			UnionFund struct {
				BTC float64 `json:"btc,string"`
				LTC float64 `json:"ltc,string"`
			} `json:"union_fund"`
		} `json:"funds"`
	} `json:"info"`
	Result bool `json:"result"`
}

type OKCoinBatchTrade struct {
	OrderInfo []struct {
		OrderID   int64 `json:"order_id"`
		ErrorCode int64 `json:"error_code"`
	} `json:"order_info"`
	Result bool `json:"result"`
}

type OKCoinCancelOrderResponse struct {
	Success string
	Error   string
}

type OKCoinOrderInfo struct {
	Amount     float64 `json:"amount"`
	AvgPrice   float64 `json:"avg_price"`
	Created    int64   `json:"create_date"`
	DealAmount float64 `json:"deal_amount"`
	OrderID    int64   `json:"order_id"`
	OrdersID   int64   `json:"orders_id"`
	Price      float64 `json:"price"`
	Status     int     `json:"status"`
	Symbol     string  `json:"symbol"`
	Type       string  `json:"type"`
}

type OKCoinOrderHistory struct {
	CurrentPage int               `json:"current_page"`
	Orders      []OKCoinOrderInfo `json:"orders"`
	PageLength  int               `json:"page_length"`
	Result      bool              `json:"result"`
	Total       int               `json:"total"`
}

type OKCoinWithdrawalResponse struct {
	WithdrawID int  `json:"withdraw_id"`
	Result     bool `json:"result"`
}

type OKCoinWithdrawInfo struct {
	Address    string  `json:"address"`
	Amount     float64 `json:"amount"`
	Created    int64   `json:"created_date"`
	ChargeFee  float64 `json:"chargefee"`
	Status     int     `json:"status"`
	WithdrawID int64   `json:"withdraw_id"`
}

type OKCoinOrderFeeInfo struct {
	Fee     float64 `json:"fee,string"`
	OrderID int64   `json:"order_id"`
	Type    string  `json:"type"`
}

type OKCoinLendDepth struct {
	Amount float64 `json:"amount"`
	Days   string  `json:"days"`
	Num    int64   `json:"num"`
	Rate   float64 `json:"rate,string"`
}

type OKCoinBorrowResponse struct {
	Result   bool `json:"result"`
	BorrowID int  `json:"borrow_id"`
}

type OKCoinWebsocketFutureIndex struct {
	FutureIndex float64 `json:"futureIndex"`
	Timestamp   int64   `json:"timestamp,string"`
}

type OKCoinWebsocketTicker struct {
	Timestamp float64
	Vol       string
	Buy       float64
	High      float64
	Last      float64
	Low       float64
	Sell      float64
}

type OKCoinWebsocketFuturesTicker struct {
	Buy        float64 `json:"buy"`
	ContractID string  `json:"contractId"`
	High       float64 `json:"high"`
	HoldAmount float64 `json:"hold_amount"`
	Last       float64 `json:"last,string"`
	Low        float64 `json:"low"`
	Sell       float64 `json:"sell"`
	UnitAmount float64 `json:"unitAmount"`
	Volume     float64 `json:"vol,string"`
}

type OKCoinWebsocketOrderbook struct {
	Asks      [][]float64 `json:"asks"`
	Bids      [][]float64 `json:"bids"`
	Timestamp int64       `json:"timestamp,string"`
}

type OKCoinWebsocketUserinfo struct {
	Info struct {
		Funds struct {
			Asset struct {
				Net   float64 `json:"net,string"`
				Total float64 `json:"total,string"`
			} `json:"asset"`
			Free struct {
				BTC float64 `json:"btc,string"`
				LTC float64 `json:"ltc,string"`
				USD float64 `json:"usd,string"`
				CNY float64 `json:"cny,string"`
			} `json:"free"`
			Frozen struct {
				BTC float64 `json:"btc,string"`
				LTC float64 `json:"ltc,string"`
				USD float64 `json:"usd,string"`
				CNY float64 `json:"cny,string"`
			} `json:"freezed"`
		} `json:"funds"`
	} `json:"info"`
	Result bool `json:"result"`
}

type OKCoinWebsocketFuturesContract struct {
	Available    float64 `json:"available"`
	Balance      float64 `json:"balance"`
	Bond         float64 `json:"bond"`
	ContractID   float64 `json:"contract_id"`
	ContractType string  `json:"contract_type"`
	Frozen       float64 `json:"freeze"`
	Profit       float64 `json:"profit"`
	Loss         float64 `json:"unprofit"`
}

type OKCoinWebsocketFuturesUserInfo struct {
	Info struct {
		BTC struct {
			Balance   float64                          `json:"balance"`
			Contracts []OKCoinWebsocketFuturesContract `json:"contracts"`
			Rights    float64                          `json:"rights"`
		} `json:"btc"`
		LTC struct {
			Balance   float64                          `json:"balance"`
			Contracts []OKCoinWebsocketFuturesContract `json:"contracts"`
			Rights    float64                          `json:"rights"`
		} `json:"ltc"`
	} `json:"info"`
	Result bool `json:"result"`
}

type OKCoinWebsocketOrder struct {
	Amount      float64 `json:"amount"`
	AvgPrice    float64 `json:"avg_price"`
	DateCreated float64 `json:"create_date"`
	TradeAmount float64 `json:"deal_amount"`
	OrderID     float64 `json:"order_id"`
	OrdersID    float64 `json:"orders_id"`
	Price       float64 `json:"price"`
	Status      int64   `json:"status"`
	Symbol      string  `json:"symbol"`
	OrderType   string  `json:"type"`
}

type OKCoinWebsocketFuturesOrder struct {
	Amount         float64 `json:"amount"`
	ContractName   string  `json:"contract_name"`
	DateCreated    float64 `json:"createdDate"`
	TradeAmount    float64 `json:"deal_amount"`
	Fee            float64 `json:"fee"`
	LeverageAmount int     `json:"lever_rate"`
	OrderID        float64 `json:"order_id"`
	Price          float64 `json:"price"`
	AvgPrice       float64 `json:"avg_price"`
	Status         int     `json:"status"`
	Symbol         string  `json:"symbol"`
	TradeType      int     `json:"type"`
	UnitAmount     float64 `json:"unit_amount"`
}

type OKCoinWebsocketRealtrades struct {
	AveragePrice         float64 `json:"averagePrice,string"`
	CompletedTradeAmount float64 `json:"completedTradeAmount,string"`
	DateCreated          float64 `json:"createdDate"`
	ID                   float64 `json:"id"`
	OrderID              float64 `json:"orderId"`
	SigTradeAmount       float64 `json:"sigTradeAmount,string"`
	SigTradePrice        float64 `json:"sigTradePrice,string"`
	Status               int64   `json:"status"`
	Symbol               string  `json:"symbol"`
	TradeAmount          float64 `json:"tradeAmount,string"`
	TradePrice           float64 `json:"buy,string"`
	TradeType            string  `json:"tradeType"`
	TradeUnitPrice       float64 `json:"tradeUnitPrice,string"`
	UnTrade              float64 `json:"unTrade,string"`
}

type OKCoinWebsocketFuturesRealtrades struct {
	Amount         float64 `json:"amount,string"`
	ContractID     float64 `json:"contract_id,string"`
	ContractName   string  `json:"contract_name"`
	ContractType   string  `json:"contract_type"`
	TradeAmount    float64 `json:"deal_amount,string"`
	Fee            float64 `json:"fee,string"`
	OrderID        float64 `json:"orderid"`
	Price          float64 `json:"price,string"`
	AvgPrice       float64 `json:"price_avg,string"`
	Status         int     `json:"status,string"`
	TradeType      int     `json:"type,string"`
	UnitAmount     float64 `json:"unit_amount,string"`
	LeverageAmount int     `json:"lever_rate,string"`
}

type OKCoinWebsocketEvent struct {
	Event   string `json:"event"`
	Channel string `json:"channel"`
}

type OKCoinWebsocketResponse struct {
	Channel string      `json:"channel"`
	Data    interface{} `json:"data"`
}

type OKCoinWebsocketEventAuth struct {
	Event      string            `json:"event"`
	Channel    string            `json:"channel"`
	Parameters map[string]string `json:"parameters"`
}

type OKCoinWebsocketEventAuthRemove struct {
	Event      string            `json:"event"`
	Channel    string            `json:"channel"`
	Parameters map[string]string `json:"parameters"`
}

type OKCoinWebsocketTradeOrderResponse struct {
	OrderID int64 `json:"order_id,string"`
	Result  bool  `json:"result,string"`
}
