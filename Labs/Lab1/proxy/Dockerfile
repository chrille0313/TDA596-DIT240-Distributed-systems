# --- BUILD ---
FROM golang:1.24 AS builder

WORKDIR /app

COPY ./go.mod ./go.sum ./
RUN go mod download

COPY ./http ./http
COPY ./proxy ./proxy

WORKDIR /app/proxy

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/proxy .

# --- RUN ---
FROM gcr.io/distroless/base-debian12

WORKDIR /app
COPY --from=builder /app/proxy .

EXPOSE 80

CMD ["./proxy"]

