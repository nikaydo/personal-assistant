CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS {{table}} (
    id TEXT PRIMARY KEY,
    category TEXT NOT NULL,
    goal TEXT NOT NULL,
    importance TEXT NOT NULL,
    status TEXT NOT NULL,
    text TEXT NOT NULL,
    embedding VECTOR({{dimension}}) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
