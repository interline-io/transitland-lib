// fgagen converts an OpenFGA DSL file to the JSON form consumed by the
// OpenFGA WriteAuthorizationModel API.
//
// Usage: fgagen <input.model> <output.json>
package main

import (
	"fmt"
	"os"

	"github.com/openfga/language/pkg/go/transformer"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <input.model> <output.json>\n", os.Args[0])
		os.Exit(2)
	}
	in, out := os.Args[1], os.Args[2]

	dsl, err := os.ReadFile(in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", in, err)
		os.Exit(1)
	}

	model, err := transformer.TransformDSLToProto(string(dsl))
	if err != nil {
		fmt.Fprintf(os.Stderr, "transform DSL: %v\n", err)
		os.Exit(1)
	}

	pretty, err := protojson.MarshalOptions{
		EmitUnpopulated: true,
		Indent:          "    ",
	}.Marshal(model)
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal: %v\n", err)
		os.Exit(1)
	}
	pretty = append(pretty, '\n')

	if err := os.WriteFile(out, pretty, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", out, err)
		os.Exit(1)
	}
}
