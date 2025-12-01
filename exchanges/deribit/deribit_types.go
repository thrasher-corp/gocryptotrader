package deribit

import (
	"errors"
	"regexp"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	sideBUY  = "buy"
	sideSELL = "sell"

	// currencies

	currencyBTC  = "BTC"
	currencyETH  = "ETH"
	currencySOL  = "SOL"
	currencyUSDC = "USDC"
	currencyUSDT = "USDT"
	currencyEURR = "EURR"
)

var (
	alphaNumericRegExp = regexp.MustCompile("^[a-zA-Z0-9_]*$")

	errUnsupportedIndexName                = errors.New("unsupported index name")
	errInvalidInstrumentID                 = errors.New("invalid instrument ID")
	errInvalidInstrumentName               = errors.New("invalid instrument name")
	errInvalidComboID                      = errors.New("invalid combo ID")
	errNoArgumentPassed                    = errors.New("no argument passed")
	errInvalidAmount                       = errors.New("invalid amount, must be greater than 0")
	errMissingNonce                        = errors.New("missing nonce")
	errInvalidTradeRole                    = errors.New("invalid trade role, only 'maker' and 'taker' are allowed")
	errInvalidPrice                        = errors.New("invalid trade price")
	errInvalidCryptoAddress                = errors.New("invalid crypto address")
	errIntervalNotSupported                = errors.New("interval not supported")
	errInvalidID                           = errors.New("invalid id")
	errInvalidMarginModel                  = errors.New("missing margin model")
	errInvalidEmailAddress                 = errors.New("invalid email address")
	errWebsocketConnectionNotAuthenticated = errors.New("websocket connection is not authenticated")
	errResolutionNotSet                    = errors.New("resolution not set")
	errInvalidDestinationID                = errors.New("invalid destination id")
	errUnacceptableAPIKey                  = errors.New("unacceptable api key name")
	errInvalidUsername                     = errors.New("new username has to be specified")
	errSubAccountNameChangeFailed          = errors.New("subaccount name change failed")
	errLanguageIsRequired                  = errors.New("language is required")
	errInvalidAPIKeyID                     = errors.New("invalid api key id")
	errMaxScopeIsRequired                  = errors.New("max scope is required")
	errTradeModeIsRequired                 = errors.New("self trading mode is required")
	errUserIDRequired                      = errors.New("userID is required")
	errInvalidOrderSideOrDirection         = errors.New("invalid direction, only 'buy' or 'sell' are supported")
	errDifferentInstruments                = errors.New("given instruments should have the same currency")
	errZeroTimestamp                       = errors.New("zero timestamps are not allowed")
	errMissingBlockTradeID                 = errors.New("missing block trade id")
	errMissingSubAccountID                 = errors.New("missing subaccount id")
	errUnsupportedInstrumentFormat         = errors.New("unsupported instrument type format")
	errSessionNameRequired                 = errors.New("session_name is required")
	errRefreshTokenRequired                = errors.New("refresh token is required")
	errSubjectIDRequired                   = errors.New("subject id is required")
	errMissingSignature                    = errors.New("missing signature")
	errStartingHeartbeat                   = errors.New("error starting heartbeat")
	errSendingHeartbeat                    = errors.New("error sending heartbeat")

	websocketRequestTimeout = time.Second * 30

	baseCurrencies = []string{
		currencyBTC,
		currencyETH,
		currencySOL,
		currencyUSDC,
		currencyUSDT,
		currencyEURR,
	}
)

// UnmarshalError is the struct which is used for unmarshalling errors
type UnmarshalError struct {
	Message string `json:"message"`
	Data    struct {
		Reason string `json:"reason"`
	}
	Code int64 `json:"code"`
}

// BookSummaryData stores summary data
type BookSummaryData struct {
	InterestRate           float64    `json:"interest_rate"`
	AskPrice               float64    `json:"ask_price"`
	VolumeUSD              float64    `json:"volume_usd"`
	Volume                 float64    `json:"volume"`
	QuoteCurrency          string     `json:"quote_currency"`
	PriceChange            float64    `json:"price_change"`
	OpenInterest           float64    `json:"open_interest"`
	MidPrice               float64    `json:"mid_price"`
	MarkPrice              float64    `json:"mark_price"`
	Low                    float64    `json:"low"`
	Last                   float64    `json:"last"`
	InstrumentName         string     `json:"instrument_name"`
	High                   float64    `json:"high"`
	EstimatedDeliveryPrice float64    `json:"estimated_delivery_price"`
	CreationTimestamp      types.Time `json:"creation_timestamp"`
	BidPrice               float64    `json:"bid_price"`
	BaseCurrency           string     `json:"base_currency"`
	Funding8H              float64    `json:"funding_8h,omitempty"`
	CurrentFunding         float64    `json:"current_funding,omitempty"`
	UnderlyingIndex        string     `json:"underlying_index"`
	UnderlyingPrice        float64    `json:"underlying_price"`
	VolumeNotional         float64    `json:"volume_notional"`
}

// ContractSizeData stores contract size for given instrument
type ContractSizeData struct {
	ContractSize float64 `json:"contract_size"`
}

// CurrencyData stores data for currencies
type CurrencyData struct {
	CoinType             string        `json:"coin_type"`
	Currency             currency.Code `json:"currency"`
	CurrencyLong         string        `json:"currency_long"`
	FeePrecision         int64         `json:"fee_precision"`
	MinConfirmations     int64         `json:"min_confirmations"`
	MinWithdrawalFee     float64       `json:"min_withdrawal_fee"`
	WithdrawalFee        float64       `json:"withdrawal_fee"`
	WithdrawalPriorities []struct {
		Value float64 `json:"value"`
		Name  string  `json:"name"`
	} `json:"withdrawal_priorities"`
}

// IndexDeliveryPrice store index delivery prices list.
type IndexDeliveryPrice struct {
	Data         []DeliveryPriceData `json:"data"`
	TotalRecords int64               `json:"records_total"`
}

// DeliveryPriceData stores index delivery_price
type DeliveryPriceData struct {
	Date          string  `json:"date"`
	DeliveryPrice float64 `json:"delivery_price"`
}

// FundingChartData stores futures funding chart data
type FundingChartData struct {
	CurrentInterest float64 `json:"current_interest"`
	Interest8H      float64 `json:"interest_8h"`
	Data            []struct {
		IndexPrice float64    `json:"index_price"`
		Interest8H float64    `json:"interest_8h"`
		Timestamp  types.Time `json:"timestamp"`
	} `json:"data"`
}

// FundingRateHistory represents a funding rate history item
type FundingRateHistory struct {
	Timestamp      types.Time `json:"timestamp"`
	IndexPrice     float64    `json:"index_price"`      // Index price in base currency
	PrevIndexPrice float64    `json:"prev_index_price"` // Previous index price in base currency
	Interest8H     float64    `json:"interest_8h"`      // 8hour interest rate
	Interest1H     float64    `json:"interest_1h"`      // 1hour interest rate
}

// HistoricalVolatilityData stores volatility data for requested symbols
type HistoricalVolatilityData struct {
	Timestamp types.Time
	Value     types.Number
}

// UnmarshalJSON  parses volatility data from a JSON array into HistoricalVolatilityData fields.
func (h *HistoricalVolatilityData) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[2]any{&h.Timestamp, &h.Value})
}

// IndexPrice holds index price for the instruments
type IndexPrice struct {
	BTC float64 `json:"BTC"`
	ETH float64 `json:"ETH"`
	Edp float64 `json:"edp"`
}

// IndexPriceData gets index price data
type IndexPriceData struct {
	EstimatedDeliveryPrice float64 `json:"estimated_delivery_price"`
	IndexPrice             float64 `json:"index_price"`
}

