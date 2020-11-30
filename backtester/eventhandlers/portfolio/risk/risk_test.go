package risk

import (
	"reflect"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
)

func TestRisk_EvaluateOrder(t *testing.T) {
	type args struct {
		order exchange.OrderEvent
		in1   interfaces.DataEventHandler
		in2   holdings.Holding
	}
	tests := []struct {
		name    string
		args    args
		want    *order.Order
		wantErr bool
	}{
		{
			"Test",
			args{
				order: new(order.Order),
			},
			&order.Order{},
			false,
		},
	}
	for x := range tests {
		test := tests[x]
		t.Run(test.name, func(t *testing.T) {
			r := &Risk{}
			got, err := r.EvaluateOrder(test.args.order, test.args.in1, test.args.in2)
			if (err != nil) != test.wantErr {
				t.Errorf("EvaluateOrder() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("EvaluateOrder() got = %v, want %v", got, test.want)
			}
		})
	}
}
