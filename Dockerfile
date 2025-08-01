# Builder stage
FROM golang:1.21-alpine AS builder
WORKDIR /app


COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o tape .

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates docker-cli

COPY --from=builder /app/tape /tape
COPY --from=builder /app/web /app/web  


EXPOSE 42113

CMD ["/tape"]
