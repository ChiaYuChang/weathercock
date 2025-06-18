-- avg_embedding and users.avg_embedding functions calculate the average embedding vector for a
-- given article and model.
CREATE FUNCTION avg_embedding(aid INTEGER, mid INTEGER) RETURNS VECTOR(1024) LANGUAGE sql AS $$
SELECT
    CASE
        WHEN COUNT(*) = 0 THEN NULL
        ELSE AVG(vector)
    END AS vector
FROM
    embeddings AS e
WHERE
    e.article_id = aid
    AND e.model_id = mid
    AND e.vector IS NOT NULL;
$$;

CREATE FUNCTION users.avg_embedding(aid INTEGER, mid INTEGER) RETURNS VECTOR(1024) LANGUAGE sql AS $$
SELECT CASE
        WHEN COUNT(*) = 0 THEN NULL
        ELSE AVG(vector)
    END
FROM users.embeddings AS e
WHERE
    e.article_id = aid
    AND e.model_id = mid
    AND vector IS NOT NULL;
$$;
