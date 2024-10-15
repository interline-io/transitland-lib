package tt

import (
	"fmt"
	"testing"
)

type ReflectCheckErrorsEntity struct {
	PlainString         string
	PlainStringRequired string `csv:",required"`
	StopID              String `csv:",required"`
	Name                String `csv:",required"`
	Desc                String
	Timezone            Timezone
	LocationType        Int `enum:"0,1,2,3,4"`
	AgencyURL           Url
	BaseEntity
}

func TestReflectCheckErrors(t *testing.T) {
	ent := ReflectCheckErrorsEntity{
		Name:         NewString("ok"),
		LocationType: NewInt(2),
		Timezone:     Timezone{Option: NewOption("asd")},
		AgencyURL:    Url{Option: NewOption("xyz")},
	}
	entErrs := ReflectCheckErrors(&ent)
	for _, entErr := range entErrs {
		fmt.Printf("entErrs: %#v\n", entErr)
	}
}
