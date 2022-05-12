package common

import (
	"fmt"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/log"
)

// DataTypeToInt converts the config string value into an int
func DataTypeToInt(dataType string) (int64, error) {
	switch dataType {
	case CandleStr:
		return DataCandle, nil
	case TradeStr:
		return DataTrade, nil
	default:
		return 0, fmt.Errorf("unrecognised dataType '%v'", dataType)
	}
}

// FitStringToLimit ensures a string is of the length of the limit
// either by truncating the string with ellipses or padding with the spacer
func FitStringToLimit(str, spacer string, limit int, upper bool) string {
	if limit < 0 {
		return str
	}
	if limit == 0 {
		return ""
	}
	limResp := limit - len(str)
	if upper {
		str = strings.ToUpper(str)
	}
	if limResp < 0 {
		if limit-3 > 0 {
			return str[0:limit-3] + "..."
		}
		return str[0:limit]
	}
	spacerLen := len(spacer)
	for i := 0; i < limResp; i++ {
		str += spacer
		for j := 0; j < spacerLen; j++ {
			if j > 0 {
				// prevent clever people from going beyond
				// the limit by having a spacer longer than 1
				i++
			}
		}
	}

	return str[0:limit]
}

// RegisterBacktesterSubLoggers sets up all custom Backtester sub-loggers
func RegisterBacktesterSubLoggers() error {
	var err error
	Backtester, err = log.NewSubLogger("Backtester")
	if err != nil {
		return err
	}
	Setup, err = log.NewSubLogger("Setup")
	if err != nil {
		return err
	}
	Strategy, err = log.NewSubLogger("Strategy")
	if err != nil {
		return err
	}
	Report, err = log.NewSubLogger("Report")
	if err != nil {
		return err
	}
	Statistics, err = log.NewSubLogger("Statistics")
	if err != nil {
		return err
	}
	CurrencyStatistics, err = log.NewSubLogger("CurrencyStatistics")
	if err != nil {
		return err
	}
	FundingStatistics, err = log.NewSubLogger("FundingStatistics")
	if err != nil {
		return err
	}
	Backtester, err = log.NewSubLogger("Sizing")
	if err != nil {
		return err
	}
	Holdings, err = log.NewSubLogger("Holdings")
	if err != nil {
		return err
	}
	Data, err = log.NewSubLogger("Data")
	if err != nil {
		return err
	}

	// Set to existing registered sub-loggers
	Config = log.ConfigMgr
	Portfolio = log.PortfolioMgr
	Exchange = log.ExchangeSys
	Fill = log.Fill

	return nil
}

// PurgeColours removes colour information
func PurgeColours() {
	ColourGreen = ""
	ColourWhite = ""
	ColourGrey = ""
	ColourDefault = ""
	ColourH1 = ""
	ColourH2 = ""
	ColourH3 = ""
	ColourH4 = ""
	ColourSuccess = ""
	ColourInfo = ""
	ColourDebug = ""
	ColourWarn = ""
	ColourDarkGrey = ""
	ColourError = ""
}

// Logo returns the logo
func Logo() string {
	sb := strings.Builder{}
	sb.WriteString("                                                                                \n")
	sb.WriteString("                               " + ColourWhite + "@@@@@@@@@@@@@@@@@                                \n")
	sb.WriteString("                            " + ColourWhite + "@@@@@@@@@@@@@@@@@@@@@@@    " + ColourGrey + ",,,,,," + ColourWhite + "                   \n")
	sb.WriteString("                           " + ColourWhite + "@@@@@@@@" + ColourGrey + ",,,,,    " + ColourWhite + "@@@@@@@@@" + ColourGrey + ",,,,,,,," + ColourWhite + "                   \n")
	sb.WriteString("                         " + ColourWhite + "@@@@@@@@" + ColourGrey + ",,,,,,,       " + ColourWhite + "@@@@@@@" + ColourGrey + ",,,,,,," + ColourWhite + "                   \n")
	sb.WriteString("                         " + ColourWhite + "@@@@@@" + ColourGrey + "(,,,,,,,,      " + ColourGrey + ",," + ColourWhite + "@@@@@@@" + ColourGrey + ",,,,,," + ColourWhite + "                   \n")
	sb.WriteString("                      " + ColourGrey + ",," + ColourWhite + "@@@@@@" + ColourGrey + ",,,,,,,,,   #,,,,,,,,,,,,,,,,,," + ColourWhite + "                   \n")
	sb.WriteString("                   " + ColourGrey + ",,,,*" + ColourWhite + "@@@@@@" + ColourGrey + ",,,,,,,,,,,,,,,,,,,,,,,,,," + ColourGreen + "%%%%%%%" + ColourWhite + "                \n")
	sb.WriteString("                " + ColourGrey + ",,,,,,,*" + ColourWhite + "@@@@@@" + ColourGrey + ",,,,,,,,,,,,,," + ColourGreen + "%%%%%" + ColourGrey + " ,,,,,," + ColourGrey + "%" + ColourGreen + "%%%%%%" + ColourWhite + "                 \n")
	sb.WriteString("               " + ColourGrey + ",,,,,,,,*" + ColourWhite + "@@@@@@" + ColourGrey + ",,,,,,,,,,," + ColourGreen + "%%%%%%%%%%%%%%%%%%" + ColourGrey + "#" + ColourGreen + "%%" + ColourGrey + "                  \n")
	sb.WriteString("                 " + ColourGrey + ",,,,,,*" + ColourWhite + "@@@@@@" + ColourGrey + ",,,,,,,,," + ColourGreen + "%%%" + ColourGrey + " ,,,,," + ColourGreen + "%%%%%%%%" + ColourGrey + ",,,,,                   \n")
	sb.WriteString("                    " + ColourGrey + ",,,*" + ColourWhite + "@@@@@@" + ColourGrey + ",,,,,," + ColourGreen + "%%" + ColourGrey + ",,  ,,,,,,," + ColourWhite + "@" + ColourGreen + "*%%," + ColourWhite + "@" + ColourGrey + ",,,,,,                   \n")
	sb.WriteString("                       " + ColourGrey + "*" + ColourWhite + "@@@@@@" + ColourGrey + ",,,,,,,,,     " + ColourGrey + ",,,,," + ColourWhite + "@@@@@@" + ColourGrey + ",,,,,," + ColourWhite + "                    \n")
	sb.WriteString("                         " + ColourWhite + "@@@@@@" + ColourGrey + ",,,,,,,,,        " + ColourWhite + "@@@@@@@" + ColourGrey + ",,,,,," + ColourWhite + "                   \n")
	sb.WriteString("                         " + ColourWhite + "@@@@@@@@" + ColourGrey + ",,,,,,,       " + ColourWhite + "@@@@@@@" + ColourGrey + ",,,,,,," + ColourWhite + "                   \n")
	sb.WriteString("                           " + ColourWhite + "@@@@@@@@@" + ColourGrey + ",,,,    " + ColourWhite + "@@@@@@@@@" + ColourGrey + "#,,,,,,," + ColourWhite + "                   \n")
	sb.WriteString("                            " + ColourWhite + "@@@@@@@@@@@@@@@@@@@@@@@     " + ColourGrey + "*,,,," + ColourWhite + "                   \n")
	sb.WriteString("                                " + ColourWhite + "@@@@@@@@@@@@@@@@" + ColourDefault + "                                \n")
	sb.WriteString(ASCIILogo)
	return sb.String()
}
