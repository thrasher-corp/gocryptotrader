package order

import (
	"errors"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
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
	StopOrReduce

	supportedTimeInForceFlag = GoodTillCancel | GoodTillDay | GoodTillTime | GoodTillCrossing | FillOrKill | ImmediateOrCancel | PostOnly | StopOrReduce
)

// time-in-force string representations
const (
	gtcStr      = "GTC"
	gtdStr      = "GTD"
	gttStr      = "GTT"
	gtxStr      = "GTX"
	fokStr      = "FOK"
	iocStr      = "IOC"
	postonlyStr = "POSTONLY"
	sorStr      = "SOR"

	// unknownTIFStr is the string of a TimeInForce with no supported flag
	unknownTIFStr = "UNKNOWN"
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
	case "IMMEDIATEORCANCEL", "IMMEDIATE_OR_CANCEL", iocStr:
		result = ImmediateOrCancel
	case "GOODTILLCANCEL", "GOODTILCANCEL", "GOOD_TIL_CANCELLED", "GOOD_TILL_CANCELLED", "GOOD_TILL_CANCELED", gtcStr:
		result = GoodTillCancel
	case "GOODTILLDAY", "GOOD_TIL_DAY", "GOOD_TILL_DAY", gtdStr:
		result = GoodTillDay
	case "GOODTILLTIME", "GOOD_TIL_TIME", gttStr:
		result = GoodTillTime
	case "GOODTILLCROSSING", "GOOD_TIL_CROSSING", "GOOD TIL CROSSING", "GOOD_TILL_CROSSING", gtxStr:
		result = GoodTillCrossing
	case "FILLORKILL", "FILL_OR_KILL", fokStr:
		result = FillOrKill
	case "POC", "POST_ONLY", "PENDINGORCANCEL", postonlyStr:
		result = PostOnly
	case "STOPORREDUCE", "STOP_OR_REDUCE", sorStr:
		result = StopOrReduce
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
	if t == UnknownTIF {
		return ""
	}
	var tifStrings []string
	if t.Is(ImmediateOrCancel) {
		tifStrings = append(tifStrings, iocStr)
	}
	if t.Is(GoodTillCancel) {
		tifStrings = append(tifStrings, gtcStr)
	}
	if t.Is(GoodTillDay) {
		tifStrings = append(tifStrings, gtdStr)
	}
	if t.Is(GoodTillTime) {
		tifStrings = append(tifStrings, gttStr)
	}
	if t.Is(GoodTillCrossing) {
		tifStrings = append(tifStrings, gtxStr)
	}
	if t.Is(FillOrKill) {
		tifStrings = append(tifStrings, fokStr)
	}
	if t.Is(PostOnly) {
		tifStrings = append(tifStrings, postonlyStr)
	}
	if t.Is(StopOrReduce) {
		tifStrings = append(tifStrings, sorStr)
	}
	if len(tifStrings) == 0 {
		return unknownTIFStr
	}
	return strings.Join(tifStrings, ",")
}

// Lower returns a lower case string representation of time-in-force
func (t TimeInForce) Lower() string {
	return strings.ToLower(t.String())
}

// UnmarshalJSON deserialises a string data into TimeInForce instance.
func (t *TimeInForce) UnmarshalJSON(data []byte) error {
	for val := range strings.SplitSeq(strings.Trim(string(data), `"`), ",") {
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

// MarshalBSONValue implements bson.ValueMarshaler, storing the TimeInForce as its string
func (t TimeInForce) MarshalBSONValue() (typ byte, data []byte, err error) {
	bt, data, err := bson.MarshalValue(t.String())
	return byte(bt), data, err
}

// UnmarshalBSONValue implements bson.ValueUnmarshaler. A TimeInForce with no supported flag is stored as
// unknownTIFStr, which StringToTimeInForce rejects as exchange input, and reads back as UnknownTIF.
func (t *TimeInForce) UnmarshalBSONValue(typ byte, data []byte) error {
	if bson.Type(typ) != bson.TypeString {
		return fmt.Errorf("%w: expected a BSON string, got %s", ErrInvalidTimeInForce, bson.Type(typ))
	}
	var str string
	if err := bson.UnmarshalValue(bson.TypeString, data, &str); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidTimeInForce, err)
	}
	if str == unknownTIFStr {
		*t = UnknownTIF
		return nil
	}
	var tif TimeInForce
	for flag := range strings.SplitSeq(str, ",") {
		f, err := StringToTimeInForce(flag)
		if err != nil {
			return err
		}
		tif |= f
	}
	*t = tif
	return nil
}
