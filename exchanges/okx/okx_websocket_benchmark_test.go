package okx

import (
	"strings"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

var benchmarkChecksum uint32

const wsOrderBookSnapshotSpotNoInstTypeJSON = `{"arg":{"channel":"books","instId":"BTC-USDT"},"action":"snapshot","data":[{"asks":[["0.07026","5","0","1"],["0.07027","765","0","3"],["0.07028","110","0","1"],["0.0703","1264","0","1"],["0.07034","280","0","1"],["0.07035","2255","0","1"],["0.07036","28","0","1"],["0.07037","63","0","1"],["0.07039","137","0","2"],["0.0704","48","0","1"],["0.07041","32","0","1"],["0.07043","3985","0","1"],["0.07057","257","0","1"],["0.07058","7870","0","1"],["0.07059","161","0","1"],["0.07061","4539","0","1"],["0.07068","1438","0","3"],["0.07088","3162","0","1"],["0.07104","99","0","1"],["0.07108","5018","0","1"],["0.07115","1540","0","1"],["0.07129","5080","0","1"],["0.07145","1512","0","1"],["0.0715","5016","0","1"],["0.07171","5026","0","1"],["0.07192","5062","0","1"],["0.07197","1517","0","1"],["0.0726","1511","0","1"],["0.07314","10376","0","1"],["0.07354","1","0","1"],["0.07466","10277","0","1"],["0.07626","269","0","1"],["0.07636","269","0","1"],["0.0809","1","0","1"],["0.08899","1","0","1"],["0.09789","1","0","1"],["0.10768","1","0","1"]],"bids":[["0.07014","56","0","2"],["0.07011","608","0","1"],["0.07009","110","0","1"],["0.07006","1264","0","1"],["0.07004","2347","0","3"],["0.07003","279","0","1"],["0.07001","52","0","1"],["0.06997","91","0","1"],["0.06996","4242","0","2"],["0.06995","486","0","1"],["0.06992","161","0","1"],["0.06991","63","0","1"],["0.06988","7518","0","1"],["0.06976","186","0","1"],["0.06975","71","0","1"],["0.06973","1086","0","1"],["0.06961","513","0","2"],["0.06959","4603","0","1"],["0.0695","186","0","1"],["0.06946","3043","0","1"],["0.06939","103","0","1"],["0.0693","5053","0","1"],["0.06909","5039","0","1"],["0.06888","5037","0","1"],["0.06886","1526","0","1"],["0.06867","5008","0","1"],["0.06846","5065","0","1"],["0.06826","1572","0","1"],["0.06801","1565","0","1"],["0.06748","67","0","1"],["0.0674","111","0","1"],["0.0672","10038","0","1"],["0.06652","1","0","1"],["0.06625","1526","0","1"],["0.06619","10924","0","1"],["0.05986","1","0","1"],["0.05387","1","0","1"],["0.04848","1","0","1"],["0.04363","1","0","1"]],"ts":"1659792392540","checksum":-1462286744}]}`

const wsOrderBookSnapshotSwapNoInstTypeJSON = `{"arg":{"channel":"books","instId":"BTC-USD-SWAP"},"action":"snapshot","data":[{"asks":[["0.07026","5","0","1"],["0.07027","765","0","3"],["0.07028","110","0","1"],["0.0703","1264","0","1"],["0.07034","280","0","1"],["0.07035","2255","0","1"],["0.07036","28","0","1"],["0.07037","63","0","1"],["0.07039","137","0","2"],["0.0704","48","0","1"],["0.07041","32","0","1"],["0.07043","3985","0","1"],["0.07057","257","0","1"],["0.07058","7870","0","1"],["0.07059","161","0","1"],["0.07061","4539","0","1"],["0.07068","1438","0","3"],["0.07088","3162","0","1"],["0.07104","99","0","1"],["0.07108","5018","0","1"],["0.07115","1540","0","1"],["0.07129","5080","0","1"],["0.07145","1512","0","1"],["0.0715","5016","0","1"],["0.07171","5026","0","1"],["0.07192","5062","0","1"],["0.07197","1517","0","1"],["0.0726","1511","0","1"],["0.07314","10376","0","1"],["0.07354","1","0","1"],["0.07466","10277","0","1"],["0.07626","269","0","1"],["0.07636","269","0","1"],["0.0809","1","0","1"],["0.08899","1","0","1"],["0.09789","1","0","1"],["0.10768","1","0","1"]],"bids":[["0.07014","56","0","2"],["0.07011","608","0","1"],["0.07009","110","0","1"],["0.07006","1264","0","1"],["0.07004","2347","0","3"],["0.07003","279","0","1"],["0.07001","52","0","1"],["0.06997","91","0","1"],["0.06996","4242","0","2"],["0.06995","486","0","1"],["0.06992","161","0","1"],["0.06991","63","0","1"],["0.06988","7518","0","1"],["0.06976","186","0","1"],["0.06975","71","0","1"],["0.06973","1086","0","1"],["0.06961","513","0","2"],["0.06959","4603","0","1"],["0.0695","186","0","1"],["0.06946","3043","0","1"],["0.06939","103","0","1"],["0.0693","5053","0","1"],["0.06909","5039","0","1"],["0.06888","5037","0","1"],["0.06886","1526","0","1"],["0.06867","5008","0","1"],["0.06846","5065","0","1"],["0.06826","1572","0","1"],["0.06801","1565","0","1"],["0.06748","67","0","1"],["0.0674","111","0","1"],["0.0672","10038","0","1"],["0.06652","1","0","1"],["0.06625","1526","0","1"],["0.06619","10924","0","1"],["0.05986","1","0","1"],["0.05387","1","0","1"],["0.04848","1","0","1"],["0.04363","1","0","1"]],"ts":"1659792392540","checksum":-1462286744}]}`

type wsOrderBookBenchmarkCase struct {
	name      string
	payload   []byte
	update    []byte
	pair      currency.Pair
	assetType asset.Item
}

var wsOrderBookBenchmarkCases = []wsOrderBookBenchmarkCase{
	{
		name:      "spot",
		payload:   []byte(wsOrderBookSnapshotSpotNoInstTypeJSON),
		update:    []byte(strings.Replace(wsOrderBookSnapshotSpotNoInstTypeJSON, `"action":"snapshot"`, `"action":"update"`, 1)),
		pair:      currency.NewPairWithDelimiter("BTC", "USDT", "-"),
		assetType: asset.Spot,
	},
	{
		name:      "swap",
		payload:   []byte(wsOrderBookSnapshotSwapNoInstTypeJSON),
		update:    []byte(strings.Replace(wsOrderBookSnapshotSwapNoInstTypeJSON, `"action":"snapshot"`, `"action":"update"`, 1)),
		pair:      currency.NewPairWithDelimiter("BTC", "USD-SWAP", "-"),
		assetType: asset.PerpetualSwap,
	},
}

// BenchmarkWsProcessOrderBooksInstrumentTypeEmpty benchmarks wsProcessOrderBooks
// with realistic OKX snapshot messages where instType is omitted.
func BenchmarkWsProcessOrderBooksInstrumentTypeEmpty(b *testing.B) {
	b.ReportAllocs()

	modeCases := []struct {
		name  string
		setup func(*Exchange)
	}{
		{name: "default-assets", setup: func(*Exchange) {}},
		{name: "spot-perp-only", setup: func(ex *Exchange) {
			marginPairStore := ex.CurrencyPairs.Pairs[asset.Margin]
			marginPairStore.AssetEnabled = false
			marginPairStore.Enabled = nil
			ex.CurrencyPairs.Pairs[asset.Margin] = marginPairStore
		}},
	}

	for _, mode := range modeCases {
		for _, benchmarkCase := range wsOrderBookBenchmarkCases {
			b.Run(mode.name+"_"+benchmarkCase.name, func(b *testing.B) {
				ex := newBenchmarkOKXExchange(b)
				mode.setup(ex)
				drainBenchmarkDataHandler(b, ex)

				b.ResetTimer()
				for b.Loop() {
					if err := ex.wsProcessOrderBooks(b.Context(), nil, benchmarkCase.payload); err != nil {
						b.Fatalf("wsProcessOrderBooks must not error: %v", err)
					}
				}
			})
		}
	}
}

func BenchmarkWsProcessOrderBooksInstrumentTypeEmptyUpdate(b *testing.B) {
	b.ReportAllocs()

	for _, benchmarkCase := range wsOrderBookBenchmarkCases {
		b.Run(benchmarkCase.name, func(b *testing.B) {
			ex := newBenchmarkOKXExchange(b)
			drainBenchmarkDataHandler(b, ex)
			if err := ex.wsProcessOrderBooks(b.Context(), nil, benchmarkCase.payload); err != nil {
				b.Fatalf("snapshot seed must not error: %v", err)
			}

			b.ResetTimer()
			for b.Loop() {
				if err := ex.wsProcessOrderBooks(b.Context(), nil, benchmarkCase.update); err != nil {
					b.Fatalf("wsProcessOrderBooks update must not error: %v", err)
				}
			}
		})
	}
}

func BenchmarkWsOrderBookUnmarshalInstrumentTypeEmpty(b *testing.B) {
	b.ReportAllocs()

	for _, benchmarkCase := range wsOrderBookBenchmarkCases {
		b.Run(benchmarkCase.name, func(b *testing.B) {
			for b.Loop() {
				var response WsOrderBook
				if err := json.Unmarshal(benchmarkCase.payload, &response); err != nil {
					b.Fatalf("Unmarshal must not error: %v", err)
				}
			}
		})
	}
}

// BenchmarkGetAssetsFromInstrumentID benchmarks the instId-to-asset resolution
// used by wsProcessOrderBooks when instType is omitted.
func BenchmarkGetAssetsFromInstrumentID(b *testing.B) {
	b.ReportAllocs()

	benchmarkCases := []struct {
		name         string
		instrumentID string
	}{
		{name: "spot", instrumentID: "BTC-USDT"},
		{name: "swap", instrumentID: "BTC-USD-SWAP"},
	}

	for _, benchmarkCase := range benchmarkCases {
		b.Run(benchmarkCase.name, func(b *testing.B) {
			ex := new(Exchange)
			b.ResetTimer()

			for b.Loop() {
				if _, err := ex.getAssetsFromInstrumentID(benchmarkCase.instrumentID); err != nil {
					b.Fatalf("getAssetsFromInstrumentID must not error: %v", err)
				}
			}
		})
	}
}

func BenchmarkCalculateOrderbookChecksum(b *testing.B) {
	b.ReportAllocs()

	var response WsOrderBook
	if err := json.Unmarshal([]byte(wsOrderBookSnapshotSpotNoInstTypeJSON), &response); err != nil {
		b.Fatalf("Unmarshal must not error: %v", err)
	}
	ex := new(Exchange)
	data := &response.Data[0]

	b.ResetTimer()
	for b.Loop() {
		checksum, err := ex.CalculateOrderbookChecksum(data)
		if err != nil {
			b.Fatalf("CalculateOrderbookChecksum must not error: %v", err)
		}
		benchmarkChecksum = checksum
	}
}

func newBenchmarkOKXExchange(b *testing.B) *Exchange {
	b.Helper()
	ex := new(Exchange)
	if err := testexch.Setup(ex); err != nil {
		b.Fatalf("setup must not error: %v", err)
	}
	return ex
}

func drainBenchmarkDataHandler(b *testing.B, ex *Exchange) {
	b.Helper()
	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-stop:
				return
			case <-ex.Websocket.DataHandler.C:
			}
		}
	}()
	b.Cleanup(func() {
		close(stop)
	})
}

