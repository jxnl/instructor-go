package core

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
)

// Test structs
type TestPerson struct {
	Name string `json:"name" validate:"required,min=2"`
	Age  int    `json:"age" validate:"required,min=0,max=150"`
}

// Mock instructor for testing
type mockInstructor struct {
	maxRetries     int
	validate       bool
	responses      []string // Queue of responses to return
	responseIndex  int
	mode           Mode
	provider       Provider
	requestHistory []interface{} // Track all requests made
}

func (m *mockInstructor) Provider() Provider {
	return m.provider
}

func (m *mockInstructor) Mode() Mode {
	return m.mode
}

func (m *mockInstructor) MaxRetries() int {
	return m.maxRetries
}

func (m *mockInstructor) Validate() bool {
	return m.validate
}

func (m *mockInstructor) InternalChat(ctx context.Context, request interface{}, schema *Schema) (string, interface{}, error) {
	// Store the request for inspection
	m.requestHistory = append(m.requestHistory, request)

	if m.responseIndex >= len(m.responses) {
		return "", nil, errors.New("no more mock responses")
	}

	response := m.responses[m.responseIndex]
	m.responseIndex++

	// Return mock response with empty usage
	mockResp := &mockResponse{
		InputTokens:  10,
		OutputTokens: 20,
		TotalTokens:  30,
	}

	return response, mockResp, nil
}

func (m *mockInstructor) InternalChatStream(ctx context.Context, request interface{}, schema *Schema) (<-chan string, error) {
	return nil, errors.New("not implemented")
}

func (m *mockInstructor) EmptyResponseWithUsageSum(usage *UsageSum) interface{} {
	return &mockResponse{
		InputTokens:  usage.InputTokens,
		OutputTokens: usage.OutputTokens,
		TotalTokens:  usage.TotalTokens,
	}
}

func (m *mockInstructor) EmptyResponseWithResponseUsage(response interface{}) interface{} {
	if response == nil {
		return &mockResponse{}
	}
	if mr, ok := response.(*mockResponse); ok {
		return &mockResponse{
			InputTokens:  mr.InputTokens,
			OutputTokens: mr.OutputTokens,
			TotalTokens:  mr.TotalTokens,
		}
	}
	return &mockResponse{}
}

func (m *mockInstructor) AddUsageSumToResponse(response interface{}, usage *UsageSum) (interface{}, error) {
	if mr, ok := response.(*mockResponse); ok {
		mr.InputTokens += usage.InputTokens
		mr.OutputTokens += usage.OutputTokens
		mr.TotalTokens += usage.TotalTokens
		return mr, nil
	}
	return response, nil
}

func (m *mockInstructor) CountUsageFromResponse(response interface{}, usage *UsageSum) *UsageSum {
	if mr, ok := response.(*mockResponse); ok {
		usage.InputTokens += mr.InputTokens
		usage.OutputTokens += mr.OutputTokens
		usage.TotalTokens += mr.TotalTokens
	}
	return usage
}

// AppendErrorToRequest returns nil to use the default handler
func (m *mockInstructor) AppendErrorToRequest(request interface{}, failedResponse string, errorMessage string) interface{} {
	return nil // Use default handler
}

// Logger returns a no-op logger for testing
func (m *mockInstructor) Logger() Logger {
	return NewNoopLogger()
}

type mockResponse struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

// Mock request type that mimics openai.ChatCompletionRequest
type mockRequest struct {
	Model    string
	Messages []mockMessage
}

type mockMessage struct {
	Role    string
	Content string
}

