FROM golang:1.24 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -buildvcs=false -o mtpo-backend ./cmd

FROM debian:12
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /app/mtpo-backend .
EXPOSE 9090 16167
CMD ["./mtpo-backend"]