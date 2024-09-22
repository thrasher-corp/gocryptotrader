package starkex

import "math/big"

// Signable an interface for hashing and signing with starkex ware
type Signable interface {
	GetPedersenHash(pedersenHash func(...string) string) (string, error)
}

// GetPedersenHash implements the Signable interface and generates a pedersen hash of CreateOrderWithFeeParams
func (s *CreateOrderWithFeeParams) GetPedersenHash(pedersenHash func(...string) string) (string, error) {
	var assetIdSell, assetIdBuy, quantumsAmountSell, quantumsAmountBuy *big.Int
	if s.IsBuyingSynthetic {
		assetIdSell = s.AssetIDCollateral
		assetIdBuy = s.AssetIDSynthetic
		quantumsAmountSell = s.QuantumAmountCollateral
		quantumsAmountBuy = s.QuantumAmountSynthetic
	} else {
		assetIdSell = s.AssetIDSynthetic
		assetIdBuy = s.AssetIDCollateral
		quantumsAmountSell = s.QuantumAmountSynthetic
		quantumsAmountBuy = s.QuantumAmountCollateral
	}
	// Part 1
	part1 := big.NewInt(0)
	part1.Add(part1, quantumsAmountSell)
	part1.Lsh(part1, ORDER_FIELD_BIT_LENGTHS["quantums_amount"])
	part1.Add(part1, quantumsAmountBuy)
	part1.Lsh(part1, ORDER_FIELD_BIT_LENGTHS["quantums_amount"])
	part1.Add(part1, s.QuantumAmountFee)
	part1.Lsh(part1, ORDER_FIELD_BIT_LENGTHS["nonce"])
	part1.Add(part1, s.Nonce)

	// Part 2
	part2 := big.NewInt(ORDER_PREFIX)
	for i := 0; i < 3; i++ {
		part2.Lsh(part2, uint(ORDER_FIELD_BIT_LENGTHS["position_id"]))
		part2.Add(part2, s.PositionID)
	}
	part2.Lsh(part2, uint(ORDER_FIELD_BIT_LENGTHS["expiration_epoch_hours"]))
	part2.Add(part2, s.ExpirationEpochHours)
	part2.Lsh(part2, uint(ORDER_PADDING_BITS))

	assetHash := pedersenHash(
		pedersenHash(
			assetIdSell.String(),
			assetIdBuy.String(),
		),
		s.AssetIDFee.String(),
	)
	part1Hash := pedersenHash(
		assetHash,
		part1.String(),
	)
	return pedersenHash(
		part1Hash,
		part2.String(),
	), nil
}

// GetPedersenHash implements the Signable interface and generates a pedersen hash of WithdrawalToAddressParams
func (s *WithdrawalToAddressParams) GetPedersenHash(pedersenHash func(...string) string) (string, error) {
	// packed
	packed := big.NewInt(WITHDRAWAL_TO_ADDRESS_PREFIX)
	packed.Lsh(packed, WITHDRAWAL_FIELD_BIT_LENGTHS["position_id"])
	packed.Add(packed, s.PositionID)
	packed.Lsh(packed, WITHDRAWAL_FIELD_BIT_LENGTHS["nonce"])
	packed.Add(packed, s.Nonce)
	packed.Lsh(packed, WITHDRAWAL_FIELD_BIT_LENGTHS["quantums_amount"])
	packed.Add(packed, s.Amount)
	packed.Lsh(packed, WITHDRAWAL_FIELD_BIT_LENGTHS["expiration_epoch_hours"])
	packed.Add(packed, s.ExpirationEpochHours)
	packed.Lsh(packed, WITHDRAWAL_PADDING_BITS)
	// pedersen hash
	return pedersenHash(pedersenHash(s.AssetIDCollateral.String(), s.EthAddress.String()), packed.String()), nil
}