func TestChatHandler_JSONUnmarshalRetry(t *testing.T) {
	tests := []struct {
		name           string
		responses      []string
		maxRetries     int
		expectError    bool
		expectRetries  int
		validateResult func(t *testing.T, result TestPerson)
	}{
		{
			name: "First response has invalid JSON, second succeeds",
			responses: []string{
				`{"name": "John", "age": }`,   // Invalid JSON
				`{"name": "John", "age": 30}`, // Valid JSON
			},
			maxRetries:    2,
			expectError:   false,
			expectRetries: 2,
			validateResult: func(t *testing.T, result TestPerson) {
				if result.Name != "John" || result.Age != 30 {
					t.Errorf("Expected John, 30, got %s, %d", result.Name, result.Age)
				}
			},
		},
		{
			name: "All responses have invalid JSON",
			responses: []string{
				`{"name": "John", "age": }`,    // Invalid JSON
				`{"name": "Jane", "age": abc}`, // Invalid JSON
				`{"name": "Bob", "age": }`,     // Invalid JSON
			},
			maxRetries:    2,
			expectError:   true,
			expectRetries: 3,
		},
		{
			name: "First response succeeds immediately",
			responses: []string{
				`{"name": "Alice", "age": 25}`,
			},
			maxRetries:    2,
			expectError:   false,
			expectRetries: 1,
			validateResult: func(t *testing.T, result TestPerson) {
				if result.Name != "Alice" || result.Age != 25 {
					t.Errorf("Expected Alice, 25, got %s, %d", result.Name, result.Age)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockInstructor{
				maxRetries: tt.maxRetries,
				validate:   false,
				responses:  tt.responses,
				mode:       ModeJSON,
				provider:   ProviderOpenAI,
			}

			var result TestPerson
			request := mockRequest{
				Model: "gpt-4",
				Messages: []mockMessage{
					{Role: "user", Content: "Generate a person"},
				},
			}

			_, err := ChatHandler(mock, context.Background(), request, &result)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if len(mock.requestHistory) != tt.expectRetries {
				t.Errorf("Expected %d retries, got %d", tt.expectRetries, len(mock.requestHistory))
			}

			// Verify error message was appended on retry
			if len(mock.requestHistory) > 1 {
				secondReq, ok := mock.requestHistory[1].(mockRequest)
				if !ok {
					t.Fatal("Request is not mockRequest type")
				}

				// Should have original message + assistant message + error message
				if len(secondReq.Messages) != 3 {
					t.Errorf("Expected 3 messages on retry, got %d", len(secondReq.Messages))
				}

				// Check that error message contains helpful information
				if len(secondReq.Messages) >= 3 {
					errorMsg := secondReq.Messages[2].Content
					if errorMsg == "" {
						t.Error("Error message should not be empty")
					}
					// Should mention JSON parsing failed
					if len(errorMsg) > 0 && !strings.Contains(errorMsg, "JSON parsing failed") {
						t.Error("Error message should mention JSON parsing failed")
					}
				}
			}

			if !tt.expectError && tt.validateResult != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

func TestChatHandler_ValidationRetry(t *testing.T) {
	tests := []struct {
		name           string
		responses      []string
		maxRetries     int
		expectError    bool
		expectRetries  int
		validateResult func(t *testing.T, result TestPerson)
	}{
		{
			name: "First response fails validation, second succeeds",
			responses: []string{
				`{"name": "J", "age": 30}`,    // Name too short (min=2)
				`{"name": "John", "age": 30}`, // Valid
			},
			maxRetries:    2,
			expectError:   false,
			expectRetries: 2,
			validateResult: func(t *testing.T, result TestPerson) {
				if result.Name != "John" || result.Age != 30 {
					t.Errorf("Expected John, 30, got %s, %d", result.Name, result.Age)
				}
			},
		},
		{
			name: "Age validation fails then succeeds",
			responses: []string{
				`{"name": "John", "age": 200}`, // Age > 150
				`{"name": "John", "age": 30}`,  // Valid
			},
			maxRetries:    2,
			expectError:   false,
			expectRetries: 2,
			validateResult: func(t *testing.T, result TestPerson) {
				if result.Name != "John" || result.Age != 30 {
					t.Errorf("Expected John, 30, got %s, %d", result.Name, result.Age)
				}
			},
		},
		{
			name: "All responses fail validation",
			responses: []string{
				`{"name": "J", "age": 30}`,   // Name too short
				`{"name": "Jo", "age": 200}`, // Age too high
				`{"name": "", "age": 30}`,    // Name empty
			},
			maxRetries:    2,
			expectError:   true,
			expectRetries: 3,
		},
		{
			name: "Missing required field",
			responses: []string{
				`{"name": "John"}`,            // Missing age
				`{"name": "John", "age": 30}`, // Valid
			},
			maxRetries:    2,
			expectError:   false,
			expectRetries: 2,
			validateResult: func(t *testing.T, result TestPerson) {
				if result.Name != "John" || result.Age != 30 {
					t.Errorf("Expected John, 30, got %s, %d", result.Name, result.Age)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockInstructor{
				maxRetries: tt.maxRetries,
				validate:   true, // Enable validation
				responses:  tt.responses,
				mode:       ModeJSON,
				provider:   ProviderOpenAI,
			}

			var result TestPerson
			request := mockRequest{
				Model: "gpt-4",
				Messages: []mockMessage{
					{Role: "user", Content: "Generate a person"},
				},
			}

			_, err := ChatHandler(mock, context.Background(), request, &result)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if len(mock.requestHistory) != tt.expectRetries {
				t.Errorf("Expected %d retries, got %d", tt.expectRetries, len(mock.requestHistory))
			}

			// Verify error message was appended on retry
			if len(mock.requestHistory) > 1 {
				secondReq, ok := mock.requestHistory[1].(mockRequest)
				if !ok {
					t.Fatal("Request is not mockRequest type")
				}

				// Should have original message + assistant message + error message
				if len(secondReq.Messages) != 3 {
					t.Errorf("Expected 3 messages on retry, got %d", len(secondReq.Messages))
				}

				// Check that error message contains helpful information
				if len(secondReq.Messages) >= 3 {
					errorMsg := secondReq.Messages[2].Content
					if errorMsg == "" {
						t.Error("Error message should not be empty")
					}
					// Should mention validation failed
					if len(errorMsg) > 0 && !strings.Contains(errorMsg, "Validation failed") {
						t.Error("Error message should mention Validation failed")
					}
				}
			}

			if !tt.expectError && tt.validateResult != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

func TestChatHandler_MixedErrors(t *testing.T) {
	// Test case where we have JSON error followed by validation error
	mock := &mockInstructor{
		maxRetries: 3,
		validate:   true,
		responses: []string{
			`{"name": "John", "age": }`,   // JSON error
			`{"name": "J", "age": 30}`,    // Validation error (name too short)
			`{"name": "John", "age": 30}`, // Success
		},
		mode:     ModeJSON,
		provider: ProviderOpenAI,
	}

	var result TestPerson
	request := mockRequest{
		Model: "gpt-4",
		Messages: []mockMessage{
			{Role: "user", Content: "Generate a person"},
		},
	}

	_, err := ChatHandler(mock, context.Background(), request, &result)

	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	if len(mock.requestHistory) != 3 {
		t.Errorf("Expected 3 attempts, got %d", len(mock.requestHistory))
	}

	if result.Name != "John" || result.Age != 30 {
		t.Errorf("Expected John, 30, got %s, %d", result.Name, result.Age)
	}

	// Verify second request has JSON error message
	secondReq := mock.requestHistory[1].(mockRequest)
	if len(secondReq.Messages) >= 3 {
		errorMsg := secondReq.Messages[2].Content
		if !strings.Contains(errorMsg, "JSON parsing failed") {
			t.Error("Second request should have JSON parsing failed message")
		}
	}

	// Verify third request has validation error message
	thirdReq := mock.requestHistory[2].(mockRequest)
	if len(thirdReq.Messages) >= 3 {
		errorMsg := thirdReq.Messages[len(thirdReq.Messages)-1].Content
		if !strings.Contains(errorMsg, "Validation failed") {
			t.Error("Third request should have validation failed message")
		}
	}
}

func TestAppendErrorToRequest(t *testing.T) {
	// Test the helper function directly
	request := mockRequest{
		Model: "gpt-4",
		Messages: []mockMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	failedResponse := `{"invalid": json}`
	errorMessage := "This is an error message"

	newRequest := defaultAppendErrorToRequest(request, failedResponse, errorMessage)

	newMockReq, ok := newRequest.(mockRequest)
	if !ok {
		t.Fatal("defaultAppendErrorToRequest did not return mockRequest type")
	}

	// Should have 3 messages: original + assistant + user error
	if len(newMockReq.Messages) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(newMockReq.Messages))
	}

	// Check assistant message
	if newMockReq.Messages[1].Role != "assistant" {
		t.Errorf("Expected second message to be assistant, got %s", newMockReq.Messages[1].Role)
	}
	if newMockReq.Messages[1].Content != failedResponse {
		t.Errorf("Expected assistant content to be failed response")
	}

	// Check user error message
	if newMockReq.Messages[2].Role != "user" {
		t.Errorf("Expected third message to be user, got %s", newMockReq.Messages[2].Role)
	}
	if newMockReq.Messages[2].Content != errorMessage {
		t.Errorf("Expected user content to be error message")
	}
}

func TestChatHandler_NoRetryOnInternalError(t *testing.T) {
	// Test that we don't retry on internal errors (like API errors)
	mock := &mockInstructor{
		maxRetries: 3,
		validate:   false,
		responses:  []string{}, // No responses - will cause InternalChat to error
		mode:       ModeJSON,
		provider:   ProviderOpenAI,
	}

	var result TestPerson
	request := mockRequest{
		Model: "gpt-4",
		Messages: []mockMessage{
			{Role: "user", Content: "Generate a person"},
		},
	}

	_, err := ChatHandler(mock, context.Background(), request, &result)

	if err == nil {
		t.Error("Expected error but got none")
	}

	// Should only try once - no retries on internal errors
	if len(mock.requestHistory) != 1 {
		t.Errorf("Expected 1 attempt (no retries), got %d", len(mock.requestHistory))
	}
}

func TestChatHandler_UsageAccumulation(t *testing.T) {
	// Test that usage is accumulated across retries
	mock := &mockInstructor{
		maxRetries: 2,
		validate:   false,
		responses: []string{
			`{"name": "John", "age": }`,   // Invalid JSON - will retry
			`{"name": "John", "age": 30}`, // Valid
		},
		mode:     ModeJSON,
		provider: ProviderOpenAI,
	}

	var result TestPerson
	request := mockRequest{
		Model: "gpt-4",
		Messages: []mockMessage{
			{Role: "user", Content: "Generate a person"},
		},
	}

	resp, err := ChatHandler(mock, context.Background(), request, &result)

	if err != nil {
		t.Fatalf("Expected no error but got: %v", err)
	}

	mockResp, ok := resp.(*mockResponse)
	if !ok {
		t.Fatal("Response is not mockResponse type")
	}

	// Each mock response has 10 input, 20 output, 30 total tokens
	// We made 2 requests, so we should have accumulated usage
	expectedInput := 20  // 10 * 2
	expectedOutput := 40 // 20 * 2
	expectedTotal := 60  // 30 * 2

	if mockResp.InputTokens != expectedInput {
		t.Errorf("Expected input tokens %d, got %d", expectedInput, mockResp.InputTokens)
	}
	if mockResp.OutputTokens != expectedOutput {
		t.Errorf("Expected output tokens %d, got %d", expectedOutput, mockResp.OutputTokens)
	}
	if mockResp.TotalTokens != expectedTotal {
		t.Errorf("Expected total tokens %d, got %d", expectedTotal, mockResp.TotalTokens)
	}
}

// Helper function - uses strings.Contains from the standard library
// Note: contains is already defined in union_test.go, but we can't use it from there
// in this file without making it exported, so we define a local helper that uses strings.Contains

// Test validation initialization
func TestValidationInitialization(t *testing.T) {
	// Ensure validator is properly initialized when validate is called
	type TestStruct struct {
		Email string `json:"email" validate:"required,email"`
	}

	mock := &mockInstructor{
		maxRetries: 1,
		validate:   true,
		responses: []string{
			`{"email": "not-an-email"}`,     // Invalid email
			`{"email": "test@example.com"}`, // Valid email
		},
		mode:     ModeJSON,
		provider: ProviderOpenAI,
	}

	var result TestStruct
	request := mockRequest{
		Model: "gpt-4",
		Messages: []mockMessage{
			{Role: "user", Content: "Generate email"},
		},
	}

	resp, err := ChatHandler(mock, context.Background(), request, &result)

	if err != nil {
		t.Fatalf("Expected no error but got: %v", err)
	}

	if result.Email != "test@example.com" {
		t.Errorf("Expected test@example.com, got %s", result.Email)
	}

	// Verify we made 2 attempts
	if len(mock.requestHistory) != 2 {
		t.Errorf("Expected 2 attempts, got %d", len(mock.requestHistory))
	}

	_ = resp // Use response to avoid unused variable warning
}

// Test with validator that should use go-playground/validator
func TestValidatorIntegration(t *testing.T) {
	// This test ensures we're using the validator correctly
	v := validator.New()

	type User struct {
		Name string `validate:"required,min=3"`
		Age  int    `validate:"required,gte=0,lte=120"`
	}

	// Test valid struct
	valid := User{Name: "John", Age: 30}
	err := v.Struct(valid)
	if err != nil {
		t.Errorf("Expected valid struct to pass validation, got: %v", err)
	}

	// Test invalid struct
	invalid := User{Name: "Jo", Age: 30} // Name too short
	err = v.Struct(invalid)
	if err == nil {
		t.Error("Expected invalid struct to fail validation")
	}
}
