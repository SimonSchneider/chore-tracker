-- migrate:up
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS chore
(
    id              TEXT    NOT NULL PRIMARY KEY,
    name            TEXT    NOT NULL,
    interval        INTEGER NOT NULL,
    last_completion INTEGER NOT NULL DEFAULT 0,
    snoozed_for     INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS chore_event
(
    id          TEXT    NOT NULL PRIMARY KEY,
    chore_id    TEXT    NOT NULL,
    occurred_at INTEGER NOT NULL,
    FOREIGN KEY (chore_id) REFERENCES chore (id) ON DELETE CASCADE
);
