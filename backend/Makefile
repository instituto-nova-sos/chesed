.PHONY: build run test test-integration test-all lint clean migrate-up migrate-down migrate-create migrate-force migrate-up-prod bench security-scan

# Pin the Go toolchain to the version declared in go.mod (and the production
# Dockerfile). golangci-lint embeds a go/types typechecker built against the Go
# version it was compiled with; running it under a newer system toolchain makes
# it emit spurious "undefined" / "requires newer Go version" typecheck errors.
# Pinning here keeps lint/build/test consistent with what actually ships.
export GOTOOLCHAIN ?= go1.25.5

MIGRATE_URL ?= postgres://chesed:chesed@localhost:5432/chesed?sslmode=disable

build:
	go build -o bin/server ./cmd/server

run:
	go run ./cmd/server

# Unit tests only (fast, no Docker required).
test:
	go test -short ./...

# Integration tests — boot a real Postgres container via testcontainers-go,
# apply all migrations, exercise the HTTP stack end-to-end. Requires Docker
# Desktop running. On macOS we explicitly point at the Docker Desktop socket
# because testcontainers does not auto-discover it.
test-integration:
	DOCKER_HOST=$${DOCKER_HOST:-unix://$$HOME/.docker/run/docker.sock} \
	TESTCONTAINERS_RYUK_DISABLED=true \
	go test -tags integration -timeout 600s ./internal/integration/...

# Full suite: unit + integration. Use before pushing.
test-all: test test-integration

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

migrate-up-prod:
	docker compose -f ../docker-compose.prod.yml run --rm migrate

# Hot-path benchmarks (S12.5 / RNF-08). No network — per-op latency/allocs for
# the public campaign listing and the sync push batch. Compiles and runs the
# Benchmark* functions under the service package.
bench:
	go test -run '^$$' -bench=. -benchmem ./internal/service/...

# Static security analysis + HTTP security-header assertion (S12.6). gosec keeps
# injection (G201/G202) and secret (G101) rules enabled. Both steps are gated on
# their tool being installed and print a NEEDS-<tool> notice otherwise, mirroring
# the Docker-gated test-integration step.
security-scan:
	@if command -v gosec >/dev/null 2>&1; then \
	  gosec -conf .gosec.json -severity medium -confidence medium -quiet ./... ; \
	else \
	  echo "security-scan: SKIPPED-NEEDS-GOSEC (go install github.com/securego/gosec/v2/cmd/gosec@latest)" ; \
	fi
	@bash ../scripts/security-headers-check.sh "$${SECURITY_CHECK_URL:-http://localhost:8080/api/v1/health}" || \
	  echo "security-scan: header check skipped/failed (needs a running API at SECURITY_CHECK_URL)"
