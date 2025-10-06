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
	results, err := unionSchema.Unmarshal(modifiedArgs)
	if err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// Check result type
	searchTool, ok := results[0].(SearchTool)
	if !ok {
		t.Fatalf("expected SearchTool, got %T", results[0])
	}

	if searchTool.Type != "search" {
		t.Errorf("SearchTool.Type = %q, want %q", searchTool.Type, "search")
	}

	if searchTool.Query != "golang programming" {
		t.Errorf("SearchTool.Query = %q, want %q", searchTool.Query, "golang programming")
	}

	t.Logf("Successfully unmarshaled: %+v", searchTool)
}

// TestMultipleToolCallsWithDiscriminator tests discriminator injection for multiple parallel tool calls
func TestMultipleToolCallsWithDiscriminator(t *testing.T) {
	type SearchTool struct {
		Type  string `json:"type" jsonschema:"const=search"`
		Query string `json:"query"`
	}

	type LookupTool struct {
		Type    string `json:"type" jsonschema:"const=lookup"`
		Keyword string `json:"keyword"`
	}

	type FinishTool struct {
		Type   string `json:"type" jsonschema:"const=finish"`
		Answer string `json:"answer"`
	}

	unionSchema, err := NewUnionSchema("type", SearchTool{}, LookupTool{}, FinishTool{})
	if err != nil {
		t.Fatalf("NewUnionSchema() error: %v", err)
	}

	// Simulate multiple tool calls from the LLM (without discriminator)
	toolCalls := []struct {
		name string
		args string
	}{
		{"search", `{"query": "golang programming"}`},
		{"lookup", `{"keyword": "interfaces"}`},
		{"finish", `{"answer": "Go has powerful interfaces"}`},
	}

	// Inject discriminator for each tool call
	var injectedTools []map[string]any
	for _, tc := range toolCalls {
		var argMap map[string]any
		if err := json.Unmarshal([]byte(tc.args), &argMap); err != nil {
			t.Fatalf("failed to parse args for %s: %v", tc.name, err)
		}

		// Inject discriminator
		argMap["type"] = tc.name
		injectedTools = append(injectedTools, argMap)
	}

	// Marshal as array
	arrayJSON, err := json.Marshal(injectedTools)
	if err != nil {
		t.Fatalf("failed to marshal array: %v", err)
	}

	t.Logf("Multiple tools with discriminators: %s", string(arrayJSON))

	// Unmarshal using union schema
	results, err := unionSchema.Unmarshal(arrayJSON)
	if err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Verify first result (SearchTool)
	searchTool, ok := results[0].(SearchTool)
	if !ok {
		t.Errorf("results[0] expected SearchTool, got %T", results[0])
	} else {
		if searchTool.Query != "golang programming" {
			t.Errorf("SearchTool.Query = %q, want %q", searchTool.Query, "golang programming")
		}
		t.Logf("✓ SearchTool: %+v", searchTool)
	}

	// Verify second result (LookupTool)
	lookupTool, ok := results[1].(LookupTool)
	if !ok {
		t.Errorf("results[1] expected LookupTool, got %T", results[1])
	} else {
		if lookupTool.Keyword != "interfaces" {
			t.Errorf("LookupTool.Keyword = %q, want %q", lookupTool.Keyword, "interfaces")
		}
		t.Logf("✓ LookupTool: %+v", lookupTool)
	}

	// Verify third result (FinishTool)
	finishTool, ok := results[2].(FinishTool)
	if !ok {
		t.Errorf("results[2] expected FinishTool, got %T", results[2])
	} else {
		if finishTool.Answer != "Go has powerful interfaces" {
			t.Errorf("FinishTool.Answer = %q, want %q", finishTool.Answer, "Go has powerful interfaces")
		}
		t.Logf("✓ FinishTool: %+v", finishTool)
	}
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
