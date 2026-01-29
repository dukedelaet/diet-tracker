-- +goose Up
-- +goose StatementBegin
ALTER TABLE meal_logs ADD COLUMN protein INTEGER NOT NULL DEFAULT 0;
ALTER TABLE meal_logs ADD COLUMN carbs INTEGER NOT NULL DEFAULT 0;
ALTER TABLE meal_logs ADD COLUMN fat INTEGER NOT NULL DEFAULT 0;
ALTER TABLE meal_logs ADD COLUMN calories INTEGER NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE meal_logs DROP COLUMN protein;
ALTER TABLE meal_logs DROP COLUMN carbs;
ALTER TABLE meal_logs DROP COLUMN fat;
ALTER TABLE meal_logs DROP COLUMN calories;
-- +goose StatementEnd
