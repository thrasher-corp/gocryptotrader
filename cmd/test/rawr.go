package main

import (
	"flag"
	"fmt"
	"math/rand"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/math"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

const (
	tradeFee = 0.002 // 0.02
)

func randFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func main() {
	var generate bool
	var seed int64
	var buyAmount float64

	flag.BoolVar(&generate, "generate", false, "generate the orderbook json data for deterministic data")
	flag.Int64Var(&seed, "seed", time.Now().UnixNano(), "the seed for use for the random number generator")
	flag.Float64Var(&buyAmount, "amount", 100000, "the buy amount to use")
	flag.Parse()

	fmt.Printf("Orderbook generation seed: %d\n", seed)
	rand.Seed(seed)

	ob := orderbook.Base{
		Pair: currency.NewPair(currency.BTC, currency.USD),
	}

	// we'll be buying from the asks
	for x := float64(1000); x < 1100; x += .5 {
		ob.Asks = append(ob.Asks, orderbook.Item{
			Price:  x,
			Amount: randFloat(1, 5),
		})
	}
	amt, total := ob.TotalAsksAmount()
	fmt.Printf("Orderbook asks generated. Len=%d BTC=%f Total USD val=%f\n",
		len(ob.Asks), amt, total)

	// we'll be selling into the bids
	for x := float64(1150); x > 1050; x -= .5 {
		ob.Bids = append(ob.Bids, orderbook.Item{
			Price:  x,
			Amount: randFloat(1, 5),
		})
	}
	var temp float64
	for key := range ob.Bids {
		temp += ob.Bids[key].Amount
		fmt.Printf("(%f, %f), ", temp, ob.Bids[key].Price)
	}
	amt, total = ob.TotalBidsAmount()
	fmt.Printf("Orderbook bids generated. Len=%d BTC=%f Total USD val=%f\n",
		len(ob.Bids), amt, total)

	p1, p2 := ob.Asks[199].Price, ob.Bids[0].Price
	fmt.Printf("%f%% price diff between %f and %f\n", math.CalculatePercentageDifference(p2,
		p1), p1, p2)

	s := ob.SimulateOrder(buyAmount, true)
	btcAmt := s.Amount
	fmt.Printf("Bought %v BTC with %f amount\n", btcAmt, buyAmount)
	s = ob.SimulateOrder(s.Amount, false)
	fmt.Printf("Sold %v BTC to USD with %f amount\n", btcAmt, buyAmount)

	netProfit := s.Amount - buyAmount
	fmt.Printf("Made $%.2f [%.2f%%]\n", netProfit, math.CalculatePercentageDifference(s.Amount, buyAmount))

	fmt.Println("Finding most revenue...")
	c := calcBestRevenue(&ob)
	fmt.Println(c)
}

func calcBestRevenue(ob *orderbook.Base) float64 {
	var bestProfit, bestAmount float64
	for amount := float64(1000); ; amount += 1000 {
		s := ob.SimulateOrder(amount, true)
		s = ob.SimulateOrder(s.Amount, false)
		netProfit := s.Amount - amount
		if netProfit > bestProfit {
			fmt.Printf("Made $%.2f [%.2f%%]\n", netProfit, math.CalculatePercentageDifference(s.Amount, amount))
			bestProfit = netProfit
			bestAmount = amount
		} else {
			return bestAmount
		}
	}
}

func calcBestPercentageReturn(ob *orderbook.Base) float64 {
	return 0
}
