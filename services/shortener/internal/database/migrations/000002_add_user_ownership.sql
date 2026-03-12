ALTER TABLE short_urls
    ADD COLUMN IF NOT EXISTS user_id TEXT NOT NULL DEFAULT '';

ALTER TABLE short_urls
    DROP CONSTRAINT IF EXISTS short_urls_original_url_key;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'short_urls_user_id_original_url_key'
    ) THEN
        ALTER TABLE short_urls
            ADD CONSTRAINT short_urls_user_id_original_url_key UNIQUE (user_id, original_url);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_short_urls_user_id ON short_urls(user_id);
