-- +goose Up
-- +goose StatementBegin
ALTER TABLE meals ADD COLUMN last_logged TEXT NOT NULL DEFAULT '';
UPDATE meals SET last_logged = created_at WHERE last_logged = '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE meals DROP COLUMN last_logged;
-- +goose StatementEnd
