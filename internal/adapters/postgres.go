package adapters

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/agambondan/pasif-income/internal/core/domain"
	"github.com/agambondan/pasif-income/internal/pkg/crypto"
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
			pin_comment TEXT DEFAULT '',
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
		ALTER TABLE generation_jobs ADD COLUMN IF NOT EXISTS pin_comment TEXT DEFAULT '';
		ALTER TABLE generation_jobs ADD COLUMN IF NOT EXISTS video_path TEXT DEFAULT '';
		ALTER TABLE generation_jobs ADD COLUMN IF NOT EXISTS current_stage TEXT DEFAULT 'queued';
		ALTER TABLE generation_jobs ADD COLUMN IF NOT EXISTS progress_pct INT DEFAULT 0;
		ALTER TABLE generation_jobs ADD COLUMN IF NOT EXISTS cancelled_at TIMESTAMP NULL;
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
			auth_method TEXT NOT NULL DEFAULT 'chromium_profile',
			email TEXT NOT NULL DEFAULT '',
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
		ALTER TABLE connected_accounts ADD COLUMN IF NOT EXISTS auth_method TEXT NOT NULL DEFAULT 'chromium_profile';
		ALTER TABLE connected_accounts ADD COLUMN IF NOT EXISTS email TEXT;
		ALTER TABLE connected_accounts ADD COLUMN IF NOT EXISTS profile_path TEXT;
		ALTER TABLE connected_accounts ADD COLUMN IF NOT EXISTS refresh_token TEXT;
		UPDATE connected_accounts
			SET email = LOWER(TRIM(COALESCE(email, ''))),
			    auth_method = COALESCE(NULLIF(TRIM(auth_method), ''), 'chromium_profile');
		ALTER TABLE connected_accounts ALTER COLUMN email SET DEFAULT '';
		ALTER TABLE connected_accounts ALTER COLUMN email SET NOT NULL;
		ALTER TABLE connected_accounts ALTER COLUMN auth_method SET DEFAULT 'chromium_profile';
		ALTER TABLE connected_accounts ALTER COLUMN auth_method SET NOT NULL;
		WITH ranked AS (
			SELECT id,
			       ROW_NUMBER() OVER (
			         PARTITION BY user_id, platform_id, auth_method, email
			         ORDER BY created_at DESC, id DESC
			       ) AS rn
			FROM connected_accounts
		)
		DELETE FROM connected_accounts ca
		USING ranked r
		WHERE ca.id = r.id AND r.rn > 1;
		CREATE UNIQUE INDEX IF NOT EXISTS connected_accounts_identity_key
			ON connected_accounts (user_id, platform_id, auth_method, email);
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
			scheduled_at TIMESTAMP NULL,
			retry_source_job_id INT REFERENCES distribution_jobs(id),
			retry_attempt INT DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create distribution_jobs table: %v", err)
	}

	_, err = db.Exec(`
		ALTER TABLE distribution_jobs ADD COLUMN IF NOT EXISTS status_detail TEXT DEFAULT '';
		ALTER TABLE distribution_jobs ADD COLUMN IF NOT EXISTS scheduled_at TIMESTAMP NULL;
		ALTER TABLE distribution_jobs ADD COLUMN IF NOT EXISTS retry_source_job_id INT REFERENCES distribution_jobs(id);
		ALTER TABLE distribution_jobs ADD COLUMN IF NOT EXISTS retry_attempt INT DEFAULT 0;
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to alter distribution_jobs table: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS video_metric_snapshots (
			id SERIAL PRIMARY KEY,
			user_id INT REFERENCES users(id),
			generation_job_id TEXT REFERENCES generation_jobs(id),
			distribution_job_id INT REFERENCES distribution_jobs(id),
			account_id TEXT REFERENCES connected_accounts(id),
			platform TEXT NOT NULL,
			niche TEXT DEFAULT '',
			external_id TEXT NOT NULL,
			video_title TEXT DEFAULT '',
			view_count BIGINT DEFAULT 0,
			like_count BIGINT DEFAULT 0,
			comment_count BIGINT DEFAULT 0,
			collected_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create video_metric_snapshots table: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS community_reply_drafts (
			id SERIAL PRIMARY KEY,
			user_id INT REFERENCES users(id),
			generation_job_id TEXT REFERENCES generation_jobs(id),
			distribution_job_id INT REFERENCES distribution_jobs(id),
			account_id TEXT REFERENCES connected_accounts(id),
			platform TEXT NOT NULL,
			niche TEXT DEFAULT '',
			video_title TEXT DEFAULT '',
			external_comment_id TEXT NOT NULL,
			parent_comment_id TEXT DEFAULT '',
			comment_author TEXT DEFAULT '',
			comment_text TEXT NOT NULL,
			suggested_reply TEXT NOT NULL,
			status TEXT DEFAULT 'draft',
			posted_external_id TEXT DEFAULT '',
			replied_at TIMESTAMP NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(distribution_job_id, external_comment_id)
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create community_reply_drafts table: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS agent_events (
			id TEXT PRIMARY KEY,
			job_id TEXT REFERENCES generation_jobs(id),
			type TEXT NOT NULL,
			content TEXT NOT NULL,
			metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
			timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent_events table: %v", err)
	}

	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_agent_events_job_id ON agent_events(job_id);
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent_events index: %v", err)
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

func (r *PostgresRepository) ListUsers(ctx context.Context) ([]domain.User, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, username, password_hash, created_at
		FROM users
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]domain.User, 0)
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
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

func (r *PostgresRepository) ListAllConnectedAccounts(ctx context.Context) ([]domain.ConnectedAccount, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, platform_id, display_name, auth_method, COALESCE(email, ''), COALESCE(profile_path, ''), COALESCE(access_token, ''), COALESCE(refresh_token, ''), expiry, created_at
		FROM connected_accounts
		ORDER BY created_at DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accs := make([]domain.ConnectedAccount, 0)
	for rows.Next() {
		var a domain.ConnectedAccount
		if err := rows.Scan(&a.ID, &a.UserID, &a.PlatformID, &a.DisplayName, &a.AuthMethod, &a.Email, &a.ProfilePath, &a.AccessToken, &a.RefreshToken, &a.Expiry, &a.CreatedAt); err != nil {
			return nil, err
		}
		if decryptedAt, err := crypto.Decrypt(a.AccessToken); err == nil {
			a.AccessToken = decryptedAt
		}
		if decryptedRt, err := crypto.Decrypt(a.RefreshToken); err == nil {
			a.RefreshToken = decryptedRt
		}
		accs = append(accs, a)
	}
	return accs, nil
}

func (r *PostgresRepository) ListConnectedAccounts(ctx context.Context, userID int) ([]domain.ConnectedAccount, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, platform_id, display_name, auth_method, COALESCE(email, ''), COALESCE(profile_path, ''), COALESCE(access_token, ''), COALESCE(refresh_token, ''), expiry, created_at
		FROM connected_accounts WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accs []domain.ConnectedAccount
	for rows.Next() {
		var a domain.ConnectedAccount
		if err := rows.Scan(&a.ID, &a.UserID, &a.PlatformID, &a.DisplayName, &a.AuthMethod, &a.Email, &a.ProfilePath, &a.AccessToken, &a.RefreshToken, &a.Expiry, &a.CreatedAt); err != nil {
			return nil, err
		}

		// Decrypt tokens
		if decryptedAt, err := crypto.Decrypt(a.AccessToken); err == nil {
			a.AccessToken = decryptedAt
		}
		if decryptedRt, err := crypto.Decrypt(a.RefreshToken); err == nil {
			a.RefreshToken = decryptedRt
		}

		accs = append(accs, a)
	}
	return accs, nil
}

func (r *PostgresRepository) GetConnectedAccountByID(ctx context.Context, accountID string) (*domain.ConnectedAccount, error) {
	var a domain.ConnectedAccount
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, platform_id, display_name, auth_method, COALESCE(email, ''), COALESCE(profile_path, ''), COALESCE(access_token, ''), COALESCE(refresh_token, ''), expiry, created_at
		FROM connected_accounts
		WHERE id = $1
	`, accountID).Scan(&a.ID, &a.UserID, &a.PlatformID, &a.DisplayName, &a.AuthMethod, &a.Email, &a.ProfilePath, &a.AccessToken, &a.RefreshToken, &a.Expiry, &a.CreatedAt)
	if err != nil {
		return nil, err
	}

	// Decrypt tokens
	if decryptedAt, err := crypto.Decrypt(a.AccessToken); err == nil {
		a.AccessToken = decryptedAt
	}
	if decryptedRt, err := crypto.Decrypt(a.RefreshToken); err == nil {
		a.RefreshToken = decryptedRt
	}

	return &a, nil
}

