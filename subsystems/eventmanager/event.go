package eventmanager

import (
	"errors"
	"fmt"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// String turns the structure event into a string
func (e *Event) String() string {
	return fmt.Sprintf(
		"If the %s [%s] %s on %s meets the following %v then %s.", e.Pair.String(),
		strings.ToUpper(e.Asset.String()), e.Item, e.Exchange, e.Condition, e.Action,
	)
}

func (e *Event) processTicker() error {
	t, err := ticker.GetTicker(e.Exchange, e.Pair, e.Asset)
	if err != nil {
		return fmt.Errorf("failed to get ticker. Err: %w", err)
	}

	if t.Last == 0 {
		return errTickerLastPriceZero
	}
	return e.shouldProcessEvent(t.Last, e.Condition.Price)
}

func (e *Event) shouldProcessEvent(actual, threshold float64) error {
	switch e.Condition.Condition {
	case ConditionGreaterThan:
		if actual > threshold {
			return nil
		}
	case ConditionGreaterThanOrEqual:
		if actual >= threshold {
			return nil
		}
	case ConditionLessThan:
		if actual < threshold {
			return nil
		}
	case ConditionLessThanOrEqual:
		if actual <= threshold {
			return nil
		}
	case ConditionIsEqual:
		if actual == threshold {
			return nil
		}
	}
	return errors.New("does not meet conditions")
}

func (e *Event) processOrderbook() error {
	ob, err := orderbook.Get(e.Exchange, e.Pair, e.Asset)
	if err != nil {
		return fmt.Errorf("events: Failed to get orderbook. Err: %w", err)
	}

	if e.Condition.CheckBids || e.Condition.CheckBidsAndAsks {
		for x := range ob.Bids {
			subtotal := ob.Bids[x].Amount * ob.Bids[x].Price
			err := e.shouldProcessEvent(subtotal, e.Condition.OrderbookAmount)
			if err == nil {
				log.Debugf(log.EventMgr, "Events: Bid Amount: %f Price: %v Subtotal: %v\n", ob.Bids[x].Amount, ob.Bids[x].Price, subtotal)
			}
		}
	}

	if !e.Condition.CheckBids || e.Condition.CheckBidsAndAsks {
		for x := range ob.Asks {
			subtotal := ob.Asks[x].Amount * ob.Asks[x].Price
			err := e.shouldProcessEvent(subtotal, e.Condition.OrderbookAmount)
			if err == nil {
				log.Debugf(log.EventMgr, "Events: Ask Amount: %f Price: %v Subtotal: %v\n", ob.Asks[x].Amount, ob.Asks[x].Price, subtotal)
			}
		}
	}
	return err
}
