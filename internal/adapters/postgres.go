package adapters

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
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
			title TEXT DEFAULT '',
			description TEXT DEFAULT '',
			video_path TEXT DEFAULT '',
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

	_, err = db.Exec(`
		ALTER TABLE generation_jobs ADD COLUMN IF NOT EXISTS title TEXT DEFAULT '';
		ALTER TABLE generation_jobs ADD COLUMN IF NOT EXISTS description TEXT DEFAULT '';
		ALTER TABLE generation_jobs ADD COLUMN IF NOT EXISTS video_path TEXT DEFAULT '';
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to alter generation_jobs table: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create users table: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			user_id INT REFERENCES users(id),
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create sessions table: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS connected_accounts (
			id TEXT PRIMARY KEY,
			user_id INT REFERENCES users(id),
			platform_id TEXT NOT NULL,
			display_name TEXT NOT NULL,
			auth_method TEXT DEFAULT 'chromium_profile',
			email TEXT,
			profile_path TEXT,
			access_token TEXT,
			refresh_token TEXT,
			expiry TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create connected_accounts table: %v", err)
	}

	_, err = db.Exec(`
		ALTER TABLE connected_accounts ADD COLUMN IF NOT EXISTS auth_method TEXT DEFAULT 'chromium_profile';
		ALTER TABLE connected_accounts ADD COLUMN IF NOT EXISTS email TEXT;
		ALTER TABLE connected_accounts ADD COLUMN IF NOT EXISTS profile_path TEXT;
		ALTER TABLE connected_accounts ADD COLUMN IF NOT EXISTS refresh_token TEXT;
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to alter connected_accounts table: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS distribution_jobs (
			id SERIAL PRIMARY KEY,
			generation_job_id TEXT REFERENCES generation_jobs(id),
			account_id TEXT REFERENCES connected_accounts(id),
			platform TEXT NOT NULL,
			status TEXT DEFAULT 'pending',
			status_detail TEXT DEFAULT '',
			external_id TEXT,
			error TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create distribution_jobs table: %v", err)
	}

	_, err = db.Exec(`
		ALTER TABLE distribution_jobs ADD COLUMN IF NOT EXISTS status_detail TEXT DEFAULT '';
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to alter distribution_jobs table: %v", err)
	}

	return &PostgresRepository{db}, nil
}

func (r *PostgresRepository) CreateUser(ctx context.Context, username, passwordHash string) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO users (username, password_hash) VALUES ($1, $2)", username, passwordHash)
	return err
}

func (r *PostgresRepository) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	var u domain.User
	err := r.db.QueryRowContext(ctx, "SELECT id, username, password_hash, created_at FROM users WHERE username = $1", username).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *PostgresRepository) CreateSession(ctx context.Context, userID int) (string, error) {
	token, err := randomToken(32)
	if err != nil {
		return "", err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO sessions (token, user_id, expires_at)
		VALUES ($1, $2, $3)
	`, token, userID, time.Now().Add(30*24*time.Hour))
	if err != nil {
		return "", err
	}
	return token, nil
}

func (r *PostgresRepository) GetUserBySessionToken(ctx context.Context, sessionToken string) (*domain.User, error) {
	var u domain.User
	err := r.db.QueryRowContext(ctx, `
		SELECT u.id, u.username, u.password_hash, u.created_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token = $1 AND s.expires_at > CURRENT_TIMESTAMP
	`, sessionToken).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *PostgresRepository) DeleteSession(ctx context.Context, sessionToken string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM sessions WHERE token = $1", sessionToken)
	return err
}

func (r *PostgresRepository) ListConnectedAccounts(ctx context.Context, userID int) ([]domain.ConnectedAccount, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, platform_id, display_name, auth_method, COALESCE(email, ''), COALESCE(profile_path, ''), COALESCE(access_token, ''), expiry, created_at 
		FROM connected_accounts WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accs []domain.ConnectedAccount
	for rows.Next() {
		var a domain.ConnectedAccount
		if err := rows.Scan(&a.ID, &a.UserID, &a.PlatformID, &a.DisplayName, &a.AuthMethod, &a.Email, &a.ProfilePath, &a.AccessToken, &a.Expiry, &a.CreatedAt); err != nil {
			return nil, err
		}
		accs = append(accs, a)
	}
	return accs, nil
}

func (r *PostgresRepository) GetConnectedAccountByID(ctx context.Context, accountID string) (*domain.ConnectedAccount, error) {
	var a domain.ConnectedAccount
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, platform_id, display_name, auth_method, COALESCE(email, ''), COALESCE(profile_path, ''), COALESCE(access_token, ''), expiry, created_at
		FROM connected_accounts
		WHERE id = $1
	`, accountID).Scan(&a.ID, &a.UserID, &a.PlatformID, &a.DisplayName, &a.AuthMethod, &a.Email, &a.ProfilePath, &a.AccessToken, &a.Expiry, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *PostgresRepository) SaveConnectedAccount(ctx context.Context, acc *domain.ConnectedAccount) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO connected_accounts (id, user_id, platform_id, display_name, auth_method, email, profile_path, access_token, expiry, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			access_token = EXCLUDED.access_token,
			expiry = EXCLUDED.expiry,
			auth_method = EXCLUDED.auth_method,
			email = EXCLUDED.email,
			profile_path = EXCLUDED.profile_path
	`, acc.ID, acc.UserID, acc.PlatformID, acc.DisplayName, acc.AuthMethod, acc.Email, acc.ProfilePath, acc.AccessToken, acc.Expiry, acc.CreatedAt)
	return err
}

