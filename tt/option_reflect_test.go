package tt

import (
	"fmt"
	"testing"
)

type CheckReflectEntity struct {
	PlainString         string
	PlainStringRequired string `csv:",required"`
	StopID              String `csv:",required"`
	Name                String `csv:",required"`
	Desc                String
	Timezone            Timezone
	LocationType        Int
	AgencyURL           Url
	BaseEntity
}

func TestCheckReflect(t *testing.T) {
	ent := CheckReflectEntity{
		Name:         NewString("ok"),
		LocationType: NewInt(2),
		Timezone:     Timezone{Option: NewOption("asd")},
		AgencyURL:    Url{Option: NewOption("xyz")},
	}
	// ent.AddError(errors.New("test load error"))
	entErrs := CheckReflect(&ent)
	for _, entErr := range entErrs {
		fmt.Printf("entErrs: %#v\n", entErr)
	}
}
