-- migrate:up
-- Drop the csrf_token column from tokens table
-- SQLite doesn't support DROP COLUMN directly, so we need to recreate the table
CREATE TABLE tokens_new
(
    user_id    TEXT    NOT NULL,
    token      TEXT    NOT NULL PRIMARY KEY,
    expires_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES user (id) ON DELETE CASCADE
);

INSERT INTO tokens_new (user_id, token, expires_at)
SELECT user_id, token, expires_at FROM tokens;

DROP TABLE tokens;

ALTER TABLE tokens_new RENAME TO tokens;

