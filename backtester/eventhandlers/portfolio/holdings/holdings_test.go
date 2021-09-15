package holdings

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const (
	testExchange = "binance"
)

var (
	riskFreeRate = decimal.NewFromFloat(0.03)
)

func pair(t *testing.T) *funding.Pair {
	t.Helper()
	b, err := funding.CreateItem(testExchange, asset.Spot, currency.BTC, decimal.Zero, decimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	q, err := funding.CreateItem(testExchange, asset.Spot, currency.USDT, decimal.NewFromInt(1337), decimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	p, err := funding.CreatePair(b, q)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCreate(t *testing.T) {
	t.Parallel()
	_, err := Create(nil, pair(t), riskFreeRate)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}
	_, err = Create(&fill.Fill{}, pair(t), riskFreeRate)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	h, err := Create(&fill.Fill{}, pair(t), riskFreeRate)
	if err != nil {
		t.Error(err)
	}
	t1 := h.Timestamp
	h.Update(&fill.Fill{
		Base: event.Base{
			Time: time.Now(),
		},
	}, pair(t))

	if t1.Equal(h.Timestamp) {
		t.Errorf("expected '%v' received '%v'", h.Timestamp, t1)
	}
}

func TestUpdateValue(t *testing.T) {
	t.Parallel()
	h, err := Create(&fill.Fill{}, pair(t), riskFreeRate)
	if err != nil {
		t.Error(err)
	}
	h.BaseSize = decimal.NewFromInt(1)
	h.UpdateValue(&kline.Kline{
		Close: decimal.NewFromInt(1337),
	})
	if !h.BaseValue.Equal(decimal.NewFromInt(1337)) {
		t.Errorf("expected '%v' received '%v'", h.BaseSize, decimal.NewFromInt(1337))
	}
}

func TestUpdateBuyStats(t *testing.T) {
	t.Parallel()
	b, err := funding.CreateItem(testExchange, asset.Spot, currency.BTC, decimal.NewFromInt(1), decimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	q, err := funding.CreateItem(testExchange, asset.Spot, currency.USDT, decimal.NewFromInt(100), decimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	p, err := funding.CreatePair(b, q)
	if err != nil {
		t.Fatal(err)
	}
	h, err := Create(&fill.Fill{}, p, riskFreeRate)
	if err != nil {
		t.Error(err)
	}

	h.update(&fill.Fill{
		Base: event.Base{
			Exchange:     testExchange,
			Time:         time.Now(),
			Interval:     gctkline.OneHour,
			CurrencyPair: currency.NewPair(currency.BTC, currency.USDT),
			AssetType:    asset.Spot,
		},
		Direction:           order.Buy,
		Amount:              decimal.NewFromInt(1),
		ClosePrice:          decimal.NewFromInt(500),
		VolumeAdjustedPrice: decimal.NewFromInt(500),
		PurchasePrice:       decimal.NewFromInt(500),
		ExchangeFee:         decimal.Zero,
		Slippage:            decimal.Zero,
		Order: &order.Detail{
			Price:       500,
			Amount:      1,
			Exchange:    testExchange,
			ID:          "decimal.NewFromInt(1337)",
			Type:        order.Limit,
			Side:        order.Buy,
			Status:      order.New,
			AssetType:   asset.Spot,
			Date:        time.Now(),
			CloseTime:   time.Now(),
			LastUpdated: time.Now(),
			Pair:        currency.NewPair(currency.BTC, currency.USDT),
			Trades:      nil,
			Fee:         1,
		},
	}, p)
	if err != nil {
		t.Error(err)
	}
	if !h.BaseSize.Equal(p.BaseAvailable()) {
		t.Errorf("expected '%v' received '%v'", 1, h.BaseSize)
	}
	if !h.BaseValue.Equal(p.BaseAvailable().Mul(decimal.NewFromInt(500))) {
		t.Errorf("expected '%v' received '%v'", 500, h.BaseValue)
	}
	if !h.QuoteSize.Equal(decimal.NewFromInt(100)) {
		t.Errorf("expected '%v' received '%v'", 100, h.QuoteSize)
	}
	if !h.TotalValue.Equal(decimal.NewFromInt(600)) {
		t.Errorf("expected '%v' received '%v'", 999, h.TotalValue)
	}
	if !h.BoughtAmount.Equal(decimal.NewFromInt(1)) {
		t.Errorf("expected '%v' received '%v'", 1, h.BoughtAmount)
	}
	if !h.BoughtValue.Equal(decimal.NewFromInt(500)) {
		t.Errorf("expected '%v' received '%v'", 500, h.BoughtValue)
	}
	if !h.SoldAmount.Equal(decimal.Zero) {
		t.Errorf("expected '%v' received '%v'", 0, h.SoldAmount)
	}
	if !h.TotalFees.Equal(decimal.NewFromInt(1)) {
		t.Errorf("expected '%v' received '%v'", 1, h.TotalFees)
	}

	h.update(&fill.Fill{
		Base: event.Base{
			Exchange:     testExchange,
			Time:         time.Now(),
			Interval:     gctkline.OneHour,
			CurrencyPair: currency.NewPair(currency.BTC, currency.USDT),
			AssetType:    asset.Spot,
		},
		Direction:           order.Buy,
		Amount:              decimal.NewFromFloat(0.5),
		ClosePrice:          decimal.NewFromInt(500),
		VolumeAdjustedPrice: decimal.NewFromInt(500),
		PurchasePrice:       decimal.NewFromInt(500),
		ExchangeFee:         decimal.Zero,
		Slippage:            decimal.Zero,
		Order: &order.Detail{
			Price:       500,
			Amount:      0.5,
			Exchange:    testExchange,
			ID:          "decimal.NewFromInt(1337)",
			Type:        order.Limit,
			Side:        order.Buy,
			Status:      order.New,
			AssetType:   asset.Spot,
			Date:        time.Now(),
			CloseTime:   time.Now(),
			LastUpdated: time.Now(),
			Pair:        currency.NewPair(currency.BTC, currency.USDT),
			Trades:      nil,
			Fee:         0.5,
		},
	}, p)
	if err != nil {
		t.Error(err)
	}

	if !h.BoughtAmount.Equal(decimal.NewFromFloat(1.5)) {
		t.Errorf("expected '%v' received '%v'", 1, h.BoughtAmount)
	}
	if !h.BoughtValue.Equal(decimal.NewFromInt(750)) {
		t.Errorf("expected '%v' received '%v'", 750, h.BoughtValue)
	}
	if !h.SoldAmount.Equal(decimal.Zero) {
		t.Errorf("expected '%v' received '%v'", 0, h.SoldAmount)
	}
	if !h.TotalFees.Equal(decimal.NewFromFloat(1.5)) {
		t.Errorf("expected '%v' received '%v'", 1.5, h.TotalFees)
	}
}

func TestUpdateSellStats(t *testing.T) {
	t.Parallel()
	b, err := funding.CreateItem(testExchange, asset.Spot, currency.BTC, decimal.NewFromInt(1), decimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	q, err := funding.CreateItem(testExchange, asset.Spot, currency.USDT, decimal.NewFromInt(100), decimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	p, err := funding.CreatePair(b, q)
	if err != nil {
		t.Fatal(err)
	}
	h, err := Create(&fill.Fill{}, p, riskFreeRate)
	if err != nil {
		t.Error(err)
	}
	h.update(&fill.Fill{
		Base: event.Base{
			Exchange:     testExchange,
			Time:         time.Now(),
			Interval:     gctkline.OneHour,
			CurrencyPair: currency.NewPair(currency.BTC, currency.USDT),
			AssetType:    asset.Spot,
		},
		Direction:           order.Buy,
		Amount:              decimal.NewFromInt(1),
		ClosePrice:          decimal.NewFromInt(500),
		VolumeAdjustedPrice: decimal.NewFromInt(500),
		PurchasePrice:       decimal.NewFromInt(500),
		ExchangeFee:         decimal.Zero,
		Slippage:            decimal.Zero,
		Order: &order.Detail{
			Price:       500,
			Amount:      1,
			Exchange:    testExchange,
			ID:          "decimal.NewFromInt(1337)",
			Type:        order.Limit,
			Side:        order.Buy,
			Status:      order.New,
			AssetType:   asset.Spot,
			Date:        time.Now(),
			CloseTime:   time.Now(),
			LastUpdated: time.Now(),
			Pair:        currency.NewPair(currency.BTC, currency.USDT),
			Trades:      nil,
			Fee:         1,
		},
	}, p)
	if err != nil {
		t.Error(err)
	}
	if !h.BaseSize.Equal(decimal.NewFromInt(1)) {
		t.Errorf("expected '%v' received '%v'", 1, h.BaseSize)
	}
	if !h.BaseValue.Equal(decimal.NewFromInt(500)) {
		t.Errorf("expected '%v' received '%v'", 500, h.BaseValue)
	}
	if !h.QuoteInitialFunds.Equal(decimal.NewFromInt(100)) {
		t.Errorf("expected '%v' received '%v'", 100, h.QuoteInitialFunds)
	}
	if !h.QuoteSize.Equal(decimal.NewFromInt(100)) {
		t.Errorf("expected '%v' received '%v'", 100, h.QuoteSize)
	}
	if !h.TotalValue.Equal(decimal.NewFromInt(600)) {
		t.Errorf("expected '%v' received '%v'", 600, h.TotalValue)
	}
	if !h.BoughtAmount.Equal(decimal.NewFromInt(1)) {
		t.Errorf("expected '%v' received '%v'", 1, h.BoughtAmount)
	}
	if !h.BoughtValue.Equal(decimal.NewFromInt(500)) {
		t.Errorf("expected '%v' received '%v'", 500, h.BoughtValue)
	}
	if !h.SoldAmount.Equal(decimal.Zero) {
		t.Errorf("expected '%v' received '%v'", 0, h.SoldAmount)
	}
	if !h.TotalFees.Equal(decimal.NewFromInt(1)) {
		t.Errorf("expected '%v' received '%v'", 1, h.TotalFees)
	}

	h.update(&fill.Fill{
		Base: event.Base{
			Exchange:     testExchange,
			Time:         time.Now(),
			Interval:     gctkline.OneHour,
			CurrencyPair: currency.NewPair(currency.BTC, currency.USDT),
			AssetType:    asset.Spot,
		},
		Direction:           order.Sell,
		Amount:              decimal.NewFromInt(1),
		ClosePrice:          decimal.NewFromInt(500),
		VolumeAdjustedPrice: decimal.NewFromInt(500),
		PurchasePrice:       decimal.NewFromInt(500),
		ExchangeFee:         decimal.Zero,
		Slippage:            decimal.Zero,
		Order: &order.Detail{
			Price:       500,
			Amount:      1,
			Exchange:    testExchange,
			ID:          "decimal.NewFromInt(1337)",
			Type:        order.Limit,
			Side:        order.Sell,
			Status:      order.New,
			AssetType:   asset.Spot,
			Date:        time.Now(),
			CloseTime:   time.Now(),
			LastUpdated: time.Now(),
			Pair:        currency.NewPair(currency.BTC, currency.USDT),
			Trades:      nil,
			Fee:         1,
		},
	}, p)

	if !h.BoughtAmount.Equal(decimal.NewFromInt(1)) {
		t.Errorf("expected '%v' received '%v'", 1, h.BoughtAmount)
	}
	if !h.BoughtValue.Equal(decimal.NewFromInt(500)) {
		t.Errorf("expected '%v' received '%v'", 500, h.BoughtValue)
	}
	if !h.SoldAmount.Equal(decimal.NewFromInt(1)) {
		t.Errorf("expected '%v' received '%v'", 1, h.SoldAmount)
	}
	if !h.TotalFees.Equal(decimal.NewFromInt(2)) {
		t.Errorf("expected '%v' received '%v'", 2, h.TotalFees)
	}
}
