-- avg_embedding and users.avg_embedding functions calculate the average embedding vector for a
-- given article and model.
CREATE FUNCTION avg_embedding(aid INTEGER, mid INTEGER) RETURNS VECTOR(1024) LANGUAGE sql AS $$
SELECT CASE
        WHEN COUNT(*) = 0 THEN NULL
        ELSE AVG(vector)
    END
FROM embeddings
WHERE article_id = $1
    AND model_id = $2
    AND vector IS NOT NULL;
$$;
CREATE FUNCTION users.avg_embedding(aid INTEGER, mid INTEGER) RETURNS VECTOR(1024) LANGUAGE sql AS $$
SELECT CASE
        WHEN COUNT(*) = 0 THEN NULL
        ELSE AVG(vector)
    END
FROM users.embeddings
WHERE article_id = $1
    AND model_id = $2
    AND vector IS NOT NULL;
$$;
