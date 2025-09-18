package tt

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/interline-io/transitland-lib/causes"
)

type CurrencyAmount struct {
	Option[float64]
	digitsPlusOne int
}

func NewCurrencyAmount(v float64) CurrencyAmount {
	return CurrencyAmount{Option: Option[float64]{Valid: true, Val: v}}
}

func (r CurrencyAmount) ToCsv() string {
	if r.Valid {
		return strconv.FormatFloat(r.Val, 'f', r.digitsPlusOne-1, 64)
	}
	return ""
}

func (r *CurrencyAmount) SetCurrency(value string) error {
	// Empty currency value allowed
	// Setting an invalid currency does not set Valid = false
	r.digitsPlusOne = 0
	if value == "" {
		return nil
	}
	if cur, ok := currencies[strings.ToUpper(value)]; ok {
		r.digitsPlusOne = cur.digits + 1
	} else {
		return errors.New("invalid currency")
	}
	return nil
}

type Currency struct {
	Option[string]
}

func NewCurrency(v string) Currency {
	return Currency{Option: NewOption(v)}
}

func (r Currency) Check() error {
	if r.Valid && !IsValidCurrency(r.Val) {
		return causes.NewInvalidFieldError("", r.Val, fmt.Errorf("invalid currency"))
	}
	return nil
}

// IsValidCurrency check is valid currency
func IsValidCurrency(value string) bool {
	_, ok := currencies[strings.ToUpper(value)]
	return ok
}

type currencyInfo struct {
	name   string
	code   int
	digits int
}

