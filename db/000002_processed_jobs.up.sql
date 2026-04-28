BEGIN;

CREATE TABLE IF NOT EXISTS system.processed_jobs (
    job_id       text        PRIMARY KEY,
    type         text        NOT NULL,
    agent_id     text,
    processed_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_processed_jobs_processed_at
    ON system.processed_jobs (processed_at);

COMMIT;
