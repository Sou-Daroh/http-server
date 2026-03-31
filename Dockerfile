# Multi-stage Docker build for minimal image size
FROM golang:alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build statically linked binary specifically for Alpine Linux
RUN CGO_ENABLED=0 GOOS=linux go build -o overwatch main.go

# Minimal runner image (Under 20MB)
FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/overwatch .
COPY --from=builder /app/config.json .
COPY --from=builder /app/static ./static

EXPOSE 8080
CMD ["./overwatch"]
