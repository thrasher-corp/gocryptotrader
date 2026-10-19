package bitfinex

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

var (
	errSymbolUnparsable    = errors.New("symbol cannot be split into a pair")
	errUnknownOrderType    = errors.New("unknown order type")
	errUnknownWalletType   = errors.New("unknown wallet type")
	errOrderSideUnresolved = errors.New("order side cannot be told from a zero amount")
	errPairNotOfAsset      = errors.New("pair does not trade as the asset")
)

const (
	// derivativeCurrencySuffix ends the currencies of derivative pairs, such as tBTCF0:USTF0
	derivativeCurrencySuffix = "F0"
	// orderMetaPostOnlyKey and orderMetaLeverageKey key an order's post only flag and leverage in its meta data
	orderMetaPostOnlyKey = "$F7"
	orderMetaLeverageKey = "$F33"
)

// symbolFor returns an asset's pair's Bitfinex symbol: a funding currency prefixed f, and a trading pair prefixed t,
// with a colon between currencies longer than three letters, such as tAVAX:USD. A derivative pair trades as the futures
// asset its collateral margins, and any other trading pair as spot or margin, so a pair given with another asset is
// refused rather than sent to a market of another asset
func symbolFor(a asset.Item, p currency.Pair) (string, error) {
	if p.IsEmpty() {
		return "", currency.ErrCurrencyPairEmpty
	}
	switch a {
	case asset.MarginFunding:
		return "f" + p.Base.Upper().String(), nil
	case asset.Spot, asset.Margin:
		if isDerivativePair(p) {
			return "", fmt.Errorf("%w: %s is a derivative pair, not %s", errPairNotOfAsset, p, a)
		}
		return "t" + pairSymbol(p), nil
	case asset.USDTMarginedFutures, asset.CoinMarginedFutures:
		if !isDerivativePair(p) || derivativeAsset(p.Quote) != a {
			return "", fmt.Errorf("%w: %s is not a %s pair", errPairNotOfAsset, p, a)
		}
		return "t" + pairSymbol(p), nil
	}
	return "", fmt.Errorf("%w: %q", asset.ErrNotSupported, a)
}

// orderSymbol returns the symbol of a pair an asset's orders trade; funding takes offers rather than orders
func orderSymbol(a asset.Item, p currency.Pair) (string, error) {
	if a == asset.MarginFunding {
		return "", fmt.Errorf("%w: %s orders", asset.ErrNotSupported, a)
	}
	return symbolFor(a, p)
}

// pairSymbol returns a pair as Bitfinex writes it, without the t prefix
func pairSymbol(p currency.Pair) string {
	base, quote := p.Base.Upper().String(), p.Quote.Upper().String()
	if len(base) > 3 || len(quote) > 3 {
		return base + ":" + quote
	}
	return base + quote
}

// pairFromSymbol returns the pair a trading pair or funding currency symbol names, the currency alone for a funding
// symbol; a symbol without its t or f prefix is read as a trading pair
func pairFromSymbol(symbol string) (currency.Pair, error) {
	switch {
	case isFundingSymbol(symbol):
		return currency.Pair{Base: currency.NewCode(symbol[1:])}, nil
	case isTradingSymbol(symbol):
		symbol = symbol[1:]
	}
	if base, quote, ok := strings.Cut(symbol, ":"); ok {
		if base == "" || quote == "" {
			return currency.EMPTYPAIR, fmt.Errorf("%w: %q", errSymbolUnparsable, symbol)
		}
		return currency.NewPair(currency.NewCode(base), currency.NewCode(quote)), nil
	}
	// Bitfinex writes a colon between any currencies longer than three letters, so a pair without one splits evenly
	if len(symbol) != 6 {
		return currency.EMPTYPAIR, fmt.Errorf("%w: %q", errSymbolUnparsable, symbol)
	}
	return currency.NewPair(currency.NewCode(symbol[:3]), currency.NewCode(symbol[3:])), nil
}

// isDerivativePair reports whether a pair is a derivative's, whose currencies Bitfinex suffixes F0
func isDerivativePair(p currency.Pair) bool {
	return isDerivativeCurrency(p.Base) && isDerivativeCurrency(p.Quote)
}

// isDerivativeCurrency reports whether a currency is a derivative's, such as BTCF0, or the collateral USTF0
func isDerivativeCurrency(c currency.Code) bool {
	return strings.HasSuffix(c.Upper().String(), derivativeCurrencySuffix)
}

