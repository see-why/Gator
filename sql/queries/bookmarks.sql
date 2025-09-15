-- name: CreateBookmark :one
INSERT INTO bookmarks (id, created_at, updated_at, user_id, post_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: DeleteBookmark :execrows
DELETE FROM bookmarks 
WHERE user_id = $1 AND post_id = $2;

-- name: GetBookmarksForUser :many
SELECT 
    b.id as bookmark_id,
    b.created_at as bookmarked_at,
    p.id,
    p.created_at,
    p.updated_at,
    p.title,
    p.url,
    p.description,
    p.published_at,
    p.feed_id,
    f.name as feed_name
FROM bookmarks b
JOIN posts p ON b.post_id = p.id
JOIN feeds f ON p.feed_id = f.id
WHERE b.user_id = $1
ORDER BY b.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetPostByID :one
SELECT * FROM posts WHERE id = $1;
