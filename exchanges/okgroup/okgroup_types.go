package okgroup

import "encoding/json"
import "github.com/thrasher-/gocryptotrader/currency/symbol"

// CurrencyResponse contains currency details from a GetCurrencies request
type CurrencyResponse struct {
	CanDeposit    int64   `json:"can_deposit"`
	CanWithdraw   int64   `json:"can_withdraw"`
	Currency      string  `json:"currency"`
	MinWithdrawal float64 `json:"min_withdrawal"`
	Name          string  `json:"name"`
}

// WalletInformationResponse contains wallet details from a GetWalletInformation request
type WalletInformationResponse struct {
	Available float64 `json:"available"`
	Balance   float64 `json:"balance"`
	Currency  string  `json:"currency"`
	Hold      float64 `json:"hold"`
}

// FundTransferRequest used to request a fund transfer
type FundTransferRequest struct {
	Currency     string  `json:"currency"`
	Amount       float64 `json:"amount"`
	From         int64   `json:"from"`
	To           int64   `json:"to"`
	SubAccountID string  `json:"sub_account,omitempty"`
	InstrumentID int64   `json:"instrument_id,omitempty"`
}

// FundTransferResponse the response after a FundTransferRequest
type FundTransferResponse struct {
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	From       int64   `json:"from"`
	Result     bool    `json:"result"`
	To         int64   `json:"to"`
	TransferID int64   `json:"transfer_id"`
}

// WithdrawRequest used to request a withdrawal
type WithdrawRequest struct {
	Amount      int64   `json:"amount"`
	Currency    string  `json:"currency"`
	Destination int64   `json:"destination"`
	Fee         float64 `json:"fee"`
	ToAddress   string  `json:"to_address"`
	TradePwd    string  `json:"trade_pwd"`
}

// WithdrawResponse the response after a WithdrawRequest
type WithdrawResponse struct {
	Amount       float64 `json:"amount"`
	Currency     string  `json:"currency"`
	Result       bool    `json:"result"`
	WithdrawalID int64   `json:"withdrawal_id"`
}

// WithdrawalFeeResponse the response after requesting withdrawal fees
type WithdrawalFeeResponse struct {
	Available float64 `json:"available"`
	Balance   float64 `json:"balance"`
	Currency  string  `json:"currency"`
	Hold      float64 `json:"hold"`
}

// WithdrawalHistoryResponse the response after requesting withdrawal history
type WithdrawalHistoryResponse struct {
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	Fee       string  `json:"fee"`
	From      string  `json:"from"`
	Status    int64   `json:"status"`
	Timestamp string  `json:"timestamp"`
	To        string  `json:"to"`
	Txid      string  `json:"txid"`
	PaymentID string  `json:"payment_id"`
	Tag       string  `json:"tag"`
}

// GetBillDetailsRequest used in GetBillDetails
type GetBillDetailsRequest struct {
	Currency string
	Type     int64
	From     int64
	To       int64
	Limit    int64
}

// GetBillDetailsResponse contains bill details from a GetBillDetailsRequest request
type GetBillDetailsResponse struct {
	Amount    float64 `json:"amount"`
	Balance   int64   `json:"balance"`
	Currency  string  `json:"currency"`
	Fee       int64   `json:"fee"`
	LedgerID  int64   `json:"ledger_id"`
	Timestamp string  `json:"timestamp"`
	Typename  string  `json:"typename"`
}

// GetDepositAddressRespoonse contains deposit address details from a GetDepositAddress request
type GetDepositAddressRespoonse struct {
	Address   string `json:"address"`
	Tag       string `json:"tag"`
	PaymentID string `json:"payment_id,omitempty"`
	Currency  string `json:"currency"`
}

// GetDepositHistoryResponse contains deposit history details from a GetDepositHistory request
type GetDepositHistoryResponse struct {
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	Status        int64   `json:"status"`
	Timestamp     string  `json:"timestamp"`
	To            string  `json:"to"`
	TransactionID string  `json:"txid"`
}

// GetSpotTradingAccountResponse contains account data for spot account
type GetSpotTradingAccountResponse struct {
	Available string `json:"available"`
	Balance   string `json:"balance"`
	Currency  string `json:"currency"`
	Frozen    string `json:"frozen"`
	Hold      string `json:"hold"`
	Holds     string `json:"holds"`
	ID        string `json:"id"`
}

