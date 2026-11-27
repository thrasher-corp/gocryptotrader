package request

import (
	"bytes"
	"encoding/json/jsontext" //nolint:depguard // Token offsets are unavailable through the backend-switching JSON wrapper.
	"errors"
	"io"
)

var errInvalidJSONLogBody = errors.New("invalid JSON log body")

// maxJSONDocumentLogDepth limits how deeply JSON documents held in string values are redacted; a document nested
// deeper is redacted whole.
const maxJSONDocumentLogDepth = 8

// redactJSONBody replaces sensitive values without reserialising public data.
// Every token is validated, including those inside a value being redacted.
func redactJSONBody(payload []byte) ([]byte, error) {
	redacted, _, err := redactJSONDocument(payload, maxJSONDocumentLogDepth)
	return redacted, err
}

// redactJSONDocument redacts a JSON document and reports whether it changed. A string value holding a JSON document,
// such as a request a venue echoes in an error, has that document redacted too, up to depth documents deep.
func redactJSONDocument(payload []byte, depth int) (redacted []byte, changed bool, err error) {
	// Match the existing unmarshalling policy for duplicate names and invalid UTF-8.
	decoder := jsontext.NewDecoder(bytes.NewBuffer(payload), jsontext.AllowDuplicateNames(true), jsontext.AllowInvalidUTF8(true))
	var output []byte
	var copyFrom int
	for {
		tokenStart := int(decoder.InputOffset())
		token, err := decoder.ReadToken()
		if errors.Is(err, io.EOF) {
			_, rootValueCount := decoder.StackIndex(0)
			if rootValueCount != 1 {
				return nil, false, errInvalidJSONLogBody
			}
			if output == nil {
				return payload, false, nil
			}
			return append(output, payload[copyFrom:]...), true, nil
		}
		if err != nil {
			return nil, false, err
		}
		// The decoder accepts streams, but a body must contain exactly one JSON value.
		_, rootValueCount := decoder.StackIndex(0)
		if rootValueCount > 1 {
			return nil, false, errInvalidJSONLogBody
		}
		if token.Kind() == '0' {
			// Match the existing decoder's rejection of float64 overflow.
			if _, err := token.Float(); err != nil {
				return nil, false, err
			}
		}
		containerKind, elementCount := decoder.StackIndex(decoder.StackDepth())
		// Object names and values count separately: an odd count identifies a name.
		isObjectName := containerKind == '{' && elementCount%2 == 1 && token.Kind() == '"'
		if token.Kind() == '"' && !isObjectName {
			tokenEnd := int(decoder.InputOffset())
			// Keep the separator and whitespace outside the replacement span.
			value := bytes.TrimLeft(payload[tokenStart:tokenEnd], ",: \t\r\n")
			if replacement, changed := redactJSONStringValue(value, depth); changed {
				if output == nil {
					output = make([]byte, 0, len(payload))
				}
				output = append(output, payload[copyFrom:tokenEnd-len(value)]...)
				output = append(output, replacement...)
				copyFrom = tokenEnd
			}
			continue
		}
		if !isObjectName || !isSensitiveLogKey(token.String()) {
			continue
		}

		// Keep the separator and whitespace outside the replacement span.
		valueStart := int(decoder.InputOffset())
		remaining := payload[valueStart:]
		valueStart += len(remaining) - len(bytes.TrimLeft(remaining, ": \t\r\n"))
		if err := skipJSONLogValue(decoder); err != nil {
			return nil, false, err
		}
		if output == nil {
			output = make([]byte, 0, len(payload)+len(`"[REDACTED]"`))
		}
		output = append(output, payload[copyFrom:valueStart]...)
		output = append(output, `"[REDACTED]"`...)
		copyFrom = int(decoder.InputOffset())
	}
}

// redactJSONStringValue returns a JSON string value holding a JSON document with the document's sensitive values
// redacted, and reports whether it changed. A document that cannot be read, or that is nested deeper than depth allows,
// is redacted whole, as it may still hold credentials.
func redactJSONStringValue(value []byte, depth int) ([]byte, bool) {
	text := bytes.TrimLeft(value[1:], " ")
	if text[0] != '{' && text[0] != '[' && text[0] != '\\' {
		return nil, false
	}
	// Invalid UTF-8 unquotes to the replacement character, as the document's reader would see it.
	document, _ := jsontext.AppendUnquote(nil, value)
	if trimmed := bytes.TrimLeft(document, " \t\r\n"); len(trimmed) == 0 || (trimmed[0] != '{' && trimmed[0] != '[') {
		return nil, false
	}
	if depth == 0 {
		return []byte(`"[REDACTED]"`), true
	}
	redacted, changed, err := redactJSONDocument(document, depth-1)
	if err != nil {
		return []byte(`"[REDACTED]"`), true
	}
	if !changed {
		return nil, false
	}
	// Quoting fails only on invalid UTF-8, which unquoting has already replaced
	quoted, _ := jsontext.AppendQuote(nil, redacted)
	return quoted, true
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