// InstrumentData gets data for instruments
type InstrumentData struct {
	InstrumentName               string        `json:"instrument_name"`
	BaseCurrency                 currency.Code `json:"base_currency"`
	Kind                         string        `json:"kind"`
	OptionType                   string        `json:"option_type"`
	QuoteCurrency                currency.Code `json:"quote_currency"`
	BlockTradeCommission         float64       `json:"block_trade_commission"`
	ContractSize                 float64       `json:"contract_size"`
	CreationTimestamp            types.Time    `json:"creation_timestamp"`
	ExpirationTimestamp          types.Time    `json:"expiration_timestamp"`
	IsActive                     bool          `json:"is_active"`
	Leverage                     float64       `json:"leverage"`
	MaxLeverage                  float64       `json:"max_leverage"`
	MakerCommission              float64       `json:"maker_commission"`
	MinimumTradeAmount           float64       `json:"min_trade_amount"`
	TickSize                     float64       `json:"tick_size"`
	TakerCommission              float64       `json:"taker_commission"`
	Strike                       float64       `json:"strike"`
	SettlementPeriod             string        `json:"settlement_period"`
	SettlementCurrency           currency.Code `json:"settlement_currency"`
	RequestForQuote              bool          `json:"rfq"`
	PriceIndex                   string        `json:"price_index"`
	InstrumentID                 int64         `json:"instrument_id"`
	CounterCurrency              string        `json:"counter_currency"`
	MaximumLiquidationCommission float64       `json:"max_liquidation_commission"`
	FutureType                   string        `json:"future_type"`
	TickSizeSteps                []struct {
		AbovePrice float64 `json:"above_price"`
		TickSize   float64 `json:"tick_size"`
	} `json:"tick_size_steps"`
	BlockTradeTickSize       float64 `json:"block_trade_tick_size"`
	BlockTradeMinTradeAmount float64 `json:"block_trade_min_trade_amount"`
	InstrumentType           string  `json:"instrument_type"`
}

// SettlementsData stores data for settlement futures
type SettlementsData struct {
	Settlements []struct {
		Funded            float64    `json:"funded"`
		Funding           float64    `json:"funding"`
		IndexPrice        float64    `json:"index_price"`
		SessionTax        float64    `json:"session_tax"`
		SessionTaxRate    float64    `json:"session_tax_rate"`
		Socialized        float64    `json:"socialized"`
		SettlementType    string     `json:"type"`
		Timestamp         types.Time `json:"timestamp"`
		SessionProfitLoss float64    `json:"session_profit_loss"`
		ProfitLoss        float64    `json:"profit_loss"`
		Position          float64    `json:"position"`
		MarkPrice         float64    `json:"mark_price"`
		InstrumentName    string     `json:"instrument_name"`
	} `json:"settlements"`
	Continuation string `json:"continuation"`
}

// PublicTradesData stores data for public trades
type PublicTradesData struct {
	Trades []struct {
		TradeSeq          float64    `json:"trade_seq"`
		TradeID           string     `json:"trade_id"`
		Timestamp         types.Time `json:"timestamp"`
		TickDirection     int64      `json:"tick_direction"`
		Price             float64    `json:"price"`
		MarkPrice         float64    `json:"mark_price"`
		Liquidation       string     `json:"liquidation"`
		ImpliedVolatility float64    `json:"iv"`
		InstrumentName    string     `json:"instrument_name"`
		IndexPrice        float64    `json:"index_price"`
		Direction         string     `json:"direction"`
		BlockTradeID      string     `json:"block_trade_id"`
		Amount            float64    `json:"amount"`
	} `json:"trades"`
	HasMore bool `json:"has_more"`
}

// MarkPriceHistory stores data for mark price history
type MarkPriceHistory struct {
	Timestamp      types.Time
	MarkPriceValue float64
}

// UnmarshalJSON deserialises the JSON info, including the timestamp.
func (a *MarkPriceHistory) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[2]any{&a.Timestamp, &a.MarkPriceValue})
}

// Orderbook stores orderbook data
type Orderbook struct {
	EstimatedDeliveryPrice float64    `json:"estimated_delivery_price"`
	UnderlyingPrice        float64    `json:"underlying_price"`
	UnderlyingIndex        string     `json:"underlying_index"`
	Timestamp              types.Time `json:"timestamp"`
	Stats                  struct {
		Volume      float64 `json:"volume"`
		PriceChange float64 `json:"price_change"`
		Low         float64 `json:"low"`
		High        float64 `json:"high"`
	} `json:"stats"`
	State           string  `json:"state"`
	SettlementPrice float64 `json:"settlement_price"`
	OpenInterest    float64 `json:"open_interest"`
	MinPrice        float64 `json:"min_price"`
	MaxPrice        float64 `json:"max_price"`
	MarkIV          float64 `json:"mark_iv"`
	MarkPrice       float64 `json:"mark_price"`
	LastPrice       float64 `json:"last_price"`
	InterestRate    float64 `json:"interest_rate"`
	InstrumentName  string  `json:"instrument_name"`
	IndexPrice      float64 `json:"index_price"`
	GreeksData      struct {
		Delta float64 `json:"delta"`
		Gamma float64 `json:"gamma"`
		Rho   float64 `json:"rho"`
		Theta float64 `json:"theta"`
		Vega  float64 `json:"vega"`
	} `json:"greeks"`
	Funding8H      float64     `json:"funding_8h"`
	CurrentFunding float64     `json:"current_funding"`
	ChangeID       int64       `json:"change_id"`
	Bids           [][]float64 `json:"bids"`
	Asks           [][]float64 `json:"asks"`
	BidIV          float64     `json:"bid_iv"`
	BestBidPrice   float64     `json:"best_bid_price"`
	BestBidAmount  float64     `json:"best_bid_amount"`
	BestAskAmount  float64     `json:"best_ask_amount"`
	AskIV          float64     `json:"ask_iv"`
}

// TradeVolumesData stores data for trade volumes
type TradeVolumesData struct {
	PutsVolume       float64 `json:"puts_volume"`
	PutsVolume7D     float64 `json:"puts_volume_7d"`
	PutsVolume30D    float64 `json:"puts_volume_30d"`
	FuturesVolume7D  float64 `json:"futures_volume_7d"`
	FuturesVolume30D float64 `json:"futures_volume_30d"`
	FuturesVolume    float64 `json:"futures_volume"`
	CurrencyPair     string  `json:"currency_pair"`
	CallsVolume7D    float64 `json:"calls_volume_7d"`
	CallsVolume30D   float64 `json:"calls_volume_30d"`
	CallsVolume      float64 `json:"calls_volume"`
}

// TVChartData stores trading view chart data
type TVChartData struct {
	Volume []float64 `json:"volume"`
	Cost   []float64 `json:"cost"`
	Ticks  []int64   `json:"ticks"` // Values of the time axis given in milliseconds since UNIX epoch
	Status string    `json:"status"`
	Open   []float64 `json:"open"`
	Low    []float64 `json:"low"`
	High   []float64 `json:"high"`
	Close  []float64 `json:"close"`
}

// VolatilityIndexRawData stores raw index data for volatility
type VolatilityIndexRawData struct {
	Data [][5]float64 `json:"data"`
}

// VolatilityIndexData stores index data for volatility
type VolatilityIndexData struct {
	TimestampMS time.Time `json:"timestamp"`
	Open        float64   `json:"open"`
	High        float64   `json:"high"`
	Low         float64   `json:"low"`
	Close       float64   `json:"close"`
}

