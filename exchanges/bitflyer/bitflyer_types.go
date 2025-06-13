package bitflyer

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var errUnhandledCurrency = errors.New("unhandled currency")

// ChainAnalysisBlock holds block information from the bitcoin network
type ChainAnalysisBlock struct {
	BlockHash     string    `json:"block_hash"`
	Height        int64     `json:"height"`
	IsMain        bool      `json:"is_main"`
	Version       float64   `json:"version"`
	PreviousBlock string    `json:"prev_block"`
	MerkleRoot    string    `json:"merkle_root"`
	Timestamp     time.Time `json:"timestamp"`
	Bits          int64     `json:"bits"`
	Nonce         int64     `json:"nonce"`
	TxNum         int64     `json:"txnum"`
	TotalFees     float64   `json:"total_fees"`
	TxHashes      []string  `json:"tx_hashes"`
}

// ChainAnalysisTransaction holds transaction data from the bitcoin network
type ChainAnalysisTransaction struct {
	TxHash        string     `json:"tx_hash"`
	BlockHeight   int64      `json:"block_height"`
	Confirmations int64      `json:"confirmed"`
	Fees          float64    `json:"fees"`
	Size          int64      `json:"size"`
	ReceivedDate  string     `json:"received_date"`
	Version       float64    `json:"version"`
	LockTime      types.Time `json:"lock_time"`
	Inputs        []struct {
		PrevHash  string `json:"prev_hash"`
		PrevIndex int    `json:"prev_index"`
		Value     int64  `json:"value"`
		Script    string `json:"script"`
		Address   string `json:"address"`
		Sequence  int64  `json:"sequence"`
	} `json:"inputs"`
	Outputs []struct {
		Value   int64  `json:"value"`
		Script  string `json:"script"`
		Address string `json:"address"`
	} `json:"outputs"`
}

// ChainAnalysisAddress holds address information from the bitcoin network
type ChainAnalysisAddress struct {
	Address            string  `json:"address"`
	UnconfirmedBalance float64 `json:"unconfirmed_balance"`
	ConfirmedBalance   float64 `json:"confirmed_balance"`
}

// MarketInfo holds market information returned from bitflyer
type MarketInfo struct {
	ProductCode string `json:"product_code"`
	Alias       string `json:"alias"`
	MarketType  string `json:"market_type"`
}

// Orderbook holds orderbook information
type Orderbook struct {
	MidPrice float64 `json:"mid_price"`
	Bids     []struct {
		Price float64 `json:"price"`
		Size  float64 `json:"size"`
	} `json:"bids"`
	Asks []struct {
		Price float64 `json:"price"`
		Size  float64 `json:"size"`
	} `json:"asks"`
}

// Ticker holds ticker information
type Ticker struct {
	ProductCode     string  `json:"product_code"`
	TimeStamp       Time    `json:"timestamp"`
	TickID          int64   `json:"tick_id"`
	BestBid         float64 `json:"best_bid"`
	BestAsk         float64 `json:"best_ask"`
	BestBidSize     float64 `json:"best_bid_size"`
	BestAskSize     float64 `json:"best_ask_size"`
	TotalBidDepth   float64 `json:"total_bid_depth"`
	TotalAskDepth   float64 `json:"total_ask_depth"`
	Last            float64 `json:"ltp"`
	Volume          float64 `json:"volume"`
	VolumeByProduct float64 `json:"volume_by_product"`
}

// ExecutedTrade holds past trade information
type ExecutedTrade struct {
	ID             int64   `json:"id"`
	Side           string  `json:"side"`
	Price          float64 `json:"price"`
	Size           float64 `json:"size"`
	ExecDate       Time    `json:"exec_date"`
	BuyAcceptedID  string  `json:"buy_child_order_acceptance_id"`
	SellAcceptedID string  `json:"sell_child_order_acceptance_id"`
}

// ChatLog holds chat log information
type ChatLog struct {
	Nickname string `json:"nickname"`
	Message  string `json:"message"`
	Date     string `json:"date"`
}

// AccountBalance holds account balance information
type AccountBalance struct {
	CurrencyCode string  `json:"currency_code"`
	Amount       float64 `json:"amount"`
	Available    float64 `json:"available"`
}

// MarginStatus holds margin status information
type MarginStatus struct {
	Collateral         float64 `json:"collateral"`
	OpenPosPNL         float64 `json:"open_position_pnl"`
	RequiredCollateral float64 `json:"require_collateral"`
	KeepRate           float64 `json:"keep_rate"`
}

// CollateralAccounts holds collateral balances
type CollateralAccounts struct {
	CurrencyCode string  `json:"currency_code"`
	Amount       float64 `json:"amount"`
}

// DepositAddress hold depositing address information
type DepositAddress struct {
	Type         string `json:"type"`
	CurrencyCode string `json:"currency_code"`
	Address      string `json:"address"`
}

// DepositHistory holds deposit history information
type DepositHistory struct {
	ID           int64   `json:"id"`
	OrderID      int64   `json:"order_id"`
	CurrencyCode string  `json:"currency_code"`
	Amount       float64 `json:"amount"`
	Address      string  `json:"address"`
	TXHash       string  `json:"tx_hash"`
	Status       string  `json:"status"`
	EventDate    string  `json:"event_date"`
}

// TransactionHistory holds prior transaction history data
type TransactionHistory struct {
	ID            int64   `json:"id"`
	OrderID       int64   `json:"order_id"`
	CurrencyCode  string  `json:"currency_code"`
	Amount        float64 `json:"amount"`
	Address       string  `json:"address"`
	TXHash        string  `json:"tx_hash"`
	Fee           float64 `json:"fee"`
	AdditionalFee float64 `json:"additional_fee"`
	Status        string  `json:"status"`
	EventDate     string  `json:"event_date"`
}

