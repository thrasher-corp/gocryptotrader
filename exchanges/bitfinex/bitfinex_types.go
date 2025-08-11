package bitfinex

import (
	"errors"
	"math"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	errSetCannotBeEmpty        = errors.New("set cannot be empty")
	errNoSeqNo                 = errors.New("no sequence number")
	errParamNotAllowed         = errors.New("param not allowed")
	errTickerInvalidSymbol     = errors.New("invalid ticker symbol")
	errTickerInvalidResp       = errors.New("invalid ticker response format")
	errTickerInvalidFieldCount = errors.New("invalid ticker response field count")
)

// AccountV2Data stores account v2 data
type AccountV2Data struct {
	ID               int64
	Email            string
	Username         string
	MTSAccountCreate int64
	Verified         int64
	Timezone         string
}

// MarginInfoV2 stores V2 margin data
type MarginInfoV2 struct {
	Symbol          string
	UserPNL         float64
	UserSwaps       float64
	MarginBalance   float64
	MarginNet       float64
	MarginMin       float64
	TradableBalance float64
	GrossBalance    float64
	BestAskAmount   float64
	BestBidAmount   float64
}

// WalletDataV2 stores wallet data for v2
type WalletDataV2 struct {
	WalletType        string
	Currency          string
	Balance           float64
	UnsettledInterest float64
}

// AcceptedOrderType defines the accepted market types, exchange strings denote non-contract order types.
var AcceptedOrderType = []string{
	"market", "limit", "stop", "trailing-stop",
	"fill-or-kill", "exchange market", "exchange limit", "exchange stop",
	"exchange trailing-stop", "exchange fill-or-kill",
}

// AcceptedWalletNames defines different wallets supported by the exchange
var AcceptedWalletNames = []string{
	"trading", "exchange", "deposit", "margin",
	"funding",
}

type acceptableMethodStore struct {
	a map[string][]string
	m sync.RWMutex
}

// acceptableMethods holds the available acceptable deposit and withdraw methods
var acceptableMethods acceptableMethodStore

func (a *acceptableMethodStore) lookup(curr currency.Code) []string {
	a.m.RLock()
	defer a.m.RUnlock()
	var methods []string
	for k, v := range a.a {
		if common.StringSliceCompareInsensitive(v, curr.Upper().String()) {
			methods = append(methods, k)
		}
	}
	return methods
}

func (a *acceptableMethodStore) load(data map[string][]string) {
	a.m.Lock()
	defer a.m.Unlock()
	a.a = data
}

func (a *acceptableMethodStore) loaded() bool {
	a.m.RLock()
	defer a.m.RUnlock()
	return len(a.a) > 0
}

// MarginV2FundingData stores margin funding data
type MarginV2FundingData struct {
	Symbol        string
	RateAverage   float64
	AmountAverage float64
}

// MarginFundingDataV2 stores margin funding data
type MarginFundingDataV2 struct {
	Sym    string
	Symbol string
	Data   struct {
		YieldLoan    float64
		YieldLend    float64
		DurationLoan float64
		DurationLend float64
	}
}

// MarginFundingData stores data for margin funding
type MarginFundingData struct {
	ID          int64
	Symbol      string
	MTSCreated  int64
	MTSUpdated  int64
	Amount      float64
	AmountOrig  float64
	OrderType   string
	OfferStatus string
	Active      string
	Rate        float64
	Period      float64
	Notify      bool
	Renew       bool
}

// Ticker holds ticker information
type Ticker struct {
	FlashReturnRate    float64
	Bid                float64
	BidPeriod          int64
	BidSize            float64
	Ask                float64
	AskPeriod          int64
	AskSize            float64
	DailyChange        float64
	DailyChangePerc    float64
	Last               float64
	Volume             float64
	High               float64
	Low                float64
	FRRAmountAvailable float64 // Flash Return Rate amount available
}

