package core

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

type StreamWrapper[T any] struct {
	Items []T `json:"items"`
}

const WRAPPER_END = `"items": [`

func ChatStreamHandler(i Instructor, ctx context.Context, request interface{}, response any) (<-chan interface{}, error) {

	responseType := reflect.TypeOf(response)

	streamWrapperType := reflect.StructOf([]reflect.StructField{
		{
			Name:      "Items",
			Type:      reflect.SliceOf(responseType),
			Tag:       `json:"items"`,
			Anonymous: false,
		},
	})

	schema, err := NewSchema(streamWrapperType)
	if err != nil {
		return nil, err
	}

	ch, err := i.InternalChatStream(ctx, request, schema)
	if err != nil {
		return nil, err
	}

	shouldValidate := i.Validate()
	if shouldValidate {
		validate = validator.New()
	}

	parsedChan := parseStream(ctx, ch, shouldValidate, responseType)

	return parsedChan, nil
}

func parseStream(ctx context.Context, ch <-chan string, shouldValidate bool, responseType reflect.Type) <-chan interface{} {

	parsedChan := make(chan any)

	go func() {
		defer close(parsedChan)

		buffer := new(strings.Builder)
		inArray := false

		for {
			select {
			case <-ctx.Done():
				return
			case text, ok := <-ch:
				if !ok {
					// Stream closed
					processRemainingBuffer(buffer, parsedChan, shouldValidate, responseType)
					return
				}

				buffer.WriteString(text)

				// Eat all input until elements stream starts
				if !inArray {
					inArray = startArray(buffer)
				}

				processBuffer(buffer, parsedChan, shouldValidate, responseType)
			}
		}
	}()

	return parsedChan
}

func startArray(buffer *strings.Builder) bool {

	data := buffer.String()

	idx := strings.Index(data, WRAPPER_END)
	if idx == -1 {
		return false
	}

	trimmed := strings.TrimSpace(data[idx+len(WRAPPER_END):])
	buffer.Reset()
	buffer.WriteString(trimmed)

	return true
}

func processBuffer(buffer *strings.Builder, parsedChan chan<- interface{}, shouldValidate bool, responseType reflect.Type) {

	data := buffer.String()

	data, remaining := getFirstFullJSONElement(&data)

	decoder := json.NewDecoder(strings.NewReader(data))

	for decoder.More() {
		instance := reflect.New(responseType).Interface()
		err := decoder.Decode(instance)
		if err != nil {
			break
		}

		if shouldValidate {
			// Validate the instance
			err = validate.Struct(instance)
			if err != nil {
				break
			}
		}

		parsedChan <- instance

		buffer.Reset()
		buffer.WriteString(remaining)
	}
}

func processRemainingBuffer(buffer *strings.Builder, parsedChan chan<- interface{}, shouldValidate bool, responseType reflect.Type) {

	data := buffer.String()

	data = ExtractJSON(&data)

	if idx := strings.LastIndex(data, "]"); idx != -1 {
		data = data[:idx]
	}

	processBuffer(buffer, parsedChan, shouldValidate, responseType)

}
