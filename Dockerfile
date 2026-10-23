# Build stage
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./ 
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -o main cmd/rest/main.go

# Run stage
FROM alpine:3.24
WORKDIR /app
COPY --from=builder /app/main .
COPY app.env wait-for.sh ./ 
COPY db/migration ./db/migration
ENTRYPOINT [ "/app/main" ]
