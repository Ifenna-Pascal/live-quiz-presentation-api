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
# Settings come from environment variables (or an app.env mounted/passed at
# run time); app.env is git-ignored, so it is not baked into the image.
COPY wait-for.sh ./
COPY db/migration ./db/migration
ENTRYPOINT [ "/app/main" ]
