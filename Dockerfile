# Stage 1: Build the statically linked Go binary for linux/amd64 (x86)
FROM --platform=linux/amd64 golang:1.25-alpine AS builder

WORKDIR /app

# Copy module files
COPY go.mod ./
RUN go mod download

# Copy all source files
COPY . .

# Statically compile the binary for Linux amd64 (x86_64)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o worldtime .

# Stage 2: Final minimal scratch image for linux/amd64
FROM --platform=linux/amd64 scratch

# Copy statically linked binary
COPY --from=builder /app/worldtime /worldtime

# Expose port
EXPOSE 8080

# Run server
ENTRYPOINT ["/worldtime"]