// DerivativeDataResponse stores data for queried derivative
type DerivativeDataResponse struct {
	Key                  string
	MTS                  float64
	DerivPrice           float64
	SpotPrice            float64
	MarkPrice            float64
	InsuranceFundBalance float64
	NextFundingEventTS   float64
	NextFundingAccrued   float64
	NextFundingStep      float64
	CurrentFunding       float64
	OpenInterest         float64
}

// Stat holds individual statistics from exchange
type Stat struct {
	Period int64   `json:"period"`
	Volume float64 `json:"volume,string"`
}

// FundingBook holds current the full margin funding book
type FundingBook struct {
	Bids []FundingBookItem `json:"bids"`
	Asks []FundingBookItem `json:"asks"`
}

// Book holds the orderbook item
type Book struct {
	OrderID int64
	Price   float64
	Rate    float64
	Period  float64
	Count   int64
	Amount  float64
}

// Orderbook holds orderbook information from bid and ask sides
type Orderbook struct {
	Bids []Book
	Asks []Book
}

// Trade holds resp information
type Trade struct {
	TID       int64
	Timestamp types.Time
	Amount    float64
	Price     float64
	Rate      float64
	Period    int64 // Funding offer period in days
	Side      order.Side
}

// UnmarshalJSON unmarshals JSON data into a Trade struct
func (t *Trade) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &[5]any{&t.TID, &t.Timestamp, &t.Amount, &t.Rate, &t.Period}); err != nil {
		return err
	}
	if t.Period == 0 {
		t.Price, t.Rate = t.Rate, 0
	}
	t.Side = order.Buy
	if t.Amount < 0 {
		t.Amount = math.Abs(t.Amount)
		t.Side = order.Sell
	}
	return nil
}

// Lendbook holds most recent funding data for a relevant currency
type Lendbook struct {
	Bids []Book `json:"bids"`
	Asks []Book `json:"asks"`
}

// FundingBookItem is a generalised sub-type to hold book information
type FundingBookItem struct {
	Rate            float64    `json:"rate,string"`
	Amount          float64    `json:"amount,string"`
	Period          int        `json:"period"`
	Timestamp       types.Time `json:"timestamp"`
	FlashReturnRate string     `json:"frr"`
}

// Lends holds the lent information by currency
type Lends struct {
	Rate       float64    `json:"rate,string"`
	AmountLent float64    `json:"amount_lent,string"`
	AmountUsed float64    `json:"amount_used,string"`
	Timestamp  types.Time `json:"timestamp"`
}

// AccountInfoFull adds the error message to Account info
type AccountInfoFull struct {
	Info    []AccountInfo
	Message string `json:"message"`
}

// AccountInfo general account information with fees
type AccountInfo struct {
	MakerFees float64           `json:"maker_fees,string"`
	TakerFees float64           `json:"taker_fees,string"`
	Fees      []AccountInfoFees `json:"fees"`
	Message   string            `json:"message"`
}

// AccountInfoFees general account information with fees
type AccountInfoFees struct {
	Pairs     string  `json:"pairs"`
	MakerFees float64 `json:"maker_fees,string"`
	TakerFees float64 `json:"taker_fees,string"`
}

// AccountFees stores withdrawal account fee data from Bitfinex
type AccountFees struct {
	Withdraw map[string]types.Number `json:"withdraw"`
}

// AccountSummary holds account summary data
type AccountSummary struct {
	TradeVolumePer30D []Currency `json:"trade_vol_30d"`
	FundingProfit30D  []Currency `json:"funding_profit_30d"`
	MakerFee          float64    `json:"maker_fee"`
	TakerFee          float64    `json:"taker_fee"`
}

// Currency is a sub-type for AccountSummary data
type Currency struct {
	Currency string  `json:"curr"`
	Volume   float64 `json:"vol,string"`
	Amount   float64 `json:"amount,string"`
}

