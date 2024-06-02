package shortener

import (
	"fmt"
	"hash/crc32"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
)

const charset = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

type Config struct {
	Logger     *slog.Logger
	Domain     string
	Prefix     string
	SlugLength int
}

type CanonicalShortener struct {
	config Config
}

func NewShortener(config Config) Shortener {
	return &CanonicalShortener{
		config: config,
	}
}

func (s *CanonicalShortener) GenerateShortURL(url string) (string, error) {
	url, err := canonicalizeURL(url)
	if err != nil {
		return "", err
	}
	slug, err := s.GenerateSlug(url)
	if err != nil {
		return "", err
	}
	return s.config.Domain + s.config.Prefix + slug, nil
}

func (s *CanonicalShortener) IsValidShortURL(u string) bool {
	URL, err := url.Parse("http://" + u)
	if err != nil {
		s.config.Logger.Error(fmt.Sprintf("invalid URL %s", err.Error()))
		return false
	}

	if !strings.HasPrefix(URL.Path, s.config.Prefix) {
		return false
	}
	return s.isValidSlug(strings.TrimPrefix(URL.Path, s.config.Prefix))
}

// GenerateSlug takes an URL string and returns a deterministic 6-character slug.
// to change with an implementation with less collision probability
func (s *CanonicalShortener) GenerateSlug(url string) (string, error) {
	hash := crc32.ChecksumIEEE([]byte(url))
	return base62Encode(hash, s.config.SlugLength), nil
}

func (s *CanonicalShortener) isValidSlug(slug string) bool {
	if len(slug) != s.config.SlugLength {
		return false
	}
	match, _ := regexp.MatchString("^[a-zA-Z0-9]+$", slug)
	return match
}

func base62Encode(number uint32, length int) string {
	encoded := make([]byte, length)

	base := uint32(len(charset))
	for i := length - 1; i >= 0; i-- {
		encoded[i] = charset[number%base]
		number /= base
	}

	return string(encoded)
}

// Normalize the URL to its canonical form
func canonicalizeURL(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	parsedURL.Host = strings.TrimPrefix(parsedURL.Host, "www.")
	parsedURL.RawQuery = canonicalizeQuery(parsedURL.RawQuery)
	return parsedURL.String(), nil
}

// Normalize and sort query parameters
// to avoid duplicated when the order of parameters change
func canonicalizeQuery(rawQuery string) string {
	params, err := url.ParseQuery(rawQuery)
	if err != nil {
		return rawQuery
	}

	canonicalQuery := url.Values{}
	for key, values := range params {
		canonicalQuery[key] = values
	}

	return canonicalQuery.Encode()
}
