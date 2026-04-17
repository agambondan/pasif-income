package adapters

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/agambondan/pasif-income/internal/core/domain"
	_ "github.com/lib/pq"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(connStr string) (*PostgresRepository, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Initialize table if not exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS clips (
			id SERIAL PRIMARY KEY,
			video_url TEXT,
			headline TEXT,
			start_time TEXT,
			end_time TEXT,
			viral_score INT,
			reasoning TEXT,
			status TEXT DEFAULT 'pending',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create table: %v", err)
	}

	return &PostgresRepository{db}, nil
}

func (r *PostgresRepository) SaveClip(ctx context.Context, clip *domain.ClipSegment, videoURL string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO clips (video_url, headline, start_time, end_time, viral_score, reasoning)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, videoURL, clip.Headline, clip.StartTime, clip.EndTime, clip.Score, clip.Reasoning)
	return err
}

func (r *PostgresRepository) UpdateStatus(ctx context.Context, clipID string, status string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE clips SET status = $1 WHERE id = $2", status, clipID)
	return err
}

func (r *PostgresRepository) ListClips(ctx context.Context) ([]domain.ClipSegment, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT headline, start_time, end_time, viral_score, reasoning FROM clips ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clips []domain.ClipSegment
	for rows.Next() {
		var c domain.ClipSegment
		err := rows.Scan(&c.Headline, &c.StartTime, &c.EndTime, &c.Score, &c.Reasoning)
		if err != nil {
			return nil, err
		}
		clips = append(clips, c)
	}
	return clips, nil
}
