package okgroup

import "encoding/json"
import "github.com/thrasher-/gocryptotrader/currency/symbol"

// GetAccountCurrenciesResponse response data for GetAccountCurrencies
type GetAccountCurrenciesResponse struct {
	CanDeposit    int64   `json:"can_deposit"`
	CanWithdraw   int64   `json:"can_withdraw"`
	Currency      string  `json:"currency"`
	MinWithdrawal float64 `json:"min_withdrawal"`
	Name          string  `json:"name"`
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
	Amount      int64   `json:"amount"`      // [required] withdrawal amount
	Currency    string  `json:"currency"`    // [required] token
	Destination int64   `json:"destination"` // [required] withdrawal address(2:OKCoin International 3:OKEx 4:others)
	Fee         float64 `json:"fee"`         // [required] Network transaction fee≥0. Withdrawals to OKCoin or OKEx are fee-free, please set as 0. Withdrawal to external digital asset address requires network transaction fee.
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
	Available float64 `json:"available"`
	Balance   float64 `json:"balance"`
	Currency  string  `json:"currency"`
	Hold      float64 `json:"hold"`
}

// WithdrawalHistoryResponse response data for WithdrawalHistoryResponse
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

// GetAccountBillDetailsRequest request data for GetAccountBillDetailsRequest
type GetAccountBillDetailsRequest struct {
	Currency string `json:"currency"` // [optional] token ,information of all tokens will be returned if the field is left blank
	Type     int64  `json:"type"`     // [optional] 1:deposit 2:withdrawal 13:cancel withdrawal 18: into futures account 19: out of futures account 20:into sub account 21:out of sub account 28: claim 29: into ETT account 30: out of ETT account 31: into C2C account 32:out of C2C account 33: into margin account 34: out of margin account 37: into spot account 38: out of spot account
	From     int64  `json:"from"`     // [optional] you would request pages after this page.
	To       int64  `json:"to"`       // [optional] you would request pages before this page
	Limit    int64  `json:"limit"`    // [optional] Number of results per request. Maximum 100. (default 100)
}

// GetAccountBillDetailsResponse response data for GetAccountBillDetails
type GetAccountBillDetailsResponse struct {
	Amount    float64 `json:"amount"`
	Balance   int64   `json:"balance"`
	Currency  string  `json:"currency"`
	Fee       int64   `json:"fee"`
	LedgerID  int64   `json:"ledger_id"`
	Timestamp string  `json:"timestamp"`
	Typename  string  `json:"typename"`
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
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	Status        int64   `json:"status"`
	Timestamp     string  `json:"timestamp"`
	To            string  `json:"to"`
	TransactionID string  `json:"txid"`
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
	Currency string `json:"currency"`               // [required] token
	From     int64  `json:"from,string,omitempty"`  // [optional] request page before(newer) this id.
	To       int64  `json:"to,string,omitempty"`    // [optional] request page after(older) this id.
	Limit    int64  `json:"limit,string,omitempty"` // [optional] number of results per request. Maximum 100.(default 100)
}

// GetSpotBillDetailsForCurrencyResponse response data for GetSpotBillDetailsForCurrency
type GetSpotBillDetailsForCurrencyResponse struct {
	LedgerID         string          `json:"ledger_id"`
	Balance          string          `json:"balance"`
	CurrencyResponse string          `json:"currency"`
	Amount           string          `json:"amount"`
	Type             string          `json:"type"`
	TimeStamp        string          `json:"timestamp"`
	Details          SpotBillDetails `json:"details"`
}

// SpotBillDetails response data for GetSpotBillDetailsForCurrency
type SpotBillDetails struct {
	OrderID      string `json:"order_id"`
	InstrumentID string `json:"instrument_id"`
}

// PlaceSpotOrderRequest request data for PlaceSpotOrder
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

// PlaceSpotOrderResponse response data for PlaceSpotOrder
type PlaceSpotOrderResponse struct {
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
	OrderID   int64  `json:"order_id"`
	Result    bool   `json:"result"`
}

