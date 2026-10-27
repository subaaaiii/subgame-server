FROM golang:1.24-alpine AS base
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download


FROM base AS dev
RUN go install github.com/air-verse/air@v1.61.7
CMD ["air", "-c", ".air.toml"]


FROM base AS builder
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/api-server .

FROM alpine:latest AS prod
WORKDIR /app
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /app/api-server .

EXPOSE 8080
CMD ["./api-server"]