package main

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
)

type orderbookCountingWriter struct {
	writes int
}

func (w *orderbookCountingWriter) Write(p []byte) (int, error) {
	w.writes++
	return len(p), nil
}

func BenchmarkOrderbookRendering(b *testing.B) {
	for _, stringValues := range []bool{false, true} {
		book := &gctrpc.OrderbookResponse{Pair: &gctrpc.CurrencyPair{Base: "BTC", Quote: "USD"}}
		for i := range 100 {
			book.Bids = append(book.Bids, &gctrpc.OrderbookItem{Price: 60000 - float64(i), Amount: 0.125})
			book.Asks = append(book.Asks, &gctrpc.OrderbookItem{Price: 60001 + float64(i), Amount: 0.25})
			if stringValues {
				book.Bids[i].StrPrice = fmt.Sprintf("%.8f", book.Bids[i].Price)
				book.Bids[i].StrAmount = "0.12500000"
				book.Asks[i].StrPrice = fmt.Sprintf("%.8f", book.Asks[i].Price)
				book.Asks[i].StrAmount = "0.25000000"
			}
		}
		for _, depth := range []int{10, 50, 100} {
			for _, exchangeStyle := range []bool{false, true} {
				b.Run(fmt.Sprintf("strings=%t/depth=%d/exchangeStyle=%t", stringValues, depth, exchangeStyle), func(b *testing.B) {
					var writer orderbookCountingWriter
					b.ReportAllocs()
					for b.Loop() {
						if exchangeStyle {
							if err := renderOrderbookExchangeStyle(&writer, book, "test", "spot", depth); err != nil {
								b.Fatal(err)
							}
						} else {
							if err := renderOrderbookStream(&writer, book, "test", depth); err != nil {
								b.Fatal(err)
							}
						}
					}
					b.ReportMetric(float64(writer.writes)/float64(b.N), "writes/frame")
				})
			}
		}
	}
}

var _ io.Writer = (*orderbookCountingWriter)(nil)

type orderbookErrorWriter struct {
	err error
}

