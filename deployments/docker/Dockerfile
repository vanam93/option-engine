FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /option-engine ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /option-engine .
COPY configs/config.yaml ./configs/config.yaml

EXPOSE 8080

ENTRYPOINT ["./option-engine"]
CMD ["-config", "configs/config.yaml"]
