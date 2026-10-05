package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/irabeny89/gateman/helper"
)

func TestLogRequest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log.txt")
	logger, err := helper.NewTextLogger(path)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		logger *slog.Logger
		next   http.Handler
		want   string
	}{
		{
			name:   "Log to filepath",
			logger: logger,
			next: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
			want: "request",
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := LogRequest(tt.logger, tt.next)
			rr := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)
			handler.ServeHTTP(rr, r)
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			fileContent := string(b)
			if !strings.Contains(fileContent, tt.want) {
				t.Errorf("want to contain %q, got %q", tt.want, fileContent)
			}
		})
	}
}