// GetSpotBillDetailsForCurrencyRequest contains request parameters fro GetSpotBillingDetailsForCurrency
type GetSpotBillDetailsForCurrencyRequest struct {
	Currency string `json:"currency"`
	From     int64  `json:"from,string,omitempty"`
	To       int64  `json:"to,string,omitempty"`
	Limit    int64  `json:"limit,string,omitempty"`
}

// GetSpotBillDetailsForCurrencyResponse contains the latest bills information
type GetSpotBillDetailsForCurrencyResponse struct {
	LedgerID         string          `json:"ledger_id"`
	Balance          string          `json:"balance"`
	CurrencyResponse string          `json:"currency"`
	Amount           string          `json:"amount"`
	Type             string          `json:"type"`
	TimeStamp        string          `json:"timestamp"`
	Details          SpotBillDetails `json:"details"`
}

// SpotBillDetails a child element of GetSpotBillDetailsForCurrencyResponse
// Contains order and instrument information
type SpotBillDetails struct {
	OrderID      string `json:"order_id"`
	InstrumentID string `json:"instrument_id"`
}

// PlaceSpotOrderRequest  contains request parameters fro PlaceSpotOrder
type PlaceSpotOrderRequest struct {
	ClientOID     string `json:"client_oid,omitempty"` // the order ID customized by yourself
	Type          string `json:"type"`                 // limit / market(default: limit)
	Side          string `json:"side"`                 // buy or sell
	InstrumentID  string `json:"instrument_id"`        // trading pair
	MarginTrading string `json:"margin_trading"`       // order type (The request value is 1)
	Size          string `json:"size"`
	Notional      string `json:"notional,omitempty"` //
	Price         string `json:"price,omitempty"`    // price (Limit order only)
}

// PlaceSpotOrderResponse contains the details from an order request
type PlaceSpotOrderResponse struct {
	ClientOid string `json:"client_oid"`
	OrderID   string `json:"order_id"`
	Result    bool   `json:"result"`
}

// CancelSpotOrderRequest contains the details from an order cancellation request
type CancelSpotOrderRequest struct {
	ClientOID    string `json:"client_oid,omitempty"` // the order ID created by yourself
	OrderID      int64  `json:"order_id,string"`      // order ID
	InstrumentID string `json:"instrument_id"`        // By providing this parameter, the corresponding order of a designated trading pair will be cancelled. If not providing this parameter, it will be back to error code.
}

// CancelSpotOrderResponse contains the results from CancelSpotOrder
type CancelSpotOrderResponse struct {
	ClientOID string `json:"client_oid"`
	OrderID   int64  `json:"order_id"`
	Result    bool   `json:"result"`
}

// CancelMultipleSpotOrdersRequest contains the details from multiple orders cancellation request
// CancelMultipleSpotOrdersRequest contains specific currency/order data
type CancelMultipleSpotOrdersRequest struct {
	OrderIDs     []int64 `json:"order_ids,omitempty"` // order ID. You may cancel up to 4 orders of a trading pair
	InstrumentID string  `json:"instrument_id"`       // by providing this parameter, the corresponding order of a designated trading pair will be cancelled. If not providing this parameter, it will be back to error code.
}

// CancelMultipleSpotOrdersResponse contains the results from CancelMultipleSpotOrders
type CancelMultipleSpotOrdersResponse struct {
	ClientOID string  `json:"client_oid"`
	OrderID   []int64 `json:"order_id,string"`
	Result    bool    `json:"result"`
}

// GetSpotOrdersRequest using in GetSpotOrders
type GetSpotOrdersRequest struct {
	Status string `json:"status"` // list the status of all orders (all, open, part_filled, canceling, filled, cancelled，ordering,failure)
	// （Multiple status separated by '|'，and '|' need encode to ' %7C'）
	InstrumentID string `json:"instrument_id"`          // trading pair ,information of all trading pair will be returned if the field is left blank
	From         int64  `json:"from,string,omitempty"`  // [optional]request page after this id (latest information) (eg. 1, 2, 3, 4, 5. There is only a 5 "from 4", while there are 1, 2, 3 "to 4")
	To           int64  `json:"to,string,omitempty"`    // [optional]request page after (older) this id.
	Limit        int64  `json:"limit,string,omitempty"` // [optional]number of results per request. Maximum 100. (default 100)
}

