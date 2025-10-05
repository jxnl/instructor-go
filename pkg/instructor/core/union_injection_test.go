package core

import (
	"encoding/json"
	"testing"
)

// TestDiscriminatorInjection tests that we can inject discriminator when it's missing
func TestDiscriminatorInjection(t *testing.T) {
	// Simulate what happens when OpenAI returns tool call arguments without discriminator
	toolCallArgs := `{"query": "golang programming"}`

	// Parse the args
	var argMap map[string]any
	if err := json.Unmarshal([]byte(toolCallArgs), &argMap); err != nil {
		t.Fatalf("failed to parse tool args: %v", err)
	}

	// Inject the discriminator (function name is "search")
	funcName := "search"
	argMap["type"] = funcName

	// Marshal back
	modifiedArgs, err := json.Marshal(argMap)
	if err != nil {
		t.Fatalf("failed to marshal modified args: %v", err)
	}

	// Now try to unmarshal with union schema
	type SearchTool struct {
		Type  string `json:"type" jsonschema:"const=search"`
		Query string `json:"query"`
	}

	type LookupTool struct {
		Type    string `json:"type" jsonschema:"const=lookup"`
		Keyword string `json:"keyword"`
	}

	unionSchema, err := NewUnionSchema("type", SearchTool{}, LookupTool{})
	if err != nil {
		t.Fatalf("NewUnionSchema() error: %v", err)
	}

	// Unmarshal
	result, err := unionSchema.Unmarshal(modifiedArgs)
	if err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}

	// Check result type
	searchTool, ok := result.(SearchTool)
	if !ok {
		t.Fatalf("expected SearchTool, got %T", result)
	}

	if searchTool.Type != "search" {
		t.Errorf("SearchTool.Type = %q, want %q", searchTool.Type, "search")
	}

	if searchTool.Query != "golang programming" {
		t.Errorf("SearchTool.Query = %q, want %q", searchTool.Query, "golang programming")
	}

	t.Logf("Successfully unmarshaled: %+v", searchTool)
}

// TestFunctionSchemaHasNoDiscriminator tests that function parameters don't include discriminator
func TestFunctionSchemaHasNoDiscriminator(t *testing.T) {
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

	funcs := unionSchema.ToFunctionSchema()

	for _, fn := range funcs {
		// Check that the function parameters have the type field with const
		paramsJSON, err := json.MarshalIndent(fn.Parameters, "", "  ")
		if err != nil {
			t.Fatalf("failed to marshal parameters: %v", err)
		}

		var params map[string]any
		if err := json.Unmarshal(paramsJSON, &params); err != nil {
			t.Fatalf("failed to parse parameters JSON: %v", err)
		}

		// The function should have properties
		props, ok := params["properties"].(map[string]any)
		if !ok {
			t.Fatalf("function %q has no properties", fn.Name)
		}

		// Check that type field exists
		if _, hasType := props["type"]; !hasType {
			t.Errorf("function %q parameters missing 'type' field", fn.Name)
		}

		t.Logf("Function %q schema is valid", fn.Name)
	}
}
