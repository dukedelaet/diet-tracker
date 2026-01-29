-- +goose Up
-- Backfill existing meal logs with values from meals table
UPDATE meal_logs
SET 
  protein = (SELECT protein FROM meals WHERE meals.id = meal_logs.meal_id),
  carbs = (SELECT carbs FROM meals WHERE meals.id = meal_logs.meal_id),
  fat = (SELECT fat FROM meals WHERE meals.id = meal_logs.meal_id),
  calories = (SELECT calories FROM meals WHERE meals.id = meal_logs.meal_id)
WHERE protein = 0 AND carbs = 0 AND fat = 0 AND calories = 0;

-- +goose Down
-- Cannot undo this - data would be lost
