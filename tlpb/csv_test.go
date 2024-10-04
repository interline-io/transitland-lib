package tlpb

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/bufbuild/protocompile"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestReadStopsPB(t *testing.T) {
	ReadStopsPB(testutil.RelPath("test/data/external/bart.zip"))
}

func BenchmarkReadStopsPB(b *testing.B) {
	for n := 0; n < b.N; n++ {
		ReadStopsPB(testutil.RelPath("test/data/external/bart.zip"))
	}
}

func BenchmarkReadStopsTT(b *testing.B) {
	for n := 0; n < b.N; n++ {
		ReadStopsTT(testutil.RelPath("test/data/external/bart.zip"))
	}
}

func TestCodegen(t *testing.T) {
	compiler := protocompile.Compiler{
		Resolver: &protocompile.SourceResolver{},
	}
	files, err := compiler.Compile(context.Background(), "gtfs.proto")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("package out")

	for _, lf := range files {
		// fmt.Printf("file %#v\n", file)
		enums := lf.Enums()
		for i := 0; i < enums.Len(); i++ {
			en := enums.Get(i)
			fmt.Printf("type %s int32\n\n", en.Name())
		}
		msgs := lf.Messages()
		for i := 0; i < msgs.Len(); i++ {
			msg := msgs.Get(i)
			fields := msg.Fields()
			if fields.Len() == 1 && fields.Get(0).Name() == "val" {
				field := fields.Get(0)
				fmt.Printf("type %s Option[%s]\n\n", msg.Name(), mapKind(field))
				continue
			}

			fmt.Printf("type %s struct {\n", msg.Name())
			for j := 0; j < fields.Len(); j++ {
				field := fields.Get(j)
				fieldName := toCamelCase(string(field.Name()))
				fieldKind := mapKind(field)
				fmt.Printf("\t%s %s\n", fieldName, fieldKind)
				// fmt.Printf("\t%s %s `json:\"%s\"`\n", fieldName, fieldKind, field.Name())
			}
			fmt.Println("}")
			fmt.Println("")

		}
	}
}

func mapKind(field protoreflect.FieldDescriptor) string {
	fieldKind := field.Kind().String()
	switch fieldKind {
	case "enum":
		fieldKind = string(field.Enum().Name())
	case "double":
		fieldKind = "float64"
	case "float":
		fieldKind = "float32"
	}
	if fmsg := field.Message(); fmsg != nil {
		fieldKind = string(fmsg.Name())
	}
	return fieldKind
}

func toCamelCase(v string) string {
	a := strings.Split(v, "_")
	for i := 0; i < len(a); i++ {
		s := a[i]
		if s == "id" {
			s = "ID"
		} else {
			s = strings.ToUpper(s[0:1]) + s[1:]
		}
		a[i] = s
	}
	return strings.Join(a, "")
}
