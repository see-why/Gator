-- name: CreateFeed :one
INSERT INTO feeds (id, created_at, updated_at, name, url, user_id)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING *;

-- name: GetFeedsWithUsers :many
SELECT 
    f.id,
    f.created_at,
    f.updated_at,
    f.name,
    f.url,
    f.user_id,
    u.name as user_name
FROM feeds f
JOIN users u ON f.user_id = u.id
ORDER BY f.created_at DESC;

-- name: GetFeedByURL :one
SELECT * FROM feeds WHERE url = $1;

-- name: CreateFeedFollow :one
WITH inserted_feed_follow AS (
    INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
    VALUES ($1, $2, $3, $4, $5)
    RETURNING id, created_at, updated_at, user_id, feed_id
)
SELECT 
    ff.id,
    ff.created_at,
    ff.updated_at,
    ff.user_id,
    ff.feed_id,
    u.name as user_name,
    f.name as feed_name
FROM inserted_feed_follow ff
JOIN users u ON ff.user_id = u.id
JOIN feeds f ON ff.feed_id = f.id;

-- name: GetFeedFollowsForUser :many
SELECT 
    ff.id,
    ff.created_at,
    ff.updated_at,
    ff.user_id,
    ff.feed_id,
    u.name as user_name,
    f.name as feed_name
FROM feed_follows ff
JOIN users u ON ff.user_id = u.id
JOIN feeds f ON ff.feed_id = f.id
WHERE ff.user_id = $1
ORDER BY ff.created_at DESC;

-- name: DeleteFeedFollowByUserAndFeedURL :execrows
DELETE FROM feed_follows 
WHERE feed_follows.user_id = $1 
AND feed_follows.feed_id = (SELECT id FROM feeds WHERE url = $2);

-- name: CreatePost :one
INSERT INTO posts (id, created_at, updated_at, title, url, description, published_at, feed_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (url) DO NOTHING
RETURNING *;

-- name: GetPostsForUser :many
SELECT 
    p.id,
    p.created_at,
    p.updated_at,
    p.title,
    p.url,
    p.description,
    p.published_at,
    p.feed_id,
    f.name as feed_name
FROM posts p
JOIN feeds f ON p.feed_id = f.id
JOIN feed_follows ff ON f.id = ff.feed_id
WHERE ff.user_id = $1
ORDER BY p.published_at DESC NULLS LAST, p.created_at DESC
LIMIT $2 OFFSET $3;

-- name: SearchPostsForUser :many
-- Search posts for a user by fuzzy match against title or description.
-- Params: user_id uuid, q text (search term), limit int, offset int
SELECT
        p.id,
        p.created_at,
        p.updated_at,
        p.title,
        p.url,
        p.description,
        p.published_at,
        p.feed_id,
        f.name as feed_name
FROM posts p
JOIN feeds f ON p.feed_id = f.id
JOIN feed_follows ff ON f.id = ff.feed_id
WHERE ff.user_id = $1
    AND (
        p.title ILIKE ('%' || $2 || '%')
        OR p.description ILIKE ('%' || $2 || '%')
    )
ORDER BY p.published_at DESC NULLS LAST, p.created_at DESC
LIMIT $3 OFFSET $4;
