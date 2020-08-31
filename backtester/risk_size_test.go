package backtest

import (
	"reflect"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

func TestRisk_EvaluateOrder(t *testing.T) {
	type args struct {
		order OrderEvent
		in1   DataEventHandler
		in2   map[currency.Pair]Positions
	}
	tests := []struct {
		name    string
		args    args
		want    *Order
		wantErr bool
	}{
		{
			"Test",
			args{},
			&Order{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Risk{}
			got, err := r.EvaluateOrder(tt.args.order, tt.args.in1, tt.args.in2)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateOrder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EvaluateOrder() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSize_SizeOrder(t *testing.T) {
	type fields struct {
		DefaultSize  float64
		DefaultValue float64
	}
	type args struct {
		order OrderEvent
		in1   DataEventHandler
		in2   PortfolioHandler
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *Order
		wantErr bool
	}{
	 {

	 },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Size{
				DefaultSize:  tt.fields.DefaultSize,
				DefaultValue: tt.fields.DefaultValue,
			}
			got, err := s.SizeOrder(tt.args.order, tt.args.in1, tt.args.in2)
			if (err != nil) != tt.wantErr {
				t.Errorf("SizeOrder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SizeOrder() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSize_setDefaultSize(t *testing.T) {
	type fields struct {
		DefaultSize  float64
		DefaultValue float64
	}
	type args struct {
		price float64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   float64
	}{
	 {
	 	"5",
	 	fields{
	 		DefaultValue: 5,
	 		DefaultSize: 5,
		},
		args{
			price: 5,
		},
		1,
	 },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Size{
				DefaultSize:  tt.fields.DefaultSize,
				DefaultValue: tt.fields.DefaultValue,
			}
			if got := s.setDefaultSize(tt.args.price); got != tt.want {
				t.Errorf("setDefaultSize() = %v, want %v", got, tt.want)
			}
		})
	}
}
