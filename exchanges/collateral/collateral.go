package collateral

import (
	"encoding/json"
	"strings"
)

// Valid returns whether the margin type is valid
func (t Mode) Valid() bool {
	return t != UnsetMode && supportedCollateralModes&t == t
}

// UnmarshalJSON converts json into margin type
func (t *Mode) UnmarshalJSON(d []byte) error {
	var mode string
	err := json.Unmarshal(d, &mode)
	if err != nil {
		return err
	}
	*t = StringToMode(mode)
	return nil
}

// String returns the string representation of the margin type in lowercase
// the absence of a lower func should hopefully highlight that String is lower
func (t Mode) String() string {
	switch t {
	case UnsetMode:
		return unsetCollateralStr
	case SingleMode:
		return singleCollateralStr
	case MultiMode:
		return multiCollateralStr
	case GlobalMode:
		return globalCollateralStr
	case UnknownMode:
		return unknownCollateralStr
	}
	return ""
}

// Upper returns the upper case string representation of the margin type
func (t Mode) Upper() string {
	return strings.ToUpper(t.String())
}

// IsValidCollateralModeString checks to see if the supplied string is a valid collateral type
func IsValidCollateralModeString(m string) bool {
	switch strings.ToLower(m) {
	case singleCollateralStr, multiCollateralStr, globalCollateralStr, unsetCollateralStr:
		return true
	}
	return false
}

// StringToMode converts a string to a collateral type
// doesn't error, just returns unknown if the string is not recognised
func StringToMode(m string) Mode {
	switch strings.ToLower(m) {
	case singleCollateralStr:
		return SingleMode
	case multiCollateralStr:
		return MultiMode
	case globalCollateralStr:
		return GlobalMode
	case "":
		return UnsetMode
	}
	return UnknownMode
}
