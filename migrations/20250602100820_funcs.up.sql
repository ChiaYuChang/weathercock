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
-- none_overlap_chunk and users.none_overlap_chunk functions return non-overlapping chunks of an article.
-- They extract the content from the chunk based on start_at and end_at indices.
CREATE FUNCTION none_overlap_chunk(aid INTEGER) RETURNS TABLE(chunk_id INTEGER, ord INTEGER, content TEXT) LANGUAGE sql AS $$
SELECT c.id AS chunk_id,
    c.ord AS ord,
    substring(
        c.content
        FROM c.start_at + 1 FOR c.end_at - c.start_at
    ) AS content
FROM chunks c
WHERE c.article_id = $1
    AND character_length(c.content) > 0
ORDER BY c.ord;
$$;
CREATE FUNCTION users.none_overlap_chunk(aid INTEGER) RETURNS TABLE(chunk_id INTEGER, ord INTEGER, content TEXT) LANGUAGE sql AS $$
SELECT c.id AS chunk_id,
    c.ord AS ord,
    substring(
        c.content
        FROM c.start_at + 1 FOR c.end_at - c.start_at
    ) AS content
FROM users.chunks c
WHERE c.article_id = $1
    AND character_length(c.content) > 0
ORDER BY c.ord;
$$;
-- concat_article_chunks and users.concat_article_chunks functions concatenate the content of all chunks
-- of an article into a single text string, ordered by the chunk's ord value.
CREATE FUNCTION concat_article_chunks(aid INTEGER) RETURNS TEXT LANGUAGE sql AS $$
SELECT STRING_AGG(
        substring(
            c.content
            FROM c.start_at + 1 FOR c.end_at - c.start_at
        ),
        '\n'
        ORDER BY c.ord
    )
FROM chunks c
WHERE c.article_id = $1
    AND character_length(c.content) > 0;
$$;
CREATE FUNCTION users.concat_article_chunks(aid INTEGER) RETURNS TEXT LANGUAGE sql AS $$
SELECT STRING_AGG(
        substring(
            c.content
            FROM c.start_at + 1 FOR c.end_at - c.start_at
        ),
        '\n'
        ORDER BY c.ord
    )
FROM users.chunks c
WHERE c.article_id = $1
    AND character_length(c.content) > 0;
$$;