// CancelMultipleSpotOrdersRequest request data for CancelMultipleSpotOrders
type CancelMultipleSpotOrdersRequest struct {
	OrderIDs     []int64 `json:"order_ids,omitempty"` // order ID. You may cancel up to 4 orders of a trading pair
	InstrumentID string  `json:"instrument_id"`       // by providing this parameter, the corresponding order of a designated trading pair will be cancelled. If not providing this parameter, it will be back to error code.
}

// CancelMultipleSpotOrdersResponse response data for CancelMultipleSpotOrders
type CancelMultipleSpotOrdersResponse struct {
	ClientOID string  `json:"client_oid"`
	OrderID   []int64 `json:"order_id,string"`
	Result    bool    `json:"result"`
}

// GetSpotOrdersRequest request data for GetSpotOrders
type GetSpotOrdersRequest struct {
	Status string `json:"status"` // list the status of all orders (all, open, part_filled, canceling, filled, cancelled，ordering,failure)
	// （Multiple status separated by '|'，and '|' need encode to ' %7C'）
	InstrumentID string `json:"instrument_id"`          // trading pair ,information of all trading pair will be returned if the field is left blank
	From         int64  `json:"from,string,omitempty"`  // [optional] request page after this id (latest information) (eg. 1, 2, 3, 4, 5. There is only a 5 "from 4", while there are 1, 2, 3 "to 4")
	To           int64  `json:"to,string,omitempty"`    // [optional] request page after (older) this id.
	Limit        int64  `json:"limit,string,omitempty"` // [optional] number of results per request. Maximum 100. (default 100)
}

// GetSpotOrderResponse response data for GetSpotOrders
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

// GetSpotOpenOrdersRequest request data for GetSpotOpenOrders
type GetSpotOpenOrdersRequest struct {
	InstrumentID string `json:"instrument_id"`          // [optional] trading pair ,information of all trading pair will be returned if the field is left blank
	From         int64  `json:"from,string,omitempty"`  // [optional] request page after this id (latest information) (eg. 1, 2, 3, 4, 5. There is only a 5 "from 4", while there are 1, 2, 3 "to 4")
	To           int64  `json:"to,string,omitempty"`    // [optional] request page after (older) this id.
	Limit        int64  `json:"limit,string,omitempty"` // [optional] number of results per request. Maximum 100. (default 100)
}

// GetSpotOrderRequest request data for GetSpotOrder
type GetSpotOrderRequest struct {
	OrderID      int64  `json:"order_id,string"` // [required] order ID
	InstrumentID string `json:"instrument_id"`   // [required]trading pair
}

// GetSpotTransactionDetailsRequest request data for GetSpotTransactionDetails
type GetSpotTransactionDetailsRequest struct {
	InstrumentID string `json:"instrument_id"`          // [required]list all transaction details of this instrument_id.
	OrderID      int64  `json:"order_id,string"`        // [required]list all transaction details of this order_id.
	From         int64  `json:"from,string,omitempty"`  // [optional] request page after this id (latest information) (eg. 1, 2, 3, 4, 5. There is only a 5 "from 4", while there are 1, 2, 3 "to 4")
	To           int64  `json:"to,string,omitempty"`    // [optional] request page after (older) this id.
	Limit        int64  `json:"limit,string,omitempty"` // [optional] number of results per request. Maximum 100. (default 100)
}

// GetSpotTransactionDetailsResponse response data for GetSpotTransactionDetails
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

// GetSpotTokenPairDetailsResponse response data for GetSpotTokenPairDetails
type GetSpotTokenPairDetailsResponse struct {
	BaseCurrency  string `json:"base_currency"`
	InstrumentID  string `json:"instrument_id"`
	MinSize       string `json:"min_size"`
	QuoteCurrency string `json:"quote_currency"`
	SizeIncrement string `json:"size_increment"`
	TickSize      string `json:"tick_size"`
}

