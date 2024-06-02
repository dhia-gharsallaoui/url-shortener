package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"url-shortener/pkg/model"
	"url-shortener/pkg/repository"
	"url-shortener/pkg/shortener"
)

type HandlerConfiguration struct {
	URLRepository  repository.URLRepository
	Logger         *slog.Logger
	Domain         string
	ExpiryDuration time.Duration
	Shortener      shortener.Shortener
}

// Handler struct holds the dependencies for the HTTP handlers
type Handler struct {
	repo           repository.URLRepository
	logger         *slog.Logger
	shortener      shortener.Shortener
	domain         string
	expiryDuration time.Duration
}

// NewHandler creates a new Handler with the given configuration
func NewHandler(config *HandlerConfiguration) *Handler {
	return &Handler{
		repo:           config.URLRepository,
		logger:         config.Logger,
		domain:         config.Domain,
		expiryDuration: config.ExpiryDuration,
		shortener:      config.Shortener,
	}
}

// ShortenURL handles the shortening of URLs
func (h *Handler) ShortenURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("invalid request", "error", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	url := model.URL{}
	if err := json.Unmarshal(body, &url); err != nil {
		h.logger.Error("invalid JSON format", "error", err)
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	if err := url.Sanitize(); err != nil {
		h.logger.Error("Invalid input data", "error", err)
		http.Error(w, "Invalid input data", http.StatusBadRequest)
		return
	}

	shortURL, err := h.shortener.GenerateShortURL(url.OriginalURL)
	if err != nil {
		h.logger.Error("Error generating short URL", "error", err)
		http.Error(w, "Failed to shorten URL", http.StatusInternalServerError)
		return
	}
	url.ShortURL = shortURL
	url.Expiry = time.Now().Add(h.expiryDuration) // Sets expiry to 30 days from now

	if err := h.repo.Save(r.Context(), &url); err != nil {
		h.logger.Error("Error saving URL", "error", err)
		http.Error(w, "Failed to save URL", http.StatusInternalServerError)
		return
	}

	h.logger.Info("URL shortened successfully", "originalURL", url.OriginalURL, "shortURL", shortURL)
	w.Header().Set("Content-Type", "text/plain")

	um, err := json.Marshal(url)
	if err != nil {
		h.logger.Error("error marshalling response %s", err)
		http.Error(w, "Error marshalling response", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write(um)
}

// Redirect handles redirection to the original URL
func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	url := urlConstruct(r)
	if !h.shortener.IsValidShortURL(url) {
		h.logger.Error("Invalid short URL provided", "URL", url)
		http.Error(w, "Invalid slug", http.StatusBadRequest)
		return
	}

	u, err := h.repo.Find(r.Context(), url)
	if err != nil {
		h.logger.Error("URL not found", "URL", url, "error", err)
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	if time.Now().After(u.Expiry) {
		h.logger.Info("Attempted to access expired URL", "URL", url)
		http.Error(w, "URL has expired", http.StatusGone)
		return
	}

	h.redirect(w, r, u.OriginalURL, url)
}

func (h *Handler) redirect(w http.ResponseWriter, r *http.Request, originalURL string, shortURL string) {
	// Increment the click count before redirecting
	if err := h.repo.IncrementClickCount(r.Context(), shortURL); err != nil {
		h.logger.Error("Failed to increment click count", "error", err)
		// Decide if we want to stop the redirect if the click count fails
	}
	http.Redirect(w, r, originalURL, http.StatusFound)
}

func urlConstruct(r *http.Request) string {
	return fmt.Sprintf("%s%s", r.Host, r.URL.Path)
}
