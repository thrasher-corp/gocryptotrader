package protocol

import (
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
)

var errStateIsNil = errors.New("protocol state is nil")

// Functionality returns a thread safe functionality list
func (s *State) Functionality() (Functionality, error) {
	if s == nil {
		return Functionality{}, errStateIsNil
	}
	return Functionality{
		ProtocolEnabled:        s.ProtocolEnabled,
		AuthenticationEnabled:  s.AuthenticationEnabled,
		TickerBatching:         isEnabled(s.TickerBatching),
		AutoPairUpdates:        isEnabled(s.AutoPairUpdates),
		AccountBalance:         isEnabled(s.AccountBalance),
		CryptoDeposit:          isEnabled(s.CryptoDeposit),
		CryptoWithdrawal:       isEnabled(s.CryptoWithdrawal),
		FiatWithdraw:           isEnabled(s.FiatWithdraw),
		GetOrder:               isEnabled(s.GetOrder),
		GetOrders:              isEnabled(s.GetOrders),
		CancelOrders:           isEnabled(s.CancelOrders),
		CancelOrder:            isEnabled(s.CancelOrder),
		SubmitOrder:            isEnabled(s.SubmitOrder),
		SubmitOrders:           isEnabled(s.SubmitOrders),
		ModifyOrder:            isEnabled(s.ModifyOrder),
		DepositHistory:         isEnabled(s.DepositHistory),
		WithdrawalHistory:      isEnabled(s.WithdrawalHistory),
		TradeHistory:           isEnabled(s.TradeHistory),
		UserTradeHistory:       isEnabled(s.UserTradeHistory),
		TradeFee:               isEnabled(s.TradeFee),
		FiatDepositFee:         isEnabled(s.FiatDepositFee),
		FiatWithdrawalFee:      isEnabled(s.FiatWithdrawalFee),
		CryptoDepositFee:       isEnabled(s.CryptoDepositFee),
		CryptoWithdrawalFee:    isEnabled(s.CryptoWithdrawalFee),
		TickerFetching:         isEnabled(s.TickerFetching),
		KlineFetching:          isEnabled(s.KlineFetching),
		TradeFetching:          isEnabled(s.TradeFetching),
		OrderbookFetching:      isEnabled(s.OrderbookFetching),
		AccountInfo:            isEnabled(s.AccountInfo),
		FiatDeposit:            isEnabled(s.FiatDeposit),
		DeadMansSwitch:         isEnabled(s.DeadMansSwitch),
		FullPayloadSubscribe:   isEnabled(s.FullPayloadSubscribe),
		Subscribe:              isEnabled(s.Subscribe),
		Unsubscribe:            isEnabled(s.Unsubscribe),
		AuthenticatedEndpoints: isEnabled(s.AuthenticatedEndpoints),
		MessageCorrelation:     isEnabled(s.MessageCorrelation),
		MessageSequenceNumbers: isEnabled(s.MessageSequenceNumbers),
		CandleHistory:          isEnabled(s.CandleHistory),
	}, nil
}

