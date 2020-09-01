package backtest

import (
	"reflect"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

func TestPortfolio_IsLong(t *testing.T) {
	type fields struct {
		initialFunds float64
		funds        float64
		holdings     map[currency.Pair]Positions
		transactions []FillEvent
		sizeManager  SizeHandler
		riskManager  RiskHandler
	}
	type args struct {
		pair currency.Pair
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantPos Positions
		wantOk  bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Portfolio{
				initialFunds: tt.fields.initialFunds,
				funds:        tt.fields.funds,
				holdings:     tt.fields.holdings,
				transactions: tt.fields.transactions,
				sizeManager:  tt.fields.sizeManager,
				riskManager:  tt.fields.riskManager,
			}
			gotPos, gotOk := p.IsLong(tt.args.pair)
			if !reflect.DeepEqual(gotPos, tt.wantPos) {
				t.Errorf("IsLong() gotPos = %v, want %v", gotPos, tt.wantPos)
			}
			if gotOk != tt.wantOk {
				t.Errorf("IsLong() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}
