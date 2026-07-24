FROM golang:1.26-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
RUN go install github.com/a-h/templ/cmd/templ@latest \
    && go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
COPY . .
RUN sqlc generate && templ generate
RUN CGO_ENABLED=1 go build -o bin/job-search-pipeline ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/bin/job-search-pipeline .
COPY --from=builder /app/static ./static
EXPOSE 8080
CMD ["./job-search-pipeline"]
