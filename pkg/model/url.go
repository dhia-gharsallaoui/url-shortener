package model

import (
	"errors"
	"net/url"
	"strings"
	"time"
)

// URL represents the structure of stored URLs
type URL struct {
	OriginalURL string    `json:"original_url,omitempty"`
	ShortURL    string    `json:"short_url,omitempty"`
	Expiry      time.Time `json:"expiry,omitempty"`
	ClickCount  int64     `json:"click_count,omitempty"`
}

// Sanitize cleans and validates the URL structure to prevent injection and ensure data integrity.
func (u *URL) Sanitize() error {
	if u.OriginalURL == "" {
		return errors.New("original URL is empty")
	}

	if !strings.HasPrefix(u.OriginalURL, "http") && !strings.HasPrefix(u.OriginalURL, "https") {
		return errors.New("unsupported URL scheme")
	}

	parsedURL, err := url.ParseRequestURI(u.OriginalURL)
	if err != nil {
		return err
	}

	u.OriginalURL = parsedURL.String()

	// Truncate data to prevent overly long URLs
	if len(u.OriginalURL) > 2048 {
		u.OriginalURL = u.OriginalURL[:2048]
	}

	return nil
}
