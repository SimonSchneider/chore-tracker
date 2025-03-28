-- migrate:up
ALTER TABLE chore
    ADD COLUMN chore_type TEXT NOT NULL DEFAULT 'interval';

UPDATE chore
SET chore_type = 'oneshot'
WHERE interval = 0;
