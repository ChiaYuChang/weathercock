CREATE TYPE party AS ENUM (
    'none', -- Not a party press release
    'KMT',  -- Kuomintang (國民黨)
    'DPP',  -- Democratic Progressive Party (民主進步黨)
    'TPP'   -- Taiwan People's Party (台灣民眾黨)
);

CREATE TABLE models (
    id          SERIAL      PRIMARY KEY,
    "name"      TEXT        NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE articles (
    id               SERIAL      PRIMARY KEY,
    title            TEXT        NOT NULL,
    "url"            TEXT        NOT NULL,
    source           TEXT        NOT NULL,       -- 聯合報, 自由時報, TVBS, etc.
    md5              TEXT        UNIQUE NOT NULL,
    party            party       NOT NULL DEFAULT 'none',
    content          TEXT        NOT NULL,        -- full article text
    cuts             INTEGER[]   NOT NULL DEFAULT '{}',
    published_at     TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at       TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Chunks table using offsets into the article content
CREATE TABLE chunks (
    id            SERIAL  PRIMARY KEY,
    article_id    INTEGER NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    "start"       INTEGER NOT NULL,  -- start index of the chunk (including left overlap)
    offset_left   INTEGER NOT NULL,  -- start index of unique content in this chunk (relative to start)
    offset_right  INTEGER NOT NULL,  -- end index of unique content in this chunk (relative to start)
    "end"         INTEGER NOT NULL,  -- end index of the chunk (including right overlap)
    created_at    TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(article_id, "start", "end")
);

CREATE TABLE embeddings (
    id         SERIAL       PRIMARY KEY,
    article_id INTEGER      NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    chunk_id   INTEGER      NOT NULL REFERENCES chunks(id) ON DELETE CASCADE,
    model_id   INTEGER      NOT NULL REFERENCES models(id) ON DELETE CASCADE,
    vector     VECTOR(1024) NOT NULL,
    created_at TIMESTAMPTZ  DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(article_id, chunk_id, model_id, vector)
);


CREATE TABLE users.articles (
    id            SERIAL      PRIMARY KEY,
    task_id       UUID        NOT NULL REFERENCES users.tasks(task_id) ON DELETE CASCADE,
    title         TEXT        NOT NULL DEFAULT 'undefined',
    "url"         TEXT        NOT NULL DEFAULT 'local',
    source        TEXT        NOT NULL DEFAULT 'user',  -- 聯合報, 自由時報, TVBS, 使用者輸入, etc.
    md5           TEXT        UNIQUE NOT NULL,
    content       TEXT        NOT NULL DEFAULT '',       -- full article text
    cuts          INTEGER[]   NOT NULL DEFAULT '{}',
    published_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at    TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE users.chunks (
    id            SERIAL  PRIMARY KEY,
    article_id    INTEGER NOT NULL REFERENCES users.articles(id) ON DELETE CASCADE,
    "start"       INTEGER NOT NULL,  -- start index of the chunk (including left overlap)
    offset_left   INTEGER NOT NULL,  -- start index of unique content in this chunk (relative to start)
    offset_right  INTEGER NOT NULL,  -- end index of unique content in this chunk (relative to start)
    "end"         INTEGER NOT NULL,  -- end index of the chunk (including right overlap)
    created_at    TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(article_id, "start", "end")
);

CREATE TABLE users.embeddings (
    id         SERIAL       PRIMARY KEY,
    article_id INTEGER      NOT NULL REFERENCES users.articles(id) ON DELETE CASCADE,
    chunk_id   INTEGER      NOT NULL REFERENCES users.chunks(id) ON DELETE CASCADE,
    model_id   INTEGER      NOT NULL REFERENCES models(id) ON DELETE CASCADE,
    vector     VECTOR(1024) NOT NULL,
    created_at TIMESTAMPTZ  DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(article_id, chunk_id, model_id, vector)
);