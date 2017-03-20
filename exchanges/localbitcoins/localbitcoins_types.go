package localbitcoins

import (
	"time"
)

type LocalBitcoinsTicker struct {
	Avg12h float64 `json:"avg_12h,string"`
	Avg1h  float64 `json:"avg_1h,string"`
	Avg24h float64 `json:"avg_24h,string"`
	Rates  struct {
		Last float64 `json:"last,string"`
	} `json:"rates"`
	VolumeBTC float64 `json:"volume_btc,string"`
}

type LocalBitcoinsTrade struct {
	TID    int64   `json:"tid"`
	Date   int64   `json:"date"`
	Amount float64 `json:"amount,string"`
	Price  float64 `json:"price,string"`
}

type LocalBitcoinsOrderbookStructure struct {
	Price  float64
	Amount float64
}

type LocalBitcoinsOrderbook struct {
	Bids []LocalBitcoinsOrderbookStructure `json:"bids"`
	Asks []LocalBitcoinsOrderbookStructure `json:"asks"`
}

type LocalBitcoinsAccountInfo struct {
	Username             string    `json:"username"`
	CreatedAt            time.Time `json:"created_at"`
	AgeText              string    `json:"age_text"`
	TradingPartners      int       `json:"trading_partners_count"`
	FeedbacksUnconfirmed int       `json:"feedbacks_unconfirmed_count"`
	TradeVolumeText      string    `json:"trade_volume_text"`
	HasCommonTrades      bool      `json:"has_common_trades"`
	HasFeedback          bool      `json:"has_feedback"`
	ConfirmedTradesText  string    `json:"confirmed_trade_count_text"`
	BlockedCount         int       `json:"blocked_count"`
	FeedbackScore        int       `json:"feedback_score"`
	FeedbackCount        int       `json:"feedback_count"`
	URL                  string    `json:"url"`
	TrustedCount         int       `json:"trusted_count"`
	IdentityVerifiedAt   time.Time `json:"identify_verified_at"`
}

type LocalBitcoinsBalance struct {
	Balance  float64 `json:"balance,string"`
	Sendable float64 `json:"Sendable,string"`
}

type LocalBitcoinsWalletTransaction struct {
	TXID        string    `json:"txid"`
	Amount      float64   `json:"amount,string"`
	Description string    `json:"description"`
	TXType      int       `json:"tx_type"`
	CreatedAt   time.Time `json:"created_at"`
}

type LocalBitcoinsWalletAddressList struct {
	Address  string  `json:"address"`
	Received float64 `json:"received,string"`
}

type LocalBitcoinsWalletInfo struct {
	Message                 string                           `json:"message"`
	Total                   LocalBitcoinsBalance             `json:"total"`
	SentTransactions30d     []LocalBitcoinsWalletTransaction `json:"sent_transactions_30d"`
	ReceivedTransactions30d []LocalBitcoinsWalletTransaction `json:"received_transactions_30d"`
	ReceivingAddressCount   int                              `json:"receiving_address_count"`
	ReceivingAddressList    []LocalBitcoinsWalletAddressList `json:"receiving_address_list"`
}

type LocalBitcoinsWalletBalanceInfo struct {
	Message               string                           `json:"message"`
	Total                 LocalBitcoinsBalance             `json:"total"`
	ReceivingAddressCount int                              `json:"receiving_address_count"` // always 1
	ReceivingAddressList  []LocalBitcoinsWalletAddressList `json:"receiving_address_list"`
}
