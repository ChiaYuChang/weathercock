CREATE TYPE task_status AS ENUM (
    'pending',     -- Task is pending
    'processing',  -- Task is currently being processed
    'done',        -- Task has been completed successfully
    'failed'       -- Task has failed
);

CREATE SCHEMA IF NOT EXISTS users;

-- TODO for multiple user systems
-- CREATE TABLE IF NOT EXISTS users.info (
--     id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
--     username   TEXT        NOT NULL UNIQUE,
--     email      TEXT        NOT NULL UNIQUE,
--     created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
-- );

CREATE TYPE source_type AS ENUM ('url', 'text');

CREATE TABLE users.tasks (
    id             SERIAL      PRIMARY KEY,
    task_id        UUID        NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    -- user_id        UUID        NOT NULL REFERENCES users.info(id) ON DELETE CASCADE,
    source         source_type NOT NULL,
    original_input TEXT        NOT NULL,
    status         task_status NOT NULL DEFAULT 'pending',
    error_message  TEXT,
    created_at     TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
); 