package order

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidTimeInForce = errors.New("invalid time in force value provided")
)

// TimeInForce enforces a standard for time-in-force values across the code base.
type TimeInForce uint16

// TimeInForce types
const (
	UnsetTIF       TimeInForce = 0
	GoodTillCancel TimeInForce = 1 << iota
	GoodTillDay
	GoodTillTime
	GoodTillCrossing
	FillOrKill
	ImmediateOrCancel
	PostOnly
	UnknownTIF

	supportedTimeInForceFlag = GoodTillCancel | GoodTillDay | GoodTillTime | GoodTillCrossing | FillOrKill | ImmediateOrCancel | PostOnly
)

// Is checks to see if the enum contains the flag
func (t TimeInForce) Is(in TimeInForce) bool {
	return in != 0 && t&in == in
}

// StringToTimeInForce converts time in force string value to TimeInForce instance.
func StringToTimeInForce(timeInForce string) (TimeInForce, error) {
	var result TimeInForce
	timeInForce = strings.ToUpper(timeInForce)
	switch timeInForce {
	case "IMMEDIATEORCANCEL", "IMMEDIATE_OR_CANCEL", ImmediateOrCancel.String():
		result |= ImmediateOrCancel
	}
	switch timeInForce {
	case "GOODTILLCANCEL", "GOODTILCANCEL", "GOOD_TIL_CANCELLED", "GOOD_TILL_CANCELLED", "GOOD_TILL_CANCELED", GoodTillCancel.String(), "POST_ONLY_GOOD_TIL_CANCELLED":
		result |= GoodTillCancel
	}
	switch timeInForce {
	case "GOODTILLDAY", GoodTillDay.String(), "GOOD_TIL_DAY", "GOOD_TILL_DAY":
		result |= GoodTillDay
	}
	switch timeInForce {
	case "GOODTILLTIME", "GOOD_TIL_TIME", GoodTillTime.String():
		result |= GoodTillTime
	}
	switch timeInForce {
	case "GOODTILLCROSSING", "GOOD_TIL_CROSSING", "GOOD TIL CROSSING", GoodTillCrossing.String(), "GOOD_TILL_CROSSING":
		result |= GoodTillCrossing
	}
	switch timeInForce {
	case "FILLORKILL", "FILL_OR_KILL", FillOrKill.String():
		result |= FillOrKill
	}
	switch timeInForce {
	case PostOnly.String(), "POC", "POST_ONLY", "PENDINGORCANCEL", "POST_ONLY_GOOD_TIL_CANCELLED":
		result |= PostOnly
	}
	if result == UnsetTIF && timeInForce != "" {
		return UnknownTIF, fmt.Errorf("%w: tif=%s", ErrInvalidTimeInForce, timeInForce)
	}
	return result, nil
}

// IsValid returns whether or not the supplied time in force value is valid or
// not
func (t TimeInForce) IsValid() bool {
	return t != UnsetTIF && supportedTimeInForceFlag&t == t
}

// String implements the stringer interface.
func (t TimeInForce) String() string {
	var tifStrings []string
	if t.Is(ImmediateOrCancel) {
		tifStrings = append(tifStrings, "IOC")
	}
	if t.Is(GoodTillCancel) {
		tifStrings = append(tifStrings, "GTC")
	}
	if t.Is(GoodTillDay) {
		tifStrings = append(tifStrings, "GTD")
	}
	if t.Is(GoodTillTime) {
		tifStrings = append(tifStrings, "GTT")
	}
	if t.Is(GoodTillCrossing) {
		tifStrings = append(tifStrings, "GTX")
	}
	if t.Is(FillOrKill) {
		tifStrings = append(tifStrings, "FOK")
	}
	if t.Is(PostOnly) {
		tifStrings = append(tifStrings, "POSTONLY")
	}
	if t == UnsetTIF {
		return ""
	}
	if len(tifStrings) == 0 {
		return "UNKNOWN"
	}
	return strings.Join(tifStrings, ",")
}

// UnmarshalJSON deserializes a string data into TimeInForce instance.
func (t *TimeInForce) UnmarshalJSON(data []byte) error {
	tifStrings := strings.Split(strings.Trim(string(data), `"`), ",")
	for _, val := range tifStrings {
		tif, err := StringToTimeInForce(val)
		if err != nil {
			return err
		}
		*t |= tif
	}
	return nil
}

// MarshalJSON returns the JSON-encoded order time-in-force value
func (t TimeInForce) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.String() + `"`), nil
}
