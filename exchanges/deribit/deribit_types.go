package deribit

import "errors"

// UnmarshalError is the struct which is used for unmarshalling errors
type UnmarshalError struct {
	Message string `json:"message"`
	Data    struct {
		Reason string `json:"reason"`
		Code   int64  `json:"code"`
	}
}

var (
	errStartTimeCannotBeAfterEndTime = errors.New("start timestamp cannot be after end timestamp")
)

// BookSummaryData stores summary data
type BookSummaryData struct {
	VolumeUSD              float64 `json:"volume_usd"`
	Volume                 float64 `json:"volume"`
	QuoteCurrency          string  `json:"quote_currency"`
	PriceChange            float64 `json:"price_change"`
	OpenInterest           float64 `json:"open_interest"`
	MidPrice               float64 `json:"mid_price"`
	MarkPlace              float64 `json:"mark_place"`
	Low                    float64 `json:"low"`
	Last                   float64 `json:"last"`
	InstrumentName         string  `json:"instrument_name"`
	High                   float64 `json:"high"`
	EstimatedDeliveryPrice float64 `json:"estimated_delivery_price"`
	CreationTimestamp      int64   `json:"creation_timestamp"`
	BidPrice               float64 `json:"bid_price"`
	BaseCurrency           string  `json:"base_currency"`
	AskPrice               float64 `json:"ask_price"`
}

// ContractSizeData stores contract size for given instrument
type ContractSizeData struct {
	ContractSize float64 `json:"contract_size"`
}

// CurrencyData stores data for currencies
type CurrencyData struct {
	CoinType             string  `json:"coin_type"`
	Currency             string  `json:"currency"`
	CurrencyLong         string  `json:"currency_long"`
	FeePrecision         int64   `json:"fee_precision"`
	MinConfirmations     int64   `json:"min_confirmations"`
	MinWithdrawalFee     float64 `json:"min_withdrawal_fee"`
	WithdrawalFee        float64 `json:"withdrawal_fee"`
	WithdrawalPriorities []struct {
		Value float64 `json:"value"`
		Name  string  `json:"name"`
	} `json:"withdrawal_priorities"`
}

// FundingChartData stores futures funding chart data
type FundingChartData struct {
	CurrentInterest float64 `json:"current_interest"`
	Data            []struct {
		IndexPrice float64 `json:"index_price"`
		Interest8H float64 `json:"interest_8h"`
		Timestamp  int64   `json:"timestamp"`
	} `json:"data"`
}

//FundingRateHistoryData stores data for funding rate history
type FundingRateHistoryData struct {
	Timestamp      int64   `json:"timestamp"`
	IndexPrice     float64 `json:"index_price"`
	PrevIndexPrice float64 `json:"prev_index_price"`
	Interest8H     float64 `json:"interest_8h"`
	Interest1H     float64 `json:"interest_1h"`
}

// FundingRateValueData stores funding rate for the requested period
type FundingRateValueData struct {
	Result float64 `json:"result"`
}

// HistoricalVolatilityData stores volatility data for requested symbols
type HistoricalVolatilityData struct {
	Result struct {
		Timestamp int64
		Value     float64
	}
}

// IndexPriceData gets index price data
type IndexPriceData struct {
	EstimatedDeliveryPrice float64 `json:"estimated_delivery_price"`
	IndexPrice             float64 `json:"index_price"`
}

// InstrumentData gets data for instruments
type InstrumentData struct {
	TickSize             float64 `json:"tick_size"`
	TakerCommission      float64 `json:"taker_commission"`
	Strike               float64 `json:"strike"`
	SettlementPeriod     string  `json:"settlement_period"`
	QuoteCurrency        string  `json:"quote_currency"`
	OptionType           string  `json:"option_type"`
	MinimumTradeAmount   float64 `json:"min_trade_amount"`
	MakerCommission      float64 `json:"maker_commission"`
	Kind                 string  `json:"kind"`
	IsActive             bool    `json:"is_active"`
	InstrumentName       string  `json:"instrument_name"`
	ExpirationTimestamp  int64   `json:"expiration_timestamp"`
	CreationTimestamp    int64   `json:"creation_timestamp"`
	ContractSize         float64 `json:"contract_size"`
	BlockTradeCommission float64 `json:"block_trade_commission"`
	BaseCurrency         string  `json:"base_currency"`
}

