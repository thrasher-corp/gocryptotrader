package json_test

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// The frames a busy Bitfinex order book sends most, and a full Bitfinex order, decoded with the types exchanges use,
// which the json package cannot import
var (
	benchmarkBookFrame  = []byte(`[17082,[8744.9,2,0.45603413],10345]`)
	benchmarkOrderTuple = []byte(`[33950998275,null,1573476747887,"tETHUSD",1573476748000,1573476748000,-0.5,-0.5,"LIMIT",null,null,null,0,"ACTIVE",null,null,220,0,0,0,null,null,null,0,1,null,null,null,"BFX",null,null,{"$F7":1}]`)
)

type benchmarkLevel struct {
	Price  float64 `tuple:"0"`
	Amount float64 `tuple:"2"`
}

func (l *benchmarkLevel) UnmarshalJSON(data []byte) error { return json.UnmarshalTuple(data, l) }

type benchmarkBookUpdate struct {
	ChannelID uint64         `tuple:"0"`
	Level     benchmarkLevel `tuple:"1"`
	Sequence  uint64         `tuple:"2"`
}

type benchmarkOrder struct {
	ID             uint64          `tuple:"0"`
	GroupID        uint64          `tuple:"1"`
	ClientOrderID  uint64          `tuple:"2"`
	Symbol         string          `tuple:"3"`
	CreatedAt      types.Time      `tuple:"4"`
	UpdatedAt      types.Time      `tuple:"5"`
	Amount         float64         `tuple:"6"`
	OriginalAmount float64         `tuple:"7"`
	Type           string          `tuple:"8"`
	Status         string          `tuple:"13"`
	Price          float64         `tuple:"16"`
	AveragePrice   float64         `tuple:"17"`
	Hidden         types.Boolean   `tuple:"24"`
	Routing        string          `tuple:"28"`
	Meta           json.RawMessage `tuple:"31"`
}

func BenchmarkUnmarshalTuple(b *testing.B) {
	b.Run("book update", func(b *testing.B) {
		for b.Loop() {
			var u benchmarkBookUpdate
			if err := json.UnmarshalTuple(benchmarkBookFrame, &u); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("order", func(b *testing.B) {
		for b.Loop() {
			var o benchmarkOrder
			if err := json.UnmarshalTuple(benchmarkOrderTuple, &o); err != nil {
				b.Fatal(err)
			}
		}
	})
}
