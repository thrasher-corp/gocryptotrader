package zklink

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/internal/utils/zklink/bn256/fr"
)

func TestRescuePermute(t *testing.T) {
	t.Parallel()

	t.Run("changes zero state", func(t *testing.T) {
		t.Parallel()
		var state [rescueStateWidth]fr.Element
		RescuePermute(&state)
		var zero fr.Element
		assert.False(t, state[0].Equal(&zero), "permutation of zero state should produce non-zero output")
	})

	t.Run("deterministic", func(t *testing.T) {
		t.Parallel()
		var s1, s2 [rescueStateWidth]fr.Element
		s1[0].SetUint64(42)
		s2[0].SetUint64(42)
		RescuePermute(&s1)
		RescuePermute(&s2)
		assert.True(t, s1[0].Equal(&s2[0]), "same input should produce same output at position 0")
		assert.True(t, s1[1].Equal(&s2[1]), "same input should produce same output at position 1")
		assert.True(t, s1[2].Equal(&s2[2]), "same input should produce same output at position 2")
	})

	t.Run("distinct inputs produce distinct outputs", func(t *testing.T) {
		t.Parallel()
		var s1, s2 [rescueStateWidth]fr.Element
		s1[0].SetUint64(1)
		s2[0].SetUint64(2)
		RescuePermute(&s1)
		RescuePermute(&s2)
		assert.False(t, s1[0].Equal(&s2[0]), "distinct inputs should produce distinct outputs")
	})

	t.Run("modifies all state positions", func(t *testing.T) {
		t.Parallel()
		var state [rescueStateWidth]fr.Element
		state[0].SetUint64(1)
		original := state
		RescuePermute(&state)
		for i := range rescueStateWidth {
			assert.Falsef(t, state[i].Equal(&original[i]), "permutation should modify element at position %d", i)
		}
	})
}

func TestRescueHash(t *testing.T) {
	t.Parallel()

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()
		result := RescueHash([]fr.Element{})
		require.NotNil(t, result, "hash of empty input must not be nil")
		var zero fr.Element
		assert.False(t, result.Equal(&zero), "hash of empty input should be non-zero")
	})

	t.Run("single element", func(t *testing.T) {
		t.Parallel()
		a := make([]fr.Element, 1)
		a[0].SetUint64(1)
		b := make([]fr.Element, 1)
		ra := RescueHash(a)
		rb := RescueHash(b)
		require.NotNil(t, ra, "hash must not be nil")
		require.NotNil(t, rb, "hash must not be nil")
		assert.False(t, ra.Equal(rb), "hash of 1 should differ from hash of 0")
	})

	t.Run("two elements fills one rate block", func(t *testing.T) {
		t.Parallel()
		input := make([]fr.Element, 2)
		input[0].SetUint64(10)
		input[1].SetUint64(20)
		result := RescueHash(input)
		require.NotNil(t, result, "hash of two-element input must not be nil")
	})

	t.Run("three elements spans two blocks", func(t *testing.T) {
		t.Parallel()
		input := make([]fr.Element, 3)
		input[0].SetUint64(10)
		input[1].SetUint64(20)
		input[2].SetUint64(30)
		result := RescueHash(input)
		require.NotNil(t, result, "hash of three-element input must not be nil")
	})

	t.Run("deterministic", func(t *testing.T) {
		t.Parallel()
		input := make([]fr.Element, 3)
		input[0].SetUint64(7)
		input[1].SetUint64(13)
		input[2].SetUint64(99)
		r1 := RescueHash(input)
		r2 := RescueHash(input)
		require.NotNil(t, r1, "hash must not be nil")
		assert.True(t, r1.Equal(r2), "repeated calls with same input should produce same hash")
	})

	t.Run("distinct inputs produce distinct hashes", func(t *testing.T) {
		t.Parallel()
		a := make([]fr.Element, 2)
		b := make([]fr.Element, 2)
		a[0].SetUint64(1)
		b[0].SetUint64(2)
		ra := RescueHash(a)
		rb := RescueHash(b)
		require.NotNil(t, ra, "hash must not be nil")
		assert.False(t, ra.Equal(rb), "distinct inputs should produce distinct hashes")
	})

	t.Run("input length affects output", func(t *testing.T) {
		t.Parallel()
		short := make([]fr.Element, 1)
		short[0].SetUint64(7)
		long := make([]fr.Element, 3)
		long[0].SetUint64(7)
		rs := RescueHash(short)
		rl := RescueHash(long)
		require.NotNil(t, rs, "hash must not be nil")
		assert.False(t, rs.Equal(rl), "inputs of different lengths should produce different hashes")
	})
}

