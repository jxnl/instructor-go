package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/invopop/jsonschema"
)

// UnionOptions configures union type extraction
type UnionOptions struct {
	Discriminator string // Field name used for discrimination (e.g. "type")
	Variants      []any  // Concrete instances of each variant type
}

// UnionSchema represents a discriminated union
type UnionSchema struct {
	Discriminator string
	Variants      []reflect.Type
	VariantMap    map[string]reflect.Type // discriminator value -> type
	Schema        *jsonschema.Schema
}

// UnionValidationError wraps validation errors with context
type UnionValidationError struct {
	DiscriminatorValue string
	ValidValues        []string
	Err                error
}

func (e *UnionValidationError) Error() string {
	return fmt.Sprintf(
		"invalid discriminator value %q, valid values: %v (error: %v)",
		e.DiscriminatorValue,
		e.ValidValues,
		e.Err,
	)
}

// NewUnionSchema creates a union schema from variants
func NewUnionSchema(discriminator string, variants ...any) (*UnionSchema, error) {
	if discriminator == "" {
		return nil, fmt.Errorf("discriminator field name cannot be empty")
	}

	if len(variants) == 0 {
		return nil, fmt.Errorf("at least one variant is required")
	}

	u := &UnionSchema{
		Discriminator: discriminator,
		Variants:      make([]reflect.Type, 0, len(variants)),
		VariantMap:    make(map[string]reflect.Type),
	}

	// Extract types and build discriminator mapping
	for _, variant := range variants {
		t := reflect.TypeOf(variant)

		// Ensure we're working with the base type (not pointer)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}

		if t.Kind() != reflect.Struct {
			return nil, fmt.Errorf("variant must be a struct type, got %s", t.Kind())
		}

		u.Variants = append(u.Variants, t)

		// Extract discriminator value from struct
		discValue, err := extractDiscriminatorValue(t, discriminator)
		if err != nil {
			return nil, fmt.Errorf("error extracting discriminator from %s: %w", t.Name(), err)
		}

		// Check for duplicate discriminator values
		if existingType, exists := u.VariantMap[discValue]; exists {
			return nil, fmt.Errorf(
				"duplicate discriminator value %q: both %s and %s have this value",
				discValue,
				existingType.Name(),
				t.Name(),
			)
		}

		u.VariantMap[discValue] = t
	}

	// Generate JSON schema for the union
	schema, err := generateUnionSchema(u)
	if err != nil {
		return nil, fmt.Errorf("error generating union schema: %w", err)
	}
	u.Schema = schema

	return u, nil
}

// extractDiscriminatorValue extracts the discriminator value from a struct type
func extractDiscriminatorValue(t reflect.Type, discriminatorField string) (string, error) {
	// Find the discriminator field
	field, found := t.FieldByName(discriminatorField)
	if !found {
		// Try to find by json tag
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			jsonTag := f.Tag.Get("json")
			if jsonTag != "" {
				jsonName := strings.Split(jsonTag, ",")[0]
				if jsonName == discriminatorField {
					field = f
					found = true
					break
				}
			}
		}
	}

	if !found {
		return "", fmt.Errorf("discriminator field %q not found in struct", discriminatorField)
	}

	// Check field type is string
	if field.Type.Kind() != reflect.String {
		return "", fmt.Errorf("discriminator field %q must be of type string, got %s", discriminatorField, field.Type.Kind())
	}

	// Extract value from jsonschema tag
	jsonschemaTag := field.Tag.Get("jsonschema")
	if jsonschemaTag != "" {
		// Parse jsonschema tag for const= or enum=
		parts := strings.Split(jsonschemaTag, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)

			// Check for const=value
			if strings.HasPrefix(part, "const=") {
				return strings.TrimPrefix(part, "const="), nil
			}

			// Check for enum=value (single value)
			if strings.HasPrefix(part, "enum=") {
				return strings.TrimPrefix(part, "enum="), nil
			}
		}
	}

	// Fallback: use struct name converted to snake_case
	return toSnakeCase(t.Name()), nil
}

// toSnakeCase converts a string to snake_case
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// generateUnionSchema generates a JSON schema for the union
func generateUnionSchema(u *UnionSchema) (*jsonschema.Schema, error) {
	schema := &jsonschema.Schema{
		Type:  "object",
		OneOf: make([]*jsonschema.Schema, 0, len(u.Variants)),
	}

	// Build definitions and oneOf
	definitions := make(map[string]*jsonschema.Schema)

	for _, variantType := range u.VariantMap {
		// Generate schema for this variant
		variantSchema := jsonschema.ReflectFromType(variantType)

		// Add to definitions
		defName := variantType.Name()
		definitions[defName] = variantSchema

		// Add to oneOf with reference
		ref := fmt.Sprintf("#/$defs/%s", defName)
		schema.OneOf = append(schema.OneOf, &jsonschema.Schema{
			Ref: ref,
		})
	}

	// Merge all definitions
	if len(definitions) > 0 {
		schema.Definitions = definitions
	}

	// Note: discriminator field mapping is stored in u.VariantMap and used during unmarshaling
	// The JSON schema spec supports discriminator but the Go library doesn't expose it,
	// so we handle it at runtime during Unmarshal

	return schema, nil
}