func (r *PostgresRepository) SaveConnectedAccount(ctx context.Context, acc *domain.ConnectedAccount) error {
	acc.Email = strings.ToLower(strings.TrimSpace(acc.Email))

	// Encrypt tokens before saving
	encAccess, err := crypto.Encrypt(acc.AccessToken)
	if err != nil {
		return err
	}
	encRefresh, err := crypto.Encrypt(acc.RefreshToken)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO connected_accounts (id, user_id, platform_id, display_name, auth_method, email, profile_path, access_token, refresh_token, expiry, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (user_id, platform_id, auth_method, email) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			expiry = EXCLUDED.expiry,
			auth_method = EXCLUDED.auth_method,
			email = EXCLUDED.email,
			profile_path = EXCLUDED.profile_path
	`, acc.ID, acc.UserID, acc.PlatformID, acc.DisplayName, acc.AuthMethod, acc.Email, acc.ProfilePath, encAccess, encRefresh, acc.Expiry, acc.CreatedAt)
	return err
}

func (r *PostgresRepository) DeleteConnectedAccount(ctx context.Context, accountID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM connected_accounts WHERE id = $1", accountID)
	return err
}

func (r *PostgresRepository) CreateDistributionJob(ctx context.Context, job *domain.DistributionJob) error {
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO distribution_jobs (generation_job_id, account_id, platform, status, status_detail, external_id, error, scheduled_at, retry_source_job_id, retry_attempt, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING id
	`, job.GenerationJobID, job.AccountID, job.Platform, job.Status, nullString(job.StatusDetail), nullString(job.ExternalID), nullString(job.Error), nullTimePtr(job.ScheduledAt), nullIntPtr(job.RetrySourceJobID), job.RetryAttempt, job.CreatedAt, job.UpdatedAt).Scan(&job.ID)
	return err
}

