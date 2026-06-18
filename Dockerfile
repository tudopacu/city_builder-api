# --- Stage 1: Build the Binary ---
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy BOTH dependency configuration maps
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o main .

# --- Stage 2: Final Light Container ---
FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .

# EXPOSE PORT 8081
EXPOSE 8081

CMD ["./main"]