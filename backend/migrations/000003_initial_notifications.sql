-- +goose Up
CREATE TABLE IF NOT EXISTS notifications (
    id BIGSERIAL PRIMARY KEY,
    github_id TEXT NOT NULL UNIQUE,
    repository_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    pull_request_id BIGINT REFERENCES pull_requests(id) ON DELETE SET NULL,
    subject_type TEXT NOT NULL,
    subject_title TEXT NOT NULL,
    subject_url TEXT,
    subject_latest_comment_url TEXT,
    reason TEXT,
    archived BOOLEAN NOT NULL DEFAULT FALSE,
    github_unread BOOLEAN,
    github_updated_at TIMESTAMPTZ,
    github_last_read_at TIMESTAMPTZ,
    github_url TEXT,
    github_subscription_url TEXT,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    payload JSONB,
    subject_raw JSONB,
    subject_fetched_at TIMESTAMPTZ,
    author_login TEXT,
    author_id BIGINT,
    is_read BOOLEAN NOT NULL DEFAULT false,
    muted BOOLEAN NOT NULL DEFAULT false,
    snoozed_until TIMESTAMPTZ,
    effective_sort_date TIMESTAMPTZ NOT NULL DEFAULT now(),
    snoozed_at TIMESTAMPTZ,
    starred BOOLEAN NOT NULL DEFAULT FALSE,
    filtered BOOLEAN NOT NULL DEFAULT FALSE,
    tag_ids bigint[] NOT NULL DEFAULT '{}',
    subject_number INTEGER NULL,
    subject_state TEXT NULL,
    subject_merged BOOLEAN NULL,
    subject_state_reason TEXT NULL
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_notifications_repository ON notifications(repository_id);
CREATE INDEX IF NOT EXISTS idx_notifications_subject_type ON notifications(subject_type);
CREATE INDEX IF NOT EXISTS idx_notifications_pull_request ON notifications(pull_request_id);
CREATE INDEX IF NOT EXISTS idx_notifications_author_login ON notifications(author_login) WHERE author_login IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_notifications_subject_raw_gin ON notifications USING GIN (subject_raw jsonb_path_ops);
CREATE INDEX IF NOT EXISTS idx_notifications_is_read ON notifications(is_read);
CREATE INDEX IF NOT EXISTS idx_notifications_archived ON notifications(archived);
CREATE INDEX IF NOT EXISTS idx_notifications_muted ON notifications(muted);
CREATE INDEX IF NOT EXISTS idx_notifications_snoozed_until ON notifications(snoozed_until) WHERE snoozed_until IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_notifications_effective_sort_date ON notifications (effective_sort_date DESC NULLS LAST);
CREATE INDEX IF NOT EXISTS idx_notifications_snoozed_at ON notifications(snoozed_at) WHERE snoozed_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_notifications_starred ON notifications(starred);
CREATE INDEX IF NOT EXISTS idx_notifications_filtered ON notifications(filtered);
CREATE INDEX IF NOT EXISTS idx_notifications_tag_ids ON notifications USING GIN(tag_ids);
CREATE INDEX IF NOT EXISTS idx_notifications_subject_number ON notifications(subject_number) WHERE subject_number IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_notifications_subject_state ON notifications(subject_state) WHERE subject_state IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_notifications_subject_merged ON notifications(subject_merged) WHERE subject_merged IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_notifications_subject_state_reason ON notifications(subject_state_reason) WHERE subject_state_reason IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS notifications;

