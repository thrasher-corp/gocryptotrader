package internalordermanager

import (
	"reflect"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

func TestOrderBook_Add(t *testing.T) {
	type fields struct {
		counter int
		orders  []OrderEvent
		history []OrderEvent
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
			ob := &Orders{
				Counter: test.fields.counter,
				Orders:  test.fields.orders,
				History: test.fields.history,
			}
			t.Log(test.args)
			t.Log(ob)
		})
	}
}

func TestOrderBook_OrderBy(t *testing.T) {
	type fields struct {
		counter int
		orders  []OrderEvent
		history []OrderEvent
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
			ob := &Orders{
				Counter: test.fields.counter,
				Orders:  test.fields.orders,
				History: test.fields.history,
			}
			got, got1 := ob.OrderBy(test.args.fn)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("orderBy() got = %v, want %v", got, test.want)
			}
			if got1 != test.want1 {
				t.Errorf("orderBy() got1 = %v, want %v", got1, test.want1)
			}
		})
	}
}

func TestOrderBook_Orders(t *testing.T) {
	type fields struct {
		counter int
		orders  []OrderEvent
		history []OrderEvent
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
			ob := &Orders{
				Counter: test.fields.counter,
				Orders:  test.fields.orders,
				History: test.fields.history,
			}
			got := ob.GetOrders()
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("GetOrders() got = %v, want %v", got, test.want)
			}
		})
	}
}

func TestOrderBook_OrdersAskBySymbol(t *testing.T) {
	type fields struct {
		counter int
		orders  []OrderEvent
		history []OrderEvent
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
			ob := &Orders{
				Counter: test.fields.counter,
				Orders:  test.fields.orders,
				History: test.fields.history,
			}
			got, got1 := ob.OrdersAskByPair(test.args.p)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("OrdersAskByPair() got = %v, want %v", got, test.want)
			}
			if got1 != test.want1 {
				t.Errorf("OrdersAskByPair() got1 = %v, want %v", got1, test.want1)
			}
		})
	}
}

func TestOrderBook_OrdersBidBySymbol(t *testing.T) {
	type fields struct {
		counter int
		orders  []OrderEvent
		history []OrderEvent
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
			ob := &Orders{
				Counter: test.fields.counter,
				Orders:  test.fields.orders,
				History: test.fields.history,
			}
			got, got1 := ob.OrdersBidByPair(test.args.p)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("OrdersBidByPair() got = %v, want %v", got, test.want)
			}
			if got1 != test.want1 {
				t.Errorf("OrdersBidByPair() got1 = %v, want %v", got1, test.want1)
			}
		})
	}
}

func TestOrderBook_OrdersBySymbol(t *testing.T) {
	type fields struct {
		counter int
		orders  []OrderEvent
		history []OrderEvent
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
			ob := &Orders{
				Counter: test.fields.counter,
				Orders:  test.fields.orders,
				History: test.fields.history,
			}
			got, got1 := ob.OrdersByPair(test.args.p)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("OrdersByPair() got = %v, want %v", got, test.want)
			}
			if got1 != test.want1 {
				t.Errorf("OrdersByPair() got1 = %v, want %v", got1, test.want1)
			}
		})
	}
}

func TestOrderBook_OrdersCanceled(t *testing.T) {
	type fields struct {
		counter int
		orders  []OrderEvent
		history []OrderEvent
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
			ob := &Orders{
				Counter: test.fields.counter,
				Orders:  test.fields.orders,
				History: test.fields.history,
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
			ob := &Orders{
				Counter: test.fields.counter,
				Orders:  test.fields.orders,
				History: test.fields.history,
			}
			got, got1 := ob.OrdersOpen()
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("OrdersOpen() got = %v, want %v", got, test.want)
			}
			if got1 != test.want1 {
				t.Errorf("OrdersOpen() got1 = %v, want %v", got1, test.want1)
			}
		})
	}
}

func TestOrderBook_Remove(t *testing.T) {
	type fields struct {
		counter int
		orders  []OrderEvent
		history []OrderEvent
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
		{
			"Valid",
			fields{
				orders: []OrderEvent{},
			},
			args{
				1,
			},
			false,
		},
	}
	for x := range tests {
		test := &tests[x]
		t.Run(test.name, func(t *testing.T) {
			ob := &Orders{
				Counter: test.fields.counter,
				Orders:  test.fields.orders,
				History: test.fields.history,
			}
			o := new(order.Order)
			o.Price = 5
			o.Amount = 5
			ob.Add(o)
			if err := ob.Remove(test.args.id); (err != nil) != test.wantErr {
				t.Errorf("Remove() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
