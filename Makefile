.PHONY: run install inspect

run:
	@go run ./cmd/diet $(ARGS)

install:
	@go install ./cmd/diet ./cmd/tui
	@echo "Installed latest binaries to ~/go/bin"

inspect:
	@echo "opening prod db"
	@sqlite3 "$${XDG_DATA_HOME:-$$HOME/.local/share}/diet-tracker/app.db"