func (w orderbookErrorWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func TestOrderbookRenderingWrites(t *testing.T) {
	t.Parallel()
	book := &gctrpc.OrderbookResponse{Pair: &gctrpc.CurrencyPair{Base: "BTC", Quote: "USD"}}
	for _, exchangeStyle := range []bool{false, true} {
		t.Run(fmt.Sprintf("exchangeStyle=%t", exchangeStyle), func(t *testing.T) {
			t.Parallel()
			render := func(w io.Writer) error {
				if exchangeStyle {
					return renderOrderbookExchangeStyle(w, book, "test", "spot", 50)
				}
				return renderOrderbookStream(w, book, "test", 50)
			}
			var writer orderbookCountingWriter
			require.NoError(t, render(&writer), "writing the frame must succeed")
			assert.Equal(t, 1, writer.writes, "the complete frame should use one write even for an empty book")
			require.ErrorIs(t, render(orderbookErrorWriter{err: io.ErrClosedPipe}), io.ErrClosedPipe, "writer errors must be returned")
			require.ErrorIs(t, render(orderbookErrorWriter{}), io.ErrShortWrite, "incomplete writes must be reported")
		})
	}
}

func TestOrderbookDisplayLevels(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		item     gctrpc.OrderbookItem
		expected orderbookDisplayLevel
	}{
		{name: "strings take precedence", item: gctrpc.OrderbookItem{Price: 1, Amount: 2, StrPrice: "123456789012345.123456789012345", StrAmount: "0.0000000012300"}, expected: orderbookDisplayLevel{price: "123456789012345.123456789012345", amount: "0.0000000012300"}},
		{name: "price string only", item: gctrpc.OrderbookItem{Price: 1, Amount: 0.125, StrPrice: "1.0000"}, expected: orderbookDisplayLevel{price: "1.0000", amount: "0.125"}},
		{name: "amount string only", item: gctrpc.OrderbookItem{Price: 0.000000001, Amount: 2, StrAmount: "2.000"}, expected: orderbookDisplayLevel{price: "0.000000001", amount: "2.000"}},
		{name: "float precision", item: gctrpc.OrderbookItem{Price: 1.234567891234567, Amount: 0.000000000001}, expected: orderbookDisplayLevel{price: "1.234567891234567", amount: "0.000000000001"}},
		{name: "integer fallback", item: gctrpc.OrderbookItem{Price: 1200, Amount: 100}, expected: orderbookDisplayLevel{price: "1200", amount: "100"}},
		{name: "zero fallback", expected: orderbookDisplayLevel{price: "0", amount: "0"}},
		{name: "explicit zero strings", item: gctrpc.OrderbookItem{StrPrice: "0.000", StrAmount: "0.00"}, expected: orderbookDisplayLevel{price: "0.000", amount: "0.00"}},
	}
	for i := range tests {
		tc := &tests[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, []orderbookDisplayLevel{tc.expected}, orderbookDisplayLevels([]*gctrpc.OrderbookItem{&tc.item}, 1), "display values should preserve supplied strings and independently fall back to compact decimals")
			book := &gctrpc.OrderbookResponse{Pair: &gctrpc.CurrencyPair{Base: "BTC", Quote: "USD"}, Bids: []*gctrpc.OrderbookItem{&tc.item}}
			for _, exchangeStyle := range []bool{false, true} {
				var output strings.Builder
				if exchangeStyle {
					require.NoError(t, renderOrderbookExchangeStyle(&output, book, "test", "spot", 1), "exchange-style rendering must succeed")
				} else {
					require.NoError(t, renderOrderbookStream(&output, book, "test", 1), "side-by-side rendering must succeed")
				}
				assert.Contains(t, output.String(), tc.expected.price, "both layouts should use the selected price representation")
				assert.Contains(t, output.String(), tc.expected.amount, "both layouts should use the selected amount representation")
			}
		})
	}
}

func TestOrderbookRenderingAlignment(t *testing.T) {
	t.Parallel()
	book := &gctrpc.OrderbookResponse{
		Pair: &gctrpc.CurrencyPair{Base: "BTC", Quote: "USD"},
		Bids: []*gctrpc.OrderbookItem{{Price: 123456, Amount: 1234}, {Price: 1, Amount: 0}},
		Asks: []*gctrpc.OrderbookItem{{Price: 234567, Amount: 5678}},
	}
	var output strings.Builder
	require.NoError(t, renderOrderbookStream(&output, book, "test", 50), "rendering must succeed")
	lines := strings.Split(output.String(), "\n")
	require.Len(t, lines, 7, "the frame must include its title, headings and both rows")
	assert.Equal(t, "Price(USD)  Amount(BTC) | Price(USD)  Amount(BTC)", lines[3], "currency headings should align with the numeric columns")
	assert.Equal(t, "    123456         1234 |     234567         5678", lines[4], "large values should fit without tabs")
	assert.Equal(t, "         1            0 |                        ", lines[5], "missing asks should be blank while genuine zero amounts remain visible")
	output.Reset()
	require.NoError(t, renderOrderbookExchangeStyle(&output, book, "test", "spot", 50), "exchange-style rendering must succeed")
	lines = strings.Split(output.String(), "\n")
	require.Len(t, lines, 8, "the frame must contain only actual asks and bids")
	assert.Equal(t, grayText+"Price(USD)  Amount(BTC)", lines[1], "exchange-style headings should use the same column widths")
	assert.Equal(t, redText+"    234567         5678", lines[2], "the ask should align with the headings")
	assert.Equal(t, greenText+"    123456         1234", lines[4], "the best bid should follow the separator")
	assert.Equal(t, greenText+"         1            0", lines[5], "short values should remain right-aligned")
	for _, exchangeStyle := range []bool{false, true} {
		var writer orderbookCountingWriter
		if exchangeStyle {
			require.NoError(t, renderOrderbookExchangeStyle(&writer, book, "test", "spot", 50), "exchange-style rendering must succeed")
		} else {
			require.NoError(t, renderOrderbookStream(&writer, book, "test", 50), "side-by-side rendering must succeed")
		}
		assert.Equal(t, 1, writer.writes, "populated frames should use one write")
	}
}

