CREATE TYPE party AS ENUM (
    'none', -- Not a party press release
    'KMT'   -- Kuomintang (國民黨)
    'DPP',  -- Democratic Progressive Party (民主進步黨)
    'TPP'   -- Taiwan People's Party (台灣民眾黨)
);

CREATE TABLE article (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    url TEXT NOT NULL,
    source TEXT NOT NULL,
    md5 TEXT UNIQUE NOT NULL,
    content TEXT[] NOT NULL,
    author TEXT NOT NULL,
    party party NOT NULL DEFAULT 'none',
    published_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE model (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT, 
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE embedding (
    id SERIAL PRIMARY KEY,
    article_id INTEGER NOT NULL REFERENCES article(id) ON DELETE CASCADE,
    model_id INTEGER NOT NULL REFERENCES model(id) ON DELETE CASCADE,
    vector VECTOR(1024) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(article_id, vector)
);

