package event

import (
	"reflect"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/signal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestEvent_GetTime(t *testing.T) {
	type fields struct {
		Time         time.Time
		CurrencyPair currency.Pair
	}
	tests := []struct {
		name   string
		fields fields
		want   time.Time
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Event{
				Time:         tt.fields.Time,
				CurrencyPair: tt.fields.CurrencyPair,
			}
			if got := e.GetTime(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvent_IsEvent(t *testing.T) {
	type fields struct {
		Time         time.Time
		CurrencyPair currency.Pair
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			"hello",
			fields{
				Time:         time.Time{},
				CurrencyPair: currency.Pair{},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Event{
				Time:         tt.fields.Time,
				CurrencyPair: tt.fields.CurrencyPair,
			}
			if got := e.IsEvent(); got != tt.want {
				t.Errorf("IsEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvent_Pair(t *testing.T) {
	type fields struct {
		Time         time.Time
		CurrencyPair currency.Pair
	}
	tests := []struct {
		name   string
		fields fields
		want   currency.Pair
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Event{
				Time:         tt.fields.Time,
				CurrencyPair: tt.fields.CurrencyPair,
			}
			if got := e.Pair(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Pair() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSignal_GetAmount(t *testing.T) {
	type fields struct {
		Event     Event
		Amount    float64
		Price     float64
		Direction order.Side
	}
	tests := []struct {
		name   string
		fields fields
		want   float64
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &signal.Signal{
				Event:     tt.fields.Event,
				Amount:    tt.fields.Amount,
				Price:     tt.fields.Price,
				Direction: tt.fields.Direction,
			}
			if got := s.GetAmount(); got != tt.want {
				t.Errorf("GetAmount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSignal_GetDirection(t *testing.T) {
	type fields struct {
		Event     Event
		Amount    float64
		Price     float64
		Direction order.Side
	}
	tests := []struct {
		name   string
		fields fields
		want   order.Side
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &signal.Signal{
				Event:     tt.fields.Event,
				Amount:    tt.fields.Amount,
				Price:     tt.fields.Price,
				Direction: tt.fields.Direction,
			}
			if got := s.GetDirection(); got != tt.want {
				t.Errorf("GetDirection() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSignal_GetPrice(t *testing.T) {
	type fields struct {
		Event     Event
		Amount    float64
		Price     float64
		Direction order.Side
	}
	tests := []struct {
		name   string
		fields fields
		want   float64
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &signal.Signal{
				Event:     tt.fields.Event,
				Amount:    tt.fields.Amount,
				Price:     tt.fields.Price,
				Direction: tt.fields.Direction,
			}
			if got := s.GetPrice(); got != tt.want {
				t.Errorf("GetPrice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSignal_IsSignal(t *testing.T) {
	type fields struct {
		Event     Event
		Amount    float64
		Price     float64
		Direction order.Side
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			"hello",
			fields{
				Event:     Event{},
				Amount:    0,
				Price:     0,
				Direction: "",
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &signal.Signal{
				Event:     tt.fields.Event,
				Amount:    tt.fields.Amount,
				Price:     tt.fields.Price,
				Direction: tt.fields.Direction,
			}
			if got := s.IsSignal(); got != tt.want {
				t.Errorf("IsSignal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSignal_Pair(t *testing.T) {
	type fields struct {
		Event     Event
		Amount    float64
		Price     float64
		Direction order.Side
	}
	tests := []struct {
		name   string
		fields fields
		want   currency.Pair
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &signal.Signal{
				Event:     tt.fields.Event,
				Amount:    tt.fields.Amount,
				Price:     tt.fields.Price,
				Direction: tt.fields.Direction,
			}
			if got := s.Pair(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Pair() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSignal_SetAmount(t *testing.T) {
	type fields struct {
		Event     Event
		Amount    float64
		Price     float64
		Direction order.Side
	}
	type args struct {
		f float64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &signal.Signal{
				Event:     tt.fields.Event,
				Amount:    tt.fields.Amount,
				Price:     tt.fields.Price,
				Direction: tt.fields.Direction,
			}
			_ = s
		})
	}
}

func TestSignal_SetDirection(t *testing.T) {
	type fields struct {
		Event     Event
		Amount    float64
		Price     float64
		Direction order.Side
	}
	type args struct {
		st order.Side
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &signal.Signal{
				Event:     tt.fields.Event,
				Amount:    tt.fields.Amount,
				Price:     tt.fields.Price,
				Direction: tt.fields.Direction,
			}
			_ = s
		})
	}
}

func TestSignal_SetPrice(t *testing.T) {
	type fields struct {
		Event     Event
		Amount    float64
		Price     float64
		Direction order.Side
	}
	type args struct {
		f float64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &signal.Signal{
				Event:     tt.fields.Event,
				Amount:    tt.fields.Amount,
				Price:     tt.fields.Price,
				Direction: tt.fields.Direction,
			}
			_ = s
		})
	}
}
