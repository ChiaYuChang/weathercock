-- Reset the sequence for embedding table
ALTER SEQUENCE embedding_id_seq RESTART WITH 1;

-- Drop the embedding table
DROP TABLE IF EXISTS embedding;

-- Reset the sequence for model table
ALTER SEQUENCE model_id_seq RESTART WITH 1;

-- Drop the model table
DROP TABLE IF EXISTS model;

-- reset the sequence for article table
ALTER SEQUENCE article_id_seq RESTART WITH 1;

-- Drop the article table
DROP TABLE IF EXISTS article;

-- Drop the party type
DROP TYPE IF EXISTS party;