func BenchmarkGenerateOrderbookChecksum(b *testing.B) {
	b.ReportAllocs()

	var response WsOrderBook
	if err := json.Unmarshal([]byte(wsOrderBookSnapshotSpotNoInstTypeJSON), &response); err != nil {
		b.Fatalf("Unmarshal must not error: %v", err)
	}
	ex := newBenchmarkOKXExchange(b)
	drainBenchmarkDataHandler(b, ex)
	response.Argument.InstrumentID.Delimiter = currency.DashDelimiter
	if err := ex.WsProcessSnapshotOrderBook(&response.Data[0], response.Argument.InstrumentID, []asset.Item{asset.Spot}); err != nil {
		b.Fatalf("WsProcessSnapshotOrderBook must not error: %v", err)
	}
	book, err := orderbook.Get(ex.Name, response.Argument.InstrumentID, asset.Spot)
	if err != nil {
		b.Fatalf("orderbook.Get must not error: %v", err)
	}

	b.ResetTimer()
	for b.Loop() {
		benchmarkChecksum = generateOrderbookChecksum(book)
	}
}

func BenchmarkWsProcessSnapshotOrderBook(b *testing.B) {
	b.ReportAllocs()

	for _, benchmarkCase := range wsOrderBookBenchmarkCases {
		b.Run(benchmarkCase.name, func(b *testing.B) {
			var response WsOrderBook
			if err := json.Unmarshal(benchmarkCase.payload, &response); err != nil {
				b.Fatalf("Unmarshal must not error: %v", err)
			}
			response.Argument.InstrumentID.Delimiter = currency.DashDelimiter

			ex := newBenchmarkOKXExchange(b)
			drainBenchmarkDataHandler(b, ex)
			assets := []asset.Item{benchmarkCase.assetType}

			b.ResetTimer()
			for b.Loop() {
				if err := ex.WsProcessSnapshotOrderBook(&response.Data[0], response.Argument.InstrumentID, assets); err != nil {
					b.Fatalf("WsProcessSnapshotOrderBook must not error: %v", err)
				}
			}
		})
	}
}

