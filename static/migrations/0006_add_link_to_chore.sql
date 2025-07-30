-- migrate:up
ALTER TABLE chore
    ADD COLUMN link TEXT;
