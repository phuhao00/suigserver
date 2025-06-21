# Stage 1: Build the Go application
FROM golang:1.21-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files from the project root
COPY go.mod go.sum ./

# Download all dependencies based on the root go.mod and go.sum.
# These will be cached if go.mod and go.sum haven't changed.
RUN go mod download

# Copy the entire project source code into the container
# This includes the 'server' directory and all other files.
COPY . .

# Build the Go server application.
# The main package is located at ./server/cmd/game/main.go relative to the project root.
# The output binary is named 'server_app' and placed in the root of the builder stage.
# CGO_ENABLED=0 ensures static linking.
# GOOS=linux is for cross-compilation if building on a non-Linux host (though less critical if builder is linux).
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /server_app ./server/cmd/game/main.go

# Stage 2: Create the production image
# Use a minimal base image like alpine
FROM alpine:latest

# Set the Current Working Directory inside the container for the production stage
WORKDIR /root/

# Copy the pre-built binary file from the builder stage
COPY --from=builder /server_app .

# Copy an example configuration file.
# In a real deployment, this config.json should be mounted as a volume or managed by a config system.
# This ensures the application has a default config if none is provided.
# The application itself (configs.CreateExampleConfigFile) can also create this.
# COPY server/configs/config.json.example ./config.json
# For now, we'll assume the application creates config.json on first run if not found,
# or that it's mounted. If not, the app might fail if it strictly requires config.json to exist.

# Expose the default TCP port the server listens on
EXPOSE 8080
# EXPOSE 8081 # If an HTTP admin/metrics port is also used

# Command to run the executable.
# The application will look for "config.json" in its working directory (/root/).
CMD ["./server_app"]

# --- Notes for users ---
# To build the Docker image:
#   docker build -f docker/Dockerfile -t sui-mmo-server .
#
# To run the Docker container:
#   1. Create a 'config.json' in your project root or desired config directory.
#      You can adapt it from 'server/configs/config.json.example' (if one were created)
#      or let the application create a default one if you've implemented that.
#   2. Run the container, mounting your config file:
#      docker run -p 8080:8080 -v $(pwd)/config.json:/root/config.json sui-mmo-server
#      (Adjust path to your config.json if it's not in $(pwd))
#
# The application is expected to load 'config.json' from its current working directory.
# The `server/configs/config.go` has a `CreateExampleConfigFile("config.json")` function.
# If `main.go` calls this, then the `COPY server/configs/config.json.example ./config.json`
# line above is less critical, as the app can bootstrap its own default config.
# For a production setup, explicitly providing the config via volume mount is recommended.
