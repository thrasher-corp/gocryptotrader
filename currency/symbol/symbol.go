package symbol

import "errors"

// Const declarations for individual currencies/tokens/fiat
// An ever growing list. Cares not for equivalence, just is
const (
	BTC      = "BTC"
	LTC      = "LTC"
	ETH      = "ETH"
	XRP      = "XRP"
	BCH      = "BCH"
	EOS      = "EOS"
	XLM      = "XLM"
	USDT     = "USDT"
	ADA      = "ADA"
	XMR      = "XMR"
	TRX      = "TRX"
	MIOTA    = "MIOTA"
	DASH     = "DASH"
	BNB      = "BNB"
	NEO      = "NEO"
	ETC      = "ETC"
	XEM      = "XEM"
	XTZ      = "XTZ"
	VET      = "VET"
	DOGE     = "DOGE"
	ZEC      = "ZEC"
	OMG      = "OMG"
	BTG      = "BTG"
	MKR      = "MKR"
	BCN      = "BCN"
	ONT      = "ONT"
	ZRX      = "ZRX"
	LSK      = "LSK"
	DCR      = "DCR"
	QTUM     = "QTUM"
	BCD      = "BCD"
	BTS      = "BTS"
	NANO     = "NANO"
	ZIL      = "ZIL"
	SC       = "SC"
	DGB      = "DGB"
	ICX      = "ICX"
	STEEM    = "STEEM"
	AE       = "AE"
	XVG      = "XVG"
	WAVES    = "WAVES"
	NPXS     = "NPXS"
	ETN      = "ETN"
	BTM      = "BTM"
	BAT      = "BAT"
	ETP      = "ETP"
	HOT      = "HOT"
	STRAT    = "STRAT"
	GNT      = "GNT"
	REP      = "REP"
	SNT      = "SNT"
	PPT      = "PPT"
	KMD      = "KMD"
	TUSD     = "TUSD"
	CNX      = "CNX"
	LINK     = "LINK"
	WTC      = "WTC"
	ARDR     = "ARDR"
	WAN      = "WAN"
	MITH     = "MITH"
	RDD      = "RDD"
	IOST     = "IOST"
	IOT      = "IOT"
	KCS      = "KCS"
	MAID     = "MAID"
	XET      = "XET"
	MOAC     = "MOAC"
	HC       = "HC"
	AION     = "AION"
	AOA      = "AOA"
	HT       = "HT"
	ELF      = "ELF"
	LRC      = "LRC"
	BNT      = "BNT"
	CMT      = "CMT"
	DGD      = "DGD"
	DCN      = "DCN"
	FUN      = "FUN"
	GXS      = "GXS"
	DROP     = "DROP"
	MANA     = "MANA"
	PAY      = "PAY"
	MCO      = "MCO"
	THETA    = "THETA"
	NXT      = "NXT"
	NOAH     = "NOAH"
	LOOM     = "LOOM"
	POWR     = "POWR"
	WAX      = "WAX"
	ELA      = "ELA"
	PIVX     = "PIVX"
	XIN      = "XIN"
	DAI      = "DAI"
	BTCP     = "BTCP"
	NEXO     = "NEXO"
	XBT      = "XBT"
	SAN      = "SAN"
	GAS      = "GAS"
	BCC      = "BCC"
	HCC      = "HCC"
	OAX      = "OAX"
	DNT      = "DNT"
	ICN      = "ICN"
	LLT      = "LLT"
	YOYO     = "YOYO"
	SNGLS    = "SNGLS"
	BQX      = "BQX"
	KNC      = "KNC"
	SNM      = "SNM"
	CTR      = "CTR"
	SALT     = "SALT"
	MDA      = "MDA"
	IOTA     = "IOTA"
	SUB      = "SUB"
	MTL      = "MTL"
	MTH      = "MTH"
	ENG      = "ENG"
	AST      = "AST"
	EVX      = "EVX"
	REQ      = "REQ"
	VIB      = "VIB"
	ARK      = "ARK"
	MOD      = "MOD"
	ENJ      = "ENJ"
	STORJ    = "STORJ"
	RCN      = "RCN"
	NULS     = "NULS"
	RDN      = "RDN"
	DLT      = "DLT"
	AMB      = "AMB"
	BCPT     = "BCPT"
	ARN      = "ARN"
	GVT      = "GVT"
	CDT      = "CDT"
	POE      = "POE"
	QSP      = "QSP"
	XZC      = "XZC"
	TNT      = "TNT"
	FUEL     = "FUEL"
	ADX      = "ADX"
	CND      = "CND"
	LEND     = "LEND"
	WABI     = "WABI"
	SBTC     = "SBTC"
	BCX      = "BCX"
	TNB      = "TNB"
	GTO      = "GTO"
	OST      = "OST"
	CVC      = "CVC"
	DATA     = "DATA"
	ETF      = "ETF"
	BRD      = "BRD"
	NEBL     = "NEBL"
	VIBE     = "VIBE"
	LUN      = "LUN"
	CHAT     = "CHAT"
	RLC      = "RLC"
	INS      = "INS"
	VIA      = "VIA"
	BLZ      = "BLZ"
	SYS      = "SYS"
	NCASH    = "NCASH"
	POA      = "POA"
	STORM    = "STORM"
	WPR      = "WPR"
	QLC      = "QLC"
	GRS      = "GRS"
	CLOAK    = "CLOAK"
	ZEN      = "ZEN"
	SKY      = "SKY"
	IOTX     = "IOTX"
	QKC      = "QKC"
	AGI      = "AGI"
	NXS      = "NXS"
	EON      = "EON"
	KEY      = "KEY"
	NAS      = "NAS"
	ADD      = "ADD"
	MEETONE  = "MEETONE"
	ATD      = "ATD"
	MFT      = "MFT"
	EOP      = "EOP"
	DENT     = "DENT"
	IQ       = "IQ"
	DOCK     = "DOCK"
	POLY     = "POLY"
	VTHO     = "VTHO"
	ONG      = "ONG"
	PHX      = "PHX"
	GO       = "GO"
	PAX      = "PAX"
	EDO      = "EDO"
	WINGS    = "WINGS"
	NAV      = "NAV"
	TRIG     = "TRIG"
	APPC     = "APPC"
	KRW      = "KRW"
	HSR      = "HSR"
	ETHOS    = "ETHOS"
	CTXC     = "CTXC"
	ITC      = "ITC"
	TRUE     = "TRUE"
	ABT      = "ABT"
	RNT      = "RNT"
	PLY      = "PLY"
	PST      = "PST"
	KICK     = "KICK"
	BTCZ     = "BTCZ"
	DXT      = "DXT"
	STQ      = "STQ"
	INK      = "INK"
	HBZ      = "HBZ"
	USDT_ETH = "USDT_ETH"
	QTUM_ETH = "QTUM_ETH"
	BTM_ETH  = "BTM_ETH"
	FIL      = "FIL"
	STX      = "STX"
	BOT      = "BOT"
	VERI     = "VERI"
	ZSC      = "ZSC"
	QBT      = "QBT"
	MED      = "MED"
	QASH     = "QASH"
	MDS      = "MDS"
	GOD      = "GOD"
	SMT      = "SMT"
	BTF      = "BTF"
	NAS_ETH  = "NAS_ETH"
	TSL      = "TSL"
	BIFI     = "BIFI"
	BNTY     = "BNTY"
	DRGN     = "DRGN"
	GTC      = "GTC"
	MDT      = "MDT"
	QUN      = "QUN"
	GNX      = "GNX"
	DDD      = "DDD"
	BTO      = "BTO"
	TIO      = "TIO"
	OCN      = "OCN"
	RUFF     = "RUFF"
	TNC      = "TNC"
	SNET     = "SNET"
	COFI     = "COFI"
	ZPT      = "ZPT"
	JNT      = "JNT"
	MTN      = "MTN"
	GEM      = "GEM"
	DADI     = "DADI"
	RFR      = "RFR"
	MOBI     = "MOBI"
	LEDU     = "LEDU"
	DBC      = "DBC"
	MKR_OLD  = "MKR_OLD"
	DPY      = "DPY"
	BCDN     = "BCDN"
	EOSDAC   = "EOSDAC"
	TIPS     = "TIPS"
	XMC      = "XMC"
	PPS      = "PPS"
	BOE      = "BOE"
	MEDX     = "MEDX"
	SMT_ETH  = "SMT_ETH"
	CS       = "CS"
	MAN      = "MAN"
	REM      = "REM"
	LYM      = "LYM"
	INSTAR   = "INSTAR"
	BFT      = "BFT"
	IHT      = "IHT"
	SENC     = "SENC"
	TOMO     = "TOMO"
	ELEC     = "ELEC"
	SHIP     = "SHIP"
	TFD      = "TFD"
	HAV      = "HAV"
	HUR      = "HUR"
	LST      = "LST"
	LINO     = "LINO"
	SWTH     = "SWTH"
	NKN      = "NKN"
	SOUL     = "SOUL"
	GALA_NEO = "GALA_NEO"
	LRN      = "LRN"
	GSE      = "GSE"
	RATING   = "RATING"
	HSC      = "HSC"
	HIT      = "HIT"
	DX       = "DX"
	BXC      = "BXC"
	GARD     = "GARD"
	FTI      = "FTI"
	SOP      = "SOP"
	LEMO     = "LEMO"
	RED      = "RED"
	LBA      = "LBA"
	KAN      = "KAN"
	OPEN     = "OPEN"
	SKM      = "SKM"
	NBAI     = "NBAI"
	UPP      = "UPP"
	ATMI     = "ATMI"
	TMT      = "TMT"
	BBK      = "BBK"
	EDR      = "EDR"
	MET      = "MET"
	TCT      = "TCT"
	EXC      = "EXC"
	CNC      = "CNC"
	TIX      = "TIX"
	XTC      = "XTC"
	BU       = "BU"
	XXBT     = "XXBT" // BTC, but XXBT instead
	HKD      = "HKD"  // Hong Kong Dollar
	AUD      = "AUD"  // Australian Dollar
	USD      = "USD"  // United States Dollar
	ZUSD     = "ZUSD" // United States Dollar, but with a Z in front of it
	EUR      = "EUR"  // Euro
	ZEUR     = "ZEUR" // Euro, but with a Z in front of it
	CAD      = "CAD"  // Canadaian Dollar
	ZCAD     = "ZCAD" // Canadaian Dollar, but with a Z in front of it
	SGD      = "SGD"  // Singapore Dollar
	RUB      = "RUB"  // Russian Ruble
	PLN      = "PLN"  // Polish złoty
	TRY      = "TRY"  // Turkish lira
	UAH      = "UAH"  // Ukrainian hryvnia
	JPY      = "JPY"  // Japanese yen
	ZJPY     = "ZJPY" // Japanese yen, but with a Z in front of it
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
