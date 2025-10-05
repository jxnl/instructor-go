package core

import (
	"encoding/json"
	"testing"
)

// TestUnionSchemaStructure tests that the generated schema has the correct structure
func TestUnionSchemaStructure(t *testing.T) {
	type Tool1 struct {
		Type string `json:"type" jsonschema:"const=tool1"`
		Arg1 string `json:"arg1"`
	}

	type Tool2 struct {
		Type string `json:"type" jsonschema:"const=tool2"`
		Arg2 int    `json:"arg2"`
	}

	unionSchema, err := NewUnionSchema("type", Tool1{}, Tool2{})
	if err != nil {
		t.Fatalf("NewUnionSchema() error: %v", err)
	}

	schema := unionSchema.Schema

	// Check that type is "object"
	if schema.Type != "object" {
		t.Errorf("schema.Type = %q, want %q", schema.Type, "object")
	}

	// Check that oneOf is present
	if len(schema.OneOf) != 2 {
		t.Errorf("len(schema.OneOf) = %d, want 2", len(schema.OneOf))
	}

	// Check that definitions are present
	if len(schema.Definitions) != 2 {
		t.Errorf("len(schema.Definitions) = %d, want 2", len(schema.Definitions))
	}

	// Verify the schema can be marshaled to JSON
	schemaJSON, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal schema to JSON: %v", err)
	}

	// Parse it back to verify it's valid JSON
	var parsedSchema map[string]any
	if err := json.Unmarshal(schemaJSON, &parsedSchema); err != nil {
		t.Fatalf("failed to parse generated schema JSON: %v", err)
	}

	// Verify type field
	if parsedSchema["type"] != "object" {
		t.Errorf("parsed schema type = %v, want %q", parsedSchema["type"], "object")
	}

	// Verify oneOf field exists
	if _, ok := parsedSchema["oneOf"]; !ok {
		t.Error("parsed schema missing oneOf field")
	}

	// Verify $defs field exists
	if _, ok := parsedSchema["$defs"]; !ok {
		t.Error("parsed schema missing $defs field")
	}

	t.Logf("Generated schema:\n%s", string(schemaJSON))
}

// TestUnionSchemaForFunctionCall tests that the schema works for function calling
func TestUnionSchemaForFunctionCall(t *testing.T) {
	type SearchTool struct {
		Type  string `json:"type" jsonschema:"const=search"`
		Query string `json:"query" jsonschema:"description=Search query"`
	}

	type LookupTool struct {
		Type    string `json:"type" jsonschema:"const=lookup"`
		Keyword string `json:"keyword" jsonschema:"description=Keyword to lookup"`
	}

	unionSchema, err := NewUnionSchema("type", SearchTool{}, LookupTool{})
	if err != nil {
		t.Fatalf("NewUnionSchema() error: %v", err)
	}

	// Get function definitions - should be one per variant
	funcs := unionSchema.ToFunctionSchema()
	if len(funcs) != 2 {
		t.Fatalf("expected 2 function definitions, got %d", len(funcs))
	}

	// Check each function
	expectedFuncs := map[string]bool{"search": true, "lookup": true}
	for _, funcDef := range funcs {
		if !expectedFuncs[funcDef.Name] {
			t.Errorf("unexpected function name: %q", funcDef.Name)
		}

		if funcDef.Parameters == nil {
			t.Fatalf("function %q has nil parameters", funcDef.Name)
		}

		// Parameters should have type "object"
		if funcDef.Parameters.Type != "object" {
			t.Errorf("function %q parameters type = %q, want %q", funcDef.Name, funcDef.Parameters.Type, "object")
		}

		// Marshal to JSON to verify structure
		paramsJSON, err := json.MarshalIndent(funcDef.Parameters, "", "  ")
		if err != nil {
			t.Fatalf("failed to marshal %q parameters to JSON: %v", funcDef.Name, err)
		}

		t.Logf("Function %q parameters:\n%s", funcDef.Name, string(paramsJSON))
		delete(expectedFuncs, funcDef.Name)
	}

	if len(expectedFuncs) > 0 {
		t.Errorf("missing expected functions: %v", expectedFuncs)
	}
}
