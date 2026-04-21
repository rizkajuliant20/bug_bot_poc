# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bug-bot ./cmd/bot

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/bug-bot .

# Copy .env file if exists (optional)
COPY .env* ./

# Create logs directory
RUN mkdir -p logs

# Expose port (optional, for health checks)
EXPOSE 3000

# Run the bot
CMD ["./bug-bot"]