func TestRescueHashBigInt(t *testing.T) {
	t.Parallel()

	t.Run("nil input", func(t *testing.T) {
		t.Parallel()
		result := RescueHashBigInt(nil)
		require.NotNil(t, result, "hash of nil input must not be nil")
	})

	t.Run("nil and zero produce same hash", func(t *testing.T) {
		t.Parallel()
		rNil := RescueHashBigInt(nil)
		rZero := RescueHashBigInt(big.NewInt(0))
		require.NotNil(t, rNil, "hash must not be nil")
		assert.True(t, rNil.Equal(rZero), "nil and zero should produce the same hash")
	})

	t.Run("one differs from zero", func(t *testing.T) {
		t.Parallel()
		rZero := RescueHashBigInt(big.NewInt(0))
		rOne := RescueHashBigInt(big.NewInt(1))
		require.NotNil(t, rOne, "hash must not be nil")
		assert.False(t, rOne.Equal(rZero), "hash of 1 should differ from hash of 0")
	})

	t.Run("248-bit value fits in single chunk", func(t *testing.T) {
		t.Parallel()
		msg := new(big.Int).Lsh(big.NewInt(1), 247)
		result := RescueHashBigInt(msg)
		require.NotNil(t, result, "hash of 248-bit value must not be nil")
	})

	t.Run("249-bit value crosses chunk boundary", func(t *testing.T) {
		t.Parallel()
		single := new(big.Int).Lsh(big.NewInt(1), 247)
		double := new(big.Int).Lsh(big.NewInt(1), 248)
		rs := RescueHashBigInt(single)
		rd := RescueHashBigInt(double)
		require.NotNil(t, rd, "hash of 249-bit value must not be nil")
		assert.False(t, rs.Equal(rd), "values spanning different numbers of chunks should produce distinct hashes")
	})

	t.Run("multi-chunk large input", func(t *testing.T) {
		t.Parallel()
		msg := new(big.Int).Lsh(big.NewInt(1), 580)
		result := RescueHashBigInt(msg)
		require.NotNil(t, result, "hash of 580-bit value must not be nil")
	})

	t.Run("deterministic", func(t *testing.T) {
		t.Parallel()
		msg := new(big.Int).Lsh(big.NewInt(1), 200)
		r1 := RescueHashBigInt(msg)
		r2 := RescueHashBigInt(msg)
		require.NotNil(t, r1, "hash must not be nil")
		assert.True(t, r1.Equal(r2), "same input should always produce the same hash")
	})

	t.Run("distinct inputs produce distinct hashes", func(t *testing.T) {
		t.Parallel()
		msg := new(big.Int).Lsh(big.NewInt(1), 200)
		r1 := RescueHashBigInt(msg)
		r2 := RescueHashBigInt(new(big.Int).Add(msg, big.NewInt(1)))
		require.NotNil(t, r1, "hash must not be nil")
		assert.False(t, r1.Equal(r2), "distinct inputs should produce distinct hashes")
	})

	t.Run("various bit sizes produce non-nil results", func(t *testing.T) {
		t.Parallel()
		for _, bits := range []uint{1, 8, 31, 32, 128, 247, 248, 249, 296, 496, 580} {
			msg := new(big.Int).Lsh(big.NewInt(1), bits)
			result := RescueHashBigInt(msg)
			require.NotNilf(t, result, "hash of %d-bit value must not be nil", bits)
		}
	})
}