// TickerData stores data for ticker
type TickerData struct {
	AskIV          float64 `json:"ask_iv"`
	BestAskAmount  float64 `json:"best_ask_amount"`
	BestAskPrice   float64 `json:"best_ask_price"`
	BestBidAmount  float64 `json:"best_bid_amount"`
	BestBidPrice   float64 `json:"best_bid_price"`
	BidIV          float64 `json:"bid_iv"`
	CurrentFunding float64 `json:"current_funding"`
	DeliveryPrice  float64 `json:"delivery_price"`
	Funding8H      float64 `json:"funding_8h"`
	GreeksData     struct {
		Delta float64 `json:"delta"`
		Gamma float64 `json:"gamma"`
		Rho   float64 `json:"rho"`
		Theta float64 `json:"theta"`
		Vega  float64 `json:"vega"`
	} `json:"greeks"`
	IndexPrice      float64 `json:"index_price"`
	InstrumentName  string  `json:"instrument_name"`
	LastPrice       float64 `json:"last_price"`
	MarkIV          float64 `json:"mark_iv"`
	MarkPrice       float64 `json:"mark_price"`
	MaxPrice        float64 `json:"max_price"`
	MinPrice        float64 `json:"min_price"`
	OpenInterest    float64 `json:"open_interest"`
	SettlementPrice float64 `json:"settlement_price"`
	State           string  `json:"state"`
	Stats           struct {
		VolumeNotional float64 `json:"volume_notional"`
		VolumeUSD      float64 `json:"volume_usd"`
		Volume         float64 `json:"volume"`
		PriceChange    float64 `json:"price_change"`
		Low            float64 `json:"low"`
		High           float64 `json:"high"`
	} `json:"stats"`
	Timestamp              types.Time `json:"timestamp"`
	UnderlyingIndex        string     `json:"underlying_index"`
	UnderlyingPrice        float64    `json:"underlying_price"`
	EstimatedDeliveryPrice float64    `json:"estimated_delivery_price"`
	InterestValue          float64    `json:"interest_value"`
}

// CancelWithdrawalData stores cancel request data for a withdrawal
type CancelWithdrawalData struct {
	Address            string     `json:"address"`
	Amount             float64    `json:"amount"`
	ConfirmedTimestamp types.Time `json:"confirmed_timestamp"`
	CreatedTimestamp   types.Time `json:"created_timestamp"`
	Currency           string     `json:"currency"`
	Fee                float64    `json:"fee"`
	ID                 int64      `json:"id"`
	Priority           float64    `json:"priority"`
	Status             string     `json:"status"`
	TransactionID      int64      `json:"transaction_id"`
	UpdatedTimestamp   types.Time `json:"updated_timestamp"`
}

// DepositAddressData stores data of a deposit address
type DepositAddressData struct {
	Address           string     `json:"address"`
	CreationTimestamp types.Time `json:"creation_timestamp"`
	Currency          string     `json:"currency"`
	Type              string     `json:"type"`
}

// DepositsData stores data of deposits
type DepositsData struct {
	Count int64 `json:"count"`
	Data  []struct {
		Address           string     `json:"address"`
		Amount            float64    `json:"amount"`
		Currency          string     `json:"currency"`
		ReceivedTimestamp types.Time `json:"received_timestamp"`
		State             string     `json:"state"`
		TransactionID     string     `json:"transaction_id"`
		UpdatedTimestamp  types.Time `json:"updated_timestamp"`
	} `json:"data"`
}

// TransfersData stores list of transfer data
type TransfersData struct {
	Count int64          `json:"count"`
	Data  []TransferData `json:"data"`
}

// TransferData stores data for a transfer
type TransferData struct {
	Amount           float64    `json:"amount"`
	CreatedTimestamp types.Time `json:"created_timestamp"`
	Currency         string     `json:"currency"`
	Direction        string     `json:"direction"`
	ID               int64      `json:"id"`
	OtherSide        string     `json:"other_side"`
	State            string     `json:"state"`
	Type             string     `json:"type"`
	UpdatedTimestamp types.Time `json:"updated_timestamp"`
}

// WithdrawData stores data of withdrawal
type WithdrawData struct {
	Address            string     `json:"address"`
	Amount             float64    `json:"amount"`
	ConfirmedTimestamp types.Time `json:"confirmed_timestamp"`
	CreatedTimestamp   types.Time `json:"created_timestamp"`
	Currency           string     `json:"currency"`
	Fee                float64    `json:"fee"`
	ID                 int64      `json:"id"`
	Priority           float64    `json:"priority"`
	State              string     `json:"state"`
	TransactionID      string     `json:"transaction_id"`
	UpdatedTimestamp   types.Time `json:"updated_timestamp"`
}

// WithdrawalsData stores data of withdrawals
type WithdrawalsData struct {
	Count int64          `json:"count"`
	Data  []WithdrawData `json:"data"`
}

// TradeData stores a data for a private trade
type TradeData struct {
	TradeSequence  int64      `json:"trade_seq"`
	TradeID        string     `json:"trade_id"`
	Timestamp      types.Time `json:"timestamp"`
	TickDirection  int64      `json:"tick_direction"`
	State          string     `json:"state"`
	SelfTrade      bool       `json:"self_trade"`
	ReduceOnly     bool       `json:"reduce_only"`
	Price          float64    `json:"price"`
	PostOnly       bool       `json:"post_only"`
	OrderType      string     `json:"order_type"`
	OrderID        string     `json:"order_id"`
	MatchingID     int64      `json:"matching_id"`
	MarkPrice      float64    `json:"mark_price"`
	Liquidity      string     `json:"liquidity"`
	Label          string     `json:"label"`
	InstrumentName string     `json:"instrument_name"`
	IndexPrice     float64    `json:"index_price"`
	FeeCurrency    string     `json:"fee_currency"`
	Fee            float64    `json:"fee"`
	Direction      string     `json:"direction"`
	Amount         float64    `json:"amount"`
}

// OrderData stores order data
type OrderData struct {
	Web                 bool       `json:"web"`
	TimeInForce         string     `json:"time_in_force"`
	Replaced            bool       `json:"replaced"`
	ReduceOnly          bool       `json:"reduce_only"`
	ProfitLoss          float64    `json:"profit_loss"`
	Price               float64    `json:"price"`
	PostOnly            bool       `json:"post_only"`
	OrderType           string     `json:"order_type"`
	OrderState          string     `json:"order_state"`
	OrderID             string     `json:"order_id"`
	MaxShow             float64    `json:"max_show"`
	LastUpdateTimestamp types.Time `json:"last_update_timestamp"`
	Label               string     `json:"label"`
	IsLiquidation       bool       `json:"is_liquidation"`
	InstrumentName      string     `json:"instrument_name"`
	FilledAmount        float64    `json:"filled_amount"`
	Direction           string     `json:"direction"`
	CreationTimestamp   types.Time `json:"creation_timestamp"`
	Commission          float64    `json:"commission"`
	AveragePrice        float64    `json:"average_price"`
	API                 bool       `json:"api"`
	Amount              float64    `json:"amount"`
	IsRebalance         bool       `json:"is_rebalance"`
}

// InitialMarginInfo represents an initial margin of an order.
type InitialMarginInfo struct {
	InitialMarginCurrency string  `json:"initial_margin_currency"`
	InitialMargin         float64 `json:"initial_margin"`
	OrderID               string  `json:"order_id"`
}

// PrivateTradeData stores data of a private buy, sell or edit
type PrivateTradeData struct {
	Trades []TradeData `json:"trades"`
	Order  OrderData   `json:"order"`
}

// CancelResp represents the detail of canceled order.
type CancelResp struct {
	InstrumentName string              `json:"instrument_name"`
	Currency       string              `json:"currency"`
	Result         []PrivateCancelData `json:"result"`
	Type           string              `json:"type"`
}

