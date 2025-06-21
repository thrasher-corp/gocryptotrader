package zklink

import (
	"math/big"

	"github.com/thrasher-corp/gocryptotrader/internal/utils/zklink/bn256/fr"
)

const (
	PACKED_POINT_SIZE         = 32
	SIGNATURE_SIZE            = 96
	NEW_PUBKEY_HASH_BYTES_LEN = 20
	NEW_PUBKEY_HASH_WIDTH     = NEW_PUBKEY_HASH_BYTES_LEN * 8
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
	AccountID    *big.Int
	SubAccountID *big.Int
	ToChainID    *big.Int
	ToAddress    *big.Int
	// L2SourceToken TokenId
	// L1TargetToken TokenId
	Amount *big.Int
	// DataHash         *H256
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

// Bn256RescueParams represents capacity, rate of hash, and other details of the rescue hash algorithm
type Bn256RescueParams struct {
	C              uint32
	R              uint32
	Rounds         uint32
	SecurityLevel  uint32
	RoundConstants []*fr.Element
	MDSMatrix      []*fr.Element
	SBox0          *PowerSBox
	SBox1          *QuinticSBox

	CustomGatesAllowed bool
}