// isDerivativeAsset reports whether an asset is one Bitfinex's derivative pairs, all perpetual contracts, trade as
func isDerivativeAsset(a asset.Item) bool {
	return a == asset.USDTMarginedFutures || a == asset.CoinMarginedFutures
}

// derivativeAsset returns the asset of the derivatives a collateral currency, a derivative pair's quote, margins: USDT
// margined futures for USTF0, Tether's USDT, and TESTUSDTF0, its paper trading twin, and coin margined futures for any
// other, such as BTCF0
func derivativeAsset(collateral currency.Code) asset.Item {
	switch strings.TrimSuffix(strings.TrimPrefix(collateral.Upper().String(), testListingMarker), derivativeCurrencySuffix) {
	case currency.UST.String(), currency.USDT.String():
		return asset.USDTMarginedFutures
	}
	return asset.CoinMarginedFutures
}

// underlyingCurrency returns the currency a derivative currency follows, such as BTC for BTCF0
func underlyingCurrency(c currency.Code) currency.Code {
	return currency.NewCode(strings.TrimSuffix(c.Upper().String(), derivativeCurrencySuffix))
}

// orderAsset returns the asset an order trades: a derivative pair's futures, by its collateral, an EXCHANGE order
// type's spot on the exchange wallet, and margin for the other order types, which trade on the margin wallet
func orderAsset(orderType string, p currency.Pair) asset.Item {
	switch {
	case isDerivativePair(p):
		return derivativeAsset(p.Quote)
	case strings.HasPrefix(orderType, ExchangeOrderTypePrefix):
		return asset.Spot
	}
	return asset.Margin
}

// orderTypeFromBitfinex returns the order type and any time in force a Bitfinex order type implies
func orderTypeFromBitfinex(orderType string) (order.Type, order.TimeInForce, error) {
	switch strings.TrimPrefix(orderType, ExchangeOrderTypePrefix) {
	case OrderTypeLimit:
		return order.Limit, order.UnknownTIF, nil
	case OrderTypeMarket:
		return order.Market, order.UnknownTIF, nil
	case OrderTypeStop:
		return order.Stop, order.UnknownTIF, nil
	case OrderTypeStopLimit:
		return order.StopLimit, order.UnknownTIF, nil
	case OrderTypeTrailingStop:
		return order.TrailingStop, order.UnknownTIF, nil
	case OrderTypeFillOrKill:
		return order.Limit, order.FillOrKill, nil
	case OrderTypeImmediateOrCancel:
		return order.Limit, order.ImmediateOrCancel, nil
	}
	return order.UnknownType, order.UnknownTIF, fmt.Errorf("%w: %q", errUnknownOrderType, orderType)
}

// orderStatusFromBitfinex returns the status an order's status text describes. Bitfinex appends how an order got
// there: "EXECUTED @ 107.6(-0.2)", "CANCELED was: PARTIALLY FILLED @ 107.6(-0.1)"
func orderStatusFromBitfinex(status string) order.Status {
	primary, history, _ := strings.Cut(status, " was:")
	primary, _, _ = strings.Cut(primary, " @")
	primary = strings.TrimSpace(primary)
	partiallyFilled := strings.Contains(history, "PARTIALLY FILLED")
	switch {
	case primary == "ACTIVE":
		return order.Active
	case primary == "EXECUTED":
		return order.Filled
	case primary == "PARTIALLY FILLED":
		return order.PartiallyFilled
	case primary == "FORCED EXECUTED":
		return order.Liquidated
	case strings.HasPrefix(primary, "INSUFFICIENT"):
		return order.InsufficientBalance
	case strings.HasSuffix(primary, "CANCELED"), strings.HasPrefix(primary, "RSN_"):
		if partiallyFilled {
			return order.PartiallyFilledCancelled
		}
		return order.Cancelled
	}
	return order.UnknownStatus
}

// closedOrderStatus returns the status of an order that has left the book. Bitfinex's history, and its websocket's
// order closed event, can report an order whose remainder was cancelled as PARTIALLY FILLED
func closedOrderStatus(s order.Status) order.Status {
	if s == order.PartiallyFilled {
		return order.PartiallyFilledCancelled
	}
	return s
}

