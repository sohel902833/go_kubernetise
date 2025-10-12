# Builder stage
FROM golang:1.25 AS builder

WORKDIR /app

# Cache modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Install Air for dev mode
RUN go install github.com/cosmtrek/air@latest

# Build app (for production, use "go build -o ./bin/app ./src/main.go")
RUN go build -o ./tmp/app ./src/main.go

# Final stage
FROM golang:1.25 AS runtime

WORKDIR /app

COPY --from=builder /app /app

# Expose port
EXPOSE 3000

# Run app using air (dev mode)
CMD ["air"]
