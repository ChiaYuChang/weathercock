-- Drop keyword_article table
DROP TABLE IF EXISTS keyword_article;

-- Reset the sequence for keyword table
ALTER SEQUENCE keyword_id_seq RESTART WITH 1;

-- Drop the keyword table
DROP TABLE IF EXISTS keyword;