package enum

import (
	"database/sql/driver"
	"encoding/json"
	"io"
	"strings"
)

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

type InvalidLanguageError struct {
	bc
}

func (e *InvalidLanguageError) Error() string { return "" }

func NewInvalidLanguageError(value string) error {
	return &InvalidLanguageError{
		bc: bc{
			Value: value,
		},
	}
}

// IsValidLanguage check is valid Language
func IsValidLanguage(value string) bool {
	if len(value) == 0 {
		return true
	}
	// Only check the prefix code
	code := strings.Split(value, "-")
	_, ok := langs[strings.ToLower(code[0])]
	return ok
}

type Language struct {
	value string
	valid bool
}

func NewLanguage(v string) Language {
	a := Language{}
	a.Set(v)
	return a
}

func (r *Language) Set(v string) bool {
	r.value, r.valid = v, false
	ok := IsValidLanguage(v)
	if v != "" && ok {
		r.value = v
		r.valid = true
	}
	return r.valid
}

func (r *Language) IsValid() bool {
	return r.valid
}

func (r *Language) String() string {
	return r.value
}

func (r *Language) Error() error {
	if r.value != "" && !r.valid {
		return NewInvalidLanguageError(r.value)
	}
	return nil
}

func (r Language) Value() (driver.Value, error) {
	if !r.valid || r.value == "" {
		return nil, nil
	}
	return r.value, nil
}

func (r *Language) Scan(src interface{}) error {
	r.Set(toString(src))
	return nil
}

func (r *Language) UnmarshalJSON(v []byte) error {
	c := ""
	if err := json.Unmarshal(v, &c); err != nil {
		return err
	}
	return r.Scan(c)
}

func (r *Language) MarshalJSON() ([]byte, error) {
	if !r.valid {
		return []byte("null"), nil
	}
	return json.Marshal(r.value)
}

func (r *Language) UnmarshalGQL(v interface{}) error {
	return r.Scan(v)
}

func (r Language) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}