// PrivateCancelData stores data of a private cancel
type PrivateCancelData struct {
	Triggered           bool         `json:"triggered"`
	Trigger             string       `json:"trigger"`
	TimeInForce         string       `json:"time_in_force"`
	TriggerPrice        float64      `json:"trigger_price"`
	ReduceOnly          bool         `json:"reduce_only"`
	ProfitLoss          float64      `json:"profit_loss"`
	Price               types.Number `json:"price"`
	PostOnly            bool         `json:"post_only"`
	OrderType           string       `json:"order_type"`
	OrderState          string       `json:"order_state"`
	OrderID             string       `json:"order_id"`
	MaxShow             float64      `json:"max_show"`
	LastUpdateTimestamp types.Time   `json:"last_update_timestamp"`
	Label               string       `json:"label"`
	IsLiquidation       bool         `json:"is_liquidation"`
	InstrumentName      string       `json:"instrument_name"`
	Direction           string       `json:"direction"`
	CreationTimestamp   types.Time   `json:"creation_timestamp"`
	API                 bool         `json:"api"`
	Amount              float64      `json:"amount"`
	Web                 bool         `json:"web"`
	StopPrice           float64      `json:"stop_price"`
	Replaced            bool         `json:"replaced"`
	IsRebalance         bool         `json:"is_rebalance"`
	RiskReducing        bool         `json:"risk_reducing"`
	Contracts           float64      `json:"contracts"`
	AveragePrice        float64      `json:"average_price"`
	FilledAmount        float64      `json:"filled_amount"`
	Mmp                 bool         `json:"mmp"`
	CancelReason        string       `json:"cancel_reason"`
}

// MultipleCancelResponse represents a response after cancelling multiple orders.
type MultipleCancelResponse struct {
	CancelCount   int64
	CancelDetails []CancelResp
}

// UnmarshalJSON deserializes order cancellation response into a MultipleCancelResponse instance.
func (a *MultipleCancelResponse) UnmarshalJSON(data []byte) error {
	var cancelCount int64
	var cancelDetails []CancelResp
	err := json.Unmarshal(data, &cancelDetails)
	if err != nil {
		err = json.Unmarshal(data, &cancelCount)
		if err != nil {
			return err
		}
		a.CancelCount = cancelCount
		return nil
	}
	a.CancelDetails = cancelDetails
	for a := range cancelDetails {
		cancelCount += int64(len(cancelDetails[a].Result))
	}
	a.CancelCount = cancelCount
	return nil
}

// MarginsData stores data for margin
type MarginsData struct {
	Buy      float64 `json:"buy"`
	MaxPrice float64 `json:"max_price"`
	MinPrice float64 `json:"min_price"`
	Sell     float64 `json:"sell"`
}

// MMPConfigData gets the current configuration data for MMP
type MMPConfigData struct {
	Currency      string     `json:"currency"`
	Interval      int64      `json:"interval"`
	FrozenTime    types.Time `json:"frozen_time"`
	QuantityLimit float64    `json:"quantity_limit"`
}

// UserTradesData stores data of user trades
type UserTradesData struct {
	Trades  []UserTradeData `json:"trades"`
	HasMore bool            `json:"has_more"`
}

// UserTradeData stores data of user trades
type UserTradeData struct {
	UnderlyingPrice float64    `json:"underlying_price"`
	TradeSequence   int64      `json:"trade_sequence"`
	TradeInstrument string     `json:"trade_id"`
	Timestamp       types.Time `json:"timestamp"`
	TickDirection   int64      `json:"tick_direction"`
	State           string     `json:"state"`
	SelfTrade       bool       `json:"self_trade"`
	ReduceOnly      bool       `json:"reduce_only"`
	Price           float64    `json:"price"`
	PostOnly        bool       `json:"post_only"`
	OrderType       string     `json:"order_type"`
	OrderID         string     `json:"order_id"`
	MatchingID      int64      `json:"matching_id"`
	MarkPrice       float64    `json:"mark_price"`
	Liquidity       string     `json:"liquidity"`
	IV              float64    `json:"iv"`
	InstrumentName  string     `json:"instrument_name"`
	IndexPrice      float64    `json:"index_price"`
	FeeCurrency     string     `json:"fee_currency"`
	Fee             float64    `json:"fee"`
	Direction       string     `json:"direction"`
	Amount          float64    `json:"amount"`
}

// PrivateSettlementsHistoryData stores data for private settlement history
type PrivateSettlementsHistoryData struct {
	Settlements  []PrivateSettlementData `json:"settlements"`
	Continuation string                  `json:"continuation"`
}

// PrivateSettlementData stores private settlement data
type PrivateSettlementData struct {
	Type              string     `json:"type"`
	Timestamp         types.Time `json:"timestamp"`
	SessionProfitLoss float64    `json:"session_profit_loss"`
	ProfitLoss        float64    `json:"profit_loss"`
	Position          float64    `json:"position"`
	MarkPrice         float64    `json:"mark_price"`
	InstrumentName    string     `json:"instrument_name"`
	IndexPrice        float64    `json:"index_price"`
}

// AccountSummaryData stores data of account summary for a given currency
type AccountSummaryData struct {
	Balance                  float64 `json:"balance"`
	OptionsSessionUPL        float64 `json:"options_session_upl"`
	DepositAddress           string  `json:"deposit_address"`
	OptionsGamma             float64 `json:"options_gamma"`
	OptionsTheta             float64 `json:"options_theta"`
	Username                 string  `json:"username"`
	Equity                   float64 `json:"equity"`
	Type                     string  `json:"type"`
	Currency                 string  `json:"currency"`
	DeltaTotal               float64 `json:"delta_total"`
	FuturesSessionRPL        float64 `json:"futures_session_rpl"`
	PortfolioManagingEnabled bool    `json:"portfolio_managing_enabled"`
	TotalPL                  float64 `json:"total_pl"`
	MarginBalance            float64 `json:"margin_balance"`
	TFAEnabled               bool    `json:"tfa_enabled"`
	OptionsSessionRPL        float64 `json:"options_session_rpl"`
	OptionsDelta             float64 `json:"options_delta"`
	FuturesPL                float64 `json:"futures_pl"`
	ReferrerID               string  `json:"referrer_id"`
	ID                       int64   `json:"id"`
	SessionUPL               float64 `json:"session_upl"`
	AvailableWithdrawalFunds float64 `json:"available_withdrawal_funds"`
	OptionsPL                float64 `json:"options_pl"`
	SystemName               string  `json:"system_name"`
	Limits                   struct {
		NonMatchingEngine struct {
			Rate  int64 `json:"rate"`
			Burst int64 `json:"burst"`
		} `json:"non_matching_engine"`
		MatchingEngine struct {
			Rate  int64 `json:"rate"`
			Burst int64 `json:"burst"`
		} `json:"matching_engine"`
	} `json:"limits"`
	InitialMargin                float64            `json:"initial_margin"`
	ProjectedInitialMargin       float64            `json:"projected_initial_margin"`
	MaintenanceMargin            float64            `json:"maintenance_margin"`
	SessionRPL                   float64            `json:"session_rpl"`
	InteruserTransfersEnabled    bool               `json:"interuser_transfers_enabled"`
	OptionsVega                  float64            `json:"options_vega"`
	ProjectedDeltaTotal          float64            `json:"projected_delta_total"`
	Email                        string             `json:"email"`
	FuturesSessionUPL            float64            `json:"futures_session_upl"`
	AvailableFunds               float64            `json:"available_funds"`
	OptionsValue                 float64            `json:"options_value"`
	DeltaTotalMap                map[string]float64 `json:"delta_total_map"`
	ProjectedMaintenanceMargin   float64            `json:"projected_maintenance_margin"`
	EstimatedLiquidationRatio    float64            `json:"estimated_liquidation_ratio"`
	PortfolioMarginingEnabled    bool               `json:"portfolio_margining_enabled"`
	EstimatedLiquidationRatioMap map[string]float64 `json:"estimated_liquidation_ratio_map"`
	FeeBalance                   float64            `json:"fee_balance"`
	SpotReserve                  float64            `json:"spot_reserve"`
}

