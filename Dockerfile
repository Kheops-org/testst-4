# Stage 1: Build
FROM golang:1-bookworm as build

# Set the working directory to /app
WORKDIR /app

# Copy the Go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY server.go .

# Build the executable
RUN go build -o helloserver server.go

# Stage 2: Production
FROM golang:1-bookworm

# Set the working directory to /app
WORKDIR /app

# Copy the executable from the build stage
COPY --from=build /app/helloserver .

# Expose the port
EXPOSE 8080

# Run the executable when the container starts
CMD ["./helloserver"]
