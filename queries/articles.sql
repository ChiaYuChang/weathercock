-- name: InsertUserArticle :one
INSERT INTO users.articles (
    task_id,
    title,
    "url",
    source,
    md5,
    content,
    cuts,
    published_at
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8
) RETURNING id;

-- name: InsertUserChunk :one
INSERT INTO users.chunks (
    article_id,
    "start",
    offset_left,
    offset_right,
    "end"
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
) RETURNING id;

-- name: InsertUserChunksBatch :batchexec
INSERT INTO users.chunks (
    article_id,
    "start",
    offset_left,
    offset_right,
    "end"
) VALUES (
    $1, 
    $2,
    $3,
    $4,
    $5
) 
ON CONFLICT DO NOTHING
RETURNING id;


-- name: ExtractUserChunks :many
SELECT
    c.article_id AS article_id,
    c.id AS chunk_id,
    substring(
        a.content 
        FROM c."start" + 1
        FOR (c."end" - c."start")
    ) AS content
FROM
    users.articles AS a
JOIN
    users.chunks AS c
ON 
    a.id = c.article_id
WHERE
    a.id = $1
ORDER BY
    c."start";

-- name: GetUserArticleByID :one
SELECT
    *
FROM
    users.articles
WHERE
    id = $1;

-- name: GetUserArticleByTaskID :one
SELECT
    *
FROM
    users.articles
WHERE
    task_id = $1;

-- name: GetUserArticleByMD5 :one
SELECT
    *
FROM
    users.articles
WHERE
    md5 = $1;


-- name: InsertArticle :one
INSERT INTO articles (
    title,
    "url",
    source,
    md5,
    party,
    content,
    cuts,
    published_at
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8
) RETURNING id;

-- name: GetArticleByID :one
SELECT
    *
FROM
    articles
WHERE
    id = $1;

-- name: GetArticleByMD5 :one
SELECT
    *
FROM
    articles
WHERE
    md5 = $1;

-- name: GetArticleByURL :one
SELECT
    *
FROM
    articles
WHERE
    "url" = $1
ORDER BY
    published_at DESC
LIMIT 1;

-- name: GetArticleWithinTimeInterval :many
SELECT
    *
FROM
    articles
WHERE
    published_at BETWEEN sqlc.arg('start') AND sqlc.arg('end')
ORDER BY
    published_at DESC
LIMIT
    sqlc.arg('limit')::integer;

-- name: GetArticlesInPastKDays :many
SELECT
    *
FROM
    articles
WHERE
    published_at >= NOW() - INTERVAL '1 day' * sqlc.arg(k)::integer
ORDER BY
    published_at DESC
LIMIT 
    sqlc.arg('limit')::integer;

-- name: InsertChunk :one
INSERT INTO chunks (
    article_id,
    "start",
    offset_left,
    offset_right,
    "end"
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
) RETURNING id;

-- name: InsertChunksBatch :batchexec
INSERT INTO users.chunks (
    article_id,
    "start",
    offset_left,
    offset_right,
    "end"
) VALUES (
    $1, 
    $2,
    $3,
    $4,
    $5
) 
ON CONFLICT DO NOTHING
RETURNING id;

-- name: ExtractChunks :many
SELECT
    c.article_id AS article_id,
    c.id AS chunk_id,
    substring(
        a.content 
        FROM c."start" + 1
        FOR (c."end" - c."start")
    ) AS content
FROM
    articles AS a
JOIN
    chunks AS c
ON 
    a.id = c.article_id
WHERE
    a.id = $1
ORDER BY
    c."start";