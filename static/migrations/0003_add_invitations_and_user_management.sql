-- migrate:up
CREATE TABLE IF NOT EXISTS invitation
(
    id            TEXT    NOT NULL PRIMARY KEY,
    created_at    INTEGER NOT NULL,
    expires_at    INTEGER NOT NULL,
    chore_list_id TEXT,
    created_by    TEXT    NOT NULL,
    FOREIGN KEY (chore_list_id) REFERENCES chore_list (id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES user (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS user
(
    id         TEXT    NOT NULL PRIMARY KEY,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE password_auth
(
    user_id  TEXT NOT NULL PRIMARY KEY,
    username TEXT NOT NULL,
    hash     TEXT NOT NULL,
    FOREIGN KEY (user_id) REFERENCES user (id) ON DELETE CASCADE,
    UNIQUE (username)
);

CREATE TABLE tokens
(
    user_id    TEXT    NOT NULL,
    token      TEXT    NOT NULL PRIMARY KEY,
    expires_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES user (id) ON DELETE CASCADE
);

CREATE TABLE chore_list
(
    id         TEXT    NOT NULL PRIMARY KEY,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    name       TEXT    NOT NULL
);

CREATE TABLE chore_list_members
(
    chore_list_id TEXT NOT NULL,
    user_id       TEXT NOT NULL,
    FOREIGN KEY (chore_list_id) REFERENCES chore_list (id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES user (id) ON DELETE CASCADE,
    PRIMARY KEY (chore_list_id, user_id)
);

DROP TABLE chore_event;
DROP TABLE chore;

CREATE TABLE chore
(
    id              TEXT    NOT NULL PRIMARY KEY,
    name            TEXT    NOT NULL,
    interval        INTEGER NOT NULL,
    last_completion INTEGER NOT NULL DEFAULT 0,
    snoozed_for     INTEGER NOT NULL DEFAULT 0,
    created_at      INTEGER NOT NULL DEFAULT 0,
    chore_list_id   TEXT    NOT NULL,
    created_by      TEXT    NOT NULL,
    FOREIGN KEY (chore_list_id) REFERENCES chore_list (id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES user (id) ON DELETE CASCADE
);

CREATE TABLE chore_event
(
    id          TEXT    NOT NULL PRIMARY KEY,
    chore_id    TEXT    NOT NULL,
    occurred_at INTEGER NOT NULL,
    event_type  TEXT    NOT NULL,
    FOREIGN KEY (chore_id) REFERENCES chore (id) ON DELETE CASCADE
);
