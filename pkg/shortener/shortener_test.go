package shortener

import (
	"io"
	"log/slog"
	"testing"
)

var mockLogger = slog.New(slog.NewJSONHandler(io.Discard, nil))

func TestCanonicalizeURL(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"http://www.example.com", "http://example.com"},
		{"https://www.example.com?b=2&a=1", "https://example.com?a=1&b=2"},
	}

	for _, tc := range testCases {
		result, err := canonicalizeURL(tc.input)
		if err != nil || result != tc.expected {
			t.Errorf("canonicalizeURL(%q) = %q, %v; want %q, nil", tc.input, result, err, tc.expected)
		}
	}
}

func setupShortener() CanonicalShortener {
	return CanonicalShortener{
		config: Config{
			Domain:     "http://sho.rt",
			Prefix:     "/s/",
			SlugLength: 6,
			Logger:     mockLogger,
		},
	}
}

func TestGenerateSlug(t *testing.T) {
	shortener := setupShortener()

	url := "http://example.com"
	expectedSlugLength := 6
	slug, err := shortener.GenerateSlug(url)
	if err != nil {
		t.Fatalf("GenerateSlug failed: %v", err)
	}
	if len(slug) != expectedSlugLength {
		t.Errorf("Expected slug length of %d, got %d", expectedSlugLength, len(slug))
	}
}

func TestIsValidShortURL(t *testing.T) {
	shortener := setupShortener()

	testCases := []struct {
		url   string
		valid bool
	}{
		{shortener.config.Prefix + "abc123", true},
		{shortener.config.Prefix + "xyz6789", false},
		{"xyz789", false},
		{shortener.config.Prefix + "", false},
	}

	for _, tc := range testCases {
		if valid := shortener.IsValidShortURL(tc.url); valid != tc.valid {
			t.Errorf("IsValidShortURL(%q) = %v; want %v", tc.url, valid, tc.valid)
		}
	}
}

func TestGenerateShortURL(t *testing.T) {
	shortener := setupShortener()

	testCases := []struct {
		inputURL   string
		shouldFail bool
	}{
		{"https://example.com?b=2&a=1", false},
		{"https://www.example.com?b=2&a=1", false},
		{"https://www.example.com?b=2&a=1", false},
		{"https://example.com?a=1&b=2", false},
		{"https://www.example.com?a=1&b=2", false},
		{"https://www.example.com?a=1&b=3", true},
		{"http://www.example.com", true},
		{"http://example.com", true},
	}

	var firstSlug string
	var firstURL string
	for i, tc := range testCases {
		shortURL, err := shortener.GenerateShortURL(tc.inputURL)
		if err != nil {
			t.Fatalf("GenerateShortURL failed for %q: %v", tc.inputURL, err)
		}
		if i == 0 {
			firstURL = tc.inputURL
			firstSlug = shortURL
		} else if (shortURL != firstSlug) != tc.shouldFail {
			t.Errorf("Expected the same slug for %s and %s, got %s and %s", tc.inputURL, firstURL, firstSlug, shortURL)
		}
	}
}
