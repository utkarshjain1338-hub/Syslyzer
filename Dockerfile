FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o sysinfo

FROM alpine:latest

WORKDIR /app

RUN addgroup -S app && adduser -S -G app app

COPY --from=builder --chown=app:app /app/sysinfo /app/sysinfo

USER app

EXPOSE 8080

CMD ["/app/sysinfo"]