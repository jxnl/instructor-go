package core

import (
	"encoding/json"
	"testing"
)

// TestUnionFlowSimulation simulates the full flow of union type handling
func TestUnionFlowSimulation(t *testing.T) {
	// Define tool types
	type SearchTool struct {
		Type  string `json:"type" jsonschema:"const=search,description=Type of tool"`
		Query string `json:"query" jsonschema:"description=Search query to execute"`
	}

	type LookupTool struct {
		Type    string `json:"type" jsonschema:"const=lookup,description=Type of tool"`
		Keyword string `json:"keyword" jsonschema:"description=Keyword to look up"`
	}

	type FinishTool struct {
		Type   string `json:"type" jsonschema:"const=finish,description=Type of tool"`
		Answer string `json:"answer" jsonschema:"description=Final answer"`
	}

	// Step 1: Create union schema
	unionSchema, err := NewUnionSchema("type", SearchTool{}, LookupTool{}, FinishTool{})
	if err != nil {
		t.Fatalf("NewUnionSchema() error: %v", err)
	}

	// Step 2: Generate function schemas (what we send to OpenAI)
	functions := unionSchema.ToFunctionSchema()
	if len(functions) != 3 {
		t.Fatalf("expected 3 functions, got %d", len(functions))
	}

	t.Logf("Generated %d functions for OpenAI:", len(functions))
	for _, fn := range functions {
		t.Logf("  - Function: %s", fn.Name)
	}

	// Step 3: Simulate OpenAI response (LLM chose "search" function)
	// OpenAI returns tool call with function name and arguments
	functionName := "search"
	functionArgs := `{"query": "Go programming language interfaces"}`

	t.Logf("\nSimulating OpenAI response:")
	t.Logf("  Function called: %s", functionName)
	t.Logf("  Arguments: %s", functionArgs)

	// Step 4: Inject discriminator (what our provider code does)
	var argMap map[string]any
	if err := json.Unmarshal([]byte(functionArgs), &argMap); err != nil {
		t.Fatalf("failed to parse arguments: %v", err)
	}

	// Check if discriminator already exists
	if _, exists := argMap["type"]; !exists {
		// Inject it
		argMap["type"] = functionName
		t.Logf("\nInjecting discriminator: type=%s", functionName)
	}

	modifiedArgs, err := json.Marshal(argMap)
	if err != nil {
		t.Fatalf("failed to marshal modified arguments: %v", err)
	}

	t.Logf("Modified arguments: %s", string(modifiedArgs))

	// Step 5: Unmarshal using union schema (what ChatHandlerUnion does)
	results, err := unionSchema.Unmarshal(modifiedArgs)
	if err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}

	// Step 6: Check we got results
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// Step 7: Type switch on result (what user code does)
	result := results[0]
	t.Logf("\nResult type: %T", result)

	switch tool := result.(type) {
	case SearchTool:
		t.Logf("✓ Successfully extracted SearchTool:")
		t.Logf("  Type: %s", tool.Type)
		t.Logf("  Query: %s", tool.Query)

		if tool.Type != "search" {
			t.Errorf("expected Type=%q, got %q", "search", tool.Type)
		}
		if tool.Query != "Go programming language interfaces" {
			t.Errorf("expected Query=%q, got %q", "Go programming language interfaces", tool.Query)
		}

	case LookupTool:
		t.Fatalf("unexpected LookupTool, expected SearchTool")
	case FinishTool:
		t.Fatalf("unexpected FinishTool, expected SearchTool")
	default:
		t.Fatalf("unexpected type: %T", result)
	}

	// Test another variant (LookupTool)
	t.Log("\n--- Testing second variant ---")

	functionName2 := "lookup"
	functionArgs2 := `{"keyword": "interfaces"}`

	var argMap2 map[string]any
	json.Unmarshal([]byte(functionArgs2), &argMap2)
	argMap2["type"] = functionName2
	modifiedArgs2, _ := json.Marshal(argMap2)

	results2, err := unionSchema.Unmarshal(modifiedArgs2)
	if err != nil {
		t.Fatalf("Unmarshal() error for lookup: %v", err)
	}

	if len(results2) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results2))
	}

	if lookupTool, ok := results2[0].(LookupTool); ok {
		t.Logf("✓ Successfully extracted LookupTool:")
		t.Logf("  Type: %s", lookupTool.Type)
		t.Logf("  Keyword: %s", lookupTool.Keyword)
	} else {
		t.Fatalf("expected LookupTool, got %T", results2[0])
	}

	// Test finish tool
	t.Log("\n--- Testing third variant ---")

	functionName3 := "finish"
	functionArgs3 := `{"answer": "Go has powerful interface system"}`

	var argMap3 map[string]any
	json.Unmarshal([]byte(functionArgs3), &argMap3)
	argMap3["type"] = functionName3
	modifiedArgs3, _ := json.Marshal(argMap3)

	results3, err := unionSchema.Unmarshal(modifiedArgs3)
	if err != nil {
		t.Fatalf("Unmarshal() error for finish: %v", err)
	}

	if len(results3) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results3))
	}

	if finishTool, ok := results3[0].(FinishTool); ok {
		t.Logf("✓ Successfully extracted FinishTool:")
		t.Logf("  Type: %s", finishTool.Type)
		t.Logf("  Answer: %s", finishTool.Answer)
	} else {
		t.Fatalf("expected FinishTool, got %T", results3[0])
	}
}
