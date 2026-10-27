# Шаг 1: Сборка
FROM golang:1.27 AS builder

WORKDIR /app

COPY go.work go.work.sum ./
COPY proto/go.mod proto/go.sum ./proto/
COPY patient-service/go.mod patient-service/go.sum ./patient-service/
COPY appointment-service/go.mod appointment-service/go.sum ./appointment-service/
COPY api-gateway/go.mod api-gateway/go.sum ./api-gateway/

RUN cd appointment-service && go mod download

COPY proto/ ./proto/
COPY appointment-service/ ./appointment-service/

RUN cd appointment-service && CGO_ENABLED=0 GOOS=linux go build -o /appointment-service ./cmd

# ---------------------------------------------------------
# Шаг 2: Финальный образ
# ---------------------------------------------------------
FROM alpine:latest

RUN apk add --no-cache tzdata

WORKDIR /root/

COPY --from=builder /appointment-service .

EXPOSE 50052

CMD ["./appointment-service"]