// APIKeyData stores data regarding the api key
type APIKeyData struct {
	Timestamp    types.Time `json:"timestamp"`
	MaxScope     string     `json:"max_scope"`
	ID           int64      `json:"id"`
	Enabled      bool       `json:"enabled"`
	Default      bool       `json:"default"`
	ClientSecret string     `json:"client_secret"`
	ClientID     string     `json:"client_id"`
	Name         string     `json:"name"`
}

// SubAccountData stores subaccount data
type SubAccountData struct {
	Email                string                           `json:"email"`
	ID                   int64                            `json:"id"`
	IsPassword           bool                             `json:"is_password"`
	LoginEnabled         bool                             `json:"login_enabled"`
	Portfolio            map[string]SubAccountBalanceData `json:"portfolio"`
	ReceiveNotifications bool                             `json:"receive_notifications"`
	SystemName           string                           `json:"system_name"`
	TFAEnabled           bool                             `json:"tfa_enabled"`
	Type                 string                           `json:"type"`
	Username             string                           `json:"username"`
}

// SubAccountBalanceData represents the subaccount balance information for each currency.
type SubAccountBalanceData struct {
	AvailableFunds           float64 `json:"available_funds"`
	AvailableWithdrawalFunds float64 `json:"available_withdrawal_funds"`
	Balance                  float64 `json:"balance"`
	Currency                 string  `json:"currency"`
	Equity                   float64 `json:"equity"`
	InitialMargin            float64 `json:"initial_margin"`
	MaintenanceMargin        float64 `json:"maintenance_margin"`
	MarginBalance            float64 `json:"margin_balance"`
}

// AffiliateProgramInfo stores info of affiliate program
type AffiliateProgramInfo struct {
	Received           map[string]float64 `json:"received"`
	NumberOfAffiliates int64              `json:"number_of_affiliates"`
	Link               string             `json:"link"`
	IsEnabled          bool               `json:"is_enabled"`
}

// PositionData stores data for account's position
type PositionData struct {
	AveragePrice              float64 `json:"average_price"`
	Delta                     float64 `json:"delta"`
	Direction                 string  `json:"direction"`
	EstimatedLiquidationPrice float64 `json:"estimated_liquidation_price"`
	FloatingProfitLoss        float64 `json:"floating_profit_loss"`
	IndexPrice                float64 `json:"index_price"`
	InitialMargin             float64 `json:"initial_margin"`
	InstrumentName            string  `json:"instrument_name"`
	Leverage                  float64 `json:"leverage"`
	Kind                      string  `json:"kind"`
	MaintenanceMargin         float64 `json:"maintenance_margin"`
	MarkPrice                 float64 `json:"mark_price"`
	OpenOrdersMargin          float64 `json:"open_orders_margin"`
	RealizedProfitLoss        float64 `json:"realized_profit_loss"`
	SettlementPrice           float64 `json:"settlement_price"`
	Size                      float64 `json:"size"`
	SizeCurrency              float64 `json:"size_currency"`
	TotalProfitLoss           float64 `json:"total_profit_loss"`
	Theta                     float64 `json:"theta"`
	Vega                      float64 `json:"vega"`
	RealizedFunding           float64 `json:"realized_funding"`
	InterestValue             float64 `json:"interest_value"`
	Gamma                     float64 `json:"gamma"`
	FloatingProfitAndLossUSD  float64 `json:"floating_profit_loss_usd"`
}

// TransactionLogData stores information regarding an account transaction
type TransactionLogData struct {
	Username        string     `json:"username"`
	UserSeq         int64      `json:"user_seq"`
	UserID          int64      `json:"user_id"`
	TransactionType string     `json:"transaction_type"`
	TradeID         string     `json:"trade_id"`
	Timestamp       types.Time `json:"timestamp"`
	Side            string     `json:"side"`
	Price           float64    `json:"price"`
	Position        float64    `json:"position"`
	OrderID         string     `json:"order_id"`
	InterestPL      float64    `json:"interest_pl"`
	InstrumentName  string     `json:"instrument_name"`
	Info            struct {
		TransferType string `json:"transfer_type"`
		OtherUserID  int64  `json:"other_user_id"`
		OtherUser    string `json:"other_user"`
	} `json:"info"`
	ID         int64   `json:"id"`
	Equity     float64 `json:"equity"`
	Currency   string  `json:"currency"`
	Commission float64 `json:"commission"`
	Change     float64 `json:"change"`
	Cashflow   float64 `json:"cashflow"`
	Balance    float64 `json:"balance"`
}

// TransactionsData stores multiple transaction logs
type TransactionsData struct {
	Logs         []TransactionLogData `json:"logs"`
	Continuation int64                `json:"continuation"`
}

// wsInput defines a request obj for the JSON-RPC login and gets a websocket
// response
type wsInput struct {
	JSONRPCVersion string         `json:"jsonrpc,omitempty"`
	ID             string         `json:"id,omitempty"`
	Method         string         `json:"method"`
	Params         map[string]any `json:"params,omitempty"`
}

// WsRequest defines a request obj for the JSON-RPC endpoints and gets a websocket
// response
type WsRequest struct {
	JSONRPCVersion string `json:"jsonrpc,omitempty"`
	ID             string `json:"id,omitempty"`
	Method         string `json:"method"`
	Params         any    `json:"params,omitempty"`
}

// WsSubscriptionInput defines a request obj for the JSON-RPC and gets a websocket
// response
type WsSubscriptionInput struct {
	JSONRPCVersion string              `json:"jsonrpc,omitempty"`
	ID             string              `json:"id,omitempty"`
	Method         string              `json:"method"`
	Params         map[string][]string `json:"params,omitempty"`
}

type wsResponse struct {
	JSONRPCVersion string `json:"jsonrpc,omitempty"`
	ID             string `json:"id,omitempty"`
	Method         string `json:"method"`
	Params         struct {
		Data    any    `json:"data"`
		Channel string `json:"channel"`
		Type    string `json:"type"` // Used in heartbeat and test_request messages
	} `json:"params"`
	Result any `json:"result,omitempty"`
	Error  struct {
		Message string `json:"message,omitempty"`
		Code    int64  `json:"code,omitempty"`
		Data    any    `json:"data"`
	} `json:"error"`
}

type wsLoginResponse struct {
	JSONRPCVersion string          `json:"jsonrpc"`
	ID             string          `json:"id"`
	Method         string          `json:"method"`
	Result         map[string]any  `json:"result"`
	Error          *UnmarshalError `json:"error"`
}

type wsSubscriptionResponse struct {
	JSONRPCVersion string   `json:"jsonrpc"`
	ID             string   `json:"id"`
	Method         string   `json:"method"`
	Result         []string `json:"result"`
}

// ComboDetail retrieves information about a combo
type ComboDetail struct {
	ID                string     `json:"id"`
	InstrumentID      int64      `json:"instrument_id"`
	CreationTimestamp types.Time `json:"creation_timestamp"`
	StateTimestamp    types.Time `json:"state_timestamp"`
	State             string     `json:"state"`
	Legs              []struct {
		InstrumentName string  `json:"instrument_name"`
		Amount         float64 `json:"amount"`
	} `json:"legs"`
}

// ComboParam represents a parameter to sell and buy combo.
type ComboParam struct {
	InstrumentName string  `json:"instrument_name"`
	Direction      string  `json:"direction"`
	Amount         float64 `json:"amount,string"`
}

// BlockTradeParam represents a block trade parameter.
type BlockTradeParam struct {
	Price          float64 `json:"price"`
	InstrumentName string  `json:"instrument_name"`
	Direction      string  `json:"direction,omitempty"`
	Amount         float64 `json:"amount"`
}

