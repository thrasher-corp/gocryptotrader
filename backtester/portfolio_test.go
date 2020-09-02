package backtest

import (
	"reflect"
	"testing"
	"time"

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
		{
			"IsLong - false",
			fields{},
			args{},
			Positions{},
			false,
		},
		{
			"IsLong - true",
			fields{
				holdings: map[currency.Pair]Positions{
					currency.NewPair(currency.BTC, currency.USDT): {
						timestamp:        time.Time{},
						pair:             currency.Pair{},
						amount:           5,
						amountBought:     0,
						amountSold:       0,
						avgPrice:         0,
						avgPriceNet:      0,
						avgPriceBought:   0,
						avgPriceSold:     0,
						value:            0,
						valueBought:      0,
						valueSold:        0,
						netValue:         0,
						netValueBought:   0,
						netValueSold:     0,
						marketPrice:      0,
						marketValue:      0,
						commission:       0,
						exchangeFee:      0,
						cost:             0,
						costBasis:        0,
						realProfitLoss:   0,
						unrealProfitLoss: 0,
						totalProfitLoss:  0,
					},
				},
			},
			args{},
			Positions{},
			false,
		},
	}
	for x := range tests {
		test := tests[x]
		t.Run(test.name, func(t *testing.T) {
			p := &Portfolio{
				initialFunds: test.fields.initialFunds,
				funds:        test.fields.funds,
				holdings:     test.fields.holdings,
				transactions: test.fields.transactions,
				sizeManager:  test.fields.sizeManager,
				riskManager:  test.fields.riskManager,
			}
			gotPos, gotOk := p.IsLong(test.args.pair)
			if !reflect.DeepEqual(gotPos, test.wantPos) {
				t.Errorf("IsLong() gotPos = %v, want %v", gotPos, test.wantPos)
			}
			if gotOk != test.wantOk {
				t.Errorf("IsLong() gotOk = %v, want %v", gotOk, test.wantOk)
			}
		})
	}
}
