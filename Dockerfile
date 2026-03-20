# Stage 1: The Builder
FROM golang:1.26.1-alpine3.23 AS builder

# Set build environment
ENV CGO_ENABLED=0
ENV GOOS=linux

WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN go build -o /jamailify ./src

# Stage 2: The Final Image
FROM alpine:latest

# Install required packages
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Create directories for volumes
RUN mkdir -p /app/data /app/config

# Copy binary from builder
COPY --from=builder /jamailify /app/jamailify

# Run the application
CMD ["./jamailify"]
