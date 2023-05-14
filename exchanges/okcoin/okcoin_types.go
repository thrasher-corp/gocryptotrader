package okcoin

import (
	"encoding/json"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
)

var (
	errNoAccountDepositAddress   = errors.New("no account deposit address")
	errIncorrectCandleDataLength = errors.New("incorrect candles data length")
)

// PerpSwapInstrumentData stores instrument data for perpetual swap contracts
type PerpSwapInstrumentData struct {
	InstrumentID        string  `json:"instrument_id"`
	UnderlyingIndex     string  `json:"underlying_index"`
	QuoteCurrency       string  `json:"quote_currency"`
	Coin                string  `json:"coin"`
	ContractValue       float64 `json:"contract_val,string"`
	Listing             string  `json:"listing"`
	Delivery            string  `json:"delivery"`
	SizeIncrement       float64 `json:"size_increment,string"`
	TickSize            float64 `json:"tick_size,string"`
	BaseCurrency        string  `json:"base_currency"`
	Underlying          string  `json:"underlying"`
	SettlementCurrency  string  `json:"settlement_currency"`
	IsInverse           bool    `json:"is_inverse,string"`
	Category            float64 `json:"category,string"`
	ContractValCurrency string  `json:"contract_val_currency"`
}

// TradingPairData stores data about a trading pair
type TradingPairData struct {
	BaseCurrency  string  `json:"base_currency"`
	InstrumentID  string  `json:"instrument_id"`
	MinSize       float64 `json:"min_size,string"`
	QuoteCurrency string  `json:"quote_currency"`
	SizeIncrement string  `json:"size_increment"`
	TickSize      float64 `json:"tick_size,string"`
}

// SwapInstrumentsData stores instruments data for perpetual swap contracts
type SwapInstrumentsData struct {
	InstrumentID          string  `json:"instrument_id"`
	UnderlyingIndex       string  `json:"underlying_index"`
	QuoteCurrency         string  `json:"quote_currency"`
	Coin                  string  `json:"coin"`
	ContractValue         float64 `json:"contract_val,string"`
	Listing               string  `json:"listing"`
	Delivery              string  `json:"delivery"`
	SizeIncrement         float64 `json:"size_increment,string"`
	TickSize              float64 `json:"tick_size,string"`
	BaseCurrency          string  `json:"base_currency"`
	Underlying            string  `json:"underlying"`
	SettlementCurrency    string  `json:"settlement_currency"`
	IsInverse             bool    `json:"is_inverse,string"`
	Category              int64   `json:"category,string"`
	ContractValueCurrency string  `json:"contract_val_currency"`
}

// MarginData stores margin trading data for a currency
type MarginData struct {
	Available     float64 `json:"available,string"`
	Leverage      float64 `json:"leverage,string"`
	LeverageRatio float64 `json:"leverage_ratio,string"`
	Rate          float64 `json:"rate,string"`
}

// MarginCurrencyData stores currency data for margin trading
type MarginCurrencyData struct {
	Data         map[string]MarginData
	InstrumentID string `json:"instrument_id"`
	ProductID    string `json:"product_id"`
}

// TickerData stores ticker data
type TickerData struct {
	InstType        string         `json:"instType"`
	InstrumentID    string         `json:"instId"`
	LastTradedPrice float64        `json:"last,string"`
	LastTradedSize  float64        `json:"lastSz,string"`
	BestAskPrice    float64        `json:"askPx,string"`
	BestAskSize     float64        `json:"askSz,string"`
	BestBidPrice    float64        `json:"bidPx,string"`
	BestBidSize     float64        `json:"bidSz,string"`
	Open24H         float64        `json:"open24h,string"` // Open price in the past 24 hours
	High24H         float64        `json:"high24h,string"` // Highest price in the past 24 hours
	Low24H          float64        `json:"low24h,string"`  // Lowest price in the past 24 hours
	VolCcy24H       string         `json:"volCcy24h"`      // 24h trading volume, with a unit of currency. The value is the quantity in quote currency.
	Vol24H          string         `json:"vol24h"`         // 24h trading volume, with a unit of contract. The value is the quantity in base currency.
	Timestamp       okcoinMilliSec `json:"ts"`
	OpenPriceInUtc0 float64        `json:"sodUtc0,string"`
	OpenPriceInUtc8 float64        `json:"sodUtc8,string"`
}

// PerpSwapFundingRates stores funding rates data
type PerpSwapFundingRates struct {
	InstrumentID string    `json:"instrument_id"`
	FundingRate  float64   `json:"funding_rate,string"`
	RealizedRate float64   `json:"realized_rate,string"`
	InterestRate float64   `json:"interest_rate,string"`
	FundingTime  time.Time `json:"funding_time"`
}

// GetAccountCurrenciesResponse response data for GetAccountCurrencies
type GetAccountCurrenciesResponse struct {
	Name          string  `json:"name"`
	Currency      string  `json:"currency"`
	Chain         string  `json:"chain"`
	CanInternal   bool    `json:"can_internal,string"`
	CanWithdraw   bool    `json:"can_withdraw,string"`
	CanDeposit    bool    `json:"can_deposit,string"`
	MinWithdrawal float64 `json:"min_withdrawal"`
}

// WalletInformationResponse response data for WalletInformation
type WalletInformationResponse struct {
	Available float64 `json:"available"`
	Balance   float64 `json:"balance"`
	Currency  string  `json:"currency"`
	Hold      float64 `json:"hold"`
}

// TransferAccountFundsRequest request data for TransferAccountFunds
type TransferAccountFundsRequest struct {
	Currency     string  `json:"currency"`                // [required] token
	Amount       float64 `json:"amount"`                  // [required] Transfer amount
	From         int64   `json:"from"`                    // [required] the remitting account (0: sub account 1: spot 3: futures 4:C2C 5: margin 6: wallet 7:ETT 8:PiggyBank 9：swap)
	To           int64   `json:"to"`                      // [required] the beneficiary account(0: sub account 1:spot 3: futures 4:C2C 5: margin 6: wallet 7:ETT 8:PiggyBank 9 :swap)
	SubAccountID string  `json:"sub_account,omitempty"`   // [optional] sub account name
	InstrumentID int64   `json:"instrument_id,omitempty"` // [optional] margin token pair ID, for supported pairs only
}

// TransferAccountFundsResponse response data for TransferAccountFunds
type TransferAccountFundsResponse struct {
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	From       int64   `json:"from"`
	Result     bool    `json:"result"`
	To         int64   `json:"to"`
	TransferID int64   `json:"transfer_id"`
}

// AccountWithdrawRequest request data for AccountWithdrawRequest
type AccountWithdrawRequest struct {
	Amount      float64 `json:"amount"`      // [required] withdrawal amount
	Currency    string  `json:"currency"`    // [required] token
	Destination int64   `json:"destination"` // [required] withdrawal address(2:OKCoin International 3:others)
	Fee         float64 `json:"fee"`         // [required] Network transaction fee≥0. Withdrawals to OKCoin are fee-free, please set as 0. Withdrawal to external digital asset address requires network transaction fee.
	ToAddress   string  `json:"to_address"`  // [required] verified digital asset address, email or mobile number,some digital asset address format is address+tag , eg: "ARDOR-7JF3-8F2E-QUWZ-CAN7F：123456"
	TradePwd    string  `json:"trade_pwd"`   // [required] fund password
}

// AccountWithdrawResponse response data for AccountWithdrawResponse
type AccountWithdrawResponse struct {
	Amount       float64 `json:"amount"`
	Currency     string  `json:"currency"`
	Result       bool    `json:"result"`
	WithdrawalID int64   `json:"withdrawal_id"`
}

// GetAccountWithdrawalFeeResponse response data for GetAccountWithdrawalFee
type GetAccountWithdrawalFeeResponse struct {
	Currency string  `json:"currency"`
	MinFee   float64 `json:"min_fee,string"`
	MaxFee   float64 `json:"max_fee,string"`
}

// WithdrawalHistoryResponse response data for WithdrawalHistoryResponse
type WithdrawalHistoryResponse struct {
	Amount        float64   `json:"amount,string"`
	Currency      string    `json:"currency"`
	Fee           string    `json:"fee"`
	From          string    `json:"from"`
	Status        int64     `json:"status,string"`
	Timestamp     time.Time `json:"timestamp"`
	To            string    `json:"to"`
	TransactionID string    `json:"txid"`
	PaymentID     string    `json:"payment_id"`
	Tag           string    `json:"tag"`
}

// GetAccountBillDetailsRequest request data for GetAccountBillDetailsRequest
type GetAccountBillDetailsRequest struct {
	Currency string `url:"currency,omitempty"` // [optional] token ,information of all tokens will be returned if the field is left blank
	Type     int64  `url:"type,omitempty"`     // [optional] 1:deposit 2:withdrawal 13:cancel withdrawal 18: into futures account 19: out of futures account 20:into sub account 21:out of sub account 28: claim 29: into ETT account 30: out of ETT account 31: into C2C account 32:out of C2C account 33: into margin account 34: out of margin account 37: into spot account 38: out of spot account
	From     int64  `url:"from,omitempty"`     // [optional] you would request pages after this page.
	To       int64  `url:"to,omitempty"`       // [optional] you would request pages before this page
	Limit    int64  `url:"limit,omitempty"`    // [optional] Number of results per request. Maximum 100. (default 100)
}

// GetAccountBillDetailsResponse response data for GetAccountBillDetails
type GetAccountBillDetailsResponse struct {
	Amount    float64   `json:"amount"`
	Balance   int64     `json:"balance"`
	Currency  string    `json:"currency"`
	Fee       int64     `json:"fee"`
	LedgerID  int64     `json:"ledger_id"`
	Timestamp time.Time `json:"timestamp"`
	Typename  string    `json:"typename"`
}

// GetDepositAddressResponse response data for GetDepositAddress
type GetDepositAddressResponse struct {
	Address   string `json:"address"`
	Tag       string `json:"tag"`
	PaymentID string `json:"payment_id,omitempty"`
	Currency  string `json:"currency"`
}

// GetAccountDepositHistoryResponse response data for GetAccountDepositHistory
type GetAccountDepositHistoryResponse struct {
	Amount        float64   `json:"amount,string"`
	Currency      string    `json:"currency"`
	From          string    `json:"from"`
	To            string    `json:"to"`
	Timestamp     time.Time `json:"timestamp"`
	Status        int64     `json:"status,string"`
	TransactionID string    `json:"txid"`
}

// GetSpotTradingAccountResponse response data for GetSpotTradingAccount
type GetSpotTradingAccountResponse struct {
	Available string `json:"available"`
	Balance   string `json:"balance"`
	Currency  string `json:"currency"`
	Frozen    string `json:"frozen"`
	Hold      string `json:"hold"`
	Holds     string `json:"holds"`
	ID        string `json:"id"`
}

// GetSpotBillDetailsForCurrencyRequest request data for GetSpotBillDetailsForCurrency
type GetSpotBillDetailsForCurrencyRequest struct {
	Currency string `url:"-"`                      // [required] token
	From     int64  `url:"from,string,omitempty"`  // [optional] request page before(newer) this id.
	To       int64  `url:"to,string,omitempty"`    // [optional] request page after(older) this id.
	Limit    int64  `url:"limit,string,omitempty"` // [optional] number of results per request. Maximum 100.(default 100)
}

// GetSpotBillDetailsForCurrencyResponse response data for GetSpotBillDetailsForCurrency
type GetSpotBillDetailsForCurrencyResponse struct {
	LedgerID         string          `json:"ledger_id"`
	Balance          string          `json:"balance"`
	CurrencyResponse string          `json:"currency"`
	Amount           string          `json:"amount"`
	Type             string          `json:"type"`
	TimeStamp        time.Time       `json:"timestamp"`
	Details          SpotBillDetails `json:"details"`
}

// SpotBillDetails response data for GetSpotBillDetailsForCurrency
type SpotBillDetails struct {
	OrderID      string `json:"order_id"`
	InstrumentID string `json:"instrument_id"`
}

// PlaceOrderRequest request data for placing an order
type PlaceOrderRequest struct {
	ClientOID     string `json:"client_oid,omitempty"` // the order ID customized by yourself
	Type          string `json:"type"`                 // limit / market(default: limit)
	Side          string `json:"side"`                 // buy or sell
	InstrumentID  string `json:"instrument_id"`        // trading pair
	MarginTrading string `json:"margin_trading"`       // margin trading
	OrderType     string `json:"order_type"`           // order type (0: Normal order (Unfilled and 0 imply normal limit order) 1: Post only 2: Fill or Kill 3: Immediate Or Cancel
	Size          string `json:"size"`
	Notional      string `json:"notional,omitempty"` //
	Price         string `json:"price,omitempty"`    // price (Limit order only)
}

// PlaceOrderResponse response data for PlaceSpotOrder
type PlaceOrderResponse struct {
	ClientOid string `json:"client_oid"`
	OrderID   string `json:"order_id"`
	Result    bool   `json:"result"`
}

// CancelSpotOrderRequest request data for CancelSpotOrder
type CancelSpotOrderRequest struct {
	ClientOID    string `json:"client_oid,omitempty"` // the order ID created by yourself
	OrderID      int64  `json:"order_id,string"`      // order ID
	InstrumentID string `json:"instrument_id"`        // By providing this parameter, the corresponding order of a designated trading pair will be cancelled. If not providing this parameter, it will be back to error code.
}

// CancelSpotOrderResponse response data for CancelSpotOrder
type CancelSpotOrderResponse struct {
	ClientOID string `json:"client_oid"`
	OrderID   int64  `json:"order_id,string"`
	Result    bool   `json:"result"`
}

// CancelMultipleSpotOrdersRequest request data for CancelMultipleSpotOrders
type CancelMultipleSpotOrdersRequest struct {
	OrderIDs     []int64 `json:"order_ids,omitempty"` // order ID. You may cancel up to 4 orders of a trading pair
	InstrumentID string  `json:"instrument_id"`       // by providing this parameter, the corresponding order of a designated trading pair will be cancelled. If not providing this parameter, it will be back to error code.
}

// CancelMultipleSpotOrdersResponse response data for CancelMultipleSpotOrders
type CancelMultipleSpotOrdersResponse struct {
	ClientOID string `json:"client_oid"`
	OrderID   int64  `json:"order_id,string"`
	Result    bool   `json:"result"`
	Error     error  // Placeholder to store errors
}

