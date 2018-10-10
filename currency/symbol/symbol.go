package symbol

import "errors"

// Const declarations for individual currencies/tokens/fiat
// An ever growing list. Cares not for equivalence, just is
const (
	BTC     = "BTC"
	LTC     = "LTC"
	ETH     = "ETH"
	XRP     = "XRP"
	BCH     = "BCH"
	EOS     = "EOS"
	XLM     = "XLM"
	USDT    = "USDT"
	ADA     = "ADA"
	XMR     = "XMR"
	TRX     = "TRX"
	MIOTA   = "MIOTA"
	DASH    = "DASH"
	BNB     = "BNB"
	NEO     = "NEO"
	ETC     = "ETC"
	XEM     = "XEM"
	XTZ     = "XTZ"
	VET     = "VET"
	DOGE    = "DOGE"
	ZEC     = "ZEC"
	OMG     = "OMG"
	BTG     = "BTG"
	MKR     = "MKR"
	BCN     = "BCN"
	ONT     = "ONT"
	ZRX     = "ZRX"
	LSK     = "LSK"
	DCR     = "DCR"
	QTUM    = "QTUM"
	BCD     = "BCD"
	BTS     = "BTS"
	NANO    = "NANO"
	ZIL     = "ZIL"
	SC      = "SC"
	DGB     = "DGB"
	ICX     = "ICX"
	STEEM   = "STEEM"
	AE      = "AE"
	XVG     = "XVG"
	WAVES   = "WAVES"
	NPXS    = "NPXS"
	ETN     = "ETN"
	BTM     = "BTM"
	BAT     = "BAT"
	ETP     = "ETP"
	HOT     = "HOT"
	STRAT   = "STRAT"
	GNT     = "GNT"
	REP     = "REP"
	SNT     = "SNT"
	PPT     = "PPT"
	KMD     = "KMD"
	TUSD    = "TUSD"
	CNX     = "CNX"
	LINK    = "LINK"
	WTC     = "WTC"
	ARDR    = "ARDR"
	WAN     = "WAN"
	MITH    = "MITH"
	RDD     = "RDD"
	IOST    = "IOST"
	IOT     = "IOT"
	KCS     = "KCS"
	MAID    = "MAID"
	XET     = "XET"
	MOAC    = "MOAC"
	HC      = "HC"
	AION    = "AION"
	AOA     = "AOA"
	HT      = "HT"
	ELF     = "ELF"
	LRC     = "LRC"
	BNT     = "BNT"
	CMT     = "CMT"
	DGD     = "DGD"
	DCN     = "DCN"
	FUN     = "FUN"
	GXS     = "GXS"
	DROP    = "DROP"
	MANA    = "MANA"
	PAY     = "PAY"
	MCO     = "MCO"
	THETA   = "THETA"
	NXT     = "NXT"
	NOAH    = "NOAH"
	LOOM    = "LOOM"
	POWR    = "POWR"
	WAX     = "WAX"
	ELA     = "ELA"
	PIVX    = "PIVX"
	XIN     = "XIN"
	DAI     = "DAI"
	BTCP    = "BTCP"
	NEXO    = "NEXO"
	XBT     = "XBT"
	SAN     = "SAN"
	GAS     = "GAS"
	BCC     = "BCC"
	HCC     = "HCC"
	OAX     = "OAX"
	DNT     = "DNT"
	ICN     = "ICN"
	LLT     = "LLT"
	YOYO    = "YOYO"
	SNGLS   = "SNGLS"
	BQX     = "BQX"
	KNC     = "KNC"
	SNM     = "SNM"
	CTR     = "CTR"
	SALT    = "SALT"
	MDA     = "MDA"
	IOTA    = "IOTA"
	SUB     = "SUB"
	MTL     = "MTL"
	MTH     = "MTH"
	ENG     = "ENG"
	AST     = "AST"
	EVX     = "EVX"
	REQ     = "REQ"
	VIB     = "VIB"
	ARK     = "ARK"
	MOD     = "MOD"
	ENJ     = "ENJ"
	STORJ   = "STORJ"
	RCN     = "RCN"
	NULS    = "NULS"
	RDN     = "RDN"
	DLT     = "DLT"
	AMB     = "AMB"
	BCPT    = "BCPT"
	ARN     = "ARN"
	GVT     = "GVT"
	CDT     = "CDT"
	POE     = "POE"
	QSP     = "QSP"
	XZC     = "XZC"
	TNT     = "TNT"
	FUEL    = "FUEL"
	ADX     = "ADX"
	CND     = "CND"
	LEND    = "LEND"
	WABI    = "WABI"
	SBTC    = "SBTC"
	BCX     = "BCX"
	TNB     = "TNB"
	GTO     = "GTO"
	OST     = "OST"
	CVC     = "CVC"
	DATA    = "DATA"
	ETF     = "ETF"
	BRD     = "BRD"
	NEBL    = "NEBL"
	VIBE    = "VIBE"
	LUN     = "LUN"
	CHAT    = "CHAT"
	RLC     = "RLC"
	INS     = "INS"
	VIA     = "VIA"
	BLZ     = "BLZ"
	SYS     = "SYS"
	NCASH   = "NCASH"
	POA     = "POA"
	STORM   = "STORM"
	WPR     = "WPR"
	QLC     = "QLC"
	GRS     = "GRS"
	CLOAK   = "CLOAK"
	ZEN     = "ZEN"
	SKY     = "SKY"
	IOTX    = "IOTX"
	QKC     = "QKC"
	AGI     = "AGI"
	NXS     = "NXS"
	EON     = "EON"
	KEY     = "KEY"
	NAS     = "NAS"
	ADD     = "ADD"
	MEETONE = "MEETONE"
	ATD     = "ATD"
	MFT     = "MFT"
	EOP     = "EOP"
	DENT    = "DENT"
	IQ      = "IQ"
	DOCK    = "DOCK"
	POLY    = "POLY"
	VTHO    = "VTHO"
	ONG     = "ONG"
	PHX     = "PHX"
	GO      = "GO"
	PAX     = "PAX"
	EDO     = "EDO"
	WINGS   = "WINGS"
	NAV     = "NAV"
	TRIG    = "TRIG"
	APPC    = "APPC"
	KRW     = "KRW"
	HSR     = "HSR"
	ETHOS   = "ETHOS"
	CTXC    = "CTXC"
	ITC     = "ITC"
	TRUE    = "TRUE"
	ABT     = "ABT"
	RNT     = "RNT"
	PLY     = "PLY"
	PST     = "PST"
	HKD     = "HKD"
	AUD     = "AUD"
	USD     = "USD"
	EUR     = "EUR"
)

