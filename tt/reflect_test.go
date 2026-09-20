package tt

import (
	"testing"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/stretchr/testify/assert"
)

func TestReflectCheckErrors(t *testing.T) {
	t.Run("required string error", func(t *testing.T) {
		ent := struct {
			Value String `csv:",required"`
		}{}
		entErr := firstError(ReflectCheckErrors(&ent))
		assert.IsType(t, &causes.RequiredFieldError{}, entErr)
	})
	t.Run("required string ok", func(t *testing.T) {
		ent := struct {
			Value String `csv:",required"`
		}{Value: NewString("ok")}
		entErr := firstError(ReflectCheckErrors(&ent))
		assert.Nil(t, entErr)
	})
	t.Run("enum error", func(t *testing.T) {
		ent := struct {
			Value Int `enum:"0,1,2"`
		}{Value: NewInt(123)}
		entErr := firstError(ReflectCheckErrors(&ent))
		assert.IsType(t, &causes.InvalidFieldError{}, entErr)
	})
	t.Run("enum ok", func(t *testing.T) {
		ent := struct {
			Value Int `enum:"0,1,2"`
		}{Value: NewInt(1)}
		entErr := firstError(ReflectCheckErrors(&ent))
		assert.Nil(t, entErr)
	})
	// GT
	t.Run("gt ok", func(t *testing.T) {
		ent := struct {
			Value Float `gt:"0"`
		}{Value: NewFloat(1)}
		entErr := firstError(ReflectCheckErrors(&ent))
		assert.Nil(t, entErr)
	})
	t.Run("gt error", func(t *testing.T) {
		ent := struct {
			Value Float `gt:"0"`
		}{Value: NewFloat(0)}
		entErr := firstError(ReflectCheckErrors(&ent))
		assert.IsType(t, &causes.InvalidFieldError{}, entErr)
	})
	t.Run("gt error", func(t *testing.T) {
		ent := struct {
			Value Float `gt:"0"`
		}{Value: NewFloat(-1)}
		entErr := firstError(ReflectCheckErrors(&ent))
		assert.IsType(t, &causes.InvalidFieldError{}, entErr)
	})
	// GTE
	t.Run("gte ok", func(t *testing.T) {
		ent := struct {
			Value Float `gte:"0"`
		}{Value: NewFloat(0)}
		entErr := firstError(ReflectCheckErrors(&ent))
		assert.Nil(t, entErr)
	})
	t.Run("gte error", func(t *testing.T) {
		ent := struct {
			Value Float `gte:"0"`
		}{Value: NewFloat(-1)}
		entErr := firstError(ReflectCheckErrors(&ent))
		assert.IsType(t, &causes.InvalidFieldError{}, entErr)
	})
	// LT
	t.Run("lt ok", func(t *testing.T) {
		ent := struct {
			Value Float `lt:"0"`
		}{Value: NewFloat(-1)}
		entErr := firstError(ReflectCheckErrors(&ent))
		assert.Nil(t, entErr)
	})
	t.Run("lt error", func(t *testing.T) {
		ent := struct {
			Value Float `lt:"0"`
		}{Value: NewFloat(0)}
		entErr := firstError(ReflectCheckErrors(&ent))
		assert.IsType(t, &causes.InvalidFieldError{}, entErr)
	})
	t.Run("lt error", func(t *testing.T) {
		ent := struct {
			Value Float `lt:"0"`
		}{Value: NewFloat(1)}
		entErr := firstError(ReflectCheckErrors(&ent))
		assert.IsType(t, &causes.InvalidFieldError{}, entErr)
	})
	// LTE
	t.Run("lte ok", func(t *testing.T) {
		ent := struct {
			Value Float `lte:"0"`
		}{Value: NewFloat(0)}
		entErr := firstError(ReflectCheckErrors(&ent))
		assert.Nil(t, entErr)
	})
	t.Run("lte error", func(t *testing.T) {
		ent := struct {
			Value Float `lte:"0"`
		}{Value: NewFloat(1)}
		entErr := firstError(ReflectCheckErrors(&ent))
		assert.IsType(t, &causes.InvalidFieldError{}, entErr)
	})
	// Range
	t.Run("range min error", func(t *testing.T) {
		ent := struct {
			Value Float `range:"0,"`
		}{Value: NewFloat(-123)}
		entErr := firstError(ReflectCheckErrors(&ent))
		assert.IsType(t, &causes.InvalidFieldError{}, entErr)
	})
	t.Run("range min ok", func(t *testing.T) {
		ent := struct {
			Value Float `range:"0,"`
		}{Value: NewFloat(123)}
		entErr := firstError(ReflectCheckErrors(&ent))
		assert.Nil(t, entErr)
	})
	t.Run("range max error", func(t *testing.T) {
		ent := struct {
			Value Float `range:",10"`
		}{Value: NewFloat(123)}
		entErr := firstError(ReflectCheckErrors(&ent))
		assert.IsType(t, &causes.InvalidFieldError{}, entErr)
	})
	t.Run("range max ok", func(t *testing.T) {
		ent := struct {
			Value Float `range:",10"`
		}{Value: NewFloat(5)}
		entErr := firstError(ReflectCheckErrors(&ent))
		assert.Nil(t, entErr)
	})
	t.Run("range min,max error", func(t *testing.T) {
		ent := struct {
			Value Float `range:"0,10"`
		}{Value: NewFloat(-123)}
		entErr := firstError(ReflectCheckErrors(&ent))
		assert.IsType(t, &causes.InvalidFieldError{}, entErr)
	})
	t.Run("range max ok", func(t *testing.T) {
		ent := struct {
			Value Float `range:"0,10"`
		}{Value: NewFloat(5)}
		entErr := firstError(ReflectCheckErrors(&ent))
		assert.Nil(t, entErr)
	})

}