func BenchmarkWsProcessUpdateOrderbook(b *testing.B) {
	b.ReportAllocs()

	for _, benchmarkCase := range wsOrderBookBenchmarkCases {
		b.Run(benchmarkCase.name, func(b *testing.B) {
			var snapshotResponse WsOrderBook
			if err := json.Unmarshal(benchmarkCase.payload, &snapshotResponse); err != nil {
				b.Fatalf("snapshot unmarshal must not error: %v", err)
			}
			snapshotResponse.Argument.InstrumentID.Delimiter = currency.DashDelimiter

			var updateResponse WsOrderBook
			if err := json.Unmarshal(benchmarkCase.update, &updateResponse); err != nil {
				b.Fatalf("update unmarshal must not error: %v", err)
			}
			updateResponse.Argument.InstrumentID.Delimiter = currency.DashDelimiter

			ex := newBenchmarkOKXExchange(b)
			drainBenchmarkDataHandler(b, ex)
			assets := []asset.Item{benchmarkCase.assetType}
			if err := ex.WsProcessSnapshotOrderBook(&snapshotResponse.Data[0], snapshotResponse.Argument.InstrumentID, assets); err != nil {
				b.Fatalf("WsProcessSnapshotOrderBook seed must not error: %v", err)
			}

			b.ResetTimer()
			for b.Loop() {
				if err := ex.WsProcessUpdateOrderbook(&updateResponse.Data[0], updateResponse.Argument.InstrumentID, assets); err != nil {
					b.Fatalf("WsProcessUpdateOrderbook must not error: %v", err)
				}
			}
		})
	}
}