// based on https://en.wikipedia.org/wiki/iso_4217
var currencies = map[string]currencyInfo{
	"CLF": {name: "CLF", code: 990, digits: 4},
	"UYW": {name: "UYW", code: 927, digits: 4},
	"BHD": {name: "BHD", code: 48, digits: 3},
	"IQD": {name: "IQD", code: 368, digits: 3},
	"JOD": {name: "JOD", code: 400, digits: 3},
	"KWD": {name: "KWD", code: 414, digits: 3},
	"LYD": {name: "LYD", code: 434, digits: 3},
	"OMR": {name: "OMR", code: 512, digits: 3},
	"TND": {name: "TND", code: 788, digits: 3},
	"AED": {name: "AED", code: 784, digits: 2},
	"AFN": {name: "AFN", code: 971, digits: 2},
	"ALL": {name: "ALL", code: 8, digits: 2},
	"AMD": {name: "AMD", code: 51, digits: 2},
	"ANG": {name: "ANG", code: 532, digits: 2},
	"AOA": {name: "AOA", code: 973, digits: 2},
	"ARS": {name: "ARS", code: 32, digits: 2},
	"AUD": {name: "AUD", code: 36, digits: 2},
	"AWG": {name: "AWG", code: 533, digits: 2},
	"AZN": {name: "AZN", code: 944, digits: 2},
	"BAM": {name: "BAM", code: 977, digits: 2},
	"BBD": {name: "BBD", code: 52, digits: 2},
	"BDT": {name: "BDT", code: 50, digits: 2},
	"BGN": {name: "BGN", code: 975, digits: 2},
	"BMD": {name: "BMD", code: 60, digits: 2},
	"BND": {name: "BND", code: 96, digits: 2},
	"BOB": {name: "BOB", code: 68, digits: 2},
	"BOV": {name: "BOV", code: 984, digits: 2},
	"BRL": {name: "BRL", code: 986, digits: 2},
	"BSD": {name: "BSD", code: 44, digits: 2},
	"BTN": {name: "BTN", code: 64, digits: 2},
	"BWP": {name: "BWP", code: 72, digits: 2},
	"BYN": {name: "BYN", code: 933, digits: 2},
	"BZD": {name: "BZD", code: 84, digits: 2},
	"CAD": {name: "CAD", code: 124, digits: 2},
	"CDF": {name: "CDF", code: 976, digits: 2},
	"CHE": {name: "CHE", code: 947, digits: 2},
	"CHF": {name: "CHF", code: 756, digits: 2},
	"CHW": {name: "CHW", code: 948, digits: 2},
	"COP": {name: "COP", code: 170, digits: 2},
	"COU": {name: "COU", code: 970, digits: 2},
	"CRC": {name: "CRC", code: 188, digits: 2},
	"CUC": {name: "CUC", code: 931, digits: 2},
	"CUP": {name: "CUP", code: 192, digits: 2},
	"CVE": {name: "CVE", code: 132, digits: 2},
	"CZK": {name: "CZK", code: 203, digits: 2},
	"DKK": {name: "DKK", code: 208, digits: 2},
	"DOP": {name: "DOP", code: 214, digits: 2},
	"DZD": {name: "DZD", code: 12, digits: 2},
	"EGP": {name: "EGP", code: 818, digits: 2},
	"ERN": {name: "ERN", code: 232, digits: 2},
	"ETB": {name: "ETB", code: 230, digits: 2},
	"EUR": {name: "EUR", code: 978, digits: 2},
	"FJD": {name: "FJD", code: 242, digits: 2},
	"FKP": {name: "FKP", code: 238, digits: 2},
	"GBP": {name: "GBP", code: 826, digits: 2},
	"GEL": {name: "GEL", code: 981, digits: 2},
	"GHS": {name: "GHS", code: 936, digits: 2},
	"GIP": {name: "GIP", code: 292, digits: 2},
	"GMD": {name: "GMD", code: 270, digits: 2},
	"GTQ": {name: "GTQ", code: 320, digits: 2},
	"GYD": {name: "GYD", code: 328, digits: 2},
	"HKD": {name: "HKD", code: 344, digits: 2},
	"HNL": {name: "HNL", code: 340, digits: 2},
	"HTG": {name: "HTG", code: 332, digits: 2},
	"HUF": {name: "HUF", code: 348, digits: 2},
	"IDR": {name: "IDR", code: 360, digits: 2},
	"ILS": {name: "ILS", code: 376, digits: 2},
	"INR": {name: "INR", code: 356, digits: 2},
	"IRR": {name: "IRR", code: 364, digits: 2},
	"JMD": {name: "JMD", code: 388, digits: 2},
	"KES": {name: "KES", code: 404, digits: 2},
	"KGS": {name: "KGS", code: 417, digits: 2},
	"KHR": {name: "KHR", code: 116, digits: 2},
	"KPW": {name: "KPW", code: 408, digits: 2},
	"KYD": {name: "KYD", code: 136, digits: 2},
	"KZT": {name: "KZT", code: 398, digits: 2},
	"LAK": {name: "LAK", code: 418, digits: 2},
	"LBP": {name: "LBP", code: 422, digits: 2},
	"LKR": {name: "LKR", code: 144, digits: 2},
	"LRD": {name: "LRD", code: 430, digits: 2},
	"LSL": {name: "LSL", code: 426, digits: 2},
	"MAD": {name: "MAD", code: 504, digits: 2},
	"MDL": {name: "MDL", code: 498, digits: 2},
	"MGA": {name: "MGA", code: 969, digits: 2},
	"MKD": {name: "MKD", code: 807, digits: 2},
	"MMK": {name: "MMK", code: 104, digits: 2},
	"MNT": {name: "MNT", code: 496, digits: 2},
	"MOP": {name: "MOP", code: 446, digits: 2},
	"MRU": {name: "MRU", code: 929, digits: 2},
	"MUR": {name: "MUR", code: 480, digits: 2},
	"MVR": {name: "MVR", code: 462, digits: 2},
	"MWK": {name: "MWK", code: 454, digits: 2},
	"MXN": {name: "MXN", code: 484, digits: 2},
	"MXV": {name: "MXV", code: 979, digits: 2},
	"MYR": {name: "MYR", code: 458, digits: 2},
	"MZN": {name: "MZN", code: 943, digits: 2},
	"NAD": {name: "NAD", code: 516, digits: 2},
	"NGN": {name: "NGN", code: 566, digits: 2},
	"NIO": {name: "NIO", code: 558, digits: 2},
	"NOK": {name: "NOK", code: 578, digits: 2},
	"NPR": {name: "NPR", code: 524, digits: 2},
	"NZD": {name: "NZD", code: 554, digits: 2},
	"PAB": {name: "PAB", code: 590, digits: 2},
	"PEN": {name: "PEN", code: 604, digits: 2},
	"PGK": {name: "PGK", code: 598, digits: 2},
	"PHP": {name: "PHP", code: 608, digits: 2},
	"PKR": {name: "PKR", code: 586, digits: 2},
	"PLN": {name: "PLN", code: 985, digits: 2},
	"QAR": {name: "QAR", code: 634, digits: 2},
	"RON": {name: "RON", code: 946, digits: 2},
	"RSD": {name: "RSD", code: 941, digits: 2},
	"CNY": {name: "CNY", code: 156, digits: 2},
	"RUB": {name: "RUB", code: 643, digits: 2},
	"SAR": {name: "SAR", code: 682, digits: 2},
	"SBD": {name: "SBD", code: 90, digits: 2},
	"SCR": {name: "SCR", code: 690, digits: 2},
	"SDG": {name: "SDG", code: 938, digits: 2},
	"SEK": {name: "SEK", code: 752, digits: 2},
	"SGD": {name: "SGD", code: 702, digits: 2},
	"SHP": {name: "SHP", code: 654, digits: 2},
	"SLE": {name: "SLE", code: 925, digits: 2},
	"SLL": {name: "SLL", code: 694, digits: 2},
	"SOS": {name: "SOS", code: 706, digits: 2},
	"SRD": {name: "SRD", code: 968, digits: 2},
	"SSP": {name: "SSP", code: 728, digits: 2},
	"STN": {name: "STN", code: 930, digits: 2},
	"SVC": {name: "SVC", code: 222, digits: 2},
	"SYP": {name: "SYP", code: 760, digits: 2},
	"SZL": {name: "SZL", code: 748, digits: 2},
	"THB": {name: "THB", code: 764, digits: 2},
	"TJS": {name: "TJS", code: 972, digits: 2},
	"TMT": {name: "TMT", code: 934, digits: 2},
	"TOP": {name: "TOP", code: 776, digits: 2},
	"TRY": {name: "TRY", code: 949, digits: 2},
	"TTD": {name: "TTD", code: 780, digits: 2},
	"TWD": {name: "TWD", code: 901, digits: 2},
	"TZS": {name: "TZS", code: 834, digits: 2},
	"UAH": {name: "UAH", code: 980, digits: 2},
	"USD": {name: "USD", code: 840, digits: 2},
	"USN": {name: "USN", code: 997, digits: 2},
	"UYU": {name: "UYU", code: 858, digits: 2},
	"UZS": {name: "UZS", code: 860, digits: 2},
	"VED": {name: "VED", code: 926, digits: 2},
	"VES": {name: "VES", code: 928, digits: 2},
	"WST": {name: "WST", code: 882, digits: 2},
	"XCD": {name: "XCD", code: 951, digits: 2},
	"YER": {name: "YER", code: 886, digits: 2},
	"ZAR": {name: "ZAR", code: 710, digits: 2},
	"ZMW": {name: "ZMW", code: 967, digits: 2},
	"ZWL": {name: "ZWL", code: 932, digits: 2},
	"BIF": {name: "BIF", code: 108, digits: 0},
	"CLP": {name: "CLP", code: 152, digits: 0},
	"DJF": {name: "DJF", code: 262, digits: 0},
	"GNF": {name: "GNF", code: 324, digits: 0},
	"ISK": {name: "ISK", code: 352, digits: 0},
	"JPY": {name: "JPY", code: 392, digits: 0},
	"KMF": {name: "KMF", code: 174, digits: 0},
	"KRW": {name: "KRW", code: 410, digits: 0},
	"PYG": {name: "PYG", code: 600, digits: 0},
	"RWF": {name: "RWF", code: 646, digits: 0},
	"UGX": {name: "UGX", code: 800, digits: 0},
	"UYI": {name: "UYI", code: 940, digits: 0},
	"VND": {name: "VND", code: 704, digits: 0},
	"VUV": {name: "VUV", code: 548, digits: 0},
	"XAF": {name: "XAF", code: 950, digits: 0},
	"XOF": {name: "XOF", code: 952, digits: 0},
	"XPF": {name: "XPF", code: 953, digits: 0},
	"XAG": {name: "XAG", code: 961, digits: 0},
	"XAU": {name: "XAU", code: 959, digits: 0},
	"XBA": {name: "XBA", code: 955, digits: 0},
	"XBB": {name: "XBB", code: 956, digits: 0},
	"XBC": {name: "XBC", code: 957, digits: 0},
	"XBD": {name: "XBD", code: 958, digits: 0},
	"XDR": {name: "XDR", code: 960, digits: 0},
	"XPD": {name: "XPD", code: 964, digits: 0},
	"XPT": {name: "XPT", code: 962, digits: 0},
	"XSU": {name: "XSU", code: 994, digits: 0},
	"XTS": {name: "XTS", code: 963, digits: 0},
	"XUA": {name: "XUA", code: 965, digits: 0},
	"XXX": {name: "XXX", code: 999, digits: 0},
}
