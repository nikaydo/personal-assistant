CREATE INDEX IF NOT EXISTS idx_{{table}}_category ON {{table}} (category);
CREATE INDEX IF NOT EXISTS idx_{{table}}_goal ON {{table}} (goal);
CREATE INDEX IF NOT EXISTS idx_{{table}}_importance ON {{table}} (importance);
CREATE INDEX IF NOT EXISTS idx_{{table}}_status ON {{table}} (status);
CREATE INDEX IF NOT EXISTS idx_{{table}}_updated_at ON {{table}} (updated_at);
CREATE INDEX IF NOT EXISTS idx_{{table}}_embedding_hnsw ON {{table}} USING hnsw (embedding vector_l2_ops);