// Unmarshal unmarshals JSON into the correct variant type
// Always returns []any containing one or more variant instances
// Supports both single objects and arrays of objects from the LLM
func (u *UnionSchema) Unmarshal(data []byte) ([]any, error) {
	// First, determine if we have an array or single object
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, fmt.Errorf("empty JSON data")
	}

	// Check if it's an array
	if data[0] == '[' {
		return u.unmarshalMany(data)
	}

	// Single object - wrap in slice
	result, err := u.unmarshalSingle(data)
	if err != nil {
		return nil, err
	}
	return []any{result}, nil
}

// unmarshalSingle unmarshals a single JSON object into the correct variant type
func (u *UnionSchema) unmarshalSingle(data []byte) (any, error) {
	// First, peek at the discriminator field
	var peek map[string]any
	if err := json.Unmarshal(data, &peek); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Get discriminator value
	discValueRaw, exists := peek[u.Discriminator]
	if !exists {
		validValues := make([]string, 0, len(u.VariantMap))
		for k := range u.VariantMap {
			validValues = append(validValues, k)
		}
		return nil, &UnionValidationError{
			DiscriminatorValue: "",
			ValidValues:        validValues,
			Err:                fmt.Errorf("discriminator field %q not found in JSON", u.Discriminator),
		}
	}

	discValue, ok := discValueRaw.(string)
	if !ok {
		return nil, fmt.Errorf("discriminator field %q must be a string, got %T", u.Discriminator, discValueRaw)
	}

	// Look up the variant type
	variantType, exists := u.VariantMap[discValue]
	if !exists {
		validValues := make([]string, 0, len(u.VariantMap))
		for k := range u.VariantMap {
			validValues = append(validValues, k)
		}
		return nil, &UnionValidationError{
			DiscriminatorValue: discValue,
			ValidValues:        validValues,
			Err:                fmt.Errorf("unknown discriminator value"),
		}
	}

	// Create a new instance of the variant type
	result := reflect.New(variantType).Interface()

	// Unmarshal into the concrete type
	if err := json.Unmarshal(data, result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal into %s: %w", variantType.Name(), err)
	}

	// Return the dereferenced value (not pointer)
	return reflect.ValueOf(result).Elem().Interface(), nil
}

// unmarshalMany unmarshals an array of JSON objects into a slice of variant types
func (u *UnionSchema) unmarshalMany(data []byte) ([]any, error) {
	// Parse as array of raw JSON objects
	var rawObjects []json.RawMessage
	if err := json.Unmarshal(data, &rawObjects); err != nil {
		return nil, fmt.Errorf("failed to parse JSON array: %w", err)
	}

	if len(rawObjects) == 0 {
		return []any{}, nil
	}

	// Unmarshal each object
	results := make([]any, 0, len(rawObjects))
	for i, rawObj := range rawObjects {
		result, err := u.unmarshalSingle(rawObj)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal array element %d: %w", i, err)
		}
		results = append(results, result)
	}

	return results, nil
}

// ToFunctionSchema generates function definitions for union (for tool call mode)
// Returns one function per variant since OpenAI doesn't support oneOf at top level
func (u *UnionSchema) ToFunctionSchema() []FunctionDefinition {
	funcs := make([]FunctionDefinition, 0, len(u.VariantMap))

	// Create a separate function for each variant
	for discValue, variantType := range u.VariantMap {
		// Get the schema for this specific variant
		variantSchema := jsonschema.ReflectFromType(variantType)

		// Extract the actual object schema from the nested structure
		var parameters *jsonschema.Schema
		if variantSchema.Ref != "" {
			// If there's a $ref, get it from definitions
			refName := variantType.Name()
			if def, ok := variantSchema.Definitions[refName]; ok {
				parameters = def
			} else {
				parameters = variantSchema
			}
		} else {
			parameters = variantSchema
		}

		// Create function definition
		fd := FunctionDefinition{
			Name:        discValue,
			Description: fmt.Sprintf("Execute %s action", discValue),
			Parameters:  parameters,
		}

		funcs = append(funcs, fd)
	}

	return funcs
}

// ValidValues returns all valid discriminator values
func (u *UnionSchema) ValidValues() []string {
	values := make([]string, 0, len(u.VariantMap))
	for k := range u.VariantMap {
		values = append(values, k)
	}
	return values
}