// GetSpotOrderResponse contains individual order details
type GetSpotOrderResponse struct {
	FilledNotional string `json:"filled_notional"`
	FilledSize     string `json:"filled_size"`
	InstrumentID   string `json:"instrument_id"`
	Notional       string `json:"notional"`
	OrderID        string `json:"order_id"`
	Price          string `json:"price"`
	Side           string `json:"side"`
	Size           string `json:"size"`
	Status         string `json:"status"`
	Timestamp      string `json:"timestamp"`
	Type           string `json:"type"`
}

// GetSpotOpenOrdersRequest using in GetSpotOpenOrders
type GetSpotOpenOrdersRequest struct {
	InstrumentID string `json:"instrument_id"`          // [optional]trading pair ,information of all trading pair will be returned if the field is left blank
	From         int64  `json:"from,string,omitempty"`  // [optional]request page after this id (latest information) (eg. 1, 2, 3, 4, 5. There is only a 5 "from 4", while there are 1, 2, 3 "to 4")
	To           int64  `json:"to,string,omitempty"`    // [optional]request page after (older) this id.
	Limit        int64  `json:"limit,string,omitempty"` // [optional]number of results per request. Maximum 100. (default 100)
}

// GetSpotOrderRequest used when requesting details for a single order
type GetSpotOrderRequest struct {
	OrderID      int64  `json:"order_id,string"` // [required] order ID
	InstrumentID string `json:"instrument_id"`   // [required]trading pair
}

// GetSpotTransactionDetailsRequest using in GetSpotTransactionDetails
type GetSpotTransactionDetailsRequest struct {
	InstrumentID string `json:"instrument_id"`          // [required]list all transaction details of this instrument_id.
	OrderID      int64  `json:"order_id,string"`        // [required]list all transaction details of this order_id.
	From         int64  `json:"from,string,omitempty"`  // [optional]request page after this id (latest information) (eg. 1, 2, 3, 4, 5. There is only a 5 "from 4", while there are 1, 2, 3 "to 4")
	To           int64  `json:"to,string,omitempty"`    // [optional]request page after (older) this id.
	Limit        int64  `json:"limit,string,omitempty"` // [optional]number of results per request. Maximum 100. (default 100)
}

// GetSpotTransactionDetailsResponse response data from GetSpotTransactionDetails
type GetSpotTransactionDetailsResponse struct {
	ExecType     string `json:"exec_type"`
	Fee          string `json:"fee"`
	InstrumentID string `json:"instrument_id"`
	LedgerID     string `json:"ledger_id"`
	OrderID      string `json:"order_id"`
	Price        string `json:"price"`
	Side         string `json:"side"`
	Size         string `json:"size"`
	Timestamp    string `json:"timestamp"`
}

// GetSpotTokenPairDetailsResponse contains market data from a GetSpotMarketData request
type GetSpotTokenPairDetailsResponse struct {
	BaseCurrency  string `json:"base_currency"`
	InstrumentID  string `json:"instrument_id"`
	MinSize       string `json:"min_size"`
	QuoteCurrency string `json:"quote_currency"`
	SizeIncrement string `json:"size_increment"`
	TickSize      string `json:"tick_size"`
}

// GetSpotOrderBookRequest Order boook request
type GetSpotOrderBookRequest struct {
	Size         int64   `json:"size,string,omitempty"`  // [optional]number of results per request. Maximum 200
	Depth        float64 `json:"depth,string,omitempty"` // [optional]the aggregation of the book. e.g . 0.1,0.001
	InstrumentID string  `json:"instrument_id"`          // [required] trading pairs
}

// GetSpotOrderBookResponse Order book response
type GetSpotOrderBookResponse struct {
	Timestamp string     `json:"timestamp"`
	Asks      [][]string `json:"asks"` // [[0]: "Price", [1]: "Size", [2]: "Num_orders"], ...
	Bids      [][]string `json:"bids"` // [[0]: "Price", [1]: "Size", [2]: "Num_orders"], ...
}