// GetSpotOrdersRequest request data for GetSpotOrders
type GetSpotOrdersRequest struct {
	Status string `url:"status"` // list the status of all orders (all, open, part_filled, canceling, filled, cancelled，ordering,failure)
	// （Multiple status separated by '|'，and '|' need encode to ' %7C'）
	InstrumentID string `url:"instrument_id"`          // trading pair ,information of all trading pair will be returned if the field is left blank
	From         int64  `url:"from,string,omitempty"`  // [optional] request page after this id (latest information) (eg. 1, 2, 3, 4, 5. There is only a 5 "from 4", while there are 1, 2, 3 "to 4")
	To           int64  `url:"to,string,omitempty"`    // [optional] request page after (older) this id.
	Limit        int64  `url:"limit,string,omitempty"` // [optional] number of results per request. Maximum 100. (default 100)
}

// GetSpotOrderResponse response data for GetSpotOrders
type GetSpotOrderResponse struct {
	FilledNotional float64   `json:"filled_notional,string"`
	FilledSize     float64   `json:"filled_size,string"`
	InstrumentID   string    `json:"instrument_id"`
	Notional       string    `json:"notional"`
	OrderID        string    `json:"order_id"`
	Price          float64   `json:"price,string"`
	PriceAvg       float64   `json:"price_avg,string"`
	Side           string    `json:"side"`
	Size           float64   `json:"size,string"`
	Status         string    `json:"status"`
	Timestamp      time.Time `json:"timestamp"`
	Type           string    `json:"type"`
}

// GetSpotOpenOrdersRequest request data for GetSpotOpenOrders
type GetSpotOpenOrdersRequest struct {
	InstrumentID string `json:"instrument_id"`          // [optional] trading pair ,information of all trading pair will be returned if the field is left blank
	From         int64  `json:"from,string,omitempty"`  // [optional] request page after this id (latest information) (eg. 1, 2, 3, 4, 5. There is only a 5 "from 4", while there are 1, 2, 3 "to 4")
	To           int64  `json:"to,string,omitempty"`    // [optional] request page after (older) this id.
	Limit        int64  `json:"limit,string,omitempty"` // [optional] number of results per request. Maximum 100. (default 100)
}

// GetSpotOrderRequest request data for GetSpotOrder
type GetSpotOrderRequest struct {
	OrderID      string `url:"-"`             // [required] order ID
	InstrumentID string `url:"instrument_id"` // [required]trading pair
}

// GetSpotTransactionDetailsRequest request data for GetSpotTransactionDetails
type GetSpotTransactionDetailsRequest struct {
	InstrumentID string `url:"instrument_id"`          // [required]list all transaction details of this instrument_id.
	OrderID      int64  `url:"order_id,string"`        // [required]list all transaction details of this order_id.
	From         int64  `url:"from,string,omitempty"`  // [optional] request page after this id (latest information) (eg. 1, 2, 3, 4, 5. There is only a 5 "from 4", while there are 1, 2, 3 "to 4")
	To           int64  `url:"to,string,omitempty"`    // [optional] request page after (older) this id.
	Limit        int64  `url:"limit,string,omitempty"` // [optional] number of results per request. Maximum 100. (default 100)
}

// GetSpotTransactionDetailsResponse response data for GetSpotTransactionDetails
type GetSpotTransactionDetailsResponse struct {
	ExecType     string    `json:"exec_type"`
	Fee          string    `json:"fee"`
	InstrumentID string    `json:"instrument_id"`
	LedgerID     string    `json:"ledger_id"`
	OrderID      string    `json:"order_id"`
	Price        string    `json:"price"`
	Side         string    `json:"side"`
	Size         string    `json:"size"`
	Timestamp    time.Time `json:"timestamp"`
}

// GetSpotTokenPairDetailsResponse response data for GetSpotTokenPairDetails
type GetSpotTokenPairDetailsResponse struct {
	BaseCurrency  string `json:"base_currency"`
	InstrumentID  string `json:"instrument_id"`
	MinSize       string `json:"min_size"`
	QuoteCurrency string `json:"quote_currency"`
	SizeIncrement string `json:"size_increment"`
	TickSize      string `json:"tick_size"`
}

// GetOrderBookRequest request data for GetOrderBook
type GetOrderBookRequest struct {
	Size         int64   `url:"size,string,omitempty"`  // [optional] number of results per request. Maximum 200
	Depth        float64 `url:"depth,string,omitempty"` // [optional] the aggregation of the book. e.g . 0.1,0.001
	InstrumentID string  `url:"-"`                      // [required] trading pairs
}

// GetOrderBookResponse response data
type GetOrderBookResponse struct {
	Timestamp okcoinMilliSec `json:"ts"`
	Asks      [][4]string    `json:"asks"` // [[0]: "Price", [1]: "Size", [2]: "Num_orders"], ...
	Bids      [][4]string    `json:"bids"` // [[0]: "Price", [1]: "Size", [2]: "Num_orders"], ...
}

// GetSpotTokenPairsInformationResponse response data for GetSpotTokenPairsInformation
type GetSpotTokenPairsInformationResponse struct {
	BaseVolume24h  float64       `json:"base_volume_24h,string"`  // 24 trading volume of the base currency
	BestAsk        float64       `json:"best_ask,string"`         // best ask price
	BestBid        float64       `json:"best_bid,string"`         // best bid price
	High24h        float64       `json:"high_24h,string"`         // 24 hour high
	InstrumentID   currency.Pair `json:"instrument_id"`           // trading pair
	Last           float64       `json:"last,string"`             // last traded price
	Low24h         float64       `json:"low_24h,string"`          // 24 hour low
	Open24h        float64       `json:"open_24h,string"`         // 24 hour open
	QuoteVolume24h float64       `json:"quote_volume_24h,string"` // 24 trading volume of the quote currency
	Timestamp      time.Time     `json:"timestamp"`
}

// GetSpotFilledOrdersInformationRequest request data for GetSpotFilledOrdersInformation
type GetSpotFilledOrdersInformationRequest struct {
	InstrumentID string `url:"-"`                      // [required] trading pairs
	From         int64  `url:"from,string,omitempty"`  // [optional] number of results per request. Maximum 100. (default 100)
	To           int64  `url:"to,string,omitempty"`    // [optional] request page after (older) this id.
	Limit        int64  `url:"limit,string,omitempty"` // [optional] number of results per request. Maximum 100. (default 100)
}

// GetSpotFilledOrdersInformationResponse response data for GetSpotFilledOrdersInformation
type GetSpotFilledOrdersInformationResponse struct {
	Price     float64   `json:"price,string"`
	Side      string    `json:"side"`
	Size      float64   `json:"size,string"`
	Timestamp time.Time `json:"timestamp"`
	TradeID   string    `json:"trade_id"`
}

// GetMarketDataRequest request data for GetMarketData
type GetMarketDataRequest struct {
	Asset        asset.Item
	Start        string `url:"start,omitempty"` // [optional] start time in ISO 8601
	End          string `url:"end,omitempty"`   // [optional] end time in ISO 8601
	Granularity  string `url:"granularity"`     // The granularity field must be one of the following values: {60, 180, 300, 900, 1800, 3600, 7200, 14400, 43200, 86400, 604800}.
	InstrumentID string `url:"-"`               // [required] trading pairs
}

// GetMarketDataResponse response data for GetMarketData
// Return Parameters
// time 	string 	Start time
// open 	string 	Open price
// high 	string 	Highest price
// low 	string 	Lowest price
// close 	string 	Close price
// volume 	string 	Trading volume
type GetMarketDataResponse struct {
}

// GetMarginAccountsResponse response data for GetMarginAccounts
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

// GetMarginAccountSettingsResponse response data for GetMarginAccountSettings
type GetMarginAccountSettingsResponse struct {
	InstrumentID string `json:"instrument_id"`
	ProductID    string `json:"product_id"`
	Currencies   map[string]MarginAccountSettingsInfo
}

// GetMarginBillDetailsRequest request data for GetMarginBillDetails
type GetMarginBillDetailsRequest struct {
	InstrumentID string `url:"-"`               // [required] trading pair
	Type         int64  `url:"type,omitempty"`  // [optional] 1:deposit 2:withdrawal 13:cancel withdrawal 18: into futures account 19: out of futures account 20:into sub account 21:out of sub account 28: claim 29: into ETT account 30: out of ETT account 31: into C2C account 32:out of C2C account 33: into margin account 34: out of margin account 37: into spot account 38: out of spot account
	From         int64  `url:"from,omitempty"`  // [optional] you would request pages after this page.
	To           int64  `url:"to,omitempty"`    // [optional] you would request pages before this page
	Limit        int64  `url:"limit,omitempty"` // [optional] Number of results per request. Maximum 100. (default 100)
}

// MarginAccountSettingsInfo contains individual currency data
type MarginAccountSettingsInfo struct {
	Available     float64 `json:"available,string"`
	Leverage      float64 `json:"leverage,string"`
	LeverageRatio float64 `json:"leverage_ratio,string"`
	Rate          float64 `json:"rate,string"`
}

// GetMarginLoanHistoryRequest request data for GetMarginLoanHistory
type GetMarginLoanHistoryRequest struct {
	InstrumentID string // [optional] Used when a specific currency response is desired
	Status       int64  `json:"status,string,omitempty"` // [optional] status(0: outstanding 1: repaid)
	From         int64  `json:"from,string,omitempty"`   // [optional] request page from(newer) this id.
	To           int64  `json:"to,string,omitempty"`     // [optional] request page to(older) this id.
	Limit        int64  `json:"limit,string,omitempty"`  // [optional] number of results per request. Maximum 100.(default 100)
}

// GetMarginLoanHistoryResponse response data for GetMarginLoanHistory
type GetMarginLoanHistoryResponse struct {
	Amount           float64   `json:"amount,string"`
	BorrowID         int64     `json:"borrow_id"`
	CreatedAt        string    `json:"created_at"`
	Currency         string    `json:"currency"`
	ForceRepayTime   string    `json:"force_repay_time"`
	InstrumentID     string    `json:"instrument_id"`
	Interest         float64   `json:"interest,string"`
	LastInterestTime string    `json:"last_interest_time"`
	PaidInterest     float64   `json:"paid_interest,string"`
	ProductID        string    `json:"product_id"`
	Rate             float64   `json:"rate,string"`
	RepayAmount      string    `json:"repay_amount"`
	RepayInterest    string    `json:"repay_interest"`
	ReturnedAmount   float64   `json:"returned_amount,string"`
	Timestamp        time.Time `json:"timestamp"`
}

// OpenMarginLoanRequest request data for OpenMarginLoan
type OpenMarginLoanRequest struct {
	QuoteCurrency string  `json:"currency"`      // [required] Second currency eg BTC-USDT: USDT is quote
	InstrumentID  string  `json:"instrument_id"` // [required] Full pair BTC-USDT
	Amount        float64 `json:"amount,string"` // [required] Amount wanting to borrow
}

// OpenMarginLoanResponse response data for OpenMarginLoan
type OpenMarginLoanResponse struct {
	BorrowID int64 `json:"borrow_id"`
	Result   bool  `json:"result"`
}

// RepayMarginLoanRequest request data for RepayMarginLoan
type RepayMarginLoanRequest struct {
	Amount        float64 `json:"amount,string"` // [required] amount repaid
	BorrowID      float64 `json:"borrow_id"`     // [optional] borrow ID . all borrowed token under this trading pair will be repay if the field is left blank
	QuoteCurrency string  `json:"currency"`      // [required] Second currency eg BTC-USDT: USDT is quote
	InstrumentID  string  `json:"instrument_id"` // [required] Full pair BTC-USDT
}

// RepayMarginLoanResponse response data for RepayMarginLoan
type RepayMarginLoanResponse struct {
	RepaymentID int64 `json:"repayment_id"`
	Result      bool  `json:"result"`
}

// GetFuturesPositionsResponse response data for GetFuturesPositions
type GetFuturesPositionsResponse struct {
	Holding [][]GetFuturePostionsDetails `json:"holding"`
	Result  bool                         `json:"result"`
}

// GetFuturesPositionsForCurrencyResponse response data for GetFuturesPositionsForCurrency
type GetFuturesPositionsForCurrencyResponse struct {
	Holding []GetFuturePostionsDetails `json:"holding"`
	Result  bool                       `json:"result"`
}

// GetFuturePostionsDetails Futures details
type GetFuturePostionsDetails struct {
	CreatedAt            string `json:"created_at"`
	InstrumentID         string `json:"instrument_id"`
	Leverage             string `json:"leverage"`
	LiquidationPrice     string `json:"liquidation_price"`
	LongAvailQty         string `json:"long_avail_qty"`
	LongAvgCost          string `json:"long_avg_cost"`
	LongLeverage         string `json:"long_leverage"`
	LongLiquiPrice       string `json:"long_liqui_price"`
	LongMargin           string `json:"long_margin"`
	LongPnlRatio         string `json:"long_pnl_ratio"`
	LongQty              string `json:"long_qty"`
	LongSettlementPrice  string `json:"long_settlement_price"`
	MarginMode           string `json:"margin_mode"`
	RealisedPnl          string `json:"realised_pnl"`
	ShortAvailQty        string `json:"short_avail_qty"`
	ShortAvgCost         string `json:"short_avg_cost"`
	ShortLeverage        string `json:"short_leverage"`
	ShortLiquiPrice      string `json:"short_liqui_price"`
	ShortMargin          string `json:"short_margin"`
	ShortPnlRatio        string `json:"short_pnl_ratio"`
	ShortQty             string `json:"short_qty"`
	ShortSettlementPrice string `json:"short_settlement_price"`
	UpdatedAt            string `json:"updated_at"`
}

// FuturesAccountForAllCurrenciesResponse response data for FuturesAccountForAllCurrencies
type FuturesAccountForAllCurrenciesResponse struct {
	Info struct {
		Currency map[string]FuturesCurrencyData
	} `json:"info"`
}

// FuturesCurrencyData Futures details
type FuturesCurrencyData struct {
	Contracts         []FuturesContractsData `json:"contracts,omitempty"`
	Equity            string                 `json:"equity,omitempty"`
	Margin            string                 `json:"margin,omitempty"`
	MarginMode        string                 `json:"margin_mode,omitempty"`
	MarginRatio       string                 `json:"margin_ratio,omitempty"`
	RealizedPnl       string                 `json:"realized_pnl,omitempty"`
	TotalAvailBalance string                 `json:"total_avail_balance,omitempty"`
	UnrealizedPnl     string                 `json:"unrealized_pnl,omitempty"`
}

// FuturesContractsData Futures details
type FuturesContractsData struct {
	AvailableQty      string `json:"available_qty"`
	FixedBalance      string `json:"fixed_balance"`
	InstrumentID      string `json:"instrument_id"`
	MarginForUnfilled string `json:"margin_for_unfilled"`
	MarginFrozen      string `json:"margin_frozen"`
	RealizedPnl       string `json:"realized_pnl"`
	UnrealizedPnl     string `json:"unrealized_pnl"`
}

