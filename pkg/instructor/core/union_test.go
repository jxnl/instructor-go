package core

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// Test variant types
type SearchTool struct {
	Type  string `json:"type" jsonschema:"const=search"`
	Query string `json:"query" jsonschema:"description=Search query"`
}

type LookupTool struct {
	Type    string `json:"type" jsonschema:"const=lookup"`
	Keyword string `json:"keyword" jsonschema:"description=Keyword to lookup"`
}

type FinishTool struct {
	Type   string `json:"type" jsonschema:"const=finish"`
	Answer string `json:"answer" jsonschema:"description=Final answer"`
}

type InvalidTool struct {
	Name string `json:"name"` // No discriminator field
}

type DuplicateTool struct {
	Type  string `json:"type" jsonschema:"const=search"` // Duplicate value
	Other string `json:"other"`
}

func TestNewUnionSchema(t *testing.T) {
	tests := []struct {
		name          string
		discriminator string
		variants      []any
		wantErr       bool
		errContains   string
	}{
		{
			name:          "valid union with three variants",
			discriminator: "type",
			variants:      []any{SearchTool{}, LookupTool{}, FinishTool{}},
			wantErr:       false,
		},
		{
			name:          "empty discriminator",
			discriminator: "",
			variants:      []any{SearchTool{}},
			wantErr:       true,
			errContains:   "discriminator field name cannot be empty",
		},
		{
			name:          "no variants",
			discriminator: "type",
			variants:      []any{},
			wantErr:       true,
			errContains:   "at least one variant is required",
		},
		{
			name:          "missing discriminator field",
			discriminator: "type",
			variants:      []any{InvalidTool{}},
			wantErr:       true,
			errContains:   "discriminator field",
		},
		{
			name:          "duplicate discriminator value",
			discriminator: "type",
			variants:      []any{SearchTool{}, DuplicateTool{}},
			wantErr:       true,
			errContains:   "duplicate discriminator value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := NewUnionSchema(tt.discriminator, tt.variants...)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewUnionSchema() expected error containing %q, got nil", tt.errContains)
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("NewUnionSchema() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("NewUnionSchema() unexpected error: %v", err)
				return
			}

			if schema == nil {
				t.Error("NewUnionSchema() returned nil schema")
				return
			}

			if schema.Discriminator != tt.discriminator {
				t.Errorf("schema.Discriminator = %q, want %q", schema.Discriminator, tt.discriminator)
			}

			if len(schema.Variants) != len(tt.variants) {
				t.Errorf("len(schema.Variants) = %d, want %d", len(schema.Variants), len(tt.variants))
			}

			if len(schema.VariantMap) != len(tt.variants) {
				t.Errorf("len(schema.VariantMap) = %d, want %d", len(schema.VariantMap), len(tt.variants))
			}
		})
	}
}

func TestUnionSchema_Unmarshal(t *testing.T) {
	schema, err := NewUnionSchema("type", SearchTool{}, LookupTool{}, FinishTool{})
	if err != nil {
		t.Fatalf("NewUnionSchema() error: %v", err)
	}

	tests := []struct {
		name        string
		json        string
		wantType    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "unmarshal SearchTool",
			json:     `{"type": "search", "query": "golang unions"}`,
			wantType: "SearchTool",
			wantErr:  false,
		},
		{
			name:     "unmarshal LookupTool",
			json:     `{"type": "lookup", "keyword": "interface"}`,
			wantType: "LookupTool",
			wantErr:  false,
		},
		{
			name:     "unmarshal FinishTool",
			json:     `{"type": "finish", "answer": "done"}`,
			wantType: "FinishTool",
			wantErr:  false,
		},
		{
			name:        "invalid JSON",
			json:        `{invalid}`,
			wantErr:     true,
			errContains: "failed to parse JSON",
		},
		{
			name:        "missing discriminator field",
			json:        `{"query": "test"}`,
			wantErr:     true,
			errContains: "discriminator field",
		},
		{
			name:        "invalid discriminator value",
			json:        `{"type": "invalid", "query": "test"}`,
			wantErr:     true,
			errContains: "unknown discriminator value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := schema.Unmarshal([]byte(tt.json))

			if tt.wantErr {
				if err == nil {
					t.Errorf("Unmarshal() expected error containing %q, got nil", tt.errContains)
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("Unmarshal() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Unmarshal() unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("Unmarshal() returned nil result")
				return
			}

			typeName := getTypeName(result)
			if typeName != tt.wantType {
				t.Errorf("Unmarshal() returned type %s, want %s", typeName, tt.wantType)
			}

			// Verify fields
			switch v := result.(type) {
			case SearchTool:
				if v.Type != "search" {
					t.Errorf("SearchTool.Type = %q, want %q", v.Type, "search")
				}
				if v.Query == "" {
					t.Error("SearchTool.Query is empty")
				}
			case LookupTool:
				if v.Type != "lookup" {
					t.Errorf("LookupTool.Type = %q, want %q", v.Type, "lookup")
				}
				if v.Keyword == "" {
					t.Error("LookupTool.Keyword is empty")
				}
			case FinishTool:
				if v.Type != "finish" {
					t.Errorf("FinishTool.Type = %q, want %q", v.Type, "finish")
				}
				if v.Answer == "" {
					t.Error("FinishTool.Answer is empty")
				}
			}
		})
	}
}

