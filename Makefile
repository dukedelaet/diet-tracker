.PHONY: run install inspect

run:
	@go run ./cmd/diet $(ARGS)

install:
	@mkdir -p "$$HOME/go/bin"
	@go build -o "$$HOME/go/bin/diet" ./cmd/diet
	@go build -o "$$HOME/go/bin/diet-tui" ./cmd/diet-tui
	@echo "Installed diet and diet-tui to ~/go/bin"

inspect:
	@echo "opening prod db"
	@sqlite3 "$${XDG_DATA_HOME:-$$HOME/.local/share}/diet-tracker/app.db"
