-- Drop keyword_article table
DROP TABLE IF EXISTS articles_keywords;
DROP TABLE IF EXISTS users.articles_keywords;

-- Reset the sequence for keyword table
ALTER SEQUENCE keywords_id_seq RESTART WITH 1;

-- Drop the keyword table
DROP TABLE IF EXISTS keywords;