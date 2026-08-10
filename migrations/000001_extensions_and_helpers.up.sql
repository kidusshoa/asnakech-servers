CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Shared trigger to keep updated_at current on row changes.
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
