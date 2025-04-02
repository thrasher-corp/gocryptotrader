package collateral

import (
	"fmt"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// Valid returns whether the collateral mode is valid
func (t Mode) Valid() bool {
	return t != UnsetMode && supportedCollateralModes&t == t
}

// UnmarshalJSON converts json into collateral mode
func (t *Mode) UnmarshalJSON(d []byte) error {
	var mode string
	err := json.Unmarshal(d, &mode)
	if err != nil {
		return err
	}
	*t, err = StringToMode(mode)
	return err
}

// String returns the string representation of the collateral mode in lowercase
// the absence of a lower func should hopefully highlight that String is lower
func (t Mode) String() string {
	switch t {
	case UnsetMode:
		return unsetCollateralStr
	case SingleMode:
		return singleCollateralStr
	case MultiMode:
		return multiCollateralStr
	case PortfolioMode:
		return portfolioCollateralStr
	case SpotFuturesMode:
		return spotFuturesCollateralStr
	case UnknownMode:
		return unknownCollateralStr
	}
	return ""
}

// Upper returns the upper case string representation of the collateral mode
func (t Mode) Upper() string {
	return strings.ToUpper(t.String())
}

// IsValidCollateralModeString checks to see if the supplied string is a valid collateral mode
func IsValidCollateralModeString(m string) bool {
	switch strings.ToLower(m) {
	case singleCollateralStr, multiCollateralStr, portfolioCollateralStr, unsetCollateralStr:
		return true
	}
	return false
}

// StringToMode converts a string to a collateral mode
// doesn't error, just returns unknown if the string is not recognised
func StringToMode(m string) (Mode, error) {
	switch strings.ToLower(m) {
	case singleCollateralStr:
		return SingleMode, nil
	case multiCollateralStr:
		return MultiMode, nil
	case portfolioCollateralStr:
		return PortfolioMode, nil
	case "":
		return UnsetMode, nil
	}
	return UnknownMode, fmt.Errorf("%w %v", ErrInvalidCollateralMode, m)
}
