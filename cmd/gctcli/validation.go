package main

import (
	"errors"
	"strings"
)

var (
	errInvalidPair = errors.New("invalid currency pair supplied")
)

func validPair(pair string) bool {
	return strings.Contains(pair, pairDelimiter)
}
