-- Reset the sequence for embedding table
ALTER SEQUENCE users.embeddings_id_seq RESTART WITH 1;
ALTER SEQUENCE embeddings_id_seq RESTART WITH 1;

-- Drop the embedding table
DROP TABLE IF EXISTS users.embeddings;
DROP TABLE IF EXISTS embeddings;

-- Reset the sequence for chunk table
ALTER SEQUENCE users.chunks_id_seq RESTART WITH 1;
ALTER SEQUENCE chunks_id_seq RESTART WITH 1;
-- Drop the chunk table
DROP TABLE IF EXISTS users.chunks;
DROP TABLE IF EXISTS chunks;

-- reset the sequence for article table
ALTER SEQUENCE users.articles_id_seq RESTART WITH 1;
ALTER SEQUENCE articles_id_seq RESTART WITH 1;

-- Drop the article table
DROP TABLE IF EXISTS users.articles;
DROP TABLE IF EXISTS articles;

-- Reset the sequence for model table
ALTER SEQUENCE models_id_seq RESTART WITH 1;

DROP SCHEMA IF EXISTS users;

-- Drop the model table
DROP TABLE IF EXISTS models;

-- Drop the party type
DROP TYPE IF EXISTS party;