FROM golang:1.24.5-alpine AS builder

# Install dependencies for building
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/tvs/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/main .

# Copy configuration and migrations
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/migrations ./migrations

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./main"]
