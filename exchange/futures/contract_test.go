package futures

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringToContractSettlementType(t *testing.T) {
	t.Parallel()
	contractSettlementTypesMap := map[string]struct {
		CT    ContractSettlementType
		Error error
	}{
		"lInear":          {Linear, nil},
		"LINEAR":          {Linear, nil},
		"Inverse":         {Inverse, nil},
		"unset":           {UnsetSettlementType, nil},
		"hybRiD":          {Hybrid, nil},
		"LinearOrInverse": {LinearOrInverse, nil},
		"":                {UnsetSettlementType, nil},
		"Quanto":          {Quanto, nil},
		"QUANTO":          {Quanto, nil},
		"Unknown":         {UnsetSettlementType, ErrInvalidContractSettlementType},
	}
	for x, v := range contractSettlementTypesMap {
		val, err := StringToContractSettlementType(x)
		assert.Equal(t, v.CT, val)
		assert.ErrorIs(t, err, v.Error)
	}
}

func TestContractSettlementTypeString(t *testing.T) {
	t.Parallel()
	contractSettlementTypeToStringMap := map[ContractSettlementType]string{
		UnsetSettlementType:         "unset",
		Linear:                      "linear",
		Inverse:                     "inverse",
		Quanto:                      "quanto",
		LinearOrInverse:             "linearOrInverse",
		Hybrid:                      "hybrid",
		ContractSettlementType(200): "unknown",
	}
	for k, v := range contractSettlementTypeToStringMap {
		assert.Equal(t, v, k.String())
	}
}

func TestContractTypeToString(t *testing.T) {
	t.Parallel()
	contractTypeToStringMap := map[ContractType]string{
		Daily:             "day",
		Perpetual:         "perpetual",
		LongDated:         "long_dated",
		Weekly:            "weekly",
		Fortnightly:       "fortnightly",
		ThreeWeekly:       "three-weekly",
		Monthly:           "monthly",
		Quarterly:         "quarterly",
		SemiAnnually:      "semi-annually",
		HalfYearly:        "half-yearly",
		NineMonthly:       "nine-monthly",
		Yearly:            "yearly",
		Unknown:           "unknown",
		UnsetContractType: "unset",
		ContractType(200): "unset",
	}
	for k, v := range contractTypeToStringMap {
		assert.Equal(t, v, k.String())
	}
}
