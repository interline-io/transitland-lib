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
