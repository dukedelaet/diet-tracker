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
ORDER BY last_logged DESC, created_at DESC, name
LIMIT ?;

-- name: UpdateMeal :exec
UPDATE meals
SET protein = ?, carbs = ?, fat = ?, calories = ?
WHERE name = ?;

-- name: UpdateMealLastLogged :exec
UPDATE meals
SET last_logged = ?
WHERE name = ?;

-- name: DeleteMeal :exec
DELETE FROM meals
WHERE id = ?;

-- Meal Logs (daily tracking)

-- Log meal with saved portions (copies from meals table)
-- name: LogMeal :one
INSERT INTO meal_logs (meal_id, protein, carbs, fat, calories, date)
SELECT id, protein, carbs, fat, calories, ? 
FROM meals 
WHERE name = ?
RETURNING *;

-- name: LogMealWithPortions :one
INSERT INTO meal_logs (meal_id, protein, carbs, fat, calories, date)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListMealLogsByDate :many
SELECT 
  ml.id,
  ml.date,
  ml.created_at,
  ml.protein,
  ml.carbs,
  ml.fat,
  ml.calories,
  m.name
FROM meal_logs ml
JOIN meals m ON m.id = ml.meal_id
WHERE ml.date = ?
ORDER BY ml.created_at;

-- name: GetDailyTotals :one
SELECT 
  COALESCE(SUM(protein), 0) as total_protein,
  COALESCE(SUM(carbs), 0) as total_carbs,
  COALESCE(SUM(fat), 0) as total_fat,
  COALESCE(SUM(calories), 0) as total_calories
FROM meal_logs
WHERE date = ?;

-- name: DeleteMealLog :exec
DELETE FROM meal_logs
WHERE id = ?;

-- name: ListDailyCaloriesFromDate :many
SELECT
  date,
  COALESCE(SUM(calories), 0) as total_calories
FROM meal_logs
WHERE date >= ?
GROUP BY date
ORDER BY date ASC;
