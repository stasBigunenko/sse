# Stage 1: Build the Go binary
FROM golang:1.22 as builder

WORKDIR /

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o app ./

# Stage 2: Create the lightweight image with the binary
FROM alpine:3.20

# Install certificates needed for HTTPS requests, etc.
RUN apk update && apk add --no-cache ca-certificates

WORKDIR /

# Copy the binary from the builder stage
COPY --from=builder /app ./

# Ensure the binary has execute permissions
RUN chmod +x app

# Correct ENTRYPOINT to reference the built binary
ENTRYPOINT ["./app"]
