-- migrate:up
ALTER TABLE chore
    ADD COLUMN
        date_glob TEXT;
