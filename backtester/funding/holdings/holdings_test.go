package holdings

import (
	"errors"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const (
	testExchange = "binance"
	riskFreeRate = 0.03
)

func TestCreate(t *testing.T) {
	t.Parallel()
	_, err := Create(nil, -1, riskFreeRate)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("expected: %v, received %v", ErrInitialFundsZero, err)
	}

	_, err = Create(&fill.Fill{}, -1, riskFreeRate)
	if !errors.Is(err, ErrInitialFundsZero) {
		t.Errorf("expected: %v, received %v", ErrInitialFundsZero, err)
	}

	_, err = Create(nil, 1, riskFreeRate)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("expected: %v, received %v", common.ErrNilEvent, err)
	}

	h, err := Create(&fill.Fill{}, 1, riskFreeRate)
	if err != nil {
		t.Error(err)
	}
	if h.InitialFunds != 1 {
		t.Errorf("expected 1, received '%v'", h.InitialFunds)
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	h, err := Create(&fill.Fill{}, 1, riskFreeRate)
	if err != nil {
		t.Error(err)
	}
	t1 := h.Timestamp
	h.Update(&fill.Fill{
		Base: event.Base{
			Time: time.Now(),
		},
	})
	if t1.Equal(h.Timestamp) {
		t.Errorf("expected '%v' received '%v'", h.Timestamp, t1)
	}
}

func TestUpdateValue(t *testing.T) {
	t.Parallel()
	h, err := Create(&fill.Fill{}, 1, riskFreeRate)
	if err != nil {
		t.Error(err)
	}
	h.PositionsSize = 1
	h.UpdateValue(&kline.Kline{
		Close: 1337,
	})
	if h.PositionsValue != 1337 {
		t.Errorf("expected '%v' received '%v'", h.PositionsValue, 1337)
	}
}

func TestUpdateBuyStats(t *testing.T) {
	t.Parallel()
	h, err := Create(&fill.Fill{}, 1000, riskFreeRate)
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
		Amount:              1,
		ClosePrice:          500,
		VolumeAdjustedPrice: 500,
		PurchasePrice:       500,
		ExchangeFee:         0,
		Slippage:            0,
		Order: &order.Detail{
			Price:       500,
			Amount:      1,
			Exchange:    testExchange,
			ID:          "1337",
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
	})
	if err != nil {
		t.Error(err)
	}
	if h.PositionsSize != 1 {
		t.Errorf("expected '%v' received '%v'", 1, h.PositionsSize)
	}
	if h.PositionsValue != 500 {
		t.Errorf("expected '%v' received '%v'", 500, h.PositionsValue)
	}
	if h.InitialFunds != 1000 {
		t.Errorf("expected '%v' received '%v'", 1000, h.InitialFunds)
	}
	if h.RemainingFunds != 499 {
		t.Errorf("expected '%v' received '%v'", 499, h.RemainingFunds)
	}
	if h.TotalValue != 999 {
		t.Errorf("expected '%v' received '%v'", 999, h.TotalValue)
	}
	if h.BoughtAmount != 1 {
		t.Errorf("expected '%v' received '%v'", 1, h.BoughtAmount)
	}
	if h.BoughtValue != 500 {
		t.Errorf("expected '%v' received '%v'", 500, h.BoughtValue)
	}
	if h.SoldAmount != 0 {
		t.Errorf("expected '%v' received '%v'", 0, h.SoldAmount)
	}
	if h.TotalFees != 1 {
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
		Amount:              0.5,
		ClosePrice:          500,
		VolumeAdjustedPrice: 500,
		PurchasePrice:       500,
		ExchangeFee:         0,
		Slippage:            0,
		Order: &order.Detail{
			Price:       500,
			Amount:      0.5,
			Exchange:    testExchange,
			ID:          "1337",
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
	})
	if err != nil {
		t.Error(err)
	}
	if h.PositionsSize != 1.5 {
		t.Errorf("expected '%v' received '%v'", 1, h.PositionsSize)
	}
	if h.PositionsValue != 750 {
		t.Errorf("expected '%v' received '%v'", 750, h.PositionsValue)
	}
	if h.InitialFunds != 1000 {
		t.Errorf("expected '%v' received '%v'", 1000, h.InitialFunds)
	}
	if h.RemainingFunds != 248.5 {
		t.Errorf("expected '%v' received '%v'", 248.5, h.RemainingFunds)
	}
	if h.TotalValue != 998.5 {
		t.Errorf("expected '%v' received '%v'", 998.5, h.TotalValue)
	}
	if h.BoughtAmount != 1.5 {
		t.Errorf("expected '%v' received '%v'", 1, h.BoughtAmount)
	}
	if h.BoughtValue != 750 {
		t.Errorf("expected '%v' received '%v'", 750, h.BoughtValue)
	}
	if h.SoldAmount != 0 {
		t.Errorf("expected '%v' received '%v'", 0, h.SoldAmount)
	}
	if h.TotalFees != 1.5 {
		t.Errorf("expected '%v' received '%v'", 1.5, h.TotalFees)
	}
}

