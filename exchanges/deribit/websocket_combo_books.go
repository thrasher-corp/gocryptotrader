package deribit

import (
	"context"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

// CreateCombo returns the combo whose legs match the trades as Exchange.CreateCombo does, over the websocket
func (w WebsocketAPI) CreateCombo(ctx context.Context, trades []ComboTrade) (*Combo, error) {
	return w.e.createCombo(ctx, trades, true)
}

// GetLegPrices returns each leg's price for a price of the whole combo structure as Exchange.GetLegPrices does, over
// the websocket
func (w WebsocketAPI) GetLegPrices(ctx context.Context, arg *LegPricesRequest) (*LegPricesResponse, error) {
	return w.e.getLegPrices(ctx, arg, true)
}

// GetComboDetails returns a live combo's legs and book state as Exchange.GetComboDetails does, over the websocket
func (w WebsocketAPI) GetComboDetails(ctx context.Context, comboID string) (*Combo, error) {
	return w.e.getComboDetails(ctx, comboID, true)
}

// GetComboIDs returns the IDs of a settlement currency's combos as Exchange.GetComboIDs does, over the websocket
func (w WebsocketAPI) GetComboIDs(ctx context.Context, arg *ComboIDsRequest) ([]string, error) {
	return w.e.getComboIDs(ctx, arg, true)
}

// GetCombos returns a settlement currency's active combos as Exchange.GetCombos does, over the websocket
func (w WebsocketAPI) GetCombos(ctx context.Context, ccy currency.Code) ([]Combo, error) {
	return w.e.getCombos(ctx, ccy, true)
}
