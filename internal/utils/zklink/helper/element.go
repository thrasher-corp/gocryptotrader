package helper

import (
	"github.com/thrasher-corp/gocryptotrader/internal/utils/zklink/bn256/fr"
)

func NewElement() *fr.Element {
	return new(fr.Element)
}

func zero() *fr.Element {
	return new(fr.Element).SetZero()
}

func one() *fr.Element {
	return new(fr.Element).SetOne()
}