// symbols map holds the currency name and symbol mappings
var symbols = map[string]string{
	"ALL": "Lek",
	"AFN": "؋",
	"ARS": "$",
	"AWG": "ƒ",
	"AUD": "$",
	"AZN": "ман",
	"BSD": "$",
	"BBD": "$",
	"BYN": "Br",
	"BZD": "BZ$",
	"BMD": "$",
	"BOB": "$b",
	"BAM": "KM",
	"BWP": "P",
	"BGN": "лв",
	"BRL": "R$",
	"BND": "$",
	"KHR": "៛",
	"CAD": "$",
	"KYD": "$",
	"CLP": "$",
	"CNY": "¥",
	"COP": "$",
	"CRC": "₡",
	"HRK": "kn",
	"CUP": "₱",
	"CZK": "Kč",
	"DKK": "kr",
	"DOP": "RD$",
	"XCD": "$",
	"EGP": "£",
	"SVC": "$",
	"EUR": "€",
	"FKP": "£",
	"FJD": "$",
	"GHS": "¢",
	"GIP": "£",
	"GTQ": "Q",
	"GGP": "£",
	"GYD": "$",
	"HNL": "L",
	"HKD": "$",
	"HUF": "Ft",
	"ISK": "kr",
	"INR": "₹",
	"IDR": "Rp",
	"IRR": "﷼",
	"IMP": "£",
	"ILS": "₪",
	"JMD": "J$",
	"JPY": "¥",
	"JEP": "£",
	"KZT": "лв",
	"KPW": "₩",
	"KRW": "₩",
	"KGS": "лв",
	"LAK": "₭",
	"LBP": "£",
	"LRD": "$",
	"MKD": "ден",
	"MYR": "RM",
	"MUR": "₨",
	"MXN": "$",
	"MNT": "₮",
	"MZN": "MT",
	"NAD": "$",
	"NPR": "₨",
	"ANG": "ƒ",
	"NZD": "$",
	"NIO": "C$",
	"NGN": "₦",
	"NOK": "kr",
	"OMR": "﷼",
	"PKR": "₨",
	"PAB": "B/.",
	"PYG": "Gs",
	"PEN": "S/.",
	"PHP": "₱",
	"PLN": "zł",
	"QAR": "﷼",
	"RON": "lei",
	"RUB": "₽",
	"RUR": "₽",
	"SHP": "£",
	"SAR": "﷼",
	"RSD": "Дин.",
	"SCR": "₨",
	"SGD": "$",
	"SBD": "$",
	"SOS": "S",
	"ZAR": "R",
	"LKR": "₨",
	"SEK": "kr",
	"CHF": "CHF",
	"SRD": "$",
	"SYP": "£",
	"TWD": "NT$",
	"THB": "฿",
	"TTD": "TT$",
	"TRY": "₺",
	"TVD": "$",
	"UAH": "₴",
	"GBP": "£",
	"USD": "$",
	"UYU": "$U",
	"UZS": "лв",
	"VEF": "Bs",
	"VND": "₫",
	"YER": "﷼",
	"ZWD": "Z$",
}

// GetSymbolByCurrencyName returns a currency symbol
func GetSymbolByCurrencyName(currency string) (string, error) {
	result, ok := symbols[currency]
	if !ok {
		return "", errors.New("currency symbol not found")
	}
	return result, nil
}
