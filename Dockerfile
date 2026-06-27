# Stage 1: Build the statically linked Go binary on the host platform
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

WORKDIR /app

# Copy module files
COPY go.mod ./
RUN go mod download

# Copy all source files
COPY . .

# Retrieve target OS and architecture from Docker buildx
ARG TARGETOS
ARG TARGETARCH

# Statically cross-compile the binary for the target OS and architecture
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-w -s" -o worldtime .

# Stage 2: Final minimal scratch image matching target platform
FROM scratch

# Copy statically linked binary
COPY --from=builder /app/worldtime /worldtime

# Expose port
EXPOSE 8080

# Run server
ENTRYPOINT ["/worldtime"]