func TestUpdateSellStats(t *testing.T) {
	t.Parallel()
	h, err := Create(&fill.Fill{}, 1000, riskFreeRate)
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
		Amount:              1,
		ClosePrice:          500,
		VolumeAdjustedPrice: 500,
		PurchasePrice:       500,
		ExchangeFee:         0,
		Slippage:            0,
		Order: &order.Detail{
			Price:       500,
			Amount:      1,
			Exchange:    testExchange,
			ID:          "1337",
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
	})
	if err != nil {
		t.Error(err)
	}
	if h.PositionsSize != 1 {
		t.Errorf("expected '%v' received '%v'", 1, h.PositionsSize)
	}
	if h.PositionsValue != 500 {
		t.Errorf("expected '%v' received '%v'", 500, h.PositionsValue)
	}
	if h.InitialFunds != 1000 {
		t.Errorf("expected '%v' received '%v'", 1000, h.InitialFunds)
	}
	if h.RemainingFunds != 499 {
		t.Errorf("expected '%v' received '%v'", 499, h.RemainingFunds)
	}
	if h.TotalValue != 999 {
		t.Errorf("expected '%v' received '%v'", 999, h.TotalValue)
	}
	if h.BoughtAmount != 1 {
		t.Errorf("expected '%v' received '%v'", 1, h.BoughtAmount)
	}
	if h.BoughtValue != 500 {
		t.Errorf("expected '%v' received '%v'", 500, h.BoughtValue)
	}
	if h.SoldAmount != 0 {
		t.Errorf("expected '%v' received '%v'", 0, h.SoldAmount)
	}
	if h.TotalFees != 1 {
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
		Amount:              1,
		ClosePrice:          500,
		VolumeAdjustedPrice: 500,
		PurchasePrice:       500,
		ExchangeFee:         0,
		Slippage:            0,
		Order: &order.Detail{
			Price:       500,
			Amount:      1,
			Exchange:    testExchange,
			ID:          "1337",
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
	})

	if h.PositionsSize != 0 {
		t.Errorf("expected '%v' received '%v'", 0, h.PositionsSize)
	}
	if h.PositionsValue != 0 {
		t.Errorf("expected '%v' received '%v'", 0, h.PositionsValue)
	}
	if h.InitialFunds != 1000 {
		t.Errorf("expected '%v' received '%v'", 1000, h.InitialFunds)
	}
	if h.RemainingFunds != 998 {
		t.Errorf("expected '%v' received '%v'", 998, h.RemainingFunds)
	}
	if h.TotalValue != 998 {
		t.Errorf("expected '%v' received '%v'", 998, h.TotalValue)
	}
	if h.BoughtAmount != 1 {
		t.Errorf("expected '%v' received '%v'", 1, h.BoughtAmount)
	}
	if h.BoughtValue != 500 {
		t.Errorf("expected '%v' received '%v'", 500, h.BoughtValue)
	}
	if h.SoldAmount != 1 {
		t.Errorf("expected '%v' received '%v'", 1, h.SoldAmount)
	}
	if h.TotalFees != 2 {
		t.Errorf("expected '%v' received '%v'", 2, h.TotalFees)
	}
}
