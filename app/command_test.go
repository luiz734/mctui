package app

import (
	"testing"
)

func TestWrapCommandOutput(t *testing.T) {
	// Test cases
	tests := []struct {
		input    string
		width    int
		expected string
	}{
		{
			input:    "longstring",
			width:    5,
			expected: "longstring",
		},
		{
			input:    "xxx yyy zzz ",
			width:    5,
			expected: "xxx\nyyy\nzzz",
		},
		{
			input:    "connect: connection refused: is the server down?",
			width:    88,
			expected: "connect: connection refused: is the server down?",
		},
		{
			input:    "hello hello hello",
			width:    11,
			expected: "hello hello\nhello",
		},
	}

	for _, tc := range tests {
		result := wrapCommandOutput(tc.input, tc.width)
		if result != tc.expected {
			t.Errorf("\nExpected:%s\nGot:%s", tc.expected, result)
		}
	}
}
