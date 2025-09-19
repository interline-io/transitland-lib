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