// GetFuturesLeverageResponse response data for GetFuturesLeverage
type GetFuturesLeverageResponse struct {
	MarginMode      string `json:"margin_mode,omitempty"`
	Currency        string `json:"currency,omitempty"`
	Leverage        int64  `json:"leverage,omitempty"`
	LeveragePerCoin map[string]GetFuturesLeverageData
}

// GetFuturesLeverageData Futures details
type GetFuturesLeverageData struct {
	LongLeverage  int64 `json:"long_leverage"`
	ShortLeverage int64 `json:"short_leverage"`
}

// SetFuturesLeverageRequest request data for SetFuturesLeverage
type SetFuturesLeverageRequest struct {
	Direction    string `json:"direction,omitempty"`     // opening side (long or short)
	InstrumentID string `json:"instrument_id,omitempty"` //  	Contract ID, e.g. "BTC-USD-180213"
	Leverage     int64  `json:"leverage,omitempty"`      //  	10x or 20x leverage
	Currency     string `json:"currency,omitempty"`
}

// SetFuturesLeverageResponse returned data for SetFuturesLeverage
type SetFuturesLeverageResponse struct {
	Currency                 string `json:"currency"`
	Leverage                 int64  `json:"leverage"`
	MarginMode               string `json:"margin_mode"`
	Result                   string `json:"result"`
	Direction                string `json:"direction"`
	ShortLongDataPerContract map[string]SetFutureLeverageShortLongData
}

// SetFutureLeverageShortLongData long and short data from SetFuturesLeverage
type SetFutureLeverageShortLongData struct {
	Long  int `json:"long"`
	Short int `json:"short"`
}

// PlaceFuturesOrderRequest request data for PlaceFuturesOrder
type PlaceFuturesOrderRequest struct {
	ClientOid    string  `json:"client_oid,omitempty"`         // [optional] 	the order ID customized by yourself
	InstrumentID string  `json:"instrument_id"`                // [required]   	Contract ID,e.g. "TC-USD-180213"
	Type         int64   `json:"type,string"`                  //  [required] 	1:open long 2:open short 3:close long 4:close short
	Price        float64 `json:"price,string"`                 //  [required] 	Price of each contract
	Size         int64   `json:"size,string"`                  //  [required] The buying or selling quantity
	MatchPrice   int64   `json:"match_price,string,omitempty"` // [optional] 	Order at best counter party price? (0:no 1:yes) the parameter is defaulted as 0. If it is set as 1, the price parameter will be ignored
	Leverage     int64   `json:"leverage,string"`              // [required]  	 	10x or 20x leverage
}

// PlaceFuturesOrderResponse response data for PlaceFuturesOrder
type PlaceFuturesOrderResponse struct {
	ClientOid     string `json:"client_oid"`
	ErrorCode     int    `json:"error_code"`
	ErrorMesssage string `json:"error_messsage"`
	OrderID       string `json:"order_id"`
	Result        bool   `json:"result"`
}

// PlaceFuturesOrderBatchRequest request data for PlaceFuturesOrderBatch
type PlaceFuturesOrderBatchRequest struct {
	InstrumentID string                                 `json:"instrument_id"` // [required] Contract ID, e.g."BTC-USD-180213"
	Leverage     int                                    `json:"leverage"`      // [required] 10x or 20x leverage
	OrdersData   []PlaceFuturesOrderBatchRequestDetails `json:"orders_data"`   // [required] the JSON word string for placing multiple orders, include：{client_oid type price size match_price}
}

// PlaceFuturesOrderBatchRequestDetails individual order details for PlaceFuturesOrderBatchRequest
type PlaceFuturesOrderBatchRequestDetails struct {
	ClientOid  string `json:"client_oid"`  // [required] To identify your order with the order ID set by you
	MatchPrice string `json:"match_price"` // undocumented
	Price      string `json:"price"`       // undocumented
	Size       string `json:"size"`        // undocumented
	Type       string `json:"type"`        // undocumented
}

// PlaceFuturesOrderBatchResponse response data from PlaceFuturesOrderBatch
type PlaceFuturesOrderBatchResponse struct {
	OrderInfo []PlaceFuturesOrderBatchResponseData `json:"order_info"`
	Result    bool                                 `json:"result"`
}

// PlaceFuturesOrderBatchResponseData individual order details from PlaceFuturesOrderBatchResponse
type PlaceFuturesOrderBatchResponseData struct {
	ClientOid    string  `json:"client_oid"`
	ErrorCode    int     `json:"error_code"`
	ErrorMessage string  `json:"error_message"`
	OrderID      float64 `json:"order_id"`
}

// CancelFuturesOrderRequest request data for CancelFuturesOrder
type CancelFuturesOrderRequest struct {
	OrderID      string `json:"order_id"`      // [required] Order ID
	InstrumentID string `json:"instrument_id"` // [required] Contract ID,e.g. "BTC-USD-180213"
}

// CancelFuturesOrderResponse response data from CancelFuturesOrder
type CancelFuturesOrderResponse struct {
	InstrumentID string `json:"instrument_id"`
	OrderID      string `json:"order_id"`
	Result       bool   `json:"result"`
}

// GetFuturesOrdersListRequest request data for GetFutureOrdersList
type GetFuturesOrdersListRequest struct {
	InstrumentID string `url:"-"`                      // [required] Contract ID, e.g. "BTC-USD-180213"
	Status       int64  `url:"status,string"`          // [required] Order Status （-1 canceled; 0: pending, 1: partially filled, 2: fully filled, 6: open (pending partially + fully filled), 7: completed (canceled + fully filled))
	From         int64  `url:"from,string,omitempty"`  // [optional] Request paging content for this page number.（Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	To           int64  `url:"to,string,omitempty"`    // [optional] Request page after (older) this pagination id. （Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	Limit        int64  `url:"limit,string,omitempty"` // [optional] Number of results per request. Maximum 100. (default 100)
}

// GetFuturesOrderListResponse response data from GetFuturesOrderList
type GetFuturesOrderListResponse struct {
	OrderInfo []GetFuturesOrderDetailsResponseData `json:"order_info"`
	Result    bool                                 `json:"result"`
}

// GetFuturesOrderDetailsResponseData individual order data from GetFuturesOrderList
type GetFuturesOrderDetailsResponseData struct {
	ContractVal  float64   `json:"contract_val,string"`
	Fee          float64   `json:"fee,string"`
	FilledQty    float64   `json:"filled_qty,string"`
	InstrumentID string    `json:"instrument_id"`
	Leverage     int64     `json:"leverage,string"` //  	Leverage value:10\20 default:10
	OrderID      int64     `json:"order_id,string"`
	Price        float64   `json:"price,string"`
	PriceAvg     float64   `json:"price_avg,string"`
	Size         float64   `json:"size,string"`
	Status       int64     `json:"status,string"` // Order Status （-1 canceled; 0: pending, 1: partially filled, 2: fully filled)
	Timestamp    time.Time `json:"timestamp"`
	Type         int64     `json:"type,string"` //  	Type (1: open long 2: open short 3: close long 4: close short)
}

// GetFuturesOrderDetailsRequest request data for GetFuturesOrderDetails
type GetFuturesOrderDetailsRequest struct {
	OrderID      int64  `json:"order_id,string"` // [required] Order ID
	InstrumentID string `json:"instrument_id"`   // [required] Contract ID, e.g. "BTC-USD-180213"
}

// GetFuturesTransactionDetailsRequest request data for GetFuturesTransactionDetails
type GetFuturesTransactionDetailsRequest struct {
	OrderID      int64  `json:"order_id,string"`        // [required] Order ID
	InstrumentID string `json:"instrument_id"`          // [required] Contract ID, e.g. "BTC-USD-180213"
	Status       int64  `json:"status,string"`          // [required] Order Status （-1 canceled; 0: pending, 1: partially filled, 2: fully filled, 6: open (pending partially + fully filled), 7: completed (canceled + fully filled))
	From         int64  `json:"from,string,omitempty"`  // [optional] Request paging content for this page number.（Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	To           int64  `json:"to,string,omitempty"`    // [optional] Request page after (older) this pagination id. （Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	Limit        int64  `json:"limit,string,omitempty"` // [optional]  	Number of results per request. Maximum 100. (default 100)
}

// GetFuturesTransactionDetailsResponse response data for GetFuturesTransactionDetails
type GetFuturesTransactionDetailsResponse struct {
	CreatedAt    string `json:"created_at"`
	ExecType     string `json:"exec_type"`
	Fee          string `json:"fee"`
	InstrumentID string `json:"instrument_id"`
	OrderID      string `json:"order_id"`
	OrderQty     string `json:"order_qty"`
	Price        string `json:"price"`
	Side         string `json:"side"`
	TradeID      string `json:"trade_id"`
}

// GetFuturesContractInformationResponse individual contract details from  GetFuturesContractInformation
type GetFuturesContractInformationResponse struct {
	ContractValue         float64 `json:"contract_val,string"`
	Alias                 string  `json:"alias"`
	BaseCurrency          string  `json:"base_currency"`
	SettlementCurrency    string  `json:"settlement_currency"`
	ContractValueCurrency string  `json:"contract_val_currency"`
	Delivery              string  `json:"delivery"`
	InstrumentID          string  `json:"instrument_id"`
	Listing               string  `json:"listing"`
	QuoteCurrency         string  `json:"quote_currency"`
	IsInverse             bool    `json:"is_inverse,string"`
	TickSize              float64 `json:"tick_size,string"`
	TradeIncrement        int64   `json:"trade_increment,string"`
	Underlying            string  `json:"underlying"`
	UnderlyingIndex       string  `json:"underlying_index"`
}

// GetFuturesTokenInfoResponse response data for GetFuturesOrderBook
type GetFuturesTokenInfoResponse struct {
	BestAsk      float64   `json:"best_ask,string"`
	BestBid      float64   `json:"best_bid,string"`
	High24h      float64   `json:"high_24h,string"`
	InstrumentID string    `json:"instrument_id"`
	Last         float64   `json:"last,string"`
	Low24h       float64   `json:"low_24h,string"`
	Timestamp    time.Time `json:"timestamp"`
	Volume24h    float64   `json:"volume_24h,string"`
}

// GetFuturesFilledOrderRequest request data for GetFuturesFilledOrder
type GetFuturesFilledOrderRequest struct {
	InstrumentID string `url:"-"`                      // [required] Contract ID, e.g. "BTC-USD-180213"
	From         int64  `url:"from,string,omitempty"`  // [optional] Request paging content for this page number.（Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	To           int64  `url:"to,string,omitempty"`    // [optional] Request page after (older) this pagination id. （Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	Limit        int64  `url:"limit,string,omitempty"` // [optional]  	Number of results per request. Maximum 100. (default 100)
}

// GetFuturesFilledOrdersResponse response data for GetFuturesFilledOrders
type GetFuturesFilledOrdersResponse struct {
	Price     float64   `json:"price,string"`
	Qty       float64   `json:"qty,string"`
	Side      string    `json:"side"`
	Timestamp time.Time `json:"timestamp"`
	TradeID   string    `json:"trade_id"`
}

// GetFuturesMarketDateRequest retrieves candle data information
type GetFuturesMarketDateRequest struct {
	Start        string `url:"start,omitempty"`       // [optional] start time in ISO 8601
	End          string `url:"end,omitempty"`         // [optional] end time in ISO 8601
	Granularity  int64  `url:"granularity,omitempty"` // [optional] The granularity field must be one of the following values: {60, 180, 300, 900, 1800, 3600, 7200, 14400, 43200, 86400, 604800}.
	InstrumentID string `url:"-"`                     // [required] trading pairs
}

// GetFuturesMarketDataResponse contains candle data from a GetMarketDataRequest
// Return Parameters
// time 			string 	Start time
// open 			string 	Open price
// high 			string 	Highest price
// low 				string 	Lowest price
// close 			string 	Close price
// volume 			string 	Trading volume
// currencyvolume 	string 	The trading volume in a specific token
type GetFuturesMarketDataResponse []interface{}

// GetFuturesHoldAmountResponse response data for GetFuturesHoldAmount
type GetFuturesHoldAmountResponse struct {
	Amount       float64   `json:"amount,string"`
	InstrumentID string    `json:"instrument_id"`
	Timestamp    time.Time `json:"timestamp"`
}

// GetFuturesIndicesResponse response data for GetFuturesIndices
type GetFuturesIndicesResponse struct {
	Index        float64   `json:"index,string"`
	InstrumentID string    `json:"instrument_id"`
	Timestamp    time.Time `json:"timestamp"`
}

// GetFuturesExchangeRatesResponse response data for GetFuturesExchangeRate
type GetFuturesExchangeRatesResponse struct {
	InstrumentID string    `json:"instrument_id"`
	Rate         float64   `json:"rate,string"`
	Timestamp    time.Time `json:"timestamp"`
}

// GetFuturesEstimatedDeliveryPriceResponse response data for GetFuturesEstimatedDeliveryPrice
type GetFuturesEstimatedDeliveryPriceResponse struct {
	InstrumentID    string    `json:"instrument_id"`
	SettlementPrice float64   `json:"settlement_price,string"`
	Timestamp       time.Time `json:"timestamp"`
}

// GetFuturesOpenInterestsResponse response data for GetFuturesOpenInterests
type GetFuturesOpenInterestsResponse struct {
	Amount       float64   `json:"amount,string"`
	InstrumentID string    `json:"instrument_id"`
	Timestamp    time.Time `json:"timestamp"`
}

// GetFuturesCurrentPriceLimitResponse response data for GetFuturesCurrentPriceLimit
type GetFuturesCurrentPriceLimitResponse struct {
	Highest      float64   `json:"highest,string"`
	InstrumentID string    `json:"instrument_id"`
	Lowest       float64   `json:"lowest,string"`
	Timestamp    time.Time `json:"timestamp"`
}

// GetFuturesCurrentMarkPriceResponse response data for GetFuturesCurrentMarkPrice
type GetFuturesCurrentMarkPriceResponse struct {
	MarkPrice    float64   `json:"mark_price,string"`
	InstrumentID string    `json:"instrument_id"`
	Timestamp    time.Time `json:"timestamp"`
}

// GetFuturesForceLiquidatedOrdersRequest request data for GetFuturesForceLiquidatedOrders
type GetFuturesForceLiquidatedOrdersRequest struct {
	InstrumentID string `url:"-"`                      // [required] Contract ID, e.g. "BTC-USD-180213"
	From         int64  `url:"from,string,omitempty"`  // [optional] Request paging content for this page number.（Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	To           int64  `url:"to,string,omitempty"`    // [optional] Request page after (older) this pagination id. （Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	Limit        int64  `url:"limit,string,omitempty"` // [optional]  	Number of results per request. Maximum 100. (default 100)
	Status       string `url:"status,omitempty"`       // [optional] Status (0:unfilled orders in the recent 7 days 1:filled orders in the recent 7 days)
}

