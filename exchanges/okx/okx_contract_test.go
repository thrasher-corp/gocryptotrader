package okx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestFuturesContractFromInstrument(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, instrumentType, underlying, state, settlementType, alias string
		item                                                           asset.Item
		currency                                                       currency.Code
		wantSettlement                                                 futures.ContractSettlementType
		wantType                                                       futures.ContractType
		wantErr                                                        error
		nilInput                                                       bool
	}{
		{name: "nil", nilInput: true, wantErr: common.ErrNilPointer},
		{name: "spot", item: asset.Spot, wantErr: futures.ErrNotFuturesAsset},
		{name: "spread", item: asset.Spread, wantErr: asset.ErrNotSupported},
		{name: "unsupported futures", item: asset.USDTMarginedFutures, wantErr: asset.ErrNotSupported},
		{name: "wrong instrument type", item: asset.PerpetualSwap, instrumentType: "SPOT", wantErr: asset.ErrNotSupported},
		{name: "active missing underlying", item: asset.PerpetualSwap, state: "live", wantErr: currency.ErrCreatingPair},
		{name: "invalid settlement", item: asset.PerpetualSwap, underlying: "BTC-USD", settlementType: "unknown", wantErr: futures.ErrInvalidContractSettlementType},
		{name: "linear", item: asset.PerpetualSwap, instrumentType: "SWAP", underlying: "BTC-USDT", state: "live", settlementType: "linear", currency: currency.USDT, wantSettlement: futures.Linear, wantType: futures.Perpetual},
		{name: "inverse empty base currency", item: asset.PerpetualSwap, instrumentType: "swap", underlying: "BTC-USD", state: "live", settlementType: "inverse", currency: currency.BTC, wantSettlement: futures.Inverse, wantType: futures.Perpetual},
		{name: "inverse fallback", item: asset.PerpetualSwap, underlying: "BTC-USD", state: "live", currency: currency.BTC, wantSettlement: futures.Inverse, wantType: futures.Perpetual},
		{name: "linear fallback", item: asset.PerpetualSwap, underlying: "BTC-USDT", state: "live", currency: currency.USDT, wantSettlement: futures.Linear, wantType: futures.Perpetual},
		{name: "explicit type wins", item: asset.PerpetualSwap, underlying: "BTC-USD", state: "live", settlementType: "linear", currency: currency.BTC, wantSettlement: futures.Linear, wantType: futures.Perpetual},
		{name: "suspended inverse", item: asset.PerpetualSwap, state: "suspend", settlementType: "inverse", currency: currency.BTC, wantSettlement: futures.Inverse, wantType: futures.Perpetual},
		{name: "inactive preserves underlying", item: asset.PerpetualSwap, state: "preopen", underlying: "BTC-USDT", settlementType: "linear", currency: currency.USDT, wantSettlement: futures.Linear, wantType: futures.Perpetual},
		{name: "inactive empty metadata", item: asset.PerpetualSwap, state: "expired", wantSettlement: futures.Linear, wantType: futures.Perpetual},
		{name: "this week", item: asset.Futures, instrumentType: "FUTURES", state: "live", underlying: "BTC-USD", alias: "this_week", settlementType: "inverse", currency: currency.BTC, wantSettlement: futures.Inverse, wantType: futures.Weekly},
		{name: "next week", item: asset.Futures, underlying: "BTC-USD", alias: "next_week", wantSettlement: futures.Linear, wantType: futures.Weekly},
		{name: "quarter", item: asset.Futures, underlying: "BTC-USD", alias: "quarter", wantSettlement: futures.Linear, wantType: futures.Quarterly},
		{name: "next quarter", item: asset.Futures, underlying: "BTC-USD", alias: "next_quarter", wantSettlement: futures.Linear, wantType: futures.Quarterly},
		{name: "unknown alias", item: asset.Futures, underlying: "BTC-USD", alias: "other", wantSettlement: futures.Linear},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			ex.Name = "Okx"
			started, expires := time.Unix(1700000000, 0).UTC(), time.Unix(1700100000, 0).UTC()
			instrument := &Instrument{InstrumentType: tc.instrumentType, InstrumentID: currency.NewPairWithDelimiter("BTC", "USD-SWAP", "-"), Underlying: tc.underlying, State: tc.state, ContractType: tc.settlementType, SettlementCurrency: tc.currency, Alias: tc.alias, ContractValue: 100, MaxLeverage: 20, ListTime: types.Time(started), ExpTime: types.Time(expires)}
			if tc.nilInput {
				instrument = nil
			}
			got, err := ex.FuturesContractFromInstrument(instrument, tc.item)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr, "invalid metadata must return the expected error")
				return
			}
			require.NoError(t, err, "valid metadata must convert without network requests")
			assert.Equal(t, tc.wantSettlement, got.SettlementType, "settlement type should follow native metadata")
			assert.Equal(t, tc.wantType, got.Type, "contract type should follow the asset and alias")
			assert.Equal(t, tc.currency, got.MarginCurrency, "margin currency should survive in every state")
			assert.Equal(t, tc.currency, got.SettlementCurrency, "settlement currency should survive in every state")
			assert.Equal(t, tc.state == "live", got.IsActive, "activity should follow instrument state")
			assert.Equal(t, tc.state, got.Status, "status should remain available")
			assert.Equal(t, "Okx", got.Exchange, "exchange should be retained")
			assert.Equal(t, instrument.InstrumentID, got.Name, "instrument identity should be retained")
			assert.Equal(t, tc.item, got.Asset, "asset should be retained")
			assert.Equal(t, started, got.StartDate.UTC(), "listing time should be retained")
			assert.Equal(t, expires, got.EndDate.UTC(), "expiry should be retained")
			assert.Equal(t, float64(100), got.Multiplier, "contract value should be retained")
			assert.Equal(t, float64(20), got.MaxLeverage, "leverage should be retained")
			if tc.underlying != "" {
				assert.Equal(t, tc.underlying, got.Underlying.String(), "underlying should be retained in every state")
			}
		})
	}
}
