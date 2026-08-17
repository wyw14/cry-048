-- 0001_init.down.sql: Drop schema in reverse dependency order.

DROP TABLE IF EXISTS idempotency_keys;
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS review_snapshots;
DROP TABLE IF EXISTS review_rounds;
DROP TABLE IF EXISTS attachments;
DROP TABLE IF EXISTS replies;
DROP TABLE IF EXISTS annotations;
DROP TABLE IF EXISTS memberships;
DROP TABLE IF EXISTS versions;
DROP TABLE IF EXISTS boards;
DROP TABLE IF EXISTS projects;
