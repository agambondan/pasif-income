package adapters

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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
			source_id TEXT,
			s3_path TEXT,
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

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS generation_jobs (
			id TEXT PRIMARY KEY,
			niche TEXT NOT NULL,
			topic TEXT NOT NULL,
			status TEXT NOT NULL,
			error TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			completed_at TIMESTAMP NULL
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create jobs table: %v", err)
	}

	return &PostgresRepository{db}, nil
}

func (r *PostgresRepository) SaveClip(ctx context.Context, clip *domain.ClipSegment, sourceID string, s3Path string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO clips (source_id, s3_path, headline, start_time, end_time, viral_score, reasoning)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, sourceID, s3Path, clip.Headline, clip.StartTime, clip.EndTime, clip.Score, clip.Reasoning)
	return err
}

func (r *PostgresRepository) UpdateStatus(ctx context.Context, clipID string, status string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE clips SET status = $1 WHERE id = $2", status, clipID)
	return err
}

func (r *PostgresRepository) ListClips(ctx context.Context) ([]domain.Clip, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, COALESCE(source_id, ''), COALESCE(s3_path, ''), COALESCE(headline, ''), COALESCE(start_time, ''), COALESCE(end_time, ''), COALESCE(status, 'pending'), COALESCE(viral_score, 0), COALESCE(reasoning, '')
		FROM clips
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	clips := []domain.Clip{}
	for rows.Next() {
		var c domain.Clip
		err := rows.Scan(&c.ID, &c.SourceID, &c.S3Path, &c.Headline, &c.StartTime, &c.EndTime, &c.Status, &c.ViralScore, &c.Reasoning)
		if err != nil {
			return nil, err
		}
		clips = append(clips, c)
	}
	return clips, nil
}

func (r *PostgresRepository) CreateJob(ctx context.Context, job *domain.GenerationJob) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO generation_jobs (id, niche, topic, status, error, created_at, updated_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, job.ID, job.Niche, job.Topic, job.Status, nullString(job.Error), job.CreatedAt, job.UpdatedAt, nullTimePtr(job.CompletedAt))
	return err
}

func (r *PostgresRepository) UpdateJobStatus(ctx context.Context, jobID string, status string, errMsg string) error {
	if status == "completed" || status == "failed" {
		_, err := r.db.ExecContext(ctx, `
			UPDATE generation_jobs
			SET status = $1, error = $2, updated_at = CURRENT_TIMESTAMP, completed_at = CURRENT_TIMESTAMP
			WHERE id = $3
		`, status, nullString(errMsg), jobID)
		return err
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE generation_jobs
		SET status = $1, error = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`, status, nullString(errMsg), jobID)
	return err
}

func (r *PostgresRepository) GetJob(ctx context.Context, jobID string) (*domain.GenerationJob, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, niche, topic, status, COALESCE(error, ''), created_at, updated_at, completed_at
		FROM generation_jobs
		WHERE id = $1
	`, jobID)

	var job domain.GenerationJob
	var errorText string
	var completedAt sql.NullTime
	if err := row.Scan(&job.ID, &job.Niche, &job.Topic, &job.Status, &errorText, &job.CreatedAt, &job.UpdatedAt, &completedAt); err != nil {
		return nil, err
	}
	job.Error = errorText
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	return &job, nil
}

func (r *PostgresRepository) ListJobs(ctx context.Context) ([]domain.GenerationJob, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, niche, topic, status, COALESCE(error, ''), created_at, updated_at, completed_at
		FROM generation_jobs
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := []domain.GenerationJob{}
	for rows.Next() {
		var job domain.GenerationJob
		var errorText string
		var completedAt sql.NullTime
		if err := rows.Scan(&job.ID, &job.Niche, &job.Topic, &job.Status, &errorText, &job.CreatedAt, &job.UpdatedAt, &completedAt); err != nil {
			return nil, err
		}
		job.Error = errorText
		if completedAt.Valid {
			job.CompletedAt = &completedAt.Time
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func nullString(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}

func nullTimePtr(value *time.Time) interface{} {
	if value == nil {
		return nil
	}
	return *value
}
