package tlpb

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/bufbuild/protocompile"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestCodegen(t *testing.T) {
	compiler := protocompile.Compiler{
		Resolver: &protocompile.SourceResolver{},
	}
	files, err := compiler.Compile(context.Background(), "gtfs.proto")
	if err != nil {
		t.Fatal(err)
	}
	outf, err := os.Create("gtfs/gtfs.go")
	if err != nil {
		t.Fatal(err)
	}
	defer outf.Close()

	outf.WriteString("package gtfs\n\n")
	outf.WriteString("type EnumValue int32\n\n")

	for _, lf := range files {
		// fmt.Printf("file %#v\n", file)
		enums := lf.Enums()
		for i := 0; i < enums.Len(); i++ {
			en := enums.Get(i)
			outf.WriteString(fmt.Sprintf("type %s int32\n\n", en.Name()))
		}
		msgs := lf.Messages()
		for i := 0; i < msgs.Len(); i++ {
			msg := msgs.Get(i)
			fields := msg.Fields()
			if fields.Len() == 1 && fields.Get(0).Name() == "val" {
				field := fields.Get(0)
				outf.WriteString(fmt.Sprintf("type %s struct { Option[%s] }\n\n", msg.Name(), mapKind(field)))
				continue
			}

			outf.WriteString(fmt.Sprintf("type %s struct {\n", msg.Name()))
			for j := 0; j < fields.Len(); j++ {
				field := fields.Get(j)
				fieldName := toCamelCase(string(field.Name()))
				fieldKind := mapKind(field)
				switch fieldKind {
				case "DatabaseEntity":
					outf.WriteString("\tDatabaseEntity\n")
				default:
					outf.WriteString(fmt.Sprintf("\t%s %s\n", fieldName, fieldKind))
				}

			}
			outf.WriteString("}\n\n")
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
