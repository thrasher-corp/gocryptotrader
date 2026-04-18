package zklink

import "math/big"

// Signable implements a GetBytes method which extracts bytes from zklink request parameters
type Signable interface {
	GetBytes() *big.Int
}

// GetBytes implements the signable interface
func (c *ContractBuilder) GetBytes() *big.Int {
	payload := big.NewInt(0)
	payload.Add(payload, big.NewInt(CONTRACT_MSG_TYPE))
	payload.Lsh(payload, ContractFieldBitLengths["accountId"])
	payload.Add(payload, c.AccountID)
	payload.Lsh(payload, ContractFieldBitLengths["subAccountId"])
	payload.Add(payload, c.SubAccountID)

	payload.Lsh(payload, ContractFieldBitLengths["slotId"])
	payload.Add(payload, c.SlotID)
	payload.Lsh(payload, ContractFieldBitLengths["nonce"])
	payload.Add(payload, c.Nonce)
	payload.Lsh(payload, ContractFieldBitLengths["pairId"])
	payload.Add(payload, c.PairID)
	payload.Lsh(payload, ContractFieldBitLengths["direction"])
	payload.Add(payload, boolToBigInt(c.Direction))
	payload.Lsh(payload, ContractFieldBitLengths["size"])
	payload.Add(payload, c.Size)
	payload.Lsh(payload, ContractFieldBitLengths["price"])
	payload.Add(payload, c.Price)
	payload.Lsh(payload, ContractFieldBitLengths["feeRates"])

	feeRates := new(big.Int).Lsh(c.MakerFeeRate, ContractFieldBitLengths["feeRates"]/2)
	feeRates.Add(feeRates, c.TakerFeeRate)

	payload.Add(payload, feeRates)
	payload.Lsh(payload, ContractFieldBitLengths["hasSubsidy"])
	return payload.Add(payload, boolToBigInt(c.HasSubsidy))
}

func boolToBigInt(b bool) *big.Int {
	if b {
		return big.NewInt(1) // If true, return 1 as big.Int
	}
	return big.NewInt(0) // If false, return 0 as big.Int
}

// GetBytes implements the signable interface
func (w *WithdrawBuilder) GetBytes() *big.Int {
	payload := big.NewInt(0)
	payload.Add(payload, big.NewInt(WITHDRAW_MSG_TYPE))
	payload.Lsh(payload, WithdrawFieldBitLengths["toChainId"])
	payload.Add(payload, w.ToChainID)
	payload.Lsh(payload, WithdrawFieldBitLengths["accountId"])
	payload.Add(payload, w.AccountID)
	payload.Lsh(payload, WithdrawFieldBitLengths["subAccountId"])
	payload.Add(payload, w.SubAccountID)
	payload.Lsh(payload, WithdrawFieldBitLengths["to"])
	payload.Add(payload, w.ToAddress)
	payload.Lsh(payload, WithdrawFieldBitLengths["l2SourceToken"])
	payload.Add(payload, w.L2SourceToken)
	payload.Lsh(payload, WithdrawFieldBitLengths["l1TargetToken"])
	payload.Add(payload, w.L1TargetToken)

	payload.Lsh(payload, WithdrawFieldBitLengths["amount"])
	payload.Add(payload, w.Amount)
	payload.Lsh(payload, WithdrawFieldBitLengths["fee"])
	payload.Add(payload, w.Fee)
	payload.Lsh(payload, WithdrawFieldBitLengths["nonce"])
	payload.Add(payload, w.Nonce)
	payload.Lsh(payload, WithdrawFieldBitLengths["withdrawToL1"])

	payload.Lsh(payload, WithdrawFieldBitLengths["withdrawFeeRatio"])
	payload.Add(payload, w.WithdrawFeeRatio)

	payload.Lsh(payload, WithdrawFieldBitLengths["callData"])

	payload.Lsh(payload, WithdrawFieldBitLengths["ts"])
	return payload.Add(payload, w.Timestamp)
}

// GetBytes implements the signable interface
func (t *TransferBuilder) GetBytes() *big.Int {
	payload := big.NewInt(0)
	payload.Add(payload, big.NewInt(TRANSFER_MSG_TYPE))
	payload.Lsh(payload, TransferFieldBigLengths["accountId"])
	payload.Add(payload, t.AccountID)
	payload.Lsh(payload, TransferFieldBigLengths["fromSubAccountId"])
	payload.Add(payload, t.FromSubAccountID)
	payload.Lsh(payload, TransferFieldBigLengths["to"])
	payload.Add(payload, t.ToAddress)
	payload.Lsh(payload, TransferFieldBigLengths["toSubAccountId"])
	payload.Add(payload, t.ToSubAccountID)
	payload.Lsh(payload, TransferFieldBigLengths["token"])
	payload.Add(payload, t.Token)
	payload.Lsh(payload, TransferFieldBigLengths["amount"])
	payload.Add(payload, t.Amount)
	payload.Lsh(payload, TransferFieldBigLengths["feeAmount"])
	payload.Add(payload, t.Fee)
	payload.Lsh(payload, TransferFieldBigLengths["nonce"])
	payload.Add(payload, t.Nonce)
	payload.Lsh(payload, TransferFieldBigLengths["ts"])
	return payload.Add(payload, t.Timestamp)
}
