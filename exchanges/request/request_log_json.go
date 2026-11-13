package request

import (
	"bytes"
	"encoding/json/jsontext" //nolint:depguard // Token offsets are unavailable through the backend-switching JSON wrapper.
	"errors"
	"io"
)

var errInvalidJSONLogBody = errors.New("invalid JSON log body")

// redactJSONBody replaces sensitive values without reserialising public data.
// Every token is validated, including those inside a value being redacted.
func redactJSONBody(payload []byte) ([]byte, error) {
	// Match the existing unmarshalling policy for duplicate names and invalid UTF-8.
	decoder := jsontext.NewDecoder(bytes.NewBuffer(payload), jsontext.AllowDuplicateNames(true), jsontext.AllowInvalidUTF8(true))
	var output []byte
	var copyFrom int
	for {
		token, err := decoder.ReadToken()
		if errors.Is(err, io.EOF) {
			_, rootValueCount := decoder.StackIndex(0)
			if rootValueCount != 1 {
				return nil, errInvalidJSONLogBody
			}
			if output == nil {
				return payload, nil
			}
			return append(output, payload[copyFrom:]...), nil
		}
		if err != nil {
			return nil, err
		}
		// The decoder accepts streams, but a body must contain exactly one JSON value.
		_, rootValueCount := decoder.StackIndex(0)
		if rootValueCount > 1 {
			return nil, errInvalidJSONLogBody
		}
		if token.Kind() == '0' {
			// Match the existing decoder's rejection of float64 overflow.
			if _, err := token.Float(); err != nil {
				return nil, err
			}
		}
		containerKind, elementCount := decoder.StackIndex(decoder.StackDepth())
		// Object names and values count separately: an odd count identifies a name.
		isObjectName := containerKind == '{' && elementCount%2 == 1 && token.Kind() == '"'
		if !isObjectName || !isSensitiveLogKey(token.String()) {
			continue
		}

		// Keep the separator and whitespace outside the replacement span.
		valueStart := int(decoder.InputOffset())
		remaining := payload[valueStart:]
		valueStart += len(remaining) - len(bytes.TrimLeft(remaining, ": \t\r\n"))
		if err := skipJSONLogValue(decoder); err != nil {
			return nil, err
		}
		if output == nil {
			output = make([]byte, 0, len(payload)+len(`"[REDACTED]"`))
		}
		output = append(output, payload[copyFrom:valueStart]...)
		output = append(output, `"[REDACTED]"`...)
		copyFrom = int(decoder.InputOffset())
	}
}

// skipJSONLogValue consumes one sensitive value, including nested containers.
// Unlike Decoder.SkipValue, it also rejects numbers that overflow float64,
// preserving the previous unmarshalling behaviour even for redacted values.
func skipJSONLogValue(decoder *jsontext.Decoder) error {
	parentDepth := decoder.StackDepth()
	for {
		token, err := decoder.ReadToken()
		if err != nil {
			return err
		}
		if token.Kind() == '0' {
			if _, err := token.Float(); err != nil {
				return err
			}
		}
		// A scalar finishes after one token; a container finishes at its closing token.
		if decoder.StackDepth() == parentDepth {
			return nil
		}
	}
}
