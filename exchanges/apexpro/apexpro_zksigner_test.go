package apexpro

import (
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/internal/utils/zklink"
)

// knownSeedsHex is a deterministic test seed (does NOT represent a real key).
const knownSeedsHex = "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"

func TestNewZKLinkSignerFromSeeds(t *testing.T) {
	t.Parallel()

	seeds, err := hex.DecodeString(knownSeedsHex)
	require.NoError(t, err)

	signer, err := zklink.NewZKLinkSignerFromSeeds(seeds)
	require.NoError(t, err)
	require.NotNil(t, signer)

	pubBytes := signer.PublicKeyBytes()
	var zero [32]byte
	assert.NotEqual(t, zero, pubBytes, "public key should not be zero")

	signer2, err := zklink.NewZKLinkSignerFromSeeds(seeds)
	require.NoError(t, err)
	assert.Equal(t, pubBytes, signer2.PublicKeyBytes(), "key derivation must be deterministic")
}

func TestNewZKLinkSignerFromSeedsErrors(t *testing.T) {
	t.Parallel()

	_, err := zklink.NewZKLinkSignerFromSeeds(nil)
	assert.Error(t, err, "nil seeds should error")

	_, err = zklink.NewZKLinkSignerFromSeeds([]byte{})
	assert.Error(t, err, "empty seeds should error")
}

func TestZKLinkSignerSign(t *testing.T) {
	t.Parallel()

	seeds, err := hex.DecodeString(knownSeedsHex)
	require.NoError(t, err)

	signer, err := zklink.NewZKLinkSignerFromSeeds(seeds)
	require.NoError(t, err)

	msg := big.NewInt(12345678)
	sig1, err := signer.Sign(msg)
	require.NoError(t, err)
	assert.Len(t, sig1, 64, "signature must be 64 bytes")

	sig2, err := signer.Sign(msg)
	require.NoError(t, err)
	assert.Equal(t, sig1, sig2, "signing same message must be deterministic")

	sig3, err := signer.Sign(big.NewInt(87654321))
	require.NoError(t, err)
	assert.NotEqual(t, sig1, sig3, "different messages must produce different signatures")
}

func TestContractBuilderGetBytes(t *testing.T) {
	t.Parallel()

	builder := &zklink.ContractBuilder{
		AccountID:    big.NewInt(1000),
		SubAccountID: big.NewInt(1),
		SlotID:       big.NewInt(0),
		Nonce:        big.NewInt(5),
		PairID:       big.NewInt(2),
		Size:         big.NewInt(100000000),
		Price:        big.NewInt(3000000000),
		Direction:    true,
		TakerFeeRate: big.NewInt(20),
		MakerFeeRate: big.NewInt(10),
		HasSubsidy:   false,
	}

	msgBytes := builder.GetBytes()
	require.NotNil(t, msgBytes)
	assert.True(t, msgBytes.BitLen() > 0, "message bytes must be non-zero")
}

func TestWithdrawBuilderGetBytes(t *testing.T) {
	t.Parallel()
	toAddr, _ := new(big.Int).SetString("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", 0)
	builder := &zklink.WithdrawBuilder{
		AccountID:        big.NewInt(1000),
		SubAccountID:     big.NewInt(1),
		ToChainID:        big.NewInt(1),
		ToAddress:        toAddr,
		L2SourceToken:    big.NewInt(140), // USDT token ID
		L1TargetToken:    big.NewInt(140),
		Amount:           big.NewInt(100_000_000), // 1 USDT (8 decimals)
		Fee:              big.NewInt(100_000),
		Nonce:            big.NewInt(3),
		WithdrawFeeRatio: big.NewInt(0),
		WithdrawToL1:     true,
		Timestamp:        big.NewInt(time.Now().Unix()),
	}

	msgBytes := builder.GetBytes()
	require.NotNil(t, msgBytes)
	assert.True(t, msgBytes.BitLen() > 0, "message bytes must be non-zero")
}

func TestRescueHashBigInt(t *testing.T) {
	t.Parallel()
	msg := big.NewInt(0).Lsh(big.NewInt(1), 200)
	result := zklink.RescueHashBigInt(msg)
	require.NotNil(t, result)

	result2 := zklink.RescueHashBigInt(msg)
	assert.Equal(t, result.Bytes(), result2.Bytes(), "hash must be deterministic")

	result3 := zklink.RescueHashBigInt(big.NewInt(0).Add(msg, big.NewInt(1)))
	assert.NotEqual(t, result.Bytes(), result3.Bytes(), "different inputs must produce different hashes")
}

func TestRescueHashBigIntDifferentSizes(t *testing.T) {
	t.Parallel()

	for _, bits := range []uint{8, 128, 248, 296, 580} {
		msg := new(big.Int).Lsh(big.NewInt(1), bits)
		result := zklink.RescueHashBigInt(msg)
		require.NotNilf(t, result, "hash of %d-bit message must not be nil", bits)
	}
}

func TestGetOrInitZKLinkerSigner_MissingCredentials(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	_, err := ex.getOrInitZKLinkerSigner("")
	assert.Error(t, err)
}

func TestProcessZKKeyOrderSignature_MissingCredentials(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	_, err := ex.ProcessZKKeyOrderSignature(t.Context(), &CreateOrderParams{
		Symbol: currency.NewPairWithDelimiter("BTC", "USDT", "-"),
		Side:   "BUY",
	})
	assert.Error(t, err)
}

func TestProcessZKKeyWithdrawalSignature_MissingCredentials(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	_, err := ex.ProcessZKKeyWithdrawalSignature(t.Context(), &AssetWithdrawalParams{
		Amount:          1.0,
		L2SourceTokenID: currency.USDT,
		L1TargetTokenID: currency.USDT,
	})
	assert.Error(t, err)
}

func TestProcessZKKeyOrderSignature_Integration(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	result, err := e.ProcessZKKeyOrderSignature(t.Context(), &CreateOrderParams{
		Symbol:    currency.NewPairWithDelimiter("BTC", "USDT", "-"),
		Side:      "BUY",
		OrderType: "LIMIT",
		Size:      0.001,
		Price:     30000.0,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Len(t, result, 128, "signature must be 128 hex chars (64 bytes)")
}
