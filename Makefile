.PHONY: run install inspect

DIET_DB_PATH=./app.db

run:
	@DIET_DB_PATH=$(DIET_DB_PATH) go run ./cmd/diet $(ARGS)

install:
	@go install ./cmd/diet
	@echo "Installed latest version to Application Support"

inspect:
	@echo "opening prod db"
	@sqlite3 ~/Library/Application\ Support/diet-tracker/app.db
