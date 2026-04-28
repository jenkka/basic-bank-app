# Build stage
FROM golang:1.26-alpine AS builder
WORKDIR /app/
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN go build -v -o main main.go
RUN apk add curl
RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.19.1/migrate.linux-amd64.tar.gz | tar xz

# Run stage
FROM alpine:3.22
WORKDIR /app/
COPY --from=builder /app/main .
COPY app.env .
COPY start.sh .
COPY --from=builder /app/migrate .
COPY db/migration ./migration

EXPOSE 8080
CMD ["./main"]
ENTRYPOINT ["/app/start.sh"]
