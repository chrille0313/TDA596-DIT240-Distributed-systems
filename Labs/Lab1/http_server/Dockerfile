# --- BUILD ---
FROM golang:1.24 AS builder

WORKDIR /app

COPY ./go.mod ./go.sum ./
RUN go mod download

# Copy files
COPY ./http ./http
COPY ./http_server/files.go ./http_server/files.go
COPY ./http_server/http_server.go ./http_server/http_server.go

WORKDIR /app/http_server

# Build only the http_server binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/http_server .

# --- RUN ---
FROM gcr.io/distroless/base-debian12

WORKDIR /app
COPY --from=builder /app/http_server .

EXPOSE 80
EXPOSE 8080

CMD ["./http_server"]