// GetSpotOrderBookRequest request data for GetSpotOrderBook
type GetSpotOrderBookRequest struct {
	Size         int64   `json:"size,string,omitempty"`  // [optional] number of results per request. Maximum 200
	Depth        float64 `json:"depth,string,omitempty"` // [optional] the aggregation of the book. e.g . 0.1,0.001
	InstrumentID string  `json:"instrument_id"`          // [required] trading pairs
}

// GetSpotOrderBookResponse response data for GetSpotOrderBook
type GetSpotOrderBookResponse struct {
	Timestamp string     `json:"timestamp"`
	Asks      [][]string `json:"asks"` // [[0]: "Price", [1]: "Size", [2]: "Num_orders"], ...
	Bids      [][]string `json:"bids"` // [[0]: "Price", [1]: "Size", [2]: "Num_orders"], ...
}

// GetSpotTokenPairsInformationResponse response data for GetSpotTokenPairsInformation
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

// GetSpotFilledOrdersInformationRequest request data for GetSpotFilledOrdersInformation
type GetSpotFilledOrdersInformationRequest struct {
	InstrumentID string `json:"instrument_id"`          // [required] trading pairs
	From         int64  `json:"from,string,omitempty"`  // [optional] number of results per request. Maximum 100. (default 100)
	To           int64  `json:"to,string,omitempty"`    // [optional] request page after (older) this id.
	Limit        int64  `json:"limit,string,omitempty"` // [optional] number of results per request. Maximum 100. (default 100)
}

// GetSpotFilledOrdersInformationResponse response data for GetSpotFilledOrdersInformation
type GetSpotFilledOrdersInformationResponse struct {
	Price     string `json:"price"`
	Side      string `json:"side"`
	Size      string `json:"size"`
	Timestamp string `json:"timestamp"`
	TradeID   string `json:"trade_id"`
}

// GetSpotMarketDataRequest request data for GetSpotMarketData
type GetSpotMarketDataRequest struct {
	Start        string `json:"start,omitempty"` // [optional] start time in ISO 8601
	End          string `json:"end,omitempty"`   // [optional] end time in ISO 8601
	Granularity  int64  `json:"granularity"`     // The granularity field must be one of the following values: {60, 180, 300, 900, 1800, 3600, 7200, 14400, 43200, 86400, 604800}.
	InstrumentID string `json:"instrument_id"`   // [required] trading pairs
}

// GetSpotMarketDataResponse response data for GetSpotMarketData
// Return Parameters
// time 	string 	Start time
// open 	string 	Open price
// high 	string 	Highest price
// low 	string 	Lowest price
// close 	string 	Close price
// volume 	string 	Trading volume
type GetSpotMarketDataResponse []interface{}

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

// ----------------------------------------------------------------------------------------------------------------------------

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
	InstrumentID string `json:"instrument_id,omitempty"` //  	Contract ID, e.g. “BTC-USD-180213”
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
	InstrumentID string  `json:"instrument_id"`                // [required]   	Contract ID,e.g. “TC-USD-180213”
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
	InstrumentID string                                 `json:"instrument_id"` // [required] Contract ID, e.g.“BTC-USD-180213”
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
	InstrumentID string `json:"instrument_id"` // [required] Contract ID,e.g. “BTC-USD-180213”
}

// CancelFuturesOrderResponse response data from CancelFuturesOrder
type CancelFuturesOrderResponse struct {
	InstrumentID string `json:"instrument_id"`
	OrderID      string `json:"order_id"`
	Result       bool   `json:"result"`
}

// GetFuturesOrdersListRequest request data for GetFutureOrdersList
type GetFuturesOrdersListRequest struct {
	InstrumentID string `json:"instrument_id"`          // [required] Contract ID, e.g. “BTC-USD-180213”
	Status       int64  `json:"status,string"`          // [required] Order Status （-1 canceled; 0: pending, 1: partially filled, 2: fully filled, 6: open (pending partially + fully filled), 7: completed (canceled + fully filled))
	From         int64  `json:"from,string,omitempty"`  // [optional] Request paging content for this page number.（Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	To           int64  `json:"to,string,omitempty"`    // [optional] Request page after (older) this pagination id. （Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	Limit        int64  `json:"limit,string,omitempty"` // [optional] Number of results per request. Maximum 100. (default 100)
}

