package common

import (
	"fmt"
	"regexp"
	"strings"

	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// CanTransact checks whether an order side is valid
// to the backtester's standards
func CanTransact(side gctorder.Side) bool {
	return side.IsLong() || side.IsShort() || side == gctorder.ClosePosition
}

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

// GenerateFileName will convert a proposed filename into something that is more
// OS friendly
func GenerateFileName(fileName, extension string) (string, error) {
	if fileName == "" {
		return "", fmt.Errorf("%w missing filename", errCannotGenerateFileName)
	}
	if extension == "" {
		return "", fmt.Errorf("%w missing filename extension", errCannotGenerateFileName)
	}

	reg := regexp.MustCompile(`[\w-]`)
	parsedFileName := reg.FindAllString(fileName, -1)
	parsedExtension := reg.FindAllString(extension, -1)
	fileName = strings.Join(parsedFileName, "") + "." + strings.Join(parsedExtension, "")

	return strings.ToLower(fileName), nil
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
	for i := range limResp {
		str += spacer
		for j := range spacerLen {
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
	LiveStrategy, err = log.NewSubLogger("LiveStrategy")
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
	Holdings, err = log.NewSubLogger("Holdings")
	if err != nil {
		return err
	}
	Data, err = log.NewSubLogger("Data")
	if err != nil {
		return err
	}
	FundManager, err = log.NewSubLogger("FundManager")
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
	CMDColours.Green = ""
	CMDColours.White = ""
	CMDColours.Grey = ""
	CMDColours.Default = ""
	CMDColours.H1 = ""
	CMDColours.H2 = ""
	CMDColours.H3 = ""
	CMDColours.H4 = ""
	CMDColours.Success = ""
	CMDColours.Info = ""
	CMDColours.Debug = ""
	CMDColours.Warn = ""
	CMDColours.DarkGrey = ""
	CMDColours.Error = ""
}

// SetColours sets cmd output colours at startup. Doing it at any other point
// risks races and this really isn't worth adding a mutex for
func SetColours(colours *Colours) {
	if colours.Default != "" && colours.Default != CMDColours.Default {
		CMDColours.Default = colours.Default
	}
	if colours.Green != "" && colours.Green != CMDColours.Green {
		CMDColours.Green = colours.Green
	}
	if colours.Error != "" && colours.Error != CMDColours.Error {
		CMDColours.Error = colours.Error
	}
	if colours.White != "" && colours.White != CMDColours.White {
		CMDColours.White = colours.White
	}
	if colours.Grey != "" && colours.Grey != CMDColours.Grey {
		CMDColours.Grey = colours.Grey
	}
	if colours.H1 != "" && colours.H1 != CMDColours.H1 {
		CMDColours.H1 = colours.H1
	}
	if colours.H2 != "" && colours.H2 != CMDColours.H2 {
		CMDColours.H2 = colours.H2
	}
	if colours.H3 != "" && colours.H3 != CMDColours.H3 {
		CMDColours.H3 = colours.H3
	}
	if colours.H4 != "" && colours.H4 != CMDColours.H4 {
		CMDColours.H4 = colours.H4
	}
	if colours.Success != "" && colours.Error != CMDColours.Success {
		CMDColours.Success = colours.Success
	}
	if colours.Info != "" && colours.Info != CMDColours.Info {
		CMDColours.Info = colours.Info
	}
	if colours.Debug != "" && colours.Debug != CMDColours.Debug {
		CMDColours.Debug = colours.Debug
	}
	if colours.Warn != "" && colours.Warn != CMDColours.Warn {
		CMDColours.Warn = colours.Warn
	}
	if colours.DarkGrey != "" && colours.DarkGrey != CMDColours.DarkGrey {
		CMDColours.DarkGrey = colours.DarkGrey
	}
}

// Logo returns the logo
func Logo() string {
	sb := strings.Builder{}
	sb.WriteString("                                                                                \n")
	sb.WriteString("                               " + CMDColours.White + "@@@@@@@@@@@@@@@@@                                \n")
	sb.WriteString("                            " + CMDColours.White + "@@@@@@@@@@@@@@@@@@@@@@@    " + CMDColours.Grey + ",,,,,," + CMDColours.White + "                   \n")
	sb.WriteString("                           " + CMDColours.White + "@@@@@@@@" + CMDColours.Grey + ",,,,,    " + CMDColours.White + "@@@@@@@@@" + CMDColours.Grey + ",,,,,,,," + CMDColours.White + "                   \n")
	sb.WriteString("                         " + CMDColours.White + "@@@@@@@@" + CMDColours.Grey + ",,,,,,,       " + CMDColours.White + "@@@@@@@" + CMDColours.Grey + ",,,,,,," + CMDColours.White + "                   \n")
	sb.WriteString("                         " + CMDColours.White + "@@@@@@" + CMDColours.Grey + "(,,,,,,,,      " + CMDColours.Grey + ",," + CMDColours.White + "@@@@@@@" + CMDColours.Grey + ",,,,,," + CMDColours.White + "                   \n")
	sb.WriteString("                      " + CMDColours.Grey + ",," + CMDColours.White + "@@@@@@" + CMDColours.Grey + ",,,,,,,,,   #,,,,,,,,,,,,,,,,,," + CMDColours.White + "                   \n")
	sb.WriteString("                   " + CMDColours.Grey + ",,,,*" + CMDColours.White + "@@@@@@" + CMDColours.Grey + ",,,,,,,,,,,,,,,,,,,,,,,,,," + CMDColours.Green + "%%%%%%%" + CMDColours.White + "                \n")
	sb.WriteString("                " + CMDColours.Grey + ",,,,,,,*" + CMDColours.White + "@@@@@@" + CMDColours.Grey + ",,,,,,,,,,,,,," + CMDColours.Green + "%%%%%" + CMDColours.Grey + " ,,,,,," + CMDColours.Grey + "%" + CMDColours.Green + "%%%%%%" + CMDColours.White + "                 \n")
	sb.WriteString("               " + CMDColours.Grey + ",,,,,,,,*" + CMDColours.White + "@@@@@@" + CMDColours.Grey + ",,,,,,,,,,," + CMDColours.Green + "%%%%%%%%%%%%%%%%%%" + CMDColours.Grey + "#" + CMDColours.Green + "%%" + CMDColours.Grey + "                  \n")
	sb.WriteString("                 " + CMDColours.Grey + ",,,,,,*" + CMDColours.White + "@@@@@@" + CMDColours.Grey + ",,,,,,,,," + CMDColours.Green + "%%%" + CMDColours.Grey + " ,,,,," + CMDColours.Green + "%%%%%%%%" + CMDColours.Grey + ",,,,,                   \n")
	sb.WriteString("                    " + CMDColours.Grey + ",,,*" + CMDColours.White + "@@@@@@" + CMDColours.Grey + ",,,,,," + CMDColours.Green + "%%" + CMDColours.Grey + ",,  ,,,,,,," + CMDColours.White + "@" + CMDColours.Green + "*%%," + CMDColours.White + "@" + CMDColours.Grey + ",,,,,,                   \n")
	sb.WriteString("                       " + CMDColours.Grey + "*" + CMDColours.White + "@@@@@@" + CMDColours.Grey + ",,,,,,,,,     " + CMDColours.Grey + ",,,,," + CMDColours.White + "@@@@@@" + CMDColours.Grey + ",,,,,," + CMDColours.White + "                    \n")
	sb.WriteString("                         " + CMDColours.White + "@@@@@@" + CMDColours.Grey + ",,,,,,,,,        " + CMDColours.White + "@@@@@@@" + CMDColours.Grey + ",,,,,," + CMDColours.White + "                   \n")
	sb.WriteString("                         " + CMDColours.White + "@@@@@@@@" + CMDColours.Grey + ",,,,,,,       " + CMDColours.White + "@@@@@@@" + CMDColours.Grey + ",,,,,,," + CMDColours.White + "                   \n")
	sb.WriteString("                           " + CMDColours.White + "@@@@@@@@@" + CMDColours.Grey + ",,,,    " + CMDColours.White + "@@@@@@@@@" + CMDColours.Grey + "#,,,,,,," + CMDColours.White + "                   \n")
	sb.WriteString("                            " + CMDColours.White + "@@@@@@@@@@@@@@@@@@@@@@@     " + CMDColours.Grey + "*,,,," + CMDColours.White + "                   \n")
	sb.WriteString("                                " + CMDColours.White + "@@@@@@@@@@@@@@@@" + CMDColours.Default + "                                \n")
	sb.WriteString(ASCIILogo)
	return sb.String()
}