// Deposit holds the deposit address info
type Deposit struct {
	Method       string
	CurrencyCode string
	Address      string // Deposit address (instead of the address, this field will show Tag/Memo/Payment_ID for currencies that require it)
	PoolAddress  string // Pool address (for currencies that require a Tag/Memo/Payment_ID)
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
	MarginBalance     float64        `json:"margin_balance,string"`
	TradableBalance   float64        `json:"tradable_balance,string"`
	UnrealizedPL      int64          `json:"unrealized_pl"`
	UnrealizedSwap    int64          `json:"unrealized_swap"`
	NetValue          float64        `json:"net_value,string"`
	RequiredMargin    int64          `json:"required_margin"`
	Leverage          float64        `json:"leverage,string"`
	MarginRequirement float64        `json:"margin_requirement,string"`
	MarginLimits      []MarginLimits `json:"margin_limits"`
}

// MarginLimits holds limit data per pair
type MarginLimits struct {
	OnPair            string  `json:"on_pair"`
	InitialMargin     float64 `json:"initial_margin,string"`
	MarginRequirement float64 `json:"margin_requirement,string"`
	TradableBalance   float64 `json:"tradable_balance,string"`
}

// Balance holds current balance data
type Balance struct {
	Type      string  `json:"type"`
	Currency  string  `json:"currency"`
	Amount    float64 `json:"amount,string"`
	Available float64 `json:"available,string"`
}

// WalletTransfer holds status of wallet to wallet content transfer on exchange
type WalletTransfer struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Withdrawal holds withdrawal status information
type Withdrawal struct {
	Status       string  `json:"status"`
	Message      string  `json:"message"`
	WithdrawalID int64   `json:"withdrawal_id"`
	Fees         string  `json:"fees"`
	WalletType   string  `json:"wallettype"`
	Method       string  `json:"method"`
	Address      string  `json:"address"`
	Invoice      string  `json:"invoice"`
	PaymentID    string  `json:"payment_id"`
	Amount       float64 `json:"amount,string"`
}

// Order holds order information when an order is in the market
type Order struct {
	ID                    int64         `json:"id"`
	Symbol                currency.Pair `json:"symbol"`
	Exchange              string        `json:"exchange"`
	Price                 float64       `json:"price,string"`
	AverageExecutionPrice float64       `json:"avg_execution_price,string"`
	Side                  string        `json:"side"`
	Type                  string        `json:"type"`
	Timestamp             types.Time    `json:"timestamp"`
	IsLive                bool          `json:"is_live"`
	IsCancelled           bool          `json:"is_cancelled"`
	IsHidden              bool          `json:"is_hidden"`
	WasForced             bool          `json:"was_forced"`
	OriginalAmount        float64       `json:"original_amount,string"`
	RemainingAmount       float64       `json:"remaining_amount,string"`
	ExecutedAmount        float64       `json:"executed_amount,string"`
	OrderID               int64         `json:"order_id,omitempty"`
}

// OrderMultiResponse holds order information on the executed orders
type OrderMultiResponse struct {
	Orders []Order `json:"order_ids"`
	Status string  `json:"status"`
}

// PlaceOrder is used for order placement
type PlaceOrder struct {
	Symbol   string  `json:"symbol"`
	Amount   float64 `json:"amount,string"`
	Price    float64 `json:"price,string"`
	Exchange string  `json:"exchange"`
	Side     string  `json:"side"`
	Type     string  `json:"type"`
}

// GenericResponse holds the result for a generic response
type GenericResponse struct {
	Result string `json:"result"`
}

// Position holds position information
type Position struct {
	ID        int64      `json:"id"`
	Symbol    string     `json:"string"`
	Status    string     `json:"active"`
	Base      float64    `json:"base,string"`
	Amount    float64    `json:"amount,string"`
	Timestamp types.Time `json:"timestamp"`
	Swap      float64    `json:"swap,string"`
	PL        float64    `json:"pl,string"`
}

// BalanceHistory holds balance history information
type BalanceHistory struct {
	Currency    string     `json:"currency"`
	Amount      float64    `json:"amount,string"`
	Balance     float64    `json:"balance,string"`
	Description string     `json:"description"`
	Timestamp   types.Time `json:"timestamp"`
}

