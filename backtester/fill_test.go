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
			if got := f.GetAmount(); got != tt.want {
				t.Errorf("GetAmount() = %v, want %v", got, tt.want)
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
			if got := f.GetCommission(); got != tt.want {
				t.Errorf("GetCommission() = %v, want %v", got, tt.want)
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
			if got := f.GetCost(); got != tt.want {
				t.Errorf("GetCost() = %v, want %v", got, tt.want)
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
			if got := f.GetDirection(); got != tt.want {
				t.Errorf("GetDirection() = %v, want %v", got, tt.want)
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
				Commission: 15,
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
			if got := f.GetPrice(); got != tt.want {
				t.Errorf("GetPrice() = %v, want %v", got, tt.want)
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
			if got := f.NetValue(); got != tt.want {
				t.Errorf("NetValue() = %v, want %v", got, tt.want)
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
			f.SetAmount(tt.args.i)
			if got := f.GetAmount(); got != tt.args.i	 {
				t.Errorf("GetAmount() = %v, want %v", got, tt.args.i)
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
			f.SetDirection(tt.args.s)
			if got := f.GetDirection(); got != tt.args.s	 {
				t.Errorf("GetDirection() = %v, want %v", got, tt.args.s)
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
				Price: 5,
				Commission: 5,
				ExchangeFee: 5,
			},
			0,
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
			if got := f.Value(); got != tt.want {
				t.Errorf("Value() = %v, want %v", got, tt.want)
			}
		})
	}
}
