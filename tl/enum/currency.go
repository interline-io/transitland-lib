package enum

import (
	"fmt"
	"strings"
)

// Currency is a nullable string describing a currency, plus validation and serialization methods.
type Currency struct {
	Option[string]
}

func NewCurrency(v string) Currency {
	a := Currency{}
	a.Scan(v)
	return a
}

func (r Currency) String() string {
	return r.Val
}

func (r *Currency) Error() error {
	if !IsValidCurrency(r.Val) {
		return &InvalidCurrencyError{Value: r.Val}
	}
	return nil
}

// Errors, helpers

// InvalidCurrencyError represents an invalid currency name.
type InvalidCurrencyError struct {
	Value string
}

func (e *InvalidCurrencyError) Error() string {
	return fmt.Sprintf("invalid currency: '%s'", e.Value)
}

// IsValidCurrency check is valid currency
func IsValidCurrency(value string) bool {
	if len(value) == 0 {
		return true
	}
	_, ok := currencies[strings.ToLower(value)]
	return ok
}

// Currency list
// https://en.wikipedia.org/wiki/iso_4217
var currencies = map[string]bool{
	"aed": true,
	"afn": true,
	"all": true,
	"amd": true,
	"ang": true,
	"aoa": true,
	"ars": true,
	"aud": true,
	"awg": true,
	"azn": true,
	"bam": true,
	"bbd": true,
	"bdt": true,
	"bgn": true,
	"bhd": true,
	"bif": true,
	"bmd": true,
	"bnd": true,
	"bob": true,
	"bov": true,
	"brl": true,
	"bsd": true,
	"btn": true,
	"bwp": true,
	"byn": true,
	"bzd": true,
	"cad": true,
	"cdf": true,
	"che": true,
	"chf": true,
	"chw": true,
	"clf": true,
	"clp": true,
	"cny": true,
	"cop": true,
	"cou": true,
	"crc": true,
	"cuc": true,
	"cup": true,
	"cve": true,
	"czk": true,
	"djf": true,
	"dkk": true,
	"dop": true,
	"dzd": true,
	"egp": true,
	"ern": true,
	"etb": true,
	"eur": true,
	"fjd": true,
	"fkp": true,
	"gbp": true,
	"gel": true,
	"ghs": true,
	"gip": true,
	"gmd": true,
	"gnf": true,
	"gtq": true,
	"gyd": true,
	"hkd": true,
	"hnl": true,
	"hrk": true,
	"htg": true,
	"huf": true,
	"idr": true,
	"ils": true,
	"inr": true,
	"iqd": true,
	"irr": true,
	"isk": true,
	"jmd": true,
	"jod": true,
	"jpy": true,
	"kes": true,
	"kgs": true,
	"khr": true,
	"kmf": true,
	"kpw": true,
	"krw": true,
	"kwd": true,
	"kyd": true,
	"kzt": true,
	"lak": true,
	"lbp": true,
	"lkr": true,
	"lrd": true,
	"lsl": true,
	"lyd": true,
	"mad": true,
	"mdl": true,
	"mga": true,
	"mkd": true,
	"mmk": true,
	"mnt": true,
	"mop": true,
	"mru": true,
	"mur": true,
	"mvr": true,
	"mwk": true,
	"mxn": true,
	"mxv": true,
	"myr": true,
	"mzn": true,
	"nad": true,
	"ngn": true,
	"nio": true,
	"nok": true,
	"npr": true,
	"nzd": true,
	"omr": true,
	"pab": true,
	"pen": true,
	"pgk": true,
	"php": true,
	"pkr": true,
	"pln": true,
	"pyg": true,
	"qar": true,
	"ron": true,
	"rsd": true,
	"rub": true,
	"rwf": true,
	"sar": true,
	"sbd": true,
	"scr": true,
	"sdg": true,
	"sek": true,
	"sgd": true,
	"shp": true,
	"sll": true,
	"sos": true,
	"srd": true,
	"ssp": true,
	"stn": true,
	"svc": true,
	"syp": true,
	"szl": true,
	"thb": true,
	"tjs": true,
	"tmt": true,
	"tnd": true,
	"top": true,
	"try": true,
	"ttd": true,
	"twd": true,
	"tzs": true,
	"uah": true,
	"ugx": true,
	"usd": true,
	"usn": true,
	"uyi": true,
	"uyu": true,
	"uyw": true,
	"uzs": true,
	"ves": true,
	"vnd": true,
	"vuv": true,
	"wst": true,
	"xaf": true,
	"xag": true,
	"xau": true,
	"xba": true,
	"xbb": true,
	"xbc": true,
	"xbd": true,
	"xcd": true,
	"xdr": true,
	"xof": true,
	"xpd": true,
	"xpf": true,
	"xpt": true,
	"xsu": true,
	"xts": true,
	"xua": true,
	"xxx": true,
	"yer": true,
	"zar": true,
	"zmw": true,
	"zwl": true,
}
