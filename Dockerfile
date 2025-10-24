FROM golang:latest AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0  go build -o /articlecrawler ./cmd

FROM alpine:3.18
COPY --from=builder /articlecrawler /articlecrawler
COPY config.yaml /config.yaml
EXPOSE 50051 8080
ENTRYPOINT ["/articlecrawler", "-config", "/config.yaml"]
