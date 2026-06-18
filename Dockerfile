# --- Stage 1: Build the Binary ---
FROM golang:1.22-alpine AS builder

WORKDIR /app

# 1. Force the public proxy configurations
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOPRIVATE=""

# 2. CRITICAL CHANGE: Copy the entire source tree into the container FIRST
COPY . .

# 3. Clean up the module dependencies and download them natively inside the environment
RUN go mod tidy
RUN go mod download

# 4. Compile the static binary file
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o main .

# --- Stage 2: Final Light Container ---
FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 8081
CMD ["./main"]