// SettlementsData stores data for settlement futures
type SettlementsData struct {
	Settlements []struct {
		Type              string  `json:"type"`
		Timestamp         int64   `json:"timestamp"`
		SessionProfitLoss float64 `json:"session_profit_loss"`
		ProfitLoss        float64 `json:"profit_loss"`
		Position          float64 `json:"position"`
		MarkPrice         float64 `json:"mark_price"`
		InstrumentName    string  `json:"instrument_name"`
		IndexPrice        float64 `json:"index_price"`
	} `json:"settlements"`
	Continuation string `json:"continuation"`
}

// PublicTradesData stores data for public trades
type PublicTradesData struct {
	Trades []struct {
		TradeSeq       float64 `json:"trade_seq"`
		TradeID        string  `json:"trade_id"`
		Timestamp      int64   `json:"timestamp"`
		TickDirection  int64   `json:"tick_direction"`
		Price          float64 `json:"price"`
		MarkPrice      float64 `json:"mark_price"`
		InstrumentName string  `json:"instrument_name"`
		IndexPrice     float64 `json:"index_price"`
		Direction      string  `json:"direction"`
		Amount         float64 `json:"amount"`
	} `json:"trades"`
	HasMore bool `json:"has_more"`
}

// MarkPriceHistory stores data for mark price history
type MarkPriceHistory struct {
	Timestamp      int64
	MarkPriceValue float64
}

// OBData stores orderbook data
type OBData struct {
	Timestamp int64 `json:"timestamp"`
	Stats     struct {
		Volume      float64 `json:"volume"`
		PriceChange float64 `json:"price_change"`
		Low         float64 `json:"low"`
		High        float64 `json:"high"`
	} `json:"stats"`
	State           string       `json:"state"`
	SettlementPrice float64      `json:"settlement_price"`
	OpenInterest    float64      `json:"open_interest"`
	MinPrice        float64      `json:"min_price"`
	MaxPrice        float64      `json:"max_price"`
	MarkPrice       float64      `json:"mark_price"`
	LastPrice       float64      `json:"last_price"`
	InstrumentName  string       `json:"instrument_name"`
	IndexPrice      float64      `json:"index_price"`
	Funding8H       float64      `json:"funding_8h"`
	CurrentFunding  float64      `json:"current_funding"`
	ChangeID        int64        `json:"change_id"`
	Bids            [][2]float64 `json:"bids"`
	Asks            [][2]float64 `json:"asks"`
	BestBidPrice    float64      `json:"best_bid_price"`
	BestBidAmount   float64      `json:"best_bid_amount"`
	BestAskAmount   float64      `json:"best_ask_amount"`
}

// TradeVolumesData stores data for trade volumes
type TradeVolumesData struct {
	PutsVolume    float64 `json:"puts_volume"`
	FuturesVolume float64 `json:"futures_volume"`
	CurrencyPair  string  `json:"currency_pair"`
	CallsVolume   float64 `json:"calls_volume"`
}

// TVChartData stores trading view chart data
type TVChartData struct {
	Volume []float64 `json:"volume"`
	Cost   []float64 `json:"cost"`
	Ticks  []float64 `json:"ticks"`
	Status string    `json:"status"`
	Open   []float64 `json:"open"`
	Low    []float64 `json:"low"`
	High   []float64 `json:"high"`
	Close  []float64 `json:"close"`
}

// VolatilityIndexData stores index data for volatility
type VolatilityIndexData struct {
	Data [][]float64 `json:"data"`
}

