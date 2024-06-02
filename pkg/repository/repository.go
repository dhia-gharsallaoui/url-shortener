package repository

import (
	"context"

	"url-shortener/pkg/model"
)

type URLRepository interface {
	Save(ctx context.Context, url *model.URL) error
	Find(ctx context.Context, shortURL string) (*model.URL, error)
	IncrementClickCount(ctx context.Context, shortURL string) error
}