// GetFuturesForceLiquidatedOrdersResponse response data for GetFuturesForceLiquidatedOrders
type GetFuturesForceLiquidatedOrdersResponse struct {
	Loss         float64 `json:"loss,string"`
	Size         int64   `json:"size,string"`
	Price        float64 `json:"price,string"`
	CreatedAt    string  `json:"created_at"`
	InstrumentID string  `json:"instrument_id"`
	Type         int64   `json:"type,string"`
}

// GetFuturesTagPriceResponse response data for GetFuturesTagPrice
type GetFuturesTagPriceResponse struct {
	MarkPrice    float64   `json:"mark_price"`
	InstrumentID string    `json:"instrument_id"`
	Timestamp    time.Time `json:"timestamp"`
}

// GetSwapPostionsResponse response data for GetSwapPostions
type GetSwapPostionsResponse struct {
	MarginMode string                           `json:"margin_mode"`
	Holding    []GetSwapPostionsResponseHolding `json:"holding"`
}

// GetSwapPostionsResponseHolding response data for GetSwapPostions
type GetSwapPostionsResponseHolding struct {
	AvailPosition    string    `json:"avail_position"`
	AvgCost          string    `json:"avg_cost"`
	InstrumentID     string    `json:"instrument_id"`
	Leverage         string    `json:"leverage"`
	LiquidationPrice string    `json:"liquidation_price"`
	Margin           string    `json:"margin"`
	Position         string    `json:"position"`
	RealizedPnl      string    `json:"realized_pnl"`
	SettlementPrice  string    `json:"settlement_price"`
	Side             string    `json:"side"`
	Timestamp        time.Time `json:"timestamp"`
}

// GetSwapAccountOfAllCurrencyResponse response data for GetSwapAccountOfAllCurrency
type GetSwapAccountOfAllCurrencyResponse struct {
	Info []GetSwapAccountOfAllCurrencyResponseInfo `json:"info"`
}

// GetSwapAccountOfAllCurrencyResponseInfo response data for GetSwapAccountOfAllCurrency
type GetSwapAccountOfAllCurrencyResponseInfo struct {
	Equity            string    `json:"equity"`
	FixedBalance      string    `json:"fixed_balance"`
	TotalAvailBalance string    `json:"total_avail_balance"`
	Margin            string    `json:"margin"`
	RealizedPnl       string    `json:"realized_pnl"`
	UnrealizedPnl     string    `json:"unrealized_pnl"`
	MarginRatio       string    `json:"margin_ratio"`
	InstrumentID      string    `json:"instrument_id"`
	MarginFrozen      string    `json:"margin_frozen"`
	Timestamp         time.Time `json:"timestamp"`
	MarginMode        string    `json:"margin_mode"`
}

// GetSwapAccountSettingsOfAContractResponse response data for GetSwapAccountSettingsOfAContract
type GetSwapAccountSettingsOfAContractResponse struct {
	LongLeverage  float64 `json:"long_leverage,string"`
	MarginMode    string  `json:"margin_mode"`
	ShortLeverage float64 `json:"short_leverage,string"`
	InstrumentID  string  `json:"instrument_id"`
}

// SetSwapLeverageLevelOfAContractRequest request data for SetSwapLeverageLevelOfAContract
type SetSwapLeverageLevelOfAContractRequest struct {
	InstrumentID string `json:"instrument_id,omitempty"` // [required] Contract ID, e.g. BTC-USD-SWAP
	Leverage     int64  `json:"leverage,string"`         // [required] New leverage level from 1-100
	Side         int64  `json:"side,string"`             // [required] Side: 1.FIXED-LONG 2.FIXED-SHORT 3.CROSSED
}

// SetSwapLeverageLevelOfAContractResponse response data for SetSwapLeverageLevelOfAContract
type SetSwapLeverageLevelOfAContractResponse struct {
	InstrumentID  string `json:"instrument_id"`
	LongLeverage  int64  `json:"long_leverage,string"`
	MarginMode    string `json:"margin_mode"`
	ShortLeverage int64  `json:"short_leverage,string"`
}

// GetSwapBillDetailsResponse response data for GetSwapBillDetails
type GetSwapBillDetailsResponse struct {
	LedgerID     string    `json:"ledger_id"`
	Amount       string    `json:"amount"`
	Type         string    `json:"type"`
	Fee          string    `json:"fee"`
	Timestamp    time.Time `json:"timestamp"`
	InstrumentID string    `json:"instrument_id"`
}

// PlaceSwapOrderRequest request data for PlaceSwapOrder
type PlaceSwapOrderRequest struct {
	ClientOID    string  `json:"client_oid,omitempty"`         // [optional] the order ID customized by yourself. 1-32 with digits and letter，The type of client_oid should be comprised of alphabets + numbers or only alphabets within 1 – 32 characters,Both uppercase and lowercase letters are supported
	Size         float64 `json:"size,string"`                  // [required] The buying or selling quantity
	Type         int64   `json:"type,string"`                  // [required] 1:open long 2:open short 3:close long 4:close short
	MatchPrice   int64   `json:"match_price,string,omitempty"` // [optional] Order at best counter party price? (0:no 1:yes)
	Price        float64 `json:"price,string"`                 // [required] Price of each contract
	InstrumentID string  `json:"instrument_id"`                // [required] Contract ID, e.g. BTC-USD-SWAP
}

// PlaceSwapOrderResponse response data for PlaceSwapOrder
type PlaceSwapOrderResponse struct {
	OrderID      string `json:"order_id"`
	ClientOID    int64  `json:"client_oid,string"`
	ErrorCode    int64  `json:"error_code,string"`
	ErrorMessage string `json:"error_message"`
	Result       bool   `json:"result,string"`
}

// PlaceMultipleSwapOrdersRequest response data for PlaceMultipleSwapOrders
type PlaceMultipleSwapOrdersRequest struct {
	InstrumentID string                       `json:"instrument_id"` // [required] Contract ID, e.g. BTC-USD-SWAP
	Leverage     int64                        `json:"leverage"`      // [required] 10x or 20x leverage
	OrdersData   []PlaceMultipleSwapOrderData `json:"orders_data"`   // [required] the JSON word string for placing multiple orders, include：{client_oid type price size match_price}
}

// PlaceMultipleSwapOrderData response data for PlaceMultipleSwapOrders
type PlaceMultipleSwapOrderData struct {
	ClientOID  string `json:"client_oid"`  // [required] To identify your order with the order ID set by you
	Type       string `json:"type"`        // Undocumented
	Price      string `json:"price"`       // Undocumented
	Size       string `json:"size"`        // Undocumented
	MatchPrice string `json:"match_price"` // Undocumented
}

// PlaceMultipleSwapOrdersResponse response data for PlaceMultipleSwapOrders
type PlaceMultipleSwapOrdersResponse struct {
	Result    bool                                  `json:"result,string"`
	OrderInfo []PlaceMultipleSwapOrdersResponseInfo `json:"order_info"`
}

// PlaceMultipleSwapOrdersResponseInfo response data for PlaceMultipleSwapOrders
type PlaceMultipleSwapOrdersResponseInfo struct {
	ErrorMessage string `json:"error_message"`
	ErrorCode    int64  `json:"error_code"`
	ClientOID    string `json:"client_oid"`
	OrderID      string `json:"order_id"`
}

// CancelSwapOrderRequest request data for CancelSwapOrder
type CancelSwapOrderRequest struct {
	OrderID      string `json:"order_id"`      // [required] Order ID
	InstrumentID string `json:"instrument_id"` // [required] Contract ID,e.g. BTC-USD-SWAP
}

// CancelSwapOrderResponse response data for CancelSwapOrder
type CancelSwapOrderResponse struct {
	Result       bool   `json:"result,string"`
	OrderID      string `json:"order_id"`
	InstrumentID string `json:"instrument_id"`
}

// CancelMultipleSwapOrdersRequest request data for CancelMultipleSwapOrders
type CancelMultipleSwapOrdersRequest struct {
	InstrumentID string  `json:"instrument_id,omitempty"` // [required] The contract of the orders to be cancelled
	OrderIDs     []int64 `json:"order_ids"`               // [required] ID's of the orders canceled
}

// CancelMultipleSwapOrdersResponse response data for CancelMultipleSwapOrders
type CancelMultipleSwapOrdersResponse struct {
	Result       bool     `json:"result,string"`
	OrderIDS     []string `json:"order_ids"`
	InstrumentID string   `json:"instrument_id"`
}

// GetSwapOrderListRequest request data for GetSwapOrderList
type GetSwapOrderListRequest struct {
	InstrumentID string `url:"-"`                      // [required] Contract ID, e.g. "BTC-USD-180213"
	Status       int64  `url:"status,string"`          // [required] Order Status （-1 canceled; 0: pending, 1: partially filled, 2: fully filled, 6: open (pending partially + fully filled), 7: completed (canceled + fully filled))
	From         int64  `url:"from,string,omitempty"`  // [optional] Request paging content for this page number.（Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	To           int64  `url:"to,string,omitempty"`    // [optional] Request page after (older) this pagination id. （Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	Limit        int64  `url:"limit,string,omitempty"` // [optional] Number of results per request. Maximum 100. (default 100)
}

// GetSwapOrderListResponse  response data for GetSwapOrderList
type GetSwapOrderListResponse struct {
	Result    bool                           `json:"result,string"`
	OrderInfo []GetSwapOrderListResponseData `json:"order_info"`
}

// GetSwapOrderListResponseData individual order data from GetSwapOrderList
type GetSwapOrderListResponseData struct {
	ContractVal  float64   `json:"contract_val,string"`
	Fee          float64   `json:"fee,string"`
	FilledQty    float64   `json:"filled_qty,string"`
	InstrumentID string    `json:"instrument_id"`
	Leverage     int64     `json:"leverage,string"` //  	Leverage value:10\20 default:10
	OrderID      int64     `json:"order_id,string"`
	Price        float64   `json:"price,string"`
	PriceAvg     float64   `json:"price_avg,string"`
	Size         float64   `json:"size,string"`
	Status       int64     `json:"status,string"` // Order Status （-1 canceled; 0: pending, 1: partially filled, 2: fully filled)
	Timestamp    time.Time `json:"timestamp"`
	Type         int64     `json:"type,string"` //  	Type (1: open long 2: open short 3: close long 4: close short)
}

// GetSwapOrderDetailsRequest request data for GetSwapOrderList
type GetSwapOrderDetailsRequest struct {
	InstrumentID string `json:"instrument_id"` // [required] Contract ID,e.g. BTC-USD-SWAP
	OrderID      string `json:"order_id"`      // [required] Order ID
}

// GetSwapTransactionDetailsRequest request data for GetSwapTransactionDetails
type GetSwapTransactionDetailsRequest struct {
	InstrumentID string `json:"instrument_id"`          // [required] Contract ID, e.g. BTC-USD-SWAP
	OrderID      string `json:"order_id"`               // [required] Order ID
	From         int64  `json:"from,string,omitempty"`  // [optional] Request paging content for this page number.（Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	To           int64  `json:"to,string,omitempty"`    // [optional] Request page after (older) this pagination id. （Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	Limit        int64  `json:"limit,string,omitempty"` // [optional] number of results per request. Maximum 100. (default 100)
}

// GetSwapTransactionDetailsResponse response data for GetSwapTransactionDetails
type GetSwapTransactionDetailsResponse struct {
	TradeID      string    `json:"trade_id"`
	InstrumentID string    `json:"instrument_id"`
	OrderID      string    `json:"order_id"`
	Price        string    `json:"price"`
	OrderQty     string    `json:"order_qty"`
	Fee          string    `json:"fee"`
	Timestamp    time.Time `json:"timestamp"`
	ExecType     string    `json:"exec_type"`
	Side         string    `json:"side"`
}

// GetSwapContractInformationResponse response data for GetSwapContractInformation
type GetSwapContractInformationResponse struct {
	InstrumentID    string  `json:"instrument_id"`
	UnderlyingIndex string  `json:"underlying_index"`
	QuoteCurrency   string  `json:"quote_currency"`
	Coin            string  `json:"coin"`
	ContractVal     float64 `json:"contract_val,string"`
	Listing         string  `json:"listing"`
	Delivery        string  `json:"delivery"`
	SizeIncrement   float64 `json:"size_increment,string"`
	TickSize        float64 `json:"tick_size,string"`
}

// GetSwapOrderBookRequest request data for GetSwapOrderBook
type GetSwapOrderBookRequest struct {
	InstrumentID string  `url:"-"`
	Size         float64 `url:"size,string,omitempty"`
}

// GetSwapOrderBookResponse response data for GetSwapOrderBook
type GetSwapOrderBookResponse struct {
	Asks      [][]interface{} `json:"asks"` // eg [["411.3","16",5,4]] [[0: Price, 1: Size price, 2: number of force liquidated orders, 3: number of orders on the price]]
	Bids      [][]interface{} `json:"bids"` // eg [["411.3","16",5,4]] [[0: Price, 1: Size price, 2: number of force liquidated orders, 3: number of orders on the price]]
	Timestamp time.Time       `json:"timestamp"`
}

// GetAllSwapTokensInformationResponse response data for GetAllSwapTokensInformation
type GetAllSwapTokensInformationResponse struct {
	InstrumentID string    `json:"instrument_id"`
	Last         float64   `json:"last,string"`
	High24H      float64   `json:"high_24h,string"`
	Low24H       float64   `json:"low_24h,string"`
	BestBid      float64   `json:"best_bid,string"`
	BestAsk      float64   `json:"best_ask,string"`
	Volume24H    float64   `json:"volume_24h,string"`
	Timestamp    time.Time `json:"timestamp"`
}

// GetSwapFilledOrdersDataRequest request data for GetSwapFilledOrdersData
type GetSwapFilledOrdersDataRequest struct {
	InstrumentID string `url:"-"`                      // [required] Contract ID, e.g. "BTC-USD-SWAP
	From         int64  `url:"from,string,omitempty"`  // [optional] Request paging content for this page number.（Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	To           int64  `url:"to,string,omitempty"`    // [optional] Request page after (older) this pagination id. （Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	Limit        int64  `url:"limit,string,omitempty"` // [optional] Number of results per request. Maximum 100. (default 100)
}

// GetSwapFilledOrdersDataResponse response data for GetSwapFilledOrdersData
type GetSwapFilledOrdersDataResponse struct {
	TradeID   string    `json:"trade_id"`
	Price     float64   `json:"price,string"`
	Size      float64   `json:"size,string"`
	Side      string    `json:"side"`
	Timestamp time.Time `json:"timestamp"`
}

// GetSwapMarketDataRequest retrieves candle data information
type GetSwapMarketDataRequest struct {
	Start        string `url:"start,omitempty"`       // [optional] start time in ISO 8601
	End          string `url:"end,omitempty"`         // [optional] end time in ISO 8601
	Granularity  int64  `url:"granularity,omitempty"` // The granularity field must be one of the following values: {60, 180, 300, 900, 1800, 3600, 7200, 14400, 43200, 86400, 604800}.
	InstrumentID string `url:"-"`                     // [required] trading pairs
}

