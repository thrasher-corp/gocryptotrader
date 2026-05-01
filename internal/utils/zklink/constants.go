package zklink

// constants for zklink signing parameters
const (
	contractMsgType = 254
	withdrawMsgType = 3
	transferMsgType = 4
)

var (
	// ContractFieldBitLengths represents contract fields bit length in constructing a musig Schnorr signature payload
	ContractFieldBitLengths = map[string]uint{
		"type":         8,
		"accountId":    32,
		"subAccountId": 8,
		"slotId":       16,
		"nonce":        24,
		"pairId":       16,
		"direction":    8,
		"size":         40,
		"price":        120,
		"feeRates":     16,
		"hasSubsidy":   8,
	}

	// WithdrawFieldBitLengths holds asset withdrawal fields bit length constructing a musig Schnorr signature payload
	WithdrawFieldBitLengths = map[string]uint{
		"type":             8,
		"toChainId":        8,
		"accountId":        36,
		"subAccountId":     8,
		"to":               256,
		"l2SourceToken":    16,
		"l1TargetToken":    16,
		"amount":           128,
		"fee":              16,
		"nonce":            32,
		"withdrawToL1":     8,
		"withdrawFeeRatio": 16,
		// "callData": No limit, be used to send to layer1 to call a smart contract
		"ts": 32,
	}

	// TransferFieldBigLengths holds asset withdrawal fields big length constructing a musig Schnorr signature
	TransferFieldBigLengths = map[string]uint{
		"type":             8,
		"accountId":        32,
		"fromSubAccountId": 8,
		"to":               256,
		"toSubAccountId":   8,
		"token":            16,
		"amount":           40,
		"feeAmount":        16,
		"nonce":            32,
		"ts":               32,
	}
)

// GHFirstBlock First 64 bytes of the BLAKE2s input during group hash.
// This is chosen to be some random string that we couldn't have anticipated when we designed
// the algorithm, for rigidity purposes.
// We deliberately use an ASCII hex string of 32 bytes here.
var GHFirstBlock = []byte("096b36a5804bfacef1691e173c366a47ff5ba84a44f26ddd7e8d9f79d5b42df0")
