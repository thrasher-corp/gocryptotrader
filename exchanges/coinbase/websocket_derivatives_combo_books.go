package coinbase

import (
	"context"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

// CreateCombo returns the combo whose legs match the trades as GlobalDerivativesAPI.CreateCombo does, over the
// websocket, which must be authenticated
func (w WebsocketAPI) CreateCombo(ctx context.Context, trades []DerivativesComboTrade) (*DerivativesComboResponse, error) {
	return w.e.GlobalDerivatives().createCombo(ctx, true, trades)
}

// GetLegPrices returns each leg's price for a price of the whole combo structure as GlobalDerivativesAPI.GetLegPrices
// does, over the websocket, which must be authenticated
func (w WebsocketAPI) GetLegPrices(ctx context.Context, req *DerivativesLegPricesRequest) (*DerivativesLegPricesResponse, error) {
	return w.e.GlobalDerivatives().getLegPrices(ctx, true, req)
}

// GetComboDetails returns a live combo's legs and book state as GlobalDerivativesAPI.GetComboDetails does, over the
// websocket
func (w WebsocketAPI) GetComboDetails(ctx context.Context, comboID string) (*DerivativesComboResponse, error) {
	return w.e.GlobalDerivatives().getComboDetails(ctx, true, comboID)
}

// GetComboIDs returns the IDs of a settlement currency's combos as GlobalDerivativesAPI.GetComboIDs does, over the
// websocket
func (w WebsocketAPI) GetComboIDs(ctx context.Context, req *DerivativesComboIDsRequest) ([]string, error) {
	return w.e.GlobalDerivatives().getComboIDs(ctx, true, req)
}

// GetCombos returns a settlement currency's active combos as GlobalDerivativesAPI.GetCombos does, over the websocket
func (w WebsocketAPI) GetCombos(ctx context.Context, ccy currency.Code) ([]DerivativesComboResponse, error) {
	return w.e.GlobalDerivatives().getCombos(ctx, true, ccy)
}
