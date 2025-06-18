package orderbook

import (
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/log"
)

// checker defines specific functionality to determine ascending/descending validation
type checker func(current, previous Level) error

// isAsc specifically defines ascending price check
var isAsc = func(current, previous Level) error {
	if current.Price < previous.Price {
		return errPriceOutOfOrder
	}
	return nil
}

// dsc specifically defines descending price check
var dsc = func(current, previous Level) error {
	if current.Price > previous.Price {
		return errPriceOutOfOrder
	}
	return nil
}

// Validate ensures that the orderbook items are correctly sorted and all fields are valid
// Bids should always go from a high price to a low price and Asks should always go from a low price to a higher price
func (b *Book) Validate() error {
	if !b.VerifyOrderbook {
		return nil
	}
	return validate(b)
}

func validate(b *Book) error {
	// Checking for both ask and bid lengths being zero has been removed and
	// a warning has been put in place for some exchanges that return zero
	// level books. In the event that there is a massive liquidity change where
	// a book dries up, this will still update so we do not traverse potential
	// incorrect old data.
	if (len(b.Asks) == 0 || len(b.Bids) == 0) && !b.Asset.IsOptions() {
		log.Warnf(log.OrderBook, bookLengthIssue, b.Exchange, b.Pair, b.Asset, len(b.Bids), len(b.Asks))
	}
	err := checkAlignment(b.Bids, b.IsFundingRate, b.PriceDuplication, b.IDAlignment, b.ChecksumStringRequired, dsc, b.Exchange)
	if err != nil {
		return fmt.Errorf(bidLoadBookFailure, b.Exchange, b.Pair, b.Asset, err)
	}
	err = checkAlignment(b.Asks, b.IsFundingRate, b.PriceDuplication, b.IDAlignment, b.ChecksumStringRequired, asc, b.Exchange)
	if err != nil {
		return fmt.Errorf(askLoadBookFailure, b.Exchange, b.Pair, b.Asset, err)
	}
	return nil
}

// checkAlignment validates full orderbook
func checkAlignment(depth Levels, fundingRate, priceDuplication, isIDAligned, requiresChecksumString bool, c checker, exch string) error {
	for i := range depth {
		if depth[i].Price == 0 {
			switch {
			case exch == "Bitfinex" && fundingRate: /* funding rate can be 0 it seems on Bitfinex */
			default:
				return ErrPriceZero
			}
		}

		if depth[i].Amount <= 0 {
			return errAmountInvalid
		}
		if fundingRate && depth[i].Period == 0 {
			return errPeriodUnset
		}
		if requiresChecksumString && (depth[i].StrAmount == "" || depth[i].StrPrice == "") {
			return errChecksumStringNotSet
		}

		if i != 0 {
			prev := i - 1
			if err := c(depth[i], depth[prev]); err != nil {
				return err
			}
			if isIDAligned && depth[i].ID < depth[prev].ID {
				return errIDOutOfOrder
			}
			if !priceDuplication && depth[i].Price == depth[prev].Price {
				return errDuplication
			}
			if depth[i].ID != 0 && depth[i].ID == depth[prev].ID {
				return errIDDuplication
			}
		}
	}
	return nil
}
