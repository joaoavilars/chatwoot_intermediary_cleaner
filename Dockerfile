# Stage 1: build
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o msg-cleaner .

# Stage 2: imagem final mínima
FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/msg-cleaner .

EXPOSE 8081
CMD ["./msg-cleaner"]