func (r *PostgresRepository) DeleteConnectedAccount(ctx context.Context, accountID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM connected_accounts WHERE id = $1", accountID)
	return err
}

func (r *PostgresRepository) CreateDistributionJob(ctx context.Context, job *domain.DistributionJob) error {
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO distribution_jobs (generation_job_id, account_id, platform, status, status_detail, external_id, error, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id
	`, job.GenerationJobID, job.AccountID, job.Platform, job.Status, nullString(job.StatusDetail), nullString(job.ExternalID), nullString(job.Error), job.CreatedAt, job.UpdatedAt).Scan(&job.ID)
	return err
}

func (r *PostgresRepository) ListDistributionJobs(ctx context.Context, generationJobID string) ([]domain.DistributionJob, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, generation_job_id, account_id, platform, status, COALESCE(status_detail, ''), COALESCE(external_id, ''), COALESCE(error, ''), created_at, updated_at
		FROM distribution_jobs
		WHERE generation_job_id = $1
		ORDER BY created_at DESC
	`, generationJobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := []domain.DistributionJob{}
	for rows.Next() {
		var job domain.DistributionJob
		var extID, errStr, detail string
		if err := rows.Scan(&job.ID, &job.GenerationJobID, &job.AccountID, &job.Platform, &job.Status, &detail, &extID, &errStr, &job.CreatedAt, &job.UpdatedAt); err != nil {
			return nil, err
		}
		job.StatusDetail = detail
		job.ExternalID = extID
		job.Error = errStr
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (r *PostgresRepository) ListPendingDistributionJobs(ctx context.Context) ([]domain.DistributionJob, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, generation_job_id, account_id, platform, status, COALESCE(status_detail, ''), COALESCE(external_id, ''), COALESCE(error, ''), created_at, updated_at
		FROM distribution_jobs
		WHERE status = 'pending'
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := []domain.DistributionJob{}
	for rows.Next() {
		var job domain.DistributionJob
		var extID, errStr, detail string
		if err := rows.Scan(&job.ID, &job.GenerationJobID, &job.AccountID, &job.Platform, &job.Status, &detail, &extID, &errStr, &job.CreatedAt, &job.UpdatedAt); err != nil {
			return nil, err
		}
		job.StatusDetail = detail
		job.ExternalID = extID
		job.Error = errStr
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (r *PostgresRepository) UpdateDistributionJobStatus(ctx context.Context, jobID int, status string, statusDetail string, externalID string, errMsg string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE distribution_jobs
		SET status = $1, status_detail = $2, external_id = $3, error = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $5
	`, status, nullString(statusDetail), nullString(externalID), nullString(errMsg), jobID)
	return err
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
		INSERT INTO generation_jobs (id, niche, topic, title, description, video_path, status, error, created_at, updated_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, job.ID, job.Niche, job.Topic, job.Title, job.Description, job.VideoPath, job.Status, nullString(job.Error), job.CreatedAt, job.UpdatedAt, nullTimePtr(job.CompletedAt))
	return err
}

func (r *PostgresRepository) UpdateJobArtifact(ctx context.Context, jobID string, title string, description string, videoPath string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE generation_jobs
		SET title = $1, description = $2, video_path = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $4
	`, title, description, videoPath, jobID)
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

func randomToken(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (r *PostgresRepository) GetJob(ctx context.Context, jobID string) (*domain.GenerationJob, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, niche, topic, COALESCE(title, ''), COALESCE(description, ''), COALESCE(video_path, ''), status, COALESCE(error, ''), created_at, updated_at, completed_at
		FROM generation_jobs
		WHERE id = $1
	`, jobID)

	var job domain.GenerationJob
	var errorText string
	var completedAt sql.NullTime
	if err := row.Scan(&job.ID, &job.Niche, &job.Topic, &job.Title, &job.Description, &job.VideoPath, &job.Status, &errorText, &job.CreatedAt, &job.UpdatedAt, &completedAt); err != nil {
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
		SELECT id, niche, topic, COALESCE(title, ''), COALESCE(description, ''), COALESCE(video_path, ''), status, COALESCE(error, ''), created_at, updated_at, completed_at
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
		if err := rows.Scan(&job.ID, &job.Niche, &job.Topic, &job.Title, &job.Description, &job.VideoPath, &job.Status, &errorText, &job.CreatedAt, &job.UpdatedAt, &completedAt); err != nil {
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
