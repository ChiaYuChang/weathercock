CREATE TABLE IF NOT EXISTS keywords (
    id SERIAL PRIMARY KEY,
    term VARCHAR(32) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS articles_keywords (
    article_id INTEGER NOT NULL REFERENCES public.articles(id) ON DELETE CASCADE,
    keyword_id INTEGER NOT NULL REFERENCES keywords(id) ON DELETE CASCADE,
    PRIMARY KEY (keyword_id, article_id)
);

CREATE TABLE IF NOT EXISTS users.articles_keywords (
    article_id INTEGER NOT NULL REFERENCES users.articles(id) ON DELETE CASCADE,
    keyword_id INTEGER NOT NULL REFERENCES keywords(id) ON DELETE CASCADE,
    PRIMARY KEY (keyword_id, article_id)
);

