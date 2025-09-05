package core

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/invopop/jsonschema"
)

type Schema struct {
	*jsonschema.Schema
	String string

	Functions []FunctionDefinition
}

type Function struct {
	Type     string             `json:"type"`
	Function FunctionDefinition `json:"function"`
}

type FunctionDefinition struct {
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Parameters  *jsonschema.Schema `json:"parameters"`
}

func NewSchema(t reflect.Type) (*Schema, error) {

	schema := jsonschema.ReflectFromType(t)

	str, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return nil, err
	}

	funcs := ToFunctionSchema(t, schema)

	s := &Schema{
		Schema: schema,
		String: string(str),

		Functions: funcs,
	}

	return s, nil
}

func ToFunctionSchema(tType reflect.Type, tSchema *jsonschema.Schema) []FunctionDefinition {

	fds := []FunctionDefinition{}

	for name, def := range tSchema.Definitions {

		parameters := &jsonschema.Schema{
			Type:       "object",
			Properties: def.Properties,
			Required:   def.Required,
		}

		fd := FunctionDefinition{
			Name:        name,
			Description: def.Description,
			Parameters:  parameters,
		}

		fds = append(fds, fd)
	}

	return fds
}

func (s *Schema) NameFromRef() string {
	return strings.Split(s.Ref, "/")[2] // ex: '#/$defs/MyStruct'
}