// BlockTradeData represents a user's block trade data.
type BlockTradeData struct {
	TradeSeq               int64      `json:"trade_seq"`
	TradeID                string     `json:"trade_id"`
	Timestamp              types.Time `json:"timestamp"`
	TickDirection          int64      `json:"tick_direction"`
	State                  string     `json:"state"`
	SelfTrade              bool       `json:"self_trade"`
	Price                  float64    `json:"price"`
	OrderType              string     `json:"order_type"`
	OrderID                string     `json:"order_id"`
	MatchingID             any        `json:"matching_id"`
	Liquidity              string     `json:"liquidity"`
	OptionmpliedVolatility float64    `json:"iv,omitempty"`
	InstrumentName         string     `json:"instrument_name"`
	IndexPrice             float64    `json:"index_price"`
	FeeCurrency            string     `json:"fee_currency"`
	Fee                    float64    `json:"fee"`
	Direction              string     `json:"direction"`
	BlockTradeID           string     `json:"block_trade_id"`
	Amount                 float64    `json:"amount"`
}

// Announcement represents public announcements.
type Announcement struct {
	Title                string     `json:"title"`
	PublicationTimestamp types.Time `json:"publication_timestamp"`
	Important            bool       `json:"important"`
	ID                   int64      `json:"id"`
	Body                 string     `json:"body"`

	// Action taken by the platform administrators.
	Action string `json:"action"`
}

// AccessLog represents access log information.
type AccessLog struct {
	RecordsTotal int64             `json:"records_total"`
	Data         []AccessLogDetail `json:"data"`
}

// AccessLogDetail represents detailed access log information.
type AccessLogDetail struct {
	Timestamp types.Time `json:"timestamp"`
	Result    string     `json:"result"`
	IP        string     `json:"ip"`
	ID        int64      `json:"id"`
	Country   string     `json:"country"`
	City      string     `json:"city"`
}

// SubAccountDetail represents subaccount positions detail.
type SubAccountDetail struct {
	UID       int64 `json:"uid"`
	Positions []struct {
		TotalProfitLoss           float64 `json:"total_profit_loss"`
		SizeCurrency              float64 `json:"size_currency"`
		Size                      float64 `json:"size"`
		SettlementPrice           float64 `json:"settlement_price"`
		RealizedProfitLoss        float64 `json:"realized_profit_loss"`
		RealizedFunding           float64 `json:"realized_funding"`
		OpenOrdersMargin          float64 `json:"open_orders_margin"`
		MarkPrice                 float64 `json:"mark_price"`
		MaintenanceMargin         float64 `json:"maintenance_margin"`
		Leverage                  float64 `json:"leverage"`
		Kind                      string  `json:"kind"`
		InstrumentName            string  `json:"instrument_name"`
		InitialMargin             float64 `json:"initial_margin"`
		IndexPrice                float64 `json:"index_price"`
		FloatingProfitLoss        float64 `json:"floating_profit_loss"`
		EstimatedLiquidationPrice float64 `json:"estimated_liquidation_price"`
		Direction                 string  `json:"direction"`
		Delta                     float64 `json:"delta"`
		AveragePrice              float64 `json:"average_price"`
	} `json:"positions"`
}

// UserLock represents a user lock information for currency.
type UserLock struct {
	Message  string `json:"message"`
	Locked   bool   `json:"locked"`
	Currency string `json:"currency"`
}

// PortfolioMarginState represents a portfolio margin state information.
type PortfolioMarginState struct {
	MaintenanceMarginRate float64 `json:"maintenance_margin_rate"`
	InitialMarginRate     float64 `json:"initial_margin_rate"`
	AvailableBalance      float64 `json:"available_balance"`
}

// TogglePortfolioMarginResponse represents a response from toggling portfolio margin for currency.
type TogglePortfolioMarginResponse struct {
	OldState PortfolioMarginState `json:"old_state"`
	NewState PortfolioMarginState `json:"new_state"`
	Currency string               `json:"currency"`
}

// BlockTradeResponse represents a block trade response.
type BlockTradeResponse struct {
	TradeSeq       int64      `json:"trade_seq"`
	TradeID        string     `json:"trade_id"`
	Timestamp      types.Time `json:"timestamp"`
	TickDirection  int64      `json:"tick_direction"`
	State          string     `json:"state"`
	SelfTrade      bool       `json:"self_trade"`
	ReduceOnly     bool       `json:"reduce_only"`
	Price          float64    `json:"price"`
	PostOnly       bool       `json:"post_only"`
	OrderType      string     `json:"order_type"`
	OrderID        string     `json:"order_id"`
	MatchingID     string     `json:"matching_id"`
	MarkPrice      float64    `json:"mark_price"`
	Liquidity      string     `json:"liquidity"`
	InstrumentName string     `json:"instrument_name"`
	IndexPrice     float64    `json:"index_price"`
	FeeCurrency    string     `json:"fee_currency"`
	Fee            float64    `json:"fee"`
	Direction      string     `json:"direction"`
	BlockTradeID   string     `json:"block_trade_id"`
	Amount         float64    `json:"amount"`
}

// BlockTradeMoveResponse represents block trade move response.
type BlockTradeMoveResponse struct {
	TargetSubAccountUID int64   `json:"target_uid"`
	SourceSubAccountUID int64   `json:"source_uid"`
	Price               float64 `json:"price"`
	InstrumentName      string  `json:"instrument_name"`
	Direction           string  `json:"direction"`
	Amount              float64 `json:"amount"`
}

// VersionInformation represents websocket version information
type VersionInformation struct {
	Version string `json:"version"`
}

// wsOrderbook represents orderbook push data for a book websocket subscription.
type wsOrderbook struct {
	Type           string     `json:"type"`
	Timestamp      types.Time `json:"timestamp"`
	InstrumentName string     `json:"instrument_name"`
	ChangeID       int64      `json:"change_id"`
	Bids           [][]any    `json:"bids"`
	Asks           [][]any    `json:"asks"`
}

// wsCandlestickData represents publicly available market data used to generate a TradingView candle chart.
type wsCandlestickData struct {
	Volume float64    `json:"volume"`
	Tick   types.Time `json:"tick"`
	Open   float64    `json:"open"`
	Low    float64    `json:"low"`
	High   float64    `json:"high"`
	Cost   float64    `json:"cost"`
	Close  float64    `json:"close"`
}

// wsIndexPrice represents information about current value (price) for Deribit Index
type wsIndexPrice struct {
	Timestamp types.Time `json:"timestamp"`
	Price     float64    `json:"price"`
	IndexName string     `json:"index_name"`
}

// wsRankingPrice
type wsRankingPrice struct {
	Weight        float64    `json:"weight"`
	Timestamp     types.Time `json:"timestamp"`
	Price         float64    `json:"price"`
	OriginalPrice float64    `json:"original_price"`
	Identifier    string     `json:"identifier"`
	Enabled       bool       `json:"enabled"`
}

// wsRankingPrices
type wsRankingPrices []wsRankingPrice

// wsPriceStatistics represents basic statistics about Deribit Index
type wsPriceStatistics struct {
	Low24H         float64 `json:"low24h"`
	IndexName      string  `json:"index_name"`
	HighVolatility bool    `json:"high_volatility"`
	High24H        float64 `json:"high24h"`
	Change24H      float64 `json:"change24h"`
}

// wsVolatilityIndex represents volatility index push data
type wsVolatilityIndex struct {
	Volatility        float64    `json:"volatility"`
	Timestamp         types.Time `json:"timestamp"`
	IndexName         string     `json:"index_name"`
	EstimatedDelivery float64    `json:"estimated_delivery"`
}

