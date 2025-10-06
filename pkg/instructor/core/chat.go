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

// defaultAppendErrorToRequest is the default implementation for OpenAI-style string content
// Providers can override this by implementing AppendErrorToRequest
func defaultAppendErrorToRequest(request interface{}, failedResponse string, errorMessage string) interface{} {
	// Try to extract messages using reflection
	v := reflect.ValueOf(request)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Look for a Messages field
	messagesField := v.FieldByName("Messages")
	if !messagesField.IsValid() {
		// If we can't find Messages field, return original request
		return request
	}

	// Check if Messages is a slice
	if messagesField.Kind() != reflect.Slice {
		return request
	}

	// Get the element type of the slice
	messageType := messagesField.Type().Elem()

	// Create new assistant message with the failed response
	assistantMsg := reflect.New(messageType).Elem()

	// Set Role field
	roleField := assistantMsg.FieldByName("Role")
	if roleField.IsValid() && roleField.CanSet() {
		if roleField.Kind() == reflect.String {
			roleField.SetString("assistant")
		}
	}

	// Set Content field (default: string content like OpenAI)
	contentField := assistantMsg.FieldByName("Content")
	if contentField.IsValid() && contentField.CanSet() {
		if contentField.Kind() == reflect.String {
			contentField.SetString(failedResponse)
		}
	}

	// Create new user message with the error
	userMsg := reflect.New(messageType).Elem()

	// Set Role field
	roleField = userMsg.FieldByName("Role")
	if roleField.IsValid() && roleField.CanSet() {
		if roleField.Kind() == reflect.String {
			roleField.SetString("user")
		}
	}

	// Set Content field
	contentField = userMsg.FieldByName("Content")
	if contentField.IsValid() && contentField.CanSet() {
		if contentField.Kind() == reflect.String {
			contentField.SetString(errorMessage)
		}
	}

	// Append the messages
	newMessages := reflect.Append(messagesField, assistantMsg)
	newMessages = reflect.Append(newMessages, userMsg)

	// Create a copy of the request with the new messages
	newRequest := reflect.New(v.Type()).Elem()
	newRequest.Set(v)
	newRequest.FieldByName("Messages").Set(newMessages)

	// Return as interface - if original was pointer, return pointer
	if reflect.ValueOf(request).Kind() == reflect.Ptr {
		return newRequest.Addr().Interface()
	}
	return newRequest.Interface()
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
			i.CountUsageFromResponse(resp, usage)

			// If we have more retries left, send back the error and the malformed JSON
			if attempt < i.MaxRetries() {
				errorMessage := fmt.Sprintf("JSON parsing failed: %s. Fix the syntax and retry.", err.Error())
				// Try provider-specific handler first, fall back to default
				if customRequest := i.AppendErrorToRequest(request, text, errorMessage); customRequest != nil {
					request = customRequest
				} else {
					request = defaultAppendErrorToRequest(request, text, errorMessage)
				}
			}
			continue
		}

		if i.Validate() {
			validate = validator.New()
			// Validate the response structure against the defined model using the validator
			err = validate.Struct(response)

			if err != nil {
				i.CountUsageFromResponse(resp, usage)

				// If we have more retries left, send back the validation error
				if attempt < i.MaxRetries() {
					errorMessage := fmt.Sprintf("Validation failed: %s. Fix the values and retry.", err.Error())
					// Try provider-specific handler first, fall back to default
					if customRequest := i.AppendErrorToRequest(request, text, errorMessage); customRequest != nil {
						request = customRequest
					} else {
						request = defaultAppendErrorToRequest(request, text, errorMessage)
					}
				}
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
			i.CountUsageFromResponse(resp, usage)

			// If we have more retries left, send back the error and the malformed JSON
			if attempt < i.MaxRetries() {
				errorMessage := fmt.Sprintf("Union type error: %s. Ensure response matches one of the expected variants.", err.Error())
				// Try provider-specific handler first, fall back to default
				if customRequest := i.AppendErrorToRequest(request, text, errorMessage); customRequest != nil {
					request = customRequest
				} else {
					request = defaultAppendErrorToRequest(request, text, errorMessage)
				}
			}
			continue
		}

		if i.Validate() {
			validate = validator.New()
			// Validate the concrete variant structure
			err = validate.Struct(result)

			if err != nil {
				i.CountUsageFromResponse(resp, usage)

				// If we have more retries left, send back the validation error
				if attempt < i.MaxRetries() {
					errorMessage := fmt.Sprintf("Validation failed: %s. Fix the values and retry.", err.Error())
					// Try provider-specific handler first, fall back to default
					if customRequest := i.AppendErrorToRequest(request, text, errorMessage); customRequest != nil {
						request = customRequest
					} else {
						request = defaultAppendErrorToRequest(request, text, errorMessage)
					}
				}
				continue
			}
		}

		// Add usage and return both result and response
		respWithUsage, err := i.AddUsageSumToResponse(resp, usage)
		return result, respWithUsage, err
	}

	return nil, i.EmptyResponseWithUsageSum(usage), errors.New("hit max retry attempts")
}