// GetSpotTokenPairsInformationResponse Ticker data response
type GetSpotTokenPairsInformationResponse struct {
	BaseVolume24h  string `json:"base_volume_24h"`  // 24 trading volume of the base currency
	BestAsk        string `json:"best_ask"`         // best ask price
	BestBid        string `json:"best_bid"`         // best bid price
	High24h        string `json:"high_24h"`         // 24 hour high
	InstrumentID   string `json:"instrument_id"`    // trading pair
	Last           string `json:"last"`             // last traded price
	Low24h         string `json:"low_24h"`          // 24 hour low
	Open24h        string `json:"open_24h"`         // 24 hour open
	QuoteVolume24h string `json:"quote_volume_24h"` // 24 trading volume of the quote currency
	Timestamp      string `json:"timestamp"`
}

// GetSpotFilledOrdersInformationRequest Filed orders request data
type GetSpotFilledOrdersInformationRequest struct {
	InstrumentID string `json:"instrument_id"`          // [required] trading pairs
	From         int64  `json:"from,string,omitempty"`  // [optional]number of results per request. Maximum 100. (default 100)
	To           int64  `json:"to,string,omitempty"`    // [optional]request page after (older) this id.
	Limit        int64  `json:"limit,string,omitempty"` // [optional]number of results per request. Maximum 100. (default 100)
}

// GetSpotFilledOrdersInformationResponse Filled orders response data
type GetSpotFilledOrdersInformationResponse struct {
	Price     string `json:"price"`
	Side      string `json:"side"`
	Size      string `json:"size"`
	Timestamp string `json:"timestamp"`
	TradeID   string `json:"trade_id"`
}

// GetSpotMarketDataRequest retrieves candel data information
type GetSpotMarketDataRequest struct {
	Start        string `json:"start,omitempty"` // [optional]start time in ISO 8601
	End          string `json:"end,omitempty"`   // [optional] end time in ISO 8601
	Granularity  int64  `json:"granularity"`     // The granularity field must be one of the following values: {60, 180, 300, 900, 1800, 3600, 7200, 14400, 43200, 86400, 604800}.
	InstrumentID string `json:"instrument_id"`   // [required] trading pairs
}

// GetSpotMarketDataResponse contains candle data from a GetSpotMarketDataRequest
// Return Parameters
// time 	string 	Start time
// open 	string 	Open price
// high 	string 	Highest price
// low 	string 	Lowest price
// close 	string 	Close price
// volume 	string 	Trading volume
type GetSpotMarketDataResponse []interface{}

// GetMarginAccountsResponse contains margin account data for each currency
type GetMarginAccountsResponse struct {
	InstrumentID     string `json:"instrument_id,omitempty"`
	LiquidationPrice string `json:"liquidation_price"`
	ProductID        string `json:"product_id,omitempty"`
	RiskRate         string `json:"risk_rate"`
	Currencies       map[string]MarginAccountInfo
}

// MarginAccountInfo contains individual currency information
type MarginAccountInfo struct {
	Available  float64 `json:"available,string"`
	Balance    float64 `json:"balance,string"`
	Borrowed   float64 `json:"borrowed,string"`
	Frozen     float64 `json:"frozen,string"`
	Hold       float64 `json:"hold,string"`
	Holds      float64 `json:"holds,string"`
	LendingFee float64 `json:"lending_fee,string"`
}

// GetMarginAccountSettingsResponse contains the results from GetMarginAccountSettings
type GetMarginAccountSettingsResponse struct {
	InstrumentID string `json:"instrument_id"`
	ProductID    string `json:"product_id"`
	Currencies   map[string]MarginAccountSettingsInfo
}

// MarginAccountSettingsInfo contains individual currency data
type MarginAccountSettingsInfo struct {
	Available     float64 `json:"available,string"`
	Leverage      float64 `json:"leverage,string"`
	LeverageRatio float64 `json:"leverage_ratio,string"`
	Rate          float64 `json:"rate,string"`
}

