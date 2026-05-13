package zklink

import (
	"math/big"
)

// ContractBuilder holds a contract builder parameters
type ContractBuilder struct {
	AccountID    *big.Int
	SubAccountID *big.Int
	SlotID       *big.Int
	Nonce        *big.Int
	PairID       *big.Int
	Size         *big.Int
	Price        *big.Int
	Direction    bool
	TakerFeeRate *big.Int
	MakerFeeRate *big.Int
	HasSubsidy   bool
}

// WithdrawBuilder holds an asset withdrawal builder parameters
type WithdrawBuilder struct {
	AccountID        *big.Int
	SubAccountID     *big.Int
	ToChainID        *big.Int
	ToAddress        *big.Int
	L2SourceToken    *big.Int // 16-bit field: L2 token ID (e.g. USDT=140)
	L1TargetToken    *big.Int // 16-bit field: L1 target token ID
	Amount           *big.Int
	Fee              *big.Int
	Nonce            *big.Int
	WithdrawFeeRatio *big.Int
	WithdrawToL1     bool
	Timestamp        *big.Int
}

// TransferBuilder holds an asset transfer builder parameters for zklink signature
type TransferBuilder struct {
	AccountID        *big.Int
	ToAddress        *big.Int
	FromSubAccountID *big.Int
	ToSubAccountID   *big.Int
	Token            *big.Int
	Amount           *big.Int
	Fee              *big.Int
	Nonce            *big.Int
	Timestamp        *big.Int
}
