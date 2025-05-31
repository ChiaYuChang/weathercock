-- Create keyword table
CREATE TABLE IF NOT EXISTS keyword (
    id SERIAL PRIMARY KEY,
    name VARCHAR(32) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create keyword_article table to establish many-to-many relationship
CREATE TABLE IF NOT EXISTS keyword_article (
    keyword_id INTEGER NOT NULL REFERENCES keyword(id) ON DELETE CASCADE,
    article_id INTEGER NOT NULL REFERENCES article(id) ON DELETE CASCADE,
    PRIMARY KEY (keyword_id, article_id)
);