// MovementHistory holds deposit and withdrawal history data
type MovementHistory struct {
	ID                 int64
	Currency           string
	CurrencyName       string // AKA Method
	TXID               string
	MTSStarted         types.Time
	MTSUpdated         types.Time
	Status             string
	Amount             types.Number // Positive for deposits, negative for withdrawals
	Fees               types.Number
	DestinationAddress string
	PaymentID          *string
	TransactionID      *string
	TransactionNote    *string
	TransactionType    string // "deposit" or "withdrawal"
}

// UnmarshalJSON unmarshals JSON data into a MovementHistory struct
func (m *MovementHistory) UnmarshalJSON(data []byte) error {
	var unusedField any
	if err := json.Unmarshal(data, &[22]any{
		&m.ID,
		&m.Currency,
		&m.CurrencyName,
		&unusedField,
		&unusedField,
		&m.MTSStarted,
		&m.MTSUpdated,
		&unusedField,
		&unusedField,
		&m.Status,
		&unusedField,
		&unusedField,
		&m.Amount,
		&m.Fees,
		&unusedField,
		&unusedField,
		&m.DestinationAddress,
		&m.PaymentID,
		&unusedField,
		&unusedField,
		&m.TransactionID,
		&m.TransactionNote,
	}); err != nil {
		return err
	}

	if m.Amount < 0 {
		m.TransactionType = "withdrawal"
	} else {
		m.TransactionType = "deposit"
	}
	return nil
}

// TradeHistory holds trade history data
type TradeHistory struct {
	Price       float64    `json:"price,string"`
	Amount      float64    `json:"amount,string"`
	Timestamp   types.Time `json:"timestamp"`
	Exchange    string     `json:"exchange"`
	Type        string     `json:"type"`
	FeeCurrency string     `json:"fee_currency"`
	FeeAmount   float64    `json:"fee_amount,string"`
	TID         int64      `json:"tid"`
	OrderID     int64      `json:"order_id"`
}

// Offer holds offer information
type Offer struct {
	ID              int64      `json:"id"`
	Currency        string     `json:"currency"`
	Rate            float64    `json:"rate,string"`
	Period          int64      `json:"period"`
	Direction       string     `json:"direction"`
	Timestamp       types.Time `json:"timestamp"`
	Type            string     `json:"type"`
	IsLive          bool       `json:"is_live"`
	IsCancelled     bool       `json:"is_cancelled"`
	OriginalAmount  float64    `json:"original_amount,string"`
	RemainingAmount float64    `json:"remaining_amount,string"`
	ExecutedAmount  float64    `json:"executed_amount,string"`
}

// MarginFunds holds active funding information used in a margin position
type MarginFunds struct {
	ID         int64      `json:"id"`
	PositionID int64      `json:"position_id"`
	Currency   string     `json:"currency"`
	Rate       float64    `json:"rate,string"`
	Period     int        `json:"period"`
	Amount     float64    `json:"amount,string"`
	Timestamp  types.Time `json:"timestamp"`
	AutoClose  bool       `json:"auto_close"`
}

// MarginTotalTakenFunds holds position funding including sum of active backing
// as total swaps
type MarginTotalTakenFunds struct {
	PositionPair string  `json:"position_pair"`
	TotalSwaps   float64 `json:"total_swaps,string"`
}

// Fee holds fee data for a specified currency
type Fee struct {
	Currency  string
	TakerFees float64
	MakerFees float64
}

// WebsocketBook holds booking information
type WebsocketBook struct {
	ID     int64
	Price  float64
	Amount float64
	Period int64
}

// Candle holds OHLCV data
type Candle struct {
	Timestamp types.Time
	Open      types.Number
	Close     types.Number
	High      types.Number
	Low       types.Number
	Volume    types.Number
}

// UnmarshalJSON unmarshals JSON data into a Candle struct
func (c *Candle) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[6]any{&c.Timestamp, &c.Open, &c.Close, &c.High, &c.Low, &c.Volume})
}

// Leaderboard keys
const (
	LeaderboardUnrealisedProfitPeriodDelta = "plu_diff"
	LeaderboardUnrealisedProfitInception   = "plu"
	LeaderboardVolume                      = "vol"
	LeaderbookRealisedProfit               = "plr"
)

