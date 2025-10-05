FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /subscription-service ./cmd/app

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=builder /subscription-service /subscription-service
EXPOSE 8080
ENTRYPOINT ["/subscription-service"]