// wsEstimatedExpirationPrice represents push data of ending price for given index.
type wsEstimatedExpirationPrice struct {
	Seconds     int64   `json:"seconds"`
	Price       float64 `json:"price"`
	IsEstimated bool    `json:"is_estimated"`
}

// wsTicker represents changes in ticker (key information about the instrument).
type wsTicker struct {
	Timestamp types.Time `json:"timestamp"`
	Stats     struct {
		VolumeUsd   float64 `json:"volume_usd"`
		Volume      float64 `json:"volume"`
		PriceChange float64 `json:"price_change"`
		Low         float64 `json:"low"`
		High        float64 `json:"high"`
	} `json:"stats"`
	State                  string  `json:"state"`
	SettlementPrice        float64 `json:"settlement_price"`
	OpenInterest           float64 `json:"open_interest"`
	MinPrice               float64 `json:"min_price"`
	MaxPrice               float64 `json:"max_price"`
	MarkPrice              float64 `json:"mark_price"`
	LastPrice              float64 `json:"last_price"`
	InstrumentName         string  `json:"instrument_name"`
	IndexPrice             float64 `json:"index_price"`
	ImpliedBid             float64 `json:"implied_bid"`
	ImpliedAsk             float64 `json:"implied_ask"`
	EstimatedDeliveryPrice float64 `json:"estimated_delivery_price"`
	ComboState             string  `json:"combo_state"`
	BestBidPrice           float64 `json:"best_bid_price"`
	BestBidAmount          float64 `json:"best_bid_amount"`
	BestAskPrice           float64 `json:"best_ask_price"`
	BestAskAmount          float64 `json:"best_ask_amount"`
}

// WsIncrementalTicker represents a ticker information for incremental ticker subscriptions.
type WsIncrementalTicker struct {
	Type      string     `json:"type"`
	Timestamp types.Time `json:"timestamp"`
	Stats     struct {
		VolumeUsd   float64 `json:"volume_usd"`
		Volume      float64 `json:"volume"`
		PriceChange float64 `json:"price_change"`
	} `json:"stats"`
	MinPrice               float64 `json:"min_price"`
	MaxPrice               float64 `json:"max_price"`
	MarkPrice              float64 `json:"mark_price"`
	InstrumentName         string  `json:"instrument_name"`
	IndexPrice             float64 `json:"index_price"`
	EstimatedDeliveryPrice float64 `json:"estimated_delivery_price"`
	BestBidAmount          float64 `json:"best_bid_amount"`
	BestAskAmount          float64 `json:"best_ask_amount"`

	// For future_combo instruments
	ImpliedAsk float64 `json:"implied_ask"`
	ImpliedBid float64 `json:"implied_bid"`

	UnderlyingPrice float64 `json:"underlying_price"`
	UnderlyingIndex string  `json:"underlying_index"`
	State           string  `json:"state"`
	SettlementPrice float64 `json:"settlement_price"`
	OpenInterest    float64 `json:"open_interest"`

	MarkIv       float64 `json:"mark_iv"`
	LastPrice    float64 `json:"last_price"`
	InterestRate float64 `json:"interest_rate"`
	Greeks       struct {
		Vega  float64 `json:"vega"`
		Theta float64 `json:"theta"`
		Rho   float64 `json:"rho"`
		Gamma float64 `json:"gamma"`
		Delta float64 `json:"delta"`
	} `json:"greeks"`
	ComboState   string  `json:"combo_state"`
	BidIv        float64 `json:"bid_iv"`
	BestBidPrice float64 `json:"best_bid_price"`
	BestAskPrice float64 `json:"best_ask_price"`
	AskIv        float64 `json:"ask_iv"`
}

// wsInstrumentState represents notifications about new or terminated instruments of given kind in given currency.
type wsInstrumentState struct {
	Timestamp      types.Time `json:"timestamp"`
	State          string     `json:"state"`
	InstrumentName string     `json:"instrument_name"`
}

// wsMarkPriceOptions represents information about options markprices.
type wsMarkPriceOptions struct {
	Timestamp      types.Time `json:"timestamp"`
	MarkPrice      float64    `json:"mark_price"`
	Iv             float64    `json:"iv"`
	InstrumentName string     `json:"instrument_name"`
}

// wsPerpetualInterest represents current interest rate - but only for perpetual instruments.
type wsPerpetualInterest struct {
	Timestamp  types.Time `json:"timestamp"`
	Interest   float64    `json:"interest"`
	IndexPrice float64    `json:"index_price"`
}

// wsPlatformState holds Information whether unauthorized public requests are allowed
type wsPlatformState struct {
	AllowUnauthenticatedPublicRequests bool `json:"allow_unauthenticated_public_requests"`
}

// wsQuoteTickerInformation represents best bid/ask price and size.
type wsQuoteTickerInformation struct {
	Timestamp      types.Time `json:"timestamp"`
	InstrumentName string     `json:"instrument_name"`
	BestBidPrice   float64    `json:"best_bid_price"`
	BestBidAmount  float64    `json:"best_bid_amount"`
	BestAskPrice   float64    `json:"best_ask_price"`
	BestAskAmount  float64    `json:"best_ask_amount"`
}

// wsTrade represents trades for an instrument.
type wsTrade struct {
	TradeSequence  int64      `json:"trade_seq"`
	TradeID        string     `json:"trade_id"`
	Timestamp      types.Time `json:"timestamp"`
	TickDirection  float64    `json:"tick_direction"`
	Price          float64    `json:"price"`
	MarkPrice      float64    `json:"mark_price"`
	InstrumentName string     `json:"instrument_name"`
	IndexPrice     float64    `json:"index_price"`
	Direction      order.Side `json:"direction"`
	Amount         float64    `json:"amount"`
}

// wsAccessLog represents security events related to the account
type wsAccessLog struct {
	Timestamp types.Time `json:"timestamp"`
	Log       string     `json:"log"`
	IP        string     `json:"ip"`
	ID        int64      `json:"id"`
	Country   string     `json:"country"`
	City      string     `json:"city"`
}

// wsChanges represents user's updates related to order, trades, etc. in an instrument.
type wsChanges struct {
	Trades []struct {
		TradeSeq       float64    `json:"trade_seq"`
		TradeID        string     `json:"trade_id"`
		Timestamp      types.Time `json:"timestamp"`
		TickDirection  float64    `json:"tick_direction"`
		State          string     `json:"state"`
		SelfTrade      bool       `json:"self_trade"`
		ReduceOnly     bool       `json:"reduce_only"`
		ProfitLoss     float64    `json:"profit_loss"`
		Price          float64    `json:"price"`
		PostOnly       bool       `json:"post_only"`
		OrderType      string     `json:"order_type"`
		OrderID        string     `json:"order_id"`
		MatchingID     any        `json:"matching_id"`
		MarkPrice      float64    `json:"mark_price"`
		Liquidity      string     `json:"liquidity"`
		InstrumentName string     `json:"instrument_name"`
		IndexPrice     float64    `json:"index_price"`
		FeeCurrency    string     `json:"fee_currency"`
		Fee            float64    `json:"fee"`
		Direction      string     `json:"direction"`
		Amount         float64    `json:"amount"`
	} `json:"trades"`
	Positions []WebsocketPosition `json:"positions"`
	Orders    []struct {
		Web                 bool       `json:"web"`
		TimeInForce         string     `json:"time_in_force"`
		Replaced            bool       `json:"replaced"`
		ReduceOnly          bool       `json:"reduce_only"`
		ProfitLoss          float64    `json:"profit_loss"`
		Price               float64    `json:"price"`
		PostOnly            bool       `json:"post_only"`
		OrderType           string     `json:"order_type"`
		OrderState          string     `json:"order_state"`
		OrderID             string     `json:"order_id"`
		MaxShow             float64    `json:"max_show"`
		LastUpdateTimestamp types.Time `json:"last_update_timestamp"`
		Label               string     `json:"label"`
		IsLiquidation       bool       `json:"is_liquidation"`
		InstrumentName      string     `json:"instrument_name"`
		FilledAmount        float64    `json:"filled_amount"`
		Direction           string     `json:"direction"`
		CreationTimestamp   types.Time `json:"creation_timestamp"`
		Commission          float64    `json:"commission"`
		AveragePrice        float64    `json:"average_price"`
		API                 bool       `json:"api"`
		Amount              float64    `json:"amount"`
	} `json:"orders"`
	InstrumentName string `json:"instrument_name"`
}

