-- name: CreateWeight :one
INSERT INTO weights (date, pounds)
VALUES (?, ?)
RETURNING *;

-- name: GetWeightByDate :one
SELECT * FROM weights
WHERE date = ?;

-- name: ListWeights :many
SELECT * FROM weights
ORDER BY date DESC
LIMIT ?;

-- name: DeleteWeight :exec
DELETE FROM weights
WHERE id = ?;