// TickerData stores data for ticker
type TickerData struct {
	BestAskAmount   float64 `json:"best_ask_amount"`
	BestAskPrice    float64 `json:"best_ask_price"`
	BestBidAmount   float64 `json:"best_bid_amount"`
	BestBidPrice    float64 `json:"best_bid_price"`
	CurrentFunding  float64 `json:"current_funding"`
	Funding8H       float64 `json:"funding_8h"`
	IndexPrice      float64 `json:"index_price"`
	InstrumentName  string  `json:"instrument_name"`
	LastPrice       float64 `json:"last_price"`
	MarkPrice       float64 `json:"mark_price"`
	MaxPrice        float64 `json:"max_price"`
	MinPrice        float64 `json:"min_price"`
	OpenInterest    float64 `json:"open_interest"`
	SettlementPrice float64 `json:"settlement_price"`
	State           string  `json:"state"`
	Stats           struct {
		Volume      float64 `json:"volume"`
		PriceChange float64 `json:"price_change"`
		Low         float64 `json:"low"`
		High        float64 `json:"high"`
	} `json:"stats"`
	Timestamp int64 `json:"timestamp"`
}

// CancelTransferData stores data for a cancel transfer
type CancelTransferData struct {
	Amount           float64 `json:"amount"`
	CreatedTimestamp int64   `json:"created_timestamp"`
	Currency         string  `json:"currency"`
	Direction        string  `json:"direction"`
	ID               int64   `json:"id"`
	OtherSide        string  `json:"other_side"`
	State            string  `json:"state"`
	Type             string  `json:"type"`
	UpdatedTimestamp int64   `json:"updated_timestamp"`
}

// CancelWithdrawalData stores cancel request data for a withdrawal
type CancelWithdrawalData struct {
	Address            string  `json:"address"`
	Amount             float64 `json:"amount"`
	ConfirmedTimestamp int64   `json:"confirmed_timestamp"`
	CreatedTimestamp   int64   `json:"created_timestamp"`
	Currency           string  `json:"currency"`
	Fee                float64 `json:"fee"`
	ID                 int64   `json:"id"`
	Priority           float64 `json:"priority"`
	Status             string  `json:"status"`
	TransactionID      int64   `json:"transaction_id"`
	UpdatedTimestamp   int64   `json:"updated_timestamp"`
}

// CreateDepositAddressData stores data for creating a deposit address
type CreateDepositAddressData struct {
	Address           string `json:"address"`
	CreationTimestamp int64  `json:"creation_timestamp"`
	Currency          string `json:"currency"`
	Type              string `json:"type"`
}

// DepositsData stores data of deposits
type DepositsData struct {
	Count int64 `json:"count"`
	Data  []struct {
		Address           string  `json:"address"`
		Amount            float64 `json:"amount"`
		Currency          string  `json:"currency"`
		ReceivedTimestamp int64   `json:"receivedTimestamp"`
		State             string  `json:"state"`
		TransactionID     string  `json:"transaction_id"`
		UpdatedTimestamp  int64   `json:"updated_timestamp"`
	} `json:"data"`
}

// TransferData stores data for a transfer
type TransferData struct {
	Amount           float64 `json:"amount"`
	CreatedTimestamp int64   `json:"created_timestamp"`
	Currency         string  `json:"currency"`
	Direction        string  `json:"direction"`
	ID               int64   `json:"id"`
	OtherSide        string  `json:"other_side"`
	State            string  `json:"state"`
	Type             string  `json:"type"`
	UpdatedTimestamp int64   `json:"updated_timestamp"`
}

// TransfersData stores data of transfers
type TransfersData struct {
	Count int64          `json:"count"`
	Data  []TransferData `json:"data"`
}

// WithdrawData stores data of withdrawal
type WithdrawData struct {
	Address            string  `json:"address"`
	Amount             float64 `json:"amount"`
	ConfirmedTimestamp int64   `json:"confirmed_timestamp"`
	CreatedTimestamp   int64   `json:"created_timestamp"`
	Currency           string  `json:"currency"`
	Fee                float64 `json:"fee"`
	ID                 int64   `json:"id"`
	Priority           float64 `json:"priority"`
	State              string  `json:"state"`
	TransactionID      int64   `json:"transaction_id"`
	UpdatedTimestamp   int64   `json:"updated_timestamp"`
}

