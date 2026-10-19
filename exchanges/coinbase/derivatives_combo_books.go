package coinbase

import (
	"context"
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

var (
	errDerivativesComboTradesRequired   = errors.New("at least one combo trade required")
	errDerivativesComboIDRequired       = errors.New("combo ID required")
	errDerivativesInvalidComboState     = errors.New("invalid combo state")
	errDerivativesInvalidComboDirection = errors.New("invalid combo trade direction")
)

// CreateCombo returns the combo whose legs match the trades, creating its book when none exists
func (g GlobalDerivativesAPI) CreateCombo(ctx context.Context, trades []DerivativesComboTrade) (*DerivativesCombo, error) {
	return g.createCombo(ctx, false, trades)
}

func (g GlobalDerivativesAPI) createCombo(ctx context.Context, overWebsocket bool, trades []DerivativesComboTrade) (*DerivativesCombo, error) {
	if err := validateDerivativesComboTrades(trades); err != nil {
		return nil, err
	}
	var resp *DerivativesCombo
	if err := g.sendRequest(ctx, overWebsocket, derivativesMatchingEPL, "private/create_combo", &derivativesCreateComboParams{Trades: trades}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetLegPrices returns each leg's price for a price of the whole combo structure, derived from the legs' mark prices;
// the gateway calibrates it for conventional option structures and future spreads within one currency pair only
func (g GlobalDerivativesAPI) GetLegPrices(ctx context.Context, req *DerivativesLegPricesRequest) (*DerivativesLegPricesResponse, error) {
	return g.getLegPrices(ctx, false, req)
}

func (g GlobalDerivativesAPI) getLegPrices(ctx context.Context, overWebsocket bool, req *DerivativesLegPricesRequest) (*DerivativesLegPricesResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := validateDerivativesComboTrades(req.Legs); err != nil {
		return nil, err
	}
	var resp *DerivativesLegPricesResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_leg_prices", &derivativesLegPricesParams{Legs: req.Legs, Price: req.Price}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// validateDerivativesComboTrades checks a combo structure has trades, each with an instrument, a positive amount and a
// direction
func validateDerivativesComboTrades(trades []DerivativesComboTrade) error {
	if len(trades) == 0 {
		return errDerivativesComboTradesRequired
	}
	for i := range trades {
		if trades[i].InstrumentName == "" {
			return fmt.Errorf("%w: combo trade %d", errInstrumentNameRequired, i)
		}
		if trades[i].Amount <= 0 {
			return fmt.Errorf("%w: combo trade %d: %v", errInvalidAmount, i, trades[i].Amount)
		}
		if trades[i].Direction != DerivativesDirectionBuy && trades[i].Direction != DerivativesDirectionSell {
			return fmt.Errorf("%w: combo trade %d: %q", errDerivativesInvalidComboDirection, i, trades[i].Direction)
		}
	}
	return nil
}

// GetComboDetails returns a combo's legs and book state, an expired combo's included
func (g GlobalDerivativesAPI) GetComboDetails(ctx context.Context, comboID string) (*DerivativesCombo, error) {
	return g.getComboDetails(ctx, false, comboID)
}

func (g GlobalDerivativesAPI) getComboDetails(ctx context.Context, overWebsocket bool, comboID string) (*DerivativesCombo, error) {
	if comboID == "" {
		return nil, errDerivativesComboIDRequired
	}
	var resp *DerivativesCombo
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_combo_details", &derivativesComboIDParams{ComboID: comboID}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetComboIDs returns the IDs of a settlement currency's combos in a state, or in either state
func (g GlobalDerivativesAPI) GetComboIDs(ctx context.Context, req *DerivativesComboIDsRequest) ([]string, error) {
	return g.getComboIDs(ctx, false, req)
}

func (g GlobalDerivativesAPI) getComboIDs(ctx context.Context, overWebsocket bool, req *DerivativesComboIDsRequest) ([]string, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	switch req.State {
	case "", DerivativesComboStateActive, DerivativesComboStateInactive, DerivativesComboStateAny:
	default:
		return nil, fmt.Errorf("%w: %q", errDerivativesInvalidComboState, req.State)
	}
	var resp []string
	return resp, g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_combo_ids", &derivativesComboIDsParams{Currency: derivativesMarketDataCurrency(req.Currency), State: req.State}, &resp)
}

// GetCombos returns a settlement currency's active combos, or every currency's for any
func (g GlobalDerivativesAPI) GetCombos(ctx context.Context, ccy currency.Code) ([]DerivativesCombo, error) {
	return g.getCombos(ctx, false, ccy)
}

func (g GlobalDerivativesAPI) getCombos(ctx context.Context, overWebsocket bool, ccy currency.Code) ([]DerivativesCombo, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp []DerivativesCombo
	return resp, g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_combos", &derivativesCurrencyParams{Currency: derivativesMarketDataCurrency(ccy)}, &resp)
}