func (r *PostgresRepository) ListDistributionJobs(ctx context.Context, generationJobID string) ([]domain.DistributionJob, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, generation_job_id, account_id, platform, status, COALESCE(status_detail, ''), COALESCE(external_id, ''), COALESCE(error, ''), scheduled_at, retry_source_job_id, COALESCE(retry_attempt, 0), created_at, updated_at
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
		var scheduledAt sql.NullTime
		var retrySource sql.NullInt64
		if err := rows.Scan(&job.ID, &job.GenerationJobID, &job.AccountID, &job.Platform, &job.Status, &detail, &extID, &errStr, &scheduledAt, &retrySource, &job.RetryAttempt, &job.CreatedAt, &job.UpdatedAt); err != nil {
			return nil, err
		}
		job.StatusDetail = detail
		job.ExternalID = extID
		job.Error = errStr
		if scheduledAt.Valid {
			job.ScheduledAt = &scheduledAt.Time
		}
		if retrySource.Valid {
			value := int(retrySource.Int64)
			job.RetrySourceJobID = &value
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (r *PostgresRepository) ListAllDistributionJobs(ctx context.Context, userID int) ([]domain.DistributionJob, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT dj.id, dj.generation_job_id, dj.account_id, dj.platform, dj.status, COALESCE(dj.status_detail, ''), COALESCE(dj.external_id, ''), COALESCE(dj.error, ''), dj.scheduled_at, dj.retry_source_job_id, COALESCE(dj.retry_attempt, 0), dj.created_at, dj.updated_at
		FROM distribution_jobs dj
		JOIN connected_accounts ca ON ca.id = dj.account_id
		WHERE ca.user_id = $1
		ORDER BY dj.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := []domain.DistributionJob{}
	for rows.Next() {
		var job domain.DistributionJob
		var extID, errStr, detail string
		var scheduledAt sql.NullTime
		var retrySource sql.NullInt64
		if err := rows.Scan(&job.ID, &job.GenerationJobID, &job.AccountID, &job.Platform, &job.Status, &detail, &extID, &errStr, &scheduledAt, &retrySource, &job.RetryAttempt, &job.CreatedAt, &job.UpdatedAt); err != nil {
			return nil, err
		}
		job.StatusDetail = detail
		job.ExternalID = extID
		job.Error = errStr
		if scheduledAt.Valid {
			job.ScheduledAt = &scheduledAt.Time
		}
		if retrySource.Valid {
			value := int(retrySource.Int64)
			job.RetrySourceJobID = &value
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (r *PostgresRepository) ListPendingDistributionJobs(ctx context.Context) ([]domain.DistributionJob, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, generation_job_id, account_id, platform, status, COALESCE(status_detail, ''), COALESCE(external_id, ''), COALESCE(error, ''), scheduled_at, retry_source_job_id, COALESCE(retry_attempt, 0), created_at, updated_at
		FROM distribution_jobs
		WHERE status = 'pending' AND (scheduled_at IS NULL OR scheduled_at <= CURRENT_TIMESTAMP)
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
		var scheduledAt sql.NullTime
		var retrySource sql.NullInt64
		if err := rows.Scan(&job.ID, &job.GenerationJobID, &job.AccountID, &job.Platform, &job.Status, &detail, &extID, &errStr, &scheduledAt, &retrySource, &job.RetryAttempt, &job.CreatedAt, &job.UpdatedAt); err != nil {
			return nil, err
		}
		job.StatusDetail = detail
		job.ExternalID = extID
		job.Error = errStr
		if scheduledAt.Valid {
			job.ScheduledAt = &scheduledAt.Time
		}
		if retrySource.Valid {
			value := int(retrySource.Int64)
			job.RetrySourceJobID = &value
		}
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

func (r *PostgresRepository) SaveVideoMetricSnapshot(ctx context.Context, snapshot *domain.VideoMetricSnapshot) error {
	return r.db.QueryRowContext(ctx, `
		INSERT INTO video_metric_snapshots (
			user_id, generation_job_id, distribution_job_id, account_id, platform, external_id, video_title, view_count, like_count, comment_count, collected_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`, snapshot.UserID, snapshot.GenerationJobID, snapshot.DistributionJobID, snapshot.AccountID, snapshot.Platform, snapshot.ExternalID, snapshot.VideoTitle, int64(snapshot.ViewCount), int64(snapshot.LikeCount), int64(snapshot.CommentCount), snapshot.CollectedAt).Scan(&snapshot.ID)
}

func (r *PostgresRepository) SaveAgentEvent(ctx context.Context, event *domain.AgentEvent) error {
	metadata := []byte(`{}`)
	if len(event.Metadata) > 0 {
		if data, err := json.Marshal(event.Metadata); err == nil {
			metadata = data
		}
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO agent_events (id, job_id, type, content, metadata, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			job_id = EXCLUDED.job_id,
			type = EXCLUDED.type,
			content = EXCLUDED.content,
			metadata = EXCLUDED.metadata,
			timestamp = EXCLUDED.timestamp
	`, event.ID, event.JobID, event.Type, event.Content, metadata, event.Timestamp)
	return err
}

func (r *PostgresRepository) ListAgentEvents(ctx context.Context, jobID string) ([]domain.AgentEvent, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, COALESCE(job_id, ''), type, content, COALESCE(metadata, '{}'::jsonb), timestamp
		FROM agent_events
		WHERE job_id = $1
		ORDER BY timestamp ASC, id ASC
	`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := []domain.AgentEvent{}
	for rows.Next() {
		var event domain.AgentEvent
		var typeText string
		var metadataBytes []byte
		if err := rows.Scan(&event.ID, &event.JobID, &typeText, &event.Content, &metadataBytes, &event.Timestamp); err != nil {
			return nil, err
		}
		event.Type = domain.AgentEventType(typeText)
		if len(metadataBytes) > 0 && string(metadataBytes) != "null" {
			_ = json.Unmarshal(metadataBytes, &event.Metadata)
		}
		if event.Metadata == nil {
			event.Metadata = map[string]any{}
		}
		events = append(events, event)
	}
	return events, nil
}

func (r *PostgresRepository) SaveCommunityReplyDraft(ctx context.Context, draft *domain.CommunityReplyDraft) error {
	return r.db.QueryRowContext(ctx, `
		INSERT INTO community_reply_drafts (
			user_id, generation_job_id, distribution_job_id, account_id, platform, niche, video_title, external_comment_id, parent_comment_id, comment_author, comment_text, suggested_reply, status, posted_external_id, replied_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (distribution_job_id, external_comment_id)
		DO UPDATE SET
			user_id = EXCLUDED.user_id,
			generation_job_id = EXCLUDED.generation_job_id,
			account_id = EXCLUDED.account_id,
			platform = EXCLUDED.platform,
			niche = EXCLUDED.niche,
			video_title = EXCLUDED.video_title,
			parent_comment_id = EXCLUDED.parent_comment_id,
			comment_author = EXCLUDED.comment_author,
			comment_text = EXCLUDED.comment_text,
			suggested_reply = EXCLUDED.suggested_reply,
			status = EXCLUDED.status,
			posted_external_id = EXCLUDED.posted_external_id,
			replied_at = EXCLUDED.replied_at,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id, created_at, updated_at
	`, draft.UserID, draft.GenerationJobID, draft.DistributionJobID, draft.AccountID, draft.Platform, draft.Niche, draft.VideoTitle, draft.ExternalCommentID, draft.ParentCommentID, draft.CommentAuthor, draft.CommentText, draft.SuggestedReply, draft.Status, nullString(draft.PostedExternalID), draft.RepliedAt).Scan(&draft.ID, &draft.CreatedAt, &draft.UpdatedAt)
}

func (r *PostgresRepository) ListVideoMetricSnapshots(ctx context.Context, userID int) ([]domain.VideoMetricSnapshot, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, generation_job_id, distribution_job_id, account_id, platform, COALESCE(niche, ''), external_id, COALESCE(video_title, ''), COALESCE(view_count, 0), COALESCE(like_count, 0), COALESCE(comment_count, 0), collected_at
		FROM video_metric_snapshots
		WHERE user_id = $1
		ORDER BY collected_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	snapshots := []domain.VideoMetricSnapshot{}
	for rows.Next() {
		var snap domain.VideoMetricSnapshot
		var viewCount, likeCount, commentCount int64
		if err := rows.Scan(&snap.ID, &snap.UserID, &snap.GenerationJobID, &snap.DistributionJobID, &snap.AccountID, &snap.Platform, &snap.Niche, &snap.ExternalID, &snap.VideoTitle, &viewCount, &likeCount, &commentCount, &snap.CollectedAt); err != nil {
			return nil, err
		}
		if viewCount > 0 {
			snap.ViewCount = uint64(viewCount)
		}
		if likeCount > 0 {
			snap.LikeCount = uint64(likeCount)
		}
		if commentCount > 0 {
			snap.CommentCount = uint64(commentCount)
		}
		snapshots = append(snapshots, snap)
	}
	return snapshots, nil
}

func (r *PostgresRepository) ListCommunityReplyDrafts(ctx context.Context, userID int) ([]domain.CommunityReplyDraft, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, COALESCE(generation_job_id, ''), COALESCE(distribution_job_id, 0), COALESCE(account_id, ''), COALESCE(platform, ''), COALESCE(niche, ''), COALESCE(video_title, ''), COALESCE(external_comment_id, ''), COALESCE(parent_comment_id, ''), COALESCE(comment_author, ''), COALESCE(comment_text, ''), COALESCE(suggested_reply, ''), COALESCE(status, 'draft'), COALESCE(posted_external_id, ''), replied_at, created_at, updated_at
		FROM community_reply_drafts
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	drafts := []domain.CommunityReplyDraft{}
	for rows.Next() {
		var draft domain.CommunityReplyDraft
		var repliedAt sql.NullTime
		if err := rows.Scan(&draft.ID, &draft.UserID, &draft.GenerationJobID, &draft.DistributionJobID, &draft.AccountID, &draft.Platform, &draft.Niche, &draft.VideoTitle, &draft.ExternalCommentID, &draft.ParentCommentID, &draft.CommentAuthor, &draft.CommentText, &draft.SuggestedReply, &draft.Status, &draft.PostedExternalID, &repliedAt, &draft.CreatedAt, &draft.UpdatedAt); err != nil {
			return nil, err
		}
		if repliedAt.Valid {
			draft.RepliedAt = &repliedAt.Time
		}
		drafts = append(drafts, draft)
	}
	return drafts, nil
}

func (r *PostgresRepository) ListVideoMetricSnapshotsByJob(ctx context.Context, generationJobID string) ([]domain.VideoMetricSnapshot, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, generation_job_id, distribution_job_id, account_id, platform, COALESCE(niche, ''), external_id, COALESCE(video_title, ''), COALESCE(view_count, 0), COALESCE(like_count, 0), COALESCE(comment_count, 0), collected_at
		FROM video_metric_snapshots
		WHERE generation_job_id = $1
		ORDER BY collected_at DESC
	`, generationJobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	snapshots := []domain.VideoMetricSnapshot{}
	for rows.Next() {
		var snap domain.VideoMetricSnapshot
		var viewCount, likeCount, commentCount int64
		if err := rows.Scan(&snap.ID, &snap.UserID, &snap.GenerationJobID, &snap.DistributionJobID, &snap.AccountID, &snap.Platform, &snap.Niche, &snap.ExternalID, &snap.VideoTitle, &viewCount, &likeCount, &commentCount, &snap.CollectedAt); err != nil {
			return nil, err
		}
		if viewCount > 0 {
			snap.ViewCount = uint64(viewCount)
		}
		if likeCount > 0 {
			snap.LikeCount = uint64(likeCount)
		}
		if commentCount > 0 {
			snap.CommentCount = uint64(commentCount)
		}
		snapshots = append(snapshots, snap)
	}
	return snapshots, nil
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
		INSERT INTO generation_jobs (id, niche, topic, title, description, video_path, status, current_stage, progress_pct, error, created_at, updated_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, job.ID, job.Niche, job.Topic, job.Title, job.Description, job.VideoPath, job.Status, job.CurrentStage, job.ProgressPct, nullString(job.Error), job.CreatedAt, job.UpdatedAt, nullTimePtr(job.CompletedAt))
	return err
}

func (r *PostgresRepository) UpdateJobArtifact(ctx context.Context, jobID string, title string, description string, pinComment string, videoPath string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE generation_jobs
		SET title = $1, description = $2, pin_comment = $3, video_path = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $5
	`, title, description, pinComment, videoPath, jobID)
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

func (r *PostgresRepository) UpdateJobProgress(ctx context.Context, jobID string, stage string, progress int) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE generation_jobs
		SET current_stage = $1, progress_pct = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`, stage, progress, jobID)
	return err
}

func (r *PostgresRepository) CancelJob(ctx context.Context, jobID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE generation_jobs
		SET status = 'failed', error = 'cancelled by operator', cancelled_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND status IN ('queued', 'running')
	`, jobID)
	return err
}

func (r *PostgresRepository) GetJob(ctx context.Context, jobID string) (*domain.GenerationJob, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, niche, topic, COALESCE(video_url, ''), COALESCE(title, ''), COALESCE(description, ''), COALESCE(pin_comment, ''), COALESCE(video_path, ''), status, COALESCE(current_stage, ''), progress_pct, COALESCE(error, ''), created_at, updated_at, completed_at, cancelled_at
		FROM generation_jobs
		WHERE id = $1
	`, jobID)

	var job domain.GenerationJob
	var errorText, stage string
	var completedAt, cancelledAt sql.NullTime
	if err := row.Scan(&job.ID, &job.Niche, &job.Topic, &job.VideoURL, &job.Title, &job.Description, &job.PinComment, &job.VideoPath, &job.Status, &stage, &job.ProgressPct, &errorText, &job.CreatedAt, &job.UpdatedAt, &completedAt, &cancelledAt); err != nil {
		return nil, err
	}
	job.Error = errorText
	job.CurrentStage = stage
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	if cancelledAt.Valid {
		job.CancelledAt = &cancelledAt.Time
	}
	return &job, nil
}

func (r *PostgresRepository) ListJobs(ctx context.Context) ([]domain.GenerationJob, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, niche, topic, COALESCE(video_url, ''), status, COALESCE(current_stage, ''), progress_pct, COALESCE(error, ''), created_at, updated_at, completed_at, cancelled_at
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
		var errorText, stage string
		var completedAt, cancelledAt sql.NullTime
		if err := rows.Scan(&job.ID, &job.Niche, &job.Topic, &job.VideoURL, &job.Status, &stage, &job.ProgressPct, &errorText, &job.CreatedAt, &job.UpdatedAt, &completedAt, &cancelledAt); err != nil {
			return nil, err
		}
		job.Error = errorText
		job.CurrentStage = stage
		if completedAt.Valid {
			job.CompletedAt = &completedAt.Time
		}
		if cancelledAt.Valid {
			job.CancelledAt = &cancelledAt.Time
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func randomToken(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
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

func nullIntPtr(value *int) interface{} {
	if value == nil {
		return nil
	}
	return *value
}