func TestOrderbookRenderingPrecision(t *testing.T) {
	t.Parallel()
	for _, stringValues := range []bool{false, true} {
		for _, exchangeStyle := range []bool{false, true} {
			t.Run(fmt.Sprintf("strings=%t/exchangeStyle=%t", stringValues, exchangeStyle), func(t *testing.T) {
				t.Parallel()
				book := &gctrpc.OrderbookResponse{
					Pair: &gctrpc.CurrencyPair{Base: "ORDI", Quote: "USDT"},
					Bids: []*gctrpc.OrderbookItem{{Price: 4.08, Amount: 4}, {Price: 4, Amount: 0}},
					Asks: []*gctrpc.OrderbookItem{{Price: 4.101, Amount: 6.88}, {Price: 4.2, Amount: 14.4}, {Price: 5.12345678, Amount: 7.12345678}},
				}
				if stringValues {
					book.Bids[0].StrPrice, book.Bids[0].StrAmount = "4.08", "4"
					book.Bids[1].StrPrice, book.Bids[1].StrAmount = "4", "0"
					book.Asks[0].StrPrice, book.Asks[0].StrAmount = "4.101", "6.88"
					book.Asks[1].StrPrice, book.Asks[1].StrAmount = "4.2", "14.4"
				}
				var output strings.Builder
				if exchangeStyle {
					require.NoError(t, renderOrderbookExchangeStyle(&output, book, "kucoin", "spot", 2), "exchange-style rendering must succeed")
					assert.Contains(t, output.String(), greenText+"      4.080          4.00\n", "the bid should use the ask's price and amount precision")
					assert.Contains(t, output.String(), greenText+"      4.000          0.00\n", "whole prices and zero amounts should be padded")
					assert.Contains(t, output.String(), redText+"      4.200         14.40\n", "asks should use the same precision as bids")
				} else {
					require.NoError(t, renderOrderbookStream(&output, book, "kucoin", 2), "side-by-side rendering must succeed")
					assert.Contains(t, output.String(), "      4.080          4.00 |       4.101          6.88\n", "prices and amounts should be padded independently across both sides")
					assert.Contains(t, output.String(), "      4.000          0.00 |       4.200         14.40\n", "decimal points should align on both sides")
				}
				assert.NotContains(t, output.String(), "12345678", "hidden levels should not affect display precision")
				assert.Equal(t, 4.08, book.Bids[0].Price, "rendering should preserve the numeric source")
				if stringValues {
					assert.Equal(t, "4.08", book.Bids[0].StrPrice, "padding should not alter the supplied price string")
					assert.Equal(t, "4", book.Bids[0].StrAmount, "padding should not alter the supplied amount string")
				}
			})
		}
	}
}

func TestNormaliseOrderbookPrecision(t *testing.T) {
	t.Parallel()
	bids := []orderbookDisplayLevel{{price: "1.0000", amount: "0.0000000012300"}, {price: "0", amount: "2"}}
	asks := []orderbookDisplayLevel{{price: "1.234567891234567890", amount: "3.50"}, {}, {price: "1e-8", amount: "NaN"}}
	normaliseOrderbookPrecision(bids, asks)
	assert.Equal(t, []orderbookDisplayLevel{
		{price: "1.000000000000000000", amount: "0.0000000012300"},
		{price: "0.000000000000000000", amount: "2.0000000000000"},
	}, bids, "padding should preserve all supplied precision including trailing zeros")
	assert.Equal(t, []orderbookDisplayLevel{
		{price: "1.234567891234567890", amount: "3.5000000000000"}, {}, {price: "1e-8", amount: "NaN"},
	}, asks, "padding should not round precise values or alter missing and non-fixed-point values")
}

