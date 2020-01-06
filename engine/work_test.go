package engine

import (
	"sync"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/btcmarkets"
	"golang.org/x/time/rate"
)

func TestStartStop(t *testing.T) {
	testWorkSuite := Get(1, true)
	err := testWorkSuite.Stop()
	if err == nil {
		t.Error("error cannot be nil")
	}

	err = testWorkSuite.Start()
	if err != nil {
		t.Error(err)
	}

	err = testWorkSuite.Stop()
	if err != nil {
		t.Error(err)
	}

	err = testWorkSuite.Stop()
	if err == nil {
		t.Error("error cannot be nil")
	}

	err = testWorkSuite.Start()
	if err != nil {
		t.Error(err)
	}
}

func TestFetchTickerLive(t *testing.T) {
	b := btcmarkets.BTCMarkets{}
	b.SetDefaults()

	cfg := config.GetConfig()
	err := cfg.LoadConfig("../testdata/configtest.json", true)
	if err != nil {
		t.Fatal(err)
	}

	btc, err := cfg.GetExchangeConfig("BTC Markets")
	if err != nil {
		t.Fatal(err)
	}

	err = b.Setup(btc)
	if err != nil {
		t.Fatal(err)
	}

	b.Verbose = true

	testWorkSuite := Get(10, true)
	err = testWorkSuite.Start()
	if err != nil {
		t.Error(err)
	}

	client, _ := uuid.NewV4()

	p := currency.NewPairFromString("BTC-AUD")
	_, err = testWorkSuite.Exchange(client, &b).FetchTicker(p, asset.Spot, make(chan int))
	if err != nil {
		t.Error(err)
	}
}

type tester struct{}

func (r *tester) Execute() {}
func (r *tester) GetReservation() *rate.Reservation {
	return nil
}

func BenchmarkWorkManagerOneWorkerConsecutive(b *testing.B) {
	testWorkSuite := Get(1, false)
	err := testWorkSuite.Start()
	if err != nil {
		b.Error(err)
	}

	c := make(chan int)

	for i := 0; i < b.N; i++ {
		err = testWorkSuite.ExecuteJob(&tester{}, low, c)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkWorkManagerDefaultWorkerAmountConsecutive(b *testing.B) {
	testWorkSuite := Get(defaultWorkerCount, false)
	err := testWorkSuite.Start()
	if err != nil {
		b.Error(err)
	}

	c := make(chan int)

	for i := 0; i < b.N; i++ {
		err = testWorkSuite.ExecuteJob(&tester{}, low, c)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkWorkManagerOneWorkerParallel(b *testing.B) {
	testWorkSuite := Get(1, false)
	err := testWorkSuite.Start()
	if err != nil {
		b.Error(err)
	}

	c := make(chan int)

	for i := 0; i < b.N; i++ {
		// Batch
		var wg sync.WaitGroup
		for x := 0; x < 12; x++ {
			wg.Add(1)
			go func(wg *sync.WaitGroup) {
				err = testWorkSuite.ExecuteJob(&tester{}, low, c)
				if err != nil {
					b.Error(err)
				}
				wg.Done()
			}(&wg)
		}
	}
}

func BenchmarkWorkManagerDefaultWorkerAmountParallel(b *testing.B) {
	testWorkSuite := Get(defaultWorkerCount, false)
	err := testWorkSuite.Start()
	if err != nil {
		b.Error(err)
	}

	c := make(chan int)

	for i := 0; i < b.N; i++ {
		// Batch
		var wg sync.WaitGroup
		for x := 0; x < 12; x++ {
			wg.Add(1)
			go func(wg *sync.WaitGroup) {
				err = testWorkSuite.ExecuteJob(&tester{}, low, c)
				if err != nil {
					b.Error(err)
				}
				wg.Done()
			}(&wg)
		}
	}
}
