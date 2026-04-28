FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . . 
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/app ./cmd/

FROM alpine:3.21
WORKDIR /app
COPY --from=builder /app/bin/app ./app
COPY --from=builder /app/config ./config
EXPOSE 50051 9090
ENTRYPOINT ["./app"]