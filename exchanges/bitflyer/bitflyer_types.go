package bitflyer

import "github.com/thrasher-/gocryptotrader/decimal"

// ChainAnalysisBlock holds block information from the bitcoin network
type ChainAnalysisBlock struct {
	BlockHash     string          `json:"block_hash"`
	Height        int64           `json:"height"`
	IsMain        bool            `json:"is_main"`
	Version       float64         `json:"version"`
	PreviousBlock string          `json:"prev_block"`
	MerkleRoot    string          `json:"merkle_root"`
	Timestamp     string          `json:"timestamp"`
	Bits          int64           `json:"bits"`
	Nonce         int64           `json:"nonce"`
	TxNum         int64           `json:"txnum"`
	TotalFees     decimal.Decimal `json:"total_fees"`
	TxHashes      []string        `json:"tx_hashes"`
}

// ChainAnalysisTransaction holds transaction data from the bitcoin network
type ChainAnalysisTransaction struct {
	TxHash        string          `json:"tx_hash"`
	BlockHeight   int64           `json:"block_height"`
	Confirmations int64           `json:"confirmed"`
	Fees          decimal.Decimal `json:"fees"`
	Size          int64           `json:"size"`
	ReceivedDate  string          `json:"received_date"`
	Version       float64         `json:"version"`
	LockTime      int64           `json:"lock_time"`
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
	Address            string          `json:"address"`
	UnconfirmedBalance decimal.Decimal `json:"unconfirmed_balance"`
	ConfirmedBalance   decimal.Decimal `json:"confirmed_balance"`
}

// MarketInfo holds market information returned from bitflyer
type MarketInfo struct {
	ProductCode string `json:"product_code"`
	Alias       string `json:"alias"`
}

// Orderbook holds orderbook information
type Orderbook struct {
	MidPrice decimal.Decimal `json:"mid_price"`
	Bids     []struct {
		Price decimal.Decimal `json:"price"`
		Size  decimal.Decimal `json:"size"`
	} `json:"bids"`
	Asks []struct {
		Price decimal.Decimal `json:"price"`
		Size  decimal.Decimal `json:"size"`
	} `json:"asks"`
}

// Ticker holds ticker information
type Ticker struct {
	ProductCode     string          `json:"product_code"`
	TimeStamp       string          `json:"timestamp"`
	TickID          int64           `json:"tick_id"`
	BestBid         decimal.Decimal `json:"best_bid"`
	BestAsk         decimal.Decimal `json:"best_ask"`
	BestBidSize     decimal.Decimal `json:"best_bid_size"`
	BestAskSize     decimal.Decimal `json:"best_ask_size"`
	TotalBidDepth   decimal.Decimal `json:"total_bid_depth"`
	TotalAskDepth   decimal.Decimal `json:"total_ask_depth"`
	Last            decimal.Decimal `json:"ltp"`
	Volume          decimal.Decimal `json:"volume"`
	VolumeByProduct decimal.Decimal `json:"volume_by_product"`
}

// ExecutedTrade holds past trade information
type ExecutedTrade struct {
	ID             int64           `json:"id"`
	Side           string          `json:"side"`
	Price          decimal.Decimal `json:"price"`
	Size           decimal.Decimal `json:"size"`
	ExecDate       string          `json:"exec_date"`
	BuyAcceptedID  string          `json:"buy_child_order_acceptance_id"`
	SellAcceptedID string          `json:"sell_child_order_acceptance_id"`
}

// ChatLog holds chat log information
type ChatLog struct {
	Nickname string `json:"nickname"`
	Message  string `json:"message"`
	Date     string `json:"date"`
}

// AccountBalance holds account balance information
type AccountBalance struct {
	CurrencyCode string          `json:"currency_code"`
	Amount       decimal.Decimal `json:"amount"`
	Available    decimal.Decimal `json:"available"`
}

// MarginStatus holds margin status information
type MarginStatus struct {
	Collateral         decimal.Decimal `json:"collateral"`
	OpenPosPNL         decimal.Decimal `json:"open_position_pnl"`
	RequiredCollateral decimal.Decimal `json:"require_collateral"`
	KeepRate           decimal.Decimal `json:"keep_rate"`
}

// CollateralAccounts holds collateral balances
type CollateralAccounts struct {
	CurrencyCode string          `json:"currency_code"`
	Amount       decimal.Decimal `json:"amount"`
}

// DepositAddress hold depositing address information
type DepositAddress struct {
	Type         string `json:"type"`
	CurrencyCode string `json:"currency_code"`
	Address      string `json:"address"`
}

// DepositHistory holds deposit history information
type DepositHistory struct {
	ID           int64           `json:"id"`
	OrderID      int64           `json:"order_id"`
	CurrencyCode string          `json:"currency_code"`
	Amount       decimal.Decimal `json:"amount"`
	Address      string          `json:"address"`
	TXHash       string          `json:"tx_hash"`
	Status       string          `json:"status"`
	EventDate    string          `json:"event_date"`
}

