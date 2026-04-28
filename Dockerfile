# Build stage
FROM golang:1.26-alpine AS builder
WORKDIR /app/
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN go build -v -o main main.go

# Run stage
FROM alpine:3.22
WORKDIR /app/
COPY --from=builder /app/main .
COPY app.env .

EXPOSE 8080
CMD ["./main"]