// orderStatusFill matches each fill an order's status lists, @ PRICE(AMOUNT), such as the two of "EXECUTED @
// 916.2(0.78): was PARTIALLY FILLED @ 915.9(0.22)", but not a note such as INSUFFICIENT BALANCE (G1)'s
var orderStatusFill = regexp.MustCompile(`@ -?(?:\d+\.?\d*|\.\d+)(?:[eE][-+]?\d{1,3})?\((-?(?:\d+\.?\d*|\.\d+)(?:[eE][-+]?\d{1,3})?)\)`)

// orderStatusExecutedAmount returns the amount an order has executed, adding up the fills its status lists as the
// decimals Bitfinex writes them in. Bitfinex keeps an order's original amount from before any update, so what an order
// has executed cannot be told from what remains of it
func orderStatusExecutedAmount(status string) decimal.Decimal {
	executed := decimal.Zero
	for _, fill := range orderStatusFill.FindAllStringSubmatch(status, -1) {
		if amount, err := decimal.NewFromString(fill[1]); err == nil {
			executed = executed.Add(amount.Abs())
		}
	}
	return executed
}

// orderSideFromAmount returns the side an order's signed amount gives: positive buys, negative sells
func orderSideFromAmount(amount float64) (order.Side, error) {
	switch {
	case amount > 0:
		return order.Buy, nil
	case amount < 0:
		return order.Sell, nil
	}
	return order.UnknownSide, errOrderSideUnresolved
}

// orderDetail returns an order as GCT records it
func (e *Exchange) orderDetail(o *Order) (*order.Detail, error) {
	pair, err := pairFromSymbol(o.Symbol)
	if err != nil {
		return nil, err
	}
	orderType, tif, err := orderTypeFromBitfinex(o.Type)
	if err != nil {
		return nil, err
	}
	side, err := orderSideFromAmount(o.OriginalAmount)
	if err != nil {
		return nil, err
	}
	// Bitfinex reports some orders' post only flag, and their leverage, in their meta data instead
	postOnly, _ := jsonparser.GetInt(o.Meta, orderMetaPostOnlyKey)
	if o.Flags&OrderFlagPostOnly != 0 || postOnly == 1 {
		tif |= order.PostOnly
	}
	if !o.TimeInForce.Time().IsZero() {
		tif |= order.GoodTillTime
	}
	remaining := math.Abs(o.Amount)
	executed := orderStatusExecutedAmount(o.Status)
	d := &order.Detail{
		Exchange:             e.Name,
		OrderID:              strconv.FormatUint(o.ID, 10),
		Pair:                 pair,
		AssetType:            orderAsset(o.Type, pair),
		Side:                 side,
		Type:                 orderType,
		TimeInForce:          tif,
		Status:               orderStatusFromBitfinex(o.Status),
		Price:                o.Price,
		Amount:               executed.Add(decimal.MustFromFloat(remaining)).InexactFloat64(),
		RemainingAmount:      remaining,
		ExecutedAmount:       executed.InexactFloat64(),
		AverageExecutedPrice: o.AveragePrice,
		HiddenOrder:          bool(o.Hidden) || o.Flags&OrderFlagHidden != 0,
		ReduceOnly:           o.Flags&OrderFlagReduceOnly != 0,
		Date:                 o.CreatedAt.Time(),
		LastUpdated:          o.UpdatedAt.Time(),
	}
	if leverage, err := jsonparser.GetFloat(o.Meta, orderMetaLeverageKey); err == nil {
		d.Leverage = leverage
	}
	if o.ClientOrderID != 0 {
		d.ClientOrderID = strconv.FormatUint(o.ClientOrderID, 10)
	}
	switch orderType {
	case order.Stop:
		// A stop's price is the one it triggers at, and it then trades at the market
		d.TriggerPrice, d.Price = o.Price, 0
	case order.StopLimit:
		d.TriggerPrice, d.Price = o.Price, o.AuxiliaryLimitPrice
	}
	return d, nil
}

// walletAsset returns the asset a wallet's balance of a currency funds, accepting the labels the websocket filters use
// too. The margin wallet is also the derivatives wallet, which holds derivatives' collateral in derivative currencies,
// such as USTF0
func walletAsset(walletType string, c currency.Code) (asset.Item, error) {
	switch walletType {
	case WalletTypeExchange:
		return asset.Spot, nil
	case WalletTypeMargin, "trading":
		if isDerivativeCurrency(c) {
			return derivativeAsset(c), nil
		}
		return asset.Margin, nil
	case WalletTypeFunding, "deposit":
		return asset.MarginFunding, nil
	}
	return asset.Empty, fmt.Errorf("%w: %q", errUnknownWalletType, walletType)
}
