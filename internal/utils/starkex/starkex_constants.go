package starkex

import "math/big"

// StarkEx configuration constants
const (
	OneHourInSeconds                    = 60 * 60
	OrderSignatureExpirationBufferHours = 24 * 7
	OrderPrefix                         = 3
	ConditionalTransferPrefix           = 5
	TransferPrefix                      = 4
	WithdrawalToAddressPrefix           = 7
	WithdrawalPrefix                    = 6

	OrderPaddingBits               = 17
	WithdrawalPaddingBits          = 49
	ConditionalTransferPaddingBits = 81
	TransferPaddingBits            = 81

	ConditionalTransferFeeAssetID   = 0
	ConditionalTransferMaxAmountFee = 0
)

// Pedersen configuration constants
var (
	// NElementBitsECDSA math.floor(math.log(FIELD_PRIME, 2))
	NElementBitsECDSA = big.NewInt(251)

	// BitMask250 (2 ** 250) - 1
	BitMask250 = big.NewInt(0).Sub(big.NewInt(0).Exp(big.NewInt(2), big.NewInt(250), nil), one)

	// OrderFieldBitLengths represents order fields bit length in constructing a pedersen hash payload
	OrderFieldBitLengths = map[string]uint{
		"asset_id_synthetic":     128,
		"asset_id_collateral":    250,
		"asset_id_fee":           250,
		"quantums_amount":        64,
		"nonce":                  32,
		"position_id":            64,
		"expiration_epoch_hours": 32,
	}

	WithdrawalFieldBitLengths = map[string]uint{
		"asset_id":               250,
		"position_id":            64,
		"nonce":                  32,
		"quantums_amount":        64,
		"expiration_epoch_hours": 32,
	}

	TransferFieldBitLengths = map[string]uint{
		"asset_id":               250,
		"receiver_public_key":    251,
		"position_id":            64,
		"quantums_amount":        64,
		"nonce":                  32,
		"expiration_epoch_hours": 32,
	}

	ConditionalTransferFieldBitLengths = map[string]uint{
		"asset_id":               250,
		"receiver_public_key":    251,
		"position_id":            64,
		"condition":              251,
		"quantums_amount":        64,
		"nonce":                  32,
		"expiration_epoch_hours": 32,
	}
)
