-- name: CreateLike :one
INSERT INTO likes (id, created_at, updated_at, user_id, post_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: DeleteLike :execrows
DELETE FROM likes 
WHERE user_id = $1 AND post_id = $2;

-- name: GetLikesForUser :many
SELECT 
    l.id as like_id,
    l.created_at as liked_at,
    p.id,
    p.created_at,
    p.updated_at,
    p.title,
    p.url,
    p.description,
    p.published_at,
    p.feed_id,
    f.name as feed_name
FROM likes l
JOIN posts p ON l.post_id = p.id
JOIN feeds f ON p.feed_id = f.id
WHERE l.user_id = $1
ORDER BY l.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetLikeByUserAndPost :one
SELECT * FROM likes WHERE user_id = $1 AND post_id = $2;

-- name: GetLikeCountForPost :one
SELECT COUNT(*) FROM likes WHERE post_id = $1;
