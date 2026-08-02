package mexc

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"fmt"
	"io"

	"github.com/thrasher-corp/gocryptotrader/common"
)

// decodeBinaryFrame decodes a MEXC binary websocket frame.
//
// MEXC pushes protobuf payloads uncompressed, which the shared default does not expect: it
// treats every non-GZIP binary frame as a raw DEFLATE stream and so corrupts them. Rather
// than changing that default for every exchange, the venue-specific handling lives here and
// is attached through ConnectionSetup.BinaryMessageDecoder — exchanges that do not opt in
// keep the upstream behaviour verbatim.
//
// Frames are classified by what they actually look like:
//   - GZIP magic (1f 8b)                    -> gzip
//   - low nibble of the first byte is 8     -> zlib, falling back to a raw DEFLATE stream
//     (that is the zlib CM field; the fallback covers headerless streams)
//   - anything else                         -> returned as is (the uncompressed protobuf case)
func decodeBinaryFrame(resp []byte) ([]byte, error) {
	if len(resp) == 0 {
		return nil, fmt.Errorf("%w: empty binary response", common.ErrNoResponse)
	}
	var reader io.ReadCloser
	switch {
	case len(resp) >= 2 && resp[0] == 0x1f && resp[1] == 0x8b:
		gr, err := gzip.NewReader(bytes.NewReader(resp))
		if err != nil {
			return nil, err
		}
		reader = gr
	case len(resp) > 2 && resp[0]&0x0F == 8:
		zr, err := zlib.NewReader(bytes.NewReader(resp))
		if err != nil {
			reader = io.NopCloser(flate.NewReader(bytes.NewReader(resp)))
		} else {
			reader = zr
		}
	default:
		return resp, nil
	}
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return decoded, reader.Close()
}
