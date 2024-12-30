# Use the official Go image for building
FROM golang:1.23.4 as builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum for dependency installation
COPY go.mod go.sum ./

# Download Go module dependencies
RUN go mod download

# Copy the entire project into the container
COPY . .

# Build the Go application with static linking
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

# Use a minimal alpine image for the final container
FROM alpine:latest

# Install certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Set the working directory
WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/app .

# Expose the port your app listens on
EXPOSE 8080

# Run the application
CMD ["./app"]