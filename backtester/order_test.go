package backtest

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestOrder_IsLeveraged(t *testing.T) {
	type fields struct {
		Event     Event
		id        int
		Direction order.Side
		Status    order.Status
		Price     float64
		Amount    float64
		OrderType order.Type
		limit     float64
		leverage  float64
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			"true",
			fields{
				leverage: 2.0,
			},
			true,
		},
		{
			"false",
			fields{
				leverage: 0.0,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Order{
				Event:     tt.fields.Event,
				id:        tt.fields.id,
				Direction: tt.fields.Direction,
				Status:    tt.fields.Status,
				Price:     tt.fields.Price,
				Amount:    tt.fields.Amount,
				OrderType: tt.fields.OrderType,
				limit:     tt.fields.limit,
				leverage:  tt.fields.leverage,
			}
			if got := o.IsLeveraged(); got != tt.want {
				t.Errorf("IsLeveraged() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrder_Leverage(t *testing.T) {
	type fields struct {
		Event     Event
		id        int
		Direction order.Side
		Status    order.Status
		Price     float64
		Amount    float64
		OrderType order.Type
		limit     float64
		leverage  float64
	}
	tests := []struct {
		name   string
		fields fields
		want   float64
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Order{
				Event:     tt.fields.Event,
				id:        tt.fields.id,
				Direction: tt.fields.Direction,
				Status:    tt.fields.Status,
				Price:     tt.fields.Price,
				Amount:    tt.fields.Amount,
				OrderType: tt.fields.OrderType,
				limit:     tt.fields.limit,
				leverage:  tt.fields.leverage,
			}
			if got := o.Leverage(); got != tt.want {
				t.Errorf("Leverage() = %v, want %v", got, tt.want)
			}
		})
	}
}