// BankAccount holds bank account information
type BankAccount struct {
	ID            int64  `json:"id"`
	IsVerified    bool   `json:"is_verified"`
	BankName      string `json:"bank_name"`
	BranchName    string `json:"branch_name"`
	AccountType   string `json:"account_type"`
	AccountNumber int    `json:"account_number"`
	AccountName   string `json:"account_name"`
}

// CashDeposit holds cash deposit information
type CashDeposit struct {
	ID           int64   `json:"id"`
	OrderID      string  `json:"order_id"`
	CurrencyCode string  `json:"currency_code"`
	Amount       float64 `json:"amount"`
	Status       string  `json:"status"`
	EventDate    string  `json:"event_date"`
}

// CancellationHistory cancellation history
type CancellationHistory struct {
	ID           int64   `json:"id"`
	OrderID      string  `json:"order_id"`
	CurrencyCode string  `json:"currency_code"`
	Amount       float64 `json:"amount"`
	Status       string  `json:"status"`
	EventDate    string  `json:"event_date"`
}

// Orders holds order full order information
type Orders struct {
	ID                     int64   `json:"id"`
	ChildOrderID           string  `json:"child_order_id"`
	ProductCode            string  `json:"product_code"`
	Side                   string  `json:"side"`
	ChildOrderType         string  `json:"child_order_type"`
	Price                  float64 `json:"price"`
	AveragePrice           float64 `json:"average_price"`
	Size                   float64 `json:"size"`
	ChildOrderState        string  `json:"child_order_state"`
	ExpireDate             string  `json:"expire_date"`
	ChildOrderDate         string  `json:"child_order_date"`
	ChildOrderAcceptanceID string  `json:"child_order_acceptance_id"`
	OutstandingSize        float64 `json:"outstanding_size"`
	CancelSize             float64 `json:"cancel_size"`
	ExecutedSize           float64 `json:"executed_size"`
	TotalCommission        float64 `json:"total_commission"`
}

// ParentOrders holds order full order information
type ParentOrders struct {
	ID                      int64   `json:"id"`
	ParentOrderID           string  `json:"parent_order_id"`
	ProductCode             string  `json:"product_code"`
	Side                    string  `json:"side"`
	ParentOrderType         string  `json:"parent_order_type"`
	Price                   float64 `json:"price"`
	AveragePrice            float64 `json:"average_price"`
	Size                    float64 `json:"size"`
	ParentOrderState        string  `json:"parent_order_state"`
	ExpireDate              string  `json:"expire_date"`
	ParentOrderDate         string  `json:"parent_order_date"`
	ParentOrderAcceptanceID string  `json:"parent_order_acceptance_id"`
	OutstandingSize         float64 `json:"outstanding_size"`
	CancelSize              float64 `json:"cancel_size"`
	ExecutedSize            float64 `json:"executed_size"`
	TotalCommission         float64 `json:"total_commission"`
}

// ParentOrderDetail holds detailed information about an order
type ParentOrderDetail struct {
	ID              int64   `json:"id"`
	ParentOrderID   string  `json:"parent_order_id"`
	OrderMethod     string  `json:"order_method"`
	MinutesToExpire float64 `json:"minute_to_expire"`
	Parameters      []struct {
		ProductCode   string  `json:"product_code"`
		ConditionType string  `json:"condition_type"`
		Side          string  `json:"side"`
		Price         float64 `json:"price"`
		Size          float64 `json:"size"`
		TriggerPrice  float64 `json:"trigger_price"`
		Offset        float64 `json:"offset"`
	} `json:"parameters"`
}

// Executions holds past executed trade details
type Executions struct {
	ID                     int64   `json:"id"`
	ChildOrderID           string  `json:"child_order_id"`
	Side                   string  `json:"side"`
	Price                  float64 `json:"price"`
	Size                   float64 `json:"size"`
	Commission             float64 `json:"commission"`
	ExecDate               string  `json:"exec_date"`
	ChildOrderAcceptanceID string  `json:"child_order_acceptance_id"`
}

// OpenInterest holds open interest information
type OpenInterest struct {
	ProductCode         string  `json:"product_code"`
	Side                string  `json:"side"`
	Price               float64 `json:"price"`
	Size                float64 `json:"size"`
	Commission          float64 `json:"commission"`
	SwapPointAccumulate float64 `json:"swap_point_accumulate"`
	RequiredCollateral  float64 `json:"require_collateral"`
	OpenDate            string  `json:"open_date"`
	Leverage            float64 `json:"leverage"`
	PNL                 float64 `json:"pnl"`
}

// CollateralHistory holds collateral history data
type CollateralHistory struct {
	ID           int64   `json:"id"`
	CurrencyCode string  `json:"currency_code"`
	Change       float64 `json:"change"`
	Amount       float64 `json:"amount"`
	Reason       string  `json:"reason_code"`
	Date         string  `json:"date"`
}

// NewOrder to send a new order
type NewOrder struct {
	ProductCode    string  `json:"product_code"`
	ChildOrderType string  `json:"child_order_type"`
	Side           string  `json:"side"`
	Price          float64 `json:"price"`
	Size           float64 `json:"size"`
	MinuteToExpire float64 `json:"minute_to_expire"`
	TimeInForce    string  `json:"time_in_force"`
}

// Time is a custom type that wraps time.Time to implement json.Unmarshaller
type Time time.Time

// UnmarshalJSON implements the json.Unmarshaller interface for Time
func (t *Time) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	parsedTime, err := time.Parse("2006-01-02T15:04:05.999999999", str)
	if err != nil {
		return err
	}
	*t = Time(parsedTime)
	return nil
}

// Time returns the time.Time representation of the DateTime type
func (t *Time) Time() time.Time { return time.Time(*t) }
