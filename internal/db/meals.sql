-- name: CreateMeal :one
INSERT INTO meals (date, name, protein, carbs, fat, calories)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListMealsByDate :many
SELECT * FROM meals
WHERE date = ?
ORDER BY created_at;

-- name: GetDailyTotals :one
SELECT 
  SUM(protein) as total_protein,
  SUM(carbs) as total_carbs,
  SUM(fat) as total_fat,
  SUM(calories) as total_calories
FROM meals
WHERE date = ?;
