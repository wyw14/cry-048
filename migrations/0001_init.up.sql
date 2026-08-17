-- 0001_init.up.sql: Initial schema for design-review platform.

CREATE TABLE IF NOT EXISTS projects (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'active',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    version      INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS boards (
    id           TEXT PRIMARY KEY,
    project_id   TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    width        INTEGER NOT NULL CHECK (width > 0 AND width <= 100000),
    height       INTEGER NOT NULL CHECK (height > 0 AND height <= 100000),
    status       TEXT NOT NULL DEFAULT 'open',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    version      INTEGER NOT NULL DEFAULT 1
);
CREATE INDEX IF NOT EXISTS idx_boards_project_id ON boards(project_id);

CREATE TABLE IF NOT EXISTS versions (
    id           TEXT PRIMARY KEY,
    board_id     TEXT NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    number       INTEGER NOT NULL CHECK (number > 0),
    label        TEXT NOT NULL DEFAULT '',
    preview_key  TEXT NOT NULL,
    notes        TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'draft',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by   TEXT NOT NULL DEFAULT '',
    version      INTEGER NOT NULL DEFAULT 1,
    UNIQUE (board_id, number)
);
CREATE INDEX IF NOT EXISTS idx_versions_board_id ON versions(board_id);

CREATE TABLE IF NOT EXISTS memberships (
    id           TEXT PRIMARY KEY,
    project_id   TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id      TEXT NOT NULL,
    role         TEXT NOT NULL CHECK (role IN ('owner', 'editor', 'viewer')),
    joined_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_memberships_project_id ON memberships(project_id);

CREATE TABLE IF NOT EXISTS annotations (
    id           TEXT PRIMARY KEY,
    project_id   TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    board_id     TEXT NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    version_id   TEXT NOT NULL REFERENCES versions(id) ON DELETE CASCADE,
    title        TEXT NOT NULL,
    body         TEXT NOT NULL DEFAULT '',
    anchor_type  TEXT NOT NULL CHECK (anchor_type IN ('point', 'region')),
    anchor_x     DOUBLE PRECISION NOT NULL DEFAULT 0,
    anchor_y     DOUBLE PRECISION NOT NULL DEFAULT 0,
    anchor_w     DOUBLE PRECISION NOT NULL DEFAULT 0,
    anchor_h     DOUBLE PRECISION NOT NULL DEFAULT 0,
    priority     TEXT NOT NULL DEFAULT 'normal' CHECK (priority IN ('low', 'normal', 'high', 'critical')),
    status       TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'replied', 'review', 'resolved', 'closed')),
    assignee_id  TEXT NOT NULL DEFAULT '',
    reporter_id  TEXT NOT NULL,
    due_at       TIMESTAMPTZ,
    resolved_at  TIMESTAMPTZ,
    resolved_by  TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    version      INTEGER NOT NULL DEFAULT 1
);
CREATE INDEX IF NOT EXISTS idx_annotations_project_id ON annotations(project_id);
CREATE INDEX IF NOT EXISTS idx_annotations_board_id ON annotations(board_id);
CREATE INDEX IF NOT EXISTS idx_annotations_version_id ON annotations(version_id);
CREATE INDEX IF NOT EXISTS idx_annotations_assignee_id ON annotations(assignee_id);
CREATE INDEX IF NOT EXISTS idx_annotations_status ON annotations(status);
CREATE INDEX IF NOT EXISTS idx_annotations_priority ON annotations(priority);
CREATE INDEX IF NOT EXISTS idx_annotations_created_at ON annotations(created_at);

CREATE TABLE IF NOT EXISTS replies (
    id           TEXT PRIMARY KEY,
    annotation_id TEXT NOT NULL REFERENCES annotations(id) ON DELETE CASCADE,
    author_id    TEXT NOT NULL,
    body         TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    edited_at    TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_replies_annotation_id ON replies(annotation_id);

CREATE TABLE IF NOT EXISTS attachments (
    id           TEXT PRIMARY KEY,
    annotation_id TEXT NOT NULL REFERENCES annotations(id) ON DELETE CASCADE,
    filename     TEXT NOT NULL,
    media_type   TEXT NOT NULL,
    size         BIGINT NOT NULL CHECK (size > 0),
    storage_key  TEXT NOT NULL,
    uploaded_by  TEXT NOT NULL,
    uploaded_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_attachments_annotation_id ON attachments(annotation_id);

CREATE TABLE IF NOT EXISTS review_rounds (
    id            TEXT PRIMARY KEY,
    project_id    TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    board_id      TEXT NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    title         TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    conclusion    TEXT NOT NULL DEFAULT '',
    recommendation TEXT NOT NULL DEFAULT '' CHECK (recommendation IN ('', 'approve', 'approve_with_conditions', 'reject', 'defer')),
    decided_by    TEXT NOT NULL DEFAULT '',
    decided_at    TIMESTAMPTZ,
    closed_at     TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    version       INTEGER NOT NULL DEFAULT 1
);
CREATE INDEX IF NOT EXISTS idx_review_rounds_board_id ON review_rounds(board_id);

CREATE TABLE IF NOT EXISTS review_snapshots (
    id           TEXT PRIMARY KEY,
    round_id     TEXT NOT NULL REFERENCES review_rounds(id) ON DELETE CASCADE,
    project_id   TEXT NOT NULL,
    board_id     TEXT NOT NULL,
    version_id   TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by   TEXT NOT NULL,
    counts       JSONB NOT NULL DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS idx_review_snapshots_round_id ON review_snapshots(round_id);

CREATE TABLE IF NOT EXISTS audit_log (
    id           BIGSERIAL PRIMARY KEY,
    actor_id      TEXT NOT NULL DEFAULT '',
    action        TEXT NOT NULL,
    entity_type   TEXT NOT NULL,
    entity_id     TEXT NOT NULL,
    before        TEXT NOT NULL DEFAULT '',
    after         TEXT NOT NULL DEFAULT '',
    at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    request_id    TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_log(actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_entity ON audit_log(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_at ON audit_log(at);

CREATE TABLE IF NOT EXISTS idempotency_keys (
    key          TEXT PRIMARY KEY,
    response     JSONB NOT NULL,
    status_code  INTEGER NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at   TIMESTAMPTZ NOT NULL
);