// GetSwapMarketDataResponse response data for GetSwapMarketData
// Return Parameters
// time 			string 	Start time
// open 			string 	Open price
// high 			string 	Highest price
// low 				string 	Lowest price
// close 			string 	Close price
// volume 			string 	Trading volume
// currency_volume 	string 	Volume in a specific token
type GetSwapMarketDataResponse []interface{}

// GetSwapIndecesResponse response data for GetSwapIndeces
type GetSwapIndecesResponse struct {
	InstrumentID string    `json:"instrument_id"`
	Index        float64   `json:"index,string"`
	Timestamp    time.Time `json:"timestamp"`
}

// GetSwapExchangeRatesResponse response data for GetSwapExchangeRates
type GetSwapExchangeRatesResponse struct {
	InstrumentID string    `json:"instrument_id"`
	Rate         float64   `json:"rate,string"`
	Timestamp    time.Time `json:"timestamp"`
}

// GetSwapOpenInterestResponse response data for GetSwapOpenInterest
type GetSwapOpenInterestResponse struct {
	InstrumentID string    `json:"instrument_id"`
	Amount       float64   `json:"amount,string"`
	Timestamp    time.Time `json:"timestamp"`
}

// GetSwapCurrentPriceLimitsResponse response data for GetSwapCurrentPriceLimits
type GetSwapCurrentPriceLimitsResponse struct {
	InstrumentID string    `json:"instrument_id"`
	Highest      float64   `json:"highest,string"`
	Lowest       float64   `json:"lowest,string"`
	Timestamp    time.Time `json:"timestamp"`
}

// GetSwapForceLiquidatedOrdersRequest request data for GetSwapForceLiquidatedOrders
type GetSwapForceLiquidatedOrdersRequest struct {
	InstrumentID string `url:"-"`                      // [required] Contract ID, e.g. "BTC-USD-180213"
	From         int64  `url:"from,string,omitempty"`  // [optional] Request paging content for this page number.（Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	To           int64  `url:"to,string,omitempty"`    // [optional] Request page after (older) this pagination id. （Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	Limit        int64  `url:"limit,string,omitempty"` // [optional]  	Number of results per request. Maximum 100. (default 100)
	Status       string `url:"status,omitempty"`       // [optional] Status (0:unfilled orders in the recent 7 days 1:filled orders in the recent 7 days)
}

// GetSwapForceLiquidatedOrdersResponse response data for GetSwapForceLiquidatedOrders
type GetSwapForceLiquidatedOrdersResponse struct {
	Loss         float64 `json:"loss,string"`
	Size         int64   `json:"size,string"`
	Price        float64 `json:"price,string"`
	CreatedAt    string  `json:"created_at"`
	InstrumentID string  `json:"instrument_id"`
	Type         int64   `json:"type,string"`
}

// GetSwapOnHoldAmountForOpenOrdersResponse response data for GetSwapOnHoldAmountForOpenOrders
type GetSwapOnHoldAmountForOpenOrdersResponse struct {
	InstrumentID string    `json:"instrument_id"`
	Amount       float64   `json:"amount,string"`
	Timestamp    time.Time `json:"timestamp"`
}

// GetSwapNextSettlementTimeResponse response data for GetSwapNextSettlementTime
type GetSwapNextSettlementTimeResponse struct {
	InstrumentID string `json:"instrument_id"`
	FundingTime  string `json:"funding_time"`
}

// GetSwapMarkPriceResponse response data for GetSwapMarkPrice
type GetSwapMarkPriceResponse struct {
	InstrumentID string `json:"instrument_id"`
	MarkPrice    string `json:"mark_price"`
	Timstamp     string `json:"timstamp"`
}

// GetSwapFundingRateHistoryRequest request data for GetSwapFundingRateHistory
type GetSwapFundingRateHistoryRequest struct {
	InstrumentID string `url:"ins-trument_id"`         // [required] Contract ID, e.g. "BTC-USD-SWAP
	From         int64  `url:"from,string,omitempty"`  // [optional] Request paging content for this page number.（Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	To           int64  `url:"to,string,omitempty"`    // [optional] Request page after (older) this pagination id. （Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	Limit        int64  `url:"limit,string,omitempty"` // [optional] Number of results per request. Maximum 100.
}

// GetSwapFundingRateHistoryResponse response data for GetSwapFundingRateHistory
type GetSwapFundingRateHistoryResponse struct {
	InstrumentID string  `json:"instrument_id"`
	FundingRate  float64 `json:"funding_rate,string,omitempty"`
	RealizedRate float64 `json:"realized_rate,string"`
	InterestRate float64 `json:"interest_rate,string"`
	FundingTime  string  `json:"funding_time"`
	FundingFee   float64 `json:"funding_fee,string,omitempty"`
}

// GetETTResponse response data for GetETT
type GetETTResponse struct {
	Currency  string  `json:"currency"`
	Balance   float64 `json:"balance"`
	Holds     float64 `json:"holds"`
	Available float64 `json:"available"`
}

// GetETTBillsDetailsResponse response data for GetETTBillsDetails
type GetETTBillsDetailsResponse struct {
	LedgerID  int64   `json:"ledger_id"`
	Currency  string  `json:"currency"`
	Balance   float64 `json:"balance"`
	Amount    float64 `json:"amount"`
	Type      string  `json:"type"`
	CreatedAt string  `json:"created_at"`
	Details   int64   `json:"details"`
}

// PlaceETTOrderRequest  request data for PlaceETTOrder
type PlaceETTOrderRequest struct {
	ClientOID     string  `json:"client_oid"`     // [optional]the order ID customized by yourself
	Type          int64   `json:"type"`           // Type of order (0:ETT subscription 1:subscribe with USDT 2:Redeem in USDT 3:Redeem in underlying)
	QuoteCurrency string  `json:"quote_currency"` // Subscription/redemption currency
	Amount        float64 `json:"amount"`         // Subscription amount. Required for usdt subscription
	Size          string  `json:"size"`           // Redemption size. Required for ETT subscription and redemption
	ETT           string  `json:"ett"`            // ETT name
}

// PlaceETTOrderResponse  response data for PlaceETTOrder
type PlaceETTOrderResponse struct {
	ClientOID string `json:"client_oid"`
	OrderID   string `json:"type"`
	Result    bool   `json:"quote_currency"`
}

// GetETTOrderListRequest request data for GetETTOrderList
type GetETTOrderListRequest struct {
	ETT    string `url:"ett"`                    //  	[required] list specific ETT order
	Type   int64  `url:"type"`                   //  	[required]（1: subscription 2: redemption）
	Status int64  `url:"status,omitempty"`       // [optional]  	List the orders of the status (0:All 1:Unfilled 2:Filled 3:Canceled)
	From   int64  `url:"from,string,omitempty"`  // [optional] Request paging content for this page number.（Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	To     int64  `url:"to,string,omitempty"`    // [optional] Request page after (older) this pagination id. （Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	Limit  int64  `url:"limit,string,omitempty"` // [optional] Number of results per request. Maximum 100.
}

// GetETTOrderListResponse  response data for GetETTOrderList
type GetETTOrderListResponse struct {
	OrderID       string `json:"order_id"`
	Price         string `json:"price"`
	Size          string `json:"size"`
	Amount        string `json:"amount"`
	QuoteCurrency string `json:"quote_currency"`
	Ett           string `json:"ett"`
	Type          int64  `json:"type"`
	CreatedAt     string `json:"created_at"`
	Status        string `json:"status"`
}

// GetETTConstituentsResponse response data for GetETTConstituents
type GetETTConstituentsResponse struct {
	NetValue     float64           `json:"net_value"`
	Ett          string            `json:"ett"`
	Constituents []ConstituentData `json:"constituents"`
}

// ConstituentData response data for GetETTConstituents
type ConstituentData struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// GetETTSettlementPriceHistoryResponse response data for GetETTSettlementPriceHistory
type GetETTSettlementPriceHistoryResponse struct {
	Date  string  `json:"date"`
	Price float64 `json:"price"`
}