// GetMarginLoanHistoryRequest optional parameters for a GetMarginAccountSettings
type GetMarginLoanHistoryRequest struct {
	InstrumentID string // [optional] Used when a specific currency response is desired
	Status       int64  `json:"status,string,omitempty"` // [optional] status(0: outstanding 1: repaid)
	From         int64  `json:"from,string,omitempty"`   // [optional]request page from(newer) this id.
	To           int64  `json:"to,string,omitempty"`     // [optional]request page to(older) this id.
	Limit        int64  `json:"limit,string,omitempty"`  // [optional]number of results per request. Maximum 100.(default 100)
}

// GetMarginLoanHistoryResponse loan history of the margin trading account
type GetMarginLoanHistoryResponse struct {
	Amount           float64 `json:"amount,string"`
	BorrowID         int64   `json:"borrow_id"`
	CreatedAt        string  `json:"created_at"`
	Currency         string  `json:"currency"`
	ForceRepayTime   string  `json:"force_repay_time"`
	InstrumentID     string  `json:"instrument_id"`
	Interest         float64 `json:"interest,string"`
	LastInterestTime string  `json:"last_interest_time"`
	PaidInterest     float64 `json:"paid_interest,string"`
	ProductID        string  `json:"product_id"`
	Rate             float64 `json:"rate,string"`
	RepayAmount      string  `json:"repay_amount"`
	RepayInterest    string  `json:"repay_interest"`
	ReturnedAmount   float64 `json:"returned_amount,string"`
	Timestamp        string  `json:"timestamp"`
}

// OpenMarginLoanRequest required to open a loan
type OpenMarginLoanRequest struct {
	QuoteCurrency string  `json:"currency"`      // [required] Second currency eg BTC-USDT: USDT is quote
	InstrumentID  string  `json:"instrument_id"` // [required] Full pair BTC-USDT
	Amount        float64 `json:"amount,string"` // [required] Amount wanting to borrow
}

// OpenMarginLoanResponse returned ID from a loan request
type OpenMarginLoanResponse struct {
	BorrowID int64 `json:"borrow_id"`
	Result   bool  `json:"result"`
}

// RepayMarginLoanRequest required params for RepayMarginLoan
type RepayMarginLoanRequest struct {
	Amount        float64 `json:"amount,string"` // [required] amount repaid
	BorrowID      float64 `json:"borrow_id"`     // [optional] borrow ID . all borrowed token under this trading pair will be repay if the field is left blank
	QuoteCurrency string  `json:"currency"`      // [required] Second currency eg BTC-USDT: USDT is quote
	InstrumentID  string  `json:"instrument_id"` // [required] Full pair BTC-USDT
}

// RepayMarginLoanResponse holds response for RepayMarginLoan
type RepayMarginLoanResponse struct {
	RepaymentID int64 `json:"repayment_id"`
	Result      bool  `json:"result"`
}

// OrderStatus Holds OKGroup order status values
var OrderStatus = map[int]string{
	-3: "pending cancel",
	-2: "cancelled",
	-1: "failed",
	0:  "pending",
	1:  "sending",
	2:  "sent",
	3:  "email confirmation",
	4:  "manual confirmation",
	5:  "awaiting identity confirmation",
}

// SpotInstrument contains spot data
type SpotInstrument struct {
	BaseCurrency   string  `json:"base_currency"`
	BaseIncrement  float64 `json:"base_increment,string"`
	BaseMinSize    float64 `json:"base_min_size,string"`
	InstrumentID   string  `json:"instrument_id"`
	MinSize        float64 `json:"min_size,string"`
	ProductID      string  `json:"product_id"`
	QuoteCurrency  string  `json:"quote_currency"`
	QuoteIncrement float64 `json:"quote_increment,string"`
	SizeIncrement  float64 `json:"size_increment,string"`
	TickSize       float64 `json:"tick_size,string"`
}

// MultiStreamData contains raw data from okex
type MultiStreamData struct {
	Channel string          `json:"channel"`
	Data    json.RawMessage `json:"data"`
}

// TickerStreamData contains ticker stream data from okex
type TickerStreamData struct {
	Buy       string  `json:"buy"`
	Change    string  `json:"change"`
	High      string  `json:"high"`
	Low       string  `json:"low"`
	Last      string  `json:"last"`
	Sell      string  `json:"sell"`
	DayLow    string  `json:"dayLow"`
	DayHigh   string  `json:"dayHigh"`
	Timestamp float64 `json:"timestamp"`
	Vol       string  `json:"vol"`
}

