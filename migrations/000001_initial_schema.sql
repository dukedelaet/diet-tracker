-- +goose Up

-- Daily weight tracking
CREATE TABLE weights (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  date TEXT NOT NULL UNIQUE,
  pounds INTEGER NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_weights_date ON weights(date);

-- Meal templates (saved recipes/meals)
CREATE TABLE meals (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,  -- "Chicken & Rice", "Protein Shake", etc
  protein INTEGER NOT NULL,
  carbs INTEGER NOT NULL,
  fat INTEGER NOT NULL,
  calories INTEGER NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_meals_name ON meals(name);

-- Log when you actually eat a meal
CREATE TABLE meal_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  meal_id INTEGER NOT NULL,
  date TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  FOREIGN KEY (meal_id) REFERENCES meals(id) ON DELETE CASCADE
);
CREATE INDEX idx_meal_logs_date ON meal_logs(date);

-- Exercise log
CREATE TABLE exercises (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  date TEXT NOT NULL,
  exercise_type TEXT NOT NULL CHECK(exercise_type IN ('cardio', 'strength')),
  duration INTEGER NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_exercises_date ON exercises(date);

-- +goose Down
DROP TABLE IF EXISTS meal_logs;
DROP TABLE IF EXISTS exercises;
DROP TABLE IF EXISTS meals;
DROP TABLE IF EXISTS weights;