func firstError(v []error) error {
	if len(v) > 0 {
		return v[0]
	}
	return nil
}

// An alias is an additional name for a field on the load path. It must not make
// the field look like a second field here: the value is checked once, and a
// reference is remapped once, under the name the field is written as.
func TestReflect_AliasIsNotASecondField(t *testing.T) {
	t.Run("value is checked once", func(t *testing.T) {
		ent := struct {
			Value Timezone `csv:"tz,alias=legacy_tz"`
		}{Value: NewTimezone("Not/AZone")}
		errs := ReflectCheckErrors(&ent)
		assert.Len(t, errs, 1, "expected one error, got %v", errs)
	})
	t.Run("tag-based check runs once", func(t *testing.T) {
		ent := struct {
			Value Int `csv:"mode,alias=legacy_mode" enum:"1,2,3"`
		}{Value: NewInt(9)}
		errs := ReflectCheckErrors(&ent)
		assert.Len(t, errs, 1, "expected one error, got %v", errs)
	})
	t.Run("reference is remapped once", func(t *testing.T) {
		ent := struct {
			Ref String `csv:"ref,alias=old_ref" target:"stops.txt"`
		}{Ref: NewString("src1")}
		emap := NewEntityMap()
		emap.Set("stops.txt", "src1", "db1")
		errs := ReflectUpdateKeys(emap, &ent)
		assert.Empty(t, errs, "expected no errors, got %v", errs)
		assert.Equal(t, "db1", ent.Ref.Val)
	})
}

func TestReflectCheckWarnings(t *testing.T) {
	t.Run("bad value is a warning, not an error", func(t *testing.T) {
		ent := struct {
			Value Language `csv:",warn"`
		}{Value: NewLanguage("xyz")}
		assert.Nil(t, firstError(ReflectCheckErrors(&ent)))
		assert.IsType(t, &causes.InvalidFieldError{}, firstError(ReflectCheckWarnings(&ent)))
	})
	t.Run("good value warns about nothing", func(t *testing.T) {
		ent := struct {
			Value Language `csv:",warn"`
		}{Value: NewLanguage("cnr")}
		assert.Nil(t, firstError(ReflectCheckErrors(&ent)))
		assert.Nil(t, firstError(ReflectCheckWarnings(&ent)))
	})
	t.Run("untagged field keeps reporting an error", func(t *testing.T) {
		ent := struct {
			Value Language
		}{Value: NewLanguage("xyz")}
		assert.IsType(t, &causes.InvalidFieldError{}, firstError(ReflectCheckErrors(&ent)))
		assert.Nil(t, firstError(ReflectCheckWarnings(&ent)))
	})
	// The tag downgrades how a bad value is judged, not whether a required
	// field has to be present at all.
	t.Run("required and warn still errors when absent", func(t *testing.T) {
		ent := struct {
			Value Language `csv:",required,warn"`
		}{}
		assert.IsType(t, &causes.RequiredFieldError{}, firstError(ReflectCheckErrors(&ent)))
		assert.Nil(t, firstError(ReflectCheckWarnings(&ent)))
	})
	t.Run("required and warn warns when malformed", func(t *testing.T) {
		ent := struct {
			Value Language `csv:",required,warn"`
		}{Value: NewLanguage("xyz")}
		assert.Nil(t, firstError(ReflectCheckErrors(&ent)))
		assert.IsType(t, &causes.InvalidFieldError{}, firstError(ReflectCheckWarnings(&ent)))
	})
	// The tag covers every check on the field's value, not only the one the
	// type makes of itself. Splitting them would report a malformed value as a
	// warning and an out of range one as an error, on the same field.
	t.Run("enum value is a warning", func(t *testing.T) {
		ent := struct {
			Value Int `csv:",warn" enum:"0,1,2"`
		}{Value: NewInt(123)}
		assert.Nil(t, firstError(ReflectCheckErrors(&ent)))
		assert.IsType(t, &causes.InvalidFieldError{}, firstError(ReflectCheckWarnings(&ent)))
	})
	t.Run("range value is a warning", func(t *testing.T) {
		ent := struct {
			Value Float `csv:",warn" range:"0,10"`
		}{Value: NewFloat(-123)}
		assert.Nil(t, firstError(ReflectCheckErrors(&ent)))
		assert.IsType(t, &causes.InvalidFieldError{}, firstError(ReflectCheckWarnings(&ent)))
	})
	// A field the tag cannot act on is a mistake in the struct, not a warning
	// about the data, so it stays an error and it is not silently dropped.
	t.Run("warn on an uncheckable type reports the mistake", func(t *testing.T) {
		ent := struct {
			Value string `csv:",warn"`
		}{Value: "anything"}
		assert.ErrorContains(t, firstError(ReflectCheckErrors(&ent)), "does not support reflect based error checks")
	})
	// The same goes for a range or enum tag the field's type cannot convert
	// for. Letting "warn" carry those into the warning pass would file a
	// developer's mistake in the feed's report, once per entity, under a
	// nameless error type.
	t.Run("range tag on an unsuitable type is an error, not a warning", func(t *testing.T) {
		ent := struct {
			Value Language `csv:",warn" range:"0,10"`
		}{Value: NewLanguage("en")}
		assert.ErrorContains(t, firstError(ReflectCheckErrors(&ent)), "could not convert")
		assert.Empty(t, ReflectCheckWarnings(&ent))
	})
	t.Run("enum tag on an unsuitable type is an error, not a warning", func(t *testing.T) {
		ent := struct {
			Value Language `csv:",warn" enum:"0,1"`
		}{Value: NewLanguage("en")}
		assert.ErrorContains(t, firstError(ReflectCheckErrors(&ent)), "could not convert")
		assert.Empty(t, ReflectCheckWarnings(&ent))
	})
}

