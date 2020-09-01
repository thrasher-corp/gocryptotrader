package backtest

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestFill_GetAmount(t *testing.T) {
	type fields struct {
		Event       Event
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

func TestFill_GetCommission(t *testing.T) {
	type fields struct {
		Event       Event
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
				Commission: 5,
			},
			5,
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
			if got := f.GetCommission(); got != test.want {
				t.Errorf("GetCommission() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestFill_GetCost(t *testing.T) {
	type fields struct {
		Event       Event
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
				Cost: 15,
			},
			15,
		},
	}
	for x := range tests {
		t.Run(tests[x].name, func(t *testing.T) {
			f := &Fill{
				Event:       tests[x].fields.Event,
				Direction:   tests[x].fields.Direction,
				Amount:      tests[x].fields.Amount,
				Price:       tests[x].fields.Price,
				Commission:  tests[x].fields.Commission,
				ExchangeFee: tests[x].fields.ExchangeFee,
				Cost:        tests[x].fields.Cost,
			}
			if got := f.GetCost(); got != tests[x].want {
				t.Errorf("GetCost() = %v, want %v", got, tests[x].want)
			}
		})
	}
}

func TestFill_GetDirection(t *testing.T) {
	type fields struct {
		Event       Event
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
			if got := f.GetDirection(); got != tests[x].want {
				t.Errorf("GetDirection() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestFill_GetExchangeFee(t *testing.T) {
	type fields struct {
		Event       Event
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Fill{
				Event:       tt.fields.Event,
				Direction:   tt.fields.Direction,
				Amount:      tt.fields.Amount,
				Price:       tt.fields.Price,
				Commission:  tt.fields.Commission,
				ExchangeFee: tt.fields.ExchangeFee,
				Cost:        tt.fields.Cost,
			}
			if got := f.GetExchangeFee(); got != tt.want {
				t.Errorf("GetExchangeFee() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFill_GetPrice(t *testing.T) {
	type fields struct {
		Event       Event
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
			if got := f.GetPrice(); got != tests[x].want {
				t.Errorf("GetPrice() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestFill_NetValue(t *testing.T) {
	type fields struct {
		Event       Event
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
		Event       Event
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
			f.SetAmount(tests[x].args.i)
			if got := f.GetAmount(); got != test.args.i {
				t.Errorf("GetAmount() = %v, want %v", got, test.args.i)
			}
		})
	}
}

func TestFill_SetDirection(t *testing.T) {
	type fields struct {
		Event       Event
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
			f.SetDirection(tests[x].args.s)
			if got := f.GetDirection(); got != test.args.s {
				t.Errorf("GetDirection() = %v, want %v", got, test.args.s)
			}
		})
	}
}

func TestFill_Value(t *testing.T) {
	type fields struct {
		Event       Event
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
