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
INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING 
    feed_follows.id,
    feed_follows.created_at,
    feed_follows.updated_at,
    feed_follows.user_id,
    feed_follows.feed_id,
    (SELECT name FROM users WHERE id = feed_follows.user_id) as user_name,
    (SELECT name FROM feeds WHERE id = feed_follows.feed_id) as feed_name;

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