// GetFuturesOrderListResponse response data from GetFuturesOrderList
type GetFuturesOrderListResponse struct {
	OrderInfo []GetFuturesOrderDetailsResponseData `json:"order_info"`
	Result    bool                                 `json:"result"`
}

// GetFuturesOrderDetailsResponseData individual order data from GetFuturesOrderList
type GetFuturesOrderDetailsResponseData struct {
	ContractVal  float64 `json:"contract_val,string"`
	Fee          float64 `json:"fee,string"`
	FilledQty    float64 `json:"filled_qty,string"`
	InstrumentID string  `json:"instrument_id"`
	Leverage     int64   `json:"leverage,string"` //  	Leverage value:10\20 default:10
	OrderID      int64   `json:"order_id,string"`
	Price        float64 `json:"price,string"`
	PriceAvg     float64 `json:"price_avg,string"`
	Size         float64 `json:"size,string"`
	Status       int64   `json:"status,string"` // Order Status （-1 canceled; 0: pending, 1: partially filled, 2: fully filled)
	Timestamp    string  `json:"timestamp"`
	Type         int64   `json:"type,string"` //  	Type (1: open long 2: open short 3: close long 4: close short)
}

// GetFuturesOrderDetailsRequest request data for GetFuturesOrderDetails
type GetFuturesOrderDetailsRequest struct {
	OrderID      int64  `json:"order_id,string"` // [required] Order ID
	InstrumentID string `json:"instrument_id"`   // [required] Contract ID, e.g. “BTC-USD-180213”
}

// GetFuturesTransactionDetailsRequest request data for GetFuturesTransactionDetails
type GetFuturesTransactionDetailsRequest struct {
	OrderID      int64  `json:"order_id,string"`        // [required] Order ID
	InstrumentID string `json:"instrument_id"`          // [required] Contract ID, e.g. “BTC-USD-180213”
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
	ContractVal     int64   `json:"contract_val,string"`
	Delivery        string  `json:"delivery"`
	InstrumentID    string  `json:"instrument_id"`
	Listing         string  `json:"listing"`
	QuoteCurrency   string  `json:"quote_currency"`
	TickSize        float64 `json:"tick_size,string"`
	TradeIncrement  int64   `json:"trade_increment,string"`
	UnderlyingIndex string  `json:"underlying_index"`
}

// GetFuturesOrderBookRequest request data for GetFuturesOrderBook
type GetFuturesOrderBookRequest struct {
	InstrumentID string `json:"instrument_id"` // [required] Contract ID, e.g. “BTC-USD-180213”
	Size         int64  `json:"size"`          // [optional] The size of the price range (max: 200)
}

// GetFuturesOrderBookResponse response data for GetFuturesOrderBook
type GetFuturesOrderBookResponse struct {
	Asks      [][]float64 `json:"asks"` // [[0: Price, 1: Size price, 2: number of force liquidated orders, 3: number of orders on the price]]
	Bids      [][]float64 `json:"bids"` // [[0: Price, 1: Size price, 2: number of force liquidated orders, 3: number of orders on the price]]
	Timestamp string      `json:"timestamp"`
}

// GetFuturesTokenInfoResponse response data for GetFuturesOrderBook
type GetFuturesTokenInfoResponse struct {
	BestAsk      float64 `json:"best_ask,string"`
	BestBid      float64 `json:"best_bid,string"`
	High24h      float64 `json:"high_24h,string"`
	InstrumentID string  `json:"instrument_id"`
	Last         float64 `json:"last,string"`
	Low24h       float64 `json:"low_24h,string"`
	Timestamp    string  `json:"timestamp"`
	Volume24h    int64   `json:"volume_24h,string"`
}

