-- name: CreateExercise :one
INSERT INTO exercises (date, exercise_type, duration)
VALUES (?, ?, ?)
RETURNING *;

-- name: ListExercisesByDate :many
SELECT * FROM exercises
WHERE date = ?
ORDER BY created_at;

-- name: ListExercises :many
SELECT * FROM exercises
ORDER BY date DESC
LIMIT ?;

-- name: ListExercisesFromDate :many
SELECT * FROM exercises
WHERE date >= ?
ORDER BY date ASC;

-- name: DeleteExercise :exec
DELETE FROM exercises
WHERE id = ?;
