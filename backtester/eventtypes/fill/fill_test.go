package fill

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestFill_GetAmount(t *testing.T) {
	type fields struct {
		Event       event.Event
		Direction   order.Side
		Amount      float64
		Price       float64
		Commission  float64
		ExchangeFee float64
		Cost        float64
	}
	tests := []struct {
		name   string
		fields fields
		want   float64
	}{
		{
			"10",
			fields{
				Amount: 10,
			},
			10,
		},
	}
	for x := range tests {
		test := tests[x]
		t.Run(test.name, func(t *testing.T) {
			f := &Fill{
				Event:       test.fields.Event,
				Direction:   test.fields.Direction,
				Amount:      test.fields.Amount,
				Price:       test.fields.Price,
				Commission:  test.fields.Commission,
				ExchangeFee: test.fields.ExchangeFee,
				Cost:        test.fields.Cost,
			}
			if got := f.GetAmount(); got != test.want {
				t.Errorf("GetAmount() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestFill_GetDirection(t *testing.T) {
	type fields struct {
		Event       event.Event
		Direction   order.Side
		Amount      float64
		Price       float64
		Commission  float64
		ExchangeFee float64
		Cost        float64
	}
	tests := []struct {
		name   string
		fields fields
		want   order.Side
	}{
		{
			"buy",
			fields{
				Direction: order.Buy,
			},
			order.Buy,
		},
		{
			"sell",
			fields{
				Direction: order.Sell,
			},
			order.Sell,
		},
		{
			"bid",
			fields{
				Direction: order.Bid,
			},
			order.Bid,
		},
	}
	for x := range tests {
		test := tests[x]
		t.Run(test.name, func(t *testing.T) {
			f := &Fill{
				Event:       test.fields.Event,
				Direction:   test.fields.Direction,
				Amount:      test.fields.Amount,
				Price:       test.fields.Price,
				Commission:  test.fields.Commission,
				ExchangeFee: test.fields.ExchangeFee,
				Cost:        test.fields.Cost,
			}
			if got := f.GetDirection(); got != test.want {
				t.Errorf("GetDirection() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestFill_GetExchangeFee(t *testing.T) {
	type fields struct {
		Event       event.Event
		Direction   order.Side
		Amount      float64
		Price       float64
		Commission  float64
		ExchangeFee float64
		Cost        float64
	}
	tests := []struct {
		name   string
		fields fields
		want   float64
	}{
		{
			"15",
			fields{
				Commission:  15,
				ExchangeFee: 15,
			},
			15,
		},
	}
	for x := range tests {
		test := tests[x]
		t.Run(test.name, func(t *testing.T) {
			f := &Fill{
				Event:       test.fields.Event,
				Direction:   test.fields.Direction,
				Amount:      test.fields.Amount,
				Price:       test.fields.Price,
				Commission:  test.fields.Commission,
				ExchangeFee: test.fields.ExchangeFee,
				Cost:        test.fields.Cost,
			}
			if got := f.GetExchangeFee(); got != test.want {
				t.Errorf("GetExchangeFee() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestFill_GetPrice(t *testing.T) {
	type fields struct {
		Event       event.Event
		Direction   order.Side
		Amount      float64
		Price       float64
		Commission  float64
		ExchangeFee float64
		Cost        float64
	}
	tests := []struct {
		name   string
		fields fields
		want   float64
	}{
		{
			"10",
			fields{},
			0,
		},
	}
	for x := range tests {
		test := tests[x]
		t.Run(test.name, func(t *testing.T) {
			f := &Fill{
				Event:       test.fields.Event,
				Direction:   test.fields.Direction,
				Amount:      test.fields.Amount,
				Price:       test.fields.Price,
				Commission:  test.fields.Commission,
				ExchangeFee: test.fields.ExchangeFee,
				Cost:        test.fields.Cost,
			}
			if got := f.GetClosePrice(); got != test.want {
				t.Errorf("GetClosePrice() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestFill_NetValue(t *testing.T) {
	type fields struct {
		Event       event.Event
		Direction   order.Side
		Amount      float64
		Price       float64
		Commission  float64
		ExchangeFee float64
		Cost        float64
	}
	tests := []struct {
		name   string
		fields fields
		want   float64
	}{
		{
			"NetValue",
			fields{
				Direction: order.Buy,
			},
			0,
		},
		{
			"NetValue",
			fields{
				Direction: order.Sell,
			},
			0,
		},
	}
	for x := range tests {
		test := tests[x]
		t.Run(test.name, func(t *testing.T) {
			f := &Fill{
				Event:       test.fields.Event,
				Direction:   test.fields.Direction,
				Amount:      test.fields.Amount,
				Price:       test.fields.Price,
				Commission:  test.fields.Commission,
				ExchangeFee: test.fields.ExchangeFee,
				Cost:        test.fields.Cost,
			}
			if got := f.NetValue(); got != test.want {
				t.Errorf("NetValue() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestFill_SetAmount(t *testing.T) {
	type fields struct {
		Event       event.Event
		Direction   order.Side
		Amount      float64
		Price       float64
		Commission  float64
		ExchangeFee float64
		Cost        float64
	}
	type args struct {
		i float64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			"5",
			fields{
				Amount: 5,
			},
			args{
				5,
			},
		},
	}
	for x := range tests {
		test := tests[x]
		t.Run(test.name, func(t *testing.T) {
			f := &Fill{
				Event:       test.fields.Event,
				Direction:   test.fields.Direction,
				Amount:      test.fields.Amount,
				Price:       test.fields.Price,
				Commission:  test.fields.Commission,
				ExchangeFee: test.fields.ExchangeFee,
				Cost:        test.fields.Cost,
			}
			f.SetAmount(test.args.i)
			if got := f.GetAmount(); got != test.args.i {
				t.Errorf("GetAmount() = %v, want %v", got, test.args.i)
			}
		})
	}
}

func TestFill_SetDirection(t *testing.T) {
	type fields struct {
		Event       event.Event
		Direction   order.Side
		Amount      float64
		Price       float64
		Commission  float64
		ExchangeFee float64
		Cost        float64
	}
	type args struct {
		s order.Side
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			"Buy",
			fields{},
			args{
				order.Buy,
			},
		},
	}
	for x := range tests {
		test := tests[x]
		t.Run(test.name, func(t *testing.T) {
			f := &Fill{
				Event:       test.fields.Event,
				Direction:   test.fields.Direction,
				Amount:      test.fields.Amount,
				Price:       test.fields.Price,
				Commission:  test.fields.Commission,
				ExchangeFee: test.fields.ExchangeFee,
				Cost:        test.fields.Cost,
			}
			f.SetDirection(test.args.s)
			if got := f.GetDirection(); got != test.args.s {
				t.Errorf("GetDirection() = %v, want %v", got, test.args.s)
			}
		})
	}
}

func TestFill_Value(t *testing.T) {
	type fields struct {
		Event       event.Event
		Direction   order.Side
		Amount      float64
		Price       float64
		Commission  float64
		ExchangeFee float64
		Cost        float64
	}
	tests := []struct {
		name   string
		fields fields
		want   float64
	}{
		{
			"5",
			fields{
				Price:       5,
				Commission:  5,
				ExchangeFee: 5,
			},
			0,
		},
	}
	for x := range tests {
		test := tests[x]
		t.Run(test.name, func(t *testing.T) {
			f := &Fill{
				Event:       test.fields.Event,
				Direction:   test.fields.Direction,
				Amount:      test.fields.Amount,
				Price:       test.fields.Price,
				Commission:  test.fields.Commission,
				ExchangeFee: test.fields.ExchangeFee,
				Cost:        test.fields.Cost,
			}
			if got := f.Value(); got != test.want {
				t.Errorf("Value() = %v, want %v", got, test.want)
			}
		})
	}
}
