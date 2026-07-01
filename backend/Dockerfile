FROM golang:1.25-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG MIGRATE_VERSION=v4.18.3
RUN wget -qO- "https://github.com/golang-migrate/migrate/releases/download/${MIGRATE_VERSION}/migrate.linux-amd64.tar.gz" \
    | tar -xz -C /usr/local/bin/ migrate
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /build/server .
COPY --from=builder /usr/local/bin/migrate /usr/local/bin/migrate
COPY --from=builder /build/migrations ./migrations/
EXPOSE 8080
CMD ["./server"]
