package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"url-shortener/pkg/model"

	"github.com/jackc/pgx/v5"
)

type PostgresURLRepository struct {
	db *pgx.Conn // to replace with pool for scaling
}

func NewPostgresURLRepository(db *pgx.Conn) URLRepository {
	return &PostgresURLRepository{db: db}
}

func (r *PostgresURLRepository) Save(ctx context.Context, url *model.URL) error {
	if err := url.Sanitize(); err != nil {
		return fmt.Errorf("failed to sanitize URL: %v", err)
	}

	query := `INSERT INTO urls (short_url, original_url, expiry, click_count) VALUES ($1, $2, $3, $4) ON CONFLICT (short_url) DO UPDATE SET original_url = EXCLUDED.original_url, expiry = EXCLUDED.expiry, click_count = EXCLUDED.click_count`
	_, err := r.db.Exec(ctx, query, url.ShortURL, url.OriginalURL, url.Expiry, url.ClickCount)
	if err != nil {
		return fmt.Errorf("error saving URL to database: %v", err)
	}
	return nil
}

func (r *PostgresURLRepository) Find(ctx context.Context, shortURL string) (*model.URL, error) {
	query := `SELECT original_url, expiry, click_count FROM urls WHERE short_url = $1`
	var originalURL string
	var expiry time.Time
	var clickCount int64
	err := r.db.QueryRow(ctx, query, shortURL).Scan(&originalURL, &expiry, &clickCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("URL not found")
		}
		return nil, fmt.Errorf("error retrieving URL from database: %v", err)
	}
	return &model.URL{
		OriginalURL: originalURL,
		ShortURL:    shortURL,
		Expiry:      expiry,
		ClickCount:  clickCount,
	}, nil
}

func (r *PostgresURLRepository) IncrementClickCount(ctx context.Context, shortURL string) error {
	query := `UPDATE urls SET click_count = click_count + 1 WHERE short_url = $1`
	_, err := r.db.Exec(ctx, query, shortURL)
	if err != nil {
		return fmt.Errorf("error incrementing click count: %v", err)
	}
	return nil
}
