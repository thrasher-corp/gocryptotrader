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
