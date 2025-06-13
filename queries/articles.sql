-- name: CreateUserArticle :one
INSERT INTO users.articles (
    task_id,
    title,
    "url",
    source,
    md5,
    content,
    paragraph_starts,
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

-- name: CreateUserChunk :one
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

-- name: CreateArticle :one
INSERT INTO articles (
    title,
    "url",
    source,
    md5,
    party,
    content,
    paragraph_starts,
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

-- name: CreateChunk :one
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