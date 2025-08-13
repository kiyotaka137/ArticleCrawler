CREATE TABLE IF NOT EXISTS articles(
    id TEXT PRIMARY KEY,
    url TEXT UNIQUE NOT NULL,
    title TEXT,
    content TEXT,
    html TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);