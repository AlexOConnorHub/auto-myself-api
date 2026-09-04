package middleware

import "testing"

func TestGetBearerFromHeader(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{"Valid Bearer Token", "Bearer validtoken", "validtoken"},
		{"No Bearer Prefix", "invalidtoken", ""},
		{"Empty Header", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBearerFromHeader(tt.header)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestAuthMiddleware(t *testing.T) {
}
