-- Meal Templates (saved meals)

-- name: CreateMeal :one
INSERT INTO meals (name, protein, carbs, fat, calories)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetMealByName :one
SELECT * FROM meals
WHERE name = ?;

-- name: ListMeals :many
SELECT * FROM meals
ORDER BY name
LIMIT ?;

-- name: UpdateMeal :exec
UPDATE meals
SET protein = ?, carbs = ?, fat = ?, calories = ?
WHERE name = ?;

-- name: DeleteMeal :exec
DELETE FROM meals
WHERE id = ?;

-- Meal Logs (daily tracking)

-- name: LogMeal :one
INSERT INTO meal_logs (meal_id, date)
VALUES (?, ?)
RETURNING *;

-- name: LogMealByName :one
INSERT INTO meal_logs (meal_id, date)
SELECT id, ? FROM meals WHERE name = ?
RETURNING *;

-- name: ListMealLogsByDate :many
SELECT 
  ml.id,
  ml.date,
  ml.created_at,
  m.name,
  m.protein,
  m.carbs,
  m.fat,
  m.calories
FROM meal_logs ml
JOIN meals m ON m.id = ml.meal_id
WHERE ml.date = ?
ORDER BY ml.created_at;

-- name: GetDailyTotals :one
SELECT 
  COALESCE(SUM(m.protein), 0) as total_protein,
  COALESCE(SUM(m.carbs), 0) as total_carbs,
  COALESCE(SUM(m.fat), 0) as total_fat,
  COALESCE(SUM(m.calories), 0) as total_calories
FROM meal_logs ml
JOIN meals m ON m.id = ml.meal_id
WHERE ml.date = ?;

-- name: DeleteMealLog :exec
DELETE FROM meal_logs
WHERE id = ?;