func TestExtractDiscriminatorValue(t *testing.T) {
	tests := []struct {
		name          string
		structType    any
		discriminator string
		want          string
		wantErr       bool
	}{
		{
			name:          "const tag",
			structType:    SearchTool{},
			discriminator: "type",
			want:          "search",
			wantErr:       false,
		},
		{
			name:          "lookup const tag",
			structType:    LookupTool{},
			discriminator: "type",
			want:          "lookup",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typ := getReflectType(tt.structType)
			got, err := extractDiscriminatorValue(typ, tt.discriminator)

			if tt.wantErr {
				if err == nil {
					t.Error("extractDiscriminatorValue() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("extractDiscriminatorValue() unexpected error: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("extractDiscriminatorValue() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"SearchTool", "search_tool"},
		{"LookupTool", "lookup_tool"},
		{"HTTPServer", "h_t_t_p_server"},
		{"myTool", "my_tool"},
		{"tool", "tool"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toSnakeCase(tt.input)
			if got != tt.want {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestUnionSchema_ValidValues(t *testing.T) {
	schema, err := NewUnionSchema("type", SearchTool{}, LookupTool{}, FinishTool{})
	if err != nil {
		t.Fatalf("NewUnionSchema() error: %v", err)
	}

	validValues := schema.ValidValues()
	if len(validValues) != 3 {
		t.Errorf("ValidValues() returned %d values, want 3", len(validValues))
	}

	expected := map[string]bool{"search": true, "lookup": true, "finish": true}
	for _, v := range validValues {
		if !expected[v] {
			t.Errorf("ValidValues() contains unexpected value %q", v)
		}
		delete(expected, v)
	}

	if len(expected) > 0 {
		t.Errorf("ValidValues() missing values: %v", expected)
	}
}

func TestUnionSchema_ToFunctionSchema(t *testing.T) {
	schema, err := NewUnionSchema("type", SearchTool{}, LookupTool{}, FinishTool{})
	if err != nil {
		t.Fatalf("NewUnionSchema() error: %v", err)
	}

	funcs := schema.ToFunctionSchema()
	if len(funcs) != 3 {
		t.Errorf("ToFunctionSchema() returned %d functions, want 3", len(funcs))
	}

	// Check that we have all expected function names (discriminator values)
	expectedNames := map[string]bool{"search": true, "lookup": true, "finish": true}
	for _, fn := range funcs {
		if !expectedNames[fn.Name] {
			t.Errorf("unexpected function name: %q", fn.Name)
		}
		if fn.Parameters == nil {
			t.Errorf("function %q has nil parameters", fn.Name)
		}
		if fn.Parameters.Type != "object" {
			t.Errorf("function %q parameters type = %q, want %q", fn.Name, fn.Parameters.Type, "object")
		}
		delete(expectedNames, fn.Name)
	}

	if len(expectedNames) > 0 {
		t.Errorf("missing function names: %v", expectedNames)
	}
}

func TestGenerateUnionSchema(t *testing.T) {
	unionSchema, err := NewUnionSchema("type", SearchTool{}, LookupTool{}, FinishTool{})
	if err != nil {
		t.Fatalf("NewUnionSchema() error: %v", err)
	}

	schema := unionSchema.Schema
	if schema == nil {
		t.Fatal("generated schema is nil")
	}

	if len(schema.OneOf) != 3 {
		t.Errorf("schema.OneOf length = %d, want 3", len(schema.OneOf))
	}

	// Check that definitions exist
	if len(schema.Definitions) != 3 {
		t.Errorf("schema.Definitions length = %d, want 3", len(schema.Definitions))
	}

	// Verify discriminator info is stored in UnionSchema
	if unionSchema.Discriminator != "type" {
		t.Errorf("unionSchema.Discriminator = %q, want %q", unionSchema.Discriminator, "type")
	}

	if len(unionSchema.VariantMap) != 3 {
		t.Errorf("len(unionSchema.VariantMap) = %d, want 3", len(unionSchema.VariantMap))
	}
}

func TestUnionValidationError(t *testing.T) {
	err := &UnionValidationError{
		DiscriminatorValue: "invalid",
		ValidValues:        []string{"search", "lookup", "finish"},
		Err:                fmt.Errorf("test error"),
	}

	errMsg := err.Error()
	if !contains(errMsg, "invalid") {
		t.Errorf("error message should contain discriminator value, got: %s", errMsg)
	}
	if !contains(errMsg, "search") {
		t.Errorf("error message should contain valid values, got: %s", errMsg)
	}
}

// Helper functions

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func getTypeName(v any) string {
	t := getReflectType(v)
	return t.Name()
}

func getReflectType(v any) reflect.Type {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}
