package protocol

import (
	"errors"
	"fmt"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
)

// Features holds all variables for the exchanges supported features
// for a protocol (e.g REST or Websocket)
type Features struct {
	state State
	sync.RWMutex
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

// State defines the different functionality states of the protocol
type State struct {
	ProtocolEnabled       bool  `json:"protocolEnabled"`
	AuthenticationEnabled bool  `json:"authenticationEnabled"`
	TickerBatching        *bool `json:"tickerBatching,omitempty"`
	AutoPairUpdates       *bool `json:"autoPairUpdates,omitempty"`
	AccountBalance        *bool `json:"accountBalance,omitempty"`
	CryptoDeposit         *bool `json:"cryptoDeposit,omitempty"`
	CryptoWithdrawal      *bool `json:"cryptoWithdrawal,omitempty"`
	FiatWithdraw          *bool `json:"fiatWithdraw,omitempty"`
	GetOrder              *bool `json:"getOrder,omitempty"`
	GetOrders             *bool `json:"getOrders,omitempty"`
	CancelOrders          *bool `json:"cancelOrders,omitempty"`
	CancelOrder           *bool `json:"cancelOrder,omitempty"`
	SubmitOrder           *bool `json:"submitOrder,omitempty"`
	SubmitOrders          *bool `json:"submitOrders,omitempty"`
	ModifyOrder           *bool `json:"modifyOrder,omitempty"`
	DepositHistory        *bool `json:"depositHistory,omitempty"`
	WithdrawalHistory     *bool `json:"withdrawalHistory,omitempty"`
	TradeHistory          *bool `json:"tradeHistory,omitempty"`
	UserTradeHistory      *bool `json:"userTradeHistory,omitempty"`
	TradeFee              *bool `json:"tradeFee,omitempty"`
	FiatDepositFee        *bool `json:"fiatDepositFee,omitempty"`
	FiatWithdrawalFee     *bool `json:"fiatWithdrawalFee,omitempty"`
	CryptoDepositFee      *bool `json:"cryptoDepositFee,omitempty"`
	CryptoWithdrawalFee   *bool `json:"cryptoWithdrawalFee,omitempty"`
	TickerFetching        *bool `json:"tickerFetching,omitempty"`
	KlineFetching         *bool `json:"klineFetching,omitempty"`
	TradeFetching         *bool `json:"tradeFetching,omitempty"`
	OrderbookFetching     *bool `json:"orderbookFetching,omitempty"`
	AccountInfo           *bool `json:"accountInfo,omitempty"`
	FiatDeposit           *bool `json:"fiatDeposit,omitempty"`
	DeadMansSwitch        *bool `json:"deadMansSwitch,omitempty"`
	// FullPayloadSubscribe flushes and changes full subscription on websocket
	// connection by subscribing with full default stream channel list
	FullPayloadSubscribe   *bool `json:"fullPayloadSubscribe,omitempty"`
	Subscribe              *bool `json:"subscribe,omitempty"`
	Unsubscribe            *bool `json:"unsubscribe,omitempty"`
	AuthenticatedEndpoints *bool `json:"authenticatedEndpoints,omitempty"`
	MessageCorrelation     *bool `json:"messageCorrelation,omitempty"`
	MessageSequenceNumbers *bool `json:"messageSequenceNumbers,omitempty"`
	CandleHistory          *bool `json:"candlehistory,omitempty"`
}

// Functionality returns a thread safe functionality list
func (f *Features) Functionality() (Functionality, error) {
	if f == nil {
		return Functionality{}, errors.New("features is nil")
	}
	f.RLock()
	defer f.RUnlock()

	return Functionality{
		ProtocolEnabled:        f.protocolEnabled,
		AuthenticationEnabled:  f.authenticationEnabled,
		TickerBatching:         isFunctional(f.tickerBatching),
		AutoPairUpdates:        isFunctional(f.autoPairUpdates),
		AccountBalance:         isFunctional(f.accountBalance),
		CryptoDeposit:          isFunctional(f.cryptoDeposit),
		CryptoWithdrawal:       isFunctional(f.cryptoWithdrawal),
		FiatWithdraw:           isFunctional(f.fiatWithdraw),
		GetOrder:               isFunctional(f.getOrder),
		GetOrders:              isFunctional(f.getOrders),
		CancelOrders:           isFunctional(f.cancelOrders),
		CancelOrder:            isFunctional(f.cancelOrder),
		SubmitOrder:            isFunctional(f.submitOrder),
		SubmitOrders:           isFunctional(f.submitOrders),
		ModifyOrder:            isFunctional(f.modifyOrder),
		DepositHistory:         isFunctional(f.depositHistory),
		WithdrawalHistory:      isFunctional(f.withdrawalHistory),
		TradeHistory:           isFunctional(f.tradeHistory),
		UserTradeHistory:       isFunctional(f.userTradeHistory),
		TradeFee:               isFunctional(f.tradeFee),
		FiatDepositFee:         isFunctional(f.fiatDepositFee),
		FiatWithdrawalFee:      isFunctional(f.fiatWithdrawalFee),
		CryptoDepositFee:       isFunctional(f.cryptoDepositFee),
		CryptoWithdrawalFee:    isFunctional(f.cryptoWithdrawalFee),
		TickerFetching:         isFunctional(f.tickerFetching),
		KlineFetching:          isFunctional(f.klineFetching),
		TradeFetching:          isFunctional(f.tradeFetching),
		OrderbookFetching:      isFunctional(f.orderbookFetching),
		AccountInfo:            isFunctional(f.accountInfo),
		FiatDeposit:            isFunctional(f.fiatDeposit),
		DeadMansSwitch:         isFunctional(f.deadMansSwitch),
		FullPayloadSubscribe:   isFunctional(f.fullPayloadSubscribe),
		Subscribe:              isFunctional(f.subscribe),
		Unsubscribe:            isFunctional(f.unsubscribe),
		AuthenticatedEndpoints: isFunctional(f.authenticatedEndpoints),
		MessageCorrelation:     isFunctional(f.messageCorrelation),
		MessageSequenceNumbers: isFunctional(f.messageSequenceNumbers),
		CandleHistory:          isFunctional(f.candleHistory),
	}, nil
}

func isFunctional(ptr *bool) bool {
	return ptr != nil && *ptr
}

// Supported returns a thread safe supported list
func (f *Features) Supported() (Functionality, error) {
	if f == nil {
		return Functionality{}, errors.New("features is nil")
	}
	f.RLock()
	defer f.RUnlock()

	return Functionality{
		ProtocolEnabled:        f.protocolEnabled,
		AuthenticationEnabled:  f.authenticationEnabled,
		TickerBatching:         isSupported(f.tickerBatching),
		AutoPairUpdates:        isSupported(f.autoPairUpdates),
		AccountBalance:         isSupported(f.accountBalance),
		CryptoDeposit:          isSupported(f.cryptoDeposit),
		CryptoWithdrawal:       isSupported(f.cryptoWithdrawal),
		FiatWithdraw:           isSupported(f.fiatWithdraw),
		GetOrder:               isSupported(f.getOrder),
		GetOrders:              isSupported(f.getOrders),
		CancelOrders:           isSupported(f.cancelOrders),
		CancelOrder:            isSupported(f.cancelOrder),
		SubmitOrder:            isSupported(f.submitOrder),
		SubmitOrders:           isSupported(f.submitOrders),
		ModifyOrder:            isSupported(f.modifyOrder),
		DepositHistory:         isSupported(f.depositHistory),
		WithdrawalHistory:      isSupported(f.withdrawalHistory),
		TradeHistory:           isSupported(f.tradeHistory),
		UserTradeHistory:       isSupported(f.userTradeHistory),
		TradeFee:               isSupported(f.tradeFee),
		FiatDepositFee:         isSupported(f.fiatDepositFee),
		FiatWithdrawalFee:      isSupported(f.fiatWithdrawalFee),
		CryptoDepositFee:       isSupported(f.cryptoDepositFee),
		CryptoWithdrawalFee:    isSupported(f.cryptoWithdrawalFee),
		TickerFetching:         isSupported(f.tickerFetching),
		KlineFetching:          isSupported(f.klineFetching),
		TradeFetching:          isSupported(f.tradeFetching),
		OrderbookFetching:      isSupported(f.orderbookFetching),
		AccountInfo:            isSupported(f.accountInfo),
		FiatDeposit:            isSupported(f.fiatDeposit),
		DeadMansSwitch:         isSupported(f.deadMansSwitch),
		FullPayloadSubscribe:   isSupported(f.fullPayloadSubscribe),
		Subscribe:              isSupported(f.subscribe),
		Unsubscribe:            isSupported(f.unsubscribe),
		AuthenticatedEndpoints: isSupported(f.authenticatedEndpoints),
		MessageCorrelation:     isSupported(f.messageCorrelation),
		MessageSequenceNumbers: isSupported(f.messageSequenceNumbers),
		CandleHistory:          isSupported(f.candleHistory),
	}, nil
}

func isSupported(functionality *bool) bool {
	return functionality != nil
}

func functionalityTranslate(feature, newstate *bool) error {
	if feature == nil {
		if newstate != nil {
			return errors.New("functionality not present in protocol features")
		}
		return nil
	}
	if *newstate {
		feature = convert.BoolPtr(true)
	} else {
		feature = convert.BoolPtr(false)
	}
	return nil
}

// SetFunctionality changes feature state
func (f *Features) SetFunctionality(newState State) error {
	if f == nil {
		return errors.New("features is nil")
	}
	f.Lock()
	defer f.Unlock()

	err := functionalityTranslate(f.tickerBatching, newState.TickerBatching)
	if err != nil {
		return fmt.Errorf("ticker batching error: %s", err)
	}

	err = functionalityTranslate(f.autoPairUpdates, newState.AutoPairUpdates)
	if err != nil {
		return fmt.Errorf("AutoPairUpdates error: %s", err)
	}

	err = functionalityTranslate(f.accountBalance, newState.AccountBalance)
	if err != nil {
		return fmt.Errorf("AccountBalance error: %s", err)
	}

	err = functionalityTranslate(f.cryptoDeposit, newState.CryptoDeposit)
	if err != nil {
		return fmt.Errorf("CryptoDeposit error: %s", err)
	}

	err = functionalityTranslate(f.cryptoWithdrawal, newState.CryptoWithdrawal)
	if err != nil {
		return fmt.Errorf("CryptoWithdrawal error: %s", err)
	}

	err = functionalityTranslate(f.fiatWithdraw, newState.FiatWithdraw)
	if err != nil {
		return fmt.Errorf("FiatWithdraw error: %s", err)
	}

	err = functionalityTranslate(f.getOrder, newState.GetOrder)
	if err != nil {
		return fmt.Errorf("GetOrder error: %s", err)
	}

	err = functionalityTranslate(f.getOrders, newState.GetOrders)
	if err != nil {
		return fmt.Errorf("GetOrders error: %s", err)
	}

	err = functionalityTranslate(f.cancelOrders, newState.CancelOrders)
	if err != nil {
		return fmt.Errorf("CancelOrders error: %s", err)
	}

	err = functionalityTranslate(f.cancelOrder, newState.CancelOrder)
	if err != nil {
		return fmt.Errorf("CancelOrder error: %s", err)
	}

	err = functionalityTranslate(f.submitOrder, newState.SubmitOrder)
	if err != nil {
		return fmt.Errorf("SubmitOrder error: %s", err)
	}

	err = functionalityTranslate(f.submitOrders, newState.SubmitOrders)
	if err != nil {
		return fmt.Errorf("SubmitOrders error: %s", err)
	}

	err = functionalityTranslate(f.modifyOrder, newState.ModifyOrder)
	if err != nil {
		return fmt.Errorf("ModifyOrder error: %s", err)
	}

	err = functionalityTranslate(f.depositHistory, newState.DepositHistory)
	if err != nil {
		return fmt.Errorf("DepositHistory error: %s", err)
	}

	err = functionalityTranslate(f.withdrawalHistory, newState.WithdrawalHistory)
	if err != nil {
		return fmt.Errorf("WithdrawalHistory error: %s", err)
	}

	err = functionalityTranslate(f.tradeHistory, newState.TradeHistory)
	if err != nil {
		return fmt.Errorf("TradeHistory error: %s", err)
	}

	err = functionalityTranslate(f.userTradeHistory, newState.UserTradeHistory)
	if err != nil {
		return fmt.Errorf("UserTradeHistory error: %s", err)
	}

	err = functionalityTranslate(f.tradeFee, newState.TradeFee)
	if err != nil {
		return fmt.Errorf("TradeFee error: %s", err)
	}

	err = functionalityTranslate(f.fiatDepositFee, newState.FiatDepositFee)
	if err != nil {
		return fmt.Errorf("FiatDepositFee error: %s", err)
	}

	err = functionalityTranslate(f.fiatWithdrawalFee, newState.FiatWithdrawalFee)
	if err != nil {
		return fmt.Errorf("FiatWithdrawalFee error: %s", err)
	}

	err = functionalityTranslate(f.cryptoDepositFee, newState.CryptoDepositFee)
	if err != nil {
		return fmt.Errorf("CryptoDepositFee error: %s", err)
	}

	err = functionalityTranslate(f.cryptoWithdrawalFee, newState.CryptoWithdrawalFee)
	if err != nil {
		return fmt.Errorf("CryptoWithdrawalFee error: %s", err)
	}

	err = functionalityTranslate(f.tickerFetching, newState.TickerFetching)
	if err != nil {
		return fmt.Errorf("TickerFetching error: %s", err)
	}

	err = functionalityTranslate(f.klineFetching, newState.KlineFetching)
	if err != nil {
		return fmt.Errorf("KlineFetching error: %s", err)
	}

	err = functionalityTranslate(f.tradeFetching, newState.TradeFetching)
	if err != nil {
		return fmt.Errorf("TradeFetching error: %s", err)
	}

	err = functionalityTranslate(f.orderbookFetching, newState.OrderbookFetching)
	if err != nil {
		return fmt.Errorf("OrderbookFetching error: %s", err)
	}

	err = functionalityTranslate(f.accountInfo, newState.AccountInfo)
	if err != nil {
		return fmt.Errorf("AccountInfo error: %s", err)
	}

	err = functionalityTranslate(f.fiatDeposit, newState.FiatDeposit)
	if err != nil {
		return fmt.Errorf("FiatDeposit error: %s", err)
	}

	err = functionalityTranslate(f.deadMansSwitch, newState.DeadMansSwitch)
	if err != nil {
		return fmt.Errorf("DeadMansSwitch error: %s", err)
	}

	err = functionalityTranslate(f.fullPayloadSubscribe, newState.FullPayloadSubscribe)
	if err != nil {
		return fmt.Errorf("FullPayloadSubscribe error: %s", err)
	}

	err = functionalityTranslate(f.subscribe, newState.Subscribe)
	if err != nil {
		return fmt.Errorf("Subscribe error: %s", err)
	}

	err = functionalityTranslate(f.unsubscribe, newState.Unsubscribe)
	if err != nil {
		return fmt.Errorf("Unsubscribe error: %s", err)
	}

	err = functionalityTranslate(f.authenticatedEndpoints, newState.AuthenticatedEndpoints)
	if err != nil {
		return fmt.Errorf("AuthenticatedEndpoints error: %s", err)
	}

	err = functionalityTranslate(f.messageCorrelation, newState.MessageCorrelation)
	if err != nil {
		return fmt.Errorf("MessageCorrelation error: %s", err)
	}

	err = functionalityTranslate(f.messageSequenceNumbers, newState.MessageSequenceNumbers)
	if err != nil {
		return fmt.Errorf("MessageSequenceNumbers error: %s", err)
	}

	err = functionalityTranslate(f.candleHistory, newState.CandleHistory)
	if err != nil {
		return fmt.Errorf("CandleHistory error: %s", err)
	}
	return nil
}
