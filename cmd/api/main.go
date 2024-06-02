package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"url-shortener/pkg/config"
	"url-shortener/pkg/handler"

	"url-shortener/pkg/repository"
	"url-shortener/pkg/shortener"

	"github.com/jackc/pgx/v5"
)

const (
	maxRetries    = 5
	retryInterval = 2 * time.Second
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.LoadConfig(logger)
	if err != nil {
		logger.Error("Failed to load configuration: %v", err)
		os.Exit(1)
	}
	ctx := context.Background()
	// Database setup
	conn, err := setupDatabse(ctx, logger, cfg.DATABASE_URL)
	if err != nil {
		logger.Error("Unable to connect to database: %v", err)
		os.Exit(1)
	}

	defer conn.Close(ctx)

	// Repository and Shortener setup
	repo := repository.NewPostgresURLRepository(conn)

	config := shortener.Config{
		Domain:     cfg.Domain,
		Prefix:     "/r/",
		SlugLength: 6,
		Logger:     logger,
	}
	urlShortener := shortener.NewShortener(config)

	handlerConfig := handler.HandlerConfiguration{
		URLRepository:  repo,
		Shortener:      urlShortener,
		Logger:         logger,
		ExpiryDuration: cfg.Expiry,
	}
	urlHandler := handler.NewHandler(&handlerConfig)

	// HTTP server setup
	http.HandleFunc("/create", urlHandler.ShortenURL)
	http.HandleFunc("/r/", urlHandler.Redirect)

	// Start the server
	logger.Info(fmt.Sprintf("Starting server on %s", cfg.Port))
	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		logger.Error(fmt.Sprintf("Failed to start server: %v", err))
		os.Exit(1)
	}
}

func setupDatabse(ctx context.Context, logger *slog.Logger, url string) (*pgx.Conn, error) {
	var conn *pgx.Conn
	var err error
	// Retry mechanism for database connection
	for i := 0; i < maxRetries; i++ {
		conn, err = pgx.Connect(ctx, url)
		if err == nil {
			break
		}
		logger.Warn(fmt.Sprintf("unable to connect to database (attempt %d/%d): %v", i+1, maxRetries, err))
		time.Sleep(retryInterval)
	}

	if err != nil {
		return conn, fmt.Errorf("exceeded maximum retries. Unable to connect to database: %v", err)
	}
	return conn, err
}