// LeaderboardEntry holds leaderboard data
type LeaderboardEntry struct {
	Timestamp     time.Time
	Username      string
	Ranking       int
	Value         float64
	TwitterHandle string
}

// WebsocketTicker holds ticker information
type WebsocketTicker struct {
	Bid             float64
	BidSize         float64
	Ask             float64
	AskSize         float64
	DailyChange     float64
	DialyChangePerc float64
	LastPrice       float64
	Volume          float64
}

// WebsocketPosition holds position information
type WebsocketPosition struct {
	Pair              string
	Status            string
	Amount            float64
	Price             float64
	MarginFunding     float64
	MarginFundingType int64
	ProfitLoss        float64
	ProfitLossPercent float64
	LiquidationPrice  float64
	Leverage          float64
}

// WebsocketWallet holds wallet information
type WebsocketWallet struct {
	Name              string
	Currency          string
	Balance           float64
	UnsettledInterest float64
}

// WebsocketOrder holds order data
type WebsocketOrder struct {
	OrderID    int64
	Pair       string
	Amount     float64
	OrigAmount float64
	OrderType  string
	Status     string
	Price      float64
	PriceAvg   float64
	Timestamp  types.Time
	Notify     int
}

// WebsocketTradeExecuted holds executed trade data
type WebsocketTradeExecuted struct {
	TradeID        int64
	Pair           string
	Timestamp      types.Time
	OrderID        int64
	AmountExecuted float64
	PriceExecuted  float64
}

// WebsocketTradeData holds executed trade data
type WebsocketTradeData struct {
	TradeID        int64
	Pair           string
	Timestamp      types.Time
	OrderID        int64
	AmountExecuted float64
	PriceExecuted  float64
	OrderType      string
	OrderPrice     float64
	Maker          bool
	Fee            float64
	FeeCurrency    string
}

// ErrorCapture is a simple type for returned errors from Bitfinex
type ErrorCapture struct {
	Message string `json:"message"`
}

// WebsocketHandshake defines the communication between the websocket API for
// initial connection
type WebsocketHandshake struct {
	Event   string  `json:"event"`
	Code    int64   `json:"code"`
	Version float64 `json:"version"`
}

// WsAuthRequest container for WS auth request
type WsAuthRequest struct {
	Event         string `json:"event"`
	APIKey        string `json:"apiKey"`
	AuthPayload   string `json:"authPayload"`
	AuthSig       string `json:"authSig"`
	AuthNonce     string `json:"authNonce"`
	DeadManSwitch int64  `json:"dms,omitempty"`
}

// WsFundingOffer funding offer received via websocket
type WsFundingOffer struct {
	ID             int64
	Symbol         string
	Created        time.Time
	Updated        time.Time
	Amount         float64
	OriginalAmount float64
	Type           string
	Flags          any
	Status         string
	Rate           float64
	Period         int64
	Notify         bool
	Hidden         bool
	Insure         bool
	Renew          bool
	RateReal       float64
}

// WsCredit credit details received via websocket
type WsCredit struct {
	ID           int64
	Symbol       string
	Side         int8 // 1 if you are the lender, 0 if you are both the lender and borrower, -1 if you're the borrower
	Created      time.Time
	Updated      time.Time
	Amount       float64
	Flags        any // Future params object (stay tuned)
	Status       string
	Rate         float64
	Period       int64
	Opened       time.Time
	LastPayout   time.Time
	Notify       bool
	Hidden       bool
	Renew        bool
	RateReal     float64
	NoClose      bool
	PositionPair string
}

// WsWallet wallet update details received via websocket
type WsWallet struct {
	Type              string
	Currency          string
	Balance           float64
	UnsettledInterest float64
	BalanceAvailable  float64
}

// WsBalanceInfo the total and net assets in your account received via websocket
type WsBalanceInfo struct {
	TotalAssetsUnderManagement float64
	NetAssetsUnderManagement   float64
}

