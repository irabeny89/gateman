package server

import (
	"net/http"
	"testing"
)

func TestStrict(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		want *http.Server
	}{
		{
			name: "Test strict server configuration",
			want: &http.Server{
				ReadHeaderTimeout: readHeaderTimeout,
				ReadTimeout:       readTimeout,
				WriteTimeout:      writeTimeout,
				IdleTimeout:       idleTimeout,
				MaxHeaderBytes:    maxHeaderBytes,
			},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Strict()
			
			if got.ReadHeaderTimeout != tt.want.ReadHeaderTimeout ||
				got.ReadTimeout != tt.want.ReadTimeout ||
				got.WriteTimeout != tt.want.WriteTimeout ||
				got.IdleTimeout != tt.want.IdleTimeout ||
				got.MaxHeaderBytes != tt.want.MaxHeaderBytes {
				t.Errorf("Strict() = %v, want %v", got, tt.want)
			}
		})
	}
}
