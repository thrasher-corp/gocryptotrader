package protocol

import (
	"sync"

	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// Features holds all variables for the exchanges supported features for a
// protocol (e.g REST or Websocket)
type Features struct {
	Protocols
	withdrawalPermissions   uint32
	autoPairUpdate          *bool
	klineSupportedIntervals map[kline.Interval]bool
	sync.RWMutex
}

// Protocols define a namespace that segregates from the features mutex
type Protocols struct {
	REST      *State `json:"rest,omitempty"`
	Websocket *State `json:"websocket,omitempty"`
	Shared    Shared `json:"sharedValues,omitempty"`
}

// Shared defines the common functionality across all protocol instances
type Shared struct {
	WithdrawPermissions uint32 `json:"withdrawPermissions,omitempty"`
}

// Functionality defines thread safe protocol ability whether on or off.
type Functionality struct {
	ProtocolEnabled        bool
	AuthenticationEnabled  bool
	TickerBatching         bool
	AutoPairUpdates        bool
	AccountBalance         bool
	CryptoDeposit          bool
	CryptoWithdrawal       bool
	FiatWithdraw           bool
	GetOrder               bool
	GetOrders              bool
	CancelOrders           bool
	CancelOrder            bool
	SubmitOrder            bool
	SubmitOrders           bool
	ModifyOrder            bool
	DepositHistory         bool
	WithdrawalHistory      bool
	TradeHistory           bool
	UserTradeHistory       bool
	TradeFee               bool
	FiatDepositFee         bool
	FiatWithdrawalFee      bool
	CryptoDepositFee       bool
	CryptoWithdrawalFee    bool
	TickerFetching         bool
	KlineFetching          bool
	TradeFetching          bool
	OrderbookFetching      bool
	AccountInfo            bool
	FiatDeposit            bool
	DeadMansSwitch         bool
	FullPayloadSubscribe   bool
	Subscribe              bool
	Unsubscribe            bool
	AuthenticatedEndpoints bool
	MessageCorrelation     bool
	MessageSequenceNumbers bool
	CandleHistory          bool
}

// Component defines the functionality type
type Component int

// Different exported component types
const (
	TickerBatching Component = iota
	AutoPairUpdates
	AccountBalance
	CryptoDeposit
	CryptoWithdrawal
	FiatWithdraw
	GetOrder
	GetOrders
	CancelOrders
	CancelOrder
	SubmitOrder
	SubmitOrders
	ModifyOrder
	DepositHistory
	WithdrawalHistory
	TradeHistory
	UserTradeHistory
	TradeFee
	FiatDepositFee
	FiatWithdrawalFee
	CryptoDepositFee
	CryptoWithdrawalFee
	TickerFetching
	KlineFetching
	TradeFetching
	OrderbookFetching
	AccountInfo
	FiatDeposit
	DeadMansSwitch
	FullPayloadSubscribe
	Subscribe
	Unsubscribe
	AuthenticatedEndpoints
	MessageCorrelation
	MessageSequenceNumbers
	CandleHistory
)

var components = []Component{
	TickerBatching,
	AutoPairUpdates,
	AccountBalance,
	CryptoDeposit,
	CryptoWithdrawal,
	FiatWithdraw,
	GetOrder,
	GetOrders,
	CancelOrders,
	CancelOrder,
	SubmitOrder,
	SubmitOrders,
	ModifyOrder,
	DepositHistory,
	WithdrawalHistory,
	TradeHistory,
	UserTradeHistory,
	TradeFee,
	FiatDepositFee,
	FiatWithdrawalFee,
	CryptoDepositFee,
	CryptoWithdrawalFee,
	TickerFetching,
	KlineFetching,
	TradeFetching,
	OrderbookFetching,
	AccountInfo,
	FiatDeposit,
	DeadMansSwitch,
	FullPayloadSubscribe,
	Subscribe,
	Unsubscribe,
	AuthenticatedEndpoints,
	MessageCorrelation,
	MessageSequenceNumbers,
	CandleHistory,
}