// WithdrawalsData stores data of withdrawals
type WithdrawalsData struct {
	Count int64          `json:"count"`
	Data  []WithdrawData `json:"data"`
}

// TradeData stores a data for a private trade
type TradeData struct {
	TradeSequence  int64   `json:"trade_seq"`
	TradeID        int64   `json:"trade_id"`
	Timestamp      int64   `json:"timestamp"`
	TickDirection  int64   `json:"tick_direction"`
	State          string  `json:"state"`
	SelfTrade      bool    `json:"self_trade"`
	ReduceOnly     bool    `json:"reduce_only"`
	Price          float64 `json:"price"`
	PostOnly       bool    `json:"post_only"`
	OrderType      string  `json:"order_type"`
	OrderID        string  `json:"order_id"`
	MatchingID     int64   `json:"matching_id"`
	MarkPrice      float64 `json:"mark_price"`
	Liquidity      string  `json:"liquidity"`
	Label          string  `json:"label"`
	InstrumentName string  `json:"instrument_name"`
	IndexPrice     float64 `json:"index_price"`
	FeeCurrency    string  `json:"fee_currency"`
	Fee            float64 `json:"fee"`
	Direction      string  `json:"direction"`
	Amount         float64 `json:"amount"`
}

// OrderData stores order data
type OrderData struct {
	Web                 bool    `json:"web"`
	TimeInForce         string  `json:"time_in_force"`
	Replaced            bool    `json:"replaced"`
	ReduceOnly          bool    `json:"reduce_only"`
	ProfitLoss          float64 `json:"profit_loss"`
	Price               float64 `json:"price"`
	PostOnly            bool    `json:"post_only"`
	OrderType           string  `json:"order_type"`
	OrderState          string  `json:"order_state"`
	OrderID             int64   `json:"order_id"`
	MaxShow             int64   `json:"max_show"`
	LastUpdateTimestamp int64   `json:"last_update_timestamp"`
	Label               string  `json:"label"`
	IsLiquidation       bool    `json:"is_liquidation"`
	InstrumentName      string  `json:"instrument_name"`
	FilledAmount        float64 `json:"filled_amount"`
	Direction           string  `json:"direction"`
	CreationTimestamp   int64   `json:"creation_timestamp"`
	Commission          float64 `json:"commission"`
	AveragePrice        float64 `json:"average_price"`
	API                 bool    `json:"api"`
	Amount              float64 `json:"amount"`
}

// PrivateTradeData stores data of a private buy, sell or edit
type PrivateTradeData struct {
	Trades []TradeData `json:"trades"`
	Order  OrderData   `json:"order"`
}

// PrivateCancelData stores data of a private cancel
type PrivateCancelData struct {
	Triggered           bool    `json:"triggered"`
	Trigger             string  `json:"trigger"`
	TimeInForce         string  `json:"time_in_force"`
	TriggerPrice        float64 `json:"trigger_price"`
	ReduceOnly          bool    `json:"reduce_only"`
	ProfitLoss          float64 `json:"profit_loss"`
	Price               string  `json:"price"`
	PostOnly            bool    `json:"post_only"`
	OrderType           string  `json:"order_type"`
	OrderState          string  `json:"order_state"`
	OrderID             string  `json:"order_id"`
	MaxShow             int64   `json:"max_show"`
	LastUpdateTimestamp int64   `json:"last_update_timestamp"`
	Label               string  `json:"label"`
	IsLiquidation       bool    `json:"is_liquidation"`
	InstrumentName      string  `json:"instrument_name"`
	Direction           string  `json:"direction"`
	CreationTimestamp   int64   `json:"creation_timestamp"`
	API                 bool    `json:"api"`
	Amount              float64 `json:"amount"`
}
