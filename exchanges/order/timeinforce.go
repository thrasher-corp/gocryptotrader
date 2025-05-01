package order

import (
	"errors"
	"fmt"
	"strings"
)

// var error definitions
var (
	ErrInvalidTimeInForce     = errors.New("invalid time in force value provided")
	ErrUnsupportedTimeInForce = errors.New("unsupported time in force value")
)

// TimeInForce enforces a standard for time-in-force values across the code base.
type TimeInForce uint16

// TimeInForce types
const (
	UnknownTIF     TimeInForce = 0
	GoodTillCancel TimeInForce = 1 << iota
	GoodTillDay
	GoodTillTime
	GoodTillCrossing
	FillOrKill
	ImmediateOrCancel
	PostOnly

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
		result = ImmediateOrCancel
	case "GOODTILLCANCEL", "GOODTILCANCEL", "GOOD_TIL_CANCELLED", "GOOD_TILL_CANCELLED", "GOOD_TILL_CANCELED", GoodTillCancel.String():
		result = GoodTillCancel
	case "GOODTILLDAY", GoodTillDay.String(), "GOOD_TIL_DAY", "GOOD_TILL_DAY":
		result = GoodTillDay
	case "GOODTILLTIME", "GOOD_TIL_TIME", GoodTillTime.String():
		result = GoodTillTime
	case "GOODTILLCROSSING", "GOOD_TIL_CROSSING", "GOOD TIL CROSSING", GoodTillCrossing.String(), "GOOD_TILL_CROSSING":
		result = GoodTillCrossing
	case "FILLORKILL", "FILL_OR_KILL", FillOrKill.String():
		result = FillOrKill
	case PostOnly.String(), "POC", "POST_ONLY", "PENDINGORCANCEL":
		result = PostOnly
	case "POST_ONLY_GOOD_TIL_CANCELLED":
		result = GoodTillCancel | PostOnly
	}
	if result == UnknownTIF && timeInForce != "" {
		return UnknownTIF, fmt.Errorf("%w: tif=%s", ErrInvalidTimeInForce, timeInForce)
	}
	return result, nil
}

// IsValid returns whether or not the supplied time in force value is valid or
// not
func (t TimeInForce) IsValid() bool {
	// Neither ImmediateOrCancel nor FillOrKill can coexist with anything else
	// If either bit is set then it must be the only bit set
	isIOCorFOK := t&(ImmediateOrCancel|FillOrKill) != 0
	hasTwoBitsSet := t&(t-1) != 0
	if isIOCorFOK && hasTwoBitsSet {
		return false
	}
	return t == UnknownTIF || supportedTimeInForceFlag&t == t
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
	if t == UnknownTIF {
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
