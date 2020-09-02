package backtest

import (
	"reflect"
	"sync"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

func TestOrderBook_Add(t *testing.T) {
	type fields struct {
		counter int
		orders  []OrderEvent
		history []OrderEvent
		m       sync.Mutex
	}
	type args struct {
		order OrderEvent
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			"valid",
			fields{
				orders: []OrderEvent{},
			},
			args{},
		},
	}
	for x := range tests {
		test := &tests[x]
		t.Run(test.name, func(t *testing.T) {
			ob := &OrderBook{
				counter: test.fields.counter,
				orders:  test.fields.orders,
				history: test.fields.history,
			}
			t.Log(ob)
		})
	}
}

func TestOrderBook_OrderBy(t *testing.T) {
	type fields struct {
		counter int
		orders  []OrderEvent
		history []OrderEvent
		m       sync.Mutex
	}
	type args struct {
		fn func(order OrderEvent) bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []OrderEvent
		want1  bool
	}{
		{},
	}
	for x := range tests {
		test := &tests[x]
		t.Run(test.name, func(t *testing.T) {
			ob := &OrderBook{
				counter: test.fields.counter,
				orders:  test.fields.orders,
				history: test.fields.history,
			}
			got, got1 := ob.OrderBy(test.args.fn)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("OrderBy() got = %v, want %v", got, test.want)
			}
			if got1 != test.want1 {
				t.Errorf("OrderBy() got1 = %v, want %v", got1, test.want1)
			}
		})
	}
}

func TestOrderBook_Orders(t *testing.T) {
	type fields struct {
		counter int
		orders  []OrderEvent
		history []OrderEvent
		m       sync.Mutex
	}
	tests := []struct {
		name   string
		fields fields
		want   []OrderEvent
		want1  bool
	}{
		{},
	}
	for x := range tests {
		test := &tests[x]
		t.Run(test.name, func(t *testing.T) {
			ob := &OrderBook{
				counter: test.fields.counter,
				orders:  test.fields.orders,
				history: test.fields.history,
			}
			got := ob.Orders()
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Orders() got = %v, want %v", got, test.want)
			}
		})
	}
}

func TestOrderBook_OrdersAskBySymbol(t *testing.T) {
	type fields struct {
		counter int
		orders  []OrderEvent
		history []OrderEvent
		m       sync.Mutex
	}
	type args struct {
		p currency.Pair
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []OrderEvent
		want1  bool
	}{
		{},
	}
	for x := range tests {
		test := &tests[x]
		t.Run(test.name, func(t *testing.T) {
			ob := &OrderBook{
				counter: test.fields.counter,
				orders:  test.fields.orders,
				history: test.fields.history,
			}
			got, got1 := ob.OrdersAskBySymbol(test.args.p)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("OrdersAskBySymbol() got = %v, want %v", got, test.want)
			}
			if got1 != test.want1 {
				t.Errorf("OrdersAskBySymbol() got1 = %v, want %v", got1, test.want1)
			}
		})
	}
}

func TestOrderBook_OrdersBidBySymbol(t *testing.T) {
	type fields struct {
		counter int
		orders  []OrderEvent
		history []OrderEvent
		m       sync.Mutex
	}
	type args struct {
		p currency.Pair
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []OrderEvent
		want1  bool
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ob := &OrderBook{
				counter: tt.fields.counter,
				orders:  tt.fields.orders,
				history: tt.fields.history,
				m:       tt.fields.m,
			}
			got, got1 := ob.OrdersBidBySymbol(tt.args.p)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OrdersBidBySymbol() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("OrdersBidBySymbol() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestOrderBook_OrdersBySymbol(t *testing.T) {
	type fields struct {
		counter int
		orders  []OrderEvent
		history []OrderEvent
		m       sync.Mutex
	}
	type args struct {
		p currency.Pair
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []OrderEvent
		want1  bool
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ob := &OrderBook{
				counter: tt.fields.counter,
				orders:  tt.fields.orders,
				history: tt.fields.history,
				m:       tt.fields.m,
			}
			got, got1 := ob.OrdersBySymbol(tt.args.p)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OrdersBySymbol() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("OrdersBySymbol() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestOrderBook_OrdersCanceled(t *testing.T) {
	type fields struct {
		counter int
		orders  []OrderEvent
		history []OrderEvent
		m       sync.Mutex
	}
	tests := []struct {
		name   string
		fields fields
		want   []OrderEvent
		want1  bool
	}{
		{},
	}
	for x := range tests {
		test := &tests[x]
		t.Run(test.name, func(t *testing.T) {
			ob := &OrderBook{
				counter: test.fields.counter,
				orders:  test.fields.orders,
				history: test.fields.history,
			}
			got, got1 := ob.OrdersCanceled()
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("OrdersCanceled() got = %v, want %v", got, test.want1)
			}
			if got1 != test.want1 {
				t.Errorf("OrdersCanceled() got1 = %v, want %v", got1, test.want1)
			}
		})
	}
}

func TestOrderBook_OrdersOpen(t *testing.T) {
	type fields struct {
		counter int
		orders  []OrderEvent
		history []OrderEvent
		m       sync.Mutex
	}
	tests := []struct {
		name   string
		fields fields
		want   []OrderEvent
		want1  bool
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ob := &OrderBook{
				counter: tt.fields.counter,
				orders:  tt.fields.orders,
				history: tt.fields.history,
				m:       tt.fields.m,
			}
			got, got1 := ob.OrdersOpen()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OrdersOpen() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("OrdersOpen() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestOrderBook_Remove(t *testing.T) {
	type fields struct {
		counter int
		orders  []OrderEvent
		history []OrderEvent
		m       sync.Mutex
	}
	type args struct {
		id int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ob := &OrderBook{
				counter: tt.fields.counter,
				orders:  tt.fields.orders,
				history: tt.fields.history,
				m:       tt.fields.m,
			}
			if err := ob.Remove(tt.args.id); (err != nil) != tt.wantErr {
				t.Errorf("Remove() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
