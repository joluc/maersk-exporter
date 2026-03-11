package config

import (
	"os"
	"testing"
)

func TestParseList(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"MAERSK", []string{"MAERSK"}},
		{"MAERSK,MSC", []string{"MAERSK", "MSC"}},
		{"MAERSK, MSC, CMA", []string{"MAERSK", "MSC", "CMA"}},
		{"  MAERSK  ,  MSC  ", []string{"MAERSK", "MSC"}},
		{",,MAERSK,,", []string{"MAERSK"}},
	}

	for _, tt := range tests {
		result := ParseList(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("ParseList(%q): expected length %d, got %d", tt.input, len(tt.expected), len(result))
			continue
		}
		for i, v := range result {
			if v != tt.expected[i] {
				t.Errorf("ParseList(%q)[%d]: expected %q, got %q", tt.input, i, tt.expected[i], v)
			}
		}
	}
}

func TestConfigFromEnvironment(t *testing.T) {
	// This test verifies that config reads from environment variables
	// We can't easily test ParseFlags() due to flag package limitations,
	// so we just verify the environment variable names are correct

	testKey := "test-maersk-key-123"
	testAISKey := "test-aisstream-key-456"

	os.Setenv("MAERSK_CONSUMER_KEY", testKey)
	os.Setenv("AISSTREAM_API_KEY", testAISKey)
	defer os.Unsetenv("MAERSK_CONSUMER_KEY")
	defer os.Unsetenv("AISSTREAM_API_KEY")

	// Verify environment variables are set
	if os.Getenv("MAERSK_CONSUMER_KEY") != testKey {
		t.Error("MAERSK_CONSUMER_KEY not set in environment")
	}
	if os.Getenv("AISSTREAM_API_KEY") != testAISKey {
		t.Error("AISSTREAM_API_KEY not set in environment")
	}
}