// OrderStatus Holds Okcoin order status values
var OrderStatus = map[int64]string{
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

// WebsocketEventRequest contains event data for a websocket channel
type WebsocketEventRequest struct {
	Operation string              `json:"op"`   // 1--subscribe 2--unsubscribe 3--login
	Arguments []map[string]string `json:"args"` // args: the value is the channel name, which can be one or more channels
}

// WebsocketEventResponse contains event data for a websocket channel
type WebsocketEventResponse struct {
	Event   string `json:"event"`
	Channel string `json:"channel,omitempty"`
	Success bool   `json:"success,omitempty"`
}

// WebsocketOrderbookResponse formats orderbook data for a websocket push data
type WebsocketOrderbookResponse struct {
	Arg struct {
		Channel string `json:"channel"`
		InstID  string `json:"instId"`
	} `json:"arg"`
	Action string               `json:"action"`
	Data   []WebsocketOrderBook `json:"data"`
}

// WebsocketOrderBook holds orderbook data
type WebsocketOrderBook struct {
	Asks      [][]string     `json:"asks"` // [ Price, Quantity, depreciated, number of orders at the price ]
	Bids      [][]string     `json:"bids"` // [ Price, Quantity, depreciated, number of orders at the price ]
	Timestamp okcoinMilliSec `json:"ts"`
	Checksum  int64          `json:"checksum"`
}

// WebsocketDataResponse formats all response data for a websocket event
type WebsocketDataResponse struct {
	Arguments struct {
		Channel        string `json:"channel"`
		InstrumentID   string `json:"instId"`
		InstrumentType string `json:"instType"`
	} `json:"arg"`
	Action string        `json:"action"`
	Data   []interface{} `json:"data"`
}

// WebsocketDataResponseReciever formats all response data for a websocket events and used to unmarshal push data into corresponding instance
type WebsocketDataResponseReciever struct {
	Arguments struct {
		Channel        string `json:"channel"`
		InstrumentID   string `json:"instId"`
		InstrumentType string `json:"instType"`
	} `json:"arg"`
	Action string      `json:"action"`
	Data   interface{} `json:"data"`
}

// WebsocketStatus represents a system status response.
type WebsocketStatus struct {
	Arg struct {
		Channel string `json:"channel"`
	} `json:"arg"`
	Data []struct {
		Title                 string         `json:"title"`
		State                 okcoinMilliSec `json:"state"`
		Begin                 okcoinMilliSec `json:"begin"`
		Href                  string         `json:"href"`
		End                   string         `json:"end"`
		ServiceType           string         `json:"serviceType"`
		System                string         `json:"system"`
		RescheduleDescription string         `json:"scheDesc"`
		Time                  okcoinMilliSec `json:"ts"`
	} `json:"data"`
}

// WebsocketAccount represents an account information. Data will be pushed when triggered by events such as placing order, canceling order, transaction execution
type WebsocketAccount struct {
	Arg struct {
		Channel string `json:"channel"`
		UID     string `json:"uid"`
	} `json:"arg"`
	Data []struct {
		AdjustedEquity string `json:"adjEq"`
		Details        []struct {
			AvailableBalance   string         `json:"availBal"`
			AvailableEquity    string         `json:"availEq"`
			CashBalance        string         `json:"cashBal"`
			Currency           string         `json:"ccy"`
			CoinUsdPrice       string         `json:"coinUsdPrice"`
			CrossLiab          string         `json:"crossLiab"`
			DiscountEquity     string         `json:"disEq"`
			Equity             string         `json:"eq"`
			EquityUsd          string         `json:"eqUsd"`
			FixedBalance       string         `json:"fixedBal"`
			FrozenBalance      string         `json:"frozenBal"`
			Interest           string         `json:"interest"`
			IsoEquity          string         `json:"isoEq"`
			IsoLiability       string         `json:"isoLiab"`
			IsoUpl             string         `json:"isoUpl"`
			Liability          string         `json:"liab"`
			MaxLoan            string         `json:"maxLoan"`
			MgnRatio           string         `json:"mgnRatio"`
			NotionalLeverage   string         `json:"notionalLever"`
			MarginFrozenOrders string         `json:"ordFrozen"`
			SpotInUseAmount    string         `json:"spotInUseAmt"`
			StrategyEquity     string         `json:"stgyEq"`
			Twap               string         `json:"twap"`
			UPL                string         `json:"upl"` // Unrealized profit and loss
			UpdateTime         okcoinMilliSec `json:"uTime"`
		} `json:"details"`
		FrozenEquity                 string         `json:"imr"`
		IsoEquity                    string         `json:"isoEq"`
		MarginRatio                  string         `json:"mgnRatio"`
		MaintenanceMarginRequirement string         `json:"mmr"`
		NotionalUsd                  string         `json:"notionalUsd"`
		MarginOrderFrozen            string         `json:"ordFroz"`
		TotalEquity                  string         `json:"totalEq"`
		UpdateTime                   okcoinMilliSec `json:"uTime"`
	} `json:"data"`
}

// WebsocketOrder represents and order information. Data will not be pushed when first subscribed.
type WebsocketOrder struct {
	Arg struct {
		Channel        string `json:"channel"`
		InstrumentType string `json:"instType"`
		InstID         string `json:"instId"`
		UID            string `json:"uid"`
	} `json:"arg"`
	Data []struct {
		AccFillSize                float64        `json:"accFillSz"`
		AmendResult                string         `json:"amendResult"`
		AveragePrice               float64        `json:"avgPx"`
		CreateTime                 okcoinMilliSec `json:"cTime"`
		Category                   string         `json:"category"`
		Currency                   string         `json:"ccy"`
		ClientOrdID                string         `json:"clOrdId"`
		Code                       string         `json:"code"`
		ExecType                   string         `json:"execType"`
		Fee                        float64        `json:"fee"`
		FeeCurrency                string         `json:"feeCcy"`
		FillFee                    string         `json:"fillFee"`
		FillFeeCurrency            string         `json:"fillFeeCcy"`
		FillNotionalUsd            string         `json:"fillNotionalUsd"`
		FillPrice                  float64        `json:"fillPx"`
		FillSize                   float64        `json:"fillSz"`
		FillTime                   okcoinMilliSec `json:"fillTime"`
		InstrumentID               string         `json:"instId"`
		InstrumentType             string         `json:"instType"`
		Leverage                   string         `json:"lever"`
		ErrorMessage               string         `json:"msg"`
		NotionalUsd                string         `json:"notionalUsd"`
		OrderID                    string         `json:"ordId"`
		OrderType                  string         `json:"ordType"`
		ProfitAndLoss              string         `json:"pnl"`
		PositionSide               string         `json:"posSide"`
		Price                      string         `json:"px"`
		Rebate                     string         `json:"rebate"`
		RebateCurrency             string         `json:"rebateCcy"`
		ReduceOnly                 string         `json:"reduceOnly"`
		ClientRequestID            string         `json:"reqId"`
		Side                       string         `json:"side"`
		StopLossOrderPrice         float64        `json:"slOrdPx"`
		StopLossTriggerPrice       float64        `json:"slTriggerPx"`
		StopLossTriggerPriceType   string         `json:"slTriggerPxType"`
		Source                     string         `json:"source"`
		State                      string         `json:"state"`
		Size                       float64        `json:"sz"`
		Tag                        string         `json:"tag"`
		TradeMode                  string         `json:"tdMode"`
		TargetCurrency             string         `json:"tgtCcy"`
		TakeProfitOrdPrice         float64        `json:"tpOrdPx"`
		TakeProfitTriggerPrice     float64        `json:"tpTriggerPx"`
		TakeProfitTriggerPriceType string         `json:"tpTriggerPxType"`
		TradeID                    string         `json:"tradeId"`
		UTime                      okcoinMilliSec `json:"uTime"`
	} `json:"data"`
}

// WebsocketAlgoOrder represents algo orders (includes trigger order, oco order, conditional order).
type WebsocketAlgoOrder struct {
	Arg struct {
		Channel  string `json:"channel"`
		UID      string `json:"uid"`
		InstType string `json:"instType"`
		InstID   string `json:"instId"`
	} `json:"arg"`
	Data []struct {
		InstrumentType             string         `json:"instType"`
		InstrumentID               string         `json:"instId"`
		OrderID                    string         `json:"ordId"`
		Currency                   string         `json:"ccy"`
		ClientOrderID              string         `json:"clOrdId"`
		AlgoID                     string         `json:"algoId"`
		Price                      float64        `json:"px,string"`
		Size                       float64        `json:"sz,string"`
		TradeMode                  string         `json:"tdMode"`
		TgtCurrency                string         `json:"tgtCcy"`
		NotionalUsd                string         `json:"notionalUsd"`
		OrderType                  string         `json:"ordType"`
		Side                       string         `json:"side"`
		PositionSide               string         `json:"posSide"`
		State                      string         `json:"state"`
		Leverage                   float64        `json:"lever"`
		TakeProfitTriggerPrice     float64        `json:"tpTriggerPx,string"`
		TakeProfitTriggerPriceType string         `json:"tpTriggerPxType"`
		TakeProfitOrdPrice         float64        `json:"tpOrdPx,string"`
		SlTriggerPrice             float64        `json:"slTriggerPx,string"`
		SlTriggerPriceType         string         `json:"slTriggerPxType"`
		TriggerPxType              string         `json:"triggerPxType"`
		TriggerPrice               float64        `json:"triggerPx,string"`
		OrderPrice                 float64        `json:"ordPx,string"`
		Tag                        string         `json:"tag"`
		ActualSize                 float64        `json:"actualSz,string"`
		ActualPrice                float64        `json:"actualPx,string"`
		ActualSide                 string         `json:"actualSide"`
		TriggerTime                okcoinMilliSec `json:"triggerTime"`
		CreateTime                 okcoinMilliSec `json:"cTime"`
	} `json:"data"`
}

// WebsocketAdvancedAlgoOrder represents advance algo orders (including Iceberg order, TWAP order, Trailing order).
type WebsocketAdvancedAlgoOrder struct {
	Arg struct {
		Channel  string `json:"channel"`
		UID      string `json:"uid"`
		InstType string `json:"instType"`
		InstID   string `json:"instId"`
	} `json:"arg"`
	Data []struct {
		ActualPx       string `json:"actualPx"`
		ActualSide     string `json:"actualSide"`
		ActualSz       string `json:"actualSz"`
		AlgoID         string `json:"algoId"`
		CTime          string `json:"cTime"`
		Ccy            string `json:"ccy"`
		ClOrdID        string `json:"clOrdId"`
		Count          string `json:"count"`
		InstID         string `json:"instId"`
		InstType       string `json:"instType"`
		Lever          string `json:"lever"`
		NotionalUsd    string `json:"notionalUsd"`
		OrdPx          string `json:"ordPx"`
		OrdType        string `json:"ordType"`
		PTime          string `json:"pTime"`
		PosSide        string `json:"posSide"`
		PxLimit        string `json:"pxLimit"`
		PxSpread       string `json:"pxSpread"`
		PxVar          string `json:"pxVar"`
		Side           string `json:"side"`
		SlOrdPx        string `json:"slOrdPx"`
		SlTriggerPx    string `json:"slTriggerPx"`
		State          string `json:"state"`
		Sz             string `json:"sz"`
		SzLimit        string `json:"szLimit"`
		TdMode         string `json:"tdMode"`
		TimeInterval   string `json:"timeInterval"`
		TpOrdPx        string `json:"tpOrdPx"`
		TpTriggerPx    string `json:"tpTriggerPx"`
		Tag            string `json:"tag"`
		TriggerPx      string `json:"triggerPx"`
		TriggerTime    string `json:"triggerTime"`
		CallbackRatio  string `json:"callbackRatio"`
		CallbackSpread string `json:"callbackSpread"`
		ActivePx       string `json:"activePx"`
		MoveTriggerPx  string `json:"moveTriggerPx"`
	} `json:"data"`
}

// WebsocketInstrumentData contains formatted data for instruments related websocket responses
type WebsocketInstrumentData struct {
	Alias                 string         `json:"alias"`
	BaseCurrency          string         `json:"baseCcy"`
	Category              string         `json:"category"`
	ContractMultiplier    string         `json:"ctMult"`
	ContractType          string         `json:"ctType"`
	ContractValue         string         `json:"ctVal"`
	ContractValueCurrency string         `json:"ctValCcy"`
	ExpiryTime            okcoinMilliSec `json:"expTime"`
	InstrumentFamily      string         `json:"instFamily"`
	InstrumentID          string         `json:"instId"`
	InstrumentType        string         `json:"instType"`
	Leverage              string         `json:"lever"`
	ListTime              okcoinMilliSec `json:"listTime"`
	LotSize               string         `json:"lotSz"`
	MaxIcebergSize        float64        `json:"maxIcebergSz,string"`
	MaxLimitSize          float64        `json:"maxLmtSz,string"`
	MaxMarketSize         float64        `json:"maxMktSz,string"`
	MaxStopSize           float64        `json:"maxStopSz,string"`
	MaxTriggerSize        float64        `json:"maxTriggerSz,string"`
	MaxTwapSize           float64        `json:"maxTwapSz,string"`
	MinimumOrderSize      float64        `json:"minSz,string"`
	OptionType            string         `json:"optType"`
	QuoteCurrency         string         `json:"quoteCcy"`
	SettleCurrency        string         `json:"settleCcy"`
	State                 string         `json:"state"`
	StrikePrice           string         `json:"stk"`
	TickSize              float64        `json:"tickSz,string"`
	Underlying            string         `json:"uly"`
}

// WsTickerData contains formatted data for ticker related websocket responses
type WsTickerData struct {
	InstrumentType string         `json:"instType"`
	InstrumentID   string         `json:"instId"`
	Last           float64        `json:"last,string"`
	LastSize       float64        `json:"lastSz,string"`
	AskPrice       float64        `json:"askPx,string"`
	AskSize        float64        `json:"askSz,string"`
	BidPrice       float64        `json:"bidPx,string"`
	BidSize        float64        `json:"bidSz,string"`
	Open24H        float64        `json:"open24h,string"`
	High24H        float64        `json:"high24h,string"`
	Low24H         float64        `json:"low24h,string"`
	SodUtc0        string         `json:"sodUtc0"`
	SodUtc8        string         `json:"sodUtc8"`
	VolCcy24H      float64        `json:"volCcy24h,string"`
	Vol24H         float64        `json:"vol24h,string"`
	Timestamp      okcoinMilliSec `json:"ts"`
}

// WebsocketTradeResponse contains formatted data for trade related websocket responses
type WebsocketTradeResponse struct {
	Arg struct {
		Channel      string `json:"channel"`
		InstrumentID string `json:"instId"`
	} `json:"arg"`
	Data []struct {
		InstrumentID string         `json:"instId"`
		TradeID      string         `json:"tradeId"`
		Price        float64        `json:"px,string"`
		Size         float64        `json:"sz,string"`
		Side         string         `json:"side"`
		Timestamp    okcoinMilliSec `json:"ts"`
	} `json:"data"`
}

// WebsocketCandlesResponse represents a candlestick response data.
type WebsocketCandlesResponse struct {
	Arg struct {
		Channel string `json:"channel"`
		InstID  string `json:"instId"`
	} `json:"arg"`
	Data [][]string `json:"data"`
}

// GetCandlesData represents a candlestick instances list.
func (a *WebsocketCandlesResponse) GetCandlesData(exchangeName string) ([]stream.KlineData, error) {
	candlesticks := make([]stream.KlineData, len(a.Data))
	cp, err := currency.NewPairFromString(a.Arg.InstID)
	if err != nil {
		return nil, err
	}
	for x := range a.Data {
		var timestamp int64
		timestamp, err = strconv.ParseInt(a.Data[x][0], 10, 64)
		if err != nil {
			return nil, err
		}
		candlesticks[x] = stream.KlineData{
			AssetType: asset.Spot,
			Pair:      cp,
			Timestamp: time.UnixMilli(timestamp),
			Exchange:  exchangeName,
		}
		candlesticks[x].OpenPrice, err = strconv.ParseFloat(a.Data[x][1], 64)
		if err != nil {
			return nil, err
		}
		candlesticks[x].HighPrice, err = strconv.ParseFloat(a.Data[x][2], 64)
		if err != nil {
			return nil, err
		}
		candlesticks[x].LowPrice, err = strconv.ParseFloat(a.Data[x][3], 64)
		if err != nil {
			return nil, err
		}
		candlesticks[x].ClosePrice, err = strconv.ParseFloat(a.Data[x][4], 64)
		if err != nil {
			return nil, err
		}
		candlesticks[x].Volume, err = strconv.ParseFloat(a.Data[x][5], 64)
		if err != nil {
			return nil, err
		}
	}
	return candlesticks, nil
}

// WebsocketOrderBooksData is the full websocket response containing orderbook data
type WebsocketOrderBooksData struct {
	Table  string               `json:"table"`
	Action string               `json:"action"`
	Data   []WebsocketOrderBook `json:"data"`
}

// WebsocketUserSwapPositionResponse contains formatted data for user position data
type WebsocketUserSwapPositionResponse struct {
	InstrumentID string                                 `json:"instrument_id"`
	Timestamp    time.Time                              `json:"timestamp,omitempty"`
	Holding      []WebsocketUserSwapPositionHoldingData `json:"holding,omitempty"`
}

// WebsocketUserSwapPositionHoldingData contains formatted data for user position holding data
type WebsocketUserSwapPositionHoldingData struct {
	AvailablePosition float64   `json:"avail_position,string,omitempty"`
	AverageCost       float64   `json:"avg_cost,string,omitempty"`
	Leverage          float64   `json:"leverage,string,omitempty"`
	LiquidationPrice  float64   `json:"liquidation_price,string,omitempty"`
	Margin            float64   `json:"margin,string,omitempty"`
	Position          float64   `json:"position,string,omitempty"`
	RealizedPnl       float64   `json:"realized_pnl,string,omitempty"`
	SettlementPrice   float64   `json:"settlement_price,string,omitempty"`
	Side              string    `json:"side,omitempty"`
	Timestamp         time.Time `json:"timestamp,omitempty"`
}

// WebsocketSpotOrderResponse contains formatted data for spot user orders
type WebsocketSpotOrderResponse struct {
	Table string `json:"table"`
	Data  []struct {
		ClientOid      string    `json:"client_oid"`
		CreatedAt      time.Time `json:"created_at"`
		FilledNotional float64   `json:"filled_notional,string"`
		FilledSize     float64   `json:"filled_size,string"`
		InstrumentID   string    `json:"instrument_id"`
		LastFillPx     float64   `json:"last_fill_px,string"`
		LastFillQty    float64   `json:"last_fill_qty,string"`
		LastFillTime   time.Time `json:"last_fill_time"`
		MarginTrading  int64     `json:"margin_trading,string"`
		Notional       string    `json:"notional"`
		OrderID        string    `json:"order_id"`
		OrderType      int64     `json:"order_type,string"`
		Price          float64   `json:"price,string"`
		Side           string    `json:"side"`
		Size           float64   `json:"size,string"`
		State          int64     `json:"state,string"`
		Status         string    `json:"status"`
		Timestamp      time.Time `json:"timestamp"`
		Type           string    `json:"type"`
	} `json:"data"`
}

// WebsocketErrorResponse yo
type WebsocketErrorResponse struct {
	Event     string `json:"event"`
	Message   string `json:"message"`
	ErrorCode int64  `json:"errorCode"`
}

// List of all websocket channels to subscribe to
const (
	// Orderbook events
	okcoinWsOrderbookUpdate  = "update"
	okcoinWsOrderbookPartial = "partial"
	// API subsections
	okcoinWsSwapSubsection    = "swap/"
	okcoinWsIndexSubsection   = "index/"
	okcoinWsFuturesSubsection = "futures/"
	okcoinWsSpotSubsection    = "spot/"
	// Shared API endpoints
	okcoinWsCandle         = "candle"
	okcoinWsCandle60s      = okcoinWsCandle + "60s"
	okcoinWsCandle180s     = okcoinWsCandle + "180s"
	okcoinWsCandle300s     = okcoinWsCandle + "300s"
	okcoinWsCandle900s     = okcoinWsCandle + "900s"
	okcoinWsCandle1800s    = okcoinWsCandle + "1800s"
	okcoinWsCandle3600s    = okcoinWsCandle + "3600s"
	okcoinWsCandle7200s    = okcoinWsCandle + "7200s"
	okcoinWsCandle14400s   = okcoinWsCandle + "14400s"
	okcoinWsCandle21600s   = okcoinWsCandle + "21600"
	okcoinWsCandle43200s   = okcoinWsCandle + "43200s"
	okcoinWsCandle86400s   = okcoinWsCandle + "86400s"
	okcoinWsCandle604900s  = okcoinWsCandle + "604800s"
	okcoinWsTicker         = "ticker"
	okcoinWsTrade          = "trade"
	okcoinWsDepth          = "depth"
	okcoinWsDepth5         = "depth5"
	okcoinWsAccount        = "account"
	okcoinWsMarginAccount  = "margin_account"
	okcoinWsOrder          = "order"
	okcoinWsFundingRate    = "funding_rate"
	okcoinWsPriceRange     = "price_range"
	okcoinWsMarkPrice      = "mark_price"
	okcoinWsPosition       = "position"
	okcoinWsEstimatedPrice = "estimated_price"
	// Spot endpoints
	okcoinWsSpotTicker        = okcoinWsSpotSubsection + okcoinWsTicker
	okcoinWsSpotCandle60s     = okcoinWsSpotSubsection + okcoinWsCandle60s
	okcoinWsSpotCandle180s    = okcoinWsSpotSubsection + okcoinWsCandle180s
	okcoinWsSpotCandle300s    = okcoinWsSpotSubsection + okcoinWsCandle300s
	okcoinWsSpotCandle900s    = okcoinWsSpotSubsection + okcoinWsCandle900s
	okcoinWsSpotCandle1800s   = okcoinWsSpotSubsection + okcoinWsCandle1800s
	okcoinWsSpotCandle3600s   = okcoinWsSpotSubsection + okcoinWsCandle3600s
	okcoinWsSpotCandle7200s   = okcoinWsSpotSubsection + okcoinWsCandle7200s
	okcoinWsSpotCandle14400s  = okcoinWsSpotSubsection + okcoinWsCandle14400s
	okcoinWsSpotCandle21600s  = okcoinWsSpotSubsection + okcoinWsCandle21600s
	okcoinWsSpotCandle43200s  = okcoinWsSpotSubsection + okcoinWsCandle43200s
	okcoinWsSpotCandle86400s  = okcoinWsSpotSubsection + okcoinWsCandle86400s
	okcoinWsSpotCandle604900s = okcoinWsSpotSubsection + okcoinWsCandle604900s
	okcoinWsSpotTrade         = okcoinWsSpotSubsection + okcoinWsTrade
	okcoinWsSpotDepth         = okcoinWsSpotSubsection + okcoinWsDepth
	okcoinWsSpotDepth5        = okcoinWsSpotSubsection + okcoinWsDepth5
	okcoinWsSpotAccount       = okcoinWsSpotSubsection + okcoinWsAccount
	okcoinWsSpotMarginAccount = okcoinWsSpotSubsection + okcoinWsMarginAccount
	okcoinWsSpotOrder         = okcoinWsSpotSubsection + okcoinWsOrder
	// Swap endpoints
	okcoinWsSwapTicker        = okcoinWsSwapSubsection + okcoinWsTicker
	okcoinWsSwapCandle60s     = okcoinWsSwapSubsection + okcoinWsCandle60s
	okcoinWsSwapCandle180s    = okcoinWsSwapSubsection + okcoinWsCandle180s
	okcoinWsSwapCandle300s    = okcoinWsSwapSubsection + okcoinWsCandle300s
	okcoinWsSwapCandle900s    = okcoinWsSwapSubsection + okcoinWsCandle900s
	okcoinWsSwapCandle1800s   = okcoinWsSwapSubsection + okcoinWsCandle1800s
	okcoinWsSwapCandle3600s   = okcoinWsSwapSubsection + okcoinWsCandle3600s
	okcoinWsSwapCandle7200s   = okcoinWsSwapSubsection + okcoinWsCandle7200s
	okcoinWsSwapCandle14400s  = okcoinWsSwapSubsection + okcoinWsCandle14400s
	okcoinWsSwapCandle21600s  = okcoinWsSwapSubsection + okcoinWsCandle21600s
	okcoinWsSwapCandle43200s  = okcoinWsSwapSubsection + okcoinWsCandle43200s
	okcoinWsSwapCandle86400s  = okcoinWsSwapSubsection + okcoinWsCandle86400s
	okcoinWsSwapCandle604900s = okcoinWsSwapSubsection + okcoinWsCandle604900s
	okcoinWsSwapTrade         = okcoinWsSwapSubsection + okcoinWsTrade
	okcoinWsSwapDepth         = okcoinWsSwapSubsection + okcoinWsDepth
	okcoinWsSwapDepth5        = okcoinWsSwapSubsection + okcoinWsDepth5
	okcoinWsSwapFundingRate   = okcoinWsSwapSubsection + okcoinWsFundingRate
	okcoinWsSwapPriceRange    = okcoinWsSwapSubsection + okcoinWsPriceRange
	okcoinWsSwapMarkPrice     = okcoinWsSwapSubsection + okcoinWsMarkPrice
	okcoinWsSwapPosition      = okcoinWsSwapSubsection + okcoinWsPosition
	okcoinWsSwapAccount       = okcoinWsSwapSubsection + okcoinWsAccount
	okcoinWsSwapOrder         = okcoinWsSwapSubsection + okcoinWsOrder
	// Index endpoints
	okcoinWsIndexTicker        = okcoinWsIndexSubsection + okcoinWsTicker
	okcoinWsIndexCandle60s     = okcoinWsIndexSubsection + okcoinWsCandle60s
	okcoinWsIndexCandle180s    = okcoinWsIndexSubsection + okcoinWsCandle180s
	okcoinWsIndexCandle300s    = okcoinWsIndexSubsection + okcoinWsCandle300s
	okcoinWsIndexCandle900s    = okcoinWsIndexSubsection + okcoinWsCandle900s
	okcoinWsIndexCandle1800s   = okcoinWsIndexSubsection + okcoinWsCandle1800s
	okcoinWsIndexCandle3600s   = okcoinWsIndexSubsection + okcoinWsCandle3600s
	okcoinWsIndexCandle7200s   = okcoinWsIndexSubsection + okcoinWsCandle7200s
	okcoinWsIndexCandle14400s  = okcoinWsIndexSubsection + okcoinWsCandle14400s
	okcoinWsIndexCandle21600s  = okcoinWsIndexSubsection + okcoinWsCandle21600s
	okcoinWsIndexCandle43200s  = okcoinWsIndexSubsection + okcoinWsCandle43200s
	okcoinWsIndexCandle86400s  = okcoinWsIndexSubsection + okcoinWsCandle86400s
	okcoinWsIndexCandle604900s = okcoinWsIndexSubsection + okcoinWsCandle604900s
	// Futures endpoints
	okcoinWsFuturesTicker         = okcoinWsFuturesSubsection + okcoinWsTicker
	okcoinWsFuturesCandle60s      = okcoinWsFuturesSubsection + okcoinWsCandle60s
	okcoinWsFuturesCandle180s     = okcoinWsFuturesSubsection + okcoinWsCandle180s
	okcoinWsFuturesCandle300s     = okcoinWsFuturesSubsection + okcoinWsCandle300s
	okcoinWsFuturesCandle900s     = okcoinWsFuturesSubsection + okcoinWsCandle900s
	okcoinWsFuturesCandle1800s    = okcoinWsFuturesSubsection + okcoinWsCandle1800s
	okcoinWsFuturesCandle3600s    = okcoinWsFuturesSubsection + okcoinWsCandle3600s
	okcoinWsFuturesCandle7200s    = okcoinWsFuturesSubsection + okcoinWsCandle7200s
	okcoinWsFuturesCandle14400s   = okcoinWsFuturesSubsection + okcoinWsCandle14400s
	okcoinWsFuturesCandle21600s   = okcoinWsFuturesSubsection + okcoinWsCandle21600s
	okcoinWsFuturesCandle43200s   = okcoinWsFuturesSubsection + okcoinWsCandle43200s
	okcoinWsFuturesCandle86400s   = okcoinWsFuturesSubsection + okcoinWsCandle86400s
	okcoinWsFuturesCandle604900s  = okcoinWsFuturesSubsection + okcoinWsCandle604900s
	okcoinWsFuturesTrade          = okcoinWsFuturesSubsection + okcoinWsTrade
	okcoinWsFuturesEstimatedPrice = okcoinWsFuturesSubsection + okcoinWsTrade
	okcoinWsFuturesPriceRange     = okcoinWsFuturesSubsection + okcoinWsPriceRange
	okcoinWsFuturesDepth          = okcoinWsFuturesSubsection + okcoinWsDepth
	okcoinWsFuturesDepth5         = okcoinWsFuturesSubsection + okcoinWsDepth5
	okcoinWsFuturesMarkPrice      = okcoinWsFuturesSubsection + okcoinWsMarkPrice
	okcoinWsFuturesAccount        = okcoinWsFuturesSubsection + okcoinWsAccount
	okcoinWsFuturesPosition       = okcoinWsFuturesSubsection + okcoinWsPosition
	okcoinWsFuturesOrder          = okcoinWsFuturesSubsection + okcoinWsOrder

	okcoinWsRateLimit = 30

	allowableIterations = 25
	delimiterColon      = ":"
	delimiterDash       = "-"

	maxConnByteLen = 4096
)

// orderbookMutex Ensures if two entries arrive at once, only one can be
// processed at a time
var orderbookMutex sync.Mutex

var defaultSpotSubscribedChannels = []string{okcoinWsSpotDepth,
	okcoinWsSpotCandle300s,
	okcoinWsSpotTicker,
	okcoinWsSpotTrade}

var defaultFuturesSubscribedChannels = []string{okcoinWsFuturesDepth,
	okcoinWsFuturesCandle300s,
	okcoinWsFuturesTicker,
	okcoinWsFuturesTrade}

var defaultIndexSubscribedChannels = []string{okcoinWsIndexCandle300s,
	okcoinWsIndexTicker}

var defaultSwapSubscribedChannels = []string{okcoinWsSwapDepth,
	okcoinWsSwapCandle300s,
	okcoinWsSwapTicker,
	okcoinWsSwapTrade,
	okcoinWsSwapFundingRate,
	okcoinWsSwapMarkPrice}

// SetErrorDefaults sets the full error default list
func (o *OKCoin) SetErrorDefaults() {
	o.ErrorCodes = map[string]error{
		"0":     errors.New("successful"),
		"1":     errors.New("invalid parameter in url normally"),
		"30001": errors.New("request header \"OK_ACCESS_KEY\" cannot be blank"),
		"30002": errors.New("request header \"OK_ACCESS_SIGN\" cannot be blank"),
		"30003": errors.New("request header \"OK_ACCESS_TIMESTAMP\" cannot be blank"),
		"30004": errors.New("request header \"OK_ACCESS_PASSPHRASE\" cannot be blank"),
		"30005": errors.New("invalid OK_ACCESS_TIMESTAMP"),
		"30006": errors.New("invalid OK_ACCESS_KEY"),
		"30007": errors.New("invalid Content_Type, please use \"application/json\" format"),
		"30008": errors.New("timestamp request expired"),
		"30009": errors.New("system error"),
		"30010": errors.New("api validation failed"),
		"30011": errors.New("invalid IP"),
		"30012": errors.New("invalid authorization"),
		"30013": errors.New("invalid sign"),
		"30014": errors.New("request too frequent"),
		"30015": errors.New("request header \"OK_ACCESS_PASSPHRASE\" incorrect"),
		"30016": errors.New("you are using v1 apiKey, please use v1 endpoint. If you would like to use v3 endpoint, please subscribe to v3 apiKey"),
		"30017": errors.New("apikey's broker id does not match"),
		"30018": errors.New("apikey's domain does not match"),
		"30020": errors.New("body cannot be blank"),
		"30021": errors.New("json data format error"),
		"30023": errors.New("required parameter cannot be blank"),
		"30024": errors.New("parameter value error"),
		"30025": errors.New("parameter category error"),
		"30026": errors.New("requested too frequent; endpoint limit exceeded"),
		"30027": errors.New("login failure"),
		"30028": errors.New("unauthorized execution"),
		"30029": errors.New("account suspended"),
		"30030": errors.New("endpoint request failed. Please try again"),
		"30031": errors.New("token does not exist"),
		"30032": errors.New("pair does not exist"),
		"30033": errors.New("exchange domain does not exist"),
		"30034": errors.New("exchange ID does not exist"),
		"30035": errors.New("trading is not supported in this website"),
		"30036": errors.New("no relevant data"),
		"30037": errors.New("endpoint is offline or unavailable"),
		"30038": errors.New("user does not exist"),
		"32001": errors.New("futures account suspended"),
		"32002": errors.New("futures account does not exist"),
		"32003": errors.New("canceling, please wait"),
		"32004": errors.New("you have no unfilled orders"),
		"32005": errors.New("max order quantity"),
		"32006": errors.New("the order price or trigger price exceeds USD 1 million"),
		"32007": errors.New("leverage level must be the same for orders on the same side of the contract"),
		"32008": errors.New("max. positions to open (cross margin)"),
		"32009": errors.New("max. positions to open (fixed margin)"),
		"32010": errors.New("leverage cannot be changed with open positions"),
		"32011": errors.New("futures status error"),
		"32012": errors.New("futures order update error"),
		"32013": errors.New("token type is blank"),
		"32014": errors.New("your number of contracts closing is larger than the number of contracts available"),
		"32015": errors.New("margin ratio is lower than 100% before opening positions"),
		"32016": errors.New("margin ratio is lower than 100% after opening position"),
		"32017": errors.New("no BBO"),
		"32018": errors.New("the order quantity is less than 1, please try again"),
		"32019": errors.New("the order price deviates from the price of the previous minute by more than 3%"),
		"32020": errors.New("the price is not in the range of the price limit"),
		"32021": errors.New("leverage error"),
		"32022": errors.New("this function is not supported in your country or region according to the regulations"),
		"32023": errors.New("this account has outstanding loan"),
		"32024": errors.New("order cannot be placed during delivery"),
		"32025": errors.New("order cannot be placed during settlement"),
		"32026": errors.New("your account is restricted from opening positions"),
		"32027": errors.New("cancelled over 20 orders"),
		"32028": errors.New("account is suspended and liquidated"),
		"32029": errors.New("order info does not exist"),
		"33001": errors.New("margin account for this pair is not enabled yet"),
		"33002": errors.New("margin account for this pair is suspended"),
		"33003": errors.New("no loan balance"),
		"33004": errors.New("loan amount cannot be smaller than the minimum limit"),
		"33005": errors.New("repayment amount must exceed 0"),
		"33006": errors.New("loan order not found"),
		"33007": errors.New("status not found"),
		"33008": errors.New("loan amount cannot exceed the maximum limit"),
		"33009": errors.New("user ID is blank"),
		"33010": errors.New("you cannot cancel an order during session 2 of call auction"),
		"33011": errors.New("no new market data"),
		"33012": errors.New("order cancellation failed"),
		"33013": errors.New("order placement failed"),
		"33014": errors.New("order does not exist"),
		"33015": errors.New("exceeded maximum limit"),
		"33016": errors.New("margin trading is not open for this token"),
		"33017": errors.New("insufficient balance"),
		"33018": errors.New("this parameter must be smaller than 1"),
		"33020": errors.New("request not supported"),
		"33021": errors.New("token and the pair do not match"),
		"33022": errors.New("pair and the order do not match"),
		"33023": errors.New("you can only place market orders during call auction"),
		"33024": errors.New("trading amount too small"),
		"33025": errors.New("base token amount is blank"),
		"33026": errors.New("transaction completed"),
		"33027": errors.New("cancelled order or order cancelling"),
		"33028": errors.New("the decimal places of the trading price exceeded the limit"),
		"33029": errors.New("the decimal places of the trading size exceeded the limit"),
		"34001": errors.New("withdrawal suspended"),
		"34002": errors.New("please add a withdrawal address"),
		"34003": errors.New("sorry, this token cannot be withdrawn to xx at the moment"),
		"34004": errors.New("withdrawal fee is smaller than minimum limit"),
		"34005": errors.New("withdrawal fee exceeds the maximum limit"),
		"34006": errors.New("withdrawal amount is lower than the minimum limit"),
		"34007": errors.New("withdrawal amount exceeds the maximum limit"),
		"34008": errors.New("insufficient balance"),
		"34009": errors.New("your withdrawal amount exceeds the daily limit"),
		"34010": errors.New("transfer amount must be larger than 0"),
		"34011": errors.New("conditions not met"),
		"34012": errors.New("the minimum withdrawal amount for NEO is 1, and the amount must be an integer"),
		"34013": errors.New("please transfer"),
		"34014": errors.New("transfer limited"),
		"34015": errors.New("subaccount does not exist"),
		"34016": errors.New("transfer suspended"),
		"34017": errors.New("account suspended"),
		"34018": errors.New("incorrect trades password"),
		"34019": errors.New("please bind your email before withdrawal"),
		"34020": errors.New("please bind your funds password before withdrawal"),
		"34021": errors.New("not verified address"),
		"34022": errors.New("withdrawals are not available for sub accounts"),
		"35001": errors.New("contract subscribing does not exist"),
		"35002": errors.New("contract is being settled"),
		"35003": errors.New("contract is being paused"),
		"35004": errors.New("pending contract settlement"),
		"35005": errors.New("perpetual swap trading is not enabled"),
		"35008": errors.New("margin ratio too low when placing order"),
		"35010": errors.New("closing position size larger than available size"),
		"35012": errors.New("placing an order with less than 1 contract"),
		"35014": errors.New("order size is not in acceptable range"),
		"35015": errors.New("leverage level unavailable"),
		"35017": errors.New("changing leverage level"),
		"35019": errors.New("order size exceeds limit"),
		"35020": errors.New("order price exceeds limit"),
		"35021": errors.New("order size exceeds limit of the current tier"),
		"35022": errors.New("contract is paused or closed"),
		"35030": errors.New("place multiple orders"),
		"35031": errors.New("cancel multiple orders"),
		"35061": errors.New("invalid instrument_id"),
	}
}

// ---------------------------------------------- New --------------------------------------------------

// SystemStatus represents system status
type SystemStatus struct {
	Title       string `json:"title"`
	State       string `json:"state"`
	Begin       string `json:"begin"`
	End         string `json:"end"`
	Href        string `json:"href"`
	ServiceType string `json:"serviceType"`
	System      string `json:"system"`
	ScheDesc    string `json:"scheDesc"`
}

// Instrument represents an instrument in an open contract.
type Instrument struct {
	Alias          string         `json:"alias"`
	BaseCurrency   string         `json:"baseCcy"`
	Category       string         `json:"category"`
	CtMult         string         `json:"ctMult"`
	CtType         string         `json:"ctType"`
	CtVal          string         `json:"ctVal"`
	CtValCurrency  string         `json:"ctValCcy"`
	ExpTime        string         `json:"expTime"`
	InstFamily     string         `json:"instFamily"`
	InstrumentID   string         `json:"instId"`
	InstrumentType string         `json:"instType"`
	Leverage       string         `json:"lever"`
	ListTime       okcoinMilliSec `json:"listTime"`
	LotSize        string         `json:"lotSz"`
	MaxIcebergSz   string         `json:"maxIcebergSz"`
	MaxLimitSize   float64        `json:"maxLmtSz,string"`
	MaxMarketSize  float64        `json:"maxMktSz,string"`
	MaxStopSize    float64        `json:"maxStopSz,string"`
	MaxTwapSize    float64        `json:"maxTwapSz,string"`
	MaxTriggerSize float64        `json:"maxTriggerSz,string"`
	MinSize        float64        `json:"minSz,string"`
	QuoteCurrency  string         `json:"quoteCcy"`
	OptionType     string         `json:"optType"`
	SettleCurrency string         `json:"settleCcy"`
	State          string         `json:"state"`
	StrikePrice    string         `json:"stk"`
	TickSize       float64        `json:"tickSz,string"`
	Underlying     string         `json:"uly"`
}

type candlestickItemResponse [9]string

// CandlestickData represents the candlestick chart
type CandlestickData struct {
	Timestamp            okcoinMilliSec
	OpenPrice            float64
	HighestPrice         float64
	LowestPrice          float64
	ClosePrice           float64
	TradingVolume        float64
	QuoteTradingVolume   float64
	TradingVolumeInQuote float64
	Confirm              string
}

// SpotTrade represents spot trades
type SpotTrade struct {
	InstID     string         `json:"instId"`
	Side       string         `json:"side"`
	TradeSize  string         `json:"sz"`
	TradePrice string         `json:"px"`
	TradeID    string         `json:"tradeId"`
	Timestamp  okcoinMilliSec `json:"ts"`
}

// TradingVolume represents the trading volume of the platform in 24 hours
type TradingVolume struct {
	VolCny    float64        `json:"volCny,string"`
	VolUsd    float64        `json:"volUsd,string"`
	Timestamp okcoinMilliSec `json:"ts"`
}

// Oracle represents crypto price of signing using Open Oracle smart contract.
type Oracle []struct {
	Messages   []string          `json:"messages"`
	Prices     map[string]string `json:"prices"`
	Signatures []string          `json:"signatures"`
	Timestamp  okcoinMilliSec    `json:"timestamp"`
}

// ExchangeRate represents average exchange rate data
type ExchangeRate struct {
	UsdCny string `json:"usdCny"`
}

// ToExtract returns a CandlestickData instance from []string
func (c *candlestickItemResponse) ToExtract() (CandlestickData, error) {
	var candle CandlestickData
	err := json.Unmarshal([]byte(c[0]), &candle.Timestamp)
	if err != nil {
		return candle, err
	}
	candle.OpenPrice, err = strconv.ParseFloat(c[1], 64)
	if err != nil {
		return candle, err
	}
	candle.HighestPrice, err = strconv.ParseFloat(c[2], 64)
	if err != nil {
		return candle, err
	}
	candle.LowestPrice, err = strconv.ParseFloat(c[3], 64)
	if err != nil {
		return candle, err
	}
	candle.ClosePrice, err = strconv.ParseFloat(c[4], 64)
	if err != nil {
		return candle, err
	}
	candle.TradingVolume, err = strconv.ParseFloat(c[5], 64)
	if err != nil {
		return candle, err
	}
	candle.QuoteTradingVolume, err = strconv.ParseFloat(c[6], 64)
	if err != nil {
		return candle, err
	}
	candle.TradingVolumeInQuote, err = strconv.ParseFloat(c[7], 64)
	if err != nil {
		return candle, err
	}
	candle.Confirm = c[8]
	return candle, nil
}

// ExtractCandlesticks retrives a list of CandlestickData
func ExtractCandlesticks(candles []candlestickItemResponse) ([]CandlestickData, error) {
	candlestickData := make([]CandlestickData, len(candles))
	var err error
	for x := range candles {
		candlestickData[x], err = candles[x].ToExtract()
		if err != nil {
			return nil, err
		}
	}
	return candlestickData, nil
}

// CurrencyInfo represents a currency instance detailed information
type CurrencyInfo struct {
	CanDep                     bool    `json:"canDep"`
	CanInternal                bool    `json:"canInternal"`
	CanWd                      bool    `json:"canWd"`
	Currency                   string  `json:"ccy"`
	Chain                      string  `json:"chain"`
	DepQuotaFixed              string  `json:"depQuotaFixed"`
	DepQuoteDailyLayer2        string  `json:"depQuoteDailyLayer2"`
	LogoLink                   string  `json:"logoLink"`
	MainNet                    bool    `json:"mainNet"`
	MaxFee                     float64 `json:"maxFee,string"`
	MaxWithdrawal              float64 `json:"maxWd,string"`
	MinDeposit                 float64 `json:"minDep,string"`
	MinDepArrivalConfirm       string  `json:"minDepArrivalConfirm"`
	MinFee                     float64 `json:"minFee,string"`
	MinWithdrawal              float64 `json:"minWd,string"`
	MinWithdrawalUnlockConfirm string  `json:"minWdUnlockConfirm"`
	Name                       string  `json:"name"`
	NeedTag                    bool    `json:"needTag"`
	UsedDepQuotaFixed          string  `json:"usedDepQuotaFixed"`
	UsedWdQuota                string  `json:"usedWdQuota"`
	WithdrawalQuota            string  `json:"wdQuota"`
	WithdrawalTickSize         float64 `json:"wdTickSz,string"`
}

// CurrencyBalance represents a currency balance information.
type CurrencyBalance struct {
	AvailableBalance float64 `json:"availBal,string"`
	Balance          float64 `json:"bal,string"`
	Currency         string  `json:"ccy"`
	FrozenBalance    float64 `json:"frozenBal,string"`
}

// AccountAssetValuation represents account asset valuation
type AccountAssetValuation struct {
	Details struct {
		Classic float64 `json:"classic,string"`
		Earn    float64 `json:"earn,string"`
		Funding float64 `json:"funding,string"`
		Trading float64 `json:"trading,string"`
	} `json:"details"`
	TotalBalance float64        `json:"totalBal,string"`
	Timestamp    okcoinMilliSec `json:"ts"`
}

// FundingTransferRequest represents a transfer of funds between your funding account and trading account
type FundingTransferRequest struct {
	Currency currency.Code `json:"ccy"`
	// Transfer type
	// 0: transfer within account
	// 1: master account to sub-account (Only applicable to APIKey from master account)
	// 2: sub-account to master account (Only applicable to APIKey from master account)
	// 3: sub-account to master account (Only applicable to APIKey from sub-account)
	// The default is 0.
	Amount       float64 `json:"amt,string"`
	From         string  `json:"from"`
	To           string  `json:"to"`
	TransferType int32   `json:"type,string,omitempty"`
	SubAccount   string  `json:"subAcct,omitempty"`
	ClientID     string  `json:"clientId,omitempty"`
}

// FundingTransferItem represents a response for a transfer of funds between your funding account and trading account
type FundingTransferItem struct {
	TransferID string  `json:"transId"`
	Currency   string  `json:"ccy"`
	ClientID   string  `json:"clientId"`
	From       string  `json:"from"`
	Amount     float64 `json:"amt,string"`
	InstID     string  `json:"instId"`
	State      string  `json:"state"`
	SubAcct    string  `json:"subAcct"`
	To         string  `json:"to"`
	ToInstID   string  `json:"toInstId"`
	Type       string  `json:"type"`
}

// AssetBillDetail represents the billing record.
type AssetBillDetail struct {
	BillID    string         `json:"billId"`
	Currency  string         `json:"ccy"`
	ClientID  string         `json:"clientId"`
	BalChange float64        `json:"balChg,string"`
	Bal       float64        `json:"bal,string"`
	Type      string         `json:"type"`
	Timestamp okcoinMilliSec `json:"ts"`
}

// LightningDepositDetail represents a lightning deposit instance detail
type LightningDepositDetail struct {
	CreationTime okcoinMilliSec `json:"cTime"`
	Invoice      string         `json:"invoice"`
}

// DepositAddress represents a currency deposit address detail
type DepositAddress struct {
	Chain                     string `json:"chain"`
	ContractAddr              string `json:"ctAddr"`
	Ccy                       string `json:"ccy"`
	To                        string `json:"to"`
	Address                   string `json:"addr"`
	Selected                  bool   `json:"selected"`
	Tag                       string `json:"tag"`
	Memo                      string `json:"memo"`
	DepositPaymentID          string `json:"pmtId"`
	DepositAddressAttachement string `json:"addrEx"`
}

// DepositHistoryItem represents deposit records according to the currency, deposit status, and time range in reverse chronological order.
type DepositHistoryItem struct {
	ActualDepBlkConfirm string         `json:"actualDepBlkConfirm"` // ActualDepBlkConfirm actual amount of blockchain confirm in a single deposit
	Amount              float64        `json:"amt,string"`
	Currency            string         `json:"ccy"`
	Chain               string         `json:"chain"`
	DepositID           string         `json:"depId"`
	From                string         `json:"from"`
	State               string         `json:"state"`
	To                  string         `json:"to"`
	Timstamp            okcoinMilliSec `json:"ts"`
	TransactionID       string         `json:"txId"`
}

// WithdrawalRequest represents withdrawal of tokens request.
type WithdrawalRequest struct {
	Amount           float64       `json:"amt,string,omitempty"`
	TransactionFee   float64       `json:"fee,string,omitempty"`
	WithdrawalMethod string        `json:"dest,omitempty"` // Withdrawal method 3: internal  4: on chain
	Ccy              currency.Code `json:"ccy,omitempty"`
	Chain            string        `json:"chain,omitempty"`
	ClientID         string        `json:"clientId,omitempty"`
	ToAddress        string        `json:"toAddr,omitempty"`
}

// WithdrawalResponse represents withdrawal of tokens response.
type WithdrawalResponse struct {
	Amt      string `json:"amt"`
	WdID     string `json:"wdId"`
	Ccy      string `json:"ccy"`
	ClientID string `json:"clientId"`
	Chain    string `json:"chain"`
}

// LightningWithdrawalsRequest represents lightning withdrawal request params
type LightningWithdrawalsRequest struct {
	Ccy     currency.Code `json:"ccy"`
	Invoice string        `json:"invoice"`
	Memo    string        `json:"memo,omitempty"`
}

// LightningWithdrawals the minimum withdrawal amount is approximately 0.000001 BTC. Sub-account does not support withdrawal.
type LightningWithdrawals struct {
	WithdrawalID string         `json:"wdId"`
	CreationTime okcoinMilliSec `json:"cTime"`
}

// WithdrawalCancelation represents a request parameter for withdrawal cancellation
type WithdrawalCancelation struct {
	WithdrawalID string `json:"wdId"`
}

// WithdrawalOrderItem represents a withdrawal instance item
type WithdrawalOrderItem struct {
	Chain         string         `json:"chain"`
	Fee           float64        `json:"fee,string"`
	Ccy           string         `json:"ccy"`
	ClientID      string         `json:"clientId"`
	Amt           float64        `json:"amt,string"`
	TransactionID string         `json:"txId"`
	From          string         `json:"from"`
	To            string         `json:"to"`
	State         string         `json:"state"`
	Timestamp     okcoinMilliSec `json:"ts"`
	WithdrawalID  string         `json:"wdId"`
}
