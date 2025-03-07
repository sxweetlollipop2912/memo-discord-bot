-- name: CreateUser :one
INSERT INTO users (user_id, username, discord_channel_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUser :one
SELECT * FROM users
WHERE user_id = $1;

-- name: UpdateUserDiscordChannel :exec
UPDATE users
SET discord_channel_id = $2
WHERE user_id = $1;

-- name: CreateMemo :one
INSERT INTO memos (discord_user_id, discord_channel_id, content, remind_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListPendingMemos :many
SELECT * FROM memos
WHERE discord_user_id = $1 AND discord_channel_id = $2 AND sent = false
ORDER BY remind_at;

-- name: GetReminderCounts :many
SELECT discord_channel_id, COUNT(*) as count
FROM memos
WHERE discord_user_id = $1 AND sent = false
GROUP BY discord_channel_id;

-- name: GetPendingReminders :many
SELECT *
FROM memos
WHERE sent = false AND remind_at <= $1
ORDER BY remind_at;

-- name: MarkMemoAsSent :exec
UPDATE memos
SET sent = true
WHERE id = $1;

-- name: DeleteMemo :exec
DELETE FROM memos
WHERE id = $1 AND discord_user_id = $2;

-- name: ListAllPendingMemosInChannel :many
SELECT *
FROM memos
WHERE discord_channel_id = $1
  AND remind_at > NOW()
  AND sent = false
ORDER BY remind_at ASC;

-- name: GetMemo :one
SELECT * FROM memos
WHERE id = $1; 