package starkex

import "math/big"

const (
	ONE_HOUR_IN_SECONDS                     = 60 * 60
	ORDER_SIGNATURE_EXPIRATION_BUFFER_HOURS = 24 * 7 // Seven days.
	ORDER_PREFIX                            = 3
	CONDITIONAL_TRANSFER_PREFIX             = 5
	TRANSFER_PREFIX                         = 4
	WITHDRAWAL_TO_ADDRESS_PREFIX            = 7
	WITHDRAWAL_PREFIX                       = 6

	ORDER_PADDING_BITS                = 17
	WITHDRAWAL_PADDING_BITS           = 49
	CONDITIONAL_TRANSFER_PADDING_BITS = 81
	TRANSFER_PADDING_BITS             = 81

	CONDITIONAL_TRANSFER_FEE_ASSET_ID   = 0
	CONDITIONAL_TRANSFER_MAX_AMOUNT_FEE = 0
)

var (
	// N_ELEMENT_BITS_ECDSA math.floor(math.log(FIELD_PRIME, 2))
	N_ELEMENT_BITS_ECDSA = big.NewInt(251)

	// BIT_MASK_250 (2 ** 250) - 1
	BIT_MASK_250 = big.NewInt(0).Sub(big.NewInt(0).Exp(big.NewInt(2), big.NewInt(250), nil), one)

	ORDER_FIELD_BIT_LENGTHS = map[string]uint{
		"asset_id_synthetic":     128,
		"asset_id_collateral":    250,
		"asset_id_fee":           250,
		"quantums_amount":        64,
		"nonce":                  32,
		"position_id":            64,
		"expiration_epoch_hours": 32,
	}

	WITHDRAWAL_FIELD_BIT_LENGTHS = map[string]uint{
		"asset_id":               250,
		"position_id":            64,
		"nonce":                  32,
		"quantums_amount":        64,
		"expiration_epoch_hours": 32,
	}

	TRANSFER_FIELD_BIT_LENGTHS = map[string]uint{
		"asset_id":               250,
		"receiver_public_key":    251,
		"position_id":            64,
		"quantums_amount":        64,
		"nonce":                  32,
		"expiration_epoch_hours": 32,
	}

	CONDITIONAL_TRANSFER_FIELD_BIT_LENGTHS = map[string]uint{
		"asset_id":               250,
		"receiver_public_key":    251,
		"position_id":            64,
		"condition":              251,
		"quantums_amount":        64,
		"nonce":                  32,
		"expiration_epoch_hours": 32,
	}
)
