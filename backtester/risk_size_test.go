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
			args{
				order: new(Order),
			},
			&Order{},
			false,
		},
	}
	for x := range tests {
		t.Run(tests[x].name, func(t *testing.T) {
			r := &Risk{}
			got, err := r.EvaluateOrder(tests[x].args.order, tests[x].args.in1, tests[x].args.in2)
			if (err != nil) != tests[x].wantErr {
				t.Errorf("EvaluateOrder() error = %v, wantErr %v", err, tests[x].wantErr)
				return
			}
			if !reflect.DeepEqual(got, tests[x].want) {
				t.Errorf("EvaluateOrder() got = %v, want %v", got, tests[x].want)
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
			"valid",
			fields{
				5,
				5,
			},
			args{
				order: new(Order),
			},
			&Order{},
			false,
		},
		{
			"invalid",
			fields{},
			args{
				order: new(Order),
			},
			&Order{},
			true,
		},
	}
	for x := range tests {
		t.Run(tests[x].name, func(t *testing.T) {
			s := &Size{
				DefaultSize:  tests[x].fields.DefaultSize,
				DefaultValue: tests[x].fields.DefaultValue,
			}
			got, err := s.SizeOrder(tests[x].args.order, tests[x].args.in1, tests[x].args.in2)
			if (err != nil) != tests[x].wantErr {
				t.Errorf("SizeOrder() error = %v, wantErr %v", err, tests[x].wantErr)
				return
			}
			if !reflect.DeepEqual(got, tests[x].want) {
				t.Errorf("SizeOrder() got = %v, want %v", got, tests[x].want)
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
				DefaultSize:  5,
			},
			args{
				price: 5,
			},
			1,
		},
	}
	for x := range tests {
		t.Run(tests[x].name, func(t *testing.T) {
			s := &Size{
				DefaultSize:  tests[x].fields.DefaultSize,
				DefaultValue: tests[x].fields.DefaultValue,
			}
			if got := s.setDefaultSize(tests[x].args.price); got != tests[x].want {
				t.Errorf("setDefaultSize() = %v, want %v", got, tests[x].want)
			}
		})
	}
}
