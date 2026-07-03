FROM golang:1.26-trixie AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/go-api .

FROM debian:trixie-20260623-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/go-api /app/
EXPOSE 8080
CMD ["/app/go-api"]
