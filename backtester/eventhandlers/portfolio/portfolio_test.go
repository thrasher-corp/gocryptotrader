package portfolio

import (
	"reflect"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/risk"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/positions"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

func TestPortfolio_IsLong(t *testing.T) {
	type fields struct {
		initialFunds float64
		funds        float64
		holdings     map[currency.Pair]positions.Positions
		transactions []fill.FillEvent
		sizeManager  SizeHandler
		riskManager  risk.RiskHandler
	}
	type args struct {
		pair currency.Pair
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantPos positions.Positions
		wantOk  bool
	}{
		{
			"IsLong - false",
			fields{},
			args{},
			positions.Positions{},
			false,
		},
		{
			"IsLong - true",
			fields{
				holdings: map[currency.Pair]positions.Positions{
					currency.NewPair(currency.BTC, currency.USDT): {
						Timestamp:          time.Time{},
						Pair:               currency.Pair{},
						Amount:             5,
						AmountBought:       0,
						AmountSold:         0,
						AveragePrice:       0,
						AveragePriceNet:    0,
						AveragePriceBought: 0,
						AveragePriceSold:   0,
						Value:              0,
						ValueBought:        0,
						ValueSold:          0,
						NetValue:           0,
						NetValueBought:     0,
						NetValueSold:       0,
						MarketPrice:        0,
						MarketValue:        0,
						Commission:         0,
						ExchangeFee:        0,
						Cost:               0,
						CostBasis:          0,
						RealProfitLoss:     0,
						UnrealProfitLoss:   0,
						TotalProfitLoss:    0,
					},
				},
			},
			args{},
			positions.Positions{},
			false,
		},
	}
	for x := range tests {
		test := tests[x]
		t.Run(test.name, func(t *testing.T) {
			p := &Portfolio{
				InitialFunds: test.fields.initialFunds,
				Funds:        test.fields.funds,
				Holdings:     test.fields.holdings,
				Transactions: test.fields.transactions,
				SizeManager:  test.fields.sizeManager,
				RiskManager:  test.fields.riskManager,
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
