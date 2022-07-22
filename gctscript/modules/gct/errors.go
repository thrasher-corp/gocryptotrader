package gct

import (
	"errors"
	"fmt"

	objects "github.com/d5/tengo/v2"
)

const standardFormatting = "%s"

var (
	errFormatStringIsEmpty = errors.New("format string is empty")
	errNoArguments         = errors.New("no arguments for error response")
)

// errorResponsef is a helper function to apply error details to a return object
// for better script side error handling
func errorResponsef(format string, a ...interface{}) (objects.Object, error) {
	if format == "" {
		return nil, fmt.Errorf("cannot generate tengo error object %w", errFormatStringIsEmpty)
	}

	if len(a) == 0 {
		return nil, fmt.Errorf("cannot generate tengo error object %w", errNoArguments)
	}

	return &objects.Error{
		Value: &objects.String{Value: fmt.Sprintf(format, a...)},
	}, nil
}