// WsFundingInfo account funding info received via websocket
type WsFundingInfo struct {
	Symbol       string
	YieldLoan    float64
	YieldLend    float64
	DurationLoan float64
	DurationLend float64
}

// WsMarginInfoBase account margin info received via websocket
type WsMarginInfoBase struct {
	UserProfitLoss float64
	UserSwaps      float64
	MarginBalance  float64
	MarginNet      float64
	MarginRequired float64
}

// WsFundingTrade recent funding trades received via websocket
type WsFundingTrade struct {
	ID         int64
	Symbol     string
	MTSCreated time.Time
	OfferID    int64
	Amount     float64
	Rate       float64
	Period     int64
	Maker      bool
}

// WsNewOrderRequest new order request...
type WsNewOrderRequest struct {
	GroupID             int64   `json:"gid,omitempty"`
	CustomID            int64   `json:"cid,omitempty"`
	Type                string  `json:"type"`
	Symbol              string  `json:"symbol"`
	Amount              float64 `json:"amount,string"`
	Price               float64 `json:"price,string"`
	Leverage            int64   `json:"lev,omitempty"`
	TrailingPrice       float64 `json:"price_trailing,string,omitempty"`
	AuxiliaryLimitPrice float64 `json:"price_aux_limit,string,omitempty"`
	StopPrice           float64 `json:"price_oco_stop,string,omitempty"`
	Flags               int64   `json:"flags,omitempty"`
	TimeInForce         string  `json:"tif,omitempty"`
}

// WsUpdateOrderRequest update order request...
type WsUpdateOrderRequest struct {
	OrderID             int64   `json:"id,omitempty"`
	CustomID            int64   `json:"cid,omitempty"`
	CustomIDDate        string  `json:"cid_date,omitempty"`
	GroupID             int64   `json:"gid,omitempty"`
	Price               float64 `json:"price,string,omitempty"`
	Amount              float64 `json:"amount,string,omitempty"`
	Leverage            int64   `json:"lev,omitempty"`
	Delta               float64 `json:"delta,string,omitempty"`
	AuxiliaryLimitPrice float64 `json:"price_aux_limit,string,omitempty"`
	TrailingPrice       float64 `json:"price_trailing,string,omitempty"`
	Flags               int64   `json:"flags,omitempty"`
	TimeInForce         string  `json:"tif,omitempty"`
}

// WsCancelOrderRequest cancel order request...
type WsCancelOrderRequest struct {
	OrderID      int64  `json:"id,omitempty"`
	CustomID     int64  `json:"cid,omitempty"`
	CustomIDDate string `json:"cid_date,omitempty"`
}

// WsCancelGroupOrdersRequest cancel orders request...
type WsCancelGroupOrdersRequest struct {
	OrderID      []int64   `json:"id,omitempty"`
	CustomID     [][]int64 `json:"cid,omitempty"`
	GroupOrderID []int64   `json:"gid,omitempty"`
}

// WsNewOfferRequest new offer request
type WsNewOfferRequest struct {
	Type   string  `json:"type,omitempty"`
	Symbol string  `json:"symbol,omitempty"`
	Amount float64 `json:"amount,string,omitempty"`
	Rate   float64 `json:"rate,string,omitempty"`
	Period float64 `json:"period,omitempty"`
	Flags  int64   `json:"flags,omitempty"`
}

// WsCancelOfferRequest cancel offer request
type WsCancelOfferRequest struct {
	OrderID int64 `json:"id"`
}

// WsCancelAllOrdersRequest cancel all orders request
type WsCancelAllOrdersRequest struct {
	All int64 `json:"all"`
}

// CancelMultiOrderResponse holds v2 cancelled order data
type CancelMultiOrderResponse struct {
	OrderID           string
	ClientOrderID     string
	GroupOrderID      string
	Symbol            string
	CreatedTime       time.Time
	UpdatedTime       time.Time
	Amount            float64
	OriginalAmount    float64
	OrderType         string
	OriginalOrderType string
	OrderFlags        string
	OrderStatus       string
	Price             float64
	AveragePrice      float64
	TrailingPrice     float64
	AuxLimitPrice     float64
}
