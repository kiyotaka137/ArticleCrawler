
CREATE TABLE IF NOT EXISTS articles (
    id bigserial PRIMARY KEY,
    url text NOT NULL UNIQUE,
    title text,
    body text,
    summary text,
    content_hash text,
    language text,
    read_time_minutes integer,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_articles_content_hash ON articles (content_hash);

CREATE TABLE IF NOT EXISTS fetch_attempts (
    id bigserial PRIMARY KEY,
    url text NOT NULL,
    attempt_time timestamptz DEFAULT now(),
    success boolean,
    response_code integer,
    error text
);

CREATE INDEX IF NOT EXISTS idx_fetch_attempts_url ON fetch_attempts (url);
