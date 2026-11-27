package nonce

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAndIncrement(t *testing.T) {
	var nonce Nonce
	n1 := nonce.GetAndIncrement(Unix)
	assert.NotZero(t, n1)
	n2 := nonce.GetAndIncrement(Unix)
	assert.NotZero(t, n2)
	assert.NotEqual(t, n1, n2)

	var nonce2 Nonce
	n3 := nonce2.GetAndIncrement(UnixNano)
	assert.NotZero(t, n3)
	n4 := nonce2.GetAndIncrement(UnixNano)
	assert.NotZero(t, n4)
	assert.NotEqual(t, n3, n4)

	assert.NotEqual(t, n1, n3)
	assert.NotEqual(t, n2, n4)
}

func TestString(t *testing.T) {
	var nonce Nonce
	nonce.n = 12312313131
	got := nonce.GetAndIncrement(Unix)
	assert.Equal(t, "12312313131", got.String())

	got = nonce.GetAndIncrement(Unix)
	assert.Equal(t, "12312313132", got.String())
}
