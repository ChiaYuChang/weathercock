-- name: InsertEmbedding :one
INSERT INTO embeddings (
    article_id,
    chunk_id,
    model_id,
    vector
) VALUES (
    $1,
    $2,
    $3,
    @vector::vector
) RETURNING id;

-- name: InsertEmbeddingBatch :batchexec
INSERT INTO embeddings (
    article_id,
    chunk_id,
    model_id,
    vector
) VALUES (
    $1, 
    $2,
    $3,
    @vector::vector
)
ON CONFLICT DO NOTHING
RETURNING id;

-- name: GetAverageEmbeddingByArticleIDs :one
SELECT
    e.article_id,
    e.model_id,
    AVG(e.vector)::vector AS avg_vector
FROM
    embeddings AS e
WHERE
    e.article_id = ANY(@article_ids::integer[])
    AND e.model_id = @model_id::integer
    AND e.vector IS NOT NULL
    AND e.vector <> '[]'::vector
GROUP BY
    e.article_id;

-- name: GetKNNEmbeddingsByL2Distance :many
SELECT
    article_id,
    (vector <-> @query::vector)::float8 AS distance -- <-> is the L2 distance operator in pgvector
FROM
    embeddings
WHERE
    model_id = @model_id::integer
    AND vector IS NOT NULL
    AND vector <> '[]'::vector
ORDER BY
    vector <-> @query
LIMIT @k::integer;

-- name: GetKNNEmbeddingsByCosineSimilarity :many
SELECT
    article_id,
    (vector <=> @query::vector)::float8 AS similarity  -- <=> is the cosine distance operator in pgvector
FROM
    embeddings
WHERE
    model_id = @model_id::integer
    AND vector IS NOT NULL
    AND vector <> '[]'::vector
ORDER BY
    vector <=> @query
LIMIT @k::integer;

-- name: GetKNNEmbeddingsByInnerProduct :many
SELECT
    article_id,
    (vector <#> @query::vector)::float8 AS inner_product  -- <#> is the inner product operator in pgvector
FROM
    embeddings
WHERE
    model_id = @model_id::integer
    AND vector IS NOT NULL
    AND vector <> '[]'::vector
ORDER BY
    vector <#> @query
LIMIT @k::integer;   