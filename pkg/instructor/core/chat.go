package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/go-playground/validator/v10"
)

type UsageSum struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

func ChatHandler(i Instructor, ctx context.Context, request interface{}, response any) (interface{}, error) {

	var err error

	t := reflect.TypeOf(response)

	schema, err := NewSchema(t)
	if err != nil {
		return nil, err
	}

	// keep a running total of usage
	usage := &UsageSum{}

	for attempt := 0; attempt <= i.MaxRetries(); attempt++ {

		text, resp, err := i.InternalChat(ctx, request, schema)
		if err != nil {
			// no retry on non-marshalling/validation errors
			return i.EmptyResponseWithResponseUsage(resp), err
		}

		text = ExtractJSON(&text)

		err = json.Unmarshal([]byte(text), &response)
		if err != nil {
			// TODO:
			// add more sophisticated retry logic (send back json and parse error for model to fix).
			//
			// Currently, its just recalling with no new information
			// or attempt to fix the error with the last generated JSON

			i.CountUsageFromResponse(resp, usage)
			continue
		}

		if i.Validate() {
			validate = validator.New()
			// Validate the response structure against the defined model using the validator
			err = validate.Struct(response)

			if err != nil {
				// TODO:
				// add more sophisticated retry logic (send back validator error and parse error for model to fix).

				i.CountUsageFromResponse(resp, usage)
				continue
			}
		}

		return i.AddUsageSumToResponse(resp, usage)
	}

	return i.EmptyResponseWithUsageSum(usage), errors.New("hit max retry attempts")
}

// ChatHandlerUnion handles chat completion with union type extraction
func ChatHandlerUnion(i Instructor, ctx context.Context, request interface{}, opts UnionOptions) (any, interface{}, error) {

	// Create union schema from options
	unionSchema, err := NewUnionSchema(opts.Discriminator, opts.Variants...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create union schema: %w", err)
	}

	// Convert union schema to regular schema for InternalChat
	schema := &Schema{
		Schema:    unionSchema.Schema,
		Functions: unionSchema.ToFunctionSchema(),
	}

	// Serialize schema for String field (used in some modes)
	schemaBytes, err := json.MarshalIndent(unionSchema.Schema, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal union schema: %w", err)
	}
	schema.String = string(schemaBytes)

	// keep a running total of usage
	usage := &UsageSum{}

	for attempt := 0; attempt <= i.MaxRetries(); attempt++ {

		text, resp, err := i.InternalChat(ctx, request, schema)
		if err != nil {
			// no retry on non-marshalling/validation errors
			return nil, i.EmptyResponseWithResponseUsage(resp), err
		}

		text = ExtractJSON(&text)

		// Use union schema to unmarshal into correct variant
		result, err := unionSchema.Unmarshal([]byte(text))
		if err != nil {
			// Retry on union validation/unmarshal errors
			i.CountUsageFromResponse(resp, usage)
			continue
		}

		if i.Validate() {
			validate = validator.New()
			// Validate the concrete variant structure
			err = validate.Struct(result)

			if err != nil {
				i.CountUsageFromResponse(resp, usage)
				continue
			}
		}

		// Add usage and return both result and response
		respWithUsage, err := i.AddUsageSumToResponse(resp, usage)
		return result, respWithUsage, err
	}

	return nil, i.EmptyResponseWithUsageSum(usage), errors.New("hit max retry attempts")
}
