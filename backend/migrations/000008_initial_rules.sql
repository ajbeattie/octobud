-- +goose Up
CREATE TABLE IF NOT EXISTS rules (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    query TEXT,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    actions JSONB NOT NULL,
    display_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    view_id BIGINT,
    CONSTRAINT fk_rules_view_id FOREIGN KEY (view_id) REFERENCES views(id) ON DELETE CASCADE,
    CONSTRAINT check_query_or_view_id CHECK (
        (view_id IS NULL AND query IS NOT NULL AND query != '') OR
        (view_id IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_rules_enabled ON rules(enabled);
CREATE INDEX IF NOT EXISTS idx_rules_display_order ON rules(display_order);
CREATE INDEX IF NOT EXISTS idx_rules_view_id ON rules(view_id) WHERE view_id IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS rules;

