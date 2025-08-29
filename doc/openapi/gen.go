//go:generate go run . rest.json
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	oa "github.com/getkin/kin-openapi/openapi3"
	"github.com/interline-io/transitland-lib/server/rest"
)

func main() {
	ctx := context.Background()
	args := os.Args
	if len(args) != 2 {
		exit(errors.New("output file required"))
	}
	outfile := args[1]

	// Generate OpenAPI schema
	outdoc, err := rest.GenerateOpenAPI("/rest")
	if err != nil {
		exit(err)
	}

	// Validate output
	jj, err := json.MarshalIndent(outdoc, "", "  ")
	if err != nil {
		exit(err)
	}

	schema, err := oa.NewLoader().LoadFromData(jj)
	if err != nil {
		exit(err)
	}
	var validationOpts []oa.ValidationOption
	if err := schema.Validate(ctx, validationOpts...); err != nil {
		exit(err)
	}

	// After validation, write to file
	outf, err := os.Create(outfile)
	if err != nil {
		exit(err)
	}
	outf.Write(jj)
}

func exit(err error) {
	fmt.Println("Error: ", err.Error())
	os.Exit(1)
}
