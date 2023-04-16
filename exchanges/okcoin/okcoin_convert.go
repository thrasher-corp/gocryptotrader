package okcoin

import (
	"encoding/json"
	"strconv"
)

type okcoinMilliSec int64

// UnmarshalJSON deserializes timestamp information to time.Time
func (o *okcoinMilliSec) UnmarshalJSON(data []byte) error {
	var timeMilliSecond string
	err := json.Unmarshal(data, &timeMilliSecond)
	if err != nil {
		return err
	}
	timeInteger, err := strconv.ParseInt(timeMilliSecond, 10, 64)
	if err != nil {
		return err
	}
	*o = okcoinMilliSec(timeInteger)
	return nil
}
