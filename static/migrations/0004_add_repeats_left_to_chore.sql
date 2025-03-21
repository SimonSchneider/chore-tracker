-- migrate:up
ALTER TABLE chore
    ADD COLUMN repeats_left INTEGER NOT NULL DEFAULT -1;