func TestOrderbookRenderingDepth(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                    string
		bids, asks, limit, rows int
	}{
		{name: "empty", limit: 50},
		{name: "single level", bids: 1, asks: 1, limit: 50, rows: 1},
		{name: "last level", bids: 3, asks: 3, limit: 50, rows: 3},
		{name: "bids only", bids: 3, limit: 50, rows: 3},
		{name: "asks only", asks: 3, limit: 50, rows: 3},
		{name: "unequal sides", bids: 1, asks: 3, limit: 50, rows: 3},
		{name: "more bids", bids: 3, asks: 1, limit: 50, rows: 3},
		{name: "at limit", bids: 3, asks: 3, limit: 3, rows: 3},
		{name: "custom limit", bids: 3, asks: 3, limit: 2, rows: 2},
		{name: "zero limit", bids: 3, asks: 3},
		{name: "negative limit", bids: 3, asks: 3, limit: -1},
		{name: "stream ceiling", bids: 51, asks: 51, limit: 50, rows: 50},
		{name: "snapshot ceiling", bids: 101, asks: 101, limit: 100, rows: 100},
	} {
		for _, exchangeStyle := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/exchangeStyle=%t", tc.name, exchangeStyle), func(t *testing.T) {
				t.Parallel()
				book := &gctrpc.OrderbookResponse{Pair: &gctrpc.CurrencyPair{Base: "BTC", Quote: "USD"}}
				for i := range tc.bids {
					book.Bids = append(book.Bids, &gctrpc.OrderbookItem{Price: float64(100 - i), Amount: float64(i + 1)})
				}
				for i := range tc.asks {
					book.Asks = append(book.Asks, &gctrpc.OrderbookItem{Price: float64(200 + i), Amount: float64(i + 2)})
				}
				var output strings.Builder
				if exchangeStyle {
					require.NoError(t, renderOrderbookExchangeStyle(&output, book, "test", "spot", tc.limit), "rendering must succeed")
				} else {
					require.NoError(t, renderOrderbookStream(&output, book, "test", tc.limit), "rendering must succeed")
				}
				var got []string
				for line := range strings.SplitSeq(output.String(), "\n") {
					if strings.HasPrefix(line, redText) || strings.HasPrefix(line, greenText) {
						line = strings.TrimPrefix(strings.TrimPrefix(line, redText), greenText)
						got = append(got, strings.Join(strings.Fields(line), " "))
					} else if strings.Contains(line, " | ") && !strings.Contains(line, "Bids") && !strings.Contains(line, "Price(") {
						sides := strings.Split(line, " | ")
						got = append(got, strings.Join(strings.Fields(sides[0]), " ")+" | "+strings.Join(strings.Fields(sides[1]), " "))
					}
				}
				var expected []string
				if exchangeStyle {
					for i := min(tc.rows, tc.asks); i > 0; i-- {
						expected = append(expected, fmt.Sprintf("%d %d", 200+i-1, i+1))
					}
					for i := range min(tc.rows, tc.bids) {
						expected = append(expected, fmt.Sprintf("%d %d", 100-i, i+1))
					}
				} else {
					for i := range tc.rows {
						var bid, ask string
						if i < tc.bids {
							bid = fmt.Sprintf("%d %d", 100-i, i+1)
						}
						if i < tc.asks {
							ask = fmt.Sprintf("%d %d", 200+i, i+2)
						}
						expected = append(expected, bid+" | "+ask)
					}
				}
				assert.Equal(t, expected, got, "rendering should retain every level up to the limit in display order")
				assert.Len(t, book.Bids, tc.bids, "rendering should not truncate the underlying bids")
				assert.Len(t, book.Asks, tc.asks, "rendering should not truncate the underlying asks")
			})
		}
	}
}
