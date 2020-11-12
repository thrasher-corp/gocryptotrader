package protocol

import (
	"errors"
	"reflect"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
)

func TestFunctionality(t *testing.T) {
	var s *State
	_, err := s.Functionality()
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	s = &State{TickerBatching: convert.BoolPtrT}
	functional, err := s.Functionality()
	if err != nil {
		t.Fatal(err)
	}
	if !functional.TickerBatching {
		t.Fatal("Should be enabled")
	}
}

func TestSupported(t *testing.T) {
	var s *State
	_, err := s.Supported()
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	s = &State{
		TickerBatching: convert.BoolPtrF,
	}
	supported, err := s.Supported()
	if err != nil {
		t.Fatal(err)
	}
	if !supported.TickerBatching {
		t.Fatal("Should be supported")
	}
}

func TestSetFunctionality(t *testing.T) {
	var s *State
	err := s.SetFunctionality(State{})
	if err == nil || err != errStateIsNil {
		t.Fatal("unexpected error")
	}

	s = &State{}

	newState := State{
		ProtocolEnabled:        true,
		AuthenticationEnabled:  true,
		TickerBatching:         convert.BoolPtrT,
		AccountBalance:         convert.BoolPtrT,
		CryptoDeposit:          convert.BoolPtrT,
		CryptoWithdrawal:       convert.BoolPtrT,
		FiatWithdraw:           convert.BoolPtrT,
		GetOrder:               convert.BoolPtrT,
		GetOrders:              convert.BoolPtrT,
		CancelOrders:           convert.BoolPtrT,
		CancelOrder:            convert.BoolPtrT,
		SubmitOrder:            convert.BoolPtrT,
		SubmitOrders:           convert.BoolPtrT,
		ModifyOrder:            convert.BoolPtrT,
		DepositHistory:         convert.BoolPtrT,
		WithdrawalHistory:      convert.BoolPtrT,
		TradeHistory:           convert.BoolPtrT,
		UserTradeHistory:       convert.BoolPtrT,
		TradeFee:               convert.BoolPtrT,
		FiatDepositFee:         convert.BoolPtrT,
		FiatWithdrawalFee:      convert.BoolPtrT,
		CryptoDepositFee:       convert.BoolPtrT,
		CryptoWithdrawalFee:    convert.BoolPtrT,
		TickerFetching:         convert.BoolPtrT,
		KlineFetching:          convert.BoolPtrT,
		TradeFetching:          convert.BoolPtrT,
		OrderbookFetching:      convert.BoolPtrT,
		AccountInfo:            convert.BoolPtrT,
		FiatDeposit:            convert.BoolPtrT,
		DeadMansSwitch:         convert.BoolPtrT,
		FullPayloadSubscribe:   convert.BoolPtrT,
		Subscribe:              convert.BoolPtrT,
		Unsubscribe:            convert.BoolPtrT,
		AuthenticatedEndpoints: convert.BoolPtrT,
		MessageCorrelation:     convert.BoolPtrT,
		MessageSequenceNumbers: convert.BoolPtrT,
		CandleHistory:          convert.BoolPtrT,
	}

	v := reflect.ValueOf(s)
	val := v.Elem()
	lenOfFields := val.NumField()
	for i := 2; i < lenOfFields; i++ {
		field := val.Field(i)
		err = s.SetFunctionality(newState)
		field.Set(reflect.ValueOf(convert.BoolPtrF))
		if err == nil {
			t.Fatal("error cannot be nil")
		}
		if !errors.Is(err, errUnsupported) {
			t.Fatal(err)
		}
	}
}

func TestCheckComponent(t *testing.T) {
	var s *State
	_, err := s.checkComponent(TickerBatching, false)
	if err != errStateIsNil {
		t.Fatal("unexpected result")
	}

	s = &State{}
	_, err = s.checkComponent(Component(1337), false)
	if err == nil || err.Error() != "component [1337] not supported" {
		t.Fatal("unexpected result")
	}

	for _, component := range components {
		_, err = s.checkComponent(component, false)
		if err != nil {
			t.Fatal(err)
		}
		_, err = s.checkComponent(component, true)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestIsEnabled(t *testing.T) {
	if isEnabled(nil) {
		t.Fatal("unexpected value")
	}
	if !isEnabled(convert.BoolPtrT) {
		t.Fatal("unexpected value")
	}
	if isEnabled(convert.BoolPtrF) {
		t.Fatal("unexpected value")
	}
}

func TestIsSupported(t *testing.T) {
	if isSupported(nil) {
		t.Fatal("unexpected value")
	}
	if !isSupported(convert.BoolPtrT) {
		t.Fatal("unexpected value")
	}
	if !isSupported(convert.BoolPtrF) {
		t.Fatal("unexpected value")
	}
}
