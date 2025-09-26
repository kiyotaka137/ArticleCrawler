FROM golang:1.20-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git build-base
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /articlecrawler ./cmd/crawler

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=builder /articlecrawler /articlecrawler
COPY config.yaml /config.yaml
EXPOSE 50051 8080
ENTRYPOINT ["/articlecrawler", "-config", "/config.yaml"]
