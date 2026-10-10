// helper contains utility functions for common operations.
package helper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"

	errs "github.com/irabeny89/gateman/error"
)

// ContextValue is a generic helper to extract a value from the context.
// It returns the value and an error if the value is not found or of the wrong type.
// If a default value is provided, it will be returned instead of an error when the value is not found.
func ContextValue[T any](ctx context.Context, key string, def *T) (*T, error) {
	val, ok := ctx.Value(key).(T)
	if !ok && def != nil {
		return def, nil
	}
	if !ok {
		return nil, fmt.Errorf("context value for key %s not found", key)
	}
	return &val, nil
}

// UnmarshalJSONBody is a generic helper to safely unmarshal a JSON request body into a target struct.
// It enforces the Content-Type header to be application/json and restricts the request body size to prevent DoS attacks.
// It returns an error if the request body is not valid JSON or if the target struct cannot be unmarshaled.
// 
// Errors are wrapped with context for easier debugging.
// ErrNotJSON is returned if the Content-Type header is not application/json.
// ErrBodyTooLarge is returned if the request body exceeds the specified size limit.
func UnmarshalJSONBody[T any](w http.ResponseWriter, r *http.Request, target *T, size int64) error {
	// Enforce JSON request headers
	if r.Header.Get("Content-Type") != "application/json" {
		return errs.ErrNotJSON
	}
	// Prevent DoS by restricting request size (e.g., 1MB max)
	if size <= 0 {
		size = 1048576 // default to 1MB
	}
	r.Body = http.MaxBytesReader(w, r.Body, size)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() // Opt-in to catch typo'd or malicious fields

	if err := decoder.Decode(target); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return errs.ErrBodyTooLarge
		}
		return fmt.Errorf("Invalid JSON structure: %w", err)
	}
	return nil
}

// MarshalJSONResponse is a generic helper to safely marshal and write a JSON response.
// It sets the Content-Type header to application/json and writes the status code.
// If marshaling fails, it logs the error and writes an internal server error response.
// It returns true if the response was successfully written, false if error response was returned as a result of failed JSON encoding.
func MarshalJSONResponse[T any](log *slog.Logger, w http.ResponseWriter, statusCode int, data *T) (ok bool) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Error("failed to marshal JSON response", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	return true
}

// GenOTP generates a random numeric OTP of the specified length.
func GenOTP(length int) string {
	// Generate a random OTP of specified length
	otp := ""
	for i := 0; i < length; i++ {
		otp += fmt.Sprintf("%d", rand.Intn(10))
	}
	return otp
}