// GetPedersenHash implements the Signable interface and generates a pedersen hash of WithdrawalParams
func (s *WithdrawalParams) GetPedersenHash(pedersenHash func(...string) string) (string, error) {
	// packed
	packed := big.NewInt(WITHDRAWAL_PREFIX)
	packed.Lsh(packed, WITHDRAWAL_FIELD_BIT_LENGTHS["position_id"])
	packed.Add(packed, s.PositionID)
	packed.Lsh(packed, WITHDRAWAL_FIELD_BIT_LENGTHS["nonce"])
	packed.Add(packed, s.Nonce)
	packed.Lsh(packed, WITHDRAWAL_FIELD_BIT_LENGTHS["quantums_amount"])
	packed.Add(packed, s.Amount)
	packed.Lsh(packed, WITHDRAWAL_FIELD_BIT_LENGTHS["expiration_epoch_hours"])
	packed.Add(packed, s.ExpirationEpochHours)
	packed.Lsh(packed, WITHDRAWAL_PADDING_BITS)
	// pedersen hash
	return pedersenHash(s.AssetIDCollateral.String(), packed.String()), nil
}

// GetPedersenHash implements the Signable interface and generates a pedersen hash of TransferParams
func (s *TransferParams) GetPedersenHash(pedersenHash func(...string) string) (string, error) {
	assetID := pedersenHash(s.AssetID.String(), s.AssetIDFee.String())
	// packed
	part1 := pedersenHash(assetID, s.ReceiverPublicKey.String())
	// packed
	part2 := big.NewInt(0).Set(s.SenderPositionID)
	part2.Lsh(part2, TRANSFER_FIELD_BIT_LENGTHS["position_id"])
	part2.Add(part2, s.ReceiverPositionID)
	part2.Lsh(part2, TRANSFER_FIELD_BIT_LENGTHS["position_id"])
	part2.Add(part2, s.SenderPositionID)
	part2.Lsh(part2, TRANSFER_FIELD_BIT_LENGTHS["nonce"])
	part2.Add(part2, s.Nonce)

	part3 := big.NewInt(TRANSFER_PREFIX)
	part3.Lsh(part3, TRANSFER_FIELD_BIT_LENGTHS["quantums_amount"])
	part3.Add(part3, s.QuantumsAmount)
	part3.Lsh(part3, TRANSFER_FIELD_BIT_LENGTHS["quantums_amount"])
	part3.Add(part3, s.MaxAmountFee)
	part3.Lsh(part3, TRANSFER_FIELD_BIT_LENGTHS["expiration_epoch_hours"])
	part3.Add(part3, s.ExpirationEpochHours)
	part3.Lsh(part3, TRANSFER_PADDING_BITS)

	return pedersenHash(
		pedersenHash(
			part1,
			part2.String(),
		),
		part3.String(),
	), nil
}

// GetPedersenHash implements the Signable interface and generates a pedersen hash of ConditionalTransferParams
func (s *ConditionalTransferParams) GetPedersenHash(pedersenHash func(...string) string) (string, error) {
	assetID := pedersenHash(s.AssetID.String(), s.AssetIDFee.String())

	// packed
	part1 := pedersenHash(pedersenHash(assetID, s.ReceiverPublicKey.String()), s.Condition.String())

	// part 2
	part2 := big.NewInt(0).Set(s.SenderPositionID)
	part2.Lsh(part2, CONDITIONAL_TRANSFER_FIELD_BIT_LENGTHS["position_id"])
	part2.Add(part2, s.ReceiverPositionID)
	part2.Lsh(part2, CONDITIONAL_TRANSFER_FIELD_BIT_LENGTHS["position_id"])
	part2.Add(part2, s.SenderPositionID)
	part2.Lsh(part2, CONDITIONAL_TRANSFER_FIELD_BIT_LENGTHS["nonce"])
	part2.Add(part2, s.Nonce)

	// part 3
	part3 := big.NewInt(CONDITIONAL_TRANSFER_PREFIX)
	part3.Lsh(part3, CONDITIONAL_TRANSFER_FIELD_BIT_LENGTHS["quantums_amount"])
	part3.Add(part3, s.QuantumsAmount)
	part3.Lsh(part3, CONDITIONAL_TRANSFER_FIELD_BIT_LENGTHS["quantums_amount"])
	part3.Add(part3, big.NewInt(CONDITIONAL_TRANSFER_MAX_AMOUNT_FEE))
	part3.Lsh(part3, CONDITIONAL_TRANSFER_FIELD_BIT_LENGTHS["expiration_epoch_hours"])
	part3.Add(part3, s.ExpTimestampHrs)
	part3.Lsh(part3, CONDITIONAL_TRANSFER_PADDING_BITS)

	return pedersenHash(
		pedersenHash(
			part1,
			part2.String(),
		),
		part3.String(),
	), nil
}
