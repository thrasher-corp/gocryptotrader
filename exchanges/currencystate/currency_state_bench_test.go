package currencystate

import (
	"strconv"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func BenchmarkGetCurrencyStateSnapshot(b *testing.B) {
	for _, currencyCount := range []int{1, 32, 512} {
		b.Run(strconv.Itoa(currencyCount), func(b *testing.B) {
			states := &States{m: map[asset.Item]map[*currency.Item]*Currency{
				asset.Spot: make(map[*currency.Item]*Currency, currencyCount),
			}}
			if currencyCount > 1 {
				states.m[asset.Futures] = make(map[*currency.Item]*Currency, currencyCount/2)
			}
			for i := range currencyCount {
				code := currency.NewCode("BENCH" + strconv.Itoa(i))
				assetType := asset.Spot
				if i%2 != 0 {
					assetType = asset.Futures
				}
				states.m[assetType][code.Item] = &Currency{
					withdrawals: true,
					deposits:    i%2 == 0,
					trading:     i%3 == 0,
				}
			}

			b.ReportAllocs()
			for b.Loop() {
				snapshots, err := states.GetCurrencyStateSnapshot()
				if err != nil {
					b.Fatal(err)
				}
				if len(snapshots) != currencyCount {
					b.Fatalf("snapshot length %d, want %d", len(snapshots), currencyCount)
				}
			}
		})
	}
}