// Supported returns a thread safe supported list
func (s *State) Supported() (Functionality, error) {
	if s == nil {
		return Functionality{}, errStateIsNil
	}
	return Functionality{
		ProtocolEnabled:        s.ProtocolEnabled,
		AuthenticationEnabled:  s.AuthenticationEnabled,
		TickerBatching:         isSupported(s.TickerBatching),
		AutoPairUpdates:        isSupported(s.AutoPairUpdates),
		AccountBalance:         isSupported(s.AccountBalance),
		CryptoDeposit:          isSupported(s.CryptoDeposit),
		CryptoWithdrawal:       isSupported(s.CryptoWithdrawal),
		FiatWithdraw:           isSupported(s.FiatWithdraw),
		GetOrder:               isSupported(s.GetOrder),
		GetOrders:              isSupported(s.GetOrders),
		CancelOrders:           isSupported(s.CancelOrders),
		CancelOrder:            isSupported(s.CancelOrder),
		SubmitOrder:            isSupported(s.SubmitOrder),
		SubmitOrders:           isSupported(s.SubmitOrders),
		ModifyOrder:            isSupported(s.ModifyOrder),
		DepositHistory:         isSupported(s.DepositHistory),
		WithdrawalHistory:      isSupported(s.WithdrawalHistory),
		TradeHistory:           isSupported(s.TradeHistory),
		UserTradeHistory:       isSupported(s.UserTradeHistory),
		TradeFee:               isSupported(s.TradeFee),
		FiatDepositFee:         isSupported(s.FiatDepositFee),
		FiatWithdrawalFee:      isSupported(s.FiatWithdrawalFee),
		CryptoDepositFee:       isSupported(s.CryptoDepositFee),
		CryptoWithdrawalFee:    isSupported(s.CryptoWithdrawalFee),
		TickerFetching:         isSupported(s.TickerFetching),
		KlineFetching:          isSupported(s.KlineFetching),
		TradeFetching:          isSupported(s.TradeFetching),
		OrderbookFetching:      isSupported(s.OrderbookFetching),
		AccountInfo:            isSupported(s.AccountInfo),
		FiatDeposit:            isSupported(s.FiatDeposit),
		DeadMansSwitch:         isSupported(s.DeadMansSwitch),
		FullPayloadSubscribe:   isSupported(s.FullPayloadSubscribe),
		Subscribe:              isSupported(s.Subscribe),
		Unsubscribe:            isSupported(s.Unsubscribe),
		AuthenticatedEndpoints: isSupported(s.AuthenticatedEndpoints),
		MessageCorrelation:     isSupported(s.MessageCorrelation),
		MessageSequenceNumbers: isSupported(s.MessageSequenceNumbers),
		CandleHistory:          isSupported(s.CandleHistory),
	}, nil
}

