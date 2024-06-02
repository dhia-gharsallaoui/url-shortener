package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"url-shortener/pkg/model"

	"github.com/stretchr/testify/assert"
)

const shortDomain = "http://short.com"

var mockLogger = slog.New(slog.NewJSONHandler(io.Discard, nil))

type MockURLRepository struct {
	Store map[string]*model.URL
}

func (m *MockURLRepository) Save(ctx context.Context, url *model.URL) error {
	if url.OriginalURL == "http://error.com" {
		return errors.New("failed to save URL")
	}
	m.Store[url.ShortURL] = url
	return nil
}

func (m *MockURLRepository) Find(ctx context.Context, shortURL string) (*model.URL, error) {
	if u, ok := m.Store[shortURL]; ok {
		return u, nil
	}
	return nil, errors.New("URL not found")
}

func (m *MockURLRepository) IncrementClickCount(ctx context.Context, shortURL string) error {
	if url, ok := m.Store[shortURL]; ok {
		url.ClickCount += 1
		m.Store[shortURL] = url
		return nil
	}
	return errors.New("failed to increment click count")
}

// MockShortener is a mock implementation of Shortener
type MockShortener struct{}

func (m *MockShortener) GenerateShortURL(url string) (string, error) {
	if url == "http://error.com" {
		return "", errors.New("failed to generate short URL")
	}
	return shortDomain + "/redirect/xyz", nil
}

func (m *MockShortener) IsValidShortURL(url string) bool {
	return url == shortDomain+"/redirect/xyz" || url == shortDomain+"/redirect/404"
}

func TestShortenURL_MethodNotAllowed(t *testing.T) {
	handler := setupHandler()
	request := httptest.NewRequest(http.MethodGet, "/shorten", nil)
	recorder := httptest.NewRecorder()

	handler.ShortenURL(recorder, request)

	res := recorder.Result()
	assert.Equal(t, http.StatusMethodNotAllowed, res.StatusCode)
}

func TestShortenURL_InvalidRequestBody(t *testing.T) {
	handler := setupHandler()
	body := bytes.NewReader([]byte("{invalid json"))
	request := httptest.NewRequest(http.MethodPost, "/shorten", body)
	recorder := httptest.NewRecorder()

	handler.ShortenURL(recorder, request)

	res := recorder.Result()
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)
}

func TestShortenURL_InvalidInputData(t *testing.T) {
	handler := setupHandler()
	body := bytes.NewReader([]byte(`{"OriginalURL":"invalid url"}`))
	request := httptest.NewRequest(http.MethodPost, "/shorten", body)
	recorder := httptest.NewRecorder()

	handler.ShortenURL(recorder, request)

	res := recorder.Result()
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)
}

func TestShortenURL_Success(t *testing.T) {
	handler := setupHandler()
	url := model.URL{OriginalURL: "http://test.com"}
	bodyBytes, _ := json.Marshal(url)
	request := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewReader(bodyBytes))
	recorder := httptest.NewRecorder()

	handler.ShortenURL(recorder, request)

	res := recorder.Result()
	assert.Equal(t, http.StatusCreated, res.StatusCode)

	var resBody model.URL
	json.NewDecoder(res.Body).Decode(&resBody)
	assert.NotEmpty(t, resBody.ShortURL)
}

func TestRedirect_InvalidShortURL(t *testing.T) {
	handler := setupHandler()
	request := RedirectRequest(http.MethodGet, "/redirect", nil)
	recorder := httptest.NewRecorder()

	handler.Redirect(recorder, request)

	res := recorder.Result()
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)
}

func TestRedirect_URLNotFound(t *testing.T) {
	handler := setupHandler()
	request := RedirectRequest(http.MethodGet, "/redirect/404", nil)
	recorder := httptest.NewRecorder()

	handler.Redirect(recorder, request)

	res := recorder.Result()
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
}

func TestRedirect_ExpiredURL(t *testing.T) {
	handler := setupHandler()
	mockRepo := handler.repo.(*MockURLRepository)
	mockRepo.Save(context.Background(), &model.URL{
		ShortURL:    shortDomain + "/redirect/xyz",
		OriginalURL: "http://test.com",
		Expiry:      time.Now().Add(-24 * time.Hour),
	})

	request := RedirectRequest(http.MethodGet, "/redirect/xyz", nil)
	recorder := httptest.NewRecorder()

	handler.Redirect(recorder, request)

	res := recorder.Result()
	assert.Equal(t, http.StatusGone, res.StatusCode)
}

func TestRedirect_Success(t *testing.T) {
	handler := setupHandler()
	mockRepo := handler.repo.(*MockURLRepository)
	mockRepo.Save(context.Background(), &model.URL{
		ShortURL:    shortDomain + "/redirect/xyz",
		OriginalURL: "http://test.com",
		Expiry:      time.Now().Add(24 * time.Hour),
	})

	request := RedirectRequest(http.MethodGet, "/redirect/xyz", nil)
	recorder := httptest.NewRecorder()

	handler.Redirect(recorder, request)

	res := recorder.Result()
	assert.Equal(t, http.StatusFound, res.StatusCode)
}

func TestIncrementClickCount(t *testing.T) {
	handler := setupHandler()
	mockRepo := handler.repo.(*MockURLRepository)
	shortURL := shortDomain + "/redirect/xyz"
	mockRepo.Save(context.Background(), &model.URL{
		ShortURL:    shortURL,
		OriginalURL: "http://test.com",
		Expiry:      time.Now().Add(24 * time.Hour),
	})
	url, err := mockRepo.Find(context.Background(), shortURL)
	assert.Nil(t, err)
	firstCounter := url.ClickCount
	request := RedirectRequest(http.MethodGet, "/redirect/xyz", nil)
	recorder := httptest.NewRecorder()

	handler.Redirect(recorder, request)

	res := recorder.Result()
	assert.Equal(t, http.StatusFound, res.StatusCode)
	url, err = mockRepo.Find(context.Background(), shortURL)
	assert.Nil(t, err)
	counter := url.ClickCount
	assert.Equal(t, firstCounter, counter-1)

	handler.Redirect(recorder, request)

	res = recorder.Result()
	assert.Equal(t, http.StatusFound, res.StatusCode)
	url, err = mockRepo.Find(context.Background(), shortURL)
	assert.Nil(t, err)
	counter = url.ClickCount
	assert.Equal(t, firstCounter, counter-2)
}

func setupHandler() *Handler {
	mockRepo := &MockURLRepository{Store: make(map[string]*model.URL)}
	mockShortener := &MockShortener{}
	return NewHandler(&HandlerConfiguration{
		URLRepository:  mockRepo,
		Logger:         mockLogger,
		Domain:         shortDomain,
		ExpiryDuration: 30 * 24 * time.Hour,
		Shortener:      mockShortener,
	})
}

func RedirectRequest(method, target string, body io.Reader) *http.Request {
	request := httptest.NewRequest(method, target, body)
	request.Host = shortDomain
	return request
}
