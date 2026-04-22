CREATE TABLE IF NOT EXISTS agent_events (
    id TEXT PRIMARY KEY,
    job_id TEXT REFERENCES generation_jobs(id),
    type TEXT NOT NULL,
    content TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_agent_events_job_id ON agent_events(job_id);