// SetFunctionality changes feature state
func (s *State) SetFunctionality(newState State) error {
	if s == nil {
		return errStateIsNil
	}

	err := functionalityTranslate(s.TickerBatching, newState.TickerBatching)
	if err != nil {
		return fmt.Errorf("ticker batching error: %s", err)
	}

	err = functionalityTranslate(s.AutoPairUpdates, newState.AutoPairUpdates)
	if err != nil {
		return fmt.Errorf("AutoPairUpdates error: %s", err)
	}

	err = functionalityTranslate(s.AccountBalance, newState.AccountBalance)
	if err != nil {
		return fmt.Errorf("AccountBalance error: %s", err)
	}

	err = functionalityTranslate(s.CryptoDeposit, newState.CryptoDeposit)
	if err != nil {
		return fmt.Errorf("CryptoDeposit error: %s", err)
	}

	err = functionalityTranslate(s.CryptoWithdrawal, newState.CryptoWithdrawal)
	if err != nil {
		return fmt.Errorf("CryptoWithdrawal error: %s", err)
	}

	err = functionalityTranslate(s.FiatWithdraw, newState.FiatWithdraw)
	if err != nil {
		return fmt.Errorf("FiatWithdraw error: %s", err)
	}

	err = functionalityTranslate(s.GetOrder, newState.GetOrder)
	if err != nil {
		return fmt.Errorf("GetOrder error: %s", err)
	}

	err = functionalityTranslate(s.GetOrders, newState.GetOrders)
	if err != nil {
		return fmt.Errorf("GetOrders error: %s", err)
	}

	err = functionalityTranslate(s.CancelOrders, newState.CancelOrders)
	if err != nil {
		return fmt.Errorf("CancelOrders error: %s", err)
	}

	err = functionalityTranslate(s.CancelOrder, newState.CancelOrder)
	if err != nil {
		return fmt.Errorf("CancelOrder error: %s", err)
	}

	err = functionalityTranslate(s.SubmitOrder, newState.SubmitOrder)
	if err != nil {
		return fmt.Errorf("SubmitOrder error: %s", err)
	}

	err = functionalityTranslate(s.SubmitOrders, newState.SubmitOrders)
	if err != nil {
		return fmt.Errorf("SubmitOrders error: %s", err)
	}

	err = functionalityTranslate(s.ModifyOrder, newState.ModifyOrder)
	if err != nil {
		return fmt.Errorf("ModifyOrder error: %s", err)
	}

	err = functionalityTranslate(s.DepositHistory, newState.DepositHistory)
	if err != nil {
		return fmt.Errorf("DepositHistory error: %s", err)
	}

	err = functionalityTranslate(s.WithdrawalHistory, newState.WithdrawalHistory)
	if err != nil {
		return fmt.Errorf("WithdrawalHistory error: %s", err)
	}

	err = functionalityTranslate(s.TradeHistory, newState.TradeHistory)
	if err != nil {
		return fmt.Errorf("TradeHistory error: %s", err)
	}

	err = functionalityTranslate(s.UserTradeHistory, newState.UserTradeHistory)
	if err != nil {
		return fmt.Errorf("UserTradeHistory error: %s", err)
	}

	err = functionalityTranslate(s.TradeFee, newState.TradeFee)
	if err != nil {
		return fmt.Errorf("TradeFee error: %s", err)
	}

	err = functionalityTranslate(s.FiatDepositFee, newState.FiatDepositFee)
	if err != nil {
		return fmt.Errorf("FiatDepositFee error: %s", err)
	}

	err = functionalityTranslate(s.FiatWithdrawalFee, newState.FiatWithdrawalFee)
	if err != nil {
		return fmt.Errorf("FiatWithdrawalFee error: %s", err)
	}

	err = functionalityTranslate(s.CryptoDepositFee, newState.CryptoDepositFee)
	if err != nil {
		return fmt.Errorf("CryptoDepositFee error: %s", err)
	}

	err = functionalityTranslate(s.CryptoWithdrawalFee, newState.CryptoWithdrawalFee)
	if err != nil {
		return fmt.Errorf("CryptoWithdrawalFee error: %s", err)
	}

	err = functionalityTranslate(s.TickerFetching, newState.TickerFetching)
	if err != nil {
		return fmt.Errorf("TickerFetching error: %s", err)
	}

	err = functionalityTranslate(s.KlineFetching, newState.KlineFetching)
	if err != nil {
		return fmt.Errorf("KlineFetching error: %s", err)
	}

	err = functionalityTranslate(s.TradeFetching, newState.TradeFetching)
	if err != nil {
		return fmt.Errorf("TradeFetching error: %s", err)
	}

	err = functionalityTranslate(s.OrderbookFetching, newState.OrderbookFetching)
	if err != nil {
		return fmt.Errorf("OrderbookFetching error: %s", err)
	}

	err = functionalityTranslate(s.AccountInfo, newState.AccountInfo)
	if err != nil {
		return fmt.Errorf("AccountInfo error: %s", err)
	}

	err = functionalityTranslate(s.FiatDeposit, newState.FiatDeposit)
	if err != nil {
		return fmt.Errorf("FiatDeposit error: %s", err)
	}

	err = functionalityTranslate(s.DeadMansSwitch, newState.DeadMansSwitch)
	if err != nil {
		return fmt.Errorf("DeadMansSwitch error: %s", err)
	}

	err = functionalityTranslate(s.FullPayloadSubscribe, newState.FullPayloadSubscribe)
	if err != nil {
		return fmt.Errorf("FullPayloadSubscribe error: %s", err)
	}

	err = functionalityTranslate(s.Subscribe, newState.Subscribe)
	if err != nil {
		return fmt.Errorf("Subscribe error: %s", err)
	}

	err = functionalityTranslate(s.Unsubscribe, newState.Unsubscribe)
	if err != nil {
		return fmt.Errorf("Unsubscribe error: %s", err)
	}

	err = functionalityTranslate(s.AuthenticatedEndpoints, newState.AuthenticatedEndpoints)
	if err != nil {
		return fmt.Errorf("AuthenticatedEndpoints error: %s", err)
	}

	err = functionalityTranslate(s.MessageCorrelation, newState.MessageCorrelation)
	if err != nil {
		return fmt.Errorf("MessageCorrelation error: %s", err)
	}

	err = functionalityTranslate(s.MessageSequenceNumbers, newState.MessageSequenceNumbers)
	if err != nil {
		return fmt.Errorf("MessageSequenceNumbers error: %s", err)
	}

	err = functionalityTranslate(s.CandleHistory, newState.CandleHistory)
	if err != nil {
		return fmt.Errorf("CandleHistory error: %s", err)
	}

	return nil
}

