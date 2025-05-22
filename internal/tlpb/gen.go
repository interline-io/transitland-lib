package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/bufbuild/protocompile"
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func main() {
	cmd := tlcli.CobraHelper(&GenGtfsCommand{}, "", "")
	cmd.Execute()
}

type GenGtfsCommand struct {
	Protopath string
	Outpath   string
	Command   *cobra.Command
}

func (cmd *GenGtfsCommand) AddFlags(fl *pflag.FlagSet) {
}

func (cmd *GenGtfsCommand) HelpDesc() (string, string) {
	return "Generate GTFS entities", ""
}

func (cmd *GenGtfsCommand) Parse(args []string) error {
	fl := tlcli.NewNArgs(args)
	if fl.NArg() < 2 {
		return errors.New("<proto> <outppath>")
	}
	cmd.Protopath = fl.Arg(0)
	cmd.Outpath = fl.Arg(1)
	return nil
}

func (cmd *GenGtfsCommand) Run() error {
	compiler := protocompile.Compiler{
		Resolver: &protocompile.SourceResolver{},
	}
	files, err := compiler.Compile(context.Background(), cmd.Protopath)
	if err != nil {
		return err
	}
	outf, err := os.Create(cmd.Outpath)
	if err != nil {
		return err
	}
	defer outf.Close()

	// Go
	outf.WriteString(`package gtfs` + "\n\n")
	outf.WriteString(`import ( "github.com/interline-io/transitland-lib/tt" )` + "\n\n")

	ttKinds := map[string]string{
		"Url":       "tt.Url",
		"Date":      "tt.Date",
		"Time":      "tt.Time",
		"Color":     "tt.Color",
		"Key":       "tt.Key",
		"Phone":     "tt.Phone",
		"Email":     "tt.Email",
		"Reference": "tt.Reference",
		"Currency":  "tt.Currency",
		"Language":  "tt.Language",
		"Int":       "tt.Int",
		"Bool":      "tt.Bool",
		"Float":     "tt.Float",
		"String":    "tt.String",
		"Timezone":  "tt.Timezone",
		"Timestamp": "tt.Timestamp",
		"Seconds":   "tt.Seconds",
	}

	for _, lf := range files {
		enums := lf.Enums()
		for i := 0; i < enums.Len(); i++ {
			en := enums.Get(i)
			outf.WriteString(fmt.Sprintf("type %s int32\n\n", en.Name()))
		}
		msgs := lf.Messages()
		for i := 0; i < msgs.Len(); i++ {
			msg := msgs.Get(i)
			fields := msg.Fields()
			if _, ok := ttKinds[string(msg.Name())]; ok {
				continue
			}
			if fields.Len() == 1 && fields.Get(0).Name() == "val" {
				field := fields.Get(0)
				outf.WriteString(fmt.Sprintf(
					"type %s struct { tt.Option[%s] }\n\n",
					msg.Name(),
					mapKind(field)),
				)
				continue
			}

			outf.WriteString(fmt.Sprintf("type %s struct {\n", msg.Name()))
			for j := 0; j < fields.Len(); j++ {
				field := fields.Get(j)
				fieldName := toCamelCase(string(field.Name()))
				fieldKind := mapKind(field)
				if ttKind, ok := ttKinds[fieldKind]; ok {
					outf.WriteString(fmt.Sprintf("\t%s %s\n", fieldName, ttKind))
					continue
				}
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
	return nil
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
