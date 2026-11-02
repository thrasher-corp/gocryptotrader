package forexprovider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
)

func TestGetSupportedForexProvidersIncludesFXMacroData(t *testing.T) {
	providers := GetSupportedForexProviders()
	assert.Contains(t, providers, "FXMacroData", "supported provider list should include FXMacroData")
}

func TestStartFXServiceFXMacroData(t *testing.T) {
	handler, err := StartFXService([]base.Settings{
		{
			Name:            "FXMacroData",
			Enabled:         true,
			APIKey:          "test-key",
			PrimaryProvider: true,
		},
	})
	require.NoError(t, err, "StartFXService must start FXMacroData")
	require.NotNil(t, handler.Primary.Provider, "primary provider must be set")
	assert.Equal(t, "FXMacroData", handler.Primary.Provider.GetName(), "primary provider should be FXMacroData")
}

func TestStartFXServiceAllRatesToday(t *testing.T) {
	handler, err := StartFXService([]base.Settings{
		{
			Name:            "AllRatesToday",
			Enabled:         true,
			APIKey:          "test-key",
			PrimaryProvider: true,
		},
	})
	require.NoError(t, err, "StartFXService must start AllRatesToday")
	require.NotNil(t, handler.Primary.Provider, "primary provider must be set")
	assert.Equal(t, "AllRatesToday", handler.Primary.Provider.GetName(), "primary provider should be AllRatesToday")
	assert.Contains(t, GetSupportedForexProviders(), "AllRatesToday", "supported provider list should include AllRatesToday")
}
