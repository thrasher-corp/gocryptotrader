package bitflyer

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

var errUnhandledCurrency = errors.New("unhandled currency")

// ChainAnalysisBlockResponse holds block information from the bitcoin network
type ChainAnalysisBlockResponse struct {
	BlockHash          string    `json:"block_hash"`
	Height             uint64    `json:"height"`
	IsMain             bool      `json:"is_main"`
	Version            int64     `json:"version"` // Signed 32-bit in the Bitcoin protocol
	PreviousBlock      string    `json:"prev_block"`
	MerkleRoot         string    `json:"merkle_root"`
	Timestamp          time.Time `json:"timestamp"`
	Bits               uint64    `json:"bits"`
	Nonce              uint64    `json:"nonce"`
	TransactionNumber  uint64    `json:"txnum"` // Number of transactions in the block
	TotalFeesInSatoshi uint64    `json:"total_fees"`
	TransactionHashes  []string  `json:"tx_hashes"`
}

// ChainAnalysisTransactionResponse holds transaction data from the bitcoin network
type ChainAnalysisTransactionResponse struct {
	TransactionHash string                           `json:"tx_hash"`
	BlockHeight     int64                            `json:"block_height"` // -1 when the transaction is not in a block
	Confirmed       uint64                           `json:"confirmed"`    // Number of confirmations
	FeesInSatoshi   uint64                           `json:"fees"`
	SizeInBytes     uint64                           `json:"size"`
	ReceivedDate    string                           `json:"received_date"` // UTC, without a zone designator
	Version         int64                            `json:"version"`       // Signed 32-bit in the Bitcoin protocol
	LockTime        uint64                           `json:"lock_time"`     // A block height below 500000000, otherwise a Unix time
	Inputs          []ChainAnalysisTransactionInput  `json:"inputs"`
	Outputs         []ChainAnalysisTransactionOutput `json:"outputs"`
}

// ChainAnalysisTransactionInput holds a transaction input from the bitcoin network
type ChainAnalysisTransactionInput struct {
	PreviousHash   string `json:"prev_hash"`
	PreviousIndex  int64  `json:"prev_index"` // Signed because a coinbase input reports -1
	ValueInSatoshi uint64 `json:"value"`
	Script         string `json:"script"`
	Address        string `json:"address"`
	Sequence       uint64 `json:"sequence"`
}

// ChainAnalysisTransactionOutput holds a transaction output from the bitcoin network
type ChainAnalysisTransactionOutput struct {
	ValueInSatoshi uint64 `json:"value"`
	Script         string `json:"script"`
	Address        string `json:"address"`
}

// ChainAnalysisAddressResponse holds address information from the bitcoin network
type ChainAnalysisAddressResponse struct {
	Address            string `json:"address"`
	UnconfirmedBalance uint64 `json:"unconfirmed_balance"` // Estimated balance
	ConfirmedBalance   uint64 `json:"confirmed_balance"`
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