// TransactionHistory holds prior transaction history data
type TransactionHistory struct {
	ID            int64           `json:"id"`
	OrderID       int64           `json:"order_id"`
	CurrencyCode  string          `json:"currency_code"`
	Amount        decimal.Decimal `json:"amount"`
	Address       string          `json:"address"`
	TXHash        string          `json:"tx_hash"`
	Fee           decimal.Decimal `json:"fee"`
	AdditionalFee decimal.Decimal `json:"additional_fee"`
	Status        string          `json:"status"`
	EventDate     string          `json:"event_date"`
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
	ID           int64           `json:"id"`
	OrderID      string          `json:"order_id"`
	CurrencyCode string          `json:"currency_code"`
	Amount       decimal.Decimal `json:"amount"`
	Status       string          `json:"status"`
	EventDate    string          `json:"event_date"`
}

// CancellationHistory cancellation history
type CancellationHistory struct {
	ID           int64           `json:"id"`
	OrderID      string          `json:"order_id"`
	CurrencyCode string          `json:"currency_code"`
	Amount       decimal.Decimal `json:"amount"`
	Status       string          `json:"status"`
	EventDate    string          `json:"event_date"`
}

// Orders holds order full order information
type Orders struct {
	ID                     int64           `json:"id"`
	ChildOrderID           string          `json:"child_order_id"`
	ProductCode            string          `json:"product_code"`
	Side                   string          `json:"side"`
	ChildOrderType         string          `json:"child_order_type"`
	Price                  decimal.Decimal `json:"price"`
	AveragePrice           decimal.Decimal `json:"average_price"`
	Size                   decimal.Decimal `json:"size"`
	ChildOrderState        string          `json:"child_order_state"`
	ExpireDate             string          `json:"expire_date"`
	ChildOrderDate         string          `json:"child_order_date"`
	ChildOrderAcceptanceID string          `json:"child_order_acceptance_id"`
	OutstandingSize        decimal.Decimal `json:"outstanding_size"`
	CancelSize             decimal.Decimal `json:"cancel_size"`
	ExecutedSize           decimal.Decimal `json:"executed_size"`
	TotalCommission        decimal.Decimal `json:"total_commission"`
}

// ParentOrders holds order full order information
type ParentOrders struct {
	ID                      int64           `json:"id"`
	ParentOrderID           string          `json:"parent_order_id"`
	ProductCode             string          `json:"product_code"`
	Side                    string          `json:"side"`
	ParentOrderType         string          `json:"parent_order_type"`
	Price                   decimal.Decimal `json:"price"`
	AveragePrice            decimal.Decimal `json:"average_price"`
	Size                    decimal.Decimal `json:"size"`
	ParentOrderState        string          `json:"parent_order_state"`
	ExpireDate              string          `json:"expire_date"`
	ParentOrderDate         string          `json:"parent_order_date"`
	ParentOrderAcceptanceID string          `json:"parent_order_acceptance_id"`
	OutstandingSize         decimal.Decimal `json:"outstanding_size"`
	CancelSize              decimal.Decimal `json:"cancel_size"`
	ExecutedSize            decimal.Decimal `json:"executed_size"`
	TotalCommission         decimal.Decimal `json:"total_commission"`
}

// ParentOrderDetail holds detailed information about an order
type ParentOrderDetail struct {
	ID              int64           `json:"id"`
	ParentOrderID   string          `json:"parent_order_id"`
	OrderMethod     string          `json:"order_method"`
	MinutesToExpire decimal.Decimal `json:"minute_to_expire"`
	Parameters      []struct {
		ProductCode   string          `json:"product_code"`
		ConditionType string          `json:"condition_type"`
		Side          string          `json:"side"`
		Price         decimal.Decimal `json:"price"`
		Size          decimal.Decimal `json:"size"`
		TriggerPrice  decimal.Decimal `json:"trigger_price"`
		Offset        decimal.Decimal `json:"offset"`
	} `json:"parameters"`
}

// Executions holds past executed trade details
type Executions struct {
	ID                     int64           `json:"id"`
	ChildOrderID           string          `json:"child_order_id"`
	Side                   string          `json:"side"`
	Price                  decimal.Decimal `json:"price"`
	Size                   decimal.Decimal `json:"size"`
	Commission             decimal.Decimal `json:"commission"`
	ExecDate               string          `json:"exec_date"`
	ChildOrderAcceptanceID string          `json:"child_order_acceptance_id"`
}

// OpenInterest holds open interest information
type OpenInterest struct {
	ProductCode         string          `json:"product_code"`
	Side                string          `json:"side"`
	Price               decimal.Decimal `json:"price"`
	Size                decimal.Decimal `json:"size"`
	Commission          decimal.Decimal `json:"commission"`
	SwapPointAccumulate decimal.Decimal `json:"swap_point_accumulate"`
	RequiredCollateral  decimal.Decimal `json:"require_collateral"`
	OpenDate            string          `json:"open_date"`
	Leverage            decimal.Decimal `json:"leverage"`
	PNL                 decimal.Decimal `json:"pnl"`
}

// CollateralHistory holds collateral history data
type CollateralHistory struct {
	ID           int64           `json:"id"`
	CurrencyCode string          `json:"currency_code"`
	Change       decimal.Decimal `json:"change"`
	Amount       decimal.Decimal `json:"amount"`
	Reason       string          `json:"reason_code"`
	Date         string          `json:"date"`
}
