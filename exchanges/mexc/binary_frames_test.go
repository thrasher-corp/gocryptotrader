package mexc

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
)

const binaryFramePayload = "the quick brown fox jumps over the lazy dog"

func TestDecodeBinaryFrameEmpty(t *testing.T) {
	t.Parallel()
	_, err := decodeBinaryFrame(nil)
	assert.ErrorIs(t, err, common.ErrNoResponse, "an empty frame should be reported")
}

// TestDecodeBinaryFrameUncompressed is the case the shared default gets wrong: MEXC pushes protobuf
// with no compression, and treating it as a raw DEFLATE stream corrupts it.
func TestDecodeBinaryFrameUncompressed(t *testing.T) {
	t.Parallel()
	// 0x0a is the protobuf tag for field 1 (the channel), which is how every push frame starts.
	raw := append([]byte{0x0a, 0x04}, []byte("spot")...)
	got, err := decodeBinaryFrame(raw)
	require.NoError(t, err, "an uncompressed frame must not error")
	assert.Equal(t, raw, got, "an uncompressed frame should be returned untouched")
}

func TestDecodeBinaryFrameGzip(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, err := w.Write([]byte(binaryFramePayload))
	require.NoError(t, err, "writing the gzip payload must not error")
	require.NoError(t, w.Close(), "closing the gzip writer must not error")

	got, err := decodeBinaryFrame(buf.Bytes())
	require.NoError(t, err, "a gzip frame must not error")
	assert.Equal(t, binaryFramePayload, string(got), "the gzip frame should be inflated")
}

func TestDecodeBinaryFrameGzipTruncated(t *testing.T) {
	t.Parallel()
	_, err := decodeBinaryFrame([]byte{0x1f, 0x8b, 0x08, 0x00})
	assert.Error(t, err, "a truncated gzip header should be reported")
}

func TestDecodeBinaryFrameZlib(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	_, err := w.Write([]byte(binaryFramePayload))
	require.NoError(t, err, "writing the zlib payload must not error")
	require.NoError(t, w.Close(), "closing the zlib writer must not error")

	raw := buf.Bytes()
	require.Equal(t, byte(8), raw[0]&0x0F, "a zlib stream must carry the DEFLATE compression method in the low nibble")

	got, err := decodeBinaryFrame(raw)
	require.NoError(t, err, "a zlib frame must not error")
	assert.Equal(t, binaryFramePayload, string(got), "the zlib frame should be inflated")
}

// TestDecodeBinaryFrameHeaderlessDeflate covers the fallback: the low nibble says DEFLATE but the
// zlib header check fails, so the frame is retried as a headerless stream.
func TestDecodeBinaryFrameHeaderlessDeflate(t *testing.T) {
	t.Parallel()
	// 0x08 0x00 is not a valid zlib header: (0x08<<8|0x00) is not a multiple of 31.
	_, err := zlib.NewReader(bytes.NewReader([]byte{0x08, 0x00, 0x01, 0x02}))
	require.Error(t, err, "the crafted frame must not be accepted as zlib, otherwise the fallback is not exercised")

	_, err = decodeBinaryFrame([]byte{0x08, 0x00, 0x01, 0x02})
	assert.Error(t, err, "a frame that is neither zlib nor valid DEFLATE should be reported")
}
