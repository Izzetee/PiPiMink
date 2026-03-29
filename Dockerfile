# Build Stage
FROM golang:1.25-alpine AS builder

# Install dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the code
COPY . .

# Run tests (the build will fail if tests fail)
RUN go test -v ./...

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/pipimink ./main.go

# Final Stage
FROM alpine:latest

# Install ca-certificates for secure connections
RUN apk --no-cache add ca-certificates

# Set working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/pipimink .

# Copy any required configuration files
COPY --from=builder /app/.env* ./
COPY --from=builder /app/providers.json* ./

# Expose the port the app runs on
EXPOSE 8080

# Command to run
CMD ["./pipimink"]
