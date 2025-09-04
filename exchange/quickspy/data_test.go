package quickspy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestHandleWSAccountChange(t *testing.T) {
	t.Parallel()
	q := mustQuickSpy(t, AccountHoldingsFocusType)
	require.ErrorIs(t, q.handleWSAccountChange(nil), common.ErrNilPointer)

	d := &account.Change{
		AssetType: q.key.ExchangeAssetPair.Asset,
		Balance: &account.Balance{
			Currency:               currency.BTC,
			Total:                  1337,
			Hold:                   1337,
			Free:                   1337,
			AvailableWithoutBorrow: 1337,
			Borrowed:               1337,
			UpdatedAt:              time.Now(),
		},
	}
	require.NoError(t, q.handleWSAccountChange(d))
	assert.Equal(t, d.Balance, &q.data.AccountBalance[0])

	d2 := &account.Change{
		AssetType: asset.Binary,
		Balance: &account.Balance{
			Currency:               currency.BTC,
			Total:                  1,
			Hold:                   1,
			Free:                   1,
			AvailableWithoutBorrow: 1,
			Borrowed:               1,
			UpdatedAt:              time.Now(),
		},
	}
	require.NoError(t, q.handleWSAccountChange(d2))
	assert.NotEqual(t, d2.Balance, &q.data.AccountBalance[0])
}

func TestHandleWSAccountChanges(t *testing.T) {
	t.Parallel()
	q := mustQuickSpy(t, AccountHoldingsFocusType)
	require.NoError(t, q.handleWSAccountChanges(nil))

	d := account.Change{
		AssetType: q.key.ExchangeAssetPair.Asset,
		Balance: &account.Balance{
			Currency:               currency.BTC,
			Total:                  1337,
			Hold:                   1337,
			Free:                   1337,
			AvailableWithoutBorrow: 1337,
			Borrowed:               1337,
			UpdatedAt:              time.Now(),
		},
	}
	require.NoError(t, q.handleWSAccountChanges([]account.Change{d}))
	assert.Equal(t, d.Balance, &q.data.AccountBalance[0])

	d2 := account.Change{
		AssetType: asset.Binary,
		Balance: &account.Balance{
			Currency:               currency.BTC,
			Total:                  1,
			Hold:                   1,
			Free:                   1,
			AvailableWithoutBorrow: 1,
			Borrowed:               1,
			UpdatedAt:              time.Now(),
		},
	}
	require.NoError(t, q.handleWSAccountChanges([]account.Change{d2}))
	assert.NotEqual(t, d2.Balance, &q.data.AccountBalance[0])
}

func TestAccountHoldingsFocusType(t *testing.T) {
	t.Parallel()
	if apiKey == "abc" || apiSecret == "123" {
		t.Skip("API credentials not set; skipping test that requires them")
	}
	qs := mustQuickSpy(t, AccountHoldingsFocusType)
	f, err := qs.GetFocusByKey(AccountHoldingsFocusType)
	require.NoError(t, err)
	require.NotNil(t, f)

	require.NoError(t, qs.handleFocusType(f.focusType, f, time.NewTimer(f.restPollTime)))
	require.NotEmpty(t, qs.data.AccountBalance)
}
