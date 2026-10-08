APP_NAME := service-share-link
MAIN_PKG := ./cmd
BIN_DIR  := bin

ifeq ($(OS),Windows_NT)
	BIN       := $(BIN_DIR)/$(APP_NAME).exe
	RM_BIN    := powershell -NoProfile -Command "Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $(BIN_DIR)"
else
	BIN       := $(BIN_DIR)/$(APP_NAME)
	RM_BIN    := rm -rf $(BIN_DIR)
endif

# Same precedence as the app: environment first, then .env, then the default.
ifneq ($(origin DATABASE_URL),environment)
-include .env
endif
DATABASE_URL   ?= postgres://postgres:postgres@localhost:5432/share_link?sslmode=disable
MIGRATIONS_DIR := migrations
MIGRATE        := migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)"

.DEFAULT_GOAL := run
.PHONY: run build start test vet fmt tidy clean \
	migrate-up migrate-down migrate-version migrate-force migrate-create

# Run from source (config is read from .env in the project root)
run:
	go run $(MAIN_PKG)

# Compile the binary into bin/
build:
	go build -o $(BIN) $(MAIN_PKG)

# Build, then run the compiled binary
start: build
	$(BIN)

test:
	go test ./...

vet:
	go vet ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy

clean:
	go clean
	$(RM_BIN)

# Migrations need the golang-migrate CLI:
#   go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@latest
# The service also applies pending up migrations on start.

# Apply all pending up migrations
migrate-up:
	$(MIGRATE) up

# Roll back the last `steps` migrations (default 1)
steps ?= 1
migrate-down:
	$(MIGRATE) down $(steps)

# Print the current version and whether it is dirty
migrate-version:
	$(MIGRATE) version

# Mark a version as applied without running it, to recover from a dirty state
migrate-force:
	$(if $(version),,$(error usage: make migrate-force version=<n>))
	$(MIGRATE) force $(version)

# Create the next up/down pair, e.g. make migrate-create name=add_expires_at
migrate-create:
	$(if $(name),,$(error usage: make migrate-create name=<name>))
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(name)