// Hand rolling Errors() opts an entity out of the reflect based error checks.
// It does not opt the entity out of having its warn tagged fields checked at
// all: CheckWarnings is the only place those are reported.
type ownErrorsEntity struct {
	Value Language `csv:",warn"`
}

func (ent *ownErrorsEntity) Errors() []error { return nil }

// Nor does it opt the field out of the checks that stay errors: whether a
// required value is there at all, and whether the field's type can act on the
// tag it was given.
type ownErrorsRequiredEntity struct {
	Value Language `csv:",required,warn"`
}

func (ent *ownErrorsRequiredEntity) Errors() []error { return nil }

type ownErrorsBadTagEntity struct {
	Value string `csv:",warn"`
}

func (ent *ownErrorsBadTagEntity) Errors() []error { return nil }

func TestCheckWarnings(t *testing.T) {
	t.Run("entity with its own Errors is still checked", func(t *testing.T) {
		ent := &ownErrorsEntity{Value: NewLanguage("xyz")}
		assert.Nil(t, firstError(CheckErrors(ent)))
		assert.IsType(t, &causes.InvalidFieldError{}, firstError(CheckWarnings(ent)))
	})
	t.Run("entity with its own Errors still reports an absent required value", func(t *testing.T) {
		ent := &ownErrorsRequiredEntity{}
		assert.IsType(t, &causes.RequiredFieldError{}, firstError(CheckErrors(ent)))
		assert.Nil(t, firstError(CheckWarnings(ent)))
	})
	t.Run("entity with its own Errors still reports a tag its type cannot act on", func(t *testing.T) {
		ent := &ownErrorsBadTagEntity{Value: "anything"}
		assert.ErrorContains(t, firstError(CheckErrors(ent)), "does not support reflect based error checks")
		assert.Nil(t, firstError(CheckWarnings(ent)))
	})
	// ReflectCheckWarnings is exported and takes any; a caller that hands it
	// something it cannot address gets nothing back rather than a panic.
	//
	// CheckWarnings is tested through it rather than directly, because it is
	// not nil safe and does not claim to be: every gtfs entity embeds
	// tt.BaseEntity, so a typed nil panics in that entity's own LoadWarnings
	// before it ever reaches here, exactly as it does today.
	t.Run("value is not a panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			assert.Nil(t, firstError(ReflectCheckWarnings(ownErrorsEntity{Value: NewLanguage("xyz")})))
		})
	})
	t.Run("nil is not a panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			assert.Nil(t, firstError(ReflectCheckWarnings(nil)))
		})
	})
	t.Run("typed nil is not a panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			assert.Nil(t, firstError(ReflectCheckWarnings((*ownErrorsEntity)(nil))))
		})
	})
}