// DealsStreamData defines Deals data
type DealsStreamData = [][]string

// KlineStreamData defines kline data
type KlineStreamData = [][]string

// DepthStreamData defines orderbook depth
type DepthStreamData struct {
	Asks      [][]string `json:"asks"`
	Bids      [][]string `json:"bids"`
	Timestamp float64    `json:"timestamp"`
}

// SpotPrice holds date and ticker price price for contracts.
type SpotPrice struct {
	Date   string `json:"date"`
	Ticker struct {
		Buy        float64 `json:"buy,string"`
		ContractID float64 `json:"contract_id"`
		High       float64 `json:"high,string"`
		Low        float64 `json:"low,string"`
		Last       float64 `json:"last,string"`
		Sell       float64 `json:"sell,string"`
		UnitAmount float64 `json:"unit_amount,string"`
		Vol        float64 `json:"vol,string"`
	} `json:"ticker"`
	Result bool        `json:"result"`
	Error  interface{} `json:"error_code"`
}

// SpotDepth response depth
type SpotDepth struct {
	Asks   []interface{} `json:"asks"`
	Bids   []interface{} `json:"bids"`
	Result bool          `json:"result"`
	Error  interface{}   `json:"error_code"`
}

// ActualSpotDepthRequestParams represents Klines request data.
type ActualSpotDepthRequestParams struct {
	Symbol string `json:"symbol"` // Symbol; example ltc_btc
	Size   int64  `json:"size"`   // value: 1-200
}

// ActualSpotDepth better manipulated structure to return
type ActualSpotDepth struct {
	Asks []struct {
		Price  float64
		Volume float64
	}
	Bids []struct {
		Price  float64
		Volume float64
	}
}

// KlinesRequestParams represents Klines request data.
type KlinesRequestParams struct {
	Symbol string       // Symbol; example btcusdt, bccbtc......
	Type   TimeInterval // Kline data time interval; 1min, 5min, 15min......
	Size   int64        // Size; [1-2000]
	Since  int64        // Since timestamp, return data after the specified timestamp (for example, 1417536000000)
}

// TimeInterval represents interval enum.
type TimeInterval string

// vars for time intervals
var (
	TimeIntervalMinute         = TimeInterval("1min")
	TimeIntervalThreeMinutes   = TimeInterval("3min")
	TimeIntervalFiveMinutes    = TimeInterval("5min")
	TimeIntervalFifteenMinutes = TimeInterval("15min")
	TimeIntervalThirtyMinutes  = TimeInterval("30min")
	TimeIntervalHour           = TimeInterval("1hour")
	TimeIntervalFourHours      = TimeInterval("4hour")
	TimeIntervalSixHours       = TimeInterval("6hour")
	TimeIntervalTwelveHours    = TimeInterval("12hour")
	TimeIntervalDay            = TimeInterval("1day")
	TimeIntervalThreeDays      = TimeInterval("3day")
	TimeIntervalWeek           = TimeInterval("1week")
)

