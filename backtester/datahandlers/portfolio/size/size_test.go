package size

import (
	"reflect"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/backtester/datahandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/orders"
)

func TestSize_SizeOrder(t *testing.T) {
	type fields struct {
		DefaultSize  float64
		DefaultValue float64
	}
	type args struct {
		order orders.OrderEvent
		in1   interfaces.DataEventHandler
		in2   portfolio.PortfolioHandler
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *order.Order
		wantErr bool
	}{
		{
			"valid",
			fields{
				5,
				5,
			},
			args{
				order: new(order.Order),
			},
			&order.Order{},
			false,
		},
		{
			"invalid",
			fields{},
			args{
				order: new(order.Order),
			},
			nil,
			true,
		},
	}
	for x := range tests {
		test := tests[x]
		t.Run(test.name, func(t *testing.T) {
			s := &Size{
				DefaultSize: test.fields.DefaultSize,
				MaxSize:     test.fields.DefaultValue,
			}
			got, err := s.SizeOrder(test.args.order, test.args.in1)
			if (err != nil) != test.wantErr {
				t.Errorf("SizeOrder() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("SizeOrder() got = %v, want %v", got, test.want)
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
		test := tests[x]
		t.Run(test.name, func(t *testing.T) {
			s := &Size{
				DefaultSize: test.fields.DefaultSize,
				MaxSize:     test.fields.DefaultValue,
			}
			if got := s.setDefaultSize(test.args.price); got != test.want {
				t.Errorf("setDefaultSize() = %v, want %v", got, test.want)
			}
		})
	}
}
