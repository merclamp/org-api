FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /app/bin/api \
    ./cmd/api

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/bin/api ./api
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./api"]