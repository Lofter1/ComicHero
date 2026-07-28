package comicvine

import (
	"testing"
)

func TestParseID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		wantErr  bool
	}{
		{
			name:     "numeric string",
			input:    "1443",
			expected: 1443,
			wantErr:  false,
		},
		{
			name:     "comic vine format",
			input:    "4000-1443",
			expected: 1443,
			wantErr:  false,
		},
		{
			name:     "api url",
			input:    "https://comicvine.gamespot.com/api/character/4000-1443/",
			expected: 1443,
			wantErr:  false,
		},
		{
			name:     "api url without trailing slash",
			input:    "https://comicvine.gamespot.com/api/character/4000-1443",
			expected: 1443,
			wantErr:  false,
		},
		{
			name:     "large number",
			input:    "4000-999999",
			expected: 999999,
			wantErr:  false,
		},
		{
			name:    "invalid input",
			input:   "not-a-number",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseID(tt.input)
			
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestFormatID(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{1443, "4000-1443"},
		{0, "4000-0"},
		{999999, "4000-999999"},
	}

	for _, tt := range tests {
		result := FormatID(tt.input)
		if result != tt.expected {
			t.Errorf("FormatID(%d) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestURLBuilder(t *testing.T) {
	builder := NewURLBuilder()
	
	tests := []struct {
		name         string
		url          string
		expectedPart string
	}{
		{
			name:         "resource url",
			url:          builder.ResourceURL(ResourceCharacter, 1443),
			expectedPart: "/character/4000-1443/",
		},
		{
			name:         "search url",
			url:          builder.SearchURL("Spider-Man"),
			expectedPart: "/search/?query=Spider-Man",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !contains(tt.url, tt.expectedPart) {
				t.Errorf("URL %s does not contain expected part %s", tt.url, tt.expectedPart)
			}
		})
	}
}

func TestGetBestImage(t *testing.T) {
	tests := []struct {
		name     string
		image    Image
		expected string
	}{
		{
			name: "super url available",
			image: Image{
				SuperURL: "https://example.com/super.jpg",
				MediumURL: "https://example.com/medium.jpg",
			},
			expected: "https://example.com/super.jpg",
		},
		{
			name: "only medium url available",
			image: Image{
				MediumURL: "https://example.com/medium.jpg",
				SmallURL:  "https://example.com/small.jpg",
			},
			expected: "https://example.com/medium.jpg",
		},
		{
			name: "only icon url available",
			image: Image{
				IconURL: "https://example.com/icon.jpg",
			},
			expected: "https://example.com/icon.jpg",
		},
		{
			name:     "no images available",
			image:    Image{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetBestImage(tt.image)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && 
		(s == substr || len(s) >= len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