// WithdrawalFees the large list of predefined withdrawal fees
// Prone to change, using highest value
var WithdrawalFees = map[string]float64{
	symbol.ZRX:   10,
	symbol.ACE:   2.2,
	symbol.ACT:   0.01,
	symbol.AAC:   5,
	symbol.AE:    1,
	symbol.AIDOC: 17,
	symbol.AST:   8,
	symbol.SOC:   20,
	symbol.ABT:   3,
	symbol.ARK:   0.1,
	symbol.ATL:   1.5,
	symbol.AVT:   1,
	symbol.BNT:   1,
	symbol.BKX:   3,
	symbol.BEC:   4,
	symbol.BTC:   0.0005,
	symbol.BCH:   0.0001,
	symbol.BCD:   0.02,
	symbol.BTG:   0.001,
	symbol.VEE:   100,
	symbol.BRD:   1.5,
	symbol.CTR:   7,
	symbol.LINK:  10,
	symbol.CAG:   2,
	symbol.CHAT:  10,
	symbol.CVC:   10,
	symbol.CIC:   150,
	symbol.CBT:   10,
	symbol.CAN:   3,
	symbol.CMT:   10,
	symbol.DADI:  10,
	symbol.DASH:  0.002,
	symbol.DAT:   2,
	symbol.MANA:  20,
	symbol.DCR:   0.03,
	symbol.DPY:   0.8,
	symbol.DENT:  100,
	symbol.DGD:   0.2,
	symbol.DNT:   20,
	symbol.EDO:   2,
	symbol.DNA:   3,
	symbol.ENG:   5,
	symbol.ENJ:   20,
	symbol.ETH:   0.01,
	symbol.ETC:   0.001,
	symbol.LEND:  10,
	symbol.EVX:   1.5,
	symbol.XUC:   5.8,
	symbol.FAIR:  15,
	symbol.FIRST: 6,
	symbol.FUN:   40,
	symbol.GTC:   40,
	symbol.GNX:   8,
	symbol.GTO:   10,
	symbol.GSC:   20,
	symbol.GNT:   5,
	symbol.HMC:   40,
	symbol.HOT:   10,
	symbol.ICN:   2,
	symbol.INS:   2.5,
	symbol.INT:   10,
	symbol.IOST:  100,
	symbol.ITC:   2,
	symbol.IPC:   2.5,
	symbol.KNC:   2,
	symbol.LA:    3,
	symbol.LEV:   20,
	symbol.LIGHT: 100,
	symbol.LSK:   0.4,
	symbol.LTC:   0.001,
	symbol.LRC:   7,
	symbol.MAG:   34,
	symbol.MKR:   0.002,
	symbol.MTL:   0.5,
	symbol.AMM:   5,
	symbol.MITH:  20,
	symbol.MDA:   2,
	symbol.MOF:   5,
	symbol.MCO:   0.2,
	symbol.MTH:   35,
	symbol.NGC:   1.5,
	symbol.NANO:  0.2,
	symbol.NULS:  2,
	symbol.OAX:   6,
	symbol.OF:    600,
	symbol.OKB:   0,
	symbol.MOT:   1.5,
	symbol.OMG:   0.1,
	symbol.RNT:   13,
	symbol.POE:   30,
	symbol.PPT:   0.2,
	symbol.PST:   10,
	symbol.PRA:   4,
	symbol.QTUM:  0.01,
	symbol.QUN:   20,
	symbol.QVT:   10,
	symbol.RDN:   0.3,
	symbol.READ:  20,
	symbol.RCT:   15,
	symbol.RFR:   200,
	symbol.REF:   0.2,
	symbol.REN:   50,
	symbol.REQ:   15,
	symbol.R:     2,
	symbol.RCN:   20,
	symbol.XRP:   0.15,
	symbol.SALT:  0.5,
	symbol.SAN:   1,
	symbol.KEY:   50,
	symbol.SSC:   8,
	symbol.SHOW:  150,
	symbol.SC:    200,
	symbol.OST:   3,
	symbol.SNGLS: 20,
	symbol.SMT:   8,
	symbol.SNM:   20,
	symbol.SPF:   5,
	symbol.SNT:   50,
	symbol.STORJ: 2,
	symbol.SUB:   4,
	symbol.SNC:   10,
	symbol.SWFTC: 350,
	symbol.PAY:   0.5,
	symbol.USDT:  2,
	symbol.TRA:   500,
	symbol.THETA: 20,
	symbol.TNB:   40,
	symbol.TCT:   50,
	symbol.TOPC:  20,
	symbol.TIO:   2.5,
	symbol.TRIO:  200,
	symbol.TRUE:  4,
	symbol.UCT:   10,
	symbol.UGC:   12,
	symbol.UKG:   2.5,
	symbol.UTK:   3,
	symbol.VIB:   6,
	symbol.VIU:   40,
	symbol.WTC:   0.4,
	symbol.WFEE:  500,
	symbol.WRC:   48,
	symbol.YEE:   70,
	symbol.YOYOW: 10,
	symbol.ZEC:   0.001,
	symbol.ZEN:   0.07,
	symbol.ZIL:   20,
	symbol.ZIP:   1000,
}
