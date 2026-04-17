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
    status TEXT NOT NULL,
    error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL
);
