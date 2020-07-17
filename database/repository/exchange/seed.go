package exchange

import (
	"errors"
)

func Seed(in interface{}) error {
	v, ok := in.([]Details)
	if !ok {
		return errors.New("unexpected data received")
	}

	var allExchanges []Details
	for x := range v {
		allExchanges = append(allExchanges, Details{
			Name: v[x].Name,
		})
	}
	return InsertMany(allExchanges)
}