// WebsocketPosition holds position information
type WebsocketPosition struct {
	TotalProfitLoss    float64 `json:"total_profit_loss"`
	SizeCurrency       float64 `json:"size_currency"`
	Size               float64 `json:"size"`
	SettlementPrice    float64 `json:"settlement_price"`
	RealizedProfitLoss float64 `json:"realized_profit_loss"`
	RealizedFunding    float64 `json:"realized_funding"`
	OpenOrdersMargin   float64 `json:"open_orders_margin"`
	MarkPrice          float64 `json:"mark_price"`
	MaintenanceMargin  float64 `json:"maintenance_margin"`
	Leverage           float64 `json:"leverage"`
	Kind               string  `json:"kind"`
	InterestValue      float64 `json:"interest_value"`
	InstrumentName     string  `json:"instrument_name"`
	InitialMargin      float64 `json:"initial_margin"`
	IndexPrice         float64 `json:"index_price"`
	FloatingProfitLoss float64 `json:"floating_profit_loss"`
	Direction          string  `json:"direction"`
	Delta              float64 `json:"delta"`
	AveragePrice       float64 `json:"average_price"`
}

// WsUserLock represents a notification data when account is locked/unlocked
type WsUserLock struct {
	Locked   bool   `json:"locked"`
	Currency string `json:"currency"`
}

// WsMMPTrigger represents mmp trigger data.
type WsMMPTrigger struct {
	FrozenUntil int64  `json:"frozen_until"`
	Currency    string `json:"currency"`
}

// WsOrder represents changes in user's orders for given instrument.
type WsOrder struct {
	TimeInForce         string     `json:"time_in_force"`
	Replaced            bool       `json:"replaced"`
	ReduceOnly          bool       `json:"reduce_only"`
	ProfitLoss          float64    `json:"profit_loss"`
	Price               float64    `json:"price"`
	PostOnly            bool       `json:"post_only"`
	OriginalOrderType   string     `json:"original_order_type"`
	OrderType           string     `json:"order_type"`
	OrderState          string     `json:"order_state"`
	OrderID             string     `json:"order_id"`
	MaxShow             float64    `json:"max_show"`
	LastUpdateTimestamp types.Time `json:"last_update_timestamp"`
	Label               string     `json:"label"`
	IsLiquidation       bool       `json:"is_liquidation"`
	InstrumentName      string     `json:"instrument_name"`
	FilledAmount        float64    `json:"filled_amount"`
	Direction           string     `json:"direction"`
	CreationTimestamp   types.Time `json:"creation_timestamp"`
	Commission          float64    `json:"commission"`
	AveragePrice        float64    `json:"average_price"`
	API                 bool       `json:"api"`
	Amount              float64    `json:"amount"`
}

// wsUserPortfolio represents current user portfolio
type wsUserPortfolio struct {
	TotalPl                    float64 `json:"total_pl"`
	SessionUpl                 float64 `json:"session_upl"`
	SessionRpl                 float64 `json:"session_rpl"`
	ProjectedMaintenanceMargin float64 `json:"projected_maintenance_margin"`
	ProjectedInitialMargin     float64 `json:"projected_initial_margin"`
	ProjectedDeltaTotal        float64 `json:"projected_delta_total"`
	PortfolioMarginingEnabled  bool    `json:"portfolio_margining_enabled"`
	OptionsVega                float64 `json:"options_vega"`
	OptionsValue               float64 `json:"options_value"`
	OptionsTheta               float64 `json:"options_theta"`
	OptionsSessionUpl          float64 `json:"options_session_upl"`
	OptionsSessionRpl          float64 `json:"options_session_rpl"`
	OptionsPl                  float64 `json:"options_pl"`
	OptionsGamma               float64 `json:"options_gamma"`
	OptionsDelta               float64 `json:"options_delta"`
	MarginBalance              float64 `json:"margin_balance"`
	MaintenanceMargin          float64 `json:"maintenance_margin"`
	InitialMargin              float64 `json:"initial_margin"`
	FuturesSessionUpl          float64 `json:"futures_session_upl"`
	FuturesSessionRpl          float64 `json:"futures_session_rpl"`
	FuturesPl                  float64 `json:"futures_pl"`
	EstimatedLiquidationRatio  float64 `json:"estimated_liquidation_ratio"`
	Equity                     float64 `json:"equity"`
	DeltaTotal                 float64 `json:"delta_total"`
	Currency                   string  `json:"currency"`
	Balance                    float64 `json:"balance"`
	AvailableWithdrawalFunds   float64 `json:"available_withdrawal_funds"`
	AvailableFunds             float64 `json:"available_funds"`
}

// OrderBuyAndSellParams represents request parameters for submit order.
type OrderBuyAndSellParams struct {
	OrderID        string  `json:"order_id,omitempty"`
	Instrument     string  `json:"instrument_name,omitempty"`
	Amount         float64 `json:"amount,omitempty"`
	OrderType      string  `json:"order_type,omitempty"`
	Price          float64 `json:"price,omitempty"`
	Label          string  `json:"label,omitempty"`
	TimeInForce    string  `json:"time_in_force,omitempty"`
	MaxShow        float64 `json:"max_show,omitempty"`
	PostOnly       bool    `json:"post_only,omitempty"`
	RejectPostOnly bool    `json:"reject_post_only,omitempty"`
	ReduceOnly     bool    `json:"reduce_only,omitempty"`
	MMP            bool    `json:"mmp,omitempty"`
	TriggerPrice   float64 `json:"trigger_price,omitempty"`
	Trigger        string  `json:"trigger,omitempty"`
	Advanced       string  `json:"advanced,omitempty"`
}

// ErrInfo represents an error response messages
type ErrInfo struct {
	Message string `json:"message"`
	Data    struct {
		Param  string `json:"param"`
		Reason string `json:"reason"`
	} `json:"data"`
	Code int64 `json:"code"`
}

// CustodyAccount retrieves user custody accounts list.
type CustodyAccount struct {
	Name                          string  `json:"name"`
	Currency                      string  `json:"currency"`
	ClientID                      string  `json:"client_id"`
	Balance                       float64 `json:"balance"`
	WithdrawalsRequireSecurityKey bool    `json:"withdrawals_require_security_key"`
	PendingWithdrawalBalance      float64 `json:"pending_withdrawal_balance"`
	AutoDeposit                   bool    `json:"auto_deposit"`
}

// LockedCurrenciesStatus represents locked currencies status information.
type LockedCurrenciesStatus struct {
	LockedCurrencies []string `json:"locked_currencies"`
	Locked           string   `json:"locked"`
}

// Info holds version information
type Info struct {
	Version string `json:"version"`
}

// CancelOnDisconnect holds scope and status information for cancel-on-disconnect
type CancelOnDisconnect struct {
	Scope  string `json:"scope"`
	Enable bool   `json:"enabled"`
}

// RefreshTokenInfo holds access token information.
type RefreshTokenInfo struct {
	AccessToken      string `json:"access_token"`
	ExpiresInSeconds int64  `json:"expires_in"`
	RefreshToken     string `json:"refresh_token"`
	Scope            string `json:"scope"`
	TokenType        string `json:"token_type"`
}