// GetFuturesFilledOrderRequest request data for GetFuturesFilledOrder
type GetFuturesFilledOrderRequest struct {
	InstrumentID string `json:"instrument_id"`          // [required] Contract ID, e.g. “BTC-USD-180213”
	From         int64  `json:"from,string,omitempty"`  // [optional] Request paging content for this page number.（Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	To           int64  `json:"to,string,omitempty"`    // [optional] Request page after (older) this pagination id. （Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	Limit        int64  `json:"limit,string,omitempty"` // [optional]  	Number of results per request. Maximum 100. (default 100)
}

// GetFuturesFilledOrdersResponse response data for GetFuturesFilledOrders
type GetFuturesFilledOrdersResponse struct {
	Price     float64 `json:"price,string"`
	Qty       int64   `json:"qty,string"`
	Side      string  `json:"side"`
	Timestamp string  `json:"timestamp"`
	TradeID   string  `json:"trade_id"`
}

// GetFuturesMarketDateRequest retrieves candle data information
type GetFuturesMarketDateRequest struct {
	Start        string `json:"start,omitempty"` // [optional] start time in ISO 8601
	End          string `json:"end,omitempty"`   // [optional] end time in ISO 8601
	Granularity  int64  `json:"granularity"`     // The granularity field must be one of the following values: {60, 180, 300, 900, 1800, 3600, 7200, 14400, 43200, 86400, 604800}.
	InstrumentID string `json:"instrument_id"`   // [required] trading pairs
}

// GetFuturesMarketDataResponse contains candle data from a GetSpotMarketDataRequest
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
	Amount       float64 `json:"amount,string"`
	InstrumentID string  `json:"instrument_id"`
	Timestamp    string  `json:"timestamp"`
}

// GetFuturesIndicesResponse response data for GetFuturesIndices
type GetFuturesIndicesResponse struct {
	Index        float64 `json:"index,string"`
	InstrumentID string  `json:"instrument_id"`
	Timestamp    string  `json:"timestamp"`
}

// GetFuturesExchangeRatesResponse response data for GetFuturesExchangeRate
type GetFuturesExchangeRatesResponse struct {
	InstrumentID string  `json:"instrument_id"`
	Rate         float64 `json:"rate,string"`
	Timestamp    string  `json:"timestamp"`
}

// GetFuturesEstimatedDeliveryPriceResponse response data for GetFuturesEstimatedDeliveryPrice
type GetFuturesEstimatedDeliveryPriceResponse struct {
	InstrumentID    string  `json:"instrument_id"`
	SettlementPrice float64 `json:"settlement_price,string"`
	Timestamp       string  `json:"timestamp"`
}

// GetFuturesOpenInterestsResponse response data for GetFuturesOpenInterests
type GetFuturesOpenInterestsResponse struct {
	Amount       float64 `json:"amount,string"`
	InstrumentID string  `json:"instrument_id"`
	Timestamp    string  `json:"timestamp"`
}

// GetFuturesCurrentPriceLimitResponse response data for GetFuturesCurrentPriceLimit
type GetFuturesCurrentPriceLimitResponse struct {
	Highest      float64 `json:"highest,string"`
	InstrumentID string  `json:"instrument_id"`
	Lowest       float64 `json:"lowest,string"`
	Timestamp    string  `json:"timestamp"`
}

// GetFuturesCurrentMarkPriceResponse response data for GetFuturesCurrentMarkPrice
type GetFuturesCurrentMarkPriceResponse struct {
	MarkPrice    float64 `json:"mark_price"`
	InstrumentID string  `json:"instrument_id"`
	Timestamp    string  `json:"timestamp"`
}

// GetFuturesForceLiquidatedOrdersRequest request data for GetFuturesForceLiquidatedOrders
type GetFuturesForceLiquidatedOrdersRequest struct {
	InstrumentID string `json:"instrument_id"`          // [required] Contract ID, e.g. “BTC-USD-180213”
	From         int64  `json:"from,string,omitempty"`  // [optional] Request paging content for this page number.（Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	To           int64  `json:"to,string,omitempty"`    // [optional] Request page after (older) this pagination id. （Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	Limit        int64  `json:"limit,string,omitempty"` // [optional]  	Number of results per request. Maximum 100. (default 100)
	Status       string `json:"status,omitempty"`       // [optional] Status (0:unfilled orders in the recent 7 days 1:filled orders in the recent 7 days)
}

