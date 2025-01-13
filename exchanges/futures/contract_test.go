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
	var val ContractSettlementType
	var err error
	for x := range contractSettlementTypesMap {
		val, err = StringToContractSettlementType(x)
		assert.Equalf(t, val, contractSettlementTypesMap[x].CT, "got %v, expected %v", val, contractSettlementTypesMap[x].CT)
		assert.ErrorIs(t, err, contractSettlementTypesMap[x].Error)
	}
}