func BenchmarkWsProcessOrderbook5(b *testing.B) {
	b.ReportAllocs()

	ob5payload := []byte(`{"arg":{"channel":"books5","instId":"OKB-USDT"},"data":[{"asks":[["0.0000007465","2290075956","0","4"],["0.0000007466","1747284705","0","4"],["0.0000007467","1338861655","0","3"],["0.0000007468","1661668387","0","6"],["0.0000007469","2715477116","0","5"]],"bids":[["0.0000007464","15693119","0","1"],["0.0000007463","2330835024","0","4"],["0.0000007462","1182926517","0","2"],["0.0000007461","3818684357","0","4"],["0.000000746","6021641435","0","7"]],"instId":"OKB-USDT","ts":"1695864901807","seqId":4826378794}]}`)

	ex := newBenchmarkOKXExchange(b)
	drainBenchmarkDataHandler(b, ex)

	b.ResetTimer()
	for b.Loop() {
		if err := ex.wsProcessOrderbook5(ob5payload); err != nil {
			b.Fatalf("wsProcessOrderbook5 must not error: %v", err)
		}
	}
}

func BenchmarkWsProcessSpreadOrderbook(b *testing.B) {
	b.ReportAllocs()

	payload := []byte(processSpreadOrderbookJSON)
	ex := newBenchmarkOKXExchange(b)
	drainBenchmarkDataHandler(b, ex)

	b.ResetTimer()
	for b.Loop() {
		if err := ex.wsProcessSpreadOrderbook(payload); err != nil {
			b.Fatalf("wsProcessSpreadOrderbook must not error: %v", err)
		}
	}
}
