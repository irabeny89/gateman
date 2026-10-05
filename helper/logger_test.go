package helper_test

import (
	"path/filepath"
	"testing"

	"github.com/irabeny89/gateman/helper"
)

func TestNewTextLogger(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "logs.txt")
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		filePath string
		wantErr  bool
	}{
		{
			name:     "Create logger without error on empty filepath",
			filePath: "",
			wantErr:  false,
		},
		{
			name:     "Create logger without error even with filepath",
			filePath: path,
			wantErr:  false,
		},
		{
			name:     "Error if filepath is directory",
			filePath: dir,
			wantErr:  true,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := helper.NewTextLogger(tt.filePath); err != nil != tt.wantErr {
				t.Fatalf("failed to get logger: %v", err)
			}
		})
	}
}

func TestNewJSONLogger(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "logs.txt")
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		filePath string
		wantErr  bool
	}{
		{
			name:     "Create logger without error on empty filepath",
			filePath: "",
			wantErr:  false,
		},
		{
			name:     "Create logger without error even with filepath",
			filePath: path,
			wantErr:  false,
		},
		{
			name:     "Error if filepath is directory",
			filePath: dir,
			wantErr:  true,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := helper.NewTextLogger(tt.filePath); err != nil != tt.wantErr {
				t.Fatalf("failed to get logger: %v", err)
			}
		})
	}
}
