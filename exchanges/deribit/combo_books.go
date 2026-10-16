package deribit

import (
	"context"
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

var (
	errComboTradesRequired = errors.New("at least one combo trade is required")
	errComboIDRequired     = errors.New("combo ID is required")
	errInvalidComboState   = errors.New("invalid combo state")
)

// CreateCombo returns the combo whose legs match the trades, creating its book when none exists
func (e *Exchange) CreateCombo(ctx context.Context, trades []ComboTrade) (*Combo, error) {
	return e.createCombo(ctx, trades, false)
}

func (e *Exchange) createCombo(ctx context.Context, trades []ComboTrade, overWebsocket bool) (*Combo, error) {
	if err := validateComboTrades(trades); err != nil {
		return nil, err
	}
	var resp *Combo
	if err := e.sendRequest(ctx, overWebsocket, matchingEPL, "private/create_combo", &createComboParams{Trades: trades}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetLegPrices returns each leg's price for a price of the whole combo structure, derived from the legs' mark prices;
// Deribit calibrates it for conventional option structures and future spreads within one currency pair only
func (e *Exchange) GetLegPrices(ctx context.Context, arg *LegPricesRequest) (*LegPricesResponse, error) {
	return e.getLegPrices(ctx, arg, false)
}

func (e *Exchange) getLegPrices(ctx context.Context, arg *LegPricesRequest, overWebsocket bool) (*LegPricesResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if err := validateComboTrades(arg.Legs); err != nil {
		return nil, err
	}
	var resp *LegPricesResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_leg_prices", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// validateComboTrades checks a combo structure has trades, each with an instrument, a positive amount and a direction
func validateComboTrades(trades []ComboTrade) error {
	if len(trades) == 0 {
		return errComboTradesRequired
	}
	for i := range trades {
		if trades[i].InstrumentName == "" {
			return fmt.Errorf("%w: combo trade %d", errInstrumentNameRequired, i)
		}
		if trades[i].Amount <= 0 {
			return fmt.Errorf("%w: combo trade %d", errInvalidAmount, i)
		}
		if trades[i].Direction != DirectionBuy && trades[i].Direction != DirectionSell {
			return fmt.Errorf("%w: combo trade %d: %q", errInvalidDirection, i, trades[i].Direction)
		}
	}
	return nil
}

// GetComboDetails returns a live combo's legs and book state; an expired or archivized combo is only available as an
// instrument from GetInstrument
func (e *Exchange) GetComboDetails(ctx context.Context, comboID string) (*Combo, error) {
	return e.getComboDetails(ctx, comboID, false)
}

func (e *Exchange) getComboDetails(ctx context.Context, comboID string, overWebsocket bool) (*Combo, error) {
	if comboID == "" {
		return nil, errComboIDRequired
	}
	var resp *Combo
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_combo_details", &comboIDParams{ComboID: comboID}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetComboIDs returns the IDs of a settlement currency's active and inactive combos, or of those in one state; any is
// not accepted as a currency
func (e *Exchange) GetComboIDs(ctx context.Context, arg *ComboIDsRequest) ([]string, error) {
	return e.getComboIDs(ctx, arg, false)
}

func (e *Exchange) getComboIDs(ctx context.Context, arg *ComboIDsRequest, overWebsocket bool) ([]string, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	switch arg.State {
	case "", ComboStateActive, ComboStateInactive, ComboStateAny:
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidComboState, arg.State)
	}
	var resp []string
	return resp, e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_combo_ids", arg, &resp)
}

// GetCombos returns a settlement currency's active combos, or every currency's for currency.ANY
func (e *Exchange) GetCombos(ctx context.Context, ccy currency.Code) ([]Combo, error) {
	return e.getCombos(ctx, ccy, false)
}

func (e *Exchange) getCombos(ctx context.Context, ccy currency.Code, overWebsocket bool) ([]Combo, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp []Combo
	return resp, e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_combos", &currencyParams{Currency: marketDataCurrency(ccy)}, &resp)
}