// GetFuturesForceLiquidatedOrdersResponse response data for GetFuturesForceLiquidatedOrders
type GetFuturesForceLiquidatedOrdersResponse struct {
	Loss         float64 `json:"loss"`
	Size         int64   `json:"size"`
	Price        float64 `json:"price"`
	CreatedAt    string  `json:"created_at"`
	InstrumentID string  `json:"instrument_id"`
	Type         int64   `json:"type"`
}

// GetFuturesTagPriceResponse response data for GetFuturesTagPrice
type GetFuturesTagPriceResponse struct {
	MarkPrice    float64 `json:"mark_price"`
	InstrumentID string  `json:"instrument_id"`
	Timestamp    string  `json:"timestamp"`
}

// ----------------------------------------------------------------------------------------------------------------------------

// GetSwapPostionsResponse response data for GetSwapPostions
type GetSwapPostionsResponse struct {
	MarginMode string                           `json:"margin_mode"`
	Holding    []GetSwapPostionsResponseHolding `json:"holding"`
}

// GetSwapPostionsResponseHolding response data for GetSwapPostions
type GetSwapPostionsResponseHolding struct {
	AvailPosition    string `json:"avail_position"`
	AvgCost          string `json:"avg_cost"`
	InstrumentID     string `json:"instrument_id"`
	Leverage         string `json:"leverage"`
	LiquidationPrice string `json:"liquidation_price"`
	Margin           string `json:"margin"`
	Position         string `json:"position"`
	RealizedPnl      string `json:"realized_pnl"`
	SettlementPrice  string `json:"settlement_price"`
	Side             string `json:"side"`
	Timestamp        string `json:"timestamp"`
}

// GetSwapAccountOfAllCurrencyResponse response data for GetSwapAccountOfAllCurrency
type GetSwapAccountOfAllCurrencyResponse struct {
	Info []GetSwapAccountOfAllCurrencyResponseInfo `json:"info"`
}

