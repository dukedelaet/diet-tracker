-- Daily weight tracking
CREATE TABLE weights (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  date TEXT NOT NULL UNIQUE,  -- 'YYYY-MM-DD'
  pounds REAL NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_weights_date ON weights(date);

-- Log individual meals as you eat them
CREATE TABLE meals (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  date TEXT NOT NULL,  -- 'YYYY-MM-DD'
  name TEXT NOT NULL,
  protein INTEGER NOT NULL,
  carbs INTEGER NOT NULL,
  fat INTEGER NOT NULL,
  calories INTEGER NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_meals_date ON meals(date);

-- Exercise log
CREATE TABLE exercises (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  date TEXT NOT NULL,  -- 'YYYY-MM-DD'
  exercise_type TEXT NOT NULL CHECK(exercise_type IN ('cardio', 'strength')),
  duration INTEGER NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_exercises_date ON exercises(date);

-- _, err = db.Exec(`
--     PRAGMA journal_mode = WAL;
--     PRAGMA busy_timeout = 5000;
--     PRAGMA synchronous = NORMAL;
--     PRAGMA cache_size = -64000;
--     PRAGMA foreign_keys = ON;
-- `)
