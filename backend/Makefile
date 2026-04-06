.PHONY: build run test lint clean migrate-up migrate-down migrate-create migrate-force

MIGRATE_URL ?= postgres://chesed:chesed@localhost:5432/chesed?sslmode=disable

build:
	go build -o bin/server ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/ tmp/

migrate-up:
	migrate -path migrations -database "$(MIGRATE_URL)" up

migrate-down:
	migrate -path migrations -database "$(MIGRATE_URL)" down 1

migrate-create:
	migrate create -ext sql -dir migrations -seq $(name)

migrate-force:
	migrate -path migrations -database "$(MIGRATE_URL)" force $(version)
