package server

import (
	"net/http"
	"time"
)

var (
	readHeaderTimeout = 3 * time.Second
	readTimeout       = 5 * time.Second
	writeTimeout      = 10 * time.Second
	idleTimeout       = 30 * time.Second
	maxHeaderBytes    = 1 << 20
)
// Strict returns a pre-configured http.Server with strict security settings to mitigate common web vulnerabilities and attacks.
func Strict() *http.Server {
	return &http.Server{
		// Crucial security infrastructure configurations
		ReadHeaderTimeout: readHeaderTimeout,  // Protects against Slowloris attacks
		ReadTimeout:       readTimeout,  // Absolute time window to read the request
		WriteTimeout:      writeTimeout, // Max time allowed to write the response
		IdleTimeout:       idleTimeout, // How long to keep a connection alive
		MaxHeaderBytes:    maxHeaderBytes,          // 1 MB limit specifically for HTTP Headers
	}
}
