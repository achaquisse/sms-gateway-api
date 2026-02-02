FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY src/go.mod src/go.sum ./
RUN go mod download

COPY src/ .

RUN CGO_ENABLED=0 GOOS=linux go build -o /sms-gateway-api .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /sms-gateway-api .
COPY db-schema.sql /app/db-schema.sql
COPY openapi.yml /app/openapi.yml

EXPOSE 8080

CMD ["./sms-gateway-api"]
