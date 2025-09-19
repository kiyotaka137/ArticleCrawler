package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

type Article struct {
	ID              int64
	URL             string
	Title           string
	Body            string
	Summary         string
	ContentHash     string
	Language        string
	ReadTimeMinutes int32
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(ctx context.Context, dbURL string) (*Repository, error) {
	cfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.ConnectConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return &Repository{pool: pool}, nil
}

func (r *Repository) Close() {
	r.pool.Close()
}

func (r *Repository) SaveArticle(ctx context.Context, a *Article) (bool, error) {
	if a.ContentHash != "" {
		var existingID int64
		err := r.pool.QueryRow(ctx, "SELECT id FROM articles WHERE content_hash=$1", a.ContentHash).Scan(&existingID)
		if err == nil {
			return false, nil
		}
	}
	var id int64
	query := `
INSERT INTO articles (url, title, body, summary, content_hash, language, read_time_minutes)
VALUES ($1,$2,$3,$4,$5,$6,$7)
ON CONFLICT (url) DO UPDATE SET
  title = EXCLUDED.title,
  body = EXCLUDED.body,
  summary = EXCLUDED.summary,
  content_hash = EXCLUDED.content_hash,
  language = EXCLUDED.language,
  read_time_minutes = EXCLUDED.read_time_minutes,
  updated_at = now()
RETURNING id
`
	err := r.pool.QueryRow(ctx, query,
		a.URL, a.Title, a.Body, a.Summary, a.ContentHash, a.Language, a.ReadTimeMinutes,
	).Scan(&id)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *Repository) GetArticleByID(ctx context.Context, id int64) (*Article, error) {
	var a Article
	row := r.pool.QueryRow(ctx, "SELECT id, url, title, body, summary, content_hash, language, read_time_minutes, created_at, updated_at FROM articles WHERE id=$1", id)
	err := row.Scan(&a.ID, &a.URL, &a.Title, &a.Body, &a.Summary, &a.ContentHash, &a.Language, &a.ReadTimeMinutes, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *Repository) ListArticles(ctx context.Context, limit, offset int32) ([]*Article, error) {
	rows, err := r.pool.Query(ctx, "SELECT id, url, title, body, summary, content_hash, language, read_time_minutes, created_at, updated_at FROM articles ORDER BY created_at DESC LIMIT $1 OFFSET $2", limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []*Article
	for rows.Next() {
		var a Article
		if err := rows.Scan(&a.ID, &a.URL, &a.Title, &a.Body, &a.Summary, &a.ContentHash, &a.Language, &a.ReadTimeMinutes, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		res = append(res, &a)
	}
	return res, nil
}

func (r *Repository) RecordFetchAttempt(ctx context.Context, url string, success bool, code int, errText string) {
	_, err := r.pool.Exec(ctx, "INSERT INTO fetch_attempts (url, success, response_code, error) VALUES ($1,$2,$3,$4)", url, success, code, errText)
	if err != nil {
		fmt.Println("failed to record fetch attempt:", err)
	}
}
