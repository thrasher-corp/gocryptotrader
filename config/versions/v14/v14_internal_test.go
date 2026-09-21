package v14

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateSubscriptions(t *testing.T) {
	t.Parallel()
	for _, upgrade := range []bool{true, false} {
		legacy := fmt.Sprintf(`{"enabled":%t,"channel":"orderbook","asset":"spot","interval":"100ms"}`, upgrade)
		v2 := fmt.Sprintf(`{"enabled":%t,"channel":"spot.obu","asset":"spot","levels":50}`, !upgrade)
		migratedLegacy := fmt.Sprintf(`{"enabled":%t,"channel":"orderbook","asset":"spot","interval":"100ms"}`, !upgrade)
		migratedV2 := fmt.Sprintf(`{"enabled":%t,"channel":"spot.obu","asset":"spot","levels":50}`, upgrade)
		otherAssets := `{"enabled":true,"channel":"orderbook","asset":"usdtmarginedfutures","interval":"100ms"},{"enabled":false,"channel":"spot.obu","asset":"margin","levels":50},{"enabled":true,"channel":"spot.order_book_update","asset":"coinmarginedfutures"}`
		missingV2Expected := ""
		if upgrade {
			missingV2Expected = `{"features":{"subscriptions":[` + migratedLegacy + `,` + migratedV2 + `]}}`
		}
		for _, test := range []struct {
			name        string
			input       string
			expected    string
			errContains string
		}{
			{
				name:     "defaults",
				input:    `{"features":{"subscriptions":[` + legacy + `,` + v2 + `]}}`,
				expected: `{"features":{"subscriptions":[` + migratedLegacy + `,` + migratedV2 + `]}}`,
			},
			{
				name:  "duplicate legacy entries",
				input: `{"features":{"subscriptions":[` + legacy + `,` + legacy + `,` + v2 + `]}}`,
			},
			{
				name:  "non-spot entries only",
				input: `{"features":{"subscriptions":[` + otherAssets + `]}}`,
			},
			{
				name:     "non-spot entries alongside spot defaults",
				input:    `{"features":{"subscriptions":[` + otherAssets + `,` + legacy + `,` + v2 + `]}}`,
				expected: `{"features":{"subscriptions":[` + otherAssets + `,` + migratedLegacy + `,` + migratedV2 + `]}}`,
			},
			{
				name:     "missing V2 entry",
				input:    `{"features":{"subscriptions":[` + legacy + `]}}`,
				expected: missingV2Expected,
			},
			{
				name:  "missing subscriptions",
				input: `{"features":{}}`,
			},
			{
				name:        "malformed subscription array",
				input:       `{"features":{"subscriptions":[}`,
				errContains: "error getting GateIO subscriptions",
			},
			{
				name:        "subscriptions object",
				input:       `{"features":{"subscriptions":{}}}`,
				errContains: "error decoding GateIO subscriptions",
			},
			{
				name:        "invalid enabled type",
				input:       `{"features":{"subscriptions":[{"enabled":"true","channel":"orderbook","asset":"spot"}]}}`,
				errContains: "error decoding GateIO subscriptions",
			},
		} {
			t.Run(fmt.Sprintf("%s/upgrade=%t", test.name, upgrade), func(t *testing.T) {
				t.Parallel()
				got, err := migrateSubscriptions([]byte(test.input), upgrade)
				if test.errContains != "" {
					require.ErrorContains(t, err, test.errContains, "migration must report the failing operation")
					assert.Equal(t, test.input, string(got), "failed migration should preserve the input")
					return
				}
				require.NoError(t, err, "migration must accept valid subscriptions")
				if test.expected == "" {
					assert.Equal(t, test.input, string(got), "ineligible migration should preserve the input byte-for-byte")
				} else {
					assert.JSONEq(t, test.expected, string(got), "migration should change only eligible spot defaults")
				}
			})
		}
	}
}
