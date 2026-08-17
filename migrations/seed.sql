-- seed.sql: Demo data for design-review platform. Idempotent: uses ON CONFLICT.
-- Run AFTER migrations.

INSERT INTO projects (id, name, description, status, created_at, updated_at, version)
VALUES ('seed-project-1', '首页改版', 'Q4 首页视觉评审', 'active', NOW(), NOW(), 1)
ON CONFLICT (id) DO NOTHING;

INSERT INTO boards (id, project_id, name, width, height, status, created_at, updated_at, version)
VALUES ('seed-board-1', 'seed-project-1', '首页桌面端', 1440, 1024, 'open', NOW(), NOW(), 1)
ON CONFLICT (id) DO NOTHING;

INSERT INTO versions (id, board_id, number, label, preview_key, notes, status, created_by, version)
VALUES ('seed-version-v1', 'seed-board-1', 1, '初稿', 'preview-v1.png', '演示版本', 'superseded', 'seed-user', 1)
ON CONFLICT (id) DO NOTHING;

INSERT INTO versions (id, board_id, number, label, preview_key, notes, status, created_by, version)
VALUES ('seed-version-v2', 'seed-board-1', 2, '修订一版', 'preview-v2.png', '演示版本', 'published', 'seed-user', 1)
ON CONFLICT (id) DO NOTHING;

INSERT INTO memberships (id, project_id, user_id, role, joined_at)
VALUES ('seed-member-1', 'seed-project-1', 'designer-1', 'editor', NOW())
ON CONFLICT (id) DO NOTHING;

INSERT INTO memberships (id, project_id, user_id, role, joined_at)
VALUES ('seed-member-2', 'seed-project-1', 'reviewer-1', 'owner', NOW())
ON CONFLICT (id) DO NOTHING;

INSERT INTO annotations (id, project_id, board_id, version_id, title, body, anchor_type, anchor_x, anchor_y, anchor_w, anchor_h, priority, status, assignee_id, reporter_id, created_at, updated_at, version)
VALUES ('seed-annotation-1', 'seed-project-1', 'seed-board-1', 'seed-version-v1', '标题区颜色对比度不足', '白色文字在浅灰色背景下对比度低于 WCAG AA 标准', 'point', 120, 40, 0, 0, 'high', 'review', 'designer-1', 'reviewer-1', NOW(), NOW(), 3)
ON CONFLICT (id) DO NOTHING;

INSERT INTO annotations (id, project_id, board_id, version_id, title, body, anchor_type, anchor_x, anchor_y, anchor_w, anchor_h, priority, status, assignee_id, reporter_id, created_at, updated_at, version)
VALUES ('seed-annotation-2', 'seed-project-1', 'seed-board-1', 'seed-version-v1', '首屏按钮缺少主操作样式', '主按钮应使用品牌色', 'region', 30, 200, 200, 60, 'normal', 'open', 'designer-1', 'reviewer-1', NOW(), NOW(), 1)
ON CONFLICT (id) DO NOTHING;

INSERT INTO replies (id, annotation_id, author_id, body, created_at)
VALUES ('seed-reply-1', 'seed-annotation-1', 'designer-1', '已改深色，请复核', NOW())
ON CONFLICT (id) DO NOTHING;

INSERT INTO review_rounds (id, project_id, board_id, title, status, conclusion, recommendation, decided_by, decided_at, created_at, updated_at, version)
VALUES ('seed-round-1', 'seed-project-1', 'seed-board-1', '第一轮评审', 'open', '整体方向正确，对比度与按钮主操作样式需修复后通过。', 'approve_with_conditions', 'reviewer-1', NOW(), NOW(), NOW(), 2)
ON CONFLICT (id) DO NOTHING;

INSERT INTO review_snapshots (id, round_id, project_id, board_id, version_id, created_by, counts, created_at)
VALUES ('seed-snapshot-1', 'seed-round-1', 'seed-project-1', 'seed-board-1', 'seed-version-v1', 'reviewer-1', '{"open":1,"replied":0,"review":1,"resolved":0,"closed":0}'::jsonb, NOW())
ON CONFLICT (id) DO NOTHING;
