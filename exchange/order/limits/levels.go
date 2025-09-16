package limits

import (
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Validate ensures MinMaxLevel fields are valid
func (m *MinMaxLevel) Validate(price, amount float64, orderType order.Type) error {
	// TODO: Verify Quote as well as Base amounts
	if m == nil {
		return nil
	}

	if m.MinimumBaseAmount != 0 && amount < m.MinimumBaseAmount {
		return fmt.Errorf("%w min: %.8f supplied %.8f", ErrAmountBelowMin, m.MinimumBaseAmount, amount)
	}
	if m.MaximumBaseAmount != 0 && amount > m.MaximumBaseAmount {
		return fmt.Errorf("%w min: %.8f supplied %.8f", ErrAmountExceedsMax, m.MaximumBaseAmount, amount)
	}
	if m.AmountStepIncrementSize != 0 {
		dAmount := decimal.NewFromFloat(amount)
		dStep := decimal.NewFromFloat(m.AmountStepIncrementSize)
		if !dAmount.Mod(dStep).IsZero() {
			return fmt.Errorf("%w stepSize: %.8f supplied %.8f", ErrAmountExceedsStep, m.AmountStepIncrementSize, amount)
		}
	}

	/*
		ContractMultiplier checking not done due to the fact we need coherence with the
		 last average price (TODO)
		 m.multiplierUp will be used to determine how far our price can go up
		 m.multiplierDown will be used to determine how far our price can go down
		 m.averagePriceMinutes will be used to determine mean over this period

		 Max iceberg parts checking not done as we do not have that
		 functionality yet (TODO)
		 m.maxIcebergParts // How many components in an iceberg order

		 Max total orders not done due to order manager limitations (TODO)
		 m.maxTotalOrders

		 Max algo orders not done due to order manager limitations (TODO)
		 m.maxAlgoOrders

		 If order type is Market we do not need to do price checks
	*/
	if orderType != order.Market {
		if m.MinPrice != 0 && price < m.MinPrice {
			return fmt.Errorf("%w min: %.8f supplied %.8f", ErrPriceBelowMin, m.MinPrice, price)
		}
		if m.MaxPrice != 0 && price > m.MaxPrice {
			return fmt.Errorf("%w max: %.8f supplied %.8f", ErrPriceExceedsMax, m.MaxPrice, price)
		}
		if m.MinNotional != 0 && (amount*price) < m.MinNotional {
			return fmt.Errorf("%w minimum notional: %.8f value of order %.8f", ErrNotionalValue, m.MinNotional, amount*price)
		}
		if m.PriceStepIncrementSize != 0 {
			dPrice := decimal.NewFromFloat(price)
			dMinPrice := decimal.NewFromFloat(m.MinPrice)
			dStep := decimal.NewFromFloat(m.PriceStepIncrementSize)
			if !dPrice.Sub(dMinPrice).Mod(dStep).IsZero() {
				return fmt.Errorf("%w stepSize: %.8f supplied %.8f", ErrPriceExceedsStep, m.PriceStepIncrementSize, price)
			}
		}
		return nil
	}

	if m.MarketMinQty != 0 && m.MinimumBaseAmount < m.MarketMinQty && amount < m.MarketMinQty {
		return fmt.Errorf("%w min: %.8f supplied %.8f", ErrMarketAmountBelowMin, m.MarketMinQty, amount)
	}
	if m.MarketMaxQty != 0 && m.MaximumBaseAmount > m.MarketMaxQty && amount > m.MarketMaxQty {
		return fmt.Errorf("%w max: %.8f supplied %.8f", ErrMarketAmountExceedsMax, m.MarketMaxQty, amount)
	}
	if m.MarketStepIncrementSize != 0 && m.AmountStepIncrementSize != m.MarketStepIncrementSize {
		dAmount := decimal.NewFromFloat(amount)
		dMinMAmount := decimal.NewFromFloat(m.MarketMinQty)
		dStep := decimal.NewFromFloat(m.MarketStepIncrementSize)
		if !dAmount.Sub(dMinMAmount).Mod(dStep).IsZero() {
			return fmt.Errorf("%w stepSize: %.8f supplied %.8f", ErrMarketAmountExceedsStep, m.MarketStepIncrementSize, amount)
		}
	}
	return nil
}

// FloorAmountToStepIncrementDecimal floors decimal amount to step increment
func (m *MinMaxLevel) FloorAmountToStepIncrementDecimal(amount decimal.Decimal) decimal.Decimal {
	if m == nil {
		return amount
	}

	dStep := decimal.NewFromFloat(m.AmountStepIncrementSize)
	if dStep.IsZero() || amount.Equal(dStep) {
		return amount
	}

	if amount.LessThan(dStep) {
		return decimal.Zero
	}
	mod := amount.Mod(dStep)
	// subtract to get the floor
	return amount.Sub(mod)
}

// FloorAmountToStepIncrement floors float amount to step increment
func (m *MinMaxLevel) FloorAmountToStepIncrement(amount float64) float64 {
	if m == nil {
		return amount
	}

	if m.AmountStepIncrementSize == 0 || amount == m.AmountStepIncrementSize {
		return amount
	}

	if amount < m.AmountStepIncrementSize {
		return 0
	}

	dAmount := decimal.NewFromFloat(amount)
	dStep := decimal.NewFromFloat(m.AmountStepIncrementSize)
	mod := dAmount.Mod(dStep)
	// subtract to get the floor
	return dAmount.Sub(mod).InexactFloat64()
}

// FloorPriceToStepIncrement floors float price to step increment
func (m *MinMaxLevel) FloorPriceToStepIncrement(price float64) float64 {
	if m == nil {
		return price
	}

	if m.PriceStepIncrementSize == 0 {
		return price
	}

	if price < m.PriceStepIncrementSize {
		return 0
	}

	dPrice := decimal.NewFromFloat(price)
	dStep := decimal.NewFromFloat(m.PriceStepIncrementSize)
	mod := dPrice.Mod(dStep)
	// subtract to get the floor
	return dPrice.Sub(mod).InexactFloat64()
}
