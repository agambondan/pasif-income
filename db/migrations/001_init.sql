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
);

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
);

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    token TEXT PRIMARY KEY,
    user_id INT REFERENCES users(id),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL
);

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
);

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
);