// functionalityTranslate validates and changes state based on the original
func functionalityTranslate(feature, newstate *bool) error {
	if feature == nil {
		if newstate != nil {
			return errUnsupported
		}
		return nil
	}
	if *newstate {
		feature = convert.BoolPtrT
	} else {
		feature = convert.BoolPtrF
	}
	return nil
}

// checkComponent matchings component to the underlying state to check to see
// if its supported (not nil) or enabled
func (s *State) checkComponent(component Component, checkSupported bool) (bool, error) {
	if s == nil {
		return false, errStateIsNil
	}
	var check *bool
	switch component {
	case TickerBatching:
		check = s.TickerBatching
	case AutoPairUpdates:
		check = s.AutoPairUpdates
	case AccountBalance:
		check = s.AccountBalance
	case CryptoDeposit:
		check = s.CryptoDeposit
	case CryptoWithdrawal:
		check = s.CryptoWithdrawal
	case FiatWithdraw:
		check = s.FiatWithdraw
	case GetOrder:
		check = s.GetOrder
	case GetOrders:
		check = s.GetOrders
	case CancelOrders:
		check = s.CancelOrders
	case CancelOrder:
		check = s.CancelOrder
	case SubmitOrder:
		check = s.SubmitOrder
	case SubmitOrders:
		check = s.SubmitOrders
	case ModifyOrder:
		check = s.ModifyOrder
	case DepositHistory:
		check = s.DepositHistory
	case WithdrawalHistory:
		check = s.WithdrawalHistory
	case TradeHistory:
		check = s.TradeHistory
	case UserTradeHistory:
		check = s.UserTradeHistory
	case TradeFee:
		check = s.TradeFee
	case FiatDepositFee:
		check = s.FiatDepositFee
	case FiatWithdrawalFee:
		check = s.FiatWithdrawalFee
	case CryptoDepositFee:
		check = s.CryptoDepositFee
	case CryptoWithdrawalFee:
		check = s.CryptoWithdrawalFee
	case TickerFetching:
		check = s.TickerFetching
	case KlineFetching:
		check = s.KlineFetching
	case TradeFetching:
		check = s.TradeFetching
	case OrderbookFetching:
		check = s.OrderbookFetching
	case AccountInfo:
		check = s.AccountInfo
	case FiatDeposit:
		check = s.FiatDeposit
	case DeadMansSwitch:
		check = s.DeadMansSwitch
	case FullPayloadSubscribe:
		check = s.FullPayloadSubscribe
	case Subscribe:
		check = s.Subscribe
	case Unsubscribe:
		check = s.Unsubscribe
	case AuthenticatedEndpoints:
		check = s.AuthenticatedEndpoints
	case MessageCorrelation:
		check = s.MessageCorrelation
	case MessageSequenceNumbers:
		check = s.MessageSequenceNumbers
	case CandleHistory:
		check = s.CandleHistory
	default:
		return false, fmt.Errorf("component [%v] not supported", component)
	}

	if checkSupported {
		return isSupported(check), nil
	}

	return isEnabled(check), nil
}

// isEnabled determines bool value from pointer bool
func isEnabled(ptr *bool) bool {
	return ptr != nil && *ptr
}

// isSupported checks not nil status of supplied pointer to determine if
// supported
func isSupported(functionality *bool) bool {
	return functionality != nil
}
