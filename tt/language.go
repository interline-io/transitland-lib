package tt

import (
	"errors"
	"fmt"
	"strings"

	"github.com/interline-io/transitland-lib/causes"
)

type Language struct {
	Option[string]
}

func (r Language) Check() error {
	if r.Valid && !IsValidLanguage(r.Val) {
		return errors.New("invalid language")
	}
	return nil
}

func NewLanguage(v string) Language {
	return Language{Option: NewOption(v)}
}

// IsValidLang check is valid language
func IsValidLanguage(value string) bool {
	if len(value) == 0 {
		return true
	}
	// Only check the prefix code
	code := strings.Split(value, "-")
	_, ok := langs[strings.ToLower(code[0])]
	return ok
}

// CheckLanguage returns an error if the value is not a known language
func CheckLanguage(field string, value string) (errs []error) {
	if !IsValidLanguage(value) {
		errs = append(errs, causes.NewInvalidFieldError(field, value, fmt.Errorf("invalid language")))
	}
	return errs
}

// http://www.loc.gov/standards/iso639-2/php/code_list.php
var langs = map[string]bool{
	"aa": true,
	"ab": true,
	"ae": true,
	"af": true,
	"ak": true,
	"am": true,
	"an": true,
	"ar": true,
	"as": true,
	"av": true,
	"ay": true,
	"az": true,
	"ba": true,
	"be": true,
	"bg": true,
	"bh": true,
	"bi": true,
	"bm": true,
	"bn": true,
	"bo": true,
	"br": true,
	"bs": true,
	"ca": true,
	"ce": true,
	"ch": true,
	"co": true,
	"cr": true,
	"cs": true,
	"cu": true,
	"cv": true,
	"cy": true,
	"da": true,
	"de": true,
	"dv": true,
	"dz": true,
	"ee": true,
	"el": true,
	"en": true,
	"eo": true,
	"es": true,
	"et": true,
	"eu": true,
	"fa": true,
	"ff": true,
	"fi": true,
	"fj": true,
	"fo": true,
	"fr": true,
	"fy": true,
	"ga": true,
	"gd": true,
	"gl": true,
	"gn": true,
	"gu": true,
	"gv": true,
	"ha": true,
	"he": true,
	"hi": true,
	"ho": true,
	"hr": true,
	"ht": true,
	"hu": true,
	"hy": true,
	"hz": true,
	"ia": true,
	"id": true,
	"ie": true,
	"ig": true,
	"ii": true,
	"ik": true,
	"io": true,
	"is": true,
	"it": true,
	"iu": true,
	"ja": true,
	"jv": true,
	"ka": true,
	"kg": true,
	"ki": true,
	"kj": true,
	"kk": true,
	"kl": true,
	"km": true,
	"kn": true,
	"ko": true,
	"kr": true,
	"ks": true,
	"ku": true,
	"kv": true,
	"kw": true,
	"ky": true,
	"la": true,
	"lb": true,
	"lg": true,
	"li": true,
	"ln": true,
	"lo": true,
	"lt": true,
	"lu": true,
	"lv": true,
	"mg": true,
	"mh": true,
	"mi": true,
	"mk": true,
	"ml": true,
	"mn": true,
	"mr": true,
	"ms": true,
	"mt": true,
	"my": true,
	"na": true,
	"nb": true,
	"nd": true,
	"ne": true,
	"ng": true,
	"nl": true,
	"nn": true,
	"no": true,
	"nr": true,
	"nv": true,
	"ny": true,
	"oc": true,
	"oj": true,
	"om": true,
	"or": true,
	"os": true,
	"pa": true,
	"pi": true,
	"pl": true,
	"ps": true,
	"pt": true,
	"qu": true,
	"rm": true,
	"rn": true,
	"ro": true,
	"ru": true,
	"rw": true,
	"sa": true,
	"sc": true,
	"sd": true,
	"se": true,
	"sg": true,
	"si": true,
	"sk": true,
	"sl": true,
	"sm": true,
	"sn": true,
	"so": true,
	"sq": true,
	"sr": true,
	"ss": true,
	"st": true,
	"su": true,
	"sv": true,
	"sw": true,
	"ta": true,
	"te": true,
	"tg": true,
	"th": true,
	"ti": true,
	"tk": true,
	"tl": true,
	"tn": true,
	"to": true,
	"tr": true,
	"ts": true,
	"tt": true,
	"tw": true,
	"ty": true,
	"ug": true,
	"uk": true,
	"ur": true,
	"uz": true,
	"ve": true,
	"vi": true,
	"vo": true,
	"wa": true,
	"wo": true,
	"xh": true,
	"yi": true,
	"yo": true,
	"za": true,
	"zh": true,
	"zu": true,
}
