-- migrate:up
ALTER TABLE chore
    ADD COLUMN
        created_at INTEGER NOT NULL DEFAULT 0;

UPDATE chore
SET created_at = (SELECT MIN(e.occurred_at)
                  FROM chore_event e
                  WHERE e.chore_id = chore.id);
