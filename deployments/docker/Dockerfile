FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o aegis ./cmd/aegis

FROM alpine:3.19

RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/aegis .
COPY configs/ ./configs/

EXPOSE 8080 9090

ENTRYPOINT ["./aegis"]
