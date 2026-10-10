package helper

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// Define custom unexported types for context keys to follow Go best practices
type contextKey string

func TestContextValue(t *testing.T) {
	// Sample data for pointers
	defaultString := "default_value"
	defaultInt := 42

	// Define test cases
	tests := []struct {
		name       string
		ctxSetup   func() context.Context
		key        string
		defString  *string // Using string and int for generic diversity
		defInt     *int
		wantString *string
		wantInt    *int
		wantErr    error
	}{
		{
			name: "Success - Found value matching correct type",
			ctxSetup: func() context.Context {
				return context.WithValue(context.Background(), "user_id", "user_123")
			},
			key:        "user_id",
			defString:  nil,
			wantString: ptr("user_123"),
			wantErr:    nil,
		},
		{
			name: "Fallback - Missing key with default provided",
			ctxSetup: func() context.Context {
				return context.Background()
			},
			key:        "missing_key",
			defString:  &defaultString,
			wantString: &defaultString,
			wantErr:    nil,
		},
		{
			name: "Error - Missing key without default",
			ctxSetup: func() context.Context {
				return context.Background()
			},
			key:        "missing_key",
			defString:  nil,
			wantString: nil,
			wantErr:    fmt.Errorf("context value for key missing_key not found"),
		},
		{
			name: "Error - Key exists but is of the wrong type (Type Mismatch)",
			ctxSetup: func() context.Context {
				// Passing an int when a string is expected
				return context.WithValue(context.Background(), "user_id", 999)
			},
			key:        "user_id",
			defString:  nil,
			wantString: nil,
			// Since type assertion fails, it acts as !ok. No default is given, so expect error.
			wantErr: fmt.Errorf("context value for key user_id not found"),
		},
		{
			name: "Fallback - Key exists but wrong type AND default provided",
			ctxSetup: func() context.Context {
				// Passing a string when an int is expected
				return context.WithValue(context.Background(), "retry_count", "not_an_int")
			},
			key:     "retry_count",
			defInt:  &defaultInt,
			wantInt: &defaultInt,
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.ctxSetup()

			// Route based on whether we are testing string or int types
			if tt.defInt != nil || tt.wantInt != nil {
				got, err := ContextValue[int](ctx, tt.key, tt.defInt)
				checkResult(t, got, tt.wantInt, err, tt.wantErr)
			} else {
				got, err := ContextValue[string](ctx, tt.key, tt.defString)
				checkResult(t, got, tt.wantString, err, tt.wantErr)
			}
		})
	}
}

// Helper to check test outcomes and handle nil pointer comparisons safely
func checkResult[T comparable](t *testing.T, got *T, want *T, err error, wantErr error) {
	t.Helper()

	// 1. Verify error expectations
	if (err != nil && wantErr == nil) || (err == nil && wantErr != nil) {
		t.Fatalf("unexpected error state: got %v, wantErr %v", err, wantErr)
	}
	if err != nil && wantErr != nil && err.Error() != wantErr.Error() {
		t.Fatalf("expected error message %q, got %q", wantErr.Error(), err.Error())
	}

	// 2. Verify returned pointers and target values
	if got == nil && want == nil {
		return
	}
	if (got == nil && want != nil) || (got != nil && want == nil) {
		t.Fatalf("mismatched pointer return: got %v, want %v", got, want)
	}
	if *got != *want {
		t.Errorf("dereferenced value mismatch: got %v, want %v", *got, *want)
	}
}

// Inline helper to turn constants into pointers for test validation
func ptr[T any](v T) *T {
	return &v
}
func TestUnmarshalJSONBody(t *testing.T) {
	size := int64(1048576) // 1MB
	type body struct {
		A string `json:"a"`
	}
	r := httptest.NewRequest("POST", "/", bytes.NewBufferString(`{"a":"b"}`))
	r.Header.Set("Content-Type", "application/json")
	type args[T any] struct {
		w      http.ResponseWriter
		r      *http.Request
		target T
		size   int64
	}
	tests := []struct {
		name    string
		args    args[any]
		wantErr bool
	}{
		{
			name: "Test successful JSON unmarshalling",
			args: args[any]{
				w:      httptest.NewRecorder(),
				r:      r,
				target: &body{A: "b"}, // valid target for JSON unmarshalling
				size:   size,
			},
			wantErr: false,
		},
		{
			name: "Test failed JSON unmarshalling",
			args: args[any]{
				w:      httptest.NewRecorder(),
				r:      httptest.NewRequest("POST", "/", nil),
				target: make(chan int), // channels cannot be unmarshaled from JSON
				size:   size,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := UnmarshalJSONBody[any](tt.args.w, tt.args.r, &tt.args.target, tt.args.size); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSONBody() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMarshalJSONResponse(t *testing.T) {
	type args struct {
		log        *slog.Logger
		w          http.ResponseWriter
		statusCode int
		data       any
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test successful JSON response",
			args: args{
				log:        slog.New(slog.NewTextHandler(os.Stdout, nil)),
				w:          httptest.NewRecorder(),
				statusCode: http.StatusOK,
				data:       map[string]string{"message": "success"},
			},
			want: true,
		},
		{
			name: "Test failed JSON response",
			args: args{
				log:        slog.New(slog.NewTextHandler(os.Stdout, nil)),
				w:          httptest.NewRecorder(),
				statusCode: http.StatusOK,
				data:       make(chan int), // channels cannot be marshaled to JSON
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MarshalJSONResponse(tt.args.log, tt.args.w, tt.args.statusCode, &tt.args.data); got != tt.want {
				t.Errorf("MarshalJSONResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenOTP(t *testing.T) {
	tests := []struct {
		name    string
		length  int
		wantLen int
	}{
		{
			name:    "Test 6-digit OTP",
			length:  6,
			wantLen: 6,
		},
		{
			name:    "Test 4-digit OTP",
			length:  4,
			wantLen: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenOTP(tt.length)
			if len(got) != tt.wantLen {
				t.Errorf("GenOTP() = %v, want %v", len(got), tt.wantLen)
			}
		})
	}
}
