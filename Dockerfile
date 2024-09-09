# Build stage
FROM golang:1.23-alpine AS builder

# Set the Current Working Directory inside the build container
WORKDIR /app

# Copy the go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source files
COPY . .

# Build the Go app
RUN go build -o main .

# Final stage
FROM alpine:latest

# Set the Current Working Directory inside the runtime container
WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /app/main .

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