// GetSwapAccountOfAllCurrencyResponseInfo response data for GetSwapAccountOfAllCurrency
type GetSwapAccountOfAllCurrencyResponseInfo struct {
	Equity            string `json:"equity"`
	FixedBalance      string `json:"fixed_balance"`
	TotalAvailBalance string `json:"total_avail_balance"`
	Margin            string `json:"margin"`
	RealizedPnl       string `json:"realized_pnl"`
	UnrealizedPnl     string `json:"unrealized_pnl"`
	MarginRatio       string `json:"margin_ratio"`
	InstrumentID      string `json:"instrument_id"`
	MarginFrozen      string `json:"margin_frozen"`
	Timestamp         string `json:"timestamp"`
	MarginMode        string `json:"margin_mode"`
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
	LedgerID     string `json:"ledger_id"`
	Amount       string `json:"amount"`
	Type         string `json:"type"`
	Fee          string `json:"fee"`
	Timestamp    string `json:"timestamp"`
	InstrumentID string `json:"instrument_id"`
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

// CancelSwapOrderResponse repsonse data for CancelSwapOrder
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

// CancelMultipleSwapOrdersResponse repsonse data for CancelMultipleSwapOrders
type CancelMultipleSwapOrdersResponse struct {
	Result       bool     `json:"result,string"`
	OrderIDS     []string `json:"order_ids"`
	InstrumentID string   `json:"instrument_id"`
}

// GetSwapOrderListRequest request data for GetSwapOrderList
type GetSwapOrderListRequest struct {
	InstrumentID string `json:"instrument_id"`          // [required] Contract ID, e.g. “BTC-USD-180213”
	Status       int64  `json:"status,string"`          // [required] Order Status （-1 canceled; 0: pending, 1: partially filled, 2: fully filled, 6: open (pending partially + fully filled), 7: completed (canceled + fully filled))
	From         int64  `json:"from,string,omitempty"`  // [optional] Request paging content for this page number.（Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	To           int64  `json:"to,string,omitempty"`    // [optional] Request page after (older) this pagination id. （Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	Limit        int64  `json:"limit,string,omitempty"` // [optional] Number of results per request. Maximum 100. (default 100)
}

// GetSwapOrderListResponse  response data for GetSwapOrderList
type GetSwapOrderListResponse struct {
	Result    bool                           `json:"result,string"`
	OrderInfo []GetSwapOrderListResponseData `json:"order_info"`
}

// GetSwapOrderListResponseData individual order data from GetSwapOrderList
type GetSwapOrderListResponseData struct {
	ContractVal  float64 `json:"contract_val,string"`
	Fee          float64 `json:"fee,string"`
	FilledQty    float64 `json:"filled_qty,string"`
	InstrumentID string  `json:"instrument_id"`
	Leverage     int64   `json:"leverage,string"` //  	Leverage value:10\20 default:10
	OrderID      int64   `json:"order_id,string"`
	Price        float64 `json:"price,string"`
	PriceAvg     float64 `json:"price_avg,string"`
	Size         float64 `json:"size,string"`
	Status       int64   `json:"status,string"` // Order Status （-1 canceled; 0: pending, 1: partially filled, 2: fully filled)
	Timestamp    string  `json:"timestamp"`
	Type         int64   `json:"type,string"` //  	Type (1: open long 2: open short 3: close long 4: close short)
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
	TradeID      string `json:"trade_id"`
	InstrumentID string `json:"instrument_id"`
	OrderID      string `json:"order_id"`
	Price        string `json:"price"`
	OrderQty     string `json:"order_qty"`
	Fee          string `json:"fee"`
	Timestamp    string `json:"timestamp"`
	ExecType     string `json:"exec_type"`
	Side         string `json:"side"`
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
	InstrumentID string  `json:"instrument_id"`
	Size         float64 `json:"size,string,omitempty"`
}

// GetSwapOrderBookResponse response data for GetSwapOrderBook
type GetSwapOrderBookResponse struct {
	Asks      [][]interface{} `json:"asks"` // eg [["411.3","16",5,4]] [[0: Price, 1: Size price, 2: number of force liquidated orders, 3: number of orders on the price]]
	Bids      [][]interface{} `json:"bids"` // eg [["411.3","16",5,4]] [[0: Price, 1: Size price, 2: number of force liquidated orders, 3: number of orders on the price]]
	Timestamp string          `json:"timestamp"`
}

// GetAllSwapTokensInformationResponse response data for GetAllSwapTokensInformation
type GetAllSwapTokensInformationResponse struct {
	InstrumentID string  `json:"instrument_id"`
	Last         float64 `json:"last,string"`
	High24H      float64 `json:"high_24h,string"`
	Low24H       float64 `json:"low_24h,string"`
	BestBid      float64 `json:"best_bid,string"`
	BestAsk      float64 `json:"best_ask,string"`
	Volume24H    float64 `json:"volume_24h,string"`
	Timestamp    string  `json:"timestamp"`
}

// GetSwapFilledOrdersDataRequest request data for GetSwapFilledOrdersData
type GetSwapFilledOrdersDataRequest struct {
	InstrumentID string `json:"instrument_id"`          // [required] Contract ID, e.g. “BTC-USD-SWAP
	From         int64  `json:"from,string,omitempty"`  // [optional] Request paging content for this page number.（Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	To           int64  `json:"to,string,omitempty"`    // [optional] Request page after (older) this pagination id. （Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	Limit        int64  `json:"limit,string,omitempty"` // [optional] Number of results per request. Maximum 100. (default 100)
}

// GetSwapFilledOrdersDataResponse response data for GetSwapFilledOrdersData
type GetSwapFilledOrdersDataResponse struct {
	TradeID   string  `json:"trade_id"`
	Price     float64 `json:"price,string"`
	Size      float64 `json:"size,string"`
	Side      string  `json:"side"`
	Timestamp string  `json:"timestamp"`
}

// GetSwapMarketDataRequest retrieves candle data information
type GetSwapMarketDataRequest struct {
	Start        string `json:"start,omitempty"` // [optional] start time in ISO 8601
	End          string `json:"end,omitempty"`   // [optional] end time in ISO 8601
	Granularity  int64  `json:"granularity"`     // The granularity field must be one of the following values: {60, 180, 300, 900, 1800, 3600, 7200, 14400, 43200, 86400, 604800}.
	InstrumentID string `json:"instrument_id"`   // [required] trading pairs
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
	InstrumentID string  `json:"instrument_id"`
	Index        float64 `json:"index,string"`
	Timestamp    string  `json:"timestamp"`
}

// GetSwapExchangeRatesResponse response data for GetSwapExchangeRates
type GetSwapExchangeRatesResponse struct {
	InstrumentID string  `json:"instrument_id"`
	Rate         float64 `json:"rate,string"`
	Timestamp    string  `json:"timestamp"`
}

// GetSwapOpenInterestResponse response data for GetSwapOpenInterest
type GetSwapOpenInterestResponse struct {
	InstrumentID string  `json:"instrument_id"`
	Amount       float64 `json:"amount,string"`
	Timestamp    string  `json:"timestamp"`
}

// GetSwapCurrentPriceLimitsResponse response data for GetSwapCurrentPriceLimits
type GetSwapCurrentPriceLimitsResponse struct {
	InstrumentID string  `json:"instrument_id"`
	Highest      float64 `json:"highest,string"`
	Lowest       float64 `json:"lowest,string"`
	Timestamp    string  `json:"timestamp"`
}

// GetSwapForceLiquidatedOrdersRequest request data for GetSwapForceLiquidatedOrders
type GetSwapForceLiquidatedOrdersRequest struct {
	InstrumentID string `json:"instrument_id"`          // [required] Contract ID, e.g. “BTC-USD-180213”
	From         int64  `json:"from,string,omitempty"`  // [optional] Request paging content for this page number.（Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	To           int64  `json:"to,string,omitempty"`    // [optional] Request page after (older) this pagination id. （Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	Limit        int64  `json:"limit,string,omitempty"` // [optional]  	Number of results per request. Maximum 100. (default 100)
	Status       string `json:"status,omitempty"`       // [optional] Status (0:unfilled orders in the recent 7 days 1:filled orders in the recent 7 days)
}

// GetSwapForceLiquidatedOrdersResponse response data for GetSwapForceLiquidatedOrders
type GetSwapForceLiquidatedOrdersResponse struct {
	Loss         float64 `json:"loss"`
	Size         int64   `json:"size"`
	Price        float64 `json:"price"`
	CreatedAt    string  `json:"created_at"`
	InstrumentID string  `json:"instrument_id"`
	Type         int64   `json:"type"`
}

// GetSwapOnHoldAmountForOpenOrdersResponse response data for GetSwapOnHoldAmountForOpenOrders
type GetSwapOnHoldAmountForOpenOrdersResponse struct {
	InstrumentID string  `json:"instrument_id"`
	Amount       float64 `json:"amount,string"`
	Timestamp    string  `json:"timestamp"`
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
	InstrumentID string `json:"instrument_id"`          // [required] Contract ID, e.g. “BTC-USD-SWAP
	From         int64  `json:"from,string,omitempty"`  // [optional] Request paging content for this page number.（Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	To           int64  `json:"to,string,omitempty"`    // [optional] Request page after (older) this pagination id. （Example: 1,2,3,4,5. From 4 we only have 4, to 4 we only have 3）
	Limit        int64  `json:"limit,string,omitempty"` // [optional] Number of results per request. Maximum 100.
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

// ----------------------------------------------------------------------------------------------------------------------------

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
