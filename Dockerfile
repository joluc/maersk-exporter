# Build stage
FROM --platform=$BUILDPLATFORM golang:1.26 AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Copy go.mod and download dependencies
COPY go.mod ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o maersk-exporter ./cmd/maersk-exporter

# Run stage
FROM gcr.io/distroless/static-debian12:latest

# Set working directory for the runtime container
WORKDIR /

# Copy the compiled binary from the builder stage
COPY --from=builder /app/maersk-exporter /maersk-exporter

# Expose the default metrics port
EXPOSE 9878

# Set the command to run the exporter
ENTRYPOINT ["/maersk-exporter"]
