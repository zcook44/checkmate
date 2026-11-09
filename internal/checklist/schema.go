package checklist

import (
	"embed"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Official DISA STIG Viewer 3 CKLB 1.0 schema, published 2024-01-10.
// Source: https://dl.dod.cyber.mil/wp-content/uploads/stigs/zip/SV3_CKLB_1_0_JSON_SCHEMA.zip
//
//go:embed schema/SV3_CKLB_1_0_JSON_SCHEMA.json
var schemaFS embed.FS
var schemaOnce sync.Once
var compiled *jsonschema.Schema
var schemaErr error

func ValidateSchema(data []byte) error {
	schemaOnce.Do(func() {
		b, err := schemaFS.ReadFile("schema/SV3_CKLB_1_0_JSON_SCHEMA.json")
		if err != nil {
			schemaErr = err
			return
		}
		var doc any
		if err = json.Unmarshal(b, &doc); err != nil {
			schemaErr = err
			return
		}
		c := jsonschema.NewCompiler()
		if err = c.AddResource("cklb.json", doc); err != nil {
			schemaErr = err
			return
		}
		compiled, schemaErr = c.Compile("cklb.json")
	})
	if schemaErr != nil {
		return fmt.Errorf("compile bundled CKLB 1.0 schema: %w", schemaErr)
	}
	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("invalid CKLB JSON: %w", err)
	}
	if err := compiled.Validate(doc); err != nil {
		return fmt.Errorf("CKLB 1.0 schema: %w", err)
	}
	return nil
}
