# Шаг 1: Сборка
FROM golang:1.27 AS builder

WORKDIR /app

COPY go.work go.work.sum ./
COPY proto/go.mod proto/go.sum ./proto/
COPY patient-service/go.mod patient-service/go.sum ./patient-service/
COPY appointment-service/go.mod appointment-service/go.sum ./appointment-service/
COPY api-gateway/go.mod api-gateway/go.sum ./api-gateway/

RUN cd api-gateway && go mod download

COPY proto/ ./proto/
COPY api-gateway/ ./api-gateway/

RUN cd api-gateway && CGO_ENABLED=0 GOOS=linux go build -o /api-gateway ./cmd

# Шаг 2: Финальный образ
FROM alpine:latest

WORKDIR /root/

COPY --from=builder /api-gateway .

EXPOSE 8080

CMD ["./api-gateway"]