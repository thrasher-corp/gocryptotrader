package starkex

import "math/big"

// Signable an interface for hashing and signing with starkex ware
type Signable interface {
	GetPedersenHash(pedersenHash func(...string) string) (string, error)
}

// GetPedersenHash implements the Signable interface and generates a pedersen hash of CreateOrderWithFeeParams
func (s *CreateOrderWithFeeParams) GetPedersenHash(pedersenHash func(...string) string) (string, error) {
	var assetIDSell, assetIDBuy, quantumsAmountSell, quantumsAmountBuy *big.Int
	if s.IsBuyingSynthetic {
		assetIDSell = s.AssetIDCollateral
		assetIDBuy = s.AssetIDSynthetic
		quantumsAmountSell = s.QuantumAmountCollateral
		quantumsAmountBuy = s.QuantumAmountSynthetic
	} else {
		assetIDSell = s.AssetIDSynthetic
		assetIDBuy = s.AssetIDCollateral
		quantumsAmountSell = s.QuantumAmountSynthetic
		quantumsAmountBuy = s.QuantumAmountCollateral
	}
	// Part 1
	part1 := big.NewInt(0)
	part1.Add(part1, quantumsAmountSell)
	part1.Lsh(part1, OrderFieldBitLengths["quantums_amount"])
	part1.Add(part1, quantumsAmountBuy)
	part1.Lsh(part1, OrderFieldBitLengths["quantums_amount"])
	part1.Add(part1, s.QuantumAmountFee)
	part1.Lsh(part1, OrderFieldBitLengths["nonce"])
	part1.Add(part1, s.Nonce)
	// Part 2
	part2 := big.NewInt(OrderPrefix)
	for range 3 {
		part2.Lsh(part2, OrderFieldBitLengths["position_id"])
		part2.Add(part2, s.PositionID)
	}
	part2.Lsh(part2, OrderFieldBitLengths["expiration_epoch_hours"])
	part2.Add(part2, s.ExpirationEpochHours)
	part2.Lsh(part2, uint(OrderPaddingBits))
	assetHash := pedersenHash(
		pedersenHash(
			assetIDSell.String(),
			assetIDBuy.String(),
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
	packed := big.NewInt(WithdrawalToAddressPrefix)
	packed.Lsh(packed, WithdrawalFieldBitLengths["position_id"])
	packed.Add(packed, s.PositionID)
	packed.Lsh(packed, WithdrawalFieldBitLengths["nonce"])
	packed.Add(packed, s.Nonce)
	packed.Lsh(packed, WithdrawalFieldBitLengths["quantums_amount"])
	packed.Add(packed, s.Amount)
	packed.Lsh(packed, WithdrawalFieldBitLengths["expiration_epoch_hours"])
	packed.Add(packed, s.ExpirationEpochHours)
	packed.Lsh(packed, WithdrawalPaddingBits)
	// pedersen hash
	return pedersenHash(pedersenHash(s.AssetIDCollateral.String(), s.EthAddress.String()), packed.String()), nil
}

// GetPedersenHash implements the Signable interface and generates a pedersen hash of WithdrawalParams
func (s *WithdrawalParams) GetPedersenHash(pedersenHash func(...string) string) (string, error) {
	// packed
	packed := big.NewInt(WithdrawalPrefix)
	packed.Lsh(packed, WithdrawalFieldBitLengths["position_id"])
	packed.Add(packed, s.PositionID)
	packed.Lsh(packed, WithdrawalFieldBitLengths["nonce"])
	packed.Add(packed, s.Nonce)
	packed.Lsh(packed, WithdrawalFieldBitLengths["quantums_amount"])
	packed.Add(packed, s.Amount)
	packed.Lsh(packed, WithdrawalFieldBitLengths["expiration_epoch_hours"])
	packed.Add(packed, s.ExpirationEpochHours)
	packed.Lsh(packed, WithdrawalPaddingBits)
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
	part2.Lsh(part2, TransferFieldBitLengths["position_id"])
	part2.Add(part2, s.ReceiverPositionID)
	part2.Lsh(part2, TransferFieldBitLengths["position_id"])
	part2.Add(part2, s.SenderPositionID)
	part2.Lsh(part2, TransferFieldBitLengths["nonce"])
	part2.Add(part2, s.Nonce)

	part3 := big.NewInt(TransferPrefix)
	part3.Lsh(part3, TransferFieldBitLengths["quantums_amount"])
	part3.Add(part3, s.QuantumsAmount)
	part3.Lsh(part3, TransferFieldBitLengths["quantums_amount"])
	part3.Add(part3, s.MaxAmountFee)
	part3.Lsh(part3, TransferFieldBitLengths["expiration_epoch_hours"])
	part3.Add(part3, s.ExpirationEpochHours)
	part3.Lsh(part3, TransferPaddingBits)

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
	part2.Lsh(part2, ConditionalTransferFieldBitLengths["position_id"])
	part2.Add(part2, s.ReceiverPositionID)
	part2.Lsh(part2, ConditionalTransferFieldBitLengths["position_id"])
	part2.Add(part2, s.SenderPositionID)
	part2.Lsh(part2, ConditionalTransferFieldBitLengths["nonce"])
	part2.Add(part2, s.Nonce)

	// part 3
	part3 := big.NewInt(ConditionalTransferPrefix)
	part3.Lsh(part3, ConditionalTransferFieldBitLengths["quantums_amount"])
	part3.Add(part3, s.QuantumsAmount)
	part3.Lsh(part3, ConditionalTransferFieldBitLengths["quantums_amount"])
	part3.Add(part3, big.NewInt(ConditionalTransferMaxAmountFee))
	part3.Lsh(part3, ConditionalTransferFieldBitLengths["expiration_epoch_hours"])
	part3.Add(part3, s.ExpTimestampHrs)
	part3.Lsh(part3, ConditionalTransferPaddingBits)

	return pedersenHash(
		pedersenHash(
			part1,
			part2.String(),
		),
		part3.String(),
	), nil
}
