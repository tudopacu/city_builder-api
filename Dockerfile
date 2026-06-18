# --- Stage 1: Build the Binary ---
# CRITICAL FIX: Match or exceed your go.mod version requirement (>= 1.25.0)
FROM golang:1.25-alpine AS builder

WORKDIR /app

ENV GOPROXY=https://proxy.golang.org,direct
ENV GOPRIVATE=""

# Copy the source tree
COPY . .

# Run tidy and download natively using the Go 1.25 compiler environment
RUN go mod tidy
RUN go mod download

# Compile the static binary file
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o main .

# --- Stage 2: Final Light Container ---
FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 8081
CMD ["./main"]